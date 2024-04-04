package cache

import (
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

type Option[K comparable, V any] func(c *cache[K, V])

type Cache[K comparable, V any] interface {
	// Get an item from the cache.
	Get(key K) (V, bool)

	// Put an item into cache.
	Put(key K, value V)

	// Returns true when the item exist in cache.
	Has(key K) bool

	// Returns true if the cache is empty.
	IsEmpty() bool

	// Returns the total count of cached items.
	Count() int

	// Loop over each entry in the cache.
	ForEach(func(key K, value V))

	// Deletes an item from the cache.
	Delete(key K)

	// Clear all items from cache.
	Clear()

	// Cleanup resources and timers.
	Close()
}

// The 'TTL' after it has been written to the cache.
func WithExpireAfterWrite[K comparable, V any](
	expireAfterWrite time.Duration,
) Option[K, V] {
	return func(c *cache[K, V]) {
		c.expireAfterWrite = expireAfterWrite
		c.cleaner = newCacheCleaner(c.data)

		cleanupInterval := time.Second * 5
		c.cleaner.Start(cleanupInterval)
	}
}

type cache[K comparable, V any] struct {
	data *csmap.CsMap[K, *entry[K, V]]

	mu         loaderMutex[K]
	loaderFunc LoaderFunc[K, V]

	expireAfterWrite time.Duration

	cleaner cleaner[K, V]
}

func NewCache[K comparable, V any](
	options ...Option[K, V],
) Cache[K, V] {
	c := &cache[K, V]{
		data: csmap.Create[K, *entry[K, V]](),
	}

	for _, option := range options {
		option(c)
	}

	return c
}

func (c *cache[K, V]) Get(key K) (V, bool) {
	return c.get(key)
}

func (c *cache[K, V]) Put(key K, value V) {
	c.data.Store(key, c.newEntry(key, value))
}

func (c *cache[K, V]) Has(key K) bool {
	_, found := c.get(key)
	return found
}

func (c *cache[K, V]) IsEmpty() bool {
	return c.Count() == 0
}

func (c *cache[K, V]) Count() int {
	count := 0
	c.forEachEntry(func(key K, entry *entry[K, V]) {
		if entry.isValid() {
			count++
		}
	})
	return count
}

func (c *cache[K, V]) ForEach(fn func(key K, value V)) {
	c.forEachEntry(func(key K, entry *entry[K, V]) {
		if entry.isValid() {
			fn(key, entry.value)
		}
	})
}

func (c *cache[K, V]) Delete(key K) {
	c.data.Delete(key)
}

func (c *cache[K, V]) Clear() {
	c.data.Clear()
}

func (c *cache[K, V]) Close() {
	if c.hasExpireAfterWrite() {
		c.cleaner.Stop()
	}
	c.data.Clear()
}

func (c *cache[K, V]) get(key K) (V, bool) {
	if entry, found := c.data.Load(key); found && entry.isValid() {
		return entry.value, true
	}

	var value V
	return value, false
}

// Loop over each entry, including expired entries
func (c *cache[K, V]) forEachEntry(fn func(key K, entry *entry[K, V])) {
	c.data.Range(func(key K, entry *entry[K, V]) (stop bool) {
		fn(key, entry)
		return false
	})
}

func (c *cache[K, V]) newEntry(key K, value V) *entry[K, V] {
	if c.hasExpireAfterWrite() {
		return newEntry(key, value, time.Now().Add(c.expireAfterWrite))
	}
	return newEntry(key, value, zeroTime)
}

func (c *cache[K, V]) hasExpireAfterWrite() bool {
	return c.expireAfterWrite > 0
}
