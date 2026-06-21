package cli

import (
	"encoding/json"
	"fmt"
	"io"

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

// newListOperationsCommand returns `unifi <app> list-operations`.
func newListOperationsCommand(app unifi.App, ops []Operation, out io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list-operations",
		Short: fmt.Sprintf("List all %s operations as JSON", app),
		RunE: func(_ *cobra.Command, _ []string) error {
			return WriteOperationIndex(out, ops)
		},
	}
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
