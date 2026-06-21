package cli

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/thathaneydude/unifi/unifi"
)

// Param describes a single CLI flag derived from an OpenAPI parameter.
type Param struct {
	Name        string
	In          string // "path" or "query"
	Required    bool
	Description string
}

// Operation is a single invocable API operation: `unifi <app> <ID>`.
type Operation struct {
	App         unifi.App
	Version     string
	ID          string
	Method      string // upper-case HTTP method
	Path        string // server-relative path template, e.g. /v1/widgets/{id}
	Summary     string
	PathParams  []Param
	QueryParams []Param
	// BodyMediaType is the request body's Content-Type ("" when no request body).
	BodyMediaType string
}

// HasBody reports whether the operation declares a request body.
func (o Operation) HasBody() bool { return o.BodyMediaType != "" }

// Mutating reports whether the operation changes server state and therefore
// requires --confirm (and supports --dry-run).
func (o Operation) Mutating() bool {
	switch o.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// selectBodyMediaType picks the Content-Type for an operation's request body. It
// prefers application/json, then falls back to the lexically-first key so the
// choice is deterministic. Returns "" when there is no usable body.
func selectBodyMediaType(rb *openapi3.RequestBodyRef) string {
	if rb == nil || rb.Value == nil || len(rb.Value.Content) == 0 {
		return ""
	}
	if _, ok := rb.Value.Content["application/json"]; ok {
		return "application/json"
	}
	keys := make([]string, 0, len(rb.Value.Content))
	for k := range rb.Value.Content {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

// synthesizeOperationID builds a deterministic identifier for an operation that
// has no operationId in the spec (the UniFi Protect spec omits them). It mirrors
// oapi-codegen's method+path naming so the CLI's IDs match the generated SDK
// methods: GET /v1/alarm-hubs/{id} becomes "GetV1AlarmHubsId".
func synthesizeOperationID(method, pathTmpl string) string {
	var b strings.Builder
	b.WriteString(capitalizeASCII(strings.ToLower(method)))
	for _, seg := range strings.Split(pathTmpl, "/") {
		if seg == "" {
			continue
		}
		seg = strings.NewReplacer("{", "", "}", "").Replace(seg)
		for _, part := range strings.FieldsFunc(seg, func(r rune) bool {
			return !isASCIIAlnum(r)
		}) {
			b.WriteString(capitalizeASCII(part))
		}
	}
	return b.String()
}

func capitalizeASCII(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 'a' - 'A'
	}
	return string(r)
}

func isASCIIAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// OperationsFor extracts every operation from a parsed spec, sorted by ID for
// deterministic output. Operations without an operationId have one synthesized
// from the HTTP method and path (mirroring oapi-codegen's naming convention).
func OperationsFor(doc *openapi3.T, app unifi.App, version string) []Operation {
	var ops []Operation
	if doc.Paths == nil {
		return ops
	}
	for pathTmpl, item := range doc.Paths.Map() {
		for method, oapiOp := range item.Operations() {
			id := oapiOp.OperationID
			if id == "" {
				id = synthesizeOperationID(method, pathTmpl)
			}
			op := Operation{
				App:           app,
				Version:       version,
				ID:            id,
				Method:        method,
				Path:          pathTmpl,
				Summary:       oapiOp.Summary,
				BodyMediaType: selectBodyMediaType(oapiOp.RequestBody),
			}

			// Merge path-item-level parameters (apply to every operation on the
			// path) with operation-level ones. Per the OpenAPI spec, an
			// operation-level parameter overrides a path-item one with the same
			// (in, name). Path-item params are added first so order is stable.
			type pkey struct{ in, name string }
			seen := map[pkey]Param{}
			var order []pkey
			addParam := func(pref *openapi3.ParameterRef) {
				if pref == nil || pref.Value == nil {
					return
				}
				k := pkey{in: pref.Value.In, name: pref.Value.Name}
				if _, ok := seen[k]; !ok {
					order = append(order, k)
				}
				seen[k] = Param{
					Name:        pref.Value.Name,
					In:          pref.Value.In,
					Required:    pref.Value.Required,
					Description: pref.Value.Description,
				}
			}
			for _, pref := range item.Parameters {
				addParam(pref)
			}
			for _, pref := range oapiOp.Parameters {
				addParam(pref)
			}
			for _, k := range order {
				p := seen[k]
				switch p.In {
				case openapi3.ParameterInPath:
					op.PathParams = append(op.PathParams, p)
				case openapi3.ParameterInQuery:
					op.QueryParams = append(op.QueryParams, p)
				}
			}

			ops = append(ops, op)
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].ID < ops[j].ID })

	// Guarantee unique IDs so cobra never silently shadows a command. Runs after
	// the sort so suffixes are deterministic; the first occurrence keeps the bare
	// ID and later collisions get -2, -3, ... . Synthesized IDs are alnum-only
	// PascalCase, so a hyphenated suffix cannot collide with a real ID.
	counts := map[string]int{}
	for i := range ops {
		n := counts[ops[i].ID]
		counts[ops[i].ID] = n + 1
		if n > 0 {
			ops[i].ID = fmt.Sprintf("%s-%d", ops[i].ID, n+1)
		}
	}
	return ops
}
