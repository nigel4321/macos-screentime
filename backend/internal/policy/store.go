package policy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgUniqueViolation is the SQLSTATE code Postgres returns when a write
// trips a unique constraint. We use it as the authoritative
// version-conflict signal: the (account_id, version) primary key is
// what actually serializes concurrent writers — the in-transaction
// MAX(version) check is just an early-out for the common case.
const pgUniqueViolation = "23505"

// Store reads/writes the per-account policy table. Each PUT inserts a
// new row at version = (current latest + 1); old rows are kept so we
// never silently drop history (useful when investigating "who shipped
// what" later, and free given how small policy bodies are).
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// ErrVersionConflict is returned by Put when the caller's
// expectedVersion does not match the current latest. The handler
// translates this to HTTP 412 Precondition Failed.
var ErrVersionConflict = errors.New("policy: version conflict")

// Current returns the latest [Document] for accountID. When the
// account has never written a policy, returns [EmptyDocument()] (v0,
// no rules) — that's the implicit starting state every account is in.
func (s *Store) Current(ctx context.Context, accountID string) (Document, error) {
	if accountID == "" {
		return Document{}, errors.New("policy: account id required")
	}

	var version int64
	var bodyJSON []byte
	err := s.pool.QueryRow(ctx, `
		SELECT version, body_json
		  FROM policy
		 WHERE account_id = $1::uuid
		 ORDER BY version DESC
		 LIMIT 1
	`, accountID).Scan(&version, &bodyJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmptyDocument(), nil
	}
	if err != nil {
		return Document{}, fmt.Errorf("policy: current: %w", err)
	}

	var doc Document
	if err := json.Unmarshal(bodyJSON, &doc); err != nil {
		return Document{}, fmt.Errorf("policy: unmarshal body: %w", err)
	}
	doc.Version = version
	// Defensive: an old row could have nil slices if a prior writer
	// missed the contract; clients have always seen `[]`.
	if doc.AppLimits == nil {
		doc.AppLimits = []AppLimit{}
	}
	if doc.DowntimeWindows == nil {
		doc.DowntimeWindows = []DowntimeWindow{}
	}
	if doc.BlockList == nil {
		doc.BlockList = []string{}
	}
	return doc, nil
}

// Put inserts a new policy version for accountID. expectedVersion is
// the version the caller observed — Put fails with [ErrVersionConflict]
// if the current latest doesn't match (someone else wrote in between).
//
// Returns the newly-written version on success.
//
// Validation is the caller's responsibility — Put trusts the document
// shape and only enforces the optimistic-concurrency invariant. The
// handler invokes [Document.Validate] before this; integration tests
// also exercise Validate independently.
func (s *Store) Put(ctx context.Context, accountID string, doc Document, expectedVersion int64) (int64, error) {
	if accountID == "" {
		return 0, errors.New("policy: account id required")
	}

	// Run the read + check + write in a single transaction so two
	// concurrent PUTs at the same expectedVersion can't both succeed.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("policy: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentVersion int64
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0)
		  FROM policy
		 WHERE account_id = $1::uuid
	`, accountID).Scan(&currentVersion)
	if err != nil {
		return 0, fmt.Errorf("policy: read current version: %w", err)
	}
	if currentVersion != expectedVersion {
		return currentVersion, ErrVersionConflict
	}

	newVersion := currentVersion + 1
	doc.Version = newVersion

	body, err := json.Marshal(doc)
	if err != nil {
		return 0, fmt.Errorf("policy: marshal body: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO policy (account_id, version, body_json)
		VALUES ($1::uuid, $2, $3::jsonb)
	`, accountID, newVersion, body)
	if err != nil {
		// Concurrent writer landed the same version first. The
		// in-transaction MAX read is an optimisation; the unique
		// constraint is the real arbiter, so translate the violation
		// into the same conflict signal callers already handle.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return newVersion - 1, ErrVersionConflict
		}
		return 0, fmt.Errorf("policy: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("policy: commit: %w", err)
	}
	return newVersion, nil
}
