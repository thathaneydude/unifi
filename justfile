# UniFi task runner.
# Run `just` (or `just help`) to list available recipes.

# Local docs preview address.
docs_addr := "127.0.0.1:8000"

# Show available recipes.
default:
    @just --list

alias help := default

# --- Docs -------------------------------------------------------------------

# Install the docs toolchain (MkDocs Material). --include-deps exposes the `mkdocs` CLI.
docs-deps:
    pipx install --include-deps mkdocs-material

# Generate Go API reference markdown for the hand-written public package.
docs-reference:
    mkdir -p docs/reference
    go tool gomarkdoc --output docs/reference/unifi.md ./unifi

# Copy the build specs into docs/openapi/ so MkDocs can publish them.
docs-openapi:
    mkdir -p docs/openapi
    cp specs/build/network/v10.3.58/openapi.json docs/openapi/network.openapi.json
    cp specs/build/protect/v7.1.46/openapi.json docs/openapi/protect.openapi.json

# Generate all docs artifacts (reference + OpenAPI specs).
docs-prepare: docs-reference docs-openapi

# Serve the documentation site locally with live reload (default http://127.0.0.1:8000).
docs-serve: docs-prepare
    mkdocs serve --dev-addr {{docs_addr}}

# Build the static documentation site into ./site (strict: fail on warnings).
docs-build: docs-prepare
    mkdocs build --strict

# --- Specs & codegen --------------------------------------------------------

# Pull pinned upstream specs from the mirror and apply overlays -> specs/build.
sync:
    go run ./tools/specgen

# Regenerate clients and fakes (oapi-codegen + counterfeiter) from specs/build.
gen:
    go generate ./...

# --- Build & quality --------------------------------------------------------

# Compile everything.
build:
    go build ./...

# Build the unifi CLI binary into ./bin.
build-cli:
    go build -o bin/unifi ./cmd/unifi

# Tidy modules.
tidy:
    go mod tidy

# Vet and lint.
lint:
    go vet ./...
    go tool golangci-lint run

# Run the Ginkgo unit suites with the race detector.
test:
    go run github.com/onsi/ginkgo/v2/ginkgo -r --race --skip-package e2e

# Run the end-to-end suite (mock servers; real console when creds are set).
test-e2e:
    go run github.com/onsi/ginkgo/v2/ginkgo --tags e2e ./e2e/...

# --- Release ----------------------------------------------------------------

# Prepend unreleased Conventional Commits to CHANGELOG.md (append-only).
# Does NOT regenerate the whole file: the 0.1.0 baseline was carried over from
# the old unifi-sdk repo (squashed history) and cannot be reproduced from git,
# so a full `git cliff --output` would clobber it. At release time, pass the
# tag, e.g. `git cliff --tag v0.2.0 --unreleased --prepend CHANGELOG.md`.
changelog:
    git cliff --unreleased --prepend CHANGELOG.md

# Validate the goreleaser config and run a snapshot release (no publish).
release-snapshot:
    goreleaser release --snapshot --clean
