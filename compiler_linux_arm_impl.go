package main

const rtgLinuxArmCodeOffset = 0x54
const rtgLinuxArmLoadAddress = 0x00010000

const rtgLinuxArmSysReadSeq = 3
const rtgLinuxArmSysWriteSeq = 4
const rtgLinuxArmSysOpen = 5
const rtgLinuxArmSysClose = 6
const rtgLinuxArmSysFchmod = 94
const rtgLinuxArmSysReadAt = 180
const rtgLinuxArmSysWriteAt = 181

func rtgArmAsmPrepareReadWriteBuf(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRsi, rtgArmRegRax)
	rtgArmAsmMovRegReg(a, rtgArmRegRdx, rtgArmRegRcx)
}

func rtgArmAsmMoveOffsetArg(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegR10, rtgArmRegRax)
}

func compileLinuxArm(input []int, output int) int {
	rtgSetTarget(rtgTargetLinuxArm)
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
	result = rtgTryCompileScalarProgramArm(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: compilation failed\n")
	return 1
}

func rtgTryCompileScalarProgramArm(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
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
	a.codeOffset = rtgLinuxArmCodeOffset
	if rtgCompilerFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := rtgAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	rtgInitFuncQueue(&g, len(meta.funcs))
	rtgLinearMarkFunc(&g, appIndex)
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgsArm(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmPrimaryImm(a, 1)
	rtgAsmSyscall(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !rtgEmitScalarFunctionScratch(&g, i) {
			var result rtgCompileResult
			return result
		}
	}
	data := rtgAsmImageArm(a)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgsArm(g *rtgLinearGen, appIndex int) bool {
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
	rtgAsmBuildArgvEnvSlicesArm(&g.asm, argsOff, envDataOff, envLenOff)
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

func rtgAsmBuildArgvEnvSlicesArm(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)

	rtgArmAsmLoadRegMem(a, rtgArmRegR8, rtgArmRegSp, 0, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegR9, rtgArmRegSp, 4)
	rtgArmAsmMovRegAbs(a, rtgArmRegR10, bssOff, rtgAbsBssReloc)
	rtgArmAsmMovRegImm(a, rtgArmRegTmp2, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgArmAsmCmpRegReg(a, rtgArmRegTmp2, rtgArmRegR8)
	rtgArmAsmBCondLabel(a, doneLabel, 0)
	rtgArmAsmAddRegRegShift(a, rtgArmRegAddr, rtgArmRegR9, rtgArmRegTmp2, 2)
	rtgArmAsmLoadRegMem(a, rtgArmRegAddr, rtgArmRegAddr, 0, 4)
	rtgArmAsmStoreRegMem(a, rtgArmRegAddr, rtgArmRegR10, 0, 4)
	rtgArmAsmMovRegImm(a, rtgArmRegRax, 0)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgArmAsmAddRegReg(a, rtgArmRegTmp, rtgArmRegAddr, rtgArmRegRax)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegTmp, 0, 1)
	rtgArmAsmCmpRegImm(a, rtgArmRegTmp, 0)
	rtgArmAsmBCondLabel(a, afterLenLabel, 0)
	rtgArmAsmAddRegImm(a, rtgArmRegRax, rtgArmRegRax, 1)
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegR10, 8, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegR10, rtgArmRegR10, 16)
	rtgArmAsmAddRegImm(a, rtgArmRegTmp2, rtgArmRegTmp2, 1)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgArmAsmAddRegRegShift(a, rtgArmRegR9, rtgArmRegR9, rtgArmRegR8, 2)
	rtgArmAsmAddRegImm(a, rtgArmRegR9, rtgArmRegR9, 4)
	rtgArmAsmMovRegAbs(a, rtgArmRegR10, envOff, rtgAbsBssReloc)
	rtgArmAsmMovRegImm(a, rtgArmRegR9, 0)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp2, rtgArmRegSp, 0, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegTmp2, rtgArmRegTmp2, 2)
	rtgArmAsmAddRegRegShift(a, rtgArmRegTmp2, rtgArmRegSp, rtgArmRegTmp2, 2)
	rtgArmAsmMovRegImm(a, rtgArmRegR9, 0)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgArmAsmLoadRegMem(a, rtgArmRegAddr, rtgArmRegTmp2, 0, 4)
	rtgArmAsmCmpRegImm(a, rtgArmRegAddr, 0)
	rtgArmAsmBCondLabel(a, envDoneLabel, 0)
	rtgArmAsmStoreRegMem(a, rtgArmRegAddr, rtgArmRegR10, 0, 4)
	rtgArmAsmMovRegImm(a, rtgArmRegRax, 0)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgArmAsmAddRegReg(a, rtgArmRegTmp, rtgArmRegAddr, rtgArmRegRax)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegTmp, 0, 1)
	rtgArmAsmCmpRegImm(a, rtgArmRegTmp, 0)
	rtgArmAsmBCondLabel(a, envAfterLenLabel, 0)
	rtgArmAsmAddRegImm(a, rtgArmRegRax, rtgArmRegRax, 1)
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegR10, 8, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegR10, rtgArmRegR10, 16)
	rtgArmAsmAddRegImm(a, rtgArmRegTmp2, rtgArmRegTmp2, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegR9, rtgArmRegR9, 1)
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgArmAsmMovRegAbs(a, rtgArmRegAddr, envLenOff, rtgAbsBssReloc)
	rtgArmAsmStoreRegMem(a, rtgArmRegR9, rtgArmRegAddr, 0, 4)

	rtgArmAsmMovRegAbs(a, rtgArmRegRdi, bssOff, rtgAbsBssReloc)
	rtgArmAsmMovRegReg(a, rtgArmRegRsi, rtgArmRegR8)
	rtgArmAsmMovRegReg(a, rtgArmRegRdx, rtgArmRegR8)
	rtgArmAsmMovRegAbs(a, rtgArmRegRcx, envOff, rtgAbsBssReloc)
	rtgArmAsmMovRegReg(a, rtgArmRegR8, rtgArmRegR9)
	rtgArmAsmMovRegReg(a, rtgArmRegR9, rtgArmRegR9)
}

