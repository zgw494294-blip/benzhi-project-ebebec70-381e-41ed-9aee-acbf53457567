package store

import "oralclear/internal/domain"

func validTransition(from, to domain.CaseStatus) bool { return domain.CanTransition(from, to) }
