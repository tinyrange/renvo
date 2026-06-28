package main

func compileWasiWasm32(input []int, output int) int {
	rtgSetTarget(rtgTargetWasiWasm32)
	src := make([]byte, 0, 655360)
	for i := 0; i < len(input); i++ {
		src = rtgReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta rtgMeta
	rtgBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	var result rtgCompileResult
	result = rtgTryCompileScalarProgramWasm32(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	rtgPrintErr("rtg: wasm32 compilation failed\n")
	return 1
}

func rtgTryCompileScalarProgramWasm32(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result rtgCompileResult
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	a := &g.asm
	rtgAsmInit(a)
	for i := 0; i < len(meta.funcs); i++ {
		label := rtgAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	g.funcReachable = make([]bool, len(meta.funcs), 1280)
	g.funcQueue = make([]int, 0, 1280)
	rtgWasm32MarkFunc(&g, appIndex)
	if !rtgLinearInitGlobals(&g) {
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgsWasm32(&g, appIndex) {
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	rtgWasm32AsmExit(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !rtgEmitScalarFunction(&g, i) {
			rtgPrintErr("rtg: wasm32 failed in function ")
			write(2, meta.prog.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
			rtgPrintErr("\n")
			var result rtgCompileResult
			return result
		}
	}
	data := rtgWasm32Image(a)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgsWasm32(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	rtgWasm32AsmBuildArgvEnvSlices(&g.asm, argsOff, envDataOff, envLenOff)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !rtgTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !rtgTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func rtgTryCompileWasiWasm32(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result rtgCompileResult
		return result
	}
	app := &meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(meta, app.resultType) {
		var result rtgCompileResult
		return result
	}
	if app.paramCount > 1 {
		var result rtgCompileResult
		return result
	}
	if app.paramCount == 1 {
		first := &meta.params[app.firstParam]
		if !rtgTypeIsStringSlice(meta, first.typ) {
			var result rtgCompileResult
			return result
		}
	}
	fn := &p.funcs[app.declIndex]
	var body rtgBodyParse
	stmts := make([]rtgStmt, 0, 1024)
	body.prog = p
	body.stmts = stmts
	body.ok = true
	i := fn.bodyStart + 1
	for body.ok && i < fn.bodyEnd {
		if rtgTokCharIs(p, i, ';') {
			i++
			continue
		}
		if rtgTokCharIs(p, i, '}') || rtgTokIsKind(p, i, rtgTokEOF) {
			break
		}
		before := len(body.stmts)
		next := rtgParseOneStatement(&body, i, fn.bodyEnd)
		if !body.ok || next <= i || len(body.stmts) <= before {
			var result rtgCompileResult
			return result
		}
		i = next
	}
	if !body.ok {
		var result rtgCompileResult
		return result
	}
	data := rtgWasiWasm32EmitBinary(p, meta, &body)
	if len(data) == 0 {
		var result rtgCompileResult
		return result
	}
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgWasiWasm32EmitBinary(p *rtgProgram, meta *rtgMeta, body *rtgBodyParse) []byte {
	dataOff := 1024
	exitCode := 0
	var code []byte
	var data []byte
	var gen rtgLinearGen
	gen.prog = p
	gen.meta = meta
	for i := 0; i < len(body.stmts); i++ {
		stmt := &body.stmts[i]
		if stmt.kind == rtgStmtExpr {
			ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			rootIndex := len(ep.exprs) - 1
			root := &ep.exprs[rootIndex]
			if root.kind != rtgExprCall || root.argCount != 1 || !rtgExprIsIdentText(p, &ep, root.left, "print") {
				return nil
			}
			arg := &ep.exprs[ep.args[root.firstArg]]
			if arg.kind != rtgExprString {
				return nil
			}
			msg := rtgDecodeStringToken(p, arg.tok)
			msgOff := dataOff + len(data)
			for j := 0; j < len(msg); j++ {
				data = append(data, msg[j])
			}
			code = rtgWasiWasm32AppendPrint(code, msgOff, len(msg))
			continue
		}
		if stmt.kind == rtgStmtReturn {
			if stmt.exprStart == stmt.exprEnd {
				exitCode = 0
				continue
			}
			ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			result := rtgEvalConstExpr(&gen, &ep, len(ep.exprs)-1)
			if !result.ok {
				return nil
			}
			exitCode = result.value
			continue
		}
		return nil
	}
	code = rtgWasmAppendI32Const(code, exitCode)
	code = rtgWasmAppendCall(code, 1)

	var out []byte
	out = append(out, 0x00)
	out = append(out, 0x61)
	out = append(out, 0x73)
	out = append(out, 0x6d)
	out = append(out, 0x01)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = append(out, 0x00)

	out = rtgWasmAppendSection(out, 1, rtgWasiWasm32TypeSection())
	out = rtgWasmAppendSection(out, 2, rtgWasiWasm32ImportSection())
	out = rtgWasmAppendSection(out, 3, rtgWasiWasm32FunctionSection())
	out = rtgWasmAppendSection(out, 5, rtgWasiWasm32MemorySection())
	out = rtgWasmAppendSection(out, 7, rtgWasiWasm32ExportSection())
	out = rtgWasmAppendSection(out, 10, rtgWasiWasm32CodeSection(code))
	out = rtgWasmAppendSection(out, 11, rtgWasiWasm32DataSection(dataOff, data))
	return out
}

func rtgWasiWasm32AppendPrint(out []byte, ptr int, length int) []byte {
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmAppendI32Const(out, ptr)
	out = rtgWasmAppendI32Store(out)
	out = rtgWasmAppendI32Const(out, 4)
	out = rtgWasmAppendI32Const(out, length)
	out = rtgWasmAppendI32Store(out)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmAppendI32Const(out, 8)
	out = rtgWasmAppendCall(out, 0)
	out = append(out, 0x1a)
	return out
}

func rtgWasiWasm32TypeSection() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 3)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 4)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = rtgWasmAppendU32(out, 0)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasiWasm32ImportSection() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 2)
	out = rtgWasmAppendName(out, "wasi_snapshot_preview1")
	out = rtgWasmAppendName(out, "fd_write")
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendName(out, "wasi_snapshot_preview1")
	out = rtgWasmAppendName(out, "proc_exit")
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, 1)
	return out
}

func rtgWasiWasm32FunctionSection() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendU32(out, 2)
	return out
}

func rtgWasiWasm32MemorySection() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, 1)
	return out
}

func rtgWasiWasm32ExportSection() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 2)
	out = rtgWasmAppendName(out, "memory")
	out = append(out, 0x02)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendName(out, "_start")
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, 2)
	return out
}

func rtgWasiWasm32CodeSection(code []byte) []byte {
	var body []byte
	body = rtgWasmAppendU32(body, 0)
	for i := 0; i < len(code); i++ {
		body = append(body, code[i])
	}
	body = append(body, 0x0b)
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendByteVec(out, body)
	return out
}

func rtgWasiWasm32DataSection(dataOff int, data []byte) []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = rtgWasmAppendI32Const(out, dataOff)
	out = append(out, 0x0b)
	out = rtgWasmAppendByteVec(out, data)
	return out
}
