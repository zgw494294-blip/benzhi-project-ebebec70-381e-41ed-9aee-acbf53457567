package domain

import "sort"

type Policy struct {
	Version           string
	AllowedCategories map[string]bool
}

func DefaultPolicy(version string) Policy {
	return Policy{Version: version, AllowedCategories: map[string]bool{"姓名": true, "联系方式": true, "精确地点": true, "受限主题": true}}
}
func (p Policy) Allows(category string) bool { return p.AllowedCategories[category] }

func (p Policy) ValidateConsent(scope string) error {
	categories := make([]string, 0, len(p.AllowedCategories))
	for category := range p.AllowedCategories {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	for _, category := range categories {
		if !ConsentAllows(scope, category) {
			return ValidationError{"consentScope." + category, "同意范围未授权该敏感类别"}
		}
	}
	return nil
}
