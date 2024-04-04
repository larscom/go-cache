package cache

import (
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

type cleaner[K comparable, V any] interface {
	// Starts cleaning at the given interval.
	Start(i time.Duration)

	// Stop cleaning.
	Stop()
}

type cacheCleaner[K comparable, V any] struct {
	data    *csmap.CsMap[K, *entry[K, V]]
	donechn chan (struct{})
}

func newCacheCleaner[K comparable, V any](
	data *csmap.CsMap[K, *entry[K, V]],
) cleaner[K, V] {
	return &cacheCleaner[K, V]{
		data:    data,
		donechn: make(chan struct{}),
	}
}

func (c *cacheCleaner[K, V]) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
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
