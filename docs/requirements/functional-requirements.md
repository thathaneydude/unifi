---
title: Functional Requirements
author: thathaneydude
description: What the UniFi SDK must do — actors, functional and non-functional requirements, acceptance criteria.
status: Accepted
date: 2026-06-20
tags:
  - requirements
---

# Functional Requirements — UniFi SDK

> This document defines **what** the UniFi SDK must do. The **how** lives in the
> [design](../design/architecture.md) docs and the
> [architecture decision records](../design/decisions/index.md).

## 1. Purpose & scope

The UniFi SDK is a SemVer-versioned **Go** module that provides typed, generated clients for the
official **UniFi Network** and **UniFi Protect** integration APIs, plus a thin hand-written layer that
makes authentication — **local or remote** — trivial. It also ships a **CLI** (`unifi`) whose primary
user is an **LLM agent**: a co-equal product built on the same core and specs.

In scope:

- UniFi **Network** integration API (pinned: `v10.3.58`).
- UniFi **Protect** integration API (pinned: `v7.1.46`; target `v7.1.83` once mirrored upstream).
- Local (per-console) and remote (cloud connector) transports.
- Protect real-time WebSocket subscriptions (**SDK only**).
- A spec-driven `unifi` CLI exposing every Network/Protect REST operation, optimized for
  LLM agents (JSON output, structured errors, discovery, safe writes).

Out of scope (for the initial releases):

- The UniFi **Site Manager** aggregate API and other UniFi apps (Access, Talk, etc.).
- Higher-level orchestration helpers beyond the generated client (pagination iterators, retries,
  caching) — see [ADR-0006](../design/decisions/0006-generated-client-only.md).
- Protect WebSocket subscriptions **in the CLI** — realtime stays an SDK feature.
- Hand-curated "friendly verb" CLI commands — CLI commands map 1:1 to OpenAPI operations.

## 2. Actors

| Actor | Description |
|---|---|
| SDK consumer | A Go developer importing the module to automate UniFi Network/Protect. |
| CLI / LLM agent | An LLM agent (or script) that invokes the `unifi` CLI via a shell, parses its JSON output, and branches on its exit codes. The CLI's **primary** user. |
| UniFi console | A local UniFi OS device (gateway/NVR) exposing app APIs under `/proxy/<app>/integration`. |
| UniFi cloud connector | `api.ui.com` reverse-proxy reaching a console remotely by `consoleId`. |
| Spec-sync bot | Scheduled automation that pulls upstream specs and proposes regeneration PRs. |

## 3. Functional requirements

### FR-AUTH — Authentication & connection
- **FR-AUTH-1** Authenticate every request with the `X-API-KEY` header.
- **FR-AUTH-2** Provide one obvious constructor for **local** access:
  `unifi.Local(host, apiKey, opts...)` targeting `https://{host}/proxy/{app}/integration`.
- **FR-AUTH-3** Provide one obvious constructor for **remote** access:
  `unifi.Remote(consoleID, apiKey, opts...)` targeting
  `https://api.ui.com/v1/connector/consoles/{consoleID}/{app}/integration`.
- **FR-AUTH-4** Support self-signed local console certificates via TLS options
  (`WithInsecureSkipVerify`, `WithRootCAs`, `WithTLSConfig`).
- **FR-AUTH-5** Allow a caller-supplied `*http.Client`, request timeout, and custom `User-Agent`.

### FR-NET — Network API coverage
- **FR-NET-1** Expose every operation of the pinned Network spec as a typed, generated client method.
- **FR-NET-2** Surface request/response models as generated Go types.
- **FR-NET-3** Map non-2xx responses to a typed SDK error carrying the API error envelope.

### FR-PRO — Protect API coverage
- **FR-PRO-1** Expose every REST operation of the pinned Protect spec as typed, generated methods.
- **FR-PRO-2** Provide a typed real-time subscription API over WebSocket for
  `/v1/subscribe/devices` and `/v1/subscribe/events`.
- **FR-PRO-3** The WebSocket client reuses the same auth + TLS configuration as REST.

### FR-VER — Versioning
- **FR-VER-1** The module is released via **SemVer git tags** (goreleaser). Releases are **rejected**
  unless the tag is strict SemVer (`vX.Y.Z[-prerelease][+build]`), enforced by goreleaser and the
  release workflow.
- **FR-VER-2** Each pinned UniFi **application version** is a coexisting package under
  `lib/<app>/<appversion>`; multiple versions may be imported simultaneously.
- **FR-VER-3** The newest pinned version is the default reached through the root convenience methods.
- **FR-VER-4** Pinned versions and the default are declared in `specs/versions.yaml`.

### FR-SPEC — Spec lifecycle
- **FR-SPEC-1** SDK-ready OpenAPI specs are produced **deterministically** from upstream by applying
  checked-in overlays (`just sync`); the same inputs always yield byte-identical output.
- **FR-SPEC-2** Raw upstream specs are **not** committed; they are pulled on demand into a gitignored
  cache.
- **FR-SPEC-3** The augmentation injects the `X-API-KEY` security scheme, parameterized local/remote
  servers, and (for Protect) synthesized tags.
