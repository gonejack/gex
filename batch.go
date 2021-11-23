package gex

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type batch struct {
	mapp map[string]*Request
	reqs []*Request
	sema *semaphore.Weighted
	mux  sync.Mutex

	onStart func(r *Request)
	onStop  func(r *Request, err error)
}

func (b *batch) Add(requests ...*Request) {
	if b.mapp == nil {
		b.mapp = make(map[string]*Request)
	}
	for _, r := range requests {
		if b.mapp[r.Url] == nil {
			b.mapp[r.Url] = r
			b.reqs = append(b.reqs, r)
		}
	}
}
func (b *batch) Reset() {
	b.mapp = nil
	b.reqs = nil
}
func (b *batch) OnStart(fn func(r *Request)) {
	b.onStart = fn
}
func (b *batch) OnStop(fn func(r *Request, err error)) {
	b.onStop = fn
}
func (b *batch) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.TODO()
	}

	var grp errgroup.Group
	for i := range b.reqs {
		r := b.reqs[i]
		b.sema.Acquire(ctx, 1)

		grp.Go(func() error {
			defer b.sema.Release(1)

			b.syncRun(func() { b.onStart(r) })
			err := r.Do(ctx)
			b.syncRun(func() { b.onStop(r, err) })

			return nil
		})
	}
	grp.Wait()
}
func (b *batch) syncRun(f func()) {
	b.mux.Lock()
	defer b.mux.Unlock()
	f()
}

func NewBatch(concurrent int) *batch {
	return &batch{
		sema:    semaphore.NewWeighted(int64(concurrent)),
		onStart: func(r *Request) {},
		onStop:  func(r *Request, err error) {},
	}
}
