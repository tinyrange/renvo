package main

const renvo386ELFCodeOffset = 0x74

func renvo386RewritePrimaryLoad(a *renvoAsm, reg int, pushed bool) bool {
	load := a.lastPrimaryLoad
	if pushed {
		load = -load
	}
	end := load >> 3
	at := end - (load & 7)
	if pushed {
		end++
	}
	if load <= 0 || end != len(a.code) {
		return false
	}
	a.code[at] += byte(reg * 8)
	if pushed {
		renvoTruncBytes(&a.code, len(a.code)-1)
	}
	a.lastPrimaryLoad = 0
	return true
}

func renvo386RewritePrimaryLoadCompare(a *renvoAsm, imm int) bool {
	load := a.lastPrimaryLoad
	end := load >> 3
	if load <= 0 || end != len(a.code) {
		return false
	}
	at := end - (load & 7)
	a.code[at-1] = 0x83
	a.code[at] += 0x38
	a.code = append(a.code, byte(imm))
	a.lastPrimaryLoad = 0
	return true
}

func renvoAsmImageObject386(emitter *renvoAsm) []byte {
	return renvoAsmImageRelocatableObject386(emitter)
}

func renvoTryCompileScalarProgram386(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	return renvoTryCompileScalarProgram386Scratch(p, meta)
}

func renvoTryCompileScalarProgram386Scratch(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram386(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsScratch(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram386(g)
}

func renvoTryCompileScalarProgram386Cached(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram386(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsCached(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram386(g)
}
func renvoBeginScalarProgram386(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	if renvoFixedTarget == 0 {
		if renvoIsHostedObject386(meta.c) {
			return renvoBeginObjectProgram(p, meta)
		}
	}
	appIndex := -1
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
	}
	if appIndex < 0 {
		return nil
	}
	g := new(renvoLinearGen)
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	a.codeOffset = renvo386ELFCodeOffset
	if targetIsWindows(meta.c.renvoTargetOS) {
		a.codeOffset = renvoWinSectionRVA
	}
	if renvoFixedTarget != 0 {
		g.funcLabels = make([]int, 0, len(meta.funcs))
	}
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(g, len(meta.funcs))
	renvoLinearMarkFunc(g, appIndex)
	renvoEmitInitializeThreadState(g)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return nil
	}
	entryOK := false
	if renvoFixedTarget == 0 && meta.c.emitImage {
		entryOK = renvoEmitImageEntryArgs386(g, appIndex)
	} else {
		entryOK = renvoEmitProgramEntryArgs386(g, appIndex)
	}
	if !entryOK {
		return nil
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return nil
	}
	if renvoFixedTarget == 0 && meta.c.emitImage {
		renvoAsmRet(a)
	} else if targetIsWindows(meta.c.renvoTargetOS) {
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportExitProcess)
		renvoAsmRet(a)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmSyscall(a)
	}
	return g
}

func renvoEmitImageEntryArgs386(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 || !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam].typ) {
		return false
	}
	// cdecl entry stack -> EBX/ESI/EDX, the first Renvo slice.
	renvoAsmEmitText(&g.asm, "\x8b\x5c\x24\x04\x8b\x74\x24\x08\x89\xf2")
	if app.paramCount == 1 {
		return true
	}
	if !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+1].typ) {
		return false
	}
	// ECX/EAX/EDI, the second Renvo slice.
	renvoAsmEmitText(&g.asm, "\x8b\x4c\x24\x0c\x8b\x44\x24\x10\x89\xc7")
	return true
}

func renvoFinishScalarProgram386(g *renvoLinearGen) renvoCompileResult {
	renvoNonNil(g)
	a := &g.asm
	if renvoFixedTarget == 0 && renvoIsHostedObject386(g.c) {
		renvoRecordObjectFunctionRanges(g)
		if g.c.code16 && !renvo386ApplyCode16(&g.asm) {
			return renvoCompileResult{}
		}
	}
	renvo_runtime_ArenaDiscard(g.meta.scratchStart, g.meta.scratchEnd)
	var data []byte
	if renvoFixedTarget == 0 && renvoIsHostedObject386(g.c) {
		data = renvoAsmImageObject386(a)
	} else if targetIsWindows(g.c.renvoTargetOS) {
		data = renvoAsmImageWindows386(a)
	} else {
		data = renvoAsmImage386(a)
	}
	var result renvoCompileResult
	if a.patchFailed || len(data) == 0 {
		return result
	}
	result.data = data
	result.ok = true
	return result
}

