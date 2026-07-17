package main

const rtgArmRegRax = 0
const rtgArmRegRdx = 1
const rtgArmRegRcx = 2
const rtgArmRegRdi = 3
const rtgArmRegRsi = 4
const rtgArmRegR8 = 5
const rtgArmRegR9 = 6
const rtgArmRegSys = 7
const rtgArmRegR10 = 8
const rtgArmRegTmp = 9
const rtgArmRegTmp2 = 10
const rtgArmRegFp = 11
const rtgArmRegAddr = 12
const rtgArmRegSp = 13
const rtgArmRegLr = 14

func rtgArmEmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
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
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	locals = make([]rtgLocalInfo, rtgFunctionLocalCap(fn))
	gotoLabels = make([]rtgGlobalInfo, 0, 0)
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
	rtgArmAsmAlign(a)
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	rtgArmAsmEmit(a, 0xe92d4800)
	rtgArmAsmMovRegReg(a, rtgArmRegFp, rtgArmRegSp)
	rtgArmAsmAddRegImm(a, rtgArmRegSp, rtgArmRegSp, -32768)
	if rtgTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgArmAsmStoreRegStack(a, rtgArmRegRdi, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgBindClosureCaptures(g, fnInfoIndex) {
		return false
	}
	if !rtgBindNamedResults(g, fnInfoIndex) {
		return false
	}
	if !rtgPrepareFunctionControl(g) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if g.deferReturnLabel > 0 {
		if !g.lastRangeReturns {
			rtgAsmJmpLabel(a, g.deferReturnLabel)
		}
		if !rtgEmitFunctionControlEpilogue(g) {
			return false
		}
	} else if !g.lastRangeReturns {
		if !rtgEmitClosureCaptureWriteback(g) {
			return false
		}
		rtgAsmPrimaryImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
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
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func rtgArmStoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRdi, offset)
		return true
	}
	if reg == 1 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRsi, offset)
		return true
	}
	if reg == 2 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRdx, offset)
		return true
	}
	if reg == 3 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRcx, offset)
		return true
	}
	if reg == 4 {
		rtgArmAsmStoreRegStack(a, rtgArmRegR8, offset)
		return true
	}
	if reg == 5 {
		rtgArmAsmStoreRegStack(a, rtgArmRegR9, offset)
		return true
	}
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegFp, 8+(reg-6)*4, 4)
	rtgArmAsmStoreRegStack(a, rtgArmRegRax, offset)
	return true
}

func rtgArmEmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgArmAsmPopReg(a, rtgArmRegRdi)
	}
	if wordCount > 1 {
		rtgArmAsmPopReg(a, rtgArmRegRsi)
	}
	if wordCount > 2 {
		rtgArmAsmPopReg(a, rtgArmRegRdx)
	}
	if wordCount > 3 {
		rtgArmAsmPopReg(a, rtgArmRegRcx)
	}
	if wordCount > 4 {
		rtgArmAsmPopReg(a, rtgArmRegR8)
	}
	if wordCount > 5 {
		rtgArmAsmPopReg(a, rtgArmRegR9)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		rtgArmAsmAddRegImm(a, rtgArmRegSp, rtgArmRegSp, (wordCount-6)*4)
	}
}

func rtgArmEmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= rtgTokCount(p) {
		return false
	}
	start := rtgTokStart(p, tok)
	end := rtgTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		rtgArmAsmAddRegReg(a, rtgArmRegRax, rtgArmRegRcx, rtgArmRegRax)
		return true
	}
	if c0 == '-' {
		rtgArmAsmSubRegReg(a, rtgArmRegRax, rtgArmRegRcx, rtgArmRegRax)
		return true
	}
	if c0 == '*' {
		rtgArmAsmMulRegReg(a, rtgArmRegRax, rtgArmRegRcx, rtgArmRegRax)
		return true
	}
	if c0 == '/' {
		rtgArmAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgArmAsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgArmAsmEmit(a, 0xe1e00000|(rtgArmRegRax<<12)|rtgArmRegRax)
			rtgArmAsmEmit(a, 0xe0000000|(rtgArmRegRcx<<16)|(rtgArmRegRax<<12)|rtgArmRegRax)
		} else {
			rtgArmAsmEmit(a, 0xe0000000|(rtgArmRegRcx<<16)|(rtgArmRegRax<<12)|rtgArmRegRax)
		}
		return true
	}
	if c0 == '|' {
		rtgArmAsmEmit(a, 0xe1800000|(rtgArmRegRcx<<16)|(rtgArmRegRax<<12)|rtgArmRegRax)
		return true
	}
	if c0 == '^' {
		rtgArmAsmEmit(a, 0xe0200000|(rtgArmRegRcx<<16)|(rtgArmRegRax<<12)|rtgArmRegRax)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgArmAsmEmit(a, 0xe1a00010|(rtgArmRegRax<<8)|(rtgArmRegRax<<12)|rtgArmRegRcx)
		} else if c1 == '=' {
			rtgArmAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgArmAsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgArmAsmEmit(a, 0xe1a00050|(rtgArmRegRax<<8)|(rtgArmRegRax<<12)|rtgArmRegRcx)
		} else if c1 == '=' {
			rtgArmAsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgArmAsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgArmAsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgArmAsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtgArmEmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if rtgTokCharIs(p, e.tok, '*') {
		if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgArmAsmMulRegReg(a, rtgArmRegRax, rtgArmRegRcx, rtgArmRegRax)
		rtgAsmSarPrimaryImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
			return false
		}
		rtgAsmShlPrimaryImm(a, 2)
		rtgAsmPushPrimary(a)
		if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
		return false
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
		return false
	}
	rtgAsmPopTertiary(a)
	return rtgEmitPrimaryTertiaryOp(g, e.tok)
}

func rtgArmEmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !rtgEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		rtgArmAsmMovRegReg(a, rtgArmRegRdi, rtgArmRegRdx)
		rtgArmAsmAddRegImm(a, rtgArmRegRsi, rtgArmRegRdx, 8)
		return true
	}
	if loc.global {
		rtgArmAsmMovRegAbs(a, rtgArmRegRdi, loc.offset, rtgAbsBssReloc)
		rtgArmAsmMovRegAbs(a, rtgArmRegRsi, loc.offset+8, rtgAbsBssReloc)
		return true
	}
	rtgArmAsmLeaRegStack(a, rtgArmRegRdi, loc.offset)
	rtgArmAsmLeaRegStack(a, rtgArmRegRsi, loc.offset-8)
	return true
}

func rtgArmEnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := rtgEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.appendAddrLabel)
	noGrowLabel := rtgAsmNewLabel(a)
	capNonZeroLabel := rtgAsmNewLabel(a)
	capReadyLabel := rtgAsmNewLabel(a)
	copyLoopLabel := rtgAsmNewLabel(a)
	copyDoneLabel := rtgAsmNewLabel(a)
	rtgArmAsmLoadRegMem(a, rtgArmRegR8, rtgArmRegRsi, 0, 4)
	rtgArmAsmLoadRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 8, 4)
	rtgArmAsmCmpRegReg(a, rtgArmRegR8, rtgArmRegRcx)
	rtgArmAsmBCondLabel(a, noGrowLabel, 11)
	rtgArmAsmMovRegReg(a, rtgArmRegR9, rtgArmRegRdx)
	rtgArmAsmMovRegReg(a, rtgArmRegR10, rtgArmRegRdi)
	rtgArmAsmCmpRegImm(a, rtgArmRegRcx, 0)
	rtgArmAsmBCondLabel(a, capNonZeroLabel, 1)
	rtgArmAsmMovRegImm(a, rtgArmRegRcx, 16)
	rtgAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	rtgArmAsmAddRegReg(a, rtgArmRegRcx, rtgArmRegRcx, rtgArmRegR8)
	rtgAsmMarkLabel(a, capReadyLabel)
	rtgArmAsmMulRegReg(a, rtgArmRegTmp, rtgArmRegRcx, rtgArmRegR9)
	rtgAsmPushTertiary(a)
	rtgArmAsmMovRegReg(a, rtgArmRegRax, rtgArmRegTmp)
	rtgArmAsmPushReg(a, rtgArmRegLr)
	rtgAsmCallLabel(a, arenaAllocLabel)
	rtgArmAsmPopReg(a, rtgArmRegLr)
	rtgAsmPopTertiary(a)
	rtgArmAsmMovRegReg(a, rtgArmRegRdx, rtgArmRegRax)
	rtgArmAsmMovRegReg(a, rtgArmRegRdi, rtgArmRegRdx)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp2, rtgArmRegR10, 0, 4)
	rtgArmAsmMulRegReg(a, rtgArmRegTmp, rtgArmRegR8, rtgArmRegR9)
	rtgAsmMarkLabel(a, copyLoopLabel)
	rtgArmAsmCmpRegImm(a, rtgArmRegTmp, 0)
	rtgArmAsmBCondLabel(a, copyDoneLabel, 0)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegTmp2, 0, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegRdi, 0, 1)
	rtgArmAsmAddRegSmallImm(a, rtgArmRegTmp2, rtgArmRegTmp2, 1)
	rtgArmAsmAddRegSmallImm(a, rtgArmRegRdi, rtgArmRegRdi, 1)
	rtgArmAsmAddRegSmallImm(a, rtgArmRegTmp, rtgArmRegTmp, -1)
	rtgAsmJmpMarkLabel(a, copyLoopLabel, copyDoneLabel)
	rtgArmAsmStoreRegMem(a, rtgArmRegRdx, rtgArmRegR10, 0, 4)
	rtgArmAsmStoreRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 8, 4)
	rtgArmAsmMulRegReg(a, rtgArmRegTmp, rtgArmRegR8, rtgArmRegR9)
	rtgArmAsmAddRegReg(a, rtgArmRegRax, rtgArmRegRdx, rtgArmRegTmp)
	rtgArmAsmAddRegImm(a, rtgArmRegR8, rtgArmRegR8, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegR8, rtgArmRegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, noGrowLabel)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegRdi, 0, 4)
	rtgArmAsmMulRegReg(a, rtgArmRegTmp, rtgArmRegR8, rtgArmRegRdx)
	rtgArmAsmAddRegReg(a, rtgArmRegRax, rtgArmRegRax, rtgArmRegTmp)
	rtgArmAsmAddRegImm(a, rtgArmRegR8, rtgArmRegR8, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegR8, rtgArmRegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgArmEnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	rtgArmAsmLoadRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 0, 4)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegRdi, 0, 4)
	rtgArmAsmAddRegReg(a, rtgArmRegTmp, rtgArmRegTmp, rtgArmRegRcx)
	rtgArmAsmStoreRegMem(a, rtgArmRegRdx, rtgArmRegTmp, 0, 1)
	rtgArmAsmAddRegImm(a, rtgArmRegRcx, rtgArmRegRcx, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtgArmEnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	rtgArmAsmLoadRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 0, 4)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegRdi, 0, 4)
	rtgArmAsmAddRegRegShift(a, rtgArmRegTmp, rtgArmRegTmp, rtgArmRegRcx, 3)
	rtgArmAsmStoreRegMem(a, rtgArmRegRdx, rtgArmRegTmp, 0, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegRcx, rtgArmRegRcx, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegRcx, rtgArmRegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtgArmEnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgArmAsmCmpRegReg(a, rtgArmRegRsi, rtgArmRegRcx)
	rtgArmAsmBCondLabel(a, notEqualLabel, 1)
	rtgArmAsmCmpRegImm(a, rtgArmRegRsi, 0)
	rtgArmAsmBCondLabel(a, equalLabel, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegRdi, 0, 1)
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp2, rtgArmRegRdx, 0, 1)
	rtgArmAsmCmpRegReg(a, rtgArmRegTmp, rtgArmRegTmp2)
	rtgArmAsmBCondLabel(a, notEqualLabel, 1)
	rtgArmAsmAddRegImm(a, rtgArmRegRdi, rtgArmRegRdi, 1)
	rtgArmAsmAddRegImm(a, rtgArmRegRdx, rtgArmRegRdx, 1)
	rtgArmAsmAddRegImm(a, rtgArmRegRsi, rtgArmRegRsi, -1)
	rtgArmAsmCmpRegImm(a, rtgArmRegRsi, 0)
	rtgArmAsmBCondLabel(a, loopLabel, 1)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmPrimaryImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtgArmAsmEmit(a *rtgAsm, insn int) {
	rtgAsmEmit32(a, insn)
}

