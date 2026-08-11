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
	renvoAsmEmitText(&g.asm, "\x0d\x00\x50\xe3\x08\x00\x00\x1a\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x06\x10\xc1\xe1\x07\x20\xc2\xe1\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x0e\x00\x50\xe3\x00\x00\x00\xba\x94\x00\x00\xea\x00\x00\x50\xe3\x08\x00\x00\x0a\x01\x00\x50\xe3\x0f\x00\x00\x0a\x02\x00\x50\xe3\x16\x00\x00\x0a\x06\x00\x50\xe3\x52\x00\x00\xda\x09\x00\x50\xe3\x2f\x00\x00\xda\x1b\x00\x00\xea\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x10\x91\xe0\x00\x10\x83\xe5\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x10\xa1\xe0\x04\x10\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x10\x51\xe0\x00\x10\x83\xe5\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x10\xc1\xe0\x04\x10\x83\xe5\x1e\xff\x2f\xe1\x00\x60\x94\xe5\x00\x70\x95\xe5\x96\x17\x82\xe0\x04\x80\x94\xe5\x98\x27\x22\xe0\x04\x80\x95\xe5\x96\x28\x22\xe0\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x0a\x00\x50\xe3\x04\x00\x00\x0a\x0b\x00\x50\xe3\x05\x00\x00\x0a\x06\x10\x21\xe0\x07\x20\x22\xe0\x04\x00\x00\xea\x06\x10\x01\xe0\x07\x20\x02\xe0\x01\x00\x00\xea\x06\x10\x81\xe1\x07\x20\x82\xe1\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x00\x10\x94\xe5\x04\x20\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\x00\x00\x57\xe3\x40\x60\xa0\x13\x40\x00\x56\xe3\x40\x60\xa0\x83\x09\x00\x50\xe3\x0d\x00\x00\x0a\x08\x00\x50\xe3\x05\x00\x00\x0a\x00\x00\x56\xe3\x0f\x00\x00\x0a\x01\x10\x91\xe0\x02\x20\xa2\xe0\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x00\x56\xe3\x09\x00\x00\x0a\xa2\x20\xb0\xe1\x61\x10\xa0\xe1\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x00\x56\xe3\x03\x00\x00\x0a\xc2\x20\xb0\xe1\x61\x10\xa0\xe1\x01\x60\x46\xe2\xf9\xff\xff\xea\x00\x10\x83\xe5\x04\x20\x83\xe5\x1e\xff\x2f\xe1\x09\x40\x2d\xe9\x00\x00\x94\xe5\x04\x10\x94\xe5\x00\x60\x95\xe5\x04\x70\x95\xe5\xc1\x4f\xa0\xe1\xc7\x5f\xa0\xe1\x00\xa0\x9d\xe5\x05\x00\x5a\xe3\x62\x00\x00\xba\x04\x00\x20\xe0\x04\x10\x21\xe0\x04\x00\x50\xe0\x04\x10\xc1\xe0\x05\x60\x26\xe0\x05\x70\x27\xe0\x05\x60\x56\xe0\x05\x70\xc7\xe0\x00\x20\xa0\xe3\x00\x30\xa0\xe3\x00\x80\xa0\xe3\x00\x90\xa0\xe3\x40\xa0\xa0\xe3\x00\x00\x90\xe0\x01\x10\xb1\xe0\x08\x80\xb8\xe0\x09\x90\xa9\xe0\x02\x20\x92\xe0\x03\x30\xa3\xe0\x07\x00\x59\xe1\x05\x00\x00\x3a\x01\x00\x00\x8a\x06\x00\x58\xe1\x02\x00\x00\x3a\x06\x80\x58\xe0\x07\x90\xc9\xe0\x01\x20\x82\xe3\x01\xa0\x5a\xe2\xef\xff\xff\x1a\x00\x00\x9d\xe5\x04\x10\x9d\xe5\x01\x00\x10\xe3\x07\x00\x00\x0a\x05\x00\x24\xe0\x00\x20\x22\xe0\x00\x30\x23\xe0\x00\x20\x52\xe0\x00\x30\xc3\xe0\x00\x20\x81\xe5\x04\x30\x81\xe5\x05\x00\x00\xea\x04\x80\x28\xe0\x04\x90\x29\xe0\x04\x80\x58\xe0\x04\x90\xc9\xe0\x00\x80\x81\xe5\x04\x90\x81\xe5\x09\x80\xbd\xe8\x0e\x00\x40\xe2\x04\x10\x94\xe5\x04\x20\x95\xe5\x02\x00\x51\xe1\x0a\x00\x00\x1a\x00\x10\x94\xe5\x00\x20\x95\xe5\x02\x00\x51\xe1\x10\x00\x00\x1a\x00\x00\x50\xe3\x25\x00\x00\x0a\x01\x00\x50\xe3\x21\x00\x00\x0a\x01\x00\x10\xe3\x21\x00\x00\x1a\x1e\x00\x00\xea\x01\x00\x50\xe3\x1e\x00\x00\x0a\x06\x00\x50\xe3\x02\x00\x00\x2a\x02\x00\x51\xe1\x10\x00\x00\xba\x06\x00\x00\xea\x02\x00\x51\xe1\x0d\x00\x00\x3a\x03\x00\x00\xea\x01\x00\x50\xe3\x14\x00\x00\x0a\x02\x00\x51\xe1\x08\x00\x00\x3a\x04\x00\x50\xe3\x10\x00\x00\x0a\x05\x00\x50\xe3\x0e\x00\x00\x0a\x08\x00\x50\xe3\x0c\x00\x00\x0a\x09\x00\x50\xe3\x0a\x00\x00\x0a\x07\x00\x00\xea\x02\x00\x50\xe3\x07\x00\x00\x0a\x03\x00\x50\xe3\x05\x00\x00\x0a\x06\x00\x50\xe3\x03\x00\x00\x0a\x07\x00\x50\xe3\x01\x00\x00\x0a\x00\x00\xa0\xe3\x1e\xff\x2f\xe1\x01\x00\xa0\xe3\x1e\xff\x2f\xe1\x00\x40\xa0\xe3\x00\x50\xa0\xe3\xa1\xff\xff\xea")
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
		renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
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
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
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

