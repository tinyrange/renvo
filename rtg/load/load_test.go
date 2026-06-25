package load

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEntriesResolvesLocalModuleImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "cmd/app/main.go", `package main

import "example.com/app/pkg/answer"

func appMain() int {
	return answer.Value()
}
`)
	writeFile(t, root, "pkg/answer/answer.go", `package answer

func Value() int {
	return 7
}
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "cmd", "app")}, Options{})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if graph.Module.Path != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", graph.Module.Path)
	}
	if len(graph.Packages) != 2 {
		t.Fatalf("loaded %d packages, want 2", len(graph.Packages))
	}
	if graph.Packages[0].ImportPath != "example.com/app/cmd/app" {
		t.Fatalf("first import path = %q", graph.Packages[0].ImportPath)
	}
	if graph.Packages[1].ImportPath != "example.com/app/pkg/answer" {
		t.Fatalf("second import path = %q", graph.Packages[1].ImportPath)
	}
	if graph.Packages[0].ImportNames["example.com/app/pkg/answer"] != "answer" {
		t.Fatalf("import local name = %q", graph.Packages[0].ImportNames["example.com/app/pkg/answer"])
	}
}

func TestLoadEntriesRejectsMissingStdPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "main.go", `package main

import "fmt"

func appMain() int { return 0 }
`)

	_, err := LoadEntries([]string{root}, Options{})
	if err == nil {
		t.Fatalf("LoadEntries succeeded with missing std package")
	}
}

func TestLoadEntriesResolvesStdImportsWithStdIdentity(t *testing.T) {
	root := t.TempDir()
	stdRoot := filepath.Join(root, "rtgstd")
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "main.go", `package main

import "fmt"

func appMain() int { return fmt.PrintInt(7) }
`)
	writeFile(t, stdRoot, "fmt/fmt.go", `package fmt

func PrintInt(v int) int { return v }
`)

	graph, err := LoadEntries([]string{root}, Options{StdRoot: stdRoot})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(graph.Packages) != 2 {
		t.Fatalf("loaded %d packages, want 2", len(graph.Packages))
	}
	if graph.Packages[0].ImportPath != "example.com/app" {
		t.Fatalf("main import path = %q, want example.com/app", graph.Packages[0].ImportPath)
	}
	if graph.Packages[1].ImportPath != "fmt" {
		t.Fatalf("std import path = %q, want fmt", graph.Packages[1].ImportPath)
	}
	if graph.Packages[0].ImportNames["fmt"] != "fmt" {
		t.Fatalf("std import local name = %q", graph.Packages[0].ImportNames["fmt"])
	}
}

func TestParseSourceInfoImportBlockAndBody(t *testing.T) {
	src := []byte(`package main

import (
	_ "example.com/side"
	alias "example.com/alias"
)

func appMain() int { return 0 }
`)
	info, err := ParseSourceInfo("input.go", src)
	if err != nil {
		t.Fatalf("ParseSourceInfo failed: %v", err)
	}
	if info.PackageName != "main" {
		t.Fatalf("package = %q, want main", info.PackageName)
	}
	if len(info.Imports) != 2 {
		t.Fatalf("imports = %v, want 2 imports", info.Imports)
	}
	if info.Imports[1].Alias != "alias" || info.Imports[1].Name != "alias" {
		t.Fatalf("aliased import = %#v", info.Imports[1])
	}
	body := string(src[info.BodyStart:])
	if body == "" || body[0] == 'i' {
		t.Fatalf("body still starts in import block: %q", body)
	}
}

func writeFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
