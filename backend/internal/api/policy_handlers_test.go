package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
)

// fakePolicyStore is the in-memory test double for policy.Store. It
// keeps a map of accountID → latest document and supports overriding
// either Current or Put with a hook so tests can inject errors.
type fakePolicyStore struct {
	mu          sync.Mutex
	docs        map[string]policy.Document
	currentHook func(ctx context.Context, accountID string) (policy.Document, error)
	putHook     func(ctx context.Context, accountID string, doc policy.Document, expectedVersion int64) (int64, error)
}

func newFakePolicyStore() *fakePolicyStore {
	return &fakePolicyStore{docs: map[string]policy.Document{}}
}

func (f *fakePolicyStore) Current(ctx context.Context, accountID string) (policy.Document, error) {
	if f.currentHook != nil {
		return f.currentHook(ctx, accountID)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	doc, ok := f.docs[accountID]
	if !ok {
		return policy.EmptyDocument(), nil
	}
	return doc, nil
}

func (f *fakePolicyStore) Put(ctx context.Context, accountID string, doc policy.Document, expectedVersion int64) (int64, error) {
	if f.putHook != nil {
		return f.putHook(ctx, accountID, doc, expectedVersion)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cur := f.docs[accountID].Version
	if cur != expectedVersion {
		return cur, policy.ErrVersionConflict
	}
	doc.Version = cur + 1
	f.docs[accountID] = doc
	return doc.Version, nil
}

func authedRequest(t *testing.T, method, path, accountID string, body []byte) *http.Request {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if accountID != "" {
		req = req.WithContext(auth.WithAccountID(context.Background(), accountID))
	}
	return req
}

func TestPolicyCurrent_ReturnsEmptyV0WhenNoRow(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyCurrentHandler(store)

	req := authedRequest(t, http.MethodGet, "/v1/policy/current", "acct-1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	if got, want := rr.Header().Get("ETag"), `"0"`; got != want {
		t.Errorf("ETag: got %q, want %q", got, want)
	}

	var doc policy.Document
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Version != 0 {
		t.Errorf("version: got %d, want 0", doc.Version)
	}
	// Empty rule families must be present as `[]`, not omitted/null,
	// so client parsers don't need nil-checks.
	if doc.AppLimits == nil || len(doc.AppLimits) != 0 {
		t.Errorf("app_limits: got %v, want []", doc.AppLimits)
	}
	if doc.DowntimeWindows == nil || len(doc.DowntimeWindows) != 0 {
		t.Errorf("downtime_windows: got %v, want []", doc.DowntimeWindows)
	}
	if doc.BlockList == nil || len(doc.BlockList) != 0 {
		t.Errorf("block_list: got %v, want []", doc.BlockList)
	}
}

func TestPolicyCurrent_EmitsEmptyArraysNotNull(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyCurrentHandler(store)

	req := authedRequest(t, http.MethodGet, "/v1/policy/current", "acct-1", nil)
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
	store := newFakePolicyStore()
	h := PolicyCurrentHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/v1/policy/current", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestPolicyCurrent_StorePropagatesError(t *testing.T) {
	store := newFakePolicyStore()
	store.currentHook = func(_ context.Context, _ string) (policy.Document, error) {
		return policy.Document{}, errors.New("db down")
	}
	h := PolicyCurrentHandler(store)
	req := authedRequest(t, http.MethodGet, "/v1/policy/current", "acct-1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

func validDocBody(t *testing.T) []byte {
	t.Helper()
	doc := policy.Document{
		AppLimits: []policy.AppLimit{
			{BundleID: "com.example.app", DailyLimitSeconds: 3600},
		},
		DowntimeWindows: []policy.DowntimeWindow{
			{Start: "21:00", End: "07:00", Days: []string{"MONDAY", "TUESDAY"}},
		},
		BlockList: []string{"com.example.bad"},
	}
	body, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return body
}

func TestPolicyPut_Success(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	if got, want := rr.Header().Get("ETag"), `"1"`; got != want {
		t.Errorf("ETag: got %q, want %q", got, want)
	}
	var doc policy.Document
	if err := json.NewDecoder(rr.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Version != 1 {
		t.Errorf("version: got %d, want 1", doc.Version)
	}
	if len(doc.AppLimits) != 1 || doc.AppLimits[0].BundleID != "com.example.app" {
		t.Errorf("app_limits: got %v", doc.AppLimits)
	}
}

func TestPolicyPut_AcceptsBareIntIfMatch(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	req.Header.Set("If-Match", "0")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
}

func TestPolicyPut_MissingIfMatch(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionRequired {
		t.Errorf("status: got %d, want 428", rr.Code)
	}
}

func TestPolicyPut_BadIfMatch(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	req.Header.Set("If-Match", "not-a-number")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestPolicyPut_BadJSON(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", []byte("{not json"))
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestPolicyPut_ValidationError(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	doc := policy.Document{
		AppLimits: []policy.AppLimit{
			{BundleID: "com.example.app", DailyLimitSeconds: 0}, // invalid
		},
	}
	body, _ := json.Marshal(doc)
	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", body)
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "daily_limit_seconds") {
		t.Errorf("body should name offending field, got %q", rr.Body.String())
	}
}

func TestPolicyPut_VersionConflict(t *testing.T) {
	store := newFakePolicyStore()
	// Seed an existing v3 row for the account.
	store.docs["acct-1"] = policy.Document{
		Version:         3,
		AppLimits:       []policy.AppLimit{},
		DowntimeWindows: []policy.DowntimeWindow{},
		BlockList:       []string{"com.example.previous"},
	}
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	req.Header.Set("If-Match", `"1"`) // stale
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusPreconditionFailed {
		t.Fatalf("status: got %d, want 412", rr.Code)
	}
	if got, want := rr.Header().Get("ETag"), `"3"`; got != want {
		t.Errorf("ETag: got %q, want %q", got, want)
	}
	var current policy.Document
	if err := json.NewDecoder(rr.Body).Decode(&current); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if current.Version != 3 {
		t.Errorf("conflict body version: got %d, want 3", current.Version)
	}
	if len(current.BlockList) != 1 || current.BlockList[0] != "com.example.previous" {
		t.Errorf("conflict body should be the server's current doc, got %v", current.BlockList)
	}
}

func TestPolicyPut_BodyVersionIgnored(t *testing.T) {
	// A hostile body sets version=999 — server must ignore it and use
	// expectedVersion + 1 from its own state.
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	body := []byte(`{
		"version": 999,
		"app_limits": [],
		"downtime_windows": [],
		"block_list": []
	}`)
	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", body)
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var doc policy.Document
	_ = json.NewDecoder(rr.Body).Decode(&doc)
	if doc.Version != 1 {
		t.Errorf("response version: got %d, want 1", doc.Version)
	}
}

func TestPolicyPut_Unauthenticated(t *testing.T) {
	store := newFakePolicyStore()
	h := PolicyPutHandler(store)

	req := httptest.NewRequest(http.MethodPut, "/v1/policy", bytes.NewReader(validDocBody(t)))
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestPolicyPut_StorePropagatesError(t *testing.T) {
	store := newFakePolicyStore()
	store.putHook = func(_ context.Context, _ string, _ policy.Document, _ int64) (int64, error) {
		return 0, errors.New("db down")
	}
	h := PolicyPutHandler(store)

	req := authedRequest(t, http.MethodPut, "/v1/policy", "acct-1", validDocBody(t))
	req.Header.Set("If-Match", `"0"`)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}
