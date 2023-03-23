package cache

import (
	"context"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type Option[Key comparable, Value any] func(c *Cache[Key, Value])

type LoaderFunc[Key comparable, Value any] func(Key) (Value, error)

type ICache[Key comparable, Value any] interface {
	Clear()
	Close()
	Count() int
	ForEach(func(Key, Value))
	Get(Key) (Value, bool, error)
	Has(Key) bool
	Keys() []Key
	Put(Key, Value)
	Reload(Key) (Value, bool, error)
	Remove(Key)
	ToMap() map[Key]Value
	Values() []Value
}

type Cache[Key comparable, Value any] struct {
	data             map[Key]*cacheEntry[Key, Value]
	loader           LoaderFunc[Key, Value]
	onExpired        func(Key, Value)
	expireAfterWrite time.Duration
	withExpiration   bool
	mu               sync.RWMutex

	rootCtx  context.Context
	closeCtx context.CancelFunc
}

func NewCache[Key comparable, Value any](
	options ...Option[Key, Value],
) ICache[Key, Value] {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Cache[Key, Value]{
		data:     make(map[Key]*cacheEntry[Key, Value]),
		rootCtx:  ctx,
		closeCtx: cancel,
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

func (c *Cache[Key, Value]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.withExpiration {
		copy := make(map[Key]*cacheEntry[Key, Value])
		for k, v := range c.data {
			copy[k] = v
		}
		for _, entry := range copy {
			entry.cancel()
		}
	}

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
	entry, found := c.get(key)
	if found {
		return entry.value, true, nil
	}

	value, ok, err := c.load(key)

	if ok {
		c.Put(key, value)
	}

	return value, ok, err
}

func (c *Cache[Key, Value]) Has(key Key) bool {
	_, found := c.get(key)
	return found
}

func (c *Cache[Key, Value]) Keys() []Key {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return maps.Keys(c.data)
}

func (c *Cache[Key, Value]) Put(key Key, value Value) {
	entry := c.newEntry(key, value)

	if c.withExpiration {
		currentEntry, found := c.get(entry.key)
		if found {
			currentEntry.cancel()
		}
		c.expire(entry)
	}

	c.put(entry)
}

func (c *Cache[Key, Value]) Reload(key Key) (Value, bool, error) {
	value, ok, err := c.load(key)

	if ok {
		c.Put(key, value)
	}

	return value, ok, err
}

func (c *Cache[Key, Value]) Remove(key Key) {
	if c.withExpiration {
		entry, found := c.get(key)
		if found {
			entry.cancel()
		}
	}
	c.remove(key)
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
		c.withExpiration = expireAfterWrite > 0
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

/*
 * @Internal
 */

type cacheEntry[Key comparable, Value any] struct {
	key    Key
	value  Value
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Cache[Key, Value]) expire(entry *cacheEntry[Key, Value]) {
	go func() {
		select {
		case <-time.After(c.expireAfterWrite):
			c.remove(entry.key)
			if c.onExpired != nil {
				c.onExpired(entry.key, entry.value)
			}
		case <-entry.ctx.Done():
			return
		}
	}()
}

func (c *Cache[Key, Value]) load(key Key) (Value, bool, error) {
	if c.loader == nil {
		var val Value
		return val, false, nil
	}

	value, err := c.loader(key)
	return value, err == nil, err
}

func (c *Cache[Key, Value]) get(key Key) (*cacheEntry[Key, Value], bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, found := c.data[key]
	return entry, found
}

func (c *Cache[Key, Value]) put(entry *cacheEntry[Key, Value]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[entry.key] = entry
}

func (c *Cache[Key, Value]) remove(key Key) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *Cache[Key, Value]) newEntry(key Key, value Value) *cacheEntry[Key, Value] {
	var ctx context.Context
	var cancel context.CancelFunc

	if c.withExpiration {
		ctx, cancel = context.WithCancel(c.rootCtx)
	}

	return &cacheEntry[Key, Value]{key, value, ctx, cancel}
}
