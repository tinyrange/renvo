//go:build !renvo

package rtg

import "bytes"

func generateCheckedInDarwinAarch64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	bits, bitsOK := integerField(document, target.Executable, "address_bits")
	cpu, cpuOK := integerField(document, target.Executable, "cpu")
	codeOffset, offsetOK := integerField(document, target.Executable, "code_offset")
	image, imageOK := architectureGoHook(target.Executable, "image")
	if !bitsOK || bits != 64 || !cpuOK || cpu != 0x0100000c || !offsetOK || codeOffset != 0x1000 || !imageOK {
		return checkedInTargetProjectionFailure(document, target.Executable, "darwin/arm64 requires the declarative Mach-O arm64 layout")
	}
	template, ok := decodeRuntimeEntryTemplate(target.Runtime)
	if !ok || template.algorithm == "" || !template.requiresState || template.maxParameters != 2 || len(template.bss) != 2 {
		return checkedInTargetProjectionFailure(document, target.Runtime, "darwin/arm64 requires a stateful two-buffer sequence entry")
	}
	entryStart, _ := architectureGoHook(target.Runtime, "emit_entry_start")
	staticCall, _ := architectureGoHook(target.Runtime, "emit_static_call")
	entryStateBytes, stateOK := integerField(document, target.Runtime, "entry_state_bytes")
	if !stateOK || entryStateBytes != 24 || entryStart == "" || staticCall == "" {
		return checkedInTargetProjectionFailure(document, target.Runtime, "darwin/arm64 requires its three-word entry state")
	}
	exitHook, _ := architectureGoHook(target.Runtime, "emit_exit")
	sequenceRoots := []string{template.algorithm, image, exitHook}
	for _, operation := range []string{"read", "write", "read_at", "write_at", "open", "close", "chmod"} {
		algorithm, _ := runtimeOperationHook(target.Runtime, operation)
		sequenceRoots = append(sequenceRoots, algorithm)
	}
	for i := 0; i < len(sequenceRoots); i++ {
		if sequenceRoots[i] == "" {
			return checkedInTargetProjectionFailure(document, target.Runtime, "darwin/arm64 is missing a required target algorithm")
		}
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(manifest, "target-projection/"+target.Descriptor.Name, packageName)
	projection := document
	projection.Unit = "builtin"
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	goRoots, sequences := resolveArchitectureSequenceProjection(projection, target.Arch, []string{entryStart, staticCall}, sequenceRoots)
	source = appendReachableEmbeddedGoExcluding(source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInAarch64EmitterMethods(source)
	source = rewriteCheckedInDarwinAarch64Registers(source)
	source = bytes.ReplaceAll(source, []byte("rtgBuiltinJumpCondition(out,rtgBuiltinEQ,"), []byte("renvoDarwinAarch64BCondEQ(out,"))
	source = bytes.ReplaceAll(source, []byte("rtgBuiltinJumpCondition(out,rtgBuiltinSGE,"), []byte("renvoDarwinAarch64BCondSGE(out,"))
	source = append(source, `

func renvoDarwinAarch64BCondEQ(out *renvoAsm, label int) { renvoAarch64AsmBCondLabel(out, label, 0) }
func renvoDarwinAarch64BCondSGE(out *renvoAsm, label int) { renvoAarch64AsmBCondLabel(out, label, 10) }
`...)
	source = appendCheckedInDarwinAarch64Runtime(source, projection, target.Runtime, template, image, entryStart, staticCall, entryStateBytes, codeOffset)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

func appendCheckedInDarwinAarch64Runtime(source []byte, document Document,
	runtime Declaration, template runtimeEntryTemplate, image string, entryStart string,
	staticCall string, entryStateBytes int, codeOffset int) []byte {
	source = append(source, "\nconst renvoDarwinArm64CodeOffset = "...)
	source = appendDecimalFrame(source, codeOffset)
	source = append(source, "\nconst renvoDarwinArm64EntryStateBytes = "...)
	source = appendDecimalFrame(source, entryStateBytes)
	source = append(source, "\n\nfunc renvoDarwinSHA256(data []byte) []byte {\n\treturn rtgBuiltinDarwinAarch64PackageMachSHA256(data)\n}\n"...)
	exitHook, _ := architectureGoHook(runtime, "emit_exit")
	wrappers := []struct{ name, algorithm string }{}
	for _, operation := range []struct{ name, suffix string }{{"read", "Read"}, {"write", "Write"}, {"read_at", "ReadAt"}, {"write_at", "WriteAt"}, {"open", "Open"}, {"close", "Close"}, {"chmod", "Chmod"}} {
		algorithm, _ := runtimeOperationHook(runtime, operation.name)
		wrappers = append(wrappers, struct{ name, algorithm string }{"renvoDarwinArm64Definition" + operation.suffix, algorithm})
	}
	for _, wrapper := range wrappers {
		source = append(source, "\nfunc "...)
		source = append(source, wrapper.name...)
		source = append(source, "(a *renvoAsm) {\n\t"...)
		source = append(source, checkedInProjectionAlgorithmName(document, wrapper.algorithm)...)
		source = append(source, "(a)\n}\n"...)
	}
	source = append(source, "\nfunc renvoDarwinArm64DefinitionExit(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, exitHook)...)
	source = append(source, "(a, 0)\n}\n"...)
	source = append(source, "\nfunc renvoDarwinArm64DefinitionEntryStart(a *renvoAsm, stateOffset int) bool {\n\treturn "...)
	source = append(source, checkedInProjectionAlgorithmName(document, entryStart)...)
	source = append(source, "(a, stateOffset)\n}\n"...)
	source = append(source, "\nfunc renvoDarwinArm64DefinitionStaticCall(a *renvoAsm, importID int, wordCount int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, staticCall)...)
	source = append(source, "(a, importID, wordCount)\n}\n"...)
	source = append(source, "\nfunc renvoDarwinArm64DefinitionImage(a *renvoAsm) []byte {\n\treturn "...)
	source = append(source, checkedInProjectionAlgorithmName(document, image)...)
	source = append(source, "(a)\n}\n"...)
	source = append(source, `
func renvoDarwinArm64Image(a *renvoAsm) []byte {
	data := renvoDarwinArm64DefinitionImage(a)
	if renvoFixedTarget == 0 && a.c.emitImage {
		return renvoAppendReplLinkTable(data, a)
	}
	return data
}
`...)
	bssSymbols := []string{"renvoDarwinArm64ArgsBSS", "renvoDarwinArm64EnvironmentBSS"}
	for i := 0; i < len(template.bss); i++ {
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Size = "...)
		source = appendDecimalFrame(source, template.bss[i].size)
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Alignment = "...)
		source = appendDecimalFrame(source, template.bss[i].alignment)
	}
	source = append(source, "\n\nfunc renvoDarwinArm64DefinitionEntry(a *renvoAsm, argsOff int, environmentOff int, stateOffset int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, template.algorithm)...)
	source = append(source, "(a, argsOff, environmentOff, stateOffset)\n}\n"...)
	source = append(source, `

func renvoEmitProgramEntryArgsDarwinArm64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) { return false }
	if app.paramCount == 0 { return true }
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) { return false }
	if app.paramCount == 2 && !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) { return false }
	argsOff := renvoAlignValue(g.asm.bssSize, renvoDarwinArm64ArgsBSSAlignment)
	g.asm.bssSize = argsOff + renvoDarwinArm64ArgsBSSSize
	envOff := renvoAlignValue(g.asm.bssSize, renvoDarwinArm64EnvironmentBSSAlignment)
	g.asm.bssSize = envOff + renvoDarwinArm64EnvironmentBSSSize
	renvoDarwinArm64DefinitionEntry(&g.asm, argsOff, envOff, g.darwinEntryOff)
	return true
}
`...)
	return source
}

func rewriteCheckedInDarwinAarch64Registers(source []byte) []byte {
	for i := 0; i < 32; i++ {
		compact := "RTGRegister{Code:" + decimalFrame(i) + ",Valid:true}"
		spaced := "RTGRegister{Code: " + decimalFrame(i) + ", Valid: true}"
		source = bytes.ReplaceAll(source, []byte(compact), []byte(decimalFrame(i)))
		source = bytes.ReplaceAll(source, []byte(spaced), []byte(decimalFrame(i)))
	}
	source = bytes.ReplaceAll(source, []byte("RTGRegister"), []byte("int"))
	source = bytes.ReplaceAll(source, []byte(".Code"), nil)
	return source
}

func generateCheckedInWindowsAarch64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	bits, bitsOK := integerField(document, target.Executable, "address_bits")
	machine, machineOK := integerField(document, target.Executable, "machine")
	codeOffset, offsetOK := integerField(document, target.Executable, "code_offset")
	fileAlignment, fileOK := integerField(document, target.Executable, "file_alignment")
	sectionAlignment, sectionOK := integerField(document, target.Executable, "section_alignment")
	if declarativeFormatImage(target.Executable) != "pe_executable" ||
		!bitsOK || bits != 64 || !machineOK || machine != 0xaa64 ||
		!offsetOK || codeOffset != 0x1000 || !fileOK || fileAlignment != 0x200 ||
		!sectionOK || sectionAlignment != 0x1000 {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"windows/arm64 requires the 64-bit ARM PE layout")
	}
	library, imports := runtimePEImports(target.Runtime)
	expectedImports := []string{"CreateFileA", "CloseHandle", "ReadFile", "WriteFile", "SetFilePointer", "GetStdHandle", "GetCommandLineA", "ExitProcess", "GetEnvironmentStringsA"}
	if library != "kernel32.dll" || len(imports) != len(expectedImports) {
		return checkedInTargetProjectionFailure(document, target.Runtime, "windows/arm64 requires the stable kernel32 import ABI")
	}
	for i := 0; i < len(imports); i++ {
		if imports[i] != expectedImports[i] {
			return checkedInTargetProjectionFailure(document, target.Runtime, "windows/arm64 requires the stable kernel32 import ABI")
		}
	}
	template, ok := decodeRuntimeEntryTemplate(target.Runtime)
	if !ok || template.algorithm == "" || template.maxParameters != 2 || len(template.bss) != 5 {
		return checkedInTargetProjectionFailure(document, target.Runtime, "windows/arm64 requires a complete five-buffer sequence entry")
	}
	roots := []string{template.algorithm}
	exit, _ := architectureGoHook(target.Runtime, "emit_exit")
	staticCall, _ := architectureGoHook(target.Runtime, "emit_static_call")
	roots = append(roots, exit, staticCall)
	for _, operation := range []string{"read", "write", "read_at", "write_at", "open", "close", "chmod"} {
		algorithm, _ := runtimeOperationHook(target.Runtime, operation)
		roots = append(roots, algorithm)
	}
	for i := 0; i < len(roots); i++ {
		if roots[i] == "" {
			return checkedInTargetProjectionFailure(document, target.Runtime, "windows/arm64 runtime is missing a required operation sequence")
		}
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(manifest, "target-projection/"+target.Descriptor.Name, packageName)
	projection := document
	projection.Unit = "builtin"
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	goRoots, sequences := resolveArchitectureSequenceProjection(projection, target.Arch, []string{staticCall}, roots)
	source = appendReachableEmbeddedGoExcluding(source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInAarch64EmitterMethods(source)
	source = bytes.ReplaceAll(source, []byte("out.ReserveBSS("), []byte("renvoWindowsArm64ReserveBSS(out,"))
	source = append(source, `

func renvoWindowsArm64ReserveBSS(out *renvoAsm, size int, alignment int) int {
	offset := renvoAlignValue(out.bssSize, alignment)
	out.bssSize = offset + size
	return offset
}
`...)
	source = appendCheckedInWindowsAarch64Runtime(source, projection, target.Runtime, template)
	source = append(source, checkedInWindowsAarch64PackagingSource...)
	source = bytes.TrimRight(source, "\n")
	source = append(source, '\n')
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

const checkedInWindowsAarch64PackagingSource = `

func compileWindowsArm64(input []int, output int) int {
	return compileWindowsArm64Arena(input, output, 0)
}
func compileWindowsArm64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindowsArm64)
	return renvoCompileAarch64(input, output, arenaSize)
}

func renvoEmitProgramEntryArgsWindowsArm64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) {
		return false
	}
	if app.paramCount == 2 && !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) {
		return false
	}
	argsOff := renvoAlignValue(g.asm.bssSize, renvoWindowsArm64ArgsBSSAlignment)
	g.asm.bssSize = argsOff + renvoWindowsArm64ArgsBSSSize
	argsTextOff := renvoAlignValue(g.asm.bssSize, renvoWindowsArm64ArgsTextBSSAlignment)
	g.asm.bssSize = argsTextOff + renvoWindowsArm64ArgsTextBSSSize
	argsLenOff := renvoAlignValue(g.asm.bssSize, renvoWindowsArm64ArgsLengthBSSAlignment)
	g.asm.bssSize = argsLenOff + renvoWindowsArm64ArgsLengthBSSSize
	envOff := renvoAlignValue(g.asm.bssSize, renvoWindowsArm64EnvironmentBSSAlignment)
	g.asm.bssSize = envOff + renvoWindowsArm64EnvironmentBSSSize
	envLenOff := renvoAlignValue(g.asm.bssSize, renvoWindowsArm64EnvironmentLengthBSSAlignment)
	g.asm.bssSize = envLenOff + renvoWindowsArm64EnvironmentLengthBSSSize
	renvoWinArm64DefinitionEntry(&g.asm, argsOff, argsTextOff, argsLenOff, envOff, envLenOff)
	return true
}

func renvoAsmPatchWindowsArm64(a *renvoAsm, layout renvoWinImportLayout) {
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsWinImportReloc {
			target = layout.iatRVAs[off]
		} else if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		insn := renvoGet32At(a.code, at)
		reg := insn & 31
		pc := a.codeOffset + at
		delta := (target >> 12) - (pc >> 12)
		imm := delta & 0x1fffff
		renvoPut32At(a.code, at, 0x90000000|((imm&3)<<29)|(((imm>>2)&0x7ffff)<<5)|reg)
		renvoPut32At(a.code, at+4, 0x91000000|((target&0xfff)<<10)|(reg<<5)|reg)
		renvoPut32At(a.code, at+8, 0xd503201f)
		renvoPut32At(a.code, at+12, 0xd503201f)
	}
}

func renvoAsmImageWindowsArm64(a *renvoAsm) []byte {
	renvoAsmPatch(a)
	for (a.codeOffset+len(a.code))%8 != 0 {
		a.code = append(a.code, 0)
	}
	textVirtualSize := len(a.code)
	textRawSize := renvoAlignValue(textVirtualSize, renvoWinFileAlign)
	dataRVA := renvoAlignValue(a.codeOffset+textVirtualSize, renvoWinSectionAlign)
	a.dataOffset = dataRVA
	var imports renvoWinImportLayout
	if renvoAsmHasWinImportRelocs(a) {
		renvoAppendWinImports(a, &imports)
	}
	renvoAsmPatchWindowsArm64(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = renvoAppendPEHeader64WithContext(a.c, out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
	out[0xc0] = 6
	out[0xc2] = 1
	out[0xc4] = 1
	out[0xc8] = 6
	out[0xca] = 1
	out[0x9a] = 3
	out[0x96] = 0x22
	out[0xde] = 0
	out[0xdf] = 0x81
	out = append(out, a.code...)
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize)
	out = append(out, a.data...)
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize+dataRawSize)
	if renvoFixedTarget == 0 && a.c.emitImage {
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}

`

func appendCheckedInWindowsAarch64Runtime(source []byte, document Document,
	runtime Declaration, template runtimeEntryTemplate) []byte {
	exit, _ := architectureGoHook(runtime, "emit_exit")
	staticCall, _ := architectureGoHook(runtime, "emit_static_call")
	readAt, _ := runtimeOperationHook(runtime, "read_at")
	writeAt, _ := runtimeOperationHook(runtime, "write_at")
	open, _ := runtimeOperationHook(runtime, "open")
	closeOperation, _ := runtimeOperationHook(runtime, "close")
	chmod, _ := runtimeOperationHook(runtime, "chmod")
	wrappers := []struct{ name, algorithm, parameters, arguments string }{
		{"renvoWinArm64DefinitionStaticCall", staticCall, "a *renvoAsm, importID int, wordCount int", "a, importID, wordCount"},
		{"renvoWinArm64DefinitionOpen", open, "a *renvoAsm", "a"},
		{"renvoWinArm64DefinitionClose", closeOperation, "a *renvoAsm", "a"},
		{"renvoWinArm64DefinitionChmod", chmod, "a *renvoAsm", "a"},
	}
	for _, wrapper := range wrappers {
		source = append(source, "\nfunc "...)
		source = append(source, wrapper.name...)
		source = append(source, '(')
		source = append(source, wrapper.parameters...)
		source = append(source, ") {\n\t"...)
		source = append(source, checkedInProjectionAlgorithmName(document, wrapper.algorithm)...)
		source = append(source, '(')
		source = append(source, wrapper.arguments...)
		source = append(source, ")\n}\n"...)
	}
	source = append(source, `
func renvoWinArm64DefinitionReadAt(g *renvoLinearGen) {
	renvoWinArm64DefinitionReadWrite(g, false)
}

func renvoWinArm64DefinitionWriteAt(g *renvoLinearGen) {
	renvoWinArm64DefinitionReadWrite(g, true)
}

func renvoWinArm64DefinitionReadWrite(g *renvoLinearGen, isWrite bool) {
	a := &g.asm
	label := g.winReadLabel
	if isWrite {
		label = g.winWriteLabel
	}
	if isWrite && g.winWriteEmitted || !isWrite && g.winReadEmitted {
		renvoAsmCallLabel(a, label)
		return
	}
	label = renvoAsmNewLabel(a)
	if isWrite {
		g.winWriteEmitted = true
		g.winWriteLabel = label
	} else {
		g.winReadEmitted = true
		g.winReadLabel = label
	}
	after := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, after, label)
`...)
	source = append(source, "\tif isWrite {\n\t\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, writeAt)...)
	source = append(source, "(a)\n\t} else {\n\t\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, readAt)...)
	source = append(source, `(a)
	}
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, after)
	renvoAsmCallLabel(a, label)
}
`...)
	source = append(source, "\nfunc renvoWinArm64DefinitionExit(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, exit)...)
	source = append(source, "(a, 0)\n}\n"...)
	bssSymbols := []string{"renvoWindowsArm64ArgsBSS", "renvoWindowsArm64ArgsTextBSS", "renvoWindowsArm64ArgsLengthBSS", "renvoWindowsArm64EnvironmentBSS", "renvoWindowsArm64EnvironmentLengthBSS"}
	for i := 0; i < len(template.bss); i++ {
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Size = "...)
		source = appendDecimalFrame(source, template.bss[i].size)
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Alignment = "...)
		source = appendDecimalFrame(source, template.bss[i].alignment)
	}
	source = append(source, "\n\nfunc renvoWinArm64DefinitionEntry(a *renvoAsm, argsOff int, argsTextOff int, argsLengthOff int, environmentOff int, environmentLengthOff int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, template.algorithm)...)
	source = append(source, "(a, argsOff, argsTextOff, argsLengthOff, environmentOff, environmentLengthOff)\n}\n"...)
	return source
}

func generateCheckedInLinuxAarch64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	bits, bitsOK := integerField(document, target.Executable, "address_bits")
	machine, machineOK := integerField(document, target.Executable, "machine")
	codeOffset, offsetOK := integerField(document, target.Executable, "code_offset")
	alignment, alignmentOK := integerField(document, target.Executable, "file_alignment")
	patch, patchOK := fieldValue(document, target.Executable, "patch_absolute")
	if declarativeFormatImage(target.Executable) != "elf_executable" ||
		!bitsOK || bits != 64 || !machineOK || machine != 183 ||
		!offsetOK || codeOffset != 176 || !alignmentOK || alignment != 0x1000 ||
		!patchOK || valueName(patch) != "aarch64_page_address" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"linux/aarch64 requires the 64-bit two-segment ELF layout")
	}
	operations, missing := checkedInLinuxAarch64RuntimeOperations(target.Runtime)
	if missing != "" {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/aarch64 runtime operation "+missing+" requires a syscall number")
	}
	template, ok := decodeRuntimeEntryTemplate(target.Runtime)
	if !ok || template.algorithm == "" || template.maxParameters != 2 || len(template.bss) != 3 {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/aarch64 requires a complete three-buffer sequence entry")
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(manifest,
		"target-projection/"+target.Descriptor.Name, packageName)
	projection := document
	projection.Unit = "builtin"
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	goRoots, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, nil, []string{template.algorithm})
	source = appendReachableEmbeddedGoExcluding(source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInAarch64EmitterMethods(source)
	source = appendCheckedInLinuxAarch64Runtime(source, operations, projection, template)
	source = append(source, checkedInLinuxAarch64ImageSource...)
	return GenerateResult{Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true}
}

func checkedInLinuxAarch64RuntimeOperations(runtime Declaration) ([]checkedInLinuxAmd64RuntimeOperation, string) {
	operations := []checkedInLinuxAmd64RuntimeOperation{
		{name: "read", symbol: "renvoLinuxAarch64SysReadSeq"},
		{name: "write", symbol: "renvoLinuxAarch64SysWriteSeq"},
		{name: "open", symbol: "renvoLinuxAarch64SysOpen"},
		{name: "close", symbol: "renvoLinuxAarch64SysClose"},
		{name: "read_at", symbol: "renvoLinuxAarch64SysReadAt"},
		{name: "write_at", symbol: "renvoLinuxAarch64SysWriteAt"},
		{name: "chmod", symbol: "renvoLinuxAarch64SysFchmod"},
		{name: "exit", symbol: "renvoLinuxAarch64SysExit"},
	}
	for i := 0; i < len(operations); i++ {
		value, ok := runtimeOperationInteger(runtime, operations[i].name, "number")
		if !ok {
			return nil, operations[i].name
		}
		operations[i].value = value
	}
	return operations, ""
}

func rewriteCheckedInAarch64EmitterMethods(source []byte) []byte {
	replacements := []string{
		"out.NewLabel()", "renvoAsmNewLabel(out)",
		"out.Mark(", "renvoAsmMarkLabel(out, ",
		"rtgBuiltinJump(out,", "renvoAarch64AsmJmpLabel(out, ",
		"rtgBuiltinCall(out,", "renvoAarch64AsmCallLabel(out, ",
		"status.Code", "status",
		"RTGRegister", "int",
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
	prefix := []byte("rtgBuiltinJumpCondition(out,RTGCondition{Code:")
	for {
		start := bytes.Index(source, prefix)
		if start < 0 {
			return source
		}
		codeStart := start + len(prefix)
		codeEndOffset := bytes.Index(source[codeStart:], []byte("},"))
		if codeEndOffset < 0 {
			return source
		}
		codeEnd := codeStart + codeEndOffset
		labelStart := codeEnd + 2
		labelEndOffset := bytes.IndexByte(source[labelStart:], ')')
		if labelEndOffset < 0 {
			return source
		}
		labelEnd := labelStart + labelEndOffset
		rewritten := make([]byte, 0, len(source))
		rewritten = append(rewritten, source[:start]...)
		rewritten = append(rewritten, "renvoAarch64AsmBCondLabel(out,"...)
		rewritten = append(rewritten, source[labelStart:labelEnd]...)
		rewritten = append(rewritten, ',')
		rewritten = append(rewritten, source[codeStart:codeEnd]...)
		rewritten = append(rewritten, source[labelEnd:]...)
		source = rewritten
	}
}

func appendCheckedInLinuxAarch64Runtime(source []byte, operations []checkedInLinuxAmd64RuntimeOperation,
	document Document, template runtimeEntryTemplate) []byte {
	for i := 0; i < len(operations); i++ {
		source = append(source, "\nconst "...)
		source = append(source, operations[i].symbol...)
		source = append(source, " = "...)
		source = appendDecimalFrame(source, operations[i].value)
	}
	source = append(source, `

func renvoAarch64AsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRsi, renvoAarch64RegRax)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRcx)
}

func renvoAarch64AsmMoveOffsetArg(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR10, renvoAarch64RegRax)
}

func compileLinuxAarch64(input []int, output int) int { return compileLinuxAarch64Arena(input, output, 0) }
func compileLinuxAarch64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxAarch64)
	return renvoCompileAarch64(input, output, arenaSize)
}
`...)
	bssSymbols := []string{"renvoLinuxAarch64ArgsBSS", "renvoLinuxAarch64EnvironmentBSS", "renvoLinuxAarch64EnvironmentLengthBSS"}
	for i := 0; i < len(template.bss); i++ {
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Size = "...)
		source = appendDecimalFrame(source, template.bss[i].size)
		source = append(source, "\nconst "...)
		source = append(source, bssSymbols[i]...)
		source = append(source, "Alignment = "...)
		source = appendDecimalFrame(source, template.bss[i].alignment)
	}
	source = append(source, "\n\nfunc renvoAsmBuildArgvEnvSlicesAarch64(a *renvoAsm, argsOff int, environmentOff int, environmentLengthOff int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, template.algorithm)...)
	source = append(source, "(a, argsOff, environmentOff, environmentLengthOff)\n}\n"...)
	source = append(source, `

func renvoEmitProgramEntryArgsAarch64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) { return false }
	argsOff := renvoAlignValue(g.asm.bssSize, renvoLinuxAarch64ArgsBSSAlignment)
	g.asm.bssSize = argsOff + renvoLinuxAarch64ArgsBSSSize
	envOff := renvoAlignValue(g.asm.bssSize, renvoLinuxAarch64EnvironmentBSSAlignment)
	g.asm.bssSize = envOff + renvoLinuxAarch64EnvironmentBSSSize
	envLenOff := renvoAlignValue(g.asm.bssSize, renvoLinuxAarch64EnvironmentLengthBSSAlignment)
	g.asm.bssSize = envLenOff + renvoLinuxAarch64EnvironmentLengthBSSSize
	renvoAsmBuildArgvEnvSlicesAarch64(&g.asm, argsOff, envOff, envLenOff)
	if app.paramCount == 0 { return true }
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) { return false }
	return app.paramCount == 1 || renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ)
}
`...)
	return source
}

const checkedInLinuxAarch64ImageSource = `

func renvoAsmImageAarch64(a *renvoAsm) []byte {
	renvoAsmPatch(a); renvoAsmPatchAarch64Abs(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data); bssOffset := renvoAsmBssOffset(a)
	if a.c.stripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		out = append(out, a.code...); out = append(out, a.data...)
		if renvoFixedTarget == 0 { return renvoAppendReplLinkTable(out, a) }; return out
	}
	var sec renvoElfSymbolSections; renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	out := make([]byte, 0, 1048576)
	out = renvoAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
	out = append(out, a.code...); out = append(out, a.data...)
	out = renvoAppendUntil(out, sec.symtabOff); out = append(out, sec.symtab...)
	out = renvoAppendUntil(out, sec.strtabOff); out = append(out, sec.strtab...)
	out = renvoAppendUntil(out, sec.shstrOff); out = append(out, sec.shstrtab...)
	out = renvoAppendUntil(out, sec.shoff); out = renvoAppendElfSectionHeaders(out, &sec, a, 0)
	if renvoFixedTarget == 0 { return renvoAppendReplLinkTable(out, a) }; return out
}

func renvoAsmPatchAarch64Abs(a *renvoAsm) {
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)); off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1)); kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off; if kind == renvoAbsBssReloc { target = renvoAsmBssOffset(a) + off }
		insn := renvoGet32At(a.code, at); reg := insn & 31; pcPage := (a.codeOffset + at) & -4096; targetPage := target & -4096; pageDelta := (targetPage - pcPage) >> 12
		immlo := pageDelta & 3; immhi := pageDelta >> 2 & 524287
		renvoPut32At(a.code, at, 0x90000000|(immlo<<29)|(immhi<<5)|reg)
		renvoPut32At(a.code, at+4, 0x91000000|((target&4095)<<10)|(reg<<5)|reg)
		renvoPut32At(a.code, at+8, 0xd503201f); renvoPut32At(a.code, at+12, 0xd503201f)
	}
}

func renvoAppendElfHeaderAarch64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := 0; out = append(out, 0x7f, 'E', 'L', 'F', 2, 1, 1, 0)
	for i := 0; i < 8; i++ { out = append(out, 0) }
	out = renvoAppend16(out, 3); out = renvoAppend16(out, 183); out = renvoAppend32(out, 1)
	out = renvoAppend64U32(out, base+entryOff); out = renvoAppend64U32(out, 64); out = renvoAppend64U32(out, shoff); out = renvoAppend32(out, 0)
	out = renvoAppend16(out, 64); out = renvoAppend16(out, 56); out = renvoAppend16(out, 2)
	if shoff == 0 { out = renvoAppend16(out, 0); out = renvoAppend16(out, 0); out = renvoAppend16(out, 0) } else { out = renvoAppend16(out, 64); out = renvoAppend16(out, 7); out = renvoAppend16(out, 6) }
	out = renvoAppend32(out, 1); out = renvoAppend32(out, 5); out = renvoAppend64U32(out, 0); out = renvoAppend64U32(out, base); out = renvoAppend64U32(out, base)
	out = renvoAppend64U32(out, fileSize); out = renvoAppend64U32(out, fileSize); out = renvoAppend64U32(out, 0x1000)
	return renvoAppendElf64LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
}
`
