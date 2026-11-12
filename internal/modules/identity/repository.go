package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"backend/internal/platform/database"
)

// Repository provides persistence for identity records.
type Repository struct{}

func NewRepository() *Repository { return &Repository{} }

const userColumns = `id, email::text, phone_e164, COALESCE(password_hash,''), full_name,
	account_status, email_verified_at, last_login_at, created_at, row_version`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PhoneE164, &u.PasswordHash, &u.FullName,
		&u.AccountStatus, &u.EmailVerifiedAt, &u.LastLoginAt, &u.CreatedAt, &u.RowVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, q database.Querier, email string) (*User, error) {
	return scanUser(q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email))
}

func (r *Repository) GetUserByID(ctx context.Context, q database.Querier, id string) (*User, error) {
	return scanUser(q.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
}

func (r *Repository) CreateUser(ctx context.Context, q database.Querier, email, fullName, passwordHash string) (*User, error) {
	return scanUser(q.QueryRow(ctx, `
		INSERT INTO users (email, full_name, password_hash, account_status)
		VALUES ($1, $2, $3, 'active')
		RETURNING `+userColumns, email, fullName, passwordHash))
}

func (r *Repository) MarkEmailVerified(ctx context.Context, q database.Querier, userID string) error {
	_, err := q.Exec(ctx, `
		UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()),
		    updated_at = now(), row_version = row_version + 1
		WHERE id = $1`, userID)
	return err
}

func (r *Repository) UpdatePasswordHash(ctx context.Context, q database.Querier, userID, hash string) error {
	_, err := q.Exec(ctx, `
		UPDATE users SET password_hash=$2, updated_at=now(), row_version=row_version+1
		WHERE id=$1`, userID, hash)
	return err
}

func (r *Repository) RecordLogin(ctx context.Context, q database.Querier, userID string) error {
	_, err := q.Exec(ctx, `
		UPDATE users SET last_login_at=now(), updated_at=now(), row_version=row_version+1
		WHERE id=$1`, userID)
	return err
}

// CreateAuthToken stores only the token hash.
func (r *Repository) CreateAuthToken(ctx context.Context, q database.Querier, userID, purpose string, hash []byte, expiresAt time.Time) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO auth_tokens (user_id, purpose, token_hash, expires_at)
		VALUES ($1,$2,$3,$4) RETURNING id`, userID, purpose, hash, expiresAt).Scan(&id)
	return id, err
}

// InvalidateAuthTokens marks outstanding tokens of a purpose consumed.
func (r *Repository) InvalidateAuthTokens(ctx context.Context, q database.Querier, userID, purpose string) error {
	_, err := q.Exec(ctx, `
		UPDATE auth_tokens SET consumed_at=now(), updated_at=now(), row_version=row_version+1
		WHERE user_id=$1 AND purpose=$2 AND consumed_at IS NULL`, userID, purpose)
	return err
}

// ConsumeAuthToken atomically consumes a valid, unexpired, single-use token.
func (r *Repository) ConsumeAuthToken(ctx context.Context, q database.Querier, hash []byte, purpose string) (string, error) {
	var userID string
	err := q.QueryRow(ctx, `
		UPDATE auth_tokens
		SET consumed_at=now(), updated_at=now(), row_version=row_version+1
		WHERE token_hash=$1 AND purpose=$2 AND consumed_at IS NULL AND expires_at > now()
		RETURNING user_id`, hash, purpose).Scan(&userID)
	return userID, err
}

// AuthTokenStatus classifies a token without consuming it.
func (r *Repository) AuthTokenStatus(ctx context.Context, q database.Querier, hash []byte, purpose string) (exists bool, consumed bool, expired bool, err error) {
	var consumedAt *time.Time
	var expiresAt time.Time
	err = q.QueryRow(ctx, `
		SELECT consumed_at, expires_at FROM auth_tokens WHERE token_hash=$1 AND purpose=$2`,
		hash, purpose).Scan(&consumedAt, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, false, nil
	}
	if err != nil {
		return false, false, false, err
	}
	return true, consumedAt != nil, time.Now().After(expiresAt), nil
}

func (r *Repository) CreateSession(ctx context.Context, q database.Querier, userID string, accessHash []byte, accessExpires time.Time, refreshHash []byte, refreshExpires time.Time, familyID string, ip, userAgent *string) (string, error) {
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO user_sessions
			(user_id, refresh_token_hash, access_token_hash, token_family_id, expires_at, access_expires_at, ip_address, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		userID, refreshHash, accessHash, familyID, refreshExpires, accessExpires, ip, userAgent).Scan(&id)
	return id, err
}

const sessionColumns = `id, user_id, token_family_id, access_expires_at, expires_at, last_used_at, revoked_at, ip_address::text, user_agent, created_at`

func scanSession(row pgx.Row) (*Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.UserID, &s.TokenFamilyID, &s.AccessExpiresAt, &s.ExpiresAt,
		&s.LastUsedAt, &s.RevokedAt, &s.IPAddress, &s.UserAgent, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetSessionByAccessHash returns an active session and its active, verified user.
func (r *Repository) GetSessionByAccessHash(ctx context.Context, q database.Querier, hash []byte) (*SessionUser, error) {
	row := q.QueryRow(ctx, `
		SELECT s.id, s.user_id, s.token_family_id, s.access_expires_at, s.expires_at,
		       s.last_used_at, s.revoked_at, s.ip_address::text, s.user_agent, s.created_at,
		       u.id, u.email::text, u.phone_e164, COALESCE(u.password_hash,''), u.full_name,
		       u.account_status, u.email_verified_at, u.last_login_at, u.created_at, u.row_version
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.access_token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND s.access_expires_at > now()
		  AND u.account_status = 'active'
		  AND u.email_verified_at IS NOT NULL`, hash)
	var su SessionUser
	err := row.Scan(&su.Session.ID, &su.Session.UserID, &su.Session.TokenFamilyID,
		&su.Session.AccessExpiresAt, &su.Session.ExpiresAt, &su.Session.LastUsedAt,
		&su.Session.RevokedAt, &su.Session.IPAddress, &su.Session.UserAgent, &su.Session.CreatedAt,
		&su.User.ID, &su.User.Email, &su.User.PhoneE164, &su.User.PasswordHash, &su.User.FullName,
		&su.User.AccountStatus, &su.User.EmailVerifiedAt, &su.User.LastLoginAt, &su.User.CreatedAt, &su.User.RowVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &su, nil
}

// LockSessionByRefreshHash locks the session row for rotation. It returns
// revoked/replaced sessions so reuse can be detected.
func (r *Repository) LockSessionByRefreshHash(ctx context.Context, q database.Querier, hash []byte) (*Session, *string, error) {
	var s Session
	var replacedBy *string
	err := q.QueryRow(ctx, `
		SELECT `+sessionColumns+`, replaced_by_session_id
		FROM user_sessions WHERE refresh_token_hash = $1 FOR UPDATE`, hash).Scan(
		&s.ID, &s.UserID, &s.TokenFamilyID, &s.AccessExpiresAt, &s.ExpiresAt,
		&s.LastUsedAt, &s.RevokedAt, &s.IPAddress, &s.UserAgent, &s.CreatedAt, &replacedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &s, replacedBy, nil
}

func (r *Repository) MarkSessionRotated(ctx context.Context, q database.Querier, sessionID, replacementID, reason string) error {
	_, err := q.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at=now(), revoke_reason=$3, replaced_by_session_id=$2,
		    updated_at=now(), row_version=row_version+1
		WHERE id=$1`, sessionID, replacementID, reason)
	return err
}

