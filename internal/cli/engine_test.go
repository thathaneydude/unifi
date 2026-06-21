package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Engine", func() {
	It("builds an app command with one subcommand per operation", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")
		cmd := cli.NewAppCommand(unifi.AppNetwork, ops)

		Expect(cmd.Use).To(Equal("network"))
		names := map[string]bool{}
		for _, sub := range cmd.Commands() {
			names[sub.Name()] = true
		}
		Expect(names).To(HaveKey("ListWidgets"))
		Expect(names).To(HaveKey("GetWidget"))
		Expect(names).To(HaveKey("CreateWidget"))
	})

	It("declares flags for path, query, body, and safe-write gating", func() {
		ops := cli.OperationsFor(loadFixture(), unifi.AppNetwork, "v1.0.0")
		cmd := cli.NewAppCommand(unifi.AppNetwork, ops)

		for _, sub := range cmd.Commands() {
			switch sub.Name() {
			case "GetWidget":
				Expect(sub.Flags().Lookup("id")).NotTo(BeNil())
			case "ListWidgets":
				Expect(sub.Flags().Lookup("limit")).NotTo(BeNil())
			case "CreateWidget":
				Expect(sub.Flags().Lookup("body")).NotTo(BeNil())
				Expect(sub.Flags().Lookup("dry-run")).NotTo(BeNil())
				Expect(sub.Flags().Lookup("confirm")).NotTo(BeNil())
			}
		}
	})
})
