package main

const renvoArmRegRax = 0
const renvoArmRegRdx = 1
const renvoArmRegRcx = 2
const renvoArmRegRdi = 3
const renvoArmRegRsi = 4
const renvoArmRegR8 = 5
const renvoArmRegR9 = 6
const renvoArmRegSys = 7
const renvoArmRegR10 = 8
const renvoArmRegTmp = 9
const renvoArmRegTmp2 = 10
const renvoArmRegFp = 11
const renvoArmRegAddr = 12
const renvoArmRegSp = 13
const renvoArmRegLr = 14

func renvoArmEmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
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
	var locals []renvoLocalInfo
	var gotoLabels []renvoGlobalInfo
	locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	gotoLabels = make([]renvoGlobalInfo, 0, 0)
	g.locals = locals
	g.localCount = 0
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.stackUsed = 0
	g.stackPeak = 0
	renvoArmAsmAlign(a)
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	renvoArmAsmEmit(a, 0xe92d4800)
	renvoArmAsmMovRegReg(a, renvoArmRegFp, renvoArmRegSp)
	framePatch := renvoArmAsmFrameStart(a)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoArmAsmStoreRegStack(a, renvoArmRegRdi, g.returnStruct)
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
	renvoArmAsmPatchFrame(a, framePatch, g.stackPeak)
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

func renvoArmStoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRdi, offset)
		return
	}
	if reg == 1 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRsi, offset)
		return
	}
	if reg == 2 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRdx, offset)
		return
	}
	if reg == 3 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRcx, offset)
		return
	}
	if reg == 4 {
		renvoArmAsmStoreRegStack(a, renvoArmRegR8, offset)
		return
	}
	if reg == 5 {
		renvoArmAsmStoreRegStack(a, renvoArmRegR9, offset)
		return
	}
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegFp, 8+(reg-6)*4, 4)
	renvoArmAsmStoreRegStack(a, renvoArmRegRax, offset)
}

func renvoArmEmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		renvoArmAsmPopReg(a, renvoArmRegRdi)
	}
	if wordCount > 1 {
		renvoArmAsmPopReg(a, renvoArmRegRsi)
	}
	if wordCount > 2 {
		renvoArmAsmPopReg(a, renvoArmRegRdx)
	}
	if wordCount > 3 {
		renvoArmAsmPopReg(a, renvoArmRegRcx)
	}
	if wordCount > 4 {
		renvoArmAsmPopReg(a, renvoArmRegR8)
	}
	if wordCount > 5 {
		renvoArmAsmPopReg(a, renvoArmRegR9)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		renvoArmAsmAddRegImm(a, renvoArmRegSp, renvoArmRegSp, (wordCount-6)*4)
	}
}

func renvoArmEmitRaxRcxOp(g *renvoLinearGen, tok int) bool {
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
		renvoArmAsmAddRegReg(a, renvoArmRegRax, renvoArmRegRcx, renvoArmRegRax)
		return true
	}
	if c0 == '-' {
		renvoArmAsmSubRegReg(a, renvoArmRegRax, renvoArmRegRcx, renvoArmRegRax)
		return true
	}
	if c0 == '*' {
		renvoArmAsmMulRegReg(a, renvoArmRegRax, renvoArmRegRcx, renvoArmRegRax)
		return true
	}
	if c0 == '/' {
		renvoArmAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		renvoArmAsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoArmAsmEmit(a, 0xe1e00000|(renvoArmRegRax<<12)|renvoArmRegRax)
			renvoArmAsmEmit(a, 0xe0000000|(renvoArmRegRcx<<16)|(renvoArmRegRax<<12)|renvoArmRegRax)
		} else {
			renvoArmAsmEmit(a, 0xe0000000|(renvoArmRegRcx<<16)|(renvoArmRegRax<<12)|renvoArmRegRax)
		}
		return true
	}
	if c0 == '|' {
		renvoArmAsmEmit(a, 0xe1800000|(renvoArmRegRcx<<16)|(renvoArmRegRax<<12)|renvoArmRegRax)
		return true
	}
	if c0 == '^' {
		renvoArmAsmEmit(a, 0xe0200000|(renvoArmRegRcx<<16)|(renvoArmRegRax<<12)|renvoArmRegRax)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoArmAsmEmit(a, 0xe1a00010|(renvoArmRegRax<<8)|(renvoArmRegRax<<12)|renvoArmRegRcx)
		} else if c1 == '=' {
			renvoArmAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			renvoArmAsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoArmAsmEmit(a, 0xe1a00050|(renvoArmRegRax<<8)|(renvoArmRegRax<<12)|renvoArmRegRcx)
		} else if c1 == '=' {
			renvoArmAsmCmpRcxRaxSet(a, 0x9d)
		} else {
			renvoArmAsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoArmAsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoArmAsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func renvoArmEmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoTokCharIs(p, e.tok, '*') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoArmAsmMulRegReg(a, renvoArmRegRax, renvoArmRegRcx, renvoArmRegRax)
		renvoAsmSarPrimaryImm(a, 2)
		return true
	}
	if renvoTokCharIs(p, e.tok, '/') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmShlPrimaryImm(a, 2)
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
		return false
	}
	renvoAsmPopTertiary(a)
	return renvoEmitPrimaryTertiaryOp(g, e.tok)
}

func renvoArmEmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoArmAsmMovRegReg(a, renvoArmRegRdi, renvoArmRegRdx)
		renvoArmAsmAddRegImm(a, renvoArmRegRsi, renvoArmRegRdx, 8)
		return true
	}
	if loc.global {
		renvoArmAsmMovRegAbs(a, renvoArmRegRdi, loc.offset, renvoAbsBssReloc)
		renvoArmAsmMovRegAbs(a, renvoArmRegRsi, loc.offset+8, renvoAbsBssReloc)
		return true
	}
	renvoArmAsmLeaRegStack(a, renvoArmRegRdi, loc.offset)
	renvoArmAsmLeaRegStack(a, renvoArmRegRsi, loc.offset-8)
	return true
}

func renvoArmEnsureAppendAddrHelper(g *renvoLinearGen) int {
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
	renvoArmAsmLoadRegMem(a, renvoArmRegR8, renvoArmRegRsi, 0, 4)
	renvoArmAsmLoadRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 8, 4)
	renvoArmAsmCmpRegReg(a, renvoArmRegR8, renvoArmRegRcx)
	renvoArmAsmBCondLabel(a, noGrowLabel, 11)
	renvoArmAsmMovRegReg(a, renvoArmRegR9, renvoArmRegRdx)
	renvoArmAsmMovRegReg(a, renvoArmRegR10, renvoArmRegRdi)
	renvoArmAsmCmpRegImm(a, renvoArmRegRcx, 0)
	renvoArmAsmBCondLabel(a, capNonZeroLabel, 1)
	renvoArmAsmMovRegImm(a, renvoArmRegRcx, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoArmAsmAddRegReg(a, renvoArmRegRcx, renvoArmRegRcx, renvoArmRegR8)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoArmAsmMulRegReg(a, renvoArmRegTmp, renvoArmRegRcx, renvoArmRegR9)
	renvoAsmPushTertiary(a)
	renvoArmAsmMovRegReg(a, renvoArmRegRax, renvoArmRegTmp)
	renvoArmAsmPushReg(a, renvoArmRegLr)
	renvoAsmCallLabel(a, arenaAllocLabel)
	renvoArmAsmPopReg(a, renvoArmRegLr)
	renvoAsmPopTertiary(a)
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegRax)
	renvoArmAsmMovRegReg(a, renvoArmRegRdi, renvoArmRegRdx)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp2, renvoArmRegR10, 0, 4)
	renvoArmAsmMulRegReg(a, renvoArmRegTmp, renvoArmRegR8, renvoArmRegR9)
	renvoAsmMarkLabel(a, copyLoopLabel)
	renvoArmAsmCmpRegImm(a, renvoArmRegTmp, 0)
	renvoArmAsmBCondLabel(a, copyDoneLabel, 0)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegTmp2, 0, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegRdi, 0, 1)
	renvoArmAsmAddRegSmallImm(a, renvoArmRegTmp2, renvoArmRegTmp2, 1)
	renvoArmAsmAddRegSmallImm(a, renvoArmRegRdi, renvoArmRegRdi, 1)
	renvoArmAsmAddRegSmallImm(a, renvoArmRegTmp, renvoArmRegTmp, -1)
	renvoAsmJmpMarkLabel(a, copyLoopLabel, copyDoneLabel)
	renvoArmAsmStoreRegMem(a, renvoArmRegRdx, renvoArmRegR10, 0, 4)
	renvoArmAsmStoreRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 8, 4)
	renvoArmAsmMulRegReg(a, renvoArmRegTmp, renvoArmRegR8, renvoArmRegR9)
	renvoArmAsmAddRegReg(a, renvoArmRegRax, renvoArmRegRdx, renvoArmRegTmp)
	renvoArmAsmAddRegImm(a, renvoArmRegR8, renvoArmRegR8, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegR8, renvoArmRegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegRdi, 0, 4)
	renvoArmAsmMulRegReg(a, renvoArmRegTmp, renvoArmRegR8, renvoArmRegRdx)
	renvoArmAsmAddRegReg(a, renvoArmRegRax, renvoArmRegRax, renvoArmRegTmp)
	renvoArmAsmAddRegImm(a, renvoArmRegR8, renvoArmRegR8, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegR8, renvoArmRegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvoArmEnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoArmAsmLoadRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 0, 4)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegRdi, 0, 4)
	renvoArmAsmAddRegReg(a, renvoArmRegTmp, renvoArmRegTmp, renvoArmRegRcx)
	renvoArmAsmStoreRegMem(a, renvoArmRegRdx, renvoArmRegTmp, 0, 1)
	renvoArmAsmAddRegImm(a, renvoArmRegRcx, renvoArmRegRcx, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvoArmEnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoArmAsmLoadRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 0, 4)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegRdi, 0, 4)
	renvoArmAsmAddRegRegShift(a, renvoArmRegTmp, renvoArmRegTmp, renvoArmRegRcx, 3)
	renvoArmAsmStoreRegMem(a, renvoArmRegRdx, renvoArmRegTmp, 0, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegRcx, renvoArmRegRcx, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegRcx, renvoArmRegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvoArmEnsureStringEqualHelper(g *renvoLinearGen) int {
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
	renvoArmAsmCmpRegReg(a, renvoArmRegRsi, renvoArmRegRcx)
	renvoArmAsmBCondLabel(a, notEqualLabel, 1)
	renvoArmAsmCmpRegImm(a, renvoArmRegRsi, 0)
	renvoArmAsmBCondLabel(a, equalLabel, 0)
	renvoAsmMarkLabel(a, loopLabel)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegRdi, 0, 1)
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp2, renvoArmRegRdx, 0, 1)
	renvoArmAsmCmpRegReg(a, renvoArmRegTmp, renvoArmRegTmp2)
	renvoArmAsmBCondLabel(a, notEqualLabel, 1)
	renvoArmAsmAddRegImm(a, renvoArmRegRdi, renvoArmRegRdi, 1)
	renvoArmAsmAddRegImm(a, renvoArmRegRdx, renvoArmRegRdx, 1)
	renvoArmAsmAddRegImm(a, renvoArmRegRsi, renvoArmRegRsi, -1)
	renvoArmAsmCmpRegImm(a, renvoArmRegRsi, 0)
	renvoArmAsmBCondLabel(a, loopLabel, 1)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func renvoArmAsmEmit(a *renvoAsm, insn int) {
	renvoAsmEmit32(a, insn)
}

