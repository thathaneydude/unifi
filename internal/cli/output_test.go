package cli_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
)

var _ = Describe("Output", func() {
	It("pretty-prints JSON by default", func() {
		var buf bytes.Buffer
		err := cli.WriteResult(&buf, cli.FormatJSON, []byte(`{"a":1,"b":2}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(ContainSubstring("\"a\": 1"))
		Expect(buf.String()).To(HaveSuffix("\n"))
	})

	It("passes bytes through in raw mode with a single trailing newline", func() {
		var buf bytes.Buffer
		err := cli.WriteResult(&buf, cli.FormatRaw, []byte(`{"a":1}`))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("{\"a\":1}\n"))
	})

	It("does not double the newline when the raw body already ends in one", func() {
		var buf bytes.Buffer
		err := cli.WriteResult(&buf, cli.FormatRaw, []byte("line\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("line\n"))
	})

	It("emits a single newline for an empty body in raw mode", func() {
		var buf bytes.Buffer
		err := cli.WriteResult(&buf, cli.FormatRaw, []byte{})
		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal("\n"))
	})
})
