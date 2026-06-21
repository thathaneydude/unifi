package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Augment takes a raw JSON OpenAPI spec and applies:
//  1. The common overlay (security schemes, servers)
//  2. The per-app overlay (app-specific server defaults)
//  3. Tag synthesis for operations that have no tags
//  4. info.version pinning (leading 'v' stripped)
//
// It returns deterministic, indented JSON.
func Augment(app, version string, raw []byte) ([]byte, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, fmt.Errorf("augment: %w", err)
	}

	overlaysDir := filepath.Join(root, "specs", "overlays")

	// Parse the raw JSON into a yaml.Node tree.
	// yaml.v3 happily parses JSON (JSON is a subset of YAML).
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("augment: unmarshal spec: %w", err)
	}

	// Apply common overlay programmatically.
	if err := applyOverlayFileProgrammatic(filepath.Join(overlaysDir, "common.overlay.yaml"), &doc); err != nil {
		return nil, fmt.Errorf("augment: common overlay: %w", err)
	}

	// Apply per-app overlay programmatically.
	appOverlayFile := filepath.Join(overlaysDir, app+".overlay.yaml")
	if _, statErr := os.Stat(appOverlayFile); statErr == nil {
		if err := applyOverlayFileProgrammatic(appOverlayFile, &doc); err != nil {
			return nil, fmt.Errorf("augment: %s overlay: %w", app, err)
		}
	}

	// Work with the document content node for modifications.
	content := docContent(&doc)
	if content == nil {
		return nil, fmt.Errorf("augment: empty document")
	}

	// Pin info.version (strip leading 'v').
	ver := strings.TrimPrefix(version, "v")
	if err := setStringField(content, "info", "version", ver); err != nil {
		return nil, fmt.Errorf("augment: pin version: %w", err)
	}

	// Synthesize tags for operations lacking them.
	if err := synthesizeTags(content); err != nil {
		return nil, fmt.Errorf("augment: synthesize tags: %w", err)
	}

	// Ensure all responses have a description (required by OpenAPI 3.x).
	ensureResponseDescriptions(content)

	// Strip invalid enum examples (e.g. "ALLOW|BLOCK" on enum:["ALLOW","BLOCK"])
	// so that the full kin-openapi doc.Validate passes.
	stripInvalidEnumExamples(content)

	// Strip examples whose type does not match the schema's declared type
	// (e.g. example: "786435|720973" on type: integer).
	stripInvalidTypeExamples(content)

	// Remove redundant discriminators from allOf schemas where multiple items
	// all declare the same discriminator.propertyName.  oapi-codegen v2 cannot
	// merge two schemas that both carry a discriminator; keeping only the first
	// occurrence is semantically equivalent because they all reference the same
	// property (e.g. "origin" in the upstream Network spec).
	deduplicateAllOfDiscriminators(content)

	// Remove info.license when it lacks a required "name" field (upstream
	// Network spec has "license": {} which kin-openapi rejects).
	stripEmptyLicense(content)

	// Marshal back to JSON via an intermediate map (deterministic key order via
	// yaml→map→json). json.MarshalIndent sorts map keys alphabetically, which
	// guarantees output stability across runs.
	var anyDoc any
	if err := doc.Decode(&anyDoc); err != nil {
		return nil, fmt.Errorf("augment: decode to map: %w", err)
	}

	// Sanitize component schema identifiers that contain characters not
	// permitted by the OpenAPI spec (only [a-zA-Z0-9._-] is allowed).
	// Some upstream specs use spaces and special chars in schema names.
	// We replace every offending character with "_" and update all $ref
	// strings consistently so the document remains self-consistent.
	if m, ok := anyDoc.(map[string]any); ok {
		if err := sanitizeSchemaIdentifiers(m); err != nil {
			return nil, fmt.Errorf("augment: sanitize schema identifiers: %w", err)
		}
	}

	// Normalize OpenAPI 3.1 nullable type arrays (["T","null"]) to oapi-codegen
	// compatible form: scalar "type": "T" + "nullable": true.  oapi-codegen v2
	// does not support multi-value type arrays and will error with
	// "unhandled Schema type: &[T null]".
	normalizeNullableTypeArrays(anyDoc)

	// Collapse OpenAPI 3.1 oneOf-nullable patterns ({ oneOf: [S, {type:"null"}] })
	// to oapi-codegen-compatible nullable schemas (S + nullable:true).
	// oapi-codegen v2 errors with "unhandled Schema type: &[null]" when it
	// encounters {type:"null"} as a oneOf member.
	anyDoc = collapseOneOfNullable(anyDoc)

	// Replace inline schema objects in discriminated oneOf/anyOf arrays with
	// $ref pointers.  oapi-codegen v2 requires $refs (not inline objects) in
	// discriminated oneOf/anyOf and errors with "ambiguous discriminator.mapping:
	// please replace inlined object with $ref" otherwise.
	if m, ok := anyDoc.(map[string]any); ok {
		allSchemas := map[string]any{}
		if comps, ok := m["components"].(map[string]any); ok {
			if schemas, ok := comps["schemas"].(map[string]any); ok {
				allSchemas = schemas
			}
		}
		inlineDiscriminatedOneOfToRefs(anyDoc, allSchemas)
	}

	// Replace inline response body schemas with $refs when they are
	// byte-for-byte identical to a named component schema.  This prevents
	// oapi-codegen from generating duplicate Go type names when the same inline
	// schema appears across multiple response bodies.
	if m, ok := anyDoc.(map[string]any); ok {
		deduplicateResponseSchemas(m)
	}

	// Collapse anyOf parameter schemas of the form [array-of-X, X] to just
	// array-of-X.  oapi-codegen generates conflicting type names (redeclared /
	// recursive) when both the array and scalar forms of the same enum appear
	// in the same anyOf.
	if m, ok := anyDoc.(map[string]any); ok {
		collapseArrayScalarAnyOf(m)
	}

	out, err := json.MarshalIndent(anyDoc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("augment: marshal JSON: %w", err)
	}

	return out, nil
}

