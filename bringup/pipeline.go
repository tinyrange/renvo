package bringup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"j5.nz/rtg/omnibus/resultabi"
)

// Command is one argv-preserving host tool invocation. Commands are never
// interpreted by a shell, so target paths and toolchain arguments remain
// explicit and reproducible.
type Command struct {
	Path string
	Args []string
	Dir  string
	Env  []string
}

type CommandRunner interface {
	Run(Command) ([]byte, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(command Command) ([]byte, error) {
	if command.Path == "" {
		return nil, nil
	}
	cmd := exec.Command(command.Path, command.Args...)
	cmd.Dir = command.Dir
	if len(command.Env) != 0 {
		cmd.Env = append(os.Environ(), command.Env...)
	}
	return cmd.CombinedOutput()
}

type PipelineArtifact struct {
	Object               string
	Image                string
	MemoryDump           string
	MemoryBase           uint64
	MemoryAtResultSymbol bool
}

// PipelinePlan describes both sides of one bring-up milestone. Reference and
// candidate commands consume the same canonical input selected by the caller;
// the plan keeps their build, link, run, and result artifacts distinct.
type PipelinePlan struct {
	Toolchain Toolchain
	Stage     Stage

	ReferenceBuild Command
	CandidateBuild Command
	ReferenceLink  Command
	CandidateLink  Command
	ReferenceRun   Command
	CandidateRun   Command

	Reference PipelineArtifact
	Candidate PipelineArtifact

	ObjectContract    ELFContract
	ResultSymbol      string
	ExpectedProfile   uint32
	ExpectedSignature uint64
}

type PipelineResult struct {
	Milestone Milestone
	Reference resultabi.Snapshot
	Candidate resultabi.Snapshot
}

type PipelineError struct {
	Side   string
	Step   string
	Output []byte
	Err    error
}

func (e *PipelineError) Error() string {
	prefix := e.Step
	if e.Side != "" {
		prefix = e.Side + " " + e.Step
	}
	if len(e.Output) == 0 {
		return prefix + ": " + e.Err.Error()
	}
	return fmt.Sprintf("%s: %v: %s", prefix, e.Err, bytes.TrimSpace(e.Output))
}

func (e *PipelineError) Unwrap() error {
	return e.Err
}

func (p PipelinePlan) Validate() error {
	if err := p.Toolchain.Validate(); err != nil {
		return err
	}
	if p.Stage.Milestone < MilestoneConstantFunction || p.Stage.Milestone > MilestoneStandaloneImage {
		return fmt.Errorf("invalid bring-up milestone %d", p.Stage.Milestone)
	}
	if p.Stage.Milestone != MilestoneStandaloneImage && (p.Reference.Object == "" || p.Candidate.Object == "") {
		return fmt.Errorf("reference and candidate object paths are required")
	}
	if p.Reference.Image == "" || p.Candidate.Image == "" || p.Reference.MemoryDump == "" || p.Candidate.MemoryDump == "" {
		return fmt.Errorf("reference and candidate image and memory-dump paths are required")
	}
	if p.ExpectedProfile == 0 {
		return fmt.Errorf("expected result profile is required")
	}
	return nil
}

func RunPipeline(plan PipelinePlan, runner CommandRunner) (PipelineResult, error) {
	var result PipelineResult
	result.Milestone = plan.Stage.Milestone
	if err := plan.Validate(); err != nil {
		return result, &PipelineError{Step: "validate-plan", Err: err}
	}
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	steps := []struct {
		side    string
		name    string
		command Command
	}{
		{side: "reference", name: "build", command: plan.ReferenceBuild},
		{side: "candidate", name: "build", command: plan.CandidateBuild},
	}
	for _, step := range steps {
		if err := runPipelineCommand(runner, step.side, step.name, step.command); err != nil {
			return result, err
		}
	}
	contract := plan.ObjectContract
	contract.RequiredExports = appendUniqueStrings(contract.RequiredExports, plan.Stage.RequiredExports...)
	if plan.Reference.Object != "" {
		if err := validatePipelineObject("reference", plan.Reference.Object, contract); err != nil {
			return result, err
		}
	}
	if plan.Candidate.Object != "" {
		if err := validatePipelineObject("candidate", plan.Candidate.Object, contract); err != nil {
			return result, err
		}
	}
	steps = []struct {
		side    string
		name    string
		command Command
	}{
		{side: "reference", name: "link", command: plan.ReferenceLink},
		{side: "candidate", name: "link", command: plan.CandidateLink},
		{side: "reference", name: "run", command: plan.ReferenceRun},
		{side: "candidate", name: "run", command: plan.CandidateRun},
	}
	for _, step := range steps {
		if err := runPipelineCommand(runner, step.side, step.name, step.command); err != nil {
			return result, err
		}
	}
	symbol := plan.ResultSymbol
	if symbol == "" {
		symbol = resultabi.SymbolName
	}
	reference, err := decodePipelineResult(plan.Reference, symbol)
	if err != nil {
		return result, &PipelineError{Side: "reference", Step: "decode-result", Err: err}
	}
	result.Reference = reference
	if err := reference.ValidatePass(plan.ExpectedProfile, plan.ExpectedSignature); err != nil {
		return result, &PipelineError{Side: "reference", Step: "validate-result", Err: err}
	}
	candidate, err := decodePipelineResult(plan.Candidate, symbol)
	if err != nil {
		return result, &PipelineError{Side: "candidate", Step: "decode-result", Err: err}
	}
	result.Candidate = candidate
	if err := candidate.ValidatePass(plan.ExpectedProfile, plan.ExpectedSignature); err != nil {
		return result, &PipelineError{Side: "candidate", Step: "validate-result", Err: err}
	}
	if reference.CompletedProbes != candidate.CompletedProbes || reference.Signature != candidate.Signature {
		return result, &PipelineError{Side: "candidate", Step: "compare-result", Err: fmt.Errorf("completed/signature = %d/%#x; reference is %d/%#x", candidate.CompletedProbes, candidate.Signature, reference.CompletedProbes, reference.Signature)}
	}
	return result, nil
}

func decodePipelineResult(artifact PipelineArtifact, symbol string) (resultabi.Snapshot, error) {
	base := artifact.MemoryBase
	if artifact.MemoryAtResultSymbol {
		address, err := resultabi.ELFSymbolAddress(artifact.Image, symbol)
		if err != nil {
			return resultabi.Snapshot{}, err
		}
		base = address
	}
	return resultabi.DecodeMemoryDump(artifact.Image, artifact.MemoryDump, base, symbol)
}

func runPipelineCommand(runner CommandRunner, side string, step string, command Command) error {
	output, err := runner.Run(command)
	if err == nil {
		return nil
	}
	return &PipelineError{Side: side, Step: step, Output: output, Err: err}
}

func validatePipelineObject(side string, path string, contract ELFContract) error {
	validation := ValidateELFObject(path, contract)
	if validation.OK() {
		return nil
	}
	return &PipelineError{Side: side, Step: "validate-object", Err: validation.Violations[0]}
}

func appendUniqueStrings(values []string, additions ...string) []string {
	result := append([]string(nil), values...)
	for _, addition := range additions {
		if !containsString(result, addition) {
			result = append(result, addition)
		}
	}
	return result
}
