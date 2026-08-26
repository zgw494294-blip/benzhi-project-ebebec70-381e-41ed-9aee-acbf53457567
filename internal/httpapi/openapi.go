package httpapi

import "encoding/json"

type APIDocument struct {
	Version string  `json:"version"`
	Title   string  `json:"title"`
	Routes  []Route `json:"routes"`
}

func Document() APIDocument {
	return APIDocument{Version: "v1", Title: "口述史授权清理服务", Routes: Routes()}
}
func DocumentJSON() ([]byte, error) { return json.Marshal(Document()) }
func routeMethods() map[string][]string {
	m := map[string][]string{}
	for _, r := range Routes() {
		m[r.Path] = append(m[r.Path], r.Method)
	}
	return m
}
func hasRoute(path, method string) bool {
	for _, r := range Routes() {
		if r.Path == path && r.Method == method {
			return true
		}
	}
	return false
}
