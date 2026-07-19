package main

const renvoAmd64ParamStoreRecipes = "\x48\x89\x7d\xbd\x48\x89\x75\xb5\x48\x89\x55\x95\x48\x89\x4d\x8d\x4c\x89\x45\x85\x4c\x89\x4d\x8d"

func renvoAmd64EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.stackUsed = 0
	g.stackPeak = 0
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	renvoAsmEmit32(a, 0x000000c8)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoAsmStackMem(a, g.returnStruct, 0x8948, 0x7d, 0xbd)
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
		if metaFn.resultType != 0 {
			renvoAsmPrimaryImm(a, 0)
		}
		renvoAsmLeave(a)
		renvoAsmRet(a)
	}
	frame := renvoAlignValue(g.stackPeak, 16)
	if frame > 65520 {
		frame = 65520
	}
	a.code[framePatch+1] = byte(frame)
	a.code[framePatch+2] = byte(frame >> 8)
	return true
}

func renvoAmd64StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	renvoNonNil(g)
	a := &g.asm
	if reg < 6 {
		at := reg * 4
		renvoAsmStackMem(a, offset, int(renvoAmd64ParamStoreRecipes[at])|int(renvoAmd64ParamStoreRecipes[at+1])<<8, int(renvoAmd64ParamStoreRecipes[at+2]), int(renvoAmd64ParamStoreRecipes[at+3]))
		return
	}
	renvoAsmEmit24(a, 0x858b48)
	renvoAsmEmit32(a, 0x10+(reg-6)*8)
	renvoAsmStorePrimaryStack(a, offset)
}
func renvoAmd64AsmMovRaxImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if imm == 0 {
		renvoAsmEmit16(a, 0xc031)
		return
	}
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit2(a, 0x6a, imm)
		renvoAsmPopPrimary(a)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		renvoAsmEmit8(a, 0x68)
		renvoAsmEmit32(a, imm)
		renvoAsmPopPrimary(a)
		return
	}
	renvoAsmEmit16(a, 0xb848)
	renvoAsmEmit64(a, imm)
}
func renvoAmd64AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x158d4c)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}
func renvoAmd64AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x058b48)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	a.lastPrimaryLoad = len(a.code)*8 + 5
}
func renvoAmd64AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x058948)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}
func renvoAmd64AsmMovRdiRax(a *renvoAsm) {
	renvoNonNil(a)
	if renvoAmd64RewritePrimaryLoad(a, 7, false) {
		return
	}
	renvoAsmEmit16(a, 0x5f50)
}
func renvoAmd64AsmMovRaxRdx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xd08948)
}
func renvoAmd64AsmMovRsiRax(a *renvoAsm) {
	renvoNonNil(a)
	if renvoAmd64RewritePrimaryLoad(a, 6, false) {
		return
	}
	renvoAsmEmit16(a, 0x5e50)
}

func renvoAmd64RewritePrimaryLoad(a *renvoAsm, reg int, pushed bool) bool {
	renvoNonNil(a)
	load := a.lastPrimaryLoad
	if pushed {
		load = -load
	}
	end := load / 8
	if pushed {
		end++
	}
	if load <= 0 || end != len(a.code) {
		return false
	}
	at := load/8 - load%8
	a.code[at] += byte(reg * 8)
	if pushed {
		renvoTruncBytes(&a.code, len(a.code)-1)
	}
	a.lastPrimaryLoad = 0
	return true
}

func renvoAmd64AsmMovR8Rax(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xc08949)
}
func renvoAmd64AsmMovR9Rax(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xc18949)
}
func renvoAmd64AsmAddRdxRcx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xca0148)
}
func renvoAmd64AsmSyscall(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x050f)
}
func renvoAmd64AsmPopRdi(a *renvoAsm) {
	renvoNonNil(a)
	if renvoAmd64RewritePrimaryLoad(a, 7, true) {
		return
	}
	renvoAsmEmit8(a, 0x5f)
}
func renvoAmd64AsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, base)
	if offset >= 0 && offset <= 128 {
		renvoAsmEmit8(a, disp8)
		renvoAsmEmit8(a, -offset)
		return
	}
	renvoAsmEmit8(a, disp32)
	renvoAsmEmit32(a, -offset)
}
func renvoAmd64AsmAddRdxImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit4(a, 0x48, 0x83, 0xc2, imm)
		return
	}
	renvoAsmEmit24(a, 0xc28148)
	renvoAsmEmit32(a, imm)
}
func renvoAmd64AsmMemDisp(a *renvoAsm, disp int, op int, disp8 int, disp32 int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, op)
	if renvoAsmImmFits8Signed(disp) {
		renvoAsmEmit2(a, disp8, disp)
		return
	}
	renvoAsmEmit8(a, disp32)
	renvoAsmEmit32(a, disp)
}
func renvoAmd64AsmLoadQwordRaxIndexRcx8(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit32(a, 0xc8048b48)
}
func renvoAmd64AsmLoadQwordRaxIndexRcxDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x8b48)
	if renvoAsmImmFits8Signed(disp) {
		renvoAsmEmit3(a, 0x44, 0x8, disp)
		return
	}
	renvoAsmEmit16(a, 0x0884)
	renvoAsmEmit32(a, disp)
}

