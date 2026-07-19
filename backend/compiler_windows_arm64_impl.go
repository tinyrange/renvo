package main

const renvoWinArm64Machine = 0xaa64
const renvoWinArm64ImageBase = 0x140000000

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
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !renvoTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	argsTextOff := g.asm.bssSize
	g.asm.bssSize += 32768
	argsLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	envOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	renvoAsmBuildWindowsArgvEnvSlicesArm64(&g.asm, argsOff, argsTextOff, argsLenOff, envOff, envLenOff)
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	return renvoTypeIsStringSlice(g.meta, second.typ)
}

func renvoAsmBuildWindowsArgvEnvSlicesArm64(a *renvoAsm, argsOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	skipLabel := renvoAsmNewLabel(a)
	startLabel := renvoAsmNewLabel(a)
	copyLabel := renvoAsmNewLabel(a)
	quoteLabel := renvoAsmNewLabel(a)
	setQuoteLabel := renvoAsmNewLabel(a)
	copyCharLabel := renvoAsmNewLabel(a)
	argDoneLabel := renvoAsmNewLabel(a)
	argsDoneLabel := renvoAsmNewLabel(a)

	renvoWinArm64CallImport(a, renvoWinImportGetCommandLineA)
	renvoAarch64AsmMovRegReg(a, 9, 0) // source
	renvoAarch64AsmMovRegAbs(a, 10, argsOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegAbs(a, 11, argsTextOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 12, 0) // argument count
	renvoAsmMarkLabel(a, skipLabel)
	renvoAarch64AsmLoadRegMem(a, 13, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 13, 0)
	renvoAarch64AsmBCondLabel(a, argsDoneLabel, 0)
	renvoAarch64AsmCmpRegImm(a, 13, ' ')
	renvoAarch64AsmBCondLabel(a, startLabel, 1)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmJmpLabel(a, skipLabel)
	renvoAsmMarkLabel(a, startLabel)
	renvoAarch64AsmCmpRegImm(a, 13, 9)
	renvoAarch64AsmBCondLabel(a, copyLabel, 1)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmJmpLabel(a, skipLabel)

	renvoAsmMarkLabel(a, copyLabel)
	renvoAarch64AsmMovRegReg(a, 14, 11) // copied argument start
	renvoAarch64AsmMovRegImm(a, 15, 0)  // copied length
	renvoAarch64AsmMovRegImm(a, 17, 0)  // quote state
	copyLoopLabel := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, copyLoopLabel)
	renvoAarch64AsmLoadRegMem(a, 13, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 13, 0)
	renvoAarch64AsmBCondLabel(a, argDoneLabel, 0)
	renvoAarch64AsmCmpRegImm(a, 13, '"')
	renvoAarch64AsmBCondLabel(a, quoteLabel, 0)
	renvoAarch64AsmCmpRegImm(a, 17, 0)
	renvoAarch64AsmBCondLabel(a, copyCharLabel, 1)
	renvoAarch64AsmCmpRegImm(a, 13, ' ')
	renvoAarch64AsmBCondLabel(a, argDoneLabel, 0)
	renvoAarch64AsmCmpRegImm(a, 13, 9)
	renvoAarch64AsmBCondLabel(a, argDoneLabel, 0)
	renvoAsmMarkLabel(a, copyCharLabel)
	renvoAarch64AsmStoreRegMem(a, 13, 11, 0, 1)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmAddRegImm(a, 15, 15, 1)
	renvoAarch64AsmJmpLabel(a, copyLoopLabel)
	renvoAsmMarkLabel(a, quoteLabel)
	renvoAarch64AsmCmpRegImm(a, 17, 0)
	renvoAarch64AsmBCondLabel(a, setQuoteLabel, 0)
	renvoAarch64AsmMovRegImm(a, 17, 0)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmJmpLabel(a, copyLoopLabel)
	renvoAsmMarkLabel(a, setQuoteLabel)
	renvoAarch64AsmMovRegImm(a, 17, 1)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmJmpLabel(a, copyLoopLabel)

	renvoAsmMarkLabel(a, argDoneLabel)
	renvoAarch64AsmStoreRegMem(a, 31, 11, 0, 1)
	renvoAarch64AsmStoreRegMem(a, 14, 10, 0, 8)
	renvoAarch64AsmStoreRegMem(a, 15, 10, 8, 8)
	renvoAarch64AsmAddRegImm(a, 10, 10, 16)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmAddRegImm(a, 12, 12, 1)
	renvoAarch64AsmCmpRegImm(a, 13, 0)
	renvoAarch64AsmBCondLabel(a, argsDoneLabel, 0)
	renvoAarch64AsmAddRegImm(a, 9, 9, 1)
	renvoAarch64AsmJmpLabel(a, skipLabel)

	renvoAsmMarkLabel(a, argsDoneLabel)
	renvoAarch64AsmMovRegAbs(a, 9, argsLenOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, 12, 9, 0, 8)

	envLoopLabel := renvoAsmNewLabel(a)
	envLengthLabel := renvoAsmNewLabel(a)
	envStringDoneLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)
	renvoWinArm64CallImport(a, renvoWinImportGetEnvironmentStringsA)
	renvoAarch64AsmMovRegReg(a, 9, 0)
	renvoAarch64AsmMovRegAbs(a, 10, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 11, 0)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAarch64AsmLoadRegMem(a, 12, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 12, 0)
	renvoAarch64AsmBCondLabel(a, envDoneLabel, 0)
	renvoAarch64AsmStoreRegMem(a, 9, 10, 0, 8)
	renvoAarch64AsmMovRegImm(a, 13, 0)
	renvoAsmMarkLabel(a, envLengthLabel)
	renvoAarch64AsmAddRegReg(a, 14, 9, 13)
	renvoAarch64AsmLoadRegMem(a, 12, 14, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 12, 0)
	renvoAarch64AsmBCondLabel(a, envStringDoneLabel, 0)
	renvoAarch64AsmAddRegImm(a, 13, 13, 1)
	renvoAarch64AsmJmpLabel(a, envLengthLabel)
	renvoAsmMarkLabel(a, envStringDoneLabel)
	renvoAarch64AsmStoreRegMem(a, 13, 10, 8, 8)
	renvoAarch64AsmAddRegImm(a, 10, 10, 16)
	renvoAarch64AsmAddRegImm(a, 13, 13, 1)
	renvoAarch64AsmAddRegReg(a, 9, 9, 13)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmJmpLabel(a, envLoopLabel)
	renvoAsmMarkLabel(a, envDoneLabel)
	renvoAarch64AsmMovRegAbs(a, 9, envLenOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, 11, 9, 0, 8)

	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRdi, argsOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegAbs(a, 9, argsLenOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRsi, 9, 0, 8)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRsi)
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRcx, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR8, 11)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR9, 11)
}