- **FR-SPEC-4** Augmented specs are validated against OpenAPI 3.1 before generation.

### FR-CLI — Command-line interface (LLM-first)
- **FR-CLI-1** Ship a single `unifi` binary that exposes **every** Network and Protect REST
  operation as `unifi <app> <operationId> [flags]`, derived at runtime from the embedded
  `specs/build/**` (no per-operation hand-written or generated command code).
- **FR-CLI-2** Default output is **JSON** on stdout (the API response body verbatim); a single
  `--format json|raw|human` flag selects the renderer (`raw` passes bytes through, `human` is a
  best-effort view).
- **FR-CLI-3** Errors are a **structured JSON envelope** on stderr (operation, HTTP status, API
  error body, hint) with **stable exit codes**: `0` success, `1` usage/validation, `2`
  auth/config, `3` API non-2xx, `4` transport.
- **FR-CLI-4** Provide **discovery** commands: `unifi schema` and `unifi <app> list-operations`
  emit JSON; `unifi <app> <operationId> --help` renders usage from the spec.
- **FR-CLI-5** Resolve auth from flags → env (`UNIFI_API_KEY`, `UNIFI_HOST` for local /
  `UNIFI_CONSOLE_ID` for remote, `UNIFI_INSECURE`) → optional config file, reusing the
  `unifi/` core. Secrets are never echoed.
- **FR-CLI-6** Mutating operations (POST/PUT/PATCH/DELETE) support `--dry-run` (print the
  intended request and exit `0` without calling the API) and require `--confirm` to execute.
- **FR-CLI-7** The newest pinned app version is the CLI default and is always used for operation
  commands. `unifi schema --api-version` selects a coexisting pinned version for inspection;
  per-operation version selection is deferred until more than one version is pinned (FR-VER-3).

### FR-DX — Developer experience
- **FR-DX-1** Provide runnable `examples/` for local and remote access of both apps, including using a
  non-default coexisting version.
- **FR-DX-2** Publish SDK documentation (guides + Go reference + rendered OpenAPI) to GitHub Pages.
- **FR-DX-3** Errors are actionable: they identify the operation, HTTP status, and API error body.

### FR-REPO — Repository & process
- **FR-REPO-1** Use **Justfiles** as the task runner (no Make/Task).
- **FR-REPO-2** Enforce **Conventional Commits**; generate `CHANGELOG.md` from commit history with
  git-cliff.
- **FR-REPO-3** CI gates every PR: build, vet, lint, unit + e2e tests, commit-lint, spec validation,
  and a **drift guard** ensuring committed generated artifacts match a fresh `just sync && just gen`.
- **FR-REPO-4** A scheduled workflow proposes regeneration and version-bump PRs.
- **FR-REPO-5** Agent guidance lives in `.github/AGENTS.md` (not `CLAUDE.md`).
- **FR-REPO-6** Target **Go 1.26**.

## 4. Non-functional requirements

- **NFR-1 Determinism** — generation and spec augmentation are reproducible in CI.
- **NFR-2 No hand edits in `lib/`** — generated packages are machine-owned; corrections happen in
  overlays or generator config.
- **NFR-3 Single module** — one `go.mod`, one tag stream, so `go get` is trivial.
- **NFR-4 Minimal dependencies** — std `net/http` transport; only `coder/websocket` for realtime and
  test/codegen tooling otherwise.
- **NFR-5 Testability** — SDK seams are interfaces fakeable by counterfeiter; suites use Ginkgo/Gomega.

## 5. Acceptance criteria

- A consumer authenticates to a local console and a remote console with one constructor each and lists
  Network sites and Protect cameras.
- A consumer subscribes to Protect events and receives typed messages.
- An agent runs `unifi protect list-operations`, then `unifi protect GetV1Cameras` and parses
  the JSON result; a bad credential yields exit code `2` and a missing `--confirm` on a write
  yields exit code `1`.
- `just sync && just gen` produce no diff on a clean checkout (drift guard green).
- A tagged release yields a goreleaser GitHub release with a git-cliff changelog.
- The docs site builds and deploys to GitHub Pages.

## 6. Traceability

Each requirement maps to design sections and ADRs:

| Requirement | Design | ADR |
|---|---|---|
| FR-AUTH-* | [authentication](../design/authentication.md) | [0002](../design/decisions/0002-local-and-remote-transports.md) |
| FR-NET/PRO-* | [architecture](../design/architecture.md) | [0001](../design/decisions/0001-generate-from-openapi-with-oapi-codegen.md) |
| FR-PRO-2/3 | [architecture](../design/architecture.md#real-time-websocket) | [0005](../design/decisions/0005-hand-written-websocket-client.md) |
| FR-VER-* | [architecture](../design/architecture.md#package-layout) | [0004](../design/decisions/0004-coexisting-versioned-lib-packages.md) |
| FR-SPEC-* | [spec-augmentation](../design/spec-augmentation.md) | [0003](../design/decisions/0003-pull-specs-in-ci-not-vendored.md) |
| FR-CLI-* | [architecture](../design/architecture.md) | — |
