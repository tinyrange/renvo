package main

func rtg386EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
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
	g.stackPeak = 0
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	rtgAsmEmit32(a, 0x000000c8)
	if rtgTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
	}
	rtgBindFunctionParams(g, fnInfoIndex)
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
		rtgMoveCapturedLocals(g, true)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	frame := rtgAlignTo8(g.stackPeak)
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

func rtg386StoreParamWord(g *rtgLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		rtgAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return
	}
	if reg == 1 {
		rtgAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return
	}
	if reg == 2 {
		rtgAsmStoreSecondaryStack(a, offset)
		return
	}
	if reg == 3 {
		rtgAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return
	}
	if reg == 4 {
		rtgAsmStorePrimaryStack(a, offset)
		return
	}
	if reg == 5 {
		rtgAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return
	}
	rtgAsmEmit16(a, 0x858b)
	rtgAsmEmit32(a, 8+(reg-6)*4)
	rtgAsmStorePrimaryStack(a, offset)
}

func rtg386AsmMovRaxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc031)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopPrimary(a)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		rtgAsmEmit8(a, 0xb8)
		rtgAsmEmit32(a, imm)
		return
	}
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit32(a, imm)
}

func rtg386AsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAsmPrimaryImm(a, imm)
}

func rtg386AsmMovRdxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xd231)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopSecondary(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	}
	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, imm)
}

func rtg386AsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, dataOff, 0)
}

func rtg386AsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtg386AsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xbb)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtg386AsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xa1)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtg386AsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xa3)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtg386AsmMovRdiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtg386AsmMovRaxRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xd089)
}

func rtg386AsmMovRsiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc189)
}

func rtg386AsmMovR8Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtg386AsmMovR9Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtg386AsmAddRdxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xca01)
}

func rtg386AsmSyscall(a *rtgAsm) {
	rtgAsmEmit16(a, 0x80cd)
}

func rtg386AsmPopRdi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5b)
}

func rtg386AsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	if base > 0xff {
		rtgAsmEmit8(a, base>>8)
	} else {
		rtgAsmEmit8(a, base)
	}
	if offset >= 0 && offset <= 128 {
		rtgAsmEmit8(a, disp8)
		rtgAsmEmit8(a, -offset)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, -offset)
}

func rtg386AsmAddRdxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit3(a, 0x83, 0xc2, imm)
		return
	}
	rtgAsmEmit16(a, 0xc281)
	rtgAsmEmit32(a, imm)
}

func rtg386AsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	if op > 0xff {
		rtgAsmEmit8(a, op>>8)
	} else {
		rtgAsmEmit8(a, op)
	}
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, disp8, disp)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, disp)
}

func rtg386AsmJccLabel(a *rtgAsm, op int, label int) {
	rtgAsmEmit2(a, 0x0f, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtg386AsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc8048b)
}

func rtg386AsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x8b)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit3(a, 0x44, 0x8, disp)
		return
	}
	rtgAsmEmit16(a, 0x0884)
	rtgAsmEmit32(a, disp)
}

func rtg386AsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit16(a, 0x028b)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8b48, 0x42, 0x82)
}

func rtg386AsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size == 1 {
		rtgAsmEmit16(a, 0xb60f)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0xbf0f)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	rtgAsmLoadPrimaryMemSecondaryDisp(a, disp)
}

func rtg386AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 0x0804b60f)
}

func rtg386AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmLoadBytePrimaryIndexTertiary(a)
		return
	}
	if size == 2 {
		rtgAsmEmit32(a, 0x4804bf0f)
		return
	}
	if size == 4 {
		rtgAsmEmit24(a, 0x88048b)
		return
	}
	rtgAsmLoadQwordPrimaryIndexTertiary8(a)
}

func rtg386AsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAsmEmit24(a, 0xca0489)
}

func rtg386AsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit16(a, 0x0289)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8948, 0x42, 0x82)
}

