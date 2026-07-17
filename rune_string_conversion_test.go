package main

import (
	"path/filepath"
	"testing"
)

func TestRuneConversionArenaExhaustion(t *testing.T) {
	targets := supportedCompilerTargets(t)
	if len(targets) == 0 {
		t.Fatal("no native compiler target")
	}
	target := targets[0]
	skipIfTargetRunnerMissing(t, target)
	outDir := t.TempDir()
	stage2 := buildStage2Compiler(t, target, outDir)
	outputFile := filepath.Join(outDir, "rune-conversion-oom")
	result, err := runTargetCommand(t, target, stage2,
		"-t", target.name,
		"-arena-size", "256",
		"-o", outputFile,
		"tests/rune_string_conversion_oom.go")
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
		t.Fatalf("rune conversion arena exhaustion unexpectedly succeeded: stdout=%q stderr=%q", actual.stdout, actual.stderr)
	}
}
