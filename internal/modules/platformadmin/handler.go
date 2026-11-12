package platformadmin

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"backend/internal/platform/auth"
	"backend/internal/platform/middleware"
	"backend/internal/shared/response"
)

// Handler exposes the limited platform administration API.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	tenants, total, err := h.svc.ListTenants(r.Context(), r.URL.Query().Get("query"),
		r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"tenants": tenants, "total": total})
}

func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	tenant, err := h.svc.GetTenant(r.Context(), chi.URLParam(r, "tenantID"))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"tenant": tenant})
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateTenantStatus(w http.ResponseWriter, r *http.Request) {
	var req updateStatusRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	identity, _ := auth.FromContext(r.Context())
	if err := h.svc.UpdateTenantStatus(r.Context(), chi.URLParam(r, "tenantID"),
		req.Status, identity.UserID); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	tenant, err := h.svc.GetTenant(r.Context(), chi.URLParam(r, "tenantID"))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK,
		map[string]any{"tenant": tenant})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.Health(r.Context())
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	code := http.StatusOK
	if status.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), code, status)
}
