//go:build integration

package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/dbtest"
)

func TestStore_FindOrCreateAccountByIdentity_Creates(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	id, err := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple",
		Subject:  "001234.apple",
	})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if id == "" {
		t.Fatal("empty account id")
	}

	var bound string
	if err := pool.QueryRow(ctx, `
		SELECT account_id::text FROM account_identity
		WHERE provider = 'apple' AND subject_id = '001234.apple'
	`).Scan(&bound); err != nil {
		t.Fatalf("read identity: %v", err)
	}
	if bound != id {
		t.Errorf("identity points at %q, expected %q", bound, id)
	}
}

func TestStore_FindOrCreateAccountByIdentity_ReusesExisting(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	id1, err := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "abc",
	})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	id2, err := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "abc",
	})
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if id1 != id2 {
		t.Errorf("got fresh account on second call: %q vs %q", id1, id2)
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("account row count: got %d, want 1", n)
	}
}

func TestStore_CreatePairingCode(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})

	code, expiresAt, err := store.CreatePairingCode(ctx, macAccount, 10*time.Minute)
	if err != nil {
		t.Fatalf("CreatePairingCode: %v", err)
	}
	if len(code) != 6 {
		t.Errorf("code length: got %d, want 6", len(code))
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt %v is not in the future", expiresAt)
	}

	var bound string
	if err := pool.QueryRow(ctx, `
		SELECT account_id::text FROM pairing_code WHERE code = $1
	`, code).Scan(&bound); err != nil {
		t.Fatalf("read pairing_code: %v", err)
	}
	if bound != macAccount {
		t.Errorf("pairing_code account: got %q, want %q", bound, macAccount)
	}
}

func TestStore_ConsumePairingCodeAndMerge_Success(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})
	androidAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "1122334455",
	})

	code, _, err := store.CreatePairingCode(ctx, macAccount, 10*time.Minute)
	if err != nil {
		t.Fatalf("CreatePairingCode: %v", err)
	}

	dst, err := store.ConsumePairingCodeAndMerge(ctx, code, androidAccount)
	if err != nil {
		t.Fatalf("ConsumePairingCodeAndMerge: %v", err)
	}
	if dst != macAccount {
		t.Errorf("merge winner: got %q, want %q", dst, macAccount)
	}

	// Both identity rows now point at the Mac account.
	var n int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM account_identity WHERE account_id = $1::uuid
	`, macAccount).Scan(&n); err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if n != 2 {
		t.Errorf("merged identities on Mac account: got %d, want 2", n)
	}

	// The Android account row is gone.
	var androidExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM account WHERE id = $1::uuid)
	`, androidAccount).Scan(&androidExists); err != nil {
		t.Fatalf("exists: %v", err)
	}
	if androidExists {
		t.Error("Android account survived merge")
	}

	// The code is consumed.
	var consumed bool
	_ = pool.QueryRow(ctx, `
		SELECT consumed_at IS NOT NULL FROM pairing_code WHERE code = $1
	`, code).Scan(&consumed)
	if !consumed {
		t.Error("pairing code not marked consumed")
	}
}

func TestStore_ConsumePairingCodeAndMerge_RejectsExpired(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})
	androidAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "g-1",
	})

	code, _, _ := store.CreatePairingCode(ctx, macAccount, 10*time.Minute)
	// Force the row into the past.
	if _, err := pool.Exec(ctx,
		`UPDATE pairing_code SET expires_at = now() - interval '1 minute' WHERE code = $1`,
		code); err != nil {
		t.Fatalf("force expiry: %v", err)
	}

	if _, err := store.ConsumePairingCodeAndMerge(ctx, code, androidAccount); err != auth.ErrPairingCodeExpired {
		t.Errorf("got %v, want ErrPairingCodeExpired", err)
	}
}

func TestStore_ConsumePairingCodeAndMerge_RejectsConsumed(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})
	androidAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "g-1",
	})

	code, _, _ := store.CreatePairingCode(ctx, macAccount, 10*time.Minute)

	if _, err := store.ConsumePairingCodeAndMerge(ctx, code, androidAccount); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	// Second attempt with same code — even if the source account was already deleted by merge.
	_, err := store.ConsumePairingCodeAndMerge(ctx, code, androidAccount)
	if err != auth.ErrPairingCodeConsumed {
		t.Errorf("got %v, want ErrPairingCodeConsumed", err)
	}
}

func TestStore_ConsumePairingCodeAndMerge_RejectsUnknown(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	androidAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "g-1",
	})

	_, err := store.ConsumePairingCodeAndMerge(ctx, "999999", androidAccount)
	if err != auth.ErrPairingCodeNotFound {
		t.Errorf("got %v, want ErrPairingCodeNotFound", err)
	}
}