func renvo386ApplyCode16(a *renvoAsm) bool {
	code := renvo386RewriteCode16(a.code)
	positions := renvo386Code16Positions(a.code)
	if code == nil || len(positions) != len(a.code)+1 {
		failure := renvo386Code16FailureOffset(a.code)
		renvoPrintErr("renvo: code16 rewrite failed at byte ")
		renvoPrintIntErr(failure)
		renvoPrintErr(" near")
		for i := failure - 6; i < failure+6; i++ {
			if i >= 0 && i < len(a.code) {
				renvoPrintErr(" ")
				renvoPrintIntErr(int(a.code[i]))
			}
		}
		renvoPrintErr("\n")
		a.patchFailed = true
		return false
	}
	for i := 0; i < len(a.labelPos); i++ {
		old := int(a.labelPos[i])
		if old < 0 {
			continue
		}
		if old >= len(positions) || positions[old] < 0 {
			a.patchFailed = true
			return false
		}
		a.labelPos[i] = int32(positions[old])
	}
	for i := 0; i < len(a.relocs); i += 2 {
		old := int(a.relocs[i])
		if old < 0 || old >= len(positions) || positions[old] < 0 {
			a.patchFailed = true
			return false
		}
		a.relocs[i] = int32(positions[old])
	}
	for i := 0; i < len(a.absRelocs); i += 3 {
		old := int(a.absRelocs[i])
		if old < 0 || old >= len(positions) || positions[old] < 0 {
			a.patchFailed = true
			return false
		}
		a.absRelocs[i] = int32(positions[old])
	}
	for i := 0; i < len(a.objectFunctions); i++ {
		old := a.objectFunctions[i].end
		if old < 0 || old >= len(positions) || positions[old] < 0 {
			a.patchFailed = true
			return false
		}
		a.objectFunctions[i].end = positions[old]
		// Code16 has byte-aligned instruction targets and Linux's setup image
		// has a strict 32 KiB ceiling. Do not inherit the ordinary hosted
		// object's discretionary 16-byte function alignment.
		a.objectFunctions[i].alignment = 1
	}
	a.code = code
	return renvo386RelaxCode16NearRelocs(a)
}

func renvo386RelaxCode16NearRelocs(a *renvoAsm) bool {
	// The generic 386 emitter represents every local near transfer with a
	// rel32 field. In code16 that requires an operand-size override, although
	// setup objects are themselves comfortably inside the signed rel16 range.
	// Resolve those local labels now and retain rel32 only for transfers whose
	// distance is genuinely too large. External ELF relocations live in
	// absRelocs and are deliberately untouched.
	type nearReloc struct {
		start int
		at    int
		end   int
		label int
		cond  bool
	}
	candidates := make([]nearReloc, 0, len(a.relocs)/2)
	byStart := make([]int, len(a.code)+1)
	for i := 0; i < len(byStart); i++ {
		byStart[i] = -1
	}
	for i := 0; i+1 < len(a.relocs); i += 2 {
		at := int(a.relocs[i])
		label := int(a.relocs[i+1])
		if label < 0 || label >= len(a.labelPos) || a.labelPos[label] < 0 {
			continue
		}
		start := -1
		end := -1
		cond := false
		if at >= 2 && at+4 <= len(a.code) && a.code[at-2] == 0x66 && a.code[at-1] == 0xe9 {
			start = at - 2
			end = at + 4
		} else if at >= 3 && at+4 <= len(a.code) && a.code[at-3] == 0x66 &&
			a.code[at-2] == 0x0f && a.code[at-1]&0xf0 == 0x80 {
			start = at - 3
			end = at + 4
			cond = true
		}
		if start < 0 {
			continue
		}
		// A field that also carries an ELF relocation is not a purely local
		// transfer and must retain its original width and relocation kind.
		external := false
		for absolute := 0; absolute+2 < len(a.absRelocs); absolute += 3 {
			if int(a.absRelocs[absolute]) == at {
				external = true
				break
			}
		}
		if external {
			continue
		}
		distance := int(a.labelPos[label]) - end
		if distance < -32768 || distance > 32767 {
			continue
		}
		candidate := nearReloc{start: start, at: at, end: end, label: label, cond: cond}
		byStart[start] = len(candidates)
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return true
	}
	oldCode := a.code
	positions := make([]int, len(oldCode)+1)
	for i := 0; i < len(positions); i++ {
		positions[i] = -1
	}
	newCode := make([]byte, 0, len(oldCode)-len(candidates)*3)
	newFields := make([]int, len(candidates))
	for old := 0; old < len(oldCode); {
		positions[old] = len(newCode)
		candidateIndex := byStart[old]
		if candidateIndex < 0 {
			newCode = append(newCode, oldCode[old])
			old++
			continue
		}
		candidate := candidates[candidateIndex]
		if candidate.cond {
			newCode = append(newCode, 0x0f, oldCode[candidate.at-1])
		} else {
			newCode = append(newCode, oldCode[candidate.at-1])
		}
		newFields[candidateIndex] = len(newCode)
		newCode = append(newCode, 0, 0)
		for interior := old + 1; interior < candidate.end; interior++ {
			positions[interior] = len(newCode)
		}
		old = candidate.end
	}
	positions[len(oldCode)] = len(newCode)
	for i := 0; i < len(a.labelPos); i++ {
		old := int(a.labelPos[i])
		if old < 0 {
			continue
		}
		if old >= len(positions) || positions[old] < 0 {
			return false
		}
		a.labelPos[i] = int32(positions[old])
	}
	keptRelocs := a.relocs[:0]
	for i := 0; i+1 < len(a.relocs); i += 2 {
		oldAt := int(a.relocs[i])
		converted := -1
		for candidateIndex := 0; candidateIndex < len(candidates); candidateIndex++ {
			if candidates[candidateIndex].at == oldAt {
				converted = candidateIndex
				break
			}
		}
		if converted >= 0 {
			// The relocatable-object writer may realign or regroup function
			// ranges after this pass. Keep a marked label relocation so it can
			// calculate the final rel16 displacement from mapped section offsets.
			keptRelocs = append(keptRelocs, int32(newFields[converted]|-2147483648), a.relocs[i+1])
			continue
		}
		if oldAt < 0 || oldAt >= len(positions) || positions[oldAt] < 0 {
			return false
		}
		keptRelocs = append(keptRelocs, int32(positions[oldAt]), a.relocs[i+1])
	}
	a.relocs = keptRelocs
	for i := 0; i < len(a.absRelocs); i += 3 {
		old := int(a.absRelocs[i])
		if old < 0 || old >= len(positions) || positions[old] < 0 {
			return false
		}
		a.absRelocs[i] = int32(positions[old])
	}
	for i := 0; i < len(a.objectFunctions); i++ {
		old := a.objectFunctions[i].end
		if old < 0 || old >= len(positions) || positions[old] < 0 {
			return false
		}
		a.objectFunctions[i].end = positions[old]
	}
	a.code = newCode
	for i := 0; i < len(candidates); i++ {
		field := newFields[i]
		distance := int(a.labelPos[candidates[i].label]) - (field + 2)
		if distance < -32768 || distance > 32767 {
			return false
		}
		// The object writer calculates the displacement after applying section
		// grouping and function alignment. Leave a zero REL addend here.
		a.code[field] = 0
		a.code[field+1] = 0
	}
	return true
}

