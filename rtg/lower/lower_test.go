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

func TestPackageSynthesizesAppMainForOrdinaryMain(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func main() {
	print("PASS\n")
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want source main plus synthetic appMain", u.Decls)
	}
	if u.Decls[0].Name != "main" || u.Decls[0].UnitName != "rtg_example_com_app_cmd_app_main" {
		t.Fatalf("source main decl = %#v", u.Decls[0])
	}
	if u.Decls[1].Name != "appMain" || u.Decls[1].Path != "rtg-entrypoint" {
		t.Fatalf("synthetic entrypoint decl = %#v", u.Decls[1])
	}
	want := "func rtg_example_com_app_cmd_app_appMain() int {\n\trtg_example_com_app_cmd_app_main()\n\treturn 0\n}\n"
	if u.Decls[1].Body != want {
		t.Fatalf("synthetic entrypoint body = %q, want %q", u.Decls[1].Body, want)
	}
}

func TestPackageWithGraphRewritesImportedSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
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

func TestPackageWithGraphUsesFileScopedImportAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "a.go",
				Source: []byte(`package main

import first "example.com/app/pkg/answer"

func A() int { return first.Value() }
`),
			},
			{
				Path: "b.go",
				Source: []byte(`package main

import second "example.com/app/pkg/answer"

func appMain() int { return A() + second.Value() }
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
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 || u.References[0].Name != "Value" {
		t.Fatalf("references = %#v, want one Value reference", u.References)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("first alias was not rewritten: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[1].Body, "return rtg_example_com_app_cmd_app_A() + rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("second alias was not rewritten: %q", u.Decls[1].Body)
	}
	if strings.Contains(u.Decls[0].Body, "first.") || strings.Contains(u.Decls[1].Body, "second.") {
		t.Fatalf("alias selector leaked into lowered bodies: %#v", u.Decls)
	}
}

func TestPackageWithGraphDoesNotLeakImportAliasAcrossFiles(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "a.go",
				Source: []byte(`package main

import answer "example.com/app/pkg/answer"

func imported() int { return answer.Value() }
`),
			},
			{
				Path: "b.go",
				Source: []byte(`package main

type localAnswer struct { Value int }

func appMain() int {
	answer := localAnswer{Value: 3}
	return answer.Value + imported()
}
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
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 || u.References[0].Name != "Value" {
		t.Fatalf("references = %#v, want only imported file reference", u.References)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "return answer.Value + rtg_example_com_app_cmd_app_imported()") {
		t.Fatalf("local selector was rewritten by leaked import alias: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_answer_Value") {
		t.Fatalf("local selector contains imported symbol: %q", body)
	}
}

func TestPackageWithGraphPreservesLocalImportNameShadow(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

type localAnswer struct { Value int }

func appMain() int {
	answer := localAnswer{Value: 3}
	return answer.Value
}
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
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 0 {
		t.Fatalf("references = %#v, want none for local shadow", u.References)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "answer := rtg_example_com_app_cmd_app_localAnswer{Value: 3}") {
		t.Fatalf("local shadow declaration was not preserved: %q", body)
	}
	if !strings.Contains(body, "return answer.Value") {
		t.Fatalf("local selector was rewritten as import reference: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_answer_Value") {
		t.Fatalf("local selector contains imported symbol: %q", body)
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

func TestPackageRewritesPackageNameBeforeLocalShadow(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7

func appMain() int {
	before := answer
	answer := 1
	return before + answer
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", u.Decls)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "before := rtg_example_com_app_answer") {
		t.Fatalf("package reference before local shadow was not rewritten: %q", body)
	}
	if !strings.Contains(body, "answer := 1") {
		t.Fatalf("local short declaration was rewritten: %q", body)
	}
	if !strings.Contains(body, "return before + answer") {
		t.Fatalf("local reference after shadow was rewritten: %q", body)
	}
}

func TestPackageRewritesPackageNameAfterInnerBlockShadow(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7

func appMain() int {
	if answer > 0 {
		answer := 1
		_ = answer
	}
	return answer
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", u.Decls)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "if rtg_example_com_app_answer > 0") {
		t.Fatalf("package reference before block shadow was not rewritten: %q", body)
	}
	if !strings.Contains(body, "answer := 1") || !strings.Contains(body, "_ = answer") {
		t.Fatalf("inner block local shadow was rewritten: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_answer") {
		t.Fatalf("package reference after inner block shadow was not rewritten: %q", body)
	}
}

func TestPackageWithGraphRewritesStdSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt"},
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

func TestPackageWithGraphExportsGroupedDeclNames(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int { return dep.Answer + dep.Next }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

const (
	Answer = 41
	Next = Answer + 1
)
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	depUnit, err := PackageWithGraph(depPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph dep failed: %v", err)
	}
	if len(depUnit.Exports) != 2 || depUnit.Exports[0].Name != "Answer" || depUnit.Exports[1].Name != "Next" {
		t.Fatalf("dep exports = %#v", depUnit.Exports)
	}
	if !strings.Contains(depUnit.Decls[0].Body, "rtg_example_com_app_dep_Answer = 41") {
		t.Fatalf("grouped const Answer was not rewritten: %q", depUnit.Decls[0].Body)
	}
	if !strings.Contains(depUnit.Decls[0].Body, "rtg_example_com_app_dep_Next = rtg_example_com_app_dep_Answer + 1") {
		t.Fatalf("grouped const Next was not rewritten: %q", depUnit.Decls[0].Body)
	}
	mainUnit, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph main failed: %v", err)
	}
	if len(mainUnit.References) != 2 || mainUnit.References[0].Name != "Answer" || mainUnit.References[1].Name != "Next" {
		t.Fatalf("main references = %#v", mainUnit.References)
	}
	if !strings.Contains(mainUnit.Decls[0].Body, "return rtg_example_com_app_dep_Answer + rtg_example_com_app_dep_Next") {
		t.Fatalf("main body did not rewrite grouped refs: %q", mainUnit.Decls[0].Body)
	}
}
