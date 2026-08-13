//go:build !renvo

package main

// These helpers compose the small generated direct-emitter contract into the
// value and stack operations used by the shared backend kernel. They are only
// reached by prepared backends (renvoArchRTG); built-in targets retain their
// checked-in, statically selected fast paths.

func renvoRTGAsmAddress(base RTGRegister, index RTGRegister, displacement int, scale int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Base = base
	address.Index = index
	address.Displacement = displacement
	address.Scale = scale
	return address
}

func renvoRTGAsmFrameAddress(offset int) renvoRTGAddress {
	return renvoRTGAsmAddress(renvoRTGFrame, RTGNoRegister, -offset, 1)
}

func renvoRTGAsmDataAddress(offset int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Kind = 1
	address.Addend = offset
	return address
}

func renvoRTGAsmBSSAddress(offset int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Kind = 2
	address.Addend = offset
	return address
}

func renvoRTGAsmPushRegister(a *renvoAsm, source RTGRegister) {
	if renvoRTGABIPushRegister(a, source) {
		return
	}
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectSubtract(a, renvoRTGStack, renvoRTGScratch)
	renvoRTGDirectStoreNative(a, renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1), source)
}

func renvoRTGAsmPushImmediate(a *renvoAsm, value int) {
	if renvoRTGABIPushImmediate(a, value) {
		return
	}
	// Store below SP before using the scratch register to adjust SP, so one
	// reserved scratch location is sufficient.
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(value))
	renvoRTGDirectStoreNative(a,
		renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, -renvoRTGStackWordBytes, 1),
		renvoRTGScratch)
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectSubtract(a, renvoRTGStack, renvoRTGScratch)
}

func renvoRTGAsmPopRegister(a *renvoAsm, destination RTGRegister) {
	if renvoRTGABIPopRegister(a, destination) {
		return
	}
	renvoRTGDirectLoadNative(a, destination,
		renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1))
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectAdd(a, renvoRTGStack, renvoRTGScratch)
}

func renvoRTGAsmLoadFrame(a *renvoAsm, destination RTGRegister, offset int) {
	if renvoRTGABILoadFrame(a, destination, offset) {
		return
	}
	renvoRTGDirectLoadNative(a, destination, renvoRTGAsmFrameAddress(offset))
}

func renvoRTGAsmStoreFrame(a *renvoAsm, offset int, source RTGRegister) {
	if renvoRTGABIStoreFrame(a, offset, source) {
		return
	}
	renvoRTGDirectStoreNative(a, renvoRTGAsmFrameAddress(offset), source)
}

func renvoRTGAsmAddressFrame(a *renvoAsm, destination RTGRegister, offset int) {
	if renvoRTGABIAddressFrame(a, destination, offset) {
		return
	}
	renvoRTGDirectAddress(a, destination, renvoRTGAsmFrameAddress(offset))
}

func renvoRTGAsmLoadSize(a *renvoAsm, destination RTGRegister, address renvoRTGAddress, size int, signed bool) {
	if size <= 1 {
		if signed {
			renvoRTGDirectLoadI8(a, destination, address)
		} else {
			renvoRTGDirectLoadU8(a, destination, address)
		}
		return
	}
	if size == 2 {
		if signed {
			renvoRTGDirectLoadI16(a, destination, address)
		} else {
			renvoRTGDirectLoadU16(a, destination, address)
		}
		return
	}
	if size == 4 && renvoRTGStackWordBytes > 4 {
		if signed {
			renvoRTGDirectLoadI32(a, destination, address)
		} else {
			renvoRTGDirectLoadU32(a, destination, address)
		}
		return
	}
	renvoRTGDirectLoadNative(a, destination, address)
}

func renvoRTGAsmStoreSize(a *renvoAsm, address renvoRTGAddress, source RTGRegister, size int) {
	if size <= 1 {
		renvoRTGDirectStoreU8(a, address, source)
	} else if size == 2 {
		renvoRTGDirectStoreU16(a, address, source)
	} else if size == 4 && renvoRTGStackWordBytes > 4 {
		renvoRTGDirectStoreU32(a, address, source)
	} else {
		renvoRTGDirectStoreNative(a, address, source)
	}
}

