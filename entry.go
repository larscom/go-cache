package cache

import "time"

var zeroTime = time.Time{}

type entry[K comparable, V any] struct {
	key      K
	value    V
	expireAt time.Time
}

func newEntry[K comparable, V any](
	key K,
	value V,
	expireAt time.Time,
) *entry[K, V] {
	return &entry[K, V]{
		key:      key,
		value:    value,
		expireAt: expireAt,
	}
}

func (e *entry[K, V]) isExpired() bool {
	if e.expireAt.IsZero() {
		return false
	}
	return time.Now().After(e.expireAt)
}

func (e *entry[K, V]) isValid() bool {
	return !e.isExpired()
}
