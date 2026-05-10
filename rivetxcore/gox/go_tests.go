package gox

import (
	"fmt"
	"rivetxgo/rivetxcore/syncx"
	"time"
)

var waitAll = syncx.NewTaskGroup()
var waitBatch = syncx.NewTaskGroup()
var waitUniq = syncx.NewTaskGroup()
var waitList = syncx.NewTaskGroup()
var waitTimer = syncx.NewTaskGroup()

type Data struct {
	N int
	T time.Time
}

var IsOpenBatch = true
var IsOpenUniq = true
var IsOpenList = true
var IsOpenTimer = true

var BatchSpawnMax = 1000
var BatchSpawnNum = 0

var UniqSpawnMax = 1000
var UniqSpawnNum = 0

var ListSpawnMax = 1000
var ListSpawnNum = 0

var TimerSpawnMax = 100
var TimerSpawnNum = 0
var TimerSpawnSleep = int64(10)

var BatchSpawnLog = make([]string, 0, 10000000)
var UniqSpawnLog = make([]string, 0, 10000000)
var ListSpawnLog = make([]string, 0, 10000000)
var TimerSpawnLog = make([]string, 0, 10000000)

func TestBatchSpawn() {
	name := "test1"
	BatchSpawn(name, 100, time.Second, func(datas []interface{}) error {
		if len(datas) <= 0 {
			return nil
		}
		defer waitBatch.Add(0 - len(datas))
		endTime := time.Now()
		lastData := datas[len(datas)-1].(*Data)
		for _, dataI := range datas {
			BatchSpawnNum += 1
			data := dataI.(*Data)
			log := fmt.Sprintf("BatchSpawn data:%+v, len:%v, BatchSpawnNum:%+v, Nanoseconds:%v\n", data, len(datas), BatchSpawnNum, (endTime.UnixNano()-lastData.T.UnixNano())/int64(len(datas)))
			BatchSpawnLog = append(BatchSpawnLog, log)
		}
		return nil
	})
	for i := 0; i < BatchSpawnMax; i++ {
		waitBatch.Add(1)
		BatchAdd(name, &Data{N: i, T: time.Now()})
	}
	BatchFlush(name)
	BatchClose(name)
}

func TestUniqSpawn() {
	name := "test1"
	for i := 0; i < UniqSpawnMax; i++ {
		data := &Data{N: i, T: time.Now()}
		waitUniq.Add(1)
		UniqSpawn(name, true, func() error {
			defer waitUniq.Done()
			UniqSpawnNum++
			log := fmt.Sprintf("UniqSpawn data:%+v, UniqSpawnNum:%v, Nanoseconds:%v\n", data, UniqSpawnNum, time.Since(data.T).Nanoseconds())
			UniqSpawnLog = append(UniqSpawnLog, log)
			return nil
		})
	}
}

func TestListSpawn() {
	name := "test1"
	ListSpawn(name, 1000, func(dataI interface{}) error {
		if dataI == nil {
			return nil
		}
		defer waitList.Done()
		endTime := time.Now()
		ListSpawnNum += 1
		data := dataI.(*Data)
		log := fmt.Sprintf("ListSpawn data:%+v, ListSpawnNum:%+v, Nanoseconds:%v\n", data, ListSpawnNum, endTime.UnixNano()-data.T.UnixNano())
		ListSpawnLog = append(ListSpawnLog, log)

		return nil
	})
	for i := 0; i < ListSpawnMax; i++ {
		waitList.Add(1)
		ListAdd(name, &Data{N: i, T: time.Now()})
	}
	ListClose(name)
}

func TestTimerSpawn() {
	name := "test1"
	waitTimer.Add(TimerSpawnMax)
	TimerSpawn(name, true, time.Duration(TimerSpawnSleep)*time.Millisecond, func() (bool, error) {
		defer waitTimer.Done()
		data := &Data{N: TimerSpawnNum}
		TimerSpawnNum++
		log := fmt.Sprintf("TimerSpawn data:%+v, TimerSpawnNum:%v\n", data, TimerSpawnNum)
		TimerSpawnLog = append(TimerSpawnLog, log)
		if TimerSpawnNum == TimerSpawnMax {
			return true, nil
		}
		return false, nil
	})
}

