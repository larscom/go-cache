package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpired(t *testing.T) {
	t.Run("expired", func(t *testing.T) {
		e := cacheEntry[int, int]{
			expiration: time.Now().Add(time.Millisecond * 5),
		}

		<-time.After(time.Millisecond * 10)
		assert.True(t, e.isExpired())
	})

	t.Run("not expired", func(t *testing.T) {
		e := cacheEntry[int, int]{
			expiration: time.Now().Add(time.Millisecond * 10),
		}

		<-time.After(time.Millisecond * 5)
		assert.False(t, e.isExpired())
	})
}
