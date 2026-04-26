package api

import (
	"encoding/json"
	"net/http"
)

// HealthHandler returns a liveness probe handler that always reports
// the process is up. Future revisions add a database ping (see §2.2).
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
}
