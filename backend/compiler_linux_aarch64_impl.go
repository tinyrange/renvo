package main

const renvoLinuxAarch64CodeOffset = 0xb0
const renvoLinuxAarch64LoadAddress = 0x400000

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

func renvoCompileAarch64(input []int, output int, arenaSize int) int {
	src := make([]byte, 0, 589824)
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(currentTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramAarch64(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramAarch64(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result renvoCompileResult
		return result
	}
	var g renvoLinearGen
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	renvoAsmInit(a)
	a.codeOffset = renvoLinuxAarch64CodeOffset
	if targetIsWindows() {
		a.codeOffset = renvoWinSectionRVA
	}
	if targetIsDarwin() {
		a.codeOffset = renvoDarwinArm64CodeOffset
		g.darwinEntryOff = a.bssSize
		a.bssSize += 24
		renvoAarch64AsmMovRegAbs(a, 9, g.darwinEntryOff, renvoAbsBssReloc)
		renvoAarch64AsmStoreRegMem(a, 0, 9, 0, 8)
		renvoAarch64AsmStoreRegMem(a, 1, 9, 8, 8)
		renvoAarch64AsmStoreRegMem(a, 2, 9, 16, 8)
	}
	if compilerFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(&g, len(meta.funcs))
	renvoLinearMarkFunc(&g, appIndex)
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) {
		var result renvoCompileResult
		return result
	}
	entryOK := false
	if targetIsWindows() {
		entryOK = renvoEmitProgramEntryArgsWindowsArm64(&g, appIndex)
	} else if targetIsDarwin() {
		entryOK = renvoEmitProgramEntryArgsDarwinArm64(&g, appIndex)
	} else {
		entryOK = renvoEmitProgramEntryArgsAarch64(&g, appIndex)
	}
	if !entryOK {
		var result renvoCompileResult
		return result
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		var result renvoCompileResult
		return result
	}
	if targetIsWindows() {
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
		renvoWinArm64CallImport(a, renvoWinImportExitProcess)
		renvoAsmRet(a)
	} else if targetIsDarwin() {
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
		renvoDarwinArm64CallImport(a, renvoDarwinImportExit)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 93)
		renvoAsmSyscall(a)
	}
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !renvoEmitScalarFunctionScratch(&g, i) {
			if targetIsDarwin() {
				renvoPrintErr("renvo: failed to emit function ")
				write(2, p.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
				renvoPrintErr("\n")
			}
			var result renvoCompileResult
			return result
		}
	}
	data := renvoAsmImageAarch64(a)
	if targetIsWindows() {
		data = renvoAsmImageWindowsArm64(a)
	} else if targetIsDarwin() {
		data = renvoAsmImageDarwinArm64(a)
	}
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
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
		return out
	}
	sec := renvoBuildElf64SymbolSections(a, renvoLinuxAarch64LoadAddress, a.codeOffset, loadFileSize)
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
	out = renvoAppendElf64SectionHeaders(out, &sec, a, renvoLinuxAarch64LoadAddress)
	return out
}

func renvoAsmPatchAarch64Abs(a *renvoAsm) {
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + r.off
		}
		insn := renvoGet32At(a.code, r.at)
		reg := insn & 31
		renvoAarch64AsmPatchMovRegImmAt(a, r.at, reg, renvoLinuxAarch64LoadAddress+target)
	}
}

func renvoAppendElfHeaderAarch64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := renvoLinuxAarch64LoadAddress

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
	out = renvoAppend16(out, 2)
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
