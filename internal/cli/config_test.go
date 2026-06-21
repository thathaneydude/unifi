package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Config", func() {
	It("builds a local connection from host + key", func() {
		conn, err := cli.ResolveConn(cli.Config{APIKey: "k", Host: "192.168.1.1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppNetwork)).To(Equal("https://192.168.1.1/proxy/network/integration"))
	})

	It("builds a remote connection from console id + key", func() {
		conn, err := cli.ResolveConn(cli.Config{APIKey: "k", ConsoleID: "abc"})
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppProtect)).To(ContainSubstring("api.ui.com/v1/connector/consoles/abc/protect"))
	})

	It("requires an API key", func() {
		_, err := cli.ResolveConn(cli.Config{Host: "192.168.1.1"})
		var cerr *cli.CLIError
		Expect(err).To(BeAssignableToTypeOf(cerr))
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})

	It("rejects ambiguous host + console id", func() {
		_, err := cli.ResolveConn(cli.Config{APIKey: "k", Host: "h", ConsoleID: "c"})
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})

	It("requires exactly one transport", func() {
		_, err := cli.ResolveConn(cli.Config{APIKey: "k"})
		Expect(err.(*cli.CLIError).ExitCode()).To(Equal(2))
	})
})
