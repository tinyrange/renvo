//go:build !renvo

package rtg

import (
	"bytes"
	"strconv"
)

func generateCheckedInWindowsAmd64Projection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	if declarativeFormatImage(target.Executable) != "pe_executable" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"windows/amd64 requires a declarative PE executable image")
	}
	bits, hasBits := integerField(document, target.Executable, "address_bits")
	machine, hasMachine := integerField(document, target.Executable, "machine")
	codeOffset, hasCodeOffset := integerField(document, target.Executable, "code_offset")
	fileAlignment, hasFileAlignment := integerField(
		document, target.Executable, "file_alignment")
	sectionAlignment, hasSectionAlignment := integerField(
		document, target.Executable, "section_alignment")
	imageBase, hasImageBase := integerField(document, target.Executable, "image_base")
	headersSize, hasHeadersSize := integerField(document, target.Executable, "headers_size")
	textRVA, hasTextRVA := integerField(document, target.Executable, "text_rva")
	stackReserve := formatIntegerDefault(document, target.Executable, "stack_reserve", 0)
	stackCommit := formatIntegerDefault(document, target.Executable, "stack_commit", 0)
	heapReserve := formatIntegerDefault(document, target.Executable, "heap_reserve", 0)
	heapCommit := formatIntegerDefault(document, target.Executable, "heap_commit", 0)
	if !hasBits || bits != 64 || !hasMachine || machine != 0x8664 ||
		!hasCodeOffset || codeOffset != 0x1000 ||
		!hasFileAlignment || fileAlignment != 0x200 ||
		!hasSectionAlignment || sectionAlignment != 0x1000 ||
		!hasImageBase || imageBase != 0x400000 ||
		!hasHeadersSize || headersSize != 0x200 ||
		!hasTextRVA || textRVA != 0x1000 ||
		stackReserve != 0x800000 || stackCommit != 0x1000 ||
		heapReserve != 0x100000 || heapCommit != 0x1000 ||
		declarativeFormatImageRelocations(target.Executable) != "pe_relative32" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"windows/amd64 requires the 64-bit two-section PE layout")
	}
	library, imports := runtimePEImports(target.Runtime)
	expectedImports := []string{
		"CreateFileA", "CloseHandle", "ReadFile", "WriteFile", "SetFilePointer",
		"GetStdHandle", "GetCommandLineA", "ExitProcess", "GetEnvironmentStringsA",
	}
	if library != "kernel32.dll" || len(imports) != len(expectedImports) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/amd64 requires the stable kernel32 import ABI")
	}
	for i := 0; i < len(imports); i++ {
		if imports[i] != expectedImports[i] {
			return checkedInTargetProjectionFailure(document, target.Runtime,
				"windows/amd64 requires the stable kernel32 import ABI")
		}
	}
	template, hasTemplate := decodeRuntimeEntryTemplate(target.Runtime)
	if !hasTemplate || !checkedInWindowsAmd64EntryTemplateValid(template) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/amd64 requires a complete five-buffer literal entry template")
	}
	ioTemplate, hasIOTemplate := decodeRuntimeIOTemplate(target.Runtime)
	if !hasIOTemplate || !checkedInWindowsAmd64IOTemplateValid(
		ioTemplate, len(expectedImports)) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/amd64 requires a complete read/write runtime template")
	}

	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	source = append(source, checkedInWindowsAmd64APISource...)
	projection := document
	projection.Unit = "builtin"
	goRoots := checkedInWindowsAmd64GoRoots(target.Runtime)
	sequenceRoots := checkedInWindowsAmd64SequenceRoots(target.Runtime)
	goRoots, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, goRoots, sequenceRoots)
	exports := architectureExports(target.Arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, goRoots, true, exports, excluded)
	source = appendArchitectureSequences(
		source, projection, target.Arch, true, false, true, sequences)
	source = rewriteCheckedInWindowsAmd64RegisterLiterals(source)
	callImport, hasCallImport := checkedInSequenceWithSuffix(
		target.Arch, "WindowsCallImport")
	if !hasCallImport {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"windows/amd64 requires its aligned import-call sequence")
	}
	source = appendCheckedInWindowsAmd64Runtime(
		source, projection, target.Runtime, ioTemplate, callImport)
	source = appendCheckedInWindowsAmd64Entry(source, template)
	source = append(source, checkedInWindowsAmd64ImageSource...)
	source = rewriteCheckedInNativeEmitterMethods(source)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

