package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Config", func() {
	It("builds a local connection from host + key", func() {
		conn, err := cli.ResolveConn(cli.Config{APIKey: "k", Host: "192.168.1.1"}, unifi.AppNetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppNetwork)).To(Equal("https://192.168.1.1/proxy/network/integration"))
	})

	It("builds a remote connection from console id + key", func() {
		conn, err := cli.ResolveConn(cli.Config{APIKey: "k", ConsoleID: "abc"}, unifi.AppProtect)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppProtect)).To(ContainSubstring("api.ui.com/v1/connector/consoles/abc/protect"))
	})

	It("requires an API key", func() {
		_, err := cli.ResolveConn(cli.Config{Host: "192.168.1.1"}, unifi.AppNetwork)
		var cerr *cli.CLIError
		Expect(err).To(BeAssignableToTypeOf(cerr))
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})

	It("names the app-specific and shared options when no key resolves", func() {
		_, err := cli.ResolveConn(cli.Config{Host: "192.168.1.1"}, unifi.AppProtect)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("--protect-api-key"))
		Expect(err.Error()).To(ContainSubstring("UNIFI_PROTECT_API_KEY"))
		Expect(err.Error()).To(ContainSubstring("--api-key"))
	})

	It("rejects ambiguous host + console id", func() {
		_, err := cli.ResolveConn(cli.Config{APIKey: "k", Host: "h", ConsoleID: "c"}, unifi.AppNetwork)
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})

	It("requires exactly one transport", func() {
		_, err := cli.ResolveConn(cli.Config{APIKey: "k"}, unifi.AppNetwork)
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})

	It("prefers the app-specific key over the shared key", func() {
		// The shared key alone reaches the Network app...
		netConn, err := cli.ResolveConn(cli.Config{
			APIKey:        "shared",
			ProtectAPIKey: "protect-only",
			Host:          "192.168.1.1",
		}, unifi.AppNetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(netConn).NotTo(BeNil())

		// ...while Protect uses its own key. Both apps resolve from one config.
		protectConn, err := cli.ResolveConn(cli.Config{
			APIKey:        "shared",
			ProtectAPIKey: "protect-only",
			Host:          "192.168.1.1",
		}, unifi.AppProtect)
		Expect(err).NotTo(HaveOccurred())
		Expect(protectConn).NotTo(BeNil())
	})

	It("uses the shared key for an app without a specific key", func() {
		conn, err := cli.ResolveConn(cli.Config{
			APIKey:        "shared",
			NetworkAPIKey: "network-only",
			Host:          "192.168.1.1",
		}, unifi.AppProtect)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
	})
})
