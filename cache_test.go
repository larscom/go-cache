package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createCache(options ...Option[int, int]) Cache[int, int] {
	return NewCache(options...)
}

func Test_Core(t *testing.T) {
	t.Run("clear", func(t *testing.T) {
		c := createCache()
		c.Put(1, 1)
		c.Put(2, 2)

		assert.Equal(t, 2, c.Count())

		c.Clear()

		assert.Zero(t, c.Count())
	})
	t.Run("close", func(t *testing.T) {
		c := createCache()
		assert.NotPanics(t, func() {
			c.Close()
		})
	})
	t.Run("count", func(t *testing.T) {
		cache := createCache()

		assert.Zero(t, cache.Count())

		cache.Put(1, 1)
		cache.Put(2, 2)

		assert.Equal(t, 2, cache.Count())
	})
	t.Run("forEach", func(t *testing.T) {
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
	t.Run("get should error without loader and value in cache", func(t *testing.T) {
		cache := createCache()
		val, err := cache.Get(1)

		assert.Zero(t, val)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("you must configure a loader, use GetIfPresent instead"), err)
	})
	t.Run("get", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		val, err := cache.Get(key)

		assert.Equal(t, 100, val)
		assert.NoError(t, err)
	})
	t.Run("get if present", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		val, found := cache.GetIfPresent(key)

		assert.Equal(t, 100, val)
		assert.True(t, found)
	})
	t.Run("get if present zero", func(t *testing.T) {
		cache := createCache()
		val, found := cache.GetIfPresent(1)
		assert.Zero(t, val)
		assert.False(t, found)
	})
	t.Run("refresh without loader should error", func(t *testing.T) {
		cache := createCache()

		val, err := cache.Refresh(1)

		assert.Zero(t, val)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("you must configure a loader, use GetIfPresent instead"), err)
	})
	t.Run("has not", func(t *testing.T) {
		cache := createCache()
		has := cache.Has(1)

		assert.False(t, has)
	})
	t.Run("has", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		has := cache.Has(key)

		assert.True(t, has)
	})
	t.Run("keys", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 100)

		assert.ElementsMatch(t, []int{1, 2}, cache.Keys())
	})
	t.Run("put", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		cache.Put(key, 200)
		val, err := cache.Get(key)

		assert.Equal(t, 1, cache.Count())
		assert.Equal(t, 200, val)
		assert.NoError(t, err)
	})
	t.Run("remove", func(t *testing.T) {
		cache := createCache()
		const key = 1
		{
			cache.Put(key, 100)
			has := cache.Has(key)
			assert.True(t, has)
		}
		cache.Remove(key)

		has := cache.Has(key)
		assert.False(t, has)
	})
	t.Run("toMap", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 200)
		cache.Put(3, 300)

		m := cache.ToMap()

		assert.Equal(t, cache.Count(), len(m))

		for key, value := range m {
			cachedVal, _ := cache.Get(key)
			assert.Equal(t, cachedVal, value)
		}
	})
	t.Run("values", func(t *testing.T) {
		cache := createCache()
		cache.Put(1, 100)
		cache.Put(2, 200)

		assert.ElementsMatch(t, []int{100, 200}, cache.Values())
	})
}

