package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// repoRootOnce caches the result of the upward walk so that repeated calls
// to repoRoot() (from main, Sync, and Augment) pay the filesystem cost only once.
var (
	repoRootOnce  sync.Once
	repoRootValue string
	repoRootErr   error
)

// repoRoot returns the repository root directory (the ancestor that contains
// "specs/overlays"). The result is computed once and cached for the lifetime
// of the process.
func repoRoot() (string, error) {
	repoRootOnce.Do(func() {
		repoRootValue, repoRootErr = findRepoRoot()
	})
	return repoRootValue, repoRootErr
}

// findRepoRoot walks upward from the current working directory looking for a
// directory that contains "specs/overlays". This makes both `go test
// ./tools/specgen/` (cwd = tools/specgen) and `just sync` (cwd = repo root)
// resolve the correct paths.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "specs", "overlays")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find repo root (no specs/overlays directory found in ancestor hierarchy)")
}
