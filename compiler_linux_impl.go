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
	if target == rtgTargetLinux386 {
		return compileLinux386(input, output)
	}
	if target != rtgTargetLinuxAmd64 {
		return 1
	}
	return compileLinuxAmd64(input, output)
}

func rtgLinuxSysWriteSeq() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysWriteSeq
	}
	return rtgLinuxAmd64SysWriteSeq
}

func rtgLinuxSysReadSeq() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysReadSeq
	}
	return rtgLinuxAmd64SysReadSeq
}

func rtgLinuxSysReadAt() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysReadAt
	}
	return rtgLinuxAmd64SysReadAt
}

func rtgLinuxSysWriteAt() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysWriteAt
	}
	return rtgLinuxAmd64SysWriteAt
}

func rtgLinuxSysOpen() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysOpen
	}
	return rtgLinuxAmd64SysOpen
}

func rtgLinuxSysClose() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysClose
	}
	return rtgLinuxAmd64SysClose
}

func rtgLinuxSysFchmod() int {
	if rtgTargetArch == rtgArch386 {
		return rtgLinux386SysFchmod
	}
	return rtgLinuxAmd64SysFchmod
}

func rtgAsmPrepareReadWriteBuf(a *rtgAsm) {
	if rtgTargetArch == rtgArch386 {
		rtg386AsmPrepareReadWriteBuf(a)
		return
	}
	rtgAmd64AsmPrepareReadWriteBuf(a)
}

func rtgAsmMoveOffsetArg(a *rtgAsm) {
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMoveOffsetArg(a)
		return
	}
	rtgAmd64AsmMoveOffsetArg(a)
}

func rtgEmitLinearPrintStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
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
	rtgAsmEmit2(a, 0x6a, 1)
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
