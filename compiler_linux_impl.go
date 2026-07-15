package main

func rtgReadAll(fd int, out []byte) []byte {
	buf := make([]byte, 0, 1024)
	buf = buf[:cap(buf)]
	for {
		base := len(out)
		if base < cap(out) {
			expanded := out[:cap(out)]
			n := read(fd, expanded[base:], -1)
			if n <= 0 {
				return out
			}
			out = expanded[:base+n]
			continue
		}
		n := read(fd, buf, -1)
		if n <= 0 {
			return out
		}
		out = append(out, buf[:n]...)
	}
}

func compileLinuxTarget(input []int, output int, target int) int {
	return compileTarget(input, output, target)
}

func rtgLinuxSysWriteSeq() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 64
	}
	if rtgTargetArch == rtgArchArm {
		return 4
	}
	if rtgTargetArch == rtgArch386 {
		return 4
	}
	return 1
}

func rtgLinuxSysReadSeq() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 63
	}
	if rtgTargetArch == rtgArchArm {
		return 3
	}
	if rtgTargetArch == rtgArch386 {
		return 3
	}
	return 0
}

func rtgLinuxSysReadAt() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 67
	}
	if rtgTargetArch == rtgArchArm {
		return 180
	}
	if rtgTargetArch == rtgArch386 {
		return 180
	}
	return 17
}

func rtgLinuxSysWriteAt() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 68
	}
	if rtgTargetArch == rtgArchArm {
		return 181
	}
	if rtgTargetArch == rtgArch386 {
		return 181
	}
	return 18
}

func rtgLinuxSysOpen() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 56
	}
	if rtgTargetArch == rtgArchArm {
		return 5
	}
	if rtgTargetArch == rtgArch386 {
		return 5
	}
	return 2
}

func rtgLinuxSysClose() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 57
	}
	if rtgTargetArch == rtgArchArm {
		return 6
	}
	if rtgTargetArch == rtgArch386 {
		return 6
	}
	return 3
}

func rtgLinuxSysFchmod() int {
	if rtgTargetArch == rtgArchAarch64 {
		return 52
	}
	if rtgTargetArch == rtgArchArm {
		return 94
	}
	if rtgTargetArch == rtgArch386 {
		return 94
	}
	return 91
}

func rtgAsmPrepareReadWriteBuf(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRsiRax(a)
		rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRcx)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPrepareReadWriteBuf(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPrepareReadWriteBuf(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmPrepareReadWriteBuf(a)
		return
	}
	rtgAmd64AsmPrepareReadWriteBuf(a)
}

func rtgAsmMoveOffsetArg(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR10, rtgWasm32RegRax)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMoveOffsetArg(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMoveOffsetArg(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMoveOffsetArg(a)
		return
	}
	rtgAmd64AsmMoveOffsetArg(a)
}

func rtgEmitLinearPrintStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	p := g.prog
	if stmt.exprStart < 0 || stmt.exprStart >= rtgTokCount(p) || !rtgBytesEqualText(p.src, int(rtgTokStart(p, stmt.exprStart)), int(rtgTokEnd(p, stmt.exprStart)), "print") {
		return false
	}
	var ep rtgExprParse
	rtgParseExpressionInto(&ep, p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, &ep, root.left, "print") {
		return false
	}
	argIndex := ep.args[root.firstArg]
	argType := rtgResolveType(g.meta, rtgInferParsedExprType(g, &ep, argIndex))
	if argType.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, &ep, argIndex) {
			return false
		}
	} else if rtgTypeKindIsScalarInt(argType.kind) {
		if !rtgEmitIntExpr(g, &ep, argIndex) {
			return false
		}
		rtgAsmNormalizePrimaryForKind(&g.asm, argType.kind)
		rtgAsmCallLabel(&g.asm, rtgEnsurePrintIntHelper(g))
	} else {
		return false
	}
	return rtgEmitPrintValueRegs(g)
}

func rtgEmitPrintValueRegs(g *rtgLinearGen) bool {
	return rtgEmitWriteValueRegs(g, 1)
}

