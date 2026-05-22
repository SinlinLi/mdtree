package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load with missing file should not error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Auth.SessionTTL.Std() != 24*time.Hour {
		t.Errorf("default session TTL = %v, want 24h", cfg.Auth.SessionTTL.Std())
	}
}

func TestLoadYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	yaml := "server:\n  port: 9999\nauth:\n  session_ttl: 2h\nlog:\n  level: debug\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 9999 {
		t.Errorf("port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Auth.SessionTTL.Std() != 2*time.Hour {
		t.Errorf("session TTL = %v, want 2h", cfg.Auth.SessionTTL.Std())
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("log level = %q, want debug", cfg.Log.Level)
	}
}

func TestValidate(t *testing.T) {
	good := Default()
	good.Root = t.TempDir()
	if err := good.Normalize(); err != nil {
		t.Fatal(err)
	}
	if err := good.Validate(); err != nil {
		t.Errorf("valid config rejected: %v", err)
	}

	badPort := Default()
	badPort.Root = t.TempDir()
	badPort.Server.Port = 0
	_ = badPort.Normalize()
	if err := badPort.Validate(); err == nil {
		t.Error("port 0 should be rejected")
	}

	badLevel := Default()
	badLevel.Root = t.TempDir()
	badLevel.Log.Level = "verbose"
	_ = badLevel.Normalize()
	if err := badLevel.Validate(); err == nil {
		t.Error("invalid log level should be rejected")
	}

	badRoot := Default()
	badRoot.Root = filepath.Join(t.TempDir(), "does-not-exist")
	_ = badRoot.Normalize()
	if err := badRoot.Validate(); err == nil {
		t.Error("non-existent root should be rejected")
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("MDTREE_PORT", "7777")
	t.Setenv("MDTREE_LOG_LEVEL", "warn")
	cfg, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("env port override = %d, want 7777", cfg.Server.Port)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("env log level override = %q, want warn", cfg.Log.Level)
	}
}

func TestInvalidDurationRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("auth:\n  session_ttl: not-a-duration\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("an invalid duration string should produce an error")
	}
}
