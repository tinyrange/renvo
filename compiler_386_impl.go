package main

func rtg386EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldStackUsed := g.stackUsed
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	g.locals = locals
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	rtgAsmEmit32(a, 0x008000c8)
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
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
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.stackUsed = oldStackUsed
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func rtg386StoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return true
	}
	if reg == 1 {
		rtgAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return true
	}
	if reg == 2 {
		rtgAsmStoreRdxStack(a, offset)
		return true
	}
	if reg == 3 {
		rtgAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return true
	}
	if reg == 4 {
		rtgAsmStoreRaxStack(a, offset)
		return true
	}
	if reg == 5 {
		rtgAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return true
	}
	rtgAsmEmit16(a, 0x858b)
	rtgAsmEmit32(a, 8+(reg-6)*4)
	rtgAsmStoreRaxStack(a, offset)
	return true
}

func rtg386AsmMovRaxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc031)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRax(a)
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
	rtgAsmMovRaxImm(a, imm)
}

func rtg386AsmMovRdxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xd231)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRdx(a)
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
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0xbf0f)
		rtgAsmRdxDisp(a, disp)
		return
	}
	rtgAsmLoadRaxMemRdxDisp(a, disp)
}

func rtg386AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 0x0804b60f)
}

func rtg386AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmLoadByteRaxIndexRcx(a)
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
	rtgAsmLoadQwordRaxIndexRcx8(a)
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
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0x8966)
		rtgAsmRdxDisp(a, disp)
		return
	}
	rtgAsmStoreRaxMemRdxDisp(a, disp)
}

func rtg386AsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		rtgAsmEmit8(a, 0x25)
		rtgAsmEmit32(a, 0xff)
		return
	}
	if kind == rtgTypeInt16 {
		rtgAsmEmit8(a, 0x98)
	}
}

func rtg386AsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x02ff)
}

func rtg386AsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x0aff)
}

func rtg386AsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc085)
	rtgAsmEmit24(a, 0xc0940f)
	rtgAsmEmit24(a, 0xc0b60f)
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
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmLoadRaxStack(a, valueOffset)
	rtgAsmMovRdiRax(a)
	rtgAsmLoadRaxStack(a, lenOffset)
	rtgAsmMovArg1Rax(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}

