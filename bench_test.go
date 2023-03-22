package cache

import (
	"testing"
	"time"
)

func Benchmark_Put(b *testing.B) {

	b.Run("With TTL", func(b *testing.B) {
		ttl := WithExpireAfterWrite[int, int](1 * time.Second)
		cache := NewCache(ttl)
		for i := 0; i < b.N; i++ {
			for y := 0; y < 50; y++ {
				cache.Put(1, y)
			}
		}
	})

	b.Run("Without TTL", func(b *testing.B) {
		cache := NewCache[int, int]()
		for i := 0; i < b.N; i++ {
			for y := 0; y < 50; y++ {
				cache.Put(1, y)
			}
		}
	})

}