const checkedInWindowsAmd64APISource = `

type renvoWindowsAmd64Register struct {
	Code int
	Valid bool
}

var renvoWindowsAmd64RaxRegister = renvoWindowsAmd64Register{Code: 0, Valid: true}

const renvoWindowsAmd64RelocationAbsoluteBSS = 1
const renvoWindowsAmd64RelocationImport = 2

func renvoWindowsAmd64Rel32(out *renvoAsm, label int) {
	at := len(out.code)
	renvoAsmEmit32(out, 0)
	if label >= 0 {
		renvoAsmAddReloc(out, at, label)
	}
}
`

const checkedInWindowsAmd64ImageSource = `

func compileWindowsAmd64(input []int, output int) int {
	return compileWindowsAmd64Arena(input, output, 0)
}

func compileWindowsAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindowsAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}

// renvoAsmImageWindowsAmd64 is the compact production specialization of the
// validated declarative PE64 layout. Prepared backends retain the generic
// definition-driven image builder.
func renvoAsmImageWindowsAmd64(a *renvoAsm) []byte {
	renvoNonNil(a)
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
	renvoAsmPatchWindows(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = renvoAppendPEHeader64WithContext(a.c, out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize+dataRawSize)
	if renvoFixedTarget == 0 && a.c.emitImage {
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}
`

func rewriteCheckedInNativeEmitterMethods(source []byte) []byte {
	replacements := []string{
		"out.NewLabel()", "renvoAsmNewLabel(out)",
		"out.Mark(", "renvoAsmMarkLabel(out, ",
		"out.Code()", "out.code",
		"out.SetCode(code)", "out.code = code",
		"out.Data()", "out.data",
		"out.SetData(data)", "out.data = data",
		"out.BSSSize()", "out.bssSize",
		"out.WindowsSubsystem()", "out.c.windowsSubsystem",
		"out.StaticImportCount()", "len(out.staticImports)",
		"out.StaticImportDLL(i)", "out.staticImports[i].dll",
		"out.StaticImportName(i)", "out.staticImports[i].name",
		"out.StaticImportDLL(index)", "out.staticImports[index].dll",
		"out.StaticImportName(index)", "out.staticImports[index].name",
		"out.AbsoluteRelocationCount()", "len(out.absRelocs)/3",
		"out.AbsoluteRelocationOffset(i)",
		"int(renvo_runtime_UnsafeInt32At(out.absRelocs, i*3))",
		"out.AbsoluteRelocationAddend(i)",
		"int(renvo_runtime_UnsafeInt32At(out.absRelocs, i*3+1))",
		"out.AbsoluteRelocationKind(i)",
		"int(renvo_runtime_UnsafeInt32At(out.absRelocs, i*3+2))",
		"out.PatchUint32(at, ", "renvoPut32At(out.code, at, ",
		"RTGRegister", "renvoWindowsAmd64Register",
		"RTGRelocationAbsoluteBSS", "renvoWindowsAmd64RelocationAbsoluteBSS",
		"RTGRelocationImport", "renvoWindowsAmd64RelocationImport",
		"renvoRTGRel32", "renvoWindowsAmd64Rel32",
		"renvoRTGResolveAbsoluteRelocation(",
		"renvoWindowsAmd64ResolveAbsoluteRelocation(",
	}
	for i := 0; i < len(replacements); i += 2 {
		source = bytes.ReplaceAll(
			source, []byte(replacements[i]), []byte(replacements[i+1]))
	}
	return source
}

