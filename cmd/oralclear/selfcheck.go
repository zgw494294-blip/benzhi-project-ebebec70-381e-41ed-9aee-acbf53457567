package main

import (
	"fmt"
	"net/http"
	"time"
)

func waitForHTTP(client *http.Client, url string, attempts int) error {
	for i := 0; i < attempts; i++ {
		r, e := client.Get(url)
		if e == nil {
			r.Body.Close()
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("服务未在预期时间内启动")
}
func checkStatus(client *http.Client, url string, want int) error {
	r, e := client.Get(url)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode != want {
		return fmt.Errorf("%s 返回 %d，期望 %d", url, r.StatusCode, want)
	}
	return nil
}
