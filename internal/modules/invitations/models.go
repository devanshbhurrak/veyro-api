package invitations

import "time"

// Invitation is a pending workspace membership invitation.
type Invitation struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	RoleID      string     `json:"role_id"`
	RoleCode    string     `json:"role_code"`
	RoleName    string     `json:"role_name"`
	Status      string     `json:"status"`
	Message     *string    `json:"message,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	InvitedBy   string     `json:"invited_by"`
	InviterName string     `json:"inviter_name,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// PendingInvitation is the authoritative record used during acceptance.
type PendingInvitation struct {
	ID           string
	TenantID     string
	TenantName   string
	TenantStatus string
	Email        string
	RoleID       string
	RoleCode     string
	Status       string
	ExpiresAt    time.Time
}

// RequestMeta carries request context for audit and email.
type RequestMeta struct {
	IPAddress string
	UserAgent string
	RequestID string
}
