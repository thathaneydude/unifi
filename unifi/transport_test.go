package unifi_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Transport", func() {
	It("builds local base URLs per app", func() {
		c := unifi.Local("192.168.1.1", "k")
		Expect(c.NetworkBaseURL()).To(Equal("https://192.168.1.1/proxy/network/integration"))
		Expect(c.ProtectBaseURL()).To(Equal("https://192.168.1.1/proxy/protect/integration"))
	})

	It("builds remote base URLs per app", func() {
		c := unifi.Remote("console-123", "k")
		Expect(c.NetworkBaseURL()).To(Equal("https://api.ui.com/v1/connector/consoles/console-123/network/integration"))
	})

	It("BaseURL returns the correct URL for a given app", func() {
		c := unifi.Local("10.0.0.1", "k")
		Expect(c.BaseURL(unifi.AppNetwork)).To(Equal("https://10.0.0.1/proxy/network/integration"))
		Expect(c.BaseURL(unifi.AppProtect)).To(Equal("https://10.0.0.1/proxy/protect/integration"))
	})

	It("adds the X-API-KEY and User-Agent headers", func() {
		c := unifi.Local("h", "secret-key", unifi.WithUserAgent("ua/9"))
		req, _ := http.NewRequest(http.MethodGet, "https://h/x", nil)
		Expect(c.RequestEditor()(context.Background(), req)).To(Succeed())
		Expect(req.Header.Get("X-API-KEY")).To(Equal("secret-key"))
		Expect(req.Header.Get("User-Agent")).To(Equal("ua/9"))
	})

	It("RequestEditor default user agent is set", func() {
		c := unifi.Local("h", "mykey")
		req, _ := http.NewRequest(http.MethodGet, "https://h/x", nil)
		Expect(c.RequestEditor()(context.Background(), req)).To(Succeed())
		Expect(req.Header.Get("X-API-KEY")).To(Equal("mykey"))
		Expect(req.Header.Get("User-Agent")).NotTo(BeEmpty())
	})
})
