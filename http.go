package raincache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

//对应一个域名指定Path的Http请求集合
type HTTPPool struct {
	domain string
	basePath string
}

func NewHTTPPool(domain string, basePath string) *HTTPPool  {
	return &HTTPPool{
		//当前域名
		domain:   domain,
		//目前绑定的Path e.g example.com/Path
		basePath: basePath,
	}
}

func (p *HTTPPool)Log(format string,v... interface{} )  {
	log.Printf("[Server %s] %s",format,fmt.Sprintf(format,v...))
}

// 服务url：domain/#{basePath}/#{groupName}/#{key}
func (p *HTTPPool)ServeHTTP(w http.ResponseWriter, r *http.Request)  {
	if !strings.HasPrefix(r.URL.Path,p.basePath){
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	p.Log("%s  %s",r.Method,r.URL.Path)
	parts:=strings.SplitN(r.URL.Path[len(p.basePath):],"/",2)
	if len(parts)!=2{
		http.Error(w,"input url path is not right",http.StatusBadRequest)
	}

	groupName:=parts[0]
	key:=parts[1]
	group:=GetGroup(groupName)
	if group==nil{
		http.Error(w,"The Cache Group you chose is not exist",http.StatusBadRequest)
		return
	}

	value,err:=group.Get(key)

	if err!=nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type","application/octet-stream")
	_,err=w.Write(value.ByteSlice())
	if err!=nil{
		http.Error(w,err.Error(),http.StatusBadRequest)
	}
}
