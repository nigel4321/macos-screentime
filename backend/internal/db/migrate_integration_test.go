//go:build integration

package db_test

import (
	"context"
	"testing"
)

// TestMigrate_AppliesAllSchemaTables checks that the migrated DB contains
// every table the v1 schema declares. pgtestdb has already run migrations
// to produce the template — this test asserts the result.
func TestMigrate_AppliesAllSchemaTables(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	want := []string{"account", "account_identity", "device", "usage_event", "policy"}
	for _, table := range want {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = $1
			)
		`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("lookup %s: %v", table, err)
		}
		if !exists {
			t.Errorf("table %s missing after migrations", table)
		}
	}
}

// TestMigrate_GooseTracksVersion verifies goose recorded the four
// migrations in goose_db_version, so a re-run is a no-op.
func TestMigrate_GooseTracksVersion(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	var maxVersion int64
	err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version`).Scan(&maxVersion)
	if err != nil {
		t.Fatalf("query goose_db_version: %v", err)
	}
	if maxVersion < 4 {
		t.Errorf("expected goose at version >= 4, got %d", maxVersion)
	}
}