func renvoAmd64AsmLoadRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if disp == 0 {
		renvoAsmEmit24(a, 0x028b48)
		a.lastPrimaryLoad = len(a.code)*8 + 1
		return
	}
	renvoAsmMemDisp(a, disp, 0x8b48, 0x42, 0x82)
	distance := 2
	if !renvoAsmImmFits8Signed(disp) {
		distance = 5
	}
	a.lastPrimaryLoad = len(a.code)*8 + distance
}
func renvoAmd64AsmLoadRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoNonNil(a)
	if size == 1 {
		renvoAsmEmit24(a, 0xb60f48)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		renvoAsmEmit24(a, 0xbf0f48)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	if size == 4 {
		renvoAsmMemDisp(a, disp, 0x6348, 0x42, 0x82)
		return
	}
	renvoAsmLoadPrimaryMemSecondaryDisp(a, disp)
}
func renvoAmd64AsmLoadByteRaxIndexRcx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit32(a, 0x04b60f48)
	renvoAsmEmit8(a, 0x8)
}
func renvoAmd64AsmLoadRaxIndexRcxSize(a *renvoAsm, size int) {
	renvoNonNil(a)
	if size == 1 {
		renvoAsmLoadBytePrimaryIndexTertiary(a)
		return
	}
	if size == 2 {
		renvoAsmEmit32(a, 0x04bf0f48)
		renvoAsmEmit8(a, 0x48)
		return
	}
	if size == 4 {
		renvoAsmEmit32(a, 0x88046348)
		return
	}
	renvoAsmLoadQwordPrimaryIndexTertiary8(a)
}
func renvoAmd64AsmStoreRaxMemRdxRcx8(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit32(a, 0xca048948)
}
func renvoAmd64AsmStoreRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if disp == 0 {
		renvoAsmEmit24(a, 0x028948)
		return
	}
	renvoAsmMemDisp(a, disp, 0x8948, 0x42, 0x82)
}
func renvoAmd64AsmStoreRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	renvoNonNil(a)
	if size == 1 {
		renvoAsmEmit8(a, 0x88)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		renvoAsmEmit16(a, 0x8966)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	if size == 4 {
		renvoAsmEmit8(a, 0x89)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp)
}
func renvoAmd64CodeEnds(a *renvoAsm, suffix string) bool {
	renvoNonNil(a)
	start := len(a.code) - len(suffix)
	if start < 0 {
		return false
	}
	for i := 0; i < len(suffix); i++ {
		if a.code[start+i] != suffix[i] {
			return false
		}
	}
	return true
}
func renvoAmd64AsmNormalizeRaxForKind(a *renvoAsm, kind int) {
	renvoNonNil(a)
	if kind == renvoTypeByte {
		if renvoAmd64CodeEnds(a, "\x0f\xb6\xc0") || renvoAmd64CodeEnds(a, "\x48\x0f\xb6\x02") || renvoAmd64CodeEnds(a, "\x48\x0f\xb6\x04\x08") {
			return
		}
		renvoAsmEmit24(a, 0xc0b60f)
		return
	}
	if kind == renvoTypeInt8 {
		renvoAsmEmit32(a, 0xc0be0f48)
		return
	}
	if kind == renvoTypeInt16 {
		renvoAsmEmit32(a, 0xc0bf0f48)
		return
	}
	if kind == renvoTypeUint16 {
		renvoAsmEmit32(a, 0xc0b70f48)
		return
	}
	if kind == renvoTypeInt32 {
		if renvoAmd64CodeEnds(a, "\x48\x63\xc0") || renvoAmd64CodeEnds(a, "\x48\x63\x04\x88") {
			return
		}
		renvoAsmEmit24(a, 0xc06348)
		return
	}
	if kind == renvoTypeUint32 {
		renvoAsmEmit16(a, 0xc089)
	}
}
func renvoAmd64AsmIncMemRdx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x02ff48)
}
func renvoAmd64AsmDecMemRdx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x0aff48)
}
func renvoAmd64AsmBoolNotRax(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit3(a, 0x83, 0xf0, 1)
}
func renvoAmd64AsmCmpRaxImm8(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if imm == 0 {
		renvoAsmEmit16(a, 0xc085)
		return
	}
	renvoAsmEmit4(a, 0x48, 0x83, 0xf8, imm)
}
func renvoAmd64AsmAddRaxRcx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xc80148)
}
func renvoAmd64AsmSubRaxRcx(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0xc82948)
}
func renvoAmd64AsmShlRcxImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit4(a, 0x48, 0xc1, 0xe1, imm)
}
func renvoAmd64AsmShlRaxImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit4(a, 0x48, 0xc1, 0xe0, imm)
}
func renvoAmd64AsmSarRaxImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit4(a, 0x48, 0xc1, 0xf8, imm)
}
func renvoAmd64AsmDivLeftRcxRightRax(a *renvoAsm, mod bool) {
	renvoNonNil(a)
	if mod {
		renvoAsmEmitText(a, "\x48\x83\xf8\xff\x75\x10\x6a\x01\x5a\x48\xc1\xe2\x3f\x48\x39\xd1\x75\x04\x31\xc0\xeb\x10\x53\x48\x89\xc3\x48\x89\xc8\x48\x99\x48\xf7\xfb\x48\x89\xd0\x5b")
		return
	}
	renvoAsmEmitText(a, "\x48\x83\xf8\xff\x75\x11\x6a\x01\x5a\x48\xc1\xe2\x3f\x48\x39\xd1\x75\x05\x48\x89\xc8\xeb\x0d\x53\x48\x89\xc3\x48\x89\xc8\x48\x99\x48\xf7\xfb\x5b")
}

func renvoAmd64AsmCmpRcxRaxSet(a *renvoAsm, setcc int) {
	renvoNonNil(a)
	renvoAsmEmit32(a, 0x0fc13948)
	renvoAsmEmit3(a, setcc, 0xc0, 0xf)
	renvoAsmEmit16(a, 0xc0b6)
}
func renvoAmd64EmitSwitchStringCaseTest(g *renvoLinearGen, valueOffset int, lenOffset int, ep *renvoExprParse, idx int, matchLabel int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	label := renvoEnsureStringEqualHelper(g)
	if !renvoEmitStringValueRegs(g, ep, idx) {
		return false
	}
	renvoAsmCopySecondaryToTertiary(a)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryStack(a, valueOffset)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOffset)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmCallLabel(a, label)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, matchLabel)
	return true
}
func renvoAmd64EmitRaxRcxOp(g *renvoLinearGen, tok int) bool {
	renvoNonNil(g)
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
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	c1 := byte(0)
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	if c0 == '+' {
		renvoAsmAddPrimaryTertiary(a)
		return true
	}
	if c0 == '-' {
		renvoAsmEmit32(a, 0x48c12948)
		renvoAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		renvoAsmEmit32(a, 0xc1af0f48)
		return true
	}
	if c0 == '/' {
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if c0 == '%' {
		renvoAsmDivLeftTertiaryRightPrimary(a, true)
		return true
	}
	op := 0
	if c0 == '&' {
		if c1 == '^' {
			renvoAsmEmit32(a, 0x48d0f748)
			renvoAsmEmit16(a, 0xc821)
			return true
		}
		op = 0xc82148
	}
	if c0 == '|' {
		op = 0xc80948
	}
	if c0 == '^' {
		op = 0xc83148
	}
	if op != 0 {
		renvoAsmEmit24(a, op)
		return true
	}
	shift := 0
	setcc := 0
	if c0 == '<' {
		if c1 == '<' {
			shift = 0xe0d348d0
		} else if c1 == '=' {
			setcc = 0x9e
		} else {
			setcc = 0x9c
		}
	}
	if c0 == '>' {
		if c1 == '>' {
			shift = 0xf8d348d0
		} else if c1 == '=' {
			setcc = 0x9d
		} else {
			setcc = 0x9f
		}
	}
	if c0 == '=' && c1 == '=' {
		setcc = 0x94
	}
	if c0 == '!' && c1 == '=' {
		setcc = 0x95
	}
	if shift != 0 {
		renvoAsmEmit32(a, 0x48ca8948)
		renvoAsmEmit32(a, 0x8948c189)
		renvoAsmEmit32(a, shift)
		return true
	}
	if setcc != 0 {
		renvoAsmCmpTertiaryPrimarySet(a, setcc)
		return true
	}
	return false
}

func renvoAmd64EmitCompareJump(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, label int, jumpIfTrue bool) bool {
	renvoNonNil(g, ep, e)
	p := g.prog
	if e.tok < 0 || e.tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, e.tok)
	end := renvoTokEnd(p, e.tok)
	if start >= end {
		return false
	}
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	c1 := byte(0)
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	if !((c0 == '=' || c0 == '!') && c1 == '=' || c0 == '<' && c1 != '<' || c0 == '>' && c1 != '>') {
		return false
	}
	// The immediate compare fast path operates on the backend's raw integer
	// representation. Let the ordinary expression emitter apply float
	// conversions before comparing mixed float/untyped-constant operands.
	if renvoBinaryUsesFloat(g, ep, e) {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	right := &ep.exprs[rightIndex]
	rightConst := renvoEvalConstExpr(g, ep, rightIndex)
	if rightConst.ok && renvoAsmImmFits8Signed(rightConst.value) {
		if !renvoEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		renvoAsmCmpPrimaryImm8(&g.asm, rightConst.value)
		renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if renvoTargetArch == renvoArchAmd64 {
		left := &ep.exprs[leftIndex]
		if left.kind == renvoExprIdent && right.kind == renvoExprIdent {
			leftLocal := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)
			rightLocal := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)
			if leftLocal >= 0 && rightLocal >= 0 && renvoTypeIsInt(g.meta, g.locals[leftLocal].typ) && renvoTypeIsInt(g.meta, g.locals[rightLocal].typ) {
				renvoAsmLoadPrimaryStack(&g.asm, g.locals[rightLocal].offset)
				renvoAsmStackMem(&g.asm, g.locals[leftLocal].offset, 0x3948, 0x45, 0x85)
				renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
				return true
			}
		}
	}
	if c0 == '=' || c0 == '!' {
		leftType := renvoInferParsedExprType(g, ep, leftIndex)
		rightType := renvoInferParsedExprType(g, ep, rightIndex)
		leftResolved := renvoResolveType(g.meta, leftType)
		renvoNonNil(leftResolved)
		if leftResolved.kind == renvoTypeArray || leftResolved.kind == renvoTypeStruct {
			return false
		}
		if renvoTypeIsString(g.meta, leftType) || renvoTypeIsString(g.meta, rightType) {
			return false
		}
		if right.kind == renvoExprString {
			return false
		}
		if right.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)
			if localIndex >= 0 {
				if renvoTypeIsString(g.meta, g.locals[localIndex].typ) {
					return false
				}
			}
		}
	}
	if !renvoEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, leftIndex) {
		return false
	}
	renvoAsmPopTertiary(&g.asm)
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmCmpRegReg(&g.asm, renvoAarch64RegRcx, renvoAarch64RegRax)
	} else if renvoTargetArch == renvoArchArm {
		renvoArmAsmCmpRegReg(&g.asm, renvoArmRegRcx, renvoArmRegRax)
	} else if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpCmpRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
	} else {
		renvoAsmEmit24(&g.asm, 0xc13948)
	}
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}
func renvoAmd64EmitStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprString {
		msg := renvoDecodeStringToken(g.prog, e.tok)
		msgOff := renvoAddStringData(g, msg)
		msgLen := len(msg)
		renvoAsmPrimaryDataAddr(a, msgOff)
		renvoAsmSecondaryImm(a, msgLen)
		return true
	}
	if e.kind == renvoExprSlice {
		return renvoEmitStringSliceValueRegs(g, ep, idx)
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !renvoTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			renvoAsmLoadPrimarySecondaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && renvoTypeIsString(meta, globalType) {
			renvoAsmLoadPrimaryBss(a, globalOffset)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset+8)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopPrimary(a)
			return true
		}
		constTok := renvoFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := renvoDecodeStringToken(g.prog, constTok)
			msgOff := renvoAddStringData(g, msg)
			msgLen := len(msg)
			renvoAsmPrimaryDataAddr(a, msgOff)
			renvoAsmSecondaryImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == renvoExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != renvoExprIdent {
			return false
		}
		localIndex := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeString {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.right) {
			return false
		}
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		renvoAsmPopTertiary(a)
		renvoAsmShlTertiaryImm(a, 4)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		renvoAsmAddSecondaryTertiary(a)
		if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchWasm32 {
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopPrimary(a)
		} else {
			renvoAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		}
		return true
	}
	if e.kind == renvoExprSelector {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, valueType) {
			return false
		}
		if offset, ok := renvoLocalStructSelectorOffset(g, ep, idx); ok {
			renvoAsmLoadPrimarySecondaryStack(a, offset, offset-8)
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopPrimary(a)
		return true
	}
	if e.kind == renvoExprCall && e.argCount == 1 && renvoExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		argType := renvoInferParsedExprType(g, ep, argIndex)
		argResolved := renvoResolveType(meta, argType)
		renvoNonNil(argResolved)
		if argResolved.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, argResolved.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeByte {
			return false
		}
		if !renvoEmitSlicePtrLen(g, ep, argIndex) {
			return false
		}
		renvoAsmPushTertiary(a)
		renvoAsmPopSecondary(a)
		return true
	}
	if e.kind == renvoExprCall {
		callType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, callType) {
			return false
		}
		if !renvoEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}
