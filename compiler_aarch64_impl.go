package main

const rtgAarch64RegRax = 0
const rtgAarch64RegRdx = 1
const rtgAarch64RegRcx = 2
const rtgAarch64RegRdi = 3
const rtgAarch64RegRsi = 4
const rtgAarch64RegR8 = 5
const rtgAarch64RegR9 = 6
const rtgAarch64RegR10 = 7
const rtgAarch64RegSys = 8
const rtgAarch64RegTmp = 9
const rtgAarch64RegTmp2 = 10
const rtgAarch64RegAddr = 12
const rtgAarch64RegFp = 29
const rtgAarch64RegLr = 30
const rtgAarch64RegZr = 31

func rtgAarch64EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldLocalCount := g.localCount
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
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
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAarch64AsmAlign(a)
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	rtgAarch64AsmEmit(a, 0xa9bf7bfd)
	rtgAarch64AsmEmit(a, 0x910003fd)
	rtgAarch64AsmEmit(a, 0xd14023ff)
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRdi, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if !g.lastRangeReturns {
		rtgAsmMovRaxImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	g.locals = oldLocals
	g.localCount = oldLocalCount
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.stackUsed = oldStackUsed
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func rtgAarch64StoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRdi, offset)
		return true
	}
	if reg == 1 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRsi, offset)
		return true
	}
	if reg == 2 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRdx, offset)
		return true
	}
	if reg == 3 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRcx, offset)
		return true
	}
	if reg == 4 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegR8, offset)
		return true
	}
	if reg == 5 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegR9, offset)
		return true
	}
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegFp, 16+(reg-6)*16, 8)
	rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRax, offset)
	return true
}

func rtgAarch64EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegRdi)
	}
	if wordCount > 1 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegRsi)
	}
	if wordCount > 2 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegRdx)
	}
	if wordCount > 3 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegRcx)
	}
	if wordCount > 4 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegR8)
	}
	if wordCount > 5 {
		rtgAarch64AsmPopReg(a, rtgAarch64RegR9)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		rtgAarch64AsmAddRegImm(a, 31, 31, (wordCount-6)*16)
	}
}