func renvoWinArm64CallImport(a *renvoAsm, importID int) {
	renvoAarch64AsmMovRegAbs(a, 16, importID, renvoAbsWinImportReloc)
	renvoAarch64AsmLoadRegMem(a, 16, 16, 0, 8)
	renvoAarch64AsmEmit(a, 0xd63f0200) // blr x16
}

func renvoWinArm64CallStaticImport(a *renvoAsm, importID int, wordCount int) {
	registerWords := wordCount
	if registerWords > 8 {
		registerWords = 8
	}
	for i := 0; i < registerWords; i++ {
		renvoAarch64AsmPopReg(a, i)
	}
	stackWords := wordCount - registerWords

	// RENVO's expression stack uses a 16-byte slot for every word. Windows ARM64
	// stack arguments are packed into 8-byte slots, so construct a temporary
	// ABI call area while retaining the exact pending-argument stack pointer.
	renvoAarch64AsmEmit(a, 0x910003e9) // add x9, sp, #0
	savedSPOff := stackWords * 8
	allocation := renvoAlignValue(savedSPOff+8, 16)
	renvoAarch64AsmAddRegImm(a, 31, 31, -allocation)
	for i := 0; i < stackWords; i++ {
		renvoAarch64AsmLoadRegMem(a, 10, 9, i*16, 8)
		renvoAarch64AsmStoreRegMem(a, 10, 31, i*8, 8)
	}
	renvoAarch64AsmStoreRegMem(a, 9, 31, savedSPOff, 8)
	renvoWinArm64CallImport(a, importID)
	renvoAarch64AsmMovRegReg(a, 10, 0)
	renvoAarch64AsmLoadRegMem(a, 9, 31, savedSPOff, 8)
	// ADD (immediate), unlike ORR-based MOV, can address SP as both operands.
	renvoAarch64AsmEmit(a, 0x9100013f)
	renvoAarch64AsmAddRegImm(a, 31, 31, stackWords*16)
	renvoAarch64AsmMovRegReg(a, 0, 10)
}

