package services

import "fmt"

// Refresh interval bounds shared by integrations and widgets
const (
	minRefreshSeconds = 10
	maxRefreshSeconds = 86400
)

// ValidationError marks bad user input. Its message is safe to show.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
