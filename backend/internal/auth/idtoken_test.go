package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// idpFixture is a minimal stand-in for Apple/Google: it owns an RSA
// key, serves a JWKS, and signs ID tokens to spec.
type idpFixture struct {
	t      *testing.T
	priv   *rsa.PrivateKey
	kid    string
	server *httptest.Server
}

func newIDPFixture(t *testing.T) *idpFixture {
	t.Helper()
	priv := mustGenerateRSAKey(t)
	f := &idpFixture{t: t, priv: priv, kid: "idp-kid-1"}
	mux := http.NewServeMux()
	mux.HandleFunc("/keys", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": f.kid,
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(f.priv.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big2Bytes(int64(f.priv.E))),
				},
			},
		})
	})
	f.server = httptest.NewServer(mux)
	return f
}

func (f *idpFixture) close() { f.server.Close() }

func (f *idpFixture) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = f.kid
	signed, err := tok.SignedString(f.priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signed
}

func TestAppleVerifier_AcceptsValidToken(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()

	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	tok := idp.sign(t, jwt.MapClaims{
		"iss":   "https://appleid.apple.com",
		"aud":   "dev.nigel.MacAgent",
		"sub":   "001234.apple.subject",
		"email": "user@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	})

	id, err := verifier.Verify(context.Background(), tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id.Provider != "apple" {
		t.Errorf("Provider: got %q", id.Provider)
	}
	if id.Subject != "001234.apple.subject" {
		t.Errorf("Subject: got %q", id.Subject)
	}
	if id.Email != "user@example.com" {
		t.Errorf("Email: got %q", id.Email)
	}
}

func TestAppleVerifier_RejectsWrongAudience(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()

	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	tok := idp.sign(t, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": "evil.attacker.app",
		"sub": "001234.apple.subject",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected wrong-audience rejection")
	}
}

func TestAppleVerifier_RejectsWrongIssuer(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()

	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	tok := idp.sign(t, jwt.MapClaims{
		"iss": "https://accounts.google.com",
		"aud": "dev.nigel.MacAgent",
		"sub": "001234",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected wrong-issuer rejection")
	}
}

func TestAppleVerifier_RejectsExpired(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()

	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	tok := idp.sign(t, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": "dev.nigel.MacAgent",
		"sub": "001234",
		"exp": time.Now().Add(-time.Hour).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected expired rejection")
	}
}

func TestGoogleVerifier_AcceptsBothIssuerForms(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()

	verifier := NewGoogleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "client-id-xyz")

	for _, iss := range []string{"accounts.google.com", "https://accounts.google.com"} {
		tok := idp.sign(t, jwt.MapClaims{
			"iss":   iss,
			"aud":   "client-id-xyz",
			"sub":   "1234567890",
			"email": "user@gmail.com",
			"exp":   time.Now().Add(time.Hour).Unix(),
		})
		id, err := verifier.Verify(context.Background(), tok)
		if err != nil {
			t.Fatalf("iss=%q Verify: %v", iss, err)
		}
		if id.Subject != "1234567890" {
			t.Errorf("iss=%q Subject: got %q", iss, id.Subject)
		}
	}
}

func TestVerifier_RejectsTokenWithoutKID(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()
	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	// Sign manually without kid header.
	claims := jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": "dev.nigel.MacAgent",
		"sub": "001234",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	delete(tok.Header, "kid")
	signed, _ := tok.SignedString(idp.priv)

	if _, err := verifier.Verify(context.Background(), signed); err == nil {
		t.Fatal("expected missing-kid rejection")
	}
}

func TestVerifier_RejectsTokenMissingSub(t *testing.T) {
	idp := newIDPFixture(t)
	defer idp.close()
	verifier := NewAppleVerifier(NewJWKSCache(idp.server.URL+"/keys"), "dev.nigel.MacAgent")

	tok := idp.sign(t, jwt.MapClaims{
		"iss": "https://appleid.apple.com",
		"aud": "dev.nigel.MacAgent",
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), tok); err == nil {
		t.Fatal("expected missing-sub rejection")
	}
}
