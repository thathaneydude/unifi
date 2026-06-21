package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
)

var _ = Describe("CLIError", func() {
	It("maps each category to its stable exit code", func() {
		Expect(cli.NewUsageError("bad flag").ExitCode()).To(Equal(1))
		Expect(cli.NewAuthError("no key").ExitCode()).To(Equal(2))
		Expect(cli.NewAPIError("GetWidget", 404, []byte(`{"x":1}`)).ExitCode()).To(Equal(3))
		Expect(cli.NewTransportError("GetWidget", "dial").ExitCode()).To(Equal(4))
	})

	It("renders a JSON envelope without secrets", func() {
		out := cli.NewAPIError("GetWidget", 404, []byte(`{"code":"not_found"}`)).JSON()
		Expect(out).To(ContainSubstring(`"operation":"GetWidget"`))
		Expect(out).To(ContainSubstring(`"status":404`))
		Expect(out).To(ContainSubstring(`not_found`))
		Expect(out).NotTo(ContainSubstring("X-API-KEY"))
	})

	It("surfaces the UniFi error envelope message", func() {
		out := cli.NewAPIError("GetWidget", 400, []byte(`{"statusCode":400,"statusName":"BadRequest","message":"id is required"}`)).JSON()
		Expect(out).To(ContainSubstring("id is required"))
	})

	It("surfaces a non-JSON error body instead of discarding it", func() {
		err := cli.NewAPIError("GetWidget", 502, []byte("502 Bad Gateway"))
		Expect(err.Error()).To(Equal("502 Bad Gateway"))
		Expect(err.JSON()).To(ContainSubstring("502 Bad Gateway"))
	})
})
