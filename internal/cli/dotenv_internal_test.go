package cli

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	})
})
