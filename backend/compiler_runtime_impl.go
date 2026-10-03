package main

func renvoReadAll(fd int, out []byte) []byte {
	var buf []byte
	for {
		base := len(out)
		if renvoFixedTarget == renvoTargetWasiWasm32 &&
			base == cap(out) && base >= 262144 {
			newCapacity := base + base/2 + 262144
			expanded := make([]byte, base, newCapacity)
			copy(expanded, out)
			out = expanded
		}
		if base < cap(out) {
			expanded := out
			renvoTruncBytes(&expanded, cap(out))
			n := read(fd, expanded[base:], -1)
			if n <= 0 {
				return out
			}
			out = expanded
			renvoTruncBytes(&out, base+n)
			continue
		}
		if len(buf) == 0 {
			buf = make([]byte, 0, 1024)
			renvoTruncBytes(&buf, cap(buf))
		}
		n := read(fd, buf, -1)
		if n <= 0 {
			return out
		}
		// buf has independent fixed backing, so a scalar append loop avoids the
		// overlap machinery required by the general append(dst, src...) lowering.
		for i := 0; i < n; i++ {
			out = append(out, buf[i])
		}
	}
}

func renvoEmitLinearPrintStmt(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.exprStart < 0 || stmt.exprStart >= renvoTokCount(p) {
		return false
	}
	ep := renvoNewExprParse()
	renvoParseExpressionInto(ep, p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != renvoExprCall {
		return false
	}
	builtinPrintln := renvoExprIdentCode(p, ep, root.left) == renvoIdentPrintln
	println := builtinPrintln || renvoExprIsIdentText(p, ep, root.left, "Println")
	fmtPrintln := false
	if !println && !renvoExprIsIdentText(p, ep, root.left, "print") {
		candidate := &ep.exprs[root.left]
		if candidate.kind != renvoExprIdent || candidate.nameEnd-candidate.nameStart != 24 || !renvoBytesEqualText(p.src, candidate.nameStart, candidate.nameEnd, "renvo_runtime_FmtPrintln") {
			return false
		}
		fmtPrintln = true
		println = true
	}
	fnIndex := renvoFuncInfoFromCall(g, ep, root.left)
	if fmtPrintln && fnIndex < 0 {
		return false
	}
	callee := &ep.exprs[root.left]
	if !fmtPrintln && (fnIndex >= 0 || renvoFindLocalIndex(g, callee.nameStart, callee.nameEnd) >= 0) {
		return false
	}
	// More than one argument must be evaluated completely before Println starts
	// writing. Keep those calls on the generic path until the backend has a
	// compact multi-value staging representation.
	if fmtPrintln && root.argCount > 1 {
		return false
	}
	fd := 1
	if builtinPrintln {
		fd = 2
	}
	for i := 0; i < root.argCount; i++ {
		if println && i > 0 && !renvoEmitPrintStaticByte(g, ' ', fd) {
			return false
		}
		argIndex := ep.args[root.firstArg+i]
		argType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
		renvoNonNil(argType)
		if fmtPrintln && argType.kind != renvoTypeString {
			return false
		}
		if argType.kind == renvoTypeString {
			if !renvoEmitStringValueRegs(g, ep, argIndex) {
				return false
			}
		} else if renvoTypeKindIsScalarInt(argType.kind) {
			if !renvoEmitIntExpr(g, ep, argIndex) {
				return false
			}
			renvoAsmNormalizePrimaryForKind(&g.asm, argType.kind)
			renvoAsmCallLabel(&g.asm, renvoEnsurePrintIntHelper(g))
		} else {
			return false
		}
		if !renvoEmitWriteValueRegs(g, fd) {
			return false
		}
	}
	return !println || renvoEmitPrintStaticByte(g, '\n', fd)
}

func renvoEmitPrintStaticByte(g *renvoLinearGen, value byte, fd int) bool {
	renvoNonNil(g)
	offset := len(g.asm.data)
	g.asm.data = append(g.asm.data, value)
	renvoAsmPrimaryDataAddr(&g.asm, offset)
	renvoAsmSecondaryImm(&g.asm, 1)
	return renvoEmitWriteValueRegs(g, fd)
}

func renvoPrintMirrorFunction(g *renvoLinearGen) int {
	renvoNonNil(g)
	for i := 0; i < len(g.meta.funcs); i++ {
		fn := &g.meta.funcs[i]
		if fn.receiverType == 0 && fn.paramCount == 1 &&
			renvoTypeIsString(g.meta, g.meta.params[fn.firstParam].typ) &&
			renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_PrintMirror") {
			return i
		}
	}
	return -1
}

