package linkedmap

import (
	"container/list"
	"sync"
)

func NewLinkedMap(maxSize int) *LinkedMap {
	return &LinkedMap{
		Keys:    list.New(),
		maxSize: maxSize,
	}
}

type LinkedMap struct {
	Map     sync.Map
	Mutex   sync.Mutex
	Keys    *list.List
	maxSize int
}

func (m *LinkedMap) Add(key interface{}, value interface{}) bool {
	_, ok := m.Map.LoadOrStore(key, value)
	if !ok {
		m.Mutex.Lock()
		defer m.Mutex.Unlock()
		m.Keys.PushBack(key)
		if m.maxSize > 0 {
			if m.Keys.Len() > m.maxSize {
				if m.Keys.Front() != nil {
					removed := m.Keys.Remove(m.Keys.Front())
					m.Map.Delete(removed)
				}
			}
		}
		return true
	}
	return false
}
