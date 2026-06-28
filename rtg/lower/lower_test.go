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

func TestPackageLowersTopLevelNameLists(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "names.go",
				Source: []byte(`package pkg

const A, B = 1, 2
var C, D int
func Use() int { return A + B + C + D }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Exports) != 5 {
		t.Fatalf("exports = %#v, want A, B, C, D, Use", u.Exports)
	}
	if !strings.Contains(u.Decls[0].Body, "const rtg_example_com_app_pkg_A, rtg_example_com_app_pkg_B = 1, 2") {
		t.Fatalf("const list was not rewritten: %q", u.Decls[0].Body)
	}
	if u.Decls[0].Name != "A, B" || u.Decls[0].UnitName != "" {
		t.Fatalf("const list metadata = %#v", u.Decls[0])
	}
	if !strings.Contains(u.Decls[1].Body, "var rtg_example_com_app_pkg_C, rtg_example_com_app_pkg_D int") {
		t.Fatalf("var list was not rewritten: %q", u.Decls[1].Body)
	}
	if u.Decls[1].Name != "C, D" || u.Decls[1].UnitName != "" {
		t.Fatalf("var list metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[2].Body, "return rtg_example_com_app_pkg_A + rtg_example_com_app_pkg_B + rtg_example_com_app_pkg_C + rtg_example_com_app_pkg_D") {
		t.Fatalf("function body did not rewrite all names: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersMethodsToFunctions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Point struct { x int; y int }

func (p Point) Sum() int {
	return p.x + p.y
}

func Use() int {
	p := Point{x: 3, y: 4}
	return p.Sum()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[1].Name != "Point_Sum" || u.Decls[1].UnitName != "rtg_example_com_app_pkg_Point_Sum" {
		t.Fatalf("method metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_Point_Sum(p rtg_example_com_app_pkg_Point) int") {
		t.Fatalf("method declaration was not lowered: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "p := rtg_example_com_app_pkg_Point{x: 3, y: 4}") {
		t.Fatalf("method receiver local type was not rewritten: %q", u.Decls[2].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "return rtg_example_com_app_pkg_Point_Sum(p)") {
		t.Fatalf("method call was not lowered: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersPointerReceiverMethodCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Counter struct { n int }

func (c *Counter) Add(v int) {
	c.n = c.n + v
}

func Use() int {
	var c Counter
	c.Add(5)
	return c.n
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[1].Name != "Counter_Add" || u.Decls[1].UnitName != "rtg_example_com_app_pkg_Counter_Add" {
		t.Fatalf("method metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_Counter_Add(c *rtg_example_com_app_pkg_Counter, v int)") {
		t.Fatalf("pointer receiver declaration was not lowered: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "rtg_example_com_app_pkg_Counter_Add(&c, 5)") {
		t.Fatalf("pointer receiver call was not lowered: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersValueReceiverMethodCallsOnPointers(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Items []int

func (items Items) Add(v int) Items {
	return append(items, v)
}

func Fill(items *Items) {
	*items = items.Add(5)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if !strings.Contains(u.Decls[2].Body, "*items = rtg_example_com_app_pkg_Items_Add(*items, 5)") {
		t.Fatalf("value receiver pointer call was not lowered with dereference: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersSamePackageTypesInFunctionSignatures(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "types.go",
				Source: []byte(`package pkg

type Item struct { Name string }
type Items []Item

func Copy(out Items, values Items) Items {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "out Items") || strings.Contains(body, "values Items") || strings.Contains(body, ") Items") {
		t.Fatalf("function signature retained source type names: %q", body)
	}
	if !strings.Contains(body, "out rtg_example_com_app_pkg_Items, values rtg_example_com_app_pkg_Items") {
		t.Fatalf("parameter types were not lowered: %q", body)
	}
	if !strings.Contains(body, ") rtg_example_com_app_pkg_Items") {
		t.Fatalf("result type was not lowered: %q", body)
	}
}

func TestPackageDoesNotRewriteCompositeFieldKeys(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "keys.go",
				Source: []byte(`package pkg

type Unit struct { Package string }
type Input struct { Name string }

func Package(input Input) Unit {
	return Unit{Package: input.Name}
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_pkg_Unit{Package: input.Name}") {
		t.Fatalf("composite field key was rewritten: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Package:") {
		t.Fatalf("composite field key contains unit name: %q", body)
	}
}

func TestPackageDoesNotRewriteStructFieldNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "fields.go",
				Source: []byte(`package pkg

type Artifact struct {
	Source []byte
}

func Source() []byte {
	return nil
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "Source []byte") {
		t.Fatalf("struct field name was rewritten: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Source []byte") {
		t.Fatalf("struct field contains unit name: %q", body)
	}
}

func TestPackageLowersStructFieldTypeNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "fields.go",
				Source: []byte(`package pkg

type Symbol struct {
	Name string
}

type Unit struct {
	Exports []Symbol
	Primary Symbol
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "[]Symbol") || strings.Contains(body, "Primary Symbol") {
		t.Fatalf("struct field type retained source name: %q", body)
	}
	if !strings.Contains(body, "Exports []rtg_example_com_app_pkg_Symbol") {
		t.Fatalf("slice field type was not lowered: %q", body)
	}
	if !strings.Contains(body, "Primary rtg_example_com_app_pkg_Symbol") {
		t.Fatalf("plain field type was not lowered: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Exports") || strings.Contains(body, "rtg_example_com_app_pkg_Primary") {
		t.Fatalf("struct field name was rewritten: %q", body)
	}
}

func TestPackageLowersImportedMethodCalls(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	var b dep.Buffer
	p := &b
	p.WriteString("PASS")
	return p.Len()
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct { buf []byte }

func (b *Buffer) WriteString(s string) int {
	return len(s)
}

func (b *Buffer) Len() int {
	return len(b.buf)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "var b rtg_example_com_app_dep_Buffer") {
		t.Fatalf("imported receiver type was not rewritten: %q", body)
	}
	if !strings.Contains(body, "p := &b") {
		t.Fatalf("address-of receiver local was not preserved: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_dep_Buffer_WriteString(p, \"PASS\")") {
		t.Fatalf("imported pointer method call was not lowered: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Buffer_Len(p)") {
		t.Fatalf("imported method call was not lowered: %q", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Buffer", "Buffer_Len", "Buffer_WriteString"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
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
	want := "func rtg_example_com_app_cmd_app_appMain(args []string, env []string) int {\n\trtg_example_com_app_cmd_app_main()\n\treturn 0\n}\n"
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

func TestPackageNormalizesNestedReturnCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int { return join(first(), second()) }
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
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("return call did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesDeepNestedReturnCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func inner(v int) int { return v + 1 }
func outer(v int) int { return v + 1 }
func appMain() int { return outer(inner(first())) }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	first := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()")
	inner := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_inner(rtg_example_com_app_appMain_tmp_0)")
	ret := strings.Index(body, "return rtg_example_com_app_outer(rtg_example_com_app_appMain_tmp_1)")
	if first < 0 || inner < 0 || ret < 0 {
		t.Fatalf("deep nested calls were not fully normalized: %q", body)
	}
	if !(first < inner && inner < ret) {
		t.Fatalf("deep nested call temps emitted in wrong order: %q", body)
	}
}

func TestPackageNormalizesNestedImportedReturnCallArguments(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int { return dep.Join(dep.First(), dep.Second()) }
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

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 3 {
		t.Fatalf("references = %#v, want First, Join, Second", u.References)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_dep_First()") {
		t.Fatalf("first imported call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_dep_Second()") {
		t.Fatalf("second imported call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("return call did not use imported lifted temps: %q", body)
	}
}

func TestPackageNormalizesNestedAssignmentCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := join(first(), second())
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first assignment call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second assignment call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("assignment did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesNestedVarInitializerCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	var total = join(first(), second())
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first var initializer call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second var initializer call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "var total = rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("var initializer did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesIndexBoundCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func index() int { return 1 }
func appMain() int {
	values := []int{10, 20}
	return values[index()]
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_index()") {
		t.Fatalf("index bound call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return values[rtg_example_com_app_appMain_tmp_0]") {
		t.Fatalf("index expression did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesSliceBoundCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func start() int { return 1 }
func end() int { return 5 }
func appMain() string {
	text := "xPASSx"
	return text[start():end()]
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_start()") {
		t.Fatalf("slice start call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_end()") {
		t.Fatalf("slice end call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return text[rtg_example_com_app_appMain_tmp_0:rtg_example_com_app_appMain_tmp_1]") {
		t.Fatalf("slice expression did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesSliceBoundCallsInsideCallArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func start() int { return 1 }
func end() int { return 5 }
func consume(text string) {}
func appMain() int {
	text := "xPASSx"
	consume(text[start():end()])
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	start := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_start()")
	end := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_end()")
	arg := strings.Index(body, "rtg_example_com_app_appMain_tmp_2 := text[rtg_example_com_app_appMain_tmp_0:rtg_example_com_app_appMain_tmp_1]")
	call := strings.Index(body, "rtg_example_com_app_consume(rtg_example_com_app_appMain_tmp_2)")
	if start < 0 || end < 0 || arg < 0 || call < 0 {
		t.Fatalf("slice bound calls inside call argument were not fully normalized: %q", body)
	}
	if !(start < end && end < arg && arg < call) {
		t.Fatalf("slice bound call argument temps emitted in wrong order: %q", body)
	}
}

func TestPackageNormalizesNestedIfConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	if join(first(), second()) == 12 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "if rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12 {") {
		t.Fatalf("condition did not use lifted temps: %q", body)
	}
}

func TestPackagePreservesShortCircuitConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Decl struct { Body string }
type Unit struct { Decls []Decl }
func declAt(u Unit, index int) Decl { return u.Decls[index] }
func bodyAt(u Unit, index int) string {
	decl := declAt(u, index)
	return decl.Body
}
func check(value string) bool { return value != "" }
func done(decl Decl) bool { return decl.Body == "" }
func appMain(args []string, env []string) int {
	var u Unit
	currentDecl := -1
	if currentDecl >= 0 && check(bodyAt(u, currentDecl)) && !done(declAt(u, currentDecl)) {
		return 1
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[len(u.Decls)-1].Body
	firstGuard := strings.Index(body, "if currentDecl >= 0 {")
	bodyTemp := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_bodyAt(u, currentDecl)")
	bodyGuard := strings.Index(body, "if rtg_example_com_app_check(rtg_example_com_app_appMain_tmp_0) {")
	declTemp := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_declAt(u, currentDecl)")
	declGuard := strings.Index(body, "if !rtg_example_com_app_done(rtg_example_com_app_appMain_tmp_1) {")
	if firstGuard < 0 || bodyTemp < 0 || bodyGuard < 0 || declTemp < 0 || declGuard < 0 {
		t.Fatalf("short-circuit condition was not lowered into nested guarded temps: %q", body)
	}
	if !(firstGuard < bodyTemp && bodyTemp < bodyGuard && bodyGuard < declTemp && declTemp < declGuard) {
		t.Fatalf("short-circuit condition temps were emitted before their guards: %q", body)
	}
}

func TestPackageNormalizesNestedIfShortStatementCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	if total := join(first(), second()); total == 12 {
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first short-statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second short-statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)\n\t\tif total == 12 {") {
		t.Fatalf("if short statement did not use lifted temps: %q", body)
	}
}

func TestPackageDoesNotHoistIfShortConditionCallsBeforeInit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func check(v int) bool { return v == 2 }
func appMain() int {
	if total := first(); check(next(total)) {
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	init := strings.Index(body, "total := rtg_example_com_app_first()")
	temp := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_next(total)")
	condition := strings.Index(body, "if rtg_example_com_app_check(rtg_example_com_app_appMain_tmp_0) {")
	if init < 0 || temp < 0 || condition < 0 {
		t.Fatalf("if short condition was not lowered into scoped temporaries: %q", body)
	}
	if !(init < temp && temp < condition) {
		t.Fatalf("if short condition temp was not after init and before condition: %q", body)
	}
}

func TestPackageNormalizesNestedSwitchTagCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	switch join(first(), second()) {
	case 12:
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first switch tag call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second switch tag call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "switch rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) {") {
		t.Fatalf("switch tag did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesNestedSwitchShortStatementCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	switch total := join(first(), second()); total {
	case 12:
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first switch short statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second switch short statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)\n\t\tswitch total {") {
		t.Fatalf("switch short statement did not use lifted temps: %q", body)
	}
}

func TestPackageDoesNotHoistSwitchShortTagCallsBeforeInit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	switch total := first(); next(total) {
	case 2:
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	init := strings.Index(body, "total := rtg_example_com_app_first()")
	tag := strings.Index(body, "switch rtg_example_com_app_next(total) {")
	if init < 0 || tag < 0 {
		t.Fatalf("switch short tag was not lowered into scoped form: %q", body)
	}
	if !(init < tag) {
		t.Fatalf("switch short tag was emitted before init: %q", body)
	}
}

func TestPackageNormalizesNestedCallStatementArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func consume(a int, b int) {}
func appMain() int {
	consume(first(), second())
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_consume(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("call statement did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesForPostClauseCallArgumentsWithoutContinue(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	for i := 0; i < 3; i = next(first()) {
		_ = i
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "{\n\t\ti := 0\n\t\tfor {") {
		t.Fatalf("for post clause was not lowered into scoped loop: %q", body)
	}
	if !strings.Contains(body, "if !(i < 3) {\n\t\t\t\tbreak\n\t\t\t}") {
		t.Fatalf("for post condition guard was not emitted: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("for post nested call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_0)") {
		t.Fatalf("for post clause did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesClassicForConditionAndPostCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a + b }
func appMain() int {
	total := 0
	for i := 0; join(first(), second()) < 4; i = join(first(), second()) {
		total = total + i
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	condFirst := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()")
	condSecond := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()")
	guard := strings.Index(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) < 4)")
	postFirst := strings.Index(body, "rtg_example_com_app_appMain_tmp_2 := rtg_example_com_app_first()")
	postSecond := strings.Index(body, "rtg_example_com_app_appMain_tmp_3 := rtg_example_com_app_second()")
	postAssign := strings.Index(body, "i = rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_2, rtg_example_com_app_appMain_tmp_3)")
	if condFirst < 0 || condSecond < 0 || guard < 0 || postFirst < 0 || postSecond < 0 || postAssign < 0 {
		t.Fatalf("classic for condition/post calls were not all normalized: %q", body)
	}
	if !(condFirst < condSecond && condSecond < guard && guard < postFirst && postFirst < postSecond && postSecond < postAssign) {
		t.Fatalf("classic for condition/post temps emitted in wrong order: %q", body)
	}
}

func TestPackageDoesNotNormalizeForPostClauseWithContinue(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	for i := 0; i < 3; i = next(first()) {
		if i == 1 {
			continue
		}
		_ = i
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "_tmp_") {
		t.Fatalf("for post clause with continue was normalized unsafely: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_first())") {
		t.Fatalf("for post clause with continue shape changed unexpectedly: %q", body)
	}
}

func TestPackageNormalizesNestedForConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := 0
	for join(first(), second()) == 12 {
		total = total + 1
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "for {\n") {
		t.Fatalf("for condition was not rewritten to loop body guard: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first for condition call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second for condition call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12) {\n\t\t\tbreak\n\t\t}") {
		t.Fatalf("for condition guard did not use lifted temps: %q", body)
	}
	if !strings.Contains(body, "total = total + 1") {
		t.Fatalf("for body was not preserved: %q", body)
	}
}

func TestPackageNormalizesClassicForInitCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	total := 0
	for i := next(first()); i < 3; i = next(first()) {
		total = total + i
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("classic for init call was not lifted before loop: %q", body)
	}
	if !strings.Contains(body, "i := rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_0)\n\t\tfor {") {
		t.Fatalf("classic for init did not use lifted temp in scoped loop: %q", body)
	}
	if !strings.Contains(body, "if !(i < 3) {\n\t\t\t\tbreak\n\t\t\t}") {
		t.Fatalf("classic for condition guard was not emitted: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_first()") {
		t.Fatalf("classic for post call was not lifted into loop body: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("classic for post clause did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesClassicForConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := 0
	for i := 0; join(first(), second()) == 12; i = i + 1 {
		total = total + i
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "for i := 0; ; i = i + 1 {") {
		t.Fatalf("classic for condition was not moved into loop body guard: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first classic for condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second classic for condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12) {\n\t\t\tbreak\n\t\t}") {
		t.Fatalf("classic for condition guard did not use lifted temps: %q", body)
	}
	if !strings.Contains(body, "total = total + i") {
		t.Fatalf("classic for body was not preserved: %q", body)
	}
}

func TestPackageNormalizesWithNonCollidingTempNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func join(a int) int { return a }
func appMain() int {
	rtg_example_com_app_appMain_tmp_0 := 99
	total := join(first())
	return total + rtg_example_com_app_appMain_tmp_0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("normalization reused an existing local name: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_first()") {
		t.Fatalf("normalization did not skip colliding temp name: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("assignment did not use non-colliding temp: %q", body)
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
