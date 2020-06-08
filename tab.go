package crawler

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Tab struct {
	ctx            context.Context
	cancel         context.CancelFunc
	browser        *browser
	LoadTimeOut    int  //秒
	WaitTime       int  //毫秒
	AcceptDialog   bool //true表示在js弹出窗中按确认， false表示取消(默认)
	resourceparams resourceParams
}

func (self *Tab) DisableCrawlResource() *resourceParams {
	self.resourceparams.disableResource = true
	return &self.resourceparams
}

func (self *Tab) Close() {
	self.cancel()
	self = nil
}

//默认为0，为0时取browser的时间
func (self *Tab) SetLoadTimeOut(loadtime int) *Tab {
	self.LoadTimeOut = loadtime
	return self
}

//默认为0，为0时取browser的时间
func (self *Tab) SetWaitTime(waittime int) *Tab {
	self.WaitTime = waittime
	return self
}

//获取pdf字节数组
func (self *Tab) GetPdfBytes(url string) ([]byte, error) {
	var er error
	var pdfBuffer []byte
	er = chromedp.Run(self.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuffer, _, err = page.PrintToPDF().WithPrintBackground(true).Do(ctx)
			return err
		}),
	)
	return pdfBuffer, er
}

//todo 暂时未测试
//页面截图
func (self *Tab) GetSnapShot(url string) (string, error) {
	var er error
	var rawimage string
	er = chromedp.Run(self.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			rawimage, err = page.CaptureSnapshot().Do(ctx)
			return err
		}),
	)
	return rawimage, er
}

//获取页面元素文本
func (self *Tab) GetDocument() (res []byte, err error) {
	err = chromedp.Run(self.ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			doc, er := dom.GetDocument().Do(ctx)
			if er != nil {
				return er
			}
			str, err := dom.GetOuterHTML().WithNodeID(doc.NodeID).Do(ctx)
			if err != nil {
				return err
			}
			dom.QuerySelectorAll(doc.NodeID, "script")
			res = []byte(str)
			return nil
		}),
	)
	return res, err
}

//跳转一个页面，并执行脚本，返回数据给v
func (self *Tab) NavigateEvaluate(rawUrl string, rule string, v interface{}) (err error) {
	_, err = url.Parse(rawUrl)
	if err != nil {
		return Err_LoadFail
	}
	if !strings.HasPrefix(strings.TrimSpace(rawUrl), "http") {
		rawUrl = "http://" + rawUrl
	}
	err = chromedp.Run(self.ctx, chromedp.Navigate(rawUrl))
	if err != nil {
		return err
	}
	err = chromedp.Run(self.ctx, chromedp.Evaluate(rule, &v))
	return err
}

//在当前页面执行脚本
func (self *Tab) Evaluate(rule string, v interface{}) error {
	return chromedp.Run(self.ctx, chromedp.Evaluate(rule, &v))
}

//建立一个分页对象
func (self *Tab) NewPagenation(pagerule string, spdierrule string, data interface{}) (page *pagenation, err error) {
	var p pagenation
	p.Datalist = data
	p.newtabfunc = self.browser.NewTab
	p.Pagenationrule = pagerule
	p.mtab = self
	p.SpiderRule = spdierrule
	return &p, nil
}

