//go:build !renvo

package rtg

func generateCheckedInLinuxArmProjection(
	resolved ResolveResult, target ResolvedTarget, packageName string,
) GenerateResult {
	document := resolved.Document
	bits, bitsOK := integerField(document, target.Executable, "address_bits")
	machine, machineOK := integerField(document, target.Executable, "machine")
	codeOffset, offsetOK := integerField(document, target.Executable, "code_offset")
	flags, flagsOK := integerField(document, target.Executable, "flags")
	patch, patchOK := fieldValue(document, target.Executable, "patch_absolute")
	if declarativeFormatImage(target.Executable) != "elf_executable" ||
		!bitsOK || bits != 32 || !machineOK || machine <= 0 ||
		!offsetOK || codeOffset <= 0 || !flagsOK ||
		!patchOK || valueName(patch) != "arm32_mov_address" {
		return checkedInTargetProjectionFailure(document, target.Executable,
			"linux/arm requires the ARM EABI ELF layout")
	}
	template, ok := decodeRuntimeEntryTemplate(target.Runtime)
	if !ok || !checkedInLiteralTemplateValid(template,
		[]string{"args", "environment", "environmentLength"}, 0) {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/arm requires its complete three-buffer literal entry")
	}
	operations, missing := checkedInLinuxArmRuntimeOperations(target.Runtime)
	if missing != "" {
		return checkedInTargetProjectionFailure(document, target.Runtime,
			"linux/arm runtime operation "+missing+" requires a syscall number")
	}
	manifest := []string{document.Unit + " " + HashText(document.Hash)}
	source := generateHeaderPackage(manifest,
		"target-projection/"+target.Descriptor.Name, packageName)
	source = appendCheckedInLinuxArmRuntime(source, operations, template,
		codeOffset, machine, flags)
	source = append(source, checkedInLinuxArmImageSource...)
	return GenerateResult{Source: source, Descriptor: target.Descriptor,
		Manifest: manifest, Ok: true}
}

