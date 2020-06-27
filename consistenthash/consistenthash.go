package consistenthash

//通过实现一致性哈希，使得对应缓存可以找到其应该存放的节点位置

import(
	"hash/crc32"
	"sort"
	"strconv"
)

//定义本一致性哈希的散列函数的函数类型：从[]byte的数据转化为uint32
type Hash func(data []byte) uint32

//一致性HashMap结构体
type ConsistentMap struct {
	//散列函数
	hash Hash
	//虚拟节点倍数因子（一个真实节点对应多少虚拟节点）
	replicas int
	//节点（包括虚拟节点）的散列值
	nodesHash []int
	//虚拟节点与真实节点的map（key为散列值，value为真实节点的名称）
	mappingMap map[int]string
}

func NewConsistentMap(replicas int,hashFn Hash) *ConsistentMap  {
	c:=&ConsistentMap{
		hash:       hashFn,
		replicas:   replicas,
		mappingMap: make(map[int]string),
	}
	//若未指定散列函数，考虑到使用32位整数保存哈希值
	//我们使用crc32.ChecksumIEEE作为默认的散列函数
	if c.hash==nil{
		c.hash=crc32.ChecksumIEEE
	}
	return c
}

//向一致性哈希节点表中添加节点
func (c *ConsistentMap)AddNode(names ...string)  {
	for _,name :=range names{
		for i := 0; i < c.replicas; i++ {
			hashValue:=int(c.hash([]byte(strconv.Itoa(i)+name)))
			c.nodesHash=append(c.nodesHash,hashValue)
			c.mappingMap[hashValue]=name
		}
	}
	//为方便查找距离最近的节点hash，使节点hash数组始终有序
	sort.Ints(c.nodesHash)
}

func (c *ConsistentMap)GetNode(key string) string  {
	if len(c.nodesHash)==0{
		return ""
	}
	keyHash:=int(c.hash([]byte(key)))
	//获取距离keyHash最近的节点Hash在nodeHash中的下标
	//因为nodeHash是有序的，所以可以用基于二分查找的sort.Search
	nodeHashIndex:=sort.Search(len(c.nodesHash), func(i int) bool {
		return c.nodesHash[i]>=keyHash
	})

	//若出现了keyHash比所有nodeHash都要大的情况（查找失败返回len(c.nodesHash)）
	//由于一致性哈希是个环，所以让它被放置在最小的nodeHash对应的节点上
	if nodeHashIndex==len(c.nodesHash){
		nodeHashIndex=0
	}

	return c.mappingMap[c.nodesHash[nodeHashIndex]]
}
