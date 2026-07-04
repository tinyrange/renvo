package rtg_tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

const extendedTestsEnv = "RTG_FRONTEND_EXTENDED_TESTS"
const frontendEnv = "RTG_FRONTEND"
const targetEnv = "RTG_FRONTEND_TARGET"

var frontendOnce sync.Once
var frontendPath string
var frontendErr error

type corpusCase struct {
	name string
	dir  string
}

type frontendConfig struct {
	compiler string
	target   string
}

func TestFrontendQuickCorpus(t *testing.T) {
	runFrontendCorpus(t, "quick", true)
}

func TestFrontendExtendedCorpus(t *testing.T) {
	if os.Getenv(extendedTestsEnv) != "1" {
		t.Skipf("set %s=1 to run extended frontend corpus", extendedTestsEnv)
	}
	runFrontendCorpus(t, "extended", false)
}

func runFrontendCorpus(t *testing.T, tier string, parallel bool) {
	t.Helper()

	root := repoRoot(t)
	cases := discoverCorpusCases(t, filepath.Join(root, "rtg_tests", tier))
	if len(cases) == 0 {
		t.Fatalf("no %s frontend corpus cases found", tier)
	}

	frontend := frontendCompiler(t, root)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if parallel {
				t.Parallel()
			}
			runFrontendCorpusCase(t, frontend, tc.dir)
		})
	}
}

func discoverCorpusCases(t *testing.T, root string) []corpusCase {
	t.Helper()

	var cases []corpusCase
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "go.mod" {
			return nil
		}
		dir := filepath.Dir(path)
		if _, err := os.Stat(filepath.Join(dir, "cmd", "app")); err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		cases = append(cases, corpusCase{name: filepath.ToSlash(rel), dir: dir})
		return nil
	})
	if err != nil {
		t.Fatalf("failed to discover frontend corpus cases: %v", err)
	}
	sortCorpusCases(cases)
	return cases
}

func sortCorpusCases(cases []corpusCase) {
	for i := 1; i < len(cases); i++ {
		item := cases[i]
		j := i - 1
		for j >= 0 && cases[j].name > item.name {
			cases[j+1] = cases[j]
			j--
		}
		cases[j+1] = item
	}
}

func runFrontendCorpusCase(t *testing.T, frontend frontendConfig, dir string) {
	t.Helper()

	hostOut := runHostCase(t, dir)
	if !bytes.Equal(hostOut, []byte("PASS\n")) {
		t.Fatalf("host output = %q, want PASS\\n", string(hostOut))
	}
	if frontend.compiler == "" {
		return
	}

	out := filepath.Join(t.TempDir(), "app")
	cmd := exec.Command(frontend.compiler, "-t", frontend.target, "-s", "-o", out, "./cmd/app")
	cmd.Dir = dir
	compileOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend compile failed: %v\n%s", err, string(compileOut))
	}

	frontOut, err := exec.Command(out).CombinedOutput()
	if err != nil {
		t.Fatalf("frontend executable failed: %v\n%s", err, string(frontOut))
	}
	if !bytes.Equal(frontOut, hostOut) {
		t.Fatalf("frontend output = %q, host output = %q", string(frontOut), string(hostOut))
	}
}

func runHostCase(t *testing.T, dir string) []byte {
	t.Helper()

	cmd := exec.Command("go", "run", "./cmd/app")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("host case failed: %v\n%s", err, string(out))
	}
	return out
}

func frontendCompiler(t *testing.T, root string) frontendConfig {
	t.Helper()

	if path := os.Getenv(frontendEnv); path != "" {
		return frontendConfig{compiler: path, target: frontendTarget(t)}
	}
	if _, err := os.Stat(filepath.Join(root, "rtg", "cmd", "rtg")); err != nil {
		t.Logf("no local frontend compiler found; set %s to run corpus through a compiler", frontendEnv)
		return frontendConfig{}
	}
	frontendOnce.Do(func() {
		dir, err := os.MkdirTemp("", "rtg-frontend-corpus-*")
		if err != nil {
			frontendErr = err
			return
		}
		frontendPath = filepath.Join(dir, "rtg")
		cmd := exec.Command("go", "build", "-o", frontendPath, "./rtg/cmd/rtg")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			frontendErr = fmt.Errorf("host frontend build failed: %v\n%s", err, string(out))
		}
	})
	if frontendErr != nil {
		t.Fatal(frontendErr)
	}
	return frontendConfig{compiler: frontendPath, target: frontendTarget(t)}
}

func frontendTarget(t *testing.T) string {
	t.Helper()

	if target := os.Getenv(targetEnv); target != "" {
		if !strings.HasPrefix(target, "linux/") {
			t.Skipf("%s=%s is not runnable by this corpus harness", targetEnv, target)
		}
		return target
	}
	if runtime.GOOS != "linux" {
		t.Skipf("frontend corpus executable comparison requires linux host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	switch runtime.GOARCH {
	case "amd64", "386", "arm":
		return "linux/" + runtime.GOARCH
	case "arm64":
		return "linux/aarch64"
	default:
		t.Skipf("unsupported frontend corpus host architecture %s", runtime.GOARCH)
		return ""
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(file))
}
