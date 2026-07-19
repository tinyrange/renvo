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

func renvoAarch64EmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
		renvoAarch64AsmEmit(a, 0x9b007c40)
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

func renvoAarch64AsmEmit(a *renvoAsm, insn int) {
	renvoAsmEmit32(a, insn)
}

func renvoAarch64AsmAlign(a *renvoAsm) {
	for len(a.code)%4 != 0 {
		renvoAsmEmit8(a, 0)
	}
}

func renvoAarch64AsmMovRegReg(a *renvoAsm, dst int, src int) {
	if dst == src {
		return
	}
	renvoAarch64AsmEmit(a, 0xaa0003e0|(src<<16)|dst)
}

func renvoAarch64AsmMovRegImm(a *renvoAsm, reg int, imm int) {
	if imm < 0 {
		inv := -imm - 1
		part := inv & 65535
		renvoAarch64AsmEmit(a, 0x92800000|(part<<5)|reg)
		for i := 1; i < 4; i++ {
			targetPart := (imm >> (i * 16)) & 65535
			if targetPart != 65535 {
				renvoAarch64AsmEmit(a, 0xf2800000|(i<<21)|(targetPart<<5)|reg)
			}
		}
		return
	}
	part := imm & 65535
	renvoAarch64AsmEmit(a, 0xd2800000|(part<<5)|reg)
	for i := 1; i < 4; i++ {
		part = (imm >> (i * 16)) & 65535
		if part != 0 {
			renvoAarch64AsmEmit(a, 0xf2800000|(i<<21)|(part<<5)|reg)
		}
	}
}

func renvoAarch64AsmPatchMovRegImmAt(a *renvoAsm, at int, reg int, imm int) {
	part := imm & 65535
	renvoPut32At(a.code, at, 0xd2800000|(part<<5)|reg)
	for i := 1; i < 4; i++ {
		part = (imm >> (i * 16)) & 65535
		renvoPut32At(a.code, at+i*4, 0xf2800000|(i<<21)|(part<<5)|reg)
	}
}

func renvoAarch64AsmMovRegAbs(a *renvoAsm, reg int, off int, kind int) {
	at := len(a.code)
	renvoAarch64AsmMovRegImm(a, reg, 0)
	renvoAarch64AsmEmit(a, 0xf2800000|(1<<21)|reg)
	renvoAarch64AsmEmit(a, 0xf2800000|(2<<21)|reg)
	renvoAarch64AsmEmit(a, 0xf2800000|(3<<21)|reg)
	renvoAsmAddAbsReloc(a, at, off, kind)
}

func renvoAarch64AsmAddRegImm(a *renvoAsm, dst int, src int, imm int) {
	if imm == 0 {
		renvoAarch64AsmMovRegReg(a, dst, src)
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
		renvoAarch64AsmEmit(a, op|(shift<<22)|(chunk<<10)|(cur<<5)|dst)
		cur = dst
	}
}

func renvoAarch64AsmFrameStart(a *renvoAsm) int {
	at := len(a.code)
	count := 2
	if targetOS == renvoOSWindows {
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
	if targetOS == renvoOSWindows {
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

func renvoAarch64AsmAddRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoAarch64AsmEmit(a, 0x8b000000|(right<<16)|(left<<5)|dst)
}

func renvoAarch64AsmSubRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoAarch64AsmEmit(a, 0xcb000000|(right<<16)|(left<<5)|dst)
}

func renvoAarch64AsmAddRegRegShift(a *renvoAsm, dst int, left int, right int, shift int) {
	renvoAarch64AsmEmit(a, 0x8b000000|(right<<16)|(shift<<10)|(left<<5)|dst)
}

func renvoAarch64AsmMulRegReg(a *renvoAsm, dst int, left int, right int) {
	renvoAarch64AsmEmit(a, 0x9b007c00|(right<<16)|(left<<5)|dst)
}

func renvoAarch64AsmAddr(a *renvoAsm, base int, disp int) int {
	if disp == 0 {
		return base
	}
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegAddr, base, disp)
	return renvoAarch64RegAddr
}

