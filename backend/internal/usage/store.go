package usage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the persistence layer for usage_event rows.
type Store struct {
	pool *pgxpool.Pool

	// now is injectable for tests; production uses time.Now.
	now func() time.Time
}

// NewStore returns a Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, now: time.Now}
}

// SetNow overrides the clock used for validation. Intended for tests
// that need deterministic windowing.
func (s *Store) SetNow(fn func() time.Time) { s.now = fn }

// InsertEvents validates and inserts up to len(events) rows for the
// given device. Each event is validated and then attempted with
// INSERT ... ON CONFLICT DO NOTHING so a re-upload of the same
// (device_id, client_event_id, started_at) is suppressed.
//
// Returns one EventResult per input event in input order:
//   - StatusAccepted  — newly inserted
//   - StatusDuplicate — already present
//   - StatusRejected  — failed Validate(); Reason carries the rule that fired
//
// A non-nil error means the whole batch could not be processed (e.g.
// connection failure mid-loop). Per-event statuses are still partially
// populated; callers should treat the call as transactional only on
// the no-error path.
func (s *Store) InsertEvents(ctx context.Context, deviceID string, events []Event) ([]EventResult, error) {
	results := make([]EventResult, len(events))

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("usage: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := s.now()
	for i, e := range events {
		results[i].ClientEventID = e.ClientEventID
		if vErr := e.Validate(now); vErr != nil {
			results[i].Status = StatusRejected
			results[i].Reason = vErr.Error()
			continue
		}

		tag, err := tx.Exec(ctx, `
			INSERT INTO usage_event (device_id, client_event_id, bundle_id, started_at, ended_at)
			VALUES ($1::uuid, $2, $3, $4, $5)
			ON CONFLICT (device_id, client_event_id, started_at) DO NOTHING
		`, deviceID, e.ClientEventID, e.BundleID, e.StartedAt, e.EndedAt)
		if err != nil {
			return results, fmt.Errorf("usage: insert event %q: %w", e.ClientEventID, err)
		}
		if tag.RowsAffected() == 1 {
			results[i].Status = StatusAccepted
		} else {
			results[i].Status = StatusDuplicate
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return results, fmt.Errorf("usage: commit: %w", err)
	}
	return results, nil
}

// ErrNoEvents is returned by handlers when a batch upload contains zero
// events. Surfacing it from the store is convenient because both
// handler and store operate on the same []Event input.
var ErrNoEvents = errors.New("usage: no events in batch")
