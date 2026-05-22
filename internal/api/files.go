package api

import (
	"log/slog"
	"net/http"
)

// Tree lists the directories and markdown files in one directory. When no
// path is given, the configured root is listed.
//
//	GET /api/tree?path=/abs/dir
func (h *Handler) Tree(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = h.Root
	}
	listing, err := h.Lister.List(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, listing)
}

// GetFile returns the content of a markdown file.
//
//	GET /api/file?path=/abs/file.md
func (h *Handler) GetFile(w http.ResponseWriter, r *http.Request) {
	fc, err := h.Files.Read(r.URL.Query().Get("path"))
	if err != nil {
		writeFileError(w, err)
		return
	}
	h.Metrics.IncFileRead()
	writeJSON(w, http.StatusOK, fc)
}

// SaveFile overwrites an existing markdown file.
//
//	PUT /api/file  {"path": "...", "content": "..."}
func (h *Handler) SaveFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	info, err := h.Files.Save(req.Path, req.Content)
	if err != nil {
		writeFileError(w, err)
		return
	}
	h.Metrics.IncFileWrite()
	h.Log.Info("file saved", slog.String("path", info.Path), slog.Int64("size", info.Size))
	writeJSON(w, http.StatusOK, info)
}

// CreateFile creates a new markdown file.
//
//	POST /api/file  {"path": "...", "content": "..."}
func (h *Handler) CreateFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	info, err := h.Files.Create(req.Path, req.Content)
	if err != nil {
		writeFileError(w, err)
		return
	}
	h.Index.Add(info.Path)
	h.Metrics.IncFileWrite()
	h.Log.Info("file created", slog.String("path", info.Path))
	writeJSON(w, http.StatusCreated, info)
}

// DeleteFile removes a markdown file.
//
//	DELETE /api/file?path=/abs/file.md
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	abs, err := h.Files.ResolvePath(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if err := h.Files.Delete(path); err != nil {
		writeFileError(w, err)
		return
	}
	h.Index.Remove(abs)
	h.Log.Info("file deleted", slog.String("path", abs))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// RenameFile moves a markdown file to a new path.
//
//	POST /api/file/rename  {"from": "...", "to": "..."}
func (h *Handler) RenameFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	absFrom, err := h.Files.ResolvePath(req.From)
	if err != nil {
		writeFileError(w, err)
		return
	}
	info, err := h.Files.Rename(req.From, req.To)
	if err != nil {
		writeFileError(w, err)
		return
	}
	h.Index.Rename(absFrom, info.Path)
	h.Log.Info("file renamed", slog.String("from", absFrom), slog.String("to", info.Path))
	writeJSON(w, http.StatusOK, info)
}

// Mkdir creates a new directory.
//
//	POST /api/dir  {"path": "..."}
func (h *Handler) Mkdir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.Files.Mkdir(req.Path); err != nil {
		writeFileError(w, err)
		return
	}
	abs, _ := h.Files.ResolvePath(req.Path)
	h.Log.Info("directory created", slog.String("path", abs))
	writeJSON(w, http.StatusCreated, map[string]any{"path": abs})
}
