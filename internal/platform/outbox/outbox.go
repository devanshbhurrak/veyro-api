// Package outbox implements the durable transactional outbox used for
// notifications. Events are written in the same transaction as the business
// operation and delivered asynchronously by the worker.
package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"backend/internal/platform/database"
)

// Event types.
const (
	EventEmailVerification = "identity.email_verification"
	EventPasswordReset     = "identity.password_reset"
	EventInvitation        = "tenancy.invitation"
)

// Event is a durable outbox row.
type Event struct {
	ID               string
	TenantID         *string
	CompanyID        *string
	AggregateType    string
	AggregateID      string
	EventType        string
	EventVersion     int
	Payload          json.RawMessage
	DeduplicationKey string
	Status           string
	AttemptCount     int
}

// Repository persists and processes outbox events.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

// Enqueue inserts an event within the caller's transaction.
func (r *Repository) Enqueue(ctx context.Context, q database.Querier, ev Event) error {
	if ev.EventVersion == 0 {
		ev.EventVersion = 1
	}
	_, err := q.Exec(ctx, `
		INSERT INTO outbox_events
			(tenant_id, company_id, aggregate_type, aggregate_id, event_type, event_version,
			 payload, deduplication_key, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'pending')
		ON CONFLICT (deduplication_key) DO NOTHING`,
		nullIfEmpty(ev.TenantID), nullIfEmpty(ev.CompanyID), ev.AggregateType, ev.AggregateID,
		ev.EventType, ev.EventVersion, []byte(ev.Payload), ev.DeduplicationKey,
	)
	if err != nil {
		return fmt.Errorf("enqueue outbox event: %w", err)
	}
	return nil
}

// Claim locks and returns a batch of deliverable events. The caller must keep
// the transaction open until the events are marked.
func (r *Repository) Claim(ctx context.Context, tx database.Querier, workerID string, batch int, lockFor time.Duration) ([]Event, error) {
	if batch <= 0 {
		batch = 20
	}
	rows, err := tx.Query(ctx, `
		SELECT id, tenant_id, company_id, aggregate_type, aggregate_id, event_type,
		       event_version, payload, deduplication_key, status, attempt_count
		FROM outbox_events
		WHERE status = 'pending'
		  AND available_at <= now()
		  AND (locked_until IS NULL OR locked_until < now())
		ORDER BY created_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, batch)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0, batch)
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.ID, &ev.TenantID, &ev.CompanyID, &ev.AggregateType, &ev.AggregateID,
			&ev.EventType, &ev.EventVersion, &ev.Payload, &ev.DeduplicationKey, &ev.Status, &ev.AttemptCount); err != nil {
			return nil, fmt.Errorf("scan outbox event: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox events: %w", err)
	}
	if len(events) == 0 {
		return events, nil
	}

	ids := make([]string, len(events))
	for i, ev := range events {
		ids[i] = ev.ID
	}
	_, err = tx.Exec(ctx, `
		UPDATE outbox_events
		SET locked_until = now() + $2::interval, locked_by = $3,
		    attempt_count = attempt_count + 1, row_version = row_version + 1
		WHERE id = ANY($1)`, ids, lockFor.String(), workerID)
	if err != nil {
		return nil, fmt.Errorf("lock outbox events: %w", err)
	}
	return events, nil
}

// MarkProcessed marks an event delivered.
func (r *Repository) MarkProcessed(ctx context.Context, q database.Querier, id string) error {
	_, err := q.Exec(ctx, `
		UPDATE outbox_events
		SET status='processed', processed_at=now(), locked_until=NULL, locked_by=NULL,
		    last_error_code=NULL, last_error_message=NULL, row_version=row_version+1
		WHERE id=$1`, id)
	return err
}

// MarkFailed schedules a retry, or marks the event dead after maxAttempts.
func (r *Repository) MarkFailed(ctx context.Context, q database.Querier, id, code, message string, attempt, maxAttempts int) error {
	status := "pending"
	if attempt >= maxAttempts {
		status = "failed"
	}
	backoff := time.Duration(1<<min(attempt, 6)) * time.Second
	_, err := q.Exec(ctx, `
		UPDATE outbox_events
		SET status=$2, last_error_code=$3, last_error_message=$4,
		    available_at = now() + $5::interval, locked_until=NULL, locked_by=NULL,
		    row_version=row_version+1
		WHERE id=$1`, id, status, code, truncate(message, 500), backoff.String())
	return err
}

func nullIfEmpty(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
