package orchestration

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStopCancelsStartupAndDiscoveryAndJoinsWork(t *testing.T) {
	entered := make(chan struct{}, 2)
	exited := make(chan struct{}, 2)
	var count atomic.Int32
	r := start(context.Background(), Hooks{Startup: func(ctx context.Context) { entered <- struct{}{}; <-ctx.Done(); exited <- struct{}{} }, DiscoveryDue: func(time.Time) bool { return true }, Discover: func(ctx context.Context) { count.Add(1); entered <- struct{}{}; <-ctx.Done(); exited <- struct{}{} }}, time.Millisecond, time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("owned work did not start")
		}
	}
	r.Stop()
	if count.Load() != 1 || len(exited) != 2 {
		t.Fatal("discovery not coalesced or workers not joined")
	}
	r.Stop()
}
