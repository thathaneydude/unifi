# Report Schema

The orchestrator emits findings + metadata as a JSON document and renders it to
a self-contained HTML report with `unifi report --in findings.json --out
./unifi-assessment-YYYY-MM-DD.html`. The renderer groups findings by severity,
pretty-prints evidence, and HTML-escapes all text (no markdown parsing).

## Finding shape

Every finding is an object with these fields:

- `severity` — one of `critical | high | medium | low | info` (see
  severity-rubric.md). These keys are the stable data contract; the HTML report
  renders them as action-oriented labels for UniFi admins — `critical`→"Act Now",
  `high`→"Address Soon", `medium`→"Recommended", `low`→"Optional",
  `info`→"Informational" — with the canonical severity shown as a badge tooltip.
- `title` — short description of the issue.
- `affected_resource` — the specific network / SSID / device / rule / camera.
- `evidence` — the CLI JSON snippet that proves the finding (an object/array),
  with sensitive values (keys, secrets, PSKs, tokens) redacted as `***`.
- `remediation` — prose guidance on how to fix. Never applied automatically.

## findings.json schema

Field names map 1:1 to the `unifi report` renderer. `evidence` and each
appendix `data` value are raw JSON (objects or arrays), not strings.

```json
{
  "date": "YYYY-MM-DD",
  "console": {
    "name": "<console name>",
    "network_version": "<e.g. v10.4.57>",
    "protect_version": "<version or \"absent\">"
  },
  "site_count": 1,
  "skill_versions": {
    "orchestrator": "0.2.0",
    "unifi-network-security": "0.1.0",
    "unifi-segmentation-wifi": "0.1.0",
    "unifi-asset-inventory": "0.1.0",
    "unifi-protect-security": "0.1.0"
  },
  "assessed_by": { "model_name": "Claude Opus 4.8", "model_id": "claude-opus-4-8" },
  "counts": { "critical": 0, "high": 0, "medium": 0, "low": 0, "info": 0 },
  "top_risks": [ "plain-language risk", "plain-language risk", "plain-language risk" ],
  "findings": [
    {
      "severity": "critical",
      "title": "<short issue>",
      "affected_resource": "<network / SSID / device / rule / camera>",
      "evidence": { "redactedKey": "***" },
      "remediation": "<prose fix guidance>"
    }
  ],
  "coverage": {
    "domains_run": [ "unifi-network-security", "unifi-segmentation-wifi", "unifi-asset-inventory" ],
    "skipped": [ "unifi-protect-security — no NVR detected" ],
    "not_assessable": [ "ops that returned not-found or were absent on this firmware" ]
  },
  "appendix": [
    { "domain": "unifi-network-security", "data": { "firewallPolicies": 12 } }
  ]
}
```

Notes:
- Record `assessed_by` (the AI model + id that ran this assessment) and every
  `skill_versions` entry so a newer model can re-evaluate and diff.
- Empty severity buckets render as "None." automatically — omit them from
  `findings`, do not add placeholder entries.
- `appendix[].data` holds the raw JSON captured per domain (secrets redacted).
- The HTML report covers the same sections as before: Executive Summary →
  Findings by severity → Coverage & Limitations → Appendix.
