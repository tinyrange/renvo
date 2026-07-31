//go:build !renvo

package backendcompiled

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"renvo.dev/internal/targetinfo"
	"renvo.dev/internal/unit"
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

func TestBuiltInBackendRejectsMismatchedUnitBinding(t *testing.T) {
	base := make([]byte, 14)
	copy(base, unit.Magic)
	base[4] = byte(unit.Version)
	base[8] = byte(unit.TagUnit)
	descriptor, ok := targetinfo.Lookup("linux/amd64")
	if !ok {
		t.Fatal("linux/amd64 descriptor missing")
	}
	bound, ok := unit.BindTarget(base, unit.TargetBinding{
		Target:            descriptor.Name,
		Definition:        string(descriptor.Definition[:]),
		DescriptorVersion: descriptor.DescriptorVersion,
	})
	if !ok || !unitBindingMatches(bound, descriptor.Name) {
		t.Fatal("matching unit binding was rejected")
	}
	bound[len(bound)-1] ^= 1
	if unitBindingMatches(bound, descriptor.Name) {
		t.Fatal("descriptor-version mismatch was accepted")
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
