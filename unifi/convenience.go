package unifi

import (
	networklatest "github.com/thathaneydude/unifi/lib/network/v10_3_58"
	protectlatest "github.com/thathaneydude/unifi/lib/protect/v7_1_46"
)

// Network returns a client for the latest pinned Network version.
func (c *Conn) Network() (*networklatest.ClientWithResponses, error) {
	return networklatest.NewClientWithResponses(
		c.NetworkBaseURL(),
		networklatest.WithRequestEditorFn(networklatest.RequestEditorFn(c.RequestEditor())),
		networklatest.WithHTTPClient(c.httpClient),
	)
}

// Protect returns a client for the latest pinned Protect version.
func (c *Conn) Protect() (*protectlatest.ClientWithResponses, error) {
	return protectlatest.NewClientWithResponses(
		c.ProtectBaseURL(),
		protectlatest.WithRequestEditorFn(protectlatest.RequestEditorFn(c.RequestEditor())),
		protectlatest.WithHTTPClient(c.httpClient),
	)
}
