// Package idempotency provides retry-safe write operations backed by the
// idempotency_records table. Records are committed with the business mutation.
package idempotency

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
	"backend/internal/shared/apperr"
)

// Record is an existing idempotency record.
type Record struct {
	ID             string
	Status         string
	RequestHash    []byte
	ResourceID     *string
	ResponseStatus int
}

// Hash builds a stable request fingerprint.
func Hash(parts ...string) []byte {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return sum[:]
}

// Reserve inserts a processing record, or loads the existing one. It returns
// acquired=true when the caller owns execution. If a completed record with the
// same payload exists, its resource is returned for replay.
func Reserve(ctx context.Context, q database.Querier, scope, key string, requestHash []byte, tenantID, companyID, userID string, ttl time.Duration) (*Record, string, bool, error) {
	if key == "" {
		return nil, "", true, nil
	}
	var tenantArg, companyArg any
	if tenantID != "" {
		tenantArg = tenantID
	}
	if companyID != "" {
		companyArg = companyID
	}

	var record Record
	err := q.QueryRow(ctx, `
		INSERT INTO idempotency_records
			(tenant_id, company_id, operation_scope, idempotency_key, request_hash, status, created_by, expires_at)
		VALUES ($1,$2,$3,$4,$5,'processing',$6,$7)
		ON CONFLICT (operation_scope, idempotency_key,
			COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid),
			COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid))
		DO NOTHING
		RETURNING id, status, request_hash, resource_id, COALESCE(response_status,0)`,
		tenantArg, companyArg, scope, key, requestHash, userID, time.Now().Add(ttl)).Scan(
		&record.ID, &record.Status, &record.RequestHash, &record.ResourceID, &record.ResponseStatus)
	if err == nil {
		return &record, "", true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, "", false, apperr.Internal(err)
	}

	// Existing record.
	err = q.QueryRow(ctx, `
		SELECT id, status, request_hash, resource_id, COALESCE(response_status,0)
		FROM idempotency_records
		WHERE operation_scope=$1 AND idempotency_key=$2
		  AND COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::uuid)=COALESCE($3::uuid,'00000000-0000-0000-0000-000000000000'::uuid)
		  AND COALESCE(company_id, '00000000-0000-0000-0000-000000000000'::uuid)=COALESCE($4::uuid,'00000000-0000-0000-0000-000000000000'::uuid)
		FOR UPDATE`, scope, key, tenantArg, companyArg).Scan(
		&record.ID, &record.Status, &record.RequestHash, &record.ResourceID, &record.ResponseStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		// The record expired between statements; retry the reservation.
		return nil, "", false, apperr.Conflict("idempotency_retry", "The request could not be reserved. Please retry.")
	}
	if err != nil {
		return nil, "", false, apperr.Internal(err)
	}
	if !equal(record.RequestHash, requestHash) {
		return nil, "", false, apperr.Conflict(apperr.CodeIdempotencyMismatch,
			"This idempotency key was already used with a different request.")
	}
	if record.Status == "completed" && record.ResourceID != nil {
		return &record, *record.ResourceID, false, nil
	}
	return nil, "", false, apperr.Conflict("idempotency_in_progress",
		"A request with this idempotency key is already in progress.")
}

// Complete marks the record completed with its resource.
func Complete(ctx context.Context, q database.Querier, recordID, resourceID string, status int) error {
	if recordID == "" {
		return nil
	}
	_, err := q.Exec(ctx, `
		UPDATE idempotency_records
		SET status='completed', resource_id=$2, response_status=$3, completed_at=now(),
		    row_version=row_version+1
		WHERE id=$1`, recordID, resourceID, status)
	return err
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
