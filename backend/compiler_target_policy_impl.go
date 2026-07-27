package main

const renvoTargetLinuxAmd64 = 1
const renvoTargetLinux386 = 2
const renvoTargetLinuxAarch64 = 3
const renvoTargetLinuxArm = 4
const renvoTargetWindowsAmd64 = 5
const renvoTargetWindows386 = 6
const renvoTargetWasiWasm32 = 7
const renvoTargetDarwinArm64 = 8
const renvoTargetLinuxKernelAmd64 = 9
const renvoTargetWindowsArm64 = 10
const renvoTargetVM32 = 11

const renvoArchAmd64 = 1
const renvoArch386 = 2
const renvoArchAarch64 = 3
const renvoArchArm = 4
const renvoArchWasm32 = 5

const renvoOSLinux = 1
const renvoOSWindows = 2
const renvoOSDarwin = 3
const renvoOSWasi = 4
const renvoOSVM = 5

const renvoEndianLittle = 1
const renvoEndianBig = 2

const renvoAddressModelFlat = 1
const renvoAddressModelHarvard = 2
const renvoAddressModelSegmented = 3
const renvoAddressModelBanked = 4

const renvoPointerSpaceData = 1
const renvoPointerSpaceCode = 2
const renvoPointerSpaceFunction = 3
const renvoPointerSpaceGeneric = 4

const renvoRuntimePrint = 1
const renvoRuntimeOpen = 2
const renvoRuntimeClose = 4
const renvoRuntimeRead = 8
const renvoRuntimeWrite = 16
const renvoRuntimeChmod = 32
const renvoRuntimeHosted = 64
const renvoRuntimeHeap = 128
const renvoRuntimeVolatileMemory = 256
const renvoRuntimeInterrupts = 512

const renvoHeapNone = 0
const renvoHeapBump = 1
const renvoHeapExternal = 2

const renvoOOMTrap = 1
const renvoOOMResult = 2
const renvoOOMPanic = 3

const renvoVolatileWidth8 = 1
const renvoVolatileWidth16 = 2
const renvoVolatileWidth32 = 4
const renvoVolatileWidth64 = 8

const renvoInterruptNone = 0
const renvoInterruptVector = 1

const renvoFloatScaledInteger = 1
const renvoFloatIEEEHardware = 2
const renvoFloatIEEESoft = 3

// The current normalized backend stores scalar values in eight-byte virtual
// slots even when the target address or language int is narrower. Keep this
// distinct from the target data model so future C and small-device backends do
// not mistake an internal lowering detail for a machine ABI requirement.
const renvoBackendValueSlotSize = 8
const renvoBackendStringWordCount = 2
const renvoBackendSliceWordCount = 3
const renvoBackendHiddenResultWordCount = 1
const renvoBackendRegisterCallWordCount = 6
const renvoBackendStringValueSize = renvoBackendValueSlotSize * renvoBackendStringWordCount
const renvoBackendSliceValueSize = renvoBackendValueSlotSize * renvoBackendSliceWordCount

type renvoTargetProfile struct {
	target          int
	os              int
	arch            int
	charBits        int
	intBits         int
	pointerBits     int
	codePointerBits int
	funcPointerBits int
	endian          int
	maxAlign        int
	backendSlotSize int
	addressModel    int
	runtimeCaps     int
	heapModel       int
	oomModel        int
	volatileWidths  int
	interruptModel  int
	floatModel      int
}

// The target IDs are dense. Keep the core identity fields in compact tables so
// profile construction and active compiler state consume the same source of
// truth without pulling the full machine-profile builder into every compiler.
const targetOSTable = "\x00\x01\x01\x01\x01\x02\x02\x04\x03\x01\x02\x05"
const targetArchTable = "\x00\x01\x02\x03\x04\x01\x02\x05\x03\x01\x03\x05"
const renvoTargetIntBitsTable = "\x00\x40\x20\x40\x20\x40\x20\x20\x40\x40\x40\x20"

