package main

func renvo386EnsureWideBinaryHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideBinaryLabel > 0 {
		return g.wideBinaryLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideBinaryLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x83\xf8\x0d\x75\x18\x8b\x0e\x8b\x56\x04\x8b\x03\xf7\xd0\x21\xc1\x8b\x43\x04\xf7\xd0\x21\xc2\x89\x0f\x89\x57\x04\xc3\x83\xf8\x0e\x72\x05\xe9\xb3\x01\x00\x00\x83\xf8\x00\x74\x1e\x83\xf8\x01\x74\x29\x83\xf8\x02\x74\x34\x83\xf8\x0a\x73\x4b\x83\xf8\x06\x0f\x86\xf2\x00\x00\x00\x83\xf8\x09\x76\x65\xc3\x8b\x06\x03\x03\x89\x07\x8b\x46\x04\x13\x43\x04\x89\x47\x04\xc3\x8b\x06\x2b\x03\x89\x07\x8b\x46\x04\x1b\x43\x04\x89\x47\x04\xc3\x8b\x06\xf7\x23\x89\x07\x89\xd1\x8b\x46\x04\x0f\xaf\x03\x01\xc1\x8b\x06\x0f\xaf\x43\x04\x01\xc8\x89\x47\x04\xc3\x8b\x0e\x8b\x56\x04\x83\xf8\x0a\x74\x0c\x83\xf8\x0b\x74\x0e\x33\x0b\x33\x53\x04\xeb\x0c\x23\x0b\x23\x53\x04\xeb\x05\x0b\x0b\x0b\x53\x04\x89\x0f\x89\x57\x04\xc3\x8b\x0e\x8b\x56\x04\x8b\x73\x04\x85\xf6\x75\x64\x8b\x1b\x83\xfb\x40\x73\x5d\x83\xfb\x20\x73\x2b\x51\x89\xd9\x83\xf8\x07\x74\x17\x5b\x89\xde\x0f\xad\xd6\x83\xf8\x08\x74\x04\xd3\xfa\xeb\x02\xd3\xea\x89\x37\x89\x57\x04\xc3\x5e\x0f\xa5\xf2\xd3\xe6\x89\x37\x89\x57\x04\xc3\x83\xeb\x20\x51\x89\xd9\x83\xf8\x07\x74\x15\x5e\x89\xd6\x83\xf8\x08\x74\x07\xd3\xfe\xc1\xfa\x1f\xeb\xd4\xd3\xee\x31\xd2\xeb\xce\x5e\x89\xf2\xd3\xe2\x31\xf6\x89\x37\x89\x57\x04\xc3\x83\xf8\x09\x74\x0a\x31\xc9\x31\xd2\x89\x0f\x89\x57\x04\xc3\xc1\xfa\x1f\x89\x17\x89\x57\x04\xc3\x55\x57\x50\x8b\x06\x8b\x56\x04\x8b\x0b\x8b\x5b\x04\x31\xf6\x31\xff\x80\x3c\x24\x05\x72\x1a\x89\xd6\xc1\xfe\x1f\x89\xdf\xc1\xff\x1f\x31\xf0\x31\xf2\x29\xf0\x19\xf2\x31\xf9\x31\xfb\x29\xf9\x19\xfb\x56\x57\x53\x51\x31\xc9\x31\xdb\x31\xf6\x31\xff\xbd\x40\x00\x00\x00\xd1\xe0\xd1\xd2\xd1\xd6\xd1\xd7\xd1\xe1\xd1\xd3\x3b\x7c\x24\x04\x72\x11\x77\x05\x3b\x34\x24\x72\x0a\x2b\x34\x24\x1b\x7c\x24\x04\x83\xc9\x01\x4d\x75\xda\xf6\x44\x24\x10\x01\x74\x1b\x8b\x44\x24\x0c\x33\x44\x24\x08\x31\xc1\x31\xc3\x29\xc1\x19\xc3\x8b\x54\x24\x14\x89\x0a\x89\x5a\x04\xeb\x15\x8b\x44\x24\x0c\x31\xc6\x31\xc7\x29\xc6\x19\xc7\x8b\x54\x24\x14\x89\x32\x89\x7a\x04\x83\xc4\x18\x5d\xc3\x83\xe8\x0e\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EnsureWideCompareHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideCompareLabel > 0 {
		return g.wideCompareLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideCompareLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EmitWideHelperCall(g *renvoLinearGen, dest int, left int, right int, mode int, label int) {
	a := &g.asm
	renvoAsmStackMem(a, dest, 0x8d, 0x7d, 0xbd)
	renvoAsmStackMem(a, left, 0x8d, 0x75, 0xb5)
	renvoAsmStackMem(a, right, 0x8d, 0x5d, 0x9d)
	renvoAsmPrimaryImm(a, mode)
	renvoAsmCallLabel(a, label)
}

func renvo386EmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	renvoNonNil(g)
	if mode >= 3 && mode <= 6 {
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvo386EmitWideHelperCall(g, dest, left, right, mode, renvo386EnsureWideBinaryHelper(g))
}

func renvo386EmitWideCompareStack(g *renvoLinearGen, left int, right int, mode int) {
	renvoNonNil(g)
	renvo386EmitWideHelperCall(g, 0, left, right, mode, renvo386EnsureWideCompareHelper(g))
}

func renvo386EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
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
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	renvoAsmEmit32(a, 0x000000c8)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
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
	frame := renvoAlignTo8(g.stackPeak)
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

func renvo386StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return
	}
	if reg == 1 {
		renvoAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return
	}
	if reg == 2 {
		renvoAsmStoreSecondaryStack(a, offset)
		return
	}
	if reg == 3 {
		renvoAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return
	}
	if reg == 4 {
		renvoAsmStorePrimaryStack(a, offset)
		return
	}
	if reg == 5 {
		renvoAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return
	}
	renvoAsmEmit16(a, 0x858b)
	renvoAsmEmit32(a, 8+(reg-6)*4)
	renvoAsmStorePrimaryStack(a, offset)
}

