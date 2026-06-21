---
title: CLI for LLM agents
author: thathaneydude
description: Using the unifi CLI — discovery, output, errors, auth, and safe writes.
status: Draft
date: 2026-06-20
tags:
  - cli
  - guides
---

# CLI for LLM agents

The `unifi` CLI exposes every Network and Protect REST operation, generated at
runtime from the embedded OpenAPI specs. Its primary user is an LLM agent.

## Install

```sh
go build -o bin/unifi ./cmd/unifi   # or `just build-cli`
```

## Authenticate

Credentials come from flags, then environment:

| Variable | Flag | Purpose |
|---|---|---|
| `UNIFI_API_KEY` | `--api-key` | API key (required) |
| `UNIFI_HOST` | `--host` | local console host (selects local transport) |
| `UNIFI_CONSOLE_ID` | `--console-id` | remote console id (selects remote transport) |
| `UNIFI_INSECURE` | `--insecure` | skip TLS verification for self-signed certs |

Set exactly one of host / console-id.

## Discover

```sh
unifi protect list-operations          # JSON array of operations
unifi schema --app protect             # full embedded OpenAPI spec
unifi protect GetV1Cameras --help      # usage for one operation
```

## Call

```sh
export UNIFI_API_KEY=... UNIFI_HOST=192.168.1.1
unifi protect GetV1Cameras                          # JSON to stdout
unifi network getAdoptedDeviceDetails --format human # human-readable view
```

## Output and errors

- stdout is JSON by default; `--format json|raw|human` selects the renderer.
- Errors are a JSON envelope on stderr with stable exit codes: `0` ok, `1` usage,
  `2` auth/config, `3` API non-2xx, `4` transport.

## Safe writes

Mutating operations require `--confirm` and support `--dry-run`:

```sh
unifi protect PatchV1AlarmHubsId --body '{...}' --dry-run   # preview only
unifi protect PatchV1AlarmHubsId --body '{...}' --confirm   # execute
```

Operation IDs for Protect are synthesized from method and path (the upstream
Protect spec omits operationIds), so they match the generated SDK method names.

Real-time WebSocket subscriptions are intentionally **not** in the CLI; use the
Go SDK for those.

## Contract notes for agents

- Every successful command prints JSON to stdout. The one exception is `--help`
  / `-h`, which prints human-readable usage (the only non-JSON, exit-0 surface).
- Running with no command, an unknown command, or an unknown operation returns
  the JSON error envelope on stderr with exit code `1`.
- An empty `2xx` response body renders as `null` in JSON mode.
- Prefer the `UNIFI_API_KEY` environment variable over `--api-key` so the key
  does not appear in the host's process argument list.