func renvoRTGAsmNormalize(a *renvoAsm, kind int) {
	bits := renvoRTGStackWordBytes * 8
	width := bits
	signed := false
	if kind == renvoTypeBool || kind == renvoTypeByte {
		width = 8
	} else if kind == renvoTypeInt8 {
		width = 8
		signed = true
	} else if kind == renvoTypeInt16 {
		width = 16
		signed = true
	} else if kind == renvoTypeInt32 {
		width = 32
		signed = true
	} else if kind == renvoTypeUint16 {
		width = 16
	} else if kind == renvoTypeUint32 {
		width = 32
	}
	if width >= bits {
		return
	}
	shift := byte(bits - width)
	renvoRTGDirectShiftLeftImmediate(a, renvoRTGPrimary, shift)
	if signed {
		renvoRTGDirectShiftRightSignedImmediate(a, renvoRTGPrimary, shift)
	} else {
		renvoRTGDirectShiftRightUnsignedImmediate(a, renvoRTGPrimary, shift)
	}
}

func renvoRTGAsmCompareImmediate(a *renvoAsm, value int) {
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(value))
	renvoRTGDirectCompare(a, renvoRTGPrimary, renvoRTGScratch)
}

func renvoRTGAsmMemoryIncrement(a *renvoAsm, decrement bool) {
	address := renvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, 0, 1)
	renvoRTGDirectLoadNative(a, renvoRTGScratch, address)
	if decrement {
		renvoRTGDirectDecrement(a, renvoRTGScratch)
	} else {
		renvoRTGDirectIncrement(a, renvoRTGScratch)
	}
	renvoRTGDirectStoreNative(a, address, renvoRTGScratch)
}

func renvoRTGAsmBoolNot(a *renvoAsm) {
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, 1)
	renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)
}

