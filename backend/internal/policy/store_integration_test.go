//go:build integration

package policy_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/dbtest"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
)

func newAccount(t *testing.T, authStore *auth.Store, subject string) string {
	t.Helper()
	id, err := authStore.FindOrCreateAccountByIdentity(context.Background(), auth.Identity{Provider: "apple", Subject: subject})
	if err != nil {
		t.Fatalf("create account %s: %v", subject, err)
	}
	return id
}

func sampleDoc() policy.Document {
	return policy.Document{
		AppLimits: []policy.AppLimit{
			{BundleID: "com.example.app", DailyLimitSeconds: 3600},
		},
		DowntimeWindows: []policy.DowntimeWindow{
			{Start: "21:00", End: "07:00", Days: []string{"MONDAY"}},
		},
		BlockList: []string{"com.example.bad"},
	}
}

func TestStore_CurrentReturnsEmptyV0WhenNoRow(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	acct := newAccount(t, authStore, "alice")

	doc, err := store.Current(context.Background(), acct)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if doc.Version != 0 {
		t.Errorf("version: got %d, want 0", doc.Version)
	}
	if doc.AppLimits == nil || doc.DowntimeWindows == nil || doc.BlockList == nil {
		t.Errorf("slices must be non-nil for empty v0, got %+v", doc)
	}
}

func TestStore_PutThenCurrentRoundTrip(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	acct := newAccount(t, authStore, "alice")
	ctx := context.Background()

	v, err := store.Put(ctx, acct, sampleDoc(), 0)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if v != 1 {
		t.Errorf("new version: got %d, want 1", v)
	}

	got, err := store.Current(ctx, acct)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Current version: got %d, want 1", got.Version)
	}
	if len(got.AppLimits) != 1 || got.AppLimits[0].BundleID != "com.example.app" {
		t.Errorf("Current AppLimits: got %+v", got.AppLimits)
	}
	if len(got.BlockList) != 1 || got.BlockList[0] != "com.example.bad" {
		t.Errorf("Current BlockList: got %+v", got.BlockList)
	}
}

func TestStore_VersionIncrementsAcrossPuts(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	acct := newAccount(t, authStore, "alice")
	ctx := context.Background()

	for i, expected := range []int64{0, 1, 2} {
		v, err := store.Put(ctx, acct, sampleDoc(), expected)
		if err != nil {
			t.Fatalf("Put #%d: %v", i+1, err)
		}
		if v != expected+1 {
			t.Errorf("Put #%d: got version %d, want %d", i+1, v, expected+1)
		}
	}

	got, err := store.Current(ctx, acct)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if got.Version != 3 {
		t.Errorf("final version: got %d, want 3", got.Version)
	}
}

func TestStore_StaleExpectedVersionReturnsConflict(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	acct := newAccount(t, authStore, "alice")
	ctx := context.Background()

	if _, err := store.Put(ctx, acct, sampleDoc(), 0); err != nil {
		t.Fatalf("seed Put: %v", err)
	}

	_, err := store.Put(ctx, acct, sampleDoc(), 0) // stale
	if !errors.Is(err, policy.ErrVersionConflict) {
		t.Fatalf("got %v, want ErrVersionConflict", err)
	}
}

func TestStore_ConcurrentPutsOneWinsOneConflicts(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	acct := newAccount(t, authStore, "alice")
	ctx := context.Background()

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := store.Put(ctx, acct, sampleDoc(), 0)
			results[idx] = err
		}(i)
	}
	wg.Wait()

	successes := 0
	conflicts := 0
	for _, err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, policy.ErrVersionConflict):
			conflicts++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Errorf("got %d success / %d conflict, want 1 / 1", successes, conflicts)
	}

	got, _ := store.Current(ctx, acct)
	if got.Version != 1 {
		t.Errorf("current version: got %d, want 1 (only one writer should have landed)", got.Version)
	}
}

func TestStore_CrossAccountIsolation(t *testing.T) {
	pool := dbtest.NewPool(t)
	authStore := auth.NewStore(pool)
	store := policy.NewStore(pool)
	a := newAccount(t, authStore, "alice")
	b := newAccount(t, authStore, "bob")
	ctx := context.Background()

	docA := sampleDoc()
	docA.BlockList = []string{"alice.only"}
	if _, err := store.Put(ctx, a, docA, 0); err != nil {
		t.Fatalf("Put A: %v", err)
	}

	// Bob still sees v0 (no row).
	gotB, err := store.Current(ctx, b)
	if err != nil {
		t.Fatalf("Current B: %v", err)
	}
	if gotB.Version != 0 || len(gotB.BlockList) != 0 {
		t.Errorf("cross-account leak: bob got %+v", gotB)
	}

	// Alice's writes don't bump bob's expected version.
	if _, err := store.Put(ctx, b, sampleDoc(), 0); err != nil {
		t.Fatalf("Put B: %v", err)
	}

	gotA, _ := store.Current(ctx, a)
	if gotA.Version != 1 || len(gotA.BlockList) != 1 || gotA.BlockList[0] != "alice.only" {
		t.Errorf("alice's doc disturbed by bob's write: %+v", gotA)
	}
}
