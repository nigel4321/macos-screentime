package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

// DeviceStore is the persistence surface DevicesRegisterHandler needs.
// auth.Store satisfies it; tests can substitute a fake.
type DeviceStore interface {
	RegisterDevice(ctx context.Context, accountID, platform, fingerprint string, tokenHash []byte) (string, error)
}

// DeviceTokenMinter mints fresh device tokens. The default
// implementation wraps auth.GenerateDeviceToken; tests can supply a
// deterministic minter.
type DeviceTokenMinter interface {
	NewToken() (string, error)
}

// deviceTokenFn adapts a plain function to DeviceTokenMinter so the
// production wiring stays a one-liner.
type deviceTokenFn func() (string, error)

func (f deviceTokenFn) NewToken() (string, error) { return f() }

// DefaultDeviceTokenMinter returns a minter backed by
// auth.GenerateDeviceToken — what production should use.
func DefaultDeviceTokenMinter() DeviceTokenMinter {
	return deviceTokenFn(auth.GenerateDeviceToken)
}

type deviceRegisterRequest struct {
	Platform    string `json:"platform"`
	Fingerprint string `json:"fingerprint"`
}

type deviceRegisterResponse struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token"`
}

// fingerprintMaxLen caps the size of the client-supplied fingerprint
// to prevent unbounded TEXT writes. 256 is generous: a UUID + a
// hardware id and platform string fit comfortably.
const fingerprintMaxLen = 256

// DevicesRegisterHandler registers (or rotates the token for) a
// device belonging to the authenticated account.
//
// Idempotency: repeating the call with the same (account, fingerprint)
// returns the same device id but a fresh token. The previous token
// stops working at that moment — by design, so a lost-token
// re-registration cannot be used as a parallel session.
func DevicesRegisterHandler(store DeviceStore, minter DeviceTokenMinter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var body deviceRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		body.Fingerprint = strings.TrimSpace(body.Fingerprint)
		if !auth.IsValidPlatform(body.Platform) {
			http.Error(w, "invalid platform", http.StatusBadRequest)
			return
		}
		if body.Fingerprint == "" {
			http.Error(w, "fingerprint required", http.StatusBadRequest)
			return
		}
		if len(body.Fingerprint) > fingerprintMaxLen {
			http.Error(w, "fingerprint too long", http.StatusBadRequest)
			return
		}

		token, err := minter.NewToken()
		if err != nil {
			slog.ErrorContext(r.Context(), "mint device token", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		deviceID, err := store.RegisterDevice(r.Context(), accountID, body.Platform, body.Fingerprint, auth.HashDeviceToken(token))
		if err != nil {
			slog.ErrorContext(r.Context(), "register device", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, deviceRegisterResponse{
			DeviceID:    deviceID,
			DeviceToken: token,
		})
	})
}
