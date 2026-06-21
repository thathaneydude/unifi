---
title: "ADR-0005: Hand-written, version-agnostic WebSocket client"
author: thathaneydude
description: Protect real-time subscriptions are served by a single hand-written WebSocket layer in the root package.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - protect
  - realtime
---

# ADR-0005: Hand-written, version-agnostic WebSocket client

## Context

Protect exposes real-time endpoints (`/v1/subscribe/devices`, `/v1/subscribe/events`) over WebSocket.
OpenAPI cannot model WebSocket streams, and generated REST clients cannot handle them. The event and
device payloads, however, are described as schemas and are generated as types per app version.

## Decision

Implement a single, **version-agnostic** WebSocket layer in `unifi/websocket.go` using
`coder/websocket`. `Conn.Subscribe(ctx, path)` dials `wss://…`, reusing the same `X-API-KEY` header
and TLS configuration as REST, and returns a stream of raw frames. `unifi.Decode[T](frame)` unmarshals
a frame into the caller's chosen `lib` version event/device type via generics.

## Consequences

- One implementation serves all coexisting versions; only the typed payload (from the chosen `lib`
  package) differs.
- Realtime shares auth/TLS with REST, so local self-signed consoles work uniformly.
- `lib/` stays purely generated; no per-version hand-written WebSocket glue.
