package cli

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"

	"github.com/thathaneydude/unifi/specs"
	"github.com/thathaneydude/unifi/unifi"
)

// Catalog holds every embedded spec parsed and indexed by app and version.
type Catalog struct {
	docs     map[unifi.App]map[string]*openapi3.T
	defaults map[unifi.App]string
}

type versionsFile struct {
	Apps map[string]struct {
		Default string `yaml:"default"`
	} `yaml:"apps"`
}

// LoadCatalog parses the embedded specs and version manifest.
func LoadCatalog() (*Catalog, error) {
	var vf versionsFile
	if err := yaml.Unmarshal(specs.VersionsYAML, &vf); err != nil {
		return nil, fmt.Errorf("parse versions.yaml: %w", err)
	}

	cat := &Catalog{
		docs:     map[unifi.App]map[string]*openapi3.T{},
		defaults: map[unifi.App]string{},
	}
	for name, a := range vf.Apps {
		cat.defaults[unifi.App(name)] = a.Default
	}

	loader := openapi3.NewLoader()
	walkErr := fs.WalkDir(specs.Build, "build", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || path.Base(p) != "openapi.json" {
			return nil
		}
		// Expected: build/<app>/<version>/openapi.json
		parts := strings.Split(p, "/")
		if len(parts) != 4 {
			return fmt.Errorf("unexpected spec path %q (want build/<app>/<version>/openapi.json)", p)
		}
		app := unifi.App(parts[1])
		version := parts[2]

		data, err := specs.Build.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		doc, err := loader.LoadFromData(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", p, err)
		}
		if cat.docs[app] == nil {
			cat.docs[app] = map[string]*openapi3.T{}
		}
		cat.docs[app][version] = doc
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return cat, nil
}

// Apps returns the apps present in the catalog, sorted for determinism.
func (c *Catalog) Apps() []unifi.App {
	out := make([]unifi.App, 0, len(c.docs))
	for app := range c.docs {
		out = append(out, app)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// DefaultVersion returns the configured default version for an app.
func (c *Catalog) DefaultVersion(app unifi.App) string { return c.defaults[app] }

// Doc returns the parsed spec for an app + version. An empty version resolves
// to the app's default. The resolved version string is returned alongside.
func (c *Catalog) Doc(app unifi.App, version string) (*openapi3.T, string, error) {
	if version == "" {
		def, ok := c.defaults[app]
		if !ok {
			return nil, "", fmt.Errorf("no default version configured for app %q", app)
		}
		version = def
	}
	versions, ok := c.docs[app]
	if !ok {
		return nil, "", fmt.Errorf("unknown app %q", app)
	}
	doc, ok := versions[version]
	if !ok {
		return nil, "", fmt.Errorf("unknown version %q for app %q", version, app)
	}
	return doc, version, nil
}
