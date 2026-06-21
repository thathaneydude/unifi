---
title: Local vs remote access
author: thathaneydude
description: Choosing between per-console (local) and cloud-connector (remote) transports.
status: Draft
date: 2026-06-20
tags:
  - guide
  - authentication
---

# Local vs remote access

The SDK reaches the same API two ways. See [authentication](../design/authentication.md) for the full
design.

| | Local | Remote |
|---|---|---|
| Constructor | `unifi.Local(host, apiKey)` | `unifi.Remote(consoleID, apiKey)` |
| Base URL | `https://{host}/proxy/{app}/integration` | `https://api.ui.com/v1/connector/consoles/{consoleId}/{app}/integration` |
| TLS | Often self-signed → `WithInsecureSkipVerify`/`WithRootCAs` | Publicly trusted |
| Key | App Integrations key | Site-Manager key |

## Choosing a specific version

The root `c.Network()` / `c.Protect()` methods return the latest pinned version
(`networkv10_3_58` / `protectv7_1_46`). When you need to target a different pinned version
(for example, if additional versions were added under `lib/network/<version>`), construct
the generated client directly by passing the connection primitives:

```go
// Import: github.com/thathaneydude/unifi/lib/network/v10_3_58
client, err := networkv10_3_58.NewClientWithResponses(
    c.NetworkBaseURL(),
    networkv10_3_58.WithRequestEditorFn(c.RequestEditor()),
    networkv10_3_58.WithHTTPClient(c.HTTPClient()),
)
if err != nil {
    log.Fatal(err)
}
```
