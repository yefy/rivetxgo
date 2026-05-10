package gox

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetChanPool(t *testing.T) {
	// Test getting a channel from pool
	ch := GetChan(10)
	if ch == nil {
		t.Fatal("GetChan returned nil")
	}
	if cap(ch.Ch) != 10 {
		t.Errorf("Expected channel capacity 10, got %d", cap(ch.Ch))
	}
	ch.Put() // Return to pool

	// Test large size (not from pool)
	ch2 := GetChan(5000)
	if ch2 == nil {
		t.Fatal("GetChan for large size returned nil")
	}
	if cap(ch2.Ch) != 5000 {
		t.Errorf("Expected channel capacity 5000, got %d", cap(ch2.Ch))
	}
	// No pool for large channels, so no Put
}

func TestSpawnFunc(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	Spawn(func(spawnId uint64) error {
		defer wg.Done()
		if spawnId == 0 {
			t.Error("spawnId should not be 0")
		}
		return nil
	})

	wg.Wait()
}

func TestUniqSpawnFunc(t *testing.T) {
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Test non-waiting spawn
	wg.Add(1)
	UniqSpawn("test1", false, func() error {
		defer wg.Done()
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	})

	wg.Wait()
	if counter != 1 {
		t.Errorf("Expected counter 1, got %d", counter)
	}

	// Test waiting spawn
	counter = 0
	UniqSpawn("test1", true, func() error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	})

	if counter != 1 {
		t.Errorf("Expected counter 1 after wait, got %d", counter)
	}
}

func TestBatchSpawnFunc(t *testing.T) {
	var processed []interface{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	name := "batch_test"
	BatchSpawn(name, 3, 100*time.Millisecond, func(datas []interface{}) error {
		mu.Lock()
		processed = append(processed, datas...)
		mu.Unlock()
		wg.Add(-len(datas))
		return nil
	})

	// Add data
	for i := 0; i < 5; i++ {
		wg.Add(1)
		BatchAdd(name, i)
	}

	// Flush remaining
	BatchFlush(name)

	// Wait for processing
	wg.Wait()

	BatchClose(name)

	mu.Lock()
	defer mu.Unlock()
	if len(processed) != 5 {
		t.Errorf("Expected 5 processed items, got %d", len(processed))
	}
}

func TestTimerSpawnFunc(t *testing.T) {
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	name := "timer_test"
	wg.Add(1)

	TimerSpawn(name, true, 50*time.Millisecond, func() (bool, error) {
		mu.Lock()
		counter++
		mu.Unlock()
		if counter >= 3 {
			wg.Done()
			return true, nil // Quit after 3 calls
		}
		return false, nil
	})

	wg.Wait()

	if counter != 3 {
		t.Errorf("Expected counter 3, got %d", counter)
	}
}

func TestListSpawnFunc(t *testing.T) {
	var processed []interface{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	name := "list_test"
	ListSpawn(name, 10, func(data interface{}) error {
		if data == nil {
			return nil
		}
		mu.Lock()
		processed = append(processed, data)
		mu.Unlock()
		wg.Done()
		return nil
	})

	// Add data
	for i := 0; i < 5; i++ {
		wg.Add(1)
		ListAdd(name, i)
	}

	// Close
	ListClose(name)

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(processed) != 5 {
		t.Errorf("Expected 5 processed items, got %d", len(processed))
	}
}

func TestNewBatchSpawns(t *testing.T) {
	bs := NewBatchSpawns(5)
	if bs == nil {
		t.Fatal("NewBatchSpawns returned nil")
	}
	if bs.BatchSize != 5 {
		t.Errorf("Expected BatchSize 5, got %d", bs.BatchSize)
	}
}

func TestNewUniqSpawns(t *testing.T) {
	us := NewUniqSpawns()
	if us == nil {
		t.Fatal("NewUniqSpawns returned nil")
	}
	if us.DataChan == nil {
		t.Fatal("DataChan is nil")
	}
}

func TestNewTimerSpawns(t *testing.T) {
	ts := NewTimerSpawns()
	if ts == nil {
		t.Fatal("NewTimerSpawns returned nil")
	}
}

func TestNewListSpawns(t *testing.T) {
	ls := NewListSpawns(10)
	if ls == nil {
		t.Fatal("NewListSpawns returned nil")
	}
	if cap(ls.Chan) != 10 {
		t.Errorf("Expected channel capacity 10, got %d", cap(ls.Chan))
	}
}

func TestSetUniqSpawnCtx(t *testing.T) {
	ctx := context.Background()
	SetUniqSpawnCtx(ctx)
	// Note: This function just assigns to a global variable, hard to test directly
}
