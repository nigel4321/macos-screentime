package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/usage"
)

// UsageStore is the persistence surface BatchUploadHandler needs.
// usage.Store satisfies it; tests substitute a fake.
type UsageStore interface {
	InsertEvents(ctx context.Context, deviceID string, events []usage.Event) ([]usage.EventResult, error)
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