func renvoAmd64EmitStructReturnExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	if g.returnStruct <= 0 {
		return false
	}
	e := &ep.exprs[idx]
	resultType := g.meta.funcs[g.currentFunc].resultType
	size := renvoTypeSize(meta, resultType)
	renvoAsmLoadSecondaryStack(a, g.returnStruct)
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || renvoTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		renvoEmitCopyStackToMemSecondary(g, g.locals[localIndex].offset, 0, size)
		return true
	}
	if e.kind == renvoExprIndex {
		leftType := renvoInferParsedExprType(g, ep, e.left)
		sliceType := renvoResolveType(meta, leftType)
		renvoNonNil(sliceType)
		elemType := renvoResolveType(meta, sliceType.elem)
		renvoNonNil(elemType)
		if sliceType.kind != renvoTypeSlice || elemType.kind != renvoTypeStruct || renvoTypeSize(meta, sliceType.elem) != size {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.right) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoAsmMulTertiaryImm(a, size)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmAddSecondaryTertiary(a)
		renvoAsmLoadTertiaryStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			renvoAsmLoadPrimaryMemSecondaryDisp(a, at)
			if renvoTargetArch == renvoArchAarch64 {
				renvoAarch64AsmStoreRegMem(a, renvoAarch64RegRax, renvoAarch64RegRcx, at, 8)
			} else if renvoTargetArch == renvoArchArm {
				renvoArmAsmStoreRegMem(a, renvoArmRegRax, renvoArmRegRcx, at, 4)
			} else if renvoTargetArch == renvoArchWasm32 {
				renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRax, renvoWasm32RegRcx, at, 4)
			} else if at == 0 {
				renvoAsmEmit24(a, 0x018948)
			} else {
				renvoAsmMemDisp(a, at, 0x8948, 0x41, 0x81)
			}
		}
		return true
	}
	if e.kind == renvoExprComposite {
		renvoAsmPrimaryImm(a, 0)
		for at := 0; at < size; at += 8 {
			renvoAsmStorePrimaryMemSecondaryDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := renvoCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 || !renvoEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == renvoExprCall {
		fnIndex, wordCount := renvoPrepareStructCall(g, ep, idx, resultType)
		if fnIndex < 0 {
			return false
		}
		renvoAsmLoadPrimaryStack(a, g.returnStruct)
		renvoAsmPushPrimary(a)
		renvoEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}
func renvoAmd64EmitNamedConversionCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	calleeExpr := &ep.exprs[e.left]
	if calleeExpr.kind != renvoExprIdent {
		return false
	}
	namedType := renvoFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
	resolved := renvoResolveType(g.meta, namedType)
	renvoNonNil(resolved)
	if resolved.kind == renvoTypeString {
		return renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	}
	if renvoTypeKindIsScalarInt(resolved.kind) {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(&g.asm, resolved.kind)
		return true
	}
	return false
}
func renvoAmd64EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	renvoNonNil(g)
	a := &g.asm
	if wordCount > 0 {
		renvoAsmPopCallWord0(a)
	}
	if wordCount > 1 {
		renvoAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		renvoAsmPopSecondary(a)
	}
	if wordCount > 3 {
		renvoAsmPopTertiary(a)
	}
	if wordCount > 4 {
		renvoAsmEmit16(a, 0x5841)
	}
	if wordCount > 5 {
		renvoAsmEmit16(a, 0x5941)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 8
		if renvoAsmImmFits8Signed(imm) {
			renvoAsmEmit4(a, 0x48, 0x83, 0xc4, imm)
		} else {
			renvoAsmEmit24(a, 0xc48148)
			renvoAsmEmit32(a, imm)
		}
	}
}

