---
title: "ADR-0003: Pull specs from the mirror in CI instead of vendoring raw upstream"
author: thathaneydude
description: specgen pulls pinned specs into a gitignored cache; only augmented specs and generated code are committed.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - openapi
  - ci
---

# ADR-0003: Pull specs from the mirror in CI instead of vendoring raw upstream

## Context

UniFi's `developer.ui.com` is a client-rendered SPA, so specs aren't directly downloadable. The
community repo `opastorello/unifi-api-docs` mirrors the official OpenAPI JSON daily and versioned.
We must keep the SDK current without manually copying large JSON blobs into the repo, yet consumers
of a Go module need committed generated code (`go get` cannot run codegen).

## Decision

Do **not** vendor raw upstream JSON. `tools/specgen` (run by `just sync` and in CI) pulls the pinned
specs from the mirror into a gitignored `specs/.cache/`, applies overlays, and writes the augmented
`specs/build/**`. We commit the **augmented specs and generated `lib/**` code**. A scheduled
`spec-sync` workflow opens regeneration PRs; CI's drift guard rejects stale generated artifacts.

## Consequences

- No large raw vendored files; upstream provenance is reproducible from `versions.yaml` + overlays.
- Generated code stays committed, so `go get` works without a build step.
- Requires network access during `just sync` (CI and local), and a dependency on the mirror's
  availability for regeneration (not for consuming the SDK).

## Future work — remove the external mirror dependency

The current `specgen` pulls already-converted OpenAPI JSON from the third-party
`opastorello/unifi-api-docs` mirror, which itself scrapes `developer.ui.com` and performs the
conversion. This couples regeneration to an external project's availability and conversion choices.

**TODO:** Implement a native Go spec-pull and conversion in `tools/specgen` that fetches the
specification directly from the official UniFi source (`developer.ui.com` / its underlying data
endpoints) and produces the OpenAPI document in-process, removing the mirror dependency entirely.
The mirror would then become an optional fallback rather than the source of truth. Tracked as a
`TODO(specgen)` in `tools/specgen/sync.go`.
