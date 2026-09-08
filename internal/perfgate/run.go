package perfgate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"renvo.dev/internal/testmeasure"
)

type Report struct {
	Policy            Policy   `json:"policy"`
	Target            Target   `json:"target"`
	Host              string   `json:"host"`
	ReferenceRevision string   `json:"reference_revision"`
	CandidateRevision string   `json:"candidate_revision"`
	CandidateDirty    bool     `json:"candidate_dirty"`
	RunnerVersion     string   `json:"runner_version"`
	Reference         []Sample `json:"reference"`
	Candidate         []Sample `json:"candidate"`
	Failures          []string `json:"failures"`
	FailedMeasurement *Sample  `json:"failed_measurement,omitempty"`
	OK                bool     `json:"ok"`
}

type build struct{ root, directory, stage2, unit string }
type harness struct {
	policy  Policy
	target  Target
	self    string
	log     io.Writer
	report  *Report
	fixture []byte
}

// Run compiles two real revisions through stage2, then alternates fresh
// stage2->stage3 self-hosts. No compiler cache or host-Go oracle participates in
// a measured build. The emitted compiler must compile and run a smoke program.
func Run(root, reference, targetName string, log io.Writer) (report Report, err error) {
	p := Load()
	report.Policy, report.Host = p, runtime.GOOS+"/"+runtime.GOARCH
	if err = p.Validate(); err != nil {
		return
	}
	target, err := p.Target(targetName)
	if err != nil {
		return report, err
	}
	report.Target = target
	if err = runnable(target); err != nil {
		return report, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return report, err
	}
	report.CandidateRevision = revision(root)
	report.CandidateDirty, err = dirty(root)
	if err != nil {
		return report, err
	}
	work, err := os.MkdirTemp("", "renvo-selfhost-")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(work)
	if reference == "" {
		reference = filepath.Join(work, "reference")
		cmd := exec.Command("git", "worktree", "add", "--detach", reference, p.ReferenceRevision)
		cmd.Dir = root
		if output, e := cmd.CombinedOutput(); e != nil {
			return report, fmt.Errorf("prepare reference: %w\n%s", e, output)
		}
		defer func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", reference)
			cmd.Dir = root
			if output, e := cmd.CombinedOutput(); e != nil {
				fmt.Fprintf(log, "reference cleanup: %v %s\n", e, output)
			}
		}()
	}
	reference, err = filepath.Abs(reference)
	if err != nil {
		return report, err
	}
	report.ReferenceRevision = revision(reference)
	if report.ReferenceRevision != p.ReferenceRevision {
		return report, fmt.Errorf("reference must be policy revision %s, got %s", p.ReferenceRevision, report.ReferenceRevision)
	}
	changed, err := dirty(reference)
	if err != nil {
		return report, err
	}
	if changed {
		return report, fmt.Errorf("reference checkout must be clean; use the pinned revision without local changes")
	}
	self, err := os.Executable()
	if err != nil {
		return report, err
	}
	h := harness{policy: p, target: target, self: self, log: log, report: &report}
	if target.Workload == "prepared-backend" {
		h.fixture, err = os.ReadFile(filepath.Join(root, "internal/backendjit/testdata/semantic_runtime.go"))
		if err != nil {
			return report, err
		}
	}
	if target.Execution == "wasmtime" || target.Execution == "qemu-arm" {
		args := []string{"--version"}
		out, e := exec.Command(target.Execution, args...).CombinedOutput()
		if e != nil {
			return report, e
		}
		report.RunnerVersion = strings.TrimSpace(string(out))
	} else if target.Execution == "vm" {
		report.RunnerVersion = "candidate std/vm (identical runner for both compilers)"
	}
	base := build{root: reference, directory: filepath.Join(work, "base")}
	next := build{root: root, directory: filepath.Join(work, "candidate")}
	for _, b := range []*build{&base, &next} {
		if target.Execution == "wasmtime" {
			// Renvo resolves WASI paths relative to a preopened workspace.
			// Keep output there too, without relying on guest mount-name lookup.
			sandbox := filepath.Join(b.root, "sandbox")
			if err = os.MkdirAll(sandbox, 0755); err != nil {
				return report, err
			}
			var directory string
			directory, err = os.MkdirTemp(sandbox, "perfgate-wasi-")
			if err != nil {
				return report, err
			}
			defer os.RemoveAll(directory)
			b.directory = filepath.Join(directory, filepath.Base(b.directory))
		}
		if err = h.prepare(b); err != nil {
			return report, err
		}
	}
	for _, b := range []*build{&base, &next} {
		fmt.Fprintf(log, "%s warm-up workload\n", filepath.Base(b.directory))
		if _, err = h.sample(b, "warmup"); err != nil {
			return report, err
		}
	}
	for i := 0; i < p.Samples; i++ {
		// Alternate which revision runs first to reduce systematic drift.
		pair := []*build{&base, &next}
		if i%2 == 1 {
			pair[0], pair[1] = pair[1], pair[0]
		}
		for _, b := range pair {
			var sample Sample
			sample, err = h.sample(b, fmt.Sprintf("sample-%d", i+1))
			if err != nil {
				return report, err
			}
			if b == &base {
				report.Reference = append(report.Reference, sample)
			} else {
				report.Candidate = append(report.Candidate, sample)
			}
			fmt.Fprintf(log, "%s %s sample %d: CPU=%s wall=%s memory=%d (%s) artifact=%d VM steps=%d\n", target.Name, filepath.Base(b.directory), i+1, time.Duration(sample.CPU), time.Duration(sample.Wall), sample.Memory, sample.MemoryMetric, sample.Artifact, sample.VMSteps)
		}
	}
	report.Failures = p.Check(report.Reference, report.Candidate, target.Execution == "vm")
	report.OK = len(report.Failures) == 0
	if !report.OK {
		return report, fmt.Errorf("compiler performance gate failed:\n%s", strings.Join(report.Failures, "\n"))
	}
	return report, nil
}

