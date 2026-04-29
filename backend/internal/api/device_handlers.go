package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

// DeviceStore is the persistence surface DevicesRegisterHandler needs.
// auth.Store satisfies it; tests can substitute a fake.
type DeviceStore interface {
	RegisterDevice(ctx context.Context, accountID, platform, fingerprint string, tokenHash []byte) (string, error)
}

// DeviceLister is the persistence surface DevicesListHandler needs.
// auth.Store satisfies it; tests can substitute a fake.
type DeviceLister interface {
	ListDevicesForAccount(ctx context.Context, accountID string) ([]auth.DeviceSummary, error)
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

type deviceListItem struct {
	ID          string     `json:"id"`
	Platform    string     `json:"platform"`
	Fingerprint string     `json:"fingerprint"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

type deviceListResponse struct {
	Devices []deviceListItem `json:"devices"`
}

// DevicesListHandler returns the calling account's registered devices.
// Account scope is enforced by the SQL WHERE clause inside
// ListDevicesForAccount, so a stolen JWT cannot enumerate devices
// belonging to a different account.
func DevicesListHandler(lister DeviceLister) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		devices, err := lister.ListDevicesForAccount(r.Context(), accountID)
		if err != nil {
			slog.ErrorContext(r.Context(), "list devices", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		out := make([]deviceListItem, 0, len(devices))
		for _, d := range devices {
			out = append(out, deviceListItem{
				ID:          d.ID,
				Platform:    d.Platform,
				Fingerprint: d.Fingerprint,
				CreatedAt:   d.CreatedAt,
				LastSeenAt:  d.LastSeenAt,
			})
		}
		writeJSON(w, http.StatusOK, deviceListResponse{Devices: out})
	})
}
