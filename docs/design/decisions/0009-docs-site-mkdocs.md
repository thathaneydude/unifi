---
title: "ADR-0009: Documentation site on MkDocs Material via GitHub Pages"
author: thathaneydude
description: Guides, Go API reference, and rendered OpenAPI are published with MkDocs Material to GitHub Pages.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - docs
---

# ADR-0009: Documentation site on MkDocs Material via GitHub Pages

## Context

The SDK documentation must be served on GitHub Pages and cover narrative guides, the Go API reference,
and the OpenAPI reference for both apps. Options included Hugo, a Redoc/Scalar-only OpenAPI page, or
MkDocs Material.

## Decision

Use **MkDocs Material**. The `docs/` tree is the source; pages carry YAML frontmatter
(`title`, `description`, `tags`). The Go API reference is generated with **gomarkdoc** into
`docs/reference/`; the OpenAPI specs are rendered with **Redoc/Scalar**. A `docs.yml` workflow builds
and deploys to GitHub Pages.

## Consequences

- A single, searchable, themed site with guides + API + OpenAPI.
- Frontmatter drives titles/metadata and keeps pages consistent.
- Docs build is part of CI/CD and can gate on build success.
