package cache

import (
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
)

// Function that gets executed by the 'Load' and 'Reload' function
type LoaderFunc[K comparable, V any] func(key K) (V, error)

type LoadingCache[K comparable, V any] interface {
	// Loads an item into cache using the provided LoaderFunc and returns the value.
	//
	// If the item is already cached, it'll return that value instead.
	//
	// Whenever the LoaderFunc returns an error, the value does NOT get saved.
	//
	// This function is thread-safe and the LoaderFunc is called only once in a concurrent environment.
	Load(key K) (V, error)

	// Reloads an item into cache using the provided LoaderFunc and returns the new value.
	//
	// Whenever the LoaderFunc returns an error, the value does NOT get saved (old value remains in cache)
	Reload(key K) (V, error)

	// Embed Cache
	Cache[K, V]
}

func NewLoadingCache[K comparable, V any](
	loaderFunc LoaderFunc[K, V],
	options ...Option[K, V],
) LoadingCache[K, V] {
	data := csmap.Create[K, *entry[K, V]]()
	cleaner := newCacheCleaner(data, time.Second*5)
	opts := append(options, withLoaderFunc(loaderFunc))
	return newCache(data, cleaner, opts...)
}

func (c *cache[K, V]) Load(key K) (V, error) {
	unlock := c.mu.lock(key)
	defer unlock()

	cached, found := c.get(key)
	if found {
		return cached, nil
	}

	value, err := c.loaderFunc(key)
	if err == nil {
		c.data.Store(key, c.newEntry(key, value))
	}

	return value, err
}

func (c *cache[K, V]) Reload(key K) (V, error) {
	unlock := c.mu.lock(key)
	defer unlock()

	value, err := c.loaderFunc(key)
	if err == nil {
		c.data.Store(key, c.newEntry(key, value))
	}

	return value, err
}

// Function that can be used inside a testing environment
func NoopLoaderFunc[K comparable, V any](key K) (V, error) {
	var empty V
	return empty, nil
}

func withLoaderFunc[K comparable, V any](
	loaderFunc LoaderFunc[K, V],
) Option[K, V] {
	return func(c *cache[K, V]) {
		c.loaderFunc = loaderFunc
	}
}
