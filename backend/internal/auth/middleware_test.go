package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubAccountStore struct {
	exists bool
	err    error
}

func (s stubAccountStore) AccountExists(_ context.Context, _ string) (bool, error) {
	return s.exists, s.err
}

func TestAuthenticator_AcceptsValidToken(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())
	tok, _ := signer.Issue("acct-1")

	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = AccountIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mw := Authenticator(verifier, stubAccountStore{exists: true})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	mw(next).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if seen != "acct-1" {
		t.Errorf("AccountID in context: got %q, want acct-1", seen)
	}
}

func TestAuthenticator_RejectsMissingHeader(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())

	mw := Authenticator(verifier, stubAccountStore{exists: true})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("handler should not run")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestAuthenticator_RejectsBadToken(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())

	mw := Authenticator(verifier, stubAccountStore{exists: true})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
	_ = signer
}

func TestAuthenticator_RejectsDeletedAccount(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())
	tok, _ := signer.Issue("acct-1")

	mw := Authenticator(verifier, stubAccountStore{exists: false})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run for deleted account")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestAuthenticator_PropagatesStoreError(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())
	tok, _ := signer.Issue("acct-1")

	mw := Authenticator(verifier, stubAccountStore{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

type stubResolver struct {
	deviceID string
	err      error
}

func (s stubResolver) ResolveDevice(_ context.Context, _, _ string) (string, error) {
	return s.deviceID, s.err
}

func TestDeviceContext_AttachesDeviceID(t *testing.T) {
	mw := DeviceContext(stubResolver{deviceID: "dev-1"})

	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = DeviceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Device-Token", "tok")
	req = req.WithContext(WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()

	mw(next).ServeHTTP(rr, req)

	if seen != "dev-1" {
		t.Errorf("device id: got %q, want dev-1", seen)
	}
}

func TestDeviceContext_PassesThroughWhenHeaderMissing(t *testing.T) {
	mw := DeviceContext(stubResolver{err: errors.New("should not be called")})

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if id := DeviceIDFromContext(r.Context()); id != "" {
			t.Errorf("expected empty device id, got %q", id)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()

	mw(next).ServeHTTP(rr, req)
	if !called {
		t.Fatal("next handler should run when header is missing")
	}
}

func TestDeviceContext_RejectsBadDeviceToken(t *testing.T) {
	mw := DeviceContext(stubResolver{err: errors.New("not your device")})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Device-Token", "wrong-token")
	req = req.WithContext(WithAccountID(req.Context(), "acct-1"))
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run for bad device token")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}
