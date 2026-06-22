---
name: unifi-asset-inventory
description: Read-only UniFi asset and client inventory audit using the unifi CLI. Flags outdated firmware, unadopted/rogue devices, unexpected clients, and never-expiring guest vouchers. Use as part of a UniFi security assessment or on its own.
version: 0.1.0
---

# UniFi Asset & Client Inventory

This skill is strictly read-only. Use only `get*` operations, `list-operations`,
and `schema`. Never call any mutating operation (for UniFi Network these use
verbs like `create*`/`update*`/`patch*`/`delete*`/`adopt*`/`remove*`/`execute*`).
Report fixes in prose; never apply them.

## Procedure

### Discover
Run `unifi network list-operations` and consult `unifi network schema`. Absent
operations become "could not assess" notes.

### Collect (read-only)
- Devices: `getAdoptedDeviceOverviewPage`, `getAdoptedDeviceDetails`,
  `getAdoptedDeviceLatestStatistics`, `getPendingDevicePage`.
- Clients: `getConnectedClientOverviewPage`, `getConnectedClientDetails`.
- Guests: `getVouchers`, `getVoucher`.
- Context: `getSiteOverviewPage`, `getInfo`.

### Analyze (checklist)
- Adopted devices: outdated firmware (reported vs latest); update-available
  flags set.
- Pending devices: unadopted / rogue gear present on the network.
- Connected clients: unknown or unexpected clients; clients on an unexpected
  VLAN.
- Vouchers: active or never-expiring guest vouchers.

### Emit
Return findings using the finding shape in
`../unifi-security-assessment/references/report-template.md`, graded with
`../unifi-security-assessment/references/severity-rubric.md`. Redact client
identifiers where sensitive. End with a one-line domain summary.
