package rtg

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var nativeDefinitionEntrypoints = map[string]string{
	"darwin/arm64":       "../../backend/definitions/darwin_aarch64.rtg",
	"freebsd/amd64":      "../../backend/definitions/freebsd_amd64.rtg",
	"linux/386":          "../../backend/definitions/linux_386.rtg",
	"linux/aarch64":      "../../backend/definitions/linux_aarch64.rtg",
	"linux/amd64":        "../../backend/definitions/linux_amd64.rtg",
	"linux/arm":          "../../backend/definitions/linux_arm.rtg",
	"linux-kernel/amd64": "../../backend/definitions/linux_kernel_amd64.rtg",
	"netbsd/amd64":       "../../backend/definitions/netbsd_amd64.rtg",
	"openbsd/amd64":      "../../backend/definitions/openbsd_amd64.rtg",
	"windows/386":        "../../backend/definitions/windows_386.rtg",
	"windows/amd64":      "../../backend/definitions/windows_amd64.rtg",
	"windows/arm64":      "../../backend/definitions/windows_aarch64.rtg",
}

var nativeArchitectureProjections = map[string]string{
	"aarch64": "../../backend/definitions/aarch64_algorithms.rtg",
	"arm":     "../../backend/definitions/arm_algorithms.rtg",
	"x86_32":  "../../backend/definitions/x86_32_algorithms.rtg",
	"x86_64":  "../../backend/definitions/x86_64_algorithms.rtg",
}

var nativeCompilerIntegrations = map[string]struct{ definition, output string }{
	"aarch64": {"../../backend/definitions/aarch64_compiler.rtg", "../../backend/compiler_aarch64_target_impl.go"},
	"x86_32":  {"../../backend/definitions/x86_32_compiler.rtg", "../../backend/compiler_386_impl.go"},
	"x86_64":  {"../../backend/definitions/x86_64_compiler.rtg", "../../backend/compiler_amd64_impl.go"},
}

func TestNativeCompilerIntegrationsAreGeneratedFromRTG(t *testing.T) {
	for arch, integration := range nativeCompilerIntegrations {
		t.Run(arch, func(t *testing.T) {
			resolved := ResolveArchitectureDefinition(parseDefinitionFile(t, integration.definition))
			if !resolved.Ok {
				t.Fatalf("resolve compiler integration: %#v", resolved.Diagnostics)
			}
			generated := GenerateCheckedInCompilerIntegration(resolved, arch, "main")
			if !generated.Ok {
				t.Fatalf("generate compiler integration: %#v", generated.Diagnostics)
			}
			checkedIn, err := os.ReadFile(integration.output)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(generated.Source, checkedIn) {
				t.Fatalf("checked-in %s compiler integration is stale; run go generate ./backend/definitions", arch)
			}
		})
	}
}

