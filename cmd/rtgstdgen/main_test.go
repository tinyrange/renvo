package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGeneratedBundleIsCurrent(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	files, dirs, err := collect(filepath.Join(root, "rtg", "std"))
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}
	want, err := generateSource(files, dirs)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	path := filepath.Join(root, "rtg", "internal", "driver", "std_bundle_generated.go")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated bundle failed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("standard library bundle is stale; run go generate ./rtg/internal/driver")
	}
}
