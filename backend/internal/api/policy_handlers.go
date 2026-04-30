package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
)

// PolicyStore is the persistence surface the policy handlers need.
// policy.Store satisfies it; tests substitute a fake.
type PolicyStore interface {
	Current(ctx context.Context, accountID string) (policy.Document, error)
	Put(ctx context.Context, accountID string, doc policy.Document, expectedVersion int64) (int64, error)
}

// versionETagHeader carries the current policy version on every
// successful GET / PUT so clients can use it in `If-Match` on the next
// PUT without parsing the body. We use the literal HTTP `ETag` header
// for cache-friendliness; the value is the raw version number quoted
// per the RFC ("strong" entity tag).
const versionETagHeader = "ETag"

// PolicyCurrentHandler returns the active policy for the authenticated
// account. v0 (empty) is returned when the account has never written
// a policy — clients render it the same way as any other version.
func PolicyCurrentHandler(store PolicyStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		doc, err := store.Current(r.Context(), accountID)
		if err != nil {
			slog.ErrorContext(r.Context(), "policy current", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set(versionETagHeader, formatETag(doc.Version))
		writeJSON(w, http.StatusOK, doc)
	})
}

// PolicyPutHandler updates the policy for the authenticated account.
//
// Optimistic concurrency:
//   - Caller MUST send `If-Match: <version>` (HTTP 428 otherwise — the
//     header is mandatory so two clients can't blindly clobber each
//     other).
//   - When the header doesn't match the current latest, returns 412
//     Precondition Failed; the response body carries the current
//     server-side document so the client can reconcile.
//
// On success the response body is the persisted document (with the new
// version field set); `ETag` carries the new version too.
//
// publisher is invoked after a successful write so live subscribers
// (the WS endpoint) learn the new version. nil is treated as a no-op
// so handler tests and dev-without-broker setups stay simple.
func PolicyPutHandler(store PolicyStore, publisher policy.Publisher) http.Handler {
	if publisher == nil {
		publisher = policy.NopPublisher{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accountID := auth.AccountIDFromContext(r.Context())
		if accountID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
		if ifMatch == "" {
			http.Error(w, "If-Match header required", http.StatusPreconditionRequired)
			return
		}
		expectedVersion, err := parseETag(ifMatch)
		if err != nil {
			http.Error(w, "invalid If-Match", http.StatusBadRequest)
			return
		}

		var doc policy.Document
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		// Caller's `version` field on the body is advisory; the
		// server-side version comes from the persistence layer. We
		// zero it before validation so a hostile body that pre-fills
		// version=999 can't influence the write.
		doc.Version = 0
		if err := doc.Validate(); err != nil {
			if errors.Is(err, policy.ErrInvalidDocument) {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			slog.ErrorContext(r.Context(), "policy validate", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		newVersion, err := store.Put(r.Context(), accountID, doc, expectedVersion)
		if err != nil {
			if errors.Is(err, policy.ErrVersionConflict) {
				// Surface the current document so the client can rebase.
				current, currentErr := store.Current(r.Context(), accountID)
				if currentErr != nil {
					slog.ErrorContext(r.Context(), "policy current after conflict", "err", currentErr)
					http.Error(w, "version conflict", http.StatusPreconditionFailed)
					return
				}
				w.Header().Set(versionETagHeader, formatETag(current.Version))
				writeJSON(w, http.StatusPreconditionFailed, current)
				return
			}
			slog.ErrorContext(r.Context(), "policy put", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		doc.Version = newVersion
		w.Header().Set(versionETagHeader, formatETag(newVersion))
		writeJSON(w, http.StatusOK, doc)
		// Publish after the response is written: the broker is in-process,
		// so this is fast, but treating it as best-effort means a slow
		// fan-out can't affect the writer's HTTP latency.
		publisher.Publish(accountID, newVersion)
	})
}

// formatETag wraps an int64 version in the strong-ETag quoted form
// `"<n>"` so clients can echo it back in If-Match without surprises
// from optional whitespace handling on either side.
func formatETag(v int64) string {
	return `"` + strconv.FormatInt(v, 10) + `"`
}

// parseETag accepts either the quoted form `"5"` (RFC 9110) or the
// bare-int form `5` (lenient — keeps client code simple). Anything
// else is a 400.
func parseETag(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return strconv.ParseInt(s, 10, 64)
}
