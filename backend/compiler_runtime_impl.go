package main

func renvoReadAll(fd int, out []byte) []byte {
	var buf []byte
	for {
		base := len(out)
		if renvoFixedTarget == renvoTargetWasiWasm32 &&
			base == cap(out) && base >= 262144 {
			newCapacity := base + base/2 + 262144
			expanded := make([]byte, base, newCapacity)
			copy(expanded, out)
			out = expanded
		}
		if base < cap(out) {
			expanded := out
			renvoTruncBytes(&expanded, cap(out))
			n := read(fd, expanded[base:], -1)
			if n <= 0 {
				return out
			}
			out = expanded
			renvoTruncBytes(&out, base+n)
			continue
		}
		if len(buf) == 0 {
			buf = make([]byte, 0, 1024)
			renvoTruncBytes(&buf, cap(buf))
		}
		n := read(fd, buf, -1)
		if n <= 0 {
			return out
		}
		// buf has independent fixed backing, so a scalar append loop avoids the
		// overlap machinery required by the general append(dst, src...) lowering.
		for i := 0; i < n; i++ {
			out = append(out, buf[i])
		}
	}
}

func renvoEmitLinearPrintStmt(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.exprStart < 0 || stmt.exprStart >= renvoTokCount(p) {
		return false
	}
	ep := renvoNewExprParse()
	renvoParseExpressionInto(ep, p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != renvoExprCall {
		return false
	}
	builtinPrintln := renvoExprIdentCode(p, ep, root.left) == renvoIdentPrintln
	println := builtinPrintln || renvoExprIsIdentText(p, ep, root.left, "Println")
	fmtPrintln := false
	if !println && !renvoExprIsIdentText(p, ep, root.left, "print") {
		candidate := &ep.exprs[root.left]
		if candidate.kind != renvoExprIdent || candidate.nameEnd-candidate.nameStart != 24 || !renvoBytesEqualText(p.src, candidate.nameStart, candidate.nameEnd, "renvo_runtime_FmtPrintln") {
			return false
		}
		fmtPrintln = true
		println = true
	}
	fnIndex := renvoFuncInfoFromCall(g, ep, root.left)
	if fmtPrintln && fnIndex < 0 {
		return false
	}
	callee := &ep.exprs[root.left]
	if !fmtPrintln && (fnIndex >= 0 || renvoFindLocalIndex(g, callee.nameStart, callee.nameEnd) >= 0) {
		return false
	}
	// More than one argument must be evaluated completely before Println starts
	// writing. Keep those calls on the generic path until the backend has a
	// compact multi-value staging representation.
	if fmtPrintln && root.argCount > 1 {
		return false
	}
	fd := 1
	if builtinPrintln {
		fd = 2
	}
	for i := 0; i < root.argCount; i++ {
		if println && i > 0 && !renvoEmitPrintStaticByte(g, ' ', fd) {
			return false
		}
		argIndex := ep.args[root.firstArg+i]
		argType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
		renvoNonNil(argType)
		if fmtPrintln && argType.kind != renvoTypeString {
			return false
		}
		if argType.kind == renvoTypeString {
			if !renvoEmitStringValueRegs(g, ep, argIndex) {
				return false
			}
		} else if renvoTypeKindIsScalarInt(argType.kind) {
			if !renvoEmitIntExpr(g, ep, argIndex) {
				return false
			}
			renvoAsmNormalizePrimaryForKind(&g.asm, argType.kind)
			renvoAsmCallLabel(&g.asm, renvoEnsurePrintIntHelper(g))
		} else {
			return false
		}
		if !renvoEmitWriteValueRegs(g, fd) {
			return false
		}
	}
	return !println || renvoEmitPrintStaticByte(g, '\n', fd)
}

func renvoEmitPrintStaticByte(g *renvoLinearGen, value byte, fd int) bool {
	renvoNonNil(g)
	offset := len(g.asm.data)
	g.asm.data = append(g.asm.data, value)
	renvoAsmPrimaryDataAddr(&g.asm, offset)
	renvoAsmSecondaryImm(&g.asm, 1)
	return renvoEmitWriteValueRegs(g, fd)
}

func renvoPrintMirrorFunction(g *renvoLinearGen) int {
	renvoNonNil(g)
	for i := 0; i < len(g.meta.funcs); i++ {
		fn := &g.meta.funcs[i]
		if fn.receiverType == 0 && fn.paramCount == 1 &&
			renvoTypeIsString(g.meta, g.meta.params[fn.firstParam].typ) &&
			renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_PrintMirror") {
			return i
		}
	}
	return -1
}

func renvoEmitPrintMirror(g *renvoLinearGen) {
	renvoNonNil(g)
	fnIndex := renvoPrintMirrorFunction(g)
	if fnIndex < 0 || fnIndex == g.currentFunc {
		return
	}
	// Preserve the pointer and length for the real stdout write below while a
	// second pair becomes the mirror function's string argument.
	renvoAsmPushSecondary(&g.asm)
	renvoAsmPushPrimary(&g.asm)
	renvoAsmPushSecondary(&g.asm)
	renvoAsmPushPrimary(&g.asm)
	renvoEmitCallWithWordCount(g, fnIndex, renvoBackendStringWordCount)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmPopSecondary(&g.asm)
}

