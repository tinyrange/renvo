package rtgx

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/unit"
)

func TestCompileSourceBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	out := filepath.Join(t.TempDir(), "app")
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	if err := CompileSource(src, Options{Target: "linux/amd64", Output: out, BackendRoot: root}); err != nil {
		t.Fatalf("CompileSource failed: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat output failed: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("output is not executable: %v", info.Mode())
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

func TestCompileSourceBytesBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	data, err := CompileSourceBytes(src, Options{Target: "linux/amd64", BackendRoot: root})
	if err != nil {
		t.Fatalf("CompileSourceBytes failed: %v", err)
	}
	out := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile output failed: %v", err)
	}
	cmd := exec.Command(out)
	outData, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(outData))
	}
	if string(outData) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(outData))
	}
}

func TestCompileSourceRequiresOutput(t *testing.T) {
	err := CompileSource([]byte("package main\n"), Options{Target: "linux/amd64"})
	if err == nil {
		t.Fatalf("CompileSource accepted missing output path")
	}
	if !strings.Contains(err.Error(), "missing output path") {
		t.Fatalf("error = %q", err)
	}
}

func TestCompileSourceRejectsUnsupportedTarget(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileSource([]byte("package main\n"), Options{Target: "linux/arm64", Output: out})
	if err == nil {
		t.Fatalf("CompileSource accepted unsupported target")
	}
	msg := err.Error()
	for _, want := range []string{"rtg: unsupported target: linux/arm64", "linux/amd64", "linux/aarch64", "wasi/wasm32"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestCompileUnitsBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	out := filepath.Join(t.TempDir(), "app")
	units := []unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Imports:    []string{"example.com/app/dep"},
			References: []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_dep_Print() }\n"},
			},
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
			Exports:    []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "Print", UnitName: "rtg_example_com_app_dep_Print", Body: "func rtg_example_com_app_dep_Print() int { print(\"PASS\\n\"); return 0 }\n"},
			},
		},
	}
	if err := CompileUnits(units, Options{Target: "linux/amd64", Output: out, BackendRoot: root}); err != nil {
		t.Fatalf("CompileUnits failed: %v", err)
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

func TestCompileUnitsValidatesUnitGraph(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnits([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Imports:    []string{"example.com/app/missing"},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return 0 }\n"},
		},
	}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnits accepted missing imported unit")
	}
	if !strings.Contains(err.Error(), "missing imported unit example.com/app/missing") {
		t.Fatalf("error = %q", err)
	}
}

func TestCompileUnitSourcesReportsParseErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnitSources([]unit.SourceFile{{Path: "broken.rtg.go", Source: []byte("//go:build rtg\n\npackage main\n")}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnitSources accepted malformed unit source")
	}
	if !strings.Contains(err.Error(), "broken.rtg.go: missing rtg unit metadata") {
		t.Fatalf("error = %q", err)
	}
}

func TestCompileUnitSourcesValidatesUnitGraph(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnitSources([]unit.SourceFile{{
		Path: "main.rtg.go",
		Source: []byte(`//go:build rtg

// rtg:unit example.com/app/main
package main

// rtg:import "example.com/app/missing"
// rtg:decl func appMain => rtg_example_com_app_main_appMain main.go
func rtg_example_com_app_main_appMain() int { return 0 }
`),
	}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnitSources accepted missing imported unit")
	}
	if !strings.Contains(err.Error(), "missing imported unit example.com/app/missing") {
		t.Fatalf("error = %q", err)
	}
}
