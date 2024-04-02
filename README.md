# GO-CACHE

[![Go Report Card](https://goreportcard.com/badge/github.com/larscom/go-cache)](https://goreportcard.com/report/github.com/larscom/go-cache)
[![Go Reference](https://pkg.go.dev/badge/github.com/larscom/go-cache.svg)](https://pkg.go.dev/github.com/larscom/go-cache)
[![codecov](https://codecov.io/gh/larscom/go-cache/graph/badge.svg?token=E9wcYNmOYN)](https://codecov.io/gh/larscom/go-cache)

> High performance, simple generic cache written in GO, including a `loading` cache.

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

## 🫱 Loading cache

> Create a new loading cache with `int` type as key and `string` type as value.

A common use case for this loading cache would be to automatically fetch data from a REST API and store it in cache. This implementation will ensure that the REST API is only called once in a concurrent environment.

```go
func main() {
  	loaderFunc := func(key int) (string, error) {
         // you may want to call your REST API here...
         return "Hello World", nil
	  }

    c := cache.NewLoadingCache[int, string](loaderFunc)
    defer c.Close()

    value, err := c.Load(1)
    if err != nil {
      log.Fatal(err)
    }
    log.Println(value) // Hello World
}
```

With `TTL` option

> Create a new loading cache with time to live of 10 seconds.

This allows you to call `Load()` as many times as you want and whenever an entry expires it'll call the `loaderFunc` once.

```go
func main() {
    c := cache.NewLoadingCache(loaderFunc, cache.WithExpireAfterWrite[int, string](time.Second * 10))
    defer c.Close()
}
```

## 🫱 Cache

> Create a `regular` cache (without `Load` and `Reload` functions) with `TTL`

```go
func main() {
    c := cache.NewCache[int, string](cache.WithExpireAfterWrite[int, string](time.Second * 10))
    defer c.Close()

    c.Put(1, "Hello World")

    value, found := c.Get(1)
    if found {
       log.Println(value) // Hello World
    }
}
```
