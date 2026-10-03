package main

type renvoRTGAssemblySource struct {
	path   []byte
	source []byte
}

type renvoRTGAssemblyBinding struct {
	function int
	source   int
	entry    int
	code     []byte
}

type renvoRTGAssemblyTable struct {
	sources  []renvoRTGAssemblySource
	bindings []renvoRTGAssemblyBinding
}

var renvoRTGAssembly renvoRTGAssemblyTable

func renvoDecodeRTGAssemblyTable(prog *renvoProgram, data []byte) bool {
	renvoNonNil(prog)
	renvoRTGAssembly = renvoRTGAssemblyTable{}
	if len(data) == 0 {
		return true
	}
	r := renvoUnitReader{src: data, end: len(data), ok: true}
	sourceCount := renvoUnitReadVar(&r)
	if !r.ok || sourceCount < 0 || sourceCount > len(data) {
		return false
	}
	table := renvoRTGAssemblyTable{sources: make([]renvoRTGAssemblySource, 0, sourceCount)}
	for i := 0; i < sourceCount; i++ {
		pathLength := renvoUnitReadVar(&r)
		if !r.ok || pathLength <= 0 || r.pos+pathLength < r.pos || r.pos+pathLength > r.end {
			return false
		}
		path := make([]byte, pathLength)
		copy(path, r.src[r.pos:r.pos+pathLength])
		r.pos += pathLength
		sourceLength := renvoUnitReadVar(&r)
		if !r.ok || sourceLength < 0 || r.pos+sourceLength < r.pos || r.pos+sourceLength > r.end {
			return false
		}
		source := make([]byte, sourceLength)
		copy(source, r.src[r.pos:r.pos+sourceLength])
		r.pos += sourceLength
		table.sources = append(table.sources, renvoRTGAssemblySource{path: path, source: source})
	}
	bindingCount := renvoUnitReadVar(&r)
	if !r.ok || bindingCount < 0 || bindingCount > len(data) {
		return false
	}
	table.bindings = make([]renvoRTGAssemblyBinding, 0, bindingCount)
	seen := make([]bool, len(prog.funcs))
	for i := 0; i < bindingCount; i++ {
		binding := renvoRTGAssemblyBinding{function: renvoUnitReadVar(&r), source: renvoUnitReadVar(&r), entry: renvoUnitReadVar(&r)}
		codeLength := renvoUnitReadVar(&r)
		if !r.ok || codeLength < 0 || r.pos+codeLength < r.pos || r.pos+codeLength > r.end {
			return false
		}
		if codeLength > 0 {
			binding.code = make([]byte, codeLength)
			copy(binding.code, r.src[r.pos:r.pos+codeLength])
		}
		r.pos += codeLength
		if !r.ok || binding.function < 0 || binding.function >= len(prog.funcs) || binding.source < 0 || binding.source >= len(table.sources) || binding.entry < 0 || seen[binding.function] {
			return false
		}
		seen[binding.function] = true
		table.bindings = append(table.bindings, binding)
	}
	if r.pos != r.end {
		return false
	}
	renvoRTGAssembly = table
	return true
}

// These helpers compose the small generated direct-emitter contract into the
// value and stack operations used by the shared backend kernel. They are only
// reached by prepared backends (renvoArchRTG); built-in targets retain their
// checked-in, statically selected fast paths.

func renvoRTGAsmAddress(base RTGRegister, index RTGRegister, displacement int, scale int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Base = base
	address.Index = index
	address.Displacement = displacement
	address.Scale = scale
	return address
}

func renvoRTGAsmFrameAddress(offset int) renvoRTGAddress {
	return renvoRTGAsmAddress(renvoRTGFrame, RTGNoRegister, -offset, 1)
}

func renvoRTGAsmDataAddress(offset int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Kind = 1
	address.Addend = offset
	return address
}

func renvoRTGAsmBSSAddress(offset int) renvoRTGAddress {
	var address renvoRTGAddress
	address.Kind = 2
	address.Addend = offset
	return address
}

func renvoRTGAsmPushRegister(a *renvoAsm, source RTGRegister) {
	if renvoRTGABIPushRegister(a, source) {
		return
	}
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectSubtract(a, renvoRTGStack, renvoRTGScratch)
	renvoRTGDirectStoreNative(a, renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1), source)
}

