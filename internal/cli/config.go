package cli

import (
	"os"

	"github.com/thathaneydude/unifi/unifi"
)

// Config is the resolved connection configuration (flags already merged).
type Config struct {
	APIKey    string
	Host      string // set => local transport
	ConsoleID string // set => remote transport
	Insecure  bool
}

// ConfigFromEnv reads defaults from the environment. Flags override these.
func ConfigFromEnv() Config {
	return Config{
		APIKey:    os.Getenv("UNIFI_API_KEY"),
		Host:      os.Getenv("UNIFI_HOST"),
		ConsoleID: os.Getenv("UNIFI_CONSOLE_ID"),
		Insecure:  os.Getenv("UNIFI_INSECURE") != "",
	}
}

// ResolveConn validates the config and constructs an authenticated connection.
func ResolveConn(cfg Config) (*unifi.Conn, error) {
	if cfg.APIKey == "" {
		return nil, NewAuthError("missing API key: set --api-key or UNIFI_API_KEY")
	}
	switch {
	case cfg.Host != "" && cfg.ConsoleID != "":
		return nil, NewAuthError("ambiguous transport: set only one of --host / --console-id")
	case cfg.Host != "":
		return unifi.Local(cfg.Host, cfg.APIKey, localOpts(cfg)...), nil
	case cfg.ConsoleID != "":
		return unifi.Remote(cfg.ConsoleID, cfg.APIKey), nil
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
