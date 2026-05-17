package syncx

import (
	"testing"
	"time"
)

// Test basic Add, Done, and Wait behavior.
func TestTaskGroupBasic(t *testing.T) {
	tg := NewTaskGroup()
	tg.Add(1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(10 * time.Millisecond)
		tg.Done()
	}()
	tg.Wait()
	select {
	case <-done:
		// success
	default:
		t.Fatalf("Wait returned before task completed")
	}
}

// Test that Quit without waiting cancels the context immediately.
func TestTaskGroupQuitCancel(t *testing.T) {
	tg := NewTaskGroup()
	ch := tg.Subscribe()
	select {
	case <-ch:
		t.Fatalf("channel closed before Quit")
	default:
	}
	tg.Quit(false)
	select {
	case <-ch:
		// ok
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("channel not closed after Quit")
	}
	tg.Quit(false) // idempotent
}

// Test Quit with waiting for pending tasks.
func TestTaskGroupQuitWait(t *testing.T) {
	tg := NewTaskGroup()
	tg.Add(1)
	go func() {
		time.Sleep(30 * time.Millisecond)
		tg.Done()
	}()
	start := time.Now()
	tg.Quit(true)
	elapsed := time.Since(start)
	if elapsed < 30*time.Millisecond {
		t.Fatalf("Quit(true) returned too early, elapsed %v", elapsed)
	}
	select {
	case <-tg.Subscribe():
		// ok
	default:
		t.Fatalf("context not cancelled after Quit")
	}
	tg.Quit(true) // safe to call again
}

// Ensure that Done without prior Add panics.
func TestTaskGroupDoneWithoutAdd(t *testing.T) {
	tg := NewTaskGroup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when Done without prior Add")
		}
	}()
	tg.Done()
}