func rewriteCheckedInWindowsAmd64RegisterLiterals(source []byte) []byte {
	registers := []string{
		"Rax", "Rcx", "Rdx", "Rbx", "Rsp", "Rbp", "Rsi", "Rdi",
		"R8", "R9", "R10", "R11", "R12", "R13", "R14", "R15",
	}
	for i := 0; i < len(registers); i++ {
		name := "renvoWindowsAmd64" + registers[i] + "Register"
		compact := "RTGRegister{Code:" + decimalFrame(i) + ",Valid:true}"
		spaced := "RTGRegister{Code: " + decimalFrame(i) + ", Valid: true}"
		source = bytes.ReplaceAll(source, []byte(compact), []byte(name))
		source = bytes.ReplaceAll(source, []byte(spaced), []byte(name))
	}
	return source
}

func appendCheckedInWindowsAmd64Registers(source []byte) []byte {
	return append(source,
		"\nvar renvoWindowsAmd64RaxRegister = RTGRegister{Code: 0, Valid: true}\n"...)
}

func checkedInSequenceWithSuffix(
	arch Declaration, suffix string,
) (string, bool) {
	sequences := architectureSequences(arch)
	for i := 0; i < len(sequences); i++ {
		name := exportedName(sequences[i].Name)
		if len(name) >= len(suffix) && name[len(name)-len(suffix):] == suffix {
			return sequences[i].Name, true
		}
	}
	return "", false
}

func checkedInWindowsAmd64GoRoots(runtime Declaration) []string {
	var roots []string
	for _, field := range []string{"emit_exit", "emit_static_call"} {
		if algorithm, found := architectureGoHook(runtime, field); found {
			roots = append(roots, algorithm)
		}
	}
	for _, operation := range []string{"open", "close", "chmod"} {
		if algorithm, found := runtimeOperationHook(runtime, operation); found {
			roots = append(roots, algorithm)
		}
	}
	return roots
}

func checkedInWindowsAmd64SequenceRoots(runtime Declaration) []string {
	var roots []string
	for _, operation := range []string{"open", "close", "chmod"} {
		if algorithm, found := runtimeOperationHook(runtime, operation); found {
			roots = append(roots, algorithm)
		}
	}
	return roots
}

func checkedInWindowsAmd64EntryTemplateValid(template runtimeEntryTemplate) bool {
	expectedBSS := []string{
		"args", "argsText", "argsLength", "environment", "environmentLength",
	}
	return checkedInLiteralTemplateValid(template, expectedBSS, 9)
}

func checkedInWindowsAmd64IOTemplateValid(
	template runtimeIOTemplate, importCount int,
) bool {
	if len(template.readCode) == 0 || len(template.writeCode) == 0 ||
		len(template.bss) != 2 || template.bss[0].name != "count" ||
		template.bss[1].name != "position" {
		return false
	}
	for i := 0; i < len(template.bss); i++ {
		if template.bss[i].size <= 0 || template.bss[i].alignment <= 0 ||
			template.bss[i].alignment&(template.bss[i].alignment-1) != 0 {
			return false
		}
	}
	return checkedInTemplateRelocationsValid(
		template.readCode, template.readRelocations, template.bss, importCount) &&
		checkedInTemplateRelocationsValid(
			template.writeCode, template.writeRelocations, template.bss, importCount)
}

func checkedInLiteralTemplateValid(
	template runtimeEntryTemplate, expectedBSS []string, importCount int,
) bool {
	if template.algorithm != "" || len(template.code) == 0 ||
		template.maxParameters != 2 || template.requiresState ||
		len(template.bss) != len(expectedBSS) || len(template.relocations) == 0 {
		return false
	}
	for i := 0; i < len(template.bss); i++ {
		bss := template.bss[i]
		if bss.name != expectedBSS[i] || bss.size <= 0 || bss.alignment <= 0 ||
			bss.alignment&(bss.alignment-1) != 0 {
			return false
		}
	}
	return checkedInTemplateRelocationsValid(
		template.code, template.relocations, template.bss, importCount)
}

