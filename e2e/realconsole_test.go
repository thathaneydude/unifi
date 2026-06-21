//go:build e2e

package e2e

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

// These specs exercise a REAL UniFi console when credentials are provided via
// environment variables. In CI those variables are wired from secrets that are
// scoped to the e2e job's run step only (see .github/workflows/ci.yml). They
// Skip cleanly when the relevant variables are unset, so the default CI run and
// local runs without hardware stay green.
var _ = Describe("Real console e2e", func() {
	It("reaches a local console's Network API when UNIFI_HOST/UNIFI_API_KEY are set", func() {
		host := os.Getenv("UNIFI_HOST")
		key := os.Getenv("UNIFI_API_KEY")
		if host == "" || key == "" {
			Skip("UNIFI_HOST/UNIFI_API_KEY not set; skipping local real-console e2e")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Local consoles typically present a self-signed certificate.
		c := unifi.Local(host, key, unifi.WithInsecureSkipVerify())
		net, err := c.Network()
		Expect(err).NotTo(HaveOccurred())

		resp, err := net.GetInfoWithResponse(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode()).To(Equal(200))
	})

	It("reaches a console's Network API remotely when UNIFI_CONSOLE_ID/UNIFI_SM_KEY are set", func() {
		consoleID := os.Getenv("UNIFI_CONSOLE_ID")
		key := os.Getenv("UNIFI_SM_KEY")
		if consoleID == "" || key == "" {
			Skip("UNIFI_CONSOLE_ID/UNIFI_SM_KEY not set; skipping remote real-console e2e")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// The remote (cloud connector) path reaches api.ui.com and is therefore
		// usable from CI runners when these secrets are configured.
		c := unifi.Remote(consoleID, key)
		net, err := c.Network()
		Expect(err).NotTo(HaveOccurred())

		resp, err := net.GetInfoWithResponse(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode()).To(Equal(200))
	})
})