func renvoArmAsmAlign(a *renvoAsm) {
	for len(a.code)%4 != 0 {
		renvoAsmEmit8(a, 0)
	}
}

func renvoArmAsmMovRegReg(a *renvoAsm, dst int, src int) {
	if dst == src {
		return
	}
	renvoArmAsmEmit(a, 0xe1a00000|(dst<<12)|src)
}

func renvoArmAsmMovRegImm(a *renvoAsm, reg int, imm int) {
	part := imm & 65535
	renvoArmAsmEmit(a, 0xe3000000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	part = (imm >> 16) & 65535
	if part != 0 {
		renvoArmAsmEmit(a, 0xe3400000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	}
}

func renvoArmAsmPatchMovRegImmAt(a *renvoAsm, at int, reg int, imm int) {
	part := imm & 65535
	renvoPut32At(a.code, at, 0xe3000000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	part = (imm >> 16) & 65535
	renvoPut32At(a.code, at+4, 0xe3400000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
}

func renvoArmAsmMovRegAbs(a *renvoAsm, reg int, off int, kind int) {
	at := len(a.code)
	renvoArmAsmMovRegImm(a, reg, 0)
	renvoArmAsmEmit(a, 0xe3400000|(reg<<12))
	renvoAsmAddAbsReloc(a, at, off, kind)
}

func renvoArmAsmAddRegImm(a *renvoAsm, dst int, src int, imm int) {
	if imm == 0 {
		renvoArmAsmMovRegReg(a, dst, src)
		return
	}
	tmp := renvoArmRegTmp
	if dst == tmp || src == tmp {
		tmp = renvoArmRegTmp2
	}
	if imm < 0 {
		renvoArmAsmMovRegImm(a, tmp, -imm)
		renvoArmAsmSubRegReg(a, dst, src, tmp)
		return
	}
	renvoArmAsmMovRegImm(a, tmp, imm)
	renvoArmAsmAddRegReg(a, dst, src, tmp)
}

func renvoArmAsmFrameStart(a *renvoAsm) int {
	at := len(a.code)
	renvoArmAsmEmit(a, 0xe3009000) // movw r9, #0
	renvoArmAsmEmit(a, 0xe04dd009) // sub sp, sp, r9
	return at
}

func renvoArmAsmPatchFrame(a *renvoAsm, at int, stackUsed int) {
	frame := renvoAlignTo8(stackUsed)
	if frame > 65528 {
		frame = 65528
	}
	renvoPut32At(a.code, at, 0xe3009000|((frame&0xf000)<<4)|(frame&0x0fff))
}

func renvoArmAsmAddRegSmallImm(a *renvoAsm, dst int, src int, imm int) {
	if imm < 0 {
		renvoArmAsmEmit(a, 0xe2400000|(src<<16)|(dst<<12)|(-imm))
		return
	}
	renvoArmAsmEmit(a, 0xe2800000|(src<<16)|(dst<<12)|imm)
}

func renvoArmAsmAddRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoArmAsmEmit(a, 0xe0800000|(left<<16)|(dst<<12)|right)
}

func renvoArmAsmSubRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoArmAsmEmit(a, 0xe0400000|(left<<16)|(dst<<12)|right)
}

func renvoArmAsmAddRegRegShift(a *renvoAsm, dst int, left int, right int, shift int) {
	renvoArmAsmEmit(a, 0xe0800000|(left<<16)|(dst<<12)|(shift<<7)|right)
}

func renvoArmAsmMulRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoArmAsmEmit(a, 0xe0000090|(dst<<16)|(right<<8)|left)
}

func renvoArmAsmAddr(a *renvoAsm, base int, disp int) int {
	if disp == 0 {
		return base
	}
	renvoArmAsmAddRegImm(a, renvoArmRegAddr, base, disp)
	return renvoArmRegAddr
}

func renvoArmAsmLoadStoreAddr(a *renvoAsm, base int, disp int, size int) int {
	if size == 2 && (disp < 0 || disp > 255) {
		return renvoArmAsmAddr(a, base, disp)
	}
	if size != 2 && (disp < -4095 || disp > 4095) {
		return renvoArmAsmAddr(a, base, disp)
	}
	return base
}

func renvoArmAsmLoadRegMem(a *renvoAsm, dst int, base int, disp int, size int) {
	addr := renvoArmAsmLoadStoreAddr(a, base, disp, size)
	if addr != base {
		disp = 0
	}
	up := 0x00800000
	if disp < 0 {
		up = 0
		disp = -disp
	}
	if size == 1 {
		renvoArmAsmEmit(a, 0xe5500000|up|(addr<<16)|(dst<<12)|disp)
		return
	}
	if size == 2 {
		renvoArmAsmEmit(a, 0xe15000f0|up|(addr<<16)|(dst<<12)|((disp&0xf0)<<4)|(disp&0x0f))
		return
	}
	renvoArmAsmEmit(a, 0xe5100000|up|(addr<<16)|(dst<<12)|disp)
}

func renvoArmAsmStoreRegMem(a *renvoAsm, src int, base int, disp int, size int) {
	addr := renvoArmAsmLoadStoreAddr(a, base, disp, size)
	if addr != base {
		disp = 0
	}
	up := 0x00800000
	if disp < 0 {
		up = 0
		disp = -disp
	}
	if size == 1 {
		renvoArmAsmEmit(a, 0xe5400000|up|(addr<<16)|(src<<12)|disp)
		return
	}
	if size == 2 {
		renvoArmAsmEmit(a, 0xe14000b0|up|(addr<<16)|(src<<12)|((disp&0xf0)<<4)|(disp&0x0f))
		return
	}
	renvoArmAsmEmit(a, 0xe5000000|up|(addr<<16)|(src<<12)|disp)
}

func renvoArmAsmLoadRegStack(a *renvoAsm, dst int, offset int) {
	renvoArmAsmLoadRegMem(a, dst, renvoArmRegFp, -offset, 4)
}

func renvoArmAsmStoreRegStack(a *renvoAsm, src int, offset int) {
	renvoArmAsmStoreRegMem(a, src, renvoArmRegFp, -offset, 4)
}

func renvoArmAsmLeaRegStack(a *renvoAsm, dst int, offset int) {
	renvoArmAsmAddRegImm(a, dst, renvoArmRegFp, -offset)
}

func renvoArmAsmPushReg(a *renvoAsm, reg int) {
	renvoArmAsmEmit(a, 0xe52d0004|(reg<<12))
}

func renvoArmAsmPopReg(a *renvoAsm, reg int) {
	renvoArmAsmEmit(a, 0xe49d0004|(reg<<12))
}

func renvoArmAsmMovRaxImm(a *renvoAsm, imm int) {
	renvoArmAsmMovRegImm(a, renvoArmRegRax, imm)
}

func renvoArmAsmMovRaxImm64(a *renvoAsm, imm int) {
	renvoArmAsmMovRaxImm(a, imm)
}

func renvoArmAsmMovRdxImm(a *renvoAsm, imm int) {
	renvoArmAsmMovRegImm(a, renvoArmRegRdx, imm)
}

func renvoArmAsmMovRaxDataAddr(a *renvoAsm, dataOff int) {
	renvoArmAsmMovRegAbs(a, renvoArmRegRax, dataOff, 0)
}

func renvoArmAsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	renvoArmAsmMovRegAbs(a, renvoArmRegRax, bssOff, renvoAbsBssReloc)
}

func renvoArmAsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	renvoArmAsmMovRegAbs(a, renvoArmRegR10, bssOff, renvoAbsBssReloc)
}

func renvoArmAsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoArmAsmMovRegAbs(a, renvoArmRegAddr, bssOff, renvoAbsBssReloc)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 4)
}

func renvoArmAsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoArmAsmMovRegAbs(a, renvoArmRegAddr, bssOff, renvoAbsBssReloc)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 4)
}

func renvoArmAsmMovRdiRax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRdi, renvoArmRegRax)
}

func renvoArmAsmMovRaxRdx(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRax, renvoArmRegRdx)
}

func renvoArmAsmMovRdxRax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegRax)
}

func renvoArmAsmMovRcxRax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRcx, renvoArmRegRax)
}

func renvoArmAsmMovRcxRdx(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRcx, renvoArmRegRdx)
}

func renvoArmAsmMovRsiRax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRsi, renvoArmRegRax)
}

func renvoArmAsmMovR8Rax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegR8, renvoArmRegRax)
}

func renvoArmAsmMovR9Rax(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegR9, renvoArmRegRax)
}

func renvoArmAsmAddRdxRcx(a *renvoAsm) {
	renvoArmAsmAddRegReg(a, renvoArmRegRdx, renvoArmRegRdx, renvoArmRegRcx)
}

