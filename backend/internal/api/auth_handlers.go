// Package api wires the public HTTP surface: middleware, route
// table, and request/response handlers for /v1/* endpoints.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

// IdentityStore is the persistence surface the auth handlers need.
// auth.Store satisfies it; tests can pass fakes.
type IdentityStore interface {
	FindOrCreateAccountByIdentity(ctx context.Context, id auth.Identity) (string, error)
	CreatePairingCode(ctx context.Context, accountID string, ttl time.Duration) (string, time.Time, error)
	ConsumePairingCodeAndMerge(ctx context.Context, code, srcAccountID string) (string, error)
}

// IDVerifier is the subset of *auth.IDTokenVerifier the handlers use.
type IDVerifier interface {
	Verify(ctx context.Context, raw string) (*auth.Identity, error)
}

// TokenIssuer issues backend JWTs given an account id.
type TokenIssuer interface {
	Issue(accountID string) (string, error)
}

type authRequest struct {
	IDToken string `json:"id_token"`
}

type authResponse struct {
	JWT       string `json:"jwt"`
	AccountID string `json:"account_id"`
}

// IdentityExchangeHandler exchanges a third-party ID token for a
// backend JWT. The same handler shape is used for /v1/auth/apple and
// /v1/auth/google — only the verifier differs.
func IdentityExchangeHandler(verifier IDVerifier, store IdentityStore, issuer TokenIssuer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body authRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.IDToken == "" {
			http.Error(w, "id_token required", http.StatusBadRequest)
			return
		}

		identity, err := verifier.Verify(r.Context(), body.IDToken)
		if err != nil {
			slog.InfoContext(r.Context(), "id token rejected", "err", err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		accountID, err := store.FindOrCreateAccountByIdentity(r.Context(), *identity)
		if err != nil {
			slog.ErrorContext(r.Context(), "store: find-or-create account", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		jwt, err := issuer.Issue(accountID)
		if err != nil {
			slog.ErrorContext(r.Context(), "jwt issue", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, authResponse{JWT: jwt, AccountID: accountID})
	})
}

type pairInitResponse struct {
	Code      string `json:"code"`
	ExpiresAt string `json:"expires_at"` // RFC3339
}

// PairInitHandler issues a fresh pairing code for the authenticated
// account. The Mac calls this; the user types the resulting 6-digit
// code into the Android app.
func PairInitHandler(store IdentityStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		code, expiresAt, err := store.CreatePairingCode(r.Context(), accountID, 10*time.Minute)
		if err != nil {
			slog.ErrorContext(r.Context(), "create pairing code", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, pairInitResponse{
			Code:      code,
			ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
		})
	})
}

type pairCompleteRequest struct {
	Code string `json:"code"`
}

// PairCompleteHandler consumes a pairing code on behalf of the
// authenticated Android caller, merges its account into the Mac's, and
// returns a freshly-issued JWT pointing at the survivor account id.
func PairCompleteHandler(store IdentityStore, issuer TokenIssuer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller := auth.AccountIDFromContext(r.Context())
		if caller == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var body pairCompleteRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.Code == "" {
			http.Error(w, "code required", http.StatusBadRequest)
			return
		}

		dst, err := store.ConsumePairingCodeAndMerge(r.Context(), body.Code, caller)
		switch {
		case errors.Is(err, auth.ErrPairingCodeNotFound):
			http.Error(w, "code not found", http.StatusNotFound)
			return
		case errors.Is(err, auth.ErrPairingCodeExpired):
			http.Error(w, "code expired", http.StatusGone)
			return
		case errors.Is(err, auth.ErrPairingCodeConsumed):
			http.Error(w, "code already used", http.StatusConflict)
			return
		case err != nil:
			slog.ErrorContext(r.Context(), "consume pairing code", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		jwt, err := issuer.Issue(dst)
		if err != nil {
			slog.ErrorContext(r.Context(), "jwt issue", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, authResponse{JWT: jwt, AccountID: dst})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
