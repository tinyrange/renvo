package lower

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/load"
)

func TestPackageBuildsDeclarationUnit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path:     "/tmp/work/pkg/b.go",
				UnitPath: "pkg/b.go",
				Source: []byte(`package pkg

func B() int { return A() }
`),
			},
			{
				Path:     "/tmp/work/pkg/a.go",
				UnitPath: "pkg/a.go",
				Source: []byte(`package pkg

const answer = 7
func A() int { return answer }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if u.ImportPath != "example.com/app/pkg" || u.Package != "pkg" {
		t.Fatalf("unit identity = %#v", u)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[0].Name != "answer" || u.Decls[1].Name != "A" || u.Decls[2].Name != "B" {
		t.Fatalf("decl order = %#v", u.Decls)
	}
	if u.Decls[0].Path != "pkg/a.go" || u.Decls[2].Path != "pkg/b.go" {
		t.Fatalf("decl paths = %#v", u.Decls)
	}
	if len(u.Exports) != 2 || u.Exports[0].Name != "A" || u.Exports[1].Name != "B" {
		t.Fatalf("exports = %#v", u.Exports)
	}
	if u.Decls[0].UnitName != "rtg_example_com_app_pkg_answer" {
		t.Fatalf("const unit name = %q", u.Decls[0].UnitName)
	}
	if !strings.Contains(u.Decls[0].Body, "const rtg_example_com_app_pkg_answer = 7") {
		t.Fatalf("const body was not rewritten: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_A() int { return rtg_example_com_app_pkg_answer }") {
		t.Fatalf("A body was not rewritten: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "func rtg_example_com_app_pkg_B() int { return rtg_example_com_app_pkg_A() }") {
		t.Fatalf("B body was not rewritten: %q", u.Decls[2].Body)
	}
	if strings.Contains(u.Decls[0].Body, "package ") {
		t.Fatalf("decl retained package clause: %q", u.Decls[0].Body)
	}
}

func TestSymbolNameIsStableIdentifier(t *testing.T) {
	got := SymbolName("example.com/team/app-pkg", "Value")
	want := "rtg_example_com_team_app_pkg_Value"
	if got != want {
		t.Fatalf("SymbolName = %q, want %q", got, want)
	}
}

func TestPackageWithGraphRewritesImportedSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath:  "example.com/app/cmd/app",
		Name:        "main",
		Imports:     []string{"example.com/app/pkg/answer"},
		ImportNames: map[string]string{"example.com/app/pkg/answer": "answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

func appMain() int { return answer.Value() }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
func hidden() int { return 9 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 {
		t.Fatalf("references = %#v, want one", u.References)
	}
	ref := u.References[0]
	if ref.ImportPath != "example.com/app/pkg/answer" || ref.Name != "Value" || ref.UnitName != "rtg_example_com_app_pkg_answer_Value" {
		t.Fatalf("reference = %#v", ref)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("imported selector was not rewritten: %q", u.Decls[0].Body)
	}
}

func TestPackagePreservesLocalShadowNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7
const total = 9

func Use(answer int) int {
	return answer
}

func appMain() int {
	answer := 1
	var total int
	total = answer
	return Use(total)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want 4", u.Decls)
	}
	use := u.Decls[2].Body
	if !strings.Contains(use, "func rtg_example_com_app_Use(answer int) int") {
		t.Fatalf("Use signature rewrote parameter name: %q", use)
	}
	if !strings.Contains(use, "return answer") {
		t.Fatalf("Use body rewrote parameter reference: %q", use)
	}
	appMain := u.Decls[3].Body
	if !strings.Contains(appMain, "answer := 1") {
		t.Fatalf("appMain rewrote local short declaration: %q", appMain)
	}
	if !strings.Contains(appMain, "var total int") || !strings.Contains(appMain, "total = answer") {
		t.Fatalf("appMain rewrote local var declaration: %q", appMain)
	}
	if !strings.Contains(appMain, "return rtg_example_com_app_Use(total)") {
		t.Fatalf("appMain did not preserve local argument while rewriting callee: %q", appMain)
	}
	if strings.Contains(appMain, "rtg_example_com_app_answer := 1") {
		t.Fatalf("appMain rewrote local shadow as package symbol: %q", appMain)
	}
	if strings.Contains(appMain, "var rtg_example_com_app_total int") {
		t.Fatalf("appMain rewrote var shadow as package symbol: %q", appMain)
	}
}

func TestPackageWithGraphRewritesStdSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath:  "example.com/app",
		Name:        "main",
		Imports:     []string{"fmt"},
		ImportNames: map[string]string{"fmt": "fmt"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "fmt"

func appMain() int { return fmt.PrintInt(7) }
`),
			},
		},
	}
	stdPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func PrintInt(v int) int { return v }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, stdPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 {
		t.Fatalf("references = %#v, want one", u.References)
	}
	ref := u.References[0]
	if ref.ImportPath != "fmt" || ref.Name != "PrintInt" || ref.UnitName != "rtg_fmt_PrintInt" {
		t.Fatalf("reference = %#v", ref)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_fmt_PrintInt(7)") {
		t.Fatalf("std selector was not rewritten: %q", u.Decls[0].Body)
	}
}
