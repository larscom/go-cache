# GO-CACHE

[![codecov](https://codecov.io/gh/larscom/go-cache/branch/master/graph/badge.svg?token=E9wcYNmOYN)](https://codecov.io/gh/larscom/go-cache)
[![Go Reference](https://pkg.go.dev/badge/github.com/larscom/go-cache.svg)](https://pkg.go.dev/github.com/larscom/go-cache)

> Simple in-memory `thread safe` cache with loader (optional) and TTL (optional)

Although performing pretty well, the goal of this cache is not to be the fastest and/or the most memory efficient. Under the hood this cache just uses a map to store its data.

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
}
```

With `loader` function

> This function gets called whenever the requested key is not available in the cache and will update the cache automatically with the value returned from the loader function.

```go
func main() {
    c := cache.NewCache(cache.WithLoader[int, string](func(key int) (string, error) {
        resp, err := http.Get(fmt.Sprintf("https://example.com/user/%d", key))
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        r, err := ioutil.ReadAll(resp.Body)
        return string(r), nil
	}))
}
```

With `onExpire` function

> This function gets called whenever an item in the cache expires

```go
func main() {
    c := cache.NewCache(cache.WithOnExpired[int, string](func(key int, value string) {
        // do something with expired key/value
	}))
}
```
