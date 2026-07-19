package main

const renvoLinux386CodeOffset = 0x74
const renvoLinux386LoadAddress = 0x08048000

const renvoLinux386SysReadSeq = 3
const renvoLinux386SysWriteSeq = 4
const renvoLinux386SysOpen = 5
const renvoLinux386SysClose = 6
const renvoLinux386SysFchmod = 94
const renvoLinux386SysReadAt = 180
const renvoLinux386SysWriteAt = 181

func renvo386AsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoAsmPushTertiary(a)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmPopSecondary(a)
}

func renvo386AsmMoveOffsetArg(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc689)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit16(a, 0xc789)
}

func compileLinux386(input []int, output int) int {
	return compileLinux386Arena(input, output, 0)
}

func compileLinux386Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinux386)
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
	result = renvoTryCompileScalarProgram386(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func compileWindows386(input []int, output int) int {
	return compileWindows386Arena(input, output, 0)
}

func compileWindows386Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindows386)
	var src []byte
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
	result = renvoTryCompileScalarProgram386(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgram386(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
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
	a.codeOffset = renvoLinux386CodeOffset
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
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) {
		var result renvoCompileResult
		return result
	}
	if !renvoEmitProgramEntryArgs386(&g, appIndex) {
		var result renvoCompileResult
		return result
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		var result renvoCompileResult
		return result
	}
	if targetIsWindows() {
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportExitProcess)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
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
		data = renvoAsmImageWindows386(a)
	} else {
		data = renvoAsmImage386(a)
	}
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoEmitProgramEntryArgs386(g *renvoLinearGen, appIndex int) bool {
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
		renvoAsmBuildWindowsArgvEnvSlices386(&g.asm, argsOff, argsTextOff, argsLenOff, envDataOff, envLenOff)
	} else {
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		renvoAsmBuildArgvEnvSlices386(&g.asm, argsOff, envDataOff, envLenOff)
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

func renvoAsmBuildWindowsArgvEnvSlices386(a *renvoAsm, bssOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	skipLabel := renvoAsmNewLabel(a)
	startLabel := renvoAsmNewLabel(a)
	skipCharLabel := renvoAsmNewLabel(a)
	copyLabel := renvoAsmNewLabel(a)
	notQuoteLabel := renvoAsmNewLabel(a)
	copyCharLabel := renvoAsmNewLabel(a)
	argDoneLabel := renvoAsmNewLabel(a)
	finishLabel := renvoAsmNewLabel(a)

	renvoWin386CallImport(a, renvoWinImportGetCommandLineA)
	renvoAsmEmit16(a, 0xc689)
	renvoWin386MovEdiBssAddr(a, bssOff)
	renvoAsmPrimaryBssAddr(a, argsTextOff)
	renvoAsmEmit16(a, 0xc289)
	renvoAsmEmit16(a, 0xdb31)

	renvoAsmMarkLabel(a, skipLabel)
	renvoAsmEmit3(a, 0x80, 0x3e, 0)
	renvoAsmJzLabel(a, finishLabel)
	renvoAsmEmit3(a, 0x80, 0x3e, ' ')
	renvoAsmJzLabel(a, skipCharLabel)
	renvoAsmEmit3(a, 0x80, 0x3e, 9)
	renvoAsmJnzLabel(a, startLabel)
	renvoAsmMarkLabel(a, skipCharLabel)
	renvoAsmEmit8(a, 0x46)
	renvoAsmJmpLabel(a, skipLabel)

	renvoAsmMarkLabel(a, startLabel)
	renvoAsmEmit16(a, 0x1789)
	renvoAsmEmit16(a, 0xc931)
	renvoAsmEmit16(a, 0xed31)
	renvoAsmMarkLabel(a, copyLabel)
	renvoAsmEmit16(a, 0x068a)
	renvoAsmEmit2(a, 0x3c, 0)
	renvoAsmJzLabel(a, argDoneLabel)
	renvoAsmEmit2(a, 0x3c, '"')
	renvoAsmJnzLabel(a, notQuoteLabel)
	renvoAsmEmit3(a, 0x83, 0xf5, 1)
	renvoAsmEmit8(a, 0x46)
	renvoAsmJmpMarkLabel(a, copyLabel, notQuoteLabel)
	renvoAsmEmit3(a, 0x83, 0xfd, 0)
	renvoAsmJnzLabel(a, copyCharLabel)
	renvoAsmEmit2(a, 0x3c, ' ')
	renvoAsmJzLabel(a, argDoneLabel)
	renvoAsmEmit2(a, 0x3c, 9)
	renvoAsmJzLabel(a, argDoneLabel)
	renvoAsmMarkLabel(a, copyCharLabel)
	renvoAsmEmit16(a, 0x0288)
	renvoAsmEmit8(a, 0x46)
	renvoAsmEmit8(a, 0x42)
	renvoAsmEmit8(a, 0x41)
	renvoAsmJmpLabel(a, copyLabel)

	renvoAsmMarkLabel(a, argDoneLabel)
	renvoAsmEmit3(a, 0xc6, 0x02, 0)
	renvoAsmEmit8(a, 0x42)
	renvoAsmEmit3(a, 0x89, 0x4f, 8)
	renvoAsmEmit3(a, 0x83, 0xc7, 16)
	renvoAsmEmit8(a, 0x43)
	renvoAsmEmit2(a, 0x3c, 0)
	renvoAsmJzLabel(a, finishLabel)
	renvoAsmEmit8(a, 0x46)
	renvoAsmJmpLabel(a, skipLabel)

	renvoAsmMarkLabel(a, finishLabel)
	renvoAsmEmit16(a, 0xd889)
	renvoAsmStorePrimaryBss(a, argsLenOff)

	envLoopLabel := renvoAsmNewLabel(a)
	envStringLabel := renvoAsmNewLabel(a)
	envStringDoneLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)
	renvoWin386CallImport(a, renvoWinImportGetEnvironmentStringsA)
	renvoAsmEmit16(a, 0xc689)
	renvoWin386MovEdiBssAddr(a, envOff)
	renvoAsmEmit16(a, 0xdb31)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAsmEmit3(a, 0x80, 0x3e, 0)
	renvoAsmJzLabel(a, envDoneLabel)
	renvoAsmEmit16(a, 0x3789)
	renvoAsmEmit16(a, 0xc931)
	renvoAsmMarkLabel(a, envStringLabel)
	renvoAsmEmit4(a, 0x80, 0x3c, 0x0e, 0)
	renvoAsmJzLabel(a, envStringDoneLabel)
	renvoAsmEmit8(a, 0x41)
	renvoAsmJmpMarkLabel(a, envStringLabel, envStringDoneLabel)
	renvoAsmEmit3(a, 0x89, 0x4f, 8)
	renvoAsmEmit16(a, 0xce01)
	renvoAsmEmit8(a, 0x46)
	renvoAsmEmit3(a, 0x83, 0xc7, 16)
	renvoAsmEmit8(a, 0x43)
	renvoAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	renvoAsmEmit16(a, 0xd889)
	renvoAsmStorePrimaryBss(a, envLenOff)

	renvoWin386MovEbxBssAddr(a, bssOff)
	renvoWin386LoadEsiBss(a, argsLenOff)
	renvoAsmEmit16(a, 0xf289)
	renvoWin386MovEcxBssAddr(a, envOff)
	renvoWin386LoadEaxBss(a, envLenOff)
	renvoAsmEmit16(a, 0xc789)
}

