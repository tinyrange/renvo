package main

func renvoArmEnsureWideBinaryHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideBinaryLabel > 0 {
		return g.wideBinaryLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideBinaryLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x0d\x00\x50\xe3\x08\x00\x00\x1a\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x06\x10\xc1\xe1\x07\x20\xc2\xe1\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x0e\x00\x50\xe3\x00\x00\x00\xba\x94\x00\x00\xea\x00\x00\x50\xe3\x08\x00\x00\x0a\x01\x00\x50\xe3\x0f\x00\x00\x0a\x02\x00\x50\xe3\x16\x00\x00\x0a\x06\x00\x50\xe3\x52\x00\x00\xda\x09\x00\x50\xe3\x2f\x00\x00\xda\x1b\x00\x00\xea\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x10\x91\xe0\x00\x10\x83\xe5\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x10\xa1\xe0\x04\x10\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x10\x51\xe0\x00\x10\x83\xe5\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x10\xc1\xe0\x04\x10\x83\xe5\x1e\xff\x2f\xe1\x00\x60\x94\xe5\x00\x70\x95\xe5\x96\x17\x82\xe0\x04\x80\x94\xe5\x98\x27\x22\xe0\x04\x80\x95\xe5\x96\x28\x22\xe0\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x0a\x00\x50\xe3\x04\x00\x00\x0a\x0b\x00\x50\xe3\x05\x00\x00\x0a\x06\x10\x21\xe0\x07\x20\x22\xe0\x04\x00\x00\xea\x06\x10\x01\xe0\x07\x20\x02\xe0\x01\x00\x00\xea\x06\x10\x81\xe1\x07\x20\x82\xe1\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x00\x00\x57\xe3\x40\x60\xa0\x13\x40\x00\x56\xe3\x40\x60\xa0\x83\x09\x00\x50\xe3\x0d\x00\x00\x0a\x08\x00\x50\xe3\x05\x00\x00\x0a\x00\x00\x56\xe3\x0f\x00\x00\x0a\x01\x10\x91\xe0\x02\x20\xa2\xe0\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x00\x56\xe3\x09\x00\x00\x0a\xa2\x20\xb0\xe1\x61\x10\xa0\xe1\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x00\x56\xe3\x03\x00\x00\x0a\xc2\x20\xb0\xe1\x61\x10\xa0\xe1\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x09\x40\x2d\xe9\x00\x00\x94\xe5\x04\x10\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\xc1\x4f\xa0\xe1\xc7\x5f\xa0\xe1\x00\xa0\x9d\xe5\x05\x00\x5a\xe3\x07\x00\x00\xba\x04\x00\x20\xe0\x04\x10\x21\xe0\x04\x00\x50\xe0\x04\x10\xc1\xe0\x05\x60\x26\xe0\x05\x70\x27\xe0\x05\x60\x56\xe0\x05\x70\xc7\xe0\x00\x20\xa0\xe3\x00\x30\xa0\xe3\x00\x80\xa0\xe3\x00\x90\xa0\xe3\x40\xa0\xa0\xe3\x00\x00\x90\xe0\x01\x10\xb1\xe0\x08\x80\xb8\xe0\x09\x90\xa9\xe0\x02\x20\x92\xe0\x03\x30\xa3\xe0\x07\x00\x59\xe1\x05\x00\x00\x3a\x01\x00\x00\x8a\x06\x00\x58\xe1\x02\x00\x00\x3a\x06\x80\x58\xe0\x07\x90\xc9\xe0\x01\x20\x82\xe3\x01\xa0\x5a\xe2\xef\xff\xff\x1a\x00\x00\x9d\xe5\x04\x10\x9d\xe5\x01\x00\x10\xe3\x07\x00\x00\x0a\x05\x00\x24\xe0\x00\x20\x22\xe0\x00\x30\x23\xe0\x00\x20\x52\xe0\x00\x30\xc3\xe0\x00\x20\x81\xe5\x04\x30\x81\xe5\x05\x00\x00\xea\x04\x80\x28\xe0\x04\x90\x29\xe0\x04\x80\x58\xe0\x04\x90\xc9\xe0\x00\x80\x81\xe5\x04\x90\x81\xe5\x09\x80\xbd\xe8\x0e\x00\x40\xe2\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x00\x51\xe1\x0a\x00\x00\x1a\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x00\x51\xe1\x10\x00\x00\x1a\x00\x00\x50\xe3\x25\x00\x00\x0a\x01\x00\x50\xe3\x21\x00\x00\x0a\x01\x00\x10\xe3\x21\x00\x00\x1a\x1e\x00\x00\xea\x01\x00\x50\xe3\x1e\x00\x00\x0a\x06\x00\x50\xe3\x02\x00\x00\x2a\x02\x00\x51\xe1\x10\x00\x00\xba\x06\x00\x00\xea\x02\x00\x51\xe1\x0d\x00\x00\x3a\x03\x00\x00\xea\x01\x00\x50\xe3\x14\x00\x00\x0a\x02\x00\x51\xe1\x08\x00\x00\x3a\x04\x00\x50\xe3\x10\x00\x00\x0a\x05\x00\x50\xe3\x0e\x00\x00\x0a\x08\x00\x50\xe3\x0c\x00\x00\x0a\x09\x00\x50\xe3\x0a\x00\x00\x0a\x07\x00\x00\xea\x02\x00\x50\xe3\x07\x00\x00\x0a\x03\x00\x50\xe3\x05\x00\x00\x0a\x06\x00\x50\xe3\x03\x00\x00\x0a\x07\x00\x50\xe3\x01\x00\x00\x0a\x00\x00\xa0\xe3\x1e\xff\x2f\xe1\x01\x00\xa0\xe3\x1e\xff\x2f\xe1")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoArmEnsureWideCompareHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideCompareLabel > 0 {
		return g.wideCompareLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideCompareLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x00\x51\xe1\x0a\x00\x00\x1a\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x00\x51\xe1\x10\x00\x00\x1a\x00\x00\x50\xe3\x25\x00\x00\x0a\x01\x00\x50\xe3\x21\x00\x00\x0a\x01\x00\x10\xe3\x21\x00\x00\x1a\x1e\x00\x00\xea\x01\x00\x50\xe3\x1e\x00\x00\x0a\x06\x00\x50\xe3\x02\x00\x00\x2a\x02\x00\x51\xe1\x10\x00\x00\xba\x06\x00\x00\xea\x02\x00\x51\xe1\x0d\x00\x00\x3a\x03\x00\x00\xea\x01\x00\x50\xe3\x14\x00\x00\x0a\x02\x00\x51\xe1\x08\x00\x00\x3a\x04\x00\x50\xe3\x10\x00\x00\x0a\x05\x00\x50\xe3\x0e\x00\x00\x0a\x08\x00\x50\xe3\x0c\x00\x00\x0a\x09\x00\x50\xe3\x0a\x00\x00\x0a\x07\x00\x00\xea\x02\x00\x50\xe3\x07\x00\x00\x0a\x03\x00\x50\xe3\x05\x00\x00\x0a\x06\x00\x50\xe3\x03\x00\x00\x0a\x07\x00\x50\xe3\x01\x00\x00\x0a\x00\x00\xa0\xe3\x1e\xff\x2f\xe1\x01\x00\xa0\xe3\x1e\xff\x2f\xe1")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoArmEmitWideHelperCall(g *renvoLinearGen, dest int, left int, right int, mode int, label int) {
	a := &g.asm
	renvoArmAsmLeaRegStack(a, renvoArmRegRdi, dest)
	renvoArmAsmLeaRegStack(a, renvoArmRegRsi, left)
	renvoArmAsmLeaRegStack(a, renvoArmRegR8, right)
	renvoAsmPrimaryImm(a, mode)
	renvoAsmCallLabel(a, label)
}

func renvoArmEmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	renvoNonNil(g)
	if mode >= 3 && mode <= 6 {
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvoArmEmitWideHelperCall(g, dest, left, right, mode, renvoArmEnsureWideBinaryHelper(g))
}

func renvoArmEmitWideCompareStack(g *renvoLinearGen, left int, right int, mode int) {
	renvoNonNil(g)
	a := &g.asm
	renvoArmAsmLeaRegStack(a, renvoArmRegRsi, left)
	renvoArmAsmLeaRegStack(a, renvoArmRegR8, right)
	renvoAsmPrimaryImm(a, mode)
	renvoAsmCallLabel(a, renvoArmEnsureWideCompareHelper(g))
}

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

func renvoArmEmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, srcPtr)
	renvoAsmLoadSecondaryStack(a, destPtr)
	renvoAsmLoadTertiaryStack(a, byteCount)
	renvoAsmEmitText(a, "\x00\x00\x51\xe1\x03\x00\x00\x9a\x02\x00\x80\xe0\x02\x10\x81\xe0\x00\x30\xe0\xe3\x02\x00\x00\xea\x01\x00\x40\xe2\x01\x10\x41\xe2\x01\x30\xa0\xe3\x00\x00\x52\xe3\x05\x00\x00\x0a\x03\x00\x80\xe0\x03\x10\x81\xe0\x00\x90\xd0\xe5\x00\x90\xc1\xe5\x01\x20\x42\xe2\xf7\xff\xff\xea")
}

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

	// This helper has a fixed ARM instruction layout. Keeping that layout as one
	// template avoids making every ARM-hosted compiler carry the much larger code
	// generator for it; the arena call remains an ordinary label relocation.
	branch := 0xea00002f
	template := "\x00P\x94\xe5\b \x94\xe5\x02\x00U\xe1$\x00\x00\xba\x01`\xa0\xe1\x03\x80\xa0\xe1\x00\x00R\xe3\x01\x00\x00\x1a\x10 \x00\xe3\x00\x00\x00\xea\x05 \x82\xe0\x92\x06\t\xe0\x04 -\xe5\t\x00\xa0\xe1\x04\xe0-\xe5\x00\x00\x00\xeb\x04\xe0\x9d\xe4\x04 \x9d\xe4\x00\x00P\xe3\x00\x00\x00\x1a\x1e\xff/\xe1\x00\x10\xa0\xe1\x010\xa0\xe1\x00\xa0\x98\xe5\x95\x06\t\xe0\x00\x00Y\xe3\x05\x00\x00\n\x00\x00\xda\xe5\x00\x00\xc3\xe5\x01\xa0\x8a\xe2\x010\x83\xe2\x01\x90I\xe2\xf7\xff\xff\xea\x00\x10\x88\xe5\b \x84\xe5\x95\x06\t\xe0\t\x00\x81\xe0\x01\x90\x00\xe3\tP\x85\xe0\x00P\x84\xe5\x1e\xff/\xe1\x00\x00\x93\xe5\x95\x01\t\xe0\t\x00\x80\xe0\x01\x90\x00\xe3\tP\x85\xe0\x00P\x84\xe5\x1e\xff/\xe1"
	if !g.meta.panicEnabled {
		branch = 0xea00002c
		template = "\x00P\x94\xe5\b \x94\xe5\x02\x00U\xe1!\x00\x00\xba\x01`\xa0\xe1\x03\x80\xa0\xe1\x00\x00R\xe3\x01\x00\x00\x1a\x10 \x00\xe3\x00\x00\x00\xea\x05 \x82\xe0\x92\x06\t\xe0\x04 -\xe5\t\x00\xa0\xe1\x04\xe0-\xe5\x00\x00\x00\xeb\x04\xe0\x9d\xe4\x04 \x9d\xe4\x00\x10\xa0\xe1\x010\xa0\xe1\x00\xa0\x98\xe5\x95\x06\t\xe0\x00\x00Y\xe3\x05\x00\x00\n\x00\x00\xda\xe5\x00\x00\xc3\xe5\x01\xa0\x8a\xe2\x010\x83\xe2\x01\x90I\xe2\xf7\xff\xff\xea\x00\x10\x88\xe5\b \x84\xe5\x95\x06\t\xe0\t\x00\x81\xe0\x01\x90\x00\xe3\tP\x85\xe0\x00P\x84\xe5\x1e\xff/\xe1\x00\x00\x93\xe5\x95\x01\t\xe0\t\x00\x80\xe0\x01\x90\x00\xe3\tP\x85\xe0\x00P\x84\xe5\x1e\xff/\xe1"
	}
	start := len(a.code)
	renvoArmAsmEmit(a, branch)
	renvoAsmMarkLabel(a, g.appendAddrLabel)
	renvoAsmEmitText(a, template)
	renvoAsmAddReloc(a, start+64, arenaAllocLabel)
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
	renvoAsmEmit32(a, 0xe3000000|(reg<<12))
	renvoAsmEmit32(a, 0xe3400000|(reg<<12))
	renvoAsmEmit32(a, 0xe08f0000|(reg<<12)|reg)
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

func renvoArmAsmAccessRaxBss(a *renvoAsm, bssOff int, insn int) {
	at := len(a.code)
	renvoAsmEmit32(a, 0xe3000000|(renvoArmRegAddr<<12))
	renvoAsmEmit32(a, 0xe3400000|(renvoArmRegAddr<<12))
	renvoAsmEmit32(a, insn)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoArmAsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoArmAsmAccessRaxBss(a, bssOff, 0xe79f0000|renvoArmRegAddr)
}

func renvoArmAsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoArmAsmAccessRaxBss(a, bssOff, 0xe78f0000|renvoArmRegAddr)
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

func renvoArmAsmShrRaxImm(a *renvoAsm, imm int) {
	renvoArmAsmEmit(a, 0xe1a00020|(renvoArmRegRax<<12)|(imm<<7)|renvoArmRegRax)
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
