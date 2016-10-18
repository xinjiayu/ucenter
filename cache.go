package ucenter

import (
	"fmt"
	"sync"
	"time"
)

// Cache implements a simple in-memory cache used token check
// expire at least 5 seconds
type Cache struct {
	sync.Mutex
	mapping map[string]*Value
	expire  int
	end     chan int
}

// Value cache value which created time
type Value struct {
	value   string
	created int64
}

// Init cache,
func (c *Cache) Init() {
	c.mapping = make(map[string]*Value)
	c.end = make(chan int, 1)
	if c.expire >= 5 { // At least 5 seconds
		go c.checkExpired()
	}
}

func (c *Cache) checkExpired() {
	for {
		select {
		case <-time.After(2 * time.Second):
			c.Lock()
			now := time.Now().Unix()
			var cleankeys []string
			for k, v := range c.mapping {
				if v != nil {
					if now-v.created > int64(c.expire) {
						cleankeys = append(
							cleankeys, k)
					}
				}
			}
			for i := 0; i < len(cleankeys); i++ {
				_, ok := c.mapping[cleankeys[i]]
				if ok {
					delete(c.mapping, cleankeys[i])
				}
			}
			c.Unlock()
		case <-c.end:
			return
		}
	}
}

// Close end goroutine
func (c *Cache) Close() {
	c.end <- 1
}

// Get get cache
func (c *Cache) Get(key string) string {
	c.Lock()
	defer c.Unlock()
	v, ok := c.mapping[key]
	if ok {
		return v.value
	}
	return ""
}

// Set set cache
func (c *Cache) Set(key string, val string) {
	c.Lock()
	defer c.Unlock()
	v := Value{val, time.Now().Unix()}
	c.mapping[key] = &v
}

// Delete delete cache
func (c *Cache) Delete(key string) {
	c.Lock()
	defer c.Unlock()
	_, ok := c.mapping[key]
	if ok {
		delete(c.mapping, key)
	}
}
