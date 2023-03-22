# GO-CACHE

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

func main() {
    // create a new cache with `int` type as key and `string` type as value
    c := cache.NewCache[int, string]()

    // with time to live of 10 seconds
    ttl := cache.WithExpireAfterWrite[int, string](time.Second * 10)
    c := cache.NewCache(ttl)

    // with loader, automatically updates the cache after retrieving the value
	loader := cache.WithLoader[int, string](func(key int) (string, error) {
        resp, err := http.Get("https://example.com")
        if err != nil {
            return "", err
        }
        defer resp.Body.Close()
        r, err := ioutil.ReadAll(resp.Body)
        return string(r), nil
	})
    c := cache.NewCache(loader)

    // with on expired callback
	onExpired := cache.WithOnExpired[int, string](func(key int, value string) {
        // do something with expired key/value
	})
    c := cache.NewCache(onExpired)

    // or you can add them all...
    c := cache.NewCache(ttl, loader, onExpired)
}
```

## 🤠 Interface

```go
type ICache[Key comparable, Value any] interface {
	Clear()
	Close()
	Count() int
	ForEach(func(Key, Value))
	Get(Key) (Value, bool, error)
	Has(Key) bool
	Keys() []Key
	Put(Key, Value)
	Reload(Key) (Value, bool, error)
	Remove(Key)
	Values() []Value
}
```