func renvoArmEmitRaxRcxOp(g *renvoLinearGen, tok int, rightShiftOpcode int) bool {
	a := &g.asm
	p := g.prog
	start := renvoTokStart(p, tok)
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	// A parsed binary operator always has a right operand, so the byte after
	// its first byte is available. For one-byte operators it cannot be mistaken
	// for an operator suffix without having been tokenized as part of this token.
	c1 := renvo_runtime_UnsafeByteAt(p.src, start+1)
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
	if c0 == '<' || c0 == '>' {
		if c1 == c0 {
			opcode := 0xe1a00010
			if c0 == '>' {
				opcode = rightShiftOpcode
			}
			renvoArmAsmEmit(a, opcode|(renvoArmRegRax<<8)|(renvoArmRegRax<<12)|renvoArmRegRcx)
		} else {
			setcc := 0x9c
			if c0 == '>' {
				setcc = 0x9f
			}
			if c1 == '=' {
				setcc = setcc ^ 2
			}
			renvoArmAsmCmpRcxRaxSet(a, setcc)
		}
		return true
	}
	if (c0 == '=' || c0 == '!') && c1 == '=' {
		setcc := 0x94
		if c0 == '!' {
			setcc++
		}
		renvoArmAsmCmpRcxRaxSet(a, setcc)
		return true
	}
	return false
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

const renvoLinuxArmCodeOffset = 0x74

const renvoLinuxArmSysReadSeq = 3
const renvoLinuxArmSysWriteSeq = 4
const renvoLinuxArmSysOpen = 5
const renvoLinuxArmSysClose = 6
const renvoLinuxArmSysFchmod = 94
const renvoLinuxArmSysReadAt = 180
const renvoLinuxArmSysWriteAt = 181

func renvoArmAsmPrepareReadWriteBuf(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegRsi, renvoArmRegRax)
	renvoArmAsmMovRegReg(a, renvoArmRegRdx, renvoArmRegRcx)
}

func renvoArmAsmMoveOffsetArg(a *renvoAsm) {
	renvoArmAsmMovRegReg(a, renvoArmRegR10, renvoArmRegRax)
}

func compileLinuxArm(input []int, output int) int {
	return compileLinuxArmArena(input, output, 0)
}

