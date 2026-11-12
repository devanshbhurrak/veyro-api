package auth

import (
	"context"
	"time"
)

type identityKey struct{}

// Identity is the trusted, server-derived request identity. None of these
// values are ever taken from request bodies.
type Identity struct {
	UserID        string
	SessionID     string
	TenantID      string
	MembershipID  string
	CompanyID     string
	BranchID      string
	Email         string
	FullName      string
	Permissions   map[string]struct{}
	PlatformAdmin bool
	SessionExpiry time.Time
}

// Has reports whether the identity holds the given permission.
func (i *Identity) Has(permission string) bool {
	if i == nil {
		return false
	}
	_, ok := i.Permissions[permission]
	return ok
}

// FromContext returns the authenticated identity, if present.
func FromContext(ctx context.Context) (*Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(*Identity)
	return id, ok
}

// WithIdentity stores the trusted identity on the context.
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}
