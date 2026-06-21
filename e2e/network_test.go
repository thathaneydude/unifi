//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Network e2e", func() {
	It("sends X-API-KEY header and receives HTTP 200 from a TLS test server", func() {
		var (
			receivedKey    string
			receivedPath   string
			serverReceived bool
		)

		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serverReceived = true
			receivedKey = r.Header.Get("X-API-KEY")
			receivedPath = r.URL.Path

			body, _ := json.Marshal(struct {
				ApplicationVersion string `json:"applicationVersion"`
			}{"10.3.58"})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
		DeferCleanup(srv.Close)

		// Extract host:port from the test server URL (strip scheme).
		host := strings.TrimPrefix(srv.URL, "https://")

		// Build a Local Conn using the TLS test server's client (trusts test CA).
		c := unifi.Local(host, "e2e-key", unifi.WithHTTPClient(srv.Client()))

		// Issue a request to /v1/info via c.RequestEditor() + c.HTTPClient().
		targetURL := srv.URL + "/v1/info"
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, targetURL, nil)
		Expect(err).NotTo(HaveOccurred())

		// Apply the SDK request editor (injects X-API-KEY and User-Agent).
		Expect(c.RequestEditor()(context.Background(), req)).To(Succeed())

		resp, err := c.HTTPClient().Do(req)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(resp.Body.Close)

		Expect(serverReceived).To(BeTrue())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
		Expect(receivedKey).To(Equal("e2e-key"))
		Expect(receivedPath).To(Equal("/v1/info"))
	})
})
