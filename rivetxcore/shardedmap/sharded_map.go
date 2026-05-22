package shardedmap

import (
	"sync"
	"sync/atomic"
)

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

const cacheLineSize = 64

type paddedMutex struct {
	sync.RWMutex
	_ [cacheLineSize - 8]byte
}

type ShardedMap struct {
	mask  uint64
	locks []paddedMutex
	maps  []map[uint64]struct{}
	size atomic.Int64
}

func NewShardedMap(size int) *ShardedMap {
	actualSize := nextPowerOfTwo(size)

	locks := make([]paddedMutex, actualSize)
	maps := make([]map[uint64]struct{}, actualSize)

	for i := 0; i < actualSize; i++ {
		maps[i] = make(map[uint64]struct{}, 1000)
	}

	return &ShardedMap{
		mask:  uint64(actualSize - 1),
		locks: locks,
		maps:  maps,
	}
}

func (s *ShardedMap) getShard(key uint64) int {
	return int(key & s.mask)
}

func (s *ShardedMap) Load(key uint64) bool {
	idx := s.getShard(key)
	s.locks[idx].RLock()
	_, ok := s.maps[idx][key]
	s.locks[idx].RUnlock()
	return ok
}

func (s *ShardedMap) LoadOrStore(key uint64) bool {
	idx := s.getShard(key)

	s.locks[idx].RLock()
	_, ok := s.maps[idx][key]
	s.locks[idx].RUnlock()
	if ok {
		return true
	}

	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	if _, ok = s.maps[idx][key]; ok {
		return true
	}

	s.maps[idx][key] = struct{}{}
	s.size.Add(1)
	return false
}

func (s *ShardedMap) LoadAndDelete(key uint64) bool {
	idx := s.getShard(key)

	s.locks[idx].RLock()
	_, ok := s.maps[idx][key]
	s.locks[idx].RUnlock()
	if !ok {
		return false
	}

	s.locks[idx].Lock()
	defer s.locks[idx].Unlock()

	_, ok = s.maps[idx][key]

	if ok {
		delete(s.maps[idx], key)
		s.size.Add(-1)
	}
	return ok
}


func (s *ShardedMap) Size() int64 {
	return s.size.Load()
}
