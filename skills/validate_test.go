package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	orchestrator   = "unifi-security-assessment"
	readOnlyMarker = "This skill is strictly read-only"
)

var semverRE = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// subSkills is grown by later tasks as each domain skill is added.
var subSkills = []string{}

func parseFrontmatter(t *testing.T, path string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		t.Fatalf("%s: missing opening frontmatter delimiter", path)
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		t.Fatalf("%s: missing closing frontmatter delimiter", path)
	}
	fm := map[string]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fm[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return fm
}

func validateSkill(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "SKILL.md")
	fm := parseFrontmatter(t, path)
	if got := fm["name"]; got != dir {
		t.Errorf("%s: frontmatter name %q != directory %q", path, got, dir)
	}
	if fm["description"] == "" {
		t.Errorf("%s: missing description", path)
	}
	if !semverRE.MatchString(fm["version"]) {
		t.Errorf("%s: version %q is not MAJOR.MINOR.PATCH", path, fm["version"])
	}
}

func assertReadOnly(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read %s/SKILL.md: %v", dir, err)
	}
	if !strings.Contains(string(data), readOnlyMarker) {
		t.Errorf("%s/SKILL.md: missing read-only marker %q", dir, readOnlyMarker)
	}
}

func TestOrchestratorSkill(t *testing.T) {
	validateSkill(t, orchestrator)
	assertReadOnly(t, orchestrator)

	for _, ref := range []string{
		"references/severity-rubric.md",
		"references/report-template.md",
	} {
		if _, err := os.Stat(filepath.Join(orchestrator, ref)); err != nil {
			t.Errorf("missing shared reference %s: %v", ref, err)
		}
	}

	body, err := os.ReadFile(filepath.Join(orchestrator, "SKILL.md"))
	if err != nil {
		t.Fatalf("read orchestrator SKILL.md: %v", err)
	}
	for _, s := range subSkills {
		if !strings.Contains(string(body), s) {
			t.Errorf("orchestrator SKILL.md does not reference sub-skill %q", s)
		}
	}
}

func TestSubSkills(t *testing.T) {
	for _, dir := range subSkills {
		t.Run(dir, func(t *testing.T) {
			validateSkill(t, dir)
			assertReadOnly(t, dir)
		})
	}
}
