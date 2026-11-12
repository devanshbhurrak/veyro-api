package tenancy

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
)

// Repository provides tenant and membership persistence.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

// Workspace is a tenant the caller belongs to.
type Workspace struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Status       string    `json:"status"`
	Timezone     string    `json:"timezone"`
	Locale       string    `json:"locale"`
	MembershipID string    `json:"membership_id"`
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
}

// Member is a workspace membership with user information.
type Member struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	Status    string     `json:"status"`
	JoinedAt  time.Time  `json:"joined_at"`
	RemovedAt *time.Time `json:"removed_at,omitempty"`
	Roles     []string   `json:"roles"`
}

// InsertTenantIfSlugFree inserts a tenant if the slug is unused.
func (r *Repository) InsertTenantIfSlugFree(ctx context.Context, q database.Querier, name, slug string) (string, bool, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO tenants (name, slug, status)
		VALUES ($1,$2,'active')
		ON CONFLICT (slug) DO NOTHING
		RETURNING id`, name, slug).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return id, true, nil
}

func (r *Repository) CreateMembership(ctx context.Context, q database.Querier, tenantID, userID string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO tenant_memberships (tenant_id, user_id, status, joined_at)
		VALUES ($1,$2,'active',now()) RETURNING id`, tenantID, userID).Scan(&id)
	return id, err
}

func (r *Repository) ListWorkspaces(ctx context.Context, q database.Querier, userID string) ([]Workspace, error) {
	rows, err := q.Query(ctx, `
		SELECT t.id, t.name, t.slug::text, t.status, t.timezone, t.locale, m.id, t.created_at,
		       COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL), '{}')
		FROM tenants t
		JOIN tenant_memberships m ON m.tenant_id=t.id AND m.user_id=$1
		  AND m.status='active' AND m.removed_at IS NULL
		LEFT JOIN membership_role_assignments a ON a.tenant_id=t.id AND a.membership_id=m.id
		  AND a.revoked_at IS NULL AND a.scope_type='tenant'
		LEFT JOIN roles r ON r.id=a.role_id AND r.tenant_id=t.id
		GROUP BY t.id, t.name, t.slug, t.status, t.timezone, t.locale, m.id, t.created_at
		ORDER BY t.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workspaces []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Slug, &w.Status, &w.Timezone, &w.Locale,
			&w.MembershipID, &w.CreatedAt, &w.Roles); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, rows.Err()
}

func (r *Repository) GetWorkspace(ctx context.Context, q database.Querier, tenantID, userID string) (*Workspace, error) {
	var w Workspace
	err := q.QueryRow(ctx, `
		SELECT t.id, t.name, t.slug::text, t.status, t.timezone, t.locale, m.id, t.created_at,
		       COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL), '{}')
		FROM tenants t
		JOIN tenant_memberships m ON m.tenant_id=t.id AND m.user_id=$2
		  AND m.status='active' AND m.removed_at IS NULL
		LEFT JOIN membership_role_assignments a ON a.tenant_id=t.id AND a.membership_id=m.id
		  AND a.revoked_at IS NULL AND a.scope_type='tenant'
		LEFT JOIN roles r ON r.id=a.role_id AND r.tenant_id=t.id
		WHERE t.id=$1
		GROUP BY t.id, t.name, t.slug, t.status, t.timezone, t.locale, m.id, t.created_at`,
		tenantID, userID).Scan(&w.ID, &w.Name, &w.Slug, &w.Status, &w.Timezone, &w.Locale,
		&w.MembershipID, &w.CreatedAt, &w.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repository) UpdateWorkspace(ctx context.Context, q database.Querier, tenantID, name, timezone, locale string) error {
	_, err := q.Exec(ctx, `
		UPDATE tenants SET name=$2, timezone=$3, locale=$4, updated_at=now(), row_version=row_version+1
		WHERE id=$1`, tenantID, name, timezone, locale)
	return err
}

func (r *Repository) ListMembers(ctx context.Context, q database.Querier, tenantID string) ([]Member, error) {
	rows, err := q.Query(ctx, `
		SELECT m.id, m.user_id, u.email::text, u.full_name, m.status, m.joined_at, m.removed_at,
		       COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL AND a.revoked_at IS NULL), '{}')
		FROM tenant_memberships m
		JOIN users u ON u.id=m.user_id
		LEFT JOIN membership_role_assignments a ON a.tenant_id=m.tenant_id AND a.membership_id=m.id AND a.revoked_at IS NULL
		LEFT JOIN roles r ON r.id=a.role_id AND r.tenant_id=m.tenant_id
		WHERE m.tenant_id=$1
		GROUP BY m.id, m.user_id, u.email, u.full_name, m.status, m.joined_at, m.removed_at
		ORDER BY m.joined_at`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.ID, &m.UserID, &m.Email, &m.FullName, &m.Status, &m.JoinedAt, &m.RemovedAt, &m.Roles); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *Repository) GetMember(ctx context.Context, q database.Querier, tenantID, membershipID string) (*Member, error) {
	var m Member
	err := q.QueryRow(ctx, `
		SELECT m.id, m.user_id, u.email::text, u.full_name, m.status, m.joined_at, m.removed_at,
		       COALESCE(array_agg(DISTINCT r.code) FILTER (WHERE r.code IS NOT NULL AND a.revoked_at IS NULL), '{}')
		FROM tenant_memberships m
		JOIN users u ON u.id=m.user_id
		LEFT JOIN membership_role_assignments a ON a.tenant_id=m.tenant_id AND a.membership_id=m.id AND a.revoked_at IS NULL
		LEFT JOIN roles r ON r.id=a.role_id AND r.tenant_id=m.tenant_id
		WHERE m.tenant_id=$1 AND m.id=$2
		GROUP BY m.id, m.user_id, u.email, u.full_name, m.status, m.joined_at, m.removed_at`,
		tenantID, membershipID).Scan(&m.ID, &m.UserID, &m.Email, &m.FullName, &m.Status,
		&m.JoinedAt, &m.RemovedAt, &m.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) UpdateMembershipStatus(ctx context.Context, q database.Querier, tenantID, membershipID, status string) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE tenant_memberships
		SET status=$3, removed_at=CASE WHEN $3='removed' THEN now() ELSE NULL END,
		    updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND id=$2`, tenantID, membershipID, status)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// LockTenant serializes owner-affecting membership changes for a tenant.
func (r *Repository) LockTenant(ctx context.Context, q database.Querier, tenantID string) (bool, error) {
	var id string
	err := q.QueryRow(ctx, `SELECT id FROM tenants WHERE id=$1 FOR UPDATE`, tenantID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CountOwners counts active owner memberships in the tenant.
func (r *Repository) CountOwners(ctx context.Context, q database.Querier, tenantID string) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(DISTINCT a.membership_id)
		FROM membership_role_assignments a
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id
		JOIN tenant_memberships m ON m.id=a.membership_id AND m.status='active' AND m.removed_at IS NULL
		WHERE a.tenant_id=$1 AND a.revoked_at IS NULL AND r.code='owner' AND r.is_system=true`, tenantID).Scan(&count)
	return count, err
}

func (r *Repository) MembershipHasRole(ctx context.Context, q database.Querier, tenantID, membershipID, roleCode string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM membership_role_assignments a
			JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id
			WHERE a.tenant_id=$1 AND a.membership_id=$2 AND a.revoked_at IS NULL AND r.code=$3)`,
		tenantID, membershipID, roleCode).Scan(&exists)
	return exists, err
}
