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
	return []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go", "rtg_main.go"}
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

	return nil
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

	for _, path := range inputFiles {
		t.Run(path, func(t *testing.T) {
			// compile the test file
			resetRuntime()

			outDir := t.TempDir()
			outputFile := filepath.Join(outDir, "hello")

			err := compile([]string{path}, outputFile)
			if err != nil {
				t.Fatalf("compilation failed: %v", err)
			}

			// Run the compiled binary and check its output
			cmd := exec.Command(outputFile)
			cmd.Env = os.Environ()
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("execution failed: %v\nOutput: %s", err, string(output))
			}

			expectedOutput := "PASS\n"
			if string(output) != expectedOutput {
				t.Fatalf("unexpected output: got %q, want %q", string(output), expectedOutput)
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
	if elapsed > 25*time.Millisecond {
		t.Fatalf("stage2 compiler took %s, want <= 25ms", elapsed)
	}

	info, err := os.Stat(stage3)
	if err != nil {
		t.Fatalf("failed to stat stage3: %v", err)
	}
	const maxBinarySize = 126 * 1024
	if info.Size() > maxBinarySize {
		t.Fatalf("stage3 binary is %d bytes, want <= %d", info.Size(), maxBinarySize)
	}

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
	if maxRSS > maxRSSKB {
		t.Fatalf("stage3 max RSS is %dKB, want <= %dKB", maxRSS, maxRSSKB)
	}
}
