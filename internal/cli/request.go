package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/thathaneydude/unifi/unifi"
)

// Values holds the user-supplied inputs for a single operation invocation.
type Values struct {
	Path  map[string]string
	Query map[string]string
	Body  []byte // nil when no request body
}

// BuildRequest turns an operation + values into an *http.Request targeting the
// connection's base URL for the operation's app. It does not apply auth.
func BuildRequest(ctx context.Context, conn *unifi.Conn, op Operation, vals Values) (*http.Request, error) {
	p := op.Path
	for name, v := range vals.Path {
		p = strings.ReplaceAll(p, "{"+name+"}", url.PathEscape(v))
	}
	target := strings.TrimRight(conn.BaseURL(op.App), "/") + p

	var body io.Reader
	if vals.Body != nil {
		body = bytes.NewReader(vals.Body)
	}
	req, err := http.NewRequestWithContext(ctx, op.Method, target, body)
	if err != nil {
		return nil, err
	}
	if len(vals.Query) > 0 {
		q := url.Values{}
		for name, v := range vals.Query {
			q.Set(name, v)
		}
		req.URL.RawQuery = q.Encode()
	}
	if vals.Body != nil {
		ct := op.BodyMediaType
		if ct == "" {
			// Body supplied for an op the spec declared none for: default to JSON.
			ct = "application/json"
		}
		req.Header.Set("Content-Type", ct)
	}
	return req, nil
}

// Execute builds, authenticates, and runs the request, returning the response
// body and status. Transport failures are returned as *CLIError (exit 4).
func Execute(ctx context.Context, conn *unifi.Conn, op Operation, vals Values) ([]byte, int, error) {
	req, err := BuildRequest(ctx, conn, op, vals)
	if err != nil {
		return nil, 0, NewUsageError(err.Error())
	}
	if editErr := conn.RequestEditor()(ctx, req); editErr != nil {
		return nil, 0, NewTransportError(op.ID, editErr.Error())
	}
	resp, err := conn.HTTPClient().Do(req)
	if err != nil {
		return nil, 0, NewTransportError(op.ID, err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, NewTransportError(op.ID, err.Error())
	}
	return body, resp.StatusCode, nil
}
