package main

const rtgLinux386CodeOffset = 0x54
const rtgLinux386LoadAddress = 0x08048000

const rtgLinux386SysReadSeq = 3
const rtgLinux386SysWriteSeq = 4
const rtgLinux386SysOpen = 5
const rtgLinux386SysClose = 6
const rtgLinux386SysFchmod = 94
const rtgLinux386SysReadAt = 180
const rtgLinux386SysWriteAt = 181

func rtg386AsmPrepareReadWriteBuf(a *rtgAsm) {
	rtgAsmPushRcx(a)
	rtgAsmMovRsiRax(a)
	rtgAsmPopRdx(a)
}

func rtg386AsmMoveOffsetArg(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc689)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmEmit16(a, 0xc789)
}

func compileLinux386(input []int, output int) int {
	rtgSetTarget(rtgTargetLinux386)
	src := make([]byte, 0, 655360)
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
	result = rtgTryCompileScalarProgram386(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func compileWindows386(input []int, output int) int {
	rtgSetTarget(rtgTargetWindows386)
	var src []byte
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
	result = rtgTryCompileScalarProgram386(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func rtgTryCompileScalarProgram386(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
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
	a.codeOffset = rtgLinux386CodeOffset
	if rtgTargetIsWindows() {
		a.codeOffset = rtgWinSectionRVA
	}
	if rtgCompilerFixedTarget != 0 {
		g.funcLabels = make([]int, 0, 1536)
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := rtgAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	g.funcReachable = make([]bool, len(meta.funcs), len(meta.funcs))
	g.funcQueue = make([]int, 0, len(meta.funcs))
	rtgLinearMarkFunc(&g, appIndex)
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgs386(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	if rtgTargetIsWindows() {
		rtgAsmPushRax(a)
		rtgWin386CallImport(a, rtgWinImportExitProcess)
		rtgAsmRet(a)
	} else {
		rtgAsmMovRdiRax(a)
		rtgAsmMovRaxImm(a, 1)
		rtgAsmSyscall(a)
	}
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !rtgEmitScalarFunction(&g, i) {
			var result rtgCompileResult
			return result
		}
	}
	var data []byte
	if rtgTargetIsWindows() {
		data = rtgAsmImageWindows386(a)
	} else {
		data = rtgAsmImage386(a)
	}
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgs386(g *rtgLinearGen, appIndex int) bool {
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
		rtgAsmBuildWindowsArgvEnvSlices386(&g.asm, argsOff, argsTextOff, argsLenOff, envDataOff, envLenOff)
	} else {
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		rtgAsmBuildArgvEnvSlices386(&g.asm, argsOff, envDataOff, envLenOff)
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

func rtgAsmBuildWindowsArgvEnvSlices386(a *rtgAsm, bssOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	pathNameOff := len(a.data)
	a.data = append(a.data, 'P')
	a.data = append(a.data, 'A')
	a.data = append(a.data, 'T')
	a.data = append(a.data, 'H')
	a.data = append(a.data, '=')
	a.data = append(a.data, 0)
	skipLabel := rtgAsmNewLabel(a)
	startLabel := rtgAsmNewLabel(a)
	skipCharLabel := rtgAsmNewLabel(a)
	copyLabel := rtgAsmNewLabel(a)
	notQuoteLabel := rtgAsmNewLabel(a)
	copyCharLabel := rtgAsmNewLabel(a)
	argDoneLabel := rtgAsmNewLabel(a)
	finishLabel := rtgAsmNewLabel(a)

	rtgWin386CallImport(a, rtgWinImportGetCommandLineA)
	rtgAsmEmit16(a, 0xc689)
	rtgWin386MovEdiBssAddr(a, bssOff)
	rtgAsmMovRaxBssAddr(a, argsTextOff)
	rtgAsmEmit16(a, 0xc289)
	rtgAsmEmit16(a, 0xdb31)

	rtgAsmMarkLabel(a, skipLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, 0)
	rtgAsmJzLabel(a, finishLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, ' ')
	rtgAsmJzLabel(a, skipCharLabel)
	rtgAsmEmit3(a, 0x80, 0x3e, 9)
	rtgAsmJnzLabel(a, startLabel)
	rtgAsmMarkLabel(a, skipCharLabel)
	rtgAsmEmit8(a, 0x46)
	rtgAsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, startLabel)
	rtgAsmEmit16(a, 0x1789)
	rtgAsmEmit16(a, 0xc931)
	rtgAsmEmit16(a, 0xed31)
	rtgAsmMarkLabel(a, copyLabel)
	rtgAsmEmit16(a, 0x068a)
	rtgAsmEmit2(a, 0x3c, 0)
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmEmit2(a, 0x3c, '"')
	rtgAsmJnzLabel(a, notQuoteLabel)
	rtgAsmEmit3(a, 0x83, 0xf5, 1)
	rtgAsmEmit8(a, 0x46)
	rtgAsmJmpLabel(a, copyLabel)
	rtgAsmMarkLabel(a, notQuoteLabel)
	rtgAsmEmit3(a, 0x83, 0xfd, 0)
	rtgAsmJnzLabel(a, copyCharLabel)
	rtgAsmEmit2(a, 0x3c, ' ')
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmEmit2(a, 0x3c, 9)
	rtgAsmJzLabel(a, argDoneLabel)
	rtgAsmMarkLabel(a, copyCharLabel)
	rtgAsmEmit16(a, 0x0288)
	rtgAsmEmit8(a, 0x46)
	rtgAsmEmit8(a, 0x42)
	rtgAsmEmit8(a, 0x41)
	rtgAsmJmpLabel(a, copyLabel)

	rtgAsmMarkLabel(a, argDoneLabel)
	rtgAsmEmit3(a, 0xc6, 0x02, 0)
	rtgAsmEmit8(a, 0x42)
	rtgAsmEmit3(a, 0x89, 0x4f, 8)
	rtgAsmEmit3(a, 0x83, 0xc7, 16)
	rtgAsmEmit8(a, 0x43)
	rtgAsmEmit2(a, 0x3c, 0)
	rtgAsmJzLabel(a, finishLabel)
	rtgAsmEmit8(a, 0x46)
	rtgAsmJmpLabel(a, skipLabel)

	rtgAsmMarkLabel(a, finishLabel)
	rtgAsmEmit16(a, 0xd889)
	rtgAsmStoreRaxBss(a, argsLenOff)

	rtgWin386MovEdiBssAddr(a, envOff)
	rtgAsmMovRaxDataAddr(a, pathNameOff)
	rtgAsmEmit16(a, 0x0789)
	rtgAsmMovRaxImm(a, 5)
	rtgAsmEmit3(a, 0x89, 0x47, 8)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmStoreRaxBss(a, envLenOff)

	rtgWin386MovEbxBssAddr(a, bssOff)
	rtgWin386LoadEsiBss(a, argsLenOff)
	rtgAsmEmit16(a, 0xf289)
	rtgWin386MovEcxBssAddr(a, envOff)
	rtgWin386LoadEaxBss(a, envLenOff)
	rtgAsmEmit16(a, 0xc789)
}

func rtgAsmBuildArgvEnvSlices386(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)

	rtgAsmEmit24(a, 0x24048b)
	rtgAsmEmit16(a, 0xe689)
	rtgAsmEmit3(a, 0x83, 0xc6, 0x04)
	rtgAsmEmit8(a, 0xbf)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
	rtgAsmEmit16(a, 0xc931)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit16(a, 0xc139)
	rtgAsmEmit16(a, 0x8d0f)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, doneLabel)
	rtgAsmEmit24(a, 0x8e148b)
	rtgAsmEmit16(a, 0x1789)
	rtgAsmEmit16(a, 0xdb31)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAsmEmit4(a, 0x80, 0x3c, 0x1a, 0x00)
	rtgAsmJzLabel(a, afterLenLabel)
	rtgAsmEmit8(a, 0x43)
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAsmEmit3(a, 0x89, 0x5f, 0x08)
	rtgAsmEmit3(a, 0x83, 0xc7, 0x10)
	rtgAsmEmit8(a, 0x41)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgAsmEmit4(a, 0x8d, 0x74, 0x86, 0x08)
	rtgAsmEmit8(a, 0xbf)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, envOff, rtgAbsBssReloc)
	rtgAsmEmit16(a, 0xc931)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit24(a, 0x8e148b)
	rtgAsmEmit16(a, 0xd285)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit16(a, 0x1789)
	rtgAsmEmit16(a, 0xdb31)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAsmEmit4(a, 0x80, 0x3c, 0x1a, 0x00)
	rtgAsmJzLabel(a, envAfterLenLabel)
	rtgAsmEmit8(a, 0x43)
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgAsmEmit3(a, 0x89, 0x5f, 0x08)
	rtgAsmEmit3(a, 0x83, 0xc7, 0x10)
	rtgAsmEmit8(a, 0x41)
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAsmEmit16(a, 0x0d89)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, envLenOff, rtgAbsBssReloc)

	rtgAsmEmit8(a, 0xbb)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
	rtgAsmEmit16(a, 0xc689)
	rtgAsmEmit16(a, 0xc289)
	rtgAsmEmit8(a, 0xb9)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, envOff, rtgAbsBssReloc)
	rtgAsmEmit8(a, 0xa1)
	at = len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, envLenOff, rtgAbsBssReloc)
	rtgAsmEmit16(a, 0xc789)
}

