package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/thathaneydude/unifi/unifi"
)

// Config is the resolved connection configuration (flags already merged).
//
// Network and Protect mint their own integration API keys, so a key may be set
// per app. APIKey is the shared fallback used for whichever app lacks a specific
// key, preserving the original single-key behaviour.
type Config struct {
	APIKey        string // shared fallback (--api-key / UNIFI_API_KEY)
	NetworkAPIKey string // --network-api-key / UNIFI_NETWORK_API_KEY
	ProtectAPIKey string // --protect-api-key / UNIFI_PROTECT_API_KEY
	Host          string // set => local transport
	ConsoleID     string // set => remote transport
	Insecure      bool
}

// ConfigFromEnv reads defaults from the environment. Flags override these.
func ConfigFromEnv() Config {
	return Config{
		APIKey:        os.Getenv("UNIFI_API_KEY"),
		NetworkAPIKey: os.Getenv("UNIFI_NETWORK_API_KEY"),
		ProtectAPIKey: os.Getenv("UNIFI_PROTECT_API_KEY"),
		Host:          os.Getenv("UNIFI_HOST"),
		ConsoleID:     os.Getenv("UNIFI_CONSOLE_ID"),
		Insecure:      os.Getenv("UNIFI_INSECURE") != "",
	}
}

// keyForApp returns the API key to use for app, preferring the app-specific key
// and falling back to the shared key.
func (c Config) keyForApp(app unifi.App) string {
	switch app {
	case unifi.AppNetwork:
		if c.NetworkAPIKey != "" {
			return c.NetworkAPIKey
		}
	case unifi.AppProtect:
		if c.ProtectAPIKey != "" {
			return c.ProtectAPIKey
		}
	}
	return c.APIKey
}

// ResolveConn validates the config and constructs an authenticated connection
// for the given app, selecting that app's API key.
func ResolveConn(cfg Config, app unifi.App) (*unifi.Conn, error) {
	key := cfg.keyForApp(app)
	if key == "" {
		return nil, NewAuthError(fmt.Sprintf(
			"missing API key for %s: set --%s-api-key (or UNIFI_%s_API_KEY), or --api-key (or UNIFI_API_KEY)",
			app, app, strings.ToUpper(string(app))))
	}
	switch {
	case cfg.Host != "" && cfg.ConsoleID != "":
		return nil, NewAuthError("ambiguous transport: set only one of --host / --console-id")
	case cfg.Host != "":
		return unifi.Local(cfg.Host, key, localOpts(cfg)...), nil
	case cfg.ConsoleID != "":
		return unifi.Remote(cfg.ConsoleID, key), nil
	default:
		return nil, NewAuthError("no transport: set --host (or UNIFI_HOST) or --console-id (or UNIFI_CONSOLE_ID)")
	}
}

func localOpts(cfg Config) []unifi.Option {
	var opts []unifi.Option
	if cfg.Insecure {
		opts = append(opts, unifi.WithInsecureSkipVerify())
	}
	return opts
}