func rtg386EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= len(p.toks) {
		return false
	}
	start := p.toks[tok].start
	end := p.toks[tok].end
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		rtgAsmAddRaxRcx(a)
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
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgAsmDivLeftRcxRightRax(a, true)
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
			rtgAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9c)
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
			rtgAsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtg386EmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
	p := g.prog
	if e.tok < 0 || e.tok >= len(p.toks) {
		return false
	}
	start := p.toks[e.tok].start
	end := p.toks[e.tok].end
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	isCompare := false
	if c0 == '=' && c1 == '=' {
		isCompare = true
	} else if c0 == '!' && c1 == '=' {
		isCompare = true
	} else if c0 == '<' && c1 != '<' {
		isCompare = true
	} else if c0 == '>' && c1 != '>' {
		isCompare = true
	}
	if !isCompare {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	right := &ep.exprs[rightIndex]
	rightValue := 0
	rightOK := false
	if right.kind == rtgExprInt {
		rightValue = rtgParseIntToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprChar {
		rightValue = rtgParseCharToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprBool {
		rightValue = rtgBoolTokenValue(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprIdent {
		rightValue = rtgFindSmallConstByName(g, right.nameStart, right.nameEnd)
		rightOK = rightValue >= -128
	}
	if rightOK && rtgAsmImmFits8Signed(rightValue) {
		if !rtgEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		rtgAsmCmpRaxImm8(&g.asm, rightValue)
		rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if c0 == '=' || c0 == '!' {
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
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitIntExpr(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	rtgAsmEmit16(&g.asm, 0xc139)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}

func rtg386EmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog, e.tok)
		msgOff := rtgAddStringData(g, msg)
		msgLen := len(msg)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmMovRdxImm(a, msgLen)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog, constTok)
			msgOff := rtgAddStringData(g, msg)
			msgLen := len(msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
			rtgAsmMovRdxImm(a, msgLen)
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
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
		rtgAsmMovRdxRax(a)
		rtgAsmEmit24(a, 0x08048b)
		rtgAsmAddRdxRcx(a)
		rtgAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
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

func rtg386EmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushSliceRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreSliceMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushStringRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreStringMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, fieldResolved.kind)
	rtgAsmLoadRdxStack(a, addrOffset)
	fieldSize := rtgScalarKindSize(fieldResolved.kind)
	rtgAsmStoreRaxMemRdxDispSize(a, fieldOffset, fieldSize)
	return true
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
	rtgAsmLoadRdxStack(a, g.returnStruct)
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, size)
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
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgAsmLoadRcxStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxMemRdxDisp(a, at)
			if at == 0 {
				rtgAsmEmit16(a, 0x0189)
			} else {
				rtgAsmMemDisp(a, at, 0x8948, 0x41, 0x81)
			}
		}
		return true
	}
	if e.kind == rtgExprComposite {
		rtgAsmMovRaxImm(a, 0)
		for at := 0; at < size; at += 8 {
			rtgAsmStoreRaxMemRdxDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 {
				return false
			}
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsStruct(meta, meta.funcs[fnIndex].resultType) {
			return false
		}
		if rtgTypeSize(meta, meta.funcs[fnIndex].resultType) != size {
			return false
		}
		wordCount := 1
		for i := e.argCount - 1; i >= 0; i-- {
			argIndex := ep.args[e.firstArg+i]
			words := rtgEmitCallArgReverse(g, ep, argIndex)
			if words < 0 {
				return false
			}
			wordCount += words
		}
		rtgAsmLoadRaxStack(a, g.returnStruct)
		rtgAsmPushRax(a)
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
		rtgAsmNormalizeRaxForKind(&g.asm, resolvedKind)
		return true
	}
	return false
}

func rtg386EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgAsmPopRdi(a)
	}
	if wordCount > 1 {
		rtgAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		rtgAsmPopRdx(a)
	}
	if wordCount > 3 {
		rtgAsmPopRcx(a)
	}
	if wordCount > 4 {
		rtgAsmPopRax(a)
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
		constResult := rtgEvalConstExpr(g, ep, idx)
		if constResult.ok {
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
	}
	if e.kind == rtgExprInt {
		rtgAsmMovRaxIntToken(a, p, e.tok)
		return true
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					return false
				}
				rtgAsmLoadRaxBss(a, globalOffset)
				return true
			}
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentInt64) {
			return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 1 && (callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32) {
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			if callee == rtgIdentByte {
				rtgAsmNormalizeRaxForKind(a, rtgTypeByte)
			} else if callee == rtgIdentInt16 {
				rtgAsmNormalizeRaxForKind(a, rtgTypeInt16)
			} else {
				rtgAsmNormalizeRaxForKind(a, rtgTypeInt32)
			}
			return true
		}
		if e.argCount == 1 && callee == rtgIdentLen {
			arg := &ep.exprs[ep.args[e.firstArg]]
			if arg.kind == rtgExprString {
				msg := rtgDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				rtgAsmMovRaxImm(a, msgLen)
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) || rtgTypeIsString(g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadRaxStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := rtgFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (rtgTypeIsString(g.meta, globalType) || rtgTypeIsSlice(g.meta, globalType)) {
					rtgAsmLoadRaxBss(a, globalOffset+8)
					return true
				}
				constTok := rtgFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := rtgDecodeStringToken(p, constTok)
					msgLen := len(msg)
					rtgAsmMovRaxImm(a, msgLen)
					return true
				}
			}
			if arg.kind == rtgExprUnary && rtgTokCharIs(p, arg.tok, '*') {
				if !rtgEmitIntExpr(g, ep, arg.left) {
					return false
				}
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRaxMemRdxDisp(a, 8)
				return true
			}
			if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmEmit16(a, 0x5851)
			return true
		}
		if callee == rtgIdentOpen {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmPopRcx(a)
			rtgAsmMovRdxImm(a, 493)
			rtgAsmMovRaxImm(a, rtgLinuxSysOpen())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentClose {
			if e.argCount != 1 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmMovRaxImm(a, rtgLinuxSysClose())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentChmod {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmMovRsiRax(a)
			rtgAsmPopRdi(a)
			rtgAsmMovRaxImm(a, rtgLinuxSysFchmod())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentRead {
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysReadSeq(), rtgLinuxSysReadAt())
		}
		if callee == rtgIdentWrite {
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
		base := &ep.exprs[e.left]
		if base.kind == rtgExprCall {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			if !rtgTypeIsStruct(g.meta, baseType) {
				return false
			}
			fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			fieldType := rtgStructFieldType(g, baseType, e.nameStart, e.nameEnd)
			offset := rtgAddTypedLocal(g, 0, 0, baseType)
			if !rtgEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			rtgAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
			fieldResolved := rtgResolveType(g.meta, fieldType)
			fieldSize := rtgScalarKindSize(fieldResolved.kind)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, fieldSize)
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		fieldType := rtgInferParsedExprType(g, ep, idx)
		fieldResolved := rtgResolveType(g.meta, fieldType)
		fieldSize := rtgScalarKindSize(fieldResolved.kind)
		rtgAsmLoadRaxMemRdxDispSize(a, 0, fieldSize)
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '&') {
			inner := &ep.exprs[e.left]
			if inner.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex >= 0 {
					rtgAsmStackMem(a, g.locals[localIndex].offset, 0x8d48, 0x45, 0x85)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					rtgAsmMovRaxBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
					return false
				}
				rtgAsmEmit16(a, 0x5852)
				return true
			}
			if inner.kind == rtgExprIndex {
				return rtgEmitIndexAddressRax(g, ep, e.left)
			}
			return false
		}
		if rtgTokCharIs(p, e.tok, '*') {
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmMovRdxRax(a)
			targetKind := rtgPointerTargetKind(g, ep, e.left)
			size := rtgScalarKindSize(targetKind)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, size)
			return true
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		if rtgTokCharIs(p, e.tok, '-') {
			rtgAsmEmit16(a, 0xd8f7)
			return true
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return true
		}
		if rtgTokCharIs(p, e.tok, '!') {
			rtgAsmBoolNotRax(a)
			return true
		}
		return false
	}
	if e.kind == rtgExprBinary {
		if rtgBinaryUsesFloat(g, ep, e) {
			return rtgEmitFloatBinaryExpr(g, ep, idx)
		}
		if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			rtgAsmMovRaxImm(a, 1)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, falseLabel)
			rtgAsmMovRaxImm(a, 0)
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
		rtgAsmPushRax(a)
		if rightKind == rtgExprInt {
			rtgAsmMovRaxIntToken(a, p, rightTok)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, e.tok) {
			return false
		}
		return true
	}
	return false
}

