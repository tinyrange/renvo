//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
)

func TestExcludedFamily(t *testing.T) {
	excluded := excludedFamily(rtg.TargetDescriptor{ISA: "amd64"})
	for _, name := range []string{"compiler_rtg_generated_impl.go", "compiler_amd64_target_impl.go"} {
		if !excluded[name] {
			t.Errorf("%s was not excluded", name)
		}
	}
	if excluded["compiler_aarch64_impl.go"] {
		t.Fatal("unrelated backend was excluded")
	}
	if excluded["compiler_amd64_impl.go"] {
		t.Fatal("amd64 semantic lowering was excluded with its generated encoder")
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

func TestConcurrentPublishNeverExposesPartialEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache", "entry.rtgb")
	values := [][]byte{
		bytes.Repeat([]byte("a"), 4096),
		bytes.Repeat([]byte("b"), 8192),
		bytes.Repeat([]byte("c"), 16384),
	}
	var wait sync.WaitGroup
	for i := 0; i < len(values); i++ {
		wait.Add(1)
		go func(value []byte) {
			defer wait.Done()
			if err := publish(path, value); err != nil {
				t.Errorf("publish: %v", err)
			}
		}(values[i])
	}
	wait.Wait()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if bytes.Equal(got, value) {
			return
		}
	}
	t.Fatalf("published entry has partial size %d", len(got))
}

type memoryRunner struct {
	request Request
}

func (runner *memoryRunner) Run(_ rtgb.Artifact, request Request) driver.BackendResult {
	runner.request = request
	return driver.BackendResult{Binary: []byte("in-memory-output"), Ok: true}
}

func TestInMemoryRunnerUsesVersionedProtocol(t *testing.T) {
	var definition [32]byte
	definition[0] = 42
	descriptor := rtg.TargetDescriptor{
		Name: "example/amd64", Definition: definition, Version: rtg.DescriptorVersion,
	}
	base := make([]byte, 14)
	copy(base, unit.Magic)
	base[4] = byte(unit.Version)
	base[8] = byte(unit.TagUnit)
	bound, ok := unit.BindTarget(base, unit.TargetBinding{
		Target: descriptor.Name, Definition: string(definition[:]),
		DescriptorVersion: descriptor.Version,
	})
	if !ok {
		t.Fatal("BindTarget failed")
	}
	runner := new(memoryRunner)
	backend := &Backend{
		runner: runner,
		prepared: Prepared{Ok: true, Artifact: rtgb.Artifact{
			Descriptor: descriptor, Protocol: ProtocolVersion,
			Unit: unit.Version, Optimization: OptimizationVersion,
		}},
	}
	result := backend.CompileUnit(bound, descriptor.Name, true, false)
	if !result.Ok || string(result.Binary) != "in-memory-output" {
		t.Fatalf("in-memory compile = %#v", result)
	}
	if runner.request.Protocol != ProtocolVersion ||
		!bytes.Equal(runner.request.Unit, bound) || !runner.request.Options.Strip {
		t.Fatalf("runner request = %#v", runner.request)
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
	builtInDefinition := filepath.Join(root, "backend", "definitions", "amd64.rtg")
	definitionSource, err := os.ReadFile(builtInDefinition)
	if err != nil {
		t.Fatal(err)
	}
	definitionSource = bytes.Replace(definitionSource,
		[]byte("target linux/amd64 {"), []byte("target example/amd64 {"), 1)
	definition := filepath.Join(t.TempDir(), "example-amd64.rtg")
	if err := os.WriteFile(definition, definitionSource, 0o600); err != nil {
		t.Fatal(err)
	}
	stdRoot := filepath.Join(root, "std")
	source := filepath.Join(root, "backend", "tests", "arithmetic_return_expression.go")
	cache := t.TempDir()
	args := []string{
		"-backend", definition,
		"-t", "example/amd64",
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
	for _, mutate := range []func(*rtgb.Artifact){
		func(artifact *rtgb.Artifact) { artifact.Protocol++ },
		func(artifact *rtgb.Artifact) { artifact.Unit++ },
		func(artifact *rtgb.Artifact) { artifact.Optimization++ },
		func(artifact *rtgb.Artifact) { artifact.Generator++ },
		func(artifact *rtgb.Artifact) { artifact.Kernel++ },
		func(artifact *rtgb.Artifact) { artifact.Host += "-wrong" },
	} {
		artifact := first.prepared.Artifact
		mutate(&artifact)
		encoded, ok := rtgb.Encode(artifact)
		if !ok {
			t.Fatal("could not encode incompatible artifact")
		}
		if loaded := Load(encoded); loaded.Ok {
			t.Fatalf("accepted incompatible artifact %#v", artifact)
		}
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
