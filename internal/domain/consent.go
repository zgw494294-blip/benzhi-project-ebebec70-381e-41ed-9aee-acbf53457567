package domain

import (
	"fmt"
	"strings"
	"unicode"
)

var consentTerms = []string{"公开", "研究", "教育", "内部", "匿名"}

func ConsentAllows(scope, category string) bool {
	scope = NormalizeConsent(scope)
	if strings.Contains(scope, "匿名") && category == "姓名" {
		return false
	}
	return strings.Contains(scope, "公开") || strings.Contains(scope, "研究") || strings.Contains(scope, "教育") || strings.Contains(scope, "内部") || strings.Contains(scope, "匿名")
}

// NormalizeConsent returns a canonical, space-separated list of supported terms.
func NormalizeConsent(scope string) string {
	scope = strings.TrimSpace(scope)
	// 常见业务表述归一为标准词项。
	for _, phrase := range []string{"姓名以外内容", "姓名以外的信息", "姓名以外"} {
		scope = strings.ReplaceAll(scope, phrase, "")
	}
	for _, r := range scope {
		if unicode.IsSpace(r) || r == ',' || r == '，' || r == ';' || r == '；' || r == '/' || r == '、' {
			scope = strings.ReplaceAll(scope, string(r), " ")
		}
	}
	parts := strings.Fields(scope)
	seen := make(map[string]bool)
	var out []string
	for _, p := range parts {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

func ValidateConsentScope(scope string) (string, error) {
	normalized := NormalizeConsent(scope)
	if normalized == "" {
		return "", ValidationError{Field: "consentScope", Message: "同意范围不能为空"}
	}
	if len([]rune(normalized)) > 128 {
		return "", ValidationError{Field: "consentScope", Message: "同意范围长度不能超过 128 个字符"}
	}
	parts := strings.Fields(normalized)
	known := make(map[string]bool, len(consentTerms))
	for _, t := range consentTerms {
		known[t] = true
	}
	for _, p := range parts {
		if !known[p] {
			return "", ValidationError{Field: "consentScope", Message: fmt.Sprintf("不支持的同意范围词项：%s", p)}
		}
	}
	if len(parts) == 0 {
		return "", ValidationError{Field: "consentScope", Message: "同意范围不能为空"}
	}
	return normalized, nil
}
