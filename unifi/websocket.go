package unifi

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/coder/websocket"
)

// Decode unmarshals a raw WebSocket frame into a typed value.
func Decode[T any](frame []byte) (T, error) {
	var v T
	err := json.Unmarshal(frame, &v)
	return v, err
}

// Stream is a live WebSocket subscription delivering raw frames.
type Stream struct {
	conn   *websocket.Conn
	frames chan []byte
	errc   chan error
	cancel context.CancelFunc
}

// Frames returns the channel of raw JSON frames. The consumer must read from
// this channel promptly; a slow consumer applies back-pressure to the
// underlying WebSocket and may stall the server.
func (s *Stream) Frames() <-chan []byte { return s.frames }

// Err returns the channel that receives exactly one terminal error after
// Frames() is closed. The error is non-nil on network failure or context
// cancellation, and is io.EOF (or a websocket close-error) on a normal server
// close. Consumers should drain Frames() first, then receive from Err() to
// obtain the terminal signal.
func (s *Stream) Err() <-chan error { return s.errc }

// Close terminates the subscription.
func (s *Stream) Close() error {
	s.cancel()
	return s.conn.Close(websocket.StatusNormalClosure, "")
}

// Subscribe opens a WebSocket subscription (e.g. "/v1/subscribe/events") on the
// given app, reusing the connection's auth header and TLS configuration.
func (c *Conn) Subscribe(ctx context.Context, app App, path string) (*Stream, error) {
	// prefix always returns an https:// URL, so this replacement always
	// yields a wss:// URL (plain http:// prefixes are left unchanged and
	// are used only in tests via the white-box prefix override).
	url := strings.Replace(c.prefix(app), "https://", "wss://", 1) + path
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		HTTPClient: c.httpClient,
		HTTPHeader: map[string][]string{
			"X-API-KEY":  {c.apiKey},
			"User-Agent": {c.userAgent},
		},
	})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Stream{
		conn:   conn,
		frames: make(chan []byte),
		errc:   make(chan error, 1),
		cancel: cancel,
	}
	go s.readLoop(ctx)
	return s, nil
}

func (s *Stream) readLoop(ctx context.Context) {
	// Always release the derived context. cancel is idempotent, so Close()
	// calling it again after readLoop exits is safe.
	defer s.cancel()
	defer close(s.frames)
	for {
		_, data, err := s.conn.Read(ctx)
		if err != nil {
			s.errc <- err
			return
		}
		select {
		case s.frames <- data:
		case <-ctx.Done():
			// Write the context error so consumers always get a terminal signal
			// after Frames() closes; the buffered channel (cap 1) absorbs it
			// without blocking.
			s.errc <- ctx.Err()
			return
		}
	}
}
