# GO-CACHE

[![codecov](https://codecov.io/gh/larscom/go-cache/branch/master/graph/badge.svg?token=E9wcYNmOYN)](https://codecov.io/gh/larscom/go-cache)
[![Go Reference](https://pkg.go.dev/badge/github.com/larscom/go-cache.svg)](https://pkg.go.dev/github.com/larscom/go-cache)

> Simple in-memory `thread safe` cache with loader (optional) and TTL (optional) and max-size (optional)

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

> Create a new cache with `int` type as key and `string` type as value. Which creates a regular cache, nothing special.

```go
func main() {
    c := cache.NewCache[int, string]()
}
```

With `TTL`

> Create a new cache with time to live of 10 seconds for all entries

```go
func main() {
    c := cache.NewCache(cache.WithExpireAfterWrite[int, string](time.Second * 10))
    defer c.Close()
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
}
```

With `max size`

> Create a new cache with a fixed size of 300. When going beyond the max size, the first item is removed from the cache.

```go
func main() {
    c := cache.NewCache(cache.WithMaxSize[int, int](300))
}
```

With `onExpire` function

> This function gets called whenever an item in the cache expires

```go
func main() {
    ttl := cache.WithExpireAfterWrite[int, string](time.Second * 10)
    c := cache.NewCache(ttl, cache.WithOnExpired[int, string](func(key int, value string) {
        // do something with expired key/value
	}))
    defer c.Close()
}
```