func rtgEmitWriteValueRegs(g *rtgLinearGen, fd int) bool {
	a := &g.asm
	if rtgTargetIsWindows() {
		if rtgTargetArch == rtgArch386 {
			label := rtgWin386EmitReadWriteHelper(g, true)
			rtgAsmEmit16(a, 0xc689)
			rtgAsmPrimaryImm(a, -1)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmPrimaryImm(a, fd)
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmCallLabel(a, label)
			return true
		}
		label := rtgWinAmd64EmitReadWriteHelper(g, true)
		rtgAsmCopyPrimaryToCallWord1(a)
		rtgAsmPrimaryImm(a, fd)
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmCallLabel(a, label)
		return true
	}
	if rtgTargetIsDarwin() {
		rtgAarch64AsmMovRegReg(a, 2, rtgAarch64RegRdx)
		rtgAarch64AsmMovRegReg(a, 1, rtgAarch64RegRax)
		rtgAarch64AsmMovRegImm(a, 0, fd)
		rtgDarwinArm64CallImport(a, rtgDarwinImportWrite)
		return true
	}
	rtgAsmPushImm(a, fd)
	rtgAsmPopCallWord0(a)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmPrimaryImm(a, rtgLinuxSysWriteSeq())
	rtgAsmSyscall(a)
	return true
}

func rtgEmitBuiltinReadWrite(g *rtgLinearGen, ep *rtgExprParse, idx int, seqSyscall int, offSyscall int) bool {
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	fdStart := ep.exprs[idx].tok + 1
	fdEnd := rtgFindExprBoundary(p, fdStart, ep.end)
	var fdEp rtgExprParse
	rtgParseExpressionInto(&fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	fdIndex := len(fdEp.exprs) - 1
	if !rtgEmitIntExpr(g, &fdEp, fdIndex) {
		return false
	}
	rtgAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := rtgEvalConstExpr(g, ep, offIndex)
	offsetRead := true
	if offConst.ok && offConst.value < 0 {
		offsetRead = false
	}
	if offsetRead {
		if offConst.ok {
			rtgAsmPrimaryImm(a, offConst.value)
		} else {
			if !rtgEmitIntExpr(g, ep, offIndex) {
				return false
			}
		}
		rtgAsmPushPrimary(a)
	}
	if !rtgEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	rtgAsmPrepareReadWriteBuf(a)
	if offsetRead {
		rtgAsmPopPrimary(a)
		rtgAsmMoveOffsetArg(a)
	}
	rtgAsmPopCallWord0(a)
	if offsetRead {
		rtgAsmPrimaryImm(a, offSyscall)
	} else {
		rtgAsmPrimaryImm(a, seqSyscall)
	}
	if rtgTargetIsDarwin() {
		importID := seqSyscall
		if offsetRead {
			importID = offSyscall
		}
		argCount := 3
		if offsetRead {
			argCount = 4
		}
		rtgDarwinArm64CallVirtualArgs(a, importID, argCount)
		return true
	}
	rtgAsmSyscall(a)
	return true
}

func rtgAsmJgeLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 0x8d0f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgAsmJlLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 0x8c0f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgWinAmd64CallImport(a *rtgAsm, importID int, shadow int) {
	rtgAsmEmit4(a, 0x48, 0x83, 0xec, shadow)
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, shadow)
}

