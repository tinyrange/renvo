package main

func compileWasiWasm32(input []int, output int) int {
	return compileWasiWasm32Arena(input, output, 0)
}

func compileWasiWasm32Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWasiWasm32)
	return compileWasm32Arena(input, output, arenaSize)
}

func compileVM32(input []int, output int) int {
	return compileVM32Arena(input, output, 0)
}

func compileVM32Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetVM32)
	return compileWasm32Arena(input, output, arenaSize)
}

func compileWasm32Arena(input []int, output int, arenaSize int) int {
	src := renvoMakeByteScratch(655360)
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
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramWasm32(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
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
		return renvoCompileResult{}
	}
	var g renvoLinearGen
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	g.fixedTargetState = 1
	g.fixedTargetValue = meta.c.renvoTarget
	if meta.c.renvoTarget == renvoTargetVM32 {
		// VM bytecode is an execution format, not a restriction on the targets
		// exposed by a compiler running inside the VM. Preserve dynamic target
		// selection so a runtime -t value remains authoritative.
		g.fixedTargetValue = 0
	}
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(&g, len(meta.funcs))
	renvoWasm32MarkFunc(&g, appIndex)
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) || !renvoEmitProgramEntryArgsWasm32(&g, appIndex) {
		return renvoCompileResult{}
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		return renvoCompileResult{}
	}
	renvoWasm32AsmExit(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if !renvoEmitScalarFunctionScratch(&g, i) {
			renvoPrintErr("renvo: wasm32 failed in function ")
			write(2, meta.prog.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
			renvoPrintErr("\n")
			return renvoCompileResult{}
		}
	}
	var result renvoCompileResult
	if meta.c.renvoTarget == renvoTargetVM32 {
		result.data = renvoVMImage(a)
	} else {
		result.data = renvoWasm32Image(a)
	}
	result.ok = true
	return result
}
func renvoEmitProgramEntryArgsWasm32(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	envDataOff := argsOff
	envLenOff := argsOff
	if g.fixedTargetValue == 0 {
		// VM execution receives arguments from the VM host.
	} else {
		g.asm.bssSize += 32768
		envDataOff = g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff = g.asm.bssSize
		g.asm.bssSize += 8
	}
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
	body.stmtData = make([]int, 1024*renvoStmtWordCount)
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
	var code renvoWasmBuffer
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
			renvoWasiWasm32AppendPrint(&code, msgOff, len(msg))
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
	renvoWasmAppendI32Const(&code, exitCode)
	renvoWasmAppendCall(&code, 1)

	var out renvoWasmBuffer
	renvoWasmAppendEncoded(&out, "\x00\x61\x73\x6d\x01\x00\x00\x00")
	renvoWasmAppendSection(&out, 1, renvoWasiWasm32TypeSection())
	renvoWasmAppendSection(&out, 2, renvoWasiWasm32ImportSection())
	renvoWasmAppendSection(&out, 3, renvoWasiWasm32FunctionSection())
	renvoWasmAppendSection(&out, 5, renvoWasiWasm32MemorySection())
	renvoWasmAppendSection(&out, 7, renvoWasiWasm32ExportSection())
	renvoWasmAppendSection(&out, 10, renvoWasiWasm32CodeSection(code.data))
	renvoWasmAppendSection(&out, 11, renvoWasiWasm32DataSection(dataOff, data))
	return out.data[:out.length]
}

func renvoWasiWasm32AppendPrint(out *renvoWasmBuffer, ptr int, length int) {
	renvoWasmAppendI32Const(out, 0)
	renvoWasmAppendI32Const(out, ptr)
	renvoWasmAppendI32Store(out)
	renvoWasmAppendI32Const(out, 4)
	renvoWasmAppendI32Const(out, length)
	renvoWasmAppendI32Store(out)
	renvoWasmAppendI32Const(out, 1)
	renvoWasmAppendI32Const(out, 0)
	renvoWasmAppendI32Const(out, 1)
	renvoWasmAppendI32Const(out, 8)
	renvoWasmAppendCall(out, 0)
	renvoWasmPut(out, 0x1a)
}

func renvoWasiWasm32TypeSection() []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 3)
	renvoWasmPut(&out, 0x60)
	renvoWasmAppendU32(&out, 4)
	renvoWasmAppend4(&out, 0x7f, 0x7f, 0x7f, 0x7f)
	renvoWasmAppendU32(&out, 1)
	renvoWasmAppend2(&out, 0x7f, 0x60)
	renvoWasmAppendU32(&out, 1)
	renvoWasmPut(&out, 0x7f)
	renvoWasmAppendU32(&out, 0)
	renvoWasmPut(&out, 0x60)
	renvoWasmAppendU32(&out, 0)
	renvoWasmAppendU32(&out, 0)
	return out.data[:out.length]
}

func renvoWasiWasm32ImportSection() []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 2)
	renvoWasmAppendName(&out, "wasi_snapshot_preview1")
	renvoWasmAppendName(&out, "fd_write")
	renvoWasmPut(&out, 0x00)
	renvoWasmAppendU32(&out, 0)
	renvoWasmAppendName(&out, "wasi_snapshot_preview1")
	renvoWasmAppendName(&out, "proc_exit")
	renvoWasmPut(&out, 0x00)
	renvoWasmAppendU32(&out, 1)
	return out.data[:out.length]
}

func renvoWasiWasm32FunctionSection() []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 1)
	renvoWasmAppendU32(&out, 2)
	return out.data[:out.length]
}

func renvoWasiWasm32MemorySection() []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 1)
	renvoWasmPut(&out, 0x00)
	renvoWasmAppendU32(&out, 1)
	return out.data[:out.length]
}

func renvoWasiWasm32ExportSection() []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 2)
	renvoWasmAppendName(&out, "memory")
	renvoWasmPut(&out, 0x02)
	renvoWasmAppendU32(&out, 0)
	renvoWasmAppendName(&out, "_start")
	renvoWasmPut(&out, 0x00)
	renvoWasmAppendU32(&out, 2)
	return out.data[:out.length]
}

func renvoWasiWasm32CodeSection(code []byte) []byte {
	var body renvoWasmBuffer
	renvoWasmAppendU32(&body, 0)
	for i := 0; i < len(code); i++ {
		renvoWasmPut(&body, code[i])
	}
	renvoWasmPut(&body, 0x0b)
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 1)
	renvoWasmAppendByteVec(&out, body.data)
	return out.data[:out.length]
}

func renvoWasiWasm32DataSection(dataOff int, data []byte) []byte {
	var out renvoWasmBuffer
	renvoWasmAppendU32(&out, 1)
	renvoWasmPut(&out, 0x00)
	renvoWasmAppendI32Const(&out, dataOff)
	renvoWasmPut(&out, 0x0b)
	renvoWasmAppendByteVec(&out, data)
	return out.data[:out.length]
}
