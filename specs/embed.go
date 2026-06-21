// Package specs exposes the committed, SDK-ready OpenAPI specs and the version
// manifest as an embedded filesystem, so the CLI can build its command tree
// without reading the repository at runtime.
package specs

import "embed"

// Build holds the committed specs under build/<app>/<version>/openapi.json.
//
//go:embed build
var Build embed.FS

// VersionsYAML is the raw specs/versions.yaml manifest (pins and defaults).
//
//go:embed versions.yaml
var VersionsYAML []byte
