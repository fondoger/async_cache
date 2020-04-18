// A golang LRU Cache that cached values are updated asynchronously.
//
// Basic Example:
//	loader := func(key string) (interface{}, error) {
//		var result string
//		// write your data loader here
//		return result, nil
//	}
//	cache := NewAsyncCache(10000, time.Hour, time.Minute, loader)
//	result, err := cache.Get("example_key")
//	fmt.Println(result, err)
//
// Description:
//
// - size: The size of LRU cache pool. If memory is enough, a larger
// size is always better, such as 100000
//
// - maxAge: The expire time of cached value. Once the value is cached,
// `Get` method will always return cached value before expire time.
//
// - updateInterval: The asynchronously update interval. We will try to
// update cached value in each interval. Suggested range is between (0, maxAge/2),
// of course the smaller, the better.
// Note the update action is invoked when calling `Get` method, instead of by a timer.
//
// - loaderFunc: the data load function. Once the loaderFunc return a <nil> error,
// the result of loaderFunc will be cached.
//
package async_cache

import (
	"log"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type cachedVal struct {
	val         interface{}
	dataTime    time.Time // the data updated time
	requestTime time.Time // previous LoaderFunc called time
}

type LoaderFunc func(key string) (interface{}, error)

type AsyncCache struct {
	Caches         *lru.Cache
	MaxAge         time.Duration
	UpdateInterval time.Duration
	LoaderFunc     LoaderFunc
	DisableLog     bool
}

func NewAsyncCache(size int, maxAge time.Duration, updateInterval time.Duration, loaderFunc LoaderFunc) *AsyncCache {
	if size <= 0 {
		size = 10000
	}
	lruCache, _ := lru.New(size)
	return &AsyncCache{
		Caches:         lruCache,
		MaxAge:         maxAge,
		UpdateInterval: updateInterval,
		LoaderFunc:     loaderFunc,
	}
}

// If exists, always get from cache (err == <nil>);
// If not exists, return the result of LoaderFunc.
func (c *AsyncCache) Get(key string) (interface{}, error) {
	now := time.Now()
	if hit, ok := c.Caches.Get(key); ok {
		value := hit.(*cachedVal)
		if now.Sub(value.dataTime) < c.MaxAge {
			// Note: no lock here, so the loaderFunc might be called
			// more than once in some extreme cases.
			if now.Sub(value.requestTime) > c.UpdateInterval {
				value.requestTime = now
				go func() {
					result, err := c.LoaderFunc(key)
					if err != nil {
						if !c.DisableLog {
							log.Printf("[AsyncCache] failed update cache for key: %s", key)
						}
					} else {
						value.val = result
						value.dataTime = time.Now()
					}
				}()
			}
			return value.val, nil
		} else {
			// remove key if expired
			c.Caches.Remove(key)
		}
	}

	result, err := c.LoaderFunc(key)
	if err == nil {
		c.Caches.Add(key, &cachedVal{
			val:         result,
			dataTime:    time.Now(),
			requestTime: now,
		})
	}
	return result, err
}

func (c *AsyncCache) MGet(keys ...string) (result map[string]interface{}, errors map[string]error) {
	if len(keys) == 0 {
		return nil, nil
	}
	result = make(map[string]interface{}, len(keys)*2)
	errors = make(map[string]error, len(keys)*2)
	now := time.Now()
	var wg sync.WaitGroup
	var lock sync.Mutex
	for _, key := range keys {
		if hit, ok := c.Caches.Get(key); ok {
			value := hit.(*cachedVal)
			if now.Sub(value.dataTime) < c.MaxAge {
				lock.Lock()
				result[key] = value.val
				lock.Unlock()
				continue
			}
		}
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			val, err := c.Get(key)
			lock.Lock()
			if err != nil {
				errors[key] = err
			} else {
				result[key] = val
			}
			lock.Unlock()
		}(key)
	}
	wg.Wait()
	return result, errors
}

func (c *AsyncCache) ClearAll() {
	c.Caches.Purge()
}
