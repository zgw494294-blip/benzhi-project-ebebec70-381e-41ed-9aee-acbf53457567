package service

import (
	"oralclear/internal/domain"
	"strings"
)

func authorize(actor string, role string) error {
	if strings.TrimSpace(actor) == "" {
		return domain.ErrForbidden
	}
	switch role {
	case "librarian", "reviewer", "admin":
		return nil
	}
	return domain.ErrForbidden
}
func sameActor(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
func (s *Service) CanPublish(id, actor string) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if c.Status != domain.Approved {
		return domain.ErrState
	}
	return authorize(actor, "librarian")
}
func (s *Service) CanReview(id, actor string) error {
	c, e := s.repo.GetCase(id)
	if e != nil {
		return e
	}
	if c.Status != domain.PendingReview {
		return domain.ErrState
	}
	return authorize(actor, "reviewer")
}
