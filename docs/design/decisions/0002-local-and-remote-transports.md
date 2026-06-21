---
title: "ADR-0002: First-class local and remote transports"
author: thathaneydude
description: Two explicit constructors (Local/Remote) build the same Conn over different base URLs.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - authentication
---

# ADR-0002: First-class local and remote transports

## Context

UniFi exposes the same integration API two ways: locally on a console under
`https://{host}/proxy/{app}/integration`, and remotely through the cloud connector at
`https://api.ui.com/v1/connector/consoles/{consoleId}/{app}/integration`. Both authenticate with
`X-API-KEY`; only the base URL differs. Local consoles use self-signed certificates.

## Decision

Provide two explicit constructors returning the same `*unifi.Conn`:

- `unifi.Local(host, apiKey, opts...)`
- `unifi.Remote(consoleID, apiKey, opts...)`

`Conn` resolves the per-app base URL and exposes `RequestEditor()` (sets `X-API-KEY`) and
`HTTPClient()`. TLS options (`WithInsecureSkipVerify`, `WithRootCAs`, `WithTLSConfig`) handle
self-signed local certs.

## Consequences

- "Authenticate locally or remotely" is a one-line choice for consumers.
- The same `Conn` drives REST and WebSocket, for any coexisting generated version.
- `WithInsecureSkipVerify` is documented as local-only.
