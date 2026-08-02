# Changelog

<!--
Baseline note: the 0.1.0 section below was carried over from the original
unifi-sdk repository, whose granular commit history was squashed into a single
commit during migration. It cannot be reproduced by regenerating from git
history, so it is maintained by hand. New entries are prepended above it by
`just changelog` (append-only); never run a full `git cliff --output`, which
would clobber this baseline.
-->

## [0.4.0] - 2026-08-01

### Features

- **renovate:** Hold code-generator bumps to reviewed PRs (#28)


## [0.3.6] - 2026-08-01

### Build & CI

- **deps:** Update dependency python to 3.14 (#17)


### Documentation

- **ci:** Record why release-on-main stays owner-only; fix stale repo claims (#26)


## [0.3.5] - 2026-08-01

### Bug Fixes

- **release:** Stop discarding the git-cliff release notes (#21)


## [0.3.4] - 2026-08-01

### Bug Fixes

- **release:** Group dependency bumps under Build & CI regardless of type (#20)


## [0.3.3] - 2026-08-01

### Build & CI

- **deps:** Update module github.com/onsi/gomega to v1.42.1 (#16)

- **deps:** Force the build(deps) commit type past config:recommended (#19)


## [0.3.2] - 2026-08-01

### Build & CI

- Onboard the repo with Renovate (#14)


## [0.3.1] - 2026-08-01

### Bug Fixes

- **specs:** Pin the upstream mirror to an immutable commit (#15)


## [0.3.0] - 2026-06-23

### Features

- **cli:** Enumerate consoles and target by name for remote accounts (#13)


## [0.2.0] - 2026-06-23

### Build & CI

- Auto-release on main via git-cliff version bump (#12)


### Documentation

- **readme:** Add CLI install options for the v0.1.0 release (#6)


### Features

- **skills:** Read-only UniFi security-assessment Agent Skills (#7)

- **cli:** Resolve API keys per app (network/protect) with shared fallback (#8)

- **cli:** Ergonomics — site resolution, field selection, redaction, terse discovery (#10)

- **report:** Self-contained UniFi-branded HTML assessment report (#11)



## [0.1.0] - 2026-06-21

### Bug Fixes

- **specgen:** Full OpenAPI validation via enum-example stripping; tidy deps; add tests

- **specgen:** Rewrite discriminator.mapping refs during schema sanitization

- **unifi:** Clone TLS config, fix WebSocket cancel/err contract, docs, tests


### Build & CI

- Add goreleaser config and release workflow enforcing SemVer tags

- Add justfile task runner with local docs preview

- Initialize go module on go 1.26

- Pin oapi-codegen as a go tool

- Add golangci-lint configuration and wire just lint


### Documentation

- Add functional requirements, design docs, ADRs, llms.txt, and mkdocs config

- **plan:** Add task-by-task SDK implementation plan

- Refine docs-site config and diagrams

- Add .github/AGENTS.md agent guidance

- **adr:** Note future work to drop the external spec mirror

- **unifi:** Add package documentation

- **examples:** Add local and remote runnable examples

- **adr:** Add ADR-0011 for CLI .env auto-loading


### Features

- **specs:** Pin network v10.3.58 and protect v7.1.46

- **specs:** Add common/network/protect OpenAPI overlays

- **specgen:** Load versions.yaml config

- **specgen:** Augment specs with security, servers, and tags

- **specgen:** Pull, validate, and write augmented build specs

- **network:** Generate v10.3.58 client

- **protect:** Generate v7.1.46 client

- **unifi:** Options, TLS config, and Local/Remote constructors

- **unifi:** Base URL builders and X-API-KEY request editor

- **unifi:** Typed APIError with envelope parsing

- **unifi:** Latest-version Network and Protect accessors

- **unifi:** Version-agnostic WebSocket subscribe and Decode

- **cli:** Auto-load `.env` and add `--env-file` flag (flags > env > .env)


### Refactor

- **specgen:** Collision-safe sanitize, deterministic sync, split fixups, add tests


### Testing

- **unifi:** Add counterfeiter HTTPDoer fake and offline client test

- **e2e:** Add build-tagged Ginkgo e2e suite



