package errors

import "errors"

// Common application errors for type-safe error handling.
// These errors can be checked using errors.Is() instead of string comparison.
var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInternal     = errors.New("internal server error")
)
