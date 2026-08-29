//go:build !renvo

package rtg

import "bytes"

func generateCheckedInWindows386Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	if declarativeFormatImage(target.Executable) != "pe_executable" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"windows/386 requires a declarative PE executable image")
	}
	bits, hasBits := integerField(document, target.Executable, "address_bits")
	machine, hasMachine := integerField(document, target.Executable, "machine")
	codeOffset, hasCodeOffset := integerField(document, target.Executable, "code_offset")
	fileAlignment, hasFileAlignment := integerField(document, target.Executable, "file_alignment")
	sectionAlignment, hasSectionAlignment := integerField(document, target.Executable, "section_alignment")
	imageBase, hasImageBase := integerField(document, target.Executable, "image_base")
	if !hasBits || bits != 32 || !hasMachine || machine != 0x014c ||
		!hasCodeOffset || codeOffset != 0x1000 || !hasFileAlignment || fileAlignment != 0x200 ||
		!hasSectionAlignment || sectionAlignment != 0x1000 || !hasImageBase || imageBase != 0x400000 {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"windows/386 requires the 32-bit two-section PE layout")
	}
	library, imports := runtimePEImports(target.Runtime)
	expectedImports := []string{
		"CreateFileA", "CloseHandle", "ReadFile", "WriteFile", "SetFilePointer",
		"GetStdHandle", "GetCommandLineA", "ExitProcess", "GetEnvironmentStringsA",
	}
	if library != "kernel32.dll" || len(imports) != len(expectedImports) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/386 requires the stable kernel32 import ABI")
	}
	for i := 0; i < len(imports); i++ {
		if imports[i] != expectedImports[i] {
			return checkedInTargetProjectionFailure(document, target.Runtime,
				"windows/386 requires the stable kernel32 import ABI")
		}
	}
	template, hasTemplate := decodeRuntimeEntryTemplate(target.Runtime)
	if !hasTemplate || !checkedInLiteralTemplateValid(template,
		[]string{"args", "argsText", "argsLength", "environment", "environmentLength"},
		len(expectedImports)) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/386 requires a complete five-buffer literal entry template")
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	projection := document
	projection.Unit = "builtin"
	roots := checkedInWindows386SequenceRoots(target.Runtime)
	if len(roots) != 7 {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/386 runtime is missing a required operation sequence")
	}
	goRoots, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, roots, nil)
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(
		source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInX8632EmitterMethods(source)
	source = rewriteCheckedInWindows386SequenceNames(source)
	source = append(source, checkedInWindows386SequenceAdapters...)
	source = append(source, "\nfunc compileWindows386(input []int, output int) int {\n\treturn compileWindows386Arena(input, output, 0)\n}\n\nfunc compileWindows386Arena(input []int, output int, arenaSize int) int {\n\trenvoSetTarget(renvoTargetWindows386)\n\treturn renvoCompile386(input, output, arenaSize)\n}\n"...)
	source = appendCheckedInWindows386Entry(source, template)
	source = append(source, checkedInWindows386ImageSource...)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

func checkedInWindows386SequenceRoots(runtime Declaration) []string {
	var roots []string
	for _, field := range []string{"emit_exit", "emit_static_call"} {
		if algorithm, found := architectureGoHook(runtime, field); found {
			roots = append(roots, algorithm)
		}
	}
	for _, operation := range []string{"open", "close", "chmod", "read_at", "write_at"} {
		if algorithm, found := runtimeOperationHook(runtime, operation); found {
			roots = append(roots, algorithm)
		}
	}
	return roots
}

func rewriteCheckedInWindows386SequenceNames(source []byte) []byte {
	source = bytes.ReplaceAll(source,
		[]byte("rtgBuiltinWindows386PackageWindows"), []byte("renvoW386"))
	replacements := []string{
		"renvoW386Exit", "renvoW386ExitSequence",
		"renvoW386StaticCall", "renvoW386CallImportSequence",
		"renvoW386RuntimeOpen", "renvoWin386EmitRuntimeOpen",
		"renvoW386RuntimeClose", "renvoWin386EmitRuntimeClose",
		"renvoW386RuntimeChmod", "renvoWin386EmitRuntimeChmod",
		"renvoW386ReadAt", "renvoW386ReadAtSequence",
		"renvoW386WriteAt", "renvoW386WriteAtSequence",
		"out.ReserveBSS(", "renvoW386ReserveBSS(out,",
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
	return source
}

const checkedInWindows386SequenceAdapters = `

func renvoW386ReserveBSS(a *renvoAsm, size int, alignment int) int {
	offset := renvoAlignValue(a.bssSize, alignment)
	a.bssSize = offset + size
	return offset
}

func renvoWin386EmitExit(a *renvoAsm) { renvoW386ExitSequence(a, 0) }

func renvoWin386CallImport(a *renvoAsm, importID int) {
	renvoW386CallImportSequence(a, importID, 0)
}

func renvoWin386EmitRuntimeReadAt(g *renvoLinearGen) {
	if g.winReadEmitted { renvoAsmCallLabel(&g.asm, g.winReadLabel); return }
	g.winReadEmitted = true
	g.winReadLabel = len(g.asm.labelPos)
	renvoW386ReadAtSequence(&g.asm)
}

func renvoWin386EmitRuntimeWriteAt(g *renvoLinearGen) {
	if g.winWriteEmitted { renvoAsmCallLabel(&g.asm, g.winWriteLabel); return }
	g.winWriteEmitted = true
	g.winWriteLabel = len(g.asm.labelPos)
	renvoW386WriteAtSequence(&g.asm)
}
`

func appendCheckedInWindows386Entry(source []byte, template runtimeEntryTemplate) []byte {
	bssSymbols := []string{
		"renvoWindows386ArgsBSS", "renvoWindows386ArgsTextBSS",
		"renvoWindows386ArgsLengthBSS", "renvoWindows386EnvironmentBSS",
		"renvoWindows386EnvironmentLengthBSS",
	}
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
	source = append(source, "\nfunc renvoAsmBuildWindowsArgvEnvSlices386(a *renvoAsm"...)
	for i := 0; i < len(template.bss); i++ {
		source = append(source, ", "...)
		source = append(source, template.bss[i].name...)
		source = append(source, "Off int"...)
	}
	source = append(source, ") {\n\tbase := len(a.code)\n\trenvoAsmEmitText(a, "...)
	source = append(source, template.code...)
	source = append(source, ")\n"...)
	for i := 0; i < len(template.relocations); i++ {
		relocation := template.relocations[i]
		source = append(source, "\trenvoAsmAdd"...)
		if relocation.kind == "import" {
			source = append(source, "WinImportReloc(a, base+"...)
			source = appendDecimalFrame(source, relocation.offset)
			source = append(source, ", "...)
			value, _ := parseInteger(relocation.target)
			source = appendDecimalFrame(source, value)
			source = append(source, ")\n"...)
			continue
		}
		source = append(source, "AbsReloc(a, base+"...)
		source = appendDecimalFrame(source, relocation.offset)
		source = append(source, ", "...)
		source = append(source, relocation.target...)
		source = append(source, "Off, renvoAbsBssReloc)\n"...)
	}
	return append(source, "}\n"...)
}

const checkedInWindows386ImageSource = `

func renvoAsmImageWindows386(a *renvoAsm) []byte {
	for (a.codeOffset+len(a.code))%4 != 0 { a.code = append(a.code, 0) }
	textVirtualSize := len(a.code)
	textRawSize := renvoAlignValue(textVirtualSize, renvoWinFileAlign)
	dataRVA := renvoAlignValue(a.codeOffset+textVirtualSize, renvoWinSectionAlign)
	a.dataOffset = dataRVA
	var imports renvoWinImportLayout
	if renvoAsmHasWinImportRelocs(a) { renvoAppendWinImports(a, &imports) }
	renvoAsmPatchWindows(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 { iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize }
	var out []byte
	out = renvoAppendPEHeader32WithContext(a.c, out, a.codeOffset, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
	out = append(out, a.code...)
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize)
	out = append(out, a.data...)
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize+dataRawSize)
	if renvoFixedTarget == 0 && a.c.emitImage {
		for i := 0; i+2 < len(a.absRelocs); i += 3 {
			at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) & 2147483647
			out = renvoAppend32(out, a.codeOffset+at)
		}
		out = renvoAppend32(out, len(a.absRelocs)/3)
		out = append(out, 'R', 'B', 'R', '1')
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}
`

func generateCheckedInLinux386Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	if declarativeFormatImage(target.Executable) != "elf_executable" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"linux/386 requires a declarative ELF executable image")
	}
	bits, hasBits := integerField(document, target.Executable, "address_bits")
	machine, hasMachine := integerField(document, target.Executable, "machine")
	codeOffset, hasCodeOffset := integerField(document, target.Executable, "code_offset")
	alignment, hasAlignment := integerField(document, target.Executable, "file_alignment")
	patchBias, hasPatchBias := integerField(document, target.Executable, "patch_pc_bias")
	patchAbsolute, hasPatchAbsolute := fieldValue(document, target.Executable, "patch_absolute")
	if !hasBits || bits != 32 || !hasMachine || machine != 3 ||
		!hasCodeOffset || codeOffset != 116 || !hasAlignment || alignment != 0x1000 ||
		!hasPatchBias || patchBias != -3 || !hasPatchAbsolute ||
		valueName(patchAbsolute) != "relative32_le" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"linux/386 requires the 32-bit two-segment ELF layout")
	}
	operations, missingOperation := checkedInLinux386RuntimeOperations(target.Runtime)
	if missingOperation != "" {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/386 runtime operation "+missingOperation+" requires a syscall number")
	}
	template, hasTemplate := decodeRuntimeEntryTemplate(target.Runtime)
	if !hasTemplate || template.algorithm == "" || template.maxParameters != 2 ||
		len(template.bss) != 3 {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/386 requires a complete three-buffer sequence entry")
	}
	entry := template.algorithm
	expectedBSS := []string{"args", "environment", "environmentLength"}
	for i := 0; i < len(template.bss); i++ {
		if template.bss[i].name != expectedBSS[i] || template.bss[i].size <= 0 ||
			template.bss[i].alignment <= 0 {
			return checkedInTargetProjectionFailure(document, target.Runtime,
				"linux/386 requires a complete three-buffer sequence entry")
		}
	}

	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	projection := document
	projection.Unit = "builtin"
	goRoots, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, nil, []string{entry})
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(
		source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInX8632EmitterMethods(source)
	source = appendCheckedInLinux386Runtime(source, operations, projection, entry, template)
	source = append(source, checkedInLinux386ImageSource...)
	source = bytes.ReplaceAll(source, []byte("const renvoLinux386ELFMachine = 3"),
		[]byte("const renvoLinux386ELFMachine = "+decimalFrame(machine)))
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

func checkedInLinux386RuntimeOperations(
	runtime Declaration,
) ([]checkedInLinuxAmd64RuntimeOperation, string) {
	operations := []checkedInLinuxAmd64RuntimeOperation{
		{name: "read", symbol: "renvoLinux386SysReadSeq"},
		{name: "write", symbol: "renvoLinux386SysWriteSeq"},
		{name: "open", symbol: "renvoLinux386SysOpen"},
		{name: "close", symbol: "renvoLinux386SysClose"},
		{name: "read_at", symbol: "renvoLinux386SysReadAt"},
		{name: "write_at", symbol: "renvoLinux386SysWriteAt"},
		{name: "chmod", symbol: "renvoLinux386SysFchmod"},
		{name: "exit", symbol: "renvoLinux386SysExit"},
	}
	for i := 0; i < len(operations); i++ {
		value, ok := runtimeOperationInteger(runtime, operations[i].name, "number")
		if !ok || value < 0 {
			return nil, operations[i].name
		}
		operations[i].value = value
	}
	return operations, ""
}

func rewriteCheckedInX8632EmitterMethods(source []byte) []byte {
	replacements := []string{
		"out.NewLabel()", "renvoAsmNewLabel(out)",
		"out.Mark(", "renvoAsmMarkLabel(out, ",
		"rtgBuiltinJump(out,", "renvoAsmJmpLabel(out, ",
		"rtgBuiltinCall(out,", "renvo386TargetCall(out,",
		"rtgBuiltinMove(out,", "renvo386TargetMove(out,",
		"rtgBuiltinTest(out,", "renvo386TargetTest(out,",
		"rtgBuiltinEAX", "0",
		"rtgBuiltinECX", "1",
		"rtgBuiltinEDX", "2",
		"rtgBuiltinEBX", "3",
		"rtgBuiltinESP", "4",
		"rtgBuiltinEBP", "5",
		"rtgBuiltinESI", "6",
		"rtgBuiltinEDI", "7",
		"renvo386RTGEmitModRM", "renvo386TargetEmitModRM",
		"renvo386RTGEncodeBinary", "renvo386TargetEncodeBinary",
		"renvo386RTGEncodeRelative", "renvo386TargetEncodeRelative",
		"renvo386RTGMoveImmediate", "renvo386TargetMoveImmediate",
		"renvo386RTGRegisterLowBits", "renvo386TargetRegisterLowBits",
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
	for i := 0; i < 8; i++ {
		compact := "RTGRegister{Code:" + decimalFrame(i) + ",Valid:true}"
		spaced := "RTGRegister{Code: " + decimalFrame(i) + ", Valid: true}"
		source = bytes.ReplaceAll(source, []byte(compact), []byte(decimalFrame(i)))
		source = bytes.ReplaceAll(source, []byte(spaced), []byte(decimalFrame(i)))
	}
	source = bytes.ReplaceAll(source, []byte("RTGRegister"), []byte("int"))
	source = bytes.ReplaceAll(source, []byte(".Code"), nil)
	return rewriteCheckedInX8632JumpConditions(source)
}

func rewriteCheckedInX8632JumpConditions(source []byte) []byte {
	prefixes := [][]byte{
		[]byte("rtgBuiltinX8632PackageJumpCondition(out,RTGCondition{JumpOpcode:"),
		[]byte("renvo386RTGJumpCondition(out,RTGCondition{JumpOpcode:"),
	}
	for _, prefix := range prefixes {
		for {
			start := bytes.Index(source, prefix)
			if start < 0 {
				break
			}
			conditionStart := start + len(prefix)
			conditionEnd := bytes.Index(source[conditionStart:], []byte("},"))
			if conditionEnd < 0 {
				break
			}
			conditionEnd += conditionStart
			rewritten := make([]byte, 0, len(source))
			rewritten = append(rewritten, source[:start]...)
			rewritten = append(rewritten, "renvo386AsmJccLabel(out,"...)
			rewritten = append(rewritten, source[conditionStart:conditionEnd]...)
			rewritten = append(rewritten, ',')
			rewritten = append(rewritten, source[conditionEnd+2:]...)
			source = rewritten
		}
	}
	return source
}

func appendCheckedInLinux386Runtime(
	source []byte, operations []checkedInLinuxAmd64RuntimeOperation,
	document Document, entry string, template runtimeEntryTemplate,
) []byte {
	for i := 0; i < len(operations); i++ {
		source = append(source, "\nconst "...)
		source = append(source, operations[i].symbol...)
		source = append(source, " = "...)
		source = appendDecimalFrame(source, operations[i].value)
	}
	source = append(source, `

func renvo386AsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoAsmPushTertiary(a)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmPopSecondary(a)
}

func renvo386AsmMoveOffsetArg(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc689)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit16(a, 0xc789)
}

func compileLinux386(input []int, output int) int {
	return compileLinux386Arena(input, output, 0)
}

func compileLinux386Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinux386)
	return renvoCompile386(input, output, arenaSize)
}
`...)
	bssSymbols := []string{"renvoLinux386ArgsBSS", "renvoLinux386EnvironmentBSS", "renvoLinux386EnvironmentLengthBSS"}
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
	source = append(source, "\n\nfunc renvoAsmBuildArgvEnvSlices386(a *renvoAsm, argsOff int, environmentOff int, environmentLengthOff int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, entry)...)
	source = append(source, "(a, argsOff, environmentOff, environmentLengthOff)\n}\n"...)
	return source
}

const checkedInLinux386ImageSource = `

const renvoLinux386ELFMachine = 3

func renvoAsmImage386(a *renvoAsm) []byte {
	renvoAsmPatch386(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if a.c.stripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeader386(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		out = append(out, a.code...)
		out = append(out, a.data...)
		if renvoFixedTarget == 0 { return renvoAppendReplLinkTable(out, a) }
		return out
	}
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	out := make([]byte, 0, sec.shoff+280)
	out = renvoAppendElfHeader386(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
	out = append(out, a.code...)
	out = append(out, a.data...)
	out = renvoAppendUntil(out, sec.symtabOff)
	out = append(out, sec.symtab...)
	out = renvoAppendUntil(out, sec.strtabOff)
	out = append(out, sec.strtab...)
	out = renvoAppendUntil(out, sec.shstrOff)
	out = append(out, sec.shstrtab...)
	out = renvoAppendUntil(out, sec.shoff)
	out = renvoAppendElfSectionHeaders(out, &sec, a, 0)
	if renvoFixedTarget == 0 { return renvoAppendReplLinkTable(out, a) }
	return out
}

func renvoAsmPatch386(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc { target = renvoAsmBssOffset(a) + off }
		renvoPut32At(a.code, at, target-(a.codeOffset+at-3))
	}
}

func renvoAppendElfHeader386(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := 0
	out = append(out, 0x7f, 'E', 'L', 'F', 1, 1, 1, 0)
	for i := 0; i < 8; i++ { out = append(out, 0) }
	out = renvoAppend16(out, 3)
	out = renvoAppend16(out, renvoLinux386ELFMachine)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, base+entryOff)
	out = renvoAppend32(out, 52)
	out = renvoAppend32(out, shoff)
	out = renvoAppend32(out, 0)
	out = renvoAppend16(out, 52)
	out = renvoAppend16(out, 32)
	out = renvoAppend16(out, 2)
	if shoff == 0 {
		out = renvoAppend16(out, 0); out = renvoAppend16(out, 0); out = renvoAppend16(out, 0)
	} else {
		out = renvoAppend16(out, 40); out = renvoAppend16(out, 7); out = renvoAppend16(out, 6)
	}
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, base)
	out = renvoAppend32(out, base)
	out = renvoAppend32(out, fileSize)
	out = renvoAppend32(out, fileSize)
	out = renvoAppend32(out, 5)
	out = renvoAppend32(out, 0x1000)
	out = renvoAppendElf32LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
	return out
}
`
