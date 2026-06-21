package unifi

import (
	"context"
	"net/http"
)

// NetworkBaseURL returns the Network integration base URL for this connection.
func (c *Conn) NetworkBaseURL() string { return c.prefix(AppNetwork) }

// ProtectBaseURL returns the Protect integration base URL for this connection.
func (c *Conn) ProtectBaseURL() string { return c.prefix(AppProtect) }

// BaseURL returns the integration base URL for the given app.
func (c *Conn) BaseURL(app App) string { return c.prefix(app) }

// RequestEditor returns the cached function that injects auth and user-agent
// headers. Its signature matches oapi-codegen's RequestEditorFn.
func (c *Conn) RequestEditor() func(ctx context.Context, req *http.Request) error {
	return c.requestEditor
}
