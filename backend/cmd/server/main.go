// Package main is the screentime API server entry point.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nigel4321/macos-screentime/backend/internal/api"
	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/config"
	"github.com/nigel4321/macos-screentime/backend/internal/db"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
	"github.com/nigel4321/macos-screentime/backend/internal/usage"
)

const (
	appleJWKSURL  = "https://appleid.apple.com/auth/keys"
	googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"
)

const shutdownTimeout = 15 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		if err := db.Migrate(ctx, cfg.DatabaseURL); err != nil {
			return err
		}
		pool, err = db.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer pool.Close()
		if err := db.EnsurePartitionsAroundNow(ctx, pool, time.Now().UTC()); err != nil {
			return err
		}
		logger.Info("database ready", "migrations", "applied", "partitions", "ensured")
	} else {
		logger.Warn("DATABASE_URL not set — running without Postgres (dev only)")
	}

	deps := api.Deps{}
	if pool != nil {
		// Assign through the typed-nil dance: a nil *pgxpool.Pool stored
		// in a Pinger interface compares non-nil and would crash on
		// Ping. Only assign when we actually have a pool.
		deps.DB = pool
		deps.UsageStore = usage.NewStore(pool)
		deps.PolicyStore = policy.NewStore(pool)
	}
	if pool != nil && cfg.JWTSigningKey != "" {
		signer, err := auth.NewSigner([]byte(cfg.JWTSigningKey))
		if err != nil {
			return fmt.Errorf("parse JWT signing key: %w", err)
		}
		verifierKeys := []*ecdsa.PublicKey{signer.PublicKey()}
		for i, raw := range cfg.JWTVerificationKeys {
			pub, err := parseECPublicKey(raw)
			if err != nil {
				return fmt.Errorf("parse JWT_VERIFICATION_KEYS[%d]: %w", i, err)
			}
			verifierKeys = append(verifierKeys, pub)
		}
		jwtVerifier, err := auth.NewVerifier(verifierKeys...)
		if err != nil {
			return fmt.Errorf("build JWT verifier: %w", err)
		}

		deps.Store = auth.NewStore(pool)
		deps.JWTSigner = signer
		deps.JWTVerifier = jwtVerifier
		if cfg.AppleAudience != "" {
			deps.AppleVerifier = auth.NewAppleVerifier(auth.NewJWKSCache(appleJWKSURL), cfg.AppleAudience)
		} else {
			logger.Warn("APPLE_AUDIENCE not set — /v1/auth/apple disabled")
		}
		if cfg.GoogleAudience != "" {
			deps.GoogleVerifier = auth.NewGoogleVerifier(auth.NewJWKSCache(googleJWKSURL), cfg.GoogleAudience)
		} else {
			logger.Warn("GOOGLE_AUDIENCE not set — /v1/auth/google disabled")
		}
		logger.Info("auth ready", "rotated_keys", len(cfg.JWTVerificationKeys))
	} else if pool != nil {
		logger.Warn("JWT_SIGNING_KEY not set — auth routes disabled")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.NewRouter(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("server stopped cleanly")
	return nil
}

func parseECPublicKey(pemStr string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not an EC public key")
	}
	return ec, nil
}
