package main

// RenvoEmitPureBlock lowers the RFE uint64 state-transform contract through the
// existing native emitters. The caller owns executable memory and invocation.
// Each four-word record is (operation, left value, right value, immediate).
// Values are SSA record indices; operations 0..11 match the documented RFE IR.
// The native entry receives its state pointer in primary and makes no calls.
func RenvoEmitPureBlock(records []int, stateWords int, arm64 bool) ([]byte, bool) {
	if stateWords < 1 || stateWords > 256 || len(records) == 0 || len(records)%4 != 0 || len(records) > 8192 {
		return nil, false
	}
	count := len(records) / 4
	for i := 0; i < count; i++ {
		op, left, right, immediate := records[i*4], records[i*4+1], records[i*4+2], records[i*4+3]
		if op < 0 || op > 11 || (op == 1 || op == 2) && (immediate < 0 || immediate >= stateWords) {
			return nil, false
		}
		if op >= 2 && (left < 0 || left >= i || records[left*4] == 2) {
			return nil, false
		}
		if op >= 3 && op != 8 && op != 9 && (right < 0 || right >= i || records[right*4] == 2) {
			return nil, false
		}
	}
	arch := renvoArchAmd64
	if arm64 {
		arch = renvoArchAarch64
	}
	// Do not allocate the whole-program emitter's multi-megabyte reserves for a
	// small block. No global compiler options or legacy context are consulted.
	context := &renvoCompileContext{renvoTargetArch: arch, renvoTargetOS: renvoOSLinux, renvoNativeIntSize: 8, stripSymbols: true}
	g := renvoLinearGen{c: context, stackPeak: (count + 1) * 8}
	g.asm.c = context
	g.asm.code = make([]byte, 0, count*32+64)
	a := &g.asm
	frame := renvoEmitGlobalInitFrameStart(&g)
	renvoAsmStorePrimaryStack(a, 8)
	for i := 0; i < count; i++ {
		op, left, right, immediate := records[i*4], records[i*4+1], records[i*4+2], records[i*4+3]
		if op == 0 {
			renvoAsmPrimaryImm(a, immediate)
		} else if op == 1 {
			renvoAsmLoadSecondaryStack(a, 8)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, immediate*8)
		} else {
			renvoAsmLoadPrimaryStack(a, (left+2)*8)
			if op == 2 {
				renvoAsmLoadSecondaryStack(a, 8)
				renvoAsmStorePrimaryMemSecondaryDisp(a, immediate*8)
				continue
			}
			if op == 8 || op == 9 {
				if immediate < 0 || immediate >= 64 {
					renvoAsmPrimaryImm(a, 0)
				} else if op == 8 {
					renvoAsmShlPrimaryImm(a, immediate)
				} else {
					renvoAsmLogicalShiftPrimaryWordImm(a, immediate)
				}
			} else {
				renvoAsmLoadTertiaryStack(a, (right+2)*8)
				if op == 3 {
					renvoAsmAddPrimaryTertiary(a)
				} else if op == 4 {
					renvoAsmSubPrimaryTertiary(a)
				} else if op >= 5 && op <= 7 {
					operator := byte('&')
					if op == 6 {
						operator = '|'
					} else if op == 7 {
						operator = '^'
					}
					renvoAsmBitwisePrimaryTertiary(a, operator)
				} else {
					condition := 0x94 // equal
					if op == 11 {
						condition = 0x97
					} // right > left, unsigned
					renvoAsmCmpTertiaryPrimarySet(a, condition)
				}
			}
		}
		renvoAsmStorePrimaryStack(a, (i+2)*8)
	}
	renvoEmitGlobalInitFrameEnd(&g, frame)
	renvoAsmRet(a)
	renvoAsmPatch(a)
	return a.code, !a.patchFailed
}

const renvoAbsBssReloc = 1
const renvoImportReloc = 2

// Undefined object symbols occupy a disjoint virtual-global range while code
// is emitted. No storage is allocated at these offsets: the relocatable ELF
// writer maps them to undefined symbols and preserves any field displacement
// as the relocation addend.
const renvoObjectExternalBase = 536870912
const renvoObjectExternalStride = 1048576

const renvoObjectABIUnavailable = 0
const renvoObjectABISysV = 1
const renvoObjectABICdecl = 2

func renvoIsSysVObject(c *renvoCompileContext) bool {
	return c != nil && c.objectFile && c.renvoTargetOS == renvoOSLinux &&
		!targetIsKernelModule(c) && renvoTargetObjectCallABI(c) == renvoObjectABISysV
}

func renvoIsCdeclObject(c *renvoCompileContext) bool {
	return c != nil && c.objectFile && c.renvoTargetOS == renvoOSLinux &&
		!targetIsKernelModule(c) && renvoTargetObjectCallABI(c) == renvoObjectABICdecl
}

func renvoIsHostedObject(c *renvoCompileContext) bool {
	if renvoPreparedBackendActive != 0 && renvoRTGPreparedObject != 0 {
		return c != nil && c.objectFile && !targetIsKernelModule(c)
	}
	return renvoIsSysVObject(c) || renvoIsCdeclObject(c)
}

func renvoAsmAddExternalImportName(a *renvoAsm, name string) int {
	renvoNonNil(a)
	if name == "" {
		return -1
	}
	for i := 0; i+1 < len(a.kernelImportOffsets); i += 2 {
		start := a.kernelImportOffsets[i]
		end := a.kernelImportOffsets[i+1]
		if renvoBytesEqualText(a.kernelImportNames, start, end, name) {
			return i / 2
		}
	}
	start := len(a.kernelImportNames)
	for i := 0; i < len(name); i++ {
		a.kernelImportNames = append(a.kernelImportNames, name[i])
	}
	a.kernelImportOffsets = append(a.kernelImportOffsets, start)
	a.kernelImportOffsets = append(a.kernelImportOffsets, len(a.kernelImportNames))
	return len(a.kernelImportOffsets)/2 - 1
}

// These bodies are used by the host Go build. Self-hosted compilers lower the
// calls as arena intrinsics so large, phase-local scratch data can be reclaimed.
func renvo_runtime_ArenaMark() int { return 0 }

func renvo_runtime_ArenaReset(mark int) {}

func renvo_runtime_ArenaDiscard(start int, end int) {}

func renvo_runtime_ArenaDiscardBytes(value []byte) {}

func renvo_runtime_ArenaDiscardDecls(value []renvoDecl) {}

func renvo_runtime_ArenaDiscardFuncs(value []renvoFuncDecl) {}

// These internal intrinsics are only used after compiler code has established
// the corresponding index invariant. Host builds retain Go's checked access;
// self-hosted builds lower the calls to an explicitly unsafe load.
func renvo_runtime_UnsafeByteAt(data []byte, index int) byte { return data[index] }

func renvo_runtime_UnsafeInt32At(data []int32, index int) int32 { return data[index] }

func renvo_runtime_UnsafeIntAt(data []int, index int) int { return data[index] }

func renvoNonNil(values ...interface{}) {}

type renvoLabelRef struct {
	at    int
	label int
}

type renvoAbsRef struct {
	at   int
	off  int
	kind int
}

type renvoAsmSymbol struct {
	nameStart    int
	nameEnd      int
	label        int
	endLabel     int
	sectionStart int
	sectionEnd   int
	alignment    int
	binding      int
	visibility   int
}

type renvoObjectDataSymbol struct {
	nameStart, nameEnd                   int
	sectionStart, sectionEnd             int
	targetStart, targetEnd               int
	offset, size, storageSize, alignment int
	binding, visibility                  int
	initialized, value                   int
	valueStart, valueEnd                 int
	kind                                 int
}

type renvoObjectDataRelocation struct {
	offset, targetStart, targetEnd, typ, addend, codeTarget, codeLabel int
}

type renvoObjectExternal struct {
	offset, importID int
}

type renvoObjectFunctionRange struct {
	label, end, nameStart, nameEnd, sectionStart, sectionEnd, alignment int
}

// renvoReplSymbol describes one persistent package-global slot in a linked
// image. The table is appended outside the ELF load segments, so normal
// execution ignores it while an in-process linker can migrate live values into
// the next generation before its global initializers run.
type renvoReplSymbol struct {
	id         int
	offset     int
	size       int
	restoreOff int
}

type renvoStaticImport struct {
	dll  string
	name string
}

type renvoDarwinStaticImport struct {
	dylib string
	name  string
	label int
	used  bool
}

type renvoAsm struct {
	code                  []byte
	labelPos              []int32
	relocs                []int32
	absRelocs             []int32
	symbols               []renvoAsmSymbol
	symbolName            []byte
	staticImports         []renvoStaticImport
	darwinImports         []renvoDarwinStaticImport
	darwinImportLabels    []int
	darwinImportUsed      []bool
	openbsdSyscalls       []int
	kernelImportNames     []byte
	kernelImportOffsets   []int
	data                  []byte
	objectStrings         *renvoObjectStrings
	bssSize               int
	codeOffset            int
	dataOffset            int
	bssOffset             int
	lastPrimaryStoreEnd   int
	lastPrimaryStoreOff   int
	lastPrimaryLoad       int
	replSymbols           []renvoReplSymbol
	wasmLocalSlots        []int32
	c                     *renvoCompileContext
	patchFailed           bool
	syscallNumber         int
	syscallNumberKnown    bool
	staticCallParamCount  int
	staticCallParamKinds  [16]byte
	staticCallResultFloat int
	// Relocatable-object bookkeeping stays after the hot executable-image
	// fields so adding object support does not widen every core field access in
	// fixed-target compilers.
	objectData       []renvoObjectDataSymbol
	objectDataValues []byte
	objectFunctions  []renvoObjectFunctionRange
	objectDataRelocs []renvoObjectDataRelocation
	objectExternals  []renvoObjectExternal
	// Per-source-word destinations: nonnegative argument-register ordinals,
	// or -(outgoing stack byte offset + 1). Valid only during a static call.
	staticCallWordLocations []int
	staticCallStackBytes    int
}

// A backend object is an unpatched function fragment plus its local labels,
// relocations, absolute data references, and newly discovered call edges. The
// cache owns fixed storage allocated before an embedded compiler records its
// transient arena mark, so objects survive successive IDE builds without
// retaining whole frontend units.
type renvoObjectCacheEntry struct {
	used   bool
	target int
	fnA    int
	fnB    int
	keyA   int
	keyB   int
	data   []byte
}

// Object-cache indexing is absent from fixed-target command-line compilers.
// Keep its comparatively large set of slices behind one optional pointer so
// ordinary function generators stay compact.
type renvoObjectGenState struct {
	contextA      int
	contextB      int
	dataBase      int
	funcIdentityA []int
	funcIdentityB []int
	funcBuckets   []int
	funcNext      []int
}

type renvoObjectStrings struct {
	// Packed offset/length pairs for static strings referenced by cached code.
	refs []int
}

func renvoMakeByteScratch(capacity int) []byte {
	return make([]byte, 0, capacity)
}

func renvoMakeByteBuffer(length int) []byte {
	return make([]byte, length)
}

func renvoMakeIntScratch(capacity int) []int {
	return make([]int, 0, capacity)
}

const renvoStructuredHelperSignedDivide = 1
const renvoStructuredHelperMakeZero = 2
const renvoStructuredHelperFault = 3
const renvoStructuredHelperStringEqual = 4
const renvoStructuredHelperArenaAlloc = 5
const renvoStructuredHelperIndexAddress = 6
const renvoStructuredHelperBoundsCheck = 7
const renvoStructuredHelperNonNil = 8

func renvoQueueStructuredHelper(g *renvoLinearGen, kind int, arg int, label int) {
	if g.structuredHelperCount >= len(g.structuredHelperKinds) {
		if renvoRTGUnsupportedOperation == 0 {
			renvoRTGUnsupportedOperation = 4003
		}
		return
	}
	index := g.structuredHelperCount
	g.structuredHelperKinds[index] = kind
	g.structuredHelperArgs[index] = arg
	g.structuredHelperLabels[index] = label
	g.structuredHelperCount++
}

func renvoEmitStructuredHelper(g *renvoLinearGen, kind int, arg int, label int) bool {
	if renvoRTGStructuredFunctions != 0 {
		renvoRTGFunctionStart(&g.asm, label)
		renvoAsmMarkLabel(&g.asm, label)
		if kind == renvoStructuredHelperSignedDivide {
			renvoEmitSignedDivisionHelperBody(g, arg != 0)
		} else if kind == renvoStructuredHelperMakeZero {
			renvoEmitMakeZeroHelperBody(g)
		} else if kind == renvoStructuredHelperFault {
			renvoEmitUncaughtFaultHelperBody(g, arg != 0)
		} else if kind == renvoStructuredHelperStringEqual {
			renvoRTGEmitStringEqualHelperBody(g)
		} else if kind == renvoStructuredHelperArenaAlloc {
			renvoEmitArenaAllocHelperBody(g, arg != 0)
		} else if kind == renvoStructuredHelperIndexAddress {
			renvoEmitIndexAddressHelperBody(g, arg)
		} else if kind == renvoStructuredHelperBoundsCheck {
			renvoEmitBoundsCheckHelperBody(g)
		} else if kind == renvoStructuredHelperNonNil {
			renvoEmitNonNilCheckHelperBody(g, arg != 0)
		} else {
			return false
		}
		renvoRTGFunctionFinish(&g.asm)
		return true
	}
	return false
}

func renvoEmitAllQueuedFunctionsScratch(g *renvoLinearGen) bool {
	renvoNonNil(g)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		// Prepared backends are cached compiler products, so keep their complete
		// reachable layout stable even when expression traversal discovers the
		// same call graph in a different order.
		if renvoPreparedBackendActive != 0 {
			for i := queueIndex + 1; i < len(g.funcQueue); i++ {
				if g.funcQueue[i] < g.funcQueue[queueIndex] {
					g.funcQueue[i], g.funcQueue[queueIndex] = g.funcQueue[queueIndex], g.funcQueue[i]
				}
			}
		}
		fnIndex := g.funcQueue[queueIndex]
		if renvoDeferUnreadyQueuedClosure(g, fnIndex) {
			continue
		}
		if !renvoEmitScalarFunctionScratch(g, fnIndex) {
			renvoPrintFailedFunction(g, fnIndex)
			return false
		}
	}
	for helperIndex := 0; helperIndex < g.structuredHelperCount; helperIndex++ {
		if !renvoEmitStructuredHelper(g, g.structuredHelperKinds[helperIndex], g.structuredHelperArgs[helperIndex], g.structuredHelperLabels[helperIndex]) {
			renvoPrintErr("renvo: failed to emit structured helper ")
			renvoPrintIntErr(g.structuredHelperKinds[helperIndex])
			renvoPrintErr("\n")
			return false
		}
	}
	return true
}
func renvoDeferUnreadyQueuedClosure(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	closureIndex := renvoClosureIndexByFunction(g.meta, fnIndex)
	if closureIndex < 0 || g.meta.closures[closureIndex].ready {
		return false
	}
	// A whole-program function-value dispatch can discover a closure before
	// its reachable parent has established the capture layout. Make that
	// speculative queue entry available for the parent to enqueue again.
	g.funcReachable[fnIndex] = false
	return true
}

const renvoWasm32FallbackSliceBackingSize = 4096

const renvoLargeProgramSourceThreshold = 1048576

func renvoAsmInit(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmInitWithContext(a, renvoLegacyCompileContext())
}

// renvoAsmReserves is the definition-owned allocation plan for one assembler.
// Allocation remains shared so the self-host compiler materializes only one
// reserve per buffer, rather than a static ring for every target branch.
type renvoAsmReserves struct {
	code            int
	labels          int
	relocs          int
	absRelocs       int
	symbols         int
	data            int
	kernelImports   bool
	openbsdSyscalls bool
}

func renvoAsmInitWithContext(a *renvoAsm, context *renvoCompileContext) {
	renvoNonNil(a, context)
	a.c = context
	reserves := renvoAssemblerReserves(a)
	a.symbols = nil
	a.symbolName = nil
	a.staticImports = nil
	a.darwinImports = nil
	if reserves.symbols != 0 {
		a.symbols = make([]renvoAsmSymbol, 0, reserves.symbols)
	}
	a.data = make([]byte, 0, reserves.data)
	if !a.c.stripSymbols || renvoAsmNeedsFunctionSymbols(a) {
		a.symbolName = make([]byte, 0, 16384)
	}
	if reserves.kernelImports {
		a.kernelImportNames = make([]byte, 0, 1024)
		a.kernelImportOffsets = make([]int, 0, 128)
	}
	if reserves.openbsdSyscalls {
		a.openbsdSyscalls = make([]int, 0, 128)
	}
	if renvoFixedTarget == 0 && len(renvoObjectCacheEntries) != 0 {
		a.objectStrings = &renvoObjectStrings{refs: make([]int, 0, 2048)}
	}
	a.labelPos = make([]int32, 0, reserves.labels)
	a.relocs = make([]int32, 0, reserves.relocs)
	a.absRelocs = make([]int32, 0, reserves.absRelocs)
	a.code = make([]byte, 0, reserves.code)
	a.bssSize = 0
	a.codeOffset = 0
	a.dataOffset = 0
	a.bssOffset = 0
	a.lastPrimaryStoreEnd = -1
	a.lastPrimaryStoreOff = 0
	a.lastPrimaryLoad = 0
}

func renvoAsmNewLabel(a *renvoAsm) int {
	renvoNonNil(a)
	label := len(a.labelPos)
	a.labelPos = append(a.labelPos, -1)
	return label
}

func renvoAsmMarkLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if label < 0 {
		return
	}
	if label >= len(a.labelPos) {
		return
	}
	codeLen := len(a.code)
	a.labelPos[label] = int32(codeLen)
	a.lastPrimaryStoreEnd = -1
	a.lastPrimaryLoad = 0
	if renvoPreparedBackendActive != 0 {
		renvoRTGMarkLabel(a, label)
	}
}

func renvoAsmLabelPosition(a *renvoAsm, label int) int {
	renvoNonNil(a)
	if label < 0 || label >= len(a.labelPos) {
		return -1
	}
	return int(renvo_runtime_UnsafeInt32At(a.labelPos, label))
}

func renvoAsmEmit8(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = append(a.code, byte(v))
}

func renvoAsmEmitText(a *renvoAsm, code string) {
	renvoNonNil(a)
	for i := 0; i < len(code); i++ {
		a.code = append(a.code, code[i])
	}
}

func renvoAsmEmit2(a *renvoAsm, v0 int, v1 int) {
	renvoNonNil(a)
	a.code = renvoAppend16(a.code, v0|(v1<<8))
}

func renvoAsmEmit3(a *renvoAsm, v0 int, v1 int, v2 int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, v0|(v1<<8)|(v2<<16))
}

func renvoAsmEmit4(a *renvoAsm, v0 int, v1 int, v2 int, v3 int) {
	renvoNonNil(a)
	a.code = renvoAppend32(a.code, v0|(v1<<8)|(v2<<16)|(v3<<24))
}

func renvoAsmEmit5(a *renvoAsm, v0 int, v1 int, v2 int, v3 int, v4 int) {
	renvoNonNil(a)
	renvoAsmEmit4(a, v0, v1, v2, v3)
	renvoAsmEmit8(a, v4)
}

func renvoAsmAddAbsReloc(a *renvoAsm, at int, off int, kind int) {
	renvoNonNil(a)
	a.absRelocs = append(a.absRelocs, int32(at&2147483647), int32(off&2147483647), int32(kind&2147483647))
	// A relocation can be recorded without emitting another byte.  Treat it as
	// an adjacency barrier so push/pop peepholes can rely on their O(1) marker
	// instead of rescanning every relocation emitted so far.
	a.lastPrimaryStoreEnd = -1
	a.lastPrimaryLoad = 0
}

func renvoAsmAddReloc(a *renvoAsm, at int, label int) {
	renvoNonNil(a)
	a.relocs = append(a.relocs, int32(at&2147483647), int32(label&2147483647))
	a.lastPrimaryStoreEnd = -1
	a.lastPrimaryLoad = 0
}

func renvoAsmAddFuncSymbol(a *renvoAsm, src []byte, nameStart int, nameEnd int, label int) {
	renvoNonNil(a)
	if a.c.stripSymbols && !renvoAsmNeedsFunctionSymbols(a) {
		return
	}
	start := len(a.symbolName)
	for i := nameStart; i < nameEnd; i++ {
		a.symbolName = append(a.symbolName, renvo_runtime_UnsafeByteAt(src, i))
	}
	end := len(a.symbolName)
	var sym renvoAsmSymbol
	sym.nameStart = start
	sym.nameEnd = end
	sym.label = label
	a.symbols = append(a.symbols, sym)
}

func renvoAsmCopyObjectText(a *renvoAsm, src []byte, start int, end int) (int, int) {
	renvoNonNil(a)
	if start >= end || renvoBytesEqualText(src, start, end, "-") {
		return 0, 0
	}
	outStart := len(a.symbolName)
	for i := start; i < end; i++ {
		a.symbolName = append(a.symbolName, renvo_runtime_UnsafeByteAt(src, i))
	}
	return outStart, len(a.symbolName)
}

func renvoAsmCopyObjectPrefixedText(a *renvoAsm, prefix string, src []byte, start int, end int) (int, int) {
	renvoNonNil(a)
	outStart := len(a.symbolName)
	for i := 0; i < len(prefix); i++ {
		a.symbolName = append(a.symbolName, prefix[i])
	}
	for i := start; i < end; i++ {
		a.symbolName = append(a.symbolName, renvo_runtime_UnsafeByteAt(src, i))
	}
	return outStart, len(a.symbolName)
}

func renvoStringFromBytes(src []byte, start int, end int) string {
	value := string(src[start:end])
	if renvoFixedTarget == 0 {
		return renvo_runtime_ArenaPersistString(value)
	}
	return value
}

// ObjectImage is the RTG-visible bridge to the production x86_64 relocatable
// writer. A custom target still selects the object format in its definition;
// the implementation is shared with the compiled-in frontend so the two paths
// cannot drift on section, symbol, or relocation semantics.
func (a *renvoAsm) ObjectImage() []byte {
	if renvoFixedTarget != 0 {
		return nil
	}
	return renvoAsmImageRelocatableObjectAmd64(a)
}

func renvoAsmAddObjectFuncSymbol(a *renvoAsm, src []byte, nameStart int, nameEnd int, label int, decl *renvoObjectDecl) int {
	renvoAsmAddFuncSymbol(a, src, nameStart, nameEnd, label)
	index := len(a.symbols) - 1
	if index < 0 {
		return index
	}
	if decl == nil {
		a.symbols[index].binding = 1
		return index
	}
	sectionStart, sectionEnd := renvoAsmCopyObjectText(a, src, decl.sectionStart, decl.sectionEnd)
	a.symbols[index].sectionStart = sectionStart
	a.symbols[index].sectionEnd = sectionEnd
	a.symbols[index].alignment = decl.alignment
	a.symbols[index].binding = decl.binding
	a.symbols[index].visibility = decl.visibility
	return index
}

func renvoAsmEmit32(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = append(a.code, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func renvoFixedByteScratch(capacity int) []byte {
	if renvoFixedTarget != 0 {
		return make([]byte, 0, capacity)
	}
	var out []byte
	return out
}

func renvoFixedIntScratch(capacity int) []int {
	if renvoFixedTarget != 0 {
		if capacity <= 4 {
			capacity = 4
		} else if capacity <= 8 {
			capacity = 8
		}
		return make([]int, 0, capacity)
	}
	var out []int
	return out
}

func renvoFixedCompositeFieldScratch(capacity int) []renvoCompositeField {
	if renvoFixedTarget != 0 {
		if capacity <= 8 {
			capacity = 8
		}
		return make([]renvoCompositeField, 0, capacity)
	}
	var out []renvoCompositeField
	return out
}

func renvoAsmEmit64(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = renvoAppend64(a.code, v)
}

func renvoAsmEmit16(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = renvoAppend16(a.code, v)
}

func renvoAsmEmit24(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = append(a.code, byte(v))
	a.code = append(a.code, byte(v>>8))
	a.code = append(a.code, byte(v>>16))
}

// Patch signed 32-bit PC-relative data/BSS displacements after layout is fixed.
// Backend definitions choose when this relocation representation applies.
func renvoAsmPatchDataDisplacements32(a *renvoAsm) {
	renvoNonNil(a)
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) & 2147483647
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1)) & 2147483647
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2)) & 2147483647
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		next := a.codeOffset + at + 4
		disp := target - next
		renvoPut32At(a.code, at, disp)
	}
}

func renvoAsmBssOffset(a *renvoAsm) int {
	renvoNonNil(a)
	if a.bssOffset > 0 {
		return a.bssOffset
	}
	return a.dataOffset + len(a.data)
}

func renvoGet32At(in []byte, at int) int {
	return int(in[at]) | (int(in[at+1]) << 8) | (int(in[at+2]) << 16) | (int(in[at+3]) << 24)
}

func renvoPut32At(out []byte, at int, v int) {
	out[at] = byte(v)
	out[at+1] = byte(v >> 8)
	out[at+2] = byte(v >> 16)
	out[at+3] = byte(v >> 24)
}

func renvoAppend16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func renvoAppend32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func renvoAppend64(out []byte, v int) []byte {
	out = renvoAppend32(out, v)
	out = renvoAppend32(out, v>>32)
	return out
}

func renvoAppend64U32(out []byte, v int) []byte {
	out = renvoAppend32(out, v)
	out = renvoAppend32(out, 0)
	return out
}

type renvoElfSymbolSections struct {
	symtab     []byte
	strtab     []byte
	shstrtab   []byte
	symtabOff  int
	strtabOff  int
	shstrOff   int
	shoff      int
	textName   int
	dataName   int
	bssName    int
	symtabName int
	strtabName int
	shstrName  int
}

func renvoAlignValue(v int, align int) int {
	rem := v % align
	if rem == 0 {
		return v
	}
	return v + align - rem
}

func renvoAppendUntil(out []byte, size int) []byte {
	for len(out) < size {
		out = append(out, 0)
	}
	return out
}

func renvoAppendStringZ(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return out
}

// ELF class width is supplied by the image writer, independent of ISA identity
// and the language-level integer or pointer widths.
func renvoAppendElfShdr(wordSize int, out []byte, name int, typ int, flags int, addr int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = renvoAppend32(out, name)
	out = renvoAppend32(out, typ)
	if wordSize == 8 {
		out = renvoAppend64U32(out, flags)
		out = renvoAppend64U32(out, addr)
		out = renvoAppend64U32(out, off)
		out = renvoAppend64U32(out, size)
	} else {
		out = renvoAppend32(out, flags)
		out = renvoAppend32(out, addr)
		out = renvoAppend32(out, off)
		out = renvoAppend32(out, size)
	}
	out = renvoAppend32(out, link)
	out = renvoAppend32(out, info)
	if wordSize == 8 {
		out = renvoAppend64U32(out, align)
		out = renvoAppend64U32(out, entsize)
	} else {
		out = renvoAppend32(out, align)
		out = renvoAppend32(out, entsize)
	}
	return out
}

func renvoAppendElf64LoadProgram(out []byte, flags int, offset int, address int, fileSize int, memorySize int) []byte {
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, flags)
	out = renvoAppend64U32(out, offset)
	out = renvoAppend64U32(out, address)
	out = renvoAppend64U32(out, address)
	out = renvoAppend64U32(out, fileSize)
	out = renvoAppend64U32(out, memorySize)
	out = renvoAppend64U32(out, 0x1000)
	return out
}

func renvoAppendElf32LoadProgram(out []byte, flags int, offset int, address int, fileSize int, memorySize int) []byte {
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, offset)
	out = renvoAppend32(out, address)
	out = renvoAppend32(out, address)
	out = renvoAppend32(out, fileSize)
	out = renvoAppend32(out, memorySize)
	out = renvoAppend32(out, flags)
	out = renvoAppend32(out, 0x1000)
	return out
}

func renvoAppendElfSym(wordSize int, out []byte, name int, info int, shndx int, value int, size int) []byte {
	out = renvoAppend32(out, name)
	if wordSize == 8 {
		out = append(out, byte(info))
		out = append(out, 0)
		out = renvoAppend16(out, shndx)
		out = renvoAppend64U32(out, value)
		out = renvoAppend64U32(out, size)
		return out
	}
	out = renvoAppend32(out, value)
	out = renvoAppend32(out, size)
	out = append(out, byte(info))
	out = append(out, 0)
	out = renvoAppend16(out, shndx)
	return out
}

func renvoBuildElfSymbolSections(a *renvoAsm, wordSize int, base int, entryOff int, loadFileSize int, sec *renvoElfSymbolSections) {
	renvoNonNil(a, sec)
	entrySize := wordSize*2 + 8
	sec.symtab = make([]byte, 0, (len(a.symbols)+2)*entrySize)
	sec.strtab = make([]byte, 0, len(a.symbolName)+16)
	sec.shstrtab = make([]byte, 0, 64)
	sectionNames := "\x00.text\x00.rodata\x00.bss\x00.symtab\x00.strtab\x00.shstrtab\x00"
	sec.shstrtab = append(sec.shstrtab, sectionNames...)
	sec.textName = 1
	sec.dataName = 7
	sec.bssName = 15
	sec.symtabName = 20
	sec.strtabName = 28
	sec.shstrName = 36
	sec.strtab = append(sec.strtab, "\x00_start\x00"...)
	startName := 1
	sec.symtab = renvoAppendElfSym(wordSize, sec.symtab, 0, 0, 0, 0, 0)
	sec.symtab = renvoAppendElfSym(wordSize, sec.symtab, startName, 18, 1, base+entryOff, 0)
	for i := 0; i < len(a.symbols); i++ {
		s := a.symbols[i]
		label := s.label
		position := renvoAsmLabelPosition(a, label)
		if position < 0 {
			continue
		}
		nameOff := len(sec.strtab)
		for i := s.nameStart; i < s.nameEnd; i++ {
			sec.strtab = append(sec.strtab, a.symbolName[i])
		}
		sec.strtab = append(sec.strtab, 0)
		value := base + a.codeOffset + position
		sec.symtab = renvoAppendElfSym(wordSize, sec.symtab, nameOff, 18, 1, value, 0)
	}

	sec.symtabOff = renvoAlignValue(loadFileSize, wordSize)
	sec.strtabOff = sec.symtabOff + len(sec.symtab)
	sec.shstrOff = sec.strtabOff + len(sec.strtab)
	sec.shoff = renvoAlignValue(sec.shstrOff+len(sec.shstrtab), wordSize)
}

func renvoAppendElfSectionHeaders(out []byte, sec *renvoElfSymbolSections, a *renvoAsm, wordSize int, base int) []byte {
	renvoNonNil(sec, a)

	out = renvoAppendElfShdr(wordSize, out, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	out = renvoAppendElfShdr(wordSize, out, sec.textName, 1, 6, base+a.codeOffset, a.codeOffset, len(a.code), 0, 0, 16, 0)
	out = renvoAppendElfShdr(wordSize, out, sec.dataName, 1, 2, base+a.dataOffset, a.dataOffset, len(a.data), 0, 0, wordSize, 0)
	out = renvoAppendElfShdr(wordSize, out, sec.bssName, 8, 3, base+renvoAsmBssOffset(a), renvoAsmBssOffset(a), a.bssSize, 0, 0, wordSize, 0)
	out = renvoAppendElfShdr(wordSize, out, sec.symtabName, 2, 0, 0, sec.symtabOff, len(sec.symtab), 5, 1, wordSize, wordSize*2+8)
	out = renvoAppendElfShdr(wordSize, out, sec.strtabName, 3, 0, 0, sec.strtabOff, len(sec.strtab), 0, 0, 1, 0)
	out = renvoAppendElfShdr(wordSize, out, sec.shstrName, 3, 0, 0, sec.shstrOff, len(sec.shstrtab), 0, 0, 1, 0)
	return out
}

const renvoTokEOF = 0
const renvoTokIdent = 1
const renvoTokNumber = 2
const renvoTokFloat = 3
const renvoTokString = 4
const renvoTokChar = 5
const renvoTokPackage = 6
const renvoTokConst = 7
const renvoTokVar = 8
const renvoTokType = 9
const renvoTokFunc = 10
const renvoTokStruct = 11
const renvoTokReturn = 12
const renvoTokIf = 13
const renvoTokElse = 14
const renvoTokFor = 15
const renvoTokBreak = 16
const renvoTokContinue = 17
const renvoTokGoto = 18
const renvoTokSwitch = 19
const renvoTokCase = 20
const renvoTokDefault = 21
const renvoTokOp = 22

type renvoTokens struct {
	data         []int32
	lineBases    []int32
	count        int
	panicEnabled bool
}

type renvoToken struct {
	start int
	end   int
}

const renvoTokenStride = 2

func renvoTokCount(p *renvoProgram) int {
	renvoNonNil(p)
	return p.toks.count
}

func renvoTokKind(p *renvoProgram, i int) int {
	renvoNonNil(p)
	return int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride)) & 255
}

func renvoTokStart(p *renvoProgram, i int) int {
	renvoNonNil(p)
	return int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride+1)) & 0xffffff
}

func renvoTokEnd(p *renvoProgram, i int) int {
	renvoNonNil(p)
	base := i * renvoTokenStride
	first := int(renvo_runtime_UnsafeInt32At(p.toks.data, base))
	packed := int(renvo_runtime_UnsafeInt32At(p.toks.data, base+1))
	start := packed & 0xffffff
	size := packed >> 24 & 255
	if first&255 != renvoTokOp {
		size |= first >> 16 & 0xff00
	}
	return start + size
}

func renvoTokLine(p *renvoProgram, i int) int {
	renvoNonNil(p)
	data := p.toks.data
	base := i * renvoTokenStride
	line := int(renvo_runtime_UnsafeInt32At(data, base)) >> 8 & 65535
	for at := len(p.toks.lineBases) - 1; at >= 0; at-- {
		packed := int(renvo_runtime_UnsafeInt32At(p.toks.lineBases, at))
		if i >= packed&0xffffff {
			line += packed >> 8 & 0xff0000
			break
		}
	}
	return line
}

func renvoTokAt(p *renvoProgram, i int) renvoToken {
	renvoNonNil(p)
	var tok renvoToken
	base := i * renvoTokenStride
	first := int(renvo_runtime_UnsafeInt32At(p.toks.data, base))
	packed := int(renvo_runtime_UnsafeInt32At(p.toks.data, base+1))
	tok.start = packed & 0xffffff
	size := packed >> 24 & 255
	if first&255 != renvoTokOp {
		size |= first >> 16 & 0xff00
	}
	tok.end = tok.start + size
	return tok
}

type renvoDecl struct {
	kind      int
	nameStart int
	nameEnd   int
	startTok  int
	endTok    int
}

type renvoFuncDecl struct {
	nameStart     int
	nameEnd       int
	startTok      int
	nameTok       int
	receiverStart int
	receiverEnd   int
	bodyStart     int
	bodyEnd       int
	endTok        int
}

type renvoPackageInfo struct {
	graphKeyA  int
	graphKeyB  int
	sourceKeyA int
	sourceKeyB int
	textStart  int
	textEnd    int
	tokenStart int
	tokenEnd   int
	declStart  int
	declEnd    int
	funcStart  int
	funcEnd    int
	pathKeyA   int
	pathKeyB   int
}

type renvoPackageTable struct {
	items []renvoPackageInfo
}

type renvoProgram struct {
	src           []byte
	toks          renvoTokens
	decls         []renvoDecl
	funcs         []renvoFuncDecl
	packageTable  *renvoPackageTable
	ok            bool
	parsedIntHigh int
	compilerInt32 bool
	c             renvoCompileContext
	c11Semantics  bool
	entryFunc     int
	foreign       *renvoForeignProgram
}

type renvoForeignProgram struct {
	next        *renvoForeignProgram
	global      int
	artifact    []byte
	entryOffset int
}

func renvoProgramPackages(p *renvoProgram) []renvoPackageInfo {
	if p.packageTable == nil {
		return nil
	}
	return p.packageTable.items
}

const renvoExprBad = 0
const renvoExprIdent = 1
const renvoExprInt = 2
const renvoExprFloat = 3
const renvoExprString = 4
const renvoExprChar = 5
const renvoExprBool = 6
const renvoExprUnary = 7
const renvoExprBinary = 8
const renvoExprCall = 9
const renvoExprIndex = 10
const renvoExprSelector = 11
const renvoExprComposite = 12
const renvoExprSlice = 13
const renvoExprFunc = 14
const renvoExprAssert = 15

const renvoStmtBad = 0
const renvoStmtReturn = 1
const renvoStmtIf = 2
const renvoStmtFor = 3
const renvoStmtBreak = 4
const renvoStmtContinue = 5
const renvoStmtGoto = 6
const renvoStmtLabel = 7
const renvoStmtVar = 8
const renvoStmtShort = 9
const renvoStmtAssign = 10
const renvoStmtExpr = 11
const renvoStmtSwitch = 12
const renvoStmtBlock = 13
const renvoStmtType = 14
const renvoStmtDefer = 15

type renvoExpr struct {
	kind      int
	tok       int
	left      int
	right     int
	firstArg  int
	argCount  int
	nameStart int
	nameEnd   int
	inferred  int
}

type renvoExprParse struct {
	prog     *renvoProgram
	pos      int
	end      int
	exprs    []renvoExpr
	args     []int
	fields   []renvoCompositeField
	ok       bool
	hasFloat bool
}

func renvoNewExprParse() *renvoExprParse {
	ep := new(renvoExprParse)
	return ep
}

type renvoCompositeField struct {
	nameStart int
	nameEnd   int
	key       int
	expr      int
}

type renvoStmt struct {
	kind      int
	startTok  int
	endTok    int
	exprStart int
	exprEnd   int
	bodyStart int
	bodyEnd   int
	elseStart int
	elseEnd   int
	nameStart int
	nameEnd   int
}

type renvoBodyParse struct {
	prog      *renvoProgram
	stmtCount int
	ok        bool
	stmt      renvoStmt
}

const renvoTypeInvalid = 0
const renvoTypeInt = 1
const renvoTypeInt64 = 2
const renvoTypeByte = 3
const renvoTypeBool = 4
const renvoTypeString = 5
const renvoTypeFloat64 = 6
const renvoTypeInt8 = 7
const renvoTypeInt16 = 8
const renvoTypeInt32 = 9
const renvoTypePointer = 10
const renvoTypeSlice = 11
const renvoTypeStruct = 12
const renvoTypeNamed = 13
const renvoTypeArray = 14

// Type ID 15 is intentionally unused. Maps are lowered by the frontend.
const renvoTypeUint16 = 16
const renvoTypeUint32 = 17
const renvoTypeUint64 = 18
const renvoTypeFunc = 19
const renvoTypeInterface = 20
const renvoTypeComplex = 21
const renvoTypeFloat32 = 22
const renvoTypeComplex64 = 23
const renvoNamedTypeAlias = 1
const renvoStructLayoutMarker = -1
const renvoStructLayoutHost = -2
const renvoStructLayoutDense = -3

const renvoBuiltinTypeUint16 = 10
const renvoBuiltinTypeUint32 = 11
const renvoBuiltinTypeUint64 = 12
const renvoBuiltinTypeComplex = 13
const renvoBuiltinTypeInterface = 14
const renvoBuiltinTypeFloat32 = 15
const renvoBuiltinTypeComplex64 = 16
const renvoBuiltinTypeError = 17

type renvoTypeInfo struct {
	kind        int
	elem        int
	first       int
	count       int
	size        int
	nativeAlign int
	resolved    int
	nameStart   int
	nameEnd     int
}

type renvoFieldInfo struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
	embedded  bool
}

type renvoSymbolInfo struct {
	nameStart    int
	nameEnd      int
	kind         int
	typ          int
	initStart    int
	initEnd      int
	iotaValue    int // const iota value; variable BSS offset during initialization
	constValue   int
	constValueOK int // const validity; variable initialization walk state
	objectDecl   int
}

type renvoFuncInfo struct {
	declIndex       int
	nameStart       int
	nameEnd         int
	firstParam      int
	paramCount      int
	firstResult     int
	resultCount     int
	resultType      int
	receiverType    int
	bodyStart       int
	bodyEnd         int
	linkStatic      int
	linkDLLStart    int
	linkDLLEnd      int
	linkMethodStart int
	linkMethodEnd   int
	literalTok      int // positive for a literal; negative after named-function init scanning
	exportNameStart int
	exportNameEnd   int
	objectDecl      int
}

type renvoObjectDecl struct {
	kind                                       int
	nameStart, nameEnd                         int
	sectionStart, sectionEnd                   int
	alignment, binding, visibility             int
	targetStart, targetEnd                     int
	size                                       int
	relocationKind                             int
	relocationTargetStart, relocationTargetEnd int
	relocationAddend                           int
}

type renvoClosureInfo struct {
	fnIndex      int
	firstCapture int
	captureCount int
	ready        bool
}

type renvoDeferSite struct {
	funcType int
	// A direct deferred call has one statically known target. Keeping that
	// identity avoids constructing a signature-wide function-value dispatch,
	// which would otherwise make unrelated functions reachable.
	directTarget int
}

type renvoMeta struct {
	typeIndexVersion int
	runtimeTypeCount int
	prog             *renvoProgram
	types            []renvoTypeInfo
	fields           []renvoFieldInfo
	globals          []renvoSymbolInfo
	params           []renvoSymbolInfo
	funcs            []renvoFuncInfo
	globalBuckets    []int32
	globalNext       []int32
	funcBuckets      []int32
	funcNext         []int32
	typeBuckets      []int32
	closures         []renvoClosureInfo
	captures         []renvoSymbolInfo
	panicEnabled     bool
	arenaSize        int
	scratchStart     int
	scratchEnd       int
	ok               bool
	c                *renvoCompileContext
	objectDecls      []renvoObjectDecl
}

type renvoCompileResult struct {
	data []byte
	ok   bool
}

type renvoConstResult struct {
	value int
	ok    bool
}

type renvoTypeResult struct {
	typ  int
	next int
}

const renvoIdentAppend = 1
const renvoIdentByteSlice = 2
const renvoIdentMake = 3
const renvoIdentInt = 5
const renvoIdentInt64 = 6
const renvoIdentByte = 7
const renvoIdentLen = 8
const renvoIdentOpen = 9
const renvoIdentClose = 10
const renvoIdentRead = 11
const renvoIdentWrite = 12
const renvoIdentChmod = 13
const renvoIdentCopy = 14
const renvoIdentInt16 = 15
const renvoIdentInt32 = 16
const renvoIdentSyscall = 17
const renvoIdentString = 18
const renvoIdentCap = 19
const renvoIdentPanic = 20
const renvoIdentInt8 = 21
const renvoIdentUint16 = 22
const renvoIdentUint32 = 23
const renvoIdentUint64 = 24
const renvoIdentDelete = 25
const renvoIdentNew = 26
const renvoIdentRecover = 27
const renvoIdentPrintln = 28
const renvoIdentReal = 29
const renvoIdentImag = 30
const renvoIdentComplex = 31

func renvoProgramError(p *renvoProgram) {
	renvoNonNil(p)
	p.ok = false
}

func renvoMetaError(m *renvoMeta) {
	renvoNonNil(m)
	m.ok = false
}

func renvoExprError(ep *renvoExprParse) {
	renvoNonNil(ep)
	ep.ok = false
}

func renvoParseProgram(src []byte) renvoProgram {
	var p renvoProgram
	p.c.stripSymbols = renvoCompilerStripSymbols
	if renvoFixedTarget == 0 {
		p.c.renvoTarget = renvoTarget
		p.c.renvoTargetOS = renvoTargetOS
		p.c.renvoTargetArch = renvoTargetArch
		p.c.renvoNativeIntSize = renvoNativeIntSize
		p.c.windowsSubsystem = renvoCompilerWindowsSubsystem
		p.c.emitImage = renvoCompilerEmitImage
	} else if targetIsWindows(renvoTargetOS) {
		p.c.windowsSubsystem = renvoCompilerWindowsSubsystem
	}
	renvoParseProgramInto(src, &p)
	return p
}

func renvoParseProgramWithContext(src []byte, context *renvoCompileContext) renvoProgram {
	var p renvoProgram
	renvoNonNil(context)
	p.c = *context
	renvoParseProgramInto(src, &p)
	return p
}

func renvoSourceHasC11Directive(src []byte) bool {
	prefix := "// renvo:c11"
	for start := 0; start < len(src); {
		end := start
		for _, c := range src[start:] {
			if c == '\n' || c == '\r' {
				break
			}
			end++
		}
		if end-start == len(prefix) && renvoBytesEqualText(src, start, end, prefix) {
			return true
		}
		start = end + 1
	}
	return false
}

func renvoProgramUsesC11Semantics(p *renvoProgram) bool {
	renvoNonNil(p)
	return p.c11Semantics
}

func renvoSetCompilerIntWidth(p *renvoProgram) {
	probe := uint32(len(p.src)) | 2147483648
	p.compilerInt32 = int(probe) < 0
}

func renvoParseProgramInto(src []byte, p *renvoProgram) {
	renvoNonNil(p)
	p.entryFunc = -1
	p.src = src
	p.c11Semantics = renvoSourceHasC11Directive(src)
	renvoSetCompilerIntWidth(p)
	renvoScan(src, &p.toks)
	declCap := len(src)/1024 + 64
	funcCap := len(src)/768 + 64
	p.decls = make([]renvoDecl, 0, declCap)
	p.funcs = make([]renvoFuncDecl, 0, funcCap)
	p.ok = true
	tokenCount := renvoTokCount(p)

	i := 0
	if !renvoTokIsKind(p, i, renvoTokPackage) {
		renvoProgramError(p)
		return
	}
	i++
	if !renvoTokIsKind(p, i, renvoTokIdent) {
		renvoProgramError(p)
		return
	}
	i++

	for i < tokenCount && renvoTokKind(p, i) != renvoTokEOF {
		if renvoTokIsKind(p, i, renvoTokPackage) {
			i++
			if !renvoTokIsKind(p, i, renvoTokIdent) {
				renvoProgramError(p)
				return
			}
			i++
			continue
		}
		if renvoTokIsKind(p, i, renvoTokConst) || renvoTokIsKind(p, i, renvoTokVar) || renvoTokIsKind(p, i, renvoTokType) {
			start := i
			kind := int(renvoTokKind(p, i))
			i++
			if renvoTokCharIs(p, i, '(') {
				end := renvoSkipBalanced(p, i, '(', ')')
				if end <= i {
					renvoProgramError(p)
					return
				}
				var decl renvoDecl
				decl.kind = kind
				decl.nameStart = int(renvoTokStart(p, start))
				decl.nameEnd = int(renvoTokEnd(p, start))
				decl.startTok = start
				decl.endTok = end
				p.decls = append(p.decls, decl)
				i = end
				continue
			}
			if !renvoTokIsKind(p, i, renvoTokIdent) {
				renvoProgramError(p)
				return
			}
			name := renvoTokAt(p, i)
			i++
			end := renvoSkipTopLevelLine(p, i)
			var decl renvoDecl
			decl.kind = kind
			decl.nameStart = int(name.start)
			decl.nameEnd = int(name.end)
			decl.startTok = start
			decl.endTok = end
			p.decls = append(p.decls, decl)
			i = end
			continue
		}
		if renvoTokIsKind(p, i, renvoTokFunc) {
			var fn renvoFuncDecl
			renvoParseFuncDecl(p, i, &fn)
			if fn.endTok <= i {
				renvoProgramError(p)
				return
			}
			if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "appMain") {
				p.entryFunc = len(p.funcs)
			}
			p.funcs = append(p.funcs, fn)
			i = fn.endTok
			continue
		}
		i++
	}
}

func renvoParseFuncDecl(p *renvoProgram, start int, fn *renvoFuncDecl) {
	renvoNonNil(p, fn)
	fn.startTok = start
	tokenCount := renvoTokCount(p)
	i := start + 1
	if !renvoTokIsKind(p, i, renvoTokIdent) {
		receiverEnd := i + 1
		for receiverEnd < tokenCount && !renvoTokCharIs(p, receiverEnd, ')') {
			receiverEnd++
		}
		if receiverEnd <= i {
			return
		}
		fn.receiverStart = i + 1
		fn.receiverEnd = receiverEnd
		i = receiverEnd + 1
	}
	if !renvoTokIsKind(p, i, renvoTokIdent) {
		return
	}
	fn.nameTok = i
	fn.nameStart = int(renvoTokStart(p, i))
	fn.nameEnd = int(renvoTokEnd(p, i))
	i++
	i = renvoFindStatementBodyOpen(p, i, tokenCount)
	if !renvoTokCharIs(p, i, '{') {
		return
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < tokenCount && depth > 0 {
		c := renvoTokSingleChar(p, i)
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		}
		i++
	}
	if depth != 0 {
		return
	}
	fn.bodyEnd = i - 1
	fn.endTok = i
}

func renvoSkipBalanced(p *renvoProgram, start int, open byte, close byte) int {
	renvoNonNil(p)
	if renvoTokSingleChar(p, start) != open {
		return start
	}
	depth := 1
	i := start + 1
	tokenCount := renvoTokCount(p)
	for i < tokenCount && depth > 0 {
		first := int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride))
		c := byte(first >> 24)
		if first&255 != renvoTokOp {
			c = 0
		}
		if c == open {
			depth++
		} else if c == close {
			depth--
		}
		i++
	}
	if depth != 0 {
		return start
	}
	return i
}

func renvoSkipTopLevelLine(p *renvoProgram, start int) int {
	renvoNonNil(p)
	tokenCount := renvoTokCount(p)
	if start >= tokenCount {
		return start
	}
	line := renvoTokLine(p, start-1)
	i := start
	depth := 0
	for i < tokenCount {
		if renvoTokKind(p, i) == renvoTokEOF {
			return i
		}
		if renvoTokLine(p, i) != line && depth == 0 {
			return i
		}
		c := renvoTokSingleChar(p, i)
		if c == '{' || c == '(' {
			depth++
		} else if c == '}' || c == ')' {
			depth--
		}
		i++
	}
	return i
}

func renvoScan(src []byte, toks *renvoTokens) {
	renvoNonNil(toks)
	srcLen := len(src)
	tokenCap := 524288
	if renvoFixedTarget != 0 {
		tokenCap = srcLen/4 + 8192
		if renvoFixedTarget == renvoTargetWasiWasm32 {
			tokenCap = srcLen/5 + 16384
		}
	}
	toks.data = make([]int32, 0, tokenCap*renvoTokenStride)
	i := 0
	line := 1
	for i < srcLen {
		c := renvo_runtime_UnsafeByteAt(src, i)
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			for {
				if c == '\n' {
					line++
				}
				i++
				if i >= srcLen {
					break
				}
				c = renvo_runtime_UnsafeByteAt(src, i)
				if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
					break
				}
			}
			if i >= srcLen {
				continue
			}
		}
		if c == '/' && i+1 < srcLen {
			next := renvo_runtime_UnsafeByteAt(src, i+1)
			if next == '/' {
				i += 2
				for i < srcLen && renvo_runtime_UnsafeByteAt(src, i) != '\n' {
					i++
				}
				continue
			}
			if next == '*' {
				i += 2
				for i+1 < srcLen && !(renvo_runtime_UnsafeByteAt(src, i) == '*' && renvo_runtime_UnsafeByteAt(src, i+1) == '/') {
					if renvo_runtime_UnsafeByteAt(src, i) == '\n' {
						line++
					}
					i++
				}
				if i+1 < srcLen {
					i += 2
				}
				continue
			}
		}
		lower := c | 32
		if lower-'a' <= 'z'-'a' || c == '_' {
			i++
			start := i - 1
			for i < srcLen {
				cc := renvo_runtime_UnsafeByteAt(src, i)
				lower = cc | 32
				if !(lower-'a' <= 'z'-'a' || cc-'0' <= 9 || cc == '_') {
					break
				}
				i++
			}
			renvoScanAppendToken(toks, renvoKeywordKind(src, start, i, toks), start, i-start, line)
			continue
		}
		if c-'0' <= 9 {
			start := i
			kind := renvoTokNumber
			if c == '0' && i+1 < srcLen && (renvo_runtime_UnsafeByteAt(src, i+1) == 'x' || renvo_runtime_UnsafeByteAt(src, i+1) == 'X' || renvo_runtime_UnsafeByteAt(src, i+1) == 'b' || renvo_runtime_UnsafeByteAt(src, i+1) == 'B') {
				hex := renvo_runtime_UnsafeByteAt(src, i+1) == 'x' || renvo_runtime_UnsafeByteAt(src, i+1) == 'X'
				i += 2
				for i < srcLen {
					cc := renvo_runtime_UnsafeByteAt(src, i)
					lower = cc | 32
					if cc == '.' && hex {
						kind = renvoTokFloat
						i++
						continue
					}
					if hex && (cc == 'p' || cc == 'P') {
						kind = renvoTokFloat
						i++
						if i < srcLen && (renvo_runtime_UnsafeByteAt(src, i) == '+' || renvo_runtime_UnsafeByteAt(src, i) == '-') {
							i++
						}
						for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
							i++
						}
						break
					}
					if !(lower-'a' <= 'z'-'a' || cc-'0' <= 9 || cc == '_') {
						break
					}
					i++
				}
			} else {
				i++
				for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
					i++
				}
				if i < srcLen && renvo_runtime_UnsafeByteAt(src, i) == '.' {
					kind = renvoTokFloat
					i++
					for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
						i++
					}
				}
				if i < srcLen && (renvo_runtime_UnsafeByteAt(src, i) == 'e' || renvo_runtime_UnsafeByteAt(src, i) == 'E') {
					kind = renvoTokFloat
					i++
					if i < srcLen && (renvo_runtime_UnsafeByteAt(src, i) == '+' || renvo_runtime_UnsafeByteAt(src, i) == '-') {
						i++
					}
					for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
						i++
					}
				}
			}
			if i < srcLen && renvo_runtime_UnsafeByteAt(src, i) == 'i' {
				kind = renvoTokFloat
				i++
			}
			renvoScanAppendToken(toks, kind, start, i-start, line)
			continue
		}
		leadingDotFloat := false
		if c == '.' && i+1 < srcLen {
			next := renvo_runtime_UnsafeByteAt(src, i+1)
			leadingDotFloat = next >= '0' && next <= '9'
		}
		if leadingDotFloat {
			start := i
			i += 2
			for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
				i++
			}
			if i < srcLen && (renvo_runtime_UnsafeByteAt(src, i) == 'e' || renvo_runtime_UnsafeByteAt(src, i) == 'E') {
				i++
				if i < srcLen && (renvo_runtime_UnsafeByteAt(src, i) == '+' || renvo_runtime_UnsafeByteAt(src, i) == '-') {
					i++
				}
				for i < srcLen && (renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 || renvo_runtime_UnsafeByteAt(src, i) == '_') {
					i++
				}
			}
			if i < srcLen && renvo_runtime_UnsafeByteAt(src, i) == 'i' {
				i++
			}
			renvoScanAppendToken(toks, renvoTokFloat, start, i-start, line)
			continue
		}
		if c == '"' || c == '`' {
			start := i
			startLine := line
			i++
			for i < srcLen && renvo_runtime_UnsafeByteAt(src, i) != c {
				if c == '"' && renvo_runtime_UnsafeByteAt(src, i) == '\\' && i+1 < srcLen {
					i += 2
				} else {
					if renvo_runtime_UnsafeByteAt(src, i) == '\n' {
						line++
					}
					i++
				}
			}
			if i < srcLen {
				i++
			}
			renvoScanAppendToken(toks, renvoTokString, start, i-start, startLine)
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < srcLen && renvo_runtime_UnsafeByteAt(src, i) != '\'' {
				if renvo_runtime_UnsafeByteAt(src, i) == '\\' && i+1 < srcLen {
					i += 2
				} else {
					i++
				}
			}
			if i < srcLen {
				i++
			}
			renvoScanAppendToken(toks, renvoTokChar, start, i-start, line)
			continue
		}
		start := i
		i++
		if i < srcLen {
			c1 := renvo_runtime_UnsafeByteAt(src, i)
			two := c1 == '=' && (c == '^' || c == '|' || c >= '!' && c <= '>' && (0x3a005631>>(c-'!'))&1 != 0) || c == c1 && (c == '|' || c >= '&' && c <= '>' && (0x14000a1>>(c-'&'))&1 != 0) || c == '&' && c1 == '^'
			if two {
				i++
				if i < srcLen && renvo_runtime_UnsafeByteAt(src, i) == '=' && (c == '<' || c == '>' || c1 == '^') {
					i++
				}
			}
		}
		size := i - start
		charBits := 0
		if size == 1 {
			charBits = int(renvo_runtime_UnsafeByteAt(src, start)) << 24
		}
		if c == '(' && len(toks.data) >= renvoTokenStride && byte(int(toks.data[len(toks.data)-renvoTokenStride])>>24) == '.' {
			toks.panicEnabled = true
		}
		renvoScanAppendToken(toks, renvoTokOp|charBits, start, size, line)
	}
	renvoScanAppendToken(toks, renvoTokEOF, srcLen, 0, line)
}

func renvoScanAppendToken(toks *renvoTokens, kind int, start int, size int, line int) {
	renvoNonNil(toks)
	sizeHigh := 0
	if kind&255 != renvoTokOp {
		sizeHigh = size >> 8 << 24
	}
	lineHigh := line >> 16
	if lineHigh != 0 && (len(toks.lineBases) == 0 || lineHigh != int(toks.lineBases[len(toks.lineBases)-1])>>24&255) {
		toks.lineBases = append(toks.lineBases, int32(toks.count&0xffffff|lineHigh<<24))
	}
	toks.data = append(toks.data, int32(kind|(line&65535)<<8|sizeHigh), int32(start&0xffffff|(size&255)<<24))
	toks.count++
}

func renvoKeywordKind(src []byte, start int, end int, toks *renvoTokens) int {
	renvoNonNil(toks)
	n := end - start
	if n > 8 {
		return renvoTokIdent
	}
	h := 0
	for i := start; i < end; i++ {
		h = h*5 + int(renvo_runtime_UnsafeByteAt(src, i))
	}
	if n == 2 {
		if h == 627 && renvoBytesEqualText(src, start, end, "if") {
			return renvoTokIf
		}
	}
	if n == 3 {
		if h == 3549 && renvoBytesEqualText(src, start, end, "var") {
			return renvoTokVar
		}
		if h == 3219 && renvoBytesEqualText(src, start, end, "for") {
			return renvoTokFor
		}
	}
	if n == 4 {
		if h == 18186 && renvoBytesEqualText(src, start, end, "type") {
			return renvoTokType
		}
		if h == 16324 && renvoBytesEqualText(src, start, end, "func") {
			return renvoTokFunc
		}
		if h == 16001 && renvoBytesEqualText(src, start, end, "else") {
			return renvoTokElse
		}
		if h == 16341 && renvoBytesEqualText(src, start, end, "goto") {
			return renvoTokGoto
		}
		if h == 15476 && renvoBytesEqualText(src, start, end, "case") {
			return renvoTokCase
		}
	}
	if n == 5 {
		if (h == 78294 && renvoBytesEqualText(src, start, end, "defer")) || (h == 85499 && renvoBytesEqualText(src, start, end, "panic")) {
			toks.panicEnabled = true
		}
		if h == 79191 && renvoBytesEqualText(src, start, end, "const") {
			return renvoTokConst
		}
		if h == 78617 && renvoBytesEqualText(src, start, end, "break") {
			return renvoTokBreak
		}
	}
	if n == 6 {
		if h == 449661 && renvoBytesEqualText(src, start, end, "struct") {
			return renvoTokStruct
		}
		if h == 437480 && renvoBytesEqualText(src, start, end, "return") {
			return renvoTokReturn
		}
		if h == 450374 && renvoBytesEqualText(src, start, end, "switch") {
			return renvoTokSwitch
		}
	}
	if n == 7 {
		if h == 2176194 && renvoBytesEqualText(src, start, end, "recover") {
			toks.panicEnabled = true
		}
		if h == 2131416 && renvoBytesEqualText(src, start, end, "package") {
			return renvoTokPackage
		}
		if h == 1957581 && renvoBytesEqualText(src, start, end, "default") {
			return renvoTokDefault
		}
	}
	if n == 8 {
		if h == 9901561 && renvoBytesEqualText(src, start, end, "continue") {
			return renvoTokContinue
		}
	}
	return renvoTokIdent
}

func renvoTokIsKind(p *renvoProgram, i int, kind int) bool {
	renvoNonNil(p)
	base := i * renvoTokenStride
	return int(renvo_runtime_UnsafeInt32At(p.toks.data, base))&255 == kind
}

func renvoTokIdentIs(p *renvoProgram, i int, text string) bool {
	renvoNonNil(p)
	data := p.toks.data
	base := i * renvoTokenStride
	if int(renvo_runtime_UnsafeInt32At(data, base))&255 != renvoTokIdent {
		return false
	}
	packed := int(renvo_runtime_UnsafeInt32At(data, base+1))
	start := packed & 0xffffff
	size := packed>>24&255 | int(renvo_runtime_UnsafeInt32At(data, base))>>16&0xff00
	return renvoBytesEqualText(p.src, start, start+size, text)
}

func renvoTokSingleChar(p *renvoProgram, i int) byte {
	renvoNonNil(p)
	base := i * renvoTokenStride
	first := int(renvo_runtime_UnsafeInt32At(p.toks.data, base))
	if first&255 != renvoTokOp {
		return 0
	}
	return byte(first >> 24)
}

func renvoTokCharIs(p *renvoProgram, i int, c byte) bool {
	renvoNonNil(p)
	base := i * renvoTokenStride
	first := int(renvo_runtime_UnsafeInt32At(p.toks.data, base))
	return byte(first>>24) == c && first&255 == renvoTokOp
}

func renvoTok2Is(p *renvoProgram, i int, a byte, b byte) bool {
	renvoNonNil(p)
	data := p.toks.data
	base := i * renvoTokenStride
	packed := int(renvo_runtime_UnsafeInt32At(data, base+1))
	start := packed & 0xffffff
	size := packed >> 24 & 255
	if size != 2 {
		return false
	}
	if int(renvo_runtime_UnsafeInt32At(data, base))&255 != renvoTokOp {
		return false
	}
	if renvo_runtime_UnsafeByteAt(p.src, start) != a {
		return false
	}
	return renvo_runtime_UnsafeByteAt(p.src, start+1) == b
}

func renvoTokStarts2(p *renvoProgram, i int, a byte, b byte) bool {
	renvoNonNil(p)
	if i < 0 || i >= renvoTokCount(p) {
		return false
	}
	start := int(renvoTokStart(p, i))
	end := int(renvoTokEnd(p, i))
	return end-start >= 2 && renvo_runtime_UnsafeByteAt(p.src, start) == a && renvo_runtime_UnsafeByteAt(p.src, start+1) == b
}

func renvoTokStartsWith(p *renvoProgram, i int, value byte) bool {
	renvoNonNil(p)
	if i < 0 || i >= renvoTokCount(p) {
		return false
	}
	start := int(renvoTokStart(p, i))
	return int(renvoTokEnd(p, i)) > start && renvo_runtime_UnsafeByteAt(p.src, start) == value
}

func renvoBoolTokenValue(p *renvoProgram, tok int) int {
	renvoNonNil(p)
	start := renvoTokStart(p, tok)
	if renvo_runtime_UnsafeByteAt(p.src, start) == 't' {
		return 1
	}
	return 0
}

const renvoIdentNames = "\x61\x70\x70\x65\x6e\x64\x00\x00\x5b\x5d\x62\x79\x74\x65\x00\x00\x6d\x61\x6b\x65\x00\x00\x00\x00\x69\x6e\x74\x00\x00\x00\x00\x00\x75\x69\x6e\x74\x00\x00\x00\x00\x62\x79\x74\x65\x00\x00\x00\x00\x69\x6e\x74\x38\x00\x00\x00\x00\x6f\x70\x65\x6e\x00\x00\x00\x00\x72\x65\x61\x64\x00\x00\x00\x00\x63\x6f\x70\x79\x00\x00\x00\x00\x70\x61\x6e\x69\x63\x00\x00\x00\x75\x69\x6e\x74\x38\x00\x00\x00\x69\x6e\x74\x31\x36\x00\x00\x00\x69\x6e\x74\x33\x32\x00\x00\x00\x69\x6e\x74\x36\x34\x00\x00\x00\x63\x6c\x6f\x73\x65\x00\x00\x00\x77\x72\x69\x74\x65\x00\x00\x00\x63\x68\x6d\x6f\x64\x00\x00\x00\x75\x69\x6e\x74\x31\x36\x00\x00\x75\x69\x6e\x74\x33\x32\x00\x00\x75\x69\x6e\x74\x36\x34\x00\x00\x64\x65\x6c\x65\x74\x65\x00\x00\x73\x74\x72\x69\x6e\x67\x00\x00\x63\x61\x70\x00\x00\x00\x00\x00\x73\x79\x73\x63\x61\x6c\x6c\x00\x6c\x65\x6e\x00\x00\x00\x00\x00\x62\x6f\x6f\x6c\x00\x00\x00\x00\x65\x72\x72\x6f\x72\x00\x00\x00\x66\x6c\x6f\x61\x74\x36\x34\x00\x6e\x65\x77\x00\x00\x00\x00\x00\x72\x65\x63\x6f\x76\x65\x72\x00\x70\x72\x69\x6e\x74\x6c\x6e\x00\x72\x65\x61\x6c\x00\x00\x00\x00\x69\x6d\x61\x67\x00\x00\x00\x00\x63\x6f\x6d\x70\x6c\x65\x78\x00"
const renvoIdentCodes = "\x01\x02\x03\x05\x05\x07\x15\x09\x0b\x0e\x14\x07\x0f\x10\x06\x0a\x0c\x0d\x16\x17\x18\x19\x12\x13\x11\x08\x00\x00\x00\x1a\x1b\x1c\x1d\x1e\x1f"
const renvoIdentTypeCodes = "\x00\x00\x00\x01\x01\x03\x07\x00\x00\x00\x00\x03\x08\x09\x02\x00\x00\x00\x0a\x0b\x0c\x00\x05\x00\x00\x00\x04\x05\x06\x00\x00\x00\x00\x00\x00"
const renvoIdentHashTable = "\x15\x21\x1a\x1e\x00\x0c\x00\x0f\x16\x00\x13\x00\x00\x00\x22\x00\x00\x0d\x00\x00\x00\x00\x00\x1b\x1f\x09\x07\x00\x17\x00\x00\x00\x12\x08\x00\x20\x00\x01\x00\x00\x19\x00\x0b\x00\x00\x23\x03\x0a\x05\x00\x02\x00\x1d\x06\x14\x04\x00\x11\x00\x1c\x00\x0e\x18\x10"

func renvoIdentEntry(src []byte, start int, end int) int {
	n := end - start
	if start < 0 || end > len(src) || n <= 0 || n > 7 {
		return 0
	}
	hash := (n + 10*int(renvo_runtime_UnsafeByteAt(src, start)) + 5*int(renvo_runtime_UnsafeByteAt(src, end-1)) + 13*int(renvo_runtime_UnsafeByteAt(src, start+n/2))) & 63
	entry := int(renvoIdentHashTable[hash])
	if entry == 0 {
		return 0
	}
	nameStart := (entry - 1) * 8
	if renvoIdentNames[nameStart+n] != 0 {
		return 0
	}
	for i := 0; i < n; i++ {
		if renvo_runtime_UnsafeByteAt(src, start+i) != renvoIdentNames[nameStart+i] {
			return 0
		}
	}
	return entry
}

func renvoExprIdentCode(p *renvoProgram, ep *renvoExprParse, idx int) int {
	renvoNonNil(p, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return 0
	}
	entry := renvoIdentEntry(p.src, e.nameStart, e.nameEnd)
	if entry == 0 {
		return 0
	}
	return int(renvoIdentCodes[entry-1])
}

func renvoResolvedNumericCalleeCode(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	code := renvoExprIdentCode(g.prog, ep, idx)
	if code != renvoIdentReal && code != renvoIdentImag && code != renvoIdentComplex && code != renvoIdentString {
		return code
	}
	e := &ep.exprs[idx]
	if renvoFindLocalIndex(g, e.nameStart, e.nameEnd) >= 0 || renvoFindGlobalType(g, e.nameStart, e.nameEnd) != 0 || renvoFindMetaFunction(g.meta, e.nameStart, e.nameEnd) >= 0 {
		return 0
	}
	return code
}

func renvoBytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := len(text) - 1; i >= 0; i-- {
		if renvo_runtime_UnsafeByteAt(src, start+i) != text[i] {
			return false
		}
	}
	return true
}

func renvoHexDigitValue(ch byte) int {
	if ch-'0' <= 9 {
		return int(ch - '0')
	}
	lower := ch | 32
	if lower-'a' <= 5 {
		return int(lower-'a') + 10
	}
	return -1
}

func renvoDecodeStringToken(p *renvoProgram, tokIndex int) []byte {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	i := int(tok.start) + 1
	end := int(tok.end) - 1
	out := renvoFixedByteScratch(end - i)
	quote := renvo_runtime_UnsafeByteAt(src, int(tok.start))
	for i < end {
		ch := renvo_runtime_UnsafeByteAt(src, i)
		if ch == '\\' && quote == '"' && i+1 < end {
			i++
			escape := renvo_runtime_UnsafeByteAt(src, i)
			if escape == 'x' && i+2 < end {
				hi := renvoHexDigitValue(renvo_runtime_UnsafeByteAt(src, i+1))
				lo := renvoHexDigitValue(renvo_runtime_UnsafeByteAt(src, i+2))
				if hi >= 0 && lo >= 0 {
					out = append(out, byte(hi*16+lo))
					i += 3
					continue
				}
			}
			if escape == 'n' {
				out = append(out, '\n')
			} else if renvo_runtime_UnsafeByteAt(src, i) == 't' {
				out = append(out, '\t')
			} else if renvo_runtime_UnsafeByteAt(src, i) == 'r' {
				out = append(out, '\r')
			} else if renvo_runtime_UnsafeByteAt(src, i) == 'b' {
				out = append(out, '\b')
			} else if escape == 'a' {
				out = append(out, '\a')
			} else if escape == 'f' {
				out = append(out, '\f')
			} else if escape == 'v' {
				out = append(out, '\v')
			} else if renvo_runtime_UnsafeByteAt(src, i) == '"' {
				out = append(out, '"')
			} else if renvo_runtime_UnsafeByteAt(src, i) == '\\' {
				out = append(out, '\\')
			} else {
				out = append(out, renvo_runtime_UnsafeByteAt(src, i))
			}
			i++
			continue
		}
		if quote == '"' || ch != '\r' {
			out = append(out, ch)
		}
		i++
	}
	return out
}

func renvoParseIntToken(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	src := p.src
	start := int(renvoTokStart(p, tokIndex))
	end := int(renvoTokEnd(p, tokIndex))
	base := 10
	if end-start > 2 && renvo_runtime_UnsafeByteAt(src, start) == '0' {
		prefix := renvo_runtime_UnsafeByteAt(src, start+1)
		if prefix == 'x' || prefix == 'X' {
			base = 16
			start += 2
		} else if prefix == 'b' || prefix == 'B' {
			base = 2
			start += 2
		} else if prefix == 'o' || prefix == 'O' {
			base = 8
			start += 2
		}
	}
	if base == 10 && end-start > 1 && renvo_runtime_UnsafeByteAt(src, start) == '0' {
		base = 8
		start++
	}
	n := 0
	if p.compilerInt32 || p.c.renvoNativeIntSize == 4 {
		low0 := 0
		low1 := 0
		high0 := 0
		high1 := 0
		for i := start; i < end; i++ {
			c := renvo_runtime_UnsafeByteAt(src, i)
			if c == '_' {
				continue
			}
			d := 0
			if c-'0' <= 9 {
				d = int(c - '0')
			} else if c >= 'a' && c <= 'f' {
				d = int(c-'a') + 10
			} else if c >= 'A' && c <= 'F' {
				d = int(c-'A') + 10
			}
			value := low0*base + d
			low0 = value & 65535
			value = low1*base + value>>16
			low1 = value & 65535
			value = high0*base + value>>16
			high0 = value & 65535
			value = high1*base + value>>16
			high1 = value & 65535
		}
		p.parsedIntHigh = high0 | high1<<16
		n = low0 | low1<<16
	} else {
		for i := start; i < end; i++ {
			c := renvo_runtime_UnsafeByteAt(src, i)
			if c == '_' {
				continue
			}
			d := 0
			if c-'0' <= 9 {
				d = int(c - '0')
			} else if c >= 'a' && c <= 'f' {
				d = int(c-'a') + 10
			} else if c >= 'A' && c <= 'F' {
				d = int(c-'A') + 10
			}
			n = n*base + d
		}
		p.parsedIntHigh = n >> 32
	}
	return n
}

// renvoParseFloatTokenScaledCompatibility preserves the representation used by
// prepared RTG descriptors that have not declared an IEEE implementation yet.
// Ordinary built-in targets never call this path.
func renvoParseFloatTokenScaledCompatibility(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	if tok.start+2 < tok.end && renvo_runtime_UnsafeByteAt(p.src, tok.start) == '0' && (renvo_runtime_UnsafeByteAt(p.src, tok.start+1) == 'x' || renvo_runtime_UnsafeByteAt(p.src, tok.start+1) == 'X') {
		return renvoParseHexFloatTokenScaledCompatibility(p, tokIndex)
	}
	value := 0
	fractionDigits := 0
	exponent := 0
	exponentSign := 1
	afterDot := false
	i := tok.start
	for i < tok.end {
		ch := renvo_runtime_UnsafeByteAt(p.src, i)
		if ch == 'e' || ch == 'E' {
			i++
			if i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) == '-' {
				exponentSign = -1
				i++
			} else if i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) == '+' {
				i++
			}
			for i < tok.end {
				ch = renvo_runtime_UnsafeByteAt(p.src, i)
				if ch-'0' <= 9 {
					exponent = exponent*10 + int(ch-'0')
				}
				i++
			}
			break
		}
		if ch == '.' {
			afterDot = true
		} else if ch-'0' <= 9 {
			value = value*10 + int(ch-'0')
			if afterDot {
				fractionDigits++
			}
		}
		i++
	}
	value *= 4
	power := exponentSign*exponent - fractionDigits
	for power > 0 {
		value *= 10
		power--
	}
	for power < 0 {
		value /= 10
		power++
	}
	return value
}

func renvoParseHexFloatTokenScaledCompatibility(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	i := tok.start + 2
	mantissa := 0
	fractionDigits := 0
	afterDot := false
	for i < tok.end {
		ch := renvo_runtime_UnsafeByteAt(src, i)
		if ch == '_' {
			i++
			continue
		}
		if ch == '.' {
			afterDot = true
			i++
			continue
		}
		if ch == 'p' || ch == 'P' {
			break
		}
		digit := renvoHexDigitValue(ch)
		if digit >= 0 {
			mantissa = mantissa*16 + digit
			if afterDot {
				fractionDigits++
			}
		}
		i++
	}
	exponent := 0
	sign := 1
	if i < tok.end {
		i++
		if i < tok.end && renvo_runtime_UnsafeByteAt(src, i) == '-' {
			sign = -1
			i++
		} else if i < tok.end && renvo_runtime_UnsafeByteAt(src, i) == '+' {
			i++
		}
		for i < tok.end {
			ch := renvo_runtime_UnsafeByteAt(src, i)
			if ch-'0' <= 9 {
				exponent = exponent*10 + int(ch-'0')
			}
			i++
		}
	}
	power := sign*exponent + 2 - fractionDigits*4
	for power > 0 {
		mantissa *= 2
		power--
	}
	for power < 0 {
		mantissa /= 2
		power++
	}
	return mantissa
}

// renvoFloatBig is deliberately fixed-size: Go numeric tokens are bounded by
// the source buffer, while binary64 conversion only needs enough precision to
// distinguish the two adjacent representable results.  Eighty words cover
// every decimal exponent which can produce a finite binary64 value, with ample
// guard precision for long literals.  Keeping this storage on the compiler
// stack also makes literal conversion deterministic in self-hosted builds.
type renvoFloatBig struct {
	word   [80]uint32
	length int
}

// renvoFloatDecimal converts decimal tokens by shifting a decimal digit
// buffer. This avoids making the correctness of literal rounding depend on a
// self-hosted compiler's multi-limb binary division.
type renvoFloatDecimal struct {
	digit [800]byte
	nd    int
	dp    int
	trunc bool
}

func renvoFloatDecimalTrim(value *renvoFloatDecimal) {
	for value.nd > 0 && value.digit[value.nd-1] == '0' {
		value.nd--
	}
	if value.nd == 0 {
		value.dp = 0
	}
}

func renvoFloatDecimalSetToken(value *renvoFloatDecimal, p *renvoProgram, tokIndex int) {
	tok := renvoTokAt(p, tokIndex)
	afterDot := false
	i := tok.start
	for i < tok.end {
		ch := renvo_runtime_UnsafeByteAt(p.src, i)
		if ch == 'e' || ch == 'E' {
			break
		}
		if ch == '.' {
			afterDot = true
			value.dp = value.nd
		} else if ch >= '0' && ch <= '9' {
			if ch == '0' && value.nd == 0 {
				if afterDot {
					value.dp--
				}
			} else if value.nd < len(value.digit) {
				value.digit[value.nd] = ch
				value.nd++
			} else if ch != '0' {
				value.trunc = true
			}
		}
		i++
	}
	if !afterDot {
		value.dp = value.nd
	}
	if i < tok.end {
		i++
		sign := 1
		if i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) == '-' {
			sign = -1
			i++
		} else if i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) == '+' {
			i++
		}
		exponent := 0
		for i < tok.end {
			ch := renvo_runtime_UnsafeByteAt(p.src, i)
			if ch >= '0' && ch <= '9' && exponent < 10000 {
				exponent = exponent*10 + int(ch-'0')
			}
			i++
		}
		value.dp += sign * exponent
	}
	renvoFloatDecimalTrim(value)
}

func renvoFloatDecimalRightShift(value *renvoFloatDecimal, shift uint32) {
	read := 0
	write := 0
	number := uint32(0)
	for number>>shift == 0 {
		if read >= value.nd {
			if number == 0 {
				value.nd = 0
				return
			}
			for number>>shift == 0 {
				number *= 10
				read++
			}
			break
		}
		number = number*10 + uint32(value.digit[read]-'0')
		read++
	}
	value.dp -= read - 1
	mask := uint32(1)<<shift - 1
	for read < value.nd {
		digit := number >> shift
		number &= mask
		value.digit[write] = byte(digit + '0')
		write++
		number = number*10 + uint32(value.digit[read]-'0')
		read++
	}
	for number > 0 {
		digit := number >> shift
		number &= mask
		if write < len(value.digit) {
			value.digit[write] = byte(digit + '0')
			write++
		} else if digit != 0 {
			value.trunc = true
		}
		number *= 10
	}
	value.nd = write
	renvoFloatDecimalTrim(value)
}

type renvoFloatDecimalLeftShift struct {
	delta  int
	cutoff string
}

var renvoFloatDecimalLeftShifts = [29]renvoFloatDecimalLeftShift{
	{0, ""}, {1, "5"}, {1, "25"}, {1, "125"}, {2, "625"},
	{2, "3125"}, {2, "15625"}, {3, "78125"}, {3, "390625"},
	{3, "1953125"}, {4, "9765625"}, {4, "48828125"},
	{4, "244140625"}, {4, "1220703125"}, {5, "6103515625"},
	{5, "30517578125"}, {5, "152587890625"}, {6, "762939453125"},
	{6, "3814697265625"}, {6, "19073486328125"}, {7, "95367431640625"},
	{7, "476837158203125"}, {7, "2384185791015625"},
	{7, "11920928955078125"}, {8, "59604644775390625"},
	{8, "298023223876953125"}, {8, "1490116119384765625"},
	{9, "7450580596923828125"}, {9, "37252902984619140625"},
}

func renvoFloatDecimalPrefixLess(digits []byte, cutoff string) bool {
	for i := 0; i < len(cutoff); i++ {
		if i >= len(digits) {
			return true
		}
		if digits[i] != cutoff[i] {
			return digits[i] < cutoff[i]
		}
	}
	return false
}

func renvoFloatDecimalLeftShiftBits(value *renvoFloatDecimal, shift uint32) {
	delta := renvoFloatDecimalLeftShifts[shift].delta
	if renvoFloatDecimalPrefixLess(value.digit[:value.nd], renvoFloatDecimalLeftShifts[shift].cutoff) {
		delta--
	}
	read := value.nd
	write := value.nd + delta
	number := uint32(0)
	for read > 0 {
		read--
		number += uint32(value.digit[read]-'0') << shift
		quotient := number / 10
		remainder := number - 10*quotient
		write--
		if write < len(value.digit) {
			value.digit[write] = byte(remainder + '0')
		} else if remainder != 0 {
			value.trunc = true
		}
		number = quotient
	}
	for number > 0 {
		quotient := number / 10
		remainder := number - 10*quotient
		write--
		if write < len(value.digit) {
			value.digit[write] = byte(remainder + '0')
		} else if remainder != 0 {
			value.trunc = true
		}
		number = quotient
	}
	value.nd += delta
	if value.nd > len(value.digit) {
		value.nd = len(value.digit)
	}
	value.dp += delta
	renvoFloatDecimalTrim(value)
}

func renvoFloatDecimalShift(value *renvoFloatDecimal, shift int) {
	if value.nd == 0 {
		return
	}
	for shift > 28 {
		renvoFloatDecimalLeftShiftBits(value, 28)
		shift -= 28
	}
	if shift > 0 {
		renvoFloatDecimalLeftShiftBits(value, uint32(shift))
		return
	}
	for shift < -28 {
		renvoFloatDecimalRightShift(value, 28)
		shift += 28
	}
	if shift < 0 {
		renvoFloatDecimalRightShift(value, uint32(-shift))
	}
}

func renvoFloatDecimalShouldRoundUp(value *renvoFloatDecimal, digits int) bool {
	if digits < 0 || digits >= value.nd {
		return false
	}
	if value.digit[digits] == '5' && digits+1 == value.nd {
		if value.trunc {
			return true
		}
		return digits > 0 && (value.digit[digits-1]-'0')&1 != 0
	}
	return value.digit[digits] >= '5'
}

func renvoFloatDecimalRoundedInteger(value *renvoFloatDecimal) uint64 {
	if value.dp > 20 {
		return ^uint64(0)
	}
	index := 0
	number := uint64(0)
	for index < value.dp && index < value.nd {
		number = number*10 + uint64(value.digit[index]-'0')
		index++
	}
	for index < value.dp {
		number *= 10
		index++
	}
	if renvoFloatDecimalShouldRoundUp(value, value.dp) {
		number++
	}
	return number
}

func renvoParseDecimalFloatTokenBits(p *renvoProgram, tokIndex int, fractionBits int, exponentBits int, bias int) uint64 {
	var value renvoFloatDecimal
	renvoFloatDecimalSetToken(&value, p, tokIndex)
	if value.nd == 0 {
		return 0
	}
	if value.dp > 310 {
		return uint64((1<<exponentBits)-1) << fractionBits
	}
	if value.dp < -330 {
		return 0
	}
	powers := [9]int{1, 3, 6, 9, 13, 16, 19, 23, 26}
	exponent := 0
	for value.dp > 0 {
		shift := 27
		if value.dp < len(powers) {
			shift = powers[value.dp]
		}
		renvoFloatDecimalShift(&value, -shift)
		exponent += shift
	}
	for value.dp < 0 || value.dp == 0 && value.digit[0] < '5' {
		shift := 27
		if -value.dp < len(powers) {
			shift = powers[-value.dp]
		}
		renvoFloatDecimalShift(&value, shift)
		exponent -= shift
	}
	exponent--
	negativeBias := -bias
	if exponent < negativeBias+1 {
		shift := negativeBias + 1 - exponent
		renvoFloatDecimalShift(&value, -shift)
		exponent += shift
	}
	if exponent-negativeBias >= 1<<exponentBits-1 {
		return uint64((1<<exponentBits)-1) << fractionBits
	}
	renvoFloatDecimalShift(&value, 1+fractionBits)
	mantissa := renvoFloatDecimalRoundedInteger(&value)
	if mantissa == uint64(2)<<fractionBits {
		mantissa >>= 1
		exponent++
		if exponent-negativeBias >= 1<<exponentBits-1 {
			return uint64((1<<exponentBits)-1) << fractionBits
		}
	}
	if mantissa&(uint64(1)<<fractionBits) == 0 {
		exponent = negativeBias
	}
	bits := mantissa & (uint64(1)<<fractionBits - 1)
	bits |= uint64((exponent-negativeBias)&(1<<exponentBits-1)) << fractionBits
	return bits
}

func renvoFloatBigSet(value *renvoFloatBig, number uint32) {
	value.length = 0
	if number != 0 {
		value.word[0] = number
		value.length = 1
	}
}

func renvoFloatBigCopy(dst *renvoFloatBig, src *renvoFloatBig) {
	dst.length = src.length
	for i := 0; i < src.length; i++ {
		dst.word[i] = src.word[i]
	}
}

func renvoFloatBigMulSmall(value *renvoFloatBig, multiplier uint32) bool {
	carry := uint32(0)
	for i := 0; i < value.length; i++ {
		word := value.word[i]
		low := (word&0xffff)*multiplier + carry
		high := (word>>16)*multiplier + (low >> 16)
		value.word[i] = low&0xffff | high<<16
		carry = high >> 16
	}
	if carry != 0 {
		if value.length >= len(value.word) {
			return false
		}
		value.word[value.length] = carry
		value.length++
	}
	return true
}

func renvoFloatBigAddSmall(value *renvoFloatBig, addend uint32) bool {
	if value.length == 0 {
		renvoFloatBigSet(value, addend)
		return true
	}
	carry := addend
	for i := 0; carry != 0 && i < value.length; i++ {
		word := value.word[i]
		value.word[i] = word + carry
		carry = 0
		if renvoFloatUint32Less(value.word[i], word) {
			carry = 1
		}
	}
	if carry != 0 {
		if value.length >= len(value.word) {
			return false
		}
		value.word[value.length] = carry
		value.length++
	}
	return true
}

func renvoFloatBigBitLength(value *renvoFloatBig) int {
	if value.length == 0 {
		return 0
	}
	word := value.word[value.length-1]
	bits := 0
	for word != 0 {
		word >>= 1
		bits++
	}
	return (value.length-1)*32 + bits
}

func renvoFloatBigShiftedWord(value *renvoFloatBig, index int, shift int) uint32 {
	wordShift := shift / 32
	bitShift := shift & 31
	source := index - wordShift
	result := uint32(0)
	if source >= 0 && source < value.length {
		result = value.word[source] << bitShift
	}
	if bitShift != 0 && source-1 >= 0 && source-1 < value.length {
		result |= value.word[source-1] >> (32 - bitShift)
	}
	return result
}

func renvoFloatBigShiftedLength(value *renvoFloatBig, shift int) int {
	if value.length == 0 {
		return 0
	}
	length := value.length + shift/32
	if shift&31 != 0 && value.word[value.length-1]>>(32-(shift&31)) != 0 {
		length++
	}
	return length
}

func renvoFloatUint32Less(left uint32, right uint32) bool {
	// Fixed 32-bit compiler targets intentionally keep the generic unsigned
	// branch fast path disabled. Biasing the sign bit makes this comparison use
	// signed int32 ordering while preserving uint32 ordering on every target.
	return int32(left^uint32(0x80000000)) < int32(right^uint32(0x80000000))
}

func renvoFloatBigCompareShifted(left *renvoFloatBig, right *renvoFloatBig, rightShift int) int {
	leftLength := left.length
	rightLength := renvoFloatBigShiftedLength(right, rightShift)
	if leftLength < rightLength {
		return -1
	}
	if leftLength > rightLength {
		return 1
	}
	for i := leftLength - 1; i >= 0; i-- {
		rightWord := renvoFloatBigShiftedWord(right, i, rightShift)
		if renvoFloatUint32Less(left.word[i], rightWord) {
			return -1
		}
		if renvoFloatUint32Less(rightWord, left.word[i]) {
			return 1
		}
	}
	return 0
}

func renvoFloatBigSubShifted(left *renvoFloatBig, right *renvoFloatBig, rightShift int) {
	borrow := uint32(0)
	for i := 0; i < left.length; i++ {
		word := left.word[i]
		subtrahend := renvoFloatBigShiftedWord(right, i, rightShift)
		next := word - subtrahend
		borrowFromWord := renvoFloatUint32Less(word, subtrahend)
		withBorrow := next - borrow
		borrowFromCarry := renvoFloatUint32Less(next, borrow)
		left.word[i] = withBorrow
		borrow = 0
		if borrowFromWord || borrowFromCarry {
			borrow = 1
		}
	}
	for left.length > 0 && left.word[left.length-1] == 0 {
		left.length--
	}
}

func renvoFloatBigShiftLeft(value *renvoFloatBig, shift int) bool {
	if value.length == 0 || shift == 0 {
		return true
	}
	wordShift := shift / 32
	newLength := renvoFloatBigShiftedLength(value, shift)
	if newLength > len(value.word) {
		return false
	}
	for i := newLength - 1; i >= 0; i-- {
		value.word[i] = renvoFloatBigShiftedWord(value, i, shift)
	}
	for i := 0; i < wordShift; i++ {
		value.word[i] = 0
	}
	value.length = newLength
	return true
}

func renvoFloatBigQuotientRounded(numerator *renvoFloatBig, denominator *renvoFloatBig, sticky bool, quotientLow *uint32, quotientHigh *uint32) {
	var remainder renvoFloatBig
	renvoFloatBigCopy(&remainder, numerator)
	difference := renvoFloatBigBitLength(&remainder) - renvoFloatBigBitLength(denominator)
	*quotientLow = 0
	*quotientHigh = 0
	for bit := difference; bit >= 0; bit-- {
		if bit < 64 && renvoFloatBigCompareShifted(&remainder, denominator, bit) >= 0 {
			renvoFloatBigSubShifted(&remainder, denominator, bit)
			if bit < 32 {
				*quotientLow = *quotientLow | uint32(1)<<bit
			} else {
				*quotientHigh = *quotientHigh | uint32(1)<<(bit-32)
			}
		}
	}
	comparison := renvoFloatBigCompareShifted(denominator, &remainder, 1)
	if comparison < 0 || comparison == 0 && (sticky || *quotientLow&1 != 0) {
		*quotientLow++
		if *quotientLow == 0 {
			*quotientHigh++
		}
	}
}

func renvoFloatBigRatioExponent(numerator *renvoFloatBig, denominator *renvoFloatBig) int {
	difference := renvoFloatBigBitLength(numerator) - renvoFloatBigBitLength(denominator)
	if difference >= 0 {
		if renvoFloatBigCompareShifted(numerator, denominator, difference) < 0 {
			difference--
		}
		return difference
	}
	if renvoFloatBigCompareShifted(denominator, numerator, -difference) > 0 {
		difference--
	}
	return difference
}

func renvoFloatBigDivideRounded(numerator *renvoFloatBig, denominator *renvoFloatBig, sticky bool, quotientLow *uint32, quotientHigh *uint32) {
	// Keep conversion entirely in 32-bit limbs. Besides avoiding a target-sized
	// fast path, this is important when a 386/ARM Renvo compiler is compiling
	// the next stage: its uint64 division helper must not become part of the
	// correctness boundary for decimal literal rounding.
	renvoFloatBigQuotientRounded(numerator, denominator, sticky, quotientLow, quotientHigh)
}

func renvoFloatBitsFromWords(high uint32, low uint32) uint64 {
	return uint64(high)<<32 | uint64(low)
}

func renvoFloatInfinityBits(fractionBits int, exponentBits int) uint64 {
	if fractionBits == 23 {
		return uint64(uint32((1<<exponentBits)-1) << 23)
	}
	return renvoFloatBitsFromWords(uint32((1<<exponentBits)-1)<<20, 0)
}

func renvoFloatBigEncode(numerator *renvoFloatBig, denominator *renvoFloatBig, binaryShift int, fractionBits int, exponentBits int, bias int, sticky bool) uint64 {
	if numerator.length == 0 {
		return 0
	}
	maxExponent := (1 << exponentBits) - 2 - bias
	minExponent := 1 - bias
	exponent := renvoFloatBigRatioExponent(numerator, denominator) + binaryShift
	if exponent > maxExponent {
		return renvoFloatInfinityBits(fractionBits, exponentBits)
	}
	var scaledNumerator renvoFloatBig
	var scaledDenominator renvoFloatBig
	renvoFloatBigCopy(&scaledNumerator, numerator)
	renvoFloatBigCopy(&scaledDenominator, denominator)
	targetExponent := exponent - fractionBits
	if exponent < minExponent {
		targetExponent = minExponent - fractionBits
	}
	shift := binaryShift - targetExponent
	if shift >= 0 {
		if !renvoFloatBigShiftLeft(&scaledNumerator, shift) {
			return renvoFloatInfinityBits(fractionBits, exponentBits)
		}
	} else if !renvoFloatBigShiftLeft(&scaledDenominator, -shift) {
		return 0
	}
	significandLow := uint32(0)
	significandHigh := uint32(0)
	renvoFloatBigDivideRounded(&scaledNumerator, &scaledDenominator, sticky, &significandLow, &significandHigh)
	if exponent < minExponent {
		if fractionBits == 23 {
			if significandHigh != 0 || !renvoFloatUint32Less(significandLow, uint32(1)<<23) {
				return uint64(uint32(1) << 23)
			}
			return uint64(significandLow)
		}
		if significandHigh >= uint32(1)<<20 {
			return renvoFloatBitsFromWords(uint32(1)<<20, 0)
		}
		return renvoFloatBitsFromWords(significandHigh, significandLow)
	}
	overflow := significandHigh != 0
	if fractionBits == 52 {
		overflow = significandHigh >= uint32(1)<<21
	} else if fractionBits == 23 {
		overflow = significandHigh != 0 || !renvoFloatUint32Less(significandLow, uint32(1)<<24)
	}
	if overflow {
		significandLow = significandLow>>1 | significandHigh<<31
		significandHigh >>= 1
		exponent++
		if exponent > maxExponent {
			return renvoFloatInfinityBits(fractionBits, exponentBits)
		}
	}
	if fractionBits == 23 {
		return uint64(uint32(exponent+bias)<<23 | significandLow&(uint32(1)<<23-1))
	}
	high := uint32(exponent+bias)<<20 | significandHigh&(uint32(1)<<20-1)
	return renvoFloatBitsFromWords(high, significandLow)
}

func renvoParseFloatTokenBits(p *renvoProgram, tokIndex int, fractionBits int, exponentBits int, bias int) uint64 {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	if tok.start+2 < tok.end && renvo_runtime_UnsafeByteAt(src, int(tok.start)) == '0' &&
		(renvo_runtime_UnsafeByteAt(src, int(tok.start)+1) == 'x' || renvo_runtime_UnsafeByteAt(src, int(tok.start)+1) == 'X') {
		return renvoParseHexFloatTokenBits(p, tokIndex, fractionBits, exponentBits, bias)
	}
	return renvoParseDecimalFloatTokenBits(p, tokIndex, fractionBits, exponentBits, bias)
}

func renvoParseHexFloatTokenBits(p *renvoProgram, tokIndex int, fractionBits int, exponentBits int, bias int) uint64 {
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	var numerator renvoFloatBig
	var denominator renvoFloatBig
	renvoFloatBigSet(&numerator, 0)
	renvoFloatBigSet(&denominator, 1)
	fractionDigits := 0
	afterDot := false
	i := tok.start + 2
	for i < tok.end {
		ch := renvo_runtime_UnsafeByteAt(src, int(i))
		if ch == 'p' || ch == 'P' {
			break
		}
		if ch == '.' {
			afterDot = true
		} else if ch != '_' {
			digit := renvoHexDigitValue(ch)
			if digit >= 0 {
				renvoFloatBigMulSmall(&numerator, 16)
				renvoFloatBigAddSmall(&numerator, uint32(digit))
				if afterDot {
					fractionDigits++
				}
			}
		}
		i++
	}
	exponent2 := -fractionDigits * 4
	if i < tok.end {
		i++
		sign := 1
		if i < tok.end && renvo_runtime_UnsafeByteAt(src, int(i)) == '-' {
			sign = -1
			i++
		} else if i < tok.end && renvo_runtime_UnsafeByteAt(src, int(i)) == '+' {
			i++
		}
		exponent := 0
		for i < tok.end {
			ch := renvo_runtime_UnsafeByteAt(src, int(i))
			if ch >= '0' && ch <= '9' && exponent < 10000 {
				exponent = exponent*10 + int(ch-'0')
			}
			i++
		}
		exponent2 += sign * exponent
	}
	return renvoFloatBigEncode(&numerator, &denominator, exponent2, fractionBits, exponentBits, bias, false)
}

func renvoParseCharToken(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	i := tok.start + 1
	if i >= tok.end-1 {
		return 0
	}
	if renvo_runtime_UnsafeByteAt(src, i) != '\\' {
		c0 := int(renvo_runtime_UnsafeByteAt(src, i))
		if c0 < 128 {
			return c0
		}
		width := 2
		bias := 192
		if c0 >= 240 {
			width = 4
			bias = 240
		} else if c0 >= 224 {
			width = 3
			bias = 224
		}
		if i+width > tok.end-1 {
			return 0
		}
		value := c0 - bias
		for j := 1; j < width; j++ {
			value = value*64 + int(renvo_runtime_UnsafeByteAt(src, i+j)-128)
		}
		return value
	}
	i++
	if i >= tok.end-1 {
		return 0
	}
	escape := renvo_runtime_UnsafeByteAt(src, i)
	if escape == 'a' {
		return '\a'
	} else if escape == 'b' {
		return '\b'
	} else if escape == 'f' {
		return '\f'
	} else if escape == 'n' {
		return '\n'
	} else if escape == 'r' {
		return '\r'
	} else if escape == 't' {
		return '\t'
	} else if escape == 'v' {
		return '\v'
	} else if escape == '\\' || escape == '\'' || escape == '"' {
		return int(escape)
	}
	digits := 0
	if escape == 'x' {
		digits = 2
	} else if escape == 'u' {
		digits = 4
	} else if escape == 'U' {
		digits = 8
	}
	if digits > 0 {
		return renvoParseEscapeDigits(src, i+1, digits, 16)
	}
	if escape >= '0' && escape <= '7' {
		return renvoParseEscapeDigits(src, i, 3, 8)
	}
	return int(escape)
}

func renvoParseEscapeDigits(src []byte, start int, count int, base int) int {
	value := 0
	for i := 0; i < count; i++ {
		digit := renvoHexDigitValue(renvo_runtime_UnsafeByteAt(src, start+i))
		if digit < 0 || digit >= base {
			return 0
		}
		value = value*base + digit
	}
	return value
}

func renvoEvalConstByName(g *renvoLinearGen, nameStart int, nameEnd int) renvoConstResult {
	renvoNonNil(g)
	var result renvoConstResult
	renvoEvalConstByNameInto(g, nameStart, nameEnd, &result)
	return result
}

func renvoEvalConstByNameInto(g *renvoLinearGen, nameStart int, nameEnd int, out *renvoConstResult) {
	renvoNonNil(g, out)
	builtin := renvoEvalBuiltinConst(g, nameStart, nameEnd)
	if builtin.ok {
		*out = builtin
		return
	}
	symIndex := renvoFindMetaGlobalIndex(g.meta, nameStart, nameEnd, renvoTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.constValueOK != 0 {
			if renvoTypeKindIsFloat(renvoResolveType(g.meta, s.typ).kind) {
				renvoSetConstResult(out, 0, false)
				return
			}
			value := s.constValue
			renvoSetConstResult(out, value, true)
			return
		}
		ep := renvoNewExprParse()
		if !renvoParseExpressionOK(ep, g.prog, s.initStart, s.initEnd) {
			renvoSetConstResult(out, 0, false)
			return
		}
		oldIota := g.constEvalIota
		oldIotaValid := g.constEvalIotaValid
		g.constEvalIota = s.iotaValue
		g.constEvalIotaValid = 1
		result := renvoEvalConstExpr(g, ep, len(ep.exprs)-1)
		g.constEvalIota = oldIota
		g.constEvalIotaValid = oldIotaValid
		renvoSetConstResult(out, result.value, result.ok)
		return
	}
	renvoSetConstResult(out, 0, false)
}

// renvoEmitFloatConstByName emits a typed floating-point constant from its
// initializer instead of passing it through the integer constant evaluator.
// It returns zero when the name is not a floating-point constant, one after a
// successful emission, and -1 when the initializer could not be emitted.
func renvoEmitFloatConstByName(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	symIndex := renvoFindMetaGlobalIndex(g.meta, nameStart, nameEnd, renvoTokConst)
	if symIndex < 0 {
		return 0
	}
	s := &g.meta.globals[symIndex]
	t := renvoResolveType(g.meta, s.typ)
	if !renvoTypeKindIsFloat(t.kind) {
		return 0
	}
	ep := renvoNewExprParse()
	root := renvoParseExpressionRoot(ep, g.prog, s.initStart, s.initEnd)
	if root < 0 || !renvoEmitScalarExprForKind(g, ep, root, t.kind) {
		return -1
	}
	return 1
}

func renvoConstResultOk(value int) renvoConstResult {
	return renvoConstResult{value: value, ok: true}
}

func renvoConvertConstInt(renvoNativeIntSize int, value int, kind int) int {
	if kind == renvoTypeByte {
		return value & 0xff
	}
	if kind == renvoTypeInt8 {
		value = value & 0xff
		if value >= 0x80 {
			value -= 0x100
		}
		return value
	}
	if kind == renvoTypeInt && renvoNativeIntSize == 4 {
		kind = renvoTypeInt32
	}
	if kind == renvoTypeInt16 {
		value = value & 0xffff
		if value >= 0x8000 {
			value -= 0x10000
		}
		return value
	}
	if kind == renvoTypeUint16 {
		return value & 0xffff
	}
	if kind == renvoTypeInt32 {
		limit := 2147483647
		if value > limit {
			value -= limit
			value -= limit
			value -= 2
		}
		return value
	}
	if kind == renvoTypeUint32 && renvoNativeIntSize > 4 {
		return value & 0xffffffff
	}
	return value
}

// Literal emission needs target-sized words, but constant evaluation can keep
// both words when the compiler host has a 64-bit int.
func renvoParseConstIntToken(p *renvoProgram, tok int) int {
	value := renvoParseIntToken(p, tok)
	if !p.compilerInt32 && p.c.renvoNativeIntSize == 4 {
		value = int(uint64(uint32(value)) | uint64(uint32(p.parsedIntHigh))<<32)
	}
	return value
}

func renvoSetConstResult(result *renvoConstResult, value int, ok bool) {
	result.value = value
	result.ok = ok
}

func renvoEvalConstExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) renvoConstResult {
	renvoNonNil(g, ep)
	var result renvoConstResult
	renvoEvalConstExprInto(g, ep, idx, &result)
	return result
}

func renvoEvalBooleanConst(g *renvoLinearGen, tok int) renvoConstResult {
	p := g.prog
	local := renvoFindLocalIndex(g, renvoTokStart(p, tok), renvoTokEnd(p, tok))
	if local >= 0 {
		if g.locals[local].constValid != 0 && renvoTypeKindIsScalarInt(renvoResolveType(g.meta, g.locals[local].typ).kind) {
			return renvoConstResultOk(g.locals[local].constValue)
		}
		return renvoConstResult{}
	}
	return renvoConstResultOk(renvoBoolTokenValue(p, tok))
}

func renvoEvalConstExprInto(g *renvoLinearGen, ep *renvoExprParse, idx int, out *renvoConstResult) {
	renvoNonNil(g, ep, out)
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		value := renvoParseConstIntToken(p, e.tok)
		if p.compilerInt32 && p.parsedIntHigh != value>>31 {
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprFloat {
		// Floating constants carry more than one compiler word on 32-bit hosts.
		// Their exact representation is materialized from the source token by the
		// typed emitter instead of passing through the integer constant evaluator.
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprChar {
		value := renvoParseCharToken(p, e.tok)
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprBool {
		*out = renvoEvalBooleanConst(g, e.tok)
		return
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if g.locals[localIndex].constValid != 0 && renvoTypeKindIsScalarInt(renvoResolveType(g.meta, g.locals[localIndex].typ).kind) {
				renvoSetConstResult(out, g.locals[localIndex].constValue, true)
				return
			}
			if renvoFixedTarget == 0 {
				if g.constEvalFlow && g.locals[localIndex].flowConstValid > 0 {
					renvoSetConstResult(out, g.locals[localIndex].flowConstValue, true)
					return
				}
			}
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoEvalConstByNameInto(g, e.nameStart, e.nameEnd, out)
		return
	}
	if e.kind == renvoExprCall {
		if renvoFixedTarget == 0 {
			if renvoIsSysVObject(g.c) && e.argCount == 0 && g.constCallDepth < 8 {
				fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
				if fnIndex >= 0 {
					fn := &g.meta.funcs[fnIndex]
					// A linkstatic declaration's Go body is only an ABI-shaped stub;
					// its return expression says nothing about the external function.
					// Treating `return 0` as the foreign result can fold away every
					// statement after a runtime query whose real value is nonzero.
					start, end, found := renvoFunctionLeadingReturnExpr(g, fn)
					if fn.linkStatic == 0 && found {
						body := renvoNewExprParse()
						renvoNonNil(body)
						bodyIndex := renvoParseExpressionRoot(body, p, start, end)
						if bodyIndex >= 0 {
							g.constCallDepth++
							value := renvoEvalConstExpr(g, body, bodyIndex)
							g.constCallDepth--
							if value.ok {
								*out = value
								return
							}
						}
					}
				}
			}
			if renvoExprIsIdentText(p, ep, e.left, "__c_bool_int") && e.argCount == 1 {
				arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
				value := renvoEvalConstExpr(g, ep, arg)
				if value.ok {
					if value.value != 0 {
						value.value = 1
					}
					*out = value
					return
				}
			}
		}
		if e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				resolved := renvoResolveType(g.meta, conversionType)
				renvoNonNil(resolved)
				if renvoTypeKindIsScalarValue(resolved.kind) {
					argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
					result := renvoEvalConstExpr(g, ep, argIndex)
					if result.ok {
						source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
						renvoNonNil(source)
						// Floating-point constants cannot be represented in the one-word
						// integer constant result. Let the typed IEEE emitter evaluate the
						// conversion instead of reviving the former scaled-integer model.
						if renvoTypeKindIsFloat(resolved.kind) || renvoTypeKindIsFloat(source.kind) {
							renvoSetConstResult(out, 0, false)
							return
						}
						result.value = renvoConvertConstInt(g.c.renvoNativeIntSize, result.value, resolved.kind)
					}
					*out = result
					return
				}
			}
		}
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprUnary {
		inner := renvoEvalConstExpr(g, ep, e.left)
		if !inner.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		value := inner.value
		if renvoTokCharIs(p, e.tok, '-') {
			value = -value
		} else if renvoTokCharIs(p, e.tok, '+') {
		} else if renvoTokCharIs(p, e.tok, '^') {
			value = ^value
			typ := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
			value = renvoConvertConstInt(g.c.renvoNativeIntSize, value, typ.kind)
		} else if renvoTokCharIs(p, e.tok, '!') {
			value = 0
			if inner.value == 0 {
				value = 1
			}
		} else {
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprBinary {
		opTok := e.tok
		rightIndex := e.right
		rightExpr := &ep.exprs[rightIndex]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		left := renvoEvalConstExpr(g, ep, e.left)
		shift := renvoTok2Is(p, e.tok, '<', '<') || renvoTok2Is(p, e.tok, '>', '>')
		if shift && !left.ok {
			left = renvoEvalIntegralShiftLiteral(g, ep, e.left)
		}
		if !left.ok {
			if renvoFixedTarget == 0 {
				right := renvoEvalConstExpr(g, ep, rightIndex)
				if right.ok && renvoConstExprSideEffectFree(g, ep, e.left) {
					if right.value == 0 && (renvoTokCharIs(p, e.tok, '&') || renvoTokCharIs(p, e.tok, '*') || renvoTok2Is(p, e.tok, '&', '&')) {
						renvoSetConstResult(out, 0, true)
						return
					}
					if right.value != 0 && renvoTok2Is(p, e.tok, '|', '|') {
						renvoSetConstResult(out, 1, true)
						return
					}
				}
			}
			renvoSetConstResult(out, 0, false)
			return
		}
		if renvoTok2Is(p, e.tok, '&', '&') {
			if left.value == 0 {
				renvoSetConstResult(out, 0, true)
				return
			}
			right := renvoEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				renvoSetConstResult(out, 0, false)
				return
			}
			value := 0
			if right.value != 0 {
				value = 1
			}
			renvoSetConstResult(out, value, true)
			return
		}
		if renvoTok2Is(p, e.tok, '|', '|') {
			if left.value != 0 {
				renvoSetConstResult(out, 1, true)
				return
			}
			right := renvoEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				renvoSetConstResult(out, 0, false)
				return
			}
			value := 0
			if right.value != 0 {
				value = 1
			}
			renvoSetConstResult(out, value, true)
			return
		}
		var right renvoConstResult
		if rightKind == renvoExprInt {
			value := renvoParseConstIntToken(p, rightTok)
			if p.compilerInt32 && p.parsedIntHigh != value>>31 {
				renvoSetConstResult(out, 0, false)
				return
			}
			right = renvoConstResultOk(value)
		} else if rightKind == renvoExprChar {
			value := renvoParseCharToken(p, rightTok)
			right = renvoConstResultOk(value)
		} else if rightKind == renvoExprBool {
			right = renvoEvalBooleanConst(g, rightTok)
		} else {
			right = renvoEvalConstExpr(g, ep, rightIndex)
		}
		if !right.ok {
			if shift {
				right = renvoEvalIntegralShiftLiteral(g, ep, rightIndex)
			}
		}
		if !right.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		usesFloat := renvoBinaryUsesFloat(g, ep, e)
		if shift {
			usesFloat = false
		}
		if usesFloat {
			// Runtime and typed constant lowering preserve the full IEEE value;
			// the integer-only evaluator must not approximate it.
			renvoSetConstResult(out, 0, false)
			return
		}
		unsignedKind := 0
		if !usesFloat && renvoExprHasUnsignedIntType(g, ep, e.left) {
			unsignedKind = renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left)).kind
		}
		if !usesFloat && !renvoTok2Is(p, opTok, '<', '<') && !renvoTok2Is(p, opTok, '>', '>') && renvoExprHasUnsignedIntType(g, ep, e.right) {
			rightUnsignedKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right)).kind
			if unsignedKind == 0 || renvoUnsignedKindSize(rightUnsignedKind) > renvoUnsignedKindSize(unsignedKind) {
				unsignedKind = rightUnsignedKind
			}
		}
		renvoEvalConstBinaryInto(g, opTok, left.value, right.value, unsignedKind, out)
		return
	}
	renvoSetConstResult(out, 0, false)
}

// A shift permits an untyped floating literal only when its exact value is an
// integer. Never round through IEEE arithmetic or truncate fractional digits.
func renvoEvalIntegralShiftLiteral(g *renvoLinearGen, ep *renvoExprParse, idx int) renvoConstResult {
	e := &ep.exprs[idx]
	p := g.prog
	if e.kind == renvoExprUnary && (renvoTokCharIs(p, e.tok, '+') || renvoTokCharIs(p, e.tok, '-')) {
		value := renvoEvalIntegralShiftLiteral(g, ep, e.left)
		if value.ok && renvoTokCharIs(p, e.tok, '-') {
			value.value = -value.value
		}
		return value
	}
	if e.kind != renvoExprFloat {
		return renvoConstResult{}
	}
	tok := renvoTokAt(p, e.tok)
	if tok.end-tok.start >= 800 || renvoExprTokenIsImaginary(p, e.tok) {
		return renvoConstResult{}
	}
	for at := tok.start; at < tok.end; at++ {
		ch := renvo_runtime_UnsafeByteAt(p.src, at)
		if ch == 'x' || ch == 'X' || ch == 'p' || ch == 'P' {
			return renvoConstResult{}
		}
	}
	var decimal renvoFloatDecimal
	renvoFloatDecimalSetToken(&decimal, p, e.tok)
	if decimal.trunc || decimal.dp < decimal.nd || decimal.dp > 19 {
		return renvoConstResult{}
	}
	value := 0
	maximum := int(^uint(0) >> 1)
	for at := 0; at < decimal.dp; at++ {
		digit := 0
		if at < decimal.nd {
			digit = int(decimal.digit[at] - '0')
		}
		if value > (maximum-digit)/10 {
			return renvoConstResult{}
		}
		value = value*10 + digit
	}
	return renvoConstResultOk(value)
}

// renvoFunctionLeadingReturnExpr finds an unconditional return at the start of
// a function. C lowering sometimes preserves a constant-folded statement as a
// standalone block ("{ return false }") before the remainder of the function.
// That block has exactly the same control-flow semantics as a direct return.
func renvoFunctionLeadingReturnExpr(g *renvoLinearGen, fn *renvoFuncInfo) (int, int, bool) {
	renvoNonNil(g, fn)
	start, end := fn.bodyStart, fn.bodyEnd
	for depth := 0; depth < 8 && start < end; depth++ {
		var bp renvoBodyParse
		bp.prog = g.prog
		bp.ok = true
		next := renvoParseOneStatement(&bp, start, end)
		if !bp.ok || next <= start || bp.stmtCount != 1 {
			return 0, 0, false
		}
		stmt := renvoBodyStmtAt(&bp, 0)
		if !bp.ok {
			return 0, 0, false
		}
		if stmt.kind == renvoStmtReturn && stmt.exprStart < stmt.exprEnd {
			return stmt.exprStart, stmt.exprEnd, true
		}
		if stmt.kind != renvoStmtBlock {
			return 0, 0, false
		}
		start, end = stmt.bodyStart, stmt.bodyEnd
	}
	return 0, 0, false
}

func renvoConstExprSideEffectFree(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt || e.kind == renvoExprFloat || e.kind == renvoExprChar ||
		e.kind == renvoExprBool || e.kind == renvoExprIdent {
		return true
	}
	if e.kind == renvoExprUnary {
		return !renvoTokCharIs(g.prog, e.tok, '*') && !renvoTokCharIs(g.prog, e.tok, '&') &&
			renvoConstExprSideEffectFree(g, ep, e.left)
	}
	if e.kind == renvoExprBinary {
		return renvoConstExprSideEffectFree(g, ep, e.left) && renvoConstExprSideEffectFree(g, ep, e.right)
	}
	if e.kind == renvoExprCall && e.argCount == 1 &&
		(renvoConversionTypeFromExpr(g, ep, e.left) != 0 || renvoExprIsIdentText(g.prog, ep, e.left, "__c_bool_int")) {
		return renvoConstExprSideEffectFree(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	}
	return false
}

func renvoCFieldAccessorOffset(src []byte, start int, end int) (int, bool) {
	prefix := "__c_ptr_"
	at := start + len(prefix)
	if !renvoBytesPrefixText(src, start, end, prefix) {
		prefix = "__c_nested_ptr_"
		if !renvoBytesPrefixText(src, start, end, prefix) {
			return 0, false
		}
		at = start + len(prefix)
		typeDigits := 0
		for at < end && src[at] >= '0' && src[at] <= '9' {
			typeDigits++
			at++
		}
		if typeDigits == 0 || at >= end || src[at] != '_' {
			return 0, false
		}
		at++
	}
	value := 0
	digits := 0
	for at < end {
		ch := src[at]
		if ch == '_' {
			return value, digits > 0 && at+1 < end
		}
		if ch < '0' || ch > '9' {
			return 0, false
		}
		value = value*10 + int(ch-'0')
		digits++
		at++
	}
	return 0, false
}

// renvoEmitCDirectDeref recognizes the pointer-valued accessors introduced by
// C lowering and folds their final dereference into the addressed load.  It
// returns -1 when the expression is not such an accessor, zero on emission
// failure, and one on success.
func renvoEmitCDirectDeref(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	if !g.c.objectFile || idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	unary := &ep.exprs[idx]
	if unary.kind != renvoExprUnary || !renvoTokCharIs(g.prog, unary.tok, '*') ||
		unary.left < 0 || unary.left >= len(ep.exprs) {
		return -1
	}
	call := &ep.exprs[unary.left]
	if call.kind != renvoExprCall || call.left < 0 || call.left >= len(ep.exprs) {
		return -1
	}
	callee := &ep.exprs[call.left]
	if callee.kind != renvoExprIdent {
		return -1
	}
	displacement := 0
	baseArg := -1
	if offset, ok := renvoCFieldAccessorOffset(g.prog.src, callee.nameStart, callee.nameEnd); ok {
		if call.argCount != 1 {
			return 0
		}
		displacement = offset
		baseArg = renvo_runtime_UnsafeIntAt(ep.args, call.firstArg)
	} else if renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_index_") ||
		renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_array_index_") {
		if call.argCount != 2 {
			return 0
		}
		baseArg = renvo_runtime_UnsafeIntAt(ep.args, call.firstArg)
		indexArg := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+1)
		index := renvoEvalConstExpr(g, ep, indexArg)
		pointer := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, unary.left))
		if pointer.kind != renvoTypePointer {
			return -1
		}
		if !index.ok {
			elementSize := renvoTypeSize(g.meta, pointer.elem)
			kind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind
			size := renvoScalarKindSize(g.c.renvoNativeIntSize, kind)
			if (!renvoTypeKindIsScalarValue(kind) && kind != renvoTypePointer && kind != renvoTypeFunc) ||
				!renvoCanFoldIndexedScalarLoad(g, ep, indexArg, elementSize, size) {
				return -1
			}
			if !renvoEmitIntExpr(g, ep, baseArg) {
				return 0
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			if !renvoEmitIntExpr(g, ep, indexArg) {
				return 0
			}
			renvoAsmFoldedIndexedScalarLoad(&g.asm, elementSize, size, kind == renvoTypeInt8 || kind == renvoTypeInt16)
			return 1
		}
		displacement = index.value * renvoTypeSize(g.meta, pointer.elem)
	} else {
		return -1
	}
	if !renvoEmitIntExpr(g, ep, baseArg) {
		return 0
	}
	renvoEmitRuntimeNonNilPrimary(g)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	kind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind
	size := renvoScalarKindSize(g.c.renvoNativeIntSize, kind)
	renvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, displacement, size)
	renvoAsmNormalizePrimaryForKind(&g.asm, kind)
	return 1
}

func renvoUnsignedConstValue(value int, kind int) uint64 {
	if kind == renvoTypeByte {
		return uint64(uint8(value))
	}
	if kind == renvoTypeUint16 {
		return uint64(uint16(value))
	}
	if kind == renvoTypeUint32 {
		return uint64(uint32(value))
	}
	return uint64(value)
}

func renvoUnsignedKindSize(kind int) int {
	if kind == renvoTypeByte {
		return 1
	}
	if kind == renvoTypeUint16 {
		return 2
	}
	if kind == renvoTypeUint32 {
		return 4
	}
	if kind == renvoTypeUint64 {
		return 8
	}
	return 0
}

func renvoEvalConstBinaryInto(g *renvoLinearGen, tok int, left int, right int, unsignedKind int, out *renvoConstResult) {
	p := g.prog
	if tok < 0 || tok >= renvoTokCount(p) {
		renvoSetConstResult(out, 0, false)
		return
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	n := end - start
	value := 0
	ok := true
	unsignedLeft := renvoUnsignedConstValue(left, unsignedKind)
	unsignedRight := renvoUnsignedConstValue(right, unsignedKind)
	if n == 1 {
		c := renvo_runtime_UnsafeByteAt(p.src, start)
		if c == '+' {
			value = left + right
		} else if c == '-' {
			value = left - right
		} else if c == '*' {
			value = left * right
		} else if c == '/' {
			if right == 0 {
				renvoSetConstResult(out, 0, false)
				return
			}
			if unsignedKind != 0 {
				value = int(unsignedLeft / unsignedRight)
			} else {
				value = left / right
			}
		} else if c == '%' {
			if right == 0 {
				renvoSetConstResult(out, 0, false)
				return
			}
			if unsignedKind != 0 {
				value = int(unsignedLeft % unsignedRight)
			} else {
				value = left % right
			}
		} else if c == '&' {
			value = left & right
		} else if c == '|' {
			value = left | right
		} else if c == '^' {
			value = left ^ right
		} else if c == '<' {
			if unsignedKind != 0 && unsignedLeft < unsignedRight || unsignedKind == 0 && left < right {
				value = 1
			}
		} else if c == '>' {
			if unsignedKind != 0 && unsignedLeft > unsignedRight || unsignedKind == 0 && left > right {
				value = 1
			}
		} else {
			ok = false
		}
	} else if n == 2 {
		c0 := renvo_runtime_UnsafeByteAt(p.src, start)
		c1 := renvo_runtime_UnsafeByteAt(p.src, start+1)
		if c0 == '&' && c1 == '^' {
			value = left & (right ^ -1)
		} else if c0 == '<' && c1 == '<' {
			if p.compilerInt32 && right >= 31 {
				renvoSetConstResult(out, 0, false)
				return
			}
			value = left << right
		} else if c0 == '>' && c1 == '>' {
			if unsignedKind != 0 {
				wide := unsignedLeft >> right
				value = int(wide)
				if p.compilerInt32 && unsignedKind == renvoTypeUint64 && uint64(value) != wide {
					renvoSetConstResult(out, 0, false)
					return
				}
			} else {
				value = left >> right
			}
		} else if c0 == '=' && c1 == '=' {
			if left == right {
				value = 1
			}
		} else if c0 == '!' && c1 == '=' {
			if left != right {
				value = 1
			}
		} else if c0 == '<' && c1 == '=' {
			if unsignedKind != 0 && unsignedLeft <= unsignedRight || unsignedKind == 0 && left <= right {
				value = 1
			}
		} else if c0 == '>' && c1 == '=' {
			if unsignedKind != 0 && unsignedLeft >= unsignedRight || unsignedKind == 0 && left >= right {
				value = 1
			}
		} else {
			ok = false
		}
	} else {
		ok = false
	}
	if ok {
		renvoSetConstResult(out, value, true)
		return
	}
	renvoSetConstResult(out, 0, false)
}

func renvoExprIsIdentText(p *renvoProgram, ep *renvoExprParse, idx int, text string) bool {
	renvoNonNil(p, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return false
	}
	return renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
}

func renvoExprIdentPrefixText(p *renvoProgram, ep *renvoExprParse, idx int, prefix string) bool {
	renvoNonNil(p, ep)
	e := &ep.exprs[idx]
	return e.kind == renvoExprIdent && renvoBytesPrefixText(p.src, e.nameStart, e.nameEnd, prefix)
}

func renvoParseExpressionInto(ep *renvoExprParse, p *renvoProgram, start int, end int) {
	renvoNonNil(ep, p)
	var zero renvoExprParse
	*ep = zero
	ep.prog = p
	ep.pos = start
	ep.end = end
	// An expression cannot initially need more AST nodes, arguments, or
	// composite fields than it has tokens. The old fixed capacities reserved
	// scratch for every small expression; with the bump allocator those
	// reservations accumulated for the entire compile.
	capacity := end - start
	if capacity < 2 {
		capacity = 2
	}
	ep.exprs = make([]renvoExpr, 0, capacity)
	ep.ok = true
	renvoParseBinaryExpr(ep, 1)
	if ep.pos < ep.end {
		renvoExprError(ep)
	}
}

func renvoParseExpressionOK(ep *renvoExprParse, p *renvoProgram, start int, end int) bool {
	renvoNonNil(ep, p)
	return renvoParseExpressionRoot(ep, p, start, end) >= 0
}

func renvoParseExpressionRoot(ep *renvoExprParse, p *renvoProgram, start int, end int) int {
	renvoNonNil(ep, p)
	renvoParseExpressionInto(ep, p, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return -1
	}
	return len(ep.exprs) - 1
}

func renvoParseBinaryExpr(ep *renvoExprParse, minPrec int) int {
	renvoNonNil(ep)
	left := renvoParseUnaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		prec := renvoTokenPrecedence(ep.prog, ep.pos)
		if prec < minPrec {
			break
		}
		opTok := ep.pos
		ep.pos++
		right := renvoParseBinaryExpr(ep, prec+1)
		left = renvoAddExpr(ep, renvoExprBinary, opTok, left, right, 0, 0, 0, 0)
	}
	return left
}

func renvoParseUnaryExpr(ep *renvoExprParse) int {
	renvoNonNil(ep)
	if ep.pos >= ep.end {
		renvoExprError(ep)
		return 0
	}
	first := int(renvo_runtime_UnsafeInt32At(ep.prog.toks.data, ep.pos*renvoTokenStride))
	c := byte(first >> 24)
	if first&255 != renvoTokOp {
		c = 0
	}
	if c == '+' || c == '-' || c == '!' || c == '^' || c == '&' || c == '*' {
		opTok := ep.pos
		ep.pos++
		inner := renvoParseUnaryExpr(ep)
		return renvoAddExpr(ep, renvoExprUnary, opTok, inner, 0, 0, 0, 0, 0)
	}
	return renvoParsePostfixExpr(ep)
}

func renvoParsePostfixExpr(ep *renvoExprParse) int {
	renvoNonNil(ep)
	left := renvoParsePrimaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		first := int(renvo_runtime_UnsafeInt32At(ep.prog.toks.data, ep.pos*renvoTokenStride))
		c := byte(first >> 24)
		if first&255 != renvoTokOp {
			c = 0
		}
		if c == '{' {
			base := ep.exprs[left]
			if base.kind != renvoExprIdent {
				renvoExprError(ep)
				return left
			}
			compositeFields := renvoFixedCompositeFieldScratch(8)
			ep.pos++
			for ep.ok && ep.pos < ep.end && !renvoTokCharIs(ep.prog, ep.pos, '}') {
				compositeFields = append(compositeFields, renvoParseCompositeField(ep))
				if renvoTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !renvoTokCharIs(ep.prog, ep.pos, '}') {
				renvoExprError(ep)
				return left
			}
			ep.pos++
			first := len(ep.fields)
			for i := 0; i < len(compositeFields); i++ {
				field := compositeFields[i]
				ep.fields = append(ep.fields, field)
			}
			count := len(compositeFields)
			left = renvoAddExpr(ep, renvoExprComposite, base.tok, 0, 0, first, count, base.nameStart, base.nameEnd)
			continue
		}
		if c == '(' {
			callTok := ep.pos
			callExpanded := false
			ep.pos++
			argsStart := ep.pos
			scanPos := ep.pos
			count := 0
			for scanPos < ep.end && !renvoTokCharIs(ep.prog, scanPos, ')') {
				argEnd := renvoFindExprBoundary(ep.prog, scanPos, ep.end)
				if renvoTokCharIs(ep.prog, argEnd, '{') {
					closeTok := renvoSkipBalanced(ep.prog, argEnd, '{', '}')
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				count++
				scanPos = argEnd
				if renvoTokCharIs(ep.prog, scanPos, ',') {
					scanPos++
				}
			}
			first := len(ep.args)
			for i := 0; i < count; i++ {
				ep.args = append(ep.args, 0)
			}
			argIndex := 0
			ep.pos = argsStart
			for ep.ok && ep.pos < ep.end && !renvoTokCharIs(ep.prog, ep.pos, ')') {
				argEnd := renvoFindExprBoundary(ep.prog, ep.pos, ep.end)
				if renvoTokCharIs(ep.prog, argEnd, '{') {
					closeTok := renvoSkipBalanced(ep.prog, argEnd, '{', '}')
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				parseEnd := argEnd
				if argEnd-ep.pos >= 4 && renvoTokCharIs(ep.prog, argEnd-3, '.') && renvoTokCharIs(ep.prog, argEnd-2, '.') && renvoTokCharIs(ep.prog, argEnd-1, '.') {
					callExpanded = true
					parseEnd = argEnd - 3
				}
				oldEnd := ep.end
				ep.end = parseEnd
				argRoot := renvoParseBinaryExpr(ep, 1)
				ep.end = oldEnd
				ep.args[first+argIndex] = argRoot
				argIndex++
				ep.pos = argEnd
				if renvoTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !renvoTokCharIs(ep.prog, ep.pos, ')') {
				renvoExprError(ep)
				return left
			}
			ep.pos++
			expanded := 0
			if callExpanded {
				expanded = 1
			}
			left = renvoAddExpr(ep, renvoExprCall, callTok, left, 0, first, count, expanded, 0)
			continue
		}
		if c == '[' {
			indexTok := ep.pos
			ep.pos++
			indexStart := ep.pos
			indexEnd := renvoFindMatchingExprClose(ep.prog, ep.pos, ep.end, '[', ']')
			if indexEnd <= ep.pos {
				renvoExprError(ep)
				return left
			}
			colon := renvoFindSliceColon(ep.prog, indexStart, indexEnd)
			if colon >= 0 {
				low := -1
				high := -1
				max := -1
				highEnd := indexEnd
				secondColon := renvoFindSliceColon(ep.prog, colon+1, indexEnd)
				if secondColon >= 0 {
					highEnd = secondColon
				}
				oldEnd := ep.end
				if colon > indexStart {
					ep.pos = indexStart
					ep.end = colon
					low = renvoParseBinaryExpr(ep, 1)
				}
				if colon+1 < highEnd {
					ep.pos = colon + 1
					ep.end = highEnd
					high = renvoParseBinaryExpr(ep, 1)
				}
				if secondColon >= 0 && secondColon+1 < indexEnd {
					ep.pos = secondColon + 1
					ep.end = indexEnd
					max = renvoParseBinaryExpr(ep, 1)
				}
				ep.end = oldEnd
				ep.pos = indexEnd + 1
				left = renvoAddExpr(ep, renvoExprSlice, indexTok, left, high, low, 0, max, 0)
				continue
			}
			oldEnd := ep.end
			ep.end = indexEnd
			right := renvoParseBinaryExpr(ep, 1)
			ep.end = oldEnd
			ep.pos = indexEnd + 1
			left = renvoAddExpr(ep, renvoExprIndex, indexTok, left, right, 0, 0, 0, 0)
			continue
		}
		if c == '.' && renvoTokIsKind(ep.prog, ep.pos+1, renvoTokIdent) {
			dotTok := ep.pos
			nameTok := renvoTokAt(ep.prog, ep.pos+1)
			ep.pos += 2
			left = renvoAddExpr(ep, renvoExprSelector, dotTok, left, 0, 0, 0, int(nameTok.start), int(nameTok.end))
			continue
		}
		if c == '.' && renvoTokCharIs(ep.prog, ep.pos+1, '(') {
			dotTok := ep.pos
			typeStart := ep.pos + 2
			typeEnd := renvoFindMatchingExprClose(ep.prog, typeStart, ep.end, '(', ')')
			if typeEnd <= typeStart {
				renvoExprError(ep)
				return left
			}
			ep.pos = typeEnd + 1
			left = renvoAddExpr(ep, renvoExprAssert, dotTok, left, typeStart, typeEnd, 0, 0, 0)
			continue
		}
		break
	}
	return left
}

func renvoFindSliceColon(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		c := renvoTokSingleChar(p, i)
		if paren == 0 && brack == 0 && brace == 0 && c == ':' {
			return i
		}
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
		} else if c == '[' {
			brack++
		} else if c == ']' {
			brack--
		} else if c == '{' {
			brace++
		} else if c == '}' {
			brace--
		}
	}
	return -1
}

func renvoParseImplicitCompositeExpr(ep *renvoExprParse) int {
	renvoNonNil(ep)
	openTok := ep.pos
	if !renvoTokCharIs(ep.prog, ep.pos, '{') {
		renvoExprError(ep)
		return 0
	}
	compositeFields := renvoFixedCompositeFieldScratch(8)
	ep.pos++
	for ep.ok && ep.pos < ep.end && !renvoTokCharIs(ep.prog, ep.pos, '}') {
		compositeFields = append(compositeFields, renvoParseCompositeField(ep))
		if renvoTokCharIs(ep.prog, ep.pos, ',') {
			ep.pos++
		}
	}
	if !renvoTokCharIs(ep.prog, ep.pos, '}') {
		renvoExprError(ep)
		return 0
	}
	ep.pos++
	first := len(ep.fields)
	for i := 0; i < len(compositeFields); i++ {
		field := compositeFields[i]
		ep.fields = append(ep.fields, field)
	}
	count := len(compositeFields)
	return renvoAddExpr(ep, renvoExprComposite, openTok, 0, 0, first, count, 0, 0)
}

func renvoParseCompositeField(ep *renvoExprParse) renvoCompositeField {
	renvoNonNil(ep)
	var field renvoCompositeField
	field.key = -1
	fieldEnd := renvoFindExprBoundary(ep.prog, ep.pos, ep.end)
	colon := renvoFindSliceColon(ep.prog, ep.pos, fieldEnd)
	oldEnd := ep.end
	if colon >= ep.pos {
		keyStart := ep.pos
		ep.end = colon
		field.key = renvoParseBinaryExpr(ep, 1)
		if colon == keyStart+1 && renvoTokIsKind(ep.prog, keyStart, renvoTokIdent) {
			field.nameStart = int(renvoTokStart(ep.prog, keyStart))
			field.nameEnd = int(renvoTokEnd(ep.prog, keyStart))
		}
		ep.pos = colon + 1
	}
	ep.end = fieldEnd
	if renvoTokCharIs(ep.prog, ep.pos, '{') {
		field.expr = renvoParseImplicitCompositeExpr(ep)
	} else {
		field.expr = renvoParseBinaryExpr(ep, 1)
	}
	ep.end = oldEnd
	ep.pos = fieldEnd
	return field
}

func renvoParsePrimaryExpr(ep *renvoExprParse) int {
	renvoNonNil(ep)
	if ep.pos >= ep.end {
		renvoExprError(ep)
		return 0
	}
	first := int(renvo_runtime_UnsafeInt32At(ep.prog.toks.data, ep.pos*renvoTokenStride))
	kind := first & 255
	c := byte(first >> 24)
	if kind != renvoTokOp {
		c = 0
	}
	nameStart := 0
	nameEnd := 0
	if kind == renvoTokIdent {
		packed := int(renvo_runtime_UnsafeInt32At(ep.prog.toks.data, ep.pos*renvoTokenStride+1))
		nameStart = packed & 0xffffff
		nameEnd = nameStart + (packed>>24&255 | first>>16&0xff00)
	}
	if renvoFixedTarget == 0 {
		if kind == renvoTokFunc {
			startTok := ep.pos
			typeEnd := renvoPrimaryTypeEnd(ep.prog, startTok, ep.end)
			functionLiteral := typeEnd < ep.end && renvoTokCharIs(ep.prog, typeEnd, '{')
			if typeEnd > startTok && !functionLiteral {
				ep.pos = typeEnd
				return renvoAddExpr(ep, renvoExprIdent, startTok, 0, 0, 0, 0, int(renvoTokStart(ep.prog, startTok)), int(renvoTokEnd(ep.prog, typeEnd-1)))
			}
		}
	}
	if kind == renvoTokStruct ||
		kind == renvoTokIdent && renvoBytesEqualText(ep.prog.src, nameStart, nameEnd, "map") || c == '[' {
		startTok := ep.pos
		typeEnd := renvoPrimaryTypeEnd(ep.prog, startTok, ep.end)
		if typeEnd > startTok {
			ep.pos = typeEnd
			return renvoAddExpr(ep, renvoExprIdent, startTok, 0, 0, 0, 0, int(renvoTokStart(ep.prog, startTok)), int(renvoTokEnd(ep.prog, typeEnd-1)))
		}
	}
	if kind == renvoTokFunc && renvoTokCharIs(ep.prog, ep.pos+1, '(') {
		funcTok := ep.pos
		bodyOpen := renvoFuncLiteralBodyOpen(ep.prog, funcTok, ep.end)
		if bodyOpen < 0 {
			renvoExprError(ep)
			return 0
		}
		bodyEnd := renvoFindMatchingBrace(ep.prog, bodyOpen, ep.end)
		if bodyEnd <= bodyOpen {
			renvoExprError(ep)
			return 0
		}
		ep.pos = bodyEnd + 1
		return renvoAddExpr(ep, renvoExprFunc, funcTok, bodyOpen, bodyEnd, 0, 0, 0, 0)
	}
	if kind == renvoTokIdent {
		ep.pos++
		if renvoBytesEqualText(ep.prog.src, nameStart, nameEnd, "true") {
			return renvoAddExpr(ep, renvoExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		if renvoBytesEqualText(ep.prog.src, nameStart, nameEnd, "false") {
			return renvoAddExpr(ep, renvoExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		return renvoAddExpr(ep, renvoExprIdent, ep.pos-1, 0, 0, 0, 0, nameStart, nameEnd)
	}
	if kind == renvoTokNumber {
		ep.pos++
		return renvoAddExpr(ep, renvoExprInt, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if kind == renvoTokFloat {
		ep.pos++
		ep.hasFloat = true
		return renvoAddExpr(ep, renvoExprFloat, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if kind == renvoTokString {
		ep.pos++
		return renvoAddExpr(ep, renvoExprString, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if kind == renvoTokChar {
		ep.pos++
		return renvoAddExpr(ep, renvoExprChar, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if c == '(' {
		if renvoFixedTarget == 0 {
			closeTok := renvoFindMatchingExprClose(ep.prog, ep.pos+1, ep.end, '(', ')')
			if closeTok > ep.pos+2 && renvoTokCharIs(ep.prog, ep.pos+1, '*') &&
				renvoTokIsKind(ep.prog, ep.pos+2, renvoTokFunc) &&
				renvoPrimaryTypeEnd(ep.prog, ep.pos+1, closeTok) == closeTok {
				starTok := ep.pos + 1
				funcTok := ep.pos + 2
				ep.pos = closeTok + 1
				inner := renvoAddExpr(ep, renvoExprIdent, funcTok, 0, 0, 0, 0,
					int(renvoTokStart(ep.prog, funcTok)), int(renvoTokEnd(ep.prog, closeTok-1)))
				return renvoAddExpr(ep, renvoExprUnary, starTok, inner, 0, 0, 0, 0, 0)
			}
		}
		ep.pos++
		inner := renvoParseBinaryExpr(ep, 1)
		if !renvoTokCharIs(ep.prog, ep.pos, ')') {
			renvoExprError(ep)
			return inner
		}
		ep.pos++
		return inner
	}
	renvoExprError(ep)
	return 0
}

func renvoFuncLiteralBodyOpen(p *renvoProgram, funcTok int, end int) int {
	renvoNonNil(p)
	if funcTok < 0 || funcTok+1 >= end || !renvoTokIsKind(p, funcTok, renvoTokFunc) || !renvoTokCharIs(p, funcTok+1, '(') {
		return -1
	}
	// The body follows the signature's type span. Statement-header scanning
	// treats a brace followed by () as a composite operand and skips it, which
	// loses the body of an immediately invoked literal with a result type.
	bodyOpen := renvoPrimaryTypeEnd(p, funcTok, end)
	if bodyOpen <= funcTok || bodyOpen >= end || !renvoTokCharIs(p, bodyOpen, '{') {
		return -1
	}
	return bodyOpen
}

func renvoPrimaryTypeEnd(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	if start >= end {
		return start
	}
	if renvoTokIsKind(p, start, renvoTokFunc) && renvoTokCharIs(p, start+1, '(') {
		paramsClose := renvoFindMatchingExprClose(p, start+2, end, '(', ')')
		if paramsClose <= start+1 {
			return start
		}
		resultStart := paramsClose + 1
		if resultStart >= end || renvoTokCharIs(p, resultStart, '{') {
			return resultStart
		}
		if renvoTokCharIs(p, resultStart, '(') {
			resultsClose := renvoFindMatchingExprClose(p, resultStart+1, end, '(', ')')
			if resultsClose <= resultStart {
				return start
			}
			return resultsClose + 1
		}
		resultEnd := renvoPrimaryTypeEnd(p, resultStart, end)
		if resultEnd <= resultStart {
			return start
		}
		return resultEnd
	}
	if (renvoTokIsKind(p, start, renvoTokStruct) || renvoTokIdentIs(p, start, "interface")) && renvoTokCharIs(p, start+1, '{') {
		closeTok := renvoFindMatchingBrace(p, start+1, end)
		if closeTok > start+1 {
			return closeTok + 1
		}
		return start
	}
	bracket := start
	if renvoTokIdentIs(p, start, "map") {
		bracket++
	}
	if renvoTokCharIs(p, bracket, '[') {
		closeTok := renvoFindMatchingExprClose(p, bracket+1, end, '[', ']')
		if closeTok > bracket {
			return renvoPrimaryTypeEnd(p, closeTok+1, end)
		}
		return start
	}
	if renvoTokCharIs(p, start, '*') {
		return renvoPrimaryTypeEnd(p, start+1, end)
	}
	if renvoTokIsKind(p, start, renvoTokIdent) {
		return start + 1
	}
	return start
}

func renvoAddExpr(ep *renvoExprParse, kind int, tok int, left int, right int, firstArg int, argCount int, nameStart int, nameEnd int) int {
	renvoNonNil(ep)
	var e renvoExpr
	e.kind = kind
	e.tok = tok
	e.left = left
	e.right = right
	e.firstArg = firstArg
	e.argCount = argCount
	e.nameStart = nameStart
	e.nameEnd = nameEnd
	ep.exprs = append(ep.exprs, e)
	index := len(ep.exprs) - 1
	return index
}

func renvoTokenPrecedence(p *renvoProgram, pos int) int {
	renvoNonNil(p)
	if pos < 0 || pos >= renvoTokCount(p) {
		return 0
	}
	start := renvoTokStart(p, pos)
	end := renvoTokEnd(p, pos)
	if end-start == 1 {
		c := renvo_runtime_UnsafeByteAt(p.src, start)
		if c == '<' || c == '>' {
			return 3
		}
		if c == '+' || c == '-' || c == '|' || c == '^' {
			return 4
		}
		if c == '*' || c == '/' || c == '%' || c == '&' {
			return 5
		}
		return 0
	}
	if end-start == 2 {
		c0 := renvo_runtime_UnsafeByteAt(p.src, start)
		c1 := renvo_runtime_UnsafeByteAt(p.src, start+1)
		if c0 == '|' && c1 == '|' {
			return 1
		}
		if c0 == '&' && c1 == '&' {
			return 2
		}
		if (c0 == '=' || c0 == '!' || c0 == '<' || c0 == '>') && c1 == '=' {
			return 3
		}
		if (c0 == '<' && c1 == '<') || (c0 == '>' && c1 == '>') || (c0 == '&' && c1 == '^') {
			return 5
		}
	}
	return 0
}

func renvoFindExprBoundary(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		first := int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride))
		c := byte(first >> 24)
		if first&255 != renvoTokOp {
			c = 0
		}
		if paren == 0 && brack == 0 && brace == 0 && c == '{' {
			closeTok := renvoSkipBalanced(p, i, '{', '}')
			if closeTok > i {
				i = closeTok
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && (c == ',' || c == ')' || c == ']' || c == '}') {
			return i
		}
		if c == '(' {
			paren++
		} else if c == ')' {
			if paren == 0 {
				return i
			}
			paren--
		} else if c == '[' {
			brack++
		} else if c == ']' {
			if brack == 0 {
				return i
			}
			brack--
		} else if c == '{' {
			brace++
		} else if c == '}' {
			if brace == 0 {
				return i
			}
			brace--
		}
		i++
	}
	return i
}

func renvoFindMatchingExprClose(p *renvoProgram, start int, end int, open byte, close byte) int {
	renvoNonNil(p)
	depth := 0
	i := start
	for i < end {
		c := renvoTokSingleChar(p, i)
		if c == open {
			depth++
		} else if c == close {
			if depth == 0 {
				return i
			}
			depth--
		}
		i++
	}
	return start
}

func renvoParseOneStatement(bp *renvoBodyParse, start int, end int) int {
	renvoNonNil(bp)
	p := bp.prog
	if start >= end {
		return end
	}
	startKind := renvoTokKind(p, start)
	if startKind == renvoTokReturn {
		if renvoTokCharIs(p, start+1, ';') {
			renvoAddStmt(bp, renvoStmtReturn, start, start+1, start+1, start+1, 0, 0, 0, 0, 0, 0)
			return start + 1
		}
		exprEnd := renvoStatementLineEnd(p, start+1, end)
		renvoAddStmt(bp, renvoStmtReturn, start, exprEnd, start+1, exprEnd, 0, 0, 0, 0, 0, 0)
		return exprEnd
	}
	if startKind == renvoTokIdent && renvoTokIdentIs(p, start, "defer") {
		exprEnd := renvoStatementLineEnd(p, start+1, end)
		renvoAddStmt(bp, renvoStmtDefer, start, exprEnd, start+1, exprEnd, 0, 0, 0, 0, 0, 0)
		return exprEnd
	}
	if startKind == renvoTokIf {
		bodyStart := renvoFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := renvoFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		stmt := renvoStmt{kind: renvoStmtIf, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd}
		next := bodyEnd + 1
		if renvoTokIsKind(p, next, renvoTokElse) {
			if renvoTokIsKind(p, next+1, renvoTokIf) {
				foundEnd := renvoFindIfStatementEnd(p, next+1, end)
				if foundEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 1
				stmt.elseEnd = foundEnd
				stmt.endTok = foundEnd
				next = foundEnd
			} else if renvoTokCharIs(p, next+1, '{') {
				elseBodyEnd := renvoFindMatchingBrace(p, next+1, end)
				if elseBodyEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 2
				stmt.elseEnd = elseBodyEnd
				stmt.endTok = elseBodyEnd + 1
				next = elseBodyEnd + 1
			}
		}
		renvoAppendStmt(bp, stmt)
		return next
	}
	if startKind == renvoTokSwitch {
		bodyStart := renvoFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := renvoFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		renvoAddStmt(bp, renvoStmtSwitch, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if startKind == renvoTokFor {
		bodyStart := renvoFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := renvoFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		renvoAddStmt(bp, renvoStmtFor, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if renvoTokCharIs(p, start, '{') {
		bodyEnd := renvoFindMatchingBrace(p, start, end)
		if bodyEnd <= start {
			return start
		}
		renvoAddStmt(bp, renvoStmtBlock, start, bodyEnd+1, 0, 0, start+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if startKind == renvoTokBreak || startKind == renvoTokContinue || startKind == renvoTokGoto {
		endTok := renvoStatementLineEnd(p, start+1, end)
		kind := renvoStmtGoto
		target := 0
		if startKind == renvoTokBreak {
			kind = renvoStmtBreak
		} else if startKind == renvoTokContinue {
			kind = renvoStmtContinue
		}
		nameStart := 0
		nameEnd := 0
		if start+1 < endTok && renvoTokIsKind(p, start+1, renvoTokIdent) {
			nameStart = int(renvoTokStart(p, start+1))
			nameEnd = int(renvoTokEnd(p, start+1))
			if kind == renvoStmtBreak {
				target = 1
				kind = renvoStmtGoto
			} else if kind == renvoStmtContinue {
				target = 2
				kind = renvoStmtGoto
			}
		}
		renvoAddStmt(bp, kind, start, endTok, target, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	if startKind == renvoTokIdent && renvoTokCharIs(p, start+1, ':') {
		name := renvoTokAt(p, start)
		renvoAddStmt(bp, renvoStmtLabel, start, start+2, 0, 0, 0, 0, 0, 0, int(name.start), int(name.end))
		return start + 2
	}
	if startKind == renvoTokType {
		endTok := renvoStatementLineEnd(p, start+1, end)
		renvoAddStmt(bp, renvoStmtType, start, endTok, 0, 0, 0, 0, 0, 0, 0, 0)
		return endTok
	}
	if startKind == renvoTokVar || startKind == renvoTokConst {
		endTok := renvoStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if renvoTokIsKind(p, start+1, renvoTokIdent) {
			nameStart = int(renvoTokStart(p, start+1))
			nameEnd = int(renvoTokEnd(p, start+1))
		}
		renvoAddStmt(bp, renvoStmtVar, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	lineEnd := renvoStatementLineEnd(p, start, end)
	assignTok := renvoFindAssignmentToken(p, start, lineEnd)
	if assignTok > start {
		kind := renvoStmtAssign
		if renvoTok2Is(p, assignTok, ':', '=') {
			kind = renvoStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if startKind == renvoTokIdent {
			nameStart = int(renvoTokStart(p, start))
			nameEnd = int(renvoTokEnd(p, start))
		}
		renvoAddStmt(bp, kind, start, lineEnd, assignTok+1, lineEnd, 0, 0, 0, 0, nameStart, nameEnd)
		return lineEnd
	}
	renvoAddStmt(bp, renvoStmtExpr, start, lineEnd, start, lineEnd, 0, 0, 0, 0, 0, 0)
	return lineEnd
}

func renvoAddStmt(bp *renvoBodyParse, kind int, startTok int, endTok int, exprStart int, exprEnd int, bodyStart int, bodyEnd int, elseStart int, elseEnd int, nameStart int, nameEnd int) {
	renvoNonNil(bp)
	var stmt renvoStmt
	stmt.kind = kind
	stmt.startTok = startTok
	stmt.endTok = endTok
	stmt.exprStart = exprStart
	stmt.exprEnd = exprEnd
	stmt.bodyStart = bodyStart
	stmt.bodyEnd = bodyEnd
	stmt.elseStart = elseStart
	stmt.elseEnd = elseEnd
	stmt.nameStart = nameStart
	stmt.nameEnd = nameEnd
	renvoAppendStmt(bp, stmt)
}

func renvoAppendStmt(bp *renvoBodyParse, stmt renvoStmt) {
	renvoNonNil(bp)
	if bp.stmtCount != 0 {
		bp.ok = false
		return
	}
	bp.stmt = stmt
	bp.stmtCount = 1
}

func renvoBodyStmtAt(bp *renvoBodyParse, index int) renvoStmt {
	renvoNonNil(bp)
	if index != 0 || bp.stmtCount != 1 {
		bp.ok = false
		return renvoStmt{}
	}
	return bp.stmt
}

func renvoFindIfStatementEnd(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	if !(renvoTokIsKind(p, start, renvoTokIf)) {
		return start
	}
	bodyStart := renvoFindStatementBodyOpen(p, start+1, end)
	if bodyStart <= start {
		return start
	}
	bodyEnd := renvoFindMatchingBrace(p, bodyStart, end)
	if bodyEnd <= bodyStart {
		return start
	}
	next := bodyEnd + 1
	if renvoTokIsKind(p, next, renvoTokElse) {
		if renvoTokIsKind(p, next+1, renvoTokIf) {
			return renvoFindIfStatementEnd(p, next+1, end)
		}
		if renvoTokCharIs(p, next+1, '{') {
			elseEnd := renvoFindMatchingBrace(p, next+1, end)
			if elseEnd <= next+1 {
				return start
			}
			return elseEnd + 1
		}
	}
	return next
}

func renvoStatementLineEnd(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	if start >= end {
		return end
	}
	data := p.toks.data
	line := renvoTokLine(p, start)
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		base := i * renvoTokenStride
		packed := int(renvo_runtime_UnsafeInt32At(data, base))
		kind := packed & 255
		c := byte(packed >> 24)
		if kind != renvoTokOp {
			c = 0
		}
		tokLine := renvoTokLine(p, i)
		if i > start && paren == 0 && brack == 0 && brace == 0 {
			if kind == renvoTokEOF {
				return i
			}
			if c == ';' {
				return i
			}
			if tokLine != line {
				if c == '{' {
					return i
				}
				if kind == renvoTokReturn || kind == renvoTokIf || kind == renvoTokFor || kind == renvoTokSwitch || kind == renvoTokCase || kind == renvoTokDefault || kind == renvoTokVar || kind == renvoTokConst || kind == renvoTokBreak || kind == renvoTokContinue || kind == renvoTokGoto {
					return i
				}
				if renvoLineContinuesAfterPrevToken(p, i) {
					line = tokLine
				} else {
					return i
				}
			}
		}
		closed := false
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
			closed = true
		} else if c == '[' {
			brack++
		} else if c == ']' {
			brack--
			closed = true
		} else if c == '{' {
			brace++
		} else if c == '}' {
			if brace == 0 {
				return i
			}
			brace--
			closed = true
		}
		if i > start && tokLine != line && paren == 0 && brack == 0 && brace == 0 {
			if renvoLineContinuesAfterPrevToken(p, i) {
				line = tokLine
			} else {
				if closed {
					if c == '}' && renvoTokLine(p, i+1) == tokLine && !renvoTokCharIs(p, i+1, ';') {
						line = tokLine
						i++
						continue
					}
					return i + 1
				}
				return i
			}
		}
		i++
	}
	return i
}

func renvoLineContinuesAfterPrevToken(p *renvoProgram, i int) bool {
	renvoNonNil(p)
	// Both statement scanner call sites only ask after consuming a token.
	prev := i - 1
	tok := renvoTokAt(p, prev)
	tokStart := tok.start
	tokEnd := tok.end
	if tokEnd <= tokStart {
		return false
	}
	c := renvo_runtime_UnsafeByteAt(p.src, tokStart)
	if c == '=' || c == ':' && tokEnd == tokStart+2 && renvo_runtime_UnsafeByteAt(p.src, tokStart+1) == '=' {
		return true
	}
	if c == ',' || c == '*' || c == '&' || c == '|' {
		return true
	}
	return c == '+' &&
		(tokEnd == tokStart+1 || renvo_runtime_UnsafeByteAt(p.src, tokStart+1) != '+')
}

func renvoFindNextTokenText(p *renvoProgram, start int, end int, text byte) int {
	renvoNonNil(p)
	i := start
	for i < end {
		if renvoTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start
}

func renvoFindStatementBodyOpen(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	i := start
	paren := 0
	brack := 0
	for i < end {
		tok := renvoTokAt(p, i)
		if tok.end == tok.start+1 {
			c := renvo_runtime_UnsafeByteAt(p.src, tok.start)
			if c == '(' {
				paren++
			} else if c == ')' {
				if paren > 0 {
					paren--
				}
			} else if c == '[' {
				brack++
			} else if c == ']' {
				if brack > 0 {
					brack--
				}
			} else if c == '{' {
				if paren == 0 && brack == 0 {
					closeTok := renvoSkipBalanced(p, i, '{', '}')
					if closeTok > i && closeTok < end {
						next := renvoTokSingleChar(p, closeTok)
						sameLine := renvoTokLine(p, closeTok-1) == renvoTokLine(p, closeTok)
						if sameLine && (renvoTokenPrecedence(p, closeTok) > 0 || next == '{' || next == '.' || next == '[' || next == '(') {
							i = closeTok
							continue
						}
					}
					return i
				}
				closeTok := renvoSkipBalanced(p, i, '{', '}')
				if closeTok > i {
					i = closeTok
					continue
				}
			}
		}
		i++
	}
	return start
}

func renvoFindMatchingBrace(p *renvoProgram, openTok int, end int) int {
	renvoNonNil(p)
	if renvoTokSingleChar(p, openTok) != '{' {
		return openTok
	}
	depth := 1
	i := openTok + 1
	for i < end {
		c := renvoTokSingleChar(p, i)
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return openTok
}

func renvoFindAssignmentToken(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	i := start
	paren := 0
	brack := 0
	for i < end {
		first := int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride))
		c := byte(first >> 24)
		if first&255 != renvoTokOp {
			c = 0
		}
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
		} else if c == '[' || c == '{' {
			brack++
		} else if c == ']' || c == '}' {
			brack--
		} else if paren == 0 && brack == 0 {
			if c == '=' || renvoTok2Is(p, i, ':', '=') || renvoTokIsCompoundAssignment(p, i) {
				return i
			}
		}
		i++
	}
	return start
}

func renvoTokIsCompoundAssignment(p *renvoProgram, tok int) bool {
	renvoNonNil(p)
	if !renvoTokIsKind(p, tok, renvoTokOp) {
		return false
	}
	start := int(renvoTokStart(p, tok))
	end := int(renvoTokEnd(p, tok))
	if end-start == 2 && renvo_runtime_UnsafeByteAt(p.src, start+1) == '=' {
		operator := renvo_runtime_UnsafeByteAt(p.src, start)
		return operator == '+' || operator == '-' || operator == '*' || operator == '/' || operator == '%' || operator == '&' || operator == '|' || operator == '^'
	}
	if end-start != 3 || renvo_runtime_UnsafeByteAt(p.src, start+2) != '=' {
		return false
	}
	first := renvo_runtime_UnsafeByteAt(p.src, start)
	second := renvo_runtime_UnsafeByteAt(p.src, start+1)
	return first == '<' && second == '<' || first == '>' && second == '>' || first == '&' && second == '^'
}

func renvoBuildMeta(pp *renvoProgram) renvoMeta {
	renvoNonNil(pp)
	var m renvoMeta
	renvoBuildMetaInto(pp, &m)
	return m
}

func renvoBuildMetaInto(pp *renvoProgram, m *renvoMeta) {
	renvoNonNil(pp, m)
	m.c = &pp.c
	m.scratchStart = renvo_runtime_ArenaMark()
	p := pp
	m.prog = p
	// Most derived types are shared, so reserving four entries per declaration
	// leaves a large unused table for linked frontend units. The slice still
	// grows normally for type-heavy inputs.
	typeCap := len(p.decls) + 512
	fieldCap := len(p.decls)*2 + 256
	globalCap := len(p.decls) + 128
	paramCap := len(p.funcs)*3 + 256
	m.types = make([]renvoTypeInfo, 0, typeCap)
	m.fields = make([]renvoFieldInfo, 0, fieldCap)
	m.globals = make([]renvoSymbolInfo, 0, globalCap)
	m.params = make([]renvoSymbolInfo, 0, paramCap)
	m.funcs = make([]renvoFuncInfo, 0, len(p.funcs)+128)
	m.typeBuckets = make([]int32, typeCap*2)
	m.closures = make([]renvoClosureInfo, 0, 16)
	m.captures = make([]renvoSymbolInfo, 0, 32)
	m.globalBuckets = make([]int32, globalCap*2)
	if renvoFixedTarget == 0 {
		m.objectDecls = append(m.objectDecls, renvoObjectDecl{})
	}
	for i := 0; i < len(m.globalBuckets); i++ {
		m.globalBuckets[i] = -1
	}
	m.funcBuckets = make([]int32, (len(p.funcs)+128)*2)
	for i := 0; i < len(m.funcBuckets); i++ {
		m.funcBuckets[i] = -1
	}
	m.ok = true
	renvoInitBuiltinTypes(m)

	parsedGroupStart := -1
	parsedGroupEnd := -1
	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != renvoTokConst {
			continue
		}
		if !renvoTokIsKind(p, decl.startTok, decl.kind) {
			groupStart, groupEnd, isGroup := renvoFindContainingTopDeclGroup(p, decl.kind, decl.startTok, decl.endTok)
			if isGroup {
				if parsedGroupStart == groupStart && parsedGroupEnd == groupEnd {
					continue
				}
				renvoParseTopDeclGroup(m, p, decl.kind, groupStart+1, groupEnd)
				parsedGroupStart = groupStart
				parsedGroupEnd = groupEnd
				continue
			}
			renvoParseTopDeclEntry(m, p, decl.kind, decl.startTok, decl.endTok)
			continue
		}
		entryStart := decl.startTok + 1
		if renvoTokCharIs(p, entryStart, '(') {
			renvoParseTopDeclGroup(m, p, decl.kind, entryStart, decl.endTok)
			continue
		}
		renvoParseConstDecls(m, p, entryStart, decl.endTok)
	}

	parsedGroupStart = -1
	parsedGroupEnd = -1
	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != renvoTokType && decl.kind != renvoTokVar {
			continue
		}
		if !renvoTokIsKind(p, decl.startTok, decl.kind) {
			groupStart, groupEnd, isGroup := renvoFindContainingTopDeclGroup(p, decl.kind, decl.startTok, decl.endTok)
			if isGroup {
				if parsedGroupStart == groupStart && parsedGroupEnd == groupEnd {
					continue
				}
				renvoParseTopDeclGroup(m, p, decl.kind, groupStart+1, groupEnd)
				parsedGroupStart = groupStart
				parsedGroupEnd = groupEnd
				continue
			}
			renvoParseTopDeclEntry(m, p, decl.kind, decl.startTok, decl.endTok)
			continue
		}
		entryStart := decl.startTok + 1
		if renvoTokCharIs(p, entryStart, '(') {
			renvoParseTopDeclGroup(m, p, decl.kind, entryStart, decl.endTok)
			continue
		}
		renvoParseTopDeclEntry(m, p, decl.kind, entryStart, decl.endTok)
	}
	for i := 0; i < len(p.funcs); i++ {
		renvoParseFuncInfo(m, i)
	}
	renvoParseFuncLiterals(m, p)
	// C identifiers may legitimately be named panic or recover, and C has none
	// of Go's panic semantics. Cleanup attributes are intentionally lowered to
	// Go defer, however, so retain the unwind machinery only for C units which
	// contain an actual lowered defer statement.
	m.panicEnabled = p.toks.panicEnabled
	if renvoProgramUsesC11Semantics(p) {
		m.panicEnabled = m.panicEnabled && renvoC11UsesDefer(p)
	}
	renvoFinalizeTypeLayouts(m)
	renvoBuildFuncLookup(m)
	renvoResolveGlobalInitTypes(m)
	if m.panicEnabled && !renvoProgramUsesC11Semantics(p) {
		m.panicEnabled = renvoReachablePanicRequired(m)
	}
	m.scratchEnd = renvo_runtime_ArenaMark()
}

// The scanner's hint includes unused library bodies. Only omit unwind state
// when a conservative executable call graph proves those bodies unreachable.
// Function values retain every signature-compatible dispatch target. Opaque
// entrypoints and anonymous functions conservatively keep unwind support.
type renvoPanicReachability struct {
	meta   *renvoMeta
	seen   []bool
	values []bool
	queue  []int
}

func renvoPanicReachMark(state *renvoPanicReachability, fn int) {
	if fn >= 0 && fn < len(state.seen) && !state.seen[fn] {
		state.seen[fn] = true
		state.queue = append(state.queue, fn)
	}
}

func renvoPanicReachFunctionValue(state *renvoPanicReachability, fnIndex int) {
	if state.values[fnIndex] {
		return
	}
	state.values[fnIndex] = true
	m := state.meta
	typ := renvoFunctionTypeFromInfo(m, fnIndex)
	for i := 0; i < len(m.funcs); i++ {
		if renvoFunctionValueMode(m, i, typ) != 0 {
			renvoPanicReachMark(state, i)
		}
	}
	if m.funcs[fnIndex].receiverType != 0 {
		typ = renvoFunctionTypeFromInfoStart(m, fnIndex, 0)
		for i := 0; i < len(m.funcs); i++ {
			if renvoFunctionValueMode(m, i, typ) != 0 {
				renvoPanicReachMark(state, i)
			}
		}
	}
}

func renvoPanicReachRange(state *renvoPanicReachability, start int, end int) bool {
	m := state.meta
	p := m.prog
	data := p.toks.data
	for tok := start; tok < end; tok++ {
		base := tok * renvoTokenStride
		first := int(renvo_runtime_UnsafeInt32At(data, base))
		kind := first & 255
		if kind == renvoTokFunc {
			return true
		}
		selector := false
		if tok > start {
			previous := int(renvo_runtime_UnsafeInt32At(data, base-renvoTokenStride))
			selector = previous&255 == renvoTokOp && byte(previous>>24) == '.'
		}
		if kind == renvoTokOp && byte(first>>24) == '(' && selector {
			return true
		}
		if kind != renvoTokIdent {
			continue
		}
		nextChar := byte(0)
		if tok+1 < end {
			next := int(renvo_runtime_UnsafeInt32At(data, base+renvoTokenStride))
			if next&255 == renvoTokOp {
				nextChar = byte(next >> 24)
			}
		}
		if nextChar == ':' {
			continue
		}
		packed := int(renvo_runtime_UnsafeInt32At(data, base+1))
		nameStart := packed & 0xffffff
		size := packed>>24&255 | first>>16&0xff00
		nameEnd := nameStart + size
		if size == 5 && (renvoBytesEqualText(p.src, nameStart, nameEnd, "panic") || renvoBytesEqualText(p.src, nameStart, nameEnd, "defer")) || size == 7 && renvoBytesEqualText(p.src, nameStart, nameEnd, "recover") {
			return true
		}
		bucket := renvoHashRange(p.src, nameStart, nameEnd) % len(m.funcBuckets)
		index := int(renvo_runtime_UnsafeInt32At(m.funcBuckets, bucket))
		for index >= 0 {
			fn := &m.funcs[index]
			if selector == (fn.receiverType != 0) && renvoBytesEqualRange(p.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) {
				if nextChar != '(' {
					renvoPanicReachFunctionValue(state, index)
				}
				renvoPanicReachMark(state, index)
			}
			index = int(renvo_runtime_UnsafeInt32At(m.funcNext, index))
		}
	}
	return false
}

func renvoReachablePanicRequired(m *renvoMeta) bool {
	p := m.prog
	if m.c.objectFile || m.c.emitImage || targetIsKernelModule(m.c) || renvoPreparedBackendActive != 0 || p.entryFunc < 0 || p.entryFunc >= len(m.funcs) {
		return true
	}
	var state renvoPanicReachability
	state.meta = m
	state.seen = make([]bool, len(m.funcs))
	state.values = make([]bool, len(m.funcs))
	state.queue = make([]int, 0, len(m.funcs))
	renvoPanicReachMark(&state, p.entryFunc)
	for i := 0; i < len(m.funcs); i++ {
		fn := &m.funcs[i]
		if fn.exportNameEnd > fn.exportNameStart || fn.nameEnd-fn.nameStart >= 11 && renvoBytesEqualText(p.src, fn.nameStart, fn.nameStart+11, "__renvoSoft") {
			renvoPanicReachMark(&state, i)
		}
	}
	for i := 0; i < len(m.globals); i++ {
		global := &m.globals[i]
		if renvoPanicReachRange(&state, global.initStart, global.initEnd) {
			return true
		}
	}
	for cursor := 0; cursor < len(state.queue); cursor++ {
		fn := &m.funcs[state.queue[cursor]]
		if fn.linkStatic != 0 {
			continue
		}
		if renvoPanicReachRange(&state, fn.bodyStart, fn.bodyEnd) {
			return true
		}
	}
	return false
}

func renvoC11UsesDefer(p *renvoProgram) bool {
	renvoNonNil(p)
	for tok := 0; tok < renvoTokCount(p); tok++ {
		if renvoTokIdentIs(p, tok, "defer") {
			return true
		}
	}
	return false
}

func renvoFindContainingTopDeclGroup(p *renvoProgram, kind int, start int, end int) (int, int, bool) {
	renvoNonNil(p)
	i := start - 1
	for i >= 0 {
		if renvoTokIsKind(p, i, kind) && renvoTokCharIs(p, i+1, '(') {
			groupClose := renvoSkipBalanced(p, i+1, '(', ')')
			if groupClose >= end {
				return i, groupClose + 1, true
			}
		}
		i--
	}
	return 0, 0, false
}

func renvoParseTopDeclGroup(m *renvoMeta, p *renvoProgram, kind int, openTok int, endTok int) {
	renvoParseScopedDeclGroup(nil, m, p, kind, openTok, endTok)
}

func renvoParseScopedDeclGroup(g *renvoLinearGen, m *renvoMeta, p *renvoProgram, kind int, openTok int, endTok int) {
	renvoNonNil(m, p)
	if !renvoTokCharIs(p, openTok, '(') || endTok <= openTok+1 {
		renvoMetaError(m)
		return
	}
	groupEnd := endTok
	if renvoTokCharIs(p, endTok-1, ')') {
		groupEnd = endTok - 1
	}
	if kind == renvoTokConst {
		renvoParseConstDecls(m, p, openTok+1, groupEnd)
		return
	}
	j := openTok + 1
	for j < groupEnd {
		if renvoTokIsKind(p, j, renvoTokIdent) {
			entryEnd := renvoStatementLineEnd(p, j, groupEnd)
			renvoParseScopedDeclEntry(g, m, p, kind, j, entryEnd)
			if entryEnd <= j {
				j++
			} else {
				j = entryEnd
			}
		} else {
			j++
		}
	}
}

func renvoHashRange(src []byte, start int, end int) int {
	size := end - start
	if size <= 0 {
		return 0
	}
	// This is only a bucket fingerprint; every lookup still verifies all bytes.
	// Sampling keeps identifier lookup bounded even for long generated names.
	return (size&127)<<24 | int(renvo_runtime_UnsafeByteAt(src, start))<<16 | int(renvo_runtime_UnsafeByteAt(src, end-1))<<8 | int(renvo_runtime_UnsafeByteAt(src, start+size/2))
}

func renvoMetaAppendGlobal(m *renvoMeta, sym renvoSymbolInfo) {
	renvoNonNil(m)
	index := len(m.globals)
	m.globals = append(m.globals, sym)
	if len(m.globalBuckets) == 0 {
		return
	}
	m.globalNext = append(m.globalNext, -1)
	hash := renvoHashRange(m.prog.src, sym.nameStart, sym.nameEnd)
	bucket := hash % len(m.globalBuckets)
	m.globalNext[index] = renvo_runtime_UnsafeInt32At(m.globalBuckets, bucket)
	m.globalBuckets[bucket] = int32(index)
}

func renvoFindMetaGlobalIndex(m *renvoMeta, nameStart int, nameEnd int, kind int) int {
	renvoNonNil(m)
	hash := renvoHashRange(m.prog.src, nameStart, nameEnd)
	i := int(renvo_runtime_UnsafeInt32At(m.globalBuckets, hash%len(m.globalBuckets)))
	for i >= 0 {
		s := m.globals[i]
		if s.kind == kind && renvoBytesEqualRange(m.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = int(renvo_runtime_UnsafeInt32At(m.globalNext, i))
	}
	return -1
}

func renvoBuildFuncLookup(m *renvoMeta) {
	renvoNonNil(m)
	m.funcNext = make([]int32, len(m.funcs))
	for i := 0; i < len(m.funcNext); i++ {
		m.funcNext[i] = -1
	}
	for i := 0; i < len(m.funcs); i++ {
		f := &m.funcs[i]
		hash := renvoHashRange(m.prog.src, f.nameStart, f.nameEnd)
		bucket := hash % len(m.funcBuckets)
		m.funcNext[i] = renvo_runtime_UnsafeInt32At(m.funcBuckets, bucket)
		m.funcBuckets[bucket] = int32(i)
	}
}

func renvoFindMetaFunction(m *renvoMeta, nameStart int, nameEnd int) int {
	renvoNonNil(m)
	hash := renvoHashRange(m.prog.src, nameStart, nameEnd)
	i := int(renvo_runtime_UnsafeInt32At(m.funcBuckets, hash%len(m.funcBuckets)))
	for i >= 0 {
		fn := &m.funcs[i]
		if fn.receiverType == 0 && renvoBytesEqualRange(m.prog.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = int(renvo_runtime_UnsafeInt32At(m.funcNext, i))
	}
	return -1
}

func renvoResolveGlobalInitTypes(m *renvoMeta) {
	renvoNonNil(m)
	gen := new(renvoLinearGen)
	gen.prog = m.prog
	gen.meta = m
	changed := true
	for changed {
		changed = false
		for i := 0; i < len(m.globals); i++ {
			global := &m.globals[i]
			// Constants retain their untyped representation until they are
			// materialized in an expression context. The integer constant
			// evaluator cannot carry an IEEE value and its exact type together.
			if global.kind != renvoTokVar || global.typ != 0 || global.initStart >= global.initEnd {
				continue
			}
			ep := renvoNewExprParse()
			renvoNonNil(ep)
			rootIndex := renvoParseExpressionRoot(ep, m.prog, global.initStart, global.initEnd)
			if rootIndex < 0 {
				continue
			}
			root := &ep.exprs[rootIndex]
			typ := 0
			if root.kind == renvoExprIdent {
				dependency := renvoFindMetaGlobalIndex(m, root.nameStart, root.nameEnd, renvoTokVar)
				if dependency >= 0 {
					typ = m.globals[dependency].typ
				} else {
					fnIndex := renvoFindMetaFunction(m, root.nameStart, root.nameEnd)
					if fnIndex >= 0 {
						typ = renvoFunctionTypeFromInfo(m, fnIndex)
					}
				}
			} else {
				typ = renvoInferParsedExprType(gen, ep, rootIndex)
			}
			if typ != 0 {
				global.typ = typ
				changed = true
			}
		}
	}
}

func renvoInitBuiltinTypes(m *renvoMeta) {
	renvoNonNil(m)
	renvoAddBuiltinType(m, renvoTypeInvalid, 0)
	renvoAddBuiltinType(m, renvoTypeInt, m.c.renvoNativeIntSize)
	renvoAddBuiltinType(m, renvoTypeInt64, 8)
	renvoAddBuiltinType(m, renvoTypeByte, 1)
	renvoAddBuiltinType(m, renvoTypeBool, 1)
	renvoAddBuiltinType(m, renvoTypeString, renvoBackendStringValueSize)
	renvoAddBuiltinType(m, renvoTypeFloat64, 8)
	renvoAddBuiltinType(m, renvoTypeInt8, 1)
	renvoAddBuiltinType(m, renvoTypeInt16, 2)
	renvoAddBuiltinType(m, renvoTypeInt32, 4)
	renvoAddBuiltinType(m, renvoTypeUint16, 2)
	renvoAddBuiltinType(m, renvoTypeUint32, 4)
	renvoAddBuiltinType(m, renvoTypeUint64, 8)
	renvoAddBuiltinType(m, renvoTypeComplex, 2*renvoBackendValueSlotSize)
	renvoAddBuiltinType(m, renvoTypeInterface, 2*renvoBackendValueSlotSize)
	renvoAddBuiltinType(m, renvoTypeFloat32, 4)
	renvoAddBuiltinType(m, renvoTypeComplex64, 8)
	renvoAddBuiltinType(m, renvoTypeInterface, 2*renvoBackendValueSlotSize)
	m.types[renvoBuiltinTypeError].elem = -1
}

func renvoAddBuiltinType(m *renvoMeta, kind int, size int) {
	renvoNonNil(m)
	m.types = append(m.types, renvoTypeInfo{kind: kind, size: size})
}

func renvoParseConstDecls(m *renvoMeta, p *renvoProgram, start int, end int) {
	renvoNonNil(m, p)
	prevTypeStart := 0
	prevTypeEnd := 0
	prevValues := renvoFixedIntScratch(8)
	iotaValue := 0
	j := start
	for j < end {
		if !renvoTokIsKind(p, j, renvoTokIdent) {
			j++
			continue
		}
		specEnd := renvoStatementLineEnd(p, j, end)
		if specEnd <= j {
			renvoMetaError(m)
			return
		}
		eq := renvoFindConstSpecEqual(p, j, specEnd)
		headEnd := specEnd
		if eq > j {
			headEnd = eq
		}
		names := renvoFixedIntScratch(4)
		k := j
		for k < headEnd {
			if !renvoTokIsKind(p, k, renvoTokIdent) {
				break
			}
			names = append(names, k)
			k++
			if renvoTokCharIs(p, k, ',') {
				k++
				continue
			}
			break
		}
		if len(names) == 0 {
			renvoMetaError(m)
			return
		}
		if eq > j {
			prevTypeStart = k
			prevTypeEnd = headEnd
			newValues, ok := renvoSplitTopLevelComma(p, eq+1, specEnd)
			if !ok {
				renvoMetaError(m)
				return
			}
			prevValues = newValues
		}
		valueCount := len(prevValues) / 2
		if valueCount == 0 {
			renvoMetaError(m)
			return
		}
		if valueCount != len(names) {
			renvoMetaError(m)
			return
		}
		typ := 0
		if prevTypeStart < prevTypeEnd {
			typeResult := renvoParseType(m, p, prevTypeStart, prevTypeEnd)
			typ = typeResult.typ
		}
		for i := 0; i < len(names); i++ {
			nameTok := names[i]
			name := renvoTokAt(p, nameTok)
			if renvoBytesEqualText(p.src, int(name.start), int(name.end), "_") {
				continue
			}
			initStart := prevValues[i*2]
			initEnd := prevValues[i*2+1]
			constantExpr := renvoNewExprParse()
			constantRoot := renvoParseExpressionRoot(constantExpr, p, initStart, initEnd)
			constType := typ
			if constType == 0 && constantRoot >= 0 {
				var g renvoLinearGen
				g.c = m.c
				g.meta = m
				g.prog = p
				constType = renvoInferParsedExprType(&g, constantExpr, constantRoot)
			}
			if constType == 0 {
				constType = renvoTypeInt
			}
			var sym renvoSymbolInfo
			sym.nameStart = int(name.start)
			sym.nameEnd = int(name.end)
			sym.kind = renvoTokConst
			sym.typ = constType
			sym.initStart = initStart
			sym.initEnd = initEnd
			sym.iotaValue = iotaValue
			constResult := renvoEvalMetaParsedConstExpr(m, p, constantExpr, constantRoot, iotaValue)
			if constResult.ok && (initStart+1 == initEnd || !renvoTypeKindIsFloat(renvoResolveType(m, constType).kind)) {
				sym.constValue = constResult.value
				sym.constValueOK = 1
			}
			renvoMetaAppendGlobal(m, sym)
		}
		iotaValue++
		j = specEnd
	}
}

func renvoEvalMetaConstExpr(m *renvoMeta, p *renvoProgram, start int, end int, iotaValue int) renvoConstResult {
	renvoNonNil(m, p)
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	if !renvoParseExpressionOK(ep, p, start, end) {
		var r renvoConstResult
		return r
	}
	rootIndex := len(ep.exprs) - 1
	return renvoEvalMetaParsedConstExpr(m, p, ep, rootIndex, iotaValue)
}

func renvoEvalMetaParsedConstExpr(m *renvoMeta, p *renvoProgram, ep *renvoExprParse, idx int, iotaValue int) renvoConstResult {
	renvoNonNil(m, p, ep)
	var result renvoConstResult
	renvoEvalMetaParsedConstExprInto(m, p, ep, idx, iotaValue, &result)
	return result
}

func renvoEvalMetaParsedConstExprInto(m *renvoMeta, p *renvoProgram, ep *renvoExprParse, idx int, iotaValue int, out *renvoConstResult) {
	renvoNonNil(m, p, ep, out)
	if idx < 0 || idx >= len(ep.exprs) {
		renvoSetConstResult(out, 0, false)
		return
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		value := renvoParseConstIntToken(p, e.tok)
		if p.compilerInt32 && p.parsedIntHigh != value>>31 {
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprFloat {
		// Exact IEEE literals require more state than the metadata evaluator's
		// integer result can hold. They are parsed by the typed emitter.
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprChar {
		renvoSetConstResult(out, renvoParseCharToken(p, e.tok), true)
		return
	}
	if e.kind == renvoExprBool {
		renvoSetConstResult(out, renvoBoolTokenValue(p, e.tok), true)
		return
	}
	if e.kind == renvoExprIdent {
		if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "iota") {
			renvoSetConstResult(out, iotaValue, true)
			return
		}
		symIndex := renvoFindMetaGlobalIndex(m, e.nameStart, e.nameEnd, renvoTokConst)
		if symIndex >= 0 {
			s := &m.globals[symIndex]
			if s.constValueOK != 0 {
				renvoSetConstResult(out, s.constValue, true)
				return
			}
		}
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprCall {
		if e.argCount == 1 {
			result := renvoEvalMetaParsedConstExpr(m, p, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), iotaValue)
			if result.ok {
				callee := &ep.exprs[e.left]
				conversionType := renvoBuiltinTypeFromToken(p, callee.tok)
				if conversionType != 0 {
					conversion := renvoResolveType(m, conversionType)
					renvoNonNil(conversion)
					result.value = renvoConvertConstInt(m.c.renvoNativeIntSize, result.value, conversion.kind)
				}
			}
			*out = result
			return
		}
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprUnary {
		inner := renvoEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !inner.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		value := inner.value
		if renvoTokCharIs(p, e.tok, '-') {
			value = -value
		} else if renvoTokCharIs(p, e.tok, '+') {
		} else if renvoTokCharIs(p, e.tok, '^') {
			value = ^value
			var g renvoLinearGen
			g.c = m.c
			g.meta = m
			g.prog = p
			typ := renvoResolveType(m, renvoInferParsedExprType(&g, ep, e.left))
			value = renvoConvertConstInt(m.c.renvoNativeIntSize, value, typ.kind)
		} else if renvoTokCharIs(p, e.tok, '!') {
			value = 0
			if inner.value == 0 {
				value = 1
			}
		} else {
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprBinary {
		left := renvoEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !left.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		right := renvoEvalMetaParsedConstExpr(m, p, ep, e.right, iotaValue)
		if !right.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		var g renvoLinearGen
		g.c = m.c
		g.meta = m
		g.prog = p
		unsignedKind := 0
		if renvoExprHasUnsignedIntType(&g, ep, e.left) {
			unsignedKind = renvoResolveType(m, renvoInferParsedExprType(&g, ep, e.left)).kind
		}
		if !renvoTok2Is(p, e.tok, '<', '<') && !renvoTok2Is(p, e.tok, '>', '>') && renvoExprHasUnsignedIntType(&g, ep, e.right) {
			unsignedKind = renvoResolveType(m, renvoInferParsedExprType(&g, ep, e.right)).kind
		}
		renvoEvalConstBinaryInto(&g, e.tok, left.value, right.value, unsignedKind, out)
		return
	}
	renvoSetConstResult(out, 0, false)
}

func renvoFindConstSpecEqual(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		c := renvoTokSingleChar(p, i)
		if c == '(' {
			paren++
		} else if c == ')' {
			if paren > 0 {
				paren--
			}
		} else if c == '[' {
			brack++
		} else if c == ']' {
			if brack > 0 {
				brack--
			}
		} else if c == '{' {
			brace++
		} else if c == '}' {
			if brace > 0 {
				brace--
			}
		} else if paren == 0 && brack == 0 && brace == 0 && c == '=' {
			return i
		}
		i++
	}
	return start
}

func renvoParseTopDeclEntry(m *renvoMeta, p *renvoProgram, kind int, start int, end int) {
	renvoParseScopedDeclEntry(nil, m, p, kind, start, end)
}

func renvoParseScopedDeclEntry(g *renvoLinearGen, m *renvoMeta, p *renvoProgram, kind int, start int, end int) {
	renvoNonNil(m, p)
	if start >= end || !renvoTokIsKind(p, start, renvoTokIdent) {
		renvoMetaError(m)
		return
	}
	name := renvoTokAt(p, start)
	if kind == renvoTokVar {
		renvoParseVarDeclEntry(m, p, start, end)
		return
	}
	if kind == renvoTokType {
		typeStart := start + 1
		isAlias := renvoTokCharIs(p, typeStart, '=')
		if isAlias {
			typeStart++
		}
		typeResult := renvoParseScopedType(g, m, p, typeStart, end)
		if typeResult.typ == 0 || typeResult.next > end {
			renvoMetaError(m)
			return
		}
		directNamedType := !isAlias && (renvoTokIsKind(p, typeStart, renvoTokStruct) || renvoTokCharIs(p, typeStart, '*') || renvoTokCharIs(p, typeStart, '['))
		if directNamedType && (m.types[typeResult.typ].kind == renvoTypeStruct || m.types[typeResult.typ].kind == renvoTypePointer || m.types[typeResult.typ].kind == renvoTypeSlice) {
			m.typeIndexVersion++
			m.types[typeResult.typ].nameStart = int(name.start)
			m.types[typeResult.typ].nameEnd = int(name.end)
			renvoIndexNamedType(m, typeResult.typ)
		} else {
			size := renvoTypeSize(m, typeResult.typ)
			namedType := renvoAddType(m, renvoTypeNamed, typeResult.typ, 0, 0, size, int(name.start), int(name.end))
			if isAlias {
				m.types[namedType].first = renvoNamedTypeAlias
			}
		}
		return
	}
	eq := start
	j := start + 1
	for j < end {
		if j >= 0 && j < renvoTokCount(p) {
			tok := renvoTokAt(p, j)
			if renvoTokKind(p, j) == renvoTokOp && tok.end-tok.start == 1 && renvo_runtime_UnsafeByteAt(p.src, tok.start) == '=' {
				eq = j
				j = end
				continue
			}
		}
		j++
	}
	typeEnd := end
	initStart := end
	initEnd := end
	if eq > start {
		typeEnd = eq
		initStart = eq + 1
		initEnd = end
	}
	typ := 0
	if start+1 < typeEnd {
		typeResult := renvoParseType(m, p, start+1, typeEnd)
		typ = typeResult.typ
	}
	if typ == 0 && initStart < initEnd {
		typ = renvoInferTopLiteralType(m, p, initStart, initEnd)
	}
	objectDecl := 0
	if renvoFixedTarget == 0 {
		objectDecl = renvoParseObjectDirective(m, p, int(name.start))
	}
	renvoMetaAppendGlobal(m, renvoSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: kind, typ: typ, initStart: initStart, initEnd: initEnd, objectDecl: objectDecl})
}

func renvoParseVarDeclEntry(m *renvoMeta, p *renvoProgram, start int, end int) {
	renvoNonNil(m, p)
	eq := renvoFindConstSpecEqual(p, start, end)
	headEnd := end
	if eq > start {
		headEnd = eq
	}
	names := renvoFixedIntScratch(4)
	k := start
	for k < headEnd {
		if !renvoTokIsKind(p, k, renvoTokIdent) {
			break
		}
		names = append(names, k)
		k++
		if renvoTokCharIs(p, k, ',') {
			k++
			continue
		}
		break
	}
	if len(names) == 0 {
		renvoMetaError(m)
		return
	}
	typ := 0
	if k < headEnd {
		typeResult := renvoParseType(m, p, k, headEnd)
		typ = typeResult.typ
	}
	var values []int
	if eq > start {
		valueBuf, ok := renvoSplitTopLevelComma(p, eq+1, end)
		if !ok {
			renvoMetaError(m)
			return
		}
		values = valueBuf
	}
	valueCount := len(values) / 2
	if valueCount != 0 && valueCount != len(names) {
		renvoMetaError(m)
		return
	}
	for i := 0; i < len(names); i++ {
		nameTok := names[i]
		name := renvoTokAt(p, nameTok)
		if renvoBytesEqualText(p.src, int(name.start), int(name.end), "_") {
			continue
		}
		initStart := end
		initEnd := end
		symType := typ
		if valueCount != 0 {
			initStart = values[i*2]
			initEnd = values[i*2+1]
			if symType == 0 {
				symType = renvoInferTopLiteralType(m, p, initStart, initEnd)
			}
		}
		objectDecl := 0
		if renvoFixedTarget == 0 {
			objectDecl = renvoParseObjectDirective(m, p, int(name.start))
		}
		renvoMetaAppendGlobal(m, renvoSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: renvoTokVar, typ: symType, initStart: initStart, initEnd: initEnd, objectDecl: objectDecl})
	}
}

func renvoInferTopLiteralType(m *renvoMeta, p *renvoProgram, start int, end int) int {
	renvoNonNil(m, p)
	if renvoTokIdentIs(p, start, "make") {
		return renvoParseType(m, p, start+2, end).typ
	}
	if start+1 == end && renvoTokIsKind(p, start, renvoTokString) {
		return renvoTypeString
	}
	if start+1 == end && renvoTokIsKind(p, start, renvoTokFloat) {
		return renvoTypeFloat64
	}
	typ := 0
	for tok := start; tok < end; tok++ {
		if renvoTokCharIs(p, tok, '{') && tok > start {
			typ := renvoParseType(m, p, start, tok).typ
			resolved := renvoResolveType(m, typ)
			renvoNonNil(resolved)
			if resolved.kind == renvoTypeArray && resolved.count < 0 {
				ep := renvoNewExprParse()
				renvoNonNil(ep)
				rootIndex := renvoParseExpressionRoot(ep, p, start, end)
				if rootIndex < 0 || !renvoResolveInferredArrayCompositeLength(m, nil, ep, rootIndex, typ) {
					return 0
				}
			}
			return typ
		}
		if renvoTokIsKind(p, tok, renvoTokFloat) {
			typ = renvoTypeFloat64
		}
	}
	return typ
}

func renvoParseFuncInfo(m *renvoMeta, fnIndex int) {
	renvoNonNil(m)
	p := m.prog
	fn := p.funcs[fnIndex]
	nameStart := fn.nameStart
	nameEnd := fn.nameEnd
	nameTok := fn.nameTok
	if nameTok <= fn.startTok {
		renvoMetaError(m)
		return
	}
	lparen := renvoFindNextTokenText(p, nameTok+1, fn.bodyStart, '(')
	if lparen <= nameTok {
		renvoMetaError(m)
		return
	}
	rparen := renvoFindMatchingExprClose(p, lparen+1, fn.bodyStart, '(', ')')
	if rparen <= lparen {
		renvoMetaError(m)
		return
	}
	firstParam := len(m.params)
	paramCount := 0
	receiverType := 0
	if fn.receiverStart < fn.receiverEnd {
		beforeReceiver := len(m.params)
		if renvoTokIsKind(p, fn.receiverStart, renvoTokIdent) && fn.receiverStart+1 < fn.receiverEnd {
			renvoParseParamList(m, p, fn.receiverStart, fn.receiverEnd, &paramCount)
		} else {
			unnamedReceiver := renvoParseType(m, p, fn.receiverStart, fn.receiverEnd)
			if unnamedReceiver.typ == 0 || unnamedReceiver.next != fn.receiverEnd {
				renvoMetaError(m)
				return
			}
			m.params = append(m.params, renvoSymbolInfo{typ: unnamedReceiver.typ})
			paramCount++
		}
		if len(m.params) <= beforeReceiver {
			renvoMetaError(m)
			return
		}
		receiverType = m.params[beforeReceiver].typ
	}
	renvoParseParamList(m, p, lparen+1, rparen, &paramCount)
	resultType := 0
	firstResult := len(m.params)
	resultCount := 0
	if rparen+1 < fn.bodyStart {
		resultType, resultCount = renvoParseFuncResults(m, p, rparen+1, fn.bodyStart)
	}
	linkStatic := renvoParseLinkStaticDirective(p, fn.nameStart)
	if renvoFixedTarget != 0 {
		m.funcs = append(m.funcs, renvoFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, firstResult: firstResult, resultCount: resultCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd, linkStatic: linkStatic.ok, linkDLLStart: linkStatic.dllStart, linkDLLEnd: linkStatic.dllEnd, linkMethodStart: linkStatic.methodStart, linkMethodEnd: linkStatic.methodEnd})
	} else {
		export := renvoParseExportDirective(p, fn.nameStart)
		objectDecl := renvoParseObjectDirective(m, p, fn.nameStart)
		m.funcs = append(m.funcs, renvoFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, firstResult: firstResult, resultCount: resultCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd, linkStatic: linkStatic.ok, linkDLLStart: linkStatic.dllStart, linkDLLEnd: linkStatic.dllEnd, linkMethodStart: linkStatic.methodStart, linkMethodEnd: linkStatic.methodEnd, exportNameStart: export.nameStart, exportNameEnd: export.nameEnd, objectDecl: objectDecl})
	}
	if receiverType != 0 && renvoResolveType(m, receiverType).kind != renvoTypePointer {
		renvoAddPointerType(m, receiverType, renvoPointerSpaceData)
	}
}

func renvoTokenRangeHasIdent(p *renvoProgram, start int, end int, name string) bool {
	renvoNonNil(p)
	for tok := start; tok < end; tok++ {
		if renvoTokIdentIs(p, tok, name) {
			return true
		}
	}
	return false
}

func renvoParseFuncLiterals(m *renvoMeta, p *renvoProgram) {
	renvoNonNil(m, p)
	tokenCount := renvoTokCount(p)
	for tok := 0; tok < tokenCount; tok++ {
		if int(p.toks.data[tok*renvoTokenStride])&255 != renvoTokFunc || !renvoTokCharIs(p, tok+1, '(') {
			continue
		}
		declarationOrSignature := false
		for i := 0; i < len(p.funcs); i++ {
			if p.funcs[i].startTok == tok || tok > p.funcs[i].startTok && tok < p.funcs[i].bodyStart {
				declarationOrSignature = true
				break
			}
		}
		if declarationOrSignature {
			continue
		}
		bodyOpen := renvoFuncLiteralBodyOpen(p, tok, tokenCount)
		if bodyOpen < 0 {
			continue
		}
		paramsClose := renvoFindMatchingExprClose(p, tok+2, bodyOpen, '(', ')')
		if paramsClose <= tok+1 {
			continue
		}
		resultStart := paramsClose + 1
		if resultStart < bodyOpen {
			if renvoTokCharIs(p, resultStart, '(') {
				resultClose := renvoFindMatchingExprClose(p, resultStart+1, bodyOpen, '(', ')')
				if resultClose+1 != bodyOpen {
					continue
				}
			} else {
				result := renvoParseType(m, p, resultStart, bodyOpen)
				if result.typ == 0 || result.next != bodyOpen {
					continue
				}
			}
		}
		bodyEnd := renvoFindMatchingBrace(p, bodyOpen, tokenCount)
		if bodyEnd <= bodyOpen {
			continue
		}
		firstParam := len(m.params)
		m.params = append(m.params, renvoSymbolInfo{typ: renvoTypeInt})
		literalParamCount := renvoParseParamsInto(m, p, tok+2, paramsClose)
		if literalParamCount < 0 {
			renvoTruncParams(&m.params, firstParam)
			continue
		}
		firstResult := len(m.params)
		resultType := 0
		resultCount := 0
		if resultStart < bodyOpen {
			resultType, resultCount = renvoParseFuncResults(m, p, resultStart, bodyOpen)
			if resultType == 0 {
				renvoTruncParams(&m.params, firstParam)
				continue
			}
		}
		declIndex := len(p.funcs)
		p.funcs = append(p.funcs, renvoFuncDecl{startTok: tok, nameTok: tok, bodyStart: bodyOpen, bodyEnd: bodyEnd, endTok: bodyEnd + 1})
		fnIndex := len(m.funcs)
		m.funcs = append(m.funcs, renvoFuncInfo{declIndex: declIndex, firstParam: firstParam, paramCount: literalParamCount + 1, firstResult: firstResult, resultCount: resultCount, resultType: resultType, bodyStart: bodyOpen + 1, bodyEnd: bodyEnd, literalTok: tok})
		m.closures = append(m.closures, renvoClosureInfo{fnIndex: fnIndex})
		tok = bodyOpen
	}
}

func renvoParseParamsInto(m *renvoMeta, p *renvoProgram, start int, end int) int {
	renvoNonNil(m, p)
	base := len(m.params)
	if start == end {
		return 0
	}
	parts, ok := renvoSplitTopLevelComma(p, start, end)
	if !ok {
		return -1
	}
	named := false
	for i := 0; i < len(parts); i += 2 {
		partStart := parts[i]
		partEnd := parts[i+1]
		if renvoTokIsKind(p, partStart, renvoTokIdent) && partStart+1 < partEnd {
			typ := renvoParseType(m, p, partStart+1, partEnd)
			if typ.typ != 0 && typ.next == partEnd {
				named = true
				break
			}
		}
	}
	if named {
		group := 0
		for i := 0; i < len(parts); i += 2 {
			partStart := parts[i]
			partEnd := parts[i+1]
			if !renvoTokIsKind(p, partStart, renvoTokIdent) {
				return -1
			}
			typ := renvoParseType(m, p, partStart+1, partEnd)
			if typ.typ == 0 || typ.next != partEnd {
				continue
			}
			variadic := 0
			if renvoTokCharIs(p, partStart+1, '.') {
				variadic = 1
			}
			for j := group; j <= i; j += 2 {
				nameStart := renvoTokStart(p, parts[j])
				nameEnd := renvoTokEnd(p, parts[j])
				m.params = append(m.params, renvoSymbolInfo{nameStart: nameStart, nameEnd: nameEnd, typ: typ.typ, initStart: variadic})
			}
			group = i + 2
		}
		if group != len(parts) {
			return -1
		}
		return len(m.params) - base
	}
	for i := 0; i < len(parts); i += 2 {
		partStart := parts[i]
		partEnd := parts[i+1]
		typ := renvoParseType(m, p, partStart, partEnd)
		if typ.typ == 0 || typ.next != partEnd {
			return -1
		}
		variadic := 0
		if renvoTokCharIs(p, partStart, '.') {
			variadic = 1
		}
		m.params = append(m.params, renvoSymbolInfo{typ: typ.typ, initStart: variadic})
	}
	return len(m.params) - base
}

type renvoLinkStaticDirective struct {
	ok          int
	dllStart    int
	dllEnd      int
	methodStart int
	methodEnd   int
}

type renvoExportDirective struct {
	nameStart int
	nameEnd   int
}

func renvoPreviousDirectiveBody(src []byte, pos int, prefix string) (int, int) {
	if pos < 0 || pos > len(src) {
		return 0, 0
	}
	lineStart := pos
	for lineStart > 0 && renvo_runtime_UnsafeByteAt(src, lineStart-1) != '\n' {
		lineStart--
	}
	end := lineStart
	for end > 0 {
		ch := renvo_runtime_UnsafeByteAt(src, end-1)
		if ch > ' ' {
			break
		}
		end--
	}
	start := end
	for start > 0 && renvo_runtime_UnsafeByteAt(src, start-1) != '\n' {
		start--
	}
	for start < end && renvo_runtime_UnsafeByteAt(src, start) <= ' ' {
		start++
	}
	if end-start < len(prefix) ||
		!renvoBytesEqualText(src, start, start+len(prefix), prefix) {
		return 0, 0
	}
	bodyStart := start + len(prefix)
	for bodyStart < end && renvo_runtime_UnsafeByteAt(src, bodyStart) <= ' ' {
		bodyStart++
	}
	for end > bodyStart && renvo_runtime_UnsafeByteAt(src, end-1) <= ' ' {
		end--
	}
	return bodyStart, end
}

func renvoParseExportDirective(p *renvoProgram, pos int) renvoExportDirective {
	renvoNonNil(p)
	var result renvoExportDirective
	nameStart, nameEnd := renvoPreviousDirectiveBody(p.src, pos, "//export ")
	if nameEnd <= nameStart {
		return result
	}
	result.nameStart = nameStart
	result.nameEnd = nameEnd
	return result
}

func renvoParseLinkStaticDirective(p *renvoProgram, pos int) renvoLinkStaticDirective {
	renvoNonNil(p)
	var d renvoLinkStaticDirective
	src := p.src
	bodyStart, end := renvoPreviousDirectiveBody(
		src, pos, "// renvo:linkstatic ")
	comma := bodyStart
	for comma < end && renvo_runtime_UnsafeByteAt(src, comma) != ',' {
		comma++
	}
	if comma <= bodyStart || comma >= end {
		return d
	}
	dllEnd := comma
	for dllEnd > bodyStart && renvo_runtime_UnsafeByteAt(src, dllEnd-1) <= ' ' {
		dllEnd--
	}
	methodStart := comma + 1
	for methodStart < end && renvo_runtime_UnsafeByteAt(src, methodStart) <= ' ' {
		methodStart++
	}
	methodEnd := methodStart
	for methodEnd < end && renvo_runtime_UnsafeByteAt(src, methodEnd) != ',' {
		methodEnd++
	}
	for methodEnd > methodStart && renvo_runtime_UnsafeByteAt(src, methodEnd-1) <= ' ' {
		methodEnd--
	}
	if dllEnd <= bodyStart || methodEnd <= methodStart {
		return d
	}
	d.ok = 1
	d.dllStart = bodyStart
	d.dllEnd = dllEnd
	d.methodStart = methodStart
	d.methodEnd = methodEnd
	return d
}

const (
	renvoObjectDeclFunction       = 1
	renvoObjectDeclVariable       = 2
	renvoObjectDeclFunctionAlias  = 3
	renvoObjectDeclVariableAlias  = 4
	renvoObjectDeclVariableExtern = 5
	renvoObjectDeclStaticCall     = 6
)

func renvoParseObjectDirective(m *renvoMeta, p *renvoProgram, pos int) int {
	renvoNonNil(m, p)
	end := pos
	for line := 0; line < 4 && end > 0; line++ {
		for end > 0 {
			ch := renvo_runtime_UnsafeByteAt(p.src, end-1)
			if ch != ' ' && ch != '\t' && ch != '\r' && ch != '\n' {
				break
			}
			end--
		}
		start := end
		for start > 0 && renvo_runtime_UnsafeByteAt(p.src, start-1) != '\n' {
			start--
		}
		at := start
		for at < end && (renvo_runtime_UnsafeByteAt(p.src, at) == ' ' || renvo_runtime_UnsafeByteAt(p.src, at) == '\t') {
			at++
		}
		prefix := "// renvo:object "
		if end-at >= len(prefix) && renvoBytesEqualText(p.src, at, at+len(prefix), prefix) {
			fields := renvoObjectDirectiveFields(p.src, at+len(prefix), end)
			if len(fields) != 22 {
				return 0
			}
			kind := 0
			if renvoBytesEqualText(p.src, fields[0], fields[1], "function") {
				kind = renvoObjectDeclFunction
			} else if renvoBytesEqualText(p.src, fields[0], fields[1], "variable") {
				kind = renvoObjectDeclVariable
			} else if renvoBytesEqualText(p.src, fields[0], fields[1], "function-alias") {
				kind = renvoObjectDeclFunctionAlias
			} else if renvoBytesEqualText(p.src, fields[0], fields[1], "variable-alias") {
				kind = renvoObjectDeclVariableAlias
			} else if renvoBytesEqualText(p.src, fields[0], fields[1], "variable-extern") {
				kind = renvoObjectDeclVariableExtern
			} else if renvoBytesEqualText(p.src, fields[0], fields[1], "static-call") {
				kind = renvoObjectDeclStaticCall
			}
			alignment, alignOK := renvoObjectDirectiveDecimal(p.src, fields[6], fields[7])
			binding, bindOK := renvoObjectDirectiveDecimal(p.src, fields[8], fields[9])
			visibility, visibilityOK := renvoObjectDirectiveDecimal(p.src, fields[10], fields[11])
			size, sizeOK := renvoObjectDirectiveDecimal(p.src, fields[14], fields[15])
			relocationKind, relocationOK := renvoObjectDirectiveDecimal(p.src, fields[16], fields[17])
			relocationAddend, addendOK := renvoObjectDirectiveDecimal(p.src, fields[20], fields[21])
			if kind == 0 || !alignOK || !bindOK || !visibilityOK || !sizeOK || !relocationOK || !addendOK ||
				binding > 2 || visibility > 3 || relocationKind != 0 && relocationKind != 1 && relocationKind != 2 && relocationKind != 10 && relocationKind != 11 {
				return 0
			}
			decl := renvoObjectDecl{kind: kind, nameStart: fields[2], nameEnd: fields[3],
				sectionStart: fields[4], sectionEnd: fields[5], alignment: alignment,
				binding: binding, visibility: visibility, targetStart: fields[12], targetEnd: fields[13], size: size,
				relocationKind: relocationKind, relocationTargetStart: fields[18], relocationTargetEnd: fields[19], relocationAddend: relocationAddend}
			m.objectDecls = append(m.objectDecls, decl)
			return len(m.objectDecls) - 1
		}
		if line > 0 && (end-at < 2 || renvo_runtime_UnsafeByteAt(p.src, at) != '/' || renvo_runtime_UnsafeByteAt(p.src, at+1) != '/') {
			return 0
		}
		end = start
	}
	return 0
}

// Each field is stored as a start/end pair, avoiding short-lived strings in
// the compiler's hottest metadata phase.
func renvoObjectDirectiveFields(src []byte, start int, end int) []int {
	fields := make([]int, 0, 22)
	for start < end {
		for start < end && (renvo_runtime_UnsafeByteAt(src, start) == ' ' || renvo_runtime_UnsafeByteAt(src, start) == '\t') {
			start++
		}
		if start >= end {
			break
		}
		fieldEnd := start
		for fieldEnd < end && renvo_runtime_UnsafeByteAt(src, fieldEnd) != ' ' && renvo_runtime_UnsafeByteAt(src, fieldEnd) != '\t' {
			fieldEnd++
		}
		fields = append(fields, start, fieldEnd)
		start = fieldEnd
	}
	return fields
}

func renvoObjectDirectiveDecimal(src []byte, start int, end int) (int, bool) {
	if start >= end {
		return 0, false
	}
	value := 0
	for start < end {
		ch := renvo_runtime_UnsafeByteAt(src, start)
		if ch < '0' || ch > '9' || value > 1<<28 {
			return 0, false
		}
		value = value*10 + int(ch-'0')
		start++
	}
	return value, true
}

func renvoParseFuncResults(m *renvoMeta, p *renvoProgram, start int, end int) (int, int) {
	renvoNonNil(m, p)
	if renvoTokCharIs(p, start, '(') {
		closeTok := renvoFindMatchingExprClose(p, start+1, end, '(', ')')
		if closeTok > start && closeTok <= end {
			parts, ok := renvoSplitTopLevelComma(p, start+1, closeTok)
			if !ok {
				return 0, 0
			}
			count := len(parts) / 2
			allUnnamed := count > 0
			typeCount := len(m.types)
			typeIndexVersion := m.typeIndexVersion
			fieldCount := len(m.fields)
			parsedTypes := make([]int, count)
			for i := 0; i < count; i++ {
				partStart := parts[i*2]
				partEnd := parts[i*2+1]
				result := renvoParseType(m, p, partStart, partEnd)
				parsedTypes[i] = result.typ
				if result.typ == 0 || result.next != partEnd {
					allUnnamed = false
				}
			}
			if allUnnamed {
				if count > 1 {
					return renvoBuildTupleType(m, parsedTypes), 0
				}
				return parsedTypes[0], 0
			}
			renvoTruncTypes(&m.types, typeCount)
			renvoTruncFields(&m.fields, fieldCount)
			if m.typeIndexVersion != typeIndexVersion {
				renvoRebuildNamedTypeIndex(m)
			}
			firstResult := len(m.params)
			resultCount := 0
			renvoParseParamList(m, p, start+1, closeTok, &resultCount)
			if resultCount == 0 || len(m.params) != firstResult+resultCount {
				return 0, 0
			}
			if resultCount == 1 {
				return m.params[firstResult].typ, 1
			}
			return renvoBuildTupleTypeFromParams(m, firstResult, resultCount), resultCount
		}
	}
	typeResult := renvoParseType(m, p, start, end)
	return typeResult.typ, 0
}

func renvoBuildTupleTypeFromParams(m *renvoMeta, first int, count int) int {
	renvoNonNil(m)
	types := renvoFixedIntScratch(count)
	for i := 0; i < count; i++ {
		types = append(types, m.params[first+i].typ)
	}
	return renvoBuildTupleType(m, types)
}

func renvoBuildTupleType(m *renvoMeta, types []int) int {
	renvoNonNil(m)
	// Interface method checks reparse signatures for each candidate type.
	// Reuse their result tuples so reparsing does not create fresh function
	// types and enlarge the candidate set at each subsequent assertion.
	for candidate := len(m.types) - 1; candidate >= 0; candidate-- {
		t := &m.types[candidate]
		if t.kind != renvoTypeStruct || t.count != len(types) || t.nameEnd > t.nameStart {
			continue
		}
		match := true
		for i := 0; i < len(types); i++ {
			field := m.fields[t.first+i]
			if field.nameEnd > field.nameStart || field.typ != types[i] {
				match = false
				break
			}
		}
		if match {
			return candidate
		}
	}
	firstField := len(m.fields)
	offset := 0
	for i := 0; i < len(types); i++ {
		typ := types[i]
		offset = renvoAlignTo8(offset)
		m.fields = append(m.fields, renvoFieldInfo{typ: typ, offset: offset})
		offset += renvoTypeCopySize(m, typ)
	}
	size := renvoAlignTo8(offset)
	return renvoAddType(m, renvoTypeStruct, 0, firstField, len(types), size, 0, 0)
}

func renvoParseParamList(m *renvoMeta, p *renvoProgram, start int, end int, count *int) {
	renvoNonNil(m, p, count)
	parsed := renvoParseParamsInto(m, p, start, end)
	if parsed < 0 {
		renvoMetaError(m)
		return
	}
	*count = *count + parsed
}

func renvoParseType(m *renvoMeta, p *renvoProgram, start int, end int) renvoTypeResult {
	return renvoParseScopedType(nil, m, p, start, end)
}

// Function-body type expressions resolve lengths through the active lexical
// bindings. Package declarations use the same parser without a local context.
func renvoParseScopedType(g *renvoLinearGen, m *renvoMeta, p *renvoProgram, start int, end int) renvoTypeResult {
	renvoNonNil(m, p)
	var result renvoTypeResult
	renvoParseTypeInto(g, m, p, start, end, &result)
	return result
}

func renvoSetTypeResult(result *renvoTypeResult, typ int, next int) {
	renvoNonNil(result)
	result.typ = typ
	result.next = next
}

func renvoParseFuncSignatureInto(m *renvoMeta, p *renvoProgram, openTok int, end int, result *renvoTypeResult) {
	renvoNonNil(m, p, result)
	renvoSetTypeResult(result, 0, openTok)
	closeTok := renvoFindMatchingExprClose(p, openTok+1, end, '(', ')')
	if closeTok <= openTok {
		return
	}
	paramBase := len(m.params)
	paramCount := renvoParseParamsInto(m, p, openTok+1, closeTok)
	if paramCount < 0 {
		renvoTruncParams(&m.params, paramBase)
		return
	}
	resultType := 0
	next := closeTok + 1
	if next < end {
		resultType, _ = renvoParseFuncResults(m, p, next, end)
		if resultType == 0 {
			renvoTruncParams(&m.params, paramBase)
			return
		}
		next = end
	}
	typ := renvoFindOrAddFuncTypeFromParams(m, paramBase, paramCount, resultType)
	renvoTruncParams(&m.params, paramBase)
	renvoSetTypeResult(result, typ, next)
}

func renvoParseTypeInto(g *renvoLinearGen, m *renvoMeta, p *renvoProgram, start int, end int, result *renvoTypeResult) {
	renvoNonNil(m, p, result)
	if start >= end {
		renvoSetTypeResult(result, 0, start)
		return
	}
	if renvoTokIsKind(p, start, renvoTokFunc) && renvoTokCharIs(p, start+1, '(') {
		renvoParseFuncSignatureInto(m, p, start+1, end, result)
		return
	}
	if renvoTokIdentIs(p, start, "interface") && renvoTokCharIs(p, start+1, '{') {
		closeTok := renvoFindMatchingBrace(p, start+1, end)
		if closeTok <= start+1 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		if closeTok == start+2 {
			renvoSetTypeResult(result, renvoBuiltinTypeInterface, closeTok+1)
			return
		}
		renvoSetTypeResult(result, renvoAddType(m, renvoTypeInterface, 0, start+2, closeTok, 2*renvoBackendValueSlotSize, 0, 0), closeTok+1)
		return
	}
	if renvoTokCharIs(p, start, '.') && renvoTokCharIs(p, start+1, '.') && renvoTokCharIs(p, start+2, '.') {
		elem := renvoParseScopedType(g, m, p, start+3, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		typ := renvoAddSequenceType(m, renvoTypeSlice, elem.typ, 0, renvoBackendSliceValueSize)
		renvoSetTypeResult(result, typ, elem.next)
		return
	}
	if renvoTokCharIs(p, start, '*') {
		elem := renvoParseScopedType(g, m, p, start+1, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		typ := renvoAddPointerType(m, elem.typ, renvoPointerSpaceData)
		renvoSetTypeResult(result, typ, elem.next)
		return
	}
	if renvoTokCharIs(p, start, '[') && !renvoTokCharIs(p, start+1, ']') {
		closeTok := renvoFindMatchingExprClose(p, start+1, end, '[', ']')
		if closeTok <= start+1 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		count := -1
		ellipsis := closeTok == start+4 && renvoTokCharIs(p, start+1, '.') && renvoTokCharIs(p, start+2, '.') && renvoTokCharIs(p, start+3, '.')
		if !ellipsis {
			var length renvoConstResult
			if g == nil {
				length = renvoEvalMetaConstExpr(m, p, start+1, closeTok, 0)
			} else {
				ep := renvoNewExprParse()
				root := renvoParseExpressionRoot(ep, p, start+1, closeTok)
				if root >= 0 {
					length = renvoEvalConstExpr(g, ep, root)
				}
			}
			if !length.ok || length.value < 0 {
				renvoSetTypeResult(result, 0, start)
				return
			}
			count = length.value
		}
		elem := renvoParseScopedType(g, m, p, closeTok+1, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		size := 0
		if count >= 0 {
			size = count * renvoTypeSize(m, elem.typ)
		}
		renvoSetTypeResult(result, renvoAddSequenceType(m, renvoTypeArray, elem.typ, count, size), elem.next)
		return
	}
	if renvoTokCharIs(p, start, '[') && renvoTokCharIs(p, start+1, ']') {
		elem := renvoParseScopedType(g, m, p, start+2, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		typ := renvoAddSequenceType(m, renvoTypeSlice, elem.typ, 0, renvoBackendSliceValueSize)
		renvoSetTypeResult(result, typ, elem.next)
		return
	}
	if renvoTokIsKind(p, start, renvoTokStruct) && renvoTokCharIs(p, start+1, '{') {
		closeTok := renvoFindMatchingBrace(p, start+1, end)
		if closeTok <= start+1 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		i := start + 2
		count := 0
		for i < closeTok {
			if renvoTokIsKind(p, i, renvoTokIdent) {
				count++
				for nameEnd := i + 1; renvoTokCharIs(p, nameEnd, ','); nameEnd += 2 {
					count++
				}
				i = renvoStatementLineEnd(p, i, closeTok)
			} else {
				i++
			}
		}
		firstField := len(m.fields)
		m.fields = append(m.fields, make([]renvoFieldInfo, count)...)
		fieldIndex := 0
		offset := 0
		hostLayoutMarker := false
		i = start + 2
		for i < closeTok {
			if renvoTokIsKind(p, i, renvoTokIdent) || renvoTokCharIs(p, i, '*') {
				lineEnd := renvoStatementLineEnd(p, i, closeTok)
				typeStart := i + 1
				for renvoTokCharIs(p, typeStart, ',') {
					typeStart += 2
				}
				nameTok := i
				namesEnd := typeStart
				if renvoTokCharIs(p, i, '*') {
					nameTok++
					namesEnd++
				}
				embedded := typeStart >= lineEnd || nameTok != i || renvoTokIsKind(p, typeStart, renvoTokString)
				if embedded {
					typeStart = i
				}
				fieldType := renvoParseScopedType(g, m, p, typeStart, lineEnd)
				if fieldType.typ == 0 {
					renvoSetTypeResult(result, 0, start)
					return
				}
				var fieldInfo renvoFieldInfo
				fieldInfo.typ = fieldType.typ
				fieldInfo.embedded = embedded
				if fieldType.next < lineEnd && !hostLayoutMarker {
					tag := renvoTokAt(p, fieldType.next)
					hostLayoutMarker = renvoBytesEqualText(p.src, int(tag.start), int(tag.end), "`r:\"h\"`")
				}
				for nameTok < namesEnd {
					fieldNameTok := renvoTokAt(p, nameTok)
					offset = renvoAlignTo8(offset)
					fieldInfo.nameStart = int(fieldNameTok.start)
					fieldInfo.nameEnd = int(fieldNameTok.end)
					fieldInfo.offset = offset
					m.fields[firstField+fieldIndex] = fieldInfo
					offset += renvoTypeSize(m, fieldType.typ)
					fieldIndex++
					nameTok += 2
				}
				i = lineEnd
			} else {
				i++
			}
		}
		size := renvoAlignTo8(offset)
		typ := renvoAddType(m, renvoTypeStruct, 0, firstField, count, size, 0, 0)
		if hostLayoutMarker {
			m.types[typ].resolved = renvoStructLayoutMarker
		}
		renvoSetTypeResult(result, typ, closeTok+1)
		return
	}
	if renvoTokIsKind(p, start, renvoTokIdent) {
		if renvoTokIdentIs(p, start, "any") {
			// any and interface{} have one identity, including when nested in
			// pointers and collections used in dynamic type assertions.
			renvoSetTypeResult(result, renvoBuiltinTypeInterface, start+1)
			return
		}
		if renvoTokIdentIs(p, start, "error") {
			renvoSetTypeResult(result, renvoBuiltinTypeError, start+1)
			return
		}
		builtin := renvoBuiltinTypeFromToken(p, start)
		if builtin != 0 {
			renvoSetTypeResult(result, builtin, start+1)
			return
		}
		tok := renvoTokAt(p, start)
		nameStart := int(tok.start)
		nameEnd := int(tok.end)
		typ := renvoFindNamedType(m, nameStart, nameEnd)
		if typ < 0 {
			typ = renvoAddType(m, renvoTypeNamed, 0, 0, 0, renvoBackendValueSlotSize, nameStart, nameEnd)
		}
		renvoSetTypeResult(result, typ, start+1)
		return
	}
	renvoSetTypeResult(result, 0, start)
}
func renvoFindOrAddFuncTypeFromParams(m *renvoMeta, first int, count int, resultType int) int {
	renvoNonNil(m)
	variadic := 0
	if count > 0 && m.params[first+count-1].initStart == 1 {
		variadic = 1
	}
	for i := len(m.types) - 1; i >= 0; i-- {
		t := &m.types[i]
		if t.kind != renvoTypeFunc || t.elem != resultType || t.count != count || t.resolved != variadic {
			continue
		}
		match := true
		for j := 0; j < count; j++ {
			if t.first+j >= len(m.fields) || m.fields[t.first+j].typ != m.params[first+j].typ {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	firstField := len(m.fields)
	for i := 0; i < count; i++ {
		m.fields = append(m.fields, renvoFieldInfo{typ: m.params[first+i].typ})
	}
	typ := renvoAddType(m, renvoTypeFunc, resultType, firstField, count, renvoBackendValueSlotSize, 0, 0)
	m.types[typ].resolved = variadic
	return typ
}

func renvoFunctionTypeFromInfo(m *renvoMeta, fnIndex int) int {
	renvoNonNil(m)
	if fnIndex < 0 || fnIndex >= len(m.funcs) {
		return 0
	}
	fn := &m.funcs[fnIndex]
	first := 0
	if fn.literalTok > 0 {
		first = 1
	} else if fn.receiverType != 0 {
		first = 1
	}
	return renvoFunctionTypeFromInfoStart(m, fnIndex, first)
}

func renvoFunctionTypeFromInfoStart(m *renvoMeta, fnIndex int, first int) int {
	renvoNonNil(m)
	if fnIndex < 0 || fnIndex >= len(m.funcs) {
		return 0
	}
	fn := &m.funcs[fnIndex]
	if first < 0 || first > fn.paramCount {
		return 0
	}
	return renvoFindOrAddFuncTypeFromParams(m, fn.firstParam+first, fn.paramCount-first, fn.resultType)
}

func renvoClosureIndexByToken(meta *renvoMeta, tok int) int {
	renvoNonNil(meta)
	for i := 0; i < len(meta.closures); i++ {
		fnIndex := meta.closures[i].fnIndex
		if fnIndex >= 0 && fnIndex < len(meta.funcs) && meta.funcs[fnIndex].literalTok == tok {
			return i
		}
	}
	return -1
}

func renvoBuiltinTypeFromToken(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	if renvoTokIdentIs(p, tokIndex, "error") {
		return renvoBuiltinTypeError
	}
	if renvoTokIdentIs(p, tokIndex, "any") {
		return renvoBuiltinTypeInterface
	}
	if renvoTokIdentIs(p, tokIndex, "uintptr") || renvoTokIdentIs(p, tokIndex, "uint") {
		if p.c.renvoNativeIntSize == 8 {
			return renvoBuiltinTypeUint64
		}
		if p.c.renvoNativeIntSize == 2 {
			return renvoBuiltinTypeUint16
		}
		return renvoBuiltinTypeUint32
	}
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "float32") {
		return renvoBuiltinTypeFloat32
	}
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "float64") {
		return renvoTypeFloat64
	}
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "complex64") {
		return renvoBuiltinTypeComplex64
	}
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "complex128") {
		return renvoBuiltinTypeComplex
	}
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "rune") {
		return renvoTypeInt32
	}
	entry := renvoIdentEntry(p.src, int(tok.start), int(tok.end))
	if entry == 0 {
		return 0
	}
	return int(renvoIdentTypeCodes[entry-1])
}

func renvoAddType(m *renvoMeta, kind int, elem int, first int, count int, size int, nameStart int, nameEnd int) int {
	renvoNonNil(m)
	m.types = append(m.types, renvoTypeInfo{kind: kind, elem: elem, first: first, count: count, size: size, nameStart: nameStart, nameEnd: nameEnd})
	index := len(m.types) - 1
	renvoIndexNamedType(m, index)
	return index
}

func renvoIndexNamedType(m *renvoMeta, index int) {
	renvoNonNil(m)
	if index >= len(m.types) {
		return
	}
	t := &m.types[index]
	if t.nameEnd <= t.nameStart || renvoFindNamedType(m, t.nameStart, t.nameEnd) >= 0 {
		return
	}
	buckets := m.typeBuckets
	bucket := renvoHashRange(m.prog.src, t.nameStart, t.nameEnd) % len(buckets)
	for probes := 0; probes < len(buckets); probes++ {
		if buckets[bucket] == 0 {
			buckets[bucket] = int32(index + 1)
			m.typeIndexVersion++
			return
		}
		bucket++
		if bucket == len(buckets) {
			bucket = 0
		}
	}
}

func renvoRebuildNamedTypeIndex(m *renvoMeta) {
	renvoNonNil(m)
	for i := 0; i < len(m.typeBuckets); i++ {
		m.typeBuckets[i] = 0
	}
	for i := 0; i < len(m.types); i++ {
		renvoIndexNamedType(m, i)
	}
}

func renvoFindNamedType(m *renvoMeta, nameStart int, nameEnd int) int {
	renvoNonNil(m)
	buckets := m.typeBuckets
	bucket := renvoHashRange(m.prog.src, nameStart, nameEnd) % len(buckets)
	for probes := 0; probes < len(buckets); probes++ {
		entry := int(renvo_runtime_UnsafeInt32At(buckets, bucket))
		if entry == 0 {
			return -1
		}
		index := entry - 1
		t := &m.types[index]
		if renvoBytesEqualRange(m.prog.src, t.nameStart, t.nameEnd, nameStart, nameEnd) {
			return index
		}
		bucket++
		if bucket == len(buckets) {
			bucket = 0
		}
	}
	return -1
}

func renvoAddPointerType(m *renvoMeta, elem int, addressSpace int) int {
	renvoNonNil(m)
	if addressSpace < renvoPointerSpaceData || addressSpace > renvoPointerSpaceGeneric {
		addressSpace = renvoPointerSpaceData
	}
	// Pointer types are most often requested repeatedly while lowering nearby
	// expressions. Recently-created types therefore provide the best hit rate.
	for i := len(m.types) - 1; i >= 1; i-- {
		typ := &m.types[i]
		if typ.kind == renvoTypePointer && typ.elem == elem && typ.first == addressSpace {
			return i
		}
	}
	return renvoAddType(m, renvoTypePointer, elem, addressSpace, 0, renvoBackendValueSlotSize, 0, 0)
}

func renvoAddSequenceType(m *renvoMeta, kind int, elem int, count int, size int) int {
	renvoNonNil(m)
	for i := len(m.types) - 1; i >= 1; i-- {
		t := &m.types[i]
		if t.kind == kind && t.elem == elem && t.count == count && t.nameStart == 0 {
			return i
		}
	}
	return renvoAddType(m, kind, elem, 0, count, size, 0, 0)
}

func renvoPointerAddressSpace(m *renvoMeta, typ int) int {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	if t.kind != renvoTypePointer {
		return 0
	}
	if t.first < renvoPointerSpaceData || t.first > renvoPointerSpaceGeneric {
		return renvoPointerSpaceData
	}
	return t.first
}

func renvoFinalizeTypeLayouts(m *renvoMeta) {
	for i := 0; i < len(m.funcs); i++ {
		fn := &m.funcs[i]
		if fn.linkStatic == 0 {
			continue
		}
		for j := 0; j < fn.paramCount; j++ {
			paramType := m.params[fn.firstParam+j].typ
			resolved := renvoResolveType(m, paramType)
			renvoNativeTypeLayout(m, paramType)
			if resolved.kind == renvoTypePointer {
				renvoNativeTypeLayout(m, resolved.elem)
			}
		}
		renvoNativeTypeLayout(m, fn.resultType)
	}
	for i := 0; i < len(m.types); i++ {
		renvoFinalizeValueLayout(m, i)
	}
	for i := 0; i < len(m.types); i++ {
		renvoMarkDenseCallWords(m, i)
	}
}

// Array strides and containing field offsets depend on the completed layout
// of their value elements, including types declared later in the source.
// Pointer edges do not contribute to value size and must not be traversed.
func renvoFinalizeValueLayout(m *renvoMeta, typ int) {
	if typ <= 0 || typ >= len(m.types) {
		return
	}
	t := renvoResolveType(m, typ)
	if t.nativeAlign != 0 {
		return
	}
	// Negative alignment records a completed ordinary value layout; positive
	// values remain reserved for native layouts. Native layout may still
	// replace this marker when an enclosing ABI requires it.
	t.nativeAlign = -1
	if t.kind == renvoTypeArray {
		renvoFinalizeValueLayout(m, t.elem)
		t.size = renvoTypeSize(m, t.elem) * t.count
	} else if t.kind == renvoTypeStruct {
		for j := 0; j < t.count; j++ {
			renvoFinalizeValueLayout(m, m.fields[t.first+j].typ)
		}
		nativeLayout := m.c.objectFile || t.count > 0 && !renvoTypeIsTuple(m, typ)
		offset := 0
		for j := 0; j < t.count; j++ {
			field := &m.fields[t.first+j]
			fieldType := renvoResolveType(m, field.typ)
			if fieldType.resolved == renvoStructLayoutMarker {
				t.resolved = renvoStructLayoutHost
				nativeLayout = true
				break
			}
			if !m.c.objectFile && (fieldType.kind == renvoTypeInt || !renvoTypeKindIsScalarValue(fieldType.kind)) {
				nativeLayout = false
			}
			offset = renvoAlignTo8(offset)
			field.offset = offset
			offset += renvoTypeSize(m, field.typ)
		}
		if nativeLayout {
			renvoNativeTypeLayout(m, typ)
		} else {
			t.size = renvoAlignTo8(offset)
		}
	}
}

func renvoMarkDenseCallWords(m *renvoMeta, typ int) bool {
	if typ <= 0 || typ >= len(m.types) {
		return false
	}
	t := renvoResolveType(m, typ)
	if renvoTypeKindIsWideInt(t.kind) || t.kind == renvoTypeArray || t.kind == renvoTypeStruct && t.nativeAlign > 0 {
		if t.kind == renvoTypeStruct {
			t.resolved = renvoStructLayoutDense
		}
		return true
	}
	if t.kind == renvoTypeStruct {
		for i := 0; i < t.count; i++ {
			if renvoMarkDenseCallWords(m, m.fields[t.first+i].typ) {
				t.resolved = renvoStructLayoutDense
				return true
			}
		}
	}
	return false
}
func renvoNativeTypeLayout(m *renvoMeta, typ int) int {
	renvoNonNil(m)
	if typ <= 0 || typ >= len(m.types) {
		return renvoBackendValueSlotSize
	}
	t := &m.types[typ]
	if t.nativeAlign > 0 {
		return t.size
	}
	size := t.size
	if size < 1 {
		size = renvoBackendValueSlotSize
	}
	align := renvoNativeAlignment(m.c, size)
	// Mark the type before descending so recursive pointer types terminate.
	t.nativeAlign = align
	if t.kind == renvoTypeNamed {
		resolved := t.elem
		if resolved == 0 {
			resolved = renvoFindResolvedNamedTypeIndex(m, typ)
		}
		if resolved > 0 && resolved < len(m.types) {
			size = renvoNativeTypeLayout(m, resolved)
			align = m.types[resolved].nativeAlign
		}
	} else if t.kind == renvoTypeStruct {
		offset := 0
		align = 1
		for i := 0; i < t.count; i++ {
			fieldIndex := t.first + i
			if fieldIndex < 0 || fieldIndex >= len(m.fields) {
				continue
			}
			fieldType := m.fields[fieldIndex].typ
			fieldSize := renvoNativeTypeLayout(m, fieldType)
			fieldAlign := m.types[fieldType].nativeAlign
			if fieldAlign < 1 {
				fieldAlign = 1
			}
			if fieldAlign > align {
				align = fieldAlign
			}
			offset = renvoAlignValue(offset, fieldAlign)
			m.fields[fieldIndex].offset = offset
			offset += fieldSize
		}
		size = renvoAlignValue(offset, align)
	} else if t.kind == renvoTypeArray {
		size = renvoNativeTypeLayout(m, t.elem) * t.count
		align = m.types[t.elem].nativeAlign
	} else if t.kind == renvoTypePointer {
		// Pointers and C object function addresses occupy one native word. Their
		// layout is independent of their pointee/signature; descending here can
		// encounter an aggregate already being laid out through a cyclic C type
		// graph and freeze that aggregate at the provisional slot size.
		size = m.c.renvoNativeIntSize
		align = renvoNativeAlignment(m.c, size)
	} else if (renvoFixedTarget == 0 || renvoRTGPreparedObject != 0) && m.c.objectFile && t.kind == renvoTypeFunc {
		size = m.c.renvoNativeIntSize
		align = renvoNativeAlignment(m.c, size)
	}
	if align < 1 {
		align = 1
	}
	t.size = size
	t.nativeAlign = align
	return size
}

func renvoNativeAlignment(context *renvoCompileContext, size int) int {
	maxAlign := renvoTargetMaxAlignment(context)
	if size >= 8 && maxAlign >= 8 {
		return 8
	}
	if size >= 4 && maxAlign >= 4 {
		return 4
	}
	if size >= 2 && maxAlign >= 2 {
		return 2
	}
	return 1
}

func renvoLanguageTypeAlignment(meta *renvoMeta, typ int) int {
	t := renvoResolveType(meta, typ)
	if t.kind == renvoTypeComplex64 {
		return renvoNativeAlignment(meta.c, 4)
	}
	if t.kind == renvoTypeArray {
		return renvoLanguageTypeAlignment(meta, t.elem)
	}
	if t.kind == renvoTypeStruct {
		alignment := 1
		for i := 0; i < t.count; i++ {
			fieldAlignment := renvoLanguageTypeAlignment(meta, meta.fields[t.first+i].typ)
			if fieldAlignment > alignment {
				alignment = fieldAlignment
			}
		}
		return alignment
	}
	return renvoNativeAlignment(meta.c, renvoTypeSize(meta, typ))
}

func renvoFindResolvedNamedTypeIndex(m *renvoMeta, typ int) int {
	renvoNonNil(m)
	if typ < 0 || typ >= len(m.types) {
		return -1
	}
	t := &m.types[typ]
	if t.resolved > 0 && t.resolved < len(m.types) {
		return t.resolved
	}
	for i := 0; i < len(m.types); i++ {
		if i == typ {
			continue
		}
		other := m.types[i]
		if other.nameEnd <= other.nameStart {
			continue
		}
		if !renvoBytesEqualRange(m.prog.src, other.nameStart, other.nameEnd, t.nameStart, t.nameEnd) {
			continue
		}
		if other.kind != renvoTypeNamed || other.elem > 0 {
			t.resolved = i
			return i
		}
	}
	return -1
}

func renvoTypeSize(m *renvoMeta, typ int) int {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	if t.size > 0 {
		return t.size
	}
	return renvoBackendValueSlotSize
}

func renvoTypeCopySize(m *renvoMeta, typ int) int {
	renvoNonNil(m)
	if renvoResolveType(m, typ).kind == renvoTypeComplex64 {
		return 2 * renvoBackendValueSlotSize
	}
	size := renvoTypeSize(m, typ)
	if size < renvoBackendValueSlotSize {
		return renvoBackendValueSlotSize
	}
	return size
}

var renvoInvalidType renvoTypeInfo

func renvoResolveType(m *renvoMeta, typ int) *renvoTypeInfo {
	renvoNonNil(m)
	for uint(typ) < uint(len(m.types)) {
		t := &m.types[typ]
		if t.kind != renvoTypeNamed {
			return t
		}
		if t.elem > 0 && t.elem < len(m.types) {
			typ = t.elem
			continue
		}
		if t.elem == 0 && t.nameEnd > t.nameStart {
			resolved := renvoFindResolvedNamedTypeIndex(m, typ)
			if resolved >= 0 {
				typ = resolved
				continue
			}
		}
		return t
	}
	return &renvoInvalidType
}

func renvoTypeIsSlice(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeSlice
}

func renvoAsmLoadSliceMemSecondary(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 16)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopSecondary(a)
	renvoAsmPopPrimary(a)
}

func renvoTypeIsStringSlice(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	return renvoTypeIsString(m, t.elem)
}

func renvoTypeIsString(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeString
}

func renvoTypeIsInt(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeInt ||
		m.prog != nil && renvoProgramUsesC11Semantics(m.prog) && t.kind == renvoTypeInt32
}

func renvoTypeIsNativeInt(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeInt
}

func renvoTypeKindIsScalarInt(kind int) bool {
	if kind > renvoTypeInvalid && kind < renvoTypeString {
		return true
	}
	return kind == renvoTypeInt8 || kind == renvoTypeInt16 || kind == renvoTypeInt32 || kind == renvoTypeUint16 || kind == renvoTypeUint32 || kind == renvoTypeUint64
}

func renvoTypeKindIsFloat(kind int) bool {
	return kind == renvoTypeFloat32 || kind == renvoTypeFloat64
}

func renvoTypeKindIsComplex(kind int) bool {
	return kind == renvoTypeComplex64 || kind == renvoTypeComplex
}

func renvoTypeKindIsWideInt(kind int) bool {
	return kind == renvoTypeInt64 || kind == renvoTypeUint64
}

func renvoTypeKindIsUnsignedInteger(kind int) bool {
	return kind == renvoTypeByte || kind == renvoTypeUint16 || kind == renvoTypeUint32 || kind == renvoTypeUint64
}

func renvoTypeKindIsWideValue(kind int) bool {
	// Prepared RTG targets without an IEEE float implementation retain the
	// legacy one-word scaled representation. Treating their float64 values as
	// two-word IEEE values routes assignments, composites, calls, and tuple
	// results through emitters those definitions do not provide.
	if renvoTypeKindIsWideInt(kind) {
		return true
	}
	if kind != renvoTypeFloat64 {
		return false
	}
	if renvoPreparedBackendActive == 0 {
		return true
	}
	return renvoRTGPreparedIEEEFloat != 0
}

func renvoTypeKindIsScalarValue(kind int) bool {
	return renvoTypeKindIsScalarInt(kind) || renvoTypeKindIsFloat(kind)
}

func renvoTypeKindUsesMemory(kind int) bool {
	return kind == renvoTypeString || (kind >= renvoTypeSlice && kind <= renvoTypeStruct) || kind == renvoTypeArray || kind == renvoTypeInterface || renvoTypeKindIsComplex(kind)
}

func renvoScalarKindSize(renvoNativeIntSize int, kind int) int {
	if kind == renvoTypeByte || kind == renvoTypeBool || kind == renvoTypeInt8 {
		return 1
	}
	if kind == renvoTypeInt {
		return renvoNativeIntSize
	}
	if kind == renvoTypeInt16 || kind == renvoTypeUint16 {
		return 2
	}
	if kind == renvoTypeInt32 || kind == renvoTypeUint32 || kind == renvoTypeFloat32 {
		return 4
	}
	return renvoBackendValueSlotSize
}

func renvoNativeScalarStorageSize(renvoNativeIntSize int, kind int) int {
	if kind == renvoTypePointer {
		return renvoNativeIntSize
	}
	return renvoScalarKindSize(renvoNativeIntSize, kind)
}

func renvoTypeIsStruct(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeStruct
}

func renvoTypeUsesNativeABI(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	if typ <= 0 || typ >= len(m.types) {
		return false
	}
	if m.types[typ].nativeAlign > 0 {
		return true
	}
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	if t.nativeAlign > 0 {
		return true
	}
	if t.kind == renvoTypePointer && t.elem > 0 && t.elem < len(m.types) {
		elem := renvoResolveType(m, t.elem)
		renvoNonNil(elem)
		return elem.nativeAlign > 0
	}
	return false
}

func renvoTypeUsesHiddenResult(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeStruct || t.kind == renvoTypeArray || t.kind == renvoTypeInterface || m.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(t.kind) || t.kind == renvoTypeComplex)
}

func renvoTypeIsTuple(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	if t.kind != renvoTypeStruct || t.count <= 1 {
		return false
	}
	for i := 0; i < t.count; i++ {
		field := m.fields[t.first+i]
		if field.nameEnd > field.nameStart {
			return false
		}
	}
	return true
}

func renvoAlignTo8(v int) int {
	return (v + renvoBackendValueSlotSize - 1) &^ (renvoBackendValueSlotSize - 1)
}

func renvoFindTokenTextInRange(p *renvoProgram, start int, end int, text byte) int {
	renvoNonNil(p)
	i := start
	for i < end {
		if renvoTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start - 1
}

func renvoBytesEqualRange(src []byte, aStart int, aEnd int, bStart int, bEnd int) bool {
	if aEnd-aStart != bEnd-bStart {
		return false
	}
	if aStart == bStart {
		return true
	}
	for aStart < aEnd {
		if renvo_runtime_UnsafeByteAt(src, aStart) != renvo_runtime_UnsafeByteAt(src, bStart) {
			return false
		}
		aStart++
		bStart++
	}
	return true
}

func renvoBytesPrefixText(src []byte, start int, end int, prefix string) bool {
	if end-start < len(prefix) {
		return false
	}
	return renvoBytesEqualText(src, start, start+len(prefix), prefix)
}

func renvoBytesSuffixText(src []byte, start int, end int, suffix string) bool {
	if end-start < len(suffix) {
		return false
	}
	return renvoBytesEqualText(src, end-len(suffix), end, suffix)
}

// Shared scalar code generation.
func renvoBindFunctionParams(g *renvoLinearGen, fnIndex int) {
	renvoNonNil(g)
	meta := g.meta
	fn := &meta.funcs[fnIndex]
	localBase := g.localCount
	g.deferStackFloor = -1
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := renvoAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if fn.literalTok > 0 && i == 0 {
			g.closureEnvOffset = offset
		}
	}
	g.deferStackFloor = 0
	callWord := 0
	if renvoTypeUsesHiddenResult(meta, fn.resultType) {
		callWord = renvoBackendHiddenResultWordCount
	}
	entry := fnIndex == g.prog.entryFunc
	for at := 0; at < fn.paramCount; at++ {
		i := fn.paramCount - 1 - at
		if entry {
			i = at
		}
		param := &meta.params[fn.firstParam+i]
		offset := g.locals[localBase+i].offset
		paramType := renvoResolveType(meta, param.typ)
		renvoNonNil(paramType)
		if paramType.kind == renvoTypeSlice {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			renvoStoreIncomingCallWord(g, callWord+2, offset-2*renvoBackendValueSlotSize)
			callWord += renvoBackendSliceWordCount
			continue
		}
		if paramType.kind == renvoTypeString {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			callWord += renvoBackendStringWordCount
			continue
		}
		if paramType.kind == renvoTypeInterface {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			callWord += 2
			continue
		}
		if renvoTypeKindIsComplex(paramType.kind) && !(g.c.renvoNativeIntSize == 4 && paramType.kind == renvoTypeComplex) {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, renvoComplexSecondaryStackOffset(g, param.typ, offset))
			callWord += 2
			continue
		}
		if renvoPreparedBackendActive != 0 && renvoStructArgByReference(g, paramType.kind) {
			// Keep the pointer in the aggregate's destination slot until every
			// incoming call register has been saved. The later copy overwrites it.
			renvoStoreIncomingCallWord(g, callWord, offset)
			callWord++
			continue
		}
		if paramType.kind == renvoTypeStruct || paramType.kind == renvoTypeArray || g.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(paramType.kind) || paramType.kind == renvoTypeComplex) {
			size := renvoTypeSize(meta, param.typ)
			wordSize := renvoCallWordSize(g, param.typ)
			for at := 0; at < size; at += wordSize {
				renvoStoreIncomingCallWord(g, callWord, offset-at)
				callWord++
			}
			continue
		}
		renvoStoreIncomingCallWord(g, callWord, offset)
		callWord++
	}
	// Copy by-reference aggregates only after every incoming call register has
	// been saved. The copy uses the secondary register, which can itself still
	// contain a later parameter while the entry sequence is being bound.
	if renvoPreparedBackendActive != 0 {
		for at := 0; at < fn.paramCount; at++ {
			i := fn.paramCount - 1 - at
			if entry {
				i = at
			}
			param := &meta.params[fn.firstParam+i]
			paramType := renvoResolveType(meta, param.typ)
			if !renvoStructArgByReference(g, paramType.kind) {
				continue
			}
			offset := g.locals[localBase+i].offset
			renvoAsmLoadSecondaryStack(&g.asm, offset)
			renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(meta, param.typ))
		}
	}
	for i := 0; i < fn.paramCount; i++ {
		local := &g.locals[localBase+i]
		if local.captureOff >= 0 {
			continue
		}
		g.stackUsed = renvoAlignTo8(g.stackUsed + renvoBackendValueSlotSize)
		renvoRecordStackPeak(g)
		local.captureOff = g.stackUsed
		renvoAllocateCapturedCell(g, local.captureOff, local.size)
	}
	renvoMoveCapturedLocals(g, true)
	if renvoFixedTarget == 0 {
		for i := 0; i < fn.paramCount; i++ {
			paramIndex := fn.firstParam + i
			if paramIndex < 0 || paramIndex >= len(g.paramConstValid) || !g.paramConstValid[paramIndex] {
				continue
			}
			param := &meta.params[paramIndex]
			if renvoLocalConstTrackable(g, param.typ, param.nameStart, param.nameEnd, fn.bodyStart) {
				resolved := renvoResolveType(meta, param.typ)
				renvoNonNil(resolved)
				renvoSetLocalConstAtOffset(g, g.locals[localBase+i].offset, g.paramConstValues[paramIndex], resolved.kind)
			}
		}
	}
}

func renvoBindClosureCaptures(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	closureIndex := renvoClosureIndexByFunction(g.meta, fnIndex)
	if closureIndex < 0 {
		return true
	}
	info := &g.meta.closures[closureIndex]
	if !info.ready || g.closureEnvOffset <= 0 {
		return false
	}
	for i := 0; i < info.captureCount; i++ {
		capture := &g.meta.captures[info.firstCapture+i]
		g.stackUsed = renvoAlignTo8(g.stackUsed + renvoBackendValueSlotSize)
		renvoRecordStackPeak(g)
		captureOff := g.stackUsed
		g.deferStackFloor = -1
		renvoAddTypedLocal(g, capture.nameStart, capture.nameEnd, capture.typ)
		g.deferStackFloor = 0
		g.locals[g.localCount-1].captureOff = captureOff
		g.hasCapturedLocals = true
		renvoAsmLoadPrimaryStackMemory(&g.asm, g.closureEnvOffset, (i+1)*renvoBackendValueSlotSize)
		renvoAsmStorePrimaryStack(&g.asm, captureOff)
		renvoMoveCapturedLocal(g, g.localCount-1, false)
	}
	return true
}

func renvoClosureIndexByFunction(meta *renvoMeta, fnIndex int) int {
	renvoNonNil(meta)
	for i := 0; i < len(meta.closures); i++ {
		if meta.closures[i].fnIndex == fnIndex {
			return i
		}
	}
	return -1
}

func renvoBindNamedResults(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if fn.firstResult < 0 || fn.resultCount < 0 || fn.firstResult+fn.resultCount > len(g.meta.params) {
		return false
	}
	for i := 0; i < fn.resultCount; i++ {
		result := &g.meta.params[fn.firstResult+i]
		offset := renvoAddTypedLocal(g, result.nameStart, result.nameEnd, result.typ)
		renvoZeroLocalAtOffset(g, offset)
	}
	renvoMoveCapturedLocals(g, true)
	return true
}

func renvoEnsurePanicState(g *renvoLinearGen) {
	renvoNonNil(g)
	if g.mainThreadStateOff > 0 {
		return
	}
	g.threadStatePointerOff = g.asm.bssSize
	g.asm.bssSize += renvoBackendValueSlotSize
	g.mainThreadStateOff = g.asm.bssSize
	g.asm.bssSize += renvoThreadStateSize
}

const renvoThreadStateSize = 7 * renvoBackendValueSlotSize
const renvoThreadPanicValueOff = 0
const renvoThreadPanicTypeOff = renvoBackendValueSlotSize
const renvoThreadPanicIDOff = 2 * renvoBackendValueSlotSize
const renvoThreadPanicNextIDOff = 3 * renvoBackendValueSlotSize
const renvoThreadPanicPrevOff = 4 * renvoBackendValueSlotSize
const renvoThreadPanicDeferPendingOff = 5 * renvoBackendValueSlotSize
const renvoThreadPanicRecoveredOff = 6 * renvoBackendValueSlotSize

func renvoEmitInitializeThreadState(g *renvoLinearGen) {
	if !g.meta.panicEnabled {
		return
	}
	renvoEnsurePanicState(g)
	renvoAsmPrimaryBssAddr(&g.asm, g.mainThreadStateOff)
	renvoEmitInstallThreadState(g)
	renvoAsmStorePrimaryBss(&g.asm, g.threadStatePointerOff)
}

func renvoAsmCopyThreadStateToStack(g *renvoLinearGen, stateOffset int, stackOffset int) {
	renvoAsmLoadPrimaryThreadState(g, stateOffset)
	renvoAsmStorePrimaryStack(&g.asm, stackOffset)
}

func renvoEmitJumpIfThreadStateEqualsStack(g *renvoLinearGen, stateOffset int, stackOffset int, label int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryThreadState(g, stateOffset)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryStack(a, stackOffset)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, label)
}

func renvoEmitStorePanicNodeField(g *renvoLinearGen, nodeOffset int, stateOffset int, displacement int) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryThreadState(g, stateOffset)
	renvoAsmLoadSecondaryStack(&g.asm, nodeOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, displacement)
}

func renvoEmitLoadPanicNodeField(g *renvoLinearGen, nodeOffset int, stateOffset int, displacement int) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryStackMemory(&g.asm, nodeOffset, displacement)
	renvoAsmStorePrimaryThreadState(g, stateOffset)
}

func renvoPrepareFunctionControl(g *renvoLinearGen) bool {
	renvoNonNil(g)
	g.deferHeadOffset = 0
	g.deferReturnLabel = 0
	g.deferResultOffset = 0
	g.panicEntryIDOffset = 0
	g.panicRecoverAllowedOffset = 0
	g.deferSites = nil
	g.emittingDefers = false
	g.suppressPanicCheck = false
	if !g.meta.panicEnabled {
		return true
	}
	if renvoProgramUsesC11Semantics(g.prog) &&
		(g.currentFunc < 0 || g.currentFunc >= len(g.meta.funcs) ||
			!renvoTokenRangeHasIdent(g.prog, g.meta.funcs[g.currentFunc].bodyStart, g.meta.funcs[g.currentFunc].bodyEnd, "defer")) {
		return true
	}
	g.panicEntryIDOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmCopyThreadStateToStack(g, renvoThreadPanicIDOff, g.panicEntryIDOffset)
	g.panicRecoverAllowedOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmCopyThreadStateToStack(g, renvoThreadPanicDeferPendingOff, g.panicRecoverAllowedOffset)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicDeferPendingOff)
	g.deferHeadOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, g.deferHeadOffset, 0)
	g.deferReturnLabel = renvoAsmNewLabel(&g.asm)
	fn := &g.meta.funcs[g.currentFunc]
	if fn.resultType != 0 && fn.resultCount == 0 && !renvoTypeUsesHiddenResult(g.meta, fn.resultType) {
		g.deferResultOffset = renvoAddUnnamedLocal(g, fn.resultType)
		renvoZeroLocalAtOffset(g, g.deferResultOffset)
	}
	return true
}

func renvoEmitPostCallPanicCheck(g *renvoLinearGen) {
	renvoNonNil(g)
	if !g.meta.panicEnabled || g.deferReturnLabel <= 0 || g.suppressPanicCheck || g.emittingDefers {
		return
	}
	renvoAsmPushPrimary(&g.asm)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmCmpPrimaryImm8(&g.asm, 0)
	noneLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmJzLabel(&g.asm, noneLabel)
	samePanicLabel := renvoAsmNewLabel(&g.asm)
	renvoEmitJumpIfThreadStateEqualsStack(g, renvoThreadPanicIDOff, g.panicEntryIDOffset, samePanicLabel)
	renvoAsmJmpMarkLabel(&g.asm, g.deferReturnLabel, samePanicLabel)
	renvoAsmMarkLabel(&g.asm, noneLabel)
	renvoAsmPopPrimary(&g.asm)
}

func renvoEmitDeferredReturn(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	fn := &g.meta.funcs[g.currentFunc]
	if stmt.exprStart < stmt.exprEnd {
		if fn.resultCount > 0 {
			parts, ok := renvoSplitTopLevelComma(g.prog, stmt.exprStart, stmt.exprEnd)
			if !ok {
				return false
			}
			if len(parts) == 2 && fn.resultCount > 1 {
				if !renvoEmitDeferredNamedTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
					return false
				}
				renvoMoveCapturedLocals(g, true)
				renvoAsmJmpLabel(&g.asm, g.deferReturnLabel)
				return true
			}
			if len(parts)/2 != fn.resultCount {
				return false
			}
			var values []int
			if fn.resultCount > 1 {
				values = make([]int, fn.resultCount)
			}
			for i := 0; i < fn.resultCount; i++ {
				result := &g.meta.params[fn.firstResult+i]
				offset := renvoFindResultLocalOffset(g, result.nameStart, result.nameEnd)
				if offset < 0 {
					return false
				}
				if len(values) > 0 {
					offset = renvoAddUnnamedLocal(g, result.typ)
					values[i] = offset
				}
				ep := renvoNewExprParse()
				renvoNonNil(ep)
				root := renvoParseExpressionRoot(ep, g.prog, parts[i*2], parts[i*2+1])
				if root < 0 || !renvoEmitExprToLocal(g, ep, root, offset) {
					return false
				}
			}
			// Return lists have assignment semantics: evaluate every value
			// before replacing any named result used by a later expression.
			for i := 0; i < len(values); i++ {
				result := &g.meta.params[fn.firstResult+i]
				offset := renvoFindResultLocalOffset(g, result.nameStart, result.nameEnd)
				if offset < 0 {
					return false
				}
				renvoEmitCopyStackToStack(g, values[i], offset, renvoTypeCopySize(g.meta, result.typ))
			}
		} else {
			if renvoTypeIsTuple(g.meta, fn.resultType) {
				if !renvoEmitTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
					return false
				}
			} else {
				ep := renvoNewExprParse()
				renvoNonNil(ep)
				root := renvoParseExpressionRoot(ep, g.prog, stmt.exprStart, stmt.exprEnd)
				if root < 0 {
					return false
				}
				if renvoTypeUsesHiddenResult(g.meta, fn.resultType) {
					if !renvoEmitStructReturnExpr(g, ep, root) {
						return false
					}
				} else if renvoTypeIsSlice(g.meta, fn.resultType) {
					if g.deferResultOffset <= 0 || !renvoEmitSliceReturnValueRegs(g, ep, root, fn.resultType) {
						return false
					}
					renvoAsmStoreSliceStack(&g.asm, g.deferResultOffset)
				} else if g.deferResultOffset <= 0 || !renvoEmitExprToLocal(g, ep, root, g.deferResultOffset) {
					return false
				}
			}
		}
	}
	renvoMoveCapturedLocals(g, true)
	renvoAsmJmpLabel(&g.asm, g.deferReturnLabel)
	return true
}

// Evaluate a forwarded result tuple once, then assign the named result locals
// before deferred calls run. Defers must observe (and may modify) those locals.
func renvoEmitDeferredNamedTupleReturn(g *renvoLinearGen, start int, end int) bool {
	fn := &g.meta.funcs[g.currentFunc]
	ep := renvoNewExprParse()
	root := renvoParseExpressionRoot(ep, g.prog, start, end)
	if root < 0 {
		return false
	}
	typ := renvoInferParsedExprType(g, ep, root)
	tuple := renvoResolveType(g.meta, typ)
	if !renvoTypeIsTuple(g.meta, typ) || tuple.count != fn.resultCount {
		return false
	}
	sourceOffset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitExprToLocal(g, ep, root, sourceOffset) {
		return false
	}
	for i := 0; i < fn.resultCount; i++ {
		from := g.meta.fields[tuple.first+i]
		result := g.meta.params[fn.firstResult+i]
		destination := renvoFindResultLocalOffset(g, result.nameStart, result.nameEnd)
		if destination < 0 {
			return false
		}
		source := sourceOffset - from.offset
		if renvoResolveType(g.meta, result.typ).kind == renvoTypeInterface && renvoResolveType(g.meta, from.typ).kind != renvoTypeInterface {
			if !renvoEmitConcreteLocalToInterface(g, from.typ, source, destination) {
				return false
			}
		} else {
			if !renvoTypesEquivalent(g.meta, from.typ, result.typ) {
				return false
			}
			renvoEmitCopyStackToStack(g, source, destination, renvoTypeCopySize(g.meta, result.typ))
		}
	}
	return true
}

func renvoEmitFunctionControlEpilogue(g *renvoLinearGen) bool {
	renvoNonNil(g)
	if g.deferReturnLabel <= 0 {
		return false
	}
	// Deferred arguments may retain pointers to locals whose lexical scopes
	// have ended. Keep the control-epilogue temporaries above every slot used
	// by the function so dispatch cannot overwrite those locals before the
	// deferred calls run.
	if g.stackUsed < g.stackPeak {
		g.stackUsed = g.stackPeak
	}
	a := &g.asm
	renvoAsmMarkLabel(a, g.deferReturnLabel)
	loopLabel := renvoAsmNewLabel(a)
	doneDefers := renvoAsmNewLabel(a)
	recordOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	tagOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	savedPanicIDOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	savedPanicPrevOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmLoadPrimaryStack(a, g.deferHeadOffset)
	renvoAsmJzPrimary(a, doneDefers)
	renvoAsmStorePrimaryStack(a, recordOffset)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmStorePrimaryStack(a, g.deferHeadOffset)
	renvoAsmLoadPrimaryStackMemory(a, recordOffset, renvoBackendValueSlotSize)
	renvoAsmStorePrimaryStack(a, tagOffset)
	for i := 0; i < len(g.deferSites); i++ {
		site := g.deferSites[i]
		nextLabel := renvoAsmNewLabel(a)
		renvoAsmJcmpStackImm(a, tagOffset, i+1, nextLabel, 0x95)
		handleOffset := renvoAddUnnamedLocal(g, site.funcType)
		renvoAsmLoadPrimaryStackMemory(a, recordOffset, 2*renvoBackendValueSlotSize)
		renvoAsmStorePrimaryStack(a, handleOffset)
		funcType := renvoResolveType(g.meta, site.funcType)
		renvoNonNil(funcType)
		argOffsets := make([]int, funcType.count)
		disp := 3 * renvoBackendValueSlotSize
		for j := 0; j < funcType.count; j++ {
			typ := g.meta.fields[funcType.first+j].typ
			argOffsets[j] = renvoAddUnnamedLocal(g, typ)
			renvoAsmLoadSecondaryStack(a, recordOffset)
			renvoAsmAddSecondaryImm(a, disp)
			size := renvoTypeCopySize(g.meta, typ)
			renvoEmitCopyMemSecondaryToStack(g, argOffsets[j], size)
			disp += renvoAlignTo8(size)
		}
		renvoAsmCopyThreadStateToStack(g, renvoThreadPanicIDOff, savedPanicIDOffset)
		renvoAsmCopyThreadStateToStack(g, renvoThreadPanicPrevOff, savedPanicPrevOffset)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmStorePrimaryThreadState(g, renvoThreadPanicRecoveredOff)
		g.emittingDefers = true
		if !renvoEmitFunctionValueDispatch(g, site.funcType, handleOffset, argOffsets, 0, site.directTarget-1) {
			g.emittingDefers = false
			return false
		}
		g.emittingDefers = false
		panicStateReady := renvoAsmNewLabel(a)
		renvoAsmLoadPrimaryStack(a, savedPanicIDOffset)
		renvoAsmJzPrimary(a, panicStateReady)
		renvoEmitJumpIfThreadStateEqualsStack(g, renvoThreadPanicIDOff, savedPanicIDOffset, panicStateReady)
		renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicRecoveredOff)
		renvoAsmJnzPrimary(a, panicStateReady)
		renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
		renvoAsmJzPrimary(a, panicStateReady)
		renvoAsmLoadPrimaryStack(a, savedPanicPrevOffset)
		renvoAsmStorePrimaryThreadState(g, renvoThreadPanicPrevOff)
		renvoAsmMarkLabel(a, panicStateReady)
		renvoAsmJmpMarkLabel(a, loopLabel, nextLabel)
	}
	renvoAsmJmpMarkLabel(a, loopLabel, doneDefers)
	renvoMoveCapturedResultsFromCells(g)
	panicReturn := renvoAsmNewLabel(a)
	normalReturn := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmJzPrimary(a, normalReturn)
	renvoEmitJumpIfThreadStateEqualsStack(g, renvoThreadPanicIDOff, g.panicEntryIDOffset, normalReturn)
	renvoAsmMarkLabel(a, panicReturn)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmLeave(a)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, normalReturn)
	fn := &g.meta.funcs[g.currentFunc]
	if fn.resultCount > 0 {
		if !renvoEmitBareReturnValues(g) {
			return false
		}
	} else if fn.resultType == 0 {
		// A void return has no register result to initialize.
	} else if renvoTypeUsesHiddenResult(g.meta, fn.resultType) {
		renvoAsmPrimaryImm(a, 0)
	} else if renvoTypeIsSlice(g.meta, fn.resultType) {
		renvoAsmLoadPrimarySecondaryStack(a, g.deferResultOffset, g.deferResultOffset-renvoBackendValueSlotSize)
		renvoAsmLoadTertiaryStack(a, g.deferResultOffset-2*renvoBackendValueSlotSize)
	} else if renvoTypeIsString(g.meta, fn.resultType) {
		renvoAsmLoadPrimarySecondaryStack(a, g.deferResultOffset, g.deferResultOffset-renvoBackendValueSlotSize)
	} else if renvoTypeKindIsComplex(renvoResolveType(g.meta, fn.resultType).kind) {
		renvoAsmLoadPrimarySecondaryStack(a, g.deferResultOffset, renvoComplexSecondaryStackOffset(g, fn.resultType, g.deferResultOffset))
	} else {
		renvoAsmLoadPrimaryStack(a, g.deferResultOffset)
	}
	renvoAsmLeave(a)
	renvoAsmRet(a)
	return true
}

func renvoMoveCapturedResultsFromCells(g *renvoLinearGen) {
	renvoNonNil(g)
	fn := &g.meta.funcs[g.currentFunc]
	for i := 0; i < fn.resultCount; i++ {
		result := &g.meta.params[fn.firstResult+i]
		if result.nameEnd <= result.nameStart {
			continue
		}
		localIndex := renvoFindLocalIndex(g, result.nameStart, result.nameEnd)
		if localIndex >= 0 {
			renvoMoveCapturedLocal(g, localIndex, false)
		}
	}
}

func renvoEmitBareReturnValues(g *renvoLinearGen) bool {
	renvoNonNil(g)
	meta := g.meta
	renvoNonNil(meta)
	fn := &meta.funcs[g.currentFunc]
	if fn.resultCount == 0 {
		return true
	}
	if fn.firstResult < 0 || fn.firstResult+fn.resultCount > len(meta.params) {
		return false
	}
	if fn.resultCount == 1 {
		result := &meta.params[fn.firstResult]
		offset := renvoFindLocalOffset(g, result.nameStart, result.nameEnd)
		if offset < 0 {
			return false
		}
		resolved := renvoResolveType(meta, result.typ)
		renvoNonNil(resolved)
		if renvoTypeUsesHiddenResult(meta, result.typ) {
			if g.returnStruct <= 0 {
				return false
			}
			renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
			renvoEmitCopyStackToMemSecondary(g, offset, 0, renvoTypeSize(g.meta, result.typ))
			return true
		}
		if resolved.kind == renvoTypeSlice {
			renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
			renvoAsmLoadTertiaryStack(&g.asm, offset-2*renvoBackendValueSlotSize)
			return renvoEmitCopySliceRegsToArena(g, result.typ)
		}
		if resolved.kind == renvoTypeString {
			renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
			return true
		}
		if renvoTypeKindIsComplex(resolved.kind) {
			renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, renvoComplexSecondaryStackOffset(g, result.typ, offset))
			return true
		}
		renvoAsmLoadPrimaryStack(&g.asm, offset)
		return true
	}
	tuple := renvoResolveType(g.meta, fn.resultType)
	renvoNonNil(tuple)
	if tuple.kind != renvoTypeStruct || tuple.count != fn.resultCount || g.returnStruct <= 0 {
		return false
	}
	for i := 0; i < fn.resultCount; i++ {
		result := &g.meta.params[fn.firstResult+i]
		offset := renvoFindLocalOffset(g, result.nameStart, result.nameEnd)
		if offset < 0 {
			return false
		}
		field := &g.meta.fields[tuple.first+i]
		renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
		renvoEmitCopyStackToMemSecondary(g, offset, field.offset, renvoTypeSize(g.meta, result.typ))
	}
	return true
}
func renvoAsmImmFits8Signed(imm int) bool {
	return imm >= -128 && imm <= 127
}
func renvoAsmLoadPrimaryIntToken(a *renvoAsm, p *renvoProgram, tokIndex int) {
	renvoNonNil(a, p)
	value := renvoParseIntToken(p, tokIndex)
	if p.compilerInt32 && p.parsedIntHigh != value>>31 {
		renvoAsmPrimaryImm64(a, value, p.parsedIntHigh)
		return
	}
	renvoAsmPrimaryImm(a, value)
}

func renvoAsmPushStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, offset)
	renvoAsmPushPrimary(a)
}

func renvoAsmPushSliceRegs(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmPushTertiary(a)
	renvoAsmPushSecondary(a)
	renvoAsmPushPrimary(a)
}
func renvoAsmPushStringRegs(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmPushSecondary(a)
	renvoAsmPushPrimary(a)
}

func renvoAsmRecordRegisterPush(a *renvoAsm, register int) {
	// Store the end position as well as the register. A trailing byte with the
	// same value as a PUSH opcode can be immediate or displacement data; only a
	// push emitted through the register operation is safe for the pop peephole.
	a.lastPrimaryStoreEnd = -(len(a.code)*32 + register + 2)
}

func renvoAsmStorePrimaryStackSize(a *renvoAsm, offset int, size int) {
	renvoNonNil(a)
	if size >= a.c.renvoNativeIntSize {
		renvoAsmStorePrimaryStack(a, offset)
		return
	}
	// Preserve the value while using the primary register to address the
	// frame slot. Frame-relative offsets remain stable across the push.
	renvoAsmPushPrimary(a)
	renvoAsmAddressPrimaryStack(a, offset)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, size)
}
func renvoAsmStorePrimarySecondaryStack(a *renvoAsm, primaryOffset int, secondaryOffset int) {
	renvoNonNil(a)
	renvoAsmStorePrimaryStack(a, primaryOffset)
	renvoAsmStoreSecondaryStack(a, secondaryOffset)
}

func renvoAsmStoreStackImm(a *renvoAsm, offset int, value int) {
	renvoNonNil(a)
	renvoAsmPrimaryImm(a, value)
	renvoAsmStorePrimaryStack(a, offset)
}

func renvoAsmCopyBssToStackSlot(a *renvoAsm, bssOffset int, stackOffset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryBss(a, bssOffset)
	renvoAsmStorePrimaryStack(a, stackOffset)
}

func renvoAsmLoadPrimaryStackMemory(a *renvoAsm, stackOffset int, displacement int) {
	renvoNonNil(a)
	renvoAsmLoadSecondaryStack(a, stackOffset)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, displacement)
}

func renvoAsmCopyStackSlot(a *renvoAsm, src int, dest int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, src)
	renvoAsmStorePrimaryStack(a, dest)
}

func renvoAsmJgeStackStack(a *renvoAsm, left int, right int, label int) {
	renvoNonNil(a)
	renvoAsmJcmpStackStack(a, left, right, label, 0x9d)
}

func renvoAsmJltStackStack(a *renvoAsm, left int, right int, label int) {
	renvoNonNil(a)
	notLess := renvoAsmNewLabel(a)
	renvoAsmJgeStackStack(a, left, right, notLess)
	renvoAsmIncPrimary(a)
	renvoAsmJmpMarkLabel(a, label, notLess)
}

func renvoAsmLoadPrimarySecondaryStack(a *renvoAsm, primaryOffset int, secondaryOffset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, primaryOffset)
	renvoAsmLoadSecondaryStack(a, secondaryOffset)
}
func renvoAsmLoadPrimaryTertiaryStack(a *renvoAsm, primaryOffset int, tertiaryOffset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, primaryOffset)
	renvoAsmLoadTertiaryStack(a, tertiaryOffset)
}
func renvoAsmLoadSecondaryTertiaryStack(a *renvoAsm, secondaryOffset int, tertiaryOffset int) {
	renvoNonNil(a)
	renvoAsmLoadSecondaryStack(a, secondaryOffset)
	renvoAsmLoadTertiaryStack(a, tertiaryOffset)
}
func renvoAsmStoreStringBss(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmPushSecondary(a)
	renvoAsmStorePrimaryBss(a, offset)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryBss(a, offset+8)
}
func renvoAsmStoreSliceBss(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmPushTertiary(a)
	renvoAsmPushSecondary(a)
	renvoAsmStorePrimaryBss(a, offset)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryBss(a, offset+8)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryBss(a, offset+16)
}
func renvoAsmPopStoreStringMemSecondary(a *renvoAsm, disp int) {
	renvoNonNil(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp+8)
}
func renvoAsmPopStoreSliceMemSecondary(a *renvoAsm, disp int) {
	renvoNonNil(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp+8)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, disp+16)
}

func renvoAsmAddScaledTertiary(a *renvoAsm, scale int) {
	if scale != 1 {
		renvoAsmMulTertiaryImm(a, scale)
	}
	renvoAsmAddPrimaryTertiary(a)
}

func renvoAsmPopPrimaryToTertiary(a *renvoAsm) {
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopPrimary(a)
}
func renvoAsmJmpMarkLabel(a *renvoAsm, jumpLabel int, markLabel int) {
	renvoNonNil(a)
	renvoAsmJmpLabel(a, jumpLabel)
	renvoAsmMarkLabel(a, markLabel)
}
func renvoAsmJzPrimary(a *renvoAsm, label int) {
	renvoNonNil(a)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJzLabel(a, label)
}
func renvoAsmJnzPrimary(a *renvoAsm, label int) {
	renvoNonNil(a)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, label)
}

type renvoLocalInfo struct {
	nameStart      int
	nameEnd        int
	nameHash       int
	offset         int
	captureOff     int
	typ            int
	size           int
	constValue     int
	constValid     int
	flowConstValue int
	// One is a flow constant; -1/-2 encode address/scalar inline aliases.
	flowConstValid int
}

type renvoGlobalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
}

type renvoSliceLocation struct {
	offset   int
	typ      int
	expr     int
	mem      bool
	deref    bool
	indirect bool
	param    bool
	global   bool
	ok       bool
}

type renvoLinearGen struct {
	wasmMemoryRanges       []int
	prog                   *renvoProgram
	meta                   *renvoMeta
	asm                    renvoAsm
	funcLabels             []int
	funcReachable          []bool
	funcQueue              []int
	structuredHelperKinds  [64]int
	structuredHelperArgs   [64]int
	structuredHelperLabels [64]int
	structuredHelperCount  int
	currentFunc            int
	returnStruct           int
	closureEnvOffset       int
	// Negative while binding captures; otherwise the frame floor retained by defer.
	deferStackFloor           int
	deferHeadOffset           int
	deferReturnLabel          int
	deferResultOffset         int
	panicEntryIDOffset        int
	panicRecoverAllowedOffset int
	deferSites                []renvoDeferSite
	emittingDefers            bool
	suppressPanicCheck        bool
	threadStatePointerOff     int
	mainThreadStateOff        int
	stackInitLabel            int
	stackSwitchLabel          int
	runtimeFaultLabel         int
	// Runtime helpers have distinct calling conventions, so keep their label
	// state named and pass the exact slot to architecture-specific emitters.
	runtimeNonNilLabel       int
	runtimeSecondaryLabel    int
	runtimeBoundsLabel       int
	runtimeByteIndexLabel    int
	runtimeWordIndexLabel    int
	runtimeWideIndexLabel    int
	runtimeSliceBoundsLabel  int
	checkedPointerLocals     int
	invalidatedPointerLocals int
	divideCheckLabel         int
	remainderCheckLabel      int
	nativeShiftLeftLabel     int
	nativeShiftSignedLabel   int
	nativeShiftUnsignedLabel int
	wideBinaryLabel          int
	wideCompareLabel         int
	locals                   []renvoLocalInfo
	localCount               int
	hasCapturedLocals        bool
	addressNamesReady        bool
	addressNameTokens        []int
	localCacheStart          int
	localCacheCount          int
	localCacheIndex          int
	stackUsed                int
	stackPeak                int
	arenaSize                int
	fieldIndex               int
	fieldOffset              int
	fieldPointerIndex        int
	fieldPointerOffset       int
	fieldCacheStart          int
	globals                  []renvoGlobalInfo
	gotoLabels               []renvoGlobalInfo
	breakLabels              []int
	continueLabels           []int
	breakDepth               int
	continueDepth            int
	pendingControl           int
	streqLabel               int
	streqEmitted             bool
	append8Label             int
	append8Emitted           bool
	append64Label            int
	append64Emitted          bool
	appendAddrLabel          int
	appendAddrEmitted        bool
	appendBytesLabel         int
	appendBytesEmitted       bool
	arenaAllocLabel          int
	persistentAllocLabel     int
	arenaFaultLabel          int
	makeZeroLabel            int
	makeZeroEmitted          bool
	stringHeapOff            int
	stringHeapEndOff         int
	stringHeapDataOff        int
	stringHeapReady          int
	winReadLabel             int
	winReadEmitted           bool
	winWriteLabel            int
	winWriteEmitted          bool
	printIntLabel            int
	printIntEmitted          bool
	printIntBufferOff        int
	darwinEntryOff           int
	lastRangeReturns         bool
	scopeBase                int
	scopeValueType           int
	scopeValueOffset         int
	scopeValueNameStart      int
	scopeValueNameEnd        int
	constEvalIota            int
	constEvalIotaValid       int
	fixedTargetValue         int
	fixedTargetState         int
	fixedPrunedReturns       bool
	kernelInitLabel          int
	kernelExitLabel          int
	kernelCallbackLabels     []int
	replRestoreOffsets       []int
	c                        *renvoCompileContext
	// Optional whole-program and object-mode analysis state is cold in the
	// compact fixed-target compiler. Keep it after the long-lived core layout.
	object                  *renvoObjectGenState
	funcSingleCallState     []int
	paramConstValues        []int
	paramConstValid         []bool
	objectCABIWrapperLabels []int
	objectCABINameStarts    []int
	objectCABINameEnds      []int
	constCallDepth          int
	constEvalFlow           bool
	flowControlDepth        int
}

const renvoStringInternSearchBytes = 512

func renvoAddStringData(g *renvoLinearGen, msg []byte) int {
	renvoNonNil(g)
	// Keep interning bounded. Large embedded assets should not make every later
	// literal rescan the entire static-data segment; missing an old match only
	// emits another copy and does not change program semantics.
	data := g.asm.data
	searchStart := len(data) - renvoStringInternSearchBytes
	if searchStart < 0 {
		searchStart = 0
	}
	size := len(msg)
	first := byte(0)
	if size > 0 {
		first = msg[0]
	}
	// The loop bounds cover the complete string and its trailing zero.
	for off := searchStart; off+size < len(data); off++ {
		if renvo_runtime_UnsafeByteAt(data, off) != first || renvo_runtime_UnsafeByteAt(data, off+size) != 0 {
			continue
		}
		match := true
		for i := 1; i < size; i++ {
			if renvo_runtime_UnsafeByteAt(data, off+i) != renvo_runtime_UnsafeByteAt(msg, i) {
				match = false
				break
			}
		}
		if match {
			return off
		}
	}
	msgOff := len(g.asm.data)
	g.asm.data = append(g.asm.data, msg...)
	g.asm.data = append(g.asm.data, 0)
	if g.asm.objectStrings != nil {
		g.asm.objectStrings.refs = append(g.asm.objectStrings.refs, msgOff, len(msg))
	}
	return msgOff
}

func renvoAddStringDataAligned(g *renvoLinearGen, msg []byte, alignment int) int {
	renvoNonNil(g)
	if alignment < 1 || alignment > 8 || alignment&(alignment-1) != 0 {
		return -1
	}
	if alignment == 1 {
		return renvoAddStringData(g, msg)
	}
	// Keep interning bounded. Large embedded assets should not make every later
	// literal rescan the entire static-data segment; missing an old match only
	// emits another copy and does not change program semantics.
	data := g.asm.data
	searchStart := len(data) - renvoStringInternSearchBytes
	if searchStart < 0 {
		searchStart = 0
	}
	size := len(msg)
	first := byte(0)
	if size > 0 {
		first = msg[0]
	}
	// The loop bounds cover the complete string and its trailing zero.
	for off := (searchStart + alignment - 1) & -alignment; off+size < len(data); off += alignment {
		if renvo_runtime_UnsafeByteAt(data, off) != first || renvo_runtime_UnsafeByteAt(data, off+size) != 0 {
			continue
		}
		match := true
		for i := 1; i < size; i++ {
			if renvo_runtime_UnsafeByteAt(data, off+i) != renvo_runtime_UnsafeByteAt(msg, i) {
				match = false
				break
			}
		}
		if match {
			return off
		}
	}
	for len(g.asm.data)&(alignment-1) != 0 {
		g.asm.data = append(g.asm.data, 0)
	}
	msgOff := len(g.asm.data)
	g.asm.data = append(g.asm.data, msg...)
	g.asm.data = append(g.asm.data, 0)
	if g.asm.objectStrings != nil {
		g.asm.objectStrings.refs = append(g.asm.objectStrings.refs, msgOff, len(msg))
	}
	return msgOff
}

func renvoFunctionLocalCap(fn *renvoFuncDecl) int {
	renvoNonNil(fn)
	localCap := 16
	if fn.bodyEnd-fn.bodyStart > 512 {
		localCap = 32
	}
	return localCap
}

func renvoEmitLinearRange(g *renvoLinearGen, start int, end int) bool {
	return renvoEmitLinearRangeMode(g, start, end, false)
}

func renvoEmitLinearRangeMode(g *renvoLinearGen, start int, end int, variableGroup bool) bool {
	renvoNonNil(g)
	constGroup := g.constEvalIotaValid != 0
	constGroupRepeatStart := 0
	constGroupRepeatEnd := 0
	var bp renvoBodyParse
	prog := g.meta.prog
	bp.prog = prog
	bp.stmtCount = 0
	bp.ok = true
	i := start
	lastKind := 0
	for bp.ok && i < end {
		if i < 0 || i >= renvoTokCount(prog) {
			break
		}
		if renvoTokCharIs(prog, i, ';') {
			i++
			continue
		}
		if renvoTokIsKind(prog, i, renvoTokEOF) {
			break
		}
		if renvoTokCharIs(prog, i, '}') {
			break
		}
		bp.stmtCount = 0
		next := renvoParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || bp.stmtCount != 1 {
			return false
		}
		stmt := renvoBodyStmtAt(&bp, 0)
		if !bp.ok {
			return false
		}
		if constGroup {
			if stmt.kind == renvoStmtAssign {
				constGroupRepeatStart = stmt.startTok
				constGroupRepeatEnd = stmt.endTok
			} else {
				if constGroupRepeatStart == 0 {
					return false
				}
				count := (stmt.endTok - stmt.startTok) * renvoTokenStride
				renvoCopyTokenData(prog, constGroupRepeatStart*renvoTokenStride, stmt.startTok*renvoTokenStride, count)
				stmt.startTok = constGroupRepeatStart
				stmt.endTok = constGroupRepeatEnd
			}
			stmt.kind = renvoStmtVar
		} else if variableGroup {
			// VarSpec entries omit the var keyword, but retain declaration
			// semantics, including initializer scope and zero initialization.
			stmt.kind = renvoStmtVar
		}
		lastKind = stmt.kind
		i = next
		statementLocalBase := g.localCount
		statementStackBase := g.stackUsed
		if !renvoEmitLinearStmt(g, &stmt) {
			if renvoFixedTarget == 0 {
				renvoPrintErr("renvo: failed to emit statement: ")
				write(2, prog.src[renvoTokStart(prog, stmt.startTok):renvoTokEnd(prog, stmt.endTok-1)], -1)
				renvoPrintErr("\n")
			}
			return false
		}
		if constGroup {
			g.constEvalIota++
		}
		for g.localCount > statementLocalBase {
			local := &g.locals[g.localCount-1]
			if local.nameStart != 0 || local.nameEnd != 0 {
				break
			}
			g.localCount--
		}
		if g.localCount > statementLocalBase {
			g.stackUsed = g.locals[g.localCount-1].offset
		} else {
			g.stackUsed = statementStackBase
		}
		renvoRestoreStackFloor(g, g.stackUsed)
		if stmt.kind == renvoStmtBlock && g.lastRangeReturns && !renvoRangeContainsControlTarget(prog, i, end) {
			return true
		}
		if g.fixedPrunedReturns {
			g.fixedPrunedReturns = false
			if !renvoRangeContainsControlTarget(prog, i, end) {
				g.lastRangeReturns = lastKind == renvoStmtReturn
				return true
			}
			g.lastRangeReturns = false
		}
	}
	g.lastRangeReturns = lastKind == renvoStmtReturn
	if !bp.ok {
		return false
	}
	return true
}

func renvoRangeContainsControlTarget(p *renvoProgram, start int, end int) bool {
	renvoNonNil(p)
	for tok := start; tok < end; tok++ {
		if renvoTokIsKind(p, tok, renvoTokGoto) ||
			renvoTokIsKind(p, tok, renvoTokIdent) && tok+1 < end && renvoTokCharIs(p, tok+1, ':') {
			return true
		}
	}
	return false
}

func renvoRangeContainsLabel(p *renvoProgram, start int, end int) bool {
	renvoNonNil(p)
	for tok := start; tok+1 < end; tok++ {
		if renvoTokIsKind(p, tok, renvoTokIdent) && renvoTokCharIs(p, tok+1, ':') {
			return true
		}
	}
	return false
}

func renvoRangeEndsControlExit(p *renvoProgram, start int, end int) bool {
	var bp renvoBodyParse
	bp.prog = p
	bp.ok = true
	var last renvoStmt
	for start < end {
		if renvoTokCharIs(p, start, ';') {
			start++
			continue
		}
		bp.stmtCount = 0
		next := renvoParseOneStatement(&bp, start, end)
		if !bp.ok || next <= start || bp.stmtCount != 1 {
			return false
		}
		last = renvoBodyStmtAt(&bp, 0)
		start = next
	}
	if last.kind == renvoStmtReturn || last.kind == renvoStmtGoto ||
		last.kind == renvoStmtBreak || last.kind == renvoStmtContinue {
		return true
	}
	if last.kind != renvoStmtFor ||
		last.exprStart != last.exprEnd &&
			(last.exprEnd != last.exprStart+1 || !renvoTokIdentIs(p, last.exprStart, "true")) {
		return false
	}
	for tok := last.bodyStart; tok < last.bodyEnd; tok++ {
		if renvoTokIsKind(p, tok, renvoTokBreak) || renvoTokIsKind(p, tok, renvoTokGoto) {
			return false
		}
	}
	return true
}

func renvoEmitScopedRange(g *renvoLinearGen, start int, end int) bool {
	renvoNonNil(g)
	oldLocalCount := g.localCount
	oldScopeBase := g.scopeBase
	oldStackUsed := g.stackUsed
	oldCheckedPointerLocals := g.checkedPointerLocals
	oldInvalidatedPointerLocals := g.invalidatedPointerLocals
	g.invalidatedPointerLocals = 0
	g.scopeBase = oldLocalCount
	g.flowControlDepth++
	typ := g.scopeValueType
	g.scopeValueType = 0
	if typ != 0 {
		offset := renvoAddTypedLocal(g, g.scopeValueNameStart, g.scopeValueNameEnd, typ)
		renvoCopyInterfaceValueToLocal(g, g.scopeValueOffset, typ, offset)
	}
	ok := renvoEmitLinearRange(g, start, end)
	g.flowControlDepth--
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	// A deferred argument can retain a pointer to a local after this lexical
	// scope ends. Keep the frame through the highest slot that was live when
	// a defer was registered, while still allowing later temporary slots
	// above that floor to be recycled normally.
	renvoRestoreStackFloor(g, oldStackUsed)
	childInvalidations := g.invalidatedPointerLocals
	g.checkedPointerLocals = oldCheckedPointerLocals &^ childInvalidations
	g.invalidatedPointerLocals = oldInvalidatedPointerLocals | childInvalidations
	return ok
}
func renvoSyncCapturedStmtTargets(g *renvoLinearGen, stmt *renvoStmt) {
	renvoNonNil(g, stmt)
	lhsStart := stmt.startTok
	lhsEnd := -1
	if stmt.kind == renvoStmtVar || stmt.kind == renvoStmtShort || stmt.kind == renvoStmtAssign {
		lhsEnd = renvoFindAssignmentToken(g.prog, stmt.startTok, stmt.endTok)
		if lhsEnd <= stmt.startTok && stmt.kind == renvoStmtVar {
			lhsEnd = stmt.endTok
		}
	} else if stmt.kind == renvoStmtExpr && stmt.endTok > stmt.startTok && (renvoTok2Is(g.prog, stmt.endTok-1, '+', '+') || renvoTok2Is(g.prog, stmt.endTok-1, '-', '-')) {
		lhsEnd = stmt.endTok - 1
	}
	if lhsEnd > lhsStart {
		rhs := lhsEnd + 1
		for rhs < stmt.endTok && renvoTokCharIs(g.prog, rhs, '(') {
			rhs++
		}
		addressResult := rhs < stmt.endTok && renvoTokCharIs(g.prog, rhs, '&')
		for tok := lhsStart; tok < lhsEnd; tok++ {
			// Selector names are fields, even when a same-named local exists.
			// Synchronizing that unrelated local can overwrite an alias write.
			if !renvoTokIsKind(g.prog, tok, renvoTokIdent) || tok > lhsStart && renvoTokCharIs(g.prog, tok-1, '.') {
				continue
			}
			localIndex := renvoFindLocalIndex(g, int(renvoTokStart(g.prog, tok)), int(renvoTokEnd(g.prog, tok)))
			if localIndex < 0 {
				continue
			}
			renvoMoveCapturedLocal(g, localIndex, true)
			if stmt.kind == renvoStmtShort || stmt.kind == renvoStmtAssign {
				next := tok + 1
				for next < lhsEnd && renvoTokCharIs(g.prog, next, ')') {
					next++
				}
				if (next == lhsEnd || renvoTokCharIs(g.prog, next, ',')) && (tok == lhsStart || !renvoTokCharIs(g.prog, tok-1, '*')) {
					renvoInvalidateCheckedPointerLocal(g, localIndex)
				}
			}
			if addressResult && localIndex < g.c.renvoNativeIntSize*8-1 && int(renvoTokStart(g.prog, tok)) == stmt.nameStart {
				g.checkedPointerLocals |= 1 << localIndex
			}
		}
	}
}

func renvoEmitLinearStmt(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	if stmt.kind == renvoStmtLabel || stmt.kind == renvoStmtGoto || stmt.kind == renvoStmtBreak || stmt.kind == renvoStmtContinue {
		g.checkedPointerLocals = 0
		if renvoFixedTarget == 0 {
			renvoClearAllLocalFlowConsts(g)
		}
	}
	renvoMoveCapturedLocals(g, false)
	if !renvoEmitLinearStmtCore(g, stmt) {
		return false
	}
	renvoSyncCapturedStmtTargets(g, stmt)
	if stmt.kind == renvoStmtLabel || stmt.kind == renvoStmtGoto || stmt.kind == renvoStmtBreak || stmt.kind == renvoStmtContinue {
		g.checkedPointerLocals = 0
	}
	return true
}

func renvoInvalidateCheckedPointerLocal(g *renvoLinearGen, localIndex int) {
	renvoNonNil(g)
	if localIndex >= g.c.renvoNativeIntSize*8-1 {
		return
	}
	bit := 1 << localIndex
	g.checkedPointerLocals &^= bit
	g.invalidatedPointerLocals |= bit
}

func renvoEmitLinearStmtCore(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	if stmt.kind != renvoStmtLabel && stmt.kind != renvoStmtFor && stmt.kind != renvoStmtSwitch {
		g.pendingControl = 0
	}
	if stmt.kind == renvoStmtExpr {
		if renvoEmitLinearPrintStmt(g, stmt) {
			return true
		}
		if renvoEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		root := &ep.exprs[rootIndex]
		if root.kind != renvoExprCall {
			return false
		}
		if renvoExprIdentCode(p, ep, root.left) == renvoIdentDelete && renvoFuncInfoFromCall(g, ep, root.left) < 0 {
			return false
		}
		resultType := renvoInferParsedExprType(g, ep, rootIndex)
		if renvoTypeUsesHiddenResult(g.meta, resultType) {
			offset := renvoAddUnnamedLocal(g, resultType)
			return renvoEmitStructCallToLocal(g, ep, rootIndex, resultType, offset)
		}
		if !renvoEmitIntExpr(g, ep, rootIndex) {
			return false
		}
		return true
	}
	if stmt.kind == renvoStmtDefer {
		return renvoEmitDeferStmt(g, stmt)
	}
	if stmt.kind == renvoStmtVar || stmt.kind == renvoStmtShort || stmt.kind == renvoStmtAssign {
		if !renvoEmitLinearAssign(g, stmt) {
			return false
		}
		return true
	}
	if stmt.kind == renvoStmtReturn {
		if g.deferReturnLabel > 0 {
			return renvoEmitDeferredReturn(g, stmt)
		}
		renvoMoveCapturedLocals(g, true)
		if stmt.exprStart == stmt.exprEnd {
			if !renvoEmitBareReturnValues(g) {
				return false
			}
			renvoAsmLeave(a)
			renvoAsmRet(a)
			return true
		}
		resultType := g.meta.funcs[g.currentFunc].resultType
		if renvoTypeIsTuple(g.meta, resultType) {
			if !renvoEmitTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
				return false
			}
			renvoAsmLeave(a)
			renvoAsmRet(a)
			return true
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		if renvoTypeUsesHiddenResult(g.meta, resultType) {
			if !renvoEmitStructReturnExpr(g, ep, rootIndex) {
				return false
			}
		} else if renvoTypeIsSlice(g.meta, resultType) {
			if !renvoEmitSliceReturnValueRegs(g, ep, rootIndex, resultType) {
				return false
			}
		} else if renvoTypeIsString(g.meta, resultType) {
			if !renvoEmitStringValueRegs(g, ep, rootIndex) {
				return false
			}
		} else {
			resultResolved := renvoResolveType(g.meta, resultType)
			renvoNonNil(resultResolved)
			if renvoTypeKindIsComplex(resultResolved.kind) {
				if !renvoEmitComplexValueRegsForKind(g, ep, rootIndex, resultResolved.kind) {
					return false
				}
			} else if !renvoEmitScalarExprForKind(g, ep, rootIndex, resultResolved.kind) {
				return false
			}
		}
		renvoAsmLeave(a)
		renvoAsmRet(a)
		return true
	}
	if stmt.kind == renvoStmtIf {
		return renvoEmitLinearIf(g, stmt)
	}
	if stmt.kind == renvoStmtFor {
		return renvoEmitLinearFor(g, stmt)
	}
	if stmt.kind == renvoStmtSwitch {
		return renvoEmitLinearSwitch(g, stmt)
	}
	if stmt.kind == renvoStmtBlock {
		if !renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
			return false
		}
		return true
	}
	if stmt.kind == renvoStmtType {
		start := stmt.startTok + 1
		if renvoTokCharIs(p, start, '(') {
			renvoParseScopedDeclGroup(g, g.meta, p, renvoTokType, start, stmt.endTok)
		} else {
			renvoParseScopedDeclEntry(g, g.meta, p, renvoTokType, start, stmt.endTok)
		}
		return g.meta.ok
	}
	if stmt.kind == renvoStmtGoto {
		label := renvoFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd) + stmt.exprStart
		renvoAsmJmpLabel(a, label)
		return true
	}
	if stmt.kind == renvoStmtLabel {
		label := renvoFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		renvoAsmMarkLabel(a, label)
		g.pendingControl = label + 1
		return true
	}
	if stmt.kind == renvoStmtBreak {
		if g.breakDepth == 0 {
			return false
		}
		renvoAsmJmpLabel(a, g.breakLabels[g.breakDepth-1])
		return true
	}
	if stmt.kind == renvoStmtContinue {
		if g.continueDepth == 0 {
			return false
		}
		renvoAsmJmpLabel(a, g.continueLabels[g.continueDepth-1])
		return true
	}
	return false
}

func renvoFindOrCreateGotoLabel(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	for i := 0; i < len(g.gotoLabels); i++ {
		info := g.gotoLabels[i]
		if renvoBytesEqualRange(g.prog.src, info.nameStart, info.nameEnd, nameStart, nameEnd) {
			return info.offset
		}
	}
	label := renvoAsmNewLabel(&g.asm)
	renvoAsmNewLabel(&g.asm)
	renvoAsmNewLabel(&g.asm)
	g.gotoLabels = append(g.gotoLabels, renvoGlobalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: label})
	return label
}

func renvoEmitDeferStmt(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	if g.deferHeadOffset <= 0 || stmt.exprStart >= stmt.exprEnd {
		return false
	}
	// Defer arguments are evaluated now, but an argument may be a pointer to a
	// local which leaves lexical scope before the deferred call runs. Pin the
	// currently live portion of the frame before allocating the defer's own
	// expression temporaries; later statements may reuse slots above this floor.
	g.deferStackFloor = g.stackUsed
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, g.prog, stmt.exprStart, stmt.exprEnd)
	if rootIndex < 0 {
		return false
	}
	call := &ep.exprs[rootIndex]
	if call.kind != renvoExprCall {
		return false
	}
	interfaceCall := renvoIsInterfaceMethodCall(g, ep, rootIndex)
	funcType := 0
	payloadOffset := 0
	if interfaceCall {
		selector := &ep.exprs[call.left]
		payloadOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		funcType = renvoEmitInterfaceMethodValue(g, ep, selector, call, payloadOffset)
		if funcType == 0 {
			return false
		}
	} else {
		funcType = renvoFunctionValueCalleeType(g, ep, call.left)
		if funcType == 0 {
			funcType = renvoInferParsedExprType(g, ep, call.left)
		}
		payloadOffset = renvoAddUnnamedLocal(g, funcType)
		if !renvoEmitExprToLocal(g, ep, call.left, payloadOffset) {
			return false
		}
	}
	t := renvoResolveType(g.meta, funcType)
	renvoNonNil(t)
	if t.kind != renvoTypeFunc || !renvoCallMatchesFuncType(t, call) {
		return false
	}
	argOffsets := make([]int, t.count)
	if !renvoPrepareFunctionValueArgs(g, ep, call, t, argOffsets) {
		return false
	}
	tag := len(g.deferSites) + 1
	disp := 3 * renvoBackendValueSlotSize
	for i := 0; i < len(argOffsets); i++ {
		typ := g.meta.fields[t.first+i].typ
		disp += renvoAlignTo8(renvoTypeCopySize(g.meta, typ))
	}
	directTarget := 0
	if renvoFixedTarget == 0 {
		if !interfaceCall {
			callee := &ep.exprs[call.left]
			if callee.kind == renvoExprIdent {
				fnIndex := renvoFuncInfoFromCall(g, ep, call.left)
				if fnIndex >= 0 {
					// Object-mode function values are raw C-ABI addresses rather than
					// the compact tags used by ordinary Renvo executables. A syntactically
					// direct defer already identifies its target, so retain that identity
					// and bypass dynamic function-value dispatch in the epilogue.
					directTarget = fnIndex + 1
				}
			}
		}
	}
	g.deferSites = append(g.deferSites, renvoDeferSite{funcType: funcType, directTarget: directTarget})
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	recordOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, disp)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(&g.asm, recordOffset)
	renvoAsmLoadPrimarySecondaryStack(&g.asm, g.deferHeadOffset, recordOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, 0)
	renvoAsmPrimaryImm(&g.asm, tag)
	renvoAsmLoadSecondaryStack(&g.asm, recordOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, renvoBackendValueSlotSize)
	renvoAsmLoadPrimarySecondaryStack(&g.asm, payloadOffset, recordOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, 2*renvoBackendValueSlotSize)
	disp = 3 * renvoBackendValueSlotSize
	for i := 0; i < len(argOffsets); i++ {
		typ := g.meta.fields[t.first+i].typ
		renvoAsmLoadSecondaryStack(&g.asm, recordOffset)
		size := renvoTypeCopySize(g.meta, typ)
		renvoEmitCopyStackToMemSecondary(g, argOffsets[i], disp, size)
		disp += renvoAlignTo8(size)
	}
	renvoAsmLoadPrimaryStack(&g.asm, recordOffset)
	renvoAsmStorePrimaryStack(&g.asm, g.deferHeadOffset)
	return true
}

func renvoEmitInterfaceMethodValue(g *renvoLinearGen, ep *renvoExprParse, selector *renvoExpr, call *renvoExpr, offset int) int {
	renvoNonNil(g, ep, selector)
	receiverType := renvoInferParsedExprType(g, ep, selector.left)
	receiverOffset := renvoAddUnnamedLocal(g, receiverType)
	if !renvoEmitInterfaceAssignToLocal(g, ep, selector.left, receiverOffset) {
		return 0
	}
	renvoAsmStoreStackImm(&g.asm, offset, 0)
	doneLabel := renvoAsmNewLabel(&g.asm)
	funcType := 0
	for fnIndex := 0; fnIndex < len(g.meta.funcs); fnIndex++ {
		fn := &g.meta.funcs[fnIndex]
		if !renvoInterfaceMethodNamed(g, fn, selector) {
			continue
		}
		candidate := renvoFunctionTypeFromInfoStart(g.meta, fnIndex, 1)
		if call != nil && !renvoCallMatchesFuncType(renvoResolveType(g.meta, candidate), call) {
			continue
		}
		if funcType == 0 {
			funcType = candidate
		} else if !renvoTypesEquivalent(g.meta, funcType, candidate) {
			return 0
		}
		nextLabel := renvoAsmNewLabel(&g.asm)
		renvoEmitInterfaceReceiverMatch(g, receiverOffset, fn.receiverType, nextLabel)
		renvoEmitBoundMethodHandle(g, fnIndex, fn.receiverType, receiverOffset,
			renvoInterfaceValueStoredIndirect(g.meta, fn.receiverType), offset)
		renvoAsmJmpMarkLabel(&g.asm, doneLabel, nextLabel)
	}
	if funcType == 0 {
		return 0
	}
	renvoAsmMarkLabel(&g.asm, doneLabel)
	return funcType
}

func renvoNewControlLabel(g *renvoLinearGen, delta int) int {
	renvoNonNil(g)
	if g.pendingControl > 0 {
		return g.pendingControl + delta
	}
	return renvoAsmNewLabel(&g.asm)
}
func renvoLoadCompilerFixedTarget(g *renvoLinearGen) {
	renvoNonNil(g)
	if g.fixedTargetState != 0 {
		return
	}
	g.fixedTargetState = -1
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if !renvoBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "renvoFixedTarget") {
			continue
		}
		if s.initStart >= s.initEnd {
			return
		}
		r := renvoEvalMetaConstExpr(g.meta, g.prog, s.initStart, s.initEnd, 0)
		if r.ok {
			g.fixedTargetState = 1
			g.fixedTargetValue = r.value
			return
		}
	}
}

func renvoEmitLinearElse(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.elseStart <= 0 {
		g.lastRangeReturns = false
		return true
	}
	if renvoTokIsKind(p, stmt.elseStart, renvoTokIf) && renvoTokIsKind(p, stmt.elseStart-1, renvoTokElse) {
		var nested renvoBodyParse
		nested.prog = p
		nested.stmtCount = 0
		nested.ok = true
		next := renvoParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || nested.stmtCount != 1 {
			return false
		}
		nestedStmt := renvoBodyStmtAt(&nested, 0)
		if !nested.ok {
			return false
		}
		return renvoEmitLinearStmt(g, &nestedStmt)
	}
	return renvoEmitScopedRange(g, stmt.elseStart, stmt.elseEnd)
}

func renvoEmitLinearIf(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	semi := renvoFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, semi)
	}
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, p, stmt.exprStart, stmt.exprEnd)
	if rootIndex < 0 {
		return false
	}
	renvoLoadCompilerFixedTarget(g)
	fixedValue := renvoEvalFixedTargetBool(g, ep, rootIndex, g.fixedTargetValue, g.fixedTargetState == 1)
	if renvoFixedTarget == 0 {
		literalBool := ep.exprs[rootIndex].kind == renvoExprBool
		if fixedValue < 0 && (literalBool || !renvoRangeContainsLabel(p, stmt.bodyStart, stmt.bodyEnd) &&
			!renvoRangeContainsLabel(p, stmt.elseStart, stmt.elseEnd)) {
			oldFlow := g.constEvalFlow
			g.constEvalFlow = oldFlow || renvoIsSysVObject(g.c)
			constant := renvoEvalConstExpr(g, ep, rootIndex)
			g.constEvalFlow = oldFlow
			if constant.ok {
				fixedValue = 0
				if constant.value != 0 {
					fixedValue = 1
				}
			}
		}
	}
	if fixedValue >= 0 {
		ok := false
		if fixedValue == 1 {
			ok = renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd)
		} else {
			ok = renvoEmitLinearElse(g, stmt)
		}
		if !ok {
			return false
		}
		if g.lastRangeReturns {
			g.fixedPrunedReturns = true
		}
		return true
	}
	flowCount := 0
	var flowBefore []int
	if renvoFixedTarget == 0 && renvoIsHostedObject(g.c) {
		flowCount = g.localCount
		flowBefore = renvoCaptureLocalFlowConsts(g, flowCount)
	}
	endLabel := renvoAsmNewLabel(a)
	elseLabel := endLabel
	if stmt.elseStart > 0 {
		elseLabel = renvoAsmNewLabel(a)
	}
	if !renvoEmitJumpIfFalse(g, ep, rootIndex, elseLabel) {
		return false
	}
	if !renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	thenReturns := g.lastRangeReturns
	thenExits := thenReturns
	if renvoFixedTarget == 0 {
		thenExits = thenExits || renvoRangeEndsControlExit(p, stmt.bodyStart, stmt.bodyEnd)
	}
	var flowThen []int
	if renvoFixedTarget == 0 && flowCount > 0 {
		flowThen = renvoCaptureLocalFlowConsts(g, flowCount)
	}
	if stmt.elseStart <= 0 {
		renvoAsmMarkLabel(a, endLabel)
		if renvoFixedTarget == 0 && flowCount > 0 {
			if thenExits {
				renvoRestoreLocalFlowConsts(g, flowBefore, flowCount)
			} else {
				renvoMergeLocalFlowConsts(g, flowBefore, flowThen, flowCount)
			}
		}
		g.lastRangeReturns = false
		return true
	}
	if !thenReturns {
		renvoAsmJmpLabel(a, endLabel)
	}
	renvoAsmMarkLabel(a, elseLabel)
	if renvoFixedTarget == 0 && flowCount > 0 {
		renvoRestoreLocalFlowConsts(g, flowBefore, flowCount)
	}
	if !renvoEmitLinearElse(g, stmt) {
		return false
	}
	elseReturns := g.lastRangeReturns
	elseExits := elseReturns
	if renvoFixedTarget == 0 {
		elseExits = elseExits || renvoRangeEndsControlExit(p, stmt.elseStart, stmt.elseEnd)
	}
	if renvoFixedTarget == 0 && flowCount > 0 {
		flowElse := renvoCaptureLocalFlowConsts(g, flowCount)
		if thenExits {
			renvoRestoreLocalFlowConsts(g, flowElse, flowCount)
		} else if elseExits {
			renvoRestoreLocalFlowConsts(g, flowThen, flowCount)
		} else {
			renvoMergeLocalFlowConsts(g, flowThen, flowElse, flowCount)
		}
	}
	renvoAsmMarkLabel(a, endLabel)
	return true
}
func renvoEmitLinearFor(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	if renvoFixedTarget == 0 {
		renvoClearAllLocalFlowConsts(g)
	}
	a := &g.asm
	p := g.prog
	semi1 := renvoFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi1 >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, semi1)
	}
	rangeTok := stmt.exprStart - 1
	for i := stmt.exprStart; i < stmt.exprEnd; i++ {
		if renvoTokIdentIs(p, i, "range") {
			rangeTok = i
			break
		}
	}
	if rangeTok >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, rangeTok)
	}
	endLabel := renvoNewControlLabel(g, 0)
	startLabel := renvoNewControlLabel(g, 1)
	g.pendingControl = 0
	renvoPushLoopLabels(g, endLabel, startLabel)
	renvoAsmMarkLabel(a, startLabel)
	renvoMoveCapturedLocals(g, false)
	if stmt.exprStart < stmt.exprEnd {
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		if !renvoEmitJumpIfFalse(g, ep, rootIndex, endLabel) {
			return false
		}
	}
	if !renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	renvoAsmJmpMarkLabel(a, startLabel, endLabel)
	renvoPopLoopLabels(g)
	return true
}

func renvoPushLoopLabels(g *renvoLinearGen, breakLabel int, continueLabel int) {
	renvoNonNil(g)
	if g.breakDepth < len(g.breakLabels) {
		g.breakLabels[g.breakDepth] = breakLabel
	} else {
		g.breakLabels = append(g.breakLabels, breakLabel)
	}
	if g.continueDepth < len(g.continueLabels) {
		g.continueLabels[g.continueDepth] = continueLabel
	} else {
		g.continueLabels = append(g.continueLabels, continueLabel)
	}
	g.breakDepth++
	g.continueDepth++
}

func renvoPopLoopLabels(g *renvoLinearGen) {
	renvoNonNil(g)
	g.breakDepth--
	g.continueDepth--
}

func renvoEmitLinearScopedControl(g *renvoLinearGen, stmt *renvoStmt, split int) bool {
	renvoNonNil(g, stmt)
	oldLocalCount := g.localCount
	oldScopeBase := g.scopeBase
	oldStackUsed := g.stackUsed
	g.scopeBase = oldLocalCount
	ok := false
	if stmt.kind == renvoStmtIf {
		if renvoEmitLinearSimpleRange(g, stmt.exprStart, split) {
			oldExprStart := stmt.exprStart
			stmt.exprStart = split + 1
			ok = renvoEmitLinearIf(g, stmt)
			stmt.exprStart = oldExprStart
		}
	} else if stmt.kind == renvoStmtSwitch {
		if renvoEmitLinearSimpleRange(g, stmt.exprStart, split) {
			oldExprStart := stmt.exprStart
			stmt.exprStart = split + 1
			ok = renvoEmitLinearSwitch(g, stmt)
			stmt.exprStart = oldExprStart
		}
	} else if renvoTokCharIs(g.prog, split, ';') {
		ok = renvoEmitLinearClassicForScoped(g, stmt, split)
	} else {
		ok = renvoEmitLinearRangeForScoped(g, stmt, split)
	}
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	renvoRestoreStackFloor(g, oldStackUsed)
	return ok
}

func renvoRestoreStackFloor(g *renvoLinearGen, stackUsed int) {
	if stackUsed < g.deferStackFloor {
		stackUsed = g.deferStackFloor
	}
	g.stackUsed = stackUsed
}

func renvoEmitLinearRangeForScoped(g *renvoLinearGen, stmt *renvoStmt, rangeTok int) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	a := &g.asm
	if rangeTok+1 >= stmt.exprEnd {
		return false
	}
	source := renvoNewExprParse()
	renvoNonNil(source)
	sourceIndex := renvoParseExpressionRoot(source, p, rangeTok+1, stmt.exprEnd)
	if sourceIndex < 0 {
		return false
	}
	sourceType := renvoInferParsedExprType(g, source, sourceIndex)
	resolved := renvoResolveType(g.meta, sourceType)
	renvoNonNil(resolved)
	arrayPointer := false
	if resolved.kind == renvoTypePointer {
		target := renvoResolveType(g.meta, resolved.elem)
		renvoNonNil(target)
		if target.kind == renvoTypeArray {
			resolved = target
			arrayPointer = true
		}
	}
	if resolved.kind != renvoTypeArray && resolved.kind != renvoTypeSlice && resolved.kind != renvoTypeString {
		return false
	}
	sourceOffset := renvoAddUnnamedLocal(g, sourceType)
	if !renvoEmitExprToLocal(g, source, sourceIndex, sourceOffset) {
		return false
	}
	sourceLenOffset := sourceOffset - 8
	indexOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(a, indexOffset, 0)
	widthOffset := 0
	if resolved.kind == renvoTypeString {
		widthOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	}

	keyOffset := 0
	valueOffset := 0
	rangeShort := false
	if rangeTok > stmt.exprStart {
		assignTok := renvoFindAssignmentToken(p, stmt.exprStart, rangeTok)
		if assignTok < stmt.exprStart || assignTok >= rangeTok {
			return false
		}
		targets, ok := renvoSplitTopLevelComma(p, stmt.exprStart, assignTok)
		if !ok || len(targets) < 2 || len(targets) > 4 {
			return false
		}
		rangeShort = renvoTok2Is(p, assignTok, ':', '=')
		keyType := renvoTypeInt
		valueType := resolved.elem
		if resolved.kind == renvoTypeString {
			valueType = renvoTypeInt32
		}
		keyOffset = renvoRangeTargetOffset(g, targets[0], targets[1], keyType, rangeShort)
		if keyOffset < 0 {
			return false
		}
		if len(targets) == 4 {
			valueOffset = renvoRangeTargetOffset(g, targets[2], targets[3], valueType, rangeShort)
			if valueOffset < 0 {
				return false
			}
		}
	}

	endLabel := renvoNewControlLabel(g, 0)
	continueLabel := renvoNewControlLabel(g, 1)
	g.pendingControl = 0
	startLabel := renvoAsmNewLabel(a)
	renvoPushLoopLabels(g, endLabel, continueLabel)
	renvoAsmMarkLabel(a, startLabel)
	renvoAsmPushStack(a, indexOffset)
	if resolved.kind == renvoTypeArray {
		renvoAsmPrimaryImm(a, resolved.count)
	} else {
		renvoAsmLoadPrimaryStack(a, sourceLenOffset)
	}
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
	renvoAsmJnzPrimary(a, endLabel)
	if rangeShort {
		for localIndex := 0; localIndex < g.localCount; localIndex++ {
			if g.locals[localIndex].offset == keyOffset || g.locals[localIndex].offset == valueOffset {
				renvoRebindCapturedLocal(g, localIndex)
			}
		}
	}
	if keyOffset > 0 {
		renvoAsmCopyStackSlot(a, indexOffset, keyOffset)
	}
	if resolved.kind == renvoTypeString {
		runeOffset := renvoAddUnnamedLocal(g, renvoTypeInt32)
		renvoEmitStringRangeDecode(g, sourceOffset, sourceLenOffset, indexOffset, runeOffset, widthOffset)
		if valueOffset > 0 {
			renvoAsmCopyStackSlot(a, runeOffset, valueOffset)
		}
	} else if valueOffset > 0 {
		renvoAsmLoadTertiaryStack(a, indexOffset)
		elemSize := renvoTypeSize(g.meta, resolved.elem)
		if elemSize != 1 {
			renvoAsmMulTertiaryImm(a, elemSize)
		}
		if resolved.kind == renvoTypeArray {
			if arrayPointer {
				renvoAsmLoadPrimaryStack(a, sourceOffset)
			} else {
				renvoAsmAddressPrimaryStack(a, sourceOffset)
			}
		} else {
			renvoAsmLoadPrimaryStack(a, sourceOffset)
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmAddSecondaryTertiary(a)
		elemKind := renvoResolveType(g.meta, resolved.elem).kind
		if elemSize < g.c.renvoNativeIntSize && renvoTypeKindIsScalarInt(elemKind) {
			// Scalar locals occupy native-word slots. Copying only the leading
			// bytes puts a narrow value in the high bits on big-endian targets.
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, elemSize)
			renvoAsmNormalizePrimaryForKind(a, elemKind)
			renvoAsmStorePrimaryStack(a, valueOffset)
		} else {
			renvoEmitCopyMemSecondaryToStack(g, valueOffset, elemSize)
		}
	}
	for localIndex := 0; localIndex < g.localCount; localIndex++ {
		if g.locals[localIndex].offset == keyOffset || g.locals[localIndex].offset == valueOffset {
			renvoMoveCapturedLocal(g, localIndex, true)
		}
	}
	if !renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	renvoAsmMarkLabel(a, continueLabel)
	if resolved.kind == renvoTypeString {
		renvoAsmLoadPrimaryTertiaryStack(a, indexOffset, widthOffset)
		renvoAsmAddPrimaryTertiary(a)
		renvoAsmStorePrimaryStack(a, indexOffset)
	} else {
		renvoAsmIncStack(a, indexOffset)
	}
	renvoAsmJmpMarkLabel(a, startLabel, endLabel)
	renvoPopLoopLabels(g)
	return true
}

func renvoEmitStringRangeDecode(g *renvoLinearGen, ptr int, length int, index int, runeOffset int, width int) {
	renvoNonNil(g)
	a := &g.asm
	b0 := renvoAddUnnamedLocal(g, renvoTypeByte)
	b1 := renvoAddUnnamedLocal(g, renvoTypeByte)
	b2 := renvoAddUnnamedLocal(g, renvoTypeByte)
	b3 := renvoAddUnnamedLocal(g, renvoTypeByte)
	next := renvoAddUnnamedLocal(g, renvoTypeInt)
	two := renvoAsmNewLabel(a)
	three := renvoAsmNewLabel(a)
	four := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	invalid := renvoAsmNewLabel(a)
	renvoEmitStringByteAt(g, ptr, index, b0)
	renvoAsmStoreStackImm(a, width, 1)
	renvoAsmCopyStackSlot(a, b0, runeOffset)
	renvoEmitStackLessImmJump(g, b0, 128, done)
	renvoAsmStoreStackImm(a, runeOffset, 65533)
	renvoEmitStackLessImmJump(g, b0, 194, done)
	renvoEmitStackLessImmJump(g, b0, 224, two)
	renvoEmitStackLessImmJump(g, b0, 240, three)
	renvoEmitStackLessImmJump(g, b0, 245, four)
	renvoAsmJmpLabel(a, done)

	renvoAsmMarkLabel(a, two)
	renvoEmitNextStringByte(g, ptr, length, index, next, b1, invalid)
	renvoEmitRunePart(g, runeOffset, b0, 192, 64, true)
	renvoEmitRunePart(g, runeOffset, b1, 128, 1, false)
	renvoAsmStoreStackImm(a, width, 2)
	renvoAsmJmpLabel(a, done)

	renvoAsmMarkLabel(a, three)
	renvoEmitNextStringByte(g, ptr, length, index, next, b1, invalid)
	e0ok := renvoAsmNewLabel(a)
	renvoEmitStackLessImmJump(g, b0, 225, e0ok)
	renvoEmitStackLessImmJump(g, b0, 237, e0ok)
	ed := renvoAsmNewLabel(a)
	renvoEmitStackLessImmJump(g, b0, 238, ed)
	renvoAsmJmpMarkLabel(a, e0ok, ed)
	renvoEmitStackGreaterEqualImmJump(g, b1, 160, invalid)
	renvoAsmJmpMarkLabel(a, e0ok, e0ok)
	notE0 := renvoAsmNewLabel(a)
	renvoEmitStackGreaterEqualImmJump(g, b0, 225, notE0)
	renvoEmitStackLessImmJump(g, b1, 160, invalid)
	renvoAsmMarkLabel(a, notE0)
	renvoEmitNextStringByte(g, ptr, length, next, next, b2, invalid)
	renvoEmitRunePart(g, runeOffset, b0, 224, 4096, true)
	renvoEmitRunePart(g, runeOffset, b1, 128, 64, false)
	renvoEmitRunePart(g, runeOffset, b2, 128, 1, false)
	renvoAsmStoreStackImm(a, width, 3)
	renvoAsmJmpLabel(a, done)

	renvoAsmMarkLabel(a, four)
	renvoEmitNextStringByte(g, ptr, length, index, next, b1, invalid)
	notF0 := renvoAsmNewLabel(a)
	renvoEmitStackGreaterEqualImmJump(g, b0, 241, notF0)
	renvoEmitStackLessImmJump(g, b1, 144, invalid)
	renvoAsmMarkLabel(a, notF0)
	f4 := renvoAsmNewLabel(a)
	validLead := renvoAsmNewLabel(a)
	renvoEmitStackGreaterEqualImmJump(g, b0, 244, f4)
	renvoAsmJmpMarkLabel(a, validLead, f4)
	renvoEmitStackGreaterEqualImmJump(g, b1, 144, invalid)
	renvoAsmMarkLabel(a, validLead)
	renvoEmitNextStringByte(g, ptr, length, next, next, b2, invalid)
	renvoEmitNextStringByte(g, ptr, length, next, next, b3, invalid)
	renvoEmitRunePart(g, runeOffset, b0, 240, 262144, true)
	renvoEmitRunePart(g, runeOffset, b1, 128, 4096, false)
	renvoEmitRunePart(g, runeOffset, b2, 128, 64, false)
	renvoEmitRunePart(g, runeOffset, b3, 128, 1, false)
	renvoAsmStoreStackImm(a, width, 4)
	renvoAsmJmpMarkLabel(a, done, invalid)
	renvoAsmMarkLabel(a, done)
}

func renvoEmitStringByteAt(g *renvoLinearGen, ptr int, index int, dest int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryTertiaryStack(a, ptr, index)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
	renvoAsmStorePrimaryStack(a, dest)
}

func renvoEmitNextStringByte(g *renvoLinearGen, ptr int, length int, from int, next int, dest int, invalid int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, from)
	renvoAsmIncPrimary(a)
	renvoAsmStorePrimaryStack(a, next)
	renvoAsmJgeStackStack(a, next, length, invalid)
	renvoEmitStringByteAt(g, ptr, next, dest)
	renvoEmitStackLessImmJump(g, dest, 128, invalid)
	renvoEmitStackGreaterEqualImmJump(g, dest, 192, invalid)
}

func renvoEmitRunePart(g *renvoLinearGen, dest int, source int, bias int, scale int, first bool) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmPrimaryImm(a, bias)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryStack(a, source)
	renvoAsmPopTertiary(a)
	renvoAsmSubPrimaryTertiary(a)
	if scale != 1 {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmMulTertiaryImm(a, scale)
		renvoAsmCopyTertiaryToPrimary(a)
	}
	if !first {
		renvoAsmLoadTertiaryStack(a, dest)
		renvoAsmAddPrimaryTertiary(a)
	}
	renvoAsmStorePrimaryStack(a, dest)
}

func renvoEmitStackLessImmJump(g *renvoLinearGen, offset int, value int, label int) {
	renvoNonNil(g)
	renvoAsmJcmpStackImm(&g.asm, offset, value, label, 0x9c)
}

func renvoEmitStackGreaterEqualImmJump(g *renvoLinearGen, offset int, value int, label int) {
	renvoNonNil(g)
	renvoAsmJcmpStackImm(&g.asm, offset, value, label, 0x9d)
}

func renvoRangeTargetOffset(g *renvoLinearGen, start int, end int, typ int, short bool) int {
	renvoNonNil(g)
	p := g.prog
	if end != start+1 || !renvoTokIsKind(p, start, renvoTokIdent) {
		return -1
	}
	nameStart := int(renvoTokStart(p, start))
	nameEnd := int(renvoTokEnd(p, start))
	if renvoBytesEqualText(p.src, nameStart, nameEnd, "_") {
		return 0
	}
	localIndex := renvoFindLocalIndex(g, nameStart, nameEnd)
	if short {
		localIndex = renvoFindLocalIndexInCurrentScope(g, nameStart, nameEnd)
		if localIndex < 0 {
			return renvoAddTypedLocal(g, nameStart, nameEnd, typ)
		}
	}
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}
func renvoEmitLinearSwitch(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	semi := renvoFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, semi)
	}
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := -1
	typeSwitch := false
	typeOperandEnd := stmt.exprEnd
	if stmt.exprEnd-stmt.exprStart >= 5 && renvoTokCharIs(p, stmt.exprEnd-4, '.') && renvoTokCharIs(p, stmt.exprEnd-3, '(') && renvoBytesEqualText(p.src, int(renvoTokStart(p, stmt.exprEnd-2)), int(renvoTokEnd(p, stmt.exprEnd-2)), "type") && renvoTokCharIs(p, stmt.exprEnd-1, ')') {
		typeSwitch = true
		typeOperandEnd -= 4
	}
	typeOperandStart := stmt.exprStart
	typeNameStart := 0
	typeNameEnd := 0
	if typeSwitch {
		assign := renvoFindAssignmentToken(p, stmt.exprStart, typeOperandEnd)
		if assign > stmt.exprStart {
			if assign != stmt.exprStart+1 || !renvoTokIsKind(p, stmt.exprStart, renvoTokIdent) || !renvoTok2Is(p, assign, ':', '=') {
				return false
			}
			typeNameStart = int(renvoTokStart(p, stmt.exprStart))
			typeNameEnd = int(renvoTokEnd(p, stmt.exprStart))
			typeOperandStart = assign + 1
		}
	}
	if stmt.exprStart < stmt.exprEnd {
		parseStart := stmt.exprStart
		parseEnd := stmt.exprEnd
		if typeSwitch {
			parseStart = typeOperandStart
			parseEnd = typeOperandEnd
		}
		rootIndex = renvoParseExpressionRoot(ep, p, parseStart, parseEnd)
		if rootIndex < 0 {
			return false
		}
	}
	stringSwitch := rootIndex >= 0 && renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, rootIndex))
	interfaceSwitch := rootIndex >= 0 && !typeSwitch && renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, rootIndex)).kind == renvoTypeInterface
	if renvoFixedTarget == 0 {
		if rootIndex >= 0 && !typeSwitch && !stringSwitch && !interfaceSwitch {
			constant := renvoEvalConstExpr(g, ep, rootIndex)
			if constant.ok {
				clause, known := renvoFindConstantSwitchClause(g, stmt, constant.value)
				if known {
					return renvoEmitConstantSwitchClause(g, stmt, clause)
				}
			}
		}
	}
	registerSwitch := false
	if renvoFixedTarget == 0 && renvoCanKeepSwitchPrimary(g) {
		if rootIndex >= 0 && !stringSwitch && !typeSwitch && !interfaceSwitch {
			registerSwitch = renvoSwitchCasesAreConstant(g, stmt)
		}
	}
	valueOffset := -1
	if !registerSwitch {
		valueType := renvoTypeInt
		if interfaceSwitch {
			valueType = renvoBuiltinTypeInterface
		}
		valueOffset = renvoAddUnnamedLocal(g, valueType)
	}
	typeValueOffset := 0
	typeValueType := 0
	lenOffset := 0
	if interfaceSwitch {
		if !renvoEmitInterfaceAssignToLocal(g, ep, rootIndex, valueOffset) {
			return false
		}
	} else if stringSwitch {
		lenOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		if !renvoEmitStringValueRegs(g, ep, rootIndex) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, valueOffset, lenOffset)
	} else if rootIndex >= 0 {
		if typeSwitch {
			typeValueType = renvoInferParsedExprType(g, ep, rootIndex)
			if renvoResolveType(g.meta, typeValueType).kind != renvoTypeInterface {
				return false
			}
			typeValueOffset = renvoAddUnnamedLocal(g, typeValueType)
			if !renvoEmitInterfaceAssignToLocal(g, ep, rootIndex, typeValueOffset) {
				return false
			}
			renvoAsmLoadPrimaryStack(a, typeValueOffset-renvoBackendValueSlotSize)
		} else if !renvoEmitIntExpr(g, ep, rootIndex) {
			return false
		}
		if !registerSwitch {
			renvoAsmStorePrimaryStack(a, valueOffset)
		}
	} else {
		renvoAsmStoreStackImm(a, valueOffset, 1)
	}

	endLabel := renvoNewControlLabel(g, 0)
	g.pendingControl = 0
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	clauseStarts := renvoFixedIntScratch(8)
	clauseLabels := renvoFixedIntScratch(8)
	defaultLabel := endLabel
	hasDefault := false
	i := stmt.bodyStart
	for i < stmt.bodyEnd {
		clause := renvoFindNextSwitchClause(p, i, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		label := renvoAsmNewLabel(a)
		clauseStarts = append(clauseStarts, clause)
		clauseLabels = append(clauseLabels, label)
		if renvoTokIsKind(p, clause, renvoTokDefault) {
			defaultLabel = label
			hasDefault = true
		}
		i = clause + 1
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		if renvoTokIsKind(p, clause, renvoTokCase) {
			if !renvoEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, typeSwitch, interfaceSwitch, clauseLabels[i]) {
				return false
			}
		}
	}
	if hasDefault {
		renvoAsmJmpLabel(a, defaultLabel)
	} else {
		renvoAsmJmpLabel(a, endLabel)
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		colon := renvoFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := renvoFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		fallsThrough := false
		bodyEmitEnd := bodyEnd
		for bodyEmitEnd > colon+1 && renvoTokCharIs(p, bodyEmitEnd-1, ';') {
			bodyEmitEnd--
		}
		if bodyEmitEnd > colon+1 && renvoTokIdentIs(p, bodyEmitEnd-1, "fallthrough") {
			fallsThrough = true
			bodyEmitEnd--
		}
		renvoAsmMarkLabel(a, clauseLabels[i])
		if typeSwitch && typeNameEnd > typeNameStart {
			caseType := typeValueType
			caseEnd := renvoFindExprBoundary(p, clause+1, colon)
			if !renvoTokIsKind(p, clause, renvoTokDefault) && caseEnd == colon && !renvoTokIdentIs(p, clause+1, "nil") {
				parsed := renvoParseType(g.meta, p, clause+1, caseEnd)
				caseType = parsed.typ
			}
			g.scopeValueType = caseType
			g.scopeValueOffset = typeValueOffset
			g.scopeValueNameStart = typeNameStart
			g.scopeValueNameEnd = typeNameEnd
			if caseType == 0 || !renvoEmitScopedRange(g, colon+1, bodyEmitEnd) {
				return false
			}
		} else if !renvoEmitScopedRange(g, colon+1, bodyEmitEnd) {
			return false
		}
		if fallsThrough && i+1 < len(clauseLabels) {
			renvoAsmJmpLabel(a, clauseLabels[i+1])
		} else {
			renvoAsmJmpLabel(a, endLabel)
		}
	}
	renvoAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}

func renvoSwitchCasesAreConstant(g *renvoLinearGen, stmt *renvoStmt) bool {
	p := g.prog
	for at := stmt.bodyStart; at < stmt.bodyEnd; {
		clause := renvoFindNextSwitchClause(p, at, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		colon := renvoFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		if renvoTokIsKind(p, clause, renvoTokCase) {
			for item := clause + 1; item < colon; {
				itemEnd := renvoFindExprBoundary(p, item, colon)
				if itemEnd <= item {
					return false
				}
				ep := renvoNewExprParse()
				root := renvoParseExpressionRoot(ep, p, item, itemEnd)
				if root < 0 || !renvoEvalConstExpr(g, ep, root).ok {
					return false
				}
				item = itemEnd
				if renvoTokCharIs(p, item, ',') {
					item++
				}
			}
		}
		at = colon + 1
	}
	return true
}

func renvoFindConstantSwitchClause(g *renvoLinearGen, stmt *renvoStmt, value int) (int, bool) {
	renvoNonNil(g, stmt)
	p := g.prog
	defaultClause := stmt.bodyEnd
	for at := stmt.bodyStart; at < stmt.bodyEnd; {
		clause := renvoFindNextSwitchClause(p, at, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		if renvoTokIsKind(p, clause, renvoTokDefault) {
			defaultClause = clause
			at = clause + 1
			continue
		}
		colon := renvoFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause+1 {
			return stmt.bodyEnd, false
		}
		for item := clause + 1; item < colon; {
			itemEnd := renvoFindExprBoundary(p, item, colon)
			if itemEnd <= item {
				return stmt.bodyEnd, false
			}
			ep := renvoNewExprParse()
			root := renvoParseExpressionRoot(ep, p, item, itemEnd)
			if root < 0 {
				return stmt.bodyEnd, false
			}
			candidate := renvoEvalConstExpr(g, ep, root)
			if !candidate.ok {
				return stmt.bodyEnd, false
			}
			if candidate.value == value {
				return clause, true
			}
			item = itemEnd
			if renvoTokCharIs(p, item, ',') {
				item++
			}
		}
		at = colon + 1
	}
	return defaultClause, true
}

func renvoEmitConstantSwitchClause(g *renvoLinearGen, stmt *renvoStmt, clause int) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	endLabel := renvoNewControlLabel(g, 0)
	g.pendingControl = 0
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)
	for clause < stmt.bodyEnd {
		colon := renvoFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := renvoFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		bodyEmitEnd := bodyEnd
		for bodyEmitEnd > colon+1 && renvoTokCharIs(p, bodyEmitEnd-1, ';') {
			bodyEmitEnd--
		}
		fallsThrough := bodyEmitEnd > colon+1 && renvoTokIdentIs(p, bodyEmitEnd-1, "fallthrough")
		if fallsThrough {
			bodyEmitEnd--
		}
		if !renvoEmitScopedRange(g, colon+1, bodyEmitEnd) {
			return false
		}
		if !fallsThrough {
			break
		}
		clause = bodyEnd
	}
	renvoAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}

func renvoCopyInterfaceValueToLocal(g *renvoLinearGen, sourceOffset int, typ int, valueOffset int) {
	renvoNonNil(g)
	kind := renvoResolveType(g.meta, typ).kind
	if kind == renvoTypeInterface {
		renvoEmitCopyStackToStack(g, sourceOffset, valueOffset, 2*renvoBackendValueSlotSize)
		return
	}
	if kind == renvoTypeComplex64 && g.c.renvoNativeIntSize == 8 {
		renvoAsmAddressPrimaryStack(&g.asm, sourceOffset)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoUnpackComplex64MemSecondaryRegs(g)
		renvoAsmStorePrimarySecondaryStack(&g.asm, valueOffset, renvoComplexSecondaryStackOffset(g, typ, valueOffset))
		return
	}
	size := renvoTypeSize(g.meta, typ)
	if !renvoInterfaceValueStoredIndirect(g.meta, typ) {
		if size <= g.c.renvoNativeIntSize &&
			(renvoTypeKindIsScalarValue(kind) || kind == renvoTypePointer || kind == renvoTypeFunc) {
			renvoAsmLoadPrimaryStack(&g.asm, sourceOffset)
			if renvoTypeKindIsScalarValue(kind) {
				renvoAsmNormalizePrimaryForKind(&g.asm, kind)
			}
			renvoAsmStorePrimaryStack(&g.asm, valueOffset)
			return
		}
		if size < renvoBackendValueSlotSize {
			renvoAsmStoreStackImm(&g.asm, valueOffset, 0)
		}
		renvoEmitCopyStackToStack(g, sourceOffset, valueOffset, size)
		return
	}
	renvoAsmLoadSecondaryStack(&g.asm, sourceOffset)
	renvoEmitCopyMemSecondaryToStack(g, valueOffset, size)
}

func renvoEmitTypeMatchJump(g *renvoLinearGen, tagOffset int, typ int, matchLabel int) {
	renvoNonNil(g)
	if renvoResolveType(g.meta, typ).kind != renvoTypeInterface {
		renvoAsmJcmpStackImm(&g.asm, tagOffset, renvoRuntimeTypeTag(g.meta, typ), matchLabel, 0x94)
		return
	}
	iface := renvoResolveType(g.meta, typ)
	renvoNonNil(iface)
	if iface.first >= iface.count && iface.elem != -1 {
		renvoAsmLoadPrimaryStack(&g.asm, tagOffset)
		renvoAsmJnzPrimary(&g.asm, matchLabel)
		return
	}
	// Signature parsing may intern types. Only the types present at entry
	// can be runtime values here; newly interned signature types are not
	// additional candidates for this assertion.
	candidateCount := len(g.meta.types)
	for candidate := 1; candidate < candidateCount; candidate++ {
		tag := renvoRuntimeTypeTag(g.meta, candidate)
		t := &g.meta.types[candidate]
		if tag == 0 || t.kind == renvoTypeNamed && t.first == renvoNamedTypeAlias || renvoResolveType(g.meta, candidate).kind == renvoTypeInterface || !renvoTypeImplementsInterface(g, candidate, typ) {
			continue
		}
		renvoAsmJcmpStackImm(&g.asm, tagOffset, tag, matchLabel, 0x94)
	}
}

func renvoTypeImplementsInterface(g *renvoLinearGen, typ int, interfaceType int) bool {
	renvoNonNil(g)
	meta := g.meta
	iface := renvoResolveType(meta, interfaceType)
	renvoNonNil(iface)
	if iface.elem == -1 {
		for i := 0; i < len(meta.funcs); i++ {
			fn := &meta.funcs[i]
			if fn.receiverType == 0 || !renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "Error") || !renvoMethodReceiverTypeMatches(meta, typ, fn.receiverType) {
				continue
			}
			if renvoResolveType(meta, fn.receiverType).kind == renvoTypePointer && renvoResolveType(meta, typ).kind != renvoTypePointer {
				return false
			}
			return fn.paramCount == 1 && renvoResolveType(meta, fn.resultType).kind == renvoTypeString
		}
		return false
	}
	for required := iface.first; required < iface.count; {
		if !renvoTokIsKind(meta.prog, required, renvoTokIdent) {
			required++
			continue
		}
		end := renvoStatementLineEnd(meta.prog, required, iface.count)
		if !renvoTokCharIs(meta.prog, required+1, '(') {
			embedded := renvoParseType(meta, meta.prog, required, end)
			if embedded.typ == 0 || !renvoTypeImplementsInterface(g, typ, embedded.typ) {
				return false
			}
			required = end
			continue
		}
		var parsed renvoTypeResult
		renvoParseFuncSignatureInto(meta, meta.prog, required+1, end, &parsed)
		if parsed.typ == 0 || parsed.next != end {
			return false
		}
		signature := renvoResolveType(meta, parsed.typ)
		renvoNonNil(signature)
		fnIndex := renvoFindMethodByTypeAndName(g, typ, int(renvoTokStart(meta.prog, required)), int(renvoTokEnd(meta.prog, required)))
		if fnIndex < 0 {
			return false
		}
		fn := &meta.funcs[fnIndex]
		if renvoResolveType(meta, fn.receiverType).kind == renvoTypePointer && renvoResolveType(meta, typ).kind != renvoTypePointer || !renvoTypesEquivalent(meta, fn.resultType, signature.elem) || !renvoFunctionParamsMatchType(meta, fn, signature, 1) {
			return false
		}
		required = end
	}
	return true
}

func renvoEmitSwitchCaseTests(g *renvoLinearGen, stmt *renvoStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, typeSwitch bool, interfaceSwitch bool, matchLabel int) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	colon := renvoFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
	if colon <= clause+1 {
		return false
	}
	i := clause + 1
	for i < colon {
		valueEnd := renvoFindExprBoundary(p, i, colon)
		if valueEnd <= i {
			return false
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := -1
		if !typeSwitch {
			rootIndex = renvoParseExpressionRoot(ep, p, i, valueEnd)
			if rootIndex < 0 {
				return false
			}
		}
		if typeSwitch {
			if renvoBytesEqualText(p.src, int(renvoTokStart(p, i)), int(renvoTokEnd(p, i)), "nil") {
				renvoAsmJcmpStackImm(a, valueOffset, 0, matchLabel, 0x94)
			} else {
				caseType := renvoParseType(g.meta, p, i, valueEnd)
				if caseType.typ == 0 || caseType.next != valueEnd {
					return false
				}
				renvoEmitTypeMatchJump(g, valueOffset, caseType.typ, matchLabel)
			}
		} else if interfaceSwitch {
			caseOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
			if !renvoEmitInterfaceAssignToLocal(g, ep, rootIndex, caseOffset) {
				return false
			}
			if !renvoEmitInterfaceCompareLocals(g, valueOffset, caseOffset, false) {
				return false
			}
			renvoAsmJnzPrimary(a, matchLabel)
		} else if stringSwitch {
			if !renvoEmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			if renvoFixedTarget == 0 && renvoCanKeepSwitchPrimary(g) {
				constant := renvoEvalConstExpr(g, ep, rootIndex)
				fast := renvoEmitSwitchCasePeephole(g, ep, rootIndex, valueOffset, matchLabel, constant.ok, constant.value)
				if fast == 0 {
					return false
				}
				if fast > 0 {
					i = valueEnd
					if renvoTokCharIs(p, i, ',') {
						i++
					}
					continue
				}
			}
			renvoAsmPushStack(a, valueOffset)
			if !renvoEmitIntExpr(g, ep, rootIndex) {
				return false
			}
			renvoAsmPopTertiary(a)
			renvoAsmCmpTertiaryPrimarySet(a, 0x94)
			renvoAsmJnzPrimary(a, matchLabel)
		}
		i = valueEnd
		if renvoTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}
func renvoFindNextSwitchClause(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	depth := 0
	i := start
	for i < end {
		if depth == 0 && (renvoTokIsKind(p, i, renvoTokCase) || renvoTokIsKind(p, i, renvoTokDefault)) {
			return i
		}
		if renvoTokCharIs(p, i, '{') {
			depth++
		} else if renvoTokCharIs(p, i, '}') {
			if depth > 0 {
				depth--
			}
		}
		i++
	}
	return end
}
func renvoFindSwitchClauseColon(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		c := renvoTokSingleChar(p, i)
		if paren == 0 && brack == 0 && brace == 0 && c == ':' {
			return i
		}
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
		} else if c == '[' {
			brack++
		} else if c == ']' {
			brack--
		} else if c == '{' {
			brace++
		} else if c == '}' {
			if brace == 0 {
				return end
			}
			brace--
		}
		i++
	}
	return end
}
func renvoEmitLinearClassicForScoped(g *renvoLinearGen, stmt *renvoStmt, semi1 int) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	semi2 := renvoFindTokenTextInRange(p, semi1+1, stmt.exprEnd, ';')
	if semi2 <= semi1 {
		return false
	}
	loopLocalBase := g.localCount
	initAssign := renvoFindAssignmentToken(p, stmt.exprStart, semi1)
	perIterationLocals := initAssign >= stmt.exprStart && renvoTok2Is(p, initAssign, ':', '=')
	if !renvoEmitLinearSimpleRange(g, stmt.exprStart, semi1) {
		return false
	}
	endLabel := renvoNewControlLabel(g, 0)
	postLabel := renvoNewControlLabel(g, 1)
	g.pendingControl = 0
	startLabel := renvoAsmNewLabel(a)
	renvoPushLoopLabels(g, endLabel, postLabel)
	renvoAsmMarkLabel(a, startLabel)
	renvoMoveCapturedLocals(g, false)
	if semi1+1 < semi2 {
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, semi1+1, semi2)
		if rootIndex < 0 {
			return false
		}
		if !renvoEmitJumpIfFalse(g, ep, rootIndex, endLabel) {
			return false
		}
	}
	if !renvoEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	renvoAsmMarkLabel(a, postLabel)
	if perIterationLocals {
		for localIndex := loopLocalBase; localIndex < g.localCount; localIndex++ {
			renvoRebindCapturedLocal(g, localIndex)
		}
	}
	if !renvoEmitLinearSimpleRange(g, semi2+1, stmt.exprEnd) {
		return false
	}
	renvoAsmJmpMarkLabel(a, startLabel, endLabel)
	renvoPopLoopLabels(g)
	return true
}
func renvoEmitLinearSimpleRange(g *renvoLinearGen, start int, end int) bool {
	renvoNonNil(g)
	p := g.prog
	if start >= end {
		return true
	}
	if renvoEmitLinearIncDec(g, start, end) {
		renvoSyncCapturedStmtTargets(g, &renvoStmt{kind: renvoStmtExpr, startTok: start, endTok: end})
		return true
	}
	assignTok := renvoFindAssignmentToken(p, start, end)
	if assignTok > start {
		kind := renvoStmtAssign
		if renvoTok2Is(p, assignTok, ':', '=') {
			kind = renvoStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if renvoTokIsKind(p, start, renvoTokIdent) {
			nameStart = int(renvoTokStart(p, start))
			nameEnd = int(renvoTokEnd(p, start))
		}
		stmt := renvoStmt{kind: kind, startTok: start, endTok: end, exprStart: assignTok + 1, exprEnd: end, nameStart: nameStart, nameEnd: nameEnd}
		renvoMoveCapturedLocals(g, false)
		if !renvoEmitLinearAssign(g, &stmt) {
			return false
		}
		renvoSyncCapturedStmtTargets(g, &stmt)
		return true
	}
	ep := renvoNewExprParse()
	rootIndex := renvoParseExpressionRoot(ep, p, start, end)
	if rootIndex < 0 {
		return false
	}
	if renvoFixedTarget == 0 {
		root := &ep.exprs[rootIndex]
		if root.kind == renvoExprCall && root.left >= 0 && root.left < len(ep.exprs) {
			callee := &ep.exprs[root.left]
			if callee.kind == renvoExprIdent {
				update := renvoEmitCUpdateIntrinsic(g, ep, rootIndex, root, callee, true)
				if update >= 0 {
					return update != 0
				}
			}
		}
	}
	return renvoEmitIntExpr(g, ep, rootIndex)
}
func renvoEmitLinearIncDec(g *renvoLinearGen, start int, end int) bool {
	renvoNonNil(g)
	a := &g.asm
	p := g.prog
	if start+2 > end {
		return false
	}
	opTok := end - 1
	if !renvoTok2Is(p, opTok, '+', '+') && !renvoTok2Is(p, opTok, '-', '-') {
		return false
	}
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, p, start, opTok)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	inc := renvoTok2Is(p, opTok, '+', '+')
	if root.kind == renvoExprIdent {
		localOffset := renvoFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			renvoClearLocalConstAtOffset(g, localOffset)
			return renvoEmitIncrementLocalWord(g, localOffset, inc)
		}
		globalOffset := renvoFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
			globalType := renvoFindGlobalType(g, root.nameStart, root.nameEnd)
			globalSize := renvoTypeSize(g.meta, globalType)
			if globalSize < g.c.renvoNativeIntSize {
				renvoAsmLoadPrimaryBssSize(a, globalOffset, globalSize)
				renvoAsmPushImm(a, 1)
				renvoAsmPopTertiary(a)
				if inc {
					renvoAsmAddPrimaryTertiary(a)
				} else {
					renvoAsmSubPrimaryTertiary(a)
				}
				renvoAsmStorePrimaryBssSize(a, globalOffset, globalSize)
				return true
			}
		}
		return renvoEmitIncrementGlobalWord(g, globalOffset, inc)
	}
	if root.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, rootIndex) {
			return false
		}
		if inc {
			renvoAsmIncMemSecondary(a)
		} else {
			renvoAsmDecMemSecondary(a)
		}
		return true
	}
	if root.kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, ep, rootIndex) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		if inc {
			renvoAsmIncMemSecondary(a)
		} else {
			renvoAsmDecMemSecondary(a)
		}
		return true
	}
	if root.kind == renvoExprUnary && renvoTokCharIs(p, root.tok, '*') {
		if !renvoEmitIntExpr(g, ep, root.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
		if inc {
			renvoAsmIncMemSecondary(a)
		} else {
			renvoAsmDecMemSecondary(a)
		}
		return true
	}
	return false
}
func renvoEmitJumpIfFalse(g *renvoLinearGen, ep *renvoExprParse, idx int, falseLabel int) bool {
	renvoNonNil(g, ep)
	return renvoEmitJump(g, ep, idx, falseLabel, false)
}
func renvoEmitJump(g *renvoLinearGen, ep *renvoExprParse, idx int, label int, jumpIfTrue bool) bool {
	renvoNonNil(g, ep)
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoFixedTarget == 0 && e.kind == renvoExprCall &&
		e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprIdent {
		inlined := renvoEmitCInlineReturn(g, ep, idx, e, &ep.exprs[e.left], label, jumpIfTrue)
		if inlined >= 0 {
			return inlined != 0
		}
	}
	if e.kind == renvoExprBinary {
		and := renvoTok2Is(p, e.tok, '&', '&')
		or := renvoTok2Is(p, e.tok, '|', '|')
		if (jumpIfTrue && or) || (!jumpIfTrue && and) {
			if !renvoEmitJump(g, ep, e.left, label, jumpIfTrue) {
				return false
			}
			return renvoEmitJump(g, ep, e.right, label, jumpIfTrue)
		}
		if and || or {
			skipLabel := renvoAsmNewLabel(a)
			if !renvoEmitJump(g, ep, e.left, skipLabel, !jumpIfTrue) {
				return false
			}
			if !renvoEmitJump(g, ep, e.right, label, jumpIfTrue) {
				return false
			}
			renvoAsmMarkLabel(a, skipLabel)
			return true
		}
		if renvoStringOrderingExpr(g, ep, e) {
			if !renvoEmitStringOrdering(g, ep, e) {
				return false
			}
			if jumpIfTrue {
				renvoAsmJnzPrimary(a, label)
			} else {
				renvoAsmJzPrimary(a, label)
			}
			return true
		}
		if !renvoBinaryComparesInterface(g, ep, e) {
			if g.c.renvoNativeIntSize == 4 && renvoEmitWideCompareExpr(g, ep, idx) {
				if jumpIfTrue {
					renvoAsmJnzPrimary(a, label)
				} else {
					renvoAsmJzPrimary(a, label)
				}
				return true
			}
			if renvoEmitWordCompareJump(g, ep, e, label, jumpIfTrue) {
				return true
			}
		}
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(p, e.tok, '!') {
		return renvoEmitJump(g, ep, e.left, label, !jumpIfTrue)
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmCmpPrimaryImm8Discard(a, 0)
	if jumpIfTrue {
		renvoAsmJnzLabel(a, label)
	} else {
		renvoAsmJzLabel(a, label)
	}
	return true
}

func renvoIsComparisonChars(c0 byte, c1 byte) bool {
	return (c0 == '=' || c0 == '!') && c1 == '=' || c0 == '<' && c1 != '<' || c0 == '>' && c1 != '>'
}

func renvoEmitUnsignedPrimaryTertiaryCompare(g *renvoLinearGen, c0 byte, c1 byte, opLen int) bool {
	if renvoFixedTarget != 0 && !renvoCanCompareUnsignedWord(g) {
		return false
	}
	renvoNonNil(g)
	if c0 != '<' && c0 != '>' {
		return false
	}
	if opLen == 2 && c1 != '=' {
		return false
	}
	setcc := 0x92
	if c0 == '>' {
		setcc = 0x97
	}
	if opLen == 2 {
		setcc = setcc ^ 4
	}
	renvoAsmCmpTertiaryPrimarySet(&g.asm, setcc)
	return true
}

const (
	renvoInitVisiting     = 1
	renvoInitDone         = 2
	renvoInitFunctionSeen = -1
)

func renvoLinearInitGlobal(g *renvoLinearGen, index int) bool {
	renvoNonNil(g)
	meta := g.meta
	renvoNonNil(meta)
	var s *renvoSymbolInfo
	start := 0
	end := 0
	if index >= 0 {
		s = &meta.globals[index]
		if s.constValueOK != 0 {
			return s.constValueOK == renvoInitDone
		}
		s.constValueOK = renvoInitVisiting
		start = s.initStart
		end = s.initEnd
	} else {
		fn := &meta.funcs[-index-1]
		if fn.literalTok == renvoInitFunctionSeen {
			return true
		}
		fn.literalTok = renvoInitFunctionSeen
		start = fn.bodyStart
		end = fn.bodyEnd
	}
	for tok := start; tok < end; tok++ {
		if !renvoTokIsKind(meta.prog, tok, renvoTokIdent) {
			continue
		}
		nameStart := int(renvoTokStart(meta.prog, tok))
		nameEnd := int(renvoTokEnd(meta.prog, tok))
		dependency := renvoFindMetaGlobalIndex(meta, nameStart, nameEnd, renvoTokVar)
		if dependency >= 0 && !renvoLinearInitGlobal(g, dependency) {
			return false
		}
		fnIndex := renvoFindMetaFunction(meta, nameStart, nameEnd)
		if fnIndex >= 0 && !renvoLinearInitGlobal(g, -fnIndex-1) {
			return false
		}
	}
	if index < 0 {
		return true
	}
	off := s.iotaValue
	var foreign *renvoForeignProgram
	for item := g.prog.foreign; item != nil; item = item.next {
		if s.nameStart == item.global {
			foreign = item
			break
		}
	}
	if foreign != nil {
		for len(g.asm.data)&15 != 0 {
			g.asm.data = append(g.asm.data, 0)
		}
		dataOffset := len(g.asm.data)
		g.asm.data = append(g.asm.data, foreign.artifact...)
		if foreign.entryOffset < 0 {
			typ := renvoResolveType(meta, s.typ)
			if typ.kind != renvoTypeSlice || renvoResolveType(meta, typ.elem).kind != renvoTypeByte {
				return false
			}
			renvoAsmPrimaryDataAddr(&g.asm, dataOffset)
			renvoAsmSecondaryImm(&g.asm, len(foreign.artifact))
			renvoAsmCopySecondaryToTertiary(&g.asm)
			renvoAsmStoreSliceBss(&g.asm, off)
		} else {
			if !renvoTypeIsNativeInt(meta, s.typ) {
				return false
			}
			renvoAsmPrimaryDataAddr(&g.asm, dataOffset+foreign.entryOffset)
			renvoAsmStorePrimaryBss(&g.asm, off)
		}
		s.constValueOK = renvoInitDone
		return true
	}
	skipInitializer := -1
	if renvoFixedTarget == 0 {
		if index < len(g.replRestoreOffsets) && g.replRestoreOffsets[index] >= 0 {
			skipInitializer = renvoAsmNewLabel(&g.asm)
			renvoAsmLoadPrimaryBss(&g.asm, g.replRestoreOffsets[index])
			renvoAsmJnzPrimary(&g.asm, skipInitializer)
		}
	}
	if renvoTypeIsNativeInt(meta, s.typ) && renvoBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "renvoDefaultTarget") {
		renvoAsmPrimaryImm(&g.asm, g.c.renvoTarget)
		renvoAsmStorePrimaryBss(&g.asm, off)
	} else if s.initStart < s.initEnd {
		localBase := g.localCount
		stackBase := g.stackUsed
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, g.prog, s.initStart, s.initEnd)
		if rootIndex < 0 {
			return false
		}
		tempOffset := renvoAddUnnamedLocal(g, s.typ)
		if !renvoEmitExprToLocal(g, ep, rootIndex, tempOffset) {
			return false
		}
		renvoEmitCopyStackToBss(g, tempOffset, off, renvoTypeCopySize(meta, s.typ))
		g.localCount = localBase
		g.stackUsed = stackBase
	} else if renvoTypeIsSlice(meta, s.typ) {
		renvoEmitInitEmptySliceBss(g, s.typ, off)
	}
	if renvoFixedTarget == 0 {
		renvoAsmMarkLabel(&g.asm, skipInitializer)
	}
	s.constValueOK = renvoInitDone
	return true
}

const renvoReplValuePrefix = "renvo_repl_value_"
const renvoReplStoragePrefix = "renvo_repl_storage_"

func renvoReplSlotID(src []byte, start int, end int) int {
	prefix := renvoReplValuePrefix
	kind := 1
	if end-start > len(renvoReplStoragePrefix) &&
		renvoBytesEqualText(src, start, start+len(renvoReplStoragePrefix), renvoReplStoragePrefix) {
		prefix = renvoReplStoragePrefix
		kind = 0
	} else if end-start <= len(prefix) ||
		!renvoBytesEqualText(src, start, start+len(prefix), prefix) {
		return -1
	}
	id := 0
	for at := start + len(prefix); at < end; at++ {
		ch := renvo_runtime_UnsafeByteAt(src, at)
		if ch < '0' || ch > '9' || id > 100000000 {
			return -1
		}
		id = id*10 + int(ch-'0')
	}
	return id*2 + kind
}

func renvoLinearInitGlobals(g *renvoLinearGen) bool {
	renvoNonNil(g)
	g.localCount = 0
	g.stackUsed = 0
	g.stackPeak = 0
	framePatch := renvoEmitGlobalInitFrameStart(g)
	meta := g.meta
	if renvoFixedTarget == 0 {
		renvoLinearPrepareReplGlobals(g)
	}
	// Allocate every global before emitting any initializer. Go permits an
	// initializer to depend on a variable declared later in the file.
	for i := 0; i < len(meta.globals); i++ {
		s := &meta.globals[i]
		if s.kind != renvoTokVar {
			continue
		}
		off := g.asm.bssSize
		s.iotaValue = off
		g.globals = append(g.globals, renvoGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		size := renvoTypeCopySize(meta, s.typ)
		g.asm.bssSize += renvoAlignTo8(size)
		if renvoFixedTarget == 0 {
			renvoLinearAddReplGlobal(g, i, off, size)
		}
	}
	for i := 0; i < len(meta.globals); i++ {
		if meta.globals[i].kind == renvoTokVar && !renvoLinearInitGlobal(g, i) {
			return false
		}
	}
	renvoEmitGlobalInitFrameEnd(g, framePatch)
	return true
}

func renvoEmitInitEmptySliceBss(g *renvoLinearGen, sliceType int, off int) {
	renvoNonNil(g)
	a := &g.asm
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	elemSize := renvoTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	renvoAsmPrimaryBssAddr(a, backingOff)
	renvoAsmStorePrimaryBss(a, off)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryBss(a, off+8)
	renvoAsmPrimaryImm(a, backingSize/elemSize)
	renvoAsmStorePrimaryBss(a, off+16)
}

func renvoEmitPointerAssignment(g *renvoLinearGen, left *renvoExprParse, pointerIndex int, right *renvoExprParse, rightIndex int, targetType int, assignTok int) bool {
	a := &g.asm
	kind := renvoResolveType(g.meta, targetType).kind
	directAddress := -1
	if renvoFixedTarget == 0 {
		directAddress = renvoEmitIndexedPointerAddressPeephole(g, left, pointerIndex)
	}
	if directAddress == 0 || directAddress < 0 && !renvoEmitIntExpr(g, left, pointerIndex) {
		return false
	}
	renvoEmitRuntimeNonNilPrimary(g)
	if renvoTokCharIs(g.prog, assignTok, '=') {
		size := renvoNativeScalarStorageSize(g.c.renvoNativeIntSize, kind)
		if renvoFixedTarget == 0 && renvoCanDirectScalarStore(g, size) &&
			(renvoTypeKindIsScalarValue(kind) || kind == renvoTypePointer || kind == renvoTypeFunc) {
			simpleRight := renvoConstExprSideEffectFree(g, right, rightIndex) || renvoScalarPreservesSecondary(g, right, rightIndex)
			if simpleRight {
				renvoAsmCopyPrimaryToSecondary(a)
			} else {
				renvoAsmPushPrimary(a)
			}
			if !renvoEmitScalarExprForKind(g, right, rightIndex, kind) {
				return false
			}
			if !simpleRight {
				renvoAsmPopSecondary(a)
			}
			renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, size)
			return true
		}
		addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStorePrimaryStack(a, addrOffset)
		if renvoFixedTarget == 0 {
			if g.c.objectFile && kind == renvoTypeFunc {
				if !renvoEmitScalarExprForKind(g, right, rightIndex, renvoTypeFunc) {
					return false
				}
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, g.c.renvoNativeIntSize)
				return true
			}
		}
		if !renvoEmitTypedExprToSavedMem(g, right, rightIndex, targetType, addrOffset) {
			return false
		}
		return true
	}
	if renvoFixedTarget == 0 && renvoEmitCompoundPointerMemoryPeephole(g, right, rightIndex, kind, assignTok) {
		return true
	}
	if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(kind) {
		addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		valueOffset := renvoAddUnnamedLocal(g, targetType)
		renvoAsmStorePrimaryStack(a, addrOffset)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoEmitCopyMemSecondaryToStack(g, valueOffset, renvoTypeSize(g.meta, targetType))
		if !renvoEmitWideCompoundLocal(g, right, rightIndex, valueOffset, kind, assignTok) {
			return false
		}
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoEmitCopyStackToMemSecondary(g, valueOffset, 0, renvoTypeSize(g.meta, targetType))
		return true
	}
	renvoAsmPushPrimary(a)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(g.c.renvoNativeIntSize, kind))
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, right, rightIndex, kind) {
		return false
	}
	renvoAsmPopTertiary(a)
	if !renvoEmitTypedPrimaryTertiaryOp(g, assignTok, kind) {
		return false
	}
	renvoAsmNormalizePrimaryForKind(a, kind)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(g.c.renvoNativeIntSize, kind))
	return true
}

// renvoEmitLinearCompoundLValue handles compound assignments whose destination
// needs an address. Zero means the destination was not one of these forms.
func renvoEmitLinearCompoundLValue(g *renvoLinearGen, stmt *renvoStmt, assignTok int) int {
	p := g.prog
	meta := g.meta
	a := &g.asm
	lhs := renvoNewExprParse()
	renvoNonNil(lhs)
	if !renvoParseExpressionOK(lhs, p, stmt.startTok, assignTok) {
		return 0
	}
	lhsIndex := len(lhs.exprs) - 1
	lhsRoot := &lhs.exprs[lhsIndex]
	if lhsRoot.kind == renvoExprIndex {
		elemTypeIndex := renvoInferParsedExprType(g, lhs, lhsIndex)
		elemType := renvoResolveType(meta, elemTypeIndex)
		renvoNonNil(elemType)
		if !renvoTypeKindIsScalarInt(elemType.kind) {
			return -1
		}
		elemSize := renvoScalarKindSize(g.c.renvoNativeIntSize, elemType.kind)
		addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		if !renvoEmitIndexAddressPrimary(g, lhs, lhsIndex) {
			return -1
		}
		renvoAsmStorePrimaryStack(a, addrOffset)
		if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(elemType.kind) {
			valueOffset := renvoAddUnnamedLocal(g, elemTypeIndex)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoEmitCopyMemSecondaryToStack(g, valueOffset, renvoTypeSize(meta, elemTypeIndex))
			rhs := renvoNewExprParse()
			renvoNonNil(rhs)
			rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
			if rhsIndex < 0 || !renvoEmitWideCompoundLocal(g, rhs, rhsIndex, valueOffset, elemType.kind, assignTok) {
				return -1
			}
			renvoAsmLoadSecondaryStack(a, addrOffset)
			renvoEmitCopyStackToMemSecondary(g, valueOffset, 0, renvoTypeSize(meta, elemTypeIndex))
			return 1
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, elemSize)
		renvoAsmPushPrimary(a)
		rhs := renvoNewExprParse()
		renvoNonNil(rhs)
		rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
		if rhsIndex < 0 || !renvoEmitIntExpr(g, rhs, rhsIndex) {
			return -1
		}
		renvoAsmPopTertiary(a)
		if !renvoEmitPrimaryTertiaryOp(g, assignTok) {
			return -1
		}
		renvoAsmNormalizePrimaryForKind(a, elemType.kind)
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
		return 1
	}
	if lhsRoot.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, lhs, lhsIndex) {
			return -1
		}
		lhsResolved := renvoResolveType(meta, renvoInferParsedExprType(g, lhs, lhsIndex))
		renvoNonNil(lhsResolved)
		addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreSecondaryStack(a, addrOffset)
		rhs := renvoNewExprParse()
		renvoNonNil(rhs)
		if !renvoParseExpressionOK(rhs, p, assignTok+1, stmt.endTok) {
			return -1
		}
		rhsIndex := len(rhs.exprs) - 1
		if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(lhsResolved.kind) {
			valueType := renvoInferParsedExprType(g, lhs, lhsIndex)
			valueOffset := renvoAddUnnamedLocal(g, valueType)
			renvoAsmLoadSecondaryStack(a, addrOffset)
			renvoEmitCopyMemSecondaryToStack(g, valueOffset, renvoTypeSize(meta, valueType))
			if !renvoEmitWideCompoundLocal(g, rhs, rhsIndex, valueOffset, lhsResolved.kind, assignTok) {
				return -1
			}
			renvoAsmLoadSecondaryStack(a, addrOffset)
			renvoEmitCopyStackToMemSecondary(g, valueOffset, 0, renvoTypeSize(meta, valueType))
			return 1
		}
		if lhsResolved.kind == renvoTypeString {
			if !renvoTok2Is(p, assignTok, '+', '=') || lhs.exprs[lhsRoot.left].kind != renvoExprIdent || !renvoEmitStringConcatPairValueRegs(g, lhs, lhsIndex, rhs, rhsIndex) {
				return -1
			}
			renvoAsmPushStringRegs(a)
			renvoAsmLoadSecondaryStack(a, addrOffset)
			renvoAsmPopStoreStringMemSecondary(a, 0)
			return 1
		}
		lhsSize := renvoScalarKindSize(g.c.renvoNativeIntSize, lhsResolved.kind)
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, lhsSize)
		renvoAsmPushPrimary(a)
		if !renvoEmitIntExpr(g, rhs, rhsIndex) {
			return -1
		}
		renvoAsmPopTertiary(a)
		if !renvoEmitTypedPrimaryTertiaryOp(g, assignTok, lhsResolved.kind) {
			return -1
		}
		renvoAsmNormalizePrimaryForKind(a, lhsResolved.kind)
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
		return 1
	}
	if lhsRoot.kind == renvoExprUnary && renvoTokCharIs(p, lhsRoot.tok, '*') {
		lhsType := renvoInferParsedExprType(g, lhs, lhsIndex)
		rhs := renvoNewExprParse()
		renvoNonNil(rhs)
		rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
		if rhsIndex >= 0 && renvoEmitPointerAssignment(g, lhs, lhsRoot.left, rhs, rhsIndex, lhsType, assignTok) {
			return 1
		}
		return -1
	}
	return 0
}

func renvoEmitLinearAssign(g *renvoLinearGen, stmt *renvoStmt) bool {
	// A standalone ConstDecl is a one-specification group. Limit its iota
	// context to emission of that declaration, including all early returns.
	if renvoTokIsKind(g.prog, stmt.startTok, renvoTokConst) && !renvoTokCharIs(g.prog, stmt.startTok+1, '(') {
		oldIota, oldValid := g.constEvalIota, g.constEvalIotaValid
		g.constEvalIota, g.constEvalIotaValid = 0, 1
		ok := renvoEmitLinearAssignCore(g, stmt)
		g.constEvalIota, g.constEvalIotaValid = oldIota, oldValid
		return ok
	}
	return renvoEmitLinearAssignCore(g, stmt)
}

func renvoEmitLinearAssignCore(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	a := &g.asm
	tokenData := p.toks.data
	startBase := stmt.startTok * renvoTokenStride
	startKind := int(tokenData[startBase]) & 255
	if startKind == renvoTokVar && renvoTokCharIs(p, stmt.startTok+1, '(') {
		return renvoEmitLinearRangeMode(g, stmt.startTok+2, stmt.endTok-1, true)
	}
	if startKind == renvoTokConst && renvoTokCharIs(p, stmt.startTok+1, '(') {
		g.constEvalIota = 0
		g.constEvalIotaValid = 1
		ok := renvoEmitLinearRange(g, stmt.startTok+2, stmt.endTok-1)
		g.constEvalIotaValid = 0
		return ok
	}
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	nextBase := startBase + renvoTokenStride
	if (startKind == renvoTokVar || startKind == renvoTokConst) && int(tokenData[nextBase])&255 == renvoTokIdent {
		nameStart = renvoTokStart(p, stmt.startTok+1)
		nameEnd = renvoTokEnd(p, stmt.startTok+1)
	} else if startKind == renvoTokIdent {
		nameStart = renvoTokStart(p, stmt.startTok)
		nameEnd = renvoTokEnd(p, stmt.startTok)
	}
	assignTok := renvoFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	compoundAssign := assignTok >= 0 && assignTok < renvoTokCount(p) && renvoTokIsCompoundAssignment(p, assignTok)
	groupedVar := renvoEmitGroupedTypedVarDecl(g, stmt, assignTok)
	if groupedVar != 0 {
		return groupedVar > 0
	}
	if assignTok > stmt.startTok {
		lhsStart := stmt.startTok
		if stmt.kind == renvoStmtVar && startKind == renvoTokVar {
			lhsStart++
		}
		shadowsOuter := stmt.kind == renvoStmtShort && renvoFindLocalIndexInCurrentScope(g, nameStart, nameEnd) < 0 &&
			(renvoFindLocalIndex(g, nameStart, nameEnd) >= 0 || renvoFindGlobalType(g, nameStart, nameEnd) != 0)
		if (renvoHasTopLevelComma(p, lhsStart, assignTok) || renvoHasTopLevelComma(p, assignTok+1, stmt.endTok) || shadowsOuter) && renvoEmitMultiAssign(g, stmt, assignTok) {
			return true
		}
	}
	if assignTok > stmt.startTok && compoundAssign {
		result := renvoEmitLinearCompoundLValue(g, stmt, assignTok)
		if result != 0 {
			return result > 0
		}
	}
	if assignTok > stmt.startTok && renvoTokCharIs(p, assignTok, '=') && (startKind != renvoTokIdent || assignTok != stmt.startTok+1) {
		lhs := renvoNewExprParse()
		renvoNonNil(lhs)
		if renvoParseExpressionOK(lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == renvoExprIdent {
				nameStart = lhsRoot.nameStart
				nameEnd = lhsRoot.nameEnd
			}
			lhsType := renvoInferParsedExprType(g, lhs, lhsIndex)
			if lhsRoot.kind == renvoExprIndex {
				elemTypeIndex := lhsType
				elemType := renvoResolveType(meta, elemTypeIndex)
				renvoNonNil(elemType)
				addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
				if !renvoEmitIndexAddressPrimary(g, lhs, lhsIndex) {
					return false
				}
				renvoAsmStorePrimaryStack(a, addrOffset)
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if renvoTypeKindUsesMemory(elemType.kind) || g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(elemType.kind) {
					return renvoEmitTypedExprToSavedMem(g, rhs, rhsIndex, elemTypeIndex, addrOffset)
				}
				if !renvoEmitScalarExprForKind(g, rhs, rhsIndex, elemType.kind) {
					return false
				}
				renvoAsmNormalizePrimaryForKind(a, elemType.kind)
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(g.c.renvoNativeIntSize, elemType.kind))
				return true
			}
			lhsResolved := renvoResolveType(meta, lhsType)
			renvoNonNil(lhsResolved)
			if lhsRoot.kind == renvoExprUnary && renvoTokCharIs(p, lhsRoot.tok, '*') {
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				rhsRoot := &rhs.exprs[rhsIndex]
				if lhsResolved.kind == renvoTypeSlice && rhsRoot.kind == renvoExprCall && rhsRoot.argCount >= 2 && renvoExprIdentCode(p, rhs, rhsRoot.left) == renvoIdentAppend {
					return renvoEmitAppendAssignGeneral(g, stmt, rhs, assignTok)
				}
				return renvoEmitPointerAssignment(g, lhs, lhsRoot.left, rhs, rhsIndex, lhsType, assignTok)
			}
			if lhsRoot.kind == renvoExprSelector && (renvoTypeKindUsesMemory(lhsResolved.kind) || g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(lhsResolved.kind)) {
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				rhsRoot := &rhs.exprs[rhsIndex]
				if lhsResolved.kind == renvoTypeSlice && rhsRoot.kind == renvoExprCall && rhsRoot.argCount >= 2 && renvoExprIdentCode(p, rhs, rhsRoot.left) == renvoIdentAppend {
					return renvoEmitAppendAssignGeneral(g, stmt, rhs, assignTok)
				}
				if !renvoEmitSelectorAddressSecondary(g, lhs, lhsIndex) {
					return false
				}
				addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
				renvoAsmStoreSecondaryStack(a, addrOffset)
				return renvoEmitTypedExprToSavedMem(g, rhs, rhsIndex, lhsType, addrOffset)
			}
			if lhsRoot.kind == renvoExprSelector {
				if !renvoEmitSelectorAddressSecondary(g, lhs, lhsIndex) {
					return false
				}
				renvoAsmPushSecondary(a)
				lhsResolved := renvoResolveType(meta, lhsType)
				renvoNonNil(lhsResolved)
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if !renvoEmitScalarExprForKind(g, rhs, rhsIndex, lhsResolved.kind) {
					return false
				}
				renvoAsmNormalizePrimaryForKind(a, lhsResolved.kind)
				renvoAsmPopSecondary(a)
				lhsSize := renvoScalarKindSize(g.c.renvoNativeIntSize, lhsResolved.kind)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		return false
	}
	if nameEnd == nameStart+1 && renvo_runtime_UnsafeByteAt(p.src, nameStart) == '_' {
		if assignTok <= stmt.startTok || !renvoTokCharIs(p, assignTok, '=') {
			return true
		}
		// Canonical frontend units represent a value-less expression statement as
		// a blank assignment. Preserve the print builtin's statement semantics
		// when that representation reaches the backend.
		discardStmt := *stmt
		discardStmt.exprStart = assignTok + 1
		discardStmt.exprEnd = stmt.endTok
		if renvoEmitLinearPrintStmt(g, &discardStmt) {
			return true
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, assignTok+1, stmt.endTok)
		if rootIndex < 0 {
			return false
		}
		discardType := renvoInferParsedExprType(g, ep, rootIndex)
		if discardType != 0 {
			discardOffset := renvoAddUnnamedLocal(g, discardType)
			if renvoEmitTypedAssign(g, ep, rootIndex, discardOffset) {
				return true
			}
		}
		return renvoEmitIntExpr(g, ep, rootIndex)
	}
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	if assignTok > stmt.startTok {
		if !renvoParseExpressionOK(ep, p, assignTok+1, stmt.endTok) {
			return false
		}
	}
	declaresLocal := stmt.kind == renvoStmtVar || startKind == renvoTokVar || stmt.kind == renvoStmtShort
	offset := renvoFindLocalOffset(g, nameStart, nameEnd)
	if declaresLocal {
		offset = -1
	}
	globalOffset := -1
	fieldStackOffset := -1
	fieldType := 0
	if startKind == renvoTokIdent && renvoTokSingleChar(p, stmt.startTok+1) == '.' && renvoTokIsKind(p, stmt.startTok+2, renvoTokIdent) {
		localIndex := renvoFindLocalIndex(g, renvoTokStart(p, stmt.startTok), renvoTokEnd(p, stmt.startTok))
		if localIndex < 0 {
			return false
		}
		fieldNameStart := renvoTokStart(p, stmt.startTok+2)
		fieldNameEnd := renvoTokEnd(p, stmt.startTok+2)
		fieldOffset := renvoStructFieldOffset(g, g.locals[localIndex].typ, fieldNameStart, fieldNameEnd)
		if fieldOffset < 0 {
			return false
		}
		fieldType = renvoStructFieldType(g, g.locals[localIndex].typ, fieldNameStart, fieldNameEnd)
		if fieldType == 0 {
			return false
		}
		fieldStackOffset = g.locals[localIndex].offset - fieldOffset
		offset = fieldStackOffset
	}
	if offset < 0 {
		if stmt.kind == renvoStmtAssign && startKind != renvoTokVar {
			globalOffset = renvoFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := renvoTypeInt
			if stmt.kind == renvoStmtVar || startKind == renvoTokVar {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				typeStart := stmt.startTok + 2
				if startKind == renvoTokIdent {
					typeStart--
				}
				if typeStart < typeEnd {
					typeResult := renvoParseScopedType(g, meta, g.prog, typeStart, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				} else if assignTok > stmt.startTok {
					inferredType := renvoInferParsedExprType(g, ep, len(ep.exprs)-1)
					if inferredType != 0 {
						localType = inferredType
					}
				}
			}
			if stmt.kind == renvoStmtShort {
				inferredType := renvoInferParsedExprType(g, ep, len(ep.exprs)-1)
				if inferredType != 0 {
					localType = inferredType
				}
			}
			if assignTok > stmt.startTok && !renvoProgramUsesC11Semantics(p) &&
				(g.constEvalIotaValid != 0 || startKind == renvoTokConst || renvoFindLocalIndex(g, nameStart, nameEnd) >= 0 ||
					renvoFindMetaGlobalIndex(meta, nameStart, nameEnd, renvoTokVar) >= 0 || renvoFindMetaGlobalIndex(meta, nameStart, nameEnd, renvoTokConst) >= 0) {
				// A Go declaration enters scope after its initializer. Preserve any
				// outer binding until the value has been completely evaluated.
				value := renvoEvalConstExpr(g, ep, len(ep.exprs)-1)
				temp := renvoAddUnnamedLocal(g, localType)
				if !renvoEmitExprToLocal(g, ep, len(ep.exprs)-1, temp) {
					return false
				}
				offset = renvoAddTypedLocal(g, nameStart, nameEnd, localType)
				renvoEmitCopyStackToStack(g, temp, offset, renvoTypeCopySize(meta, localType))
				if (g.constEvalIotaValid != 0 || startKind == renvoTokConst) && value.ok {
					renvoSetLocalConstAtOffset(g, offset, value.value, renvoResolveType(meta, localType).kind)
				}
				return true
			}
			offset = renvoAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			renvoAsmPrimaryImm(a, 0)
			if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
				globalType := renvoFindGlobalType(g, nameStart, nameEnd)
				renvoAsmStorePrimaryBssSize(a, globalOffset, renvoTypeSize(meta, globalType))
			} else {
				renvoAsmStorePrimaryBss(a, globalOffset)
			}
		} else if renvoFixedTarget == 0 && g.c.objectFile && renvoProgramUsesC11Semantics(g.prog) && declaresLocal && fieldStackOffset < 0 {
			// An automatic C object without an initializer has an indeterminate
			// value. The C frontend represents it as a Go var declaration, whose
			// normal zeroing would both invent semantics and emit dead stores.
			renvoClearLocalConstAtOffset(g, offset)
			renvoClearLocalFlowConstAtOffset(g, offset)
		} else {
			renvoZeroLocalAtOffset(g, offset)
			localType := renvoLocalTypeAtOffset(g, offset)
			if renvoFixedTarget == 0 {
				if declaresLocal && fieldStackOffset < 0 && renvoLocalFlowConstTrackable(g, localType, nameStart, nameEnd) {
					renvoSetLocalFlowConstAtOffset(g, offset, 0, renvoResolveType(g.meta, localType).kind)
				} else {
					renvoClearLocalFlowConstAtOffset(g, offset)
				}
			}
			if declaresLocal && fieldStackOffset < 0 && renvoLocalConstTrackable(g, localType, nameStart, nameEnd, stmt.endTok) {
				renvoSetLocalConstAtOffset(g, offset, 0, renvoResolveType(g.meta, localType).kind)
			} else {
				renvoClearLocalConstAtOffset(g, offset)
			}
		}
		return true
	}
	rootIndex := len(ep.exprs) - 1
	targetType := renvoTypeInt
	if globalOffset >= 0 {
		targetType = renvoFindGlobalType(g, nameStart, nameEnd)
	} else if fieldStackOffset >= 0 {
		targetType = fieldType
	} else {
		targetType = renvoLocalTypeAtOffset(g, offset)
	}
	targetResolved := renvoResolveType(meta, targetType)
	renvoNonNil(targetResolved)
	trackSliceArena := globalOffset < 0 && fieldStackOffset < 0 && targetResolved.kind == renvoTypeSlice
	sliceArena := 0
	if trackSliceArena && renvoReturnedSliceCanReuseDescriptor(g, ep, rootIndex) {
		sliceArena = 1
	}
	// Declaration facts are function-wide because trackability proves that no
	// later write exists. A fact learned from a subsequent assignment is
	// path-sensitive and belongs to flowConst instead; otherwise an assignment
	// inside a branch or loop can incorrectly freeze the local everywhere.
	dominates := declaresLocal
	if renvoFixedTarget == 0 && !dominates {
		dominates = renvoTopLevelAssignmentDominates(g, stmt.startTok)
	}
	trackLocalConst := globalOffset < 0 && fieldStackOffset < 0 && dominates &&
		renvoLocalConstTrackable(g, targetType, nameStart, nameEnd, stmt.endTok)
	localConst := renvoConstResult{}
	if trackLocalConst {
		localConst = renvoEvalConstExpr(g, ep, rootIndex)
	}
	trackFlowConst := false
	flowConst := renvoConstResult{}
	if renvoFixedTarget == 0 {
		trackFlowConst = globalOffset < 0 && fieldStackOffset < 0 && renvoLocalFlowConstTrackable(g, targetType, nameStart, nameEnd)
		if trackFlowConst {
			oldFlow := g.constEvalFlow
			g.constEvalFlow = true
			flowConst = renvoEvalConstExpr(g, ep, rootIndex)
			g.constEvalFlow = oldFlow
		}
	}
	if globalOffset < 0 && fieldStackOffset < 0 && !declaresLocal {
		renvoClearLocalConstAtOffset(g, offset)
		if renvoFixedTarget == 0 {
			renvoClearLocalFlowConstAtOffset(g, offset)
		}
	}
	// Append assignment materializes its source itself. Pre-initializing a
	// short declaration here would evaluate a source call twice.
	if renvoEmitAppendAssignGeneral(g, stmt, ep, assignTok) {
		if globalOffset < 0 && fieldStackOffset < 0 {
			renvoClearLocalConstAtOffset(g, offset)
			if sliceArena != 0 {
				renvoSetLocalConstAtOffset(g, offset, 0, targetResolved.kind)
			}
		}
		return true
	}
	if renvoFixedTarget == 0 && !compoundAssign && globalOffset < 0 && fieldStackOffset < 0 &&
		renvoTypeKindIsScalarInt(targetResolved.kind) && renvoTypeSize(meta, targetType) == 4 &&
		renvoEmitSelfBinaryLocalAssignPeephole(g, ep, rootIndex, offset) {
		return true
	}
	if compoundAssign {
		compoundZero := renvoConstResult{}
		if renvoTok2Is(p, assignTok, '&', '=') && globalOffset < 0 && fieldStackOffset < 0 &&
			renvoLocalConstTrackable(g, targetType, nameStart, nameEnd, stmt.endTok) {
			value := renvoEvalConstExpr(g, ep, rootIndex)
			if value.ok && value.value == 0 {
				compoundZero.ok = true
			}
		}
		if targetResolved.kind == renvoTypeString && renvoTok2Is(p, assignTok, '+', '=') {
			left := renvoNewExprParse()
			renvoNonNil(left)
			leftIndex := renvoParseExpressionRoot(left, p, stmt.startTok, assignTok)
			if leftIndex < 0 || !renvoEmitStringConcatPairValueRegs(g, left, leftIndex, ep, rootIndex) {
				return false
			}
			if globalOffset >= 0 {
				renvoAsmStoreStringBss(a, globalOffset)
			} else {
				renvoAsmStorePrimarySecondaryStack(a, offset, offset-8)
			}
			return true
		}
		if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(targetResolved.kind) {
			valueOffset := offset
			if globalOffset >= 0 {
				valueOffset = renvoAddUnnamedLocal(g, targetType)
				for at := 0; at < renvoTypeSize(meta, targetType); at += g.c.renvoNativeIntSize {
					renvoAsmCopyBssToStackSlot(a, globalOffset+at, valueOffset-at)
				}
			}
			if !renvoEmitWideCompoundLocal(g, ep, rootIndex, valueOffset, targetResolved.kind, assignTok) {
				return false
			}
			if globalOffset >= 0 {
				renvoEmitCopyStackToBss(g, valueOffset, globalOffset, renvoTypeSize(meta, targetType))
			} else if fieldStackOffset < 0 {
				renvoClearLocalConstAtOffset(g, offset)
			}
			return true
		}
		if globalOffset < 0 && fieldStackOffset < 0 && renvoTypeKindIsScalarInt(targetResolved.kind) {
			var operation byte
			// Only two-character assignments map to these binary operators;
			// shifts and &^= must retain their full operator semantics.
			operatorStart := renvoTokStart(p, assignTok)
			if renvoTokEnd(p, assignTok)-operatorStart == 2 && renvo_runtime_UnsafeByteAt(p.src, operatorStart+1) == '=' {
				operation = renvo_runtime_UnsafeByteAt(p.src, operatorStart)
			}
			fast := renvoEmitCompoundLocalAssignPeephole(g, ep, rootIndex, offset, assignTok, operation, targetResolved.kind, renvoTypeSize(meta, targetType))
			if fast == 0 {
				return false
			}
			if fast > 0 {
				if compoundZero.ok {
					if renvoFixedTarget == 0 {
						renvoSetLocalFlowConstAtOffset(g, offset, 0, targetResolved.kind)
					}
				} else {
					renvoClearLocalConstAtOffset(g, offset)
					if renvoFixedTarget == 0 {
						renvoClearLocalFlowConstAtOffset(g, offset)
					}
				}
				return true
			}
		}
		if globalOffset >= 0 {
			if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
				renvoAsmLoadPrimaryBssSize(a, globalOffset, renvoTypeSize(meta, targetType))
			} else {
				renvoAsmLoadPrimaryBss(a, globalOffset)
			}
		} else {
			renvoAsmLoadPrimaryStack(a, offset)
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, rootIndex, targetResolved.kind) {
			return false
		}
		renvoAsmPopTertiary(a)
		if !renvoEmitTypedPrimaryTertiaryOp(g, assignTok, targetResolved.kind) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(a, targetResolved.kind)
		if globalOffset >= 0 {
			if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
				renvoAsmStorePrimaryBssSize(a, globalOffset, renvoTypeSize(meta, targetType))
			} else {
				renvoAsmStorePrimaryBss(a, globalOffset)
			}
		} else {
			renvoAsmStorePrimaryStack(a, offset)
			if fieldStackOffset < 0 {
				if compoundZero.ok {
					if renvoFixedTarget == 0 {
						renvoSetLocalFlowConstAtOffset(g, offset, 0, targetResolved.kind)
					}
				} else {
					renvoClearLocalConstAtOffset(g, offset)
					if renvoFixedTarget == 0 {
						renvoClearLocalFlowConstAtOffset(g, offset)
					}
				}
			}
		}
		return true
	}
	if globalOffset >= 0 && renvoTypeIsString(meta, targetType) {
		if !renvoEmitStringValueRegs(g, ep, rootIndex) {
			return false
		}
		renvoAsmStoreStringBss(a, globalOffset)
		return true
	}
	if globalOffset >= 0 && renvoTypeIsSlice(meta, targetType) {
		if !renvoEmitSliceValueRegs(g, ep, rootIndex) {
			return false
		}
		renvoAsmStoreSliceBss(a, globalOffset)
		return true
	}
	if globalOffset >= 0 && targetResolved.kind == renvoTypeComplex64 {
		if !renvoEmitComplexValueRegsForKind(g, ep, rootIndex, targetResolved.kind) {
			return false
		}
		if g.c.renvoNativeIntSize == 4 && renvoUsesStackIEEEFloat(&g.asm) {
			renvoAsmStorePrimaryBssSize(a, globalOffset, 4)
			renvoAsmCopySecondaryToPrimary(a)
			renvoAsmStorePrimaryBssSize(a, globalOffset+4, 4)
			return true
		}
		renvoPackComplex64RegsPrimary(g)
		renvoAsmStorePrimaryBssSize(a, globalOffset, 8)
		return true
	}
	if globalOffset >= 0 && (renvoTypeIsStruct(meta, targetType) || targetResolved.kind == renvoTypeInterface || renvoTypeKindIsComplex(targetResolved.kind) || g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(targetResolved.kind)) {
		tempOffset := renvoAddUnnamedLocal(g, targetType)
		if !renvoEmitTypedAssign(g, ep, rootIndex, tempOffset) {
			return false
		}
		size := renvoTypeSize(meta, targetType)
		renvoEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if globalOffset < 0 && renvoEmitTypedAssign(g, ep, rootIndex, offset) {
		if fieldStackOffset < 0 {
			if renvoFixedTarget == 0 {
				if trackFlowConst && flowConst.ok {
					renvoSetLocalFlowConstAtOffset(g, offset, flowConst.value, targetResolved.kind)
				} else {
					renvoClearLocalFlowConstAtOffset(g, offset)
				}
			}
			if trackLocalConst && localConst.ok {
				renvoSetLocalConstAtOffset(g, offset, localConst.value, targetResolved.kind)
			} else {
				renvoClearLocalConstAtOffset(g, offset)
			}
			if sliceArena != 0 {
				renvoSetLocalConstAtOffset(g, offset, 0, targetResolved.kind)
			}
		}
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, rootIndex, targetResolved.kind) {
		return false
	}
	if globalOffset >= 0 {
		if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
			renvoAsmStorePrimaryBssSize(a, globalOffset, renvoTypeSize(meta, targetType))
		} else {
			renvoAsmStorePrimaryBss(a, globalOffset)
		}
	} else {
		renvoAsmStorePrimaryStack(a, offset)
		if fieldStackOffset < 0 {
			if renvoFixedTarget == 0 {
				if trackFlowConst && flowConst.ok {
					renvoSetLocalFlowConstAtOffset(g, offset, flowConst.value, targetResolved.kind)
				} else {
					renvoClearLocalFlowConstAtOffset(g, offset)
				}
			}
			if trackLocalConst && localConst.ok {
				renvoSetLocalConstAtOffset(g, offset, localConst.value, targetResolved.kind)
			} else {
				renvoClearLocalConstAtOffset(g, offset)
			}
		}
	}
	return true
}

func renvoEmitTypedExprToSavedMem(g *renvoLinearGen, ep *renvoExprParse, idx int, typ int, addrOffset int) bool {
	renvoNonNil(g, ep)
	resolvedKind := renvoResolveType(g.meta, typ).kind
	if resolvedKind == renvoTypeComplex64 {
		if !renvoEmitComplexValueRegsForKind(g, ep, idx, resolvedKind) {
			return false
		}
		if g.c.renvoNativeIntSize == 4 && renvoUsesStackIEEEFloat(&g.asm) {
			realPart := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
			imagPart := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
			renvoAsmStorePrimaryStack(&g.asm, realPart)
			renvoAsmCopySecondaryToPrimary(&g.asm)
			renvoAsmStorePrimaryStack(&g.asm, imagPart)
			renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
			renvoAsmLoadPrimaryStack(&g.asm, realPart)
			renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 0, 4)
			renvoAsmLoadPrimaryStack(&g.asm, imagPart)
			renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 4, 4)
			return true
		}
		renvoPackComplex64RegsPrimary(g)
		renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 0, 8)
		return true
	}
	if renvoFixedTarget == 0 && renvoCanDirectScalarStore(g, renvoScalarKindSize(g.c.renvoNativeIntSize, resolvedKind)) &&
		(renvoTypeKindIsScalarValue(resolvedKind) || resolvedKind == renvoTypePointer || resolvedKind == renvoTypeFunc) &&
		renvoTypeSize(g.meta, typ) <= g.c.renvoNativeIntSize {
		if !renvoEmitScalarExprForKind(g, ep, idx, resolvedKind) {
			return false
		}
		renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 0,
			renvoScalarKindSize(g.c.renvoNativeIntSize, resolvedKind))
		return true
	}
	tempOffset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
		return false
	}
	renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, renvoTypeSize(g.meta, typ))
	return true
}

func renvoCopyTokenData(p *renvoProgram, to int, from int, count int) {
	for i := 0; i < count; i++ {
		p.toks.data[to+i] = renvo_runtime_UnsafeInt32At(p.toks.data, from+i)
	}
}

// renvoEmitGroupedTypedVarDecl handles VarSpecs whose identifier list shares one
// explicit type, such as "var first, second int". It returns zero when the
// statement is not such a declaration, one on success, and -1 on an emission
// failure.
func renvoEmitGroupedTypedVarDecl(g *renvoLinearGen, stmt *renvoStmt, assignTok int) int {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.kind != renvoStmtVar {
		return 0
	}
	typeEnd := stmt.endTok
	if assignTok > stmt.startTok {
		typeEnd = assignTok
	}
	names := renvoFixedIntScratch(4)
	pos := stmt.startTok
	if renvoTokIsKind(p, pos, renvoTokVar) {
		pos++
	}
	for {
		if pos >= typeEnd || !renvoTokIsKind(p, pos, renvoTokIdent) {
			return 0
		}
		names = append(names, pos)
		pos++
		if pos >= typeEnd || !renvoTokCharIs(p, pos, ',') {
			break
		}
		pos++
	}
	nameCount := len(names)
	if nameCount < 2 || pos >= typeEnd {
		return 0
	}
	typeResult := renvoParseScopedType(g, g.meta, p, pos, typeEnd)
	if typeResult.typ == 0 || typeResult.next != typeEnd {
		return -1
	}
	var temps []int
	if assignTok > stmt.startTok {
		rhs, ok := renvoSplitTopLevelComma(p, assignTok+1, stmt.endTok)
		if !ok || len(rhs)/2 != nameCount {
			return -1
		}
		temps = renvoFixedIntScratch(nameCount)
		// Evaluate every initializer before the new names enter scope and before
		// assigning any destination. This preserves Go's VarSpec scope and
		// left-to-right multi-assignment semantics.
		for i := 0; i < nameCount; i++ {
			ep := renvoNewExprParse()
			renvoNonNil(ep)
			rootIndex := renvoParseExpressionRoot(ep, p, rhs[i*2], rhs[i*2+1])
			if rootIndex < 0 {
				return -1
			}
			temp := renvoAddUnnamedLocal(g, typeResult.typ)
			if !renvoEmitExprToLocal(g, ep, rootIndex, temp) {
				return -1
			}
			temps = append(temps, temp)
		}
	}
	size := renvoTypeCopySize(g.meta, typeResult.typ)
	for i := 0; i < nameCount; i++ {
		tok := names[i]
		nameStart := int(renvoTokStart(p, tok))
		nameEnd := int(renvoTokEnd(p, tok))
		if nameEnd == nameStart+1 && renvo_runtime_UnsafeByteAt(p.src, nameStart) == '_' {
			continue
		}
		offset := renvoAddTypedLocal(g, nameStart, nameEnd, typeResult.typ)
		if assignTok > stmt.startTok {
			renvoEmitCopyStackToStack(g, temps[i], offset, size)
		} else {
			renvoZeroLocalAtOffset(g, offset)
		}
	}
	return 1
}

func renvoEmitMultiAssign(g *renvoLinearGen, stmt *renvoStmt, assignTok int) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	lhsStart := stmt.startTok
	if stmt.kind == renvoStmtVar && renvoTokIsKind(p, lhsStart, renvoTokVar) {
		lhsStart++
	}
	lhs, ok := renvoSplitTopLevelComma(p, lhsStart, assignTok)
	if !ok {
		return false
	}
	rhs, ok := renvoSplitTopLevelComma(p, assignTok+1, stmt.endTok)
	if !ok {
		return false
	}
	lhsCount := len(lhs) / 2
	rhsCount := len(rhs) / 2
	if lhsCount > 1 && rhsCount == 1 {
		if renvoEmitCommaOKAssign(g, stmt.kind, lhs, rhs[0], rhs[1]) {
			return true
		}
		if renvoEmitTupleCallAssign(g, stmt.kind, lhs, lhsCount, rhs[0], rhs[1]) {
			return true
		}
	}
	if lhsCount != rhsCount {
		return false
	}
	targetTypes := renvoFixedIntScratch(lhsCount)
	for i := 0; i < lhsCount; i++ {
		target := renvoNewExprParse()
		renvoNonNil(target)
		targetType := 0
		if renvoParseExpressionOK(target, p, lhs[i*2], lhs[i*2+1]) {
			rootIndex := len(target.exprs) - 1
			root := &target.exprs[rootIndex]
			if stmt.kind == renvoStmtAssign {
				targetType = renvoInferParsedExprType(g, target, rootIndex)
			}
			if root.kind == renvoExprIdent &&
				(root.nameEnd != root.nameStart+1 || renvo_runtime_UnsafeByteAt(p.src, root.nameStart) != '_') {
				localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
				if stmt.kind == renvoStmtShort {
					localIndex = renvoFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
				}
				if localIndex >= 0 {
					renvoClearLocalConstAtOffset(g, g.locals[localIndex].offset)
				}
			}
		}
		targetTypes = append(targetTypes, targetType)
	}
	if stmt.kind == renvoStmtAssign && lhsCount > 1 {
		for i := 0; i < lhsCount; i++ {
			ep := renvoNewExprParse()
			renvoNonNil(ep)
			rootIndex := renvoParseExpressionRoot(ep, p, lhs[i*2], lhs[i*2+1])
			if rootIndex < 0 {
				return false
			}
			root := &ep.exprs[rootIndex]
			captured := false
			if root.kind == renvoExprIdent {
				localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
				captured = localIndex >= 0 && g.locals[localIndex].captureOff > 0
			}
			if (root.kind != renvoExprIdent || captured) && renvoEmitAddressPrimary(g, ep, rootIndex) {
				address := renvoAddUnnamedLocal(g, renvoTypeInt)
				renvoAsmStorePrimaryStack(&g.asm, address)
				lhs[i*2] = -address
				lhs[i*2+1] = 0
			}
		}
	}
	tempOffsets := renvoFixedIntScratch(4)
	tempTypes := renvoFixedIntScratch(4)
	for i := 0; i < rhsCount; i++ {
		rhsStart := rhs[i*2]
		rhsEnd := rhs[i*2+1]
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, p, rhsStart, rhsEnd)
		if rootIndex < 0 {
			return false
		}
		typ := renvoInferParsedExprType(g, ep, rootIndex)
		if targetTypes[i] != 0 {
			typ = targetTypes[i]
		}
		if typ == 0 {
			typ = renvoTypeInt
		}
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitExprToLocal(g, ep, rootIndex, offset) {
			return false
		}
		tempOffsets = append(tempOffsets, offset)
		tempTypes = append(tempTypes, typ)
	}
	for i := 0; i < lhsCount; i++ {
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if lhsStart < 0 && lhsEnd == 0 {
			renvoAsmLoadSecondaryStack(&g.asm, -lhsStart)
			renvoEmitCopyStackToMemSecondary(g, tempOffsets[i], 0, renvoTypeSize(g.meta, tempTypes[i]))
			continue
		}
		if !renvoEmitTempToTarget(g, stmt.kind, lhsStart, lhsEnd, tempOffsets[i], tempTypes[i]) {
			return false
		}
	}
	if stmt.kind == renvoStmtAssign && lhsCount > 1 {
		// Every captured target above was staged by its cell address. Refresh
		// mirrors only after committing all writes in source order.
		renvoMoveCapturedLocals(g, false)
	}
	return true
}

func renvoEmitCommaOKAssign(g *renvoLinearGen, kind int, lhs []int, rhsStart int, rhsEnd int) bool {
	renvoNonNil(g)
	if len(lhs) != 4 {
		return false
	}
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	root := renvoParseExpressionRoot(ep, g.prog, rhsStart, rhsEnd)
	if root < 0 {
		return false
	}
	e := &ep.exprs[root]
	if e.kind != renvoExprAssert {
		return false
	}
	typ := renvoInferParsedExprType(g, ep, root)
	value := renvoAddUnnamedLocal(g, typ)
	ok := renvoAddUnnamedLocal(g, renvoTypeBool)
	if !renvoEmitTypeAssertionToLocal(g, ep, root, value, ok, false) {
		return false
	}
	if !renvoEmitTempToTarget(g, kind, lhs[0], lhs[1], value, typ) {
		return false
	}
	return renvoEmitTempToTarget(g, kind, lhs[2], lhs[3], ok, renvoTypeBool)
}

func renvoEmitTupleCallAssign(g *renvoLinearGen, kind int, lhs []int, lhsCount int, rhsStart int, rhsEnd int) bool {
	renvoNonNil(g)
	p := g.prog
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, p, rhsStart, rhsEnd)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	if root.kind != renvoExprCall {
		return false
	}
	fnIndex := renvoFuncInfoFromCall(g, ep, root.left)
	resultType := 0
	if fnIndex >= 0 {
		resultType = g.meta.funcs[fnIndex].resultType
	} else {
		resultType = renvoInterfaceMethodCallResultType(g, ep, rootIndex)
	}
	if !renvoTypeIsTuple(g.meta, resultType) {
		return false
	}
	tuple := renvoResolveType(g.meta, resultType)
	renvoNonNil(tuple)
	if tuple.count != lhsCount {
		return false
	}
	offset := renvoAddUnnamedLocal(g, resultType)
	if !renvoEmitStructCallToLocal(g, ep, rootIndex, resultType, offset) {
		return false
	}
	for i := 0; i < lhsCount; i++ {
		field := g.meta.fields[tuple.first+i]
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !renvoEmitTempToTarget(g, kind, lhsStart, lhsEnd, offset-field.offset, field.typ) {
			return false
		}
	}
	return true
}
func renvoEmitExprToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	renvoNonNil(g, ep)
	if renvoEmitTypedAssign(g, ep, idx, offset) {
		return true
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmStorePrimaryStack(&g.asm, offset)
	return true
}
func renvoEmitTempToTarget(g *renvoLinearGen, kind int, targetStart int, targetEnd int, tempOffset int, tempType int) bool {
	renvoNonNil(g)
	p := g.prog
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, p, targetStart, targetEnd)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	size := renvoTypeCopySize(g.meta, tempType)
	if root.kind == renvoExprIdent {
		if root.nameEnd == root.nameStart+1 && renvo_runtime_UnsafeByteAt(p.src, root.nameStart) == '_' {
			return true
		}
		localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
		if kind == renvoStmtShort || kind == renvoStmtVar {
			if kind == renvoStmtVar {
				localIndex = -1
			} else {
				localIndex = renvoFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
			}
			if localIndex < 0 {
				offset := renvoAddTypedLocal(g, root.nameStart, root.nameEnd, tempType)
				renvoEmitCopyStackToStack(g, tempOffset, offset, size)
				renvoClearLocalConstAtOffset(g, offset)
				return true
			}
		}
		if localIndex >= 0 {
			renvoEmitCopyStackToStack(g, tempOffset, g.locals[localIndex].offset, size)
			renvoClearLocalConstAtOffset(g, g.locals[localIndex].offset)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		renvoEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if kind == renvoStmtShort || kind == renvoStmtVar {
		return false
	}
	if root.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, rootIndex) {
			return false
		}
		targetType := renvoInferParsedExprType(g, ep, rootIndex)
		targetSize := renvoTypeSize(g.meta, targetType)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, ep, rootIndex) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		targetType := renvoInferParsedExprType(g, ep, rootIndex)
		targetSize := renvoTypeSize(g.meta, targetType)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == renvoExprUnary && renvoTokCharIs(p, root.tok, '*') {
		if !renvoEmitIntExpr(g, ep, root.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		targetType := renvoInferParsedExprType(g, ep, rootIndex)
		targetSize := renvoTypeSize(g.meta, targetType)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	return false
}

func renvoEmitTempToMapEntry(g *renvoLinearGen, tempOffset int, elemType int) {
	renvoNonNil(g)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	elem := renvoResolveType(g.meta, elemType)
	if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(elem.kind) {
		renvoEmitCopyStackToMemSecondary(g, tempOffset, 16, renvoTypeSize(g.meta, elemType))
	} else {
		renvoAsmLoadPrimaryStack(&g.asm, tempOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 16, renvoScalarKindSize(g.c.renvoNativeIntSize, elem.kind))
	}
}
func renvoEmitCopyStackToBss(g *renvoLinearGen, srcOffset int, bssOffset int, size int) {
	renvoNonNil(g)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	for at := 0; at < size; at += g.c.renvoNativeIntSize {
		renvoAsmLoadPrimaryStack(&g.asm, srcOffset-at)
		renvoAsmStorePrimaryBss(&g.asm, bssOffset+at)
	}
}
func renvoFindLocalIndexInCurrentScope(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	index := renvoFindLocalIndex(g, nameStart, nameEnd)
	if index >= g.scopeBase {
		return index
	}
	return -1
}

func renvoSetLocalConstAtOffset(g *renvoLinearGen, offset int, value int, kind int) {
	renvoNonNil(g)
	value = renvoConvertConstInt(g.c.renvoNativeIntSize, value, kind)
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		if local.offset == offset {
			local.constValue = value
			local.constValid = 1
			return
		}
	}
}

func renvoClearLocalConstAtOffset(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		if local.offset == offset {
			local.constValid = 0
			return
		}
	}
}

func renvoSetLocalFlowConstAtOffset(g *renvoLinearGen, offset int, value int, kind int) {
	renvoNonNil(g)
	value = renvoConvertConstInt(g.c.renvoNativeIntSize, value, kind)
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		if local.offset == offset {
			local.flowConstValue = value
			local.flowConstValid = 1
			return
		}
	}
}

func renvoClearLocalFlowConstAtOffset(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].flowConstValid = 0
			return
		}
	}
}

func renvoLocalFlowConstTrackable(g *renvoLinearGen, typ int, nameStart int, nameEnd int) bool {
	if !renvoIsSysVObject(g.c) {
		return false
	}
	resolved := renvoResolveType(g.meta, typ)
	renvoNonNil(resolved)
	return renvoTypeKindIsScalarInt(resolved.kind) && !renvoLocalNameAddressTaken(g, nameStart, nameEnd)
}

// A top-level assignment dominates all following structured control flow. It
// also dominates a later label unless an earlier goto can enter after the
// assignment. Such a stable final assignment may therefore remain known across
// loops and backwards gotos without turning branch-local facts into
// function-wide constants.
func renvoTopLevelAssignmentDominates(g *renvoLinearGen, assignmentTok int) bool {
	renvoNonNil(g)
	if !renvoIsSysVObject(g.c) || g.flowControlDepth != 0 ||
		g.currentFunc < 0 || g.currentFunc >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[g.currentFunc]
	for tok := fn.bodyStart; tok < assignmentTok; tok++ {
		if renvoTokIsKind(g.prog, tok, renvoTokGoto) {
			return false
		}
	}
	return true
}

func renvoCaptureLocalFlowConsts(g *renvoLinearGen, count int) []int {
	values := renvoFixedIntScratch(count * 2)
	for i := 0; i < count; i++ {
		values = append(values, g.locals[i].flowConstValue, g.locals[i].flowConstValid)
	}
	return values
}

func renvoRestoreLocalFlowConsts(g *renvoLinearGen, values []int, count int) {
	for i := 0; i < count && i*2+1 < len(values); i++ {
		g.locals[i].flowConstValue = values[i*2]
		g.locals[i].flowConstValid = values[i*2+1]
	}
}

func renvoMergeLocalFlowConsts(g *renvoLinearGen, left []int, right []int, count int) {
	for i := 0; i < count; i++ {
		at := i * 2
		valid := at+1 < len(left) && at+1 < len(right) && left[at+1] != 0 && right[at+1] != 0 && left[at] == right[at]
		g.locals[i].flowConstValid = renvoBoolInt(valid)
		if valid {
			g.locals[i].flowConstValue = left[at]
		}
	}
}

func renvoClearAllLocalFlowConsts(g *renvoLinearGen) {
	for i := 0; i < g.localCount; i++ {
		g.locals[i].flowConstValid = 0
	}
}

func renvoLocalConstTrackable(g *renvoLinearGen, typ int, nameStart int, nameEnd int, afterTok int) bool {
	if renvoFixedTarget != 0 {
		return false
	}
	renvoNonNil(g)
	if !renvoIsSysVObject(g.c) {
		return false
	}
	resolved := renvoResolveType(g.meta, typ)
	renvoNonNil(resolved)
	if !renvoTypeKindIsScalarInt(resolved.kind) {
		return false
	}
	return !renvoLocalNameAddressTaken(g, nameStart, nameEnd) &&
		!renvoLocalNameWrittenAfter(g, nameStart, nameEnd, afterTok)
}

func renvoLocalNameAddressTaken(g *renvoLinearGen, nameStart int, nameEnd int) bool {
	renvoNonNil(g)
	if nameEnd <= nameStart {
		return true
	}
	p := g.prog
	end := renvoTokCount(p)
	start := 0
	if g.currentFunc >= 0 && g.currentFunc < len(g.meta.funcs) {
		start = g.meta.funcs[g.currentFunc].bodyStart
		end = g.meta.funcs[g.currentFunc].bodyEnd
	}
	for i := start; i < end; i++ {
		if !renvoTokIsKind(p, i, renvoTokIdent) ||
			!renvoBytesEqualRange(p.src, renvoTokStart(p, i), renvoTokEnd(p, i), nameStart, nameEnd) {
			continue
		}
		addressTok := -1
		if i > 0 && renvoTokCharIs(p, i-1, '&') {
			addressTok = i - 1
		} else if i > 1 && renvoTokCharIs(p, i-1, '(') && renvoTokCharIs(p, i-2, '&') {
			addressTok = i - 2
		}
		if addressTok < 0 {
			continue
		}
		// The C frontend emits `_ = &local` solely to satisfy Go's local-use
		// rule. The address is immediately discarded and cannot mutate or
		// escape the object, so it must not suppress a constant fact.
		if addressTok >= 2 && renvoTokCharIs(p, addressTok-1, '=') && renvoTokIdentIs(p, addressTok-2, "_") {
			continue
		}
		return true
	}
	return false
}

// C lowering keeps explicit parentheses around many lvalues, including the
// address arguments used by pre/post increment helpers. Treat &name and
// &(name) alike when deciding whether a local can remain a compile-time fact.
func renvoLocalNameWrittenAfter(g *renvoLinearGen, nameStart int, nameEnd int, afterTok int) bool {
	renvoNonNil(g)
	if nameEnd <= nameStart {
		return true
	}
	p := g.prog
	src := p.src
	nameSize := nameEnd - nameStart
	nameFirst := renvo_runtime_UnsafeByteAt(src, nameStart)
	end := renvoTokCount(p)
	if g.currentFunc >= 0 && g.currentFunc < len(g.meta.funcs) {
		end = g.meta.funcs[g.currentFunc].bodyEnd
	}
	i := afterTok
	if i < 0 {
		i = 0
	}
	for i < end {
		base := i * renvoTokenStride
		first := int(renvo_runtime_UnsafeInt32At(p.toks.data, base))
		packed := int(renvo_runtime_UnsafeInt32At(p.toks.data, base+1))
		tokenStart := packed & 0xffffff
		tokenEnd := tokenStart + (packed>>24&255 | first>>16&0xff00)
		if first&255 == renvoTokIdent && tokenEnd-tokenStart == nameSize && renvo_runtime_UnsafeByteAt(src, tokenStart) == nameFirst && renvoBytesEqualRange(src, tokenStart, tokenEnd, nameStart, nameEnd) {
			addressTok := -1
			if i > 0 && renvoTokCharIs(p, i-1, '&') {
				addressTok = i - 1
			} else if i > 1 && renvoTokCharIs(p, i-1, '(') && renvoTokCharIs(p, i-2, '&') {
				addressTok = i - 2
			}
			if addressTok >= 0 &&
				!(addressTok >= 2 && renvoTokCharIs(p, addressTok-1, '=') && renvoTokIdentIs(p, addressTok-2, "_")) {
				return true
			}
			if renvoTok2Is(p, i+1, '+', '+') || renvoTok2Is(p, i+1, '-', '-') {
				return true
			}
			lineEnd := renvoStatementLineEnd(p, i, end)
			assignTok := renvoFindAssignmentToken(p, i, lineEnd)
			if assignTok > i {
				return true
			}
		}
		i++
	}
	return false
}

func renvoSplitTopLevelComma(p *renvoProgram, start int, end int) ([]int, bool) {
	renvoNonNil(p)
	var ranges []int
	if renvoFixedTarget != 0 {
		ranges = make([]int, 0, 16)
	}
	partStart := start
	depth := 0
	i := start
	for i < end {
		if i < renvoTokCount(p) {
			c := renvoTokSingleChar(p, i)
			if c == '(' || c == '[' || c == '{' {
				depth++
			} else if c == ')' || c == ']' || c == '}' {
				if depth > 0 {
					depth--
				}
			} else if depth == 0 && c == ',' {
				ranges = append(ranges, partStart)
				ranges = append(ranges, i)
				partStart = i + 1
			}
		}
		i++
	}
	if partStart < end {
		ranges = append(ranges, partStart)
		ranges = append(ranges, end)
	}
	return ranges, true
}

func renvoHasTopLevelComma(p *renvoProgram, start int, end int) bool {
	renvoNonNil(p)
	depth := 0
	for i := start; i < end; i++ {
		if i >= renvoTokCount(p) {
			return false
		}
		c := renvoTokSingleChar(p, i)
		if c == '(' || c == '[' || c == '{' {
			depth++
		} else if c == ')' || c == ']' || c == '}' {
			if depth > 0 {
				depth--
			}
		} else if depth == 0 && c == ',' {
			return true
		}
	}
	return false
}

func renvoEmitTupleReturn(g *renvoLinearGen, start int, end int) bool {
	renvoNonNil(g)
	resultType := g.meta.funcs[g.currentFunc].resultType
	tuple := renvoResolveType(g.meta, resultType)
	renvoNonNil(tuple)
	parts, ok := renvoSplitTopLevelComma(g.prog, start, end)
	if !ok {
		return false
	}
	count := len(parts) / 2
	if count == tuple.count {
		for i := 0; i < count; i++ {
			partStart := parts[i*2]
			partEnd := parts[i*2+1]
			field := g.meta.fields[tuple.first+i]
			if !renvoEmitTupleReturnField(g, partStart, partEnd, field.typ, field.offset) {
				return false
			}
		}
		return true
	}
	if count == 1 {
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, g.prog, start, end)
		if rootIndex < 0 {
			return false
		}
		sourceType := renvoInferParsedExprType(g, ep, rootIndex)
		source := renvoResolveType(g.meta, sourceType)
		if source.kind != renvoTypeStruct || source.count != tuple.count {
			return false
		}
		unchanged := true
		for i := 0; i < tuple.count; i++ {
			from := g.meta.fields[source.first+i]
			to := g.meta.fields[tuple.first+i]
			if from.offset != to.offset || !renvoTypesEquivalent(g.meta, from.typ, to.typ) {
				unchanged = false
			}
		}
		if unchanged {
			return renvoEmitStructReturnExpr(g, ep, rootIndex)
		}
		// Forwarded result lists still require assignment conversion for each
		// position. Evaluate the call once before boxing any concrete results.
		sourceOffset := renvoAddUnnamedLocal(g, sourceType)
		if !renvoEmitExprToLocal(g, ep, rootIndex, sourceOffset) {
			return false
		}
		for i := 0; i < tuple.count; i++ {
			from := g.meta.fields[source.first+i]
			to := g.meta.fields[tuple.first+i]
			offset := sourceOffset - from.offset
			if renvoResolveType(g.meta, to.typ).kind == renvoTypeInterface && renvoResolveType(g.meta, from.typ).kind != renvoTypeInterface {
				boxed := renvoAddUnnamedLocal(g, to.typ)
				if !renvoEmitConcreteLocalToInterface(g, from.typ, offset, boxed) {
					return false
				}
				offset = boxed
			} else if !renvoTypesEquivalent(g.meta, from.typ, to.typ) {
				return false
			}
			renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
			renvoEmitCopyStackToMemSecondary(g, offset, to.offset, renvoTypeCopySize(g.meta, to.typ))
		}
		return true
	}
	return false
}
func renvoEmitTupleReturnField(g *renvoLinearGen, start int, end int, typ int, fieldOffset int) bool {
	renvoNonNil(g)
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := renvoParseExpressionRoot(ep, g.prog, start, end)
	if rootIndex < 0 {
		return false
	}
	if renvoTypeIsSlice(g.meta, typ) {
		if !renvoEmitSliceReturnValueRegs(g, ep, rootIndex, typ) {
			return false
		}
		renvoAsmPushSliceRegs(&g.asm)
		renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
		renvoAsmPopStoreSliceMemSecondary(&g.asm, fieldOffset)
		return true
	}
	tempOffset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitExprToLocal(g, ep, rootIndex, tempOffset) {
		return false
	}
	size := renvoTypeCopySize(g.meta, typ)
	renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, fieldOffset, size)
	return true
}
func renvoInferParsedExprType(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return 0
	}
	if ep.exprs[idx].inferred != 0 {
		return ep.exprs[idx].inferred
	}
	typ := renvoInferParsedExprTypeUncached(g, ep, idx)
	if typ != 0 {
		ep.exprs[idx].inferred = typ
	}
	return typ
}

func renvoInferParsedExprTypeUncached(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	e := &ep.exprs[idx]
	if (e.kind == renvoExprInt || e.kind == renvoExprFloat) && renvoExprTokenIsImaginary(p, e.tok) {
		return renvoBuiltinTypeComplex
	}
	if e.kind == renvoExprBool {
		return renvoTypeBool
	}
	if e.kind == renvoExprInt || e.kind == renvoExprChar {
		return renvoTypeInt
	}
	if e.kind == renvoExprFloat {
		return renvoTypeFloat64
	}
	if e.kind == renvoExprString {
		return renvoTypeString
	}
	if e.kind == renvoExprFunc {
		closureIndex := renvoClosureIndexByToken(meta, e.tok)
		if closureIndex >= 0 {
			return renvoFunctionTypeFromInfo(meta, meta.closures[closureIndex].fnIndex)
		}
		return 0
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].typ
		}
		symIndex := renvoFindMetaGlobalIndex(meta, e.nameStart, e.nameEnd, renvoTokVar)
		if symIndex < 0 {
			symIndex = renvoFindMetaGlobalIndex(meta, e.nameStart, e.nameEnd, renvoTokConst)
		}
		if symIndex >= 0 {
			return meta.globals[symIndex].typ
		}
		constStringTok := renvoFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constStringTok >= 0 {
			return renvoTypeString
		}
		fnIndex := renvoFindMetaFunction(meta, e.nameStart, e.nameEnd)
		if fnIndex >= 0 {
			return renvoFunctionTypeFromInfo(meta, fnIndex)
		}
		return renvoTypeInt
	}
	if e.kind == renvoExprCall {
		if renvoExprIsErrorStringCall(g, ep, idx) {
			return renvoTypeString
		}
		callee := renvoResolvedNumericCalleeCode(g, ep, e.left)
		if callee == renvoIdentRecover && e.argCount == 0 {
			return renvoBuiltinTypeInterface
		}
		if callee == renvoIdentAppend && e.argCount >= 1 {
			return renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		if callee == renvoIdentByteSlice && e.argCount == 1 {
			return renvoAddSequenceType(meta, renvoTypeSlice, renvoTypeByte, 0, renvoBackendSliceValueSize)
		}
		if callee == renvoIdentString && e.argCount == 1 {
			return renvoTypeString
		}
		if callee == renvoIdentMake && e.argCount >= 1 {
			return renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		if callee == renvoIdentNew && e.argCount == 1 {
			targetType := renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
			if targetType != 0 {
				return renvoAddType(meta, renvoTypePointer, targetType, 0, 0, renvoBackendValueSlotSize, 0, 0)
			}
		}
		if callee == renvoIdentCap || callee == renvoIdentLen || callee == renvoIdentOpen || callee == renvoIdentClose || callee == renvoIdentRead || callee == renvoIdentWrite || callee == renvoIdentChmod || callee == renvoIdentCopy || callee == renvoIdentSyscall {
			return renvoTypeInt
		}
		if callee == renvoIdentReal || callee == renvoIdentImag {
			if e.argCount == 1 {
				arg := renvoResolveType(meta, renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)))
				if arg.kind == renvoTypeComplex64 {
					return renvoBuiltinTypeFloat32
				}
			}
			return renvoTypeFloat64
		}
		if callee == renvoIdentComplex {
			if e.argCount == 2 {
				first := renvoResolveType(meta, renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)))
				secondIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)
				second := renvoResolveType(meta, renvoInferParsedExprType(g, ep, secondIndex))
				if first.kind == renvoTypeFloat32 && (second.kind == renvoTypeFloat32 || renvoExprIsUntypedNumber(ep, secondIndex)) {
					return renvoBuiltinTypeComplex64
				}
			}
			return renvoBuiltinTypeComplex
		}
		funcType := renvoFunctionValueCalleeType(g, ep, e.left)
		if funcType != 0 {
			return renvoResolveType(meta, funcType).elem
		}
		if resultType := renvoInterfaceMethodCallResultType(g, ep, idx); resultType != 0 {
			return resultType
		}
		fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			return meta.funcs[fnIndex].resultType
		}
		if e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				return conversionType
			}
		}
	}
	if e.kind == renvoExprIndex {
		leftType := renvoInferParsedExprType(g, ep, e.left)
		t := renvoResolveType(meta, leftType)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			pointerElem := t.elem
			t = renvoResolveType(meta, pointerElem)
			if t.kind != renvoTypeArray && t.kind != renvoTypeSlice {
				return pointerElem
			}
		}
		if t.kind == renvoTypeSlice || t.kind == renvoTypeArray {
			return t.elem
		}
		if t.kind == renvoTypeString {
			return renvoTypeByte
		}
	}
	if e.kind == renvoExprSlice {
		baseType := renvoInferParsedExprType(g, ep, e.left)
		base := renvoResolveType(meta, baseType)
		renvoNonNil(base)
		if base.kind == renvoTypePointer {
			base = renvoResolveType(meta, base.elem)
		}
		if base.kind == renvoTypeArray {
			return renvoAddSequenceType(meta, renvoTypeSlice, base.elem, 0, renvoBackendSliceValueSize)
		}
		return baseType
	}
	if e.kind == renvoExprSelector {
		baseType := renvoInferParsedExprType(g, ep, e.left)
		if renvoResolveType(meta, baseType).kind == renvoTypeInterface {
			return renvoInterfaceMethodType(g, baseType, e)
		}
		fnIndex, expression := renvoMethodSelectorInfo(g, ep, idx)
		if fnIndex >= 0 {
			first := 1
			if expression {
				first = 0
			}
			return renvoFunctionTypeFromInfoStart(meta, fnIndex, first)
		}
		fieldType := renvoStructFieldType(g, baseType, e.nameStart, e.nameEnd)
		if fieldType != 0 {
			return fieldType
		}
	}
	if e.kind == renvoExprAssert {
		asserted := renvoParseType(meta, p, e.right, e.firstArg)
		if asserted.typ != 0 && asserted.next == e.firstArg {
			return asserted.typ
		}
		return 0
	}
	if e.kind == renvoExprComposite {
		return renvoTypeFromExpr(g, ep, idx)
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(p, e.tok, '!') {
			return renvoTypeBool
		}
		if renvoTokCharIs(p, e.tok, '+') || renvoTokCharIs(p, e.tok, '-') || renvoTokCharIs(p, e.tok, '^') {
			return renvoInferParsedExprType(g, ep, e.left)
		}
		if renvoTokCharIs(p, e.tok, '&') {
			elemType := renvoInferParsedExprType(g, ep, e.left)
			if elemType == 0 {
				return 0
			}
			return renvoAddPointerType(meta, elemType, renvoPointerSpaceData)
		}
		if renvoTokCharIs(p, e.tok, '*') {
			innerType := renvoInferParsedExprType(g, ep, e.left)
			inner := renvoResolveType(meta, innerType)
			renvoNonNil(inner)
			if inner.kind == renvoTypePointer {
				return inner.elem
			}
		}
	}
	if e.kind == renvoExprBinary {
		start := int(renvoTokStart(p, e.tok))
		end := int(renvoTokEnd(p, e.tok))
		c0 := renvo_runtime_UnsafeByteAt(p.src, start)
		var c1 byte
		if start+1 < end {
			c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
		}
		if renvoIsComparisonChars(c0, c1) {
			return renvoTypeBool
		}
		if renvoTok2Is(p, e.tok, '&', '&') || renvoTok2Is(p, e.tok, '|', '|') {
			return renvoTypeBool
		}
		leftTypeIndex := renvoInferParsedExprType(g, ep, e.left)
		if renvoTok2Is(p, e.tok, '<', '<') || renvoTok2Is(p, e.tok, '>', '>') {
			if renvoExprIsUntypedNumber(ep, e.left) {
				return renvoTypeInt
			}
			return leftTypeIndex
		}
		rightTypeIndex := renvoInferParsedExprType(g, ep, e.right)
		leftType := renvoResolveType(meta, leftTypeIndex)
		renvoNonNil(leftType)
		rightType := renvoResolveType(meta, rightTypeIndex)
		renvoNonNil(rightType)
		if renvoTokCharIs(p, e.tok, '+') && leftType.kind == renvoTypeString && rightType.kind == renvoTypeString {
			if leftTypeIndex == rightTypeIndex {
				return leftTypeIndex
			}
			if leftTypeIndex != renvoTypeString && ep.exprs[e.right].kind == renvoExprString {
				return leftTypeIndex
			}
			if rightTypeIndex != renvoTypeString && ep.exprs[e.left].kind == renvoExprString {
				return rightTypeIndex
			}
			return renvoTypeString
		}
		if renvoTypeKindIsFloat(leftType.kind) || renvoTypeKindIsFloat(rightType.kind) {
			if leftType.kind == renvoTypeFloat32 && (rightType.kind == renvoTypeFloat32 || renvoExprIsUntypedNumber(ep, e.right)) {
				return leftTypeIndex
			}
			if rightType.kind == renvoTypeFloat32 && renvoExprIsUntypedNumber(ep, e.left) {
				return rightTypeIndex
			}
			return renvoTypeFloat64
		}
		if renvoTypeKindIsComplex(leftType.kind) || renvoTypeKindIsComplex(rightType.kind) {
			if leftType.kind == renvoTypeComplex64 && rightType.kind == renvoTypeComplex64 {
				return leftTypeIndex
			}
			return renvoBuiltinTypeComplex
		}
		if renvoTok2Is(p, e.tok, '<', '<') || renvoTok2Is(p, e.tok, '>', '>') {
			return leftTypeIndex
		}
		if leftType.kind == rightType.kind && renvoTypeKindIsScalarInt(leftType.kind) {
			return leftTypeIndex
		}
		if renvoTypeKindIsScalarInt(leftType.kind) && renvoExprIsUntypedInteger(ep, e.right) {
			return leftTypeIndex
		}
		if renvoTypeKindIsScalarInt(rightType.kind) && renvoExprIsUntypedInteger(ep, e.left) {
			return rightTypeIndex
		}
	}
	return renvoTypeInt
}

func renvoExprTokenIsImaginary(p *renvoProgram, tok int) bool {
	renvoNonNil(p)
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	start := int(renvoTokStart(p, tok))
	end := int(renvoTokEnd(p, tok))
	return end > start && renvo_runtime_UnsafeByteAt(p.src, end-1) == 'i'
}

func renvoExprIsUntypedInteger(ep *renvoExprParse, idx int) bool {
	renvoNonNil(ep)
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt || e.kind == renvoExprChar {
		return true
	}
	if e.kind == renvoExprUnary {
		return renvoExprIsUntypedInteger(ep, e.left)
	}
	if e.kind == renvoExprBinary {
		return renvoExprIsUntypedInteger(ep, e.left) && renvoExprIsUntypedInteger(ep, e.right)
	}
	return false
}

func renvoExprIsUntypedNumber(ep *renvoExprParse, idx int) bool {
	renvoNonNil(ep)
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt || e.kind == renvoExprFloat || e.kind == renvoExprChar {
		return true
	}
	if e.kind == renvoExprUnary {
		return renvoExprIsUntypedNumber(ep, e.left)
	}
	if e.kind == renvoExprBinary {
		return renvoExprIsUntypedNumber(ep, e.left) && renvoExprIsUntypedNumber(ep, e.right)
	}
	return false
}

func renvoExprIsUntypedZeroFloat(p *renvoProgram, ep *renvoExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.kind == renvoExprFloat {
		return renvoParseFloatTokenBits(p, e.tok, 52, 11, 1023)&0x7fffffffffffffff == 0
	}
	if e.kind == renvoExprUnary {
		return renvoExprIsUntypedZeroFloat(p, ep, e.left)
	}
	return false
}

func renvoBinaryFloatKind(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) int {
	left := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
	right := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right))
	if left.kind == renvoTypeFloat32 && (right.kind == renvoTypeFloat32 || renvoExprIsUntypedNumber(ep, e.right)) {
		return renvoTypeFloat32
	}
	if right.kind == renvoTypeFloat32 && renvoExprIsUntypedNumber(ep, e.left) {
		return renvoTypeFloat32
	}
	return renvoTypeFloat64
}
func renvoPointerTargetKind(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	pointerType := renvoTypeInt
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			localInfo := &g.locals[localIndex]
			pointerType = localInfo.typ
		} else {
			globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalType != 0 {
				pointerType = globalType
			}
		}
	} else {
		pointerType = renvoInferParsedExprType(g, ep, idx)
	}
	pointerResolved := renvoResolveType(g.meta, pointerType)
	renvoNonNil(pointerResolved)
	if pointerResolved.kind == renvoTypePointer {
		targetResolved := renvoResolveType(g.meta, pointerResolved.elem)
		renvoNonNil(targetResolved)
		return targetResolved.kind
	}
	return pointerResolved.kind
}
func renvoTypeFromExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	renvoNonNil(p)
	renvoNonNil(meta)
	e := &ep.exprs[idx]
	tokenCount := renvoTokCount(p)
	if e.tok < 0 || e.tok >= tokenCount {
		return 0
	}
	endTok := e.tok
	for endTok < tokenCount && int(renvoTokEnd(p, endTok)) <= e.nameEnd {
		endTok++
	}
	typeResult := renvoParseScopedType(g, meta, p, e.tok, endTok)
	if !renvoResolveInferredArrayCompositeLength(meta, g, ep, idx, typeResult.typ) {
		return 0
	}
	return typeResult.typ
}
func renvoResolveInferredArrayCompositeLength(meta *renvoMeta, g *renvoLinearGen, ep *renvoExprParse, idx int, typ int) bool {
	renvoNonNil(meta, ep)
	if typ <= 0 || typ >= len(meta.types) {
		return true
	}
	t := &meta.types[typ]
	e := &ep.exprs[idx]
	if t.kind != renvoTypeArray || t.count >= 0 || e.kind != renvoExprComposite {
		return true
	}
	count := 0
	next := 0
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		at := next
		if field.key >= 0 {
			key := renvoEvalMetaParsedConstExpr(meta, meta.prog, ep, field.key, 0)
			if g != nil {
				key = renvoEvalConstExpr(g, ep, field.key)
			}
			if !key.ok || key.value < 0 {
				return false
			}
			at = key.value
		}
		next = at + 1
		if next > count {
			count = next
		}
	}
	t.count = count
	t.size = count * renvoTypeSize(meta, t.elem)
	return true
}
func renvoFindTypeByRange(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	typ := renvoFindNamedType(g.meta, nameStart, nameEnd)
	if typ >= 0 {
		return typ
	}
	return 0
}
func renvoConversionTypeFromExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	callee := &ep.exprs[idx]
	if callee.kind == renvoExprUnary && renvoTokCharIs(g.prog, callee.tok, '*') {
		target := renvoConversionTypeFromExpr(g, ep, callee.left)
		if target == 0 {
			target = renvoTypeFromExpr(g, ep, callee.left)
		}
		if target != 0 {
			return renvoAddPointerType(g.meta, target, renvoPointerSpaceData)
		}
	}
	if callee.kind != renvoExprIdent {
		return 0
	}
	if renvoTokIsKind(g.prog, callee.tok, renvoTokFunc) {
		end := renvoPrimaryTypeEnd(g.prog, callee.tok, renvoTokCount(g.prog))
		parsed := renvoParseType(g.meta, g.prog, callee.tok, end)
		return parsed.typ
	}
	if renvoFindLocalIndex(g, callee.nameStart, callee.nameEnd) >= 0 || renvoFindGlobalType(g, callee.nameStart, callee.nameEnd) != 0 || renvoFindMetaFunction(g.meta, callee.nameStart, callee.nameEnd) >= 0 {
		return 0
	}
	builtin := renvoBuiltinTypeFromToken(g.prog, callee.tok)
	if builtin != 0 {
		return builtin
	}
	if renvoTokCharIs(g.prog, callee.tok, '[') {
		return renvoTypeFromExpr(g, ep, idx)
	}
	return renvoFindTypeByRange(g, callee.nameStart, callee.nameEnd)
}
func renvoLocalTypeAtOffset(g *renvoLinearGen, offset int) int {
	renvoNonNil(g)
	// Nested scopes reuse frame slots. The newest active declaration owns an
	// offset when an older local and the current local share that slot.
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		if local.offset == offset {
			return local.typ
		}
	}
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		t := renvoResolveType(g.meta, local.typ)
		renvoNonNil(t)
		if t.kind == renvoTypeStruct {
			for j := 0; j < t.count; j++ {
				field := g.meta.fields[t.first+j]
				if local.offset-field.offset == offset {
					return field.typ
				}
			}
		}
	}
	return 0
}
func renvoEmitTypedAssign(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	renvoNonNil(meta)
	destType := renvoLocalTypeAtOffset(g, offset)
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if e.kind == renvoExprAssert {
		return renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true)
	}
	destResolved := renvoResolveType(meta, destType)
	renvoNonNil(destResolved)
	if g.c.renvoNativeIntSize == 4 && destResolved.kind == renvoTypeComplex {
		return renvoEmitStackComplex128ToLocal(g, ep, idx, offset)
	}
	if renvoTypeKindIsComplex(destResolved.kind) {
		if !renvoEmitComplexValueRegsForKind(g, ep, idx, destResolved.kind) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, offset, renvoComplexSecondaryStackOffset(g, destType, offset))
		return true
	}
	if destResolved.kind == renvoTypeInterface {
		return renvoEmitInterfaceAssignToLocal(g, ep, idx, offset)
	}
	if (destResolved.kind == renvoTypeArray || destResolved.kind == renvoTypeStruct) && e.kind == renvoExprIdent {
		size := renvoTypeSize(meta, destType)
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if renvoTypeSize(meta, g.locals[localIndex].typ) != size {
				return false
			}
			renvoEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, size)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset < 0 || renvoTypeSize(meta, renvoFindGlobalType(g, e.nameStart, e.nameEnd)) != size {
			return false
		}
		if size < renvoBackendValueSlotSize {
			size = renvoBackendValueSlotSize
		}
		for at := 0; at < size; at += g.c.renvoNativeIntSize {
			renvoAsmCopyBssToStackSlot(&g.asm, globalOffset+at, offset-at)
		}
		return true
	}
	if destResolved.kind == renvoTypeArray || destResolved.kind == renvoTypeStruct {
		if e.kind == renvoExprCall {
			if e.argCount == 1 && renvoConversionTypeFromExpr(g, ep, e.left) != 0 && renvoIsSliceArrayConversion(g, ep, ep.args[e.firstArg], destType) {
				return renvoEmitSliceArrayConversion(g, ep, ep.args[e.firstArg], destType, offset)
			}
			return renvoEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == renvoExprIndex {
			valueType := renvoInferParsedExprType(g, ep, idx)
			valueResolved := renvoResolveType(meta, valueType)
			if valueResolved.kind != destResolved.kind || renvoTypeSize(meta, valueType) != renvoTypeSize(meta, destType) || !renvoEmitIndexAddressPrimary(g, ep, idx) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(meta, destType))
			return true
		}
		if e.kind == renvoExprSelector {
			fieldType := renvoInferParsedExprType(g, ep, idx)
			if renvoResolveType(meta, fieldType).kind != destResolved.kind || renvoTypeSize(meta, fieldType) != renvoTypeSize(meta, destType) {
				return false
			}
			if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
				return false
			}
			renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(meta, destType))
			return true
		}
		if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
			valueType := renvoInferParsedExprType(g, ep, idx)
			if renvoResolveType(meta, valueType).kind != destResolved.kind || renvoTypeSize(meta, valueType) != renvoTypeSize(meta, destType) {
				return false
			}
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(meta, destType))
			return true
		}
		if e.kind == renvoExprComposite {
			valueOffset := offset
			copyValue := false
			// Use a separate frame slot only when evaluating a field can observe
			// the destination. Most compiler literals are pure construction, so
			// clearing their final slot first is both safe and substantially smaller.
			if renvoCompositeNeedsTemporary(g, ep, e, offset) {
				valueOffset = renvoAddUnnamedLocal(g, destType)
				copyValue = true
			}
			renvoZeroLocalAtOffset(g, valueOffset)
			if destResolved.kind == renvoTypeArray {
				if !renvoEmitCompositeFieldToStack(g, ep, idx, destType, valueOffset) {
					return false
				}
				if copyValue {
					renvoEmitCopyStackToStack(g, valueOffset, offset, renvoTypeSize(meta, destType))
				}
				return true
			}
			for i := 0; i < e.argCount; i++ {
				field := ep.fields[e.firstArg+i]
				fieldIndex := renvoCompositeStructFieldIndex(g, destType, &field, i)
				if fieldIndex < 0 {
					return false
				}
				fieldOffset := g.meta.fields[fieldIndex].offset
				fieldType := g.meta.fields[fieldIndex].typ
				if fieldType == 0 {
					return false
				}
				if !renvoEmitCompositeFieldToStack(g, ep, field.expr, fieldType, valueOffset-fieldOffset) {
					if renvoFixedTarget == 0 {
						renvoPrintErr("renvo: failed composite field: ")
						if field.nameEnd > field.nameStart {
							write(2, g.prog.src[field.nameStart:field.nameEnd], -1)
						}
						renvoPrintErr("\n")
					}
					return false
				}
			}
			if copyValue {
				renvoEmitCopyStackToStack(g, valueOffset, offset, renvoTypeSize(meta, destType))
			}
			return true
		}
		return false
	}
	if destResolved.kind == renvoTypeString {
		if !renvoEmitStringValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, offset, offset-8)
		return true
	}
	if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(destResolved.kind) {
		return renvoEmitWideExprToLocal(g, ep, idx, offset, destResolved.kind)
	}
	if renvoTypeKindIsScalarValue(destResolved.kind) || destResolved.kind == renvoTypePointer || destResolved.kind == renvoTypeFunc {
		if destResolved.kind == renvoTypePointer && e.kind == renvoExprComposite && e.nameStart == e.nameEnd {
			if !renvoEmitTypedPointerCompositeLiteral(g, ep, idx, destResolved.elem) {
				return false
			}
		} else if !renvoEmitScalarExprForKind(g, ep, idx, destResolved.kind) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, offset)
		return true
	}
	if !renvoTypeIsSlice(meta, destType) {
		return false
	}
	// A nested composite literal may omit its type. Its enclosing collection
	// provides the element type through this destination slot.
	if e.kind == renvoExprComposite && e.nameStart == e.nameEnd {
		if !renvoEmitSliceLiteralRegs(g, ep, idx, destType) {
			return false
		}
	} else if !renvoEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	renvoAsmStoreSliceStack(&g.asm, offset)
	return true
}

func renvoCompositeNeedsTemporary(g *renvoLinearGen, ep *renvoExprParse, composite *renvoExpr, offset int) bool {
	for i := 0; i < composite.argCount; i++ {
		field := &ep.fields[composite.firstArg+i]
		if renvoExprMayObserveLocal(g, ep, field.expr, offset) {
			return true
		}
	}
	return false
}

func renvoExprMayObserveLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return true
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		return localIndex >= 0 && g.locals[localIndex].offset == offset
	}
	// Calls and indirect memory reads can observe an escaped destination even
	// when it is not named directly in this expression.
	if e.kind == renvoExprCall || e.kind == renvoExprIndex || e.kind == renvoExprAssert || e.kind == renvoExprFunc {
		return true
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(g.prog, e.tok, '*') {
			return true
		}
		return renvoExprMayObserveLocal(g, ep, e.left, offset)
	}
	if e.kind == renvoExprBinary {
		return renvoExprMayObserveLocal(g, ep, e.left, offset) ||
			renvoExprMayObserveLocal(g, ep, e.right, offset)
	}
	if e.kind == renvoExprSelector {
		return renvoExprMayObserveLocal(g, ep, e.left, offset)
	}
	if e.kind == renvoExprComposite {
		return renvoCompositeNeedsTemporary(g, ep, e, offset)
	}
	return false
}

func renvoExprIsNil(p *renvoProgram, e *renvoExpr) bool {
	if e.kind != renvoExprIdent {
		return false
	}
	return renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "nil")
}

func renvoEmitInterfaceAssignToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	renvoNonNil(g, ep)
	if idx >= 0 && idx < len(ep.exprs) {
		renvoRefreshCapturedExpr(g, ep, idx)
		e := &ep.exprs[idx]
		if renvoExprIsNil(g.prog, e) {
			renvoAsmStoreStackImm(&g.asm, offset, 0)
			renvoAsmStorePrimaryStack(&g.asm, offset-renvoBackendValueSlotSize)
			return true
		}
		if e.kind == renvoExprCall && renvoExprIdentCode(g.prog, ep, e.left) == renvoIdentRecover {
			if e.argCount != 0 {
				return false
			}
			return renvoEmitRecoverToLocal(g, offset)
		}
		if e.kind == renvoExprCall && e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 && renvoResolveType(g.meta, conversionType).kind == renvoTypeInterface {
				return renvoEmitInterfaceAssignToLocal(g, ep, ep.args[e.firstArg], offset)
			}
		}
	}
	sourceType := renvoInferParsedExprType(g, ep, idx)
	source := renvoResolveType(g.meta, sourceType)
	renvoNonNil(source)
	if source.kind == renvoTypeInterface {
		e := &ep.exprs[idx]
		if e.kind == renvoExprCall {
			return renvoEmitStructCallToLocal(g, ep, idx, sourceType, offset)
		}
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, sourceType))
		return true
	}
	if source.kind == renvoTypeComplex64 && !renvoInterfaceValueStoredIndirect(g.meta, sourceType) {
		if !renvoEmitComplexValueRegs(g, ep, idx) {
			return false
		}
		if g.c.renvoNativeIntSize == 4 {
			renvoAsmStorePrimarySecondaryStack(&g.asm, offset, offset-4)
		} else {
			renvoPackComplex64RegsPrimary(g)
			renvoAsmStorePrimaryStack(&g.asm, offset)
		}
		renvoAsmStoreStackImm(&g.asm, offset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, sourceType))
		return true
	}
	size := renvoTypeSize(g.meta, sourceType)
	if size < 0 {
		return false
	}
	if size == 0 {
		size = renvoBackendValueSlotSize
	}
	valueOffset := renvoAddUnnamedLocal(g, sourceType)
	if !renvoEmitExprToLocal(g, ep, idx, valueOffset) {
		return false
	}
	return renvoEmitConcreteLocalToInterface(g, sourceType, valueOffset, offset)
}

func renvoEmitConcreteLocalToInterface(g *renvoLinearGen, sourceType int, valueOffset int, offset int) bool {
	renvoNonNil(g)
	source := renvoResolveType(g.meta, sourceType)
	size := renvoTypeSize(g.meta, sourceType)
	if size < 0 {
		return false
	}
	if size == 0 {
		size = renvoBackendValueSlotSize
	}
	if !renvoInterfaceValueStoredIndirect(g.meta, sourceType) {
		if size <= g.c.renvoNativeIntSize &&
			(renvoTypeKindIsScalarValue(source.kind) || source.kind == renvoTypePointer || source.kind == renvoTypeFunc) {
			renvoAsmLoadPrimaryStack(&g.asm, valueOffset)
			if renvoTypeKindIsScalarValue(source.kind) {
				renvoAsmNormalizePrimaryForKind(&g.asm, source.kind)
			}
			renvoAsmStorePrimaryStack(&g.asm, offset)
		} else {
			if size < renvoBackendValueSlotSize {
				renvoAsmStoreStackImm(&g.asm, offset, 0)
			}
			renvoEmitCopyStackToStack(g, valueOffset, offset, size)
		}
	} else {
		sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreStackImm(&g.asm, sizeOffset, size)
		renvoEmitPersistentAllocToPrimary(g, sizeOffset)
		renvoAsmStorePrimaryStack(&g.asm, addrOffset)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyStackToMemSecondary(g, valueOffset, 0, size)
		renvoAsmLoadPrimaryStack(&g.asm, addrOffset)
		renvoAsmStorePrimaryStack(&g.asm, offset)
	}
	renvoAsmStoreStackImm(&g.asm, offset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, sourceType))
	return true
}

func renvoBinaryComparesInterface(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	if !renvoTok2Is(g.prog, e.tok, '=', '=') && !renvoTok2Is(g.prog, e.tok, '!', '=') {
		return false
	}
	if !renvoExprIsNil(g.prog, &ep.exprs[e.left]) {
		left := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
		renvoNonNil(left)
		if left.kind == renvoTypeInterface {
			return true
		}
	}
	if !renvoExprIsNil(g.prog, &ep.exprs[e.right]) {
		right := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right))
		renvoNonNil(right)
		return right.kind == renvoTypeInterface
	}
	return false
}

func renvoEmitInterfaceCompare(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	leftNil := renvoExprIsNil(g.prog, &ep.exprs[e.left])
	rightNil := renvoExprIsNil(g.prog, &ep.exprs[e.right])
	if leftNil || rightNil {
		// Nil equality only depends on the interface's dynamic type tag. The
		// general comparison ladder includes every comparable runtime type and
		// is both unnecessary and especially expensive in large programs.
		valueIndex := e.left
		if leftNil {
			valueIndex = e.right
		}
		value := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
		if !renvoEmitInterfaceAssignToLocal(g, ep, valueIndex, value) {
			return false
		}
		a := &g.asm
		renvoAsmPrimaryImm(a, 0)
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmLoadPrimaryStack(a, value-renvoBackendValueSlotSize)
		setcc := 0x94
		if renvoTok2Is(g.prog, e.tok, '!', '=') {
			setcc = 0x95
		}
		renvoAsmCmpTertiaryPrimarySet(a, setcc)
		return true
	}
	left := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	right := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitInterfaceAssignToLocal(g, ep, e.left, left) || !renvoEmitInterfaceAssignToLocal(g, ep, e.right, right) {
		return false
	}
	return renvoEmitInterfaceCompareLocals(g, left, right, renvoTok2Is(g.prog, e.tok, '!', '='))
}

// Switch tags are evaluated once and retained across the ordered case tests.
// Share the full dynamic-type and payload comparison with ordinary equality.
func renvoEmitInterfaceCompareLocals(g *renvoLinearGen, left int, right int, notEqual bool) bool {
	a := &g.asm
	indirectTypeBase := renvoInterfaceIndirectTypeBaseFor(g.meta)
	different := renvoAsmNewLabel(a)
	nonNil := renvoAsmNewLabel(a)
	indirect := renvoAsmNewLabel(a)
	nonComparable := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	renvoAsmJcmpStackStack(a, left-renvoBackendValueSlotSize, right-renvoBackendValueSlotSize, different, 0x95)
	renvoAsmLoadPrimaryStack(a, left-renvoBackendValueSlotSize)
	renvoAsmJnzPrimary(a, nonNil)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmJmpLabel(a, done)
	renvoAsmMarkLabel(a, nonNil)
	renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, 0, nonComparable, 0x9c)
	renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, indirectTypeBase, indirect, 0x9d)
	for typ := 1; typ < len(g.meta.types); typ++ {
		tag := renvoRuntimeTypeTag(g.meta, typ)
		kind := renvoResolveType(g.meta, typ).kind
		// Compact prepared targets cannot represent the fixed-width floating
		// interface payloads. Do not make otherwise-valid interface equality
		// depend on unreachable x87/soft-float helpers merely because the builtin
		// types exist in metadata.
		if g.c.renvoNativeIntSize == 2 && (renvoTypeKindIsFloat(kind) || kind == renvoTypeComplex64) {
			continue
		}
		if tag <= 0 || tag >= indirectTypeBase ||
			!renvoTypeKindIsFloat(kind) && kind != renvoTypeComplex64 {
			continue
		}
		next := renvoAsmNewLabel(a)
		renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, tag, next, 0x95)
		if renvoTypeKindIsFloat(kind) {
			renvoEmit32IEEECompareStack(g, left, right, kind, '=', '=')
			renvoAsmJmpLabel(a, done)
		} else {
			leftValue := renvoAddUnnamedLocal(g, typ)
			rightValue := renvoAddUnnamedLocal(g, typ)
			if g.c.renvoNativeIntSize == 8 {
				renvoAsmAddressPrimaryStack(a, left)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoUnpackComplex64MemSecondaryRegs(g)
				renvoAsmStorePrimarySecondaryStack(a, leftValue, renvoComplexSecondaryStackOffset(g, typ, leftValue))
				renvoAsmAddressPrimaryStack(a, right)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoUnpackComplex64MemSecondaryRegs(g)
				renvoAsmStorePrimarySecondaryStack(a, rightValue, renvoComplexSecondaryStackOffset(g, typ, rightValue))
			} else {
				renvoEmitCopyStackToStack(g, left, leftValue, 8)
				renvoEmitCopyStackToStack(g, right, rightValue, 8)
			}
			renvoEmitCompositeCompareAt(g, typ, leftValue, rightValue, different)
			renvoAsmPrimaryImm(a, 1)
			renvoAsmJmpLabel(a, done)
		}
		renvoAsmMarkLabel(a, next)
	}
	renvoAsmLoadPrimaryTertiaryStack(a, left, right)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJmpLabel(a, done)
	renvoAsmMarkLabel(a, indirect)
	for typ := 1; typ < len(g.meta.types); typ++ {
		tag := renvoRuntimeTypeTag(g.meta, typ)
		kind := renvoResolveType(g.meta, typ).kind
		if g.c.renvoNativeIntSize == 2 && (renvoTypeKindIsFloat(kind) || renvoTypeKindIsComplex(kind)) {
			continue
		}
		if tag != indirectTypeBase+typ || kind == renvoTypeInterface {
			continue
		}
		next := renvoAsmNewLabel(a)
		renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, tag, next, 0x95)
		leftValue := renvoAddUnnamedLocal(g, typ)
		rightValue := renvoAddUnnamedLocal(g, typ)
		size := renvoTypeSize(g.meta, typ)
		if renvoResolveType(g.meta, typ).kind == renvoTypeComplex64 && g.c.renvoNativeIntSize == 8 {
			// Interfaces store complex64 densely in memory while native locals keep
			// one component in each internal value slot.
			renvoAsmLoadSecondaryStack(a, left)
			renvoUnpackComplex64MemSecondaryRegs(g)
			renvoAsmStorePrimarySecondaryStack(a, leftValue, renvoComplexSecondaryStackOffset(g, typ, leftValue))
			renvoAsmLoadSecondaryStack(a, right)
			renvoUnpackComplex64MemSecondaryRegs(g)
			renvoAsmStorePrimarySecondaryStack(a, rightValue, renvoComplexSecondaryStackOffset(g, typ, rightValue))
		} else {
			renvoAsmLoadSecondaryStack(a, left)
			renvoEmitCopyMemSecondaryToStack(g, leftValue, size)
			renvoAsmLoadSecondaryStack(a, right)
			renvoEmitCopyMemSecondaryToStack(g, rightValue, size)
		}
		renvoEmitCompositeCompareAt(g, typ, leftValue, rightValue, different)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmJmpLabel(a, done)
		renvoAsmMarkLabel(a, next)
	}
	renvoAsmPrimaryImm(a, 0)
	renvoAsmJmpLabel(a, done)
	renvoAsmMarkLabel(a, nonComparable)
	renvoEmitRuntimeFault(g)
	renvoAsmJmpMarkLabel(a, done, different)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmMarkLabel(a, done)
	if notEqual {
		renvoAsmBoolNotPrimary(a)
	}
	return true
}

func renvoTypeComparable(meta *renvoMeta, typ int) bool {
	renvoNonNil(meta)
	t := renvoResolveType(meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypeSlice || t.kind == renvoTypeFunc {
		return false
	}
	if t.kind == renvoTypeArray {
		return renvoTypeComparable(meta, t.elem)
	}
	if t.kind == renvoTypeStruct {
		for i := 0; i < t.count; i++ {
			if !renvoTypeComparable(meta, meta.fields[t.first+i].typ) {
				return false
			}
		}
	}
	return true
}

const renvoInterfaceIndirectTypeBase = 1048576
const renvoPanicTypeAssertionTag = 1048575
const renvoPanicOutOfMemoryTag = 1048574
const renvoPanicNilTag = 1048573

func renvoInterfaceIndirectTypeBaseFor(meta *renvoMeta) int {
	if meta != nil && meta.c.renvoNativeIntSize == 2 {
		// Runtime type tags travel in one native register. Keep the direct and
		// indirect ranges distinct after materialization on a 16-bit target.
		return 16384
	}
	return renvoInterfaceIndirectTypeBase
}

func renvoRuntimeTypeTag(meta *renvoMeta, typ int) int {
	renvoNonNil(meta)
	if typ <= 0 || typ >= len(meta.types) {
		return 0
	}
	t := meta.types[typ]
	// A tag embedded in generated code must keep its type identity after
	// function-local compiler scratch is reclaimed.
	if typ+1 > meta.runtimeTypeCount {
		meta.runtimeTypeCount = typ + 1
	}
	if t.kind == renvoTypeNamed && t.first == renvoNamedTypeAlias && t.elem > 0 {
		return renvoRuntimeTypeTag(meta, t.elem)
	}
	if !renvoTypeComparable(meta, typ) {
		return -typ
	}
	if renvoInterfaceValueStoredIndirect(meta, typ) {
		return renvoInterfaceIndirectTypeBaseFor(meta) + typ
	}
	return typ
}

func renvoInterfaceValueStoredIndirect(meta *renvoMeta, typ int) bool {
	renvoNonNil(meta)
	inlineSize := meta.c.renvoNativeIntSize
	if inlineSize <= 0 || inlineSize > renvoBackendValueSlotSize {
		inlineSize = renvoBackendValueSlotSize
	}
	return renvoTypeSize(meta, typ) > inlineSize
}

func renvoEmitSliceReturnValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int, resultType int) bool {
	renvoNonNil(g, ep)
	if !renvoEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	// An address-taken arena allocation must retain its backing address when
	// returning a view (for example a DMA-aligned subslice). Do not propagate
	// this return-only decision into assignment allocation/constant facts.
	e := &ep.exprs[idx]
	if e.kind == renvoExprSlice && renvoTypeIsSlice(g.meta, renvoInferParsedExprType(g, ep, e.left)) {
		base := &ep.exprs[e.left]
		if base.kind == renvoExprIdent {
			local := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
			if local >= 0 && !renvoLocalIsCurrentFuncParam(g, local) && g.locals[local].constValid != 0 && renvoLocalNameAddressTaken(g, base.nameStart, base.nameEnd) {
				return true
			}
		}
	}
	if renvoReturnedSliceCanReuseDescriptor(g, ep, idx) {
		return true
	}
	return renvoEmitCopySliceRegsToArena(g, resultType)
}

func renvoReturnedSliceCanReuseDescriptor(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	renvoNonNil(p)
	renvoNonNil(meta)
	if idx < 0 {
		return false
	}
	if idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprCall {
		callee := renvoExprIdentCode(p, ep, e.left)
		if callee == renvoIdentMake && e.argCount >= 2 {
			capacity := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+e.argCount-1))
			return !capacity.ok || capacity.value <= 0
		}
		if callee == renvoIdentAppend && e.argCount >= 1 {
			return renvoReturnedSliceCanReuseDescriptor(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 && fnIndex < len(meta.funcs) {
			fn := &meta.funcs[fnIndex]
			if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistBytes") ||
				renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistCheckNameRefs") ||
				renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistCheckSelectorRefs") ||
				renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistCheckTypeRefs") ||
				renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistCheckBools") {
				return true
			}
			if fn.receiverType != 0 {
				callee := &ep.exprs[e.left]
				if callee.kind != renvoExprSelector {
					return false
				}
				receiverType := renvoInferParsedExprType(g, ep, callee.left)
				if renvoTypeIsSlice(meta, receiverType) && !renvoReturnedSliceCanReuseDescriptor(g, ep, callee.left) {
					return false
				}
			}
			for i := 0; i < e.argCount; i++ {
				argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
				argType := renvoInferParsedExprType(g, ep, argIndex)
				if renvoTypeIsSlice(meta, argType) && !renvoReturnedSliceCanReuseDescriptor(g, ep, argIndex) {
					return false
				}
			}
			return true
		}
	}
	if e.kind != renvoExprIdent {
		return false
	}
	if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "nil") {
		return true
	}
	localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
	if localIndex < 0 {
		return true
	}
	return g.locals[localIndex].constValid != 0 || renvoLocalIsCurrentFuncParam(g, localIndex)
}

func renvoLocalIsCurrentFuncParam(g *renvoLinearGen, localIndex int) bool {
	renvoNonNil(g)
	if localIndex < 0 || localIndex >= g.localCount {
		return false
	}
	if g.currentFunc < 0 || g.currentFunc >= len(g.meta.funcs) {
		return false
	}
	local := &g.locals[localIndex]
	fn := &g.meta.funcs[g.currentFunc]
	for i := 0; i < fn.paramCount; i++ {
		param := &g.meta.params[fn.firstParam+i]
		if local.nameStart == param.nameStart && local.nameEnd == param.nameEnd {
			return true
		}
	}
	return false
}

func renvoEmitCopySliceRegsToArena(g *renvoLinearGen, sliceType int) bool {
	renvoNonNil(g)
	a := &g.asm
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	// Returned slices need enough spare capacity for the largest small
	// in-place metadata append. Target binding currently needs up to 70 bytes;
	// keeping 80 avoids copying a multi-megabyte linked unit at peak arena use.
	slackSize := 80
	if elemSize > slackSize {
		slackSize = elemSize
	}
	slackCapacity := slackSize / elemSize
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	capOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	copyCapOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	byteCountOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	allocSizeOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	nonNilLabel := renvoAsmNewLabel(a)
	capOKLabel := renvoAsmNewLabel(a)
	returnLabel := renvoAsmNewLabel(a)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoAsmStoreTertiaryStack(a, capOff)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmPushImm(a, slackCapacity)
	renvoAsmPopTertiary(a)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, copyCapOff)
	renvoAsmJcmpStackStack(a, capOff, copyCapOff, capOKLabel, 0x9e)
	renvoAsmCopyStackSlot(a, copyCapOff, capOff)
	renvoAsmMarkLabel(a, capOKLabel)
	renvoAsmLoadPrimaryStack(a, srcOff)
	renvoAsmJnzPrimary(a, nonNilLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmLoadSecondaryTertiaryStack(a, lenOff, capOff)
	renvoAsmJmpMarkLabel(a, returnLabel, nonNilLabel)
	renvoAsmLoadPrimaryStack(a, lenOff)
	if elemSize != 1 {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmStoreTertiaryStack(a, byteCountOff)
	} else {
		renvoAsmStorePrimaryStack(a, byteCountOff)
	}
	renvoAsmLoadPrimaryStack(a, capOff)
	if elemSize != 1 {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmCopyTertiaryToPrimary(a)
	}
	renvoAsmStorePrimaryStack(a, allocSizeOff)
	renvoEmitArenaAllocStackPrimary(g, allocSizeOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyToFreshArena(g, srcOff, destOff, byteCountOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	renvoAsmLoadTertiaryStack(a, capOff)
	renvoAsmMarkLabel(a, returnLabel)
	return true
}

func renvoEmitSliceValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if e.kind == renvoExprAssert {
		typ := renvoInferParsedExprType(g, ep, idx)
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(a, offset, offset-renvoBackendValueSlotSize)
		renvoAsmLoadTertiaryStack(a, offset-2*renvoBackendValueSlotSize)
		return true
	}
	if e.kind == renvoExprSlice {
		baseType := renvoInferParsedExprType(g, ep, e.left)
		baseResolved := renvoResolveType(meta, baseType)
		renvoNonNil(baseResolved)
		arrayType := baseResolved
		if baseResolved.kind == renvoTypePointer {
			arrayType = renvoResolveType(meta, baseResolved.elem)
		}
		if arrayType.kind == renvoTypeArray {
			if baseResolved.kind == renvoTypePointer {
				if !renvoEmitIntExpr(g, ep, e.left) {
					return false
				}
				renvoEmitRuntimeNonNilPrimary(g)
			} else if !renvoEmitAddressPrimary(g, ep, e.left) {
				return false
			}
			renvoAsmSecondaryImm(a, arrayType.count)
			renvoAsmCopySecondaryToTertiary(a)
		} else {
			if baseResolved.kind != renvoTypeSlice || !renvoEmitSliceValueRegs(g, ep, e.left) {
				return false
			}
		}
		if e.firstArg >= 0 || e.nameStart >= 0 || e.right >= 0 {
			elemSize := renvoTypeSize(meta, arrayType.elem)
			if elemSize < 1 {
				elemSize = 8
			}
			// Save a pointer/length/capacity descriptor even when the source is
			// a small array or pointer-to-array. Source-sized scratch storage
			// can be smaller than the three words stored below and overwrite
			// adjacent locals.
			baseOff := renvoAddUnnamedLocal(g, renvoInferParsedExprType(g, ep, idx))
			lowOff := renvoAddUnnamedLocal(g, renvoTypeInt)
			highOff := renvoAddUnnamedLocal(g, renvoTypeInt)
			maxOff := renvoAddUnnamedLocal(g, renvoTypeInt)
			renvoAsmStoreSliceStack(a, baseOff)
			if e.firstArg >= 0 {
				if !renvoEmitIntExpr(g, ep, e.firstArg) {
					return false
				}
			} else {
				renvoAsmPrimaryImm(a, 0)
			}
			renvoAsmStorePrimaryStack(a, lowOff)
			if e.right >= 0 {
				if !renvoEmitIntExpr(g, ep, e.right) {
					return false
				}
				renvoAsmStorePrimaryStack(a, highOff)
			} else {
				renvoAsmCopyStackSlot(a, baseOff-8, highOff)
			}
			if e.nameStart >= 0 {
				if !renvoEmitIntExpr(g, ep, e.nameStart) {
					return false
				}
				renvoAsmStorePrimaryStack(a, maxOff)
			} else {
				renvoAsmCopyStackSlot(a, baseOff-16, maxOff)
			}
			renvoEmitSliceBoundsChecks(g, lowOff, highOff, maxOff, baseOff-16)
			renvoAsmLoadPrimaryTertiaryStack(a, maxOff, lowOff)
			renvoAsmSubPrimaryTertiary(a)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryTertiaryStack(a, highOff, lowOff)
			renvoAsmSubPrimaryTertiary(a)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryTertiaryStack(a, baseOff, lowOff)
			renvoAsmAddScaledTertiary(a, elemSize)
			renvoAsmPopSecondary(a)
			renvoAsmPopTertiary(a)
			return true
		}
		return true
	}
	if e.kind == renvoExprIdent {
		if renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") && renvoFindLocalIndex(g, e.nameStart, e.nameEnd) < 0 && renvoFindGlobalType(g, e.nameStart, e.nameEnd) == 0 {
			renvoAsmPrimaryImm(a, 0)
			renvoAsmSecondaryImm(a, 0)
			renvoAsmCopySecondaryToTertiary(a)
			return true
		}
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !renvoTypeIsSlice(meta, globalType) {
				return false
			}
			renvoAsmLoadPrimaryBss(a, globalOffset+16)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset+8)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset)
			renvoAsmPopSecondary(a)
			renvoAsmPopTertiary(a)
			return true
		}
		if !renvoTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-8)
		renvoAsmLoadTertiaryStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == renvoExprIndex {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, valueType) || !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadSliceMemSecondary(a)
		return true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, valueType) {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadSliceMemSecondary(a)
		return true
	}
	if e.kind == renvoExprSelector {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, valueType) {
			return false
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadSliceMemSecondary(a)
		return true
	}
	if e.kind == renvoExprComposite {
		sliceType := renvoTypeFromExpr(g, ep, idx)
		if !renvoTypeIsSlice(meta, sliceType) {
			return false
		}
		return renvoEmitSliceLiteralRegs(g, ep, idx, sliceType)
	}
	if e.kind == renvoExprCall {
		prog := g.prog
		calleeLeft := e.left
		callee := renvoExprIdentCode(prog, ep, calleeLeft)
		if e.argCount >= 1 && callee == renvoIdentAppend {
			source := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			typ := renvoInferParsedExprType(g, ep, source)
			if !renvoTypeIsSlice(meta, typ) || !renvoEmitSliceValueRegs(g, ep, source) {
				return false
			}
			offset := renvoAddUnnamedLocal(g, typ)
			renvoAsmStoreSliceStack(a, offset)
			loc := renvoSliceLocation{offset: offset, typ: typ, ok: true}
			var stmt renvoStmt
			if !renvoEmitAppendToLocation(g, &stmt, ep, ep, &loc, e) {
				return false
			}
			renvoAsmLoadPrimarySecondaryStack(a, offset, offset-8)
			renvoAsmLoadTertiaryStack(a, offset-16)
			return true
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == renvoIdentMake {
				return renvoEmitMakeSliceRegs(g, ep, idx)
			}
		}
		if e.argCount == 1 {
			conversion := renvoResolveType(meta, renvoConversionTypeFromExpr(g, ep, calleeLeft))
			renvoNonNil(conversion)
			argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			if conversion.kind == renvoTypeSlice && renvoExprIsNil(prog, &ep.exprs[argIndex]) {
				return renvoEmitSliceValueRegs(g, ep, argIndex)
			}
			argType := renvoInferParsedExprType(g, ep, argIndex)
			argResolved := renvoResolveType(meta, argType)
			if conversion.kind == renvoTypeSlice && argResolved.kind == renvoTypeSlice && conversion.elem == argResolved.elem {
				return renvoEmitSliceValueRegs(g, ep, argIndex)
			}
			if conversion.kind == renvoTypeSlice && renvoTypeIsString(meta, argType) {
				elem := renvoResolveType(meta, conversion.elem)
				renvoNonNil(elem)
				if elem.kind == renvoTypeByte {
					return renvoEmitByteSliceConversionRegs(g, ep, idx)
				}
				if elem.kind == renvoTypeInt32 {
					return renvoEmitRuneSliceConversionRegs(g, ep, idx)
				}
			}
		}
		callType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, callType) {
			return false
		}
		if !renvoEmitIntExpr(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func renvoEmitStringSliceValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind != renvoExprSlice {
		return false
	}
	baseType := renvoInferParsedExprType(g, ep, e.left)
	if !renvoTypeIsString(meta, baseType) {
		return false
	}
	if !renvoEmitStringValueRegs(g, ep, e.left) {
		return false
	}
	if e.nameStart >= 0 {
		return false
	}
	if e.firstArg >= 0 || e.right >= 0 {
		baseOff := renvoAddUnnamedLocal(g, renvoTypeString)
		lowOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		highOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStorePrimaryStack(a, baseOff)
		renvoAsmCopySecondaryToPrimary(a)
		renvoAsmStorePrimaryStack(a, baseOff-8)
		if e.firstArg >= 0 {
			if !renvoEmitIntExpr(g, ep, e.firstArg) {
				return false
			}
		} else {
			renvoAsmPrimaryImm(a, 0)
		}
		renvoAsmStorePrimaryStack(a, lowOff)
		if e.right >= 0 {
			if !renvoEmitIntExpr(g, ep, e.right) {
				return false
			}
			renvoAsmStorePrimaryStack(a, highOff)
		} else {
			renvoAsmCopyStackSlot(a, baseOff-8, highOff)
		}
		renvoEmitSliceBoundsChecks(g, lowOff, highOff, highOff, baseOff-8)
		renvoAsmLoadPrimaryTertiaryStack(a, highOff, lowOff)
		renvoAsmSubPrimaryTertiary(a)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryTertiaryStack(a, baseOff, lowOff)
		renvoAsmAddPrimaryTertiary(a)
		renvoAsmPopSecondary(a)
		return true
	}
	return true
}
func renvoEmitSliceLiteralRegs(g *renvoLinearGen, ep *renvoExprParse, idx int, sliceType int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	needSize := e.argCount * elemSize
	backingSize := renvoStaticSliceBackingSize(needSize, elemSize)
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	backingAddrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(a, sizeOffset, backingSize)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(a, backingAddrOffset)
	if !renvoEmitSliceLiteralBacking(g, ep, idx, sliceType, backingAddrOffset) {
		return false
	}
	renvoAsmPrimaryImm(a, e.argCount)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryStack(a, backingAddrOffset)
	renvoAsmSecondaryImm(a, e.argCount)
	renvoAsmPopTertiary(a)
	return true
}
func renvoEmitSliceLiteralBacking(g *renvoLinearGen, ep *renvoExprParse, idx int, sliceType int, backingAddrOffset int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elemType := t.elem
	elemResolved := renvoResolveType(g.meta, elemType)
	renvoNonNil(elemResolved)
	elemSize := renvoTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		if field.nameEnd > field.nameStart {
			return false
		}
		disp := i * elemSize
		if elemResolved.kind == renvoTypeString {
			if !renvoEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			renvoAsmPushStringRegs(a)
			renvoAsmLoadSecondaryStack(a, backingAddrOffset)
			renvoAsmPopStoreStringMemSecondary(a, disp)
			continue
		}
		if elemResolved.kind == renvoTypeInterface {
			tempOffset := renvoAddUnnamedLocal(g, elemType)
			if !renvoEmitInterfaceAssignToLocal(g, ep, field.expr, tempOffset) {
				return false
			}
			renvoAsmLoadSecondaryStack(a, backingAddrOffset)
			if disp != 0 {
				renvoAsmAddSecondaryImm(a, disp)
			}
			renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
			continue
		}
		if elemResolved.kind == renvoTypeStruct {
			fieldAddrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
			renvoAsmLoadSecondaryStack(a, backingAddrOffset)
			if disp != 0 {
				renvoAsmAddSecondaryImm(a, disp)
			}
			renvoAsmStoreSecondaryStack(a, fieldAddrOffset)
			if !renvoEmitCompositeFieldToMem(g, ep, field.expr, elemType, fieldAddrOffset, 0) {
				return false
			}
			continue
		}
		if elemResolved.kind == renvoTypeArray || elemResolved.kind == renvoTypeSlice || elemResolved.kind == renvoTypePointer && ep.exprs[field.expr].kind == renvoExprComposite || g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(elemResolved.kind) {
			tempOffset := renvoAddUnnamedLocal(g, elemType)
			if !renvoEmitTypedAssign(g, ep, field.expr, tempOffset) {
				return false
			}
			renvoAsmLoadSecondaryStack(a, backingAddrOffset)
			if disp != 0 {
				renvoAsmAddSecondaryImm(a, disp)
			}
			renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
			continue
		}
		if !renvoTypeKindIsScalarValue(elemResolved.kind) && elemResolved.kind != renvoTypePointer && elemResolved.kind != renvoTypeFunc {
			return false
		}
		if renvoTypeKindIsFloat(elemResolved.kind) {
			if !renvoEmitScalarExprForKind(g, ep, field.expr, elemResolved.kind) {
				return false
			}
		} else if !renvoEmitIntExpr(g, ep, field.expr) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(a, elemResolved.kind)
		renvoAsmPushPrimary(a)
		renvoAsmLoadSecondaryStack(a, backingAddrOffset)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, disp, elemSize)
	}
	return true
}
func renvoEmitMakeSliceRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 && e.argCount != 3 {
		return false
	}
	sliceType := renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	lenOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	capOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmStorePrimaryStack(a, lenOffset)
	if e.argCount == 3 {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+2)) {
			return false
		}
		renvoAsmStorePrimaryStack(a, capOffset)
	} else {
		renvoAsmCopyStackSlot(a, lenOffset, capOffset)
	}
	// Constant bounds do not prove that previous allocations are dead. Every
	// evaluation needs fresh backing storage, including selector assignments.
	capacityArg := e.firstArg + e.argCount - 1
	capacity := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, capacityArg))
	length := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1))
	if capacity.ok && length.ok && length.value >= 0 && length.value <= capacity.value && capacity.value > 0 && capacity.value <= 1073741824/elemSize {
		size := capacity.value * elemSize
		renvoEmitMakeStaticRingPrimary(g, size, size)
	} else {
		sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmLoadTertiaryStack(a, capOffset)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmCopyTertiaryToPrimary(a)
		renvoAsmStorePrimaryStack(a, sizeOffset)
		renvoEmitArenaAllocStackPrimary(g, sizeOffset)
		renvoAsmLoadTertiaryStack(a, sizeOffset)
		renvoEmitMakeZero(g)
	}
	renvoAsmLoadSecondaryTertiaryStack(a, lenOffset, capOffset)
	return true
}

func renvoEnsureMakeZeroHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.makeZeroEmitted {
		return g.makeZeroLabel
	}
	g.makeZeroEmitted = true
	g.makeZeroLabel = renvoAsmNewLabel(a)
	if renvoRTGStructuredFunctions != 0 {
		renvoQueueStructuredHelper(g, renvoStructuredHelperMakeZero, 0, g.makeZeroLabel)
		return g.makeZeroLabel
	}
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.makeZeroLabel)
	if !renvoEmitOptimizedMakeZeroHelper(g) {
		renvoEmitMakeZeroHelperBody(g)
	}
	renvoAsmMarkLabel(a, afterLabel)
	return g.makeZeroLabel
}

// Keep the widest retired low range and the lowest retired persistent range.
// These boundaries describe potentially dirty memory even after later rewinds
// move either allocation cursor in the opposite direction.
func renvoEmitArenaRememberReset(g *renvoLinearGen, persistent bool) {
	a := &g.asm
	cursor := g.stringHeapOff
	dirty := cursor + 16
	condition := 0x97
	if persistent {
		cursor = g.stringHeapEndOff
		dirty = g.stringHeapOff + 24
		condition = 0x92
	}
	done := renvoAsmNewLabel(a)
	record := renvoAsmNewLabel(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, cursor)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmLoadPrimaryBss(a, dirty)
	if persistent {
		renvoAsmJzPrimary(a, record)
	}
	renvoAsmCmpTertiaryPrimarySet(a, condition)
	renvoAsmJzPrimary(a, done)
	renvoAsmMarkLabel(a, record)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryBss(a, dirty)
	renvoAsmMarkLabel(a, done)
	renvoAsmPopPrimary(a)
}

// Retain the historical helper entry point, but allocate fresh backing storage
// through the same arena as dynamic-capacity make calls.
func renvoEmitMakeStaticRingPrimary(g *renvoLinearGen, backingSize int, zeroSize int) {
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, backingSize)
	renvoEmitArenaAllocStackPrimary(g, sizeOffset)
	if zeroSize > 0 {
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoAsmPrimaryImm(&g.asm, zeroSize)
		renvoAsmCopyPrimaryToTertiary(&g.asm)
		renvoAsmCopySecondaryToPrimary(&g.asm)
		renvoEmitMakeZero(g)
	}
}

func renvoEmitByteSliceConversionRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	idxOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	if !renvoEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoEmitArenaAllocStackPrimary(g, lenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoAsmStoreStackImm(a, idxOff, 0)
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmJgeStackStack(a, idxOff, lenOff, doneLabel)
	renvoAsmPushStack(a, idxOff)
	renvoAsmLoadPrimaryStack(a, srcOff)
	renvoAsmPopTertiary(a)
	renvoAsmLoadBytePrimaryIndexTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmPushStack(a, idxOff)
	renvoAsmLoadSecondaryStack(a, destOff)
	renvoAsmPopTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStoreByteMemSecondaryTertiary(a)
	renvoAsmIncStack(a, idxOff)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	renvoAsmCopySecondaryToTertiary(a)
	return true
}

func renvoEmitRuneSliceConversionRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	runeOff := renvoAddUnnamedLocal(g, renvoTypeInt32)
	widthOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	sliceType := renvoInferParsedExprType(g, ep, idx)
	destOff := renvoAddUnnamedLocal(g, sliceType)
	renvoZeroLocalAtOffset(g, destOff)
	loc := renvoSliceLocation{offset: destOff, typ: sliceType, ok: true}
	if !renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoAsmStoreStackImm(a, indexOff, 0)
	loop := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loop)
	renvoAsmJgeStackStack(a, indexOff, lenOff, done)
	renvoEmitStringRangeDecode(g, srcOff, lenOff, indexOff, runeOff, widthOff)
	renvoAsmPushStack(a, runeOff)
	if !renvoEmitAppendDestPrimary(g, ep, &loc, 4) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, 4)
	renvoAsmLoadPrimaryTertiaryStack(a, indexOff, widthOff)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, indexOff)
	renvoAsmJmpMarkLabel(a, loop, done)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, destOff-8)
	renvoAsmLoadTertiaryStack(a, destOff-16)
	return true
}
func renvoEmitCompositeFieldToStack(g *renvoLinearGen, ep *renvoExprParse, idx int, fieldType int, destOffset int) bool {
	renvoNonNil(g, ep)
	fieldResolved := renvoResolveType(g.meta, fieldType)
	renvoNonNil(fieldResolved)
	if fieldResolved.kind == renvoTypeArray {
		e := &ep.exprs[idx]
		if e.kind != renvoExprComposite {
			tempOffset := renvoAddUnnamedLocal(g, fieldType)
			if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
				return false
			}
			renvoEmitCopyStackToStack(g, tempOffset, destOffset, renvoTypeSize(g.meta, fieldType))
			return true
		}
		elemSize := renvoTypeSize(g.meta, fieldResolved.elem)
		next := 0
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			at := next
			if field.key >= 0 {
				key := renvoEvalConstExpr(g, ep, field.key)
				if !key.ok {
					return false
				}
				at = key.value
			}
			if at < 0 || at >= fieldResolved.count {
				return false
			}
			if !renvoEmitCompositeFieldToStack(g, ep, field.expr, fieldResolved.elem, destOffset-at*elemSize) {
				return false
			}
			next = at + 1
		}
		return true
	}
	a := &g.asm
	if fieldResolved.kind == renvoTypeSlice {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmStoreSliceStack(a, destOffset)
		return true
	}
	if fieldResolved.kind == renvoTypeString {
		if !renvoEmitStringValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, destOffset, destOffset-8)
		return true
	}
	if fieldResolved.kind == renvoTypeComplex64 {
		if !renvoEmitComplexValueRegsForKind(g, ep, idx, fieldResolved.kind) {
			return false
		}
		if g.c.renvoNativeIntSize == 4 && renvoUsesStackIEEEFloat(&g.asm) {
			renvoAsmStorePrimaryStack(a, destOffset)
			renvoAsmCopySecondaryToPrimary(a)
			renvoAsmStorePrimaryStack(a, destOffset-4)
			return true
		}
		renvoPackComplex64RegsPrimary(g)
		renvoAsmStorePrimaryStack(a, destOffset)
		return true
	}
	if fieldResolved.kind == renvoTypeStruct || fieldResolved.kind == renvoTypeInterface || fieldResolved.kind == renvoTypePointer && ep.exprs[idx].kind == renvoExprComposite || renvoTypeKindIsComplex(fieldResolved.kind) || g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(fieldResolved.kind) {
		tempOffset := renvoAddUnnamedLocal(g, fieldType)
		if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := renvoTypeSize(g.meta, fieldType)
		renvoEmitCopyStackToStack(g, tempOffset, destOffset, size)
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, idx, fieldResolved.kind) {
		return false
	}
	renvoAsmStorePrimaryStackSize(a, destOffset, renvoNativeScalarStorageSize(g.c.renvoNativeIntSize, fieldResolved.kind))
	return true
}
func renvoEmitCopyStackToStack(g *renvoLinearGen, srcOffset int, destOffset int, size int) {
	renvoNonNil(g)
	// The backend chooses when bulk copies beat straight-line word copies.
	if renvoPreferBulkStackCopy(g, size) {
		source := renvoAddUnnamedLocal(g, renvoTypeInt)
		destination := renvoAddUnnamedLocal(g, renvoTypeInt)
		count := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmAddressPrimaryStack(&g.asm, srcOffset)
		renvoAsmStorePrimaryStack(&g.asm, source)
		renvoAsmAddressPrimaryStack(&g.asm, destOffset)
		renvoAsmStorePrimaryStack(&g.asm, destination)
		renvoAsmStoreStackImm(&g.asm, count, size)
		renvoEmitCopyBytes(g, source, destination, count)
		return
	}
	renvoEmitCopyNative(g, srcOffset, destOffset, size, renvoNativeCopyStackToStack)
}
func renvoEmitCopyStackToMemSecondary(g *renvoLinearGen, srcOffset int, destDisp int, size int) {
	renvoNonNil(g)
	renvoEmitCopyNative(g, srcOffset, destDisp, size, renvoNativeCopyStackToMem)
}
func renvoEmitCopyMemSecondaryToStack(g *renvoLinearGen, destOffset int, size int) {
	renvoNonNil(g)
	renvoEmitCopyNative(g, 0, destOffset, size, renvoNativeCopyMemToStack)
}

func renvoEmitRTGCopyAddressRange(g *renvoLinearGen, sourceOffset int, sourceDisp int, destOffset int, destDisp int, size int) {
	renvoNonNil(g)
	a := &g.asm
	for at := 0; at < size; at += g.c.renvoNativeIntSize {
		chunkSize := g.c.renvoNativeIntSize
		if size-at < chunkSize {
			chunkSize = size - at
		}
		renvoAsmLoadSecondaryStack(a, sourceOffset)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, sourceDisp+at, chunkSize)
		renvoAsmLoadSecondaryStack(a, destOffset)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, destDisp+at, chunkSize)
	}
}

func renvoEmitRTGCopyStructAddressToAddress(g *renvoLinearGen, typ int, sourceOffset int, destOffset int) {
	renvoNonNil(g)
	resolved := renvoResolveType(g.meta, typ)
	renvoNonNil(resolved)
	for i := 0; i < resolved.count; i++ {
		field := &g.meta.fields[resolved.first+i]
		renvoEmitRTGCopyAddressRange(g, sourceOffset, field.offset, destOffset, field.offset, renvoTypeSize(g.meta, field.typ))
	}
}

const renvoNativeCopyStackToStack = 1
const renvoNativeCopyStackToMem = 2
const renvoNativeCopyMemToStack = 3
const renvoNativeCopyStackToBSS = 4
const renvoNativeCopyBSSToStack = 5

func renvoEmitCopyNative(g *renvoLinearGen, srcOffset int, destOffset int, size int, mode int) {
	renvoNonNil(g)
	// Large aggregate loads and stores use the existing overlap-safe copy
	// operation instead of expanding a load/store pair for every word.
	if renvoPreferBulkIndirectCopy(g, size) &&
		(mode == renvoNativeCopyMemToStack || mode == renvoNativeCopyStackToMem) {
		source := renvoAddUnnamedLocal(g, renvoTypeInt)
		destination := renvoAddUnnamedLocal(g, renvoTypeInt)
		count := renvoAddUnnamedLocal(g, renvoTypeInt)
		if mode == renvoNativeCopyMemToStack {
			renvoAsmStoreSecondaryStack(&g.asm, source)
			renvoAsmAddressPrimaryStack(&g.asm, destOffset)
			renvoAsmStorePrimaryStack(&g.asm, destination)
		} else {
			renvoAsmAddSecondaryImm(&g.asm, destOffset)
			renvoAsmStoreSecondaryStack(&g.asm, destination)
			renvoAsmAddressPrimaryStack(&g.asm, srcOffset)
			renvoAsmStorePrimaryStack(&g.asm, source)
		}
		renvoAsmStoreStackImm(&g.asm, count, size)
		renvoEmitCopyBytes(g, source, destination, count)
		return
	}
	a := &g.asm
	for at := 0; at < size; {
		chunkSize := g.c.renvoNativeIntSize
		if size-at < chunkSize {
			chunkSize = size - at
		}
		// Scalar load/store emitters accept power-of-two widths. Split an
		// aggregate tail such as a three-byte array into 2+1 bytes instead of
		// accidentally selecting the native-width fallback and overwriting the
		// object immediately following it.
		if chunkSize > 4 && chunkSize < 8 {
			chunkSize = 4
		} else if chunkSize == 3 {
			chunkSize = 2
		}
		if mode == renvoNativeCopyMemToStack {
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, at, chunkSize)
		} else if mode == renvoNativeCopyBSSToStack {
			renvoAsmLoadPrimaryBss(a, srcOffset+at)
		} else {
			renvoAsmLoadPrimaryStack(a, srcOffset-at)
		}
		if mode == renvoNativeCopyStackToMem {
			renvoAsmStorePrimaryMemSecondaryDispSize(a, destOffset+at, chunkSize)
		} else if mode == renvoNativeCopyStackToBSS {
			renvoAsmStorePrimaryBss(a, destOffset+at)
		} else {
			// A narrow frame store uses secondary to address its destination.
			// Keep the source base across that store when a split tail still
			// needs another memory load (for example, a three-byte RGB value).
			preserveSource := mode == renvoNativeCopyMemToStack && chunkSize < g.c.renvoNativeIntSize && at+chunkSize < size
			if preserveSource {
				renvoAsmPushSecondary(a)
			}
			renvoAsmStorePrimaryStackSize(a, destOffset-at, chunkSize)
			if preserveSource {
				renvoAsmPopSecondary(a)
			}
		}
		at += chunkSize
	}
}

const renvoPushStack = 1
const renvoPushBss = 2

func renvoPrepareStructCall(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int) (int, int) {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	meta := g.meta
	renvoNonNil(meta)
	fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !renvoTypeUsesHiddenResult(meta, meta.funcs[fnIndex].resultType) {
		return -1, 0
	}
	if renvoTypeSize(meta, destType) != renvoTypeSize(meta, meta.funcs[fnIndex].resultType) {
		return -1, 0
	}
	fn := &meta.funcs[fnIndex]
	receiverIndex := -1
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != renvoExprSelector {
			return -1, 0
		}
		receiverIndex = callee.left
	}
	wordCount := 1
	if receiverIndex >= 0 {
		words := renvoEmitMethodReceiverArgReverse(g, ep, receiverIndex, meta.params[fn.firstParam].typ)
		if words < 0 {
			return -1, 0
		}
		wordCount += words
	}
	words := -1
	if renvoFixedTarget == 0 && g.c.objectFile && fn.linkStatic != 0 && receiverIndex < 0 &&
		renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "libc") {
		words = renvoEmitCObjectCallArgsReverse(g, ep, e, fn)
	} else {
		words = renvoEmitCallArgsReverse(g, ep, e, fn, receiverIndex)
	}
	if words < 0 {
		return -1, 0
	}
	wordCount += words
	return fnIndex, wordCount
}

func renvoEmitCallArgsReverse(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, fn *renvoFuncInfo, receiverIndex int) int {
	renvoNonNil(g, ep, e, fn)
	fixed := fn.paramCount
	firstParam := fn.firstParam
	if receiverIndex >= 0 {
		fixed--
		firstParam++
	}
	variadic := false
	if e.nameStart == 0 && fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart == 1 {
		fixed--
		if e.argCount < fixed {
			return -1
		}
		variadic = true
	} else {
		fixed = e.argCount
	}
	if fixed == 1 {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		typ := renvoInferParsedExprType(g, ep, arg)
		if renvoTypeIsTuple(g.meta, typ) {
			if renvoPreparedBackendActive != 0 {
				tupleParams := fn.paramCount
				if receiverIndex >= 0 {
					tupleParams--
				}
				return renvoEmitPreparedTupleParamArgsReverse(g, ep, arg, typ, firstParam, tupleParams)
			}
			return renvoEmitTupleArgReverse(g, ep, arg, typ)
		}
	}
	wordCount := 0
	for i := 0; i < fixed; i++ {
		words := renvoEmitCallParamArgReverse(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i), firstParam+i)
		if words < 0 {
			return -1
		}
		wordCount += words
	}
	if variadic {
		sliceType := g.meta.params[fn.firstParam+fn.paramCount-1].typ
		offset := renvoAddUnnamedLocal(g, sliceType)
		if !renvoEmitVariadicArgsToLocal(g, ep, e.firstArg+fixed, e.argCount-fixed, sliceType, offset) || renvoEmitTypedLocalArgReverse(g, offset, sliceType) != renvoBackendSliceWordCount {
			return -1
		}
		wordCount += renvoBackendSliceWordCount
	}
	return wordCount
}

func renvoEmitPreparedTupleParamArgsReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, typ int, firstParam int, paramCount int) int {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprCall {
		return -1
	}
	offset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := renvoResolveType(g.meta, typ)
	renvoNonNil(tuple)
	if tuple.count != paramCount || firstParam < 0 || firstParam+paramCount > len(g.meta.params) {
		return -1
	}
	wordCount := 0
	for i := 0; i < tuple.count; i++ {
		field := g.meta.fields[tuple.first+i]
		paramType := g.meta.params[firstParam+i].typ
		resolved := renvoResolveType(g.meta, paramType)
		renvoNonNil(resolved)
		if renvoStructArgByReference(g, resolved.kind) {
			renvoAsmAddressPrimaryStack(&g.asm, offset-field.offset)
			renvoAsmPushPrimary(&g.asm)
			wordCount++
			continue
		}
		size := renvoTypeCopySize(g.meta, paramType)
		wordSize := renvoBackendValueSlotSize
		usesCallWords := resolved.kind == renvoTypeArray || resolved.kind == renvoTypeStruct
		if g.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(resolved.kind) || resolved.kind == renvoTypeComplex) {
			usesCallWords = true
		}
		if usesCallWords {
			wordSize = renvoCallWordSize(g, paramType)
		}
		aligned := renvoAlignValue(size, wordSize)
		renvoEmitPushWords(g, offset-field.offset, aligned, wordSize, renvoPushStack)
		wordCount += aligned / wordSize
	}
	return wordCount
}

func renvoEmitStructCallToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int, offset int) bool {
	renvoNonNil(g, ep)
	if renvoIsInterfaceMethodCall(g, ep, idx) {
		return renvoEmitInterfaceMethodCall(g, ep, idx, offset, destType)
	}
	if renvoFunctionValueCalleeType(g, ep, ep.exprs[idx].left) != 0 {
		return renvoEmitFunctionValueCall(g, ep, idx, offset)
	}
	objectCABI := renvoFixedTarget == 0
	if renvoPreparedBackendActive != 0 {
		objectCABI = true
	}
	if objectCABI {
		if renvoEmitCObjectSmallAggregateCallToLocal(g, ep, idx, destType, offset) {
			return true
		}
	}
	fnIndex, wordCount := renvoPrepareStructCall(g, ep, idx, destType)
	if fnIndex < 0 {
		return false
	}
	renvoAsmAddressResultBuffer(&g.asm, offset)
	renvoAsmPushPrimary(&g.asm)
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoEmitCObjectSmallAggregateCallToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int, offset int) bool {
	if renvoFixedTarget != 0 {
		if renvoPreparedBackendActive == 0 {
			return false
		}
	}
	renvoNonNil(g, ep)
	if !renvoIsSysVObject(g.c) || idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if fn.linkStatic == 0 ||
		!renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "libc") ||
		!renvoObjectCABIIntegerAggregate(g.meta, fn.resultType) ||
		renvoTypeSize(g.meta, destType) != renvoTypeSize(g.meta, fn.resultType) {
		return false
	}
	wordCount := renvoEmitCObjectCallArgsReverse(g, ep, e, fn)
	if wordCount < 0 || !renvoCObjectReverseRegisterCallEligible(g, fn, wordCount) {
		return false
	}
	importID := renvoAsmAddPreparedStaticImport(&g.asm,
		fn.linkDLLStart, fn.linkDLLEnd,
		fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if importID < 0 {
		return false
	}
	if !renvoEmitCObjectReverseRegisterStaticCall(g, importID, wordCount) {
		return false
	}
	size := renvoTypeSize(g.meta, fn.resultType)
	primarySize := size
	if primarySize > 8 {
		primarySize = 8
	}
	renvoAsmStorePrimaryStackSize(&g.asm, offset, primarySize)
	if size > 8 {
		renvoAsmStoreSecondaryStack(&g.asm, offset-8)
	}
	return true
}
func renvoEmitUserCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if renvoIsInterfaceMethodCall(g, ep, idx) {
		return renvoEmitInterfaceMethodCall(g, ep, idx, 0, renvoInterfaceMethodCallResultType(g, ep, idx))
	}
	if renvoFunctionValueCalleeType(g, ep, e.left) != 0 {
		return renvoEmitFunctionValueCall(g, ep, idx, 0)
	}
	fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		if e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				resolved := renvoResolveType(g.meta, conversionType)
				renvoNonNil(resolved)
				if renvoTypeKindIsScalarValue(resolved.kind) {
					return renvoEmitScalarExprForKind(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), resolved.kind)
				}
			}
		}
		return renvoEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if renvoFixedTarget == 0 {
		if renvoIsSysVObject(g.c) && fn.resultCount == 0 && fn.literalTok <= 0 &&
			fn.linkStatic == 0 && fn.bodyStart == fn.bodyEnd &&
			!renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_") &&
			renvoCallArgumentsDiscardable(g, ep, e) {
			return true
		}
	}
	if renvoProgramUsesC11Semantics(g.prog) && e.argCount == 1 &&
		renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "__c_bool_int") {
		// The C frontend uses this helper to turn a Go bool into C's integer
		// 0/1 representation. Renvo boolean expressions already materialize
		// exactly those values, so a function call only adds ABI traffic.
		return renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	}
	if fn.nameEnd > fn.nameStart+15 &&
		renvo_runtime_UnsafeByteAt(g.prog.src, fn.nameStart+5) == '_' &&
		renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameStart+14, "renvo_runtime_") {
		if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
			result := renvoEmitRuntimePlatformIntrinsic(g, ep, e, fn)
			if result >= 0 {
				return result != 0
			}
		}
		if renvo_runtime_UnsafeByteAt(g.prog.src, fn.nameStart+14) == 'M' {
			size := int(renvo_runtime_UnsafeByteAt(g.prog.src, fn.nameStart+15) - '0')
			if size == 0 {
				size = g.c.renvoNativeIntSize
			}
			if e.argCount == 1 {
				return renvoEmitRuntimeUnsafeIndex(g, ep, e, size)
			}
			return renvoEmitRuntimeTruncateSlice(g, ep, e, size)
		}
		if renvoEmitRuntimeArenaCall(g, ep, idx, fn) {
			return true
		}
	}
	receiverIndex := -1
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != renvoExprSelector {
			return false
		}
		receiverIndex = callee.left
	}
	wordCount := 0
	if receiverIndex >= 0 {
		words := renvoEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			return false
		}
		wordCount += words
	}
	if renvoFixedTarget == 0 {
		return renvoEmitDynamicUserCallTail(g, ep, e, fn, fnIndex, receiverIndex, wordCount)
	}
	words := renvoEmitCallArgsReverse(g, ep, e, fn, receiverIndex)
	if words < 0 {
		return false
	}
	wordCount += words
	if fn.linkStatic != 0 && renvoTargetResolvesStaticImport(g.c, renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) == '/') {
		g.stackUsed = renvoAlignTo8(g.stackUsed + wordCount*renvoBackendValueSlotSize)
		renvoRecordStackPeak(g)
		tempBase := g.stackUsed
		for i := 0; i < wordCount; i++ {
			renvoAsmPopPrimary(&g.asm)
			renvoAsmStorePrimaryStack(&g.asm, tempBase-i*renvoBackendValueSlotSize)
		}
		word := 0
		for i := fn.paramCount - 1; i >= 0; i-- {
			typ := g.meta.params[fn.firstParam+i].typ
			resolved := renvoResolveType(g.meta, typ)
			renvoNonNil(resolved)
			callWords := 1
			if renvoPreparedBackendActive == 0 || !renvoStructArgByReference(g, resolved.kind) {
				callWordSize := renvoCallWordSize(g, typ)
				callWords = renvoAlignValue(renvoTypeCopySize(g.meta, typ), callWordSize) / callWordSize
			}
			renvoEmitPushWords(g, tempBase-word*renvoBackendValueSlotSize, callWords*renvoBackendValueSlotSize, renvoBackendValueSlotSize, renvoPushStack)
			word += callWords
		}
		if word != wordCount {
			return false
		}
		return renvoEmitTargetStaticCall(g, fn, wordCount) > 0
	}
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true

}

func renvoEmitDynamicUserCallTail(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, fn *renvoFuncInfo, fnIndex int, receiverIndex int, wordCount int) bool {
	cObjectForeign := g.c.objectFile && fn.linkStatic != 0 && receiverIndex < 0 &&
		renvoBytesEqualText(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd, "libc")
	if renvoFixedTarget == 0 {
		renvoRecordSingleCallConstants(g, ep, e, fnIndex)
	}
	words := -1
	if renvoFixedTarget == 0 && cObjectForeign {
		words = renvoEmitCObjectCallArgsReverse(g, ep, e, fn)
	} else {
		words = renvoEmitCallArgsReverse(g, ep, e, fn, receiverIndex)
	}
	if words < 0 {
		return false
	}
	wordCount += words
	if renvoFixedTarget == 0 && cObjectForeign && renvoProgramUsesC11Semantics(g.prog) && renvoCObjectReverseRegisterCallEligible(g, fn, wordCount) {
		importID := renvoAsmAddPreparedStaticImport(&g.asm,
			fn.linkDLLStart, fn.linkDLLEnd,
			fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
		if importID < 0 {
			return false
		}
		return renvoEmitCObjectReverseRegisterStaticCall(g, importID, wordCount)
	}
	if fn.linkStatic != 0 && renvoTargetResolvesStaticImport(g.c, renvo_runtime_UnsafeByteAt(g.prog.src, fn.linkDLLStart) == '/') {
		g.stackUsed = renvoAlignTo8(g.stackUsed + wordCount*renvoBackendValueSlotSize)
		renvoRecordStackPeak(g)
		tempBase := g.stackUsed
		for i := 0; i < wordCount; i++ {
			renvoAsmPopPrimary(&g.asm)
			renvoAsmStorePrimaryStack(&g.asm, tempBase-i*renvoBackendValueSlotSize)
		}
		word := 0
		for i := fn.paramCount - 1; i >= 0; i-- {
			typ := g.meta.params[fn.firstParam+i].typ
			resolved := renvoResolveType(g.meta, typ)
			renvoNonNil(resolved)
			callWords := 1
			if cObjectForeign && renvoTypeIsString(g.meta, typ) {
				callWords = 1
			} else if renvoPreparedBackendActive == 0 || !renvoStructArgByReference(g, resolved.kind) {
				callWordSize := renvoCallWordSize(g, typ)
				callWords = renvoAlignValue(renvoTypeCopySize(g.meta, typ), callWordSize) / callWordSize
			}
			renvoEmitPushWords(g, tempBase-word*renvoBackendValueSlotSize,
				callWords*renvoBackendValueSlotSize, renvoBackendValueSlotSize, renvoPushStack)
			word += callWords
		}
		if word != wordCount {
			return false
		}
		if renvoFixedTarget == 0 && cObjectForeign && wordCount > 6 && renvoIsSysVObject(g.c) {
			memoryAggregate := renvoEmitCObjectMemoryAggregateCall(g, fn, wordCount)
			if memoryAggregate >= 0 {
				return memoryAggregate != 0
			}
			return renvoEmitCObjectIntegerStackCall(g, fn, wordCount)
		}
		staticResult := renvoEmitTargetStaticCall(g, fn, wordCount)
		return staticResult > 0
	}
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoEmitRuntimePlatformIntrinsic(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, fn *renvoFuncInfo) int {
	irqStackPrefix1 := "renvo_runtime_CIRQStackCall1_"
	irqStackPrefix2 := "renvo_runtime_CIRQStackCall2_"
	if renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, irqStackPrefix1) ||
		renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, irqStackPrefix2) {
		return renvoBoolInt(renvoEmitIRQStackCall(g, ep, e, fn))
	}
	if renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_CMSABICall_") {
		return renvoBoolInt(renvoEmitMSABICall(g, ep, e, fn))
	}
	wideStringPointer := renvoBytesPrefixText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_CWideStringPointer")
	if e.argCount == 1 && (renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_CStringPointer") || wideStringPointer) {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if arg < 0 || arg >= len(ep.exprs) || ep.exprs[arg].kind != renvoExprString {
			return 0
		}
		msg := renvoDecodeStringToken(g.prog, ep.exprs[arg].tok)
		alignment := 1
		if wideStringPointer {
			alignment = int(renvo_runtime_UnsafeByteAt(g.prog.src, fn.nameEnd-1) - '0')
		}
		dataOffset := renvoAddStringDataAligned(g, msg, alignment)
		if dataOffset < 0 {
			return 0
		}
		renvoAsmPrimaryDataAddr(&g.asm, dataOffset)
		return 1
	}
	if e.argCount == 0 &&
		(renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_CMemoryBarrier") ||
			renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "renvo_runtime_CConditionClobber")) {
		// These are explicit operations in the shared stream. They constrain
		// scheduling and value reuse even though the current straight-line
		// backend needs no instruction bytes for either boundary.
		return 1
	}
	return renvoEmitTargetPlatformIntrinsic(g, ep, e, fn)
}

func renvoCObjectReverseRegisterCallEligible(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if !renvoIsSysVObject(g.c) || wordCount > 6 {
		return false
	}
	expectedWords := 0
	for i := 0; i < fn.paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		if renvoTypeIsString(g.meta, paramType) {
			expectedWords++
			continue
		}
		param := renvoResolveType(g.meta, paramType)
		if param.kind == renvoTypeStruct {
			if !renvoObjectCABIIntegerAggregate(g.meta, paramType) {
				return false
			}
			expectedWords += renvoAlignValue(renvoTypeSize(g.meta, paramType), 8) / 8
			continue
		}
		if !renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc {
			return false
		}
		expectedWords++
	}
	return wordCount == expectedWords
}

func renvoEmitCObjectReverseRegisterStaticCall(g *renvoLinearGen, importID int, wordCount int) bool {
	renvoNonNil(g)
	if wordCount < 0 || wordCount > renvoObjectArgumentRegisterCount(g.c) || importID < 0 {
		return false
	}
	return renvoAsmObjectReverseRegisterCall(&g.asm, importID, wordCount)
}

// renvoCallArgumentsDiscardable is deliberately narrower than general purity:
// it recognizes the literal and address expressions used as arguments to C
// feature stubs, while retaining calls whose discarded arguments could call,
// dereference, index, or otherwise trap.
func renvoCallArgumentsDiscardable(g *renvoLinearGen, ep *renvoExprParse, call *renvoExpr) bool {
	renvoNonNil(g, ep, call)
	for i := 0; i < call.argCount; i++ {
		if !renvoDiscardableExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+i)) {
			return false
		}
	}
	return true
}

func renvoDiscardableExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt || e.kind == renvoExprFloat || e.kind == renvoExprChar ||
		e.kind == renvoExprBool || e.kind == renvoExprIdent {
		return true
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(g.prog, e.tok, '*') {
			return false
		}
		return renvoDiscardableExpr(g, ep, e.left)
	}
	if e.kind == renvoExprBinary {
		return renvoDiscardableExpr(g, ep, e.left) && renvoDiscardableExpr(g, ep, e.right)
	}
	if e.kind == renvoExprCall && e.argCount == 1 &&
		(renvoConversionTypeFromExpr(g, ep, e.left) != 0 || renvoExprIsIdentText(g.prog, ep, e.left, "__c_bool_int")) {
		return renvoDiscardableExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	}
	return false
}

func renvoEmitCObjectIntegerStackCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	// The compact load/store sequence below uses an unsigned displacement byte.
	// Six register arguments plus sixteen stack words therefore fit without a
	// wider addressing form.
	if !renvoIsSysVObject(g.c) || wordCount != fn.paramCount || wordCount <= 6 || wordCount > 22 {
		return false
	}
	for i := 0; i < fn.paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		if renvoTypeIsString(g.meta, paramType) {
			continue
		}
		param := renvoResolveType(g.meta, paramType)
		if !renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc {
			return false
		}
	}
	externalID := renvoAsmAddExternalImportRange(&g.asm, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
	if externalID < 0 {
		return false
	}
	return renvoAsmObjectIntegerStackCall(&g.asm, externalID, wordCount)
}

func renvoEmitCObjectCallArgsReverse(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, fn *renvoFuncInfo) int {
	renvoNonNil(g, ep, e, fn)
	if e.argCount != fn.paramCount {
		return -1
	}
	wordCount := 0
	for i := 0; i < e.argCount; i++ {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
		param := fn.firstParam + i
		paramType := renvoResolveType(g.meta, g.meta.params[param].typ)
		if arg >= 0 && arg < len(ep.exprs) && ep.exprs[arg].kind == renvoExprString && paramType.kind == renvoTypePointer {
			if !renvoEmitStringValueRegs(g, ep, arg) {
				return -1
			}
			// C string literals decay to their first byte. The shared syntax keeps
			// them as strings so their storage stays compact until object emission.
			renvoAsmPushPrimary(&g.asm)
			wordCount++
			continue
		}
		if renvoTypeIsString(g.meta, g.meta.params[param].typ) {
			argType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
			if argType.kind == renvoTypePointer {
				if !renvoEmitIntExpr(g, ep, arg) {
					return -1
				}
			} else if !renvoEmitStringValueRegs(g, ep, arg) {
				return -1
			}
			// The C frontend uses string as a compact checked carrier for
			// NUL-terminated char pointers. Only its data word crosses SysV.
			renvoAsmPushPrimary(&g.asm)
			wordCount++
			continue
		}
		words := renvoEmitCallParamArgReverse(g, ep, arg, param)
		if words < 0 {
			return -1
		}
		wordCount += words
	}
	return wordCount
}

func renvoFunctionValueCalleeType(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	typ := 0
	if e.kind == renvoExprSelector {
		fnIndex, expression := renvoMethodSelectorInfo(g, ep, idx)
		if fnIndex >= 0 {
			if !expression {
				return 0
			}
			return renvoFunctionTypeFromInfoStart(g.meta, fnIndex, 0)
		}
		typ = renvoInferParsedExprType(g, ep, idx)
	} else if e.kind != renvoExprIdent {
		typ = renvoInferParsedExprType(g, ep, idx)
	} else {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			typ = g.locals[localIndex].typ
		} else {
			typ = renvoFindGlobalType(g, e.nameStart, e.nameEnd)
		}
	}
	if renvoResolveType(g.meta, typ).kind != renvoTypeFunc {
		return 0
	}
	return typ
}

func renvoMethodSelectorInfo(g *renvoLinearGen, ep *renvoExprParse, idx int) (int, bool) {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprSelector {
		return -1, false
	}
	base := &ep.exprs[e.left]
	if base.kind == renvoExprIdent && renvoFindLocalIndex(g, base.nameStart, base.nameEnd) < 0 && renvoFindGlobalType(g, base.nameStart, base.nameEnd) == 0 {
		typ := renvoFindTypeByRange(g, base.nameStart, base.nameEnd)
		if typ != 0 {
			fnIndex := renvoFindMethodByTypeAndName(g, typ, e.nameStart, e.nameEnd)
			if fnIndex >= 0 {
				return fnIndex, true
			}
		}
	}
	if base.kind == renvoExprUnary && renvoTokCharIs(g.prog, base.tok, '*') {
		pointee := &ep.exprs[base.left]
		if pointee.kind == renvoExprIdent && renvoFindLocalIndex(g, pointee.nameStart, pointee.nameEnd) < 0 {
			typ := renvoConversionTypeFromExpr(g, ep, e.left)
			if typ != 0 {
				fnIndex := renvoFindMethodByTypeAndName(g, typ, e.nameStart, e.nameEnd)
				if fnIndex >= 0 {
					return fnIndex, true
				}
			}
		}
	}
	baseType := renvoInferParsedExprType(g, ep, e.left)
	fnIndex := renvoFindMethodByTypeAndName(g, baseType, e.nameStart, e.nameEnd)
	if fnIndex >= 0 {
		return fnIndex, false
	}
	return -1, false
}

func renvoFindMethodByTypeAndName(g *renvoLinearGen, typ int, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	p := g.prog
	meta := g.meta
	renvoNonNil(p)
	renvoNonNil(meta)
	hash := renvoHashRange(p.src, nameStart, nameEnd)
	i := int(renvo_runtime_UnsafeInt32At(meta.funcBuckets, hash%len(meta.funcBuckets)))
	for i >= 0 {
		fn := &meta.funcs[i]
		if fn.receiverType != 0 && renvoBytesEqualRange(p.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) && renvoMethodReceiverTypeMatches(meta, typ, fn.receiverType) {
			return i
		}
		i = int(renvo_runtime_UnsafeInt32At(meta.funcNext, i))
	}
	return -1
}

func renvoTypesEquivalent(meta *renvoMeta, left int, right int) bool {
	renvoNonNil(meta)
	if left == right {
		return true
	}
	l := renvoResolveType(meta, left)
	renvoNonNil(l)
	r := renvoResolveType(meta, right)
	renvoNonNil(r)
	if l.kind != r.kind {
		return false
	}
	if l.kind == renvoTypePointer || l.kind == renvoTypeSlice {
		return renvoTypesEquivalent(meta, l.elem, r.elem)
	}
	if l.kind == renvoTypeArray {
		return l.count == r.count && renvoTypesEquivalent(meta, l.elem, r.elem)
	}
	if l.kind == renvoTypeStruct && (renvoTypeIsTuple(meta, left) || renvoTypeIsTuple(meta, right)) {
		if !renvoTypeIsTuple(meta, left) || !renvoTypeIsTuple(meta, right) || l.count != r.count {
			return false
		}
		for i := 0; i < l.count; i++ {
			if !renvoTypesEquivalent(meta, meta.fields[l.first+i].typ, meta.fields[r.first+i].typ) {
				return false
			}
		}
		return true
	}
	if l.kind == renvoTypeFunc {
		if l.count != r.count || l.resolved != r.resolved || !renvoTypesEquivalent(meta, l.elem, r.elem) {
			return false
		}
		for i := 0; i < l.count; i++ {
			if !renvoTypesEquivalent(meta, meta.fields[l.first+i].typ, meta.fields[r.first+i].typ) {
				return false
			}
		}
	}
	return true
}

const renvoFunctionValueDirect = 1
const renvoFunctionValueClosure = 2
const renvoFunctionValueMethodExpression = 3
const renvoFunctionValueBoundMethod = 4

func renvoFunctionValueMode(meta *renvoMeta, fnIndex int, funcType int) int {
	renvoNonNil(meta)
	if fnIndex < 0 || fnIndex >= len(meta.funcs) {
		return 0
	}
	fn := &meta.funcs[fnIndex]
	t := renvoResolveType(meta, funcType)
	renvoNonNil(t)
	if t.kind != renvoTypeFunc || !renvoTypesEquivalent(meta, fn.resultType, t.elem) {
		return 0
	}
	if fn.literalTok > 0 {
		if renvoFunctionParamsMatchType(meta, fn, t, 1) {
			return renvoFunctionValueClosure
		}
		return 0
	}
	if fn.receiverType != 0 {
		if renvoFunctionParamsMatchType(meta, fn, t, 0) {
			return renvoFunctionValueMethodExpression
		}
		if renvoFunctionParamsMatchType(meta, fn, t, 1) {
			return renvoFunctionValueBoundMethod
		}
		return 0
	}
	if renvoFunctionParamsMatchType(meta, fn, t, 0) {
		return renvoFunctionValueDirect
	}
	return 0
}

// renvoFunctionValueTag preserves the compact sequential tags used by normal
// whole-program binaries. REPL images need a stronger ABI: their function
// values can outlive one linked generation, while later generations are free
// to add packages and renumber the backend function table. Derive those tags
// from stable source identity instead.
func renvoFunctionValueTag(g *renvoLinearGen, fnIndex int) int {
	if renvoFixedTarget == 0 {
		return renvoReplFunctionValueTag(g, fnIndex)
	}
	return fnIndex + 1
}

func renvoReplFunctionValueTag(g *renvoLinearGen, fnIndex int) int {
	renvoNonNil(g)
	if len(g.replRestoreOffsets) == 0 || fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return fnIndex + 1
	}
	packageIndex := renvoObjectFunctionPackage(g, fnIndex)
	if packageIndex < 0 {
		literalTok := g.meta.funcs[fnIndex].literalTok
		if literalTok > 0 && literalTok < renvoTokCount(g.prog) {
			position := int(renvoTokStart(g.prog, literalTok))
			packages := renvoProgramPackages(g.prog)
			for i := 0; i < len(packages); i++ {
				if position >= packages[i].textStart && position < packages[i].textEnd {
					packageIndex = i
					break
				}
			}
		}
	}
	if packageIndex < 0 {
		return fnIndex + 1
	}
	pkg := renvoProgramPackages(g.prog)[packageIndex]
	fn := g.meta.funcs[fnIndex]
	a, b := renvoObjectHashInt(1879, 3761, pkg.pathKeyA)
	a, b = renvoObjectHashInt(a, b, pkg.pathKeyB)
	a, b = renvoObjectHashRange(a, b, g.prog.src, fn.nameStart, fn.nameEnd)
	if fn.declIndex >= 0 && fn.declIndex < len(g.prog.funcs) {
		decl := g.prog.funcs[fn.declIndex]
		if decl.receiverStart < decl.receiverEnd {
			start := int(renvoTokStart(g.prog, decl.receiverStart))
			end := int(renvoTokEnd(g.prog, decl.receiverEnd-1))
			a, b = renvoObjectHashRange(a, b, g.prog.src, start, end)
		}
	}
	if fn.nameStart == fn.nameEnd {
		a, b = renvoFunctionLiteralStableHash(g, fnIndex, a, b, pkg.textStart)
	}
	tag := (a ^ b) & 2147483647
	if tag == 0 {
		tag = 1
	}
	return tag
}

func renvoFunctionLiteralStableHash(g *renvoLinearGen, fnIndex int, a int, b int, packageTextStart int) (int, int) {
	renvoNonNil(g)
	fn := g.meta.funcs[fnIndex]
	literalTok := fn.literalTok
	if literalTok <= 0 {
		literalTok = fn.bodyStart
	}
	for i := 0; i < len(g.meta.globals); i++ {
		global := g.meta.globals[i]
		if literalTok < global.initStart || literalTok >= global.initEnd {
			continue
		}
		a, b = renvoObjectHashInt(a, b, 1)
		a, b = renvoObjectHashRange(a, b, g.prog.src, global.nameStart, global.nameEnd)
		return renvoObjectHashInt(a, b, renvoFunctionLiteralOrdinal(g, literalTok, global.initStart, global.initEnd))
	}
	for i := 0; i < len(g.meta.funcs); i++ {
		owner := g.meta.funcs[i]
		if owner.literalTok > 0 || literalTok < owner.bodyStart || literalTok >= owner.bodyEnd {
			continue
		}
		a, b = renvoObjectHashInt(a, b, 2)
		a, b = renvoObjectHashRange(a, b, g.prog.src, owner.nameStart, owner.nameEnd)
		if owner.declIndex >= 0 && owner.declIndex < len(g.prog.funcs) {
			decl := g.prog.funcs[owner.declIndex]
			if decl.receiverStart < decl.receiverEnd {
				start := int(renvoTokStart(g.prog, decl.receiverStart))
				end := int(renvoTokEnd(g.prog, decl.receiverEnd-1))
				a, b = renvoObjectHashRange(a, b, g.prog.src, start, end)
			}
		}
		return renvoObjectHashInt(a, b, renvoFunctionLiteralOrdinal(g, literalTok, owner.bodyStart, owner.bodyEnd))
	}
	if literalTok >= 0 && literalTok < renvoTokCount(g.prog) {
		return renvoObjectHashInt(a, b, int(renvoTokStart(g.prog, literalTok))-packageTextStart)
	}
	return renvoObjectHashInt(a, b, fnIndex)
}

func renvoFunctionLiteralOrdinal(g *renvoLinearGen, literalTok int, start int, end int) int {
	renvoNonNil(g)
	ordinal := 0
	for i := 0; i < len(g.meta.funcs); i++ {
		tok := g.meta.funcs[i].literalTok
		if tok > 0 && tok >= start && tok < end && tok < literalTok {
			ordinal++
		}
	}
	return ordinal
}

func renvoFunctionParamsMatchType(meta *renvoMeta, fn *renvoFuncInfo, t *renvoTypeInfo, first int) bool {
	renvoNonNil(meta, fn, t)
	if fn.paramCount-first != t.count || t.count > 0 && meta.params[fn.firstParam+fn.paramCount-1].initStart != t.resolved {
		return false
	}
	for i := 0; i < t.count; i++ {
		if !renvoTypesEquivalent(meta, meta.params[fn.firstParam+first+i].typ, meta.fields[t.first+i].typ) {
			return false
		}
	}
	return true
}

func renvoEmitFunctionValueCall(g *renvoLinearGen, ep *renvoExprParse, idx int, resultOffset int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	funcType := renvoFunctionValueCalleeType(g, ep, e.left)
	t := renvoResolveType(g.meta, funcType)
	renvoNonNil(t)
	if t.kind != renvoTypeFunc || !renvoCallMatchesFuncType(t, e) {
		return false
	}
	handleOffset := renvoAddUnnamedLocal(g, funcType)
	if !renvoEmitExprToLocal(g, ep, e.left, handleOffset) {
		return false
	}
	argOffsets := make([]int, t.count)
	if !renvoPrepareFunctionValueArgs(g, ep, e, t, argOffsets) {
		return false
	}
	if renvoIsHostedObject(g.c) {
		return renvoEmitCObjectFunctionPointerCall(g, t, handleOffset, argOffsets, resultOffset)
	}
	return renvoEmitFunctionValueDispatch(g, funcType, handleOffset, argOffsets, resultOffset, -1)
}

func renvoEmitCObjectFunctionPointerCall(g *renvoLinearGen, functionType *renvoTypeInfo, handleOffset int, argOffsets []int, resultOffset int) bool {
	renvoNonNil(g, functionType)
	if renvoFixedTarget == 0 && renvoIsCdeclObject(g.c) {
		return renvoEmitCdeclObjectFunctionPointerCall(g, functionType, handleOffset, argOffsets, resultOffset)
	}
	if functionType.resolved != 0 || len(argOffsets) > 20 || renvoPreparedBackendActive != 0 && len(argOffsets) > renvoRTGObjectRegisterCount() {
		return false
	}
	wordOffsets := make([]int, 0, len(argOffsets))
	hasAggregate := false
	for i := 0; i < len(argOffsets); i++ {
		paramType := g.meta.fields[functionType.first+i].typ
		param := renvoResolveType(g.meta, paramType)
		if param.kind == renvoTypeStruct {
			if renvoPreparedBackendActive != 0 || !renvoObjectCABIIntegerAggregate(g.meta, paramType) {
				return false
			}
			hasAggregate = true
			words := renvoAlignValue(renvoTypeSize(g.meta, paramType), 8) / 8
			for word := 0; word < words; word++ {
				wordOffsets = append(wordOffsets, argOffsets[i]-word*8)
			}
			continue
		}
		if !renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc {
			return false
		}
		wordOffsets = append(wordOffsets, argOffsets[i])
	}
	// A SysV aggregate is assigned wholly to registers or wholly to the stack.
	// Keep register exhaustion conservative; ordinary scalar stack calls retain
	// their existing twenty-word path.
	if len(wordOffsets) > 20 || hasAggregate && len(wordOffsets) > 6 {
		return false
	}
	registerWords := renvoObjectArgumentRegisterCount(g.c)
	for i := 0; i < len(wordOffsets) && i < registerWords; i++ {
		if !renvoAsmLoadObjectArgumentWord(&g.asm, i, wordOffsets[i]) {
			return false
		}
	}
	result := renvoResolveType(g.meta, functionType.elem)
	if functionType.elem != 0 && !renvoTypeKindIsScalarInt(result.kind) && result.kind != renvoTypePointer {
		return false
	}
	if len(wordOffsets) > registerWords {
		if !renvoEmitCObjectFunctionPointerIntegerStackCall(g, handleOffset, wordOffsets) {
			return false
		}
	} else {
		renvoAsmObjectIndirectRegisterCall(&g.asm, handleOffset)
	}
	if resultOffset != 0 && functionType.elem != 0 {
		renvoAsmStorePrimaryStack(&g.asm, resultOffset)
	}
	return true
}

func renvoEmitCObjectFunctionPointerIntegerStackCall(g *renvoLinearGen, handleOffset int, argOffsets []int) bool {
	renvoNonNil(g)
	if !renvoIsSysVObject(g.c) || len(argOffsets) <= 6 || len(argOffsets) > 20 {
		return false
	}
	return renvoAsmObjectIndirectStackCall(&g.asm, handleOffset, argOffsets)
}

func renvoCallMatchesFuncType(t *renvoTypeInfo, e *renvoExpr) bool {
	renvoNonNil(t, e)
	if t.resolved == 0 || e.nameStart != 0 {
		return e.argCount == t.count
	}
	return e.argCount >= t.count-1
}

func renvoPrepareFunctionValueArgs(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, t *renvoTypeInfo, argOffsets []int) bool {
	renvoNonNil(g, ep, e, t)
	fixed := t.count
	if t.resolved != 0 && e.nameStart == 0 {
		fixed--
	}
	for i := 0; i < fixed; i++ {
		paramType := g.meta.fields[t.first+i].typ
		argOffsets[i] = renvoAddUnnamedLocal(g, paramType)
		if !renvoEmitExprToLocal(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i), argOffsets[i]) {
			return false
		}
	}
	if fixed < t.count {
		paramType := g.meta.fields[t.first+t.count-1].typ
		argOffsets[t.count-1] = renvoAddUnnamedLocal(g, paramType)
		if !renvoEmitVariadicArgsToLocal(g, ep, e.firstArg+fixed, e.argCount-fixed, paramType, argOffsets[t.count-1]) {
			return false
		}
	}
	return true
}

func renvoEmitVariadicArgsToLocal(g *renvoLinearGen, ep *renvoExprParse, first int, count int, sliceType int, offset int) bool {
	renvoNonNil(g, ep)
	t := renvoResolveType(g.meta, sliceType)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		return false
	}
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	if count == 0 {
		renvoAsmStoreStackImm(&g.asm, addrOffset, 0)
	} else {
		sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreStackImm(&g.asm, sizeOffset, count*elemSize)
		renvoEmitPersistentAllocToPrimary(g, sizeOffset)
		renvoAsmStorePrimaryStack(&g.asm, addrOffset)
	}
	for i := 0; i < count; i++ {
		tempOffset := renvoAddUnnamedLocal(g, t.elem)
		if !renvoEmitExprToLocal(g, ep, renvo_runtime_UnsafeIntAt(ep.args, first+i), tempOffset) {
			return false
		}
		renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, i*elemSize, elemSize)
	}
	renvoAsmPrimaryImm(&g.asm, count)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmCopyPrimaryToTertiary(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, addrOffset)
	renvoAsmStoreSliceStack(&g.asm, offset)
	return true
}

func renvoEmitTypedLocalArgReverse(g *renvoLinearGen, offset int, typ int) int {
	renvoNonNil(g)
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if renvoPreparedBackendActive != 0 && renvoStructArgByReference(g, t.kind) {
		renvoAsmAddressPrimaryStack(&g.asm, offset)
		renvoAsmPushPrimary(&g.asm)
		return 1
	}
	size := renvoTypeSize(g.meta, typ)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	wordSize := renvoCallWordSize(g, typ)
	renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
	return renvoAlignValue(size, wordSize) / wordSize
}

func renvoCallWordSize(g *renvoLinearGen, typ int) int {
	t := renvoResolveType(g.meta, typ)
	if t.kind == renvoTypeArray ||
		g.c.renvoNativeIntSize == 2 && t.kind == renvoTypeStruct ||
		g.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(t.kind) || t.kind == renvoTypeComplex || t.kind == renvoTypeStruct && renvoTypeNeedsDenseCallWords(g.meta, typ)) {
		return g.c.renvoNativeIntSize
	}
	return renvoBackendValueSlotSize
}

func renvoTypeNeedsDenseCallWords(meta *renvoMeta, typ int) bool {
	if typ <= 0 || typ >= len(meta.types) {
		return false
	}
	t := renvoResolveType(meta, typ)
	return renvoTypeKindIsWideValue(t.kind) || t.kind == renvoTypeComplex || t.kind == renvoTypeArray || t.kind == renvoTypeStruct && t.resolved == renvoStructLayoutDense
}

func renvoEmitFunctionValueDispatch(g *renvoLinearGen, funcType int, handleOffset int, argOffsets []int, resultOffset int, directTarget int) bool {
	renvoNonNil(g)
	meta := g.meta
	renvoNonNil(meta)
	if directTarget < 0 {
		renvoAsmLoadPrimaryStack(&g.asm, handleOffset)
		renvoEmitRuntimeNonNilPrimary(g)
	}
	doneLabel := renvoAsmNewLabel(&g.asm)
	funcInfo := renvoResolveType(meta, funcType)
	renvoNonNil(funcInfo)
	closureTagOffset := -1
	previousDeferPendingOffset := 0
	if g.emittingDefers {
		previousDeferPendingOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmCopyThreadStateToStack(g, renvoThreadPanicDeferPendingOff, previousDeferPendingOffset)
	}
	hiddenResultOffset := resultOffset
	resultType := funcInfo.elem
	if renvoTypeUsesHiddenResult(meta, resultType) && hiddenResultOffset == 0 {
		hiddenResultOffset = renvoAddUnnamedLocal(g, resultType)
		renvoZeroLocalAtOffset(g, hiddenResultOffset)
	}
	// Direct values are integer tags, whereas bound methods and closures
	// hold pointers. Exhaust direct candidates before dereferencing a handle.
	for pass := 0; pass < 2; pass++ {
		for fnIndex := 0; fnIndex < len(meta.funcs); fnIndex++ {
			if directTarget >= 0 && fnIndex != directTarget {
				continue
			}
			mode := renvoFunctionValueMode(meta, fnIndex, funcType)
			direct := mode == renvoFunctionValueDirect || mode == renvoFunctionValueMethodExpression
			closure := mode == renvoFunctionValueClosure || mode == renvoFunctionValueBoundMethod
			if !direct && !closure || pass == 0 && !direct || pass == 1 && !closure {
				continue
			}
			if mode == renvoFunctionValueClosure {
				closureIndex := renvoClosureIndexByFunction(g.meta, fnIndex)
				if closureIndex >= 0 && !g.meta.closures[closureIndex].ready {
					literalTok := g.meta.funcs[fnIndex].literalTok
					parentReady := true
					for parent := 0; parent < len(g.meta.funcs); parent++ {
						fn := &g.meta.funcs[parent]
						if parent != fnIndex && literalTok >= fn.bodyStart && literalTok < fn.bodyEnd && (parent >= len(g.funcReachable) || !g.funcReachable[parent]) {
							parentReady = false
						}
					}
					if !parentReady {
						continue
					}
				}
			}
			compareOffset := handleOffset
			if closure {
				if closureTagOffset < 0 {
					closureTagOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
					renvoAsmLoadPrimaryStackMemory(&g.asm, handleOffset, 0)
					renvoAsmStorePrimaryStack(&g.asm, closureTagOffset)
				}
				compareOffset = closureTagOffset
			}
			nextLabel := renvoAsmNewLabel(&g.asm)
			tag := renvoFunctionValueTag(g, fnIndex)
			if directTarget < 0 {
				renvoAsmJcmpStackImm(&g.asm, compareOffset, tag, nextLabel, 0x95)
			}
			wordCount := 0
			extra := 0
			if mode == renvoFunctionValueClosure {
				renvoAsmPushStackWord(&g.asm, handleOffset)
				extra = 1
			} else if mode == renvoFunctionValueBoundMethod {
				receiverType := g.meta.params[g.meta.funcs[fnIndex].firstParam].typ
				receiverOffset := renvoAddUnnamedLocal(g, receiverType)
				renvoAsmLoadSecondaryStack(&g.asm, handleOffset)
				renvoAsmAddSecondaryImm(&g.asm, renvoBackendValueSlotSize)
				renvoEmitCopyMemSecondaryToStack(g, receiverOffset, renvoTypeCopySize(g.meta, receiverType))
				extra = renvoEmitTypedLocalArgReverse(g, receiverOffset, receiverType)
			}
			for i := 0; i < len(argOffsets); i++ {
				wordCount += renvoEmitTypedLocalArgReverse(g, argOffsets[i], g.meta.fields[funcInfo.first+i].typ)
			}
			if hiddenResultOffset > 0 {
				renvoAsmAddressPrimaryStack(&g.asm, hiddenResultOffset)
				renvoAsmPushPrimary(&g.asm)
				extra++
			}
			oldSuppress := g.suppressPanicCheck
			g.suppressPanicCheck = true
			if g.emittingDefers {
				renvoAsmPrimaryImm(&g.asm, 1)
				renvoAsmStorePrimaryThreadState(g, renvoThreadPanicDeferPendingOff)
			}
			renvoEmitCallWithWordCount(g, fnIndex, wordCount+extra)
			if g.emittingDefers {
				renvoAsmLoadPrimaryStack(&g.asm, previousDeferPendingOffset)
				renvoAsmStorePrimaryThreadState(g, renvoThreadPanicDeferPendingOff)
			}
			g.suppressPanicCheck = oldSuppress
			if mode == renvoFunctionValueClosure {
				renvoAsmPushSliceRegs(&g.asm)
				if !renvoReloadClosureCaptures(g, fnIndex, handleOffset) {
					return false
				}
				renvoAsmPopPrimary(&g.asm)
				renvoAsmPopSecondary(&g.asm)
				renvoAsmPopTertiary(&g.asm)
			}
			if !g.emittingDefers {
				renvoEmitPostCallPanicCheck(g)
			}
			renvoAsmJmpMarkLabel(&g.asm, doneLabel, nextLabel)
		}
	}

	// A nil handle, an invalid handle, or a function signature without any
	// concrete whole-program targets is still a valid call site. It faults only
	// if execution reaches it; guarded nil calls therefore compile normally.
	renvoEmitRuntimeFault(g)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmMarkLabel(&g.asm, doneLabel)
	return true
}

func renvoReloadClosureCaptures(g *renvoLinearGen, fnIndex int, _ int) bool {
	renvoNonNil(g)
	closureIndex := renvoClosureIndexByFunction(g.meta, fnIndex)
	if closureIndex < 0 {
		return true
	}
	info := &g.meta.closures[closureIndex]
	if !info.ready {
		// Calls through a closure returned by another function can be emitted
		// before that factory is reached by the function queue. There are no
		// caller locals to refresh in that case; the factory will establish the
		// capture layout before the closure body itself is emitted.
		return true
	}
	for i := 0; i < info.captureCount; i++ {
		capture := &g.meta.captures[info.firstCapture+i]
		localIndex := renvoFindLocalIndex(g, capture.nameStart, capture.nameEnd)
		if localIndex < 0 {
			continue
		}
		renvoMoveCapturedLocal(g, localIndex, false)
	}
	return true
}

func renvoEmitRuntimeArenaCall(g *renvoLinearGen, ep *renvoExprParse, idx int, fn *renvoFuncInfo) bool {
	renvoNonNil(g, ep, fn)
	p := g.prog
	renvoNonNil(p)
	intrinsic := renvoRuntimeIntrinsicID(p.src, fn.nameStart, fn.nameEnd)
	if intrinsic == 1 {
		e := &ep.exprs[idx]
		if e.argCount != 1 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		return renvoEmitExitStatus(g)
	}
	if intrinsic == 2 {
		if ep.exprs[idx].argCount != 0 {
			return false
		}
		a := &g.asm
		renvoStringHeapOffsets(g)
		readyLabel := renvoAsmNewLabel(a)
		renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
		renvoAsmJnzPrimary(a, readyLabel)
		renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
		renvoAsmStorePrimaryBss(a, g.stringHeapOff)
		renvoAsmMarkLabel(a, readyLabel)
		renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
		return true
	}
	if intrinsic == 3 {
		e := &ep.exprs[idx]
		if e.argCount != 1 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		renvoStringHeapOffsets(g)
		renvoEmitArenaRememberReset(g, false)
		renvoAsmStorePrimaryBss(&g.asm, g.stringHeapOff)
		return true
	}
	if intrinsic == 4 {
		if ep.exprs[idx].argCount != 0 {
			return false
		}
		renvoEmitPersistentArenaReady(g)
		renvoAsmLoadPrimaryBss(&g.asm, g.stringHeapEndOff)
		return true
	}
	if intrinsic == 5 {
		return renvoEmitRuntimeArenaPersistReset(g, ep, idx)
	}
	if intrinsic == 6 {
		return renvoEmitRuntimeArenaPersistString(g, ep, idx)
	}
	if intrinsic == 7 {
		return renvoEmitRuntimeArenaPersistBytes(g, ep, idx)
	}
	if intrinsic == 8 {
		return renvoEmitRuntimeArenaPersistSlice(g, ep, idx)
	}
	if intrinsic == 23 {
		e := &ep.exprs[idx]
		if e.argCount != 1 {
			return false
		}
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		sliceType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, argIndex))
		if sliceType.kind != renvoTypeSlice {
			return false
		}
		return renvoEmitSliceValueRegs(g, ep, argIndex)
	}
	if intrinsic == 12 {
		return renvoEmitRuntimeArenaDiscard(g, ep, idx)
	}
	if intrinsic == 13 {
		return renvoEmitRuntimeArenaDiscardSlice(g, ep, idx)
	}
	if intrinsic == 14 {
		e := &ep.exprs[idx]
		if e.argCount != 1 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		renvoEmitSwapThreadState(g)
		return true
	}
	if intrinsic == 16 {
		if ep.exprs[idx].argCount != 0 {
			return false
		}
		renvoEmitThreadStateRegisterCapability(g)
		return true
	}
	if intrinsic == 17 || intrinsic == 18 {
		return renvoEmitRuntimeStack(g, ep, idx)
	}
	if intrinsic >= 19 && intrinsic <= 22 {
		e := &ep.exprs[idx]
		if e.argCount != 1 {
			return false
		}
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if intrinsic == 19 {
			return renvoEmitScalarExprForKind(g, ep, arg, renvoTypeFloat32)
		}
		if intrinsic == 21 {
			return renvoEmitScalarExprForKind(g, ep, arg, renvoTypeFloat64)
		}
		return renvoEmitIntExpr(g, ep, arg)
	}
	return false
}

func renvoEnsureRuntimeStackHelpers(g *renvoLinearGen, fn int) {
	if g.stackInitLabel > 0 {
		return
	}
	renvoLinearMarkFunc(g, fn)
	a := &g.asm
	init := renvoAsmNewLabel(a)
	switchStack := renvoAsmNewLabel(a)
	g.stackInitLabel = init + 1
	g.stackSwitchLabel = switchStack + 1
	renvoAsmRuntimeStackHelpers(a, init, switchStack, g.funcLabels[fn])
}

func renvoEmitRuntimeStack(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	e := &ep.exprs[idx]
	count := e.argCount - 1
	if count != 2 && count != 5 {
		return false
	}
	return renvoEmitTargetRuntimeStack(g, ep, e, count)
}

// Compiler-private intrinsics live in the reserved renvo_runtime namespace.
// A bounded name hash mixed with an independent byte checksum keeps their
// dispatch table compact while making accidental aliases impractical.
const renvoRuntimeIntrinsicTable = "\x9f\x85\x31\x61\x01\xcb\x5d\x4c\x2e\x02\x03\x1e\x4f\x00\x03\x67\x75\x10\x6e\x04\xaf\xd8\xf6\x20\x05\x1b\xfe\x37\x3f\x06\xe7\x1a\x8d\x21\x07\x15\x6b\xc1\x4f\x08\x07\xf9\x8f\x0d\x08\x8b\x07\x40\x3f\x08\x3b\x59\x62\x4e\x08\x47\x47\xc5\x5f\x0c\x47\x02\x93\x57\x0d\xc5\x07\xc6\x53\x0d\xad\xfc\x67\x17\x0d\x0b\x3b\x57\x66\x0d\x4f\x60\xcb\x57\x0d\x8b\xd1\xdd\x57\x0d\x95\xc5\x1f\x2c\x0e\x71\xbf\x72\x5d\x10\xbb\x84\xa2\x5b\x11\x31\xdd\xa5\x1c\x12\x3d\x21\xa6\x45\x13\x1f\x36\x4c\x7f\x14\xd9\x61\xdf\x55\x15\xe3\xec\x79\x7a\x16\x51\x35\x60\x42\x17"

func renvoRuntimeIntrinsicID(src []byte, start int, end int) int {
	hash1 := 5381
	hash2 := 0
	for i := start; i < end; i++ {
		ch := int(renvo_runtime_UnsafeByteAt(src, i))
		hash1 = (((hash1 << 5) + hash1) ^ ch) & 2147483647
		hash2 += ch
	}
	key := (hash1 ^ hash2*65537) & 2147483647
	for i := 0; i < len(renvoRuntimeIntrinsicTable); i += 5 {
		entryHash := int(renvoRuntimeIntrinsicTable[i]) | int(renvoRuntimeIntrinsicTable[i+1])<<8 | int(renvoRuntimeIntrinsicTable[i+2])<<16 | int(renvoRuntimeIntrinsicTable[i+3])<<24
		if key == entryHash {
			return int(renvoRuntimeIntrinsicTable[i+4])
		}
	}
	return 0
}

func renvoEmitStaticWrite(g *renvoLinearGen, text string, fd int) bool {
	renvoNonNil(g)
	var data []byte
	for i := 0; i < len(text); i++ {
		data = append(data, text[i])
	}
	offset := renvoAddStringData(g, data)
	renvoAsmPrimaryDataAddr(&g.asm, offset)
	renvoAsmSecondaryImm(&g.asm, len(data))
	return renvoEmitWriteValueRegs(g, fd)
}

func renvoEmitBuiltinPanic(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	valueOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitInterfaceAssignToLocal(g, ep, argIndex, valueOffset) {
		return false
	}
	// A nil interface is not a recoverable nil result in the current Go
	// language semantics. Preserve typed nil values, which have a type tag.
	nonNil := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset-renvoBackendValueSlotSize)
	renvoAsmJnzPrimary(&g.asm, nonNil)
	renvoAsmStoreStackImm(&g.asm, valueOffset-renvoBackendValueSlotSize, renvoPanicNilTag)
	renvoAsmMarkLabel(&g.asm, nonNil)
	return renvoEmitPanicState(g, valueOffset)
}

func renvoEmitPanicState(g *renvoLinearGen, valueOffset int) bool {
	renvoNonNil(g)
	// A linked mixed C/Go unit retains the C11 marker so translated C helpers
	// keep their pointer and cleanup semantics.  When that marker suppresses
	// whole-program panic state, ordinary Go type assertions can still reach
	// this helper.  Transfer to the target's uncaught-fault path instead of
	// emitting a jump through the unset per-function return label.
	if !g.meta.panicEnabled || g.deferReturnLabel <= 0 {
		renvoEmitUncaughtFaultTransfer(g, false)
		return true
	}
	noPrevious := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmJzPrimary(&g.asm, noPrevious)
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	nodeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, 4*renvoBackendValueSlotSize)
	oldSuppressPanicCheck := g.suppressPanicCheck
	g.suppressPanicCheck = true
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	g.suppressPanicCheck = oldSuppressPanicCheck
	renvoAsmStorePrimaryStack(&g.asm, nodeOffset)
	renvoEmitStorePanicNodeField(g, nodeOffset, renvoThreadPanicValueOff, 0)
	if g.c.renvoNativeIntSize == 4 {
		renvoEmitStorePanicNodeField(g, nodeOffset, renvoThreadPanicValueOff+4, 4)
	}
	renvoEmitStorePanicNodeField(g, nodeOffset, renvoThreadPanicTypeOff, renvoBackendValueSlotSize)
	renvoEmitStorePanicNodeField(g, nodeOffset, renvoThreadPanicIDOff, 2*renvoBackendValueSlotSize)
	renvoEmitStorePanicNodeField(g, nodeOffset, renvoThreadPanicPrevOff, 3*renvoBackendValueSlotSize)
	renvoAsmLoadPrimaryStack(&g.asm, nodeOffset)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicPrevOff)
	renvoAsmMarkLabel(&g.asm, noPrevious)
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicValueOff)
	if g.c.renvoNativeIntSize == 4 {
		renvoAsmLoadPrimaryStack(&g.asm, valueOffset-4)
		renvoAsmStorePrimaryThreadState(g, renvoThreadPanicValueOff+4)
	}
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset-renvoBackendValueSlotSize)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicTypeOff)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicNextIDOff)
	renvoAsmIncPrimary(&g.asm)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicNextIDOff)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicRecoveredOff)
	renvoAsmJmpLabel(&g.asm, g.deferReturnLabel)
	return true
}

func renvoEmitRuntimeFault(g *renvoLinearGen) {
	renvoEmitRuntimeFaultKind(g, renvoRuntimeTypeTag(g.meta, renvoTypeInt), false)
}

func renvoEmitRuntimeFaultKind(g *renvoLinearGen, panicTag int, outOfMemory bool) {
	renvoNonNil(g)
	a := &g.asm
	if g.meta.panicEnabled && g.deferReturnLabel > 0 {
		valueOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
		renvoAsmStoreStackImm(a, valueOffset, 1)
		renvoAsmStoreStackImm(a, valueOffset-renvoBackendValueSlotSize, panicTag)
		renvoEmitPanicState(g, valueOffset)
	} else {
		renvoEmitUncaughtFaultTransfer(g, outOfMemory)
	}
}

func renvoEnsureUncaughtRuntimeFaultHelper(g *renvoLinearGen) int {
	return renvoEnsureUncaughtFaultHelper(g, false)
}

func renvoEmitUncaughtFaultTransfer(g *renvoLinearGen, outOfMemory bool) {
	label := renvoEnsureUncaughtFaultHelper(g, outOfMemory)
	if renvoRTGStructuredFunctions != 0 {
		renvoAsmCallLabel(&g.asm, label)
		renvoAsmRet(&g.asm)
		return
	}
	renvoAsmJmpLabel(&g.asm, label)
}

func renvoEnsureUncaughtFaultHelper(g *renvoLinearGen, outOfMemory bool) int {
	renvoNonNil(g)
	labelState := g.runtimeFaultLabel
	if outOfMemory {
		labelState = g.arenaFaultLabel
	}
	if labelState > 0 {
		return labelState - 1
	}
	a := &g.asm
	label := renvoAsmNewLabel(a)
	if outOfMemory {
		g.arenaFaultLabel = label + 1
	} else {
		g.runtimeFaultLabel = label + 1
	}
	if renvoRTGStructuredFunctions != 0 {
		argument := 0
		if outOfMemory {
			argument = 1
		}
		renvoQueueStructuredHelper(g, renvoStructuredHelperFault, argument, label)
		return label
	}
	after := renvoAsmNewLabel(a)
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		// A freestanding object has no userspace process or syscall ABI. Model an
		// impossible checked-runtime path as a normal compiler trap that objtool
		// and the kernel linker both understand.
		renvoAsmMarkLabel(a, label)
		renvoAsmEmit16(a, 0x0b0f)
	} else {
		renvoAsmJmpMarkLabel(a, after, label)
		renvoEmitUncaughtFaultHelperBody(g, outOfMemory)
	}
	renvoAsmMarkLabel(a, after)
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		renvoAsmAddLocalObjectFuncSymbolText(a, "__renvo_object_fault", label, after)
	}
	return label
}

func renvoEmitUncaughtFaultHelperBody(g *renvoLinearGen, outOfMemory bool) {
	message := "panic\n"
	if outOfMemory {
		message = "out of memory\n"
	}
	renvoEmitStaticWrite(g, message, 2)
	renvoAsmPrimaryImm(&g.asm, 2)
	renvoEmitExitStatus(g)
}

func renvoEmitRuntimeNonNilPrimary(g *renvoLinearGen) {
	renvoNonNil(g)
	if renvoProgramUsesC11Semantics(g.prog) {
		return
	}
	a := &g.asm
	if !g.meta.panicEnabled {
		renvoEmitUncheckedNonNilPrimary(g)
		return
	}
	ok := renvoAsmNewLabel(a)
	renvoAsmJnzPrimary(a, ok)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(a, ok)
}

func renvoEnsureNonNilCheckHelper(g *renvoLinearGen, secondary bool) int {
	renvoNonNil(g)
	targetHelper := renvoEmitTargetNonNilCheckHelper(g, secondary)
	if targetHelper >= 0 {
		return targetHelper
	}
	labelSlot := &g.runtimeNonNilLabel
	if secondary {
		labelSlot = &g.runtimeSecondaryLabel
	}
	if *labelSlot > 0 {
		return *labelSlot - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	*labelSlot = label + 1
	if renvoRTGStructuredFunctions != 0 {
		argument := 0
		if secondary {
			argument = 1
		}
		renvoQueueStructuredHelper(g, renvoStructuredHelperNonNil, argument, label)
		return label
	}
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoEmitNonNilCheckHelperBody(g, secondary)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitNonNilCheckHelperBody(g *renvoLinearGen, secondary bool) {
	ok := renvoAsmNewLabel(&g.asm)
	if secondary {
		renvoAsmPushSecondary(&g.asm)
		renvoAsmPopPrimary(&g.asm)
	}
	renvoAsmJnzPrimary(&g.asm, ok)
	renvoEmitUncaughtFaultTransfer(g, false)
	renvoAsmMarkLabel(&g.asm, ok)
	renvoAsmRet(&g.asm)
}

func renvoEmitRuntimeNonNilSecondary(g *renvoLinearGen) {
	renvoNonNil(g)
	if renvoProgramUsesC11Semantics(g.prog) {
		return
	}
	a := &g.asm
	if !g.meta.panicEnabled {
		renvoEmitUncheckedNonNilSecondary(g)
		return
	}
	renvoAsmPushSecondary(a)
	renvoAsmPopPrimary(a)
	renvoEmitRuntimeNonNilPrimary(g)
	renvoAsmCopyPrimaryToSecondary(a)
}

func renvoRuntimeNonNilLocalNeeded(g *renvoLinearGen, localIndex int) bool {
	renvoNonNil(g)
	if localIndex < g.c.renvoNativeIntSize*8-1 {
		bit := 1 << localIndex
		if g.checkedPointerLocals&bit != 0 {
			return false
		}
		g.checkedPointerLocals |= bit
	}
	return true
}

func renvoEmitRuntimeUnsafeIndex(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, size int) bool {
	renvoNonNil(g, ep, e)
	if e.argCount == 1 {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, 0, size)
		return true
	}
	if e.argCount != 2 || !renvoEmitSlicePtrLen(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmPopPrimaryToTertiary(&g.asm)
	renvoAsmLoadPrimaryIndexTertiarySize(&g.asm, size)
	if size == 4 {
		renvoAsmNormalizePrimaryForKind(&g.asm, renvoTypeInt32)
	}
	return true
}

func renvoEmitRuntimeTruncateSlice(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, size int) bool {
	renvoNonNil(g)
	renvoNonNil(ep)
	renvoNonNil(e)
	if e.argCount != 2 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmPopSecondary(&g.asm)
	displacement := 0
	if size < 0 {
		displacement = renvoBackendValueSlotSize
		size = g.c.renvoNativeIntSize
	}
	renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, displacement, size)
	return true
}

func renvoEmitRuntimeTrustPointer(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	for i := 0; i < e.argCount; i++ {
		arg := &ep.exprs[renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)]
		if arg.kind == renvoExprIdent {
			local := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
			if local >= 0 && local < g.c.renvoNativeIntSize*8-1 {
				g.checkedPointerLocals |= 1 << local
			}
		}
	}
	return true
}

func renvoEmitBuiltinRecover(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 0 {
		return false
	}
	valueOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitRecoverToLocal(g, valueOffset) {
		return false
	}
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset)
	return true
}

func renvoEmitProgramPanicCheck(g *renvoLinearGen) bool {
	renvoNonNil(g)
	if !g.meta.panicEnabled {
		return true
	}
	renvoEnsurePanicState(g)
	a := &g.asm
	normalLabel := renvoAsmNewLabel(a)
	stringLabel := renvoAsmNewLabel(a)
	assertionLabel := renvoAsmNewLabel(a)
	outOfMemoryLabel := renvoAsmNewLabel(a)
	nilLabel := renvoAsmNewLabel(a)
	exitLabel := renvoAsmNewLabel(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmJzPrimary(a, normalLabel)
	renvoEmitStaticWrite(g, "panic: ", 2)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicTypeOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, renvoPanicOutOfMemoryTag)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, outOfMemoryLabel)
	renvoAsmPrimaryImm(a, renvoPanicNilTag)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, nilLabel)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmPrimaryImm(a, renvoPanicTypeAssertionTag)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, assertionLabel)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmPrimaryImm(a, renvoRuntimeTypeTag(g.meta, renvoTypeString))
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, stringLabel)
	renvoEmitStaticWrite(g, "value", 2)
	renvoAsmJmpMarkLabel(a, exitLabel, stringLabel)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicValueOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, renvoBackendValueSlotSize)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicValueOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmPopSecondary(a)
	renvoEmitWriteValueRegs(g, 2)
	renvoAsmJmpMarkLabel(a, exitLabel, assertionLabel)
	renvoEmitStaticWrite(g, "interface conversion failed", 2)
	renvoAsmJmpMarkLabel(a, exitLabel, outOfMemoryLabel)
	renvoEmitStaticWrite(g, "out of memory", 2)
	renvoAsmJmpMarkLabel(a, exitLabel, nilLabel)
	renvoEmitStaticWrite(g, "panic called with nil argument", 2)
	renvoAsmMarkLabel(a, exitLabel)
	renvoEmitStaticWrite(g, "\n", 2)
	renvoAsmPrimaryImm(a, 2)
	if !renvoEmitExitStatus(g) {
		return false
	}
	renvoAsmMarkLabel(a, normalLabel)
	renvoAsmPopPrimary(a)
	return true
}

func renvoEmitRecoverToLocal(g *renvoLinearGen, offset int) bool {
	renvoNonNil(g)
	a := &g.asm
	noneLabel := renvoAsmNewLabel(a)
	clearLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	previousOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmLoadPrimaryStack(a, g.panicRecoverAllowedOffset)
	renvoAsmJzPrimary(a, noneLabel)
	renvoAsmLoadPrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmJzPrimary(a, noneLabel)
	renvoAsmCopyThreadStateToStack(g, renvoThreadPanicValueOff, offset)
	if g.c.renvoNativeIntSize == 4 {
		renvoAsmCopyThreadStateToStack(g, renvoThreadPanicValueOff+4, offset-4)
	}
	renvoAsmCopyThreadStateToStack(g, renvoThreadPanicTypeOff, offset-renvoBackendValueSlotSize)
	renvoAsmStoreStackImm(a, g.panicRecoverAllowedOffset, 0)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicRecoveredOff)
	renvoAsmCopyThreadStateToStack(g, renvoThreadPanicPrevOff, previousOffset)
	renvoAsmJzPrimary(a, clearLabel)
	renvoEmitLoadPanicNodeField(g, previousOffset, renvoThreadPanicValueOff, 0)
	if g.c.renvoNativeIntSize == 4 {
		renvoEmitLoadPanicNodeField(g, previousOffset, renvoThreadPanicValueOff+4, 4)
	}
	renvoEmitLoadPanicNodeField(g, previousOffset, renvoThreadPanicTypeOff, renvoBackendValueSlotSize)
	renvoEmitLoadPanicNodeField(g, previousOffset, renvoThreadPanicIDOff, 2*renvoBackendValueSlotSize)
	renvoEmitLoadPanicNodeField(g, previousOffset, renvoThreadPanicPrevOff, 3*renvoBackendValueSlotSize)
	renvoAsmJmpMarkLabel(a, doneLabel, clearLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicIDOff)
	renvoAsmStorePrimaryThreadState(g, renvoThreadPanicPrevOff)
	renvoAsmJmpMarkLabel(a, doneLabel, noneLabel)
	renvoAsmStoreStackImm(a, offset, 0)
	renvoAsmStoreStackImm(a, offset-renvoBackendValueSlotSize, 0)
	renvoAsmMarkLabel(a, doneLabel)
	return true
}

func renvoEmitCallParamArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, paramIndex int) int {
	renvoNonNil(g, ep)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	if paramIndex >= 0 && paramIndex < len(meta.params) {
		param := &meta.params[paramIndex]
		if renvoTypeIsSlice(meta, param.typ) {
			e := &ep.exprs[idx]
			if e.kind == renvoExprIdent && renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "nil") {
				if !renvoEmitSliceValueRegs(g, ep, idx) {
					return -1
				}
				renvoAsmPushSliceRegs(&g.asm)
				return 3
			}
		}
		resolved := renvoResolveType(meta, param.typ)
		renvoNonNil(resolved)
		if resolved.kind == renvoTypePointer && renvoProgramUsesC11Semantics(g.prog) {
			e := &ep.exprs[idx]
			if e.kind == renvoExprString {
				if !renvoEmitStringValueRegs(g, ep, idx) {
					return -1
				}
				// C string literals decay to their first byte. The C frontend
				// retains a Go string literal as compact storage, but its length
				// word must not become a second argument in an internal call.
				renvoAsmPushPrimary(&g.asm)
				return 1
			}
		}
		if renvoFixedTarget == 0 {
			if resolved.kind == renvoTypeFunc && g.c.objectFile {
				if !renvoEmitScalarExprForKind(g, ep, idx, renvoTypeFunc) {
					return -1
				}
				renvoAsmPushPrimary(&g.asm)
				return 1
			}
		}
		if resolved.kind == renvoTypeFunc && targetIsKernelModule(g.c) {
			return renvoEmitKernelCallbackArgReverse(g, ep, idx, param.typ)
		}
		if resolved.kind == renvoTypeInterface {
			tempOffset := renvoAddUnnamedLocal(g, param.typ)
			if !renvoEmitInterfaceAssignToLocal(g, ep, idx, tempOffset) {
				return -1
			}
			renvoAsmPushStackWord(&g.asm, tempOffset-renvoBackendValueSlotSize)
			renvoAsmPushStackWord(&g.asm, tempOffset)
			return 2
		}
		if renvoTypeKindIsComplex(resolved.kind) && !(g.c.renvoNativeIntSize == 4 && resolved.kind == renvoTypeComplex) {
			if !renvoEmitComplexValueRegsForKind(g, ep, idx, resolved.kind) {
				return -1
			}
			renvoAsmPushSecondary(&g.asm)
			renvoAsmPushPrimary(&g.asm)
			return 2
		}
		if resolved.kind == renvoTypeStruct || resolved.kind == renvoTypeArray || g.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(resolved.kind) || resolved.kind == renvoTypeComplex) {
			return renvoEmitStructArgReverse(g, ep, idx, param.typ)
		}
		source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(source)
		if renvoTypeKindIsFloat(resolved.kind) || renvoTypeKindIsFloat(source.kind) {
			if !renvoEmitScalarExprForKind(g, ep, idx, resolved.kind) {
				return -1
			}
			renvoAsmPushPrimary(&g.asm)
			return 1
		}
	}
	return renvoEmitCallArgReverse(g, ep, idx)
}

func renvoEmitMethodReceiverArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, receiverType int) int {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	receiver := renvoResolveType(meta, receiverType)
	renvoNonNil(receiver)
	exprType := renvoInferParsedExprType(g, ep, idx)
	actualExprType := exprType
	e := &ep.exprs[idx]
	if e.kind == renvoExprCall {
		fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			actualExprType = meta.funcs[fnIndex].resultType
		}
	} else if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			actualExprType = g.locals[localIndex].typ
		}
	}
	actualExprResolved := renvoResolveType(meta, actualExprType)
	renvoNonNil(actualExprResolved)
	if renvoPreparedBackendActive != 0 && renvoStructArgByReference(g, receiver.kind) {
		if actualExprResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, idx) {
				return -1
			}
		} else if !renvoEmitAddressPrimary(g, ep, idx) {
			offset := renvoAddUnnamedLocal(g, receiverType)
			if !renvoEmitTypedAssign(g, ep, idx, offset) {
				return -1
			}
			renvoAsmAddressPrimaryStack(a, offset)
		}
		renvoAsmPushPrimary(a)
		return 1
	}
	if receiver.kind == renvoTypePointer {
		if actualExprResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, idx) {
				return -1
			}
			// A selector may find a method through one or more pointer layers.
			// Its receiver is the value at the declared pointer depth, not the
			// address of the outer pointer. C aggregate accessors exercise this
			// with chains such as a **T field followed by a method on *T.
			receiverDereferences := renvoPointerDereferenceDistance(meta, actualExprType, receiverType)
			for i := 0; i < receiverDereferences; i++ {
				renvoEmitRuntimeNonNilPrimary(g)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
			}
			renvoAsmPushPrimary(a)
			return 1
		}
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return -1
		}
		renvoAsmPushPrimary(a)
		return 1
	}
	if receiver.kind != renvoTypePointer && actualExprResolved.kind == renvoTypePointer {
		if !renvoEmitIntExpr(g, ep, idx) {
			return -1
		}
		renvoAsmCopyPrimaryToSecondary(a)
		size := renvoTypeSize(meta, receiverType)
		wordSize := renvoCallWordSize(g, receiverType)
		if size <= wordSize {
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
			renvoAsmPushPrimary(a)
			return 1
		}
		renvoEmitPushWords(g, 0, size, wordSize, 0)
		return renvoAlignValue(size, wordSize) / wordSize
	}
	return renvoEmitCallArgReverse(g, ep, idx)
}

func renvoPointerDereferenceDistance(meta *renvoMeta, actual int, declared int) int {
	renvoNonNil(meta)
	for depth := 0; actual > 0 && actual < len(meta.types); depth++ {
		if renvoTypesEquivalent(meta, actual, declared) {
			return depth
		}
		resolved := renvoResolveType(meta, actual)
		renvoNonNil(resolved)
		if resolved.kind != renvoTypePointer {
			break
		}
		actual = resolved.elem
	}
	return -1
}
func renvoEmitMethodReceiverArgTokensReverse(g *renvoLinearGen, dotTok int, receiverType int) int {
	renvoNonNil(g)
	if dotTok <= 0 {
		return -1
	}
	start := dotTok - 1
	if !renvoTokIsKind(g.prog, start, renvoTokIdent) {
		return -1
	}
	receiverEp := renvoNewExprParse()
	renvoNonNil(receiverEp)
	if !renvoParseExpressionOK(receiverEp, g.prog, start, dotTok) {
		return -1
	}
	return renvoEmitMethodReceiverArgReverse(g, receiverEp, len(receiverEp.exprs)-1, receiverType)
}
func renvoEmitCapturedAddress(g *renvoLinearGen, ep *renvoExprParse, idx int) {
	if !g.hasCapturedLocals {
		return
	}
	root := idx
	for ep.exprs[root].kind == renvoExprSelector || ep.exprs[root].kind == renvoExprIndex {
		root = ep.exprs[root].left
	}
	e := &ep.exprs[root]
	if root == idx || e.kind != renvoExprIdent {
		return
	}
	localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
	if localIndex < 0 || g.locals[localIndex].captureOff <= 0 {
		return
	}
	for at := idx; at != root; at = ep.exprs[at].left {
		part := &ep.exprs[at]
		typ := renvoInferParsedExprType(g, ep, part.left)
		kind := renvoResolveType(g.meta, typ).kind
		if part.kind == renvoExprIndex && kind != renvoTypeArray || part.kind == renvoExprSelector && (kind != renvoTypeStruct || renvoStructPromotedPointerField(g, typ, part.nameStart, part.nameEnd) >= 0) {
			return
		}
	}
	// Preserve the computed field/index displacement, but use the cell's
	// lifetime rather than the temporary stack mirror's lifetime.
	a := &g.asm
	renvoAsmPushPrimary(a)
	renvoAsmAddressPrimaryStack(a, g.locals[localIndex].offset)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryStack(a, g.locals[localIndex].captureOff)
	renvoAsmPopTertiary(a)
	renvoAsmAddPrimaryTertiary(a)
}

func renvoRefreshCapturedExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) {
	if !g.hasCapturedLocals {
		return
	}
	for ep.exprs[idx].kind == renvoExprSelector || ep.exprs[idx].kind == renvoExprIndex {
		idx = ep.exprs[idx].left
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return
	}
	renvoMoveCapturedLocal(g, renvoFindLocalIndex(g, e.nameStart, e.nameEnd), false)
}

func renvoEmitAddressPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if g.locals[localIndex].captureOff > 0 {
				renvoAsmLoadPrimaryStack(a, g.locals[localIndex].captureOff)
				return true
			}
			renvoAsmAddressPrimaryStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			renvoAsmPrimaryBssAddr(a, globalOffset)
			return true
		}
	}
	if e.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmCopySecondaryToPrimary(a)
		renvoEmitCapturedAddress(g, ep, idx)
		return true
	}
	if e.kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		renvoEmitCapturedAddress(g, ep, idx)
		return true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		return true
	}
	if e.kind == renvoExprCall || e.kind == renvoExprComposite {
		typ := renvoInferParsedExprType(g, ep, idx)
		resolved := renvoResolveType(g.meta, typ)
		renvoNonNil(resolved)
		if resolved.kind != renvoTypeStruct && resolved.kind != renvoTypeArray {
			return false
		}
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitTypedAssign(g, ep, idx, offset) {
			return false
		}
		renvoAsmAddressPrimaryStack(a, offset)
		return true
	}
	return false
}
func renvoEmitCallArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	renvoNonNil(p)
	renvoNonNil(meta)
	a := &g.asm
	typ := renvoInferParsedExprType(g, ep, idx)
	if renvoResolveType(meta, typ).kind == renvoTypeInterface {
		e := &ep.exprs[idx]
		if e.kind != renvoExprIdent {
			return -1
		}
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			return -1
		}
		renvoAsmPushStackWord(a, g.locals[localIndex].offset-renvoBackendValueSlotSize)
		renvoAsmPushStackWord(a, g.locals[localIndex].offset)
		return 2
	}
	if renvoTypeIsSlice(meta, typ) {
		e := &ep.exprs[idx]
		if e.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex >= 0 {
				offset := g.locals[localIndex].offset
				renvoAsmPushStackWord(a, offset-16)
				renvoAsmPushStackWord(a, offset-8)
				renvoAsmPushStackWord(a, offset)
				return 3
			}
		}
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return -1
		}
		renvoAsmPushSliceRegs(&g.asm)
		return 3
	}
	if renvoTypeIsString(g.meta, typ) {
		e := &ep.exprs[idx]
		if e.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex >= 0 {
				offset := g.locals[localIndex].offset
				renvoAsmPushStackWord(a, offset-8)
				renvoAsmPushStackWord(a, offset)
				return 2
			}
		}
		if !renvoEmitStringValueRegs(g, ep, idx) {
			return -1
		}
		renvoAsmPushStringRegs(&g.asm)
		return 2
	}
	if renvoTypeIsTuple(g.meta, typ) {
		return renvoEmitTupleArgReverse(g, ep, idx, typ)
	}
	resolved := renvoResolveType(g.meta, typ)
	renvoNonNil(resolved)
	if renvoTypeKindIsComplex(resolved.kind) && !(g.c.renvoNativeIntSize == 4 && resolved.kind == renvoTypeComplex) {
		if !renvoEmitComplexValueRegs(g, ep, idx) {
			return -1
		}
		renvoAsmPushSecondary(a)
		renvoAsmPushPrimary(a)
		return 2
	}
	if resolved.kind == renvoTypeStruct || resolved.kind == renvoTypeArray || g.c.renvoNativeIntSize == 4 && (renvoTypeKindIsWideValue(resolved.kind) || resolved.kind == renvoTypeComplex) {
		return renvoEmitStructArgReverse(g, ep, idx, typ)
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		value := renvoParseIntToken(p, e.tok)
		if renvoPreparedBackendActive != 0 && g.c.renvoNativeIntSize == 8 && p.compilerInt32 && p.parsedIntHigh != value>>31 {
			renvoAsmLoadPrimaryIntToken(a, p, e.tok)
			renvoAsmPushPrimary(a)
			return 1
		}
		renvoAsmPushImm(a, value)
		return 1
	}
	if e.kind == renvoExprChar {
		value := renvoParseCharToken(p, e.tok)
		renvoAsmPushImm(a, value)
		return 1
	}
	if e.kind == renvoExprBool {
		value := renvoBoolTokenValue(p, e.tok)
		renvoAsmPushImm(a, value)
		return 1
	}
	if e.kind == renvoExprIdent {
		constResult := renvoEvalConstExpr(g, ep, idx)
		if constResult.ok {
			renvoAsmPushImm(a, constResult.value)
			return 1
		}
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			renvoAsmPushStackWord(a, g.locals[localIndex].offset)
			return 1
		}
	}
	if e.kind == renvoExprSelector {
		if offset, ok := renvoLocalStructSelectorOffset(g, ep, idx); ok {
			renvoAsmPushStackWord(a, offset)
			return 1
		}
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return -1
	}
	renvoAsmPushPrimary(a)
	return 1
}

func renvoLocalStructSelectorOffset(g *renvoLinearGen, ep *renvoExprParse, idx int) (int, bool) {
	renvoNonNil(g, ep)
	if !renvoAsmFoldedFieldAddressing(&g.asm) {
		return 0, false
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprSelector {
		return 0, false
	}
	base := &ep.exprs[e.left]
	if base.kind != renvoExprIdent {
		return 0, false
	}
	localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
	if localIndex < 0 {
		return 0, false
	}
	baseType := renvoResolveType(g.meta, g.locals[localIndex].typ)
	renvoNonNil(baseType)
	if baseType.kind != renvoTypeStruct {
		return 0, false
	}
	fieldOffset := renvoStructFieldOffset(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return 0, false
	}
	return g.locals[localIndex].offset - fieldOffset, true
}

func renvoEmitTupleArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, typ int) int {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprCall {
		return -1
	}
	offset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := renvoResolveType(g.meta, typ)
	renvoNonNil(tuple)
	wordCount := 0
	for i := 0; i < tuple.count; i++ {
		field := g.meta.fields[tuple.first+i]
		size := renvoTypeCopySize(g.meta, field.typ)
		wordSize := renvoCallWordSize(g, field.typ)
		renvoEmitPushWords(g, offset-field.offset, size, wordSize, renvoPushStack)
		wordCount += renvoAlignValue(size, wordSize) / wordSize
	}
	return wordCount
}

func renvoEmitStructArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, typ int) int {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	size := renvoTypeSize(meta, typ)
	if size <= 0 {
		return -1
	}
	e := &ep.exprs[idx]
	if renvoPreparedBackendActive != 0 && renvoStructArgByReference(g, renvoResolveType(meta, typ).kind) {
		if !renvoEmitAddressPrimary(g, ep, idx) {
			offset := renvoAddUnnamedLocal(g, typ)
			if !renvoEmitTypedAssign(g, ep, idx, offset) {
				return -1
			}
			renvoAsmAddressPrimaryStack(a, offset)
		}
		renvoAsmPushPrimary(a)
		return 1
	}
	wordSize := renvoCallWordSize(g, typ)
	wordCount := renvoAlignValue(size, wordSize) / wordSize
	if e.kind == renvoExprIdent {
		resolved := renvoResolveType(meta, typ)
		renvoNonNil(resolved)
		if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideValue(resolved.kind) {
			constant := renvoEvalConstExpr(g, ep, idx)
			if constant.ok {
				offset := renvoAddUnnamedLocal(g, typ)
				if !renvoEmitWideExprToLocal(g, ep, idx, offset, resolved.kind) {
					return -1
				}
				renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
				return wordCount
			}
		}
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if renvoTypeSize(meta, g.locals[localIndex].typ) != size {
				return -1
			}
			renvoEmitPushWords(g, g.locals[localIndex].offset, size, wordSize, renvoPushStack)
			return wordCount
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset < 0 || renvoTypeSize(meta, globalType) != size {
			return -1
		}
		renvoEmitPushWords(g, globalOffset, size, wordSize, renvoPushBss)
		return wordCount
	}
	if e.kind == renvoExprIndex {
		leftType := renvoInferParsedExprType(g, ep, e.left)
		sliceType := renvoResolveType(meta, leftType)
		renvoNonNil(sliceType)
		elemType := renvoResolveType(meta, sliceType.elem)
		renvoNonNil(elemType)
		if (sliceType.kind != renvoTypeSlice && sliceType.kind != renvoTypeArray) ||
			!renvoTypeUsesHiddenResult(meta, sliceType.elem) || renvoTypeSize(meta, sliceType.elem) != size {
			return -1
		}
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return -1
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoEmitPushWords(g, 0, size, wordSize, 0)
		return wordCount
	}
	if e.kind == renvoExprSelector {
		fieldType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeUsesHiddenResult(meta, fieldType) || renvoTypeSize(meta, fieldType) != size {
			return -1
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return -1
		}
		renvoEmitPushWords(g, 0, size, wordSize, 0)
		return wordCount
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeUsesHiddenResult(meta, valueType) || renvoTypeSize(meta, valueType) != size {
			return -1
		}
		if !renvoEmitIntExpr(g, ep, e.left) {
			return -1
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoEmitPushWords(g, 0, size, wordSize, 0)
		return wordCount
	}
	offset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitTypedAssign(g, ep, idx, offset) {
		return -1
	}
	renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
	return wordCount
}
func renvoEmitAppendAssignGeneral(g *renvoLinearGen, stmt *renvoStmt, ep *renvoExprParse, assignTok int) bool {
	renvoNonNil(g, stmt, ep)
	p := g.prog
	if len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != renvoExprCall || root.argCount < 2 || renvoExprIdentCode(p, ep, root.left) != renvoIdentAppend {
		return false
	}
	if assignTok > stmt.startTok {
		matches := renvoAppendAssignLhsMatchesSource(p, stmt, ep, root, assignTok)
		if !matches {
			return renvoEmitAppendAssignDifferentSource(g, stmt, ep, root, assignTok)
		}
	}
	var loc renvoSliceLocation
	locEp := ep
	if assignTok > stmt.startTok {
		lhs := renvoNewExprParse()
		renvoNonNil(lhs)
		if renvoParseExpressionOK(lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			renvoSetSliceLocationFromExpr(g, lhs, lhsIndex, &loc)
			locEp = lhs
		}
	}
	if !loc.ok {
		renvoSetSliceLocationFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, root.firstArg), &loc)
		locEp = ep
	}
	if !loc.ok {
		return false
	}
	return renvoEmitAppendToLocation(g, stmt, ep, locEp, &loc, root)
}

func renvoEmitAppendAssignDifferentSource(g *renvoLinearGen, stmt *renvoStmt, ep *renvoExprParse, root *renvoExpr, assignTok int) bool {
	renvoNonNil(g, stmt, ep, root)
	p := g.prog
	lhs := renvoNewExprParse()
	renvoNonNil(lhs)
	lhsIndex := renvoParseExpressionRoot(lhs, p, stmt.startTok, assignTok)
	if lhsIndex < 0 {
		return false
	}
	var lhsLoc renvoSliceLocation
	renvoSetSliceLocationFromExpr(g, lhs, lhsIndex, &lhsLoc)
	if !lhsLoc.ok {
		return false
	}
	if lhsLoc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, lhs, &lhsLoc) {
			return false
		}
		headerOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreSecondaryStack(&g.asm, headerOffset)
		lhsLoc = renvoSliceLocation{offset: headerOffset, typ: lhsLoc.typ, mem: true, indirect: true, ok: true}
	}
	sourceType := renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, root.firstArg))
	if !renvoTypeIsSlice(g.meta, sourceType) {
		return false
	}
	tempOffset := renvoAddUnnamedLocal(g, sourceType)
	if !renvoEmitSliceValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, root.firstArg)) {
		return false
	}
	renvoAsmStoreSliceStack(&g.asm, tempOffset)
	tempLoc := renvoSliceLocation{offset: tempOffset, typ: sourceType, ok: true}
	if !renvoEmitAppendToLocation(g, stmt, ep, ep, &tempLoc, root) {
		return false
	}
	renvoAsmLoadPrimarySecondaryStack(&g.asm, tempOffset, tempOffset-8)
	renvoAsmLoadTertiaryStack(&g.asm, tempOffset-16)
	return renvoStoreSliceRegsToLocation(g, lhs, &lhsLoc)
}

func renvoStoreSliceRegsToLocation(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation) bool {
	renvoNonNil(g, locEp, loc)
	if loc.global {
		renvoAsmStoreSliceBss(&g.asm, loc.offset)
		return true
	}
	if loc.mem {
		renvoAsmPushSliceRegs(&g.asm)
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoAsmPopStoreSliceMemSecondary(&g.asm, 0)
		return true
	}
	renvoAsmStoreSliceStack(&g.asm, loc.offset)
	return true
}

func renvoEmitSliceLocationHeaderAddressSecondary(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation) bool {
	renvoNonNil(g, locEp, loc)
	if loc.indirect {
		renvoAsmLoadSecondaryStack(&g.asm, loc.offset)
		return true
	}
	if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
		return false
	}
	if loc.deref {
		if !renvoEmitIntExpr(g, locEp, loc.expr) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		return true
	}
	if locEp.exprs[loc.expr].kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, locEp, loc.expr) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		return true
	}
	return renvoEmitSelectorAddressSecondary(g, locEp, loc.expr)
}

func renvoAppendAssignLhsMatchesSource(p *renvoProgram, stmt *renvoStmt, ep *renvoExprParse, root *renvoExpr, assignTok int) bool {
	renvoNonNil(p, stmt, ep, root)
	for i := stmt.startTok; i < assignTok; i++ {
		if renvoTokCharIs(p, i, '(') {
			return false
		}
	}
	firstStart := root.tok + 1
	closeTok := renvoFindMatchingExprClose(p, root.tok+1, ep.end, '(', ')')
	if closeTok <= firstStart {
		return false
	}
	firstEnd := renvoFindExprBoundary(p, firstStart, closeTok)
	if firstEnd <= firstStart {
		return false
	}
	aStartTok := stmt.startTok
	aEndTok := assignTok
	bStartTok := firstStart
	bEndTok := firstEnd
	for aStartTok < aEndTok && renvoTokIsSpaceLike(p, aStartTok) {
		aStartTok++
	}
	for bStartTok < bEndTok && renvoTokIsSpaceLike(p, bStartTok) {
		bStartTok++
	}
	for aEndTok > aStartTok && renvoTokIsSpaceLike(p, aEndTok-1) {
		aEndTok--
	}
	for bEndTok > bStartTok && renvoTokIsSpaceLike(p, bEndTok-1) {
		bEndTok--
	}
	if aStartTok >= aEndTok || bStartTok >= bEndTok {
		return false
	}
	aStart := int(renvoTokStart(p, aStartTok))
	aEnd := int(renvoTokEnd(p, aEndTok-1))
	bStart := int(renvoTokStart(p, bStartTok))
	bEnd := int(renvoTokEnd(p, bEndTok-1))
	return renvoBytesEqualRange(p.src, aStart, aEnd, bStart, bEnd)
}

func renvoTokIsSpaceLike(p *renvoProgram, tok int) bool {
	renvoNonNil(p)
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	return renvoTokCharIs(p, tok, ';')
}

func renvoEmitAppendToLocation(g *renvoLinearGen, stmt *renvoStmt, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, root *renvoExpr) bool {
	renvoNonNil(g, stmt, ep, locEp, loc, root)
	t := renvoResolveType(g.meta, loc.typ)
	renvoNonNil(t)
	if t.kind != renvoTypeSlice {
		return false
	}
	elem := renvoResolveType(g.meta, t.elem)
	renvoNonNil(elem)
	if root.nameStart == 1 {
		if root.argCount != 2 {
			return false
		}
		valueIndex := renvo_runtime_UnsafeIntAt(ep.args, root.firstArg+1)
		value := &ep.exprs[valueIndex]
		if renvoExprIsNil(g.prog, value) && renvoFindLocalIndex(g, value.nameStart, value.nameEnd) < 0 && renvoFindGlobalType(g, value.nameStart, value.nameEnd) == 0 {
			// Expanding untyped nil appends no elements. Still evaluate a
			// memory-backed destination, including its bounds/dereference checks.
			if loc.mem {
				if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
					return false
				}
				renvoAsmLoadSliceMemSecondary(&g.asm)
			}
			return true
		}
		if elem.kind == renvoTypeByte && renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, valueIndex)) {
			return renvoEmitAppendStringBytesToLocation(g, ep, valueIndex, locEp, loc)
		}
		return renvoEmitAppendExpansionToLocation(g, ep, locEp, loc, t.elem, valueIndex)
	}
	if root.argCount > 2 {
		temps := renvoFixedIntScratch(root.argCount - 1)
		for arg := 1; arg < root.argCount; arg++ {
			valueIndex := renvo_runtime_UnsafeIntAt(ep.args, root.firstArg+arg)
			temp := renvoAddUnnamedLocal(g, t.elem)
			if !renvoEmitExprToLocal(g, ep, valueIndex, temp) {
				return false
			}
			temps = append(temps, temp)
		}
		for i := 0; i < len(temps); i++ {
			if !renvoEmitAppendLocalToLocation(g, locEp, loc, elem, t.elem, temps[i]) {
				return false
			}
		}
		return true
	}
	for arg := 1; arg < root.argCount; arg++ {
		valueIndex := renvo_runtime_UnsafeIntAt(ep.args, root.firstArg+arg)
		if !renvoEmitAppendOneToLocation(g, stmt, ep, locEp, loc, root, elem, t.elem, valueIndex) {
			return false
		}
	}
	return true
}

func renvoEmitAppendOneToLocation(g *renvoLinearGen, stmt *renvoStmt, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, root *renvoExpr, elem *renvoTypeInfo, elemType int, valueIndex int) bool {
	renvoNonNil(g, stmt, ep, locEp, loc, root, elem)
	p := g.prog
	if elem.kind == renvoTypeStruct {
		value := &ep.exprs[valueIndex]
		if value.kind != renvoExprComposite {
			if value.kind == renvoExprUnary && renvoTokCharIs(p, value.tok, '*') {
				return renvoEmitAppendStructDeref(g, ep, locEp, loc, elemType, valueIndex)
			}
			if value.kind == renvoExprIdent {
				typeTok := value.tok
				if !renvoTokCharIs(p, typeTok+1, '{') {
					typeTok = 0
					for i := root.tok; i < stmt.endTok; i++ {
						if int(renvoTokStart(p, i)) == value.nameStart {
							typeTok = i
							break
						}
					}
				}
				if renvoTokCharIs(p, typeTok+1, '{') {
					return renvoEmitAppendStructCompositeTokens(g, locEp, loc, elemType, typeTok)
				}
				return renvoEmitAppendStructLocal(g, ep, locEp, loc, elemType, valueIndex)
			}
			if value.kind == renvoExprCall {
				return renvoEmitAppendStructComposite(g, ep, locEp, loc, elemType, valueIndex)
			}
			if value.kind == renvoExprIndex || value.kind == renvoExprSelector {
				valueType := renvoInferParsedExprType(g, ep, valueIndex)
				if renvoTypeIsStruct(g.meta, valueType) && renvoTypeSize(g.meta, valueType) == renvoTypeSize(g.meta, elemType) {
					return renvoEmitAppendStructComposite(g, ep, locEp, loc, elemType, valueIndex)
				}
			}
			typeTok := -1
			openTok := root.tok
			end := stmt.endTok
			if openTok >= 0 && openTok < end && renvoTokCharIs(p, openTok, '(') {
				i := openTok + 1
				paren := 0
				brack := 0
				brace := 0
				for i < end {
					if paren == 0 && brack == 0 && brace == 0 && renvoTokCharIs(p, i, ',') {
						candidate := i + 1
						if renvoTokIsKind(p, candidate, renvoTokIdent) && renvoTokCharIs(p, candidate+1, '{') {
							typeTok = candidate
						}
						break
					}
					if renvoTokCharIs(p, i, '(') {
						paren++
					} else if renvoTokCharIs(p, i, ')') {
						if paren == 0 {
							break
						}
						paren--
					} else if renvoTokCharIs(p, i, '[') {
						brack++
					} else if renvoTokCharIs(p, i, ']') {
						brack--
					} else if renvoTokCharIs(p, i, '{') {
						brace++
					} else if renvoTokCharIs(p, i, '}') {
						brace--
					}
					i++
				}
			}
			if typeTok >= 0 {
				return renvoEmitAppendStructCompositeTokens(g, locEp, loc, elemType, typeTok)
			}
			return false
		}
		if !renvoEmitAppendStructComposite(g, ep, locEp, loc, elemType, valueIndex) {
			return false
		}
		return true
	}
	if renvoTypeKindIsScalarValue(elem.kind) || elem.kind == renvoTypePointer || elem.kind == renvoTypeFunc {
		elemSize := renvoScalarKindSize(g.c.renvoNativeIntSize, elem.kind)
		if elem.kind != renvoTypePointer && elemSize > g.c.renvoNativeIntSize {
			temp := renvoAddUnnamedLocal(g, elemType)
			if !renvoEmitExprToLocal(g, ep, valueIndex, temp) {
				return false
			}
			return renvoEmitAppendLocalToLocation(g, locEp, loc, elem, elemType, temp)
		}
		if !renvoEmitAppendScalarToLocation(g, ep, locEp, loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == renvoTypeString {
		if !renvoEmitAppendStringToLocation(g, ep, locEp, loc, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == renvoTypeSlice {
		if !renvoEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		return renvoEmitAppendSliceRegs(g, locEp, loc)
	}
	if elem.kind == renvoTypeInterface {
		temp := renvoAddUnnamedLocal(g, elemType)
		if !renvoEmitInterfaceAssignToLocal(g, ep, valueIndex, temp) {
			return false
		}
		return renvoEmitAppendLocalToLocation(g, locEp, loc, elem, elemType, temp)
	}
	return false
}

func renvoEmitAppendLocalToLocation(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elem *renvoTypeInfo, elemType int, offset int) bool {
	renvoNonNil(g, locEp, loc, elem)
	a := &g.asm
	if renvoTypeKindIsScalarValue(elem.kind) || elem.kind == renvoTypePointer || elem.kind == renvoTypeFunc {
		elemSize := renvoScalarKindSize(g.c.renvoNativeIntSize, elem.kind)
		if elem.kind != renvoTypePointer && elemSize > g.c.renvoNativeIntSize {
			if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(a)
			renvoEmitCopyStackToMemSecondary(g, offset, 0, elemSize)
			return true
		}
		renvoAsmPushStack(a, offset)
		if elem.kind == renvoTypePointer || renvoAsmCanStoreScalar(a, elemSize) {
			if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopPrimary(a)
			renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
			return true
		}
		label := renvoEnsureAppendScalarHelper(g, elem.kind)
		if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
			return false
		}
		renvoAsmPopSecondary(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	if elem.kind == renvoTypeString {
		renvoAsmLoadPrimarySecondaryStack(a, offset, offset-8)
		renvoAsmPushStringRegs(a)
		if !renvoEmitAppendDestPrimary(g, locEp, loc, 16) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopStoreStringMemSecondary(a, 0)
		return true
	}
	if elem.kind == renvoTypeSlice {
		renvoAsmLoadPrimarySecondaryStack(a, offset, offset-8)
		renvoAsmLoadTertiaryStack(a, offset-16)
		return renvoEmitAppendSliceRegs(g, locEp, loc)
	}
	if elem.kind == renvoTypeStruct || elem.kind == renvoTypeInterface {
		elemSize := renvoTypeSize(g.meta, elemType)
		if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoEmitCopyStackToMemSecondary(g, offset, 0, elemSize)
		return true
	}
	return false
}

func renvoEmitAppendSliceRegs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation) bool {
	a := &g.asm
	renvoAsmPushSliceRegs(a)
	if !renvoEmitAppendDestPrimary(g, locEp, loc, renvoBackendSliceValueSize) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopStoreSliceMemSecondary(a, 0)
	return true
}

func renvoBinaryUsesFloat(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	p := g.prog
	if renvoTok2Is(p, e.tok, '&', '&') {
		return false
	}
	if renvoTok2Is(p, e.tok, '|', '|') {
		return false
	}
	left := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
	renvoNonNil(left)
	if renvoTypeKindIsFloat(left.kind) {
		return true
	}
	right := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right))
	renvoNonNil(right)
	if renvoTypeKindIsFloat(right.kind) {
		return true
	}
	if !ep.hasFloat {
		return false
	}
	if renvoExprValueIsFloat(g, ep, e.left) {
		return true
	}
	return renvoExprValueIsFloat(g, ep, e.right)
}
func renvoExprValueIsFloat(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == renvoExprFloat {
		return true
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(p, e.tok, '+') || renvoTokCharIs(p, e.tok, '-') {
			return renvoExprValueIsFloat(g, ep, e.left)
		}
		typ := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(typ)
		return renvoTypeKindIsFloat(typ.kind)
	}
	if e.kind == renvoExprBinary {
		if renvoTok2Is(p, e.tok, '=', '=') || renvoTok2Is(p, e.tok, '!', '=') || renvoTokCharIs(p, e.tok, '<') || renvoTokCharIs(p, e.tok, '>') || renvoTok2Is(p, e.tok, '&', '&') || renvoTok2Is(p, e.tok, '|', '|') {
			return false
		}
		if renvoExprValueIsFloat(g, ep, e.left) {
			return true
		}
		return renvoExprValueIsFloat(g, ep, e.right)
	}
	if e.kind == renvoExprIdent || e.kind == renvoExprCall || e.kind == renvoExprIndex || e.kind == renvoExprSelector {
		typ := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(typ)
		return renvoTypeKindIsFloat(typ.kind)
	}
	return false
}
func renvoEmitAppendExpansionToLocation(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemType int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	elemSize := renvoTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	sourceType := renvoInferParsedExprType(g, ep, valueIndex)
	source := renvoResolveType(g.meta, sourceType)
	renvoNonNil(source)
	if source.kind != renvoTypeSlice {
		return false
	}
	if renvoTypeSize(g.meta, source.elem) != elemSize {
		return false
	}
	// Byte expansions dominate source linking and are commonly only a handful
	// of bytes long. Keep their overlap-safe growth and copy in one shared
	// helper instead of repeating the full reservation sequence at every call.
	if elemSize == 1 && renvoHasAppendBytesHelper(g) {
		srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
		srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
		done := renvoAsmNewLabel(a)
		if !renvoEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
		renvoAsmLoadPrimaryStack(a, srcLen)
		renvoAsmJzPrimary(a, done)
		label := renvoEnsureAppendBytesHelper(g)
		if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
			return false
		}
		renvoAsmLoadSecondaryStack(a, srcPtr)
		renvoAsmLoadTertiaryStack(a, srcLen)
		renvoAsmCallLabel(a, label)
		renvoEmitArenaAllocationCheck(g)
		renvoAsmMarkLabel(a, done)
		return true
	}
	srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	destPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	destLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	destCap := renvoAddUnnamedLocal(g, renvoTypeInt)
	finalLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	reserveIndex := renvoAddUnnamedLocal(g, renvoTypeInt)
	headerOffset := 0
	appendLoc := loc
	done := renvoAsmNewLabel(a)
	if !renvoEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
	renvoAsmLoadPrimaryStack(a, srcLen)
	renvoAsmJzPrimary(a, done)
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		headerOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreSecondaryStack(a, headerOffset)
		appendLoc = &renvoSliceLocation{offset: headerOffset, typ: loc.typ, mem: true, indirect: true, ok: true}
		renvoEmitEnsureMemSlice(g, elemSize)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmStorePrimaryStack(a, destPtr)
		renvoAsmLoadPrimaryStackMemory(a, headerOffset, 8)
		renvoAsmStorePrimaryStack(a, destLen)
		renvoAsmLoadPrimaryStackMemory(a, headerOffset, 16)
		renvoAsmStorePrimaryStack(a, destCap)
	} else if loc.global {
		renvoAsmCopyBssToStackSlot(a, loc.offset, destPtr)
		renvoAsmCopyBssToStackSlot(a, loc.offset+8, destLen)
		renvoAsmCopyBssToStackSlot(a, loc.offset+16, destCap)
	} else {
		renvoAsmCopyStackSlot(a, loc.offset, destPtr)
		renvoAsmCopyStackSlot(a, loc.offset-8, destLen)
		renvoAsmCopyStackSlot(a, loc.offset-16, destCap)
	}
	renvoAsmLoadPrimaryTertiaryStack(a, destLen, srcLen)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, finalLen)
	// Reserve destination elements before copying when the current capacity is
	// insufficient. The old expansion path
	// only increased the length after raw stores, so append(dst, src...) wrote
	// beyond dst's capacity when the expansion required growth. Besides leaving
	// len greater than cap, an adjacent source backing could be overwritten while
	// it was still being copied.
	renvoAsmStoreStackImm(a, reserveIndex, 0)
	reserveLoop := renvoAsmNewLabel(a)
	reserveDone := renvoAsmNewLabel(a)
	renvoAsmJgeStackStack(a, destCap, finalLen, reserveDone)
	renvoAsmMarkLabel(a, reserveLoop)
	renvoAsmJgeStackStack(a, reserveIndex, srcLen, reserveDone)
	if !renvoEmitAppendDestPrimary(g, locEp, appendLoc, elemSize) {
		return false
	}
	renvoAsmIncStack(a, reserveIndex)
	renvoAsmJmpMarkLabel(a, reserveLoop, reserveDone)
	// Growth can replace the destination backing. Keep the original length but
	// reload the pointer from the now-authoritative slice header.
	if loc.mem {
		renvoAsmLoadSecondaryStack(a, headerOffset)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmStorePrimaryStack(a, destPtr)
	} else if loc.global {
		renvoAsmCopyBssToStackSlot(a, loc.offset, destPtr)
	} else {
		renvoAsmCopyStackSlot(a, loc.offset, destPtr)
	}
	// Expanded append has copy-like overlap semantics. Convert the destination
	// to the first appended byte and share the byte-accurate memmove emitter
	// used by copy; this also handles packed element sizes without over-copying.
	renvoAsmLoadPrimaryTertiaryStack(a, destPtr, destLen)
	renvoAsmAddScaledTertiary(a, elemSize)
	renvoAsmStorePrimaryStack(a, destPtr)
	renvoAsmLoadTertiaryStack(a, srcLen)
	renvoAsmMulTertiaryImm(a, elemSize)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, reserveIndex)
	renvoEmitCopyBytes(g, srcPtr, destPtr, reserveIndex)
	renvoAsmLoadPrimaryStack(a, finalLen)
	if loc.mem {
		renvoAsmLoadSecondaryStack(a, headerOffset)
		renvoAsmStorePrimaryMemSecondaryDisp(a, 8)
	} else if loc.global {
		renvoAsmStorePrimaryBss(a, loc.offset+8)
	} else {
		renvoAsmStorePrimaryStack(a, loc.offset-8)
	}
	renvoAsmMarkLabel(a, done)
	return true
}

func renvoEmitCopyByteAt(g *renvoLinearGen, srcPtr int, destPtr int, index int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryTertiaryStack(a, srcPtr, index)
	renvoAsmLoadBytePrimaryIndexTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadSecondaryTertiaryStack(a, destPtr, index)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
}
func renvoEmitAppendScalarToLocation(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemKind int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	elemSize := renvoScalarKindSize(g.c.renvoNativeIntSize, elemKind)
	if !renvoEmitScalarExprForKind(g, ep, valueIndex, elemKind) {
		return false
	}
	renvoAsmPushPrimary(a)
	if elemKind == renvoTypePointer || renvoAsmCanStoreScalar(a, elemSize) {
		if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
		return true
	}
	label := renvoEnsureAppendScalarHelper(g, elemKind)
	if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmPopSecondary(a)
	renvoAsmCallLabel(a, label)
	return true
}
func renvoEmitAppendDestPrimary(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	renvoNonNil(g, locEp, loc)
	if renvoPreparedBackendActive != 0 {
		return renvoEmitRTGAppendDestPrimary(g, locEp, loc, elemSize)
	}
	label := renvoEnsureAppendAddrHelper(g)
	if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmSecondaryImm(&g.asm, elemSize)
	renvoAsmCallLabel(&g.asm, label)
	renvoEmitArenaAllocationCheck(g)
	return true
}

func renvoEmitRTGAppendDestPrimary(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	renvoNonNil(g, locEp, loc)
	if elemSize < 1 {
		return false
	}
	a := &g.asm
	dataSlot := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenSlot := renvoAddUnnamedLocal(g, renvoTypeInt)
	capSlot := renvoAddUnnamedLocal(g, renvoTypeInt)
	data := renvoAddUnnamedLocal(g, renvoTypeInt)
	length := renvoAddUnnamedLocal(g, renvoTypeInt)
	capacity := renvoAddUnnamedLocal(g, renvoTypeInt)
	result := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord0)
	renvoAsmStorePrimaryStack(a, dataSlot)
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord1)
	renvoAsmStorePrimaryStack(a, lenSlot)
	renvoRTGDirectMove(a, renvoRTGPrimary, renvoRTGCallWord5)
	renvoAsmStorePrimaryStack(a, capSlot)
	renvoAsmLoadPrimaryStackMemory(a, dataSlot, 0)
	renvoAsmStorePrimaryStack(a, data)
	renvoAsmLoadPrimaryStackMemory(a, lenSlot, 0)
	renvoAsmStorePrimaryStack(a, length)
	renvoAsmLoadPrimaryStackMemory(a, capSlot, 0)
	renvoAsmStorePrimaryStack(a, capacity)

	grow := renvoAsmNewLabel(a)
	finish := renvoAsmNewLabel(a)
	renvoAsmJgeStackStack(a, length, capacity, grow)
	renvoAsmLoadPrimaryTertiaryStack(a, data, length)
	renvoAsmAddScaledTertiary(a, elemSize)
	renvoAsmStorePrimaryStack(a, result)
	renvoAsmJmpLabel(a, finish)

	renvoAsmMarkLabel(a, grow)
	newCapacity := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmLoadPrimaryStack(a, capacity)
	haveCapacity := renvoAsmNewLabel(a)
	renvoAsmJnzPrimary(a, haveCapacity)
	renvoAsmPrimaryImm(a, 8)
	renvoAsmMarkLabel(a, haveCapacity)
	renvoAsmShlPrimaryImm(a, 1)
	renvoAsmStorePrimaryStack(a, newCapacity)
	byteCount := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, elemSize)
	renvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGTertiary)
	renvoAsmStorePrimaryStack(a, byteCount)
	renvoEmitArenaAllocStackPrimary(g, byteCount)
	newData := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(a, newData)
	renvoAsmLoadTertiaryStack(a, length)
	renvoAsmPrimaryImm(a, elemSize)
	renvoRTGDirectMultiply(a, renvoRTGPrimary, renvoRTGTertiary)
	renvoAsmStorePrimaryStack(a, byteCount)
	renvoEmitCopyBytes(g, data, newData, byteCount)
	renvoAsmLoadSecondaryStack(a, dataSlot)
	renvoAsmLoadPrimaryStack(a, newData)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadSecondaryStack(a, capSlot)
	renvoAsmLoadPrimaryStack(a, newCapacity)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryTertiaryStack(a, newData, length)
	renvoAsmAddScaledTertiary(a, elemSize)
	renvoAsmStorePrimaryStack(a, result)

	renvoAsmMarkLabel(a, finish)
	renvoAsmLoadPrimaryStack(a, length)
	renvoAsmIncPrimary(a)
	renvoAsmLoadSecondaryStack(a, lenSlot)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryStack(a, result)
	return true
}

func renvoEmitAppendStringToLocation(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	if renvoPreparedBackendActive == 0 {
		renvoEnsureAppendAddrHelper(g)
	}
	if !renvoEmitStringValueRegs(g, ep, valueIndex) {
		return false
	}
	renvoAsmPushStringRegs(a)
	if !renvoEmitAppendDestPrimary(g, locEp, loc, 16) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopStoreStringMemSecondary(a, 0)
	return true
}
func renvoSetSliceLocationFromExpr(g *renvoLinearGen, ep *renvoExprParse, idx int, loc *renvoSliceLocation) {
	renvoNonNil(g, ep, loc)
	meta := g.meta
	renvoNonNil(meta)
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
			kind := renvoResolveType(meta, globalType).kind
			renvoNonNil(kind)
			if globalOffset < 0 || kind != renvoTypeSlice {
				return
			}
			loc.offset = globalOffset
			loc.typ = globalType
			loc.global = true
			loc.ok = true
			return
		}
		kind := renvoResolveType(meta, g.locals[localIndex].typ).kind
		renvoNonNil(kind)
		if kind != renvoTypeSlice {
			return
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.param = renvoLocalIsCurrentFuncParam(g, localIndex)
		loc.ok = true
		return
	}
	if e.kind == renvoExprSelector {
		fieldType := renvoInferParsedExprType(g, ep, idx)
		kind := renvoResolveType(g.meta, fieldType).kind
		renvoNonNil(kind)
		if kind != renvoTypeSlice {
			return
		}
		loc.expr = idx
		loc.typ = fieldType
		loc.mem = true
		loc.ok = true
		return
	}
	if e.kind == renvoExprIndex {
		valueType := renvoInferParsedExprType(g, ep, idx)
		kind := renvoResolveType(g.meta, valueType).kind
		renvoNonNil(kind)
		if kind != renvoTypeSlice {
			return
		}
		loc.expr = idx
		loc.typ = valueType
		loc.mem = true
		loc.ok = true
		return
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(g.meta, valueType) {
			return
		}
		loc.expr = e.left
		loc.typ = valueType
		loc.mem = true
		loc.deref = true
		loc.ok = true
		return
	}
}
func renvoEmitEnsureMemSlice(g *renvoLinearGen, elemSize int) {
	renvoNonNil(g)
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmJnzPrimary(a, okLabel)
	backingSize := renvoAsmSliceBackingSize(a, elemSize)
	label := renvoEnsureArenaAllocHelper(g)
	renvoAsmPrimaryImm(a, backingSize)
	renvoAsmCallLabel(a, label)
	renvoEmitArenaAllocationCheck(g)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmPrimaryImm(a, backingSize/elemSize)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 16)
	renvoAsmMarkLabel(a, okLabel)
}
func renvoEmitAppendStructCompositeTokens(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemType int, typeTok int) bool {
	renvoNonNil(g, locEp, loc)
	p := g.prog
	openTok := typeTok + 1
	closeTok := renvoSkipBalanced(p, openTok, '{', '}')
	if closeTok <= openTok {
		return false
	}
	elemSize := renvoTypeSize(g.meta, elemType)
	if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	destOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(&g.asm, destOffset)
	i := openTok + 1
	for i < closeTok-1 {
		if !renvoTokIsKind(p, i, renvoTokIdent) || !renvoTokCharIs(p, i+1, ':') {
			return false
		}
		fieldTok := renvoTokAt(p, i)
		exprStart := i + 2
		exprEnd := renvoFindExprBoundary(p, exprStart, closeTok-1)
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		if !renvoParseExpressionOK(ep, p, exprStart, exprEnd) {
			return false
		}
		fieldOffset := renvoStructFieldOffset(g, elemType, int(fieldTok.start), int(fieldTok.end))
		if fieldOffset < 0 {
			return false
		}
		fieldType := renvoStructFieldType(g, elemType, int(fieldTok.start), int(fieldTok.end))
		if fieldType == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !renvoEmitCompositeFieldToMem(g, ep, rootIndex, fieldType, destOffset, fieldOffset) {
			return false
		}
		i = exprEnd
		if renvoTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}
func renvoEmitAppendStructDeref(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemType int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	value := &ep.exprs[valueIndex]
	valueType := renvoInferParsedExprType(g, ep, valueIndex)
	if !renvoTypeIsStruct(g.meta, valueType) || renvoTypeSize(g.meta, valueType) != renvoTypeSize(g.meta, elemType) {
		return false
	}
	elemSize := renvoTypeSize(g.meta, elemType)
	tempOffset := renvoAddUnnamedLocal(g, elemType)
	if !renvoEmitIntExpr(g, ep, value.left) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoEmitCopyMemSecondaryToStack(g, tempOffset, elemSize)
	if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
	return true
}
func renvoEmitAppendStructLocal(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemType int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	value := &ep.exprs[valueIndex]
	localIndex := renvoFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := renvoTypeSize(g.meta, elemType)
	if renvoTypeSize(g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoEmitCopyStackToMemSecondary(g, g.locals[localIndex].offset, 0, elemSize)
	return true
}
func renvoEmitAppendStructComposite(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemType int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	elemSize := renvoTypeSize(g.meta, elemType)
	tempOffset := renvoAddUnnamedLocal(g, elemType)
	if !renvoEmitTypedAssign(g, ep, valueIndex, tempOffset) {
		return false
	}
	if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
	return true
}
func renvoStringOrderingExpr(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	p := g.prog
	if !renvoTokCharIs(p, e.tok, '<') && !renvoTokCharIs(p, e.tok, '>') && !renvoTok2Is(p, e.tok, '<', '=') && !renvoTok2Is(p, e.tok, '>', '=') {
		return false
	}
	return renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, e.left)) || renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, e.right))
}

func renvoEmitStringOrdering(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	a := &g.asm
	left := renvoAddUnnamedLocal(g, renvoTypeString)
	right := renvoAddUnnamedLocal(g, renvoTypeString)
	if !renvoEmitStringValueRegs(g, ep, e.left) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, left, left-8)
	if !renvoEmitStringValueRegs(g, ep, e.right) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, right, right-8)
	index := renvoAddUnnamedLocal(g, renvoTypeInt)
	lbyte := renvoAddUnnamedLocal(g, renvoTypeInt)
	rbyte := renvoAddUnnamedLocal(g, renvoTypeInt)
	loop := renvoAsmNewLabel(a)
	lengths := renvoAsmNewLabel(a)
	less := renvoAsmNewLabel(a)
	greater := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	renvoAsmStoreStackImm(a, index, 0)
	renvoAsmMarkLabel(a, loop)
	renvoAsmJgeStackStack(a, index, left-8, lengths)
	renvoAsmJgeStackStack(a, index, right-8, lengths)
	renvoAsmLoadPrimaryTertiaryStack(a, left, index)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
	renvoAsmStorePrimaryStack(a, lbyte)
	renvoAsmLoadPrimaryTertiaryStack(a, right, index)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
	renvoAsmStorePrimaryStack(a, rbyte)
	renvoAsmJcmpStackStack(a, lbyte, rbyte, less, 0x9c)
	renvoAsmJcmpStackStack(a, lbyte, rbyte, greater, 0x9f)
	renvoAsmIncStack(a, index)
	renvoAsmJmpMarkLabel(a, loop, lengths)
	renvoAsmJcmpStackStack(a, left-8, right-8, less, 0x9c)
	renvoAsmJcmpStackStack(a, left-8, right-8, greater, 0x9f)
	equalValue := 0
	if renvoTok2Is(g.prog, e.tok, '<', '=') || renvoTok2Is(g.prog, e.tok, '>', '=') {
		equalValue = 1
	}
	lessValue := 0
	if renvo_runtime_UnsafeByteAt(g.prog.src, int(renvoTokStart(g.prog, e.tok))) == '<' {
		lessValue = 1
	}
	renvoAsmPrimaryImm(a, equalValue)
	renvoAsmJmpMarkLabel(a, done, less)
	renvoAsmPrimaryImm(a, lessValue)
	renvoAsmJmpMarkLabel(a, done, greater)
	renvoAsmPrimaryImm(a, 1-lessValue)
	renvoAsmMarkLabel(a, done)
	return true
}

func renvoEmitStringCompare(g *renvoLinearGen, ep *renvoExprParse, left int, right int, notEqual bool) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	label := renvoEnsureStringEqualHelper(g)
	if renvoPreparedBackendActive != 0 {
		// A prepared target may use ABI argument registers which overlap the
		// primary/secondary value pair. Preserve both descriptors in the frame
		// before populating the four call words so no move can destroy a later
		// argument (AArch64 uses x0/x1 for both roles).
		leftOff := renvoAddUnnamedLocal(g, renvoTypeString)
		rightOff := renvoAddUnnamedLocal(g, renvoTypeString)
		if !renvoEmitStringValueRegs(g, ep, left) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, leftOff, leftOff-8)
		if !renvoEmitStringValueRegs(g, ep, right) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, rightOff, rightOff-8)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord0, leftOff)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord1, leftOff-8)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord2, rightOff)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord3, rightOff-8)
		renvoAsmCallLabel(a, label)
		if notEqual {
			renvoAsmBoolNotPrimary(a)
		}
		return true
	}
	rightExpr := &ep.exprs[right]
	if rightExpr.kind == renvoExprSelector {
		leftOff := renvoAddUnnamedLocal(g, renvoTypeString)
		rightOff := renvoAddUnnamedLocal(g, renvoTypeString)
		if !renvoEmitStringValueRegs(g, ep, left) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, leftOff, leftOff-8)
		if !renvoEmitStringValueRegs(g, ep, right) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, rightOff, rightOff-8)
		renvoAsmLoadPrimarySecondaryStack(a, leftOff, leftOff-8)
		renvoAsmPushStringRegs(a)
		renvoAsmLoadPrimarySecondaryStack(a, rightOff, rightOff-8)
		renvoAsmCopySecondaryToTertiary(a)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmPopCallWord1(a)
		renvoAsmCallLabel(a, label)
		if notEqual {
			renvoAsmBoolNotPrimary(a)
		}
		return true
	}
	if !renvoEmitStringValueRegs(g, ep, left) {
		return false
	}
	renvoAsmPushStringRegs(a)
	if !renvoEmitStringValueRegs(g, ep, right) {
		return false
	}
	renvoAsmCopySecondaryToTertiary(a)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopCallWord0(a)
	renvoAsmPopCallWord1(a)
	renvoAsmCallLabel(a, label)
	if notEqual {
		renvoAsmBoolNotPrimary(a)
	}
	return true
}
func renvoEmitCompositeCompare(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, typ int) bool {
	renvoNonNil(g, ep, e)
	left := renvoAddUnnamedLocal(g, typ)
	right := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitTypedAssign(g, ep, e.left, left) {
		return false
	}
	if !renvoEmitTypedAssign(g, ep, e.right, right) {
		return false
	}
	fail := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoEmitCompositeCompareAt(g, typ, left, right, fail)
	renvoAsmPrimaryImm(&g.asm, 1)
	renvoAsmJmpMarkLabel(&g.asm, done, fail)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmMarkLabel(&g.asm, done)
	if renvoTok2Is(g.prog, e.tok, '!', '=') {
		renvoAsmBoolNotPrimary(&g.asm)
	}
	return true
}
func renvoEmitCompositeCompareAt(g *renvoLinearGen, typ int, left int, right int, fail int) {
	renvoNonNil(g)
	a := &g.asm
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypeStruct {
		for i := 0; i < t.count; i++ {
			field := g.meta.fields[t.first+i]
			if field.nameEnd == field.nameStart+1 && renvo_runtime_UnsafeByteAt(g.prog.src, field.nameStart) == '_' {
				continue
			}
			renvoEmitCompositeCompareAt(g, field.typ, left-field.offset, right-field.offset, fail)
		}
		return
	}
	if t.kind == renvoTypeArray {
		size := renvoTypeSize(g.meta, t.elem)
		for i := 0; i < t.count; i++ {
			renvoEmitCompositeCompareAt(g, t.elem, left-i*size, right-i*size, fail)
		}
		return
	}
	if t.kind == renvoTypeString {
		if renvoPreparedBackendActive != 0 {
			renvoRTGAsmLoadFrame(a, renvoRTGCallWord0, left)
			renvoRTGAsmLoadFrame(a, renvoRTGCallWord1, left-renvoBackendValueSlotSize)
			renvoRTGAsmLoadFrame(a, renvoRTGCallWord2, right)
			renvoRTGAsmLoadFrame(a, renvoRTGCallWord3, right-renvoBackendValueSlotSize)
		} else {
			renvoAsmLoadPrimarySecondaryStack(a, left, left-8)
			renvoAsmPushStringRegs(a)
			renvoAsmLoadPrimarySecondaryStack(a, right, right-8)
			renvoAsmCopySecondaryToTertiary(a)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopCallWord0(a)
			renvoAsmPopCallWord1(a)
		}
		renvoAsmCallLabel(a, renvoEnsureStringEqualHelper(g))
	} else if t.kind == renvoTypeComplex64 {
		renvoEmit32IEEECompareStack(g, left, right, renvoTypeFloat32, '=', '=')
		renvoAsmJzPrimary(a, fail)
		renvoEmit32IEEECompareStack(g,
			renvoComplexSecondaryStackOffset(g, typ, left),
			renvoComplexSecondaryStackOffset(g, typ, right),
			renvoTypeFloat32, '=', '=')
	} else if t.kind == renvoTypeComplex {
		renvoEmit32IEEECompareStack(g, left, right, renvoTypeFloat64, '=', '=')
		renvoAsmJzPrimary(a, fail)
		renvoEmit32IEEECompareStack(g, left-8, right-8, renvoTypeFloat64, '=', '=')
	} else {
		kind := t.kind
		if kind == renvoTypeBool {
			kind = renvoTypeByte
		}
		renvoAsmLoadPrimaryStack(a, left)
		renvoAsmNormalizePrimaryForKind(a, kind)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryStack(a, right)
		renvoAsmNormalizePrimaryForKind(a, kind)
		renvoAsmPopTertiary(a)
		renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	}
	renvoAsmJzPrimary(a, fail)
}
func renvoEmitBuiltinCopy(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	destIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	srcIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)
	destType := renvoInferParsedExprType(g, ep, destIndex)
	srcType := renvoInferParsedExprType(g, ep, srcIndex)
	destSlice := renvoResolveType(g.meta, destType)
	renvoNonNil(destSlice)
	srcSlice := renvoResolveType(g.meta, srcType)
	renvoNonNil(srcSlice)
	stringSource := srcSlice.kind == renvoTypeString
	if destSlice.kind != renvoTypeSlice || srcSlice.kind != renvoTypeSlice && !stringSource {
		return false
	}
	elemSize := renvoTypeSize(g.meta, destSlice.elem)
	if stringSource && renvoResolveType(g.meta, destSlice.elem).kind != renvoTypeByte || !stringSource && elemSize != renvoTypeSize(g.meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	destLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	copyLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	copyBytes := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, destPtr, destLen)
	if stringSource {
		if !renvoEmitStringValueRegs(g, ep, srcIndex) {
			return false
		}
	} else {
		if !renvoEmitSliceValueRegs(g, ep, srcIndex) {
			return false
		}
	}
	renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
	renvoAsmCopyStackSlot(a, destLen, copyLen)
	useSourceLen := renvoAsmNewLabel(a)
	lengthReady := renvoAsmNewLabel(a)
	renvoAsmJgeStackStack(a, destLen, srcLen, useSourceLen)
	renvoAsmJmpMarkLabel(a, lengthReady, useSourceLen)
	renvoAsmCopyStackSlot(a, srcLen, copyLen)
	renvoAsmMarkLabel(a, lengthReady)
	renvoAsmLoadTertiaryStack(a, copyLen)
	renvoAsmMulTertiaryImm(a, elemSize)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, copyBytes)
	renvoEmitCopyBytes(g, srcPtr, destPtr, copyBytes)
	renvoAsmLoadPrimaryStack(a, copyLen)
	return true
}

func renvoIsSliceArrayConversion(g *renvoLinearGen, ep *renvoExprParse, arg int, target int) bool {
	typ := renvoResolveType(g.meta, target)
	if typ.kind == renvoTypePointer {
		typ = renvoResolveType(g.meta, typ.elem)
	}
	return typ.kind == renvoTypeArray && renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg)).kind == renvoTypeSlice
}

func renvoEmitSliceArrayConversion(g *renvoLinearGen, ep *renvoExprParse, arg int, target int, offset int) bool {
	typ := renvoResolveType(g.meta, target)
	pointer := typ.kind == renvoTypePointer
	if pointer {
		typ = renvoResolveType(g.meta, typ.elem)
	}
	source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
	if typ.kind != renvoTypeArray || source.kind != renvoTypeSlice || !renvoTypesEquivalent(g.meta, typ.elem, source.elem) {
		return false
	}
	ptr := renvoAddUnnamedLocal(g, renvoTypeInt)
	length := renvoAddUnnamedLocal(g, renvoTypeInt)
	minimum := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitSliceValueRegs(g, ep, arg) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(&g.asm, ptr, length)
	renvoAsmStoreStackImm(&g.asm, minimum, typ.count)
	valid := renvoAsmNewLabel(&g.asm)
	renvoAsmJgeStackStack(&g.asm, length, minimum, valid)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(&g.asm, valid)
	if pointer {
		renvoAsmLoadPrimaryStack(&g.asm, ptr)
	} else {
		renvoAsmLoadSecondaryStack(&g.asm, ptr)
		renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, target))
	}
	return true
}
func renvoEmitDirectSelectorWords(g *renvoLinearGen, ep *renvoExprParse, idx int, primaryDisp int, tertiaryDisp int, size int) bool {
	renvoNonNil(g, ep)
	if !renvoAsmFoldedFieldAddressing(&g.asm) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprSelector {
		return false
	}
	base := &ep.exprs[e.left]
	if base.kind != renvoExprIdent {
		return false
	}
	baseType := renvoInferParsedExprType(g, ep, e.left)
	if !renvoLoadStructFieldPath(g, baseType, e.nameStart, e.nameEnd) || g.fieldPointerIndex >= 0 {
		return false
	}
	fieldOffset := g.fieldOffset
	localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
	if localIndex < 0 || renvoResolveType(g.meta, g.locals[localIndex].typ).kind != renvoTypePointer {
		return false
	}
	renvoAsmLoadSecondaryStack(&g.asm, g.locals[localIndex].offset)
	needCheck := renvoRuntimeNonNilLocalNeeded(g, localIndex)
	primaryOffset := fieldOffset + primaryDisp
	tertiaryOffset := -1
	if tertiaryDisp >= 0 {
		tertiaryOffset = fieldOffset + tertiaryDisp
	}
	if needCheck {
		renvoEmitRuntimeNonNilSecondary(g)
	}
	renvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, primaryOffset, size)
	if tertiaryOffset >= 0 {
		renvoAsmLoadTertiaryMemSecondaryDisp(&g.asm, tertiaryOffset)
	}
	return true
}

func renvoEmitSlicePtrLen(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprSlice {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
			globalKind := renvoResolveType(meta, globalType).kind
			renvoNonNil(globalKind)
			if globalOffset < 0 || (globalKind != renvoTypeSlice && globalKind != renvoTypeString) {
				return false
			}
			renvoAsmLoadPrimaryBss(a, globalOffset+8)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		localKind := renvoResolveType(meta, g.locals[localIndex].typ).kind
		renvoNonNil(localKind)
		if localKind != renvoTypeSlice && localKind != renvoTypeString {
			return false
		}
		renvoAsmLoadPrimaryTertiaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == renvoExprComposite {
		sliceType := renvoTypeFromExpr(g, ep, idx)
		if !renvoTypeIsSlice(meta, sliceType) {
			return false
		}
		if !renvoEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		renvoAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == renvoExprSelector {
		fieldType := renvoInferParsedExprType(g, ep, idx)
		fieldKind := renvoResolveType(meta, fieldType).kind
		renvoNonNil(fieldKind)
		if fieldKind != renvoTypeSlice && fieldKind != renvoTypeString {
			return false
		}
		if renvoEmitDirectSelectorWords(g, ep, idx, 0, 8, g.c.renvoNativeIntSize) {
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmLoadTertiaryMemSecondaryDisp(a, 8)
		return true
	}
	if e.kind == renvoExprIndex {
		if !renvoEmitSliceValueRegs(g, ep, idx) && !renvoEmitStringValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == renvoExprCall {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, valueType) {
			return false
		}
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoTypeIsSlice(meta, renvoInferParsedExprType(g, ep, idx)) || !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmCopySecondaryToTertiary(a)
		return true
	}
	return false
}
func renvoEmitSlicePtrCap(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprSlice {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !renvoTypeIsSlice(meta, globalType) {
				return false
			}
			renvoAsmLoadPrimaryBss(a, globalOffset+16)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		if !renvoTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		renvoAsmLoadPrimaryTertiaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == renvoExprComposite {
		sliceType := renvoTypeFromExpr(g, ep, idx)
		if !renvoTypeIsSlice(meta, sliceType) {
			return false
		}
		if !renvoEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		return true
	}
	if e.kind == renvoExprSelector {
		fieldType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsSlice(meta, fieldType) {
			return false
		}
		if renvoEmitDirectSelectorWords(g, ep, idx, 0, 16, g.c.renvoNativeIntSize) {
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 16)
		renvoAsmPopPrimaryToTertiary(a)
		return true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') || e.kind == renvoExprIndex || e.kind == renvoExprCall {
		if !renvoTypeIsSlice(meta, renvoInferParsedExprType(g, ep, idx)) {
			return false
		}
		return renvoEmitSliceValueRegs(g, ep, idx)
	}
	return false
}
func renvoEmitIndexAddressPrimary(g *renvoLinearGen, ep *renvoExprParse, indexIdx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	renvoNonNil(meta)
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	sliceType := renvoResolveType(meta, renvoInferParsedExprType(g, ep, indexExpr.left))
	renvoNonNil(sliceType)
	if sliceType.kind == renvoTypePointer {
		elem := renvoResolveType(meta, sliceType.elem)
		if elem.kind != renvoTypeArray && elem.kind != renvoTypeSlice {
			return renvoEmitCPointerIndexAddressPrimary(g, ep, indexIdx, sliceType)
		}
	}
	pointerArray := sliceType.kind == renvoTypePointer
	if pointerArray {
		sliceType = renvoResolveType(meta, sliceType.elem)
	}
	if sliceType.kind != renvoTypeArray && sliceType.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(meta, sliceType.elem)
	baseExpr := &ep.exprs[indexExpr.left]
	if pointerArray {
		if !renvoEmitIntExpr(g, ep, indexExpr.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmPushPrimary(a)
		renvoAsmPrimaryImm(a, sliceType.count)
		renvoAsmPopPrimaryToTertiary(a)
	} else if sliceType.kind == renvoTypeArray {
		base := baseExpr
		if base.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
			if localIndex < 0 {
				globalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)
				globalType := renvoResolveType(g.meta, renvoFindGlobalType(g, base.nameStart, base.nameEnd))
				renvoNonNil(globalType)
				if globalOffset < 0 || globalType.kind != renvoTypeArray {
					return false
				}
				renvoAsmPrimaryBssAddr(a, globalOffset)
			} else {
				renvoAsmAddressPrimaryStack(a, g.locals[localIndex].offset)
			}
		} else if base.kind == renvoExprIndex {
			if !renvoEmitIndexAddressPrimary(g, ep, indexExpr.left) {
				return false
			}
		} else if base.kind == renvoExprSelector {
			if !renvoEmitSelectorAddressSecondary(g, ep, indexExpr.left) {
				return false
			}
			renvoAsmCopySecondaryToPrimary(a)
		} else if base.kind == renvoExprUnary && renvoTokCharIs(g.prog, base.tok, '*') {
			if !renvoEmitAddressPrimary(g, ep, indexExpr.left) {
				return false
			}
		} else if base.kind == renvoExprCall || base.kind == renvoExprComposite {
			baseType := renvoInferParsedExprType(g, ep, indexExpr.left)
			tempOffset := renvoAddUnnamedLocal(g, baseType)
			if !renvoEmitTypedAssign(g, ep, indexExpr.left, tempOffset) {
				return false
			}
			renvoAsmAddressPrimaryStack(a, tempOffset)
		} else {
			return false
		}
		renvoAsmPushPrimary(a)
		renvoAsmPrimaryImm(a, sliceType.count)
		renvoAsmPopPrimaryToTertiary(a)
	} else {
		if !renvoEmitSlicePtrLen(g, ep, indexExpr.left) {
			return false
		}
	}
	renvoAsmPushPrimary(a)
	renvoAsmPushTertiary(a)
	if !renvoEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	if !g.meta.panicEnabled && (elemSize == 1 || elemSize == 8 || elemSize == 72) {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmPopSecondary(a)
		renvoAsmPopPrimary(a)
		renvoEmitCheckedIndexAddress(g, elemSize)
		return true
	}
	renvoAsmPopTertiary(a)
	renvoEmitRuntimeBoundsCheck(g)
	renvoAsmCopySecondaryToTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmAddScaledTertiary(a, elemSize)
	return true
}

func renvoEmitCPointerIndexAddressPrimary(g *renvoLinearGen, ep *renvoExprParse, indexIdx int, pointerType *renvoTypeInfo) bool {
	indexExpr := &ep.exprs[indexIdx]
	if !renvoEmitIntExpr(g, ep, indexExpr.left) {
		return false
	}
	renvoEmitRuntimeNonNilPrimary(g)
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	renvoAsmCopyPrimaryToTertiary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmAddScaledTertiary(&g.asm, renvoTypeSize(g.meta, pointerType.elem))
	return true
}

func renvoEnsureIndexAddressHelper(g *renvoLinearGen, elemSize int) int {
	renvoNonNil(g)
	if targetLabel := renvoEmitTargetIndexAddressHelper(g, elemSize); targetLabel >= 0 {
		return targetLabel
	}
	if elemSize == 1 && g.runtimeByteIndexLabel > 0 {
		return g.runtimeByteIndexLabel - 1
	}
	if elemSize == 8 && g.runtimeWordIndexLabel > 0 {
		return g.runtimeWordIndexLabel - 1
	}
	if elemSize == 72 && g.runtimeWideIndexLabel > 0 {
		return g.runtimeWideIndexLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	if elemSize == 1 {
		g.runtimeByteIndexLabel = label + 1
	} else if elemSize == 8 {
		g.runtimeWordIndexLabel = label + 1
	} else {
		g.runtimeWideIndexLabel = label + 1
	}
	if renvoRTGStructuredFunctions != 0 {
		renvoQueueStructuredHelper(g, renvoStructuredHelperIndexAddress, elemSize, label)
		return label
	}
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoEmitIndexAddressHelperBody(g, elemSize)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitRuntimeBoundsCheck(g *renvoLinearGen) {
	renvoNonNil(g)
	if !g.meta.panicEnabled {
		renvoEmitUncheckedBoundsCheck(g)
		return
	}
	done := renvoAsmNewLabel(&g.asm)
	renvoEmitBoundsSuccessBranch(g, done)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(&g.asm, done)
}

func renvoEmitSliceBoundsChecks(g *renvoLinearGen, lowOff int, highOff int, maxOff int, capOff int) {
	renvoNonNil(g)
	a := &g.asm
	if renvoEmitOptimizedSliceBoundsChecks(g, lowOff, highOff, maxOff, capOff) {
		return
	}
	invalid := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	renvoEmitStackLessImmJump(g, lowOff, 0, invalid)
	renvoAsmJltStackStack(a, highOff, lowOff, invalid)
	renvoAsmJltStackStack(a, maxOff, highOff, invalid)
	renvoAsmJltStackStack(a, capOff, maxOff, invalid)
	renvoAsmJmpMarkLabel(a, done, invalid)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(a, done)
}

func renvoEnsureBoundsCheckHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if targetLabel := renvoEmitTargetBoundsCheckHelper(g); targetLabel >= 0 {
		return targetLabel
	}
	if g.runtimeBoundsLabel > 0 {
		return g.runtimeBoundsLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.runtimeBoundsLabel = label + 1
	if renvoRTGStructuredFunctions != 0 {
		renvoQueueStructuredHelper(g, renvoStructuredHelperBoundsCheck, 0, label)
		return label
	}
	after := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoEmitBoundsCheckHelperBody(g)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitIndexExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	baseResolved := renvoResolveType(meta, renvoInferParsedExprType(g, ep, e.left))
	renvoNonNil(baseResolved)
	if baseResolved.kind == renvoTypePointer {
		baseResolved = renvoResolveType(meta, baseResolved.elem)
	}
	if baseResolved.kind == renvoTypeString {
		if !renvoEmitStringValueRegs(g, ep, e.left) {
			return false
		}
		renvoAsmPushPrimary(a)
		renvoAsmPushSecondary(a)
		if !renvoEmitIntExpr(g, ep, e.right) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoEmitRuntimeBoundsCheck(g)
		renvoAsmCopySecondaryToTertiary(a)
		renvoAsmPopPrimary(a)
		renvoAsmLoadBytePrimaryIndexTertiary(a)
		return true
	}
	if baseResolved.kind == renvoTypeArray || baseResolved.kind == renvoTypeSlice {
		elem := renvoResolveType(meta, baseResolved.elem)
		renvoNonNil(elem)
		if !renvoTypeKindIsScalarValue(elem.kind) && elem.kind != renvoTypePointer && elem.kind != renvoTypeFunc {
			return false
		}
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(g.c.renvoNativeIntSize, elem.kind))
		// Size-only loads sign-extend halfwords. Restore the parsed element's
		// signedness before its value reaches comparisons or wider arithmetic.
		renvoAsmNormalizePrimaryForKind(a, elem.kind)
		return true
	}
	return false
}
func renvoFindLocalOffset(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	localIndex := renvoFindLocalIndex(g, nameStart, nameEnd)
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}

// A return list assigns to the result declarations, even when a lexical local
// with the same spelling is visible while evaluating the return expressions.
func renvoFindResultLocalOffset(g *renvoLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		if local.nameStart == nameStart && local.nameEnd == nameEnd {
			return local.offset
		}
	}
	return -1
}

func renvoFindLocalIndex(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if g.localCacheStart == nameStart {
		if g.localCacheIndex >= 0 && g.localCacheIndex < g.localCount {
			local := &g.locals[g.localCacheIndex]
			if renvoBytesEqualRange(g.prog.src, local.nameStart, local.nameEnd, nameStart, nameEnd) {
				return g.localCacheIndex
			}
		}
	}
	nameHash := renvoHashRange(g.prog.src, nameStart, nameEnd)
	g.localCacheStart = nameStart
	g.localCacheCount = g.localCount
	g.localCacheIndex = -1
	for i := g.localCount - 1; i >= 0; i-- {
		local := &g.locals[i]
		if local.nameHash == nameHash && renvoBytesEqualRange(g.prog.src, local.nameStart, local.nameEnd, nameStart, nameEnd) {
			g.localCacheIndex = i
			return i
		}
	}
	return -1
}

func renvoLoadStructFieldPath(g *renvoLinearGen, typ int, nameStart int, nameEnd int) bool {
	renvoNonNil(g)
	// A source position identifies one selector occurrence and therefore one
	// base type throughout a compilation.
	if g.fieldCacheStart == nameStart {
		return g.fieldIndex >= 0
	}
	g.fieldCacheStart = nameStart
	g.fieldIndex = -1
	g.fieldPointerIndex = -1
	return renvoFindStructFieldPath(g, typ, nameStart, nameEnd, 0, 0, -1, 0, 0) == 1
}

func renvoFindStructFieldPath(g *renvoLinearGen, typ int, nameStart int, nameEnd int, beforePointer int, afterPointer int, pointerIndex int, pointerOffset int, depth int) int {
	renvoNonNil(g)
	meta := g.meta
	if typ < 0 || typ >= len(meta.types) || depth > 16 {
		return 0
	}
	t := renvoResolveType(meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypePointer && t.elem > 0 && t.elem < len(meta.types) {
		t = renvoResolveType(meta, t.elem)
	}
	if t.kind != renvoTypeStruct {
		return 0
	}
	fields := meta.fields
	for i := 0; i < t.count; i++ {
		fieldIndex := t.first + i
		field := fields[fieldIndex]
		if renvoBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			g.fieldIndex = fieldIndex
			g.fieldPointerIndex = pointerIndex
			g.fieldPointerOffset = pointerOffset
			g.fieldOffset = beforePointer + field.offset
			if pointerIndex >= 0 {
				g.fieldOffset = afterPointer + field.offset
			}
			return 1
		}
	}
	found := 0
	for i := 0; i < t.count; i++ {
		fieldIndex := t.first + i
		field := fields[fieldIndex]
		if !field.embedded {
			continue
		}
		nextBefore := beforePointer
		nextAfter := afterPointer
		nextPointer := pointerIndex
		nextPointerOffset := pointerOffset
		if pointerIndex >= 0 {
			nextAfter += field.offset
		} else if renvoResolveType(meta, field.typ).kind == renvoTypePointer {
			nextPointer = fieldIndex
			nextPointerOffset = beforePointer + field.offset
			nextAfter = 0
		} else {
			nextBefore += field.offset
		}
		found += renvoFindStructFieldPath(g, field.typ, nameStart, nameEnd, nextBefore, nextAfter, nextPointer, nextPointerOffset, depth+1)
		if found > 1 {
			return found
		}
	}
	return found
}
func renvoStructFieldOffset(g *renvoLinearGen, typ int, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if !renvoLoadStructFieldPath(g, typ, nameStart, nameEnd) {
		return -1
	}
	if g.fieldPointerIndex >= 0 {
		return g.fieldPointerOffset + g.fieldOffset
	}
	return g.fieldOffset
}
func renvoStructFieldType(g *renvoLinearGen, typ int, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if !renvoLoadStructFieldPath(g, typ, nameStart, nameEnd) {
		return 0
	}
	return g.meta.fields[g.fieldIndex].typ
}

func renvoStructPromotedPointerField(g *renvoLinearGen, typ int, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if renvoLoadStructFieldPath(g, typ, nameStart, nameEnd) {
		return g.fieldPointerIndex
	}
	return -1
}

func renvoEmitPromotedPointerSelectorAddress(g *renvoLinearGen, ep *renvoExprParse, idx int, baseType int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if !renvoLoadStructFieldPath(g, baseType, e.nameStart, e.nameEnd) || g.fieldPointerIndex < 0 {
		return false
	}
	pointerOffset := g.fieldPointerOffset
	fieldOffset := g.fieldOffset
	a := &g.asm
	base := &ep.exprs[e.left]
	if base.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex >= 0 {
			if renvoResolveType(g.meta, g.locals[localIndex].typ).kind == renvoTypePointer {
				renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
				renvoEmitRuntimeNonNilPrimary(g)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDisp(a, pointerOffset)
			} else {
				renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset-pointerOffset)
			}
		} else {
			globalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)
			if globalOffset < 0 {
				return false
			}
			if renvoResolveType(g.meta, renvoFindGlobalType(g, base.nameStart, base.nameEnd)).kind == renvoTypePointer {
				renvoAsmLoadPrimaryBss(a, globalOffset)
				renvoEmitRuntimeNonNilPrimary(g)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDisp(a, pointerOffset)
			} else {
				renvoAsmLoadPrimaryBss(a, globalOffset+pointerOffset)
			}
		}
		renvoAsmCopyPrimaryToSecondary(a)
	} else if base.kind == renvoExprComposite {
		offset := renvoAddUnnamedLocal(g, baseType)
		if !renvoEmitTypedAssign(g, ep, e.left, offset) {
			return false
		}
		renvoAsmLoadSecondaryStack(a, offset-pointerOffset)
	} else {
		return false
	}
	renvoEmitRuntimeNonNilSecondary(g)
	if fieldOffset != 0 {
		renvoAsmAddSecondaryImm(a, fieldOffset)
	}
	return true
}
func renvoCompositeStructFieldIndex(g *renvoLinearGen, typ int, field *renvoCompositeField, pos int) int {
	renvoNonNil(g, field)
	if field.nameEnd > field.nameStart {
		if renvoLoadStructFieldPath(g, typ, field.nameStart, field.nameEnd) {
			return g.fieldIndex
		}
		return -1
	}
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind != renvoTypeStruct || pos < 0 || pos >= t.count {
		return -1
	}
	return t.first + pos
}
func renvoFindGlobalOffset(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	for i := 0; i < len(g.globals); i++ {
		global := &g.globals[i]
		if renvoBytesEqualRange(g.prog.src, global.nameStart, global.nameEnd, nameStart, nameEnd) {
			return global.offset
		}
	}
	return -1
}
func renvoFindGlobalType(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	symIndex := renvoFindMetaGlobalIndex(g.meta, nameStart, nameEnd, renvoTokVar)
	if symIndex >= 0 {
		return g.meta.globals[symIndex].typ
	}
	return 0
}
func renvoFindConstStringToken(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	symIndex := renvoFindMetaGlobalIndex(g.meta, nameStart, nameEnd, renvoTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.initStart+1 == s.initEnd && renvoTokIsKind(g.prog, s.initStart, renvoTokString) {
			return s.initStart
		}
	}
	return -1
}
func renvoFindSmallConstByName(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if renvoFindLocalIndex(g, nameStart, nameEnd) >= 0 {
		return -129
	}
	if renvoBytesEqualText(g.prog.src, nameStart, nameEnd, "nil") {
		return 0
	}
	symIndex := renvoFindMetaGlobalIndex(g.meta, nameStart, nameEnd, renvoTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.initStart+1 != s.initEnd {
			return -129
		}
		if renvoTokIsKind(g.prog, s.initStart, renvoTokNumber) {
			value := renvoParseConstIntToken(g.prog, s.initStart)
			if g.prog.compilerInt32 && g.prog.parsedIntHigh != value>>31 {
				return -129
			}
			if renvoAsmImmFits8Signed(value) {
				return value
			}
		}
		if renvoTokIsKind(g.prog, s.initStart, renvoTokChar) {
			value := renvoParseCharToken(g.prog, s.initStart)
			if renvoAsmImmFits8Signed(value) {
				return value
			}
		}
		return -129
	}
	return -129
}

func renvoTypeCanCarryLocalAddress(meta *renvoMeta, typ int, depth int) bool {
	if depth > 32 {
		return true
	}
	t := renvoResolveType(meta, typ)
	if t.kind == renvoTypePointer || t.kind == renvoTypeSlice || t.kind == renvoTypeInterface || t.kind == renvoTypeFunc {
		return true
	}
	if t.kind == renvoTypeArray {
		return renvoTypeCanCarryLocalAddress(meta, t.elem, depth+1)
	}
	if t.kind == renvoTypeStruct {
		for i := 0; i < t.count; i++ {
			if renvoTypeCanCarryLocalAddress(meta, meta.fields[t.first+i].typ, depth+1) {
				return true
			}
		}
	}
	return false
}

// An address into pointer or slice storage does not retain the local holding
// that pointer/header. Only paths entirely within a value own the local cell.
func renvoLocalStorageAddress(g *renvoLinearGen, name int, typ int, end int) bool {
	p := g.prog
	for tok := name + 1; tok < end; {
		if renvoTokCharIs(p, tok, ')') {
			tok++
			continue
		}
		t := renvoResolveType(g.meta, typ)
		if renvoTokCharIs(p, tok, '.') && tok+1 < end {
			if t.kind != renvoTypeStruct {
				return false
			}
			g.fieldCacheStart = -1
			found := renvoLoadStructFieldPath(g, typ, int(renvoTokStart(p, tok+1)), int(renvoTokEnd(p, tok+1)))
			g.fieldCacheStart = -1
			if !found || g.fieldPointerIndex >= 0 {
				return false
			}
			typ = g.meta.fields[g.fieldIndex].typ
			tok += 2
			continue
		}
		if renvoTokCharIs(p, tok, '[') {
			if t.kind != renvoTypeArray {
				return false
			}
			close := renvoFindMatchingExprClose(p, tok+1, end, '[', ']')
			if close <= tok {
				return false
			}
			typ = t.elem
			tok = close + 1
			continue
		}
		return true
	}
	return true
}

func renvoLocalStorageAddressTaken(g *renvoLinearGen, nameStart int, nameEnd int, typ int) bool {
	p := g.prog
	fn := &g.meta.funcs[g.currentFunc]
	if !g.addressNamesReady {
		g.addressNamesReady = true
		for tok := fn.bodyStart; tok+1 < fn.bodyEnd; tok++ {
			if !renvoTokCharIs(p, tok, '&') {
				continue
			}
			name := tok + 1
			for name < fn.bodyEnd && renvoTokCharIs(p, name, '(') {
				name++
			}
			if name < fn.bodyEnd && renvoTokIsKind(p, name, renvoTokIdent) {
				g.addressNameTokens = append(g.addressNameTokens, name)
			}
		}
	}
	for _, name := range g.addressNameTokens {
		if renvoBytesEqualRange(p.src, renvoTokStart(p, name), renvoTokEnd(p, name), nameStart, nameEnd) && renvoLocalStorageAddress(g, name, typ, fn.bodyEnd) {
			return true
		}
	}
	return false
}

func renvoLocalAddressInReturn(g *renvoLinearGen, nameStart int, nameEnd int) bool {
	p := g.prog
	fn := &g.meta.funcs[g.currentFunc]
	for tok := fn.bodyStart; tok < fn.bodyEnd; tok++ {
		if !renvoTokIsKind(p, tok, renvoTokReturn) {
			continue
		}
		end := renvoStatementLineEnd(p, tok+1, fn.bodyEnd)
		for i := tok + 1; i+1 < end; i++ {
			if !renvoTokCharIs(p, i, '&') {
				continue
			}
			name := i + 1
			if renvoTokCharIs(p, name, '(') {
				name++
			}
			if name < end && renvoTokIsKind(p, name, renvoTokIdent) && renvoBytesEqualRange(p.src, renvoTokStart(p, name), renvoTokEnd(p, name), nameStart, nameEnd) {
				return true
			}
		}
		tok = end - 1
	}
	return false
}

func renvoLocalAddressStored(g *renvoLinearGen, nameStart int, nameEnd int) bool {
	p := g.prog
	fn := &g.meta.funcs[g.currentFunc]
	for tok := fn.bodyStart; tok+1 < fn.bodyEnd; tok++ {
		if !renvoTokCharIs(p, tok, '&') {
			continue
		}
		name := tok + 1
		for renvoTokCharIs(p, name, '(') {
			name++
		}
		if !renvoTokIsKind(p, name, renvoTokIdent) || !renvoBytesEqualRange(p.src, renvoTokStart(p, name), renvoTokEnd(p, name), nameStart, nameEnd) {
			continue
		}
		before := tok - 1
		for before > fn.bodyStart && renvoTokCharIs(p, before, '(') {
			before--
		}
		if renvoTokCharIs(p, before, '=') {
			if before > fn.bodyStart && renvoTokIdentIs(p, before-1, "_") {
				continue
			}
			return true
		}
		if renvoTok2Is(p, before, ':', '=') || renvoTokCharIs(p, before, ':') {
			return true
		}
	}
	return false
}

func renvoLocalCapturedInCurrentFunction(g *renvoLinearGen, nameStart int, nameEnd int, typ int) bool {
	renvoNonNil(g)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	if nameEnd <= nameStart {
		return false
	}
	outer := &meta.funcs[g.currentFunc]
	if !renvoProgramUsesC11Semantics(p) && renvoLocalStorageAddressTaken(g, nameStart, nameEnd, typ) {
		// A stored address may outlive its lexical block even when this
		// function returns no pointer. Keep the cell alive after stack slots
		// are reused, and synchronize subsequent writes through either alias.
		if renvoLocalAddressStored(g, nameStart, nameEnd) {
			return true
		}
		// Returning a pointer-containing aggregate can retain a local's address
		// just as returning the address directly can (including closure cells).
		mayEscape := renvoResolveType(meta, outer.resultType).kind == renvoTypePointer
		if !mayEscape && renvoTypeCanCarryLocalAddress(meta, outer.resultType, 0) {
			mayEscape = renvoLocalAddressInReturn(g, nameStart, nameEnd)
		}
		// Range and three-clause declaration bindings get fresh cells on
		// iteration boundaries, just like captured loop variables.
		if !mayEscape && nameStart >= int(renvoTokStart(p, outer.bodyStart)) {
			for tok := outer.bodyStart; tok < outer.bodyEnd && int(renvoTokStart(p, tok)) < nameStart; tok++ {
				if !renvoTokIsKind(p, tok, renvoTokFor) {
					continue
				}
				open := renvoFindStatementBodyOpen(p, tok+1, outer.bodyEnd)
				if open > tok && nameStart < int(renvoTokStart(p, open)) {
					mayEscape = true
					break
				}
			}
		}
		if mayEscape && renvoLocalNameAddressTaken(g, nameStart, nameEnd) {
			return true
		}
	}
	for i := 0; i < len(meta.closures); i++ {
		fnIndex := meta.closures[i].fnIndex
		closure := &meta.funcs[fnIndex]
		if closure.literalTok < outer.bodyStart || closure.literalTok >= outer.bodyEnd || renvoClosureNameDeclared(meta, closure, nameStart, nameEnd) {
			continue
		}
		for tok := closure.bodyStart; tok < closure.bodyEnd; tok++ {
			if renvoTokIsKind(p, tok, renvoTokIdent) && renvoBytesEqualRange(p.src, nameStart, nameEnd, int(renvoTokStart(p, tok)), int(renvoTokEnd(p, tok))) {
				return true
			}
		}
	}
	return false
}

func renvoAddTypedLocal(g *renvoLinearGen, nameStart int, nameEnd int, typ int) int {
	renvoNonNil(g)
	size := renvoTypeCopySize(g.meta, typ)
	compactScalar := false
	if renvoProgramUsesC11Semantics(g.prog) {
		kind := renvoResolveType(g.meta, typ).kind
		compactScalar = renvoTypeSize(g.meta, typ) <= g.c.renvoNativeIntSize &&
			(renvoTypeKindIsScalarInt(kind) || kind == renvoTypePointer || kind == renvoTypeFunc)
	}
	unit := renvoLocalStorageUnit(g.c, compactScalar)
	if compactScalar || size < unit {
		size = unit
	}
	captureOff := 0
	if renvoLocalCapturedInCurrentFunction(g, nameStart, nameEnd, typ) {
		captureOff = -1
		if g.deferStackFloor >= 0 {
			g.stackUsed = renvoAlignTo8(g.stackUsed + renvoBackendValueSlotSize)
			renvoRecordStackPeak(g)
			captureOff = g.stackUsed
			renvoAllocateCapturedCell(g, captureOff, size)
		}
	}
	g.stackUsed = renvoAlignValue(g.stackUsed+size, unit)
	renvoRecordStackPeak(g)
	offset := g.stackUsed
	renvoRecordLocalStorage(g, offset, size, captureOff, typ)
	if g.localCount >= len(g.locals) {
		renvoGrowLocalTable(g)
	}
	nameHash := 0
	if nameEnd > nameStart {
		nameHash = renvoHashRange(g.prog.src, nameStart, nameEnd)
		// A named declaration can shadow a cached lookup. Unnamed expression
		// temporaries cannot, so retain positive lookup hits across those locals.
		g.localCacheStart = -1
	}
	g.locals[g.localCount] = renvoLocalInfo{nameStart: nameStart, nameEnd: nameEnd, nameHash: nameHash, offset: offset, captureOff: captureOff, typ: typ, size: size}
	g.localCount++
	if captureOff != 0 {
		g.hasCapturedLocals = true
	}
	return offset
}

func renvoRecordStackPeak(g *renvoLinearGen) {
	renvoNonNil(g)
	// Retain scoped locals and expression temporaries after stackUsed is
	// rewound. Balanced hardware push/pop operands live below the persistent
	// frame and therefore do not contribute to its size.
	if g.stackUsed > g.stackPeak {
		g.stackPeak = g.stackUsed
	}
}

func renvoAddUnnamedLocal(g *renvoLinearGen, typ int) int {
	renvoNonNil(g)
	return renvoAddTypedLocal(g, 0, 0, typ)
}

func renvoGrowLocalTable(g *renvoLinearGen) {
	renvoNonNil(g)
	newCap := len(g.locals) * 2
	if newCap < 64 {
		newCap = 64
	}
	newLocals := make([]renvoLocalInfo, newCap)
	for i := 0; i < g.localCount; i++ {
		newLocals[i] = g.locals[i]
	}
	g.locals = newLocals
}

func renvoMoveCapturedLocal(g *renvoLinearGen, localIndex int, toCell bool) {
	renvoNonNil(g)
	if localIndex < 0 || localIndex >= g.localCount || g.locals[localIndex].captureOff <= 0 {
		return
	}
	local := &g.locals[localIndex]
	skip := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, local.captureOff)
	renvoAsmJzPrimary(&g.asm, skip)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	if local.size >= 64 {
		oldCount, oldStack := g.localCount, g.stackUsed
		source := renvoAddUnnamedLocal(g, renvoTypeInt)
		destination := renvoAddUnnamedLocal(g, renvoTypeInt)
		count := renvoAddUnnamedLocal(g, renvoTypeInt)
		if toCell {
			renvoAsmStorePrimaryStack(&g.asm, destination)
			renvoAsmAddressPrimaryStack(&g.asm, local.offset)
			renvoAsmStorePrimaryStack(&g.asm, source)
		} else {
			renvoAsmStorePrimaryStack(&g.asm, source)
			renvoAsmAddressPrimaryStack(&g.asm, local.offset)
			renvoAsmStorePrimaryStack(&g.asm, destination)
		}
		renvoAsmStoreStackImm(&g.asm, count, local.size)
		renvoEmitCopyBytes(g, source, destination, count)
		g.localCount = oldCount
		renvoRestoreStackFloor(g, oldStack)
	} else if toCell {
		renvoEmitCopyStackToMemSecondary(g, local.offset, 0, local.size)
	} else {
		renvoEmitCopyMemSecondaryToStack(g, local.offset, local.size)
	}
	renvoAsmMarkLabel(&g.asm, skip)
}

func renvoAllocateCapturedCell(g *renvoLinearGen, captureOff int, size int) {
	renvoNonNil(g)
	renvoAsmStoreStackImm(&g.asm, captureOff, size)
	renvoEmitPersistentAllocToPrimary(g, captureOff)
	renvoAsmStorePrimaryStack(&g.asm, captureOff)
}

func renvoRebindCapturedLocal(g *renvoLinearGen, localIndex int) {
	renvoNonNil(g)
	if localIndex < 0 || localIndex >= g.localCount || g.locals[localIndex].captureOff <= 0 {
		return
	}
	local := &g.locals[localIndex]
	renvoAllocateCapturedCell(g, local.captureOff, local.size)
	renvoMoveCapturedLocal(g, localIndex, true)
}

func renvoMoveCapturedLocals(g *renvoLinearGen, toCell bool) {
	renvoNonNil(g)
	if !g.hasCapturedLocals {
		return
	}
	for i := 0; i < g.localCount; i++ {
		renvoMoveCapturedLocal(g, i, toCell)
	}
}

func renvoZeroLocalAtOffset(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	size := 8
	typ := renvoTypeInt
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		if local.offset == offset {
			size = local.size
			typ = local.typ
		}
	}
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypeSlice {
		renvoInitEmptySliceStack(g, offset)
		return
	}
	renvoZeroLocalStorage(g, offset, size)
	if t.kind == renvoTypeStruct {
		renvoInitStructSliceFields(g, typ, offset)
	}
}
func renvoInitEmptySliceStack(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmStoreStackImm(a, offset, 0)
	renvoAsmStorePrimaryStack(a, offset-8)
	renvoAsmStorePrimaryStack(a, offset-16)
}
func renvoInitStructSliceFields(g *renvoLinearGen, typ int, offset int) {
	renvoNonNil(g)
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind != renvoTypeStruct {
		return
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		fieldOffset := offset - field.offset
		fieldType := renvoResolveType(g.meta, field.typ)
		renvoNonNil(fieldType)
		if fieldType.kind == renvoTypeSlice {
			renvoInitEmptySliceStack(g, fieldOffset)
		} else if fieldType.kind == renvoTypeStruct {
			renvoInitStructSliceFields(g, field.typ, fieldOffset)
		}
	}
}
func renvoEmitCopyReturnedStructSliceFields(g *renvoLinearGen, typ int, srcOffset int, destOffset int) bool {
	renvoNonNil(g)
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind != renvoTypeStruct {
		return true
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		fieldType := renvoResolveType(g.meta, field.typ)
		renvoNonNil(fieldType)
		fieldSrcOffset := srcOffset - field.offset
		fieldDestOffset := destOffset + field.offset
		if fieldType.kind == renvoTypeSlice {
			renvoAsmLoadPrimarySecondaryStack(&g.asm, fieldSrcOffset, fieldSrcOffset-8)
			renvoAsmLoadTertiaryStack(&g.asm, fieldSrcOffset-16)
			if !renvoEmitCopySliceRegsToArena(g, field.typ) {
				return false
			}
			renvoAsmPushSliceRegs(&g.asm)
			renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
			renvoAsmPopStoreSliceMemSecondary(&g.asm, fieldDestOffset)
		} else if fieldType.kind == renvoTypeStruct {
			if !renvoEmitCopyReturnedStructSliceFields(g, field.typ, fieldSrcOffset, fieldDestOffset) {
				return false
			}
		}
	}
	return true
}
func renvoFuncInfoFromCall(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	e := &ep.exprs[idx]
	if e.right > 0 {
		return e.right - 1
	}
	nameStart := e.nameStart
	nameEnd := e.nameEnd
	wantMethod := false
	wantReceiverType := 0
	if e.kind == renvoExprSelector {
		wantMethod = true
		wantReceiverType = renvoInferParsedExprType(g, ep, e.left)
	} else if e.kind != renvoExprIdent {
		return -1
	}
	hash := renvoHashRange(p.src, nameStart, nameEnd)
	i := int(renvo_runtime_UnsafeInt32At(meta.funcBuckets, hash%len(meta.funcBuckets)))
	for i >= 0 {
		f := meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && renvoBytesEqualRange(p.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			if !wantMethod || renvoMethodReceiverTypeMatches(g.meta, wantReceiverType, f.receiverType) {
				e.right = i + 1
				return i
			}
		}
		i = int(renvo_runtime_UnsafeInt32At(meta.funcNext, i))
	}
	return -1
}

func renvoIsInterfaceMethodCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	call := &ep.exprs[idx]
	if call.kind != renvoExprCall {
		return false
	}
	selector := &ep.exprs[call.left]
	return selector.kind == renvoExprSelector && renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, selector.left)).kind == renvoTypeInterface
}

func renvoInterfaceMethodCallResultType(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	if !renvoIsInterfaceMethodCall(g, ep, idx) {
		return 0
	}
	selector := &ep.exprs[ep.exprs[idx].left]
	methodType := renvoInterfaceMethodType(g, renvoInferParsedExprType(g, ep, selector.left), selector)
	if methodType != 0 {
		return renvoResolveType(g.meta, methodType).elem
	}
	if renvoBytesEqualText(g.prog.src, selector.nameStart, selector.nameEnd, "Error") {
		return renvoTypeString
	}
	return 0
}

func renvoInterfaceMethodType(g *renvoLinearGen, interfaceType int, selector *renvoExpr) int {
	renvoNonNil(g, selector)
	iface := renvoResolveType(g.meta, interfaceType)
	renvoNonNil(iface)
	for required := iface.first; required < iface.count; {
		end := renvoStatementLineEnd(g.prog, required, iface.count)
		if renvoTokCharIs(g.prog, required+1, '(') {
			if renvoBytesEqualRange(g.prog.src, int(renvoTokStart(g.prog, required)), int(renvoTokEnd(g.prog, required)), selector.nameStart, selector.nameEnd) {
				var parsed renvoTypeResult
				renvoParseFuncSignatureInto(g.meta, g.prog, required+1, end, &parsed)
				return parsed.typ
			}
		} else {
			embedded := renvoParseType(g.meta, g.prog, required, end)
			if embedded.typ != 0 {
				if result := renvoInterfaceMethodType(g, embedded.typ, selector); result != 0 {
					return result
				}
			}
		}
		required = end
	}
	return 0
}

func renvoEmitInterfaceMethodCall(g *renvoLinearGen, ep *renvoExprParse, idx int, resultOffset int, resultType int) bool {
	renvoNonNil(g, ep)
	call := &ep.exprs[idx]
	selector := &ep.exprs[call.left]
	receiverType := renvoInferParsedExprType(g, ep, selector.left)
	receiverOffset := renvoAddUnnamedLocal(g, receiverType)
	if !renvoEmitInterfaceAssignToLocal(g, ep, selector.left, receiverOffset) {
		return false
	}
	usesHiddenResult := resultOffset > 0
	if usesHiddenResult {
		renvoZeroLocalAtOffset(g, resultOffset)
	}
	doneLabel := renvoAsmNewLabel(&g.asm)
	matched := false
	for fnIndex := 0; fnIndex < len(g.meta.funcs); fnIndex++ {
		fn := &g.meta.funcs[fnIndex]
		if fn.paramCount != call.argCount+1 || !renvoInterfaceMethodNamed(g, fn, selector) {
			continue
		}
		if resultType != 0 && !renvoTypesEquivalent(g.meta, resultType, fn.resultType) {
			continue
		}
		if renvoTypeUsesHiddenResult(g.meta, fn.resultType) != usesHiddenResult {
			continue
		}
		matched = true
		nextLabel := renvoAsmNewLabel(&g.asm)
		receiverSize := renvoTypeSize(g.meta, fn.receiverType)
		renvoEmitInterfaceReceiverMatch(g, receiverOffset, fn.receiverType, nextLabel)

		wordCount := 0
		receiverResolved := renvoResolveType(g.meta, fn.receiverType)
		if renvoPreparedBackendActive != 0 && renvoStructArgByReference(g, receiverResolved.kind) {
			renvoAsmPushStackWord(&g.asm, receiverOffset)
			wordCount++
		} else if renvoInterfaceValueStoredIndirect(g.meta, fn.receiverType) {
			receiverValue := renvoAddUnnamedLocal(g, fn.receiverType)
			renvoAsmLoadSecondaryStack(&g.asm, receiverOffset)
			renvoEmitCopyMemSecondaryToStack(g, receiverValue, receiverSize)
			wordCount += renvoEmitTypedLocalArgReverse(g, receiverValue, fn.receiverType)
		} else {
			wordCount += renvoEmitTypedLocalArgReverse(g, receiverOffset, fn.receiverType)
		}
		for i := 0; i < call.argCount; i++ {
			words := renvoEmitCallParamArgReverse(g, ep, renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+i), fn.firstParam+i+1)
			if words < 0 {
				return false
			}
			wordCount += words
		}
		if usesHiddenResult {
			renvoAsmAddressPrimaryStack(&g.asm, resultOffset)
			renvoAsmPushPrimary(&g.asm)
			wordCount++
		}
		renvoEmitCallWithWordCount(g, fnIndex, wordCount)
		renvoAsmJmpMarkLabel(&g.asm, doneLabel, nextLabel)
	}
	if !matched {
		// A closed program can contain a polymorphic helper whose interface method
		// has no linked concrete implementation. The helper is still valid when
		// that path is unreachable (for example, a formatter's error case in a
		// program that never constructs an error). Keep it compilable and provide
		// a useful failure if an invalid interface value reaches the call.
		renvoAsmMarkLabel(&g.asm, doneLabel)
		renvoEmitStaticWrite(g, "interface method unavailable\n", 2)
		renvoAsmPrimaryImm(&g.asm, 2)
		return renvoEmitExitStatus(g)
	}
	if !usesHiddenResult {
		renvoAsmPrimaryImm(&g.asm, 0)
	}
	renvoAsmMarkLabel(&g.asm, doneLabel)
	return true
}

func renvoInterfaceMethodNamed(g *renvoLinearGen, fn *renvoFuncInfo, selector *renvoExpr) bool {
	renvoNonNil(g, fn, selector)
	return fn.receiverType != 0 && renvoBytesEqualRange(g.prog.src, fn.nameStart, fn.nameEnd, selector.nameStart, selector.nameEnd)
}

func renvoEmitInterfaceReceiverMatch(g *renvoLinearGen, receiverOffset int, receiverType int, nextLabel int) {
	renvoNonNil(g)
	if renvoResolveType(g.meta, receiverType).kind == renvoTypePointer {
		renvoAsmJcmpStackImm(&g.asm, receiverOffset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, receiverType), nextLabel, 0x95)
		return
	}
	pointerType := renvoAddPointerType(g.meta, receiverType, renvoPointerSpaceData)
	matchedLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmJcmpStackImm(&g.asm, receiverOffset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, receiverType), matchedLabel, 0x94)
	renvoAsmJcmpStackImm(&g.asm, receiverOffset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, pointerType), nextLabel, 0x95)
	if renvoTypeSize(g.meta, receiverType) <= renvoBackendValueSlotSize {
		renvoAsmLoadSecondaryStack(&g.asm, receiverOffset)
		renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, 0)
		renvoAsmStorePrimaryStack(&g.asm, receiverOffset)
	}
	renvoAsmMarkLabel(&g.asm, matchedLabel)
}

func renvoMethodReceiverTypeMatches(meta *renvoMeta, actual int, declared int) bool {
	renvoNonNil(meta)
	if actual == declared {
		return true
	}
	actual = renvoCanonicalMethodReceiverType(meta, actual)
	declared = renvoCanonicalMethodReceiverType(meta, declared)
	if actual == declared {
		return true
	}
	t := renvoResolveType(meta, actual)
	renvoNonNil(t)
	if t.kind != renvoTypeStruct || t.count == 0 {
		return false
	}
	field := meta.fields[t.first]
	return field.embedded && field.offset == 0 && renvoCanonicalMethodReceiverType(meta, field.typ) == declared
}

func renvoCanonicalMethodReceiverType(meta *renvoMeta, typ int) int {
	renvoNonNil(meta)
	for typ > 0 && typ < len(meta.types) {
		t := meta.types[typ]
		if t.kind == renvoTypePointer {
			typ = t.elem
			continue
		}
		if t.kind == renvoTypeNamed && t.first == renvoNamedTypeAlias && t.elem > 0 && t.elem < len(meta.types) {
			typ = t.elem
			continue
		}
		if t.kind == renvoTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
			resolved := renvoFindResolvedNamedTypeIndex(meta, typ)
			if resolved > 0 && resolved < len(meta.types) {
				typ = resolved
				continue
			}
		}
		break
	}
	return typ
}

func renvoEmitCompactCValueHelper(g *renvoLinearGen, fnInfoIndex int) bool {
	if renvoFixedTarget != 0 || !g.c.code16 || !g.c.objectFile || fnInfoIndex < 0 || fnInfoIndex >= len(g.meta.funcs) {
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
	signedValue := result.kind == renvoTypeInt8 || result.kind == renvoTypeInt16
	return renvoEmitCompactCValueHelperBody(&g.asm, g.funcLabels[fnInfoIndex], size, signedValue, postDec)
}

func renvoEmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	if renvoEmitCompactCValueHelper(g, fnInfoIndex) {
		return true
	}
	override := renvoEmitFunctionOverride(a, metaFn.declIndex, g.funcLabels[fnInfoIndex])
	if override != 0 {
		return override > 0
	}
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
	localCapacity := 16
	if metaFn.bodyEnd-metaFn.bodyStart >= 512 {
		localCapacity = 32
	}
	g.locals = make([]renvoLocalInfo, localCapacity)
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
	g.deferHeadOffset = 0
	g.deferReturnLabel = 0
	g.deferResultOffset = 0
	g.deferSites = nil
	g.emittingDefers = false
	g.suppressPanicCheck = false
	g.lastRangeReturns = false
	framePatch := renvoFunctionFrameStart(g, g.funcLabels[fnInfoIndex])
	if !renvoEmitFunctionBody(g, fnInfoIndex, renvoZeroVoidReturn(g.c)) {
		return false
	}
	renvoFunctionFrameFinish(g, framePatch)
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

// Function body semantics are shared across frame encodings. Targets choose
// only whether a void fallthrough needs a deterministic primary register.
func renvoEmitFunctionBody(g *renvoLinearGen, fnInfoIndex int, zeroVoidResult bool) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoStoreHiddenResult(g, g.returnStruct)
	}
	renvoBindFunctionParams(g, fnInfoIndex)
	if !renvoBindClosureCaptures(g, fnInfoIndex) ||
		!renvoBindNamedResults(g, fnInfoIndex) ||
		!renvoPrepareFunctionControl(g) ||
		!renvoEmitLinearRange(g, metaFn.bodyStart, metaFn.bodyEnd) {
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
		if zeroVoidResult || metaFn.resultType != 0 {
			renvoAsmPrimaryImm(a, 0)
		}
		renvoAsmLeave(a)
		renvoAsmRet(a)
	}
	return true
}

func renvoEmitScalarFunctionScratch(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	g.checkedPointerLocals = 0
	g.invalidatedPointerLocals = 0
	g.hasCapturedLocals = false
	g.addressNamesReady = false
	g.addressNameTokens = nil
	persistentCapacity := renvoLinearPersistentCapacity(g)
	typeCount := len(g.meta.types)
	typeIndexVersion := g.meta.typeIndexVersion
	fieldCount := len(g.meta.fields)
	captureCount := len(g.meta.captures)
	mark := renvo_runtime_ArenaMark()
	ok := renvoEmitScalarFunction(g, fnInfoIndex)
	if len(g.meta.captures) == captureCount && g.meta.runtimeTypeCount <= typeCount {
		renvoTruncTypes(&g.meta.types, typeCount)
		renvoTruncFields(&g.meta.fields, fieldCount)
		// Unnamed scratch types never enter the name index. Rebuild only when
		// emission changed that index or renamed a type.
		if g.meta.typeIndexVersion != typeIndexVersion {
			renvoRebuildNamedTypeIndex(g.meta)
		}
	}
	if persistentCapacity == renvoLinearPersistentCapacity(g) {
		renvo_runtime_ArenaReset(mark)
	}
	return ok
}

func renvoLinearPersistentCapacity(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	m := g.meta
	objectStringCapacity := 0
	if a.objectStrings != nil {
		objectStringCapacity = cap(a.objectStrings.refs)
	}
	// The remaining slices are either fixed-size or completely populated before
	// function emission begins. Only slices which can grow while a function is
	// emitted need to prevent the scratch arena from being rewound.
	capacity := cap(a.code) + cap(a.labelPos) + cap(a.relocs) + cap(a.absRelocs) + cap(a.symbols) + cap(a.symbolName) + cap(a.staticImports) + cap(a.darwinImports) + cap(a.darwinImportLabels) + cap(a.darwinImportUsed) + cap(a.data) + cap(a.wasmLocalSlots) + objectStringCapacity + cap(g.breakLabels) + cap(g.continueLabels) + cap(m.types) + cap(m.fields) + cap(m.captures)
	// Unused target-owned buffers have zero capacity. Account for all growth,
	// rather than using target identity to decide whether a reset is safe.
	capacity += cap(a.openbsdSyscalls)
	return capacity
}

func renvoEmitFloat64BitsPrimary(a *renvoAsm, bits uint64) {
	renvoNonNil(a)
	low := int(uint32(bits))
	high := int(uint32(bits >> 32))
	if a.c.renvoNativeIntSize == 8 {
		renvoAsmPrimaryImm64(a, low, high)
		return
	}
	// Wide 32-bit float expressions are stored through the paired-word path.
	// Loading the low word here preserves the existing scalar contract until
	// that path consumes the high half explicitly.
	renvoAsmPrimaryImm(a, low)
}
func renvoAsmLoadPrimaryBssSize(a *renvoAsm, bssOff int, size int) {
	renvoNonNil(a)
	if size >= a.c.renvoNativeIntSize {
		renvoAsmLoadPrimaryBss(a, bssOff)
		return
	}
	renvoAsmPrimaryBssAddr(a, bssOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
}
func renvoAsmStorePrimaryBssSize(a *renvoAsm, bssOff int, size int) {
	renvoNonNil(a)
	if size >= a.c.renvoNativeIntSize {
		renvoAsmStorePrimaryBss(a, bssOff)
		return
	}
	renvoAsmPushPrimary(a)
	renvoAsmPrimaryBssAddr(a, bssOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, size)
}

func renvoEmitTypedPrimaryTertiaryOp(g *renvoLinearGen, tok int, kind int) bool {
	renvoNonNil(g)
	float := renvoTypeKindIsFloat(kind)
	if float {
		if renvoUsesScaledFloat(&g.asm) {
			if renvoTok2Is(g.prog, tok, '*', '=') {
				if !renvoEmitPrimaryTertiaryOp(g, tok) {
					return false
				}
				renvoAsmSarPrimaryImm(&g.asm, 2)
				return true
			}
			if renvoTok2Is(g.prog, tok, '/', '=') {
				renvoAsmShlTertiaryImm(&g.asm, 2)
			}
			return renvoEmitPrimaryTertiaryOp(g, tok)
		}
		return renvoEmitIEEEFloatPrimaryTertiaryOp(g, tok, kind)
	}
	if !float && (kind == renvoTypeByte || kind >= renvoTypeUint16 && kind <= renvoTypeUint64) && (renvoTokStartsWith(g.prog, tok, '/') || renvoTokStartsWith(g.prog, tok, '%')) {
		return renvoEmitUnsignedPrimaryTertiaryOp(g, tok, kind)
	}
	// Compound right shifts retain the destination's unsigned type. The
	// untyped machine operation defaults to arithmetic shift on a full word.
	if (kind == renvoTypeByte || kind >= renvoTypeUint16 && kind <= renvoTypeUint64) && renvoTokStartsWith(g.prog, tok, '>') {
		return renvoEmitTargetUnsignedShiftRight(g, tok)
	}
	return renvoEmitPrimaryTertiaryOp(g, tok)
}

func renvoEmitUnsignedPrimaryTertiaryOp(g *renvoLinearGen, tok int, kind int) bool {
	if kind == renvoTypeByte || kind == renvoTypeUint16 {
		return renvoEmitPrimaryTertiaryOp(g, tok)
	}
	mod := renvoTokStartsWith(g.prog, tok, '%')
	renvoEmitRuntimeNonNilPrimary(g)
	return renvoEmitUnsignedDividePrimaryTertiary(g, mod)
}

func renvoEmitPrimaryTertiaryOp(g *renvoLinearGen, tok int) bool {
	renvoNonNil(g)
	divide := renvoTokCharIs(g.prog, tok, '/')
	mod := renvoTokCharIs(g.prog, tok, '%')
	if (divide || mod) && !g.meta.panicEnabled {
		renvoEmitUncheckedSignedDivision(g, mod)
		return true
	}
	done := -1
	if divide || mod {
		renvoEmitRuntimeNonNilPrimary(g)
		done = renvoEmitTargetSignedDivisionGuard(g, mod)
	}
	ok := renvoEmitTargetPrimaryTertiaryOp(g, tok)
	if done >= 0 {
		renvoAsmMarkLabel(&g.asm, done)
	}
	return ok
}

func renvoEnsureSignedDivisionHelper(g *renvoLinearGen, mod bool) int {
	renvoNonNil(g)
	a := &g.asm
	slot := &g.divideCheckLabel
	if mod {
		slot = &g.remainderCheckLabel
	}
	renvoNonNil(slot)
	if *slot > 0 {
		return *slot - 1
	}
	label := renvoAsmNewLabel(a)
	*slot = label + 1
	if renvoRTGStructuredFunctions != 0 {
		argument := 0
		if mod {
			argument = 1
		}
		renvoQueueStructuredHelper(g, renvoStructuredHelperSignedDivide, argument, label)
		return label
	}
	after := renvoAsmNewLabel(a)
	helperEnd := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, after, label)
	renvoEmitSignedDivisionHelperBody(g, mod)
	renvoAsmMarkLabel(a, helperEnd)
	renvoAsmMarkLabel(a, after)
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		name := "__renvo_signed_divide"
		if mod {
			name = "__renvo_signed_remainder"
		}
		renvoAsmAddLocalObjectFuncSymbolText(a, name, label, helperEnd)
	}
	return label
}

func renvoEmitSignedDivisionHelperBody(g *renvoLinearGen, mod bool) {
	a := &g.asm
	nonzero := renvoAsmNewLabel(a)
	renvoAsmJnzPrimary(a, nonzero)
	renvoEmitUncaughtFaultTransfer(g, false)
	renvoAsmMarkLabel(a, nonzero)
	done := renvoEmitTargetSignedDivisionGuard(g, mod)
	renvoAsmDivLeftTertiaryRightPrimary(a, mod)
	if done >= 0 {
		renvoAsmMarkLabel(a, done)
	}
	renvoAsmRet(a)
}

func renvoEmitSignedDivisionOverflowGuard(g *renvoLinearGen, mod bool) int {
	renvoNonNil(g)
	a := &g.asm
	normal := renvoAsmNewLabel(a)
	restoreDivisor := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	renvoAsmCmpPrimaryImm8(a, -1)
	renvoAsmJnzLabel(a, normal)
	renvoAsmPushTertiary(a)
	renvoAsmPrimaryImm(a, -1)
	renvoAsmShlPrimaryImm(a, g.c.renvoNativeIntSize*8-1)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJzPrimary(a, restoreDivisor)
	if mod {
		renvoAsmPrimaryImm(a, 0)
	} else {
		renvoAsmCopyTertiaryToPrimary(a)
	}
	renvoAsmJmpMarkLabel(a, done, restoreDivisor)
	renvoAsmPrimaryImm(a, -1)
	renvoAsmMarkLabel(a, normal)
	return done
}

func renvoEmitWideCompareOperand(g *renvoLinearGen, ep *renvoExprParse, idx int, floatKind int) bool {
	if renvoTypeKindIsFloat(floatKind) {
		return renvoEmitScalarExprForKind(g, ep, idx, floatKind)
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoNormalizeNativeExprPrimary(g, ep, idx)
	return true
}

func renvoEmitWideFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	if !renvoUsesStackIEEEFloat(&g.asm) {
		return false
	}
	e := &ep.exprs[idx]
	kind := renvoBinaryFloatKind(g, ep, e)
	leftType := renvoTypeFloat64
	if kind == renvoTypeFloat32 {
		leftType = renvoBuiltinTypeFloat32
	}
	left := renvoAddUnnamedLocal(g, leftType)
	right := renvoAddUnnamedLocal(g, leftType)
	if !renvoEmitWideFloatBinaryOperands(g, ep, e, kind, left, right) {
		return false
	}
	c0, c1, comparison := renvoFloatComparisonChars(g.prog, e.tok)
	if comparison {
		return renvoEmit32IEEECompareStack(g, left, right, kind, c0, c1)
	}
	result := renvoAddUnnamedLocal(g, leftType)
	size := 8
	if kind == renvoTypeFloat32 {
		size = 4
	}
	if !renvo32IEEEBinaryStack(g, result, left, right, c0, size) {
		return false
	}
	if kind == renvoTypeFloat32 {
		renvoAsmLoadPrimaryStack(&g.asm, result)
		return true
	}
	return false
}

func renvoEmitWideFloatBinaryOperands(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, kind int, left int, right int) bool {
	if kind == renvoTypeFloat64 && renvoUsesDirectFloat64Operands(&g.asm) {
		return renvoEmitStackFloat64ExprToLocal(g, ep, e.left, left) && renvoEmitStackFloat64ExprToLocal(g, ep, e.right, right)
	}
	return renvoEmitTypedAssign(g, ep, e.left, left) && renvoEmitTypedAssign(g, ep, e.right, right)
}

// renvoEmitNonWordBinaryExpr handles language-level binary operations before
// scalar register lowering. A negative result leaves a word operation to the
// caller; zero reports a failed emission and one reports a completed expression.
func renvoEmitNonWordBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoBinaryUsesFloat(g, ep, e) {
		if renvoEmitFloatBinaryExpr(g, ep, idx) {
			return 1
		}
		return 0
	}
	if renvoStringOrderingExpr(g, ep, e) {
		if renvoEmitStringOrdering(g, ep, e) {
			return 1
		}
		return 0
	}
	if renvoTok2Is(p, e.tok, '=', '=') || renvoTok2Is(p, e.tok, '!', '=') {
		leftType := renvoInferParsedExprType(g, ep, e.left)
		leftResolved := renvoResolveType(g.meta, leftType)
		if leftResolved.kind == renvoTypeArray || leftResolved.kind == renvoTypeStruct || renvoTypeKindIsComplex(leftResolved.kind) {
			if renvoEmitCompositeCompare(g, ep, e, leftType) {
				return 1
			}
			return 0
		}
		rightType := renvoInferParsedExprType(g, ep, e.right)
		if renvoTypeIsString(g.meta, leftType) || renvoTypeIsString(g.meta, rightType) {
			if renvoEmitStringCompare(g, ep, e.left, e.right, renvoTok2Is(p, e.tok, '!', '=')) {
				return 1
			}
			return 0
		}
	}
	if renvoTok2Is(p, e.tok, '&', '&') || renvoTok2Is(p, e.tok, '|', '|') {
		falseLabel := renvoAsmNewLabel(a)
		endLabel := renvoAsmNewLabel(a)
		if !renvoEmitJumpIfFalse(g, ep, idx, falseLabel) {
			return 0
		}
		renvoAsmPrimaryImm(a, 1)
		renvoAsmJmpMarkLabel(a, endLabel, falseLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmMarkLabel(a, endLabel)
		return 1
	}
	return -1
}

// renvoEmitWordBinaryOperands preserves left-to-right evaluation and leaves
// the right operand in primary and the left operand in tertiary.
func renvoEmitWordBinaryOperands(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	rightExpr := &ep.exprs[e.right]
	rightKind := rightExpr.kind
	rightTok := rightExpr.tok
	if !renvoEmitIntExpr(g, ep, e.left) {
		return false
	}
	renvoAsmPushPrimary(a)
	if rightKind == renvoExprInt {
		renvoAsmLoadPrimaryIntToken(a, p, rightTok)
	} else if rightKind == renvoExprChar {
		renvoAsmPrimaryImm(a, renvoParseCharToken(p, rightTok))
	} else if rightKind == renvoExprBool {
		renvoAsmPrimaryImm(a, renvoBoolTokenValue(p, rightTok))
	} else if !renvoEmitIntExpr(g, ep, e.right) {
		return false
	}
	renvoAsmPopTertiary(a)
	return true
}

// renvoEmitScalarSelectorExpr resolves field semantics once; definitions own
// the field load width and frame-address emission for the value representation.
func renvoEmitScalarSelectorExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	baseType := renvoInferParsedExprType(g, ep, e.left)
	nativeABI := renvoTypeUsesNativeABI(g.meta, baseType)
	fieldType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	fieldSize := renvoNativeScalarStorageSize(g.c.renvoNativeIntSize, fieldType.kind)
	base := &ep.exprs[e.left]
	if renvoEmitDirectSelectorWords(g, ep, idx, 0, -1, fieldSize) {
		if nativeABI {
			renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
		}
		return true
	}
	if base.kind == renvoExprCall {
		baseResolved := renvoResolveType(g.meta, baseType)
		if baseResolved.kind == renvoTypePointer {
			if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
				return false
			}
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, fieldSize)
			if nativeABI {
				renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
			}
			return true
		}
		if !renvoTypeIsStruct(g.meta, baseType) {
			return false
		}
		fieldOffset := renvoStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
		if fieldOffset < 0 {
			return false
		}
		offset := renvoAddTypedLocal(g, 0, 0, baseType)
		if !renvoEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
			return false
		}
		renvoAsmLoadFrameFieldValue(a, offset-fieldOffset, fieldSize, nativeABI)
	} else if base.kind == renvoExprIndex {
		return renvoEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
	} else if offset, ok := renvoLocalStructSelectorOffset(g, ep, idx); ok {
		renvoAsmLoadFrameFieldValue(a, offset, fieldSize, nativeABI)
	} else {
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadIndirectFieldValue(a, fieldSize, nativeABI)
	}
	if nativeABI {
		renvoAsmNormalizePrimaryForKind(a, fieldType.kind)
	}
	return true
}

// renvoEmitUnaryExpr keeps address resolution, capture handling, and pointer
// invalidation in shared lowering; definitions select address emission support.
func renvoEmitUnaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if !renvoTokCharIs(p, e.tok, '&') {
		return renvoEmitUnaryValueExpr(g, ep, idx)
	}
	inner := &ep.exprs[e.left]
	if inner.kind == renvoExprUnary && renvoTokCharIs(p, inner.tok, '*') {
		if !renvoEmitIntExpr(g, ep, inner.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		return true
	}
	if inner.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, inner.nameStart, inner.nameEnd)
		if localIndex >= 0 {
			renvoInvalidateCheckedPointerLocal(g, localIndex)
			if g.locals[localIndex].captureOff > 0 {
				renvoAsmLoadPrimaryStack(a, g.locals[localIndex].captureOff)
			} else {
				renvoAsmAddressTakenLocal(a, g.locals[localIndex].offset)
			}
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
		if globalOffset >= 0 {
			renvoAsmPrimaryBssAddr(a, globalOffset)
			return true
		}
		if renvoCanTakeNamedFunctionAddress(g) {
			fnIndex := renvoFindMetaFunction(g.meta, inner.nameStart, inner.nameEnd)
			if fnIndex >= 0 && renvoIsHostedObject(g.c) {
				return renvoEmitObjectFunctionAddress(g, fnIndex)
			}
		}
		return false
	}
	if inner.kind == renvoExprSelector || inner.kind == renvoExprIndex {
		return renvoEmitAddressPrimary(g, ep, e.left)
	}
	return false
}

// renvoEmitWordCallExpr resolves conversions and builtins before ordinary calls.
// Target-specific C call preambles have already had an opportunity to handle
// the expression; function-word representation support is definition-owned.
func renvoEmitWordCallExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	e := &ep.exprs[idx]
	if intrinsic := renvoEmitWordIntrinsicCall(g, ep, idx); intrinsic >= 0 {
		return intrinsic != 0
	}
	callee := renvoExprIdentCode(p, ep, e.left)
	if e.argCount == 1 {
		firstArgIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		conversion := renvoResolveType(g.meta, conversionType)
		if renvoIsSliceArrayConversion(g, ep, firstArgIndex, conversionType) {
			return renvoEmitSliceArrayConversion(g, ep, firstArgIndex, conversionType, 0)
		}
		if renvoTypeKindIsScalarValue(conversion.kind) || conversion.kind == renvoTypePointer ||
			g.c.objectFile && conversion.kind == renvoTypeFunc && renvoSupportsFunctionWordConversion(g) {
			return renvoEmitScalarExprForKind(g, ep, firstArgIndex, conversion.kind)
		}
		if callee == renvoIdentCap || callee == renvoIdentLen {
			return renvoEmitLengthCapacityCall(g, ep, idx)
		}
	}
	if callee >= renvoIdentOpen && callee <= renvoIdentChmod {
		return renvoEmitTargetRuntime(g, ep, idx, callee)
	}
	if callee == renvoIdentCopy {
		return renvoEmitBuiltinCopy(g, ep, idx)
	}
	return renvoEmitUserCall(g, ep, idx)
}

func renvoEmitWordBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	e := &ep.exprs[idx]
	opStart := int(renvoTokStart(p, e.tok))
	opLen := int(renvoTokEnd(p, e.tok)) - opStart
	op0 := renvo_runtime_UnsafeByteAt(p.src, opStart)
	op1 := byte(0)
	if opLen == 2 {
		op1 = renvo_runtime_UnsafeByteAt(p.src, opStart+1)
	}
	if result := renvoEmitNonWordBinaryExpr(g, ep, idx); result >= 0 {
		return result != 0
	}
	if optimized := renvoEmitOptimizedNativeBinaryExpr(g, ep, idx); optimized >= 0 {
		return optimized != 0
	}
	if !renvoEmitWordBinaryOperands(g, ep, idx) {
		return false
	}
	resultKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind
	if (op0 == '<' || op0 == '>') && !(opLen == 2 && op1 == op0) &&
		(renvoExprHasUnsignedIntType(g, ep, e.left) || renvoExprHasUnsignedIntType(g, ep, e.right)) &&
		renvoEmitUnsignedWordOrderResult(g, op0, op1, opLen, resultKind) {
		return true
	}
	unsigned := resultKind == renvoTypeByte || resultKind >= renvoTypeUint16 && resultKind <= renvoTypeUint64
	if opLen == 2 && (op0 == '<' && op1 == '<' || op0 == '>' && op1 == '>') {
		if !renvoEmitBoundedWordShift(g, e.tok, op0 == '>', renvoExprHasUnsignedIntType(g, ep, e.left), unsigned) {
			return false
		}
	} else if unsigned && (op0 == '/' || op0 == '%') {
		if !renvoEmitUnsignedPrimaryTertiaryOp(g, e.tok, resultKind) {
			return false
		}
	} else if !renvoEmitPrimaryTertiaryOp(g, e.tok) {
		return false
	}
	renvoAsmNormalizePrimaryForKind(&g.asm, resultKind)
	return true
}

func renvoEmitCNativeIntCall(g *renvoLinearGen, ep *renvoExprParse, idx int, e *renvoExpr) int {
	p := g.prog
	meta := g.meta
	a := &g.asm
	if e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprSelector {
		selector := &ep.exprs[e.left]
		if offset, ok := renvoCFieldAccessorOffset(p.src, selector.nameStart, selector.nameEnd); ok {
			if e.argCount != 0 {
				return 0
			}
			receiverType := renvoInferParsedExprType(g, ep, selector.left)
			receiver := renvoResolveType(meta, receiverType)
			renvoNonNil(receiver)
			if receiver.kind == renvoTypePointer {
				if !renvoEmitIntExpr(g, ep, selector.left) {
					return 0
				}
				fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
				if fnIndex >= 0 {
					depth := renvoPointerDereferenceDistance(meta, receiverType, meta.funcs[fnIndex].receiverType)
					for i := 0; i < depth; i++ {
						renvoEmitRuntimeNonNilPrimary(g)
						renvoAsmCopyPrimaryToSecondary(a)
						renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
					}
				}
			} else if !renvoEmitAddressPrimary(g, ep, selector.left) {
				return 0
			}
			if offset != 0 {
				renvoAsmPushImm(a, offset)
				renvoAsmPopTertiary(a)
				renvoAsmAddPrimaryTertiary(a)
			}
			return 1
		}
	}
	if e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprIdent {
		calleeExpr := &ep.exprs[e.left]
		if renvoFixedTarget == 0 {
			inlined := renvoEmitCInlineReturn(g, ep, idx, e, calleeExpr, -1, false)
			if inlined >= 0 {
				return inlined
			}
			update := renvoEmitCUpdateIntrinsic(g, ep, idx, e, calleeExpr, false)
			if update >= 0 {
				return update
			}
		}
		if offset, ok := renvoCFieldAccessorOffset(p.src, calleeExpr.nameStart, calleeExpr.nameEnd); ok {
			if e.argCount != 1 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
				return 0
			}
			if offset != 0 {
				renvoAsmPrimaryAddressOffset(a, offset)
			}
			return 1
		}
		if renvoBytesPrefixText(p.src, calleeExpr.nameStart, calleeExpr.nameEnd, "__c_pointer_diff_") {
			return renvoBoolInt(renvoEmitCPointerDifference(g, ep, e))
		}
		if renvoBytesPrefixText(p.src, calleeExpr.nameStart, calleeExpr.nameEnd, "__c_pointer_step_inc_") ||
			renvoBytesPrefixText(p.src, calleeExpr.nameStart, calleeExpr.nameEnd, "__c_pointer_index_") ||
			renvoBytesPrefixText(p.src, calleeExpr.nameStart, calleeExpr.nameEnd, "__c_array_index_") {
			return renvoBoolInt(renvoEmitCPointerStep(g, ep, idx, e, false))
		}
		if renvoBytesPrefixText(p.src, calleeExpr.nameStart, calleeExpr.nameEnd, "__c_pointer_step_dec_") {
			return renvoBoolInt(renvoEmitCPointerStep(g, ep, idx, e, true))
		}
	}
	return -1
}

func renvoEmitCPointerStep(g *renvoLinearGen, ep *renvoExprParse, callIndex int, call *renvoExpr, decrement bool) bool {
	if call.argCount != 2 {
		return false
	}
	pointerArg := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg)
	countArg := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+1)
	resultType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, callIndex))
	if resultType.kind != renvoTypePointer {
		return false
	}
	if !renvoEmitIntExpr(g, ep, pointerArg) {
		return false
	}
	elementSize := renvoTypeSize(g.meta, resultType.elem)
	constant := renvoEvalConstExpr(g, ep, countArg)
	if renvoFixedTarget == 0 && constant.ok {
		delta := constant.value * elementSize
		if decrement {
			delta = -delta
		}
		if renvoEmitConstantPointerStep(g, delta) {
			return true
		}
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, countArg) {
		return false
	}
	if elementSize > 1 {
		renvoAsmPushImm(&g.asm, elementSize)
		renvoAsmPopTertiary(&g.asm)
		renvoAsmMulPrimaryTertiary(&g.asm)
	}
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	if decrement {
		renvoAsmSubPrimaryTertiary(&g.asm)
	} else {
		renvoAsmAddPrimaryTertiary(&g.asm)
	}
	return true
}

func renvoEmitCPointerDifference(g *renvoLinearGen, ep *renvoExprParse, call *renvoExpr) bool {
	if call.argCount != 2 {
		return false
	}
	left := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg)
	right := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+1)
	leftType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, left))
	if leftType.kind != renvoTypePointer || !renvoEmitIntExpr(g, ep, left) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, right) {
		return false
	}
	renvoAsmCopyPrimaryToTertiary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmSubPrimaryTertiary(&g.asm)
	size := renvoTypeSize(g.meta, leftType.elem)
	if size < 1 {
		return false
	}
	divisor := size
	shift := 0
	for size > 1 && size&1 == 0 {
		shift++
		size >>= 1
	}
	if size != 1 {
		renvoAsmPushPrimary(&g.asm)
		renvoAsmPrimaryImm(&g.asm, divisor)
		renvoAsmPopTertiary(&g.asm)
		renvoAsmDivLeftTertiaryRightPrimary(&g.asm, false)
		return true
	}
	if shift != 0 {
		renvoAsmSarPrimaryImm(&g.asm, shift)
	}
	return true
}

func renvoEmitStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
		renvoAsmPrimaryImm(&g.asm, 0)
		renvoAsmSecondaryImm(&g.asm, 0)
		return true
	}
	if e.kind == renvoExprAssert {
		typ := renvoInferParsedExprType(g, ep, idx)
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
		return true
	}
	if renvoExprIsErrorStringCall(g, ep, idx) {
		callee := &ep.exprs[e.left]
		return renvoEmitStringValueRegs(g, ep, callee.left)
	}
	if e.kind == renvoExprCall && e.argCount == 1 && renvoTypeIsString(g.meta, renvoConversionTypeFromExpr(g, ep, e.left)) {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, argIndex)) {
			return renvoEmitStringValueRegs(g, ep, argIndex)
		}
		argType := renvoInferParsedExprType(g, ep, argIndex)
		argResolved := renvoResolveType(g.meta, argType)
		renvoNonNil(argResolved)
		if argResolved.kind == renvoTypeSlice {
			elem := renvoResolveType(g.meta, argResolved.elem)
			renvoNonNil(elem)
			if elem.kind == renvoTypeByte {
				return renvoEmitByteSliceStringCopyValueRegs(g, ep, argIndex)
			}
			if elem.kind == renvoTypeInt32 {
				return renvoEmitRuneSliceStringCopyValueRegs(g, ep, argIndex)
			}
		}
	}
	if e.kind == renvoExprBinary && renvoTokCharIs(g.prog, e.tok, '+') && renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, idx)) {
		return renvoEmitStringConcatValueRegs(g, ep, idx)
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(g.meta, valueType) {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, 0)
		renvoAsmPushPrimary(&g.asm)
		renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, 8)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoAsmPopPrimary(&g.asm)
		return true
	}
	return renvoGenericEmitStringValueRegs(g, ep, idx)
}

func renvoEmitIndexedStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprIndex {
		return false
	}
	container := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
	renvoNonNil(container)
	if container.kind == renvoTypePointer {
		container = renvoResolveType(g.meta, container.elem)
		renvoNonNil(container)
	}
	if container.kind != renvoTypeArray && container.kind != renvoTypeSlice {
		return false
	}
	elem := renvoResolveType(g.meta, container.elem)
	renvoNonNil(elem)
	if elem.kind != renvoTypeString || !renvoEmitIndexAddressPrimary(g, ep, idx) {
		return false
	}

	// Index-address lowering performs the ordinary array/slice bounds check and
	// leaves the element address in primary. A string occupies two backend value
	// slots: data pointer followed by byte length.
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, 0)
	renvoAsmPushPrimary(&g.asm)
	renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, renvoBackendValueSlotSize)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	return true
}

func renvoGenericEmitStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprString {
		msg := renvoDecodeStringToken(g.prog, e.tok)
		msgOff := renvoAddStringData(g, msg)
		msgLen := len(msg)
		renvoAsmPrimaryDataAddr(a, msgOff)
		renvoAsmSecondaryImm(a, msgLen)
		return true
	}
	if e.kind == renvoExprSlice {
		return renvoEmitStringSliceValueRegs(g, ep, idx)
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !renvoTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			renvoAsmLoadPrimarySecondaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && renvoTypeIsString(meta, globalType) {
			renvoAsmLoadPrimaryBss(a, globalOffset)
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset+8)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopPrimary(a)
			return true
		}
		constTok := renvoFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := renvoDecodeStringToken(g.prog, constTok)
			msgOff := renvoAddStringData(g, msg)
			msgLen := len(msg)
			renvoAsmPrimaryDataAddr(a, msgOff)
			renvoAsmSecondaryImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == renvoExprIndex {
		return renvoEmitIndexedStringValueRegs(g, ep, idx)
	}
	if e.kind == renvoExprSelector {
		valueType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, valueType) {
			return false
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopPrimary(a)
		return true
	}
	if e.kind == renvoExprCall && e.argCount == 1 && renvoExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		argType := renvoInferParsedExprType(g, ep, argIndex)
		argResolved := renvoResolveType(meta, argType)
		renvoNonNil(argResolved)
		if argResolved.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, argResolved.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeByte {
			return false
		}
		if !renvoEmitSlicePtrLen(g, ep, argIndex) {
			return false
		}
		renvoAsmPushTertiary(a)
		renvoAsmPopSecondary(a)
		return true
	}
	if e.kind == renvoExprCall {
		callType := renvoInferParsedExprType(g, ep, idx)
		if !renvoTypeIsString(meta, callType) {
			return false
		}
		if !renvoEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func renvoStringHeapOffsets(g *renvoLinearGen) {
	renvoNonNil(g)
	if g.stringHeapReady != 0 {
		return
	}
	g.stringHeapReady = 1
	g.stringHeapOff = g.asm.bssSize
	g.stringHeapEndOff = g.stringHeapOff + 8
	g.stringHeapDataOff = g.stringHeapOff + 32
	g.asm.bssSize += 32 + renvoStringArenaSize(g)
}

func renvoStringArenaSize(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.arenaSize > 0 {
		return g.arenaSize
	}
	return renvoDefaultArenaSize(g.c.renvoTarget)
}

func renvoEmitArenaAllocStackPrimary(g *renvoLinearGen, sizeOff int) {
	renvoNonNil(g)
	label := renvoEnsureArenaAllocHelper(g)
	renvoAsmLoadPrimaryStack(&g.asm, sizeOff)
	renvoAsmCallLabel(&g.asm, label)
	renvoEmitArenaAllocationCheck(g)
}

func renvoEmitArenaAllocationCheck(g *renvoLinearGen) {
	renvoNonNil(g)
	if !g.meta.panicEnabled {
		return
	}
	okLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmJnzPrimary(&g.asm, okLabel)
	if g.suppressPanicCheck {
		renvoEmitUncaughtFaultTransfer(g, true)
	} else {
		renvoEmitRuntimeFaultKind(g, renvoPanicOutOfMemoryTag, true)
	}
	renvoAsmMarkLabel(&g.asm, okLabel)
}

func renvoEnsureArenaAllocHelper(g *renvoLinearGen) int {
	return renvoEnsureDirectionalArenaAllocHelper(g, false)
}

func renvoEnsureDirectionalArenaAllocHelper(g *renvoLinearGen, persistent bool) int {
	renvoNonNil(g)
	a := &g.asm
	label := g.arenaAllocLabel
	if persistent {
		label = g.persistentAllocLabel
	}
	if label > 0 {
		return label - 1
	}
	label = renvoAsmNewLabel(a)
	if persistent {
		g.persistentAllocLabel = label + 1
	} else {
		g.arenaAllocLabel = label + 1
	}
	renvoStringHeapOffsets(g)
	if renvoRTGStructuredFunctions != 0 {
		argument := 0
		if persistent {
			argument = 1
		}
		renvoQueueStructuredHelper(g, renvoStructuredHelperArenaAlloc, argument, label)
		return label
	}
	afterLabel := renvoAsmNewLabel(a)
	helperEnd := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, label)
	renvoEmitArenaAllocHelperBody(g, persistent)
	renvoAsmMarkLabel(a, helperEnd)
	renvoAsmMarkLabel(a, afterLabel)
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		name := "__renvo_arena_alloc"
		if persistent {
			name = "__renvo_persistent_alloc"
		}
		renvoAsmAddLocalObjectFuncSymbolText(a, name, label, helperEnd)
	}
	return label
}

func renvoEmitArenaAllocHelperBody(g *renvoLinearGen, persistent bool) {
	a := &g.asm
	belowOrEqual := 0x9e
	aboveOrEqual := 0x9d
	if g.c.renvoNativeIntSize == 2 {
		// A valid compact arena can cross the native signed high bit (8000h),
		// so its address bounds must use unsigned comparisons.
		belowOrEqual = 0x96
		aboveOrEqual = 0x93
	}
	if renvoFixedTarget == 0 {
		if persistent && renvoIsSysVObject(g.c) {
			// Relocatable objects have no process entry at which to initialize their
			// private arena. Preserve the requested size while the first allocation
			// lazily establishes both bounds; later allocations take the ready branch.
			renvoAsmPushPrimary(a)
			renvoEmitPersistentArenaReady(g)
			renvoAsmPopPrimary(a)
		}
	}
	oomLabel := renvoAsmNewLabel(a)
	renvoAsmCopyPrimaryToTertiary(a)
	if persistent {
		renvoAsmLoadPrimaryBss(a, g.stringHeapEndOff)
		renvoAsmPushPrimary(a)
		renvoAsmSubPrimaryTertiary(a)
		renvoAsmPushPrimary(a)
		renvoAsmPopTertiary(a)
		renvoAsmPopPrimary(a)
		renvoAsmCmpTertiaryPrimarySet(a, belowOrEqual)
		renvoAsmJzPrimary(a, oomLabel)
		renvoAsmPushTertiary(a)
		renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
		renvoAsmPopTertiary(a)
		renvoAsmCmpTertiaryPrimarySet(a, aboveOrEqual)
		renvoAsmJzPrimary(a, oomLabel)
		renvoAsmCopyTertiaryToPrimary(a)
		renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	} else {
		renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
		renvoAsmPushPrimary(a)
		renvoAsmPushPrimary(a)
		renvoAsmAddPrimaryTertiary(a)
		renvoAsmPushPrimary(a)
		renvoAsmPopTertiary(a)
		renvoAsmPopPrimary(a)
		renvoAsmCmpTertiaryPrimarySet(a, aboveOrEqual)
		renvoAsmJzPrimary(a, oomLabel)
		renvoAsmPushTertiary(a)
		renvoAsmLoadPrimaryBss(a, g.stringHeapEndOff)
		renvoAsmPopTertiary(a)
		renvoAsmCmpTertiaryPrimarySet(a, belowOrEqual)
		renvoAsmJzPrimary(a, oomLabel)
		renvoAsmCopyTertiaryToPrimary(a)
		renvoAsmStorePrimaryBss(a, g.stringHeapOff)
		renvoAsmPopPrimary(a)
	}
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, oomLabel)
	if !persistent {
		renvoAsmPopPrimary(a)
	}
	if !g.meta.panicEnabled {
		renvoEmitUncaughtFaultTransfer(g, true)
	}
	renvoAsmPrimaryImm(a, 0)
	renvoAsmRet(a)
}

const renvoPrintIntBufferSize = 24

func renvoEmitPrintIntBufferByte(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	lenOff := g.printIntBufferOff + renvoPrintIntBufferSize + 8
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, lenOff)
	renvoAsmPushPrimary(a)
	renvoAsmPrimaryImm(a, renvoPrintIntBufferSize-1)
	renvoAsmPopTertiary(a)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryBssAddr(a, g.printIntBufferOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStoreByteMemSecondaryTertiary(a)
	renvoAsmLoadPrimaryBss(a, lenOff)
	renvoAsmIncPrimary(a)
	renvoAsmStorePrimaryBss(a, lenOff)
}

func renvoEnsurePrintIntHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.printIntEmitted {
		return g.printIntLabel
	}
	g.printIntEmitted = true
	g.printIntBufferOff = a.bssSize
	a.bssSize += renvoPrintIntBufferSize + 24
	g.printIntLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	positiveDigitLabel := renvoAsmNewLabel(a)
	digitReadyLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	valueOff := g.printIntBufferOff + renvoPrintIntBufferSize
	lenOff := valueOff + 8
	negativeOff := lenOff + 8
	renvoAsmJmpMarkLabel(a, afterLabel, g.printIntLabel)
	renvoAsmStorePrimaryBss(a, valueOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9c)
	renvoAsmStorePrimaryBss(a, negativeOff)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryBss(a, lenOff)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmLoadPrimaryBss(a, valueOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, 10)
	renvoAsmDivLeftTertiaryRightPrimary(a, true)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, negativeOff)
	renvoAsmJzPrimary(a, positiveDigitLabel)
	renvoAsmPrimaryImm(a, '0')
	renvoAsmPopTertiary(a)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmJmpMarkLabel(a, digitReadyLabel, positiveDigitLabel)
	renvoAsmPrimaryImm(a, '0')
	renvoAsmPopTertiary(a)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmMarkLabel(a, digitReadyLabel)
	renvoEmitPrintIntBufferByte(g)
	renvoAsmLoadPrimaryBss(a, valueOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, 10)
	renvoAsmDivLeftTertiaryRightPrimary(a, false)
	renvoAsmStorePrimaryBss(a, valueOff)
	renvoAsmJnzPrimary(a, loopLabel)
	renvoAsmLoadPrimaryBss(a, negativeOff)
	renvoAsmJzPrimary(a, doneLabel)
	renvoAsmPrimaryImm(a, '-')
	renvoEmitPrintIntBufferByte(g)
	renvoAsmMarkLabel(a, doneLabel)
	renvoAsmLoadPrimaryBss(a, lenOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmCopySecondaryToPrimary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryBssAddr(a, g.printIntBufferOff+renvoPrintIntBufferSize)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.printIntLabel
}

func renvoStaticSliceBackingSize(needSize int, elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if needSize < elemSize {
		needSize = elemSize
	}
	return renvoAlignTo8(needSize)
}

func renvoEmitByteSliceStringCopyValueRegs(g *renvoLinearGen, ep *renvoExprParse, argIndex int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	if !renvoEmitSlicePtrLen(g, ep, argIndex) {
		return false
	}
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStorePrimaryStack(a, srcOff)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoEmitArenaAllocStackPrimary(g, lenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoEmitCopyToFreshArena(g, srcOff, destOff, lenOff)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	return true
}

func renvoEmitUTF8StoreByte(g *renvoLinearGen, ep *renvoExprParse, loc *renvoSliceLocation, valueOff int, divisor int, prefix int) bool {
	renvoNonNil(g, ep, loc)
	a := &g.asm
	renvoAsmLoadPrimaryStack(a, valueOff)
	if divisor != 1 {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmPrimaryImm(a, divisor)
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
	}
	if prefix == 128 {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmPrimaryImm(a, 64)
		renvoAsmDivLeftTertiaryRightPrimary(a, true)
	}
	if prefix != 0 {
		renvoAsmPushPrimary(a)
		renvoAsmPrimaryImm(a, prefix)
		renvoAsmPopTertiary(a)
		renvoAsmAddPrimaryTertiary(a)
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitAppendDestPrimary(g, ep, loc, 1) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)
	return true
}

func renvoEmitUTF8Cases(g *renvoLinearGen, ep *renvoExprParse, loc *renvoSliceLocation, valueOff int) bool {
	renvoNonNil(g, ep, loc)
	a := &g.asm
	done := renvoAsmNewLabel(a)
	for width := 1; width <= 4; width++ {
		next := renvoAsmNewLabel(a)
		if width < 4 {
			limit := 1 << (width*5 + 2 - width/2)
			renvoEmitStackGreaterEqualImmJump(g, valueOff, limit, next)
		}
		divisor := 1 << (6 * (width - 1))
		prefix := 0
		if width > 1 {
			prefix = 256 - (256 >> width)
		}
		if !renvoEmitUTF8StoreByte(g, ep, loc, valueOff, divisor, prefix) {
			return false
		}
		for divisor > 1 {
			divisor = divisor / 64
			if !renvoEmitUTF8StoreByte(g, ep, loc, valueOff, divisor, 128) {
				return false
			}
		}
		renvoAsmJmpMarkLabel(a, done, next)
	}
	renvoAsmMarkLabel(a, done)
	return true
}

func renvoEmitRuneSliceStringCopyValueRegs(g *renvoLinearGen, ep *renvoExprParse, argIndex int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	srcOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	valueOff := renvoAddUnnamedLocal(g, renvoTypeInt32)
	byteSliceType := renvoAddSequenceType(g.meta, renvoTypeSlice, renvoTypeByte, 0, renvoBackendSliceValueSize)
	destOff := renvoAddUnnamedLocal(g, byteSliceType)
	renvoZeroLocalAtOffset(g, destOff)
	loc := renvoSliceLocation{offset: destOff, typ: byteSliceType, ok: true}
	if !renvoEmitSlicePtrLen(g, ep, argIndex) {
		return false
	}
	renvoAsmStorePrimaryStack(a, srcOff)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoAsmStoreStackImm(a, indexOff, 0)
	loop := renvoAsmNewLabel(a)
	done := renvoAsmNewLabel(a)
	invalid := renvoAsmNewLabel(a)
	valid := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loop)
	renvoAsmJgeStackStack(a, indexOff, lenOff, done)
	renvoAsmLoadPrimaryTertiaryStack(a, srcOff, indexOff)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 4)
	renvoAsmStorePrimaryStack(a, valueOff)
	renvoEmitStackLessImmJump(g, valueOff, 0, invalid)
	renvoEmitStackGreaterEqualImmJump(g, valueOff, 1114112, invalid)
	renvoEmitStackLessImmJump(g, valueOff, 55296, valid)
	renvoEmitStackGreaterEqualImmJump(g, valueOff, 57344, valid)
	renvoAsmJmpMarkLabel(a, invalid, invalid)
	renvoAsmStoreStackImm(a, valueOff, 65533)
	renvoAsmMarkLabel(a, valid)
	if !renvoEmitUTF8Cases(g, ep, &loc, valueOff) {
		return false
	}
	renvoAsmIncStack(a, indexOff)
	renvoAsmJmpMarkLabel(a, loop, done)
	renvoAsmLoadPrimarySecondaryStack(a, destOff, destOff-8)
	return true
}

func renvoEmitStringConcatValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	byteSliceType := renvoAddSequenceType(g.meta, renvoTypeSlice, renvoTypeByte, 0, renvoBackendSliceValueSize)
	offset := renvoAddUnnamedLocal(g, byteSliceType)
	renvoZeroLocalAtOffset(g, offset)
	loc := renvoSliceLocation{offset: offset, typ: byteSliceType, ok: true}
	if !renvoEmitStringConcatIntoLocation(g, ep, idx, &loc) {
		return false
	}
	return renvoEmitStringConcatLocationValueRegs(g, offset)
}

func renvoEmitStringConcatPairValueRegs(g *renvoLinearGen, left *renvoExprParse, leftIndex int, right *renvoExprParse, rightIndex int) bool {
	renvoNonNil(g, left, right)
	byteSliceType := renvoAddSequenceType(g.meta, renvoTypeSlice, renvoTypeByte, 0, renvoBackendSliceValueSize)
	offset := renvoAddUnnamedLocal(g, byteSliceType)
	renvoZeroLocalAtOffset(g, offset)
	loc := renvoSliceLocation{offset: offset, typ: byteSliceType, ok: true}
	if !renvoEmitStringConcatIntoLocation(g, left, leftIndex, &loc) || !renvoEmitStringConcatIntoLocation(g, right, rightIndex, &loc) {
		return false
	}
	return renvoEmitStringConcatLocationValueRegs(g, offset)
}

func renvoEmitStringConcatIntoLocation(g *renvoLinearGen, ep *renvoExprParse, idx int, loc *renvoSliceLocation) bool {
	renvoNonNil(g, ep, loc)
	e := &ep.exprs[idx]
	if e.kind == renvoExprBinary && renvoTokCharIs(g.prog, e.tok, '+') && renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, idx)) {
		if !renvoEmitStringConcatIntoLocation(g, ep, e.left, loc) {
			return false
		}
		return renvoEmitStringConcatIntoLocation(g, ep, e.right, loc)
	}
	return renvoEmitAppendStringBytesToLocation(g, ep, idx, ep, loc)
}

func renvoEmitAppendStringBytesToLocation(g *renvoLinearGen, ep *renvoExprParse, idx int, locEp *renvoExprParse, loc *renvoSliceLocation) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcIndex := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitStringValueRegs(g, ep, idx) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
	renvoAsmStoreStackImm(a, srcIndex, 0)
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmJgeStackStack(a, srcIndex, srcLen, doneLabel)
	renvoAsmLoadPrimaryTertiaryStack(a, srcPtr, srcIndex)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
	renvoAsmPushPrimary(a)
	if !renvoEmitAppendDestPrimary(g, locEp, loc, 1) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)
	renvoAsmIncStack(a, srcIndex)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	return true
}

func renvoEmitCompositeFieldToMem(g *renvoLinearGen, ep *renvoExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	renvoNonNil(g, ep)
	tempOffset := renvoAddTypedLocal(g, 0, 0, fieldType)
	if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
		return false
	}
	renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, fieldOffset, renvoTypeSize(g.meta, fieldType))
	return true
}

func renvoEmitStructReturnExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	resultType := g.meta.funcs[g.currentFunc].resultType
	resultKind := renvoResolveType(g.meta, resultType).kind
	renvoNonNil(resultKind)
	indirect := idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprUnary && renvoTokCharIs(g.prog, ep.exprs[idx].tok, '*')
	functionValueCall := idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprCall && renvoFunctionValueCalleeType(g, ep, ep.exprs[idx].left) != 0
	if resultKind != renvoTypeStruct || indirect || functionValueCall || idx >= 0 && idx < len(ep.exprs) && (ep.exprs[idx].kind == renvoExprAssert || ep.exprs[idx].kind == renvoExprSelector || ep.exprs[idx].kind == renvoExprIdent && renvoFindLocalIndex(g, ep.exprs[idx].nameStart, ep.exprs[idx].nameEnd) < 0) {
		if g.returnStruct <= 0 {
			return false
		}
		tempOffset := renvoAddUnnamedLocal(g, resultType)
		if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		renvoAsmLoadSecondaryStack(&g.asm, g.returnStruct)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, renvoTypeSize(g.meta, resultType))
		return true
	}
	return renvoEmitNativeStructReturnExpr(g, ep, idx)
}
func renvoLinearMarkFunc(g *renvoLinearGen, fnIndex int) {
	renvoNonNil(g)
	if fnIndex < 0 || fnIndex >= len(g.funcReachable) || g.funcReachable[fnIndex] {
		return
	}
	g.funcReachable[fnIndex] = true
	g.funcQueue = append(g.funcQueue, fnIndex)
	// Relocatable targets publish only explicit export-wrapper labels. The
	// implementation functions and any dependencies they reach remain local to
	// the translation unit and therefore do not enter the ELF global symtab.
	if renvoPreparedBackendActive != 0 && renvoRTGPreparedObject != 0 {
		return
	}
	if renvoFixedTarget == 0 {
		if renvoIsHostedObject(g.c) {
			return
		}
	}
	if g.c.stripSymbols && !renvoAsmNeedsFunctionSymbols(&g.asm) {
		return
	}
	src := g.meta.prog.src
	nameStart := g.meta.funcs[fnIndex].nameStart
	nameEnd := g.meta.funcs[fnIndex].nameEnd
	renvoAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func renvoObjectExportWordCount(meta *renvoMeta, fn *renvoFuncInfo) int {
	renvoNonNil(meta, fn)
	if fn.receiverType != 0 || fn.literalTok != 0 || fn.linkStatic != 0 {
		return -1
	}
	sret := renvoObjectExportUsesSRet(meta, fn)
	smallAggregateResult := renvoObjectExportUsesSmallAggregateResult(meta, fn)
	if renvoTypeUsesHiddenResult(meta, fn.resultType) && !sret && !smallAggregateResult {
		return -1
	}
	if fn.resultType != 0 && !sret && !smallAggregateResult {
		result := renvoResolveType(meta, fn.resultType)
		renvoNonNil(result)
		if !renvoTypeKindIsScalarInt(result.kind) && result.kind != renvoTypePointer && result.kind != renvoTypeFunc {
			return -1
		}
	}
	wordCount := 0
	for i := 0; i < fn.paramCount; i++ {
		paramType := meta.params[fn.firstParam+i].typ
		param := renvoResolveType(meta, paramType)
		renvoNonNil(param)
		// In C object mode a function value is the raw C function address, so it
		// occupies one SysV argument word just like an ordinary data pointer.
		paramWords := 1
		if param.kind == renvoTypeStruct {
			size := renvoTypeSize(meta, paramType)
			if size <= 16 && !renvoObjectCABIIntegerAggregate(meta, paramType) {
				return -1
			}
			paramWords = renvoAlignValue(size, 8) / 8
		} else if !renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc {
			return -1
		}
		wordCount += paramWords
	}
	if wordCount > 20 {
		return -1
	}
	return wordCount
}

func renvoObjectExportUsesSmallAggregateResult(meta *renvoMeta, fn *renvoFuncInfo) bool {
	renvoNonNil(meta, fn)
	if fn.resultType <= 0 || !renvoTypeUsesHiddenResult(meta, fn.resultType) {
		return false
	}
	return renvoObjectCABIIntegerAggregate(meta, fn.resultType)
}

func renvoObjectExportUsesSRet(meta *renvoMeta, fn *renvoFuncInfo) bool {
	renvoNonNil(meta, fn)
	if fn.resultType <= 0 || !renvoTypeUsesHiddenResult(meta, fn.resultType) {
		return false
	}
	result := renvoResolveType(meta, fn.resultType)
	return result.kind == renvoTypeStruct && renvoTypeSize(meta, fn.resultType) > 16
}

func renvoObjectExportHasMemoryAggregate(meta *renvoMeta, fn *renvoFuncInfo) bool {
	integerRegister := 0
	if renvoObjectExportUsesSRet(meta, fn) {
		integerRegister = 1
	}
	for i := 0; i < fn.paramCount; i++ {
		paramType := meta.params[fn.firstParam+i].typ
		param := renvoResolveType(meta, paramType)
		if param.kind == renvoTypeStruct {
			size := renvoTypeSize(meta, paramType)
			words := renvoAlignValue(size, 8) / 8
			if size > 16 || integerRegister+words > 6 {
				return true
			}
			integerRegister += words
		} else if integerRegister < 6 {
			integerRegister++
		}
	}
	return false
}

func renvoObjectCABIIntegerAggregate(meta *renvoMeta, typ int) bool {
	renvoNonNil(meta)
	resolved := renvoResolveType(meta, typ)
	if resolved.kind != renvoTypeStruct {
		return false
	}
	size := renvoTypeSize(meta, typ)
	if size <= 0 || size > 16 {
		return false
	}
	for i := 0; i < resolved.count; i++ {
		field := &meta.fields[resolved.first+i]
		fieldType := renvoResolveType(meta, field.typ)
		fieldSize := renvoTypeSize(meta, field.typ)
		alignment := fieldSize
		if alignment > 8 {
			alignment = 8
		}
		if alignment < 1 || field.offset%alignment != 0 || field.offset+fieldSize > size {
			return false
		}
		if fieldType.kind == renvoTypeStruct {
			if !renvoObjectCABIIntegerAggregate(meta, field.typ) {
				return false
			}
		} else if !renvoTypeKindIsScalarInt(fieldType.kind) && fieldType.kind != renvoTypePointer && fieldType.kind != renvoTypeFunc {
			return false
		}
	}
	return true
}

func renvoBeginObjectProgram(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	g := renvoBeginLinearProgram(p, meta)
	if g == nil {
		return nil
	}
	// Relocatable objects have no process entrypoint that can run the ordinary
	// global initializer function. Allocate each declaration directly into its
	// requested ELF storage and retain a compact virtual-BSS offset for common
	// load/store lowering; the object writer maps that offset back to a section.
	for i := 0; i < len(meta.globals); i++ {
		s := &meta.globals[i]
		if s.kind != renvoTokVar {
			continue
		}
		var decl *renvoObjectDecl
		if s.objectDecl > 0 && s.objectDecl < len(meta.objectDecls) {
			decl = &meta.objectDecls[s.objectDecl]
		}
		if decl != nil && (decl.kind == renvoObjectDeclFunctionAlias || decl.kind == renvoObjectDeclVariableAlias) {
			renvoObjectAppendDataSymbol(&g.asm, p.src, decl, 0, 0, 0, false, 0, nil)
			if decl.kind == renvoObjectDeclFunctionAlias {
				target := renvoFindMetaFunction(meta, decl.targetStart, decl.targetEnd)
				if target >= 0 {
					renvoLinearMarkFunc(g, target)
				}
			}
			continue
		}
		if decl != nil && decl.kind == renvoObjectDeclVariableExtern {
			importID := renvoAsmAddExternalImportRange(&g.asm, p.src, decl.nameStart, decl.nameEnd)
			if importID < 0 || importID >= (2147483647-renvoObjectExternalBase)/renvoObjectExternalStride {
				return nil
			}
			off := renvoObjectExternalBase + importID*renvoObjectExternalStride
			s.iotaValue = off
			g.globals = append(g.globals, renvoGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
			g.asm.objectExternals = append(g.asm.objectExternals, renvoObjectExternal{offset: off, importID: importID})
			continue
		}
		renvoNativeTypeLayout(meta, s.typ)
		storageSize := renvoTypeCopySize(meta, s.typ)
		alignment := meta.types[s.typ].nativeAlign
		if alignment < 1 {
			alignment = 1
		}
		if decl != nil && decl.alignment > alignment {
			alignment = decl.alignment
		}
		size := storageSize
		if decl != nil && decl.size >= 0 {
			size = decl.size
		}
		// Some C layouts deliberately use an indirect Go carrier that is much
		// smaller than the declared object (for example an aligned page whose
		// only represented member occupies its first bytes).  Reserve and encode
		// the complete C object so following symbols cannot overlap its tail.
		objectStorageSize := storageSize
		if size > objectStorageSize {
			objectStorageSize = size
		}
		off := renvoAlignValue(g.asm.bssSize, alignment)
		s.iotaValue = off
		g.globals = append(g.globals, renvoGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		g.asm.bssSize = off + objectStorageSize
		initialized := s.initStart < s.initEnd
		value := 0
		var objectValue []byte
		if initialized {
			if decl == nil || decl.relocationKind == 0 || decl.kind == renvoObjectDeclStaticCall {
				var valueOK bool
				objectValue, valueOK = renvoObjectConstantData(g, s, size, off)
				if !valueOK {
					renvoPrintErr("renvo: object constant failed: ")
					write(2, p.src[s.nameStart:s.nameEnd], -1)
					renvoPrintErr("\n")
					return nil
				}
				for at := 0; at < len(objectValue) && at < 8; at++ {
					value |= int(objectValue[at]) << (at * 8)
				}
			}
		}
		if decl == nil {
			decl = &renvoObjectDecl{kind: renvoObjectDeclVariable, nameStart: s.nameStart, nameEnd: s.nameEnd, size: size}
		}
		relocationBase := len(g.asm.objectDataRelocs)
		renvoObjectAppendDataSymbol(&g.asm, p.src, decl, off, size, objectStorageSize, initialized, value, objectValue)
		if decl.relocationKind != 0 && len(g.asm.objectDataRelocs) > relocationBase {
			fnIndex := renvoFindMetaFunction(meta, decl.relocationTargetStart, decl.relocationTargetEnd)
			if fnIndex >= 0 {
				fn := &meta.funcs[fnIndex]
				relocation := &g.asm.objectDataRelocs[len(g.asm.objectDataRelocs)-1]
				if fn.linkStatic != 0 {
					if renvoAsmAddExternalImportRange(&g.asm, p.src, fn.linkMethodStart, fn.linkMethodEnd) < 0 {
						return nil
					}
					relocation.targetStart, relocation.targetEnd = renvoAsmCopyObjectText(&g.asm, p.src, fn.linkMethodStart, fn.linkMethodEnd)
				} else if fn.exportNameEnd <= fn.exportNameStart {
					start, end, wrapperOK := renvoEnsureCObjectFunctionPointerWrapper(g, fnIndex)
					if !wrapperOK {
						return nil
					}
					relocation.targetStart, relocation.targetEnd = start, end
				}
			}
		}
	}
	// Executables allocate this state while emitting their entrypoint. Objects
	// have no entrypoint, so reserve it after declared globals before wrapper
	// discovery can mark and emit a defer-aware address-taken function.
	if meta.panicEnabled {
		renvoEnsurePanicState(g)
	}
	for i := 0; i < len(meta.funcs); i++ {
		fn := &meta.funcs[i]
		if fn.exportNameEnd <= fn.exportNameStart {
			continue
		}
		if !renvoEmitObjectExport(g, i) {
			renvoPrintErr("renvo: object export failed: ")
			write(2, p.src[fn.nameStart:fn.nameEnd], -1)
			renvoPrintErr("\n")
			return nil
		}
	}
	if !renvoEnsureObjectAddressTakenFunctionWrappers(g) {
		return nil
	}
	return g
}

func renvoEnsureObjectAddressTakenFunctionWrappers(g *renvoLinearGen) bool {
	renvoNonNil(g)
	// Export and static-data wrappers seed funcQueue. Discover callback uses
	// over their direct-call graph without changing the compiler's normal
	// emission queue or function order.
	queue := make([]int, len(g.funcQueue))
	copy(queue, g.funcQueue)
	seen := make([]bool, len(g.meta.funcs))
	for cursor := 0; cursor < len(queue); cursor++ {
		caller := queue[cursor]
		if caller < 0 || caller >= len(g.meta.funcs) {
			return false
		}
		if seen[caller] {
			continue
		}
		seen[caller] = true
		fn := &g.meta.funcs[caller]
		for tok := fn.bodyStart; tok < fn.bodyEnd; tok++ {
			if !renvoTokIsKind(g.prog, tok, renvoTokIdent) ||
				tok > fn.bodyStart && renvoTokCharIs(g.prog, tok-1, '.') {
				continue
			}
			target := renvoFindMetaFunction(g.meta, int(renvoTokStart(g.prog, tok)), int(renvoTokEnd(g.prog, tok)))
			if target < 0 {
				continue
			}
			shadowed := false
			for paramIndex := 0; paramIndex < fn.paramCount; paramIndex++ {
				param := &g.meta.params[fn.firstParam+paramIndex]
				if renvoBytesEqualRange(g.prog.src, int(renvoTokStart(g.prog, tok)), int(renvoTokEnd(g.prog, tok)),
					param.nameStart, param.nameEnd) {
					shadowed = true
					break
				}
			}
			if shadowed {
				continue
			}
			targetFn := &g.meta.funcs[target]
			directCall := tok+1 < fn.bodyEnd && renvoTokCharIs(g.prog, tok+1, '(') &&
				!(tok > fn.bodyStart && renvoTokIsKind(g.prog, tok-1, renvoTokIdent) &&
					renvoTokIdentIs(g.prog, tok-1, "defer"))
			if targetFn.linkStatic != 0 {
				continue
			}
			if directCall {
				if !seen[target] {
					queue = append(queue, target)
				}
				continue
			}
			if targetFn.exportNameEnd > targetFn.exportNameStart {
				continue
			}
			if _, _, ok := renvoEnsureCObjectFunctionPointerWrapper(g, target); !ok {
				renvoPrintErr("renvo: object callback wrapper failed: ")
				write(2, g.prog.src[targetFn.nameStart:targetFn.nameEnd], -1)
				renvoPrintErr(" near ")
				if tok > fn.bodyStart {
					write(2, g.prog.src[renvoTokStart(g.prog, tok-1):renvoTokEnd(g.prog, tok-1)], -1)
				}
				renvoPrintErr(" <target> ")
				if tok+1 < fn.bodyEnd {
					write(2, g.prog.src[renvoTokStart(g.prog, tok+1):renvoTokEnd(g.prog, tok+1)], -1)
				}
				renvoPrintErr("\n")
				return false
			}
			if !seen[target] {
				queue = append(queue, target)
			}
		}
	}
	return true
}

func renvoObjectConstantData(g *renvoLinearGen, symbol *renvoSymbolInfo, size int, objectOffset int) ([]byte, bool) {
	renvoNonNil(g, symbol)
	if size < 0 || symbol.initStart >= symbol.initEnd {
		renvoPrintErr("renvo: object constant has invalid range\n")
		return nil, false
	}
	ep := renvoNewExprParse()
	if !renvoParseExpressionOK(ep, g.prog, symbol.initStart, symbol.initEnd) {
		renvoPrintErr("renvo: object constant expression parse failed\n")
		renvoPrintErr("renvo: object constant parser stopped at ")
		renvoPrintIntErr(ep.pos)
		renvoPrintErr(" of ")
		renvoPrintIntErr(ep.end)
		renvoPrintErr("\n")
		if ep.pos >= 0 && ep.pos < ep.end {
			start := int(renvoTokStart(g.prog, ep.pos))
			end := int(renvoTokEnd(g.prog, ep.pos))
			renvoPrintErr("renvo: object constant token: ")
			write(2, g.prog.src[start:end], -1)
			renvoPrintErr("\n")
		}
		return nil, false
	}
	data := make([]byte, size)
	if !renvoObjectStoreConstant(g, ep, len(ep.exprs)-1, symbol.typ, data, 0, objectOffset) {
		root := &ep.exprs[len(ep.exprs)-1]
		renvoPrintErr("renvo: object constant root kind/type ")
		renvoPrintIntErr(root.kind)
		renvoPrintErr(" ")
		renvoPrintIntErr(renvoResolveType(g.meta, symbol.typ).kind)
		renvoPrintErr("\n")
		return nil, false
	}
	return data, true
}

func renvoObjectStoreConstant(g *renvoLinearGen, ep *renvoExprParse, idx int, typ int, data []byte, offset int, objectOffset int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) || offset < 0 {
		return false
	}
	// GNU C permits empty structs and unions. Their shared-Go carriers retain a
	// synthetic byte so fields remain addressable, but explicit object metadata
	// records the true zero-byte C storage. There can be no value or relocation
	// to encode in an empty object, regardless of the carrier initializer shape.
	if len(data) == 0 {
		return offset == 0
	}
	resolved := renvoResolveType(g.meta, typ)
	e := &ep.exprs[idx]
	functionExpr := idx
	for functionExpr >= 0 && functionExpr < len(ep.exprs) {
		candidate := &ep.exprs[functionExpr]
		if candidate.kind == renvoExprUnary && renvoTokCharIs(g.prog, candidate.tok, '&') &&
			candidate.left >= 0 && candidate.left < len(ep.exprs) {
			functionExpr = candidate.left
			continue
		}
		if candidate.kind == renvoExprCall && candidate.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, candidate.left)
			if conversionType != 0 && renvoTypeSize(g.meta, conversionType) == g.c.renvoNativeIntSize {
				functionExpr = renvo_runtime_UnsafeIntAt(ep.args, candidate.firstArg)
				continue
			}
		}
		break
	}
	functionIdent := e
	if functionExpr >= 0 && functionExpr < len(ep.exprs) {
		functionIdent = &ep.exprs[functionExpr]
	}
	if functionIdent.kind == renvoExprIdent {
		fnIndex := renvoFindMetaFunction(g.meta, functionIdent.nameStart, functionIdent.nameEnd)
		if fnIndex >= 0 {
			fn := &g.meta.funcs[fnIndex]
			if fn.linkStatic != 0 {
				if renvoAsmAddExternalImportRange(&g.asm, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd) < 0 {
					return false
				}
				start, end := renvoAsmCopyObjectText(&g.asm, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
				g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
					offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1})
				return true
			}
			start, end, ok := renvoEnsureCObjectFunctionPointerWrapper(g, fnIndex)
			if !ok {
				return false
			}
			g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
				offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1})
			return true
		}
		if e.kind == renvoExprIdent && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
			return true
		}
	}
	objectAggregatePrefix := "__c_object_aggregate_"
	if e.kind == renvoExprCall && e.left >= 0 && e.left < len(ep.exprs) &&
		ep.exprs[e.left].kind == renvoExprIdent && e.argCount > 0 {
		callee := &ep.exprs[e.left]
		fnIndex := renvoFindMetaFunction(g.meta, callee.nameStart, callee.nameEnd)
		if fnIndex >= 0 && renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, objectAggregatePrefix) {
			fn := &g.meta.funcs[fnIndex]
			if fn.paramCount != e.argCount {
				return false
			}
			at := callee.nameStart + len(objectAggregatePrefix)
			for i := 0; i < e.argCount; i++ {
				fieldOffset := 0
				start := at
				for at < callee.nameEnd {
					value := renvo_runtime_UnsafeByteAt(g.prog.src, at)
					if value < '0' || value > '9' {
						break
					}
					fieldOffset = fieldOffset*10 + int(value-'0')
					at++
				}
				if at == start || at >= callee.nameEnd || renvo_runtime_UnsafeByteAt(g.prog.src, at) != '_' {
					return false
				}
				at++
				arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
				if !renvoObjectStoreConstant(g, ep, arg, g.meta.params[fn.firstParam+i].typ,
					data, offset+fieldOffset, objectOffset) {
					return false
				}
			}
			return at < callee.nameEnd && renvo_runtime_UnsafeByteAt(g.prog.src, at) == 'x'
		}
	}
	objectUnionPrefix := "__c_object_union_init_"
	if e.kind == renvoExprCall && e.left >= 0 && e.left < len(ep.exprs) && e.argCount == 1 &&
		ep.exprs[e.left].kind == renvoExprIdent {
		callee := &ep.exprs[e.left]
		fnIndex := renvoFindMetaFunction(g.meta, callee.nameStart, callee.nameEnd)
		if fnIndex >= 0 && renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, objectUnionPrefix) {
			fn := &g.meta.funcs[fnIndex]
			if fn.paramCount != 1 {
				return false
			}
			at := callee.nameStart + len(objectUnionPrefix)
			fieldOffset := 0
			start := at
			for at < callee.nameEnd {
				value := renvo_runtime_UnsafeByteAt(g.prog.src, at)
				if value < '0' || value > '9' {
					break
				}
				fieldOffset = fieldOffset*10 + int(value-'0')
				at++
			}
			if at == start || at+1 >= callee.nameEnd || renvo_runtime_UnsafeByteAt(g.prog.src, at) != '_' ||
				renvo_runtime_UnsafeByteAt(g.prog.src, at+1) != 'x' {
				return false
			}
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			return renvoObjectStoreConstant(g, ep, arg, g.meta.params[fn.firstParam].typ, data, offset+fieldOffset, objectOffset)
		}
	}
	if resolved.kind == renvoTypeArray || resolved.kind == renvoTypeStruct {
		if e.kind != renvoExprComposite {
			return false
		}
		next := 0
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldType, fieldOffset := 0, 0
			if resolved.kind == renvoTypeArray {
				at := next
				if field.key >= 0 {
					key := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, field.key, 0)
					if !key.ok {
						return false
					}
					at = key.value
				}
				if at < 0 || at >= resolved.count {
					return false
				}
				fieldType = resolved.elem
				elementSize := renvoTypeSize(g.meta, fieldType)
				if g.c.objectFile && renvoResolveType(g.meta, fieldType).kind == renvoTypeFunc {
					// A C function pointer occupies one native pointer even though
					// Renvo function values also carry an internal context word.
					elementSize = g.c.renvoNativeIntSize
				}
				fieldOffset = at * elementSize
				next = at + 1
			} else {
				fieldIndex := renvoCompositeStructFieldIndex(g, typ, &field, i)
				if fieldIndex < 0 {
					return false
				}
				fieldType = g.meta.fields[fieldIndex].typ
				fieldOffset = g.meta.fields[fieldIndex].offset
			}
			if !renvoObjectStoreConstant(g, ep, field.expr, fieldType, data, offset+fieldOffset, objectOffset) {
				renvoPrintErr("renvo: object aggregate constant field failed ")
				renvoPrintIntErr(i)
				renvoPrintErr(" type ")
				renvoPrintIntErr(renvoResolveType(g.meta, fieldType).kind)
				renvoPrintErr(" expr ")
				if field.expr >= 0 && field.expr < len(ep.exprs) {
					renvoPrintIntErr(ep.exprs[field.expr].kind)
				} else {
					renvoPrintIntErr(-1)
				}
				if field.expr >= 0 && field.expr < len(ep.exprs) && ep.exprs[field.expr].kind == renvoExprIdent {
					value := &ep.exprs[field.expr]
					renvoPrintErr(": ")
					write(2, g.prog.src[value.nameStart:value.nameEnd], -1)
				}
				renvoPrintErr("\n")
				return false
			}
		}
		return true
	}
	if resolved.kind == renvoTypePointer && e.kind == renvoExprIdent {
		targetType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		if targetType.kind == renvoTypeArray {
			start, end := renvoAsmCopyObjectText(&g.asm, g.prog.src, e.nameStart, e.nameEnd)
			g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
				offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1})
			return true
		}
	}
	if resolved.kind == renvoTypePointer || renvoTypeKindIsScalarInt(resolved.kind) && renvoTypeSize(g.meta, typ) == g.c.renvoNativeIntSize {
		nameStart, nameEnd, addend, ok := renvoObjectConstantPointerAddress(g, ep, idx)
		if ok {
			start, end := renvoAsmCopyObjectText(&g.asm, g.prog.src, nameStart, nameEnd)
			g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
				offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1, addend: addend})
			return true
		}
	}
	if resolved.kind == renvoTypePointer {
		value, ok := renvoObjectConstantPointerValue(g, ep, idx)
		size := renvoTypeSize(g.meta, typ)
		if ok && size >= 1 && size <= 8 && offset+size <= len(data) {
			for at := 0; at < size; at++ {
				data[offset+at] = byte(value >> (at * 8))
			}
			return true
		}
	}
	if resolved.kind == renvoTypePointer && e.kind == renvoExprCall && e.argCount == 1 &&
		(renvoExprIsIdentText(g.prog, ep, e.left, "renvo_runtime_CStringPointer") ||
			renvoExprIdentPrefixText(g.prog, ep, e.left, "renvo_runtime_CWideStringPointer")) {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if arg < 0 || arg >= len(ep.exprs) || ep.exprs[arg].kind != renvoExprString {
			return false
		}
		msg := renvoDecodeStringToken(g.prog, ep.exprs[arg].tok)
		alignment := 1
		callee := &ep.exprs[e.left]
		if renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "renvo_runtime_CWideStringPointer") {
			alignment = int(renvo_runtime_UnsafeByteAt(g.prog.src, callee.nameEnd-1) - '0')
		}
		dataOffset := renvoAddStringDataAligned(g, msg, alignment)
		if dataOffset < 0 {
			return false
		}
		g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
			offset: objectOffset + offset, typ: 1, addend: dataOffset})
		return true
	}
	if resolved.kind == renvoTypePointer && e.kind == renvoExprCall && e.argCount == 1 {
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 && renvoResolveType(g.meta, conversionType).kind == renvoTypePointer {
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			return renvoObjectStoreConstant(g, ep, arg, typ, data, offset, objectOffset)
		}
	}
	if resolved.kind == renvoTypePointer && e.kind == renvoExprCall && e.argCount == 1 {
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 && renvoResolveType(g.meta, conversionType).kind == renvoTypePointer {
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			if arg >= 0 && arg < len(ep.exprs) && ep.exprs[arg].kind == renvoExprIdent {
				target := &ep.exprs[arg]
				targetType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
				if targetType.kind == renvoTypeArray {
					start, end := renvoAsmCopyObjectText(&g.asm, g.prog.src, target.nameStart, target.nameEnd)
					g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
						offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1})
					return true
				}
			}
		}
	}
	if resolved.kind == renvoTypePointer && e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '&') &&
		e.left >= 0 && e.left < len(ep.exprs) {
		nameStart, nameEnd, addend, ok := renvoObjectStaticAddress(g, ep, e.left)
		if !ok {
			return false
		}
		start, end := renvoAsmCopyObjectText(&g.asm, g.prog.src, nameStart, nameEnd)
		g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
			offset: objectOffset + offset, targetStart: start, targetEnd: end, typ: 1, addend: addend})
		return true
	}
	if renvoTypeKindIsFloat(resolved.kind) {
		bits, ok := renvoObjectFloatConstantBits(g.prog, ep, idx, resolved.kind)
		size := renvoTypeSize(g.meta, typ)
		if !ok || size != 4 && size != 8 || offset+size > len(data) {
			return false
		}
		for at := 0; at < size; at++ {
			data[offset+at] = byte(bits >> (at * 8))
		}
		return true
	}
	// Integer tokens are split into low and high compiler words whenever the
	// target has a 32-bit native int.  The ordinary one-word constant evaluator
	// intentionally returns only the low word, but an eight-byte object
	// initializer must materialize both.  Reconstruct the literal here before
	// falling back to the scalar evaluator used by narrower values and compound
	// expressions.
	if e.kind == renvoExprInt && renvoTypeKindIsScalarInt(resolved.kind) &&
		renvoTypeSize(g.meta, typ) == 8 && offset+8 <= len(data) {
		low := renvoParseIntToken(g.prog, e.tok)
		bits := uint64(uint32(low)) | uint64(uint32(g.prog.parsedIntHigh))<<32
		for at := 0; at < 8; at++ {
			data[offset+at] = byte(bits >> (at * 8))
		}
		return true
	}
	if !renvoTypeKindIsScalarInt(resolved.kind) && resolved.kind != renvoTypePointer && resolved.kind != renvoTypeBool {
		return false
	}
	value := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, idx, 0)
	size := renvoTypeSize(g.meta, typ)
	if !value.ok || size < 1 || size > 8 || offset+size > len(data) {
		return false
	}
	for at := 0; at < size; at++ {
		data[offset+at] = byte(value.value >> (at * 8))
	}
	return true
}

func renvoObjectFloatConstantBits(p *renvoProgram, ep *renvoExprParse, idx int, kind int) (uint64, bool) {
	renvoNonNil(p, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return 0, false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprFloat {
		if kind == renvoTypeFloat32 {
			return renvoParseFloatTokenBits(p, e.tok, 23, 8, 127), true
		}
		return renvoParseFloatTokenBits(p, e.tok, 52, 11, 1023), true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(p, e.tok, '-') {
		value, ok := renvoObjectFloatConstantBits(p, ep, e.left, kind)
		if kind == renvoTypeFloat32 {
			value ^= uint64(1) << 31
		} else {
			value ^= uint64(1) << 63
		}
		return value, ok
	}
	return 0, false
}

func renvoObjectConstantPointerValue(g *renvoLinearGen, ep *renvoExprParse, idx int) (int, bool) {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return 0, false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprCall && e.argCount == 1 {
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 && renvoTypeSize(g.meta, conversionType) == g.c.renvoNativeIntSize {
			return renvoObjectConstantPointerValue(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
	}
	if e.kind == renvoExprCall && e.argCount == 2 && e.left >= 0 && e.left < len(ep.exprs) {
		callee := &ep.exprs[e.left]
		if callee.kind == renvoExprIdent {
			increment := renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_step_inc_")
			decrement := renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_step_dec_")
			if increment || decrement {
				baseExpr := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
				countExpr := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)
				base, baseOK := renvoObjectConstantPointerValue(g, ep, baseExpr)
				count := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, countExpr, 0)
				baseType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, baseExpr))
				if !baseOK || !count.ok || baseType.kind != renvoTypePointer {
					return 0, false
				}
				delta := count.value * renvoTypeSize(g.meta, baseType.elem)
				if decrement {
					delta = -delta
				}
				return base + delta, true
			}
		}
	}
	value := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, idx, 0)
	return value.value, value.ok
}

func renvoObjectConstantPointerAddress(g *renvoLinearGen, ep *renvoExprParse, idx int) (int, int, int, bool) {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return 0, 0, 0, false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		typ := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		if typ.kind == renvoTypeArray && renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokVar) >= 0 {
			return e.nameStart, e.nameEnd, 0, true
		}
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '&') {
		return renvoObjectStaticAddress(g, ep, e.left)
	}
	if e.kind == renvoExprBinary && (renvoTokCharIs(g.prog, e.tok, '+') || renvoTokCharIs(g.prog, e.tok, '-')) {
		nameStart, nameEnd, addend, ok := renvoObjectConstantPointerAddress(g, ep, e.left)
		right := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, e.right, 0)
		if ok && right.ok {
			if renvoTokCharIs(g.prog, e.tok, '-') {
				right.value = -right.value
			}
			return nameStart, nameEnd, addend + right.value, true
		}
		if renvoTokCharIs(g.prog, e.tok, '+') {
			nameStart, nameEnd, addend, ok = renvoObjectConstantPointerAddress(g, ep, e.right)
			left := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, e.left, 0)
			if ok && left.ok {
				return nameStart, nameEnd, addend + left.value, true
			}
		}
	}
	if e.kind != renvoExprCall {
		return 0, 0, 0, false
	}
	if e.argCount == 1 {
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 && renvoTypeSize(g.meta, conversionType) == g.c.renvoNativeIntSize {
			return renvoObjectConstantPointerAddress(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		callee := &ep.exprs[e.left]
		if callee.kind == renvoExprSelector &&
			renvoBytesEqualText(g.prog.src, callee.nameStart, callee.nameEnd, "Pointer") &&
			(renvoExprIsIdentText(g.prog, ep, callee.left, "unsafe") ||
				renvoExprIsIdentText(g.prog, ep, callee.left, "__c_unsafe")) {
			return renvoObjectConstantPointerAddress(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
	}
	if e.argCount != 2 || e.left < 0 || e.left >= len(ep.exprs) {
		return 0, 0, 0, false
	}
	callee := &ep.exprs[e.left]
	if callee.kind != renvoExprIdent {
		return 0, 0, 0, false
	}
	arrayIndex := renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_array_index_")
	increment := renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_step_inc_") ||
		renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_index_") || arrayIndex
	decrement := renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_step_dec_")
	if !increment && !decrement {
		return 0, 0, 0, false
	}
	base := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	countExpr := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)
	nameStart, nameEnd, addend, ok := renvoObjectConstantPointerAddress(g, ep, base)
	count := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, countExpr, 0)
	baseType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, base))
	if arrayIndex {
		baseType = renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	}
	if !ok || !count.ok || baseType.kind != renvoTypePointer {
		return 0, 0, 0, false
	}
	delta := count.value * renvoTypeSize(g.meta, baseType.elem)
	if decrement {
		delta = -delta
	}
	return nameStart, nameEnd, addend + delta, true
}

func renvoObjectStaticAddress(g *renvoLinearGen, ep *renvoExprParse, idx int) (int, int, int, bool) {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return 0, 0, 0, false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprCall && e.argCount == 0 && e.left >= 0 && e.left < len(ep.exprs) {
		selector := &ep.exprs[e.left]
		if selector.kind == renvoExprSelector {
			fieldOffset, fieldOK := renvoCFieldAccessorOffset(g.prog.src, selector.nameStart, selector.nameEnd)
			nameStart, nameEnd, addend, baseOK := renvoObjectStaticAddress(g, ep, selector.left)
			if fieldOK && baseOK {
				return nameStart, nameEnd, addend + fieldOffset, true
			}
		}
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') && e.left >= 0 && e.left < len(ep.exprs) {
		call := &ep.exprs[e.left]
		if call.kind == renvoExprCall && call.argCount == 0 && call.left >= 0 && call.left < len(ep.exprs) {
			selector := &ep.exprs[call.left]
			if selector.kind == renvoExprSelector {
				fieldOffset, fieldOK := renvoCFieldAccessorOffset(g.prog.src, selector.nameStart, selector.nameEnd)
				nameStart, nameEnd, addend, baseOK := renvoObjectStaticAddress(g, ep, selector.left)
				if fieldOK && baseOK {
					return nameStart, nameEnd, addend + fieldOffset, true
				}
			}
		}
	}
	if e.kind == renvoExprIdent {
		if renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokVar) < 0 {
			return 0, 0, 0, false
		}
		return e.nameStart, e.nameEnd, 0, true
	}
	if e.kind == renvoExprIndex {
		nameStart, nameEnd, addend, ok := renvoObjectStaticAddress(g, ep, e.left)
		if !ok {
			return 0, 0, 0, false
		}
		index := renvoEvalMetaParsedConstExpr(g.meta, g.prog, ep, e.right, 0)
		baseType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
		if !index.ok || baseType.kind != renvoTypeArray || index.value < 0 || index.value >= baseType.count {
			return 0, 0, 0, false
		}
		return nameStart, nameEnd, addend + index.value*renvoTypeSize(g.meta, baseType.elem), true
	}
	if e.kind == renvoExprSelector {
		nameStart, nameEnd, addend, ok := renvoObjectStaticAddress(g, ep, e.left)
		if !ok {
			return 0, 0, 0, false
		}
		baseType := renvoInferParsedExprType(g, ep, e.left)
		if renvoResolveType(g.meta, baseType).kind == renvoTypePointer {
			return 0, 0, 0, false
		}
		fieldOffset := renvoStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
		if fieldOffset < 0 || g.fieldPointerIndex >= 0 {
			return 0, 0, 0, false
		}
		return nameStart, nameEnd, addend + fieldOffset, true
	}
	return 0, 0, 0, false
}

func renvoFindMetaFunctionText(m *renvoMeta, name string) int {
	renvoNonNil(m)
	for i := 0; i < len(m.funcs); i++ {
		fn := &m.funcs[i]
		if fn.receiverType == 0 && renvoBytesEqualText(m.prog.src, fn.nameStart, fn.nameEnd, name) {
			return i
		}
	}
	return -1
}

func renvoEmitObjectRawFunctionCall(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if fn.linkStatic == 0 {
		renvoLinearMarkFunc(g, fnIndex)
		renvoAsmCallLabel(&g.asm, g.funcLabels[fnIndex])
		return true
	}
	return renvoEmitObjectExternalCall(g, fn)
}

func renvoEnsureCObjectFunctionPointerWrapper(g *renvoLinearGen, fnIndex int) (int, int, bool) {
	renvoNonNil(g)
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return 0, 0, false
	}
	if len(g.objectCABIWrapperLabels) == 0 {
		g.objectCABIWrapperLabels = make([]int, len(g.meta.funcs))
		g.objectCABINameStarts = make([]int, len(g.meta.funcs))
		g.objectCABINameEnds = make([]int, len(g.meta.funcs))
	}
	if g.objectCABIWrapperLabels[fnIndex] != 0 {
		return g.objectCABINameStarts[fnIndex], g.objectCABINameEnds[fnIndex], true
	}
	fn := &g.meta.funcs[fnIndex]
	wordCount := renvoObjectExportWordCount(g.meta, fn)
	if renvoFixedTarget == 0 && renvoIsCdeclObject(g.c) {
		wordCount = renvoObjectExportWordCount386(g.meta, fn)
		if wordCount < 0 {
			wordCount = renvoObjectExportWordCount(g.meta, fn)
		}
	}
	if wordCount < 0 {
		return 0, 0, false
	}
	wrapper := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, wrapper)
	start, end := renvoAsmCopyObjectPrefixedText(&g.asm, "__renvo_cabi_", g.prog.src, fn.nameStart, fn.nameEnd)
	symbolIndex := len(g.asm.symbols)
	g.asm.symbols = append(g.asm.symbols, renvoAsmSymbol{nameStart: start, nameEnd: end, label: wrapper})
	if fn.objectDecl > 0 && fn.objectDecl < len(g.meta.objectDecls) {
		decl := &g.meta.objectDecls[fn.objectDecl]
		sectionStart, sectionEnd := renvoAsmCopyObjectText(&g.asm, g.prog.src, decl.sectionStart, decl.sectionEnd)
		g.asm.symbols[symbolIndex].sectionStart = sectionStart
		g.asm.symbols[symbolIndex].sectionEnd = sectionEnd
		g.asm.symbols[symbolIndex].alignment = decl.alignment
	}
	if renvoFixedTarget == 0 && renvoIsCdeclObject(g.c) {
		if !renvoEmitCdeclObjectWrapperBody(g, fnIndex, wordCount, false) {
			return 0, 0, false
		}
		endLabel := renvoAsmNewLabel(&g.asm)
		renvoAsmMarkLabel(&g.asm, endLabel)
		g.asm.symbols[symbolIndex].endLabel = endLabel
		g.objectCABIWrapperLabels[fnIndex] = wrapper
		g.objectCABINameStarts[fnIndex] = start
		g.objectCABINameEnds[fnIndex] = end
		return start, end, true
	}
	if !renvoEmitObjectRegisterWrapperBody(g, fnIndex, wordCount, false) {
		return 0, 0, false
	}
	endLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, endLabel)
	if symbolIndex >= 0 && symbolIndex < len(g.asm.symbols) {
		g.asm.symbols[symbolIndex].endLabel = endLabel
	}
	g.objectCABIWrapperLabels[fnIndex] = wrapper
	g.objectCABINameStarts[fnIndex] = start
	g.objectCABINameEnds[fnIndex] = end
	return start, end, true
}

func renvoAsmAddExternalImportRange(a *renvoAsm, src []byte, nameStart int, nameEnd int) int {
	renvoNonNil(a)
	if nameEnd <= nameStart {
		return -1
	}
	for i := 0; i+1 < len(a.kernelImportOffsets); i += 2 {
		start := a.kernelImportOffsets[i]
		end := a.kernelImportOffsets[i+1]
		if end-start != nameEnd-nameStart {
			continue
		}
		match := true
		for j := 0; j < end-start; j++ {
			if a.kernelImportNames[start+j] != src[nameStart+j] {
				match = false
			}
		}
		if match {
			return i / 2
		}
	}
	start := len(a.kernelImportNames)
	for i := nameStart; i < nameEnd; i++ {
		a.kernelImportNames = append(a.kernelImportNames, src[i])
	}
	a.kernelImportOffsets = append(a.kernelImportOffsets, start, len(a.kernelImportNames))
	return len(a.kernelImportOffsets)/2 - 1
}

func renvoObjectExternalOffset(a *renvoAsm, importID int) int {
	renvoNonNil(a)
	if importID < 0 || importID >= (2147483647-renvoObjectExternalBase)/renvoObjectExternalStride {
		return -1
	}
	offset := renvoObjectExternalBase + importID*renvoObjectExternalStride
	for i := 0; i < len(a.objectExternals); i++ {
		if a.objectExternals[i].importID == importID {
			return a.objectExternals[i].offset
		}
	}
	a.objectExternals = append(a.objectExternals, renvoObjectExternal{offset: offset, importID: importID})
	return offset
}

func renvoEmitObjectFunctionAddress(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	if !renvoIsHostedObject(g.c) || fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	nameStart, nameEnd := fn.nameStart, fn.nameEnd
	nameSource := g.prog.src
	if fn.linkStatic != 0 {
		nameStart, nameEnd = fn.linkMethodStart, fn.linkMethodEnd
	} else if fn.exportNameEnd > fn.exportNameStart {
		// Object exports are C ABI wrappers. Taking the address of a source
		// function must therefore name the wrapper, not its internal Go-ABI body.
		nameStart, nameEnd = fn.exportNameStart, fn.exportNameEnd
	} else {
		// A local function can escape through an automatic initializer or as a
		// call argument, neither of which produces an object-data relocation.
		// Indirect object calls use the platform C ABI, so those addresses must
		// name a C-ABI wrapper just like addresses stored in static data do.
		if len(g.objectCABIWrapperLabels) == 0 || fnIndex >= len(g.objectCABIWrapperLabels) ||
			g.objectCABIWrapperLabels[fnIndex] == 0 {
			return false
		}
		nameStart = g.objectCABINameStarts[fnIndex]
		nameEnd = g.objectCABINameEnds[fnIndex]
		nameSource = g.asm.symbolName
	}
	importID := renvoAsmAddExternalImportRange(&g.asm, nameSource, nameStart, nameEnd)
	offset := renvoObjectExternalOffset(&g.asm, importID)
	if offset < 0 {
		return false
	}
	// The object writer resolves this virtual-BSS address against the named
	// symbol. If the function is defined in this object it reuses that symbol;
	// otherwise it emits SHN_UNDEF for the system linker to resolve.
	renvoAsmPrimaryBssAddr(&g.asm, offset)
	if fn.linkStatic == 0 {
		renvoLinearMarkFunc(g, fnIndex)
	}
	return true
}

func renvoEmitObjectKernelLinkAddress(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if !renvoIsSysVObject(g.c) {
		return false
	}
	nameStart, nameEnd, addend, ok := renvoObjectConstantPointerAddress(g, ep, idx)
	if !ok || renvoAsmAddExternalImportRange(&g.asm, g.prog.src, nameStart, nameEnd) < 0 {
		return false
	}
	targetStart, targetEnd := renvoAsmCopyObjectText(&g.asm, g.prog.src, nameStart, nameEnd)
	// The kernel code model requires the linker's absolute symbol value even
	// while early startup is executing through an identity mapping. A normal
	// RIP-relative LEA would instead produce the temporary physical address.
	renvoAsmEmit16(&g.asm, 0xb848)
	sourceLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, sourceLabel)
	renvoAsmEmit64(&g.asm, 0)
	g.asm.objectDataRelocs = append(g.asm.objectDataRelocs, renvoObjectDataRelocation{
		offset: -sourceLabel - 1, targetStart: targetStart, targetEnd: targetEnd, typ: 1, addend: addend})
	return true
}

// Entry signatures are language policy. Target hooks receive only the number
// of validated slices and the runtime's entry-state location.
func renvoEntryParameterCount(g *renvoLinearGen, appIndex int) int {
	if appIndex < 0 || appIndex >= len(g.meta.funcs) {
		return -1
	}
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) || app.paramCount > 2 {
		return -1
	}
	for i := 0; i < app.paramCount; i++ {
		if !renvoTypeIsStringSlice(g.meta, g.meta.params[app.firstParam+i].typ) {
			return -1
		}
	}
	return app.paramCount
}

func renvoEmitProgramEntryArgs(g *renvoLinearGen, appIndex int, entryStateOffset int) bool {
	count := renvoEntryParameterCount(g, appIndex)
	return count >= 0 && renvoEmitProcessEntryWords(g, count, entryStateOffset)
}

func renvoEmitImageEntryArgs(g *renvoLinearGen, appIndex int) bool {
	count := renvoEntryParameterCount(g, appIndex)
	return count >= 0 && renvoEmitImageEntryWords(g, count)
}

// renvoEmitApplicationEntry owns language-level startup ordering. Definitions
// preserve physical incoming words, restore reserved registers, and terminate
// the entry; they do not inspect parsed function metadata or initialize globals.
func renvoEmitApplicationEntry(g *renvoLinearGen, appIndex int, image bool, entryStateOffset int) bool {
	renvoLinearMarkFunc(g, appIndex)
	if !renvoEmitProgramEntryFrame(&g.asm, image) {
		return false
	}
	if !g.meta.panicEnabled {
		renvoEmitEntryRuntimeRegisters(g)
	}
	renvoEmitInitializeThreadState(g)
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return false
	}
	if image {
		if !renvoEmitImageEntryArgs(g, appIndex) {
			return false
		}
	} else {
		if !renvoEmitProgramEntryArgs(g, appIndex, entryStateOffset) {
			return false
		}
		// Process argument decoding may use reserved runtime registers as scratch.
		if !g.meta.panicEnabled {
			renvoEmitEntryRuntimeRegisters(g)
		}
	}
	renvoAsmCallLabel(&g.asm, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return false
	}
	return renvoEmitProgramExit(&g.asm, image)
}

// Program orchestration is shared. Definitions own only layout, physical
// entry setup, object-code transforms and image construction.
func renvoBeginScalarProgram(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	renvoNonNil(p, meta)
	if renvoObjectProgram(meta.c) {
		return renvoBeginObjectProgram(p, meta)
	}
	appIndex := p.entryFunc
	if appIndex < 0 {
		return nil
	}
	if renvoReleaseProgramDeclarations(meta.c) {
		renvo_runtime_ArenaDiscardDecls(p.decls)
		renvo_runtime_ArenaDiscardFuncs(p.funcs)
	}
	g := new(renvoLinearGen)
	renvoInitLinearProgram(g, p, meta, renvoPreparedBackendActive != 0 || renvoFixedTarget == 0)
	// Some execution formats compile a target-specific compiler, while others
	// preserve a source-declared selector or a dynamic command-line fallback.
	mode := renvoProgramTargetMode(g.c)
	if mode == 1 {
		renvoLoadCompilerFixedTarget(g)
		if g.fixedTargetState != 1 {
			g.fixedTargetState = 1
			g.fixedTargetValue = 0
		}
	} else if mode == 2 {
		g.fixedTargetState = 1
		g.fixedTargetValue = g.c.renvoTarget
	}
	image := renvoFixedTarget == 0 && meta.c.emitImage && renvoProgramImageEntry(g.c)
	g.darwinEntryOff = renvoSetupProgramLayout(&g.asm, image, len(meta.funcs))
	if g.darwinEntryOff == -2 {
		return nil
	}
	renvoInitProgramFunctions(g, renvoPreparedBackendActive == 0 && renvoFixedTarget != 0 && mode == 0)
	if renvoKernelProgram(g.c) {
		if !renvoBeginKernelModule(g, appIndex) {
			return nil
		}
	} else if !renvoEmitApplicationEntry(g, appIndex, image, g.darwinEntryOff) {
		return nil
	}
	return g
}

// Whole-program function-value dispatch can reference a closure whose parent
// is folded away. Backends requiring a complete label space select this repair;
// source reachability stays shared, and definitions only emit an empty body.
func renvoResolveSpeculativeClosureLabels(g *renvoLinearGen) {
	for closureIndex := 0; closureIndex < len(g.meta.closures); closureIndex++ {
		closure := &g.meta.closures[closureIndex]
		fnIndex := closure.fnIndex
		if closure.ready || fnIndex < 0 || fnIndex >= len(g.funcLabels) ||
			renvoAsmLabelPosition(&g.asm, g.funcLabels[fnIndex]) >= 0 {
			continue
		}
		renvoEmitEmptyFunction(&g.asm, g.funcLabels[fnIndex])
	}
}

func renvoFinishScalarProgram(g *renvoLinearGen) renvoCompileResult {
	renvoNonNil(g)
	a := &g.asm
	if renvoResolveUnemittedClosures(g.c) {
		renvoResolveSpeculativeClosureLabels(g)
	}
	if renvoPreparedBackendActive != 0 && renvoRTGUnsupportedOperation != 0 {
		renvoRTGReportFailure(g)
		return renvoCompileResult{}
	}
	if renvoObjectProgram(g.c) {
		// Names and spans belong to the frontend arena: copy them before release.
		renvoRecordObjectFunctionRanges(g)
		if !renvoFinalizeObjectCode(a) {
			return renvoCompileResult{}
		}
	}
	if renvoReleaseProgramScratch(g.c) {
		renvo_runtime_ArenaDiscard(g.meta.scratchStart, g.meta.scratchEnd)
	}
	var result renvoCompileResult
	renvoBuildProgramImage(a, g.kernelInitLabel, g.kernelExitLabel, &result)
	if renvoPreparedBackendActive != 0 {
		renvoRTGValidateRelocations(a)
		if renvoRTGUnsupportedOperation != 0 {
			renvoRTGReportFailure(g)
			return renvoCompileResult{}
		}
		if len(result.data) == 0 && !renvoObjectProgram(g.c) && !renvoKernelProgram(g.c) {
			if renvoRTGImageLimit > 0 {
				renvoRTGReportImageSize(g)
			} else {
				renvoPrintErr("renvo: error RENVO-BUG-020 (backend): target image encoder returned no output or diagnostic\n")
			}
			renvoRTGUnsupportedOperation = 5001
			return renvoCompileResult{}
		}
	}
	result.ok = !a.patchFailed && len(result.data) != 0
	return result
}

func renvoTryCompileScalarProgramScratch(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram(p, meta)
	if g == nil {
		renvoPrintErr("renvo: failed to begin program\n")
		return renvoCompileResult{}
	}
	if !renvoEmitAllQueuedFunctionsScratch(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram(g)
}

func renvoTryCompileScalarProgramCached(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	g := renvoBeginScalarProgram(p, meta)
	if g == nil || !renvoEmitAllQueuedFunctionsCached(g) {
		return renvoCompileResult{}
	}
	return renvoFinishScalarProgram(g)
}

// A resumable program has the same emitter and finalizer as a synchronous
// compilation. Only the number of queue entries visited per step differs.
type renvoProgramSession struct {
	gen        *renvoLinearGen
	queueIndex int
	done       bool
	result     renvoCompileResult
}

func renvoBeginProgramSession(p *renvoProgram, meta *renvoMeta) *renvoProgramSession {
	g := renvoBeginScalarProgram(p, meta)
	if g == nil {
		return nil
	}
	return &renvoProgramSession{gen: g}
}

func (s *renvoProgramSession) step(functionLimit int) bool {
	if s == nil || s.done {
		return true
	}
	if functionLimit < 1 {
		functionLimit = 1
	}
	failed := renvoEmitQueuedFunctionsCached(s.gen, &s.queueIndex, functionLimit)
	if failed >= 0 {
		renvoPrintFailedFunction(s.gen, failed)
		s.done = true
		return true
	}
	if s.queueIndex < len(s.gen.funcQueue) {
		return false
	}
	s.result = renvoFinishScalarProgram(s.gen)
	s.done = true
	return true
}

func renvoPrintFailedFunction(g *renvoLinearGen, fnIndex int) {
	if renvoFixedTarget == 0 {
		fn := &g.meta.funcs[fnIndex]
		renvoPrintErr("renvo: failed function: ")
		write(2, g.prog.src[fn.nameStart:fn.nameEnd], -1)
		renvoPrintErr("\n")
	}
}

func renvoInitLinearProgram(g *renvoLinearGen, p *renvoProgram, meta *renvoMeta, optimizeRuntime bool) {
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	g.c.optimizeRuntime = optimizeRuntime && len(p.src) >= renvoLargeProgramSourceThreshold
	renvoAsmInitWithContext(&g.asm, g.c)
}

func renvoInitProgramFunctions(g *renvoLinearGen, reserve bool) {
	count := len(g.meta.funcs)
	if reserve {
		g.funcLabels = make([]int, 0, count)
	}
	for i := 0; i < count; i++ {
		g.funcLabels = append(g.funcLabels, renvoAsmNewLabel(&g.asm))
	}
	renvoInitFuncQueue(g, count)
}

func renvoBeginLinearProgram(p *renvoProgram, meta *renvoMeta) *renvoLinearGen {
	renvoNonNil(p, meta)
	renvo_runtime_ArenaDiscardDecls(p.decls)
	renvo_runtime_ArenaDiscardFuncs(p.funcs)
	g := new(renvoLinearGen)
	renvoInitLinearProgram(g, p, meta, renvoFixedTarget == 0)
	renvoInitProgramFunctions(g, renvoFixedTarget != 0)
	return g
}

func renvoObjectExportWordCount386(meta *renvoMeta, fn *renvoFuncInfo) int {
	if fn.receiverType != 0 || fn.literalTok != 0 || fn.linkStatic != 0 {
		return -1
	}
	if fn.resultType != 0 {
		result := renvoResolveType(meta, fn.resultType)
		if (!renvoTypeKindIsScalarInt(result.kind) && result.kind != renvoTypePointer && result.kind != renvoTypeFunc) ||
			renvoTypeSize(meta, fn.resultType) > 8 {
			return -1
		}
		// The internal 32-bit ABI returns wide scalars through a hidden pointer;
		// the i386 C ABI publishes the same two words in edx:eax.
		if renvoTypeUsesHiddenResult(meta, fn.resultType) && renvoTypeSize(meta, fn.resultType) != 8 {
			return -1
		}
	}
	words := 0
	for i := 0; i < fn.paramCount; i++ {
		typ := meta.params[fn.firstParam+i].typ
		param := renvoResolveType(meta, typ)
		if (!renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc) ||
			renvoTypeKindIsScalarInt(param.kind) && renvoTypeSize(meta, typ) > 4 {
			return -1
		}
		words++
	}
	return words
}

func renvoEmitCdeclObjectWrapperBody(g *renvoLinearGen, fnIndex int, wordCount int, variadic bool) bool {
	if wordCount < 0 || fnIndex < 0 || fnIndex >= len(g.meta.funcs) || variadic && wordCount < 1 {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	wideResult := fn.resultType != 0 && renvoTypeUsesHiddenResult(g.meta, fn.resultType) &&
		renvoTypeSize(g.meta, fn.resultType) == 2*g.c.renvoNativeIntSize
	renvoObjectExportFrame(g, true)
	if wideResult && !renvoBeginObjectAggregateResult(&g.asm, false) {
		return false
	}
	fixedWords := wordCount
	registerWords := renvoObjectArgumentRegisterCount(g.c)
	if variadic {
		fixedWords--
		registerWords = 0
		renvoReserveObjectVariadicArgs(g, fixedWords)
	}
	// The entry function binds its incoming words in source order, unlike
	// ordinary functions. Only ordering belongs here; locations are target-owned.
	for at := 0; at < fixedWords; at++ {
		word := at
		if fnIndex == g.prog.entryFunc {
			word = fixedWords - 1 - at
		}
		if word < registerWords {
			if !renvoAsmPushObjectRegisterWordKind(&g.asm, word, 0) {
				return false
			}
		} else if !renvoAsmPushObjectStackWordKind(&g.asm, word-registerWords, 0) {
			return false
		}
	}
	if variadic {
		renvoPushObjectVariadicArgs(&g.asm, fixedWords)
	}
	callWords := wordCount
	if wideResult {
		if !renvoPushObjectPrivateResult(&g.asm, wordCount) {
			return false
		}
		callWords++
	}
	renvoLinearMarkFunc(g, fnIndex)
	renvoObjectCallWithWordCount(g, fnIndex, callWords)
	if variadic {
		renvoFinishObjectVariadicArgs(&g.asm)
	}
	if wideResult {
		renvoFinishObjectAggregateResult(&g.asm, 2)
	}
	renvoObjectExportFrame(g, false)
	renvoAsmRet(&g.asm)
	return true
}

func renvoEmitObjectExport386(g *renvoLinearGen, fnIndex int) bool {
	fn := &g.meta.funcs[fnIndex]
	wordCount := renvoObjectExportWordCount386(g.meta, fn)
	if wordCount < 0 {
		wordCount = renvoObjectExportWordCount(g.meta, fn)
	}
	if wordCount < 0 || fn.exportNameEnd <= fn.exportNameStart {
		return false
	}
	wrapper := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, wrapper)
	var decl *renvoObjectDecl
	if fn.objectDecl > 0 && fn.objectDecl < len(g.meta.objectDecls) {
		decl = &g.meta.objectDecls[fn.objectDecl]
	}
	variadic := decl != nil && decl.kind == renvoObjectDeclFunction && decl.relocationAddend != 0
	symbolIndex := renvoAsmAddObjectFuncSymbol(
		&g.asm, g.prog.src, fn.exportNameStart, fn.exportNameEnd, wrapper, decl)
	if !renvoEmitCdeclObjectWrapperBody(g, fnIndex, wordCount, variadic) {
		return false
	}
	endLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, endLabel)
	if symbolIndex >= 0 && symbolIndex < len(g.asm.symbols) {
		g.asm.symbols[symbolIndex].endLabel = endLabel
	}
	return true
}

// renvoEmitObjectRegisterWrapperBody adapts an incoming register-ABI call to
// the compiler's internal word-based convention. Target operations own the
// physical frame, incoming locations, and aggregate-result storage.
func renvoEmitObjectRegisterWrapperBody(g *renvoLinearGen, fnIndex int, wordCount int, variadic bool) bool {
	fn := &g.meta.funcs[fnIndex]
	sret := renvoObjectExportUsesSRet(g.meta, fn)
	smallAggregateResult := renvoObjectExportUsesSmallAggregateResult(g.meta, fn)
	memoryAggregate := renvoObjectExportHasMemoryAggregate(g.meta, fn)
	if variadic && (wordCount < 1 || wordCount > 7 || renvoPreparedBackendActive != 0 || sret || smallAggregateResult) ||
		renvoPreparedBackendActive != 0 && (wordCount > renvoRTGObjectRegisterCount() && !sret || wordCount > renvoRTGObjectRegisterCount()-1 && sret || memoryAggregate) {
		return false
	}
	renvoObjectExportFrame(g, true)
	registerWords := renvoObjectArgumentRegisterCount(g.c)
	if sret {
		registerWords--
	}
	if !variadic && (wordCount > registerWords || memoryAggregate) {
		renvoBeginObjectStackArgs(&g.asm)
	}
	if sret {
		if !renvoBeginObjectAggregateResult(&g.asm, true) {
			return false
		}
		if !renvoPushObjectExportArgs(g, fn, true, fn.paramCount) {
			return false
		}
		if !renvoPushObjectSRetPointer(&g.asm) {
			return false
		}
	} else {
		if smallAggregateResult {
			if !renvoBeginObjectAggregateResult(&g.asm, false) {
				return false
			}
		}
		if variadic {
			fixedCount := wordCount - 1
			renvoReserveObjectVariadicArgs(g, fixedCount)
			if !renvoPushObjectExportArgs(g, fn, false, fn.paramCount-1) {
				return false
			}
			renvoPushObjectVariadicArgs(&g.asm, fixedCount)
		} else if !renvoPushObjectExportArgs(g, fn, false, fn.paramCount) {
			return false
		}
		if smallAggregateResult {
			if !renvoPushObjectPrivateResult(&g.asm, wordCount) {
				return false
			}
		}
	}
	renvoLinearMarkFunc(g, fnIndex)
	callWords := wordCount
	if sret || smallAggregateResult {
		callWords++
	}
	renvoObjectCallWithWordCount(g, fnIndex, callWords)
	if variadic {
		renvoFinishObjectVariadicArgs(&g.asm)
	}
	if sret || smallAggregateResult {
		resultWords := 1
		if !sret && renvoTypeSize(g.meta, fn.resultType) > 8 {
			resultWords = 2
		}
		renvoFinishObjectAggregateResult(&g.asm, resultWords)
	}
	renvoObjectExportFrame(g, false)
	renvoAsmRet(&g.asm)
	return true
}

func renvoEmitObjectExport(g *renvoLinearGen, fnIndex int) bool {
	renvoNonNil(g)
	if renvoFixedTarget == 0 && renvoIsCdeclObject(g.c) {
		return renvoEmitObjectExport386(g, fnIndex)
	}
	fn := &g.meta.funcs[fnIndex]
	wordCount := renvoObjectExportWordCount(g.meta, fn)
	if wordCount < 0 || fn.exportNameEnd <= fn.exportNameStart {
		return false
	}
	wrapper := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, wrapper)
	var decl *renvoObjectDecl
	if fn.objectDecl > 0 && fn.objectDecl < len(g.meta.objectDecls) {
		decl = &g.meta.objectDecls[fn.objectDecl]
	}
	variadic := decl != nil && decl.kind == renvoObjectDeclFunction && decl.relocationAddend != 0
	symbolIndex := renvoAsmAddObjectFuncSymbol(
		&g.asm, g.prog.src, fn.exportNameStart, fn.exportNameEnd, wrapper, decl)
	if !renvoEmitObjectRegisterWrapperBody(g, fnIndex, wordCount, variadic) {
		return false
	}
	endLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmMarkLabel(&g.asm, endLabel)
	if symbolIndex >= 0 && symbolIndex < len(g.asm.symbols) {
		g.asm.symbols[symbolIndex].endLabel = endLabel
	}
	return true
}

// Prepared object backends express the private aggregate-result carrier using
// the selected RTG architecture and ABI instead of embedding amd64 opcodes in
// the shared object lowering. Two words preserve the SysV stack alignment and
// accommodate every integer aggregate returned in registers.
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

func renvoObjectRegisterWordCount(wordCount int, limit int) int {
	if wordCount > limit {
		return limit
	}
	return wordCount
}

func renvoPushObjectExportArgs(g *renvoLinearGen, fn *renvoFuncInfo, sret bool, paramCount int) bool {
	registerLimit := renvoObjectArgumentRegisterCount(g.c)
	integerRegister := 0
	if sret {
		integerRegister = 1
	}
	stackWord := 0
	for i := 0; i < paramCount; i++ {
		paramType := g.meta.params[fn.firstParam+i].typ
		param := renvoResolveType(g.meta, paramType)
		renvoNonNil(param)
		words := 1
		memory := false
		if param.kind == renvoTypeStruct {
			size := renvoTypeSize(g.meta, paramType)
			words = renvoAlignValue(size, 8) / 8
			memory = size > 16 || integerRegister+words > registerLimit
		} else {
			memory = integerRegister >= registerLimit
		}
		if memory {
			for word := 0; word < words; word++ {
				kind := 0
				if param.kind != renvoTypeStruct {
					kind = renvoObjectABINormalizeKind(param.kind)
				}
				if !renvoAsmPushObjectStackWordKind(&g.asm, stackWord, kind) {
					return false
				}
				stackWord++
			}
		} else {
			for word := 0; word < words; word++ {
				register := integerRegister + word
				if param.kind == renvoTypeStruct {
					// Internal Renvo calls bind parameters in reverse source order,
					// but retain natural word order inside each aggregate. Reverse
					// the pushes within the C aggregate so the later register pops
					// preserve that word order.
					register = integerRegister + words - 1 - word
				}
				kind := 0
				if param.kind != renvoTypeStruct {
					kind = renvoObjectABINormalizeKind(param.kind)
				}
				if !renvoAsmPushObjectRegisterWordKind(&g.asm, register, kind) {
					return false
				}
			}
			integerRegister += words
		}
	}
	return true
}

func renvoObjectABINormalizeKind(kind int) int {
	if kind == renvoTypeBool {
		return renvoTypeByte
	}
	if kind == renvoTypeByte || kind == renvoTypeInt8 || kind == renvoTypeInt16 || kind == renvoTypeInt32 ||
		kind == renvoTypeUint16 || kind == renvoTypeUint32 {
		return kind
	}
	return 0
}

func renvoObjectAppendDataSymbol(a *renvoAsm, src []byte, decl *renvoObjectDecl, offset int, size int, storageSize int, initialized bool, value int, objectValue []byte) {
	renvoNonNil(a, decl)
	nameStart, nameEnd := renvoAsmCopyObjectText(a, src, decl.nameStart, decl.nameEnd)
	sectionStart, sectionEnd := renvoAsmCopyObjectText(a, src, decl.sectionStart, decl.sectionEnd)
	targetStart, targetEnd := renvoAsmCopyObjectText(a, src, decl.targetStart, decl.targetEnd)
	valueStart := len(a.objectDataValues)
	a.objectDataValues = append(a.objectDataValues, objectValue...)
	a.objectData = append(a.objectData, renvoObjectDataSymbol{nameStart: nameStart, nameEnd: nameEnd,
		sectionStart: sectionStart, sectionEnd: sectionEnd, targetStart: targetStart, targetEnd: targetEnd,
		offset: offset, size: size, storageSize: storageSize, alignment: decl.alignment, binding: decl.binding,
		visibility: decl.visibility, initialized: renvoBoolInt(initialized), value: value,
		valueStart: valueStart, valueEnd: len(a.objectDataValues), kind: decl.kind})
	if decl.relocationKind != 0 {
		targetStart, targetEnd := renvoAsmCopyObjectText(a, src, decl.relocationTargetStart, decl.relocationTargetEnd)
		relocationOffset, addend := offset, decl.relocationAddend
		if decl.kind == renvoObjectDeclStaticCall {
			// The x86 static-call template begins with E9 followed by a PC32
			// displacement relative to the end of that four-byte field.
			relocationOffset++
			addend -= 4
		}
		a.objectDataRelocs = append(a.objectDataRelocs, renvoObjectDataRelocation{offset: relocationOffset,
			targetStart: targetStart, targetEnd: targetEnd, typ: decl.relocationKind, addend: addend})
	}
}

func renvoBoolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func renvoRecordObjectFunctionRanges(g *renvoLinearGen) {
	renvoNonNil(g)
	sectionOwners, addressEscapes := renvoInferObjectFunctionSectionOwners(g)
	for i := 0; i < len(g.meta.funcs) && i < len(g.funcLabels); i++ {
		fn := &g.meta.funcs[i]
		if renvoAsmLabelPosition(&g.asm, g.funcLabels[i]) < 0 {
			continue
		}
		nameStart, nameEnd := 0, 0
		if fn.exportNameEnd > fn.exportNameStart {
			nameStart, nameEnd = renvoAsmCopyObjectPrefixedText(&g.asm, "__renvo_impl_", g.prog.src, fn.nameStart, fn.nameEnd)
		} else {
			nameStart, nameEnd = renvoAsmCopyObjectText(&g.asm, g.prog.src, fn.nameStart, fn.nameEnd)
		}
		sectionStart, sectionEnd, alignment := 0, 0, 0
		sectionOwner := i
		if i < len(sectionOwners) && sectionOwners[i] >= 0 {
			sectionOwner = sectionOwners[i]
		}
		sectionFn := &g.meta.funcs[sectionOwner]
		sectionAlignment := 0
		if sectionFn.objectDecl > 0 && sectionFn.objectDecl < len(g.meta.objectDecls) {
			decl := &g.meta.objectDecls[sectionFn.objectDecl]
			sectionStart, sectionEnd = renvoAsmCopyObjectText(&g.asm, g.prog.src, decl.sectionStart, decl.sectionEnd)
			sectionAlignment = decl.alignment
			if sectionOwner == i {
				alignment = decl.alignment
			}
		}
		if i < len(g.objectCABIWrapperLabels) && g.objectCABIWrapperLabels[i] > 0 {
			for symbol := 0; symbol < len(g.asm.symbols); symbol++ {
				if g.asm.symbols[symbol].label != g.objectCABIWrapperLabels[i] {
					continue
				}
				if i < len(addressEscapes) && addressEscapes[i] {
					// Object data points at the C-ABI wrapper, not the internal
					// implementation. Keep that persistent callback in .text even
					// when the implementation itself is explicitly init-only.
					g.asm.symbols[symbol].sectionStart = 0
					g.asm.symbols[symbol].sectionEnd = 0
					g.asm.symbols[symbol].alignment = 0
				} else {
					g.asm.symbols[symbol].sectionStart = sectionStart
					g.asm.symbols[symbol].sectionEnd = sectionEnd
					g.asm.symbols[symbol].alignment = sectionAlignment
				}
				break
			}
		}
		start := renvoAsmLabelPosition(&g.asm, g.funcLabels[i])
		end := len(g.asm.code)
		for j := 0; j < len(g.funcLabels); j++ {
			position := renvoAsmLabelPosition(&g.asm, g.funcLabels[j])
			if position > start && position < end {
				end = position
			}
		}
		g.asm.objectFunctions = append(g.asm.objectFunctions, renvoObjectFunctionRange{label: g.funcLabels[i], end: end,
			nameStart: nameStart, nameEnd: nameEnd, sectionStart: sectionStart, sectionEnd: sectionEnd, alignment: alignment})
	}
}

func renvoInferObjectFunctionSectionOwners(g *renvoLinearGen) ([]int, []bool) {
	renvoNonNil(g)
	count := len(g.meta.funcs)
	owners := make([]int, count)
	canInfer := make([]bool, count)
	addressEscapes := make([]bool, count)
	for i := 0; i < count; i++ {
		owners[i] = -1
		fn := &g.meta.funcs[i]
		if fn.objectDecl <= 0 || fn.objectDecl >= len(g.meta.objectDecls) {
			continue
		}
		decl := &g.meta.objectDecls[fn.objectDecl]
		if renvoObjectCodeSectionIsSpecial(g.prog, decl.sectionStart, decl.sectionEnd) {
			owners[i] = i
		}
		canInfer[i] = decl.binding == 0 && fn.linkStatic == 0 && fn.receiverType == 0 &&
			fn.literalTok == 0 && fn.exportNameEnd <= fn.exportNameStart
	}
	for target := 0; target < count && target < len(g.objectCABINameStarts); target++ {
		nameStart, nameEnd := g.objectCABINameStarts[target], g.objectCABINameEnds[target]
		if nameEnd <= nameStart {
			continue
		}
		for relocation := 0; relocation < len(g.asm.objectDataRelocs); relocation++ {
			entry := &g.asm.objectDataRelocs[relocation]
			if !renvoBytesEqualRange(g.asm.symbolName, entry.targetStart, entry.targetEnd, nameStart, nameEnd) {
				continue
			}
			if !renvoObjectDataRelocationHasPersistentLifetime(g, entry) {
				continue
			}
			// Aggregate constant lowering discovers function-pointer fields after
			// metadata construction. Its emitted relocation is the authoritative
			// indication that the C-ABI wrapper escapes into persistent data.
			canInfer[target] = false
			owners[target] = -1
			addressEscapes[target] = true
			break
		}
	}
	edges := make([]int, 0, count*2)
	for caller := 0; caller < count; caller++ {
		fn := &g.meta.funcs[caller]
		for tok := fn.bodyStart; tok+1 < fn.bodyEnd; tok++ {
			if !renvoTokIsKind(g.prog, tok, renvoTokIdent) || !renvoTokCharIs(g.prog, tok+1, '(') ||
				tok > fn.bodyStart && renvoTokCharIs(g.prog, tok-1, '.') {
				continue
			}
			target := renvoFindMetaFunction(g.meta, int(renvoTokStart(g.prog, tok)), int(renvoTokEnd(g.prog, tok)))
			if target >= 0 && target != caller && canInfer[target] {
				edges = append(edges, caller, target)
			}
		}
	}
	changed := true
	candidateOwners := make([]int, count)
	haveCallers := make([]bool, count)
	blocked := make([]bool, count)
	for changed {
		changed = false
		for i := 0; i < count; i++ {
			candidateOwners[i] = -1
			haveCallers[i] = false
			blocked[i] = false
		}
		for at := 0; at+1 < len(edges); at += 2 {
			caller, target := edges[at], edges[at+1]
			if !canInfer[target] || owners[target] >= 0 {
				continue
			}
			haveCallers[target] = true
			callerOwner := owners[caller]
			if callerOwner < 0 {
				blocked[target] = true
			} else if candidateOwners[target] < 0 {
				candidateOwners[target] = callerOwner
			} else {
				candidateOwners[target] = renvoObjectMergeFunctionSectionOwner(g, candidateOwners[target], callerOwner)
				if candidateOwners[target] < 0 {
					blocked[target] = true
				}
			}
		}
		for target := 0; target < count; target++ {
			if canInfer[target] && owners[target] < 0 && haveCallers[target] &&
				!blocked[target] && candidateOwners[target] >= 0 {
				owners[target] = candidateOwners[target]
				changed = true
			}
		}
	}
	return owners, addressEscapes
}

func renvoObjectDataRelocationHasPersistentLifetime(g *renvoLinearGen, relocation *renvoObjectDataRelocation) bool {
	renvoNonNil(g, relocation)
	for i := 0; i < len(g.asm.objectData); i++ {
		item := &g.asm.objectData[i]
		size := item.storageSize
		if size < item.size {
			size = item.size
		}
		if size < 1 {
			size = 1
		}
		if relocation.offset < item.offset || relocation.offset >= item.offset+size {
			continue
		}
		return !renvoBytesPrefixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".init") &&
			!renvoBytesSuffixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".init") &&
			!renvoBytesPrefixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".exit") &&
			!renvoBytesSuffixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".exit") &&
			!renvoBytesPrefixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".ref") &&
			!renvoBytesPrefixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".meminit") &&
			!renvoBytesPrefixText(g.asm.symbolName, item.sectionStart, item.sectionEnd, ".discard") &&
			// The linker consumes the early-console declarations during boot and
			// discards the table with the rest of init data. Its historical name
			// predates the usual .init.* convention.
			!renvoBytesEqualText(g.asm.symbolName, item.sectionStart, item.sectionEnd, "__earlycon_table")
	}
	return true
}

func renvoObjectCodeSectionIsSpecial(p *renvoProgram, start int, end int) bool {
	renvoNonNil(p)
	return end > start && !renvoBytesEqualText(p.src, start, end, "-") &&
		!renvoBytesEqualText(p.src, start, end, ".text") &&
		!renvoBytesPrefixText(p.src, start, end, ".text.")
}

func renvoObjectFunctionsShareSection(g *renvoLinearGen, left int, right int) bool {
	if left < 0 || right < 0 || left >= len(g.meta.funcs) || right >= len(g.meta.funcs) {
		return false
	}
	leftFn := &g.meta.funcs[left]
	rightFn := &g.meta.funcs[right]
	if leftFn.objectDecl <= 0 || leftFn.objectDecl >= len(g.meta.objectDecls) ||
		rightFn.objectDecl <= 0 || rightFn.objectDecl >= len(g.meta.objectDecls) {
		return false
	}
	leftDecl := &g.meta.objectDecls[leftFn.objectDecl]
	rightDecl := &g.meta.objectDecls[rightFn.objectDecl]
	return renvoBytesEqualRange(g.prog.src, leftDecl.sectionStart, leftDecl.sectionEnd,
		rightDecl.sectionStart, rightDecl.sectionEnd)
}

func renvoObjectMergeFunctionSectionOwner(g *renvoLinearGen, left int, right int) int {
	if renvoObjectFunctionsShareSection(g, left, right) {
		return left
	}
	if renvoObjectFunctionSectionHasPrefix(g, left, ".ref.text") {
		return left
	}
	if renvoObjectFunctionSectionHasPrefix(g, right, ".ref.text") {
		return right
	}
	return -1
}

func renvoObjectFunctionSectionHasPrefix(g *renvoLinearGen, fnIndex int, prefix string) bool {
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if fn.objectDecl <= 0 || fn.objectDecl >= len(g.meta.objectDecls) {
		return false
	}
	decl := &g.meta.objectDecls[fn.objectDecl]
	return renvoBytesPrefixText(g.prog.src, decl.sectionStart, decl.sectionEnd, prefix)
}

func renvoInitFuncQueue(g *renvoLinearGen, count int) {
	renvoNonNil(g)
	g.funcReachable = make([]bool, count)
	g.funcQueue = make([]int, 0, count)
	if renvoFixedTarget == 0 && renvoIsSysVObject(g.c) {
		g.funcSingleCallState = make([]int, count)
		g.paramConstValues = make([]int, len(g.meta.params))
		g.paramConstValid = make([]bool, len(g.meta.params))
		renvoRecordFunctionDirectUseCounts(g)
	}
	// Allocate reusable control-flow stacks before per-function scratch marks;
	// their first growth must not pin an entire function's parser scratch.
	g.breakLabels = make([]int, 0, 32)
	g.continueLabels = make([]int, 0, 32)
}

func renvoRecordFunctionDirectUseCounts(g *renvoLinearGen) {
	renvoNonNil(g)
	p := g.prog
	for tok := 0; tok < renvoTokCount(p); tok++ {
		if !renvoTokIsKind(p, tok, renvoTokIdent) {
			continue
		}
		fnIndex := renvoFindMetaFunction(g.meta, int(renvoTokStart(p, tok)), int(renvoTokEnd(p, tok)))
		if fnIndex < 0 || fnIndex >= len(g.funcSingleCallState) || tok > 0 && renvoTokIsKind(p, tok-1, renvoTokFunc) {
			continue
		}
		state := g.funcSingleCallState[fnIndex]
		if tok+1 >= renvoTokCount(p) || !renvoTokCharIs(p, tok+1, '(') || state == 2 {
			g.funcSingleCallState[fnIndex] = 1
		} else if state == 0 {
			g.funcSingleCallState[fnIndex] = 2
		}
	}
}

func renvoRecordSingleCallConstants(g *renvoLinearGen, ep *renvoExprParse, call *renvoExpr, fnIndex int) {
	renvoNonNil(g, ep, call)
	if !renvoIsSysVObject(g.c) || fnIndex < 0 || fnIndex >= len(g.meta.funcs) ||
		fnIndex >= len(g.funcSingleCallState) || g.continueDepth != 0 ||
		!renvoFunctionHasSingleDirectUse(g, fnIndex) {
		return
	}
	fn := &g.meta.funcs[fnIndex]
	if fn.linkStatic != 0 || fn.exportNameEnd > fn.exportNameStart || fn.receiverType != 0 ||
		fn.literalTok != 0 || call.argCount != fn.paramCount {
		return
	}
	for i := 0; i < fn.paramCount; i++ {
		param := fn.firstParam + i
		if param < 0 || param >= len(g.paramConstValid) {
			continue
		}
		arg := renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+i)
		oldFlow := g.constEvalFlow
		g.constEvalFlow = true
		value := renvoEvalConstExpr(g, ep, arg)
		g.constEvalFlow = oldFlow
		if value.ok && renvoConstExprSideEffectFree(g, ep, arg) {
			g.paramConstValues[param] = value.value
			g.paramConstValid[param] = true
		}
	}
}

func renvoFunctionHasSingleDirectUse(g *renvoLinearGen, fnIndex int) bool {
	return g.funcSingleCallState[fnIndex] == 2
}

func renvoEmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	renvoNonNil(g)
	renvoLinearMarkFunc(g, fnIndex)
	renvoEmitTargetCallWithWordCount(g, fnIndex, wordCount)
	renvoEmitPostCallPanicCheck(g)
}

func renvoEmitPointerCompositeLiteral(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	innerIndex := e.left
	elemType := renvoInferParsedExprType(g, ep, innerIndex)
	return renvoEmitTypedPointerCompositeLiteral(g, ep, innerIndex, elemType)
}

// The collection element type supplies both the omitted type and address-of
// operator in literals such as []*T{{field: value}}.
func renvoEmitTypedPointerCompositeLiteral(g *renvoLinearGen, ep *renvoExprParse, innerIndex int, elemType int) bool {
	inner := &ep.exprs[innerIndex]
	resolved := renvoResolveType(g.meta, elemType)
	renvoNonNil(resolved)
	if resolved.kind != renvoTypeStruct && resolved.kind != renvoTypeArray {
		return false
	}
	size := renvoTypeSize(g.meta, elemType)
	if size <= 0 {
		return false
	}
	a := &g.asm
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(a, sizeOffset, renvoAlignTo8(size))
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(a, addrOffset)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPrimaryImm(a, 0)
	for at := 0; at < size; at += renvoBackendValueSlotSize {
		renvoAsmStorePrimaryMemSecondaryDisp(a, at)
	}
	elemSize := renvoTypeSize(g.meta, resolved.elem)
	next := 0
	for i := 0; i < inner.argCount; i++ {
		field := ep.fields[inner.firstArg+i]
		fieldType := resolved.elem
		fieldOffset := 0
		if resolved.kind == renvoTypeStruct {
			fieldIndex := renvoCompositeStructFieldIndex(g, elemType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldType = g.meta.fields[fieldIndex].typ
			fieldOffset = g.meta.fields[fieldIndex].offset
		} else {
			at := next
			if field.key >= 0 {
				key := renvoEvalConstExpr(g, ep, field.key)
				if !key.ok {
					return false
				}
				at = key.value
			}
			fieldOffset = at * elemSize
			next = at + 1
		}
		if !renvoEmitCompositeFieldToMem(g, ep, field.expr, fieldType, addrOffset, fieldOffset) {
			return false
		}
	}
	renvoAsmLoadPrimaryStack(a, addrOffset)
	return true
}
func renvoEmitKernelLinkAddressCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	if (renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64) &&
		renvoEmitObjectKernelLinkAddress(g, ep, arg) {
		return true
	}
	return renvoEmitIntExpr(g, ep, arg)
}

func renvoEmitPointerDifferenceCall(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	e := &ep.exprs[idx]
	if e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprIdent {
		callee := &ep.exprs[e.left]
		if renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_diff_") {
			return renvoBoolInt(renvoEmitCPointerDifference(g, ep, e))
		}
	}
	return -1
}

func renvoEmitMachineIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	renvoNonNil(p)
	meta := g.meta
	renvoNonNil(meta)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprUnary || e.kind == renvoExprBinary || e.kind == renvoExprCall {
		constResult := renvoEvalConstExpr(g, ep, idx)
		resultType := renvoInferParsedExprType(g, ep, idx)
		result := renvoResolveType(meta, resultType)
		renvoNonNil(result)
		exactShift := e.kind == renvoExprBinary && (renvoTok2Is(p, e.tok, '<', '<') || renvoTok2Is(p, e.tok, '>', '>'))
		immediateKind := result.kind != renvoTypeByte && result.kind != renvoTypeInt8 && result.kind != renvoTypeInt16 && result.kind != renvoTypeInt32 && result.kind != renvoTypeUint16 && result.kind != renvoTypeUint32 && !renvoTypeKindIsFloat(result.kind)
		if constResult.ok && (!ep.hasFloat || exactShift) && immediateKind && renvoAsmWordConstantImmediate(a, result.kind, constResult.value) {
			return true
		}
	}
	if renvoFixedTarget == 0 {
		fast := renvoEmitWordExpressionPeephole(g, ep, idx)
		if fast >= 0 {
			return fast != 0
		}
	}
	if e.kind == renvoExprInt || e.kind == renvoExprFloat || e.kind == renvoExprIdent || e.kind == renvoExprChar || e.kind == renvoExprBool {
		return renvoEmitAtomExpr(g, ep, idx)
	}
	if e.kind == renvoExprCall {
		if result := renvoEmitWordCallIntrinsic(g, ep, idx); result >= 0 {
			return result != 0
		}
		return renvoEmitWordCallExpr(g, ep, idx)
	}
	if e.kind == renvoExprIndex {
		return renvoEmitIndexExpr(g, ep, idx)
	}
	if e.kind == renvoExprSelector {
		return renvoEmitScalarSelectorExpr(g, ep, idx)
	}
	if e.kind == renvoExprUnary {
		return renvoEmitUnaryExpr(g, ep, idx)
	}
	if e.kind == renvoExprBinary {
		return renvoEmitWordBinaryExpr(g, ep, idx)
	}
	return false
}

func renvoEmitIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if renvoExprIsNil(g.prog, e) {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	if renvoFixedTarget == 0 && renvoCanDirectScalarDeref(g) {
		direct := renvoEmitCDirectDeref(g, ep, idx)
		if direct >= 0 {
			return direct != 0
		}
	}
	if e.kind == renvoExprCall && e.argCount == 1 {
		callee := &ep.exprs[e.left]
		unsafePointer := callee.kind == renvoExprIdent &&
			renvoBytesEqualText(g.prog.src, callee.nameStart, callee.nameEnd, "Pointer") &&
			renvoFuncInfoFromCall(g, ep, e.left) < 0
		if callee.kind == renvoExprSelector {
			unsafePointer = renvoBytesEqualText(g.prog.src, callee.nameStart, callee.nameEnd, "Pointer") &&
				renvoExprIsIdentText(g.prog, ep, callee.left, "unsafe")
		}
		if unsafePointer {
			return renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
	}
	if e.kind == renvoExprIdent && renvoFindLocalIndex(g, e.nameStart, e.nameEnd) < 0 {
		symIndex := renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokConst)
		if symIndex >= 0 && g.meta.globals[symIndex].constValueOK == 0 {
			s := &g.meta.globals[symIndex]
			value := renvoNewExprParse()
			if !renvoParseExpressionOK(value, g.prog, s.initStart, s.initEnd) {
				return false
			}
			oldIota := g.constEvalIota
			oldIotaValid := g.constEvalIotaValid
			g.constEvalIota = s.iotaValue
			g.constEvalIotaValid = 1
			ok := renvoEmitIntExpr(g, value, len(value.exprs)-1)
			g.constEvalIota = oldIota
			g.constEvalIotaValid = oldIotaValid
			return ok
		}
	}
	if e.kind == renvoExprSelector {
		renvoLoadCompilerFixedTarget(g)
		value := renvoFixedTargetUnknown
		if g.fixedTargetState == 1 &&
			g.fixedTargetValue >= renvoTargetLinuxAmd64 && g.fixedTargetValue <= renvoTargetNetBSDAmd64 {
			nameSize := e.nameEnd - e.nameStart
			if nameSize >= 5 && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameStart+5, "renvo") {
				if nameSize == 15 {
					value = int(targetArchTable[g.fixedTargetValue])
				} else if nameSize == 13 {
					value = int(targetOSTable[g.fixedTargetValue])
				} else if nameSize == 11 {
					value = g.fixedTargetValue
				} else if nameSize == 18 {
					value = int(renvoTargetIntBitsTable[g.fixedTargetValue]) / 8
				}
			}
		}
		if value != renvoFixedTargetUnknown {
			renvoAsmPrimaryImm(&g.asm, value)
			return true
		}
	}
	if g.c.renvoNativeIntSize == 4 && renvoEmitWideCompareExpr(g, ep, idx) {
		return true
	}
	if e.kind == renvoExprBinary && renvoBinaryComparesInterface(g, ep, e) {
		return renvoEmitInterfaceCompare(g, ep, e)
	}
	if e.kind == renvoExprCall {
		if renvoFixedTarget == 0 {
			if renvoExprIsIdentText(g.prog, ep, e.left, "renvo_runtime_CallJIT") {
				return renvoEmitJITCall(g, ep, idx)
			}
		}
		callee := renvoResolvedNumericCalleeCode(g, ep, e.left)
		if callee == renvoIdentRecover {
			return renvoEmitBuiltinRecover(g, ep, idx)
		}
		if callee == renvoIdentReal || callee == renvoIdentImag {
			return renvoEmitComplexComponentPrimary(g, ep, idx, callee == renvoIdentImag)
		}
		valueType := renvoInferParsedExprType(g, ep, idx)
		if renvoResolveType(g.meta, valueType).kind == renvoTypeInterface {
			offset := renvoAddUnnamedLocal(g, valueType)
			if !renvoEmitInterfaceAssignToLocal(g, ep, idx, offset) {
				return false
			}
			renvoAsmLoadPrimaryStack(&g.asm, offset)
			return true
		}
	}
	if e.kind == renvoExprAssert {
		asserted := renvoInferParsedExprType(g, ep, idx)
		if asserted == 0 || renvoTypeSize(g.meta, asserted) > renvoBackendValueSlotSize {
			return false
		}
		offset := renvoAddUnnamedLocal(g, asserted)
		if !renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true) {
			return false
		}
		renvoAsmLoadPrimaryStack(&g.asm, offset)
		return true
	}
	if e.kind == renvoExprFunc {
		return renvoEmitClosureValuePrimary(g, e.tok)
	}
	if e.kind == renvoExprComposite && e.argCount == 0 {
		typ := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(typ)
		if typ.kind == renvoTypeFunc {
			renvoAsmPrimaryImm(&g.asm, 0)
			return true
		}
	}
	if e.kind == renvoExprSelector {
		if renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left)).kind == renvoTypeInterface {
			offset := renvoAddUnnamedLocal(g, renvoTypeInt)
			if renvoEmitInterfaceMethodValue(g, ep, e, nil, offset) == 0 {
				return false
			}
			renvoAsmLoadPrimaryStack(&g.asm, offset)
			return true
		}
		fnIndex, expression := renvoMethodSelectorInfo(g, ep, idx)
		if fnIndex >= 0 {
			return renvoEmitMethodSelectorValuePrimary(g, ep, idx, fnIndex, expression)
		}
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '&') && e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprComposite {
		return renvoEmitPointerCompositeLiteral(g, ep, idx)
	}
	return renvoEmitMachineIntExpr(g, ep, idx)
}

func renvoExprHasUnsignedIntType(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	// Signedness is a source-type property, including on fixed 32-bit targets.
	// Constant folding must not turn an unsigned shift into an arithmetic one
	// just because a particular machine emitter is selected.
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt || e.kind == renvoExprChar || e.kind == renvoExprBool {
		return false
	}
	if e.inferred == 0 && e.kind == renvoExprUnary {
		return renvoExprHasUnsignedIntType(g, ep, e.left)
	}
	resolved := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	kind := resolved.kind
	return kind == renvoTypeByte || kind >= renvoTypeUint16 && kind <= renvoTypeUint64
}

func renvoEmitWideScalarToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int, sourceKind int, destKind int) bool {
	if !renvoSplitWordLoweringEnabled(&g.asm) {
		return false
	}
	renvoNonNil(g, ep)
	ok := false
	if sourceKind == renvoTypeFloat64 {
		ok = renvoEmitScalarExprForKind(g, ep, idx, destKind)
	} else {
		ok = renvoEmitIntExpr(g, ep, idx)
	}
	if !ok {
		return false
	}
	if sourceKind != renvoTypeFloat64 {
		renvoAsmNormalizePrimaryForKind(&g.asm, sourceKind)
	}
	renvoAsmStorePrimaryStack(&g.asm, offset)
	if sourceKind == renvoTypeInt || sourceKind == renvoTypeInt8 || sourceKind == renvoTypeInt16 || sourceKind == renvoTypeInt32 || sourceKind == renvoTypeInt64 || sourceKind == renvoTypeFloat64 && destKind == renvoTypeInt64 {
		renvoAsmSarPrimaryImm(&g.asm, 31)
	} else {
		renvoAsmPrimaryImm(&g.asm, 0)
	}
	renvoAsmStorePrimaryStack(&g.asm, offset-g.c.renvoNativeIntSize)
	return true
}

func renvoStoreFloat64BitsStack(a *renvoAsm, offset int, bits uint64) {
	renvoAsmStoreStackImm(a, offset, int(uint32(bits)))
	renvoAsmStoreStackImm(a, offset-4, int(uint32(bits>>32)))
}

func renvoRuntimeIntrinsicForCall(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) int {
	fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return 0
	}
	fn := &g.meta.funcs[fnIndex]
	return renvoRuntimeIntrinsicID(g.prog.src, fn.nameStart, fn.nameEnd)
}

func renvoEmitStackFloat64ExprToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	if !renvoUsesStackIEEEFloat(&g.asm) {
		return false
	}
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if e.kind == renvoExprFloat {
		renvoStoreFloat64BitsStack(&g.asm, offset, renvoParseFloatTokenBits(g.prog, e.tok, 52, 11, 1023))
		return true
	}
	if e.kind == renvoExprInt || e.kind == renvoExprChar {
		temp := renvoAddUnnamedLocal(g, renvoTypeInt64)
		if e.kind == renvoExprInt {
			low := renvoParseIntToken(g.prog, e.tok)
			renvoAsmStoreStackImm(&g.asm, temp, low)
			renvoAsmStoreStackImm(&g.asm, temp-4, g.prog.parsedIntHigh)
		} else {
			value := renvoParseCharToken(g.prog, e.tok)
			renvoAsmStoreStackImm(&g.asm, temp, value)
			renvoAsmStoreStackImm(&g.asm, temp-4, value>>31)
		}
		renvo32IEEEIntToFloatStack(g, temp, 8, 8, true)
		renvoEmitCopyStackToStack(g, temp, offset, 8)
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			renvoEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, 8)
			return true
		}
		symIndex := renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokConst)
		if symIndex >= 0 {
			s := &g.meta.globals[symIndex]
			value := renvoNewExprParse()
			root := renvoParseExpressionRoot(value, g.prog, s.initStart, s.initEnd)
			return root >= 0 && renvoEmitStackFloat64ExprToLocal(g, value, root, offset)
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			for at := 0; at < 8; at += 4 {
				renvoAsmCopyBssToStackSlot(&g.asm, globalOffset+at, offset-at)
			}
			return true
		}
		return false
	}
	if e.kind == renvoExprSelector || e.kind == renvoExprIndex || e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyMemSecondaryToStack(g, offset, 8)
		return true
	}
	if e.kind == renvoExprUnary {
		if !renvoTokCharIs(g.prog, e.tok, '+') && !renvoTokCharIs(g.prog, e.tok, '-') {
			return false
		}
		if !renvoEmitStackFloat64ExprToLocal(g, ep, e.left, offset) {
			return false
		}
		if renvoTokCharIs(g.prog, e.tok, '-') && !renvoExprIsUntypedZeroFloat(g.prog, ep, e.left) {
			renvo32IEEENegateStack(g, offset, 8)
		}
		return true
	}
	if e.kind == renvoExprBinary {
		// A constant shift remains integer-valued when used in a floating
		// expression. Evaluate it before converting, rather than issuing an
		// unsupported floating-point shift operation on 32-bit targets.
		if renvoTok2Is(g.prog, e.tok, '<', '<') || renvoTok2Is(g.prog, e.tok, '>', '>') {
			constant := renvoEvalConstExpr(g, ep, idx)
			if constant.ok {
				unsigned := renvoExprHasUnsignedIntType(g, ep, idx)
				high := constant.value >> 32
				if g.prog.compilerInt32 && unsigned {
					high = 0
				}
				temp := renvoAddUnnamedLocal(g, renvoTypeInt64)
				renvoAsmStoreStackImm(&g.asm, temp, constant.value)
				renvoAsmStoreStackImm(&g.asm, temp-4, high)
				renvo32IEEEIntToFloatStack(g, temp, 8, 8, !unsigned)
				renvoEmitCopyStackToStack(g, temp, offset, 8)
				return true
			}
		}
		kind := renvoBinaryFloatKind(g, ep, e)
		if kind == renvoTypeFloat32 {
			temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
			if !renvoEmitTypedAssign(g, ep, idx, temp) {
				return false
			}
			renvo32IEEEConvertFloatStack(g, offset, temp, 4, 8)
			return true
		}
		left := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		right := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		if !renvoEmitStackFloat64ExprToLocal(g, ep, e.left, left) || !renvoEmitStackFloat64ExprToLocal(g, ep, e.right, right) {
			return false
		}
		c0, _, comparison := renvoFloatComparisonChars(g.prog, e.tok)
		return !comparison && renvo32IEEEBinaryStack(g, offset, left, right, c0, 8)
	}
	if e.kind == renvoExprCall {
		callee := renvoResolvedNumericCalleeCode(g, ep, e.left)
		if e.argCount == 1 && (callee == renvoIdentReal || callee == renvoIdentImag) {
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			argType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
			if argType.kind == renvoTypeComplex {
				temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeComplex)
				if !renvoEmitStackComplex128ToLocal(g, ep, arg, temp) {
					return false
				}
				if callee == renvoIdentImag {
					temp -= 8
				}
				renvoEmitCopyStackToStack(g, temp, offset, 8)
				return true
			}
		}
		if e.argCount == 1 {
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			intrinsic := renvoRuntimeIntrinsicForCall(g, ep, e)
			if intrinsic == 22 {
				return renvoEmitWideExprToLocal(g, ep, arg, offset, renvoTypeUint64)
			}
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				// Untyped integer constants retain arbitrary precision until the
				// floating conversion. Materialize their full parsed words rather
				// than first narrowing them to the target's default int.
				if ep.exprs[arg].kind == renvoExprInt || ep.exprs[arg].kind == renvoExprChar {
					return renvoEmitStackFloat64ExprToLocal(g, ep, arg, offset)
				}
				source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
				if source.kind == renvoTypeFloat64 {
					return renvoEmitStackFloat64ExprToLocal(g, ep, arg, offset)
				}
				if source.kind == renvoTypeFloat32 {
					temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
					if !renvoEmitTypedAssign(g, ep, arg, temp) {
						return false
					}
					renvo32IEEEConvertFloatStack(g, offset, temp, 4, 8)
					return true
				}
				tempType := renvoTypeInt64
				if !renvoTypeKindIsWideInt(source.kind) {
					tempType = renvoTypeInt32
				}
				temp := renvoAddUnnamedLocal(g, tempType)
				if renvoTypeKindIsWideInt(source.kind) {
					if !renvoEmitWideExprToLocal(g, ep, arg, temp, source.kind) {
						return false
					}
				} else {
					if !renvoEmitScalarExprForKind(g, ep, arg, source.kind) {
						return false
					}
					renvoAsmStorePrimaryStack(&g.asm, temp)
				}
				intSize := 4
				if renvoTypeKindIsWideInt(source.kind) {
					intSize = 8
				}
				renvo32IEEEIntToFloatStack(g, temp, intSize, 8, !renvoTypeKindIsUnsignedInteger(source.kind))
				renvoEmitCopyStackToStack(g, temp, offset, 8)
				return true
			}
		}
		return renvoEmitStructCallToLocal(g, ep, idx, renvoInferParsedExprType(g, ep, idx), offset)
	}
	return false
}

func renvoEmitWideExprToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int, destKind int) bool {
	if !renvoSplitWordLoweringEnabled(&g.asm) && (renvoPreparedBackendActive == 0 || renvoRTGPreparedIEEEFloat == 0) {
		return false
	}
	renvoNonNil(g, ep)
	if destKind == renvoTypeFloat64 {
		if renvoUsesStackIEEEFloat(&g.asm) {
			return renvoEmitStackFloat64ExprToLocal(g, ep, idx, offset)
		}
	}
	renvoRefreshCapturedExpr(g, ep, idx)
	e := &ep.exprs[idx]
	if e.kind == renvoExprCall && e.argCount == 1 && renvoRuntimeIntrinsicForCall(g, ep, e) == 21 {
		return renvoEmitStackFloat64ExprToLocal(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), offset)
	}
	if e.kind == renvoExprInt {
		low := renvoParseIntToken(g.prog, e.tok)
		renvoAsmStoreStackImm(&g.asm, offset, low)
		renvoAsmStoreStackImm(&g.asm, offset-g.c.renvoNativeIntSize, g.prog.parsedIntHigh)
		return true
	}
	if e.kind == renvoExprChar {
		return renvoEmitWideScalarToLocal(g, ep, idx, offset, renvoTypeInt, destKind)
	}
	if e.kind == renvoExprIdent && renvoFindLocalIndex(g, e.nameStart, e.nameEnd) < 0 {
		// A 32-bit compiler cannot represent a wide constant in constResult.
		// Emit its parsed initializer into both destination words instead.
		if g.prog.compilerInt32 {
			symbol := renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokConst)
			if symbol >= 0 {
				s := &g.meta.globals[symbol]
				constantExpr := renvoNewExprParse()
				root := renvoParseExpressionRoot(constantExpr, g.prog, s.initStart, s.initEnd)
				if root < 0 {
					return false
				}
				oldIota := g.constEvalIota
				oldValid := g.constEvalIotaValid
				g.constEvalIota = s.iotaValue
				g.constEvalIotaValid = 1
				ok := renvoEmitWideExprToLocal(g, constantExpr, root, offset, destKind)
				g.constEvalIota = oldIota
				g.constEvalIotaValid = oldValid
				return ok
			}
		}
		constant := renvoEvalConstExpr(g, ep, idx)
		if constant.ok {
			renvoAsmStoreStackImm(&g.asm, offset, constant.value)
			high := constant.value >> 32
			if g.prog.compilerInt32 && destKind != renvoTypeInt64 {
				high = 0
			}
			renvoAsmStoreStackImm(&g.asm, offset-g.c.renvoNativeIntSize, high)
			return true
		}
	}
	if e.kind == renvoExprCall {
		if e.left >= 0 && e.left < len(ep.exprs) && ep.exprs[e.left].kind == renvoExprIdent {
			callee := &ep.exprs[e.left]
			if renvoBytesPrefixText(g.prog.src, callee.nameStart, callee.nameEnd, "__c_pointer_diff_") {
				if !renvoEmitCPointerDifference(g, ep, e) {
					return false
				}
				renvoAsmStorePrimaryStack(&g.asm, offset)
				renvoAsmSarPrimaryImm(&g.asm, 31)
				renvoAsmStorePrimaryStack(&g.asm, offset-g.c.renvoNativeIntSize)
				return true
			}
		}
		conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 {
			arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
			source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, arg))
			renvoNonNil(source)
			if source.kind == renvoTypeFloat64 && renvoTypeKindIsWideInt(destKind) {
				if renvoUsesStackIEEEFloat(&g.asm) {
					floatTemp := renvoAddUnnamedLocal(g, renvoTypeFloat64)
					if !renvoEmitStackFloat64ExprToLocal(g, ep, arg, floatTemp) {
						return false
					}
					renvo32IEEEFloatToIntStack(g, offset, floatTemp, 8, 8, !renvoTypeKindIsUnsignedInteger(destKind))
					return true
				}
			}
			argKind := ep.exprs[arg].kind
			wideSource := g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideInt(source.kind)
			untypedInteger := (argKind == renvoExprInt || argKind == renvoExprUnary || argKind == renvoExprBinary) && renvoExprIsUntypedInteger(ep, arg)
			if wideSource || untypedInteger {
				return renvoEmitWideExprToLocal(g, ep, arg, offset, destKind)
			}
			return renvoEmitWideScalarToLocal(g, ep, arg, offset, source.kind, destKind)
		}
		return renvoEmitStructCallToLocal(g, ep, idx, renvoInferParsedExprType(g, ep, idx), offset)
	}
	if e.kind == renvoExprAssert {
		return renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true)
	}
	if renvoFixedTarget == 0 && renvoEmitWideIdentToLocal(g, e, offset) {
		return true
	}
	if e.kind == renvoExprIdent || e.kind == renvoExprSelector || e.kind == renvoExprIndex || e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyMemSecondaryToStack(g, offset, renvoBackendValueSlotSize)
		return true
	}
	if e.kind == renvoExprUnary {
		if renvoTokCharIs(g.prog, e.tok, '+') {
			return renvoEmitWideExprToLocal(g, ep, e.left, offset, destKind)
		}
		temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		if !renvoEmitWideExprToLocal(g, ep, e.left, temp, destKind) {
			return false
		}
		return renvoEmitWideUnaryStack(g, offset, temp, e.tok)
	}
	if e.kind == renvoExprBinary {
		signed := destKind == renvoTypeInt64
		left := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		right := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		if !renvoEmitWideExprToLocal(g, ep, e.left, left, destKind) {
			return false
		}
		shift := renvoTok2Is(g.prog, e.tok, '<', '<') || renvoTok2Is(g.prog, e.tok, '>', '>')
		if shift {
			if !renvoEmitIntExpr(g, ep, e.right) {
				return false
			}
			renvoAsmStorePrimaryStack(&g.asm, right)
			renvoAsmStoreStackImm(&g.asm, right-g.c.renvoNativeIntSize, 0)
		} else if !renvoEmitWideExprToLocal(g, ep, e.right, right, destKind) {
			return false
		}
		return renvoEmitWideBinaryValue(g, offset, left, right, e.tok, signed)
	}
	return false
}

func renvoEmitWideCompareExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	if !renvoSplitWordLoweringEnabled(&g.asm) && (renvoPreparedBackendActive == 0 || renvoRTGPreparedIEEEFloat == 0) {
		return false
	}
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprBinary {
		return false
	}
	start := int(renvoTokStart(g.prog, e.tok))
	end := int(renvoTokEnd(g.prog, e.tok))
	c0 := renvo_runtime_UnsafeByteAt(g.prog.src, start)
	var c1 byte
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(g.prog.src, start+1)
	}
	if !renvoIsComparisonChars(c0, c1) {
		return false
	}
	leftKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left)).kind
	rightKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right)).kind
	if renvoTypeKindIsFloat(leftKind) || renvoTypeKindIsFloat(rightKind) {
		if !renvoUsesStackIEEEFloat(&g.asm) {
			return false
		}
		kind := renvoBinaryFloatKind(g, ep, e)
		typ := renvoTypeFloat64
		if kind == renvoTypeFloat32 {
			typ = renvoBuiltinTypeFloat32
		}
		left := renvoAddUnnamedLocal(g, typ)
		right := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitWideFloatBinaryOperands(g, ep, e, kind, left, right) {
			return false
		}
		return renvoEmit32IEEECompareStack(g, left, right, kind, c0, c1)
	}
	if !(g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideInt(leftKind)) && !(g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideInt(rightKind)) {
		return false
	}
	if !renvoTypeKindIsWideInt(leftKind) {
		leftKind = rightKind
	}
	left := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	right := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	if !renvoEmitWideExprToLocal(g, ep, e.left, left, leftKind) || !renvoEmitWideExprToLocal(g, ep, e.right, right, leftKind) {
		return false
	}
	return renvoEmitWideCompareValue(g, left, right, e.tok, leftKind == renvoTypeInt64)
}

func renvoEmitWideUnaryStack(g *renvoLinearGen, dest int, source int, tok int) bool {
	if !renvoSplitWordLoweringEnabled(&g.asm) {
		return false
	}
	renvoNonNil(g)
	other := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	if renvoTokCharIs(g.prog, tok, '^') {
		renvoAsmStoreStackImm(&g.asm, other, -1)
		renvoAsmStoreStackImm(&g.asm, other-g.c.renvoNativeIntSize, -1)
	} else if renvoTokCharIs(g.prog, tok, '-') {
		renvoZeroLocalAtOffset(g, other)
	} else {
		return false
	}
	return renvoEmitWideBinaryValue(g, dest, other, source, tok, true)
}

func renvoEmitPortableWideCompareStack(g *renvoLinearGen, left int, right int, tok int, signed bool) bool {
	if !renvoSplitWordLoweringEnabled(&g.asm) {
		return false
	}
	renvoNonNil(g)
	p := g.prog
	start := int(renvoTokStart(p, tok))
	end := int(renvoTokEnd(p, tok))
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	var c1 byte
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	equality := false
	if c1 == '=' {
		equality = c0 == '='
		if c0 == '!' {
			equality = true
		}
	}
	if equality {
		notEqual := renvoAsmNewLabel(&g.asm)
		done := renvoAsmNewLabel(&g.asm)
		renvoEmitNativeCompareStack(g, left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x94)
		renvoAsmJzPrimary(&g.asm, notEqual)
		renvoEmitNativeCompareStack(g, left, right, 0x94)
		renvoAsmJmpMarkLabel(&g.asm, done, notEqual)
		renvoAsmPrimaryImm(&g.asm, 0)
		renvoAsmMarkLabel(&g.asm, done)
		if c0 == '!' {
			renvoAsmBoolNotPrimary(&g.asm)
		}
		return true
	}
	greater := c0 == '>'
	inclusive := false
	if c1 == '=' {
		inclusive = c0 == '<'
		if c0 == '>' {
			inclusive = true
		}
	}
	if greater != inclusive {
		left, right = right, left
	}
	renvoEmitWideLessStack(g, left, right, signed)
	if inclusive {
		renvoAsmBoolNotPrimary(&g.asm)
	}
	return true
}

func renvoEmitRTGWideStack(g *renvoLinearGen, dest int, left int, right int, mode int) bool {
	renvoNonNil(g)
	if mode == 0 {
		renvoEmitWideAddStack(g, dest, left, right)
		return true
	}
	if mode == 1 {
		renvoEmitWideSubStack(g, dest, left, right)
		return true
	}
	if mode == 2 {
		renvoEmitWideMulStack(g, dest, left, right)
		return true
	}
	if mode >= 3 && mode <= 6 {
		renvoEmitWideDivStack(g, dest, left, right, mode >= 5, mode == 4 || mode == 6)
		return true
	}
	if mode >= 7 && mode <= 9 {
		renvoEmitWideShiftStack(g, dest, left, right, mode != 7, mode == 9)
		return true
	}
	if mode >= 10 && mode <= 13 {
		renvoEmitRTGWideBitwiseStack(g, dest, left, right, mode)
		return true
	}
	if mode >= 14 && mode <= 23 {
		renvoEmitRTGWideCompareStack(g, left, right, mode)
		return true
	}
	return false
}

func renvoEmitRTGWideBitwiseStack(g *renvoLinearGen, dest int, left int, right int, mode int) {
	renvoNonNil(g)
	for word := 0; word < 2; word++ {
		leftWord := left - word*g.c.renvoNativeIntSize
		rightWord := right - word*g.c.renvoNativeIntSize
		destWord := dest - word*g.c.renvoNativeIntSize
		renvoAsmLoadPrimaryStack(&g.asm, leftWord)
		renvoAsmLoadTertiaryStack(&g.asm, rightWord)
		if mode == 10 {
			renvoRTGDirectBitAnd(&g.asm, renvoRTGPrimary, renvoRTGTertiary)
		} else if mode == 11 {
			renvoRTGDirectBitOr(&g.asm, renvoRTGPrimary, renvoRTGTertiary)
		} else if mode == 12 {
			renvoRTGDirectBitXor(&g.asm, renvoRTGPrimary, renvoRTGTertiary)
		} else {
			renvoRTGDirectMoveImmediate(&g.asm, renvoRTGScratch, -1)
			renvoRTGDirectBitXor(&g.asm, renvoRTGTertiary, renvoRTGScratch)
			renvoRTGDirectBitAnd(&g.asm, renvoRTGPrimary, renvoRTGTertiary)
		}
		renvoAsmStorePrimaryStack(&g.asm, destWord)
	}
}

func renvoEmitRTGWideCompareStack(g *renvoLinearGen, left int, right int, mode int) {
	renvoNonNil(g)
	if mode == 14 || mode == 15 {
		different := renvoAsmNewLabel(&g.asm)
		done := renvoAsmNewLabel(&g.asm)
		renvoEmitNativeCompareStack(g,
			left-g.c.renvoNativeIntSize, right-g.c.renvoNativeIntSize, 0x95)
		renvoAsmJnzPrimary(&g.asm, different)
		renvoEmitNativeCompareStack(g, left, right, 0x95)
		renvoAsmJmpLabel(&g.asm, done)
		renvoAsmMarkLabel(&g.asm, different)
		renvoAsmPrimaryImm(&g.asm, 1)
		renvoAsmMarkLabel(&g.asm, done)
		if mode == 14 {
			renvoRTGAsmBoolNot(&g.asm)
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
		renvoRTGAsmBoolNot(&g.asm)
	}
}

func renvoWideBinaryMode(g *renvoLinearGen, tok int, signed bool) int {
	renvoNonNil(g)
	start := int(renvoTokStart(g.prog, tok))
	end := int(renvoTokEnd(g.prog, tok))
	c0 := renvo_runtime_UnsafeByteAt(g.prog.src, start)
	table := "\xff\x04\x0a\xff\xff\x0f\xff\xff\x01\xff\x03\xff\xff\xff\x00\x02\xff\xff\xff\x0b\xff\x0c\xff\xff\xff\x12\x0e\x10\xff\xff\xff\xff"
	mode := int(table[(int(c0)^int(c0)>>3)&31])
	if end-start == 3 && renvo_runtime_UnsafeByteAt(g.prog.src, start+2) == '=' {
		c1 := renvo_runtime_UnsafeByteAt(g.prog.src, start+1)
		if c0 == '<' && c1 == '<' {
			mode = 7
		} else if c0 == '>' && c1 == '>' {
			mode = 8
		}
	} else if end-start == 2 {
		c1 := renvo_runtime_UnsafeByteAt(g.prog.src, start+1)
		if c1 == '=' {
			if mode >= 16 {
				mode++
			}
		} else if c0 == '<' && c1 == '<' {
			mode = 7
		} else if c0 == '>' && c1 == '>' {
			mode = 8
		} else {
			mode = 13
		}
	}
	if signed {
		if mode == 3 || mode == 4 {
			mode += 2
		}
		if mode == 8 {
			mode++
		}
	} else if mode >= 16 {
		mode += 4
	}
	return mode
}

func renvoEmitWideCompoundLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int, kind int, tok int) bool {
	if kind == renvoTypeFloat64 && renvoUsesStackIEEEFloat(&g.asm) {
		right := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		if !renvoEmitStackFloat64ExprToLocal(g, ep, idx, right) {
			return false
		}
		c0, _, comparison := renvoFloatComparisonChars(g.prog, tok)
		return !comparison && renvo32IEEEBinaryStack(g, offset, offset, right, c0, 8)
	}
	left := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	right := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	result := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	renvoEmitCopyStackToStack(g, offset, left, renvoBackendValueSlotSize)
	mode := renvoWideBinaryMode(g, tok, kind == renvoTypeInt64)
	if mode >= 7 && mode <= 9 {
		if !renvoEmitIntExpr(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, right)
		renvoAsmStoreStackImm(&g.asm, right-g.c.renvoNativeIntSize, 0)
	} else if !renvoEmitWideExprToLocal(g, ep, idx, right, kind) {
		return false
	}
	if !renvoEmitWideBinaryValue(g, result, left, right, tok, kind == renvoTypeInt64) {
		return false
	}
	renvoEmitCopyStackToStack(g, result, offset, renvoBackendValueSlotSize)
	return true
}

func renvoEmitNativeCompareStack(g *renvoLinearGen, left int, right int, setcc int) {
	if !renvoSplitWordLoweringEnabled(&g.asm) {
		return
	}
	renvoNonNil(g)
	renvoAsmLoadPrimaryStack(&g.asm, right)
	renvoAsmLoadTertiaryStack(&g.asm, left)
	renvoAsmCmpTertiaryPrimarySet(&g.asm, setcc)
}

func renvoEmitWideDivStack(g *renvoLinearGen, dest int, left int, right int, signed bool, remainderResult bool) {
	if !renvoSplitWordLoweringEnabled(&g.asm) {
		return
	}
	renvoNonNil(g)
	// Preserve the existing runtime-fault path for a zero divisor.
	nonzero := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
	renvoAsmJnzPrimary(&g.asm, nonzero)
	renvoAsmLoadPrimaryStack(&g.asm, right)
	renvoEmitRuntimeNonNilPrimary(g)
	renvoAsmMarkLabel(&g.asm, nonzero)
	dividend := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	divisor := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	renvoEmitCopyStackToStack(g, left, dividend, renvoBackendValueSlotSize)
	renvoEmitCopyStackToStack(g, right, divisor, renvoBackendValueSlotSize)
	leftNegative := renvoAddUnnamedLocal(g, renvoTypeInt)
	rightNegative := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, leftNegative, 0)
	renvoAsmStoreStackImm(&g.asm, rightNegative, 0)
	if signed {
		zero := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreStackImm(&g.asm, zero, 0)
		renvoEmitNativeCompareStack(g, dividend-g.c.renvoNativeIntSize, zero, 0x9c)
		renvoAsmStorePrimaryStack(&g.asm, leftNegative)
		leftReady := renvoAsmNewLabel(&g.asm)
		renvoAsmJzPrimary(&g.asm, leftReady)
		renvoEmitWideNegateInPlace(g, dividend)
		renvoAsmMarkLabel(&g.asm, leftReady)
		renvoEmitNativeCompareStack(g, divisor-g.c.renvoNativeIntSize, zero, 0x9c)
		renvoAsmStorePrimaryStack(&g.asm, rightNegative)
		rightReady := renvoAsmNewLabel(&g.asm)
		renvoAsmJzPrimary(&g.asm, rightReady)
		renvoEmitWideNegateInPlace(g, divisor)
		renvoAsmMarkLabel(&g.asm, rightReady)
	}
	quotient := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	remainder := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
	renvoEmitWideUnsignedDivStack(g, quotient, remainder, dividend, divisor)
	if remainderResult {
		renvoEmitCopyStackToStack(g, remainder, dest, renvoBackendValueSlotSize)
		if signed {
			done := renvoAsmNewLabel(&g.asm)
			renvoAsmLoadPrimaryStack(&g.asm, leftNegative)
			renvoAsmJzPrimary(&g.asm, done)
			renvoEmitWideNegateInPlace(g, dest)
			renvoAsmMarkLabel(&g.asm, done)
		}
		return
	}
	renvoEmitCopyStackToStack(g, quotient, dest, renvoBackendValueSlotSize)
	if signed {
		sameSign := renvoAsmNewLabel(&g.asm)
		renvoEmitNativeCompareStack(g, leftNegative, rightNegative, 0x94)
		renvoAsmJnzPrimary(&g.asm, sameSign)
		renvoEmitWideNegateInPlace(g, dest)
		renvoAsmMarkLabel(&g.asm, sameSign)
	}
}

func renvoEmitStackComplex128ToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	if !renvoUsesStackIEEEFloat(&g.asm) {
		return false
	}
	e := &ep.exprs[idx]
	if (e.kind == renvoExprInt || e.kind == renvoExprFloat) && renvoExprTokenIsImaginary(g.prog, e.tok) {
		renvoAsmStoreStackImm(&g.asm, offset, 0)
		renvoAsmStoreStackImm(&g.asm, offset-4, 0)
		renvoStoreFloat64BitsStack(&g.asm, offset-8, renvoParseFloatTokenBits(g.prog, e.tok, 52, 11, 1023))
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			renvoEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, 16)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			for at := 0; at < 16; at += 4 {
				renvoAsmCopyBssToStackSlot(&g.asm, globalOffset+at, offset-at)
			}
			return true
		}
	}
	if e.kind == renvoExprIndex || e.kind == renvoExprSelector || e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyMemSecondaryToStack(g, offset, 16)
		return true
	}
	if e.kind == renvoExprCall && renvoResolvedNumericCalleeCode(g, ep, e.left) == renvoIdentComplex {
		if e.argCount != 2 || !renvoEmitStackFloat64ExprToLocal(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), offset) ||
			!renvoEmitStackFloat64ExprToLocal(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1), offset-8) {
			return false
		}
		return true
	}
	if e.kind == renvoExprCall {
		return renvoEmitStructCallToLocal(g, ep, idx, renvoInferParsedExprType(g, ep, idx), offset)
	}
	if e.kind == renvoExprUnary && (renvoTokCharIs(g.prog, e.tok, '+') || renvoTokCharIs(g.prog, e.tok, '-')) {
		if !renvoEmitStackComplex128ToLocal(g, ep, e.left, offset) {
			return false
		}
		if renvoTokCharIs(g.prog, e.tok, '-') {
			renvo32IEEENegateStack(g, offset, 8)
			renvo32IEEENegateStack(g, offset-8, 8)
		}
		return true
	}
	if e.kind == renvoExprBinary {
		left := renvoAddUnnamedLocal(g, renvoBuiltinTypeComplex)
		right := renvoAddUnnamedLocal(g, renvoBuiltinTypeComplex)
		if !renvoEmitStackComplex128ToLocal(g, ep, e.left, left) || !renvoEmitStackComplex128ToLocal(g, ep, e.right, right) {
			return false
		}
		op := renvo_runtime_UnsafeByteAt(g.prog.src, renvoTokStart(g.prog, e.tok))
		if op == '+' || op == '-' {
			return renvo32IEEEBinaryStack(g, offset, left, right, op, 8) &&
				renvo32IEEEBinaryStack(g, offset-8, left-8, right-8, op, 8)
		}
		if op != '*' && op != '/' {
			return false
		}
		first := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		second := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		if op == '*' {
			renvo32IEEEBinaryStack(g, first, left, right, '*', 8)
			renvo32IEEEBinaryStack(g, second, left-8, right-8, '*', 8)
			renvo32IEEEBinaryStack(g, offset, first, second, '-', 8)
			renvo32IEEEBinaryStack(g, first, left-8, right, '*', 8)
			renvo32IEEEBinaryStack(g, second, left, right-8, '*', 8)
			return renvo32IEEEBinaryStack(g, offset-8, first, second, '+', 8)
		}
		denominator := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		renvo32IEEEBinaryStack(g, first, right, right, '*', 8)
		renvo32IEEEBinaryStack(g, second, right-8, right-8, '*', 8)
		renvo32IEEEBinaryStack(g, denominator, first, second, '+', 8)
		renvo32IEEEBinaryStack(g, first, left, right, '*', 8)
		renvo32IEEEBinaryStack(g, second, left-8, right-8, '*', 8)
		renvo32IEEEBinaryStack(g, first, first, second, '+', 8)
		renvo32IEEEBinaryStack(g, offset, first, denominator, '/', 8)
		renvo32IEEEBinaryStack(g, first, left-8, right, '*', 8)
		renvo32IEEEBinaryStack(g, second, left, right-8, '*', 8)
		renvo32IEEEBinaryStack(g, first, first, second, '-', 8)
		return renvo32IEEEBinaryStack(g, offset-8, first, denominator, '/', 8)
	}
	resolved := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	if renvoTypeKindIsScalarValue(resolved.kind) {
		if !renvoEmitStackFloat64ExprToLocal(g, ep, idx, offset) {
			return false
		}
		renvoAsmStoreStackImm(&g.asm, offset-8, 0)
		renvoAsmStoreStackImm(&g.asm, offset-12, 0)
		return true
	}
	return false
}

func renvoEmitComplexComponentPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int, imaginary bool) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 || !renvoEmitComplexValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	if imaginary {
		renvoAsmPushSecondary(&g.asm)
		renvoAsmPopPrimary(&g.asm)
	}
	return true
}

func renvoComplexSecondaryStackOffset(g *renvoLinearGen, typ int, offset int) int {
	if g.c.renvoNativeIntSize == 4 && renvoResolveType(g.meta, typ).kind == renvoTypeComplex64 {
		return offset - 4
	}
	return offset - renvoBackendValueSlotSize
}

func renvoEmitComplexValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	complexKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind
	return renvoEmitComplexValueRegsForKind(g, ep, idx, complexKind)
}

func renvoEmitComplexValueRegsForKind(g *renvoLinearGen, ep *renvoExprParse, idx int, complexKind int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	if !renvoTypeKindIsComplex(complexKind) {
		complexKind = renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind
	}
	e := &ep.exprs[idx]
	componentKind := renvoTypeFloat64
	if complexKind == renvoTypeComplex64 {
		componentKind = renvoTypeFloat32
	}
	if e.kind == renvoExprAssert {
		typ := renvoInferParsedExprType(g, ep, idx)
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitTypedAssign(g, ep, idx, offset) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, renvoComplexSecondaryStackOffset(g, typ, offset))
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || !renvoTypeKindIsComplex(renvoResolveType(g.meta, g.locals[localIndex].typ).kind) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(&g.asm, g.locals[localIndex].offset,
			renvoComplexSecondaryStackOffset(g, g.locals[localIndex].typ, g.locals[localIndex].offset))
		return true
	}
	if e.kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		return renvoEmitComplexMemSecondaryRegs(g, renvoInferParsedExprType(g, ep, idx))
	}
	if e.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		return renvoEmitComplexMemSecondaryRegs(g, renvoInferParsedExprType(g, ep, idx))
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		return renvoEmitComplexMemSecondaryRegs(g, renvoInferParsedExprType(g, ep, idx))
	}
	if (e.kind == renvoExprInt || e.kind == renvoExprFloat) && renvoExprTokenIsImaginary(g.prog, e.tok) {
		renvoAsmPrimaryImm(&g.asm, 0)
		bits := renvoParseFloatTokenBits(g.prog, e.tok, 52, 11, 1023)
		if componentKind == renvoTypeFloat32 {
			bits = renvoParseFloatTokenBits(g.prog, e.tok, 23, 8, 127)
		}
		renvoAsmPushPrimary(&g.asm)
		renvoEmitFloat64BitsPrimary(&g.asm, bits)
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoAsmPopPrimary(&g.asm)
		return true
	}
	if e.kind == renvoExprCall && renvoResolvedNumericCalleeCode(g, ep, e.left) == renvoIdentComplex {
		if e.argCount != 2 {
			return false
		}
		realOffset := renvoAddUnnamedLocal(g, componentKind)
		if !renvoEmitScalarExprForKind(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), componentKind) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, realOffset)
		if !renvoEmitScalarExprForKind(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1), componentKind) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, realOffset)
		renvoAsmPopSecondary(&g.asm)
		return true
	}
	if e.kind == renvoExprCall && renvoTypeKindIsComplex(renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind) {
		return renvoEmitUserCall(g, ep, idx)
	}
	if e.kind == renvoExprBinary && (renvoTokCharIs(g.prog, e.tok, '+') || renvoTokCharIs(g.prog, e.tok, '-') || renvoTokCharIs(g.prog, e.tok, '*') || renvoTokCharIs(g.prog, e.tok, '/')) {
		leftReal := renvoAddUnnamedLocal(g, componentKind)
		leftImag := renvoAddUnnamedLocal(g, componentKind)
		rightReal := renvoAddUnnamedLocal(g, componentKind)
		rightImag := renvoAddUnnamedLocal(g, componentKind)
		if !renvoEmitComplexValueRegsForKind(g, ep, e.left, complexKind) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, leftReal, leftImag)
		if !renvoEmitComplexValueRegsForKind(g, ep, e.right, complexKind) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, rightReal, rightImag)
		if renvoTokCharIs(g.prog, e.tok, '*') || renvoTokCharIs(g.prog, e.tok, '/') {
			return renvoEmitComplexMultiplyDivide(g, leftReal, leftImag, rightReal, rightImag, e.tok, componentKind)
		}
		renvoEmitComplexBinaryComponent(g, leftReal, rightReal, renvo_runtime_UnsafeByteAt(g.prog.src, renvoTokStart(g.prog, e.tok)), componentKind)
		renvoAsmStorePrimaryStack(&g.asm, leftReal)
		renvoEmitComplexBinaryComponent(g, leftImag, rightImag, renvo_runtime_UnsafeByteAt(g.prog.src, renvoTokStart(g.prog, e.tok)), componentKind)
		renvoAsmPushPrimary(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, leftReal)
		renvoAsmPopSecondary(&g.asm)
		return true
	}
	if renvoTypeKindIsScalarValue(renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind) {
		if !renvoEmitScalarExprForKind(g, ep, idx, componentKind) {
			return false
		}
		renvoAsmSecondaryImm(&g.asm, 0)
		return true
	}
	return false
}

func renvoEmitComplexProduct(g *renvoLinearGen, left int, right int, kind int) {
	renvoAsmLoadPrimaryTertiaryStack(&g.asm, right, left)
	renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, '*', kind)
}

func renvoEmitComplexProductPair(g *renvoLinearGen, firstLeft int, firstRight int, secondLeft int, secondRight int, subtract bool, kind int) {
	temp := renvoAddUnnamedLocal(g, kind)
	renvoEmitComplexProduct(g, firstLeft, firstRight, kind)
	renvoAsmStorePrimaryStack(&g.asm, temp)
	renvoEmitComplexProduct(g, secondLeft, secondRight, kind)
	renvoAsmLoadTertiaryStack(&g.asm, temp)
	if subtract {
		renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, '-', kind)
	} else {
		renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, '+', kind)
	}
}

func renvoEmitComplexMultiplyDivide(g *renvoLinearGen, leftReal int, leftImag int, rightReal int, rightImag int, tok int, kind int) bool {
	realPart := renvoAddUnnamedLocal(g, kind)
	imagPart := renvoAddUnnamedLocal(g, kind)
	multiply := renvoTokCharIs(g.prog, tok, '*')
	renvoEmitComplexProductPair(g, leftReal, rightReal, leftImag, rightImag, multiply, kind)
	renvoAsmStorePrimaryStack(&g.asm, realPart)
	renvoEmitComplexProductPair(g, leftImag, rightReal, leftReal, rightImag, !multiply, kind)
	renvoAsmStorePrimaryStack(&g.asm, imagPart)
	if !multiply {
		denominator := renvoAddUnnamedLocal(g, kind)
		renvoEmitComplexProductPair(g, rightReal, rightReal, rightImag, rightImag, false, kind)
		renvoAsmStorePrimaryStack(&g.asm, denominator)
		renvoAsmLoadPrimaryTertiaryStack(&g.asm, denominator, realPart)
		if !renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, '/', kind) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, realPart)
		renvoAsmLoadPrimaryTertiaryStack(&g.asm, denominator, imagPart)
		if !renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, '/', kind) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, imagPart)
	}
	renvoAsmLoadPrimarySecondaryStack(&g.asm, realPart, imagPart)
	return true
}

func renvoPackComplex64RegsPrimary(g *renvoLinearGen) {
	realPart := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
	renvoAsmStorePrimaryStack(&g.asm, realPart)
	renvoAsmCopySecondaryToPrimary(&g.asm)
	renvoAsmShlPrimaryImm(&g.asm, 32)
	renvoAsmLoadTertiaryStack(&g.asm, realPart)
	renvoAsmAddPrimaryTertiary(&g.asm)
}

func renvoUnpackComplex64MemSecondaryRegs(g *renvoLinearGen) bool {
	address := renvoAddUnnamedLocal(g, renvoTypeInt)
	realPart := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
	renvoAsmStoreSecondaryStack(&g.asm, address)
	renvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, 0, 4)
	renvoAsmStorePrimaryStack(&g.asm, realPart)
	renvoAsmLoadSecondaryStack(&g.asm, address)
	renvoAsmLoadPrimaryMemSecondaryDispSize(&g.asm, 4, 4)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, realPart)
	return true
}

func renvoEmitComplexMemSecondaryRegs(g *renvoLinearGen, typ int) bool {
	kind := renvoResolveType(g.meta, typ).kind
	if !renvoTypeKindIsComplex(kind) {
		return false
	}
	if kind == renvoTypeComplex64 {
		return renvoUnpackComplex64MemSecondaryRegs(g)
	}
	offset := renvoAddUnnamedLocal(g, typ)
	renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, typ))
	renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
	return true
}

func renvoEmitComplexBinaryComponent(g *renvoLinearGen, left int, right int, op byte, kind int) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryTertiaryStack(&g.asm, right, left)
	renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, op, kind)
}

func renvoEmitTypeAssertionToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, valueOffset int, okOffset int, panicMismatch bool) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	asserted := renvoInferParsedExprType(g, ep, idx)
	if asserted == 0 || renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left)).kind != renvoTypeInterface {
		return false
	}
	sourceOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitInterfaceAssignToLocal(g, ep, e.left, sourceOffset) {
		return false
	}
	matchLabel := renvoAsmNewLabel(&g.asm)
	doneLabel := renvoAsmNewLabel(&g.asm)
	renvoEmitTypeMatchJump(g, sourceOffset-renvoBackendValueSlotSize, asserted, matchLabel)
	if panicMismatch {
		panicValue := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
		renvoAsmStoreStackImm(&g.asm, panicValue, 0)
		renvoAsmStoreStackImm(&g.asm, panicValue-renvoBackendValueSlotSize, renvoPanicTypeAssertionTag)
		renvoEmitPanicState(g, panicValue)
	} else {
		renvoZeroLocalAtOffset(g, valueOffset)
		renvoAsmStoreStackImm(&g.asm, okOffset, 0)
	}
	renvoAsmJmpMarkLabel(&g.asm, doneLabel, matchLabel)
	renvoCopyInterfaceValueToLocal(g, sourceOffset, asserted, valueOffset)
	if okOffset > 0 {
		renvoAsmStoreStackImm(&g.asm, okOffset, 1)
	}
	renvoAsmMarkLabel(&g.asm, doneLabel)
	return true
}

func renvoEmitMethodSelectorValuePrimary(g *renvoLinearGen, ep *renvoExprParse, idx int, fnIndex int, expression bool) bool {
	renvoNonNil(g, ep)
	if expression {
		renvoAsmPrimaryImm(&g.asm, renvoFunctionValueTag(g, fnIndex))
		return true
	}
	e := &ep.exprs[idx]
	fn := &g.meta.funcs[fnIndex]
	if fn.paramCount == 0 {
		return false
	}
	receiverType := g.meta.params[fn.firstParam].typ
	receiverOffset := renvoAddUnnamedLocal(g, receiverType)
	if !renvoEmitMethodReceiverToLocal(g, ep, e.left, receiverType, receiverOffset) {
		return false
	}
	handleOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoEmitBoundMethodHandle(g, fnIndex, receiverType, receiverOffset, false, handleOffset)
	renvoAsmLoadPrimaryStack(&g.asm, handleOffset)
	return true
}

func renvoEmitBoundMethodHandle(g *renvoLinearGen, fnIndex int, receiverType int, receiverOffset int, indirect bool, offset int) {
	renvoNonNil(g)
	receiverSize := renvoTypeCopySize(g.meta, receiverType)
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, renvoBackendValueSlotSize+receiverSize)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(&g.asm, addrOffset)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmPrimaryImm(&g.asm, renvoFunctionValueTag(g, fnIndex))
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, 0)
	if indirect {
		tempOffset := renvoAddUnnamedLocal(g, receiverType)
		renvoAsmLoadSecondaryStack(&g.asm, receiverOffset)
		renvoEmitCopyMemSecondaryToStack(g, tempOffset, receiverSize)
		receiverOffset = tempOffset
	}
	renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
	renvoEmitCopyStackToMemSecondary(g, receiverOffset, renvoBackendValueSlotSize, receiverSize)
	renvoAsmCopyStackSlot(&g.asm, addrOffset, offset)
}

func renvoEmitMethodReceiverToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, receiverType int, offset int) bool {
	renvoNonNil(g, ep)
	declared := renvoResolveType(g.meta, receiverType)
	renvoNonNil(declared)
	actualType := renvoInferParsedExprType(g, ep, idx)
	actual := renvoResolveType(g.meta, actualType)
	renvoNonNil(actual)
	if declared.kind == renvoTypePointer {
		if actual.kind == renvoTypePointer {
			return renvoEmitExprToLocal(g, ep, idx, offset)
		}
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, offset)
		return true
	}
	if actual.kind != renvoTypePointer {
		return renvoEmitExprToLocal(g, ep, idx, offset)
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, receiverType))
	return true
}

func renvoEmitClosureValuePrimary(g *renvoLinearGen, literalTok int) bool {
	renvoNonNil(g)
	closureIndex := renvoClosureIndexByToken(g.meta, literalTok)
	if closureIndex < 0 || !renvoPrepareClosureCaptures(g, closureIndex) {
		return false
	}
	info := &g.meta.closures[closureIndex]
	fnIndex := info.fnIndex
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	size := (info.captureCount + 1) * renvoBackendValueSlotSize
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, size)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(&g.asm, addrOffset)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	tag := renvoFunctionValueTag(g, fnIndex)
	renvoAsmPrimaryImm(&g.asm, tag)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, 0)
	for i := 0; i < info.captureCount; i++ {
		capture := &g.meta.captures[info.firstCapture+i]
		localIndex := renvoFindLocalIndex(g, capture.nameStart, capture.nameEnd)
		if localIndex < 0 || g.locals[localIndex].captureOff <= 0 {
			return false
		}
		renvoMoveCapturedLocal(g, localIndex, true)
		renvoAsmLoadPrimarySecondaryStack(&g.asm, g.locals[localIndex].captureOff, addrOffset)
		renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, (i+1)*renvoBackendValueSlotSize)
	}
	renvoAsmLoadPrimaryStack(&g.asm, addrOffset)
	return true
}

func renvoPrepareClosureCaptures(g *renvoLinearGen, closureIndex int) bool {
	renvoNonNil(g)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	if closureIndex < 0 || closureIndex >= len(meta.closures) {
		return false
	}
	info := &meta.closures[closureIndex]
	if info.ready {
		return true
	}
	if info.fnIndex < 0 || info.fnIndex >= len(meta.funcs) {
		return false
	}
	fn := &meta.funcs[info.fnIndex]
	info.firstCapture = len(meta.captures)
	for localIndex := 0; localIndex < g.localCount; localIndex++ {
		local := &g.locals[localIndex]
		if local.nameEnd <= local.nameStart || renvoClosureNameDeclared(meta, fn, local.nameStart, local.nameEnd) {
			continue
		}
		used := false
		for tok := fn.bodyStart; tok < fn.bodyEnd; tok++ {
			if renvoTokIsKind(p, tok, renvoTokIdent) && renvoBytesEqualRange(p.src, local.nameStart, local.nameEnd, int(renvoTokStart(p, tok)), int(renvoTokEnd(p, tok))) {
				used = true
				break
			}
		}
		if !used {
			continue
		}
		g.meta.captures = append(g.meta.captures, renvoSymbolInfo{nameStart: local.nameStart, nameEnd: local.nameEnd, typ: local.typ})
	}
	info.captureCount = len(g.meta.captures) - info.firstCapture
	info.ready = true
	return true
}

func renvoClosureNameDeclared(meta *renvoMeta, fn *renvoFuncInfo, nameStart int, nameEnd int) bool {
	renvoNonNil(meta, fn)
	for i := 1; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		if param.nameEnd > param.nameStart && renvoBytesEqualRange(meta.prog.src, param.nameStart, param.nameEnd, nameStart, nameEnd) {
			return true
		}
	}
	for i := fn.bodyStart; i < fn.bodyEnd; i++ {
		if !renvoTokIsKind(meta.prog, i, renvoTokIdent) || !renvoBytesEqualRange(meta.prog.src, int(renvoTokStart(meta.prog, i)), int(renvoTokEnd(meta.prog, i)), nameStart, nameEnd) {
			continue
		}
		if renvoTokIsKind(meta.prog, i-1, renvoTokVar) || renvoTok2Is(meta.prog, i+1, ':', '=') {
			return true
		}
	}
	return false
}

func renvoEmitScalarExprForKind(g *renvoLinearGen, ep *renvoExprParse, idx int, destKind int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if destKind == renvoTypeFloat32 {
		if bits, ok := renvoObjectFloatConstantBits(g.prog, ep, idx, renvoTypeFloat32); ok {
			renvoEmitFloat64BitsPrimary(&g.asm, bits)
			return true
		}
	}
	source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	renvoNonNil(source)
	if renvoUsesScaledFloat(&g.asm) {
		if renvoTypeKindIsFloat(source.kind) || renvoTypeKindIsFloat(destKind) {
			if !renvoEmitIntExpr(g, ep, idx) {
				return false
			}
			if !renvoTypeKindIsFloat(source.kind) && renvoTypeKindIsFloat(destKind) {
				renvoAsmShlPrimaryImm(&g.asm, 2)
			} else if renvoTypeKindIsFloat(source.kind) && !renvoTypeKindIsFloat(destKind) {
				renvoAsmCopyPrimaryToTertiary(&g.asm)
				renvoAsmPrimaryImm(&g.asm, 4)
				renvoAsmDivLeftTertiaryRightPrimary(&g.asm, false)
			}
			renvoAsmNormalizePrimaryForKind(&g.asm, destKind)
			return true
		}
	}
	if source.kind == renvoTypeFloat64 && destKind != renvoTypeFloat64 {
		if renvoUsesStackIEEEFloat(&g.asm) {
			return renvoEmit32BitFloat64ScalarConversion(g, ep, idx, destKind)
		}
	}
	if renvoFixedTarget == 0 {
		if g.c.objectFile && destKind == renvoTypeFunc && renvoExprIsNil(g.prog, e) {
			renvoAsmPrimaryImm(&g.asm, 0)
			return true
		}
		// The C frontend represents a C function-pointer cast as a one-argument
		// function-type conversion. In relocatable objects function values are raw
		// addresses, so the conversion itself is a word-preserving operation.
		if g.c.objectFile && destKind == renvoTypeFunc && e.kind == renvoExprCall && e.argCount == 1 {
			conversionType := renvoConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 && renvoResolveType(g.meta, conversionType).kind == renvoTypeFunc {
				return renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
			}
		}
	}
	if g.c.renvoNativeIntSize == 4 && e.kind == renvoExprCall && e.argCount == 1 && renvoConversionTypeFromExpr(g, ep, e.left) != 0 {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if ep.exprs[arg].kind == renvoExprBinary && renvoExprIsUntypedInteger(ep, arg) {
			temp := renvoAddUnnamedLocal(g, renvoTypeInt64)
			if !renvoEmitWideExprToLocal(g, ep, arg, temp, renvoTypeInt64) {
				return false
			}
			renvoAsmLoadPrimaryStack(&g.asm, temp)
			renvoAsmNormalizePrimaryForKind(&g.asm, destKind)
			return true
		}
	}
	if g.c.renvoNativeIntSize == 4 && renvoTypeKindIsWideInt(source.kind) &&
		!renvoTypeKindIsWideInt(destKind) {
		temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		if !renvoEmitWideExprToLocal(g, ep, idx, temp, source.kind) {
			return false
		}
		renvoAsmLoadPrimaryStack(&g.asm, temp)
		renvoAsmNormalizePrimaryForKind(&g.asm, destKind)
		return true
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmNormalizePrimaryForKind(&g.asm, source.kind)
	if renvoTypeKindIsFloat(destKind) && source.kind != destKind {
		if !renvoEmitIEEEFloatConversionPrimary(g, source.kind, destKind) {
			return false
		}
	} else if !renvoTypeKindIsFloat(destKind) && renvoTypeKindIsFloat(source.kind) {
		if !renvoEmitIEEEFloatConversionPrimary(g, source.kind, destKind) {
			return false
		}
	}
	renvoAsmNormalizePrimaryForKind(&g.asm, destKind)
	return true
}

func renvoEmit32BitFloat64ScalarConversion(g *renvoLinearGen, ep *renvoExprParse, idx int, destKind int) bool {
	floatTemp := renvoAddUnnamedLocal(g, renvoTypeFloat64)
	if !renvoEmitStackFloat64ExprToLocal(g, ep, idx, floatTemp) {
		return false
	}
	resultTemp := renvoAddUnnamedLocal(g, renvoTypeInt64)
	if destKind == renvoTypeFloat32 {
		renvo32IEEEConvertFloatStack(g, resultTemp, floatTemp, 8, 4)
	} else {
		intSize := 4
		if renvoTypeKindIsWideInt(destKind) {
			intSize = 8
		}
		renvo32IEEEFloatToIntStack(g, resultTemp, floatTemp, 8, intSize, !renvoTypeKindIsUnsignedInteger(destKind))
	}
	renvoAsmLoadPrimaryStack(&g.asm, resultTemp)
	return true
}

func renvoArrayBuiltinCount(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) int {
	renvoNonNil(g, ep, e)
	t := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)))
	renvoNonNil(t)
	if t.kind == renvoTypePointer {
		t = renvoResolveType(g.meta, t.elem)
		renvoNonNil(t)
	}
	if t.kind == renvoTypeArray {
		return t.count
	}
	return -1
}

func renvoEmit32BitIEEEFloatNegatePrimary(g *renvoLinearGen) bool {
	temp := renvoAddUnnamedLocal(g, renvoBuiltinTypeFloat32)
	renvoAsmStorePrimaryStack(&g.asm, temp)
	renvo32IEEENegateStack(g, temp, 4)
	renvoAsmLoadPrimaryStack(&g.asm, temp)
	return true
}

func renvoFloatComparisonChars(p *renvoProgram, tok int) (byte, byte, bool) {
	if tok < 0 || tok >= renvoTokCount(p) {
		return 0, 0, false
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	if start >= end {
		return 0, 0, false
	}
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	c1 := byte(0)
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	return c0, c1, renvoIsComparisonChars(c0, c1)
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

func renvoEmitIEEEFloatPrimaryTertiaryOp(g *renvoLinearGen, tok int, kind int) bool {
	c0, c1, comparison := renvoFloatComparisonChars(g.prog, tok)
	if !comparison {
		return renvoEmitIEEEFloatArithmeticPrimaryTertiary(g, c0, kind)
	}
	return renvoEmitIEEEComparePrimaryTertiary(g, c0, c1, kind)
}

func renvoEmitNativeFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	kind := renvoBinaryFloatKind(g, ep, e)
	if !renvoEmitScalarExprForKind(g, ep, e.left, kind) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, ep, e.right, kind) {
		return false
	}
	renvoAsmPopTertiary(a)
	return renvoEmitIEEEFloatPrimaryTertiaryOp(g, e.tok, kind)
}

func renvoEmitScaledCompatibilityFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	e := &ep.exprs[idx]
	kind := renvoBinaryFloatKind(g, ep, e)
	if !renvoEmitScalarExprForKind(g, ep, e.left, kind) {
		return false
	}
	if renvoTokCharIs(g.prog, e.tok, '/') {
		renvoAsmShlPrimaryImm(&g.asm, 2)
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitScalarExprForKind(g, ep, e.right, kind) {
		return false
	}
	renvoAsmPopTertiary(&g.asm)
	if renvoTokCharIs(g.prog, e.tok, '*') {
		renvoAsmMulPrimaryTertiary(&g.asm)
		renvoAsmSarPrimaryImm(&g.asm, 2)
		return true
	}
	if renvoTokCharIs(g.prog, e.tok, '/') {
		renvoAsmDivLeftTertiaryRightPrimary(&g.asm, false)
		return true
	}
	return renvoEmitPrimaryTertiaryOp(g, e.tok)
}

func renvoEmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if renvoUsesScaledFloat(&g.asm) {
		return renvoEmitScaledCompatibilityFloatBinaryExpr(g, ep, idx)
	}
	if renvoUsesStackIEEEFloat(&g.asm) {
		return renvoEmitWideFloatBinaryExpr(g, ep, idx)
	}
	return renvoEmitNativeFloatBinaryExpr(g, ep, idx)
}

func renvoEmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	renvoNonNil(g, locEp, loc)
	a := &g.asm
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		renvoAsmSliceHeaderAddressesSecondary(a)
		return true
	}
	if loc.global {
		renvoAsmSliceHeaderAddressesBss(a, loc.offset)
		return true
	}
	renvoAsmSliceHeaderAddressesStack(a, loc.offset)
	return true
}
func renvoEmitStringPtrExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	return renvoEmitStringValueRegs(g, ep, idx)
}

func renvoExprIsErrorStringCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.kind != renvoExprCall || e.argCount != 0 {
		return false
	}
	callee := &ep.exprs[e.left]
	return callee.kind == renvoExprSelector && renvoBytesEqualText(g.prog.src, callee.nameStart, callee.nameEnd, "Error") && renvoTypeIsString(g.meta, renvoInferParsedExprType(g, ep, callee.left))
}

func renvoEmitIndexedSelectorAddressSecondary(g *renvoLinearGen, ep *renvoExprParse, idx int, fieldOffset int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	indexExpr := &ep.exprs[idx]
	leftType := renvoInferParsedExprType(g, ep, indexExpr.left)
	sliceType := renvoResolveType(meta, leftType)
	renvoNonNil(sliceType)
	elemTypeIndex := 0
	if sliceType.kind == renvoTypePointer {
		pointerElem := sliceType.elem
		pointee := renvoResolveType(meta, pointerElem)
		renvoNonNil(pointee)
		if pointee.kind == renvoTypeArray || pointee.kind == renvoTypeSlice {
			sliceType = pointee
		} else {
			elemTypeIndex = pointerElem
		}
	}
	if elemTypeIndex == 0 && sliceType.kind != renvoTypeSlice && sliceType.kind != renvoTypeArray {
		return false
	}
	if elemTypeIndex == 0 {
		elemTypeIndex = sliceType.elem
	}
	elemType := renvoResolveType(meta, elemTypeIndex)
	renvoNonNil(elemType)
	if elemType.kind != renvoTypeStruct && elemType.kind != renvoTypePointer {
		return false
	}
	if !renvoEmitIndexAddressPrimary(g, ep, idx) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	if elemType.kind == renvoTypePointer {
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
	}
	if fieldOffset != 0 {
		renvoAsmAddSecondaryImm(a, fieldOffset)
	}
	return true
}

func renvoTruncParams(data *[]renvoSymbolInfo, count int) {
	renvoNonNil(data)
	*data = (*data)[:count]
}

func renvoTruncTypes(data *[]renvoTypeInfo, count int) {
	renvoNonNil(data)
	*data = (*data)[:count]
}

func renvoTruncFields(data *[]renvoFieldInfo, count int) {
	renvoNonNil(data)
	*data = (*data)[:count]
}

func renvoTruncBytes(data *[]byte, count int) {
	renvoNonNil(data)
	*data = (*data)[:count]
}

func renvoEmitLengthCapacityCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	callee := renvoExprIdentCode(p, ep, e.left)
	if e.argCount != 1 || (callee != renvoIdentCap && callee != renvoIdentLen) {
		return false
	}
	firstArgIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	count := renvoArrayBuiltinCount(g, ep, e)
	if count >= 0 {
		renvoAsmPrimaryImm(a, count)
		return true
	}
	if callee == renvoIdentCap {
		if renvoEmitDirectSelectorWords(g, ep, firstArgIndex, 16, -1, g.c.renvoNativeIntSize) {
			return true
		}
		if !renvoEmitSlicePtrCap(g, ep, firstArgIndex) {
			return false
		}
		renvoAsmSliceCountResult(a)
		return true
	}
	if callee == renvoIdentLen {
		arg := &ep.exprs[firstArgIndex]
		if arg.kind == renvoExprString {
			msg := renvoDecodeStringToken(p, arg.tok)
			msgLen := len(msg)
			renvoAsmPrimaryImm(a, msgLen)
			return true
		}
		if arg.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
			if localIndex >= 0 && (renvoTypeIsSlice(meta, g.locals[localIndex].typ) || renvoTypeIsString(meta, g.locals[localIndex].typ)) {
				renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset-8)
				return true
			}
			globalOffset := renvoFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
			globalType := renvoFindGlobalType(g, arg.nameStart, arg.nameEnd)
			if globalOffset >= 0 && (renvoTypeIsString(meta, globalType) || renvoTypeIsSlice(meta, globalType)) {
				renvoAsmLoadPrimaryBss(a, globalOffset+8)
				return true
			}
			constTok := renvoFindConstStringToken(g, arg.nameStart, arg.nameEnd)
			if constTok >= 0 {
				msg := renvoDecodeStringToken(p, constTok)
				msgLen := len(msg)
				renvoAsmPrimaryImm(a, msgLen)
				return true
			}
		}
		if arg.kind == renvoExprSelector && renvoCanLoadDirectSliceCountSelector(g) {
			argType := renvoInferParsedExprType(g, ep, firstArgIndex)
			if renvoTypeIsSlice(meta, argType) || renvoTypeIsString(meta, argType) {
				if renvoEmitDirectSelectorWords(g, ep, firstArgIndex, 8, -1, g.c.renvoNativeIntSize) {
					return true
				}
				if offset, ok := renvoLocalStructSelectorOffset(g, ep, firstArgIndex); ok {
					renvoAsmLoadPrimaryStack(a, offset-8)
					return true
				}
				if !renvoEmitSelectorAddressSecondary(g, ep, firstArgIndex) {
					return false
				}
				renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
				return true
			}
		}
		if arg.kind == renvoExprUnary && renvoTokCharIs(p, arg.tok, '*') {
			if !renvoEmitIntExpr(g, ep, arg.left) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
			return true
		}
		argIndex := firstArgIndex
		if renvoTypeIsString(meta, renvoInferParsedExprType(g, ep, argIndex)) {
			if !renvoEmitStringValueRegs(g, ep, argIndex) {
				return false
			}
			renvoAsmPushSecondary(a)
			renvoAsmPopPrimary(a)
			return true
		}
		if !renvoEmitSlicePtrLen(g, ep, firstArgIndex) {
			return false
		}
		renvoAsmSliceCountResult(a)
		return true
	}
	return false
}

func renvoNormalizeNativeExprPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) {
	renvoNonNil(g, ep)
	resultType := renvoInferParsedExprType(g, ep, idx)
	renvoAsmNormalizePrimaryForKind(&g.asm, renvoResolveType(g.meta, resultType).kind)
}

func renvoEmitSwitchStringCaseTest(g *renvoLinearGen, valueOffset int, lenOffset int, ep *renvoExprParse, idx int, matchLabel int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	label := renvoEnsureStringEqualHelper(g)
	if renvoPreparedBackendActive != 0 {
		caseOff := renvoAddUnnamedLocal(g, renvoTypeString)
		if !renvoEmitStringValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, caseOff, caseOff-renvoBackendValueSlotSize)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord0, valueOffset)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord1, lenOffset)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord2, caseOff)
		renvoRTGAsmLoadFrame(a, renvoRTGCallWord3, caseOff-renvoBackendValueSlotSize)
		renvoAsmCallLabel(a, label)
		renvoAsmCmpPrimaryImm8(a, 0)
		renvoAsmJnzLabel(a, matchLabel)
		return true
	}
	if !renvoEmitStringValueRegs(g, ep, idx) {
		return false
	}
	renvoAsmCopySecondaryToTertiary(a)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryStack(a, valueOffset)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOffset)
	renvoAsmStringEqualLeftLength(a)
	renvoAsmCallLabel(a, label)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, matchLabel)
	return true
}

func renvoEmitWordCompareJump(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, label int, jumpIfTrue bool) bool {
	renvoNonNil(g, ep, e)
	p := g.prog
	if e.tok < 0 || e.tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, e.tok)
	end := renvoTokEnd(p, e.tok)
	if start >= end {
		return false
	}
	c0 := renvo_runtime_UnsafeByteAt(p.src, start)
	var c1 byte
	if start+1 < end {
		c1 = renvo_runtime_UnsafeByteAt(p.src, start+1)
	}
	if !renvoIsComparisonChars(c0, c1) {
		return false
	}
	// The immediate compare fast path operates on raw integer bits. Floating
	// operands must use IEEE comparison so NaNs remain unordered.
	usesFloat := renvoBinaryUsesFloat(g, ep, e)
	floatKind := 0
	if usesFloat {
		floatKind = renvoBinaryFloatKind(g, ep, e)
	}
	leftIndex := e.left
	rightIndex := e.right
	unsigned := (c0 == '<' || c0 == '>') &&
		(renvoExprHasUnsignedIntType(g, ep, e.left) ||
			renvoExprHasUnsignedIntType(g, ep, e.right))
	if (c0 == '<' || c0 == '>') && !unsigned && renvoUsesUnsignedPointerOrdering(g) {
		leftType := renvoInferParsedExprType(g, ep, leftIndex)
		rightType := renvoInferParsedExprType(g, ep, rightIndex)
		unsigned = renvoResolveType(g.meta, leftType).kind == renvoTypePointer ||
			renvoResolveType(g.meta, rightType).kind == renvoTypePointer
	}
	right := &ep.exprs[rightIndex]
	rightConst := renvoConstResult{}
	if !usesFloat {
		rightConst = renvoEvalConstExpr(g, ep, rightIndex)
	}
	if renvoFixedTarget == 0 && !usesFloat && rightConst.ok && rightConst.value == 0 && (c0 == '=' || c0 == '!') &&
		renvoEmitLocalBitTestJump(g, ep, leftIndex, c0, label, jumpIfTrue) {
		return true
	}
	if renvoFixedTarget == 0 && !usesFloat && rightConst.ok && rightConst.value == 0 && (c0 == '=' || c0 == '!') {
		left := &ep.exprs[leftIndex]
		if left.kind == renvoExprCall && left.left >= 0 && left.left < len(ep.exprs) &&
			ep.exprs[left.left].kind == renvoExprIdent {
			jumpOnValue := jumpIfTrue
			if c0 == '=' {
				jumpOnValue = !jumpOnValue
			}
			inlined := renvoEmitCInlineReturn(g, ep, leftIndex, left, &ep.exprs[left.left], label, jumpOnValue)
			if inlined >= 0 {
				return inlined != 0
			}
		}
	}
	if renvoFixedTarget == 0 && !usesFloat && rightConst.ok &&
		renvoEmitDerefCompareJump(g, ep, leftIndex, rightConst.value, c0, c1, label, jumpIfTrue, unsigned) {
		return true
	}
	if renvoFixedTarget == 0 && !usesFloat && rightConst.ok {
		left := &ep.exprs[leftIndex]
		if left.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)
			if localIndex >= 0 && renvoTypeSize(g.meta, g.locals[localIndex].typ) == 4 {
				if renvoEmitLocalImmediateCompareJump(g, g.locals[localIndex].offset, rightConst.value, c0, c1, label, jumpIfTrue, unsigned) {
					return true
				}
			}
		}
	}
	if !usesFloat && renvoCanCompareWordImmediate(g, unsigned) {
		if rightConst.ok && renvoAsmImmFits8Signed(rightConst.value) {
			if !renvoEmitIntExpr(g, ep, leftIndex) {
				return false
			}
			// Locals occupy a native-sized backend slot even when their language
			// type is narrower. A pointer write may update only the low byte, word,
			// or dword, so normalize the loaded operand before an immediate branch.
			leftKind := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, leftIndex)).kind
			renvoAsmCompareWordImmediateKind(&g.asm, rightConst.value, leftKind)
			renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)
			return true
		}
	}
	left := &ep.exprs[leftIndex]
	if left.kind == renvoExprIdent && right.kind == renvoExprIdent {
		leftLocal := renvoFindLocalIndex(g, left.nameStart, left.nameEnd)
		rightLocal := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)
		if leftLocal >= 0 && rightLocal >= 0 && !usesFloat {
			leftType := g.locals[leftLocal].typ
			rightType := g.locals[rightLocal].typ
			leftKind := renvoResolveType(g.meta, leftType).kind
			rightKind := renvoResolveType(g.meta, rightType).kind
			leftSize := renvoTypeSize(g.meta, leftType)
			rightSize := renvoTypeSize(g.meta, rightType)
			if renvoEmitLocalWordCompareJump(g, g.locals[leftLocal].offset, g.locals[rightLocal].offset, c0, c1, label, jumpIfTrue, unsigned, leftKind, rightKind, leftSize, rightSize) {
				return true
			}
		}
	}
	if c0 == '=' || c0 == '!' {
		leftType := renvoInferParsedExprType(g, ep, leftIndex)
		rightType := renvoInferParsedExprType(g, ep, rightIndex)
		leftResolved := renvoResolveType(g.meta, leftType)
		renvoNonNil(leftResolved)
		if leftResolved.kind == renvoTypeArray || leftResolved.kind == renvoTypeStruct || renvoTypeKindIsComplex(leftResolved.kind) {
			return false
		}
		if renvoTypeIsString(g.meta, leftType) || renvoTypeIsString(g.meta, rightType) {
			return false
		}
		if right.kind == renvoExprString {
			return false
		}
		if right.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)
			if localIndex >= 0 && renvoTypeIsString(g.meta, g.locals[localIndex].typ) {
				return false
			}
		}
	}
	if !renvoEmitWideCompareOperand(g, ep, leftIndex, floatKind) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitWideCompareOperand(g, ep, rightIndex, floatKind) {
		return false
	}
	renvoAsmPopTertiary(&g.asm)
	if usesFloat && renvoUsesRegisterIEEEComparison(g) {
		if !renvoEmitIEEEFloatPrimaryTertiaryOp(g, e.tok, floatKind) {
			return false
		}
		if jumpIfTrue {
			renvoAsmJnzPrimary(&g.asm, label)
		} else {
			renvoAsmJzPrimary(&g.asm, label)
		}
		return true
	}
	// Evaluate calls left to right, then arrange the integer comparison operands.
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmCopyTertiaryToPrimary(&g.asm)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	unsigned = renvoEmitCompareWordOperands(g, unsigned)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	renvoEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue, unsigned)
	return true
}

func renvoEmitNativeStructReturnExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	if g.returnStruct <= 0 {
		return false
	}
	e := &ep.exprs[idx]
	resultType := g.meta.funcs[g.currentFunc].resultType
	size := renvoTypeSize(meta, resultType)
	renvoAsmLoadSecondaryStack(a, g.returnStruct)
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || renvoTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		renvoEmitCopyStackToMemSecondary(g, g.locals[localIndex].offset, 0, size)
		return true
	}
	if e.kind == renvoExprIndex {
		leftType := renvoInferParsedExprType(g, ep, e.left)
		sliceType := renvoResolveType(meta, leftType)
		renvoNonNil(sliceType)
		elemType := renvoResolveType(meta, sliceType.elem)
		renvoNonNil(elemType)
		if sliceType.kind != renvoTypeSlice || elemType.kind != renvoTypeStruct || renvoTypeSize(meta, sliceType.elem) != size {
			return false
		}
		if !renvoEmitIntExpr(g, ep, e.right) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoAsmMulTertiaryImm(a, size)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmAddSecondaryTertiary(a)
		if renvoPreparedBackendActive != 0 {
			temp := renvoAddUnnamedLocal(g, resultType)
			renvoEmitCopyMemSecondaryToStack(g, temp, size)
			renvoAsmLoadSecondaryStack(a, g.returnStruct)
			renvoEmitCopyStackToMemSecondary(g, temp, 0, size)
			return true
		}
		renvoAsmLoadTertiaryStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			renvoAsmLoadPrimaryMemSecondaryDisp(a, at)
			renvoAsmStorePrimaryMemTertiaryDisp(a, at)
		}
		return true
	}
	if e.kind == renvoExprComposite {
		renvoAsmPrimaryImm(a, 0)
		for at := 0; at < size; at += g.c.renvoNativeIntSize {
			renvoAsmStorePrimaryMemSecondaryDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := renvoCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 || !renvoEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == renvoExprCall {
		fnIndex, wordCount := renvoPrepareStructCall(g, ep, idx, resultType)
		if fnIndex < 0 {
			return false
		}
		renvoAsmLoadPrimaryStack(a, g.returnStruct)
		renvoAsmPushPrimary(a)
		renvoEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}

func renvoEmitNamedConversionCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	calleeExpr := &ep.exprs[e.left]
	if calleeExpr.kind != renvoExprIdent {
		return false
	}
	namedType := renvoFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
	resolved := renvoResolveType(g.meta, namedType)
	renvoNonNil(resolved)
	if resolved.kind == renvoTypeString {
		return renvoEmitStringValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
	}
	if renvoTypeKindIsScalarInt(resolved.kind) {
		if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(&g.asm, resolved.kind)
		return true
	}
	return false
}

func renvoEmitIndexedStructField(g *renvoLinearGen, ep *renvoExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := renvoInferParsedExprType(g, ep, indexExpr.left)
	sliceType := renvoResolveType(g.meta, leftType)
	renvoNonNil(sliceType)
	if sliceType.kind == renvoTypePointer {
		sliceType = renvoResolveType(g.meta, sliceType.elem)
		renvoNonNil(sliceType)
	}
	if sliceType.kind != renvoTypeSlice && sliceType.kind != renvoTypeArray {
		return false
	}
	elemType := renvoResolveType(g.meta, sliceType.elem)
	renvoNonNil(elemType)
	if elemType.kind != renvoTypeStruct && elemType.kind != renvoTypePointer {
		return false
	}
	fieldOffset := renvoStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	fieldType := renvoStructFieldType(g, sliceType.elem, fieldStart, fieldEnd)
	if !renvoEmitIndexedSelectorAddressSecondary(g, ep, indexIdx, fieldOffset) {
		return false
	}
	renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(g.c.renvoNativeIntSize, renvoResolveType(g.meta, fieldType).kind))
	return true
}

func renvoEmitSelectorAddressSecondary(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	renvoNonNil(meta)
	a := &g.asm
	e := &ep.exprs[idx]
	base := &ep.exprs[e.left]
	baseType := renvoInferParsedExprType(g, ep, e.left)
	fieldOffset := renvoStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if renvoStructPromotedPointerField(g, baseType, e.nameStart, e.nameEnd) >= 0 {
		return renvoEmitPromotedPointerSelectorAddress(g, ep, idx, baseType)
	}
	baseResolved := renvoResolveType(meta, baseType)
	renvoNonNil(baseResolved)
	if base.kind == renvoExprUnary && renvoTokCharIs(g.prog, base.tok, '*') {
		pointerExpr := base.left
		if baseResolved.kind == renvoTypePointer {
			pointerExpr = e.left
		}
		if !renvoEmitIntExpr(g, ep, pointerExpr) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAddNativeSecondaryFieldOffset(a, fieldOffset)
		return true
	}
	if baseResolved.kind == renvoTypePointer && base.kind != renvoExprIdent && base.kind != renvoExprSelector {
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAddCheckedNativeSecondaryFieldOffset(g, fieldOffset)
		return true
	}
	if base.kind == renvoExprCall || base.kind == renvoExprAssert {
		if baseResolved.kind != renvoTypePointer && baseResolved.kind != renvoTypeStruct {
			return false
		}
		if baseResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAddCheckedNativeSecondaryFieldOffset(g, fieldOffset)
			return true
		}
	}
	if base.kind == renvoExprComposite || base.kind == renvoExprCall || base.kind == renvoExprAssert {
		offset := renvoAddUnnamedLocal(g, baseType)
		if !renvoEmitTypedAssign(g, ep, e.left, offset) {
			return false
		}
		renvoEmitSecondaryFrameAddress(g, offset-fieldOffset)
		return true
	}
	if renvoAsmFoldedFieldAddressing(&g.asm) && base.kind == renvoExprSelector {
		totalOffset := fieldOffset
		rootIndex := e.left
		for ep.exprs[rootIndex].kind == renvoExprSelector {
			part := &ep.exprs[rootIndex]
			partType := renvoResolveType(meta, renvoInferParsedExprType(g, ep, rootIndex))
			renvoNonNil(partType)
			if partType.kind == renvoTypePointer {
				break
			}
			partBaseType := renvoInferParsedExprType(g, ep, part.left)
			partOffset := renvoStructFieldOffset(g, partBaseType, part.nameStart, part.nameEnd)
			if partOffset < 0 {
				break
			}
			totalOffset += partOffset
			rootIndex = part.left
		}
		root := &ep.exprs[rootIndex]
		if root.kind == renvoExprIdent {
			localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
			if localIndex >= 0 {
				rootType := renvoResolveType(meta, g.locals[localIndex].typ)
				renvoNonNil(rootType)
				if rootType.kind == renvoTypePointer {
					renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
					renvoCheckNativeSecondaryFieldBase(g, localIndex)
					renvoAddNativeSecondaryFieldOffset(a, totalOffset)
					return true
				}
			}
		}
	}
	if base.kind == renvoExprIndex {
		return renvoEmitIndexedSelectorAddressSecondary(g, ep, e.left, fieldOffset)
	}
	if base.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := renvoFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := renvoFindGlobalType(g, base.nameStart, base.nameEnd)
			t := renvoResolveType(meta, globalType)
			renvoNonNil(t)
			if globalOffset < 0 {
				return false
			}
			if t.kind == renvoTypePointer {
				renvoAsmLoadPrimaryBss(a, globalOffset)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAddCheckedNativeSecondaryFieldOffset(g, fieldOffset)
				return true
			}
			if t.kind != renvoTypeStruct {
				return false
			}
			renvoAsmPrimaryBssAddr(a, globalOffset)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAddNativeSecondaryFieldOffset(a, fieldOffset)
			return true
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			renvoCheckNativeSecondaryFieldBase(g, localIndex)
			renvoAddNativeSecondaryFieldOffset(a, fieldOffset)
			return true
		}
		renvoEmitSecondaryFrameAddress(g, g.locals[localIndex].offset-fieldOffset)
		return true
	}
	if base.kind == renvoExprSelector {
		if !renvoEmitSelectorAddressSecondary(g, ep, e.left) {
			return false
		}
		t := renvoResolveType(meta, baseType)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			renvoEmitDereferenceSecondary(g)
			renvoAddCheckedNativeSecondaryFieldOffset(g, fieldOffset)
			return true
		}
		renvoAddNativeSecondaryFieldOffset(a, fieldOffset)
		return true
	}
	return false
}

func renvoAddNativeSecondaryFieldOffset(a *renvoAsm, fieldOffset int) {
	renvoNonNil(a)
	if fieldOffset != 0 {
		renvoAsmAddSecondaryImm(a, fieldOffset)
	}
}

func renvoCheckNativeSecondaryFieldBase(g *renvoLinearGen, localIndex int) {
	renvoNonNil(g)
	if !renvoRuntimeNonNilLocalNeeded(g, localIndex) {
		return
	}
	renvoEmitRuntimeNonNilSecondary(g)
}

func renvoAddCheckedNativeSecondaryFieldOffset(g *renvoLinearGen, fieldOffset int) {
	renvoNonNil(g)
	renvoEmitRuntimeNonNilSecondary(g)
	renvoAddNativeSecondaryFieldOffset(&g.asm, fieldOffset)
}

func renvoCompileProgramToOutput(prog *renvoProgram, output int, target int, arenaSize int) int {
	renvoNonNil(prog)
	renvoSetTarget(target)
	context := renvoNewCompileContext(target, renvoCompilerStripSymbols, renvoCompilerWindowsSubsystem == 2, renvoCompilerEmitImage)
	prog.c = *context
	if !prog.ok {
		renvoPrintErr("renvo: parse failed\n")
		return 1
	}
	if targetIsKernelModule(context) && !context.objectFile {
		if !renvoPrepareKernelMetadata(context) {
			renvoPrintErr("renvo: kernel metadata unavailable\n")
			return 1
		}
		if target == renvoTargetLinuxKernelAmd64 {
			renvoCaptureKernelCompileContext(context)
		} else {
			renvoPopulateKernelCompileContext(context)
		}
		prog.c = *context
	}
	var meta renvoMeta
	renvoBuildMetaInto(prog, &meta)
	if !meta.ok {
		renvoPrintErr("renvo: meta failed\n")
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(target, arenaSize)
	var result renvoCompileResult
	if renvoPreparedBackendActive != 0 || renvoFixedTarget == 0 && target == renvoTargetRTG {
		result = renvoTryCompileScalarProgramRTG(prog, &meta)
	} else if !renvoProgramCacheSupported(meta.c) {
		result = renvoTryCompileScalarProgramScratch(prog, &meta)
	} else {
		result = renvoTryCompileScalarProgramCached(prog, &meta)
	}
	if result.ok {
		write(output, renvoCompileOutputDataWithContext(context, result.data, target), -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	if renvoPreparedBackendActive != 0 && renvoRTGUnsupportedOperation == 5001 {
		// Preserve a machine-readable distinction for embedders: the target image
		// builder rejected an otherwise-emitted program (for example because a COM
		// image cannot fit its single segment).
		return 125
	}
	return 1
}

func renvoCompileUnitInput(input []int, output int, target int, arenaSize int) int {
	if len(input) != 1 {
		return -1
	}
	if input[0] == 0 {
		var src []byte
		src = renvoReadAll(input[0], src)
		if len(src) >= 4 && src[0] == 'R' && src[1] == 'N' && src[2] == 'V' && src[3] == 'O' {
			prog, isUnit, ok := renvoDecodeUnitProgram(src)
			if !isUnit {
				return -1
			}
			if !ok {
				renvoPrintErr("renvo: invalid unit input\n")
				return 1
			}
			return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
		}
		prog := renvoParseProgram(src)
		return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
	}
	header := make([]byte, 4)
	n := read(input[0], header, 0)
	if n != 4 || header[0] != 'R' || header[1] != 'N' || header[2] != 'V' || header[3] != 'O' {
		return -1
	}
	var unit []byte
	unit = renvoReadAll(input[0], unit)
	prog, isUnit, ok := renvoDecodeUnitProgram(unit)
	if !isUnit {
		return -1
	}
	if !ok {
		renvoPrintErr("renvo: invalid unit input\n")
		return 1
	}
	return renvoCompileProgramToOutput(&prog, output, target, arenaSize)
}

func renvoEmitAtomExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		renvoAsmLoadPrimaryIntToken(a, p, e.tok)
		return true
	}
	if e.kind == renvoExprFloat {
		if renvoPreparedBackendActive != 0 {
			renvoAsmPrimaryImm(a, renvoParseFloatTokenScaledCompatibility(p, e.tok))
			return true
		}
		renvoEmitFloat64BitsPrimary(a, renvoParseFloatTokenBits(p, e.tok, 52, 11, 1023))
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			floatConst := renvoEmitFloatConstByName(g, e.nameStart, e.nameEnd)
			if floatConst != 0 {
				return floatConst > 0
			}
			constResult := renvoEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := renvoFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					fnIndex := renvoFindMetaFunction(meta, e.nameStart, e.nameEnd)
					if fnIndex < 0 {
						return false
					}
					if renvoIsHostedObject(g.c) {
						return renvoEmitObjectFunctionAddress(g, fnIndex)
					}
					renvoAsmPrimaryImm(a, renvoFunctionValueTag(g, fnIndex))
					return true
				}
				renvoAsmLoadPrimaryBss(a, globalOffset)
				if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
					globalType := renvoFindGlobalType(g, e.nameStart, e.nameEnd)
					globalResolved := renvoResolveType(meta, globalType)
					renvoNonNil(globalResolved)
					renvoAsmNormalizePrimaryForKind(a, globalResolved.kind)
				}
				return true
			}
			renvoAsmPrimaryImm(a, constResult.value)
			return true
		}
		renvoAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		if renvoFixedTarget == 0 || renvoFixedTarget == renvoTargetLinuxKernelAmd64 {
			localResolved := renvoResolveType(meta, g.locals[localIndex].typ)
			renvoNonNil(localResolved)
			renvoAsmNormalizePrimaryForKind(a, localResolved.kind)
		}
		return true
	}
	if e.kind == renvoExprChar {
		value := renvoParseCharToken(p, e.tok)
		renvoAsmPrimaryImm(a, value)
		return true
	}
	if e.kind == renvoExprBool {
		value := renvoBoolTokenValue(p, e.tok)
		renvoAsmPrimaryImm(a, value)
		return true
	}
	return false
}

func renvoEmitUnaryValueExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoTokCharIs(p, e.tok, '*') {
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmCopyPrimaryToSecondary(a)
		targetKind := renvoPointerTargetKind(g, ep, e.left)
		size := renvoScalarKindSize(g.c.renvoNativeIntSize, targetKind)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
		renvoAsmNormalizePrimaryForKind(a, targetKind)
		return true
	}
	if !renvoEmitIntExpr(g, ep, e.left) {
		return false
	}
	if renvoTokCharIs(p, e.tok, '-') {
		resultType := renvoInferParsedExprType(g, ep, idx)
		result := renvoResolveType(meta, resultType)
		if renvoTypeKindIsFloat(result.kind) {
			if renvoExprIsUntypedZeroFloat(p, ep, e.left) {
				return true
			}
			return renvoEmitIEEEFloatNegatePrimary(g, result.kind)
		}
		renvoAsmNegatePrimaryWord(a)
		renvoAsmNormalizePrimaryForKind(a, result.kind)
		return true
	}
	if renvoTokCharIs(p, e.tok, '+') {
		renvoNormalizeNativeExprPrimary(g, ep, idx)
		return true
	}
	if renvoTokCharIs(p, e.tok, '!') {
		renvoAsmBoolNotPrimary(a)
		return true
	}
	if renvoTokCharIs(p, e.tok, '^') {
		renvoAsmBitwiseNotPrimary(a)
		renvoNormalizeNativeExprPrimary(g, ep, idx)
		return true
	}
	return false
}

// renvoEmitWordIntrinsicCall returns -1 for ordinary builtin/conversion dispatch.
func renvoEmitWordIntrinsicCall(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	renvoNonNil(g, ep)
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoExprIsIdentText(p, ep, e.left, "renvoNonNil") {
		return renvoBoolInt(renvoEmitRuntimeTrustPointer(g, ep, e))
	}
	if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeByteAt") {
		return renvoBoolInt(renvoEmitRuntimeUnsafeIndex(g, ep, e, 1))
	}
	if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeInt32At") {
		return renvoBoolInt(renvoEmitRuntimeUnsafeIndex(g, ep, e, 4))
	}
	if renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_UnsafeIntAt") {
		return renvoBoolInt(renvoEmitRuntimeUnsafeIndex(g, ep, e, g.c.renvoNativeIntSize))
	}
	if renvoExprIsIdentText(p, ep, e.left, "renvoTruncBytes") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncParams") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncTypes") || renvoExprIsIdentText(p, ep, e.left, "renvoTruncFields") {
		return renvoBoolInt(renvoEmitRuntimeTruncateSlice(g, ep, e, -1))
	}
	callee := renvoExprIdentCode(p, ep, e.left)
	// The linker gives the Renvo os adapter and RBE-supplied typed syscall
	// declarations one reserved alias. Keep ordinary user functions named
	// syscall on the normal call path without compiling adapter stub bodies.
	if callee == renvoIdentSyscall && e.argCount == 4 ||
		renvoExprIsIdentText(p, ep, e.left, "renvo_runtime_Syscall") {
		return renvoBoolInt(renvoEmitArbitrarySyscall(g, ep, idx))
	}
	if renvoConversionTypeFromExpr(g, ep, e.left) == 0 &&
		(renvoFunctionValueCalleeType(g, ep, e.left) != 0 || renvoFuncInfoFromCall(g, ep, e.left) >= 0) {
		return renvoBoolInt(renvoEmitUserCall(g, ep, idx))
	}
	if e.argCount == 1 {
		arg := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		if renvoExprIsIdentText(p, ep, e.left, "Sizeof") {
			renvoAsmPrimaryImm(a, renvoTypeSize(meta, renvoInferParsedExprType(g, ep, arg)))
			return 1
		}
		if renvoExprIsIdentText(p, ep, e.left, "Alignof") {
			renvoAsmPrimaryImm(a, renvoLanguageTypeAlignment(meta, renvoInferParsedExprType(g, ep, arg)))
			return 1
		}
		if renvoExprIsIdentText(p, ep, e.left, "Offsetof") {
			selector := &ep.exprs[arg]
			renvoAsmPrimaryImm(a, renvoStructFieldOffset(g, renvoInferParsedExprType(g, ep, selector.left), selector.nameStart, selector.nameEnd))
			return 1
		}
	}
	if callee == renvoIdentPanic {
		return renvoBoolInt(renvoEmitBuiltinPanic(g, ep, idx))
	}
	if callee == renvoIdentNew {
		return renvoBoolInt(renvoEmitBuiltinNew(g, ep, idx))
	}
	return -1
}

func renvoEmitCdeclObjectFunctionPointerCall(g *renvoLinearGen, functionType *renvoTypeInfo, handleOffset int, argOffsets []int, resultOffset int) bool {
	renvoNonNil(g, functionType)
	if functionType.resolved != 0 || len(argOffsets) > 127 {
		return false
	}
	for i := 0; i < len(argOffsets); i++ {
		paramType := g.meta.fields[functionType.first+i].typ
		param := renvoResolveType(g.meta, paramType)
		if (!renvoTypeKindIsScalarInt(param.kind) && param.kind != renvoTypePointer && param.kind != renvoTypeFunc) ||
			renvoTypeSize(g.meta, paramType) > g.c.renvoNativeIntSize {
			return false
		}
	}
	result := renvoResolveType(g.meta, functionType.elem)
	if functionType.elem != 0 &&
		((!renvoTypeKindIsScalarInt(result.kind) && result.kind != renvoTypePointer && result.kind != renvoTypeFunc) ||
			renvoTypeSize(g.meta, functionType.elem) > g.c.renvoNativeIntSize) {
		return false
	}
	if !renvoAsmObjectIndirectStackCall(&g.asm, handleOffset, argOffsets) {
		return false
	}
	if resultOffset != 0 && functionType.elem != 0 {
		renvoAsmStorePrimaryStack(&g.asm, resultOffset)
	}
	return true
}

// renvoCompileSourceInputs owns the shared raw-source pipeline. Target policy
// chooses source storage and optional runtime support, not language traversal.
func renvoCompileSourceInputs(input []int, output int, arenaSize int) int {
	context := renvoLegacyCompileContext()
	capacity := renvoSourceCapacity(context)
	var src []byte
	if renvoSourceScratch(context) {
		src = renvoMakeByteScratch(capacity)
	} else {
		src = make([]byte, 0, capacity)
	}
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if renvoKernelProgram(&prog.c) {
		if !renvoPrepareKernelMetadata(&prog.c) {
			renvoPrintErr("renvo: kernel metadata unavailable\n")
			return 1
		}
		renvoCaptureKernelCompileContext(&prog.c)
	}
	if !prog.ok {
		return 1
	}
	if renvoSourceSoftFloat(&prog.c) && renvoProgramNeedsSoftFloat(&prog) {
		src = renvoAppendSoftFloatSource(src)
		prog = renvoParseProgram(src)
		if !prog.ok {
			return 1
		}
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramScratch(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
		return 0
	}
	if renvoProgramTargetMode(&prog.c) != 0 {
		renvoPrintErr("renvo: wasm32 compilation failed\n")
	} else {
		renvoPrintErr("renvo: compilation failed\n")
	}
	return 1
}

func renvoProgramNeedsSoftFloat(prog *renvoProgram) bool {
	for i := 0; i < renvoTokCount(prog); i++ {
		if renvoTokIsKind(prog, i, renvoTokFloat) {
			return true
		}
		if !renvoTokIsKind(prog, i, renvoTokIdent) {
			continue
		}
		tok := renvoTokAt(prog, i)
		if renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "float32") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "float64") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "complex64") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "complex128") {
			return true
		}
	}
	return false
}