func renvoEmitProgramEntryArgs386(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
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
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	if targetIsWindows(g.c.renvoTargetOS) {
		argsTextOff := g.asm.bssSize
		g.asm.bssSize += 32768
		argsLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		renvoAsmBuildWindowsArgvEnvSlices386(&g.asm, argsOff, argsTextOff, argsLenOff, envDataOff, envLenOff)
	} else {
		envDataOff := g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff := g.asm.bssSize
		g.asm.bssSize += 8
		renvoAsmBuildArgvEnvSlices386(&g.asm, argsOff, envDataOff, envLenOff)
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
func renvo386AsmMovRaxAddr(a *renvoAsm, offset int, relocKind int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 0, offset, relocKind)
		return
	}
	renvoAsmEmit8(a, 0xb8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, offset, relocKind)
}

func renvo386AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 3, bssOff, renvoAbsBssReloc)
		return
	}
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvo386AsmMovRegPCRel(a, 0, bssOff, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x008b)
		return
	}
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	if a.c.renvoTargetOS == renvoOSLinux && !a.c.objectFile {
		renvoAsmEmit8(a, 0x53)
		renvo386AsmMovRegPCRel(a, 3, bssOff, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x0389)
		renvoAsmEmit8(a, 0x5b)
		return
	}
	renvoAsmEmit8(a, 0xa3)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvo386AsmX87BinaryStack(a *renvoAsm, dest int, left int, right int, op byte, size int) bool {
	loadOpcode := 0xdd
	storeOpcode := 0xdd
	if size == 4 {
		loadOpcode = 0xd9
		storeOpcode = 0xd9
	}
	renvoAsmStackMem(a, left, loadOpcode, 0x45, 0x85) // fld
	opcode := 0xdc
	shortModRM := 0x45
	longModRM := 0x85
	if size == 4 {
		opcode = 0xd8
	}
	if op == '*' {
		shortModRM, longModRM = 0x4d, 0x8d
	} else if op == '-' {
		shortModRM, longModRM = 0x65, 0xa5
	} else if op == '/' {
		shortModRM, longModRM = 0x75, 0xb5
	} else if op != '+' {
		return false
	}
	renvoAsmStackMem(a, right, opcode, shortModRM, longModRM)
	renvoAsmStackMem(a, dest, storeOpcode, 0x5d, 0x9d) // fstp
	return true
}

func renvo386AsmX87CompareStack(a *renvoAsm, left int, right int, size int) {
	opcode := 0xdd
	if size == 4 {
		opcode = 0xd9
	}
	renvoAsmStackMem(a, right, opcode, 0x45, 0x85)
	renvoAsmStackMem(a, left, opcode, 0x45, 0x85)
	renvoAsmEmit16(a, 0xe9df) // fucomip st(0), st(1)
	renvoAsmEmit16(a, 0xd8dd) // fstp st(0)
}

func renvo386AsmX87IntToFloatStack(a *renvoAsm, offset int, intSize int, floatSize int, signed bool) {
	if !signed && intSize < 8 {
		renvoAsmStoreStackImm(a, offset-4, 0)
		intSize = 8
	}
	if intSize == 8 {
		renvoAsmStackMem(a, offset, 0xdf, 0x6d, 0xad) // fild qword
	} else {
		renvoAsmStackMem(a, offset, 0xdb, 0x45, 0x85) // fild dword
	}
	if !signed && intSize == 8 {
		converted := renvoAsmNewLabel(a)
		renvoAsmJcmpStackImm(a, offset-4, 0, converted, 0x9d)
		// FILD is signed. For the upper uint64 half, adding exactly 2^64
		// converts its negative two's-complement interpretation to the
		// corresponding non-negative value before the final rounding step.
		renvoAsmStoreStackImm(a, offset, 0)
		renvoAsmStoreStackImm(a, offset-4, 0x43f00000)
		renvoAsmStackMem(a, offset, 0xdc, 0x45, 0x85) // fadd qword
		renvoAsmMarkLabel(a, converted)
	}
	if floatSize == 4 {
		renvoAsmStackMem(a, offset, 0xd9, 0x5d, 0x9d)
	} else {
		renvoAsmStackMem(a, offset, 0xdd, 0x5d, 0x9d)
	}
}

