---
name: unifi-network-security
description: Read-only audit of UniFi Network firewall policies, zones, ACL rules, DNS policies, VPN, and RADIUS using the unifi CLI. Flags overly permissive rules, WAN exposure, ordering bugs, and missing controls. Use as part of a UniFi security assessment or on its own.
version: 0.1.0
---

# UniFi Network Security Audit

This skill is strictly read-only. Use only `Get*` operations, `list-operations`,
and `schema`. Never call `Patch*`/`Post*`/`Put*`/`Delete*`. Report fixes in
prose; never apply them.

## Procedure

### Discover
Run `unifi network list-operations` and consult `unifi network schema` to learn
which operations this firmware exposes. Any operation below that is absent
becomes a "could not assess" note, not a failure.

### Collect (read-only)
- Firewall: `GetFirewallPolicies`, `GetFirewallPolicy`, `GetFirewallZones`,
  `GetFirewallZone`, `GetFirewallPolicyOrdering`.
- ACL: `GetAclRule`, `GetAclRulePage`, `GetAclRuleOrdering`.
- DNS: `GetDnsPolicy`, `GetDnsPolicyPage`.
- VPN: `GetVpnServerPage`, `GetSiteToSiteVpnTunnelPage`.
- RADIUS: `GetRadiusProfileOverviewPage`.
- Traffic: `GetTrafficMatchingLists`.

### Analyze (checklist)
- Firewall policies: any-any / default-allow permits; WAN→LAN inbound permits;
  management access reachable from WAN; overly broad port/protocol ranges;
  disabled rules implying an intent gap.
- Zones & ordering: zone trust mismatches; allow-before-deny ordering bugs
  (cross-check `GetFirewallPolicyOrdering` and `GetAclRuleOrdering`).
- DNS: no filtering / permissive resolution.
- VPN: servers exposed on default ports or with weak configuration; site-to-site
  tunnels present without the expected crypto.
- RADIUS: presence/absence of a profile for 802.1X. Secrets are usually not
  readable — flag absence, not value.

### Emit
Return findings using the finding shape in
`../unifi-security-assessment/references/report-template.md`, graded with
`../unifi-security-assessment/references/severity-rubric.md`. Redact any keys,
secrets, or tokens in evidence. End with a one-line domain summary.
