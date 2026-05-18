package gox

import (
	"context"
	"github.com/yefy/log4go/ee"
	"github.com/yefy/rivetxgo/rivetxcore/config"
	"github.com/yefy/rivetxgo/rivetxcore/session"
	"strings"
	"sync"
	"time"

	"github.com/yefy/log4go/log4"
)

var uniqMap sync.Map
var ctx context.Context

var chanPools [4097]sync.Pool

type Chan struct {
	Ch   chan interface{}
	pool *sync.Pool
}

func (ch *Chan) Put() {
	if ch.pool != nil {
		ch.pool.Put(ch)
	}
}

func NewChan(size int, pool *sync.Pool) *Chan {
	if size < 0 {
		size = 0
	}
	return &Chan{
		Ch:   make(chan interface{}, size),
		pool: pool,
	}
}

func init() {
	for i := range chanPools {
		i := i
		func(i int) {
			chanPools[i].New = func() interface{} {
				return NewChan(i, &chanPools[i])
			}
		}(i)
	}
}

func GetChan(size int) *Chan {
	if size < 0 {
		size = 0
	}
	if size >= len(chanPools) {
		return NewChan(size, nil)
	}
	ch := chanPools[size].Get().(*Chan)
	for {
		select {
		case <-ch.Ch:
		default:
			return ch
		}
	}
}

func Spawn(spawnFunc func(uint64) error) {
	go func() {
		StatNumStartAdd("Spawn")
		defer func() {
			StatNumEndAdd("Spawn")
		}()
		spawnId := session.SessionId()
		defer func() {
			if r := recover(); r != nil {
				log4.Recover(r)
			}
		}()
		err := spawnFunc(spawnId)
		if err != nil {
			IgnoreErr := "i/o timeout"
			NeedFindStr := err.Error()
			NeedFindStrLen := len(NeedFindStr)
			if NeedFindStrLen > 30 {
				NeedFindStr = NeedFindStr[NeedFindStrLen-30 : NeedFindStrLen]
			}

			isIgnorePrint := false
			if e, ok := err.(*ee.Error); ok {
				isIgnorePrint = e.IgnorePrint()
			}
			if !isIgnorePrint {
				if len(NeedFindStr) >= len(IgnoreErr) && strings.Contains(NeedFindStr, IgnoreErr) {
					if config.IsOpenStackInfoToErrorLog() {
						log4.Warn("[spawnId:%d] err:%+#v", spawnId, err)
					} else {
						log4.Warn("[spawnId:%d] err:%v", spawnId, err)
					}
				} else {
					if config.IsOpenStackInfoToErrorLog() {
						log4.Error("[spawnId:%d] err:%+#v", spawnId, err)
					} else {
						log4.Error("[spawnId:%d] err:%v", spawnId, err)
					}
				}
			}
		}
	}()
}

func SetUniqSpawnCtx(c context.Context) {
	ctx = c
}

type UniqData struct {
	Func     func() error
	WaitChan *Chan
}

func NewUniqSpawns() *UniqSpawns {
	us := &UniqSpawns{
		DataChan: make(chan UniqData, 1000),
	}
	return us
}

type UniqSpawns struct {
	DataChan chan UniqData
}

func (us *UniqSpawns) Add(isWait bool, Func func() error) {
	uniqChan := us.DataChan
	UniqData := UniqData{Func: Func}
	if isWait {
		waitChan := GetChan(1)
		UniqData.WaitChan = waitChan
	}
	uniqChan <- UniqData
	if UniqData.WaitChan != nil {
		<-UniqData.WaitChan.Ch
		UniqData.WaitChan.Put()
	}
}

func (us *UniqSpawns) Run() {
	Spawn(func(spawnId uint64) error {
		uniqChan := us.DataChan
		for {
			isOk := func() bool {
				defer func() {
					if r := recover(); r != nil {
						log4.Recover(r)
					}
				}()
				if ctx != nil {
					done := ctx.Done()
					for {
						select {
						case uniqData, ok := <-uniqChan:
							if !ok {
								return true
							}
							err := uniqData.Func()
							if err != nil {
								isIgnorePrint := false
								if e, ok := err.(*ee.Error); ok {
									isIgnorePrint = e.IgnorePrint()
								}
								if !isIgnorePrint {
									if config.IsOpenStackInfoToErrorLog() {
										log4.Error("err:%+#v", err)
									} else {
										log4.Error("err:%v", err)
									}
								}
							}
							if uniqData.WaitChan != nil {
								uniqData.WaitChan.Ch <- true
							}
						case <-done:
							return true
						}
					}
				} else {
					for {
						uniqData, ok := <-uniqChan
						if !ok {
							return true
						}
						err := uniqData.Func()
						if err != nil {
							isIgnorePrint := false
							if e, ok := err.(*ee.Error); ok {
								isIgnorePrint = e.IgnorePrint()
							}
							if !isIgnorePrint {
								if config.IsOpenStackInfoToErrorLog() {
									log4.Error("err:%+#v", err)
								} else {
									log4.Error("err:%v", err)
								}
							}
						}
						if uniqData.WaitChan != nil {
							uniqData.WaitChan.Ch <- true
						}
					}
				}
			}()
			if isOk {
				return nil
			}
		}
	})
}

