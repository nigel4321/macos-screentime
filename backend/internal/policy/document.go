// Package policy owns the persistence layer, validation, and document
// shape for the per-account `policy` rows: each row is an immutable
// version of the account's enforcement rules (per-app daily limits,
// scheduled downtime windows, hard block list).
package policy

import (
	"errors"
	"fmt"
	"regexp"
)

// Document is the wire + storage shape of one policy version. The
// struct doubles as the JSONB body persisted in `policy.body_json` and
// the response/request body of `GET|PUT /v1/policy/current`.
//
// Slices are non-nil at the boundary so JSON encoding produces `[]`
// rather than `null` — clients (Android `:core-data` already does
// this) iterate without nil-checks.
type Document struct {
	Version         int64            `json:"version"`
	AppLimits       []AppLimit       `json:"app_limits"`
	DowntimeWindows []DowntimeWindow `json:"downtime_windows"`
	BlockList       []string         `json:"block_list"`
}

// AppLimit caps how long the user may run a given bundle id per local
// day. The Mac agent's PolicyEngine resets at the start of the user's
// local day (DST handled there); the backend stores the cap only.
type AppLimit struct {
	BundleID          string `json:"bundle_id"`
	DailyLimitSeconds int64  `json:"daily_limit_seconds"`
}

// DowntimeWindow blocks every app on the BlockList during [Start, End)
// on the listed days. Crossing midnight is allowed (Start > End); the
// PolicyEngine handles that case.
type DowntimeWindow struct {
	Start string   `json:"start"` // "HH:MM" (24h)
	End   string   `json:"end"`   // "HH:MM" (24h)
	Days  []string `json:"days"`  // ["MONDAY", ..., "SUNDAY"]
}

// EmptyDocument returns a fresh v0 policy — every account starts here
// implicitly until it issues its first `PUT /v1/policy`.
func EmptyDocument() Document {
	return Document{
		Version:         0,
		AppLimits:       []AppLimit{},
		DowntimeWindows: []DowntimeWindow{},
		BlockList:       []string{},
	}
}

// Validation rules. Bounds are deliberately loose so future Mac/Android
// policy editors don't have to round-trip them — they exist to keep a
// hostile or buggy client from writing rows that brick the engine.
const (
	maxBundleIDLength    = 256
	maxAppLimits         = 200
	maxDowntimeWindows   = 50
	maxBlockListEntries  = 200
	maxDailyLimitSeconds = 24 * 60 * 60 // a 24h cap is effectively "no limit"
)

var (
	// HH:MM, leading zero required so parsing is unambiguous.
	timeOfDayPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

	weekdaySet = map[string]bool{
		"MONDAY":    true,
		"TUESDAY":   true,
		"WEDNESDAY": true,
		"THURSDAY":  true,
		"FRIDAY":    true,
		"SATURDAY":  true,
		"SUNDAY":    true,
	}
)

// ErrInvalidDocument is returned by Validate when the document fails
// any rule. The wrapped message names the offending field.
var ErrInvalidDocument = errors.New("policy: invalid document")

// Validate enforces shape + bounds. Returns ErrInvalidDocument wrapped
// with a field-pointing message on failure. Idempotent.
func (d *Document) Validate() error {
	if len(d.AppLimits) > maxAppLimits {
		return fmt.Errorf("%w: app_limits exceeds %d entries", ErrInvalidDocument, maxAppLimits)
	}
	for i, a := range d.AppLimits {
		if a.BundleID == "" {
			return fmt.Errorf("%w: app_limits[%d].bundle_id is empty", ErrInvalidDocument, i)
		}
		if len(a.BundleID) > maxBundleIDLength {
			return fmt.Errorf("%w: app_limits[%d].bundle_id too long", ErrInvalidDocument, i)
		}
		if a.DailyLimitSeconds <= 0 {
			return fmt.Errorf("%w: app_limits[%d].daily_limit_seconds must be > 0", ErrInvalidDocument, i)
		}
		if a.DailyLimitSeconds > maxDailyLimitSeconds {
			return fmt.Errorf("%w: app_limits[%d].daily_limit_seconds exceeds 24h", ErrInvalidDocument, i)
		}
	}

	if len(d.DowntimeWindows) > maxDowntimeWindows {
		return fmt.Errorf("%w: downtime_windows exceeds %d entries", ErrInvalidDocument, maxDowntimeWindows)
	}
	for i, w := range d.DowntimeWindows {
		if !timeOfDayPattern.MatchString(w.Start) {
			return fmt.Errorf("%w: downtime_windows[%d].start %q must be HH:MM", ErrInvalidDocument, i, w.Start)
		}
		if !timeOfDayPattern.MatchString(w.End) {
			return fmt.Errorf("%w: downtime_windows[%d].end %q must be HH:MM", ErrInvalidDocument, i, w.End)
		}
		if w.Start == w.End {
			return fmt.Errorf("%w: downtime_windows[%d].start equals end (zero-length window)", ErrInvalidDocument, i)
		}
		if len(w.Days) == 0 {
			return fmt.Errorf("%w: downtime_windows[%d].days is empty", ErrInvalidDocument, i)
		}
		seen := map[string]bool{}
		for _, day := range w.Days {
			if !weekdaySet[day] {
				return fmt.Errorf("%w: downtime_windows[%d].days contains %q (must be MONDAY..SUNDAY)", ErrInvalidDocument, i, day)
			}
			if seen[day] {
				return fmt.Errorf("%w: downtime_windows[%d].days has duplicate %q", ErrInvalidDocument, i, day)
			}
			seen[day] = true
		}
	}

	if len(d.BlockList) > maxBlockListEntries {
		return fmt.Errorf("%w: block_list exceeds %d entries", ErrInvalidDocument, maxBlockListEntries)
	}
	for i, b := range d.BlockList {
		if b == "" {
			return fmt.Errorf("%w: block_list[%d] is empty", ErrInvalidDocument, i)
		}
		if len(b) > maxBundleIDLength {
			return fmt.Errorf("%w: block_list[%d] too long", ErrInvalidDocument, i)
		}
	}

	return nil
}
