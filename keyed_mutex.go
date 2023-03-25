package cache

import "sync"

type keyedMutex[Key comparable] struct {
	mutexes sync.Map
}

func (m *keyedMutex[Key]) lock(key Key) func() {
	value, _ := m.mutexes.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return func() { mu.Unlock() }
}
