package driver

import (
	"testing"

	"renvo.dev/backend/unit"
	"renvo.dev/internal/load"
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

func TestCollectSourcesAppliesRenvoBuildTags(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/host.go", Src: []byte(`//go:build !renvo

package main

import "fmt"

func appMain() int { fmt.Println("host"); return 1 }
`)},
		{Path: "/repo/case/cmd/app/linux.go", Src: []byte(`//go:build renvo && linux && amd64

package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/cmd/app/arm.go", Src: []byte(`//go:build renvo && linux && arm64

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
		{Path: "/repo/case/cmd/app/darwin.go", Src: []byte(`//go:build renvo && darwin && unix && arm64

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

func TestCollectSourcesAppliesCustomBuildTags(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/plain.go", Src: []byte(`//go:build !renvo_bundle

package main

func appMain() int { return 0 }
`)},
		{Path: "/repo/case/cmd/app/bundle.go", Src: []byte(`//go:build renvo_bundle

package main

func appMain() int { return 1 }
`)},
	}}
	result := CollectSourcesForTargetTags("/repo/case/cmd/app", "/std", ".", "linux/amd64", []string{"renvo_bundle"}, fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	want := []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/bundle.go",
	}
	assertSourcePaths(t, result.Files, want)
}

func TestSourceFilenameSelectionAcrossTargets(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{"main_linux.go", "linux/amd64", true},
		{"main_linux.go", "windows/amd64", false},
		{"main_linux_386.go", "linux/386", true},
		{"main_arm64.go", "linux/aarch64", true},
		{"main_arm64.go", "darwin/arm64", true},
		{"main_arm64.go", "windows/arm64", true},
		{"main_arm64.go", "linux/arm", false},
		{"main_linux_arm64.go", "linux/aarch64", true},
		{"main_linux_arm64.go", "darwin/arm64", false},
		{"main_windows_386.go", "windows/386", true},
		{"main_windows_386.go", "windows/amd64", false},
		{"main_windows_arm64.go", "windows/arm64", true},
		{"main_windows_arm64.go", "windows/amd64", false},
		{"main_wasip1_wasm.go", "wasi/wasm32", true},
		{"main_wasi_wasm32.go", "wasi/wasm32", true},
		{"main_browser_wasm32.go", "browser/wasm32", true},
		{"main_wasi_wasm32.go", "browser/wasm32", true},
		{"main_browser_wasm32.go", "wasi/wasm32", false},
		{"main_plan9.go", "linux/amd64", false},
		{"main_feature.go", "linux/amd64", true},
	}
	for _, test := range tests {
		if got := sourceFilenameEnabled(test.name, test.target); got != test.want {
			t.Errorf("sourceFilenameEnabled(%q, %q) = %v, want %v", test.name, test.target, got, test.want)
		}
	}
}

func TestBrowserTargetProvidesBrowserAndWASITags(t *testing.T) {
	for _, expression := range []string{"renvo && browser && wasm32", "renvo && wasi && wasm32", "renvo && wasip1 && wasm"} {
		enabled, valid := evalBuildExprWithTags([]byte(expression), "browser/wasm32", nil)
		if !valid || !enabled {
			t.Fatalf("browser expression %q = %v, %v", expression, enabled, valid)
		}
	}
	enabled, valid := evalBuildExprWithTags([]byte("!browser"), "browser/wasm32", nil)
	if !valid || enabled {
		t.Fatalf("!browser expression = %v, %v", enabled, valid)
	}
}

func TestCollectSourcesCombinesFilenameAndBuildConstraints(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/platform_linux_amd64.go", Src: []byte("//go:build renvo\n\npackage main\nfunc appMain() int { return 0 }\n")},
		{Path: "/repo/case/cmd/app/platform_linux_arm64.go", Src: []byte("//go:build renvo\n\npackage main\nfunc appMain() int { return 1 }\n")},
		{Path: "/repo/case/cmd/app/platform_windows_amd64.go", Src: []byte("package main\nfunc appMain() int { return 2 }\n")},
	}}
	result := CollectSourcesForTarget("/repo/case/cmd/app", "/std", ".", "linux/amd64", fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	assertSourcePaths(t, result.Files, []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/platform_linux_amd64.go",
	})
}

func TestCollectSourcesAppliesLegacyBuildConstraints(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/linux.go", Src: []byte("// +build linux,amd64 darwin\n\npackage main\nfunc appMain() int { return 0 }\n")},
		{Path: "/repo/case/cmd/app/windows.go", Src: []byte("// +build windows\n\npackage main\nfunc appMain() int { return 1 }\n")},
	}}
	result := CollectSourcesForTarget("/repo/case/cmd/app", "/std", ".", "linux/amd64", fs)
	if !result.Ok {
		t.Fatalf("CollectSources failed: err=%d path=%q", result.Error, result.ErrorPath)
	}
	assertSourcePaths(t, result.Files, []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/linux.go",
	})
}

func TestLegacyBuildConstraintExpressions(t *testing.T) {
	tests := []struct {
		expr    string
		target  string
		enabled bool
		valid   bool
	}{
		{"linux,amd64 darwin", "linux/amd64", true, true},
		{"linux,amd64 darwin", "linux/arm", false, true},
		{"!windows", "linux/amd64", true, true},
		{"linux,!amd64", "linux/amd64", false, true},
		{"linux,,amd64", "linux/amd64", false, false},
		{"!", "linux/amd64", false, false},
		{"", "linux/amd64", false, false},
	}
	for _, test := range tests {
		enabled, valid := evalPlusBuildLine([]byte(test.expr), test.target, nil)
		if enabled != test.enabled || valid != test.valid {
			t.Errorf("evalPlusBuildLine(%q, %q) = (%v, %v), want (%v, %v)", test.expr, test.target, enabled, valid, test.enabled, test.valid)
		}
	}
}

func TestCollectSourcesRejectsMalformedBuildConstraints(t *testing.T) {
	for _, src := range []string{
		"//go:build linux &&\n\npackage main\n",
		"//go:build linux\n//go:build amd64\n\npackage main\n",
		"// +build linux,,amd64\n\npackage main\n",
	} {
		result := CollectSourcesForTarget("/repo/case/cmd/app", "/std", ".", "linux/amd64", memorySourceFS{files: []load.SourceFile{
			{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
			{Path: "/repo/case/cmd/app/main.go", Src: []byte(src)},
		}})
		if result.Ok || result.Error != SourceErrBuildConstraint || result.ErrorPath != "/repo/case/cmd/app/main.go" {
			t.Fatalf("malformed constraint result = %#v", result)
		}
	}
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
	if !badSyntax.Ok || len(badSyntax.Files) != 2 {
		t.Fatalf("source collection should leave full syntax validation to the parser: %#v", badSyntax)
	}

	badImport := CollectSources("/repo/case", "/std", "./cmd/app", memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.net/lib\"\n")},
	}})
	if badImport.Ok || badImport.Error != SourceErrImport || badImport.ErrorPath != "example.net/lib" {
		t.Fatalf("bad import result = %#v", badImport)
	}
}

func TestCollectSourcesResolvesLocalReplacementGraph(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\nrequire example.com/lib v1.0.0\nreplace example.com/lib => ../lib\nreplace example.com/value => ../value\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib\"\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/lib/go.mod", Src: []byte("module example.com/lib\nrequire example.com/value v1.0.0\nreplace example.com/value => ../value\n")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\nimport \"example.com/value\"\nfunc Value() int { return value.Number() }\n")},
		{Path: "/repo/value/go.mod", Src: []byte("module example.com/value\n")},
		{Path: "/repo/value/value.go", Src: []byte("package value\nfunc Number() int { return 42 }\n")},
	}}
	result := CollectSources("/repo/app", "/std", ".", fs)
	if !result.Ok {
		t.Fatalf("local replacement graph failed: %#v", result)
	}
	assertSourcePaths(t, result.Files, []string{
		"/repo/app/go.mod", "/repo/app/main.go",
		"/repo/lib/go.mod", "/repo/lib/lib.go",
		"/repo/value/go.mod", "/repo/value/value.go",
	})
	built := BuildFromFS([]string{"-o", "app", "."}, "/repo/app", "/std", fs)
	if !built.Ok {
		t.Fatalf("multi-module build failed: %#v diagnostic=%#v", built, built.Diagnostic)
	}
}

func TestCollectSourcesResolvesImportsWithinStandardLibrary(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"strings\"\nfunc appMain() int { return strings.Count() }\n")},
		{Path: "/std/strings/strings.go", Src: []byte("package strings\nimport \"unsafe\"\nfunc Count() int { return unsafe.Size() }\n")},
		{Path: "/std/unsafe/unsafe.go", Src: []byte("package unsafe\nfunc Size() int { return 1 }\n")},
	}}
	result := CollectSources("/repo/app", "/std", ".", fs)
	if !result.Ok {
		t.Fatalf("standard dependency graph failed: %#v", result)
	}
	assertSourcePaths(t, result.Files, []string{
		"/repo/app/go.mod", "/repo/app/main.go",
		"/std/strings/strings.go", "/std/unsafe/unsafe.go",
	})
}

func TestCollectSourcesRestartsAfterTransitiveVersionUpgrade(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\nrequire (\nexample.com/a v1.0.0\nexample.com/b v1.0.0\n)\nreplace example.com/a => ../a\nreplace example.com/b => ../b\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport (\n\"example.com/a\"\n_ \"example.com/b\"\n)\nfunc appMain() int { return a.Value() }\n")},
		{Path: "/repo/a/go.mod", Src: []byte("module example.com/a\nrequire example.com/value v1.9.0\n")},
		{Path: "/repo/a/a.go", Src: []byte("package a\nimport \"example.com/value\"\nfunc Value() int { return value.Number() }\n")},
		{Path: "/repo/b/go.mod", Src: []byte("module example.com/b\nrequire example.com/value v1.10.0\n")},
		{Path: "/repo/b/b.go", Src: []byte("package b\nimport _ \"example.com/value\"\n")},
		{Path: "/cache/example.com/value@v1.9.0/go.mod", Src: []byte("module example.com/value\n")},
		{Path: "/cache/example.com/value@v1.9.0/value.go", Src: []byte("package value\nfunc Number() int { return 9 }\n")},
		{Path: "/cache/example.com/value@v1.10.0/go.mod", Src: []byte("module example.com/value\n")},
		{Path: "/cache/example.com/value@v1.10.0/value.go", Src: []byte("package value\nfunc Number() int { return 10 }\n")},
	}}
	result := CollectSourcesForTargetTagsWithModuleCache("/repo/app", "/std", ".", "linux/amd64", nil, "/cache", fs)
	if !result.Ok {
		t.Fatalf("transitive version selection failed: %#v", result)
	}
	foundSelected := false
	for i := 0; i < len(result.Files); i++ {
		if result.Files[i].Path == "/cache/example.com/value@v1.9.0/value.go" {
			t.Fatalf("retained superseded module source: %#v", result.Files)
		}
		if result.Files[i].Path == "/cache/example.com/value@v1.10.0/value.go" {
			foundSelected = true
		}
	}
	if !foundSelected {
		t.Fatalf("selected module source missing: %#v", result.Files)
	}
	built := BuildFromFSWithModuleCache([]string{"-o", "app", "."}, "/repo/app", "/std", "/cache", fs)
	if !built.Ok {
		t.Fatalf("selected module build failed: %#v diagnostic=%#v", built, built.Diagnostic)
	}
}

func TestCompareModuleVersion(t *testing.T) {
	for _, test := range []struct {
		left  string
		right string
		want  int
	}{
		{left: "v1.10.0", right: "v1.9.0", want: 1},
		{left: "v2.0.0", right: "v10.0.0", want: -1},
		{left: "v1.0.0", right: "v1.0.0-rc.1", want: 1},
		{left: "v1.0.0-rc.10", right: "v1.0.0-rc.2", want: 1},
		{left: "v1.0.0+meta", right: "v1.0.0", want: 0},
	} {
		got := compareModuleVersion(test.left, test.right)
		if got < 0 {
			got = -1
		} else if got > 0 {
			got = 1
		}
		if got != test.want {
			t.Fatalf("compareModuleVersion(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestCollectSourcesResolvesVendorBeforeCache(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\nrequire example.com/lib v1.0.0\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib/pkg\"\nfunc appMain() int { return pkg.Value() }\n")},
		{Path: "/repo/app/vendor/example.com/lib/pkg/lib.go", Src: []byte("package pkg\nfunc Value() int { return 42 }\n")},
		{Path: "/cache/example.com/lib@v1.0.0/go.mod", Src: []byte("module example.com/lib\n")},
		{Path: "/cache/example.com/lib@v1.0.0/pkg/lib.go", Src: []byte("package pkg\nfunc Value() int { return 7 }\n")},
	}}
	result := CollectSourcesForTargetTagsWithModuleCache("/repo/app", "/std", ".", "linux/amd64", nil, "/cache", fs)
	if !result.Ok {
		t.Fatalf("vendor collection failed: %#v", result)
	}
	if len(result.Files) != 4 || result.Files[2].Path != "/repo/app/vendor/example.com/lib/go.mod" || result.Files[3].Path != "/repo/app/vendor/example.com/lib/pkg/lib.go" {
		t.Fatalf("vendor did not take precedence: %#v", result.Files)
	}
}

func TestCollectSourcesResolvesLocalReplacementBeforeVendor(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\nrequire example.com/lib v1.0.0\nreplace example.com/lib => ../lib\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib\"\nfunc appMain() int { return lib.Value() }\n")},
		{Path: "/repo/app/vendor/example.com/lib/lib.go", Src: []byte("package lib\nfunc Value() int { return 7 }\n")},
		{Path: "/repo/lib/go.mod", Src: []byte("module example.com/lib\n")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\nfunc Value() int { return 42 }\n")},
	}}
	result := CollectSources("/repo/app", "/std", ".", fs)
	if !result.Ok {
		t.Fatalf("local replacement collection failed: %#v", result)
	}
	if len(result.Files) != 4 || result.Files[3].Path != "/repo/lib/lib.go" {
		t.Fatalf("local replacement did not take precedence: %#v", result.Files)
	}
}

func TestCollectSourcesResolvesReplacementFromReadOnlyCache(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\nrequire example.com/Upper v1.0.0\nreplace example.com/Upper => mirror.example/Upper v1.4.0\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/Upper/pkg\"\nfunc appMain() int { return pkg.Value() }\n")},
		{Path: "/cache/mirror.example/!upper@v1.4.0/go.mod", Src: []byte("module mirror.example/Upper\n")},
		{Path: "/cache/mirror.example/!upper@v1.4.0/pkg/lib.go", Src: []byte("package pkg\nfunc Value() int { return 42 }\n")},
	}}
	result := CollectSourcesForTargetTagsWithModuleCache("/repo/app", "/std", ".", "linux/amd64", nil, "/cache", fs)
	if !result.Ok {
		t.Fatalf("cache collection failed: %#v", result)
	}
	if len(result.Files) != 4 || result.Files[3].Path != "/cache/mirror.example/!upper@v1.4.0/pkg/lib.go" {
		t.Fatalf("replacement cache mapping = %#v", result.Files)
	}
}

func TestCollectSourcesReportsOfflineModuleFailuresAtImport(t *testing.T) {
	tests := []struct {
		name      string
		goMod     string
		cache     string
		wantError int
		wantPath  string
	}{
		{name: "missing", goMod: "module example.com/app\nrequire example.com/lib v1.0.0\n", wantError: SourceErrDependencyMissing, wantPath: "example.com/lib@v1.0.0"},
		{name: "missing cache entry", goMod: "module example.com/app\nrequire example.com/lib v1.0.0\n", cache: "/cache", wantError: SourceErrDependencyMissing, wantPath: "example.com/lib@v1.0.0"},
		{name: "excluded", goMod: "module example.com/app\nrequire example.com/lib v1.0.0\nexclude example.com/lib v1.0.0\n", wantError: SourceErrDependencyExcluded, wantPath: "example.com/lib@v1.0.0"},
	}
	for _, test := range tests {
		fs := memorySourceFS{files: []load.SourceFile{
			{Path: "/repo/app/go.mod", Src: []byte(test.goMod)},
			{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/lib\"\nfunc appMain() int { return 0 }\n")},
		}}
		result := BuildFromFSWithModuleCache([]string{"-o", "app", "."}, "/repo/app", "/std", test.cache, fs)
		if result.Ok || result.Sources.Error != test.wantError || result.Sources.ErrorPath != test.wantPath {
			t.Fatalf("%s result = %#v", test.name, result)
		}
		if result.Diagnostic.Path != "/repo/app/main.go" || result.Diagnostic.Line != 2 || result.Diagnostic.Column < 1 {
			t.Fatalf("%s diagnostic location = %#v", test.name, result.Diagnostic)
		}
	}
}

func TestCollectSourcesRejectsNestedModuleAsMainPackage(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/app/main.go", Src: []byte("package main\nimport \"example.com/app/nested\"\nfunc appMain() int { return 0 }\n")},
		{Path: "/repo/app/nested/go.mod", Src: []byte("module example.com/app/nested\n")},
		{Path: "/repo/app/nested/nested.go", Src: []byte("package nested\n")},
	}}
	result := CollectSources("/repo/app", "/std", ".", fs)
	if result.Ok || result.Error != SourceErrDependencyAmbiguous || result.ErrorPath != "example.com/app/nested" || result.ErrorSourcePath != "/repo/app/main.go" {
		t.Fatalf("nested module boundary result = %#v", result)
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
	decoded, err := unit.Unmarshal(result.Unit)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" || len(decoded.Funcs) != 2 {
		t.Fatalf("decoded unit = package %q funcs %d", decoded.Package, len(decoded.Funcs))
	}
}

func TestBuildFromFSExplicitFileListUsesOnlyNamedFiles(t *testing.T) {
	fs := memorySourceFS{files: []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("//go:build windows\n\npackage main\nimport \"example.com/case/lib\"\nfunc main() { print(message() + lib.Value()) }\n")},
		{Path: "/repo/case/cmd/app/helper_linux_arm64.go", Src: []byte("//go:build windows\n\npackage main\nfunc message() string { return \"PASS\" }\n")},
		{Path: "/repo/case/cmd/app/sibling.go", Src: []byte("package main\nfunc message() string { return \"FAIL\" }\n")},
		{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Value() string { return \"\\n\" }\n")},
	}}
	result := BuildFromFS([]string{"-o", "app", "./cmd/app/main.go", "./cmd/app/helper_linux_arm64.go"}, "/repo/case", "/std", fs)
	if !result.Ok {
		t.Fatalf("explicit BuildFromFS failed: diagnostic=%#v result=%#v", result.Diagnostic, result)
	}
	assertSourcePaths(t, result.Sources.Files, []string{
		"/repo/case/go.mod",
		"/repo/case/cmd/app/main.go",
		"/repo/case/lib/lib.go",
		"/repo/case/cmd/app/helper_linux_arm64.go",
	})
}

func TestBuildFromFSExplicitFileListDiagnostics(t *testing.T) {
	base := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/a/main.go", Src: []byte("package main\nfunc main() {}\n")},
		{Path: "/repo/case/a/other.go", Src: []byte("package other\n")},
		{Path: "/repo/case/b/helper.go", Src: []byte("package main\n")},
	}
	differentDirs := BuildFromFS([]string{"-o", "app", "./a/main.go", "./b/helper.go"}, "/repo/case", "/std", memorySourceFS{files: base})
	if differentDirs.Ok || differentDirs.Diagnostic.Code != "RENVO-LOAD-021" || differentDirs.Diagnostic.Path != "/repo/case/b/helper.go" {
		t.Fatalf("different-directory diagnostic = %#v", differentDirs)
	}
	differentPackages := BuildFromFS([]string{"-o", "app", "./a/main.go", "./a/other.go"}, "/repo/case", "/std", memorySourceFS{files: base})
	if differentPackages.Ok || differentPackages.Diagnostic.Code != "RENVO-LOAD-012" || differentPackages.Diagnostic.Path != "/repo/case/a/other.go" {
		t.Fatalf("different-package diagnostic = %#v", differentPackages)
	}
	testOnly := append(base, load.SourceFile{Path: "/repo/case/a/main_test.go", Src: []byte("package main\n")})
	empty := BuildFromFS([]string{"-o", "app", "./a/main_test.go"}, "/repo/case", "/std", memorySourceFS{files: testOnly})
	if empty.Ok || empty.Diagnostic.Code != "RENVO-LOAD-022" {
		t.Fatalf("test-only diagnostic = %#v", empty)
	}
}

func TestBuildFromFSExplicitFileListRetainsSemanticDiagnostics(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/main.go", Src: []byte("package main\nfunc main() { unused := 1 }\n")},
		{Path: "/repo/case/no_main.go", Src: []byte("package main\nfunc helper() {}\n")},
	}
	unused := BuildFromFS([]string{"-o", "app", "main.go"}, "/repo/case", "/std", memorySourceFS{files: files})
	if unused.Ok || unused.Diagnostic.Code != "RENVO-CHECK-020" || unused.Diagnostic.Path != "/repo/case/main.go" || unused.Diagnostic.Line != 2 {
		t.Fatalf("explicit unused-local diagnostic = %#v", unused)
	}
	missingMain := BuildFromFS([]string{"-o", "app", "no_main.go"}, "/repo/case", "/std", memorySourceFS{files: files})
	if missingMain.Ok || missingMain.Diagnostic.Code != "RENVO-CHECK-021" {
		t.Fatalf("explicit missing-main diagnostic = %#v", missingMain)
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
