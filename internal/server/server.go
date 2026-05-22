// Package server wires the HTTP router, middleware and lifecycle for mdtree.
package server

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/SinlinLi/mdtree/internal/api"
	"github.com/SinlinLi/mdtree/internal/auth"
	"github.com/SinlinLi/mdtree/internal/metrics"
)

// Config holds everything needed to build a Server.
type Config struct {
	Addr    string
	API     *api.Handler
	Auth    *auth.Authenticator
	Metrics *metrics.Metrics
	DistFS  fs.FS
	Log     *slog.Logger
}

// Server is the mdtree HTTP server.
type Server struct {
	httpServer *http.Server
	log        *slog.Logger
}

// New builds a Server with all routes and middleware wired up.
func New(cfg Config) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cfg.Metrics.Middleware)
	r.Use(requestLogger(cfg.Log))

	// Health check — intentionally unauthenticated for load balancers.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api", func(r chi.Router) {
		// Public authentication endpoints.
		r.Post("/auth/login", cfg.API.Login)
		r.Post("/auth/logout", cfg.API.Logout)
		r.Get("/auth/status", cfg.API.Status)

		// Everything else requires a valid session.
		r.Group(func(r chi.Router) {
			r.Use(cfg.Auth.Middleware)
			r.Get("/tree", cfg.API.Tree)
			r.Get("/file", cfg.API.GetFile)
			r.Post("/file", cfg.API.CreateFile)
			r.Put("/file", cfg.API.SaveFile)
			r.Delete("/file", cfg.API.DeleteFile)
			r.Post("/file/rename", cfg.API.RenameFile)
			r.Post("/dir", cfg.API.Mkdir)
			r.Get("/search", cfg.API.Search)
			r.Post("/search/reindex", cfg.API.Reindex)
			r.Get("/stats", cfg.API.Stats)
		})

		// Unknown API routes return JSON, never the SPA shell.
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		})
	})

	// Embedded single-page app for all other routes.
	r.NotFound(spaHandler(cfg.DistFS).ServeHTTP)

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Addr,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
		log: cfg.Log,
	}
}

// Start begins serving HTTP and blocks until the server is shut down.
func (s *Server) Start() error {
	s.log.Info("http server listening", slog.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server, waiting for in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.httpServer.Shutdown(ctx)
}