func UniqSpawn(name string, isWait bool, Func func() error) {
	UniqSpawnsI, ok := uniqMap.Load(name)
	if !ok {
		isLoaded := false
		UniqSpawnsI, isLoaded = uniqMap.LoadOrStore(name, NewUniqSpawns())
		if !isLoaded {
			UniqSpawns := UniqSpawnsI.(*UniqSpawns)
			UniqSpawns.Run()
		}
	}

	UniqSpawns := UniqSpawnsI.(*UniqSpawns)
	UniqSpawns.Add(isWait, Func)
}

func NewBatchSpawns(BatchSize int) *BatchSpawns {
	bs := &BatchSpawns{
		NewBatchData(BatchSize),
	}
	return bs
}

type BatchSpawns struct {
	*BatchData
}

func (bs *BatchSpawns) Add(data interface{}) {
	BatchData := bs.BatchData
	size := func() int {
		defer BatchData.Mutex.Unlock()
		BatchData.Mutex.Lock()
		BatchData.Datas = append(BatchData.Datas, data)
		return len(BatchData.Datas)
	}()

	if size == BatchData.BatchSize {
		select {
		case BatchData.Chan <- BatchDataFlush:
		default:
		}
	}
}

func (bs *BatchSpawns) Flush() {
	BatchData := bs.BatchData
	select {
	case BatchData.Chan <- BatchDataFlush:
	case <-time.After(5 * time.Second):
	}
}

func (bs *BatchSpawns) Close() {
	BatchData := bs.BatchData
	select {
	case BatchData.Chan <- BatchDataClose:
	case <-time.After(5 * time.Second):
	}
}

func (bs *BatchSpawns) Run(waitTime time.Duration, Func func([]interface{}) error) {
	BatchData := bs.BatchData
	Spawn(func(spawnId uint64) error {
		ticker := time.NewTicker(waitTime)
		defer ticker.Stop()
		for {
			flag := BatchDataFlush
			select {
			case flag_, ok := <-BatchData.Chan:
				if !ok {
					return nil
				}
				flag = int(flag_)
			doneLoop:
				for {
					select {
					case flag_ = <-BatchData.Chan: //ok  notify chan
						flag = int(flag_)
					default:
						break doneLoop
					}
				}
			case <-ticker.C:
				//do
			}
			datas := func() []interface{} {
				defer BatchData.Mutex.Unlock()
				BatchData.Mutex.Lock()
				if len(BatchData.Datas) <= 0 {
					return nil
				}
				datas := BatchData.Datas
				BatchData.Datas = make([]interface{}, 0, BatchData.BatchSize*2)
				return datas
			}()
			err := Func(datas)
			if err != nil {
				isIgnorePrint := false
				if e, ok := err.(*ee.Error); ok {
					isIgnorePrint = e.IgnorePrint()
				}
				if !isIgnorePrint {
					if config.IsOpenStackInfoToErrorLog() {
						log4.Error("err:%+#v", err)
					} else {
						log4.Error("err:%v", err)
					}
				}
			}
			if flag == BatchDataClose {
				return nil
			}
		}
	})
}

func NewBatchData(BatchSize int) *BatchData {
	if BatchSize <= 0 {
		BatchSize = 10
	}
	return &BatchData{
		Datas:     make([]interface{}, 0, BatchSize*2),
		Mutex:     sync.Mutex{},
		Chan:      make(chan uint8, 10),
		BatchSize: BatchSize,
	}
}

const BatchDataFlush = 0
const BatchDataClose = 1

type BatchData struct {
	Datas        []interface{}
	Mutex        sync.Mutex
	Chan         chan uint8
	BatchSize    int
	WaitTimeMs   int64
	CallbackFunc func([]interface{}) error
}

var BatchMap sync.Map

func BatchAdd(name string, data interface{}) {
	BatchSpawnsI, ok := BatchMap.Load(name)
	if !ok {
		log4.Error("BatchAdd not find name:%v", name)
		return
	}
	BatchSpawns := BatchSpawnsI.(*BatchSpawns)
	BatchSpawns.Add(data)
}

func BatchFlush(name string) {
	BatchSpawnsI, ok := BatchMap.Load(name)
	if !ok {
		log4.Error("BatchClose not find name:%v", name)
		return
	}
	BatchSpawns := BatchSpawnsI.(*BatchSpawns)
	BatchSpawns.Flush()
}

