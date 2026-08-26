package service

import (
	"oralclear/internal/domain"
	"strings"
)

func reviewIndependent(reviewer string, recent []domain.Review) error {
	for _, r := range recent {
		if strings.EqualFold(r.Reviewer, reviewer) {
			return domain.ErrForbidden
		}
	}
	return nil
}

func (s *Service) Reviews(id string) ([]domain.Review, error) {
	if _, err := s.repo.GetCase(id); err != nil {
		return nil, err
	}
	return s.repo.ListReviews(id)
}
