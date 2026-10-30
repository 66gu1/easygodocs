package apperr

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

const (
	CodeBadRequest   Code = "core/bad_request"
	CodeUnauthorized Code = "core/unauthorized"
	CodeForbidden    Code = "core/forbidden"
	CodeInternal     Code = "core/internal_error"
	CodeRateLimited  Code = "core/rate_limited"
)

const (
	BadRequestMsg   = "Bad request"
	UnauthorizedMsg = "Unauthorized"
	ForbiddenMsg    = "Forbidden"
	InternalMsg     = "Internal server error"
	RateLimitedMsg  = "Too many requests"
)

func ErrBadRequest() *AppError {
	return &AppError{
		Message:  BadRequestMsg,
		Code:     CodeBadRequest,
		class:    ClassBadRequest,
		logLevel: LogLevelWarn,
		detail:   BadRequestMsg,
	}
}

func ErrUnauthorized() *AppError {
	return &AppError{
		Message:  UnauthorizedMsg,
		Code:     CodeUnauthorized,
		class:    ClassUnauthorized,
		logLevel: LogLevelWarn,
		detail:   UnauthorizedMsg,
	}
}

func ErrForbidden() *AppError {
	return &AppError{
		Message:  ForbiddenMsg,
		Code:     CodeForbidden,
		class:    ClassForbidden,
		logLevel: LogLevelWarn,
		detail:   ForbiddenMsg,
	}
}

func ErrRateLimited() *AppError {
	return &AppError{
		Message:  RateLimitedMsg,
		Code:     CodeRateLimited,
		class:    ClassTooManyRequests,
		logLevel: LogLevelWarn,
		detail:   RateLimitedMsg,
	}
}

func ErrNilUUID(field Field) *AppError {
	return &AppError{
		Message:  ErrBadRequest().Error(),
		Code:     CodeBadRequest,
		class:    ClassBadRequest,
		logLevel: LogLevelWarn,
		detail:   fmt.Sprintf("%s cannot be nil", field.String()),
	}
}

// AppError is used for all application-level errors that should be shown to the user (e.g. 400, 401, 403).
// For internal server errors (500), use fmt.Errorf and handle them separately to avoid exposing internal details to the client.
type AppError struct {
	Message    string      `json:"message"` // Message for user
	Code       Code        `json:"code"`
	Violations []Violation `json:"violations,omitempty"`
	class      Class
	logLevel   LogLevel
	detail     string        // detail for logs
	retryAfter time.Duration // sent as the Retry-After header
}

func New(message string, code Code, class Class, logLevel LogLevel) *AppError {
	return &AppError{
		Message:  message,
		class:    class,
		logLevel: logLevel,
		Code:     code,
		detail:   message,
	}
}

// The With* methods return a modified copy, so a shared error value is never mutated.

func (e *AppError) WithUserMessage(message string) *AppError {
	c := e.clone()
	c.Message = message
	return c
}

func (e *AppError) WithDetail(detail string) *AppError {
	c := e.clone()
	c.detail = detail
	return c
}

func (e *AppError) WithViolation(v Violation) *AppError {
	c := e.clone()
	c.Violations = append(c.Violations, v)
	return c
}

// WithRetryAfter tells the client when to retry, e.g. after a rate limit.
func (e *AppError) WithRetryAfter(d time.Duration) *AppError {
	c := e.clone()
	c.retryAfter = d
	return c
}

func (e *AppError) RetryAfter() time.Duration {
	return e.retryAfter
}

func (e *AppError) clone() *AppError {
	c := *e
	c.Violations = slices.Clone(e.Violations)
	return &c
}

func (e *AppError) Error() string {
	return e.detail
}

func (e *AppError) Is(target error) bool {
	if t, ok := target.(*AppError); ok {
		if e.Code != t.Code {
			return false
		}

		return slices.EqualFunc(e.Violations, t.Violations, func(a, b Violation) bool {
			return a.Field == b.Field && a.Rule == b.Rule
		})
	}

	return false
}

type Violation struct {
	Field  Field          `json:"field"`
	Rule   Rule           `json:"rule"`
	Params map[string]any `json:"params,omitempty"`
}

type Field string

func (f Field) String() string { return string(f) }

const (
	FieldRequest Field = "request"
)

type Code string

type Class uint8

const (
	ClassInternal        Class = 1
	ClassBadRequest      Class = 2
	ClassNotFound        Class = 3
	ClassUnauthorized    Class = 4
	ClassForbidden       Class = 5
	ClassConflict        Class = 6
	ClassTooManyRequests Class = 7
)

type LogLevel int

const (
	LogLevelError LogLevel = 0
	LogLevelWarn  LogLevel = 1
)

func ClassOf(err error) Class {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.class
	}
	return ClassInternal
}

func CodeOf(err error) Code {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.Code
	}
	return CodeInternal
}

func LogLevelOf(err error) LogLevel {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae.logLevel
	}
	return LogLevelError
}

func FromError(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return &AppError{
		Message:  "Internal server error",
		Code:     CodeInternal,
		class:    ClassInternal,
		logLevel: LogLevelError,
		detail:   err.Error(),
	}
}
