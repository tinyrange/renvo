//go:build renvo_bundle && !renvo

package driver

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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
	if _, ok := fs.ReadFile("/std/bytes/bytes.go"); ok {
		t.Fatal("host-only standard library source was embedded")
	}
	if _, ok := fs.ReadFile("/std/bytes/bytes_renvo.go"); !ok {
		t.Fatal("RENVO standard library source was not embedded")
	}
}

func TestBundledStandardLibraryMatchesRepository(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "std"))
	fs := OSFS{}
	var walk func(string)
	walk = func(dir string) {
		entries, ok := fs.ReadDir(dir)
		if !ok {
			t.Fatalf("bundled directory missing: %s", dir)
		}
		for _, entry := range entries {
			path := dir + "/" + entry.Name
			if entry.IsDir {
				walk(path)
				continue
			}
			got, ok := fs.ReadFile(path)
			if !ok {
				t.Fatalf("bundled file missing: %s", path)
			}
			want, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path[len("/std/"):])))
			if err != nil {
				t.Fatalf("read repository source %s: %v", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("bundled source differs: %s", path)
			}
		}
	}
	walk("/std")
}
