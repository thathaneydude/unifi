---
name: unifi-security-assessment
description: Read-only security assessment of a UniFi deployment (Network + Protect) using the unifi CLI. Enumerates consoles on a remote account, lets the user pick which to assess, orchestrates four domain skills as parallel subagents per console, and produces a severity-ranked findings report per console. Use when asked to audit, assess, or review the security posture of one or more UniFi consoles.
version: 0.2.0
---

# UniFi Security Assessment (orchestrator)

This skill is strictly read-only. It gathers, analyzes, and reports. It MUST NOT
mutate configuration: only `get*` (Network) and `GetV1*` (Protect) operations,
`list-operations`, and `schema` are permitted. Mutating operations are out of
scope — for UniFi Network these use verbs like
`create*`/`update*`/`patch*`/`delete*`/`adopt*`/`remove*`/`execute*`; for
UniFi Protect they are `PostV1*`/`PutV1*`/`PatchV1*`/`DeleteV1*`. Findings
recommend fixes in prose; they are never applied.

## Prerequisites
- The `unifi` CLI on PATH (see the repo README for installation).
- Credentials resolved by the CLI itself (`flags > env > .env`). Never print or
  persist secrets; redact any sensitive values that appear in evidence.

## Flow

### 0. Select console(s) — remote accounts
A UniFi cloud account can hold several consoles (UDM, Cloud Key, NVR, …). When
targeting remotely (i.e. `--host`/`UNIFI_HOST` is NOT set), discover them first:

```
unifi consoles list
```

This needs only the shared API key (`--api-key` / `UNIFI_API_KEY`); it does not
need `--console-id`. Then:
- **0 consoles** → stop and report a likely bad/again-scoped API key (the Site
  Manager key from unifi.ui.com → Settings → API Keys is required for remote).
- **1 console** → auto-select it and proceed (same as today's single-console run).
- **>1 consoles** → present the list to the user (`name` · `model` · `id`) and ask
  which to assess: one, several, or all. Do not assume — let the user choose.

For each chosen console, target it by setting `--console <name>` (or `--console-id
<id>` using the `id` from the list) on every CLI call in the steps below, then run
steps 1–4 for that console. Assess consoles sequentially (or as one subagent per
console); each produces its **own** report. When done, print a summary index of
the reports written.

Local runs (`--host` set) skip this step entirely — there is exactly one console.

### 1. Preflight (per selected console)
- Run `unifi network getInfo` to confirm the console is reachable and capture
  the deployment fingerprint (name, version, site count via
  `unifi network getSiteOverviewPage`).
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
Emit the aggregated findings and metadata as a JSON document following the
schema in `references/report-template.md`, then render it to a self-contained
HTML report with the CLI:

```
unifi report --in findings.json --out ./unifi-assessment-<console>-YYYY-MM-DD.html
```

Name the output per console (e.g. a slug of the console name) so multiple
consoles never overwrite each other; for a single-console run the `<console>`
segment may be omitted. The `.html` is the deliverable; `findings.json` is an
intermediate you may leave beside it. After all selected consoles are done,
print an index listing each console and its report path. In the report metadata (`assessed_by` / `skill_versions`), record this
skill's frontmatter `version` and each sub-skill's version, AND the AI model name
+ id running this assessment (e.g. `Claude Opus 4.8` / `claude-opus-4-8`) — state
your own model identity. This lets the report be regenerated and diffed when
skills improve or a newer model re-evaluates the same deployment.

## Guardrails
- Read-only: never call mutating operations (Network `create*`/`update*`/`patch*`/`delete*`/`adopt*`/`remove*`/`execute*`; Protect `PostV1*`/`PutV1*`/`PatchV1*`/`DeleteV1*`).
- Redact secrets in evidence.
- State limitations honestly in the report — this is integration-API scope, not
  a full pentest.
