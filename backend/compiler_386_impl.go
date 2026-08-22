package main

const renvo386ELFCodeOffset = 0x74

func renvoAsmImageObject386(emitter *renvoAsm) []byte {
	return renvoAsmImageRelocatableObject386(emitter)
}

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
	if renvoFixedTarget == 0 {
		if renvoIsHostedObject386(meta.c) {
			return renvoBeginObjectProgram(p, meta)
		}
	}
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
	renvoEmitInitializeThreadState(g)
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
	if renvoFixedTarget == 0 && renvoIsHostedObject386(g.c) {
		renvoRecordObjectFunctionRanges(g)
	}
	renvo_runtime_ArenaDiscard(g.meta.scratchStart, g.meta.scratchEnd)
	var data []byte
	if renvoFixedTarget == 0 && renvoIsHostedObject386(g.c) {
		data = renvoAsmImageObject386(a)
	} else if targetIsWindows(g.c.renvoTargetOS) {
		data = renvoAsmImageWindows386(a)
	} else {
		data = renvoAsmImage386(a)
	}
	var result renvoCompileResult
	if a.patchFailed || len(data) == 0 {
		return result
	}
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
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 0, dataOff, 0)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, dataOff, 0)
}

func renvo386AsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 0, bssOff, renvoAbsBssReloc)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 3, bssOff, renvoAbsBssReloc)
		return
	}
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
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
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
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

func renvo386EmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, srcPtr)
	renvoAsmEmit16(a, 0xc689)
	renvoAsmLoadPrimaryStack(a, destPtr)
	renvoAsmEmit16(a, 0xc789)
	renvoAsmLoadTertiaryStack(a, byteCount)
	renvoAsmEmitText(a, "\x39\xf7\x76\x0e\x8d\x74\x0e\xff\x8d\x7c\x0f\xff\xfd\xf3\xa4\xfc\xeb\x03\xfc\xf3\xa4")
}