func renvoAarch64AsmLoadRegMem(a *renvoAsm, dst int, base int, disp int, size int) {
	if disp != 0 && disp >= -256 && disp <= 255 {
		imm := (disp & 511) << 12
		if size == 1 {
			renvoAarch64AsmEmit(a, 0x38400000|imm|(base<<5)|dst)
			return
		}
		if size == 2 {
			renvoAarch64AsmEmit(a, 0x78400000|imm|(base<<5)|dst)
			return
		}
		if size == 4 {
			renvoAarch64AsmEmit(a, 0xb8400000|imm|(base<<5)|dst)
			return
		}
		renvoAarch64AsmEmit(a, 0xf8400000|imm|(base<<5)|dst)
		return
	}
	addr := renvoAarch64AsmAddr(a, base, disp)
	if size == 1 {
		renvoAarch64AsmEmit(a, 0x39400000|(addr<<5)|dst)
		return
	}
	if size == 2 {
		renvoAarch64AsmEmit(a, 0x79800000|(addr<<5)|dst)
		return
	}
	if size == 4 {
		renvoAarch64AsmEmit(a, 0xb9800000|(addr<<5)|dst)
		return
	}
	renvoAarch64AsmEmit(a, 0xf9400000|(addr<<5)|dst)
}

func renvoAarch64AsmStoreRegMem(a *renvoAsm, src int, base int, disp int, size int) {
	if disp != 0 && disp >= -256 && disp <= 255 {
		imm := (disp & 511) << 12
		if size == 1 {
			renvoAarch64AsmEmit(a, 0x38000000|imm|(base<<5)|src)
			return
		}
		if size == 2 {
			renvoAarch64AsmEmit(a, 0x78000000|imm|(base<<5)|src)
			return
		}
		if size == 4 {
			renvoAarch64AsmEmit(a, 0xb8000000|imm|(base<<5)|src)
			return
		}
		renvoAarch64AsmEmit(a, 0xf8000000|imm|(base<<5)|src)
		return
	}
	addr := renvoAarch64AsmAddr(a, base, disp)
	if size == 1 {
		renvoAarch64AsmEmit(a, 0x39000000|(addr<<5)|src)
		return
	}
	if size == 2 {
		renvoAarch64AsmEmit(a, 0x79000000|(addr<<5)|src)
		return
	}
	if size == 4 {
		renvoAarch64AsmEmit(a, 0xb9000000|(addr<<5)|src)
		return
	}
	renvoAarch64AsmEmit(a, 0xf9000000|(addr<<5)|src)
}

func renvoAarch64AsmLoadRegStack(a *renvoAsm, dst int, offset int) {
	renvoAarch64AsmLoadRegMem(a, dst, renvoAarch64RegFp, -offset, 8)
}

func renvoAarch64AsmStoreRegStack(a *renvoAsm, src int, offset int) {
	renvoAarch64AsmStoreRegMem(a, src, renvoAarch64RegFp, -offset, 8)
}

func renvoAarch64AsmLeaRegStack(a *renvoAsm, dst int, offset int) {
	renvoAarch64AsmAddRegImm(a, dst, renvoAarch64RegFp, -offset)
}

func renvoAarch64AsmPushReg(a *renvoAsm, reg int) {
	renvoAarch64AsmEmit(a, 0xf81f0fe0|reg)
}

func renvoAarch64AsmPopReg(a *renvoAsm, reg int) {
	renvoAarch64AsmEmit(a, 0xf84107e0|reg)
}

func renvoAarch64AsmMovRaxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegRax, imm)
}

func renvoAarch64AsmMovRaxImm64(a *renvoAsm, imm int) {
	renvoAarch64AsmMovRaxImm(a, imm)
}

func renvoAarch64AsmMovRdxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegRdx, imm)
}

func renvoAarch64AsmMovRaxDataAddr(a *renvoAsm, dataOff int) {
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRax, dataOff, 0)
}

func renvoAarch64AsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRax, bssOff, renvoAbsBssReloc)
}

func renvoAarch64AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegR10, bssOff, renvoAbsBssReloc)
}

func renvoAarch64AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegAddr, bssOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 8)
}

func renvoAarch64AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegAddr, bssOff, renvoAbsBssReloc)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 8)
}