func rtgWin386CallImport(a *rtgAsm, importID int) {
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWinAmd64LoadImportPtrRax(a *rtgAsm, importID int) {
	rtgAsmEmit24(a, 0x058b48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWinAmd64LoadImportPtrR9(a *rtgAsm, importID int) {
	rtgAsmEmit24(a, 0x0d8b4c)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWinAmd64LoadImportPtrR10(a *rtgAsm, importID int) {
	rtgAsmEmit24(a, 0x158b4c)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWin386LoadImportPtrRax(a *rtgAsm, importID int) {
	rtgAsmEmit8(a, 0xa1)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWin386LoadImportPtrRsi(a *rtgAsm, importID int) {
	rtgAsmEmit16(a, 0x358b)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
}

func rtgWin386StoreEcxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit16(a, 0x0d89)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386MovEbxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xbb)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386MovEcxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xb9)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386MovEdiBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xbf)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386PushBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0x68)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386LoadEsiBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit16(a, 0x358b)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWin386LoadEaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xa1)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgWinAmd64EmitReadWriteHelper(g *rtgLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = rtgAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = rtgAsmNewLabel(a)
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	label := g.winReadLabel
	importID := rtgWinImportReadFile
	if isWrite {
		label = g.winWriteLabel
		importID = rtgWinImportWriteFile
	}
	afterLabel := rtgAsmNewLabel(a)
	seqLabel := rtgAsmNewLabel(a)
	failLabel := rtgAsmNewLabel(a)
	seqFailLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, label)
	rtgAsmEmit8(a, 0x57)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x52)
	rtgAsmEmit8(a, 0x51)
	if isWrite {
		stdOutLabel := rtgAsmNewLabel(a)
		stdErrLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit5(a, 0x48, 0x83, 0x7c, 0x24, 24)
		rtgAsmEmit8(a, 1)
		rtgAsmJzLabel(a, stdOutLabel)
		rtgAsmEmit5(a, 0x48, 0x83, 0x7c, 0x24, 24)
		rtgAsmEmit8(a, 2)
		rtgAsmJzLabel(a, stdErrLabel)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdOutLabel)
		rtgWinAmd64SetStdHandle(a, -11)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdErrLabel)
		rtgWinAmd64SetStdHandle(a, -12)
		rtgAsmMarkLabel(a, afterStdLabel)
	} else {
		stdInLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit5(a, 0x48, 0x83, 0x7c, 0x24, 24)
		rtgAsmEmit8(a, 0)
		rtgAsmJzLabel(a, stdInLabel)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdInLabel)
		rtgWinAmd64SetStdHandle(a, -10)
		rtgAsmMarkLabel(a, afterStdLabel)
	}
	rtgAsmEmit5(a, 0x48, 0x83, 0x3c, 0x24, 0)
	rtgAsmJlLabel(a, seqLabel)

	rtgWinAmd64LoadSavedHandle(a)
	rtgAsmEmit16(a, 0xd231)
	rtgAsmEmit24(a, 0xc03145)
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 1)
	rtgWinAmd64CallImport(a, rtgWinImportSetFilePointer, 40)
	rtgAsmStorePrimaryBss(a, posOff)

	rtgWinAmd64LoadSavedHandle(a)
	rtgAsmEmit32(a, 0x24148b48)
	rtgAsmEmit24(a, 0xc03145)
	rtgAsmEmit24(a, 0xc93145)
	rtgWinAmd64CallImport(a, rtgWinImportSetFilePointer, 40)
	rtgWinAmd64EmitKernelReadWriteCall(a, importID, countOff)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, seqLabel)
	rtgWinAmd64EmitKernelReadWriteCall(a, importID, countOff)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, seqFailLabel)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 32)
	rtgAsmRet(a)

	rtgAsmMarkLabel(a, seqFailLabel)
	rtgAsmPrimaryImm(a, -1)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 32)
	rtgAsmRet(a)

	rtgAsmMarkLabel(a, failLabel)
	rtgAsmPrimaryImm(a, -1)

	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmStorePrimaryBss(a, countOff)
	rtgWinAmd64LoadSavedHandle(a)
	rtgAsmLoadPrimaryBss(a, posOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmEmit24(a, 0xc03145)
	rtgAsmEmit24(a, 0xc93145)
	rtgWinAmd64CallImport(a, rtgWinImportSetFilePointer, 40)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 32)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return label
}

func rtgWinAmd64SetStdHandle(a *rtgAsm, stdHandle int) {
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, stdHandle)
	rtgWinAmd64CallImport(a, rtgWinImportGetStdHandle, 40)
	rtgAsmEmit5(a, 0x48, 0x89, 0x44, 0x24, 24)
}

func rtgWinAmd64LoadSavedHandle(a *rtgAsm) {
	rtgAsmEmit5(a, 0x48, 0x8b, 0x4c, 0x24, 24)
}

func rtgWinAmd64EmitKernelReadWriteCall(a *rtgAsm, importID int, countOff int) {
	rtgWinAmd64LoadSavedHandle(a)
	rtgAsmEmit5(a, 0x48, 0x8b, 0x54, 0x24, 16)
	rtgAsmEmit5(a, 0x4c, 0x8b, 0x44, 0x24, 8)
	rtgAsmPrimaryBssAddr(a, countOff)
	rtgAsmCopyPrimaryToCallWord5(a)
	rtgAsmEmit4(a, 0x48, 0x83, 0xec, 40)
	rtgAsmEmit5(a, 0x48, 0xc7, 0x44, 0x24, 32)
	rtgAsmEmit32(a, 0)
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 40)
}

