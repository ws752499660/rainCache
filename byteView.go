package raincache


//ByteView是一个
type ByteView struct {
	//实际存放缓存数据的地方
	b []byte
}

//返回当前缓存元素的字节数
func (v ByteView)Len() int{
	return len(v.b)
}

//字符串化
func (v ByteView)ToString() string{
	return string(v.b)
}

//返回实际存放数据的切片(深拷贝)
func (v ByteView)ByteSlice() []byte  {
	temp:=make([]byte,len(v.b))
	copy(temp,v.b)
	return temp
}