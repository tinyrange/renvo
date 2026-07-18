package main

const rtgLinuxAarch64CodeOffset = 0x78
const rtgLinuxAarch64LoadAddress = 0x400000

const rtgLinuxAarch64SysReadSeq = 63
const rtgLinuxAarch64SysWriteSeq = 64
const rtgLinuxAarch64SysOpen = 56
const rtgLinuxAarch64SysClose = 57
const rtgLinuxAarch64SysFchmod = 52
const rtgLinuxAarch64SysReadAt = 67
const rtgLinuxAarch64SysWriteAt = 68

func rtgAarch64AsmPrepareReadWriteBuf(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRsi, rtgAarch64RegRax)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRcx)
}

func rtgAarch64AsmMoveOffsetArg(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR10, rtgAarch64RegRax)
}

func compileLinuxAarch64(input []int, output int) int {
	return compileLinuxAarch64Arena(input, output, 0)
}

func compileLinuxAarch64Arena(input []int, output int, arenaSize int) int {
	rtgSetTarget(rtgTargetLinuxAarch64)
	return rtgCompileAarch64(input, output, arenaSize)
}

func rtgCompileAarch64(input []int, output int, arenaSize int) int {
	src := make([]byte, 0, 589824)
	for i := 0; i < len(input); i++ {
		src = rtgReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta rtgMeta
	rtgBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = rtgResolveArenaSize(rtgCurrentTarget, arenaSize)
	var result rtgCompileResult
	result = rtgTryCompileScalarProgramAarch64(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func rtgTryCompileScalarProgramAarch64(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result rtgCompileResult
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	rtgAsmInit(a)
	a.codeOffset = rtgLinuxAarch64CodeOffset
	if rtgTargetIsWindows() {
		a.codeOffset = rtgWinSectionRVA
	}
	if rtgTargetIsDarwin() {
		a.codeOffset = rtgDarwinArm64CodeOffset
		g.darwinEntryOff = a.bssSize
		a.bssSize += 24
		rtgAarch64AsmMovRegAbs(a, 9, g.darwinEntryOff, rtgAbsBssReloc)
		rtgAarch64AsmStoreRegMem(a, 0, 9, 0, 8)
		rtgAarch64AsmStoreRegMem(a, 1, 9, 8, 8)
		rtgAarch64AsmStoreRegMem(a, 2, 9, 16, 8)
	}
	if rtgCompilerFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := rtgAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	rtgInitFuncQueue(&g, len(meta.funcs))
	rtgLinearMarkFunc(&g, appIndex)
	rtgEmitPersistentArenaReady(&g)
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	entryOK := false
	if rtgTargetIsWindows() {
		entryOK = rtgEmitProgramEntryArgsWindowsArm64(&g, appIndex)
	} else if rtgTargetIsDarwin() {
		entryOK = rtgEmitProgramEntryArgsDarwinArm64(&g, appIndex)
	} else {
		entryOK = rtgEmitProgramEntryArgsAarch64(&g, appIndex)
	}
	if !entryOK {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	if !rtgEmitProgramPanicCheck(&g) {
		var result rtgCompileResult
		return result
	}
	if rtgTargetIsWindows() {
		rtgAarch64AsmMovRegReg(a, 0, rtgAarch64RegRax)
		rtgWinArm64CallImport(a, rtgWinImportExitProcess)
		rtgAsmRet(a)
	} else if rtgTargetIsDarwin() {
		rtgAarch64AsmMovRegReg(a, 0, rtgAarch64RegRax)
		rtgDarwinArm64CallImport(a, rtgDarwinImportExit)
		rtgAsmRet(a)
	} else {
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmPrimaryImm(a, 93)
		rtgAsmSyscall(a)
	}
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !rtgEmitScalarFunctionScratch(&g, i) {
			if rtgTargetIsDarwin() {
				rtgPrintErr("rtg: failed to emit function ")
				write(2, p.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
				rtgPrintErr("\n")
			}
			var result rtgCompileResult
			return result
		}
	}
	data := rtgAsmImageAarch64(a)
	if rtgTargetIsWindows() {
		data = rtgAsmImageWindowsArm64(a)
	} else if rtgTargetIsDarwin() {
		data = rtgAsmImageDarwinArm64(a)
	}
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgsAarch64(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	rtgAsmBuildArgvEnvSlicesAarch64(&g.asm, argsOff, envDataOff, envLenOff)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !rtgTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !rtgTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func rtgAsmBuildArgvEnvSlicesAarch64(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)

	rtgAarch64AsmLoadRegMem(a, 13, 31, 0, 8)
	rtgAarch64AsmAddRegImm(a, 10, 31, 8)
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegR10, bssOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 11, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAarch64AsmCmpRegReg(a, 11, 13)
	rtgAarch64AsmBCondLabel(a, doneLabel, 0)
	rtgAarch64AsmAddRegRegShift(a, 12, 10, 11, 3)
	rtgAarch64AsmLoadRegMem(a, 12, 12, 0, 8)
	rtgAarch64AsmStoreRegMem(a, 12, rtgAarch64RegR10, 0, 8)
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegRax, 0)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAarch64AsmAddRegReg(a, 9, 12, rtgAarch64RegRax)
	rtgAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 9, 0)
	rtgAarch64AsmBCondLabel(a, afterLenLabel, 0)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRax, rtgAarch64RegRax, 1)
	rtgAarch64AsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegR10, 8, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegR10, rtgAarch64RegR10, 16)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgAarch64AsmAddRegRegShift(a, 10, 10, 13, 3)
	rtgAarch64AsmAddRegImm(a, 10, 10, 8)
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegR10, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 11, 0)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAarch64AsmLoadRegMem(a, 12, 10, 0, 8)
	rtgAarch64AsmCmpRegImm(a, 12, 0)
	rtgAarch64AsmBCondLabel(a, envDoneLabel, 0)
	rtgAarch64AsmStoreRegMem(a, 12, rtgAarch64RegR10, 0, 8)
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegRax, 0)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAarch64AsmAddRegReg(a, 9, 12, rtgAarch64RegRax)
	rtgAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 9, 0)
	rtgAarch64AsmBCondLabel(a, envAfterLenLabel, 0)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRax, rtgAarch64RegRax, 1)
	rtgAarch64AsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegR10, 8, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegR10, rtgAarch64RegR10, 16)
	rtgAarch64AsmAddRegImm(a, 10, 10, 8)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAarch64AsmMovRegAbs(a, 12, envLenOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, 11, 12, 0, 8)

	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRdi, bssOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRsi, 13)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, 13)
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRcx, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR8, 11)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR9, 11)
}

