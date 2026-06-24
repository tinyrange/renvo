package main

func rtgReadAll(fd int, out []byte) []byte {
	buf := make([]byte, 1024)
	for {
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

func compileTarget(input []int, output int, target int) int {
	if target == rtgTargetWindowsAmd64 {
		return compileWindowsAmd64(input, output)
	}
	if target == rtgTargetWindows386 {
		return compileWindows386(input, output)
	}
	if target == rtgTargetWasiWasm32 {
		return compileWasiWasm32(input, output)
	}
	if target == rtgTargetLinux386 {
		return compileLinux386(input, output)
	}
	if target == rtgTargetLinuxAarch64 {
		return compileLinuxAarch64(input, output)
	}
	if target == rtgTargetLinuxArm {
		return compileLinuxArm(input, output)
	}
	if target != rtgTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64(input, output)
}

func rtgLinuxSysWriteSeq() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysWriteSeq
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysWriteSeq
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysWriteSeq
	}
	return rtgLinuxAmd64SysWriteSeq
}

func rtgLinuxSysReadSeq() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysReadSeq
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysReadSeq
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysReadSeq
	}
	return rtgLinuxAmd64SysReadSeq
}

func rtgLinuxSysReadAt() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysReadAt
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysReadAt
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysReadAt
	}
	return rtgLinuxAmd64SysReadAt
}

func rtgLinuxSysWriteAt() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysWriteAt
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysWriteAt
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysWriteAt
	}
	return rtgLinuxAmd64SysWriteAt
}

func rtgLinuxSysOpen() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysOpen
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysOpen
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysOpen
	}
	return rtgLinuxAmd64SysOpen
}

func rtgLinuxSysClose() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysClose
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysClose
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysClose
	}
	return rtgLinuxAmd64SysClose
}

func rtgLinuxSysFchmod() int {
	if rtgTargetArch == rtgArchAarch64 {
		return rtgLinuxAarch64SysFchmod
	}
	if rtgTargetArch == rtgArchArm {
		return rtgLinuxArmSysFchmod
	}
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysFchmod
	}
	return rtgLinuxAmd64SysFchmod
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
	if rtgTargetIsWindows() {
		return rtgEmitWindowsPrintStmt(g, stmt)
	}
	p := g.prog
	a := &g.asm
	if stmt.exprStart < 0 || stmt.exprStart >= len(p.toks) || !rtgBytesEqualText(p.src, p.toks[stmt.exprStart].start, p.toks[stmt.exprStart].end, "print") {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, &ep, root.left, "print") {
		return false
	}
	if !rtgEmitStringValueRegs(g, &ep, ep.args[root.firstArg]) {
		return false
	}
	rtgAsmPushImm(a, 1)
	rtgAsmPopRdi(a)
	rtgAsmMovRsiRax(a)
	rtgAsmMovRaxImm(a, rtgLinuxSysWriteSeq())
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
	fdEp := rtgParseExpression(p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	fdIndex := len(fdEp.exprs) - 1
	if !rtgEmitIntExpr(g, &fdEp, fdIndex) {
		return false
	}
	rtgAsmPushRax(a)
	offIndex := ep.args[firstArg+2]
	offConst := rtgEvalConstExpr(g, ep, offIndex)
	offsetRead := true
	if offConst.ok && offConst.value < 0 {
		offsetRead = false
	}
	if offsetRead {
		if offConst.ok {
			rtgAsmMovRaxImm(a, offConst.value)
		} else {
			if !rtgEmitIntExpr(g, ep, offIndex) {
				return false
			}
		}
		rtgAsmPushRax(a)
	}
	if !rtgEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	rtgAsmPrepareReadWriteBuf(a)
	if offsetRead {
		rtgAsmPopRax(a)
		rtgAsmMoveOffsetArg(a)
	}
	rtgAsmPopRdi(a)
	if offsetRead {
		rtgAsmMovRaxImm(a, offSyscall)
	} else {
		rtgAsmMovRaxImm(a, seqSyscall)
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
	stdCountOff := 0
	if isWrite {
		stdCountOff = a.bssSize
		a.bssSize += 8
	}
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
	label := g.winReadLabel
	importID := rtgWinImportRead
	if isWrite {
		label = g.winWriteLabel
		importID = rtgWinImportWrite
	}
	afterLabel := rtgAsmNewLabel(a)
	seqLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, label)
	if isWrite {
		stdOutLabel := rtgAsmNewLabel(a)
		stdErrLabel := rtgAsmNewLabel(a)
		afterStdLabel := rtgAsmNewLabel(a)
		rtgAsmEmit4(a, 0x48, 0x83, 0xff, 1)
		rtgAsmJzLabel(a, stdOutLabel)
		rtgAsmEmit4(a, 0x48, 0x83, 0xff, 2)
		rtgAsmJzLabel(a, stdErrLabel)
		rtgAsmJmpLabel(a, afterStdLabel)
		rtgAsmMarkLabel(a, stdOutLabel)
		rtgWinAmd64EmitStdWrite(a, -11, stdCountOff)
		rtgAsmMarkLabel(a, stdErrLabel)
		rtgWinAmd64EmitStdWrite(a, -12, stdCountOff)
		rtgAsmMarkLabel(a, afterStdLabel)
	}
	rtgAsmEmit4(a, 0x48, 0x83, 0xf9, 0)
	rtgAsmJlLabel(a, seqLabel)

	rtgAsmEmit8(a, 0x57)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x52)
	rtgAsmEmit8(a, 0x51)
	rtgAsmEmit8(a, 0x50)
	rtgAsmEmit4(a, 0x48, 0x8b, 0x4c, 0x24)
	rtgAsmEmit8(a, 32)
	rtgAsmEmit16(a, 0xd231)
	rtgAsmEmit3(a, 0x41, 0xb8, 1)
	rtgAsmEmit24(a, 0)
	rtgWinAmd64CallImport(a, rtgWinImportLseek, 32)
	rtgAsmEmit32(a, 0x24048948)

	rtgAsmEmit4(a, 0x48, 0x8b, 0x4c, 0x24)
	rtgAsmEmit8(a, 32)
	rtgAsmEmit4(a, 0x48, 0x8b, 0x54, 0x24)
	rtgAsmEmit8(a, 8)
	rtgAsmEmit24(a, 0xc03145)
	rtgWinAmd64CallImport(a, rtgWinImportLseek, 32)

	rtgAsmEmit4(a, 0x48, 0x8b, 0x4c, 0x24)
	rtgAsmEmit8(a, 32)
	rtgAsmEmit4(a, 0x48, 0x8b, 0x54, 0x24)
	rtgAsmEmit8(a, 24)
	rtgAsmEmit4(a, 0x4c, 0x8b, 0x44, 0x24)
	rtgAsmEmit8(a, 16)
	rtgWinAmd64CallImport(a, importID, 32)
	rtgAsmEmit4(a, 0x48, 0x89, 0x44, 0x24)
	rtgAsmEmit8(a, 8)

	rtgAsmEmit4(a, 0x48, 0x8b, 0x4c, 0x24)
	rtgAsmEmit8(a, 32)
	rtgAsmEmit32(a, 0x24148b48)
	rtgAsmEmit24(a, 0xc03145)
	rtgWinAmd64CallImport(a, rtgWinImportLseek, 32)
	rtgAsmEmit4(a, 0x48, 0x8b, 0x44, 0x24)
	rtgAsmEmit8(a, 8)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 40)
	rtgAsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, seqLabel)
	rtgAsmEmit24(a, 0xd08949)
	rtgAsmEmit24(a, 0xf28948)
	rtgAsmEmit24(a, 0xf98948)
	rtgWinAmd64CallImport(a, importID, 40)

	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return label
}

