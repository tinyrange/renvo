package load

import (
	"testing"

	"j5.nz/rtg/rtg/internal/syntax"
)

func TestParseModule(t *testing.T) {
	mod := ParseModule("/repo/case", []byte(`// comment
module example.com/rtgtests/quick/packages/case000

go 1.25
`))
	if !mod.Ok {
		t.Fatalf("ParseModule failed: err=%d offset=%d", mod.Error, mod.ErrorOffset)
	}
	if mod.Root != "/repo/case" {
		t.Fatalf("root = %q, want /repo/case", mod.Root)
	}
	if mod.Path != "example.com/rtgtests/quick/packages/case000" {
		t.Fatalf("module path = %q", mod.Path)
	}
}

func TestParseModuleQuotedAndCommented(t *testing.T) {
	mod := ParseModule("./repo/../case", []byte(`/* leading */
module "example.com/rtgtests/quoted"
`))
	if !mod.Ok {
		t.Fatalf("ParseModule failed: err=%d offset=%d", mod.Error, mod.ErrorOffset)
	}
	if mod.Root != "case" {
		t.Fatalf("root = %q, want case", mod.Root)
	}
	if mod.Path != "example.com/rtgtests/quoted" {
		t.Fatalf("module path = %q", mod.Path)
	}
}

func TestParseModuleErrors(t *testing.T) {
	missing := ParseModule(".", []byte("go 1.25\n"))
	if missing.Ok || missing.Error != ModuleErrMissing {
		t.Fatalf("missing module = %#v", missing)
	}
	bad := ParseModule(".", []byte("module\n"))
	if bad.Ok || bad.Error != ModuleErrPath {
		t.Fatalf("bad module = %#v", bad)
	}
}

func TestParseModuleDependencyDirectives(t *testing.T) {
	config := &ModuleConfig{}
	mod := ParseModuleConfig("/repo/app", []byte(`module example.com/app

require (
	example.com/lib v1.2.3
	example.com/feature v0.4.0 // indirect
)
replace example.com/lib v1.2.3 => ../lib
	replace example.com/feature => mirror.example/feature v0.4.1
exclude example.com/lib v1.1.0
exclude (
	example.com/lib v1.0.0
)
`), config)
	if !mod.Ok {
		t.Fatalf("ParseModule failed: %#v", mod)
	}
	if len(config.Requires) != 2 || config.Requires[0].Path != "example.com/lib" || config.Requires[0].Version != "v1.2.3" {
		t.Fatalf("requires = %#v", config.Requires)
	}
	if len(config.Replaces) != 2 || !config.Replaces[0].Local || config.Replaces[0].NewPath != "../lib" || config.Replaces[1].Local || config.Replaces[1].NewVersion != "v0.4.1" {
		t.Fatalf("replaces = %#v", config.Replaces)
	}
	if len(config.Excludes) != 2 {
		t.Fatalf("excludes = %#v", config.Excludes)
	}
}

func TestParseModuleRejectsConflictingOrMalformedDirectives(t *testing.T) {
	for _, src := range []string{
		"module example.com/app\nrequire example.com/lib v1\nrequire example.com/lib v2\n",
		"module example.com/app\nreplace example.com/lib => mirror.example/lib\n",
		"module example.com/app\nrequire (\nexample.com/lib v1\n",
		"module example.com/app\nexclude example.com/lib\n",
		"module example.com/app\nrequire example.com/lib latest\n",
		"module example.com/app\nreplace example.com/lib => mirror.example/lib latest\n",
	} {
		mod := ParseModule("/repo/app", []byte(src))
		if mod.Ok || mod.Error != ModuleErrDirective {
			t.Fatalf("malformed module accepted: %q => %#v", src, mod)
		}
	}
}

func TestResolvePackageArg(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	ref := ResolvePackageArg(mod, "/repo/case", "./cmd/app")
	if !ref.Ok {
		t.Fatalf("ResolvePackageArg failed: %#v", ref)
	}
	if ref.Kind != PackageInModule {
		t.Fatalf("kind = %d, want in-module", ref.Kind)
	}
	if ref.Dir != "/repo/case/cmd/app" {
		t.Fatalf("dir = %q", ref.Dir)
	}
	if ref.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("import path = %q", ref.ImportPath)
	}

	root := ResolvePackageArg(mod, "/repo/case/cmd/app", "../..")
	if !root.Ok || root.Dir != "/repo/case" || root.ImportPath != "example.com/case" {
		t.Fatalf("root package = %#v", root)
	}

	outside := ResolvePackageArg(mod, "/repo/case", "../other")
	if outside.Ok || outside.Error != ResolveErrOutsideModule {
		t.Fatalf("outside package = %#v", outside)
	}

	windowsMod := Module{Root: `C:\repo\case`, Path: "example.com/case", Ok: true}
	windows := ResolvePackageArg(windowsMod, `C:\repo\case`, `.\cmd\app`)
	if !windows.Ok || windows.Dir != "C:/repo/case/cmd/app" || windows.ImportPath != "example.com/case/cmd/app" {
		t.Fatalf("windows package = %#v", windows)
	}
}