func renvo386EnsureWideBinaryHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideBinaryLabel > 0 {
		return g.wideBinaryLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideBinaryLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x83\xf8\x0d\x75\x18\x8b\x0e\x8b\x56\x04\x8b\x03\xf7\xd0\x21\xc1\x8b\x43\x04\xf7\xd0\x21\xc2\x89\x0f\x89\x57\x04\xc3\x83\xf8\x0e\x72\x05\xe9\xb3\x01\x00\x00\x83\xf8\x00\x74\x1e\x83\xf8\x01\x74\x29\x83\xf8\x02\x74\x34\x83\xf8\x0a\x73\x4b\x83\xf8\x06\x0f\x86\xf2\x00\x00\x00\x83\xf8\x09\x76\x65\xc3\x8b\x06\x03\x03\x89\x07\x8b\x46\x04\x13\x43\x04\x89\x47\x04\xc3\x8b\x06\x2b\x03\x89\x07\x8b\x46\x04\x1b\x43\x04\x89\x47\x04\xc3\x8b\x06\xf7\x23\x89\x07\x89\xd1\x8b\x46\x04\x0f\xaf\x03\x01\xc1\x8b\x06\x0f\xaf\x43\x04\x01\xc8\x89\x47\x04\xc3\x8b\x0e\x8b\x56\x04\x83\xf8\x0a\x74\x0c\x83\xf8\x0b\x74\x0e\x33\x0b\x33\x53\x04\xeb\x0c\x23\x0b\x23\x53\x04\xeb\x05\x0b\x0b\x0b\x53\x04\x89\x0f\x89\x57\x04\xc3\x8b\x0e\x8b\x56\x04\x8b\x73\x04\x85\xf6\x75\x64\x8b\x1b\x83\xfb\x40\x73\x5d\x83\xfb\x20\x73\x2b\x51\x89\xd9\x83\xf8\x07\x74\x17\x5b\x89\xde\x0f\xad\xd6\x83\xf8\x08\x74\x04\xd3\xfa\xeb\x02\xd3\xea\x89\x37\x89\x57\x04\xc3\x5e\x0f\xa5\xf2\xd3\xe6\x89\x37\x89\x57\x04\xc3\x83\xeb\x20\x51\x89\xd9\x83\xf8\x07\x74\x15\x5e\x89\xd6\x83\xf8\x08\x74\x07\xd3\xfe\xc1\xfa\x1f\xeb\xd4\xd3\xee\x31\xd2\xeb\xce\x5e\x89\xf2\xd3\xe2\x31\xf6\x89\x37\x89\x57\x04\xc3\x83\xf8\x09\x74\x0a\x31\xc9\x31\xd2\x89\x0f\x89\x57\x04\xc3\xc1\xfa\x1f\x89\x17\x89\x57\x04\xc3\x55\x57\x50\x8b\x06\x8b\x56\x04\x8b\x0b\x8b\x5b\x04\x31\xf6\x31\xff\x80\x3c\x24\x05\x72\x1a\x89\xd6\xc1\xfe\x1f\x89\xdf\xc1\xff\x1f\x31\xf0\x31\xf2\x29\xf0\x19\xf2\x31\xf9\x31\xfb\x29\xf9\x19\xfb\x56\x57\x53\x51\x31\xc9\x31\xdb\x31\xf6\x31\xff\xbd\x40\x00\x00\x00\xd1\xe0\xd1\xd2\xd1\xd6\xd1\xd7\xd1\xe1\xd1\xd3\x3b\x7c\x24\x04\x72\x11\x77\x05\x3b\x34\x24\x72\x0a\x2b\x34\x24\x1b\x7c\x24\x04\x83\xc9\x01\x4d\x75\xda\xf6\x44\x24\x10\x01\x74\x1b\x8b\x44\x24\x0c\x33\x44\x24\x08\x31\xc1\x31\xc3\x29\xc1\x19\xc3\x8b\x54\x24\x14\x89\x0a\x89\x5a\x04\xeb\x15\x8b\x44\x24\x0c\x31\xc6\x31\xc7\x29\xc6\x19\xc7\x8b\x54\x24\x14\x89\x32\x89\x7a\x04\x83\xc4\x18\x5d\xc3\x83\xe8\x0e\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EnsureWideCompareHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideCompareLabel > 0 {
		return g.wideCompareLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideCompareLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EmitWideHelperCall(g *renvoLinearGen, dest int, left int, right int, mode int, label int) {
	a := &g.asm
	renvoAsmStackMem(a, dest, 0x8d, 0x7d, 0xbd)
	renvoAsmStackMem(a, left, 0x8d, 0x75, 0xb5)
	renvoAsmStackMem(a, right, 0x8d, 0x5d, 0x9d)
	renvoAsmPrimaryImm(a, mode)
	renvoAsmCallLabel(a, label)
}

func renvo386EmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	renvoNonNil(g)
	if mode >= 3 && mode <= 6 {
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvo386EmitWideHelperCall(g, dest, left, right, mode, renvo386EnsureWideBinaryHelper(g))
}

func renvo386EmitWideCompareStack(g *renvoLinearGen, left int, right int, mode int) {
	renvoNonNil(g)
	renvo386EmitWideHelperCall(g, 0, left, right, mode, renvo386EnsureWideCompareHelper(g))
}

func renvo386EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldLocalCount := g.localCount
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldClosureEnvOffset := g.closureEnvOffset
	oldDeferHeadOffset := g.deferHeadOffset
	oldDeferReturnLabel := g.deferReturnLabel
	oldDeferResultOffset := g.deferResultOffset
	oldDeferSites := g.deferSites
	oldEmittingDefers := g.emittingDefers
	oldSuppressPanicCheck := g.suppressPanicCheck
	oldStackUsed := g.stackUsed
	oldStackPeak := g.stackPeak
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.stackUsed = 0
	g.stackPeak = 0
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	renvoAsmEmit32(a, 0x000000c8)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
	}
	renvoBindFunctionParams(g, fnInfoIndex)
	if !renvoBindClosureCaptures(g, fnInfoIndex) {
		return false
	}
	if !renvoBindNamedResults(g, fnInfoIndex) {
		return false
	}
	if !renvoPrepareFunctionControl(g) {
		return false
	}
	if !renvoEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if g.deferReturnLabel > 0 {
		if !g.lastRangeReturns {
			renvoAsmJmpLabel(a, g.deferReturnLabel)
		}
		if !renvoEmitFunctionControlEpilogue(g) {
			return false
		}
	} else if !g.lastRangeReturns {
		renvoMoveCapturedLocals(g, true)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmLeave(a)
		renvoAsmRet(a)
	}
	frame := renvoAlignTo8(g.stackPeak)
	if frame > 65528 {
		frame = 65528
	}
	a.code[framePatch+1] = byte(frame & 255)
	a.code[framePatch+2] = byte((frame / 256) & 255)
	g.locals = oldLocals
	g.localCount = oldLocalCount
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.closureEnvOffset = oldClosureEnvOffset
	g.deferHeadOffset = oldDeferHeadOffset
	g.deferReturnLabel = oldDeferReturnLabel
	g.deferResultOffset = oldDeferResultOffset
	g.deferSites = oldDeferSites
	g.emittingDefers = oldEmittingDefers
	g.suppressPanicCheck = oldSuppressPanicCheck
	g.stackUsed = oldStackUsed
	g.stackPeak = oldStackPeak
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func renvo386StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return
	}
	if reg == 1 {
		renvoAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return
	}
	if reg == 2 {
		renvoAsmStoreSecondaryStack(a, offset)
		return
	}
	if reg == 3 {
		renvoAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return
	}
	if reg == 4 {
		renvoAsmStorePrimaryStack(a, offset)
		return
	}
	if reg == 5 {
		renvoAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return
	}
	renvoAsmEmit16(a, 0x858b)
	renvoAsmEmit32(a, 8+(reg-6)*4)
	renvoAsmStorePrimaryStack(a, offset)
}

