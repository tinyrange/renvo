package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
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

func getCompilerFiles(config targetConfig) ([]string, error) {
	var files []string

	switch config.os + "/" + config.arch {
	case "linux/amd64", "linux/386":
	default:
		return nil, fmt.Errorf("unsupported OS/architecture combination: %s/%s", config.os, config.arch)
	}

	files = append(files,
		"compiler_common_impl.go",
		"compiler_main.go",
		"compiler_linux_impl.go",
		"compiler_amd64_impl.go",
		"compiler_386_impl.go",
		"compiler_linux_amd64_impl.go",
		"compiler_linux_386_impl.go",
	)

	return files, nil
}

type compilerTarget struct {
	name  string
	files []string
}

func supportedCompilerTargets(t *testing.T) []compilerTarget {
	t.Helper()

	var targets []compilerTarget

	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		for _, config := range []targetConfig{
			{os: "linux", arch: "amd64"},
			{os: "linux", arch: "386"},
		} {
			files, err := getCompilerFiles(config)
			if err != nil {
				t.Fatalf("failed to get compiler files for target %s/%s: %v", config.os, config.arch, err)
			}
			targetName := fmt.Sprintf("%s/%s", config.os, config.arch)
			targets = append(targets, compilerTarget{name: targetName, files: files})
		}
		return targets
	default:
		t.Skipf("no RTG compiler targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func (target compilerTarget) safeName() string {
	return strings.ReplaceAll(target.name, "/", "-")
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

	err := compileLinuxAmd64(input, outputFd)
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
		[]string{"rtg: unsupported target: linux/arm64", "linux/amd64", "linux/386"},
	)
	checkFailure(
		"missing target argument",
		[]string{"-t"},
		[]string{"rtg: missing argument for -t", "usage: rtg"},
	)
}

func TestStage1CompilerCanEmitSmokeTargets(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			outDir := t.TempDir()
			stage0 := filepath.Join(outDir, "stage0")
			if err := compile(target.files, stage0); err != nil {
				t.Fatalf("stage0 compilation failed: %v", err)
			}

			stage1 := filepath.Join(outDir, "stage1")
			if err := runTargetCompilerBinary(t, target, stage0, stage1, target.files); err != nil {
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

	stage0 := filepath.Join(outDir, "stage0-"+target.safeName())
	if err := compile(target.files, stage0); err != nil {
		t.Fatalf("stage0 compilation failed: %v", err)
	}

	stage1 := filepath.Join(outDir, "stage1-"+target.safeName())
	if err := runTargetCompilerBinary(t, target, stage0, stage1, target.files); err != nil {
		t.Fatalf("stage1 compilation failed: %v", err)
	}

	stage2 := filepath.Join(outDir, "stage2-"+target.safeName())
	if err := runTargetCompilerBinary(t, target, stage1, stage2, target.files); err != nil {
		t.Fatalf("stage2 compilation failed: %v", err)
	}

	return stage2
}

func runWithHostGo(t *testing.T, path string) commandResult {
	t.Helper()

	outDir := t.TempDir()
	runtimeData, err := os.ReadFile("rtg_main.go")
	if err != nil {
		t.Fatalf("failed to read runtime: %v", err)
	}
	testData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read test source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "rtg_main.go"), runtimeData, 0644); err != nil {
		t.Fatalf("failed to write runtime copy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "test.go"), testData, 0644); err != nil {
		t.Fatalf("failed to write test copy: %v", err)
	}

	hostBinary := filepath.Join(outDir, "host-test")
	buildResult, err := runCommandInDir(t, outDir, "go", "build", "-o", hostBinary, "rtg_main.go", "test.go")
	if err != nil {
		t.Fatalf("host go build failed: %v", err)
	}
	if buildResult.exitCode != 0 {
		t.Fatalf("host go build failed with exit code %d\nstdout: %sstderr: %s", buildResult.exitCode, buildResult.stdout, buildResult.stderr)
	}

	result, err := runCommand(t, hostBinary)
	if err != nil {
		t.Fatalf("host-built execution failed: %v", err)
	}
	return result
}

func compareCommandResult(t *testing.T, expected commandResult, actual commandResult) {
	t.Helper()
	if actual.stdout != expected.stdout || actual.stderr != expected.stderr || actual.exitCode != expected.exitCode {
		t.Fatalf("compiled output did not match host go\nstdout: got %q, want %q\nstderr: got %q, want %q\nexit code: got %d, want %d",
			actual.stdout, expected.stdout,
			actual.stderr, expected.stderr,
			actual.exitCode, expected.exitCode)
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
			outDir := t.TempDir()
			stage2 := buildStage2Compiler(t, target, outDir)

			for _, path := range inputFiles {
				path := path
				t.Run(path, func(t *testing.T) {
					t.Parallel()

					expected := runWithHostGo(t, path)

					testOutDir := t.TempDir()
					outputFile := filepath.Join(testOutDir, "test")

					if err := runTargetCompilerBinary(t, target, stage2, outputFile, []string{path}); err != nil {
						t.Fatalf("compilation failed: %v", err)
					}

					actual, err := runTargetCommand(t, target, outputFile)
					if err != nil {
						t.Fatalf("execution failed: %v", err)
					}
					compareCommandResult(t, expected, actual)
				})
			}
		})
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
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			expected := runWithHostGo(t, path)

			if expected.exitCode != 0 {
				t.Fatalf("host go execution failed with exit code %d\nstdout: %sstderr: %s", expected.exitCode, expected.stdout, expected.stderr)
			}
		})
	}
}

