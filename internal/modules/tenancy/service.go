package tenancy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/modules/authorization"
	"backend/internal/platform/audit"
	"backend/internal/platform/database"
	"backend/internal/platform/idempotency"
	"backend/internal/shared/apperr"
	"backend/internal/shared/idgen"
	"backend/internal/shared/validate"
)

// RoleInitializer seeds system roles and assigns ownership.
type RoleInitializer interface {
	SeedDefaultRoles(ctx context.Context, tx pgx.Tx, tenantID string) (string, error)
	AssignOwner(ctx context.Context, tx pgx.Tx, tenantID, membershipID, ownerRoleID string) error
}

// Service implements workspace and membership workflows.
type Service struct {
	db    *database.Database
	repo  *Repository
	roles RoleInitializer
	audit *audit.Recorder
	now   func() time.Time
}

func NewService(db *database.Database, repo *Repository, roles RoleInitializer, auditRecorder *audit.Recorder) *Service {
	return &Service{db: db, repo: repo, roles: roles, audit: auditRecorder, now: time.Now}
}

type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}

// CreateWorkspace atomically creates a tenant, owner membership and system roles.
func (s *Service) CreateWorkspace(ctx context.Context, userID, name, preferredSlug, idempotencyKey string, meta RequestMeta) (*Workspace, error) {
	cleanName, ok := validate.Required(name, 160)
	if !ok {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Workspace name is required (maximum 160 characters).")
	}
	baseSlug := validate.Slugify(cleanName)
	if strings.TrimSpace(preferredSlug) != "" {
		baseSlug = strings.ToLower(strings.TrimSpace(preferredSlug))
		if !validate.Slug(baseSlug) {
			return nil, apperr.BadRequest(apperr.CodeValidation,
				"Workspace slug must contain lowercase letters, digits and hyphens, and be at most 60 characters.")
		}
	}

	requestHash := idempotency.Hash("workspace.create", userID, cleanName, baseSlug)
	var workspaceID string
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		record, existingID, acquired, err := idempotency.Reserve(ctx, tx,
			"workspace.create", idempotencyKey, requestHash, "", "", userID, 24*time.Hour)
		if err != nil {
			return err
		}
		if !acquired {
			workspaceID = existingID
			return nil
		}

		tenantID, err := s.allocateTenant(ctx, tx, cleanName, baseSlug)
		if err != nil {
			return err
		}
		workspaceID = tenantID

		membershipID, err := s.repo.CreateMembership(ctx, tx, tenantID, userID)
		if err != nil {
			return apperr.Internal(err)
		}
		ownerRoleID, err := s.roles.SeedDefaultRoles(ctx, tx, tenantID)
		if err != nil {
			return apperr.Internal(err)
		}
		if err := s.roles.AssignOwner(ctx, tx, tenantID, membershipID, ownerRoleID); err != nil {
			return apperr.Internal(err)
		}
		if err := s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &userID, ActorType: "user", Action: "tenancy.workspace_created",
			EntityType: "tenant", EntityID: &tenantID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
			Metadata: map[string]any{"slug": baseSlug},
		}); err != nil {
			return apperr.Internal(err)
		}
		if record != nil {
			if err := idempotency.Complete(ctx, tx, record.ID, tenantID, 201); err != nil {
				return apperr.Internal(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	workspace, err := s.repo.GetWorkspace(ctx, s.db.Pool, workspaceID, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if workspace == nil {
		return nil, apperr.Internal(fmt.Errorf("workspace %s not readable after creation", workspaceID))
	}
	return workspace, nil
}

// allocateTenant retries collision-safe slug allocation within the transaction.
func (s *Service) allocateTenant(ctx context.Context, tx pgx.Tx, name, base string) (string, error) {
	candidates := make([]string, 0, 12)
	candidates = append(candidates, base)
	for i := 2; i <= 10; i++ {
		suffix := fmt.Sprintf("-%d", i)
		candidates = append(candidates, trimSlug(base, suffix))
	}
	for _, candidate := range candidates {
		id, inserted, err := s.repo.InsertTenantIfSlugFree(ctx, tx, name, candidate)
		if err != nil {
			return "", apperr.Internal(err)
		}
		if inserted {
			return id, nil
		}
	}
	for i := 0; i < 5; i++ {
		random, err := idgen.New()
		if err != nil {
			return "", apperr.Internal(err)
		}
		candidate := trimSlug(base, "-"+random[:8])
		id, inserted, err := s.repo.InsertTenantIfSlugFree(ctx, tx, name, candidate)
		if err != nil {
			return "", apperr.Internal(err)
		}
		if inserted {
			return id, nil
		}
	}
	return "", apperr.Conflict("workspace_slug_conflict", "A workspace slug could not be allocated. Please try another slug.")
}

func trimSlug(base, suffix string) string {
	max := 60 - len(suffix)
	if len(base) > max {
		base = strings.TrimRight(base[:max], "-")
	}
	return base + suffix
}

func (s *Service) ListWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	workspaces, err := s.repo.ListWorkspaces(ctx, s.db.Pool, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return workspaces, nil
}

func (s *Service) GetWorkspace(ctx context.Context, tenantID, userID string) (*Workspace, error) {
	workspace, err := s.repo.GetWorkspace(ctx, s.db.Pool, tenantID, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if workspace == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
	}
	return workspace, nil
}

type UpdateWorkspaceInput struct {
	Name     string
	Timezone string
	Locale   string
}

func (s *Service) UpdateWorkspace(ctx context.Context, tenantID, userID string, in UpdateWorkspaceInput, meta RequestMeta) (*Workspace, error) {
	current, err := s.GetWorkspace(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	name := current.Name
	if strings.TrimSpace(in.Name) != "" {
		clean, ok := validate.Required(in.Name, 160)
		if !ok {
			return nil, apperr.BadRequest(apperr.CodeValidation, "Workspace name must be at most 160 characters.")
		}
		name = clean
	}
	timezone := strings.TrimSpace(in.Timezone)
	if timezone == "" {
		timezone = current.Timezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Timezone must be a valid IANA timezone.")
	}
	locale := strings.TrimSpace(in.Locale)
	if locale == "" {
		locale = current.Locale
	}
	if len(locale) > 12 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Locale must be at most 12 characters.")
	}
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.repo.UpdateWorkspace(ctx, tx, tenantID, name, timezone, locale); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &userID, ActorType: "user", Action: "tenancy.workspace_updated",
			EntityType: "tenant", EntityID: &tenantID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
	if err != nil {
		return nil, err
	}
	return s.GetWorkspace(ctx, tenantID, userID)
}

func (s *Service) ListMembers(ctx context.Context, tenantID string) ([]Member, error) {
	members, err := s.repo.ListMembers(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	return members, nil
}

func (s *Service) GetMember(ctx context.Context, tenantID, membershipID string) (*Member, error) {
	if !validate.UUID(membershipID) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
	}
	member, err := s.repo.GetMember(ctx, s.db.Pool, tenantID, membershipID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if member == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
	}
	return member, nil
}

type UpdateMemberInput struct {
	Status string
}

func (s *Service) UpdateMember(ctx context.Context, tenantID, membershipID, actorUserID string, in UpdateMemberInput, meta RequestMeta) (*Member, error) {
	member, err := s.GetMember(ctx, tenantID, membershipID)
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(in.Status)
	if status != "active" && status != "suspended" {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Membership status must be active or suspended.")
	}
	if status == "suspended" && member.UserID == actorUserID {
		return nil, apperr.Forbidden("self_suspend", "You cannot suspend your own membership.")
	}
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.lockTenant(ctx, tx, tenantID); err != nil {
			return err
		}
		if status == "suspended" {
			if err := s.ensureNotLastOwner(ctx, tx, tenantID, membershipID); err != nil {
				return err
			}
		}
		found, err := s.repo.UpdateMembershipStatus(ctx, tx, tenantID, membershipID, status)
		if err != nil {
			return apperr.Internal(err)
		}
		if !found {
			return apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorUserID, ActorType: "user", Action: "tenancy.membership_status_changed",
			EntityType: "tenant_membership", EntityID: &membershipID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
			Metadata: map[string]any{"status": status},
		})
	})
	if err != nil {
		return nil, err
	}
	return s.GetMember(ctx, tenantID, membershipID)
}

