package renvo

import (
	"bytes"
	"testing"
)

func TestStandardLibraryAvailableInEveryBuild(t *testing.T) {
	for _, path := range []string{"std/fmt/fmt_renvo.go", "/std/strings/strings.go", "std/bytes/bytes_renvo.go", "std/unsafe/unsafe.go", "std/graphics/gofont/Go-Mono.ttf"} {
		data, ok := BundledStdReadFile(path)
		if !ok || len(data) == 0 {
			t.Errorf("missing bundled source or asset %q", path)
		}
	}
	for _, path := range []string{"std/strings/strings_test.go", "std/bytes/bytes.go"} {
		if _, ok := BundledStdReadFile(path); ok {
			t.Errorf("exposed host-only or test source %q", path)
		}
	}
	entries, ok := BundledStdReadDir("/std/strings")
	if !ok || len(entries) != 1 || entries[0].Name != "strings.go" || entries[0].IsDir {
		t.Fatalf("strings entries = %v, %v", entries, ok)
	}
}

func TestBundleContentsMatchBuildMode(t *testing.T) {
	for _, path := range []string{"forms", "device", "x", "libc"} {
		if _, ok := bundledStdRawReadDir(path); ok != BundledExtrasEnabled {
			t.Errorf("raw bundle directory %q present = %v, want %v", path, ok, BundledExtrasEnabled)
		}
	}
	for _, path := range []string{"/modules/renvo.dev@v0.0.0", "/modules/renvo.dev@v0.0.0/forms", "/libc/include"} {
		if _, ok := BundledStdReadDir(path); ok != BundledExtrasEnabled {
			t.Errorf("bundle directory %q present = %v, want %v", path, ok, BundledExtrasEnabled)
		}
	}
	for _, path := range []string{"/modules/renvo.dev@v0.0.0/go.mod", "/modules/renvo.dev@v0.0.0/forms/forms.go", "/libc/include/stdio.h"} {
		data, ok := BundledStdReadFile(path)
		if ok != BundledExtrasEnabled || ok && len(bytes.TrimSpace(data)) == 0 {
			t.Errorf("bundle file %q present = %v, length = %d", path, ok, len(data))
		}
	}
}