func TestNativeDefinitionEmbeddedGoMetrics(t *testing.T) {
	type authoredDeclaration struct {
		bytes int
	}
	authored := make(map[string]authoredDeclaration)
	var source []byte
	for _, filename := range sortedNativeEntrypoints() {
		resolved := ResolveDefinitions(parseDefinitionFile(t, filename))
		if !resolved.Ok {
			t.Fatalf("ResolveDefinitions(%s) failed: %#v",
				filename, resolved.Diagnostics)
		}
		if len(resolved.Targets) != 1 {
			t.Fatalf("%s exports %d targets, want exactly one",
				filename, len(resolved.Targets))
		}
		source = append(source, resolved.Document.Source...)
		parts := reachableEmbeddedGoParts(
			resolved.Document, embeddedGoNames(resolved.Document), nil)
		for i := 0; i < len(parts); i++ {
			body := authoredVirtualSource(resolved.Document, parts[i].source)
			key := parts[i].name + "\x00" + string(body)
			authored[key] = authoredDeclaration{bytes: len(body)}
		}
	}
	var goBytes int
	for _, declaration := range authored {
		goBytes += declaration.bytes
	}
	t.Logf("deduplicated native embedded Go = %d bytes / %d declarations",
		goBytes, len(authored))
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

func TestNativeBuiltInAndPreparedProjectionsShareResolvedSemantics(t *testing.T) {
	for _, targetName := range sortedNativeTargetNames() {
		t.Run(strings.ReplaceAll(targetName, "/", "-"), func(t *testing.T) {
			resolved := resolveNativeTarget(t, targetName)
			builtIn := GenerateCheckedInTargetProjection(resolved, targetName, "main")
			if !builtIn.Ok {
				t.Fatalf("generate checked-in projection: %#v", builtIn.Diagnostics)
			}
			prepared := GeneratePreparedBackend(resolved, targetName)
			if !prepared.Ok {
				t.Fatalf("generate prepared projection: %#v", prepared.Diagnostics)
			}
			if !reflect.DeepEqual(builtIn.Descriptor, prepared.Descriptor) {
				t.Fatalf("descriptor mismatch:\nchecked-in: %#v\nprepared:   %#v",
					builtIn.Descriptor, prepared.Descriptor)
			}
			if len(builtIn.Manifest) != 1 || len(prepared.Manifest) != 1 ||
				builtIn.Manifest[0] != prepared.Manifest[0] {
				t.Fatalf("resolved-definition manifest mismatch:\nchecked-in: %#v\nprepared:   %#v",
					builtIn.Manifest, prepared.Manifest)
			}
			if bytes.Equal(builtIn.Source, prepared.Source) {
				t.Fatal("checked-in and prepared projections unexpectedly use the same packaging")
			}
			if containsText(string(builtIn.Source), "const renvoRTGPreparedOS") ||
				!containsText(string(prepared.Source), "const renvoRTGPreparedOS") ||
				containsText(string(builtIn.Source), "func rtgNativeDirectMove(") ||
				!containsText(string(prepared.Source), "func rtgNativeDirectMove(") {
				t.Fatal("machine semantics did not retain the direct/prepared packaging boundary")
			}
		})
	}
}

func TestCompactRuntimeOperationLists(t *testing.T) {
	resolved := resolveNativeTarget(t, "linux/386")
	target, found := lookupResolvedTarget(resolved, "linux/386")
	if !found {
		t.Fatal("linux/386 definition lost its target")
	}
	want := []string{"fd", "buffer", "count"}
	if got := runtimeOperationList(target.Runtime, "write", "args"); !reflect.DeepEqual(got, want) {
		t.Fatalf("compact write args = %#v, want %#v", got, want)
	}
	prepared := GeneratePreparedBackend(resolved, "linux/386")
	if !prepared.Ok {
		t.Fatalf("generate prepared linux/386: %#v", prepared.Diagnostics)
	}
	for _, move := range []string{
		"renvoRTGAsmPushRegister(out, rtgNativeESI)",
		"renvoRTGAsmPopRegister(out, rtgNativeECX)",
	} {
		if !containsText(string(prepared.Source), move) {
			t.Errorf("prepared linux/386 runtime omitted %s", move)
		}
	}
}

func TestAArch64DefinitionVerticalSlice(t *testing.T) {
	darwinResolved := resolveNativeTarget(t, "darwin/arm64")
	darwin, found := lookupResolvedTarget(darwinResolved, "darwin/arm64")
	if !found {
		t.Fatal("Darwin entrypoint lost darwin/arm64")
	}
	if stateBytes, ok := integerField(
		darwinResolved.Document, darwin.Runtime, "entry_state_bytes"); !ok || stateBytes != 24 {
		t.Fatalf("darwin/arm64 entry state bytes = %d, %v; want 24, true",
			stateBytes, ok)
	}
	prepared := GeneratePreparedBackend(darwinResolved, "darwin/arm64")
	if !prepared.Ok {
		t.Fatalf("generate prepared darwin/arm64: %#v", prepared.Diagnostics)
	}
	for _, mapping := range []string{
		"var renvoRTGCallWord0 = rtgNativeX3",
		"var renvoRTGCallWord1 = rtgNativeX4",
		"var renvoRTGCallWord2 = rtgNativeX1",
	} {
		if !containsText(string(prepared.Source), mapping) {
			t.Errorf("prepared darwin/arm64 omitted internal call-word mapping %q", mapping)
		}
	}
	for _, target := range []string{"linux/aarch64", "darwin/arm64", "windows/arm64"} {
		resolved := resolveNativeTarget(t, target)
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
	const filename = "../../backend/definitions/linux_amd64.rtg"
	document := parseDefinitionFile(t, filename)
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
	root, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	extendedSource := append(append([]byte(nil), root...), []byte(extra)...)
	extended := Resolve(ParseImports(
		extendedSource, filename, testFilesystemImportLoader{}))
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
	resolved := resolveNativeArchitectureProjection(t, "aarch64")
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
	resolved := resolveNativeArchitectureProjection(t, "x86_64")
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
		"renvoFixedTarget == 0 && a.c.optimizeRuntime",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated amd64 output is missing algorithm binding %s", binding)
		}
	}
	targetProjection := GenerateCheckedInTargetProjection(
		resolveNativeTarget(t, "linux/amd64"), "linux/amd64", "main")
	if !targetProjection.Ok {
		t.Fatalf("generate linux/amd64 target projection: %#v", targetProjection.Diagnostics)
	}
	checkedInTarget, err := os.ReadFile("../../backend/compiler_linux_amd64_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(targetProjection.Source, checkedInTarget) {
		t.Fatal("checked-in linux/amd64 target projection is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"target: target-projection/linux/amd64",
		"const renvoLinuxAmd64SysReadSeq = 0",
		"const renvoLinuxAmd64ArgsBSSSize = 32768",
		"const renvoLinuxAmd64EnvironmentBSSAlignment = 16",
		"func renvoAsmBuildArgvEnvSlicesAmd64(",
		"func renvoAsmImageAmd64(",
		"const renvoLinuxAmd64ELFMachine = 62",
	} {
		if !containsText(string(checkedInTarget), binding) {
			t.Errorf("generated linux/amd64 target output is missing %s", binding)
		}
	}
	for _, forbidden := range []string{"type RTGEmitter struct", "func renvoRTGImage("} {
		if containsText(string(checkedInTarget), forbidden) {
			t.Errorf("generated linux/amd64 target output retained prepared bridge %s", forbidden)
		}
	}
	prepared := GeneratePreparedBackend(
		resolveNativeTarget(t, "linux/amd64"), "linux/amd64")
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

func TestCheckedInLinuxAmd64ProjectionOwnsEntryBSSLayout(t *testing.T) {
	const filename = "../../backend/definitions/linux_amd64.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(original,
		[]byte("bss args 32768 16"), []byte("bss args 16384 32"), 1)
	if bytes.Equal(modified, original) {
		t.Fatal("fixture did not contain the args BSS declaration")
	}
	resolved := ResolveDefinitions(ParseImports(
		modified, filename, testFilesystemImportLoader{}))
	if !resolved.Ok {
		t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
	}
	generated := GenerateCheckedInTargetProjection(resolved, "linux/amd64", "main")
	if !generated.Ok {
		t.Fatalf("modified projection failed: %#v", generated.Diagnostics)
	}
	for _, binding := range []string{
		"const renvoLinuxAmd64ArgsBSSSize = 16384",
		"const renvoLinuxAmd64ArgsBSSAlignment = 32",
	} {
		if !containsText(string(generated.Source), binding) {
			t.Errorf("modified projection is missing %s", binding)
		}
	}
}

func TestCheckedInLinuxAmd64ProjectionRejectsIncompleteDefinition(t *testing.T) {
	const filename = "../../backend/definitions/linux_amd64.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{
			name: "missing syscall number",
			old:  "operation read { number = 0 args = [fd, buffer, count] }",
			new:  "operation read { args = [fd, buffer, count] }",
		},
		{
			name: "incompatible entry arity",
			old:  "max_parameters = 2",
			new:  "max_parameters = 1",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modified := bytes.Replace(original, []byte(test.old), []byte(test.new), 1)
			if bytes.Equal(modified, original) {
				t.Fatalf("fixture did not contain %q", test.old)
			}
			resolved := ResolveDefinitions(ParseImports(
				modified, filename, testFilesystemImportLoader{}))
			if !resolved.Ok {
				t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
			}
			generated := GenerateCheckedInTargetProjection(
				resolved, "linux/amd64", "main")
			if generated.Ok || len(generated.Diagnostics) != 1 ||
				generated.Diagnostics[0].Code != "RTG-GENERATE-006" {
				t.Fatalf("incomplete projection result = %#v", generated)
			}
		})
	}
}

