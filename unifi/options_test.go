package unifi_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Options", func() {
	It("applies an insecure TLS option to the resolved transport config", func() {
		c := unifi.Local("10.0.0.1", "key", unifi.WithInsecureSkipVerify())
		transport, ok := c.HTTPClient().Transport.(*http.Transport)
		Expect(ok).To(BeTrue(), "expected *http.Transport")
		Expect(transport.TLSClientConfig).NotTo(BeNil())
		Expect(transport.TLSClientConfig.InsecureSkipVerify).To(BeTrue())
	})

	It("does not mutate the caller's *tls.Config when combining WithTLSConfig and WithInsecureSkipVerify", func() {
		original := &tls.Config{MinVersion: tls.VersionTLS12} //nolint:gosec
		_ = unifi.Local("10.0.0.1", "key",
			unifi.WithTLSConfig(original),
			unifi.WithInsecureSkipVerify(),
		)
		// The caller's config must remain unmodified.
		Expect(original.InsecureSkipVerify).To(BeFalse())
	})

	It("uses a caller-supplied user agent", func() {
		c := unifi.Local("10.0.0.1", "key", unifi.WithUserAgent("test-agent/1.0"))
		Expect(c.UserAgent()).To(Equal("test-agent/1.0"))
	})

	It("uses a caller-supplied HTTP client", func() {
		custom := &http.Client{}
		c := unifi.Local("10.0.0.1", "key", unifi.WithHTTPClient(custom))
		Expect(c.HTTPClient()).To(BeIdenticalTo(custom))
	})

	Describe("TLS enforcement", func() {
		var srv *httptest.Server

		BeforeEach(func() {
			srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			DeferCleanup(srv.Close)
		})

		It("WithInsecureSkipVerify allows connections to a self-signed TLS server", func() {
			c := unifi.Local("unused", "key", unifi.WithInsecureSkipVerify())
			resp, err := c.HTTPClient().Get(srv.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("default client (no insecure) fails against a self-signed TLS server", func() {
			c := unifi.Local("unused", "key") // no insecure option
			_, err := c.HTTPClient().Get(srv.URL)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Or(
				ContainSubstring("certificate"),
				ContainSubstring("tls"),
				ContainSubstring("x509"),
				ContainSubstring("unknown authority"),
			))
		})

		It("WithTLSConfig propagates the supplied TLS config", func() {
			cfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec
			c := unifi.Local("unused", "key", unifi.WithTLSConfig(cfg))
			resp, err := c.HTTPClient().Get(srv.URL)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("WithRootCAs allows a client to reach the TLS server without WithInsecureSkipVerify", func() {
			// Build a cert pool from the test server's certificate.
			pool := x509.NewCertPool()
			pool.AddCert(srv.Certificate())

			c := unifi.Local("unused", "key", unifi.WithRootCAs(pool))
			resp, err := c.HTTPClient().Get(srv.URL)
			Expect(err).NotTo(HaveOccurred(), "expected WithRootCAs to trust the test server cert")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