func renvoArmAsmSyscall(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegSys, renvoArmRegRax)
	renvoArmAsmMovRegReg(a, renvoArmRegTmp, renvoArmRegRdx)
	renvoArmAsmMovRegReg(a, 0, renvoArmRegRdi)
	renvoArmAsmMovRegReg(a, 1, renvoArmRegRsi)
	renvoArmAsmMovRegReg(a, 2, renvoArmRegTmp)
	renvoArmAsmMovRegReg(a, 3, renvoArmRegR10)
	renvoArmAsmMovRegReg(a, 4, renvoArmRegR10)
	renvoArmAsmMovRegImm(a, 5, 0)
	renvoArmAsmEmit(a, 0xef000000)
}

func renvoArmAsmPopRdi(a *renvoAsm) {
	renvoArmAsmPopReg(a, renvoArmRegRdi)
}

func renvoArmAsmPopRsi(a *renvoAsm) {
	renvoArmAsmPopReg(a, renvoArmRegRsi)
}

func renvoArmAsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	if base == 0x8948 && disp8 == 0x45 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRax, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x55 {
		renvoArmAsmStoreRegStack(a, renvoArmRegRdx, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x4d {
		renvoArmAsmStoreRegStack(a, renvoArmRegRcx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x45 {
		renvoArmAsmLoadRegStack(a, renvoArmRegRax, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x55 {
		renvoArmAsmLoadRegStack(a, renvoArmRegRdx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x4d {
		renvoArmAsmLoadRegStack(a, renvoArmRegRcx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x45 {
		renvoArmAsmLeaRegStack(a, renvoArmRegRax, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x55 {
		renvoArmAsmLeaRegStack(a, renvoArmRegRdx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x7d {
		renvoArmAsmLeaRegStack(a, renvoArmRegRdi, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x75 {
		renvoArmAsmLeaRegStack(a, renvoArmRegRsi, offset)
		return
	}
}

func renvoArmAsmAddRdxImm(a *renvoAsm, imm int) {
	renvoArmAsmAddRegImm(a, renvoArmRegRdx, renvoArmRegRdx, imm)
}

func renvoArmAsmMemDisp(a *renvoAsm, disp int, op int, disp8 int, disp32 int) {
	if op == 0x8b48 && disp8 == 0x4a {
		renvoArmAsmLoadRegMem(a, renvoArmRegRcx, renvoArmRegRdx, disp, 4)
		return
	}
	if op == 0x8b48 && disp8 == 0x52 {
		renvoArmAsmLoadRegMem(a, renvoArmRegRdx, renvoArmRegRdx, disp, 4)
		return
	}
	if op == 0x8948 && disp8 == 0x41 {
		renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegRcx, disp, 4)
		return
	}
}

func renvoArmAsmLoadQwordRaxIndexRcx8(a *renvoAsm) {
	renvoArmAsmAddRegRegShift(a, renvoArmRegAddr, renvoArmRegRax, renvoArmRegRcx, 3)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 4)
}

func renvoArmAsmLoadQwordRaxIndexRcxDisp(a *renvoAsm, disp int) {
	renvoArmAsmAddRegReg(a, renvoArmRegAddr, renvoArmRegRax, renvoArmRegRcx)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegAddr, disp, 4)
}

func renvoArmAsmLoadRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegRdx, disp, 4)
}

func renvoArmAsmLoadRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegRdx, disp, size)
}

func renvoArmAsmLoadByteRaxIndexRcx(a *renvoAsm) {
	renvoArmAsmAddRegReg(a, renvoArmRegAddr, renvoArmRegRax, renvoArmRegRcx)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 1)
}

func renvoArmAsmLoadRaxIndexRcxSize(a *renvoAsm, size int) {
	shift := 3
	if size == 1 {
		shift = 0
	}
	if size == 2 {
		shift = 1
	}
	if size == 4 {
		shift = 2
	}
	renvoArmAsmAddRegRegShift(a, renvoArmRegAddr, renvoArmRegRax, renvoArmRegRcx, shift)
	renvoArmAsmLoadRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, size)
}

func renvoArmAsmStoreRaxMemRdxRcx8(a *renvoAsm) {
	renvoArmAsmAddRegRegShift(a, renvoArmRegAddr, renvoArmRegRdx, renvoArmRegRcx, 3)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 4)
}

func renvoArmAsmStoreRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegRdx, disp, 4)
}

func renvoArmAsmStoreRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegRdx, disp, size)
}

func renvoArmAsmNormalizeRaxForKind(a *renvoAsm, kind int) {
	if kind == renvoTypeByte {
		renvoArmAsmEmit(a, 0xe6ef0070)
		return
	}
	if kind == renvoTypeInt8 {
		renvoArmAsmEmit(a, 0xe6af0070)
		return
	}
	if kind == renvoTypeInt16 {
		renvoArmAsmEmit(a, 0xe6bf0070)
		return
	}
	if kind == renvoTypeUint16 {
		renvoArmAsmEmit(a, 0xe6ff0070)
	}
}

func renvoArmAsmIncMemRdx(a *renvoAsm) {
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegRdx, 0, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegTmp, renvoArmRegTmp, 1)
	renvoArmAsmStoreRegMem(a, renvoArmRegTmp, renvoArmRegRdx, 0, 4)
}

