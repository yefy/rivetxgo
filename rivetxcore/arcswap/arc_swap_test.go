package arcswap

import (
	"sync"
	"testing"
)

//go test ./rivetxcore/arcswap -v

type TestConfig struct {
	Value int
}

func TestArcSwap_Basic(t *testing.T) {
	initial := &TestConfig{Value: 1}
	as := NewArcSwap(initial)

	if as.Load().Value != 1 {
		t.Errorf("expected 1, got %d", as.Load().Value)
	}

	newVal := &TestConfig{Value: 2}
	as.Store(newVal)
	if as.Load().Value != 2 {
		t.Errorf("expected 2, got %d", as.Load().Value)
	}

	oldVal := as.Swap(&TestConfig{Value: 3})
	if oldVal.Value != 2 {
		t.Errorf("expected old value 2, got %d", oldVal.Value)
	}
	if as.Load().Value != 3 {
		t.Errorf("expected current value 3, got %d", as.Load().Value)
	}

	current := as.Load()
	success := as.CompareAndSwap(current, &TestConfig{Value: 4})
	if !success {
		t.Error("CAS should have succeeded")
	}

	success = as.CompareAndSwap(current, &TestConfig{Value: 5})
	if success {
		t.Error("CAS should have failed due to outdated pointer")
	}
}

func TestArcSwap_LoadOrStore(t *testing.T) {
	as := NewArcSwap[TestConfig](nil)
	val1 := &TestConfig{Value: 100}

	actual, loaded := as.LoadOrStore(val1)
	if loaded != true {
		t.Error("expected loaded=true for first store")
	}
	if actual != val1 {
		t.Error("returned value should be val1")
	}

	val2 := &TestConfig{Value: 200}
	actual2, loaded2 := as.LoadOrStore(val2)
	if loaded2 != false {
		t.Error("expected loaded=false for second call")
	}
	if actual2 != val1 {
		t.Error("should return existing val1, not val2")
	}
}

func TestArcSwap_Concurrency(t *testing.T) {
	initial := &TestConfig{Value: 0}
	as := NewArcSwap(initial)

	const (
		numGoroutines = 100
		iterations    = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// simulate concurrent read/write mix
				as.Store(&TestConfig{Value: id*iterations + j})
				_ = as.Load()
			}
		}(i)
	}

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				_ = as.Load()
			}
		}
	}()

	wg.Wait()
	close(stop)
}

func TestArcSwap_CASConcurrency(t *testing.T) {
	as := NewArcSwap[TestConfig](nil)
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, loaded := as.LoadOrStore(&TestConfig{Value: 999})
			results <- loaded
		}()
	}

	wg.Wait()
	close(results)

	trueCount := 0
	for r := range results {
		if r {
			trueCount++
		}
	}

	if trueCount != 1 {
		t.Errorf("only one goroutine should have stored the value, got %d", trueCount)
	}
}
