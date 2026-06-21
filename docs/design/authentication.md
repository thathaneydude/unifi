---
title: Authentication & Transport
author: thathaneydude
description: API keys, the X-API-KEY header, local vs remote base URLs, and TLS options.
status: Accepted
date: 2026-06-20
tags:
  - design
  - authentication
---

# Authentication & Transport — UniFi

## API key

Both UniFi Network and UniFi Protect integration APIs authenticate with a single header:

```
X-API-KEY: <key>
```

The key is created in each application's **Integrations** section (UniFi OS). A local key and a
remote/Site-Manager key may differ, but both are sent in the same header.

## Two transports

The SDK supports the two access modes UniFi documents. Both differ only by **base URL**; the auth
header and the `/v1/...` operation paths are identical.

### Local (per-console)

```
https://{host}/proxy/{app}/integration/v1/...
```

- `{host}` — the console's address (IP or hostname).
- `{app}` — `network` or `protect`.
- Example:
  ```
  curl -X GET "https://$UNIFI_HOST/proxy/network/integration/v1/info" \
       -H "X-API-KEY: $UNIFI_API_KEY"
  ```

Local consoles typically present a **self-signed TLS certificate**, so the SDK must allow relaxed or
custom TLS verification (see TLS options below).

### Remote (cloud connector)

```
https://api.ui.com/v1/connector/consoles/{consoleId}/{app}/integration/v1/...
```

- `{consoleId}` — the console identifier in the UniFi cloud.
- Example:
  ```
  curl -X GET "https://api.ui.com/v1/connector/consoles/$CONSOLE_ID/network/integration/v1/info" \
       -H "X-API-Key: $UNIFI_SM_KEY"
  ```

`api.ui.com` presents a normal publicly-trusted certificate; no TLS relaxation is needed.

## SDK surface

```go
// Local console (self-signed cert)
c := unifi.Local("192.168.1.1", apiKey, unifi.WithInsecureSkipVerify())

// Remote via cloud connector
c := unifi.Remote(consoleID, apiKey)

// Latest pinned version, generated client + types:
net, err := c.Network()
if err != nil { /* handle */ }
info, err := net.GetInfoWithResponse(ctx)

// A specific coexisting version (e.g. when pinning to an exact release):
old, err := networkv10_3_58.NewClientWithResponses(
    c.NetworkBaseURL(),
    networkv10_3_58.WithRequestEditorFn(c.RequestEditor()),
    networkv10_3_58.WithHTTPClient(c.HTTPClient()),
)
```

## Options

| Option | Effect |
|---|---|
| `WithHTTPClient(*http.Client)` | Use a caller-supplied client (proxies, transports, instrumentation). |
| `WithTimeout(d)` | Per-request timeout on the default client. |
| `WithUserAgent(s)` | Override the `User-Agent`. |
| `WithInsecureSkipVerify()` | Skip TLS verification — for local self-signed consoles only. |
| `WithRootCAs(*x509.CertPool)` | Trust a specific CA (e.g. a pinned console cert). |
| `WithTLSConfig(*tls.Config)` | Full control over TLS. |

`WithInsecureSkipVerify` is documented as **local-only** and emits guidance against use over the
remote transport.

## Request flow

1. `Local`/`Remote` build a `Conn` with the resolved base-URL prefix and a configured `*http.Client`.
2. `Conn.RequestEditor()` returns a `RequestEditorFn` that sets `X-API-KEY` on every outgoing request.
3. Generated clients are constructed with `(baseURL, WithRequestEditorFn(editor), WithHTTPClient(...))`.
4. WebSocket subscriptions reuse the same header + TLS config via `Conn.Subscribe`.

See [ADR-0002](decisions/0002-local-and-remote-transports.md).