// overlayFile is the YAML structure of an overlay file.
type overlayFile struct {
	Actions []overlayAction `yaml:"actions"`
}

type overlayAction struct {
	Target string    `yaml:"target"`
	Update yaml.Node `yaml:"update"`
}

// applyOverlayFileProgrammatic reads an overlay YAML file and applies each
// action to the document by merging the update value into the targeted node.
// If the targeted path does not exist, it is created.
func applyOverlayFileProgrammatic(path string, doc *yaml.Node) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read overlay %s: %w", path, err)
	}

	var ov overlayFile
	if err := yaml.Unmarshal(data, &ov); err != nil {
		return fmt.Errorf("parse overlay %s: %w", path, err)
	}

	content := docContent(doc)
	if content == nil {
		return fmt.Errorf("empty document")
	}

	for _, action := range ov.Actions {
		if err := applyActionProgrammatic(content, action.Target, &action.Update); err != nil {
			return fmt.Errorf("action target %q: %w", action.Target, err)
		}
	}

	return nil
}

// applyActionProgrammatic applies a single overlay action by navigating the
// JSONPath-like target and merging/replacing the update value.
// Supported target forms:
//   - $.key                         → top-level key
//   - $.key1.key2                   → nested key
//   - $.key1[*].key2               → all sequence elements' key2
func applyActionProgrammatic(content *yaml.Node, target string, update *yaml.Node) error {
	// Normalise: strip leading "$." or "$"
	path := strings.TrimPrefix(target, "$.")
	path = strings.TrimPrefix(path, "$")

	segments := parseTargetPath(path)
	if len(segments) == 0 {
		return nil
	}

	return applySegments(content, segments, update)
}

type pathSegment struct {
	key     string
	wildSeq bool // true for [*]
}

func parseTargetPath(path string) []pathSegment {
	var segments []pathSegment
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			continue
		}
		if strings.HasSuffix(part, "[*]") {
			key := strings.TrimSuffix(part, "[*]")
			segments = append(segments, pathSegment{key: key})
			segments = append(segments, pathSegment{wildSeq: true})
		} else {
			segments = append(segments, pathSegment{key: part})
		}
	}
	return segments
}

