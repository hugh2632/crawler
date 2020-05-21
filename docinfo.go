package crawler

import (
	"github.com/chromedp/cdproto/network"
)

type DocumentInfo struct {
	//服务器IP
	Ip string
	//服务器端口
	Port int
	//响应url
	RespUrl string
	//DNS加载时间,毫秒
	DnsTime int
	//页面加载时间,毫秒
	LoadTime int
	//网站响应时间,毫秒
	ResponseTime int
	//状态码
	StatusCode int
	//可以做筛选
	Resources map[string]Resource
}

type resourceMap struct {
	referUrl	string
	tp      network.ResourceType
	requestID network.RequestID
}

type Resource struct {
	Referer	string
	Type  network.ResourceType
	Value []byte
}

type resourceParams struct {
	disableResource bool //是否捕获网页资源，默认为false
	blockImage      bool
	blockCss        bool
	blockJs         bool
	blockFont       bool
	blockMedia      bool
}

func (self *resourceParams) BlockImage() *resourceParams {
	self.blockImage = true
	return self
}

func (self *resourceParams) BlockCss() *resourceParams {
	self.blockCss = true
	return self
}

//todo 启用后有时会影响主页源码获取
func (self *resourceParams) BlockJs() *resourceParams {
	self.blockJs = true
	return self
}

func (self *resourceParams) BlockFont() *resourceParams {
	self.blockFont = true
	return self
}

func (self *resourceParams) BlockMedia() *resourceParams {
	self.blockMedia = true
	return self
}
