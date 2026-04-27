package usage

import (
	"strings"
	"testing"
	"time"
)

func validEventAt(now time.Time) Event {
	return Event{
		ClientEventID: "evt-1",
		BundleID:      "com.apple.Safari",
		StartedAt:     now.Add(-1 * time.Hour),
		EndedAt:       now.Add(-30 * time.Minute),
	}
}

func TestEvent_Validate_HappyPath(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	if err := validEventAt(now).Validate(now); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
}

func TestEvent_Validate_Rules(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		mut  func(*Event)
		want error
	}{
		{"empty client_event_id", func(e *Event) { e.ClientEventID = "" }, ErrEmptyClientEventID},
		{"long client_event_id", func(e *Event) { e.ClientEventID = strings.Repeat("x", maxClientEventIDLen+1) }, ErrLongClientEventID},
		{"empty bundle_id", func(e *Event) { e.BundleID = "" }, ErrEmptyBundleID},
		{"long bundle_id", func(e *Event) { e.BundleID = strings.Repeat("x", maxBundleIDLen+1) }, ErrLongBundleID},
		{"zero started_at", func(e *Event) { e.StartedAt = time.Time{} }, ErrZeroStartedAt},
		{"zero ended_at", func(e *Event) { e.EndedAt = time.Time{} }, ErrZeroEndedAt},
		{"end before start", func(e *Event) { e.EndedAt = e.StartedAt.Add(-time.Second) }, ErrEndBeforeStart},
		{"duration too long", func(e *Event) { e.EndedAt = e.StartedAt.Add(MaxEventDuration + time.Second) }, ErrTooLong},
		{"started_at too old", func(e *Event) { e.StartedAt = now.Add(AcceptStartedAtFloor - time.Hour); e.EndedAt = e.StartedAt.Add(time.Minute) }, ErrStartedAtOutOfRange},
		{"started_at too future", func(e *Event) { e.StartedAt = now.Add(AcceptStartedAtCeil + time.Hour); e.EndedAt = e.StartedAt.Add(time.Minute) }, ErrStartedAtOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := validEventAt(now)
			tt.mut(&e)
			if err := e.Validate(now); err != tt.want {
				t.Errorf("got %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEvent_Validate_AllowsZeroLengthEvent(t *testing.T) {
	now := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	e := validEventAt(now)
	e.EndedAt = e.StartedAt
	if err := e.Validate(now); err != nil {
		t.Errorf("zero-length event should be valid: %v", err)
	}
}
