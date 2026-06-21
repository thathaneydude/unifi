package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// identifierInvalidCharRE matches any character not allowed in an OpenAPI
// component name (allowed: [a-zA-Z0-9._-]).
var identifierInvalidCharRE = regexp.MustCompile(`[^a-zA-Z0-9._\-]`)

// synthesizeTags walks all paths and adds a synthesized tag to any operation
// that has no "tags" field. The tag is derived from the first non-empty, non-"v1",
// non-parameter path segment (dashes replaced by spaces, first letter uppercased).
func synthesizeTags(doc *yaml.Node) error {
	pathsNode := mappingValue(doc, "paths")
	if pathsNode == nil || pathsNode.Kind != yaml.MappingNode {
		return nil
	}

	// Hoisted out of the per-path loop so the set is built only once.
	httpMethods := map[string]bool{
		"get": true, "put": true, "post": true, "delete": true,
		"options": true, "head": true, "patch": true, "trace": true,
	}

	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		pathKey := pathsNode.Content[i].Value // e.g. "/v1/cameras/{id}"
		pathItem := pathsNode.Content[i+1]   // mapping of method → operation
		tag := tagFromPath(pathKey)
		if tag == "" {
			continue
		}

		if pathItem.Kind != yaml.MappingNode {
			continue
		}

		for j := 0; j+1 < len(pathItem.Content); j += 2 {
			methodKey := strings.ToLower(pathItem.Content[j].Value)
			if !httpMethods[methodKey] {
				continue
			}

			opNode := pathItem.Content[j+1]
			if opNode.Kind != yaml.MappingNode {
				continue
			}

			// Only add tag if operation has no "tags" field.
			if mappingValue(opNode, "tags") != nil {
				continue
			}

			// Append tags: [tag] to the operation.
			tagsKeyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "tags", Tag: "!!str"}
			tagValNode := &yaml.Node{Kind: yaml.ScalarNode, Value: tag, Tag: "!!str"}
			tagsSeqNode := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{tagValNode}}
			opNode.Content = append(opNode.Content, tagsKeyNode, tagsSeqNode)
		}
	}

	return nil
}

// tagFromPath derives a tag name from an OpenAPI path string.
// It takes the first non-empty segment that is not "v1" and not a path parameter.
// Dashes are replaced by spaces and the result has its first letter uppercased.
func tagFromPath(path string) string {
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		// Skip version segments like "v1", "v2", etc.
		if len(part) >= 2 && part[0] == 'v' && part[1] >= '0' && part[1] <= '9' {
			continue
		}
		// Skip path parameters like "{id}"
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			continue
		}
		return upperFirst(strings.ReplaceAll(part, "-", " "))
	}
	return ""
}

