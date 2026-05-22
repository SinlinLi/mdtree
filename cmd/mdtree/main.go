// Command mdtree is a self-hosted markdown file browser and editor: it serves
// a markdown-only file tree, a browse/edit UI and indexed filename search,
// behind password authentication.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/SinlinLi/mdtree/internal/api"
	"github.com/SinlinLi/mdtree/internal/auth"
	"github.com/SinlinLi/mdtree/internal/config"
	"github.com/SinlinLi/mdtree/internal/files"
	"github.com/SinlinLi/mdtree/internal/logger"
	"github.com/SinlinLi/mdtree/internal/metrics"
	"github.com/SinlinLi/mdtree/internal/search"
	"github.com/SinlinLi/mdtree/internal/server"
	"github.com/SinlinLi/mdtree/web"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hash":
			runHash()
			return
		case "version", "--version", "-v":
			fmt.Printf("mdtree %s\n", version)
			return
		}
	}
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "mdtree: %v\n", err)
		os.Exit(1)
	}
}

// run loads configuration, builds every component and serves until a signal
// triggers a graceful shutdown.
func run() error {
	fs := flag.NewFlagSet("mdtree", flag.ExitOnError)
	var (
		configPath = fs.String("config", "config.yaml", "path to the YAML config file")
		host       = fs.String("host", "", "override the server host")
		port       = fs.Int("port", 0, "override the server port")
		root       = fs.String("root", "", "override the browsable root directory")
		logLevel   = fs.String("log-level", "", "override the log level")
	)
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	// Command-line flags take precedence over the file and environment.
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *root != "" {
		cfg.Root = *root
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}
	if err := cfg.Normalize(); err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	log, err := logger.New(logger.Options{
		Level:      cfg.Log.Level,
		Dir:        cfg.Log.Dir,
		Console:    cfg.Log.Console,
		MaxBackups: cfg.Log.MaxBackups,
		MaxSizeMB:  cfg.Log.MaxSizeMB,
	})
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info("starting mdtree",
		slog.String("version", version),
		slog.String("root", cfg.Root),
		slog.String("addr", addr),
	)

	passwordHash, err := resolvePassword(cfg, log)
	if err != nil {
		return err
	}

	// Build the core components.
	resolver := files.NewResolver(cfg.Root, cfg.Search.FollowSymlinks)
	manager := files.NewManager(resolver)
	lister := files.NewLister(resolver, cfg.Search.Ignore, false)
	index := search.NewIndex(cfg.Root, cfg.Search.Ignore, cfg.Search.MaxFiles, log.Module("search"))
	met := metrics.New()

	sessions := auth.NewSessionStore(cfg.Auth.SessionTTL.Std())
	authn := auth.NewAuthenticator(sessions, cfg.Auth.CookieSecure)
	limiter := auth.NewRateLimiter(8, time.Minute)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	sessions.StartGC(ctx, 5*time.Minute)

	// Build the search index in the background so startup is not blocked by a
	// large filesystem walk.
	go func() {
		count, elapsed, err := index.Build()
		if err != nil {
			log.Module("search").Error("initial index build failed", slog.Any("error", err))
			return
		}
		met.SetIndex(count, elapsed)
	}()

	apiHandler := &api.Handler{
		Files:        manager,
		Lister:       lister,
		Index:        index,
		Auth:         authn,
		Sessions:     sessions,
		Limiter:      limiter,
		Metrics:      met,
		PasswordHash: passwordHash,
		Root:         cfg.Root,
		Log:          log.Module("api"),
	}

	dist, err := web.DistFS()
	if err != nil {
		return fmt.Errorf("load embedded frontend: %w", err)
	}

	srv := server.New(server.Config{
		Addr:    addr,
		API:     apiHandler,
		Auth:    authn,
		Metrics: met,
		DistFS:  dist,
		Log:     log.Module("http"),
	})

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// resolvePassword determines the bcrypt password hash to authenticate against.
//
// Precedence: an explicit password_hash, then a plaintext password (hashed
// now), and finally a freshly generated random password printed once to the
// console. The random password is drawn from the system CSPRNG.
func resolvePassword(cfg config.Config, log *logger.Logger) (string, error) {
	if cfg.Auth.PasswordHash != "" {
		return cfg.Auth.PasswordHash, nil
	}
	if cfg.Auth.Password != "" {
		hash, err := auth.HashPassword(cfg.Auth.Password)
		if err != nil {
			return "", err
		}
		log.Warn("using plaintext password from config; prefer auth.password_hash (see `mdtree hash`)")
		return hash, nil
	}

	plain, err := auth.GeneratePassword(18)
	if err != nil {
		return "", err
	}
	hash, err := auth.HashPassword(plain)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "\n"+
		"  ┌─────────────────────────────────────────────────────────────┐\n"+
		"  │  mdtree: no password configured — generated a temporary one.  │\n"+
		"  └─────────────────────────────────────────────────────────────┘\n"+
		"     password: "+plain+"\n"+
		"     This password is NOT persisted and changes on every restart.\n"+
		"     Run `mdtree hash` and set auth.password_hash to make it stable.\n\n")
	log.Warn("generated temporary random password (not persisted)")
	return hash, nil
}

// runHash reads a password and prints its bcrypt hash. With an interactive
// terminal it prompts twice without echo; otherwise it reads one line from
// stdin and prints only the hash, which is convenient for scripted setup:
//
//	echo -n 'my-password' | mdtree hash
func runHash() {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	var password string
	if interactive {
		pw1 := promptPassword("Password: ")
		pw2 := promptPassword("Confirm:  ")
		if pw1 != pw2 {
			fmt.Fprintln(os.Stderr, "mdtree: passwords do not match")
			os.Exit(1)
		}
		password = pw1
	} else {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtree: read password: %v\n", err)
			os.Exit(1)
		}
		password = strings.TrimRight(string(data), "\r\n")
	}

	if len(password) < 8 {
		fmt.Fprintln(os.Stderr, "mdtree: password must be at least 8 characters")
		os.Exit(1)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdtree: %v\n", err)
		os.Exit(1)
	}

	if interactive {
		fmt.Print("\nAdd the following to your config.yaml under the auth section:\n\n")
		fmt.Printf("  password_hash: %q\n", hash)
	} else {
		fmt.Println(hash)
	}
}

// promptPassword reads a password from the terminal without echoing it.
func promptPassword(label string) string {
	fmt.Fprint(os.Stderr, label)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdtree: read password: %v\n", err)
		os.Exit(1)
	}
	return string(data)
}