func renvo386AsmMovRaxImm(a *renvoAsm, imm int) {
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
		renvoAsmEmit8(a, 0xb8)
		renvoAsmEmit32(a, imm)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	renvoAsmEmit32(a, imm)
}

func renvo386AsmMovRaxImm64(a *renvoAsm, imm int) {
	renvoAsmPrimaryImm(a, imm)
}

func renvo386AsmMovRdxImm(a *renvoAsm, imm int) {
	if imm == 0 {
		renvoAsmEmit16(a, 0xd231)
		return
	}
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit2(a, 0x6a, imm)
		renvoAsmPopSecondary(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			renvoAsmEmit8(a, 0xba)
			renvoAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			renvoAsmEmit8(a, 0xba)
			renvoAsmEmit32(a, imm)
			return
		}
	}
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, imm)
}

func renvo386AsmMovRaxDataAddr(a *renvoAsm, dataOff int) {
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, dataOff, 0)
}

func renvo386AsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoAsmEmit8(a, 0xa3)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmMovRdiRax(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc389)
}

func renvo386AsmMovRaxRdx(a *renvoAsm) {
	renvoAsmEmit16(a, 0xd089)
}

func renvo386AsmMovRsiRax(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc189)
}

func renvo386AsmMovR8Rax(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc389)
}

func renvo386AsmMovR9Rax(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc389)
}

func renvo386AsmAddRdxRcx(a *renvoAsm) {
	renvoAsmEmit16(a, 0xca01)
}

func renvo386AsmSyscall(a *renvoAsm) {
	renvoAsmEmit16(a, 0x80cd)
}

func renvo386AsmPopRdi(a *renvoAsm) {
	renvoAsmEmit8(a, 0x5b)
}

func renvo386AsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	if base > 0xff {
		renvoAsmEmit8(a, base>>8)
	} else {
		renvoAsmEmit8(a, base)
	}
	if offset >= 0 && offset <= 128 {
		renvoAsmEmit8(a, disp8)
		renvoAsmEmit8(a, -offset)
		return
	}
	renvoAsmEmit8(a, disp32)
	renvoAsmEmit32(a, -offset)
}

func renvo386AsmAddRdxImm(a *renvoAsm, imm int) {
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit3(a, 0x83, 0xc2, imm)
		return
	}
	renvoAsmEmit16(a, 0xc281)
	renvoAsmEmit32(a, imm)
}

