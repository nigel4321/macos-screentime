package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

func TestPolicyCurrent_ReturnsEmptyV0(t *testing.T) {
	h := PolicyCurrentHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/policy/current", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp PolicyResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Version != 0 {
		t.Errorf("version: got %d, want 0", resp.Version)
	}
	// Empty rule families must be present as `[]`, not omitted/null,
	// so client parsers don't need nil-checks.
	if resp.AppLimits == nil || len(resp.AppLimits) != 0 {
		t.Errorf("app_limits: got %v, want []", resp.AppLimits)
	}
	if resp.DowntimeWindows == nil || len(resp.DowntimeWindows) != 0 {
		t.Errorf("downtime_windows: got %v, want []", resp.DowntimeWindows)
	}
	if resp.BlockList == nil || len(resp.BlockList) != 0 {
		t.Errorf("block_list: got %v, want []", resp.BlockList)
	}
}

func TestPolicyCurrent_EmitsEmptyArraysNotNull(t *testing.T) {
	// Regression guard for the JSON wire format: client libraries
	// should iterate without nil-checks.
	h := PolicyCurrentHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/policy/current", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var raw map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, k := range []string{"app_limits", "downtime_windows", "block_list"} {
		v, ok := raw[k]
		if !ok {
			t.Errorf("%s: missing", k)
			continue
		}
		if _, ok := v.([]any); !ok {
			t.Errorf("%s: want []any (JSON array), got %T", k, v)
		}
	}
}

func TestPolicyCurrent_Unauthenticated(t *testing.T) {
	h := PolicyCurrentHandler()
	req := httptest.NewRequest(http.MethodGet, "/v1/policy/current", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}
