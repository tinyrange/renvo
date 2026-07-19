package main

func compileWasiWasm32(input []int, output int) int {
	return compileWasiWasm32Arena(input, output, 0)
}

func compileWasiWasm32Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWasiWasm32)
	src := make([]byte, 0, 655360)
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
	meta.arenaSize = renvoResolveArenaSize(currentTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramWasm32(&prog, &meta)
	if result.ok {
		write(output, result.data, -1)
		return 0
	}
	renvoPrintErr("renvo: wasm32 compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramWasm32(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result renvoCompileResult
		return result
	}
	var g renvoLinearGen
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	g.fixedTargetState = 1
	g.fixedTargetValue = renvoTargetWasiWasm32
	a := &g.asm
	renvoAsmInit(a)
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(&g, len(meta.funcs))
	renvoWasm32MarkFunc(&g, appIndex)
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) {
		var result renvoCompileResult
		return result
	}
	if !renvoEmitProgramEntryArgsWasm32(&g, appIndex) {
		var result renvoCompileResult
		return result
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		var result renvoCompileResult
		return result
	}
	renvoWasm32AsmExit(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !renvoEmitScalarFunctionScratch(&g, i) {
			renvoPrintErr("renvo: wasm32 failed in function ")
			write(2, meta.prog.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
			renvoPrintErr("\n")
			var result renvoCompileResult
			return result
		}
	}
	data := renvoWasm32Image(a)
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}
func renvoEmitProgramEntryArgsWasm32(g *renvoLinearGen, appIndex int) bool {
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
	renvoWasm32AsmBuildArgvEnvSlices(&g.asm, argsOff, envDataOff, envLenOff)
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

func renvoTryCompileWasiWasm32(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		var result renvoCompileResult
		return result
	}
	app := &meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(meta, app.resultType) {
		var result renvoCompileResult
		return result
	}
	if app.paramCount > 1 {
		var result renvoCompileResult
		return result
	}
	if app.paramCount == 1 {
		first := &meta.params[app.firstParam]
		if !renvoTypeIsStringSlice(meta, first.typ) {
			var result renvoCompileResult
			return result
		}
	}
	fn := &p.funcs[app.declIndex]
	var body renvoBodyParse
	stmtData := make([]int, 1024*renvoStmtWordCount)
	renvoBodyStmtData = stmtData
	body.prog = p
	body.stmtCount = 0
	body.ok = true
	i := fn.bodyStart + 1
	for body.ok && i < fn.bodyEnd {
		if renvoTokCharIs(p, i, ';') {
			i++
			continue
		}
		if renvoTokCharIs(p, i, '}') || renvoTokIsKind(p, i, renvoTokEOF) {
			break
		}
		renvoBodyStmtData = stmtData
		before := body.stmtCount
		next := renvoParseOneStatement(&body, i, fn.bodyEnd)
		if !body.ok || next <= i || body.stmtCount <= before {
			var result renvoCompileResult
			return result
		}
		i = next
	}
	if !body.ok {
		var result renvoCompileResult
		return result
	}
	data := renvoWasiWasm32EmitBinary(p, meta, &body)
	if len(data) == 0 {
		var result renvoCompileResult
		return result
	}
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoWasiWasm32EmitBinary(p *renvoProgram, meta *renvoMeta, body *renvoBodyParse) []byte {
	dataOff := 1024
	exitCode := 0
	var code []byte
	var data []byte
	var gen renvoLinearGen
	gen.prog = p
	gen.meta = meta
	for i := 0; i < body.stmtCount; i++ {
		stmtValue := renvoBodyStmtAt(body, i)
		if !body.ok {
			return nil
		}
		stmt := &stmtValue
		if stmt.kind == renvoStmtExpr {
			var ep renvoExprParse
			renvoParseExpressionInto(&ep, p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			rootIndex := len(ep.exprs) - 1
			root := &ep.exprs[rootIndex]
			if root.kind != renvoExprCall || root.argCount != 1 || !renvoExprIsIdentText(p, &ep, root.left, "print") {
				return nil
			}
			arg := &ep.exprs[ep.args[root.firstArg]]
			if arg.kind != renvoExprString {
				return nil
			}
			msg := renvoDecodeStringToken(p, arg.tok)
			msgOff := dataOff + len(data)
			for j := 0; j < len(msg); j++ {
				data = append(data, msg[j])
			}
			code = renvoWasiWasm32AppendPrint(code, msgOff, len(msg))
			continue
		}
		if stmt.kind == renvoStmtReturn {
			if stmt.exprStart == stmt.exprEnd {
				exitCode = 0
				continue
			}
			var ep renvoExprParse
			renvoParseExpressionInto(&ep, p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			result := renvoEvalConstExpr(&gen, &ep, len(ep.exprs)-1)
			if !result.ok {
				return nil
			}
			exitCode = result.value
			continue
		}
		return nil
	}
	code = renvoWasmAppendI32Const(code, exitCode)
	code = renvoWasmAppendCall(code, 1)

	var out []byte
	out = append(out, 0x00)
	out = append(out, 0x61)
	out = append(out, 0x73)
	out = append(out, 0x6d)
	out = append(out, 0x01)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = append(out, 0x00)

	out = renvoWasmAppendSection(out, 1, renvoWasiWasm32TypeSection())
	out = renvoWasmAppendSection(out, 2, renvoWasiWasm32ImportSection())
	out = renvoWasmAppendSection(out, 3, renvoWasiWasm32FunctionSection())
	out = renvoWasmAppendSection(out, 5, renvoWasiWasm32MemorySection())
	out = renvoWasmAppendSection(out, 7, renvoWasiWasm32ExportSection())
	out = renvoWasmAppendSection(out, 10, renvoWasiWasm32CodeSection(code))
	out = renvoWasmAppendSection(out, 11, renvoWasiWasm32DataSection(dataOff, data))
	return out
}

func renvoWasiWasm32AppendPrint(out []byte, ptr int, length int) []byte {
	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmAppendI32Const(out, ptr)
	out = renvoWasmAppendI32Store(out)
	out = renvoWasmAppendI32Const(out, 4)
	out = renvoWasmAppendI32Const(out, length)
	out = renvoWasmAppendI32Store(out)
	out = renvoWasmAppendI32Const(out, 1)
	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmAppendI32Const(out, 1)
	out = renvoWasmAppendI32Const(out, 8)
	out = renvoWasmAppendCall(out, 0)
	out = append(out, 0x1a)
	return out
}

func renvoWasiWasm32TypeSection() []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 3)
	out = append(out, 0x60)
	out = renvoWasmAppendU32(out, 4)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = append(out, 0x7f)
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = renvoWasmAppendU32(out, 0)
	out = append(out, 0x60)
	out = renvoWasmAppendU32(out, 0)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasiWasm32ImportSection() []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 2)
	out = renvoWasmAppendName(out, "wasi_snapshot_preview1")
	out = renvoWasmAppendName(out, "fd_write")
	out = append(out, 0x00)
	out = renvoWasmAppendU32(out, 0)
	out = renvoWasmAppendName(out, "wasi_snapshot_preview1")
	out = renvoWasmAppendName(out, "proc_exit")
	out = append(out, 0x00)
	out = renvoWasmAppendU32(out, 1)
	return out
}

func renvoWasiWasm32FunctionSection() []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = renvoWasmAppendU32(out, 2)
	return out
}

func renvoWasiWasm32MemorySection() []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = renvoWasmAppendU32(out, 1)
	return out
}

func renvoWasiWasm32ExportSection() []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 2)
	out = renvoWasmAppendName(out, "memory")
	out = append(out, 0x02)
	out = renvoWasmAppendU32(out, 0)
	out = renvoWasmAppendName(out, "_start")
	out = append(out, 0x00)
	out = renvoWasmAppendU32(out, 2)
	return out
}

func renvoWasiWasm32CodeSection(code []byte) []byte {
	var body []byte
	body = renvoWasmAppendU32(body, 0)
	for i := 0; i < len(code); i++ {
		body = append(body, code[i])
	}
	body = append(body, 0x0b)
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = renvoWasmAppendByteVec(out, body)
	return out
}

func renvoWasiWasm32DataSection(dataOff int, data []byte) []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = renvoWasmAppendI32Const(out, dataOff)
	out = append(out, 0x0b)
	out = renvoWasmAppendByteVec(out, data)
	return out
}