func renvo386AsmMemDisp(a *renvoAsm, disp int, op int, disp8 int, disp32 int) {
	if op > 0xff {
		renvoAsmEmit8(a, op>>8)
	} else {
		renvoAsmEmit8(a, op)
	}
	if renvoAsmImmFits8Signed(disp) {
		renvoAsmEmit2(a, disp8, disp)
		return
	}
	renvoAsmEmit8(a, disp32)
	renvoAsmEmit32(a, disp)
}

func renvo386AsmJccLabel(a *renvoAsm, op int, label int) {
	renvoAsmEmit2(a, 0x0f, op)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}

func renvo386AsmLoadQwordRaxIndexRcx8(a *renvoAsm) {
	renvoAsmEmit24(a, 0xc8048b)
}

func renvo386AsmLoadQwordRaxIndexRcxDisp(a *renvoAsm, disp int) {
	renvoAsmEmit8(a, 0x8b)
	if renvoAsmImmFits8Signed(disp) {
		renvoAsmEmit3(a, 0x44, 0x8, disp)
		return
	}
	renvoAsmEmit16(a, 0x0884)
	renvoAsmEmit32(a, disp)
}

func renvo386AsmLoadRaxMemRdxDisp(a *renvoAsm, disp int) {
	if disp == 0 {
		renvoAsmEmit16(a, 0x028b)
		return
	}
	renvoAsmMemDisp(a, disp, 0x8b48, 0x42, 0x82)
}

func renvo386AsmLoadRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	if size == 1 {
		renvoAsmEmit16(a, 0xb60f)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	if size == 2 {
		renvoAsmEmit16(a, 0xbf0f)
		renvoAsmSecondaryDisp(a, disp)
		return
	}
	renvoAsmLoadPrimaryMemSecondaryDisp(a, disp)
}

func renvo386AsmLoadByteRaxIndexRcx(a *renvoAsm) {
	renvoAsmEmit32(a, 0x0804b60f)
}

func renvo386AsmLoadRaxIndexRcxSize(a *renvoAsm, size int) {
	if size == 1 {
		renvoAsmLoadBytePrimaryIndexTertiary(a)
		return
	}
	if size == 2 {
		renvoAsmEmit32(a, 0x4804bf0f)
		return
	}
	if size == 4 {
		renvoAsmEmit24(a, 0x88048b)
		return
	}
	renvoAsmLoadQwordPrimaryIndexTertiary8(a)
}

func renvo386AsmStoreRaxMemRdxRcx8(a *renvoAsm) {
	renvoAsmEmit24(a, 0xca0489)
}

func renvo386AsmStoreRaxMemRdxDisp(a *renvoAsm, disp int) {
	if disp == 0 {
		renvoAsmEmit16(a, 0x0289)
		return
	}
	renvoAsmMemDisp(a, disp, 0x8948, 0x42, 0x82)
}

func renvo386AsmStoreRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
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
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp)
}

func renvo386AsmNormalizeRaxForKind(a *renvoAsm, kind int) {
	if kind == renvoTypeByte {
		renvoAsmEmit24(a, 0xc0b60f)
		return
	}
	if kind == renvoTypeInt8 {
		renvoAsmEmit24(a, 0xc0be0f)
		return
	}
	if kind == renvoTypeInt16 {
		renvoAsmEmit8(a, 0x98)
		return
	}
	if kind == renvoTypeUint16 {
		renvoAsmEmit24(a, 0xc0b70f)
	}
}

func renvo386AsmIncMemRdx(a *renvoAsm) {
	renvoAsmEmit16(a, 0x02ff)
}

func renvo386AsmDecMemRdx(a *renvoAsm) {
	renvoAsmEmit16(a, 0x0aff)
}

func renvo386AsmBoolNotRax(a *renvoAsm) {
	renvoAsmEmit3(a, 0x83, 0xf0, 1)
}

func renvo386AsmCmpRaxImm8(a *renvoAsm, imm int) {
	if imm == 0 {
		renvoAsmEmit16(a, 0xc085)
		return
	}
	renvoAsmEmit3(a, 0x83, 0xf8, imm)
}

