package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Config captures the process-wide settings the server reads at startup.
type Config struct {
	Port        string
	LogLevel    slog.Level
	DatabaseURL string // empty disables Postgres-dependent features (dev only)
}

// Load reads configuration from environment variables, applying sane
// defaults for unset values. Returns an error only when a value is
// present but unparseable (e.g. an unknown LOG_LEVEL).
func Load() (Config, error) {
	cfg := Config{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    slog.LevelInfo,
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		lvl, err := parseLogLevel(raw)
		if err != nil {
			return Config{}, err
		}
		cfg.LogLevel = lvl
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q (want debug|info|warn|error)", s)
	}
}
