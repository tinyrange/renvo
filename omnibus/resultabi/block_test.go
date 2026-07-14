package resultabi

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBlockFailureRetainsProbeAndValues(t *testing.T) {
	block := New(0x1234)
	block.BeginProbe(17)
	block.FailComparison(17, 0x1122334455667788, 0x8877665544332211)
	snapshot, err := Decode(block[:])
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != StateFailedComparison || snapshot.CurrentProbe != 17 || snapshot.FailureProbe != 17 {
		t.Fatalf("unexpected failure location: %+v", snapshot)
	}
	if snapshot.Expected != 0x1122334455667788 || snapshot.Observed != 0x8877665544332211 {
		t.Fatalf("comparison values were not retained: %+v", snapshot)
	}
}

func TestTrapRetainsLastEnteredProbe(t *testing.T) {
	block := New(7)
	block.BeginProbe(91)
	block.MarkTrap()
	snapshot, err := Decode(block[:])
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != StateTrapReset || snapshot.CurrentProbe != 91 || snapshot.CompletedProbes != 0 {
		t.Fatalf("trap lost location: %+v", snapshot)
	}
}

func TestInterruptedRunningProbeRetainsLastEnteredProbe(t *testing.T) {
	block := New(7)
	block.BeginProbe(92)
	// Model a debugger halting an infinite loop before the probe can commit a
	// result. No target-side failure path gets an opportunity to run.
	snapshot, err := Decode(block[:])
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.State != StateRunning || snapshot.CurrentProbe != 92 || snapshot.CompletedProbes != 0 {
		t.Fatalf("interrupted probe lost location: %+v", snapshot)
	}
}

func TestValidatePassRequiresProfileAndSignature(t *testing.T) {
	block := New(3)
	block.BeginProbe(1)
	block.CompleteProbe(1, 42)
	snapshot, err := Decode(block[:])
	if err != nil {
		t.Fatal(err)
	}
	block.Pass(snapshot.Signature)
	snapshot, err = Decode(block[:])
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.ValidatePass(3, snapshot.Signature); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.ValidatePass(4, snapshot.Signature); err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("profile mismatch error = %v", err)
	}
	if err := snapshot.ValidatePass(3, snapshot.Signature+1); err == nil || !strings.Contains(err.Error(), "signature") {
		t.Fatalf("signature mismatch error = %v", err)
	}
}

func TestLayoutIsSmallAndLittleEndian(t *testing.T) {
	if Size > 64 {
		t.Fatalf("result block grew to %d bytes", Size)
	}
	block := New(0x01020304)
	if got := binary.LittleEndian.Uint32(block[OffsetProfile:]); got != 0x01020304 {
		t.Fatalf("profile encoding = %#x", got)
	}
	if string(block[OffsetMagic:OffsetMagic+4]) != "RTGR" {
		t.Fatalf("magic bytes = %q", block[OffsetMagic:OffsetMagic+4])
	}
}

func TestDecodeRejectsTornOrForeignHeaders(t *testing.T) {
	block := New(1)
	block[OffsetVersion]++
	if _, err := Decode(block[:]); err == nil {
		t.Fatal("version mismatch accepted")
	}
	if _, err := Decode(make([]byte, Size-1)); err == nil {
		t.Fatal("short block accepted")
	}
}

func TestGeneratedDefinitionsAreCurrent(t *testing.T) {
	command := exec.Command("go", "run", "../../cmd/rtgresultgen", "-spec", "protocol.json", "-go", "layout_generated.go", "-c", "result_abi.h", "-check")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated definitions are stale: %v\n%s", err, output)
	}
}

func TestC89HeaderAndELFSymbol(t *testing.T) {
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("C compiler not installed")
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "result.c")
	object := filepath.Join(dir, "result.o")
	program := "#include \"result_abi.h\"\nunsigned char rtgres[RTG_RESULT_SIZE];\nint rtg_result_fixture(void) { return (int)RTG_RESULT_OFFSET_SIGNATURE; }\n"
	if err := os.WriteFile(source, []byte(program), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(compiler, "-std=c89", "-pedantic-errors", "-Wall", "-Werror", "-I.", "-c", source, "-o", object)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("strict C89 header compile: %v\n%s", err, output)
	}
	if runtime.GOOS != "linux" {
		return
	}
	address, err := ELFSymbolAddress(object, SymbolName)
	if err != nil {
		t.Fatal(err)
	}
	block := New(0x21)
	block.Pass(0x123456789abcdef0)
	memory := filepath.Join(dir, "memory.bin")
	if err := os.WriteFile(memory, block[:], 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := DecodeMemoryDump(object, memory, address, SymbolName)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.ValidatePass(0x21, 0x123456789abcdef0); err != nil {
		t.Fatal(err)
	}
}
