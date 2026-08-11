package frontend_tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"renvo.dev/internal/testprogress"
)

const frontendEnv = "RENVO_FRONTEND"
const targetEnv = "RENVO_FRONTEND_TARGET"

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
	backend  string
	target   string
	env      []string
}

func frontendCommand(frontend frontendConfig, args ...string) *exec.Cmd {
	if frontend.backend != "" {
		args = append([]string{"-bootstrap-backend", frontend.backend}, args...)
	}
	return exec.Command(frontend.compiler, args...)
}

func TestFrontendQuickCorpus(t *testing.T) {
	runFrontendCorpus(t, "quick", true)
}

func TestFrontendRegressionCorpus(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusDirectory(t, filepath.Join(root, "frontend_tests", "regressions"), true, frontendCompiler(t, root))
}

func TestFrontendExtendedCorpus(t *testing.T) {
	runFrontendCorpus(t, "extended", false)
}

func TestFrontendStage3QuickCorpus(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusWithConfig(t, root, "quick", false, selfHostedFrontendCompiler(t, root))
}

func TestFrontendStage3RegressionCorpus(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusDirectory(t, filepath.Join(root, "frontend_tests", "regressions"), false, selfHostedFrontendCompiler(t, root))
}

func TestFrontendStage3ExtendedCorpus(t *testing.T) {
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
	runFrontendCorpusDirectory(t, filepath.Join(root, "frontend_tests", tier), parallel, frontend)
}

func runFrontendCorpusDirectory(t *testing.T, corpusRoot string, parallel bool, frontend frontendConfig) {
	t.Helper()

	cases := discoverCorpusCases(t, corpusRoot)
	if len(cases) == 0 {
		t.Fatalf("no frontend corpus cases found in %s", corpusRoot)
	}
	names := make([]string, len(cases))
	for i := range cases {
		names[i] = cases[i].name
	}
	selectedNames, err := testprogress.Filter(names, os.Getenv(testprogress.FilterEnv))
	if err != nil {
		t.Fatal(err)
	}
	selectedSet := make(map[string]bool, len(selectedNames))
	for _, name := range selectedNames {
		selectedSet[name] = true
	}
	selected := make([]corpusCase, 0, len(selectedNames))
	for _, tc := range cases {
		if selectedSet[tc.name] {
			selected = append(selected, tc)
		}
	}
	if len(selected) == 0 {
		t.Skipf("no frontend corpus cases in %s match %s=%q", corpusRoot, testprogress.FilterEnv, os.Getenv(testprogress.FilterEnv))
	}

	tasks := make(chan corpusCase, len(selected))
	for _, tc := range selected {
		tasks <- tc
	}
	close(tasks)
	progress := testprogress.New(t, filepath.Base(corpusRoot), len(selected))
	t.Cleanup(progress.Close)
	workers := frontendCorpusWorkers(parallel)
	if workers > len(selected) {
		workers = len(selected)
	}
	for worker := 0; worker < workers; worker++ {
		worker := worker
		t.Run(fmt.Sprintf("worker-%02d", worker+1), func(t *testing.T) {
			if parallel {
				t.Parallel()
			}
			for tc := range tasks {
				func() {
					done := progress.Begin(tc.name)
					defer done()
					defer func() {
						if t.Failed() {
							t.Logf("failing corpus case: %s", tc.name)
						}
					}()
					runFrontendCorpusCase(t, frontend, tc.dir)
				}()
			}
		})
	}
}

