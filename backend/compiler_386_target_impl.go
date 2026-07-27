package main

const renvo386ELFCodeOffset = 0x74

func renvoTryCompileScalarProgram386(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	return renvoTryCompileScalarProgram386Scratch(p, meta)
}

func renvoTryCompileScalarProgram386Scratch(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram386(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsScratch(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram386(g)
}

func renvoTryCompileScalarProgram386Cached(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram386(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsCached(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram386(g)
}
func renvoBeginScalarProgram386(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		return nil
	}
	g := new(renvoLinearGen)
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	a.codeOffset = renvo386ELFCodeOffset
	if targetIsWindows(meta.c.renvoTargetOS) {
		a.codeOffset = renvoWinSectionRVA
	}
	if renvoFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(g, len(meta.funcs))
	renvoLinearMarkFunc(g, appIndex)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return nil
	}
	entryOK := false
	if renvoFixedTarget == 0 && meta.c.emitImage {
		entryOK = renvoEmitImageEntryArgs386(g, appIndex)
	} else {
		entryOK = renvoEmitProgramEntryArgs386(g, appIndex)
	}
	if !entryOK {
		return nil
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return nil
	}
	if renvoFixedTarget == 0 && meta.c.emitImage {
		renvoAsmRet(a)
	} else if targetIsWindows(meta.c.renvoTargetOS) {
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportExitProcess)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmSyscall(a)
	}
	return g
}

func renvoEmitImageEntryArgs386(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) {
		return false
	}
	// cdecl entry stack -> EBX/ESI/EDX, the first Renvo slice.
	renvoAsmEmitText(&g.asm, "\x8b\x5c\x24\x04\x8b\x74\x24\x08\x89\xf2")
	if app.paramCount == 1 {
		return true
	}
	if !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) {
		return false
	}
	// ECX/EAX/EDI, the second Renvo slice.
	renvoAsmEmitText(&g.asm, "\x8b\x4c\x24\x0c\x8b\x44\x24\x10\x89\xc7")
	return true
}

func renvoFinishScalarProgram386(g *renvoLinearGen) renvoCompileResult {
	renvoNonNil(g)
	a := &g.asm
	var data []byte
	if targetIsWindows(g.c.renvoTargetOS) {
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
	if targetIsWindows(g.c.renvoTargetOS) {
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
func renvo386AsmMovRaxDataAddr(a *renvoAsm, dataOff int) {
	if a.c.renvoTargetOS == renvoOSLinux {
		renvo386AsmMovRegPCRel(a, 0, dataOff, 0)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, dataOff, 0)
}

func renvo386AsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux {
		renvo386AsmMovRegPCRel(a, 0, bssOff, renvoAbsBssReloc)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux {
		renvo386AsmMovRegPCRel(a, 3, bssOff, renvoAbsBssReloc)
		return
	}
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux {
		renvo386AsmMovRegPCRel(a, 0, bssOff, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x008b)
		return
	}
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux {
		renvoAsmEmit8(a, 0x53)
		renvo386AsmMovRegPCRel(a, 3, bssOff, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x0389)
		renvoAsmEmit8(a, 0x5b)
		return
	}
	renvoAsmEmit8(a, 0xa3)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386EmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoAsmEmit16(a, 0x5f52)
		renvoAsmEmit16(a, 0x728d)
		renvoAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		if g.c.renvoTargetOS == renvoOSLinux {
			renvo386AsmMovRegPCRel(a, 7, loc.offset, renvoAbsBssReloc)
			renvo386AsmMovRegPCRel(a, 6, loc.offset+8, renvoAbsBssReloc)
			return true
		}
		renvoAsmEmit16(a, 0x3d8d)
		at := len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x358d)
		at = len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset+8, renvoAbsBssReloc)
		return true
	}
	renvoAsmAddressCallWord0Stack(a, loc.offset)
	renvoAsmAddressCallWord1Stack(a, loc.offset-8)
	return true
}
