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
	if stmt.exprStart < 0 || stmt.exprStart >= rtgTokCount(p) {
		return false
	}
	ep := rtgNewExprParse()
	rtgParseExpressionInto(ep, p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall {
		return false
	}
	builtinPrintln := rtgExprIdentCode(p, ep, root.left) == rtgIdentPrintln
	println := builtinPrintln || rtgExprIsIdentText(p, ep, root.left, "Println")
	if !println && !rtgExprIsIdentText(p, ep, root.left, "print") {
		return false
	}
	callee := &ep.exprs[root.left]
	if rtgFuncInfoFromCall(g, ep, root.left) >= 0 || rtgFindLocalIndex(g, callee.nameStart, callee.nameEnd) >= 0 {
		return false
	}
	fd := 1
	if builtinPrintln {
		fd = 2
	}
	for i := 0; i < root.argCount; i++ {
		if println && i > 0 && !rtgEmitPrintStaticByte(g, ' ', fd) {
			return false
		}
		argIndex := ep.args[root.firstArg+i]
		argType := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, argIndex))
		if argType.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, argIndex) {
				return false
			}
		} else if rtgTypeKindIsScalarInt(argType.kind) {
			if !rtgEmitIntExpr(g, ep, argIndex) {
				return false
			}
			rtgAsmNormalizePrimaryForKind(&g.asm, argType.kind)
			rtgAsmCallLabel(&g.asm, rtgEnsurePrintIntHelper(g))
		} else {
			return false
		}
		if !rtgEmitWriteValueRegs(g, fd) {
			return false
		}
	}
	return !println || rtgEmitPrintStaticByte(g, '\n', fd)
}

