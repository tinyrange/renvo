package main

const rtgLinuxAmd64CodeOffset = 0xb0

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
	return compileLinuxAmd64Arena(input, output, 0)
}

func compileLinuxAmd64Arena(input []int, output int, arenaSize int) int {
	rtgSetTarget(rtgTargetLinuxAmd64)
	return rtgCompileAmd64(input, output, arenaSize)
}

func compileWindowsAmd64(input []int, output int) int {
	return compileWindowsAmd64Arena(input, output, 0)
}

func compileWindowsAmd64Arena(input []int, output int, arenaSize int) int {
	rtgSetTarget(rtgTargetWindowsAmd64)
	return rtgCompileAmd64(input, output, arenaSize)
}

func rtgCompileAmd64(input []int, output int, arenaSize int) int {
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
	g.arenaSize = meta.arenaSize
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
	// This pre-relaxed instruction template is invariant except for its import
	// and BSS operands, whose relocations are recorded immediately after it.
	base := len(a.code)
	rtgAsmEmitText(a, "\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\xc6\x4c\x8d\x15\x00\x00\x00\x00\x48\x8d\x05\x00\x00\x00\x00\x50\x5a\x4d\x31\xdb\x80\x3e\x00\x0f\x84\x62\x00\x00\x00\x80\x3e\x20\x74\x05\x80\x3e\x09\x75\x05\x48\xff\xc6\xeb\xe8\x49\x89\x12\x31\xc9\x45\x31\xc0\x8a\x06\x3c\x00\x74\x28\x3c\x22\x75\x09\x41\x83\xf0\x01\x48\xff\xc6\xeb\xed\x41\x83\xf8\x00\x75\x08\x3c\x20\x74\x11\x3c\x09\x74\x0d\x88\x02\x48\xff\xc6\x48\xff\xc2\x48\xff\xc1\xeb\xd2\xc6\x02\x00\x48\xff\xc2\x49\x89\x4a\x08\x49\x83\xc2\x10\x49\xff\xc3\x3c\x00\x74\x08\x48\xff\xc6\xe9\x95\xff\xff\xff\x4c\x89\xd8\x48\x89\x05\x00\x00\x00\x00\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\xc6\x4c\x8d\x15\x00\x00\x00\x00\x4d\x31\xdb\x80\x3e\x00\x74\x23\x49\x89\x32\x31\xc9\x80\x3c\x0e\x00\x74\x05\x48\xff\xc1\xeb\xf5\x49\x89\x4a\x08\x48\x01\xce\x48\xff\xc6\x49\x83\xc2\x10\x49\xff\xc3\xeb\xd8\x4c\x89\xd8\x48\x89\x05\x00\x00\x00\x00\x48\x8d\x05\x00\x00\x00\x00\x50\x5f\x48\x8b\x05\x00\x00\x00\x00\x50\x5e\x50\x5a\x48\x8d\x05\x00\x00\x00\x00\x50\x59\x48\x8b\x05\x00\x00\x00\x00\x49\x89\xc0\x49\x89\xc1")
	rtgAsmAddWinImportReloc(a, base+6, rtgWinImportGetCommandLineA)
	rtgAsmAddAbsReloc(a, base+20, bssOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+27, argsTextOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+149, argsLenOff, rtgAbsBssReloc)
	rtgAsmAddWinImportReloc(a, base+159, rtgWinImportGetEnvironmentStringsA)
	rtgAsmAddAbsReloc(a, base+173, envOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+226, envLenOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+233, bssOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+242, argsLenOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+253, envOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, base+262, envLenOff, rtgAbsBssReloc)
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
	bssOffset := rtgAsmBssOffset(a)
	if rtgCompilerStripSymbols {
		oldCodeLen := len(a.code)
		var out []byte
		out = a.code[:loadFileSize]
		for i := 0; i < oldCodeLen; i++ {
			src := oldCodeLen - 1 - i
			out[a.codeOffset+src] = out[src]
		}
		var header []byte
		header = rtgAppendElfHeaderAmd64(header, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
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
	out = rtgAppendElfHeaderAmd64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
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

func rtgAppendElfHeaderAmd64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	base := 0x400000
	header := "\x7f\x45\x4c\x46\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x38\x00\x01\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00"
	out = append(out, header...)
	out[start+56] = 2
	out[start+68] = 5
	rtgPut32At(out, start+24, base+entryOff)
	rtgPut32At(out, start+40, shoff)
	if shoff != 0 {
		out[start+58] = 64
		out[start+60] = 7
		out[start+62] = 6
	}
	rtgPut32At(out, start+96, fileSize)
	rtgPut32At(out, start+104, fileSize)
	out = rtgAppendElf64LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
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
		imports = rtgAppendWinImports(a)
	}
	rtgAsmPatchWindows(a, imports)
	dataRawSize := rtgAlignValue(len(a.data), rtgWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (rtgWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = rtgAppendPEHeader64(out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
