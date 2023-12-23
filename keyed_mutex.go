package cache

import "sync"

type keyedMutex[Key comparable] struct {
	sync.Map
}

func (m *keyedMutex[Key]) lock(key Key) func() {
	value, _ := m.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)

	mu.Lock()

	return func() {
		mu.Unlock()
		m.Delete(key)
	}
}
