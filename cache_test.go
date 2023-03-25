package cache

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func createCache(options ...Option[int, int]) ICache[int, int] {
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
	t.Run("get zero", func(t *testing.T) {
		cache := createCache()
		val, found, err := cache.Get(1)
		assert.Zero(t, val)
		assert.False(t, found)
		assert.NoError(t, err)
	})
	t.Run("get value", func(t *testing.T) {
		cache := createCache()
		const key = 1

		cache.Put(key, 100)
		val, found, err := cache.Get(key)

		assert.Equal(t, 100, val)
		assert.True(t, found)
		assert.NoError(t, err)
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
		val, found, err := cache.Get(key)

		assert.Equal(t, 1, cache.Count())
		assert.Equal(t, 200, val)
		assert.True(t, found)
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
			cachedVal, _, _ := cache.Get(key)
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

		v, found, err := c.Get(key)

		assert.Equal(t, 100, v)
		assert.True(t, found)
		assert.NoError(t, err)
	})
	t.Run("get is expired", func(t *testing.T) {
		ttl := time.Millisecond * 10
		c := createCache(WithExpireAfterWrite[int, int](ttl))

		const key = 1
		c.Put(key, 100)

		<-time.After(time.Millisecond * 15)

		v, found, err := c.Get(key)

		assert.Zero(t, v)
		assert.False(t, found)
		assert.NoError(t, err)
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

		value, ok, err := c.Get(1)

		assert.Equal(t, 12345, value)
		assert.True(t, ok)
		assert.NoError(t, err)
	})
	t.Run("get from cache", func(t *testing.T) {
		c := createCache(WithLoader(func(key int) (int, error) {
			return 12345, nil
		}))

		const key = 1
		c.Put(key, 100)

		value, ok, err := c.Get(key)

		assert.Equal(t, 100, value)
		assert.True(t, ok)
		assert.NoError(t, err)
	})
	t.Run("get loader error", func(t *testing.T) {
		c := createCache(WithLoader(func(key int) (int, error) {
			return 0, fmt.Errorf("ERROR")
		}))

		value, found, err := c.Get(1)

		assert.False(t, found)
		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("ERROR"), err)
		assert.Zero(t, value)
	})
	t.Run("call loader once per key", func(t *testing.T) {
		var counter int32
		cache := createCache(WithLoader(func(key int) (int, error) {
			atomic.AddInt32(&counter, 1)
			time.Sleep(time.Millisecond * 10)
			return 12345, nil
		}))

		go func() {
			cache.Get(1)
		}()
		go func() {
			cache.Get(1)
		}()

		go func() {
			cache.Get(2)
		}()
		go func() {
			cache.Get(2)
		}()

		<-time.After(time.Millisecond * 15)

		r := atomic.LoadInt32(&counter)

		assert.Equal(t, 2, int(r))
	})

}

func Test_WithMaxSize(t *testing.T) {
	t.Run("put", func(t *testing.T) {
		c := createCache(WithMaxSize[int, int](3))
		c.Put(1, 100)
		c.Put(2, 200)
		c.Put(3, 300)
		c.Put(4, 400)

		assert.Equal(t, 3, c.Count())
		assert.ElementsMatch(t, []int{2, 3, 4}, c.Keys())
	})

	t.Run("remove", func(t *testing.T) {
		c := createCache(WithMaxSize[int, int](3))
		c.Put(1, 100)
		c.Put(2, 200)
		c.Put(3, 300)

		c.Remove(2)

		c.Put(2, 202)
		c.Put(4, 400)

		vals := c.Values()
		keys := c.Keys()

		assert.Len(t, vals, 3)
		assert.Len(t, keys, 3)
		assert.Equal(t, []int{3, 2, 4}, keys)

		val, _, _ := c.Get(3)
		assert.Equal(t, 300, val)

		val, _, _ = c.Get(2)
		assert.Equal(t, 202, val)

		val, _, _ = c.Get(4)
		assert.Equal(t, 400, val)
	})

}
