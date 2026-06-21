package cli

import (
	"encoding/json"

	"github.com/thathaneydude/unifi/unifi"
)

// Exit codes are part of the CLI's contract; do not renumber.
const (
	exitOK        = 0
	exitUsage     = 1
	exitAuth      = 2
	exitAPI       = 3
	exitTransport = 4
)

// CLIError is a structured, agent-parseable failure.
type CLIError struct {
	code      int
	Operation string          `json:"operation,omitempty"`
	Status    int             `json:"status,omitempty"`
	Message   string          `json:"message,omitempty"`
	APIError  json.RawMessage `json:"apiError,omitempty"`
	Hint      string          `json:"hint,omitempty"`
}

func (e *CLIError) Error() string { return e.Message }
func (e *CLIError) ExitCode() int { return e.code }

// JSON renders the error as the {"error": ...} envelope written to stderr.
func (e *CLIError) JSON() string {
	wrapper := struct {
		Error *CLIError `json:"error"`
	}{Error: e}
	b, err := json.Marshal(wrapper)
	if err != nil {
		return `{"error":{"message":"failed to render error"}}`
	}
	return string(b)
}

// NewUsageError reports a bad invocation (exit 1).
func NewUsageError(msg string) *CLIError {
	return &CLIError{code: exitUsage, Message: msg}
}

// NewAuthError reports missing or invalid credentials/config (exit 2).
func NewAuthError(msg string) *CLIError {
	return &CLIError{code: exitAuth, Message: msg}
}

// NewAPIError reports a non-2xx API response (exit 3). body is the raw response.
// It reuses the SDK's envelope parser so the server's real message (whether a
// parsed UniFi error envelope or a plain/HTML body) reaches the caller, rather
// than a generic placeholder.
func NewAPIError(operation string, status int, body []byte) *CLIError {
	apiErr := unifi.NewAPIError(operation, status, body)
	msg := apiErr.Message
	if msg == "" {
		msg = "API returned a non-2xx response"
	}
	e := &CLIError{
		code:      exitAPI,
		Operation: operation,
		Status:    status,
		Message:   msg,
	}
	if json.Valid(body) {
		e.APIError = json.RawMessage(body)
	}
	return e
}

// NewTransportError reports a network/transport failure (exit 4).
func NewTransportError(operation, msg string) *CLIError {
	return &CLIError{
		code:      exitTransport,
		Operation: operation,
		Message:   msg,
	}
}