func TestCheckedInWindowsAmd64ProjectionOutput(t *testing.T) {
	resolved := resolveNativeTarget(t, "windows/amd64")
	generated := GenerateCheckedInTargetProjection(resolved, "windows/amd64", "main")
	if !generated.Ok {
		t.Fatalf("generate windows/amd64 target projection: %#v", generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile("../../backend/compiler_windows_amd64_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in windows/amd64 target projection is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"target: target-projection/windows/amd64",
		"const renvoWindowsAmd64ArgsBSSSize = 32768",
		"func renvoWinAmd64CallImport(",
		"func renvoWinAmd64EmitReadWriteHelper(",
		"func renvoAsmBuildWindowsArgvEnvSlicesAmd64(",
		"func renvoAsmImageWindowsAmd64(",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated windows/amd64 target output is missing %s", binding)
		}
	}
	for _, forbidden := range []string{
		"type RTGEmitter struct", "func renvoRTGImage(", "rtgBuiltinX8664",
	} {
		if containsText(string(checkedIn), forbidden) {
			t.Errorf("generated windows/amd64 target retained pruned bridge %s", forbidden)
		}
	}
}

func TestCheckedInWindowsAmd64ProjectionOwnsEntryBSSLayout(t *testing.T) {
	const filename = "../../backend/definitions/windows_amd64.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(original,
		[]byte("bss args 32768 16"), []byte("bss args 16384 32"), 1)
	resolved := ResolveDefinitions(ParseImports(
		modified, filename, testFilesystemImportLoader{}))
	if !resolved.Ok {
		t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
	}
	generated := GenerateCheckedInTargetProjection(resolved, "windows/amd64", "main")
	if !generated.Ok {
		t.Fatalf("modified projection failed: %#v", generated.Diagnostics)
	}
	for _, binding := range []string{
		"const renvoWindowsAmd64ArgsBSSSize = 16384",
		"const renvoWindowsAmd64ArgsBSSAlignment = 32",
	} {
		if !containsText(string(generated.Source), binding) {
			t.Errorf("modified projection is missing %s", binding)
		}
	}
}

