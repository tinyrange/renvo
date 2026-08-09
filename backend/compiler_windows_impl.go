package main

func renvoWin386CallImport(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x15ff)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386LoadImportPtrRax(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386LoadImportPtrRsi(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x358b)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386StoreEcxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x0d89)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEbxBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEcxBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xb9)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEdiBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xbf)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386PushBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0x68)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386LoadEsiBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x358b)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386LoadEaxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386EmitReadWriteHelper(g *renvoLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = renvoAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = renvoAsmNewLabel(a)
	}
	countOff := a.bssSize
	a.bssSize = countOff + 16
	// The helper instruction streams are invariant. Relocation pairs encode
	// their patch offset and either a BSS slot or Windows import identifier.
	label := g.winReadLabel
	base := len(a.code)
	relocs := ""
	if isWrite {
		label = g.winWriteLabel
		renvoAsmEmitText(a, "\xe9\xc4\x00\x00\x00\x83\xfb\x01\x0f\x84\x0e\x00\x00\x00\x83\xfb\x02\x0f\x84\x14\x00\x00\x00\xe9\x19\x00\x00\x00\x6a\xf5\xff\x15\x00\x00\x00\x00\x89\xc3\xe9\x0a\x00\x00\x00\x6a\xf4\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x20\x08\x2f\x08\x49\x07\x4e\x01\x5e\x07\x67\x00\x70\x06\x7e\x00\x8a\x00\x93\x06\xa1\x00\xae\x00\xb7\x01\xbf\x07\xc4\x00"
	} else {
		renvoAsmEmitText(a, "\xe9\xac\x00\x00\x00\x83\xfb\x00\x0f\x84\x05\x00\x00\x00\xe9\x0a\x00\x00\x00\x6a\xf6\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x17\x08\x31\x07\x36\x01\x46\x07\x4f\x00\x58\x05\x66\x00\x72\x00\x7b\x05\x89\x00\x96\x00\x9f\x01\xa7\x07\xac\x00"
	}
	for i := 0; i < len(relocs); i += 2 {
		at := base + int(relocs[i])
		kind := int(relocs[i+1])
		if kind < 2 {
			renvoAsmAddAbsReloc(a, at, countOff+(kind<<3), renvoAbsBssReloc)
		} else {
			renvoAsmAddWinImportReloc(a, at, kind-2)
		}
	}
	a.labelPos[label] = int32(base + 5)
	return label
}

func renvoWin386SetStdHandle(a *renvoAsm, stdHandle int) {
	renvoNonNil(a)
	renvoAsmPushImm(a, stdHandle)
	renvoWin386CallImport(a, renvoWinImportGetStdHandle)
	renvoAsmCopyPrimaryToCallWord0(a)
}

func renvoWin386EmitKernelReadWriteCall(a *renvoAsm, importID int, countOff int) {
	renvoNonNil(a)
	renvoAsmPushImm(a, 0)
	renvoWin386PushBssAddr(a, countOff)
	renvoAsmPushSecondary(a)
	renvoAsmEmit8(a, 0x56)
	renvoAsmEmit8(a, 0x53)
	renvoWin386CallImport(a, importID)
}

func renvoWin386TranslateCreateFileFlags(a *renvoAsm) {
	renvoNonNil(a)
	notRDWRLabel := renvoAsmNewLabel(a)
	accessDoneLabel := renvoAsmNewLabel(a)
	noCreateLabel := renvoAsmNewLabel(a)
	createDoneLabel := renvoAsmNewLabel(a)

	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, -2147483648)
	renvoAsmEmit2(a, 0xa8, 2)
	renvoAsmJzLabel(a, notRDWRLabel)
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, -1073741824)
	renvoAsmJmpMarkLabel(a, accessDoneLabel, notRDWRLabel)
	renvoAsmEmit2(a, 0xa8, 1)
	renvoAsmJzLabel(a, accessDoneLabel)
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, 0x40000000)
	renvoAsmMarkLabel(a, accessDoneLabel)

	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 3)
	renvoAsmEmit2(a, 0xa8, 64)
	renvoAsmJzLabel(a, noCreateLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 4)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 2)
	renvoAsmJmpMarkLabel(a, createDoneLabel, noCreateLabel)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 5)
	renvoAsmMarkLabel(a, createDoneLabel)
}

func renvoEmitWindowsReadWrite(g *renvoLinearGen, ep *renvoExprParse, idx int, isWrite bool) bool {
	renvoNonNil(g, ep)
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
	if !renvoEmitIntExpr(g, fdEp, len(fdEp.exprs)-1) {
		return false
	}
	renvoAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := renvoEvalConstExpr(g, ep, offIndex)
	if offConst.ok && offConst.value < 0 {
		renvoAsmPrimaryImm(a, -1)
	} else if offConst.ok {
		renvoAsmPrimaryImm(a, offConst.value)
	} else {
		if !renvoEmitIntExpr(g, ep, offIndex) {
			return false
		}
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	if g.c.renvoTargetArch == renvoArch386 {
		label := renvoWin386EmitReadWriteHelper(g, isWrite)
		renvoAsmEmit16(a, 0xc689)
		renvoAsmEmit16(a, 0xca89)
		renvoAsmPopTertiary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		label := renvoWinArm64EmitReadWriteHelper(g, isWrite)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRcx)
		renvoAsmPopTertiary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	label := renvoWinAmd64EmitReadWriteHelper(g, isWrite)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmEmit24(a, 0xca8948)
	renvoAsmPopTertiary(a)
	renvoAsmPopCallWord0(a)
	renvoAsmCallLabel(a, label)
	return true
}

func renvoEmitWindowsOpen(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
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
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmEmit16(a, 0xc689)
		renvoAsmPopPrimary(a)
		renvoWin386TranslateCreateFileFlags(a)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 0x80)
		renvoAsmPushTertiary(a)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 3)
		renvoAsmPushSecondary(a)
		renvoAsmEmit8(a, 0x56)
		renvoWin386CallImport(a, renvoWinImportCreateFileA)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRegReg(a, 9, 0)
		renvoAsmPopPrimary(a)
		renvoWinArm64TranslateCreateFileFlags(a)
		renvoAarch64AsmMovRegReg(a, 0, 9)
		renvoWinArm64CallImport(a, renvoWinImportCreateFileA)
		return true
	}
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmPopCallWord1(a)
	renvoWinAmd64EmitRuntimeOpen(a)
	return true
}

func renvoEmitWindowsClose(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	failLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportCloseHandle)
		renvoAsmEmit3(a, 0x83, 0xf8, 0)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoWinArm64CallImport(a, renvoWinImportCloseHandle)
		renvoAsmCmpPrimaryImm8(a, 0)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	renvoAsmCopyPrimaryToTertiary(a)
	renvoWinAmd64EmitRuntimeClose(a)
	return true
}

func renvoEmitWindowsChmod(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
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
	failLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmPopPrimary(a)
		renvoAsmPushImm(a, 1)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 0)
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportSetFilePointer)
		renvoAsmEmit3(a, 0x83, 0xf8, -1)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmPopPrimary(a)
		renvoAarch64AsmMovRegImm(a, 1, 0)
		renvoAarch64AsmMovRegImm(a, 2, 0)
		renvoAarch64AsmMovRegImm(a, 3, 1)
		renvoWinArm64CallImport(a, renvoWinImportSetFilePointer)
		renvoAarch64AsmCmpRegImm(a, 0, -1)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	renvoAsmPopTertiary(a)
	renvoWinAmd64EmitRuntimeChmod(a)
	return true
}
