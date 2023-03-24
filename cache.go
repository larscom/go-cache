package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type Option[Key comparable, Value any] func(c *Cache[Key, Value])

type LoaderFunc[Key comparable, Value any] func(Key) (Value, error)

type ICache[Key comparable, Value any] interface {
	// Clears the whole cache
	Clear()
	// Close any remaining timers
	Close()
	// Total amount of entries
	Count() int
	// Loop over each entry in the cache
	ForEach(func(Key, Value))
	// If cache is created with 'WithLoader' it'll use the loader function to get
	// an item if it's not available in the cache.
	// It'll be in the cache afterwards
	// The loader function is only ever called once, even if multiple goroutines ask for it.
	Get(Key) (Value, bool, error)
	// Check to see if the cache contains a key
	Has(Key) bool
	// If cached is created with 'WithMaxSize' option you get the keys in order from oldest to newest.
	// Otherwise the keys will be in an indeterminate order.
	Keys() []Key
	// Add a new item to the cache
	Put(Key, Value)
	// Reload the item in the cache, this will remove the entry from the cache first and
	// call the loader function afterwards.
	Reload(Key) (Value, bool, error)
	// Remove one item from the cache
	Remove(Key)
	// Get the map with the key/value pairs, it will be in indeterminate order.
	ToMap() map[Key]Value
	// Get all values, it will be in indeterminate order.
	Values() []Value
}

type Cache[Key comparable, Value any] struct {
	data map[Key]*cacheEntry[Key, Value]
	keys []Key

	maxSize     int
	withMaxSize bool

	loader     LoaderFunc[Key, Value]
	withLoader bool

	keyedMutex KeyedMutex[Key]

	onExpired        func(Key, Value)
	expireAfterWrite time.Duration

	mu sync.RWMutex

	rootCtx  context.Context
	closeCtx context.CancelFunc

	ticker *ticker
}

func NewCache[Key comparable, Value any](
	options ...Option[Key, Value],
) ICache[Key, Value] {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Cache[Key, Value]{
		data:     make(map[Key]*cacheEntry[Key, Value]),
		rootCtx:  ctx,
		closeCtx: cancel,
		ticker:   newTicker(ctx, time.Second*5),
	}
	for _, opt := range options {
		opt(c)
	}
	c.ticker.start(c.cleanup)
	return c
}

func (c *Cache[Key, Value]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[Key]*cacheEntry[Key, Value])
}

func (c *Cache[Key, Value]) Close() {
	c.closeCtx()
}

func (c *Cache[Key, Value]) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

func (c *Cache[Key, Value]) ForEach(fn func(Key, Value)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for key, entry := range c.data {
		fn(key, entry.value)
	}
}

func (c *Cache[Key, Value]) Get(key Key) (Value, bool, error) {
	unlock := c.keyedMutex.lock(key)

	entry, found := c.getSafe(key)
	if found {
		unlock()
		return entry.value, true, nil
	} else {
		defer unlock()
	}

	value, ok, err := c.load(key)

	if ok {
		c.Put(key, value)
	}

	return value, ok, err
}

func (c *Cache[Key, Value]) Has(key Key) bool {
	_, found := c.getSafe(key)
	return found
}

func (c *Cache[Key, Value]) Keys() []Key {
	if c.withMaxSize {
		return c.keys
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return maps.Keys(c.data)
}

func (c *Cache[Key, Value]) Put(key Key, value Value) {
	entry := c.newEntry(key, value)

	if c.withMaxSize && len(c.keys) == c.maxSize {
		c.mu.Lock()
		firstKey := c.keys[0]

		delete(c.data, firstKey)
		c.keys = c.keys[1:]
		c.keys = append(c.keys, entry.key)
		c.mu.Unlock()
	}

	c.putSafe(entry)
}

func (c *Cache[Key, Value]) Reload(key Key) (Value, bool, error) {
	if !c.withLoader {
		var val Value
		return val, false, fmt.Errorf("Cache doesn't contain a loader function")
	}

	c.Remove(key)
	return c.Get(key)
}

func (c *Cache[Key, Value]) Remove(key Key) {
	c.removeSafe(key)
}

func (c *Cache[Key, Value]) ToMap() map[Key]Value {
	c.mu.RLock()
	defer c.mu.RUnlock()
	m := make(map[Key]Value)
	for key, entry := range c.data {
		m[key] = entry.value
	}
	return m
}

func (c *Cache[Key, Value]) Values() []Value {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entries := maps.Values(c.data)
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
	return func(c *Cache[Key, Value]) {
		c.expireAfterWrite = expireAfterWrite
	}
}

func WithLoader[Key comparable, Value any](
	loader LoaderFunc[Key, Value],
) Option[Key, Value] {
	return func(c *Cache[Key, Value]) {
		c.loader = loader
		c.withLoader = loader != nil
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
		c.maxSize = maxSize
		c.withMaxSize = maxSize > 0
		c.keys = make([]Key, maxSize)
	}
}

/*
 * @Internal
 */

type cacheEntry[Key comparable, Value any] struct {
	key     Key
	value   Value
	updated time.Time
}

func (c *Cache[Key, Value]) cleanup() {
	keys := c.Keys()

	for _, key := range keys {
		entry, found := c.getSafe(key)
		if found && entry.isExpired(c.expireAfterWrite) {
			c.removeSafe(key)
			if c.onExpired != nil {
				c.onExpired(entry.key, entry.value)
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

}

func (c *Cache[Key, Value]) load(key Key) (Value, bool, error) {
	if !c.withLoader {
		var val Value
		return val, false, nil
	}

	value, err := c.loader(key)

	return value, err == nil, err
}

func (c *Cache[Key, Value]) getSafe(key Key) (*cacheEntry[Key, Value], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, found := c.data[key]
	return entry, found
}

func (c *Cache[Key, Value]) putSafe(entry *cacheEntry[Key, Value]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[entry.key] = entry
}

func (c *Cache[Key, Value]) removeSafe(key Key) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *Cache[Key, Value]) newEntry(key Key, value Value) *cacheEntry[Key, Value] {
	updated := time.Now()
	return &cacheEntry[Key, Value]{key, value, updated}
}

func (e *cacheEntry[Key, Value]) isExpired(expireAfterWrite time.Duration) bool {
	if expireAfterWrite > 0 {
		now := time.Now()
		return now.Sub(e.updated) > expireAfterWrite
	}
	return false
}
