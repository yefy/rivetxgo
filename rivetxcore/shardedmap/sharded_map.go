package shardedmap

import (
	"sync"
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
	}
	return ok
}