func renvoWinArm64TestRegImm(a *renvoAsm, reg int, imm int) {
	renvoAarch64AsmMovRegImm(a, 15, imm)
	renvoAarch64AsmEmit(a, 0xea00001f|(15<<16)|(reg<<5))
}

func renvoWinArm64TranslateCreateFileFlags(a *renvoAsm) {
	notReadWriteLabel := renvoAsmNewLabel(a)
	accessDoneLabel := renvoAsmNewLabel(a)
	noCreateLabel := renvoAsmNewLabel(a)
	createDoneLabel := renvoAsmNewLabel(a)

	renvoAarch64AsmMovRegImm(a, 1, -2147483648)
	renvoWinArm64TestRegImm(a, 0, 2)
	renvoAarch64AsmBCondLabel(a, notReadWriteLabel, 0)
	renvoAarch64AsmMovRegImm(a, 1, -1073741824)
	renvoAarch64AsmJmpLabel(a, accessDoneLabel)
	renvoAsmMarkLabel(a, notReadWriteLabel)
	renvoWinArm64TestRegImm(a, 0, 1)
	renvoAarch64AsmBCondLabel(a, accessDoneLabel, 0)
	renvoAarch64AsmMovRegImm(a, 1, 0x40000000)
	renvoAsmMarkLabel(a, accessDoneLabel)

	renvoAarch64AsmMovRegImm(a, 4, 3)
	renvoWinArm64TestRegImm(a, 0, 64)
	renvoAarch64AsmBCondLabel(a, noCreateLabel, 0)
	renvoAarch64AsmMovRegImm(a, 4, 4)
	renvoWinArm64TestRegImm(a, 0, 512)
	renvoAarch64AsmBCondLabel(a, createDoneLabel, 0)
	renvoAarch64AsmMovRegImm(a, 4, 2)
	renvoAarch64AsmJmpLabel(a, createDoneLabel)
	renvoAsmMarkLabel(a, noCreateLabel)
	renvoWinArm64TestRegImm(a, 0, 512)
	renvoAarch64AsmBCondLabel(a, createDoneLabel, 0)
	renvoAarch64AsmMovRegImm(a, 4, 5)
	renvoAsmMarkLabel(a, createDoneLabel)
	renvoAarch64AsmMovRegImm(a, 2, 3)
	renvoAarch64AsmMovRegImm(a, 3, 0)
	renvoAarch64AsmMovRegImm(a, 5, 0x80)
	renvoAarch64AsmMovRegImm(a, 6, 0)
}

