//go:build integration

package db_test

import (
	"context"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/db"
	"github.com/nigel4321/macos-screentime/backend/internal/dbtest"
)

func TestEnsureMonthPartition_CreatesPartitionAndIsIdempotent(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	march := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	if err := db.EnsureMonthPartition(ctx, pool, march); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := db.EnsureMonthPartition(ctx, pool, march); err != nil {
		t.Fatalf("second call (should be no-op): %v", err)
	}

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'usage_event_2026_03'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("lookup partition: %v", err)
	}
	if !exists {
		t.Fatal("usage_event_2026_03 partition was not created")
	}
}

func TestEnsureMonthPartition_AcceptsInsertsForThatMonth(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	when := time.Date(2026, 5, 10, 9, 30, 0, 0, time.UTC)
	if err := db.EnsureMonthPartition(ctx, pool, when); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	// Seed account + device so the FK is valid.
	var accountID, deviceID string
	if err := pool.QueryRow(ctx, `INSERT INTO account DEFAULT VALUES RETURNING id::text`).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO device (account_id, platform, fingerprint, device_token_hash)
		VALUES ($1::uuid, 'macos', 'fp-1', '\x00') RETURNING id::text
	`, accountID).Scan(&deviceID); err != nil {
		t.Fatalf("insert device: %v", err)
	}

	_, err := pool.Exec(ctx, `
		INSERT INTO usage_event (device_id, client_event_id, bundle_id, started_at, ended_at)
		VALUES ($1::uuid, 'evt-1', 'com.apple.Safari', $2, $3)
	`, deviceID, when, when.Add(5*time.Minute))
	if err != nil {
		t.Fatalf("insert usage_event: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM usage_event_2026_05`).Scan(&count); err != nil {
		t.Fatalf("count partition rows: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row in usage_event_2026_05, got %d", count)
	}
}

func TestEnsureCurrentAndNextMonthPartitions_CreatesBoth(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	now := time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC) // crosses year boundary
	if err := db.EnsureCurrentAndNextMonthPartitions(ctx, pool, now); err != nil {
		t.Fatalf("ensure: %v", err)
	}

	for _, name := range []string{"usage_event_2026_11", "usage_event_2026_12"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, name).Scan(&exists); err != nil {
			t.Fatalf("lookup %s: %v", name, err)
		}
		if !exists {
			t.Errorf("%s partition missing", name)
		}
	}
}
