---
name: unifi-security-assessment
description: Read-only security assessment of a UniFi deployment (Network + Protect) using the unifi CLI. Orchestrates four domain skills as parallel subagents and produces a severity-ranked findings report. Use when asked to audit, assess, or review the security posture of a UniFi console.
version: 0.1.0
---

# UniFi Security Assessment (orchestrator)

This skill is strictly read-only. It gathers, analyzes, and reports. It MUST NOT
mutate configuration: only `Get*` operations, `list-operations`, and `schema`
are permitted. Mutating operations (`Patch*`, `Post*`, `Put*`, `Delete*`) are
out of scope — findings recommend fixes in prose, they are never applied.

## Prerequisites
- The `unifi` CLI on PATH (see the repo README for installation).
- Credentials resolved by the CLI itself (`flags > env > .env`). Never print or
  persist secrets; redact any sensitive values that appear in evidence.

## Flow

### 1. Preflight
- Run `unifi network GetInfo` to confirm the console is reachable and capture
  the deployment fingerprint (name, version, site count via
  `unifi network GetSiteOverviewPage`).
- Detect Protect: run `unifi protect list-operations`. If it fails or there is
  no NVR (`unifi protect GetV1Nvrs` returns nothing), mark Protect absent and
  skip that domain cleanly.

### 2. Dispatch (parallel subagents)
Launch the four domain skills as parallel, read-only subagents. Give each:
- its domain skill name to follow,
- the instruction to return ONLY a findings list (using the finding shape in
  `references/report-template.md`) plus a one-line domain summary,
- the reminder that any mutating op is out of scope.

The four domains:
- `unifi-network-security` — firewall, ACL, DNS, VPN, RADIUS.
- `unifi-segmentation-wifi` — VLANs/networks, WiFi encryption, guest isolation.
- `unifi-asset-inventory` — devices, firmware, clients, vouchers.
- `unifi-protect-security` — cameras, RTSP, Protect users. Skip if Protect absent.

### 3. Aggregate
Collect all findings, de-duplicate, and sort by severity using the order in
`references/severity-rubric.md`.

### 4. Write report
Render `references/report-template.md` to `./unifi-assessment-YYYY-MM-DD.md` in
the working directory. Record this skill's version (0.1.0) and each sub-skill's
version in the executive summary so the report can be regenerated and diffed
against improved skills.

## Guardrails
- Read-only: never call mutating operations.
- Redact secrets in evidence.
- State limitations honestly in the report — this is integration-API scope, not
  a full pentest.
