package crawler

import (
	"sync"
	"testing"
)

//并发测试，此处不适用benchmark测试
func TestTab_Navigate(t *testing.T) {
	//不使用无头模式
	Crawler_Headless = false
	//并行数量
	var num = 1
	var wg sync.WaitGroup
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func() {
			defer wg.Done()
			navigate(t)
		}()
	}
	wg.Wait()
	Instance().Close()
}

func navigate(t *testing.T) {
	var tab = Instance().NewTab()
	defer tab.Close()

	//屏蔽某些资源
	tab.DisableCrawlResource().BlockImage().BlockFont()

	doc, _ := tab.Navigate("www.baidu.com")

	text, er := tab.GetDocument()
	if er != nil {
		t.Log("获取文档失败", er.Error())
	}

	t.Log("IP:", doc.Ip)
	t.Log("端口:", doc.Port)
	t.Log("dns响应时间", doc.DnsTime)
	t.Log("响应时间:", doc.ResponseTime)
	t.Log("加载时间:", doc.LoadTime)
	t.Log("状态码:", doc.StatusCode)
	for k, v := range doc.Resources {
		t.Logf("url:%s, resource长度:%d\n", k, len(v.Value))
	}

	t.Log("\n" + string(text))
}

//分页结果实体
type Data struct {
	//标题
	Ttile string
	//链接
	Url string
	//发布日期
	Date string
}

//静态分页测试，不要过多测试，以免我背锅
func TestPagenation_RunStatic(t *testing.T) {
	var tab = Instance().NewTab()
	defer tab.Close()
	var datalist = make([]Data, 0)
	_, _ = tab.Navigate("https://www.freebuf.com/ics-articles")
	var pagerule = `for (i=1; i< 100; i++){
    var url = 'https://www.freebuf.com/ics-articles/page/' + i ;
		if(!tab.RunStatic(url)){
			break;
		}
	}`
	var spiderrule = `var res = [];
$('.news-list').each(function(){
	var tmp = {};
	tmp.Title = $(this).find('a').text();
	tmp.Url = $(this).find('a').prop('href');
	tmp.Date = $(this).find('.time').text().trim();
	res.push(tmp);
});
res;
`
	page, _ := tab.NewPagenation(pagerule, spiderrule, &datalist)
	err := page.Run()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(page.Datalist)
}

//动态脚本分页，不要过多测试，以免我背锅
func TestPagenation_RunDynic(t *testing.T) {
	var tab = Instance().NewTab()
	defer tab.Close()
	var datalist = make([]Data, 0)
	_, _ = tab.Navigate("http://its.dlut.edu.cn/wlaqy/aqgg.htm")
	var pagerule = `for (i=1; i< 100; i++){
	if(!tab.RunDynic('$(\'.Next :first\')[0].click()', 3000)){
		break;
	}
}`
	var spiderrule = `var res = [];
$('.winstyle79526 tr[height$=20]').each(function(){
	var tmp = {};
	tmp.Title = $(this).find('a').prop('title');
	tmp.Url = $(this).find('a').prop('href');
	tmp.Date = $(this).find('.timestyle79526').text().trim();
	res.push(tmp);
});
res;
`
	page, _ := tab.NewPagenation(pagerule, spiderrule, &datalist)
	err := page.Run()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(page.Datalist)
}

//获取所有链接
func TestTab_GetAllLinks(t *testing.T) {
	var browser = Instance()
	defer browser.Close()
	var tab = browser.NewTab()
	defer tab.Close()
	_, err := tab.Navigate("www.baidu.com")
	if err != nil {
		t.Fatal(err.Error())
	}
	list, err := tab.GetAllLinks()
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(list)
}
