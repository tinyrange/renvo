package main

func rtgAmd64EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
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
		rtgAsmStackMem(a, g.returnStruct, 0x8948, 0x7d, 0xbd)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		rtgPrintErr("rtg: amd64 bind params failed\n")
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		rtgPrintErr("rtg: amd64 emit range failed\n")
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
		rtgAsmStoreRdxStack(a, offset)
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
	rtgAsmStoreRaxStack(a, offset)
	return true
}
func rtgAmd64AsmMovRaxImm(a *rtgAsm, imm int) {
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
		rtgAsmEmit8(a, 0x68)
		rtgAsmEmit32(a, imm)
		rtgAsmPopRax(a)
		return
	}
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}
func rtgAmd64AsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}
func rtgAmd64AsmMovRdxImm(a *rtgAsm, imm int) {
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
			rtgAsmEmit24(a, 0xc2c748)
			rtgAsmEmit32(a, imm)
			return
		}
	}
	rtgAsmEmit16(a, 0xba48)
	rtgAsmEmit64(a, imm)
}
func rtgAmd64AsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAsmEmit24(a, 0x058d48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, dataOff, 0)
}
func rtgAmd64AsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit24(a, 0x058d48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
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
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit24(a, 0xbf0f48)
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 4 {
		rtgAsmMemDisp(a, disp, 0x6348, 0x42, 0x82)
		return
	}
	rtgAsmLoadRaxMemRdxDisp(a, disp)
}
func rtgAmd64AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 0x04b60f48)
	rtgAsmEmit8(a, 0x8)
}
func rtgAmd64AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmLoadByteRaxIndexRcx(a)
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
	rtgAsmLoadQwordRaxIndexRcx8(a)
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
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0x8966)
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 4 {
		rtgAsmEmit8(a, 0x89)
		rtgAsmRdxDisp(a, disp)
		return
	}
	rtgAsmStoreRaxMemRdxDisp(a, disp)
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
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmLoadRaxStack(a, valueOffset)
	rtgAsmMovRdiRax(a)
	rtgAsmLoadRaxStack(a, lenOffset)
	rtgAsmMovRsiRax(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}
func rtgAmd64EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
		rtgAsmEmit32(a, 0x48c12948)
		rtgAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		rtgAsmEmit32(a, 0xc1af0f48)
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
			rtgAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAsmEmit32(a, 0x48ca8948)
			rtgAsmEmit32(a, 0x8948c189)
			rtgAsmEmit32(a, 0xf8d348d0)
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
func rtgAmd64EmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
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
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmMovRdxImm(a, msgLen)
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
		rtgAsmLoadQwordRaxIndexRcxDisp(a, 0)
		rtgAsmAddRdxRcx(a)
		if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushRax(a)
			rtgAsmLoadRaxMemRdxDisp(a, 8)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
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
		rtgAsmPushRcx(a)
		rtgAsmPopRdx(a)
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
	rtgAsmStoreRaxMemRdxDisp(a, fieldOffset)
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
		rtgAsmNormalizeRaxForKind(&g.asm, resolved.kind)
		return true
	}
	return false
}
func rtgAmd64EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
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
			if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
				rtgAsmPushRcx(a)
				rtgAsmPopRax(a)
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
			if rtgTargetArch == rtgArchAarch64 {
				if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
					return false
				}
				rtgAsmPushRax(a)
				if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmMovRsiRax(a)
				rtgAsmPopRdx(a)
				rtgAarch64AsmMovRegImm(a, rtgAarch64RegRdi, -100)
				rtgAarch64AsmMovRegImm(a, rtgAarch64RegR10, 493)
				rtgAsmMovRaxImm(a, rtgLinuxSysOpen())
				rtgAsmSyscall(a)
				return true
			}
			if rtgTargetArch == rtgArchWasm32 {
				if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
					return false
				}
				rtgAsmPushRax(a)
				if !rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmMovRdiRax(a)
				rtgAsmPopRsi(a)
				rtgAsmMovRaxImm(a, rtgLinuxSysOpen())
				rtgAsmSyscall(a)
				return true
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmMovRsiRax(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmMovRdxImm(a, 493)
			rtgAsmMovRaxImm(a, rtgLinuxSysOpen())
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
			rtgAsmMovRdiRax(a)
			rtgAsmMovRaxImm(a, rtgLinuxSysClose())
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
			rtgAsmLoadRaxStack(a, offset-fieldOffset)
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
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
				if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
					rtgAsmMovRaxRdx(a)
				} else {
					rtgAsmEmit16(a, 0x5852)
				}
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
func rtgAmd64EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmEmit32(a, 0xc1af0f48)
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
func rtgAmd64EmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		if rtgTargetArch == rtgArchAarch64 {
			rtgAsmPushRdx(a)
			rtgAsmPopRdi(a)
			rtgAarch64AsmAddRegImm(a, rtgAarch64RegRsi, rtgAarch64RegRdx, 8)
		} else if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushRdx(a)
			rtgAsmPopRdi(a)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRsi, rtgWasm32RegRdx)
			rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRsi, 8)
		} else {
			rtgAsmEmit16(a, 0x5f52)
			rtgAsmEmit24(a, 0x728d48)
			rtgAsmEmit8(a, 8)
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
		} else {
			rtgAsmEmit24(a, 0x3d8d48)
			at := len(a.code)
			rtgAsmEmit32(a, 0)
			rtgAsmAddAbsReloc(a, at, loc.offset, rtgAbsBssReloc)
			rtgAsmEmit24(a, 0x358d48)
			at = len(a.code)
			rtgAsmEmit32(a, 0)
			rtgAsmAddAbsReloc(a, at, loc.offset+8, rtgAbsBssReloc)
		}
		return true
	}
	rtgEmitEnsureLocalSlice(g, loc.offset, elemSize)
	rtgAsmLeaRdiStack(a, loc.offset)
	rtgAsmLeaRsiStack(a, loc.offset-8)
	return true
}
func rtgAmd64EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmEmit24(a, 0x078b48)
	rtgAsmEmit32(a, 0xcaaf0f48)
	rtgAsmAddRaxRcx(a)
	rtgAsmEmit24(a, 0x0e8b48)
	rtgAsmIncRcx(a)
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
	rtgAsmIncRcx(a)
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
	rtgAsmIncRcx(a)
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
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
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
	rtgAsmMovRaxImm(a, 0)
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
	rtgAsmMovRaxImm(a, 1)
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
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(a)
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmAddRaxRcx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmLoadRaxMemRdxDisp(a, fieldOffset)
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
		rtgAsmLoadQwordRaxIndexRcxDisp(a, 0)
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
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	return false
}
