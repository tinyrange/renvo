package main

const renvoLinuxArmCodeOffset = 0x74

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
	base := len(a.code)
	renvoAsmEmitText(a, "\x00\x50\x9d\xe5\x04\x90\x00\xe3\x09\x60\x8d\xe0\x00\x80\x00\xe3\x00\x80\x40\xe3\x08\x80\x8f\xe0\x00\xa0\x00\xe3\x05\x00\x5a\xe1\x10\x00\x00\x0a\x0a\xc1\x86\xe0\x00\xc0\x9c\xe5\x00\xc0\x88\xe5\x00\x00\x00\xe3\x00\x90\x8c\xe0\x00\x90\xd9\xe5\x00\x00\x59\xe3\x02\x00\x00\x0a\x01\x90\x00\xe3\x09\x00\x80\xe0\xf8\xff\xff\xea\x08\x00\x88\xe5\x10\x90\x00\xe3\x09\x80\x88\xe0\x01\x90\x00\xe3\x09\xa0\x8a\xe0\xec\xff\xff\xea\x05\x61\x86\xe0\x04\x90\x00\xe3\x09\x60\x86\xe0\x00\x80\x00\xe3\x00\x80\x40\xe3\x08\x80\x8f\xe0\x00\x60\x00\xe3\x00\xa0\x9d\xe5\x02\x90\x00\xe3\x09\xa0\x8a\xe0\x0a\xa1\x8d\xe0\x00\x60\x00\xe3\x00\xc0\x9a\xe5\x00\x00\x5c\xe3\x10\x00\x00\x0a\x00\xc0\x88\xe5\x00\x00\x00\xe3\x00\x90\x8c\xe0\x00\x90\xd9\xe5\x00\x00\x59\xe3\x02\x00\x00\x0a\x01\x90\x00\xe3\x09\x00\x80\xe0\xf8\xff\xff\xea\x08\x00\x88\xe5\x10\x90\x00\xe3\x09\x80\x88\xe0\x04\x90\x00\xe3\x09\xa0\x8a\xe0\x01\x90\x00\xe3\x09\x60\x86\xe0\xeb\xff\xff\xea\x00\xc0\x00\xe3\x00\xc0\x40\xe3\x0c\xc0\x8f\xe0\x00\x60\x8c\xe5\x00\x30\x00\xe3\x00\x30\x40\xe3\x03\x30\x8f\xe0\x05\x40\xa0\xe1\x05\x10\xa0\xe1\x00\x20\x00\xe3\x00\x20\x40\xe3\x02\x20\x8f\xe0\x06\x50\xa0\xe1")
	renvoAsmAddAbsReloc(a, base+12, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+116, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+232, envLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+248, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+268, envOff, renvoAbsBssReloc)
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
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
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
	out = renvoAppendElfSectionHeaders(out, &sec, a, 0)
	return out
}

func renvoAsmPatchArm(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		insn := renvoGet32At(a.code, at)
		reg := (insn >> 12) & 15
		delta := target - (a.codeOffset + at + 16)
		renvoArmAsmPatchMovRegImmAt(a, at, reg, delta)
	}
}

func renvoAppendElfHeaderArm(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	base := 0
	// The ELF and program-header layouts are fixed. Keep their invariant bytes
	// together and patch the seven fields that vary per output image.
	out = append(out, "\x7f\x45\x4c\x46\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x28\x00\x01\x00\x00\x00\x00\x00\x01\x00\x34\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x34\x00\x20\x00\x02\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x10\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x06\x00\x00\x00\x00\x10\x00\x00"...)
	renvoPut32At(out, start+24, base+entryOff)
	renvoPut32At(out, start+32, shoff)
	if shoff != 0 {
		out[start+46] = 40
		out[start+48] = 7
		out[start+50] = 6
	}
	renvoPut32At(out, start+60, base)
	renvoPut32At(out, start+64, base)
	renvoPut32At(out, start+68, fileSize)
	renvoPut32At(out, start+72, fileSize)
	renvoPut32At(out, start+88, bssOffset)
	renvoPut32At(out, start+92, base+bssOffset)
	renvoPut32At(out, start+96, base+bssOffset)
	renvoPut32At(out, start+104, bssSize)
	return out
}