func TestCheckedInWindowsAmd64ProjectionRejectsIncompatibleDefinition(t *testing.T) {
	const filename = "../../backend/definitions/windows_amd64.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct{ old, new string }{
		{`stack_commit = 0x1000`, `stack_commit = 0x2000`},
		{`GetCommandLineA,`, `GetCommandLineW,`},
		{`max_parameters = 2`, `max_parameters = 1`},
	}
	for _, test := range tests {
		modified := bytes.Replace(original, []byte(test.old), []byte(test.new), 1)
		resolved := ResolveDefinitions(ParseImports(
			modified, filename, testFilesystemImportLoader{}))
		if !resolved.Ok {
			continue
		}
		generated := GenerateCheckedInTargetProjection(resolved, "windows/amd64", "main")
		if generated.Ok || len(generated.Diagnostics) != 1 ||
			generated.Diagnostics[0].Code != "RTG-GENERATE-006" {
			t.Fatalf("incompatible projection result = %#v", generated)
		}
	}
}

func TestCheckedInLinuxKernelAmd64ProjectionOutput(t *testing.T) {
	resolved := resolveNativeTarget(t, "linux-kernel/amd64")
	generated := GenerateCheckedInTargetProjection(
		resolved, "linux-kernel/amd64", "main")
	if !generated.Ok {
		t.Fatalf("generate linux-kernel/amd64 target projection: %#v",
			generated.Diagnostics)
	}
	checkedIn, err := os.ReadFile(
		"../../backend/compiler_linux_kernel_amd64_target_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in linux-kernel/amd64 target projection is stale; run go generate ./backend/definitions")
	}
	for _, binding := range []string{
		"target: target-projection/linux-kernel/amd64",
		"func renvoAmd64KernelEntryPrologue(",
		"func renvoAmd64KernelCallbackAddress(",
		"func renvoAmd64EmitKernelPrintValue(",
		"func renvoAmd64EmitKernelStaticCall(",
		"func renvoAsmImageKernelModuleAmd64(",
		"Generated from the declarative Linux-module ELF format",
	} {
		if !containsText(string(checkedIn), binding) {
			t.Errorf("generated linux-kernel/amd64 target output is missing %s", binding)
		}
	}
	for _, forbidden := range []string{
		"type RTGEmitter struct", "func renvoRTGImage(", "rtgBuiltinX8664",
		"RTGRegister", "renvoRTGAddress", ".KernelBTF()",
	} {
		if containsText(string(checkedIn), forbidden) {
			t.Errorf("generated linux-kernel/amd64 target retained pruned bridge %s", forbidden)
		}
	}
}

