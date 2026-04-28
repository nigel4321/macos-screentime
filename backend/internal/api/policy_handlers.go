package api

import (
	"net/http"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
)

// PolicyResponse is the wire shape of /v1/policy/current. Version is
// monotonically incremented by the server on each PUT; the rule families
// match the policy domains called out in ARCHITECTURE.md (per-app limits,
// scheduled downtime, hard block list). The struct ships with empty
// slices so JSON encoding produces `[]` rather than `null` — clients
// can iterate without nil-checks.
type PolicyResponse struct {
	Version         int      `json:"version"`
	AppLimits       []string `json:"app_limits"`
	DowntimeWindows []string `json:"downtime_windows"`
	BlockList       []string `json:"block_list"`
}

// PolicyCurrentHandler returns the active policy for the authenticated
// account. M2.7 stub: always returns the empty v0 policy. The persistence
// layer lands in milestone 3 alongside the policy editor.
func PolicyCurrentHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth.AccountIDFromContext(r.Context()) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeJSON(w, http.StatusOK, PolicyResponse{
			Version:         0,
			AppLimits:       []string{},
			DowntimeWindows: []string{},
			BlockList:       []string{},
		})
	})
}