func renvoArmAsmDecMemRdx(a *renvoAsm) {
	renvoArmAsmLoadRegMem(a, renvoArmRegTmp, renvoArmRegRdx, 0, 4)
	renvoArmAsmAddRegImm(a, renvoArmRegTmp, renvoArmRegTmp, -1)
	renvoArmAsmStoreRegMem(a, renvoArmRegTmp, renvoArmRegRdx, 0, 4)
}

func renvoArmAsmBoolNotRax(a *renvoAsm) {
	renvoArmAsmCmpRaxImm8(a, 0)
	renvoArmAsmCsetRax(a, 0)
}

func renvoArmAsmNegRax(a *renvoAsm) {
	renvoArmAsmEmit(a, 0xe2600000)
}

func renvoArmAsmCmpRaxImm8(a *renvoAsm, imm int) {
	renvoArmAsmCmpRegImm(a, renvoArmRegRax, imm)
}

func renvoArmAsmCmpRegImm(a *renvoAsm, reg int, imm int) {
	if imm >= 0 && imm <= 255 {
		renvoArmAsmEmit(a, 0xe3500000|(reg<<16)|imm)
		return
	}
	tmp := renvoArmRegTmp
	if reg == tmp {
		tmp = renvoArmRegTmp2
	}
	renvoArmAsmMovRegImm(a, tmp, imm)
	renvoArmAsmCmpRegReg(a, reg, tmp)
}

