package raincache

import (
	"errors"
	"fmt"
	"sync"
)

//缓存未命中时获取源数据的回调的接口（是一个回调函数）
//定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。
//这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。
type Getter interface {
	Get(key string) ([]byte, error)
}

type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group 可以认为是一个缓存的命名空间
type Group struct {
	name string
	getter Getter
	cache cache
}

var(
	mu sync.RWMutex
	groups=make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group  {
	if getter==nil{
		panic("Getter is nil!")
	}
	mu.Lock()
	defer mu.Unlock()
	g:= &Group{
		name:   name,
		getter: getter,
		cache:  cache{cacheBytes: cacheBytes},
	}
	groups[name]=g
	return g
}

func GetGroup(name string) *Group{
	mu.RLock()
	defer mu.RUnlock()
	return groups[name]
}

func (g *Group)Get(key string) (ByteView,error){
	if key=="" {
		return ByteView{},errors.New("the key is empty")
	}

	v,ok:=g.cache.get(key)
	if ok{
		fmt.Printf("sprintcache %s[%s] hit!\n",g.name,key)
		return v,nil
	}else {
		//缓存未命中进行的操作
		fmt.Printf("sprintcache %s[%s] NOT hit!\n",g.name,key)
		return g.load(key)
	}
}

func (g *Group)load(key string) (ByteView,error){
	return g.getLocally(key)
}

func (g *Group)getLocally(key string) (ByteView,error){
	bytes,err:=g.getter.Get(key)
	if err!=nil{
		return ByteView{}, err
	}
	value:=ByteView{b: bytes}
	g.pushToCache(key,value)
	return value,nil

}

func (g *Group)pushToCache(key string,value ByteView) {
	g.cache.add(key,value)
}