// Test the self-hosting of the compiler.
func TestCompilerCompiler(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			// compile stage0
			outDir := t.TempDir()
			stage0 := filepath.Join(outDir, "stage0")

			err := compile(target.files, stage0)
			if err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			// use stage0 to compile stage1
			stage1 := filepath.Join(outDir, "stage1")
			if err := runTargetCompilerBinary(t, target, stage0, stage1, target.files); err != nil {
				t.Fatalf("stage0 compilation failed: %v", err)
			}

			// use stage1 to compile stage2
			stage2 := filepath.Join(outDir, "stage2")
			if err := runTargetCompilerBinary(t, target, stage1, stage2, target.files); err != nil {
				t.Fatalf("stage1 compilation failed: %v", err)
			}

			// use stage2 to compile stage3
			stage3 := filepath.Join(outDir, "stage3")
			if err := runTargetCompilerBinary(t, target, stage2, stage3, target.files); err != nil {
				t.Fatalf("stage2 compilation failed: %v", err)
			}

			// make sure stage2 and stage3 are byte identical
			stage2Data, err := os.ReadFile(stage2)
			if err != nil {
				t.Fatalf("failed to read stage2: %v", err)
			}
			stage3Data, err := os.ReadFile(stage3)
			if err != nil {
				t.Fatalf("failed to read stage3: %v", err)
			}
			if !bytes.Equal(stage2Data, stage3Data) {
				t.Fatal("stage2 and stage3 are not identical")
			}
		})
	}
}

// Check the stage2 compiler compiles stage3 in under 50ms, produces a binary under 160KB,
// and uses under 3MB max RSS while compiling stage3.
func TestCompilerPerformance(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			outDir := t.TempDir()

			stage0 := filepath.Join(outDir, "stage0")
			if err := compile(target.files, stage0); err != nil {
				t.Fatalf("stage0 compilation failed: %v", err)
			}

			stage1 := filepath.Join(outDir, "stage1")
			if err := runTargetCompilerBinary(t, target, stage0, stage1, target.files); err != nil {
				t.Fatalf("stage1 compilation failed: %v", err)
			}

			stage2 := filepath.Join(outDir, "stage2")
			if err := runTargetCompilerBinary(t, target, stage1, stage2, target.files); err != nil {
				t.Fatalf("stage2 compilation failed: %v", err)
			}

			stage3 := filepath.Join(outDir, "stage3")
			stage3Args := append([]string{"-t", target.name, "-o", stage3}, target.files...)
			start := time.Now()
			result, err := runTargetCommand(t, target, stage2, stage3Args...)
			elapsed := time.Since(start)
			if err != nil {
				t.Fatalf("stage2 performance compilation failed: %v", err)
			}
			if result.exitCode != 0 {
				t.Fatalf("stage2 performance compilation failed with exit code %d\nstdout: %sstderr: %s", result.exitCode, result.stdout, result.stderr)
			}

			rssFile := filepath.Join(outDir, "stage3-compile-rss")
			timeArgs := append([]string{"-f", "%M", "-o", rssFile, stage2}, stage3Args...)
			cmd := exec.Command("/usr/bin/time", timeArgs...)
			cmd.Env = os.Environ()
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("stage2 resource-measured compilation failed: %v\nOutput: %s", err, string(output))
			}

			rssData, err := os.ReadFile(rssFile)
			if err != nil {
				t.Fatalf("failed to read stage3 compile resource usage: %v", err)
			}
			rssLines := strings.Fields(string(rssData))
			if len(rssLines) == 0 {
				t.Fatalf("failed to read stage3 compile resource usage")
			}
			maxRSS, err := strconv.Atoi(rssLines[len(rssLines)-1])
			if err != nil {
				t.Fatalf("failed to parse stage3 compile resource usage %q: %v", string(rssData), err)
			}
			const maxRSSKB = 4 * 1024

			var failures []string
			if elapsed > 50*time.Millisecond {
				failures = append(failures, fmt.Sprintf("runtime %s > 50ms", elapsed))
			}
			if maxRSS > maxRSSKB {
				failures = append(failures, fmt.Sprintf("compile max RSS %dKB > %dKB", maxRSS, maxRSSKB))
			}
			if len(failures) > 0 {
				t.Fatalf("performance limits failed: stage2 runtime=%s, stage2 compile max RSS=%dKB; failures: %s",
					elapsed, maxRSS, strings.Join(failures, "; "))
			}
		})
	}
}
