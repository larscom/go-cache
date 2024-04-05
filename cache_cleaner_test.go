package cache

import (
	"testing"
	"time"

	csmap "github.com/mhmtszr/concurrent-swiss-map"
	"github.com/stretchr/testify/assert"
)

func TestStartCleaner(t *testing.T) {
	data := csmap.Create[int, *entry[int, int]]()

	data.Store(1, newEntry(1, 100, time.Now()))
	data.Store(2, newEntry(2, 200, time.Now().Add(time.Millisecond*20)))

	cleaner := newCacheCleaner(data, time.Millisecond)
	defer cleaner.Stop()

	cleaner.Start()

	<-time.After(time.Millisecond * 5)

	assert.False(t, data.Has(1))
	assert.True(t, data.Has(2))
}

func TestStopCleaner(t *testing.T) {
	data := csmap.Create[int, *entry[int, int]]()

	const key = 1

	data.Store(key, newEntry(key, 100, time.Now().Add(time.Millisecond*20)))

	cleaner := newCacheCleaner(data, time.Millisecond)
	cleaner.Start()

	<-time.After(time.Millisecond * 5)
	assert.True(t, data.Has(key))

	cleaner.Stop()
	<-time.After(time.Millisecond * 30)

	assert.True(t, data.Has(key))
}