func checkedInLinuxArmRuntimeOperations(
	runtime Declaration,
) ([]checkedInLinuxAmd64RuntimeOperation, string) {
	operations := []checkedInLinuxAmd64RuntimeOperation{
		{name: "read", symbol: "renvoLinuxArmSysReadSeq"},
		{name: "write", symbol: "renvoLinuxArmSysWriteSeq"},
		{name: "open", symbol: "renvoLinuxArmSysOpen"},
		{name: "close", symbol: "renvoLinuxArmSysClose"},
		{name: "read_at", symbol: "renvoLinuxArmSysReadAt"},
		{name: "write_at", symbol: "renvoLinuxArmSysWriteAt"},
		{name: "chmod", symbol: "renvoLinuxArmSysFchmod"},
		{name: "exit", symbol: "renvoLinuxArmSysExit"},
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

func appendCheckedInLinuxArmRuntime(source []byte,
	operations []checkedInLinuxAmd64RuntimeOperation, template runtimeEntryTemplate,
	codeOffset int, machine int, flags int,
) []byte {
	source = append(source, "\nconst renvoLinuxArmCodeOffset = "...)
	source = appendDecimalFrame(source, codeOffset)
	source = append(source, "\nconst renvoLinuxArmELFMachine = "...)
	source = appendDecimalFrame(source, machine)
	source = append(source, "\nconst renvoLinuxArmELFFlags = "...)
	source = appendDecimalFrame(source, flags)
	for i := 0; i < len(operations); i++ {
		source = append(source, "\nconst "...)
		source = append(source, operations[i].symbol...)
		source = append(source, " = "...)
		source = appendDecimalFrame(source, operations[i].value)
	}
	source = append(source, checkedInLinuxArmCompileSource...)
	bssSymbols := []string{"renvoLinuxArmArgsBSS", "renvoLinuxArmEnvironmentBSS",
		"renvoLinuxArmEnvironmentLengthBSS"}
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
	source = append(source, "\n\nfunc renvoAsmBuildArgvEnvSlicesArm(a *renvoAsm, argsOff int, environmentOff int, environmentLengthOff int) {\n\tbase := len(a.code)\n\trenvoAsmEmitText(a, "...)
	source = append(source, template.code...)
	source = append(source, ")\n"...)
	bssParameters := []string{"argsOff", "environmentOff", "environmentLengthOff"}
	for i := 0; i < len(template.relocations); i++ {
		relocation := template.relocations[i]
		target := ""
		for j := 0; j < len(template.bss); j++ {
			if template.bss[j].name == relocation.target {
				target = bssParameters[j]
				break
			}
		}
		source = append(source, "\trenvoAsmAddAbsReloc(a, base+"...)
		source = appendDecimalFrame(source, relocation.offset)
		source = append(source, ", "...)
		source = append(source, target...)
		if relocation.addend != 0 {
			source = append(source, '+')
			source = appendDecimalFrame(source, relocation.addend)
		}
		source = append(source, ", renvoAbsBssReloc)\n"...)
	}
	source = append(source, "}\n"...)
	source = append(source, checkedInLinuxArmEntrySource...)
	return source
}

const checkedInLinuxArmCompileSource = `

func renvoArmAsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRsi, renvoArmRegRax)
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegRcx)
}

func renvoArmAsmMoveOffsetArg(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegR10, renvoArmRegRax)
}

func compileLinuxArm(input []int, output int) int {
	return compileLinuxArmArena(input, output, 0)
}

func compileLinuxArmArena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxArm)
	src := renvoMakeByteScratch(786432)
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if !prog.ok { return 1 }
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok { return 1 }
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	result := renvoTryCompileScalarProgramArmScratch(&prog, &meta)
	if !result.ok {
		renvoPrintErr("renvo: compilation failed\n")
		return 1
	}
	data := result.data
	if renvoFixedTarget == 0 { data = renvoCompileOutputData(data, renvoTarget) }
	write(output, data, -1)
	return 0
}
`

const checkedInLinuxArmEntrySource = `

func renvoEmitProgramEntryArgsArm(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) { return false }
	argsOff := renvoAlignValue(g.asm.bssSize, renvoLinuxArmArgsBSSAlignment)
	g.asm.bssSize = argsOff + renvoLinuxArmArgsBSSSize
	environmentOff := renvoAlignValue(g.asm.bssSize, renvoLinuxArmEnvironmentBSSAlignment)
	g.asm.bssSize = environmentOff + renvoLinuxArmEnvironmentBSSSize
	environmentLengthOff := renvoAlignValue(g.asm.bssSize, renvoLinuxArmEnvironmentLengthBSSAlignment)
	g.asm.bssSize = environmentLengthOff + renvoLinuxArmEnvironmentLengthBSSSize
	renvoAsmBuildArgvEnvSlicesArm(&g.asm, argsOff, environmentOff, environmentLengthOff)
	if app.paramCount == 0 { return true }
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) { return false }
	return app.paramCount == 1 || renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ)
}
`

const checkedInLinuxArmImageSource = `

func renvoAsmImageArm(a *renvoAsm) []byte {
	renvoAsmPatchArm(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if a.c.stripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		out = append(out, a.code...)
		out = append(out, a.data...)
		if renvoFixedTarget == 0 { return renvoAppendReplLinkTable(out, a) }
		return out
	}
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	out := make([]byte, 0, sec.shoff+280)
	out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
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

func renvoAsmPatchArm(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc { target = renvoAsmBssOffset(a) + off }
		insn := renvoGet32At(a.code, at)
		reg := (insn >> 12) & 15
		delta := target - (a.codeOffset + at + 16)
		renvoArmAsmPatchMovRegImmAt(a, at, reg, delta)
	}
}

func renvoAppendElfHeaderArm(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	base := 0
	out = renvoAppendUntil(out, start+116)
	out[start] = 0x7f
	out[start+1] = 'E'
	out[start+2] = 'L'
	out[start+3] = 'F'
	out[start+4] = 1
	out[start+5] = 1
	out[start+6] = 1
	out[start+16] = 3
	out[start+18] = byte(renvoLinuxArmELFMachine)
	out[start+19] = byte(renvoLinuxArmELFMachine >> 8)
	renvoPut32At(out, start+20, 1)
	renvoPut32At(out, start+24, base+entryOff)
	renvoPut32At(out, start+28, 52)
	renvoPut32At(out, start+32, shoff)
	renvoPut32At(out, start+36, renvoLinuxArmELFFlags)
	out[start+40] = 52
	out[start+42] = 32
	out[start+44] = 2
	if shoff != 0 { out[start+46] = 40; out[start+48] = 7; out[start+50] = 6 }
	renvoPut32At(out, start+52, 1)
	renvoPut32At(out, start+60, base)
	renvoPut32At(out, start+64, base)
	renvoPut32At(out, start+68, fileSize)
	renvoPut32At(out, start+72, fileSize)
	renvoPut32At(out, start+76, 5)
	renvoPut32At(out, start+80, 4096)
	renvoPut32At(out, start+84, 1)
	renvoPut32At(out, start+88, bssOffset)
	renvoPut32At(out, start+92, base+bssOffset)
	renvoPut32At(out, start+96, base+bssOffset)
	renvoPut32At(out, start+104, bssSize)
	renvoPut32At(out, start+108, 6)
	renvoPut32At(out, start+112, 4096)
	return out
}
`
