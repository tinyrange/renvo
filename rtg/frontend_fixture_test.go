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
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
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
	out := filepath.Join(t.TempDir(), "hello")
	if err := rtgx.CompileSource(link.Source(plan), rtgx.Options{Target: "linux/amd64", Output: out}); err != nil {
		t.Fatalf("CompileSource failed: %v", err)
	}
	front := exec.Command(out)
	frontOut, err := front.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
	}
	if !bytes.Equal(frontOut, hostOut) {
		t.Fatalf("frontend output = %q, host output = %q", string(frontOut), string(hostOut))
	}
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
