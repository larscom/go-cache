package cache

import "sync"

type KeyedMutex[Key comparable] struct {
	sync.Map
}

func (m *KeyedMutex[Key]) lock(key Key) func() {
	value, _ := m.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)

	mu.Lock()

	return func() {
		mu.Unlock()
		m.Delete(key)
	}
}
