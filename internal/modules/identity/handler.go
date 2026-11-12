package identity

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"backend/internal/platform/auth"
	"backend/internal/platform/config"
	"backend/internal/platform/middleware"
	"backend/internal/shared/apperr"
	"backend/internal/shared/response"
)

// Handler exposes identity HTTP endpoints.
type Handler struct {
	svc *Service
	cfg config.AuthConfig
}

func NewHandler(svc *Service, cfg config.AuthConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

type registerRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if err := h.svc.Register(r.Context(), req.FullName, req.Email, req.Password, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusCreated, map[string]string{
		"message": "Registration successful. Check your email to verify your address.",
	})
}

type tokenRequest struct {
	Token string `json:"token"`
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if err := h.svc.VerifyEmail(r.Context(), req.Token); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]string{
		"message": "Email verified successfully.",
	})
}

type emailRequest struct {
	Email string `json:"email"`
}

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if err := h.svc.ResendVerification(r.Context(), req.Email, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusAccepted, map[string]string{
		"message": "If the account exists and requires verification, an email has been sent.",
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	result, err := h.svc.Login(r.Context(), req.Email, req.Password, meta(r))
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.setAuthCookies(w, result)
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, h.authPayload(result))
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if r.ContentLength != 0 {
		if err := response.Decode(w, r, &req); err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
			return
		}
	}

	fromCookie := false
	if req.RefreshToken == "" {
		cookie, err := r.Cookie(h.cfg.RefreshCookieName)
		if err != nil {
			response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
				apperr.Unauthorized(apperr.CodeInvalidToken, "Refresh token is required."))
			return
		}
		req.RefreshToken = cookie.Value
		fromCookie = true
	}
	if fromCookie && !h.csrfValid(r) {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Forbidden("csrf_failed", "A valid CSRF token is required."))
		return
	}

	result, err := h.svc.Refresh(r.Context(), req.RefreshToken, meta(r))
	if err != nil {
		h.clearAuthCookies(w)
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.setAuthCookies(w, result)
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, h.authPayload(result))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
		return
	}
	if err := h.svc.Logout(r.Context(), identity.UserID, identity.SessionID, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.clearAuthCookies(w)
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
		return
	}
	if err := h.svc.LogoutAll(r.Context(), identity.UserID, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	h.clearAuthCookies(w)
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
		return
	}
	sessions, err := h.svc.ListSessions(r.Context(), identity.UserID, identity.SessionID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"sessions": sessions})
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
		return
	}
	if err := h.svc.RevokeSession(r.Context(), identity.UserID, chi.URLParam(r, "sessionID"), meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req emailRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if err := h.svc.ForgotPassword(r.Context(), req.Email, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusAccepted, map[string]string{
		"message": "If the account exists, a password reset email has been sent.",
	})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := response.Decode(w, r, &req); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	if err := h.svc.ResetPassword(r.Context(), req.Token, req.NewPassword, meta(r)); err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusNoContent, nil)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()),
			apperr.Unauthorized(apperr.CodeUnauthenticated, "Authentication is required."))
		return
	}
	profile, err := h.svc.Me(r.Context(), identity.UserID)
	if err != nil {
		response.WriteError(w, middleware.RequestIDFromContext(r.Context()), err)
		return
	}
	response.JSON(w, middleware.RequestIDFromContext(r.Context()), http.StatusOK, map[string]any{"user": profile})
}

func (h *Handler) authPayload(result *AuthResult) map[string]any {
	return map[string]any{
		"access_token":       result.AccessToken,
		"refresh_token":      result.RefreshToken,
		"token_type":         "Bearer",
		"expires_in":         int(time.Until(result.AccessExpiresAt).Seconds()),
		"refresh_expires_in": int(time.Until(result.RefreshExpiresAt).Seconds()),
		"session_id":         result.SessionID,
		"user": map[string]any{
			"id": result.User.ID, "email": result.User.Email, "full_name": result.User.FullName,
			"email_verified": result.User.Verified(),
		},
	}
}

func (h *Handler) cookieSameSite() http.SameSite {
	switch h.cfg.CookieSameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *Handler) setAuthCookies(w http.ResponseWriter, result *AuthResult) {
	sameSite := h.cookieSameSite()
	http.SetCookie(w, &http.Cookie{
		Name: h.cfg.RefreshCookieName, Value: result.RefreshToken, Path: "/api/v1/auth",
		Domain: h.cfg.CookieDomain, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: sameSite,
		Expires: result.RefreshExpiresAt, MaxAge: int(time.Until(result.RefreshExpiresAt).Seconds()),
	})
	csrf, err := auth.NewToken()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "veyro_csrf", Value: csrf, Path: "/api/v1/auth", Domain: h.cfg.CookieDomain,
		HttpOnly: false, Secure: h.cfg.CookieSecure, SameSite: sameSite,
		Expires: result.RefreshExpiresAt, MaxAge: int(time.Until(result.RefreshExpiresAt).Seconds()),
	})
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{h.cfg.RefreshCookieName, "veyro_csrf"} {
		http.SetCookie(w, &http.Cookie{
			Name: name, Value: "", Path: "/api/v1/auth", Domain: h.cfg.CookieDomain,
			HttpOnly: name == h.cfg.RefreshCookieName, Secure: h.cfg.CookieSecure,
			SameSite: h.cookieSameSite(), MaxAge: -1, Expires: time.Unix(1, 0),
		})
	}
}

func (h *Handler) csrfValid(r *http.Request) bool {
	header := r.Header.Get("X-CSRF-Token")
	cookie, err := r.Cookie("veyro_csrf")
	if err != nil || header == "" || cookie.Value == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) == 1
}

func meta(r *http.Request) RequestMeta {
	return RequestMeta{
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: middleware.RequestIDFromContext(r.Context()),
	}
}
