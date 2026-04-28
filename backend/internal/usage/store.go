package usage

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// SummaryGroup is one axis a Summarise query can group on.
type SummaryGroup string

// SummaryGroup axes accepted by Summarise.
const (
	GroupByBundle SummaryGroup = "bundle"
	GroupByDay    SummaryGroup = "day"
)

// SummaryRow is one aggregate bucket. BundleID is set iff GroupByBundle
// was requested; Day is set iff GroupByDay was requested. DurationSeconds
// is always populated.
type SummaryRow struct {
	BundleID        string `json:"bundle_id,omitempty"`
	Day             string `json:"day,omitempty"` // ISO YYYY-MM-DD in UTC
	DurationSeconds int64  `json:"duration_seconds"`
}

// SummaryQuery captures the inputs to Summarise. Range is half-open:
// [From, To) — From inclusive, To exclusive — to match SQL conventions
// and avoid double-counting events on the boundary.
type SummaryQuery struct {
	AccountID string
	From      time.Time
	To        time.Time
	GroupBy   []SummaryGroup
}

// ErrInvalidRange is returned by Summarise when To <= From or either
// bound is zero.
var ErrInvalidRange = errors.New("usage: invalid time range")

// Summarise returns aggregate usage for all devices owned by
// AccountID within [From, To), optionally grouped on the requested
// axes. Without GroupBy, a single total row is returned (always — even
// if zero events match — so the client can render an empty state).
func (s *Store) Summarise(ctx context.Context, q SummaryQuery) ([]SummaryRow, error) {
	if q.AccountID == "" {
		return nil, errors.New("usage: account id required")
	}
	if q.From.IsZero() || q.To.IsZero() || !q.To.After(q.From) {
		return nil, ErrInvalidRange
	}

	groupBundle, groupDay := false, false
	for _, g := range q.GroupBy {
		switch g {
		case GroupByBundle:
			groupBundle = true
		case GroupByDay:
			groupDay = true
		default:
			return nil, fmt.Errorf("usage: unknown groupBy %q", g)
		}
	}

	// Build the SELECT and GROUP BY clauses dynamically based on the
	// requested axes. The duration expression is the same in every
	// case; only the keys vary.
	selects := []string{}
	groups := []string{}
	if groupBundle {
		selects = append(selects, "bundle_id")
		groups = append(groups, "bundle_id")
	}
	if groupDay {
		selects = append(selects, "to_char(date_trunc('day', started_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day")
		groups = append(groups, "date_trunc('day', started_at AT TIME ZONE 'UTC')")
	}
	selects = append(selects, "COALESCE(SUM(EXTRACT(EPOCH FROM (ended_at - started_at)))::bigint, 0) AS duration_s")

	sqlStr := "SELECT " + strings.Join(selects, ", ") +
		" FROM usage_event ue" +
		" JOIN device d ON d.id = ue.device_id" +
		" WHERE d.account_id = $1::uuid" +
		"   AND ue.started_at >= $2" +
		"   AND ue.started_at <  $3"
	if len(groups) > 0 {
		sqlStr += " GROUP BY " + strings.Join(groups, ", ")
		// Stable order: bundle then day if both, else by whichever is present.
		sqlStr += " ORDER BY " + strings.Join(groups, ", ")
	}

	rows, err := s.pool.Query(ctx, sqlStr, q.AccountID, q.From, q.To)
	if err != nil {
		return nil, fmt.Errorf("usage: summarise: %w", err)
	}
	defer rows.Close()

	out := []SummaryRow{}
	for rows.Next() {
		var r SummaryRow
		// scan target list mirrors selects order
		dest := []any{}
		if groupBundle {
			dest = append(dest, &r.BundleID)
		}
		if groupDay {
			dest = append(dest, &r.Day)
		}
		dest = append(dest, &r.DurationSeconds)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("usage: scan: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("usage: rows: %w", err)
	}
	// No-grouping case returns 0 rows on empty range; surface a zero
	// total so the client always has one row to render.
	if len(out) == 0 && !groupBundle && !groupDay {
		out = append(out, SummaryRow{DurationSeconds: 0})
	}
	return out, nil
}
