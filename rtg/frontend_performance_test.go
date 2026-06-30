package rtg_test

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

const selfHostedFrontendMaxRuntime = time.Second
const selfHostedFrontendMaxRSSKB = 16 * 1024
const selfHostedFrontendMaxCompilerSize = 1024 * 1024

func TestSelfHostedFrontendPerformance(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend performance test", selfHostTestsEnv)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("self-hosted frontend performance requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if _, err := exec.LookPath("/usr/bin/time"); err != nil {
		t.Skipf("/usr/bin/time is required for frontend performance measurement")
	}

	tmp := t.TempDir()
	root := ".."
	stage1 := filepath.Join(tmp, "stage1")
	cmd := exec.Command("go", "run", "./rtg/cmd/rtg", "-t", "linux/amd64", "-s", "-o", stage1, "./rtg/cmd/rtg")
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 frontend build failed: %v\n%s", err, string(data))
	}

	stage1UnitDir := filepath.Join(tmp, "stage1-units")
	if err := os.MkdirAll(stage1UnitDir, 0755); err != nil {
		t.Fatalf("MkdirAll unit dir failed: %v", err)
	}
	cmd = exec.Command(stage1, "-emit-unit", "-t", "linux/amd64", "-o", stage1UnitDir, "./rtg/cmd/rtg")
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 emit-unit failed: %v\n%s", err, string(data))
	}

	stage2 := filepath.Join(tmp, "stage2")
	cmd = exec.Command(stage1, "-link", "-t", "linux/amd64", "-s", "-o", stage2, stage1UnitDir)
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 link stage2 failed: %v\n%s", err, string(data))
	}

	stage3 := ""
	if ok := t.Run("stage2-selfhost-build", func(t *testing.T) {
		best := measureSelfHostedFrontendBuild(t, tmp, root, stage2, "stage3", "./rtg/cmd/rtg")
		stage3 = best.output
		checkSelfHostedFrontendPerf(t, best, mustStatSelfHostedFrontend(t, stage3))
	}); !ok {
		return
	}

	t.Run("stage3-selfhost-build", func(t *testing.T) {
		best := measureSelfHostedFrontendBuild(t, tmp, root, stage3, "stage4", "./rtg/cmd/rtg")
		checkSelfHostedFrontendPerf(t, best, mustStatSelfHostedFrontend(t, best.output))
	})
}

type selfHostedFrontendPerf struct {
	elapsed time.Duration
	maxRSS  int
	output  string
}

func measureSelfHostedFrontend(t *testing.T, tmp string, dir string, self string, argsForAttempt func(int) []string) selfHostedFrontendPerf {
	t.Helper()

	best := selfHostedFrontendPerf{elapsed: 24 * time.Hour, maxRSS: 1 << 30}
	for attempt := 0; attempt < 3; attempt++ {
		args := argsForAttempt(attempt)
		rssFile := filepath.Join(tmp, fmt.Sprintf("%s-rss-%d", strings.ReplaceAll(t.Name(), "/", "-"), attempt))
		timeArgs := append([]string{"-f", "%e %M", "-o", rssFile, self}, args...)
		cmd := exec.Command("/usr/bin/time", timeArgs...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("resource-measured frontend command failed: %v\nOutput: %s", err, string(output))
		}
		elapsed, maxRSS := readSelfHostedFrontendResourceUsage(t, rssFile)
		perf := selfHostedFrontendPerf{
			elapsed: elapsed,
			maxRSS:  maxRSS,
			output:  selfHostedFrontendOutputPath(args),
		}
		if selfHostedFrontendPerfIsBetter(perf, best) {
			best = perf
		}
	}
	return best
}

func measureSelfHostedFrontendBuild(t *testing.T, tmp string, dir string, self string, outputPrefix string, input string) selfHostedFrontendPerf {
	t.Helper()

	best := selfHostedFrontendPerf{elapsed: 24 * time.Hour, maxRSS: 1 << 30}
	for attempt := 0; attempt < 3; attempt++ {
		unitDir := filepath.Join(tmp, fmt.Sprintf("%s-units-%d", outputPrefix, attempt))
		if err := os.MkdirAll(unitDir, 0755); err != nil {
			t.Fatalf("MkdirAll unit dir failed: %v", err)
		}
		output := filepath.Join(tmp, fmt.Sprintf("%s-%d", outputPrefix, attempt))
		emit := measureSelfHostedFrontendCommand(t, tmp, dir, self, []string{"-emit-unit", "-t", "linux/amd64", "-o", unitDir, input}, "emit", attempt)
		link := measureSelfHostedFrontendCommand(t, tmp, dir, self, []string{"-link", "-t", "linux/amd64", "-s", "-o", output, unitDir}, "link", attempt)
		perf := selfHostedFrontendPerf{
			elapsed: emit.elapsed + link.elapsed,
			maxRSS:  maxInt(emit.maxRSS, link.maxRSS),
			output:  output,
		}
		if selfHostedFrontendPerfIsBetter(perf, best) {
			best = perf
		}
	}
	return best
}

