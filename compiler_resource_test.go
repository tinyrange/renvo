package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const compilerResourceGatesEnv = "RTG_COMPILER_RESOURCE_GATES"

func TestCompilerResourceGates(t *testing.T) {
	if os.Getenv(compilerResourceGatesEnv) != "1" {
		t.Skipf("set %s=1 to run compiler resource gates", compilerResourceGatesEnv)
	}
	for _, target := range performanceCompilerTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			outDir := t.TempDir()
			files := getPerformanceCompilerFiles(t, target, outDir)

			compilerPath := filepath.Join(outDir, "compiler")
			oldStrip := rtgCompilerStripSymbols
			rtgCompilerStripSymbols = true
			if err := compile(files, compilerPath); err != nil {
				rtgCompilerStripSymbols = oldStrip
				t.Fatalf("compiler build failed: %v", err)
			}
			rtgCompilerStripSymbols = oldStrip

			compilerInfo, err := os.Stat(compilerPath)
			if err != nil {
				t.Fatalf("failed to stat compiler binary: %v", err)
			}
			const maxRSSKB = 16 * 1024
			const maxBinarySize = 256 * 1024
			bestRSS := 1 << 30
			for attempt := 0; attempt < 3; attempt++ {
				outputPath := filepath.Join(outDir, fmt.Sprintf("compiler-output-%d", attempt))
				compileArgs := append([]string{"-s", "-o", outputPath}, files...)

				rssFile := filepath.Join(outDir, fmt.Sprintf("compile-rss-%d", attempt))
				timeArgs := append([]string{"-f", "%M", "-o", rssFile, compilerPath}, compileArgs...)
				cmd := exec.Command("/usr/bin/time", timeArgs...)
				cmd.Env = []string{}
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("resource-measured compilation failed: %v\nOutput: %s", err, string(output))
				}

				rssData, err := os.ReadFile(rssFile)
				if err != nil {
					t.Fatalf("failed to read compile resource usage: %v", err)
				}
				rssFields := strings.Fields(string(rssData))
				if len(rssFields) == 0 {
					t.Fatalf("failed to read compile resource usage")
				}
				maxRSS, err := strconv.Atoi(rssFields[len(rssFields)-1])
				if err != nil {
					t.Fatalf("failed to parse compile resource usage %q: %v", string(rssData), err)
				}
				if maxRSS < bestRSS {
					bestRSS = maxRSS
				}
				if maxRSS <= maxRSSKB && compilerInfo.Size() <= maxBinarySize {
					return
				}
			}

			var failures []string
			if bestRSS > maxRSSKB {
				failures = append(failures, fmt.Sprintf("compile max RSS %dKB > %dKB", bestRSS, maxRSSKB))
			}
			if compilerInfo.Size() > maxBinarySize {
				failures = append(failures, fmt.Sprintf("compiler binary size %dB > %dB", compilerInfo.Size(), maxBinarySize))
			}
			if len(failures) > 0 {
				t.Fatalf("resource limits failed: best compile max RSS=%dKB, compiler binary size=%dB; failures: %s",
					bestRSS, compilerInfo.Size(), strings.Join(failures, "; "))
			}
		})
	}
}
