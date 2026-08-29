//go:build !renvo

package backendjit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/runimage"
)

type benchmarkArtifactCache map[string][]byte

func (cache benchmarkArtifactCache) Load(key string) ([]byte, bool) {
	source, ok := cache[key]
	return source, ok
}

func (cache benchmarkArtifactCache) Store(key string, source []byte) error {
	cache[key] = append([]byte(nil), source...)
	return nil
}

type jitBenchmarkFixture struct {
	root       string
	definition string
	target     string
	source     []byte
	prepared   Prepared
	unit       []byte
}

func BenchmarkDefinitionPipeline(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	b.ReportAllocs()
	b.ResetTimer()
	generatedBytes := 0
	for b.Loop() {
		parsed := rtg.ParseImports(fixture.source, fixture.definition, filesystemImportLoader{})
		resolved := rtg.ResolveDefinitions(parsed)
		if !resolved.Ok {
			b.Fatal(resolved.Diagnostics)
		}
		generated := rtg.GeneratePreparedBackend(resolved, fixture.target)
		if !generated.Ok {
			b.Fatal(generated.Diagnostics)
		}
		generatedBytes = len(generated.Source)
	}
	b.ReportMetric(float64(len(fixture.source)), "entrypoint-B")
	b.ReportMetric(float64(generatedBytes), "generated-go-B")
}

func BenchmarkPrepareCold(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	config := fixture.prepareConfig(nil)
	b.ReportAllocs()
	b.ResetTimer()
	var prepared Prepared
	for b.Loop() {
		prepared = Prepare(config)
		if !prepared.Ok || prepared.CacheHit {
			b.Fatalf("cold preparation failed: hit=%v diagnostic=%#v", prepared.CacheHit, prepared.Diagnostic)
		}
	}
	b.ReportMetric(float64(len(fixture.source)), "entrypoint-B")
	b.ReportMetric(float64(len(prepared.Artifact.Payload)), "image-B")
	b.ReportMetric(float64(len(prepared.Encoded)), "artifact-B")
}

func BenchmarkPrepareCacheHit(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	cache := make(benchmarkArtifactCache)
	config := fixture.prepareConfig(cache)
	seed := Prepare(config)
	if !seed.Ok || seed.CacheHit {
		b.Fatalf("seed preparation failed: %#v", seed.Diagnostic)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		prepared := Prepare(config)
		if !prepared.Ok || !prepared.CacheHit {
			b.Fatalf("cached preparation failed: hit=%v diagnostic=%#v", prepared.CacheHit, prepared.Diagnostic)
		}
	}
	b.ReportMetric(float64(len(fixture.source)), "entrypoint-B")
	b.ReportMetric(float64(len(seed.Encoded)), "artifact-B")
}

func BenchmarkFrontendUnitForJITBackend(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	args := fixture.compileArgs(true, false)
	b.ReportAllocs()
	b.ResetTimer()
	unitBytes := 0
	for b.Loop() {
		built := driver.BuildFromFSWithBackend(args, fixture.root, filepath.Join(fixture.root, "std"), driver.OSFS{})
		if !built.Ok {
			b.Fatalf("frontend build failed: %#v", built.Diagnostic)
		}
		unitBytes = len(built.Unit)
	}
	b.ReportMetric(float64(unitBytes), "unit-B")
}

func BenchmarkFrontendUnitForJITBackendParallel(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	args := fixture.compileArgs(true, false)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			built := driver.BuildFromFSWithBackend(args, fixture.root, filepath.Join(fixture.root, "std"), driver.OSFS{})
			if !built.Ok {
				b.Errorf("frontend build failed: %#v", built.Diagnostic)
				return
			}
		}
	})
}

func BenchmarkPreparedBackendCompile(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, true)
	runner := ProcessRunner{}
	request := Request{
		Protocol: ProtocolVersion,
		Unit:     fixture.unit,
		Options: driver.BackendCompileOptions{
			Target: fixture.target,
			Strip:  true,
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	outputBytes := 0
	for b.Loop() {
		result := runner.Run(fixture.prepared.Artifact, request)
		if !result.Ok {
			b.Fatalf("prepared backend execution failed: %#v", result.Diagnostic)
		}
		outputBytes = len(result.Binary)
	}
	b.ReportMetric(float64(len(fixture.prepared.Artifact.Payload)), "image-B")
	b.ReportMetric(float64(len(fixture.unit)), "unit-B")
	b.ReportMetric(float64(outputBytes), "output-B")
}

func BenchmarkPreparedBackendCompileParallel(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, true)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		backend := &Backend{prepared: fixture.prepared, runner: ProcessRunner{}}
		for pb.Next() {
			result := backend.CompileUnit(fixture.unit, fixture.target, true, false)
			if !result.Ok {
				b.Errorf("prepared backend execution failed: %#v", result.Diagnostic)
				return
			}
		}
	})
}

