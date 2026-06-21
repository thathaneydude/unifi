package unifi

// websocket_internal_test.go — white-box WebSocket integration test.
// Package: unifi (not unifi_test) so it can access and override the
// unexported prefix field to point Subscribe at an httptest WS server.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// TestSubscribe_DialFailure asserts that Subscribe returns a non-nil error and
// a nil *Stream when the WebSocket upgrade fails (server returns 401).
func TestSubscribe_DialFailure(t *testing.T) {
	// Server that rejects WS upgrades with a 401.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	// Use the white-box prefix override: return http:// so the replacement is
	// a no-op and Subscribe dials ws:// against the plain httptest server.
	c := newConn("bad-key", func(app App) string {
		return srv.URL
	}, WithHTTPClient(srv.Client()))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := c.Subscribe(ctx, AppProtect, "/v1/subscribe/events")
	if err == nil {
		t.Fatal("expected Subscribe to return an error on 401, got nil")
	}
	if stream != nil {
		_ = stream.Close()
		t.Fatalf("expected Subscribe to return nil *Stream on failure, got non-nil")
	}
}

type wsEvent struct {
	Type string `json:"type"`
}

// TestSubscribe_HeaderAndFrameDelivery runs a real WebSocket round-trip:
//  1. Starts an httptest server that accepts a WS upgrade.
//  2. Checks the upgrade request carries X-API-KEY.
//  3. Writes one JSON frame from the server.
//  4. Asserts Conn.Subscribe delivers it on Frames() and Decode parses it.
func TestSubscribe_HeaderAndFrameDelivery(t *testing.T) {
	var receivedKey string
	serverDone := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-KEY")

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true,
		})
		if err != nil {
			t.Logf("websocket.Accept error: %v", err)
			close(serverDone)
			return
		}
		defer func() {
			_ = conn.Close(websocket.StatusNormalClosure, "")
			close(serverDone)
		}()

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		_ = wsjson.Write(ctx, conn, wsEvent{Type: "motion"})
		// Wait briefly so the client can read the frame before we close.
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	// Build a Conn with the prefix overridden to return an http:// URL so that
	// strings.Replace(..., "https://", "wss://", 1) leaves it as http://, which
	// coder/websocket dials as ws:// (plain WebSocket, no TLS).
	c := newConn("ws-test-key", func(app App) string {
		// Return http:// — Subscribe's strings.Replace is a no-op, yielding ws://.
		return srv.URL
	}, WithHTTPClient(srv.Client()), WithUserAgent("ws-test-agent/1"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := c.Subscribe(ctx, AppProtect, "/v1/subscribe/events")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Read one frame or time out.
	var frame []byte
	select {
	case f, ok := <-stream.Frames():
		if !ok {
			// Channel closed; check Err.
			select {
			case e := <-stream.Err():
				t.Fatalf("stream error: %v", e)
			default:
				t.Fatal("frames channel closed with no error and no frame")
			}
		}
		frame = f
	case e := <-stream.Err():
		t.Fatalf("stream error before frame: %v", e)
	case <-ctx.Done():
		t.Fatal("timed out waiting for frame")
	}

	_ = stream.Close()

	// Wait for server goroutine to finish.
	select {
	case <-serverDone:
	case <-time.After(3 * time.Second):
		t.Log("server goroutine did not finish in time (non-fatal)")
	}

	// Assert the upgrade request carried the correct API key.
	if receivedKey != "ws-test-key" {
		t.Errorf("X-API-KEY: got %q, want %q", receivedKey, "ws-test-key")
	}

	// Assert the frame can be decoded via Decode[T].
	evt, err := Decode[wsEvent](frame)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if evt.Type != "motion" {
		t.Errorf("event type: got %q, want %q", evt.Type, "motion")
	}

	// Sanity-check the raw frame is valid JSON.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(frame, &raw); err != nil {
		t.Errorf("raw frame is not valid JSON: %v", err)
	}
}
