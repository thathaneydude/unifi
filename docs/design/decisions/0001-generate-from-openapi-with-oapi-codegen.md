---
title: "ADR-0001: Generate the SDK from OpenAPI with oapi-codegen"
author: thathaneydude
description: Use oapi-codegen (v2) to generate idiomatic net/http clients and models.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - codegen
---

# ADR-0001: Generate the SDK from OpenAPI with oapi-codegen

## Context

UniFi publishes OpenAPI 3.1 documents for the Network and Protect integration APIs. The SDK should
track those APIs closely with minimal hand-maintained surface. Candidate Go generators: oapi-codegen,
ogen, openapi-generator (Java), or generating types only and hand-writing the client.

## Decision

Use **oapi-codegen (v2)** to generate clients and models. It produces idiomatic Go on the standard
`net/http` stack, supports `RequestEditorFn` hooks (needed for `X-API-KEY` injection) and pluggable
`*http.Client` (needed for local self-signed TLS), and tolerates the quirks of the upstream specs
better than the stricter alternatives.

## Consequences

- Generated `…WithResponses` methods expose typed per-status responses.
- The root package wires base URL + auth editor + HTTP client into generated constructors.
- `ogen`/`openapi-generator` are not used; their stricter validation or Java toolchain add friction.