func renvo386AsmAddRaxRcx(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc801)
}

func renvo386AsmSubRaxRcx(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc829)
}

func renvo386AsmShlRcxImm(a *renvoAsm, imm int) {
	renvoAsmEmit3(a, 0xc1, 0xe1, imm)
}

func renvo386AsmShlRaxImm(a *renvoAsm, imm int) {
	renvoAsmEmit3(a, 0xc1, 0xe0, imm)
}

func renvo386AsmSarRaxImm(a *renvoAsm, imm int) {
	renvoAsmEmit3(a, 0xc1, 0xf8, imm)
}

func renvo386AsmShrRaxImm(a *renvoAsm, imm int) {
	renvoAsmEmit3(a, 0xc1, 0xe8, imm)
}

func renvo386AsmDivLeftRcxRightRax(a *renvoAsm, mod bool) {
	renvoAsmEmit16(a, 0xc389)
	renvoAsmEmit16(a, 0xc889)
	renvoAsmEmit8(a, 0x99)
	renvoAsmEmit16(a, 0xfbf7)
	if mod {
		renvoAsmEmit16(a, 0xd089)
	}
}

func renvo386AsmCmpRcxRaxSet(a *renvoAsm, setcc int) {
	renvoAsmEmit24(a, 0x0fc139)
	renvoAsmEmit3(a, setcc, 0xc0, 0xf)
	renvoAsmEmit16(a, 0xc0b6)
}

func renvo386EmitSwitchStringCaseTest(g *renvoLinearGen, valueOffset int, lenOffset int, ep *renvoExprParse, idx int, matchLabel int) bool {
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
	renvoAsmMovArg1Rax(a)
	renvoAsmCallLabel(a, label)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, matchLabel)
	return true
}

func renvo386EmitRaxRcxOp(g *renvoLinearGen, tok int) bool {
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
		renvoAsmAddPrimaryTertiary(a)
		return true
	}
	if c0 == '-' {
		renvoAsmEmit16(a, 0xc129)
		renvoAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		renvoAsmEmit24(a, 0xc1af0f)
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
	if c0 == '&' {
		if c1 == '^' {
			renvoAsmEmit16(a, 0xd0f7)
			renvoAsmEmit16(a, 0xc821)
		} else {
			renvoAsmEmit16(a, 0xc821)
		}
		return true
	}
	if c0 == '|' {
		renvoAsmEmit16(a, 0xc809)
		return true
	}
	if c0 == '^' {
		renvoAsmEmit16(a, 0xc831)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			renvoAsmEmit16(a, 0xe0d3)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9e)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			renvoAsmEmit16(a, 0xf8d3)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x95)
		return true
	}
	return false
}

func renvo386EmitCompareJump(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, label int, jumpIfTrue bool) bool {
	p := g.prog
	if e.tok < 0 || e.tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, e.tok)
	end := renvoTokEnd(p, e.tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if !renvoIsComparisonChars(c0, c1) {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	usesFloat := renvoBinaryUsesFloat(g, ep, e)
	right := &ep.exprs[rightIndex]
	rightConst := renvoEvalConstExpr(g, ep, rightIndex)
	if !usesFloat && rightConst.ok && renvoAsmImmFits8Signed(rightConst.value) {
		if !renvoEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		renvoAsmCmpPrimaryImm8(&g.asm, rightConst.value)
		renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, false)
		return true
	}
	if c0 == '=' || c0 == '!' {
		leftType := renvoInferParsedExprType(g, ep, leftIndex)
		rightType := renvoInferParsedExprType(g, ep, rightIndex)
		leftResolved := renvoResolveType(g.meta, leftType)
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
	if !renvo386EmitCompareOperand(g, ep, rightIndex, usesFloat) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvo386EmitCompareOperand(g, ep, leftIndex, usesFloat) {
		return false
	}
	renvoAsmPopTertiary(&g.asm)
	renvoAsmEmit16(&g.asm, 0xc139)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, false)
	return true
}

func renvo386EmitCompareOperand(g *renvoLinearGen, ep *renvoExprParse, idx int, useFloat bool) bool {
	if useFloat {
		return renvoEmitScalarExprForKind(g, ep, idx, renvoTypeFloat64)
	}
	return renvoEmitIntExpr(g, ep, idx)
}

