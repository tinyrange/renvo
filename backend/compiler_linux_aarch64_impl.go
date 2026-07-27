package main

const renvoLinuxAarch64SysReadSeq = 63
const renvoLinuxAarch64SysWriteSeq = 64
const renvoLinuxAarch64SysOpen = 56
const renvoLinuxAarch64SysClose = 57
const renvoLinuxAarch64SysFchmod = 52
const renvoLinuxAarch64SysReadAt = 67
const renvoLinuxAarch64SysWriteAt = 68

func renvoAarch64AsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRsi, renvoAarch64RegRax)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRcx)
}

func renvoAarch64AsmMoveOffsetArg(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR10, renvoAarch64RegRax)
}

func compileLinuxAarch64(input []int, output int) int {
	return compileLinuxAarch64Arena(input, output, 0)
}

func compileLinuxAarch64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxAarch64)
	return renvoCompileAarch64(input, output, arenaSize)
}

func renvoEmitProgramEntryArgsAarch64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	renvoAsmBuildArgvEnvSlicesAarch64(&g.asm, argsOff, envDataOff, envLenOff)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !renvoTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !renvoTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func renvoAsmBuildArgvEnvSlicesAarch64(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := renvoAsmNewLabel(a)
	strlenLabel := renvoAsmNewLabel(a)
	afterLenLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	envLoopLabel := renvoAsmNewLabel(a)
	envStrlenLabel := renvoAsmNewLabel(a)
	envAfterLenLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)

	renvoAarch64AsmLoadRegMem(a, 13, 31, 0, 8)
	renvoAarch64AsmAddRegImm(a, 10, 31, 8)
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegR10, bssOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 11, 0)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAarch64AsmCmpRegReg(a, 11, 13)
	renvoAarch64AsmBCondLabel(a, doneLabel, 0)
	renvoAarch64AsmAddRegRegShift(a, 12, 10, 11, 3)
	renvoAarch64AsmLoadRegMem(a, 12, 12, 0, 8)
	renvoAarch64AsmStoreRegMem(a, 12, renvoAarch64RegR10, 0, 8)
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegRax, 0)
	renvoAsmMarkLabel(a, strlenLabel)
	renvoAarch64AsmAddRegReg(a, 9, 12, renvoAarch64RegRax)
	renvoAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 9, 0)
	renvoAarch64AsmBCondLabel(a, afterLenLabel, 0)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRax, renvoAarch64RegRax, 1)
	renvoAarch64AsmJmpLabel(a, strlenLabel)
	renvoAsmMarkLabel(a, afterLenLabel)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegR10, 8, 8)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegR10, renvoAarch64RegR10, 16)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmJmpLabel(a, loopLabel)
	renvoAsmMarkLabel(a, doneLabel)

	renvoAarch64AsmAddRegRegShift(a, 10, 10, 13, 3)
	renvoAarch64AsmAddRegImm(a, 10, 10, 8)
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegR10, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 11, 0)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAarch64AsmLoadRegMem(a, 12, 10, 0, 8)
	renvoAarch64AsmCmpRegImm(a, 12, 0)
	renvoAarch64AsmBCondLabel(a, envDoneLabel, 0)
	renvoAarch64AsmStoreRegMem(a, 12, renvoAarch64RegR10, 0, 8)
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegRax, 0)
	renvoAsmMarkLabel(a, envStrlenLabel)
	renvoAarch64AsmAddRegReg(a, 9, 12, renvoAarch64RegRax)
	renvoAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 9, 0)
	renvoAarch64AsmBCondLabel(a, envAfterLenLabel, 0)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRax, renvoAarch64RegRax, 1)
	renvoAarch64AsmJmpLabel(a, envStrlenLabel)
	renvoAsmMarkLabel(a, envAfterLenLabel)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegR10, 8, 8)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegR10, renvoAarch64RegR10, 16)
	renvoAarch64AsmAddRegImm(a, 10, 10, 8)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmJmpLabel(a, envLoopLabel)
	renvoAsmMarkLabel(a, envDoneLabel)
	renvoAarch64AsmMovRegAbs(a, 12, envLenOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, 11, 12, 0, 8)

	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRdi, bssOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRsi, 13)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, 13)
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRcx, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR8, 11)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR9, 11)
}

func renvoAsmImageAarch64(a *renvoAsm) []byte {
	renvoAsmPatch(a)
	renvoAsmPatchAarch64Abs(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if renvoCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		if renvoFixedTarget == 0 {
			return renvoAppendReplLinkTable(out, a)
		}
		return out
	}
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	out := make([]byte, 0, 1048576)
	out = renvoAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
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
	if renvoFixedTarget == 0 {
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}

func renvoAsmPatchAarch64Abs(a *renvoAsm) {
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		insn := renvoGet32At(a.code, at)
		reg := insn & 31
		pcPage := (a.codeOffset + at) & -4096
		targetPage := target & -4096
		pageDelta := (targetPage - pcPage) >> 12
		immlo := pageDelta & 3
		immhi := pageDelta >> 2 & 524287
		renvoPut32At(a.code, at, 0x90000000|(immlo<<29)|(immhi<<5)|reg)
		renvoPut32At(a.code, at+4, 0x91000000|((target&4095)<<10)|(reg<<5)|reg)
		renvoPut32At(a.code, at+8, 0xd503201f)
		renvoPut32At(a.code, at+12, 0xd503201f)
	}
}

func renvoAppendElfHeaderAarch64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := 0

	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	out = append(out, 2)
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 0)
	for i := 0; i < 8; i++ {
		out = append(out, 0)
	}
	out = renvoAppend16(out, 3)
	out = renvoAppend16(out, 183)
	out = renvoAppend32(out, 1)
	out = renvoAppend64U32(out, base+entryOff)
	out = renvoAppend64U32(out, 64)
	out = renvoAppend64U32(out, shoff)
	out = renvoAppend32(out, 0)
	out = renvoAppend16(out, 64)
	out = renvoAppend16(out, 56)
	out = renvoAppend16(out, 2)
	if shoff == 0 {
		out = renvoAppend16(out, 0)
		out = renvoAppend16(out, 0)
		out = renvoAppend16(out, 0)
	} else {
		out = renvoAppend16(out, 64)
		out = renvoAppend16(out, 7)
		out = renvoAppend16(out, 6)
	}

	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 5)
	out = renvoAppend64U32(out, 0)
	out = renvoAppend64U32(out, base)
	out = renvoAppend64U32(out, base)
	out = renvoAppend64U32(out, fileSize)
	out = renvoAppend64U32(out, fileSize)
	out = renvoAppend64U32(out, 0x1000)
	out = renvoAppendElf64LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
	return out
}