func frontendCorpusWorkers(parallel bool) int {
	if !parallel {
		return 1
	}
	if value := os.Getenv("RENVO_TEST_JOBS"); value != "" {
		if jobs, err := strconv.Atoi(value); err == nil && jobs > 0 {
			return jobs
		}
	}
	workers := runtime.NumCPU()
	if workers > 16 {
		workers = 16
	}
	if workers < 1 {
		workers = 1
	}
	return workers
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
		dir, err := os.MkdirTemp("", "renvo-frontend-selfhost-*")
		if err != nil {
			selfHostErr = err
			return
		}
		if err := os.Symlink(filepath.Join(root, "std"), filepath.Join(dir, "std")); err != nil {
			selfHostErr = fmt.Errorf("frontend std symlink failed: %w", err)
			return
		}
		selfHostBackendPath = filepath.Join(dir, "renvo-backend")
		cmd := exec.Command("go", "build", "-o", selfHostBackendPath, "./backend")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			selfHostErr = fmt.Errorf("backend build failed: %v\n%s", err, string(out))
			return
		}
		stage0 := filepath.Join(dir, "renvo-bootstrap")
		cmd = exec.Command("go", "build", "-o", stage0, "./cmd/renvobootstrap")
		cmd.Dir = root
		out, err = cmd.CombinedOutput()
		if err != nil {
			selfHostErr = fmt.Errorf("frontend stage0 host build failed: %v\n%s", err, string(out))
			return
		}
		stage1 := filepath.Join(dir, "renvo-stage1")
		stage2 := filepath.Join(dir, "renvo-stage2")
		selfHostPath = filepath.Join(dir, "renvo-stage3")
		stage0Env := []string{"RENVO_STDROOT=" + filepath.Join(root, "std")}
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
	cmd := exec.Command(compiler, "-t", target, "-s", "-o", output, "./cmd/renvo")
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

	expected, err := os.ReadFile(filepath.Join(dir, "expected.txt"))
	if err != nil {
		t.Fatalf("read checked-in expectation: %v", err)
	}
	if string(expected) != "PASS\n" {
		t.Fatalf("invalid positive-test expectation %q, want PASS\\n", string(expected))
	}
	if frontend.compiler == "" {
		return
	}

	outDir, err := os.MkdirTemp("", "renvo-frontend-corpus-case-*")
	if err != nil {
		t.Fatalf("create corpus output directory: %v", err)
	}
	defer os.RemoveAll(outDir)
	out := filepath.Join(outDir, "app")
	cmd := frontendCommand(frontend, "-t", frontend.target, "-s", "-o", out, "./cmd/app")
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
	if string(frontOut) != string(expected) {
		size := int64(-1)
		if info, statErr := os.Stat(out); statErr == nil {
			size = info.Size()
		}
		retryCmd := exec.Command(out)
		retryCmd.Dir = dir
		retryCmd.Env = []string{"PWD=" + dir}
		retryOut, retryErr := retryCmd.CombinedOutput()
		t.Fatalf("frontend output = %q, checked-in output = %q, size=%d, retryErr=%v, retryOut=%q",
			string(frontOut), string(expected), size, retryErr, string(retryOut))
	}
}

func frontendCompiler(t *testing.T, root string) frontendConfig {
	t.Helper()

	if path := os.Getenv(frontendEnv); path != "" {
		return frontendConfig{compiler: path, target: frontendTarget(t)}
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "renvo")); err != nil {
		t.Logf("no local frontend compiler found; set %s to run corpus through a compiler", frontendEnv)
		return frontendConfig{}
	}
	frontendOnce.Do(func() {
		dir, err := os.MkdirTemp("", "renvo-frontend-corpus-*")
		if err != nil {
			frontendErr = err
			return
		}
		frontendPath = filepath.Join(dir, "renvo-bootstrap")
		cmd := exec.Command("go", "build", "-o", frontendPath, "./cmd/renvobootstrap")
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			frontendErr = fmt.Errorf("host frontend build failed: %v\n%s", err, string(out))
			return
		}
		frontendBackendPath = filepath.Join(dir, "renvo-backend")
		cmd = exec.Command("go", "build", "-o", frontendBackendPath, "./backend")
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
		backend:  frontendBackendPath,
		target:   frontendTarget(t),
		env:      []string{"RENVO_STDROOT=" + filepath.Join(root, "std")},
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
