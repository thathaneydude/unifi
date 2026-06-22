# Report Template

The orchestrator renders this skeleton to `./unifi-assessment-YYYY-MM-DD.md`.

## Finding shape

Every finding is an object with these fields:

- `severity` — one of `critical | high | medium | low | info` (see
  severity-rubric.md).
- `title` — short description of the issue.
- `affected_resource` — the specific network / SSID / device / rule / camera.
- `evidence` — the CLI JSON snippet that proves the finding, with sensitive
  values (keys, secrets, PSKs, tokens) redacted as `***`.
- `remediation` — prose guidance on how to fix. Never applied automatically.

## Report skeleton

```markdown
# UniFi Security Assessment — {date}

## Executive Summary
- Console: {name} ({network_version}; Protect {protect_version_or_"absent"})
- Sites: {site_count}
- Skill versions: orchestrator {orch_version}; {per_subskill_versions}
- Findings: {critical} critical, {high} high, {medium} medium, {low} low, {info} info
- Top risks:
  1. {plain-language risk}
  2. {plain-language risk}
  3. {plain-language risk}

## Findings (by severity)
### Critical
{findings or "None."}
### High
{findings or "None."}
### Medium
{findings or "None."}
### Low
{findings or "None."}
### Info
{findings or "None."}

Each finding renders as:
> **{title}** — `{affected_resource}`
> Evidence:
> ```json
> {evidence}
> ```
> Remediation: {remediation}

## Coverage & Limitations
- Domains run: {list}
- Skipped: {e.g. "Protect — no NVR detected"}
- Not assessable via the integration API: {ops that returned not-found or were
  absent on this firmware}
- This assessment reflects only what the official UniFi integration API exposes.
  It is not a substitute for a full pentest or packet-level audit.

## Appendix — Raw collected data
{references to the JSON captured per domain}
```