// upperFirst capitalises only the first rune of s (without using strings.Title,
// which would title-case every word). This intentionally capitalises only the
// first segment-word derived from the path.
func upperFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// sanitizeSchemaIdentifiers renames component schema keys that contain
// characters outside [a-zA-Z0-9._-] (e.g. spaces) by replacing each invalid
// character with "_", then rewrites every "$ref" string in the document that
// points to a renamed schema.  This is a narrow fixup for upstream Network
// specs that use human-readable names like "ACL rule" as schema identifiers.
//
// The function operates on the decoded map[string]any tree (after yaml→Go
// decode, before JSON serialisation) so that renaming and ref-rewriting are
// both done in a single pass on a uniform representation.
//
// If two distinct schema names would sanitize to the same identifier, the
// function returns an error rather than silently dropping one definition.
func sanitizeSchemaIdentifiers(doc map[string]any) error {
	components, _ := doc["components"].(map[string]any)
	if components == nil {
		return nil
	}
	schemas, _ := components["schemas"].(map[string]any)
	if schemas == nil {
		return nil
	}

	// Build a rename map: oldName → newName for names that need changing.
	// At the same time detect collisions: two names that sanitize to the same
	// target, or a sanitized name that collides with an already-existing key.
	rename := make(map[string]string)
	// sanitizedTo maps each sanitized target back to whichever original name
	// claimed it first, for collision reporting.
	sanitizedTo := make(map[string]string, len(schemas))

	for name := range schemas {
		sanitized := identifierInvalidCharRE.ReplaceAllString(name, "_")
		if sanitized == name {
			// No rename needed; still register the name so collisions are caught.
			sanitizedTo[name] = name
			continue
		}
		// Check whether the target is already claimed.
		if prior, conflict := sanitizedTo[sanitized]; conflict {
			return fmt.Errorf(
				"schema identifier collision: %q and %q both sanitize to %q",
				prior, name, sanitized,
			)
		}
		// Check whether the sanitized name already exists as a schema key
		// (i.e. an existing clean key would be overwritten).
		if _, exists := schemas[sanitized]; exists {
			return fmt.Errorf(
				"schema identifier collision: sanitizing %q to %q would overwrite an existing schema",
				name, sanitized,
			)
		}
		sanitizedTo[sanitized] = name
		rename[name] = sanitized
	}

	if len(rename) == 0 {
		return nil // nothing to do
	}

	// Apply renames in the schemas map itself.
	for oldName, newName := range rename {
		schemas[newName] = schemas[oldName]
		delete(schemas, oldName)
	}

	// Rewrite all $ref strings in the document that reference renamed schemas.
	rewriteRefs(doc, rename)
	return nil
}

// rewriteRefs recursively walks v (a decoded JSON value: map, slice, or
// scalar) and rewrites "$ref" string values that reference renamed schemas.
// It also rewrites discriminator.mapping values, which are plain map values
// (not "$ref" keys) that may contain "#/components/schemas/<name>" references.
func rewriteRefs(v any, rename map[string]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			switch k {
			case "$ref":
				if s, ok := child.(string); ok {
					val[k] = rewriteSchemaRef(s, rename)
				}
			case "discriminator":
				// Rewrite mapping values inside discriminator objects.
				if discMap, ok := child.(map[string]any); ok {
					if mapping, ok2 := discMap["mapping"].(map[string]any); ok2 {
						for mappingKey, mappingVal := range mapping {
							if s, ok3 := mappingVal.(string); ok3 {
								mapping[mappingKey] = rewriteSchemaRef(s, rename)
							}
						}
					}
				}
				// Still recurse into the discriminator object for nested structures.
				rewriteRefs(child, rename)
			default:
				rewriteRefs(child, rename)
			}
		}
	case []any:
		for _, item := range val {
			rewriteRefs(item, rename)
		}
	}
}

// rewriteSchemaRef rewrites a single "#/components/schemas/<name>" reference
// string using the rename map.  It handles both the full "#/components/schemas/"
// prefix form and bare schema-name shorthands (per OpenAPI spec).  The input
// string is returned unchanged when no rename applies.
func rewriteSchemaRef(s string, rename map[string]string) string {
	const prefix = "#/components/schemas/"
	if after, ok := strings.CutPrefix(s, prefix); ok {
		if newName, changed := rename[after]; changed {
			return prefix + newName
		}
		return s
	}
	// Bare schema-name shorthand (no prefix): rewrite if it matches a rename.
	if newName, changed := rename[s]; changed {
		return newName
	}
	return s
}

