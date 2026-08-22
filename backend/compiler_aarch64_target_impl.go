package main

const renvoAarch64RegRax = 0
const renvoAarch64RegRdx = 1
const renvoAarch64RegRcx = 2
const renvoAarch64RegRdi = 3
const renvoAarch64RegRsi = 4
const renvoAarch64RegR8 = 5
const renvoAarch64RegR9 = 6
const renvoAarch64RegR10 = 7
const renvoAarch64RegSys = 8
const renvoAarch64RegTmp = 9
const renvoAarch64RegTmp2 = 10
const renvoAarch64RegAddr = 12
const renvoAarch64RegFp = 29
const renvoAarch64RegLr = 30
const renvoAarch64RegZr = 31

func renvoAarch64EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
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
	renvoAarch64AsmAlign(a)
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	renvoAarch64AsmEmit(a, 0xa9bf7bfd)
	renvoAarch64AsmEmit(a, 0x910003fd)
	framePatch := renvoAarch64AsmFrameStart(a)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRdi, g.returnStruct)
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
	renvoAarch64AsmPatchFrame(a, framePatch, g.stackPeak)
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

func renvoAarch64StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRdi, offset)
		return
	}
	if reg == 1 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRsi, offset)
		return
	}
	if reg == 2 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRdx, offset)
		return
	}
	if reg == 3 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRcx, offset)
		return
	}
	if reg == 4 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegR8, offset)
		return
	}
	if reg == 5 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegR9, offset)
		return
	}
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegFp, 16+(reg-6)*16, 8)
	renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRax, offset)
}

func renvoAarch64EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegRdi)
	}
	if wordCount > 1 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegRsi)
	}
	if wordCount > 2 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegRdx)
	}
	if wordCount > 3 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegRcx)
	}
	if wordCount > 4 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegR8)
	}
	if wordCount > 5 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegR9)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		renvoAarch64AsmAddRegImm(a, 31, 31, (wordCount-6)*16)
	}
}

