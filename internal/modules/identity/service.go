package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/audit"
	"backend/internal/platform/auth"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/notification"
	"backend/internal/platform/outbox"
	"backend/internal/platform/throttle"
	"backend/internal/shared/apperr"
	"backend/internal/shared/validate"
)

const (
	purposeEmailVerification = "email_verification"
	purposePasswordReset     = "password_reset"
)

// Service implements identity and authentication workflows.
type Service struct {
	db        *database.Database
	repo      *Repository
	hasher    *auth.PasswordHasher
	encryptor *auth.Encryptor
	authCfg   config.AuthConfig
	appCfg    config.AppConfig
	outbox    *outbox.Repository
	audit     *audit.Recorder

	registerLimiter *throttle.Limiter
	loginLimiter    *throttle.Limiter
	resendLimiter   *throttle.Limiter
	resetLimiter    *throttle.Limiter

	dummyHash string
	now       func() time.Time
}

func NewService(
	db *database.Database,
	repo *Repository,
	hasher *auth.PasswordHasher,
	encryptor *auth.Encryptor,
	authCfg config.AuthConfig,
	appCfg config.AppConfig,
	outboxRepo *outbox.Repository,
	auditRecorder *audit.Recorder,
) (*Service, error) {
	dummy, err := hasher.Hash("veyro-timing-equalisation-only-1")
	if err != nil {
		return nil, err
	}
	return &Service{
		db: db, repo: repo, hasher: hasher, encryptor: encryptor,
		authCfg: authCfg, appCfg: appCfg, outbox: outboxRepo, audit: auditRecorder,
		registerLimiter: throttle.New(authCfg.LoginWindow, authCfg.LoginMaxAttempts),
		loginLimiter:    throttle.New(authCfg.LoginWindow, authCfg.LoginMaxAttempts),
		resendLimiter:   throttle.New(authCfg.ResendWindow, authCfg.ResendMaxAttempts),
		resetLimiter:    throttle.New(authCfg.PasswordResetWindow, authCfg.PasswordResetMaxAttempts),
		dummyHash:       dummy,
		now:             time.Now,
	}, nil
}

// Register creates a user, a verification token and an outbox notification in
// one transaction. Email delivery happens asynchronously.
func (s *Service) Register(ctx context.Context, fullName, email, password string, meta RequestMeta) error {
	name, ok := validate.Required(fullName, 160)
	if !ok {
		return apperr.BadRequest(apperr.CodeValidation, "Full name is required (maximum 160 characters).")
	}
	normalized, ok := validate.Email(email)
	if !ok {
		return apperr.BadRequest(apperr.CodeValidation, "A valid email address is required.")
	}
	if err := auth.ValidatePassword(password); err != nil {
		return apperr.BadRequest(apperr.CodeValidation, err.Error())
	}
	if !s.registerLimiter.Allow("register:" + meta.IPAddress) {
		return apperr.TooManyRequests(apperr.CodeTooManyRequests, "Too many registration attempts. Please try again later.")
	}

	existing, err := s.repo.GetUserByEmail(ctx, s.db.Pool, normalized)
	if err != nil {
		return apperr.Internal(err)
	}
	if existing != nil {
		return apperr.Conflict("registration_conflict", "An account with these details already exists.")
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return apperr.Internal(err)
	}

	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		user, err := s.repo.CreateUser(ctx, tx, normalized, name, hash)
		if err != nil {
			if database.IsUniqueViolation(err) {
				return apperr.Conflict("registration_conflict", "An account with these details already exists.")
			}
			return apperr.Internal(err)
		}
		if err := s.issueEmailToken(ctx, tx, user, purposeEmailVerification, "/verify-email", notification.KindEmailVerification, meta); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &user.ID, ActorType: "user", Action: "identity.registered",
			EntityType: "user", EntityID: &user.ID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// VerifyEmail consumes a single-use verification token and marks the email verified.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return apperr.BadRequest(apperr.CodeValidation, "Verification token is required.")
	}
	hash := auth.HashToken(token)
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		userID, err := s.repo.ConsumeAuthToken(ctx, tx, hash, purposeEmailVerification)
		if err != nil {
			return s.tokenError(ctx, tx, hash, purposeEmailVerification, err)
		}
		if err := s.repo.MarkEmailVerified(ctx, tx, userID); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &userID, ActorType: "user", Action: "identity.email_verified",
			EntityType: "user", EntityID: &userID,
		})
	})
}

