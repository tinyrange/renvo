package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompilerPerformanceWASI(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("WASI compiler performance gate requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	outDir := t.TempDir()
	files := getPerformanceCompilerFiles(t, compilerTarget{name: "wasi/wasm32"}, outDir)
	compilerPath := filepath.Join(outDir, "compiler")
	oldStrip := renvoCompilerStripSymbols
	renvoCompilerStripSymbols = true
	err := compile(files, compilerPath)
	renvoCompilerStripSymbols = oldStrip
	if err != nil {
		t.Fatalf("WASI compiler build failed: %v", err)
	}
	compilerInfo, err := os.Stat(compilerPath)
	if err != nil {
		t.Fatalf("stat WASI compiler: %v", err)
	}

	const maxElapsed = 100 * time.Millisecond
	const maxRSSKB = 16 * 1024
	const maxBinarySize = 288 * 1024
	bestElapsed := 24 * time.Hour
	bestRSS := 1 << 30
	for attempt := 0; attempt < 3; attempt++ {
		outputPath := filepath.Join(outDir, fmt.Sprintf("compiler-output-%d", attempt))
		compileArgs := append([]string{"-s", "-o", outputPath}, files...)
		usage, runErr := runMeasuredProcess(t, "", []string{}, compilerPath, compileArgs...)
		if runErr != nil {
			t.Fatalf("resource-measured WASI compilation failed: %v\nOutput: %s", runErr, string(usage.output))
		}
		elapsed := usage.elapsed
		rss := usage.maxRSSKB
		if elapsed < bestElapsed {
			bestElapsed = elapsed
		}
		if rss < bestRSS {
			bestRSS = rss
		}
		if elapsed <= maxElapsed && rss <= maxRSSKB && compilerInfo.Size() <= maxBinarySize {
			return
		}
	}

	var failures []string
	if bestElapsed > maxElapsed {
		failures = append(failures, fmt.Sprintf("runtime %s > %s", bestElapsed, maxElapsed))
	}
	if bestRSS > maxRSSKB {
		failures = append(failures, fmt.Sprintf("max RSS %dKB > %dKB", bestRSS, maxRSSKB))
	}
	if compilerInfo.Size() > maxBinarySize {
		failures = append(failures, fmt.Sprintf("compiler binary size %dB > %dB", compilerInfo.Size(), maxBinarySize))
	}
	t.Fatalf("WASI performance limits failed: best runtime=%s, best max RSS=%dKB, compiler size=%dB; failures: %s", bestElapsed, bestRSS, compilerInfo.Size(), strings.Join(failures, "; "))
}