func renvoAarch64EmitRaxRcxOp(g *renvoLinearGen, tok int) bool {
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
		renvoAarch64AsmAddRegReg(a, renvoAarch64RegRax, renvoAarch64RegRcx, renvoAarch64RegRax)
		return true
	}
	if c0 == '-' {
		renvoAarch64AsmSubRegReg(a, renvoAarch64RegRax, renvoAarch64RegRcx, renvoAarch64RegRax)
		return true
	}
	if c0 == '*' {
		renvoAarch64AsmEmit(a, 0x9b007c40)
		return true
	}
	if c0 == '/' {
		renvoAarch64AsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		renvoAarch64AsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoAarch64AsmEmit(a, 0x8a200040)
		} else {
			renvoAarch64AsmEmit(a, 0x8a000040)
		}
		return true
	}
	if c0 == '|' {
		renvoAarch64AsmEmit(a, 0xaa000040)
		return true
	}
	if c0 == '^' {
		renvoAarch64AsmEmit(a, 0xca000040)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoAarch64AsmEmit(a, 0x9ac02040)
		} else if c1 == '=' {
			renvoAarch64AsmCmpRcxRaxSet(a, 0x9e)
		} else {
			renvoAarch64AsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoAarch64AsmEmit(a, 0x9ac02840)
		} else if c1 == '=' {
			renvoAarch64AsmCmpRcxRaxSet(a, 0x9d)
		} else {
			renvoAarch64AsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoAarch64AsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoAarch64AsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func renvoAarch64EnsureAppendAddrHelper(g *renvoLinearGen) int {
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
	copyLoopLabel := renvoAsmNewLabel(a)
	copyDoneLabel := renvoAsmNewLabel(a)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegR8, renvoAarch64RegRsi, 0, 8)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 8, 8)
	renvoAarch64AsmCmpRegReg(a, renvoAarch64RegR8, renvoAarch64RegRcx)
	renvoAarch64AsmBCondLabel(a, noGrowLabel, 11)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR9, renvoAarch64RegRdx)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR10, renvoAarch64RegRdi)
	renvoAarch64AsmCmpRegImm(a, renvoAarch64RegRcx, 0)
	renvoAarch64AsmBCondLabel(a, capNonZeroLabel, 1)
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegRcx, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegRcx, renvoAarch64RegRcx, renvoAarch64RegR8)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoAarch64AsmMulRegReg(a, renvoAarch64RegTmp, renvoAarch64RegRcx, renvoAarch64RegR9)
	renvoAsmPushTertiary(a)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRax, renvoAarch64RegTmp)
	renvoAarch64AsmPushReg(a, renvoAarch64RegLr)
	renvoAsmCallLabel(a, arenaAllocLabel)
	renvoAarch64AsmPopReg(a, renvoAarch64RegLr)
	renvoAsmPopTertiary(a)
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAarch64AsmCmpRegImm(a, renvoAarch64RegRax, 0)
		renvoAarch64AsmBCondLabel(a, allocOKLabel, 1)
		renvoAsmRet(a)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRax)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdi, renvoAarch64RegRdx)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp2, renvoAarch64RegR10, 0, 8)
	renvoAarch64AsmMulRegReg(a, renvoAarch64RegTmp, renvoAarch64RegR8, renvoAarch64RegR9)
	renvoAsmMarkLabel(a, copyLoopLabel)
	renvoAarch64AsmCmpRegImm(a, renvoAarch64RegTmp, 0)
	renvoAarch64AsmBCondLabel(a, copyDoneLabel, 0)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegTmp2, 0, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdi, 0, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegTmp2, renvoAarch64RegTmp2, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRdi, renvoAarch64RegRdi, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegTmp, renvoAarch64RegTmp, -1)
	renvoAsmJmpMarkLabel(a, copyLoopLabel, copyDoneLabel)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRdx, renvoAarch64RegR10, 0, 8)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 8, 8)
	renvoAarch64AsmMulRegReg(a, renvoAarch64RegTmp, renvoAarch64RegR8, renvoAarch64RegR9)
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegRax, renvoAarch64RegRdx, renvoAarch64RegTmp)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegR8, renvoAarch64RegR8, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegR8, renvoAarch64RegRsi, 0, 8)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdi, 0, 8)
	renvoAarch64AsmMulRegReg(a, renvoAarch64RegTmp, renvoAarch64RegR8, renvoAarch64RegRdx)
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegRax, renvoAarch64RegRax, renvoAarch64RegTmp)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegR8, renvoAarch64RegR8, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegR8, renvoAarch64RegRsi, 0, 8)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvoAarch64EnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 0, 8)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdi, 0, 8)
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegTmp, renvoAarch64RegTmp, renvoAarch64RegRcx)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRdx, renvoAarch64RegTmp, 0, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRcx, renvoAarch64RegRcx, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 0, 8)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvoAarch64EnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 0, 8)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdi, 0, 8)
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegTmp, renvoAarch64RegTmp, renvoAarch64RegRcx, 3)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRdx, renvoAarch64RegTmp, 0, 8)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRcx, renvoAarch64RegRcx, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRsi, 0, 8)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvoAarch64EnsureStringEqualHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAarch64AsmCmpRegReg(a, renvoAarch64RegRsi, renvoAarch64RegRcx)
	renvoAarch64AsmBCondLabel(a, notEqualLabel, 1)
	renvoAarch64AsmCmpRegImm(a, renvoAarch64RegRsi, 0)
	renvoAarch64AsmBCondLabel(a, equalLabel, 0)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdi, 0, 1)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp2, renvoAarch64RegRdx, 0, 1)
	renvoAarch64AsmCmpRegReg(a, renvoAarch64RegTmp, renvoAarch64RegTmp2)
	renvoAarch64AsmBCondLabel(a, notEqualLabel, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRdi, renvoAarch64RegRdi, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRdx, renvoAarch64RegRdx, 1)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRsi, renvoAarch64RegRsi, -1)
	renvoAarch64AsmCmpRegImm(a, renvoAarch64RegRsi, 0)
	renvoAarch64AsmBCondLabel(a, loopLabel, 1)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

const renvoAarch64ELFCodeOffset = 0xb0

