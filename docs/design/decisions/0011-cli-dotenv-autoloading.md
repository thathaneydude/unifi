---
title: "ADR-0011: CLI auto-loads a .env file with a zero-dependency parser"
author: thathaneydude
description: The unifi CLI loads ./.env (or --env-file) into the environment, real env always winning.
status: Accepted
date: 2026-06-21
tags:
  - adr
  - cli
  - config
---

# ADR-0011: CLI auto-loads a .env file with a zero-dependency parser

## Context

The `unifi` CLI resolves its connection config from environment variables — `UNIFI_API_KEY`,
`UNIFI_HOST`, `UNIFI_CONSOLE_ID`, `UNIFI_INSECURE` — read via `os.Getenv` in
`internal/cli/config.go`. Flags (`--api-key`, `--host`, …) override those. There is no support for a
`.env` file, so local and agent-driven use requires exporting variables into the shell by hand, which
is awkward and tends to leak secrets into shell history.

We want ergonomic `.env` support without changing the precedence guarantees, without surprising CI
or exported-shell workflows, and without pulling in a new dependency for what is a small parse job.

## Decision

Auto-load a `.env` file with a **small hand-written, zero-dependency parser** in `internal/cli`
(`dotenv.go`), rather than adding `github.com/joho/godotenv`.

- **Discovery:** load `./.env` from the working directory if present (silent no-op when absent). A
  `--env-file <path>` persistent flag overrides the path; a missing file named explicitly via
  `--env-file` is a usage error.
- **Precedence — real env always wins.** Parsed pairs are applied via `os.Setenv` only for keys not
  already set (`os.LookupEnv`). The full chain is **flags > real env > `.env`**.
- **Scope:** loading happens while resolving a connection, so credential-free discovery commands
  (`schema`, `list-operations`) neither require nor fail on a `.env`/`--env-file`.
- **Grammar:** blank lines and `#` comment lines are skipped; an optional leading `export ` is
  stripped; the line splits on the first `=`; keys must match `[A-Za-z_][A-Za-z0-9_]*`. Values
  wrapped in matching single or double quotes keep their inner text literally; unquoted values are
  whitespace-trimmed. There is **no** inline-comment stripping and **no** escape expansion — the
  grammar stays small and predictable. Malformed lines are usage errors that cite the line number.

`.env` is already listed in `.gitignore`, so secrets stay out of version control.

## Consequences

- No new module dependency; we own a small, well-tested parser instead.
- Local and LLM-agent workflows become ergonomic — drop secrets in `.env` and run — without exposing
  them in shell history.
- CI and exported shell variables remain authoritative; `.env` only fills gaps, so it cannot silently
  mask an intentionally exported value.
- We accept ownership of parser edge cases (quoting, whitespace). The deliberately minimal grammar
  (no inline comments, no escapes) bounds that surface and is documented for users.
