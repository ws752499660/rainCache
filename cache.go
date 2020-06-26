package raincache

import (
	"./lru"
	"log"
	"sync"
)

//实现了单机并发的cache结构
type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	//缓存最大空间，用于实例化lru成员
	cacheBytes int64
}

func (c *cache)add(key string,value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru==nil{
		c.lru=lru.New(c.cacheBytes,func(key string, value lru.Value){
			log.Printf("key:%s value:%s has been Eliminated from the cache",key,value)
		})
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