func rtgWin386EmitReadWriteHelper(g *rtgLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = rtgAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = rtgAsmNewLabel(a)
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	label := g.winReadLabel
	importID := rtgWinImportReadFile
	if isWrite {
		label = g.winWriteLabel
		importID = rtgWinImportWriteFile
	}
	afterLabel := rtgAsmNewLabel(a)
	seqLabel := rtgAsmNewLabel(a)
	failLabel := rtgAsmNewLabel(a)
	seqFailLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, label)
	if isWrite {
		stdOutLabel := rtgAsmNewLabel(a)
		stdErrLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit3(a, 0x83, 0xfb, 1)
		rtgAsmJzLabel(a, stdOutLabel)
		rtgAsmEmit3(a, 0x83, 0xfb, 2)
		rtgAsmJzLabel(a, stdErrLabel)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdOutLabel)
		rtgWin386SetStdHandle(a, -11)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdErrLabel)
		rtgWin386SetStdHandle(a, -12)
		rtgAsmMarkLabel(a, afterStdLabel)
	} else {
		stdInLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit3(a, 0x83, 0xfb, 0)
		rtgAsmJzLabel(a, stdInLabel)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdInLabel)
		rtgWin386SetStdHandle(a, -10)
		rtgAsmMarkLabel(a, afterStdLabel)
	}
	rtgAsmEmit3(a, 0x83, 0xf9, 0)
	rtgAsmJlLabel(a, seqLabel)

	rtgAsmEmit8(a, 0x51)
	rtgAsmPushSecondary(a)
	rtgAsmPushImm(a, 1)
	rtgAsmPushImm(a, 0)
	rtgAsmPushImm(a, 0)
	rtgAsmEmit8(a, 0x53)
	rtgWin386CallImport(a, rtgWinImportSetFilePointer)
	rtgAsmStorePrimaryBss(a, posOff)
	rtgAsmPopSecondary(a)
	rtgAsmPopTertiary(a)

	rtgAsmEmit8(a, 0x51)
	rtgAsmPushSecondary(a)
	rtgAsmPushImm(a, 0)
	rtgAsmPushImm(a, 0)
	rtgAsmPushTertiary(a)
	rtgAsmEmit8(a, 0x53)
	rtgWin386CallImport(a, rtgWinImportSetFilePointer)
	rtgAsmPopSecondary(a)
	rtgAsmPopTertiary(a)
	rtgWin386EmitKernelReadWriteCall(a, importID, countOff)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, seqLabel)
	rtgWin386EmitKernelReadWriteCall(a, importID, countOff)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, seqFailLabel)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmRet(a)

	rtgAsmMarkLabel(a, seqFailLabel)
	rtgAsmPrimaryImm(a, -1)
	rtgAsmRet(a)

	rtgAsmMarkLabel(a, failLabel)
	rtgAsmPrimaryImm(a, -1)

	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmStorePrimaryBss(a, countOff)
	rtgAsmPushImm(a, 0)
	rtgAsmPushImm(a, 0)
	rtgWin386LoadEaxBss(a, posOff)
	rtgAsmPushPrimary(a)
	rtgAsmEmit8(a, 0x53)
	rtgWin386CallImport(a, rtgWinImportSetFilePointer)
	rtgAsmLoadPrimaryBss(a, countOff)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return label
}

func rtgWin386SetStdHandle(a *rtgAsm, stdHandle int) {
	rtgAsmPushImm(a, stdHandle)
	rtgWin386CallImport(a, rtgWinImportGetStdHandle)
	rtgAsmCopyPrimaryToCallWord0(a)
}

func rtgWin386EmitKernelReadWriteCall(a *rtgAsm, importID int, countOff int) {
	rtgAsmPushImm(a, 0)
	rtgWin386PushBssAddr(a, countOff)
	rtgAsmPushSecondary(a)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x53)
	rtgWin386CallImport(a, importID)
}

