//go:build integration

package usage_test

import (
	"context"
	"testing"
	"time"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/db"
	"github.com/nigel4321/macos-screentime/backend/internal/dbtest"
	"github.com/nigel4321/macos-screentime/backend/internal/usage"
)

// withDevice provisions an account + device and returns the device id
// alongside a usage store anchored to a fixed clock so validation
// windows are deterministic.
func withDevice(t *testing.T) (deviceID string, store *usage.Store, now time.Time) {
	t.Helper()
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	if err := db.EnsurePartitionsAroundNow(ctx, pool, time.Now().UTC()); err != nil {
		t.Fatalf("ensure partitions: %v", err)
	}

	authStore := auth.NewStore(pool)
	account, err := authStore.FindOrCreateAccountByIdentity(ctx, auth.Identity{Provider: "apple", Subject: "u"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	tok, _ := auth.GenerateDeviceToken()
	deviceID, err = authStore.RegisterDevice(ctx, account, auth.PlatformMacOS, "fp", auth.HashDeviceToken(tok))
	if err != nil {
		t.Fatalf("register device: %v", err)
	}

	store = usage.NewStore(pool)
	now = time.Now().UTC()
	store.SetNow(func() time.Time { return now })
	return deviceID, store, now
}

func TestUsageStore_InsertEvents_AcceptsThenDuplicates(t *testing.T) {
	deviceID, store, now := withDevice(t)
	ctx := context.Background()

	events := []usage.Event{{
		ClientEventID: "evt-1",
		BundleID:      "com.apple.Safari",
		StartedAt:     now.Add(-1 * time.Hour),
		EndedAt:       now.Add(-30 * time.Minute),
	}}

	first, err := store.InsertEvents(ctx, deviceID, events)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if len(first) != 1 || first[0].Status != usage.StatusAccepted {
		t.Fatalf("first result: got %+v", first)
	}

	second, err := store.InsertEvents(ctx, deviceID, events)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if len(second) != 1 || second[0].Status != usage.StatusDuplicate {
		t.Errorf("second result: got %+v", second)
	}
}

func TestUsageStore_InsertEvents_ValidationRejectsMixedBatch(t *testing.T) {
	deviceID, store, now := withDevice(t)
	ctx := context.Background()

	events := []usage.Event{
		{ // ok
			ClientEventID: "ok-1",
			BundleID:      "com.apple.Safari",
			StartedAt:     now.Add(-2 * time.Hour),
			EndedAt:       now.Add(-1 * time.Hour),
		},
		{ // missing bundle
			ClientEventID: "bad-1",
			BundleID:      "",
			StartedAt:     now.Add(-2 * time.Hour),
			EndedAt:       now.Add(-1 * time.Hour),
		},
		{ // ok
			ClientEventID: "ok-2",
			BundleID:      "com.apple.Mail",
			StartedAt:     now.Add(-30 * time.Minute),
			EndedAt:       now.Add(-15 * time.Minute),
		},
	}

	results, err := store.InsertEvents(ctx, deviceID, events)
	if err != nil {
		t.Fatalf("InsertEvents: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("results length: got %d", len(results))
	}
	if results[0].Status != usage.StatusAccepted {
		t.Errorf("results[0]: got %+v", results[0])
	}
	if results[1].Status != usage.StatusRejected || results[1].Reason == "" {
		t.Errorf("results[1]: got %+v", results[1])
	}
	if results[2].Status != usage.StatusAccepted {
		t.Errorf("results[2]: got %+v", results[2])
	}
}

func TestUsageStore_InsertEvents_DistinctStartTimesWithSameClientID(t *testing.T) {
	// The unique key includes started_at, so the same client_event_id
	// at two different start times is intentionally allowed (a client
	// regenerating an id is a bug, but the row-count behaviour should
	// match the schema).
	deviceID, store, now := withDevice(t)
	ctx := context.Background()

	results, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{ClientEventID: "evt", BundleID: "com.app", StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-1 * time.Hour)},
		{ClientEventID: "evt", BundleID: "com.app", StartedAt: now.Add(-30 * time.Minute), EndedAt: now.Add(-15 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("InsertEvents: %v", err)
	}
	if results[0].Status != usage.StatusAccepted || results[1].Status != usage.StatusAccepted {
		t.Errorf("expected both accepted: got %+v", results)
	}
}

func TestUsageStore_InsertEvents_RejectsOutOfWindow(t *testing.T) {
	deviceID, store, now := withDevice(t)
	ctx := context.Background()

	results, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{
			ClientEventID: "old",
			BundleID:      "com.app",
			StartedAt:     now.Add(usage.AcceptStartedAtFloor - time.Hour),
			EndedAt:       now.Add(usage.AcceptStartedAtFloor - time.Hour + time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("InsertEvents: %v", err)
	}
	if results[0].Status != usage.StatusRejected {
		t.Errorf("got %+v, want rejected", results[0])
	}
}

// withAccountAndDevice is like withDevice but returns the account id too,
// so the caller can drive Summarise queries scoped to that account.
func withAccountAndDevice(t *testing.T) (accountID, deviceID string, store *usage.Store, now time.Time) {
	t.Helper()
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	if err := db.EnsurePartitionsAroundNow(ctx, pool, time.Now().UTC()); err != nil {
		t.Fatalf("ensure partitions: %v", err)
	}

	authStore := auth.NewStore(pool)
	accountID, err := authStore.FindOrCreateAccountByIdentity(ctx, auth.Identity{Provider: "apple", Subject: "u"})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	tok, _ := auth.GenerateDeviceToken()
	deviceID, err = authStore.RegisterDevice(ctx, accountID, auth.PlatformMacOS, "fp", auth.HashDeviceToken(tok))
	if err != nil {
		t.Fatalf("register device: %v", err)
	}

	store = usage.NewStore(pool)
	now = time.Now().UTC()
	store.SetNow(func() time.Time { return now })
	return accountID, deviceID, store, now
}

func TestUsageStore_Summarise_TotalAndByBundle(t *testing.T) {
	accountID, deviceID, store, now := withAccountAndDevice(t)
	ctx := context.Background()

	// Two Safari events totalling 45 min, one Mail event of 15 min.
	_, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{ClientEventID: "s1", BundleID: "com.apple.Safari",
			StartedAt: now.Add(-3 * time.Hour), EndedAt: now.Add(-3*time.Hour + 30*time.Minute)},
		{ClientEventID: "s2", BundleID: "com.apple.Safari",
			StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-2*time.Hour + 15*time.Minute)},
		{ClientEventID: "m1", BundleID: "com.apple.Mail",
			StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1*time.Hour + 15*time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	from := now.Add(-24 * time.Hour)
	to := now.Add(1 * time.Hour)

	// No grouping — single total row.
	rows, err := store.Summarise(ctx, usage.SummaryQuery{AccountID: accountID, From: from, To: to})
	if err != nil {
		t.Fatalf("Summarise total: %v", err)
	}
	if len(rows) != 1 || rows[0].DurationSeconds != int64((30+15+15)*60) {
		t.Errorf("total: got %+v", rows)
	}

	// Group by bundle.
	rows, err = store.Summarise(ctx, usage.SummaryQuery{
		AccountID: accountID, From: from, To: to,
		GroupBy: []usage.SummaryGroup{usage.GroupByBundle},
	})
	if err != nil {
		t.Fatalf("Summarise by bundle: %v", err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		got[r.BundleID] = r.DurationSeconds
	}
	if got["com.apple.Safari"] != int64(45*60) || got["com.apple.Mail"] != int64(15*60) {
		t.Errorf("by bundle: got %+v", got)
	}
}

func TestUsageStore_Summarise_GroupByDay(t *testing.T) {
	accountID, deviceID, store, _ := withAccountAndDevice(t)
	ctx := context.Background()

	// Pin the clock inside the day so events on either side are
	// inside the validation window.
	pinned := time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
	store.SetNow(func() time.Time { return pinned })

	apr26 := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	apr27 := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	_, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{ClientEventID: "a", BundleID: "com.app",
			StartedAt: apr26, EndedAt: apr26.Add(20 * time.Minute)},
		{ClientEventID: "b", BundleID: "com.app",
			StartedAt: apr27, EndedAt: apr27.Add(40 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	rows, err := store.Summarise(ctx, usage.SummaryQuery{
		AccountID: accountID,
		From:      time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC),
		To:        time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC),
		GroupBy:   []usage.SummaryGroup{usage.GroupByDay},
	})
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		got[r.Day] = r.DurationSeconds
	}
	if got["2026-04-26"] != int64(20*60) || got["2026-04-27"] != int64(40*60) {
		t.Errorf("by day: got %+v", got)
	}
}

func TestUsageStore_Summarise_CrossAccountIsolation(t *testing.T) {
	// Two accounts share a pool; one account's events must not appear
	// in the other's summary.
	pool := dbtest.NewPool(t)
	ctx := context.Background()
	if err := db.EnsurePartitionsAroundNow(ctx, pool, time.Now().UTC()); err != nil {
		t.Fatalf("ensure partitions: %v", err)
	}

	authStore := auth.NewStore(pool)
	store := usage.NewStore(pool)
	now := time.Now().UTC()
	store.SetNow(func() time.Time { return now })

	mkDevice := func(subject string) (string, string) {
		acct, err := authStore.FindOrCreateAccountByIdentity(ctx, auth.Identity{Provider: "apple", Subject: subject})
		if err != nil {
			t.Fatalf("account %s: %v", subject, err)
		}
		tok, _ := auth.GenerateDeviceToken()
		dev, err := authStore.RegisterDevice(ctx, acct, auth.PlatformMacOS, "fp-"+subject, auth.HashDeviceToken(tok))
		if err != nil {
			t.Fatalf("device %s: %v", subject, err)
		}
		return acct, dev
	}
	acctA, devA := mkDevice("alice")
	acctB, devB := mkDevice("bob")

	_, err := store.InsertEvents(ctx, devA, []usage.Event{
		{ClientEventID: "a", BundleID: "com.app",
			StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-30 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed A: %v", err)
	}
	_, err = store.InsertEvents(ctx, devB, []usage.Event{
		{ClientEventID: "b", BundleID: "com.app",
			StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-50 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed B: %v", err)
	}

	from := now.Add(-24 * time.Hour)
	to := now.Add(1 * time.Hour)

	rowsA, err := store.Summarise(ctx, usage.SummaryQuery{AccountID: acctA, From: from, To: to})
	if err != nil {
		t.Fatalf("Summarise A: %v", err)
	}
	if len(rowsA) != 1 || rowsA[0].DurationSeconds != int64(30*60) {
		t.Errorf("A: got %+v, want 30min only", rowsA)
	}

	rowsB, err := store.Summarise(ctx, usage.SummaryQuery{AccountID: acctB, From: from, To: to})
	if err != nil {
		t.Fatalf("Summarise B: %v", err)
	}
	if len(rowsB) != 1 || rowsB[0].DurationSeconds != int64(10*60) {
		t.Errorf("B: got %+v, want 10min only", rowsB)
	}
}

func TestUsageStore_Summarise_EmptyRangeReturnsZero(t *testing.T) {
	accountID, _, store, now := withAccountAndDevice(t)
	ctx := context.Background()

	rows, err := store.Summarise(ctx, usage.SummaryQuery{
		AccountID: accountID,
		From:      now.Add(-24 * time.Hour),
		To:        now.Add(1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	if len(rows) != 1 || rows[0].DurationSeconds != 0 {
		t.Errorf("empty range: got %+v, want single zero row", rows)
	}
}

func TestUsageStore_AppMetadata_RoundTripJoinsIntoSummary(t *testing.T) {
	accountID, deviceID, store, now := withAccountAndDevice(t)
	ctx := context.Background()

	// Two events for two bundles; only Safari has metadata.
	_, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{ClientEventID: "s1", BundleID: "com.apple.Safari",
			StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-2*time.Hour + 30*time.Minute)},
		{ClientEventID: "m1", BundleID: "com.apple.Mail",
			StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1*time.Hour + 15*time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed events: %v", err)
	}

	written, err := store.UpsertAppMetadataBatch(ctx, accountID, map[string]string{
		"com.apple.Safari": "Safari",
	})
	if err != nil {
		t.Fatalf("upsert metadata: %v", err)
	}
	if written != 1 {
		t.Errorf("written: got %d, want 1", written)
	}

	rows, err := store.Summarise(ctx, usage.SummaryQuery{
		AccountID: accountID, From: now.Add(-24 * time.Hour), To: now.Add(time.Hour),
		GroupBy: []usage.SummaryGroup{usage.GroupByBundle},
	})
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	got := map[string]string{}
	for _, r := range rows {
		got[r.BundleID] = r.DisplayName
	}
	if got["com.apple.Safari"] != "Safari" {
		t.Errorf("Safari name: got %q, want %q", got["com.apple.Safari"], "Safari")
	}
	if got["com.apple.Mail"] != "" {
		t.Errorf("Mail name: got %q, want empty (no metadata row)", got["com.apple.Mail"])
	}
}

func TestUsageStore_AppMetadata_LatestWriteWins(t *testing.T) {
	accountID, deviceID, store, now := withAccountAndDevice(t)
	ctx := context.Background()

	_, err := store.InsertEvents(ctx, deviceID, []usage.Event{
		{ClientEventID: "s1", BundleID: "com.app",
			StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1*time.Hour + 30*time.Minute)},
	})
	if err != nil {
		t.Fatalf("seed events: %v", err)
	}

	if _, err := store.UpsertAppMetadataBatch(ctx, accountID, map[string]string{"com.app": "Old Name"}); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	if _, err := store.UpsertAppMetadataBatch(ctx, accountID, map[string]string{"com.app": "New Name"}); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	rows, err := store.Summarise(ctx, usage.SummaryQuery{
		AccountID: accountID, From: now.Add(-24 * time.Hour), To: now.Add(time.Hour),
		GroupBy: []usage.SummaryGroup{usage.GroupByBundle},
	})
	if err != nil {
		t.Fatalf("Summarise: %v", err)
	}
	if len(rows) != 1 || rows[0].DisplayName != "New Name" {
		t.Errorf("latest-write-wins: got %+v", rows)
	}
}

func TestUsageStore_AppMetadata_CrossAccountIsolation(t *testing.T) {
	pool := dbtest.NewPool(t)
	ctx := context.Background()

	if err := db.EnsurePartitionsAroundNow(ctx, pool, time.Now().UTC()); err != nil {
		t.Fatalf("ensure partitions: %v", err)
	}

	authStore := auth.NewStore(pool)
	a, _ := authStore.FindOrCreateAccountByIdentity(ctx, auth.Identity{Provider: "apple", Subject: "alice"})
	b, _ := authStore.FindOrCreateAccountByIdentity(ctx, auth.Identity{Provider: "apple", Subject: "bob"})

	tokA, _ := auth.GenerateDeviceToken()
	tokB, _ := auth.GenerateDeviceToken()
	devA, _ := authStore.RegisterDevice(ctx, a, auth.PlatformMacOS, "fp-a", auth.HashDeviceToken(tokA))
	devB, _ := authStore.RegisterDevice(ctx, b, auth.PlatformMacOS, "fp-b", auth.HashDeviceToken(tokB))

	store := usage.NewStore(pool)
	now := time.Now().UTC()
	store.SetNow(func() time.Time { return now })

	// Both accounts use the same bundle id; only Alice has metadata.
	if _, err := store.InsertEvents(ctx, devA, []usage.Event{{
		ClientEventID: "ea", BundleID: "com.shared",
		StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1*time.Hour + 10*time.Minute),
	}}); err != nil {
		t.Fatalf("seed alice: %v", err)
	}
	if _, err := store.InsertEvents(ctx, devB, []usage.Event{{
		ClientEventID: "eb", BundleID: "com.shared",
		StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1*time.Hour + 20*time.Minute),
	}}); err != nil {
		t.Fatalf("seed bob: %v", err)
	}
	if _, err := store.UpsertAppMetadataBatch(ctx, a, map[string]string{"com.shared": "Alice's App"}); err != nil {
		t.Fatalf("upsert alice metadata: %v", err)
	}

	// Bob's summary must not see Alice's display name.
	rows, err := store.Summarise(ctx, usage.SummaryQuery{
		AccountID: b, From: now.Add(-24 * time.Hour), To: now.Add(time.Hour),
		GroupBy: []usage.SummaryGroup{usage.GroupByBundle},
	})
	if err != nil {
		t.Fatalf("Summarise bob: %v", err)
	}
	if len(rows) != 1 || rows[0].DisplayName != "" {
		t.Errorf("cross-account leak: bob got %+v", rows)
	}
}

func TestUsageStore_AppMetadata_EmptyMapIsNoOp(t *testing.T) {
	accountID, _, store, _ := withAccountAndDevice(t)
	ctx := context.Background()

	written, err := store.UpsertAppMetadataBatch(ctx, accountID, map[string]string{})
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if written != 0 {
		t.Errorf("written: got %d, want 0", written)
	}
}

func TestUsageStore_AppMetadata_SkipsEmptyValues(t *testing.T) {
	accountID, _, store, _ := withAccountAndDevice(t)
	ctx := context.Background()

	written, err := store.UpsertAppMetadataBatch(ctx, accountID, map[string]string{
		"":         "OK",      // empty bundle id — skipped
		"com.app":  "",        // empty display name — skipped
		"com.real": "Real App", // ok
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if written != 1 {
		t.Errorf("written: got %d, want 1 (only com.real should land)", written)
	}
}