func rtg386AsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size == 1 {
		rtgAsmEmit8(a, 0x88)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0x8966)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp)
}

func rtg386AsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte {
		rtgAsmEmit24(a, 0xc0b60f)
		return
	}
	if kind == rtgTypeInt8 {
		rtgAsmEmit24(a, 0xc0be0f)
		return
	}
	if kind == rtgTypeInt16 {
		rtgAsmEmit8(a, 0x98)
		return
	}
	if kind == rtgTypeUint16 {
		rtgAsmEmit24(a, 0xc0b70f)
	}
}

func rtg386AsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x02ff)
}

func rtg386AsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x0aff)
}

func rtg386AsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit3(a, 0x83, 0xf0, 1)
}

func rtg386AsmCmpRaxImm8(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc085)
		return
	}
	rtgAsmEmit3(a, 0x83, 0xf8, imm)
}

func rtg386AsmAddRaxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc801)
}

func rtg386AsmSubRaxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc829)
}

func rtg386AsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xe1, imm)
}

func rtg386AsmShlRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xe0, imm)
}

func rtg386AsmSarRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xf8, imm)
}

func rtg386AsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAsmEmit16(a, 0xc389)
	rtgAsmEmit16(a, 0xc889)
	rtgAsmEmit8(a, 0x99)
	rtgAsmEmit16(a, 0xfbf7)
	if mod {
		rtgAsmEmit16(a, 0xd089)
	}
}

func rtg386AsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAsmEmit24(a, 0x0fc139)
	rtgAsmEmit3(a, setcc, 0xc0, 0xf)
	rtgAsmEmit16(a, 0xc0b6)
}

func rtg386EmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep *rtgExprParse, idx int, matchLabel int) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmCopySecondaryToTertiary(a)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmLoadPrimaryStack(a, valueOffset)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, lenOffset)
	rtgAsmMovArg1Rax(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}

func rtg386EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
		rtgAsmAddPrimaryTertiary(a)
		return true
	}
	if c0 == '-' {
		rtgAsmEmit16(a, 0xc129)
		rtgAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		rtgAsmEmit24(a, 0xc1af0f)
		return true
	}
	if c0 == '/' {
		rtgAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if c0 == '%' {
		rtgAsmDivLeftTertiaryRightPrimary(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgAsmEmit16(a, 0xd0f7)
			rtgAsmEmit16(a, 0xc821)
		} else {
			rtgAsmEmit16(a, 0xc821)
		}
		return true
	}
	if c0 == '|' {
		rtgAsmEmit16(a, 0xc809)
		return true
	}
	if c0 == '^' {
		rtgAsmEmit16(a, 0xc831)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgAsmEmit16(a, 0xca89)
			rtgAsmEmit16(a, 0xc189)
			rtgAsmEmit16(a, 0xd089)
			rtgAsmEmit16(a, 0xe0d3)
		} else if c1 == '=' {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9e)
		} else {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAsmEmit16(a, 0xca89)
			rtgAsmEmit16(a, 0xc189)
			rtgAsmEmit16(a, 0xd089)
			rtgAsmEmit16(a, 0xf8d3)
		} else if c1 == '=' {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9d)
		} else {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgAsmCmpTertiaryPrimarySet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgAsmCmpTertiaryPrimarySet(a, 0x95)
		return true
	}
	return false
}