func renvoRTGAsmPushImmediate(a *renvoAsm, value int) {
	if renvoRTGABIPushImmediate(a, value) {
		return
	}
	// Move SP before materializing the value. An architecture may implement an
	// immediate move with a temporary push/pop sequence, so staging the value
	// below the old SP first would let that sequence overwrite it.
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectSubtract(a, renvoRTGStack, renvoRTGScratch)
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(value))
	renvoRTGDirectStoreNative(a,
		renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1),
		renvoRTGScratch)
}

func renvoRTGAsmPopRegister(a *renvoAsm, destination RTGRegister) {
	if renvoRTGABIPopRegister(a, destination) {
		return
	}
	renvoRTGDirectLoadNative(a, destination,
		renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1))
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	renvoRTGDirectAdd(a, renvoRTGStack, renvoRTGScratch)
}

func renvoRTGAsmLoadFrame(a *renvoAsm, destination RTGRegister, offset int) {
	if renvoRTGABILoadFrame(a, destination, offset) {
		return
	}
	renvoRTGDirectLoadNative(a, destination, renvoRTGAsmFrameAddress(offset))
}

func renvoRTGAsmStoreFrame(a *renvoAsm, offset int, source RTGRegister) {
	if renvoRTGABIStoreFrame(a, offset, source) {
		return
	}
	renvoRTGDirectStoreNative(a, renvoRTGAsmFrameAddress(offset), source)
}

func renvoRTGAsmAddressFrame(a *renvoAsm, destination RTGRegister, offset int) {
	if renvoRTGABIAddressFrame(a, destination, offset) {
		return
	}
	renvoRTGDirectAddress(a, destination, renvoRTGAsmFrameAddress(offset))
}

func renvoRTGAsmLoadSize(a *renvoAsm, destination RTGRegister, address renvoRTGAddress, size int, signed bool) {
	if size <= 1 {
		if signed {
			renvoRTGDirectLoadI8(a, destination, address)
		} else {
			renvoRTGDirectLoadU8(a, destination, address)
		}
		return
	}
	if size == 2 {
		if signed {
			renvoRTGDirectLoadI16(a, destination, address)
		} else {
			renvoRTGDirectLoadU16(a, destination, address)
		}
		return
	}
	if size == 4 && renvoRTGStackWordBytes > 4 {
		if signed {
			renvoRTGDirectLoadI32(a, destination, address)
		} else {
			renvoRTGDirectLoadU32(a, destination, address)
		}
		return
	}
	renvoRTGDirectLoadNative(a, destination, address)
}

func renvoRTGAsmStoreSize(a *renvoAsm, address renvoRTGAddress, source RTGRegister, size int) {
	if size <= 1 {
		renvoRTGDirectStoreU8(a, address, source)
	} else if size == 2 {
		renvoRTGDirectStoreU16(a, address, source)
	} else if size == 4 && renvoRTGStackWordBytes > 4 {
		renvoRTGDirectStoreU32(a, address, source)
	} else {
		renvoRTGDirectStoreNative(a, address, source)
	}
}

func renvoRTGAsmNormalize(a *renvoAsm, kind int) {
	bits := renvoRTGStackWordBytes * 8
	width := bits
	signed := false
	if kind == renvoTypeBool || kind == renvoTypeByte {
		width = 8
	} else if kind == renvoTypeInt8 {
		width = 8
		signed = true
	} else if kind == renvoTypeInt16 {
		width = 16
		signed = true
	} else if kind == renvoTypeInt32 {
		width = 32
		signed = true
	} else if kind == renvoTypeUint16 {
		width = 16
	} else if kind == renvoTypeUint32 {
		width = 32
	}
	if width >= bits {
		return
	}
	shift := byte(bits - width)
	renvoRTGDirectShiftLeftImmediate(a, renvoRTGPrimary, shift)
	if signed {
		renvoRTGDirectShiftRightSignedImmediate(a, renvoRTGPrimary, shift)
	} else {
		renvoRTGDirectShiftRightUnsignedImmediate(a, renvoRTGPrimary, shift)
	}
}

func renvoRTGAsmCompareImmediate(a *renvoAsm, value int) {
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(value))
	renvoRTGDirectCompare(a, renvoRTGPrimary, renvoRTGScratch)
}

