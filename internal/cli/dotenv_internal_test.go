package cli

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("parseDotenv", func() {
	It("parses simple KEY=VALUE pairs", func() {
		out, err := parseDotenv(strings.NewReader("UNIFI_HOST=192.168.1.1\nUNIFI_API_KEY=abc"))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"UNIFI_HOST": "192.168.1.1", "UNIFI_API_KEY": "abc"}))
	})

	It("skips blank lines and # comments", func() {
		out, err := parseDotenv(strings.NewReader("\n# a comment\nUNIFI_HOST=h\n\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"UNIFI_HOST": "h"}))
	})

	It("strips an optional leading export", func() {
		out, err := parseDotenv(strings.NewReader("export UNIFI_API_KEY=k"))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"UNIFI_API_KEY": "k"}))
	})

	It("strips matching single and double quotes, preserving inner spaces", func() {
		out, err := parseDotenv(strings.NewReader("A=\" x \"\nB='y y'"))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"A": " x ", "B": "y y"}))
	})

	It("trims surrounding whitespace on unquoted values and around keys", func() {
		out, err := parseDotenv(strings.NewReader("  UNIFI_HOST =   192.168.1.1  "))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"UNIFI_HOST": "192.168.1.1"}))
	})

	It("splits on the first = so values may contain =", func() {
		out, err := parseDotenv(strings.NewReader("TOKEN=a=b=c"))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"TOKEN": "a=b=c"}))
	})

	It("allows empty values", func() {
		out, err := parseDotenv(strings.NewReader("EMPTY="))
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(map[string]string{"EMPTY": ""}))
	})

	It("errors with the line number on a line without =", func() {
		_, err := parseDotenv(strings.NewReader("UNIFI_HOST=h\nnonsense"))
		Expect(err).To(MatchError(ContainSubstring("line 2")))
	})

	It("errors on an invalid key", func() {
		_, err := parseDotenv(strings.NewReader("1BAD=x"))
		Expect(err).To(MatchError(ContainSubstring("invalid key")))
		Expect(err).To(MatchError(ContainSubstring("line 1")))
	})

	It("errors on an invalid key with export prefix", func() {
		_, err := parseDotenv(strings.NewReader("export 1BAD=x"))
		Expect(err).To(MatchError(ContainSubstring("invalid key")))
		Expect(err).To(MatchError(ContainSubstring("line 1")))
	})
})

var _ = Describe("loadDotenv", func() {
	write := func(dir, body string) string {
		p := filepath.Join(dir, ".env")
		Expect(os.WriteFile(p, []byte(body), 0o600)).To(Succeed())
		return p
	}

	It("is a no-op when the default file is absent and not required", func() {
		missing := filepath.Join(GinkgoT().TempDir(), "nope.env")
		Expect(loadDotenv(missing, false)).To(Succeed())
	})

	It("is a usage error when an explicit file is missing", func() {
		missing := filepath.Join(GinkgoT().TempDir(), "nope.env")
		err := loadDotenv(missing, true)
		var cerr *CLIError
		Expect(err).To(BeAssignableToTypeOf(cerr))
		Expect(err.(*CLIError).ExitCode()).To(Equal(1))
	})

	It("sets variables that are not already in the environment", func() {
		p := write(GinkgoT().TempDir(), "UNIFI_FROM_FILE=fromfile")
		DeferCleanup(func() { _ = os.Unsetenv("UNIFI_FROM_FILE") })
		Expect(os.Getenv("UNIFI_FROM_FILE")).To(BeEmpty())
		Expect(loadDotenv(p, true)).To(Succeed())
		Expect(os.Getenv("UNIFI_FROM_FILE")).To(Equal("fromfile"))
	})

	It("never overwrites a variable already set in the real environment", func() {
		GinkgoT().Setenv("UNIFI_API_KEY", "real")
		p := write(GinkgoT().TempDir(), "UNIFI_API_KEY=fromfile")
		Expect(loadDotenv(p, true)).To(Succeed())
		Expect(os.Getenv("UNIFI_API_KEY")).To(Equal("real"))
	})

	It("returns a usage error on a malformed file", func() {
		p := write(GinkgoT().TempDir(), "nonsense")
		err := loadDotenv(p, true)
		var cerr *CLIError
		Expect(err).To(BeAssignableToTypeOf(cerr))
		Expect(err.(*CLIError).ExitCode()).To(Equal(1))
	})
})

var _ = Describe("resolveFromFlags with env files", func() {
	It("loads connection config from an explicit --env-file", func() {
		dir := GinkgoT().TempDir()
		p := filepath.Join(dir, "creds.env")
		Expect(os.WriteFile(p, []byte("UNIFI_API_KEY=k\nUNIFI_HOST=192.168.1.1\n"), 0o600)).To(Succeed())
		DeferCleanup(func() {
			_ = os.Unsetenv("UNIFI_API_KEY")
			_ = os.Unsetenv("UNIFI_HOST")
		})

		conn, err := resolveFromFlags(&globalFlags{envFile: p}, unifi.AppNetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
	})

	It("auto-loads ./.env from the working directory", func() {
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, ".env"), []byte("UNIFI_API_KEY=k\nUNIFI_CONSOLE_ID=abc\n"), 0o600)).To(Succeed())
		GinkgoT().Chdir(dir)
		DeferCleanup(func() {
			_ = os.Unsetenv("UNIFI_API_KEY")
			_ = os.Unsetenv("UNIFI_CONSOLE_ID")
		})

		conn, err := resolveFromFlags(&globalFlags{}, unifi.AppNetwork)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
	})

	It("errors when an explicit --env-file is missing", func() {
		_, err := resolveFromFlags(&globalFlags{envFile: filepath.Join(GinkgoT().TempDir(), "nope.env")}, unifi.AppNetwork)
		var cerr *CLIError
		Expect(err).To(BeAssignableToTypeOf(cerr))
		Expect(err.(*CLIError).ExitCode()).To(Equal(1))
	})
})
