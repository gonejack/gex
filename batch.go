package gex

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Batch struct {
	mapp  map[string]*Task
	tasks []*Task

	onStart func(t *Task)
	onStop  func(t *Task)

	mux sync.Mutex
}

func (b *Batch) Add(tasks ...*Task) {
	if b.mapp == nil {
		b.mapp = make(map[string]*Task)
	}
	for _, t := range tasks {
		if b.mapp[t.URL()] == nil {
			b.mapp[t.URL()] = t
			b.tasks = append(b.tasks, t)
		}
	}
}
func (b *Batch) OnStart(fn func(t *Task)) {
	b.onStart = fn
}
func (b *Batch) OnStop(fn func(t *Task)) {
	b.onStop = fn
}
func (b *Batch) Run() {
	b.RunAll(context.TODO(), 3)
}
func (b *Batch) RunAll(ctx context.Context, concurrent int) {
	var sema = semaphore.NewWeighted(int64(concurrent))
	var wait sync.WaitGroup
	for i := range b.tasks {
		t := b.tasks[i]
		sema.Acquire(ctx, 1)
		wait.Add(1)
		go func() {
			defer wait.Done()
			defer sema.Release(1)

			b.syncRun(t, b.onStart)
			_ = t.Do(ctx)
			b.syncRun(t, b.onStop)
		}()
	}
	wait.Wait()
}
func (b *Batch) syncRun(t *Task, fn func(t *Task)) {
	b.mux.Lock()
	defer b.mux.Unlock()
	t.hook(fn)
}
