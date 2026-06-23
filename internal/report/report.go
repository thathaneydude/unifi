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
	"html/template"
	"io"
	"strings"
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
	"prettyJSON": prettyJSON,
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