//跳转页面并获取各种页面信息
func (self *Tab) Navigate(rawUrl string) (doc DocumentInfo, err error) {
	_, err = url.Parse(rawUrl)
	if err != nil {
		return doc, Err_LoadFail
	}
	if !strings.HasPrefix(strings.TrimSpace(rawUrl), "http") {
		rawUrl = "http://" + rawUrl
	}
	doc = DocumentInfo{
		Resources: make(map[string]Resource, 0),
	}
	var res sync.Map

	var start = time.Now()
	//计算加载时长
	defer func() {doc.LoadTime = int(time.Since(start).Milliseconds())}()

	if self.LoadTimeOut == 0 {
		self.LoadTimeOut = Crawler_LoadTimeOut
	}
	var done = make(chan struct{})
	var documentReceived = false

	{
		var blockpatterns = make([]*fetch.RequestPattern, 0)
		if self.resourceparams.disableResource {
			if self.resourceparams.blockImage {
				blockpatterns = append(blockpatterns, &fetch.RequestPattern{
					ResourceType: network.ResourceTypeImage,
					RequestStage: "Request",
				})
			}
			if self.resourceparams.blockJs {
				blockpatterns = append(blockpatterns, &fetch.RequestPattern{
					ResourceType: network.ResourceTypeScript,
					RequestStage: "Request",
				})
			}
			if self.resourceparams.blockCss {
				blockpatterns = append(blockpatterns, &fetch.RequestPattern{
					ResourceType: network.ResourceTypeStylesheet,
					RequestStage: "Request",
				})
			}
			if self.resourceparams.blockFont {
				blockpatterns = append(blockpatterns, &fetch.RequestPattern{
					ResourceType: network.ResourceTypeFont,
					RequestStage: "Request",
				})
			}
			if self.resourceparams.blockMedia {
				blockpatterns = append(blockpatterns, &fetch.RequestPattern{
					ResourceType: network.ResourceTypeMedia,
					RequestStage: "Request",
				})
			}
		}

		var actions = make([]chromedp.Action, 0)
		if self.resourceparams.disableResource && len(blockpatterns) > 0 {
			actions = append(actions, fetch.Enable().WithPatterns(blockpatterns))
		}
		actions = append(actions, network.Enable(), chromedp.Navigate(rawUrl))
		var ctxErr error
		go func() {
			chromedp.ListenTarget(self.ctx, func(ev interface{}) {
				//退出则放弃
				ctxErr = self.ctx.Err()
				if ctxErr != nil && err == context.Canceled{
					log.Println("已退出")
					return
				}
				switch event := ev.(type) {
				case *network.EventResponseReceived:
					go func(evt *network.EventResponseReceived) {
						var resp = evt.Response
						if !documentReceived && evt.Type == network.ResourceTypeDocument {
							documentReceived = true //第一个下载成功的document为未渲染的源码
							doc.StatusCode = int(resp.Status)
							doc.Ip = resp.RemoteIPAddress   //网站IP
							doc.Port = int(resp.RemotePort) //网站端口
							doc.RespUrl = resp.URL
							if resp.Timing != nil {
								doc.DnsTime = int(resp.Timing.DNSEnd - resp.Timing.DNSStart)
								doc.ResponseTime = int(resp.Timing.ReceiveHeadersEnd)
							}
						} else {
							//页面资源类型的舍弃
							if  strings.HasPrefix(strings.ToLower(strings.TrimSpace(resp.URL)), "data:"){
								return
							}
							val, ok := res.Load(resp.URL)
							if ok {
								res.Store(resp.URL, resourceMap{
									tp:      evt.Type,
									requestID: evt.RequestID,
									referUrl: val.(resourceMap).referUrl,
								})
							}else{
								res.Store(resp.URL, resourceMap{
									tp:      evt.Type,
									requestID: evt.RequestID,
								})
							}
						}
					}(event)

					//通过fetch屏蔽资源
				case *fetch.EventRequestPaused:
					go func(evt *fetch.EventRequestPaused) {
						nctx := chromedp.FromContext(self.ctx)
						lctx := cdp.WithExecutor(self.ctx, nctx.Target)
						_ = fetch.FailRequest(evt.RequestID, network.ErrorReasonAborted).Do(lctx)
					}(event)
				case *network.EventRequestWillBeSent:
					go func(evt *network.EventRequestWillBeSent) {
						//页面资源类型,XHR异步数据的舍弃
						if  strings.HasPrefix(strings.ToLower(strings.TrimSpace(evt.Request.URL)), "data:") || evt.Type == network.ResourceTypeXHR{
							return
						}
						if self.resourceparams.disableResource {
							if self.resourceparams.blockImage && evt.Type == network.ResourceTypeImage {
								return
							}
							if self.resourceparams.blockJs  && evt.Type == network.ResourceTypeScript  {
								return
							}
							if self.resourceparams.blockCss  && evt.Type == network.ResourceTypeStylesheet {
								return
							}
							if self.resourceparams.blockFont  && evt.Type == network.ResourceTypeFont {
								return
							}
							if self.resourceparams.blockMedia  && evt.Type == network.ResourceTypeMedia {
								return
							}
						}
						var refer, ok = event.Request.Headers["Referer"]
						if ok {
							val, ok := res.Load(evt.Request.URL)
							if ok {
								res.Store(evt.Request.URL, resourceMap{
									referUrl: val.(resourceMap).referUrl,
								})
							}else{
								res.Store(evt.Request.URL, resourceMap{referUrl:refer.(string)})
							}
						}
					}(event)
				case *page.EventJavascriptDialogOpening:
					//弹窗自动关闭，不太好用，不能正确匹配确认或者取消
					go func() {
						t := page.HandleJavaScriptDialog(self.AcceptDialog)
						nctx := chromedp.FromContext(self.ctx)
						lctx := cdp.WithExecutor(self.ctx, nctx.Target)
						go func() {
							_ = chromedp.Run(lctx, t)
						}()
					}()
				}
			})
			//network.enable必须在navigate之前
			errr := chromedp.Run(self.ctx, actions...)
			if errr != nil {
				log.Println("标签执行任务失败" + errr.Error())
			}
			done <- struct{}{}
		}()
	}

	select {
	case <-time.After(time.Second * time.Duration(self.LoadTimeOut)):
		//加载失败
		if !documentReceived || doc.Ip == "" {
			return doc, Err_LoadFail
		}
		//超时
		return doc, Err_UrlTimeout
	case <-done:
		//加载失败
		if !documentReceived || doc.Ip == "" {
			return doc, Err_LoadFail
		}
		//强制等待时间
		if self.WaitTime == 0 {
			self.WaitTime = Crawler_WaitTime
		}
		time.Sleep(time.Duration(self.WaitTime) * time.Millisecond)
	}

	//为资源赋值responsebody
	nctx := chromedp.FromContext(self.ctx)
	lctx := cdp.WithExecutor(self.ctx, nctx.Target)
	err = chromedp.Run(lctx, chromedp.ActionFunc(func(ctx context.Context) error {
		//暂时不选择并行，因为有丢失的问题，当前采用单协程重试机制
		res.Range(func(key, value interface{}) bool {
			var newval = Resource{
				Type: value.(resourceMap).tp,
				Referer: value.(resourceMap).referUrl,
			}
			body, er := network.GetResponseBody(value.(resourceMap).requestID).Do(lctx)
			if er == nil && body != nil && len(body) > 0 {
				newval.Value = body
			}
			if newval.Value == nil || len(newval.Value) == 0 {
				bs, tp, er := SimpleGet(key.(string))
				if er != nil {
					log.Println(key.(string), "资源错误: ", er.Error())
				} else {
					newval.Value = bs
					newval.Type = ConvertResourceType(tp)
				}
			}
			doc.Resources[key.(string)] = newval
			return true
		})
		return nil
	}))
	return doc, err
}

// 获取当前页面的所有链接
func (self *Tab) GetAllLinks() ([]string, error) {
	var list = make([]string, 0)
	err := self.Evaluate(`
			var ls = [];
			for(i=0;i<document.links.length;i++){
				ls.push(document.links[i].href);
			}
			ls;
		`, &list)
	return list, err
}