func revision(root string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func dirty(root string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain", "--untracked-files=normal")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("inspect checkout: %w", err)
	}
	return len(out) != 0, nil
}

func runnable(t Target) error {
	host := NativeTarget()
	if t.Execution == "native" {
		if t.Name == host || host == "linux/amd64" && t.Name == "linux/386" || host == "windows/amd64" && t.Name == "windows/386" {
			return nil
		}
		return fmt.Errorf("%s must execute on a compatible native host; current host is %s", t.Name, host)
	}
	if t.Execution == "qemu-arm" || t.Execution == "wasmtime" {
		if t.Execution == "qemu-arm" && runtime.GOOS != "linux" {
			return fmt.Errorf("%s performance runner requires Linux", t.Execution)
		}
		if _, err := exec.LookPath(t.Execution); err != nil {
			return fmt.Errorf("required runner %s missing: %w", t.Execution, err)
		}
	}
	return nil
}

func compilerArgs(p Policy, target, output, source string) []string {
	return []string{"-tags", "renvo_bundle", "-t", target, "-arena-size", fmt.Sprint(p.ArenaBytes(target)), "-s", "-o", output, source}
}
func (h harness) prepare(b *build) error {
	if err := os.MkdirAll(b.directory, 0755); err != nil {
		return err
	}
	stage0 := filepath.Join(b.directory, "stage0")
	if runtime.GOOS == "windows" {
		stage0 += ".exe"
	}
	fmt.Fprintf(h.log, "%s: bootstrap %s\n", h.target.Name, filepath.Base(b.directory))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.policy.InvocationTimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-tags", "renvo_bundle", "-o", stage0, "./cmd/renvo")
	cmd.Dir = b.root
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bootstrap: %w\n%s", err, out)
	}
	if h.target.Workload == "prepared-backend" {
		return h.prepareVMBackend(b, stage0)
	}
	stage1 := filepath.Join(b.directory, "stage1")
	if strings.HasPrefix(h.target.Name, "windows/") {
		stage1 += ".exe"
	}
	cmd = exec.CommandContext(ctx, stage0, compilerArgs(h.policy, h.target.Name, stage1, "./cmd/renvo")...)
	cmd.Dir = b.root
	cmd.Env = compilerEnv(b.root)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build stage1: %w\n%s", err, out)
	}
	b.stage2 = filepath.Join(b.directory, "stage2")
	if strings.HasPrefix(h.target.Name, "windows/") {
		b.stage2 += ".exe"
	}
	_, err := h.execute(b, stage1, compilerArgs(h.policy, h.target.Name, b.stage2, "./cmd/renvo"), b.stage2)
	return err
}

func compilerEnv(root string) []string {
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "RENVO_") || key == "PWD" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "PWD="+root, "RENVO_STDROOT="+filepath.Join(root, "std"))
}