func renvo386EmitStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
			renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset-8)
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
		leftType := renvoInferParsedExprType(g, ep, e.left)
		t := renvoResolveType(meta, leftType)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
		if elem.kind != renvoTypeString {
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
		renvoAsmShlTertiaryImm(a, 4)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		renvoAsmAddSecondaryTertiary(a)
		renvoAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		return true
	}
	if e.kind == renvoExprSelector {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, valueType) {
			return false
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

func renvo386EmitStructReturnExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
		elemType := renvoResolveType(meta, sliceType.elem)
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
			if at == 0 {
				renvoAsmEmit16(a, 0x0189)
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

func renvo386EmitNamedConversionCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
	if resolved.kind == renvoTypeString {
		return renvoEmitStringValueRegs(g, ep, ep.args[e.firstArg])
	}
	resolvedKind := resolved.kind
	if renvoTypeKindIsScalarInt(resolvedKind) {
		if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(&g.asm, resolvedKind)
		return true
	}
	return false
}

func renvo386EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
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
		renvoAsmPopPrimary(a)
	}
	if wordCount > 5 {
		renvoAsmEmit8(a, 0x5f)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if renvoAsmImmFits8Signed(imm) {
			renvoAsmEmit3(a, 0x83, 0xc4, imm)
		} else {
			renvoAsmEmit16(a, 0xc481)
			renvoAsmEmit32(a, imm)
		}
	}
}

