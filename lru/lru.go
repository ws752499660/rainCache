package lru

import "container/list"

type Cache struct {
	maxBytes int64
	usedBytes int64
	lruList *list.List
	//字典中value存的是list的Element的指针
	cache map[string] *list.Element
	//某条记录被移除时的回调函数
	onEvicted func(key string, value Value)
}

//ele是指链表中的元素，在本程序中ele.Value的类型是*item
//kv是链表中的元素(Element.Value)经类型断言转化后的item类型（的指针）
//而item又实现了自定义的Value接口(必须要实现Len()函数)

//list所有的单元素插入函数（push）都会返回当前插入到表中后存在于表中的Element

//所有lruList中Element的Value接口均为item的实现
//直接加上类型断言进行转化
type item struct {
	key string
	value Value
}

//Value需要实现Len()，以使得可以得知其所占空间为多少
type Value interface {
	Len() int
}

//类构造函数（简单工厂）
func New(maxBytes int64,onEvicated func(string,Value)) *Cache{
	return &Cache{
		maxBytes:  maxBytes,
		usedBytes: 0,
		lruList:   list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicated,
	}
}

//添加/修改
func (c *Cache)Add(key string,value Value)  {
	ele,ok := c.cache[key]
	if ok{
		c.lruList.MoveToFront(ele)
		kv:=ele.Value.(*item)
		c.usedBytes=c.usedBytes-int64(kv.value.Len())+int64(value.Len())
		kv.value=value
	}else {
		ele := c.lruList.PushFront(&item{key, value})
		c.cache[key]=ele
		c.usedBytes=c.usedBytes+int64(value.Len()+len(key))
	}

	//不在之前检查空间溢出是因为本缓存设计目的是要永远保证当前一个元素的可用
	//所以现在添加/更改的元素不考虑是否溢出，只在添加/修改结束后再对缓存进行清理
	for  c.maxBytes != 0 && c.usedBytes>c.maxBytes{
		c.Eliminate()
	}

}

//从缓存中获取key对应的value
func (c *Cache)Get(key string) (value Value, ok bool)  {
	ele, ok := c.cache[key]
	if ok {
		//约定list的开始端为队尾
		c.lruList.MoveToFront(ele)
		//类型断言
		//kv是list的Element中Value接口承接的item
		kv := ele.Value.(*item)
		return kv.value,true
	}
	return nil,false
}

//进行缓存淘汰
func (c *Cache)Eliminate()  {
	//把队首的节点掏出来
	ele:=c.lruList.Back()
	if ele!=nil {
		kv := ele.Value.(*item)
		c.lruList.Remove(ele)
		delete(c.cache, kv.key)
		c.usedBytes = c.usedBytes - int64(kv.value.Len()+len(kv.key))
		if c.onEvicted!=nil{
			c.onEvicted(kv.key,kv.value)
		}
	}
}

//返回当前缓存中有多少元素
func (c *Cache)Len() int  {
	return len(c.cache)
}