func TestCheckedInLinuxKernelAmd64ProjectionOwnsRuntimeAndEntry(t *testing.T) {
	const filename = "../../backend/definitions/linux_kernel_amd64.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(original,
		[]byte(`out.Text("\xf3\x0f\x1e\xfa`),
		[]byte(`out.Text("\x90\x0f\x1e\xfa`), 1)
	if bytes.Equal(modified, original) {
		t.Fatal("fixture did not contain the kernel entry sequence")
	}
	resolved := ResolveDefinitions(ParseImports(
		modified, filename, testFilesystemImportLoader{}))
	if !resolved.Ok {
		t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
	}
	generated := GenerateCheckedInTargetProjection(
		resolved, "linux-kernel/amd64", "main")
	if !generated.Ok {
		t.Fatalf("modified projection failed: %#v", generated.Diagnostics)
	}
	if !containsText(string(generated.Source), `"\x90\x0f\x1e\xfa`) {
		t.Fatal("modified kernel entry sequence did not reach the target projection")
	}
}

func TestArmDefinitionAndCheckedInArchitectureOutput(t *testing.T) {
	resolveNativeTarget(t, "linux/arm")
	resolved := resolveNativeArchitectureProjection(t, "arm")
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
	resolved := resolveNativeArchitectureProjection(t, "x86_32")
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

func TestCheckedIn386TargetProjectionOutput(t *testing.T) {
	tests := []struct {
		target string
		file   string
		want   []string
	}{
		{
			target: "linux/386",
			file:   "../../backend/compiler_linux_386_impl.go",
			want: []string{
				"target: target-projection/linux/386",
				"const renvoLinux386SysReadSeq = 3",
				"func rtgBuiltinLinux386PackageLinuxEntry(",
				"func renvoAsmImage386(",
			},
		},
		{
			target: "windows/386",
			file:   "../../backend/compiler_windows_386_impl.go",
			want: []string{
				"target: target-projection/windows/386",
				"func renvoWin386EmitRuntimeOpen(",
				"func renvoAsmBuildWindowsArgvEnvSlices386(",
				"func renvoAsmImageWindows386(",
			},
		},
	}
	for _, test := range tests {
		t.Run(strings.ReplaceAll(test.target, "/", "-"), func(t *testing.T) {
			resolved := resolveNativeTarget(t, test.target)
			generated := GenerateCheckedInTargetProjection(resolved, test.target, "main")
			if !generated.Ok {
				t.Fatalf("generate %s target projection: %#v", test.target, generated.Diagnostics)
			}
			checkedIn, err := os.ReadFile(test.file)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(generated.Source, checkedIn) {
				t.Fatalf("checked-in %s target projection is stale; run go generate ./backend/definitions", test.target)
			}
			for _, binding := range test.want {
				if !containsText(string(checkedIn), binding) {
					t.Errorf("generated %s target output is missing %s", test.target, binding)
				}
			}
			for _, forbidden := range []string{"type RTGEmitter struct", "func renvoRTGImage("} {
				if containsText(string(checkedIn), forbidden) {
					t.Errorf("generated %s target retained prepared bridge %s", test.target, forbidden)
				}
			}
		})
	}
}

func TestCheckedInNativeTargetProjectionOwnsDefinitionSemantics(t *testing.T) {
	tests := []struct {
		filename string
		target   string
		old      string
		new      string
		want     string
	}{
		{
			filename: "../../backend/definitions/linux_386.rtg",
			target:   "linux/386",
			old:      "operation read { number = 3",
			new:      "operation read { number = 13",
			want:     "const renvoLinux386SysReadSeq = 13",
		},
		{
			filename: "../../backend/definitions/windows_386.rtg",
			target:   "windows/386",
			old:      "bss args 32768 16",
			new:      "bss args 16384 32",
			want:     "const renvoWindows386ArgsBSSSize = 16384",
		},
		{
			filename: "../../backend/definitions/linux_aarch64.rtg",
			target:   "linux/aarch64",
			old:      "operation read { number = 63",
			new:      "operation read { number = 163",
			want:     "const renvoLinuxAarch64SysReadSeq = 163",
		},
		{
			filename: "../../backend/definitions/windows_aarch64.rtg",
			target:   "windows/arm64",
			old:      "bss args 32768 16",
			new:      "bss args 16384 32",
			want:     "const renvoWindowsArm64ArgsBSSSize = 16384",
		},
		{
			filename: "../../backend/definitions/darwin_aarch64.rtg",
			target:   "darwin/arm64",
			old:      "bss args 32768 16",
			new:      "bss args 16384 32",
			want:     "const renvoDarwinArm64ArgsBSSSize = 16384",
		},
		{
			filename: "../../backend/definitions/linux_arm.rtg",
			target:   "linux/arm",
			old:      "operation read { number = 3",
			new:      "operation read { number = 13",
			want:     "const renvoLinuxArmSysReadSeq = 13",
		},
		{
			filename: "../../backend/definitions/linux_arm.rtg",
			target:   "linux/arm",
			old:      "operation exit { number = 1",
			new:      "operation exit { number = 11",
			want:     "const renvoLinuxArmSysExit = 11",
		},
		{
			filename: "../../backend/definitions/linux_arm.rtg",
			target:   "linux/arm",
			old:      "flags = 0x05000000",
			new:      "flags = 0x04000000",
			want:     "const renvoLinuxArmELFFlags = 67108864",
		},
	}
	for _, test := range tests {
		original, err := os.ReadFile(test.filename)
		if err != nil {
			t.Fatal(err)
		}
		modified := bytes.Replace(original, []byte(test.old), []byte(test.new), 1)
		if bytes.Equal(modified, original) {
			t.Fatalf("fixture %s did not contain %q", test.filename, test.old)
		}
		resolved := ResolveDefinitions(ParseImports(
			modified, test.filename, testFilesystemImportLoader{}))
		if !resolved.Ok {
			t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
		}
		generated := GenerateCheckedInTargetProjection(resolved, test.target, "main")
		if !generated.Ok {
			t.Fatalf("modified projection failed: %#v", generated.Diagnostics)
		}
		if !containsText(string(generated.Source), test.want) {
			t.Errorf("modified %s projection is missing %s", test.target, test.want)
		}
	}
}

func TestCheckedInWindows386DefinitionChangesInvalidateProjection(t *testing.T) {
	const filename = "../../backend/definitions/windows_386.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(original,
		[]byte("out.Int32(0x80)"), []byte("out.Int32(0x81)"), 1)
	if bytes.Equal(modified, original) {
		t.Fatal("Windows/386 runtime mutation did not change the fixture")
	}
	resolved := ResolveDefinitions(ParseImports(
		modified, filename, testFilesystemImportLoader{}))
	if !resolved.Ok {
		t.Fatalf("modified definition did not resolve: %#v", resolved.Diagnostics)
	}
	generated := GenerateCheckedInTargetProjection(resolved, "windows/386", "main")
	if !generated.Ok {
		t.Fatalf("modified runtime generation failed: %#v", generated.Diagnostics)
	}
	baseline := GenerateCheckedInTargetProjection(
		resolveNativeTarget(t, "windows/386"), "windows/386", "main")
	if !baseline.Ok {
		t.Fatalf("baseline runtime generation failed: %#v", baseline.Diagnostics)
	}
	if bytes.Equal(generated.Source, baseline.Source) {
		t.Fatal("modified RTG runtime did not invalidate the checked-in projection")
	}
}