func renvoArmAsmCmpRegReg(a *renvoAsm, left int, right int) {
	renvoArmAsmEmit(a, 0xe1500000|(left<<16)|right)
}

func renvoArmAsmCsetRax(a *renvoAsm, cond int) {
	renvoArmAsmEmit(a, 0xe3a00000)
	renvoArmAsmEmit(a, (cond<<28)|0x03a00001)
}

func renvoArmCondFromSetcc(setcc int) int {
	if setcc == 0x94 {
		return 0
	}
	if setcc == 0x95 {
		return 1
	}
	if setcc == 0x9c {
		return 11
	}
	if setcc == 0x9e {
		return 13
	}
	if setcc == 0x9f {
		return 12
	}
	return 10
}

func renvoArmAsmCmpRcxRaxSet(a *renvoAsm, setcc int) {
	renvoArmAsmCmpRegReg(a, renvoArmRegRcx, renvoArmRegRax)
	cond := renvoArmCondFromSetcc(setcc)
	renvoArmAsmCsetRax(a, cond)
}

func renvoArmAsmAddRaxRcx(a *renvoAsm) {
	renvoArmAsmAddRegReg(a, renvoArmRegRax, renvoArmRegRax, renvoArmRegRcx)
}

func renvoArmAsmSubRaxRcx(a *renvoAsm) {
	renvoArmAsmSubRegReg(a, renvoArmRegRax, renvoArmRegRax, renvoArmRegRcx)
}

func renvoArmAsmShlRcxImm(a *renvoAsm, imm int) {
	renvoArmAsmEmit(a, 0xe1a00000|(renvoArmRegRcx<<12)|(imm<<7)|renvoArmRegRcx)
}

func renvoArmAsmShlRaxImm(a *renvoAsm, imm int) {
	renvoArmAsmEmit(a, 0xe1a00000|(renvoArmRegRax<<12)|(imm<<7)|renvoArmRegRax)
}

func renvoArmAsmSarRaxImm(a *renvoAsm, imm int) {
	renvoArmAsmEmit(a, 0xe1a00040|(renvoArmRegRax<<12)|(imm<<7)|renvoArmRegRax)
}

func renvoArmAsmDivLeftRcxRightRax(a *renvoAsm, mod bool) {
	renvoArmAsmMovRegReg(a, renvoArmRegTmp, renvoArmRegRax)
	renvoArmAsmEmit(a, 0xe710f010|(renvoArmRegRax<<16)|(renvoArmRegRax<<8)|renvoArmRegRcx)
	if mod {
		renvoArmAsmEmit(a, 0xe0600090|(renvoArmRegRcx<<12)|(renvoArmRegTmp<<8)|renvoArmRegRax)
	}
}

