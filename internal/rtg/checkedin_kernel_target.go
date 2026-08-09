//go:build !renvo

package rtg

import "bytes"

func generateCheckedInLinuxKernelAmd64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	if declarativeFormatKernelImage(target.Object) != "linux_module_elf" {
		return checkedInTargetProjectionFailure(document, target.Object,
			"linux-kernel/amd64 requires a declarative Linux-module ELF image")
	}
	prologue, hasPrologue := architectureGoHook(target.Runtime, "entry_prologue")
	epilogue, hasEpilogue := architectureGoHook(target.Runtime, "entry_epilogue")
	callback, hasCallback := architectureGoHook(target.Runtime, "emit_callback_address")
	if !hasPrologue || !hasEpilogue || !hasCallback {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux-kernel/amd64 requires entry and callback ABI sequences")
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	source = append(source, checkedInLinuxKernelAmd64APISource...)
	projection := document
	projection.Unit = "builtin"
	staticCall, hasStaticCall := architectureGoHook(target.Runtime, "emit_static_call")
	runtimeOperation, hasRuntimeOperation := architectureGoHook(target.Runtime, "emit_operation")
	if !hasStaticCall || !hasRuntimeOperation {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux-kernel/amd64 requires static-call and runtime-operation hooks")
	}
	sequenceRoots := []string{prologue, epilogue, callback}
	for _, field := range []string{
		"append64", "append_header", "append_relocation", "append_section", "append_symbol",
	} {
		algorithm, found := architectureGoHook(target.Object, field)
		if !found {
			return checkedInTargetProjectionFailure(document, target.Object,
				"linux-kernel/amd64 requires all Linux-module ELF format hooks")
		}
		sequenceRoots = append(sequenceRoots, algorithm)
	}
	goRoots, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, []string{staticCall, runtimeOperation}, sequenceRoots)
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(
		source, projection, target.Arch, true, false, true, sequences)
	source = bytes.ReplaceAll(source, []byte("renvoRTGRel32"),
		[]byte("renvoKernelAmd64Rel32"))
	source = appendDeclarativeKernelImage(source, projection, target.Object, true)
	kernelImage := declarativeFormatKernelImageName(projection, target.Object)
	kernelFormat := kernelImage[:len(kernelImage)-len("KernelImage")]
	source = append(source, "\nfunc renvoAmd64KernelBTFModuleLayout(data []byte) (int, int, int, int, bool) {\n\tlayout := "...)
	source = append(source, kernelFormat...)
	source = append(source, "BTFModuleLayout(data)\n\treturn layout.size, layout.nameOffset, layout.initOffset, layout.exitOffset, layout.ok\n}\n"...)
	source = append(source, "\nfunc renvoAmd64KernelEntryPrologue(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(projection, prologue)...)
	source = append(source, "(a)\n}\n\nfunc renvoAmd64KernelEntryEpilogue(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(projection, epilogue)...)
	source = append(source, "(a)\n}\n\nfunc renvoAmd64KernelCallbackAddress(a *renvoAsm, label int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(projection, callback)...)
	source = append(source, "(a, label)\n}\n"...)
	source = append(source, "\nfunc renvoAmd64EmitKernelPrintValue(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(projection, runtimeOperation)...)
	source = append(source, "(a, 2)\n}\n\nfunc renvoAmd64EmitKernelStaticCall(a *renvoAsm, importID int, wordCount int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(projection, staticCall)...)
	source = append(source, "(a, importID, wordCount)\n}\n"...)
	source = append(source, `

func compileLinuxKernelAmd64(input []int, output int) int {
	return compileLinuxKernelAmd64Arena(input, output, 0)
}

func compileLinuxKernelAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxKernelAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}
`...)
	source = append(source, "\nfunc renvoAsmImageKernelModuleAmd64(a *renvoAsm, initLabel int, exitLabel int) []byte {\n\trenvoNonNil(a)\n\trenvoAsmPatch(a)\n\treturn "...)
	source = append(source, kernelImage...)
	source = append(source, "(a, initLabel, exitLabel)\n}\n"...)
	source = rewriteCheckedInLinuxKernelAmd64EmitterMethods(source)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

const checkedInLinuxKernelAmd64APISource = `

const renvoKernelAmd64RelocationAbsoluteData = 0
const renvoKernelAmd64RelocationAbsoluteBSS = 1
const renvoKernelAmd64RelocationImport = 2
const renvoKernelAmd64RelocationAbsoluteBSSEnd = 3

func renvoKernelAmd64Rel32(out *renvoAsm, label int) {
	at := len(out.code)
	renvoAsmEmit32(out, 0)
	if label >= 0 {
		renvoAsmAddReloc(out, at, label)
	}
}

func renvoKernelAmd64ExternalImportCount(out *renvoAsm) int {
	return len(out.kernelImportOffsets) / 2
}

func renvoKernelAmd64ExternalImportName(out *renvoAsm, index int) string {
	at := index * 2
	if at < 0 || at+1 >= len(out.kernelImportOffsets) {
		return ""
	}
	start := out.kernelImportOffsets[at]
	end := out.kernelImportOffsets[at+1]
	return string(out.kernelImportNames[start:end])
}

func renvoKernelAmd64AbsoluteRelocationOffset(out *renvoAsm, index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.absRelocs, index*3))
}

func renvoKernelAmd64AbsoluteRelocationAddend(out *renvoAsm, index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.absRelocs, index*3+1))
}

func renvoKernelAmd64AbsoluteRelocationKind(out *renvoAsm, index int) int {
	return int(renvo_runtime_UnsafeInt32At(out.absRelocs, index*3+2))
}
`

func rewriteCheckedInLinuxKernelAmd64EmitterMethods(source []byte) []byte {
	replacements := []string{
		"emitter.LabelPosition(", "renvoAsmLabelPosition(emitter, ",
		"emitter.KernelBTF()", "emitter.c.kernel.kernelBTF",
		"emitter.KernelModuleName()", "emitter.c.kernel.kernelModuleName",
		"emitter.KernelLicense()", "emitter.c.kernel.kernelLicense",
		"emitter.KernelSymvers()", "emitter.c.kernel.kernelSymvers",
		"emitter.KernelRelease()", "emitter.c.kernel.kernelRelease",
		"emitter.KernelVersion()", "emitter.c.kernel.kernelVersion",
		"emitter.ExternalImportCount()", "renvoKernelAmd64ExternalImportCount(emitter)",
		"emitter.ExternalImportName(", "renvoKernelAmd64ExternalImportName(emitter, ",
		"emitter.AbsoluteRelocationCount()", "len(emitter.absRelocs)/3",
		"emitter.AbsoluteRelocationOffset(", "renvoKernelAmd64AbsoluteRelocationOffset(emitter, ",
		"emitter.AbsoluteRelocationAddend(", "renvoKernelAmd64AbsoluteRelocationAddend(emitter, ",
		"emitter.AbsoluteRelocationKind(", "renvoKernelAmd64AbsoluteRelocationKind(emitter, ",
		"emitter.BSSSize()", "emitter.bssSize",
		"emitter.Code()", "emitter.code",
		"emitter.Data()", "emitter.data",
		"RTGRelocationAbsoluteData", "renvoKernelAmd64RelocationAbsoluteData",
		"RTGRelocationAbsoluteBSSEnd", "renvoKernelAmd64RelocationAbsoluteBSSEnd",
		"RTGRelocationAbsoluteBSS", "renvoKernelAmd64RelocationAbsoluteBSS",
		"RTGRelocationImport", "renvoKernelAmd64RelocationImport",
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(
			source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
	return source
}
