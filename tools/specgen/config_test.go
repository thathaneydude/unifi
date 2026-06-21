package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadConfig", func() {
	var raw = []byte(`
mirror: https://example.test/HEAD
retain: all
apps:
  network:
    default: v10.3.58
    versions:
      - v10.3.58
  protect:
    default: v7.1.46
    versions:
      - v7.1.46
`)

	It("parses mirror", func() {
		cfg, err := LoadConfig(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Mirror).To(Equal("https://example.test/HEAD"))
	})

	It("parses network default version", func() {
		cfg, err := LoadConfig(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Apps["network"].Default).To(Equal("v10.3.58"))
	})

	It("parses protect versions list", func() {
		cfg, err := LoadConfig(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Apps["protect"].Versions).To(ContainElement("v7.1.46"))
	})
})
