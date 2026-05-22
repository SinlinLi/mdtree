// Package config loads mdtree configuration from a YAML file and environment
// variables, applying sane defaults and validating the result.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration is a time.Duration that unmarshals from a Go duration string
// (for example "24h" or "30m") in YAML.
type Duration time.Duration

// UnmarshalYAML decodes a duration string into a Duration.
func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Std returns the value as a standard time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// Config is the fully resolved mdtree configuration.
type Config struct {
	Server ServerConfig `yaml:"server"`
	Root   string       `yaml:"root"`
	Auth   AuthConfig   `yaml:"auth"`
	Log    LogConfig    `yaml:"log"`
	Search SearchConfig `yaml:"search"`
}

// ServerConfig controls the HTTP listener.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig controls authentication.
type AuthConfig struct {
	// Password is an optional plaintext password, hashed at startup.
	// Prefer PasswordHash for production deployments.
	Password string `yaml:"password"`
	// PasswordHash is a bcrypt hash produced by `mdtree hash`.
	PasswordHash string `yaml:"password_hash"`
	// SessionTTL is how long a login session stays valid.
	SessionTTL Duration `yaml:"session_ttl"`
	// CookieSecure marks the session cookie Secure (HTTPS-only). Enable it
	// when mdtree is served over HTTPS, including behind a TLS-terminating
	// reverse proxy.
	CookieSecure bool `yaml:"cookie_secure"`
}

// LogConfig controls structured logging.
type LogConfig struct {
	Level      string `yaml:"level"`       // debug|info|warn|error
	Dir        string `yaml:"dir"`         // directory for rotating log files
	Console    bool   `yaml:"console"`     // also log to stderr
	MaxBackups int    `yaml:"max_backups"` // rotated files to retain
	MaxSizeMB  int    `yaml:"max_size_mb"` // rotate when a file exceeds this size
}

// SearchConfig controls the filename index.
type SearchConfig struct {
	Ignore         []string `yaml:"ignore"`          // directory names to skip
	FollowSymlinks bool     `yaml:"follow_symlinks"` // follow symlinked directories
	MaxFiles       int      `yaml:"max_files"`       // safety cap on indexed files
}

// Default returns a Config populated with safe defaults.
func Default() Config {
	return Config{
		Server: ServerConfig{Host: "127.0.0.1", Port: 8080},
		Root:   "/",
		Auth:   AuthConfig{SessionTTL: Duration(24 * time.Hour)},
		Log: LogConfig{
			Level:      "info",
			Dir:        "./logs",
			Console:    true,
			MaxBackups: 5,
			MaxSizeMB:  10,
		},
		Search: SearchConfig{
			Ignore:   []string{".git", "node_modules", ".cache", "vendor", ".Trash"},
			MaxFiles: 200000,
		},
	}
}

// Load reads configuration from the given YAML file (if it exists), then
// applies environment-variable overrides on top of the defaults. A missing
// file is not an error.
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return cfg, fmt.Errorf("parse config %s: %w", path, err)
			}
		case os.IsNotExist(err):
			// A missing config file is fine; defaults and env vars are used.
		default:
			return cfg, fmt.Errorf("read config %s: %w", path, err)
		}
	}
	applyEnv(&cfg)
	return cfg, nil
}

// applyEnv overlays MDTREE_* environment variables onto cfg.
func applyEnv(cfg *Config) {
	if v := os.Getenv("MDTREE_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("MDTREE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("MDTREE_ROOT"); v != "" {
		cfg.Root = v
	}
	if v := os.Getenv("MDTREE_PASSWORD"); v != "" {
		cfg.Auth.Password = v
	}
	if v := os.Getenv("MDTREE_PASSWORD_HASH"); v != "" {
		cfg.Auth.PasswordHash = v
	}
	if v := os.Getenv("MDTREE_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("MDTREE_LOG_DIR"); v != "" {
		cfg.Log.Dir = v
	}
}

// Normalize cleans paths and resolves the root to an absolute path. It must be
// called before Validate.
func (c *Config) Normalize() error {
	abs, err := filepath.Abs(c.Root)
	if err != nil {
		return fmt.Errorf("resolve root %q: %w", c.Root, err)
	}
	c.Root = filepath.Clean(abs)
	c.Log.Level = strings.ToLower(strings.TrimSpace(c.Log.Level))
	c.Server.Host = strings.TrimSpace(c.Server.Host)
	return nil
}

// Validate reports whether the configuration is internally consistent and the
// root directory is usable.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port %d is out of range (1-65535)", c.Server.Port)
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level %q must be one of debug|info|warn|error", c.Log.Level)
	}
	info, err := os.Stat(c.Root)
	if err != nil {
		return fmt.Errorf("root %q is not accessible: %w", c.Root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("root %q is not a directory", c.Root)
	}
	if c.Auth.SessionTTL <= 0 {
		return fmt.Errorf("auth.session_ttl must be a positive duration")
	}
	return nil
}
