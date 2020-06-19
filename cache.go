package raincache

import (
	"./lru"
	"sync"
)

//实现了单机并发的cache结构
type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	cacheBytes int64
}

func (c *cache)add(key string,value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru==nil{
		c.lru=lru.New(c.cacheBytes,nil)
	}
	c.lru.Add(key,value)
}

func (c *cache)get(key string) (value ByteView,ok bool)  {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru==nil{
		return
	}

	v,ok:=c.lru.Get(key)
	if ok{
		value=v.(ByteView)
		return
	}

	return
}


