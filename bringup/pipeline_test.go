package bringup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"j5.nz/rtg/omnibus/resultabi"
)

func TestRunPipelineBuildsValidatesLinksAndComparesResults(t *testing.T) {
	compiler := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "rtg_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, dir, source, signature, profile)
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
	compiler := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	referenceSource := filepath.Join(dir, "reference.c")
	writePipelineCSource(t, referenceSource, "rtg_stage0")
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, dir, referenceSource, signature, profile)
	plan.CandidateBuild.Args = []string{"-std=c89", "-Drtg_stage0=wrong_export", referenceSource, "-c", "-o", plan.Candidate.Object}
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
	compiler := cCompilerForPipelineTest(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "omnibus.c")
	writePipelineCSource(t, source, "rtg_stage0")
	digest, err := CanonicalInputSHA256(source)
	if err != nil {
		t.Fatal(err)
	}
	profile := uint32(0x10001)
	signature := writePassingPipelineResult(t, filepath.Join(dir, "reference.bin"), profile)
	writePassingPipelineResult(t, filepath.Join(dir, "candidate.bin"), profile)

	plan := testPipelinePlan(compiler, dir, source, signature, profile)
	plan.CandidateBuild.CanonicalInput = filepath.Join(dir, "different.rtgu")
	if _, err := RunPipeline(plan, ExecCommandRunner{}); err == nil || !strings.Contains(err.Error(), "candidate build") {
		t.Fatalf("different candidate input error = %v", err)
	}

	plan = testPipelinePlan(compiler, dir, source, signature, profile)
	if err := os.WriteFile(source, []byte("changed after planning\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RunPipeline(plan, ExecCommandRunner{}); err == nil || !strings.Contains(err.Error(), digest) {
		t.Fatalf("changed canonical input error = %v", err)
	}
}

func testPipelinePlan(compiler string, dir string, source string, signature uint64, profile uint32) PipelinePlan {
	referenceObject := filepath.Join(dir, "reference.o")
	candidateObject := filepath.Join(dir, "candidate.o")
	referenceImage := filepath.Join(dir, "reference")
	candidateImage := filepath.Join(dir, "candidate")
	digest, err := CanonicalInputSHA256(source)
	if err != nil {
		panic(err)
	}
	return PipelinePlan{
		Toolchain:         Toolchain{CCompiler: compiler, Linker: compiler, ObjectFormat: "elf", ABI: "host-test", TargetShell: "host-test"},
		Stage:             StandardStages("rtg")[0],
		CanonicalInput:    source,
		CanonicalSHA256:   digest,
		ReferenceBuild:    Command{Path: compiler, Args: []string{"-std=c89", source, "-c", "-o", referenceObject}, CanonicalInput: source, CanonicalSHA256: digest},
		CandidateBuild:    Command{Path: compiler, Args: []string{"-std=c89", source, "-c", "-o", candidateObject}, CanonicalInput: source, CanonicalSHA256: digest},
		ReferenceLink:     Command{Path: compiler, Args: []string{referenceObject, "-o", referenceImage}},
		CandidateLink:     Command{Path: compiler, Args: []string{candidateObject, "-o", candidateImage}},
		Reference:         PipelineArtifact{Object: referenceObject, Image: referenceImage, MemoryDump: filepath.Join(dir, "reference.bin"), MemoryAtResultSymbol: true},
		Candidate:         PipelineArtifact{Object: candidateObject, Image: candidateImage, MemoryDump: filepath.Join(dir, "candidate.bin"), MemoryAtResultSymbol: true},
		ExpectedProfile:   profile,
		ExpectedSignature: signature,
	}
}

func cCompilerForPipelineTest(t *testing.T) string {
	t.Helper()
	for _, name := range []string{"cc", "gcc", "clang"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path
		}
	}
	t.Skip("C compiler not installed")
	return ""
}

func writePipelineCSource(t *testing.T, path string, export string) {
	t.Helper()
	source := "unsigned char rtgres[64];\nunsigned long " + export + "(void) { return 37UL; }\nint main(void) { return 0; }\n"
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
