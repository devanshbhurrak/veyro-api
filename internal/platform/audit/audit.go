// Package audit writes security-relevant events to the audit_events table.
package audit

import (
	"context"
	"encoding/json"
	"fmt"

	"backend/internal/platform/database"
)

// Event describes an auditable action. Sensitive values (passwords, raw
// tokens, full request payloads) must never be placed in Metadata.
type Event struct {
	TenantID    *string
	CompanyID   *string
	BranchID    *string
	ActorUserID *string
	ActorType   string
	Action      string
	EntityType  string
	EntityID    *string
	RequestID   string
	IPAddress   *string
	UserAgent   *string
	Changes     map[string]any
	Metadata    map[string]any
}

// Recorder persists audit events, normally inside the operation's transaction.
type Recorder struct{}

func NewRecorder() *Recorder { return &Recorder{} }

// Record inserts an audit event using the supplied querier.
func (r *Recorder) Record(ctx context.Context, q database.Querier, ev Event) error {
	if ev.ActorType == "" {
		ev.ActorType = "user"
	}
	changes, err := json.Marshal(ev.Changes)
	if err != nil {
		return fmt.Errorf("marshal audit changes: %w", err)
	}
	metadata, err := json.Marshal(ev.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	var requestID any
	if ev.RequestID != "" {
		requestID = ev.RequestID
	}
	_, err = q.Exec(ctx, `
		INSERT INTO audit_events
			(tenant_id, company_id, branch_id, actor_user_id, actor_type, action,
			 entity_type, entity_id, request_id, ip_address, user_agent, changes, metadata)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		nullIfEmpty(ev.TenantID), nullIfEmpty(ev.CompanyID), nullIfEmpty(ev.BranchID),
		nullIfEmpty(ev.ActorUserID), ev.ActorType, ev.Action, ev.EntityType,
		nullIfEmpty(ev.EntityID), requestID, nullIfEmpty(ev.IPAddress), nullIfEmpty(ev.UserAgent),
		changes, metadata,
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func nullIfEmpty(v *string) any {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}
