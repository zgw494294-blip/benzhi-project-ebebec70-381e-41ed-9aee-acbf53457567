package domain

import "strings"

func ValidCategory(c string) bool {
	switch c {
	case "姓名", "联系方式", "精确地点", "受限主题":
		return true
	}
	return false
}
func CanonicalDecision(d string) string { return strings.ToUpper(strings.TrimSpace(d)) }
func DecisionNeedsText(d string) bool   { return CanonicalDecision(d) == "REPLACE" }
func DecisionIsBlocking(d string) bool  { return CanonicalDecision(d) == "PENDING" }
func AllowedDecisions() []string        { return []string{"KEEP", "DELETE", "GENERALIZE", "REPLACE"} }
