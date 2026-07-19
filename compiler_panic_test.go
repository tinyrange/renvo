package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var uncaughtPanicProgram = []byte(`package main

func appMain(args []string) int {
	print("before panic\n")
	var value interface{} = "boom"
	panic(value)
}
`)

var uncaughtRuntimeFaultProgram = []byte(`package main

func appMain(args []string) int {
	print("before fault\n")
	values := []int{1}
	index := 2
	return values[index]
}
`)

func TestUncaughtPanicTermination(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			image, ok := RtgCompileSourceToBytes(uncaughtPanicProgram, target.name)
			if !ok {
				t.Fatal("failed to compile uncaught panic program")
			}
			output := filepath.Join(t.TempDir(), "uncaught-panic")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runTargetCommand(t, target, output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "before panic\n" || result.stderr != "panic: boom\n" {
				t.Fatalf("uncaught panic mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func TestUncaughtRuntimeFaultTermination(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			image, ok := RtgCompileSourceToBytes(uncaughtRuntimeFaultProgram, target.name)
			if !ok {
				t.Fatal("failed to compile uncaught runtime fault program")
			}
			output := filepath.Join(t.TempDir(), "uncaught-runtime-fault")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runTargetCommand(t, target, output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "before fault\n" || result.stderr != "panic\n" {
				t.Fatalf("uncaught runtime fault mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func TestWindowsUncaughtPanicTermination(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows execution requires a native Windows host")
	}
	targets := []string{"windows/amd64", "windows/386", "windows/arm64"}
	for _, target := range targets {
		target := target
		t.Run(target, func(t *testing.T) {
			if target == "windows/arm64" && runtime.GOARCH != "arm64" {
				t.Skip("Windows ARM64 execution requires a native ARM64 host")
			}
			image, ok := RtgCompileSourceToBytes(uncaughtPanicProgram, target)
			if !ok {
				t.Fatal("failed to compile uncaught panic program")
			}
			output := filepath.Join(t.TempDir(), "uncaught-panic.exe")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runWindowsCommand(t, t.TempDir(), output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "before panic\n" || result.stderr != "panic: boom\n" {
				t.Fatalf("uncaught panic mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}
