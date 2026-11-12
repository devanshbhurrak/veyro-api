package invitations

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/modules/authorization"
	"backend/internal/modules/tenancy"
	"backend/internal/platform/audit"
	"backend/internal/platform/auth"
	"backend/internal/platform/database"
	"backend/internal/platform/notification"
	"backend/internal/platform/outbox"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

const defaultInvitationTTL = 7 * 24 * time.Hour

// Service implements workspace member invitations.
type Service struct {
	db        *database.Database
	repo      *Repository
	authz     *authorization.Repository
	tenancy   *tenancy.Repository
	outbox    *outbox.Repository
	encryptor *auth.Encryptor
	audit     *audit.Recorder
	baseURL   string
	ttl       time.Duration
	now       func() time.Time
}

func NewService(db *database.Database, repo *Repository, authz *authorization.Repository,
	tenancyRepo *tenancy.Repository, outboxRepo *outbox.Repository, encryptor *auth.Encryptor,
	recorder *audit.Recorder, baseURL string, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = defaultInvitationTTL
	}
	return &Service{db: db, repo: repo, authz: authz, tenancy: tenancyRepo, outbox: outboxRepo,
		encryptor: encryptor, audit: recorder, baseURL: strings.TrimRight(baseURL, "/"),
		ttl: ttl, now: time.Now}
}

func (r *Repository) TenantName(ctx context.Context, q database.Querier, tenantID string) (string, error) {
	var name string
	err := q.QueryRow(ctx, `SELECT name FROM tenants WHERE id=$1`, tenantID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

// CreateInput is an invitation request.
type CreateInput struct {
	Email   string
	RoleID  string
	Message string
}

// Create issues a pending invitation and queues its email.
func (s *Service) Create(ctx context.Context, tenantID, actorID, inviterName string, in CreateInput, meta RequestMeta) (*Invitation, error) {
	email, ok := validate.Email(in.Email)
	if !ok {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Enter a valid email address.").
			WithDetails(map[string]string{"email": "Enter a valid email address."})
	}
	if !validate.UUID(in.RoleID) {
		return nil, apperr.BadRequest(apperr.CodeValidation, "A valid role is required.").
			WithDetails(map[string]string{"role_id": "Select a role."})
	}
	message := strings.TrimSpace(in.Message)
	if len([]rune(message)) > 500 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "Message must be at most 500 characters.")
	}
	role, err := s.authz.GetRole(ctx, s.db.Pool, tenantID, in.RoleID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if role == nil || role.Status != "active" {
		return nil, apperr.BadRequest(apperr.CodeValidation, "The selected role is not available.")
	}
	if role.Code == authorization.RoleOwner {
		return nil, apperr.Forbidden("owner_role_invitation", "The owner role cannot be granted through an invitation.")
	}
	status, exists, err := s.repo.MembershipStatus(ctx, s.db.Pool, tenantID, email)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if exists && status == "active" {
		return nil, apperr.Conflict("already_member", "This person is already a member of the workspace.")
	}

	token, err := auth.NewToken()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	tokenHash := auth.HashToken(token)
	encrypted, err := s.encryptor.Encrypt(token)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	tenantName, err := s.repo.TenantName(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	payload, err := notification.Encode(notification.Payload{
		Kind: notification.KindInvitation, To: email, TokenEnc: encrypted,
		LinkPath: "/accept-invitation", TenantName: tenantName, InviterName: inviterName,
		Message: message,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	expiresAt := s.now().Add(s.ttl)

	var invitation *Invitation
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.repo.ExpireStale(ctx, tx, tenantID, email); err != nil {
			return apperr.Internal(err)
		}
		created, err := s.repo.Create(ctx, tx, tenantID, email, in.RoleID, actorID,
			optional(message), tokenHash, expiresAt)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return apperr.Conflict("invitation_exists", "An invitation is already pending for this email.")
			}
			return apperr.Internal(err)
		}
		invitation = created
		if err := s.outbox.Enqueue(ctx, tx, outbox.Event{
			TenantID: &tenantID, AggregateType: "tenant_invitation", AggregateID: created.ID,
			EventType: outbox.EventInvitation, Payload: payload,
			DeduplicationKey: "invitation:" + created.ID + ":" + shortHash(tokenHash),
		}); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, ActorType: "user",
			Action: "invitation.created", EntityType: "tenant_invitation", EntityID: &created.ID,
			RequestID: meta.RequestID, IPAddress: optional(meta.IPAddress),
			UserAgent: optional(meta.UserAgent),
			Metadata:  map[string]any{"email": email, "role_id": in.RoleID},
		})
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	return invitation, nil
}