func renvo386AsmX87ConvertFloatStack(a *renvoAsm, dest int, source int, sourceSize int, destSize int) {
	loadOpcode := 0xdd
	if sourceSize == 4 {
		loadOpcode = 0xd9
	}
	storeOpcode := 0xdd
	if destSize == 4 {
		storeOpcode = 0xd9
	}
	renvoAsmStackMem(a, source, loadOpcode, 0x45, 0x85) // fld
	renvoAsmStackMem(a, dest, storeOpcode, 0x5d, 0x9d)  // fstp
}

func renvo386AsmX87FloatToIntStack(a *renvoAsm, dest int, source int, floatSize int, intSize int, signed bool) {
	loadOpcode := 0xdd
	if floatSize == 4 {
		loadOpcode = 0xd9
	}
	renvoAsmStackMem(a, source, loadOpcode, 0x45, 0x85) // fld
	storeOpcode := 0xdb                                 // fisttp dword
	if intSize == 8 || !signed {
		storeOpcode = 0xdd // fisttp qword
	}
	renvoAsmStackMem(a, dest, storeOpcode, 0x4d, 0x8d)
}

func renvo386AsmX87NegateStack(a *renvoAsm, offset int, size int) {
	highOffset := offset
	if size == 8 {
		highOffset -= 4
	}
	renvoAsmLoadPrimaryStack(a, highOffset)
	renvoAsmEmit8(a, 0x35) // xor eax, imm32
	renvoAsmEmit32(a, -2147483648)
	renvoAsmStorePrimaryStack(a, highOffset)
}

func renvo386EmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoAsmEmit16(a, 0x5f52)
		renvoAsmEmit16(a, 0x728d)
		renvoAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		if g.c.renvoTargetOS == renvoOSLinux {
			renvo386AsmMovRegPCRel(a, 7, loc.offset, renvoAbsBssReloc)
			renvo386AsmMovRegPCRel(a, 6, loc.offset+8, renvoAbsBssReloc)
			return true
		}
		renvoAsmEmit16(a, 0x3d8d)
		at := len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset, renvoAbsBssReloc)
		renvoAsmEmit16(a, 0x358d)
		at = len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, loc.offset+8, renvoAbsBssReloc)
		return true
	}
	renvoAsmAddressCallWord0Stack(a, loc.offset)
	renvoAsmAddressCallWord1Stack(a, loc.offset-8)
	return true
}

func renvo386EmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, srcPtr)
	renvoAsmEmit16(a, 0xc689)
	renvoAsmLoadPrimaryStack(a, destPtr)
	renvoAsmEmit16(a, 0xc789)
	renvoAsmLoadTertiaryStack(a, byteCount)
	renvoAsmEmitText(a, "\x39\xf7\x76\x0e\x8d\x74\x0e\xff\x8d\x7c\x0f\xff\xfd\xf3\xa4\xfc\xeb\x03\xfc\xf3\xa4")
}

