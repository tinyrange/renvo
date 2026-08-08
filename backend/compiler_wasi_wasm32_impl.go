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
	localSlotCapacity := len(meta.funcs) * 4
	if localSlotCapacity < 256 {
		localSlotCapacity = 256
	}
	a.wasmLocalSlots = make([]int32, 0, localSlotCapacity)
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

func renvoWasm32EmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, tok int, signed bool) bool {
	renvoNonNil(g)
	if g.c.renvoTarget == renvoTargetVM32 {
		return renvoWasm32EmitWideBinaryPortable(g, dest, left, right, tok, signed)
	}
	if renvoTokStarts2(g.prog, tok, '&', '^') {
		ones := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		renvoAsmStoreStackImm(&g.asm, ones, -1)
		renvoAsmStoreStackImm(&g.asm, ones-g.c.renvoNativeIntSize, -1)
		renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, right, ones, 0x85)
		renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, left, dest, 0x83)
		return true
	}
	op := 0
	if renvoTokStartsWith(g.prog, tok, '+') {
		op = 0x7c
	} else if renvoTokStartsWith(g.prog, tok, '-') {
		op = 0x7d
	} else if renvoTokStartsWith(g.prog, tok, '*') {
		op = 0x7e
	} else if renvoTokStartsWith(g.prog, tok, '/') {
		op = 0x80
		if signed {
			op = 0x7f
		}
	} else if renvoTokStartsWith(g.prog, tok, '%') {
		op = 0x82
		if signed {
			op = 0x81
		}
	} else if renvoTokStartsWith(g.prog, tok, '&') {
		op = 0x83
	} else if renvoTokStartsWith(g.prog, tok, '|') {
		op = 0x84
	} else if renvoTokStartsWith(g.prog, tok, '^') {
		op = 0x85
	} else if renvoTokStarts2(g.prog, tok, '<', '<') {
		op = 0x86
	} else if renvoTokStarts2(g.prog, tok, '>', '>') {
		op = 0x88
		if signed {
			op = 0x87
		}
	}
	if op == 0 {
		return false
	}
	if op >= 0x7f && op <= 0x82 {
		// Keep division by zero on Renvo's panic path rather than allowing a
		// WebAssembly trap, so recover continues to work.
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, left, right, op)
	return true
}

func renvoWasm32EmitWideBinaryPortable(g *renvoLinearGen, dest int, left int, right int, tok int, signed bool) bool {
	if renvoTokStartsWith(g.prog, tok, '+') {
		renvoEmitWideAddStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '-') {
		renvoEmitWideSubStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '*') {
		renvoEmitWideMulStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '/') || renvoTokStartsWith(g.prog, tok, '%') {
		renvoEmitWideDivStack(g, dest, left, right, signed, renvoTokStartsWith(g.prog, tok, '%'))
		return true
	}
	if renvoTokStarts2(g.prog, tok, '<', '<') || renvoTokStarts2(g.prog, tok, '>', '>') {
		renvoEmitWideShiftStack(g, dest, left, right, renvoTokStarts2(g.prog, tok, '>', '>'), signed)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '&') || renvoTokStartsWith(g.prog, tok, '|') || renvoTokStartsWith(g.prog, tok, '^') {
		for at := 0; at < renvoBackendValueSlotSize; at += g.c.renvoNativeIntSize {
			renvoAsmLoadPrimaryStack(&g.asm, right-at)
			renvoAsmLoadTertiaryStack(&g.asm, left-at)
			if !renvoEmitPrimaryTertiaryOp(g, tok) {
				return false
			}
			renvoAsmStorePrimaryStack(&g.asm, dest-at)
		}
		return true
	}
	return false
}