func BenchmarkLinkedImageLoadAndRun(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, false)
	compiled := driver.CompileFromFS([]string{
		"-t", fixture.target,
		"-s",
		"-emit-image",
		"-o", "-",
		filepath.Join(fixture.root, "internal", "backendjit", "testdata", "minimal.go"),
	}, fixture.root, filepath.Join(fixture.root, "std"), driver.OSFS{}, backendcompiled.Backend{})
	if !compiled.Ok {
		b.Fatalf("prepare benchmark image: %#v", compiled.Diagnostic)
	}
	image, err := linkedimage.Decode(compiled.Binary)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result := runimage.Run(image, "benchmark.go", nil, nil, os.Stdin, os.Stdout, os.Stderr)
		if result.Err != nil || result.ExitCode != 0 || result.Loader != "jit" {
			b.Fatalf("linked image run = %#v", result)
		}
	}
	b.ReportMetric(float64(len(image.Native)), "native-B")
}

func BenchmarkCustomBackendEndToEnd(b *testing.B) {
	fixture := newJITBenchmarkFixture(b, true)
	backend := &Backend{prepared: fixture.prepared, runner: ProcessRunner{}}
	args := fixture.compileArgs(false, false)
	b.ReportAllocs()
	b.ResetTimer()
	outputBytes := 0
	for b.Loop() {
		compiled := driver.CompileFromFS(args, fixture.root, filepath.Join(fixture.root, "std"), driver.OSFS{}, backend)
		if !compiled.Ok {
			b.Fatalf("end-to-end compile failed: %#v", compiled.Diagnostic)
		}
		outputBytes = len(compiled.Binary)
	}
	b.ReportMetric(float64(outputBytes), "output-B")
}

func newJITBenchmarkFixture(b *testing.B, prepare bool) jitBenchmarkFixture {
	b.Helper()
	target := hostTarget()
	if target == "" {
		b.Skipf("no JIT backend target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		b.Fatal(err)
	}
	definition := filepath.Join(root, "backend", "definitions", jitDefinitionName(target))
	source, err := os.ReadFile(definition)
	if err != nil {
		b.Fatal(err)
	}
	fixture := jitBenchmarkFixture{
		root: root, definition: definition, target: target, source: source,
	}
	if !prepare {
		return fixture
	}
	fixture.prepared = Prepare(fixture.prepareConfig(nil))
	if !fixture.prepared.Ok {
		b.Fatalf("prepare JIT backend: %#v", fixture.prepared.Diagnostic)
	}
	built := driver.BuildFromFSWithBackend(
		fixture.compileArgs(true, false), root, filepath.Join(root, "std"), driver.OSFS{},
	)
	if !built.Ok {
		b.Fatalf("build benchmark unit: %#v", built.Diagnostic)
	}
	fixture.unit = built.Unit
	return fixture
}

func (fixture jitBenchmarkFixture) prepareConfig(cache ArtifactCache) PrepareConfig {
	return PrepareConfig{
		Definition: fixture.source, Filename: fixture.definition,
		ImportLoader: filesystemImportLoader{}, Target: fixture.target,
		BackendRoot: filepath.Join(fixture.root, "backend"), WorkDir: fixture.root,
		StdRoot: filepath.Join(fixture.root, "std"), Cache: cache,
		Bootstrap: backendcompiled.Backend{},
	}
}

func (fixture jitBenchmarkFixture) compileArgs(emitUnit bool, emitImage bool) []string {
	args := []string{
		"-backend", fixture.definition,
		"-t", fixture.target,
		"-s",
	}
	if emitUnit {
		args = append(args, "-emit-unit")
	}
	if emitImage {
		args = append(args, "-emit-image")
	}
	return append(args,
		"-o", "benchmark-output",
		filepath.Join(fixture.root, "internal", "backendjit", "testdata", "minimal.go"),
	)
}

func jitDefinitionName(target string) string {
	switch target {
	case "linux/amd64":
		return "linux_amd64.rtg"
	case "linux/386":
		return "linux_386.rtg"
	case "linux/aarch64":
		return "linux_aarch64.rtg"
	case "linux/arm":
		return "linux_arm.rtg"
	case "windows/amd64":
		return "windows_amd64.rtg"
	case "windows/386":
		return "windows_386.rtg"
	case "windows/arm64":
		return "windows_aarch64.rtg"
	case "darwin/arm64":
		return "darwin_aarch64.rtg"
	default:
		return ""
	}
}
