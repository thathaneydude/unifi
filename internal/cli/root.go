package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

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
	console       string
	insecure      bool
	format        string
	envFile       string
	fields        []string
	redact        bool
	limit         int
	site          string
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
	pf.StringVar(&gf.console, "console", "", "remote console by name, model, or id; resolved via 'consoles list'")
	pf.BoolVar(&gf.insecure, "insecure", false, "skip TLS verification (or UNIFI_INSECURE)")
	pf.StringVar(&gf.format, "format", "json", "output format: json|raw|human")
	pf.StringVar(&gf.envFile, "env-file", "", "path to a .env file (default ./.env if present)")
	pf.StringSliceVar(&gf.fields, "fields", nil, "keep only these dot-paths from each record (e.g. name,action.type)")
	pf.BoolVar(&gf.redact, "redact", false, "mask values under secret-like keys (key/secret/psk/token/…)")
	pf.IntVar(&gf.limit, "limit", 0, "cap a result array (or .data) to N items (0 = all)")
	pf.StringVar(&gf.site, "site", "", "site name, 'default', or id (or UNIFI_SITE); auto-selects when only one site")

	// resolveSite looks up a siteId from --site/UNIFI_SITE (or the sole site),
	// fetching the sites overview once per process. sitesPath is filled in below
	// from the network spec; the closure reads it at run time.
	var (
		sitesPath string
		siteOnce  sync.Once
		siteCache []site
		siteErr   error
	)
	resolveSite := func(ctx context.Context, conn *unifi.Conn, want string) (string, error) {
		if sitesPath == "" {
			return "", NewUsageError("cannot resolve site: sites operation not found; pass --siteId explicitly")
		}
		siteOnce.Do(func() {
			op := Operation{App: unifi.AppNetwork, ID: "getSiteOverviewPage", Method: http.MethodGet, Path: sitesPath}
			body, status, execErr := Execute(ctx, conn, op, Values{Path: map[string]string{}, Query: map[string]string{}})
			switch {
			case execErr != nil:
				siteErr = execErr
			case status < 200 || status >= 300:
				siteErr = NewAPIError("getSiteOverviewPage", status, body)
			default:
				siteCache, siteErr = parseSites(body)
			}
		})
		if siteErr != nil {
			return "", siteErr
		}
		return selectSite(siteCache, want)
	}

	// resolveConsoleID looks up a console id from --console (name|model|id),
	// fetching the account's consoles once per process (mirrors resolveSite).
	var (
		consolesOnce  sync.Once
		consolesCache []console
		consolesErr   error
	)
	resolveConsoleID := func(want string) (string, error) {
		consolesOnce.Do(func() {
			conn, err := resolveAccountConn(gf)
			if err != nil {
				consolesErr = err
				return
			}
			consolesCache, consolesErr = listConsoles(context.Background(), conn)
		})
		if consolesErr != nil {
			return "", consolesErr
		}
		return selectConsole(consolesCache, want)
	}

	deps := runDeps{
		connFn:      func(app unifi.App) (*unifi.Conn, error) { return resolveConn(gf, app, resolveConsoleID) },
		format:      func() Format { return formatFromFlags(gf) },
		render:      func() RenderOptions { return RenderOptions{Fields: gf.fields, Redact: gf.redact, Limit: gf.limit} },
		resolveSite: resolveSite,
		site: func() string {
			if gf.site != "" {
				return gf.site
			}
			return os.Getenv("UNIFI_SITE")
		},
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
		if app == unifi.AppNetwork {
			for _, op := range ops {
				if op.ID == "getSiteOverviewPage" {
					sitesPath = op.Path
				}
			}
		}
		appCmd := NewAppCommand(app, ops, deps)
		appCmd.AddCommand(newListOperationsCommand(app, ops, os.Stdout, deps.format))
		root.AddCommand(appCmd)
	}
	root.AddCommand(newSchemaCommand(cat, os.Stdout))
	root.AddCommand(newReportCommand(os.Stdout))
	root.AddCommand(newConsolesCommand(
		os.Stdout,
		func() (*unifi.Conn, error) { return resolveAccountConn(gf) },
		deps.format,
		deps.render,
	))
	return root, nil
}

// mergeConfig loads the .env file and merges environment + flags into a Config.
// It performs no validation and makes no network calls.
func mergeConfig(gf *globalFlags) (Config, error) {
	envPath, required := ".env", false
	if gf.envFile != "" {
		envPath, required = gf.envFile, true
	}
	if err := loadDotenv(envPath, required); err != nil {
		return Config{}, err
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
	return cfg, nil
}

// resolveFromFlags merges config and builds an app connection with no --console
// name resolution. Retained for the resolution it shares with tests.
func resolveFromFlags(gf *globalFlags, app unifi.App) (*unifi.Conn, error) {
	cfg, err := mergeConfig(gf)
	if err != nil {
		return nil, err
	}
	return ResolveConn(cfg, app)
}

// resolveConn builds an app connection, first resolving --console (name|model|id)
// to a concrete console id via resolveID when set. A raw --console-id keeps the
// fast path and makes no network call.
func resolveConn(gf *globalFlags, app unifi.App, resolveID func(string) (string, error)) (*unifi.Conn, error) {
	cfg, err := mergeConfig(gf)
	if err != nil {
		return nil, err
	}
	if gf.console != "" {
		if gf.consoleID != "" || gf.host != "" {
			return nil, NewUsageError("set only one of --console / --console-id / --host")
		}
		id, rerr := resolveID(gf.console)
		if rerr != nil {
			return nil, rerr
		}
		cfg.ConsoleID = id // --console always targets the cloud connector
	}
	return ResolveConn(cfg, app)
}

// resolveAccountConn builds an account-level (Site Manager API) connection for
// console enumeration. It needs only the shared API key — not --host/--console-id.
func resolveAccountConn(gf *globalFlags) (*unifi.Conn, error) {
	cfg, err := mergeConfig(gf)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey == "" {
		return nil, NewAuthError("missing API key: set --api-key (or UNIFI_API_KEY) to list consoles")
	}
	return unifi.Account(cfg.APIKey), nil
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
