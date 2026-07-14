//go:build rtg_bundle && !rtg

package driver

import (
	"bytes"
	"testing"
)

func TestBundledStandardLibraryFS(t *testing.T) {
	fs := OSFS{}
	data, ok := fs.ReadFile("/std/strings/strings.go")
	if !ok || !bytes.Contains(data, []byte("package strings")) {
		t.Fatalf("bundled strings source = %q/%v", string(data), ok)
	}
	entries, ok := fs.ReadDir("/std/strings")
	if !ok {
		t.Fatal("bundled strings directory missing")
	}
	if len(entries) != 1 || entries[0].Name != "strings.go" || entries[0].IsDir {
		t.Fatalf("bundled strings entries = %#v", entries)
	}
	if _, ok := fs.ReadFile("/std/strings/strings_test.go"); ok {
		t.Fatal("standard library tests were embedded")
	}
}