func renvoWinArm64EmitReadWriteHelper(g *renvoLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = renvoAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = renvoAsmNewLabel(a)
	}
	label := g.winReadLabel
	importID := renvoWinImportReadFile
	if isWrite {
		label = g.winWriteLabel
		importID = renvoWinImportWriteFile
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	resultOff := a.bssSize
	a.bssSize += 8
	afterLabel := renvoAsmNewLabel(a)
	standardLabel := renvoAsmNewLabel(a)
	standardErrorLabel := renvoAsmNewLabel(a)
	handleReadyLabel := renvoAsmNewLabel(a)
	sequentialLabel := renvoAsmNewLabel(a)
	positionFailedLabel := renvoAsmNewLabel(a)
	ioFailedLabel := renvoAsmNewLabel(a)
	restoreLabel := renvoAsmNewLabel(a)
	failLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, label)
	// Preserve the helper return address and its internal-convention arguments:
	// x3=fd, x4=buffer, x1=count, x2=offset.
	renvoAarch64AsmPushReg(a, renvoAarch64RegLr)
	renvoAarch64AsmPushReg(a, renvoAarch64RegRdi)
	renvoAarch64AsmPushReg(a, renvoAarch64RegRsi)
	renvoAarch64AsmPushReg(a, renvoAarch64RegRdx)
	renvoAarch64AsmPushReg(a, renvoAarch64RegRcx)
	renvoAarch64AsmLoadRegMem(a, 9, 31, 48, 8)
	if isWrite {
		renvoAarch64AsmCmpRegImm(a, 9, 1)
		renvoAarch64AsmBCondLabel(a, standardLabel, 0)
		renvoAarch64AsmCmpRegImm(a, 9, 2)
		renvoAarch64AsmBCondLabel(a, standardErrorLabel, 0)
	} else {
		renvoAarch64AsmCmpRegImm(a, 9, 0)
		renvoAarch64AsmBCondLabel(a, standardLabel, 0)
	}
	renvoAarch64AsmStoreRegMem(a, 9, 31, 48, 8)
	renvoAarch64AsmJmpLabel(a, handleReadyLabel)
	renvoAsmMarkLabel(a, standardLabel)
	standardHandle := -10
	if isWrite {
		standardHandle = -11
	}
	renvoAarch64AsmMovRegImm(a, 0, standardHandle)
	renvoWinArm64CallImport(a, renvoWinImportGetStdHandle)
	renvoAarch64AsmStoreRegMem(a, 0, 31, 48, 8)
	renvoAarch64AsmJmpLabel(a, handleReadyLabel)
	renvoAsmMarkLabel(a, standardErrorLabel)
	renvoAarch64AsmMovRegImm(a, 0, -12)
	renvoWinArm64CallImport(a, renvoWinImportGetStdHandle)
	renvoAarch64AsmStoreRegMem(a, 0, 31, 48, 8)
	renvoAsmMarkLabel(a, handleReadyLabel)
	// Negative offsets use and advance the current file position. Non-negative
	// offsets are positional: preserve the handle's position around the I/O.
	renvoAarch64AsmLoadRegMem(a, 9, 31, 0, 8)
	renvoAarch64AsmCmpRegImm(a, 9, 0)
	renvoAarch64AsmBCondLabel(a, sequentialLabel, 11)

	// Save the current position.
	renvoAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	renvoAarch64AsmMovRegImm(a, 1, 0)
	renvoAarch64AsmMovRegImm(a, 2, 0)
	renvoAarch64AsmMovRegImm(a, 3, 1)
	renvoWinArm64CallImport(a, renvoWinImportSetFilePointer)
	renvoAarch64AsmMovRegImm(a, 9, 0xffffffff)
	renvoAarch64AsmCmpRegReg(a, 0, 9)
	renvoAarch64AsmBCondLabel(a, positionFailedLabel, 0)
	renvoAarch64AsmMovRegAbs(a, 9, posOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, 0, 9, 0, 8)

	// Seek to the requested offset.
	renvoAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	renvoAarch64AsmLoadRegMem(a, 1, 31, 0, 8)
	renvoAarch64AsmMovRegImm(a, 2, 0)
	renvoAarch64AsmMovRegImm(a, 3, 0)
	renvoWinArm64CallImport(a, renvoWinImportSetFilePointer)
	renvoAarch64AsmMovRegImm(a, 9, 0xffffffff)
	renvoAarch64AsmCmpRegReg(a, 0, 9)
	renvoAarch64AsmBCondLabel(a, ioFailedLabel, 0)

	renvoWinArm64EmitKernelReadWriteCall(a, importID, countOff)
	renvoAarch64AsmCmpRegImm(a, 0, 0)
	renvoAarch64AsmBCondLabel(a, ioFailedLabel, 0)
	renvoAarch64AsmMovRegAbs(a, 9, countOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, 9, 9, 0, 8)
	renvoAarch64AsmJmpLabel(a, restoreLabel)
	renvoAsmMarkLabel(a, ioFailedLabel)
	renvoAarch64AsmMovRegImm(a, 9, -1)
	renvoAsmMarkLabel(a, restoreLabel)
	renvoAarch64AsmMovRegAbs(a, 10, resultOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, 9, 10, 0, 8)
	renvoAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	renvoAarch64AsmMovRegAbs(a, 9, posOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, 1, 9, 0, 8)
	renvoAarch64AsmMovRegImm(a, 2, 0)
	renvoAarch64AsmMovRegImm(a, 3, 0)
	renvoWinArm64CallImport(a, renvoWinImportSetFilePointer)
	renvoAarch64AsmMovRegAbs(a, 9, resultOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, 0, 9, 0, 8)
	renvoAarch64AsmJmpLabel(a, doneLabel)

	renvoAsmMarkLabel(a, positionFailedLabel)
	renvoAarch64AsmMovRegImm(a, 0, -1)
	renvoAarch64AsmJmpLabel(a, doneLabel)

	renvoAsmMarkLabel(a, sequentialLabel)
	renvoWinArm64EmitKernelReadWriteCall(a, importID, countOff)
	renvoAarch64AsmCmpRegImm(a, 0, 0)
	renvoAarch64AsmBCondLabel(a, failLabel, 0)
	renvoAarch64AsmMovRegAbs(a, 9, countOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, 0, 9, 0, 8)
	renvoAarch64AsmJmpLabel(a, doneLabel)
	renvoAsmMarkLabel(a, failLabel)
	renvoAarch64AsmMovRegImm(a, 0, -1)
	renvoAsmMarkLabel(a, doneLabel)
	renvoAarch64AsmAddRegImm(a, 31, 31, 64)
	renvoAarch64AsmPopReg(a, renvoAarch64RegLr)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return label
}

