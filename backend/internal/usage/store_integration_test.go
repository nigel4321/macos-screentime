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
