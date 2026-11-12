package authorization

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/audit"
	"backend/internal/platform/database"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,59}$`)

// Service implements role and assignment management.
type Service struct {
	db    *database.Database
	repo  *Repository
	audit *audit.Recorder
}

func NewService(db *database.Database, repo *Repository, auditRecorder *audit.Recorder) *Service {
	return &Service{db: db, repo: repo, audit: auditRecorder}
}

// SeedDefaultRoles creates the baseline roles for a new workspace.
func (s *Service) SeedDefaultRoles(ctx context.Context, tx pgx.Tx, tenantID string) (string, error) {
	return s.repo.SeedDefaultRoles(ctx, tx, tenantID)
}

// AssignOwner grants the Owner role at tenant scope.
func (s *Service) AssignOwner(ctx context.Context, tx pgx.Tx, tenantID, membershipID, ownerRoleID string) error {
	_, err := s.repo.AssignRole(ctx, tx, tenantID, membershipID, ownerRoleID, "tenant", "", "", "")
	return err
}

func (s *Service) ListRoles(ctx context.Context, tenantID string) ([]Role, error) {
	roles, err := s.repo.ListRoles(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return roles, nil
}

func (s *Service) GetRole(ctx context.Context, tenantID, roleID string) (*Role, error) {
	if !validate.UUID(roleID) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Role not found.")
	}
	role, err := s.repo.GetRole(ctx, s.db.Pool, tenantID, roleID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if role == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Role not found.")
	}
	return role, nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	permissions, err := s.repo.ListPermissions(ctx, s.db.Pool)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return permissions, nil
}

func (s *Service) ListAssignments(ctx context.Context, tenantID, membershipID string) ([]Assignment, error) {
	membership, err := s.repo.GetMembership(ctx, s.db.Pool, tenantID, membershipID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if membership == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
	}
	assignments, err := s.repo.ListMembershipAssignments(ctx, s.db.Pool, tenantID, membershipID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return assignments, nil
}

type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

func (s *Service) CreateRole(ctx context.Context, tenantID string, actor *audit.Event, in CreateRoleInput) (*Role, error) {
	code := strings.TrimSpace(in.Code)
	if !roleCodePattern.MatchString(code) {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Role code must be 2-60 lowercase letters, digits or underscores.")
	}
	name, ok := validate.Required(in.Name, 100)
	if !ok {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Role name is required (maximum 100 characters).")
	}
	permissions, err := validatePermissions(in.Permissions)
	if err != nil {
		return nil, err
	}
	var roleID string
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		id, err := s.repo.CreateRole(ctx, tx, tenantID, code, name, in.Description, permissions)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return apperr.Conflict(apperr.CodeConflict, "A role with this code already exists.")
			}
			return apperr.Internal(err)
		}
		roleID = id
		actor.TenantID = &tenantID
		actor.Action = "authorization.role_created"
		actor.EntityType = "role"
		actor.EntityID = &id
		actor.Metadata = map[string]any{"code": code}
		return s.audit.Record(ctx, tx, *actor)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRole(ctx, tenantID, roleID)
}

type UpdateRoleInput struct {
	Name string
	// Description and Permissions are nil when the client did not send them,
	// which preserves the existing values.
	Description *string
	Permissions *[]string
}

func (s *Service) UpdateRole(ctx context.Context, tenantID, roleID string, actor *audit.Event, in UpdateRoleInput) (*Role, error) {
	existing, err := s.GetRole(ctx, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	name, ok := validate.Required(in.Name, 100)
	if !ok {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Role name is required (maximum 100 characters).")
	}
	var permissions []string
	if in.Permissions != nil {
		var err error
		permissions, err = validatePermissions(*in.Permissions)
		if err != nil {
			return nil, err
		}
	}
	replace := !existing.IsSystem && in.Permissions != nil
	description := ""
	if existing.Description != nil {
		description = *existing.Description
	}
	if in.Description != nil {
		description = strings.TrimSpace(*in.Description)
	}
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		found, err := s.repo.UpdateRole(ctx, tx, tenantID, roleID, name, description, permissions, replace)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return apperr.Conflict(apperr.CodeConflict, "A role with this name already exists.")
			}
			return apperr.Internal(err)
		}
		if !found {
			return apperr.NotFound(apperr.CodeNotFound, "Role not found.")
		}
		actor.TenantID = &tenantID
		actor.Action = "authorization.role_updated"
		actor.EntityType = "role"
		actor.EntityID = &roleID
		return s.audit.Record(ctx, tx, *actor)
	})
	if err != nil {
		return nil, err
	}
	return s.GetRole(ctx, tenantID, roleID)
}

type AssignRoleInput struct {
	ScopeType string
	CompanyID string
	BranchID  string
}

func (s *Service) AssignRole(ctx context.Context, tenantID, membershipID, roleID, actorUserID string, actor *audit.Event, in AssignRoleInput) (string, error) {
	if !validate.UUID(membershipID) || !validate.UUID(roleID) {
		return "", apperr.NotFound(apperr.CodeNotFound, "Membership or role not found.")
	}
	membership, err := s.repo.GetMembership(ctx, s.db.Pool, tenantID, membershipID)
	if err != nil {
		return "", apperr.Internal(err)
	}
	if membership == nil || membership.Status != "active" {
		return "", apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
	}
	if membership.UserID == actorUserID {
		return "", apperr.Forbidden("self_role_change", "You cannot change your own role assignments.")
	}
	exists, err := s.repo.RoleBelongsToTenant(ctx, s.db.Pool, tenantID, roleID)
	if err != nil {
		return "", apperr.Internal(err)
	}
	if !exists {
		return "", apperr.BadRequest(apperr.CodeValidation, "The role does not belong to this workspace.")
	}

	scope := in.ScopeType
	if scope == "" {
		scope = "tenant"
	}
	switch scope {
	case "tenant":
		if in.CompanyID != "" || in.BranchID != "" {
			return "", apperr.BadRequest(apperr.CodeValidation, "Tenant-scoped assignments cannot include company or branch scope.")
		}
	case "company":
		if !validate.UUID(in.CompanyID) {
			return "", apperr.BadRequest(apperr.CodeValidation, "A valid company is required for company scope.")
		}
		ok, err := s.repo.CompanyBelongsToTenant(ctx, s.db.Pool, tenantID, in.CompanyID)
		if err != nil {
			return "", apperr.Internal(err)
		}
		if !ok {
			return "", apperr.BadRequest(apperr.CodeValidation, "The company does not belong to this workspace.")
		}
	case "branch":
		if !validate.UUID(in.CompanyID) || !validate.UUID(in.BranchID) {
			return "", apperr.BadRequest(apperr.CodeValidation, "A valid company and branch are required for branch scope.")
		}
		ok, err := s.repo.BranchBelongsToCompany(ctx, s.db.Pool, tenantID, in.CompanyID, in.BranchID)
		if err != nil {
			return "", apperr.Internal(err)
		}
		if !ok {
			return "", apperr.BadRequest(apperr.CodeValidation, "The branch does not belong to this company.")
		}
	default:
		return "", apperr.BadRequest(apperr.CodeValidation, "Scope type must be tenant, company or branch.")
	}

	duplicate, err := s.repo.HasAssignments(ctx, s.db.Pool, tenantID, membershipID, roleID)
	if err != nil {
		return "", apperr.Internal(err)
	}
	if duplicate && scope == "tenant" {
		return "", apperr.Conflict(apperr.CodeConflict, "The member already holds this role.")
	}

	var assignmentID string
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		id, err := s.repo.AssignRole(ctx, tx, tenantID, membershipID, roleID, scope, in.CompanyID, in.BranchID, actorUserID)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return apperr.Conflict(apperr.CodeConflict, "The member already holds this role at this scope.")
			}
			return apperr.Internal(err)
		}
		assignmentID = id
		actor.TenantID = &tenantID
		actor.Action = "authorization.role_assigned"
		actor.EntityType = "membership_role_assignment"
		actor.EntityID = &id
		actor.Metadata = map[string]any{"membership_id": membershipID, "role_id": roleID, "scope_type": scope}
		return s.audit.Record(ctx, tx, *actor)
	})
	if err != nil {
		return "", err
	}
	return assignmentID, nil
}

func (s *Service) RevokeAssignment(ctx context.Context, tenantID, membershipID, assignmentID, actorUserID string, actor *audit.Event) error {
	if !validate.UUID(assignmentID) {
		return apperr.NotFound(apperr.CodeNotFound, "Role assignment not found.")
	}
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		locked, err := s.repo.LockTenant(ctx, tx, tenantID)
		if err != nil {
			return apperr.Internal(err)
		}
		if !locked {
			return apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
		}
		roleCode, found, err := s.repo.AssignmentRoleCode(ctx, tx, tenantID, membershipID, assignmentID)
		if err != nil {
			return apperr.Internal(err)
		}
		if !found {
			return apperr.NotFound(apperr.CodeNotFound, "Role assignment not found.")
		}
		if roleCode == RoleOwner {
			count, err := s.repo.CountActiveOwners(ctx, tx, tenantID)
			if err != nil {
				return apperr.Internal(err)
			}
			if count <= 1 {
				return apperr.Conflict(apperr.CodeLastOwner, "The last active owner cannot lose the Owner role.")
			}
		}
		revoked, err := s.repo.RevokeAssignment(ctx, tx, tenantID, membershipID, assignmentID, actorUserID)
		if err != nil {
			return apperr.Internal(err)
		}
		if !revoked {
			return apperr.NotFound(apperr.CodeNotFound, "Role assignment not found.")
		}
		actor.TenantID = &tenantID
		actor.Action = "authorization.role_revoked"
		actor.EntityType = "membership_role_assignment"
		actor.EntityID = &assignmentID
		return s.audit.Record(ctx, tx, *actor)
	})
}

func validatePermissions(permissions []string) ([]string, error) {
	approved := make(map[string]struct{}, len(AllPermissions))
	for _, code := range AllPermissions {
		approved[code] = struct{}{}
	}
	seen := make(map[string]struct{}, len(permissions))
	result := make([]string, 0, len(permissions))
	for _, code := range permissions {
		code = strings.TrimSpace(code)
		if _, ok := approved[code]; !ok {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Unknown permission: "+code)
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result, nil
}

// RequestMeta builds audit context from request metadata.
func AuditActor(userID, requestID, ip, userAgent string) *audit.Event {
	return &audit.Event{
		ActorUserID: &userID, ActorType: "user", RequestID: requestID,
		IPAddress: nilIfEmpty(ip), UserAgent: nilIfEmpty(userAgent),
	}
}

func nilIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

// Timestamp is exposed for handlers producing response metadata.
type Timestamp = time.Time
