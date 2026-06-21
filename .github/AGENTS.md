# Agent guidance — unifi

## Golden rules
- **Never hand-edit anything under `lib/`** — it is generated. Fix issues in
  `specs/overlays/*` or `lib/<app>/<ver>/oapi-codegen.yaml`, then `just gen`.
- **Never commit raw upstream specs.** `just sync` pulls them into the gitignored
  `specs/.cache/`. Only `specs/build/**` and generated `lib/**` are committed.
- Conventional Commits are mandatory (`feat:`, `fix:`, `docs:`, `ci:`, `build:`,
  `test:`, `refactor:`, `chore:`). CI rejects non-conforming messages.
- Tests use Ginkgo/Gomega; fakes come from counterfeiter (`go generate ./...`).
- Tasks run through the `justfile`. Prefer `just <recipe>` over raw commands.
- **The CLI is spec-driven.** `cmd/unifi` + `internal/cli` build their command tree at
  runtime from the embedded `specs/build/**`. Never hand-write per-operation commands; if
  the surface is wrong, fix the spec/overlay or the engine, not a per-command file.
- **Keep docs in sync with the code.** When a change affects behavior, layout, requirements,
  workflow, or public surface, update the relevant docs in the **same** PR — at minimum
  `README.md`, `llms.txt`, `.github/AGENTS.md`, and the affected files under `docs/`
  (`docs/requirements/`, `docs/design/`, `docs/guides/`). Treat stale docs as a bug.

## Layout
- `unifi/` — hand-written core: `Local`/`Remote`, transport, TLS, errors, WebSocket.
  Shared by both the SDK and the CLI.
- `lib/<app>/<appversion>/` — generated clients; app versions coexist. The public Go SDK.
- `cmd/unifi/` — thin entrypoint for the `unifi` CLI binary.
- `internal/cli/` — spec-driven CLI engine (command-tree builder, request mapper, output,
  errors, discovery). Reuses the `unifi/` core for auth/TLS/transport.
- `specs/` — `versions.yaml` pins, `overlays/`, committed `build/`.
- `tools/specgen/` — pulls + augments specs.

## Before opening a PR
Run `just sync && just gen && just lint && just test` and ensure no diff
(`git status` clean) — CI enforces this drift guard.
