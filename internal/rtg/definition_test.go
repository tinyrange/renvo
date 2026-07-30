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
	generated := GenerateCheckedInArchitectureAlgorithms(resolved, "aarch64", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_aarch64_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in AArch64 architecture output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func renvoAarch64AsmRet(",
		"func renvoAarch64AsmAddRegImm(",
		"func renvoAarch64AsmMovRegImm(",
		"func renvoAarch64AsmJmpLabel(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated AArch64 output is missing algorithm binding %s", binding)
		}
	}
	contract := GenerateCheckedInArchitectureContract(resolved, "aarch64", "main")
	if !contract.Ok {
		t.Fatalf("generate AArch64 contract: %#v", contract.Diagnostics)
	}
	checkedContract, err := os.ReadFile("../../backend/rtg_aarch64_contract_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract.Source, checkedContract) {
		t.Fatal("checked-in AArch64 contract output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgAarch64DirectMove(",
		"func rtgAarch64DirectAddress(",
		"func rtgAarch64DirectLoadI32(",
		"func rtgAarch64DirectJumpCondition(",
		"func rtgAarch64DirectCopyBytes(",
		"func rtgAarch64PatchRelocations(",
	} {
		if !containsText(string(checkedContract), binding) {
			t.Errorf("generated AArch64 contract is missing semantic binding %s", binding)
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
	generated := GenerateCheckedInArchitectureAlgorithms(resolved, "x86_64", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_amd64_target_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in amd64 architecture output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func renvoAmd64AsmMovRaxImm(",
		"func renvoAmd64AsmLoadRaxMemRdxDisp(",
		"func renvoAmd64AsmCmpRaxImm8(",
		"func renvoAmd64AsmDivLeftRcxRightRax(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated amd64 output is missing algorithm binding %s", binding)
		}
	}
	contract := GenerateCheckedInArchitectureContract(resolved, "x86_64", "main")
	if !contract.Ok {
		t.Fatalf("generate amd64 contract: %#v", contract.Diagnostics)
	}
	checkedContract, err := os.ReadFile("../../backend/rtg_amd64_contract_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract.Source, checkedContract) {
		t.Fatal("checked-in amd64 contract output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgX8664DirectMove(",
		"func rtgX8664DirectLoadU8(",
		"func rtgX8664DirectJumpCondition(",
		"func rtgX8664DirectCopyBytes(",
		"func rtgX8664PatchRelocations(",
	} {
		if !containsText(string(checkedContract), binding) {
			t.Errorf("generated amd64 contract is missing semantic binding %s", binding)
		}
	}
	prepared := GeneratePreparedBackend(resolved, "linux/amd64")
	if !prepared.Ok {
		t.Fatalf("generate prepared amd64 backend: %#v", prepared.Diagnostics)
	}
	if !containsText(string(prepared.Source), "func rtgX8664DirectMove(") {
		t.Error("prepared amd64 backend omitted the direct emitter contract")
	}
	if containsText(string(prepared.Source), directEmitterV1Schema) {
		t.Error("prepared amd64 backend retained the direct emitter schema as runtime data")
	}
}

func TestArmDefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/arm.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "arm.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 || resolved.Targets[0].Descriptor.Name != "linux/arm" {
		t.Fatalf("targets = %#v, want linux/arm", resolved.Targets)
	}
	generated := GenerateCheckedInArchitectureAlgorithms(resolved, "arm", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_arm_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in ARM output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func renvoArmAsmMovRegImm(",
		"func renvoArmAsmLoadRegMem(",
		"func renvoArmAsmCallLabel(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated ARM output is missing algorithm binding %s", binding)
		}
	}
	contract := GenerateCheckedInArchitectureContract(resolved, "arm", "main")
	if !contract.Ok {
		t.Fatalf("generate ARM contract: %#v", contract.Diagnostics)
	}
	checkedContract, err := os.ReadFile("../../backend/rtg_arm_contract_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract.Source, checkedContract) {
		t.Fatal("checked-in ARM contract output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgArmDirectMove(",
		"func rtgArmDirectAddress(",
		"func rtgArmDirectLoadI8(",
		"func rtgArmDirectJumpCondition(",
		"func rtgArmDirectCopyBytes(",
		"func rtgArmPatchRelocations(",
	} {
		if !containsText(string(checkedContract), binding) {
			t.Errorf("generated ARM contract is missing semantic binding %s", binding)
		}
	}
}

