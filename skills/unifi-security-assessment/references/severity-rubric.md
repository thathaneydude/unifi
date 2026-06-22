# Severity Rubric

Apply consistently across all UniFi assessment domains.

- **critical** — Direct external exposure or trivially exploitable weakness.
  Examples: open WAN→LAN any-any firewall permit; an SSID with no encryption;
  management interface reachable from the WAN.
- **high** — Significant weakening of posture, exploitable with little effort or
  adjacency. Examples: guest network without client/network isolation; badly
  outdated device or camera firmware; VPN server on a default port with weak
  configuration.
- **medium** — Hardening gap that increases risk but is not directly
  exploitable. Examples: WPA2 used where WPA3 is available; PMF disabled; flat
  topology with no VLAN segmentation.
- **low** — Minor or best-practice deviation. Examples: never-expiring guest
  vouchers; cosmetic naming/labeling gaps that hinder review.
- **info** — Inventory or context with no action required. Examples: device
  counts, site fingerprint, list of SSIDs.

When unsure between two levels, choose the lower and explain the reasoning in
the finding's remediation text.
