package database

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE codes used by the application.
const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
	codeNotNullViolation    = "23502"
	codeSerialization       = "40001"
	codeDeadlock            = "40P01"
)

// PgError returns the underlying *pgconn.PgError, if any.
func PgError(err error) *pgconn.PgError {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr
	}
	return nil
}

// IsUniqueViolation reports a unique index/constraint violation.
func IsUniqueViolation(err error) bool {
	pgErr := PgError(err)
	return pgErr != nil && pgErr.Code == codeUniqueViolation
}

// IsForeignKeyViolation reports a foreign-key violation.
func IsForeignKeyViolation(err error) bool {
	pgErr := PgError(err)
	return pgErr != nil && pgErr.Code == codeForeignKeyViolation
}

// IsCheckViolation reports a check-constraint violation.
func IsCheckViolation(err error) bool {
	pgErr := PgError(err)
	return pgErr != nil && pgErr.Code == codeCheckViolation
}

// IsNotNullViolation reports a not-null violation.
func IsNotNullViolation(err error) bool {
	pgErr := PgError(err)
	return pgErr != nil && pgErr.Code == codeNotNullViolation
}

// IsRetryable reports serialization failures and deadlocks.
func IsRetryable(err error) bool {
	pgErr := PgError(err)
	if pgErr == nil {
		return false
	}
	return pgErr.Code == codeSerialization || pgErr.Code == codeDeadlock
}

// ConstraintName returns the violated constraint, when available.
func ConstraintName(err error) string {
	if pgErr := PgError(err); pgErr != nil {
		return pgErr.ConstraintName
	}
	return ""
}

// IsConstraintViolation reports any recognised integrity-constraint violation.
func IsConstraintViolation(err error) bool {
	pgErr := PgError(err)
	if pgErr == nil {
		return false
	}
	switch pgErr.Code {
	case codeUniqueViolation, codeForeignKeyViolation, codeCheckViolation, codeNotNullViolation:
		return true
	}
	return false
}

// SafeMessage returns a short, non-sensitive description of a database error
// suitable for logs. It never includes SQL or parameter values.
func SafeMessage(err error) string {
	if pgErr := PgError(err); pgErr != nil {
		return strings.TrimSpace(pgErr.Code + " " + pgErr.ConstraintName)
	}
	return "database error"
}
