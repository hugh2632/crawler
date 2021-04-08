package crawler

import (
	"context"
	"errors"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/hugh2632/pool"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

//全局配置参数
//无头模式
var Crawler_Headless = true

//全局超时时间(秒)
var Crawler_LoadTimeOut = 30

//全局页面加载完成后等待时间(毫秒)
var Crawler_WaitTime = 0

//标签页面上限
var Crawler_Capacity int = 10

//指定缓存目录
var Crawler_CacheDirectory = ""

var Default_ResourceType_Allow = map[network.ResourceType]struct{}{network.ResourceTypeImage:struct{}{}, network.ResourceTypeScript:struct{}{}, network.ResourceTypeStylesheet:struct{}{}, network.ResourceTypeFont:struct{}{}}

var ERR_INVALID_URL error = errors.New("无效的网站")
var ERR_URL_TIMEOUT error = errors.New("网站已超时")
var ERR_URL_LOAD_FAIL error = errors.New("网站加载失败")
var ERR_INVALID_RESPONSE error = errors.New("无效的响应")

//单例对象
var _instace *browser
var locker sync.RWMutex

type browser struct {
	ctx    context.Context
	cancel context.CancelFunc
	pool.ConcurrencyPool
}

func (self browser) Close() {
	self.cancel()
}

//实例对象
func New() *browser {
	ctx, cancel := newctx()
	var res = &browser{
		ctx:    ctx,
		cancel: cancel,
	}
	res.Initial(Crawler_Capacity)
	return res
}

//单例对象
func Instance() *browser {
	if _instace == nil {
		locker.Lock()
		if _instace == nil {
			ctx, cancel := newctx()
			_instace = &browser{
				ctx:    ctx,
				cancel: cancel,
			}
			_instace.Initial(Crawler_Capacity)
		}
		locker.Unlock()
	}
	return _instace
}

func newctx() (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoDefaultBrowserCheck,
		chromedp.NoSandbox,
		chromedp.Flag("headless", Crawler_Headless),
		chromedp.Flag("ignore-certificate-errors", true),

		//无痕模式（只对浏览器有效，新的tab会新开一个不是无痕的浏览器 - -!/
		//chromedp.Flag("incognito", ""),

		//开启后可以通过devtool http/json api访问9222端口控制
		//chromedp.Flag("remote-debugging-port", "9222"),
		//chromedp.Flag("--user-data-dir", "remote-profile"),
	)
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	br, cancel := chromedp.NewContext(allocCtx)
	err := chromedp.Run(br)
	if err != nil {
		log.Fatal("无法启动Chrome,原因可能是未安装或安装在非默认位置;也有可能是浏览器崩溃了。" + err.Error())
	}
	return br, cancel
}

func (self *browser) NewTab() *Tab {
	self.Wait()
	var ers = self.ctx.Err()
	if ers != nil {
		log.Println("浏览器被关闭，强制重开一个")
		_instace = nil
		err := ClearCache()
		if err != nil {
			log.Println("清理缓存失败" + err.Error())
		}
		self = Instance()
	}
	taskCtx, cancel := chromedp.NewContext(self.ctx)
	var tab = Tab{
		ctx:     taskCtx,
		cancel:  cancel,
		browser: self,
	}
	go func() {
		<-tab.ctx.Done()
		self.Done()
	}()
	return &tab
}

//清理缓存，预防第一次爬取就304
func ClearCache() error {
	var kill *exec.Cmd
	if runtime.GOOS == "windows" {
		kill = exec.Command("taskkill", "/f", "/im", "chrome.exe")
	} else if runtime.GOOS == "linux" {
		kill = exec.Command("pkill", "chrome")
	}
	//有奇怪的错误， exit status 128
	_ = kill.Run()
	return os.RemoveAll(Crawler_CacheDirectory)
}
