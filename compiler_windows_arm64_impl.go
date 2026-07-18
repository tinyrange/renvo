package main

const rtgWinArm64Machine = 0xaa64
const rtgWinArm64ImageBase = 0x140000000

func compileWindowsArm64(input []int, output int) int {
	return compileWindowsArm64Arena(input, output, 0)
}

func compileWindowsArm64Arena(input []int, output int, arenaSize int) int {
	rtgSetTarget(rtgTargetWindowsArm64)
	return rtgCompileAarch64(input, output, arenaSize)
}

func rtgEmitProgramEntryArgsWindowsArm64(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(g.meta, app.resultType) {
		return false
	}
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
	rtgAsmBuildWindowsArgvEnvSlicesArm64(&g.asm, argsOff, argsTextOff, argsLenOff, envOff, envLenOff)
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	return rtgTypeIsStringSlice(g.meta, second.typ)
}

func rtgAsmBuildWindowsArgvEnvSlicesArm64(a *rtgAsm, argsOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	skipLabel := rtgAsmNewLabel(a)
	startLabel := rtgAsmNewLabel(a)
	copyLabel := rtgAsmNewLabel(a)
	quoteLabel := rtgAsmNewLabel(a)
	setQuoteLabel := rtgAsmNewLabel(a)
	copyCharLabel := rtgAsmNewLabel(a)
	argDoneLabel := rtgAsmNewLabel(a)
	argsDoneLabel := rtgAsmNewLabel(a)

	rtgWinArm64CallImport(a, rtgWinImportGetCommandLineA)
	rtgAarch64AsmMovRegReg(a, 9, 0) // source
	rtgAarch64AsmMovRegAbs(a, 10, argsOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegAbs(a, 11, argsTextOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 12, 0) // argument count
	rtgAsmMarkLabel(a, skipLabel)
	rtgAarch64AsmLoadRegMem(a, 13, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 13, 0)
	rtgAarch64AsmBCondLabel(a, argsDoneLabel, 0)
	rtgAarch64AsmCmpRegImm(a, 13, ' ')
	rtgAarch64AsmBCondLabel(a, startLabel, 1)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmJmpLabel(a, skipLabel)
	rtgAsmMarkLabel(a, startLabel)
	rtgAarch64AsmCmpRegImm(a, 13, 9)
	rtgAarch64AsmBCondLabel(a, copyLabel, 1)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, copyLabel)
	rtgAarch64AsmMovRegReg(a, 14, 11) // copied argument start
	rtgAarch64AsmMovRegImm(a, 15, 0)  // copied length
	rtgAarch64AsmMovRegImm(a, 17, 0)  // quote state
	copyLoopLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, copyLoopLabel)
	rtgAarch64AsmLoadRegMem(a, 13, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 13, 0)
	rtgAarch64AsmBCondLabel(a, argDoneLabel, 0)
	rtgAarch64AsmCmpRegImm(a, 13, '"')
	rtgAarch64AsmBCondLabel(a, quoteLabel, 0)
	rtgAarch64AsmCmpRegImm(a, 17, 0)
	rtgAarch64AsmBCondLabel(a, copyCharLabel, 1)
	rtgAarch64AsmCmpRegImm(a, 13, ' ')
	rtgAarch64AsmBCondLabel(a, argDoneLabel, 0)
	rtgAarch64AsmCmpRegImm(a, 13, 9)
	rtgAarch64AsmBCondLabel(a, argDoneLabel, 0)
	rtgAsmMarkLabel(a, copyCharLabel)
	rtgAarch64AsmStoreRegMem(a, 13, 11, 0, 1)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmAddRegImm(a, 15, 15, 1)
	rtgAarch64AsmJmpLabel(a, copyLoopLabel)
	rtgAsmMarkLabel(a, quoteLabel)
	rtgAarch64AsmCmpRegImm(a, 17, 0)
	rtgAarch64AsmBCondLabel(a, setQuoteLabel, 0)
	rtgAarch64AsmMovRegImm(a, 17, 0)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmJmpLabel(a, copyLoopLabel)
	rtgAsmMarkLabel(a, setQuoteLabel)
	rtgAarch64AsmMovRegImm(a, 17, 1)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmJmpLabel(a, copyLoopLabel)

	rtgAsmMarkLabel(a, argDoneLabel)
	rtgAarch64AsmStoreRegMem(a, 31, 11, 0, 1)
	rtgAarch64AsmStoreRegMem(a, 14, 10, 0, 8)
	rtgAarch64AsmStoreRegMem(a, 15, 10, 8, 8)
	rtgAarch64AsmAddRegImm(a, 10, 10, 16)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmAddRegImm(a, 12, 12, 1)
	rtgAarch64AsmCmpRegImm(a, 13, 0)
	rtgAarch64AsmBCondLabel(a, argsDoneLabel, 0)
	rtgAarch64AsmAddRegImm(a, 9, 9, 1)
	rtgAarch64AsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, argsDoneLabel)
	rtgAarch64AsmMovRegAbs(a, 9, argsLenOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, 12, 9, 0, 8)

	envLoopLabel := rtgAsmNewLabel(a)
	envLengthLabel := rtgAsmNewLabel(a)
	envStringDoneLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgWinArm64CallImport(a, rtgWinImportGetEnvironmentStringsA)
	rtgAarch64AsmMovRegReg(a, 9, 0)
	rtgAarch64AsmMovRegAbs(a, 10, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 11, 0)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAarch64AsmLoadRegMem(a, 12, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 12, 0)
	rtgAarch64AsmBCondLabel(a, envDoneLabel, 0)
	rtgAarch64AsmStoreRegMem(a, 9, 10, 0, 8)
	rtgAarch64AsmMovRegImm(a, 13, 0)
	rtgAsmMarkLabel(a, envLengthLabel)
	rtgAarch64AsmAddRegReg(a, 14, 9, 13)
	rtgAarch64AsmLoadRegMem(a, 12, 14, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 12, 0)
	rtgAarch64AsmBCondLabel(a, envStringDoneLabel, 0)
	rtgAarch64AsmAddRegImm(a, 13, 13, 1)
	rtgAarch64AsmJmpLabel(a, envLengthLabel)
	rtgAsmMarkLabel(a, envStringDoneLabel)
	rtgAarch64AsmStoreRegMem(a, 13, 10, 8, 8)
	rtgAarch64AsmAddRegImm(a, 10, 10, 16)
	rtgAarch64AsmAddRegImm(a, 13, 13, 1)
	rtgAarch64AsmAddRegReg(a, 9, 9, 13)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAarch64AsmMovRegAbs(a, 9, envLenOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, 11, 9, 0, 8)

	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRdi, argsOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegAbs(a, 9, argsLenOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRsi, 9, 0, 8)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRsi)
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRcx, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR8, 11)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR9, 11)
}

