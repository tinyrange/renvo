package testmeasure

import (
	"errors"
	"os/exec"
	"time"
)

// Run measures a fresh process. The self-hosting workload links its backend
// in-process; OS CPU accounting includes every compiler thread. Virtual-machine
// runners also execute in-process, so their host CPU and memory are included.
// A command which delegates compilation to subprocesses is not this workload.
func Run(cmd *exec.Cmd) (Result, error) {
	observer, err := prepare(cmd)
	if err != nil {
		return Result{}, err
	}
	defer observer.close()
	started := time.Now()
	if err := cmd.Start(); err != nil {
		return Result{}, err
	}
	if err := observer.started(cmd); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return Result{}, err
	}
	waitErr := cmd.Wait()
	result := Result{ElapsedNanoseconds: int64(time.Since(started))}
	if cmd.ProcessState != nil {
		result.CPUNanoseconds = int64(cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime())
	}
	measureErr := observer.finish(cmd, &result)
	if measureErr == nil && (result.PeakMemoryBytes == 0 || result.MemoryMetric == "" || result.CPUNanoseconds <= 0) {
		measureErr = errors.New("OS did not report positive CPU and peak memory measurements")
	}
	return result, errors.Join(waitErr, measureErr)
}
