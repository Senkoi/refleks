// Package orchestration owns training background work. Domain services never
// start application loops or depend on Steam/Wails.
package orchestration

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Hooks struct {
	Startup      func(context.Context)
	PollLocal    func(time.Time)
	Tick         func(time.Time)
	DiscoveryDue func(time.Time) bool
	Discover     func(context.Context)
}
type Runner struct {
	cancel      context.CancelFunc
	pending     sync.WaitGroup
	discovering atomic.Bool
}

func Start(parent context.Context, hooks Hooks) *Runner {
	return start(parent, hooks, 15*time.Second, 2*time.Second)
}
func start(parent context.Context, h Hooks, pollInterval, tickInterval time.Duration) *Runner {
	ctx, cancel := context.WithCancel(parent)
	r := &Runner{cancel: cancel}
	r.pending.Add(2)
	go func() {
		defer r.pending.Done()
		if h.Startup != nil {
			h.Startup(ctx)
		}
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if ctx.Err() == nil && h.PollLocal != nil {
					h.PollLocal(now)
				}
			}
		}
	}()
	go func() {
		defer r.pending.Done()
		ticker := time.NewTicker(tickInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if ctx.Err() != nil {
					return
				}
				if h.Tick != nil {
					h.Tick(now)
				}
				if h.DiscoveryDue != nil && h.Discover != nil && h.DiscoveryDue(now) && r.discovering.CompareAndSwap(false, true) {
					r.pending.Add(1)
					go func() { defer r.pending.Done(); defer r.discovering.Store(false); h.Discover(ctx) }()
				}
			}
		}
	}()
	return r
}

// Stop waits for owned work before the app pauses/persists the session.
func (r *Runner) Stop() {
	if r == nil {
		return
	}
	r.cancel()
	r.pending.Wait()
}