func renvoCompileAarch64(input []int, output int, arenaSize int) int {
	sourceCapacity := 786432
	src := make([]byte, 0, sourceCapacity)
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
	result = renvoTryCompileScalarProgramAarch64Scratch(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramAarch64(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	return renvoTryCompileScalarProgramAarch64Scratch(p, meta)
}

func renvoTryCompileScalarProgramAarch64Scratch(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	session := renvoBeginScalarProgramAarch64(p, meta)
	if session == nil {
		return renvoCompileResult{}
	}
	for !session.stepScratch(64) {
	}
	return session.result
}

func renvoTryCompileScalarProgramAarch64Cached(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	session := renvoBeginScalarProgramAarch64(p, meta)
	if session == nil {
		return renvoCompileResult{}
	}
	for !session.stepCached(64) {
	}
	return session.result
}

type renvoAarch64ProgramSession struct {
	prog       *renvoProgram
	gen        renvoLinearGen
	queueIndex int
	done       bool
	result     renvoCompileResult
}

func renvoBeginScalarProgramAarch64(p *renvoProgram, meta *renvoMeta) *renvoAarch64ProgramSession {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		return nil
	}
	session := &renvoAarch64ProgramSession{prog: p}
	g := &session.gen
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	a.codeOffset = renvoAarch64ELFCodeOffset
	if targetIsWindows(meta.c.renvoTargetOS) {
		a.codeOffset = renvoWinSectionRVA
	}
	if targetIsDarwin(meta.c.renvoTargetOS) {
		a.codeOffset = renvoDarwinArm64CodeOffset
		if !(renvoFixedTarget == 0 && meta.c.emitImage) {
			g.darwinEntryOff = a.bssSize
			a.bssSize += 24
			renvoAarch64AsmMovRegAbs(a, 9, g.darwinEntryOff, renvoAbsBssReloc)
			renvoAarch64AsmStoreRegMem(a, 0, 9, 0, 8)
			renvoAarch64AsmStoreRegMem(a, 1, 9, 8, 8)
			renvoAarch64AsmStoreRegMem(a, 2, 9, 16, 8)
		}
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
	if renvoFixedTarget == 0 && meta.c.emitImage {
		// Give the callable image entry a normal frame and preserve its four
		// ABI words across runtime/global initialization.
		renvoAarch64AsmEmit(a, 0xa9bf7bfd)
		renvoAarch64AsmEmit(a, 0x910003fd)
		renvoAarch64AsmAddRegImm(a, 31, 31, -32)
		renvoAarch64AsmStoreRegMem(a, 0, 31, 0, 8)
		renvoAarch64AsmStoreRegMem(a, 1, 31, 8, 8)
		renvoAarch64AsmStoreRegMem(a, 2, 31, 16, 8)
		renvoAarch64AsmStoreRegMem(a, 3, 31, 24, 8)
	}
	renvoEmitInitializeThreadState(g)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return nil
	}
	entryOK := false
	if renvoFixedTarget == 0 && meta.c.emitImage {
		entryOK = renvoEmitImageEntryArgsAarch64(g, appIndex)
	} else if targetIsWindows(meta.c.renvoTargetOS) {
		entryOK = renvoEmitProgramEntryArgsWindowsArm64(g, appIndex)
	} else if targetIsDarwin(meta.c.renvoTargetOS) {
		entryOK = renvoEmitProgramEntryArgsDarwinArm64(g, appIndex)
	} else {
		entryOK = renvoEmitProgramEntryArgsAarch64(g, appIndex)
	}
	if !entryOK {
		return nil
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return nil
	}
	if renvoFixedTarget == 0 && meta.c.emitImage {
		renvoAsmLeave(a)
		renvoAsmRet(a)
	} else if targetIsWindows(meta.c.renvoTargetOS) {
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
		renvoWinArm64CallImport(a, renvoWinImportExitProcess)
		renvoAsmRet(a)
	} else if targetIsDarwin(meta.c.renvoTargetOS) {
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
		renvoDarwinArm64CallImport(a, renvoDarwinImportExit)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 93)
		renvoAsmSyscall(a)
	}
	return session
}

func renvoEmitImageEntryArgsAarch64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	renvoAarch64AsmLoadRegMem(&g.asm, 0, 31, 0, 8)
	renvoAarch64AsmLoadRegMem(&g.asm, 1, 31, 8, 8)
	renvoAarch64AsmLoadRegMem(&g.asm, 2, 31, 16, 8)
	renvoAarch64AsmLoadRegMem(&g.asm, 3, 31, 24, 8)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) {
		return false
	}
	// Native entry ABI: X0=argsData, X1=argsLen, X2=envData,
	// X3=envLen. Renvo slice words use X3/X4/X1 and X2/X5/X6.
	if app.paramCount == 2 {
		if !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) {
			return false
		}
		renvoAarch64AsmMovRegReg(&g.asm, renvoAarch64RegR8, 3)
		renvoAarch64AsmMovRegReg(&g.asm, renvoAarch64RegR9, 3)
	}
	renvoAarch64AsmMovRegReg(&g.asm, renvoAarch64RegRdi, 0)
	renvoAarch64AsmMovRegReg(&g.asm, renvoAarch64RegRsi, 1)
	return true
}