func renvoAarch64AsmMovRdiRax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdi, renvoAarch64RegRax)
}

func renvoAarch64AsmMovRaxRdx(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRax, renvoAarch64RegRdx)
}

func renvoAarch64AsmMovRdxRax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRax)
}

func renvoAarch64AsmMovRcxRax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRcx, renvoAarch64RegRax)
}

func renvoAarch64AsmMovRcxRdx(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRcx, renvoAarch64RegRdx)
}

func renvoAarch64AsmMovRsiRax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRsi, renvoAarch64RegRax)
}

func renvoAarch64AsmMovR8Rax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR8, renvoAarch64RegRax)
}

func renvoAarch64AsmMovR9Rax(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR9, renvoAarch64RegRax)
}

func renvoAarch64AsmAddRdxRcx(a *renvoAsm) {
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRdx, renvoAarch64RegRcx)
}

func renvoAarch64AsmSyscall(a *renvoAsm) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegSys, renvoAarch64RegRax)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegTmp, renvoAarch64RegRdx)
	renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRdi)
	renvoAarch64AsmMovRegReg(a, 1, renvoAarch64RegRsi)
	renvoAarch64AsmMovRegReg(a, 2, renvoAarch64RegTmp)
	renvoAarch64AsmMovRegReg(a, 3, renvoAarch64RegR10)
	renvoAarch64AsmEmit(a, 0xd4000001)
}

func renvoAarch64AsmPopRdi(a *renvoAsm) {
	renvoAarch64AsmPopReg(a, renvoAarch64RegRdi)
}

func renvoAarch64AsmPopRsi(a *renvoAsm) {
	renvoAarch64AsmPopReg(a, renvoAarch64RegRsi)
}

func renvoAarch64AsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	if base == 0x8948 && disp8 == 0x45 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRax, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x55 {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRdx, offset)
		return
	}
	if base == 0x8948 && disp8 == 0x4d {
		renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRcx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x45 {
		renvoAarch64AsmLoadRegStack(a, renvoAarch64RegRax, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x55 {
		renvoAarch64AsmLoadRegStack(a, renvoAarch64RegRdx, offset)
		return
	}
	if base == 0x8b48 && disp8 == 0x4d {
		renvoAarch64AsmLoadRegStack(a, renvoAarch64RegRcx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x45 {
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRax, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x55 {
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRdx, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x7d {
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRdi, offset)
		return
	}
	if base == 0x8d48 && disp8 == 0x75 {
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRsi, offset)
		return
	}
}

func renvoAarch64AsmAddRdxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRdx, renvoAarch64RegRdx, imm)
}

func renvoAarch64AsmMemDisp(a *renvoAsm, disp int, op int, disp8 int, disp32 int) {
	if op == 0x8b48 && disp8 == 0x4a {
		renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRcx, renvoAarch64RegRdx, disp, 8)
		return
	}
	if op == 0x8b48 && disp8 == 0x52 {
		renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRdx, renvoAarch64RegRdx, disp, 8)
		return
	}
}

func renvoAarch64AsmLoadQwordRaxIndexRcx8(a *renvoAsm) {
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRax, renvoAarch64RegRcx, 3)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 8)
}

func renvoAarch64AsmLoadQwordRaxIndexRcxDisp(a *renvoAsm, disp int) {
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegAddr, renvoAarch64RegRax, renvoAarch64RegRcx)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, disp, 8)
}

func renvoAarch64AsmLoadRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdx, disp, 8)
}

func renvoAarch64AsmLoadRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdx, disp, size)
}

func renvoAarch64AsmLoadByteRaxIndexRcx(a *renvoAsm) {
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRax, renvoAarch64RegRcx, 0)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 1)
}

func renvoAarch64AsmLoadRaxIndexRcxSize(a *renvoAsm, size int) {
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
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRax, renvoAarch64RegRcx, shift)
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, size)
}

func renvoAarch64AsmStoreRaxMemRdxRcx8(a *renvoAsm) {
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRdx, renvoAarch64RegRcx, 3)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 8)
}

func renvoAarch64AsmStoreRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdx, disp, 8)
}

