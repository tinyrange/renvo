package main

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
	} else if !renvoEmitIntExpr(g, ep, offIndex) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	if g.c.renvoTargetArch == renvoArch386 {
		// The definition-owned operations consume EBX, ESI, EDX, and ECX.
		renvoAsmEmit16(a, 0xc689)
		renvoAsmEmit16(a, 0xca89)
		renvoAsmPopTertiary(a)
		renvoAsmEmit8(a, 0x5b)
		if isWrite {
			renvoWin386EmitRuntimeWriteAt(g)
		} else {
			renvoWin386EmitRuntimeReadAt(g)
		}
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRcx)
		renvoAsmPopTertiary(a)
		renvoAsmPopCallWord0(a)
		if isWrite {
			renvoWinArm64DefinitionWriteAt(g)
		} else {
			renvoWinArm64DefinitionReadAt(g)
		}
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
		// Preserve the path in EBX while restoring flags to EAX.
		renvoAsmEmit16(a, 0xc389)
		renvoAsmPopPrimary(a)
		renvoWin386EmitRuntimeOpen(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPopCallWord1(a)
		renvoWinArm64DefinitionOpen(a)
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
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmEmit16(a, 0xc389)
		renvoWin386EmitRuntimeClose(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoWinArm64DefinitionClose(a)
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
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmPopPrimary(a)
		renvoAsmEmit16(a, 0xc389)
		renvoWin386EmitRuntimeChmod(a)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmPopPrimary(a)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoWinArm64DefinitionChmod(a)
		return true
	}
	renvoAsmPopTertiary(a)
	renvoWinAmd64EmitRuntimeChmod(a)
	return true
}
