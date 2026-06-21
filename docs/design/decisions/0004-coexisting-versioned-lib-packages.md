---
title: "ADR-0004: Coexisting versioned generated packages under lib/"
author: thathaneydude
description: Generated clients live only under lib/<app>/<appversion>; multiple app versions coexist.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - versioning
---

# ADR-0004: Coexisting versioned generated packages under lib/

## Context

UniFi consoles run different firmware, and each application version can change the API surface.
Generated code should be cleanly separated from hand-written code, and consumers may need a client
matching their console's exact version. We weighed: a single current version per app; coexisting
versions; or versioning only by API path (`/v1`).

## Decision

Place generated code **only** under `lib/<app>/<appversion>` (e.g. `lib/network/v10_3_58`,
package `networkv10_3_58`), purely generated and with **no dependency on the root**. Multiple
application versions **coexist**; `specs/versions.yaml` lists them and marks a `default`. The newest
is the default reached via the root convenience methods (`Conn.Network()/Protect()`); older versions
are constructed directly from their generated package using the root `Conn`'s primitives.

Retention default is **keep all**; pruning a version is an explicit, breaking (major-version) decision.

## Consequences

- Consumers can pin to the exact version their console exposes; new versions are additive.
- The generated/hand-written boundary is explicit, containing `go generate` blast radius.
- Trade-off: repository and module size grow as versions accrete; mitigated by the documented prune
  policy and that this is generated, reviewable code.
- Root imports only the latest version → strictly one-way dependency, no import cycle.
