package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger is the minimal interface /healthz needs from the database
// layer. *pgxpool.Pool satisfies it; tests inject fakes.
type Pinger interface {
	Ping(ctx context.Context) error
}

const pingTimeout = 2 * time.Second

// HealthHandler returns a liveness probe handler. When db is non-nil it
// performs a short-deadline DB ping and reports the result; a failed
// ping yields HTTP 503. When db is nil (dev without DATABASE_URL) the
// handler reports the database as "disabled" and still returns 200.
func HealthHandler(db Pinger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]string{"status": "ok", "database": "disabled"}
		status := http.StatusOK

		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				body["status"] = "degraded"
				body["database"] = "unreachable"
				status = http.StatusServiceUnavailable
			} else {
				body["database"] = "ok"
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	})
}
