package report

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func renderSample(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("testdata/sample-findings.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	rep, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func TestRenderContainsExpectedContent(t *testing.T) {
	html := renderSample(t)
	wants := []string{
		"UniFi Security Assessment",
		"UDM-Pro-SE",
		"v10.4.57",
		"Claude Opus 4.8",
		"claude-opus-4-8",
		"Broad WAN→Internal", // title arrow preserved as UTF-8, not mangled
		"Guest network without client isolation",
		"Coverage &amp; Limitations",
		"Appendix",
		"unifi-network-security",
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("rendered HTML missing %q", w)
		}
	}
}

func TestRenderGroupsAllSeverities(t *testing.T) {
	html := renderSample(t)
	for _, sev := range Severities {
		// Each severity section header carries a tinted dot with this class.
		if !strings.Contains(html, "dot-"+sev) {
			t.Errorf("missing severity section for %q", sev)
		}
	}
}

func TestRenderIsSelfContained(t *testing.T) {
	html := renderSample(t)
	if strings.Contains(html, "<link ") {
		t.Error("report references an external stylesheet (<link>); must be self-contained")
	}
	if strings.Contains(html, "<script src") || strings.Contains(html, "src=\"http") {
		t.Error("report references an external script/resource; must be self-contained")
	}
}

func TestRenderEscapesUntrustedEvidence(t *testing.T) {
	html := renderSample(t)
	if strings.Contains(html, "<script>alert('xss')</script>") {
		t.Error("untrusted evidence rendered unescaped — potential injection")
	}
	if !strings.Contains(html, "alert(&#39;xss&#39;)") && !strings.Contains(html, "&lt;script&gt;") {
		t.Error("expected the XSS payload to appear in escaped form")
	}
}

func TestRenderEmptySeverityShowsNone(t *testing.T) {
	rep := Report{
		Date:     "2026-01-01",
		Console:  Console{Name: "Test", NetworkVersion: "v1", ProtectVersion: "absent"},
		Findings: []Finding{{Severity: "high", Title: "only one", Evidence: json.RawMessage(`{}`)}},
	}
	var buf bytes.Buffer
	if err := Render(&buf, rep); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), "None.") {
		t.Error("expected empty severity buckets to render \"None.\"")
	}
}

func TestRenderEvidenceAsHumanizedTable(t *testing.T) {
	html := renderSample(t)
	wants := []string{
		`class="evidence"`,        // evidence is a table, not a <pre> JSON dump
		"Allow Return Traffic",    // humanized key for allowReturnTraffic
		"Zone Id",                 // humanized key for zoneId
		`<div class="kv-val">Yes`, // boolean true -> Yes
		`<div class="kv-val">—`,   // null -> em dash
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("humanized evidence missing %q", w)
		}
	}
	// The critical finding's evidence must no longer render as a raw JSON object.
	if strings.Contains(html, `"allowReturnTraffic": true`) {
		t.Error("evidence still rendered as raw JSON")
	}
}

func TestHumanizeKey(t *testing.T) {
	cases := map[string]string{
		"allowReturnTraffic": "Allow Return Traffic",
		"zoneId":             "Zone Id",
		"network_version":    "Network Version",
		"id":                 "Id",
		"type":               "Type",
	}
	for in, want := range cases {
		if got := humanizeKey(in); got != want {
			t.Errorf("humanizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEvidenceHTMLEscapesAndFallsBack(t *testing.T) {
	out := string(evidenceHTML(json.RawMessage(`{"hostname":"<script>alert('x')</script>"}`)))
	if strings.Contains(out, "<script>alert('x')</script>") {
		t.Error("evidenceHTML did not escape untrusted value")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Error("evidenceHTML missing escaped value")
	}
	// Malformed JSON falls back to escaped raw text rather than panicking.
	bad := string(evidenceHTML(json.RawMessage(`{not json`)))
	if !strings.Contains(bad, "&#39;") && !strings.Contains(bad, "not json") {
		t.Errorf("fallback unexpected: %q", bad)
	}
}

func TestParseRejectsInvalidJSON(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestPrettyJSONFallbacks(t *testing.T) {
	if got := prettyJSON(json.RawMessage("  ")); got != "" {
		t.Errorf("blank evidence: want empty, got %q", got)
	}
	if got := prettyJSON(json.RawMessage("not-json")); got != "not-json" {
		t.Errorf("invalid evidence: want verbatim, got %q", got)
	}
}