func rtgAarch64EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
		rtgAarch64AsmAddRegReg(a, rtgAarch64RegRax, rtgAarch64RegRcx, rtgAarch64RegRax)
		return true
	}
	if c0 == '-' {
		rtgAarch64AsmSubRegReg(a, rtgAarch64RegRax, rtgAarch64RegRcx, rtgAarch64RegRax)
		return true
	}
	if c0 == '*' {
		rtgAarch64AsmEmit(a, 0x9b007c40)
		return true
	}
	if c0 == '/' {
		rtgAarch64AsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgAarch64AsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgAarch64AsmEmit(a, 0x8a200040)
		} else {
			rtgAarch64AsmEmit(a, 0x8a000040)
		}
		return true
	}
	if c0 == '|' {
		rtgAarch64AsmEmit(a, 0xaa000040)
		return true
	}
	if c0 == '^' {
		rtgAarch64AsmEmit(a, 0xca000040)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgAarch64AsmEmit(a, 0x9ac02040)
		} else if c1 == '=' {
			rtgAarch64AsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgAarch64AsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAarch64AsmEmit(a, 0x9ac02840)
		} else if c1 == '=' {
			rtgAarch64AsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgAarch64AsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgAarch64AsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgAarch64AsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtgAarch64EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if rtgTokCharIs(p, e.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAarch64AsmEmit(a, 0x9b007c40)
		rtgAsmSarRaxImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmShlRaxImm(a, 2)
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if !rtgEmitIntExpr(g, ep, e.left) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPopRcx(a)
	return rtgEmitRaxRcxOp(g, e.tok)
}

func rtgAarch64EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	noGrowLabel := rtgAsmNewLabel(a)
	capNonZeroLabel := rtgAsmNewLabel(a)
	capReadyLabel := rtgAsmNewLabel(a)
	heapReadyLabel := rtgAsmNewLabel(a)
	copyLoopLabel := rtgAsmNewLabel(a)
	copyDoneLabel := rtgAsmNewLabel(a)
	rtgStringHeapOffsets(g)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegR8, rtgAarch64RegRsi, 0, 8)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 8, 8)
	rtgAarch64AsmCmpRegReg(a, rtgAarch64RegR8, rtgAarch64RegRcx)
	rtgAarch64AsmBCondLabel(a, noGrowLabel, 11)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR9, rtgAarch64RegRdx)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR10, rtgAarch64RegRdi)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRcx, 0)
	rtgAarch64AsmBCondLabel(a, capNonZeroLabel, 1)
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegRcx, 16)
	rtgAsmJmpLabel(a, capReadyLabel)
	rtgAsmMarkLabel(a, capNonZeroLabel)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRcx, rtgAarch64RegRcx, rtgAarch64RegR8)
	rtgAsmMarkLabel(a, capReadyLabel)
	rtgAarch64AsmMulRegReg(a, rtgAarch64RegTmp, rtgAarch64RegRcx, rtgAarch64RegR9)
	rtgAsmLoadRaxBss(a, g.stringHeapOff)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, heapReadyLabel)
	rtgAsmMovRaxBssAddr(a, g.stringHeapDataOff)
	rtgAsmStoreRaxBss(a, g.stringHeapOff)
	rtgAsmMarkLabel(a, heapReadyLabel)
	rtgAsmLoadRaxBss(a, g.stringHeapOff)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRax)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdi, rtgAarch64RegRdx)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRax, rtgAarch64RegRax, rtgAarch64RegTmp)
	rtgAsmStoreRaxBss(a, g.stringHeapOff)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp2, rtgAarch64RegR10, 0, 8)
	rtgAarch64AsmMulRegReg(a, rtgAarch64RegTmp, rtgAarch64RegR8, rtgAarch64RegR9)
	rtgAsmMarkLabel(a, copyLoopLabel)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegTmp, 0)
	rtgAarch64AsmBCondLabel(a, copyDoneLabel, 0)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegTmp2, 0, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdi, 0, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegTmp2, rtgAarch64RegTmp2, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdi, rtgAarch64RegRdi, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegTmp, rtgAarch64RegTmp, -1)
	rtgAsmJmpLabel(a, copyLoopLabel)
	rtgAsmMarkLabel(a, copyDoneLabel)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRdx, rtgAarch64RegR10, 0, 8)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 8, 8)
	rtgAarch64AsmMulRegReg(a, rtgAarch64RegTmp, rtgAarch64RegR8, rtgAarch64RegR9)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRax, rtgAarch64RegRdx, rtgAarch64RegTmp)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegR8, rtgAarch64RegR8, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegR8, rtgAarch64RegRsi, 0, 8)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, noGrowLabel)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdi, 0, 8)
	rtgAarch64AsmMulRegReg(a, rtgAarch64RegTmp, rtgAarch64RegR8, rtgAarch64RegRdx)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRax, rtgAarch64RegRax, rtgAarch64RegTmp)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegR8, rtgAarch64RegR8, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegR8, rtgAarch64RegRsi, 0, 8)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgAarch64EnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdi, 0, 8)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegTmp, rtgAarch64RegTmp, rtgAarch64RegRcx)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRdx, rtgAarch64RegTmp, 0, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRcx, rtgAarch64RegRcx, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtgAarch64EnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdi, 0, 8)
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegTmp, rtgAarch64RegTmp, rtgAarch64RegRcx, 3)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRdx, rtgAarch64RegTmp, 0, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRcx, rtgAarch64RegRcx, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtgAarch64EnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRdi, rtgAarch64RegRdi, 0, 8)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRdi, rtgAarch64RegRdi, rtgAarch64RegRcx)
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRcx, rtgAarch64RegRcx, rtgAarch64RegRdx)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRsi, 0, 8)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRdx, 0)
	rtgAarch64AsmBCondLabel(a, doneLabel, 0)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRax, 0, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdi, 0, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRax, rtgAarch64RegRax, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdi, rtgAarch64RegRdi, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdx, rtgAarch64RegRdx, -1)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

