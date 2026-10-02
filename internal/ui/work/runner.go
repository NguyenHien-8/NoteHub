// Package work owns the lifetime of background operations for one window.
package work

import (
	"context"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
)

type Runner struct {
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	stopped  bool
	wg       sync.WaitGroup
	pending  atomic.Int64
	dispatch func(func())
}

func New() *Runner { return NewWithDispatcher(fyne.Do) }

// NewWithDispatcher also supports drivers which own a different event queue.
func NewWithDispatcher(dispatch func(func())) *Runner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runner{ctx: ctx, cancel: cancel, dispatch: dispatch}
}

func (r *Runner) Busy() bool { return r.pending.Load() > 0 }

func (r *Runner) Alive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.stopped
}

func (r *Runner) Post(f func()) {
	if !r.Alive() {
		return
	}
	r.dispatch(func() {
		if r.Alive() {
			f()
		}
	})
}

// Stop prevents new work/callbacks and cancels queries. Wait must be called
// outside the UI thread before closing the database used by workers.
func (r *Runner) Stop() {
	r.mu.Lock()
	r.stopped = true
	r.cancel()
	r.mu.Unlock()
}

func (r *Runner) Wait() { r.wg.Wait() }

func Run[T any](r *Runner, operation func(context.Context) (T, error), complete func(T, error)) {
	r.mu.Lock()
	if r.stopped {
		r.mu.Unlock()
		return
	}
	r.wg.Add(1)
	r.pending.Add(1)
	r.mu.Unlock()
	go func() {
		defer r.wg.Done()
		defer r.pending.Add(-1)
		value, err := operation(r.ctx)
		r.Post(func() { complete(value, err) })
	}()
}
