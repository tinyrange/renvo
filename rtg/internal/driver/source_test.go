package driver

import (
	"testing"

	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtgunit"
)

func TestCollectSourcesDiscoversPackageGraph(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/cmd/app/main_test.go", Src: []byte("package broken\n")},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

import "runtime"

func Value() int {
	runtime.KeepAlive(0)
	return 42
}
`)},
		{Path: "/std/runtime/runtime.go", Src: []byte(`package runtime

func KeepAlive(v int) {}
`)},
	}}
	result := CollectSources("/repo/case/cmd/app", "/std", ".", fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	want := []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/main.go",
		"/repo/case/pkg/lib/lib.go",
		"/std/runtime/runtime.go",
	}
	assertSourcePaths(t, result.Files, want)
	if result.Module.Path != "example.com/case" || result.Root.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("module/root = %#v %#v", result.Module, result.Root)
	}
}

func TestCollectSourcesAppliesRTGBuildTags(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/host.go", Src: []byte(`//go:build !rtg

package main

import "fmt"

func appMain() int { fmt.Println("host"); return 1 }
`)},
		{Path: "/repo/case/cmd/app/linux.go", Src: []byte(`//go:build rtg && linux && amd64

package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/cmd/app/arm.go", Src: []byte(`//go:build rtg && linux && arm64

package main

func appMain() int { return 2 }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

func Value() int { return 42 }
`)},
	}}
	result := CollectSourcesForTarget("/repo/case/cmd/app", "/std", ".", "linux/amd64", fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	want := []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/linux.go",
		"/repo/case/pkg/lib/lib.go",
	}
	assertSourcePaths(t, result.Files, want)
}

func TestCollectSourcesAppliesDarwinArm64BuildTags(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/darwin.go", Src: []byte(`//go:build rtg && darwin && unix && arm64

package main

func appMain() int { return 0 }
`)},
		{Path: "/repo/case/cmd/app/linux.go", Src: []byte(`//go:build linux

package main

func appMain() int { return 1 }
`)},
	}}
	result := CollectSourcesForTarget("/repo/case/cmd/app", "/std", ".", "darwin/arm64", fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	want := []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/darwin.go",
	}
	assertSourcePaths(t, result.Files, want)
}

func TestCollectSourcesReportsErrors(t *testing.T) {
	missingModule := CollectSources("/repo/case", "/std", "./cmd/app", memorySourceFS{})
	if missingModule.Ok || missingModule.Error != SourceErrMissingModule {
		t.Fatalf("missing module result = %#v", missingModule)
	}

	missingPackage := CollectSources("/repo/case", "/std", "./cmd/app", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
	}})
	if missingPackage.Ok || missingPackage.Error != SourceErrReadDir || missingPackage.ErrorPath != "/repo/case/cmd/app" {
		t.Fatalf("missing package result = %#v", missingPackage)
	}

	badSyntax := CollectSources("/repo/case", "/std", "./cmd/app", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc f()\n")},
	}})
	if badSyntax.Ok || badSyntax.Error != SourceErrParse || badSyntax.ErrorPath != "/repo/case/cmd/app/main.go" {
		t.Fatalf("bad syntax result = %#v", badSyntax)
	}

	badImport := CollectSources("/repo/case", "/std", "./cmd/app", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.net/lib\"\n")},
	}})
	if badImport.Ok || badImport.Error != SourceErrImport || badImport.ErrorPath != "example.net/lib" {
		t.Fatalf("bad import result = %#v", badImport)
	}
}

func TestBuildFromFS(t *testing.T) {
	result := BuildFromFS([]string{"-t", "linux/amd64", "-s", "-o", "app", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: driverTestFiles()})
	if !result.Ok {
		t.Fatalf("BuildFromFS failed: err=%d path=%q arg=%q pkg=%d file=%d tok=%d", result.Error, result.ErrorPath, result.ErrorArg, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	if result.Options.Output != "app" || result.Options.Target != "linux/amd64" || !result.Options.Strip {
		t.Fatalf("options = %#v", result.Options)
	}
	if len(result.Sources.Files) != 3 {
		t.Fatalf("source count = %d, want 3", len(result.Sources.Files))
	}
	decoded, err := rtgunit.Unmarshal(result.Unit)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" || len(decoded.Funcs) != 2 {
		t.Fatalf("decoded unit = package %q funcs %d", decoded.Package, len(decoded.Funcs))
	}
}

func TestBuildFromFSReportsSourceError(t *testing.T) {
	result := BuildFromFS([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
	}})
	if result.Ok || result.Error != BuildErrSource {
		t.Fatalf("BuildFromFS missing package result = %#v", result)
	}
	if result.Sources.Error != SourceErrReadDir || result.ErrorPath != "/repo/case/cmd/app" {
		t.Fatalf("source location = err %d path %q", result.Sources.Error, result.ErrorPath)
	}
}

type memorySourceFS struct {
	files []load.SourceFile
}

func (fs memorySourceFS) ReadFile(path string) ([]byte, bool) {
	path = load.CleanPath(path)
	for i := 0; i < len(fs.files); i++ {
		if load.CleanPath(fs.files[i].Path) == path {
			return fs.files[i].Src, true
		}
	}
	return nil, false
}

func (fs memorySourceFS) ReadDir(path string) ([]DirEntry, bool) {
	path = load.CleanPath(path)
	var entries []DirEntry
	for i := 0; i < len(fs.files); i++ {
		filePath := load.CleanPath(fs.files[i].Path)
		if load.DirPath(filePath) == path {
			entries = append(entries, DirEntry{Name: load.BasePath(filePath)})
		}
	}
	if len(entries) == 0 {
		return nil, false
	}
	return entries, true
}

func assertSourcePaths(t *testing.T, files []load.SourceFile, want []string) {
	t.Helper()
	if len(files) != len(want) {
		t.Fatalf("source count = %d, want %d: %#v", len(files), len(want), files)
	}
	for i := 0; i < len(want); i++ {
		if files[i].Path != want[i] {
			t.Fatalf("source %d = %q, want %q in %#v", i, files[i].Path, want[i], files)
		}
	}
}
