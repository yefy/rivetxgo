package arcswap

import (
	"sync"
	"sync/atomic"
)

type ArcSwap[T any] struct {
	ptr atomic.Pointer[T]
}

func NewArcSwap[T any](initial *T) *ArcSwap[T] {
	as := &ArcSwap[T]{}
	as.ptr.Store(initial)
	return as
}

func (as *ArcSwap[T]) Load() *T {
	return as.ptr.Load()
}

func (as *ArcSwap[T]) Store(newVal *T) {
	as.ptr.Store(newVal)
}

func (as *ArcSwap[T]) Swap(newVal *T) *T {
	return as.ptr.Swap(newVal)
}

func (as *ArcSwap[T]) CompareAndSwap(old, new *T) bool {
	return as.ptr.CompareAndSwap(old, new)
}

func (as *ArcSwap[T]) LoadOrStore(newVal *T) (*T, bool) {
	for {
		current := as.Load()
		if current != nil {
			return current, false
		}
		if as.CompareAndSwap(nil, newVal) {
			return newVal, true
		}
	}
}
