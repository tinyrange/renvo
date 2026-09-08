// Package perfgate owns the complete-compiler self-hosting performance policy.
package perfgate

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"strings"

	"renvo.dev/internal/targetinfo"
)

//go:embed policy.json
var policyData []byte

type Target struct {
	Name       string `json:"name"`
	Runner     string `json:"runner"`
	Execution  string `json:"execution"`
	Workload   string `json:"workload"`
	ArenaBytes uint64 `json:"arena_bytes,omitempty"`
}
type Policy struct {
	Version                  int      `json:"version"`
	ReferenceRevision        string   `json:"reference_revision"`
	Samples                  int      `json:"samples"`
	CompilerBytes            uint64   `json:"compiler_bytes"`
	PeakMemoryBytes          uint64   `json:"peak_memory_bytes"`
	CompilerArenaBytes       uint64   `json:"compiler_arena_bytes"`
	CPUGrowth                int      `json:"cpu_growth_percent"`
	MemoryGrowth             int      `json:"memory_growth_percent"`
	ArtifactGrowth           int      `json:"artifact_growth_percent"`
	VMStepsGrowth            int      `json:"vm_steps_growth_percent"`
	InvocationTimeoutSeconds int      `json:"invocation_timeout_seconds"`
	VMStepLimit              int64    `json:"vm_step_limit"`
	Targets                  []Target `json:"targets"`
}

func Load() Policy {
	var policy Policy
	if err := json.Unmarshal(policyData, &policy); err != nil {
		panic(err)
	}
	return policy
}

func (p Policy) Validate() error {
	if p.Version != 1 || len(p.ReferenceRevision) != 40 || p.Samples < 3 || p.Samples%2 == 0 || p.CompilerBytes == 0 || p.PeakMemoryBytes == 0 || p.CompilerArenaBytes == 0 || p.CompilerArenaBytes >= p.PeakMemoryBytes || p.InvocationTimeoutSeconds <= 0 || p.VMStepLimit <= 0 {
		return fmt.Errorf("invalid self-hosting policy")
	}
	if p.CPUGrowth <= 0 || p.MemoryGrowth <= 0 || p.ArtifactGrowth <= 0 || p.VMStepsGrowth <= 0 {
		return fmt.Errorf("invalid regression thresholds")
	}
	seen := map[string]bool{}
	for _, target := range p.Targets {
		if !slices.Contains([]string{"selfhost", "prepared-backend"}, target.Workload) || target.Workload == "prepared-backend" && target.Execution != "vm" {
			return fmt.Errorf("invalid workload for %q", target.Name)
		}
		if _, ok := targetinfo.Lookup(target.Name); !ok || seen[target.Name] || target.Runner == "" || !slices.Contains([]string{"native", "qemu-arm", "wasmtime", "vm"}, target.Execution) {
			return fmt.Errorf("invalid or duplicate performance target %q", target.Name)
		}
		seen[target.Name] = true
	}
	if len(seen) == 0 {
		return fmt.Errorf("no performance targets configured")
	}
	return nil
}

func (p Policy) ArenaBytes(targetName string) uint64 {
	for _, target := range p.Targets {
		if target.Name == targetName && target.ArenaBytes != 0 {
			return target.ArenaBytes
		}
	}
	return p.CompilerArenaBytes
}

func NativeTarget() string {
	arch := runtime.GOARCH
	if arch == "arm64" && runtime.GOOS == "linux" {
		arch = "aarch64"
	}
	return runtime.GOOS + "/" + arch
}

func (p Policy) Target(name string) (Target, error) {
	for _, target := range p.Targets {
		if target.Name == name {
			return target, nil
		}
	}
	return Target{}, fmt.Errorf("%q is not a Tier 1 performance target", name)
}

