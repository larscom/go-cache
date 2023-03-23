package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createCache(options ...Option[int, int]) ICache[int, int] {
	return NewCache(options...)
}

func Test_Default(t *testing.T) {
	t.Run("should clear the whole map", func(t *testing.T) {
		cache := createCache()

		cache.Put(1, 1)
		cache.Put(2, 2)
		cache.Put(3, 3)

		assert.Equal(t, 3, cache.Count())

		cache.Clear()

		assert.Zero(t, cache.Count())
	})

	t.Run("should not panic when calling close", func(t *testing.T) {
		cache := createCache()
		assert.NotPanics(t, func() {
			cache.Close()
		})
	})

	t.Run("should count total size of the cache", func(t *testing.T) {
		cache := createCache()

		assert.Zero(t, cache.Count())

		cache.Put(1, 1)
		cache.Put(2, 2)
		cache.Put(3, 3)

		assert.Equal(t, 3, cache.Count())
	})

	t.Run("should be able to loop over the cache entries with the for each function", func(t *testing.T) {
		cache := createCache()
		keys := []int{1, 2, 3}

		for i := 0; i < len(keys); i++ {
			cache.Put(i, keys[i])
		}
		assert.Equal(t, len(keys), cache.Count())

		cache.ForEach(func(key, value int) {
			assert.Equal(t, key+1, value)
		})
	})

	t.Run("should be able to call get when the key doesnt exist", func(t *testing.T) {
		cache := createCache()
		const key = 1

		val, found, err := cache.Get(key)

		assert.Zero(t, val)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("should get value from cache", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		val, found, err := cache.Get(key)

		assert.Equal(t, 100, val)
		assert.True(t, found)
		assert.NoError(t, err)
	})

	t.Run("has function should return false when cache doesnt contain key", func(t *testing.T) {
		cache := createCache()
		const key = 1
		found := cache.Has(key)

		assert.False(t, found)
	})

	t.Run("has function should return true when cache contains key", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		found := cache.Has(key)

		assert.True(t, found)
	})

	t.Run("should be able to put a new value into the cache", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		cache.Put(key, 200)
		val, found, err := cache.Get(key)

		assert.Equal(t, 1, cache.Count())
		assert.Equal(t, 200, val)
		assert.True(t, found)
		assert.NoError(t, err)
	})

	t.Run("should be able to call reload without loader function", func(t *testing.T) {
		cache := createCache()

		val, ok, err := cache.Reload(1)

		assert.Zero(t, val)
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("should be able to call remove on a key that doesnt exist", func(t *testing.T) {
		cache := createCache()
		assert.NotPanics(t, func() {
			cache.Remove(1)
		})
	})

	t.Run("should be able to remove a key/value from cache", func(t *testing.T) {
		cache := createCache()
		const key = 1

		{
			cache.Put(key, 100)
			found := cache.Has(key)
			assert.True(t, found)
		}

		cache.Remove(key)

		found := cache.Has(key)
		assert.False(t, found)
	})

	t.Run("should get back a map implementation", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 200)
		cache.Put(3, 300)

		m := cache.ToMap()

		assert.Equal(t, cache.Count(), len(m))

		for key, value := range m {
			cachedVal, _, _ := cache.Get(key)
			assert.Equal(t, cachedVal, value)
		}
	})

	t.Run("should be able to get all keys from cache", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 100)
		cache.Put(3, 100)

		assert.ElementsMatch(t, []int{1, 2, 3}, cache.Keys())
	})

	t.Run("should be able to get all values from cache", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 200)
		cache.Put(3, 300)

		assert.ElementsMatch(t, []int{100, 200, 300}, cache.Values())
	})
}

func Test_With_Loader(t *testing.T) {

	t.Run("should get value from loader when not in cache", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		value, found, err := cache.Get(1)

		assert.Equal(t, 12345, value)
		assert.True(t, found)
		assert.NoError(t, err)
	})

	t.Run("should get value from cache when present", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		const key = 1

		cache.Put(key, 100)

		value, found, err := cache.Get(key)

		assert.Equal(t, 100, value)
		assert.True(t, found)
		assert.NoError(t, err)
	})

	t.Run("should get error from loader", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 0, fmt.Errorf("ERROR")
		}))

		value, found, err := cache.Get(1)

		assert.False(t, found)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("ERROR"), err)
		assert.Zero(t, value)
	})

	t.Run("should reload value with loader", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		const key = 1

		cache.Put(key, 100)

		value, _, _ := cache.Get(key)
		assert.Equal(t, 100, value)

		value, ok, err := cache.Reload(key)
		assert.True(t, ok)
		assert.NoError(t, err)
		assert.Equal(t, 12345, value)

		value, _, _ = cache.Get(key)

		assert.Equal(t, 12345, value)
	})

	t.Run("should not update cache when loader returns an error", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 0, fmt.Errorf("ERROR")
		}))

		const key = 1

		cache.Put(key, 100)

		value, _, _ := cache.Get(key)
		assert.Equal(t, 100, value)

		value, ok, err := cache.Reload(key)
		assert.Zero(t, value)
		assert.False(t, ok)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("ERROR"), err)

		value, ok, _ = cache.Get(key)

		assert.True(t, ok)
		assert.Equal(t, 100, value)
	})
}

