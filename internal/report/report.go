// Package report renders a UniFi security-assessment findings document
// (produced as JSON by the unifi-security-assessment skill) into a single,
// self-contained, UniFi-branded HTML report.
//
// The JSON schema mirrors the canonical "finding shape" and report skeleton in
// skills/unifi-security-assessment/references/report-template.md, so the skill
// can emit findings.json and hand it to `unifi report` without any markdown
// parsing.
package report

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"strings"
	"unicode"
)

//go:embed template.html
var templateFS embed.FS

// Severities lists the severity buckets in rubric order (most to least severe).
// Findings render grouped into these sections; an empty bucket renders "None."
var Severities = []string{"critical", "high", "medium", "low", "info"}

// Report is the top-level document. Field names map 1:1 to the JSON the
// assessment skill emits.
type Report struct {
	Date          string            `json:"date"`
	Console       Console           `json:"console"`
	SiteCount     int               `json:"site_count"`
	SkillVersions map[string]string `json:"skill_versions"`
	AssessedBy    AssessedBy        `json:"assessed_by"`
	Counts        Counts            `json:"counts"`
	TopRisks      []string          `json:"top_risks"`
	Findings      []Finding         `json:"findings"`
	Coverage      Coverage          `json:"coverage"`
	Appendix      []AppendixSection `json:"appendix"`
}

// Console identifies the assessed deployment.
type Console struct {
	Name           string `json:"name"`
	NetworkVersion string `json:"network_version"`
	ProtectVersion string `json:"protect_version"` // version string, or "absent"
}

// AssessedBy records the AI model that produced the assessment so reports stay
// diffable when a newer model re-evaluates the same deployment.
type AssessedBy struct {
	ModelName string `json:"model_name"`
	ModelID   string `json:"model_id"`
}

// Counts is the per-severity finding tally shown in the executive summary.
type Counts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Finding is the canonical finding shape. Evidence is raw JSON captured from the
// CLI (with secrets redacted upstream as "***").
type Finding struct {
	Severity         string          `json:"severity"`
	Title            string          `json:"title"`
	AffectedResource string          `json:"affected_resource"`
	Evidence         json.RawMessage `json:"evidence"`
	Remediation      string          `json:"remediation"`
}

// Coverage records what was assessed and what the integration-API scope could
// not reach.
type Coverage struct {
	DomainsRun    []string `json:"domains_run"`
	Skipped       []string `json:"skipped"`
	NotAssessable []string `json:"not_assessable"`
}

// AppendixSection holds the raw JSON collected for one domain.
type AppendixSection struct {
	Domain string          `json:"domain"`
	Data   json.RawMessage `json:"data"`
}

// view is the data shape handed to the template after grouping/formatting.
type view struct {
	Report
	Groups []severityGroup
}

type severityGroup struct {
	Severity string
	Label    string
	Count    int
	Findings []Finding
}

var tmpl = template.Must(template.New("template.html").Funcs(template.FuncMap{
	"prettyJSON":   prettyJSON,
	"evidenceHTML": evidenceHTML,
	"severityLabel": func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ToUpper(s[:1]) + s[1:]
	},
}).ParseFS(templateFS, "template.html"))

// Render writes the self-contained HTML report for r to w. All user- and
// evidence-supplied text is auto-escaped by html/template; the output contains
// no external stylesheet or script references.
func Render(w io.Writer, r Report) error {
	groups := make([]severityGroup, 0, len(Severities))
	for _, sev := range Severities {
		var fs []Finding
		for _, f := range r.Findings {
			if strings.EqualFold(f.Severity, sev) {
				fs = append(fs, f)
			}
		}
		groups = append(groups, severityGroup{
			Severity: sev,
			Label:    strings.ToUpper(sev[:1]) + sev[1:],
			Count:    len(fs),
			Findings: fs,
		})
	}
	return tmpl.Execute(w, view{Report: r, Groups: groups})
}