func renvoEmitPrintMirror(g *renvoLinearGen) {
	renvoNonNil(g)
	fnIndex := renvoPrintMirrorFunction(g)
	if fnIndex < 0 || fnIndex == g.currentFunc {
		return
	}
	// Preserve the pointer and length for the real stdout write below while a
	// second pair becomes the mirror function's string argument.
	renvoAsmPushSecondary(&g.asm)
	renvoAsmPushPrimary(&g.asm)
	renvoAsmPushSecondary(&g.asm)
	renvoAsmPushPrimary(&g.asm)
	renvoEmitCallWithWordCount(g, fnIndex, renvoBackendStringWordCount)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmPopSecondary(&g.asm)
}

func renvoEmitWriteValueRegs(g *renvoLinearGen, fd int) bool {
	renvoEmitPrintMirror(g)
	return renvoEmitTargetWriteValueRegs(g, fd)
}

func renvoEmitBuiltinReadWrite(g *renvoLinearGen, ep *renvoExprParse, idx int, operation int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	if renvoPreparedBackendActive != 0 {
		if !renvoEmitIntExpr(g, ep, ep.args[firstArg]) {
			return false
		}
	} else {
		fdStart := ep.exprs[idx].tok + 1
		fdEnd := renvoFindExprBoundary(p, fdStart, ep.end)
		fdEp := renvoNewExprParse()
		renvoParseExpressionInto(fdEp, p, fdStart, fdEnd)
		if !fdEp.ok || len(fdEp.exprs) == 0 {
			return false
		}
		fdIndex := len(fdEp.exprs) - 1
		if !renvoEmitIntExpr(g, fdEp, fdIndex) {
			return false
		}
	}
	renvoAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := renvoEvalConstExpr(g, ep, offIndex)
	offsetRead := true
	if offConst.ok && offConst.value < 0 {
		offsetRead = false
	}
	if offsetRead {
		if offConst.ok {
			renvoAsmPrimaryImm(a, offConst.value)
		} else {
			if !renvoEmitIntExpr(g, ep, offIndex) {
				return false
			}
		}
		renvoAsmPushPrimary(a)
	}
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	renvoAsmPrepareReadWriteBuf(a)
	if offsetRead {
		renvoAsmPopReadWriteOffset(a)
	}
	renvoAsmPopCallWord0(a)
	return renvoAsmReadWriteFile(a, operation, offsetRead)
}

func renvoEvalBuiltinConst(g *renvoLinearGen, nameStart int, nameEnd int) renvoConstResult {
	renvoNonNil(g)
	p := g.prog
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "iota") {
		if g.constEvalIotaValid != 0 {
			return renvoConstResultOk(g.constEvalIota)
		}
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "nil") {
		return renvoConstResultOk(0)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_RDONLY") {
		return renvoConstResultOk(0)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_WRONLY") {
		return renvoConstResultOk(1)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_RDWR") {
		return renvoConstResultOk(2)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_CREATE") {
		if targetIsDarwin(g.c.renvoTargetOS) || targetIsBSD(g.c.renvoTargetOS) {
			return renvoConstResultOk(512)
		}
		return renvoConstResultOk(64)
	}
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "O_TRUNC") {
		if targetIsDarwin(g.c.renvoTargetOS) || targetIsBSD(g.c.renvoTargetOS) {
			return renvoConstResultOk(1024)
		}
		return renvoConstResultOk(512)
	}
	var r renvoConstResult
	return r
}

