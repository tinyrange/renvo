//go:build !renvo

package backendcompiled

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestBuiltInTargetRegistryIsCompiled(t *testing.T) {
	for _, target := range []string{
		"linux/amd64", "linux/386", "linux/aarch64", "linux/arm",
		"windows/amd64", "windows/386", "windows/arm64",
		"darwin/arm64", "wasi/wasm32", "vm/vm32",
	} {
		if renvoParseTargetArg(target) == 0 {
			t.Errorf("compiled bundle is missing %s", target)
		}
	}
}

func TestCompiledBundleMatchesEmbeddedSources(t *testing.T) {
	hash := sha256.New()
	for i := 0; i < CompilerSourceCount; i++ {
		name, text, ok := CompilerSource(i)
		if !ok {
			t.Fatalf("could not decompress source %d", i)
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(text))
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != CompilerSourceDigest {
		t.Fatalf("compiled backend source digest = %s, embedded source digest = %s; run go generate ./internal/backendcompiled", CompilerSourceDigest, got)
	}
}
