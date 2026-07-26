// Package errno provides standard error codes and a typed Error type used
// across all services in the TCM-History-AI backend.
package errno

import (
	"errors"
	"fmt"
	"net/http"
)

// Errno is a numeric error code.
type Errno int

const (
	// OK success.
	OK Errno = 0
	// InternalError generic server-side error.
	InternalError Errno = 10000
	// InvalidParams invalid request parameters.
	InvalidParams Errno = 10001
	// Unauthorized missing or invalid authentication.
	Unauthorized Errno = 10002
	// Forbidden access denied.
	Forbidden Errno = 10003
	// NotFound resource not found.
	NotFound Errno = 10004
	// AlreadyExists resource conflict.
	AlreadyExists Errno = 10005
	// ValidationFailed domain validation failed.
	ValidationFailed Errno = 10006
	// RateLimited request throttled.
	RateLimited Errno = 10007
	// DependencyUnavailable downstream dependency unavailable.
	DependencyUnavailable Errno = 10008
)

var messageMap = map[Errno]string{
	OK:                    "ok",
	InternalError:         "internal server error",
	InvalidParams:         "invalid parameters",
	Unauthorized:          "unauthorized",
	Forbidden:             "forbidden",
	NotFound:              "resource not found",
	AlreadyExists:         "resource already exists",
	ValidationFailed:      "validation failed",
	RateLimited:           "rate limited",
	DependencyUnavailable: "dependency unavailable",
}

// Message returns the human-readable message for the code.
func (e Errno) Message() string {
	if msg, ok := messageMap[e]; ok {
		return msg
	}
	return "unknown error"
}

// HTTPStatus maps an Errno to a sensible HTTP status code.
func (e Errno) HTTPStatus() int {
	switch e {
	case OK:
		return http.StatusOK
	case InvalidParams, ValidationFailed:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists:
		return http.StatusConflict
	case RateLimited:
		return http.StatusTooManyRequests
	case DependencyUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Error is a typed error carrying an Errno, a custom message and an optional cause.
type Error struct {
	Code    Errno
	Message string
	Cause   error
}

// New builds a new Error with the given code and message.
func New(code Errno, message string) *Error {
	if message == "" {
		message = code.Message()
	}
	return &Error{Code: code, Message: message}
}

// Wrap wraps an underlying error with a code and message.
func Wrap(code Errno, message string, cause error) *Error {
	if message == "" {
		message = code.Message()
	}
	return &Error{Code: code, Message: message, Cause: cause}
}

// Error implements error.
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("code=%d msg=%s cause=%v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("code=%d msg=%s", e.Code, e.Message)
}

// Unwrap supports errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Cause }

// From extracts an *Error from any error; returns nil if not present.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return New(InternalError, err.Error())
}

// Is reports whether target is the same Errno.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}