// stripInvalidEnumExamples recursively walks a yaml.Node tree and removes the
// "example" key from any schema-like mapping that has BOTH "enum" (a sequence)
// and "example" (a scalar), when the example value is NOT a member of the
// enum sequence.  Membership is checked by comparing both the node Value and
// Tag (e.g. an integer example 3 does not match a string enum member "3").
// Valid examples are preserved unchanged.
//
// This fixes ~46 upstream spec quirks such as `"example": "ALLOW|BLOCK"` on
// `"enum": ["ALLOW", "BLOCK"]` that kin-openapi's full doc.Validate rejects.
func stripInvalidEnumExamples(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		// Find the indices of "enum" and "example" keys (if present).
		enumIdx := -1
		exampleKeyIdx := -1
		for i := 0; i+1 < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "enum":
				enumIdx = i + 1
			case "example":
				exampleKeyIdx = i
			}
		}

		if enumIdx >= 0 && exampleKeyIdx >= 0 {
			enumNode := node.Content[enumIdx]
			exampleValNode := node.Content[exampleKeyIdx+1]

			if enumNode.Kind == yaml.SequenceNode && exampleValNode.Kind == yaml.ScalarNode {
				// Check whether example is a valid enum member.
				validMember := false
				for _, enumItem := range enumNode.Content {
					if enumItem.Kind == yaml.ScalarNode &&
						enumItem.Value == exampleValNode.Value &&
						enumItem.Tag == exampleValNode.Tag {
						validMember = true
						break
					}
				}
				if !validMember {
					// Remove the "example" key-value pair from Content.
					newContent := make([]*yaml.Node, 0, len(node.Content)-2)
					for i := 0; i+1 < len(node.Content); i += 2 {
						if i == exampleKeyIdx {
							continue // skip this key-value pair
						}
						newContent = append(newContent, node.Content[i], node.Content[i+1])
					}
					node.Content = newContent
				}
			}
		}

		// Recurse into all children (values only; keys are scalars).
		for i := 1; i < len(node.Content); i += 2 {
			stripInvalidEnumExamples(node.Content[i])
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			stripInvalidEnumExamples(child)
		}

	case yaml.DocumentNode:
		for _, child := range node.Content {
			stripInvalidEnumExamples(child)
		}
	}
}

// stripInvalidTypeExamples recursively walks a yaml.Node tree and removes the
// "example" key from any schema-like mapping where the example's YAML tag is
// inconsistent with the schema's declared "type".  This handles upstream spec
// quirks like `example: "786435|720973"` (a string) on `type: integer`.
//
// Specifically, the example is removed when:
//   - type is "integer" or "number" and example is not tagged !!int or !!float
//   - type is "boolean" and example is not tagged !!bool
//   - type is "array" and example is not a sequence node
//   - type is "object" and example is not a mapping node
//
// String-typed schemas are left untouched because any string example is valid.
func stripInvalidTypeExamples(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		typeIdx := -1
		exampleKeyIdx := -1
		for i := 0; i+1 < len(node.Content); i += 2 {
			switch node.Content[i].Value {
			case "type":
				typeIdx = i + 1
			case "example":
				exampleKeyIdx = i
			}
		}

		if typeIdx >= 0 && exampleKeyIdx >= 0 {
			typeNode := node.Content[typeIdx]
			exampleValNode := node.Content[exampleKeyIdx+1]

			if typeNode.Kind == yaml.ScalarNode {
				schemaType := typeNode.Value
				remove := false
				switch schemaType {
				case "integer":
					remove = exampleValNode.Kind != yaml.ScalarNode ||
						(exampleValNode.Tag != "!!int" && exampleValNode.Tag != "!!float")
				case "number":
					remove = exampleValNode.Kind != yaml.ScalarNode ||
						(exampleValNode.Tag != "!!int" && exampleValNode.Tag != "!!float")
				case "boolean":
					remove = exampleValNode.Kind != yaml.ScalarNode ||
						exampleValNode.Tag != "!!bool"
				case "string":
					// A string example must be a scalar with tag !!str or !!null.
					// Numbers/booleans decoded as !!int/!!float/!!bool are invalid.
					remove = exampleValNode.Kind != yaml.ScalarNode ||
						(exampleValNode.Tag != "!!str" && exampleValNode.Tag != "!!null")
				case "array":
					remove = exampleValNode.Kind != yaml.SequenceNode
				case "object":
					remove = exampleValNode.Kind != yaml.MappingNode
				}
				if remove {
					newContent := make([]*yaml.Node, 0, len(node.Content)-2)
					for i := 0; i+1 < len(node.Content); i += 2 {
						if i == exampleKeyIdx {
							continue
						}
						newContent = append(newContent, node.Content[i], node.Content[i+1])
					}
					node.Content = newContent
				}
			}
		}

		// Recurse into values.
		for i := 1; i < len(node.Content); i += 2 {
			stripInvalidTypeExamples(node.Content[i])
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			stripInvalidTypeExamples(child)
		}

	case yaml.DocumentNode:
		for _, child := range node.Content {
			stripInvalidTypeExamples(child)
		}
	}
}

