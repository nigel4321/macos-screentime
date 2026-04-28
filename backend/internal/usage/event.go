// Package usage owns the persistence layer and validation for
// usage_event rows: closed [bundle_id, started_at, ended_at] tuples
// reported by Mac agents and stored against a device.
package usage

import (
	"errors"
	"time"
)

// Event is one closed usage interval reported by a client. The fields
// are deliberately the wire-shape — translation between this and the
// database row happens in Store.InsertEvents.
type Event struct {
	ClientEventID string    `json:"client_event_id"`
	BundleID      string    `json:"bundle_id"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at"`
}

// EventStatus is the per-event outcome returned to the client.
type EventStatus string

// EventStatus values reported per input event.
const (
	StatusAccepted  EventStatus = "accepted"  // newly inserted
	StatusDuplicate EventStatus = "duplicate" // already present (idempotent suppression)
	StatusRejected  EventStatus = "rejected"  // failed validation, see Reason
)

// EventResult mirrors the input order so the client can correlate.
type EventResult struct {
	ClientEventID string      `json:"client_event_id"`
	Status        EventStatus `json:"status"`
	Reason        string      `json:"reason,omitempty"`
}

// Field-length limits guard against unbounded TEXT writes from a
// compromised client. They are lenient enough to fit any realistic
// macOS bundle id or UUID.
const (
	maxBundleIDLen      = 256
	maxClientEventIDLen = 128
)

// AcceptStartedAtFloor is how far in the past an event's started_at
// may be relative to now. Events older than this are rejected outright;
// see Validate. The window is sized to fit comfortably inside the
// previous-month partition that startup pre-creates.
const AcceptStartedAtFloor = -24 * 7 * time.Hour

// AcceptStartedAtCeil is how far in the future an event's started_at
// may be relative to now. A small positive slack accommodates client
// clock skew without admitting events for partitions we have not
// pre-created.
const AcceptStartedAtCeil = 24 * time.Hour

// MaxEventDuration caps a single event's length. Longer intervals are
// almost certainly client bugs (e.g. a missed sleep transition that
// left an event open) and storing them inflates aggregate queries.
const MaxEventDuration = 24 * time.Hour

// Errors returned by Validate. Handlers map them to EventResult.Reason
// strings.
var (
	ErrEmptyClientEventID  = errors.New("client_event_id required")
	ErrLongClientEventID   = errors.New("client_event_id too long")
	ErrEmptyBundleID       = errors.New("bundle_id required")
	ErrLongBundleID        = errors.New("bundle_id too long")
	ErrZeroStartedAt       = errors.New("started_at required")
	ErrZeroEndedAt         = errors.New("ended_at required")
	ErrEndBeforeStart      = errors.New("ended_at must be at or after started_at")
	ErrTooLong             = errors.New("event duration exceeds 24h")
	ErrStartedAtOutOfRange = errors.New("started_at outside accepted window")
)

// Validate checks an event against the rules enforced before any
// database round-trip. now is injectable for tests.
func (e Event) Validate(now time.Time) error {
	switch {
	case e.ClientEventID == "":
		return ErrEmptyClientEventID
	case len(e.ClientEventID) > maxClientEventIDLen:
		return ErrLongClientEventID
	case e.BundleID == "":
		return ErrEmptyBundleID
	case len(e.BundleID) > maxBundleIDLen:
		return ErrLongBundleID
	case e.StartedAt.IsZero():
		return ErrZeroStartedAt
	case e.EndedAt.IsZero():
		return ErrZeroEndedAt
	case e.EndedAt.Before(e.StartedAt):
		return ErrEndBeforeStart
	case e.EndedAt.Sub(e.StartedAt) > MaxEventDuration:
		return ErrTooLong
	}
	floor := now.Add(AcceptStartedAtFloor)
	ceil := now.Add(AcceptStartedAtCeil)
	if e.StartedAt.Before(floor) || e.StartedAt.After(ceil) {
		return ErrStartedAtOutOfRange
	}
	return nil
}
