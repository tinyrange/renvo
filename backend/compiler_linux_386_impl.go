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
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
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
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
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
	if renvoFixedTarget != 0 {
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
	base := len(a.code)
	renvoAsmEmitText(a, "\xff\x15\x00\x00\x00\x00\x89\xc6\xbf\x00\x00\x00\x00\xb8\x00\x00\x00\x00\x89\xc2\x31\xdb\x80\x3e\x00\x0f\x84\x75\x00\x00\x00\x80\x3e\x20\x0f\x84\x09\x00\x00\x00\x80\x3e\x09\x0f\x85\x06\x00\x00\x00\x46\xe9\xdf\xff\xff\xff\x89\x17\x31\xc9\x31\xed\x8a\x06\x3c\x00\x0f\x84\x34\x00\x00\x00\x3c\x22\x0f\x85\x09\x00\x00\x00\x83\xf5\x01\x46\xe9\xe5\xff\xff\xff\x83\xfd\x00\x0f\x85\x10\x00\x00\x00\x3c\x20\x0f\x84\x12\x00\x00\x00\x3c\x09\x0f\x84\x0a\x00\x00\x00\x88\x02\x46\x42\x41\xe9\xc2\xff\xff\xff\xc6\x02\x00\x42\x89\x4f\x08\x83\xc7\x10\x43\x3c\x00\x0f\x84\x06\x00\x00\x00\x46\xe9\x82\xff\xff\xff\x89\xd8\xa3\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x89\xc6\xbf\x00\x00\x00\x00\x31\xdb\x80\x3e\x00\x0f\x84\x23\x00\x00\x00\x89\x37\x31\xc9\x80\x3c\x0e\x00\x0f\x84\x06\x00\x00\x00\x41\xe9\xf0\xff\xff\xff\x89\x4f\x08\x01\xce\x46\x83\xc7\x10\x43\xe9\xd4\xff\xff\xff\x89\xd8\xa3\x00\x00\x00\x00\xbb\x00\x00\x00\x00\x8b\x35\x00\x00\x00\x00\x89\xf2\xb9\x00\x00\x00\x00\xa1\x00\x00\x00\x00\x89\xc7")
	renvoAsmAddWinImportReloc(a, base+2, renvoWinImportGetCommandLineA)
	renvoAsmAddAbsReloc(a, base+9, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+14, argsTextOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+151, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, base+157, renvoWinImportGetEnvironmentStringsA)
	renvoAsmAddAbsReloc(a, base+164, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+217, envLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+222, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+228, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+235, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+240, envLenOff, renvoAbsBssReloc)
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

	// Stack walking is invariant; keep fixed instruction runs together and
	// leave relocations and branch labels explicit below.
	renvoAsmEmitText(a, "\x8b\x04\x24\x89\xe6\x83\xc6\x04\xbf")
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	renvoAsmEmitText(a, "\x31\xc9")
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmitText(a, "\x39\xc1\x0f\x8d")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, doneLabel)
	renvoAsmEmitText(a, "\x8b\x14\x8e\x89\x17\x31\xdb")
	renvoAsmMarkLabel(a, strlenLabel)
	renvoAsmEmitText(a, "\x80\x3c\x1a\x00")
	renvoAsmJzLabel(a, afterLenLabel)
	renvoAsmEmitText(a, "\x43")
	renvoAsmJmpMarkLabel(a, strlenLabel, afterLenLabel)
	renvoAsmEmitText(a, "\x89\x5f\x08\x83\xc7\x10\x41")
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)

	renvoAsmEmitText(a, "\x8d\x74\x86\x08\xbf")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envOff, renvoAbsBssReloc)
	renvoAsmEmitText(a, "\x31\xc9")
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoAsmEmitText(a, "\x8b\x14\x8e\x85\xd2")
	renvoAsmJzLabel(a, envDoneLabel)
	renvoAsmEmitText(a, "\x89\x17\x31\xdb")
	renvoAsmMarkLabel(a, envStrlenLabel)
	renvoAsmEmitText(a, "\x80\x3c\x1a\x00")
	renvoAsmJzLabel(a, envAfterLenLabel)
	renvoAsmEmitText(a, "\x43")
	renvoAsmJmpMarkLabel(a, envStrlenLabel, envAfterLenLabel)
	renvoAsmEmitText(a, "\x89\x5f\x08\x83\xc7\x10\x41")
	renvoAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	renvoAsmEmitText(a, "\x89\x0d")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envLenOff, renvoAbsBssReloc)

	renvoAsmEmitText(a, "\xbb")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	renvoAsmEmitText(a, "\x89\xc6\x89\xc2\xb9")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envOff, renvoAbsBssReloc)
	renvoAsmEmitText(a, "\xa1")
	at = len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, envLenOff, renvoAbsBssReloc)
	renvoAsmEmitText(a, "\x89\xc7")
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
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, renvoLinux386LoadAddress, a.codeOffset, loadFileSize, &sec)
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
	out = renvoAppendElfSectionHeaders(out, &sec, a, renvoLinux386LoadAddress)
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
		renvoAppendWinImports(a, &imports)
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
