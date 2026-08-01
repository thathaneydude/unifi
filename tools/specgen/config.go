package main

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// AppConfig holds the pinned version info for a single UniFi application.
type AppConfig struct {
	Default  string   `yaml:"default"`
	Versions []string `yaml:"versions"`
}

// Config is the top-level structure of specs/versions.yaml.
type Config struct {
	Mirror       string               `yaml:"mirror"`
	MirrorCommit string               `yaml:"mirror-commit"`
	Retain       string               `yaml:"retain"`
	Apps         map[string]AppConfig `yaml:"apps"`
}

// fullCommitSHA matches a complete 40-character hex git object name.
// Abbreviated SHAs and named refs are both rejected: only a full SHA names one
// immutable commit for the life of the mirror repo.
var fullCommitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Validate reports whether the config can produce a reproducible fetch.
//
// The mirror pin is the invariant that matters. The mirror URL ended in the ref
// HEAD until 2026-07-31, which meant `just sync` fetched whatever the mirror
// held at the moment it ran. The upstream scraper rewrites specs in place under
// an unchanged version directory, so a spec that was correct when committed
// silently stopped matching a fresh fetch, and CI's drift guard failed with
// nothing in this repo having changed. A full commit SHA makes any given tree of
// this repo fetch the same bytes forever.
func (c *Config) Validate() error {
	if c.Mirror == "" {
		return fmt.Errorf("mirror is empty")
	}
	if !fullCommitSHA.MatchString(c.MirrorCommit) {
		return fmt.Errorf(
			"mirror-commit %q is not a full 40-character commit SHA: a mutable ref "+
				"(HEAD, a branch, a tag) makes `just sync` non-reproducible",
			c.MirrorCommit,
		)
	}
	return nil
}

// LoadConfig parses a versions.yaml byte slice into a Config. It does not
// validate; call Config.Validate for that.
func LoadConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