func renvoRTGAsmMemoryIncrement(a *renvoAsm, decrement bool) {
	address := renvoRTGAsmAddress(renvoRTGSecondary, RTGNoRegister, 0, 1)
	renvoRTGDirectLoadNative(a, renvoRTGScratch, address)
	if decrement {
		renvoRTGDirectDecrement(a, renvoRTGScratch)
	} else {
		renvoRTGDirectIncrement(a, renvoRTGScratch)
	}
	renvoRTGDirectStoreNative(a, address, renvoRTGScratch)
}

func renvoRTGAsmBoolNot(a *renvoAsm) {
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, 1)
	renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)
}

func renvoRTGEmitPrimaryTertiaryOp(g *renvoLinearGen, tok int) bool {
	renvoNonNil(g)
	p := g.prog
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	c1 := byte(0)
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	a := &g.asm
	if c0 == '+' {
		renvoRTGDirectAdd(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '-' {
		renvoRTGDirectMove(a, renvoRTGScratch, renvoRTGTertiary)
		renvoRTGDirectSubtract(a, renvoRTGScratch, renvoRTGPrimary)
		renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGScratch)
		return true
	}
	if c0 == '*' {
		renvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '/' {
		renvoRTGDirectSignedDivide(a, false)
		return true
	}
	if c0 == '%' {
		renvoRTGDirectSignedDivide(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoRTGDirectMoveImmediate(a, renvoRTGScratch, -1)
			renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGScratch)
		}
		renvoRTGDirectBitAnd(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '|' {
		renvoRTGDirectBitOr(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '^' {
		renvoRTGDirectBitXor(a, renvoRTGPrimary, renvoRTGTertiary)
		return true
	}
	if c0 == '<' && c1 == '<' {
		renvoRTGEmitBoundedVariableShift(a, RTGShiftLeft, false)
		return true
	}
	if c0 == '>' && c1 == '>' {
		renvoRTGEmitBoundedVariableShift(a, RTGShiftRight, true)
		return true
	}
	setcc := 0
	if c0 == '<' {
		if c1 == '=' {
			setcc = 0x9e
		} else {
			setcc = 0x9c
		}
	} else if c0 == '>' {
		if c1 == '=' {
			setcc = 0x9d
		} else {
			setcc = 0x9f
		}
	} else if c0 == '=' && c1 == '=' {
		setcc = 0x94
	} else if c0 == '!' && c1 == '=' {
		setcc = 0x95
	}
	if setcc != 0 {
		renvoRTGDirectCompare(a, renvoRTGTertiary, renvoRTGPrimary)
		renvoRTGDirectSetCondition(a, renvoRTGConditionFromSetcc(setcc), renvoRTGPrimary)
		return true
	}
	return false
}

// Prepared ISA emitters expose machine shifts, whose count masking differs.
// Apply the language's full-word boundary before invoking those instructions.
func renvoRTGEmitBoundedVariableShift(a *renvoAsm, direction RTGShiftDirection, signed bool) {
	shift := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	bits := a.c.renvoNativeIntSize * 8
	oversized := renvoAsmNewLabel(a)
	// Two signed comparisons also work on bytecode backends whose branch
	// instructions have no unsigned condition encoding.
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, 0)
	renvoRTGDirectCompare(a, renvoRTGPrimary, renvoRTGScratch)
	renvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x9c), oversized)
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(bits))
	renvoRTGDirectCompare(a, renvoRTGPrimary, renvoRTGScratch)
	renvoRTGDirectJumpCondition(a, renvoRTGConditionFromSetcc(0x9c), shift)
	renvoAsmMarkLabel(a, oversized)
	if direction == RTGShiftRight && signed {
		renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGTertiary)
		renvoRTGDirectShiftRightSignedImmediate(a, renvoRTGPrimary, byte(bits-1))
	} else {
		renvoRTGDirectMoveImmediate(a, renvoRTGPrimary, 0)
	}
	renvoAsmJmpLabel(a, done)
	renvoAsmMarkLabel(a, shift)
	renvoRTGDirectVariableShift(a, direction, signed)
	renvoAsmMarkLabel(a, done)
}

