package domain

type ValidationError struct{ Field, Message string }

func (e ValidationError) Error() string { return e.Field + ": " + e.Message }
func ValidateConsent(scope string) error {
	_, err := ValidateConsentScope(scope)
	return err
}
