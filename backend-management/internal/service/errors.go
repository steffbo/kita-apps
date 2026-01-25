package service

// ErrorCode defines a service error code.
type ErrorCode string

const (
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeConflict     ErrorCode = "CONFLICT"
)

// ServiceError represents a typed error for handlers.
type ServiceError struct {
	Code    ErrorCode
	Message string
}

func (e ServiceError) Error() string {
	return e.Message
}

func NewNotFound(message string) error {
	return ServiceError{Code: ErrCodeNotFound, Message: message}
}

func NewBadRequest(message string) error {
	return ServiceError{Code: ErrCodeBadRequest, Message: message}
}

func NewUnauthorized(message string) error {
	return ServiceError{Code: ErrCodeUnauthorized, Message: message}
}

func NewForbidden(message string) error {
	return ServiceError{Code: ErrCodeForbidden, Message: message}
}

func NewConflict(message string) error {
	return ServiceError{Code: ErrCodeConflict, Message: message}
}

// GetCode extracts the error code if available.
func GetCode(err error) ErrorCode {
	if err == nil {
		return ""
	}
	if svcErr, ok := err.(ServiceError); ok {
		return svcErr.Code
	}
	return ""
}