// List returns all invitations for the workspace.
func (s *Service) List(ctx context.Context, tenantID string) ([]Invitation, error) {
	if err := s.repo.ExpireStaleAll(ctx, s.db.Pool, tenantID); err != nil {
		return nil, apperr.Internal(err)
	}
	list, err := s.repo.List(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if list == nil {
		list = []Invitation{}
	}
	return list, nil
}

// Resend rotates the token and queues a new email.
func (s *Service) Resend(ctx context.Context, tenantID, id, actorID, inviterName string, meta RequestMeta) (*Invitation, error) {
	if !validate.UUID(id) {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Invitation not found.")
	}
	invitation, err := s.repo.Get(ctx, s.db.Pool, tenantID, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if invitation == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "Invitation not found.")
	}
	if invitation.Status != "pending" {
		return nil, apperr.Conflict("invitation_not_pending", "Only pending invitations can be resent.")
	}

	token, err := auth.NewToken()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	tokenHash := auth.HashToken(token)
	encrypted, err := s.encryptor.Encrypt(token)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	tenantName, err := s.repo.TenantName(ctx, s.db.Pool, tenantID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var message string
	if invitation.Message != nil {
		message = *invitation.Message
	}
	payload, err := notification.Encode(notification.Payload{
		Kind: notification.KindInvitation, To: invitation.Email, TokenEnc: encrypted,
		LinkPath: "/accept-invitation", TenantName: tenantName, InviterName: inviterName,
		Message: message,
	})
	if err != nil {
		return nil, apperr.Internal(err)
	}
	expiresAt := s.now().Add(s.ttl)

	var updated bool
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		updated, err = s.repo.Refresh(ctx, tx, tenantID, id, tokenHash, expiresAt)
		if err != nil {
			return apperr.Internal(err)
		}
		if !updated {
			return nil
		}
		if err := s.outbox.Enqueue(ctx, tx, outbox.Event{
			TenantID: &tenantID, AggregateType: "tenant_invitation", AggregateID: id,
			EventType: outbox.EventInvitation, Payload: payload,
			DeduplicationKey: "invitation:" + id + ":" + shortHash(tokenHash),
		}); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, ActorType: "user",
			Action: "invitation.resent", EntityType: "tenant_invitation", EntityID: &id,
			RequestID: meta.RequestID, IPAddress: optional(meta.IPAddress),
			UserAgent: optional(meta.UserAgent),
		})
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	if !updated {
		return nil, apperr.Conflict("invitation_not_pending", "Only pending invitations can be resent.")
	}
	return s.repo.Get(ctx, s.db.Pool, tenantID, id)
}

// Revoke cancels a pending invitation.
func (s *Service) Revoke(ctx context.Context, tenantID, id, actorID string, meta RequestMeta) error {
	if !validate.UUID(id) {
		return apperr.NotFound(apperr.CodeNotFound, "Invitation not found.")
	}
	var revoked bool
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		revoked, err = s.repo.Revoke(ctx, tx, tenantID, id, actorID)
		if err != nil {
			return apperr.Internal(err)
		}
		if !revoked {
			return nil
		}
		return s.audit.Record(ctx, tx, audit.Event{
			TenantID: &tenantID, ActorUserID: &actorID, ActorType: "user",
			Action: "invitation.revoked", EntityType: "tenant_invitation", EntityID: &id,
			RequestID: meta.RequestID, IPAddress: optional(meta.IPAddress),
			UserAgent: optional(meta.UserAgent),
		})
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return err
		}
		return apperr.Internal(err)
	}
	if !revoked {
		return apperr.NotFound(apperr.CodeNotFound, "Pending invitation not found.")
	}
	return nil
}

// AcceptResult reports the membership created by accepting an invitation.
type AcceptResult struct {
	TenantID     string `json:"tenant_id"`
	MembershipID string `json:"membership_id"`
}