func rtgAarch64EnsureCopyWordsHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.copyWordsEmitted {
		return g.copyWordsLabel
	}
	g.copyWordsEmitted = true
	g.copyWordsLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.copyWordsLabel)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRdx, 0)
	rtgAarch64AsmBCondLabel(a, doneLabel, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegRsi, 0, 8)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdi, 0, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRsi, rtgAarch64RegRsi, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdi, rtgAarch64RegRdi, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdx, rtgAarch64RegRdx, -1)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRdx, 0)
	rtgAarch64AsmBCondLabel(a, loopLabel, 1)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}

func rtgAarch64EnsureStringEqualHelper(g *rtgLinearGen) int {
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
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.streqLabel)
	rtgAsmMovRaxImm(a, 0)
	rtgAarch64AsmCmpRegReg(a, rtgAarch64RegRsi, rtgAarch64RegRcx)
	rtgAarch64AsmBCondLabel(a, notEqualLabel, 1)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRsi, 0)
	rtgAarch64AsmBCondLabel(a, equalLabel, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdi, 0, 1)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp2, rtgAarch64RegRdx, 0, 1)
	rtgAarch64AsmCmpRegReg(a, rtgAarch64RegTmp, rtgAarch64RegTmp2)
	rtgAarch64AsmBCondLabel(a, notEqualLabel, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdi, rtgAarch64RegRdi, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdx, rtgAarch64RegRdx, 1)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRsi, rtgAarch64RegRsi, -1)
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRsi, 0)
	rtgAarch64AsmBCondLabel(a, loopLabel, 1)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtgAarch64AsmEmit(a *rtgAsm, insn int) {
	rtgAsmEmit32(a, insn)
}

func rtgAarch64AsmAlign(a *rtgAsm) {
	for len(a.code)%4 != 0 {
		rtgAsmEmit8(a, 0)
	}
}

func rtgAarch64AsmMovRegReg(a *rtgAsm, dst int, src int) {
	if dst == src {
		return
	}
	rtgAarch64AsmEmit(a, 0xaa0003e0|(src<<16)|dst)
}

func rtgAarch64AsmMovRegImm(a *rtgAsm, reg int, imm int) {
	if imm < 0 {
		inv := -imm - 1
		part := inv & 65535
		rtgAarch64AsmEmit(a, 0x92800000|(part<<5)|reg)
		for i := 1; i < 4; i++ {
			targetPart := (imm >> (i * 16)) & 65535
			if targetPart != 65535 {
				rtgAarch64AsmEmit(a, 0xf2800000|(i<<21)|(targetPart<<5)|reg)
			}
		}
		return
	}
	part := imm & 65535
	rtgAarch64AsmEmit(a, 0xd2800000|(part<<5)|reg)
	for i := 1; i < 4; i++ {
		part = (imm >> (i * 16)) & 65535
		if part != 0 {
			rtgAarch64AsmEmit(a, 0xf2800000|(i<<21)|(part<<5)|reg)
		}
	}
}

func rtgAarch64AsmPatchMovRegImmAt(a *rtgAsm, at int, reg int, imm int) {
	part := imm & 65535
	rtgPut32At(a.code, at, 0xd2800000|(part<<5)|reg)
	for i := 1; i < 4; i++ {
		part = (imm >> (i * 16)) & 65535
		rtgPut32At(a.code, at+i*4, 0xf2800000|(i<<21)|(part<<5)|reg)
	}
}

func rtgAarch64AsmMovRegAbs(a *rtgAsm, reg int, off int, kind int) {
	at := len(a.code)
	rtgAarch64AsmMovRegImm(a, reg, 0)
	rtgAarch64AsmEmit(a, 0xf2800000|(1<<21)|reg)
	rtgAarch64AsmEmit(a, 0xf2800000|(2<<21)|reg)
	rtgAarch64AsmEmit(a, 0xf2800000|(3<<21)|reg)
	rtgAsmAddAbsReloc(a, at, off, kind)
}

