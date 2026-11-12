package identity

import "time"

// User is the global identity record. PasswordHash is never serialized.
type User struct {
	ID              string
	Email           string
	PhoneE164       *string
	PasswordHash    string
	FullName        string
	AccountStatus   string
	EmailVerifiedAt *time.Time
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	RowVersion      int64
}

// Verified reports whether the account email has been verified.
func (u *User) Verified() bool { return u != nil && u.EmailVerifiedAt != nil }

// Session is an authenticated device session.
type Session struct {
	ID              string
	UserID          string
	TokenFamilyID   string
	AccessExpiresAt *time.Time
	ExpiresAt       time.Time
	LastUsedAt      *time.Time
	RevokedAt       *time.Time
	IPAddress       *string
	UserAgent       *string
	CreatedAt       time.Time
}

// SessionView is safe session metadata returned to the user.
type SessionView struct {
	ID         string     `json:"id"`
	Current    bool       `json:"current"`
	IPAddress  *string    `json:"ip_address,omitempty"`
	UserAgent  *string    `json:"user_agent,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

// Profile is the safe authenticated-user payload.
type Profile struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	FullName      string     `json:"full_name"`
	Phone         *string    `json:"phone,omitempty"`
	EmailVerified bool       `json:"email_verified"`
	AccountStatus string     `json:"account_status"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

// AuthResult is returned by login and refresh.
type AuthResult struct {
	User             *User
	SessionID        string
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// SessionUser couples a session with its user for access-token authentication.
type SessionUser struct {
	Session Session
	User    User
}

// RequestMeta carries safe request metadata for sessions and audit.
type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}