func TestWindows386RuntimeSequenceChangesTargetIdentity(t *testing.T) {
	const filename = "../../backend/definitions/windows_386.rtg"
	original, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(original,
		[]byte("out.Bytes2(0x6a, 1)"), []byte("out.Bytes2(0x6a, 2)"), 1)
	if bytes.Equal(modified, original) {
		t.Fatal("Windows/386 runtime sequence mutation did not change the fixture")
	}
	base := ResolveDefinitions(ParseImports(
		original, filename, testFilesystemImportLoader{}))
	changed := ResolveDefinitions(ParseImports(
		modified, filename, testFilesystemImportLoader{}))
	if !base.Ok || !changed.Ok {
		t.Fatalf("resolve Windows/386 identities: base=%#v changed=%#v",
			base.Diagnostics, changed.Diagnostics)
	}
	if base.Targets[0].Descriptor.Definition == changed.Targets[0].Descriptor.Definition {
		t.Fatal("runtime sequence behavior did not change the target identity")
	}
}

func TestCheckedInRemainingNativeTargetProjectionOutput(t *testing.T) {
	tests := []struct{ target, file, binding string }{
		{"linux/aarch64", "../../backend/compiler_linux_aarch64_impl.go", "func rtgBuiltinLinuxAarch64PackageLinuxEntry("},
		{"windows/arm64", "../../backend/compiler_windows_arm64_target_impl.go", "func rtgBuiltinWindowsAarch64PackageWindowsRuntimeOpen("},
		{"darwin/arm64", "../../backend/compiler_darwin_arm64_target_impl.go", "func rtgBuiltinDarwinAarch64PackageMachImage("},
		{"linux/arm", "../../backend/compiler_linux_arm_impl.go", "const renvoLinuxArmSysReadSeq = 3"},
	}
	for _, test := range tests {
		t.Run(strings.ReplaceAll(test.target, "/", "-"), func(t *testing.T) {
			generated := GenerateCheckedInTargetProjection(resolveNativeTarget(t, test.target), test.target, "main")
			if !generated.Ok {
				t.Fatalf("generate target projection: %#v", generated.Diagnostics)
			}
			checkedIn, err := os.ReadFile(test.file)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(generated.Source, checkedIn) {
				t.Fatalf("checked-in %s projection is stale; run go generate ./backend/definitions", test.target)
			}
			if !containsText(string(checkedIn), test.binding) {
				t.Errorf("projection is missing %s", test.binding)
			}
			for _, forbidden := range []string{"type RTGEmitter struct", "func renvoRTGImage("} {
				if containsText(string(checkedIn), forbidden) {
					t.Errorf("projection retained %s", forbidden)
				}
			}
		})
	}
}

