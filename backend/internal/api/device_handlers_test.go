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