func renvoEmitWriteValueRegs(g *renvoLinearGen, fd int) bool {
	renvoNonNil(g)
	a := &g.asm
	renvoEmitPrintMirror(g)
	if renvoPreparedBackend != 0 {
		renvoRTGDirectMove(a, renvoRTGCallWord2, renvoRTGSecondary)
		renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)
		renvoRTGDirectMoveImmediate(a, renvoRTGCallWord0, int64(fd))
		return renvoRTGEmitRuntimeOperation(a, RTGRuntimeWrite)
	}
	if renvoFixedTarget == renvoTargetLinuxKernelAmd64 ||
		renvoPreparedBackend == 0 && renvoFixedTarget == 0 && targetIsKernelModule(g.c) {
		renvoAmd64EmitKernelPrintValue(a)
		return true
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		if g.c.renvoTargetArch == renvoArch386 {
			label := renvoWin386EmitReadWriteHelper(g, true)
			renvoAsmEmit16(a, 0xc689)
			renvoAsmPrimaryImm(a, -1)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmPrimaryImm(a, fd)
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmCallLabel(a, label)
			return true
		}
		if g.c.renvoTargetArch == renvoArchAarch64 {
			label := renvoWinArm64EmitReadWriteHelper(g, true)
			renvoAsmCopyPrimaryToCallWord1(a)
			renvoAsmPrimaryImm(a, fd)
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPrimaryImm(a, -1)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmCallLabel(a, label)
			return true
		}
		label := renvoWinAmd64EmitReadWriteHelper(g, true)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAsmPrimaryImm(a, fd)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		renvoAarch64AsmMovRegReg(a, 2, renvoAarch64RegRdx)
		renvoAarch64AsmMovRegReg(a, 1, renvoAarch64RegRax)
		renvoAarch64AsmMovRegImm(a, 0, fd)
		renvoDarwinArm64CallImport(a, renvoDarwinImportWrite)
		return true
	}
	renvoAsmPushImm(a, fd)
	renvoAsmPopCallWord0(a)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmPrimaryImm(a, renvoLinuxSysWriteSeq(g.c.renvoTargetOS, g.c.renvoTargetArch))
	renvoAsmSyscall(a)
	return true
}

func renvoEmitBuiltinReadWrite(g *renvoLinearGen, ep *renvoExprParse, idx int, seqSyscall int, offSyscall int) bool {
	renvoNonNil(g, ep)
	if renvoPreparedBackend != 0 {
		operation := RTGRuntimeRead
		if seqSyscall == renvoLinuxSysWriteSeq(g.c.renvoTargetOS, g.c.renvoTargetArch) ||
			seqSyscall == renvoDarwinImportWrite {
			operation = RTGRuntimeWrite
		}
		return renvoEmitPreparedReadWrite(g, ep, idx, operation)
	}
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	fdStart := ep.exprs[idx].tok + 1
	fdEnd := renvoFindExprBoundary(p, fdStart, ep.end)
	fdEp := renvoNewExprParse()
	renvoParseExpressionInto(fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	fdIndex := len(fdEp.exprs) - 1
	if !renvoEmitIntExpr(g, fdEp, fdIndex) {
		return false
	}
	renvoAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := renvoEvalConstExpr(g, ep, offIndex)
	offsetRead := true
	if offConst.ok && offConst.value < 0 {
		offsetRead = false
	}
	if offsetRead {
		if offConst.ok {
			renvoAsmPrimaryImm(a, offConst.value)
		} else {
			if !renvoEmitIntExpr(g, ep, offIndex) {
				return false
			}
		}
		renvoAsmPushPrimary(a)
	}
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	renvoAsmPrepareReadWriteBuf(a)
	if offsetRead {
		renvoAsmPopPrimary(a)
		renvoAsmMoveOffsetArg(a)
	}
	renvoAsmPopCallWord0(a)
	if offsetRead {
		renvoAsmPrimaryImm(a, offSyscall)
	} else {
		renvoAsmPrimaryImm(a, seqSyscall)
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		importID := seqSyscall
		if offsetRead {
			importID = offSyscall
		}
		argCount := 3
		if offsetRead {
			argCount = 4
		}
		renvoDarwinArm64CallVirtualArgs(a, importID, argCount)
		return true
	}
	renvoAsmSyscall(a)
	return true
}

func renvoEmitPreparedReadWrite(
	g *renvoLinearGen, ep *renvoExprParse, idx int, operation int,
) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	expression := &ep.exprs[idx]
	if expression.argCount != 3 {
		return false
	}
	firstArg := expression.firstArg
	if !renvoEmitIntExpr(g, ep, ep.args[firstArg]) {
		return false
	}
	renvoAsmPushPrimary(a)
	offsetIndex := ep.args[firstArg+2]
	offset := renvoEvalConstExpr(g, ep, offsetIndex)
	hasOffset := !offset.ok || offset.value >= 0
	if hasOffset {
		if offset.ok {
			renvoAsmPrimaryImm(a, offset.value)
		} else if !renvoEmitIntExpr(g, ep, offsetIndex) {
			return false
		}
		renvoAsmPushPrimary(a)
	}
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)
	renvoRTGDirectMove(a, renvoRTGCallWord2, renvoRTGTertiary)
	if hasOffset {
		renvoRTGAsmPopRegister(a, renvoRTGCallWord3)
		operation += RTGRuntimeReadAt - RTGRuntimeRead
	}
	renvoRTGAsmPopRegister(a, renvoRTGCallWord0)
	return renvoRTGEmitRuntimeOperation(a, operation)
}

