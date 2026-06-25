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

func TestFileAcceptsSimpleSubsetProgram(t *testing.T) {
	file, err := parse.FileSource("ok.go", []byte(`package main

type box struct { value int }
func appMain() int {
	var b box
	b.value = 7
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

func messages(diags Diagnostics) []string {
	var out []string
	for _, diag := range diags {
		out = append(out, diag.Message)
	}
	return out
}
