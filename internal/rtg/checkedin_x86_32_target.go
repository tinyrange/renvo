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
	allRoots := checkedInWindows386SequenceRoots(target.Runtime)
	for _, root := range allRoots {
		if root == "" {
			return checkedInTargetProjectionFailure(document, target.Runtime,
				"windows/386 runtime is missing a required operation sequence")
		}
	}
	identity := HashText(checkedInWindows386RuntimeIdentity(document, target.Arch, allRoots))
	if identity != checkedInWindows386RuntimeIdentityText {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/386 compact runtime templates are stale for definition semantics "+identity)
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	source = appendCheckedInWindows386Runtime(source)
	source = appendCheckedInWindows386Entry(source, template)
	source = append(source, checkedInWindows386ImageSource...)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

const checkedInWindows386RuntimeIdentityText = "ca163c6559929832aee6d96dff7e696a75def9a68bf1ce37834df41b04f2227e"

func checkedInWindows386RuntimeIdentity(
	document Document, arch Declaration, roots []string,
) [32]byte {
	goRoots, sequenceRoots := resolveArchitectureSequenceProjection(
		document, arch, nil, roots)
	var canonical []byte
	canonical = appendFramed(canonical, "windows-386-compact-runtime")
	sequences := targetSemanticSequences(arch, sequenceRoots)
	for i := 0; i < len(sequences); i++ {
		canonical = appendFramed(canonical, sequences[i].name)
		canonical = appendFramedBytes(canonical, sequences[i].body)
	}
	parts := reachableEmbeddedGoParts(document, goRoots, nil)
	for i := 0; i < len(parts); i++ {
		parts[i].source = canonicalTokenStream(parts[i].source)
	}
	for i := 1; i < len(parts); i++ {
		value := parts[i]
		at := i
		for at > 0 && embeddedGoPartLess(value, parts[at-1]) {
			parts[at] = parts[at-1]
			at--
		}
		parts[at] = value
	}
	for i := 0; i < len(parts); i++ {
		canonical = appendFramed(canonical, parts[i].name)
		canonical = appendFramedBytes(canonical, parts[i].source)
	}
	return sha256Bytes(canonical)
}

func checkedInWindows386SequenceRoots(runtime Declaration) []string {
	var roots []string
	for _, field := range []string{"emit_exit", "emit_static_call"} {
		algorithm, _ := architectureGoHook(runtime, field)
		roots = append(roots, algorithm)
	}
	for _, operation := range []string{"open", "close", "chmod", "read", "write", "read_at", "write_at"} {
		algorithm, _ := runtimeOperationHook(runtime, operation)
		roots = append(roots, algorithm)
	}
	return roots
}

func appendCheckedInWindows386Runtime(source []byte) []byte {
	source = append(source, checkedInWindows386OperationsSource...)
	source = append(source, checkedInWindows386ReadWriteSource...)
	source = append(source, `

func compileWindows386(input []int, output int) int {
	return compileWindows386Arena(input, output, 0)
}

func compileWindows386Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindows386)
	return renvoCompile386(input, output, arenaSize)
}
`...)
	return source
}

const checkedInWindows386OperationsSource = `

func renvoWin386CallImport(a *renvoAsm, importID int) {
	base := len(a.code)
	renvoAsmEmitText(a, "\xff\x15\x00\x00\x00\x00")
	renvoAsmAddWinImportReloc(a, base+2, importID)
}

func renvoWin386EmitRuntimeOpen(a *renvoAsm) {
	base := len(a.code)
	renvoAsmEmitText(a, "\xba\x00\x00\x00\x80\xa8\x02\x0f\x84\x0a\x00\x00\x00\xba\x00\x00\x00\xc0\xe9\x0d\x00\x00\x00\xa8\x01\x0f\x84\x05\x00\x00\x00\xba\x00\x00\x00\x40\xb9\x03\x00\x00\x00\xa8\x40\x0f\x84\x1a\x00\x00\x00\xb9\x04\x00\x00\x00\xa9\x00\x02\x00\x00\x0f\x84\x1a\x00\x00\x00\xb9\x02\x00\x00\x00\xe9\x10\x00\x00\x00\xa9\x00\x02\x00\x00\x0f\x84\x05\x00\x00\x00\xb9\x05\x00\x00\x00\x6a\x00\x68\x80\x00\x00\x00\x51\x6a\x00\x6a\x03\x52\x53\xff\x15\x00\x00\x00\x00")
	renvoAsmAddWinImportReloc(a, base+107, 1)
}

func renvoWin386EmitRuntimeClose(a *renvoAsm) {
	base := len(a.code)
	renvoAsmEmitText(a, "\x53\xff\x15\x00\x00\x00\x00\x85\xc0\x0f\x84\x07\x00\x00\x00\x31\xc0\xe9\x05\x00\x00\x00\xb8\xff\xff\xff\xff")
	renvoAsmAddWinImportReloc(a, base+3, 2)
}

func renvoWin386EmitRuntimeChmod(a *renvoAsm) {
	base := len(a.code)
	renvoAsmEmitText(a, "\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xb9\xff\xff\xff\xff\x39\xc8\x0f\x84\x07\x00\x00\x00\x31\xc0\xe9\x05\x00\x00\x00\xb8\xff\xff\xff\xff")
	renvoAsmAddWinImportReloc(a, base+9, 5)
}

func renvoWin386EmitExit(a *renvoAsm) {
	base := len(a.code)
	renvoAsmEmitText(a, "\x50\xff\x15\x00\x00\x00\x00")
	renvoAsmAddWinImportReloc(a, base+3, 8)
}
`

const checkedInWindows386ReadWriteSource = `

func renvoWin386EmitRuntimeReadWrite(g *renvoLinearGen, isWrite bool) {
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
	countOff := renvoAlignValue(a.bssSize, 4)
	a.bssSize = countOff + 16
	base := len(a.code)
	relocs := ""
	if isWrite {
		renvoAsmEmitText(a, "\xe9\xc4\x00\x00\x00\x83\xfb\x01\x0f\x84\x0e\x00\x00\x00\x83\xfb\x02\x0f\x84\x14\x00\x00\x00\xe9\x19\x00\x00\x00\x6a\xf5\xff\x15\x00\x00\x00\x00\x89\xc3\xe9\x0a\x00\x00\x00\x6a\xf4\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x20\x08\x2f\x08\x49\x07\x4e\x01\x5e\x07\x67\x00\x70\x06\x7e\x00\x8a\x00\x93\x06\xa1\x00\xae\x00\xb7\x01\xbf\x07\xc4\x00"
	} else {
		renvoAsmEmitText(a, "\xe9\xac\x00\x00\x00\x83\xfb\x00\x0f\x84\x05\x00\x00\x00\xe9\x0a\x00\x00\x00\x6a\xf6\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x17\x08\x31\x07\x36\x01\x46\x07\x4f\x00\x58\x05\x66\x00\x72\x00\x7b\x05\x89\x00\x96\x00\x9f\x01\xa7\x07\xac\x00"
	}
	for i := 0; i < len(relocs); i += 2 {
		at := base + int(relocs[i])
		kind := int(relocs[i+1])
		if kind < 2 {
			renvoAsmAddAbsReloc(a, at, countOff+(kind<<3), renvoAbsBssReloc)
		} else {
			renvoAsmAddWinImportReloc(a, at, kind-2)
		}
	}
	a.labelPos[label] = int32(base + 5)
	renvoAsmCallLabel(a, label)
}

func renvoWin386EmitRuntimeReadAt(g *renvoLinearGen) {
	renvoWin386EmitRuntimeReadWrite(g, false)
}

func renvoWin386EmitRuntimeWriteAt(g *renvoLinearGen) {
	renvoWin386EmitRuntimeReadWrite(g, true)
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
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
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