func checkedInTemplateRelocationsValid(
	code string, relocations []runtimeEntryRelocation,
	bss []runtimeEntryBSS, importCount int,
) bool {
	for i := 0; i < len(relocations); i++ {
		relocation := relocations[i]
		if relocation.offset < 0 || relocation.offset+4 > len(code) {
			return false
		}
		if relocation.kind == "import" {
			value, ok := parseInteger(relocation.target)
			if !ok || value <= 0 || value > importCount {
				return false
			}
			continue
		}
		if relocation.kind != "bss" {
			return false
		}
		found := false
		for j := 0; j < len(bss); j++ {
			if relocation.target == bss[j].name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func checkedInProjectionAlgorithmName(document Document, algorithm string) string {
	return "rtg" + exportedName(document.Unit) + exportedName(algorithm)
}

func appendCheckedInWindowsAmd64Runtime(
	source []byte, document Document, runtime Declaration, template runtimeIOTemplate,
	callImport string,
) []byte {
	exit, _ := architectureGoHook(runtime, "emit_exit")
	staticCall, _ := architectureGoHook(runtime, "emit_static_call")
	open, _ := runtimeOperationHook(runtime, "open")
	closeOperation, _ := runtimeOperationHook(runtime, "close")
	chmod, _ := runtimeOperationHook(runtime, "chmod")
	source = append(source, "\nfunc renvoWinAmd64EmitExit(a *renvoAsm) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, exit)...)
	source = append(source, "(a, renvoWindowsAmd64RaxRegister)\n}\n"...)
	source = append(source, "\nfunc renvoWinAmd64CallImport(a *renvoAsm, importID int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, callImport)...)
	source = append(source, "(a, importID)\n}\n"...)
	source = append(source, "\nfunc renvoWinAmd64CallStaticImport(a *renvoAsm, importID int, wordCount int) {\n\t"...)
	source = append(source, checkedInProjectionAlgorithmName(document, staticCall)...)
	source = append(source, "(a, importID, wordCount)\n}\n"...)
	for _, operation := range []struct {
		name      string
		algorithm string
	}{
		{name: "Open", algorithm: open},
		{name: "Close", algorithm: closeOperation},
		{name: "Chmod", algorithm: chmod},
	} {
		source = append(source, "\nfunc renvoWinAmd64EmitRuntime"...)
		source = append(source, operation.name...)
		source = append(source, "(a *renvoAsm) {\n\t"...)
		source = append(source, checkedInProjectionAlgorithmName(document, operation.algorithm)...)
		source = append(source, "(a)\n}\n"...)
	}
	source = append(source, "\nfunc renvoWinAmd64EmitReadWriteHelper(g *renvoLinearGen, isWrite bool) int {\n"...)
	source = append(source, "\trenvoNonNil(g)\n\ta := &g.asm\n\tif isWrite {\n\t\tif g.winWriteEmitted { return g.winWriteLabel }\n\t} else if g.winReadEmitted { return g.winReadLabel }\n"...)
	source = append(source, "\tlabel := renvoAsmNewLabel(a)\n\tif isWrite { g.winWriteEmitted = true; g.winWriteLabel = label } else { g.winReadEmitted = true; g.winReadLabel = label }\n"...)
	for i := 0; i < len(template.bss); i++ {
		source = append(source, '\t')
		source = append(source, template.bss[i].name...)
		source = append(source, "Off := renvoAlignValue(a.bssSize, "...)
		source = appendDecimalFrame(source, template.bss[i].alignment)
		source = append(source, ")\n\ta.bssSize = "...)
		source = append(source, template.bss[i].name...)
		source = append(source, "Off + "...)
		source = appendDecimalFrame(source, template.bss[i].size)
		source = append(source, '\n')
	}
	return appendCheckedInWindowsAmd64IOBody(source, template)
}

func appendCheckedInWindowsAmd64IOBody(
	source []byte, template runtimeIOTemplate,
) []byte {
	readCode, _ := strconv.Unquote(template.readCode)
	writeCode, _ := strconv.Unquote(template.writeCode)
	commonLength := 0
	for commonLength < len(readCode) && commonLength < len(writeCode) &&
		readCode[len(readCode)-1-commonLength] == writeCode[len(writeCode)-1-commonLength] {
		commonLength++
	}
	for commonLength > 0 {
		readStart := len(readCode) - commonLength
		writeStart := len(writeCode) - commonLength
		readRelocations := checkedInRuntimeCommonRelocations(
			template.readRelocations, readStart)
		writeRelocations := checkedInRuntimeCommonRelocations(
			template.writeRelocations, writeStart)
		compatible := len(readRelocations) == len(writeRelocations)
		for i := 0; compatible && i < len(readRelocations); i++ {
			compatible = readRelocations[i].offset-readStart ==
				writeRelocations[i].offset-writeStart &&
				readRelocations[i].kind == writeRelocations[i].kind
		}
		if compatible {
			break
		}
		commonLength--
	}
	readStart := len(readCode) - commonLength
	writeStart := len(writeCode) - commonLength
	source = append(source, "\tcode := "...)
	source = appendQuoted(source, readCode[:readStart])
	source = append(source, "\n\tcodeSize := "...)
	source = appendDecimalFrame(source, len(readCode))
	source = append(source, "\n\tif isWrite { code = "...)
	source = appendQuoted(source, writeCode[:writeStart])
	source = append(source, "; codeSize = "...)
	source = appendDecimalFrame(source, len(writeCode))
	source = append(source, " }\n\tbase := len(a.code)\n\trenvoAsmEmit8(a, 0xe9)\n\trenvoAsmEmit32(a, codeSize)\n\ta.labelPos[label] = int32(base + 5)\n\trenvoAsmEmitText(a, code)\n\tcommonBase := len(a.code)\n\trenvoAsmEmitText(a, "...)
	source = appendQuoted(source, readCode[readStart:])
	source = append(source, ")\n"...)

	source = append(source, "\tif isWrite {\n"...)
	for i := 0; i < len(template.writeRelocations); i++ {
		if template.writeRelocations[i].offset < writeStart {
			source = appendCheckedInWindowsAmd64Relocation(
				source, template.writeRelocations[i], template.bss, "base + 5 + ", 0)
		}
	}
	source = append(source, "\t} else {\n"...)
	for i := 0; i < len(template.readRelocations); i++ {
		if template.readRelocations[i].offset < readStart {
			source = appendCheckedInWindowsAmd64Relocation(
				source, template.readRelocations[i], template.bss, "base + 5 + ", 0)
		}
	}
	source = append(source, "\t}\n"...)

	readCommon := checkedInRuntimeCommonRelocations(template.readRelocations, readStart)
	writeCommon := checkedInRuntimeCommonRelocations(template.writeRelocations, writeStart)
	for i := 0; i < len(readCommon) && i < len(writeCommon); i++ {
		read := readCommon[i]
		write := writeCommon[i]
		if read.offset-readStart != write.offset-writeStart || read.kind != write.kind {
			continue
		}
		if read.target == write.target && read.addend == write.addend {
			source = appendCheckedInWindowsAmd64Relocation(
				source, read, template.bss, "commonBase + ", readStart)
			continue
		}
		source = append(source, "\tif isWrite {\n"...)
		source = appendCheckedInWindowsAmd64Relocation(
			source, write, template.bss, "commonBase + ", writeStart)
		source = append(source, "\t} else {\n"...)
		source = appendCheckedInWindowsAmd64Relocation(
			source, read, template.bss, "commonBase + ", readStart)
		source = append(source, "\t}\n"...)
	}
	return append(source, "\treturn label\n}\n"...)
}

func checkedInRuntimeCommonRelocations(
	relocations []runtimeEntryRelocation, start int,
) []runtimeEntryRelocation {
	var result []runtimeEntryRelocation
	for i := 0; i < len(relocations); i++ {
		if relocations[i].offset >= start {
			result = append(result, relocations[i])
		}
	}
	return result
}

func appendCheckedInWindowsAmd64Relocation(
	source []byte, relocation runtimeEntryRelocation, bss []runtimeEntryBSS,
	base string, subtract int,
) []byte {
	source = append(source, "\t\trenvoAsmAddAbsReloc(a, "...)
	source = append(source, base...)
	source = appendDecimalFrame(source, relocation.offset-subtract)
	source = append(source, ", "...)
	if relocation.kind == "import" {
		value, _ := parseInteger(relocation.target)
		source = appendDecimalFrame(source, value)
		source = append(source, ", RTGRelocationImport)\n"...)
		return source
	}
	for i := 0; i < len(bss); i++ {
		if relocation.target == bss[i].name {
			source = append(source, bss[i].name...)
			source = append(source, "Off"...)
			break
		}
	}
	if relocation.addend != 0 {
		source = append(source, '+')
		source = appendDecimalFrame(source, relocation.addend)
	}
	return append(source, ", RTGRelocationAbsoluteBSS)\n"...)
}

func appendRuntimeRelocationTable(
	source []byte, relocations []runtimeEntryRelocation, bss []runtimeEntryBSS,
) []byte {
	source = append(source, "[]int{"...)
	for i := 0; i < len(relocations); i++ {
		if i != 0 {
			source = append(source, ',')
		}
		source = appendDecimalFrame(source, relocations[i].offset)
		source = append(source, ',')
		if relocations[i].kind == "import" {
			value, _ := parseInteger(relocations[i].target)
			source = appendDecimalFrame(source, value)
			source = append(source, ",RTGRelocationImport"...)
			continue
		}
		for j := 0; j < len(bss); j++ {
			if relocations[i].target == bss[j].name {
				source = append(source, bss[j].name...)
				source = append(source, "Off"...)
				break
			}
		}
		if relocations[i].addend != 0 {
			source = append(source, '+')
			source = appendDecimalFrame(source, relocations[i].addend)
		}
		source = append(source, ",RTGRelocationAbsoluteBSS"...)
	}
	return append(source, '}')
}

func appendCheckedInWindowsAmd64Entry(
	source []byte, template runtimeEntryTemplate,
) []byte {
	bssSymbols := []string{
		"renvoWindowsAmd64ArgsBSS", "renvoWindowsAmd64ArgsTextBSS",
		"renvoWindowsAmd64ArgsLengthBSS", "renvoWindowsAmd64EnvironmentBSS",
		"renvoWindowsAmd64EnvironmentLengthBSS",
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
	source = append(source, "\n\nfunc renvoAsmBuildWindowsArgvEnvSlicesAmd64(a *renvoAsm"...)
	for i := 0; i < len(template.bss); i++ {
		source = append(source, ", "...)
		source = append(source, template.bss[i].name...)
		source = append(source, "Off int"...)
	}
	source = append(source, ") {\n\trenvoNonNil(a)\n\tbase := len(a.code)\n\trenvoAsmEmitText(a, "...)
	source = append(source, template.code...)
	source = append(source, ")\n"...)
	for i := 0; i < len(template.relocations); i++ {
		relocation := template.relocations[i]
		source = append(source, "\trenvoAsmAddAbsReloc(a, base+"...)
		source = appendDecimalFrame(source, relocation.offset)
		source = append(source, ", "...)
		if relocation.kind == "import" {
			value, _ := parseInteger(relocation.target)
			source = appendDecimalFrame(source, value)
			source = append(source, ", RTGRelocationImport)\n"...)
			continue
		}
		source = append(source, relocation.target...)
		source = append(source, "Off, RTGRelocationAbsoluteBSS)\n"...)
	}
	return append(source, "}\n"...)
}