// prettyJSON pretty-prints raw JSON for display. Invalid or empty JSON is
// returned verbatim so the report never fails on imperfect evidence.
func prettyJSON(raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		return string(raw)
	}
	return buf.String()
}

// evidenceHTML renders raw evidence JSON as a humanized key/value table:
// camelCase/snake_case keys become Title Case, booleans become Yes/No, null
// becomes an em dash, and nested objects/arrays indent. All keys and values are
// HTML-escaped here (the result is trusted template.HTML), so untrusted evidence
// can never inject markup. Malformed JSON falls back to escaped raw text.
func evidenceHTML(raw json.RawMessage) template.HTML {
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	n, err := parseOrdered(raw)
	if err != nil {
		return template.HTML("<pre><code>" + html.EscapeString(string(raw)) + "</code></pre>")
	}
	var b strings.Builder
	renderNode(&b, n)
	return template.HTML(b.String())
}

// node is an order-preserving JSON value (Go maps lose object key order).
type node struct {
	kind     nodeKind
	scalar   any      // string, json.Number, bool, or nil
	keys     []string // object keys, in source order
	children []*node  // object values (parallel to keys) or array items
}

type nodeKind int

const (
	kindScalar nodeKind = iota
	kindObject
	kindArray
)

func parseOrdered(raw json.RawMessage) (*node, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return parseValue(dec)
}

func parseValue(dec *json.Decoder) (*node, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := t.(json.Delim); ok {
		switch d {
		case '{':
			n := &node{kind: kindObject}
			for dec.More() {
				kt, kerr := dec.Token()
				if kerr != nil {
					return nil, kerr
				}
				key, _ := kt.(string)
				child, verr := parseValue(dec)
				if verr != nil {
					return nil, verr
				}
				n.keys = append(n.keys, key)
				n.children = append(n.children, child)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return nil, err
			}
			return n, nil
		case '[':
			n := &node{kind: kindArray}
			for dec.More() {
				child, verr := parseValue(dec)
				if verr != nil {
					return nil, verr
				}
				n.children = append(n.children, child)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return nil, err
			}
			return n, nil
		}
	}
	return &node{kind: kindScalar, scalar: t}, nil
}

// allScalars reports whether an array node holds only scalar items, so it can
// render inline as a comma-separated list rather than a stack of sub-tables.
func allScalars(n *node) bool {
	if n.kind != kindArray {
		return false
	}
	for _, c := range n.children {
		if c.kind != kindScalar {
			return false
		}
	}
	return true
}

func renderNode(b *strings.Builder, n *node) {
	switch n.kind {
	case kindObject:
		renderObject(b, n)
	case kindArray:
		renderArray(b, n)
	default:
		b.WriteString(`<div class="kv-val">`)
		b.WriteString(scalarHTML(n.scalar))
		b.WriteString(`</div>`)
	}
}

