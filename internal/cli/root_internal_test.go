package cli

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("resolveFromFlags", func() {
	It("lets explicit flags override the environment", func() {
		GinkgoT().Setenv("UNIFI_API_KEY", "env-key")
		GinkgoT().Setenv("UNIFI_HOST", "env-host")
		GinkgoT().Setenv("UNIFI_CONSOLE_ID", "")

		conn, err := resolveFromFlags(&globalFlags{apiKey: "flag-key", host: "flag-host"})
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppNetwork)).To(ContainSubstring("flag-host"))
	})

	It("falls back to the environment when flags are unset", func() {
		GinkgoT().Setenv("UNIFI_API_KEY", "env-key")
		GinkgoT().Setenv("UNIFI_HOST", "")
		GinkgoT().Setenv("UNIFI_CONSOLE_ID", "env-console")

		conn, err := resolveFromFlags(&globalFlags{})
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppProtect)).To(ContainSubstring("env-console"))
	})
})

var _ = Describe("parseFormat", func() {
	It("accepts json, raw, human, and empty (default json)", func() {
		for in, want := range map[string]Format{
			"":      FormatJSON,
			"json":  FormatJSON,
			"raw":   FormatRaw,
			"human": FormatHuman,
		} {
			got, err := parseFormat(in)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(want))
		}
	})

	It("rejects an unknown format as a usage error", func() {
		_, err := parseFormat("yaml")
		Expect(err).To(HaveOccurred())
		var cerr *CLIError
		Expect(errors.As(err, &cerr)).To(BeTrue())
		Expect(cerr.ExitCode()).To(Equal(exitUsage))
	})
})