func rtgWinAmd64EmitStdWrite(a *rtgAsm, stdHandle int, countOff int) {
	failLabel := rtgAsmNewLabel(a)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x52)
	rtgAsmEmit8(a, 0xb9)
	rtgAsmEmit32(a, stdHandle)
	rtgWinAmd64CallImport(a, rtgWinImportGetStdHandle, 40)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, countOff)
	rtgAsmMovR9Rax(a)
	rtgAsmPopRcx(a)
	rtgAsmEmit32(a, 0x24048b4c)
	rtgAsmEmit4(a, 0x48, 0x8b, 0x54, 0x24)
	rtgAsmEmit8(a, 8)
	rtgAsmEmit4(a, 0x48, 0x83, 0xec, 40)
	rtgAsmEmit4(a, 0x48, 0xc7, 0x44, 0x24)
	rtgAsmEmit8(a, 32)
	rtgAsmEmit32(a, 0)
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, rtgWinImportWriteFile)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 40)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmLoadRaxBss(a, countOff)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 16)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, failLabel)
	rtgAsmMovRaxImm(a, -1)
	rtgAsmEmit4(a, 0x48, 0x83, 0xc4, 16)
	rtgAsmRet(a)
}

func rtgWin386EmitReadWriteHelper(g *rtgLinearGen, isWrite bool) int {
	a := &g.asm
	stdCountOff := 0
	if isWrite {
		stdCountOff = a.bssSize
		a.bssSize += 8
	}
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
	label := g.winReadLabel
	importID := rtgWinImportRead
	if isWrite {
		label = g.winWriteLabel
		importID = rtgWinImportWrite
	}
	afterLabel := rtgAsmNewLabel(a)
	seqLabel := rtgAsmNewLabel(a)
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
		rtgWin386EmitStdWrite(a, -11, stdCountOff)
		rtgAsmMarkLabel(a, stdErrLabel)
		rtgWin386EmitStdWrite(a, -12, stdCountOff)
		rtgAsmMarkLabel(a, afterStdLabel)
	}
	rtgAsmEmit3(a, 0x83, 0xf9, 0)
	rtgAsmJlLabel(a, seqLabel)

	rtgAsmEmit8(a, 0x53)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x52)
	rtgAsmEmit8(a, 0x51)
	rtgAsmEmit8(a, 0x50)
	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 16)
	rtgAsmPushImm(a, 1)
	rtgAsmPushImm(a, 0)
	rtgAsmPushRax(a)
	rtgWin386CallImport(a, rtgWinImportLseek)
	rtgAsmEmit3(a, 0x83, 0xc4, 12)
	rtgAsmEmit24(a, 0x240489)

	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 16)
	rtgAsmPushImm(a, 0)
	rtgAsmEmit4(a, 0xff, 0x74, 0x24, 8)
	rtgAsmPushRax(a)
	rtgWin386CallImport(a, rtgWinImportLseek)
	rtgAsmEmit3(a, 0x83, 0xc4, 12)

	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 16)
	rtgAsmEmit4(a, 0xff, 0x74, 0x24, 8)
	rtgAsmEmit4(a, 0xff, 0x74, 0x24, 16)
	rtgAsmPushRax(a)
	rtgWin386CallImport(a, importID)
	rtgAsmEmit3(a, 0x83, 0xc4, 12)
	rtgAsmEmit4(a, 0x89, 0x44, 0x24, 4)

	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 16)
	rtgAsmPushImm(a, 0)
	rtgAsmEmit4(a, 0xff, 0x74, 0x24, 4)
	rtgAsmPushRax(a)
	rtgWin386CallImport(a, rtgWinImportLseek)
	rtgAsmEmit3(a, 0x83, 0xc4, 12)
	rtgAsmEmit4(a, 0x8b, 0x44, 0x24, 4)
	rtgAsmEmit3(a, 0x83, 0xc4, 20)
	rtgAsmJmpLabel(a, doneLabel)

	rtgAsmMarkLabel(a, seqLabel)
	rtgAsmPushRdx(a)
	rtgAsmEmit8(a, 0x56)
	rtgAsmEmit8(a, 0x53)
	rtgWin386CallImport(a, importID)
	rtgAsmEmit3(a, 0x83, 0xc4, 12)

	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return label
}

