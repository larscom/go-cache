package cache

import (
	"container/list"
	"context"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type Option[Key comparable, Value any] func(c *Cache[Key, Value])

type LoaderFunc[Key comparable, Value any] func(Key) (Value, error)

type ICache[Key comparable, Value any] interface {
	// Clears the whole cache
	Clear()
	// Stop the timers
	Close()
	// Total amount of entries
	Count() int
	// Loop over each entry in the cache
	ForEach(func(Key, Value))
	// If cache is created with 'WithLoader' it'll use the loader function to get
	// an item and put it in the cache.
	// The loader function is only ever called once, even if it's called from multiple goroutines.
	Get(Key) (Value, bool, error)
	// Check to see if the cache contains a key
	Has(Key) bool
	// If cache is created with 'WithMaxSize' option you get the keys in order from oldest to newest.
	// Otherwise the keys will be in an indeterminate order.
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

type Cache[Key comparable, Value any] struct {
	entries    map[Key]*cacheEntry[Key, Value]
	entryQueue *list.List

	maxSize int
	loader  LoaderFunc[Key, Value]

	expireAfterWrite time.Duration
	onExpired        func(Key, Value)

	mu      sync.RWMutex
	keyedMu keyedMutex[Key]

	cancel context.CancelFunc
	ticker *ticker
}

func NewCache[Key comparable, Value any](
	options ...Option[Key, Value],
) ICache[Key, Value] {
	c := &Cache[Key, Value]{
		entries: make(map[Key]*cacheEntry[Key, Value]),
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func (c *Cache[Key, Value]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[Key]*cacheEntry[Key, Value])
}

func (c *Cache[Key, Value]) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Cache[Key, Value]) Count() int {
	return len(c.nonExpiredEntries())
}

func (c *Cache[Key, Value]) ForEach(fn func(Key, Value)) {
	for key, entry := range c.nonExpiredEntries() {
		fn(key, entry.value)
	}
}

func (c *Cache[Key, Value]) Get(key Key) (Value, bool, error) {
	unlock := c.keyedMu.lock(key)
	defer unlock()

	entry, found := c.getSafe(key)
	if found && !entry.isExpired() {
		return entry.value, true, nil
	}

	value, ok, err := c.load(key)

	if ok {
		c.Put(key, value)
	}

	return value, ok, err
}

func (c *Cache[Key, Value]) Has(key Key) bool {
	entry, found := c.getSafe(key)
	return found && !entry.isExpired()
}

func (c *Cache[Key, Value]) Keys() []Key {
	entries := c.nonExpiredEntries()
	if c.maxSize > 0 {
		keys := make([]Key, 0, len(entries))
		for e := c.entryQueue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*cacheEntry[Key, Value])
			if !entry.isExpired() {
				keys = append(keys, entry.key)
			}
		}
		return keys
	}
	return maps.Keys(entries)
}

func (c *Cache[Key, Value]) Put(key Key, value Value) {
	entry := c.newEntry(key, value)

	if c.maxSize > 0 {
		if c.entryQueue.Len() == c.maxSize {
			e := c.dequeueSafe()
			c.removeSafe(e.Value.(*cacheEntry[Key, Value]).key)
		}
		c.entryQueue.PushBack(entry)
	}

	c.putSafe(entry)
}

func (c *Cache[Key, Value]) Remove(key Key) {
	if c.maxSize > 0 {
		for e := c.entryQueue.Front(); e != nil; e = e.Next() {
			entry := e.Value.(*cacheEntry[Key, Value])
			if entry.key == key {
				c.entryQueue.Remove(e)
				break
			}
		}
	}
	c.removeSafe(key)
}

func (c *Cache[Key, Value]) ToMap() map[Key]Value {
	m := make(map[Key]Value)
	for key, entry := range c.nonExpiredEntries() {
		m[key] = entry.value
	}
	return m
}

func (c *Cache[Key, Value]) Values() []Value {
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
	return func(c *Cache[Key, Value]) {
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
	return func(c *Cache[Key, Value]) {
		c.loader = loader
	}
}

func WithOnExpired[Key comparable, Value any](
	onExpired func(Key, Value),
) Option[Key, Value] {
	return func(c *Cache[Key, Value]) {
		c.onExpired = onExpired
	}
}

func WithMaxSize[Key comparable, Value any](
	maxSize int,
) Option[Key, Value] {
	return func(c *Cache[Key, Value]) {
		c.entries = make(map[Key]*cacheEntry[Key, Value], maxSize)
		c.maxSize = maxSize
		c.entryQueue = list.New()
	}
}

func (c *Cache[Key, Value]) newEntry(key Key, value Value) *cacheEntry[Key, Value] {
	var expiration time.Time

	if c.expireAfterWrite > 0 {
		expiration = time.Now().Add(c.expireAfterWrite)
	}

	return &cacheEntry[Key, Value]{key, value, expiration}
}

func (c *Cache[Key, Value]) nonExpiredEntries() map[Key]*cacheEntry[Key, Value] {
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

func (c *Cache[Key, Value]) cleanup() {
	c.mu.RLock()
	keys := maps.Keys(c.entries)
	c.mu.RUnlock()

	for _, key := range keys {
		entry, found := c.getSafe(key)
		if found && entry.isExpired() {
			c.Remove(key)
			if c.onExpired != nil {
				c.onExpired(entry.key, entry.value)
			}
		}
	}
}

func (c *Cache[Key, Value]) load(key Key) (Value, bool, error) {
	if c.loader == nil {
		var val Value
		return val, false, nil
	}

	value, err := c.loader(key)

	return value, err == nil, err
}

func (c *Cache[Key, Value]) getSafe(key Key) (*cacheEntry[Key, Value], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, found := c.entries[key]
	return entry, found
}

func (c *Cache[Key, Value]) putSafe(entry *cacheEntry[Key, Value]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[entry.key] = entry
}

func (c *Cache[Key, Value]) removeSafe(key Key) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *Cache[Key, Value]) dequeueSafe() *list.Element {
	e := c.entryQueue.Front()
	c.entryQueue.Remove(e)
	return e
}
