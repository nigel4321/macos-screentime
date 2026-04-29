package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

type fakeDeviceStore struct {
	deviceID string
	err      error

	gotAccount     string
	gotPlatform    string
	gotFingerprint string
	gotTokenHash   []byte
}

func (f *fakeDeviceStore) RegisterDevice(_ context.Context, account, platform, fingerprint string, hash []byte) (string, error) {
	f.gotAccount = account
	f.gotPlatform = platform
	f.gotFingerprint = fingerprint
	f.gotTokenHash = append([]byte(nil), hash...)
	return f.deviceID, f.err
}

type staticMinter string

func (s staticMinter) NewToken() (string, error) { return string(s), nil }

func TestDevicesRegister_Success_StoresHashOfReturnedToken(t *testing.T) {
	store := &fakeDeviceStore{deviceID: "dev-1"}
	minter := staticMinter("tok-abc")

	h := DevicesRegisterHandler(store, minter)
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/register",
		strings.NewReader(`{"platform":"macos","fingerprint":"hw-aaaa"}`))
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp deviceRegisterResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.DeviceID != "dev-1" {
		t.Errorf("DeviceID: got %q", resp.DeviceID)
	}
	if resp.DeviceToken != "tok-abc" {
		t.Errorf("DeviceToken: got %q", resp.DeviceToken)
	}

	wantHash := sha256.Sum256([]byte("tok-abc"))
	if string(store.gotTokenHash) != string(wantHash[:]) {
		t.Errorf("store hash: got %x, want %x", store.gotTokenHash, wantHash[:])
	}
	if store.gotAccount != "acct-1" {
		t.Errorf("account: got %q", store.gotAccount)
	}
	if store.gotPlatform != "macos" || store.gotFingerprint != "hw-aaaa" {
		t.Errorf("payload: platform=%q fingerprint=%q", store.gotPlatform, store.gotFingerprint)
	}
}

func TestDevicesRegister_Unauthenticated(t *testing.T) {
	h := DevicesRegisterHandler(&fakeDeviceStore{}, staticMinter("tok"))
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/register",
		strings.NewReader(`{"platform":"macos","fingerprint":"hw"}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestDevicesRegister_Validation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"empty platform", `{"platform":"","fingerprint":"hw"}`},
		{"unknown platform", `{"platform":"linux","fingerprint":"hw"}`},
		{"missing fingerprint", `{"platform":"macos","fingerprint":""}`},
		{"whitespace fingerprint", `{"platform":"macos","fingerprint":"   "}`},
		{"oversize fingerprint", `{"platform":"macos","fingerprint":"` + strings.Repeat("x", fingerprintMaxLen+1) + `"}`},
		{"malformed json", `{not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := DevicesRegisterHandler(&fakeDeviceStore{}, staticMinter("tok"))
			req := httptest.NewRequest(http.MethodPost, "/v1/devices/register", strings.NewReader(tt.body))
			req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want 400 (body=%s)", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestDevicesRegister_StoreErrorIs500(t *testing.T) {
	store := &fakeDeviceStore{err: errors.New("db down")}
	h := DevicesRegisterHandler(store, staticMinter("tok"))
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/register",
		strings.NewReader(`{"platform":"macos","fingerprint":"hw"}`))
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

type errMinter struct{}

func (errMinter) NewToken() (string, error) { return "", errors.New("rng dead") }

func TestDevicesRegister_MinterErrorIs500(t *testing.T) {
	h := DevicesRegisterHandler(&fakeDeviceStore{}, errMinter{})
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/register",
		strings.NewReader(`{"platform":"macos","fingerprint":"hw"}`))
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

type fakeDeviceLister struct {
	gotAccount string
	devices    []auth.DeviceSummary
	err        error
}

func (f *fakeDeviceLister) ListDevicesForAccount(_ context.Context, accountID string) ([]auth.DeviceSummary, error) {
	f.gotAccount = accountID
	return f.devices, f.err
}

func TestDevicesList_Success_ReturnsDevicesScopedToAccount(t *testing.T) {
	created := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	seen := time.Date(2026, 4, 29, 9, 0, 0, 0, time.UTC)
	lister := &fakeDeviceLister{
		devices: []auth.DeviceSummary{
			{ID: "dev-1", Platform: "macos", Fingerprint: "fp-mac", CreatedAt: created, LastSeenAt: &seen},
			{ID: "dev-2", Platform: "android", Fingerprint: "fp-droid", CreatedAt: created},
		},
	}
	h := DevicesListHandler(lister)
	req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	if lister.gotAccount != "acct-1" {
		t.Errorf("account scope: got %q, want acct-1", lister.gotAccount)
	}
	var resp deviceListResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got, want := len(resp.Devices), 2; got != want {
		t.Fatalf("len(devices): got %d, want %d", got, want)
	}
	if resp.Devices[0].ID != "dev-1" || resp.Devices[0].Platform != "macos" {
		t.Errorf("dev[0]: %+v", resp.Devices[0])
	}
	if resp.Devices[0].LastSeenAt == nil || !resp.Devices[0].LastSeenAt.Equal(seen) {
		t.Errorf("dev[0].LastSeenAt: %+v", resp.Devices[0].LastSeenAt)
	}
	// last_seen_at omits when nil thanks to the `,omitempty` tag.
	if resp.Devices[1].LastSeenAt != nil {
		t.Errorf("dev[1].LastSeenAt: got %+v, want nil", resp.Devices[1].LastSeenAt)
	}
}

func TestDevicesList_EmptyReturnsEmptyArrayNotNull(t *testing.T) {
	lister := &fakeDeviceLister{devices: nil}
	h := DevicesListHandler(lister)
	req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	// JSON shape contract: zero-device case returns `"devices":[]`,
	// not `"devices":null`. Android relies on this for the empty-state
	// branch in the pairing UI.
	if !strings.Contains(rr.Body.String(), `"devices":[]`) {
		t.Errorf("expected empty array, got %s", rr.Body.String())
	}
}

func TestDevicesList_Unauthenticated(t *testing.T) {
	h := DevicesListHandler(&fakeDeviceLister{})
	req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestDevicesList_StoreErrorIs500(t *testing.T) {
	lister := &fakeDeviceLister{err: errors.New("db down")}
	h := DevicesListHandler(lister)
	req := httptest.NewRequest(http.MethodGet, "/v1/devices", nil)
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}
