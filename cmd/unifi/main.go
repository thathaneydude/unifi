// Command unifi is the LLM-first CLI for the UniFi Network and Protect APIs.
package main

import (
	"os"

	"github.com/thathaneydude/unifi/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
