package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LoadConfig", func() {
	var raw = []byte(`
mirror: https://example.test
mirror-commit: 6425bbc2e248a956070500877ce0cd24aedddd43
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
		Expect(cfg.Mirror).To(Equal("https://example.test"))
	})

	It("parses mirror-commit", func() {
		cfg, err := LoadConfig(raw)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.MirrorCommit).To(Equal("6425bbc2e248a956070500877ce0cd24aedddd43"))
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

var _ = Describe("Config.Validate", func() {
	newConfig := func(mirror, commit string) *Config {
		return &Config{Mirror: mirror, MirrorCommit: commit}
	}

	It("accepts a full 40-character commit SHA", func() {
		cfg := newConfig("https://example.test", "6425bbc2e248a956070500877ce0cd24aedddd43")
		Expect(cfg.Validate()).To(Succeed())
	})

	It("rejects an empty mirror", func() {
		cfg := newConfig("", "6425bbc2e248a956070500877ce0cd24aedddd43")
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("mirror is empty")))
	})

	// The regression this guards: a mutable ref makes `just sync` return
	// different bytes over time for an unchanged tree, so CI's drift guard
	// fails on upstream re-scrapes rather than on real drift in this repo.
	It("rejects the mutable ref HEAD", func() {
		cfg := newConfig("https://example.test", "HEAD")
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("non-reproducible")))
	})

	It("rejects a branch name", func() {
		cfg := newConfig("https://example.test", "main")
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("not a full 40-character commit SHA")))
	})

	It("rejects an abbreviated SHA", func() {
		cfg := newConfig("https://example.test", "6425bbc")
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("not a full 40-character commit SHA")))
	})

	It("rejects an empty mirror-commit", func() {
		cfg := newConfig("https://example.test", "")
		Expect(cfg.Validate()).To(MatchError(ContainSubstring("not a full 40-character commit SHA")))
	})
})
