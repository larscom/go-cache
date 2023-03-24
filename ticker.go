package cache

import (
	"context"
	"sync"
	"time"
)

type ticker struct {
	ctx      context.Context
	interval time.Duration
	done     chan struct{}
	once     sync.Once
}

func newTicker(c context.Context, i time.Duration) *ticker {
	return &ticker{
		ctx:      c,
		interval: i,
		done:     make(chan struct{}),
	}
}

func (t *ticker) start(tick func()) {
	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tick()
			case <-t.done:
				tick()
				return
			case <-t.ctx.Done():
				t.stop()
			}
		}
	}()
}

func (t *ticker) stop() {
	t.once.Do(func() {
		close(t.done)
	})
}
