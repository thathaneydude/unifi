# Changelog

<!--
Baseline note: the 0.1.0 section below was carried over from the original
unifi-sdk repository, whose granular commit history was squashed into a single
commit during migration. It cannot be reproduced by regenerating from git
history, so it is maintained by hand. New entries are prepended above it by
`just changelog` (append-only); never run a full `git cliff --output`, which
would clobber this baseline.
-->

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


### Refactor

- **specgen:** Collision-safe sanitize, deterministic sync, split fixups, add tests


### Testing

- **unifi:** Add counterfeiter HTTPDoer fake and offline client test

- **e2e:** Add build-tagged Ginkgo e2e suite