func renvoAsmBuildArgvEnvSlices386(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := renvoAsmNewLabel(a)
	strlenLabel := renvoAsmNewLabel(a)
	afterLenLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	envLoopLabel := renvoAsmNewLabel(a)
	envStrlenLabel := renvoAsmNewLabel(a)
	envAfterLenLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)

	renvoAsmEmit24(a, 0x24048b)
	renvoAsmEmit16(a, 0xe689)
	renvoAsmEmit3(a, 0x83, 0xc6, 0x04)
	renvoAsmEmit8(a, 0xbf)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	renvoAsmEmit16(a, 0xc931)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmit16(a, 0xc139)
	renvoAsmEmit16(a, 0x8d0f)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, doneLabel)
	renvoAsmEmit24(a, 0x8e148b)
	renvoAsmEmit16(a, 0x1789)
	renvoAsmEmit16(a, 0xdb31)
	renvoAsmMarkLabel(a, strlenLabel)
	renvoAsmEmit4(a, 0x80, 0x3c, 0x1a, 0x00)
	renvoAsmJzLabel(a, afterLenLabel)
	renvoAsmEmit8(a, 0x43)
	renvoAsmJmpMarkLabel(a, strlenLabel, afterLenLabel)
	renvoAsmEmit3(a, 0x89, 0x5f, 0x08)
	renvoAsmEmit3(a, 0x83, 0xc7, 0x10)
	renvoAsmEmit8(a, 0x41)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)

	renvoAsmEmit4(a, 0x8d, 0x74, 0x86, 0x08)
	renvoAsmEmit8(a, 0xbf)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envOff, renvoAbsBssReloc)
	renvoAsmEmit16(a, 0xc931)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAsmEmit24(a, 0x8e148b)
	renvoAsmEmit16(a, 0xd285)
	renvoAsmJzLabel(a, envDoneLabel)
	renvoAsmEmit16(a, 0x1789)
	renvoAsmEmit16(a, 0xdb31)
	renvoAsmMarkLabel(a, envStrlenLabel)
	renvoAsmEmit4(a, 0x80, 0x3c, 0x1a, 0x00)
	renvoAsmJzLabel(a, envAfterLenLabel)
	renvoAsmEmit8(a, 0x43)
	renvoAsmJmpMarkLabel(a, envStrlenLabel, envAfterLenLabel)
	renvoAsmEmit3(a, 0x89, 0x5f, 0x08)
	renvoAsmEmit3(a, 0x83, 0xc7, 0x10)
	renvoAsmEmit8(a, 0x41)
	renvoAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	renvoAsmEmit16(a, 0x0d89)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envLenOff, renvoAbsBssReloc)

	renvoAsmEmit8(a, 0xbb)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	renvoAsmEmit16(a, 0xc689)
	renvoAsmEmit16(a, 0xc289)
	renvoAsmEmit8(a, 0xb9)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envOff, renvoAbsBssReloc)
	renvoAsmEmit8(a, 0xa1)
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envLenOff, renvoAbsBssReloc)
	renvoAsmEmit16(a, 0xc789)
}

