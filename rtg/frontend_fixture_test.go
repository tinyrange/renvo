package rtg_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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
	absFixture, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatalf("Abs fixture failed: %v", err)
	}
	for _, u := range units {
		name := emit.FileName(u.ImportPath)
		got := normalizeFixturePath(string(emit.Source(u)), absFixture)
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

func normalizeFixturePath(src string, fixture string) string {
	src = filepath.ToSlash(src)
	fixture = filepath.ToSlash(fixture)
	return strings.ReplaceAll(src, fixture, "$FIXTURE")
}

func diffText(want string, got string) string {
	if want == got {
		return ""
	}
	return "want:\n" + want + "\ngot:\n" + got
}
