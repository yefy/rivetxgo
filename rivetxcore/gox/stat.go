package gox

import (
	"github.com/shirou/gopsutil/v3/process"
	"github.com/yefy/log4go/log4"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type StatFunc func()

var StatFuncs = make([]StatFunc, 0, 100)

var StatNum = sync.Map{}

type StatNumContext struct {
	StartNum atomic.Int64
	EndNum   atomic.Int64
}

func init() {
	Spawn(func(u uint64) error {
		for {
			time.Sleep(time.Duration(10) * time.Second)

			for _, f := range StatFuncs {
				f()
			}

			type Data struct {
				Key       string
				MallocNum int64
				FreeNum   int64
				RunNum    int64
			}

			datas := make([]Data, 0, 50)

			StatNum.Range(func(keyI, valueI any) bool {
				key := keyI.(string)
				value := valueI.(*StatNumContext)
				StartNum := value.StartNum.Load()
				EndNum := value.EndNum.Load()
				diffNum := StartNum - EndNum
				datas = append(datas, Data{key, StartNum, EndNum, diffNum})
				return true
			})

			sort.Slice(datas, func(i, j int) bool {
				return datas[i].Key < datas[j].Key
			})

			for _, data := range datas {
				log4.Info("stat_key_log %+v", data)
			}

			monitorPool()
		}
	})
}

func monitorPool() {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		log4.Error("Failed to get process info: %v", err)
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Go 堆和栈
	log4.Info("pool_log Heap: Alloc=%vMB, Sys=%vMB, Idle=%vMB, Inuse=%vMB, Released=%vMB, HeapObjects=%v",
		m.HeapAlloc/1024/1024,
		m.HeapSys/1024/1024,
		m.HeapIdle/1024/1024,
		m.HeapInuse/1024/1024,
		m.HeapReleased/1024/1024,
		m.HeapObjects)

	log4.Info("pool_log Stack: Inuse=%vMB, Sys=%vMB, MSpanInuse=%vMB, MSpanSys=%vMB, MCacheInuse=%vMB, MCacheSys=%vMB",
		m.StackInuse/1024/1024,
		m.StackSys/1024/1024,
		m.MSpanInuse/1024/1024,
		m.MSpanSys/1024/1024,
		m.MCacheInuse/1024/1024,
		m.MCacheSys/1024/1024)

	// GC 状态
	log4.Info("pool_log GC: NumGC=%v, PauseTotal=%vms, NextGC=%vMB, LastGC=%v",
		m.NumGC,
		float64(m.PauseTotalNs)/1e6,
		m.NextGC/1024/1024,
		m.LastGC)

	// 基本分配统计
	log4.Info("pool_log Goroutines/Alloc/Mallocs/Frees: Goroutines:%v, Alloc=%vMB, TotalAlloc=%vMB, Mallocs=%v, Frees=%v",
		runtime.NumGoroutine(),
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Mallocs,
		m.Frees)

	// OS 进程内存
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		log4.Info("Failed to get OS memory info: %v", err)
		return
	}
	log4.Info("pool_log OS Memory: RSS=%vMB, VMS=%vMB, Swap=%vMB",
		memInfo.RSS/1024/1024,
		memInfo.VMS/1024/1024,
		memInfo.Swap/1024/1024)

	log4.Info("pool_log --------------------------------------------------")
}

func StatFuncAdd(f StatFunc) {
	StatFuncs = append(StatFuncs, f)
}

func StatNumDataGet(name string) *StatNumContext {
	dataI, ok := StatNum.Load(name)
	if !ok {
		dataI, _ = StatNum.LoadOrStore(name, &StatNumContext{})
	}
	data := dataI.(*StatNumContext)
	return data
}

func StatNumStartAdd(name string) {
	data := StatNumDataGet(name)
	data.StartNum.Add(1)
}

func StatNumEndAdd(name string) {
	data := StatNumDataGet(name)
	data.EndNum.Add(1)

}

func StatNumStartAddN(name string, num int64) {
	data := StatNumDataGet(name)
	data.StartNum.Add(num)
}

func StatNumEndAddN(name string, num int64) {
	data := StatNumDataGet(name)
	data.EndNum.Add(num)

}

func StatNumStartSet(name string, num int64) {
	data := StatNumDataGet(name)
	data.StartNum.Store(num)
}

func StatNumEndSet(name string, num int64) {
	data := StatNumDataGet(name)
	data.EndNum.Store(num)

}

func StatNumData(name string) (int64, int64) {
	data := StatNumDataGet(name)
	return data.StartNum.Load(), data.EndNum.Load()
}
