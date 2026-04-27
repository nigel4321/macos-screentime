//go:build integration

package db_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/pressly/goose/v3"

	"github.com/nigel4321/macos-screentime/backend/migrations"
)

// embeddedMigrator runs the goose migrations from the embedded FS. It
// satisfies pgtestdb.Migrator so each test gets a freshly migrated
// database (cached as a template by pgtestdb).
type embeddedMigrator struct{}

func (embeddedMigrator) Hash() (string, error) {
	h := sha256.New()
	err := fs.WalkDir(migrations.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(migrations.FS, path)
		if err != nil {
			return err
		}
		_, _ = h.Write([]byte(path))
		_, _ = h.Write(b)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (embeddedMigrator) Migrate(ctx context.Context, sqlDB *sql.DB, _ pgtestdb.Config) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrations.FS)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}

// testDBConfig builds a pgtestdb.Config from PG_TEST_* env vars with
// dev-friendly defaults.
func testDBConfig() pgtestdb.Config {
	return pgtestdb.Config{
		DriverName: "pgx",
		Host:       envOr("PG_TEST_HOST", "localhost"),
		Port:       envOr("PG_TEST_PORT", "5432"),
		User:       envOr("PG_TEST_USER", "postgres"),
		Password:   envOr("PG_TEST_PASSWORD", "postgres"),
		Options:    envOr("PG_TEST_OPTIONS", "sslmode=disable"),
	}
}

func envOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

// newTestPool returns a freshly migrated database wrapped as a pgxpool.
// Cleanup is registered via t.Cleanup.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	conf := pgtestdb.Custom(t, testDBConfig(), embeddedMigrator{})
	pool, err := pgxpool.New(context.Background(), conf.URL())
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
