package main

import (
	"fmt"
	"os"

	"github.com/thathaneydude/unifi/unifi"
)

// local demonstrates connecting to a UniFi console's local API.
//
// Usage:
//
//	UNIFI_HOST=192.168.1.1 UNIFI_API_KEY=<key> go run ./examples/local
func main() {
	host := os.Getenv("UNIFI_HOST")
	apiKey := os.Getenv("UNIFI_API_KEY")

	c := unifi.Local(host, apiKey, unifi.WithInsecureSkipVerify())

	// Retrieve the latest Network client using the pinned version.
	_, err := c.Network()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create network client: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Network base URL:", c.NetworkBaseURL())
}
