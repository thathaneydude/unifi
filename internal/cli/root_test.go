package cli_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
)

var _ = Describe("Root", func() {
	It("builds a root command exposing apps and discovery", func() {
		root, err := cli.NewRootCommand()
		Expect(err).NotTo(HaveOccurred())

		names := map[string]bool{}
		for _, sub := range root.Commands() {
			names[sub.Name()] = true
		}
		Expect(names).To(HaveKey("network"))
		Expect(names).To(HaveKey("protect"))
		Expect(names).To(HaveKey("schema"))
	})

	It("returns a usage error exit code for an unknown command", func() {
		root, err := cli.NewRootCommand()
		Expect(err).NotTo(HaveOccurred())
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"nonexistent-app"})

		code := cli.RunRoot(root)
		Expect(code).To(Equal(1))
	})

	It("returns a usage error exit code when no command is given", func() {
		root, err := cli.NewRootCommand()
		Expect(err).NotTo(HaveOccurred())
		root.SetArgs([]string{})
		Expect(cli.RunRoot(root)).To(Equal(1))
	})
})
