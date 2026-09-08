package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"renvo.dev/internal/testprogress"
)

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func resetRuntime() {
	for k := range files {
		if k >= 3 {
			files[k].Close()
		}
		delete(files, k)
	}

	files = make(map[int]file)
	files[0] = os.Stdin
	files[1] = os.Stdout
	files[2] = os.Stderr
}

type targetConfig struct {
	os   string
	arch string
}

const crossArchTestsEnv = "RENVO_CROSS_ARCH_TESTS"

func getCompilerFiles(config targetConfig) ([]string, error) {
	switch config.os + "/" + config.arch {
	case "linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "wasi/wasm32", "darwin/arm64":
	default:
		return nil, fmt.Errorf("unsupported OS/architecture combination: %s/%s", config.os, config.arch)
	}

	data, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		return nil, fmt.Errorf("read compiler source manifest: %w", err)
	}
	var files []string
	for _, line := range strings.Split(string(data), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			files = append(files, path)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("compiler source manifest is empty")
	}
	return files, nil
}

type compilerTarget struct {
	name     string
	files    []string
	emulated bool
	runner   []string
}

func supportedCompilerTargets(t *testing.T) []compilerTarget {
	t.Helper()

	var targets []compilerTarget
	configs := []targetConfig{}

	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		configs = []targetConfig{
			{os: "linux", arch: "amd64"},
		}
		if crossArchTestsEnabled() {
			configs = append(configs,
				targetConfig{os: "linux", arch: "386"},
				targetConfig{os: "linux", arch: "aarch64"},
				targetConfig{os: "linux", arch: "arm"},
				targetConfig{os: "wasi", arch: "wasm32"},
			)
		}
	case "linux/arm64":
		configs = []targetConfig{
			{os: "linux", arch: "aarch64"},
		}
	case "darwin/arm64":
		configs = []targetConfig{
			{os: "darwin", arch: "arm64"},
		}
	default:
		t.Skipf("no Renvo compiler targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
	for _, config := range configs {
		files, err := getCompilerFiles(config)
		if err != nil {
			t.Fatalf("failed to get compiler files for target %s/%s: %v", config.os, config.arch, err)
		}
		targetName := fmt.Sprintf("%s/%s", config.os, config.arch)
		target := compilerTarget{name: targetName, files: files}
		if runtime.GOARCH == "amd64" && config.arch == "aarch64" {
			target.emulated = true
			target.runner = []string{"qemu-aarch64"}
		}
		if runtime.GOARCH == "amd64" && config.arch == "arm" {
			target.emulated = true
			target.runner = []string{"qemu-arm"}
		}
		if config.os == "wasi" && config.arch == "wasm32" {
			target.emulated = true
			target.runner = []string{"wasmtime", "run", "--dir=.", "--dir=/", "--env", "PWD", "--env", "PATH"}
		}
		targets = append(targets, target)
	}
	return targets
}

func crossArchTestsEnabled() bool {
	return os.Getenv(crossArchTestsEnv) == "1"
}

func TestSupportedCompilerTargetsDefaultNativeOnly(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("target selection regression is linux/amd64-specific, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	t.Setenv(crossArchTestsEnv, "")

	targets := supportedCompilerTargets(t)
	if len(targets) != 1 {
		t.Fatalf("default targets = %#v, want one native target", targets)
	}
	if targets[0].name != "linux/amd64" {
		t.Fatalf("default target = %q, want linux/amd64", targets[0].name)
	}
}

func TestSupportedCompilerTargetsCrossArchOptIn(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("target selection regression is linux/amd64-specific, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	t.Setenv(crossArchTestsEnv, "1")

	targets := supportedCompilerTargets(t)
	var names []string
	for _, target := range targets {
		names = append(names, target.name)
	}
	want := []string{"linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "wasi/wasm32"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("opt-in targets = %v, want %v", names, want)
	}
}

func (target compilerTarget) safeName() string {
	return strings.ReplaceAll(target.name, "/", "-")
}

func skipIfTargetRunnerMissing(t *testing.T, target compilerTarget) {
	t.Helper()
	if len(target.runner) == 0 {
		return
	}
	if _, err := exec.LookPath(target.runner[0]); err != nil {
		t.Skipf("runner %s is not installed", target.runner[0])
	}
}

func compile(inputFiles []string, outputFile string) error {
	resetRuntime()

	var input []int
	for _, path := range inputFiles {
		fd := open(path, O_RDONLY)
		if fd < 0 {
			return fmt.Errorf("failed to open input file: %s", path)
		}
		input = append(input, fd)
	}

	outputFd := open(outputFile, O_RDWR|O_CREATE|O_TRUNC)
	if outputFd < 0 {
		return fmt.Errorf("failed to open output file: %s", outputFile)
	}

	err := 1
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		err = compileDarwinArm64(input, outputFd)
	} else if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		err = compileLinuxAarch64(input, outputFd)
	} else {
		err = compileLinuxAmd64(input, outputFd)
	}
	if err != 0 {
		return fmt.Errorf("compilation failed")
	}
	if chmod(outputFd, 0755) != 0 {
		return fmt.Errorf("failed to set output file permissions")
	}
	close(outputFd)

	return nil
}

func runCommand(t *testing.T, path string, args ...string) (commandResult, error) {
	t.Helper()
	return runCommandInDir(t, t.TempDir(), path, args...)
}

func runCommandInDir(t *testing.T, dir string, path string, args ...string) (commandResult, error) {
	t.Helper()
	release := acquireTestProcess()
	defer release()

	var result commandResult
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	if cmd.ProcessState != nil {
		result.exitCode = cmd.ProcessState.ExitCode()
		return result, nil
	}
	return result, err
}

func runCompilerBinary(t *testing.T, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-o", outputFile}, inputFiles...)
	result, err := runCommand(t, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func runTargetCommand(t *testing.T, target compilerTarget, path string, args ...string) (commandResult, error) {
	t.Helper()
	if len(target.runner) > 0 {
		runArgs := append([]string{path}, args...)
		return runCommand(t, target.runner[0], append(target.runner[1:], runArgs...)...)
	}
	return runCommand(t, path, args...)
}

func runTargetCompilerBinary(t *testing.T, target compilerTarget, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-t", target.name, "-o", outputFile}, inputFiles...)
	result, err := runTargetCommand(t, target, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func runHostCompilerBinaryForTarget(t *testing.T, target compilerTarget, path string, outputFile string, inputFiles []string) error {
	t.Helper()
	args := append([]string{"-t", target.name, "-o", outputFile}, inputFiles...)
	result, err := runCommand(t, path, args...)
	if err != nil {
		return err
	}
	if result.exitCode != 0 {
		return fmt.Errorf("exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	return nil
}

func TestCompilerTargetDiagnostics(t *testing.T) {
	if runtime.GOOS+"/"+runtime.GOARCH != "linux/amd64" {
		t.Skipf("compiler target diagnostics require linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	files, err := getCompilerFiles(targetConfig{os: "linux", arch: "amd64"})
	if err != nil {
		t.Fatalf("failed to get compiler files: %v", err)
	}

	outDir := t.TempDir()
	stage0 := filepath.Join(outDir, "stage0")
	if err := compile(files, stage0); err != nil {
		t.Fatalf("stage0 compilation failed: %v", err)
	}

	checkFailure := func(name string, args []string, wants []string) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			result, err := runCommand(t, stage0, args...)
			if err != nil {
				t.Fatalf("compiler execution failed: %v", err)
			}
			if result.exitCode == 0 {
				t.Fatalf("compiler accepted invalid arguments\nstdout: %sstderr: %s", result.stdout, result.stderr)
			}
			for _, want := range wants {
				if !strings.Contains(result.stderr, want) {
					t.Fatalf("diagnostic missing %q\nstdout: %sstderr: %s", want, result.stdout, result.stderr)
				}
			}
		})
	}

	outputFile := filepath.Join(outDir, "out")
	checkFailure(
		"unsupported target",
		[]string{"-t", "linux/arm64", "-o", outputFile, "tests/print_pass_smoke.go"},
		[]string{"renvo: unsupported target: linux/arm64", "linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "windows/amd64", "windows/386", "wasi/wasm32", "darwin/arm64"},
	)
	checkFailure(
		"missing target argument",
		[]string{"-t"},
		[]string{"renvo: missing argument for -t", "usage: renvo"},
	)
	checkFailure(
		"missing arena size",
		[]string{"-arena-size"},
		[]string{"renvo: missing argument for -arena-size", "usage: renvo"},
	)
	checkFailure(
		"invalid arena size",
		[]string{"-arena-size", "128", "-o", outputFile, "tests/print_pass_smoke.go"},
		[]string{"renvo: invalid arena size: 128"},
	)
}

func TestStage1CompilerCanEmitSmokeTargets(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			outDir := t.TempDir()
			if target.name == "linux/aarch64" || target.name == "linux/amd64" {
				var err error
				outDir, err = os.MkdirTemp("/tmp", "renvo-"+target.safeName()+"-stage1-")
				if err != nil {
					t.Fatalf("failed to create debug temp dir: %v", err)
				}
				t.Logf("preserving debug dir: %s", outDir)
			}
			stage0 := filepath.Join(outDir, "stage0")
			if err := compile(target.files, stage0); err != nil {
				t.Fatalf("stage0 compilation failed: %v", err)
			}

			stage1 := filepath.Join(outDir, "stage1")
			if err := runHostCompilerBinaryForTarget(t, target, stage0, stage1, target.files); err != nil {
				t.Fatalf("stage1 compilation failed: %v", err)
			}

			smoke := filepath.Join(outDir, "smoke")
			if err := runTargetCompilerBinary(t, target, stage1, smoke, []string{"tests/appmain_no_args.go"}); err != nil {
				t.Fatalf("stage1 smoke compilation failed: %v", err)
			}

			result, err := runTargetCommand(t, target, smoke)
			if err != nil {
				t.Fatalf("stage1 smoke execution failed: %v", err)
			}
			if result.exitCode != 0 || result.stdout != "PASS\n" || result.stderr != "" {
				t.Fatalf("stage1 smoke output mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func buildStage2Compiler(t *testing.T, target compilerTarget, outDir string) string {
	t.Helper()
	key := testArtifactKeyForFiles(t, []string{"stage2", target.name}, target.files)
	return cachedTestArtifact(t, "stage2", key, func(stage2 string) error {
		stage0 := filepath.Join(outDir, "stage0-"+target.safeName())
		if err := compile(target.files, stage0); err != nil {
			return fmt.Errorf("stage0 compilation failed: %w", err)
		}
		stage1 := filepath.Join(outDir, "stage1-"+target.safeName())
		if err := runHostCompilerBinaryForTarget(t, target, stage0, stage1, target.files); err != nil {
			return fmt.Errorf("stage1 compilation failed: %w", err)
		}
		if err := runTargetCompilerBinary(t, target, stage1, stage2, target.files); err != nil {
			return fmt.Errorf("stage2 compilation failed: %w", err)
		}
		return nil
	})
}

func expectedCommandResult(t *testing.T, path string) commandResult {
	t.Helper()
	expectedPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".expected"
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read checked-in expectation %s: %v", expectedPath, err)
	}
	if len(data) > 0 && data[0] == '{' {
		var expected struct {
			Stdout   string `json:"stdout"`
			Stderr   string `json:"stderr"`
			ExitCode int    `json:"exit_code"`
		}
		if err := json.Unmarshal(data, &expected); err != nil {
			t.Fatalf("decode checked-in expectation %s: %v", expectedPath, err)
		}
		return commandResult{stdout: expected.Stdout, stderr: expected.Stderr, exitCode: expected.ExitCode}
	}
	return commandResult{stdout: string(data)}
}

func compareCommandResult(t *testing.T, expected commandResult, actual commandResult) {
	t.Helper()
	if actual.stdout != expected.stdout || actual.stderr != expected.stderr || actual.exitCode != expected.exitCode {
		t.Fatalf("compiled output did not match the checked-in expectation\nstdout: got %q, want %q\nstderr: got %q, want %q\nexit code: got %d, want %d",
			actual.stdout, expected.stdout,
			actual.stderr, expected.stderr,
			actual.exitCode, expected.exitCode)
	}
}

func runBackendCorpusCases(t *testing.T, label string, inputFiles []string, run func(*testing.T, string)) {
	t.Helper()
	selected, err := testprogress.Filter(inputFiles, os.Getenv(testprogress.FilterEnv))
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) == 0 {
		t.Skipf("no backend corpus cases match %s=%q", testprogress.FilterEnv, os.Getenv(testprogress.FilterEnv))
	}
	workers := testProcessLimit()
	if workers > len(selected) {
		workers = len(selected)
	}
	tasks := make(chan string, len(selected)+workers)
	for _, path := range selected {
		tasks <- path
	}
	for worker := 0; worker < workers; worker++ {
		tasks <- ""
	}

	progress := testprogress.New(t, label, len(selected))
	t.Cleanup(progress.Close)
	for worker := 0; worker < workers; worker++ {
		worker := worker
		t.Run(fmt.Sprintf("worker-%02d", worker+1), func(t *testing.T) {
			t.Parallel()
			for {
				path := <-tasks
				if path == "" {
					return
				}
				func() {
					done := progress.Begin(path)
					defer done()
					defer func() {
						if t.Failed() {
							t.Logf("failing corpus case: %s", path)
						}
					}()
					run(t, path)
				}()
			}
		})
	}
}

// test that the compiler can compile and run a simple "hello, world!" program.
func TestCompileTests(t *testing.T) {
	targets := supportedCompilerTargets(t)

	// discover all files under tests/ that end with .go
	var inputFiles []string
	err := filepath.Walk("tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			inputFiles = append(inputFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to discover test files: %v", err)
	}

	for _, target := range targets {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)

			outDir := t.TempDir()
			stage2 := buildStage2Compiler(t, target, outDir)

			runBackendCorpusCases(t, target.name+" direct source", inputFiles, func(t *testing.T, path string) {
				expected := expectedCommandResult(t, path)
				outputFile := cachedTargetProgram(t, target, stage2, "source", []string{path})
				actual, err := runTargetCommand(t, target, outputFile)
				if err != nil {
					t.Fatalf("execute %s: %v", path, err)
				}
				compareCommandResult(t, expected, actual)
			})
		})
	}
}

func TestConfiguredArenaExhaustionCannotEscapeBSS(t *testing.T) {
	targets := supportedCompilerTargets(t)
	if len(targets) == 0 {
		t.Fatal("no native compiler target")
	}
	target := targets[0]
	skipIfTargetRunnerMissing(t, target)
	outDir := t.TempDir()
	stage2 := buildStage2Compiler(t, target, outDir)
	outputFile := filepath.Join(outDir, "arena-exhaustion")
	result, err := runTargetCommand(t, target, stage2,
		"-t", target.name,
		"-arena-size", "256",
		"-o", outputFile,
		"tests/arena_bounded_allocation.go")
	if err != nil {
		t.Fatalf("compiler execution failed: %v", err)
	}
	if result.exitCode != 0 {
		t.Fatalf("compilation failed with exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
	}
	actual, err := runTargetCommand(t, target, outputFile)
	if err != nil {
		t.Fatalf("bounded output execution failed: %v", err)
	}
	if actual.exitCode == 0 {
		t.Fatalf("arena exhaustion unexpectedly succeeded: stdout=%q stderr=%q", actual.stdout, actual.stderr)
	}
}

func TestRunTests(t *testing.T) {
	// discover all files under tests/ that end with .go
	var inputFiles []string
	err := filepath.Walk("tests", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			inputFiles = append(inputFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to discover test files: %v", err)
	}

	for _, path := range inputFiles {
		expected := expectedCommandResult(t, path)
		if expected.stdout+expected.stderr != "PASS\n" || expected.exitCode != 0 {
			t.Fatalf("invalid positive-test expectation for %s: exit=%d stdout=%q stderr=%q", path, expected.exitCode, expected.stdout, expected.stderr)
		}
	}
}
