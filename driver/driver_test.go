package driver

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"
)

type memorySourceFS struct{ files fstest.MapFS }

func (m memorySourceFS) ReadFile(path string) ([]byte, bool) {
	data, err := fs.ReadFile(m.files, path)
	return data, err == nil
}

func (m memorySourceFS) ReadDir(path string) ([]DirEntry, bool) {
	entries, err := fs.ReadDir(m.files, path)
	if err != nil {
		return nil, false
	}
	var result []DirEntry
	for _, entry := range entries {
		result = append(result, DirEntry{Name: entry.Name(), IsDir: entry.IsDir()})
	}
	return result, true
}

func (m memorySourceFS) PathExists(path string) bool {
	_, err := fs.Stat(m.files, path)
	return err == nil
}

func TestCompileWithDefaultStandardLibrary(t *testing.T) {
	source := memorySourceFS{files: fstest.MapFS{
		"src/go.mod":  &fstest.MapFile{Data: []byte("module hello\n")},
		"src/main.go": &fstest.MapFile{Data: []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello, world!\") }\n")},
	}}
	for _, target := range []struct {
		name  string
		magic []byte
	}{
		{"linux/amd64", []byte("\x7fELF")},
		{"windows/386", []byte("MZ")},
		{"darwin/arm64", []byte{0xcf, 0xfa, 0xed, 0xfe}},
		{"wasi/wasm32", []byte("\x00asm")},
	} {
		t.Run(target.name, func(t *testing.T) {
			result, err := Compile(&Request{Filesystem: source, Input: []string{"src"}, Target: target.name, ArenaSize: 32 * 1024 * 1024})
			if err != nil {
				t.Fatal(err)
			}
			if !result.Ok {
				t.Fatalf("compile failed: %+v", result.Diagnostic)
			}
			if !bytes.HasPrefix(result.Binary, target.magic) {
				t.Fatalf("invalid %s binary (length %d)", target.name, len(result.Binary))
			}
		})
	}
}

func TestBundledSourceFSExists(t *testing.T) {
	fs := BundledSourceFS()
	for _, path := range []string{"std", "std/fmt", "std/strings/strings.go", "/std", "/std/fmt"} {
		if !fs.PathExists(path) {
			t.Errorf("missing bundled path %q", path)
		}
	}
	for _, path := range []string{"std/missing", "std/fmt/missing.go", "std/fmt/fmt_test.go"} {
		if fs.PathExists(path) {
			t.Errorf("unexpected bundled path %q", path)
		}
	}
}