func rtgAarch64AsmAddRegImm(a *rtgAsm, dst int, src int, imm int) {
	if imm == 0 {
		rtgAarch64AsmMovRegReg(a, dst, src)
		return
	}
	op := 0x91000000
	if imm < 0 {
		op = 0xd1000000
		imm = -imm
	}
	cur := src
	for imm > 0 {
		shift := 0
		chunk := imm
		if chunk > 4095 {
			chunk = imm / 4096
			if chunk > 4095 {
				chunk = 4095
			}
			shift = 1
			imm = imm - chunk*4096
		} else {
			imm = 0
		}
		rtgAarch64AsmEmit(a, op|(shift<<22)|(chunk<<10)|(cur<<5)|dst)
		cur = dst
	}
}

func rtgAarch64AsmAddRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgAarch64AsmEmit(a, 0x8b000000|(right<<16)|(left<<5)|dst)
}

func rtgAarch64AsmSubRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgAarch64AsmEmit(a, 0xcb000000|(right<<16)|(left<<5)|dst)
}

func rtgAarch64AsmAddRegRegShift(a *rtgAsm, dst int, left int, right int, shift int) {
	rtgAarch64AsmEmit(a, 0x8b000000|(right<<16)|(shift<<10)|(left<<5)|dst)
}

func rtgAarch64AsmMulRegReg(a *rtgAsm, dst int, left int, right int) {
	rtgAarch64AsmEmit(a, 0x9b007c00|(right<<16)|(left<<5)|dst)
}

func rtgAarch64AsmAddr(a *rtgAsm, base int, disp int) int {
	if disp == 0 {
		return base
	}
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegAddr, base, disp)
	return rtgAarch64RegAddr
}

func rtgAarch64AsmLoadRegMem(a *rtgAsm, dst int, base int, disp int, size int) {
	addr := rtgAarch64AsmAddr(a, base, disp)
	if size == 1 {
		rtgAarch64AsmEmit(a, 0x39400000|(addr<<5)|dst)
		return
	}
	if size == 2 {
		rtgAarch64AsmEmit(a, 0x79800000|(addr<<5)|dst)
		return
	}
	if size == 4 {
		rtgAarch64AsmEmit(a, 0xb9800000|(addr<<5)|dst)
		return
	}
	rtgAarch64AsmEmit(a, 0xf9400000|(addr<<5)|dst)
}

func rtgAarch64AsmStoreRegMem(a *rtgAsm, src int, base int, disp int, size int) {
	addr := rtgAarch64AsmAddr(a, base, disp)
	if size == 1 {
		rtgAarch64AsmEmit(a, 0x39000000|(addr<<5)|src)
		return
	}
	if size == 2 {
		rtgAarch64AsmEmit(a, 0x79000000|(addr<<5)|src)
		return
	}
	if size == 4 {
		rtgAarch64AsmEmit(a, 0xb9000000|(addr<<5)|src)
		return
	}
	rtgAarch64AsmEmit(a, 0xf9000000|(addr<<5)|src)
}

func rtgAarch64AsmLoadRegStack(a *rtgAsm, dst int, offset int) {
	rtgAarch64AsmLoadRegMem(a, dst, rtgAarch64RegFp, -offset, 8)
}

func rtgAarch64AsmStoreRegStack(a *rtgAsm, src int, offset int) {
	rtgAarch64AsmStoreRegMem(a, src, rtgAarch64RegFp, -offset, 8)
}

func rtgAarch64AsmLeaRegStack(a *rtgAsm, dst int, offset int) {
	rtgAarch64AsmAddRegImm(a, dst, rtgAarch64RegFp, -offset)
}