// deduplicateAllOfDiscriminators walks the document tree and removes redundant
// discriminator objects from allOf schemas.  When a single allOf contains
// multiple items that each declare a discriminator on the same propertyName,
// oapi-codegen v2 refuses to merge them ("merging two schemas with
// discriminators is not supported").  Because all of the discriminator entries
// in such an allOf reference the same property (e.g. "origin") they are
// semantically redundant — the first one is authoritative.  This fixup retains
// the discriminator on the first allOf item that declares it and strips the
// discriminator key from every subsequent item in the same allOf that uses the
// same propertyName.  Non-discriminator fields (properties, required, etc.) on
// the stripped items are preserved unchanged.
//
// This fixes ~50 upstream Network spec locations such as
// Adopted_device_details.*.metadata that have 7 allOf items each carrying an
// identical "discriminator.propertyName: origin" object.
func deduplicateAllOfDiscriminators(node *yaml.Node) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.MappingNode:
		// Look for an "allOf" key whose sequence has multiple discriminators.
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == "allOf" {
				seqNode := node.Content[i+1]
				if seqNode.Kind == yaml.SequenceNode {
					deduplicateDiscriminatorsInAllOf(seqNode)
				}
			}
		}
		// Recurse into all value nodes.
		for i := 1; i < len(node.Content); i += 2 {
			deduplicateAllOfDiscriminators(node.Content[i])
		}

	case yaml.SequenceNode:
		for _, child := range node.Content {
			deduplicateAllOfDiscriminators(child)
		}

	case yaml.DocumentNode:
		for _, child := range node.Content {
			deduplicateAllOfDiscriminators(child)
		}
	}
}

// deduplicateDiscriminatorsInAllOf processes a single allOf SequenceNode and
// removes discriminator keys from items that repeat a propertyName already
// seen in an earlier item.
func deduplicateDiscriminatorsInAllOf(seqNode *yaml.Node) {
	// seenPropertyNames tracks discriminator.propertyName values already claimed
	// by an earlier allOf item.
	seenPropertyNames := map[string]bool{}

	for _, item := range seqNode.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}

		// Find the "discriminator" key-value pair in this item.
		discIdx := -1
		for j := 0; j+1 < len(item.Content); j += 2 {
			if item.Content[j].Value == "discriminator" {
				discIdx = j
				break
			}
		}
		if discIdx < 0 {
			continue // no discriminator in this item
		}

		discNode := item.Content[discIdx+1]
		if discNode.Kind != yaml.MappingNode {
			continue
		}

		// Extract propertyName from the discriminator object.
		propNameNode := mappingValue(discNode, "propertyName")
		if propNameNode == nil {
			continue
		}
		propName := propNameNode.Value

		if seenPropertyNames[propName] {
			// This discriminator is a duplicate — remove it from this item.
			newContent := make([]*yaml.Node, 0, len(item.Content)-2)
			for j := 0; j+1 < len(item.Content); j += 2 {
				if j == discIdx {
					continue // skip key + value pair
				}
				newContent = append(newContent, item.Content[j], item.Content[j+1])
			}
			item.Content = newContent
		} else {
			seenPropertyNames[propName] = true
		}
	}
}

