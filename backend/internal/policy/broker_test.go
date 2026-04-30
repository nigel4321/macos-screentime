package policy

import (
	"sync"
	"testing"
	"time"
)

// recvWithin reads one value from ch with a deadline. Returns
// (v, true) on success, (0, false) on timeout. Used to make tests
// fail fast instead of hanging.
func recvWithin(t *testing.T, ch <-chan int64, d time.Duration) (int64, bool) {
	t.Helper()
	select {
	case v := <-ch:
		return v, true
	case <-time.After(d):
		return 0, false
	}
}

func TestBroker_SubscribeReceivesPublish(t *testing.T) {
	b := NewBroker()
	ch, cleanup := b.Subscribe("acct-1")
	defer cleanup()

	b.Publish("acct-1", 42)

	got, ok := recvWithin(t, ch, time.Second)
	if !ok {
		t.Fatal("did not receive within 1s")
	}
	if got != 42 {
		t.Errorf("got %d, want 42", got)
	}
}

func TestBroker_LateSubscriberMissesHistory(t *testing.T) {
	// Broker does not retransmit; a subscriber that joins after a
	// Publish should not see the prior event.
	b := NewBroker()
	b.Publish("acct-1", 7) // before anyone is listening

	ch, cleanup := b.Subscribe("acct-1")
	defer cleanup()

	if _, ok := recvWithin(t, ch, 50*time.Millisecond); ok {
		t.Error("late subscriber received historical event")
	}
}

func TestBroker_FanOutToMultipleSubscribers(t *testing.T) {
	b := NewBroker()
	ch1, c1 := b.Subscribe("acct-1")
	ch2, c2 := b.Subscribe("acct-1")
	defer c1()
	defer c2()

	b.Publish("acct-1", 1)

	for i, ch := range []<-chan int64{ch1, ch2} {
		got, ok := recvWithin(t, ch, time.Second)
		if !ok || got != 1 {
			t.Errorf("subscriber %d: got %d ok=%v, want 1", i, got, ok)
		}
	}
}

func TestBroker_CrossAccountIsolation(t *testing.T) {
	b := NewBroker()
	chA, cleanupA := b.Subscribe("acct-A")
	chB, cleanupB := b.Subscribe("acct-B")
	defer cleanupA()
	defer cleanupB()

	b.Publish("acct-A", 1)

	got, ok := recvWithin(t, chA, time.Second)
	if !ok || got != 1 {
		t.Errorf("acct-A: got %d ok=%v, want 1", got, ok)
	}
	if _, ok := recvWithin(t, chB, 50*time.Millisecond); ok {
		t.Errorf("acct-B should not have received acct-A's event")
	}
}

func TestBroker_SlowConsumerDoesNotBlock(t *testing.T) {
	// Send more events than the buffer; Publish must never block,
	// and no panic; excess events are dropped on the floor.
	b := NewBroker()
	ch, cleanup := b.Subscribe("acct-1")
	defer cleanup()

	const n = subscriberBufferSize * 4
	done := make(chan struct{})
	go func() {
		for i := 0; i < n; i++ {
			b.Publish("acct-1", int64(i))
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked — subscriber buffer is supposed to drop")
	}

	// We expect at least one event landed; not asserting an exact
	// count because timing-dependent.
	if _, ok := recvWithin(t, ch, time.Second); !ok {
		t.Error("expected at least one event to land, got none")
	}
}

func TestBroker_CleanupRemovesSubscriber(t *testing.T) {
	b := NewBroker()
	_, cleanup := b.Subscribe("acct-1")
	if got := b.SubscriberCount("acct-1"); got != 1 {
		t.Fatalf("count after subscribe: got %d, want 1", got)
	}
	cleanup()
	if got := b.SubscriberCount("acct-1"); got != 0 {
		t.Errorf("count after cleanup: got %d, want 0", got)
	}
}

func TestBroker_CleanupIsIdempotent(t *testing.T) {
	b := NewBroker()
	_, cleanup := b.Subscribe("acct-1")
	cleanup()
	cleanup() // must not panic
}

func TestBroker_PublishToNoSubscribersIsHarmless(t *testing.T) {
	b := NewBroker()
	b.Publish("acct-nobody", 1) // must not panic
}

func TestBroker_ConcurrentSubscribeAndPublish(t *testing.T) {
	// Stress: race detector must not flag concurrent Subscribe /
	// Publish / cleanup operations.
	b := NewBroker()
	var wg sync.WaitGroup
	const fanout = 16
	for i := 0; i < fanout; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, cleanup := b.Subscribe("acct-shared")
			defer cleanup()
			for j := 0; j < 32; j++ {
				select {
				case <-ch:
				default:
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 256; j++ {
			b.Publish("acct-shared", int64(j))
		}
	}()
	wg.Wait()
}

func TestNopPublisher_Publish(t *testing.T) {
	var p Publisher = NopPublisher{}
	p.Publish("acct-1", 1) // must not panic
}
