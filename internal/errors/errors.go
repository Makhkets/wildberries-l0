package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Базовые типы ошибок
var (
	// Repository layer errors
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrDatabaseError = errors.New("database error")
	ErrInvalidData   = errors.New("invalid data")

	// Service layer errors
	ErrValidation   = errors.New("validation error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")

	// External service errors
	ErrExternalAPI = errors.New("external api error")
	ErrTimeout     = errors.New("timeout error")
)

// AppError представляет структурированную ошибку приложения
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
	Internal   error     `json:"-"`
}

// ErrorType определяет тип ошибки
type ErrorType string

const (
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeValidation   ErrorType = "VALIDATION"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeInternal     ErrorType = "INTERNAL"
	ErrorTypeExternalAPI  ErrorType = "EXTERNAL_API"
	ErrorTypeTimeout      ErrorType = "TIMEOUT"
)

// Error реализует интерфейс error
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// Unwrap позволяет использовать errors.Is и errors.As
func (e *AppError) Unwrap() error {
	return e.Internal
}

// NewAppError создает новую структурированную ошибку
func NewAppError(errType ErrorType, message string) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: getStatusCodeByType(errType),
	}
}

// NewAppErrorWithDetails создает ошибку с дополнительными деталями
func NewAppErrorWithDetails(errType ErrorType, message, details string) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		Details:    details,
		StatusCode: getStatusCodeByType(errType),
	}
}

// WrapError оборачивает внутреннюю ошибку в AppError
func WrapError(errType ErrorType, message string, internal error) *AppError {
	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: getStatusCodeByType(errType),
		Internal:   internal,
	}
}

// getStatusCodeByType возвращает HTTP статус код по типу ошибки
func getStatusCodeByType(errType ErrorType) int {
	switch errType {
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeValidation:
		return http.StatusBadRequest
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeTimeout:
		return http.StatusRequestTimeout
	case ErrorTypeExternalAPI:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// Repository layer error helpers
func NewNotFoundError(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource))
}

func NewValidationError(field, reason string) *AppError {
	return NewAppErrorWithDetails(ErrorTypeValidation, "Validation failed", fmt.Sprintf("Field '%s': %s", field, reason))
}

func NewDatabaseError(operation string, internal error) *AppError {
	return WrapError(ErrorTypeInternal, fmt.Sprintf("Database operation failed: %s", operation), internal)
}

// NewUnauthorizedError Service layer error helpers
func NewUnauthorizedError(reason string) *AppError {
	return NewAppErrorWithDetails(ErrorTypeUnauthorized, "Unauthorized access", reason)
}

func NewForbiddenError(reason string) *AppError {
	return NewAppErrorWithDetails(ErrorTypeForbidden, "Access forbidden", reason)
}

func NewConflictError(resource string) *AppError {
	return NewAppError(ErrorTypeConflict, fmt.Sprintf("%s already exists", resource))
}

// IsErrorType проверяет, является ли ошибка определенного типа
func IsErrorType(err error, errType ErrorType) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Type == errType
	}
	return false
}

// GetStatusCode возвращает HTTP статус код из ошибки
func GetStatusCode(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.StatusCode
	}
	return http.StatusInternalServerError
}

// IsAppError проверяет, является ли ошибка типом AppError и извлекает её
func IsAppError(err error, appErr **AppError) bool {
	return errors.As(err, appErr)
}