func rtgArmAsmAlign(a *rtgAsm) {
	for len(a.code)%4 != 0 {
		rtgAsmEmit8(a, 0)
	}
}

func rtgArmAsmMovRegReg(a *rtgAsm, dst int, src int) {
	if dst == src {
		return
	}
	rtgArmAsmEmit(a, 0xe1a00000|(dst<<12)|src)
}

func rtgArmAsmMovRegImm(a *rtgAsm, reg int, imm int) {
	part := imm & 65535
	rtgArmAsmEmit(a, 0xe3000000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	part = (imm >> 16) & 65535
	if part != 0 {
		rtgArmAsmEmit(a, 0xe3400000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	}
}

func rtgArmAsmPatchMovRegImmAt(a *rtgAsm, at int, reg int, imm int) {
	part := imm & 65535
	rtgPut32At(a.code, at, 0xe3000000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
	part = (imm >> 16) & 65535
	rtgPut32At(a.code, at+4, 0xe3400000|((part&0xf000)<<4)|(reg<<12)|(part&0x0fff))
}

func rtgArmAsmMovRegAbs(a *rtgAsm, reg int, off int, kind int) {
	at := len(a.code)
	rtgArmAsmMovRegImm(a, reg, 0)
	rtgArmAsmEmit(a, 0xe3400000|(reg<<12))
	rtgAsmAddAbsReloc(a, at, off, kind)
}

func rtgArmAsmAddRegImm(a *rtgAsm, dst int, src int, imm int) {
	if imm == 0 {
		rtgArmAsmMovRegReg(a, dst, src)
		return
	}
	tmp := rtgArmRegTmp
	if dst == tmp || src == tmp {
		tmp = rtgArmRegTmp2
	}
	if imm < 0 {
		rtgArmAsmMovRegImm(a, tmp, -imm)
		rtgArmAsmSubRegReg(a, dst, src, tmp)
		return
	}
	rtgArmAsmMovRegImm(a, tmp, imm)
	rtgArmAsmAddRegReg(a, dst, src, tmp)
}

func rtgArmAsmAddRegSmallImm(a *rtgAsm, dst int, src int, imm int) {
	if imm < 0 {
		rtgArmAsmEmit(a, 0xe2400000|(src<<16)|(dst<<12)|(-imm))
		return
	}
	rtgArmAsmEmit(a, 0xe2800000|(src<<16)|(dst<<12)|imm)
}

func rtgArmAsmAddRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgArmAsmEmit(a, 0xe0800000|(left<<16)|(dst<<12)|right)
}

func rtgArmAsmSubRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgArmAsmEmit(a, 0xe0400000|(left<<16)|(dst<<12)|right)
}

func rtgArmAsmAddRegRegShift(a *rtgAsm, dst int, left int, right int, shift int) {
	rtgArmAsmEmit(a, 0xe0800000|(left<<16)|(dst<<12)|(shift<<7)|right)
}

func rtgArmAsmMulRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgArmAsmEmit(a, 0xe0000090|(dst<<16)|(right<<8)|left)
}

func rtgArmAsmAddr(a *rtgAsm, base int, disp int) int {
	if disp == 0 {
		return base
	}
	rtgArmAsmAddRegImm(a, rtgArmRegAddr, base, disp)
	return rtgArmRegAddr
}

func rtgArmAsmLoadStoreAddr(a *rtgAsm, base int, disp int, size int) int {
	if size == 2 && (disp < 0 || disp > 255) {
		return rtgArmAsmAddr(a, base, disp)
	}
	if size != 2 && (disp < -4095 || disp > 4095) {
		return rtgArmAsmAddr(a, base, disp)
	}
	return base
}

