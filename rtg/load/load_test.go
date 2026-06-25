package load

import (
	"os"
	"path/filepath"
	"strings"
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
	if graph.Packages[0].Files[0].UnitPath != "cmd/app/main.go" {
		t.Fatalf("main unit path = %q, want cmd/app/main.go", graph.Packages[0].Files[0].UnitPath)
	}
	if graph.Packages[1].Files[0].UnitPath != "pkg/answer/answer.go" {
		t.Fatalf("dep unit path = %q, want pkg/answer/answer.go", graph.Packages[1].Files[0].UnitPath)
	}
}

func TestLoadEntriesGroupsExplicitFilesWithoutReadingWholeDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/a.go", `package pkg

const A = 1
`)
	writeFile(t, root, "pkg/b.go", `package pkg

const B = A + 1
`)
	writeFile(t, root, "pkg/ignored.go", `package other

const C = 3
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "pkg", "b.go"), filepath.Join(root, "pkg", "a.go")}, Options{})
	if err != nil {
		t.Fatalf("LoadEntries explicit files failed: %v", err)
	}
	if len(graph.Packages) != 1 {
		t.Fatalf("loaded %d packages, want one", len(graph.Packages))
	}
	pkg := graph.Packages[0]
	if pkg.ImportPath != "example.com/app/pkg" || pkg.Name != "pkg" {
		t.Fatalf("package identity = %#v", pkg)
	}
	if len(pkg.Files) != 2 {
		t.Fatalf("files = %#v, want selected files only", pkg.Files)
	}
	if filepath.Base(pkg.Files[0].Path) != "a.go" || filepath.Base(pkg.Files[1].Path) != "b.go" {
		t.Fatalf("file order = %#v, want a.go then b.go", pkg.Files)
	}
}

func TestLoadEntriesFiltersDirectoryFilesByTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/common.go", `package pkg

const Common = 1
`)
	writeFile(t, root, "pkg/value_linux.go", `package pkg

const Value = Common + 1
`)
	writeFile(t, root, "pkg/value_windows.go", `package pkg

const Value = Common + 2
`)
	writeFile(t, root, "pkg/arch_amd64.go", `package pkg

const Arch = 64
`)
	writeFile(t, root, "pkg/arch_arm64.go", `package pkg

const Arch = 128
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "linux/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries linux/amd64 failed: %v", err)
	}
	got := packageFileNames(graph.Packages[0])
	want := []string{"arch_amd64.go", "common.go", "value_linux.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("linux/amd64 files = %v, want %v", got, want)
	}

	graph, err = LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "windows/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries windows/amd64 failed: %v", err)
	}
	got = packageFileNames(graph.Packages[0])
	want = []string{"arch_amd64.go", "common.go", "value_windows.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("windows/amd64 files = %v, want %v", got, want)
	}

	graph, err = LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "linux/aarch64"})
	if err != nil {
		t.Fatalf("LoadEntries linux/aarch64 failed: %v", err)
	}
	got = packageFileNames(graph.Packages[0])
	want = []string{"arch_arm64.go", "common.go", "value_linux.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("linux/aarch64 files = %v, want %v", got, want)
	}
}

func TestLoadEntriesFiltersDirectoryFilesByGoBuildConstraint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/common.go", `package pkg

const Common = 1
`)
	writeFile(t, root, "pkg/linux.go", `//go:build linux && amd64

package pkg

const Value = Common + 1
`)
	writeFile(t, root, "pkg/windows.go", `//go:build windows

package pkg

const Value = Common + 2
`)
	writeFile(t, root, "pkg/notwin.go", `//go:build !windows

package pkg

const NotWindows = 1
`)
	writeFile(t, root, "pkg/ignored.go", `//go:build ignore

package pkg

const Ignored = 1
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "linux/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries linux/amd64 failed: %v", err)
	}
	got := packageFileNames(graph.Packages[0])
	want := []string{"common.go", "linux.go", "notwin.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("linux/amd64 files = %v, want %v", got, want)
	}

	graph, err = LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "windows/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries windows/amd64 failed: %v", err)
	}
	got = packageFileNames(graph.Packages[0])
	want = []string{"common.go", "windows.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("windows/amd64 files = %v, want %v", got, want)
	}
}

func TestLoadEntriesFiltersDirectoryFilesByLegacyBuildConstraint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/common.go", `package pkg

const Common = 1
`)
	writeFile(t, root, "pkg/linux.go", `// +build linux,amd64

