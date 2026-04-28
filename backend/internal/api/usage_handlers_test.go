package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/usage"
)

type fakeUsageStore struct {
	results []usage.EventResult
	err     error

	gotDevice string
	gotEvents []usage.Event

	summaryRows []usage.SummaryRow
	summaryErr  error
	gotSummary  usage.SummaryQuery
}

func (f *fakeUsageStore) InsertEvents(_ context.Context, deviceID string, events []usage.Event) ([]usage.EventResult, error) {
	f.gotDevice = deviceID
	f.gotEvents = events
	return f.results, f.err
}

func (f *fakeUsageStore) Summarise(_ context.Context, q usage.SummaryQuery) ([]usage.SummaryRow, error) {
	f.gotSummary = q
	return f.summaryRows, f.summaryErr
}

func ctxWithAuth(deviceID string) context.Context {
	ctx := auth.WithAccountID(context.Background(), "acct-1")
	if deviceID != "" {
		ctx = auth.WithDeviceID(ctx, deviceID)
	}
	return ctx
}

func TestBatchUpload_Success(t *testing.T) {
	store := &fakeUsageStore{
		results: []usage.EventResult{
			{ClientEventID: "e1", Status: usage.StatusAccepted},
			{ClientEventID: "e2", Status: usage.StatusDuplicate},
		},
	}
	body := `{"events":[
		{"client_event_id":"e1","bundle_id":"com.app","started_at":"2026-04-27T11:00:00Z","ended_at":"2026-04-27T11:30:00Z"},
		{"client_event_id":"e2","bundle_id":"com.app","started_at":"2026-04-27T11:31:00Z","ended_at":"2026-04-27T12:00:00Z"}
	]}`

	h := BatchUploadHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(body))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp batchUploadResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Results) != 2 || resp.Results[0].Status != usage.StatusAccepted || resp.Results[1].Status != usage.StatusDuplicate {
		t.Errorf("results: got %+v", resp.Results)
	}
	if store.gotDevice != "dev-1" {
		t.Errorf("device id: got %q", store.gotDevice)
	}
	if len(store.gotEvents) != 2 || store.gotEvents[0].ClientEventID != "e1" {
		t.Errorf("events: got %+v", store.gotEvents)
	}
}

func TestBatchUpload_Unauthenticated(t *testing.T) {
	h := BatchUploadHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(`{"events":[]}`))
	// No account on context.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestBatchUpload_MissingDeviceTokenIs401(t *testing.T) {
	h := BatchUploadHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(`{"events":[{}]}`))
	req = req.WithContext(ctxWithAuth("")) // account but no device
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestBatchUpload_EmptyBatchIs400(t *testing.T) {
	h := BatchUploadHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(`{"events":[]}`))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestBatchUpload_OversizeBatchIs413(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`{"events":[`)
	for i := 0; i < MaxBatchSize+1; i++ {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `{"client_event_id":"e%d","bundle_id":"a","started_at":"2026-04-27T11:00:00Z","ended_at":"2026-04-27T11:30:00Z"}`, i)
	}
	sb.WriteString(`]}`)

	h := BatchUploadHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(sb.String()))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status: got %d, want 413", rr.Code)
	}
}

func TestBatchUpload_StoreErrorIs500(t *testing.T) {
	store := &fakeUsageStore{err: errors.New("db down")}
	body := `{"events":[{"client_event_id":"e1","bundle_id":"a","started_at":"2026-04-27T11:00:00Z","ended_at":"2026-04-27T11:30:00Z"}]}`
	h := BatchUploadHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(body))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

func TestBatchUpload_MalformedJSONIs400(t *testing.T) {
	h := BatchUploadHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(`{"events":[`))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestUsageSummary_Success(t *testing.T) {
	store := &fakeUsageStore{
		summaryRows: []usage.SummaryRow{
			{BundleID: "com.apple.Safari", DurationSeconds: 1800},
			{BundleID: "com.apple.Mail", DurationSeconds: 600},
		},
	}
	h := UsageSummaryHandler(store)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z&groupBy=bundle", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, body=%s", rr.Code, rr.Body.String())
	}
	var resp summaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Results) != 2 || resp.Results[0].BundleID != "com.apple.Safari" || resp.Results[0].DurationSeconds != 1800 {
		t.Errorf("results: got %+v", resp.Results)
	}
	if store.gotSummary.AccountID != "acct-1" {
		t.Errorf("account id: got %q", store.gotSummary.AccountID)
	}
	if len(store.gotSummary.GroupBy) != 1 || store.gotSummary.GroupBy[0] != usage.GroupByBundle {
		t.Errorf("groupBy: got %+v", store.gotSummary.GroupBy)
	}
	wantFrom := time.Date(2026, 4, 27, 0, 0, 0, 0, time.UTC)
	if !store.gotSummary.From.Equal(wantFrom) {
		t.Errorf("from: got %v, want %v", store.gotSummary.From, wantFrom)
	}
}

