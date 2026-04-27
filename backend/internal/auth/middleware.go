package auth

import (
	"context"
	"net/http"
	"strings"
)

// AccountStore is the surface Authenticator needs from the persistence
// layer: confirm an account is still live before honoring a JWT.
type AccountStore interface {
	AccountExists(ctx context.Context, accountID string) (bool, error)
}

// Authenticator validates a Bearer JWT, confirms the account still
// exists, and stashes the account id on the request context.
//
// Failure modes (all 401, no body details to avoid leaking schema):
//   - Missing/malformed Authorization header
//   - Token fails JWT verification
//   - Account row was deleted (e.g. merged away)
func Authenticator(verifier *Verifier, store AccountStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, ok := bearerToken(r)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := verifier.Parse(raw)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			exists, err := store.AccountExists(r.Context(), claims.AccountID)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			if !exists {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithAccountID(r.Context(), claims.AccountID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// DeviceResolver resolves an opaque device-token header to a device id
// for the authenticated account. Implementations are expected to scope
// the lookup to the account from context to prevent token-replay across
// accounts.
type DeviceResolver interface {
	ResolveDevice(ctx context.Context, accountID, deviceToken string) (deviceID string, err error)
}

// DeviceContext reads the X-Device-Token header and, if present,
// resolves the device and stashes its id on the request context. A
// missing header is allowed — routes that require a device should
// check DeviceIDFromContext and reject empty values themselves.
func DeviceContext(resolver DeviceResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := r.Header.Get("X-Device-Token")
			if tok == "" {
				next.ServeHTTP(w, r)
				return
			}
			accountID := AccountIDFromContext(r.Context())
			if accountID == "" {
				// DeviceContext must always be mounted after Authenticator.
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			deviceID, err := resolver.ResolveDevice(r.Context(), accountID, tok)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := WithDeviceID(r.Context(), deviceID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	tok := strings.TrimSpace(h[len(prefix):])
	return tok, tok != ""
}
