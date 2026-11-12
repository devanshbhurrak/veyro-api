package invitations

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
)

// Repository provides invitation persistence.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

const invitationColumns = `i.id, i.email::text, i.default_role_id, r.code, r.name, i.status,
	i.message, i.expires_at, i.accepted_at, i.revoked_at, i.invited_by,
	COALESCE(u.full_name, ''), i.created_at`

const invitationFrom = `FROM tenant_invitations i
	JOIN roles r ON r.id=i.default_role_id AND r.tenant_id=i.tenant_id
	LEFT JOIN users u ON u.id=i.invited_by`

func scanInvitation(row pgx.Row) (*Invitation, error) {
	var inv Invitation
	err := row.Scan(&inv.ID, &inv.Email, &inv.RoleID, &inv.RoleCode, &inv.RoleName, &inv.Status,
		&inv.Message, &inv.ExpiresAt, &inv.AcceptedAt, &inv.RevokedAt, &inv.InvitedBy,
		&inv.InviterName, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repository) List(ctx context.Context, q database.Querier, tenantID string) ([]Invitation, error) {
	rows, err := q.Query(ctx, `SELECT `+invitationColumns+` `+invitationFrom+`
		WHERE i.tenant_id=$1 ORDER BY i.created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Invitation
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *inv)
	}
	return list, rows.Err()
}

func (r *Repository) Get(ctx context.Context, q database.Querier, tenantID, id string) (*Invitation, error) {
	return scanInvitation(q.QueryRow(ctx, `SELECT `+invitationColumns+` `+invitationFrom+`
		WHERE i.tenant_id=$1 AND i.id=$2`, tenantID, id))
}

// ExpireStale marks pending invitations past their expiry so a new invitation
// can be issued for the same address.
func (r *Repository) ExpireStale(ctx context.Context, q database.Querier, tenantID, email string) error {
	_, err := q.Exec(ctx, `
		UPDATE tenant_invitations
		SET status='expired', updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND email=$2 AND status='pending' AND expires_at <= now()`,
		tenantID, email)
	return err
}

// ExpireStaleAll expires pending invitations for a tenant.
func (r *Repository) ExpireStaleAll(ctx context.Context, q database.Querier, tenantID string) error {
	_, err := q.Exec(ctx, `
		UPDATE tenant_invitations
		SET status='expired', updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND status='pending' AND expires_at <= now()`, tenantID)
	return err
}

func (r *Repository) Create(ctx context.Context, q database.Querier, tenantID, email, roleID, invitedBy string, message *string, tokenHash []byte, expiresAt time.Time) (*Invitation, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO tenant_invitations
			(tenant_id, email, default_role_id, invited_by, token_hash, message, status, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,'pending',$7)
		RETURNING id`,
		tenantID, email, roleID, invitedBy, tokenHash, message, expiresAt).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, q, tenantID, id)
}

// GetPendingByToken locks a pending invitation for acceptance.
func (r *Repository) GetPendingByToken(ctx context.Context, q database.Querier, tokenHash []byte) (*PendingInvitation, error) {
	var inv PendingInvitation
	err := q.QueryRow(ctx, `
		SELECT i.id, i.tenant_id, t.name, t.status, i.email::text, i.default_role_id, r.code,
		       i.status, i.expires_at
		FROM tenant_invitations i
		JOIN tenants t ON t.id=i.tenant_id
		JOIN roles r ON r.id=i.default_role_id AND r.tenant_id=i.tenant_id
		WHERE i.token_hash=$1 AND i.status='pending'
		FOR UPDATE OF i`, tokenHash).Scan(&inv.ID, &inv.TenantID, &inv.TenantName,
		&inv.TenantStatus, &inv.Email, &inv.RoleID, &inv.RoleCode, &inv.Status, &inv.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// Refresh rotates the token of a pending invitation.
func (r *Repository) Refresh(ctx context.Context, q database.Querier, tenantID, id string, tokenHash []byte, expiresAt time.Time) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE tenant_invitations
		SET token_hash=$3, expires_at=$4, updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND id=$2 AND status='pending'`,
		tenantID, id, tokenHash, expiresAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) Revoke(ctx context.Context, q database.Querier, tenantID, id, actorID string) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE tenant_invitations
		SET status='revoked', revoked_at=now(), revoked_by=$3, updated_at=now(),
		    row_version=row_version+1
		WHERE tenant_id=$1 AND id=$2 AND status='pending'`, tenantID, id, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) MarkAccepted(ctx context.Context, q database.Querier, tenantID, id, userID string) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE tenant_invitations
		SET status='accepted', accepted_at=now(), accepted_by=$3, updated_at=now(),
		    row_version=row_version+1
		WHERE tenant_id=$1 AND id=$2 AND status='pending'`, tenantID, id, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// MembershipStatus reports whether a user with the email already belongs.
func (r *Repository) MembershipStatus(ctx context.Context, q database.Querier, tenantID, email string) (string, bool, error) {
	var status string
	err := q.QueryRow(ctx, `
		SELECT m.status FROM tenant_memberships m
		JOIN users u ON u.id=m.user_id
		WHERE m.tenant_id=$1 AND u.email=$2
		LIMIT 1`, tenantID, email).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return status, true, nil
}