func compileLinuxArmArena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetLinuxArm)
	src := renvoMakeByteScratch(786432)
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramArmScratch(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramArm(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	return renvoTryCompileScalarProgramArmScratch(p, meta)
}

func renvoTryCompileScalarProgramArmScratch(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgramArm(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsScratch(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgramArm(g)
}

func renvoTryCompileScalarProgramArmCached(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgramArm(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsCached(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgramArm(g)
}
func renvoBeginScalarProgramArm(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		return nil
	}
	g := new(renvoLinearGen)
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	a.codeOffset = renvoLinuxArmCodeOffset
	if renvoFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(g, len(meta.funcs))
	renvoLinearMarkFunc(g, appIndex)
	if renvoFixedTarget == 0 && meta.c.emitImage {
		renvoArmAsmEmit(a, 0xe92d4800)
		renvoArmAsmMovRegReg(a, renvoArmRegFp, renvoArmRegSp)
		renvoArmAsmAddRegImm(a, renvoArmRegSp, renvoArmRegSp, -16)
		renvoArmAsmStoreRegMem(a, 0, renvoArmRegSp, 0, 4)
		renvoArmAsmStoreRegMem(a, 1, renvoArmRegSp, 4, 4)
		renvoArmAsmStoreRegMem(a, 2, renvoArmRegSp, 8, 4)
		renvoArmAsmStoreRegMem(a, 3, renvoArmRegSp, 12, 4)
	}
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return nil
	}
	entryOK := false
	if renvoFixedTarget == 0 && meta.c.emitImage {
		entryOK = renvoEmitImageEntryArgsArm(g, appIndex)
	} else {
		entryOK = renvoEmitProgramEntryArgsArm(g, appIndex)
	}
	if !entryOK {
		return nil
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return nil
	}
	if renvoFixedTarget == 0 && meta.c.emitImage {
		renvoAsmLeave(a)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmSyscall(a)
	}
	return g
}

func renvoEmitImageEntryArgsArm(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	renvoArmAsmLoadRegMem(&g.asm, 0, renvoArmRegSp, 0, 4)
	renvoArmAsmLoadRegMem(&g.asm, 1, renvoArmRegSp, 4, 4)
	renvoArmAsmLoadRegMem(&g.asm, 2, renvoArmRegSp, 8, 4)
	renvoArmAsmLoadRegMem(&g.asm, 3, renvoArmRegSp, 12, 4)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) {
		return false
	}
	// Native entry ABI: R0=argsData, R1=argsLen, R2=envData,
	// R3=envLen. Renvo slice words use R3/R4/R1 and R2/R5/R6.
	if app.paramCount == 2 {
		if !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) {
			return false
		}
		renvoArmAsmMovRegReg(&g.asm, renvoArmRegR8, 3)
		renvoArmAsmMovRegReg(&g.asm, renvoArmRegR9, 3)
	}
	renvoArmAsmMovRegReg(&g.asm, renvoArmRegRdi, 0)
	renvoArmAsmMovRegReg(&g.asm, renvoArmRegRsi, 1)
	return true
}

func renvoFinishScalarProgramArm(g *renvoLinearGen) renvoCompileResult {
	renvoNonNil(g)
	a := &g.asm
	data := renvoAsmImageArm(a)
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoEmitProgramEntryArgsArm(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	renvoAsmBuildArgvEnvSlicesArm(&g.asm, argsOff, envDataOff, envLenOff)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !renvoTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !renvoTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func renvoAsmBuildArgvEnvSlicesArm(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	base := len(a.code)
	renvoAsmEmitText(a, "\x00\x50\x9d\xe5\x04\x90\x00\xe3\x09\x60\x8d\xe0\x00\x80\x00\xe3\x00\x80\x40\xe3\x08\x80\x8f\xe0\x00\xa0\x00\xe3\x05\x00\x5a\xe1\x10\x00\x00\x0a\x0a\xc1\x86\xe0\x00\xc0\x9c\xe5\x00\xc0\x88\xe5\x00\x00\x00\xe3\x00\x90\x8c\xe0\x00\x90\xd9\xe5\x00\x00\x59\xe3\x02\x00\x00\x0a\x01\x90\x00\xe3\x09\x00\x80\xe0\xf8\xff\xff\xea\x08\x00\x88\xe5\x10\x90\x00\xe3\x09\x80\x88\xe0\x01\x90\x00\xe3\x09\xa0\x8a\xe0\xec\xff\xff\xea\x05\x61\x86\xe0\x04\x90\x00\xe3\x09\x60\x86\xe0\x00\x80\x00\xe3\x00\x80\x40\xe3\x08\x80\x8f\xe0\x00\x60\x00\xe3\x00\xa0\x9d\xe5\x02\x90\x00\xe3\x09\xa0\x8a\xe0\x0a\xa1\x8d\xe0\x00\x60\x00\xe3\x00\xc0\x9a\xe5\x00\x00\x5c\xe3\x10\x00\x00\x0a\x00\xc0\x88\xe5\x00\x00\x00\xe3\x00\x90\x8c\xe0\x00\x90\xd9\xe5\x00\x00\x59\xe3\x02\x00\x00\x0a\x01\x90\x00\xe3\x09\x00\x80\xe0\xf8\xff\xff\xea\x08\x00\x88\xe5\x10\x90\x00\xe3\x09\x80\x88\xe0\x04\x90\x00\xe3\x09\xa0\x8a\xe0\x01\x90\x00\xe3\x09\x60\x86\xe0\xeb\xff\xff\xea\x00\xc0\x00\xe3\x00\xc0\x40\xe3\x0c\xc0\x8f\xe0\x00\x60\x8c\xe5\x00\x30\x00\xe3\x00\x30\x40\xe3\x03\x30\x8f\xe0\x05\x40\xa0\xe1\x05\x10\xa0\xe1\x00\x20\x00\xe3\x00\x20\x40\xe3\x02\x20\x8f\xe0\x06\x50\xa0\xe1")
	renvoAsmAddAbsReloc(a, base+12, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+116, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+232, envLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+248, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+268, envOff, renvoAbsBssReloc)
}

func renvoAsmImageArm(a *renvoAsm) []byte {
	renvoAsmPatchArm(a)
	loadFileSize := a.codeOffset + len(a.code) + len(a.data)
	bssOffset := renvoAsmBssOffset(a)
	if a.c.stripSymbols {
		out := make([]byte, 0, loadFileSize)
		out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, 0)
		out = append(out, a.code...)
		out = append(out, a.data...)
		if renvoFixedTarget == 0 {
			return renvoAppendReplLinkTable(out, a)
		}
		return out
	}
	var sec renvoElfSymbolSections
	renvoBuildElfSymbolSections(a, 0, a.codeOffset, loadFileSize, &sec)
	out := make([]byte, 0, sec.shoff+280)
	out = renvoAppendElfHeaderArm(out, a.codeOffset, loadFileSize, bssOffset, a.bssSize, sec.shoff)
	out = append(out, a.code...)
	out = append(out, a.data...)
	out = renvoAppendUntil(out, sec.symtabOff)
	for i := 0; i < len(sec.symtab); i++ {
		out = append(out, sec.symtab[i])
	}
	out = renvoAppendUntil(out, sec.strtabOff)
	for i := 0; i < len(sec.strtab); i++ {
		out = append(out, sec.strtab[i])
	}
	out = renvoAppendUntil(out, sec.shstrOff)
	for i := 0; i < len(sec.shstrtab); i++ {
		out = append(out, sec.shstrtab[i])
	}
	out = renvoAppendUntil(out, sec.shoff)
	out = renvoAppendElfSectionHeaders(out, &sec, a, 0)
	if renvoFixedTarget == 0 {
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}

func renvoAsmPatchArm(a *renvoAsm) {
	renvoAsmPatch(a)
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		insn := renvoGet32At(a.code, at)
		reg := (insn >> 12) & 15
		delta := target - (a.codeOffset + at + 16)
		renvoArmAsmPatchMovRegImmAt(a, at, reg, delta)
	}
}

func renvoAppendElfHeaderArm(out []byte, entryOff int, fileSize int, bssOffset int, bssSize int, shoff int) []byte {
	start := len(out)
	base := 0
	// The ELF and program-header layouts are fixed. Keep their invariant bytes
	// together and patch the seven fields that vary per output image.
	header := "\x7f\x45\x4c\x46\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x28\x00\x01\x00\x00\x00\x00\x00\x01\x00\x34\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x34\x00\x20\x00\x02\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x10\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x06\x00\x00\x00\x00\x10\x00\x00"
	for i := 0; i < len(header); i++ {
		out = append(out, header[i])
	}
	renvoPut32At(out, start+24, base+entryOff)
	renvoPut32At(out, start+32, shoff)
	if shoff != 0 {
		out[start+46] = 40
		out[start+48] = 7
		out[start+50] = 6
	}
	renvoPut32At(out, start+60, base)
	renvoPut32At(out, start+64, base)
	renvoPut32At(out, start+68, fileSize)
	renvoPut32At(out, start+72, fileSize)
	renvoPut32At(out, start+88, bssOffset)
	renvoPut32At(out, start+92, base+bssOffset)
	renvoPut32At(out, start+96, base+bssOffset)
	renvoPut32At(out, start+104, bssSize)
	return out
}
