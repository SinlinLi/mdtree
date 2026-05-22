package api

import (
	"log/slog"
	"net/http"

	"github.com/SinlinLi/mdtree/internal/auth"
)

// Login authenticates a password and starts a session.
//
//	POST /api/auth/login  {"password": "..."}
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.Limiter.Allowed(ip) {
		w.Header().Set("Retry-After", "60")
		h.Log.Warn("login rate limited", slog.String("ip", ip))
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if !auth.VerifyPassword(h.PasswordHash, req.Password) {
		h.Limiter.Fail(ip)
		h.Log.Warn("failed login attempt", slog.String("ip", ip))
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}

	h.Limiter.Reset(ip)
	sess, err := h.Sessions.Create()
	if err != nil {
		h.Log.Error("create session failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}
	h.Auth.SetCookie(w, sess)
	h.Log.Info("login succeeded", slog.String("ip", ip))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Logout ends the current session.
//
//	POST /api/auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := h.Auth.Token(r); token != "" {
		h.Sessions.Delete(token)
	}
	h.Auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// Status reports whether the request carries a valid session.
//
//	GET /api/auth/status
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": h.Auth.Authenticated(r),
	})
}
