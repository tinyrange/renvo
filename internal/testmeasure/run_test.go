package testmeasure

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

var retained uint64

func TestMeasuredChild(t *testing.T) {
	mode := os.Getenv("RENVO_MEASURE_CHILD")
	if mode == "" {
		return
	}
	pages := make([]byte, 32*1024*1024)
	for i := 0; i < len(pages); i += 4096 {
		pages[i] = byte(i / 4096)
	}
	start := time.Now()
	var sum uint64
	for time.Since(start) < 80*time.Millisecond {
		for i := uint64(0); i < 10000; i++ {
			sum = sum*33 + i
		}
	}
	retained = sum
	// Waiting contributes to wall time but not CPU time.
	time.Sleep(350 * time.Millisecond)
	runtime.KeepAlive(pages)
	if mode == "fail" {
		os.Exit(7)
	}
	os.Exit(0)
}

func TestRunAccountsCPUAndPeakMemory(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		t.Skip("unsupported measurement host")
	}
	for _, mode := range []string{"success", "fail"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMeasuredChild$")
			cmd.Env = append(os.Environ(), "RENVO_MEASURE_CHILD="+mode)
			result, err := Run(cmd)
			if mode == "success" && err != nil {
				t.Fatal(err)
			}
			if mode == "fail" {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 7 {
					t.Fatalf("exit error: %v", err)
				}
			}
			if result.CPUNanoseconds <= 0 || result.CPUNanoseconds >= result.ElapsedNanoseconds*3/4 {
				t.Fatalf("CPU must exclude sleep: %+v", result)
			}
			if result.PeakMemoryBytes < 16*1024*1024 {
				t.Fatalf("missing peak memory (including unit normalization): %+v", result)
			}
			want := "peak_rss"
			if runtime.GOOS == "windows" {
				want = "peak_job_commit"
			}
			if result.MemoryMetric != want {
				t.Fatalf("metric %q, want %q", result.MemoryMetric, want)
			}
		})
	}
}

func TestStartFailureIsReported(t *testing.T) {
	_, err := Run(exec.Command("renvo-nonexistent-measurement-fixture"))
	if err == nil {
		t.Fatal("missing executable was accepted")
	}
}