func rtgEmitWindowsReadWrite(g *rtgLinearGen, ep *rtgExprParse, idx int, isWrite bool) bool {
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	fdStart := ep.exprs[idx].tok + 1
	fdEnd := rtgFindExprBoundary(p, fdStart, ep.end)
	var fdEp rtgExprParse
	rtgParseExpressionInto(&fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	if !rtgEmitIntExpr(g, &fdEp, len(fdEp.exprs)-1) {
		return false
	}
	rtgAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := rtgEvalConstExpr(g, ep, offIndex)
	if offConst.ok && offConst.value < 0 {
		rtgAsmPrimaryImm(a, -1)
	} else if offConst.ok {
		rtgAsmPrimaryImm(a, offConst.value)
	} else {
		if !rtgEmitIntExpr(g, ep, offIndex) {
			return false
		}
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	if rtgTargetArch == rtgArch386 {
		label := rtgWin386EmitReadWriteHelper(g, isWrite)
		rtgAsmEmit16(a, 0xc689)
		rtgAsmEmit16(a, 0xca89)
		rtgAsmPopTertiary(a)
		rtgAsmPopCallWord0(a)
		rtgAsmCallLabel(a, label)
		return true
	}
	label := rtgWinAmd64EmitReadWriteHelper(g, isWrite)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmEmit24(a, 0xca8948)
	rtgAsmPopTertiary(a)
	rtgAsmPopCallWord0(a)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgEmitWindowsOpen(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := ep.exprs[idx]
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
	if rtgTargetArch == rtgArch386 {
		createFileImport := rtgWinImportCreateFileA
		rtgAsmEmit16(a, 0xc689)
		rtgAsmPopPrimary(a)
		rtgWin386TranslateCreateFileFlags(a)
		rtgAsmPushImm(a, 0)
		rtgAsmPushImm(a, 0x80)
		rtgAsmPushTertiary(a)
		rtgAsmPushImm(a, 0)
		rtgAsmPushImm(a, 3)
		rtgAsmPushSecondary(a)
		rtgAsmEmit8(a, 0x56)
		rtgWin386CallImport(a, createFileImport)
		return true
	}
	createFileImport := rtgWinImportCreateFileA
	rtgAsmPushPrimary(a)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPopTertiary(a)
	rtgAsmPopPrimary(a)
	rtgWinAmd64TranslateCreateFileFlags(a)
	rtgAsmEmit4(a, 0x48, 0x83, 0xec, 56)
	rtgAsmEmit5(a, 0x44, 0x89, 0x54, 0x24, 32)
	rtgAsmEmit4(a, 0xc7, 0x44, 0x24, 40)
	rtgAsmEmit32(a, 0x80)
	rtgAsmEmit5(a, 0x48, 0xc7, 0x44, 0x24, 48)
	rtgAsmEmit32(a, 0)
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, createFileImport)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 56)
	return true
}

func rtgWinAmd64TranslateCreateFileFlags(a *rtgAsm) {
	notRDWRLabel := rtgAsmNewLabel(a)
	accessDoneLabel := rtgAsmNewLabel(a)
	noCreateLabel := rtgAsmNewLabel(a)
	createDoneLabel := rtgAsmNewLabel(a)

	rtgAsmSecondaryImm(a, -2147483648)
	rtgAsmEmit2(a, 0xa8, 2)
	rtgAsmJzLabel(a, notRDWRLabel)
	rtgAsmSecondaryImm(a, -1073741824)
	rtgAsmJmpLabel(a, accessDoneLabel)
	rtgAsmMarkLabel(a, notRDWRLabel)
	rtgAsmEmit2(a, 0xa8, 1)
	rtgAsmJzLabel(a, accessDoneLabel)
	rtgAsmSecondaryImm(a, 0x40000000)
	rtgAsmMarkLabel(a, accessDoneLabel)

	rtgWinAmd64MovR10Imm(a, 3)
	rtgAsmEmit2(a, 0xa8, 64)
	rtgAsmJzLabel(a, noCreateLabel)
	rtgWinAmd64MovR10Imm(a, 4)
	rtgAsmEmit8(a, 0xa9)
	rtgAsmEmit32(a, 512)
	rtgAsmJzLabel(a, createDoneLabel)
	rtgWinAmd64MovR10Imm(a, 2)
	rtgAsmJmpLabel(a, createDoneLabel)
	rtgAsmMarkLabel(a, noCreateLabel)
	rtgAsmEmit8(a, 0xa9)
	rtgAsmEmit32(a, 512)
	rtgAsmJzLabel(a, createDoneLabel)
	rtgWinAmd64MovR10Imm(a, 5)
	rtgAsmMarkLabel(a, createDoneLabel)
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit32(a, 3)
	rtgAsmEmit24(a, 0xc93145)
}

func rtgWinAmd64MovR10Imm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, imm)
}

