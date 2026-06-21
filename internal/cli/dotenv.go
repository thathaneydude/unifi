package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// parseDotenv reads KEY=VALUE lines from r and returns the parsed pairs. Blank
// lines and lines beginning with '#' are ignored, and one optional leading
// "export " is stripped. Each line splits on the first '='; the key must match
// [A-Za-z_][A-Za-z0-9_]*. A value wrapped in matching single or double quotes
// keeps its inner text literally; an unquoted value is trimmed of surrounding
// whitespace. There is no inline-comment stripping and no escape expansion.
// Malformed lines produce an error citing the 1-based line number.
func parseDotenv(r io.Reader) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			return nil, fmt.Errorf("line %d: missing '=' in %q", n, line)
		}
		key := strings.TrimSpace(line[:eq])
		if !validKey(key) {
			return nil, fmt.Errorf("line %d: invalid key %q", n, key)
		}
		out[key] = unquote(strings.TrimSpace(line[eq+1:]))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// validKey reports whether s matches [A-Za-z_][A-Za-z0-9_]*.
func validKey(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_':
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return true
}

// unquote strips one layer of matching single or double quotes, if present.
func unquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