func renvoRTGEmitAssemblyFunction(a *renvoAsm, declIndex int, label int) int {
	for i := 0; i < len(renvoRTGAssembly.bindings); i++ {
		binding := &renvoRTGAssembly.bindings[i]
		if binding.function != declIndex {
			continue
		}
		if len(binding.code) == 0 {
			renvoPrintErr("renvo: RTGASM entry was not evaluated by CompilerJIT\n")
			return -1
		}
		renvoRTGFunctionStart(a, label)
		renvoAsmMarkLabel(a, label)
		for at := 0; at < len(binding.code); at++ {
			a.code = append(a.code, binding.code[at])
		}
		renvoRTGFunctionFinish(a)
		return 1
	}
	return 0
}

func renvoRTGStoreParamWord(g *renvoLinearGen, word int, offset int) {
	if renvoRTGABIStoreParamWord(&g.asm, word, offset) {
		return
	}
	if word >= 0 && word < 6 {
		registers := []RTGRegister{
			renvoRTGCallWord0, renvoRTGCallWord1, renvoRTGCallWord2,
			renvoRTGCallWord3, renvoRTGCallWord4, renvoRTGCallWord5,
		}
		if registers[word].Valid {
			renvoRTGAsmStoreFrame(&g.asm, offset, registers[word])
			return
		}
		if renvoRTGUnsupportedOperation == 0 {
			renvoRTGUnsupportedOperation = 3001
		}
		return
	}
	overflowBase := 2 * renvoRTGStackWordBytes
	// Current link-register ABIs also use their first call word as the primary
	// result register. Their frame prologues save the link and old frame below
	// the incoming stack pointer, so overflow arguments start at frame itself.
	if renvoRTGCallWord0.Code == renvoRTGPrimary.Code {
		overflowBase = 0
	}
	source := renvoRTGAsmAddress(renvoRTGFrame, RTGNoRegister,
		overflowBase+(word-6)*renvoRTGStackWordBytes, 1)
	renvoRTGDirectLoadNative(&g.asm, renvoRTGPrimary, source)
	renvoRTGAsmStoreFrame(&g.asm, offset, renvoRTGPrimary)
}

func renvoRTGEmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	if renvoRTGABICallWordCount(&g.asm, g.funcLabels[fnIndex], wordCount) {
		return
	}
	registers := []RTGRegister{
		renvoRTGCallWord0, renvoRTGCallWord1, renvoRTGCallWord2,
		renvoRTGCallWord3, renvoRTGCallWord4, renvoRTGCallWord5,
	}
	limit := wordCount
	if limit > len(registers) {
		limit = len(registers)
	}
	for i := 0; i < limit; i++ {
		if !registers[i].Valid {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 3002
			}
			return
		}
		renvoRTGAsmPopRegister(&g.asm, registers[i])
	}
	renvoAsmCallLabel(&g.asm, g.funcLabels[fnIndex])
	if wordCount > len(registers) {
		stackBytes := (wordCount - len(registers)) * renvoRTGStackWordBytes
		renvoRTGDirectMoveImmediate(&g.asm, renvoRTGScratch,
			int64(stackBytes))
		renvoRTGDirectAdd(&g.asm, renvoRTGStack, renvoRTGScratch)
	}
}

func renvoRTGEmitCopyBytes(g *renvoLinearGen, srcPtr int, destPtr int, byteCount int) {
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopySource, srcPtr)
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopyDestination, destPtr)
	renvoRTGAsmLoadFrame(&g.asm, renvoRTGCopyCount, byteCount)
	renvoRTGDirectCopyBytes(&g.asm)
}

func renvoTryCompileScalarProgramRTG(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	renvoRTGUnsupportedOperation = 0
	renvoRTGFailureDetail = -1
	renvoRTGImageLimitMemory = false
	renvoRTGImageLimitNeeded = 0
	renvoRTGImageLimit = 0
	return renvoTryCompileScalarProgramScratch(p, meta)
}

func renvoRTGAdjustObjectStack(a *renvoAsm, reserve bool) {
	renvoRTGDirectMoveImmediate(a, renvoRTGScratch, int64(renvoRTGStackWordBytes))
	if reserve {
		renvoRTGDirectSubtract(a, renvoRTGStack, renvoRTGScratch)
	} else {
		renvoRTGDirectAdd(a, renvoRTGStack, renvoRTGScratch)
	}
}

