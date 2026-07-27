//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
)

func TestExcludedFamily(t *testing.T) {
	excluded := excludedFamily(rtg.TargetDescriptor{ISA: "amd64"})
	for _, name := range []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go", "compiler_windows_amd64_impl.go"} {
		if !excluded[name] {
			t.Errorf("%s was not excluded", name)
		}
	}
	if excluded["compiler_aarch64_impl.go"] {
		t.Fatal("unrelated backend was excluded")
	}
}

func TestPublishIsReusable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "entry.rtgb")
	if err := publish(path, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := publish(path, []byte("second")); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != "second" {
		t.Fatalf("cache source = %q", source)
	}
}

func TestCompiledInBootstrapPreparesAndCachesBackend(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("in-process prepared backend requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backend", "definitions", "amd64.rtg")
	stdRoot := filepath.Join(root, "std")
	source := filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go")
	cache := t.TempDir()
	args := []string{
		"-backend", definition,
		"-t", "linux/amd64",
		"-s",
		"-o", "app",
		source,
	}

	first := New(definition, filepath.Join(root, "backend"), stdRoot, cache, backendcompiled.Backend{})
	firstResult := driver.CompileFromFS(args, root, stdRoot, driver.OSFS{}, first)
	if !firstResult.Ok {
		t.Fatalf("cold custom backend compile failed: %#v", firstResult.Diagnostic)
	}
	if first.prepared.CacheHit {
		t.Fatal("cold custom backend unexpectedly hit cache")
	}
	if !bytes.HasPrefix(firstResult.Binary, []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("cold custom backend output prefix = % x", firstResult.Binary[:minInt(4, len(firstResult.Binary))])
	}

	second := New(definition, filepath.Join(root, "backend"), stdRoot, cache, backendcompiled.Backend{})
	secondResult := driver.CompileFromFS(args, root, stdRoot, driver.OSFS{}, second)
	if !secondResult.Ok {
		t.Fatalf("warm custom backend compile failed: %#v", secondResult.Diagnostic)
	}
	if !second.prepared.CacheHit {
		t.Fatal("warm custom backend did not hit cache")
	}
	if !bytes.Equal(firstResult.Binary, secondResult.Binary) {
		t.Fatal("cold and warm custom backend outputs differ")
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
