package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/modules/authorization"
	"backend/internal/modules/identity"
	"backend/internal/modules/invitations"
	"backend/internal/modules/organization"
	"backend/internal/modules/platformadmin"
	"backend/internal/modules/tenancy"
	"backend/internal/platform/config"
	"backend/internal/platform/database"
	"backend/internal/platform/logger"
	"backend/internal/platform/middleware"
	"backend/internal/shared/response"
)

// Dependencies is the explicit set of HTTP handlers and services the router
// needs. It keeps route composition in one place.
type Dependencies struct {
	Config        *config.Config
	Logger        *logger.Logger
	DB            *database.Database
	Authz         *authorization.Middleware
	Identity      *identity.Handler
	Authorization *authorization.Handler
	Tenancy       *tenancy.Handler
	Organization  *organization.Handler
	Invitations   *invitations.Handler
	Platform      *platformadmin.Handler
}

// NewRouter builds the complete API router with deliberate middleware order.
func NewRouter(deps Dependencies) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AccessLog(deps.Logger))
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS(deps.Config.HTTP.AllowedOrigins))

	r.Get("/health", liveness)
	r.Get("/ready", database.ReadyHandler(deps.DB))
	r.Get("/api/v1/health", liveness)

	r.Route("/api/v1", func(api chi.Router) {
		api.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", deps.Identity.Register)
			auth.Post("/verify-email", deps.Identity.VerifyEmail)
			auth.Post("/verify-email/resend", deps.Identity.ResendVerification)
			auth.Post("/login", deps.Identity.Login)
			auth.Post("/refresh", deps.Identity.Refresh)
			auth.Post("/password/forgot", deps.Identity.ForgotPassword)
			auth.Post("/password/reset", deps.Identity.ResetPassword)
			auth.Group(func(authed chi.Router) {
				authed.Use(deps.Authz.Authenticate)
				authed.Get("/me", deps.Identity.Me)
				authed.Post("/logout", deps.Identity.Logout)
				authed.Post("/logout-all", deps.Identity.LogoutAll)
				authed.Get("/sessions", deps.Identity.ListSessions)
				authed.Delete("/sessions/{sessionID}", deps.Identity.DeleteSession)
			})
		})

		api.Group(func(authed chi.Router) {
			authed.Use(deps.Authz.Authenticate)

			authed.Post("/invitations/accept", deps.Invitations.Accept)
			authed.Post("/workspaces", deps.Tenancy.Create)
			authed.Get("/workspaces", deps.Tenancy.List)

			authed.Route("/workspaces/{tenantID}", func(ws chi.Router) {
				ws.Use(deps.Authz.RequireTenant)

				ws.Get("/", deps.Tenancy.Get)
				ws.With(authorization.RequirePermission("workspace.update")).Patch("/", deps.Tenancy.Update)
				ws.With(authorization.RequirePermission("workspace.view")).Get("/overview", deps.Organization.Overview)
				ws.With(authorization.RequirePermission("workspace.view")).
					Get("/onboarding/status", deps.Organization.OnboardingStatus)

				ws.Get("/members", require("workspace.members.view", deps.Tenancy.ListMembers))
				ws.Get("/members/{membershipID}", require("workspace.members.view", deps.Tenancy.GetMember))
				ws.Patch("/members/{membershipID}", require("workspace.members.manage", deps.Tenancy.UpdateMember))
				ws.Delete("/members/{membershipID}", require("workspace.members.manage", deps.Tenancy.RemoveMember))
				ws.Get("/members/{membershipID}/roles",
					require("workspace.members.view", deps.Authorization.ListAssignments))
				ws.Post("/members/{membershipID}/roles",
					require("workspace.members.manage", deps.Authorization.AssignRole))
				ws.Delete("/members/{membershipID}/roles/{assignmentID}",
					require("workspace.members.manage", deps.Authorization.RevokeAssignment))

				ws.Get("/roles", require("workspace.roles.view", deps.Authorization.ListRoles))
				ws.Post("/roles", require("workspace.roles.manage", deps.Authorization.CreateRole))
				ws.Get("/roles/{roleID}", require("workspace.roles.view", deps.Authorization.GetRole))
				ws.Patch("/roles/{roleID}", require("workspace.roles.manage", deps.Authorization.UpdateRole))
				ws.Get("/permissions", require("workspace.roles.view", deps.Authorization.ListPermissions))

				ws.Get("/invitations", require("workspace.members.view", deps.Invitations.List))
				ws.Post("/invitations", require("workspace.members.invite", deps.Invitations.Create))
				ws.Post("/invitations/{invitationID}/resend",
					require("workspace.members.invite", deps.Invitations.Resend))
				ws.Delete("/invitations/{invitationID}",
					require("workspace.members.manage", deps.Invitations.Revoke))

				ws.Post("/companies", require("company.create", deps.Organization.CreateCompany))
				ws.Get("/company", require("company.view", deps.Organization.GetCompany))
				ws.Patch("/company", require("company.update", deps.Organization.UpdateCompany))
				ws.Get("/company/settings", require("company.view", deps.Organization.GetSettings))
				ws.Patch("/company/settings",
					require("company.settings.manage", deps.Organization.UpdateSettings))
				ws.Get("/company/tax-registrations",
					require("company.view", deps.Organization.ListTaxRegistrations))
				ws.Post("/company/tax-registrations",
					require("company.tax.manage", deps.Organization.CreateTaxRegistration))
				ws.Patch("/company/tax-registrations/{registrationID}",
					require("company.tax.manage", deps.Organization.UpdateTaxRegistration))
			})

			authed.Route("/platform", func(platform chi.Router) {
				platform.Use(deps.Authz.RequirePlatformAdmin)
				platform.Get("/health", deps.Platform.Health)
				platform.Get("/tenants", deps.Platform.ListTenants)
				platform.Get("/tenants/{tenantID}", deps.Platform.GetTenant)
				platform.Patch("/tenants/{tenantID}", deps.Platform.UpdateTenantStatus)
			})
		})
	})

	return r
}

func require(permission string, handler http.HandlerFunc) http.HandlerFunc {
	wrapped := authorization.RequirePermission(permission)(handler)
	return wrapped.ServeHTTP
}

func liveness(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]string{"status": "ok", "service": "veyro-backend"})
}
