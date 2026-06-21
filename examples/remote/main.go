package main

import (
	"fmt"
	"os"

	networkv10_3_58 "github.com/thathaneydude/unifi/lib/network/v10_3_58"
	"github.com/thathaneydude/unifi/unifi"
)

// remote demonstrates connecting via the UniFi cloud connector and constructing
// a specific versioned client directly, instead of using the default via c.Network().
// This is useful when you need a specific API version that differs from the pinned default.
//
// Usage:
//
//	UNIFI_CONSOLE_ID=<id> UNIFI_SM_KEY=<key> go run ./examples/remote
func main() {
	consoleID := os.Getenv("UNIFI_CONSOLE_ID")
	smKey := os.Getenv("UNIFI_SM_KEY")

	c := unifi.Remote(consoleID, smKey)

	// Construct a specific coexisting version client directly instead of
	// using c.Network() (which returns the latest default pinned version).
	netClient, err := networkv10_3_58.NewClientWithResponses(
		c.NetworkBaseURL(),
		networkv10_3_58.WithRequestEditorFn(networkv10_3_58.RequestEditorFn(c.RequestEditor())),
		networkv10_3_58.WithHTTPClient(c.HTTPClient()),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create network client: %v\n", err)
		os.Exit(1)
	}

	_ = netClient // use the client for API calls here

	fmt.Println("Network base URL:", c.NetworkBaseURL())
}
