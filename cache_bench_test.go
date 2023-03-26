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
	cache := NewCache[string, string]()

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
	value := strings.Repeat("a", 256)
	cache := NewCache[int, string]()
	for i := 0; i < 100000; i++ {
		cache.Put(i, value)
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := rand.Intn(100000)
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