func rtgArmAsmLoadRegMem(a *rtgAsm, dst int, base int, disp int, size int) {
	addr := rtgArmAsmLoadStoreAddr(a, base, disp, size)
	if addr != base {
		disp = 0
	}
	up := 0x00800000
	if disp < 0 {
		up = 0
		disp = -disp
	}
	if size == 1 {
		rtgArmAsmEmit(a, 0xe5500000|up|(addr<<16)|(dst<<12)|disp)
		return
	}
	if size == 2 {
		rtgArmAsmEmit(a, 0xe15000f0|up|(addr<<16)|(dst<<12)|((disp&0xf0)<<4)|(disp&0x0f))
		return
	}
	rtgArmAsmEmit(a, 0xe5100000|up|(addr<<16)|(dst<<12)|disp)
}

func rtgArmAsmStoreRegMem(a *rtgAsm, src int, base int, disp int, size int) {
	addr := rtgArmAsmLoadStoreAddr(a, base, disp, size)
	if addr != base {
		disp = 0
	}
	up := 0x00800000
	if disp < 0 {
		up = 0
		disp = -disp
	}
	if size == 1 {
		rtgArmAsmEmit(a, 0xe5400000|up|(addr<<16)|(src<<12)|disp)
		return
	}
	if size == 2 {
		rtgArmAsmEmit(a, 0xe14000b0|up|(addr<<16)|(src<<12)|((disp&0xf0)<<4)|(disp&0x0f))
		return
	}
	rtgArmAsmEmit(a, 0xe5000000|up|(addr<<16)|(src<<12)|disp)
}

func rtgArmAsmLoadRegStack(a *rtgAsm, dst int, offset int) {
	rtgArmAsmLoadRegMem(a, dst, rtgArmRegFp, -offset, 4)
}

func rtgArmAsmStoreRegStack(a *rtgAsm, src int, offset int) {
	rtgArmAsmStoreRegMem(a, src, rtgArmRegFp, -offset, 4)
}

func rtgArmAsmLeaRegStack(a *rtgAsm, dst int, offset int) {
	rtgArmAsmAddRegImm(a, dst, rtgArmRegFp, -offset)
}

func rtgArmAsmPushReg(a *rtgAsm, reg int) {
	rtgArmAsmEmit(a, 0xe52d0004|(reg<<12))
}

func rtgArmAsmPopReg(a *rtgAsm, reg int) {
	rtgArmAsmEmit(a, 0xe49d0004|(reg<<12))
}

func rtgArmAsmMovRaxImm(a *rtgAsm, imm int) {
	rtgArmAsmMovRegImm(a, rtgArmRegRax, imm)
}

func rtgArmAsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgArmAsmMovRaxImm(a, imm)
}

func rtgArmAsmMovRdxImm(a *rtgAsm, imm int) {
	rtgArmAsmMovRegImm(a, rtgArmRegRdx, imm)
}

func rtgArmAsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgArmAsmMovRegAbs(a, rtgArmRegRax, dataOff, 0)
}

func rtgArmAsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgArmAsmMovRegAbs(a, rtgArmRegRax, bssOff, rtgAbsBssReloc)
}

func rtgArmAsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgArmAsmMovRegAbs(a, rtgArmRegR10, bssOff, rtgAbsBssReloc)
}

func rtgArmAsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgArmAsmMovRegAbs(a, rtgArmRegAddr, bssOff, rtgAbsBssReloc)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 4)
}

func rtgArmAsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgArmAsmMovRegAbs(a, rtgArmRegAddr, bssOff, rtgAbsBssReloc)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 4)
}

func rtgArmAsmMovRdiRax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRdi, rtgArmRegRax)
}

func rtgArmAsmMovRaxRdx(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRax, rtgArmRegRdx)
}

func rtgArmAsmMovRdxRax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRdx, rtgArmRegRax)
}

func rtgArmAsmMovRcxRax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRcx, rtgArmRegRax)
}

func rtgArmAsmMovRcxRdx(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRcx, rtgArmRegRdx)
}

func rtgArmAsmMovRsiRax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegRsi, rtgArmRegRax)
}

func rtgArmAsmMovR8Rax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegR8, rtgArmRegRax)
}

