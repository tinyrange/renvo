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
	font, ok := fs.ReadFile("/std/graphics/gofont/Go-Mono.ttf")
	if !ok || len(font) < 4 || !bytes.Equal(font[:4], []byte{0, 1, 0, 0}) {
		t.Fatal("standard library embed asset was not embedded")
	}
}

func TestBundledFormsModuleCache(t *testing.T) {
	data, ok := bundledStdReadFile("/modules/renvo.dev@v0.0.0/go.mod")
	if !ok || string(data) != "module renvo.dev\n" {
		t.Fatalf("bundled module file = %q/%v", string(data), ok)
	}
	data, ok = bundledStdReadFile("/modules/renvo.dev@v0.0.0/forms/forms.go")
	if !ok || len(data) == 0 {
		t.Fatal("bundled Forms source missing")
	}
	entries, ok := bundledStdReadDir("/modules/renvo.dev@v0.0.0")
	if !ok || len(entries) != 3 || entries[0].Name != "go.mod" || entries[1].Name != "forms" || entries[2].Name != "std" {
		t.Fatalf("bundled module root = %#v/%v", entries, ok)
	}
}

func TestBundledFormsModuleCompilesOffline(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\nrequire renvo.dev v0.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	source := []byte("package main\nimport \"renvo.dev/forms\"\nfunc main() { _ = forms.NewButton() }\n")
	if err := os.WriteFile(filepath.Join(root, "main.go"), source, 0644); err != nil {
		t.Fatal(err)
	}
	result := BuildFromFSWithModuleCache([]string{"-t", "browser/wasm32", "-o", "app", "."}, root, "/std", "/modules", OSFS{})
	if !result.Ok {
		t.Fatalf("offline Forms build failed: %#v", result.Diagnostic)
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
