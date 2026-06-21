package unifi

// internal_test.go — white-box tests that need access to unexported fields.
// Package: unifi (not unifi_test) so it can manipulate Conn internals.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networklatest "github.com/thathaneydude/unifi/lib/network/v10_3_58"
)

// TestNetworkClientIntegration_HeaderWiring verifies that Conn.Network() wires
// the X-API-KEY header into requests sent to the generated client. It does this
// by overriding the unexported prefix field to point at an httptest server,
// then calling GetInfoWithResponse and asserting the received header.
func TestNetworkClientIntegration_HeaderWiring(t *testing.T) {
	var (
		receivedKey string
		receivedUA  string
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKey = r.Header.Get("X-API-KEY")
		receivedUA = r.Header.Get("User-Agent")
		// Return a minimal valid GetInfo response.
		body, _ := json.Marshal(struct {
			ApplicationVersion string `json:"applicationVersion"`
		}{"10.3.58"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	// Build a Conn and override the prefix function to point at the test server.
	// This is the white-box seam: prefix is unexported but accessible here.
	c := newConn("test-api-key", func(app App) string {
		return srv.URL // point directly at the test server root
	}, WithHTTPClient(srv.Client()), WithUserAgent("test-agent/1"))

	netClient, err := networklatest.NewClientWithResponses(
		c.NetworkBaseURL(),
		networklatest.WithRequestEditorFn(networklatest.RequestEditorFn(c.RequestEditor())),
		networklatest.WithHTTPClient(c.httpClient),
	)
	if err != nil {
		t.Fatalf("NewClientWithResponses: %v", err)
	}

	resp, err := netClient.GetInfoWithResponse(context.Background())
	if err != nil {
		t.Fatalf("GetInfoWithResponse: %v", err)
	}
	if resp.StatusCode() != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode())
	}
	if receivedKey != "test-api-key" {
		t.Errorf("X-API-KEY header: got %q, want %q", receivedKey, "test-api-key")
	}
	if receivedUA != "test-agent/1" {
		t.Errorf("User-Agent header: got %q, want %q", receivedUA, "test-agent/1")
	}
	if resp.JSON200 == nil {
		t.Error("expected JSON200 to be parsed")
	} else if resp.JSON200.ApplicationVersion != "10.3.58" {
		t.Errorf("ApplicationVersion: got %q, want %q", resp.JSON200.ApplicationVersion, "10.3.58")
	}
}