func renvoWasm32EmitWideCompareStack(g *renvoLinearGen, left int, right int, tok int, signed bool) bool {
	renvoNonNil(g)
	if g.c.renvoTarget == renvoTargetVM32 {
		return renvoWasm32EmitWideComparePortable(g, left, right, tok, signed)
	}
	p := g.prog
	op := 0
	if renvoTok2Is(p, tok, '=', '=') {
		op = 0x51
	} else if renvoTok2Is(p, tok, '!', '=') {
		op = 0x52
	} else if renvoTokCharIs(p, tok, '<') {
		op = 0x54
		if signed {
			op = 0x53
		}
	} else if renvoTokCharIs(p, tok, '>') {
		op = 0x56
		if signed {
			op = 0x55
		}
	} else if renvoTok2Is(p, tok, '<', '=') {
		op = 0x58
		if signed {
			op = 0x57
		}
	} else if renvoTok2Is(p, tok, '>', '=') {
		op = 0x5a
		if signed {
			op = 0x59
		}
	}
	if op == 0 {
		return false
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideCompare, 0, left, right, op)
	return true
}

func renvoEmitVM32WideShiftBy(g *renvoLinearGen, count int, opcode int) {
	renvoAsmLoadTertiaryStack(&g.asm, count)
	renvoWasm32EmitRegReg(&g.asm, opcode, renvoWasm32RegRax, renvoWasm32RegRcx)
}

func renvoEmitVM32WideShiftStack(g *renvoLinearGen, dest int, left int, count int, right bool, signed bool) {
	renvoEmitCopyStackToStack(g, left, dest, renvoBackendValueSlotSize)
	large := renvoAsmNewLabel(&g.asm)
	atLeastWord := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, count-g.c.renvoNativeIntSize)
	renvoAsmJnzPrimary(&g.asm, large)
	// A count whose low word has its sign bit set is necessarily at least 2^31.
	renvoAsmJcmpStackImm(&g.asm, count, 0, large, 0x9c)
	renvoAsmJcmpStackImm(&g.asm, count, 32, atLeastWord, 0x9d)
	renvoAsmLoadPrimaryStack(&g.asm, count)
	renvoAsmJzPrimary(&g.asm, done)
	inverse := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmPrimaryImm(&g.asm, 32)
	renvoAsmLoadTertiaryStack(&g.asm, count)
	renvoAsmSubPrimaryTertiary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, inverse)
	if right {
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShrUnsignedRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShlRegReg)
		renvoAsmLoadTertiaryStack(&g.asm, dest)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		opcode := renvoWasm32OpShrUnsignedRegReg
		if signed {
			opcode = renvoWasm32OpShrRegReg
		}
		renvoEmitVM32WideShiftBy(g, count, opcode)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
	} else {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShrUnsignedRegReg)
		renvoAsmLoadTertiaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest)
	}
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, atLeastWord)
	renvoAsmJcmpStackImm(&g.asm, count, 64, large, 0x9d)
	renvoAsmLoadPrimaryStack(&g.asm, count)
	renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpAddRegImm, renvoWasm32RegRax, -32)
	renvoAsmStorePrimaryStack(&g.asm, inverse)
	if right {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		opcode := renvoWasm32OpShrUnsignedRegReg
		if signed {
			opcode = renvoWasm32OpShrRegReg
		}
		renvoEmitVM32WideShiftBy(g, inverse, opcode)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		if signed {
			renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
			renvoAsmSarPrimaryImm(&g.asm, 31)
			renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		} else {
			renvoAsmStoreStackImm(&g.asm, dest-g.c.renvoNativeIntSize, 0)
		}
	} else {
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmStoreStackImm(&g.asm, dest, 0)
	}
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, large)
	if right && signed {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoAsmSarPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
	} else {
		renvoAsmStoreStackImm(&g.asm, dest, 0)
		renvoAsmStoreStackImm(&g.asm, dest-g.c.renvoNativeIntSize, 0)
	}
	renvoAsmMarkLabel(&g.asm, done)
}

