package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// Validate loads a JSON/YAML OpenAPI 3.x spec and runs full kin-openapi
// validation (doc.Validate).  Invalid enum-example values are stripped by
// Augment before this is called, so the upstream spec quirks do not surface
// here.
func Validate(spec []byte) error {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(spec)
	if err != nil {
		return fmt.Errorf("load spec: %w", err)
	}
	return doc.Validate(loader.Context)
}

// Sync fetches, augments, validates, and writes all specs described in cfg.
// It resolves all paths relative to the repo root (found via repoRoot).
// Apps are processed in sorted order for deterministic error messages.
func Sync(cfg *Config) error {
	root, err := repoRoot()
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	mirror := strings.TrimSuffix(cfg.Mirror, "/")

	// Sort app names so iteration order is deterministic (map range is random).
	apps := make([]string, 0, len(cfg.Apps))
	for app := range cfg.Apps {
		apps = append(apps, app)
	}
	sort.Strings(apps)

	for _, app := range apps {
		appCfg := cfg.Apps[app]
		for _, ver := range appCfg.Versions {
			if err := syncOne(root, mirror, app, ver); err != nil {
				return err
			}
		}
	}

	return nil
}

func syncOne(root, mirror, app, ver string) error {
	// TODO(specgen): Replace the dependency on the opastorello/unifi-api-docs mirror with a
	// native Go implementation that pulls the spec directly from the official UniFi source
	// (developer.ui.com / its underlying data endpoints) and performs the OpenAPI conversion
	// in-process, removing the external mirror dependency. See ADR-0003 "Future work".
	url := fmt.Sprintf("%s/%s/%s/openapi.json", mirror, app, ver)

	// Fetch.
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck // body close error is not actionable in a defer

	if resp.StatusCode != http.StatusOK {
		// Drain the body so the underlying TCP connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body %s: %w", url, err)
	}

	// Write raw to cache.
	cacheDir := filepath.Join(root, "specs", ".cache", app, ver)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("mkdir cache %s: %w", cacheDir, err)
	}
	cachePath := filepath.Join(cacheDir, "openapi.json")
	if err := os.WriteFile(cachePath, raw, 0o644); err != nil {
		return fmt.Errorf("write cache %s: %w", cachePath, err)
	}

	// Augment.
	augmented, err := Augment(app, ver, raw)
	if err != nil {
		return fmt.Errorf("augment %s/%s: %w", app, ver, err)
	}

	// Validate.
	if err := Validate(augmented); err != nil {
		return fmt.Errorf("validate %s/%s: %w", app, ver, err)
	}

	// Write augmented to build dir.
	buildDir := filepath.Join(root, "specs", "build", app, ver)
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return fmt.Errorf("mkdir build %s: %w", buildDir, err)
	}
	buildPath := filepath.Join(buildDir, "openapi.json")
	// Trailing newline for clean diffs.
	output := append(augmented, '\n')
	if err := os.WriteFile(buildPath, output, 0o644); err != nil {
		return fmt.Errorf("write build %s: %w", buildPath, err)
	}

	fmt.Printf("wrote %s\n", buildPath)
	return nil
}