func renvoAmd64EmitIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	renvoNonNil(p)
	meta := g.meta
	renvoNonNil(meta)
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == renvoExprUnary || e.kind == renvoExprBinary || e.kind == renvoExprCall) && renvoExprCanFoldConst(g, ep, idx) {
		resultType := renvoInferParsedExprType(g, ep, idx)
		result := renvoResolveType(meta, resultType)
		renvoNonNil(result)
		if result.kind != renvoTypeByte && result.kind != renvoTypeInt8 && result.kind != renvoTypeInt16 && result.kind != renvoTypeInt32 && result.kind != renvoTypeUint16 && result.kind != renvoTypeUint32 {
			constResult := renvoEvalConstExpr(g, ep, idx)
			if constResult.ok {
				renvoAsmPrimaryImm(a, constResult.value)
				return true
			}
		}
	}
	if e.kind == renvoExprInt {
		renvoAsmLoadPrimaryIntToken(a, p, e.tok)
		return true
	}
	if e.kind == renvoExprFloat {
		value := renvoParseFloatTokenScaled(p, e.tok)
		renvoAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			constResult := renvoEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					fnIndex := renvoFindMetaFunction(meta, e.nameStart, e.nameEnd)
					if fnIndex < 0 {
						return false
					}
					renvoAsmPrimaryImm(a, fnIndex+1)
					return true
				}
				renvoAsmLoadPrimaryBss(a, globalOffset)
				return true
			}
			renvoAsmPrimaryImm(a, constResult.value)
			return true
		}
		renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == renvoExprChar {
		value := renvoParseCharToken(p, e.tok)
		renvoAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == renvoExprBool {
		value := renvoBoolTokenValue(p, e.tok)
		renvoAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == renvoExprCall {
		if renvoExprIsIdentText(p, ep, e.left, "renvoNonNil") {
			return renvoEmitRuntimeTrustPointer(g, ep, e)
		}
		if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeByteAt") {
			return renvoEmitRuntimeUnsafeIndex(g, ep, e, 1)
		}
		if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeInt32At") {
			return renvoEmitRuntimeUnsafeIndex(g, ep, e, 4)
		}
		if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeIntAt") {
			return renvoEmitRuntimeUnsafeIndex(g, ep, e, renvoNativeIntSize)
		}
		if renvoExprIsIdentText(p, ep, e.left, "renvoTruncBytes") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncParams") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncTypes") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncFields") {
			return renvoEmitRuntimeTruncateSlice(g, ep, e)
		}
		callee := renvoExprIdentCode(p, ep, e.left)
		if callee != renvoIdentSyscall && (renvoFunctionValueCalleeType(g, ep, e.left) != 0 || renvoFuncInfoFromCall(g, ep, e.left) >= 0) {
			return renvoEmitUserCall(g, ep, idx)
		}
		if e.argCount == 1 && renvoExprIsIdentText(p, ep, e.left, "Sizeof") {
			renvoAsmPrimaryImm(a, renvoTypeSize(meta, renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))))
			return true
		}
		if callee == renvoIdentPanic {
			return renvoEmitBuiltinPanic(g, ep, idx)
		}
		if callee == renvoIdentSyscall {
			return renvoEmitArbitrarySyscall(g, ep, idx)
		}
		if callee == renvoIdentNew {
			return renvoEmitBuiltinNew(g, ep, idx)
		}
		firstArgIndex := -1
		if e.argCount > 0 {
			firstArgIndex = renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		}
		if e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			conversion := renvoResolveType(meta, conversionType)
			renvoNonNil(conversion)
			if renvoTypeKindIsScalarValue(conversion.kind) {
				return renvoEmitScalarExprForKind(g, ep, firstArgIndex, conversion.kind)
			}
		}
		if e.argCount == 1 && (callee == renvoIdentCap || callee == renvoIdentLen) {
			count := renvoArrayBuiltinCount(g, ep, e)
			if count >= 0 {
				renvoAsmPrimaryImm(a, count)
				return true
			}
		}
		if e.argCount == 1 && callee == renvoIdentCap {
			if renvoEmitDirectSelectorWords(g, ep, firstArgIndex, 16, -1, renvoNativeIntSize) {
				return true
			}
			if !renvoEmitSlicePtrCap(g, ep, firstArgIndex) {
				return false
			}
			if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm || renvoTargetArch == renvoArchWasm32 {
				renvoAsmPushTertiary(a)
				renvoAsmPopPrimary(a)
			} else {
				renvoAsmEmit16(a, 0x5851)
			}
			return true
		}
		if e.argCount == 1 && callee == renvoIdentLen {
			arg := &ep.exprs[firstArgIndex]
			if arg.kind == renvoExprString {
				msg := renvoDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				renvoAsmPrimaryImm(a, msgLen)
				return true
			}
			if arg.kind == renvoExprIdent {
				localIndex := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (renvoTypeIsSlice(meta, g.locals[localIndex].typ) || renvoTypeIsString(meta, g.locals[localIndex].typ)) {
					renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := renvoFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := renvoFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (renvoTypeIsString(meta, globalType) || renvoTypeIsSlice(meta, globalType)) {
					renvoAsmLoadPrimaryBss(a, globalOffset+8)
					return true
				}
				constTok := renvoFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := renvoDecodeStringToken(p, constTok)
					msgLen := len(msg)
					renvoAsmPrimaryImm(a, msgLen)
					return true
				}
			}
			if arg.kind == renvoExprSelector {
				argType := renvoInferParsedExprType(g, ep, firstArgIndex)
				if renvoTypeIsSlice(meta, argType) || renvoTypeIsString(meta, argType) {
					if renvoEmitDirectSelectorWords(g, ep, firstArgIndex, 8, -1, renvoNativeIntSize) {
						return true
					}
					if offset, ok := renvoLocalStructSelectorOffset(g, ep, firstArgIndex); ok {
						renvoAsmLoadPrimaryStack(a, offset-8)
						return true
					}
					if !renvoEmitSelectorAddressSecondary(g, ep, firstArgIndex) {
						return false
					}
					renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
					return true
				}
			}
			if arg.kind == renvoExprUnary && renvoTokCharIs(p, arg.tok, '*') {
				if !renvoEmitIntExpr(g, ep, arg.left) {
					return false
				}
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
				return true
			}
			argIndex := firstArgIndex
			if renvoTypeIsString(meta, renvoInferParsedExprType(g, ep, argIndex)) {
				if !renvoEmitStringValueRegs(g, ep, argIndex) {
					return false
				}
				renvoAsmPushSecondary(a)
				renvoAsmPopPrimary(a)
				return true
			}
			if !renvoEmitSlicePtrLen(g, ep, firstArgIndex) {
				return false
			}
			if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm || renvoTargetArch == renvoArchWasm32 {
				renvoAsmPushTertiary(a)
				renvoAsmPopPrimary(a)
			} else {
				renvoAsmEmit16(a, 0x5851)
			}
			return true
		}
		if callee == renvoIdentOpen {
			if targetIsWindows() {
				return renvoEmitWindowsOpen(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if targetIsDarwin() {
				if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
					return false
				}
				renvoAsmPushPrimary(a)
				if !renvoEmitStringPtrExpr(g, ep, firstArgIndex) {
					return false
				}
				renvoAsmCopyPrimaryToCallWord0(a)
				renvoAsmPopCallWord1(a)
				renvoAsmSecondaryImm(a, 493)
				// mode is the first variadic argument to open. Darwin's arm64
				// ABI passes variadic arguments on the stack.
				renvoAarch64AsmPushReg(a, renvoAarch64RegRdx)
				renvoDarwinArm64CallVirtualArgs(a, renvoDarwinImportOpen, 2)
				renvoAarch64AsmAddRegImm(a, 31, 31, 16)
				return true
			}
			if renvoTargetArch == renvoArchAarch64 {
				if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
					return false
				}
				renvoAsmPushPrimary(a)
				if !renvoEmitStringPtrExpr(g, ep, firstArgIndex) {
					return false
				}
				renvoAsmCopyPrimaryToCallWord1(a)
				renvoAsmPopSecondary(a)
				renvoAarch64AsmMovRegImm(a, renvoAarch64RegRdi, -100)
				renvoAarch64AsmMovRegImm(a, renvoAarch64RegR10, 493)
				renvoAsmPrimaryImm(a, renvoLinuxSysOpen())
				renvoAsmSyscall(a)
				return true
			}
			if renvoTargetArch == renvoArchWasm32 {
				if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
					return false
				}
				renvoAsmPushPrimary(a)
				if !renvoEmitStringValueRegs(g, ep, firstArgIndex) {
					return false
				}
				renvoAsmCopyPrimaryToCallWord0(a)
				renvoAsmPopCallWord1(a)
				renvoAsmPrimaryImm(a, renvoLinuxSysOpen())
				renvoAsmSyscall(a)
				return true
			}
			if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
				return false
			}
			renvoAsmPushPrimary(a)
			if !renvoEmitStringPtrExpr(g, ep, firstArgIndex) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPopCallWord1(a)
			renvoAsmSecondaryImm(a, 493)
			renvoAsmPrimaryImm(a, renvoLinuxSysOpen())
			renvoAsmSyscall(a)
			return true
		}
		if callee == renvoIdentClose {
			if targetIsWindows() {
				return renvoEmitWindowsClose(g, ep, idx)
			}
			if e.argCount != 1 {
				return false
			}
			if !renvoEmitIntExpr(g, ep, firstArgIndex) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord0(a)
			if targetIsDarwin() {
				renvoDarwinArm64CallVirtualArgs(a, renvoDarwinImportClose, 1)
				return true
			}
			renvoAsmPrimaryImm(a, renvoLinuxSysClose())
			renvoAsmSyscall(a)
			return true
		}
		if callee == renvoIdentChmod {
			if targetIsWindows() {
				return renvoEmitWindowsChmod(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if !renvoEmitIntExpr(g, ep, firstArgIndex) {
				return false
			}
			renvoAsmPushPrimary(a)
			if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord1(a)
			renvoAsmPopCallWord0(a)
			if targetIsDarwin() {
				renvoDarwinArm64CallVirtualArgs(a, renvoDarwinImportFchmod, 2)
				return true
			}
			renvoAsmPrimaryImm(a, renvoLinuxSysFchmod())
			renvoAsmSyscall(a)
			return true
		}
		if callee == renvoIdentRead {
			if targetIsWindows() {
				return renvoEmitWindowsReadWrite(g, ep, idx, false)
			}
			if targetIsDarwin() {
				return renvoEmitBuiltinReadWrite(g, ep, idx, renvoDarwinImportRead, renvoDarwinImportPread)
			}
			return renvoEmitBuiltinReadWrite(g, ep, idx, renvoLinuxSysReadSeq(), renvoLinuxSysReadAt())
		}
		if callee == renvoIdentWrite {
			if targetIsWindows() {
				return renvoEmitWindowsReadWrite(g, ep, idx, true)
			}
			if targetIsDarwin() {
				return renvoEmitBuiltinReadWrite(g, ep, idx, renvoDarwinImportWrite, renvoDarwinImportPwrite)
			}
			return renvoEmitBuiltinReadWrite(g, ep, idx, renvoLinuxSysWriteSeq(), renvoLinuxSysWriteAt())
		}
		if callee == renvoIdentCopy {
			return renvoEmitBuiltinCopy(g, ep, idx)
		}
		return renvoEmitUserCall(g, ep, idx)
	}
	if e.kind == renvoExprIndex {
		return renvoEmitIndexExpr(g, ep, idx)
	}
	if e.kind == renvoExprSelector {
		baseType := renvoInferParsedExprType(g, ep, e.left)
		nativeABI := renvoTypeUsesNativeABI(meta, baseType)
		fieldType := renvoResolveType(meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(fieldType)
		fieldSize := renvoNativeScalarStorageSize(fieldType.kind)
		base := &ep.exprs[e.left]
		if renvoEmitDirectSelectorWords(g, ep, idx, 0, -1, fieldSize) {
			if nativeABI {
				renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
			}
			return true
		}
		if base.kind == renvoExprCall {
			baseResolved := renvoResolveType(meta, baseType)
			renvoNonNil(baseResolved)
			if baseResolved.kind == renvoTypePointer {
				if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
					return false
				}
				renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
				if nativeABI {
					renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
				}
				return true
			}
			if !renvoTypeIsStruct(meta, baseType) {
				return false
			}
			fieldOffset := renvoStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			offset := renvoAddTypedLocal(g, 0, 0, baseType)
			if !renvoEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			if nativeABI {
				renvoAsmAddressPrimaryStack(a, offset-fieldOffset)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
				renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
			} else {
				renvoAsmLoadPrimaryStack(a, offset-fieldOffset)
			}
			return true
		}
		if base.kind == renvoExprIndex {
			return renvoEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if offset, ok := renvoLocalStructSelectorOffset(g, ep, idx); ok {
			if nativeABI {
				renvoAsmAddressPrimaryStack(a, offset)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
				renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
			} else {
				renvoAsmLoadPrimaryStack(a, offset)
			}
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		if nativeABI {
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
			renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
		} else {
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		}
		return true
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(p, e.tok, '&') {
			inner := &ep.exprs[e.left]
			if inner.kind == renvoExprIdent {
				localIndex := renvoFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex >= 0 {
					renvoInvalidateCheckedPointerLocal(g, localIndex)
					renvoAsmStackMem(a, g.locals[localIndex].offset, 0x8d48, 0x45, 0x85)
					return true
				}
				globalOffset := renvoFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					renvoAsmPrimaryBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == renvoExprSelector {
				if !renvoEmitSelectorAddressSecondary(g, ep, e.left) {
					return false
				}
				if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm || renvoTargetArch == renvoArchWasm32 {
					renvoAsmCopySecondaryToPrimary(a)
				} else {
					renvoAsmEmit16(a, 0x5852)
				}
				return true
			}
			if inner.kind == renvoExprIndex {
				return renvoEmitIndexAddressPrimary(g, ep, e.left)
			}
			return false
		}
		if renvoTokCharIs(p, e.tok, '*') {
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmCopyPrimaryToSecondary(a)
			targetKind := renvoPointerTargetKind(g, ep, e.left)
			size := renvoScalarKindSize(targetKind)
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
			return true
		}
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		if renvoTokCharIs(p, e.tok, '-') {
			if renvoTargetArch == renvoArchAarch64 {
				renvoAarch64AsmNegRax(a)
			} else if renvoTargetArch == renvoArchArm {
				renvoArmAsmNegRax(a)
			} else if renvoTargetArch == renvoArchWasm32 {
				renvoWasm32AsmNegRax(a)
			} else {
				renvoAsmEmit24(a, 0xd8f748)
			}
			renvoAmd64NormalizeExprPrimary(g, ep, idx)
			return true
		}
		if renvoTokCharIs(p, e.tok, '+') {
			renvoAmd64NormalizeExprPrimary(g, ep, idx)
			return true
		}
		if renvoTokCharIs(p, e.tok, '!') {
			renvoAsmBoolNotPrimary(a)
			return true
		}
		return false
	}
	if e.kind == renvoExprBinary {
		if renvoBinaryUsesFloat(g, ep, e) {
			return renvoEmitFloatBinaryExpr(g, ep, idx)
		}
		if renvoTok2Is(p, e.tok, '=', '=') || renvoTok2Is(p, e.tok, '!', '=') {
			leftType := renvoInferParsedExprType(g, ep, e.left)
			leftResolved := renvoResolveType(meta, leftType)
			renvoNonNil(leftResolved)
			if leftResolved.kind == renvoTypeArray || leftResolved.kind == renvoTypeStruct {
				return renvoEmitCompositeCompare(g, ep, e, leftType)
			}
		}
		if renvoTok2Is(p, e.tok, '&', '&') || renvoTok2Is(p, e.tok, '|', '|') {
			falseLabel := renvoAsmNewLabel(a)
			endLabel := renvoAsmNewLabel(a)
			if !renvoEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			renvoAsmPrimaryImm(a, 1)
			renvoAsmJmpMarkLabel(a, endLabel, falseLabel)
			renvoAsmPrimaryImm(a, 0)
			renvoAsmMarkLabel(a, endLabel)
			return true
		}
		if renvoTok2Is(p, e.tok, '=', '=') || renvoTok2Is(p, e.tok, '!', '=') {
			leftType := renvoInferParsedExprType(g, ep, e.left)
			rightType := renvoInferParsedExprType(g, ep, e.right)
			if renvoTypeIsString(meta, leftType) || renvoTypeIsString(meta, rightType) {
				notEqual := renvoTok2Is(p, e.tok, '!', '=')
				return renvoEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := &ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		immOpcode := 0
		immGroup := 0
		immMultiply := false
		immShift := 0
		immDivide := false
		if renvoTokCharIs(p, e.tok, '+') {
			immOpcode, immGroup = 0x05, 0xc0
		} else if renvoTokCharIs(p, e.tok, '-') {
			immOpcode, immGroup = 0x2d, 0xe8
		} else if renvoTokCharIs(p, e.tok, '*') {
			immMultiply = true
		} else if renvoTokCharIs(p, e.tok, '&') {
			immOpcode, immGroup = 0x25, 0xe0
		} else if renvoTokCharIs(p, e.tok, '|') {
			immOpcode, immGroup = 0x0d, 0xc8
		} else if renvoTokCharIs(p, e.tok, '^') {
			immOpcode, immGroup = 0x35, 0xf0
		} else if renvoTok2Is(p, e.tok, '<', '<') {
			immShift = 0xe0
		} else if renvoTok2Is(p, e.tok, '>', '>') {
			immShift = 0xf8
		} else if renvoTokCharIs(p, e.tok, '/') {
			immDivide = true
		}
		value := 0
		immediate := false
		if rightKind == renvoExprInt {
			value = renvoParseIntToken(p, rightTok)
			immediate = true
		} else if rightKind == renvoExprChar {
			value = renvoParseCharToken(p, rightTok)
			immediate = true
		} else if rightKind == renvoExprIdent {
			value = renvoFindSmallConstByName(g, rightExpr.nameStart, rightExpr.nameEnd)
			immediate = value >= -128
		}
		if renvoTargetArch == renvoArchAmd64 && immediate && (immOpcode != 0 || immMultiply || immShift != 0) {
			if value >= -2147483647 && value <= 2147483647 && (immShift == 0 || value >= 0 && value < 64) {
				if !renvoEmitIntExpr(g, ep, e.left) {
					return false
				}
				if immShift != 0 {
					renvoAsmEmit4(a, 0x48, 0xc1, immShift, value)
				} else if immMultiply && renvoAsmImmFits8Signed(value) {
					renvoAsmEmit4(a, 0x48, 0x6b, 0xc0, value)
				} else if immMultiply {
					renvoAsmEmit3(a, 0x48, 0x69, 0xc0)
					renvoAsmEmit32(a, value)
				} else if renvoAsmImmFits8Signed(value) {
					renvoAsmEmit4(a, 0x48, 0x83, immGroup, value)
				} else {
					renvoAsmEmit2(a, 0x48, immOpcode)
					renvoAsmEmit32(a, value)
				}
				renvoAmd64NormalizeExprPrimary(g, ep, idx)
				return true
			}
		}
		if renvoTargetArch == renvoArchAmd64 && immediate && value == 2 && immDivide {
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoAsmEmitText(a, "\x48\x85\xc0\x79\x04\x48\x83\xc0\x01\x48\xd1\xf8")
			renvoAmd64NormalizeExprPrimary(g, ep, idx)
			return true
		}
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoAsmPushPrimary(a)
		if rightKind == renvoExprInt {
			renvoAsmLoadPrimaryIntToken(a, p, rightTok)
		} else if rightKind == renvoExprChar {
			value := renvoParseCharToken(p, rightTok)
			renvoAsmPrimaryImm(a, value)
		} else if rightKind == renvoExprBool {
			value := renvoBoolTokenValue(p, rightTok)
			renvoAsmPrimaryImm(a, value)
		} else {
			if !renvoEmitIntExpr(g, ep, e.right) {
				return false
			}
		}
		renvoAsmPopTertiary(a)
		if !renvoEmitPrimaryTertiaryOp(g, e.tok) {
			return false
		}
		renvoAmd64NormalizeExprPrimary(g, ep, idx)
		return true
	}
	return false
}

func renvoAmd64NormalizeExprPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) {
	renvoNonNil(g, ep)
	resultType := renvoInferParsedExprType(g, ep, idx)
	renvoAsmNormalizePrimaryForKind(&g.asm, renvoResolveType(g.meta, resultType).kind)
}

func renvoAmd64EmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	multiply := renvoTokCharIs(p, e.tok, '*')
	divide := renvoTokCharIs(p, e.tok, '/')
	if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
		return false
	}
	if divide {
		renvoAsmShlPrimaryImm(a, 2)
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
		return false
	}
	renvoAsmPopTertiary(a)
	if multiply {
		renvoAsmEmit32(a, 0xc1af0f48)
		renvoAsmSarPrimaryImm(a, 2)
		return true
	}
	if divide {
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	return renvoEmitPrimaryTertiaryOp(g, e.tok)
}
func renvoAmd64EmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	renvoNonNil(g, locEp, loc)
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		if renvoTargetArch == renvoArchAarch64 {
			renvoAsmPushSecondary(a)
			renvoAsmPopCallWord0(a)
			renvoAarch64AsmAddRegImm(a, renvoAarch64RegRsi, renvoAarch64RegRdx, 8)
		} else if renvoTargetArch == renvoArchWasm32 {
			renvoAsmPushSecondary(a)
			renvoAsmPopCallWord0(a)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRsi, renvoWasm32RegRdx)
			renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRsi, 8)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegR9, renvoWasm32RegRdx)
			renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegR9, 16)
		} else {
			renvoAsmEmit16(a, 0x5f52)
			renvoAsmEmit24(a, 0x728d48)
			renvoAsmEmit8(a, 8)
			renvoAsmEmit4(a, 0x4c, 0x8d, 0x4a, 16)
		}
		return true
	}
	if loc.global {
		if renvoTargetArch == renvoArchAarch64 {
			renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRdi, loc.offset, renvoAbsBssReloc)
			renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRsi, loc.offset+8, renvoAbsBssReloc)
		} else if renvoTargetArch == renvoArchWasm32 {
			renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdi, 0)
			renvoAsmAddAbsReloc(a, len(a.code)-4, loc.offset, renvoAbsBssReloc)
			renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRsi, 0)
			renvoAsmAddAbsReloc(a, len(a.code)-4, loc.offset+8, renvoAbsBssReloc)
			renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegR9, 0)
			renvoAsmAddAbsReloc(a, len(a.code)-4, loc.offset+16, renvoAbsBssReloc)
		} else {
			renvoAsmEmit24(a, 0x3d8d48)
			at := len(a.code)
			renvoAsmEmit32(a, 0)
			renvoAsmAddAbsReloc(a, at, loc.offset, renvoAbsBssReloc)
			renvoAsmEmit24(a, 0x358d48)
			at = len(a.code)
			renvoAsmEmit32(a, 0)
			renvoAsmAddAbsReloc(a, at, loc.offset+8, renvoAbsBssReloc)
			renvoAsmEmit24(a, 0x0d8d4c)
			at = len(a.code)
			renvoAsmEmit32(a, 0)
			renvoAsmAddAbsReloc(a, at, loc.offset+16, renvoAbsBssReloc)
		}
		return true
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRdi, loc.offset)
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegRsi, loc.offset-8)
		renvoAarch64AsmLeaRegStack(a, renvoAarch64RegR9, loc.offset-16)
		return true
	}
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitStack(a, renvoWasm32OpLeaStack, renvoWasm32RegRdi, loc.offset)
		renvoWasm32EmitStack(a, renvoWasm32OpLeaStack, renvoWasm32RegRsi, loc.offset-8)
		renvoWasm32EmitStack(a, renvoWasm32OpLeaStack, renvoWasm32RegR9, loc.offset-16)
		return true
	}
	renvoAsmAddressCallWord0Stack(a, loc.offset)
	renvoAsmAddressCallWord1Stack(a, loc.offset-8)
	r9Offset := loc.offset - 16
	renvoAsmEmit16(a, 0x8d4c)
	if r9Offset >= 0 && r9Offset <= 128 {
		renvoAsmEmit8(a, 0x4d)
		renvoAsmEmit8(a, -r9Offset)
		return true
	}
	renvoAsmEmit8(a, 0x8d)
	renvoAsmEmit32(a, -r9Offset)
	return true
}