func renvo386EnsureWideBinaryHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideBinaryLabel > 0 {
		return g.wideBinaryLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideBinaryLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x83\xf8\x0d\x75\x18\x8b\x0e\x8b\x56\x04\x8b\x03\xf7\xd0\x21\xc1\x8b\x43\x04\xf7\xd0\x21\xc2\x89\x0f\x89\x57\x04\xc3\x83\xf8\x0e\x72\x05\xe9\xb3\x01\x00\x00\x83\xf8\x00\x74\x1e\x83\xf8\x01\x74\x29\x83\xf8\x02\x74\x34\x83\xf8\x0a\x73\x4b\x83\xf8\x06\x0f\x86\xf2\x00\x00\x00\x83\xf8\x09\x76\x65\xc3\x8b\x06\x03\x03\x89\x07\x8b\x46\x04\x13\x43\x04\x89\x47\x04\xc3\x8b\x06\x2b\x03\x89\x07\x8b\x46\x04\x1b\x43\x04\x89\x47\x04\xc3\x8b\x06\xf7\x23\x89\x07\x89\xd1\x8b\x46\x04\x0f\xaf\x03\x01\xc1\x8b\x06\x0f\xaf\x43\x04\x01\xc8\x89\x47\x04\xc3\x8b\x0e\x8b\x56\x04\x83\xf8\x0a\x74\x0c\x83\xf8\x0b\x74\x0e\x33\x0b\x33\x53\x04\xeb\x0c\x23\x0b\x23\x53\x04\xeb\x05\x0b\x0b\x0b\x53\x04\x89\x0f\x89\x57\x04\xc3\x8b\x0e\x8b\x56\x04\x8b\x73\x04\x85\xf6\x75\x64\x8b\x1b\x83\xfb\x40\x73\x5d\x83\xfb\x20\x73\x2b\x51\x89\xd9\x83\xf8\x07\x74\x17\x5b\x89\xde\x0f\xad\xd6\x83\xf8\x08\x74\x04\xd3\xfa\xeb\x02\xd3\xea\x89\x37\x89\x57\x04\xc3\x5e\x0f\xa5\xf2\xd3\xe6\x89\x37\x89\x57\x04\xc3\x83\xeb\x20\x51\x89\xd9\x83\xf8\x07\x74\x15\x5e\x89\xd6\x83\xf8\x08\x74\x07\xd3\xfe\xc1\xfa\x1f\xeb\xd4\xd3\xee\x31\xd2\xeb\xce\x5e\x89\xf2\xd3\xe2\x31\xf6\x89\x37\x89\x57\x04\xc3\x83\xf8\x09\x74\x0a\x31\xc9\x31\xd2\x89\x0f\x89\x57\x04\xc3\xc1\xfa\x1f\x89\x17\x89\x57\x04\xc3\x55\x57\x50\x8b\x06\x8b\x56\x04\x8b\x0b\x8b\x5b\x04\x31\xf6\x31\xff\x80\x3c\x24\x05\x72\x1a\x89\xd6\xc1\xfe\x1f\x89\xdf\xc1\xff\x1f\x31\xf0\x31\xf2\x29\xf0\x19\xf2\x31\xf9\x31\xfb\x29\xf9\x19\xfb\x56\x57\x53\x51\x31\xc9\x31\xdb\x31\xf6\x31\xff\xbd\x40\x00\x00\x00\xd1\xe0\xd1\xd2\xd1\xd6\xd1\xd7\xd1\xe1\xd1\xd3\x3b\x7c\x24\x04\x72\x11\x77\x05\x3b\x34\x24\x72\x0a\x2b\x34\x24\x1b\x7c\x24\x04\x83\xc9\x01\x4d\x75\xda\xf6\x44\x24\x10\x01\x74\x1b\x8b\x44\x24\x0c\x33\x44\x24\x08\x31\xc1\x31\xc3\x29\xc1\x19\xc3\x8b\x54\x24\x14\x89\x0a\x89\x5a\x04\xeb\x15\x8b\x44\x24\x0c\x31\xc6\x31\xc7\x29\xc6\x19\xc7\x8b\x54\x24\x14\x89\x32\x89\x7a\x04\x83\xc4\x18\x5d\xc3\x83\xe8\x0e\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EnsureWideCompareHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.wideCompareLabel > 0 {
		return g.wideCompareLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.wideCompareLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmEmitText(&g.asm, "\x8b\x56\x04\x3b\x53\x04\x75\x26\x8b\x16\x3b\x13\x75\x38\x83\xf8\x00\x74\x69\x83\xf8\x01\x74\x61\x83\xf8\x03\x74\x5f\x83\xf8\x05\x74\x5a\x83\xf8\x07\x74\x55\x83\xf8\x09\x74\x50\xeb\x4b\x83\xf8\x01\x74\x49\x83\xf8\x06\x73\x07\x3b\x53\x04\x7c\x28\xeb\x10\x3b\x53\x04\x72\x21\xeb\x09\x83\xf8\x01\x74\x31\x3b\x13\x72\x16\x83\xf8\x04\x74\x28\x83\xf8\x05\x74\x23\x83\xf8\x08\x74\x1e\x83\xf8\x09\x74\x19\xeb\x14\x83\xf8\x02\x74\x12\x83\xf8\x03\x74\x0d\x83\xf8\x06\x74\x08\x83\xf8\x07\x74\x03\x31\xc0\xc3\xb8\x01\x00\x00\x00\xc3")
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvo386EmitWideHelperCall(g *renvoLinearGen, dest int, left int, right int, mode int, label int) {
	a := &g.asm
	renvoAsmStackMem(a, dest, 0x8d, 0x7d, 0xbd)
	renvoAsmStackMem(a, left, 0x8d, 0x75, 0xb5)
	renvoAsmStackMem(a, right, 0x8d, 0x5d, 0x9d)
	renvoAsmPrimaryImm(a, mode)
	renvoAsmCallLabel(a, label)
}

func renvo386EmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	renvoNonNil(g)
	// Keep the common integer operations at their call site.  Pulling in the
	// general dispatcher for a single add, multiply, bitwise operation, or
	// comparison costs several hundred bytes in small 386 objects and also
	// hides the operation from the ordinary branch and load peepholes.
	if renvoFixedTarget == 0 && g.c.code16 && mode == 0 {
		renvoEmitWideAddStack(g, dest, left, right)
		return
	}
	if renvoFixedTarget == 0 && g.c.code16 && mode == 1 {
		renvoEmitWideSubStack(g, dest, left, right)
		return
	}
	if renvoFixedTarget == 0 && g.c.code16 && mode == 2 {
		renvoEmitWideMulStack(g, dest, left, right)
		return
	}
	if renvoFixedTarget == 0 && g.c.code16 && mode >= 10 && mode <= 13 {
		renvo386EmitWideBitwiseStack(g, dest, left, right, mode)
		return
	}
	if renvoFixedTarget == 0 && g.c.code16 && mode >= 14 && mode <= 23 {
		renvo386EmitWideCompareInlineStack(g, left, right, mode)
		return
	}
	if mode >= 3 && mode <= 6 {
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvo386EmitWideHelperCall(g, dest, left, right, mode, renvo386EnsureWideBinaryHelper(g))
}

func renvo386EmitWideBitwiseStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	for word := 0; word < 2; word++ {
		leftWord := left - word*g.c.renvoNativeIntSize
		rightWord := right - word*g.c.renvoNativeIntSize
		destWord := dest - word*g.c.renvoNativeIntSize
		renvoAsmLoadPrimaryStack(&g.asm, leftWord)
		renvoAsmLoadTertiaryStack(&g.asm, rightWord)
		if mode == 10 {
			renvoAsmEmit16(&g.asm, 0xc821)
		} else if mode == 11 {
			renvoAsmEmit16(&g.asm, 0xc809)
		} else if mode == 12 {
			renvoAsmEmit16(&g.asm, 0xc831)
		} else {
			renvoAsmEmit16(&g.asm, 0xd1f7)
			renvoAsmEmit16(&g.asm, 0xc821)
		}
		renvoAsmStorePrimaryStack(&g.asm, destWord)
	}
}

func renvo386EmitWideCompareInlineStack(g *renvoLinearGen, left int, right int, mode int) {
	if mode == 14 || mode == 15 {
		different := renvoAsmNewLabel(&g.asm)
		done := renvoAsmNewLabel(&g.asm)
		renvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x95)
		renvoAsmJnzPrimary(&g.asm, different)
		renvoEmitNativeCompareStack(g, left, right, 0x95)
		renvoAsmJmpLabel(&g.asm, done)
		renvoAsmMarkLabel(&g.asm, different)
		renvoAsmPrimaryImm(&g.asm, 1)
		renvoAsmMarkLabel(&g.asm, done)
		if mode == 14 {
			renvoAsmBoolNotPrimary(&g.asm)
		}
		return
	}
	signed := mode >= 16 && mode <= 19
	inclusive := mode == 17 || mode == 19 || mode == 21 || mode == 23
	leftFirst := mode == 16 || mode == 19 || mode == 20 || mode == 23
	if leftFirst {
		renvoEmitWideLessStack(g, left, right, signed)
	} else {
		renvoEmitWideLessStack(g, right, left, signed)
	}
	if inclusive {
		renvoAsmBoolNotPrimary(&g.asm)
	}
}