func rtg386EmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
	p := g.prog
	if e.tok < 0 || e.tok >= rtgTokCount(p) {
		return false
	}
	start := rtgTokStart(p, e.tok)
	end := rtgTokEnd(p, e.tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if !((c0 == '=' || c0 == '!') && c1 == '=' || c0 == '<' && c1 != '<' || c0 == '>' && c1 != '>') {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	usesFloat := rtgBinaryUsesFloat(g, ep, e)
	right := &ep.exprs[rightIndex]
	rightConst := rtgEvalConstExpr(g, ep, rightIndex)
	if !usesFloat && rightConst.ok && rtgAsmImmFits8Signed(rightConst.value) {
		if !rtgEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		rtgAsmCmpPrimaryImm8(&g.asm, rightConst.value)
		rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if c0 == '=' || c0 == '!' {
		leftType := rtgInferParsedExprType(g, ep, leftIndex)
		rightType := rtgInferParsedExprType(g, ep, rightIndex)
		leftResolved := rtgResolveType(g.meta, leftType)
		if leftResolved.kind == rtgTypeArray || leftResolved.kind == rtgTypeStruct {
			return false
		}
		if rtgTypeIsString(g.meta, leftType) || rtgTypeIsString(g.meta, rightType) {
			return false
		}
		if right.kind == rtgExprString {
			return false
		}
		if right.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, right.nameStart, right.nameEnd)
			if localIndex >= 0 {
				if rtgTypeIsString(g.meta, g.locals[localIndex].typ) {
					return false
				}
			}
		}
	}
	if !rtg386EmitCompareOperand(g, ep, rightIndex, usesFloat) {
		return false
	}
	rtgAsmPushPrimary(&g.asm)
	if !rtg386EmitCompareOperand(g, ep, leftIndex, usesFloat) {
		return false
	}
	rtgAsmPopTertiary(&g.asm)
	rtgAsmEmit16(&g.asm, 0xc139)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}

func rtg386EmitCompareOperand(g *rtgLinearGen, ep *rtgExprParse, idx int, useFloat bool) bool {
	if useFloat {
		return rtgEmitScalarExprForKind(g, ep, idx, rtgTypeFloat64)
	}
	return rtgEmitIntExpr(g, ep, idx)
}

func rtg386EmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog, e.tok)
		msgOff := rtgAddStringData(g, msg)
		msgLen := len(msg)
		rtgAsmPrimaryDataAddr(a, msgOff)
		rtgAsmSecondaryImm(a, msgLen)
		return true
	}
	if e.kind == rtgExprSlice {
		return rtgEmitStringSliceValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadPrimaryBss(a, globalOffset)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset+8)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopPrimary(a)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog, constTok)
			msgOff := rtgAddStringData(g, msg)
			msgLen := len(msg)
			rtgAsmPrimaryDataAddr(a, msgOff)
			rtgAsmSecondaryImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmShlTertiaryImm(a, 4)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		rtgAsmAddSecondaryTertiary(a)
		rtgAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtg386EmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if g.returnStruct <= 0 {
		return false
	}
	e := &ep.exprs[idx]
	resultType := g.meta.funcs[g.currentFunc].resultType
	size := rtgTypeSize(meta, resultType)
	rtgAsmLoadSecondaryStack(a, g.returnStruct)
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		rtgEmitCopyStackToMemSecondary(g, g.locals[localIndex].offset, 0, size)
		return true
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmMulTertiaryImm(a, size)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmAddSecondaryTertiary(a)
		rtgAsmLoadTertiaryStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadPrimaryMemSecondaryDisp(a, at)
			if at == 0 {
				rtgAsmEmit16(a, 0x0189)
			} else {
				rtgAsmMemDisp(a, at, 0x8948, 0x41, 0x81)
			}
		}
		return true
	}
	if e.kind == rtgExprComposite {
		rtgAsmPrimaryImm(a, 0)
		for at := 0; at < size; at += 8 {
			rtgAsmStorePrimaryMemSecondaryDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 || !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex, wordCount := rtgPrepareStructCall(g, ep, idx, resultType)
		if fnIndex < 0 {
			return false
		}
		rtgAsmLoadPrimaryStack(a, g.returnStruct)
		rtgAsmPushPrimary(a)
		rtgEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}

func rtg386EmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	calleeExpr := &ep.exprs[e.left]
	if calleeExpr.kind != rtgExprIdent {
		return false
	}
	namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
	resolved := rtgResolveType(g.meta, namedType)
	if resolved.kind == rtgTypeString {
		return rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg])
	}
	resolvedKind := resolved.kind
	if rtgTypeKindIsScalarInt(resolvedKind) {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
			return false
		}
		rtgAsmNormalizePrimaryForKind(&g.asm, resolvedKind)
		return true
	}
	return false
}

