package crawler

import (
	"errors"
	"github.com/robertkrimen/otto"
	"github.com/robertkrimen/otto/parser"
	"log"
	"reflect"
	"strconv"
	"time"
)

type pagenation struct {
	mtab           *Tab
	Pagenationrule string
	SpiderRule     string
	Datalist       interface{}
	newtabfunc     func() *Tab
}

func NewPagenation(tab *Tab, pagerule string, spdierrule string, data interface{}, f func() *Tab) (*pagenation, error) {
	var p pagenation
	p.Datalist = data
	p.newtabfunc = f
	p.Pagenationrule = pagerule
	p.mtab = tab
	p.SpiderRule = spdierrule
	return &p, nil
}

func (p *pagenation) Run() (err error) {
	var vm = otto.New()
	_ = vm.Set("tab", p)
	_, err = vm.Run(p.Pagenationrule)
	if err != nil {
		val, ok := err.(parser.ErrorList)
		if ok {
			var errMsg = ""
			for _, vv := range val {
				errMsg += "第" + strconv.Itoa(vv.Position.Line) + "第" + strconv.Itoa(vv.Position.Column) + "列分页规则有错误,信息:" + vv.Message + "\n"
			}
			return errors.New(errMsg)
		}
		return errors.New("分页规则有错误!" + err.Error())
	}
	return nil
}

//动态分页,pagescript:动态触发规则（可以是js或者jquery等等当前页面支持的脚本）
func (p *pagenation) RunDynic(pagescript string, millisecond int) bool {
	err := p.mtab.Evaluate(pagescript, nil)
	if err != nil && err.Error() != "encountered an undefined value" {
		return false
	}
	time.Sleep(time.Duration(millisecond) * time.Millisecond)
	var newobj = reflect.New(reflect.TypeOf(p.Datalist).Elem()).Interface()
	err = p.mtab.Evaluate(p.SpiderRule, newobj)
	if err != nil {
		log.Println("执行脚本失败或超过分页范围," + err.Error())
		return false
	}
	if newobj == nil || reflect.ValueOf(newobj).Elem().Len() == 0 {
		//未取到数据，已经到底或者错误
		return false
	}
	reflect.ValueOf(p.Datalist).Elem().Set(reflect.AppendSlice(reflect.ValueOf(p.Datalist).Elem(), reflect.ValueOf(newobj).Elem()))
	return true
}

//静态分页
func (p *pagenation) RunStatic(url string) bool {
	var tab = p.newtabfunc()
	defer tab.Close()
	var newobj = reflect.New(reflect.TypeOf(p.Datalist).Elem()).Interface()
	err := tab.NavigateEvaluate(url, p.SpiderRule, newobj)
	if err != nil && err.Error() != "encountered an undefined value" {
		log.Println(p.SpiderRule + "执行脚本失败," + err.Error())
		return false
	}
	if newobj == nil || reflect.ValueOf(newobj).Elem().Len() == 0 {
		//未取到数据，已经到底或者错误
		return false
	}
	reflect.ValueOf(p.Datalist).Elem().Set(reflect.AppendSlice(reflect.ValueOf(p.Datalist).Elem(), reflect.ValueOf(newobj).Elem()))
	return true
}

//并行静态分页，调用时需要根据分页规则的数量加上waitgroup
func (p *pagenation) RunStaticParallel(url string) {
	go func() {
		var tab = p.newtabfunc()
		defer tab.Close()
		var newobj = reflect.New(reflect.TypeOf(p.Datalist).Elem()).Interface()
		err := tab.NavigateEvaluate(url, p.SpiderRule, newobj)
		if err != nil && err.Error() != "encountered an undefined value" {
			log.Println(p.SpiderRule + "执行脚本失败," + err.Error())
			return
		}
		if newobj == nil || reflect.ValueOf(newobj).Elem().Len() == 0 {
			//未取到数据，已经到底或者错误
			return
		}
		reflect.ValueOf(p.Datalist).Elem().Set(reflect.AppendSlice(reflect.ValueOf(p.Datalist).Elem(), reflect.ValueOf(newobj).Elem()))
	}()
}
