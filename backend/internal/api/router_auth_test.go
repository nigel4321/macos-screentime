package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRouter_AuthRoutesDisabledWithoutDeps asserts that without auth
// dependencies (Store, JWTSigner) the auth and pairing routes do not
// mount — the empty Deps path is what tests and dev setups rely on.
func TestRouter_AuthRoutesDisabledWithoutDeps(t *testing.T) {
	srv := httptest.NewServer(NewRouter(Deps{}))
	defer srv.Close()

	for _, path := range []string{"/v1/auth/apple", "/v1/auth/google", "/v1/account:pair-init", "/v1/devices/register", "/v1/usage:batchUpload"} {
		resp, err := http.Post(srv.URL+path, "application/json", strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: got %d, want 404 (route should be disabled)", path, resp.StatusCode)
		}
	}
}

// TestRouter_HealthzWithEmptyDeps exercises the no-DB code path
// end-to-end through the router. Regression for the typed-nil gotcha
// that crashed live /healthz when Deps.DB held a nil *pgxpool.Pool.
func TestRouter_HealthzWithEmptyDeps(t *testing.T) {
	srv := httptest.NewServer(NewRouter(Deps{}))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want 200", resp.StatusCode)
	}
}
