package main

func rtgAmd64EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
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
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	rtgAsmEmit32(a, 0x000000c8)
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStackMem(a, g.returnStruct, 0x8948, 0x7d, 0xbd)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if !g.lastRangeReturns {
		rtgAsmPrimaryImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	frame := rtgAlignValue(g.stackUsed+2048, 16)
	if frame < 2048 {
		frame = 2048
	}
	if frame > 65520 {
		frame = 65520
	}
	a.code[framePatch+1] = byte(frame & 255)
	a.code[framePatch+2] = byte((frame / 256) & 255)
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

func rtgAmd64StoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgAsmStackMem(a, offset, 0x8948, 0x7d, 0xbd)
		return true
	}
	if reg == 1 {
		rtgAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return true
	}
	if reg == 2 {
		rtgAsmStoreSecondaryStack(a, offset)
		return true
	}
	if reg == 3 {
		rtgAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return true
	}
	if reg == 4 {
		rtgAsmStackMem(a, offset, 0x894c, 0x45, 0x85)
		return true
	}
	if reg == 5 {
		rtgAsmStackMem(a, offset, 0x894c, 0x4d, 0x8d)
		return true
	}
	rtgAsmEmit24(a, 0x858b48)
	rtgAsmEmit32(a, 0x10+(reg-6)*8)
	rtgAsmStorePrimaryStack(a, offset)
	return true
}
func rtgAmd64AsmMovRaxImm(a *rtgAsm, imm int) {
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
		rtgAsmEmit8(a, 0x68)
		rtgAsmEmit32(a, imm)
		rtgAsmPopPrimary(a)
		return
	}
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}
func rtgAmd64AsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 0x158d4c)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}
func rtgAmd64AsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 0x058b48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}
func rtgAmd64AsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 0x058948)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}
func rtgAmd64AsmMovRdiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0x5f50)
}
func rtgAmd64AsmMovRaxRdx(a *rtgAsm) {
	rtgAsmEmit24(a, 0xd08948)
}
func rtgAmd64AsmMovRsiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0x5e50)
}
func rtgAmd64AsmMovR8Rax(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc08949)
}
func rtgAmd64AsmMovR9Rax(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc18949)
}
func rtgAmd64AsmAddRdxRcx(a *rtgAsm) {
	rtgAsmEmit24(a, 0xca0148)
}
func rtgAmd64AsmSyscall(a *rtgAsm) {
	rtgAsmEmit16(a, 0x050f)
}
func rtgAmd64AsmPopRdi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5f)
}
func rtgAmd64AsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	rtgAsmEmit16(a, base)
	if offset >= 0 && offset <= 128 {
		rtgAsmEmit8(a, disp8)
		rtgAsmEmit8(a, -offset)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, -offset)
}
func rtgAmd64AsmAddRdxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit4(a, 0x48, 0x83, 0xc2, imm)
		return
	}
	rtgAsmEmit24(a, 0xc28148)
	rtgAsmEmit32(a, imm)
}
func rtgAmd64AsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	rtgAsmEmit16(a, op)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, disp8, disp)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, disp)
}
func rtgAmd64AsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAsmEmit32(a, 0xc8048b48)
}
func rtgAmd64AsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit16(a, 0x8b48)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit3(a, 0x44, 0x8, disp)
		return
	}
	rtgAsmEmit16(a, 0x0884)
	rtgAsmEmit32(a, disp)
}
func rtgAmd64AsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit24(a, 0x028b48)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8b48, 0x42, 0x82)
}
func rtgAmd64AsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size == 1 {
		rtgAsmEmit24(a, 0xb60f48)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit24(a, 0xbf0f48)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	if size == 4 {
		rtgAsmMemDisp(a, disp, 0x6348, 0x42, 0x82)
		return
	}
	rtgAsmLoadPrimaryMemSecondaryDisp(a, disp)
}
func rtgAmd64AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 0x04b60f48)
	rtgAsmEmit8(a, 0x8)
}
func rtgAmd64AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmLoadBytePrimaryIndexTertiary(a)
		return
	}
	if size == 2 {
		rtgAsmEmit32(a, 0x04bf0f48)
		rtgAsmEmit8(a, 0x48)
		return
	}
	if size == 4 {
		rtgAsmEmit32(a, 0x88046348)
		return
	}
	rtgAsmLoadQwordPrimaryIndexTertiary8(a)
}
func rtgAmd64AsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAsmEmit32(a, 0xca048948)
}
func rtgAmd64AsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit24(a, 0x028948)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8948, 0x42, 0x82)
}
func rtgAmd64AsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
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
	if size == 4 {
		rtgAsmEmit8(a, 0x89)
		rtgAsmSecondaryDisp(a, disp)
		return
	}
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp)
}
func rtgAmd64AsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		rtgAsmEmit8(a, 0x25)
		rtgAsmEmit32(a, 0xff)
		return
	}
	if kind == rtgTypeInt16 {
		rtgAsmEmit32(a, 0xc0bf0f48)
		return
	}
	if kind == rtgTypeInt32 {
		rtgAsmEmit24(a, 0xc06348)
	}
}
func rtgAmd64AsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit24(a, 0x02ff48)
}
func rtgAmd64AsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit24(a, 0x0aff48)
}
func rtgAmd64AsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc08548)
	rtgAsmEmit24(a, 0xc0940f)
	rtgAsmEmit24(a, 0xc0b60f)
}
func rtgAmd64AsmCmpRaxImm8(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc085)
		return
	}
	rtgAsmEmit4(a, 0x48, 0x83, 0xf8, imm)
}
func rtgAmd64AsmAddRaxRcx(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc80148)
}
func rtgAmd64AsmSubRaxRcx(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc82948)
}
func rtgAmd64AsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xe1, imm)
}
func rtgAmd64AsmShlRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xe0, imm)
}
func rtgAmd64AsmSarRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit4(a, 0x48, 0xc1, 0xf8, imm)
}
func rtgAmd64AsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAsmEmit32(a, 0x48c38948)
	rtgAsmEmit32(a, 0x9948c889)
	rtgAsmEmit24(a, 0xfbf748)
	if mod {
		rtgAsmEmit24(a, 0xd08948)
	}
}
func rtgAmd64AsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAsmEmit32(a, 0x0fc13948)
	rtgAsmEmit3(a, setcc, 0xc0, 0xf)
	rtgAsmEmit16(a, 0xc0b6)
}
func rtgAmd64EmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep *rtgExprParse, idx int, matchLabel int) bool {
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
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}
func rtgAmd64EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
		rtgAsmEmit32(a, 0x48c12948)
		rtgAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		rtgAsmEmit32(a, 0xc1af0f48)
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
			rtgAsmEmit32(a, 0x48d0f748)
			rtgAsmEmit16(a, 0xc821)
		} else {
			rtgAsmEmit24(a, 0xc82148)
		}
		return true
	}
	if c0 == '|' {
		rtgAsmEmit24(a, 0xc80948)
		return true
	}
	if c0 == '^' {
		rtgAsmEmit24(a, 0xc83148)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgAsmEmit32(a, 0x48ca8948)
			rtgAsmEmit32(a, 0x8948c189)
			rtgAsmEmit32(a, 0xe0d348d0)
		} else if c1 == '=' {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9e)
		} else {
			rtgAsmCmpTertiaryPrimarySet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAsmEmit32(a, 0x48ca8948)
			rtgAsmEmit32(a, 0x8948c189)
			rtgAsmEmit32(a, 0xf8d348d0)
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
func rtgAmd64EmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
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
		rtgAsmCmpPrimaryImm8(&g.asm, rightValue)
		rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if c0 == '=' || c0 == '!' {
		leftType := rtgInferParsedExprType(g, ep, leftIndex)
		rightType := rtgInferParsedExprType(g, ep, rightIndex)
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
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushPrimary(&g.asm)
	if !rtgEmitIntExpr(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopTertiary(&g.asm)
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmCmpRegReg(&g.asm, rtgAarch64RegRcx, rtgAarch64RegRax)
	} else if rtgTargetArch == rtgArchArm {
		rtgArmAsmCmpRegReg(&g.asm, rtgArmRegRcx, rtgArmRegRax)
	} else if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitRegReg(&g.asm, rtgWasm32OpCmpRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
	} else {
		rtgAsmEmit24(&g.asm, 0xc13948)
	}
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}
func rtgAmd64EmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		rtgAsmAddSecondaryTertiary(a)
		if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopPrimary(a)
		} else {
			rtgAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		}
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
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		argType := rtgInferParsedExprType(g, ep, argIndex)
		argResolved := rtgResolveType(meta, argType)
		if argResolved.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, argResolved.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		if !rtgEmitSlicePtrLen(g, ep, argIndex) {
			return false
		}
		rtgAsmPushTertiary(a)
		rtgAsmPopSecondary(a)
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
func rtgAmd64EmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushSliceRegs(a)
		rtgAsmLoadSecondaryStack(a, addrOffset)
		rtgAsmPopStoreSliceMemSecondary(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushStringRegs(a)
		rtgAsmLoadSecondaryStack(a, addrOffset)
		rtgAsmPopStoreStringMemSecondary(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgAsmLoadSecondaryStack(a, addrOffset)
		rtgEmitCopyStackToMemSecondary(g, tempOffset, fieldOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmNormalizePrimaryForKind(a, fieldResolved.kind)
	rtgAsmLoadSecondaryStack(a, addrOffset)
	rtgAsmStorePrimaryMemSecondaryDisp(a, fieldOffset)
	return true
}
func rtgAmd64EmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
			if rtgTargetArch == rtgArchAarch64 {
				rtgAarch64AsmStoreRegMem(a, rtgAarch64RegRax, rtgAarch64RegRcx, at, 8)
			} else if rtgTargetArch == rtgArchArm {
				rtgArmAsmStoreRegMem(a, rtgArmRegRax, rtgArmRegRcx, at, 4)
			} else if rtgTargetArch == rtgArchWasm32 {
				rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRcx, at, 4)
			} else if at == 0 {
				rtgAsmEmit24(a, 0x018948)
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
		fn := &meta.funcs[fnIndex]
		receiverIndex := -1
		receiverDotTok := 0
		if fn.receiverType != 0 {
			callee := &ep.exprs[e.left]
			if callee.kind != rtgExprSelector {
				return false
			}
			receiverIndex = callee.left
			receiverDotTok = callee.tok
		}
		wordCount := 1
		for i := e.argCount - 1; i >= 0; i-- {
			argIndex := ep.args[e.firstArg+i]
			paramIndex := i
			if receiverIndex >= 0 {
				paramIndex = i + 1
			}
			words := rtgEmitCallParamArgReverse(g, ep, argIndex, fn.firstParam+paramIndex)
			if words < 0 {
				return false
			}
			wordCount += words
		}
		if receiverIndex >= 0 {
			words := rtgEmitMethodReceiverArgReverse(g, ep, receiverIndex, meta.params[fn.firstParam].typ)
			if words < 0 {
				words = rtgEmitMethodReceiverArgTokensReverse(g, receiverDotTok, meta.params[fn.firstParam].typ)
				if words < 0 {
					return false
				}
			}
			wordCount += words
		}
		rtgAsmLoadPrimaryStack(a, g.returnStruct)
		rtgAsmPushPrimary(a)
		rtgEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}
func rtgAmd64EmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
	if rtgTypeKindIsScalarInt(resolved.kind) {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
			return false
		}
		rtgAsmNormalizePrimaryForKind(&g.asm, resolved.kind)
		return true
	}
	return false
}
func rtgAmd64EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
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
		rtgAsmEmit16(a, 0x5841)
	}
	if wordCount > 5 {
		rtgAsmEmit16(a, 0x5941)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 8
		if rtgAsmImmFits8Signed(imm) {
			rtgAsmEmit4(a, 0x48, 0x83, 0xc4, imm)
		} else {
			rtgAsmEmit24(a, 0xc48148)
			rtgAsmEmit32(a, imm)
		}
	}
}
func rtgAmd64EmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == rtgExprUnary || e.kind == rtgExprBinary || e.kind == rtgExprCall) && rtgExprCanFoldConst(g, ep, idx) {
		constResult := rtgEvalConstExpr(g, ep, idx)
		if constResult.ok {
			rtgAsmPrimaryImm(a, constResult.value)
			return true
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
					return false
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
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee == rtgIdentSyscall {
			return rtgEmitArbitrarySyscall(g, ep, idx)
		}
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentInt64) {
			return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 1 && (callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32) {
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			if callee == rtgIdentByte {
				rtgAsmNormalizePrimaryForKind(a, rtgTypeByte)
			} else if callee == rtgIdentInt16 {
				rtgAsmNormalizePrimaryForKind(a, rtgTypeInt16)
			} else {
				rtgAsmNormalizePrimaryForKind(a, rtgTypeInt32)
			}
			return true
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
			if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
				rtgAsmPushTertiary(a)
				rtgAsmPopPrimary(a)
			} else {
				rtgAsmEmit16(a, 0x5851)
			}
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
			if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
				rtgAsmPushTertiary(a)
				rtgAsmPopPrimary(a)
			} else {
				rtgAsmEmit16(a, 0x5851)
			}
			return true
		}
		if callee == rtgIdentOpen {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsOpen(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if rtgTargetIsDarwin() {
				if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
					return false
				}
				rtgAsmPushPrimary(a)
				if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmCopyPrimaryToCallWord0(a)
				rtgAsmPopCallWord1(a)
				rtgAsmSecondaryImm(a, 493)
				// mode is the first variadic argument to open. Darwin's arm64
				// ABI passes variadic arguments on the stack.
				rtgAarch64AsmPushReg(a, rtgAarch64RegRdx)
				rtgDarwinArm64CallVirtualArgs(a, rtgDarwinImportOpen, 2)
				rtgAarch64AsmAddRegImm(a, 31, 31, 16)
				return true
			}
			if rtgTargetArch == rtgArchAarch64 {
				if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
					return false
				}
				rtgAsmPushPrimary(a)
				if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmCopyPrimaryToCallWord1(a)
				rtgAsmPopSecondary(a)
				rtgAarch64AsmMovRegImm(a, rtgAarch64RegRdi, -100)
				rtgAarch64AsmMovRegImm(a, rtgAarch64RegR10, 493)
				rtgAsmPrimaryImm(a, rtgLinuxSysOpen())
				rtgAsmSyscall(a)
				return true
			}
			if rtgTargetArch == rtgArchWasm32 {
				if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
					return false
				}
				rtgAsmPushPrimary(a)
				if !rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmCopyPrimaryToCallWord0(a)
				rtgAsmPopCallWord1(a)
				rtgAsmPrimaryImm(a, rtgLinuxSysOpen())
				rtgAsmSyscall(a)
				return true
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPopCallWord1(a)
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
			if rtgTargetIsDarwin() {
				rtgDarwinArm64CallVirtualArgs(a, rtgDarwinImportClose, 1)
				return true
			}
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
			if rtgTargetIsDarwin() {
				rtgDarwinArm64CallVirtualArgs(a, rtgDarwinImportFchmod, 2)
				return true
			}
			rtgAsmPrimaryImm(a, rtgLinuxSysFchmod())
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentRead {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsReadWrite(g, ep, idx, false)
			}
			if rtgTargetIsDarwin() {
				return rtgEmitBuiltinReadWrite(g, ep, idx, rtgDarwinImportRead, rtgDarwinImportPread)
			}
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysReadSeq(), rtgLinuxSysReadAt())
		}
		if callee == rtgIdentWrite {
			if rtgTargetIsWindows() {
				return rtgEmitWindowsReadWrite(g, ep, idx, true)
			}
			if rtgTargetIsDarwin() {
				return rtgEmitBuiltinReadWrite(g, ep, idx, rtgDarwinImportWrite, rtgDarwinImportPwrite)
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
			offset := rtgAddTypedLocal(g, 0, 0, baseType)
			if !rtgEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			rtgAsmLoadPrimaryStack(a, offset-fieldOffset)
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
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
					rtgAsmPrimaryBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressSecondary(g, ep, e.left) {
					return false
				}
				if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
					rtgAsmCopySecondaryToPrimary(a)
				} else {
					rtgAsmEmit16(a, 0x5852)
				}
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
			if rtgTargetArch == rtgArchAarch64 {
				rtgAarch64AsmNegRax(a)
			} else if rtgTargetArch == rtgArchArm {
				rtgArmAsmNegRax(a)
			} else if rtgTargetArch == rtgArchWasm32 {
				rtgWasm32AsmNegRax(a)
			} else {
				rtgAsmEmit24(a, 0xd8f748)
			}
			return true
		}
		if rtgTokCharIs(p, e.tok, '+') {
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
		if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			rtgAsmPrimaryImm(a, 1)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, falseLabel)
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
		return true
	}
	return false
}
func rtgAmd64EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if rtgTokCharIs(p, e.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmEmit32(a, 0xc1af0f48)
		rtgAsmSarPrimaryImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmShlPrimaryImm(a, 2)
		rtgAsmPushPrimary(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !rtgEmitIntExpr(g, ep, e.left) {
		return false
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPopTertiary(a)
	return rtgEmitPrimaryTertiaryOp(g, e.tok)
}
func rtgAmd64EmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !rtgEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		if rtgTargetArch == rtgArchAarch64 {
			rtgAsmPushSecondary(a)
			rtgAsmPopCallWord0(a)
			rtgAarch64AsmAddRegImm(a, rtgAarch64RegRsi, rtgAarch64RegRdx, 8)
		} else if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushSecondary(a)
			rtgAsmPopCallWord0(a)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRsi, rtgWasm32RegRdx)
			rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRsi, 8)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR9, rtgWasm32RegRdx)
			rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegR9, 16)
		} else {
			rtgAsmEmit16(a, 0x5f52)
			rtgAsmEmit24(a, 0x728d48)
			rtgAsmEmit8(a, 8)
			rtgAsmEmit4(a, 0x4c, 0x8d, 0x4a, 16)
		}
		return true
	}
	if loc.global {
		if rtgTargetArch == rtgArchAarch64 {
			rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRdi, loc.offset, rtgAbsBssReloc)
			rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRsi, loc.offset+8, rtgAbsBssReloc)
		} else if rtgTargetArch == rtgArchWasm32 {
			rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdi, 0)
			rtgAsmAddAbsReloc(a, len(a.code)-4, loc.offset, rtgAbsBssReloc)
			rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRsi, 0)
			rtgAsmAddAbsReloc(a, len(a.code)-4, loc.offset+8, rtgAbsBssReloc)
			rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegR9, 0)
			rtgAsmAddAbsReloc(a, len(a.code)-4, loc.offset+16, rtgAbsBssReloc)
		} else {
			rtgAsmEmit24(a, 0x3d8d48)
			at := len(a.code)
			rtgAsmEmit32(a, 0)
			rtgAsmAddAbsReloc(a, at, loc.offset, rtgAbsBssReloc)
			rtgAsmEmit24(a, 0x358d48)
			at = len(a.code)
			rtgAsmEmit32(a, 0)
			rtgAsmAddAbsReloc(a, at, loc.offset+8, rtgAbsBssReloc)
			rtgAsmEmit24(a, 0x0d8d4c)
			at = len(a.code)
			rtgAsmEmit32(a, 0)
			rtgAsmAddAbsReloc(a, at, loc.offset+16, rtgAbsBssReloc)
		}
		return true
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRdi, loc.offset)
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegRsi, loc.offset-8)
		rtgAarch64AsmLeaRegStack(a, rtgAarch64RegR9, loc.offset-16)
		return true
	}
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitStack(a, rtgWasm32OpLeaStack, rtgWasm32RegRdi, loc.offset)
		rtgWasm32EmitStack(a, rtgWasm32OpLeaStack, rtgWasm32RegRsi, loc.offset-8)
		rtgWasm32EmitStack(a, rtgWasm32OpLeaStack, rtgWasm32RegR9, loc.offset-16)
		return true
	}
	rtgAsmAddressCallWord0Stack(a, loc.offset)
	rtgAsmAddressCallWord1Stack(a, loc.offset-8)
	r9Offset := loc.offset - 16
	rtgAsmEmit16(a, 0x8d4c)
	if r9Offset >= 0 && r9Offset <= 128 {
		rtgAsmEmit8(a, 0x4d)
		rtgAsmEmit8(a, -r9Offset)
		return true
	}
	rtgAsmEmit8(a, 0x8d)
	rtgAsmEmit32(a, -r9Offset)
	return true
}

