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
	if !ok || len(entries) != 3 {
		t.Fatalf("strings entries = %v, %v", entries, ok)
	}
	for i, name := range []string{"builder.go", "runes.go", "strings.go"} {
		if entries[i].Name != name || entries[i].IsDir {
			t.Fatalf("strings entry %d = %v, want %s", i, entries[i], name)
		}
	}
}

func TestBundleContentsMatchBuildMode(t *testing.T) {
	for _, path := range []string{"forms", "device", "libc"} {
		if _, ok := bundledStdRawReadDir(path); ok != BundledExtrasEnabled {
			t.Errorf("raw bundle directory %q present = %v, want %v", path, ok, BundledExtrasEnabled)
		}
	}
	if _, ok := bundledStdRawReadDir("x/runtime"); !ok {
		t.Error("core language runtime must be present in every build")
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

func TestCoreRuntimeModuleAvailableWithoutExtras(t *testing.T) {
	const root = "/std/__renvo_runtime_module"
	for _, name := range []string{"go.mod", "x/runtime/runtime.go", "x/runtime/serial/serial.go", "x/runtime/serial/copy_renvo.go"} {
		if data, ok := BundledStdReadFile(root + "/" + name); !ok || len(data) == 0 {
			t.Errorf("missing core runtime file %q", name)
		}
	}
	for _, name := range []string{"forms/forms.go", "device/device.go", "x/runtime/runtime_test.go", "x/runtime/serial/copy_host.go"} {
		if _, ok := BundledStdReadFile(root + "/" + name); ok {
			t.Errorf("core runtime view exposed non-runtime or host-only file %q", name)
		}
	}
	entries, ok := BundledStdReadDir(root)
	if !ok || len(entries) != 2 || entries[0].Name != "go.mod" || entries[1].Name != "x" {
		t.Fatalf("core runtime root entries=%v ok=%v", entries, ok)
	}
	entries, ok = BundledStdReadDir(root + "/x")
	if !ok || len(entries) != 1 || entries[0].Name != "runtime" {
		t.Fatalf("core runtime x entries=%v ok=%v", entries, ok)
	}
}