func (p Policy) PlatformTargets(platform string) ([]Target, error) {
	if !slices.Contains([]string{"linux", "windows", "darwin", "virtual"}, platform) {
		return nil, fmt.Errorf("unknown performance platform %q", platform)
	}
	var targets []Target
	for _, target := range p.Targets {
		os, _, _ := strings.Cut(target.Name, "/")
		if os == platform || platform == "virtual" && (os == "wasi" || os == "vm") {
			targets = append(targets, target)
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no performance targets for %q", platform)
	}
	return targets, nil
}

type Sample struct {
	CPU          int64  `json:"cpu_nanoseconds"`
	Wall         int64  `json:"elapsed_nanoseconds"`
	Memory       uint64 `json:"peak_memory_bytes"`
	MemoryMetric string `json:"memory_metric"`
	Artifact     uint64 `json:"artifact_bytes"`
	VMSteps      int    `json:"vm_steps,omitempty"`
	VMMemory     uint64 `json:"vm_peak_memory_bytes,omitempty"`
}

func Median(samples []Sample) Sample {
	cpus, walls := make([]int64, len(samples)), make([]int64, len(samples))
	memory, sizes, vmMemory := make([]uint64, len(samples)), make([]uint64, len(samples)), make([]uint64, len(samples))
	steps := make([]int, len(samples))
	for i, sample := range samples {
		cpus[i], walls[i], memory[i], sizes[i], steps[i], vmMemory[i] = sample.CPU, sample.Wall, sample.Memory, sample.Artifact, sample.VMSteps, sample.VMMemory
	}
	slices.Sort(cpus)
	slices.Sort(walls)
	slices.Sort(memory)
	slices.Sort(sizes)
	slices.Sort(steps)
	slices.Sort(vmMemory)
	middle := len(samples) / 2
	return Sample{CPU: cpus[middle], Wall: walls[middle], Memory: memory[middle], Artifact: sizes[middle], MemoryMetric: samples[0].MemoryMetric, VMSteps: steps[middle], VMMemory: vmMemory[middle]}
}

// Check blocks inclusion on large growth; it never edits or ratchets a budget.
// Absolute ceilings apply to every sample, even when the median is smaller.
func (p Policy) Check(reference, candidate []Sample, vm bool) []string {
	var failures []string
	for name, samples := range map[string][]Sample{"reference": reference, "candidate": candidate} {
		if len(samples) != p.Samples {
			failures = append(failures, fmt.Sprintf("%s: expected %d samples, got %d", name, p.Samples, len(samples)))
			continue
		}
		for i, sample := range samples {
			if sample.CPU <= 0 || sample.Memory == 0 || sample.Artifact == 0 || sample.MemoryMetric == "" || sample.MemoryMetric != samples[0].MemoryMetric || vm && (sample.VMSteps <= 0 || sample.VMMemory == 0) {
				failures = append(failures, fmt.Sprintf("%s sample %d: missing or inconsistent measurements", name, i+1))
			}
			if name == "candidate" && (sample.Artifact > p.CompilerBytes || sample.Memory > p.PeakMemoryBytes || sample.VMMemory > p.PeakMemoryBytes) {
				failures = append(failures, fmt.Sprintf("candidate sample %d exceeds absolute ceiling: artifact=%d/%d memory=%d/%d VM memory=%d", i+1, sample.Artifact, p.CompilerBytes, sample.Memory, p.PeakMemoryBytes, sample.VMMemory))
			}
		}
	}
	if len(failures) != 0 {
		return failures
	}
	base, next := Median(reference), Median(candidate)
	if base.MemoryMetric != next.MemoryMetric {
		return []string{"reference and candidate memory metrics differ"}
	}
	compare := func(name string, old, current uint64, percent int) {
		if float64(current) > float64(old)*(1+float64(percent)/100) {
			failures = append(failures, fmt.Sprintf("%s regression: %d -> %d exceeds +%d%%; blocks feature inclusion pending maintainer evaluation", name, old, current, percent))
		}
	}
	compare("CPU", uint64(base.CPU), uint64(next.CPU), p.CPUGrowth)
	compare("peak memory", base.Memory, next.Memory, p.MemoryGrowth)
	compare("compiler artifact", base.Artifact, next.Artifact, p.ArtifactGrowth)
	if vm {
		compare("VM instructions", uint64(base.VMSteps), uint64(next.VMSteps), p.VMStepsGrowth)
		compare("VM memory", base.VMMemory, next.VMMemory, p.MemoryGrowth)
	}
	return failures
}
