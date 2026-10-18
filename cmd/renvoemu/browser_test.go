package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBrowserCompilerProvenanceIncludesSiblingBackend(t *testing.T) {
	dir := t.TempDir()
	frontend := filepath.Join(dir, "renvo-bootstrap")
	backend := filepath.Join(dir, "renvo-backend")
	if err := os.WriteFile(frontend, []byte("unchanged frontend"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backend, []byte("old backend"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := browserCompilerInputHashes(frontend)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backend, []byte("fixed backend"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := browserCompilerInputHashes(frontend)
	if err != nil {
		t.Fatal(err)
	}
	if before.FrontendHash != after.FrontendHash || before.BackendHash == after.BackendHash || before == after || after.Backend != backend {
		t.Fatal("frontend identity hid a changed sibling backend", before, after)
	}
}
