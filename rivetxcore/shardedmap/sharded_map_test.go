package shardedmap

import (
    "runtime"
    "sync"
    "testing"
)

// TestNewShardedMap verifies that the map size is rounded up to the next power of two
// and that the internal mask is set correctly.
func TestNewShardedMap(t *testing.T) {
    t.Parallel()
    sm := NewShardedMap(10) // not a power of two, should become 16
    if len(sm.locks) != 16 {
        t.Fatalf("expected 16 shards, got %d", len(sm.locks))
    }
    if sm.mask != 15 { // 16-1
        t.Fatalf("expected mask 15, got %d", sm.mask)
    }
    // size 1 should stay 1
    sm1 := NewShardedMap(1)
    if len(sm1.locks) != 1 {
        t.Fatalf("expected 1 shard, got %d", len(sm1.locks))
    }
    if sm1.mask != 0 {
        t.Fatalf("expected mask 0, got %d", sm1.mask)
    }
}

// Test basic Load/LoadOrStore/LoadAndDelete semantics.
func TestShardedMapBasicOps(t *testing.T) {
    t.Parallel()
    sm := NewShardedMap(4)
    key := uint64(42)
    // initially not present
    if sm.Load(key) {
        t.Fatalf("expected Load to be false for missing key")
    }
    // first insert should return false (meaning it was absent)
    if sm.LoadOrStore(key) {
        t.Fatalf("expected first LoadOrStore to return false (new entry)")
    }
    // now key exists
    if !sm.Load(key) {
        t.Fatalf("expected Load to be true after insertion")
    }
    // duplicate insert should return true
    if !sm.LoadOrStore(key) {
        t.Fatalf("expected duplicate LoadOrStore to return true (already present)")
    }
    // Delete it
    if !sm.LoadAndDelete(key) {
        t.Fatalf("expected LoadAndDelete to return true for existing key")
    }
    // ensure it's gone
    if sm.Load(key) {
        t.Fatalf("expected Load to be false after deletion")
    }
    // deleting again should return false
    if sm.LoadAndDelete(key) {
        t.Fatalf("expected LoadAndDelete to return false for missing key")
    }
}

// Test concurrent access to ensure no data races and correct final state.
func TestShardedMapConcurrent(t *testing.T) {
    t.Parallel()
    const (
        shards   = 64
        workers  = 8
        perWorker = 1000
    )
    sm := NewShardedMap(shards)
    var wg sync.WaitGroup
    wg.Add(workers)
    for i := 0; i < workers; i++ {
        go func(id int) {
            defer wg.Done()
            base := uint64(id * perWorker)
            for j := 0; j < perWorker; j++ {
                key := base + uint64(j)
                sm.LoadOrStore(key)
            }
        }(i)
    }
    wg.Wait()
    // Verify that all keys are present.
    total := workers * perWorker
    found := 0
    for i := 0; i < total; i++ {
        if sm.Load(uint64(i)) {
            found++
        }
    }
    if found != total {
        t.Fatalf("expected %d keys after concurrent inserts, got %d", total, found)
    }
    // Force GC and ensure no hidden races.
    runtime.GC()
}
