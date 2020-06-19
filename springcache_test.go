package raincache

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	v, _ := f.Get("key")
	if !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"收割机":  "油料不足",
	"拖拉机": "发动机爆缸",
	"拖车":  "冷却液温度异常",
	"植保无人机": "电量不足",
	"温室温度传感器":"无网络连接",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 100, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)

			v, ok := db[key]
			if ok {
				_, ok := loadCounts[key]
				if !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		fmt.Printf("key:%s,value:%v\n",k,v)
		view, err := gee.Get(k)
		if  err != nil || view.ToString() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		_, err = gee.Get(k)
		if err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}
	view, err := gee.Get("unknown")
	if err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
