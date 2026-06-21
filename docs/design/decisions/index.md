---
title: Architecture Decision Records
author: thathaneydude
description: Index of accepted design decisions for UniFi.
tags:
  - adr
---

# Architecture Decision Records

Each ADR captures one significant decision, its context, and consequences.

| ADR | Decision |
|---|---|
| [0001](0001-generate-from-openapi-with-oapi-codegen.md) | Generate the SDK from OpenAPI with oapi-codegen |
| [0002](0002-local-and-remote-transports.md) | First-class local and remote transports |
| [0003](0003-pull-specs-in-ci-not-vendored.md) | Pull specs from the mirror in CI instead of vendoring |
| [0004](0004-coexisting-versioned-lib-packages.md) | Coexisting versioned generated packages under `lib/` |
| [0005](0005-hand-written-websocket-client.md) | Hand-written, version-agnostic WebSocket client |
| [0006](0006-generated-client-only.md) | Expose the generated client, minimal wrapping |
| [0007](0007-protect-version-pin.md) | Pin Protect to v7.1.46 until v7.1.83 is mirrored |
| [0008](0008-conventional-commits-and-releases.md) | Conventional Commits, git-cliff, and goreleaser |
| [0009](0009-docs-site-mkdocs.md) | Documentation site on MkDocs Material via GitHub Pages |
| [0010](0010-testing-ginkgo-gomega-counterfeiter.md) | Testing with Ginkgo, Gomega, and counterfeiter |