func renvoRTGEmitPrimaryTertiaryOp(g *renvoLinearGen, tok int) bool {
	renvoNonNil(g)
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
	a := &g.asm
	if c0 == '+' {
		renvoRTGDirectAdd(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '-' {
		renvoRTGDirectMove(a, renvoRTGScratch, renvoRTGTertiary)
		renvoRTGDirectSubtract(a, renvoRTGScratch, renvoRTGPrimary)
		renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGScratch)
		return true
	}
	if c0 == '*' {
		renvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '/' {
		renvoRTGDirectSignedDivide(a, false)
		return true
	}
	if c0 == '%' {
		renvoRTGDirectSignedDivide(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoRTGDirectMoveImmediate(a, renvoRTGScratch, -1)
			renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)
		}
		renvoRTGDirectBitAnd(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '|' {
		renvoRTGDirectBitOr(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '^' {
		renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '<' && c1 == '<' {
		renvoRTGDirectVariableShift(a, RTGShiftLeft, false)
		return true
	}
	if c0 == '>' && c1 == '>' {
		renvoRTGDirectVariableShift(a, RTGShiftRight, true)
		return true
	}
	setcc := 0
	if c0 == '<' {
		if c1 == '=' {
			setcc = 0x9e
		} else {
			setcc = 0x9c
		}
	} else if c0 == '>' {
		if c1 == '=' {
			setcc = 0x9d
		} else {
			setcc = 0x9f
		}
	} else if c0 == '=' && c1 == '=' {
		setcc = 0x94
	} else if c0 == '!' && c1 == '=' {
		setcc = 0x95
	}
	if setcc != 0 {
		renvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)
		renvoRTGDirectSetCondition(a, renvoRTGConditionFromSetcc(setcc), renvoRTGPrimary)
		return true
	}
	return false
}

func renvoRTGEmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	localCapacity := 16
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
	framePatch := renvoRTGFrameStart(a)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoRTGAsmStoreFrame(a, g.returnStruct, renvoRTGCallWord0)
	}
	renvoBindFunctionParams(g, fnInfoIndex)
	if !renvoBindClosureCaptures(g, fnInfoIndex) ||
		!renvoBindNamedResults(g, fnInfoIndex) ||
		!renvoPrepareFunctionControl(g) ||
		!renvoEmitLinearRange(g, metaFn.bodyStart, metaFn.bodyEnd) {
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
	renvoRTGFrameFinish(a, framePatch, g.stackPeak)
	return true
}

func renvoRTGStoreParamWord(g *renvoLinearGen, word int, offset int) {
	if renvoRTGABIStoreParamWord(&g.asm, word, offset) {
		return
	}
	if word >= 0 && word < 6 {
		registers := []RTGRegister{
			renvoRTGCallWord0, renvoRTGCallWord1, renvoRTGCallWord2,
			renvoRTGCallWord3, renvoRTGCallWord4, renvoRTGCallWord5,
		}
		if registers[word].Valid {
			renvoRTGAsmStoreFrame(&g.asm, offset, registers[word])
			return
		}
		if renvoRTGUnsupportedOperation == 0 {
			renvoRTGUnsupportedOperation = 3001
		}
		return
	}
	overflowBase := 2 * renvoRTGStackWordBytes
	// Current link-register ABIs also use their first call word as the primary
	// result register. Their frame prologues save the link and old frame below
	// the incoming stack pointer, so overflow arguments start at frame itself.
	if renvoRTGCallWord0.Code == renvoRTGPrimary.Code {
		overflowBase = 0
	}
	source := renvoRTGAsmAddress(renvoRTGFrame, RTGNoRegister,
		overflowBase+(word-6)*renvoRTGStackWordBytes, 1)
	renvoRTGDirectLoadNative(&g.asm, renvoRTGPrimary, source)
	renvoRTGAsmStoreFrame(&g.asm, offset, renvoRTGPrimary)
}

func renvoRTGEmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	if renvoRTGABICallWordCount(&g.asm, g.funcLabels[fnIndex], wordCount) {
		return
	}
	registers := []RTGRegister{
		renvoRTGCallWord0, renvoRTGCallWord1, renvoRTGCallWord2,
		renvoRTGCallWord3, renvoRTGCallWord4, renvoRTGCallWord5,
	}
	limit := wordCount
	if limit > len(registers) {
		limit = len(registers)
	}
	for i := 0; i < limit; i++ {
		if !registers[i].Valid {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 3002
			}
			return
		}
		renvoRTGAsmPopRegister(&g.asm, registers[i])
	}
	renvoAsmCallLabel(&g.asm, g.funcLabels[fnIndex])
	if wordCount > len(registers) {
		stackBytes := (wordCount - len(registers)) * renvoRTGStackWordBytes
		renvoRTGDirectMoveImmediate(&g.asm, renvoRTGScratch,
			int64(stackBytes))
		renvoRTGDirectAdd(&g.asm, renvoRTGStack, renvoRTGScratch)
	}
}

func renvoRTGEmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopySource, srcPtr)
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopyDestination, destPtr)
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopyCount, byteCount)
	renvoRTGDirectCopyBytes(&g.asm)
}

