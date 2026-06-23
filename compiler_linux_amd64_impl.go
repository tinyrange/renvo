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
	rtgAsmMovRsiRax(a)
	rtgAsmEmit16(a, 0x5a51)
}

func rtgAmd64AsmMoveOffsetArg(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc28949)
}

func compileLinuxAmd64(input []int, output int) int {
	rtgSetTarget(rtgTargetLinuxAmd64)
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
	meta = rtgBuildMeta(&prog)
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
	for i := 0; i < len(meta.funcs); i++ {
		label := rtgAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
		src := meta.prog.src
		nameStart := meta.funcs[i].nameStart
		nameEnd := meta.funcs[i].nameEnd
		rtgAsmAddFuncSymbol(a, src, nameStart, nameEnd, label)
	}
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgsAmd64(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	rtgAsmMovRdiRax(a)
	rtgAsmMovRaxImm(a, 60)
	rtgAsmSyscall(a)
	for i := 0; i < len(meta.funcs); i++ {
		if !rtgEmitScalarFunction(&g, i) {
			rtgPrintErr("rtg: amd64 failed in function ")
			write(2, meta.prog.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
			rtgPrintErr("\n")
			var result rtgCompileResult
			return result
		}
	}
	data := rtgAsmImageAmd64(a)
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
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	rtgAsmBuildArgvEnvSlicesAmd64(&g.asm, argsOff, envDataOff, envLenOff)
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
	rtgAsmMovR10BssAddr(a, bssOff)
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
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAsmEmit32(a, 0x08428949)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgAsmEmit32(a, 0x244c8d4c)
	rtgAsmEmit8(a, 0x8)
	rtgAsmMarkLabel(a, envScanLabel)
	rtgAsmEmit32(a, 0x00398349)
	rtgAsmJzLabel(a, envStartLabel)
	rtgAsmEmit32(a, 0x08c18349)
	rtgAsmJmpLabel(a, envScanLabel)
	rtgAsmMarkLabel(a, envStartLabel)
	rtgAsmEmit32(a, 0x08c18349)
	rtgAsmMovR10BssAddr(a, envOff)
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
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgAsmEmit32(a, 0x08428949)
	rtgAsmEmit32(a, 0x10c28349)
	rtgAsmEmit24(a, 0xc3ff49)
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAsmEmit24(a, 0xd8894c)
	rtgAsmStoreRaxBss(a, envLenOff)

	rtgAsmEmit32(a, 0x4ce7894c)
	rtgAsmEmit32(a, 0x894cc689)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmMovRaxBssAddr(a, envOff)
	rtgAsmMovRcxRax(a)
	rtgAsmLoadRaxBss(a, envLenOff)
	rtgAsmMovR8Rax(a)
	rtgAsmMovR9Rax(a)
}

func rtgAsmImageAmd64(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := loadFileSize + a.bssSize
	sec := rtgBuildElf64SymbolSections(a, 0x400000, a.codeOffset, loadFileSize)
	var out []byte
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
	out = rtgAppend64(out, base+entryOff)
	out = rtgAppend64(out, 64)
	out = rtgAppend64(out, shoff)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 56)
	out = rtgAppend16(out, 1)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 7)
	out = rtgAppend16(out, 6)

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 7)
	out = rtgAppend64(out, 0)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend64(out, memSize)
	out = rtgAppend64(out, 0x1000)
	return out
}
