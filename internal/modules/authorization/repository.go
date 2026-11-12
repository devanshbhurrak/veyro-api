package authorization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
)

// Role is a tenant role.
type Role struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsSystem    bool      `json:"is_system"`
	Status      string    `json:"status"`
	Permissions []string  `json:"permissions,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Permission is a platform permission.
type Permission struct {
	ID          string  `json:"id"`
	Code        string  `json:"code"`
	Module      string  `json:"module"`
	Action      string  `json:"action"`
	Description *string `json:"description,omitempty"`
}

// MembershipContext is the trusted resolved scope for a request.
type MembershipContext struct {
	MembershipID string
	TenantStatus string
	Permissions  map[string]struct{}
}

// Repository provides authorization persistence.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

// ResolveMembership loads an active membership and tenant status.
func (r *Repository) ResolveMembership(ctx context.Context, q database.Querier, tenantID, userID string) (*MembershipContext, error) {
	var mc MembershipContext
	mc.Permissions = make(map[string]struct{})
	err := q.QueryRow(ctx, `
		SELECT m.id, t.status
		FROM tenant_memberships m
		JOIN tenants t ON t.id = m.tenant_id
		WHERE m.tenant_id=$1 AND m.user_id=$2
		  AND m.status='active' AND m.removed_at IS NULL`, tenantID, userID).Scan(&mc.MembershipID, &mc.TenantStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &mc, nil
}

// ResolvePermissions loads effective permissions for a membership at the given
// company/branch scope. Tenant-scoped assignments always apply.
func (r *Repository) ResolvePermissions(ctx context.Context, q database.Querier, tenantID, membershipID, companyID, branchID string) (map[string]struct{}, error) {
	var companyArg, branchArg any
	if companyID != "" {
		companyArg = companyID
	}
	if branchID != "" {
		branchArg = branchID
	}
	rows, err := q.Query(ctx, `
		SELECT DISTINCT p.code
		FROM membership_role_assignments a
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id AND r.status='active'
		JOIN role_permissions rp ON rp.role_id=a.role_id AND rp.tenant_id=a.tenant_id
		JOIN permissions p ON p.id=rp.permission_id
		WHERE a.tenant_id=$1 AND a.membership_id=$2 AND a.revoked_at IS NULL
		  AND (
		    a.scope_type='tenant'
		    OR (a.scope_type='company' AND $3::uuid IS NOT NULL AND a.company_id=$3)
		    OR (a.scope_type='branch' AND $4::uuid IS NOT NULL AND a.branch_id=$4)
		  )`, tenantID, membershipID, companyArg, branchArg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := make(map[string]struct{})
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions[code] = struct{}{}
	}
	return permissions, rows.Err()
}

// SeedDefaultRoles creates the system roles and permission mappings for a new
// tenant. It returns the Owner role ID.
func (r *Repository) SeedDefaultRoles(ctx context.Context, q database.Querier, tenantID string) (string, error) {
	var ownerRoleID string
	for _, role := range DefaultRoles {
		var roleID string
		err := q.QueryRow(ctx, `
			INSERT INTO roles (tenant_id, code, name, description, is_system, status)
			VALUES ($1,$2,$3,$4,true,'active')
			ON CONFLICT (tenant_id, code) DO UPDATE SET name=EXCLUDED.name
			RETURNING id`, tenantID, role.Code, role.Name, role.Description).Scan(&roleID)
		if err != nil {
			return "", fmt.Errorf("seed role %s: %w", role.Code, err)
		}
		if role.Code == RoleOwner {
			ownerRoleID = roleID
		}
		_, err = q.Exec(ctx, `
			INSERT INTO role_permissions (tenant_id, role_id, permission_id)
			SELECT $1, $2, p.id FROM permissions p WHERE p.code = ANY($3)
			ON CONFLICT DO NOTHING`, tenantID, roleID, role.Permissions)
		if err != nil {
			return "", fmt.Errorf("seed permissions for %s: %w", role.Code, err)
		}
	}
	if ownerRoleID == "" {
		return "", errors.New("owner role was not seeded")
	}
	return ownerRoleID, nil
}

// AssignRole creates an active role assignment.
func (r *Repository) AssignRole(ctx context.Context, q database.Querier, tenantID, membershipID, roleID, scopeType, companyID, branchID, actorID string) (string, error) {
	var companyArg, branchArg any
	if companyID != "" {
		companyArg = companyID
	}
	if branchID != "" {
		branchArg = branchID
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO membership_role_assignments
			(tenant_id, membership_id, role_id, scope_type, company_id, branch_id)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		tenantID, membershipID, roleID, scopeType, companyArg, branchArg).Scan(&id)
	return id, err
}

// RoleBelongsToTenant reports whether a role is an active role in the tenant.
func (r *Repository) RoleBelongsToTenant(ctx context.Context, q database.Querier, tenantID, roleID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM roles WHERE tenant_id=$1 AND id=$2 AND status='active')`,
		tenantID, roleID).Scan(&exists)
	return exists, err
}