func renvoTryCompileScalarProgramRTG(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	renvoRTGUnsupportedOperation = 0
	renvoRTGFailureDetail = -1
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		return renvoCompileResult{}
	}
	g := new(renvoLinearGen)
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	renvoAsmInitWithContext(&g.asm, g.c)
	g.asm.codeOffset = renvoRTGCodeOffset
	for i := 0; i < len(meta.funcs); i++ {
		g.funcLabels = append(g.funcLabels, renvoAsmNewLabel(&g.asm))
	}
	renvoInitFuncQueue(g, len(meta.funcs))
	if renvoRTGPreparedKernelModule != 0 {
		if !renvoBeginKernelModuleRTG(g, appIndex) {
			return renvoCompileResult{}
		}
		if !renvoEmitAllQueuedFunctionsScratch(g) {
			return renvoCompileResult{}
		}
		if renvoRTGUnsupportedOperation != 0 {
			renvoRTGReportFailure(g)
			return renvoCompileResult{}
		}
		renvoAsmPatch(&g.asm)
		data := renvoRTGKernelImage(&g.asm, g.kernelInitLabel, g.kernelExitLabel)
		renvoRTGValidateRelocations(&g.asm)
		if renvoRTGUnsupportedOperation != 0 {
			renvoRTGReportFailure(g)
			return renvoCompileResult{}
		}
		if len(data) == 0 {
			return renvoCompileResult{}
		}
		return renvoCompileResult{data: data, ok: true}
	}
	entryStateOffset := -1
	if renvoRTGEntryStateBytes > 0 {
		entryStateOffset = g.asm.ReserveBSS(renvoRTGEntryStateBytes, renvoRTGStackWordBytes)
	}
	if !renvoRTGEmitEntryStart(&g.asm, entryStateOffset) {
		return renvoCompileResult{}
	}
	renvoLinearMarkFunc(g, appIndex)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return renvoCompileResult{}
	}
	app := &meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(meta, app.resultType) {
		return renvoCompileResult{}
	}
	if app.paramCount > 2 {
		return renvoCompileResult{}
	}
	for i := 0; i < app.paramCount; i++ {
		param := &meta.params[app.firstParam+i]
		if !renvoTypeIsStringSlice(meta, param.typ) {
			return renvoCompileResult{}
		}
	}
	// Native process argument and environment decoding belongs to the selected
	// runtime definition. The hook leaves the ordinary Renvo call words ready
	// for appMain, so the shared lowering path does not know an OS entry ABI.
	if !renvoRTGEmitEntry(&g.asm, app.paramCount, entryStateOffset) {
		return renvoCompileResult{}
	}
	renvoAsmCallLabel(&g.asm, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return renvoCompileResult{}
	}
	if !renvoRTGEmitExit(&g.asm, renvoRTGPrimary) {
		return renvoCompileResult{}
	}
	if !renvoEmitAllQueuedFunctionsScratch(g) {
		return renvoCompileResult{}
	}
	if renvoRTGUnsupportedOperation != 0 {
		renvoRTGReportFailure(g)
		return renvoCompileResult{}
	}
	renvoAsmPatch(&g.asm)
	data := renvoRTGImage(&g.asm)
	renvoRTGValidateRelocations(&g.asm)
	if renvoRTGUnsupportedOperation != 0 {
		renvoRTGReportFailure(g)
		return renvoCompileResult{}
	}
	if len(data) == 0 {
		return renvoCompileResult{}
	}
	return renvoCompileResult{data: data, ok: true}
}

