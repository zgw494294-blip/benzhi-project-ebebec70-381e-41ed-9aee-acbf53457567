package service

import (
	"fmt"
	"oralclear/internal/domain"
)

func CheckPolicy(c *domain.ClearanceCase) error {
	if c.PolicyVersion == "" {
		return fmt.Errorf("缺少策略版本")
	}
	if c.ConsentScope == "" {
		return fmt.Errorf("缺少同意范围")
	}
	return nil
}
func (s *Service) Policy(id string) (domain.Policy, error) {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return domain.Policy{}, e
	}
	if e = CheckPolicy(c); e != nil {
		return domain.Policy{}, e
	}
	return domain.DefaultPolicy(c.PolicyVersion), nil
}
