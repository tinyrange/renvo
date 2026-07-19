package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var uncaughtRuntimeFaultKindsProgram = []byte(`package main

type runtimeFaultKindRecord struct {
	value int
}

func runtimeFaultKindZero() int {
	return 0
}

func appMain(args []string) int {
	print("before fault\n")
	mode := args[1]
	values := []int{1}
	array := [1]int{1}
	text := "x"
	high := 2
	negative := runtimeFaultKindZero() - 1
	switch mode {
	case "slice-high":
		return values[high]
	case "array-high":
		return array[high]
	case "string-high":
		return int(text[high])
	case "string-negative":
		return int(text[negative])
	case "nil-dereference":
		var pointer *runtimeFaultKindRecord
		return pointer.value
	case "division-zero":
		return 10 / runtimeFaultKindZero()
	}
	return 0
}
`)

var uncaughtTypeAssertionProgram = []byte(`package main

func appMain() int {
	print("before fault\n")
	var value interface{} = 1
	result := value.(string)
	return len(result)
}
`)

var uncaughtRuntimeFaultKinds = []string{
	"slice-high",
	"array-high",
	"string-high",
	"string-negative",
	"nil-dereference",
	"division-zero",
}

func TestUncaughtRuntimeFaultKindsTermination(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			image, ok := RenvoCompileSourceToBytes(uncaughtRuntimeFaultKindsProgram, target.name)
			if !ok {
				t.Fatal("failed to compile runtime fault kinds program")
			}
			output := filepath.Join(t.TempDir(), "runtime-fault-kinds")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			for _, mode := range uncaughtRuntimeFaultKinds {
				mode := mode
				t.Run(mode, func(t *testing.T) {
					result, err := runTargetCommand(t, target, output, mode)
					if err != nil {
						t.Fatalf("execution failed: %v", err)
					}
					if result.exitCode != 2 || result.stdout != "before fault\n" || result.stderr != "panic\n" {
						t.Fatalf("uncaught runtime fault mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
					}
				})
			}
		})
	}
}

func TestUncaughtTypeAssertionTermination(t *testing.T) {
	for _, target := range supportedCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)
			image, ok := RenvoCompileSourceToBytes(uncaughtTypeAssertionProgram, target.name)
			if !ok {
				t.Fatal("failed to compile uncaught type assertion program")
			}
			output := filepath.Join(t.TempDir(), "uncaught-type-assertion")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runTargetCommand(t, target, output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "before fault\n" || result.stderr != "panic: interface conversion failed\n" {
				t.Fatalf("uncaught type assertion mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func TestWindowsUncaughtRuntimeFaultKindsTermination(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows execution requires a native Windows host")
	}
	for _, target := range []string{"windows/amd64", "windows/386", "windows/arm64"} {
		target := target
		t.Run(target, func(t *testing.T) {
			if target == "windows/arm64" && runtime.GOARCH != "arm64" {
				t.Skip("Windows ARM64 execution requires a native ARM64 host")
			}
			image, ok := RenvoCompileSourceToBytes(uncaughtRuntimeFaultKindsProgram, target)
			if !ok {
				t.Fatal("failed to compile runtime fault kinds program")
			}
			output := filepath.Join(t.TempDir(), "runtime-fault-kinds.exe")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			for _, mode := range uncaughtRuntimeFaultKinds {
				mode := mode
				t.Run(mode, func(t *testing.T) {
					result, err := runWindowsCommand(t, t.TempDir(), output, mode)
					if err != nil {
						t.Fatalf("execution failed: %v", err)
					}
					if result.exitCode != 2 || result.stdout != "before fault\n" || result.stderr != "panic\n" {
						t.Fatalf("uncaught runtime fault mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
					}
				})
			}
		})
	}
}

func TestWindowsUncaughtTypeAssertionTermination(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows execution requires a native Windows host")
	}
	for _, target := range []string{"windows/amd64", "windows/386", "windows/arm64"} {
		target := target
		t.Run(target, func(t *testing.T) {
			if target == "windows/arm64" && runtime.GOARCH != "arm64" {
				t.Skip("Windows ARM64 execution requires a native ARM64 host")
			}
			image, ok := RenvoCompileSourceToBytes(uncaughtTypeAssertionProgram, target)
			if !ok {
				t.Fatal("failed to compile uncaught type assertion program")
			}
			output := filepath.Join(t.TempDir(), "uncaught-type-assertion.exe")
			if err := os.WriteFile(output, image, 0755); err != nil {
				t.Fatal(err)
			}
			result, err := runWindowsCommand(t, t.TempDir(), output)
			if err != nil {
				t.Fatalf("execution failed: %v", err)
			}
			if result.exitCode != 2 || result.stdout != "before fault\n" || result.stderr != "panic: interface conversion failed\n" {
				t.Fatalf("uncaught type assertion mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}