func TestDarwinAArch64ProjectionUsesPlatformSHA256(t *testing.T) {
	generated := GenerateCheckedInTargetProjection(
		resolveNativeTarget(t, "darwin/arm64"), "darwin/arm64", "main")
	if !generated.Ok {
		t.Fatalf("generate Darwin target projection: %#v", generated.Diagnostics)
	}
	if !bytes.Contains(generated.Source, []byte("if renvoRTGSHA256(data, hardware)")) {
		t.Fatal("Darwin target projection omitted the platform SHA-256 fast path")
	}
}

func TestCheckedInWindowsProjectionsCacheRuntimeIOHelpers(t *testing.T) {
	for _, target := range []string{"windows/386", "windows/arm64"} {
		generated := GenerateCheckedInTargetProjection(
			resolveNativeTarget(t, target), target, "main")
		if !generated.Ok {
			t.Fatalf("generate %s: %#v", target, generated.Diagnostics)
		}
		if !containsText(string(generated.Source), "g.winReadEmitted") ||
			!containsText(string(generated.Source), "g.winWriteEmitted") {
			t.Errorf("%s projection duplicates runtime I/O helpers at each call site", target)
		}
	}
}

func TestWindowsArm64ParameterlessEntrySkipsArgvRuntime(t *testing.T) {
	generated := GenerateCheckedInTargetProjection(
		resolveNativeTarget(t, "windows/arm64"), "windows/arm64", "main")
	if !generated.Ok {
		t.Fatalf("generate windows/arm64: %#v", generated.Diagnostics)
	}
	source := string(generated.Source)
	entry := strings.Index(source, "func renvoEmitProgramEntryArgsWindowsArm64(")
	if entry < 0 {
		t.Fatal("windows/arm64 projection omitted its entry adapter")
	}
	guard := strings.Index(source[entry:], "if app.paramCount == 0")
	allocation := strings.Index(source[entry:], "renvoWindowsArm64ArgsBSSAlignment")
	if guard < 0 || allocation < 0 || guard > allocation {
		t.Fatalf("parameterless entry guard does not precede argv runtime allocation")
	}
}

