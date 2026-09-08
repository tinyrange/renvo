//go:build linux || darwin

package testmeasure

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
)

type observation struct{}

func prepare(cmd *exec.Cmd) (*observation, error) { return &observation{}, nil }
func (*observation) started(cmd *exec.Cmd) error  { return nil }
func (*observation) close()                       {}
func (*observation) finish(cmd *exec.Cmd, result *Result) error {
	if cmd.ProcessState == nil {
		return fmt.Errorf("missing child process state")
	}
	usage, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	if !ok || usage.Maxrss <= 0 {
		return fmt.Errorf("missing child peak RSS")
	}
	result.PeakMemoryBytes = uint64(usage.Maxrss)
	if runtime.GOOS == "linux" {
		result.PeakMemoryBytes *= 1024
	}
	result.MaxRSSKB = int((result.PeakMemoryBytes + 1023) / 1024)
	result.MemoryMetric = "peak_rss"
	return nil
}