func applySegments(node *yaml.Node, segments []pathSegment, update *yaml.Node) error {
	if len(segments) == 0 {
		mergeYAMLNodes(node, update)
		return nil
	}

	seg := segments[0]
	rest := segments[1:]

	if seg.wildSeq {
		// Apply to all children of a sequence node.
		if node.Kind != yaml.SequenceNode {
			return nil
		}
		for _, child := range node.Content {
			if err := applySegments(child, rest, update); err != nil {
				return err
			}
		}
		return nil
	}

	// Navigate into a mapping node by key.
	if node.Kind != yaml.MappingNode {
		return nil
	}

	// Find existing key.
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == seg.key {
			return applySegments(node.Content[i+1], rest, update)
		}
	}

	// Key not found: create it.
	if len(rest) == 0 {
		// Leaf: set the value directly.
		keyNode := scalarNode(seg.key)
		valNode := cloneYAMLNode(update)
		node.Content = append(node.Content, keyNode, valNode)
		return nil
	}

	// Intermediate key: create empty mapping or sequence and recurse.
	keyNode := scalarNode(seg.key)
	var childNode *yaml.Node
	if len(rest) > 0 && rest[0].wildSeq {
		childNode = &yaml.Node{Kind: yaml.SequenceNode}
	} else {
		childNode = &yaml.Node{Kind: yaml.MappingNode}
	}
	node.Content = append(node.Content, keyNode, childNode)
	return applySegments(childNode, rest, update)
}

// mergeYAMLNodes merges src into dst. For mappings, keys are added/updated.
// For sequences and scalars, the value is replaced.
func mergeYAMLNodes(dst, src *yaml.Node) {
	if dst.Kind != src.Kind {
		*dst = *cloneYAMLNode(src)
		return
	}

	switch dst.Kind {
	case yaml.MappingNode:
		mergeMappingNodes(dst, src)
	default:
		// Scalar, sequence: replace entirely.
		*dst = *cloneYAMLNode(src)
	}
}

func mergeMappingNodes(dst, src *yaml.Node) {
nextKey:
	for i := 0; i+1 < len(src.Content); i += 2 {
		srcKey := src.Content[i].Value
		srcVal := src.Content[i+1]

		for j := 0; j+1 < len(dst.Content); j += 2 {
			if dst.Content[j].Value == srcKey {
				mergeYAMLNodes(dst.Content[j+1], srcVal)
				continue nextKey
			}
		}
		// Key not found: append.
		dst.Content = append(dst.Content, cloneYAMLNode(src.Content[i]), cloneYAMLNode(srcVal))
	}
}

func cloneYAMLNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	c := &yaml.Node{
		Kind:        n.Kind,
		Style:       n.Style,
		Tag:         n.Tag,
		Value:       n.Value,
		Anchor:      n.Anchor,
		HeadComment: n.HeadComment,
		LineComment:  n.LineComment,
		FootComment:  n.FootComment,
	}
	if n.Alias != nil {
		c.Alias = cloneYAMLNode(n.Alias)
	}
	if n.Content != nil {
		c.Content = make([]*yaml.Node, len(n.Content))
		for i, child := range n.Content {
			c.Content[i] = cloneYAMLNode(child)
		}
	}
	return c
}

func scalarNode(val string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"}
}

// docContent returns the mapping node that is the document body.
// yaml.v3 wraps documents in a DocumentNode whose single child is the content.
func docContent(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) == 1 {
		return doc.Content[0]
	}
	return doc
}

// setStringField sets mapping[parentKey][childKey] = value in a yaml MappingNode.
// If parentKey or childKey does not exist, it is created so that the value is
// always set (e.g. info.version is always pinned even when absent upstream).
func setStringField(mapping *yaml.Node, parentKey, childKey, value string) error {
	parentNode := mappingValue(mapping, parentKey)
	if parentNode == nil {
		// Create the parent mapping node.
		keyNode := scalarNode(parentKey)
		parentNode = &yaml.Node{Kind: yaml.MappingNode}
		mapping.Content = append(mapping.Content, keyNode, parentNode)
	}
	childNode := mappingValue(parentNode, childKey)
	if childNode == nil {
		// Create the child scalar node.
		keyNode := scalarNode(childKey)
		childNode = scalarNode("")
		parentNode.Content = append(parentNode.Content, keyNode, childNode)
	}
	childNode.Value = value
	return nil
}

// mappingValue returns the value node for the given key in a yaml MappingNode.
func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}
