package gex

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type batch struct {
	keep     map[string]*Request
	requests []*Request
	sema     *semaphore.Weighted
	mutex    sync.Mutex
	onStart  func(r *Request)
	onStop   func(r *Request, err error)
}

func (b *batch) Add(requests ...*Request) {
	if b.keep == nil {
		b.keep = make(map[string]*Request)
	}
	for _, r := range requests {
		if b.keep[r.Url] == nil {
			b.keep[r.Url] = r
			b.requests = append(b.requests, r)
		}
	}
}
func (b *batch) Reset() {
	b.keep = nil
	b.requests = nil
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
	var g errgroup.Group
	for i := range b.requests {
		r := b.requests[i]
		b.sema.Acquire(ctx, 1)
		g.Go(func() error {
			defer b.sema.Release(1)
			b.sync(func() { b.onStart(r) })
			err := r.Do(ctx)
			b.sync(func() { b.onStop(r, err) })
			return nil
		})
	}
	g.Wait()
}
func (b *batch) sync(f func()) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	f()
}

func NewBatch(concurrent int) *batch {
	return &batch{
		sema:    semaphore.NewWeighted(int64(concurrent)),
		onStart: func(r *Request) {},
		onStop:  func(r *Request, err error) {},
	}
}