func (s *Service) RemoveMember(ctx context.Context, tenantID, membershipID, actorUserID string, meta RequestMeta) error {
	if _, err := s.GetMember(ctx, tenantID, membershipID); err != nil {
		return err
	}
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.lockTenant(ctx, tx, tenantID); err != nil {
			return err
		}
		if err := s.ensureNotLastOwner(ctx, tx, tenantID, membershipID); err != nil {
			return err
		}
		found, err := s.repo.UpdateMembershipStatus(ctx, tx, tenantID, membershipID, "removed")
		if err != nil {
			return apperr.Internal(err)
		}
		if !found {
			return apperr.NotFound(apperr.CodeNotFound, "Membership not found.")
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorUserID, ActorType: "user", Action: "tenancy.membership_removed",
			EntityType: "tenant_membership", EntityID: &membershipID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

func (s *Service) lockTenant(ctx context.Context, tx pgx.Tx, tenantID string) error {
	found, err := s.repo.LockTenant(ctx, tx, tenantID)
	if err != nil {
		return apperr.Internal(err)
	}
	if !found {
		return apperr.NotFound(apperr.CodeNotFound, "Workspace not found.")
	}
	return nil
}

func (s *Service) ensureNotLastOwner(ctx context.Context, q database.Querier, tenantID, membershipID string) error {
	hasOwner, err := s.repo.MembershipHasRole(ctx, q, tenantID, membershipID, authorization.RoleOwner)
	if err != nil {
		return apperr.Internal(err)
	}
	if !hasOwner {
		return nil
	}
	count, err := s.repo.CountOwners(ctx, q, tenantID)
	if err != nil {
		return apperr.Internal(err)
	}
	if count <= 1 {
		return apperr.Conflict(apperr.CodeLastOwner, "The last active owner cannot be removed. Transfer ownership first.")
	}
	return nil
}

func optional(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