func (h harness) execute(b *build, compiler string, args []string, output string) (Sample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.policy.InvocationTimeoutSeconds)*time.Second)
	defer cancel()
	command := compiler
	statsPath := output + ".vm.json"
	switch h.target.Execution {
	case "qemu-arm":
		args = append([]string{compiler}, args...)
		command = "qemu-arm"
	case "wasmtime":
		mapped := append([]string(nil), args...)
		relative, err := filepath.Rel(b.root, output)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return Sample{}, fmt.Errorf("WASI output must be inside its workspace: %s", output)
		}
		for i, arg := range mapped {
			if arg == output {
				mapped[i] = filepath.ToSlash(relative)
			}
		}
		args = append([]string{"run", "--dir", b.root + "::.", "--env", "PWD=.", "--env", "RENVO_STDROOT=std", compiler}, mapped...)
		command = "wasmtime"
	case "vm":
		request := VMRequest{Root: b.root, Compiler: compiler, Output: output, Stats: statsPath, Args: args, Memory: int(h.policy.PeakMemoryBytes), Steps: h.policy.VMStepLimit, Input: b.unit}
		data, err := json.Marshal(request)
		if err != nil {
			return Sample{}, err
		}
		path := output + ".request.json"
		if err = os.WriteFile(path, data, 0600); err != nil {
			return Sample{}, err
		}
		args = []string{"-execute-vm", path}
		command = h.self
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = b.root
	cmd.Env = compilerEnv(b.root)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	usage, err := testmeasure.Run(cmd)
	sample := Sample{CPU: usage.CPUNanoseconds, Wall: usage.ElapsedNanoseconds, Memory: usage.PeakMemoryBytes, MemoryMetric: usage.MemoryMetric}
	if err != nil {
		if h.report != nil {
			h.report.FailedMeasurement = &sample
		}
		return sample, fmt.Errorf("%s %s compiler execution: %w\n%s", h.target.Name, filepath.Base(compiler), err, log.String())
	}
	if h.target.Execution == "vm" {
		data, err := os.ReadFile(statsPath)
		if err != nil {
			return sample, err
		}
		var stats VMStats
		if err = json.Unmarshal(data, &stats); err != nil {
			return sample, err
		}
		sample.VMSteps = stats.Steps
		sample.VMMemory = uint64(stats.Memory)
	}
	return sample, nil
}

func (h harness) sample(b *build, name string) (Sample, error) {
	if h.target.Workload == "prepared-backend" {
		return h.sampleVMBackend(b, name)
	}
	output := filepath.Join(b.directory, name)
	if strings.HasPrefix(h.target.Name, "windows/") {
		output += ".exe"
	}
	sample, err := h.execute(b, b.stage2, compilerArgs(h.policy, h.target.Name, output, "./cmd/renvo"), output)
	if err != nil {
		return sample, err
	}
	info, err := os.Stat(output)
	if err != nil {
		return sample, err
	}
	sample.Artifact = uint64(info.Size())
	// Exercise the generated compiler, not just its executable header. The
	// output program is portable across every Tier 1 and prints exactly PASS.
	smokeDir := filepath.Join(b.root, "sandbox")
	if err = os.MkdirAll(smokeDir, 0755); err != nil {
		return sample, err
	}
	fixture, err := os.MkdirTemp(smokeDir, "selfhost-smoke-")
	if err != nil {
		return sample, err
	}
	defer os.RemoveAll(fixture)
	source := filepath.Join(fixture, "main.go")
	if err = os.WriteFile(source, []byte("package main\nfunc main() { value := 6; if value*7 != 42 { panic(\"bad arithmetic\") }; print(\"PASS\\n\") }\n"), 0644); err != nil {
		return sample, err
	}
	relative, _ := filepath.Rel(b.root, source)
	program := filepath.Join(b.directory, "smoke")
	if strings.HasPrefix(h.target.Name, "windows/") {
		program += ".exe"
	}
	if _, err = h.execute(b, output, compilerArgs(h.policy, h.target.Name, program, "./"+filepath.ToSlash(relative)), program); err != nil {
		return sample, fmt.Errorf("stage3 smoke compile: %w", err)
	}
	return sample, h.smoke(b, program)
}

func (h harness) smoke(b *build, program string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command, args := program, []string{}
	switch h.target.Execution {
	case "qemu-arm":
		command, args = "qemu-arm", []string{program}
	case "wasmtime":
		command, args = "wasmtime", []string{"run", program}
	case "vm":
		request := VMRequest{Root: b.root, Compiler: program, Memory: int(h.policy.PeakMemoryBytes), Steps: h.policy.VMStepLimit}
		data, _ := json.Marshal(request)
		path := program + ".smoke.json"
		if err := os.WriteFile(path, data, 0600); err != nil {
			return err
		}
		command, args = h.self, []string{"-execute-vm", path}
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = b.root
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		return fmt.Errorf("stage3 generated-program smoke: %v, output %q", err, out)
	}
	return nil
}