func rtg386EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgAsmPopCallWord0(a)
	}
	if wordCount > 1 {
		rtgAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		rtgAsmPopSecondary(a)
	}
	if wordCount > 3 {
		rtgAsmPopTertiary(a)
	}
	if wordCount > 4 {
		rtgAsmPopPrimary(a)
	}
	if wordCount > 5 {
		rtgAsmEmit8(a, 0x5f)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if rtgAsmImmFits8Signed(imm) {
			rtgAsmEmit3(a, 0x83, 0xc4, imm)
		} else {
			rtgAsmEmit16(a, 0xc481)
			rtgAsmEmit32(a, imm)
		}
	}
}

func rtg386EmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == rtgExprUnary || e.kind == rtgExprBinary || e.kind == rtgExprCall) && rtgExprCanFoldConst(g, ep, idx) {
		resultType := rtgInferParsedExprType(g, ep, idx)
		result := rtgResolveType(g.meta, resultType)
		if result.kind != rtgTypeByte && result.kind != rtgTypeInt8 && result.kind != rtgTypeInt16 && result.kind != rtgTypeInt32 && result.kind != rtgTypeUint16 && result.kind != rtgTypeUint32 {
			constResult := rtgEvalConstExpr(g, ep, idx)
			if constResult.ok {
				rtgAsmPrimaryImm(a, constResult.value)
				return true
			}
		}
	}
	if e.kind == rtgExprInt {
		rtgAsmLoadPrimaryIntToken(a, p, e.tok)
		return true
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		rtgAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					fnIndex := rtgFindMetaFunction(g.meta, e.nameStart, e.nameEnd)
					if fnIndex < 0 {
						return false
					}
					rtgAsmPrimaryImm(a, fnIndex+1)
					return true
				}
				rtgAsmLoadPrimaryBss(a, globalOffset)
				return true
			}
			rtgAsmPrimaryImm(a, constResult.value)
			return true
		}
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == rtgExprCall {
		if rtgExprIsIdentText(p, ep, e.left, "rtgTrustNonNil") {
			return rtgEmitRuntimeTrustPointer(g, ep, e)
		}
		if rtgExprIsIdentText(p, ep, e.left, "rtg_runtime_UnsafeByteAt") {
			return rtgEmitRuntimeUnsafeIndex(g, ep, e, 1)
		}
		if rtgExprIsIdentText(p, ep, e.left, "rtg_runtime_UnsafeInt32At") {
			return rtgEmitRuntimeUnsafeIndex(g, ep, e, 4)
		}
		if rtgExprIsIdentText(p, ep, e.left, "rtg_runtime_UnsafeIntAt") {
			return rtgEmitRuntimeUnsafeIndex(g, ep, e, rtgNativeIntSize)
		}
		if rtgExprIsIdentText(p, ep, e.left, "rtgTruncateBytes") || rtgExprIsIdentText(p, ep, e.left, "rtgTruncateParams") || rtgExprIsIdentText(p, ep, e.left, "rtgTruncateTypes") || rtgExprIsIdentText(p, ep, e.left, "rtgTruncateFields") {
			return rtgEmitRuntimeTruncateSlice(g, ep, e)
		}
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee != rtgIdentSyscall && (rtgFunctionValueCalleeType(g, ep, e.left) != 0 || rtgFuncInfoFromCall(g, ep, e.left) >= 0) {
			return rtgEmitUserCall(g, ep, idx)
		}
		if e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "Sizeof") {
			rtgAsmPrimaryImm(a, rtgTypeSize(g.meta, rtgInferParsedExprType(g, ep, ep.args[e.firstArg])))
			return true
		}
		if callee == rtgIdentPanic {
			return rtgEmitBuiltinPanic(g, ep, idx)
		}
		if callee == rtgIdentSyscall {
			return rtgEmitArbitrarySyscall(g, ep, idx)
		}
		if callee == rtgIdentNew {
			return rtgEmitBuiltinNew(g, ep, idx)
		}
		if e.argCount == 1 {
			conversionType := rtgConversionTypeFromExpr(g, ep, e.left)
			conversion := rtgResolveType(g.meta, conversionType)
			if rtgTypeKindIsScalarValue(conversion.kind) {
				return rtgEmitScalarExprForKind(g, ep, ep.args[e.firstArg], conversion.kind)
			}
		}
		if e.argCount == 1 && (callee == rtgIdentCap || callee == rtgIdentLen) {
			count := rtgArrayBuiltinCount(g, ep, e)
			if count >= 0 {
				rtgAsmPrimaryImm(a, count)
				return true
			}
		}
		if e.argCount == 1 && callee == rtgIdentCap {
			if !rtgEmitSlicePtrCap(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmEmit16(a, 0x5851)
			return true
		}
		if e.argCount == 1 && callee == rtgIdentLen {
			arg := &ep.exprs[ep.args[e.firstArg]]
			if arg.kind == rtgExprString {
				msg := rtgDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				rtgAsmPrimaryImm(a, msgLen)
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) || rtgTypeIsString(g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := rtgFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (rtgTypeIsString(g.meta, globalType) || rtgTypeIsSlice(g.meta, globalType)) {
					rtgAsmLoadPrimaryBss(a, globalOffset+8)
					return true
				}
				constTok := rtgFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := rtgDecodeStringToken(p, constTok)
					msgLen := len(msg)
					rtgAsmPrimaryImm(a, msgLen)
					return true
				}
			}
			if arg.kind == rtgExprUnary && rtgTokCharIs(p, arg.tok, '*') {
				if !rtgEmitIntExpr(g, ep, arg.left) {
					return false
				}
				rtgAsmCopyPrimaryToSecondary(a)
				rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
				return true
			}
			argIndex := ep.args[e.firstArg]
			if rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, argIndex)) {
				if !rtgEmitStringValueRegs(g, ep, argIndex) {
					return false
				}
				rtgAsmPushSecondary(a)
				rtgAsmPopPrimary(a)
				return true
			}
			if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmEmit16(a, 0x5851)
			return true
		}
		if callee == rtgIdentOpen {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsOpen(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPopTertiary(a)
			rtgAsmSecondaryImm(a, 493)
			rtgAsmPrimaryImm(a, rtgLinuxSysOpen())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentClose {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsClose(g, ep, idx)
			}
			if e.argCount != 1 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPrimaryImm(a, rtgLinuxSysClose())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentChmod {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsChmod(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmCopyPrimaryToCallWord1(a)
			rtgAsmPopCallWord0(a)
			rtgAsmPrimaryImm(a, rtgLinuxSysFchmod())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentRead {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsReadWrite(g, ep, idx, false)
			}
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysReadSeq(), rtgLinuxSysReadAt())
		}
		if callee == rtgIdentWrite {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsReadWrite(g, ep, idx, true)
			}
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysWriteSeq(), rtgLinuxSysWriteAt())
		}
		if callee == rtgIdentCopy {
			return rtgEmitBuiltinCopy(g, ep, idx)
		}
		return rtgEmitUserCall(g, ep, idx)
	}
	if e.kind == rtgExprIndex {
		return rtgEmitIndexExpr(g, ep, idx)
	}
	if e.kind == rtgExprSelector {
		baseType := rtgInferParsedExprType(g, ep, e.left)
		nativeABI := rtgTypeUsesNativeABI(g.meta, baseType)
		fieldType := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		fieldSize := rtgNativeScalarStorageSize(fieldType.kind)
		base := &ep.exprs[e.left]
		if base.kind == rtgExprCall {
			baseResolved := rtgResolveType(g.meta, baseType)
			if baseResolved.kind == rtgTypePointer {
				if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
					return false
				}
				rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
				if nativeABI {
					rtgAsmNormalizePrimaryForKind(a, fieldType.kind)
				}
				return true
			}
			if !rtgTypeIsStruct(g.meta, baseType) {
				return false
			}
			fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			offset := rtgAddTypedLocal(g, 0, 0, baseType)
			if !rtgEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			rtgAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
			rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
			if nativeABI {
				rtgAsmNormalizePrimaryForKind(a, fieldType.kind)
			}
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
		if nativeABI {
			rtgAsmNormalizePrimaryForKind(a, fieldType.kind)
		}
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '&') {
			inner := &ep.exprs[e.left]
			if inner.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex >= 0 {
					rtgInvalidateCheckedPointerLocal(g, localIndex)
					rtgAsmStackMem(a, g.locals[localIndex].offset, 0x8d48, 0x45, 0x85)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					rtgAsmPrimaryBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressSecondary(g, ep, e.left) {
					return false
				}
				rtgAsmEmit16(a, 0x5852)
				return true
			}
			if inner.kind == rtgExprIndex {
				return rtgEmitIndexAddressPrimary(g, ep, e.left)
			}
			return false
		}
		if rtgTokCharIs(p, e.tok, '*') {
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgEmitRuntimeNonNilPrimary(g)
			rtgAsmCopyPrimaryToSecondary(a)
			targetKind := rtgPointerTargetKind(g, ep, e.left)
			size := rtgScalarKindSize(targetKind)
			rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
			return true
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		if rtgTokCharIs(p, e.tok, '-') {
			rtgAsmEmit16(a, 0xd8f7)
			resultType := rtgInferParsedExprType(g, ep, idx)
			result := rtgResolveType(g.meta, resultType)
			rtgAsmNormalizePrimaryForKind(a, result.kind)
			return true
		}
		if rtgTokCharIs(p, e.tok, '+') {
			resultType := rtgInferParsedExprType(g, ep, idx)
			result := rtgResolveType(g.meta, resultType)
			rtgAsmNormalizePrimaryForKind(a, result.kind)
			return true
		}
		if rtgTokCharIs(p, e.tok, '!') {
			rtgAsmBoolNotPrimary(a)
			return true
		}
		return false
	}
	if e.kind == rtgExprBinary {
		if rtgBinaryUsesFloat(g, ep, e) {
			return rtgEmitFloatBinaryExpr(g, ep, idx)
		}
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') {
			leftType := rtgInferParsedExprType(g, ep, e.left)
			leftResolved := rtgResolveType(g.meta, leftType)
			if leftResolved.kind == rtgTypeArray || leftResolved.kind == rtgTypeStruct {
				return rtgEmitCompositeCompare(g, ep, e, leftType)
			}
		}
		if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			rtgAsmPrimaryImm(a, 1)
			rtgAsmJmpMarkLabel(a, endLabel, falseLabel)
			rtgAsmPrimaryImm(a, 0)
			rtgAsmMarkLabel(a, endLabel)
			return true
		}
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') {
			leftType := rtgInferParsedExprType(g, ep, e.left)
			rightType := rtgInferParsedExprType(g, ep, e.right)
			if rtgTypeIsString(g.meta, leftType) || rtgTypeIsString(g.meta, rightType) {
				notEqual := rtgTok2Is(p, e.tok, '!', '=')
				return rtgEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := &ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushPrimary(a)
		if rightKind == rtgExprInt {
			rtgAsmLoadPrimaryIntToken(a, p, rightTok)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			rtgAsmPrimaryImm(a, value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			rtgAsmPrimaryImm(a, value)
		} else {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
		}
		rtgAsmPopTertiary(a)
		if !rtgEmitPrimaryTertiaryOp(g, e.tok) {
			return false
		}
		resultType := rtgInferParsedExprType(g, ep, idx)
		result := rtgResolveType(g.meta, resultType)
		rtgAsmNormalizePrimaryForKind(a, result.kind)
		return true
	}
	return false
}