func renvoWasm32EmitWideComparePortable(g *renvoLinearGen, left int, right int, tok int, signed bool) bool {
	p := g.prog
	equal := renvoTok2Is(p, tok, '=', '=') || renvoTok2Is(p, tok, '!', '=')
	if equal {
		notEqual := renvoAsmNewLabel(&g.asm)
		done := renvoAsmNewLabel(&g.asm)
		renvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x94)
		renvoAsmJzPrimary(&g.asm, notEqual)
		renvoEmitNativeCompareStack(g, left, right, 0x94)
		renvoAsmJmpMarkLabel(&g.asm, done, notEqual)
		renvoAsmPrimaryImm(&g.asm, 0)
		renvoAsmMarkLabel(&g.asm, done)
		if renvoTok2Is(p, tok, '!', '=') {
			renvoAsmBoolNotPrimary(&g.asm)
		}
		return true
	}
	greater := renvoTokCharIs(p, tok, '>') || renvoTok2Is(p, tok, '>', '=')
	inclusive := renvoTok2Is(p, tok, '<', '=') || renvoTok2Is(p, tok, '>', '=')
	if greater != inclusive {
		left, right = right, left
	}
	renvoEmitWideLessStack(g, left, right, signed)
	if inclusive {
		renvoAsmBoolNotPrimary(&g.asm)
	}
	return true
}

func renvoWasm32StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdi, offset)
		return
	}
	if reg == 1 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRsi, offset)
		return
	}
	if reg == 2 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdx, offset)
		return
	}
	if reg == 3 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRcx, offset)
		return
	}
	if reg == 4 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegR8, offset)
		return
	}
	if reg == 5 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegR9, offset)
		return
	}
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRax)
	renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRax, offset)
}

func renvoWasm32MarkFunc(g *renvoLinearGen, fnIndex int) {
	if fnIndex < 0 || fnIndex >= len(g.funcReachable) {
		return
	}
	if g.funcReachable[fnIndex] {
		return
	}
	g.funcReachable[fnIndex] = true
	g.funcQueue = append(g.funcQueue, fnIndex)
	src := g.meta.prog.src
	nameStart := g.meta.funcs[fnIndex].nameStart
	nameEnd := g.meta.funcs[fnIndex].nameEnd
	renvoAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func renvoWasm32RangesOverlap(aOffset int, aSize int, bOffset int, bSize int) bool {
	aStart := aOffset - aSize
	bStart := bOffset - bSize
	return aStart < bOffset && bStart < aOffset
}

func renvoWasm32RecordDirectLocals(g *renvoLinearGen, functionPC int) {
	a := &g.asm
	candidates := make([]int, 0, g.localCount)
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		if local.size != renvoBackendValueSlotSize || local.captureOff != 0 || renvoTypeSize(g.meta, local.typ) > g.c.renvoNativeIntSize {
			continue
		}
		found := false
		for j := 0; j < len(candidates); j++ {
			if candidates[j] == local.offset {
				found = true
				break
			}
		}
		if !found {
			candidates = append(candidates, local.offset)
		}
	}
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		direct := local.size == renvoBackendValueSlotSize && local.captureOff == 0 && renvoTypeSize(g.meta, local.typ) <= g.c.renvoNativeIntSize
		for j := 0; j < len(candidates); j++ {
			candidate := candidates[j]
			if candidate != 0 && renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, local.offset, local.size) && (!direct || candidate != local.offset) {
				candidates[j] = 0
			}
			if candidate != 0 && local.captureOff > 0 && renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, local.captureOff, renvoBackendValueSlotSize) {
				candidates[j] = 0
			}
		}
	}
	for pc := functionPC; pc < len(a.code); pc += int(renvoWasm32InstructionSizes[int(renvo_runtime_UnsafeByteAt(a.code, pc))]) {
		op := int(renvo_runtime_UnsafeByteAt(a.code, pc))
		// Wide operations read and write whole frame-backed slots. Stack slots
		// are reused across expression lifetimes, so a scalar local at the same
		// offset cannot be proven coherent with those accesses. Keep the routine
		// frame-backed when either operation is present.
		if op == renvoWasm32OpWideBinary || op == renvoWasm32OpWideCompare {
			for j := 0; j < len(candidates); j++ {
				candidates[j] = 0
			}
			continue
		}
		memoryOffsets := make([]int, 0, 3)
		memorySizes := make([]int, 0, 3)
		if op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack {
			memoryOffsets = append(memoryOffsets, renvoWasm32GetS32(a.code, pc+2))
			memorySizes = append(memorySizes, g.c.renvoNativeIntSize)
		} else if op == renvoWasm32OpLeaStack {
			memoryOffsets = append(memoryOffsets, renvoWasm32GetS32(a.code, pc+2))
			memorySizes = append(memorySizes, renvoBackendValueSlotSize)
		}
		for k := 0; k < len(memoryOffsets); k++ {
			for j := 0; j < len(candidates); j++ {
				candidate := candidates[j]
				if candidate == 0 {
					continue
				}
				directScalarAccess := (op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack) && memoryOffsets[k] == candidate
				if !directScalarAccess {
					if renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, memoryOffsets[k], memorySizes[k]) {
						candidates[j] = 0
					}
				}
			}
		}
	}
	recordStart := len(a.wasmLocalSlots)
	a.wasmLocalSlots = append(a.wasmLocalSlots, int32(functionPC), 0)
	for i := 0; i < len(candidates); i++ {
		if candidates[i] != 0 {
			a.wasmLocalSlots = append(a.wasmLocalSlots, int32(candidates[i]))
		}
	}
	a.wasmLocalSlots[recordStart+1] = int32(len(a.wasmLocalSlots) - recordStart - 2)
}

