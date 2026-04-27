package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the persistence layer for the auth package: accounts,
// account-identity bindings, and pairing codes.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore returns a Store backed by the given connection pool.
func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// FindOrCreateAccountByIdentity looks up an existing account_identity
// for (provider, subject) and returns its account id. If none exists, a
// new account is created and bound to the identity, all in one
// transaction. Concurrent first sign-ins for the same identity may
// produce one orphan account row in the conflict path, which is
// removed before the transaction commits.
func (s *Store) FindOrCreateAccountByIdentity(ctx context.Context, identity Identity) (string, error) {
	var accountID string
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// Speculatively allocate a new account; if the identity
		// already exists, ON CONFLICT preserves the existing binding
		// and we delete the orphan below.
		var newAccount string
		if err := tx.QueryRow(ctx,
			`INSERT INTO account DEFAULT VALUES RETURNING id::text`,
		).Scan(&newAccount); err != nil {
			return err
		}
		var resolved string
		if err := tx.QueryRow(ctx, `
			INSERT INTO account_identity (provider, subject_id, account_id)
			VALUES ($1, $2, $3::uuid)
			ON CONFLICT (provider, subject_id) DO UPDATE
			    SET account_id = account_identity.account_id
			RETURNING account_id::text
		`, identity.Provider, identity.Subject, newAccount).Scan(&resolved); err != nil {
			return err
		}
		if resolved != newAccount {
			if _, err := tx.Exec(ctx,
				`DELETE FROM account WHERE id = $1::uuid`, newAccount); err != nil {
				return err
			}
		}
		accountID = resolved
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("auth: find-or-create account: %w", err)
	}
	return accountID, nil
}

// CreatePairingCode allocates a fresh 6-digit code bound to accountID,
// expiring `ttl` from now (according to the database clock). On
// collision (extraordinarily unlikely) it retries up to 5 times.
func (s *Store) CreatePairingCode(ctx context.Context, accountID string, ttl time.Duration) (string, time.Time, error) {
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code, err := GeneratePairingCode()
		if err != nil {
			return "", time.Time{}, err
		}
		var expiresAt time.Time
		err = s.pool.QueryRow(ctx, `
			INSERT INTO pairing_code (code, account_id, expires_at)
			VALUES ($1, $2::uuid, now() + $3::interval)
			RETURNING expires_at
		`, code, accountID, ttl.String()).Scan(&expiresAt)
		if err == nil {
			return code, expiresAt, nil
		}
		if !isUniqueViolation(err) {
			return "", time.Time{}, fmt.Errorf("auth: insert pairing code: %w", err)
		}
		// Otherwise: collision, retry with a new code.
	}
	return "", time.Time{}, errors.New("auth: exhausted pairing-code attempts")
}

// ConsumePairingCodeAndMerge redeems code and merges srcAccountID into
// the account that initiated pairing. Returns the surviving destination
// account id.
//
// In a single SERIALIZABLE transaction:
//  1. Lock the pairing_code row, validate not expired, not consumed.
//  2. If the caller is already on the destination account (idempotent
//     re-pair), mark consumed and return.
//  3. Otherwise: move account_identity and device rows from src to dst,
//     delete src, mark consumed.
func (s *Store) ConsumePairingCodeAndMerge(ctx context.Context, code, srcAccountID string) (string, error) {
	var dstAccountID string
	err := pgx.BeginTxFunc(ctx, s.pool, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(tx pgx.Tx) error {
		var consumed, expired bool
		err := tx.QueryRow(ctx, `
			SELECT account_id::text,
			       consumed_at IS NOT NULL,
			       expires_at < now()
			FROM pairing_code
			WHERE code = $1
			FOR UPDATE
		`, code).Scan(&dstAccountID, &consumed, &expired)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPairingCodeNotFound
		}
		if err != nil {
			return err
		}
		if consumed {
			return ErrPairingCodeConsumed
		}
		if expired {
			return ErrPairingCodeExpired
		}

		if _, err := tx.Exec(ctx,
			`UPDATE pairing_code SET consumed_at = now() WHERE code = $1`, code); err != nil {
			return err
		}

		if srcAccountID == dstAccountID {
			return nil // already paired — idempotent success
		}

		if _, err := tx.Exec(ctx, `
			UPDATE account_identity SET account_id = $1::uuid WHERE account_id = $2::uuid
		`, dstAccountID, srcAccountID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE device SET account_id = $1::uuid WHERE account_id = $2::uuid
		`, dstAccountID, srcAccountID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM account WHERE id = $1::uuid`, srcAccountID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return dstAccountID, nil
}

// AccountExists reports whether an account row with the given id is
// still present. The Authenticator middleware uses this to reject JWTs
// whose account has been deleted (e.g. by a merge).
func (s *Store) AccountExists(ctx context.Context, accountID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM account WHERE id = $1::uuid)`, accountID,
	).Scan(&exists)
	return exists, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}