// normalizeNullableTypeArrays walks a decoded JSON document (map[string]any)
// and converts OpenAPI 3.1 nullable type arrays to oapi-codegen-compatible
// scalar types.  OpenAPI 3.1 allows `"type": ["string", "null"]` to express
// a nullable string, but oapi-codegen v2 does not handle multi-value type
// arrays and emits "unhandled Schema type: &[...]".
//
// The transformation applied:
//   - `"type": ["T", "null"]` or `"type": ["null", "T"]`
//     → `"type": "T"` + `"nullable": true`
//
// When the array contains more than two elements, or two non-null elements,
// the array is left unchanged (complex unions are out of scope).
// This fixup must run on the map[string]any representation so that the
// resulting "nullable" key sorts alongside other string keys in the
// deterministic JSON output.
func normalizeNullableTypeArrays(v any) {
	switch val := v.(type) {
	case map[string]any:
		// Check for "type" key whose value is a []any.
		if rawType, ok := val["type"]; ok {
			if typeArr, ok := rawType.([]any); ok {
				if scalar, isNullable := extractNullableScalarType(typeArr); isNullable {
					val["type"] = scalar
					val["nullable"] = true
				}
				// If not a simple nullable pair, leave unchanged.
			}
		}
		// Recurse into all values.
		for _, child := range val {
			normalizeNullableTypeArrays(child)
		}
	case []any:
		for _, item := range val {
			normalizeNullableTypeArrays(item)
		}
	}
}

// extractNullableScalarType checks whether typeArr is a two-element array of
// the form ["T", "null"] or ["null", "T"] where T is a non-null scalar string.
// If so, it returns (T, true).  Otherwise it returns ("", false).
func extractNullableScalarType(typeArr []any) (string, bool) {
	if len(typeArr) != 2 {
		return "", false
	}
	a, aOK := typeArr[0].(string)
	b, bOK := typeArr[1].(string)
	if !aOK || !bOK {
		return "", false
	}
	switch {
	case b == "null" && a != "null":
		return a, true
	case a == "null" && b != "null":
		return b, true
	}
	return "", false
}

// collapseOneOfNullable walks a decoded JSON document (map[string]any) and
// collapses OpenAPI 3.1 "oneOf with null" patterns into oapi-codegen-compatible
// nullable schemas.
//
// OpenAPI 3.1 allows expressing a nullable type via:
//
//	{ "oneOf": [ <schema>, { "type": "null" } ] }
//
// oapi-codegen v2 does not support "type": "null" as a oneOf member and
// errors with "unhandled Schema type: &[null]".
//
// When a schema object has a "oneOf" array containing exactly two items where
// exactly one item is {"type": "null"} and the other is a non-null schema,
// this function replaces the entire enclosing object with the non-null schema
// plus "nullable": true, merging all other top-level keys from the enclosing
// object (e.g. "description" at the outer level).
//
// This fixes ~530 upstream Protect spec locations such as
// aiPort.properties.name: { oneOf: [{description:..., type:"string"}, {type:"null"}] }.
func collapseOneOfNullable(v any) any {
	switch val := v.(type) {
	case map[string]any:
		// Recurse into all children first (bottom-up).
		for k, child := range val {
			val[k] = collapseOneOfNullable(child)
		}

		// Check if this map has a "oneOf" with exactly one null-type item.
		rawOneOf, hasOneOf := val["oneOf"]
		if !hasOneOf {
			return val
		}
		oneOfArr, ok := rawOneOf.([]any)
		if !ok || len(oneOfArr) != 2 {
			return val
		}

		nullIdx := -1
		nonNullIdx := -1
		for i, item := range oneOfArr {
			m, ok := item.(map[string]any)
			if !ok {
				return val // not a map, skip
			}
			if m["type"] == "null" && len(m) == 1 {
				nullIdx = i
			} else {
				nonNullIdx = i
			}
		}
		if nullIdx < 0 || nonNullIdx < 0 {
			return val // not the nullable pattern
		}

		// Build the replacement: start with the non-null schema, add nullable.
		nonNullSchema, ok := oneOfArr[nonNullIdx].(map[string]any)
		if !ok {
			return val
		}
		result := make(map[string]any, len(nonNullSchema)+len(val))
		// Copy keys from the non-null schema.
		for k, child := range nonNullSchema {
			result[k] = child
		}
		// Overlay any extra keys from the enclosing object (excluding "oneOf").
		for k, child := range val {
			if k == "oneOf" {
				continue
			}
			result[k] = child
		}
		result["nullable"] = true
		return result

	case []any:
		for i, item := range val {
			val[i] = collapseOneOfNullable(item)
		}
		return val
	}
	return v
}