func rtgWinArm64CallImport(a *rtgAsm, importID int) {
	rtgAarch64AsmMovRegAbs(a, 16, importID, rtgAbsWinImportReloc)
	rtgAarch64AsmLoadRegMem(a, 16, 16, 0, 8)
	rtgAarch64AsmEmit(a, 0xd63f0200) // blr x16
}

func rtgWinArm64CallStaticImport(a *rtgAsm, importID int, wordCount int) {
	registerWords := wordCount
	if registerWords > 8 {
		registerWords = 8
	}
	for i := 0; i < registerWords; i++ {
		rtgAarch64AsmPopReg(a, i)
	}
	stackWords := wordCount - registerWords

	// RTG's expression stack uses a 16-byte slot for every word. Windows ARM64
	// stack arguments are packed into 8-byte slots, so construct a temporary
	// ABI call area while retaining the exact pending-argument stack pointer.
	rtgAarch64AsmEmit(a, 0x910003e9) // add x9, sp, #0
	savedSPOff := stackWords * 8
	allocation := rtgAlignValue(savedSPOff+8, 16)
	rtgAarch64AsmAddRegImm(a, 31, 31, -allocation)
	for i := 0; i < stackWords; i++ {
		rtgAarch64AsmLoadRegMem(a, 10, 9, i*16, 8)
		rtgAarch64AsmStoreRegMem(a, 10, 31, i*8, 8)
	}
	rtgAarch64AsmStoreRegMem(a, 9, 31, savedSPOff, 8)
	rtgWinArm64CallImport(a, importID)
	rtgAarch64AsmMovRegReg(a, 10, 0)
	rtgAarch64AsmLoadRegMem(a, 9, 31, savedSPOff, 8)
	// ADD (immediate), unlike ORR-based MOV, can address SP as both operands.
	rtgAarch64AsmEmit(a, 0x9100013f)
	rtgAarch64AsmAddRegImm(a, 31, 31, stackWords*16)
	rtgAarch64AsmMovRegReg(a, 0, 10)
}

