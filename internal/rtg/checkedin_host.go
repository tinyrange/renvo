//go:build !renvo

package rtg

// GenerateCheckedInTargetProjection emits the target-owned production glue
// used by a built-in compiler. Unlike GeneratePreparedBackend, this projection
// does not emit the stateful RTG adapter or another copy of the ISA contract;
// it calls the checked-in architecture algorithms directly.
func GenerateCheckedInTargetProjection(
	resolved ResolveResult, targetName string, packageName string,
) GenerateResult {
	ensureDirectEmitterV1()
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	target, ok := lookupResolvedTarget(resolved, targetName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-001",
			Message:  "definition does not export target " + targetName,
		}}}
	}
	if target.Descriptor.Name == "windows/amd64" && target.Arch.Name == "x86_64" {
		return generateCheckedInWindowsAmd64Projection(
			resolved, target, packageName)
	}
	if target.Descriptor.Name == "linux-kernel/amd64" && target.Arch.Name == "x86_64" {
		return generateCheckedInLinuxKernelAmd64Projection(
			resolved, target, packageName)
	}
	if (target.Descriptor.Name == "freebsd/amd64" ||
		target.Descriptor.Name == "openbsd/amd64" ||
		target.Descriptor.Name == "netbsd/amd64") && target.Arch.Name == "x86_64" {
		return generateCheckedInBSDAmd64Projection(resolved, target, packageName)
	}
	if target.Descriptor.Name != "linux/amd64" || target.Arch.Name != "x86_64" {
		return checkedInTargetProjectionFailure(resolved.Document, target.Declaration,
			"checked-in production projection is not implemented for "+target.Descriptor.Name)
	}
	productionImage, hasProductionImage := fieldValue(
		resolved.Document, target.Executable, "production_image")
	if !hasProductionImage || valueName(productionImage) != "elf_executable_symbols" {
		return checkedInTargetProjectionFailure(resolved.Document, target.Executable,
			"linux/amd64 requires production_image elf_executable_symbols")
	}
	prepareBuffer, hasPrepareBuffer := architectureGoHook(
		target.Runtime, "prepare_read_write_buffer")
	moveOffset, hasMoveOffset := architectureGoHook(
		target.Runtime, "move_offset_argument")
	if !hasPrepareBuffer || !hasMoveOffset {
		return checkedInTargetProjectionFailure(resolved.Document, target.Runtime,
			"linux/amd64 runtime is missing its syscall ABI sequences")
	}
	operations, missingOperation := checkedInLinuxAmd64RuntimeOperations(target.Runtime)
	if missingOperation != "" {
		return checkedInTargetProjectionFailure(resolved.Document, target.Runtime,
			"linux/amd64 runtime operation "+missingOperation+" requires a syscall number")
	}
	template, hasTemplate := decodeRuntimeEntryTemplate(target.Runtime)
	if !hasTemplate || !checkedInLinuxAmd64EntryTemplateValid(template) {
		return checkedInTargetProjectionFailure(resolved.Document, target.Runtime,
			"linux/amd64 requires a complete three-buffer literal entry template")
	}
	codeOffset, hasCodeOffset := integerField(
		resolved.Document, target.Executable, "code_offset")
	machine, hasMachine := integerField(resolved.Document, target.Executable, "machine")
	addressBits, hasAddressBits := integerField(
		resolved.Document, target.Executable, "address_bits")
	alignment, hasAlignment := integerField(
		resolved.Document, target.Executable, "file_alignment")
	if !hasCodeOffset || codeOffset != 176 || !hasMachine || !hasAddressBits ||
		addressBits != 64 || !hasAlignment || alignment != 0x1000 {
		return checkedInTargetProjectionFailure(resolved.Document, target.Executable,
			"linux/amd64 production ELF requires the 64-bit two-segment layout")
	}

	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(
		manifest, "target-projection/"+target.Descriptor.Name, packageName)
	projection := resolved.Document
	projection.Unit = "builtin"
	_, sequences := resolveArchitectureSequenceProjection(
		projection, target.Arch, nil, []string{prepareBuffer, moveOffset})
	source = appendArchitectureSequences(
		source, projection, target.Arch, true, false, false, sequences)
	source = appendCheckedInLinuxAmd64Runtime(
		source, operations, projection.Unit, prepareBuffer, moveOffset)
	source = appendCheckedInLinuxAmd64Entry(source, template)
	source = append(source, "\nconst renvoLinuxAmd64ELFMachine = "...)
	source = appendDecimalFrame(source, machine)
	source = append(source, checkedInLinuxAmd64ImageSource...)
	return GenerateResult{
		Source: source, Descriptor: target.Descriptor, Manifest: manifest, Ok: true,
	}
}