func renvoProfileForTarget(target int) (renvoTargetProfile, bool) {
	var p renvoTargetProfile
	if target < renvoTargetLinuxAmd64 || target > renvoTargetVM32 {
		return p, false
	}
	p.target = target
	p.os = int(targetOSTable[target])
	p.arch = int(targetArchTable[target])
	p.intBits = int(renvoTargetIntBitsTable[target])
	p.pointerBits = p.intBits
	p.maxAlign = p.intBits / 8
	if target == renvoTargetWasiWasm32 || target == renvoTargetVM32 {
		p.maxAlign = 8
	}
	p.charBits = 8
	p.endian = renvoEndianLittle
	p.backendSlotSize = renvoBackendValueSlotSize
	p.addressModel = renvoAddressModelFlat
	p.runtimeCaps = renvoRuntimePrint | renvoRuntimeOpen | renvoRuntimeClose | renvoRuntimeRead | renvoRuntimeWrite | renvoRuntimeChmod | renvoRuntimeHosted
	p.heapModel = renvoHeapNone
	p.oomModel = renvoOOMResult
	p.interruptModel = renvoInterruptNone
	p.floatModel = renvoFloatScaledInteger
	p.codePointerBits = p.pointerBits
	p.funcPointerBits = p.pointerBits
	return p, true
}

func renvoProfileHasRuntime(p renvoTargetProfile, capability int) bool {
	return p.runtimeCaps&capability == capability
}

func renvoProfileIsValid(p renvoTargetProfile) bool {
	if p.charBits < 8 || p.charBits%8 != 0 {
		return false
	}
	if p.intBits != 16 && p.intBits != 32 && p.intBits != 64 {
		return false
	}
	if p.pointerBits != 16 && p.pointerBits != 24 && p.pointerBits != 32 && p.pointerBits != 64 {
		return false
	}
	if p.codePointerBits != 16 && p.codePointerBits != 24 && p.codePointerBits != 32 && p.codePointerBits != 64 {
		return false
	}
	if p.funcPointerBits != 16 && p.funcPointerBits != 24 && p.funcPointerBits != 32 && p.funcPointerBits != 64 {
		return false
	}
	if p.endian != renvoEndianLittle && p.endian != renvoEndianBig {
		return false
	}
	if p.backendSlotSize < 1 || p.maxAlign < 1 {
		return false
	}
	if p.addressModel < renvoAddressModelFlat || p.addressModel > renvoAddressModelBanked {
		return false
	}
	if p.heapModel < renvoHeapNone || p.heapModel > renvoHeapExternal {
		return false
	}
	if p.oomModel < renvoOOMTrap || p.oomModel > renvoOOMPanic {
		return false
	}
	if renvoProfileHasRuntime(p, renvoRuntimeHeap) && p.heapModel == renvoHeapNone {
		return false
	}
	if renvoProfileHasRuntime(p, renvoRuntimeVolatileMemory) && p.volatileWidths == 0 {
		return false
	}
	if renvoProfileHasRuntime(p, renvoRuntimeInterrupts) && p.interruptModel == renvoInterruptNone {
		return false
	}
	if p.floatModel < renvoFloatScaledInteger || p.floatModel > renvoFloatIEEESoft {
		return false
	}
	return true
}

var renvoTargetArch int = renvoArchAmd64
var renvoTargetOS int = renvoOSLinux
var renvoNativeIntSize int = 8
var renvoTarget int = renvoTargetLinuxAmd64
var renvoCompilerWindowsSubsystem int = 3
var renvoCompilerEmitImage bool

const renvoArenaSize64BitHosted = 134217728
const renvoArenaSize32BitHosted = 67108864
const renvoArenaSizeWasi = 33554432
const renvoArenaSizeKernelModule = 65536
const renvoArenaSizeMinimum = 256
const renvoArenaSizeMaximum = 1073741824

func renvoDefaultArenaSize(target int) int {
	if target == renvoTargetLinuxKernelAmd64 {
		return renvoArenaSizeKernelModule
	}
	if target == renvoTargetWasiWasm32 {
		return renvoArenaSizeWasi
	}
	if target > 0 && target < len(renvoTargetIntBitsTable) && int(renvoTargetIntBitsTable[target]) == 32 {
		return renvoArenaSize32BitHosted
	}
	return renvoArenaSize64BitHosted
}

func renvoResolveArenaSize(target int, requested int) int {
	if requested == 0 {
		return renvoDefaultArenaSize(target)
	}
	return requested
}

