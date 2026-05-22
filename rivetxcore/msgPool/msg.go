package msgPool

import (
	"fmt"
	"github.com/yefy/log4go/log4"
	"github.com/yefy/rivetxgo/rivetxcore/gox"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

var isOpenMsgPool bool = true

func SetMsgPoolFlag(isOpen bool) {
	isOpenMsgPool = isOpen
}

var msgPoolsBlock int = 64
var msgPools [16384]sync.Pool

func init() {
	for i := range msgPools {
		i := i
		func(i int) {
			msgPools[i].New = func() interface{} {
				msg := NewMsg((i+1)*msgPoolsBlock, &msgPools[i])
				return msg
			}
		}(i)
	}
}

func GetMsg(size int) *Msg {
	// 0 1024-1   0
	// 1024 1024*2-1 1
	index := (size + msgPoolsBlock - 1) / msgPoolsBlock
	if !isOpenMsgPool || index >= len(msgPools) {
		msg := NewMsg((index+1)*msgPoolsBlock, nil)
		msg.Init(size)
		return msg
	} else {
		for {
			msg := msgPools[index].Get().(*Msg)
			if msg.RefCount.Load() > 0 {
				gox.StatNumEndAdd("etcp_msg_err_Get")
				log4.Error("log_tcp_msg GetMsg RefCount > 0, Stack:%s", string(debug.Stack()))
				panic("log_tcp_msg: Msg reference counter went negative (double Put? or use-after-free?)")
				continue
			}
			msg.Init(size)
			return msg
		}
	}
}

type Msg struct {
	RefCount atomic.Int32
	Size     int
	Cap      int
	Datas    []byte
	Pool     *sync.Pool
}

func NewMsg(cap int, Pool *sync.Pool) *Msg {
	msg := &Msg{
		Datas:    make([]byte, cap),
		Size:     0,
		Cap:      cap,
		Pool:     Pool,
		RefCount: atomic.Int32{},
	}
	//msg.RefCount.Add(1)
	return msg
}

func NewMsgFromBytes(Data []byte) *Msg {
	msg := &Msg{
		Datas:    Data,
		Size:     len(Data),
		Cap:      len(Data),
		Pool:     nil,
		RefCount: atomic.Int32{},
	}
	msg.RefCount.Add(1)
	return msg
}

func (msg *Msg) Init(size int) {
	if size > msg.Cap {
		gox.StatNumEndAdd("etcp_msg_err_Init")
		log4.Error("log_tcp_msg size:%v > msg.Cap:%v, Stack:%s", size, msg.Cap, string(debug.Stack()))
		panic(fmt.Sprintf("log_tcp_msg size:%v > msg.Cap:%v", size, msg.Cap))
	}
	msg.Size = size
	msg.RefCount.Store(1)
	gox.StatNumStartAdd("etcp_msg")
}

func (msg *Msg) GetData() []byte {
	if msg.RefCount.Load() <= 0 {
		gox.StatNumEndAdd("etcp_msg_err_GetData")
		log4.Error("log_tcp_msg GetData RefCount <= 0, msg.Cap:%v, Stack:%s", msg.Cap, string(debug.Stack()))
		panic("log_tcp_msg: Msg reference counter went negative (double Put? or use-after-free?)")
	}
	return msg.Datas[0:msg.Size]
}

func (msg *Msg) Clone() *Msg {
	if msg.RefCount.Load() <= 0 {
		gox.StatNumEndAdd("etcp_msg_err_Clone")
		log4.Error("log_tcp_msg Clone RefCount <= 0, msg.Cap:%v, Stack:%s", msg.Cap, string(debug.Stack()))
		panic("log_tcp_msg: Msg reference counter went negative (double Put? or use-after-free?)")
	}
	msg.RefCount.Add(1)
	return msg
}

func (msg *Msg) Put() {
	RefCount := msg.RefCount.Add(-1)
	if RefCount == 0 {
		gox.StatNumEndAdd("etcp_msg")
		if msg.Pool != nil {
			msg.Pool.Put(msg)
		}
	} else if RefCount < 0 {
		gox.StatNumEndAdd("etcp_msg_err_put")
		log4.Error("log_tcp_msg RefCount < 0, Stack:%s", string(debug.Stack()))
		panic("log_tcp_msg: Msg reference counter went negative (double Put? or use-after-free?)")
	}
}