func rtgEmitPrintStaticByte(g *rtgLinearGen, value byte, fd int) bool {
	offset := len(g.asm.data)
	g.asm.data = append(g.asm.data, value)
	rtgAsmPrimaryDataAddr(&g.asm, offset)
	rtgAsmSecondaryImm(&g.asm, 1)
	return rtgEmitWriteValueRegs(g, fd)
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
		if rtgTargetArch == rtgArchAarch64 {
			label := rtgWinArm64EmitReadWriteHelper(g, true)
			rtgAsmCopyPrimaryToCallWord1(a)
			rtgAsmPrimaryImm(a, fd)
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPrimaryImm(a, -1)
			rtgAsmCopyPrimaryToTertiary(a)
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
	fdEp := rtgNewExprParse()
	rtgParseExpressionInto(fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	fdIndex := len(fdEp.exprs) - 1
	if !rtgEmitIntExpr(g, fdEp, fdIndex) {
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
	// The template is the relaxed form of the instruction sequence previously
	// assembled one operation at a time. Explicit relocations record the BSS and
	// import operands that still vary between compiler invocations and self-host.
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
	} else if g.winReadEmitted {
		return g.winReadLabel
	}
	label := rtgAsmNewLabel(a)
	if isWrite {
		g.winWriteEmitted = true
		g.winWriteLabel = label
	} else {
		g.winReadEmitted = true
		g.winReadLabel = label
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	importID := rtgWinImportReadFile
	prefix := "\xe9\x32\x01\x00\x00\x57\x56\x52\x51\x48\x83\x7c\x24\x18\x00\x74\x02\xeb\x18\xb9\xf6"
	if isWrite {
		importID = rtgWinImportWriteFile
		prefix = "\xe9\x54\x01\x00\x00\x57\x56\x52\x51\x48\x83\x7c\x24\x18\x01\x74\x0a\x48\x83\x7c\x24\x18\x02\x74\x1c\xeb\x32\xb9\xf5\xff\xff\xff\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x44\x24\x18\xeb\x18\xb9\xf4"
	}
	base := len(a.code)
	rtgAsmEmitText(a, prefix)
	a.labelPos[label] = base + 5
	a.labelSet[label] = true
	a.lastPrimaryStoreEnd = -1
	commonBase := len(a.code)
	rtgAsmEmitText(a, "\xff\xff\xff\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x44\x24\x18\x48\x83\x3c\x24\x00\x0f\x8c\x80\x00\x00\x00\x48\x8b\x4c\x24\x18\x31\xd2\x45\x31\xc0\x41\xb9\x01\x00\x00\x00\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x05\x00\x00\x00\x00\x48\x8b\x4c\x24\x18\x48\x8b\x14\x24\x45\x31\xc0\x45\x31\xc9\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x8b\x4c\x24\x18\x48\x8b\x54\x24\x10\x4c\x8b\x44\x24\x08\x48\x8d\x05\x00\x00\x00\x00\x49\x89\xc1\x48\x83\xec\x28\x48\xc7\x44\x24\x20\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x83\xf8\x00\x74\x52\x48\x8b\x05\x00\x00\x00\x00\xeb\x4c\x48\x8b\x4c\x24\x18\x48\x8b\x54\x24\x10\x4c\x8b\x44\x24\x08\x48\x8d\x05\x00\x00\x00\x00\x49\x89\xc1\x48\x83\xec\x28\x48\xc7\x44\x24\x20\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x83\xf8\x00\x74\x0c\x48\x8b\x05\x00\x00\x00\x00\x48\x83\xc4\x20\xc3\x6a\xff\x58\x48\x83\xc4\x20\xc3\x6a\xff\x58\x48\x89\x05\x00\x00\x00\x00\x48\x8b\x4c\x24\x18\x48\x8b\x05\x00\x00\x00\x00\x50\x5a\x45\x31\xc0\x45\x31\xc9\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x8b\x05\x00\x00\x00\x00\x48\x83\xc4\x20\xc3")
	if isWrite {
		rtgAsmAddWinImportReloc(a, base+38, rtgWinImportGetStdHandle)
	}
	rtgAsmAddWinImportReloc(a, commonBase+9, rtgWinImportGetStdHandle)
	rtgAsmAddWinImportReloc(a, commonBase+55, rtgWinImportSetFilePointer)
	rtgAsmAddAbsReloc(a, commonBase+66, posOff, rtgAbsBssReloc)
	rtgAsmAddWinImportReloc(a, commonBase+91, rtgWinImportSetFilePointer)
	rtgAsmAddAbsReloc(a, commonBase+117, countOff, rtgAbsBssReloc)
	rtgAsmAddWinImportReloc(a, commonBase+139, importID)
	rtgAsmAddAbsReloc(a, commonBase+155, countOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, commonBase+179, countOff, rtgAbsBssReloc)
	rtgAsmAddWinImportReloc(a, commonBase+201, importID)
	rtgAsmAddAbsReloc(a, commonBase+217, countOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, commonBase+240, countOff, rtgAbsBssReloc)
	rtgAsmAddAbsReloc(a, commonBase+252, posOff, rtgAbsBssReloc)
	rtgAsmAddWinImportReloc(a, commonBase+270, rtgWinImportSetFilePointer)
	rtgAsmAddAbsReloc(a, commonBase+281, countOff, rtgAbsBssReloc)
	return label
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
	rtgAsmJmpMarkLabel(a, afterLabel, label)
	if isWrite {
		stdOutLabel := rtgAsmNewLabel(a)
		stdErrLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit3(a, 0x83, 0xfb, 1)
		rtgAsmJzLabel(a, stdOutLabel)
		rtgAsmEmit3(a, 0x83, 0xfb, 2)
		rtgAsmJzLabel(a, stdErrLabel)
		rtgAsmJmpMarkLabel(a, afterStdLabel, stdOutLabel)
		rtgWin386SetStdHandle(a, -11)
		rtgAsmJmpMarkLabel(a, afterStdLabel, stdErrLabel)
		rtgWin386SetStdHandle(a, -12)
		rtgAsmMarkLabel(a, afterStdLabel)
	} else {
		stdInLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit3(a, 0x83, 0xfb, 0)
		rtgAsmJzLabel(a, stdInLabel)
		rtgAsmJmpMarkLabel(a, afterStdLabel, stdInLabel)
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
	fdEp := rtgNewExprParse()
	rtgParseExpressionInto(fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	if !rtgEmitIntExpr(g, fdEp, len(fdEp.exprs)-1) {
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
	if rtgTargetArch == rtgArchAarch64 {
		label := rtgWinArm64EmitReadWriteHelper(g, isWrite)
		rtgAsmCopyPrimaryToCallWord1(a)
		rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, rtgAarch64RegRcx)
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
	if rtgTargetArch == rtgArchAarch64 {
		createFileImport := rtgWinImportCreateFileA
		rtgAarch64AsmMovRegReg(a, 9, 0)
		rtgAsmPopPrimary(a)
		rtgWinArm64TranslateCreateFileFlags(a)
		rtgAarch64AsmMovRegReg(a, 0, 9)
		rtgWinArm64CallImport(a, createFileImport)
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
	rtgAsmJmpMarkLabel(a, accessDoneLabel, notRDWRLabel)
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
	rtgAsmJmpMarkLabel(a, createDoneLabel, noCreateLabel)
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
	rtgAsmJmpMarkLabel(a, accessDoneLabel, notRDWRLabel)
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
	rtgAsmJmpMarkLabel(a, createDoneLabel, noCreateLabel)
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
		rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgWinArm64CallImport(a, rtgWinImportCloseHandle)
		rtgAsmCmpPrimaryImm8(a, 0)
		rtgAsmJzLabel(a, failLabel)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	rtgAsmCopyPrimaryToTertiary(a)
	rtgWinAmd64CallImport(a, rtgWinImportCloseHandle, 40)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
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
		rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
		rtgAsmPrimaryImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAsmPopPrimary(a)
		rtgAarch64AsmMovRegImm(a, 1, 0)
		rtgAarch64AsmMovRegImm(a, 2, 0)
		rtgAarch64AsmMovRegImm(a, 3, 1)
		rtgWinArm64CallImport(a, rtgWinImportSetFilePointer)
		rtgAarch64AsmCmpRegImm(a, 0, -1)
		failLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmJzLabel(a, failLabel)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
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
	rtgAsmJmpMarkLabel(a, doneLabel, failLabel)
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
