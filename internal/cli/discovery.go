package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/thathaneydude/unifi/unifi"
)

type opIndexEntry struct {
	OperationID string   `json:"operationId"`
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Summary     string   `json:"summary,omitempty"`
	Params      []string `json:"params,omitempty"`
}

// WriteOperationIndex writes the operations as a JSON array for agents.
func WriteOperationIndex(w io.Writer, ops []Operation) error {
	entries := make([]opIndexEntry, 0, len(ops))
	for _, op := range ops {
		e := opIndexEntry{OperationID: op.ID, Method: op.Method, Path: op.Path, Summary: op.Summary}
		for _, p := range op.PathParams {
			e.Params = append(e.Params, "path:"+p.Name)
		}
		for _, p := range op.QueryParams {
			e.Params = append(e.Params, "query:"+p.Name)
		}
		entries = append(entries, e)
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

// requiredParamNames returns the names of an operation's required path and query
// parameters, path first, in declaration order.
func requiredParamNames(op Operation) []string {
	var names []string
	for _, p := range op.PathParams {
		if p.Required {
			names = append(names, p.Name)
		}
	}
	for _, p := range op.QueryParams {
		if p.Required {
			names = append(names, p.Name)
		}
	}
	return names
}

// WriteOperationIDs writes one operationId per line — the terse form agents want
// for discovery without parsing JSON.
func WriteOperationIDs(w io.Writer, ops []Operation) error {
	for _, op := range ops {
		if _, err := fmt.Fprintln(w, op.ID); err != nil {
			return err
		}
	}
	return nil
}

// WriteOperationIndexHuman writes an aligned table: METHOD, operationId, summary,
// and required parameters.
func WriteOperationIndexHuman(w io.Writer, ops []Operation) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	for _, op := range ops {
		req := ""
		if names := requiredParamNames(op); len(names) > 0 {
			req = "[requires: " + strings.Join(names, ", ") + "]"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", op.Method, op.ID, op.Summary, req); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// newListOperationsCommand returns `unifi <app> list-operations`. Output honors
// the global --format (json default, human table) plus an --ids shortcut.
func newListOperationsCommand(app unifi.App, ops []Operation, out io.Writer, format func() Format) *cobra.Command {
	var ids bool
	cmd := &cobra.Command{
		Use:   "list-operations",
		Short: fmt.Sprintf("List all %s operations (JSON; --format human or --ids for terse)", app),
		RunE: func(_ *cobra.Command, _ []string) error {
			switch {
			case ids:
				return WriteOperationIDs(out, ops)
			case format != nil && format() == FormatHuman:
				return WriteOperationIndexHuman(out, ops)
			default:
				return WriteOperationIndex(out, ops)
			}
		},
	}
	cmd.Flags().BoolVar(&ids, "ids", false, "print only operation ids, one per line")
	return cmd
}

// newSchemaCommand returns `unifi schema` which dumps the embedded spec JSON.
func newSchemaCommand(cat *Catalog, out io.Writer) *cobra.Command {
	var appFlag, versionFlag string
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the embedded OpenAPI spec as JSON",
		RunE: func(_ *cobra.Command, _ []string) error {
			if appFlag == "" {
				return NewUsageError("schema requires --app network|protect")
			}
			doc, _, err := cat.Doc(unifi.App(appFlag), versionFlag)
			if err != nil {
				return NewUsageError(err.Error())
			}
			b, err := doc.MarshalJSON()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, string(b))
			return err
		},
	}
	cmd.Flags().StringVar(&appFlag, "app", "", "app: network|protect")
	cmd.Flags().StringVar(&versionFlag, "api-version", "", "spec version (default: app default)")
	return cmd
}
