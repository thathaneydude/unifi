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
| [`unifi-security-assessment`](unifi-security-assessment/SKILL.md) | 0.2.0 | **Orchestrator.** Enumerates consoles on a remote account and lets the user pick which to assess; per console preflights, detects Protect, runs the four domain skills below as parallel subagents, and aggregates one report each. |
| [`unifi-network-security`](unifi-network-security/SKILL.md) | 0.1.0 | Firewall policies/zones, ACL rules + ordering, DNS policies, VPN, RADIUS. |
| [`unifi-segmentation-wifi`](unifi-segmentation-wifi/SKILL.md) | 0.1.0 | VLAN/network segmentation, WiFi encryption, guest/IoT isolation. |
| [`unifi-asset-inventory`](unifi-asset-inventory/SKILL.md) | 0.1.0 | Device firmware currency, rogue/pending devices, unexpected clients, vouchers. |
| [`unifi-protect-security`](unifi-protect-security/SKILL.md) | 0.1.0 | Camera/NVR firmware, RTSP/RTSPS exposure, Protect user access scope. |

The orchestrator owns two shared references used by every domain skill:
`unifi-security-assessment/references/severity-rubric.md` (severity definitions)
and `unifi-security-assessment/references/report-template.md` (findings.json
schema + finding shape).

## Prerequisites

1. **The `unifi` CLI on your `PATH`.** See the [main README](../README.md#cli)
   for install options (prebuilt binary or `go install`).
2. **Credentials**, resolved by the CLI itself (`flags > environment > .env`):

   ```sh
   export UNIFI_API_KEY=your-api-key
   export UNIFI_HOST=192.168.1.1     # local console; or UNIFI_CONSOLE_ID for remote
   ```

   A combined assessment touches both Network and Protect, which mint **separate**
   local API keys. Set both so each domain skill authenticates without swapping:

   ```sh
   export UNIFI_NETWORK_API_KEY=your-network-key
   export UNIFI_PROTECT_API_KEY=your-protect-key
   ```

   Each app falls back to `UNIFI_API_KEY` when its app-specific key is unset.
   The skills never read, print, or persist credentials themselves — they only
   shell out to `unifi`, which handles auth. Run `unifi network getInfo` and
   `unifi protect list-operations` once to confirm the CLI reaches both apps
   before assessing.

## Install

Use the [`skills` CLI](https://github.com/vercel-labs/skills) to install these
directly from the repo — it auto-discovers the skills under `skills/`:

```sh
# See what's available
npx skills add thathaneydude/unifi --list

# Install all of them for Claude Code, globally (available in every session)
npx skills add thathaneydude/unifi -g -a claude-code

# …or a subset — always include the orchestrator (see note)
npx skills add thathaneydude/unifi \
  --skill unifi-security-assessment --skill unifi-network-security -g -a claude-code
```

Drop `-g` to install into the current project (`.claude/skills/`) instead of
`~/.claude/skills/`. Use `-a` to target other agents (e.g. `-a codex`,
`-a opencode`) — the `SKILL.md` files are model-agnostic Markdown, so any
shell-capable agent can follow them.

> **Install the orchestrator alongside any domain skill.** The four domain skills
> reference the orchestrator's shared files via
> `../unifi-security-assessment/references/…`. Installing everything (the default)
> is simplest; if you cherry-pick with `--skill`, always include
> `unifi-security-assessment`.

## Use

**Full assessment** — invoke the orchestrator:

```
Run a UniFi security assessment of my deployment.
```

The agent loads `unifi-security-assessment`, which fans out to the four domain
skills and renders a self-contained, UniFi-branded HTML report (via
`unifi report`) to `./unifi-assessment-YYYY-MM-DD.html` in the working
directory. The report contains:

- an **executive summary** (console/version fingerprint, the skill versions
  **and the AI model + version** that produced it, finding counts by severity,
  and the top risks),
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
both the skill versions **and the AI model + version** that produced the report in
its executive summary. When a skill's checklist or severity logic improves, bump
its `version` (SemVer); when a more capable model becomes available, re-run the
assessment with it. Either way, a prior report can be regenerated and meaningfully
diffed — by skill version or by model — against the same deployment.

## Scope

These skills assess what the official UniFi integration API exposes. They are a
repeatable hardening review, **not** a substitute for a full penetration test or
packet-level audit.