func rtgWinArm64TestRegImm(a *rtgAsm, reg int, imm int) {
	rtgAarch64AsmMovRegImm(a, 15, imm)
	rtgAarch64AsmEmit(a, 0xea00001f|(15<<16)|(reg<<5))
}

func rtgWinArm64TranslateCreateFileFlags(a *rtgAsm) {
	notReadWriteLabel := rtgAsmNewLabel(a)
	accessDoneLabel := rtgAsmNewLabel(a)
	noCreateLabel := rtgAsmNewLabel(a)
	createDoneLabel := rtgAsmNewLabel(a)

	rtgAarch64AsmMovRegImm(a, 1, -2147483648)
	rtgWinArm64TestRegImm(a, 0, 2)
	rtgAarch64AsmBCondLabel(a, notReadWriteLabel, 0)
	rtgAarch64AsmMovRegImm(a, 1, -1073741824)
	rtgAarch64AsmJmpLabel(a, accessDoneLabel)
	rtgAsmMarkLabel(a, notReadWriteLabel)
	rtgWinArm64TestRegImm(a, 0, 1)
	rtgAarch64AsmBCondLabel(a, accessDoneLabel, 0)
	rtgAarch64AsmMovRegImm(a, 1, 0x40000000)
	rtgAsmMarkLabel(a, accessDoneLabel)

	rtgAarch64AsmMovRegImm(a, 4, 3)
	rtgWinArm64TestRegImm(a, 0, 64)
	rtgAarch64AsmBCondLabel(a, noCreateLabel, 0)
	rtgAarch64AsmMovRegImm(a, 4, 4)
	rtgWinArm64TestRegImm(a, 0, 512)
	rtgAarch64AsmBCondLabel(a, createDoneLabel, 0)
	rtgAarch64AsmMovRegImm(a, 4, 2)
	rtgAarch64AsmJmpLabel(a, createDoneLabel)
	rtgAsmMarkLabel(a, noCreateLabel)
	rtgWinArm64TestRegImm(a, 0, 512)
	rtgAarch64AsmBCondLabel(a, createDoneLabel, 0)
	rtgAarch64AsmMovRegImm(a, 4, 5)
	rtgAsmMarkLabel(a, createDoneLabel)
	rtgAarch64AsmMovRegImm(a, 2, 3)
	rtgAarch64AsmMovRegImm(a, 3, 0)
	rtgAarch64AsmMovRegImm(a, 5, 0x80)
	rtgAarch64AsmMovRegImm(a, 6, 0)
}

