// Package apperr defines stable application errors that map to safe HTTP
// responses. Internal causes are retained for logging but never serialized.
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is a stable, client-facing application error.
type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	}
	return e.Code + ": " + e.Message
}

// Unwrap exposes the internal cause.
func (e *Error) Unwrap() error { return e.cause }

// WithCause attaches an internal cause used only for logging.
func (e *Error) WithCause(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

// WithDetails attaches safe, non-sensitive field details.
func (e *Error) WithDetails(details map[string]string) *Error {
	clone := *e
	clone.Details = details
	return &clone
}

// As returns the application error wrapped in err, if any.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

func Forbidden(code, message string) *Error {
	return New(http.StatusForbidden, code, message)
}

func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

func TooManyRequests(code, message string) *Error {
	return New(http.StatusTooManyRequests, code, message)
}

func Unprocessable(code, message string) *Error {
	return New(http.StatusUnprocessableEntity, code, message)
}

func Internal(err error) *Error {
	return New(http.StatusInternalServerError, "internal_error", "An unexpected error occurred.").WithCause(err)
}

func Unavailable(code, message string) *Error {
	return New(http.StatusServiceUnavailable, code, message)
}

// Common error codes used across modules.
const (
	CodeValidation          = "validation_failed"
	CodeUnauthenticated     = "unauthenticated"
	CodeInvalidCredentials  = "invalid_credentials"
	CodeForbidden           = "forbidden"
	CodeNotFound            = "not_found"
	CodeConflict            = "conflict"
	CodeTooManyRequests     = "too_many_requests"
	CodeInvalidToken        = "invalid_token"
	CodeExpiredToken        = "expired_token"
	CodeUsedToken           = "token_already_used"
	CodeEmailUnverified     = "email_not_verified"
	CodeAccountDisabled     = "account_disabled"
	CodeTenantSuspended     = "tenant_suspended"
	CodeIdempotencyMismatch = "idempotency_payload_mismatch"
	CodeInvalidPermission   = "permission_denied"
	CodeLastOwner           = "last_owner_protected"
	CodeEmailMismatch       = "email_mismatch"
	CodeInvalidInvitation   = "invalid_invitation"
	CodeUnsupported         = "unsupported"
)