// ResendVerification issues a fresh verification token, superseding older ones.
func (s *Service) ResendVerification(ctx context.Context, email string, meta RequestMeta) error {
	normalized, ok := validate.Email(email)
	if !ok {
		return apperr.BadRequest(apperr.CodeValidation, "A valid email address is required.")
	}
	if !s.resendLimiter.Allow("verify:" + normalized) {
		return apperr.TooManyRequests(apperr.CodeTooManyRequests, "Please wait before requesting another verification email.")
	}
	user, err := s.repo.GetUserByEmail(ctx, s.db.Pool, normalized)
	if err != nil {
		return apperr.Internal(err)
	}
	if user == nil || user.Verified() || user.AccountStatus != "active" {
		return nil // Do not reveal account existence or state.
	}
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		return s.issueEmailToken(ctx, tx, user, purposeEmailVerification, "/verify-email", notification.KindEmailVerification, meta)
	})
}

// Login authenticates a user and creates a rotated session.
func (s *Service) Login(ctx context.Context, email, password string, meta RequestMeta) (*AuthResult, error) {
	normalized, ok := validate.Email(email)
	if !ok {
		return nil, apperr.Unauthorized(apperr.CodeInvalidCredentials, "Invalid email or password.")
	}
	throttleKey := meta.IPAddress + "|" + normalized
	if !s.loginLimiter.Allow(throttleKey) {
		return nil, apperr.TooManyRequests(apperr.CodeTooManyRequests, "Too many login attempts. Please try again later.")
	}

	user, err := s.repo.GetUserByEmail(ctx, s.db.Pool, normalized)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if user == nil {
		s.hasher.Verify(password, s.dummyHash) // equalise timing
		return nil, apperr.Unauthorized(apperr.CodeInvalidCredentials, "Invalid email or password.")
	}
	if !s.hasher.Verify(password, user.PasswordHash) {
		return nil, apperr.Unauthorized(apperr.CodeInvalidCredentials, "Invalid email or password.")
	}
	if user.AccountStatus != "active" {
		return nil, apperr.Forbidden(apperr.CodeAccountDisabled, "This account is not active.")
	}
	if !user.Verified() {
		return nil, apperr.Forbidden(apperr.CodeEmailUnverified, "Please verify your email address before signing in.")
	}

	result, err := s.createSession(ctx, user, meta)
	if err != nil {
		return nil, err
	}
	s.loginLimiter.Reset(throttleKey)
	return result, nil
}

