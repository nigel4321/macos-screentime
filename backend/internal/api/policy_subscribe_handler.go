package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/nigel4321/macos-screentime/backend/internal/auth"
	"github.com/nigel4321/macos-screentime/backend/internal/policy"
)

// WebSocket subscribe protocol parameters.
//
//   - wsAuthDeadline: how long the client has to send the auth frame
//     after the upgrade. Short by design — there's nothing useful a
//     connection can do until it has authenticated.
//   - wsIdleTimeout: max time we'll wait for any inbound traffic
//     (auth, app message, or pong frame) before declaring the client
//     dead. Comfortably longer than wsPingInterval so a healthy
//     ping/pong cycle never trips it.
//   - wsPingInterval: how often the server sends a control-frame
//     ping. Pongs reset the read deadline at the library level.
//   - wsWriteTimeout: per-write deadline. Applies equally to JSON
//     frames and to the Ping call's wait-for-pong.
const (
	wsAuthDeadline = 5 * time.Second
	wsIdleTimeout  = 90 * time.Second
	wsPingInterval = 30 * time.Second
	wsWriteTimeout = 10 * time.Second
)

// wsAuthFrame is the first message a client must send. We accept the
// JWT in the message body rather than the HTTP `Authorization` header
// so the wire protocol is the same for native clients and any future
// browser client (where headers can't be set on a WS upgrade).
type wsAuthFrame struct {
	Type  string `json:"type"` // must be "auth"
	Token string `json:"token"`
}

// wsVersionFrame is the only payload the server ever sends in normal
// operation. It is intentionally minimal — clients re-fetch
// `/v1/policy/current` to obtain the actual document. Keeping the
// broker out of the document path means we never have to reason about
// document staleness on the wire.
type wsVersionFrame struct {
	Type    string `json:"type"` // always "version"
	Version int64  `json:"version"`
}

// wsErrorFrame is sent best-effort before a close in failure paths.
// Clients use it for log messages — they shouldn't branch on `code`.
type wsErrorFrame struct {
	Type string `json:"type"` // always "error"
	Code string `json:"code"`
}

// PolicySubscribeHandler returns a WebSocket handler that streams
// policy-version notifications to the authenticated account.
//
// # Wire protocol
//
//  1. Client connects to /v1/policy/subscribe (no auth header).
//  2. Within 5 seconds, client sends one JSON frame:
//     {"type":"auth","token":"<JWT>"}
//     A missing, late, or invalid auth frame closes the connection
//     with code 1008 (policy violation).
//  3. Server replies with the current version as
//     {"type":"version","version":N}
//     and continues to send a fresh frame every time the account's
//     policy is PUT, in commit order.
//  4. Server pings every 30s; if the client doesn't pong, the
//     connection is closed.
//  5. If no inbound traffic arrives for 90s the server closes too.
//
// # Reconnection semantics
//
// The broker does NOT replay history. On reconnect, clients receive
// exactly one initial version frame (the current latest) and then the
// live stream resumes. Clients that suspect they missed updates
// (e.g. after a long network partition) should hit GET /v1/policy/current
// and reconcile against the body's version field. The version-only
// frame is deliberately a poke, not a delivery — the GET is the
// authoritative read path.
//
// `accounts` is the same surface used by the HTTP authenticator:
// confirms the account row still exists before honouring the JWT.
func PolicySubscribeHandler(
	verifier *auth.Verifier,
	accounts auth.AccountStore,
	store PolicyStore,
	broker *policy.Broker,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Native clients (Mac/Android) connect directly to our host;
		// there is no browser session-cookie context to protect via
		// origin checks. Skip the library's same-origin check rather
		// than maintaining a host allowlist that would have to track
		// every parent's network.
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			slog.ErrorContext(r.Context(), "policy ws accept", "err", err)
			return
		}
		defer conn.CloseNow()

		ctx := r.Context()

		accountID, err := readAuthFrame(ctx, conn, verifier, accounts)
		if err != nil {
			// Best-effort: tell the client why before tearing down.
			// Errors here are ignored — the close below is what
			// matters.
			_ = writeJSONFrame(ctx, conn, wsErrorFrame{Type: "error", Code: "unauthorized"})
			_ = conn.Close(websocket.StatusPolicyViolation, "auth failed")
			return
		}

		cur, err := store.Current(ctx, accountID)
		if err != nil {
			slog.ErrorContext(ctx, "policy ws current", "err", err)
			_ = conn.Close(websocket.StatusInternalError, "current failed")
			return
		}
		if err := writeJSONFrame(ctx, conn, wsVersionFrame{Type: "version", Version: cur.Version}); err != nil {
			return
		}

		ch, cleanup := broker.Subscribe(accountID)
		defer cleanup()

		runSubscribe(ctx, conn, ch)
	})
}

// readAuthFrame waits up to wsAuthDeadline for a single text frame,
// validates the embedded JWT, and returns the resolved account id.
func readAuthFrame(
	ctx context.Context,
	conn *websocket.Conn,
	verifier *auth.Verifier,
	accounts auth.AccountStore,
) (string, error) {
	authCtx, cancel := context.WithTimeout(ctx, wsAuthDeadline)
	defer cancel()

	typ, data, err := conn.Read(authCtx)
	if err != nil {
		return "", err
	}
	if typ != websocket.MessageText {
		return "", errors.New("auth: not a text frame")
	}
	var frame wsAuthFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return "", err
	}
	if frame.Type != "auth" || frame.Token == "" {
		return "", errors.New("auth: missing type/token")
	}
	claims, err := verifier.Parse(frame.Token)
	if err != nil {
		return "", err
	}
	exists, err := accounts.AccountExists(authCtx, claims.AccountID)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New("auth: account does not exist")
	}
	return claims.AccountID, nil
}

// writeJSONFrame encodes v as JSON and writes it as a single text
// frame, with a write-side deadline so a stuck client can't pin a
// goroutine indefinitely.
func writeJSONFrame(ctx context.Context, conn *websocket.Conn, v any) error {
	wctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(wctx, websocket.MessageText, body)
}

// runSubscribe is the post-auth steady-state loop. It runs three
// concurrent activities by spawning a reader goroutine and selecting
// on the broker channel + a ping ticker.
//
// We don't act on incoming messages: the protocol is server-push only.
// The reader exists to detect disconnects (via Read returning an error)
// and to enforce wsIdleTimeout — clients that go silent get evicted.
func runSubscribe(ctx context.Context, conn *websocket.Conn, ch <-chan int64) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			rctx, rcancel := context.WithTimeout(ctx, wsIdleTimeout)
			_, _, err := conn.Read(rctx)
			rcancel()
			if err != nil {
				return
			}
		}
	}()

	pingTick := time.NewTicker(wsPingInterval)
	defer pingTick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-readDone:
			return
		case v := <-ch:
			if err := writeJSONFrame(ctx, conn, wsVersionFrame{Type: "version", Version: v}); err != nil {
				return
			}
		case <-pingTick.C:
			pctx, pcancel := context.WithTimeout(ctx, wsWriteTimeout)
			err := conn.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		}
	}
}