func renvo386EmitWideCompareStack(g *renvoLinearGen, left int, right int, mode int) {
	renvoNonNil(g)
	renvo386EmitWideHelperCall(g, 0, left, right, mode, renvo386EnsureWideCompareHelper(g))
}

func renvo386EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldLocalCount := g.localCount
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldClosureEnvOffset := g.closureEnvOffset
	oldDeferHeadOffset := g.deferHeadOffset
	oldDeferReturnLabel := g.deferReturnLabel
	oldDeferResultOffset := g.deferResultOffset
	oldDeferSites := g.deferSites
	oldEmittingDefers := g.emittingDefers
	oldSuppressPanicCheck := g.suppressPanicCheck
	oldStackUsed := g.stackUsed
	oldStackPeak := g.stackPeak
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.stackUsed = 0
	g.stackPeak = 0
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	framePatch := len(a.code)
	renvoAsmEmit32(a, 0x000000c8)
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
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
	frame := renvoAlignTo8(g.stackPeak)
	if frame > 65528 {
		frame = 65528
	}
	renvoPut32At(a.code, framePatch, 0x000000c8|frame<<8)
	g.locals = oldLocals
	g.localCount = oldLocalCount
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.closureEnvOffset = oldClosureEnvOffset
	g.deferHeadOffset = oldDeferHeadOffset
	g.deferReturnLabel = oldDeferReturnLabel
	g.deferResultOffset = oldDeferResultOffset
	g.deferSites = oldDeferSites
	g.emittingDefers = oldEmittingDefers
	g.suppressPanicCheck = oldSuppressPanicCheck
	g.stackUsed = oldStackUsed
	g.stackPeak = oldStackPeak
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func renvo386EmitCompactCValueHelper(g *renvoLinearGen, fnInfoIndex int) bool {
	if !g.c.code16 || !g.c.objectFile || fnInfoIndex < 0 || fnInfoIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnInfoIndex]
	postInc := renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, "__c_post_assign_inc_")
	postDec := renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, "__c_post_assign_dec_")
	if !postInc && !postDec || fn.paramCount != 2 || fn.resultType == 0 {
		return false
	}
	result := renvoResolveType(g.meta, fn.resultType)
	size := renvoTypeSize(g.meta, fn.resultType)
	if size < 1 || size > 4 ||
		(!renvoTypeKindIsScalarValue(result.kind) && result.kind != renvoTypePointer && result.kind != renvoTypeFunc) {
		return false
	}
	a := &g.asm
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	// Internal word 1 (ESI) is **T and word 0 (EBX) is the assignment
	// value. Advance *p, store the value through the old pointer, and leave the
	// expression result in EAX without constructing a frame.
	renvoAsmEmit16(a, 0x168b) // mov edx,[esi]
	delta := size
	if postDec {
		delta = -delta
	}
	if renvoAsmImmFits8Signed(delta) {
		renvoAsmEmit3(a, 0x8d, 0x4a, delta) // lea ecx,[edx+delta]
	} else {
		renvoAsmEmit16(a, 0x8a8d)
		renvoAsmEmit32(a, delta)
	}
	renvoAsmEmit16(a, 0x0e89) // mov [esi],ecx
	if size == 1 {
		renvoAsmEmit16(a, 0x1a88) // mov [edx],bl
		if result.kind == renvoTypeInt8 {
			renvoAsmEmitText(a, "\x0f\xbe\xc3")
		} else {
			renvoAsmEmitText(a, "\x0f\xb6\xc3")
		}
	} else if size == 2 {
		renvoAsmEmitText(a, "\x66\x89\x1a") // mov [edx],bx
		if result.kind == renvoTypeInt16 {
			renvoAsmEmitText(a, "\x0f\xbf\xc3")
		} else {
			renvoAsmEmitText(a, "\x0f\xb7\xc3")
		}
	} else {
		renvoAsmEmit16(a, 0x1a89) // mov [edx],ebx
		renvoAsmEmit16(a, 0xd889) // mov eax,ebx
	}
	renvoAsmRet(a)
	return true
}