func rtg386EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmEmit24(a, 0xc1af0f)
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

func rtg386EmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
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
	rtgEmitEnsureLocalSlice(g, loc.offset, elemSize)
	rtgAsmLeaRdiStack(a, loc.offset)
	rtgAsmLeaRsiStack(a, loc.offset-8)
	return true
}

func rtg386EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmAddRaxRcx(a)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmIncRcx(a)
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
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0x081488)
	rtgAsmIncRcx(a)
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
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xc81489)
	rtgAsmIncRcx(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtg386EnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x3f8b)
	rtgAsmEmit16(a, 0xcf01)
	rtgAsmEmit16(a, 0x1601)
	rtgAsmEmit16(a, 0xc689)
	rtgAsmEmit16(a, 0xd189)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

func rtg386EnsureCopyWordsHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.copyWordsEmitted {
		return g.copyWordsLabel
	}
	g.copyWordsEmitted = true
	g.copyWordsLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.copyWordsLabel)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit16(a, 0xd285)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit16(a, 0x068b)
	rtgAsmEmit16(a, 0x0789)
	rtgAsmEmit24(a, 0x04468b)
	rtgAsmEmit24(a, 0x044789)
	rtgAsmEmit3(a, 0x83, 0xc6, 8)
	rtgAsmEmit3(a, 0x83, 0xc7, 8)
	rtgAsmEmit8(a, 0x4a)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}

func rtg386EnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.streqLabel)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmMovRaxImm(a, 0)
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
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtg386EmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftIndex := indexExpr.left
	rightIndex := indexExpr.right
	leftType := rtgInferParsedExprType(g, ep, leftIndex)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(g.meta, sliceType.elem)
	if elemType.kind != rtgTypeStruct {
		return false
	}
	fieldOffset := rtgStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	fieldType := rtgStructFieldType(g, sliceType.elem, fieldStart, fieldEnd)
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(a)
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmAddRaxRcx(a)
	rtgAsmMovRdxRax(a)
	fieldResolved := rtgResolveType(g.meta, fieldType)
	fieldSize := rtgScalarKindSize(fieldResolved.kind)
	rtgAsmLoadRaxMemRdxDispSize(a, fieldOffset, fieldSize)
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
		rtgAsmMovRaxDataAddr(a, msgOff)
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
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
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
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(p, constTok)
			msgOff := rtgAddStringData(g, msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
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
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
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
	if base.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, base.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct {
			return false
		}
		if !rtgEmitIntExpr(g, ep, base.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, base.left) {
			return false
		}
		rtgAsmPopRcx(a)
		elemSize := rtgTypeSize(meta, sliceType.elem)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
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
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmMovRdxRax(a)
				if fieldOffset != 0 {
					rtgAsmAddRdxImm(a, fieldOffset)
				}
				return true
			}
			if t.kind != rtgTypeStruct {
				return false
			}
			rtgAsmMovRaxBssAddr(a, globalOffset)
			rtgAsmMovRdxRax(a)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if base.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
			return false
		}
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			rtgAsmEmit16(a, 0x128b)
		}
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	return false
}

func rtgAsmMovArg1Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc689)
}