func TestStore_ConsumePairingCodeAndMerge_IdempotentSelfMerge(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	// Caller is somehow already on the destination account (e.g. they
	// previously merged via a different code, then started over).
	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})
	code, _, _ := store.CreatePairingCode(ctx, macAccount, 10*time.Minute)

	dst, err := store.ConsumePairingCodeAndMerge(ctx, code, macAccount)
	if err != nil {
		t.Fatalf("self-merge: %v", err)
	}
	if dst != macAccount {
		t.Errorf("dst: got %q, want %q", dst, macAccount)
	}
}

func TestStore_RegisterDevice_Inserts(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	account, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})

	tok, err := auth.GenerateDeviceToken()
	if err != nil {
		t.Fatalf("GenerateDeviceToken: %v", err)
	}
	deviceID, err := store.RegisterDevice(ctx, account, auth.PlatformMacOS, "hw-fp-1", auth.HashDeviceToken(tok))
	if err != nil {
		t.Fatalf("RegisterDevice: %v", err)
	}
	if deviceID == "" {
		t.Fatal("empty device id")
	}

	resolved, err := store.ResolveDevice(ctx, account, tok)
	if err != nil {
		t.Fatalf("ResolveDevice: %v", err)
	}
	if resolved != deviceID {
		t.Errorf("ResolveDevice: got %q, want %q", resolved, deviceID)
	}
}

func TestStore_RegisterDevice_IdempotentRotatesToken(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	account, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})

	tok1, _ := auth.GenerateDeviceToken()
	id1, err := store.RegisterDevice(ctx, account, auth.PlatformMacOS, "hw-fp", auth.HashDeviceToken(tok1))
	if err != nil {
		t.Fatalf("first register: %v", err)
	}

	tok2, _ := auth.GenerateDeviceToken()
	id2, err := store.RegisterDevice(ctx, account, auth.PlatformMacOS, "hw-fp", auth.HashDeviceToken(tok2))
	if err != nil {
		t.Fatalf("second register: %v", err)
	}
	if id1 != id2 {
		t.Errorf("device id rotated on re-register: %q vs %q", id1, id2)
	}

	// Old token no longer resolves.
	if _, err := store.ResolveDevice(ctx, account, tok1); err != auth.ErrUnknownDevice {
		t.Errorf("old token resolves: err=%v, want ErrUnknownDevice", err)
	}
	// New token does.
	if _, err := store.ResolveDevice(ctx, account, tok2); err != nil {
		t.Errorf("new token: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM device WHERE account_id = $1::uuid`, account).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("device row count: got %d, want 1", n)
	}
}

func TestStore_RegisterDevice_DistinctFingerprintsCreateRows(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	account, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})

	tok1, _ := auth.GenerateDeviceToken()
	tok2, _ := auth.GenerateDeviceToken()
	id1, _ := store.RegisterDevice(ctx, account, auth.PlatformMacOS, "fp-A", auth.HashDeviceToken(tok1))
	id2, _ := store.RegisterDevice(ctx, account, auth.PlatformAndroid, "fp-B", auth.HashDeviceToken(tok2))
	if id1 == id2 {
		t.Errorf("distinct fingerprints collapsed onto one device row")
	}
}

func TestStore_RegisterDevice_RejectsBadPlatform(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	account, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "001234.apple",
	})
	tok, _ := auth.GenerateDeviceToken()
	if _, err := store.RegisterDevice(ctx, account, "linux", "fp", auth.HashDeviceToken(tok)); err == nil {
		t.Error("expected error for invalid platform")
	}
}

func TestStore_ResolveDevice_ScopedToAccount(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	macAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "a-mac",
	})
	otherAccount, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "google", Subject: "g-other",
	})

	tok, _ := auth.GenerateDeviceToken()
	if _, err := store.RegisterDevice(ctx, macAccount, auth.PlatformMacOS, "fp-mac", auth.HashDeviceToken(tok)); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Same plaintext, queried against the wrong account, must not resolve.
	if _, err := store.ResolveDevice(ctx, otherAccount, tok); err != auth.ErrUnknownDevice {
		t.Errorf("cross-account resolve: err=%v, want ErrUnknownDevice", err)
	}
}

func TestStore_ResolveDevice_UnknownToken(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	account, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "x",
	})

	if _, err := store.ResolveDevice(ctx, account, "never-issued"); err != auth.ErrUnknownDevice {
		t.Errorf("got %v, want ErrUnknownDevice", err)
	}
}

func TestStore_AccountExists(t *testing.T) {
	pool := dbtest.NewPool(t)
	store := auth.NewStore(pool)
	ctx := context.Background()

	id, _ := store.FindOrCreateAccountByIdentity(ctx, auth.Identity{
		Provider: "apple", Subject: "x",
	})
	ok, err := store.AccountExists(ctx, id)
	if err != nil || !ok {
		t.Fatalf("expected existing account: ok=%v err=%v", ok, err)
	}
	ok, err = store.AccountExists(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("AccountExists: %v", err)
	}
	if ok {
		t.Error("nonexistent account reported as existing")
	}
}