// renvoEmitOpenFileCall evaluates flags before the path, preserving the
// runtime builtin's evaluation order. Definitions own the path representation
// and consume the primary path plus the saved flags at the call boundary.
func renvoEmitOpenFileCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	pathIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	if renvoOpenPathNeedsLength(g) {
		if !renvoEmitStringValueRegs(g, ep, pathIndex) {
			return false
		}
	} else if !renvoEmitStringPtrExpr(g, ep, pathIndex) {
		return false
	}
	return renvoAsmOpenFile(&g.asm)
}

// renvoEmitDescriptorFileCall shares argument validation and ordered evaluation
// for close and chmod. The definition consumes the descriptor in call word zero
// and, for chmod, the mode in call word one.
func renvoEmitDescriptorFileCall(g *renvoLinearGen, ep *renvoExprParse, idx int, callee int) bool {
	e := &ep.exprs[idx]
	a := &g.asm
	if callee == renvoIdentClose {
		if e.argCount != 1 {
			return false
		}
	} else if callee != renvoIdentChmod || e.argCount != 2 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	if callee == renvoIdentClose {
		renvoAsmCopyPrimaryToCallWord0(a)
		return renvoAsmCloseFile(a)
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmPopCallWord0(a)
	return renvoAsmChmodFile(a)
}

func renvoEmitTargetRuntime(g *renvoLinearGen, ep *renvoExprParse, idx int, callee int) bool {
	renvoNonNil(g, ep)
	if renvoPreparedBackendActive != 0 {
		return renvoEmitPreparedTargetRuntime(g, ep, idx, callee)
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		if callee == renvoIdentRead || callee == renvoIdentWrite {
			return renvoEmitWindowsReadWrite(g, ep, idx, callee == renvoIdentWrite)
		}
		if callee == renvoIdentOpen {
			return renvoEmitWindowsOpen(g, ep, idx)
		}
		if callee == renvoIdentClose {
			return renvoEmitWindowsClose(g, ep, idx)
		}
		return renvoEmitWindowsChmod(g, ep, idx)
	}
	if callee == renvoIdentRead || callee == renvoIdentWrite {
		operation := RTGRuntimeRead
		if callee == renvoIdentWrite {
			operation = RTGRuntimeWrite
		}
		return renvoEmitBuiltinReadWrite(g, ep, idx, operation)
	}
	if callee == renvoIdentOpen {
		return renvoEmitOpenFileCall(g, ep, idx)
	}
	return renvoEmitDescriptorFileCall(g, ep, idx, callee)
}

func renvoEmitPreparedTargetRuntime(
	g *renvoLinearGen, ep *renvoExprParse, idx int, callee int,
) bool {
	renvoNonNil(g, ep)
	if callee == renvoIdentRead || callee == renvoIdentWrite {
		operation := RTGRuntimeRead
		if callee == renvoIdentWrite {
			operation = RTGRuntimeWrite
		}
		return renvoEmitBuiltinReadWrite(g, ep, idx, operation)
	}
	if callee == renvoIdentOpen {
		return renvoEmitOpenFileCall(g, ep, idx)
	}
	return renvoEmitDescriptorFileCall(g, ep, idx, callee)
}
func renvoEmitExitStatus(g *renvoLinearGen) bool {
	return renvoAsmExitStatus(&g.asm)
}

func renvoEmitLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if renvoFixedTarget == 0 && renvoIsCdeclObject(g.c) {
		importID := renvoAsmAddExternalImportRange(&g.asm,
			g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
		if importID < 0 {
			return false
		}
		variadic := fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart != 0
		return renvoAsmCdeclObjectCall(&g.asm, importID, wordCount, variadic)
	}
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		memoryAggregate := renvoEmitCObjectMemoryAggregateCall(g, fn, wordCount)
		if memoryAggregate >= 0 {
			return memoryAggregate != 0
		}
		vectorMask := renvoObjectCallVectorMask(g, fn, wordCount)
		if vectorMask < 0 {
			return false
		}
		importID := renvoAsmAddPreparedStaticImport(&g.asm,
			fn.linkDLLStart, fn.linkDLLEnd,
			fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
		if importID < 0 {
			return false
		}
		return renvoAsmObjectRegisterCall(&g.asm, importID, wordCount, vectorMask)
	}
	policy := renvoTargetStaticCallPolicy(g.c)
	if policy == renvoStaticCallUnavailable {
		return false
	}
	if policy == renvoStaticCallSplitRegisters && !renvoPrepareStaticCallShape(g, fn, wordCount) {
		return false
	}
	importID := renvoAsmAddLinkedStaticImport(&g.asm,
		fn.linkDLLStart, fn.linkDLLEnd, fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if importID < 0 {
		return false
	}
	return renvoAsmHostedStaticCall(&g.asm, importID, wordCount)
}

// Static-call policy selects a bounded argument protocol, never an OS/ISA.
const (
	renvoStaticCallUnavailable = iota
	renvoStaticCallWords
	renvoStaticCallSplitRegisters
)

// Semantic kinds in the static-call transport. Register assignment and result
// register encoding belong to the selected definition, not this classifier.
const (
	renvoStaticCallInteger = iota
	renvoStaticCallString
	renvoStaticCallSlice
	renvoStaticCallFloat32
	renvoStaticCallFloat64
	renvoStaticCallIntegerFloat64
	renvoStaticCallIntegerFloat32
)

func renvoPrepareStaticCallShape(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	var kinds [16]byte
	consumed := 0
	integerCount := 0
	floatCount := 0
	allInteger := true
	abi := renvoLinkStaticOption(g.prog.src, fn.linkMethodEnd, fn.nameStart, "float64") |
		renvoLinkStaticOption(g.prog.src, fn.linkMethodEnd, fn.nameStart, "float32")<<8
	for i := 0; i < fn.paramCount; i++ {
		typ := renvoResolveType(g.meta, g.meta.params[fn.firstParam+i].typ)
		kind := renvoStaticCallInteger
		if abi&(1<<(i+8)) != 0 {
			kind = renvoStaticCallIntegerFloat32
		} else if abi&(1<<i) != 0 {
			kind = renvoStaticCallIntegerFloat64
		} else if typ.kind == renvoTypeFloat32 {
			kind = renvoStaticCallFloat32
		} else if typ.kind == renvoTypeFloat64 {
			kind = renvoStaticCallFloat64
		} else if typ.kind == renvoTypeString {
			kind = renvoStaticCallString
		} else if typ.kind == renvoTypeSlice {
			kind = renvoStaticCallSlice
		} else if typ.kind == renvoTypeStruct || typ.kind == renvoTypeArray {
			return false
		}
		if kind == renvoStaticCallFloat32 || kind == renvoStaticCallFloat64 ||
			kind == renvoStaticCallIntegerFloat64 || kind == renvoStaticCallIntegerFloat32 {
			floatCount++
			allInteger = false
		} else {
			integerCount++
			if kind != renvoStaticCallInteger {
				allInteger = false
			}
		}
		if i >= len(kinds) && kind != renvoStaticCallInteger {
			return false
		}
		if i < len(kinds) {
			kinds[i] = byte(kind)
		}
		consumed++
		if kind == renvoStaticCallString {
			consumed++
		} else if kind == renvoStaticCallSlice {
			consumed += 2
		}
	}
	if consumed != wordCount {
		return false
	}
	resultFloat := -1
	resultKind := renvoStaticCallInteger
	typ := renvoResolveType(g.meta, fn.resultType)
	if renvoTypeKindIsFloat(typ.kind) {
		resultFloat = renvoLinkStaticOption(
			g.prog.src, fn.linkMethodEnd, fn.nameStart, "result-float64")
		if typ.kind == renvoTypeFloat32 {
			resultKind = renvoStaticCallFloat32
		} else {
			resultKind = renvoStaticCallFloat64
		}
	}
	if !renvoAsmFinishStaticCallShape(&g.asm, integerCount, floatCount, allInteger, resultFloat, resultKind) {
		return false
	}
	g.asm.staticCallParamCount = fn.paramCount
	g.asm.staticCallParamKinds = kinds
	return true
}

func renvoLinkStaticOption(src []byte, start int, end int, name string) int {
	for start < end && renvo_runtime_UnsafeByteAt(src, start) != '\n' {
		if renvo_runtime_UnsafeByteAt(src, start) != ',' {
			start++
			continue
		}
		start++
		for start < end && renvo_runtime_UnsafeByteAt(src, start) == ' ' {
			start++
		}
		if start+len(name) < end && renvoBytesEqualText(src, start, start+len(name), name) &&
			renvo_runtime_UnsafeByteAt(src, start+len(name)) == '=' {
			value := 0
			for start += len(name) + 1; start < end; start++ {
				ch := renvo_runtime_UnsafeByteAt(src, start)
				if ch < '0' || ch > '9' {
					return value
				}
				value = value*10 + int(ch-'0')
			}
		}
	}
	return 0
}

// renvoEmitCObjectMemoryAggregateCall handles foreign calls whose SysV stack
// arguments include an aggregate. The ordinary object-call path maps scalar
// and small aggregate words directly to registers; once stack arguments are
// present, an aggregate must be classified as a unit while later scalar
// parameters can continue consuming argument registers.
func renvoEmitCObjectMemoryAggregateCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) int {
	if !renvoIsSysVObject(g.c) || wordCount < 1 {
		return -1
	}
	containsAggregate := false
	for i := 0; i < fn.paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		containsAggregate = containsAggregate || renvoResolveType(g.meta, paramType).kind == renvoTypeStruct
	}
	if !containsAggregate {
		return -1
	}
	wordBytes := g.c.renvoNativeIntSize
	registerCount := renvoObjectArgumentRegisterCount(g.c)
	aggregateRegisterBytes := renvoObjectAggregateRegisterBytes(g.c)
	if wordBytes <= 0 || registerCount <= 0 || aggregateRegisterBytes <= 0 {
		return 0
	}
	integerRegisters := 0
	stackBytes := 0
	var locations []int
	for i := 0; i < fn.paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		param := renvoResolveType(g.meta, paramType)
		renvoNonNil(param)
		words := 1
		memory := false
		if param.kind == renvoTypeStruct {
			size := renvoTypeSize(g.meta, paramType)
			words = renvoAlignValue(size, wordBytes) / wordBytes
			if size <= aggregateRegisterBytes && !renvoObjectCABIIntegerAggregate(g.meta, paramType) {
				return 0
			}
			memory = size > aggregateRegisterBytes || integerRegisters+words > registerCount
		} else {
			if !renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc && !renvoTypeIsString(g.meta, paramType) {
				return 0
			}
			memory = integerRegisters >= registerCount
		}
		if words < 0 || words > 32-len(locations) {
			return 0
		}
		for word := 0; word < words; word++ {
			if memory {
				locations = append(locations, -stackBytes-1)
				stackBytes += wordBytes
			} else {
				locations = append(locations, integerRegisters)
				integerRegisters++
			}
		}
	}
	if stackBytes == 0 {
		return -1
	}
	if len(locations) != wordCount || wordCount > 32 {
		return 0
	}
	a := &g.asm
	importID := renvoAsmAddPreparedStaticImport(a,
		fn.linkDLLStart, fn.linkDLLEnd, fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if importID < 0 {
		return 0
	}
	// Classification is shared; the target consumes this plan synchronously
	// and owns stack alignment, register placement, relocation, and cleanup.
	a.staticCallWordLocations = locations
	a.staticCallStackBytes = stackBytes
	ok := renvoAsmObjectRegisterCall(a, importID, wordCount, 0)
	a.staticCallWordLocations = nil
	a.staticCallStackBytes = 0
	return renvoBoolInt(ok)
}

