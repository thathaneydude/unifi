---
name: unifi-protect-security
description: Read-only UniFi Protect security audit using the unifi CLI. Flags outdated camera/NVR firmware, exposed RTSP/RTSPS streams, and Protect user access scope. Use as part of a UniFi security assessment or on its own. Skip when no Protect NVR is present.
version: 0.1.0
---

# UniFi Protect Security Audit

This skill is strictly read-only. Use only `GetV1*` operations, `list-operations`,
and `schema`. Never call any mutating operation (`PostV1*`/`PutV1*`/`PatchV1*`/`DeleteV1*`).
Report fixes in prose; never apply them.

## Procedure

### Discover
Run `unifi protect list-operations` and consult `unifi protect schema`. If this
fails or `GetV1Nvrs` returns nothing, Protect is absent — report "skipped: no
NVR detected" and stop.

### Collect (read-only)
- Cameras: `GetV1Cameras`, `GetV1CamerasId`, `GetV1CamerasIdRtspsStream`.
- NVR/meta: `GetV1MetaInfo`, `GetV1Nvrs`.
- Users: `GetV1Users`, `GetV1UsersId`, `GetV1UlpUsers`, `GetV1UlpUsersId`.

### Analyze (checklist)
- Cameras: outdated firmware; RTSP/RTSPS streams enabled (exposure surface).
- NVR/meta: version currency.
- Users: Protect and ULP user accounts and their access scope — flag broad or
  unexpected access.

### Emit
Return findings using the finding shape in
`../unifi-security-assessment/references/report-template.md`, graded with
`../unifi-security-assessment/references/severity-rubric.md`. Redact user
identifiers and stream credentials in evidence. End with a one-line domain
summary.