func TestWindows386RuntimeIOUsesBoundedSequences(t *testing.T) {
	const filename = "../../backend/definitions/windows_386.rtg"
	source, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"go compiler {", "io_template", "read_code", "write_code",
		"renvoAsmEmitText(a, \"\\xe9",
	} {
		if strings.Contains(string(source), forbidden) {
			t.Fatalf("Windows/386 retained duplicated or opaque runtime implementation %q", forbidden)
		}
	}
	resolved := resolveNativeTarget(t, "windows/386")
	sequences := architectureSequences(resolved.Targets[0].Arch)
	for _, sequence := range []string{
		"WindowsRead", "WindowsWrite", "WindowsReadAt", "WindowsWriteAt",
		"WindowsRuntimeIOBody",
	} {
		found := false
		for i := 0; i < len(sequences); i++ {
			if strings.HasSuffix(sequences[i].Name, sequence) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Windows/386 architecture omitted %s sequence", sequence)
		}
	}
}

func sortedNativeEntrypoints() []string {
	filenames := make([]string, 0, len(nativeDefinitionEntrypoints))
	for _, filename := range nativeDefinitionEntrypoints {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	return filenames
}

func sortedNativeTargetNames() []string {
	targets := make([]string, 0, len(nativeDefinitionEntrypoints))
	for target := range nativeDefinitionEntrypoints {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func resolveNativeTarget(t *testing.T, target string) ResolveResult {
	t.Helper()
	filename, ok := nativeDefinitionEntrypoints[target]
	if !ok {
		t.Fatalf("no native definition entrypoint for %s", target)
	}
	resolved := Resolve(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 1 ||
		resolved.Targets[0].Descriptor.Name != target {
		t.Fatalf("%s resolved targets = %#v, want only %s",
			filename, resolved.Targets, target)
	}
	return resolved
}

func resolveNativeArchitectureProjection(t *testing.T, arch string) ResolveResult {
	t.Helper()
	filename, ok := nativeArchitectureProjections[arch]
	if !ok {
		t.Fatalf("no native architecture projection for %s", arch)
	}
	resolved := ResolveArchitectureDefinition(parseDefinitionFile(t, filename))
	if !resolved.Ok {
		t.Fatalf("resolve %s: %#v", filename, resolved.Diagnostics)
	}
	if len(resolved.Targets) != 0 {
		t.Fatalf("%s resolved targets = %#v, want none",
			filename, resolved.Targets)
	}
	return resolved
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
	resolved := Resolve(parseDefinitionFile(t, "../../backend/definitions/wasm32.rtg"))
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
		"(a.c == nil || a.c.renvoTarget == renvoTargetVM32)",
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
}

func TestCheckedInLLVMPreparedOutput(t *testing.T) {
	resolved := ResolveDefinitions(parseDefinitionFile(t, "../../backends/llvm_amd64.rtg"))
	if !resolved.Ok {
		t.Fatalf("resolve LLVM definition: %#v", resolved.Diagnostics)
	}
	generated := GeneratePreparedBackend(resolved, "llvm/linux-amd64")
	if !generated.Ok {
		t.Fatalf("generate prepared LLVM backend: %#v", generated.Diagnostics)
	}
	generated.Source = append([]byte("//go:build renvo_prepared\n\n"), generated.Source...)
	checkedIn, err := os.ReadFile("../../backend/compiler_llvm_prepared_impl.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(generated.Source, checkedIn) {
		t.Fatal("checked-in prepared LLVM backend is stale; run go run ./internal/rtg/cmd/rtggen -prepared -build-tag renvo_prepared -t llvm/linux-amd64 -o backend/compiler_llvm_prepared_impl.go backends/llvm_amd64.rtg")
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