func renderObject(b *strings.Builder, n *node) {
	if len(n.keys) == 0 {
		b.WriteString(`<div class="kv-val">—</div>`)
		return
	}
	b.WriteString(`<div class="kv">`)
	for i, key := range n.keys {
		child := n.children[i]
		label := html.EscapeString(humanizeKey(key))
		if child.kind == kindScalar || allScalars(child) {
			b.WriteString(`<div class="kv-row"><div class="kv-key">`)
			b.WriteString(label)
			b.WriteString(`</div><div class="kv-val">`)
			b.WriteString(inlineValue(child))
			b.WriteString(`</div></div>`)
			continue
		}
		b.WriteString(`<div class="kv-group"><div class="kv-grouphead">`)
		b.WriteString(label)
		b.WriteString(`</div>`)
		renderNode(b, child)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
}

func renderArray(b *strings.Builder, n *node) {
	if len(n.children) == 0 {
		b.WriteString(`<div class="kv-val">—</div>`)
		return
	}
	if allScalars(n) {
		b.WriteString(`<div class="kv-val">`)
		b.WriteString(inlineValue(n))
		b.WriteString(`</div>`)
		return
	}
	b.WriteString(`<div class="kv">`)
	for i, child := range n.children {
		b.WriteString(`<div class="kv-group"><div class="kv-grouphead">#`)
		fmt.Fprintf(b, "%d", i+1)
		b.WriteString(`</div>`)
		renderNode(b, child)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div>`)
}

// inlineValue renders a scalar, or an array of scalars as a comma-separated
// list, as escaped text suitable for a single value cell.
func inlineValue(n *node) string {
	if n.kind == kindScalar {
		return scalarHTML(n.scalar)
	}
	parts := make([]string, 0, len(n.children))
	for _, c := range n.children {
		parts = append(parts, scalarHTML(c.scalar))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

// scalarHTML renders a JSON scalar as escaped, human-friendly text.
func scalarHTML(v any) string {
	switch x := v.(type) {
	case nil:
		return "—"
	case bool:
		if x {
			return "Yes"
		}
		return "No"
	case json.Number:
		return html.EscapeString(x.String())
	case string:
		if x == "" {
			return "—"
		}
		return html.EscapeString(x)
	default:
		return html.EscapeString(fmt.Sprintf("%v", x))
	}
}

// acronyms maps a lowercased word to its canonical casing so network/security
// field labels read correctly (e.g. "ssid" -> "SSID") instead of "Ssid".
var acronyms = map[string]string{
	"id": "ID", "ip": "IP", "mac": "MAC", "ssid": "SSID", "vlan": "VLAN",
	"vlans": "VLANs", "lan": "LAN", "wan": "WAN", "wlan": "WLAN", "dns": "DNS",
	"dhcp": "DHCP", "vpn": "VPN", "pmf": "PMF", "nvr": "NVR", "rtsp": "RTSP",
	"rtsps": "RTSPS", "acl": "ACL", "url": "URL", "uri": "URI", "api": "API",
	"ui": "UI", "cpu": "CPU", "mtu": "MTU", "nat": "NAT", "ntp": "NTP",
	"tcp": "TCP", "udp": "UDP", "http": "HTTP", "https": "HTTPS", "ssh": "SSH",
	"tls": "TLS", "ssl": "SSL", "poe": "PoE", "sfp": "SFP", "ap": "AP",
	"aps": "APs", "qos": "QoS", "iot": "IoT", "wifi": "Wi-Fi", "radius": "RADIUS",
}

// humanizeKey turns an API field name into a Title Case label, expanding known
// acronyms, e.g. "allowReturnTraffic" -> "Allow Return Traffic",
// "zoneId" -> "Zone ID", "ssid" -> "SSID".
func humanizeKey(s string) string {
	if s == "" {
		return ""
	}
	var words []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = nil
		}
	}
	var prev rune
	for i, r := range s {
		switch {
		case r == '_' || r == '-' || r == ' ' || r == '.':
			flush()
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(prev) || unicode.IsDigit(prev)):
			flush()
			cur = append(cur, r)
		default:
			cur = append(cur, r)
		}
		prev = r
	}
	flush()
	for i, w := range words {
		if canon, ok := acronyms[strings.ToLower(w)]; ok {
			words[i] = canon
			continue
		}
		rs := []rune(w)
		rs[0] = unicode.ToUpper(rs[0])
		for j := 1; j < len(rs); j++ {
			rs[j] = unicode.ToLower(rs[j])
		}
		words[i] = string(rs)
	}
	return strings.Join(words, " ")
}

// Parse decodes a findings JSON document into a Report, rejecting malformed
// input with a descriptive error.
func Parse(data []byte) (Report, error) {
	var r Report
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&r); err != nil {
		return Report{}, fmt.Errorf("invalid findings JSON: %w", err)
	}
	return r, nil
}