func renvoRTGReportFailure(g *renvoLinearGen) {
	renvoPrintErr("renvo: prepared backend ")
	name, _, _, found := renvoRTGTargetBinding(renvoTargetRTG)
	if found {
		renvoPrintErr(name)
	} else {
		renvoPrintErr("<unknown>")
	}
	if renvoRTGUnsupportedOperation >= 1000 && renvoRTGUnsupportedOperation < 2000 {
		renvoPrintErr(" is missing condition selector ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 1000)
	} else if renvoRTGUnsupportedOperation >= 4000 {
		renvoPrintErr(" reached an unsupported compiler helper ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 4000)
	} else if renvoRTGUnsupportedOperation >= 3000 {
		renvoPrintErr(" used an invalid required register")
	} else if renvoRTGUnsupportedOperation >= 2000 {
		renvoPrintErr(" left an invalid or unresolved relocation ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 2000)
		if renvoRTGFailureDetail >= 0 {
			renvoPrintErr(" targeting label ")
			renvoPrintIntErr(renvoRTGFailureDetail)
			for i := 0; i < len(g.funcLabels); i++ {
				if g.funcLabels[i] == renvoRTGFailureDetail {
					renvoPrintErr(" (")
					write(2, g.prog.src[g.meta.funcs[i].nameStart:g.meta.funcs[i].nameEnd], -1)
					renvoPrintErr(")")
				}
			}
		}
	} else {
		renvoPrintErr(" is missing direct emitter operation ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation)
	}
	renvoPrintErr("\n")
}

var renvoRTGFailureDetail = -1

func renvoRTGValidateRelocations(out *renvoAsm) {
	if out.patchFailed && renvoRTGUnsupportedOperation == 0 {
		renvoRTGUnsupportedOperation = 2004
	}
	for i := 0; i+1 < len(out.relocs); i += 2 {
		at := int(renvo_runtime_UnsafeInt32At(out.relocs, i)) & 2147483647
		label := int(renvo_runtime_UnsafeInt32At(out.relocs, i+1)) & 2147483647
		if at < 0 || at+4 > len(out.code) || renvoAsmLabelPosition(out, label) < 0 {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 2004
			}
			if renvoRTGFailureDetail < 0 {
				renvoRTGFailureDetail = label
			}
		}
	}
	for i := 0; i+2 < len(out.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(out.absRelocs, i)) & 2147483647
		if at < 0 || at+4 > len(out.code) {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 2005
			}
		}
	}
}

func renvoBeginKernelModuleRTG(g *renvoLinearGen, appIndex int) bool {
	renvoNonNil(g)
	a := &g.asm
	g.kernelCallbackLabels = make([]int, len(g.meta.funcs))
	for i := 0; i < len(g.kernelCallbackLabels); i++ {
		g.kernelCallbackLabels[i] = -1
	}
	exitIndex := -1
	for i := 0; i < len(g.meta.funcs); i++ {
		fn := &g.meta.funcs[i]
		if renvoBytesEqualText(g.meta.prog.src, fn.nameStart, fn.nameEnd, "moduleExit") {
			exitIndex = i
		}
	}
	g.kernelInitLabel = renvoAsmNewLabel(a)
	g.kernelExitLabel = -1
	renvoAsmMarkLabel(a, g.kernelInitLabel)
	renvoRTGKernelEntryPrologue(a)
	renvoLinearMarkFunc(g, appIndex)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return false
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return false
	}
	renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, 0)
	renvoRTGKernelEntryEpilogue(a)
	if exitIndex >= 0 {
		g.kernelExitLabel = renvoAsmNewLabel(a)
		renvoAsmMarkLabel(a, g.kernelExitLabel)
		renvoRTGKernelEntryPrologue(a)
		renvoLinearMarkFunc(g, exitIndex)
		renvoAsmCallLabel(a, g.funcLabels[exitIndex])
		renvoRTGKernelEntryEpilogue(a)
	}
	return true
}

func renvoRTGEmitKernelCallbackArgReverse(
	g *renvoLinearGen, ep *renvoExprParse, idx int, funcType int,
) int {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return -1
	}
	fnIndex := renvoFindMetaFunction(g.meta, e.nameStart, e.nameEnd)
	if fnIndex < 0 ||
		renvoFunctionValueMode(g.meta, fnIndex, funcType) != renvoFunctionValueDirect {
		return -1
	}
	renvoLinearMarkFunc(g, fnIndex)
	a := &g.asm
	label := g.kernelCallbackLabels[fnIndex]
	first := label < 0
	if first {
		label = renvoAsmNewLabel(a)
		g.kernelCallbackLabels[fnIndex] = label
	}
	renvoRTGKernelCallbackAddress(a, label)
	renvoRTGAsmPushRegister(a, renvoRTGPrimary)
	if first {
		after := renvoAsmNewLabel(a)
		renvoAsmJmpLabel(a, after)
		renvoAsmMarkLabel(a, label)
		renvoRTGKernelEntryPrologue(a)
		renvoAsmCallLabel(a, g.funcLabels[fnIndex])
		renvoRTGKernelEntryEpilogue(a)
		renvoAsmMarkLabel(a, after)
	}
	return 1
}
