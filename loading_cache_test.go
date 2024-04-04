package cache

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
}

func TestLoadingCache(t *testing.T) {
	var (
		defaultLoaderFunc = func(key int) (int, error) {
			return key * 2, nil
		}
		defaultLoaderFuncError = func(key int) (int, error) {
			return 0, fmt.Errorf("got error on key: %d", key)
		}
		defaultTTL = time.Millisecond * 30
	)

	TestLoad := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc)
		defer cache.Close()

		value, err := cache.Load(1)

		assert.NoError(t, err)
		assert.Equal(t, 2, value)
	}
	t.Run("TestLoad", TestLoad)

	TestLoadError := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFuncError)
		defer cache.Close()

		_, err := cache.Load(1)

		assert.EqualError(t, err, "got error on key: 1")
		assert.Zero(t, cache.Count())
	}
	t.Run("TestLoadError", TestLoadError)

	TestLoadWithExpireAfterWrite := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		value, err := cache.Load(key)
		assert.NoError(t, err)
		assert.Equal(t, 2, value)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)
		assert.False(t, cache.Has(key))
	}
	t.Run("TestLoadWithExpireAfterWrite", TestLoadWithExpireAfterWrite)

	TestLoadCalledOnceInConcurrentEnvironment := func(t *testing.T) {
		counter := int64(0)

		loaderFunc := func(key int) (int, error) {
			atomic.AddInt64(&counter, 1)
			time.Sleep(time.Millisecond * 20)
			return key, nil
		}
		cache := NewLoadingCache(loaderFunc)
		defer cache.Close()

		wg := new(sync.WaitGroup)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				start := time.Now()
				value, err := cache.Load(100)
				end := time.Since(start)
				if err != nil {
					t.Error(err)
				}
				slog.Debug("Done", "index", index, "time_taken", end, "value", value)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int64(1), atomic.LoadInt64(&counter))
		assert.Equal(t, 1, cache.Count())
	}
	t.Run("TestLoadCalledOnceInConcurrentEnvironment", TestLoadCalledOnceInConcurrentEnvironment)

	TestLoadCalledTwiceInConcurrentEnvironment := func(t *testing.T) {
		counter := int64(0)

		loaderFunc := func(key int) (int, error) {
			atomic.AddInt64(&counter, 1)
			time.Sleep(time.Millisecond * 20)
			return key, nil
		}
		cache := NewLoadingCache(loaderFunc)
		defer cache.Close()

		wg := new(sync.WaitGroup)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				start := time.Now()
				value, err := cache.Load(100)
				end := time.Since(start)
				if err != nil {
					t.Error(err)
				}
				slog.Debug("Done", "index", index, "time_taken", end, "value", value)
			}(i)
		}

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				start := time.Now()
				value, err := cache.Load(200)
				end := time.Since(start)
				if err != nil {
					t.Error(err)
				}
				slog.Debug("Done", "index", index, "time_taken", end, "value", value)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int64(2), atomic.LoadInt64(&counter))
		assert.Equal(t, 2, cache.Count())
	}
	t.Run("TestLoadCalledTwiceInConcurrentEnvironment", TestLoadCalledTwiceInConcurrentEnvironment)

	TestReload := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc)
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.Equal(t, 1, cache.Count())

		value, err := cache.Reload(key)

		assert.NoError(t, err)
		assert.Equal(t, 2, value)
		assert.Equal(t, 1, cache.Count())
	}
	t.Run("TestReload", TestReload)

	TestReloadError := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFuncError)
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.Equal(t, 1, cache.Count())

		_, err := cache.Reload(key)
		assert.EqualError(t, err, "got error on key: 1")

		value, _ := cache.Get(key)
		assert.Equal(t, 100, value)
		assert.Equal(t, 1, cache.Count())
	}
	t.Run("TestReloadError", TestReloadError)

	TestReloadWithExpireAfterWrite := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		value, err := cache.Reload(key)
		assert.NoError(t, err)
		assert.Equal(t, 2, value)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)
		assert.False(t, cache.Has(key))
	}
	t.Run("TestReloadWithExpireAfterWrite", TestReloadWithExpireAfterWrite)

	TestGet := func(t *testing.T) {
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
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
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
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
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
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
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)

		assert.False(t, cache.Has(key))
	}
	t.Run("TestPutWithExpireAfterWrite", TestPutWithExpireAfterWrite)

	TestHas := func(t *testing.T) {
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
		defer cache.Close()

		const key = 1
		cache.Put(1, 100)

		assert.True(t, cache.Has(key))
		assert.False(t, cache.Has(2))
	}
	t.Run("TestHas", TestHas)

	TestHasWithExpireAfterWrite := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		const key = 1

		cache.Put(key, 100)
		assert.True(t, cache.Has(key))

		<-time.After(defaultTTL + 5)

		assert.False(t, cache.Has(key))
	}
	t.Run("TestHasWithExpireAfterWrite", TestHasWithExpireAfterWrite)

	TestIsEmpty := func(t *testing.T) {
		filledCache := NewLoadingCache[int, int](defaultLoaderFunc)
		defer filledCache.Close()

		filledCache.Put(1, 100)
		assert.False(t, filledCache.IsEmpty())

		emptyCache := NewLoadingCache[int, int](defaultLoaderFunc)
		defer emptyCache.Close()

		assert.True(t, emptyCache.IsEmpty())
	}
	t.Run("TestIsEmpty", TestIsEmpty)

	TestIsEmptyWithExpireAfterWrite := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
		defer cache.Close()

		cache.Put(1, 100)
		assert.False(t, cache.IsEmpty())

		<-time.After(defaultTTL + 5)
		assert.True(t, cache.IsEmpty())
	}
	t.Run("TestIsEmptyWithExpireAfterWrite", TestIsEmptyWithExpireAfterWrite)

	TestCount := func(t *testing.T) {
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
		defer cache.Close()

		for i := 0; i < 5; i++ {
			cache.Put(i, i)
		}

		assert.Equal(t, 5, cache.Count())
	}
	t.Run("TestCount", TestCount)

	TestCountWithExpireAfterWrite := func(t *testing.T) {
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
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
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
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
		cache := NewLoadingCache(defaultLoaderFunc, WithExpireAfterWrite[int, int](defaultTTL))
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
		cache := NewLoadingCache[int, int](defaultLoaderFunc)
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
		cache := NewLoadingCache(defaultLoaderFunc)
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
		cache := NewLoadingCache(defaultLoaderFunc)

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