func checkedInTargetProjectionFailure(
	document Document, declaration Declaration, message string,
) GenerateResult {
	return GenerateResult{Diagnostics: []Diagnostic{documentDiagnostic(
		document, declaration.Span, "RTG-GENERATE-006", message)}}
}

type checkedInLinuxAmd64RuntimeOperation struct {
	name   string
	symbol string
	value  int
}

func checkedInLinuxAmd64RuntimeOperations(
	runtime Declaration,
) ([]checkedInLinuxAmd64RuntimeOperation, string) {
	operations := []checkedInLinuxAmd64RuntimeOperation{
		{name: "read", symbol: "renvoLinuxAmd64SysReadSeq"},
		{name: "write", symbol: "renvoLinuxAmd64SysWriteSeq"},
		{name: "open", symbol: "renvoLinuxAmd64SysOpen"},
		{name: "close", symbol: "renvoLinuxAmd64SysClose"},
		{name: "read_at", symbol: "renvoLinuxAmd64SysReadAt"},
		{name: "write_at", symbol: "renvoLinuxAmd64SysWriteAt"},
		{name: "chmod", symbol: "renvoLinuxAmd64SysFchmod"},
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

func checkedInLinuxAmd64EntryTemplateValid(template runtimeEntryTemplate) bool {
	if template.algorithm != "" || len(template.code) == 0 || template.maxParameters != 2 ||
		template.requiresState || len(template.bss) != 3 || len(template.relocations) == 0 {
		return false
	}
	expectedBSS := []string{"args", "environment", "environmentLength"}
	for i := 0; i < len(template.bss); i++ {
		bss := template.bss[i]
		if bss.name != expectedBSS[i] || bss.size <= 0 || bss.alignment <= 0 ||
			bss.alignment&(bss.alignment-1) != 0 {
			return false
		}
	}
	for i := 0; i < len(template.relocations); i++ {
		relocation := template.relocations[i]
		if relocation.kind != "bss" || relocation.offset < 0 ||
			relocation.offset+4 > len(template.code) {
			return false
		}
		found := false
		for j := 0; j < len(template.bss); j++ {
			if relocation.target == template.bss[j].name {
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

func appendCheckedInLinuxAmd64Runtime(
	source []byte, operations []checkedInLinuxAmd64RuntimeOperation,
	unit string, prepareBuffer string, moveOffset string,
) []byte {
	for i := 0; i < len(operations); i++ {
		source = append(source, "\nconst "...)
		source = append(source, operations[i].symbol...)
		source = append(source, " = "...)
		source = appendDecimalFrame(source, operations[i].value)
	}
	source = append(source, "\n\nfunc renvoAmd64AsmPrepareReadWriteBuf(a *renvoAsm) {\n\trenvoNonNil(a)\n"...)
	source = append(source, "\trenvoAsmCopyPrimaryToCallWord1(a)\n\trtg"...)
	source = append(source, exportedName(unit)...)
	source = append(source, exportedName(prepareBuffer)...)
	source = append(source, "(a)\n}\n\nfunc renvoAmd64AsmMoveOffsetArg(a *renvoAsm) {\n\trenvoNonNil(a)\n\trtg"...)
	source = append(source, exportedName(unit)...)
	source = append(source, exportedName(moveOffset)...)
	source = append(source, "(a)\n}\n\nfunc compileLinuxAmd64(input []int, output int) int {\n\treturn compileLinuxAmd64Arena(input, output, 0)\n}\n\nfunc compileLinuxAmd64Arena(input []int, output int, arenaSize int) int {\n\trenvoSetTarget(renvoTargetLinuxAmd64)\n\treturn renvoCompileAmd64(input, output, arenaSize)\n}\n"...)
	return source
}

func appendCheckedInLinuxAmd64Entry(source []byte, template runtimeEntryTemplate) []byte {
	bssSymbols := []string{
		"renvoLinuxAmd64ArgsBSS",
		"renvoLinuxAmd64EnvironmentBSS",
		"renvoLinuxAmd64EnvironmentLengthBSS",
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
	source = append(source, "\n\nfunc renvoAsmBuildArgvEnvSlicesAmd64(a *renvoAsm"...)
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
		if relocation.kind != "bss" {
			continue
		}
		source = append(source, "\trenvoAsmAddAbsReloc(a, base+"...)
		source = appendDecimalFrame(source, relocation.offset)
		source = append(source, ", "...)
		source = append(source, relocation.target...)
		source = append(source, "Off"...)
		if relocation.addend != 0 {
			source = append(source, '+')
			source = appendDecimalFrame(source, relocation.addend)
		}
		source = append(source, ", renvoAbsBssReloc)\n"...)
	}
	source = append(source, "}\n"...)
	return source
}

const checkedInLinuxAmd64ImageSource = `

func renvoAsmImageAmd64(a *renvoAsm) []byte {
	renvoNonNil(a)
	renvoAsmPatch(a)
	loadFileSize := a.dataOffset + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	syscallTableSize := len(a.openbsdSyscalls) / 2 * 8
	if a.c.stripSymbols {
		oldCodeLen := len(a.code)
		var out []byte
		out = a.code
		if renvoFixedTarget == 0 && loadFileSize > cap(out) {
			out = renvoAppendUntil(out, loadFileSize)
		} else {
			renvoTruncBytes(&out, loadFileSize)
		}
		if renvoFixedTarget == 0 {
			copy(out[a.codeOffset:a.codeOffset+oldCodeLen], out[:oldCodeLen])
		} else {
			for i := oldCodeLen; i > 0; i-- {
				out[a.codeOffset+i-1] = out[i-1]
			}
		}
		var header []byte
		header = renvoAppendElfHeaderAmd64(header, a, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0, loadFileSize, syscallTableSize)
		if renvoFixedTarget == 0 {
			copy(out, header)
		} else {
			for i := 0; i < len(header); i++ {
				out[i] = header[i]
			}
		}
		pos := a.dataOffset
		if a.c.renvoTargetOS == renvoOSOpenBSD {
			out = renvoAppendOpenBSDSyscallTable(out, a)
		}
		if renvoFixedTarget == 0 {
			copy(out[pos:], a.data)
		} else {
			for i := 0; i < len(a.data); i++ {
				out[pos+i] = a.data[i]
			}
		}
		if renvoFixedTarget == 0 {
			return renvoAppendReplLinkTable(out, a)
		}
		return out
	}
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	finalSize := sec.shoff + 448
	syscallTableOff := finalSize
	finalSize += syscallTableSize
	out := make([]byte, finalSize)
	renvoTruncBytes(&out, 0)
	out = renvoAppendElfHeaderAmd64(out, a, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff, syscallTableOff, syscallTableSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = renvoAppendUntil(out, a.dataOffset)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = renvoAppendUntil(out, sec.symtabOff)
	for i := 0; i < len(sec.symtab); i++ {
		out = append(out, sec.symtab[i])
	}
	out = renvoAppendUntil(out, sec.strtabOff)
	for i := 0; i < len(sec.strtab); i++ {
		out = append(out, sec.strtab[i])
	}
	out = renvoAppendUntil(out, sec.shstrOff)
	for i := 0; i < len(sec.shstrtab); i++ {
		out = append(out, sec.shstrtab[i])
	}
	out = renvoAppendUntil(out, sec.shoff)
	out = renvoAppendElfSectionHeaders(out, &sec, a, 0)
	if a.c.renvoTargetOS == renvoOSOpenBSD {
		out = renvoAppendOpenBSDSyscallTable(out, a)
	}
	if renvoFixedTarget == 0 {
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}

func renvoAppendElfHeaderAmd64(out []byte, a *renvoAsm, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int, syscallTableOff int, syscallTableSize int) []byte {
	if a.c.renvoTargetOS == renvoOSOpenBSD {
		return renvoAppendOpenBSDElfHeaderAmd64(out, entryOff, a.dataOffset, fileSize, bssOffset, bssSize, shoff, syscallTableOff, syscallTableSize)
	}
	if a.c.renvoTargetOS == renvoOSNetBSD {
		return renvoAppendNetBSDElfHeaderAmd64(out, entryOff, fileSize, bssOffset, bssSize, shoff)
	}
	start := len(out)
	base := 0
	header := "\x7f\x45\x4c\x46\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x38\x00\x02\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00"
	for i := 0; i < len(header); i++ {
		out = append(out, header[i])
	}
	out[start+18] = byte(renvoLinuxAmd64ELFMachine)
	out[start+19] = byte(renvoLinuxAmd64ELFMachine >> 8)
	if a.c.renvoTargetOS == renvoOSFreeBSD {
		out[start+7] = 9
	}
	// All Linux/amd64 references are RIP-relative. ET_DYN lets the kernel
	// randomize this self-contained image without a dynamic loader.
	out[start+16] = 3
	renvoPut32At(out, start+24, base+entryOff)
	renvoPut32At(out, start+40, shoff)
	renvoPut32At(out, start+80, base)
	renvoPut32At(out, start+88, base)
	if shoff != 0 {
		out[start+58] = 64
		out[start+60] = 7
		out[start+62] = 6
	}
	renvoPut32At(out, start+96, fileSize)
	renvoPut32At(out, start+104, fileSize)
	out = renvoAppendElf64LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
	return out
}

func renvoAppendElf64Program(out []byte, kind int, flags int, offset int, address int, fileSize int, memorySize int, alignment int) []byte {
	out = renvoAppend32(out, kind)
	out = renvoAppend32(out, flags)
	out = renvoAppend64(out, offset)
	out = renvoAppend64(out, address)
	out = renvoAppend64(out, address)
	out = renvoAppend64(out, fileSize)
	out = renvoAppend64(out, memorySize)
	return renvoAppend64(out, alignment)
}
`

// GenerateCheckedInArchitectureAlgorithms emits the definition-owned
// algorithms reachable from the stable names used by the checked-in compiler.
// Fixed Renvo source manifests include this projection and therefore never
// parse unrelated semantic-contract helpers.
func GenerateCheckedInArchitectureAlgorithms(resolved ResolveResult, archName string, packageName string) GenerateResult {
	ensureDirectEmitterV1()
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := generateHeaderPackage(manifest, "algorithms/"+archName, packageName)
	projection := checkedInProjectionDocument(resolved.Document, arch)
	roots := architectureExportGoRoots(arch)
	roots, sequences := resolveArchitectureSequenceProjection(resolved.Document, arch,
		roots, architectureSequenceRoots(arch, true, false))
	source = appendReachableEmbeddedGo(source, projection, roots, true, architectureExports(arch))
	source = appendArchitectureSequences(source, projection, arch, true, true, false, sequences)
	if declarativeArchitectureRelocations(arch) != "" {
		source = appendCheckedInArchitectureHooks(source, projection, arch)
	}
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

// GenerateCheckedInArchitectureContract emits the complete typed semantic
// contract as ordinary production Go. It is compiled by normal Go-module and
// release builds, but omitted from fixed Renvo source manifests so source-level
// pruning remains effective.
func GenerateCheckedInArchitectureContract(resolved ResolveResult, archName string, packageName string) GenerateResult {
	ensureDirectEmitterV1()
	if !resolved.Ok {
		diagnostics := make([]Diagnostic, len(resolved.Diagnostics))
		copy(diagnostics, resolved.Diagnostics)
		return GenerateResult{Diagnostics: diagnostics}
	}
	arch, ok := resolved.Document.Declaration(DeclArch, archName)
	if !ok {
		return GenerateResult{Diagnostics: []Diagnostic{{
			Filename: resolved.Document.Filename,
			Span:     sourceSpan(resolved.Document.Source, 0, 0),
			Code:     "RTG-GENERATE-004",
			Message:  "definition does not declare architecture " + archName,
		}}}
	}
	if len(architectureBindings(arch)) == 0 {
		return GenerateResult{Diagnostics: []Diagnostic{documentDiagnostic(
			resolved.Document, arch.Span, "RTG-GENERATE-005",
			"architecture "+archName+" has no direct emitter bindings")}}
	}
	manifest := []string{resolved.Document.Unit + " " + HashText(resolved.Document.Hash)}
	source := []byte("//go:build !renvo\n\n")
	source = append(source, generateHeaderPackage(manifest, "contract/"+archName, packageName)...)
	projection := checkedInProjectionDocument(resolved.Document, arch)
	source = appendArchitectureFacts(source, projection, arch, true)
	roots := architectureDirectGoRoots(arch)
	roots, sequences := resolveArchitectureSequenceProjection(resolved.Document, arch,
		roots, architectureSequenceRoots(arch, false, true))
	exports := architectureExports(arch)
	excluded := make([]string, 0, len(exports))
	for i := 0; i < len(exports); i++ {
		excluded = append(excluded, exports[i].Local)
	}
	source = appendReachableEmbeddedGoExcluding(
		source, projection, roots, true, exports, excluded)
	source = appendArchitectureSequences(source, projection, arch, true, false, true, sequences)
	source = appendArchitectureBindings(source, projection, arch, true, true)
	source = appendDirectEmitterBindings(source, projection, arch, true, true)
	source = appendArchitectureHooks(source, projection, arch, true)
	return GenerateResult{Source: source, Manifest: manifest, Ok: true}
}

func checkedInProjectionDocument(document Document, arch Declaration) Document {
	count := 0
	for i := 0; i < len(document.Declarations); i++ {
		if document.Declarations[i].Kind == DeclArch {
			count++
		}
	}
	// Native target entrypoints deliberately share the stable "native" unit
	// name. Architecture projections still use the architecture name so the
	// checked-in compiler-facing symbols do not depend on which target root
	// supplied their shared ISA.
	if document.Unit == "native" || count > 1 {
		document.Unit = arch.Name
	}
	return document
}
