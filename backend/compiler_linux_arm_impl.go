package main

const renvoLinuxArmCodeOffset = 0x74
const renvoLinuxArmLoadAddress = 0x00010000

const renvoLinuxArmSysReadSeq = 3
const renvoLinuxArmSysWriteSeq = 4
const renvoLinuxArmSysOpen = 5
const renvoLinuxArmSysClose = 6
const renvoLinuxArmSysFchmod = 94
const renvoLinuxArmSysReadAt = 180
const renvoLinuxArmSysWriteAt = 181

func renvoArmAsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRsi, renvoArmRegRax)
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegRcx)
}

func renvoArmAsmMoveOffsetArg(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegR10, renvoArmRegRax)
}

func compileLinuxArm(input []int, output int) int {
	return compileLinuxArmArena(input, output, 0)
}

func compileLinuxArmArena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxArm)
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
	result = renvoTryCompileScalarProgramArm(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramArm(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
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
	a.codeOffset = renvoLinuxArmCodeOffset
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
	if !renvoEmitProgramEntryArgsArm(&g, appIndex) {
		var result renvoCompileResult
		return result
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		var result renvoCompileResult
		return result
	}
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmSyscall(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !renvoEmitScalarFunctionScratch(&g, i) {
			var result renvoCompileResult
			return result
		}
	}
	data := renvoAsmImageArm(a)
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoEmitProgramEntryArgsArm(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	renvoAsmBuildArgvEnvSlicesArm(&g.asm, argsOff, envDataOff, envLenOff)
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
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !renvoTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func renvoAsmBuildArgvEnvSlicesArm(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := renvoAsmNewLabel(a)
	strlenLabel := renvoAsmNewLabel(a)
	afterLenLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	envLoopLabel := renvoAsmNewLabel(a)
	envStrlenLabel := renvoAsmNewLabel(a)
	envAfterLenLabel := renvoAsmNewLabel(a)
	envDoneLabel := renvoAsmNewLabel(a)

	renvoArmAsmLoadRegMem(a, renvoArmRegR8, renvoArmRegSp, 0, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegR9, renvoArmRegSp, 4)
	renvoArmAsmMovRegAbs(a, renvoArmRegR10, bssOff, renvoAbsBssReloc)
	renvoArmAsmMovRegImm(a, renvoArmRegTmp2, 0)
	renvoAsmMarkLabel(a, loopLabel)
	renvoArmAsmCmpRegReg(a, renvoArmRegTmp2, renvoArmRegR8)
	renvoArmAsmBCondLabel(a, doneLabel, 0)
	renvoArmAsmAddRegRegShift(a, renvoArmRegAddr, renvoArmRegR9, renvoArmRegTmp2, 2)
	renvoArmAsmLoadRegMem(a, renvoArmRegAddr, renvoArmRegAddr, 0, 4)
	renvoArmAsmStoreRegMem(a, renvoArmRegAddr, renvoArmRegR10, 0, 4)
	renvoArmAsmMovRegImm(a, renvoArmRegRax, 0)
	renvoAsmMarkLabel(a, strlenLabel)
	renvoArmAsmAddRegReg(a, renvoArmRegTmp, renvoArmRegAddr, renvoArmRegRax)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegTmp, 0, 1)
	renvoArmAsmCmpRegImm(a, renvoArmRegTmp, 0)
	renvoArmAsmBCondLabel(a, afterLenLabel, 0)
	renvoArmAsmAddRegImm(a, renvoArmRegRax, renvoArmRegRax, 1)
	renvoAsmJmpMarkLabel(a, strlenLabel, afterLenLabel)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegR10, 8, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegR10, renvoArmRegR10, 16)
	renvoArmAsmAddRegImm(a, renvoArmRegTmp2, renvoArmRegTmp2, 1)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)

	renvoArmAsmAddRegRegShift(a, renvoArmRegR9, renvoArmRegR9, renvoArmRegR8, 2)
	renvoArmAsmAddRegImm(a, renvoArmRegR9, renvoArmRegR9, 4)
	renvoArmAsmMovRegAbs(a, renvoArmRegR10, envOff, renvoAbsBssReloc)
	renvoArmAsmMovRegImm(a, renvoArmRegR9, 0)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp2, renvoArmRegSp, 0, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegTmp2, renvoArmRegTmp2, 2)
	renvoArmAsmAddRegRegShift(a, renvoArmRegTmp2, renvoArmRegSp, renvoArmRegTmp2, 2)
	renvoArmAsmMovRegImm(a, renvoArmRegR9, 0)
	renvoAsmMarkLabel(a, envLoopLabel)
	renvoArmAsmLoadRegMem(a, renvoArmRegAddr, renvoArmRegTmp2, 0, 4)
	renvoArmAsmCmpRegImm(a, renvoArmRegAddr, 0)
	renvoArmAsmBCondLabel(a, envDoneLabel, 0)
	renvoArmAsmStoreRegMem(a, renvoArmRegAddr, renvoArmRegR10, 0, 4)
	renvoArmAsmMovRegImm(a, renvoArmRegRax, 0)
	renvoAsmMarkLabel(a, envStrlenLabel)
	renvoArmAsmAddRegReg(a, renvoArmRegTmp, renvoArmRegAddr, renvoArmRegRax)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegTmp, 0, 1)
	renvoArmAsmCmpRegImm(a, renvoArmRegTmp, 0)
	renvoArmAsmBCondLabel(a, envAfterLenLabel, 0)
	renvoArmAsmAddRegImm(a, renvoArmRegRax, renvoArmRegRax, 1)
	renvoAsmJmpMarkLabel(a, envStrlenLabel, envAfterLenLabel)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegR10, 8, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegR10, renvoArmRegR10, 16)
	renvoArmAsmAddRegImm(a, renvoArmRegTmp2, renvoArmRegTmp2, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegR9, renvoArmRegR9, 1)
	renvoAsmJmpMarkLabel(a, envLoopLabel, envDoneLabel)
	renvoArmAsmMovRegAbs(a, renvoArmRegAddr, envLenOff, renvoAbsBssReloc)
	renvoArmAsmStoreRegMem(a, renvoArmRegR9, renvoArmRegAddr, 0, 4)

	renvoArmAsmMovRegAbs(a, renvoArmRegRdi, bssOff, renvoAbsBssReloc)
	renvoArmAsmMovRegReg(a, renvoArmRegRsi, renvoArmRegR8)
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegR8)
	renvoArmAsmMovRegAbs(a, renvoArmRegRcx, envOff, renvoAbsBssReloc)
	renvoArmAsmMovRegReg(a, renvoArmRegR8, renvoArmRegR9)
	renvoArmAsmMovRegReg(a, renvoArmRegR9, renvoArmRegR9)
}

func renvoAsmImageArm(a *renvoAsm) []byte {
	renvoAsmPatchArm(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if renvoCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		return out
	}
	sec := renvoBuildElf32SymbolSections(a, renvoLinuxArmLoadAddress, a.codeOffset, loadFileSize)
	out := make([]byte, 0, sec.shoff+280)
	out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
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
	out = renvoAppendElf32SectionHeaders(out, &sec, a, renvoLinuxArmLoadAddress)
	return out
}

func renvoAsmPatchArm(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + r.off
		}
		insn := renvoGet32At(a.code, r.at)
		reg := (insn >> 12) & 15
		renvoArmAsmPatchMovRegImmAt(a, r.at, reg, renvoLinuxArmLoadAddress+target)
	}
}

func renvoAppendElfHeaderArm(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	base := renvoLinuxArmLoadAddress

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
	out = renvoAppend16(out, 40)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, base+entryOff)
	out = renvoAppend32(out, 52)
	out = renvoAppend32(out, shoff)
	out = renvoAppend32(out, 0x05000000)
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