func renvoWasm32EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.deferHeadOffset = 0
	g.deferReturnLabel = 0
	g.deferResultOffset = 0
	g.deferSites = nil
	g.emittingDefers = false
	g.suppressPanicCheck = false
	g.stackUsed = 0
	g.stackPeak = 0
	g.lastRangeReturns = false
	functionPC := len(a.code)
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdi, g.returnStruct)
	}
	renvoBindFunctionParams(g, fnInfoIndex)
	if !renvoBindClosureCaptures(g, fnInfoIndex) {
		return false
	}
	if !renvoBindNamedResults(g, fnInfoIndex) {
		return false
	}
	if !renvoPrepareFunctionControl(g) {
		return false
	}
	if !renvoEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if g.deferReturnLabel > 0 {
		if !g.lastRangeReturns {
			renvoAsmJmpLabel(a, g.deferReturnLabel)
		}
		if !renvoEmitFunctionControlEpilogue(g) {
			return false
		}
	} else if !g.lastRangeReturns {
		renvoMoveCapturedLocals(g, true)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmLeave(a)
		renvoAsmRet(a)
	}
	renvoWasm32RecordDirectLocals(g, functionPC)
	return true
}

func renvoWasm32EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	renvoWasm32MarkFunc(g, fnIndex)
	if wordCount > 0 {
		renvoWasm32AsmPopRdi(a)
	}
	if wordCount > 1 {
		renvoWasm32AsmPopRsi(a)
	}
	if wordCount > 2 {
		renvoWasm32AsmPopRdx(a)
	}
	if wordCount > 3 {
		renvoWasm32AsmPopRcx(a)
	}
	if wordCount > 4 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR8)
	}
	if wordCount > 5 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR9)
	}
	renvoWasm32EmitCallLabel(a, g.funcLabels[fnIndex], wordCount)
}

func renvoWasm32EmitRaxRcxOp(g *renvoLinearGen, tok int, unsigned bool) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		renvoWasm32AsmAddRaxRcx(a)
		return true
	}
	if c0 == '-' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRcx)
		renvoWasm32EmitRegReg(a, renvoWasm32OpSubRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		return true
	}
	if c0 == '*' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '/' {
		renvoWasm32AsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		renvoWasm32AsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpAndNotRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		} else {
			renvoWasm32EmitRegReg(a, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		}
		return true
	}
	if c0 == '|' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '^' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
			renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		} else if c1 == '=' {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9e)
		} else {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
			opcode := renvoWasm32OpShrRegReg
			if unsigned {
				opcode = renvoWasm32OpShrUnsignedRegReg
			}
			renvoWasm32EmitRegReg(a, opcode, renvoWasm32RegRax, renvoWasm32RegRdx)
		} else if c1 == '=' {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9d)
		} else {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoWasm32AsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoWasm32AsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func renvoWasm32EmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoTokCharIs(p, e.tok, '*') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmSarPrimaryImm(a, 2)
		return true
	}
	if renvoTokCharIs(p, e.tok, '/') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmShlPrimaryImm(a, 2)
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
		return false
	}
	renvoAsmPopTertiary(a)
	return renvoEmitPrimaryTertiaryOp(g, e.tok)
}

