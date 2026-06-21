---
title: UniFi
author: thathaneydude
description: A SemVer-versioned Go SDK for the UniFi Network and Protect integration APIs.
status: Draft
date: 2026-06-20
---

# UniFi

A SemVer-versioned **Go** SDK — and a spec-driven **CLI** for LLM agents — for the official
**UniFi Network** and **UniFi Protect** integration APIs, generated from OpenAPI with a thin
hand-written layer for **local or remote** authentication.

- **Module:** `github.com/thathaneydude/unifi`
- **Go:** 1.26
- **Apps:** Network (`v10.3.58`), Protect (`v7.1.46`)

## Documentation map

| Section | Contents |
|---|---|
| [Functional requirements](requirements/functional-requirements.md) | What the SDK and CLI must do |
| [Architecture](design/architecture.md) | How it is structured |
| [Authentication](design/authentication.md) | Local vs remote, API keys, TLS |
| [Spec augmentation](design/spec-augmentation.md) | How SDK-ready OpenAPI specs are produced |
| [Decision records](design/decisions/index.md) | Why the key choices were made |
| [Guides](guides/getting-started.md) | Getting started, local vs remote, Protect realtime |
| [Changelog](changelog.md) | Release history |

> This site is published to GitHub Pages with MkDocs Material. The Go API reference and the rendered
> OpenAPI specs are generated into `reference/` during the docs build.