func renvo386EmitRaxRcxOp(g *renvoLinearGen, tok int, unsigned bool) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		renvoAsmAddPrimaryTertiary(a)
		return true
	}
	if c0 == '-' {
		renvoAsmEmit16(a, 0xc129)
		renvoAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		renvoAsmEmit24(a, 0xc1af0f)
		return true
	}
	if c0 == '/' {
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if c0 == '%' {
		renvoAsmDivLeftTertiaryRightPrimary(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoAsmEmit16(a, 0xd0f7)
			renvoAsmEmit16(a, 0xc821)
		} else {
			renvoAsmEmit16(a, 0xc821)
		}
		return true
	}
	if c0 == '|' {
		renvoAsmEmit16(a, 0xc809)
		return true
	}
	if c0 == '^' {
		renvoAsmEmit16(a, 0xc831)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			renvoAsmEmit16(a, 0xe0d3)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9e)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			opcode := 0xf8d3
			if unsigned {
				opcode = 0xe8d3
			}
			renvoAsmEmit16(a, opcode)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x95)
		return true
	}
	return false
}

func renvo386EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		renvoAsmPopCallWord0(a)
	}
	if wordCount > 1 {
		renvoAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		renvoAsmPopSecondary(a)
	}
	if wordCount > 3 {
		renvoAsmPopTertiary(a)
	}
	if wordCount > 4 {
		renvoAsmPopPrimary(a)
	}
	if wordCount > 5 {
		renvoAsmEmit8(a, 0x5f)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if renvoAsmImmFits8Signed(imm) {
			renvoAsmEmit3(a, 0x83, 0xc4, imm)
		} else {
			renvoAsmEmit16(a, 0xc481)
			renvoAsmEmit32(a, imm)
		}
	}
}
func renvo386EmitEnsureMemSlice(g *renvoLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	renvoAsmPrimaryBssAddr(a, backingOff)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmPrimaryImm(a, backingSize/elemSize)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 16)
	renvoAsmMarkLabel(a, okLabel)
}

func renvo386EnsureAppendAddrHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := renvoEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.appendAddrLabel)
	noGrowLabel := renvoAsmNewLabel(a)
	capNonZeroLabel := renvoAsmNewLabel(a)
	capReadyLabel := renvoAsmNewLabel(a)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc139)
	renvo386AsmJccLabel(a, 0x8c, noGrowLabel)
	renvoAsmEmit8(a, 0x57)
	renvoAsmEmit8(a, 0x56)
	renvoAsmPushSecondary(a)
	renvoAsmPushTertiary(a)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc085)
	renvo386AsmJccLabel(a, 0x85, capNonZeroLabel)
	renvoAsmPrimaryImm(a, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoAsmPushPrimary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0x4c)
	renvoAsmEmit2(a, 0x24, 8)
	renvoAsmPushTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmCallLabel(a, arenaAllocLabel)
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmEmit16(a, 0xc085)
		renvo386AsmJccLabel(a, 0x85, allocOKLabel)
		renvoAsmEmit3(a, 0x83, 0xc4, 20)
		renvoAsmRet(a)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
	renvoAsmPushPrimary(a)
	renvoAsmEmit3(a, 0x8b, 0x3c, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 20)
	renvoAsmEmit16(a, 0x368b)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xc8)
	renvoAsmEmit16(a, 0xa4f3)
	renvoAsmEmit4(a, 0x8b, 0x7c, 0x24, 20)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit16(a, 0x0789)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 4)
	renvoAsmEmit3(a, 0x89, 0x46, 8)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x54, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xca)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmEmit3(a, 0x83, 0xc4, 24)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit16(a, 0x078b)
	renvoAsmEmit24(a, 0xcaaf0f)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvo386EnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x88\x14\x08\x41\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvo386EnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x89\x14\xc8\x41\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvo386EnsureStringEqualHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit16(a, 0xce39)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit16(a, 0xf685)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmit16(a, 0x0b8a)
	renvoAsmEmit16(a, 0x0a38)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit8(a, 0x43)
	renvoAsmEmit8(a, 0x42)
	renvoAsmEmit8(a, 0x4e)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
