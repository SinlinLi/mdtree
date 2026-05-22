// Package api implements mdtree's HTTP JSON handlers.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/SinlinLi/mdtree/internal/auth"
	"github.com/SinlinLi/mdtree/internal/files"
	"github.com/SinlinLi/mdtree/internal/metrics"
	"github.com/SinlinLi/mdtree/internal/search"
)

// Handler holds the dependencies shared by all API endpoints.
type Handler struct {
	Files        *files.Manager
	Lister       *files.Lister
	Index        *search.Index
	Auth         *auth.Authenticator
	Sessions     *auth.SessionStore
	Limiter      *auth.RateLimiter
	Metrics      *metrics.Metrics
	PasswordHash string
	Root         string
	Log          *slog.Logger
}

// maxRequestBody caps decoded JSON request bodies. It allows headroom above
// MaxFileSize for JSON string escaping of file content.
const maxRequestBody = files.MaxFileSize*2 + (1 << 20)

// writeJSON encodes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// writeError sends a JSON error body with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeFileError maps a files-package sentinel error to an HTTP response.
func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrInvalidPath),
		errors.Is(err, files.ErrNotMarkdown),
		errors.Is(err, files.ErrNotDir),
		errors.Is(err, files.ErrIsDir):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, files.ErrOutsideRoot):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, files.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, files.ErrExists):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, files.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// decodeJSON reads a size-capped JSON request body into v, rejecting unknown
// fields. It writes a 400 response and returns false on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

// clientIP returns a best-effort client identifier for rate limiting.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
