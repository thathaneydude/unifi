package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thathaneydude/unifi/internal/cli"
	"github.com/thathaneydude/unifi/unifi"
)

var _ = Describe("Catalog", func() {
	It("loads both apps with their default versions", func() {
		cat, err := cli.LoadCatalog()
		Expect(err).NotTo(HaveOccurred())

		Expect(cat.Apps()).To(ConsistOf(unifi.AppNetwork, unifi.AppProtect))
		Expect(cat.DefaultVersion(unifi.AppProtect)).To(Equal("v7.1.46"))
		Expect(cat.DefaultVersion(unifi.AppNetwork)).To(Equal("v10.3.58"))
	})

	It("returns the default doc when version is empty", func() {
		cat, err := cli.LoadCatalog()
		Expect(err).NotTo(HaveOccurred())

		doc, resolved, err := cat.Doc(unifi.AppProtect, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(Equal("v7.1.46"))
		Expect(doc.Paths).NotTo(BeNil())
	})

	It("errors on an unknown version", func() {
		cat, err := cli.LoadCatalog()
		Expect(err).NotTo(HaveOccurred())

		_, _, err = cat.Doc(unifi.AppProtect, "v0.0.0")
		Expect(err).To(HaveOccurred())
	})

	It("errors on an unknown app", func() {
		cat, err := cli.LoadCatalog()
		Expect(err).NotTo(HaveOccurred())

		_, _, err = cat.Doc("nonexistent", "")
		Expect(err).To(HaveOccurred())
	})
})
