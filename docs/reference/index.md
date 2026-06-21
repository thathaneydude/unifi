---
title: Go API Reference
author: thathaneydude
description: Go API reference for the unifi package — the hand-written public surface of the UniFi SDK.
status: Draft
date: 2026-06-20
---

# Go API Reference

This section documents the **`unifi` package** — the hand-written public surface of the UniFi SDK.

The generated reference (`unifi.md`) is produced during the docs build by
[gomarkdoc](https://github.com/princjef/gomarkdoc) via `just docs-reference`. It is not committed to
the repository; run `just docs-prepare` or `just docs-build` to generate it locally.

## Package overview

| Symbol | Purpose |
|---|---|
| `Local(host, apiKey, ...Option)` | Connect to a console's local API |
| `Remote(consoleID, apiKey, ...Option)` | Connect via the UniFi cloud connector |
| `Conn` | Authenticated connection — entry-point for all API calls |
| `Conn.Network()` | Return the latest generated Network client |
| `Conn.Protect()` | Return the latest generated Protect client |
| `Conn.WebSocket(ctx, app, path, handler)` | Subscribe to a Protect real-time feed |
| `Option` | Functional option for HTTP client, TLS, user-agent, etc. |

See [unifi.md](unifi.md) for the full generated symbol reference.
