//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

type protectEvent struct {
	Type string `json:"type"`
}

var _ = Describe("Protect subscribe e2e", func() {
	It("delivers a WebSocket frame through the subscription and validates X-API-KEY", func() {
		var receivedKey string
		serverDone := make(chan struct{})

		// Stand up a TLS test server that performs a WebSocket upgrade and
		// sends one JSON frame before closing normally.
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedKey = r.Header.Get("X-API-KEY")

			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				InsecureSkipVerify: true,
			})
			if err != nil {
				GinkgoWriter.Printf("websocket.Accept error: %v\n", err)
				close(serverDone)
				return
			}
			defer func() {
				conn.Close(websocket.StatusNormalClosure, "done")
				close(serverDone)
			}()

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			_ = wsjson.Write(ctx, conn, protectEvent{Type: "motion"})
			// Give the client time to read the frame before we close.
			time.Sleep(50 * time.Millisecond)
		}))
		DeferCleanup(srv.Close)

		// Extract host:port from the TLS server URL.
		// Subscribe converts https://HOST/proxy/protect/integration → wss://HOST/proxy/protect/integration
		// so this black-box approach works: set host = the test server's host:port.
		host := strings.TrimPrefix(srv.URL, "https://")

		// Build a Local Conn using the TLS test server's trust-enabled client.
		c := unifi.Local(host, "protect-e2e-key", unifi.WithHTTPClient(srv.Client()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		DeferCleanup(cancel)

		// Subscribe opens a wss:// connection to the test server.
		stream, err := c.Subscribe(ctx, unifi.AppProtect, "/v1/subscribe/events")
		Expect(err).NotTo(HaveOccurred())
		// stream.Close() may return an error if the server already closed the
		// connection; treat that as a benign cleanup condition.
		DeferCleanup(func() { _ = stream.Close() })

		// Receive the frame from the server.
		var frame []byte
		select {
		case f, ok := <-stream.Frames():
			if !ok {
				// Channel closed early; read terminal error.
				select {
				case e := <-stream.Err():
					Fail("stream closed early: " + e.Error())
				default:
					Fail("stream frames channel closed with no frame and no error")
				}
			}
			frame = f
		case e := <-stream.Err():
			Fail("stream error before frame: " + e.Error())
		case <-ctx.Done():
			Fail("timed out waiting for WebSocket frame")
		}

		// Wait for server goroutine to finish.
		Eventually(serverDone, 3*time.Second).Should(BeClosed())

		// Validate the API key was present on the upgrade request.
		Expect(receivedKey).To(Equal("protect-e2e-key"))

		// Decode the frame and assert payload.
		evt, err := unifi.Decode[protectEvent](frame)
		Expect(err).NotTo(HaveOccurred())
		Expect(evt.Type).To(Equal("motion"))
	})
})
