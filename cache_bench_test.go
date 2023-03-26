package cache

import (
	"math/rand"
	"strings"
	"testing"
)

func Benchmark_Get(b *testing.B) {
	cache := NewCache[int, int]()
	for n := 0; n < b.N; n++ {
		cache.Get(n)
	}
	b.ReportAllocs()
}

func Benchmark_GetPutMultipleConcurrent(b *testing.B) {
	cache := NewCache[string, string]()

	data := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
		"k4": "v4",
		"k5": "v5",
		"k6": "v6",
		"k7": "v7",
		"k8": "v8",
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			for k, v := range data {
				cache.Put(k, v)
				cache.Get(k)
			}
		}
	})
}

func Benchmark_GetConcurrently(b *testing.B) {
	cache := NewCache[int, string]()

	n := 100000
	value := strings.Repeat("a", 256)
	for i := 0; i < n; i++ {
		cache.Put(i, value)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(n)
			val, ok, _ := cache.Get(key)
			if !ok {
				b.Errorf("key: %v; value: %v", key, val)
			}
			if val != value {
				b.Errorf("expected: %v; got: %v", val, value)
			}
		}
	})
	b.ReportAllocs()
}

func Benchmark_GetIfPresentConcurrently(b *testing.B) {
	cache := NewCache[int, string]()

	n := 100000
	value := strings.Repeat("a", 256)
	for i := 0; i < n; i++ {
		cache.Put(i, value)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(n)
			val, ok := cache.GetIfPresent(key)
			if !ok {
				b.Errorf("key: %v; value: %v", key, val)
			}
			if val != value {
				b.Errorf("expected: %v; got: %v", val, value)
			}
		}
	})
	b.ReportAllocs()
}

func Benchmark_CountConcurrently(b *testing.B) {
	cache := NewCache[int, int]()

	n := 100000
	for i := 0; i < n; i++ {
		cache.Put(i, i)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := n / 2
			for i := 0; i < c; i++ {
				cache.Remove(i)
			}
			after := cache.Count()

			if after != c {
				b.Errorf("expected: %v; got: %v", c, after)
			}

		}
	})
	b.ReportAllocs()
}

func Benchmark_KeysConcurrently(b *testing.B) {
	cache := NewCache[int, int]()

	n := 100000
	for i := 0; i < n; i++ {
		cache.Put(i, i)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := n / 2
			for i := 0; i < c; i++ {
				cache.Remove(i)
			}
			after := cache.Keys()

			if len(after) != c {
				b.Errorf("expected: %v; got: %v", c, after)
			}

		}
	})
	b.ReportAllocs()
}

func Benchmark_ValuesConcurrently(b *testing.B) {
	cache := NewCache[int, int]()

	n := 100000
	for i := 0; i < n; i++ {
		cache.Put(i, i)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := n / 2
			for i := 0; i < c; i++ {
				cache.Remove(i)
			}
			after := cache.Values()

			if len(after) != c {
				b.Errorf("expected: %v; got: %v", c, after)
			}

		}
	})
	b.ReportAllocs()
}
