async_cache
=====

A golang LRU Cache that cached values are updated asynchronously.

Install:
```bash
go get github.com/fondoger/async_cache
```

Example:
```go
loader := func(key string) (interface{}, error){
    var result string
    // write your data loader here
    return result, nil
}
cache := NewAsyncCache(10000, time.Hour, time.Minute, loader)
result, err := cache.Get("example_key")
fmt.Println(result, err)
```

Go to [Go Doc Page](https://pkg.go.dev/github.com/fondoger/async_cache?tab=doc) for documentation.