func GoTests() error {
	startTime := time.Now()
	BatchSpawnDiffTime := int64(0)
	UniqSpawnDiffTime := int64(0)
	ListSpawnDiffTime := int64(0)
	TimerSpawnDiffTime := int64(0)
	if IsOpenBatch {
		waitAll.Add(1)
		Spawn(func(u uint64) error {
			defer waitAll.Done()
			startTime := time.Now()
			TestBatchSpawn()
			waitBatch.Wait()
			endTime := time.Now()
			BatchSpawnDiffTime = endTime.UnixMilli() - startTime.UnixMilli()
			return nil
		})
	}
	if IsOpenUniq {
		waitAll.Add(1)
		Spawn(func(u uint64) error {
			defer waitAll.Done()
			startTime := time.Now()
			TestUniqSpawn()
			waitUniq.Wait()
			endTime := time.Now()
			UniqSpawnDiffTime = endTime.UnixMilli() - startTime.UnixMilli()
			return nil
		})
	}

	if IsOpenList {
		waitAll.Add(1)
		Spawn(func(u uint64) error {
			defer waitAll.Done()
			startTime := time.Now()
			TestListSpawn()
			waitList.Wait()
			endTime := time.Now()
			ListSpawnDiffTime = endTime.UnixMilli() - startTime.UnixMilli()
			return nil
		})
	}

	if IsOpenTimer {
		waitAll.Add(1)
		Spawn(func(u uint64) error {
			defer waitAll.Done()
			startTime := time.Now()
			TestTimerSpawn()
			waitTimer.Wait()
			endTime := time.Now()
			TimerSpawnDiffTime = endTime.UnixMilli() - startTime.UnixMilli()
			return nil
		})
	}
	waitAll.Wait()
	endTime := time.Now()
	diffTime := endTime.UnixMilli() - startTime.UnixMilli()

	for _, log := range BatchSpawnLog {
		fmt.Printf(log)
	}

	for _, log := range UniqSpawnLog {
		fmt.Printf(log)
	}

	for _, log := range ListSpawnLog {
		fmt.Printf(log)
	}

	for _, log := range TimerSpawnLog {
		fmt.Printf(log)
	}

	fmt.Printf("BatchSpawnDiffTime:%v\n", BatchSpawnDiffTime)
	fmt.Printf("UniqSpawnDiffTime:%v\n", UniqSpawnDiffTime)
	fmt.Printf("ListSpawnDiffTime:%v\n", ListSpawnDiffTime)
	fmt.Printf("TimerSpawnDiffTime:%v, avg:%v\n", TimerSpawnDiffTime, TimerSpawnDiffTime/int64(TimerSpawnMax))
	fmt.Printf("diffTime:%v\n", diffTime)

	fmt.Printf("BatchSpawnNum:%v, BatchSpawnMax:%v\n", BatchSpawnNum, BatchSpawnMax)
	if BatchSpawnNum != BatchSpawnMax {
		fmt.Printf("BatchSpawnNum != BatchSpawnMax\n")
	}

	fmt.Printf("UniqSpawnNum:%v, UniqSpawnMax:%v\n", UniqSpawnNum, UniqSpawnMax)
	if UniqSpawnNum != UniqSpawnMax {
		fmt.Printf("UniqSpawnNum != UniqSpawnMax\n")
	}

	fmt.Printf("ListSpawnNum:%v, ListSpawnMax:%v\n", ListSpawnNum, ListSpawnMax)
	if ListSpawnNum != ListSpawnMax {
		fmt.Printf("ListSpawnNum != ListSpawnMax\n")
	}

	fmt.Printf("TimerSpawnNum:%v, TimerSpawnMax:%v\n", TimerSpawnNum, TimerSpawnMax)
	if TimerSpawnNum != TimerSpawnMax {
		fmt.Printf("TimerSpawnNum != TimerSpawnMax\n")
	}

	return nil
}
