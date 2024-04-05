package cache

import (
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

type mockCleaner struct {
	started bool
	stopped bool
}

func (c *mockCleaner) Start() {
	c.started = true
}

func (c *mockCleaner) Stop() {
	c.stopped = true
}

type cleaner[K comparable, V any] interface {
	// Start cleaning at intervals.
	Start()

	// Stop cleaning.
	Stop()
}

type cacheCleaner[K comparable, V any] struct {
	data            *csmap.CsMap[K, *entry[K, V]]
	cleanupInterval time.Duration
	donechn         chan (struct{})
}

func newCacheCleaner[K comparable, V any](
	data *csmap.CsMap[K, *entry[K, V]],
	cleanupInterval time.Duration,
) cleaner[K, V] {
	return &cacheCleaner[K, V]{
		data:            data,
		cleanupInterval: cleanupInterval,
		donechn:         make(chan struct{}),
	}
}

func (c *cacheCleaner[K, V]) Start() {
	go func() {
		ticker := time.NewTicker(c.cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.donechn:
				return
			}
		}
	}()
}

func (c *cacheCleaner[K, V]) Stop() {
	c.donechn <- struct{}{}
}

func (c *cacheCleaner[K, V]) cleanup() {
	keys := make([]K, 0)
	c.data.Range(func(key K, entry *entry[K, V]) (stop bool) {
		if entry.isExpired() {
			keys = append(keys, key)
		}
		return false
	})
	for _, key := range keys {
		c.data.Delete(key)
	}
}
