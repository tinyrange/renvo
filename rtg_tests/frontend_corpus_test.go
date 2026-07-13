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
const selfHostTestsEnv = "RTG_FRONTEND_SELFHOST_TESTS"

var frontendOnce sync.Once
var frontendPath string
var frontendBackendPath string
var frontendErr error
var selfHostOnce sync.Once
var selfHostPath string
var selfHostBackendPath string
var selfHostErr error

type corpusCase struct {
	name string
	dir  string
}

type frontendConfig struct {
	compiler string
	target   string
	env      []string
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

func TestFrontendStage3QuickCorpus(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend corpus", selfHostTestsEnv)
	}
	root := repoRoot(t)
	runFrontendCorpusWithConfig(t, root, "quick", false, selfHostedFrontendCompiler(t, root))
}

func TestFrontendStage3ExtendedCorpus(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend corpus", selfHostTestsEnv)
	}
	if os.Getenv(extendedTestsEnv) != "1" {
		t.Skipf("set %s=1 to run extended frontend corpus", extendedTestsEnv)
	}
	root := repoRoot(t)
	runFrontendCorpusWithConfig(t, root, "extended", false, selfHostedFrontendCompiler(t, root))
}

func runFrontendCorpus(t *testing.T, tier string, parallel bool) {
	t.Helper()

	root := repoRoot(t)
	runFrontendCorpusWithConfig(t, root, tier, parallel, frontendCompiler(t, root))
}

func runFrontendCorpusWithConfig(t *testing.T, root string, tier string, parallel bool, frontend frontendConfig) {
	t.Helper()

	cases := discoverCorpusCases(t, filepath.Join(root, "rtg_tests", tier))
	if len(cases) == 0 {
		t.Fatalf("no %s frontend corpus cases found", tier)
	}

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

func selfHostedFrontendCompiler(t *testing.T, root string) frontendConfig {
	t.Helper()

	target := frontendTarget(t)
	selfHostOnce.Do(func() {
		dir, err := os.MkdirTemp("", "rtg-frontend-selfhost-*")
		if err != nil {
			selfHostErr = err
			return
		}
		if err := os.Symlink(filepath.Join(root, "rtg", "std"), filepath.Join(dir, "std")); err != nil {
			selfHostErr = fmt.Errorf("frontend std symlink failed: %w", err)
			return
		}
		selfHostBackendPath = filepath.Join(dir, "rtgx-backend")
		cmd := exec.Command("go", "build", "-o", selfHostBackendPath, ".")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			selfHostErr = fmt.Errorf("backend build failed: %v\n%s", err, string(out))
			return
		}
		stage0 := filepath.Join(dir, "rtg-stage0")
		cmd = exec.Command("go", "build", "-o", stage0, "./rtg/cmd/rtg")
		cmd.Dir = root
		out, err = cmd.CombinedOutput()
		if err != nil {
			selfHostErr = fmt.Errorf("frontend stage0 host build failed: %v\n%s", err, string(out))
			return
		}
		stage1 := filepath.Join(dir, "rtg-stage1")
		stage2 := filepath.Join(dir, "rtg-stage2")
		selfHostPath = filepath.Join(dir, "rtg-stage3")
		stage0Env := []string{"RTG_BACKEND=" + selfHostBackendPath, "RTG_STDROOT=" + filepath.Join(root, "rtg", "std")}
		if err := compileFrontendSource(root, stage0, target, stage1, stage0Env); err != nil {
			selfHostErr = fmt.Errorf("frontend stage1 build failed: %w", err)
			return
		}
		if err := compileFrontendSource(root, stage1, target, stage2, nil); err != nil {
			selfHostErr = fmt.Errorf("frontend stage2 build failed: %w", err)
			return
		}
		if err := compileFrontendSource(root, stage2, target, selfHostPath, nil); err != nil {
			selfHostErr = fmt.Errorf("frontend stage3 build failed: %w", err)
			return
		}
	})
	if selfHostErr != nil {
		t.Fatal(selfHostErr)
	}
	return frontendConfig{
		compiler: selfHostPath,
		target:   target,
	}
}

func compileFrontendSource(root string, compiler string, target string, output string, env []string) error {
	cmd := exec.Command(compiler, "-t", target, "-s", "-o", output, "./rtg/cmd/rtg")
	cmd.Dir = root
	cmd.Env = frontendCommandEnv(env, root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, string(out))
	}
	return nil
}

func frontendCommandEnv(extra []string, pwd string) []string {
	env := make([]string, 0, len(extra)+1)
	for _, item := range extra {
		if envKey(item) != "PWD" {
			env = append(env, item)
		}
	}
	env = append(env, "PWD="+pwd)
	return env
}

func envKey(item string) string {
	for i := 0; i < len(item); i++ {
		if item[i] == '=' {
			return item[:i]
		}
	}
	return item
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
	cmd.Env = frontendCommandEnv(frontend.env, dir)
	compileOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend compile failed: %v\n%s", err, string(compileOut))
	}

	runCmd := exec.Command(out)
	runCmd.Dir = dir
	runCmd.Env = []string{"PWD=" + dir}
	frontOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("frontend executable failed: %v\n%s", err, string(frontOut))
	}
	if !bytes.Equal(frontOut, hostOut) {
		size := int64(-1)
		if info, statErr := os.Stat(out); statErr == nil {
			size = info.Size()
		}
		retryCmd := exec.Command(out)
		retryCmd.Dir = dir
		retryCmd.Env = []string{"PWD=" + dir}
		retryOut, retryErr := retryCmd.CombinedOutput()
		t.Fatalf("frontend output = %q, host output = %q, size=%d, retryErr=%v, retryOut=%q",
			string(frontOut), string(hostOut), size, retryErr, string(retryOut))
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
			return
		}
		frontendBackendPath = filepath.Join(dir, "rtgx-backend")
		cmd = exec.Command("go", "build", "-o", frontendBackendPath, ".")
		cmd.Dir = root
		out, err = cmd.CombinedOutput()
		if err != nil {
			frontendErr = fmt.Errorf("backend build failed: %v\n%s", err, string(out))
		}
	})
	if frontendErr != nil {
		t.Fatal(frontendErr)
	}
	return frontendConfig{
		compiler: frontendPath,
		target:   frontendTarget(t),
		env:      []string{"RTG_BACKEND=" + frontendBackendPath, "RTG_STDROOT=" + filepath.Join(root, "rtg", "std")},
	}
}

func frontendTarget(t *testing.T) string {
	t.Helper()

	if target := os.Getenv(targetEnv); target != "" {
		if !strings.HasPrefix(target, "linux/") && target != "darwin/arm64" {
			t.Skipf("%s=%s is not runnable by this corpus harness", targetEnv, target)
		}
		return target
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "darwin/arm64"
	}
	if runtime.GOOS != "linux" {
		t.Skipf("frontend corpus executable comparison requires a supported host, got %s/%s", runtime.GOOS, runtime.GOARCH)
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
