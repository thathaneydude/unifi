package unifi

import (
	"context"
	"net/http"
)

// App identifies a UniFi application.
type App string

const (
	AppNetwork App = "network"
	AppProtect App = "protect"
)

// Conn is an authenticated connection to a UniFi console (local or remote).
type Conn struct {
	apiKey        string
	prefix        func(App) string
	httpClient    *http.Client
	userAgent     string
	requestEditor func(context.Context, *http.Request) error
}

// HTTPClient returns the underlying HTTP client.
func (c *Conn) HTTPClient() *http.Client { return c.httpClient }

// UserAgent returns the configured User-Agent.
func (c *Conn) UserAgent() string { return c.userAgent }

func newConn(apiKey string, prefix func(App) string, opts ...Option) *Conn {
	cfg := &config{userAgent: defaultUserAgent}
	for _, o := range opts {
		o(cfg)
	}
	c := &Conn{
		apiKey:     apiKey,
		prefix:     prefix,
		httpClient: cfg.resolveHTTPClient(),
		userAgent:  cfg.userAgent,
	}
	// Cache the request-editor closure once to avoid a per-call allocation.
	c.requestEditor = func(_ context.Context, req *http.Request) error {
		req.Header.Set("X-API-KEY", c.apiKey)
		req.Header.Set("User-Agent", c.userAgent)
		return nil
	}
	return c
}

// Local builds a connection to a console's local API.
func Local(host, apiKey string, opts ...Option) *Conn {
	return newConn(apiKey, func(app App) string {
		return "https://" + host + "/proxy/" + string(app) + "/integration"
	}, opts...)
}

// Remote builds a connection via the UniFi cloud connector.
func Remote(consoleID, apiKey string, opts ...Option) *Conn {
	return newConn(apiKey, func(app App) string {
		return "https://api.ui.com/v1/connector/consoles/" + consoleID + "/" + string(app) + "/integration"
	}, opts...)
}
