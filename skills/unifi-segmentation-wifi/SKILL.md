---
name: unifi-segmentation-wifi
description: Read-only audit of UniFi network segmentation and WiFi security using the unifi CLI. Flags flat topology, missing guest/IoT isolation, open or weakly encrypted SSIDs, and management-frame-protection gaps. Use as part of a UniFi security assessment or on its own.
version: 0.1.0
---

# UniFi Segmentation & WiFi Audit

This skill is strictly read-only. Use only `Get*` operations, `list-operations`,
and `schema`. Never call `Patch*`/`Post*`/`Put*`/`Delete*`. Report fixes in
prose; never apply them.

## Procedure

### Discover
Run `unifi network list-operations` and consult `unifi network schema`. Absent
operations become "could not assess" notes.

### Collect (read-only)
- Networks/VLANs: `GetNetworksOverviewPage`, `GetNetworkDetails`,
  `GetNetworkReferences`.
- WiFi: `GetWifiBroadcastPage`, `GetWifiBroadcastDetails`.

### Analyze (checklist)
- Segmentation: flat topology (everything one subnet); IoT or guest networks not
  segmented; inter-VLAN reachability that should be blocked.
- WiFi: open / no-encryption SSIDs; WPA2 used where WPA3 is available; weak PSK
  posture; guest SSID without client/network isolation; PMF / management-frame
  protection disabled; reliance on hidden SSIDs as a security control.

### Emit
Return findings using the finding shape in
`../unifi-security-assessment/references/report-template.md`, graded with
`../unifi-security-assessment/references/severity-rubric.md`. Redact PSKs and
other secrets in evidence. End with a one-line domain summary.
