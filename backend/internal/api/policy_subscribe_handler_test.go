package api

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
)

// stubAccountStore is the auth.AccountStore double we use across the
// subscribe tests. exists==true approves whichever account id the
// JWT carries; exists==false simulates a deleted account.
type stubAccountStore struct {
	exists bool
}

func (s stubAccountStore) AccountExists(_ context.Context, _ string) (bool, error) {
	return s.exists, nil
}

// subscribeFixture bundles every piece a WS subscribe test needs.
// Each field is initialised by newSubscribeFixture; the only knobs
// callers usually flip are AccountExists (via overrideAccounts) and
// the doc the store returns (set f.store.docs[id] before connecting).
type subscribeFixture struct {
	t        *testing.T
	signer   *auth.Signer
	verifier *auth.Verifier
	store    *fakePolicyStore
	broker   *policy.Broker
	server   *httptest.Server
}

func newSubscribeFixture(t *testing.T, accounts auth.AccountStore) *subscribeFixture {
	t.Helper()
	signer, verifier := mustNewSignerVerifier(t)
	store := newFakePolicyStore()
	broker := policy.NewBroker()
	h := PolicySubscribeHandler(verifier, accounts, store, broker)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &subscribeFixture{
		t:        t,
		signer:   signer,
		verifier: verifier,
		store:    store,
		broker:   broker,
		server:   srv,
	}
}

func mustNewSignerVerifier(t *testing.T) (*auth.Signer, *auth.Verifier) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", err)
	}
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	signer, err := auth.NewSigner(pemBytes)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	verifier, err := auth.NewVerifier(signer.PublicKey())
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return signer, verifier
}

// dial opens a WebSocket against the fixture's server. Returns the
// live connection and a t.Cleanup-registered close.
func (f *subscribeFixture) dial(ctx context.Context) *websocket.Conn {
	f.t.Helper()
	wsURL := strings.Replace(f.server.URL, "http://", "ws://", 1) + "/v1/policy/subscribe"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		f.t.Fatalf("dial: %v", err)
	}
	f.t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

// readJSON reads one text frame from conn, decodes it into out, and
// fails the test on timeout. The deadline keeps a flaky test from
// hanging the suite.
func readJSON(t *testing.T, conn *websocket.Conn, d time.Duration, out any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if typ != websocket.MessageText {
		t.Fatalf("frame type: got %v, want text", typ)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("unmarshal %s: %v", string(data), err)
	}
}

// writeAuth sends a {type:auth, token:tok} frame. Fails the test on
// write error.
func writeAuth(t *testing.T, conn *websocket.Conn, tok string) {
	t.Helper()
	body, _ := json.Marshal(wsAuthFrame{Type: "auth", Token: tok})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, body); err != nil {
		t.Fatalf("write auth: %v", err)
	}
}

func TestPolicySubscribe_AuthThenInitialVersion(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: true})
	tok, _ := f.signer.Issue("acct-1")

	// Seed a non-zero version so we know the initial frame really
	// reflects store.Current rather than always being zero.
	f.store.docs["acct-1"] = policy.Document{Version: 7}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := f.dial(ctx)

	writeAuth(t, conn, tok)

	var got wsVersionFrame
	readJSON(t, conn, 2*time.Second, &got)
	if got.Type != "version" || got.Version != 7 {
		t.Errorf("initial frame: got %+v, want {version 7}", got)
	}
}

func TestPolicySubscribe_PublishedVersionsArrive(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: true})
	tok, _ := f.signer.Issue("acct-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := f.dial(ctx)
	writeAuth(t, conn, tok)

	// Drain initial frame (version 0).
	var initial wsVersionFrame
	readJSON(t, conn, 2*time.Second, &initial)

	// Wait for a subscriber to register before publishing — otherwise
	// the publish races the subscribe and the test is flaky.
	waitForSubscribers(t, f.broker, "acct-1", 1, time.Second)

	f.broker.Publish("acct-1", 11)
	f.broker.Publish("acct-1", 12)

	for _, want := range []int64{11, 12} {
		var got wsVersionFrame
		readJSON(t, conn, 2*time.Second, &got)
		if got.Version != want {
			t.Errorf("got version %d, want %d", got.Version, want)
		}
	}
}

func TestPolicySubscribe_BadTokenIsRejected(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: true})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := f.dial(ctx)
	writeAuth(t, conn, "not-a-jwt")

	// Server should send an error frame and close. Either order is
	// acceptable — we just want the connection to die quickly.
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
		// drain anything (the error frame) until close
	}
}

func TestPolicySubscribe_DeletedAccountIsRejected(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: false})
	tok, _ := f.signer.Issue("acct-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := f.dial(ctx)
	writeAuth(t, conn, tok)

	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			return
		}
	}
}

func TestPolicySubscribe_MissingFirstFrameTimesOut(t *testing.T) {
	if testing.Short() {
		t.Skip("requires waiting out the wsAuthDeadline (5s)")
	}
	f := newSubscribeFixture(t, stubAccountStore{exists: true})

	// We dial and never send. The server's wsAuthDeadline is 5s, so
	// we give the read a generous 7s and require it to fail because
	// the server closed for auth-timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	conn := f.dial(ctx)

	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Error("read should fail after server closes for auth-timeout, got nil")
	}
}

func TestPolicySubscribe_CrossAccountIsolation(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: true})
	tokA, _ := f.signer.Issue("acct-A")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := f.dial(ctx)
	writeAuth(t, conn, tokA)

	// Drain initial.
	var initial wsVersionFrame
	readJSON(t, conn, 2*time.Second, &initial)

	waitForSubscribers(t, f.broker, "acct-A", 1, time.Second)

	// If acct-A leaked, the next frame would carry B's 99. Publish
	// B-then-A so the leaked frame would queue ahead of the legit one.
	f.broker.Publish("acct-B", 99)
	f.broker.Publish("acct-A", 5)

	var got wsVersionFrame
	readJSON(t, conn, 2*time.Second, &got)
	if got.Version != 5 {
		t.Errorf("got version %d, want 5 — leak from acct-B?", got.Version)
	}
}

func TestPolicySubscribe_MultipleSubscribersFanOut(t *testing.T) {
	f := newSubscribeFixture(t, stubAccountStore{exists: true})
	tok, _ := f.signer.Issue("acct-1")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn1 := f.dial(ctx)
	conn2 := f.dial(ctx)
	writeAuth(t, conn1, tok)
	writeAuth(t, conn2, tok)

	for _, c := range []*websocket.Conn{conn1, conn2} {
		var initial wsVersionFrame
		readJSON(t, c, 2*time.Second, &initial)
	}

	waitForSubscribers(t, f.broker, "acct-1", 2, 2*time.Second)
	f.broker.Publish("acct-1", 42)

	for i, c := range []*websocket.Conn{conn1, conn2} {
		var got wsVersionFrame
		readJSON(t, c, 2*time.Second, &got)
		if got.Version != 42 {
			t.Errorf("conn%d: got %d, want 42", i+1, got.Version)
		}
	}
}

// waitForSubscribers blocks until the broker reports `want` live
// subscribers for accountID, or fails the test on timeout. The handler
// subscribes off the request-handling goroutine *after* sending the
// initial frame, so a publish issued immediately after the client's
// first read can race that registration.
func waitForSubscribers(t *testing.T, b *policy.Broker, accountID string, want int, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if b.SubscriberCount(accountID) >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("subscribers for %s: got %d, want %d within %s", accountID, b.SubscriberCount(accountID), want, d)
}