// Refresh rotates a refresh token. Reuse of an already-rotated token revokes
// the whole token family.
func (s *Service) Refresh(ctx context.Context, refreshToken string, meta RequestMeta) (*AuthResult, error) {
	if refreshToken == "" {
		return nil, apperr.Unauthorized(apperr.CodeInvalidToken, "Refresh token is required.")
	}
	hash := auth.HashToken(refreshToken)
	var result *AuthResult
	err := s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		session, replacedBy, err := s.repo.LockSessionByRefreshHash(ctx, tx, hash)
		if err != nil {
			return apperr.Internal(err)
		}
		if session == nil {
			return apperr.Unauthorized(apperr.CodeInvalidToken, "The refresh token is invalid.")
		}
		if session.RevokedAt != nil || replacedBy != nil {
			if err := s.repo.RevokeFamily(ctx, tx, session.TokenFamilyID, "refresh_token_reuse"); err != nil {
				return apperr.Internal(err)
			}
			_ = s.audit.Record(ctx, tx, audit.Event{
				ActorUserID: &session.UserID, ActorType: "user", Action: "identity.refresh_reuse_detected",
				EntityType: "user_session", EntityID: &session.ID, RequestID: meta.RequestID,
				IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
			})
			return apperr.Unauthorized(apperr.CodeInvalidToken, "The refresh token is no longer valid.")
		}
		if session.ExpiresAt.Before(s.now()) {
			return apperr.Unauthorized(apperr.CodeExpiredToken, "The session has expired.")
		}
		user, err := s.repo.GetUserByID(ctx, tx, session.UserID)
		if err != nil {
			return apperr.Internal(err)
		}
		if user == nil || user.AccountStatus != "active" || !user.Verified() {
			return apperr.Forbidden(apperr.CodeAccountDisabled, "This account is not active.")
		}

		access, refresh, accessExp, refreshExp, err := s.newCredentials()
		if err != nil {
			return apperr.Internal(err)
		}
		newID, err := s.repo.CreateSession(ctx, tx, user.ID, auth.HashToken(access), accessExp,
			auth.HashToken(refresh), refreshExp, session.TokenFamilyID, optional(meta.IPAddress), optional(meta.UserAgent))
		if err != nil {
			return apperr.Internal(err)
		}
		if err := s.repo.MarkSessionRotated(ctx, tx, session.ID, newID, "rotated"); err != nil {
			return apperr.Internal(err)
		}
		result = &AuthResult{User: user, SessionID: newID, AccessToken: access, RefreshToken: refresh,
			AccessExpiresAt: accessExp, RefreshExpiresAt: refreshExp}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Logout revokes the current session.
func (s *Service) Logout(ctx context.Context, userID, sessionID string, meta RequestMeta) error {
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if _, err := s.repo.RevokeSession(ctx, tx, userID, sessionID, "logout"); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &userID, ActorType: "user", Action: "identity.logout",
			EntityType: "user_session", EntityID: &sessionID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// LogoutAll revokes every active session for the user.
func (s *Service) LogoutAll(ctx context.Context, userID string, meta RequestMeta) error {
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.repo.RevokeAllSessions(ctx, tx, userID, "logout_all"); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &userID, ActorType: "user", Action: "identity.logout_all",
			EntityType: "user", EntityID: &userID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// ListSessions returns safe metadata for active sessions.
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID string) ([]SessionView, error) {
	sessions, err := s.repo.ListSessions(ctx, s.db.Pool, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	views := make([]SessionView, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, SessionView{
			ID: session.ID, Current: session.ID == currentSessionID, IPAddress: session.IPAddress,
			UserAgent: session.UserAgent, CreatedAt: session.CreatedAt, LastUsedAt: session.LastUsedAt,
			ExpiresAt: session.ExpiresAt,
		})
	}
	return views, nil
}

// RevokeSession revokes a specific session owned by the user.
func (s *Service) RevokeSession(ctx context.Context, userID, sessionID string, meta RequestMeta) error {
	if !validate.UUID(sessionID) {
		return apperr.NotFound(apperr.CodeNotFound, "Session not found.")
	}
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		revoked, err := s.repo.RevokeSession(ctx, tx, userID, sessionID, "revoked_by_user")
		if err != nil {
			return apperr.Internal(err)
		}
		if !revoked {
			return apperr.NotFound(apperr.CodeNotFound, "Session not found.")
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &userID, ActorType: "user", Action: "identity.session_revoked",
			EntityType: "user_session", EntityID: &sessionID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// ForgotPassword always returns success to avoid account enumeration.
func (s *Service) ForgotPassword(ctx context.Context, email string, meta RequestMeta) error {
	normalized, ok := validate.Email(email)
	if !ok {
		return nil
	}
	if !s.resetLimiter.Allow("reset:" + normalized) {
		return nil
	}
	user, err := s.repo.GetUserByEmail(ctx, s.db.Pool, normalized)
	if err != nil {
		return apperr.Internal(err)
	}
	if user == nil || user.AccountStatus != "active" {
		return nil
	}
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		if err := s.issueEmailToken(ctx, tx, user, purposePasswordReset, "/reset-password", notification.KindPasswordReset, meta); err != nil {
			return err
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &user.ID, ActorType: "user", Action: "identity.password_reset_requested",
			EntityType: "user", EntityID: &user.ID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// ResetPassword consumes a reset token, updates the password and revokes all sessions.
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string, meta RequestMeta) error {
	if strings.TrimSpace(token) == "" {
		return apperr.BadRequest(apperr.CodeValidation, "Reset token is required.")
	}
	if err := auth.ValidatePassword(newPassword); err != nil {
		return apperr.BadRequest(apperr.CodeValidation, err.Error())
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return apperr.Internal(err)
	}
	tokenHash := auth.HashToken(token)
	return s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		userID, err := s.repo.ConsumeAuthToken(ctx, tx, tokenHash, purposePasswordReset)
		if err != nil {
			return s.tokenError(ctx, tx, tokenHash, purposePasswordReset, err)
		}
		if err := s.repo.UpdatePasswordHash(ctx, tx, userID, hash); err != nil {
			return apperr.Internal(err)
		}
		if err := s.repo.RevokeAllSessions(ctx, tx, userID, "password_reset"); err != nil {
			return apperr.Internal(err)
		}
		if err := s.repo.InvalidateAuthTokens(ctx, tx, userID, purposePasswordReset); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &userID, ActorType: "user", Action: "identity.password_reset",
			EntityType: "user", EntityID: &userID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
}

// Me returns the safe profile for the authenticated user.
func (s *Service) Me(ctx context.Context, userID string) (*Profile, error) {
	user, err := s.repo.GetUserByID(ctx, s.db.Pool, userID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if user == nil {
		return nil, apperr.NotFound(apperr.CodeNotFound, "User not found.")
	}
	return &Profile{
		ID: user.ID, Email: user.Email, FullName: user.FullName, Phone: user.PhoneE164,
		EmailVerified: user.Verified(), AccountStatus: user.AccountStatus,
		CreatedAt: user.CreatedAt, LastLoginAt: user.LastLoginAt,
	}, nil
}

// AuthenticateAccess validates an opaque access token and returns its user/session.
func (s *Service) AuthenticateAccess(ctx context.Context, accessToken string) (*SessionUser, error) {
	if accessToken == "" {
		return nil, apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required.")
	}
	su, err := s.repo.GetSessionByAccessHash(ctx, s.db.Pool, auth.HashToken(accessToken))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if su == nil {
		return nil, apperr.Unauthorized(apperr.CodeUnauthenticated, "The session is invalid or has expired.")
	}
	s.repo.TouchSession(ctx, s.db.Pool, su.Session.ID)
	return su, nil
}

func (s *Service) createSession(ctx context.Context, user *User, meta RequestMeta) (*AuthResult, error) {
	access, refresh, accessExp, refreshExp, err := s.newCredentials()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	familyID, err := newUUID()
	if err != nil {
		return nil, apperr.Internal(err)
	}
	var sessionID string
	err = s.db.WithTransaction(ctx, func(tx pgx.Tx) error {
		sessionID, err = s.repo.CreateSession(ctx, tx, user.ID, auth.HashToken(access), accessExp,
			auth.HashToken(refresh), refreshExp, familyID, optional(meta.IPAddress), optional(meta.UserAgent))
		if err != nil {
			return apperr.Internal(err)
		}
		if err := s.repo.RecordLogin(ctx, tx, user.ID); err != nil {
			return apperr.Internal(err)
		}
		return s.audit.Record(ctx, tx, audit.Event{
			ActorUserID: &user.ID, ActorType: "user", Action: "identity.login_succeeded",
			EntityType: "user_session", EntityID: &sessionID, RequestID: meta.RequestID,
			IPAddress: optional(meta.IPAddress), UserAgent: optional(meta.UserAgent),
		})
	})
	if err != nil {
		return nil, err
	}
	return &AuthResult{User: user, SessionID: sessionID, AccessToken: access, RefreshToken: refresh,
		AccessExpiresAt: accessExp, RefreshExpiresAt: refreshExp}, nil
}

func (s *Service) newCredentials() (access, refresh string, accessExp, refreshExp time.Time, err error) {
	access, err = auth.NewToken()
	if err != nil {
		return
	}
	refresh, err = auth.NewToken()
	if err != nil {
		return
	}
	now := s.now()
	accessExp = now.Add(s.authCfg.AccessTokenTTL)
	refreshExp = now.Add(s.authCfg.RefreshTokenTTL)
	return
}

func (s *Service) issueEmailToken(ctx context.Context, tx pgx.Tx, user *User, purpose, linkPath, kind string, meta RequestMeta) error {
	if err := s.repo.InvalidateAuthTokens(ctx, tx, user.ID, purpose); err != nil {
		return apperr.Internal(err)
	}
	raw, err := auth.NewToken()
	if err != nil {
		return apperr.Internal(err)
	}
	ttl := s.authCfg.EmailTokenTTL
	if purpose == purposePasswordReset {
		ttl = s.authCfg.PasswordResetTTL
	}
	tokenID, err := s.repo.CreateAuthToken(ctx, tx, user.ID, purpose, auth.HashToken(raw), s.now().Add(ttl))
	if err != nil {
		return apperr.Internal(err)
	}
	encrypted, err := s.encryptor.Encrypt(raw)
	if err != nil {
		return apperr.Internal(err)
	}
	payload, err := notification.Encode(notification.Payload{
		Kind: kind, To: user.Email, RecipientName: user.FullName, TokenEnc: encrypted, LinkPath: linkPath,
	})
	if err != nil {
		return apperr.Internal(err)
	}
	eventType := outbox.EventEmailVerification
	if purpose == purposePasswordReset {
		eventType = outbox.EventPasswordReset
	}
	return s.outbox.Enqueue(ctx, tx, outbox.Event{
		AggregateType: "user", AggregateID: user.ID, EventType: eventType,
		Payload: payload, DeduplicationKey: fmt.Sprintf("%s:%s", eventType, tokenID),
	})
}

func (s *Service) tokenError(ctx context.Context, q database.Querier, hash []byte, purpose string, err error) error {
	if !errors.Is(err, pgx.ErrNoRows) {
		return apperr.Internal(err)
	}
	exists, consumed, expired, statusErr := s.repo.AuthTokenStatus(ctx, q, hash, purpose)
	if statusErr != nil {
		return apperr.Internal(statusErr)
	}
	switch {
	case !exists:
		return apperr.BadRequest(apperr.CodeInvalidToken, "The token is invalid.")
	case consumed:
		return apperr.BadRequest(apperr.CodeUsedToken, "The token has already been used.")
	case expired:
		return apperr.BadRequest(apperr.CodeExpiredToken, "The token has expired.")
	default:
		return apperr.BadRequest(apperr.CodeInvalidToken, "The token is invalid.")
	}
}

func optional(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