func renvoArmAsmPushRax(a *renvoAsm) {
	renvoArmAsmPushReg(a, renvoArmRegRax)
}

func renvoArmAsmPushRcx(a *renvoAsm) {
	renvoArmAsmPushReg(a, renvoArmRegRcx)
}

func renvoArmAsmPushRdx(a *renvoAsm) {
	renvoArmAsmPushReg(a, renvoArmRegRdx)
}

func renvoArmAsmPopRax(a *renvoAsm) {
	renvoArmAsmPopReg(a, renvoArmRegRax)
}

func renvoArmAsmPopRcx(a *renvoAsm) {
	renvoArmAsmPopReg(a, renvoArmRegRcx)
}

func renvoArmAsmPopRdx(a *renvoAsm) {
	renvoArmAsmPopReg(a, renvoArmRegRdx)
}

func renvoArmAsmPushImm(a *renvoAsm, imm int) {
	renvoArmAsmMovRegImm(a, renvoArmRegTmp, imm)
	renvoArmAsmPushReg(a, renvoArmRegTmp)
}

func renvoArmAsmStoreSliceStack(a *renvoAsm, offset int) {
	renvoArmAsmStoreRegStack(a, renvoArmRegRax, offset)
	renvoArmAsmStoreRegStack(a, renvoArmRegRdx, offset-8)
	renvoArmAsmStoreRegStack(a, renvoArmRegRcx, offset-16)
}

func renvoArmAsmStoreAlMemRdxRcx1(a *renvoAsm) {
	renvoArmAsmAddRegReg(a, renvoArmRegAddr, renvoArmRegRdx, renvoArmRegRcx)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, 1)
}

func renvoArmAsmStoreRaxMemRdxRcxSize(a *renvoAsm, size int) {
	shift := 3
	if size == 1 {
		shift = 0
	}
	if size == 2 {
		shift = 1
	}
	if size == 4 {
		shift = 2
	}
	renvoArmAsmAddRegRegShift(a, renvoArmRegAddr, renvoArmRegRdx, renvoArmRegRcx, shift)
	renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegAddr, 0, size)
}

func renvoArmAsmIncRcx(a *renvoAsm) {
	renvoArmAsmAddRegImm(a, renvoArmRegRcx, renvoArmRegRcx, 1)
}

func renvoArmAsmIncRax(a *renvoAsm) {
	renvoArmAsmAddRegImm(a, renvoArmRegRax, renvoArmRegRax, 1)
}

func renvoArmAsmImulRcxImm(a *renvoAsm, imm int) {
	renvoArmAsmMovRegImm(a, renvoArmRegTmp, imm)
	renvoArmAsmMulRegReg(a, renvoArmRegRcx, renvoArmRegRcx, renvoArmRegTmp)
}

func renvoArmAsmLeave(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegSp, renvoArmRegFp)
	renvoArmAsmEmit(a, 0xe8bd4800)
}

func renvoArmAsmRet(a *renvoAsm) {
	renvoArmAsmEmit(a, 0xe12fff1e)
}

func renvoArmAsmCallLabel(a *renvoAsm, label int) {
	at := len(a.code)
	renvoArmAsmEmit(a, 0xeb000000)
	renvoAsmAddReloc(a, at, label)
}

func renvoArmAsmJmpLabel(a *renvoAsm, label int) {
	at := len(a.code)
	renvoArmAsmEmit(a, 0xea000000)
	renvoAsmAddReloc(a, at, label)
}

func renvoArmAsmBCondLabel(a *renvoAsm, label int, cond int) {
	at := len(a.code)
	renvoArmAsmEmit(a, (cond<<28)|0x0a000000)
	renvoAsmAddReloc(a, at, label)
}

func renvoArmAsmJzLabel(a *renvoAsm, label int) {
	renvoArmAsmBCondLabel(a, label, 0)
}

func renvoArmAsmJnzLabel(a *renvoAsm, label int) {
	renvoArmAsmBCondLabel(a, label, 1)
}
