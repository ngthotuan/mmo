package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Newf(code int, format string, args ...any) *AppError {
	return &AppError{Code: code, Message: fmt.Sprintf(format, args...)}
}

func WithDetail(code int, message, detail string) *AppError {
	return &AppError{Code: code, Message: message, Detail: detail}
}

var (
	ErrNotFound          = New(http.StatusNotFound, "resource not found")
	ErrUnauthorized      = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden         = New(http.StatusForbidden, "forbidden")
	ErrBadRequest        = New(http.StatusBadRequest, "bad request")
	ErrConflict          = New(http.StatusConflict, "resource already exists")
	ErrInternalServer    = New(http.StatusInternalServerError, "internal server error")
	ErrInvalidToken      = New(http.StatusUnauthorized, "invalid or expired token")
	ErrInvalidCredential = New(http.StatusUnauthorized, "invalid email or password")
)

func IsNotFound(err error) bool {
	var e *AppError
	return errors.As(err, &e) && e.Code == http.StatusNotFound
}

func As(err error) (*AppError, bool) {
	var e *AppError
	return e, errors.As(err, &e)
}
