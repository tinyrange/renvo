package main

const rtgLinuxAmd64CodeOffset = 0x78

const rtgLinuxAmd64SysReadSeq = 0
const rtgLinuxAmd64SysWriteSeq = 1
const rtgLinuxAmd64SysOpen = 2
const rtgLinuxAmd64SysClose = 3
const rtgLinuxAmd64SysReadAt = 17
const rtgLinuxAmd64SysWriteAt = 18
const rtgLinuxAmd64SysFchmod = 91

func rtgAmd64AsmPrepareReadWriteBuf(a *rtgAsm) {
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmEmit16(a, 0x5a51)
}

func rtgAmd64AsmMoveOffsetArg(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc28949)
}

func compileLinuxAmd64(input []int, output int) int {
	rtgSetTarget(rtgTargetLinuxAmd64)
	return rtgCompileAmd64(input, output)
}

func compileWindowsAmd64(input []int, output int) int {
	rtgSetTarget(rtgTargetWindowsAmd64)
	return rtgCompileAmd64(input, output)
}

func rtgCompileAmd64(input []int, output int) int {
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
	var result rtgCompileResult
	result = rtgTryCompileScalarProgramAmd64(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func rtgTryCompileScalarProgramAmd64(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
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
	a := &g.asm
	rtgAsmInit(a)
	a.codeOffset = rtgLinuxAmd64CodeOffset
	if rtgTargetIsWindows() {
		a.codeOffset = rtgWinSectionRVA
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
	if !rtgEmitProgramEntryArgsAmd64(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	if !rtgEmitProgramPanicCheck(&g) {
		var result rtgCompileResult
		return result
	}
	if rtgTargetIsWindows() {
		rtgAsmCopyPrimaryToTertiary(a)
		rtgWinAmd64CallImport(a, rtgWinImportExitProcess, 40)
		rtgAsmRet(a)
	} else {
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmPrimaryImm(a, 60)
		rtgAsmSyscall(a)
	}
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !rtgEmitScalarFunctionScratch(&g, i) {
			var result rtgCompileResult
			return result
		}
	}
	var data []byte
	if rtgTargetIsWindows() {
		data = rtgAsmImageWindowsAmd64(a)
	} else {
		data = rtgAsmImageAmd64(a)
	}
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgsAmd64(g *rtgLinearGen, appIndex int) bool {
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
	if rtgTargetIsWindows() {
		argsTextOff := g.asm.bssSize
		g.asm.bssSize += 32768
		argsLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		rtgAsmBuildWindowsArgvEnvSlicesAmd64(&g.asm, argsOff, argsTextOff, argsLenOff, envDataOff, envLenOff)
	} else {
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		rtgAsmBuildArgvEnvSlicesAmd64(&g.asm, argsOff, envDataOff, envLenOff)
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

func rtgAsmBuildWindowsArgvEnvSlicesAmd64(a *rtgAsm, bssOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	skipLabel := rtgAsmNewLabel(a)
	startLabel := rtgAsmNewLabel(a)
	skipCharLabel := rtgAsmNewLabel(a)
	copyLabel := rtgAsmNewLabel(a)
	notQuoteLabel := rtgAsmNewLabel(a)
	copyCharLabel := rtgAsmNewLabel(a)
	argDoneLabel := rtgAsmNewLabel(a)
	finishLabel := rtgAsmNewLabel(a)

	rtgWinAmd64CallImport(a, rtgWinImportGetCommandLineA, 40)
	rtgAsmEmit24(a, 0xc68948)
	rtgAsmScratchBssAddr(a, bssOff)
	rtgAsmPrimaryBssAddr(a, argsTextOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmEmit24(a, 0xdb314d)

	rtgAsmMarkLabel(a, skipLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, 0)
	rtgAsmJzLabel(a, finishLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, ' ')
	rtgAsmJzLabel(a, skipCharLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, 9)
	rtgAsmJnzLabel(a, startLabel)
	rtgAsmMarkLabel(a, skipCharLabel)
	rtgAsmEmit24(a, 0xc6ff48)
	rtgAsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, startLabel)
	rtgAsmEmit24(a, 0x128949)
	rtgAsmEmit16(a, 0xc931)
	rtgAsmEmit24(a, 0xc03145)
	rtgAsmMarkLabel(a, copyLabel)
	rtgAsmEmit16(a, 0x068a)
	rtgAsmEmit2(a, 0x3c, 0)
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmEmit2(a, 0x3c, '"')
	rtgAsmJnzLabel(a, notQuoteLabel)
	rtgAsmEmit4(a, 0x41, 0x83, 0xf0, 1)
	rtgAsmEmit24(a, 0xc6ff48)
	rtgAsmJmpMarkLabel(a, copyLabel, notQuoteLabel)
	rtgAsmEmit4(a, 0x41, 0x83, 0xf8, 0)
	rtgAsmJnzLabel(a, copyCharLabel)
	rtgAsmEmit2(a, 0x3c, ' ')
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmEmit2(a, 0x3c, 9)
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmMarkLabel(a, copyCharLabel)
	rtgAsmEmit16(a, 0x0288)
	rtgAsmEmit24(a, 0xc6ff48)
	rtgAsmEmit24(a, 0xc2ff48)
	rtgAsmEmit24(a, 0xc1ff48)
	rtgAsmJmpLabel(a, copyLabel)

	rtgAsmMarkLabel(a, argDoneLabel)
	rtgAsmEmit3(a, 0xc6, 0x02, 0)
	rtgAsmEmit24(a, 0xc2ff48)
	rtgAsmEmit32(a, 0x084a8949)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmEmit2(a, 0x3c, 0)
	rtgAsmJzLabel(a, finishLabel)
	rtgAsmEmit24(a, 0xc6ff48)
	rtgAsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, finishLabel)
	rtgAsmEmit24(a, 0xd8894c)
	rtgAsmStorePrimaryBss(a, argsLenOff)

	envLoopLabel := rtgAsmNewLabel(a)
	envStringLabel := rtgAsmNewLabel(a)
	envStringDoneLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgWinAmd64CallImport(a, rtgWinImportGetEnvironmentStringsA, 40)
	rtgAsmEmit24(a, 0xc68948)
	rtgAsmScratchBssAddr(a, envOff)
	rtgAsmEmit24(a, 0xdb314d)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, 0)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit24(a, 0x328949)
	rtgAsmEmit16(a, 0xc931)
	rtgAsmMarkLabel(a, envStringLabel)
	rtgAsmEmit4(a, 0x80, 0x3c, 0x0e, 0)
	rtgAsmJzLabel(a, envStringDoneLabel)
	rtgAsmEmit24(a, 0xc1ff48)
	rtgAsmJmpMarkLabel(a, envStringLabel, envStringDoneLabel)
	rtgAsmEmit32(a, 0x084a8949)
	rtgAsmEmit24(a, 0xce0148)
	rtgAsmEmit24(a, 0xc6ff48)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	rtgAsmEmit24(a, 0xd8894c)
	rtgAsmStorePrimaryBss(a, envLenOff)

	rtgAsmPrimaryBssAddr(a, bssOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryBss(a, argsLenOff)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPrimaryBssAddr(a, envOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmLoadPrimaryBss(a, envLenOff)
	rtgAsmCopyPrimaryToCallWord4(a)
	rtgAsmCopyPrimaryToCallWord5(a)
}

func rtgAsmBuildArgvEnvSlicesAmd64(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envScanLabel := rtgAsmNewLabel(a)
	envStartLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit32(a, 0x24048b48)
	rtgAsmEmit24(a, 0xc08949)
	rtgAsmEmit32(a, 0x244c8d4c)
	rtgAsmEmit8(a, 0x8)
	rtgAsmScratchBssAddr(a, bssOff)
	rtgAsmEmit32(a, 0x4dd4894d)
	rtgAsmEmit16(a, 0xdb31)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 0xc3394d)
	rtgAsmEmit16(a, 0x8d0f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, doneLabel)
	rtgAsmEmit32(a, 0xd93c8b4b)
	rtgAsmEmit32(a, 0x483a8949)
	rtgAsmEmit16(a, 0xc031)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAsmEmit32(a, 0x00073c80)
	rtgAsmJzLabel(a, afterLenLabel)
	rtgAsmEmit24(a, 0xc0ff48)
	rtgAsmJmpMarkLabel(a, strlenLabel, afterLenLabel)
	rtgAsmEmit32(a, 0x08428949)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmJmpMarkLabel(a, loopLabel, doneLabel)

	rtgAsmEmit32(a, 0x244c8d4c)
	rtgAsmEmit8(a, 0x8)
	rtgAsmMarkLabel(a, envScanLabel)
	rtgAsmEmit32(a, 0x00398349)
	rtgAsmJzLabel(a, envStartLabel)
	rtgAsmEmit32(a, 0x08c18349)
	rtgAsmJmpMarkLabel(a, envScanLabel, envStartLabel)
	rtgAsmEmit32(a, 0x08c18349)
	rtgAsmScratchBssAddr(a, envOff)
	rtgAsmEmit24(a, 0xdb314d)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit32(a, 0xd93c8b4b)
	rtgAsmEmit32(a, 0x00ff8348)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit32(a, 0x483a8949)
	rtgAsmEmit16(a, 0xc031)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAsmEmit32(a, 0x00073c80)
	rtgAsmJzLabel(a, envAfterLenLabel)
	rtgAsmEmit24(a, 0xc0ff48)
	rtgAsmJmpMarkLabel(a, envStrlenLabel, envAfterLenLabel)
	rtgAsmEmit32(a, 0x08428949)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	rtgAsmEmit24(a, 0xd8894c)
	rtgAsmStorePrimaryBss(a, envLenOff)

	rtgAsmEmit32(a, 0x4ce7894c)
	rtgAsmEmit32(a, 0x894cc689)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmPrimaryBssAddr(a, envOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmLoadPrimaryBss(a, envLenOff)
	rtgAsmCopyPrimaryToCallWord4(a)
	rtgAsmCopyPrimaryToCallWord5(a)
}

func rtgAsmImageAmd64(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := loadFileSize + a.bssSize
	if rtgCompilerStripSymbols {
		oldCodeLen := len(a.code)
		var out []byte
		out = a.code[:loadFileSize]
		for i := 0; i < oldCodeLen; i++ {
			src := oldCodeLen - 1 - i
			out[a.codeOffset+src] = out[src]
		}
		var header []byte
		header = rtgAppendElfHeaderAmd64(header, a.codeOffset, loadFileSize, memSize, 0)
		for i := 0; i < len(header); i++ {
			out[i] = header[i]
		}
		pos := a.codeOffset + oldCodeLen
		for i := 0; i < len(a.data); i++ {
			out[pos+i] = a.data[i]
		}
		return out
	}
	sec := rtgBuildElf64SymbolSections(a, 0x400000, a.codeOffset, loadFileSize)
	finalSize := sec.shoff + 448
	out := make([]byte, finalSize)
	out = out[:0]
	out = rtgAppendElfHeaderAmd64(out, a.codeOffset, loadFileSize, memSize, sec.shoff)
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
	out = rtgAppendElf64SectionHeaders(out, &sec, a, 0x400000)
	return out
}

func rtgAppendElfHeaderAmd64(out []byte, entryOff int, fileSize int, memSize int, shoff int) []byte {
	base := 0x400000

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
	out = rtgAppend16(out, 0x3e)
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

func rtgAsmImageWindowsAmd64(a *rtgAsm) []byte {
	for (a.codeOffset+len(a.code))%8 != 0 {
		a.code = append(a.code, 0)
	}
	textVirtualSize := len(a.code)
	textRawSize := rtgAlignValue(textVirtualSize, rtgWinFileAlign)
	dataRVA := rtgAlignValue(a.codeOffset+textVirtualSize, rtgWinSectionAlign)
	a.dataOffset = dataRVA
	var imports rtgWinImportLayout
	if rtgAsmHasWinImportRelocs(a) {
		imports = rtgAppendWinImports(a, true)
	}
	rtgAsmPatchWindows(a, imports, rtgWinImageBase, true)
	dataRawSize := rtgAlignValue(len(a.data), rtgWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (rtgWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = rtgAppendPEHeader64(out, a.codeOffset, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
