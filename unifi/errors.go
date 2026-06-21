package unifi

import (
	"encoding/json"
	"fmt"
)

// APIError is a typed error returned for non-2xx UniFi API responses.
type APIError struct {
	Operation string
	Status    int
	Message   string
	Body      []byte
}

type errorEnvelope struct {
	StatusCode int    `json:"statusCode"`
	StatusName string `json:"statusName"`
	Message    string `json:"message"`
}

// NewAPIError builds an APIError, parsing the UniFi error envelope when present.
func NewAPIError(operation string, status int, body []byte) *APIError {
	e := &APIError{Operation: operation, Status: status, Body: body}
	var env errorEnvelope
	if json.Unmarshal(body, &env) == nil && env.Message != "" {
		e.Message = env.Message
	} else {
		e.Message = string(body)
	}
	return e
}

func (e *APIError) Error() string {
	return fmt.Sprintf("unifi: %s failed with HTTP %d: %s", e.Operation, e.Status, e.Message)
}

// Unwrap returns nil. It exists to future-proof APIError for use in error
// chains via errors.As/errors.Is without wrapping a cause today.
func (e *APIError) Unwrap() error { return nil }
