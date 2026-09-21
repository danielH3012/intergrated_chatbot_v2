package errs

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// AppError is an HTTP error entity mirroring the coreEntity pattern in modules.
type AppError struct {
	StatusCode int    `json:"code"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Details    any    `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

// SendResponse transmits the HTTP error payload directly to the Fiber client.
func (e *AppError) SendResponse(c *fiber.Ctx) error {
	return c.Status(e.StatusCode).JSON(fiber.Map{
		"code":    e.StatusCode,
		"status":  "error",
		"message": e.Message,
		"details": e.Details,
		"error":   e.Message,
	})
}

// New creates a custom AppError with an HTTP status code.
func New(statusCode int, message string, details ...any) *AppError {
	var det any = nil
	if len(details) > 0 {
		det = details[0]
	}
	return &AppError{
		StatusCode: statusCode,
		Status:     "error",
		Message:    message,
		Details:    det,
	}
}

// BadRequest returns 400 Bad Request error.
func BadRequest(message string, details ...any) *AppError {
	return New(http.StatusBadRequest, message, details...)
}

// Unauthorized returns 401 Unauthorized error.
func Unauthorized(message string, details ...any) *AppError {
	return New(http.StatusUnauthorized, message, details...)
}

// Forbidden returns 403 Forbidden error.
func Forbidden(message string, details ...any) *AppError {
	return New(http.StatusForbidden, message, details...)
}

// NotFound returns 404 Not Found error.
func NotFound(message string, details ...any) *AppError {
	return New(http.StatusNotFound, message, details...)
}

// Conflict returns 409 Conflict error.
func Conflict(message string, details ...any) *AppError {
	return New(http.StatusConflict, message, details...)
}

// InternalServerError returns 500 Internal Server Error.
func InternalServerError(message string, details ...any) *AppError {
	return New(http.StatusInternalServerError, message, details...)
}

// ToHttpError converts any standard Go error into an AppError entity.
func ToHttpError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return InternalServerError(err.Error())
}
