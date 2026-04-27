package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// jwksFixture serves a JWKS document containing one RSA key and counts
// how many times it was fetched.
type jwksFixture struct {
	pub      *rsa.PublicKey
	kid      string
	hits     atomic.Int64
	override func() ([]byte, int) // optional: return body + status
}

func newJWKSFixture(t *testing.T, kid string) *jwksFixture {
	t.Helper()
	priv := mustGenerateRSAKey(t)
	return &jwksFixture{pub: &priv.PublicKey, kid: kid}
}

func (f *jwksFixture) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		f.hits.Add(1)
		if f.override != nil {
			body, status := f.override()
			w.WriteHeader(status)
			_, _ = w.Write(body)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": f.kid,
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(f.pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big2Bytes(int64(f.pub.E))),
				},
			},
		})
	})
}

func TestJWKSCache_FetchesOnFirstLookup(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	k, err := cache.Key(context.Background(), "kid-1")
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	if k == nil {
		t.Fatal("Key returned nil")
	}
	if got := fx.hits.Load(); got != 1 {
		t.Errorf("hits: got %d, want 1", got)
	}
}

func TestJWKSCache_CachesBetweenLookups(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	for i := 0; i < 5; i++ {
		if _, err := cache.Key(context.Background(), "kid-1"); err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
	}
	if got := fx.hits.Load(); got != 1 {
		t.Errorf("hits after 5 lookups: got %d, want 1", got)
	}
}

func TestJWKSCache_RefreshesOnUnknownKID_RespectingMinInterval(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	cache.MinRefreshInterval = time.Hour

	if _, err := cache.Key(context.Background(), "kid-1"); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	// Unknown kid: with MinRefreshInterval=1h and lastFetch just now,
	// no extra fetch should happen — the lookup should fail fast.
	if _, err := cache.Key(context.Background(), "kid-unknown"); err == nil {
		t.Fatal("expected unknown kid error")
	}
	if got := fx.hits.Load(); got != 1 {
		t.Errorf("hits: got %d, want 1 (refresh stampede)", got)
	}
}

func TestJWKSCache_RefreshesOnUnknownKID_PastMinInterval(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	cache.MinRefreshInterval = time.Millisecond
	now := time.Now()
	cache.now = func() time.Time { return now }

	if _, err := cache.Key(context.Background(), "kid-1"); err != nil {
		t.Fatalf("warmup: %v", err)
	}
	now = now.Add(time.Hour) // past MinRefreshInterval
	if _, err := cache.Key(context.Background(), "kid-unknown"); err == nil {
		t.Fatal("expected unknown kid error")
	}
	if got := fx.hits.Load(); got != 2 {
		t.Errorf("hits: got %d, want 2 (warmup + refresh)", got)
	}
}

func TestJWKSCache_PropagatesHTTPError(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	fx.override = func() ([]byte, int) { return []byte("nope"), http.StatusInternalServerError }
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	if _, err := cache.Key(context.Background(), "kid-1"); err == nil {
		t.Fatal("expected error from 5xx upstream")
	}
}

func TestJWKSCache_RejectsMalformedJSON(t *testing.T) {
	fx := newJWKSFixture(t, "kid-1")
	fx.override = func() ([]byte, int) { return []byte("not json"), http.StatusOK }
	srv := httptest.NewServer(fx.handler())
	defer srv.Close()

	cache := NewJWKSCache(srv.URL)
	if _, err := cache.Key(context.Background(), "kid-1"); err == nil {
		t.Fatal("expected parse error")
	}
}

func big2Bytes(e int64) []byte {
	// Minimum-length big-endian encoding for the RSA exponent.
	if e == 0 {
		return []byte{0}
	}
	var b []byte
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	return b
}