func renvoAarch64AsmStoreRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegRdx, disp, size)
}

func renvoAarch64AsmNormalizeRaxForKind(a *renvoAsm, kind int) {
	if kind == renvoTypeByte {
		renvoAarch64AsmEmit(a, 0x92401c00)
		return
	}
	if kind == renvoTypeInt8 {
		renvoAarch64AsmEmit(a, 0x93401c00)
		return
	}
	if kind == renvoTypeInt16 {
		renvoAarch64AsmEmit(a, 0x93403c00)
		return
	}
	if kind == renvoTypeUint16 {
		renvoAarch64AsmEmit(a, 0x92403c00)
		return
	}
	if kind == renvoTypeInt32 {
		renvoAarch64AsmEmit(a, 0x93407c00)
		return
	}
	if kind == renvoTypeUint32 {
		renvoAarch64AsmEmit(a, 0x92407c00)
	}
}

func renvoAarch64AsmIncMemRdx(a *renvoAsm) {
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdx, 0, 8)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegTmp, renvoAarch64RegTmp, 1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdx, 0, 8)
}

func renvoAarch64AsmDecMemRdx(a *renvoAsm) {
	renvoAarch64AsmLoadRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdx, 0, 8)
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegTmp, renvoAarch64RegTmp, -1)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegTmp, renvoAarch64RegRdx, 0, 8)
}

func renvoAarch64AsmBoolNotRax(a *renvoAsm) {
	renvoAarch64AsmCmpRaxImm8(a, 0)
	renvoAarch64AsmCsetRax(a, 0)
}

func renvoAarch64AsmNegRax(a *renvoAsm) {
	renvoAarch64AsmSubRegReg(a, renvoAarch64RegRax, renvoAarch64RegZr, renvoAarch64RegRax)
}

func renvoAarch64AsmCmpRaxImm8(a *renvoAsm, imm int) {
	renvoAarch64AsmCmpRegImm(a, renvoAarch64RegRax, imm)
}

func renvoAarch64AsmCmpRegImm(a *renvoAsm, reg int, imm int) {
	if imm >= 0 && imm <= 4095 {
		renvoAarch64AsmEmit(a, 0xf100001f|(imm<<10)|(reg<<5))
		return
	}
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegTmp, imm)
	renvoAarch64AsmCmpRegReg(a, reg, renvoAarch64RegTmp)
}

func renvoAarch64AsmCmpRegReg(a *renvoAsm, left int, right int) {
	renvoAarch64AsmEmit(a, 0xeb00001f|(right<<16)|(left<<5))
}

func renvoAarch64AsmCsetRax(a *renvoAsm, cond int) {
	renvoAarch64AsmEmit(a, 0x9a9f07e0|((cond^1)<<12))
}

func renvoAarch64AsmAddRaxRcx(a *renvoAsm) {
	renvoAarch64AsmAddRegReg(a, renvoAarch64RegRax, renvoAarch64RegRax, renvoAarch64RegRcx)
}

func renvoAarch64AsmSubRaxRcx(a *renvoAsm) {
	renvoAarch64AsmSubRegReg(a, renvoAarch64RegRax, renvoAarch64RegRax, renvoAarch64RegRcx)
}

func renvoAarch64AsmShlRcxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmEmit(a, 0xd3400000|((64-imm)<<16)|((63-imm)<<10)|(renvoAarch64RegRcx<<5)|renvoAarch64RegRcx)
}

func renvoAarch64AsmShlRaxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmEmit(a, 0xd3400000|((64-imm)<<16)|((63-imm)<<10))
}

func renvoAarch64AsmSarRaxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmEmit(a, 0x9340fc00|(imm<<16))
}

func renvoAarch64AsmDivLeftRcxRightRax(a *renvoAsm, mod bool) {
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegTmp, renvoAarch64RegRax)
	renvoAarch64AsmEmit(a, 0x9ac00c40)
	if mod {
		renvoAarch64AsmEmit(a, 0x9b098800)
	}
}

