package authorization

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backend/internal/modules/identity"
	"backend/internal/platform/auth"
	"backend/internal/platform/database"
	"backend/internal/platform/middleware"
	"backend/internal/shared/apperr"
	"backend/internal/shared/response"
	"backend/internal/shared/validate"
)

// TokenAuthenticator validates opaque access tokens.
type TokenAuthenticator interface {
	AuthenticateAccess(ctx context.Context, token string) (*identity.SessionUser, error)
}

// Middleware enforces authentication, tenant/company resolution and permissions.
type Middleware struct {
	q           database.Querier
	repo        *Repository
	authn       TokenAuthenticator
	adminEmails map[string]struct{}
}

func NewMiddleware(q database.Querier, repo *Repository, authn TokenAuthenticator, adminEmails []string) *Middleware {
	admins := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		admins[strings.ToLower(strings.TrimSpace(email))] = struct{}{}
	}
	return &Middleware{q: q, repo: repo, authn: authn, adminEmails: admins}
}

// Authenticate resolves the bearer token into a trusted identity.
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
			return
		}
		su, err := m.authn.AuthenticateAccess(r.Context(), token)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
			return
		}
		_, isAdmin := m.adminEmails[strings.ToLower(su.User.Email)]
		identity := &auth.Identity{
			UserID: su.User.ID, SessionID: su.Session.ID, Email: su.User.Email,
			FullName: su.User.FullName, PlatformAdmin: isAdmin,
			Permissions: map[string]struct{}{}, SessionExpiry: su.Session.ExpiresAt,
		}
		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	})
}

// RequireTenant resolves an active membership for the {tenantID} route param.
func (m *Middleware) RequireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := auth.FromContext(r.Context())
		if !ok {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
			return
		}
		tenantID := chi.URLParam(r, "tenantID")
		if !validate.UUID(tenantID) {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.NotFound(apperr.CodeNotFound, "Workspace not found."))
			return
		}
		mc, err := m.repo.ResolveMembership(r.Context(), m.q, tenantID, identity.UserID)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), apperr.Internal(err))
			return
		}
		if mc == nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Forbidden(apperr.CodeForbidden, "You do not have access to this workspace."))
			return
		}
		if mc.TenantStatus != "active" && mc.TenantStatus != "trial" {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Forbidden(apperr.CodeTenantSuspended, "This workspace is not available."))
			return
		}
		permissions, err := m.repo.ResolvePermissions(r.Context(), m.q, tenantID, mc.MembershipID, "", "")
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), apperr.Internal(err))
			return
		}
		identity.TenantID = tenantID
		identity.MembershipID = mc.MembershipID
		identity.Permissions = permissions
		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	})
}

// RequireCompany resolves a company's tenant and the caller's membership.
func (m *Middleware) RequireCompany(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := auth.FromContext(r.Context())
		if !ok {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
			return
		}
		companyID := chi.URLParam(r, "companyID")
		if !validate.UUID(companyID) {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.NotFound(apperr.CodeNotFound, "Company not found."))
			return
		}
		tenantID, _, found, err := m.repo.GetCompanyTenant(r.Context(), m.q, companyID)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), apperr.Internal(err))
			return
		}
		if !found {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.NotFound(apperr.CodeNotFound, "Company not found."))
			return
		}
		mc, err := m.repo.ResolveMembership(r.Context(), m.q, tenantID, identity.UserID)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), apperr.Internal(err))
			return
		}
		if mc == nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Forbidden(apperr.CodeForbidden, "You do not have access to this company."))
			return
		}
		if mc.TenantStatus != "active" && mc.TenantStatus != "trial" {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Forbidden(apperr.CodeTenantSuspended, "This workspace is not available."))
			return
		}
		permissions, err := m.repo.ResolvePermissions(r.Context(), m.q, tenantID, mc.MembershipID, companyID, "")
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), apperr.Internal(err))
			return
		}
		identity.TenantID = tenantID
		identity.CompanyID = companyID
		identity.MembershipID = mc.MembershipID
		identity.Permissions = permissions
		next.ServeHTTP(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	})
}

// RequirePermission denies unless the identity holds the permission.
func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := auth.FromContext(r.Context())
			if !ok {
				response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
					apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
				return
			}
			if !identity.Has(permission) {
				response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
					apperr.Forbidden(apperr.CodeInvalidPermission, "You do not have permission to perform this action."))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePlatformAdmin restricts internal platform administration.
func (m *Middleware) RequirePlatformAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := auth.FromContext(r.Context())
		if !ok || !identity.PlatformAdmin {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Forbidden(apperr.CodeForbidden, "Platform administration access is required."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