func rtgAarch64AsmPushReg(a *rtgAsm, reg int) {
	rtgAarch64AsmEmit(a, 0xf81f0fe0|reg)
}

func rtgAarch64AsmPopReg(a *rtgAsm, reg int) {
	rtgAarch64AsmEmit(a, 0xf84107e0|reg)
}

func rtgAarch64AsmMovRaxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegRax, imm)
}

func rtgAarch64AsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAarch64AsmMovRaxImm(a, imm)
}

func rtgAarch64AsmMovRdxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegRdx, imm)
}

func rtgAarch64AsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRax, dataOff, 0)
}

func rtgAarch64AsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRax, bssOff, rtgAbsBssReloc)
}

func rtgAarch64AsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegR10, bssOff, rtgAbsBssReloc)
}

func rtgAarch64AsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegAddr, bssOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 8)
}

func rtgAarch64AsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegAddr, bssOff, rtgAbsBssReloc)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 8)
}

func rtgAarch64AsmMovRdiRax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdi, rtgAarch64RegRax)
}

func rtgAarch64AsmMovRaxRdx(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRax, rtgAarch64RegRdx)
}

func rtgAarch64AsmMovRdxRax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRax)
}

func rtgAarch64AsmMovRcxRax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRcx, rtgAarch64RegRax)
}

func rtgAarch64AsmMovRcxRdx(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRcx, rtgAarch64RegRdx)
}

func rtgAarch64AsmMovRsiRax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRsi, rtgAarch64RegRax)
}

func rtgAarch64AsmMovR8Rax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR8, rtgAarch64RegRax)
}

func rtgAarch64AsmMovR9Rax(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR9, rtgAarch64RegRax)
}

func rtgAarch64AsmAddRdxRcx(a *rtgAsm) {
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRdx, rtgAarch64RegRcx)
}

func rtgAarch64AsmSyscall(a *rtgAsm) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegSys, rtgAarch64RegRax)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegTmp, rtgAarch64RegRdx)
	rtgAarch64AsmMovRegReg(a, 0, rtgAarch64RegRdi)
	rtgAarch64AsmMovRegReg(a, 1, rtgAarch64RegRsi)
	rtgAarch64AsmMovRegReg(a, 2, rtgAarch64RegTmp)
	rtgAarch64AsmMovRegReg(a, 3, rtgAarch64RegR10)
	rtgAarch64AsmEmit(a, 0xd4000001)
}

func rtgAarch64AsmPopRdi(a *rtgAsm) {
	rtgAarch64AsmPopReg(a, rtgAarch64RegRdi)
}

func rtgAarch64AsmPopRsi(a *rtgAsm) {
	rtgAarch64AsmPopReg(a, rtgAarch64RegRsi)
}

func rtgAarch64AsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	if base == 0x8948 && disp8 == 0x45 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRax, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x55 {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRdx, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x4d {
		rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRcx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x45 {
		rtgAarch64AsmLoadRegStack(a, rtgAarch64RegRax, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x55 {
		rtgAarch64AsmLoadRegStack(a, rtgAarch64RegRdx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x4d {
		rtgAarch64AsmLoadRegStack(a, rtgAarch64RegRcx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x45 {
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRax, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x55 {
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRdx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x7d {
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRdi, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x75 {
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRsi, offset)
		return
	}
}

func rtgAarch64AsmAddRdxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRdx, rtgAarch64RegRdx, imm)
}

func rtgAarch64AsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	if op == 0x8b48 && disp8 == 0x4a {
		rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRcx, rtgAarch64RegRdx, disp, 8)
		return
	}
	if op == 0x8b48 && disp8 == 0x52 {
		rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRdx, rtgAarch64RegRdx, disp, 8)
		return
	}
}

func rtgAarch64AsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRax, rtgAarch64RegRcx, 3)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 8)
}

func rtgAarch64AsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegAddr, rtgAarch64RegRax, rtgAarch64RegRcx)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, disp, 8)
}

func rtgAarch64AsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdx, disp, 8)
}

