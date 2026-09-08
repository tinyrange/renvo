//go:build windows

package testmeasure

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var createJob = kernel32.NewProc("CreateJobObjectW")
var assignJob = kernel32.NewProc("AssignProcessToJobObject")
var queryJob = kernel32.NewProc("QueryInformationJobObject")
var setJob = kernel32.NewProc("SetInformationJobObject")
var snapshotThreads = kernel32.NewProc("CreateToolhelp32Snapshot")
var threadFirst = kernel32.NewProc("Thread32First")
var threadNext = kernel32.NewProc("Thread32Next")
var openThread = kernel32.NewProc("OpenThread")
var resumeThread = kernel32.NewProc("ResumeThread")

type jobLimits struct {
	ProcessTime, JobTime         int64
	Flags                        uint32
	MinWorkingSet, MaxWorkingSet uintptr
	ActiveProcesses              uint32
	Affinity                     uintptr
	Priority, Scheduling         uint32
}
type jobExtended struct {
	Basic jobLimits
	// Windows aligns LARGE_INTEGER structures to eight bytes even on x86.
	// Go's 386 ABI needs explicit padding before IO_COUNTERS.
	_                                                          [8 - unsafe.Sizeof(uintptr(0))]byte
	IO                                                         [6]uint64
	ProcessMemory, JobMemory, PeakProcessMemory, PeakJobMemory uintptr
}
type jobAccounting struct {
	UserTime, KernelTime, PeriodUserTime, PeriodKernelTime      int64
	PageFaults, Processes, ActiveProcesses, TerminatedProcesses uint32
}
type threadEntry struct {
	Size, Usage, ID, ProcessID  uint32
	BasePriority, DeltaPriority int32
	Flags                       uint32
}
type observation struct{ job syscall.Handle }

func prepare(cmd *exec.Cmd) (*observation, error) {
	job, _, err := createJob.Call(0, 0)
	if job == 0 {
		return nil, fmt.Errorf("CreateJobObject: %w", err)
	}
	o := &observation{job: syscall.Handle(job)}
	limits := jobExtended{}
	// Never let a failed observer leave its suspended process behind.
	limits.Basic.Flags = 0x2000 // JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if ok, _, err := setJob.Call(job, 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits)); ok == 0 {
		o.close()
		return nil, fmt.Errorf("SetInformationJobObject: %w", err)
	}
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= 0x4 // CREATE_SUSPENDED: account from the first instruction.
	return o, nil
}
func (o *observation) started(cmd *exec.Cmd) error {
	process, err := syscall.OpenProcess(0x100|0x1, false, uint32(cmd.Process.Pid)) // SET_QUOTA | TERMINATE
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(process)
	if ok, _, err := assignJob.Call(uintptr(o.job), uintptr(process)); ok == 0 {
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	snapshot, _, err := snapshotThreads.Call(4, 0) // TH32CS_SNAPTHREAD
	if snapshot == ^uintptr(0) {
		return fmt.Errorf("thread snapshot: %w", err)
	}
	defer syscall.CloseHandle(syscall.Handle(snapshot))
	entry := threadEntry{Size: uint32(unsafe.Sizeof(threadEntry{}))}
	ok, _, _ := threadFirst.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	for ok != 0 {
		if entry.ProcessID == uint32(cmd.Process.Pid) {
			thread, _, err := openThread.Call(2, 0, uintptr(entry.ID)) // THREAD_SUSPEND_RESUME
			if thread == 0 {
				return fmt.Errorf("OpenThread: %w", err)
			}
			count, _, resumeErr := resumeThread.Call(thread)
			syscall.CloseHandle(syscall.Handle(thread))
			if uint32(count) == 0xffffffff {
				return fmt.Errorf("ResumeThread: %w", resumeErr)
			}
			return nil
		}
		ok, _, _ = threadNext.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	}
	return fmt.Errorf("suspended process has no primary thread")
}
func (o *observation) finish(cmd *exec.Cmd, result *Result) error {
	accounting := jobAccounting{}
	if ok, _, err := queryJob.Call(uintptr(o.job), 1, uintptr(unsafe.Pointer(&accounting)), unsafe.Sizeof(accounting), 0); ok == 0 {
		return fmt.Errorf("query job CPU: %w", err)
	}
	if accounting.Processes != 1 || accounting.ActiveProcesses != 0 {
		return fmt.Errorf("self-hosting must use its in-process backend; observed %d processes (%d active)", accounting.Processes, accounting.ActiveProcesses)
	}
	limits := jobExtended{}
	if ok, _, err := queryJob.Call(uintptr(o.job), 9, uintptr(unsafe.Pointer(&limits)), unsafe.Sizeof(limits), 0); ok == 0 {
		return fmt.Errorf("query job memory: %w", err)
	}
	result.CPUNanoseconds = (accounting.UserTime + accounting.KernelTime) * 100
	// Job commit is retained after process exit and is not affected by working
	// set trimming. Label it accurately; it is not RSS or sampled working set.
	result.PeakMemoryBytes = uint64(limits.PeakJobMemory)
	result.MemoryMetric = "peak_job_commit"
	return nil
}
func (o *observation) close() { syscall.CloseHandle(o.job) }
