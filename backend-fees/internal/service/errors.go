package service

import "errors"

// Common service errors
var (
	ErrNotFound              = errors.New("not found")
	ErrDuplicateMemberNumber = errors.New("duplicate member number")
	ErrInvalidInput          = errors.New("invalid input")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrConflict              = errors.New("conflict")
)
