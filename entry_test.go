package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEntryExpired(t *testing.T) {
	entry := newEntry(0, 0, time.Now().Add(time.Millisecond*5))

	<-time.After(time.Millisecond * 10)

	assert.True(t, entry.isExpired())
	assert.False(t, entry.isValid())
}

func TestEntryNotExpired(t *testing.T) {
	entry := newEntry(0, 0, time.Now().Add(time.Millisecond*10))

	<-time.After(time.Millisecond * 5)

	assert.False(t, entry.isExpired())
	assert.True(t, entry.isValid())
}

func TestEntryNotExpiredZeroTime(t *testing.T) {
	entry := newEntry(0, 0, zeroTime)

	assert.False(t, entry.isExpired())
	assert.True(t, entry.isValid())
}
