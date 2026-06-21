---
title: Getting started
author: thathaneydude
description: Install the SDK, create an API key, and make your first request.
status: Draft
date: 2026-06-20
tags:
  - guide
---

# Getting started

## Install

```bash
go get github.com/thathaneydude/unifi@latest
```

## Create an API key

In the UniFi application (Network or Protect), open **Settings → Integrations** and create an API key.
Use it in the `X-API-KEY` header — the SDK does this for you.

## First request (local console)

```go
c := unifi.Local("192.168.1.1", os.Getenv("UNIFI_API_KEY"), unifi.WithInsecureSkipVerify())
net, err := c.Network()
if err != nil {
    log.Fatal(err)
}
info, err := net.GetInfoWithResponse(ctx)
```

## First request (remote)

```go
c := unifi.Remote(os.Getenv("UNIFI_CONSOLE_ID"), os.Getenv("UNIFI_SM_KEY"))
protect, err := c.Protect()
if err != nil {
    log.Fatal(err)
}
cams, err := protect.GetV1CamerasWithResponse(ctx)
```

See also: [local vs remote](local-vs-remote.md) and [Protect realtime](protect-realtime.md).