// Accept validates the token, creates the membership, and assigns the role in
// one transaction. The authenticated user's email must match the invitation.
func (s *Service) Accept(ctx context.Context, userID, userEmail, token string, meta RequestMeta) (*AcceptResult, error) {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 512 {
		return nil, apperr.BadRequest(apperr.CodeValidation, "An invitation token is required.")
	}
	tokenHash := auth.HashToken(token)
	email := validate.NormalizeEmail(userEmail)

	var result AcceptResult
	var acceptErr error
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		inv, err := s.repo.GetPendingByToken(ctx, tx, tokenHash)
		if err != nil {
			return apperr.Internal(err)
		}
		if inv == nil {
			acceptErr = apperr.BadRequest(apperr.CodeInvalidInvitation, "The invitation is invalid or has already been used.")
			return nil
		}
		if !inv.ExpiresAt.After(s.now()) {
			if _, err := tx.Exec(ctx, `UPDATE tenant_invitations SET status='expired', updated_at=now(),
				row_version=row_version+1 WHERE id=$1 AND status='pending'`, inv.ID); err != nil {
				return apperr.Internal(err)
			}
			acceptErr = apperr.BadRequest(apperr.CodeExpiredToken, "The invitation has expired. Ask an administrator to resend it.")
			return nil
		}
		if inv.Email != email {
			acceptErr = apperr.Forbidden(apperr.CodeEmailMismatch, "This invitation was sent to a different email address.")
			return nil
		}
		if inv.TenantStatus != "active" {
			acceptErr = apperr.Forbidden(apperr.CodeTenantSuspended, "This workspace is not active.")
			return nil
		}
		if inv.RoleCode == authorization.RoleOwner {
			acceptErr = apperr.BadRequest(apperr.CodeInvalidInvitation, "The invitation grants an unavailable role.")
			return nil
		}
		status, exists, err := s.repo.MembershipStatus(ctx, tx, inv.TenantID, email)
		if err != nil {
			return apperr.Internal(err)
		}
		if exists {
			if status == "active" {
				acceptErr = apperr.Conflict("already_member", "You are already a member of this workspace.")
			} else {
				acceptErr = apperr.Conflict("membership_inactive",
					"Your membership is not active. Ask a workspace administrator to restore access.")
			}
			return nil
		}

		membershipID, err := s.tenancy.CreateMembership(ctx, tx, inv.TenantID, userID)
		if err != nil {
			if database.IsUniqueViolation(err) {
				acceptErr = apperr.Conflict("already_member", "You are already a member of this workspace.")
				return nil
			}
			return apperr.Internal(err)
		}
		if _, err := s.authz.AssignRole(ctx, tx, inv.TenantID, membershipID, inv.RoleID,
			"tenant", "", "", userID); err != nil {
			if database.IsUniqueViolation(err) {
				acceptErr = apperr.Conflict(apperr.CodeConflict, "The role is already assigned.")
				return nil
			}
			return apperr.Internal(err)
		}
		accepted, err := s.repo.MarkAccepted(ctx, tx, inv.TenantID, inv.ID, userID)
		if err != nil {
			return apperr.Internal(err)
		}
		if !accepted {
			acceptErr = apperr.Conflict(apperr.CodeInvalidInvitation, "The invitation was just used.")
			return nil
		}
		if err := s.audit.Record(ctx, tx, audit.Event{
			TenantID: &inv.TenantID, ActorUserID: &userID, ActorType: "user",
			Action: "invitation.accepted", EntityType: "tenant_invitation", EntityID: &inv.ID,
			RequestID: meta.RequestID, IPAddress: optional(meta.IPAddress),
			UserAgent: optional(meta.UserAgent),
			Metadata:  map[string]any{"membership_id": membershipID},
		}); err != nil {
			return apperr.Internal(err)
		}
		result = AcceptResult{TenantID: inv.TenantID, MembershipID: membershipID}
		return nil
	})
	if err != nil {
		if _, ok := apperr.As(err); ok {
			return nil, err
		}
		return nil, apperr.Internal(err)
	}
	if acceptErr != nil {
		return nil, acceptErr
	}
	return &result, nil
}

func optional(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func shortHash(hash []byte) string {
	if len(hash) < 8 {
		return hex.EncodeToString(hash)
	}
	return hex.EncodeToString(hash[:8])
}

// BaseURL exposes the configured frontend URL for links.
func (s *Service) BaseURL() string { return s.baseURL }
