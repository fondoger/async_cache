package async_cache

import (
	"fmt"
	"testing"
	"time"
)

var counter = 0

func GetDataRemotely(key string) (interface{}, error) {
	time.Sleep(time.Millisecond * 50)
	counter += 1
	return fmt.Sprintf("exampleKey: %s, counter[%v]", key, counter), nil
}

func TestAsyncCache(t *testing.T) {
	counter = 0
	{
		cacheStorage := NewAsyncCache(100, time.Second*10, time.Second, GetDataRemotely)
		for i := 0; i < 5; i++ {
			result, _ := cacheStorage.Get("example_key")
			t.Logf("result: %s\n", result)
		}
		if counter != 1 {
			t.Fail()
		}
	}
	counter = 0
	{
		cacheStorage := NewAsyncCache(100, time.Hour, time.Second, GetDataRemotely)
		endTime := time.Now().Add(time.Second*10 + time.Millisecond*200)
		threads := 3
		totalTimes := 0
		for i := 0; i < threads; i++ {
			go func() {
				for time.Now().Before(endTime) {
					_, _ = cacheStorage.Get("example_key")
					totalTimes += 1
				}
			}()
		}
		time.Sleep(time.Second * 12)
		t.Logf("counter: %d, totalTimes: %d\n", counter, totalTimes)
		if !(counter >= 11 && counter <= 11*threads) {
			t.Fail()
		}
	}
	counter = 0
	{
		cacheStorage := NewAsyncCache(100, time.Second, time.Second*2, GetDataRemotely)
		_, _ = cacheStorage.Get("example_key")
		time.Sleep(time.Second)
		_, _ = cacheStorage.Get("example_key")
		time.Sleep(time.Second)
		_, _ = cacheStorage.Get("example_key")
		if counter != 3 {
			t.Fail()
		}
	}
}

func TestMGet(t *testing.T) {
	cache := NewAsyncCache(100, time.Second, time.Second*2, GetDataRemotely)
	result, errors := cache.MGet("key1", "key2", "key3")
	if len(errors) != 0 || len(result) != 3 {
		t.Fatal("has error")
	}
}
