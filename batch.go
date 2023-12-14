package gex

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

type batch struct {
	uniq map[string]*Request
	list []*Request

	before func(r *Request)
	after  func(r *Request, err error)

	se *semaphore.Weighted
	mu sync.Mutex
}

func (b *batch) Add(requests ...*Request) {
	if b.uniq == nil {
		b.uniq = make(map[string]*Request)
	}
	for _, r := range requests {
		if b.uniq[r.Url] == nil {
			b.uniq[r.Url] = r
			b.list = append(b.list, r)
		}
	}
}
func (b *batch) Reset() {
	b.uniq = nil
	b.list = nil
}
func (b *batch) OnStart(fn func(r *Request)) {
	b.before = fn
}
func (b *batch) OnStop(fn func(r *Request, err error)) {
	b.after = fn
}
func (b *batch) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.TODO()
	}
	var g errgroup.Group
	for i := range b.list {
		r := b.list[i]
		b.se.Acquire(ctx, 1)
		g.Go(func() error {
			defer b.se.Release(1)
			b.sync(func() { b.before(r) })
			err := r.Do(ctx)
			b.sync(func() { b.after(r, err) })
			return nil
		})
	}
	g.Wait()
}
func (b *batch) sync(f func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	f()
}

func NewBatch(concurrent int) *batch {
	return &batch{
		se:     semaphore.NewWeighted(int64(concurrent)),
		before: func(r *Request) {},
		after:  func(r *Request, err error) {},
	}
}
