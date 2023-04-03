package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type Option[Key comparable, Value any] func(c *CacheImpl[Key, Value])

type LoaderFunc[Key comparable, Value any] func(Key) (Value, error)

type Cache[Key comparable, Value any] interface {
	// Clears the whole cache
	Clear()
	// Stop the timers
	Close()
	// Total amount of entries
	Count() int
	// Loop over each entry in the cache
	ForEach(func(Key, Value))
	// Get item with the loader function (if configured)
	// it is only ever called once, even if it's called from multiple goroutines.
	// When no loader is configured, use GetIfPresent instead
	Get(Key) (Value, error)
	// Get item from cache (if present) without loader
	GetIfPresent(Key) (Value, bool)
	// Refresh item in cache
	Refresh(Key) (Value, error)
	// Check to see if the cache contains a key
	Has(Key) bool
	// Get all keys, it will be in indeterminate order.
	Keys() []Key
	// Add a new item to the cache
	Put(Key, Value)
	// Remove an item from the cache
	Remove(Key)
	// Get the map with the key/value pairs, it will be in indeterminate order.
	ToMap() map[Key]Value
	// Get all values, it will be in indeterminate order.
	Values() []Value
}

type CacheImpl[Key comparable, Value any] struct {
	entries map[Key]*cacheEntry[Key, Value]
	loader  LoaderFunc[Key, Value]

	expireAfterWrite time.Duration
	onExpired        func(Key, Value)

	mu  sync.RWMutex
	kmu KeyedMutex[Key]

	cancel context.CancelFunc
	ticker *ticker
}

func NewCache[Key comparable, Value any](
	options ...Option[Key, Value],
) Cache[Key, Value] {
	c := &CacheImpl[Key, Value]{
		entries: make(map[Key]*cacheEntry[Key, Value]),
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func (c *CacheImpl[Key, Value]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[Key]*cacheEntry[Key, Value])
}

func (c *CacheImpl[Key, Value]) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *CacheImpl[Key, Value]) Count() int {
	return len(c.nonExpiredEntries())
}

func (c *CacheImpl[Key, Value]) ForEach(fn func(Key, Value)) {
	for key, entry := range c.nonExpiredEntries() {
		fn(key, entry.value)
	}
}

func (c *CacheImpl[Key, Value]) Get(key Key) (Value, error) {
	unlock := c.kmu.lock(key)

	entry, found := c.getSafe(key)
	if found && !entry.isExpired() {
		unlock()
		return entry.value, nil
	}

	value, err := c.load(key)

	unlock()

	if err == nil {
		c.Put(key, value)
	}

	return value, err
}

func (c *CacheImpl[Key, Value]) GetIfPresent(key Key) (Value, bool) {
	entry, found := c.getSafe(key)

	if found && !entry.isExpired() {
		return entry.value, true
	}

	var value Value
	return value, false
}

func (c *CacheImpl[Key, Value]) Refresh(key Key) (Value, error) {
	unlock := c.kmu.lock(key)

	value, err := c.load(key)

	unlock()

	if err == nil {
		c.Put(key, value)
	}

	return value, err
}

func (c *CacheImpl[Key, Value]) Has(key Key) bool {
	_, found := c.GetIfPresent(key)
	return found
}

func (c *CacheImpl[Key, Value]) Keys() []Key {
	return maps.Keys(c.nonExpiredEntries())
}

func (c *CacheImpl[Key, Value]) Put(key Key, value Value) {
	entry := c.newEntry(key, value)
	c.putSafe(entry)
}

func (c *CacheImpl[Key, Value]) Remove(key Key) {
	c.removeSafe(key)
}

func (c *CacheImpl[Key, Value]) ToMap() map[Key]Value {
	m := make(map[Key]Value)
	for key, entry := range c.nonExpiredEntries() {
		m[key] = entry.value
	}
	return m
}

func (c *CacheImpl[Key, Value]) Values() []Value {
	entries := maps.Values(c.nonExpiredEntries())
	n := len(entries)
	values := make([]Value, n)
	for i := 0; i < n; i++ {
		values[i] = entries[i].value
	}
	return values
}

func WithExpireAfterWrite[Key comparable, Value any](
	expireAfterWrite time.Duration,
) Option[Key, Value] {
	return WithExpireAfterWriteCustom[Key, Value](expireAfterWrite, time.Minute)
}

func WithExpireAfterWriteCustom[Key comparable, Value any](
	expireAfterWrite time.Duration,
	cleanupInterval time.Duration,
) Option[Key, Value] {
	return func(c *CacheImpl[Key, Value]) {
		c.expireAfterWrite = expireAfterWrite
		if c.ticker == nil {
			ctx, cancel := context.WithCancel(context.Background())
			c.cancel = cancel
			c.ticker = newTicker(ctx, cleanupInterval)
			c.ticker.start(c.cleanup)
		}
	}
}

func WithLoader[Key comparable, Value any](
	loader LoaderFunc[Key, Value],
) Option[Key, Value] {
	return func(c *CacheImpl[Key, Value]) {
		c.loader = loader
	}
}

func WithOnExpired[Key comparable, Value any](
	onExpired func(Key, Value),
) Option[Key, Value] {
	return func(c *CacheImpl[Key, Value]) {
		c.onExpired = onExpired
	}
}

func (c *CacheImpl[Key, Value]) newEntry(key Key, value Value) *cacheEntry[Key, Value] {
	var expiration time.Time

	if c.expireAfterWrite > 0 {
		expiration = time.Now().Add(c.expireAfterWrite)
	}

	return &cacheEntry[Key, Value]{key, value, expiration}
}

func (c *CacheImpl[Key, Value]) nonExpiredEntries() map[Key]*cacheEntry[Key, Value] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e := make(map[Key]*cacheEntry[Key, Value])
	for key, entry := range c.entries {
		if !entry.isExpired() {
			e[key] = entry
		}
	}
	return e
}

func (c *CacheImpl[Key, Value]) cleanup() {
	c.mu.RLock()
	keys := maps.Keys(c.entries)
	c.mu.RUnlock()

	for _, key := range keys {
		entry, found := c.getSafe(key)
		if found && entry.isExpired() {
			c.removeSafe(key)
			if c.onExpired != nil {
				c.onExpired(entry.key, entry.value)
			}
		}
	}
}

func (c *CacheImpl[Key, Value]) load(key Key) (Value, error) {
	if c.loader == nil {
		var val Value
		return val, fmt.Errorf("you must configure a loader, use GetIfPresent instead")
	}

	value, err := c.loader(key)

	return value, err
}

func (c *CacheImpl[Key, Value]) getSafe(key Key) (*cacheEntry[Key, Value], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, found := c.entries[key]
	return entry, found
}

func (c *CacheImpl[Key, Value]) putSafe(entry *cacheEntry[Key, Value]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[entry.key] = entry
}

func (c *CacheImpl[Key, Value]) removeSafe(key Key) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}
