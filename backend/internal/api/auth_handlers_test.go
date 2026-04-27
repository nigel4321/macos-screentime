package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

type fakeVerifier struct {
	identity *auth.Identity
	err      error
}

func (f fakeVerifier) Verify(_ context.Context, _ string) (*auth.Identity, error) {
	return f.identity, f.err
}

type fakeIssuer struct {
	out string
	err error
}

func (f fakeIssuer) Issue(_ string) (string, error) { return f.out, f.err }

type fakeStore struct {
	findOrCreateID  string
	findOrCreateErr error

	pairCode      string
	pairExpiresAt time.Time
	pairErr       error

	consumeDst string
	consumeErr error

	gotIdentity     auth.Identity
	gotPairAccount  string
	gotConsumeCode  string
	gotConsumeOwner string
}

func (f *fakeStore) FindOrCreateAccountByIdentity(_ context.Context, id auth.Identity) (string, error) {
	f.gotIdentity = id
	return f.findOrCreateID, f.findOrCreateErr
}

func (f *fakeStore) CreatePairingCode(_ context.Context, accountID string, _ time.Duration) (string, time.Time, error) {
	f.gotPairAccount = accountID
	return f.pairCode, f.pairExpiresAt, f.pairErr
}

func (f *fakeStore) ConsumePairingCodeAndMerge(_ context.Context, code, src string) (string, error) {
	f.gotConsumeCode = code
	f.gotConsumeOwner = src
	return f.consumeDst, f.consumeErr
}

func TestIdentityExchange_Success(t *testing.T) {
	store := &fakeStore{findOrCreateID: "acct-1"}
	issuer := fakeIssuer{out: "jwt-token"}
	verifier := fakeVerifier{identity: &auth.Identity{Provider: "apple", Subject: "001234.apple"}}

	h := IdentityExchangeHandler(verifier, store, issuer)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/apple", strings.NewReader(`{"id_token":"abc"}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp authResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.JWT != "jwt-token" || resp.AccountID != "acct-1" {
		t.Errorf("response: got %+v", resp)
	}
	if store.gotIdentity.Subject != "001234.apple" {
		t.Errorf("identity passed to store: got %+v", store.gotIdentity)
	}
}

func TestIdentityExchange_RejectsMissingIDToken(t *testing.T) {
	h := IdentityExchangeHandler(fakeVerifier{}, &fakeStore{}, fakeIssuer{})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/apple", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestIdentityExchange_VerifierErrorIs401(t *testing.T) {
	h := IdentityExchangeHandler(fakeVerifier{err: errors.New("bad token")}, &fakeStore{}, fakeIssuer{})
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/apple", strings.NewReader(`{"id_token":"abc"}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestPairInit_Success(t *testing.T) {
	store := &fakeStore{pairCode: "482917", pairExpiresAt: time.Now().Add(10 * time.Minute)}

	h := PairInitHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/v1/account:pair-init", nil)
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-mac"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp pairInitResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Code != "482917" {
		t.Errorf("Code: got %q", resp.Code)
	}
	if store.gotPairAccount != "acct-mac" {
		t.Errorf("store called with %q, want acct-mac", store.gotPairAccount)
	}
}

func TestPairInit_RejectsUnauthenticated(t *testing.T) {
	h := PairInitHandler(&fakeStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/account:pair-init", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestPairComplete_Success(t *testing.T) {
	store := &fakeStore{consumeDst: "acct-mac"}
	issuer := fakeIssuer{out: "fresh-jwt"}

	h := PairCompleteHandler(store, issuer)
	req := httptest.NewRequest(http.MethodPost, "/v1/account:pair-complete", strings.NewReader(`{"code":"482917"}`))
	req = req.WithContext(auth.WithAccountID(req.Context(), "acct-android"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp authResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.JWT != "fresh-jwt" || resp.AccountID != "acct-mac" {
		t.Errorf("response: got %+v", resp)
	}
	if store.gotConsumeCode != "482917" || store.gotConsumeOwner != "acct-android" {
		t.Errorf("store called with code=%q owner=%q", store.gotConsumeCode, store.gotConsumeOwner)
	}
}

func TestPairComplete_MapsStoreErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", auth.ErrPairingCodeNotFound, http.StatusNotFound},
		{"expired", auth.ErrPairingCodeExpired, http.StatusGone},
		{"consumed", auth.ErrPairingCodeConsumed, http.StatusConflict},
		{"other", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{consumeErr: tt.err}
			h := PairCompleteHandler(store, fakeIssuer{})
			req := httptest.NewRequest(http.MethodPost, "/v1/account:pair-complete", strings.NewReader(`{"code":"482917"}`))
			req = req.WithContext(auth.WithAccountID(req.Context(), "acct-android"))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}