func rtgAarch64AsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdx, disp, size)
}

func rtgAarch64AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRax, rtgAarch64RegRcx, 0)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 1)
}

func rtgAarch64AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
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
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRax, rtgAarch64RegRcx, shift)
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, size)
}

func rtgAarch64AsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRdx, rtgAarch64RegRcx, 3)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 8)
}

func rtgAarch64AsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdx, disp, 8)
}

func rtgAarch64AsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegRdx, disp, size)
}

func rtgAarch64AsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		rtgAarch64AsmEmit(a, 0x92401c00)
		return
	}
	if kind == rtgTypeInt16 {
		rtgAarch64AsmEmit(a, 0x93403c00)
		return
	}
	if kind == rtgTypeInt32 {
		rtgAarch64AsmEmit(a, 0x93407c00)
	}
}

func rtgAarch64AsmIncMemRdx(a *rtgAsm) {
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdx, 0, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegTmp, rtgAarch64RegTmp, 1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdx, 0, 8)
}

func rtgAarch64AsmDecMemRdx(a *rtgAsm) {
	rtgAarch64AsmLoadRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdx, 0, 8)
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegTmp, rtgAarch64RegTmp, -1)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegTmp, rtgAarch64RegRdx, 0, 8)
}

func rtgAarch64AsmBoolNotRax(a *rtgAsm) {
	rtgAarch64AsmCmpRaxImm8(a, 0)
	rtgAarch64AsmCsetRax(a, 0)
}

func rtgAarch64AsmNegRax(a *rtgAsm) {
	rtgAarch64AsmSubRegReg(a, rtgAarch64RegRax, rtgAarch64RegZr, rtgAarch64RegRax)
}

func rtgAarch64AsmCmpRaxImm8(a *rtgAsm, imm int) {
	rtgAarch64AsmCmpRegImm(a, rtgAarch64RegRax, imm)
}

func rtgAarch64AsmCmpRegImm(a *rtgAsm, reg int, imm int) {
	if imm >= 0 && imm <= 4095 {
		rtgAarch64AsmEmit(a, 0xf100001f|(imm<<10)|(reg<<5))
		return
	}
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegTmp, imm)
	rtgAarch64AsmCmpRegReg(a, reg, rtgAarch64RegTmp)
}

func rtgAarch64AsmCmpRegReg(a *rtgAsm, left int, right int) {
	rtgAarch64AsmEmit(a, 0xeb00001f|(right<<16)|(left<<5))
}

func rtgAarch64AsmCsetRax(a *rtgAsm, cond int) {
	rtgAarch64AsmEmit(a, 0x9a9f07e0|((cond^1)<<12))
}

func rtgAarch64AsmAddRaxRcx(a *rtgAsm) {
	rtgAarch64AsmAddRegReg(a, rtgAarch64RegRax, rtgAarch64RegRax, rtgAarch64RegRcx)
}

func rtgAarch64AsmSubRaxRcx(a *rtgAsm) {
	rtgAarch64AsmSubRegReg(a, rtgAarch64RegRax, rtgAarch64RegRax, rtgAarch64RegRcx)
}

func rtgAarch64AsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmEmit(a, 0xd3400000|((64-imm)<<16)|((63-imm)<<10)|(rtgAarch64RegRcx<<5)|rtgAarch64RegRcx)
}

func rtgAarch64AsmShlRaxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmEmit(a, 0xd3400000|((64-imm)<<16)|((63-imm)<<10))
}

func rtgAarch64AsmSarRaxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmEmit(a, 0x9340fc00|(imm<<16))
}

func rtgAarch64AsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegTmp, rtgAarch64RegRax)
	rtgAarch64AsmEmit(a, 0x9ac00c40)
	if mod {
		rtgAarch64AsmEmit(a, 0x9b098800)
	}
}

func rtgAarch64CondFromSetcc(setcc int) int {
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

func rtgAarch64AsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAarch64AsmCmpRegReg(a, rtgAarch64RegRcx, rtgAarch64RegRax)
	cond := rtgAarch64CondFromSetcc(setcc)
	rtgAarch64AsmCsetRax(a, cond)
}