func rtgWinArm64EmitReadWriteHelper(g *rtgLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = rtgAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = rtgAsmNewLabel(a)
	}
	label := g.winReadLabel
	importID := rtgWinImportReadFile
	if isWrite {
		label = g.winWriteLabel
		importID = rtgWinImportWriteFile
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	resultOff := a.bssSize
	a.bssSize += 8
	afterLabel := rtgAsmNewLabel(a)
	standardLabel := rtgAsmNewLabel(a)
	standardErrorLabel := rtgAsmNewLabel(a)
	handleReadyLabel := rtgAsmNewLabel(a)
	sequentialLabel := rtgAsmNewLabel(a)
	positionFailedLabel := rtgAsmNewLabel(a)
	ioFailedLabel := rtgAsmNewLabel(a)
	restoreLabel := rtgAsmNewLabel(a)
	failLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, label)
	// Preserve the helper return address and its internal-convention arguments:
	// x3=fd, x4=buffer, x1=count, x2=offset.
	rtgAarch64AsmPushReg(a, rtgAarch64RegLr)
	rtgAarch64AsmPushReg(a, rtgAarch64RegRdi)
	rtgAarch64AsmPushReg(a, rtgAarch64RegRsi)
	rtgAarch64AsmPushReg(a, rtgAarch64RegRdx)
	rtgAarch64AsmPushReg(a, rtgAarch64RegRcx)
	rtgAarch64AsmLoadRegMem(a, 9, 31, 48, 8)
	if isWrite {
		rtgAarch64AsmCmpRegImm(a, 9, 1)
		rtgAarch64AsmBCondLabel(a, standardLabel, 0)
		rtgAarch64AsmCmpRegImm(a, 9, 2)
		rtgAarch64AsmBCondLabel(a, standardErrorLabel, 0)
	} else {
		rtgAarch64AsmCmpRegImm(a, 9, 0)
		rtgAarch64AsmBCondLabel(a, standardLabel, 0)
	}
	rtgAarch64AsmStoreRegMem(a, 9, 31, 48, 8)
	rtgAarch64AsmJmpLabel(a, handleReadyLabel)
	rtgAsmMarkLabel(a, standardLabel)
	standardHandle := -10
	if isWrite {
		standardHandle = -11
	}
	rtgAarch64AsmMovRegImm(a, 0, standardHandle)
	rtgWinArm64CallImport(a, rtgWinImportGetStdHandle)
	rtgAarch64AsmStoreRegMem(a, 0, 31, 48, 8)
	rtgAarch64AsmJmpLabel(a, handleReadyLabel)
	rtgAsmMarkLabel(a, standardErrorLabel)
	rtgAarch64AsmMovRegImm(a, 0, -12)
	rtgWinArm64CallImport(a, rtgWinImportGetStdHandle)
	rtgAarch64AsmStoreRegMem(a, 0, 31, 48, 8)
	rtgAsmMarkLabel(a, handleReadyLabel)
	// Negative offsets use and advance the current file position. Non-negative
	// offsets are positional: preserve the handle's position around the I/O.
	rtgAarch64AsmLoadRegMem(a, 9, 31, 0, 8)
	rtgAarch64AsmCmpRegImm(a, 9, 0)
	rtgAarch64AsmBCondLabel(a, sequentialLabel, 11)

	// Save the current position.
	rtgAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	rtgAarch64AsmMovRegImm(a, 1, 0)
	rtgAarch64AsmMovRegImm(a, 2, 0)
	rtgAarch64AsmMovRegImm(a, 3, 1)
	rtgWinArm64CallImport(a, rtgWinImportSetFilePointer)
	rtgAarch64AsmMovRegImm(a, 9, 0xffffffff)
	rtgAarch64AsmCmpRegReg(a, 0, 9)
	rtgAarch64AsmBCondLabel(a, positionFailedLabel, 0)
	rtgAarch64AsmMovRegAbs(a, 9, posOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, 0, 9, 0, 8)

	// Seek to the requested offset.
	rtgAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	rtgAarch64AsmLoadRegMem(a, 1, 31, 0, 8)
	rtgAarch64AsmMovRegImm(a, 2, 0)
	rtgAarch64AsmMovRegImm(a, 3, 0)
	rtgWinArm64CallImport(a, rtgWinImportSetFilePointer)
	rtgAarch64AsmMovRegImm(a, 9, 0xffffffff)
	rtgAarch64AsmCmpRegReg(a, 0, 9)
	rtgAarch64AsmBCondLabel(a, ioFailedLabel, 0)

	rtgWinArm64EmitKernelReadWriteCall(a, importID, countOff)
	rtgAarch64AsmCmpRegImm(a, 0, 0)
	rtgAarch64AsmBCondLabel(a, ioFailedLabel, 0)
	rtgAarch64AsmMovRegAbs(a, 9, countOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, 9, 9, 0, 8)
	rtgAarch64AsmJmpLabel(a, restoreLabel)
	rtgAsmMarkLabel(a, ioFailedLabel)
	rtgAarch64AsmMovRegImm(a, 9, -1)
	rtgAsmMarkLabel(a, restoreLabel)
	rtgAarch64AsmMovRegAbs(a, 10, resultOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, 9, 10, 0, 8)
	rtgAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	rtgAarch64AsmMovRegAbs(a, 9, posOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, 1, 9, 0, 8)
	rtgAarch64AsmMovRegImm(a, 2, 0)
	rtgAarch64AsmMovRegImm(a, 3, 0)
	rtgWinArm64CallImport(a, rtgWinImportSetFilePointer)
	rtgAarch64AsmMovRegAbs(a, 9, resultOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, 0, 9, 0, 8)
	rtgAarch64AsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, positionFailedLabel)
	rtgAarch64AsmMovRegImm(a, 0, -1)
	rtgAarch64AsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, sequentialLabel)
	rtgWinArm64EmitKernelReadWriteCall(a, importID, countOff)
	rtgAarch64AsmCmpRegImm(a, 0, 0)
	rtgAarch64AsmBCondLabel(a, failLabel, 0)
	rtgAarch64AsmMovRegAbs(a, 9, countOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, 0, 9, 0, 8)
	rtgAarch64AsmJmpLabel(a, doneLabel)
	rtgAsmMarkLabel(a, failLabel)
	rtgAarch64AsmMovRegImm(a, 0, -1)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAarch64AsmAddRegImm(a, 31, 31, 64)
	rtgAarch64AsmPopReg(a, rtgAarch64RegLr)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return label
}

