package main

const renvoAmd64ParamStoreRecipes = "\x48\x89\x7d\xbd\x48\x89\x75\xb5\x48\x89\x55\x95\x48\x89\x4d\x8d\x4c\x89\x45\x85\x4c\x89\x4d\x8d"

func renvoAmd64EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	localCapacity := 16
	// bodyStart points after the opening brace, unlike renvoFuncDecl.bodyStart.
	if metaFn.bodyEnd-metaFn.bodyStart >= 512 {
		localCapacity = 32
	}
	g.locals = make([]renvoLocalInfo, localCapacity)
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
	if !renvoEmitLinearRange(g, metaFn.bodyStart, metaFn.bodyEnd) {
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
	if imm>>32 == 0 {
		renvoAsmEmit8(a, 0xb8)
		renvoAsmEmit32(a, imm)
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

func renvoAmd64EmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, srcPtr)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmLoadPrimaryStack(a, destPtr)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadTertiaryStack(a, byteCount)
	// A destination above the source only needs a backward copy when the
	// ranges actually overlap. Forward REP MOVSB is substantially faster for
	// the overwhelmingly common disjoint case. The two forward branches and
	// final short jump are entirely internal to this fixed instruction block.
	renvoAsmEmitText(a, "\x48\x39\xf7\x0f\x86\x1d\x00\x00\x00\x48\x8d\x04\x0e\x48\x39\xc7\x0f\x83\x10\x00\x00\x00\x48\x8d\x74\x0e\xff\x48\x8d\x7c\x0f\xff\xfd\xf3\xa4\xfc\xeb\x03\xfc\xf3\xa4")
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

func renvoAmd64RewritePrimaryLoadCompare(a *renvoAsm, imm int) bool {
	renvoNonNil(a)
	load := a.lastPrimaryLoad
	if load <= 0 || load/8 != len(a.code) {
		return false
	}
	at := load/8 - load%8
	// RIP-relative displacements use the original instruction end as their
	// base, so only stack and register-relative loads can grow by one byte.
	if a.code[at]&0xc7 == 0x05 {
		return false
	}
	// Turn `movq memory, %rax; cmpq immediate, %rax` into the shorter direct
	// memory comparison. Callers use this only when the loaded value is dead
	// after the flags are consumed.
	a.code[at-1] = 0x83
	a.code[at] += 0x38
	a.code = append(a.code, byte(imm))
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
func renvoAmd64AsmCmpRaxImm8(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if imm == 0 {
		// Integer values occupy the full native register on amd64. Testing only
		// EAX makes values whose low word is zero (for example 1 << 40) compare
		// equal to zero and can send both user control flow and the compiler's
		// own immediate-selection logic down the wrong path.
		renvoAsmEmit24(a, 0xc08548)
		return
	}
	renvoAsmEmit4(a, 0x48, 0x83, 0xf8, imm)
}

func renvoAmd64AsmCmpRaxImm8Discard(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoAmd64RewritePrimaryLoadCompare(a, imm) {
		return
	}
	renvoAmd64AsmCmpRaxImm8(a, imm)
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
	return renvoEmitNativeSwitchStringCaseTest(g, valueOffset, lenOffset, ep, idx, matchLabel)
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
	return renvoEmitNativeCompareJump(g, ep, e, label, jumpIfTrue)
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
		if g.c.renvoTargetArch == renvoArchAarch64 || g.c.renvoTargetArch == renvoArchWasm32 {
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
	return renvoEmitNativeStructReturnExpr(g, ep, idx)
}
func renvoAmd64EmitNamedConversionCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	return renvoEmitNativeNamedConversionCall(g, ep, idx)
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
	return renvoEmitNativeIntExpr(g, ep, idx)
}
func renvoAmd64NormalizeExprPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) {
	renvoNormalizeNativeExprPrimary(g, ep, idx)
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
	return renvoEmitNativeSliceSlotAddrs(g, locEp, loc, elemSize)
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
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmEmitText(a, "\x48\x85\xc0")
		renvoAmd64AsmJccLabel(a, 0x85, allocOKLabel)
		renvoAsmEmitText(a, "\x48\x83\xc4\x30\xc3")
		renvoAsmMarkLabel(a, allocOKLabel)
	}
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

func renvoAmd64EnsureAppendBytesHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	arenaAllocLabel := renvoEnsureArenaAllocHelper(g)
	g.appendBytesEmitted = true
	g.appendBytesLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.appendBytesLabel)
	// rdi, rsi, and r9 point at the destination data, length, and capacity
	// slots. rdx and rcx hold the source data and length. The compact helper
	// grows geometrically, uses overlap-safe memmove direction, and returns zero
	// only when growth fails. Its sole external relocation is the arena call at
	// byte 77; internal branches have fixed displacements within this bytecode.
	renvoAsmEmitText(a, "\x48\x85\xc9\x0f\x84\xc8\x00\x00\x00\x4c\x8b\x06\x4d\x89\xc2\x49\x01\xca\x0f\x82\xbd\x00\x00\x00\x4d\x39\x11\x0f\x83\x83\x00\x00\x00\x4d\x8b\x19\x4d\x85\xdb\x75\x06\x41\xbb\x10\x00\x00\x00\x4d\x01\xdb\x0f\x82\x9d\x00\x00\x00\x4d\x39\xd3\x72\xf2\x57\x56\x41\x51\x52\x51\x41\x50\x41\x52\x41\x53\x4c\x89\xd8\xe8\x00\x00\x00\x00\x48\x85\xc0\x75\x05\x48\x83\xc4\x40\xc3\x50\x48\x8b\x4c\x24\x18\x48\x89\xc7\x48\x8b\x74\x24\x40\x48\x8b\x36\xf3\xa4\x48\x8b\x7c\x24\x40\x48\x8b\x04\x24\x48\x89\x07\x4c\x8b\x4c\x24\x30\x4c\x8b\x5c\x24\x08\x4d\x89\x19\x48\x8b\x4c\x24\x20\x48\x8b\x54\x24\x28\x4c\x8b\x44\x24\x18\x4c\x8b\x54\x24\x10\x48\x8b\x74\x24\x38\x48\x83\xc4\x48\x4c\x89\x16\x48\x8b\x07\x4a\x8d\x3c\x00\x48\x89\xd6\x48\x39\xf7\x76\x19\x48\x8d\x04\x0e\x48\x39\xc7\x73\x10\x48\x8d\x74\x0e\xff\x48\x8d\x7c\x0f\xff\xfd\xf3\xa4\xfc\xeb\x02\xf3\xa4\x6a\x01\x58\xc3\x31\xc0\xc3")
	renvoAsmAddReloc(a, len(a.code)-139, arenaAllocLabel)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

// Kept as an empty compatibility hook for the performance source slicer. The
// legacy helper has no callers in the compiler.
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
	return renvoEmitNativeIndexedStructField(g, ep, indexIdx, fieldStart, fieldEnd)
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
	return renvoEmitNativeSelectorAddressSecondary(g, ep, idx)
}
func renvoAmd64AddSecondaryFieldOffset(a *renvoAsm, fieldOffset int) {
	renvoAddNativeSecondaryFieldOffset(a, fieldOffset)
}
func renvoAmd64CheckSecondaryFieldBase(g *renvoLinearGen, localIndex int) {
	renvoCheckNativeSecondaryFieldBase(g, localIndex)
}
func renvoAmd64AddCheckedSecondaryFieldOffset(g *renvoLinearGen, fieldOffset int) {
	renvoAddCheckedNativeSecondaryFieldOffset(g, fieldOffset)
}
