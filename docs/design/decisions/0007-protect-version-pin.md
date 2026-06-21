---
title: "ADR-0007: Pin Protect to v7.1.46 until v7.1.83 is mirrored"
author: thathaneydude
description: The requested Protect v7.1.83 is not cleanly extractable; pin v7.1.46 now and promote v7.1.83 via the version-bump bot.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - protect
  - versioning
---

# ADR-0007: Pin Protect to v7.1.46 until v7.1.83 is mirrored

## Context

The target Protect version is **v7.1.83**, but it is newer than the latest version in the
`opastorello/unifi-api-docs` mirror (v7.1.46) and is not cleanly extractable from the client-rendered
`developer.ui.com` SPA. Network's target (v10.3.58) is available exactly.

## Decision

Pin Protect to **v7.1.46** in `specs/versions.yaml` for now. When the mirror publishes v7.1.83, the
scheduled version-bump workflow opens a PR that **appends** v7.1.83 as a coexisting package
(`lib/protect/v7_1_83`) and promotes it to `default`. v7.1.46 remains importable per the coexistence
policy ([ADR-0004](0004-coexisting-versioned-lib-packages.md)).

## Consequences

- The SDK ships now with a clean, validated Protect spec.
- No manual scraping of the SPA is required; the upgrade path is automated and reviewable.
- Until then, examples and the default Protect client target v7.1.46.
