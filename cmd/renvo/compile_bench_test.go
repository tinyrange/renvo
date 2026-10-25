package main

import (
	"path/filepath"
	"testing"

	"renvo.dev/internal/driver"
)

// BenchmarkCompilerSelfHost profiles the in-process compilation pipeline. The
// native stage2 performance policy remains the acceptance gate.
func BenchmarkCompilerSelfHost(b *testing.B) {
	root, err := filepath.Abs("../..")
	if err != nil {
		b.Fatal(err)
	}
	args := []string{"-tags", "renvo_bundle", "-t", "linux/amd64", "-arena-size", "201326592", "-s", "-o", "compiler", "./cmd/renvo"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		result := driver.CompileFromFSWithModuleCache(args, root, filepath.Join(root, "std"), "", driver.OSFS{}, checkoutBackend())
		if !result.Ok {
			b.Fatalf("self-host compilation failed: %#v", result.Diagnostic)
		}
		b.ReportMetric(float64(len(result.Binary)), "artifact-B")
	}
}
