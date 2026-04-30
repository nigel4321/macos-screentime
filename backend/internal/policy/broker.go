package policy

import "sync"

// Publisher is the producer half of the in-process pub/sub registry
// used by the WebSocket subscribe endpoint. The PUT handler calls
// Publish after a successful write; subscribers receive the new
// version on their channel.
//
// A nil Publisher is safe — the policy handlers default to a no-op
// when no broker has been wired in (e.g. unit tests, dev without WS).
type Publisher interface {
	Publish(accountID string, version int64)
}

// NopPublisher is a Publisher that drops every event. Useful as a
// default when no broker is wired up.
type NopPublisher struct{}

// Publish is a no-op.
func (NopPublisher) Publish(string, int64) {}

// Broker is an in-process pub/sub registry keyed by account id. It
// fans new policy versions out to every subscriber currently watching
// that account. It does not durably store anything — a subscriber that
// joins after a Publish does not see history. The intended client
// behaviour is "on connect, GET /v1/policy/current; thereafter, treat
// each broker message as a poke to GET again," which keeps the broker
// out of the consistency-critical path.
type Broker struct {
	mu   sync.RWMutex
	subs map[string]map[*subscriber]struct{}
}

// NewBroker returns an empty Broker.
func NewBroker() *Broker {
	return &Broker{subs: map[string]map[*subscriber]struct{}{}}
}

// subscriber holds the per-connection delivery channel. The buffer
// absorbs short bursts; once full, Publish drops events for that
// subscriber rather than blocking the publishing goroutine. Slow or
// dead clients therefore can't backpressure the PUT path.
type subscriber struct {
	ch chan int64
}

// subscriberBufferSize is the per-subscriber channel depth. Sized for
// short bursts: in steady state a connection sees ≪1 event/sec, so 4
// is enough to absorb a thundering-herd of writes without the buffer
// becoming the surface where slow consumers manifest.
const subscriberBufferSize = 4

// Subscribe registers a new listener for accountID. The returned
// channel receives every version published for that account from
// after this call, up to the buffer depth. Callers MUST invoke the
// cleanup func when done — failing to do so leaks the entry from the
// registry until the process exits.
//
// The channel is never closed by the broker; readers should range
// over it inside a select that also watches their own done channel
// (typically the WebSocket request context).
func (b *Broker) Subscribe(accountID string) (<-chan int64, func()) {
	s := &subscriber{ch: make(chan int64, subscriberBufferSize)}

	b.mu.Lock()
	if b.subs[accountID] == nil {
		b.subs[accountID] = map[*subscriber]struct{}{}
	}
	b.subs[accountID][s] = struct{}{}
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if set, ok := b.subs[accountID]; ok {
			delete(set, s)
			if len(set) == 0 {
				delete(b.subs, accountID)
			}
		}
		// The channel is intentionally not closed: Publish takes
		// RLock before sending, so it can't race with the deletion
		// above; once the entry is removed, no further sends are
		// possible. Leaving the channel unclosed keeps any in-flight
		// receive in the WS goroutine race-free during shutdown.
	}
	return s.ch, cleanup
}

// Publish fans version out to every subscriber currently watching
// accountID. Sends are non-blocking — a full subscriber channel drops
// the event on the floor rather than blocking the caller, which keeps
// the PUT handler's latency independent of WS subscriber health.
//
// The expected client recovery is to reconnect (or re-fetch via GET)
// when it suspects gaps; the broker itself does not retransmit.
func (b *Broker) Publish(accountID string, version int64) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for s := range b.subs[accountID] {
		select {
		case s.ch <- version:
		default:
		}
	}
}

// SubscriberCount returns the number of live subscribers for
// accountID. Test-only helper — production code never branches on it.
func (b *Broker) SubscriberCount(accountID string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[accountID])
}