func (s *renvoAarch64ProgramSession) step(functionLimit int) bool {
	return s.stepCached(functionLimit)
}

func (s *renvoAarch64ProgramSession) stepScratch(functionLimit int) bool {
	if s == nil || s.done {
		return true
	}
	if functionLimit < 1 {
		functionLimit = 1
	}
	emitted := 0
	for s.queueIndex < len(s.gen.funcQueue) && emitted < functionLimit {
		i := s.gen.funcQueue[s.queueIndex]
		s.queueIndex++
		emitted++
		if renvoDeferUnreadyQueuedClosure(&s.gen, i) {
			continue
		}
		if !renvoEmitScalarFunctionScratch(&s.gen, i) {
			return s.failFunction(i)
		}
	}
	return s.finishStep()
}

func (s *renvoAarch64ProgramSession) stepCached(functionLimit int) bool {
	if s == nil || s.done {
		return true
	}
	if functionLimit < 1 {
		functionLimit = 1
	}
	emitted := 0
	for s.queueIndex < len(s.gen.funcQueue) && emitted < functionLimit {
		i := s.gen.funcQueue[s.queueIndex]
		s.queueIndex++
		emitted++
		if renvoDeferUnreadyQueuedClosure(&s.gen, i) {
			continue
		}
		if !renvoEmitScalarFunctionObjectCached(&s.gen, i) {
			return s.failFunction(i)
		}
	}
	return s.finishStep()
}

func (s *renvoAarch64ProgramSession) failFunction(i int) bool {
	if targetIsDarwin(s.gen.c.renvoTargetOS) {
		renvoPrintErr("renvo: failed to emit function ")
		write(2, s.prog.src[s.gen.meta.funcs[i].nameStart:s.gen.meta.funcs[i].nameEnd], -1)
		renvoPrintErr("\n")
	}
	s.done = true
	return true
}

func (s *renvoAarch64ProgramSession) finishStep() bool {
	if s.queueIndex < len(s.gen.funcQueue) {
		return false
	}
	a := &s.gen.asm
	data := renvoAsmImageAarch64(a)
	if targetIsWindows(s.gen.c.renvoTargetOS) {
		data = renvoAsmImageWindowsArm64(a)
	} else if targetIsDarwin(s.gen.c.renvoTargetOS) {
		data = renvoAsmImageDarwinArm64(a)
	}
	if a.patchFailed || len(data) == 0 {
		s.done = true
		return true
	}
	s.result.data = data
	s.result.ok = true
	s.done = true
	return true
}
func renvoAarch64AsmFrameStart(a *renvoAsm) int {
	at := len(a.code)
	count := 2
	if a.c.renvoTargetOS == renvoOSWindows {
		// Windows requires every intervening page to be touched when a frame
		// crosses a guard page. The compact loop patched below handles an
		// arbitrary calculated frame without reserving one probe per page.
		count = 8
	}
	for i := 0; i < count; i++ {
		renvoAarch64AsmEmit(a, 0xd503201f) // nop
	}
	return at
}

func renvoAarch64AsmPatchFrame(a *renvoAsm, at int, stackUsed int) {
	frame := renvoAlignValue(stackUsed, 16)
	if a.c.renvoTargetOS == renvoOSWindows {
		pages := frame / 4096
		tail := frame % 4096
		if frame == 0 {
			return
		}
		// movz x9, #pages
		renvoPut32At(a.code, at, 0xd2800009|(pages<<5))
		// cbz x9, tail (five instructions forward)
		renvoPut32At(a.code, at+4, 0xb40000a9)
		// loop: sub sp, sp, #1, lsl #12; touch the new page.
		renvoPut32At(a.code, at+8, 0xd14007ff)
		renvoPut32At(a.code, at+12, 0xf90003ff)
		// subs x9, x9, #1; b.ne loop (three instructions back).
		renvoPut32At(a.code, at+16, 0xf1000529)
		renvoPut32At(a.code, at+20, 0x54ffffa1)
		if tail > 0 {
			renvoPut32At(a.code, at+24, 0xd10003ff|(tail<<10))
			renvoPut32At(a.code, at+28, 0xf90003ff)
		}
		return
	}
	high := frame / 4096
	low := frame % 4096
	if high > 0 {
		renvoPut32At(a.code, at, 0xd14003ff|(high<<10))
	}
	if low > 0 {
		renvoPut32At(a.code, at+4, 0xd10003ff|(low<<10))
	}
}