func rtgAmd64AsmJccLabel(a *rtgAsm, op int, label int) {
	rtgAsmEmit2(a, 0x0f, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgAmd64EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := rtgEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	noGrowLabel := rtgAsmNewLabel(a)
	haveCapLabel := rtgAsmNewLabel(a)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x018b4d)
	rtgAsmEmit24(a, 0xc1394c)
	rtgAmd64AsmJccLabel(a, 0x8c, noGrowLabel)
	rtgAsmEmit8(a, 0x57)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit16(a, 0x5141)
	rtgAsmPushSecondary(a)
	rtgAsmPushTertiary(a)
	rtgAsmEmit24(a, 0xc0854d)
	rtgAmd64AsmJccLabel(a, 0x85, haveCapLabel)
	rtgAsmEmit24(a, 0xc0c749)
	rtgAsmEmit32(a, 16)
	rtgAsmMarkLabel(a, haveCapLabel)
	rtgAsmEmit24(a, 0xc0014d)
	rtgAsmEmit16(a, 0x5041)
	rtgAsmEmit24(a, 0xc1894c)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmCallLabel(a, arenaAllocLabel)
	rtgAsmPushPrimary(a)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 16)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x54, 0x24, 24)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x7c, 0x24, 48)
	rtgAsmEmit24(a, 0x378b48)
	rtgAsmEmit32(a, 0x243c8b48)
	rtgAsmEmit8(a, 0xfc)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x7c, 0x24, 48)
	rtgAsmEmit32(a, 0x24048b48)
	rtgAsmEmit24(a, 0x078948)
	rtgAsmEmit5(a, 0x4c, 0x8b, 0x4c, 0x24, 32)
	rtgAsmEmit5(a, 0x4c, 0x8b, 0x44, 0x24, 8)
	rtgAsmEmit24(a, 0x01894d)
	rtgAsmEmit32(a, 0x24048b48)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 16)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x54, 0x24, 24)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x74, 0x24, 40)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 16)
	rtgAsmIncTertiary(a)
	rtgAsmEmit24(a, 0x0e8948)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 56)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, noGrowLabel)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x078b48)
	rtgAsmEmit32(a, 0xcaaf0f48)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmIncTertiary(a)
	rtgAsmEmit24(a, 0x0e8948)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgAmd64EnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x078b4c)
	rtgAsmEmit32(a, 0x08148841)
	rtgAsmIncTertiary(a)
	rtgAsmEmit24(a, 0x0e8948)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}
