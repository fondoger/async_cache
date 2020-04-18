// A golang LRU Cache that cached values are updated asynchronously
package async_cache

import (
	"log"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

type CachedVal struct {
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

// Instructions:
// * size: The size of LRU cache pool. If memory is enough, a larger
//   size is always better, such as 100000
//
// * maxAge: The expire time of cached value. Once the value is cached,
//   `Get` method will always return cached value before expire time.
//
// * updateInterval: The asynchronously update interval. We will try to
//   update cached value in each interval. Suggested range is between (0, maxAge/2),
//   of course the smaller, the better.
//   Note the update action is invoked when calling `Get` method, instead of by a timer.
//
// * loaderFunc: the data load function. Once the loaderFunc return a <nil> error,
//   the result of loaderFunc will be cached.
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
		value := hit.(*CachedVal)
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
		c.Caches.Add(key, &CachedVal{
			val:         result,
			dataTime:    time.Now(),
			requestTime: now,
		})
	}
	return result, err
}

func (c *AsyncCache) ClearAll() {
	c.Caches.Purge()
}