func rtg386EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmEmit24(a, 0xc1af0f)
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

func rtg386EmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !rtgEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		rtgAsmEmit16(a, 0x5f52)
		rtgAsmEmit16(a, 0x728d)
		rtgAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		rtgAsmEmit16(a, 0x3d8d)
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset, rtgAbsBssReloc)
		rtgAsmEmit16(a, 0x358d)
		at = len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset+8, rtgAbsBssReloc)
		return true
	}
	rtgAsmAddressCallWord0Stack(a, loc.offset)
	rtgAsmAddressCallWord1Stack(a, loc.offset-8)
	return true
}

func rtg386EmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
	rtgAsmPrimaryImm(a, backingSize/elemSize)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 16)
	rtgAsmMarkLabel(a, okLabel)
}

func rtg386EnsureAppendAddrHelper(g *rtgLinearGen) int {
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
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit3(a, 0x8b, 0x46, 8)
	rtgAsmEmit16(a, 0xc139)
	rtg386AsmJccLabel(a, 0x8c, noGrowLabel)
	rtgAsmEmit8(a, 0x57)
	rtgAsmEmit8(a, 0x56)
	rtgAsmPushSecondary(a)
	rtgAsmPushTertiary(a)
	rtgAsmEmit3(a, 0x8b, 0x46, 8)
	rtgAsmEmit16(a, 0xc085)
	rtg386AsmJccLabel(a, 0x85, capNonZeroLabel)
	rtgAsmPrimaryImm(a, 16)
	rtgAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmMarkLabel(a, capReadyLabel)
	rtgAsmPushPrimary(a)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmEmit3(a, 0x0f, 0xaf, 0x4c)
	rtgAsmEmit2(a, 0x24, 8)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmCallLabel(a, arenaAllocLabel)
	rtgAsmPushPrimary(a)
	rtgAsmEmit3(a, 0x8b, 0x3c, 0x24)
	rtgAsmEmit4(a, 0x8b, 0x74, 0x24, 20)
	rtgAsmEmit16(a, 0x368b)
	rtgAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 12)
	rtgAsmEmit3(a, 0x0f, 0xaf, 0xc8)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmEmit4(a, 0x8b, 0x7c, 0x24, 20)
	rtgAsmEmit3(a, 0x8b, 0x04, 0x24)
	rtgAsmEmit16(a, 0x0789)
	rtgAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 4)
	rtgAsmEmit3(a, 0x89, 0x46, 8)
	rtgAsmEmit3(a, 0x8b, 0x04, 0x24)
	rtgAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	rtgAsmEmit4(a, 0x8b, 0x54, 0x24, 12)
	rtgAsmEmit3(a, 0x0f, 0xaf, 0xca)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	rtgAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	rtgAsmIncTertiary(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmEmit3(a, 0x83, 0xc4, 24)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, noGrowLabel)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmIncTertiary(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtg386EnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0x081488)
	rtgAsmIncTertiary(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtg386EnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xc81489)
	rtgAsmIncTertiary(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtg386EnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmEmit16(a, 0xce39)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit16(a, 0xf685)
	rtgAsmJzLabel(a, equalLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit16(a, 0x0b8a)
	rtgAsmEmit16(a, 0x0a38)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit8(a, 0x43)
	rtgAsmEmit8(a, 0x42)
	rtgAsmEmit8(a, 0x4e)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmPrimaryImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtg386EmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(g.meta, sliceType.elem)
	if elemType.kind != rtgTypeStruct && elemType.kind != rtgTypePointer {
		return false
	}
	fieldOffset := rtgStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	fieldType := rtgStructFieldType(g, sliceType.elem, fieldStart, fieldEnd)
	if !rtgEmitIndexedSelectorAddressSecondary(g, ep, indexIdx, fieldOffset) {
		return false
	}
	fieldResolved := rtgResolveType(g.meta, fieldType)
	fieldSize := rtgScalarKindSize(fieldResolved.kind)
	rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
	return true
}

func rtg386EmitStringPtrExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(p, e.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmPrimaryDataAddr(a, msgOff)
		return true
	}
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		arg := &ep.exprs[argIndex]
		if arg.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		return rtgEmitStringValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(p, constTok)
			msgOff := rtgAddStringData(g, msg)
			rtgAsmPrimaryDataAddr(a, msgOff)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		rtgAsmPopTertiary(a)
		rtgAsmShlTertiaryImm(a, 4)
		rtgAsmEmit24(a, 0x08048b)
		return true
	}
	return false
}

