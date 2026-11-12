// Package platformadmin implements the deliberately small internal platform
// administration surface. It is gated by the configured admin email list.
package platformadmin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/audit"
	"backend/internal/platform/database"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

// TenantSummary is a paginated tenant row.
type TenantSummary struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	MemberCount  int       `json:"member_count"`
	CompanyCount int       `json:"company_count"`
}

// TenantDetail adds onboarding state and owner contacts.
type TenantDetail struct {
	TenantSummary
	PrimaryCompany *CompanySummary `json:"primary_company,omitempty"`
	Owners         []OwnerSummary  `json:"owners"`
}

type CompanySummary struct {
	ID        string    `json:"id"`
	TradeName string    `json:"trade_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type OwnerSummary struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
}

// Service provides read-mostly platform administration.
type Service struct {
	db    *database.Database
	audit *audit.Recorder
}

func NewService(db *database.Database, recorder *audit.Recorder) *Service {
	return &Service{db: db, audit: recorder}
}

// ListTenants returns tenants ordered by creation time.
func (s *Service) ListTenants(ctx context.Context, search, status string, limit, offset int) ([]TenantSummary, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	search = strings.TrimSpace(search)
	status = strings.ToLower(strings.TrimSpace(status))
	if len(search) > 100 {
		search = search[:100]
	}
	var total int
	if err := s.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM tenants t
		WHERE ($1='' OR t.name ILIKE '%'||$1||'%' OR t.slug::text ILIKE '%'||$1||'%')
		  AND ($2='' OR t.status=$2)`, search, status).Scan(&total); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	rows, err := s.db.Pool.Query(ctx, `
		SELECT t.id, t.name, t.slug::text, t.status, t.created_at,
		       (SELECT COUNT(*) FROM tenant_memberships m WHERE m.tenant_id=t.id
		          AND m.status='active' AND m.removed_at IS NULL),
		       (SELECT COUNT(*) FROM companies c WHERE c.tenant_id=t.id)
		FROM tenants t
		WHERE ($3='' OR t.name ILIKE '%'||$3||'%' OR t.slug::text ILIKE '%'||$3||'%')
		  AND ($4='' OR t.status=$4)
		ORDER BY t.created_at DESC LIMIT $1 OFFSET $2`, limit, offset, search, status)
	if err != nil {
		return nil, 0, apperr.Internal(err)
	}
	defer rows.Close()
	var list []TenantSummary
	for rows.Next() {
		var t TenantSummary
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt,
			&t.MemberCount, &t.CompanyCount); err != nil {
			return nil, 0, apperr.Internal(err)
		}
		list = append(list, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperr.Internal(err)
	}
	if list == nil {
		list = []TenantSummary{}
	}
	return list, total, nil
}

// GetTenant returns tenant detail with its primary company and owners.
func (s *Service) GetTenant(ctx context.Context, tenantID string) (*TenantDetail, error) {
	if !validate.UUID(tenantID) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
	}
	var detail TenantDetail
	err := s.db.Pool.QueryRow(ctx, `
		SELECT t.id, t.name, t.slug::text, t.status, t.created_at,
		       (SELECT COUNT(*) FROM tenant_memberships m WHERE m.tenant_id=t.id
		          AND m.status='active' AND m.removed_at IS NULL),
		       (SELECT COUNT(*) FROM companies c WHERE c.tenant_id=t.id)
		FROM tenants t WHERE t.id=$1`, tenantID).Scan(&detail.ID, &detail.Name, &detail.Slug,
		&detail.Status, &detail.CreatedAt, &detail.MemberCount, &detail.CompanyCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
	}
	if err != nil {
		return nil, apperr.Internal(err)
	}

	var company CompanySummary
	err = s.db.Pool.QueryRow(ctx, `
		SELECT id, trade_name, status, created_at FROM companies
		WHERE tenant_id=$1 AND is_primary=true ORDER BY created_at LIMIT 1`, tenantID).Scan(
		&company.ID, &company.TradeName, &company.Status, &company.CreatedAt)
	if err == nil {
		detail.PrimaryCompany = &company
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Internal(err)
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT u.id, u.email::text, u.full_name
		FROM tenant_memberships m
		JOIN users u ON u.id=m.user_id
		JOIN membership_role_assignments a ON a.tenant_id=m.tenant_id AND a.membership_id=m.id
		  AND a.revoked_at IS NULL AND a.scope_type='tenant'
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=m.tenant_id
		WHERE m.tenant_id=$1 AND r.code='owner' AND m.status='active' AND m.removed_at IS NULL
		GROUP BY u.id, u.email, u.full_name`, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer rows.Close()
	detail.Owners = []OwnerSummary{}
	for rows.Next() {
		var o OwnerSummary
		if err := rows.Scan(&o.UserID, &o.Email, &o.FullName); err != nil {
			return nil, apperr.Internal(err)
		}
		detail.Owners = append(detail.Owners, o)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Internal(err)
	}
	return &detail, nil
}

// UpdateTenantStatus suspends or restores a tenant. No other mutation is
// exposed deliberately.
func (s *Service) UpdateTenantStatus(ctx context.Context, tenantID, status, actorID string) error {
	if !validate.UUID(tenantID) {
		return apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "active" && status != "suspended" {
		return apperr.BadRequest(apperr.CodeValidation, "Status must be active or suspended.").
			WithDetails(map[string]string{"status": "Choose active or suspended."})
	}
	var updated bool
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE tenants SET status=$2, updated_at=now(),
			row_version=row_version+1 WHERE id=$1 AND status<>$2`, tenantID, status)
		if err != nil {
			return apperr.Internal(err)
		}
		if tag.RowsAffected() == 0 {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tenants WHERE id=$1)`, tenantID).Scan(&exists); err != nil {
				return apperr.Internal(err)
			}
			if !exists {
				return apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
			}
			return nil
		}
		updated = true
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, ActorType: "user",
			Action: "platform.tenant_status_changed", EntityType: "tenant", EntityID: &tenantID,
			Metadata: map[string]any{"status": status},
		})
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return err
		}
		return apperr.Internal(err)
	}
	_ = updated
	return nil
}

// HealthStatus is the platform health summary.
type HealthStatus struct {
	Status        string `json:"status"`
	Database      string `json:"database"`
	OutboxPending int    `json:"outbox_pending"`
	OutboxFailed  int    `json:"outbox_failed"`
}

// Health reports database reachability and outbox backlog.
func (s *Service) Health(ctx context.Context) (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	status := &HealthStatus{Status: "ok", Database: "up"}
	if err := s.db.Pool.Ping(ctx); err != nil {
		status.Status = "degraded"
		status.Database = "down"
		return status, nil
	}
	if err := s.db.Pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM outbox_events WHERE status='pending'),
		  (SELECT COUNT(*) FROM outbox_events WHERE status='failed')`).Scan(
		&status.OutboxPending, &status.OutboxFailed); err != nil {
		status.Status = "degraded"
		return status, nil
	}
	if status.OutboxFailed > 0 {
		status.Status = "degraded"
	}
	return status, nil
}
