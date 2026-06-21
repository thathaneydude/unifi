---
title: Protect real-time subscriptions
author: thathaneydude
description: Subscribe to Protect device and event streams over WebSocket.
status: Draft
date: 2026-06-20
tags:
  - guide
  - protect
  - realtime
---

# Protect real-time subscriptions

Protect publishes real-time updates over WebSocket at `/v1/subscribe/devices` and
`/v1/subscribe/events`. The SDK dials these reusing the same auth and TLS as REST.

```go
c := unifi.Local(host, apiKey, unifi.WithInsecureSkipVerify())

stream, err := c.Subscribe(ctx, unifi.AppProtect, "/v1/subscribe/events")
if err != nil { /* ... */ }
defer stream.Close()

for frame := range stream.Frames() {
    // Use any event type from the generated lib/protect/<version> package.
    // CameraMotionEvent is one concrete example; choose the type that matches
    // the event payload your application handles.
    evt, err := unifi.Decode[protectv7_1_46.CameraMotionEvent](frame)
    if err != nil { continue }
    // handle evt
}
```

The decode type comes from whichever `lib/protect/<version>` package you choose, so the same
WebSocket layer serves every coexisting version.
