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

		conn, err := resolveFromFlags(&globalFlags{apiKey: "flag-key", host: "flag-host"}, unifi.AppNetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppNetwork)).To(ContainSubstring("flag-host"))
	})

	It("falls back to the environment when flags are unset", func() {
		GinkgoT().Setenv("UNIFI_API_KEY", "env-key")
		GinkgoT().Setenv("UNIFI_HOST", "")
		GinkgoT().Setenv("UNIFI_CONSOLE_ID", "env-console")

		conn, err := resolveFromFlags(&globalFlags{}, unifi.AppProtect)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn.BaseURL(unifi.AppProtect)).To(ContainSubstring("env-console"))
	})
})

var _ = Describe("Config.keyForApp", func() {
	It("prefers the app-specific key over the shared key", func() {
		cfg := Config{APIKey: "shared", NetworkAPIKey: "net", ProtectAPIKey: "protect"}
		Expect(cfg.keyForApp(unifi.AppNetwork)).To(Equal("net"))
		Expect(cfg.keyForApp(unifi.AppProtect)).To(Equal("protect"))
	})

	It("falls back to the shared key when the app-specific key is unset", func() {
		cfg := Config{APIKey: "shared", NetworkAPIKey: "net"}
		Expect(cfg.keyForApp(unifi.AppNetwork)).To(Equal("net"))
		Expect(cfg.keyForApp(unifi.AppProtect)).To(Equal("shared"))
	})

	It("returns empty when neither key is set", func() {
		Expect(Config{}.keyForApp(unifi.AppNetwork)).To(BeEmpty())
	})
})

var _ = Describe("resolveFromFlags app-specific keys", func() {
	It("merges the app-specific flag and selects it for that app", func() {
		GinkgoT().Setenv("UNIFI_API_KEY", "")
		GinkgoT().Setenv("UNIFI_HOST", "h")
		GinkgoT().Setenv("UNIFI_CONSOLE_ID", "")

		// Protect has its own key via flag; Network would have none and error.
		conn, err := resolveFromFlags(&globalFlags{protectAPIKey: "protect-flag"}, unifi.AppProtect)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())

		_, nerr := resolveFromFlags(&globalFlags{protectAPIKey: "protect-flag"}, unifi.AppNetwork)
		Expect(nerr).To(HaveOccurred())
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
