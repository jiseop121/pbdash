package storage

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewValidationError(msg string) error {
	return &ValidationError{Message: msg}
}