package pkg

const Linux = Common + 1
`)
	writeFile(t, root, "pkg/other.go", `// +build darwin windows

package pkg

const Other = Common + 2
`)
	writeFile(t, root, "pkg/notwin.go", `// +build !windows
// +build amd64

package pkg

const NotWindows = Common + 3
`)
	writeFile(t, root, "pkg/ignored.go", `// +build ignore

package pkg

const Ignored = Common + 4
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "linux/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries linux/amd64 failed: %v", err)
	}
	got := packageFileNames(graph.Packages[0])
	want := []string{"common.go", "linux.go", "notwin.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("linux/amd64 files = %v, want %v", got, want)
	}

	graph, err = LoadEntries([]string{filepath.Join(root, "pkg")}, Options{Target: "windows/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries windows/amd64 failed: %v", err)
	}
	got = packageFileNames(graph.Packages[0])
	want = []string{"common.go", "other.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("windows/amd64 files = %v, want %v", got, want)
	}
}

func TestLoadEntriesExplicitFilesIgnoreGoBuildConstraint(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/explicit.go", `//go:build windows

package pkg

const Explicit = 1
`)

	graph, err := LoadEntries([]string{filepath.Join(root, "pkg", "explicit.go")}, Options{Target: "linux/amd64"})
	if err != nil {
		t.Fatalf("LoadEntries explicit file failed: %v", err)
	}
	got := packageFileNames(graph.Packages[0])
	want := []string{"explicit.go"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("explicit files = %v, want %v", got, want)
	}
}

func TestLoadEntriesDeduplicatesExplicitFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "pkg/a.go", `package pkg

const A = 1
`)
	path := filepath.Join(root, "pkg", "a.go")
	graph, err := LoadEntries([]string{path, path}, Options{})
	if err != nil {
		t.Fatalf("LoadEntries explicit duplicate files failed: %v", err)
	}
	if len(graph.Packages) != 1 {
		t.Fatalf("loaded %d packages, want one", len(graph.Packages))
	}
	if len(graph.Packages[0].Files) != 1 {
		t.Fatalf("files = %#v, want one selected file", graph.Packages[0].Files)
	}
}

func TestLoadEntriesRejectsInvalidExplicitFrontendFiles(t *testing.T) {
	tests := []struct {
		name string
		file string
		want string
	}{
		{name: "non-go", file: "pkg/input.txt", want: "must be a .go source file"},
		{name: "test", file: "pkg/input_test.go", want: "must not be a Go test file"},
		{name: "unit", file: "pkg/input.rtg.go", want: "use -link for .rtg.go files"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root, "go.mod", "module example.com/app\n")
			writeFile(t, root, tt.file, `package pkg

const A = 1
`)
			_, err := LoadEntries([]string{filepath.Join(root, tt.file)}, Options{})
			if err == nil {
				t.Fatalf("LoadEntries accepted %s explicit file", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestLoadEntriesRejectsMissingStdPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "main.go", `package main

import "fmt"

func appMain() int { return 0 }
`)

	_, err := LoadEntries([]string{root}, Options{StdRoot: filepath.Join(root, "missing-std")})
	if err == nil {
		t.Fatalf("LoadEntries succeeded with missing std package")
	}
}

func TestLoadEntriesResolvesLocalReplaceImports(t *testing.T) {
	root := t.TempDir()
	libRoot := filepath.Join(t.TempDir(), "lib")
	writeFile(t, root, "go.mod", "module example.com/app\n\nreplace example.com/lib => "+filepath.ToSlash(libRoot)+"\n")
	writeFile(t, root, "main.go", `package main

import "example.com/lib/pkg/answer"

func appMain() int { return answer.Value() }
`)
	writeFile(t, libRoot, "pkg/answer/answer.go", `package answer

func Value() int { return 7 }
`)

	graph, err := LoadEntries([]string{root}, Options{})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(graph.Packages) != 2 {
		t.Fatalf("loaded %d packages, want 2", len(graph.Packages))
	}
	dep := graph.Packages[1]
	if dep.ImportPath != "example.com/lib/pkg/answer" {
		t.Fatalf("dep import path = %q, want example.com/lib/pkg/answer", dep.ImportPath)
	}
	if dep.Dir != filepath.Join(libRoot, "pkg", "answer") {
		t.Fatalf("dep dir = %q, want replaced dir", dep.Dir)
	}
	if dep.Files[0].UnitPath != "example.com/lib/pkg/answer/answer.go" {
		t.Fatalf("dep unit path = %q, want import-relative path", dep.Files[0].UnitPath)
	}
}

func TestLoadEntriesRejectsNonLocalReplaceImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\nreplace example.com/lib => example.com/other v1.2.3\n")
	writeFile(t, root, "main.go", `package main

import "example.com/lib/pkg"

func appMain() int { return pkg.Value() }
`)

	_, err := LoadEntries([]string{root}, Options{})
	if err == nil {
		t.Fatalf("LoadEntries succeeded with non-local replace")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"non-local replace target", "external module fetching is not supported"}) {
		t.Fatalf("error = %q", got)
	}
}

func TestLoadEntriesRejectsRequiredExternalModuleImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\nrequire example.com/lib v1.2.3\n")
	writeFile(t, root, "main.go", `package main

import "example.com/lib/pkg"

func appMain() int { return pkg.Value() }
`)

	_, err := LoadEntries([]string{root}, Options{})
	if err == nil {
		t.Fatalf("LoadEntries succeeded with required external module")
	}
	if got := err.Error(); got == "" || !containsAll(got, []string{"required module", "example.com/lib", "external module fetching is not supported"}) {
		t.Fatalf("error = %q", got)
	}
}

func TestLoadEntriesRejectsMalformedReplaceDirective(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app\n\nreplace example.com/lib ../lib\n")
	writeFile(t, root, "main.go", `package main

func appMain() int { return 0 }
`)
	_, err := LoadEntries([]string{root}, Options{})
	if err == nil {
		t.Fatalf("LoadEntries accepted malformed replace directive")
	}
	if !containsAll(err.Error(), []string{"go.mod", "malformed replace directive"}) {
		t.Fatalf("error = %q", err)
	}
}

func TestLoadEntriesRejectsMalformedModuleDirective(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/app extra\n")
	writeFile(t, root, "main.go", `package main

func appMain() int { return 0 }
`)
	_, err := LoadEntries([]string{root}, Options{})
	if err == nil {
		t.Fatalf("LoadEntries accepted malformed module directive")
	}
	if !containsAll(err.Error(), []string{"go.mod", "malformed module directive"}) {
		t.Fatalf("error = %q", err)
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
}

func TestLoadEntriesUsesRTGStdEnv(t *testing.T) {
	root := t.TempDir()
	stdRoot := filepath.Join(t.TempDir(), "std")
	t.Setenv("RTG_STD", stdRoot)
	writeFile(t, root, "go.mod", "module example.com/app\n")
	writeFile(t, root, "main.go", `package main

import "fmt"

func appMain() int { return fmt.PrintInt(7) }
`)
	writeFile(t, stdRoot, "fmt/fmt.go", `package fmt

func PrintInt(v int) int { return v }
`)

	graph, err := LoadEntries([]string{root}, Options{})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if len(graph.Packages) != 2 {
		t.Fatalf("loaded %d packages, want 2", len(graph.Packages))
	}
	if graph.Packages[1].Dir != filepath.Join(stdRoot, "fmt") {
		t.Fatalf("std package dir = %q", graph.Packages[1].Dir)
	}
	if graph.Packages[1].ImportPath != "fmt" {
		t.Fatalf("std import path = %q, want fmt", graph.Packages[1].ImportPath)
	}
	if graph.Packages[1].Files[0].UnitPath != "fmt/fmt.go" {
		t.Fatalf("std unit path = %q, want fmt/fmt.go", graph.Packages[1].Files[0].UnitPath)
	}
}

func TestParseSourceInfoImportBlock(t *testing.T) {
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
	if info.Imports[1].Alias != "alias" {
		t.Fatalf("aliased import = %#v", info.Imports[1])
	}
}

func TestParseSourceInfoRejectsMalformedImportBlockEntry(t *testing.T) {
	src := []byte(`package main

import (
	fmt
)

func appMain() int { return 0 }
`)
	_, err := ParseSourceInfo("input.go", src)
	if err == nil {
		t.Fatalf("ParseSourceInfo accepted malformed import block entry")
	}
	if !strings.Contains(err.Error(), "input.go: malformed import declaration") {
		t.Fatalf("error = %q", err)
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

func packageFileNames(pkg Package) []string {
	names := make([]string, 0, len(pkg.Files))
	for _, file := range pkg.Files {
		names = append(names, filepath.Base(file.Path))
	}
	return names
}

func containsAll(s string, parts []string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