func renvo386StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return
	}
	if reg == 1 {
		renvoAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return
	}
	if reg == 2 {
		renvoAsmStoreSecondaryStack(a, offset)
		return
	}
	if reg == 3 {
		renvoAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return
	}
	if reg == 4 {
		renvoAsmStorePrimaryStack(a, offset)
		return
	}
	if reg == 5 {
		renvoAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return
	}
	renvoAsmEmit16(a, 0x858b)
	renvoAsmEmit32(a, 8+(reg-6)*4)
	renvoAsmStorePrimaryStack(a, offset)
}

func renvo386EmitRaxRcxOp(g *renvoLinearGen, tok int, unsigned bool) bool {
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
		renvoAsmAddPrimaryTertiary(a)
		return true
	}
	if c0 == '-' {
		renvoAsmEmit16(a, 0xc129)
		renvoAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		renvoAsmEmit24(a, 0xc1af0f)
		return true
	}
	if c0 == '/' {
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if c0 == '%' {
		renvoAsmDivLeftTertiaryRightPrimary(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoAsmEmit16(a, 0xd0f7)
			renvoAsmEmit16(a, 0xc821)
		} else {
			renvoAsmEmit16(a, 0xc821)
		}
		return true
	}
	if c0 == '|' {
		renvoAsmEmit16(a, 0xc809)
		return true
	}
	if c0 == '^' {
		renvoAsmEmit16(a, 0xc831)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			renvoAsmEmit16(a, 0xe0d3)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9e)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoAsmEmit16(a, 0xca89)
			renvoAsmEmit16(a, 0xc189)
			renvoAsmEmit16(a, 0xd089)
			opcode := 0xf8d3
			if unsigned {
				opcode = 0xe8d3
			}
			renvoAsmEmit16(a, opcode)
		} else if c1 == '=' {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
		} else {
			renvoAsmCmpTertiaryPrimarySet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoAsmCmpTertiaryPrimarySet(a, 0x95)
		return true
	}
	return false
}

func renvo386EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		renvoAsmPopCallWord0(a)
	}
	if wordCount > 1 {
		renvoAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		renvoAsmPopSecondary(a)
	}
	if wordCount > 3 {
		renvoAsmPopTertiary(a)
	}
	if wordCount > 4 {
		renvoAsmPopPrimary(a)
	}
	if wordCount > 5 {
		renvoAsmEmit8(a, 0x5f)
	}
	renvoAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if renvoAsmImmFits8Signed(imm) {
			renvoAsmEmit3(a, 0x83, 0xc4, imm)
		} else {
			renvoAsmEmit16(a, 0xc481)
			renvoAsmEmit32(a, imm)
		}
	}
}