func renvoWasm32EnsureAppendAddrHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := renvoEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.appendAddrLabel)
	noGrowLabel := renvoAsmNewLabel(a)
	capNonZeroLabel := renvoAsmNewLabel(a)
	capReadyLabel := renvoAsmNewLabel(a)
	copyLoopLabel := renvoAsmNewLabel(a)
	copyDoneLabel := renvoAsmNewLabel(a)
	returnLabel := renvoAsmNewLabel(a)
	ptrSlotOff := g.asm.bssSize
	lenSlotOff := ptrSlotOff + 4
	capSlotOff := lenSlotOff + 4
	elemSizeOff := capSlotOff + 4
	oldLenOff := elemSizeOff + 4
	oldPtrOff := oldLenOff + 4
	newCapOff := oldPtrOff + 4
	allocSizeOff := newCapOff + 4
	copySizeOff := allocSizeOff + 4
	destOff := copySizeOff + 4
	copyIndexOff := destOff + 4
	g.asm.bssSize += 44

	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR8, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegR9, 0, 4)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegR8, renvoWasm32RegRcx)
	renvoWasm32EmitCondBranch(a, renvoWasm32CondLt, noGrowLabel)

	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	renvoAsmStorePrimaryBss(a, elemSizeOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdi)
	renvoAsmStorePrimaryBss(a, ptrSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRsi)
	renvoAsmStorePrimaryBss(a, lenSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegR9)
	renvoAsmStorePrimaryBss(a, capSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegR8)
	renvoAsmStorePrimaryBss(a, oldLenOff)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoAsmStorePrimaryBss(a, oldPtrOff)

	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, capNonZeroLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoWasm32EmitRegReg(a, renvoWasm32OpAddRegReg, renvoWasm32RegRcx, renvoWasm32RegR8)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryBss(a, newCapOff)
	renvoAsmLoadPrimaryBss(a, elemSizeOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, newCapOff)
	renvoAsmPopTertiary(a)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryBss(a, allocSizeOff)

	renvoAsmLoadPrimaryBss(a, allocSizeOff)
	renvoAsmCallLabel(a, arenaAllocLabel)
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmJnzPrimary(a, allocOKLabel)
		renvoAsmJmpLabel(a, returnLabel)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
	renvoAsmStorePrimaryBss(a, destOff)

	renvoAsmLoadPrimaryBss(a, oldLenOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, elemSizeOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	renvoAsmStorePrimaryBss(a, copySizeOff)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryBss(a, copyIndexOff)
	renvoAsmMarkLabel(a, copyLoopLabel)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, copySizeOff)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, copyDoneLabel)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, oldPtrOff)
	renvoAsmPopTertiary(a)
	renvoAsmLoadBytePrimaryIndexTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmIncPrimary(a)
	renvoAsmStorePrimaryBss(a, copyIndexOff)
	renvoAsmJmpMarkLabel(a, copyLoopLabel, copyDoneLabel)

	renvoAsmLoadPrimaryBss(a, ptrSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, capSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, newCapOff)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, lenSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, oldLenOff)
	renvoAsmIncPrimary(a)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, copySizeOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmJmpLabel(a, returnLabel)

	renvoAsmMarkLabel(a, noGrowLabel)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRcx, renvoWasm32RegRdx)
	renvoWasm32AsmAddRaxRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmMarkLabel(a, returnLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvoWasm32EnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRdx, renvoWasm32RegRax, renvoWasm32RegRcx, 1, 0, 1)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvoWasm32EnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRdx, renvoWasm32RegRax, renvoWasm32RegRcx, 8, 0, 4)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvoWasm32EnsureStringEqualHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegRsi, renvoWasm32RegRcx)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpCmpRegImm, renvoWasm32RegRsi, 0)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR8, renvoWasm32RegRdi, 0, 1)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR9, renvoWasm32RegRdx, 0, 1)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegR8, renvoWasm32RegR9)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRdi, 1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRdx, 1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRsi, -1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpCmpRegImm, renvoWasm32RegRsi, 0)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