func renvo386EmitIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == renvoExprUnary || e.kind == renvoExprBinary || e.kind == renvoExprCall) && renvoExprCanFoldConst(g, ep, idx) {
		resultType := renvoInferParsedExprType(g, ep, idx)
		result := renvoResolveType(g.meta, resultType)
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
					fnIndex := renvoFindMetaFunction(g.meta, e.nameStart, e.nameEnd)
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
			renvoAsmPrimaryImm(a, renvoTypeSize(g.meta, renvoInferParsedExprType(g, ep, ep.args[e.firstArg])))
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
		if e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			conversion := renvoResolveType(g.meta, conversionType)
			if renvoTypeKindIsScalarValue(conversion.kind) {
				return renvoEmitScalarExprForKind(g, ep, ep.args[e.firstArg], conversion.kind)
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
			if !renvoEmitSlicePtrCap(g, ep, ep.args[e.firstArg]) {
				return false
			}
			renvoAsmEmit16(a, 0x5851)
			return true
		}
		if e.argCount == 1 && callee == renvoIdentLen {
			arg := &ep.exprs[ep.args[e.firstArg]]
			if arg.kind == renvoExprString {
				msg := renvoDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				renvoAsmPrimaryImm(a, msgLen)
				return true
			}
			if arg.kind == renvoExprIdent {
				localIndex := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (renvoTypeIsSlice(g.meta, g.locals[localIndex].typ) || renvoTypeIsString(g.meta, g.locals[localIndex].typ)) {
					renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := renvoFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := renvoFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (renvoTypeIsString(g.meta, globalType) || renvoTypeIsSlice(g.meta, globalType)) {
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
			if arg.kind == renvoExprUnary && renvoTokCharIs(p, arg.tok, '*') {
				if !renvoEmitIntExpr(g, ep, arg.left) {
					return false
				}
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
				return true
			}
			argIndex := ep.args[e.firstArg]
			if renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, argIndex)) {
				if !renvoEmitStringValueRegs(g, ep, argIndex) {
					return false
				}
				renvoAsmPushSecondary(a)
				renvoAsmPopPrimary(a)
				return true
			}
			if !renvoEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
				return false
			}
			renvoAsmEmit16(a, 0x5851)
			return true
		}
		if callee == renvoIdentOpen {
			if targetIsWindows() {
				return renvoEmitWindowsOpen(g, ep, idx)
			}
			if e.argCount != 2 {
				return false
			}
			if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			renvoAsmPushPrimary(a)
			if !renvoEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPopTertiary(a)
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
			if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord0(a)
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
			if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			renvoAsmPushPrimary(a)
			if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord1(a)
			renvoAsmPopCallWord0(a)
			renvoAsmPrimaryImm(a, renvoLinuxSysFchmod())
			renvoAsmSyscall(a)
			return true
		}
		if callee == renvoIdentRead {
			if targetIsWindows() {
				return renvoEmitWindowsReadWrite(g, ep, idx, false)
			}
			return renvoEmitBuiltinReadWrite(g, ep, idx, renvoLinuxSysReadSeq(), renvoLinuxSysReadAt())
		}
		if callee == renvoIdentWrite {
			if targetIsWindows() {
				return renvoEmitWindowsReadWrite(g, ep, idx, true)
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
		nativeABI := renvoTypeUsesNativeABI(g.meta, baseType)
		fieldType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		fieldSize := renvoNativeScalarStorageSize(fieldType.kind)
		base := &ep.exprs[e.left]
		if base.kind == renvoExprCall {
			baseResolved := renvoResolveType(g.meta, baseType)
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
			if !renvoTypeIsStruct(g.meta, baseType) {
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
			renvoAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
			if nativeABI {
				renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
			}
			return true
		}
		if base.kind == renvoExprIndex {
			return renvoEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
		if nativeABI {
			renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
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
				renvoAsmEmit16(a, 0x5852)
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
			renvoAsmEmit16(a, 0xd8f7)
			resultType := renvoInferParsedExprType(g, ep, idx)
			result := renvoResolveType(g.meta, resultType)
			renvoAsmNormalizePrimaryForKind(a, result.kind)
			return true
		}
		if renvoTokCharIs(p, e.tok, '+') {
			resultType := renvoInferParsedExprType(g, ep, idx)
			result := renvoResolveType(g.meta, resultType)
			renvoAsmNormalizePrimaryForKind(a, result.kind)
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
			leftResolved := renvoResolveType(g.meta, leftType)
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
			if renvoTypeIsString(g.meta, leftType) || renvoTypeIsString(g.meta, rightType) {
				notEqual := renvoTok2Is(p, e.tok, '!', '=')
				return renvoEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := &ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
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
		resultType := renvoInferParsedExprType(g, ep, idx)
		result := renvoResolveType(g.meta, resultType)
		renvoAsmNormalizePrimaryForKind(a, result.kind)
		return true
	}
	return false
}

func renvo386EmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
		renvoAsmEmit24(a, 0xc1af0f)
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

func renvo386EmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoAsmEmit16(a, 0x5f52)
		renvoAsmEmit16(a, 0x728d)
		renvoAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		renvoAsmEmit16(a, 0x3d8d)
		at := len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x358d)
		at = len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset+8, renvoAbsBssReloc)
		return true
	}
	renvoAsmAddressCallWord0Stack(a, loc.offset)
	renvoAsmAddressCallWord1Stack(a, loc.offset-8)
	return true
}

func renvo386EmitEnsureMemSlice(g *renvoLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	renvoAsmPrimaryBssAddr(a, backingOff)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmPrimaryImm(a, backingSize/elemSize)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 16)
	renvoAsmMarkLabel(a, okLabel)
}

