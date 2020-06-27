# RainCache/雨缓存

本项目是为解决[springRain](https://github.com/ws752499660/springRain)农机区块链项目中，承载农机上链业务服务端程序无法及时处理大量并发请求的问题。

通过实现**简洁，高效，易实现**的分布式缓存，缓解大量并发时对承载上链业务的服务器和背后联盟内区块链网络的压力。

雨缓存支持以下特性：

* 基于LRU的缓存淘汰机制



## 总体设计

### 数据结构

* lru.Value(接口，需要实现len())：缓存中存放数据的类型
* lru.item：封装了string类型的key和实现Value接口的value，以一个整体放入lru双向链表的list.Element中
* ByteView：单个缓存元素内容存放的结构
* cache：支持并发的封装了lru缓存的缓存

## 缓存实现lru

### 思路

缓存淘汰有多重机制，较为常见的有FIFO，时钟方法，LRU和随机方法。

在redis中，使用了一种近似LRU算法。默认情况下，Redis会随机挑选5个键，并从中选择一个最久未使用的key进行淘汰。

**因此，考虑性能与简洁，我们使用标准的LRU算法作为缓存淘汰算法。**

我们设定缓存的元素为键值对<k,v>。

对于LRU而言，我们要维护一个队列，该队列是缓存中所有的元素的集合。

每当有一个新的元素要进入缓存时，若队列未满则进入队尾；若已满，则从队首出队一个元素，再使新元素入队。

若有一个队内元素被使用了，需要将其移动到队尾。

以此，队首便永远是最近最少访问的元素（优先出队/淘汰）。

### 数据结构

由思路可以知道，我们需要一个字典（map）来存储键值对，也需要一个队列来用于LRU的实现。

所以我们在lru的Cache中设置一个字典和一个双向链表。

```go
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
```

### 方法

* 构造函数（lru.New）（简单工厂）
* 向缓存中添加/修改元素（Add）
* 从缓存中获取key对应的value（Get）
* 缓存淘汰（Eliminate）
* 返回当前缓存中有多少元素（Len）

### 需要说明的点

#### 关于缓存中的存储的值的类型Value

lru中键值对的类型为string—Value

其中Value是一个接口，需要实现Len()方法（返回其所占空间）

使用接口的另一个好处就是在这里我们**并不关心其具体的数据类型**

我们只需要要求存入缓存的内容是实现了Len()方法的Value接口实现就好



#### 关于双向链表中的元素list.Element

双向链表中存储的元素类型为list.Element，其存储值的成员为一个接口（无任何方法），我们用item将<k,v>封装起来，传入list.Element中

为了方便操作，字典的<k,v>也是将list.Element直接传入：<string, *list.Element>



#### 缓存淘汰的逻辑

当已用空间>最大空间时，会发生缓存淘汰。

每当Add方法被调用时，该方法会先执行增加/修改操作，操作完成后计算缓存的已用空间。

（这样处理是因为你不会希望你刚放入缓存的数据瞬间被清理掉，即便大于最大空间你也会希望在用完本次后该数据再被清除）

若已用空间会超出最大空间，则会一直触发缓存淘汰，知道已用空间小于最大空间：

1. 从队首（链表表尾）获得最久未使用的元素
2. 将该元素出队（从链表中删除）
3. 从map中将该元素删除
4. 重新计算当前缓存已用空间
5. 如有缓存淘汰时需要执行的回调函数，执行其回调函数



## 用来表示缓存值的只读数据结构ByteView

### 思路

为了防止缓存值被获取缓存值的外部程序随意修改，需要一种对外部只读的数据结构来存储缓存值

在ByteView内部使用一个[]byte来存储各种不同的缓存数据

使这个[]byte为私有（开头以小写字母）并不对外暴露修改接口来防止外部访问

### 数据结构

```go
type ByteView struct {
	//实际存放缓存数据的地方
	b []byte
}
```

### 方法

* 返回当前缓存元素的字节数（Len）
* 字符串化（ToString）
* 返回实际存放数据的字节数组的切片（深拷贝）（ByteSlice）



## 单机并发缓存cache

cache封装了lru.cache

lru.cache是线程不安全的（协程不安全？)，考略到本缓存的使用场景，需要实现一个支持并发的cache对lru.cache进行封装。

