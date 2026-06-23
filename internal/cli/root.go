package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/thathaneydude/unifi/unifi"
)

// global flags shared by all commands.
type globalFlags struct {
	apiKey        string
	networkAPIKey string
	protectAPIKey string
	host          string
	consoleID     string
	insecure      bool
	format        string
	envFile       string
	fields        []string
	redact        bool
	limit         int
}

// NewRootCommand assembles the full `unifi` command tree from the embedded specs.
func NewRootCommand() (*cobra.Command, error) {
	cat, err := LoadCatalog()
	if err != nil {
		return nil, err
	}

	gf := &globalFlags{}
	root := &cobra.Command{
		Use:           "unifi",
		Short:         "UniFi Network/Protect CLI (LLM-first)",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if _, err := parseFormat(gf.format); err != nil {
				return err
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return NewUsageError("no command given; run 'unifi <app> list-operations' to discover operations, or 'unifi schema --app <app>' (use --help for usage)")
			}
			return NewUsageError(fmt.Sprintf("unknown command %q; run 'unifi <app> list-operations' to discover operations", args[0]))
		},
	}
	pf := root.PersistentFlags()
	pf.StringVar(&gf.apiKey, "api-key", "", "API key shared by both apps (or UNIFI_API_KEY)")
	pf.StringVar(&gf.networkAPIKey, "network-api-key", "", "Network API key (or UNIFI_NETWORK_API_KEY); falls back to --api-key")
	pf.StringVar(&gf.protectAPIKey, "protect-api-key", "", "Protect API key (or UNIFI_PROTECT_API_KEY); falls back to --api-key")
	pf.StringVar(&gf.host, "host", "", "local console host (or UNIFI_HOST)")
	pf.StringVar(&gf.consoleID, "console-id", "", "remote console id (or UNIFI_CONSOLE_ID)")
	pf.BoolVar(&gf.insecure, "insecure", false, "skip TLS verification (or UNIFI_INSECURE)")
	pf.StringVar(&gf.format, "format", "json", "output format: json|raw|human")
	pf.StringVar(&gf.envFile, "env-file", "", "path to a .env file (default ./.env if present)")
	pf.StringSliceVar(&gf.fields, "fields", nil, "keep only these dot-paths from each record (e.g. name,action.type)")
	pf.BoolVar(&gf.redact, "redact", false, "mask values under secret-like keys (key/secret/psk/token/…)")
	pf.IntVar(&gf.limit, "limit", 0, "cap a result array (or .data) to N items (0 = all)")

	deps := runDeps{
		connFn: func(app unifi.App) (*unifi.Conn, error) { return resolveFromFlags(gf, app) },
		format: func() Format { return formatFromFlags(gf) },
		render: func() RenderOptions { return RenderOptions{Fields: gf.fields, Redact: gf.redact, Limit: gf.limit} },
		stdout: os.Stdout,
	}

	for _, app := range cat.Apps() {
		// Operation commands target each app's default (newest pinned) version.
		// Per-operation version selection will be added when more than one
		// version is pinned; `schema --api-version` already supports selection.
		doc, version, derr := cat.Doc(app, "")
		if derr != nil {
			return nil, derr
		}
		ops := OperationsFor(doc, app, version)
		appCmd := NewAppCommand(app, ops, deps)
		appCmd.AddCommand(newListOperationsCommand(app, ops, os.Stdout, deps.format))
		root.AddCommand(appCmd)
	}
	root.AddCommand(newSchemaCommand(cat, os.Stdout))
	return root, nil
}

func resolveFromFlags(gf *globalFlags, app unifi.App) (*unifi.Conn, error) {
	envPath, required := ".env", false
	if gf.envFile != "" {
		envPath, required = gf.envFile, true
	}
	if err := loadDotenv(envPath, required); err != nil {
		return nil, err
	}
	cfg := ConfigFromEnv()
	if gf.apiKey != "" {
		cfg.APIKey = gf.apiKey
	}
	if gf.networkAPIKey != "" {
		cfg.NetworkAPIKey = gf.networkAPIKey
	}
	if gf.protectAPIKey != "" {
		cfg.ProtectAPIKey = gf.protectAPIKey
	}
	if gf.host != "" {
		cfg.Host = gf.host
	}
	if gf.consoleID != "" {
		cfg.ConsoleID = gf.consoleID
	}
	if gf.insecure {
		cfg.Insecure = true
	}
	return ResolveConn(cfg, app)
}

// parseFormat validates and maps the --format value. An empty value defaults to
// JSON; anything outside json|raw|human is a usage error.
func parseFormat(s string) (Format, error) {
	switch s {
	case "", string(FormatJSON):
		return FormatJSON, nil
	case string(FormatRaw):
		return FormatRaw, nil
	case string(FormatHuman):
		return FormatHuman, nil
	default:
		return "", NewUsageError(fmt.Sprintf("invalid --format %q; want json|raw|human", s))
	}
}

func formatFromFlags(gf *globalFlags) Format {
	// Validity is enforced by PersistentPreRunE; default to JSON defensively.
	f, err := parseFormat(gf.format)
	if err != nil {
		return FormatJSON
	}
	return f
}

// RunRoot executes the command and maps any error to a stable exit code,
// writing a JSON error envelope to stderr.
func RunRoot(root *cobra.Command) int {
	err := root.Execute()
	if err == nil {
		return exitOK
	}
	var cerr *CLIError
	if errors.As(err, &cerr) {
		_, _ = os.Stderr.WriteString(cerr.JSON() + "\n")
		return cerr.ExitCode()
	}
	// Unknown errors (e.g. cobra flag/arg parsing) are usage errors.
	_, _ = os.Stderr.WriteString(NewUsageError(err.Error()).JSON() + "\n")
	return exitUsage
}

// Main is the binary entrypoint. (Named Main rather than Execute to avoid
// shadowing the Execute function in request.go which runs an HTTP operation.)
func Main() int {
	root, err := NewRootCommand()
	if err != nil {
		_, _ = os.Stderr.WriteString(NewUsageError(err.Error()).JSON() + "\n")
		return exitUsage
	}
	return RunRoot(root)
}