func rtgAmd64EnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x078b4c)
	rtgAsmEmit32(a, 0xc8148949)
	rtgAsmIncTertiary(a)
	rtgAsmEmit24(a, 0x0e8948)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}
func rtgAmd64EnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	arenaAllocLabel := rtgEnsureArenaAllocHelper(g)
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	noGrowLabel := rtgAsmNewLabel(a)
	capNonZeroLabel := rtgAsmNewLabel(a)
	capReadyLabel := rtgAsmNewLabel(a)
	capOKLabel := rtgAsmNewLabel(a)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x018b4d)
	rtgAsmEmit24(a, 0xca8949)
	rtgAsmEmit24(a, 0xd20149)
	rtgAsmEmit24(a, 0xc2394d)
	rtgAmd64AsmJccLabel(a, 0x8e, noGrowLabel)
	rtgAsmEmit8(a, 0x57)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit16(a, 0x5141)
	rtgAsmPushPrimary(a)
	rtgAsmPushSecondary(a)
	rtgAsmPushTertiary(a)
	rtgAsmEmit16(a, 0x5241)
	rtgAsmEmit24(a, 0xc0854d)
	rtgAmd64AsmJccLabel(a, 0x85, capNonZeroLabel)
	rtgAsmEmit24(a, 0xc0c749)
	rtgAsmEmit32(a, 16)
	rtgAsmJmpLabel(a, capReadyLabel)
	rtgAsmMarkLabel(a, capNonZeroLabel)
	rtgAsmEmit24(a, 0xc0014d)
	rtgAsmMarkLabel(a, capReadyLabel)
	rtgAsmEmit32(a, 0x24148b4c)
	rtgAsmEmit24(a, 0xc2394d)
	rtgAmd64AsmJccLabel(a, 0x8e, capOKLabel)
	rtgAsmEmit24(a, 0xd0894d)
	rtgAsmMarkLabel(a, capOKLabel)
	rtgAsmEmit16(a, 0x5041)
	rtgAsmEmit24(a, 0xc1894c)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmCallLabel(a, arenaAllocLabel)
	rtgAsmPushPrimary(a)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x7c, 0x24, 64)
	rtgAsmEmit24(a, 0x378b48)
	rtgAsmEmit32(a, 0x243c8b48)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 24)
	rtgAsmEmit8(a, 0xfc)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x7c, 0x24, 64)
	rtgAsmEmit32(a, 0x24048b48)
	rtgAsmEmit24(a, 0x078948)
	rtgAsmEmit5(a, 0x4c, 0x8b, 0x4c, 0x24, 48)
	rtgAsmEmit5(a, 0x4c, 0x8b, 0x44, 0x24, 8)
	rtgAsmEmit24(a, 0x01894d)
	rtgAsmEmit32(a, 0x243c8b48)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 24)
	rtgAsmEmit24(a, 0xcf0148)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x74, 0x24, 40)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 32)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x74, 0x24, 56)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 16)
	rtgAsmEmit24(a, 0x0e8948)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 72)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, noGrowLabel)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x3f8b48)
	rtgAsmEmit24(a, 0xcf0148)
	rtgAsmEmit24(a, 0x160148)
	rtgAsmEmit24(a, 0xc68948)
	rtgAsmEmit24(a, 0xd18948)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}
