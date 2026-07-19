package bringup

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"renvo.dev/backend/omnibus/resultabi"
	"renvo.dev/backend/target"
)

func TestRunPipelineBuildsValidatesLinksAndComparesResults(t *testing.T) {
	requireHostELFLinker(t)
	compiler, compilerArgs := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "renvo_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, compilerArgs, dir, source, signature, profile)
	result, err := RunPipeline(plan, ExecCommandRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Milestone != MilestoneConstantFunction || result.Reference.CompletedProbes != 1 || result.Candidate.CompletedProbes != 1 {
		t.Fatalf("pipeline result = %#v", result)
	}
	if result.Reference.Signature != result.Candidate.Signature || result.Candidate.Signature != signature {
		t.Fatalf("result signatures = %#x/%#x, want %#x", result.Reference.Signature, result.Candidate.Signature, signature)
	}
}

func TestRunPipelineStopsAtCandidateObjectContract(t *testing.T) {
	compiler, compilerArgs := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	referenceSource := filepath.Join(dir, "reference.c")
	writePipelineCSource(t, referenceSource, "renvo_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, compilerArgs, dir, referenceSource, signature, profile)
	plan.CandidateBuild.Args = append([]string(nil), compilerArgs...)
	plan.CandidateBuild.Args = append(plan.CandidateBuild.Args, "-std=c89", "-Drenvo_stage0=wrong_export", referenceSource, "-c", "-o", plan.Candidate.Object)
	_, err := RunPipeline(plan, ExecCommandRunner{})
	if err == nil {
		t.Fatal("pipeline accepted candidate without milestone export")
	}
	var pipelineErr *PipelineError
	if !errors.As(err, &pipelineErr) || pipelineErr.Side != "candidate" || pipelineErr.Step != "validate-object" {
		t.Fatalf("pipeline error = %#v", err)
	}
	if _, statErr := os.Stat(plan.Candidate.Image); !os.IsNotExist(statErr) {
		t.Fatalf("candidate link ran after object failure: %v", statErr)
	}
}

func TestPipelineRejectsDifferentOrChangedCanonicalUnits(t *testing.T) {
	compiler, compilerArgs := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "renvo_stage0")
	digest, err := CanonicalInputSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, compilerArgs, dir, source, signature, profile)
	plan.CandidateBuild.CanonicalInput = filepath.Join(dir, "different.unit")
	if _, err := RunPipeline(plan, ExecCommandRunner{}); err == nil || !strings.Contains(err.Error(), "candidate build") {
		t.Fatalf("different candidate input error = %v", err)
	}

	plan = testPipelinePlan(compiler, compilerArgs, dir, source, signature, profile)
	if err := os.WriteFile(source, []byte("changed after planning\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RunPipeline(plan, ExecCommandRunner{}); err == nil || !strings.Contains(err.Error(), digest) {
		t.Fatalf("changed canonical input error = %v", err)
	}
}

func TestPipelineRejectsBoardImageBeforeRunningTarget(t *testing.T) {
	requireHostELFLinker(t)
	compiler, compilerArgs := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "renvo_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, compilerArgs, dir, source, signature, profile)
	plan.Composition = target.CH32V003()
	plan.Toolchain.ObjectFormat = plan.Composition.Object.Format.Name
	plan.Toolchain.ABI = plan.Composition.Object.ABI
	plan.BoardELF = target.ELFArtifactOptions{VectorSymbol: "renvo_vectors"}
	plan.ReferenceRun = Command{Path: "reference-run-must-not-start"}
	plan.CandidateRun = Command{Path: "candidate-run-must-not-start"}
	runner := &recordingRunner{}
	_, err := RunPipeline(plan, runner)
	if err == nil {
		t.Fatal("pipeline accepted a host image for the CH32V003 board")
	}
	var pipelineErr *PipelineError
	if !errors.As(err, &pipelineErr) || pipelineErr.Side != "reference" || pipelineErr.Step != "validate-image" {
		t.Fatalf("pipeline error = %#v", err)
	}
	for _, command := range runner.commands {
		if strings.Contains(command.Path, "must-not-start") {
			t.Fatalf("target run started before board validation: %#v", command)
		}
	}
}

func TestPipelineRejectsToolchainCompositionDrift(t *testing.T) {
	compiler, compilerArgs := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "renvo_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, compilerArgs, dir, source, signature, profile)
	plan.Composition = target.CH32V003()
	plan.BoardELF = target.ELFArtifactOptions{VectorSymbol: "renvo_vectors"}
	if _, err := RunPipeline(plan, nil); err == nil || !strings.Contains(err.Error(), "does not match composition") {
		t.Fatalf("toolchain/composition drift error = %v", err)
	}

	plan.Toolchain.ObjectFormat = plan.Composition.Object.Format.Name
	plan.Toolchain.ABI = plan.Composition.Object.ABI
	plan.BoardELF.VectorSymbol = "different_vectors"
	if _, err := RunPipeline(plan, nil); err == nil || !strings.Contains(err.Error(), "vector symbol") {
		t.Fatalf("vector/composition drift error = %v", err)
	}

	plan.BoardELF.VectorSymbol = plan.Composition.Board.Startup.VectorSymbol
	plan.ResultSymbol = "different_result"
	if _, err := RunPipeline(plan, nil); err == nil || !strings.Contains(err.Error(), "result symbol") {
		t.Fatalf("result/composition drift error = %v", err)
	}
}

type recordingRunner struct {
	commands []Command
}

func (r *recordingRunner) Run(command Command) ([]byte, error) {
	r.commands = append(r.commands, command)
	return (ExecCommandRunner{}).Run(command)
}

func testPipelinePlan(compiler string, compilerArgs []string, dir string, source string, signature uint64, profile uint32) PipelinePlan {
	referenceObject := filepath.Join(dir, "reference.o")
	candidateObject := filepath.Join(dir, "candidate.o")
	referenceImage := filepath.Join(dir, "reference")
	candidateImage := filepath.Join(dir, "candidate")
	digest, err := CanonicalInputSHA256(source)
	if err != nil {
		panic(err)
	}
	referenceBuildArgs := append([]string(nil), compilerArgs...)
	referenceBuildArgs = append(referenceBuildArgs, "-std=c89", source, "-c", "-o", referenceObject)
	candidateBuildArgs := append([]string(nil), compilerArgs...)
	candidateBuildArgs = append(candidateBuildArgs, "-std=c89", source, "-c", "-o", candidateObject)
	return PipelinePlan{
		Toolchain:         Toolchain{CCompiler: compiler, Linker: compiler, ObjectFormat: "elf", ABI: "host-test", TargetShell: "host-test"},
		Stage:             StandardStages("renvo")[0],
		CanonicalInput:    source,
		CanonicalSHA256:   digest,
		ReferenceBuild:    Command{Path: compiler, Args: referenceBuildArgs, CanonicalInput: source, CanonicalSHA256: digest},
		CandidateBuild:    Command{Path: compiler, Args: candidateBuildArgs, CanonicalInput: source, CanonicalSHA256: digest},
		ReferenceLink:     Command{Path: compiler, Args: []string{referenceObject, "-o", referenceImage}},
		CandidateLink:     Command{Path: compiler, Args: []string{candidateObject, "-o", candidateImage}},
		Reference:         PipelineArtifact{Object: referenceObject, Image: referenceImage, MemoryDump: filepath.Join(dir, "reference.bin"), MemoryAtResultSymbol: true},
		Candidate:         PipelineArtifact{Object: candidateObject, Image: candidateImage, MemoryDump: filepath.Join(dir, "candidate.bin"), MemoryAtResultSymbol: true},
		ExpectedProfile:   profile,
		ExpectedSignature: signature,
	}
}

func cCompilerForPipelineTest(t *testing.T) (string, []string) {
	t.Helper()
	return elfObjectCompiler(t)
}

func requireHostELFLinker(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("host ELF linker integration is Linux-specific")
	}
}

func writePipelineCSource(t *testing.T, path string, export string) {
	t.Helper()
	source := "unsigned char renvores[64];\nunsigned long renvo_vectors(void) { return 0UL; }\nunsigned long " + export + "(void) { return 37UL; }\nint main(void) { return 0; }\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writePassingPipelineResult(t *testing.T, path string, profile uint32) uint64 {
	t.Helper()
	block := resultabi.New(profile)
	block.BeginProbe(1)
	block.CompleteProbe(1, 37)
	signature := resultabi.MixSignature(0, 1, 37)
	block.Pass(signature)
	if err := os.WriteFile(path, block[:], 0o644); err != nil {
		t.Fatal(err)
	}
	return signature
}
