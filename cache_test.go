package ucenter

import (
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := Cache{expire: 1, checkInterval: 1}
	cache.Init()
	defer cache.Close()
	cache.Set("name", "xu")
	v := cache.Get("name")
	if v != "xu" {
		t.Fatal("cache get and set error")
	}
	time.Sleep(3 * time.Second)
	v = cache.Get("name")
	if v != "" {
		t.Fatal("cache set expire error")
	}
}