func Test_WithExpireAfterWrite(t *testing.T) {
	t.Run("count", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))
		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)
		c.Put(2, 200)

		assert.Equal(t, 1, c.Count())
	})
	t.Run("forEach", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))
		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)
		c.Put(2, 200)

		var wg sync.WaitGroup
		wg.Add(1)

		c.ForEach(func(key, value int) {
			assert.Equal(t, 2, key)
			assert.Equal(t, 200, value)
			wg.Done()
		})

		wg.Wait()
	})
	t.Run("get", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		const key = 1
		c.Put(key, 100)

		v, err := c.Get(key)

		assert.Equal(t, 100, v)
		assert.NoError(t, err)
	})
	t.Run("get is expired", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		const key = 1
		c.Put(key, 100)

		<-time.After(time.Millisecond * 15)

		v, err := c.Get(key)

		assert.Zero(t, v)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("you must configure a loader, use GetIfPresent instead"), err)
	})
	t.Run("get if present", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		const key = 1
		c.Put(key, 100)

		v, found := c.GetIfPresent(key)

		assert.Equal(t, 100, v)
		assert.True(t, found)
	})
	t.Run("get if present is expired", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		const key = 1
		c.Put(key, 100)

		<-time.After(time.Millisecond * 15)

		v, found := c.GetIfPresent(key)

		assert.Zero(t, v)
		assert.False(t, found)
	})
	t.Run("has", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		c.Put(1, 100)
		<-time.After(time.Millisecond * 15)
		has := c.Has(1)
		assert.False(t, has)
	})
	t.Run("keys", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)
		c.Put(2, 200)
		c.Put(3, 300)

		assert.ElementsMatch(t, []int{2, 3}, c.Keys())
	})
	t.Run("toMap", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)
		c.Put(2, 200)

		var wg sync.WaitGroup
		wg.Add(1)

		m := c.ToMap()
		for key, value := range m {
			assert.Equal(t, 2, key)
			assert.Equal(t, 200, value)
			wg.Done()
		}

		wg.Wait()
	})
	t.Run("values", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)
		c.Put(2, 200)
		c.Put(3, 300)

		assert.ElementsMatch(t, []int{200, 300}, c.Values())
	})
}

func Test_Expiration(t *testing.T) {
	t.Run("onExpired", func(t *testing.T) {
		ttl := time.Millisecond * 10
		cleanupInterval := time.Millisecond * 5

		var wg sync.WaitGroup
		wg.Add(1)

		expiredFunc := func(key, value int) {
			assert.Equal(t, 1, key)
			assert.Equal(t, 100, value)
			wg.Done()
		}
		c := createCache(WithExpireAfterWriteCustom[int, int](ttl, cleanupInterval), WithOnExpired(expiredFunc))

		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)

		wg.Wait()
	})
	t.Run("close triggers cleanup", func(t *testing.T) {
		ttl := time.Millisecond * 10
		cleanupInterval := time.Millisecond * 500

		var wg sync.WaitGroup
		wg.Add(1)

		expiredFunc := func(key, value int) {
			assert.Equal(t, 1, key)
			assert.Equal(t, 100, value)
			wg.Done()
		}

		c := createCache(WithExpireAfterWriteCustom[int, int](ttl, cleanupInterval), WithOnExpired(expiredFunc))

		c.Put(1, 100)

		<-time.After(time.Millisecond * 15)

		c.Close()

		wg.Wait()
	})
}

func Test_WithLoader(t *testing.T) {
	t.Run("get from loader", func(t *testing.T) {
		c := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		value, err := c.Get(1)

		assert.Equal(t, 12345, value)
		assert.NoError(t, err)
	})
	t.Run("get from cache", func(t *testing.T) {
		c := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		const key = 1
		c.Put(key, 100)

		value, err := c.Get(key)

		assert.Equal(t, 100, value)
		assert.NoError(t, err)
	})
	t.Run("get loader error", func(t *testing.T) {
		c := createCache(WithLoader(func(key int) (int, error) {
			return 0, fmt.Errorf("ERROR")
		}))

		value, err := c.Get(1)

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("ERROR"), err)
		assert.Zero(t, value)
	})
	t.Run("refresh", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		const key = 1

		cache.Put(key, 100)

		value, _ := cache.Get(key)
		assert.Equal(t, 100, value)

		value, err := cache.Refresh(key)
		assert.NoError(t, err)
		assert.Equal(t, 12345, value)

		value, _ = cache.Get(key)

		assert.Equal(t, 12345, value)
	})
	t.Run("refresh no update on error", func(t *testing.T) {
		cache := createCache(WithLoader(func(key int) (int, error) {
			return 0, fmt.Errorf("ERROR")
		}))

		const key = 1

		cache.Put(key, 100)

		value, _ := cache.Get(key)
		assert.Equal(t, 100, value)

		value, err := cache.Refresh(key)
		assert.Zero(t, value)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("ERROR"), err)

		value, err = cache.Get(key)

		assert.NoError(t, err)
		assert.Equal(t, 100, value)
	})
}