func TestUsageSummary_NoGroupingReturnsTotalOnly(t *testing.T) {
	store := &fakeUsageStore{
		summaryRows: []usage.SummaryRow{{DurationSeconds: 0}},
	}
	h := UsageSummaryHandler(store)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	if len(store.gotSummary.GroupBy) != 0 {
		t.Errorf("groupBy: got %+v, want empty", store.gotSummary.GroupBy)
	}
}

func TestUsageSummary_GroupByBundleAndDay(t *testing.T) {
	store := &fakeUsageStore{summaryRows: []usage.SummaryRow{}}
	h := UsageSummaryHandler(store)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z&groupBy=bundle,day", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	if len(store.gotSummary.GroupBy) != 2 ||
		store.gotSummary.GroupBy[0] != usage.GroupByBundle ||
		store.gotSummary.GroupBy[1] != usage.GroupByDay {
		t.Errorf("groupBy: got %+v", store.gotSummary.GroupBy)
	}
}

func TestUsageSummary_Unauthenticated(t *testing.T) {
	h := UsageSummaryHandler(&fakeUsageStore{})
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestUsageSummary_BadParams(t *testing.T) {
	cases := []struct {
		name string
		url  string
	}{
		{"missing from", "/v1/usage:summary?to=2026-04-28T00:00:00Z"},
		{"missing to", "/v1/usage:summary?from=2026-04-27T00:00:00Z"},
		{"unparseable from", "/v1/usage:summary?from=yesterday&to=2026-04-28T00:00:00Z"},
		{"to before from", "/v1/usage:summary?from=2026-04-28T00:00:00Z&to=2026-04-27T00:00:00Z"},
		{"equal bounds", "/v1/usage:summary?from=2026-04-28T00:00:00Z&to=2026-04-28T00:00:00Z"},
		{"range too large", "/v1/usage:summary?from=2025-01-01T00:00:00Z&to=2026-04-28T00:00:00Z"},
		{"unknown groupBy", "/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z&groupBy=user"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := UsageSummaryHandler(&fakeUsageStore{})
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want 400", rr.Code)
			}
		})
	}
}

func TestUsageSummary_StoreErrorIs500(t *testing.T) {
	store := &fakeUsageStore{summaryErr: errors.New("db down")}
	h := UsageSummaryHandler(store)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want 500", rr.Code)
	}
}

func TestUsageSummary_InvalidRangeFromStoreIs400(t *testing.T) {
	store := &fakeUsageStore{summaryErr: usage.ErrInvalidRange}
	h := UsageSummaryHandler(store)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/usage:summary?from=2026-04-27T00:00:00Z&to=2026-04-28T00:00:00Z", nil)
	req = req.WithContext(auth.WithAccountID(context.Background(), "acct-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

// Sanity: time.Time decoding round-trips so the store sees real times.
func TestBatchUpload_DecodesTimes(t *testing.T) {
	store := &fakeUsageStore{results: []usage.EventResult{{ClientEventID: "e", Status: usage.StatusAccepted}}}
	body := `{"events":[{"client_event_id":"e","bundle_id":"a","started_at":"2026-04-27T11:00:00Z","ended_at":"2026-04-27T11:30:00Z"}]}`
	h := BatchUploadHandler(store)
	req := httptest.NewRequest(http.MethodPost, "/v1/usage:batchUpload", strings.NewReader(body))
	req = req.WithContext(ctxWithAuth("dev-1"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d", rr.Code)
	}
	want := time.Date(2026, 4, 27, 11, 0, 0, 0, time.UTC)
	if !store.gotEvents[0].StartedAt.Equal(want) {
		t.Errorf("StartedAt: got %v, want %v", store.gotEvents[0].StartedAt, want)
	}
}
