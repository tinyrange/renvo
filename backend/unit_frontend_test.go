package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"renvo.dev/backend/unit"
)

const unitFrontendCrossArchEnv = "RENVO_UNIT_FRONTEND_CROSS_ARCH_TESTS"

func TestUnitFrontendCompileTests(t *testing.T) {
	targets := unitFrontendTargets(t)

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

			for _, path := range inputFiles {
				path := path
				t.Run(path, func(t *testing.T) {
					t.Parallel()

					expected := runWithHostGo(t, path)

					testOutDir := t.TempDir()
					unitPath := filepath.Join(testOutDir, "test.unit")
					program, err := unit.ConvertFiles([]string{path})
					if err != nil {
						t.Fatalf("unit conversion failed: %v", err)
					}
					if err := unit.WriteFile(unitPath, program); err != nil {
						t.Fatalf("unit write failed: %v", err)
					}

					outputFile := cachedTargetProgram(t, target, stage2, "unit", []string{unitPath})

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

func TestUnitFrontendCompilerCompiler(t *testing.T) {
	for _, target := range unitFrontendTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			skipIfTargetRunnerMissing(t, target)

			outDir := t.TempDir()
			stage2 := buildStage2Compiler(t, target, outDir)

			program, err := unit.ConvertFiles(target.files)
			if err != nil {
				t.Fatalf("compiler unit conversion failed: %v", err)
			}
			unitPath := filepath.Join(outDir, "compiler.unit")
			if err := unit.WriteFile(unitPath, program); err != nil {
				t.Fatalf("compiler unit write failed: %v", err)
			}

			stage3 := cachedStage3Compiler(t, target, stage2, unitPath)

			smoke := filepath.Join(outDir, "smoke")
			if err := runTargetCompilerBinary(t, target, stage3, smoke, []string{"tests/appmain_no_args.go"}); err != nil {
				t.Fatalf("stage3 smoke compilation failed: %v", err)
			}

			result, err := runTargetCommand(t, target, smoke)
			if err != nil {
				t.Fatalf("stage3 smoke execution failed: %v", err)
			}
			if result.exitCode != 0 || result.stdout != "PASS\n" || result.stderr != "" {
				t.Fatalf("stage3 smoke output mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
			}
		})
	}
}

func unitFrontendTargets(t *testing.T) []compilerTarget {
	t.Helper()
	if os.Getenv(unitFrontendCrossArchEnv) == "1" {
		return supportedCompilerTargets(t)
	}
	var config targetConfig
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		config = targetConfig{os: "linux", arch: "amd64"}
	case "linux/arm64":
		config = targetConfig{os: "linux", arch: "aarch64"}
	case "darwin/arm64":
		config = targetConfig{os: "darwin", arch: "arm64"}
	default:
		t.Skipf("no native RENVO unit frontend target supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
	files, err := getCompilerFiles(config)
	if err != nil {
		t.Fatalf("failed to get compiler files for target %s/%s: %v", config.os, config.arch, err)
	}
	return []compilerTarget{{name: config.os + "/" + config.arch, files: files}}
}
