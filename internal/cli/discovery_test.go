package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Discovery", func() {
	It("lists operations as JSON", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")
		var buf bytes.Buffer
		Expect(cli.WriteOperationIndex(&buf, ops)).To(Succeed())

		var got []map[string]any
		Expect(json.Unmarshal(buf.Bytes(), &got)).To(Succeed())
		Expect(got).To(HaveLen(3))
		Expect(got[0]).To(HaveKey("operationId"))
		Expect(got[0]).To(HaveKey("method"))
		Expect(got[0]).To(HaveKey("path"))
	})

	It("lists operation ids only", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")
		var buf bytes.Buffer
		Expect(cli.WriteOperationIDs(&buf, ops)).To(Succeed())

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		Expect(lines).To(HaveLen(3))
		Expect(buf.String()).NotTo(ContainSubstring("{"))
		Expect(lines[0]).To(Equal(ops[0].ID))
	})

	It("lists operations as an aligned human table", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")
		var buf bytes.Buffer
		Expect(cli.WriteOperationIndexHuman(&buf, ops)).To(Succeed())

		out := buf.String()
		Expect(out).To(ContainSubstring(ops[0].ID))
		Expect(out).To(ContainSubstring(ops[0].Method))
		Expect(out).NotTo(ContainSubstring("\"operationId\""))
		Expect(strings.Split(strings.TrimRight(out, "\n"), "\n")).To(HaveLen(3))
	})
})