func rtgWin386EmitStdWrite(a *rtgAsm, stdHandle int, countOff int) {
	failLabel := rtgAsmNewLabel(a)
	rtgAsmEmit8(a, 0x56)
	rtgAsmPushRdx(a)
	rtgAsmPushImm(a, stdHandle)
	rtgWin386CallImport(a, rtgWinImportGetStdHandle)
	rtgAsmEmit3(a, 0x8b, 0x0c, 0x24)
	rtgAsmEmit4(a, 0x8b, 0x54, 0x24, 4)
	rtgAsmPushImm(a, 0)
	rtgWin386PushBssAddr(a, countOff)
	rtgAsmPushRcx(a)
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
	rtgWin386CallImport(a, rtgWinImportWriteFile)
	rtgAsmEmit3(a, 0x83, 0xf8, 0)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmLoadRaxBss(a, countOff)
	rtgAsmEmit3(a, 0x83, 0xc4, 8)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, failLabel)
	rtgAsmMovRaxImm(a, -1)
	rtgAsmEmit3(a, 0x83, 0xc4, 8)
	rtgAsmRet(a)
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
	fdEp := rtgParseExpression(p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	if !rtgEmitIntExpr(g, &fdEp, len(fdEp.exprs)-1) {
		return false
	}
	rtgAsmPushRax(a)
	offIndex := ep.args[firstArg+2]
	offConst := rtgEvalConstExpr(g, ep, offIndex)
	if offConst.ok && offConst.value < 0 {
		rtgAsmMovRaxImm(a, -1)
	} else if offConst.ok {
		rtgAsmMovRaxImm(a, offConst.value)
	} else {
		if !rtgEmitIntExpr(g, ep, offIndex) {
			return false
		}
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	if rtgTargetArch == rtgArch386 {
		label := rtgWin386EmitReadWriteHelper(g, isWrite)
		rtgAsmEmit16(a, 0xc689)
		rtgAsmEmit16(a, 0xca89)
		rtgAsmPopRcx(a)
		rtgAsmPopRdi(a)
		rtgAsmCallLabel(a, label)
		return true
	}
	label := rtgWinAmd64EmitReadWriteHelper(g, isWrite)
	rtgAsmMovRsiRax(a)
	rtgAsmEmit24(a, 0xca8948)
	rtgAsmPopRcx(a)
	rtgAsmPopRdi(a)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgEmitWindowsPrintStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	p := g.prog
	a := &g.asm
	if stmt.exprStart < 0 || stmt.exprStart >= len(p.toks) || !rtgBytesEqualText(p.src, p.toks[stmt.exprStart].start, p.toks[stmt.exprStart].end, "print") {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, &ep, root.left, "print") {
		return false
	}
	if !rtgEmitStringValueRegs(g, &ep, ep.args[root.firstArg]) {
		return false
	}
	if rtgTargetArch == rtgArch386 {
		label := rtgWin386EmitReadWriteHelper(g, true)
		rtgAsmEmit16(a, 0xc689)
		rtgAsmMovRaxImm(a, -1)
		rtgAsmMovRcxRax(a)
		rtgAsmMovRaxImm(a, 1)
		rtgAsmMovRdiRax(a)
		rtgAsmCallLabel(a, label)
		return true
	}
	label := rtgWinAmd64EmitReadWriteHelper(g, true)
	rtgAsmMovRsiRax(a)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMovRdiRax(a)
	rtgAsmMovRaxImm(a, -1)
	rtgAsmMovRcxRax(a)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgWinAmd64TranslateOpenFlags(a *rtgAsm) {
	rtgAsmEmit24(a, 0xd08948)
	rtgAsmEmit4(a, 0x48, 0x83, 0xe2, 0xbf)
	rtgAsmEmit3(a, 0x83, 0xe0, 0x40)
	rtgAsmEmit3(a, 0xc1, 0xe0, 2)
	rtgAsmEmit16(a, 0xc209)
	rtgAsmEmit16(a, 0xca81)
	rtgAsmEmit32(a, 0x8000)
}

func rtgWin386TranslateOpenFlags(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc189)
	rtgAsmEmit3(a, 0x83, 0xe1, 0xbf)
	rtgAsmEmit3(a, 0x83, 0xe0, 0x40)
	rtgAsmEmit3(a, 0xc1, 0xe0, 2)
	rtgAsmEmit16(a, 0xc109)
	rtgAsmEmit16(a, 0xc981)
	rtgAsmEmit32(a, 0x8000)
}

func rtgEmitWindowsOpen(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	if rtgTargetArch == rtgArch386 {
		rtgAsmPopRcx(a)
		rtgAsmPushRax(a)
		rtgAsmEmit16(a, 0xc889)
		rtgWin386TranslateOpenFlags(a)
		rtgAsmPopRax(a)
		rtgAsmPushImm(a, 438)
		rtgAsmPushRcx(a)
		rtgAsmPushRax(a)
		rtgWin386CallImport(a, rtgWinImportOpen)
		rtgAsmEmit3(a, 0x83, 0xc4, 12)
		return true
	}
	rtgAsmMovRcxRax(a)
	rtgAsmPopRdx(a)
	rtgAsmMovRaxRdx(a)
	rtgWinAmd64TranslateOpenFlags(a)
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit32(a, 438)
	rtgWinAmd64CallImport(a, rtgWinImportOpen, 40)
	return true
}

func rtgEmitWindowsClose(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	if rtgTargetArch == rtgArch386 {
		rtgAsmPushRax(a)
		rtgWin386CallImport(a, rtgWinImportClose)
		rtgAsmEmit3(a, 0x83, 0xc4, 4)
		return true
	}
	rtgAsmMovRcxRax(a)
	rtgWinAmd64CallImport(a, rtgWinImportClose, 40)
	return true
}

func rtgEmitWindowsChmod(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
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
	if rtgTargetArch == rtgArch386 {
		rtgAsmPopRax(a)
		rtgAsmPushImm(a, 1)
		rtgAsmPushImm(a, 0)
		rtgAsmPushRax(a)
		rtgWin386CallImport(a, rtgWinImportLseek)
		rtgAsmEmit3(a, 0x83, 0xc4, 12)
		rtgAsmEmit3(a, 0x83, 0xf8, -1)
		failLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmJzLabel(a, failLabel)
		rtgAsmMovRaxImm(a, 0)
		rtgAsmJmpLabel(a, doneLabel)
		rtgAsmMarkLabel(a, failLabel)
		rtgAsmMovRaxImm(a, -1)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	rtgAsmPopRcx(a)
	rtgAsmEmit16(a, 0xd231)
	rtgAsmEmit8(a, 0x41)
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit32(a, 1)
	rtgWinAmd64CallImport(a, rtgWinImportLseek, 40)
	rtgAsmEmit3(a, 0x83, 0xf8, -1)
	failLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJzLabel(a, failLabel)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmJmpLabel(a, doneLabel)
	rtgAsmMarkLabel(a, failLabel)
	rtgAsmMovRaxImm(a, -1)
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
		return rtgConstResultOk(64)
	}
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "O_TRUNC") {
		return rtgConstResultOk(512)
	}
	var r rtgConstResult
	return r
}
