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
	self := filepath.Join(tmp, "rtg-self")
	cmd := exec.Command("go", "run", "./cmd/rtg", "-t", "linux/amd64", "-o", self, "./cmd/rtg")
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self frontend build failed: %v\n%s", err, string(data))
	}

	info, err := os.Stat(self)
	if err != nil {
		t.Fatalf("failed to stat self-hosted frontend: %v", err)
	}
	const maxCompilerSize = 1024 * 1024
	if info.Size() > maxCompilerSize {
		t.Fatalf("self-hosted frontend size %dB > %dB", info.Size(), maxCompilerSize)
	}

	t.Run("build", func(t *testing.T) {
		best := measureSelfHostedFrontend(t, tmp, self, func(attempt int) []string {
			return []string{"-t", "linux/amd64", "-o", filepath.Join(tmp, fmt.Sprintf("hello-%d", attempt)), "./testdata/hello_module/cmd/app"}
		})
		checkSelfHostedFrontendPerf(t, best, info.Size())
		out := filepath.Join(tmp, "hello-0")
		data, err := exec.Command(out).CombinedOutput()
		if err != nil {
			t.Fatalf("self-hosted built fixture failed: %v\n%s", err, string(data))
		}
		if string(data) != "PASS\n" {
			t.Fatalf("self-hosted built fixture output = %q, want PASS", string(data))
		}
	})

	t.Run("emit-unit", func(t *testing.T) {
		best := measureSelfHostedFrontend(t, tmp, self, func(attempt int) []string {
			dir := filepath.Join(tmp, fmt.Sprintf("units-%d", attempt))
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("MkdirAll unit dir failed: %v", err)
			}
			return []string{"-emit-unit", "-o", dir, "./testdata/hello_module/cmd/app"}
		})
		checkSelfHostedFrontendPerf(t, best, info.Size())
	})

	t.Run("link", func(t *testing.T) {
		unitDir := filepath.Join(tmp, "link-units")
		if err := os.MkdirAll(unitDir, 0755); err != nil {
			t.Fatalf("MkdirAll unit dir failed: %v", err)
		}
		cmd := exec.Command(self, "-emit-unit", "-o", unitDir, "./testdata/hello_module/cmd/app")
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("self-hosted fixture emit-unit failed: %v\n%s", err, string(data))
		}
		best := measureSelfHostedFrontend(t, tmp, self, func(attempt int) []string {
			return []string{"-link", "-t", "linux/amd64", "-o", filepath.Join(tmp, fmt.Sprintf("linked-%d", attempt)), unitDir}
		})
		checkSelfHostedFrontendPerf(t, best, info.Size())
		out := filepath.Join(tmp, "linked-0")
		data, err = exec.Command(out).CombinedOutput()
		if err != nil {
			t.Fatalf("self-hosted linked fixture failed: %v\n%s", err, string(data))
		}
		if string(data) != "PASS\n" {
			t.Fatalf("self-hosted linked fixture output = %q, want PASS", string(data))
		}
	})
}

type selfHostedFrontendPerf struct {
	elapsed time.Duration
	maxRSS  int
}

func measureSelfHostedFrontend(t *testing.T, tmp string, self string, argsForAttempt func(int) []string) selfHostedFrontendPerf {
	t.Helper()

	best := selfHostedFrontendPerf{elapsed: 24 * time.Hour, maxRSS: 1 << 30}
	for attempt := 0; attempt < 3; attempt++ {
		args := argsForAttempt(attempt)
		rssFile := filepath.Join(tmp, fmt.Sprintf("%s-rss-%d", strings.ReplaceAll(t.Name(), "/", "-"), attempt))
		timeArgs := append([]string{"-f", "%e %M", "-o", rssFile, self}, args...)
		cmd := exec.Command("/usr/bin/time", timeArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("resource-measured frontend command failed: %v\nOutput: %s", err, string(output))
		}
		elapsed, maxRSS := readSelfHostedFrontendResourceUsage(t, rssFile)
		if elapsed < best.elapsed {
			best.elapsed = elapsed
		}
		if maxRSS < best.maxRSS {
			best.maxRSS = maxRSS
		}
	}
	return best
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

	const maxRuntime = 50 * time.Millisecond
	const maxRSSKB = 16 * 1024
	const maxCompilerSize = 1024 * 1024
	var failures []string
	if perf.elapsed > maxRuntime {
		failures = append(failures, fmt.Sprintf("runtime %s > %s", perf.elapsed, maxRuntime))
	}
	if perf.maxRSS > maxRSSKB {
		failures = append(failures, fmt.Sprintf("max RSS %dKB > %dKB", perf.maxRSS, maxRSSKB))
	}
	if compilerSize > maxCompilerSize {
		failures = append(failures, fmt.Sprintf("compiler binary size %dB > %dB", compilerSize, maxCompilerSize))
	}
	if len(failures) > 0 {
		t.Fatalf("frontend performance limits failed: runtime=%s, max RSS=%dKB, compiler binary size=%dB; failures: %s",
			perf.elapsed, perf.maxRSS, compilerSize, strings.Join(failures, "; "))
	}
}