func renvoAmd64AsmJccLabel(a *renvoAsm, op int, label int) {
	renvoNonNil(a)
	renvoAsmEmit2(a, 0x0f, op)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}

func renvoAmd64InitRuntimeCheckRegs(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	renvoAmd64InitRuntimeCheckReg(a, 0x258d4c, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeBoundsLabel, 2, "\x48\x89\xc2\x48\x39\xc8\x73\x01\xc3\xe9\x00\x00\x00\x00"))
	renvoAmd64InitRuntimeCheckReg(a, 0x2d8d4c, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeSecondaryLabel, 1, "\x48\x85\xd2\x74\x01\xc3\xe9\x00\x00\x00\x00"))
	renvoAmd64InitRuntimeCheckReg(a, 0x358d4c, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeByteIndexLabel, 3, "\x48\x39\xd1\x73\x04\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00"))
	renvoAmd64InitRuntimeCheckReg(a, 0x3d8d4c, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeWordIndexLabel, 4, "\x48\x39\xd1\x73\x08\x48\xc1\xe1\x03\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00"))
	renvoAmd64InitRuntimeCheckReg(a, 0x1d8d48, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeWideIndexLabel, 5, "\x48\x39\xd1\x73\x08\x48\x6b\xc9\x48\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00"))
}