func rtg386EmitSelectorAddressRdx(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	base := &ep.exprs[e.left]
	baseType := rtgInferParsedExprType(g, ep, e.left)
	fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if rtgStructPromotedPointerField(g, baseType, e.nameStart, e.nameEnd) >= 0 {
		return rtgEmitPromotedPointerSelectorAddress(g, ep, idx, baseType)
	}
	if base.kind == rtgExprCall {
		baseResolved := rtgResolveType(meta, baseType)
		if baseResolved.kind != rtgTypePointer && baseResolved.kind != rtgTypeStruct {
			return false
		}
		if baseResolved.kind == rtgTypePointer {
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgEmitRuntimeNonNilPrimary(g)
			rtgAsmCopyPrimaryToSecondary(a)
			if fieldOffset != 0 {
				rtgAsmAddSecondaryImm(a, fieldOffset)
			}
			return true
		}
	}
	if base.kind == rtgExprComposite || base.kind == rtgExprCall {
		offset := rtgAddUnnamedLocal(g, baseType)
		if !rtgEmitTypedAssign(g, ep, e.left, offset) {
			return false
		}
		rtgAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if base.kind == rtgExprIndex {
		return rtgEmitIndexedSelectorAddressSecondary(g, ep, e.left, fieldOffset)
	}
	if base.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := rtgFindGlobalType(g, base.nameStart, base.nameEnd)
			t := rtgResolveType(meta, globalType)
			if globalOffset < 0 {
				return false
			}
			if t.kind == rtgTypePointer {
				rtgAsmLoadPrimaryBss(a, globalOffset)
				rtgEmitRuntimeNonNilPrimary(g)
				rtgAsmCopyPrimaryToSecondary(a)
				if fieldOffset != 0 {
					rtgAsmAddSecondaryImm(a, fieldOffset)
				}
				return true
			}
			if t.kind != rtgTypeStruct {
				return false
			}
			rtgAsmPrimaryBssAddr(a, globalOffset)
			rtgAsmCopyPrimaryToSecondary(a)
			if fieldOffset != 0 {
				rtgAsmAddSecondaryImm(a, fieldOffset)
			}
			return true
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			rtgEmitRuntimeNonNilLocalSecondary(g, localIndex)
			if fieldOffset != 0 {
				rtgAsmAddSecondaryImm(a, fieldOffset)
			}
			return true
		}
		rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if base.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressSecondary(g, ep, e.left) {
			return false
		}
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			rtgAsmEmit16(a, 0x128b)
			rtgEmitRuntimeNonNilSecondary(g)
		}
		if fieldOffset != 0 {
			rtgAsmAddSecondaryImm(a, fieldOffset)
		}
		return true
	}
	return false
}

func rtgAsmMovArg1Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc689)
}
