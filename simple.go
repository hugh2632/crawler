package crawler

import (
	"errors"
	"github.com/chromedp/cdproto/network"
	"github.com/hugh2632/pool"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var simplepool pool.ConcurrencyPool
var client = &http.Client{}

func init(){
	simplepool.Initial(50)
	client.Timeout = 10 * time.Second
}

func SimpleGet(url string) (res []byte, contentType string, statuscode int, err error) {
	simplepool.Wait()
	defer simplepool.Done()
	req ,_:=http.NewRequest("GET",url,nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0,  err
	}
	if IsValidStatus(resp.StatusCode) {
		if resp.Body != nil {
			res, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, "", resp.StatusCode, err
			}
			//contentType = http.DetectContentType(res)//内置包有错误
			content, ok := resp.Header["Content-Type"]
			if !ok {
				return nil, "", resp.StatusCode,errors.New("没有contentType")
			}
			return res, strings.Join(content, ";"), resp.StatusCode,nil
		}
	}
	return nil, "", resp.StatusCode, ERR_INVALID_RESPONSE
}

func ConvertResourceType(contentType string) network.ResourceType {
	var list = strings.Split(contentType, ";")
	for i, _ := range list {
		var tpStr = strings.ToLower(strings.TrimSpace(list[i]))
		if strings.HasPrefix(tpStr, "image/") || tpStr == "application/x-ico" || tpStr == "application/x-jpe" || tpStr == "application/x-png" || tpStr == "application/x-tif" {
			return network.ResourceTypeImage
		} else if tpStr == "text/html"{
			return network.ResourceTypeDocument
		} else if tpStr == "text/css" {
			return network.ResourceTypeStylesheet
		} else if tpStr == "application/javascript" || tpStr ==  "text/javascript" || tpStr ==  "text/ecmascript" ||  tpStr == "text/jscript" || tpStr ==  "text/vbscript" {
			return network.ResourceTypeScript
		}
		//todo 更多类型转换，将来补充
	}
	return network.ResourceTypeOther
}

func IsValidStatus(statuscode int) bool {
	if statuscode == 304 || (statuscode >= 200 && statuscode < 300) {
		return true
	}
	return false
}
