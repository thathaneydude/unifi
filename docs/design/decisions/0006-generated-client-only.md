---
title: "ADR-0006: Expose the generated client, minimal wrapping"
author: thathaneydude
description: The SDK surfaces generated types directly and adds only auth, transport, and WebSocket glue.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - api-design
---

# ADR-0006: Expose the generated client, minimal wrapping

## Context

A generated SDK can either expose the generated client directly or wrap it in a hand-written facade
adding conveniences (pagination iterators, retries, typed-error helpers, stable re-exports). Wrappers
improve ergonomics but multiply hand-maintained surface — especially with coexisting versions, where a
stable facade would need per-version code.

## Decision

Expose the **generated client and types directly**. The root `unifi` package adds only what every
caller needs: connection/auth (`Local`/`Remote`), the `X-API-KEY` request editor, TLS options, error
typing, and the WebSocket layer. `Conn.Network()/Protect()` return the latest version's generated
`*ClientWithResponses` as-is.

## Consequences

- Minimal hand-written code; less to maintain as versions accrete.
- Consumers work with generated types (`lib/...`), keeping a single source of truth.
- No built-in pagination/retry helpers in the initial releases; consumers compose their own. This can
  be revisited without breaking the generated surface.
