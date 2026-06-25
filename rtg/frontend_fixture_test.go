package rtg_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"j5.nz/rtg/rtg/build"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/link"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/unit"
)

const crossArchTestsEnv = "RTG_CROSS_ARCH_TESTS"

type frontendSmokeTarget struct {
	name   string
	runner []string
}

func TestHelloFixtureGoldenUnits(t *testing.T) {
	fixture := filepath.Join("testdata", "hello_module")
	units := loadFixtureUnits(t, fixture)
	for _, u := range units {
		name := emit.FileName(u.ImportPath)
		got := string(emit.Source(u))
		wantPath := filepath.Join("testdata", "golden", "hello_module", name)
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("ReadFile golden %s failed: %v", wantPath, err)
		}
		if got != string(want) {
			t.Fatalf("golden mismatch for %s\n%s", name, diffText(string(want), got))
		}
	}
}

func TestHelloFixtureFrontendMatchesHostGo(t *testing.T) {
	fixture := filepath.Join("testdata", "hello_module")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nested\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/nested/pkg/dep"

func value() int {
	total := dep.Join(dep.First(), dep.Second())
	if dep.Join(dep.First(), dep.Second()) == 12 {
		return total
	}
	return 0
}

func main() {
	if value() == 12 {
		dep.Emit(dep.First(), dep.Second())
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
func Emit(a int, b int) int {
	if Join(a, b) == 12 {
		print("PASS\n")
	}
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func runFrontendFixtureMatchesHostGo(t *testing.T, fixture string) {
	t.Helper()
	host := exec.Command("go", "run", "./cmd/app")
	host.Dir = fixture
	hostOut, err := host.CombinedOutput()
	if err != nil {
		t.Fatalf("host fixture failed: %v\n%s", err, string(hostOut))
	}
	units := loadFixtureUnits(t, fixture)
	plan, err := link.Build(units)
	if err != nil {
		t.Fatalf("link.Build failed: %v", err)
	}
	source := link.Source(plan)
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			out := filepath.Join(t.TempDir(), "hello")
			if err := rtgx.CompileSource(source, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileSource failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err != nil {
				t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
			}
			if !bytes.Equal(frontOut, hostOut) {
				t.Fatalf("frontend output = %q, host output = %q", string(frontOut), string(hostOut))
			}
		})
	}
}

func writeFixtureFile(t *testing.T, root string, path string, data string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", full, err)
	}
}

func frontendSmokeTargets(t *testing.T) []frontendSmokeTarget {
	t.Helper()
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		targets := []frontendSmokeTarget{{name: "linux/amd64"}}
		if os.Getenv(crossArchTestsEnv) == "1" {
			targets = append(targets,
				frontendSmokeTarget{name: "linux/386"},
				frontendSmokeTarget{name: "linux/aarch64", runner: []string{"qemu-aarch64"}},
				frontendSmokeTarget{name: "linux/arm", runner: []string{"qemu-arm"}},
				frontendSmokeTarget{name: "wasi/wasm32", runner: []string{"wasmtime", "run", "--dir=.", "--dir=/", "--env", "PWD", "--env", "PATH"}},
			)
		}
		return targets
	case "linux/arm64":
		return []frontendSmokeTarget{{name: "linux/aarch64"}}
	default:
		t.Skipf("no frontend smoke targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func runFrontendTarget(target frontendSmokeTarget, path string) ([]byte, error) {
	if len(target.runner) == 0 {
		cmd := exec.Command(path)
		return cmd.CombinedOutput()
	}
	args := append([]string{}, target.runner[1:]...)
	args = append(args, path)
	cmd := exec.Command(target.runner[0], args...)
	return cmd.CombinedOutput()
}

func loadFixtureUnits(t *testing.T, fixture string) []unit.Unit {
	t.Helper()
	graph, err := load.LoadEntries([]string{filepath.Join(fixture, "cmd", "app")}, load.Options{})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	units, err := build.Units(graph)
	if err != nil {
		t.Fatalf("build.Units failed: %v", err)
	}
	sort.Slice(units, func(i int, j int) bool {
		return units[i].ImportPath < units[j].ImportPath
	})
	return units
}

func diffText(want string, got string) string {
	if want == got {
		return ""
	}
	return "want:\n" + want + "\ngot:\n" + got
}