func renvoEvalBuiltinConst(g *renvoLinearGen, nameStart int, nameEnd int) renvoConstResult {
	renvoNonNil(g)
	p := g.prog
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "iota") {
		if g.constEvalIotaValid != 0 {
			return renvoConstResultOk(g.constEvalIota)
		}
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "nil") {
		return renvoConstResultOk(0)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_RDONLY") {
		return renvoConstResultOk(0)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_WRONLY") {
		return renvoConstResultOk(1)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_RDWR") {
		return renvoConstResultOk(2)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_CREATE") {
		if targetIsDarwin(g.c.renvoTargetOS) || targetIsBSD(g.c.renvoTargetOS) {
			return renvoConstResultOk(512)
		}
		return renvoConstResultOk(64)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_TRUNC") {
		if targetIsDarwin(g.c.renvoTargetOS) || targetIsBSD(g.c.renvoTargetOS) {
			return renvoConstResultOk(1024)
		}
		return renvoConstResultOk(512)
	}
	var r renvoConstResult
	return r
}

func renvoEmitTargetRuntime(g *renvoLinearGen, ep *renvoExprParse, idx int, callee int) bool {
	renvoNonNil(g, ep)
	if renvoPreparedBackend != 0 {
		return renvoEmitPreparedTargetRuntime(g, ep, idx, callee)
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		if callee == renvoIdentRead || callee == renvoIdentWrite {
			return renvoEmitWindowsReadWrite(g, ep, idx, callee == renvoIdentWrite)
		}
		if callee == renvoIdentOpen {
			return renvoEmitWindowsOpen(g, ep, idx)
		}
		if callee == renvoIdentClose {
			return renvoEmitWindowsClose(g, ep, idx)
		}
		return renvoEmitWindowsChmod(g, ep, idx)
	}
	if callee == renvoIdentRead || callee == renvoIdentWrite {
		isWrite := callee == renvoIdentWrite
		if targetIsDarwin(g.c.renvoTargetOS) {
			if isWrite {
				return renvoEmitBuiltinReadWrite(g, ep, idx, renvoDarwinImportWrite, renvoDarwinImportPwrite)
			}
			return renvoEmitBuiltinReadWrite(g, ep, idx, renvoDarwinImportRead, renvoDarwinImportPread)
		}
		if isWrite {
			return renvoEmitBuiltinReadWrite(g, ep, idx, renvoLinuxSysWriteSeq(g.c.renvoTargetOS, g.c.renvoTargetArch), renvoLinuxSysWriteAt(g.c.renvoTargetOS, g.c.renvoTargetArch))
		}
		return renvoEmitBuiltinReadWrite(g, ep, idx, renvoLinuxSysReadSeq(g.c.renvoTargetOS, g.c.renvoTargetArch), renvoLinuxSysReadAt(g.c.renvoTargetOS, g.c.renvoTargetArch))
	}
	e := &ep.exprs[idx]
	a := &g.asm
	firstArgIndex := -1
	if e.argCount > 0 {
		firstArgIndex = renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	}
	if callee == renvoIdentOpen {
		if e.argCount != 2 {
			return false
		}
		if targetIsDarwin(g.c.renvoTargetOS) {
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
		if g.c.renvoTargetArch == renvoArchAarch64 {
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
			renvoAsmPrimaryImm(a, renvoLinuxSysOpen(g.c.renvoTargetOS, g.c.renvoTargetArch))
			renvoAsmSyscall(a)
			return true
		}
		if g.c.renvoTargetArch == renvoArchWasm32 {
			if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
				return false
			}
			renvoAsmPushPrimary(a)
			if !renvoEmitStringValueRegs(g, ep, firstArgIndex) {
				return false
			}
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPopCallWord1(a)
			renvoAsmPrimaryImm(a, renvoLinuxSysOpen(g.c.renvoTargetOS, g.c.renvoTargetArch))
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
		if g.c.renvoTargetArch == renvoArch386 {
			renvoAsmPopTertiary(a)
		} else {
			renvoAsmPopCallWord1(a)
		}
		renvoAsmSecondaryImm(a, 493)
		renvoAsmPrimaryImm(a, renvoLinuxSysOpen(g.c.renvoTargetOS, g.c.renvoTargetArch))
		renvoAsmSyscall(a)
		return true
	}
	if callee == renvoIdentClose {
		if e.argCount != 1 {
			return false
		}
		if !renvoEmitIntExpr(g, ep, firstArgIndex) {
			return false
		}
		renvoAsmCopyPrimaryToCallWord0(a)
		if targetIsDarwin(g.c.renvoTargetOS) {
			renvoDarwinArm64CallVirtualArgs(a, renvoDarwinImportClose, 1)
			return true
		}
		renvoAsmPrimaryImm(a, renvoLinuxSysClose(g.c.renvoTargetOS, g.c.renvoTargetArch))
		renvoAsmSyscall(a)
		return true
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
	if targetIsDarwin(g.c.renvoTargetOS) {
		renvoDarwinArm64CallVirtualArgs(a, renvoDarwinImportFchmod, 2)
		return true
	}
	renvoAsmPrimaryImm(a, renvoLinuxSysFchmod(g.c.renvoTargetOS, g.c.renvoTargetArch))
	renvoAsmSyscall(a)
	return true
}

func renvoEmitPreparedTargetRuntime(
	g *renvoLinearGen, ep *renvoExprParse, idx int, callee int,
) bool {
	renvoNonNil(g, ep)
	expression := &ep.exprs[idx]
	a := &g.asm
	if callee == renvoIdentRead || callee == renvoIdentWrite {
		operation := RTGRuntimeRead
		if callee == renvoIdentWrite {
			operation = RTGRuntimeWrite
		}
		return renvoEmitPreparedReadWrite(g, ep, idx, operation)
	}
	if callee == renvoIdentOpen {
		if expression.argCount != 2 {
			return false
		}
		if !renvoEmitIntExpr(g, ep, ep.args[expression.firstArg+1]) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitStringPtrExpr(g, ep, ep.args[expression.firstArg]) {
			return false
		}
		renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGPrimary)
		renvoRTGAsmPopRegister(a, renvoRTGCallWord1)
		renvoRTGDirectMoveImmediate(a, renvoRTGCallWord2, 493)
		return renvoRTGEmitRuntimeOperation(a, RTGRuntimeOpen)
	}
	if callee == renvoIdentClose {
		if expression.argCount != 1 ||
			!renvoEmitIntExpr(g, ep, ep.args[expression.firstArg]) {
			return false
		}
		renvoRTGDirectMove(a, renvoRTGCallWord0, renvoRTGPrimary)
		return renvoRTGEmitRuntimeOperation(a, RTGRuntimeClose)
	}
	if callee != renvoIdentChmod || expression.argCount != 2 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, ep.args[expression.firstArg]) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitIntExpr(g, ep, ep.args[expression.firstArg+1]) {
		return false
	}
	renvoRTGDirectMove(a, renvoRTGCallWord1, renvoRTGPrimary)
	renvoRTGAsmPopRegister(a, renvoRTGCallWord0)
	return renvoRTGEmitRuntimeOperation(a, RTGRuntimeChmod)
}
func renvoEmitExitStatus(g *renvoLinearGen) bool {
	renvoNonNil(g)
	a := &g.asm
	if renvoPreparedBackend != 0 {
		return renvoRTGEmitExit(a, renvoRTGPrimary)
	}
	if g.c.renvoTargetArch == renvoArchWasm32 {
		renvoAsmEmit8(a, renvoWasm32OpExit)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		if targetIsDarwin(g.c.renvoTargetOS) {
			renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
			renvoDarwinArm64CallImport(a, renvoDarwinImportExit)
		} else {
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPrimaryImm(a, 93)
			renvoAsmSyscall(a)
		}
		return true
	}
	if g.c.renvoTargetArch == renvoArchArm {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmSyscall(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArch386 {
		if targetIsWindows(g.c.renvoTargetOS) {
			renvoAsmPushPrimary(a)
			renvoWin386CallImport(a, renvoWinImportExitProcess)
		} else {
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPrimaryImm(a, 1)
			renvoAsmSyscall(a)
		}
		return true
	}
	if g.c.renvoTargetArch != renvoArchAmd64 {
		return false
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		renvoWinAmd64EmitExit(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, renvoHostedAmd64SysExit(g.c.renvoTargetOS))
		renvoAsmSyscall(a)
	}
	return true
}

func renvoEmitLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if renvoIsHostedObjectAmd64(g.c) {
		vectorMask := renvoObjectCallVectorMask(g, fn, wordCount)
		if vectorMask < 0 {
			return false
		}
		importID := renvoAsmAddPreparedStaticImport(&g.asm,
			fn.linkDLLStart, fn.linkDLLEnd,
			fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
		if importID < 0 {
			return false
		}
		renvoAmd64EmitObjectStaticCall(&g.asm, importID, wordCount|vectorMask<<8)
		return true
	}
	if renvoFixedTarget == renvoTargetLinuxKernelAmd64 ||
		renvoPreparedBackend == 0 && renvoFixedTarget == 0 && targetIsKernelModule(g.c) {
		return renvoAmd64EmitKernelLinkStaticCall(g, fn, wordCount)
	}
	if renvoPreparedBackend != 0 && renvoRTGPreparedKernelModule != 0 &&
		renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "kernel") {
		importID := renvoAsmAddKernelImport(
			&g.asm, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
		if importID < 0 {
			return false
		}
		return renvoRTGEmitStaticCall(&g.asm, importID, wordCount)
	}
	if renvoPreparedBackend != 0 {
		importID := renvoAsmAddPreparedStaticImport(&g.asm,
			fn.linkDLLStart, fn.linkDLLEnd,
			fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
		return renvoRTGEmitStaticCall(&g.asm, importID, wordCount)
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		return renvoDarwinArm64EmitLinkStaticCall(g, fn, wordCount)
	}
	if g.c.renvoTargetOS != renvoOSWindows {
		return false
	}
	importID := renvoAsmAddWinStaticImport(&g.asm, fn.linkDLLStart, fn.linkDLLEnd, fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if renvoPreparedBackend != 0 {
		return renvoRTGEmitStaticCall(&g.asm, importID, wordCount)
	}
	if g.c.renvoTargetArch == renvoArch386 {
		renvoWin386CallImport(&g.asm, importID)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoWinArm64CallStaticImport(&g.asm, importID, wordCount)
		return true
	}
	if g.c.renvoTargetArch != renvoArchAmd64 {
		return false
	}
	renvoWinAmd64CallStaticImport(&g.asm, importID, wordCount)
	return true
}

func renvoObjectCallVectorMask(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) int {
	if wordCount < 0 || wordCount != fn.paramCount {
		return -1
	}
	mask := 0
	integers := 0
	vectors := 0
	for i := 0; i < fn.paramCount; i++ {
		typ := renvoResolveType(g.meta, g.meta.params[fn.firstParam+i].typ)
		if typ.kind == renvoTypeFloat64 {
			mask |= 1 << i
			vectors++
		} else {
			integers++
		}
	}
	if integers > 6 || vectors > 8 {
		return -1
	}
	return mask
}

// renvoEmitTargetStaticCall returns -1 when the declaration is not a foreign
// call for the active target, zero for an invalid target binding, and one when
// the call was emitted.
func renvoEmitTargetStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) int {
	renvoNonNil(g, fn)
	if g.c.objectFile {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoPreparedBackend != 0 && renvoRTGPreparedObject != 0 {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoFixedTarget == renvoTargetLinuxKernelAmd64 ||
		renvoPreparedBackend == 0 && renvoFixedTarget == 0 && targetIsKernelModule(g.c) {
		if !renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "kernel") {
			return 0
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoPreparedBackend != 0 {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		if renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) != '/' {
			return -1
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		if renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) == '/' {
			return -1
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	return -1
}

func renvoAsmAddPreparedStaticImport(
	a *renvoAsm, libraryStart int, libraryEnd int,
	nameStart int, nameEnd int, src []byte,
) int {
	renvoNonNil(a)
	library := renvoStringFromBytes(src, libraryStart, libraryEnd)
	name := renvoStringFromBytes(src, nameStart, nameEnd)
	for i := 0; i < len(a.staticImports); i++ {
		if a.staticImports[i].dll == library && a.staticImports[i].name == name {
			return i
		}
	}
	a.staticImports = append(a.staticImports, renvoStaticImport{dll: library, name: name})
	return len(a.staticImports) - 1
}
func renvoEmitRuntimeArenaDiscard(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if g.c.renvoTargetArch != renvoArchAmd64 || g.c.renvoTargetOS != renvoOSLinux {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	endOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmStorePrimaryStack(&g.asm, startOff)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmStorePrimaryStack(&g.asm, endOff)
	return renvoEmitRuntimeArenaDiscardStackRange(g, startOff, endOff)
}

func renvoEmitRuntimeArenaDiscardSlice(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if g.c.renvoTargetArch != renvoArchAmd64 || g.c.renvoTargetOS != renvoOSLinux {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	sliceType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, sliceType.elem)
	if !renvoEmitSlicePtrLen(g, ep, argIndex) {
		return false
	}
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	endOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(&g.asm, startOff)
	renvoAsmMulTertiaryImm(&g.asm, elemSize)
	renvoAsmCopyTertiaryToPrimary(&g.asm)
	renvoAsmLoadTertiaryStack(&g.asm, startOff)
	renvoAsmAddPrimaryTertiary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, endOff)
	return renvoEmitRuntimeArenaDiscardStackRange(g, startOff, endOff)
}

func renvoEmitRuntimeArenaDiscardStackRange(g *renvoLinearGen, startOff int, endOff int) bool {
	renvoNonNil(g)
	if renvoFixedTarget != 0 &&
		renvoFixedTarget != renvoTargetLinuxAmd64 &&
		renvoFixedTarget != renvoTargetLinux386 &&
		renvoFixedTarget != renvoTargetLinuxAarch64 &&
		renvoFixedTarget != renvoTargetLinuxArm ||
		renvoFixedTarget == 0 && g.c.renvoTargetOS != renvoOSLinux {
		return true
	}
	a := &g.asm
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryStack(a, startOff)
	renvoAmd64AsmAddRaxImm32(a, 4095)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmStorePrimaryStack(a, startOff)
	renvoAsmLoadPrimaryStack(a, endOff)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmLoadTertiaryStack(a, startOff)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAmd64AsmJccLabel(a, 0x8e, doneLabel)
	renvoAsmLoadPrimaryStack(a, startOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmSecondaryImm(a, 4)
	renvoAsmPrimaryImm(a, 28)
	renvoAsmSyscall(a)
	renvoAsmMarkLabel(a, doneLabel)
	renvoAsmPrimaryImm(a, 0)
	return true
}

func renvoEmitRuntimeArenaPersistReset(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoStringHeapOffsets(g)
	a := &g.asm
	if g.c.renvoTargetArch == renvoArchAmd64 && g.c.renvoTargetOS == renvoOSLinux {
		renvoEmitRuntimeArenaPersistResetMadvise(g)
		return true
	}
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	return true
}

func renvoEmitRuntimeArenaPersistResetMadvise(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	markOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	oldOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmStorePrimaryStack(a, markOff)
	renvoAsmCopyBssToStackSlot(a, g.stringHeapEndOff, oldOff)
	renvoAsmLoadPrimaryStack(a, markOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	renvoAsmLoadPrimaryStack(a, oldOff)
	renvoAmd64AsmAddRaxImm32(a, 4095)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmStorePrimaryStack(a, startOff)
	renvoAsmLoadPrimaryStack(a, markOff)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmLoadTertiaryStack(a, startOff)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAmd64AsmJccLabel(a, 0x8e, doneLabel)
	renvoAsmLoadPrimaryStack(a, startOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmSecondaryImm(a, 4)
	renvoAsmPrimaryImm(a, 28)
	renvoAsmSyscall(a)
	renvoAsmMarkLabel(a, doneLabel)
}

func renvoAmd64AsmAddRaxImm32(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x0548)
	renvoAsmEmit32(a, imm)
}

func renvoAmd64AsmAndRaxImm32(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x2548)
	renvoAsmEmit32(a, imm)
}

func renvoEmitRuntimeArenaPersistString(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	a := &g.asm
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoEmitPersistentAllocToPrimary(g, lenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyBytesToPersistent(g, srcOff, lenOff, destOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	return true
}

func renvoEmitRuntimeArenaPersistBytes(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	return renvoEmitRuntimeArenaPersistSlice(g, ep, idx)
}

func renvoEmitRuntimeArenaPersistSlice(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	sliceType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
	if sliceType.kind != renvoTypeSlice || !renvoEmitSliceValueRegs(g, ep, argIndex) {
		return false
	}
	a := &g.asm
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	byteLenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmMulTertiaryImm(a, renvoTypeSize(g.meta, sliceType.elem))
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, byteLenOff)
	renvoEmitPersistentAllocToPrimary(g, byteLenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyBytesToPersistent(g, srcOff, byteLenOff, destOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	renvoAsmLoadTertiaryStack(a, lenOff)
	return true
}

func renvoEmitCopyBytesToPersistent(g *renvoLinearGen, srcOff int, lenOff int, destOff int) {
	renvoNonNil(g)
	renvoEmitCopyBytes(g, srcOff, destOff, lenOff)
}

func renvoEmitPersistentAllocToPrimary(g *renvoLinearGen, sizeOff int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, sizeOff)
	renvoAsmCallLabel(a, renvoEnsureDirectionalArenaAllocHelper(g, true))
	renvoEmitArenaAllocationCheck(g)
}

func renvoEmitBuiltinNew(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprCall || e.argCount != 1 {
		return false
	}
	targetType := renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	if targetType == 0 {
		return false
	}
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, renvoTypeSize(g.meta, targetType))
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmLoadTertiaryStack(&g.asm, sizeOffset)
	renvoAsmCallLabel(&g.asm, renvoEnsureMakeZeroHelper(g))
	return true
}

func renvoEmitPersistentArenaReady(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	renvoStringHeapOffsets(g)
	readyLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	renvoAsmJnzPrimary(a, readyLabel)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	renvoAsmPushImm(a, renvoStringArenaSize(g))
	renvoAsmPopTertiary(a)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	lowReadyLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
	renvoAsmJnzPrimary(a, lowReadyLabel)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapOff)
	renvoAsmMarkLabel(a, lowReadyLabel)
	renvoAsmMarkLabel(a, readyLabel)
}

func renvoEmitArbitrarySyscall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount < 1 || e.argCount > 7 {
		return false
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		if e.argCount != 4 {
			return false
		}
		number := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		// The Darwin directory adapter uses one compiler-intrinsic selector,
		// which is lowered to libc getdirentries rather than issued as a raw
		// Darwin syscall number.
		if !number.ok || number.value != 217 {
			return false
		}
	}
	syscallNumber := -1
	if renvoFixedTarget == renvoTargetOpenBSDAmd64 ||
		renvoFixedTarget == 0 && g.c.renvoTargetOS == renvoOSOpenBSD {
		number := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		if !number.ok {
			return false
		}
		syscallNumber = number.value
	}
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
		if !renvoEmitSyscallArg(g, ep, argIndex) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
	}
	return renvoEmitSyscallFromStack(g, e.argCount, syscallNumber)
}

// renvo_runtime_CallJIT is a frontend-only escape hatch used by the bundled
// runner. It switches to an isolated native stack and calls a linked-image
// entry directly. Keeping this as one compiler intrinsic avoids exposing
// arbitrary code pointers as ordinary Go function values.
func renvoEmitJITCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 6 || g.c.renvoTargetArch == renvoArchWasm32 {
		return false
	}
	for i := e.argCount - 1; i >= 0; i-- {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
	}
	a := &g.asm
	if g.c.renvoTargetArch == renvoArchAmd64 {
		renvoAsmPopCallWord0(a)
		renvoAsmEmit8(a, 0x5e)
		renvoAsmPopSecondary(a)
		renvoAsmPopTertiary(a)
		renvoAsmEmit16(a, 0x5841)
		renvoAsmEmit16(a, 0x5941)
		renvoAsmEmitText(a, "\x53\x55\x41\x54\x41\x55\x41\x56\x41\x57")
		renvoAsmEmitText(a, "\x48\x89\xf8\x49\x89\xe2\x48\x89\xf4\x48\x83\xec\x10\x4c\x89\x54\x24\x08")
		renvoAsmEmitText(a, "\x48\x89\xd7\x48\x89\xce\x4c\x89\xc2\x4c\x89\xc9\xff\xd0")
		renvoAsmEmitText(a, "\x49\x89\xc2\x4c\x8b\x5c\x24\x08\x4c\x89\xdc")
		renvoAsmEmitText(a, "\x41\x5f\x41\x5e\x41\x5d\x41\x5c\x5d\x5b\x4c\x89\xd0")
		return true
	}
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmEmitText(a, "\x5b\x5e\x5a\x59\x58\x5f")
		renvoAsmEmitText(a, "\x89\x66\xfc\x89\xf4\x83\xec\x08\x57\x50\x51\x52\xff\xd3")
		renvoAsmEmitText(a, "\x83\xc4\x10\x8b\x54\x24\x04\x89\xd4")
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopReg(a, renvoAarch64RegRdi)
		renvoAarch64AsmPopReg(a, renvoAarch64RegRsi)
		renvoAarch64AsmPopReg(a, renvoAarch64RegRdx)
		renvoAarch64AsmPopReg(a, renvoAarch64RegRcx)
		renvoAarch64AsmPopReg(a, renvoAarch64RegR8)
		renvoAarch64AsmPopReg(a, renvoAarch64RegR9)
		renvoAarch64AsmMovRegReg(a, 9, renvoAarch64RegRdi)
		renvoAarch64AsmMovRegReg(a, 10, renvoAarch64RegRsi)
		renvoAarch64AsmEmit(a, 0x910003eb) // ADD X11, SP, #0
		renvoAarch64AsmEmit(a, 0x9100015f) // ADD SP, X10, #0
		renvoAarch64AsmEmit(a, 0xd10043ff) // SUB SP, SP, #16
		renvoAarch64AsmStoreRegMem(a, 11, 31, 8, 8)
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRdx)
		renvoAarch64AsmMovRegReg(a, 1, renvoAarch64RegRcx)
		renvoAarch64AsmMovRegReg(a, 2, renvoAarch64RegR8)
		renvoAarch64AsmMovRegReg(a, 3, renvoAarch64RegR9)
		renvoAarch64AsmEmit(a, 0xd63f0120) // BLR X9
		renvoAarch64AsmLoadRegMem(a, 11, 31, 8, 8)
		renvoAarch64AsmEmit(a, 0x9100017f) // ADD SP, X11, #0
		return true
	}
	if g.c.renvoTargetArch == renvoArchArm {
		renvoArmAsmPopReg(a, renvoArmRegRdi)
		renvoArmAsmPopReg(a, renvoArmRegRsi)
		renvoArmAsmPopReg(a, renvoArmRegRdx)
		renvoArmAsmPopReg(a, renvoArmRegRcx)
		renvoArmAsmPopReg(a, renvoArmRegR8)
		renvoArmAsmPopReg(a, renvoArmRegR9)
		renvoArmAsmMovRegReg(a, renvoArmRegTmp, renvoArmRegRdi)
		renvoArmAsmMovRegReg(a, renvoArmRegTmp2, renvoArmRegRsi)
		renvoArmAsmMovRegReg(a, renvoArmRegAddr, renvoArmRegSp)
		renvoArmAsmMovRegReg(a, renvoArmRegSp, renvoArmRegTmp2)
		renvoArmAsmAddRegImm(a, renvoArmRegSp, renvoArmRegSp, -8)
		renvoArmAsmStoreRegMem(a, renvoArmRegAddr, renvoArmRegSp, 4, 4)
		renvoArmAsmMovRegReg(a, 0, renvoArmRegRdx)
		renvoArmAsmMovRegReg(a, 1, renvoArmRegRcx)
		renvoArmAsmMovRegReg(a, 2, renvoArmRegR8)
		renvoArmAsmMovRegReg(a, 3, renvoArmRegR9)
		renvoArmAsmEmit(a, 0xe12fff39) // BLX R9
		renvoArmAsmLoadRegMem(a, renvoArmRegAddr, renvoArmRegSp, 4, 4)
		renvoArmAsmMovRegReg(a, renvoArmRegSp, renvoArmRegAddr)
		return true
	}
	return false
}

func renvoEmitSyscallArg(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	typ := renvoInferParsedExprType(g, ep, idx)
	if renvoTypeIsString(g.meta, typ) {
		return renvoEmitStringPtrExpr(g, ep, idx)
	}
	if renvoTypeIsSlice(g.meta, typ) {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		return true
	}
	return renvoEmitIntExpr(g, ep, idx)
}

func renvoEmitSyscallFromStack(g *renvoLinearGen, wordCount int, syscallNumber int) bool {
	renvoNonNil(g)
	a := &g.asm
	if renvoPreparedBackend != 0 {
		if wordCount > 7 {
			return false
		}
		registers := []RTGRegister{
			renvoRTGSyscallNumber,
			renvoRTGSyscallWord0, renvoRTGSyscallWord1, renvoRTGSyscallWord2,
			renvoRTGSyscallWord3, renvoRTGSyscallWord4, renvoRTGSyscallWord5,
		}
		for i := 0; i < wordCount; i++ {
			if !registers[i].Valid {
				return false
			}
			renvoRTGAsmPopRegister(a, registers[i])
		}
		renvoRTGDirectHostSyscall(a)
		if renvoRTGSyscallResult.Valid &&
			renvoRTGSyscallResult.Code != renvoRTGPrimary.Code {
			renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGSyscallResult)
		}
		return true
	}
	if g.c.renvoTargetArch == renvoArchAmd64 {
		renvoAsmPopPrimary(a)
		if wordCount > 1 {
			renvoAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			renvoAsmEmit8(a, 0x5e)
		}
		if wordCount > 3 {
			renvoAsmPopSecondary(a)
		}
		if wordCount > 4 {
			renvoAsmPopTertiary(a)
			renvoAsmEmit24(a, 0xca8949)
		}
		if wordCount > 5 {
			renvoAsmEmit16(a, 0x5841)
		}
		if wordCount > 6 {
			renvoAsmEmit16(a, 0x5941)
		}
		if renvoFixedTarget == renvoTargetOpenBSDAmd64 ||
			renvoFixedTarget == 0 && g.c.renvoTargetOS == renvoOSOpenBSD {
			a.syscallNumber = syscallNumber
			a.syscallNumberKnown = syscallNumber >= 0
		}
		renvoAsmSyscall(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArch386 {
		if wordCount > 7 {
			return false
		}
		renvoAsmPopPrimary(a)
		if wordCount > 1 {
			renvoAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			renvoAsmEmit8(a, 0x59)
		}
		if wordCount > 3 {
			renvoAsmPopSecondary(a)
		}
		if wordCount > 4 {
			renvoAsmEmit8(a, 0x5e)
		}
		if wordCount > 5 {
			renvoAsmEmit8(a, 0x5f)
		}
		if wordCount > 6 {
			// Linux/i386 carries the sixth syscall argument in EBP. Exchange
			// it with the saved frame pointer at the top of the expression
			// stack, then restore EBP after int 0x80.
			renvoAsmEmitText(a, "\x87\x2c\x24")
			renvoAsmSyscall(a)
			renvoAsmEmit8(a, 0x5d)
			return true
		}
		renvoAsmSyscall(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		if targetIsDarwin(g.c.renvoTargetOS) {
			if wordCount != 4 {
				return false
			}
			renvoAarch64AsmPopReg(a, 9)
			renvoAarch64AsmPopReg(a, 0)
			renvoAarch64AsmPopReg(a, 1)
			renvoAarch64AsmPopReg(a, 2)
			baseOff := a.bssSize
			a.bssSize += 8
			renvoAarch64AsmMovRegAbs(a, 3, baseOff, renvoAbsBssReloc)
			renvoDarwinArm64CallImport(a, renvoDarwinImportGetdirentries)
			return true
		}
		renvoAarch64AsmPopReg(a, renvoAarch64RegSys)
		if wordCount > 1 {
			renvoAarch64AsmPopReg(a, 0)
		}
		if wordCount > 2 {
			renvoAarch64AsmPopReg(a, 1)
		}
		if wordCount > 3 {
			renvoAarch64AsmPopReg(a, 2)
		}
		if wordCount > 4 {
			renvoAarch64AsmPopReg(a, 3)
		}
		if wordCount > 5 {
			renvoAarch64AsmPopReg(a, 4)
		}
		if wordCount > 6 {
			renvoAarch64AsmPopReg(a, 5)
		}
		renvoAarch64AsmEmit(a, 0xd4000001)
		return true
	}
	if g.c.renvoTargetArch == renvoArchArm {
		renvoArmAsmPopReg(a, renvoArmRegSys)
		if wordCount > 1 {
			renvoArmAsmPopReg(a, 0)
		}
		if wordCount > 2 {
			renvoArmAsmPopReg(a, 1)
		}
		if wordCount > 3 {
			renvoArmAsmPopReg(a, 2)
		}
		if wordCount > 4 {
			renvoArmAsmPopReg(a, 3)
		}
		if wordCount > 5 {
			renvoArmAsmPopReg(a, 4)
		}
		if wordCount > 6 {
			renvoArmAsmPopReg(a, 5)
		}
		renvoArmAsmEmit(a, 0xef000000)
		return true
	}
	if g.c.renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRax)
		if wordCount > 1 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdi)
		}
		if wordCount > 2 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRsi)
		}
		if wordCount > 3 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdx)
		}
		if wordCount > 4 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
		}
		if wordCount > 5 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR8)
		}
		if wordCount > 6 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR9)
		}
		renvoAsmSyscall(a)
		return true
	}
	return false
}
