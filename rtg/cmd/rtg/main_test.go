package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunEmitUnit(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func appMain() int {
	return 0
}
`)
	out := filepath.Join(root, "out.rtg.go")
	cfg := config{output: out, emitUnit: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	if err := run(cfg); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "//go:build rtg\n") {
		t.Fatalf("emitted unit missing build tag:\n%s", src)
	}
	if !strings.Contains(src, "// rtg:unit example.com/app/cmd/app\n") {
		t.Fatalf("emitted unit missing import path:\n%s", src)
	}
	if strings.Contains(src, "import ") {
		t.Fatalf("emitted unit retained import declaration:\n%s", src)
	}
	if !strings.Contains(src, "// rtg:decl func appMain => rtg_example_com_app_cmd_app_appMain ") {
		t.Fatalf("emitted unit missing parsed declaration marker:\n%s", src)
	}
	if !strings.Contains(src, "func rtg_example_com_app_cmd_app_appMain() int") {
		t.Fatalf("emitted unit did not rewrite appMain symbol:\n%s", src)
	}
}

func TestRunEmitUnitDefaultsToCacheDirectory(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func appMain() int {
	return 0
}
`)
	cfg := config{emitUnit: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	if err := run(cfg); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	unitPath := filepath.Join(root, ".rtg", "units", "example_com_app_cmd_app.rtg.go")
	data, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("ReadFile default unit failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "//go:build rtg\n") {
		t.Fatalf("default emitted unit missing build tag:\n%s", src)
	}
	if !strings.Contains(src, "// rtg:unit example.com/app/cmd/app\n") {
		t.Fatalf("default emitted unit missing import path:\n%s", src)
	}
	sourceDirUnit := filepath.Join(root, "cmd", "app", "example_com_app_cmd_app.rtg.go")
	if _, err := os.Stat(sourceDirUnit); !os.IsNotExist(err) {
		t.Fatalf("unit was emitted beside source at %s", sourceDirUnit)
	}
}

