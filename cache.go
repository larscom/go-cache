package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/smallnest/safemap"
)

type CacheEntry[Key comparable, Value any] struct {
	Key   Key
	Value Value
}

type Option[Key comparable, Value any] func(c *cache[Key, Value])

type LoaderFunc[Key comparable, Value any] func(Key) (Value, error)

type Cache[Key comparable, Value any] interface {
	// GetIfPresent Get item from cache (if present) without loader
	GetIfPresent(Key) (Value, bool)
	// Has See if cache contains a key
	Has(Key) bool
	// IsEmpty See if there are any entries in the cache
	IsEmpty() bool
	// Get Retrieve item with the loader function (if configured)
	// (thread safe) it is only ever called once, even if it's called from multiple goroutines.
	Get(Key) (Value, error)
	// Put Add a new item to the cache
	Put(Key, Value)
	// Count Total amount of entries
	Count() int
	// Channel Returns a buffered channel, it can be used to range over all entries
	Channel() <-chan CacheEntry[Key, Value]
	// Refresh item in cache
	Refresh(Key) (Value, error)
	// Remove an item from the cache
	Remove(Key)
	// ToMap Get the map with the key/value pairs, it will be in indeterminate order.
	ToMap() map[Key]Value
	// ForEach Loop over each entry in the cache
	ForEach(func(Key, Value))
	// Values Get all values, it will be in indeterminate order.
	Values() []Value
	// Keys Get all keys, it will be in indeterminate order.
	Keys() []Key
	// Clear Clears the whole cache
	Clear()
	// Close Cleanup any timers
	Close()
}

type cache[Key comparable, Value any] struct {
	entries *safemap.SafeMap[Key, *cacheEntry[Key, Value]]

	loaderMu keyedMutex[Key]
	loader   LoaderFunc[Key, Value]

	expireAfterWrite time.Duration
	onExpired        func(Key, Value)

	cancel context.CancelFunc
	ticker *ticker
}

func New[Key comparable, Value any](
	options ...Option[Key, Value],
) Cache[Key, Value] {
	c := &cache[Key, Value]{
		entries: safemap.New[Key, *cacheEntry[Key, Value]](),
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func (c *cache[Key, Value]) Clear() {
	c.entries.Clear()
}

func (c *cache[Key, Value]) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *cache[Key, Value]) Count() int {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	return e.Count()
}

func (c *cache[Key, Value]) Channel() <-chan CacheEntry[Key, Value] {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	itemchn := make(chan CacheEntry[Key, Value], e.Count())

	go func() {
		defer close(itemchn)

		for item := range e.IterBuffered() {
			itemchn <- CacheEntry[Key, Value]{
				Key:   item.Key,
				Value: item.Val.value,
			}
		}
	}()

	return itemchn
}

func (c *cache[Key, Value]) ForEach(fn func(Key, Value)) {
	for item := range c.Channel() {
		fn(item.Key, item.Value)
	}
}

func (c *cache[Key, Value]) Get(key Key) (Value, error) {
	unlock := c.loaderMu.lock(key)

	entry, found := c.entries.Get(key)
	if found && !entry.isExpired() {
		unlock()
		return entry.value, nil
	}

	value, err := c.load(key)

	defer unlock()

	if err == nil {
		c.Put(key, value)
	}

	return value, err
}

func (c *cache[Key, Value]) GetIfPresent(key Key) (Value, bool) {
	entry, found := c.entries.Get(key)

	if found && !entry.isExpired() {
		return entry.value, true
	}

	var value Value
	return value, false
}

func (c *cache[Key, Value]) Refresh(key Key) (Value, error) {
	unlock := c.loaderMu.lock(key)
	defer unlock()
	value, err := c.load(key)

	if err == nil {
		c.Put(key, value)
	}

	return value, err
}

func (c *cache[Key, Value]) Has(key Key) bool {
	_, found := c.GetIfPresent(key)
	return found
}

func (c *cache[Key, Value]) IsEmpty() bool {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	return e.IsEmpty()
}

func (c *cache[Key, Value]) Keys() []Key {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	return e.Keys()
}

func (c *cache[Key, Value]) Put(key Key, value Value) {
	c.entries.Set(key, c.newEntry(key, value))
}

func (c *cache[Key, Value]) Remove(key Key) {
	c.entries.Remove(key)
}

func (c *cache[Key, Value]) ToMap() map[Key]Value {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	m := make(map[Key]Value)
	for item := range e.IterBuffered() {
		m[item.Key] = item.Val.value
	}

	return m
}

func (c *cache[Key, Value]) Values() []Value {
	e := c.entries
	if c.isTimerEnabled() {
		e = c.getActiveEntries()
	}

	values := make([]Value, 0)
	for item := range e.IterBuffered() {
		values = append(values, item.Val.value)
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
	return func(c *cache[Key, Value]) {
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
	return func(c *cache[Key, Value]) {
		c.loader = loader
	}
}

func WithOnExpired[Key comparable, Value any](
	onExpired func(Key, Value),
) Option[Key, Value] {
	return func(c *cache[Key, Value]) {
		c.onExpired = onExpired
	}
}

func (c *cache[Key, Value]) newEntry(key Key, value Value) *cacheEntry[Key, Value] {
	var expiration time.Time

	if c.isTimerEnabled() {
		expiration = time.Now().Add(c.expireAfterWrite)
	}

	return &cacheEntry[Key, Value]{key, value, expiration}
}

func (c *cache[Key, Value]) getActiveEntries() *safemap.SafeMap[Key, *cacheEntry[Key, Value]] {
	m := safemap.New[Key, *cacheEntry[Key, Value]]()
	for item := range c.entries.IterBuffered() {
		if !item.Val.isExpired() {
			m.Set(item.Key, item.Val)
		}
	}
	return m
}

func (c *cache[Key, Value]) cleanup() {
	keys := c.entries.Keys()

	for _, key := range keys {
		entry, found := c.entries.Get(key)
		if found && entry.isExpired() {
			c.Remove(key)
			if c.onExpired != nil {
				c.onExpired(entry.key, entry.value)
			}
		}
	}
}

func (c *cache[Key, Value]) load(key Key) (Value, error) {
	if c.loader == nil {
		var val Value
		return val, fmt.Errorf("you must configure a loader, use GetIfPresent instead")
	}

	value, err := c.loader(key)

	return value, err
}

func (c *cache[Key, Value]) isTimerEnabled() bool {
	return c.expireAfterWrite > 0
}
