package cli_test

import (
	"bytes"
	"encoding/json"

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
})
