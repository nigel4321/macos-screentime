package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/usage"
)

// UsageStore is the persistence surface the usage handlers need.
// usage.Store satisfies it; tests substitute a fake.
type UsageStore interface {
	InsertEvents(ctx context.Context, deviceID string, events []usage.Event) ([]usage.EventResult, error)
	Summarise(ctx context.Context, q usage.SummaryQuery) ([]usage.SummaryRow, error)
}

// MaxBatchSize caps the number of events a single batchUpload request
// may carry. Bounded so a malformed (or hostile) client cannot pin a
// connection inside one INSERT loop. The Mac agent flushes far below
// this in practice.
const MaxBatchSize = 500

type batchUploadRequest struct {
	Events []usage.Event `json:"events"`
}

type batchUploadResponse struct {
	Results []usage.EventResult `json:"results"`
}

// BatchUploadHandler ingests a JSON batch of usage events for the
// device on the request context and returns a per-event status array
// in the same order as the input.
//
// Authorization model: the Authenticator middleware confirmed the JWT
// and attached account id; the DeviceContext middleware translated
// the X-Device-Token header to a device id. We trust those two and
// reject the request only if either is missing — i.e. if the route
// was mounted incorrectly or the client forgot the device-token
// header. The store does not get to see the account id directly: the
// device row already encodes that linkage and trusting it avoids a
// double-lookup on the hot path.
func BatchUploadHandler(store UsageStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.AccountIDFromContext(r.Context()) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		deviceID := auth.DeviceIDFromContext(r.Context())
		if deviceID == "" {
			http.Error(w, "device token required", http.StatusUnauthorized)
			return
		}

		var body batchUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(body.Events) == 0 {
			http.Error(w, "events required", http.StatusBadRequest)
			return
		}
		if len(body.Events) > MaxBatchSize {
			http.Error(w, "batch too large", http.StatusRequestEntityTooLarge)
			return
		}

		results, err := store.InsertEvents(r.Context(), deviceID, body.Events)
		if err != nil {
			slog.ErrorContext(r.Context(), "insert events", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, batchUploadResponse{Results: results})
	})
}

type summaryResponse struct {
	Results []usage.SummaryRow `json:"results"`
}

// MaxSummaryRange caps the [from, to) span a single summary request may
// cover. Bounded so a runaway dashboard query cannot scan years of data.
// 90 days is more than enough for the views planned in milestone 3
// (today + last-week) with comfortable headroom.
const MaxSummaryRange = 90 * 24 * time.Hour

// UsageSummaryHandler returns aggregate screen-time durations for the
// account on the request context, optionally bucketed by bundle and/or
// day. The Authenticator middleware is sufficient — there is no
// X-Device-Token requirement because the summary spans all of the
// account's devices. The store enforces ownership via the device join.
//
// Query params:
//   - from, to: RFC3339 timestamps. Range is half-open [from, to).
//   - groupBy:  optional, comma-separated subset of {bundle, day}.
func UsageSummaryHandler(store UsageStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		q := r.URL.Query()
		from, err := time.Parse(time.RFC3339, q.Get("from"))
		if err != nil {
			http.Error(w, "invalid from", http.StatusBadRequest)
			return
		}
		to, err := time.Parse(time.RFC3339, q.Get("to"))
		if err != nil {
			http.Error(w, "invalid to", http.StatusBadRequest)
			return
		}
		if !to.After(from) {
			http.Error(w, "to must be after from", http.StatusBadRequest)
			return
		}
		if to.Sub(from) > MaxSummaryRange {
			http.Error(w, "range too large", http.StatusBadRequest)
			return
		}

		var groupBy []usage.SummaryGroup
		if raw := strings.TrimSpace(q.Get("groupBy")); raw != "" {
			for _, part := range strings.Split(raw, ",") {
				switch g := usage.SummaryGroup(strings.TrimSpace(part)); g {
				case usage.GroupByBundle, usage.GroupByDay:
					groupBy = append(groupBy, g)
				default:
					http.Error(w, "invalid groupBy", http.StatusBadRequest)
					return
				}
			}
		}

		rows, err := store.Summarise(r.Context(), usage.SummaryQuery{
			AccountID: accountID,
			From:      from,
			To:        to,
			GroupBy:   groupBy,
		})
		if err != nil {
			if errors.Is(err, usage.ErrInvalidRange) {
				http.Error(w, "invalid range", http.StatusBadRequest)
				return
			}
			slog.ErrorContext(r.Context(), "summarise", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaryResponse{Results: rows})
	})
}
