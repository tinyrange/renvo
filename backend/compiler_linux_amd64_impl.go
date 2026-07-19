package main

const renvoLinuxAmd64CodeOffset = 0xb0

const renvoLinuxAmd64SysReadSeq = 0
const renvoLinuxAmd64SysWriteSeq = 1
const renvoLinuxAmd64SysOpen = 2
const renvoLinuxAmd64SysClose = 3
const renvoLinuxAmd64SysReadAt = 17
const renvoLinuxAmd64SysWriteAt = 18
const renvoLinuxAmd64SysFchmod = 91

func renvoAmd64AsmPrepareReadWriteBuf(a *renvoAsm) {
	trustNonNil(a)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmEmit16(a, 0x5a51)
}

func renvoAmd64AsmMoveOffsetArg(a *renvoAsm) {
	trustNonNil(a)
	renvoAsmEmit24(a, 0xc28949)
}

func compileLinuxAmd64(input []int, output int) int {
	return compileLinuxAmd64Arena(input, output, 0)
}

func compileLinuxAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}

func compileWindowsAmd64(input []int, output int) int {
	return compileWindowsAmd64Arena(input, output, 0)
}

func compileWindowsAmd64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindowsAmd64)
	return renvoCompileAmd64(input, output, arenaSize)
}

func renvoCompileAmd64(input []int, output int, arenaSize int) int {
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
	result = renvoTryCompileScalarProgramAmd64(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramAmd64(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	trustNonNil(p, meta)
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
	a.codeOffset = renvoLinuxAmd64CodeOffset
	if targetIsWindows() {
		a.codeOffset = renvoWinSectionRVA
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
	if !meta.panicEnabled {
		renvoAmd64InitRuntimeCheckRegs(&g)
	}
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) {
		var result renvoCompileResult
		return result
	}
	if !renvoEmitProgramEntryArgsAmd64(&g, appIndex) {
		var result renvoCompileResult
		return result
	}
	// Entry argument setup uses R12 as scratch, so restore all reserved runtime
	// check registers after it has consumed the process stack.
	if !meta.panicEnabled {
		renvoAmd64InitRuntimeCheckRegs(&g)
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		var result renvoCompileResult
		return result
	}
	if targetIsWindows() {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoWinAmd64CallImport(a, renvoWinImportExitProcess, 40)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 60)
		renvoAsmSyscall(a)
	}
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !renvoEmitScalarFunctionScratch(&g, i) {
			var result renvoCompileResult
			return result
		}
	}
	var data []byte
	if targetIsWindows() {
		data = renvoAsmImageWindowsAmd64(a)
	} else {
		data = renvoAsmImageAmd64(a)
	}
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoEmitProgramEntryArgsAmd64(g *renvoLinearGen, appIndex int) bool {
	trustNonNil(g)
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
	if targetIsWindows() {
		argsTextOff := g.asm.bssSize
		g.asm.bssSize += 32768
		argsLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		renvoAsmBuildWindowsArgvEnvSlicesAmd64(&g.asm, argsOff, argsTextOff, argsLenOff, envDataOff, envLenOff)
	} else {
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		renvoAsmBuildArgvEnvSlicesAmd64(&g.asm, argsOff, envDataOff, envLenOff)
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

func renvoAsmBuildWindowsArgvEnvSlicesAmd64(a *renvoAsm, bssOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	trustNonNil(a)
	// This pre-relaxed instruction template is invariant except for its import
	// and BSS operands, whose relocations are recorded immediately after it.
	base := len(a.code)
	renvoAsmEmitText(a, "\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\xc6\x4c\x8d\x15\x00\x00\x00\x00\x48\x8d\x05\x00\x00\x00\x00\x50\x5a\x4d\x31\xdb\x80\x3e\x00\x0f\x84\x62\x00\x00\x00\x80\x3e\x20\x74\x05\x80\x3e\x09\x75\x05\x48\xff\xc6\xeb\xe8\x49\x89\x12\x31\xc9\x45\x31\xc0\x8a\x06\x3c\x00\x74\x28\x3c\x22\x75\x09\x41\x83\xf0\x01\x48\xff\xc6\xeb\xed\x41\x83\xf8\x00\x75\x08\x3c\x20\x74\x11\x3c\x09\x74\x0d\x88\x02\x48\xff\xc6\x48\xff\xc2\x48\xff\xc1\xeb\xd2\xc6\x02\x00\x48\xff\xc2\x49\x89\x4a\x08\x49\x83\xc2\x10\x49\xff\xc3\x3c\x00\x74\x08\x48\xff\xc6\xe9\x95\xff\xff\xff\x4c\x89\xd8\x48\x89\x05\x00\x00\x00\x00\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\xc6\x4c\x8d\x15\x00\x00\x00\x00\x4d\x31\xdb\x80\x3e\x00\x74\x23\x49\x89\x32\x31\xc9\x80\x3c\x0e\x00\x74\x05\x48\xff\xc1\xeb\xf5\x49\x89\x4a\x08\x48\x01\xce\x48\xff\xc6\x49\x83\xc2\x10\x49\xff\xc3\xeb\xd8\x4c\x89\xd8\x48\x89\x05\x00\x00\x00\x00\x48\x8d\x05\x00\x00\x00\x00\x50\x5f\x48\x8b\x05\x00\x00\x00\x00\x50\x5e\x50\x5a\x48\x8d\x05\x00\x00\x00\x00\x50\x59\x48\x8b\x05\x00\x00\x00\x00\x49\x89\xc0\x49\x89\xc1")
	renvoAsmAddWinImportReloc(a, base+6, renvoWinImportGetCommandLineA)
	renvoAsmAddAbsReloc(a, base+20, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+27, argsTextOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+149, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, base+159, renvoWinImportGetEnvironmentStringsA)
	renvoAsmAddAbsReloc(a, base+173, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+226, envLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+233, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+242, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+253, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+262, envLenOff, renvoAbsBssReloc)
}

func renvoAsmBuildArgvEnvSlicesAmd64(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	trustNonNil(a)
	loopLabel := renvoAsmNewLabel(a)
	strlenLabel := renvoAsmNewLabel(a)
	afterLenLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	envScanLabel := renvoAsmNewLabel(a)
	envStartLabel := renvoAsmNewLabel(a)
	envLoopLabel := renvoAsmNewLabel(a)
	envStrlenLabel := renvoAsmNewLabel(a)
	envAfterLenLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)
	renvoAsmEmit32(a, 0x24048b48)
	renvoAsmEmit24(a, 0xc08949)
	renvoAsmEmit32(a, 0x244c8d4c)
	renvoAsmEmit8(a, 0x8)
	renvoAsmScratchBssAddr(a, bssOff)
	renvoAsmEmit32(a, 0x4dd4894d)
	renvoAsmEmit16(a, 0xdb31)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmit24(a, 0xc3394d)
	renvoAsmEmit16(a, 0x8d0f)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, doneLabel)
	renvoAsmEmit32(a, 0xd93c8b4b)
	renvoAsmEmit32(a, 0x483a8949)
	renvoAsmEmit16(a, 0xc031)
	renvoAsmMarkLabel(a, strlenLabel)
	renvoAsmEmit32(a, 0x00073c80)
	renvoAsmJzLabel(a, afterLenLabel)
	renvoAsmEmit24(a, 0xc0ff48)
	renvoAsmJmpMarkLabel(a, strlenLabel, afterLenLabel)
	renvoAsmEmit32(a, 0x08428949)
	renvoAsmEmit32(a, 0x10c28349)
	renvoAsmEmit24(a, 0xc3ff49)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)

	renvoAsmEmit32(a, 0x244c8d4c)
	renvoAsmEmit8(a, 0x8)
	renvoAsmMarkLabel(a, envScanLabel)
	renvoAsmEmit32(a, 0x00398349)
	renvoAsmJzLabel(a, envStartLabel)
	renvoAsmEmit32(a, 0x08c18349)
	renvoAsmJmpMarkLabel(a, envScanLabel, envStartLabel)
	renvoAsmEmit32(a, 0x08c18349)
	renvoAsmScratchBssAddr(a, envOff)
	renvoAsmEmit24(a, 0xdb314d)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAsmEmit32(a, 0xd93c8b4b)
	renvoAsmEmit32(a, 0x00ff8348)
	renvoAsmJzLabel(a, envDoneLabel)
	renvoAsmEmit32(a, 0x483a8949)
	renvoAsmEmit16(a, 0xc031)
	renvoAsmMarkLabel(a, envStrlenLabel)
	renvoAsmEmit32(a, 0x00073c80)
	renvoAsmJzLabel(a, envAfterLenLabel)
	renvoAsmEmit24(a, 0xc0ff48)
	renvoAsmJmpMarkLabel(a, envStrlenLabel, envAfterLenLabel)
	renvoAsmEmit32(a, 0x08428949)
	renvoAsmEmit32(a, 0x10c28349)
	renvoAsmEmit24(a, 0xc3ff49)
	renvoAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	renvoAsmEmit24(a, 0xd8894c)
	renvoAsmStorePrimaryBss(a, envLenOff)

	renvoAsmEmit32(a, 0x4ce7894c)
	renvoAsmEmit32(a, 0x894cc689)
	renvoAsmEmit8(a, 0xc2)
	renvoAsmPrimaryBssAddr(a, envOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmLoadPrimaryBss(a, envLenOff)
	renvoAsmCopyPrimaryToCallWord4(a)
	renvoAsmCopyPrimaryToCallWord5(a)
}

func renvoAsmImageAmd64(a *renvoAsm) []byte {
	trustNonNil(a)
	renvoAsmPatch(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if renvoCompilerStripSymbols {
		oldCodeLen := len(a.code)
		var out []byte
		out = a.code
		truncateBytes(&out, loadFileSize)
		for i := 0; i < oldCodeLen; i++ {
			src := oldCodeLen - 1 - i
			out[a.codeOffset+src] = out[src]
		}
		var header []byte
		header = renvoAppendElfHeaderAmd64(header, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		for i := 0; i < len(header); i++ {
			out[i] = header[i]
		}
		pos := a.codeOffset + oldCodeLen
		for i := 0; i < len(a.data); i++ {
			out[pos+i] = a.data[i]
		}
		return out
	}
	sec := renvoBuildElf64SymbolSections(a, 0x400000, a.codeOffset, loadFileSize)
	finalSize := sec.shoff + 448
	out := make([]byte, finalSize)
	truncateBytes(&out, 0)
	out = renvoAppendElfHeaderAmd64(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
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
	out = renvoAppendElf64SectionHeaders(out, &sec, a, 0x400000)
	return out
}

func renvoAppendElfHeaderAmd64(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	base := 0x400000
	header := "\x7f\x45\x4c\x46\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x38\x00\x01\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x07\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x40\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00"
	out = append(out, header...)
	out[start+56] = 2
	out[start+68] = 5
	renvoPut32At(out, start+24, base+entryOff)
	renvoPut32At(out, start+40, shoff)
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

func renvoAsmImageWindowsAmd64(a *renvoAsm) []byte {
	trustNonNil(a)
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
	renvoAsmPatchWindows(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = renvoAppendPEHeader64(out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
