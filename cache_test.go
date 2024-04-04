package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	const defaultTTL = time.Millisecond * 30

	TestGet := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)

		value, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, 100, value)

		value, found = cache.Get(2)
		assert.False(t, found)
		assert.Zero(t, value)
	}
	t.Run("TestGet", TestGet)

	TestGetWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)

		value, found := cache.Get(key)
		assert.True(t, found)
		assert.Equal(t, 100, value)

		<-time.After(defaultTTL + 5)

		value, found = cache.Get(key)
		assert.False(t, found)
		assert.Zero(t, value)
	}
	t.Run("TestGetWithExpireAfterWrite", TestGetWithExpireAfterWrite)

	TestPut := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		cache.Put(key, 200)

		value, found := cache.Get(key)

		assert.True(t, found)
		assert.Equal(t, 200, value)
		assert.Equal(t, 1, cache.Count())
	}
	t.Run("TestPut", TestPut)

	TestPutWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)

		assert.False(t, cache.Has(key))
	}
	t.Run("TestPutWithExpireAfterWrite", TestPutWithExpireAfterWrite)

	TestHas := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		const key = 1
		cache.Put(1, 100)

		assert.True(t, cache.Has(key))
		assert.False(t, cache.Has(2))
	}
	t.Run("TestHas", TestHas)

	TestHasWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)

		assert.False(t, cache.Has(key))
	}
	t.Run("TestHasWithExpireAfterWrite", TestHasWithExpireAfterWrite)

	TestIsEmpty := func(t *testing.T) {
		filledCache := NewCache[int, int]()
		defer filledCache.Close()

		filledCache.Put(1, 100)
		assert.False(t, filledCache.IsEmpty())

		emptyCache := NewCache[int, int]()
		defer emptyCache.Close()

		assert.True(t, emptyCache.IsEmpty())
	}
	t.Run("TestIsEmpty", TestIsEmpty)

	TestIsEmptyWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		cache.Put(1, 100)
		assert.False(t, cache.IsEmpty())

		<-time.After(defaultTTL + 5)
		assert.True(t, cache.IsEmpty())
	}
	t.Run("TestIsEmptyWithExpireAfterWrite", TestIsEmptyWithExpireAfterWrite)

	TestCount := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		for i := 0; i < 5; i++ {
			cache.Put(i, i)
		}

		assert.Equal(t, 5, cache.Count())
	}
	t.Run("TestCount", TestCount)

	TestCountWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		for i := 0; i < 5; i++ {
			cache.Put(i, i)
		}
		assert.Equal(t, 5, cache.Count())

		<-time.After(defaultTTL + 5)

		assert.Zero(t, cache.Count())
	}
	t.Run("TestCountWithExpireAfterWrite", TestCountWithExpireAfterWrite)

	TestForEach := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		const length = 5
		for i := 0; i < length; i++ {
			cache.Put(i, i)
		}
		assert.Equal(t, length, cache.Count())

		wg := new(sync.WaitGroup)
		wg.Add(length)

		cache.ForEach(func(key, value int) {
			defer wg.Done()
			cached, _ := cache.Get(key)
			assert.Equal(t, cached, value)
		})

		wg.Wait()
	}
	t.Run("TestForEach", TestForEach)

	TestForEachWithExpireAfterWrite := func(t *testing.T) {
		cache := NewCache(WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		cache.Put(1, 100)
		<-time.After(defaultTTL + 5)

		cache.Put(2, 200)

		wg := new(sync.WaitGroup)
		wg.Add(1)

		cache.ForEach(func(key, value int) {
			defer wg.Done()
			cached, _ := cache.Get(key)
			assert.Equal(t, cached, value)
			assert.Equal(t, 2, key)
			assert.Equal(t, 200, value)
		})

		wg.Wait()
	}
	t.Run("TestForEachWithExpireAfterWrite", TestForEachWithExpireAfterWrite)

	TestDelete := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		cache.Put(1, 100)
		cache.Put(2, 200)
		assert.Equal(t, 2, cache.Count())

		cache.Delete(1)
		assert.Equal(t, 1, cache.Count())

		value, found := cache.Get(2)
		assert.True(t, found)
		assert.Equal(t, 200, value)
	}
	t.Run("TestDelete", TestDelete)

	TestClear := func(t *testing.T) {
		cache := NewCache[int, int]()
		defer cache.Close()

		const length = 5
		for i := 0; i < length; i++ {
			cache.Put(i, i)
		}
		assert.Equal(t, length, cache.Count())

		cache.Clear()

		assert.Zero(t, cache.Count())
	}
	t.Run("TestClear", TestClear)

	TestCloseShouldClear := func(t *testing.T) {
		cache := NewCache[int, int]()

		const length = 5
		for i := 0; i < length; i++ {
			cache.Put(i, i)
		}
		assert.Equal(t, length, cache.Count())

		cache.Close()

		assert.Zero(t, cache.Count())
	}
	t.Run("TestCloseShouldClear", TestCloseShouldClear)
}
