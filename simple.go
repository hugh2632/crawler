package crawler

import (
	"io/ioutil"
	"net/http"
)

func SimpleGet(url string) (res []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if IsValidStatus(resp.StatusCode) {
		if resp.Body != nil {
			return ioutil.ReadAll(resp.Body)
		}
	}
	return nil, Err_InValidResponse
}

func IsValidStatus(statuscode int) bool {
	if statuscode >= 200 && statuscode < 300 {
		return true
	}
	return false
}
