package check

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
)

func TestFileRejectsExcludedFeatures(t *testing.T) {
	src := []byte(`package main

type Box[T any] struct { value T }
type Runner interface { Run() }
type fixed [4]int
type holder struct { values [2]int }
func takesArray(values [3]int) int { return 0 }
func useMap(m map[string]int) int { return 0 }
func makeChan(ch chan int) { go makeChan(ch); select {} }
func appMain() int {
	defer print("done")
	fn := func() int { return 1 }
	_ = fn
	for _, v := range []int{1, 2} {
		_ = v
	}
	return 0
}
`)
	file, err := parse.FileSource("bad.go", src)
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	messages := strings.Join(messages(diags), "\n")
	for _, want := range []string{
		"generics are not supported",
		"interfaces are not supported",
		"maps are not supported",
		"arrays are not supported",
		"channels are not supported",
		"goroutines are not supported",
		"select statements are not supported",
		"range is not supported",
		"defer is not supported",
		"function values and function types are not supported",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, messages)
		}
	}
}

func TestFileRejectsGenericInstantiations(t *testing.T) {
	file, err := parse.FileSource("generics.go", []byte(`package main

type Box struct { value int }
func id(value int) int { return value }

func appMain() int {
	_ = Box[int]{value: 1}
	_ = id[int](1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted generic instantiations")
	}
	msg := err.Error()
	for _, want := range []string{
		"generics.go:7:9: generics are not supported",
		"generics.go:8:8: generics are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsTypeAssertionsAndSwitches(t *testing.T) {
	file, err := parse.FileSource("assertions.go", []byte(`package main

func appMain() int {
var x any
_ = x.(int)
switch x.(type) {
}
return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted type assertion forms")
	}
	msg := err.Error()
	for _, want := range []string{
		"assertions.go:5:7: type assertions and type switches are not supported",
		"assertions.go:6:10: type assertions and type switches are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsFallthrough(t *testing.T) {
	file, err := parse.FileSource("switch.go", []byte(`package main

func appMain() int {
	switch 1 {
	case 1:
		fallthrough
	case 2:
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted fallthrough")
	}
	if !strings.Contains(err.Error(), "switch.go:6:3: fallthrough is not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsUnsupportedLiterals(t *testing.T) {
	file, err := parse.FileSource("literals.go", []byte("package main\n\nfunc appMain() int {\n\t_ = `raw`\n\t_ = 077\n\t_ = 0o77\n\t_ = 1i\n\t_ = 0x10\n\t_ = 0b10\n\t_ = 0.5\n\treturn 0\n}\n"))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported literals")
	}
	msg := err.Error()
	for _, want := range []string{
		"literals.go:4:6: raw string literals are not supported",
		"literals.go:5:6: octal literals are not supported",
		"literals.go:6:6: octal literals are not supported",
		"literals.go:7:6: imaginary literals are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	for _, allowed := range []string{"0x10", "0b10", "0.5"} {
		if strings.Contains(msg, allowed) {
			t.Fatalf("supported literal %s was diagnosed:\n%s", allowed, msg)
		}
	}
}

func TestFileRejectsIotaConst(t *testing.T) {
	file, err := parse.FileSource("iota.go", []byte(`package main

const (
	A = iota
	B
)

func appMain() int { return A + B }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted iota")
	}
	if !strings.Contains(err.Error(), "iota.go:4:6: iota is not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsAnyInterfaceTypeAlias(t *testing.T) {
	file, err := parse.FileSource("any.go", []byte(`package main

type Box struct { value any }
type Alias any
func use(value any) any { return value }
func appMain() int {
	var value any
	var pointer *any
	var values []any
	_ = value
	_ = pointer
	_ = values
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted any interface aliases")
	}
	msg := err.Error()
	for _, want := range []string{
		"any.go:3:25: interfaces are not supported",
		"any.go:4:12: interfaces are not supported",
		"any.go:5:16: interfaces are not supported",
		"any.go:5:21: interfaces are not supported",
		"any.go:7:12: interfaces are not supported",
		"any.go:8:15: interfaces are not supported",
		"any.go:9:15: interfaces are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsNestedArrayTypes(t *testing.T) {
	file, err := parse.FileSource("arrays.go", []byte(`package main

type Pointer *[3]int
type SliceOfArray [][2]int
type Box struct { values []*[4]int }
func use(values []*[5]int) int { return 0 }
func appMain() int {
	var values [][6]int
	_ = values
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted nested array types")
	}
	msg := err.Error()
	for _, want := range []string{
		"arrays.go:3:15: arrays are not supported",
		"arrays.go:4:21: arrays are not supported",
		"arrays.go:5:29: arrays are not supported",
		"arrays.go:6:20: arrays are not supported",
		"arrays.go:8:15: arrays are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAcceptsSimpleSubsetProgram(t *testing.T) {
	file, err := parse.FileSource("ok.go", []byte(`package main

type box struct { value int }
func appMain() int {
	var b box
	b.value = 7
	any := 3
	values := []int{1, 2}
	if values[0] == 1 {
		b.value = b.value + values[1] + any
	}
	return b.value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileAcceptsOrdinaryMainEntrypoint(t *testing.T) {
	file, err := parse.FileSource("main.go", []byte(`package main

func main() {
	print("PASS\n")
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileRejectsInvalidOrdinaryMainSignatures(t *testing.T) {
	file, err := parse.FileSource("main.go", []byte(`package main

func main(args []string) {}
func mainResult() {}
func mainWithResult() int { return 0 }
func main() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid ordinary main signatures")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:3:6: main function must have no parameters or results",
		"main.go:6:6: main function must have no parameters or results",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "mainResult") || strings.Contains(msg, "mainWithResult") {
		t.Fatalf("non-entrypoint helper names were diagnosed:\n%s", msg)
	}
}

func TestFileRejectsNamedResultParameters(t *testing.T) {
	file, err := parse.FileSource("results.go", []byte(`package main

func value() (out int) {
	out = 1
	return
}

func appMain() int { return value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted named result parameters")
	}
	if !strings.Contains(err.Error(), "results.go:3:15: named result parameters are not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsFullSliceExpressions(t *testing.T) {
	file, err := parse.FileSource("slice.go", []byte(`package main

func appMain() int {
	values := []int{1, 2, 3}
	x := values[1:2:3]
	return x[0]
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted full slice expression")
	}
	if !strings.Contains(err.Error(), "slice.go:5:17: full slice expressions are not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsVariadicSyntax(t *testing.T) {
	file, err := parse.FileSource("variadic.go", []byte(`package main

func sum(values ...int) int { return 0 }

func appMain() int {
	values := []int{1, 2}
	return sum(values...)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted variadic syntax")
	}
	msg := err.Error()
	for _, want := range []string{
		"variadic.go:3:17: variadic syntax is not supported",
		"variadic.go:7:19: variadic syntax is not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsMethods(t *testing.T) {
	file, err := parse.FileSource("method.go", []byte(`package main

type box struct { value int }

func (b box) Value() int {
	return b.value
}

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted method declaration")
	}
	if !strings.Contains(err.Error(), "method.go:5:14: methods are not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsUnsupportedImportForms(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	_ "example.com/side"
	. "example.com/dot"
	alias "example.com/alias"
)

func appMain() int { return alias.Value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported import forms")
	}
	msg := err.Error()
	for _, want := range []string{
		"imports.go:4:4: blank imports are not supported",
		"imports.go:5:4: dot imports are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "alias") {
		t.Fatalf("ordinary import alias was rejected:\n%s", msg)
	}
}

func TestFileRejectsDuplicateImportNames(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	"example.com/one/fmt"
	"fmt"
	lib "example.com/lib/a"
	lib "example.com/lib/b"
)

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted duplicate import names")
	}
	msg := err.Error()
	for _, want := range []string{
		"imports.go:5:2: duplicate import name: fmt",
		"imports.go:7:6: duplicate import name: lib",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAcceptsDistinctImportAliases(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	fmt1 "example.com/one/fmt"
	fmt2 "fmt"
)

func appMain() int { return fmt1.Value() + fmt2.Value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileRejectsUnusedImports(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	"fmt"
	alias "example.com/alias"
)

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unused imports")
	}
	msg := err.Error()
	for _, want := range []string{
		"imports.go:4:2: unused import: fmt",
		"imports.go:5:8: unused import: alias",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileDoesNotCountLocalShadowSelectorAsImportUse(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import "example.com/dep"

type localDep struct { Value int }

func appMain() int {
	dep := localDep{Value: 1}
	return dep.Value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unused import shadowed by local selector")
	}
	if !strings.Contains(err.Error(), "imports.go:3:8: unused import: dep") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsUnsupportedBuiltins(t *testing.T) {
	file, err := parse.FileSource("builtins.go", []byte(`package main

func appMain() int {
	panic("bad")
	println("bad")
	close(nil)
	delete(nil, "key")
	_ = cap(nil)
	_ = new(int)
	_ = real(1)
	_ = imag(1)
	_ = complex(1, 2)
	_ = recover()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported builtins")
	}
	msg := err.Error()
	for _, want := range []string{
		"builtins.go:4:2: unsupported builtin: panic",
		"builtins.go:5:2: unsupported builtin: println",
		"builtins.go:6:2: unsupported builtin: close",
		"builtins.go:7:2: unsupported builtin: delete",
		"builtins.go:8:6: unsupported builtin: cap",
		"builtins.go:9:6: unsupported builtin: new",
		"builtins.go:10:6: unsupported builtin: real",
		"builtins.go:11:6: unsupported builtin: imag",
		"builtins.go:12:6: unsupported builtin: complex",
		"builtins.go:13:6: unsupported builtin: recover",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsSupportedBuiltins(t *testing.T) {
	file, err := parse.FileSource("builtins.go", []byte(`package main

func appMain() int {
	values := []int{1}
	dst := []int{0}
	_ = len(values)
	_ = append(values, 2)
	_ = copy(dst, values)
	_ = make([]int, 1)
	print("ok")
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestGraphRejectsDuplicatePackageLevelNames(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package pkg

func Value() int { return 1 }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package pkg

func Value() int { return 2 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with duplicate declaration")
	}
	msg := err.Error()
	if !strings.Contains(msg, "a.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing first duplicate diagnostic in:\n%s", msg)
	}
	if !strings.Contains(msg, "b.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing second duplicate diagnostic in:\n%s", msg)
	}
}

func TestGraphRejectsDuplicateGroupedPackageLevelNames(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "group.go",
						Source: []byte(`package pkg

const (
	Answer = 1
	Other = 2
)

var (
	Other = 3
)
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with duplicate grouped declaration")
	}
	msg := err.Error()
	for _, want := range []string{
		"group.go:5:2: duplicate package-level declaration: Other",
		"group.go:9:2: duplicate package-level declaration: Other",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsUnresolvedImportedSelectors(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	return dep.Missing() + dep.hidden()
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Value() int { return 1 }
func hidden() int { return 2 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with unresolved imported selectors")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:6:13: unresolved imported selector: example.com/app/dep.Missing",
		"main.go:6:29: unresolved imported selector: example.com/app/dep.hidden",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphIgnoresLocalShadowedImportSelector(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "example.com/app/dep"

type localDep struct { hidden int }

func useImport() int {
	return dep.Value()
}

func appMain() int {
	dep := localDep{hidden: 1}
	return dep.hidden + useImport()
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected local shadowed import selector: %v", err)
	}
}

func TestGraphUsesFileScopedImportAliases(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package main

import first "example.com/app/dep"

func A() int { return first.Value() }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package main

import second "example.com/app/dep"

func appMain() int { return A() + second.Value() }
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected file-scoped import aliases: %v", err)
	}
}

func TestGraphDoesNotLeakImportAliasesAcrossFiles(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package main

import dep "example.com/app/dep"

func A() int { return dep.Value() }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package main

type localDep struct { hidden int }

func appMain() int {
	dep := localDep{hidden: 1}
	return dep.hidden + A()
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected local selector through leaked import alias: %v", err)
	}
}

func TestGraphAcceptsExportedImportedSelectors(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	return dep.Value()
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected exported selector: %v", err)
	}
}

func TestGraphPreservesParseDiagnosticPositions(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

bad := 1
`),
					},
					{
						Path:   "broken.go",
						Source: []byte("package main\n\nfunc appMain() int { print(\"unterminated)\n"),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted parse errors")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:3:1: expected top-level declaration",
		"broken.go:3:28: unterminated literal",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "main.go:1:1:") || strings.Contains(msg, "broken.go:1:1:") {
		t.Fatalf("parse diagnostic was wrapped at 1:1:\n%s", msg)
	}
}

func messages(diags Diagnostics) []string {
	var out []string
	for _, diag := range diags {
		out = append(out, diag.Message)
	}
	return out
}
