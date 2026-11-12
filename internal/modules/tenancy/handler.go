package tenancy

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/platform/auth"
	"backend/internal/platform/middleware"
	"backend/internal/shared/apperr"
	"backend/internal/shared/response"
)

// Handler exposes workspace and membership endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createWorkspaceRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createWorkspaceRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	workspace, err := h.svc.CreateWorkspace(r.Context(), identity.UserID, req.Name, req.Slug,
		r.Header.Get("Idempotency-Key"), meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated, map[string]any{"workspace": workspace})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	workspaces, err := h.svc.ListWorkspaces(r.Context(), identity.UserID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"workspaces": workspaces})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	workspace, err := h.svc.GetWorkspace(r.Context(), identity.TenantID, identity.UserID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"workspace": workspace})
}

type updateWorkspaceRequest struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
	Locale   string `json:"locale"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateWorkspaceRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	workspace, err := h.svc.UpdateWorkspace(r.Context(), identity.TenantID, identity.UserID,
		UpdateWorkspaceInput{Name: req.Name, Timezone: req.Timezone, Locale: req.Locale}, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"workspace": workspace})
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	members, err := h.svc.ListMembers(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"members": members})
}

func (h *Handler) GetMember(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	member, err := h.svc.GetMember(r.Context(), identity.TenantID, chi.URLParam(r, "membershipID"))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"member": member})
}

type updateMemberRequest struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	var req updateMemberRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	member, err := h.svc.UpdateMember(r.Context(), identity.TenantID, chi.URLParam(r, "membershipID"),
		identity.UserID, UpdateMemberInput{Status: req.Status}, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"member": member})
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	membershipID := chi.URLParam(r, "membershipID")
	if identity.MembershipID == membershipID {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Forbidden("self_removal", "You cannot remove your own membership."))
		return
	}
	if err := h.svc.RemoveMember(r.Context(), identity.TenantID, membershipID, identity.UserID, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func meta(r *http.Request) RequestMeta {
	return RequestMeta{
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: middleware.RequestIDFromContext(r.Context()),
	}
}