func rtgAarch64AsmPushRax(a *rtgAsm) {
	rtgAarch64AsmPushReg(a, rtgAarch64RegRax)
}

func rtgAarch64AsmPushRcx(a *rtgAsm) {
	rtgAarch64AsmPushReg(a, rtgAarch64RegRcx)
}

func rtgAarch64AsmPushRdx(a *rtgAsm) {
	rtgAarch64AsmPushReg(a, rtgAarch64RegRdx)
}

func rtgAarch64AsmPopRax(a *rtgAsm) {
	rtgAarch64AsmPopReg(a, rtgAarch64RegRax)
}

func rtgAarch64AsmPopRcx(a *rtgAsm) {
	rtgAarch64AsmPopReg(a, rtgAarch64RegRcx)
}

func rtgAarch64AsmPopRdx(a *rtgAsm) {
	rtgAarch64AsmPopReg(a, rtgAarch64RegRdx)
}

func rtgAarch64AsmPushImm(a *rtgAsm, imm int) {
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegTmp, imm)
	rtgAarch64AsmPushReg(a, rtgAarch64RegTmp)
}

func rtgAarch64AsmStoreSliceStack(a *rtgAsm, offset int) {
	rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRax, offset)
	rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRdx, offset-8)
	rtgAarch64AsmStoreRegStack(a, rtgAarch64RegRcx, offset-16)
}

func rtgAarch64AsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRdx, rtgAarch64RegRcx, 0)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, 1)
}

func rtgAarch64AsmStoreRaxMemRdxRcxSize(a *rtgAsm, size int) {
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
	rtgAarch64AsmAddRegRegShift(a, rtgAarch64RegAddr, rtgAarch64RegRdx, rtgAarch64RegRcx, shift)
	rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegAddr, 0, size)
}

func rtgAarch64AsmIncRcx(a *rtgAsm) {
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRcx, rtgAarch64RegRcx, 1)
}

func rtgAarch64AsmIncRax(a *rtgAsm) {
	rtgAarch64AsmAddRegImm(a, rtgAarch64RegRax, rtgAarch64RegRax, 1)
}

func rtgAarch64AsmImulRcxImm(a *rtgAsm, imm int) {
	rtgAarch64AsmMovRegImm(a, rtgAarch64RegTmp, imm)
	rtgAarch64AsmEmit(a, 0x9b007c00|(rtgAarch64RegTmp<<16)|(rtgAarch64RegRcx<<5)|rtgAarch64RegRcx)
}

func rtgAarch64AsmLeave(a *rtgAsm) {
	rtgAarch64AsmEmit(a, 0x910003bf)
	rtgAarch64AsmEmit(a, 0xa8c17bfd)
}

func rtgAarch64AsmRet(a *rtgAsm) {
	rtgAarch64AsmEmit(a, 0xd65f03c0)
}

func rtgAarch64AsmCallLabel(a *rtgAsm, label int) {
	at := len(a.code)
	rtgAarch64AsmEmit(a, 0x94000000)
	rtgAsmAddReloc(a, at, label)
}

func rtgAarch64AsmJmpLabel(a *rtgAsm, label int) {
	at := len(a.code)
	rtgAarch64AsmEmit(a, 0x14000000)
	rtgAsmAddReloc(a, at, label)
}

func rtgAarch64AsmBCondLabel(a *rtgAsm, label int, cond int) {
	at := len(a.code)
	rtgAarch64AsmEmit(a, 0x54000000|cond)
	rtgAsmAddReloc(a, at, label)
}

func rtgAarch64AsmJzLabel(a *rtgAsm, label int) {
	rtgAarch64AsmBCondLabel(a, label, 0)
}

func rtgAarch64AsmJnzLabel(a *rtgAsm, label int) {
	rtgAarch64AsmBCondLabel(a, label, 1)
}
