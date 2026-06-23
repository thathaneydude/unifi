# unifi

**Module:** `github.com/thathaneydude/unifi` | **Go:** 1.26

A SemVer-versioned Go SDK **and CLI** for the official [UniFi Network](https://ui.com/consoles) and
[UniFi Protect](https://ui.com/camera-security) integration APIs. Clients are generated from
OpenAPI specs with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen), wrapped by a
thin hand-written layer that handles local and remote authentication, TLS options, and a
version-agnostic WebSocket layer for Protect real-time subscriptions. The same core powers a
spec-driven `unifi` **CLI** designed for LLM agents (JSON output, structured errors, runtime
discovery).

- **Network:** v10.3.58
- **Protect:** v7.1.46
- **Docs:** <https://thathaneydude.github.io/unifi/>

## Installation

### CLI

Download a prebuilt binary for your platform (macOS, Linux, Windows — amd64/arm64) from the
[latest release](https://github.com/thathaneydude/unifi/releases/latest), extract it, and put the
`unifi` binary on your `PATH`. Or install with Go:

```
go install github.com/thathaneydude/unifi/cmd/unifi@latest
```

### SDK

```
go get github.com/thathaneydude/unifi@latest
```

## Usage

### Local (on-premises console)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/thathaneydude/unifi/unifi"
)

func main() {
    // Connect directly to a console on the LAN.
    // WithInsecureSkipVerify skips TLS verification for self-signed console certs.
    c := unifi.Local("192.168.1.1", "your-api-key", unifi.WithInsecureSkipVerify())

    network, err := c.Network()
    if err != nil {
        log.Fatal(err)
    }

    resp, err := network.GetInfoWithResponse(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.StatusCode())
}
```

### Remote (UniFi cloud connector)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/thathaneydude/unifi/unifi"
)

func main() {
    // Connect through the UniFi cloud connector using a console ID + API key.
    c := unifi.Remote("your-console-id", "your-api-key")

    protect, err := c.Protect()
    if err != nil {
        log.Fatal(err)
    }

    resp, err := protect.GetV1CamerasWithResponse(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(resp.StatusCode())
}
```

## CLI (for LLM agents)

> Status: in active development on `main`.

The `unifi` CLI exposes every Network and Protect REST operation, generated at runtime from the
same OpenAPI specs that drive the SDK. Its primary user is an LLM agent that shells out to it,
so it defaults to machine-readable behavior:

- **JSON by default** on stdout (`--format human` for a readable view, `--format raw` for passthrough).
- **Structured errors** on stderr with stable exit codes (`0` ok, `1` usage, `2` auth, `3` API
  non-2xx, `4` transport).
- **Discovery** built in: `unifi schema` and `unifi <app> list-operations`.
- **Safe writes:** mutating operations support `--dry-run` and require `--confirm`.

```sh
# Credentials via env (flags --host/--console-id/--api-key override)
export UNIFI_API_KEY=your-api-key
export UNIFI_HOST=192.168.1.1        # or UNIFI_CONSOLE_ID for remote

unifi protect list-operations             # discover the surface
unifi protect GetV1Cameras                # call an operation → JSON
unifi network getInfo --format human      # human-readable view
```

Network and Protect mint **separate** integration API keys, so over the local transport each app
needs its own key. Set both to drive both apps from one configuration:

```sh
export UNIFI_NETWORK_API_KEY=your-network-key   # or --network-api-key
export UNIFI_PROTECT_API_KEY=your-protect-key   # or --protect-api-key

unifi network getInfo         # uses the Network key
unifi protect GetV1Cameras    # uses the Protect key
```

Each app falls back to the shared `UNIFI_API_KEY` / `--api-key` when its app-specific key is unset,
so a single key still works for whichever app you target. (Over the remote transport, one
account-level Site-Manager key reaches both apps.)

The CLI also auto-loads a `.env` file from the working directory if present (use `--env-file <path>`
for a custom location). Real environment variables and flags take precedence, so the resolution order
is **flags > environment > `.env`**:

```sh
# .env in the current directory
UNIFI_API_KEY=your-api-key
UNIFI_HOST=192.168.1.1

unifi network getInfo            # picks up .env automatically
unifi network getInfo --env-file ./prod.env
```

### Ergonomics

Global flags reduce the shell glue agents otherwise need:

```sh
# Site resolution — most Network ops need a siteId. Omit it and the CLI uses the
# only site, or pass a name / "default" / id via --site (or UNIFI_SITE).
unifi network getNetworksOverviewPage                 # auto-selects the sole site
unifi network getNetworksOverviewPage --site default

# Terse discovery — no JSON parsing to find operation ids.
unifi network list-operations --ids                   # bare operation ids
unifi network list-operations --format human          # aligned table + required params

# Response shaping (applied to JSON; --format raw stays verbatim).
unifi network getNetworksOverviewPage --fields name,vlanId   # keep dot-path record fields
unifi network getNetworksOverviewPage --limit 5              # cap a result (or .data) array
unifi network getVpnServerPage --redact                      # mask secret-like fields as ***
```

Required parameters (path and query) are validated up front, so an operation like
`getFirewallPolicyOrdering` reports its missing zone-id flags instead of failing at the API.

Realtime Protect subscriptions are intentionally **SDK-only** (see below), not exposed by the CLI.

## Security assessment skills

The [`skills/`](skills/) directory ships read-only [Agent Skills](https://docs.claude.com/en/docs/claude-code/skills)
that drive the CLI to audit a UniFi deployment and produce a severity-ranked
findings report. The `unifi-security-assessment` orchestrator runs four focused
domain skills (`unifi-network-security`, `unifi-segmentation-wifi`,
`unifi-asset-inventory`, `unifi-protect-security`) as parallel subagents. They
never mutate configuration — only read-only operations are used, enforced by
`just validate-skills`.

See [`skills/README.md`](skills/README.md) for installation and usage. In short:

```sh
# Install with the skills CLI (https://github.com/vercel-labs/skills)
npx skills add thathaneydude/unifi -g -a claude-code

# Then, with the unifi CLI on PATH and credentials set, ask your agent:
#   "Run a UniFi security assessment of my deployment."
# → writes ./unifi-assessment-YYYY-MM-DD.md
```

## Documentation

Full documentation is published at <https://thathaneydude.github.io/unifi/>, including:

- [Architecture](docs/design/architecture.md) and [authentication](docs/design/authentication.md) design
- [Getting started guide](docs/guides/getting-started.md)
- [Go API reference](docs/reference/index.md) (generated by `just docs-reference`)
- [Network OpenAPI viewer](docs/openapi/network.md) and [Protect OpenAPI viewer](docs/openapi/protect.md)
- [Decision records](docs/design/decisions/index.md)

## Development

Prerequisites: Go 1.26, [just](https://github.com/casey/just).

| Recipe | Description |
|---|---|
| `just sync` | Pull pinned upstream specs and apply overlays → `specs/build/` |
| `just gen` | Regenerate clients and fakes from `specs/build/` |
| `just build` | Compile all packages |
| `just lint` | `go vet` + golangci-lint |
| `just test` | Run unit suites with the race detector |
| `just test-e2e` | Run end-to-end suite (mock servers; real console when creds are set) |
| `just docs-serve` | Live-preview the docs site at <http://127.0.0.1:8000> |
| `just docs-build` | Build the static docs site into `./site/` |

Run `just` with no arguments to list all available recipes.

## License

This project is licensed under the [MIT License](LICENSE).
