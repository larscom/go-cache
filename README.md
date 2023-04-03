# GO-CACHE

[![Go Report Card](https://goreportcard.com/badge/github.com/larscom/go-cache)](https://goreportcard.com/report/github.com/larscom/go-cache)
[![Go Reference](https://pkg.go.dev/badge/github.com/larscom/go-cache.svg)](https://pkg.go.dev/github.com/larscom/go-cache)
[![codecov](https://codecov.io/gh/larscom/go-cache/branch/master/graph/badge.svg?token=E9wcYNmOYN)](https://codecov.io/gh/larscom/go-cache)

> Simple in-memory `thread safe` cache

- With Loader (optional)
  - Common use case: fetch some data from an API and store in cache
- With TTL (optional)

## 🚀 Install

```sh
go get github.com/larscom/go-cache
```

## 💡 Usage

You can import `go-cache` using:

```go
import (
    "github.com/larscom/go-cache"
)
```

> Create a new cache with `int` type as key and `string` type as value. Which creates a regular cache.

```go
func main() {
    c := cache.NewCache[int, string]()
}
```

With `loader` function

> This function gets called whenever the requested key is not available in the cache and will update the cache automatically with the value returned from the loader function.

> Note: this function will only get called once if called from multiple go routines at the same time.

```go
func main() {
    // this loader function gets only called once, even when calling from multiple go routines
    loader := cache.WithLoader[int, string](func(key int) (string, error) {
        resp, err := http.Get(fmt.Sprintf("https://example.com/user/%d", key))
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        r, err := ioutil.ReadAll(resp.Body)
        return string(r), nil
	})

    c := cache.NewCache(loader)

    // use Get() with a loader
    value, err := c.Get(123)
}
```

With `TTL`

> Create a new cache with time to live of 10 seconds.

```go
func main() {
    c := cache.NewCache(cache.WithExpireAfterWrite[int, string](time.Second * 10))
    defer c.Close()

    // use GetIfPresent() without a loader
    value, found := c.GetIfPresent(123)
}
```

With `onExpired` function

> This function gets called whenever an item in the cache expires.

```go
func main() {
    ttl := cache.WithExpireAfterWrite[int, string](time.Second * 10)
    c := cache.NewCache(ttl, cache.WithOnExpired[int, string](func(key int, value string) {
        // do something with expired key/value
	}))
    defer c.Close()
}
```

## ⚡️ Interface

```go
type Cache[Key comparable, Value any] interface {
	// Clears the whole cache
	Clear()
	// Stop the timers
	Close()
	// Total amount of entries
	Count() int
	// Loop over each entry in the cache
	ForEach(func(Key, Value))
	// Get item with the loader function (if configured)
	// it is only ever called once, even if it's called from multiple goroutines.
	// When no loader is configured, use GetIfPresent instead
	Get(Key) (Value, error)
	// Get item from cache (if present) without loader
	GetIfPresent(Key) (Value, bool)
	// Refresh item in cache
	Refresh(Key) (Value, error)
	// Check to see if the cache contains a key
	Has(Key) bool
	// Get all keys, it will be in indeterminate order.
	Keys() []Key
	// Add a new item to the cache
	Put(Key, Value)
	// Remove an item from the cache
	Remove(Key)
	// Get the map with the key/value pairs, it will be in indeterminate order.
	ToMap() map[Key]Value
	// Get all values, it will be in indeterminate order.
	Values() []Value
}
```
