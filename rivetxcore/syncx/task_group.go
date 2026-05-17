package syncx

import (
	"context"
	"sync"
)

func NewTaskGroup() *TaskGroup {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskGroup{
		WaitGroup: &sync.WaitGroup{},
		Ctx:       ctx,
		Cancel:    cancel,
		IsQuit:    false,
	}
}

type TaskGroup struct {
	WaitGroup *sync.WaitGroup
	Ctx       context.Context
	Cancel    context.CancelFunc
	IsQuit    bool
	QuitOnce  sync.Once
}

func (w *TaskGroup) Quit(isWait bool) {
	w.QuitOnce.Do(func() {
		if w.IsQuit {
			return
		}
		w.IsQuit = true
		w.Cancel()
		if isWait {
			w.Wait()
		}
	})
}

func (w *TaskGroup) Wait() {
	w.WaitGroup.Wait()
}

func (w *TaskGroup) Add(delta int) {
	w.WaitGroup.Add(delta)
}

func (w *TaskGroup) Done() {
	w.WaitGroup.Done()
}

func (w *TaskGroup) Subscribe() <-chan struct{} {
	return w.Ctx.Done()
}