func renvo386EnsureAppendAddrHelper(g *renvoLinearGen) int {
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
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc139)
	renvo386AsmJccLabel(a, 0x8c, noGrowLabel)
	renvoAsmEmit8(a, 0x57)
	renvoAsmEmit8(a, 0x56)
	renvoAsmPushSecondary(a)
	renvoAsmPushTertiary(a)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc085)
	renvo386AsmJccLabel(a, 0x85, capNonZeroLabel)
	renvoAsmPrimaryImm(a, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoAsmPushPrimary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0x4c)
	renvoAsmEmit2(a, 0x24, 8)
	renvoAsmPushTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmCallLabel(a, arenaAllocLabel)
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmEmit16(a, 0xc085)
		renvo386AsmJccLabel(a, 0x85, allocOKLabel)
		renvoAsmEmit3(a, 0x83, 0xc4, 20)
		renvoAsmRet(a)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
	renvoAsmPushPrimary(a)
	renvoAsmEmit3(a, 0x8b, 0x3c, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 20)
	renvoAsmEmit16(a, 0x368b)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xc8)
	renvoAsmEmit16(a, 0xa4f3)
	renvoAsmEmit4(a, 0x8b, 0x7c, 0x24, 20)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit16(a, 0x0789)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 4)
	renvoAsmEmit3(a, 0x89, 0x46, 8)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x54, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xca)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmEmit3(a, 0x83, 0xc4, 24)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit16(a, 0x078b)
	renvoAsmEmit24(a, 0xcaaf0f)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvo386EnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x88\x14\x08\x41\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvo386EnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x89\x14\xc8\x41\x89\x0e\xc3")
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvo386EnsureStringEqualHelper(g *renvoLinearGen) int {
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
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit16(a, 0xce39)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit16(a, 0xf685)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmit16(a, 0x0b8a)
	renvoAsmEmit16(a, 0x0a38)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit8(a, 0x43)
	renvoAsmEmit8(a, 0x42)
	renvoAsmEmit8(a, 0x4e)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func renvo386EmitIndexedStructField(g *renvoLinearGen, ep *renvoExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := renvoInferParsedExprType(g, ep, indexExpr.left)
	sliceType := renvoResolveType(g.meta, leftType)
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemType := renvoResolveType(g.meta, sliceType.elem)
	if elemType.kind != renvoTypeStruct && elemType.kind != renvoTypePointer {
		return false
	}
	fieldOffset := renvoStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	fieldType := renvoStructFieldType(g, sliceType.elem, fieldStart, fieldEnd)
	if !renvoEmitIndexedSelectorAddressSecondary(g, ep, indexIdx, fieldOffset) {
		return false
	}
	fieldResolved := renvoResolveType(g.meta, fieldType)
	fieldSize := renvoScalarKindSize(fieldResolved.kind)
	renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
	return true
}

func renvo386EmitStringPtrExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
		argIndex := ep.args[e.firstArg]
		arg := &ep.exprs[argIndex]
		if arg.kind != renvoExprIdent {
			return false
		}
		localIndex := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
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
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
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
		renvoAsmEmit24(a, 0x08048b)
		return true
	}
	return false
}

func renvo386EmitSelectorAddressRdx(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	meta := g.meta
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
		if baseResolved.kind != renvoTypePointer && baseResolved.kind != renvoTypeStruct {
			return false
		}
		if baseResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmCopyPrimaryToSecondary(a)
			if fieldOffset != 0 {
				renvoAsmAddSecondaryImm(a, fieldOffset)
			}
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
	if base.kind == renvoExprIndex {
		return renvoEmitIndexedSelectorAddressSecondary(g, ep, e.left, fieldOffset)
	}
	if base.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := renvoFindGlobalType(g, base.nameStart, base.nameEnd)
			t := renvoResolveType(meta, globalType)
			if globalOffset < 0 {
				return false
			}
			if t.kind == renvoTypePointer {
				renvoAsmLoadPrimaryBss(a, globalOffset)
				renvoEmitRuntimeNonNilPrimary(g)
				renvoAsmCopyPrimaryToSecondary(a)
				if fieldOffset != 0 {
					renvoAsmAddSecondaryImm(a, fieldOffset)
				}
				return true
			}
			if t.kind != renvoTypeStruct {
				return false
			}
			renvoAsmPrimaryBssAddr(a, globalOffset)
			renvoAsmCopyPrimaryToSecondary(a)
			if fieldOffset != 0 {
				renvoAsmAddSecondaryImm(a, fieldOffset)
			}
			return true
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		if t.kind == renvoTypePointer {
			renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			renvoEmitRuntimeNonNilLocalSecondary(g, localIndex)
			if fieldOffset != 0 {
				renvoAsmAddSecondaryImm(a, fieldOffset)
			}
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
		if t.kind == renvoTypePointer {
			renvoAsmEmit16(a, 0x128b)
			renvoEmitRuntimeNonNilSecondary(g)
		}
		if fieldOffset != 0 {
			renvoAsmAddSecondaryImm(a, fieldOffset)
		}
		return true
	}
	return false
}

func renvoAsmMovArg1Rax(a *renvoAsm) {
	renvoAsmEmit16(a, 0xc689)
}