func rtgAsmImageArm(a *rtgAsm) []byte {
	rtgAsmPatchArm(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := loadFileSize + a.bssSize
	if rtgCompilerStripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = rtgAppendElfHeaderArm(out, a.codeOffset, loadFileSize, memSize, 0)
		for i := 0; i < len(a.code); i++ {
			out = append(out, a.code[i])
		}
		for i := 0; i < len(a.data); i++ {
			out = append(out, a.data[i])
		}
		return out
	}
	sec := rtgBuildElf32SymbolSections(a, rtgLinuxArmLoadAddress, a.codeOffset, loadFileSize)
	out := make([]byte, 0, sec.shoff+280)
	out = rtgAppendElfHeaderArm(out, a.codeOffset, loadFileSize, memSize, sec.shoff)
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
	out = rtgAppendElf32SectionHeaders(out, &sec, a, rtgLinuxArmLoadAddress)
	return out
}

func rtgAsmPatchArm(a *rtgAsm) {
	rtgAsmPatch(a)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		insn := rtgGet32At(a.code, r.at)
		reg := (insn >> 12) & 15
		rtgArmAsmPatchMovRegImmAt(a, r.at, reg, rtgLinuxArmLoadAddress+target)
	}
}

func rtgAppendElfHeaderArm(out []byte, entryOff int, fileSize int, memSize int, shoff int) []byte {
	base := rtgLinuxArmLoadAddress

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
	out = rtgAppend16(out, 40)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, base+entryOff)
	out = rtgAppend32(out, 52)
	out = rtgAppend32(out, shoff)
	out = rtgAppend32(out, 0x05000000)
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