// inlineDiscriminatedOneOfToRefs walks a decoded JSON document and replaces
// inline schema objects in discriminated oneOf/anyOf arrays with $ref pointers.
//
// oapi-codegen v2 requires that every member of a discriminated oneOf/anyOf
// be a $ref rather than an inline schema, and errors with
// "ambiguous discriminator.mapping: please replace inlined object with $ref"
// when inline objects are encountered.
//
// When a schema object has both a "discriminator" and a "oneOf"/"anyOf" key,
// each inline member of the array (i.e. without a "$ref" key) is replaced by
// {"$ref": "#/components/schemas/<title>"} provided the item's "title" field
// names an existing key in components.schemas.  Items that already are $refs,
// or whose title cannot be resolved, are left unchanged.
//
// The allSchemas parameter is the map[string]any from doc["components"]["schemas"],
// used to verify that the title resolves before rewriting.
func inlineDiscriminatedOneOfToRefs(v any, allSchemas map[string]any) {
	switch val := v.(type) {
	case map[string]any:
		for _, key := range []string{"oneOf", "anyOf"} {
			if _, hasDisc := val["discriminator"]; !hasDisc {
				continue
			}
			rawArr, ok := val[key]
			if !ok {
				continue
			}
			arr, ok := rawArr.([]any)
			if !ok {
				continue
			}
			for i, item := range arr {
				m, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if _, isRef := m["$ref"]; isRef {
					continue // already a $ref
				}
				title, _ := m["title"].(string)
				if title == "" {
					continue // no title to resolve
				}
				if _, exists := allSchemas[title]; !exists {
					continue // title not in components.schemas
				}
				arr[i] = map[string]any{
					"$ref": "#/components/schemas/" + title,
				}
			}
		}
		// Recurse into all values.
		for _, child := range val {
			inlineDiscriminatedOneOfToRefs(child, allSchemas)
		}
	case []any:
		for _, item := range val {
			inlineDiscriminatedOneOfToRefs(item, allSchemas)
		}
	}
}

// deduplicateResponseSchemas replaces inline response body schemas with $ref
// pointers when the inline schema is byte-for-byte identical to a named
// component schema.  This avoids oapi-codegen emitting duplicate Go type names
// when the same inline schema appears in multiple response bodies.
//
// The upstream Protect spec inlines the same schema (e.g. "linkStation") in
// 127 response content objects, causing oapi-codegen to generate the same
// type name multiple times (e.g. GetV1AlarmHubs200JSONResponseBodyAlarmHub
// Connector12v appears twice), which fails compilation.
//
// For each path/operation/response/content schema that:
//   - is not already a $ref
//   - is byte-for-byte identical to a named component schema → replace with $ref
//   - is a {type:array, items: <schema>} where items matches a component schema
//     → replace items with $ref
//
// The comparison uses canonical JSON serialisation (sorted keys, json.Marshal).
func deduplicateResponseSchemas(doc map[string]any) {
	comps, _ := doc["components"].(map[string]any)
	if comps == nil {
		return
	}
	schemas, _ := comps["schemas"].(map[string]any)
	if schemas == nil {
		return
	}

	// Build a canonical-JSON → schema-name lookup.
	canonicals := make(map[string]string, len(schemas))
	for name, schema := range schemas {
		b, err := json.Marshal(schema)
		if err != nil {
			continue
		}
		canonicals[string(b)] = name
	}

	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return
	}

	httpMethods := []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

	for _, pathItem := range paths {
		pi, _ := pathItem.(map[string]any)
		if pi == nil {
			continue
		}
		for _, method := range httpMethods {
			op, _ := pi[method].(map[string]any)
			if op == nil {
				continue
			}
			responses, _ := op["responses"].(map[string]any)
			if responses == nil {
				continue
			}
			for _, response := range responses {
				resp, _ := response.(map[string]any)
				if resp == nil {
					continue
				}
				content, _ := resp["content"].(map[string]any)
				if content == nil {
					continue
				}
				for _, ctObj := range content {
					ct, _ := ctObj.(map[string]any)
					if ct == nil {
						continue
					}
					schema, _ := ct["schema"].(map[string]any)
					if schema == nil {
						continue
					}
					// If schema already has a $ref, skip.
					if _, isRef := schema["$ref"]; isRef {
						continue
					}
					// Try direct match.
					if ref := canonicalRef(schema, canonicals); ref != "" {
						ct["schema"] = map[string]any{"$ref": ref}
						continue
					}
					// Try array-of-component: {type:"array", items: <component>}
					if schema["type"] == "array" {
						items, _ := schema["items"].(map[string]any)
						if items != nil {
							if _, isRef := items["$ref"]; !isRef {
								if ref := canonicalRef(items, canonicals); ref != "" {
									schema["items"] = map[string]any{"$ref": ref}
								}
							}
						}
					}
				}
			}
		}
	}
}