func measureSelfHostedFrontendCommand(t *testing.T, tmp string, dir string, self string, args []string, phase string, attempt int) selfHostedFrontendPerf {
	t.Helper()

	rssFile := filepath.Join(tmp, fmt.Sprintf("%s-%s-rss-%d", strings.ReplaceAll(t.Name(), "/", "-"), phase, attempt))
	timeArgs := append([]string{"-f", "%e %M", "-o", rssFile, self}, args...)
	cmd := exec.Command("/usr/bin/time", timeArgs...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("resource-measured frontend %s command failed: %v\nOutput: %s", phase, err, string(output))
	}
	elapsed, maxRSS := readSelfHostedFrontendResourceUsage(t, rssFile)
	return selfHostedFrontendPerf{elapsed: elapsed, maxRSS: maxRSS, output: selfHostedFrontendOutputPath(args)}
}

func selfHostedFrontendOutputPath(args []string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-o" {
			return args[i+1]
		}
	}
	return ""
}

func selfHostedFrontendPerfIsBetter(candidate selfHostedFrontendPerf, best selfHostedFrontendPerf) bool {
	candidatePasses := candidate.elapsed <= selfHostedFrontendMaxRuntime && candidate.maxRSS <= selfHostedFrontendMaxRSSKB
	bestPasses := best.elapsed <= selfHostedFrontendMaxRuntime && best.maxRSS <= selfHostedFrontendMaxRSSKB
	if candidatePasses != bestPasses {
		return candidatePasses
	}
	if candidate.elapsed != best.elapsed {
		return candidate.elapsed < best.elapsed
	}
	return candidate.maxRSS < best.maxRSS
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func readSelfHostedFrontendResourceUsage(t *testing.T, path string) (time.Duration, int) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read frontend resource usage: %v", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		t.Fatalf("failed to read frontend resource usage from %q", string(data))
	}
	elapsedSeconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		t.Fatalf("failed to parse frontend elapsed time %q: %v", string(data), err)
	}
	maxRSS, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil {
		t.Fatalf("failed to parse frontend max RSS %q: %v", string(data), err)
	}
	return time.Duration(elapsedSeconds * float64(time.Second)), maxRSS
}

func checkSelfHostedFrontendPerf(t *testing.T, perf selfHostedFrontendPerf, compilerSize int64) {
	t.Helper()

	var failures []string
	if perf.elapsed > selfHostedFrontendMaxRuntime {
		failures = append(failures, fmt.Sprintf("runtime %s > %s", perf.elapsed, selfHostedFrontendMaxRuntime))
	}
	if perf.maxRSS > selfHostedFrontendMaxRSSKB {
		failures = append(failures, fmt.Sprintf("max RSS %dKB > %dKB", perf.maxRSS, selfHostedFrontendMaxRSSKB))
	}
	appendSelfHostedFrontendBinarySizeFailure(&failures, compilerSize)
	if len(failures) > 0 {
		t.Fatalf("selfhost frontend performance limits failed: runtime=%s, max RSS=%dKB, compiler binary size=%dB; failures: %s",
			perf.elapsed, perf.maxRSS, compilerSize, strings.Join(failures, "; "))
	}
}

func appendSelfHostedFrontendBinarySizeFailure(failures *[]string, compilerSize int64) {
	if compilerSize > selfHostedFrontendMaxCompilerSize {
		*failures = append(*failures, fmt.Sprintf("compiler binary size %dB > %dB", compilerSize, selfHostedFrontendMaxCompilerSize))
	}
}

func mustStatSelfHostedFrontend(t *testing.T, path string) int64 {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat self-hosted frontend %s: %v", path, err)
	}
	return info.Size()
}