func renvoAmd64InitRuntimeCheckReg(a *renvoAsm, op int, label int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, op)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}

func renvoAmd64CallIndexAddressHelper(a *renvoAsm, elemSize int) {
	renvoNonNil(a)
	if elemSize == 1 {
		renvoAsmEmit24(a, 0xd6ff41)
	} else if elemSize == 8 {
		renvoAsmEmit24(a, 0xd7ff41)
	} else {
		renvoAsmEmit16(a, 0xd3ff)
	}
}

func renvoAmd64EnsureRuntimeCheck(g *renvoLinearGen, slot *int, kind int, code string) int {
	renvoNonNil(g, slot)
	a := &g.asm
	if *slot > 0 {
		return *slot - 1
	}
	label := renvoAsmNewLabel(a)
	*slot = label + 1
	after := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, after, label)
	if kind == 2 && g.meta.panicEnabled {
		renvoAsmEmitText(a, "\x48\x89\xc2\x48\x39\xc8\x0f\x92\xc0\x48\x0f\xb6\xc0\xc3")
	} else if kind == 6 && g.meta.panicEnabled {
		renvoAsmEmitText(a, "\x48\x39\xc2\x72\x0e\x48\x39\xd1\x72\x09\x48\x39\xcf\x72\x04\x6a\x01\x58\xc3\x31\xc0\xc3")
	} else {
		renvoAsmEmitText(a, code)
		renvoAsmAddReloc(a, len(a.code)-4, renvoEnsureUncaughtRuntimeFaultHelper(g))
	}
	renvoAsmMarkLabel(a, after)
	return label
}