func Test_With_ExpireAfterWrite(t *testing.T) {

	t.Run("should get value before expiring", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](50 * time.Millisecond)
		cache := createCache(ttl)

		const key = 1

		cache.Put(key, 100)

		<-time.After(25 * time.Millisecond)

		value, found, err := cache.Get(key)

		assert.Equal(t, 100, value)
		assert.True(t, found)
		assert.NoError(t, err)
	})

	t.Run("should not get value because it is epired", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](25 * time.Millisecond)
		cache := createCache(ttl)

		const key = 1

		cache.Put(key, 100)

		<-time.After(35 * time.Millisecond)

		value, found, err := cache.Get(key)

		assert.Zero(t, value)
		assert.False(t, found)
		assert.NoError(t, err)
	})

	t.Run("expire time should count since last item was added", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](25 * time.Millisecond)
		cache := createCache(ttl)

		const key = 1

		n := 25
		for i := 0; i <= n; i++ {
			time.Sleep(time.Millisecond * 5)
			cache.Put(key, i)
		}

		<-time.After(time.Millisecond * 15)

		actual, _, _ := cache.Get(key)

		assert.Equal(t, n, actual)
	})

	t.Run("should clear the whole cache and cancel current entries", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](50 * time.Millisecond)
		cache := createCache(ttl)

		n := 5
		for i := 0; i < n; i++ {
			cache.Put(i, i)
		}

		<-time.After(15 * time.Millisecond)

		assert.Equal(t, n, cache.Count())

		cache.Clear()

		assert.Zero(t, cache.Count())

		cache.Put(n, 100)

		<-time.After(40 * time.Millisecond)

		actual, _, _ := cache.Get(n)

		assert.Equal(t, 100, actual)
	})

	t.Run("should remove from cache and cancel the current entry", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](50 * time.Millisecond)
		cache := createCache(ttl)

		cache.Put(1, 1)

		assert.Equal(t, 1, cache.Count())

		cache.Remove(1)

		<-time.After(40 * time.Millisecond)

		assert.Zero(t, cache.Count())

		cache.Put(1, 100)

		<-time.After(40 * time.Millisecond)

		actual, _, _ := cache.Get(1)

		assert.Equal(t, 100, actual)
	})

	t.Run("should close context and stop current entries from expiring", func(t *testing.T) {
		ttl := WithExpireAfterWrite[int, int](50 * time.Millisecond)
		cache := createCache(ttl)

		n := 50
		for i := 0; i < n; i++ {
			cache.Put(i, i)
		}

		assert.Equal(t, n, cache.Count())

		cache.Close()

		<-time.After(100 * time.Millisecond)

		assert.Equal(t, n, cache.Count())
	})
}

func Test_With_OnExpired(t *testing.T) {

	t.Run("should call function when expired and receive the key,value", func(t *testing.T) {

		var wg sync.WaitGroup
		wg.Add(1)

		onExpired := WithOnExpired(func(key int, value int) {
			defer wg.Done()
			assert.Equal(t, 1, key)
			assert.Equal(t, 100, value)
		})

		ttl := WithExpireAfterWrite[int, int](25 * time.Millisecond)

		cache := createCache(ttl, onExpired)
		cache.Put(1, 100)

		<-time.After(35 * time.Millisecond)

		wg.Wait()
	})
}

func Test_With_MaxSize(t *testing.T) {

	t.Run("should remove the first key if going above max size", func(t *testing.T) {

		cache := createCache(WithMaxSize[int, int](3))
		cache.Put(1, 100)
		cache.Put(2, 200)
		cache.Put(3, 300)
		cache.Put(4, 400)

		assert.Equal(t, 3, cache.Count())
		assert.ElementsMatch(t, []int{2, 3, 4}, cache.Keys())
	})
}
