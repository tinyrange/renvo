package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type measuredProcessUsage struct {
	output   []byte
	elapsed  time.Duration
	cpu      time.Duration
	maxRSSKB int
}

func measurementHelper(t *testing.T) string {
	paths := []string{
		"../internal/testmeasure/result.go",
		"../internal/testmeasure/cmd/renvomeasure/main.go",
		"../internal/testmeasure/cmd/renvomeasure/usage_linux.go",
		"../internal/testmeasure/cmd/renvomeasure/usage_other.go",
	}
	key := testArtifactKeyForFiles(t, []string{"measurement-helper"}, paths)
	return cachedTestArtifact(t, "measurement-helper", key, func(output string) error {
		cmd := exec.Command("go", "build", "-o", output, "../internal/testmeasure/cmd/renvomeasure")
		combined, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("build measurement helper: %w: %s", err, combined)
		}
		return nil
	})
}

func runMeasuredProcess(t *testing.T, dir string, env []string, path string, args ...string) (measuredProcessUsage, error) {
	t.Helper()
	resultFile, err := os.CreateTemp("", "renvo-process-usage-*.json")
	if err != nil {
		return measuredProcessUsage{}, err
	}
	resultPath := resultFile.Name()
	if err := resultFile.Close(); err != nil {
		return measuredProcessUsage{}, err
	}
	defer os.Remove(resultPath)

	helper := measurementHelper(t)
	helperArgs := append([]string{resultPath, path}, args...)
	cmd := exec.Command(helper, helperArgs...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	data, readErr := os.ReadFile(resultPath)
	if readErr != nil {
		return measuredProcessUsage{output: output}, fmt.Errorf("read measurement result: %w", readErr)
	}
	var result struct {
		ElapsedNanoseconds int64 `json:"elapsed_nanoseconds"`
		CPUNanoseconds     int64 `json:"cpu_nanoseconds"`
		MaxRSSKB           int   `json:"max_rss_kb"`
	}
	if decodeErr := json.Unmarshal(data, &result); decodeErr != nil {
		return measuredProcessUsage{output: output}, fmt.Errorf("decode measurement result %s: %w", filepath.Base(resultPath), decodeErr)
	}
	usage := measuredProcessUsage{
		output: output, elapsed: time.Duration(result.ElapsedNanoseconds),
		cpu: time.Duration(result.CPUNanoseconds), maxRSSKB: result.MaxRSSKB,
	}
	return usage, err
}

func TestRunMeasuredProcess(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process usage assertions require Linux wait4 metrics")
	}
	usage, err := runMeasuredProcess(t, "", []string{}, "/bin/sh", "-c", "printf measured; i=0; while [ $i -lt 10000 ]; do i=$((i+1)); done")
	if err != nil {
		t.Fatalf("measured command failed: %v", err)
	}
	if string(usage.output) != "measured" {
		t.Fatalf("measured output = %q, want %q", usage.output, "measured")
	}
	if usage.elapsed <= 0 || usage.cpu <= 0 {
		t.Fatalf("invalid measured durations: elapsed=%s CPU=%s", usage.elapsed, usage.cpu)
	}
	if usage.maxRSSKB <= 0 {
		t.Fatalf("invalid measured max RSS: %dKB", usage.maxRSSKB)
	}

	_, err = runMeasuredProcess(t, "", []string{}, "/bin/sh", "-c", "exit 7")
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 7 {
		t.Fatalf("measured exit error = %v, want exit code 7", err)
	}
}
