package api

import (
	"log/slog"
	"net/http"
	"strconv"
)

// Search ranks markdown files by filename against a query, using the
// prebuilt in-memory index.
//
//	GET /api/search?q=foo&limit=50
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	results := h.Index.Search(q, limit)
	h.Metrics.IncSearch()
	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"count":   len(results),
		"results": results,
	})
}

// Reindex rebuilds the search index from disk.
//
//	POST /api/search/reindex
func (h *Handler) Reindex(w http.ResponseWriter, r *http.Request) {
	count, elapsed, err := h.Index.Build()
	if err != nil {
		h.Log.Error("reindex failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "reindex failed")
		return
	}
	h.Metrics.SetIndex(count, elapsed)
	h.Log.Info("search index rebuilt", slog.Int("files", count), slog.Duration("elapsed", elapsed))
	writeJSON(w, http.StatusOK, map[string]any{
		"files":      count,
		"durationMs": float64(elapsed.Microseconds()) / 1000.0,
	})
}

// Stats returns runtime metrics and search index statistics.
//
//	GET /api/stats
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"metrics":  h.Metrics.Snapshot(),
		"index":    h.Index.Stats(),
		"sessions": h.Sessions.Count(),
	})
}