func renvoSetTarget(target int) {
	if renvoFixedTarget != 0 {
		target = renvoFixedTarget
	}
	renvoTarget = target
	if target >= renvoTargetLinuxAmd64 && target <= renvoTargetVM32 {
		renvoTargetOS = int(targetOSTable[target])
		renvoTargetArch = int(targetArchTable[target])
		renvoNativeIntSize = int(renvoTargetIntBitsTable[target]) / 8
		return
	}
	// Preserve the historical fallback for internal callers that pass an
	// invalid target. Public entry points reject it before reaching this code.
	renvoTargetOS = renvoOSLinux
	renvoTargetArch = renvoArchAmd64
	renvoNativeIntSize = 8
}

func targetIsWindows() bool {
	return renvoTargetOS == renvoOSWindows
}

func targetIsDarwin() bool {
	return renvoTargetOS == renvoOSDarwin
}

const renvoFixedTargetUnknown = -2147483647

func renvoEvalFixedTargetInt(g *renvoLinearGen, ep *renvoExprParse, idx int, fixedTarget int, fixedTargetKnown bool) int {
	renvoNonNil(g, ep)
	p := g.prog
	renvoNonNil(p)
	if idx < 0 || idx >= len(ep.exprs) {
		return renvoFixedTargetUnknown
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		return renvoParseIntToken(p, e.tok)
	}
	if e.kind == renvoExprChar {
		return renvoParseCharToken(p, e.tok)
	}
	if e.kind == renvoExprBool {
		return renvoBoolTokenValue(p, e.tok)
	}
	if e.kind == renvoExprIdent {
		if fixedTarget >= renvoTargetLinuxAmd64 && fixedTarget <= renvoTargetVM32 {
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTargetArch") {
				return int(targetArchTable[fixedTarget])
			}
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTargetOS") {
				return int(targetOSTable[fixedTarget])
			}
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTarget") {
				return fixedTarget
			}
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoNativeIntSize") {
				return int(renvoTargetIntBitsTable[fixedTarget]) / 8
			}
		}
		if fixedTargetKnown && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "renvoFixedTarget") {
			return fixedTarget
		}
		value := renvoFindSmallConstByName(g, e.nameStart, e.nameEnd)
		if value >= -128 {
			return value
		}
	}
	return renvoFixedTargetUnknown
}
func renvoEvalFixedTargetBool(g *renvoLinearGen, ep *renvoExprParse, idx int, fixedTarget int, fixedTargetKnown bool) int {
	renvoNonNil(g, ep)
	if !fixedTargetKnown && fixedTarget == 0 || idx < 0 || idx >= len(ep.exprs) {
		return -1
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprBool {
		return renvoBoolTokenValue(g.prog, e.tok)
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '!') {
		inner := renvoEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
		if inner == 0 {
			return 1
		}
		if inner == 1 {
			return 0
		}
		return -1
	}
	if e.kind == renvoExprCall && e.argCount == 0 && fixedTarget >= renvoTargetLinuxAmd64 && fixedTarget <= renvoTargetVM32 {
		wantOS := 0
		if renvoExprIsIdentText(g.prog, ep, e.left, "targetIsWindows") {
			wantOS = renvoOSWindows
		} else if renvoExprIsIdentText(g.prog, ep, e.left, "targetIsDarwin") {
			wantOS = renvoOSDarwin
		}
		if wantOS != 0 {
			if int(targetOSTable[fixedTarget]) == wantOS {
				return 1
			}
			return 0
		}
	}
	if e.kind == renvoExprCall && e.argCount == 1 && fixedTarget >= renvoTargetLinuxAmd64 && fixedTarget <= renvoTargetVM32 && int(renvoTargetIntBitsTable[fixedTarget]) == 64 && renvoExprIsIdentText(g.prog, ep, e.left, "renvoTypeKindNeedsWideLowering") {
		return 0
	}
	if e.kind == renvoExprBinary {
		if renvoTok2Is(g.prog, e.tok, '&', '&') {
			left := renvoEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
			if left == 0 {
				return 0
			}
			right := renvoEvalFixedTargetBool(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if left == 1 && right == 1 {
				return 1
			}
			if right == 0 {
				return 0
			}
			return -1
		}
		if renvoTok2Is(g.prog, e.tok, '|', '|') {
			left := renvoEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
			if left == 1 {
				return 1
			}
			right := renvoEvalFixedTargetBool(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if left == 0 && right == 0 {
				return 0
			}
			if right == 1 {
				return 1
			}
			return -1
		}
		if renvoTok2Is(g.prog, e.tok, '=', '=') || renvoTok2Is(g.prog, e.tok, '!', '=') {
			left := renvoEvalFixedTargetInt(g, ep, e.left, fixedTarget, fixedTargetKnown)
			right := renvoEvalFixedTargetInt(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if left == renvoFixedTargetUnknown || right == renvoFixedTargetUnknown {
				return -1
			}
			eq := left == right
			if renvoTok2Is(g.prog, e.tok, '!', '=') {
				eq = !eq
			}
			if eq {
				return 1
			}
			return 0
		}
	}
	return -1
}
func renvoObjectContextHash(g *renvoLinearGen) (int, int) {
	p := g.prog
	m := g.meta
	packages := renvoProgramPackages(p)
	a, b := 313, 733
	if len(packages) == 0 {
		position := 0
		count := len(p.funcs)
		for i := 0; i < count; i++ {
			fn := p.funcs[i]
			bodyStart := int(renvoTokEnd(p, fn.bodyStart))
			bodyEnd := int(renvoTokStart(p, fn.bodyEnd))
			if bodyStart < position || bodyEnd < bodyStart || bodyEnd > len(p.src) {
				continue
			}
			a, b = renvoObjectHashRange(a, b, p.src, position, bodyStart)
			position = bodyEnd
		}
		a, b = renvoObjectHashRange(a, b, p.src, position, len(p.src))
	}
	a, b = renvoObjectHashInt(a, b, renvoTarget)
	a, b = renvoObjectHashInt(a, b, renvoCompilerWindowsSubsystem)
	a, b = renvoObjectHashInt(a, b, m.arenaSize)
	for i := 0; i < len(m.types); i++ {
		t := m.types[i]
		a, b = renvoObjectHashInt(a, b, t.kind)
		a, b = renvoObjectHashInt(a, b, t.elem)
		a, b = renvoObjectHashInt(a, b, t.first)
		a, b = renvoObjectHashInt(a, b, t.count)
		a, b = renvoObjectHashInt(a, b, t.size)
		a, b = renvoObjectHashRange(a, b, p.src, t.nameStart, t.nameEnd)
	}
	for i := 0; i < len(m.fields); i++ {
		field := m.fields[i]
		a, b = renvoObjectHashInt(a, b, field.typ)
		a, b = renvoObjectHashInt(a, b, field.offset)
		embedded := 0
		if field.embedded {
			embedded = 1
		}
		a, b = renvoObjectHashInt(a, b, embedded)
		a, b = renvoObjectHashRange(a, b, p.src, field.nameStart, field.nameEnd)
	}
	for i := 0; i < len(m.globals); i++ {
		global := m.globals[i]
		a, b = renvoObjectHashInt(a, b, global.kind)
		a, b = renvoObjectHashInt(a, b, global.typ)
		a, b = renvoObjectHashInt(a, b, global.iotaValue)
		a, b = renvoObjectHashRange(a, b, p.src, global.nameStart, global.nameEnd)
	}
	if len(packages) == 0 {
		for i := 0; i < len(m.funcs); i++ {
			fn := m.funcs[i]
			a, b = renvoObjectHashRange(a, b, p.src, fn.nameStart, fn.nameEnd)
			a, b = renvoObjectHashInt(a, b, fn.paramCount)
			a, b = renvoObjectHashInt(a, b, fn.resultCount)
			a, b = renvoObjectHashInt(a, b, fn.resultType)
			a, b = renvoObjectHashInt(a, b, fn.receiverType)
			a, b = renvoObjectHashInt(a, b, fn.linkStatic)
			for j := 0; j < fn.paramCount; j++ {
				a, b = renvoObjectHashInt(a, b, m.params[fn.firstParam+j].typ)
			}
		}
	}
	for i := 0; i < len(m.captures); i++ {
		capture := m.captures[i]
		a, b = renvoObjectHashInt(a, b, capture.typ)
		a, b = renvoObjectHashRange(a, b, p.src, capture.nameStart, capture.nameEnd)
	}
	return a, b
}
func renvoAsmSetDataOffsets(a *renvoAsm) {
	renvoNonNil(a)
	a.dataOffset = a.codeOffset + len(a.code)
	if renvoTargetOS == renvoOSLinux {
		a.bssOffset = renvoAlignValue(a.dataOffset+len(a.data), 0x1000)
	}
}
