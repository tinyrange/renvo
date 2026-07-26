//go:build !renvo

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/driver"
)

type guiBenchmarkProject struct {
	root       string
	stdRoot    string
	output     string
	moduleFile string
}

func newGUIBenchmarkProject(b *testing.B) guiBenchmarkProject {
	b.Helper()
	root := b.TempDir()
	if created, message := ensureHelloWorldProject(root); !created {
		b.Fatalf("create Forms benchmark project: %s", message)
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		b.Fatal("locate repository root")
	}
	repository := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	module := []byte("module example.com/hello\n\nrequire renvo.dev v0.0.0\nreplace renvo.dev => " + filepath.ToSlash(repository) + "\n")
	if err := os.WriteFile(filepath.Join(root, projectModuleFile), module, 0644); err != nil {
		b.Fatal(err)
	}
	return guiBenchmarkProject{
		root:       root,
		stdRoot:    filepath.Join(repository, "std"),
		output:     filepath.Join(root, "hello"),
		moduleFile: filepath.Join(root, projectUserFormFile),
	}
}

func (p guiBenchmarkProject) build(cached bool) driver.BuildResult {
	args := []string{"-t", "browser/wasm32", "-s", "-o", p.output, "."}
	if !cached {
		return driver.BuildFromFS(args, p.root, p.stdRoot, driver.OSFS{})
	}
	session := driver.BeginFSBuildSession(args, p.root, p.stdRoot, "", driver.OSFS{}, true)
	for !session.Step() {
	}
	return session.Result()
}

// BenchmarkGUIFrontendCold records CPU time and Go heap allocations for a
// complete generated Forms application without incremental package caches.
func BenchmarkGUIFrontendCold(b *testing.B) {
	project := newGUIBenchmarkProject(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := project.build(false)
		if !result.Ok {
			b.Fatalf("cold GUI build: %#v", result.Diagnostic)
		}
	}
}

// BenchmarkGUIFrontendUnchanged measures the common Build-button path after a
// successful output has already been written.
func BenchmarkGUIFrontendUnchanged(b *testing.B) {
	project := newGUIBenchmarkProject(b)
	first := project.build(true)
	if !first.Ok || first.CacheHit {
		b.Fatalf("initial GUI build: %#v", first.Diagnostic)
	}
	if err := os.WriteFile(project.output, []byte("benchmark output"), 0755); err != nil {
		b.Fatal(err)
	}
	driver.RememberBuildOutput(first)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := project.build(true)
		if !result.Ok || !result.CacheHit {
			b.Fatalf("unchanged GUI build did not hit cache: %#v", result.Diagnostic)
		}
	}
}

// BenchmarkGUIFrontendEdited measures an editor rebuild after the root package
// changes while Forms and graphics dependencies remain cached.
func BenchmarkGUIFrontendEdited(b *testing.B) {
	project := newGUIBenchmarkProject(b)
	first := project.build(true)
	if !first.Ok {
		b.Fatalf("initial GUI build: %#v", first.Diagnostic)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		source := append(helloUserFormSource(), '\n')
		source = append(source, '/', '/', byte('0'+i%10), '\n')
		if err := os.WriteFile(project.moduleFile, source, 0644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		result := project.build(true)
		if !result.Ok || result.CacheHit {
			b.Fatalf("edited GUI build: %#v", result.Diagnostic)
		}
	}
}
