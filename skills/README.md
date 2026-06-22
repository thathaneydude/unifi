# UniFi Security Assessment Skills

Read-only [Agent Skills](https://docs.claude.com/en/docs/claude-code/skills) that
drive the [`unifi` CLI](../README.md) to assess the security posture of a UniFi
deployment (Network + Protect) and produce a severity-ranked findings report.

Each skill is a plain-Markdown `SKILL.md` that instructs an LLM agent which CLI
operations to run, what to look for, and how to grade findings. They **never
mutate configuration** — only `get*` / `GetV1*` operations, `list-operations`,
and `schema` are used.

## What's here

| Skill | Version | Purpose |
|---|---|---|
| [`unifi-security-assessment`](unifi-security-assessment/SKILL.md) | 0.1.0 | **Orchestrator.** Preflights the console, detects Protect, runs the four domain skills below as parallel subagents, and aggregates one report. |
| [`unifi-network-security`](unifi-network-security/SKILL.md) | 0.1.0 | Firewall policies/zones, ACL rules + ordering, DNS policies, VPN, RADIUS. |
| [`unifi-segmentation-wifi`](unifi-segmentation-wifi/SKILL.md) | 0.1.0 | VLAN/network segmentation, WiFi encryption, guest/IoT isolation. |
| [`unifi-asset-inventory`](unifi-asset-inventory/SKILL.md) | 0.1.0 | Device firmware currency, rogue/pending devices, unexpected clients, vouchers. |
| [`unifi-protect-security`](unifi-protect-security/SKILL.md) | 0.1.0 | Camera/NVR firmware, RTSP/RTSPS exposure, Protect user access scope. |

The orchestrator owns two shared references used by every domain skill:
`unifi-security-assessment/references/severity-rubric.md` (severity definitions)
and `unifi-security-assessment/references/report-template.md` (report skeleton +
finding shape).

## Prerequisites

1. **The `unifi` CLI on your `PATH`.** See the [main README](../README.md#cli)
   for install options (prebuilt binary or `go install`).
2. **Credentials**, resolved by the CLI itself (`flags > environment > .env`):

   ```sh
   export UNIFI_API_KEY=your-api-key
   export UNIFI_HOST=192.168.1.1     # local console; or UNIFI_CONSOLE_ID for remote
   ```

   The skills never read, print, or persist credentials themselves — they only
   shell out to `unifi`, which handles auth. Run `unifi network getInfo` once to
   confirm the CLI reaches your console before assessing.

## Install (Claude Code)

Skills are discovered from a `skills/` directory under `~/.claude` (personal) or
`.claude` in a project (project-scoped). Copy the five skill directories into one
of those locations:

```sh
# Personal — available in every session
mkdir -p ~/.claude/skills
cp -R unifi-security-assessment unifi-network-security unifi-segmentation-wifi \
      unifi-asset-inventory unifi-protect-security ~/.claude/skills/

# …or project-scoped — available only inside a specific repo
mkdir -p /path/to/project/.claude/skills
cp -R unifi-* /path/to/project/.claude/skills/
```

To track upstream changes instead of copying, symlink them:

```sh
for d in unifi-security-assessment unifi-network-security unifi-segmentation-wifi \
         unifi-asset-inventory unifi-protect-security; do
  ln -s "$(pwd)/$d" ~/.claude/skills/"$d"
done
```

> **Other agents:** these `SKILL.md` files are model-agnostic Markdown
> instructions. Any agent that can run shell commands can follow them — install
> them wherever that agent loads skills, or just point it at a skill file.

## Use

**Full assessment** — invoke the orchestrator:

```
Run a UniFi security assessment of my deployment.
```

The agent loads `unifi-security-assessment`, which fans out to the four domain
skills and writes a report to `./unifi-assessment-YYYY-MM-DD.md` in the working
directory. The report contains:

- an **executive summary** (console/version fingerprint, the skill versions that
  produced it, finding counts by severity, and the top risks),
- **findings by severity** (critical → info) with affected resource, evidence
  (the CLI JSON that proves it, secrets redacted), and remediation guidance,
- **coverage & limitations** (what ran, what was skipped, what the integration
  API can't see), and an appendix.

**Single domain** — invoke any domain skill on its own, e.g.:

```
Audit just my UniFi firewall and ACL rules.   →  unifi-network-security
Check my WiFi and VLAN segmentation.          →  unifi-segmentation-wifi
```

## Read-only guarantee

Every skill is scoped to read-only operations and recommends fixes in prose
rather than applying them. This is enforced mechanically: `skills/validate_test.go`
checks each `SKILL.md` for valid frontmatter, a SemVer `version`, the
`This skill is strictly read-only` marker, the orchestrator's cross-references,
and — via `TestNoMutatingOps` — that no skill references a mutating operation
(`create*`/`update*`/`patch*`/`delete*`/`adopt*`/`remove*`/`execute*` for
Network; `PostV1*`/`PutV1*`/`PatchV1*`/`DeleteV1*` for Protect).

```sh
just validate-skills        # from the repo root
```

## Versioning & regeneration

Each skill carries a `version` in its frontmatter, and the orchestrator records
the versions it used in the report's executive summary. When a skill's checklist
or severity logic improves, bump its `version` (SemVer) — a prior report can then
be regenerated and meaningfully diffed against the newer skills.

## Scope

These skills assess what the official UniFi integration API exposes. They are a
repeatable hardening review, **not** a substitute for a full penetration test or
packet-level audit.