### 思路

我们使用互斥锁（sync.Mutex）来控制对cache的操作

我们只允许一次只有一个线程（协程？）可以对一个cache进行操作

且由于无论是cache的读、写都是要对lru的队列进行更改操作，所以不使用读写锁

所以对cache的所有操作方法均加入互斥锁

### 数据结构

```go
type cache struct {
	mu sync.Mutex
	lru *lru.Cache
	//缓存最大空间，用于实例化lru成员
	cacheBytes int64
}
```

### 方法

* 向cache中添加元素（add）
* 从cache中获取元素（get）

## 负责与外部交互，控制缓存存储和获取的主流程springcache

### 思路

为了使得本缓存系统支持多个命名空间，以便于在存储不同场景下不同意义但同key值的数据，需要引入封装不同cache实例的数据结构。

且这种数据结构应当基于cache的方法，实现缓存所需要暴露在外的功能：

1. 返回key对应的数据
2. key未在缓存中命中时，从数据源拿到数据放入缓存中并返回该数据

且对于该种数据结构本身，由于可能会存在多个，也需要实现对其本身的管理：

1. 命名空间的新建
2. 命名空间的获取

关于从数据源拿到源数据的具体方法，应该是创建缓存命名空间时指定。我们可以允许用户传入一个函数来执行该操作，该函数必须实现传入key，返回[]byte和error的功能。

#### 关于用户传入函数实现获取源数据的操作

我们需要将用户传入的函数保存为命名空间结构体的成员，以便在需要获取源数据的时候调用。

为了更好的语义性、通用性和拓展性，我们不使用直接传函数的形式，而使用传接口的形式（接口可以后续再添加新的功能）

在这里，为了避免每次都要传入接口实例的繁琐，我们将用户传入的函数抽象为一个实现了Get方法的接口，具体做法是：

用户传给命名空间的，是一个GetterFunc函数类型的函数（springcache.GetterFunc(匿名函数)）（有和GetterFunc一样的参数和返回类型），且该函数类型在springcache中被定义为实现了Get方法的接口。因此在命名空间的结构体中由Getter接口来保存该函数类型。

Getter接口实现的Get方法则是直接return GetterFunc本身。且在命名空间的实际使用中也是调用Get方法而非函数类型本身。

*附代码实现*

```go
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
```

#### 并发

由于可能存在多个命名空间，所以对于命名空间本身的新建和获取也应该是线程安全的操作。

我们引入读写锁，对于新建命名空间NewGroup我们加上写锁，对于获取命名空间GetGroup我们加上读锁。

这一过程利用sync.RWMutex实现

### 数据结构

```go
type Group struct {
	name string
	getter Getter
	cache cache
}
```

其中，关于Group自身的管理，我们使用map[string] *Group来管理不同Group的指针

### 方法

* 工厂函数，创建Group的实例（NewGroup）
* 获取已经创建的Group（GetGroup）
* 封装了并发的cache的get的调用和缓存未命中时逻辑（Get）
* 缓存未命中时加载数据到缓存中的方法（load）
* 从本地获取数据到缓存中的方法（getLocally）
* 将数据放入缓存中的方法（pushToCache）

## HTTP服务端http

### 思路

要实现服务端，使得通过访问某个URL时可以返回对应key的值

我们将服务url设计为：

```
http://#{domain}/#{basePath}/#{groupName}/#{key}
```

我们使用net/http库来实现服务端

我们设计一个记录domain和basePath的结构体HTTPPool，并将其作为http.Handler接口的一个实现（即使其实现`ServeHTTP(w ResponseWriter, r *Request)`方法）

在该结构体的ServeHTTP方法中，我们通过url中指定的groupName和Key的参数，获取指定的Group，并调用Group的Get方法，传入Key得到我们想要的数据，并将其返回给请求者。

### 数据结构

```go
type HTTPPool struct {
	domain string
	basePath string
}
```

### 方法

* 工厂函数（NewHTTPPool）
* 日志打印方法（Log）
* 请求承接方法，通过url中传入的参数返回对应的缓存数据（ServeHTTP）

## 一致性哈希