func rtgAmd64EnsureCopyWordsHelper(g *rtgLinearGen) int {
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
	rtgAsmEmit24(a, 0xd28548)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 0x068b48)
	rtgAsmEmit24(a, 0x078948)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc6, 8)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc7, 8)
	rtgAsmEmit24(a, 0xcaff48)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}
func rtgAmd64EnsureStringEqualHelper(g *rtgLinearGen) int {
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
	rtgAsmPrimaryImm(a, 0)
	rtgAsmEmit24(a, 0xce3948)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit24(a, 0xf68548)
	rtgAsmJzLabel(a, equalLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 0x078a44)
	rtgAsmEmit24(a, 0x023844)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit24(a, 0xc7ff48)
	rtgAsmEmit24(a, 0xc2ff48)
	rtgAsmEmit24(a, 0xceff48)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmPrimaryImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
func rtgAmd64EmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
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
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitSlicePtrLen(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopTertiary(a)
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	rtgAsmMulTertiaryImm(a, elemSize)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmLoadPrimaryMemSecondaryDisp(a, fieldOffset)
	return true
}
func rtgAmd64EmitStringPtrExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		return true
	}
	return false
}
func rtgAmd64EmitSelectorAddressRdx(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmPushPrimary(a)
		if !rtgEmitSlicePtrLen(g, ep, base.left) {
			return false
		}
		rtgAsmPopTertiary(a)
		elemSize := rtgTypeSize(meta, sliceType.elem)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmAddSecondaryTertiary(a)
		if fieldOffset != 0 {
			rtgAsmAddSecondaryImm(a, fieldOffset)
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
				rtgAsmLoadPrimaryBss(a, globalOffset)
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
			if rtgTargetArch == rtgArchAarch64 {
				rtgAarch64AsmLoadRegMem(a, rtgAarch64RegRdx, rtgAarch64RegRdx, 0, 8)
			} else if rtgTargetArch == rtgArchArm {
				rtgArmAsmLoadRegMem(a, rtgArmRegRdx, rtgArmRegRdx, 0, 4)
			} else if rtgTargetArch == rtgArchWasm32 {
				rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRdx, rtgWasm32RegRdx, 0, 4)
			} else {
				rtgAsmEmit24(a, 0x128b48)
			}
		}
		if fieldOffset != 0 {
			rtgAsmAddSecondaryImm(a, fieldOffset)
		}
		return true
	}
	return false
}