func Test386DefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/386.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "386.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 2 {
		t.Fatalf("target count = %d, want 2", len(resolved.Targets))
	}
	generated := GenerateCheckedInArchitectureAlgorithms(resolved, "x86_32", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_386_target_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in 386 output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func renvo386AsmMovRaxImm(",
		"func renvo386AsmLoadRaxMemRdxDisp(",
		"func renvoAsmMovArg1Rax(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated 386 output is missing algorithm binding %s", binding)
		}
	}
	contract := GenerateCheckedInArchitectureContract(resolved, "x86_32", "main")
	if !contract.Ok {
		t.Fatalf("generate 386 contract: %#v", contract.Diagnostics)
	}
	checkedContract, err := os.ReadFile("../../backend/rtg_386_contract_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract.Source, checkedContract) {
		t.Fatal("checked-in 386 contract output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgX8632DirectMove(",
		"func rtgX8632DirectAddress(",
		"func rtgX8632DirectLoadI16(",
		"func rtgX8632DirectJumpCondition(",
		"func rtgX8632DirectCopyBytes(",
		"func rtgX8632PatchRelocations(",
	} {
		if !containsText(string(checkedContract), binding) {
			t.Errorf("generated 386 contract is missing semantic binding %s", binding)
		}
	}
}

func TestWasm32DefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	source, err := os.ReadFile("../../backend/definitions/wasm32.rtg")
	if err != nil {
		t.Fatal(err)
	}
	resolved := Resolve(Parse(source, "wasm32.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 3 {
		t.Fatalf("target count = %d, want wasi/wasm32, browser/wasm32, and vm/vm32", len(resolved.Targets))
	}
	for _, target := range resolved.Targets {
		wantISA := "wasm32"
		if target.Descriptor.Name == "vm/vm32" {
			wantISA = "vm32"
		}
		if target.Descriptor.ISA != wantISA {
			t.Errorf("%s frontend architecture = %q, want %q",
				target.Descriptor.Name, target.Descriptor.ISA, wantISA)
		}
		if target.Arch.Name != "vm32" {
			t.Errorf("%s emitter architecture = %q, want vm32",
				target.Descriptor.Name, target.Arch.Name)
		}
	}
	generated := GenerateCheckedInArchitectureAlgorithms(resolved, "vm32", "main")
	if !generated.Ok {
		t.Fatalf("generate architecture: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_wasm32_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in wasm32 output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func renvoWasm32AsmMovRaxImm(",
		"func renvoWasm32Image(",
		"func renvoVMImage(",
		"func renvoWasiWasm32ImportSection(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated VM32 output is missing algorithm binding %s", binding)
		}
	}
	contract := GenerateCheckedInArchitectureContract(resolved, "vm32", "main")
	if !contract.Ok {
		t.Fatalf("generate VM32 contract: %#v", contract.Diagnostics)
	}
	checkedContract, err := os.ReadFile("../../backend/rtg_vm32_contract_generated.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract.Source, checkedContract) {
		t.Fatal("checked-in VM32 contract output is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"func rtgWasm32DirectMove(",
		"func rtgWasm32DirectAddress(",
		"func rtgWasm32DirectLoadI8(",
		"func rtgWasm32DirectJumpCondition(",
		"func rtgWasm32DirectCopyBytes(",
		"func rtgWasm32PatchRelocations(",
	} {
		if !containsText(string(checkedContract), binding) {
			t.Errorf("generated VM32 contract is missing semantic binding %s", binding)
		}
	}
	if containsText(string(checkedContract), "func rtgWasm32DirectCallIndirect(") {
		t.Error("vm32 contract generated explicitly rejected call_indirect operation")
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
	inactive := GenerateInactiveArchitectureKernel("main")
	checkedInactive, err := os.ReadFile("../../backend/compiler_rtg_inactive_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inactive.Source, checkedInactive) {
		t.Fatal("checked-in inactive RTG architecture kernel is stale; run go generate ./backend/definitions")
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
