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
	if _, ok := fs.ReadFile("/std/unsafe/unsafe.go"); !ok {
		t.Fatal("unsafe standard library source was not embedded")
	}
	if _, ok := fs.ReadFile("/std/sync/sync_renvo.go"); !ok {
		t.Fatal("sync standard library source was not embedded")
	}
	stdio, ok := fs.ReadFile("/libc/include/stdio.h")
	if !ok || !bytes.Contains(stdio, []byte("int printf")) {
		t.Fatal("C standard library headers were not embedded")
	}
	if implementation, ok := fs.ReadFile("/libc/src/stdio.c"); !ok || !bytes.Contains(implementation, []byte("int vfprintf")) {
		t.Fatal("C standard library implementation was not embedded")
	}
	font, ok := fs.ReadFile("/std/graphics/gofont/Go-Mono.ttf")
	if !ok || len(font) < 4 || !bytes.Equal(font[:4], []byte{0, 1, 0, 0}) {
		t.Fatal("standard library embed asset was not embedded")
	}
}

func TestBundledOptionalModuleCache(t *testing.T) {
	data, ok := bundledStdReadFile("/modules/renvo.dev@v0.0.0/go.mod")
	if !ok || string(data) != "module renvo.dev\n" {
		t.Fatalf("bundled module file = %q/%v", string(data), ok)
	}
	data, ok = bundledStdReadFile("/modules/renvo.dev@v0.0.0/forms/forms.go")
	if !ok || len(data) == 0 {
		t.Fatal("bundled Forms source missing")
	}
	data, ok = bundledStdReadFile("/modules/renvo.dev@v0.0.0/device/i2c/i2c.go")
	if !ok || len(data) == 0 {
		t.Fatal("bundled device source missing")
	}
	data, ok = bundledStdReadFile("/modules/renvo.dev@v0.0.0/x/runtime/runtime.go")
	if !ok || len(data) == 0 {
		t.Fatal("bundled runtime source missing")
	}
	entries, ok := bundledStdReadDir("/modules/renvo.dev@v0.0.0")
	if !ok || len(entries) != 5 || entries[0].Name != "go.mod" || entries[1].Name != "device" || entries[2].Name != "forms" || entries[3].Name != "std" || entries[4].Name != "x" {
		t.Fatalf("bundled module root = %#v/%v", entries, ok)
	}
}

func TestBundledDeviceModuleCompilesOffline(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\nrequire renvo.dev v0.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	source := []byte("package main\nimport \"renvo.dev/device/i2c\"\nfunc main() { _ = i2c.ErrBusy }\n")
	if err := os.WriteFile(filepath.Join(root, "main.go"), source, 0644); err != nil {
		t.Fatal(err)
	}
	result := BuildFromFSWithModuleCache([]string{"-t", "linux/amd64", "-o", "app", "."}, root, "/std", "/modules", OSFS{})
	if !result.Ok {
		t.Fatalf("offline device build failed: %#v", result.Diagnostic)
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

func TestBundledRuntimeModuleCompilesConcurrencyOffline(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\nrequire renvo.dev v0.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	source := []byte(`package main
import (
	"sync"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)
func main() {
	runtime.EnableGoroutines(serial.New())
	var mutex sync.Mutex
	mutex.Lock()
	mutex.Unlock()
	values := make(chan int, 1)
	values <- 7
	print(<-values)
}
`)
	if err := os.WriteFile(filepath.Join(root, "main.go"), source, 0644); err != nil {
		t.Fatal(err)
	}
	result := BuildFromFSWithModuleCache([]string{"-t", "linux/amd64", "-o", "app", "."}, root, "/std", "/modules", OSFS{})
	if !result.Ok {
		t.Fatalf("offline runtime build failed: %#v", result.Diagnostic)
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
