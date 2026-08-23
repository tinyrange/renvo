//go:build !renvo

package backendbuiltin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/rtg"
	"renvo.dev/internal/targetinfo"
)

func TestEmbeddedDefinitionCatalogIsFresh(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "backend", "definitions", "*.rtg"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("list checked-in definitions: %v", err)
	}
	for _, path := range paths {
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := definitionSource(filepath.Base(path))
		if !ok || !bytes.Equal(got, want) {
			t.Fatalf("embedded definition %s is stale; run go generate ./internal/backendbuiltin", path)
		}
	}
}

func TestEmbeddedDefinitionsResolveExactTargetSemantics(t *testing.T) {
	for _, target := range []string{
		"linux/amd64", "windows/amd64", "darwin/arm64", "wasi/wasm32", "vm/vm32",
	} {
		resolved := Resolve(target)
		if !resolved.Ok {
			t.Fatalf("resolve %s: %#v", target, resolved.Diagnostics)
		}
		generated := rtg.GeneratePreparedBackend(resolved, target)
		if !generated.Ok {
			t.Fatalf("generate %s: %#v", target, generated.Diagnostics)
		}
		want, known := targetinfo.Lookup(target)
		if !known || generated.Descriptor.Name != want.Name ||
			generated.Descriptor.Definition != want.Definition ||
			generated.Descriptor.Version != want.DescriptorVersion {
			t.Fatalf("embedded %s descriptor = %#v, registry = %#v", target, generated.Descriptor, want)
		}
	}
}
