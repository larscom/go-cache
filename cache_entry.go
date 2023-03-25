package cache

import "time"

type cacheEntry[Key comparable, Value any] struct {
	key        Key
	value      Value
	expiration time.Time
}

func (e *cacheEntry[Key, Value]) isExpired() bool {
	if e.expiration.IsZero() {
		return false
	}
	return time.Now().After(e.expiration)
}