func renvoAsmImage386(a *renvoAsm) []byte {
	renvoAsmPatch386(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if renvoCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeader386(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		return out
	}
	sec := renvoBuildElf32SymbolSections(a, renvoLinux386LoadAddress, a.codeOffset, loadFileSize)
	out := make([]byte, 0, sec.shoff+280)
	out = renvoAppendElfHeader386(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
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
	out = renvoAppendElf32SectionHeaders(out, &sec, a, renvoLinux386LoadAddress)
	return out
}

func renvoAsmPatch386(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + r.off
		}
		renvoPut32At(a.code, r.at, renvoLinux386LoadAddress+target)
	}
}

func renvoAppendElfHeader386(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := renvoLinux386LoadAddress

	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 0)
	for i := 0; i < 8; i++ {
		out = append(out, 0)
	}
	out = renvoAppend16(out, 2)
	out = renvoAppend16(out, 3)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, base+entryOff)
	out = renvoAppend32(out, 52)
	out = renvoAppend32(out, shoff)
	out = renvoAppend32(out, 0)
	out = renvoAppend16(out, 52)
	out = renvoAppend16(out, 32)
	out = renvoAppend16(out, 2)
	if shoff == 0 {
		out = renvoAppend16(out, 0)
		out = renvoAppend16(out, 0)
		out = renvoAppend16(out, 0)
	} else {
		out = renvoAppend16(out, 40)
		out = renvoAppend16(out, 7)
		out = renvoAppend16(out, 6)
	}

	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, base)
	out = renvoAppend32(out, base)
	out = renvoAppend32(out, fileSize)
	out = renvoAppend32(out, fileSize)
	out = renvoAppend32(out, 5)
	out = renvoAppend32(out, 0x1000)
	out = renvoAppendElf32LoadProgram(out, 6, bssOffset, base+bssOffset, 0, bssSize)
	return out
}

func renvoAsmImageWindows386(a *renvoAsm) []byte {
	for (a.codeOffset+len(a.code))%4 != 0 {
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
	out = renvoAppendPEHeader32(out, a.codeOffset, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