func renvoObjectCallVectorMask(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) int {
	if wordCount < 0 {
		return -1
	}
	mask := 0
	integers := 0
	vectors := 0
	word := 0
	for i := 0; i < fn.paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		typ := renvoResolveType(g.meta, paramType)
		if typ.kind == renvoTypeFloat64 {
			mask |= 1 << word
			vectors++
			word++
		} else if typ.kind == renvoTypeStruct {
			if !renvoObjectCABIIntegerAggregate(g.meta, paramType) {
				return -1
			}
			words := renvoAlignValue(renvoTypeSize(g.meta, paramType), 8) / 8
			integers += words
			word += words
		} else {
			integers++
			word++
		}
	}
	if word != wordCount || integers > 6 || vectors > 8 {
		return -1
	}
	return mask
}

// renvoEmitTargetStaticCall returns -1 when the declaration is not a foreign
// call for the active target, zero for an invalid target binding, and one when
// the call was emitted.
func renvoEmitTargetStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) int {
	renvoNonNil(g, fn)
	if g.c.objectFile {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoPreparedBackendActive != 0 && renvoRTGPreparedObject != 0 {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoFixedTarget == renvoTargetLinuxKernelAmd64 ||
		renvoPreparedBackendActive == 0 && renvoFixedTarget == 0 && targetIsKernelModule(g.c) {
		if !renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "kernel") {
			return 0
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if renvoPreparedBackendActive != 0 {
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		if renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) != '/' {
			return -1
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	if targetIsWindows(g.c.renvoTargetOS) {
		if renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) == '/' {
			return -1
		}
		if renvoEmitLinkStaticCall(g, fn, wordCount) {
			return 1
		}
		return 0
	}
	return -1
}

func renvoAsmAddPreparedStaticImport(
	a *renvoAsm, libraryStart int, libraryEnd int,
	nameStart int, nameEnd int, src []byte,
) int {
	renvoNonNil(a)
	library := renvoStringFromBytes(src, libraryStart, libraryEnd)
	name := renvoStringFromBytes(src, nameStart, nameEnd)
	for i := 0; i < len(a.staticImports); i++ {
		if a.staticImports[i].dll == library && a.staticImports[i].name == name {
			return i
		}
	}
	a.staticImports = append(a.staticImports, renvoStaticImport{dll: library, name: name})
	return len(a.staticImports) - 1
}
func renvoEmitRuntimeArenaDiscard(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if !renvoArenaDiscardSupported(g.c) {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	endOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmStorePrimaryStack(&g.asm, startOff)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmStorePrimaryStack(&g.asm, endOff)
	return renvoEmitRuntimeArenaDiscardStackRange(g, startOff, endOff)
}

func renvoEmitRuntimeArenaDiscardSlice(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoArenaDiscardSupported(g.c) {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	sliceType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, sliceType.elem)
	if !renvoEmitSlicePtrLen(g, ep, argIndex) {
		return false
	}
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	endOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(&g.asm, startOff)
	renvoAsmMulTertiaryImm(&g.asm, elemSize)
	renvoAsmCopyTertiaryToPrimary(&g.asm)
	renvoAsmLoadTertiaryStack(&g.asm, startOff)
	renvoAsmAddPrimaryTertiary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, endOff)
	return renvoEmitRuntimeArenaDiscardStackRange(g, startOff, endOff)
}

func renvoEmitRuntimeArenaDiscardStackRange(g *renvoLinearGen, startOff int, endOff int) bool {
	if !renvoArenaDiscardSupported(g.c) {
		return true
	}
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmDiscardArenaPages(&g.asm, startOff, endOff, lenOff)
	renvoAsmPrimaryImm(&g.asm, 0)
	return true
}

func renvoEmitRuntimeArenaPersistReset(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoStringHeapOffsets(g)
	renvoEmitArenaRememberReset(g, true)
	a := &g.asm
	if renvoArenaDiscardSupported(g.c) {
		renvoEmitRuntimeArenaPersistResetMadvise(g)
		return true
	}
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	return true
}

func renvoEmitRuntimeArenaPersistResetMadvise(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	markOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	oldOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(a, markOff)
	renvoAsmCopyBssToStackSlot(a, g.stringHeapEndOff, oldOff)
	renvoAsmLoadPrimaryStack(a, markOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	renvoAsmDiscardArenaPages(a, oldOff, markOff, lenOff)
}

func renvoEmitRuntimeArenaPersistString(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	a := &g.asm
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoEmitPersistentAllocToPrimary(g, lenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyBytesToPersistent(g, srcOff, lenOff, destOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	return true
}

func renvoEmitRuntimeArenaPersistBytes(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	return renvoEmitRuntimeArenaPersistSlice(g, ep, idx)
}

func renvoEmitRuntimeArenaPersistSlice(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	sliceType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
	if sliceType.kind != renvoTypeSlice || !renvoEmitSliceValueRegs(g, ep, argIndex) {
		return false
	}
	a := &g.asm
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	byteLenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmMulTertiaryImm(a, renvoTypeSize(g.meta, sliceType.elem))
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, byteLenOff)
	renvoEmitPersistentAllocToPrimary(g, byteLenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyBytesToPersistent(g, srcOff, byteLenOff, destOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	renvoAsmLoadTertiaryStack(a, lenOff)
	return true
}

func renvoEmitCopyBytesToPersistent(g *renvoLinearGen, srcOff int, lenOff int, destOff int) {
	renvoNonNil(g)
	renvoEmitCopyBytes(g, srcOff, destOff, lenOff)
}

func renvoEmitPersistentAllocToPrimary(g *renvoLinearGen, sizeOff int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, sizeOff)
	renvoAsmCallLabel(a, renvoEnsureDirectionalArenaAllocHelper(g, true))
	renvoEmitArenaAllocationCheck(g)
}

func renvoEmitBuiltinNew(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprCall || e.argCount != 1 {
		return false
	}
	targetType := renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	if targetType == 0 {
		return false
	}
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, renvoTypeSize(g.meta, targetType))
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmLoadTertiaryStack(&g.asm, sizeOffset)
	renvoAsmCallLabel(&g.asm, renvoEnsureMakeZeroHelper(g))
	return true
}

func renvoEmitPersistentArenaReady(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	renvoStringHeapOffsets(g)
	readyLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	renvoAsmJnzPrimary(a, readyLabel)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	renvoAsmPushImm(a, renvoStringArenaSize(g))
	renvoAsmPopTertiary(a)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	lowReadyLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
	renvoAsmJnzPrimary(a, lowReadyLabel)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapOff)
	renvoAsmMarkLabel(a, lowReadyLabel)
	renvoAsmMarkLabel(a, readyLabel)
}

func renvoEmitArbitrarySyscall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount < 1 || e.argCount > 7 {
		return false
	}
	if targetIsDarwin(g.c.renvoTargetOS) {
		if e.argCount != 4 {
			return false
		}
		number := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		// The Darwin directory adapter uses one compiler-intrinsic selector,
		// which is lowered to libc getdirentries rather than issued as a raw
		// Darwin syscall number.
		if !number.ok || number.value != 217 {
			return false
		}
	}
	syscallNumber := -1
	if renvoFixedTarget == renvoTargetOpenBSDAmd64 ||
		renvoFixedTarget == 0 && g.c.renvoTargetOS == renvoOSOpenBSD {
		number := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		if !number.ok {
			return false
		}
		syscallNumber = number.value
	}
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
		if !renvoEmitSyscallArg(g, ep, argIndex) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
	}
	return renvoEmitSyscallFromStack(g, e.argCount, syscallNumber)
}

// renvo_runtime_CallJIT is a frontend-only escape hatch used by the bundled
// runner. It switches to an isolated native stack and calls a linked-image
// entry directly. Keeping this as one compiler intrinsic avoids exposing
// arbitrary code pointers as ordinary Go function values.
func renvoEmitJITCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 6 || g.c.renvoTargetArch == renvoArchWasm32 {
		return false
	}
	for i := e.argCount - 1; i >= 0; i-- {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
	}
	return renvoAsmJITCallFromStack(&g.asm)
}

func renvoEmitSyscallArg(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	typ := renvoInferParsedExprType(g, ep, idx)
	if renvoTypeIsString(g.meta, typ) {
		return renvoEmitStringPtrExpr(g, ep, idx)
	}
	if renvoTypeIsSlice(g.meta, typ) {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		return true
	}
	return renvoEmitIntExpr(g, ep, idx)
}

func renvoEmitSyscallFromStack(g *renvoLinearGen, wordCount int, syscallNumber int) bool {
	return renvoAsmSyscallFromStack(&g.asm, wordCount, syscallNumber)
}

func renvoBeginKernelModule(g *renvoLinearGen, appIndex int) bool {
	renvoNonNil(g)
	a := &g.asm
	g.kernelCallbackLabels = make([]int, len(g.meta.funcs))
	for i := 0; i < len(g.kernelCallbackLabels); i++ {
		g.kernelCallbackLabels[i] = -1
	}
	exitIndex := -1
	for i := 0; i < len(g.meta.funcs); i++ {
		if renvoBytesEqualText(g.meta.prog.src, g.meta.funcs[i].nameStart, g.meta.funcs[i].nameEnd, "moduleExit") {
			exitIndex = i
		}
	}
	g.kernelInitLabel = renvoAsmNewLabel(a)
	g.kernelExitLabel = -1
	renvoAsmMarkLabel(a, g.kernelInitLabel)
	renvoEmitKernelEntryFrame(g)
	renvoLinearMarkFunc(g, appIndex)
	renvoEmitInitializeThreadState(g)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return false
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return false
	}
	renvoAsmPrimaryImm(a, 0)
	renvoAsmKernelEntryReturn(a)
	if exitIndex >= 0 {
		g.kernelExitLabel = renvoAsmNewLabel(a)
		renvoAsmMarkLabel(a, g.kernelExitLabel)
		renvoEmitKernelEntryFrame(g)
		renvoLinearMarkFunc(g, exitIndex)
		renvoAsmCallLabel(a, g.funcLabels[exitIndex])
		renvoAsmKernelEntryReturn(a)
	}
	return true
}

func renvoEmitKernelCallbackArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, funcType int) int {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return -1
	}
	fnIndex := renvoFindMetaFunction(g.meta, e.nameStart, e.nameEnd)
	if fnIndex < 0 || renvoFunctionValueMode(g.meta, fnIndex, funcType) != renvoFunctionValueDirect {
		return -1
	}
	renvoLinearMarkFunc(g, fnIndex)
	a := &g.asm
	label := g.kernelCallbackLabels[fnIndex]
	first := label < 0
	if first {
		label = renvoAsmNewLabel(a)
		g.kernelCallbackLabels[fnIndex] = label
	}
	// The target owns the callback address relocation and entry ABI.
	renvoAsmKernelCallbackAddress(a, label)
	renvoAsmPushPrimary(a)
	if first {
		after := renvoAsmNewLabel(a)
		renvoAsmJmpLabel(a, after)
		renvoAsmMarkLabel(a, label)
		renvoEmitKernelEntryFrame(g)
		renvoAsmCallLabel(a, g.funcLabels[fnIndex])
		renvoAsmKernelEntryReturn(a)
		renvoAsmMarkLabel(a, after)
	}
	return 1
}
