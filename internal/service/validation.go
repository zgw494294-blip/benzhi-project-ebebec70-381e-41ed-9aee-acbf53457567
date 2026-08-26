package service

import (
	"oralclear/internal/domain"
	"strings"
)

func validateActor(actor string) error {
	if strings.TrimSpace(actor) == "" {
		return domain.ErrInvalid
	}
	return nil
}
func validateExpected(actual, expected int64) error {
	if expected > 0 && actual != expected {
		return domain.ErrConflict
	}
	return nil
}