func rtgWinArm64EmitKernelReadWriteCall(a *rtgAsm, importID int, countOff int) {
	rtgAarch64AsmLoadRegMem(a, 0, 31, 48, 8)
	rtgAarch64AsmLoadRegMem(a, 1, 31, 32, 8)
	rtgAarch64AsmLoadRegMem(a, 2, 31, 16, 8)
	rtgAarch64AsmMovRegAbs(a, 3, countOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 4, 0)
	rtgWinArm64CallImport(a, importID)
}

func rtgAsmPatchWindowsArm64(a *rtgAsm, layout rtgWinImportLayout) {
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsWinImportReloc {
			target = rtgWinImportIATRVA(layout, r.off)
		} else if r.kind == rtgAbsBssReloc {
			target = rtgAsmBssOffset(a) + r.off
		}
		insn := rtgGet32At(a.code, r.at)
		reg := insn & 31
		pc := a.codeOffset + r.at
		delta := (target >> 12) - (pc >> 12)
		imm := delta & 0x1fffff
		rtgPut32At(a.code, r.at, 0x90000000|((imm&3)<<29)|(((imm>>2)&0x7ffff)<<5)|reg)
		rtgPut32At(a.code, r.at+4, 0x91000000|((target&0xfff)<<10)|(reg<<5)|reg)
		rtgPut32At(a.code, r.at+8, 0xd503201f)
		rtgPut32At(a.code, r.at+12, 0xd503201f)
	}
}

func rtgAsmImageWindowsArm64(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	for (a.codeOffset+len(a.code))%8 != 0 {
		a.code = append(a.code, 0)
	}
	textVirtualSize := len(a.code)
	textRawSize := rtgAlignValue(textVirtualSize, rtgWinFileAlign)
	dataRVA := rtgAlignValue(a.codeOffset+textVirtualSize, rtgWinSectionAlign)
	a.dataOffset = dataRVA
	var imports rtgWinImportLayout
	if rtgAsmHasWinImportRelocs(a) {
		imports = rtgAppendWinImports(a)
	}
	rtgAsmPatchWindowsArm64(a, imports)
	dataRawSize := rtgAlignValue(len(a.data), rtgWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (rtgWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = rtgAppendPEHeader64(out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
	out = rtgAppendUntil(out, rtgWinHeadersSize+textRawSize)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = rtgAppendUntil(out, rtgWinHeadersSize+textRawSize+dataRawSize)
	return out
}