func renvoAmd64EnsureAppendAddrHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
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
	haveCapLabel := renvoAsmNewLabel(a)
	renvoAsmEmitText(a, "\x48\x8b\x0e\x4d\x8b\x01\x4c\x39\xc1")
	renvoAmd64AsmJccLabel(a, 0x8c, noGrowLabel)
	renvoAsmEmitText(a, "\x57\x56\x41\x51\x52\x51\x4d\x85\xc0")
	renvoAmd64AsmJccLabel(a, 0x85, haveCapLabel)
	renvoAsmEmitText(a, "\x49\xc7\xc0\x10\x00\x00\x00")
	renvoAsmMarkLabel(a, haveCapLabel)
	renvoAsmEmitText(a, "\x4d\x01\xc0\x41\x50\x4c\x89\xc1\x0f\xaf\xca\x51\x58")
	renvoAsmCallLabel(a, arenaAllocLabel)
	renvoAsmEmitText(a, "\x50\x48\x8b\x4c\x24\x10\x48\x8b\x54\x24\x18\x0f\xaf\xca\x48\x8b\x7c\x24\x30\x48\x8b\x37\x48\x8b\x3c\x24\xfc\xf3\xa4\x48\x8b\x7c\x24\x30\x48\x8b\x04\x24\x48\x89\x07\x4c\x8b\x4c\x24\x20\x4c\x8b\x44\x24\x08\x4d\x89\x01\x48\x8b\x04\x24\x48\x8b\x4c\x24\x10\x48\x8b\x54\x24\x18\x0f\xaf\xca\x48\x01\xc8\x48\x8b\x74\x24\x28\x48\x8b\x4c\x24\x10\x48\xff\xc1\x48\x89\x0e\x48\x83\xc4\x38\xc3")
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoAsmEmitText(a, "\x48\x8b\x0e\x48\x8b\x07\x48\x0f\xaf\xca\x48\x01\xc8\x48\x8b\x0e\x48\xff\xc1\x48\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvoAmd64EnsureAppend8Helper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoAsmEmitText(a, "\x48\x8b\x0e\x4c\x8b\x07\x41\x88\x14\x08\x48\xff\xc1\x48\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}
func renvoAmd64EnsureAppend64Helper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoAsmEmitText(a, "\x48\x8b\x0e\x4c\x8b\x07\x49\x89\x14\xc8\x48\xff\xc1\x48\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

// Kept as empty compatibility hooks for the performance source slicer. The
// legacy helpers have no callers in the compiler.
func renvoAmd64EnsureAppendBytesHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	return 0
}
func renvoAmd64EnsureCopyWordsHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	return 0
}
func renvoAmd64EnsureStringEqualHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmEmitText(a, "\x31\xc0\x48\x39\xce")
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmitText(a, "\x48\x85\xf6")
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmitText(a, "\x44\x8a\x07\x44\x38\x02")
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmitText(a, "\x48\xff\xc7\x48\xff\xc2\x48\xff\xce")
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmEmitText(a, "\x6a\x01\x58")
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmEmitText(a, "\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
func renvoAmd64EmitIndexedStructField(g *renvoLinearGen, ep *renvoExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := renvoInferParsedExprType(g, ep, indexExpr.left)
	sliceType := renvoResolveType(g.meta, leftType)
	renvoNonNil(sliceType)
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemType := renvoResolveType(g.meta, sliceType.elem)
	renvoNonNil(elemType)
	if elemType.kind != renvoTypeStruct && elemType.kind != renvoTypePointer {
		return false
	}
	fieldOffset := renvoStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	if !renvoEmitIndexedSelectorAddressSecondary(g, ep, indexIdx, fieldOffset) {
		return false
	}
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	return true
}
func renvoAmd64EmitStringPtrExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprString {
		msg := renvoDecodeStringToken(p, e.tok)
		msgOff := renvoAddStringData(g, msg)
		renvoAsmPrimaryDataAddr(a, msgOff)
		return true
	}
	if e.kind == renvoExprCall && e.argCount == 1 && renvoExprIsIdentText(p, ep, e.left, "string") {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		arg := &ep.exprs[argIndex]
		if arg.kind != renvoExprIdent {
			return false
		}
		localIndex := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeByte {
			return false
		}
		renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == renvoExprCall {
		callType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, callType) {
			return false
		}
		return renvoEmitStringValueRegs(g, ep, idx)
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !renvoTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && renvoTypeIsString(meta, globalType) {
			renvoAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		constTok := renvoFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := renvoDecodeStringToken(p, constTok)
			msgOff := renvoAddStringData(g, msg)
			renvoAsmPrimaryDataAddr(a, msgOff)
			return true
		}
		return false
	}
	if e.kind == renvoExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != renvoExprIdent {
			return false
		}
		localIndex := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeString {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.right) {
			return false
		}
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		renvoAsmPopTertiary(a)
		renvoAsmShlTertiaryImm(a, 4)
		renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		return true
	}
	return false
}
func renvoAmd64EmitSelectorAddressRdx(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	renvoNonNil(meta)
	a := &g.asm
	e := &ep.exprs[idx]
	base := &ep.exprs[e.left]
	baseType := renvoInferParsedExprType(g, ep, e.left)
	fieldOffset := renvoStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if renvoStructPromotedPointerField(g, baseType, e.nameStart, e.nameEnd) >= 0 {
		return renvoEmitPromotedPointerSelectorAddress(g, ep, idx, baseType)
	}
	if base.kind == renvoExprCall {
		baseResolved := renvoResolveType(meta, baseType)
		renvoNonNil(baseResolved)
		if baseResolved.kind != renvoTypePointer && baseResolved.kind != renvoTypeStruct {
			return false
		}
		if baseResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAmd64AddCheckedSecondaryFieldOffset(g, fieldOffset)
			return true
		}
	}
	if base.kind == renvoExprComposite || base.kind == renvoExprCall {
		offset := renvoAddUnnamedLocal(g, baseType)
		if !renvoEmitTypedAssign(g, ep, e.left, offset) {
			return false
		}
		renvoAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if renvoTargetArch == renvoArchAmd64 && base.kind == renvoExprSelector {
		totalOffset := fieldOffset
		rootIndex := e.left
		for ep.exprs[rootIndex].kind == renvoExprSelector {
			part := &ep.exprs[rootIndex]
			partType := renvoResolveType(meta, renvoInferParsedExprType(g, ep, rootIndex))
			renvoNonNil(partType)
			if partType.kind == renvoTypePointer {
				break
			}
			partBaseType := renvoInferParsedExprType(g, ep, part.left)
			partOffset := renvoStructFieldOffset(g, partBaseType, part.nameStart, part.nameEnd)
			if partOffset < 0 {
				break
			}
			totalOffset += partOffset
			rootIndex = part.left
		}
		root := &ep.exprs[rootIndex]
		if root.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
			if localIndex >= 0 {
				rootType := renvoResolveType(meta, g.locals[localIndex].typ)
				renvoNonNil(rootType)
				if rootType.kind == renvoTypePointer {
					renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
					renvoAmd64CheckSecondaryFieldBase(g, localIndex)
					renvoAmd64AddSecondaryFieldOffset(a, totalOffset)
					return true
				}
			}
		}
	}
	if base.kind == renvoExprIndex {
		return renvoEmitIndexedSelectorAddressSecondary(g, ep, e.left, fieldOffset)
	}
	if base.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := renvoFindGlobalType(g, base.nameStart, base.nameEnd)
			t := renvoResolveType(meta, globalType)
			renvoNonNil(t)
			if globalOffset < 0 {
				return false
			}
			if t.kind == renvoTypePointer {
				renvoAsmLoadPrimaryBss(a, globalOffset)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAmd64AddCheckedSecondaryFieldOffset(g, fieldOffset)
				return true
			}
			if t.kind != renvoTypeStruct {
				return false
			}
			renvoAsmPrimaryBssAddr(a, globalOffset)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAmd64AddSecondaryFieldOffset(a, fieldOffset)
			return true
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			renvoAmd64CheckSecondaryFieldBase(g, localIndex)
			renvoAmd64AddSecondaryFieldOffset(a, fieldOffset)
			return true
		}
		renvoAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if base.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, e.left) {
			return false
		}
		t := renvoResolveType(meta, baseType)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			if renvoTargetArch == renvoArchAarch64 {
				renvoAarch64AsmLoadRegMem(a, renvoAarch64RegRdx, renvoAarch64RegRdx, 0, 8)
			} else if renvoTargetArch == renvoArchArm {
				renvoArmAsmLoadRegMem(a, renvoArmRegRdx, renvoArmRegRdx, 0, 4)
			} else if renvoTargetArch == renvoArchWasm32 {
				renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRdx, renvoWasm32RegRdx, 0, 4)
			} else {
				renvoAsmEmit24(a, 0x128b48)
			}
			renvoAmd64AddCheckedSecondaryFieldOffset(g, fieldOffset)
			return true
		}
		renvoAmd64AddSecondaryFieldOffset(a, fieldOffset)
		return true
	}
	return false
}

func renvoAmd64AddSecondaryFieldOffset(a *renvoAsm, fieldOffset int) {
	renvoNonNil(a)
	if fieldOffset != 0 {
		renvoAsmAddSecondaryImm(a, fieldOffset)
	}
}

func renvoAmd64CheckSecondaryFieldBase(g *renvoLinearGen, localIndex int) {
	renvoNonNil(g)
	if !renvoRuntimeNonNilLocalNeeded(g, localIndex) {
		return
	}
	renvoEmitRuntimeNonNilSecondary(g)
}

func renvoAmd64AddCheckedSecondaryFieldOffset(g *renvoLinearGen, fieldOffset int) {
	renvoNonNil(g)
	renvoEmitRuntimeNonNilSecondary(g)
	renvoAmd64AddSecondaryFieldOffset(&g.asm, fieldOffset)
}
