// Package config loads runtime configuration from environment
// variables with sensible defaults for local development.
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

	// Auth (§2.3). Empty JWTSigningKey disables auth routes — useful in
	// dev. Verification keys are additional public keys accepted during
	// JWT key rotation.
	JWTSigningKey       string   // PEM-encoded EC P-256 private key
	JWTVerificationKeys []string // PEM-encoded EC P-256 public keys (rotated-out)
	AppleAudience       string   // Mac app bundle id
	GoogleAudience      string   // OAuth client id
}

// Load reads configuration from environment variables, applying sane
// defaults for unset values. Returns an error only when a value is
// present but unparseable (e.g. an unknown LOG_LEVEL).
func Load() (Config, error) {
	cfg := Config{
		Port:           getEnv("PORT", "8080"),
		LogLevel:       slog.LevelInfo,
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSigningKey:  os.Getenv("JWT_SIGNING_KEY"),
		AppleAudience:  os.Getenv("APPLE_AUDIENCE"),
		GoogleAudience: os.Getenv("GOOGLE_AUDIENCE"),
	}
	if raw := os.Getenv("JWT_VERIFICATION_KEYS"); raw != "" {
		// Comma-separated PEMs: PEMs themselves contain newlines but
		// no commas, so this split is unambiguous.
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				cfg.JWTVerificationKeys = append(cfg.JWTVerificationKeys, p)
			}
		}
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
