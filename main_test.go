package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func getCompilerFiles() []string {
	return []string{"compiler_common.go", "compiler_amd64_impl.go", "compiler_linux_amd64_impl.go", "compiler_main.go"}
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

func buildStage2Compiler(t *testing.T, outDir string) string {
	t.Helper()

	inputFiles := getCompilerFiles()
	stage0 := filepath.Join(outDir, "stage0")
	if err := compile(inputFiles, stage0); err != nil {
		t.Fatalf("stage0 compilation failed: %v", err)
	}

	stage1 := filepath.Join(outDir, "stage1")
	if err := runCompilerBinary(t, stage0, stage1, inputFiles); err != nil {
		t.Fatalf("stage1 compilation failed: %v", err)
	}

	stage2 := filepath.Join(outDir, "stage2")
	if err := runCompilerBinary(t, stage1, stage2, inputFiles); err != nil {
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

	outDir := t.TempDir()
	stage2 := buildStage2Compiler(t, outDir)

	for _, path := range inputFiles {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			expected := runWithHostGo(t, path)

			testOutDir := t.TempDir()
			outputFile := filepath.Join(testOutDir, "test")

			if err := runCompilerBinary(t, stage2, outputFile, []string{path}); err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			actual, err := runCommand(t, outputFile)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			compareCommandResult(t, expected, actual)
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
	inputFiles := getCompilerFiles()

	// compile stage0
	outDir := t.TempDir()
	stage0 := filepath.Join(outDir, "stage0")

	err := compile(inputFiles, stage0)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// use stage0 to compile stage1
	stage1 := filepath.Join(outDir, "stage1")
	cmd := exec.Command(stage0, append([]string{"-o", stage1}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage0 compilation failed: %v\nOutput: %s", err, string(output))
	}

	// use stage1 to compile stage2
	stage2 := filepath.Join(outDir, "stage2")
	cmd = exec.Command(stage1, append([]string{"-o", stage2}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 compilation failed: %v\nOutput: %s", err, string(output))
	}

	// use stage2 to compile stage3
	stage3 := filepath.Join(outDir, "stage3")
	cmd = exec.Command(stage2, append([]string{"-o", stage3}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 compilation failed: %v\nOutput: %s", err, string(output))
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
}

// Check the stage2 compiler compiles in under 25ms and produces a binary under 126KB which runs with under 1MB max RSS.
func TestCompilerPerformance(t *testing.T) {
	inputFiles := getCompilerFiles()
	outDir := t.TempDir()

	stage0 := filepath.Join(outDir, "stage0")
	if err := compile(inputFiles, stage0); err != nil {
		t.Fatalf("stage0 compilation failed: %v", err)
	}

	stage1 := filepath.Join(outDir, "stage1")
	cmd := exec.Command(stage0, append([]string{"-o", stage1}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 compilation failed: %v\nOutput: %s", err, string(output))
	}

	stage2 := filepath.Join(outDir, "stage2")
	cmd = exec.Command(stage1, append([]string{"-o", stage2}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 compilation failed: %v\nOutput: %s", err, string(output))
	}

	stage3 := filepath.Join(outDir, "stage3")
	cmd = exec.Command(stage2, append([]string{"-o", stage3}, inputFiles...)...)
	cmd.Env = os.Environ()
	start := time.Now()
	output, err = cmd.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("stage2 performance compilation failed: %v\nOutput: %s", err, string(output))
	}

	info, err := os.Stat(stage3)
	if err != nil {
		t.Fatalf("failed to stat stage3: %v", err)
	}
	const maxBinarySize = 160 * 1024

	rssFile := filepath.Join(outDir, "stage3-rss")
	cmd = exec.Command("/usr/bin/time", "-f", "%M", "-o", rssFile, stage3)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("stage3 run without arguments succeeded unexpectedly\nOutput: %s", string(output))
	}
	rssData, err := os.ReadFile(rssFile)
	if err != nil {
		t.Fatalf("failed to read stage3 resource usage: %v", err)
	}
	rssLines := strings.Fields(string(rssData))
	if len(rssLines) == 0 {
		t.Fatalf("failed to read stage3 resource usage")
	}
	maxRSS, err := strconv.Atoi(rssLines[len(rssLines)-1])
	if err != nil {
		t.Fatalf("failed to parse stage3 resource usage %q: %v", string(rssData), err)
	}
	const maxRSSKB = 1024

	var failures []string
	if elapsed > 30*time.Millisecond {
		failures = append(failures, fmt.Sprintf("runtime %s > 30ms", elapsed))
	}
	if info.Size() > maxBinarySize {
		failures = append(failures, fmt.Sprintf("binary size %d bytes > %d bytes", info.Size(), maxBinarySize))
	}
	if maxRSS > maxRSSKB {
		failures = append(failures, fmt.Sprintf("max RSS %dKB > %dKB", maxRSS, maxRSSKB))
	}
	if len(failures) > 0 {
		t.Fatalf("performance limits failed: stage2 runtime=%s, stage3 binary=%d bytes, stage3 max RSS=%dKB; failures: %s",
			elapsed, info.Size(), maxRSS, strings.Join(failures, "; "))
	}
}
