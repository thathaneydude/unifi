---
title: "ADR-0008: Conventional Commits, git-cliff, and goreleaser"
author: thathaneydude
description: Commit messages drive changelog generation and SemVer releases.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - release
  - process
---

# ADR-0008: Conventional Commits, git-cliff, and goreleaser

## Context

The repository must enforce commit linting for changelog generation and release the SDK on SemVer.
Options spanned Node-based tooling (commitlint + semantic-release) and Go-native/Rust tooling.

## Decision

Adopt **Conventional Commits**, enforced in CI with a Go-native linter (no Node toolchain). Generate
`CHANGELOG.md` with **git-cliff** (`cliff.toml`). Cut releases with **goreleaser** on git tags;
because this is a library, goreleaser's build artifacts are skipped and it is used for the GitHub
release plus release notes sourced from git-cliff.

**SemVer is enforced at two layers.** goreleaser refuses to run on a tag it cannot parse as semantic
versioning; `.goreleaser.yaml` adds an explicit `before` hook that rejects any tag not matching strict
`vMAJOR.MINOR.PATCH[-prerelease][+build]`. The `release.yml` workflow performs the same check before
goreleaser starts (defense in depth) and maps SemVer prereleases to GitHub pre-releases
(`release.prerelease: auto`).

## Consequences

- A pure Go/Rust release toolchain; no `package.json` in a Go repo.
- Changelog and release notes derive deterministically from commit history.
- Contributors must follow Conventional Commits; CI rejects non-conforming messages.
