package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Execer is the minimal subset of pgx that the partition helper needs.
// pgxpool.Pool, pgx.Conn, and pgx.Tx all satisfy it.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// EnsureMonthPartition creates the usage_event partition for the calendar
// month containing anyTimeInMonth (interpreted in UTC). It is idempotent:
// re-running with a time in the same month is a no-op.
func EnsureMonthPartition(ctx context.Context, ex Execer, anyTimeInMonth time.Time) error {
	monthStart := time.Date(
		anyTimeInMonth.Year(), anyTimeInMonth.Month(), 1,
		0, 0, 0, 0, time.UTC,
	)
	nextMonth := monthStart.AddDate(0, 1, 0)

	name := fmt.Sprintf("usage_event_%04d_%02d", monthStart.Year(), int(monthStart.Month()))
	quoted := pgx.Identifier{name}.Sanitize()

	stmt := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s PARTITION OF usage_event "+
			"FOR VALUES FROM ('%s') TO ('%s')",
		quoted,
		monthStart.Format("2006-01-02 15:04:05-07"),
		nextMonth.Format("2006-01-02 15:04:05-07"),
	)
	if _, err := ex.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("ensure partition %s: %w", name, err)
	}
	return nil
}

// EnsureCurrentAndNextMonthPartitions is the convenience caller for
// startup: it provisions the partition for "now" and the one after, so
// inserts crossing the month boundary do not fail at midnight UTC.
func EnsureCurrentAndNextMonthPartitions(ctx context.Context, ex Execer, now time.Time) error {
	if err := EnsureMonthPartition(ctx, ex, now); err != nil {
		return err
	}
	return EnsureMonthPartition(ctx, ex, now.AddDate(0, 1, 0))
}
