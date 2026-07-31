package rtg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeDefinitionEmbeddedGoBudget(t *testing.T) {
	document := parseDefinitionFile(t, "../../backend/definitions/native.rtg")
	source := document.Source
	resolved := ResolveDefinitions(document)
	if !resolved.Ok {
		t.Fatalf("ResolveDefinitions failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) == 0 {
		t.Fatal("native definition exports no targets")
	}
	metrics := MeasureTarget(resolved.Document, resolved.Targets[0])
	const maxBytes = 60000
	const maxDeclarations = 200
	if metrics.CatalogGoBytes > maxBytes || metrics.CatalogGoDecls > maxDeclarations {
		t.Fatalf("native embedded Go = %d bytes / %d declarations, limit %d / %d; "+
			"move regular machine, ABI, runtime, relocation, and format structure into "+
			"bounded declarations rather than hiding it in another hook",
			metrics.CatalogGoBytes, metrics.CatalogGoDecls, maxBytes, maxDeclarations)
	}
	for _, legacy := range []string{
		"x86KernelModuleImage", "aarch64WindowsRuntimeReadWrite",
		"x86PatchRelocations", "x86_32PatchRelocations",
		"aarch64PatchRelocations", "armPatchRelocations",
		"x86ELFPatchAbsoluteRelocations", "x86_32ELFPatchAbsoluteRelocations",
		"aarch64ELFPatchAbsoluteRelocations", "armELFPatchAbsoluteRelocations",
		"x86PEPatch", "x86_32PEPatch",
	} {
		if strings.Contains(string(source), legacy) {
			t.Errorf("legacy opaque backend algorithm remains: %s", legacy)
		}
	}
}

func TestAArch64DefinitionVerticalSlice(t *testing.T) {
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/native.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 9 {
		t.Fatalf("native target count = %d, want 9", len(resolved.Targets))
	}
	darwin, found := lookupResolvedTarget(resolved, "darwin/arm64")
	if !found {
		t.Fatal("native definition lost darwin/arm64")
	}
	if stateBytes, ok := integerField(
		resolved.Document, darwin.Runtime, "entry_state_bytes"); !ok || stateBytes != 24 {
		t.Fatalf("darwin/arm64 entry state bytes = %d, %v; want 24, true",
			stateBytes, ok)
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

func TestNativeCatalogTargetIdentityIgnoresUnreachableArchitecture(t *testing.T) {
	document := parseDefinitionFile(t, "../../backend/definitions/native.rtg")
	source := document.Source
	base := Resolve(document)
	if !base.Ok {
		t.Fatalf("base definition failed: %#v", base.Diagnostics)
	}
	ensureDirectEmitterV1()
	extra := "\narch unreachable_fixture {\n\tendian = little\n\tword_bits = 8\n\treject = [\n"
	for i := 0; i < len(directEmitterV1); i++ {
		extra += "\t\t" + directEmitterV1[i].Name + ",\n"
	}
	extra += "\t]\n}\n"
	extendedSource := append(append([]byte(nil), source...), []byte(extra)...)
	extended := Resolve(Parse(extendedSource, "native-extended.rtg"))
	if !extended.Ok {
		t.Fatalf("extended definition failed: %#v", extended.Diagnostics)
	}
	if base.Document.Hash == extended.Document.Hash {
		t.Fatal("unreachable architecture did not change catalog identity")
	}
	for i := 0; i < len(base.Targets); i++ {
		target, ok := lookupResolvedTarget(extended, base.Targets[i].Descriptor.Name)
		if !ok {
			t.Fatalf("extended catalog lost %s", base.Targets[i].Descriptor.Name)
		}
		if target.Descriptor.Definition != base.Targets[i].Descriptor.Definition {
			t.Fatalf("unreachable architecture changed %s identity", target.Descriptor.Name)
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
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/native.rtg"))
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
}

func TestAmd64DefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/native.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 9 {
		t.Fatalf("native target count = %d, want 9", len(resolved.Targets))
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
	prepared := GeneratePreparedBackend(resolved, "linux/amd64")
	if !prepared.Ok {
		t.Fatalf("generate prepared amd64 backend: %#v", prepared.Diagnostics)
	}
	if !containsText(string(prepared.Source), "func rtgNativeDirectMove(") {
		t.Error("prepared amd64 backend omitted the direct emitter contract")
	}
	if containsText(string(prepared.Source), directEmitterV1Schema) {
		t.Error("prepared amd64 backend retained the direct emitter schema as runtime data")
	}
}

func TestArmDefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/native.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if _, ok := lookupResolvedTarget(resolved, "linux/arm"); !ok {
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
}

func Test386DefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/native.rtg"))
	if !resolved.Ok {
		t.Fatalf("definition failed: %#v", resolved.Diagnostics)
	}
	if len(resolved.Targets) != 9 {
		t.Fatalf("native target count = %d, want 9", len(resolved.Targets))
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
}

func parseDefinitionFile(t *testing.T, filename string) Document {
	t.Helper()
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	document := ParseImports(source, filename, testFilesystemImportLoader{})
	if !document.Ok {
		t.Fatalf("parse %s: %#v", filename, document.Diagnostics)
	}
	return document
}

type testFilesystemImportLoader struct{}

func (testFilesystemImportLoader) LoadImport(
	importingFilename string, importPath string,
) ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return ImportSource{Source: source, Filename: path, Ok: err == nil}
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
