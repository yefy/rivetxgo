package arcswap

import (
	"sync"
	"sync/atomic"
)

// ArcSwap 线程安全的指针交换容器
type ArcSwap[T any] struct {
	ptr atomic.Pointer[T]
}

// New 创建新的 ArcSwap
func NewArcSwap[T any](initial *T) *ArcSwap[T] {
	as := &ArcSwap[T]{}
	as.ptr.Store(initial)
	return as
}

// Load 获取当前值的指针
func (as *ArcSwap[T]) Load() *T {
	return as.ptr.Load()
}

// Store 存储新值
func (as *ArcSwap[T]) Store(newVal *T) {
	as.ptr.Store(newVal)
}

// Swap 原子性地交换值并返回旧值
func (as *ArcSwap[T]) Swap(newVal *T) *T {
	return as.ptr.Swap(newVal)
}

// CompareAndSwap CAS操作
func (as *ArcSwap[T]) CompareAndSwap(old, new *T) bool {
	return as.ptr.CompareAndSwap(old, new)
}

// LoadOrStore 加载或存储
func (as *ArcSwap[T]) LoadOrStore(newVal *T) (*T, bool) {
	for {
		current := as.Load()
		if current != nil {
			return current, false
		}
		if as.CompareAndSwap(nil, newVal) {
			return newVal, true
		}
		// 失败了就继续重试
	}
}

// 使用示例
func ExampleUsage() {
	type ServerConfig struct {
		Address string
		Port    int
		Timeout int
	}

	// 初始化
	defaultConfig := &ServerConfig{
		Address: "0.0.0.0",
		Port:    8080,
		Timeout: 30,
	}

	config := NewArcSwap(defaultConfig)

	// 热更新配置
	go func() {
		newConfig := &ServerConfig{
			Address: "127.0.0.1",
			Port:    9090,
			Timeout: 60,
		}
		config.Store(newConfig)
	}()

	// 多个goroutine安全读取
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			current := config.Load()
			_ = current // 使用配置
		}(i)
	}
	wg.Wait()
}
