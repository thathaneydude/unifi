---
title: "ADR-0010: Testing with Ginkgo, Gomega, and counterfeiter"
author: thathaneydude
description: Unit and e2e suites use Ginkgo/Gomega BDD with counterfeiter-generated fakes.
status: Accepted
date: 2026-06-20
tags:
  - adr
  - testing
---

# ADR-0010: Testing with Ginkgo, Gomega, and counterfeiter

## Context

The SDK needs unit and end-to-end tests that exercise transport, auth, TLS, error mapping, and
WebSocket decoding without depending on a live console, plus optional real-console verification.

## Decision

Use **Ginkgo** (BDD specs) with **Gomega** matchers for both unit and e2e suites, and
**counterfeiter** to generate fakes for the HTTP doer / request seams and other SDK interfaces.
Unit specs run against fakes and `httptest` servers with golden fixtures; a build-tagged `e2e/` suite
runs against mock servers and, when env credentials are present, a real console (skipped otherwise).

## Consequences

- Expressive, readable suites with strong assertions.
- Network-free unit tests via counterfeiter fakes; deterministic and fast.
- Generated fakes are committed and regenerate with no diff (`go generate ./...`).
