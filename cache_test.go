package ucenter

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := Cache{expire: 5, checkInterval: 2}
	cache.Init()
	defer cache.Close()
	cache.Set("name", "xu")
	v := cache.Get("name")
	if v != "xu" {
		t.Fatal("cache get and set error")
	}
	time.Sleep(13 * time.Second)
	v = cache.Get("name")
	if v != "" {
		t.Fatal("cache set expire error")
	}
}