func rtgWin386TranslateCreateFileFlags(a *rtgAsm) {
	notRDWRLabel := rtgAsmNewLabel(a)
	accessDoneLabel := rtgAsmNewLabel(a)
	noCreateLabel := rtgAsmNewLabel(a)
	createDoneLabel := rtgAsmNewLabel(a)

	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, -2147483648)
	rtgAsmEmit2(a, 0xa8, 2)
	rtgAsmJzLabel(a, notRDWRLabel)
	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, -1073741824)
	rtgAsmJmpLabel(a, accessDoneLabel)
	rtgAsmMarkLabel(a, notRDWRLabel)
	rtgAsmEmit2(a, 0xa8, 1)
	rtgAsmJzLabel(a, accessDoneLabel)
	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, 0x40000000)
	rtgAsmMarkLabel(a, accessDoneLabel)

	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 3)
	rtgAsmEmit2(a, 0xa8, 64)
	rtgAsmJzLabel(a, noCreateLabel)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 4)
	rtgAsmEmit8(a, 0xa9)
	rtgAsmEmit32(a, 512)
	rtgAsmJzLabel(a, createDoneLabel)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 2)
	rtgAsmJmpLabel(a, createDoneLabel)
	rtgAsmMarkLabel(a, noCreateLabel)
	rtgAsmEmit8(a, 0xa9)
	rtgAsmEmit32(a, 512)
	rtgAsmJzLabel(a, createDoneLabel)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 5)
	rtgAsmMarkLabel(a, createDoneLabel)
}

func rtgEmitWindowsClose(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	failLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	if rtgTargetArch == rtgArch386 {
		rtgAsmPushPrimary(a)
		rtgWin386CallImport(a, rtgWinImportCloseHandle)
		rtgAsmEmit3(a, 0x83, 0xf8, 0)
		rtgAsmJzLabel(a, failLabel)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmJmpLabel(a, doneLabel)
		rtgAsmMarkLabel(a, failLabel)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	rtgAsmCopyPrimaryToTertiary(a)
	rtgWinAmd64CallImport(a, rtgWinImportCloseHandle, 40)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmJmpLabel(a, doneLabel)
	rtgAsmMarkLabel(a, failLabel)
	rtgAsmPrimaryImm(a, -1)
	rtgAsmMarkLabel(a, doneLabel)
	return true
}

func rtgEmitWindowsChmod(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := ep.exprs[idx]
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
	if rtgTargetArch == rtgArch386 {
		rtgAsmPopPrimary(a)
		rtgAsmPushImm(a, 1)
		rtgAsmPushImm(a, 0)
		rtgAsmPushImm(a, 0)
		rtgAsmPushPrimary(a)
		rtgWin386CallImport(a, rtgWinImportSetFilePointer)
		rtgAsmEmit3(a, 0x83, 0xf8, -1)
		failLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmJzLabel(a, failLabel)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmJmpLabel(a, doneLabel)
		rtgAsmMarkLabel(a, failLabel)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	rtgAsmPopTertiary(a)
	rtgAsmEmit16(a, 0xd231)
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, 1)
	rtgAsmEmit24(a, 0xc03145)
	rtgWinAmd64CallImport(a, rtgWinImportSetFilePointer, 40)
	rtgAsmEmit3(a, 0x83, 0xf8, -1)
	failLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmJmpLabel(a, doneLabel)
	rtgAsmMarkLabel(a, failLabel)
	rtgAsmPrimaryImm(a, -1)
	rtgAsmMarkLabel(a, doneLabel)
	return true
}

func rtgEvalBuiltinConst(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	p := g.prog
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "iota") {
		if g.constEvalIotaValid != 0 {
			return rtgConstResultOk(g.constEvalIota)
		}
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "nil") {
		return rtgConstResultOk(0)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_RDONLY") {
		return rtgConstResultOk(0)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_WRONLY") {
		return rtgConstResultOk(1)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_RDWR") {
		return rtgConstResultOk(2)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_CREATE") {
		if rtgTargetIsDarwin() {
			return rtgConstResultOk(512)
		}
		return rtgConstResultOk(64)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_TRUNC") {
		if rtgTargetIsDarwin() {
			return rtgConstResultOk(1024)
		}
		return rtgConstResultOk(512)
	}
	var r rtgConstResult
	return r
}