func (r *Repository) RevokeSession(ctx context.Context, q database.Querier, userID, sessionID, reason string) (bool, error) {
	tag, err := q.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at=now(), revoke_reason=$3, updated_at=now(), row_version=row_version+1
		WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, sessionID, userID, reason)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Repository) RevokeAllSessions(ctx context.Context, q database.Querier, userID, reason string) error {
	_, err := q.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at=now(), revoke_reason=$2, updated_at=now(), row_version=row_version+1
		WHERE user_id=$1 AND revoked_at IS NULL`, userID, reason)
	return err
}

func (r *Repository) RevokeFamily(ctx context.Context, q database.Querier, familyID, reason string) error {
	_, err := q.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at=now(), revoke_reason=$2, updated_at=now(), row_version=row_version+1
		WHERE token_family_id=$1 AND revoked_at IS NULL`, familyID, reason)
	return err
}

func (r *Repository) ListSessions(ctx context.Context, q database.Querier, userID string) ([]Session, error) {
	rows, err := q.Query(ctx, `
		SELECT `+sessionColumns+` FROM user_sessions
		WHERE user_id=$1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sessions []Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, *s)
	}
	return sessions, rows.Err()
}

// TouchSession records activity at most once per minute to avoid a write on
// every authenticated request.
func (r *Repository) TouchSession(ctx context.Context, q database.Querier, sessionID string) {
	_, _ = q.Exec(ctx, `
		UPDATE user_sessions SET last_used_at=now()
		WHERE id=$1 AND (last_used_at IS NULL OR last_used_at < now() - interval '1 minute')`, sessionID)
}