func renvoAarch64CondFromSetcc(setcc int) int {
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

func renvoAarch64AsmCmpRcxRaxSet(a *renvoAsm, setcc int) {
	renvoAarch64AsmCmpRegReg(a, renvoAarch64RegRcx, renvoAarch64RegRax)
	cond := renvoAarch64CondFromSetcc(setcc)
	renvoAarch64AsmCsetRax(a, cond)
}

func renvoAarch64AsmPushRax(a *renvoAsm) {
	renvoAarch64AsmPushReg(a, renvoAarch64RegRax)
}

func renvoAarch64AsmPushRcx(a *renvoAsm) {
	renvoAarch64AsmPushReg(a, renvoAarch64RegRcx)
}

func renvoAarch64AsmPushRdx(a *renvoAsm) {
	renvoAarch64AsmPushReg(a, renvoAarch64RegRdx)
}

func renvoAarch64AsmPopRax(a *renvoAsm) {
	renvoAarch64AsmPopReg(a, renvoAarch64RegRax)
}

func renvoAarch64AsmPopRcx(a *renvoAsm) {
	renvoAarch64AsmPopReg(a, renvoAarch64RegRcx)
}

func renvoAarch64AsmPopRdx(a *renvoAsm) {
	renvoAarch64AsmPopReg(a, renvoAarch64RegRdx)
}

func renvoAarch64AsmPushImm(a *renvoAsm, imm int) {
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegTmp, imm)
	renvoAarch64AsmPushReg(a, renvoAarch64RegTmp)
}

func renvoAarch64AsmStoreSliceStack(a *renvoAsm, offset int) {
	renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRax, offset)
	renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRdx, offset-8)
	renvoAarch64AsmStoreRegStack(a, renvoAarch64RegRcx, offset-16)
}

func renvoAarch64AsmStoreAlMemRdxRcx1(a *renvoAsm) {
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRdx, renvoAarch64RegRcx, 0)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, 1)
}

func renvoAarch64AsmStoreRaxMemRdxRcxSize(a *renvoAsm, size int) {
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
	renvoAarch64AsmAddRegRegShift(a, renvoAarch64RegAddr, renvoAarch64RegRdx, renvoAarch64RegRcx, shift)
	renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegAddr, 0, size)
}

func renvoAarch64AsmIncRcx(a *renvoAsm) {
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRcx, renvoAarch64RegRcx, 1)
}

func renvoAarch64AsmIncRax(a *renvoAsm) {
	renvoAarch64AsmAddRegImm(a, renvoAarch64RegRax, renvoAarch64RegRax, 1)
}

func renvoAarch64AsmImulRcxImm(a *renvoAsm, imm int) {
	renvoAarch64AsmMovRegImm(a, renvoAarch64RegTmp, imm)
	renvoAarch64AsmEmit(a, 0x9b007c00|(renvoAarch64RegTmp<<16)|(renvoAarch64RegRcx<<5)|renvoAarch64RegRcx)
}

func renvoAarch64AsmLeave(a *renvoAsm) {
	renvoAarch64AsmEmit(a, 0x910003bf)
	renvoAarch64AsmEmit(a, 0xa8c17bfd)
}

func renvoAarch64AsmRet(a *renvoAsm) {
	renvoAarch64AsmEmit(a, 0xd65f03c0)
}

func renvoAarch64AsmCallLabel(a *renvoAsm, label int) {
	at := len(a.code)
	renvoAarch64AsmEmit(a, 0x94000000)
	renvoAsmAddReloc(a, at, label)
}

func renvoAarch64AsmJmpLabel(a *renvoAsm, label int) {
	at := len(a.code)
	renvoAarch64AsmEmit(a, 0x14000000)
	renvoAsmAddReloc(a, at, label)
}

func renvoAarch64AsmBCondLabel(a *renvoAsm, label int, cond int) {
	at := len(a.code)
	renvoAarch64AsmEmit(a, 0x54000000|cond)
	renvoAsmAddReloc(a, at, label)
}

func renvoAarch64AsmJzLabel(a *renvoAsm, label int) {
	renvoAarch64AsmBCondLabel(a, label, 0)
}

func renvoAarch64AsmJnzLabel(a *renvoAsm, label int) {
	renvoAarch64AsmBCondLabel(a, label, 1)
}
