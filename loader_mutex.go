package cache

import "sync"

type loaderMutex[K comparable] struct {
	sync.Map
}

func (m *loaderMutex[K]) lock(key K) func() {
	value, _ := m.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return func() {
		mu.Unlock()
		m.Delete(key)
	}
}