// GetRole returns one role with its permissions.
func (r *Repository) GetRole(ctx context.Context, q database.Querier, tenantID, roleID string) (*Role, error) {
	var role Role
	err := q.QueryRow(ctx, `
		SELECT id, code, name, description, is_system, status, created_at
		FROM roles WHERE tenant_id=$1 AND id=$2`, tenantID, roleID).Scan(
		&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.Status, &role.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	role.Permissions, err = r.rolePermissions(ctx, q, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// ListRoles returns all roles for a tenant with permission codes.
func (r *Repository) ListRoles(ctx context.Context, q database.Querier, tenantID string) ([]Role, error) {
	rows, err := q.Query(ctx, `
		SELECT id, code, name, description, is_system, status, created_at
		FROM roles WHERE tenant_id=$1 ORDER BY is_system DESC, name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem, &role.Status, &role.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range roles {
		permissions, err := r.rolePermissions(ctx, q, tenantID, roles[i].ID)
		if err != nil {
			return nil, err
		}
		roles[i].Permissions = permissions
	}
	return roles, nil
}

func (r *Repository) rolePermissions(ctx context.Context, q database.Querier, tenantID, roleID string) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT p.code FROM role_permissions rp
		JOIN permissions p ON p.id=rp.permission_id
		WHERE rp.tenant_id=$1 AND rp.role_id=$2 ORDER BY p.code`, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	permissions := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		permissions = append(permissions, code)
	}
	return permissions, rows.Err()
}

// CreateRole creates a tenant custom role with optional permissions.
func (r *Repository) CreateRole(ctx context.Context, q database.Querier, tenantID, code, name, description string, permissions []string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO roles (tenant_id, code, name, description, is_system, status)
		VALUES ($1,$2,$3,$4,false,'active') RETURNING id`,
		tenantID, code, name, description).Scan(&id)
	if err != nil {
		return "", err
	}
	if len(permissions) > 0 {
		_, err = q.Exec(ctx, `
			INSERT INTO role_permissions (tenant_id, role_id, permission_id)
			SELECT $1, $2, p.id FROM permissions p WHERE p.code = ANY($3)`,
			tenantID, id, permissions)
		if err != nil {
			return "", err
		}
	}
	return id, nil
}

// UpdateRole updates mutable metadata and, for custom roles, permissions.
func (r *Repository) UpdateRole(ctx context.Context, q database.Querier, tenantID, roleID, name, description string, permissions []string, replacePermissions bool) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE roles SET name=$3, description=$4, updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND id=$2`, tenantID, roleID, name, description)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if replacePermissions {
		if _, err := q.Exec(ctx, `DELETE FROM role_permissions WHERE tenant_id=$1 AND role_id=$2`, tenantID, roleID); err != nil {
			return false, err
		}
		if len(permissions) > 0 {
			_, err = q.Exec(ctx, `
				INSERT INTO role_permissions (tenant_id, role_id, permission_id)
				SELECT $1, $2, p.id FROM permissions p WHERE p.code = ANY($3)`,
				tenantID, roleID, permissions)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// ListPermissions returns the platform catalogue.
func (r *Repository) ListPermissions(ctx context.Context, q database.Querier) ([]Permission, error) {
	rows, err := q.Query(ctx, `SELECT id, code, module, action, description FROM permissions ORDER BY module, action`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var permissions []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Module, &p.Action, &p.Description); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, rows.Err()
}

// GetCompanyTenant resolves the tenant that owns a company.
func (r *Repository) GetCompanyTenant(ctx context.Context, q database.Querier, companyID string) (tenantID string, status string, found bool, err error) {
	err = q.QueryRow(ctx, `SELECT tenant_id, status FROM companies WHERE id=$1`, companyID).Scan(&tenantID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	return tenantID, status, true, nil
}

// RevokeAssignment marks an assignment revoked without deleting history.
func (r *Repository) RevokeAssignment(ctx context.Context, q database.Querier, tenantID, membershipID, assignmentID, actorID string) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE membership_role_assignments
		SET revoked_at=now(), revoked_by=$4, updated_at=now(), row_version=row_version+1
		WHERE tenant_id=$1 AND membership_id=$2 AND id=$3 AND revoked_at IS NULL`,
		tenantID, membershipID, assignmentID, actorID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// CountActiveOwners counts active owner assignments in a tenant.
func (r *Repository) CountActiveOwners(ctx context.Context, q database.Querier, tenantID string) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(DISTINCT a.membership_id)
		FROM membership_role_assignments a
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id
		JOIN tenant_memberships m ON m.id=a.membership_id AND m.status='active' AND m.removed_at IS NULL
		WHERE a.tenant_id=$1 AND a.revoked_at IS NULL AND r.code=$2 AND r.is_system=true`, tenantID, RoleOwner).Scan(&count)
	return count, err
}

// ListMembershipAssignments returns active role assignments for a membership.
func (r *Repository) ListMembershipAssignments(ctx context.Context, q database.Querier, tenantID, membershipID string) ([]Assignment, error) {
	rows, err := q.Query(ctx, `
		SELECT a.id, a.role_id, r.code, r.name, a.scope_type, a.company_id, a.branch_id, a.created_at
		FROM membership_role_assignments a
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id
		WHERE a.tenant_id=$1 AND a.membership_id=$2 AND a.revoked_at IS NULL
		ORDER BY a.created_at`, tenantID, membershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assignments []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.RoleID, &a.RoleCode, &a.RoleName, &a.ScopeType, &a.CompanyID, &a.BranchID, &a.CreatedAt); err != nil {
			return nil, err
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

// Assignment is an active role assignment.
type Assignment struct {
	ID        string    `json:"id"`
	RoleID    string    `json:"role_id"`
	RoleCode  string    `json:"role_code"`
	RoleName  string    `json:"role_name"`
	ScopeType string    `json:"scope_type"`
	CompanyID *string   `json:"company_id,omitempty"`
	BranchID  *string   `json:"branch_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MembershipSummary is the minimal membership record needed for role management.
type MembershipSummary struct {
	ID     string
	UserID string
	Status string
}

// GetMembership returns a tenant membership.
func (r *Repository) GetMembership(ctx context.Context, q database.Querier, tenantID, membershipID string) (*MembershipSummary, error) {
	var m MembershipSummary
	err := q.QueryRow(ctx, `
		SELECT id, user_id, status FROM tenant_memberships
		WHERE tenant_id=$1 AND id=$2 AND removed_at IS NULL`, tenantID, membershipID).Scan(&m.ID, &m.UserID, &m.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// CompanyBelongsToTenant reports whether a company is in the tenant.
func (r *Repository) CompanyBelongsToTenant(ctx context.Context, q database.Querier, tenantID, companyID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE tenant_id=$1 AND id=$2)`, tenantID, companyID).Scan(&exists)
	return exists, err
}

// BranchBelongsToCompany reports whether a branch is in the company/tenant.
func (r *Repository) BranchBelongsToCompany(ctx context.Context, q database.Querier, tenantID, companyID, branchID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM branches WHERE tenant_id=$1 AND company_id=$2 AND id=$3)`, tenantID, companyID, branchID).Scan(&exists)
	return exists, err
}

// LockTenant serializes owner-affecting role changes for a tenant.
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

// AssignmentRoleCode returns the role code for an active assignment.
func (r *Repository) AssignmentRoleCode(ctx context.Context, q database.Querier, tenantID, membershipID, assignmentID string) (string, bool, error) {
	var code string
	err := q.QueryRow(ctx, `
		SELECT r.code FROM membership_role_assignments a
		JOIN roles r ON r.id=a.role_id AND r.tenant_id=a.tenant_id
		WHERE a.tenant_id=$1 AND a.membership_id=$2 AND a.id=$3 AND a.revoked_at IS NULL`,
		tenantID, membershipID, assignmentID).Scan(&code)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return code, true, nil
}

// HasAssignments reports whether a membership already holds the same role at
// the same scope (active only).
func (r *Repository) HasAssignments(ctx context.Context, q database.Querier, tenantID, membershipID, roleID string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM membership_role_assignments
			WHERE tenant_id=$1 AND membership_id=$2 AND role_id=$3
			  AND scope_type='tenant' AND revoked_at IS NULL)`,
		tenantID, membershipID, roleID).Scan(&exists)
	return exists, err
}