// canonicalRef returns the $ref string for the component schema that is
// byte-for-byte identical to schema, or "" if no match.
func canonicalRef(schema map[string]any, canonicals map[string]string) string {
	b, err := json.Marshal(schema)
	if err != nil {
		return ""
	}
	if name, ok := canonicals[string(b)]; ok {
		return "#/components/schemas/" + name
	}
	return ""
}

// collapseArrayScalarAnyOf walks path operation parameters and collapses
// anyOf schemas of the form [array-of-X, X] to just array-of-X.
//
// oapi-codegen v2 cannot generate non-conflicting type names when a parameter
// schema's anyOf contains both an array variant and the scalar form of the same
// underlying type: it generates the same type name for both (e.g.
// DeleteV1CamerasIdRtspsStreamParamsQualities0 as both []self and string),
// causing a "redeclared in this block" / "invalid recursive type" compile error.
//
// When an anyOf has exactly one {type:"array", items:X} member and one member
// whose content equals X (i.e. the scalar base type), the entire anyOf schema
// is replaced with just the array schema.  If X matches a named component
// schema, the items are further replaced with a $ref for cleaner output.
func collapseArrayScalarAnyOf(doc map[string]any) {
	comps, _ := doc["components"].(map[string]any)
	schemas := map[string]any{}
	if comps != nil {
		if s, ok := comps["schemas"].(map[string]any); ok {
			schemas = s
		}
	}

	// Build canonical hash → schema name for component schemas.
	canonicals := make(map[string]string, len(schemas))
	for name, schema := range schemas {
		b, err := json.Marshal(schema)
		if err == nil {
			canonicals[string(b)] = name
		}
	}

	paths, _ := doc["paths"].(map[string]any)
	if paths == nil {
		return
	}
	httpMethods := []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

	for _, pathItem := range paths {
		pi, _ := pathItem.(map[string]any)
		if pi == nil {
			continue
		}
		for _, method := range httpMethods {
			op, _ := pi[method].(map[string]any)
			if op == nil {
				continue
			}
			params, _ := op["parameters"].([]any)
			for _, param := range params {
				p, _ := param.(map[string]any)
				if p == nil {
					continue
				}
				schema, _ := p["schema"].(map[string]any)
				if schema == nil {
					continue
				}
				if _, isRef := schema["$ref"]; isRef {
					continue
				}
				anyOf, _ := schema["anyOf"].([]any)
				if len(anyOf) == 0 {
					continue
				}

				// Find the array-of-X and scalar-X members.
				arrayIdx := -1
				scalarIdx := -1
				var arrayItems map[string]any
				for i, item := range anyOf {
					m, _ := item.(map[string]any)
					if m == nil {
						continue
					}
					if m["type"] == "array" {
						if items, ok := m["items"].(map[string]any); ok {
							arrayIdx = i
							arrayItems = items
						}
					} else {
						scalarIdx = i
					}
				}
				if arrayIdx < 0 || scalarIdx < 0 || arrayItems == nil {
					continue
				}

				scalarM, _ := anyOf[scalarIdx].(map[string]any)
				if scalarM == nil {
					continue
				}

				// Check if scalar equals items content.
				bItems, err1 := json.Marshal(arrayItems)
				bScalar, err2 := json.Marshal(scalarM)
				if err1 != nil || err2 != nil {
					continue
				}
				if string(bItems) != string(bScalar) {
					continue // not matching
				}

				// Replace anyOf with just the array schema.
				// Try to use a $ref for items if it matches a component.
				newItems := map[string]any(arrayItems)
				if name, ok := canonicals[string(bItems)]; ok {
					newItems = map[string]any{"$ref": "#/components/schemas/" + name}
				}
				arrayMember, _ := anyOf[arrayIdx].(map[string]any)
				if arrayMember == nil {
					continue
				}

				// Build replacement: copy the array member, swap in possibly-ref items.
				replacement := make(map[string]any)
				for k, v := range arrayMember {
					replacement[k] = v
				}
				replacement["items"] = newItems
				// Copy non-anyOf top-level keys from schema (e.g. description).
				for k, v := range schema {
					if k == "anyOf" {
						continue
					}
					replacement[k] = v
				}
				p["schema"] = replacement
			}
		}
	}
}

