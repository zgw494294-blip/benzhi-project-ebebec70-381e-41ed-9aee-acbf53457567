package httpapi

import "net/http"

type Route struct{ Method, Path, Description string }

func Routes() []Route {
	return []Route{{"POST", "/api/v1/clearance-cases", "创建授权清理案"}, {"POST", "/api/v1/clearance-cases/{caseId}/segments", "登记片段"}, {"POST", "/api/v1/clearance-cases/{caseId}/lock", "锁定策略"}, {"POST", "/api/v1/clearance-cases/{caseId}/scan", "扫描敏感项"}, {"GET", "/api/v1/clearance-cases/{caseId}/findings", "查询发现项"}, {"POST", "/api/v1/clearance-cases/{caseId}/findings/batch", "批量裁定发现项"}, {"POST", "/api/v1/clearance-cases/{caseId}/candidate", "生成候选稿"}, {"POST", "/api/v1/clearance-cases/{caseId}/reviews", "提交复核"}, {"GET", "/api/v1/clearance-cases/{caseId}/reviews", "查询复核审计"}, {"POST", "/api/v1/clearance-cases/{caseId}/publish", "发布版本"}, {"GET", "/api/v1/releases/{releaseId}", "查询发布版本"}, {"GET", "/api/v1/releases?contentDigest=...", "摘要验证"}}
}
func RouteHandler(h http.Handler) http.Handler { return withRecovery(h) }
