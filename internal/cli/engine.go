package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/thathaneydude/unifi/unifi"
)

// runDeps carries everything an operation's RunE needs, resolved lazily so the
// command tree can be built without credentials present.
type runDeps struct {
	connFn func() (*unifi.Conn, error) // resolves config -> Conn at run time
	format func() Format               // global output format
	stdout io.Writer                   // results go here; errors are rendered in RunRoot
}

// NewAppCommand builds `unifi <app>` with one subcommand per operation. deps may
// be nil for structural tests; RunE guards against a nil deps.
func NewAppCommand(app unifi.App, ops []Operation, deps ...runDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   string(app),
		Short: fmt.Sprintf("UniFi %s API operations", app),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return NewUsageError(fmt.Sprintf("no operation given; run 'unifi %s list-operations' to discover operations", app))
			}
			return NewUsageError(fmt.Sprintf("unknown operation %q; run 'unifi %s list-operations'", args[0], app))
		},
	}
	var d *runDeps
	if len(deps) > 0 {
		d = &deps[0]
	}
	for _, op := range ops {
		cmd.AddCommand(newOperationCommand(op, d))
	}
	return cmd
}

// paramBinding ties a spec parameter (the Values map key) to the cobra flag it
// was registered under and the destination it writes to. The flag name can
// differ from the param name when a path and query param share a name.
type paramBinding struct {
	name string // spec parameter name (the Values map key)
	flag string // registered cobra flag name
	dst  *string
}

func newOperationCommand(op Operation, d *runDeps) *cobra.Command {
	var (
		bodyInline string
		bodyFile   string
		dryRun     bool
		confirm    bool
	)
	var pathBindings, queryBindings []paramBinding

	sub := &cobra.Command{
		Use:   op.ID,
		Short: op.Summary,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if d == nil {
				return NewUsageError("command not wired for execution")
			}
			// Inclusion is decided by whether the flag was set, not by value
			// emptiness, so a deliberately empty value is sent and a required
			// path param is never silently dropped.
			vals := Values{Path: map[string]string{}, Query: map[string]string{}}
			for _, b := range pathBindings {
				if cmd.Flags().Changed(b.flag) {
					vals.Path[b.name] = *b.dst
				}
			}
			for _, b := range queryBindings {
				if cmd.Flags().Changed(b.flag) {
					vals.Query[b.name] = *b.dst
				}
			}
			return runOperation(cmd.Context(), *d, op, vals, bodyInline, bodyFile, dryRun, confirm)
		},
	}

	pathNames := map[string]bool{}
	for _, p := range op.PathParams {
		pathNames[p.Name] = true
	}
	for _, p := range op.PathParams {
		dst := new(string)
		sub.Flags().StringVar(dst, p.Name, "", "path: "+p.Description)
		if p.Required {
			// Cannot fail: the flag was just registered above.
			_ = sub.MarkFlagRequired(p.Name)
		}
		pathBindings = append(pathBindings, paramBinding{name: p.Name, flag: p.Name, dst: dst})
	}
	for _, p := range op.QueryParams {
		dst := new(string)
		// A path and query param may legally share a name (location
		// disambiguates). Registering the same flag name twice panics pflag at
		// tree-build, so disambiguate the query flag on collision.
		flagName := p.Name
		if pathNames[p.Name] {
			flagName = "query-" + p.Name
		}
		sub.Flags().StringVar(dst, flagName, "", "query: "+p.Description)
		queryBindings = append(queryBindings, paramBinding{name: p.Name, flag: flagName, dst: dst})
	}
	if op.HasBody() {
		sub.Flags().StringVar(&bodyInline, "body", "", "request body as a JSON string")
		sub.Flags().StringVar(&bodyFile, "body-file", "", "path to a JSON request body file")
	}
	if op.Mutating() {
		sub.Flags().BoolVar(&dryRun, "dry-run", false, "print the request without sending it")
		sub.Flags().BoolVar(&confirm, "confirm", false, "required to execute a mutating operation")
	}
	return sub
}

func runOperation(
	ctx context.Context, d runDeps, op Operation,
	vals Values, bodyInline, bodyFile string,
	dryRun, confirm bool,
) error {
	body, err := resolveBody(bodyInline, bodyFile)
	if err != nil {
		return err
	}
	vals.Body = body

	// Enforce the confirm gate before resolving credentials so a mis-typed
	// destructive command never triggers credential resolution.
	if op.Mutating() && !dryRun && !confirm {
		return NewUsageError(fmt.Sprintf("%s is a %s operation; pass --confirm to execute (or --dry-run to preview)", op.ID, op.Method))
	}

	conn, err := d.connFn()
	if err != nil {
		return err
	}

	if op.Mutating() && dryRun {
		return writeDryRun(ctx, d, conn, op, vals)
	}

	respBody, status, err := Execute(ctx, conn, op, vals)
	if err != nil {
		return err
	}
	if status < 200 || status >= 300 {
		return NewAPIError(op.ID, status, respBody)
	}
	return WriteResult(d.stdout, d.format(), respBody)
}

func resolveBody(inline, file string) ([]byte, error) {
	switch {
	case inline != "" && file != "":
		return nil, NewUsageError("set only one of --body / --body-file")
	case inline != "":
		return []byte(inline), nil
	case file != "":
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, NewUsageError("read --body-file: " + err.Error())
		}
		return b, nil
	default:
		return nil, nil
	}
}

func writeDryRun(ctx context.Context, d runDeps, conn *unifi.Conn, op Operation, vals Values) error {
	req, err := BuildRequest(ctx, conn, op, vals)
	if err != nil {
		return NewUsageError(err.Error())
	}
	preview := map[string]any{
		"method": req.Method,
		"url":    req.URL.String(),
		"body":   string(vals.Body),
	}
	return writeJSONValue(d.stdout, map[string]any{"dryRun": preview})
}
