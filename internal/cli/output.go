package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Format selects how a successful result is rendered to stdout.
type Format string

const (
	FormatJSON  Format = "json"  // pretty-printed JSON (default)
	FormatRaw   Format = "raw"   // response bytes verbatim
	FormatHuman Format = "human" // best-effort human view
)

// WriteResult renders the API response body to w in the chosen format.
func WriteResult(w io.Writer, format Format, body []byte) error {
	switch format {
	case FormatRaw:
		return writeRaw(w, body)
	case FormatHuman:
		return writeHuman(w, body)
	case FormatJSON, "":
		return writeJSON(w, body)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

// writeRaw emits the response body verbatim, guaranteeing exactly one trailing
// newline so an empty 2xx body is still discernible (one newline, not zero
// bytes) and piped output never runs into the next shell prompt.
func writeRaw(w io.Writer, body []byte) error {
	if _, err := w.Write(body); err != nil {
		return err
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, err := w.Write([]byte{'\n'})
		return err
	}
	return nil
}

func writeJSON(w io.Writer, body []byte) error {
	if len(bytes.TrimSpace(body)) == 0 {
		_, err := fmt.Fprintln(w, "null")
		return err
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, body, "", "  "); err != nil {
		// Not JSON (unexpected): emit verbatim so nothing is lost.
		_, err := fmt.Fprintln(w, string(body))
		return err
	}
	_, err := fmt.Fprintln(w, indented.String())
	return err
}

// writeHuman renders a top-level JSON array as a simple line list, falling back
// to pretty JSON for anything else. Kept intentionally small; --raw/--json are
// the contract, --human is a convenience.
func writeHuman(w io.Writer, body []byte) error {
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err != nil {
		return writeJSON(w, body)
	}
	for _, item := range arr {
		if _, err := fmt.Fprintln(w, string(item)); err != nil {
			return err
		}
	}
	return nil
}

func writeJSONValue(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}
