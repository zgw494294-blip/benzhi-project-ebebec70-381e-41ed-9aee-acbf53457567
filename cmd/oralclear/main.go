package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"oralclear/internal/httpapi"
	"oralclear/internal/service"
	"oralclear/internal/store"
	"os"
	"strings"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19081", "监听地址")
	self := flag.Bool("selfcheck", false, "运行端到端自检")
	flag.Parse()
	*addr = listenAddress(*addr)
	dbPath := "file:oralclear.db?mode=rwc"
	if *self {
		dbPath = "file:selfcheck?mode=memory&cache=shared"
	}
	db, e := store.New(dbPath)
	if e != nil {
		panic(e)
	}
	defer db.Close()
	h := httpapi.RouteHandler(httpapi.New(service.New(db)))
	srv := &http.Server{Addr: *addr, Handler: h}
	if *self {
		go srv.ListenAndServe()
		if e := selfcheck(*addr); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		_ = srv.Shutdown(context.Background())
		return
	}
	fmt.Printf("口述史授权清理服务监听 %s\n", *addr)
	if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		panic(e)
	}
}
func selfcheck(addr string) error {
	base := "http://" + addr
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; i < 20; i++ {
		req, _ := http.NewRequest("GET", base+"/healthz", nil)
		if _, e := client.Do(req); e == nil {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	post := func(path string, body string) ([]byte, error) {
		req, _ := http.NewRequest("POST", base+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Actor", "librarian")
		res, e := client.Do(req)
		if e != nil {
			return nil, e
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			return nil, fmt.Errorf("%s 返回 %d: %s", path, res.StatusCode, b)
		}
		return b, nil
	}
	c, e := post("/api/v1/clearance-cases", `{"title":"自检访谈","collectionCode":"COL-1","consentScope":"公开姓名以外内容","policyVersion":"v1","segments":[{"sequence":1,"speakerLabel":"访谈者","originalText":"我叫张三，电话 13800138000。"}]}`)
	if e != nil {
		return e
	}
	id := extract(string(c), "caseId")
	if id == "" {
		return fmt.Errorf("创建案未返回 caseId")
	}
	head := func(path string) ([]byte, error) {
		req, _ := http.NewRequest("POST", base+path, nil)
		req.Header.Set("X-Expected-Version", "1")
		req.Header.Set("X-Actor", "librarian")
		res, e := client.Do(req)
		if e != nil {
			return nil, e
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			return nil, fmt.Errorf("%s 返回 %d: %s", path, res.StatusCode, b)
		}
		return b, nil
	}
	if _, e = head("/api/v1/clearance-cases/" + id + "/lock"); e != nil {
		return e
	}
	// scan is the second state transition and therefore expects version 2.
	reqScan, _ := http.NewRequest("POST", base+"/api/v1/clearance-cases/"+id+"/scan", nil)
	reqScan.Header.Set("X-Expected-Version", "2")
	reqScan.Header.Set("X-Actor", "librarian")
	rs, es := client.Do(reqScan)
	if es != nil {
		return es
	}
	if rs.StatusCode >= 300 {
		bb, _ := io.ReadAll(rs.Body)
		rs.Body.Close()
		return fmt.Errorf("扫描返回 %d: %s", rs.StatusCode, bb)
	}
	rs.Body.Close()
	var fs []struct {
		FindingID string `json:"findingId"`
	}
	// scan response is intentionally fetched again so selfcheck exercises GET serialization.
	req, _ := http.NewRequest("GET", base+"/api/v1/clearance-cases/"+id+"/findings", nil)
	res, e := client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(b, &fs)
	if len(fs) == 0 {
		return fmt.Errorf("扫描未产生发现项")
	}
	dec := func(fid string) error {
		body := `{"decision":"REPLACE","replacementText":"[已匿名]","rationale":"自检裁定"}`
		req, _ := http.NewRequest("POST", base+"/api/v1/clearance-cases/"+id+"/findings/"+fid, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Expected-Version", "3")
		req.Header.Set("X-Actor", "librarian")
		rr, e := client.Do(req)
		if e != nil {
			return e
		}
		defer rr.Body.Close()
		if rr.StatusCode >= 300 {
			return fmt.Errorf("裁定返回 %d", rr.StatusCode)
		}
		return nil
	}
	for _, f := range fs {
		if e := dec(f.FindingID); e != nil {
			return e
		}
	}
	postVersion := func(path, body string, expected string) ([]byte, error) {
		req, _ := http.NewRequest("POST", base+path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Expected-Version", expected)
		req.Header.Set("X-Actor", "librarian")
		rr, e := client.Do(req)
		if e != nil {
			return nil, e
		}
		defer rr.Body.Close()
		out, _ := io.ReadAll(rr.Body)
		if rr.StatusCode >= 300 {
			return nil, fmt.Errorf("%s 返回 %d: %s", path, rr.StatusCode, out)
		}
		return out, nil
	}
	cand, e := postVersion("/api/v1/clearance-cases/"+id+"/candidate", "{}", "3")
	if e != nil {
		return e
	}
	_ = cand
	if _, e = postVersion("/api/v1/clearance-cases/"+id+"/reviews", `{"reviewer":"ethics-1","decision":"APPROVE","comment":"通过"}`, "4"); e != nil {
		return e
	}
	rel, e := postVersion("/api/v1/clearance-cases/"+id+"/publish", "{}", "5")
	if e != nil {
		return e
	}
	if !strings.Contains(string(rel), "releaseId") {
		return fmt.Errorf("发布未返回 releaseId")
	}
	return nil
}
func extract(s, key string) string {
	needle := `"` + key + `":"`
	i := bytes.Index([]byte(s), []byte(needle))
	if i < 0 {
		return ""
	}
	rest := s[i+len(needle):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}
