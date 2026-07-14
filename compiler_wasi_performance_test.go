package main

import (
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

func TestCompilerPerformanceWASI(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("WASI compiler performance gate requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if _, err := exec.LookPath("/usr/bin/time"); err != nil {
		t.Skip("WASI compiler performance gate requires /usr/bin/time")
	}

	outDir := t.TempDir()
	files := getPerformanceCompilerFiles(t, compilerTarget{name: "wasi/wasm32"}, outDir)
	compilerPath := filepath.Join(outDir, "compiler")
	oldStrip := rtgCompilerStripSymbols
	rtgCompilerStripSymbols = true
	err := compile(files, compilerPath)
	rtgCompilerStripSymbols = oldStrip
	if err != nil {
		t.Fatalf("WASI compiler build failed: %v", err)
	}
	compilerInfo, err := os.Stat(compilerPath)
	if err != nil {
		t.Fatalf("stat WASI compiler: %v", err)
	}

	const maxElapsed = 50 * time.Millisecond
	const maxRSSKB = 16 * 1024
	const maxBinarySize = 256 * 1024
	bestElapsed := 24 * time.Hour
	bestRSS := 1 << 30
	for attempt := 0; attempt < 3; attempt++ {
		outputPath := filepath.Join(outDir, fmt.Sprintf("compiler-output-%d", attempt))
		rssPath := filepath.Join(outDir, fmt.Sprintf("compile-rss-%d", attempt))
		compileArgs := append([]string{"-s", "-o", outputPath}, files...)
		timeArgs := append([]string{"-f", "%e %M", "-o", rssPath, compilerPath}, compileArgs...)
		cmd := exec.Command("/usr/bin/time", timeArgs...)
		cmd.Env = []string{}
		output, runErr := cmd.CombinedOutput()
		if runErr != nil {
			t.Fatalf("resource-measured WASI compilation failed: %v\nOutput: %s", runErr, string(output))
		}
		rssData, readErr := os.ReadFile(rssPath)
		if readErr != nil {
			t.Fatalf("read WASI compiler resource usage: %v", readErr)
		}
		fields := strings.Fields(string(rssData))
		if len(fields) < 2 {
			t.Fatalf("invalid WASI compiler resource usage %q", string(rssData))
		}
		seconds, parseErr := strconv.ParseFloat(fields[0], 64)
		if parseErr != nil {
			t.Fatalf("parse WASI compiler elapsed time %q: %v", string(rssData), parseErr)
		}
		rss, parseErr := strconv.Atoi(fields[len(fields)-1])
		if parseErr != nil {
			t.Fatalf("parse WASI compiler RSS %q: %v", string(rssData), parseErr)
		}
		elapsed := time.Duration(seconds * float64(time.Second))
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