func rtgAsmImage386(a *rtgAsm) []byte {
	rtgAsmPatch386(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := loadFileSize + a.bssSize
	if rtgCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = rtgAppendElfHeader386(out, a.codeOffset, loadFileSize, memSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		return out
	}
	sec := rtgBuildElf32SymbolSections(a, rtgLinux386LoadAddress, a.codeOffset, loadFileSize)
	out := make([]byte, 0, sec.shoff+280)
	out = rtgAppendElfHeader386(out, a.codeOffset, loadFileSize, memSize, sec.shoff)
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
	out = rtgAppendElf32SectionHeaders(out, &sec, a, rtgLinux386LoadAddress)
	return out
}

func rtgAsmPatch386(a *rtgAsm) {
	rtgAsmPatch(a)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		rtgPut32At(a.code, r.at, rtgLinux386LoadAddress+target)
	}
}

func rtgAppendElfHeader386(out []byte, entryOff int, fileSize int, memSize int, shoff int) []byte {
	base := rtgLinux386LoadAddress

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
	out = rtgAppend16(out, 2)
	out = rtgAppend16(out, 3)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, base+entryOff)
	out = rtgAppend32(out, 52)
	out = rtgAppend32(out, shoff)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 52)
	out = rtgAppend16(out, 32)
	out = rtgAppend16(out, 1)
	if shoff == 0 {
		out = rtgAppend16(out, 0)
		out = rtgAppend16(out, 0)
		out = rtgAppend16(out, 0)
	} else {
		out = rtgAppend16(out, 40)
		out = rtgAppend16(out, 7)
		out = rtgAppend16(out, 6)
	}

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, base)
	out = rtgAppend32(out, base)
	out = rtgAppend32(out, fileSize)
	out = rtgAppend32(out, memSize)
	out = rtgAppend32(out, 7)
	out = rtgAppend32(out, 0x1000)
	return out
}

func rtgAsmImageWindows386(a *rtgAsm) []byte {
	for (a.codeOffset+len(a.code))%4 != 0 {
		a.code = append(a.code, 0)
	}
	textVirtualSize := len(a.code)
	textRawSize := rtgAlignValue(textVirtualSize, rtgWinFileAlign)
	dataRVA := rtgAlignValue(a.codeOffset+textVirtualSize, rtgWinSectionAlign)
	a.dataOffset = dataRVA
	var imports rtgWinImportLayout
	if rtgAsmHasWinImportRelocs(a) {
		imports = rtgAppendWinImports(a, false)
	}
	rtgAsmPatchWindows(a, imports, rtgWinImageBase, false)
	dataRawSize := rtgAlignValue(len(a.data), rtgWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (rtgWinImportCount() + 1) * imports.thunkSize
	}
	var out []byte
	out = rtgAppendPEHeader32(out, a.codeOffset, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
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