func BatchClose(name string) {
	BatchSpawnsI, ok := BatchMap.Load(name)
	if !ok {
		log4.Error("BatchClose not find name:%v", name)
		return
	}
	BatchSpawns := BatchSpawnsI.(*BatchSpawns)
	BatchSpawns.Close()
}

func BatchSpawn(name string, BatchSize int, waitTime time.Duration, Func func([]interface{}) error) {
	_, ok := BatchMap.Load(name)
	if !ok {
		BatchSpawnsI, isLoaded := BatchMap.LoadOrStore(name, NewBatchSpawns(BatchSize))
		if !isLoaded {
			BatchSpawns := BatchSpawnsI.(*BatchSpawns)
			BatchSpawns.Run(waitTime, Func)
		}
	}
}

func NewTimerSpawns() *TimerSpawns {
	bs := &TimerSpawns{
		NewTimerData(),
	}
	return bs
}

type TimerSpawns struct {
	*TimerData
}

func (bs *TimerSpawns) Run(isFirstCall bool, waitTime time.Duration, Func func() (bool, error)) {
	//TimerData := bs.TimerData
	Spawn(func(spawnId uint64) error {
		isFirst := true
		ticker := time.NewTicker(waitTime)
		defer ticker.Stop()
		for {
			isWait := func() bool {
				if isFirst {
					isFirst = false
					if isFirstCall {
						return false
					}
				}
				return true
			}()
			if isWait {
				select {
				case <-ticker.C:
					//do
				}
			}

			isQuit, err := Func()
			if err != nil {
				isIgnorePrint := false
				if e, ok := err.(*ee.Error); ok {
					isIgnorePrint = e.IgnorePrint()
				}
				if !isIgnorePrint {
					if config.IsOpenStackInfoToErrorLog() {
						log4.Error("err:%+#v", err)
					} else {
						log4.Error("err:%v", err)
					}
				}
			}
			if isQuit {
				return nil
			}
		}
	})
}

func NewTimerData() *TimerData {
	return &TimerData{}
}

type TimerData struct {
}

var TimerMap sync.Map

func TimerSpawn(name string, isFirstCall bool, waitTime time.Duration, Func func() (bool, error)) {
	_, ok := TimerMap.Load(name)
	if !ok {
		TimerSpawnsI, isLoaded := TimerMap.LoadOrStore(name, NewTimerSpawns())
		if !isLoaded {
			TimerSpawns := TimerSpawnsI.(*TimerSpawns)
			TimerSpawns.Run(isFirstCall, waitTime, Func)
		}
	}
}

func NewListSpawns(size int) *ListSpawns {
	bs := &ListSpawns{
		NewListData(size),
	}
	return bs
}

type ListSpawns struct {
	*ListData
}

func (bs *ListSpawns) Add(data interface{}) {
	if data == nil {
		return
	}
	ListData := bs.ListData
	ListData.Chan <- data
}

func (bs *ListSpawns) Close() {
	ListData := bs.ListData
	ListData.Chan <- nil
}

func (bs *ListSpawns) Run(Func func(interface{}) error) {
	ListData := bs.ListData
	Spawn(func(spawnId uint64) error {
		for {
			data, ok := <-ListData.Chan
			if !ok {
				return nil
			}
			err := Func(data)
			if err != nil {
				isIgnorePrint := false
				if e, ok := err.(*ee.Error); ok {
					isIgnorePrint = e.IgnorePrint()
				}
				if !isIgnorePrint {
					if config.IsOpenStackInfoToErrorLog() {
						log4.Error("err:%+#v", err)
					} else {
						log4.Error("err:%v", err)
					}
				}
			}
			if data == nil {
				return nil
			}
		}
	})
}

func NewListData(size int) *ListData {
	return &ListData{
		Chan: make(chan interface{}, size),
	}
}

type ListData struct {
	Chan         chan interface{}
	CallbackFunc func(interface{}) error
}

var ListMap sync.Map

func ListAdd(name string, data interface{}) {
	ListSpawnsI, ok := ListMap.Load(name)
	if !ok {
		log4.Error("ListAdd not find name:%v", name)
		return
	}
	ListSpawns := ListSpawnsI.(*ListSpawns)
	ListSpawns.Add(data)
}

func ListClose(name string) {
	ListSpawnsI, ok := ListMap.Load(name)
	if !ok {
		log4.Error("ListClose not find name:%v", name)
		return
	}
	ListSpawns := ListSpawnsI.(*ListSpawns)
	ListSpawns.Close()
}

func ListSpawn(name string, size int, Func func(interface{}) error) {
	_, ok := ListMap.Load(name)
	if !ok {
		ListSpawnsI, isLoaded := ListMap.LoadOrStore(name, NewListSpawns(size))
		if !isLoaded {
			ListSpawns := ListSpawnsI.(*ListSpawns)
			ListSpawns.Run(Func)
		}
	}
}