// C lowering frequently stages call arguments in frame slots, then emits a
// load/push pair for every word immediately before the internal call adapter
// pops those same words into registers. Replace that lossless stack round trip
// with direct frame-to-register loads. The conservative metadata checks keep
// branch targets and relocation fields out of the rewritten range.
func renvo386TryLoadCallWordsFromFrame(a *renvoAsm, wordCount int) bool {
	if wordCount < 1 || wordCount > 6 {
		return false
	}
	starts := make([]int, wordCount)
	displacements := make([]int, wordCount)
	wide := make([]bool, wordCount)
	cursor := len(a.code)
	for i := wordCount - 1; i >= 0; i-- {
		if cursor < 4 || a.code[cursor-1] != 0x50 {
			return false
		}
		if cursor >= 4 && a.code[cursor-4] == 0x8b && a.code[cursor-3] == 0x45 {
			starts[i] = cursor - 4
			displacements[i] = int(int8(a.code[cursor-2]))
			cursor -= 4
			continue
		}
		if cursor >= 7 && a.code[cursor-7] == 0x8b && a.code[cursor-6] == 0x85 {
			starts[i] = cursor - 7
			displacements[i] = int(int32(uint32(a.code[cursor-5]) | uint32(a.code[cursor-4])<<8 |
				uint32(a.code[cursor-3])<<16 | uint32(a.code[cursor-2])<<24))
			wide[i] = true
			cursor -= 7
			continue
		}
		return false
	}
	start := starts[0]
	for i := 0; i < len(a.labelPos); i++ {
		if int(a.labelPos[i]) >= start {
			return false
		}
	}
	for i := 0; i < len(a.relocs); i += 2 {
		if int(a.relocs[i]) >= start {
			return false
		}
	}
	for i := 0; i < len(a.absRelocs); i += 3 {
		if int(a.absRelocs[i]) >= start {
			return false
		}
	}
	a.code = a.code[:start]
	registers := []int{3, 6, 2, 1, 0, 7}
	for i := 0; i < wordCount; i++ {
		register := registers[wordCount-1-i]
		renvoAsmEmit8(a, 0x8b)
		if wide[i] {
			renvoAsmEmit8(a, 0x85|register<<3)
			renvoAsmEmit32(a, displacements[i])
		} else {
			renvoAsmEmit8(a, 0x45|register<<3)
			renvoAsmEmit8(a, displacements[i])
		}
	}
	return true
}
func renvo386EmitEnsureMemSlice(g *renvoLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	renvoAsmPrimaryBssAddr(a, backingOff)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmPrimaryImm(a, backingSize/elemSize)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 16)
	renvoAsmMarkLabel(a, okLabel)
}

func renvo386EnsureAppendAddrHelper(g *renvoLinearGen) int {
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
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc139)
	renvo386AsmJccLabel(a, 0x8c, noGrowLabel)
	renvoAsmEmit8(a, 0x57)
	renvoAsmEmit8(a, 0x56)
	renvoAsmPushSecondary(a)
	renvoAsmPushTertiary(a)
	renvoAsmEmit3(a, 0x8b, 0x46, 8)
	renvoAsmEmit16(a, 0xc085)
	renvo386AsmJccLabel(a, 0x85, capNonZeroLabel)
	renvoAsmPrimaryImm(a, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoAsmPushPrimary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0x4c)
	renvoAsmEmit2(a, 0x24, 8)
	renvoAsmPushTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmCallLabel(a, arenaAllocLabel)
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmEmit16(a, 0xc085)
		renvo386AsmJccLabel(a, 0x85, allocOKLabel)
		renvoAsmEmit3(a, 0x83, 0xc4, 20)
		renvoAsmRet(a)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
	renvoAsmPushPrimary(a)
	renvoAsmEmit3(a, 0x8b, 0x3c, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 20)
	renvoAsmEmit16(a, 0x368b)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xc8)
	renvoAsmEmit16(a, 0xa4f3)
	renvoAsmEmit4(a, 0x8b, 0x7c, 0x24, 20)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit16(a, 0x0789)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x44, 0x24, 4)
	renvoAsmEmit3(a, 0x89, 0x46, 8)
	renvoAsmEmit3(a, 0x8b, 0x04, 0x24)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmEmit4(a, 0x8b, 0x54, 0x24, 12)
	renvoAsmEmit3(a, 0x0f, 0xaf, 0xca)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit4(a, 0x8b, 0x74, 0x24, 16)
	renvoAsmEmit4(a, 0x8b, 0x4c, 0x24, 8)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmEmit3(a, 0x83, 0xc4, 24)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, noGrowLabel)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmEmit16(a, 0x078b)
	renvoAsmEmit24(a, 0xcaaf0f)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmEmit16(a, 0x0e8b)
	renvoAsmIncTertiary(a)
	renvoAsmEmit16(a, 0x0e89)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvo386EnsureAppendScalarHelper(g *renvoLinearGen, small bool) int {
	a := &g.asm
	label := g.append64Label
	if small {
		label = g.append8Label
		if g.append8Emitted {
			return label
		}
		g.append8Emitted = true
	} else {
		if g.append64Emitted {
			return label
		}
		g.append64Emitted = true
	}
	label = renvoAsmNewLabel(a)
	if small {
		g.append8Label = label
	} else {
		g.append64Label = label
	}
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, label)
	if small {
		renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x88\x14\x08\x41\x89\x0e\xc3")
	} else {
		renvoAsmEmitText(a, "\x8b\x0e\x8b\x07\x89\x14\xc8\x41\x89\x0e\xc3")
	}
	renvoAsmMarkLabel(a, afterLabel)
	return label
}

func renvo386EnsureStringEqualHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit16(a, 0xce39)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit16(a, 0xf685)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmEmit16(a, 0x0b8a)
	renvoAsmEmit16(a, 0x0a38)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoAsmEmit8(a, 0x43)
	renvoAsmEmit8(a, 0x42)
	renvoAsmEmit8(a, 0x4e)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
