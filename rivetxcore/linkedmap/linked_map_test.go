package linkedmap

import (
	"sync"
	"testing"
	"fmt"
)

// Test basic Add functionality: adding a new key should succeed and store the value.
func TestLinkedMapAddBasic(t *testing.T) {
	t.Parallel()
	lm := NewLinkedMap(0) // no size limit
	added := lm.Add("key1", "value1")
	if !added {
		t.Fatalf("expected Add to return true for a new key")
	}
	if v, ok := lm.Map.Load("key1"); !ok || v != "value1" {
		t.Fatalf("expected key to be stored with correct value, got %v, exists %v", v, ok)
	}
	// Adding the same key again should return false and not modify the stored value.
	added = lm.Add("key1", "newvalue")
	if added {
		t.Fatalf("expected Add to return false when key already exists")
	}
	if v, _ := lm.Map.Load("key1"); v != "value1" {
		t.Fatalf("value should remain unchanged after duplicate Add, got %v", v)
	}
}

// Test maxSize eviction: when the map exceeds its maxSize, the oldest key should be evicted.
func TestLinkedMapMaxSizeEviction(t *testing.T) {
	t.Parallel()
	const maxSize = 3
	lm := NewLinkedMap(maxSize)
	keys := []string{"k1", "k2", "k3", "k4"}
	for _, k := range keys {
		if !lm.Add(k, k+"_val") {
			t.Fatalf("failed to add key %s", k)
		}
	}
	// After adding 4 keys with maxSize 3, the first key (k1) should be evicted.
	if _, ok := lm.Map.Load("k1"); ok {
		t.Fatalf("expected first key to be evicted due to maxSize limit")
	}
	// The remaining keys should still be present.
	for _, k := range []string{"k2", "k3", "k4"} {
		if v, ok := lm.Map.Load(k); !ok || v != k+"_val" {
			t.Fatalf("expected key %s to exist with correct value, got %v, exists %v", k, v, ok)
		}
	}
	// The internal list length should never exceed maxSize.
	if lm.Keys.Len() > maxSize {
		t.Fatalf("internal keys list length exceeds maxSize: got %d, max %d", lm.Keys.Len(), maxSize)
	}
}

// Test Load for an existing key returns the correct value.
func TestLinkedMapLoadExisting(t *testing.T) {
	t.Parallel()
	lm := NewLinkedMap(0)
	lm.Add("key1", "value1")

	v, ok := lm.Load("key1")
	if !ok {
		t.Fatalf("Load: expected ok=true for existing key, got false")
	}
	if v != "value1" {
		t.Fatalf("Load: expected value1, got %v", v)
	}
}

// Test Load for a non-existing key returns nil, false.
func TestLinkedMapLoadNotFound(t *testing.T) {
	t.Parallel()
	lm := NewLinkedMap(0)

	v, ok := lm.Load("nonexistent")
	if ok {
		t.Fatalf("Load: expected ok=false for missing key, got true")
	}
	if v != nil {
		t.Fatalf("Load: expected nil value for missing key, got %v", v)
	}
}

// Test Load after the key has been evicted due to maxSize.
func TestLinkedMapLoadAfterEviction(t *testing.T) {
	t.Parallel()
	lm := NewLinkedMap(2)
	lm.Add("k1", "v1")
	lm.Add("k2", "v2")
	lm.Add("k3", "v3") // k1 should be evicted

	// evicted key
	if _, ok := lm.Load("k1"); ok {
		t.Fatalf("Load: expected evicted key k1 to be missing")
	}
	// remaining keys
	for _, tc := range []struct{ key, want string }{{"k2", "v2"}, {"k3", "v3"}} {
		v, ok := lm.Load(tc.key)
		if !ok {
			t.Fatalf("Load: expected key %s to exist", tc.key)
		}
		if v != tc.want {
			t.Fatalf("Load: key %s: expected %s, got %v", tc.key, tc.want, v)
		}
	}
}

// Test Load with various value types.
func TestLinkedMapLoadValueTypes(t *testing.T) {
	t.Parallel()
	lm := NewLinkedMap(0)
	lm.Add("int", 42)
	lm.Add("bool", true)
	lm.Add("struct", struct{ X int }{X: 7})

	if v, ok := lm.Load("int"); !ok || v != 42 {
		t.Fatalf("Load: int value mismatch, got %v, ok=%v", v, ok)
	}
	if v, ok := lm.Load("bool"); !ok || v != true {
		t.Fatalf("Load: bool value mismatch, got %v, ok=%v", v, ok)
	}
	if v, ok := lm.Load("struct"); !ok || v != (struct{ X int }{X: 7}) {
		t.Fatalf("Load: struct value mismatch, got %v, ok=%v", v, ok)
	}
}

// Test concurrent Load and Add to verify thread-safety of Load.
func TestLinkedMapConcurrentLoad(t *testing.T) {
	t.Parallel()
	const workers = 10
	lm := NewLinkedMap(0)
	// Pre-populate keys
	for i := 0; i < workers; i++ {
		lm.Add(fmt.Sprintf("key%d", i), i)
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", id)
			v, ok := lm.Load(key)
			if !ok {
				t.Errorf("Load: concurrent load: key %s not found", key)
				return
			}
			if v != id {
				t.Errorf("Load: concurrent load: key %s: expected %d, got %v", key, id, v)
			}
		}(i)
	}
	wg.Wait()
}

// Test concurrent Add calls to ensure thread‑safety.
func TestLinkedMapConcurrentAdd(t *testing.T) {
	t.Parallel()
	const (
		workers   = 10
		perWorker = 100
		maxSize   = 0 // unlimited for this test
	)
	lm := NewLinkedMap(maxSize)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				key := id*perWorker + j // unique key per insertion
				lm.Add(key, key)
			}
		}(i)
	}
	wg.Wait()
	// Verify that all expected keys are present.
	expectedCount := workers * perWorker
	actualCount := 0
	lm.Map.Range(func(k, v interface{}) bool {
		actualCount++
		// each value should equal its key
		if k != v {
			t.Fatalf("mismatch: key %v has value %v", k, v)
		}
		return true
	})
	if actualCount != expectedCount {
		t.Fatalf("expected %d entries after concurrent adds, got %d", expectedCount, actualCount)
	}
	// Ensure the internal linked list length matches the number of entries.
	if lm.Keys.Len() != expectedCount {
		t.Fatalf("linked list length %d does not match map size %d", lm.Keys.Len(), expectedCount)
	}
}
