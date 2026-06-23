package cli

import (
	"encoding/json"
	"strings"
)

// RenderOptions are client-side response post-processing controls applied to a
// successful JSON response before it is rendered. They never change what the API
// returns — only what the CLI prints.
type RenderOptions struct {
	Fields []string // dot-path projection applied to each record (e.g. "name", "action.type")
	Redact bool      // mask values under secret-like keys
	Limit  int       // cap a top-level (or .data) array to N items (0 = off)
}

func (o RenderOptions) empty() bool {
	return len(o.Fields) == 0 && !o.Redact && o.Limit <= 0
}

// sensitiveSubstrings are matched (case-insensitive, substring) against JSON
// field names under --redact.
var sensitiveSubstrings = []string{
	"secret", "password", "passphrase", "psk", "token", "privatekey", "presharedkey", "apikey",
}

func sensitiveKey(name string) bool {
	n := strings.ToLower(name)
	if n == "key" {
		return true
	}
	for _, s := range sensitiveSubstrings {
		if strings.Contains(n, s) {
			return true
		}
	}
	return false
}

// ApplyTransforms applies limit -> fields -> redact to a JSON response body. It
// returns body unchanged when no option is set or the body is not JSON, so it is
// always safe to call.
func ApplyTransforms(body []byte, o RenderOptions) []byte {
	if o.empty() {
		return body
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	if o.Limit > 0 {
		v = limitValue(v, o.Limit)
	}
	if len(o.Fields) > 0 {
		v = projectValue(v, o.Fields)
	}
	if o.Redact {
		redactValue(v)
	}
	out, err := json.Marshal(v)
	if err != nil {
		return body
	}
	return out
}

// limitValue caps a top-level array, or the "data" array inside a page object,
// to n elements.
func limitValue(v any, n int) any {
	switch t := v.(type) {
	case []any:
		if len(t) > n {
			return t[:n]
		}
	case map[string]any:
		if data, ok := t["data"].([]any); ok && len(data) > n {
			t["data"] = data[:n]
		}
	}
	return v
}

// projectValue keeps only the given dot-paths. Paths are relative to each record:
// for a top-level array or a page object's "data" array it projects each element;
// for a plain object it projects the object itself.
func projectValue(v any, paths []string) any {
	switch t := v.(type) {
	case []any:
		for i, el := range t {
			t[i] = projectOne(el, paths)
		}
		return t
	case map[string]any:
		if data, ok := t["data"].([]any); ok {
			for i, el := range data {
				data[i] = projectOne(el, paths)
			}
			return t
		}
		return projectOne(t, paths)
	default:
		return v
	}
}

func projectOne(v any, paths []string) any {
	obj, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := map[string]any{}
	for _, p := range paths {
		segs := strings.Split(p, ".")
		if val, found := pick(obj, segs); found {
			set(out, segs, val)
		}
	}
	return out
}

func pick(obj map[string]any, path []string) (any, bool) {
	var cur any = obj
	for _, seg := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[seg]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func set(obj map[string]any, path []string, val any) {
	for i, seg := range path {
		if i == len(path)-1 {
			obj[seg] = val
			return
		}
		next, ok := obj[seg].(map[string]any)
		if !ok {
			next = map[string]any{}
			obj[seg] = next
		}
		obj = next
	}
}

// redactValue masks, in place, any value held under a secret-like key.
func redactValue(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if sensitiveKey(k) {
				if val != nil {
					t[k] = "***"
				}
				continue
			}
			redactValue(val)
		}
	case []any:
		for _, el := range t {
			redactValue(el)
		}
	}
}