func TestRunEmitUnitRewritesLoadedPackageSelector(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

import "example.com/app/pkg/answer"

func appMain() int {
	return answer.Value()
}
`)
	writeCLIFile(t, root, "pkg/answer/answer.go", `package answer

func Value() int {
	return 7
}
`)
	out := filepath.Join(root, "units")
	cfg := config{output: out, emitUnit: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	if err := run(cfg); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	mainUnit := filepath.Join(out, "example_com_app_cmd_app.rtg.go")
	depUnit := filepath.Join(out, "example_com_app_pkg_answer.rtg.go")
	data, err := os.ReadFile(mainUnit)
	if err != nil {
		t.Fatalf("ReadFile main unit failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "// rtg:ref example.com/app/pkg/answer Value => rtg_example_com_app_pkg_answer_Value\n") {
		t.Fatalf("emitted unit missing reference:\n%s", src)
	}
	if !strings.Contains(src, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("emitted unit did not rewrite imported selector:\n%s", src)
	}
	data, err = os.ReadFile(depUnit)
	if err != nil {
		t.Fatalf("ReadFile dep unit failed: %v", err)
	}
	src = string(data)
	if !strings.Contains(src, "// rtg:unit example.com/app/pkg/answer\n") {
		t.Fatalf("dep unit missing identity:\n%s", src)
	}
	if !strings.Contains(src, "// rtg:export Value => rtg_example_com_app_pkg_answer_Value\n") {
		t.Fatalf("dep unit missing export:\n%s", src)
	}
}

func TestRunEmitUnitWritesStdDependencyUnit(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

import "fmt"

func appMain() int {
	return fmt.PrintInt(7)
}
`)
	out := filepath.Join(root, "units")
	cfg := config{output: out, emitUnit: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	if err := run(cfg); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	mainUnit := filepath.Join(out, "example_com_app_cmd_app.rtg.go")
	stdUnit := filepath.Join(out, "fmt.rtg.go")
	data, err := os.ReadFile(mainUnit)
	if err != nil {
		t.Fatalf("ReadFile main unit failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, "// rtg:ref fmt PrintInt => rtg_fmt_PrintInt\n") {
		t.Fatalf("main unit missing std reference:\n%s", src)
	}
	if !strings.Contains(src, "return rtg_fmt_PrintInt(7)") {
		t.Fatalf("main unit did not rewrite std selector:\n%s", src)
	}
	data, err = os.ReadFile(stdUnit)
	if err != nil {
		t.Fatalf("ReadFile std unit failed: %v", err)
	}
	src = string(data)
	if !strings.Contains(src, "// rtg:unit fmt\n") {
		t.Fatalf("std unit missing identity:\n%s", src)
	}
	if !strings.Contains(src, "// rtg:export PrintInt => rtg_fmt_PrintInt\n") {
		t.Fatalf("std unit missing export:\n%s", src)
	}
}

func TestRunEmitUnitRejectsFileOutputForPackageGraph(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

import "example.com/app/pkg/answer"

func appMain() int { return answer.Value() }
`)
	writeCLIFile(t, root, "pkg/answer/answer.go", `package answer

func Value() int { return 7 }
`)
	out := filepath.Join(root, "out.rtg.go")
	cfg := config{output: out, emitUnit: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	err := run(cfg)
	if err == nil {
		t.Fatalf("run succeeded with file output for package graph")
	}
	if !strings.Contains(err.Error(), "requires output directory") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunCheckRejectsExcludedFeature(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func appMain() int {
	ch := make(chan int)
	go func() { ch <- 1 }()
	return <-ch
}
`)
	cfg := config{check: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	err := run(cfg)
	if err == nil {
		t.Fatalf("run check succeeded for excluded features")
	}
	msg := err.Error()
	for _, want := range []string{"channels are not supported", "goroutines are not supported"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestRunCheckAcceptsSimplePackage(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func appMain() int {
	return 0
}
`)
	cfg := config{check: true, inputs: []string{filepath.Join(root, "cmd", "app")}}
	if err := run(cfg); err != nil {
		t.Fatalf("run check failed: %v", err)
	}
}

func TestRunBuildCompilesLinkedPackageGraph(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

import "example.com/app/pkg/answer"

func appMain() int {
	return answer.Print()
}
`)
	writeCLIFile(t, root, "pkg/answer/answer.go", `package answer

func Print() int {
	print("PASS\n")
	return 0
}
`)
	out := filepath.Join(root, "app")
	if err := run(config{output: out, inputs: []string{filepath.Join(root, "cmd", "app")}}); err != nil {
		t.Fatalf("run build failed: %v", err)
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(data))
	}
}

func TestRunBuildRequiresOutput(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func appMain() int { return 0 }
`)
	err := run(config{inputs: []string{filepath.Join(root, "cmd", "app")}})
	if err == nil {
		t.Fatalf("run build succeeded without output")
	}
	if !strings.Contains(err.Error(), "build requires -o") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunBuildRequiresAppMainEntrypoint(t *testing.T) {
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

func main() {}
`)
	err := run(config{output: filepath.Join(root, "app"), inputs: []string{filepath.Join(root, "cmd", "app")}})
	if err == nil {
		t.Fatalf("run build succeeded without appMain")
	}
	if !strings.Contains(err.Error(), "missing entrypoint") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunLinkValidatesUnitReferences(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root := t.TempDir()
	mainUnit := filepath.Join(root, "main.rtg.go")
	depUnit := filepath.Join(root, "dep.rtg.go")
	out := filepath.Join(root, "linked")
	writeCLIFile(t, root, "main.rtg.go", `//go:build rtg

// Code generated by rtg; DO NOT EDIT.
// rtg:unit example.com/app/main
package main

// rtg:ref example.com/app/dep Print => rtg_example_com_app_dep_Print
// rtg:decl func appMain => rtg_example_com_app_main_appMain main.go
func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_dep_Print() }
`)
	writeCLIFile(t, root, "dep.rtg.go", `//go:build rtg

// Code generated by rtg; DO NOT EDIT.
// rtg:unit example.com/app/dep
package dep

// rtg:export Print => rtg_example_com_app_dep_Print
// rtg:decl func Print => rtg_example_com_app_dep_Print dep.go
func rtg_example_com_app_dep_Print() int { print("PASS\n"); return 0 }
`)
	cfg := config{link: true, output: out, inputs: []string{mainUnit, depUnit}}
	if err := run(cfg); err != nil {
		t.Fatalf("run link failed: %v", err)
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("linked app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("linked app output = %q", string(data))
	}
}

func TestRunLinkAcceptsUnitDirectory(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root := t.TempDir()
	writeCLIFile(t, root, "go.mod", "module example.com/app\n")
	writeCLIFile(t, root, "cmd/app/main.go", `package main

import "example.com/app/pkg/answer"

func appMain() int {
	return answer.Print()
}
`)
	writeCLIFile(t, root, "pkg/answer/answer.go", `package answer

func Print() int {
	print("PASS\n")
	return 0
}
`)
	unitDir := filepath.Join(root, "units")
	if err := run(config{emitUnit: true, output: unitDir, inputs: []string{filepath.Join(root, "cmd", "app")}}); err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	out := filepath.Join(root, "linked")
	if err := run(config{link: true, output: out, inputs: []string{unitDir}}); err != nil {
		t.Fatalf("link failed: %v", err)
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("linked app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("linked app output = %q", string(data))
	}
}

func TestRunLinkRequiresInputUnits(t *testing.T) {
	root := t.TempDir()
	err := run(config{link: true, output: filepath.Join(root, "linked.rtg.go")})
	if err == nil {
		t.Fatalf("link succeeded without input units")
	}
	if !strings.Contains(err.Error(), "requires input units") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunLinkRejectsDirectoryWithoutUnits(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "linked.rtg.go")
	err := run(config{link: true, output: out, inputs: []string{root}})
	if err == nil {
		t.Fatalf("link succeeded with empty unit directory")
	}
	if !strings.Contains(err.Error(), "no unit files") {
		t.Fatalf("error = %q", err)
	}
}

func TestRunLinkRejectsMissingExport(t *testing.T) {
	root := t.TempDir()
	mainUnit := filepath.Join(root, "main.rtg.go")
	out := filepath.Join(root, "link.txt")
	writeCLIFile(t, root, "main.rtg.go", `//go:build rtg

// Code generated by rtg; DO NOT EDIT.
// rtg:unit example.com/app/main
package main

// rtg:ref example.com/app/dep Value => rtg_example_com_app_dep_Value
`)
	cfg := config{link: true, output: out, inputs: []string{mainUnit}}
	err := run(cfg)
	if err == nil {
		t.Fatalf("run link succeeded with missing export")
	}
	if !strings.Contains(err.Error(), "unresolved reference") {
		t.Fatalf("error = %q", err)
	}
}

func writeCLIFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}