func renvoWinArm64EmitKernelReadWriteCall(a *renvoAsm, importID int, countOff int) {
	renvoAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	renvoAarch64AsmLoadRegMem(a, 1, 31, 32, 8)
	renvoAarch64AsmLoadRegMem(a, 2, 31, 16, 8)
	renvoAarch64AsmMovRegAbs(a, 3, countOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 4, 0)
	renvoWinArm64CallImport(a, importID)
}

func renvoAsmPatchWindowsArm64(a *renvoAsm, layout renvoWinImportLayout) {
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == renvoAbsWinImportReloc {
			target = renvoWinImportIATRVA(layout, r.off)
		} else if r.kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + r.off
		}
		insn := renvoGet32At(a.code, r.at)
		reg := insn & 31
		pc := a.codeOffset + r.at
		delta := (target >> 12) - (pc >> 12)
		imm := delta & 0x1fffff
		renvoPut32At(a.code, r.at, 0x90000000|((imm&3)<<29)|(((imm>>2)&0x7ffff)<<5)|reg)
		renvoPut32At(a.code, r.at+4, 0x91000000|((target&0xfff)<<10)|(reg<<5)|reg)
		renvoPut32At(a.code, r.at+8, 0xd503201f)
		renvoPut32At(a.code, r.at+12, 0xd503201f)
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
		imports = renvoAppendWinImports(a)
	}
	renvoAsmPatchWindowsArm64(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = renvoAppendPEHeader64(out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
	// Windows on ARM64 requires the modern PE subsystem contract; unlike the
	// x86 targets, there is no legacy Windows 4.x loader to preserve.
	out[0xc0] = 6
	out[0xc2] = 1
	out[0xc4] = 1
	out[0xc8] = 6
	out[0xca] = 1
	out[0x9a] = 3
	out[0x96] = 0x22
	out[0xde] = 0x00
	out[0xdf] = 0x81
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize+dataRawSize)
	return out
}
