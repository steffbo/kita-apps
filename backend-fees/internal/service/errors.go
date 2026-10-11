package service

import (
	"errors"
	"fmt"
)

// Common service errors
var (
	ErrNotFound              = errors.New("not found")
	ErrDuplicateMemberNumber = errors.New("duplicate member number")
	ErrInvalidInput          = errors.New("invalid input")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrConflict              = errors.New("conflict")
	ErrAlreadyExists         = errors.New("already exists")
	ErrHouseholdMismatch     = errors.New("child and parent belong to different households")
	ErrEmailDisabled         = errors.New("email service disabled")
)

// ValidationError is invalid input with a message for the user. It matches
// ErrInvalidInput with errors.Is.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// Is lets errors.Is(err, ErrInvalidInput) match.
func (e *ValidationError) Is(target error) bool { return target == ErrInvalidInput }

func newValidationError(format string, args ...interface{}) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
