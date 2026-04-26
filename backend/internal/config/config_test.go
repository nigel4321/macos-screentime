package config

import (
	"log/slog"
	"testing"
)

func TestLoad_DefaultsWhenEnvUnset(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port: got %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel: got %v, want %v", cfg.LogLevel, slog.LevelInfo)
	}
}

func TestLoad_ReadsEnvVars(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port: got %q, want 9090", cfg.Port)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel: got %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoad_RejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("LOG_LEVEL", "shouty")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL, got nil")
	}
}
