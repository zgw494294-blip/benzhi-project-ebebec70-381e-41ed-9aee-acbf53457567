package httpapi

import (
	"fmt"
	"net/http"
	"strings"
)

func requireMethod(r *http.Request, method string) error {
	if r.Method != method {
		return fmt.Errorf("仅支持 %s", method)
	}
	return nil
}
func requireHeader(r *http.Request, name string) error {
	if strings.TrimSpace(r.Header.Get(name)) == "" {
		return fmt.Errorf("缺少请求头 %s", name)
	}
	return nil
}
func maxBody(r *http.Request, n int64) error {
	if r.ContentLength > n {
		return fmt.Errorf("请求体不能超过 %d 字节", n)
	}
	return nil
}
