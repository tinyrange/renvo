package rtg

import (
	"bytes"
	"os"
	"testing"
)

func TestAArch64DefinitionVerticalSlice(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/aarch64.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "aarch64.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 3 {
		t.Fatalf("target count = %d, want 3", len(resolved.Targets))
	}
	for _, target := range []string{"linux/aarch64", "darwin/arm64", "windows/arm64"} {
		generated := GenerateFixedBackend(resolved, target)
		if !generated.Ok {
			t.Fatalf("generate %s: %#v", target, generated.Diagnostics)
		}
		if containsText(string(generated.Source), "RTGLookupTarget") {
			t.Errorf("fixed %s output retained universal dispatch", target)
		}
		if target == "linux/aarch64" {
			golden, err := os.ReadFile("testdata/aarch64_linux.golden")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(generated.Source, golden) {
				t.Fatal("linux/aarch64 generated backend is stale; run rtggen")
			}
		}
	}
}

func TestAArch64EncodingSlice(t *testing.T) {
	if got := encodeAArch64RET(30); got != 0xd65f03c0 {
		t.Fatalf("RET x30 = %#08x", got)
	}
	if got := encodeAArch64ADDImmediate(0, 1, 7); got != 0x91001c20 {
		t.Fatalf("ADD x0, x1, #7 = %#08x", got)
	}
	if got := encodeAArch64MOVZ(0, 0x1234, 16); got != 0xd2a24680 {
		t.Fatalf("MOVZ x0, #0x1234, LSL #16 = %#08x", got)
	}
	if got := encodeAArch64B(8); got != 0x14000002 {
		t.Fatalf("B +8 = %#08x", got)
	}
}

func TestAArch64CheckedInArchitectureOutput(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/aarch64.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "aarch64.rtg"))
	generated := GenerateArchitectureBackend(resolved, "aarch64", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_aarch64_generated_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in AArch64 architecture output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgAarch64Ret(",
		"func rtgAarch64AddImmediate(",
		"func rtgAarch64MoveWideZero(",
		"func rtgAarch64Branch(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated AArch64 output is missing direct binding %s", binding)
		}
	}
}

func TestAmd64DefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/amd64.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "amd64.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 3 {
		t.Fatalf("target count = %d, want 3", len(resolved.Targets))
	}
	generated := GenerateStatefulArchitectureBackend(resolved, "x86_64", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_amd64_generated_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in amd64 architecture output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgX8664Mov64(",
		"func rtgX8664Load64(",
		"func rtgX8664CallRel32(",
		"func rtgX8664PushRegister(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated amd64 output is missing direct binding %s", binding)
		}
	}
}

func TestCheckedInArchitectureKernelOutput(t *testing.T) {
	generated := GenerateArchitectureKernel("main")
	checkedIn, err := os.ReadFile("../../backend/compiler_rtg_generated_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in RTG architecture kernel is stale; run go generate ./backend/definitions")
	}
}

// These mirrors make the expected instruction words reviewable independently
// from generated-source spelling. The generated functions use the same direct
// expressions and are syntax-checked by TestAArch64DefinitionVerticalSlice.
func encodeAArch64RET(register uint32) uint32 {
	return 0xd65f0000 | (register&31)<<5
}

func encodeAArch64ADDImmediate(destination uint32, source uint32, value uint32) uint32 {
	return 0x91000000 | (value&0xfff)<<10 | (source&31)<<5 | destination&31
}

func encodeAArch64MOVZ(destination uint32, value uint32, shift uint32) uint32 {
	return 0xd2800000 | ((shift/16)&3)<<21 | (value&0xffff)<<5 | destination&31
}

func encodeAArch64B(displacement int32) uint32 {
	return 0x14000000 | uint32(displacement>>2)&0x03ffffff
}
