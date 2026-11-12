package invitations

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/platform/auth"
	"backend/internal/platform/middleware"
	"backend/internal/shared/response"
)

// Handler exposes invitation endpoints.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type createInvitationRequest struct {
	Email   string `json:"email"`
	RoleID  string `json:"role_id"`
	Message string `json:"message"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInvitationRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	invitation, err := h.svc.Create(r.Context(), identity.TenantID, identity.UserID, identity.FullName,
		CreateInput{Email: req.Email, RoleID: req.RoleID, Message: req.Message}, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated,
		map[string]any{"invitation": invitation})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	list, err := h.svc.List(r.Context(), identity.TenantID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"invitations": list})
}

func (h *Handler) Resend(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	invitation, err := h.svc.Resend(r.Context(), identity.TenantID,
		chi.URLParam(r, "invitationID"), identity.UserID, identity.FullName, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"invitation": invitation})
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	if err := h.svc.Revoke(r.Context(), identity.TenantID,
		chi.URLParam(r, "invitationID"), identity.UserID, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

type acceptInvitationRequest struct {
	Token string `json:"token"`
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	var req acceptInvitationRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	result, err := h.svc.Accept(r.Context(), identity.UserID, identity.Email, req.Token, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"membership": result})
}

func meta(r *http.Request) RequestMeta {
	return RequestMeta{
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: middleware.RequestIDFromContext(r.Context()),
	}
}