func TestResolveImport(t *testing.T) {
	mod := Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	local := ResolveImport(mod, "/rtg/std", "example.com/case/pkg/lib")
	if !local.Ok || local.Kind != PackageInModule {
		t.Fatalf("local import = %#v", local)
	}
	if local.Dir != "/repo/case/pkg/lib" {
		t.Fatalf("local dir = %q", local.Dir)
	}
	std := ResolveImport(mod, "/rtg/std", "runtime")
	if !std.Ok || std.Kind != PackageStandard || std.Dir != "/rtg/std/runtime" {
		t.Fatalf("std import = %#v", std)
	}
	external := ResolveImport(mod, "/rtg/std", "other.example/pkg")
	if external.Ok || external.Error != ResolveErrUnsupported {
		t.Fatalf("external import = %#v", external)
	}
	relative := ResolveImport(mod, "/rtg/std", "./pkg")
	if relative.Ok || relative.Error != ResolveErrImport {
		t.Fatalf("relative import = %#v", relative)
	}
}

func TestResolveImportUsesLongestDependencyModule(t *testing.T) {
	mod := Module{Root: "/repo/app", Path: "example.com/app", Ok: true}
	dependencies := []ModuleDependency{
		{Path: "example.com/lib", Root: "/cache/lib"},
		{Path: "example.com/lib/plugin", Root: "/cache/plugin"},
		{Path: "example.com/app/nested", Root: "/cache/nested"},
	}
	plugin := ResolveImportWithDependencies(mod, "/std", "example.com/lib/plugin/api", dependencies)
	if !plugin.Ok || plugin.Kind != PackageDependency || plugin.Dir != "/cache/plugin/api" {
		t.Fatalf("plugin import = %#v", plugin)
	}
	nested := ResolveImportWithDependencies(mod, "/std", "example.com/app/nested/pkg", dependencies)
	if !nested.Ok || nested.Kind != PackageDependency || nested.Dir != "/cache/nested/pkg" {
		t.Fatalf("nested import = %#v", nested)
	}
}

func TestFileImports(t *testing.T) {
	mod := Module{
		Root: "/repo/case",
		Path: "example.com/rtgtests/quick/packages/case000",
		Ok:   true,
	}
	src := []byte(`package main

import (
	"runtime"
	lib "example.com/rtgtests/quick/packages/case000/pkg/lib"
)
`)
	file := syntax.ParseFile(src)
	if !file.Ok {
		t.Fatalf("ParseFile failed: err=%d tok=%d", file.Error, file.ErrorTok)
	}
	imports := FileImports(mod, "/rtg/std", file)
	if len(imports) != 2 {
		t.Fatalf("import count = %d, want 2", len(imports))
	}
	if !imports[0].Ok || imports[0].Kind != PackageStandard || imports[0].Dir != "/rtg/std/runtime" {
		t.Fatalf("runtime import = %#v", imports[0])
	}
	if !imports[1].Ok || imports[1].Kind != PackageInModule || imports[1].Dir != "/repo/case/pkg/lib" {
		t.Fatalf("local import = %#v", imports[1])
	}
}

func TestCleanAndRelPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "."},
		{in: ".", want: "."},
		{in: "/repo//case/./cmd/../pkg", want: "/repo/case/pkg"},
		{in: "a/b/../../c", want: "c"},
		{in: "../a", want: "../a"},
		{in: `C:\repo\\case\.\cmd\..\pkg`, want: "C:/repo/case/pkg"},
	}
	for _, tc := range cases {
		got := CleanPath(tc.in)
		if got != tc.want {
			t.Fatalf("CleanPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	rel, ok := RelPath("/repo/case", "/repo/case/pkg/lib")
	if !ok || rel != "pkg/lib" {
		t.Fatalf("RelPath = %q %v", rel, ok)
	}
	_, ok = RelPath("/repo/case", "/repo/case2/pkg")
	if ok {
		t.Fatal("RelPath accepted sibling prefix")
	}
	if rel, ok := RelPath(".", "cmd/app"); !ok || rel != "cmd/app" {
		t.Fatalf("RelPath relative root = %q %v", rel, ok)
	}
	if _, ok := RelPath(".", "../outside"); ok {
		t.Fatal("RelPath accepted path above relative root")
	}
}
