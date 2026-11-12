package authorization

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/platform/audit"
	"backend/internal/platform/auth"
	"backend/internal/platform/middleware"
	"backend/internal/shared/response"
)

// Handler exposes role, permission and assignment endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	permissions, err := h.svc.ListPermissions(r.Context())
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"permissions": permissions})
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	roles, err := h.svc.ListRoles(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"roles": roles})
}

func (h *Handler) GetRole(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	role, err := h.svc.GetRole(r.Context(), identity.TenantID, chi.URLParam(r, "roleID"))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"role": role})
}

type createRoleRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	role, err := h.svc.CreateRole(r.Context(), identity.TenantID, actor(r), CreateRoleInput{
		Code: req.Code, Name: req.Name, Description: req.Description, Permissions: req.Permissions,
	})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated, map[string]any{"role": role})
}

type updateRoleRequest struct {
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	Permissions *[]string `json:"permissions"`
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	var req updateRoleRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	role, err := h.svc.UpdateRole(r.Context(), identity.TenantID, chi.URLParam(r, "roleID"), actor(r), UpdateRoleInput{
		Name: req.Name, Description: req.Description, Permissions: req.Permissions,
	})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"role": role})
}

func (h *Handler) ListAssignments(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	assignments, err := h.svc.ListAssignments(r.Context(), identity.TenantID, chi.URLParam(r, "membershipID"))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"assignments": assignments})
}

type assignRoleRequest struct {
	RoleID    string `json:"role_id"`
	ScopeType string `json:"scope_type"`
	CompanyID string `json:"company_id"`
	BranchID  string `json:"branch_id"`
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	var req assignRoleRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	id, err := h.svc.AssignRole(r.Context(), identity.TenantID, chi.URLParam(r, "membershipID"), req.RoleID,
		identity.UserID, actor(r), AssignRoleInput{ScopeType: req.ScopeType, CompanyID: req.CompanyID, BranchID: req.BranchID})
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated, map[string]any{"assignment_id": id})
}

func (h *Handler) RevokeAssignment(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	if err := h.svc.RevokeAssignment(r.Context(), identity.TenantID, chi.URLParam(r, "membershipID"),
		chi.URLParam(r, "assignmentID"), identity.UserID, actor(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func actor(r *http.Request) *audit.Event {
	identity, _ := auth.FromContext(r.Context())
	return AuditActor(identity.UserID, middleware.RequestIDFromContext(r.Context()),
		middleware.ClientIP(r), r.UserAgent())
}