// stripEmptyLicense removes info.license from the document when it exists but
// has no "name" field.  OpenAPI 3.x requires license.name when license is
// present; some upstream specs include an empty "license": {} object.
func stripEmptyLicense(doc *yaml.Node) {
	infoNode := mappingValue(doc, "info")
	if infoNode == nil || infoNode.Kind != yaml.MappingNode {
		return
	}
	licenseNode := mappingValue(infoNode, "license")
	if licenseNode == nil {
		return
	}
	// If license has a non-empty "name", leave it alone.
	if nameNode := mappingValue(licenseNode, "name"); nameNode != nil && nameNode.Value != "" {
		return
	}
	// Remove the "license" key-value pair from info.
	newContent := make([]*yaml.Node, 0, len(infoNode.Content)-2)
	for i := 0; i+1 < len(infoNode.Content); i += 2 {
		if infoNode.Content[i].Value == "license" {
			continue
		}
		newContent = append(newContent, infoNode.Content[i], infoNode.Content[i+1])
	}
	infoNode.Content = newContent
}

// ensureResponseDescriptions walks all path operations and adds an empty
// description to any response that lacks one.  OpenAPI 3.x requires a
// description on every response object; some upstream specs omit it.
func ensureResponseDescriptions(doc *yaml.Node) {
	pathsNode := mappingValue(doc, "paths")
	if pathsNode == nil || pathsNode.Kind != yaml.MappingNode {
		return
	}

	httpMethods := map[string]bool{
		"get": true, "put": true, "post": true, "delete": true,
		"options": true, "head": true, "patch": true, "trace": true,
	}

	for i := 0; i+1 < len(pathsNode.Content); i += 2 {
		pathItem := pathsNode.Content[i+1]
		if pathItem.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(pathItem.Content); j += 2 {
			if !httpMethods[strings.ToLower(pathItem.Content[j].Value)] {
				continue
			}
			opNode := pathItem.Content[j+1]
			responsesNode := mappingValue(opNode, "responses")
			if responsesNode == nil || responsesNode.Kind != yaml.MappingNode {
				continue
			}
			for k := 0; k+1 < len(responsesNode.Content); k += 2 {
				responseObj := responsesNode.Content[k+1]
				if responseObj.Kind != yaml.MappingNode {
					continue
				}
				if mappingValue(responseObj, "description") == nil {
					keyNode := scalarNode("description")
					valNode := scalarNode("")
					responseObj.Content = append(responseObj.Content, keyNode, valNode)
				}
			}
		}
	}
}