func rtgAsmImageAarch64(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	rtgAsmPatchAarch64Abs(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := loadFileSize + a.bssSize
	if rtgCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = rtgAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, memSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		return out
	}
	sec := rtgBuildElf64SymbolSections(a, rtgLinuxAarch64LoadAddress, a.codeOffset, loadFileSize)
	out := make([]byte, 0, 1048576)
	out = rtgAppendElfHeaderAarch64(out, a.codeOffset, loadFileSize, memSize, sec.shoff)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = rtgAppendUntil(out, sec.symtabOff)
	for i := 0; i < len(sec.symtab); i++ {
		out = append(out, sec.symtab[i])
	}
	out = rtgAppendUntil(out, sec.strtabOff)
	for i := 0; i < len(sec.strtab); i++ {
		out = append(out, sec.strtab[i])
	}
	out = rtgAppendUntil(out, sec.shstrOff)
	for i := 0; i < len(sec.shstrtab); i++ {
		out = append(out, sec.shstrtab[i])
	}
	out = rtgAppendUntil(out, sec.shoff)
	out = rtgAppendElf64SectionHeaders(out, &sec, a, rtgLinuxAarch64LoadAddress)
	return out
}

func rtgAsmPatchAarch64Abs(a *rtgAsm) {
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		insn := rtgGet32At(a.code, r.at)
		reg := insn & 31
		rtgAarch64AsmPatchMovRegImmAt(a, r.at, reg, rtgLinuxAarch64LoadAddress+target)
	}
}

func rtgAppendElfHeaderAarch64(out []byte, entryOff int, fileSize int, memSize int, shoff int) []byte {
	base := rtgLinuxAarch64LoadAddress

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
	out = rtgAppend16(out, 2)
	out = rtgAppend16(out, 183)
	out = rtgAppend32(out, 1)
	out = rtgAppend64U32(out, base+entryOff)
	out = rtgAppend64U32(out, 64)
	out = rtgAppend64U32(out, shoff)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 56)
	out = rtgAppend16(out, 1)
	if shoff == 0 {
		out = rtgAppend16(out, 0)
		out = rtgAppend16(out, 0)
		out = rtgAppend16(out, 0)
	} else {
		out = rtgAppend16(out, 64)
		out = rtgAppend16(out, 7)
		out = rtgAppend16(out, 6)
	}

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 7)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, base)
	out = rtgAppend64U32(out, base)
	out = rtgAppend64U32(out, fileSize)
	out = rtgAppend64U32(out, memSize)
	out = rtgAppend64U32(out, 0x1000)
	return out
}