func renvoRTGObjectExportFrame(a *renvoAsm, reserve bool) {
	if reserve {
		patch := renvoRTGFrameStart(a)
		renvoRTGFrameFinish(a, patch, 0)
	} else {
		renvoAsmLeave(a)
	}
}

func renvoRTGObjectRegisters() []RTGRegister {
	return []RTGRegister{
		renvoRTGObjectArgument0, renvoRTGObjectArgument1, renvoRTGObjectArgument2,
		renvoRTGObjectArgument3, renvoRTGObjectArgument4, renvoRTGObjectArgument5,
		renvoRTGObjectArgument6, renvoRTGObjectArgument7,
	}
}

func renvoRTGObjectRegisterCount() int {
	registers := renvoRTGObjectRegisters()
	for i := 0; i < len(registers); i++ {
		if !registers[i].Valid {
			return i
		}
	}
	return len(registers)
}

func renvoRTGPushObjectCallWord(a *renvoAsm, word int) bool {
	registers := renvoRTGObjectRegisters()
	if word < 0 || word >= len(registers) || !registers[word].Valid {
		return false
	}
	renvoRTGAsmPushRegister(a, registers[word])
	return true
}

func renvoTryCompileObjectProgramRTG(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	return renvoTryCompileScalarProgramRTG(p, meta)
}

func renvoRTGReportFailure(g *renvoLinearGen) {
	renvoPrintErr("renvo: prepared backend ")
	name, _, _, found := renvoRTGTargetBinding(renvoTargetRTG)
	if found {
		renvoPrintErr(name)
	} else {
		renvoPrintErr("<unknown>")
	}
	if renvoRTGUnsupportedOperation >= 1000 && renvoRTGUnsupportedOperation < 2000 {
		renvoPrintErr(" is missing condition selector ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 1000)
	} else if renvoRTGUnsupportedOperation >= 4000 {
		renvoPrintErr(" reached an unsupported compiler helper ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 4000)
	} else if renvoRTGUnsupportedOperation >= 3000 {
		renvoPrintErr(" used an invalid required register")
	} else if renvoRTGUnsupportedOperation >= 2000 {
		renvoPrintErr(" left an invalid or unresolved relocation ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation - 2000)
		if renvoRTGFailureDetail >= 0 {
			renvoPrintErr(" targeting label ")
			renvoPrintIntErr(renvoRTGFailureDetail)
			for i := 0; i < len(g.funcLabels); i++ {
				if g.funcLabels[i] == renvoRTGFailureDetail {
					renvoPrintErr(" (")
					write(2, g.prog.src[g.meta.funcs[i].nameStart:g.meta.funcs[i].nameEnd], -1)
					renvoPrintErr(")")
				}
			}
		}
	} else {
		renvoPrintErr(" is missing direct emitter operation ")
		renvoPrintIntErr(renvoRTGUnsupportedOperation)
	}
	renvoPrintErr("\n")
}

var renvoRTGFailureDetail = -1
var renvoRTGImageLimitMemory bool
var renvoRTGImageLimitNeeded int
var renvoRTGImageLimit int

func renvoRTGReportImageSize(g *renvoLinearGen) {
	renvoPrintErr("renvo: error RENVO-BACKEND-011 (backend): target ")
	name, _, _, found := renvoRTGTargetBinding(renvoTargetRTG)
	if found {
		renvoPrintErr(name)
	} else {
		renvoPrintErr("<unknown>")
	}
	if renvoRTGImageLimitMemory {
		renvoPrintErr(" program exceeds its single-segment memory limit: needs ")
	} else {
		renvoPrintErr(" program exceeds its executable image limit: needs ")
	}
	renvoPrintIntErr(renvoRTGImageLimitNeeded)
	renvoPrintErr(" bytes, limit ")
	renvoPrintIntErr(renvoRTGImageLimit)
	renvoPrintErr(" bytes (code ")
	renvoPrintIntErr(len(g.asm.code))
	renvoPrintErr(", initialized data ")
	renvoPrintIntErr(len(g.asm.data))
	renvoPrintErr(", zero-initialized data ")
	renvoPrintIntErr(g.asm.bssSize)
	renvoPrintErr(", arena ")
	renvoPrintIntErr(g.arenaSize)
	renvoPrintErr("); reduce -arena-size or program code/global data\n")
}

func renvoRTGValidateRelocations(out *renvoAsm) {
	if out.patchFailed && renvoRTGUnsupportedOperation == 0 {
		renvoRTGUnsupportedOperation = 2004
	}
	for i := 0; i+1 < len(out.relocs); i += 2 {
		at := int(renvo_runtime_UnsafeInt32At(out.relocs, i)) & 2147483647
		label := int(renvo_runtime_UnsafeInt32At(out.relocs, i+1)) & 2147483647
		if at < 0 || at+4 > len(out.code) || renvoAsmLabelPosition(out, label) < 0 {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 2004
			}
			if renvoRTGFailureDetail < 0 {
				renvoRTGFailureDetail = label
			}
		}
	}
	for i := 0; i+2 < len(out.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(out.absRelocs, i)) & 2147483647
		if at < 0 || at+4 > len(out.code) {
			if renvoRTGUnsupportedOperation == 0 {
				renvoRTGUnsupportedOperation = 2005
			}
		}
	}
}

// Prepared physical carriers use the selected descriptor registers.
func renvoRTGBeginObjectAggregateResult(a *renvoAsm, preserveSRet bool) bool {
	if renvoRTGStackWordBytes != 8 || !renvoRTGStack.Valid ||
		!renvoRTGPrimary.Valid || !renvoRTGSecondary.Valid {
		return false
	}
	renvoRTGAdjustObjectStack(a, true)
	renvoRTGAdjustObjectStack(a, true)
	if preserveSRet {
		if !renvoRTGCallWord0.Valid {
			return false
		}
		renvoRTGDirectStoreNative(a,
			renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1),
			renvoRTGCallWord0)
	}
	return true
}

