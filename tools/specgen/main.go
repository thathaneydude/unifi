package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		log.Fatalf("cannot locate repo root: %v", err)
	}

	versionsPath := filepath.Join(root, "specs", "versions.yaml")
	data, err := os.ReadFile(versionsPath)
	if err != nil {
		log.Fatalf("read %s: %v", versionsPath, err)
	}

	cfg, err := LoadConfig(data)
	if err != nil {
		log.Fatalf("parse versions.yaml: %v", err)
	}

	if err := Sync(cfg); err != nil {
		log.Fatalf("sync: %v", err)
	}

	fmt.Println("sync complete")
}
