package bringup

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"

	"renvo.dev/backend/omnibus/resultabi"
	"renvo.dev/backend/target"
)

// Command is one argv-preserving host tool invocation. Commands are never
// interpreted by a shell, so target paths and toolchain arguments remain
// explicit and reproducible.
type Command struct {
	Path            string
	Args            []string
	Dir             string
	Env             []string
	CanonicalInput  string
	CanonicalSHA256 string
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
	// CanonicalInput is the one linked compiler unit consumed by both emitters.
	// Its digest makes the shared-input guarantee auditable even when the two
	// build commands use different tools and output formats.
	CanonicalInput  string
	CanonicalSHA256 string

	ReferenceBuild Command
	CandidateBuild Command
	ReferenceLink  Command
	CandidateLink  Command
	ReferenceRun   Command
	CandidateRun   Command

	Reference PipelineArtifact
	Candidate PipelineArtifact

	ObjectContract    ELFContract
	Composition       target.Composition
	BoardELF          target.ELFArtifactOptions
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
	if p.CanonicalInput == "" || p.CanonicalSHA256 == "" {
		return fmt.Errorf("canonical linked unit path and SHA-256 are required")
	}
	if err := validateCanonicalInput(p.CanonicalInput, p.CanonicalSHA256); err != nil {
		return err
	}
	if err := validateBuildCanonicalInput("reference", p.ReferenceBuild, p.CanonicalInput, p.CanonicalSHA256); err != nil {
		return err
	}
	if err := validateBuildCanonicalInput("candidate", p.CandidateBuild, p.CanonicalInput, p.CanonicalSHA256); err != nil {
		return err
	}
	if p.Reference.Image == "" || p.Candidate.Image == "" || p.Reference.MemoryDump == "" || p.Candidate.MemoryDump == "" {
		return fmt.Errorf("reference and candidate image and memory-dump paths are required")
	}
	if p.ExpectedProfile == 0 {
		return fmt.Errorf("expected result profile is required")
	}
	if p.Composition.Board.Name != "" {
		if err := p.Composition.Validate(); err != nil {
			return fmt.Errorf("invalid board composition: %w", err)
		}
		if p.Toolchain.ObjectFormat != p.Composition.Object.Format.Name || p.Toolchain.ABI != p.Composition.Object.ABI {
			return fmt.Errorf("toolchain object format/ABI %s/%s does not match composition %s/%s",
				p.Toolchain.ObjectFormat, p.Toolchain.ABI, p.Composition.Object.Format.Name, p.Composition.Object.ABI)
		}
		if p.BoardELF.VectorSymbol != p.Composition.Board.Startup.VectorSymbol {
			return fmt.Errorf("board image vector symbol %q does not match composition %q",
				p.BoardELF.VectorSymbol, p.Composition.Board.Startup.VectorSymbol)
		}
		resultSymbol := p.ResultSymbol
		if resultSymbol == "" {
			resultSymbol = resultabi.SymbolName
		}
		if p.Composition.Board.Runtime.Result.Transport == target.ResultTransportDebuggerMemory && resultSymbol != p.Composition.Board.Runtime.Result.Symbol {
			return fmt.Errorf("result symbol %q does not match composition %q", resultSymbol, p.Composition.Board.Runtime.Result.Symbol)
		}
	}
	return nil
}

// CanonicalInputSHA256 returns the lowercase digest recorded by a pipeline
// plan. Callers compute it after producing the canonical linked unit and before
// constructing either backend command.
func CanonicalInputSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read canonical linked unit: %w", err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func validateCanonicalInput(path string, expected string) error {
	decoded, err := hex.DecodeString(expected)
	if err != nil || len(decoded) != sha256.Size || expected != hex.EncodeToString(decoded) {
		return fmt.Errorf("canonical linked unit SHA-256 must be 64 lowercase hexadecimal digits")
	}
	actual, err := CanonicalInputSHA256(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("canonical linked unit SHA-256 is %s; want %s", actual, expected)
	}
	return nil
}

func validateBuildCanonicalInput(side string, command Command, path string, digest string) error {
	if command.CanonicalInput != path || command.CanonicalSHA256 != digest {
		return fmt.Errorf("%s build does not declare the canonical linked unit and SHA-256", side)
	}
	found := false
	for _, argument := range command.Args {
		if argument == path {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%s build command does not consume the canonical linked unit", side)
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
	}
	for _, step := range steps {
		if err := runPipelineCommand(runner, step.side, step.name, step.command); err != nil {
			return result, err
		}
	}
	if plan.Composition.Board.Name != "" {
		if err := validatePipelineImage("reference", plan.Reference.Image, plan.Composition, plan.BoardELF); err != nil {
			return result, err
		}
		if err := validatePipelineImage("candidate", plan.Candidate.Image, plan.Composition, plan.BoardELF); err != nil {
			return result, err
		}
	}
	steps = []struct {
		side    string
		name    string
		command Command
	}{
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

func validatePipelineImage(side string, path string, composition target.Composition, options target.ELFArtifactOptions) error {
	artifact, err := target.ArtifactFromELF(path, options)
	if err != nil {
		return &PipelineError{Side: side, Step: "validate-image", Err: err}
	}
	validation := target.Validate(composition, artifact)
	if validation.OK() {
		return nil
	}
	return &PipelineError{Side: side, Step: "validate-image", Err: validation.Violations[0]}
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