func renvoRTGPushObjectSRetPointer(a *renvoAsm) bool {
	if !renvoRTGCallWord0.Valid {
		return false
	}
	renvoRTGAsmPushRegister(a, renvoRTGCallWord0)
	return true
}

func renvoRTGPushObjectPrivateResult(a *renvoAsm, argumentWords int) bool {
	if argumentWords < 0 || !renvoRTGStack.Valid || !renvoRTGPrimary.Valid {
		return false
	}
	address := renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister,
		argumentWords*renvoRTGStackWordBytes, 1)
	renvoRTGDirectAddress(a, renvoRTGPrimary, address)
	renvoRTGAsmPushRegister(a, renvoRTGPrimary)
	return true
}

func renvoRTGFinishObjectAggregateResult(a *renvoAsm, resultWords int) {
	renvoRTGDirectLoadNative(a, renvoRTGPrimary,
		renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister, 0, 1))
	if resultWords > 1 {
		renvoRTGDirectLoadNative(a, renvoRTGSecondary,
			renvoRTGAsmAddress(renvoRTGStack, RTGNoRegister,
				renvoRTGStackWordBytes, 1))
	}
	renvoRTGAdjustObjectStack(a, false)
	renvoRTGAdjustObjectStack(a, false)
}

func renvoRTGIEEEHostSyscall(g *renvoLinearGen, number int, addressCount int, offset0 int, offset1 int, offset2 int) {
	renvoRTGAsmAddressFrame(&g.asm, renvoRTGSyscallWord0, offset0)
	if addressCount > 1 {
		renvoRTGAsmAddressFrame(&g.asm, renvoRTGSyscallWord1, offset1)
	}
	if addressCount > 2 {
		renvoRTGAsmAddressFrame(&g.asm, renvoRTGSyscallWord2, offset2)
	}
	renvoRTGDirectMoveImmediate(&g.asm, renvoRTGSyscallNumber, int64(number))
	renvoRTGDirectHostSyscall(&g.asm)
}

func renvoRTGSaveSliceSlotAddresses(a *renvoAsm, dataSlot int, lenSlot int, capSlot int) {
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord0)
	renvoAsmStorePrimaryStack(a, dataSlot)
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord1)
	renvoAsmStorePrimaryStack(a, lenSlot)
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord5)
	renvoAsmStorePrimaryStack(a, capSlot)
}