func rtgArmAsmMovR9Rax(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegR9, rtgArmRegRax)
}

func rtgArmAsmAddRdxRcx(a *rtgAsm) {
	rtgArmAsmAddRegReg(a, rtgArmRegRdx, rtgArmRegRdx, rtgArmRegRcx)
}

func rtgArmAsmSyscall(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegSys, rtgArmRegRax)
	rtgArmAsmMovRegReg(a, rtgArmRegTmp, rtgArmRegRdx)
	rtgArmAsmMovRegReg(a, 0, rtgArmRegRdi)
	rtgArmAsmMovRegReg(a, 1, rtgArmRegRsi)
	rtgArmAsmMovRegReg(a, 2, rtgArmRegTmp)
	rtgArmAsmMovRegReg(a, 3, rtgArmRegR10)
	rtgArmAsmMovRegReg(a, 4, rtgArmRegR10)
	rtgArmAsmMovRegImm(a, 5, 0)
	rtgArmAsmEmit(a, 0xef000000)
}

func rtgArmAsmPopRdi(a *rtgAsm) {
	rtgArmAsmPopReg(a, rtgArmRegRdi)
}

func rtgArmAsmPopRsi(a *rtgAsm) {
	rtgArmAsmPopReg(a, rtgArmRegRsi)
}

func rtgArmAsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	if base == 0x8948 && disp8 == 0x45 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRax, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x55 {
		rtgArmAsmStoreRegStack(a, rtgArmRegRdx, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x4d {
		rtgArmAsmStoreRegStack(a, rtgArmRegRcx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x45 {
		rtgArmAsmLoadRegStack(a, rtgArmRegRax, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x55 {
		rtgArmAsmLoadRegStack(a, rtgArmRegRdx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x4d {
		rtgArmAsmLoadRegStack(a, rtgArmRegRcx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x45 {
		rtgArmAsmLeaRegStack(a, rtgArmRegRax, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x55 {
		rtgArmAsmLeaRegStack(a, rtgArmRegRdx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x7d {
		rtgArmAsmLeaRegStack(a, rtgArmRegRdi, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x75 {
		rtgArmAsmLeaRegStack(a, rtgArmRegRsi, offset)
		return
	}
}

func rtgArmAsmAddRdxImm(a *rtgAsm, imm int) {
	rtgArmAsmAddRegImm(a, rtgArmRegRdx, rtgArmRegRdx, imm)
}

func rtgArmAsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	if op == 0x8b48 && disp8 == 0x4a {
		rtgArmAsmLoadRegMem(a, rtgArmRegRcx, rtgArmRegRdx, disp, 4)
		return
	}
	if op == 0x8b48 && disp8 == 0x52 {
		rtgArmAsmLoadRegMem(a, rtgArmRegRdx, rtgArmRegRdx, disp, 4)
		return
	}
	if op == 0x8948 && disp8 == 0x41 {
		rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegRcx, disp, 4)
		return
	}
}

func rtgArmAsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgArmAsmAddRegRegShift(a, rtgArmRegAddr, rtgArmRegRax, rtgArmRegRcx, 3)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 4)
}

func rtgArmAsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgArmAsmAddRegReg(a, rtgArmRegAddr, rtgArmRegRax, rtgArmRegRcx)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegAddr, disp, 4)
}

func rtgArmAsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegRdx, disp, 4)
}

func rtgArmAsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegRdx, disp, size)
}

func rtgArmAsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgArmAsmAddRegReg(a, rtgArmRegAddr, rtgArmRegRax, rtgArmRegRcx)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 1)
}

func rtgArmAsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
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
	rtgArmAsmAddRegRegShift(a, rtgArmRegAddr, rtgArmRegRax, rtgArmRegRcx, shift)
	rtgArmAsmLoadRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, size)
}

func rtgArmAsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgArmAsmAddRegRegShift(a, rtgArmRegAddr, rtgArmRegRdx, rtgArmRegRcx, 3)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 4)
}

func rtgArmAsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegRdx, disp, 4)
}

func rtgArmAsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegRdx, disp, size)
}

func rtgArmAsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte {
		rtgArmAsmEmit(a, 0xe6ef0070)
		return
	}
	if kind == rtgTypeInt8 {
		rtgArmAsmEmit(a, 0xe6af0070)
		return
	}
	if kind == rtgTypeInt16 {
		rtgArmAsmEmit(a, 0xe6bf0070)
		return
	}
	if kind == rtgTypeUint16 {
		rtgArmAsmEmit(a, 0xe6ff0070)
	}
}

func rtgArmAsmIncMemRdx(a *rtgAsm) {
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegRdx, 0, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegTmp, rtgArmRegTmp, 1)
	rtgArmAsmStoreRegMem(a, rtgArmRegTmp, rtgArmRegRdx, 0, 4)
}

func rtgArmAsmDecMemRdx(a *rtgAsm) {
	rtgArmAsmLoadRegMem(a, rtgArmRegTmp, rtgArmRegRdx, 0, 4)
	rtgArmAsmAddRegImm(a, rtgArmRegTmp, rtgArmRegTmp, -1)
	rtgArmAsmStoreRegMem(a, rtgArmRegTmp, rtgArmRegRdx, 0, 4)
}

func rtgArmAsmBoolNotRax(a *rtgAsm) {
	rtgArmAsmCmpRaxImm8(a, 0)
	rtgArmAsmCsetRax(a, 0)
}

func rtgArmAsmNegRax(a *rtgAsm) {
	rtgArmAsmEmit(a, 0xe2600000)
}

func rtgArmAsmCmpRaxImm8(a *rtgAsm, imm int) {
	rtgArmAsmCmpRegImm(a, rtgArmRegRax, imm)
}

func rtgArmAsmCmpRegImm(a *rtgAsm, reg int, imm int) {
	if imm >= 0 && imm <= 255 {
		rtgArmAsmEmit(a, 0xe3500000|(reg<<16)|imm)
		return
	}
	tmp := rtgArmRegTmp
	if reg == tmp {
		tmp = rtgArmRegTmp2
	}
	rtgArmAsmMovRegImm(a, tmp, imm)
	rtgArmAsmCmpRegReg(a, reg, tmp)
}

func rtgArmAsmCmpRegReg(a *rtgAsm, left int, right int) {
	rtgArmAsmEmit(a, 0xe1500000|(left<<16)|right)
}

func rtgArmAsmCsetRax(a *rtgAsm, cond int) {
	rtgArmAsmEmit(a, 0xe3a00000)
	rtgArmAsmEmit(a, (cond<<28)|0x03a00001)
}

func rtgArmCondFromSetcc(setcc int) int {
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

func rtgArmAsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgArmAsmCmpRegReg(a, rtgArmRegRcx, rtgArmRegRax)
	cond := rtgArmCondFromSetcc(setcc)
	rtgArmAsmCsetRax(a, cond)
}

func rtgArmAsmAddRaxRcx(a *rtgAsm) {
	rtgArmAsmAddRegReg(a, rtgArmRegRax, rtgArmRegRax, rtgArmRegRcx)
}

func rtgArmAsmSubRaxRcx(a *rtgAsm) {
	rtgArmAsmSubRegReg(a, rtgArmRegRax, rtgArmRegRax, rtgArmRegRcx)
}

func rtgArmAsmShlRcxImm(a *rtgAsm, imm int) {
	rtgArmAsmEmit(a, 0xe1a00000|(rtgArmRegRcx<<12)|(imm<<7)|rtgArmRegRcx)
}

func rtgArmAsmShlRaxImm(a *rtgAsm, imm int) {
	rtgArmAsmEmit(a, 0xe1a00000|(rtgArmRegRax<<12)|(imm<<7)|rtgArmRegRax)
}

func rtgArmAsmSarRaxImm(a *rtgAsm, imm int) {
	rtgArmAsmEmit(a, 0xe1a00040|(rtgArmRegRax<<12)|(imm<<7)|rtgArmRegRax)
}

func rtgArmAsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgArmAsmMovRegReg(a, rtgArmRegTmp, rtgArmRegRax)
	rtgArmAsmEmit(a, 0xe710f010|(rtgArmRegRax<<16)|(rtgArmRegRax<<8)|rtgArmRegRcx)
	if mod {
		rtgArmAsmEmit(a, 0xe0600090|(rtgArmRegRcx<<12)|(rtgArmRegTmp<<8)|rtgArmRegRax)
	}
}

func rtgArmAsmPushRax(a *rtgAsm) {
	rtgArmAsmPushReg(a, rtgArmRegRax)
}

func rtgArmAsmPushRcx(a *rtgAsm) {
	rtgArmAsmPushReg(a, rtgArmRegRcx)
}

func rtgArmAsmPushRdx(a *rtgAsm) {
	rtgArmAsmPushReg(a, rtgArmRegRdx)
}

func rtgArmAsmPopRax(a *rtgAsm) {
	rtgArmAsmPopReg(a, rtgArmRegRax)
}

func rtgArmAsmPopRcx(a *rtgAsm) {
	rtgArmAsmPopReg(a, rtgArmRegRcx)
}

func rtgArmAsmPopRdx(a *rtgAsm) {
	rtgArmAsmPopReg(a, rtgArmRegRdx)
}

func rtgArmAsmPushImm(a *rtgAsm, imm int) {
	rtgArmAsmMovRegImm(a, rtgArmRegTmp, imm)
	rtgArmAsmPushReg(a, rtgArmRegTmp)
}

func rtgArmAsmStoreSliceStack(a *rtgAsm, offset int) {
	rtgArmAsmStoreRegStack(a, rtgArmRegRax, offset)
	rtgArmAsmStoreRegStack(a, rtgArmRegRdx, offset-8)
	rtgArmAsmStoreRegStack(a, rtgArmRegRcx, offset-16)
}

func rtgArmAsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgArmAsmAddRegReg(a, rtgArmRegAddr, rtgArmRegRdx, rtgArmRegRcx)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, 1)
}

func rtgArmAsmStoreRaxMemRdxRcxSize(a *rtgAsm, size int) {
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
	rtgArmAsmAddRegRegShift(a, rtgArmRegAddr, rtgArmRegRdx, rtgArmRegRcx, shift)
	rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegAddr, 0, size)
}

func rtgArmAsmIncRcx(a *rtgAsm) {
	rtgArmAsmAddRegImm(a, rtgArmRegRcx, rtgArmRegRcx, 1)
}

func rtgArmAsmIncRax(a *rtgAsm) {
	rtgArmAsmAddRegImm(a, rtgArmRegRax, rtgArmRegRax, 1)
}

func rtgArmAsmImulRcxImm(a *rtgAsm, imm int) {
	rtgArmAsmMovRegImm(a, rtgArmRegTmp, imm)
	rtgArmAsmMulRegReg(a, rtgArmRegRcx, rtgArmRegRcx, rtgArmRegTmp)
}

func rtgArmAsmLeave(a *rtgAsm) {
	rtgArmAsmMovRegReg(a, rtgArmRegSp, rtgArmRegFp)
	rtgArmAsmEmit(a, 0xe8bd4800)
}

func rtgArmAsmRet(a *rtgAsm) {
	rtgArmAsmEmit(a, 0xe12fff1e)
}

func rtgArmAsmCallLabel(a *rtgAsm, label int) {
	at := len(a.code)
	rtgArmAsmEmit(a, 0xeb000000)
	rtgAsmAddReloc(a, at, label)
}

func rtgArmAsmJmpLabel(a *rtgAsm, label int) {
	at := len(a.code)
	rtgArmAsmEmit(a, 0xea000000)
	rtgAsmAddReloc(a, at, label)
}

func rtgArmAsmBCondLabel(a *rtgAsm, label int, cond int) {
	at := len(a.code)
	rtgArmAsmEmit(a, (cond<<28)|0x0a000000)
	rtgAsmAddReloc(a, at, label)
}

func rtgArmAsmJzLabel(a *rtgAsm, label int) {
	rtgArmAsmBCondLabel(a, label, 0)
}

func rtgArmAsmJnzLabel(a *rtgAsm, label int) {
	rtgArmAsmBCondLabel(a, label, 1)
}
