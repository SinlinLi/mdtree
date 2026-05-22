// Package logger configures structured, leveled logging for mdtree with dual
// output (console + rotating file) on top of the standard log/slog package.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

const logFileName = "mdtree.log"

// Options controls logger construction.
type Options struct {
	Level      string   // debug|info|warn|error
	Dir        string   // directory for rotating log files
	Console    bool     // also write human-readable logs to stderr
	MaxBackups int      // number of rotated files to retain
	MaxSizeMB  int      // rotate when the active file exceeds this size
	Modules    []string // when non-empty, only these modules are logged
}

// Logger is the application logger plus a handle to adjust the level at
// runtime and release the underlying log file.
type Logger struct {
	*slog.Logger
	level  *slog.LevelVar
	closer io.Closer
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New builds a Logger that fans out to a rotating file and (optionally) the
// console. The previous run's log file is rotated on startup so each run
// begins with a fresh mdtree.log.
func New(opts Options) (*Logger, error) {
	levelVar := new(slog.LevelVar)
	levelVar.Set(parseLevel(opts.Level))

	if err := os.MkdirAll(opts.Dir, 0o750); err != nil {
		return nil, fmt.Errorf("create log dir %s: %w", opts.Dir, err)
	}
	rotator := &lumberjack.Logger{
		Filename:   filepath.Join(opts.Dir, logFileName),
		MaxBackups: opts.MaxBackups,
		MaxSize:    opts.MaxSizeMB,
		LocalTime:  true,
	}
	// Start every run in a fresh file, keeping the previous run as a backup.
	if fi, err := os.Stat(rotator.Filename); err == nil && fi.Size() > 0 {
		_ = rotator.Rotate()
	}

	handlerOpts := &slog.HandlerOptions{Level: levelVar}
	handlers := []slog.Handler{slog.NewJSONHandler(rotator, handlerOpts)}
	if opts.Console {
		handlers = append(handlers, slog.NewTextHandler(os.Stderr, handlerOpts))
	}

	var handler slog.Handler = &fanoutHandler{handlers: handlers}
	if len(opts.Modules) > 0 {
		allowed := make(map[string]bool, len(opts.Modules))
		for _, m := range opts.Modules {
			allowed[strings.TrimSpace(m)] = true
		}
		handler = &moduleFilter{Handler: handler, allowed: allowed}
	}

	return &Logger{
		Logger: slog.New(handler),
		level:  levelVar,
		closer: rotator,
	}, nil
}

// SetLevel adjusts the active log level at runtime.
func (l *Logger) SetLevel(level string) { l.level.Set(parseLevel(level)) }

// Module returns a child logger tagged with the given module name, enabling
// per-module filtering and grouping.
func (l *Logger) Module(name string) *slog.Logger {
	return l.With(slog.String("module", name))
}

// Close flushes and releases the underlying log file.
func (l *Logger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}

// fanoutHandler dispatches every record to each wrapped handler, giving the
// console + file dual-channel behaviour.
type fanoutHandler struct {
	handlers []slog.Handler
}

func (h *fanoutHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, sub := range h.handlers {
		if sub.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *fanoutHandler) Handle(ctx context.Context, r slog.Record) error {
	var firstErr error
	for _, sub := range h.handlers {
		if !sub.Enabled(ctx, r.Level) {
			continue
		}
		if err := sub.Handle(ctx, r.Clone()); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (h *fanoutHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	subs := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		subs[i] = sub.WithAttrs(attrs)
	}
	return &fanoutHandler{handlers: subs}
}

func (h *fanoutHandler) WithGroup(name string) slog.Handler {
	subs := make([]slog.Handler, len(h.handlers))
	for i, sub := range h.handlers {
		subs[i] = sub.WithGroup(name)
	}
	return &fanoutHandler{handlers: subs}
}

// moduleFilter drops records whose "module" attribute is not in the allowlist,
// implementing per-module log filtering.
type moduleFilter struct {
	slog.Handler
	allowed map[string]bool
	module  string
}

func (h *moduleFilter) Handle(ctx context.Context, r slog.Record) error {
	module := h.module
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "module" {
			module = a.Value.String()
			return false
		}
		return true
	})
	if module != "" && !h.allowed[module] {
		return nil
	}
	return h.Handler.Handle(ctx, r)
}

func (h *moduleFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	module := h.module
	for _, a := range attrs {
		if a.Key == "module" {
			module = a.Value.String()
		}
	}
	return &moduleFilter{Handler: h.Handler.WithAttrs(attrs), allowed: h.allowed, module: module}
}

func (h *moduleFilter) WithGroup(name string) slog.Handler {
	return &moduleFilter{Handler: h.Handler.WithGroup(name), allowed: h.allowed, module: h.module}
}
