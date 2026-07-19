package main

const renvoAbsBssReloc = 1
const renvoAbsWinImportReloc = 2

const renvoTargetLinuxAmd64 = 1
const renvoTargetLinux386 = 2
const renvoTargetLinuxAarch64 = 3
const renvoTargetLinuxArm = 4
const renvoTargetWindowsAmd64 = 5
const renvoTargetWindows386 = 6
const renvoTargetWasiWasm32 = 7
const renvoTargetDarwinArm64 = 8
const renvoTargetWindowsArm64 = 9

const renvoArchAmd64 = 1
const renvoArch386 = 2
const renvoArchAarch64 = 3
const renvoArchArm = 4
const renvoArchWasm32 = 5

const renvoOSLinux = 1
const renvoOSWindows = 2
const renvoOSDarwin = 3
const renvoOSWasi = 4

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
const targetOSTable = "\x00\x01\x01\x01\x01\x02\x02\x04\x03\x02"
const targetArchTable = "\x00\x01\x02\x03\x04\x01\x02\x05\x03\x03"
const renvoTargetIntBitsTable = "\x00\x40\x20\x40\x20\x40\x20\x20\x40\x40"

func renvoProfileForTarget(target int) (renvoTargetProfile, bool) {
	var p renvoTargetProfile
	if target < renvoTargetLinuxAmd64 || target > renvoTargetWindowsArm64 {
		return p, false
	}
	p.target = target
	p.os = int(targetOSTable[target])
	p.arch = int(targetArchTable[target])
	p.intBits = int(renvoTargetIntBitsTable[target])
	p.pointerBits = p.intBits
	p.maxAlign = p.intBits / 8
	if target == renvoTargetWasiWasm32 {
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

const renvoArenaSize64BitHosted = 134217728
const renvoArenaSize32BitHosted = 67108864
const renvoArenaSizeWasi = 33554432
const renvoArenaSizeMinimum = 256
const renvoArenaSizeMaximum = 1073741824

func renvoDefaultArenaSize(target int) int {
	if target == renvoTargetWasiWasm32 {
		return renvoArenaSizeWasi
	}
	if target > 0 && target < len(renvoTargetIntBitsTable) && int(renvoTargetIntBitsTable[target]) == 32 {
		return renvoArenaSize32BitHosted
	}
	return renvoArenaSize64BitHosted
}

func renvoArenaSizeValid(target int, size int) bool {
	if target <= 0 || target >= len(renvoTargetIntBitsTable) {
		return false
	}
	return size >= renvoArenaSizeMinimum && size <= renvoArenaSizeMaximum
}

func renvoResolveArenaSize(target int, requested int) int {
	if requested == 0 {
		return renvoDefaultArenaSize(target)
	}
	return requested
}

// These bodies are used by the host Go build. Self-hosted compilers lower the
// calls as arena intrinsics so large, phase-local scratch data can be reclaimed.
func renvo_runtime_ArenaMark() int { return 0 }

func renvo_runtime_ArenaReset(mark int) {}

// These internal intrinsics are only used after compiler code has established
// the corresponding index invariant. Host builds retain Go's checked access;
// self-hosted builds lower the calls to an explicitly unsafe load.
func renvo_runtime_UnsafeByteAt(data []byte, index int) byte { return data[index] }

func renvo_runtime_UnsafeInt32At(data []int32, index int) int32 { return data[index] }

func renvo_runtime_UnsafeIntAt(data []int, index int) int { return data[index] }

func renvoNonNil(values ...interface{}) {}

func renvoSetTarget(target int) {
	if renvoFixedTarget != 0 {
		target = renvoFixedTarget
	}
	renvoTarget = target
	if target >= renvoTargetLinuxAmd64 && target <= renvoTargetWindowsArm64 {
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
	nameStart int
	nameEnd   int
	label     int
}

type renvoWinStaticImport struct {
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
	code                []byte
	labelPos            []int
	labelSet            []bool
	relocs              []renvoLabelRef
	absRelocs           []renvoAbsRef
	symbols             []renvoAsmSymbol
	symbolName          []byte
	winImports          []renvoWinStaticImport
	darwinImports       []renvoDarwinStaticImport
	darwinImportLabels  []int
	darwinImportUsed    []bool
	data                []byte
	bssSize             int
	codeOffset          int
	dataOffset          int
	bssOffset           int
	lastPrimaryStoreEnd int
	lastPrimaryStoreOff int
	lastPrimaryLoad     int
}

const renvoWasm32FallbackSliceBackingSize = 4096

func renvoAsmInit(a *renvoAsm) {
	renvoNonNil(a)
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []renvoLabelRef
	var absRelocs []renvoAbsRef
	var symbols []renvoAsmSymbol
	var symbolName []byte
	var winImports []renvoWinStaticImport
	var darwinImports []renvoDarwinStaticImport
	var data []byte
	if renvoFixedTarget != 0 {
		code = make([]byte, 0, 2097152)
		labelPos = make([]int, 0, 16384)
		labelSet = make([]bool, 0, 16384)
		relocs = make([]renvoLabelRef, 0, 32768)
		absRelocs = make([]renvoAbsRef, 0, 16384)
		symbols = make([]renvoAsmSymbol, 0, 1024)
		if renvoCompilerStripSymbols && renvoTargetArch != renvoArchWasm32 {
			symbols = make([]renvoAsmSymbol, 0, 0)
		}
	} else if renvoTargetArch == renvoArchWasm32 {
		code = make([]byte, 0, 655360)
		labelPos = make([]int, 0, 32768)
		labelSet = make([]bool, 0, 32768)
		relocs = make([]renvoLabelRef, 0, 65536)
		absRelocs = make([]renvoAbsRef, 0, 32768)
		symbols = make([]renvoAsmSymbol, 0, 2048)
	} else {
		code = make([]byte, 0, 2097152)
		labelPos = make([]int, 0, 32768)
		labelSet = make([]bool, 0, 32768)
		relocs = make([]renvoLabelRef, 0, 65536)
		absRelocs = make([]renvoAbsRef, 0, 32768)
		symbols = make([]renvoAsmSymbol, 0, 4096)
	}
	data = make([]byte, 0, 16384)
	if renvoCompilerStripSymbols && renvoTargetArch != renvoArchWasm32 {
		symbolName = make([]byte, 0, 0)
	} else {
		symbolName = make([]byte, 0, 16384)
	}
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.absRelocs = absRelocs
	a.symbols = symbols
	a.symbolName = symbolName
	a.winImports = winImports
	a.darwinImports = darwinImports
	a.data = data
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
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	return label
}

func renvoAsmMarkLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if label < 0 {
		return
	}
	if label >= len(a.labelPos) || label >= len(a.labelSet) {
		return
	}
	codeLen := len(a.code)
	a.labelPos[label] = codeLen
	a.labelSet[label] = true
	a.lastPrimaryStoreEnd = -1
	a.lastPrimaryLoad = 0
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
	a.absRelocs = append(a.absRelocs, renvoAbsRef{at: at & 2147483647, off: off & 2147483647, kind: kind & 2147483647})
}

func renvoAsmAddReloc(a *renvoAsm, at int, label int) {
	renvoNonNil(a)
	a.relocs = append(a.relocs, renvoLabelRef{at: at & 2147483647, label: label & 2147483647})
}

func renvoAsmAddFuncSymbol(a *renvoAsm, src []byte, nameStart int, nameEnd int, label int) {
	renvoNonNil(a)
	if renvoCompilerStripSymbols && renvoTargetArch != renvoArchWasm32 {
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

func renvoAsmEmit32(a *renvoAsm, v int) {
	renvoNonNil(a)
	a.code = renvoAppend32(a.code, v)
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

func renvoAmd64RelaxBranches(a *renvoAsm) {
	renvoNonNil(a)
	oldLen := len(a.code)
	// Keep only the positions of branches that become short. The previous
	// representation used one byte plus one int32 for every byte of generated
	// code, then allocated a second full code buffer. Large self-hosted builds
	// therefore retained several megabytes of scratch until process exit.
	branches := make([]int32, 0, len(a.relocs))
	savings := make([]int32, 0, len(a.relocs))
	totalSaving := 0
	for i := 0; i < len(a.relocs); i++ {
		at := a.relocs[i].at & 2147483647
		label := a.relocs[i].label & 2147483647
		if label < 0 || label >= len(a.labelPos) || label >= len(a.labelSet) || !a.labelSet[label] {
			continue
		}
		target := a.labelPos[label]
		disp := target - (at + 4)
		if disp < -128 || disp > 127 {
			continue
		}
		start := -1
		kind := 0
		saving := 0
		if at >= 1 && a.code[at-1] == 0xe9 {
			start = at - 1
			saving = 3
		} else if at >= 2 && a.code[at-2] == 0x0f && a.code[at-1] >= 0x80 && a.code[at-1] <= 0x8f {
			start = at - 2
			kind = 1
			saving = 4
		}
		if start >= 0 {
			if len(branches) > 0 && int(branches[len(branches)-1])/2 >= start {
				continue
			}
			totalSaving += saving
			branches = append(branches, int32(start*2+kind))
			savings = append(savings, int32(totalSaving))
		}
	}
	read := 0
	write := 0
	branch := 0
	for read < oldLen {
		if branch < len(branches) && int(branches[branch])/2 == read {
			kind := int(branches[branch]) & 1
			op := 0xeb
			size := 5
			if kind != 0 {
				op = int(a.code[read+1]) - 0x10
				size = 6
			}
			a.code[write] = byte(op)
			a.code[write+1] = 0
			write += 2
			read += size
			branch++
			continue
		}
		a.code[write] = a.code[read]
		write++
		read++
	}
	renvoTruncBytes(&a.code, write)
	relocCount := 0
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		at := r.at & 2147483647
		label := r.label & 2147483647
		start := at - 1
		if at >= 2 && renvoAmd64RelaxedBranchIndex(branches, at-2) >= 0 {
			start = at - 2
		}
		if start >= 0 && renvoAmd64RelaxedBranchIndex(branches, start) >= 0 && label >= 0 && label < len(a.labelPos) && label < len(a.labelSet) && a.labelSet[label] {
			newAt := renvoAmd64RelaxedPosition(branches, savings, start) + 1
			target := renvoAmd64RelaxedPosition(branches, savings, a.labelPos[label])
			disp := target - (newAt + 1)
			if disp >= -128 && disp <= 127 {
				a.code[newAt] = byte(disp)
				continue
			}
		}
		r.at = renvoAmd64RelaxedPosition(branches, savings, at)
		a.relocs[relocCount] = r
		relocCount++
	}
	a.relocs = a.relocs[:relocCount]
	for i := 0; i < len(a.absRelocs); i++ {
		at := a.absRelocs[i].at & 2147483647
		a.absRelocs[i].at = renvoAmd64RelaxedPosition(branches, savings, at)
	}
	for i := 0; i < len(a.labelPos); i++ {
		if a.labelSet[i] {
			a.labelPos[i] = renvoAmd64RelaxedPosition(branches, savings, a.labelPos[i])
		}
	}
}

func renvoAmd64RelaxedBranchIndex(branches []int32, position int) int {
	lo := 0
	hi := len(branches)
	for lo < hi {
		mid := lo + (hi-lo)/2
		at := int(renvo_runtime_UnsafeInt32At(branches, mid)) / 2
		if at < position {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(branches) && int(renvo_runtime_UnsafeInt32At(branches, lo))/2 == position {
		return lo
	}
	return -1
}

func renvoAmd64RelaxedPosition(branches []int32, savings []int32, position int) int {
	lo := 0
	hi := len(branches)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if int(renvo_runtime_UnsafeInt32At(branches, mid))/2 < position {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo == 0 {
		return position
	}
	return position - int(renvo_runtime_UnsafeInt32At(savings, lo-1))
}

func renvoAsmPatch(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchArm {
		for i := 0; i < len(a.relocs); i++ {
			at := a.relocs[i].at & 2147483647
			label := a.relocs[i].label & 2147483647
			if label < 0 {
				continue
			}
			if label >= len(a.labelPos) || label >= len(a.labelSet) {
				continue
			}
			if !a.labelSet[label] {
				continue
			}
			target := a.labelPos[label]
			disp := target - (at + 8)
			insn := renvoGet32At(a.code, at)
			if (insn & 0x0e000000) == 0x0a000000 {
				renvoPut32At(a.code, at, (insn&0xff000000)|((disp/4)&0x00ffffff))
			}
		}
		renvoAsmSetDataOffsets(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		for i := 0; i < len(a.relocs); i++ {
			at := a.relocs[i].at & 2147483647
			label := a.relocs[i].label & 2147483647
			if label < 0 {
				continue
			}
			if label >= len(a.labelPos) || label >= len(a.labelSet) {
				continue
			}
			if !a.labelSet[label] {
				continue
			}
			target := a.labelPos[label]
			disp := target - at
			insn := renvoGet32At(a.code, at)
			if (insn & 0xfc000000) == 0x94000000 {
				renvoPut32At(a.code, at, 0x94000000|((disp/4)&0x03ffffff))
			} else if (insn & 0xfc000000) == 0x14000000 {
				renvoPut32At(a.code, at, 0x14000000|((disp/4)&0x03ffffff))
			} else if (insn & 0xff000010) == 0x54000000 {
				renvoPut32At(a.code, at, (insn&0xff00001f)|(((disp/4)&0x7ffff)<<5))
			}
		}
		renvoAsmSetDataOffsets(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 {
		renvoAmd64RelaxBranches(a)
	}
	for i := 0; i < len(a.relocs); i++ {
		at := a.relocs[i].at & 2147483647
		label := a.relocs[i].label & 2147483647
		if label < 0 {
			continue
		}
		if label >= len(a.labelPos) || label >= len(a.labelSet) {
			continue
		}
		if !a.labelSet[label] {
			continue
		}
		target := a.labelPos[label]
		disp := target - (at + 4)
		renvoPut32At(a.code, at, disp)
	}
	renvoAsmSetDataOffsets(a)
	for i := 0; i < len(a.absRelocs); i++ {
		at := a.absRelocs[i].at & 2147483647
		off := a.absRelocs[i].off & 2147483647
		kind := a.absRelocs[i].kind & 2147483647
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		next := a.codeOffset + at + 4
		disp := target - next
		renvoPut32At(a.code, at, disp)
	}
}

func renvoAsmSetDataOffsets(a *renvoAsm) {
	renvoNonNil(a)
	a.dataOffset = a.codeOffset + len(a.code)
	if renvoTargetOS == renvoOSLinux {
		a.bssOffset = renvoAlignValue(a.dataOffset+len(a.data), 0x1000)
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
	b0 := byte(v)
	b1 := byte(v >> 8)
	b2 := byte(v >> 16)
	b3 := byte(v >> 24)
	out[at] = b0
	out[at+1] = b1
	out[at+2] = b2
	out[at+3] = b3
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

type renvoElf64SymbolSections struct {
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

func renvoAppendBytesRangeZ(out []byte, s []byte, start int, end int) []byte {
	for i := start; i < end; i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return out
}

func renvoAppendElf64Sym(out []byte, name int, info int, shndx int, value int, size int) []byte {
	out = renvoAppend32(out, name)
	out = append(out, byte(info))
	out = append(out, 0)
	out = renvoAppend16(out, shndx)
	out = renvoAppend64U32(out, value)
	out = renvoAppend64U32(out, size)
	return out
}

func renvoAppendElf64Shdr(out []byte, name int, typ int, flags int, addr int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = renvoAppend32(out, name)
	out = renvoAppend32(out, typ)
	out = renvoAppend64U32(out, flags)
	out = renvoAppend64U32(out, addr)
	out = renvoAppend64U32(out, off)
	out = renvoAppend64U32(out, size)
	out = renvoAppend32(out, link)
	out = renvoAppend32(out, info)
	out = renvoAppend64U32(out, align)
	out = renvoAppend64U32(out, entsize)
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

func renvoBuildElf64SymbolSections(a *renvoAsm, base int, entryOff int, loadFileSize int) renvoElf64SymbolSections {
	renvoNonNil(a)
	var sec renvoElf64SymbolSections
	sec.symtab = make([]byte, 0, (len(a.symbols)+2)*24)
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
	sec.symtab = renvoAppendElf64Sym(sec.symtab, 0, 0, 0, 0, 0)
	sec.symtab = renvoAppendElf64Sym(sec.symtab, startName, 18, 1, base+entryOff, 0)
	for i := 0; i < len(a.symbols); i++ {
		s := a.symbols[i]
		label := s.label
		if label < 0 || label >= len(a.labelPos) || label >= len(a.labelSet) || !a.labelSet[label] {
			continue
		}
		nameOff := len(sec.strtab)
		sec.strtab = renvoAppendBytesRangeZ(sec.strtab, a.symbolName, s.nameStart, s.nameEnd)
		value := base + a.codeOffset + a.labelPos[label]
		sec.symtab = renvoAppendElf64Sym(sec.symtab, nameOff, 18, 1, value, 0)
	}

	sec.symtabOff = renvoAlignValue(loadFileSize, 8)
	sec.strtabOff = sec.symtabOff + len(sec.symtab)
	sec.shstrOff = sec.strtabOff + len(sec.strtab)
	sec.shoff = renvoAlignValue(sec.shstrOff+len(sec.shstrtab), 8)
	return sec
}

func renvoAppendElf64SectionHeaders(out []byte, sec *renvoElf64SymbolSections, a *renvoAsm, base int) []byte {
	renvoNonNil(sec, a)
	codeOff := a.codeOffset
	codeSize := len(a.code)
	dataOff := a.dataOffset
	dataSize := len(a.data)
	bssSize := a.bssSize
	symtabSize := len(sec.symtab)
	strtabSize := len(sec.strtab)
	shstrtabSize := len(sec.shstrtab)

	out = renvoAppendElf64Shdr(out, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	out = renvoAppendElf64Shdr(out, sec.textName, 1, 6, base+codeOff, codeOff, codeSize, 0, 0, 16, 0)
	out = renvoAppendElf64Shdr(out, sec.dataName, 1, 2, base+dataOff, dataOff, dataSize, 0, 0, 8, 0)
	out = renvoAppendElf64Shdr(out, sec.bssName, 8, 3, base+renvoAsmBssOffset(a), renvoAsmBssOffset(a), bssSize, 0, 0, 8, 0)
	out = renvoAppendElf64Shdr(out, sec.symtabName, 2, 0, 0, sec.symtabOff, symtabSize, 5, 1, 8, 24)
	out = renvoAppendElf64Shdr(out, sec.strtabName, 3, 0, 0, sec.strtabOff, strtabSize, 0, 0, 1, 0)
	out = renvoAppendElf64Shdr(out, sec.shstrName, 3, 0, 0, sec.shstrOff, shstrtabSize, 0, 0, 1, 0)
	return out
}

type renvoElf32SymbolSections struct {
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

func renvoAppendElf32Sym(out []byte, name int, info int, shndx int, value int, size int) []byte {
	out = renvoAppend32(out, name)
	out = renvoAppend32(out, value)
	out = renvoAppend32(out, size)
	out = append(out, byte(info))
	out = append(out, 0)
	out = renvoAppend16(out, shndx)
	return out
}

func renvoAppendElf32Shdr(out []byte, name int, typ int, flags int, addr int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = renvoAppend32(out, name)
	out = renvoAppend32(out, typ)
	out = renvoAppend32(out, flags)
	out = renvoAppend32(out, addr)
	out = renvoAppend32(out, off)
	out = renvoAppend32(out, size)
	out = renvoAppend32(out, link)
	out = renvoAppend32(out, info)
	out = renvoAppend32(out, align)
	out = renvoAppend32(out, entsize)
	return out
}

func renvoBuildElf32SymbolSections(a *renvoAsm, base int, entryOff int, loadFileSize int) renvoElf32SymbolSections {
	renvoNonNil(a)
	var sec renvoElf32SymbolSections
	sec.symtab = make([]byte, 0, (len(a.symbols)+2)*16)
	sec.strtab = make([]byte, 0, len(a.symbolName)+16)
	sec.shstrtab = make([]byte, 0, 64)
	sec.shstrtab = append(sec.shstrtab, 0)
	sec.textName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".text")
	sec.dataName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".rodata")
	sec.bssName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".bss")
	sec.symtabName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".symtab")
	sec.strtabName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".strtab")
	sec.shstrName = len(sec.shstrtab)
	sec.shstrtab = renvoAppendStringZ(sec.shstrtab, ".shstrtab")

	sec.strtab = append(sec.strtab, 0)
	startName := len(sec.strtab)
	sec.strtab = renvoAppendStringZ(sec.strtab, "_start")
	sec.symtab = renvoAppendElf32Sym(sec.symtab, 0, 0, 0, 0, 0)
	sec.symtab = renvoAppendElf32Sym(sec.symtab, startName, 18, 1, base+entryOff, 0)
	for i := 0; i < len(a.symbols); i++ {
		s := a.symbols[i]
		label := s.label
		if label < 0 {
			continue
		}
		if label >= len(a.labelPos) || label >= len(a.labelSet) {
			continue
		}
		if !a.labelSet[label] {
			continue
		}
		nameOff := len(sec.strtab)
		sec.strtab = renvoAppendBytesRangeZ(sec.strtab, a.symbolName, s.nameStart, s.nameEnd)
		value := base + a.codeOffset + a.labelPos[label]
		sec.symtab = renvoAppendElf32Sym(sec.symtab, nameOff, 18, 1, value, 0)
	}

	sec.symtabOff = renvoAlignValue(loadFileSize, 4)
	sec.strtabOff = sec.symtabOff + len(sec.symtab)
	sec.shstrOff = sec.strtabOff + len(sec.strtab)
	sec.shoff = renvoAlignValue(sec.shstrOff+len(sec.shstrtab), 4)
	return sec
}

func renvoAppendElf32SectionHeaders(out []byte, sec *renvoElf32SymbolSections, a *renvoAsm, base int) []byte {
	renvoNonNil(sec, a)
	codeOff := a.codeOffset
	codeSize := len(a.code)
	dataOff := a.dataOffset
	dataSize := len(a.data)
	bssSize := a.bssSize
	symtabSize := len(sec.symtab)
	strtabSize := len(sec.strtab)
	shstrtabSize := len(sec.shstrtab)

	out = renvoAppendElf32Shdr(out, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	out = renvoAppendElf32Shdr(out, sec.textName, 1, 6, base+codeOff, codeOff, codeSize, 0, 0, 16, 0)
	out = renvoAppendElf32Shdr(out, sec.dataName, 1, 2, base+dataOff, dataOff, dataSize, 0, 0, 4, 0)
	out = renvoAppendElf32Shdr(out, sec.bssName, 8, 3, base+renvoAsmBssOffset(a), renvoAsmBssOffset(a), bssSize, 0, 0, 4, 0)
	out = renvoAppendElf32Shdr(out, sec.symtabName, 2, 0, 0, sec.symtabOff, symtabSize, 5, 1, 4, 16)
	out = renvoAppendElf32Shdr(out, sec.strtabName, 3, 0, 0, sec.strtabOff, strtabSize, 0, 0, 1, 0)
	out = renvoAppendElf32Shdr(out, sec.shstrName, 3, 0, 0, sec.shstrOff, shstrtabSize, 0, 0, 1, 0)
	return out
}

const renvoWinImageBase = 0x400000
const renvoWinSectionRVA = 0x1000
const renvoWinHeadersSize = 0x200
const renvoWinFileAlign = 0x200
const renvoWinSectionAlign = 0x1000

const renvoWinImportCreateFileA = 1
const renvoWinImportCloseHandle = 2
const renvoWinImportReadFile = 3
const renvoWinImportWriteFile = 4
const renvoWinImportSetFilePointer = 5
const renvoWinImportGetStdHandle = 6
const renvoWinImportGetCommandLineA = 7
const renvoWinImportExitProcess = 8
const renvoWinImportGetEnvironmentStringsA = 9
const renvoWinImportFixedCount = 9

type renvoWinImportLayout struct {
	importRVA    int
	importSize   int
	kernelIATRVA int
	thunkSize    int
	iatRVAs      []int
}

func renvoWinImportName(id int) string {
	if id == renvoWinImportCreateFileA {
		return "CreateFileA"
	}
	if id == renvoWinImportCloseHandle {
		return "CloseHandle"
	}
	if id == renvoWinImportReadFile {
		return "ReadFile"
	}
	if id == renvoWinImportWriteFile {
		return "WriteFile"
	}
	if id == renvoWinImportSetFilePointer {
		return "SetFilePointer"
	}
	if id == renvoWinImportGetStdHandle {
		return "GetStdHandle"
	}
	if id == renvoWinImportGetCommandLineA {
		return "GetCommandLineA"
	}
	if id == renvoWinImportGetEnvironmentStringsA {
		return "GetEnvironmentStringsA"
	}
	return "ExitProcess"
}

func renvoWinImportIATRVA(layout renvoWinImportLayout, id int) int {
	if id >= 0 && id < len(layout.iatRVAs) && layout.iatRVAs[id] != 0 {
		return layout.iatRVAs[id]
	}
	return layout.kernelIATRVA + (id-1)*layout.thunkSize
}

func renvoAsmAddWinImportReloc(a *renvoAsm, at int, importID int) {
	renvoNonNil(a)
	renvoAsmAddAbsReloc(a, at, importID, renvoAbsWinImportReloc)
}

func renvoAsmAddWinStaticImport(a *renvoAsm, dllStart int, dllEnd int, nameStart int, nameEnd int, src []byte) int {
	renvoNonNil(a)
	dll := renvoStringFromBytes(src, dllStart, dllEnd)
	name := renvoStringFromBytes(src, nameStart, nameEnd)
	for i := 0; i < len(a.winImports); i++ {
		imp := a.winImports[i]
		if imp.dll == dll && imp.name == name {
			return renvoWinImportFixedCount + 1 + i
		}
	}
	a.winImports = append(a.winImports, renvoWinStaticImport{dll: dll, name: name})
	return renvoWinImportFixedCount + len(a.winImports)
}

func renvoStringFromBytes(src []byte, start int, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	out := renvoFixedByteScratch(end - start)
	for i := start; i < end; i++ {
		out = append(out, renvo_runtime_UnsafeByteAt(src, i))
	}
	return string(out)
}

func renvoAsmHasWinImportRelocs(a *renvoAsm) bool {
	renvoNonNil(a)
	for i := 0; i < len(a.absRelocs); i++ {
		if a.absRelocs[i].kind == renvoAbsWinImportReloc {
			return true
		}
	}
	return false
}

func renvoAppendWinImports(a *renvoAsm) renvoWinImportLayout {
	renvoNonNil(a)
	var layout renvoWinImportLayout
	thunkSize := 4
	if renvoTargetArch != renvoArch386 {
		thunkSize = 8
	}
	layout.thunkSize = thunkSize
	dataRVA := a.dataOffset
	if dataRVA == 0 {
		dataRVA = a.codeOffset + len(a.code)
	}
	layout.iatRVAs = make([]int, renvoWinImportFixedCount+len(a.winImports)+1)
	groupCount := len(a.winImports) + 1
	importOff := renvoAlignValue(len(a.data), thunkSize)
	a.data = renvoAppendUntil(a.data, importOff)
	descOff := importOff
	kernelILTOff := descOff + (groupCount+1)*20
	kernelIATOff := kernelILTOff + (renvoWinImportFixedCount+1)*thunkSize
	customTablesOff := kernelIATOff + (renvoWinImportFixedCount+1)*thunkSize
	tableOff := customTablesOff + len(a.winImports)*4*thunkSize
	a.data = renvoAppendUntil(a.data, tableOff)

	for id := 1; id <= renvoWinImportFixedCount; id++ {
		renvoAppendWinImportEntry(a, &layout, kernelILTOff, kernelIATOff, dataRVA, id, id-1, renvoWinImportName(id))
	}
	for i := 0; i < len(a.winImports); i++ {
		imp := a.winImports[i]
		iltOff := customTablesOff + i*4*thunkSize
		renvoAppendWinImportEntry(a, &layout, iltOff, iltOff+2*thunkSize, dataRVA, renvoWinImportFixedCount+1+i, 0, imp.name)
	}
	for i := 0; i < groupCount; i++ {
		iltOff := kernelILTOff
		iatOff := kernelIATOff
		dllNameOff := len(a.data)
		dll := "kernel32.dll"
		if i > 0 {
			dll = a.winImports[i-1].dll
			iltOff = customTablesOff + (i-1)*4*thunkSize
			iatOff = iltOff + 2*thunkSize
		}
		a.data = renvoAppendStringZ(a.data, dll)
		at := descOff + i*20
		renvoPut32At(a.data, at, dataRVA+iltOff)
		renvoPut32At(a.data, at+12, dataRVA+dllNameOff)
		renvoPut32At(a.data, at+16, dataRVA+iatOff)
	}

	layout.importRVA = dataRVA + importOff
	layout.importSize = len(a.data) - importOff
	layout.kernelIATRVA = layout.iatRVAs[renvoWinImportGetStdHandle]
	return layout
}

func renvoAppendWinImportEntry(a *renvoAsm, layout *renvoWinImportLayout, iltOff int, iatOff int, dataRVA int, id int, slot int, name string) {
	renvoNonNil(a, layout)
	nameAt := len(a.data)
	a.data = renvoAppend16(a.data, 0)
	a.data = renvoAppendStringZ(a.data, name)
	if len(a.data)&1 != 0 {
		a.data = append(a.data, 0)
	}
	nameRVA := dataRVA + nameAt
	iltAt := iltOff + slot*layout.thunkSize
	iatAt := iatOff + slot*layout.thunkSize
	if renvoTargetArch != renvoArch386 {
		renvoPut32At(a.data, iltAt, nameRVA)
		renvoPut32At(a.data, iatAt, nameRVA)
	} else {
		renvoPut32At(a.data, iltAt, nameRVA)
		renvoPut32At(a.data, iatAt, nameRVA)
	}
	layout.iatRVAs[id] = dataRVA + iatAt
}

func renvoAsmPatchWindows(a *renvoAsm, layout renvoWinImportLayout) {
	renvoNonNil(a)
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		label := r.label
		if label < 0 {
			continue
		}
		if label >= len(a.labelPos) || label >= len(a.labelSet) {
			continue
		}
		if !a.labelSet[label] {
			continue
		}
		target := a.labelPos[label]
		disp := target - (r.at + 4)
		renvoPut32At(a.code, r.at, disp)
	}
	if a.dataOffset == 0 {
		a.dataOffset = a.codeOffset + len(a.code)
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		if r.kind == renvoAbsWinImportReloc {
			target := renvoWinImportIATRVA(layout, r.off)
			if renvoTargetArch != renvoArch386 {
				next := a.codeOffset + r.at + 4
				renvoPut32At(a.code, r.at, target-next)
			} else {
				renvoPut32At(a.code, r.at, renvoWinImageBase+target)
			}
			continue
		}
		target := a.dataOffset + r.off
		if r.kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + r.off
		}
		if renvoTargetArch != renvoArch386 {
			next := a.codeOffset + r.at + 4
			renvoPut32At(a.code, r.at, target-next)
		} else {
			renvoPut32At(a.code, r.at, renvoWinImageBase+target)
		}
	}
}

func renvoAppendPEHeader64(out []byte, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	machine := 0x8664
	imageBase := renvoWinImageBase
	stackReserve := 0x100000
	stackCommit := 0x100000
	if renvoTargetArch == renvoArchAarch64 {
		machine = 0xaa64
		imageBase = 0x140000000
		stackReserve = 0x800000
		stackCommit = 0x1000
	}
	sizeOfImage := renvoAlignValue(dataRVA+dataVirtualSize, renvoWinSectionAlign)
	out = renvoAppend32(out, 0x5a4d)
	out = renvoAppendUntil(out, 0x3c)
	out = renvoAppend32(out, 0x80)
	out = renvoAppendUntil(out, 0x80)
	pe := len(out)
	// Start with the invariant COFF and PE32+ fields. The zero placeholders are
	// patched below; representing the fixed layout as data keeps fixed-target
	// Windows compilers within the binary-size budget.
	out = renvoAppendStringZ(out, "\x50\x45\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xf0\x00\x22\x00\x0b\x02\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x02\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00")
	renvoTruncBytes(&out, len(out)-1)
	out = renvoAppendUntil(out, pe+24+240)
	renvoPut32At(out, pe+4, 0x00020000|machine)
	opt := pe + 24
	renvoPut32At(out, opt+4, textRawSize)
	renvoPut32At(out, opt+8, dataRawSize)
	renvoPut32At(out, opt+16, renvoWinSectionRVA)
	renvoPut32At(out, opt+24, imageBase)
	renvoPut32At(out, opt+28, imageBase>>32)
	renvoPut32At(out, opt+56, sizeOfImage)
	renvoPut32At(out, opt+68, 0x01000000|renvoCompilerWindowsSubsystem)
	renvoPut32At(out, opt+72, stackReserve)
	renvoPut32At(out, opt+80, stackCommit)
	renvoPut32At(out, opt+120, importRVA)
	renvoPut32At(out, opt+124, importSize)
	renvoPut32At(out, opt+208, iatRVA)
	renvoPut32At(out, opt+212, iatSize)
	out = renvoAppendPESection(out, ".text", textVirtualSize, renvoWinSectionRVA, textRawSize, renvoWinHeadersSize, 0x60000020)
	out = renvoAppendPESection(out, ".data", dataVirtualSize, dataRVA, dataRawSize, renvoWinHeadersSize+textRawSize, 0xc0000040)
	out = renvoAppendUntil(out, renvoWinHeadersSize)
	return out
}

func renvoAppendPEHeader32(out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	sizeOfImage := renvoAlignValue(dataRVA+dataVirtualSize, renvoWinSectionAlign)
	out = renvoAppendDOSStub(out)
	out = renvoAppend32(out, 0x4550)
	out = renvoAppend32(out, 0x0002014c)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0x010200e0)
	out = renvoAppend32(out, 0x0001010b)
	out = renvoAppend32(out, textRawSize)
	out = renvoAppend32(out, dataRawSize)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, entryRVA)
	out = renvoAppend32(out, renvoWinSectionRVA)
	out = renvoAppend32(out, dataRVA)
	out = renvoAppend32(out, renvoWinImageBase)
	out = renvoAppend32(out, renvoWinSectionAlign)
	out = renvoAppend32(out, renvoWinFileAlign)
	out = renvoAppend64U32(out, 4)
	out = renvoAppend64U32(out, 4)
	out = renvoAppend32(out, sizeOfImage)
	out = renvoAppend32(out, renvoWinHeadersSize)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0x01000000|renvoCompilerWindowsSubsystem)
	out = renvoAppend32(out, 0x100000)
	out = renvoAppend32(out, 0x100000)
	out = renvoAppend32(out, 0x100000)
	out = renvoAppend32(out, 0x1000)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 16)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, importRVA)
	out = renvoAppend32(out, importSize)
	for i := 2; i < 12; i++ {
		out = renvoAppend32(out, 0)
		out = renvoAppend32(out, 0)
	}
	out = renvoAppend32(out, iatRVA)
	out = renvoAppend32(out, iatSize)
	for i := 13; i < 16; i++ {
		out = renvoAppend32(out, 0)
		out = renvoAppend32(out, 0)
	}
	out = renvoAppendPESection(out, ".text", textVirtualSize, renvoWinSectionRVA, textRawSize, renvoWinHeadersSize, 0x60000020)
	out = renvoAppendPESection(out, ".data", dataVirtualSize, dataRVA, dataRawSize, renvoWinHeadersSize+textRawSize, 0xc0000040)
	out = renvoAppendUntil(out, renvoWinHeadersSize)
	return out
}

func renvoAppendDOSStub(out []byte) []byte {
	out = renvoAppend32(out, 0x5a4d)
	out = renvoAppendUntil(out, 0x3c)
	out = renvoAppend32(out, 0x80)
	out = renvoAppendUntil(out, 0x80)
	return out
}

func renvoAppendPESection(out []byte, name string, virtualSize int, rva int, rawSize int, rawPtr int, characteristics int) []byte {
	start := len(out)
	out = renvoAppendUntil(out, start+40)
	for i := 0; i < len(name); i++ {
		out[start+i] = name[i]
	}
	renvoPut32At(out, start+8, virtualSize)
	renvoPut32At(out, start+12, rva)
	renvoPut32At(out, start+16, rawSize)
	renvoPut32At(out, start+20, rawPtr)
	renvoPut32At(out, start+36, characteristics)
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
	count        int
	panicEnabled bool
}

type renvoToken struct {
	start int
	end   int
}

const renvoTokenStride = 3

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
	return int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride+1))
}

func renvoTokEnd(p *renvoProgram, i int) int {
	renvoNonNil(p)
	return int(renvo_runtime_UnsafeInt32At(p.toks.data, i*renvoTokenStride+2))
}

func renvoTokLine(p *renvoProgram, i int) int {
	renvoNonNil(p)
	data := p.toks.data
	return int(renvo_runtime_UnsafeInt32At(data, i*renvoTokenStride)) >> 8 & 65535
}

func renvoTokAt(p *renvoProgram, i int) renvoToken {
	renvoNonNil(p)
	var tok renvoToken
	base := i * renvoTokenStride
	data := p.toks.data
	tok.start = int(renvo_runtime_UnsafeInt32At(data, base+1))
	tok.end = int(renvo_runtime_UnsafeInt32At(data, base+2))
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

type renvoProgram struct {
	src   []byte
	toks  renvoTokens
	decls []renvoDecl
	funcs []renvoFuncDecl
	ok    bool
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
}

const renvoStmtWordCount = 11

var renvoBodyStmtData []int

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
const renvoTypeMap = 15
const renvoTypeUint16 = 16
const renvoTypeUint32 = 17
const renvoTypeUint64 = 18
const renvoTypeFunc = 19
const renvoTypeInterface = 20
const renvoTypeComplex = 21
const renvoNamedTypeAlias = 1

const renvoBuiltinTypeUint16 = 10
const renvoBuiltinTypeUint32 = 11
const renvoBuiltinTypeUint64 = 12
const renvoBuiltinTypeComplex = 13
const renvoBuiltinTypeInterface = 14

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
}

type renvoClosureInfo struct {
	fnIndex      int
	firstCapture int
	captureCount int
	ready        bool
}

type renvoDeferSite struct {
	funcType int
}

type renvoMeta struct {
	prog          *renvoProgram
	types         []renvoTypeInfo
	fields        []renvoFieldInfo
	globals       []renvoSymbolInfo
	params        []renvoSymbolInfo
	funcs         []renvoFuncInfo
	globalBuckets []int
	globalNext    []int
	funcBuckets   []int
	funcNext      []int
	typeBuckets   []int
	closures      []renvoClosureInfo
	captures      []renvoSymbolInfo
	panicEnabled  bool
	arenaSize     int
	ok            bool
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
	renvoParseProgramInto(src, &p)
	return p
}

func renvoParseProgramInto(src []byte, p *renvoProgram) {
	renvoNonNil(p)
	var zero renvoProgram
	*p = zero
	p.src = src
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
		if renvoTokCharIs(p, i, '{') {
			depth++
		} else if renvoTokCharIs(p, i, '}') {
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
		c := renvoTokSingleChar(p, i)
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
	}
	toks.data = make([]int32, 0, tokenCap*renvoTokenStride)
	i := 0
	line := 1
	for i < srcLen {
		c := renvo_runtime_UnsafeByteAt(src, i)
		if c >= '\t' && c <= ' ' && (0x800011>>(c-'\t'))&1 != 0 {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '/' && i+1 < srcLen && renvo_runtime_UnsafeByteAt(src, i+1) == '/' {
			i += 2
			for i < srcLen && renvo_runtime_UnsafeByteAt(src, i) != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < srcLen && renvo_runtime_UnsafeByteAt(src, i+1) == '*' {
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
				for i < srcLen && renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 {
					i++
				}
				if i < srcLen && renvo_runtime_UnsafeByteAt(src, i) == '.' {
					kind = renvoTokFloat
					i++
					for i < srcLen && renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 {
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
		if c == '"' {
			start := i
			i++
			for i < srcLen && renvo_runtime_UnsafeByteAt(src, i) != '"' {
				if renvo_runtime_UnsafeByteAt(src, i) == '\\' && i+1 < srcLen {
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
			renvoScanAppendToken(toks, renvoTokString, start, i-start, line)
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
	toks.data = append(toks.data, int32(kind|line<<8), int32(start), int32(start+size))
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
		if h == 627 {
			return renvoTokIf
		}
	}
	if n == 3 {
		if h == 3549 {
			return renvoTokVar
		}
		if h == 3219 {
			return renvoTokFor
		}
	}
	if n == 4 {
		if h == 18186 {
			return renvoTokType
		}
		if h == 16324 {
			return renvoTokFunc
		}
		if h == 16001 {
			return renvoTokElse
		}
		if h == 16341 {
			return renvoTokGoto
		}
		if h == 15476 {
			return renvoTokCase
		}
	}
	if n == 5 {
		if h == 78294 || h == 85499 {
			toks.panicEnabled = true
		}
		if h == 79191 {
			return renvoTokConst
		}
		if h == 78617 {
			return renvoTokBreak
		}
	}
	if n == 6 {
		if h == 449661 {
			return renvoTokStruct
		}
		if h == 437480 {
			return renvoTokReturn
		}
		if h == 450374 {
			return renvoTokSwitch
		}
	}
	if n == 7 {
		if h == 2176194 {
			toks.panicEnabled = true
		}
		if h == 2131416 {
			return renvoTokPackage
		}
		if h == 1957581 {
			return renvoTokDefault
		}
	}
	if n == 8 {
		if h == 9901561 {
			return renvoTokContinue
		}
	}
	return renvoTokIdent
}

func renvoTokIsKind(p *renvoProgram, i int, kind int) bool {
	renvoNonNil(p)
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i * renvoTokenStride
	if base >= len(data) {
		return false
	}
	return int(renvo_runtime_UnsafeInt32At(data, base))&255 == kind
}

func renvoTokIdentIs(p *renvoProgram, i int, text string) bool {
	renvoNonNil(p)
	data := p.toks.data
	base := i * renvoTokenStride
	if i < 0 || base >= len(data) || int(renvo_runtime_UnsafeInt32At(data, base))&255 != renvoTokIdent {
		return false
	}
	start := int(renvo_runtime_UnsafeInt32At(data, base+1))
	if int(renvo_runtime_UnsafeInt32At(data, base+2))-start != len(text) {
		return false
	}
	for j := 0; j < len(text); j++ {
		if renvo_runtime_UnsafeByteAt(p.src, start+j) != text[j] {
			return false
		}
	}
	return true
}

func renvoTokSingleChar(p *renvoProgram, i int) byte {
	renvoNonNil(p)
	if i < 0 {
		return 0
	}
	data := p.toks.data
	base := i * renvoTokenStride
	if base >= len(data) {
		return 0
	}
	return byte(int(renvo_runtime_UnsafeInt32At(data, base)) >> 24)
}

func renvoTokCharIs(p *renvoProgram, i int, c byte) bool {
	renvoNonNil(p)
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i * renvoTokenStride
	return base < len(data) && byte(int(renvo_runtime_UnsafeInt32At(data, base))>>24) == c
}

func renvoTok2Is(p *renvoProgram, i int, a byte, b byte) bool {
	renvoNonNil(p)
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i * renvoTokenStride
	if base >= len(data) {
		return false
	}
	if int(renvo_runtime_UnsafeInt32At(data, base))&255 != renvoTokOp {
		return false
	}
	start := int(renvo_runtime_UnsafeInt32At(data, base+1))
	size := int(renvo_runtime_UnsafeInt32At(data, base+2)) - start
	if size != 2 {
		return false
	}
	if renvo_runtime_UnsafeByteAt(p.src, start) != a {
		return false
	}
	return renvo_runtime_UnsafeByteAt(p.src, start+1) == b
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
	e := ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return 0
	}
	entry := renvoIdentEntry(p.src, e.nameStart, e.nameEnd)
	if entry == 0 {
		return 0
	}
	return int(renvoIdentCodes[entry-1])
}

func renvoBytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
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
	for i < end {
		if renvo_runtime_UnsafeByteAt(src, i) == '\\' && i+1 < end {
			i++
			if renvo_runtime_UnsafeByteAt(src, i) == 'x' && i+2 < end {
				hi := renvoHexDigitValue(renvo_runtime_UnsafeByteAt(src, i+1))
				lo := renvoHexDigitValue(renvo_runtime_UnsafeByteAt(src, i+2))
				if hi >= 0 && lo >= 0 {
					out = append(out, byte(hi*16+lo))
					i += 3
					continue
				}
			}
			if renvo_runtime_UnsafeByteAt(src, i) == 'n' {
				out = append(out, '\n')
			} else if renvo_runtime_UnsafeByteAt(src, i) == 't' {
				out = append(out, '\t')
			} else if renvo_runtime_UnsafeByteAt(src, i) == 'r' {
				out = append(out, '\r')
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
		out = append(out, renvo_runtime_UnsafeByteAt(src, i))
		i++
	}
	return out
}

func renvoParseIntToken(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	start := int(tok.start)
	end := int(tok.end)
	base := 10
	if end-start > 2 && renvo_runtime_UnsafeByteAt(src, start) == '0' {
		prefix := renvo_runtime_UnsafeByteAt(src, start+1)
		if prefix == 'x' || prefix == 'X' {
			base = 16
			start += 2
		} else if prefix == 'b' || prefix == 'B' {
			base = 2
			start += 2
		}
	}
	if base == 10 && end-start > 1 && renvo_runtime_UnsafeByteAt(src, start) == '0' {
		base = 8
		start++
	}
	n := 0
	for i := start; i < end; i++ {
		d := 0
		if renvo_runtime_UnsafeByteAt(src, i)-'0' <= 9 {
			d = int(renvo_runtime_UnsafeByteAt(src, i) - '0')
		} else if renvo_runtime_UnsafeByteAt(src, i) >= 'a' && renvo_runtime_UnsafeByteAt(src, i) <= 'f' {
			d = int(renvo_runtime_UnsafeByteAt(src, i)-'a') + 10
		} else if renvo_runtime_UnsafeByteAt(src, i) >= 'A' && renvo_runtime_UnsafeByteAt(src, i) <= 'F' {
			d = int(renvo_runtime_UnsafeByteAt(src, i)-'A') + 10
		}
		n = n*base + d
	}
	return n
}

func renvoParseFloatTokenScaled(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	if tok.start+2 < tok.end && renvo_runtime_UnsafeByteAt(p.src, tok.start) == '0' && (renvo_runtime_UnsafeByteAt(p.src, tok.start+1) == 'x' || renvo_runtime_UnsafeByteAt(p.src, tok.start+1) == 'X') {
		return renvoParseHexFloatTokenScaled(p, tokIndex)
	}
	value := 0
	i := tok.start
	for i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) != '.' {
		if renvo_runtime_UnsafeByteAt(p.src, i)-'0' <= 9 {
			value = value*10 + int(renvo_runtime_UnsafeByteAt(p.src, i)-'0')
		}
		i++
	}
	value = value * 4
	if i < tok.end && renvo_runtime_UnsafeByteAt(p.src, i) == '.' {
		i++
		frac := 0
		scale := 1
		for i < tok.end {
			if renvo_runtime_UnsafeByteAt(p.src, i)-'0' <= 9 {
				frac = frac*10 + int(renvo_runtime_UnsafeByteAt(p.src, i)-'0')
				scale = scale * 10
			}
			i++
		}
		if scale > 1 {
			value += (frac * 4) / scale
		}
	}
	return value
}

func renvoParseHexFloatTokenScaled(p *renvoProgram, tokIndex int) int {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	src := p.src
	i := tok.start + 2
	mantissa := 0
	fracDigits := 0
	afterDot := false
	for i < tok.end {
		c := renvo_runtime_UnsafeByteAt(src, i)
		if c == '_' {
			i++
			continue
		}
		if c == '.' {
			afterDot = true
			i++
			continue
		}
		if c == 'p' || c == 'P' {
			break
		}
		digit := renvoHexFloatDigit(c)
		if digit >= 0 {
			mantissa = mantissa*16 + digit
			if afterDot {
				fracDigits++
			}
		}
		i++
	}
	exponent := 0
	sign := 1
	if i < tok.end && (renvo_runtime_UnsafeByteAt(src, i) == 'p' || renvo_runtime_UnsafeByteAt(src, i) == 'P') {
		i++
		if i < tok.end && renvo_runtime_UnsafeByteAt(src, i) == '-' {
			sign = -1
			i++
		} else if i < tok.end && renvo_runtime_UnsafeByteAt(src, i) == '+' {
			i++
		}
		for i < tok.end {
			c := renvo_runtime_UnsafeByteAt(src, i)
			if c-'0' <= 9 {
				exponent = exponent*10 + int(c-'0')
			}
			i++
		}
	}
	power := sign*exponent + 2 - fracDigits*4
	for power > 0 {
		mantissa = mantissa * 2
		power--
	}
	for power < 0 {
		mantissa = mantissa / 2
		power++
	}
	return mantissa
}

func renvoHexFloatDigit(c byte) int {
	if c-'0' <= 9 {
		return int(c - '0')
	}
	lower := c | 32
	if lower-'a' <= 5 {
		return int(lower-'a') + 10
	}
	return -1
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
	simple := "abfnrtv\\'\""
	values := "\a\b\f\n\r\t\v\\'\""
	for j := 0; j < len(simple); j++ {
		if renvo_runtime_UnsafeByteAt(src, i) == simple[j] {
			return int(values[j])
		}
	}
	digits := 0
	if renvo_runtime_UnsafeByteAt(src, i) == 'x' {
		digits = 2
	} else if renvo_runtime_UnsafeByteAt(src, i) == 'u' {
		digits = 4
	} else if renvo_runtime_UnsafeByteAt(src, i) == 'U' {
		digits = 8
	}
	if digits > 0 {
		return renvoParseEscapeDigits(src, i+1, digits, 16)
	}
	if renvo_runtime_UnsafeByteAt(src, i) >= '0' && renvo_runtime_UnsafeByteAt(src, i) <= '7' {
		return renvoParseEscapeDigits(src, i, 3, 8)
	}
	return int(renvo_runtime_UnsafeByteAt(src, i))
}

func renvoParseEscapeDigits(src []byte, start int, count int, base int) int {
	value := 0
	for i := 0; i < count; i++ {
		digit := renvoHexFloatDigit(renvo_runtime_UnsafeByteAt(src, start+i))
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
			value := s.constValue
			if renvoTokIsKind(g.prog, s.initStart, renvoTokNumber) && renvoResolveType(g.meta, s.typ).kind == renvoTypeFloat64 {
				value = value << 2
			}
			renvoSetConstResult(out, value, true)
			return
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		if !renvoParseExpressionOK(ep, g.prog, s.initStart, s.initEnd) {
			renvoSetConstResult(out, 0, false)
			return
		}
		rootIndex := len(ep.exprs) - 1
		oldIota := g.constEvalIota
		oldIotaValid := g.constEvalIotaValid
		g.constEvalIota = s.iotaValue
		g.constEvalIotaValid = 1
		result := renvoEvalConstExpr(g, ep, rootIndex)
		value := result.value
		ok := result.ok
		g.constEvalIota = oldIota
		g.constEvalIotaValid = oldIotaValid
		if ok {
			renvoSetConstResult(out, value, true)
			return
		}
		renvoSetConstResult(out, 0, false)
		return
	}
	renvoSetConstResult(out, 0, false)
}

func renvoConstResultOk(value int) renvoConstResult {
	var r renvoConstResult
	r.value = value
	r.ok = true
	return r
}

func renvoConvertConstInt(value int, kind int) int {
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

func renvoConvertConstScalar(value int, sourceKind int, destKind int) int {
	if destKind == renvoTypeFloat64 && sourceKind != renvoTypeFloat64 {
		return value << 2
	}
	if destKind != renvoTypeFloat64 && sourceKind == renvoTypeFloat64 {
		value = value / 4
	}
	return renvoConvertConstInt(value, destKind)
}

func renvoSetConstResult(result *renvoConstResult, value int, ok bool) {
	renvoNonNil(result)
	result.value = value
	result.ok = ok
}

func renvoEvalConstExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) renvoConstResult {
	renvoNonNil(g, ep)
	var result renvoConstResult
	renvoEvalConstExprInto(g, ep, idx, &result)
	return result
}

func renvoEvalConstExprInto(g *renvoLinearGen, ep *renvoExprParse, idx int, out *renvoConstResult) {
	renvoNonNil(g, ep, out)
	p := g.prog
	e := ep.exprs[idx]
	if e.kind == renvoExprInt {
		value := renvoParseIntToken(p, e.tok)
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprFloat {
		value := renvoParseFloatTokenScaled(p, e.tok)
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprChar {
		value := renvoParseCharToken(p, e.tok)
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprBool {
		value := renvoBoolTokenValue(p, e.tok)
		renvoSetConstResult(out, value, true)
		return
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if g.locals[localIndex].constValid != 0 {
				renvoSetConstResult(out, g.locals[localIndex].constValue, true)
				return
			}
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoEvalConstByNameInto(g, e.nameStart, e.nameEnd, out)
		return
	}
	if e.kind == renvoExprCall {
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
						result.value = renvoConvertConstScalar(result.value, source.kind, resolved.kind)
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
		if renvoTokCharIs(p, e.tok, '-') {
			renvoSetConstResult(out, -inner.value, true)
			return
		}
		if renvoTokCharIs(p, e.tok, '+') {
			renvoSetConstResult(out, inner.value, true)
			return
		}
		if renvoTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				renvoSetConstResult(out, 1, true)
				return
			}
			renvoSetConstResult(out, 0, true)
			return
		}
		renvoSetConstResult(out, 0, false)
		return
	}
	if e.kind == renvoExprBinary {
		opTok := e.tok
		rightIndex := e.right
		rightExpr := &ep.exprs[rightIndex]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		left := renvoEvalConstExpr(g, ep, e.left)
		if !left.ok {
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
			if right.value != 0 {
				renvoSetConstResult(out, 1, true)
				return
			}
			renvoSetConstResult(out, 0, true)
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
			if right.value != 0 {
				renvoSetConstResult(out, 1, true)
				return
			}
			renvoSetConstResult(out, 0, true)
			return
		}
		var right renvoConstResult
		if rightKind == renvoExprInt {
			value := renvoParseIntToken(p, rightTok)
			right = renvoConstResultOk(value)
		} else if rightKind == renvoExprChar {
			value := renvoParseCharToken(p, rightTok)
			right = renvoConstResultOk(value)
		} else if rightKind == renvoExprBool {
			value := renvoBoolTokenValue(p, rightTok)
			right = renvoConstResultOk(value)
		} else {
			right = renvoEvalConstExpr(g, ep, rightIndex)
		}
		if !right.ok {
			renvoSetConstResult(out, 0, false)
			return
		}
		renvoEvalConstBinaryInto(g, opTok, left.value, right.value, out)
		return
	}
	renvoSetConstResult(out, 0, false)
}

func renvoEvalConstBinaryInto(g *renvoLinearGen, tok int, left int, right int, out *renvoConstResult) {
	renvoNonNil(g, out)
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
			value = left / right
		} else if c == '%' {
			if right == 0 {
				renvoSetConstResult(out, 0, false)
				return
			}
			value = left % right
		} else if c == '&' {
			value = left & right
		} else if c == '|' {
			value = left | right
		} else if c == '^' {
			value = left ^ right
		} else if c == '<' {
			if left < right {
				value = 1
			} else {
				value = 0
			}
		} else if c == '>' {
			if left > right {
				value = 1
			} else {
				value = 0
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
			value = left << right
		} else if c0 == '>' && c1 == '>' {
			value = left >> right
		} else if c0 == '=' && c1 == '=' {
			if left == right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '!' && c1 == '=' {
			if left != right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '<' && c1 == '=' {
			if left <= right {
				value = 1
			} else {
				value = 0
			}
		} else if c0 == '>' && c1 == '=' {
			if left >= right {
				value = 1
			} else {
				value = 0
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
	e := ep.exprs[idx]
	if e.kind != renvoExprIdent {
		return false
	}
	return renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
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
	ep.args = make([]int, 0, capacity)
	ep.fields = make([]renvoCompositeField, 0, (capacity+1)/2)
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
	c := renvoTokSingleChar(ep.prog, ep.pos)
	if c == '+' || c == '-' || c == '!' || c == '&' || c == '*' {
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
		if renvoTokCharIs(ep.prog, ep.pos, '{') {
			base := &ep.exprs[left]
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
		if renvoTokCharIs(ep.prog, ep.pos, '(') {
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
		if renvoTokCharIs(ep.prog, ep.pos, '[') {
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
		if renvoTokCharIs(ep.prog, ep.pos, '.') && renvoTokIsKind(ep.prog, ep.pos+1, renvoTokIdent) {
			dotTok := ep.pos
			nameTok := renvoTokAt(ep.prog, ep.pos+1)
			ep.pos += 2
			left = renvoAddExpr(ep, renvoExprSelector, dotTok, left, 0, 0, 0, int(nameTok.start), int(nameTok.end))
			continue
		}
		if renvoTokCharIs(ep.prog, ep.pos, '.') && renvoTokCharIs(ep.prog, ep.pos+1, '(') {
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
		if paren == 0 && brack == 0 && brace == 0 && renvoTokCharIs(p, i, ':') {
			return i
		}
		if renvoTokCharIs(p, i, '(') {
			paren++
		} else if renvoTokCharIs(p, i, ')') {
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
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokStruct) || renvoTokIdentIs(ep.prog, ep.pos, "map") || renvoTokCharIs(ep.prog, ep.pos, '[') {
		startTok := ep.pos
		typeEnd := renvoPrimaryTypeEnd(ep.prog, startTok, ep.end)
		if typeEnd > startTok {
			ep.pos = typeEnd
			return renvoAddExpr(ep, renvoExprIdent, startTok, 0, 0, 0, 0, int(renvoTokStart(ep.prog, startTok)), int(renvoTokEnd(ep.prog, typeEnd-1)))
		}
	}
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokFunc) && renvoTokCharIs(ep.prog, ep.pos+1, '(') {
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
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokIdent) {
		tokStart := int(renvoTokStart(ep.prog, ep.pos))
		tokEnd := int(renvoTokEnd(ep.prog, ep.pos))
		ep.pos++
		if renvoBytesEqualText(ep.prog.src, tokStart, tokEnd, "true") {
			return renvoAddExpr(ep, renvoExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		if renvoBytesEqualText(ep.prog.src, tokStart, tokEnd, "false") {
			return renvoAddExpr(ep, renvoExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		return renvoAddExpr(ep, renvoExprIdent, ep.pos-1, 0, 0, 0, 0, tokStart, tokEnd)
	}
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokNumber) {
		ep.pos++
		return renvoAddExpr(ep, renvoExprInt, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokFloat) {
		ep.pos++
		ep.hasFloat = true
		return renvoAddExpr(ep, renvoExprFloat, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokString) {
		ep.pos++
		return renvoAddExpr(ep, renvoExprString, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if renvoTokIsKind(ep.prog, ep.pos, renvoTokChar) {
		ep.pos++
		return renvoAddExpr(ep, renvoExprChar, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if renvoTokCharIs(ep.prog, ep.pos, '(') {
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
	paramsClose := renvoFindMatchingExprClose(p, funcTok+2, end, '(', ')')
	if paramsClose <= funcTok+1 {
		return -1
	}
	if renvoTokCharIs(p, paramsClose+1, '{') {
		return paramsClose + 1
	}
	resultStart := paramsClose + 1
	bodyOpen := renvoFindStatementBodyOpen(p, resultStart, end)
	if bodyOpen <= resultStart {
		return -1
	}
	return bodyOpen
}

func renvoPrimaryTypeEnd(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	if start >= end {
		return start
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
		c := renvoTokSingleChar(p, i)
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
	base := bp.stmtCount * renvoStmtWordCount
	if bp.stmtCount < 0 || base < 0 || base+renvoStmtWordCount > len(renvoBodyStmtData) {
		bp.ok = false
		return
	}
	renvoBodyStmtData[base+0] = stmt.kind
	renvoBodyStmtData[base+1] = stmt.startTok
	renvoBodyStmtData[base+2] = stmt.endTok
	renvoBodyStmtData[base+3] = stmt.exprStart
	renvoBodyStmtData[base+4] = stmt.exprEnd
	renvoBodyStmtData[base+5] = stmt.bodyStart
	renvoBodyStmtData[base+6] = stmt.bodyEnd
	renvoBodyStmtData[base+7] = stmt.elseStart
	renvoBodyStmtData[base+8] = stmt.elseEnd
	renvoBodyStmtData[base+9] = stmt.nameStart
	renvoBodyStmtData[base+10] = stmt.nameEnd
	bp.stmtCount++
}

func renvoBodyStmtAt(bp *renvoBodyParse, index int) renvoStmt {
	renvoNonNil(bp)
	var stmt renvoStmt
	base := index * renvoStmtWordCount
	if index < 0 || base < 0 || base+renvoStmtWordCount > len(renvoBodyStmtData) {
		bp.ok = false
		return stmt
	}
	stmt.kind = renvoBodyStmtData[base+0]
	stmt.startTok = renvoBodyStmtData[base+1]
	stmt.endTok = renvoBodyStmtData[base+2]
	stmt.exprStart = renvoBodyStmtData[base+3]
	stmt.exprEnd = renvoBodyStmtData[base+4]
	stmt.bodyStart = renvoBodyStmtData[base+5]
	stmt.bodyEnd = renvoBodyStmtData[base+6]
	stmt.elseStart = renvoBodyStmtData[base+7]
	stmt.elseEnd = renvoBodyStmtData[base+8]
	stmt.nameStart = renvoBodyStmtData[base+9]
	stmt.nameEnd = renvoBodyStmtData[base+10]
	return stmt
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
	line := renvoTokLine(p, start)
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		c := renvoTokSingleChar(p, i)
		if i > start && paren == 0 && brack == 0 && brace == 0 {
			if renvoTokIsKind(p, i, renvoTokEOF) {
				return i
			}
			if c == ';' {
				return i
			}
			if renvoTokLine(p, i) != line {
				if c == '{' {
					return i
				}
				if renvoTokIsKind(p, i, renvoTokReturn) || renvoTokIsKind(p, i, renvoTokIf) || renvoTokIsKind(p, i, renvoTokFor) || renvoTokIsKind(p, i, renvoTokSwitch) || renvoTokIsKind(p, i, renvoTokCase) || renvoTokIsKind(p, i, renvoTokDefault) || renvoTokIsKind(p, i, renvoTokVar) || renvoTokIsKind(p, i, renvoTokConst) || renvoTokIsKind(p, i, renvoTokBreak) || renvoTokIsKind(p, i, renvoTokContinue) || renvoTokIsKind(p, i, renvoTokGoto) {
					return i
				}
				if renvoLineContinuesAfterPrevToken(p, i) {
					line = renvoTokLine(p, i)
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
		if i > start && renvoTokLine(p, i) != line && paren == 0 && brack == 0 && brace == 0 {
			if renvoLineContinuesAfterPrevToken(p, i) {
				line = renvoTokLine(p, i)
			} else {
				if closed {
					if c == '}' && renvoTokLine(p, i+1) == renvoTokLine(p, i) && (renvoTokCharIs(p, i+1, '(') || renvoTokCharIs(p, i+1, '.') || renvoTokCharIs(p, i+1, '[')) {
						line = renvoTokLine(p, i)
						i++
						continue
					}
					if c == '}' && renvoTokSingleChar(p, i+1) == '{' && renvoTokLine(p, i+1) == renvoTokLine(p, i) {
						line = renvoTokLine(p, i)
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
	if i <= 0 {
		return false
	}
	prev := i - 1
	tok := renvoTokAt(p, prev)
	tokStart := tok.start
	tokEnd := tok.end
	if tokEnd <= tokStart {
		return false
	}
	c := renvo_runtime_UnsafeByteAt(p.src, tokStart)
	if c == ',' {
		return true
	}
	if c == '*' || c == '&' {
		return true
	}
	if c == '+' {
		if tokEnd == tokStart+1 || renvo_runtime_UnsafeByteAt(p.src, tokStart+1) != '+' {
			return true
		}
	}
	return false
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
					if closeTok > i && closeTok < end && renvoTokCharIs(p, closeTok, '{') {
						i = closeTok
						continue
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
		c := renvoTokSingleChar(p, i)
		if c == '(' {
			paren++
		} else if c == ')' {
			paren--
		} else if c == '[' {
			brack++
		} else if c == ']' {
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
	p := pp
	m.prog = p
	typeCap := len(p.decls)*4 + 256
	fieldCap := len(p.decls)*2 + 256
	globalCap := len(p.decls) + 128
	paramCap := len(p.funcs)*3 + 256
	m.types = make([]renvoTypeInfo, 0, typeCap)
	m.fields = make([]renvoFieldInfo, 0, fieldCap)
	m.globals = make([]renvoSymbolInfo, 0, globalCap)
	m.params = make([]renvoSymbolInfo, 0, paramCap)
	m.funcs = make([]renvoFuncInfo, 0, len(p.funcs)+128)
	m.typeBuckets = make([]int, typeCap*2)
	m.closures = make([]renvoClosureInfo, 0, 16)
	m.captures = make([]renvoSymbolInfo, 0, 32)
	m.globalBuckets = make([]int, globalCap*2)
	for i := 0; i < len(m.globalBuckets); i++ {
		m.globalBuckets[i] = -1
	}
	m.funcBuckets = make([]int, (len(p.funcs)+128)*2)
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
	m.panicEnabled = p.toks.panicEnabled
	renvoFinalizeTypeLayouts(m)
	renvoBuildFuncLookup(m)
	renvoResolveGlobalCallTypes(m)
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
			renvoParseTopDeclEntry(m, p, kind, j, entryEnd)
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
	return (((size*33+int(renvo_runtime_UnsafeByteAt(src, start)))*33+int(renvo_runtime_UnsafeByteAt(src, end-1)))*33 + int(renvo_runtime_UnsafeByteAt(src, start+size/2))) & 2147483647
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
	m.globalNext[index] = m.globalBuckets[bucket]
	m.globalBuckets[bucket] = index
}

func renvoFindMetaGlobalIndex(m *renvoMeta, nameStart int, nameEnd int, kind int) int {
	renvoNonNil(m)
	hash := renvoHashRange(m.prog.src, nameStart, nameEnd)
	i := m.globalBuckets[hash%len(m.globalBuckets)]
	for i >= 0 {
		s := m.globals[i]
		if s.kind == kind && renvoBytesEqualRange(m.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = m.globalNext[i]
	}
	return -1
}

func renvoBuildFuncLookup(m *renvoMeta) {
	renvoNonNil(m)
	m.funcNext = make([]int, len(m.funcs))
	for i := 0; i < len(m.funcNext); i++ {
		m.funcNext[i] = -1
	}
	for i := 0; i < len(m.funcs); i++ {
		f := &m.funcs[i]
		hash := renvoHashRange(m.prog.src, f.nameStart, f.nameEnd)
		bucket := hash % len(m.funcBuckets)
		m.funcNext[i] = m.funcBuckets[bucket]
		m.funcBuckets[bucket] = i
	}
}

func renvoFindMetaFunction(m *renvoMeta, nameStart int, nameEnd int) int {
	renvoNonNil(m)
	hash := renvoHashRange(m.prog.src, nameStart, nameEnd)
	i := m.funcBuckets[hash%len(m.funcBuckets)]
	for i >= 0 {
		fn := &m.funcs[i]
		if fn.receiverType == 0 && renvoBytesEqualRange(m.prog.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = m.funcNext[i]
	}
	return -1
}

func renvoResolveGlobalCallTypes(m *renvoMeta) {
	renvoNonNil(m)
	for i := 0; i < len(m.globals); i++ {
		global := &m.globals[i]
		if global.typ != 0 || global.initStart >= global.initEnd {
			continue
		}
		ep := renvoNewExprParse()
		renvoNonNil(ep)
		rootIndex := renvoParseExpressionRoot(ep, m.prog, global.initStart, global.initEnd)
		if rootIndex < 0 {
			continue
		}
		root := &ep.exprs[rootIndex]
		if root.kind == renvoExprIdent {
			fnIndex := renvoFindMetaFunction(m, root.nameStart, root.nameEnd)
			if fnIndex >= 0 {
				global.typ = renvoFunctionTypeFromInfo(m, fnIndex)
			}
			continue
		}
		if root.kind != renvoExprCall || root.left < 0 || root.left >= len(ep.exprs) {
			continue
		}
		callee := &ep.exprs[root.left]
		if callee.kind != renvoExprIdent {
			continue
		}
		fnIndex := renvoFindMetaFunction(m, callee.nameStart, callee.nameEnd)
		if fnIndex >= 0 {
			global.typ = m.funcs[fnIndex].resultType
		}
	}
}

func renvoInitBuiltinTypes(m *renvoMeta) {
	renvoNonNil(m)
	renvoAddBuiltinType(m, renvoTypeInvalid, 0)
	renvoAddBuiltinType(m, renvoTypeInt, renvoNativeIntSize)
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
			constType := typ
			if constType == 0 {
				constType = renvoInferTopLiteralType(m, p, initStart, initEnd)
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
			constResult := renvoEvalMetaConstExpr(m, p, initStart, initEnd, iotaValue)
			if constResult.ok {
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
	e := ep.exprs[idx]
	if e.kind == renvoExprInt {
		renvoSetConstResult(out, renvoParseIntToken(p, e.tok), true)
		return
	}
	if e.kind == renvoExprFloat {
		renvoSetConstResult(out, renvoParseFloatTokenScaled(p, e.tok), true)
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
					result.value = renvoConvertConstInt(result.value, conversion.kind)
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
		if renvoTokCharIs(p, e.tok, '-') {
			renvoSetConstResult(out, -inner.value, true)
			return
		}
		if renvoTokCharIs(p, e.tok, '+') {
			renvoSetConstResult(out, inner.value, true)
			return
		}
		if renvoTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				renvoSetConstResult(out, 1, true)
				return
			}
			renvoSetConstResult(out, 0, true)
			return
		}
		renvoSetConstResult(out, 0, false)
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
		g.prog = p
		renvoEvalConstBinaryInto(&g, e.tok, left.value, right.value, out)
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
		if renvoTokCharIs(p, i, '(') {
			paren++
		} else if renvoTokCharIs(p, i, ')') {
			if paren > 0 {
				paren--
			}
		} else if renvoTokCharIs(p, i, '[') {
			brack++
		} else if renvoTokCharIs(p, i, ']') {
			if brack > 0 {
				brack--
			}
		} else if renvoTokCharIs(p, i, '{') {
			brace++
		} else if renvoTokCharIs(p, i, '}') {
			if brace > 0 {
				brace--
			}
		} else if paren == 0 && brack == 0 && brace == 0 && renvoTokCharIs(p, i, '=') {
			return i
		}
		i++
	}
	return start
}

func renvoParseTopDeclEntry(m *renvoMeta, p *renvoProgram, kind int, start int, end int) {
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
		typeResult := renvoParseType(m, p, typeStart, end)
		if typeResult.typ == 0 || typeResult.next > end {
			renvoMetaError(m)
			return
		}
		directNamedType := !isAlias && (renvoTokIsKind(p, typeStart, renvoTokStruct) || renvoTokCharIs(p, typeStart, '*') || renvoTokCharIs(p, typeStart, '['))
		if directNamedType && (m.types[typeResult.typ].kind == renvoTypeStruct || m.types[typeResult.typ].kind == renvoTypePointer || m.types[typeResult.typ].kind == renvoTypeSlice) {
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
	renvoMetaAppendGlobal(m, renvoSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
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
		renvoMetaAppendGlobal(m, renvoSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: renvoTokVar, typ: symType, initStart: initStart, initEnd: initEnd})
	}
}

func renvoInferTopLiteralType(m *renvoMeta, p *renvoProgram, start int, end int) int {
	renvoNonNil(m, p)
	if start+1 == end && renvoTokIsKind(p, start, renvoTokString) {
		return renvoTypeString
	}
	if start+1 == end && renvoTokIsKind(p, start, renvoTokFloat) {
		return renvoTypeFloat64
	}
	open := start
	depth := 0
	for open < end {
		if depth == 0 && renvoTokCharIs(p, open, '{') {
			typeResult := renvoParseType(m, p, start, open)
			if typeResult.typ != 0 {
				return typeResult.typ
			}
			return 0
		}
		if renvoTokCharIs(p, open, '(') || renvoTokCharIs(p, open, '[') {
			depth++
		} else if renvoTokCharIs(p, open, ')') || renvoTokCharIs(p, open, ']') {
			if depth > 0 {
				depth--
			}
		}
		open++
	}
	return 0
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
		unnamedReceiver := renvoParseType(m, p, fn.receiverStart, fn.receiverEnd)
		if unnamedReceiver.typ != 0 && unnamedReceiver.next == fn.receiverEnd {
			m.params = append(m.params, renvoSymbolInfo{typ: unnamedReceiver.typ})
			paramCount++
		} else {
			renvoParseParamList(m, p, fn.receiverStart, fn.receiverEnd, &paramCount)
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
	linkOK := 0
	linkDLLStart := 0
	linkDLLEnd := 0
	linkMethodStart := 0
	linkMethodEnd := 0
	if renvoHasLinkStaticDirectivePrefix(p, fn.nameStart) {
		linkStatic := renvoParseLinkStaticDirective(p, fn.nameStart)
		linkOK = linkStatic.ok
		linkDLLStart = linkStatic.dllStart
		linkDLLEnd = linkStatic.dllEnd
		linkMethodStart = linkStatic.methodStart
		linkMethodEnd = linkStatic.methodEnd
	}
	m.funcs = append(m.funcs, renvoFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, firstResult: firstResult, resultCount: resultCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd, linkStatic: linkOK, linkDLLStart: linkDLLStart, linkDLLEnd: linkDLLEnd, linkMethodStart: linkMethodStart, linkMethodEnd: linkMethodEnd})
	if receiverType != 0 && renvoResolveType(m, receiverType).kind != renvoTypePointer {
		renvoAddPointerType(m, receiverType, renvoPointerSpaceData)
	}
}

func renvoParseFuncLiterals(m *renvoMeta, p *renvoProgram) {
	renvoNonNil(m, p)
	tokenCount := renvoTokCount(p)
	for tok := 0; tok < tokenCount; tok++ {
		if int(p.toks.data[tok*renvoTokenStride])&255 != renvoTokFunc || !renvoTokCharIs(p, tok+1, '(') || renvoFuncTokenIsDeclarationOrSignature(p, tok) {
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

func renvoFuncTokenIsDeclarationOrSignature(p *renvoProgram, tok int) bool {
	renvoNonNil(p)
	for i := 0; i < len(p.funcs); i++ {
		if p.funcs[i].startTok == tok || tok > p.funcs[i].startTok && tok < p.funcs[i].bodyStart {
			return true
		}
	}
	return false
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

func renvoHasLinkStaticDirectivePrefix(p *renvoProgram, pos int) bool {
	renvoNonNil(p)
	src := p.src
	if pos < 0 || pos > len(src) {
		return false
	}
	lineStart := pos
	for lineStart > 0 {
		prev := lineStart - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c == '\n' {
			break
		}
		lineStart--
	}
	end := lineStart
	for end > 0 {
		prev := end - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		end--
	}
	if end <= 0 {
		return false
	}
	start := end
	for start > 0 {
		prev := start - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c == '\n' {
			break
		}
		start--
	}
	for start < end {
		c := renvo_runtime_UnsafeByteAt(src, start)
		if c != ' ' && c != '\t' {
			break
		}
		start++
	}
	prefix := "// renvo:linkstatic "
	return renvoBytesHasText(src, start, end, prefix)
}

func renvoParseLinkStaticDirective(p *renvoProgram, pos int) renvoLinkStaticDirective {
	renvoNonNil(p)
	var d renvoLinkStaticDirective
	src := p.src
	if pos < 0 || pos > len(src) {
		return d
	}
	lineStart := pos
	for lineStart > 0 {
		prev := lineStart - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c == '\n' {
			break
		}
		lineStart--
	}
	end := lineStart
	for end > 0 {
		prev := end - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		end--
	}
	if end <= 0 {
		return d
	}
	start := end
	for start > 0 {
		prev := start - 1
		c := renvo_runtime_UnsafeByteAt(src, prev)
		if c == '\n' {
			break
		}
		start--
	}
	for start < end && (renvo_runtime_UnsafeByteAt(src, start) == ' ' || renvo_runtime_UnsafeByteAt(src, start) == '\t') {
		start++
	}
	prefix := "// renvo:linkstatic "
	if !renvoBytesHasText(src, start, end, prefix) {
		return d
	}
	bodyStart := start + len(prefix)
	for bodyStart < end && (renvo_runtime_UnsafeByteAt(src, bodyStart) == ' ' || renvo_runtime_UnsafeByteAt(src, bodyStart) == '\t') {
		bodyStart++
	}
	comma := bodyStart
	for comma < end && renvo_runtime_UnsafeByteAt(src, comma) != ',' {
		comma++
	}
	if comma <= bodyStart || comma >= end {
		return d
	}
	dllEnd := comma
	for dllEnd > bodyStart && (renvo_runtime_UnsafeByteAt(src, dllEnd-1) == ' ' || renvo_runtime_UnsafeByteAt(src, dllEnd-1) == '\t') {
		dllEnd--
	}
	methodStart := comma + 1
	for methodStart < end && (renvo_runtime_UnsafeByteAt(src, methodStart) == ' ' || renvo_runtime_UnsafeByteAt(src, methodStart) == '\t') {
		methodStart++
	}
	methodEnd := end
	for methodEnd > methodStart && (renvo_runtime_UnsafeByteAt(src, methodEnd-1) == ' ' || renvo_runtime_UnsafeByteAt(src, methodEnd-1) == '\t') {
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

func renvoBytesHasText(src []byte, start int, end int, text string) bool {
	if start < 0 || end > len(src) || end-start < len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if renvo_runtime_UnsafeByteAt(src, start+i) != text[i] {
			return false
		}
	}
	return true
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
			renvoRebuildNamedTypeIndex(m)
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
	firstField := len(m.fields)
	offset := 0
	for i := 0; i < count; i++ {
		typ := m.params[first+i].typ
		offset = renvoAlignTo8(offset)
		m.fields = append(m.fields, renvoFieldInfo{typ: typ, offset: offset})
		offset += renvoTypeCopySize(m, typ)
	}
	return renvoAddType(m, renvoTypeStruct, 0, firstField, count, renvoAlignTo8(offset), 0, 0)
}

func renvoBuildTupleType(m *renvoMeta, types []int) int {
	renvoNonNil(m)
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
	renvoNonNil(m, p)
	var result renvoTypeResult
	renvoParseTypeInto(m, p, start, end, &result)
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

func renvoParseTypeInto(m *renvoMeta, p *renvoProgram, start int, end int, result *renvoTypeResult) {
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
		renvoSetTypeResult(result, renvoAddType(m, renvoTypeInterface, 0, start+2, closeTok, 2*renvoBackendValueSlotSize, 0, 0), closeTok+1)
		return
	}
	if renvoTokCharIs(p, start, '.') && renvoTokCharIs(p, start+1, '.') && renvoTokCharIs(p, start+2, '.') {
		elem := renvoParseType(m, p, start+3, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		typ := renvoAddSequenceType(m, renvoTypeSlice, elem.typ, 0, renvoBackendSliceValueSize)
		renvoSetTypeResult(result, typ, elem.next)
		return
	}
	if renvoTokCharIs(p, start, '*') {
		elem := renvoParseType(m, p, start+1, end)
		if elem.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		typ := renvoAddPointerType(m, elem.typ, renvoPointerSpaceData)
		renvoSetTypeResult(result, typ, elem.next)
		return
	}
	if renvoTokIdentIs(p, start, "map") && renvoTokCharIs(p, start+1, '[') {
		closeTok := renvoFindMatchingExprClose(p, start+2, end, '[', ']')
		if closeTok <= start+1 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		key := renvoParseType(m, p, start+2, closeTok)
		value := renvoParseType(m, p, closeTok+1, end)
		if key.typ == 0 || value.typ == 0 {
			renvoSetTypeResult(result, 0, start)
			return
		}
		renvoSetTypeResult(result, renvoAddType(m, renvoTypeMap, value.typ, key.typ, 0, renvoBackendValueSlotSize, 0, 0), value.next)
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
			length := renvoEvalMetaConstExpr(m, p, start+1, closeTok, 0)
			if !length.ok || length.value < 0 {
				renvoSetTypeResult(result, 0, start)
				return
			}
			count = length.value
		}
		elem := renvoParseType(m, p, closeTok+1, end)
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
		elem := renvoParseType(m, p, start+2, end)
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
		renvoTruncFields(&m.fields, firstField+count)
		fieldIndex := 0
		offset := 0
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
				fieldType := renvoParseType(m, p, typeStart, lineEnd)
				if fieldType.typ == 0 {
					renvoSetTypeResult(result, 0, start)
					return
				}
				var fieldInfo renvoFieldInfo
				fieldInfo.typ = fieldType.typ
				fieldInfo.embedded = embedded
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
		renvoSetTypeResult(result, typ, closeTok+1)
		return
	}
	if renvoTokIsKind(p, start, renvoTokIdent) {
		if renvoTokIdentIs(p, start, "any") || renvoTokIdentIs(p, start, "error") {
			renvoSetTypeResult(result, renvoAddType(m, renvoTypeInterface, 0, 0, 0, 2*renvoBackendValueSlotSize, 0, 0), start+1)
			return
		}
		builtin := renvoBuiltinTypeFromToken(p, start)
		if builtin != 0 {
			renvoSetTypeResult(result, builtin, start+1)
			return
		}
		renvoSetTypeResult(result, renvoNamedTypeFromToken(m, p, start), start+1)
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
	for i := 0; i < len(m.types); i++ {
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
	if renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "complex64") || renvoBytesEqualText(p.src, int(tok.start), int(tok.end), "complex128") {
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

func renvoNamedTypeFromToken(m *renvoMeta, p *renvoProgram, tokIndex int) int {
	renvoNonNil(m, p)
	tok := renvoTokAt(p, tokIndex)
	start := int(tok.start)
	end := int(tok.end)
	if typ := renvoFindNamedType(m, start, end); typ >= 0 {
		return typ
	}
	return renvoAddType(m, renvoTypeNamed, 0, 0, 0, renvoBackendValueSlotSize, start, end)
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
			buckets[bucket] = index + 1
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
		entry := buckets[bucket]
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
	for i := 1; i < len(m.types); i++ {
		typ := &m.types[i]
		if typ.kind == renvoTypePointer && typ.elem == elem && typ.first == addressSpace {
			return i
		}
	}
	return renvoAddType(m, renvoTypePointer, elem, addressSpace, 0, renvoBackendValueSlotSize, 0, 0)
}

func renvoAddSequenceType(m *renvoMeta, kind int, elem int, count int, size int) int {
	renvoNonNil(m)
	for i := 1; i < len(m.types); i++ {
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
	renvoNonNil(m)
	for i := 0; i < len(m.funcs); i++ {
		fn := &m.funcs[i]
		if fn.linkStatic == 0 {
			continue
		}
		for j := 0; j < fn.paramCount; j++ {
			renvoNativeTypeLayout(m, m.params[fn.firstParam+j].typ)
		}
		renvoNativeTypeLayout(m, fn.resultType)
	}
	for i := 0; i < len(m.types); i++ {
		t := m.types[i]
		if t.kind != renvoTypeStruct || t.nativeAlign > 0 {
			continue
		}
		offset := 0
		for j := 0; j < t.count; j++ {
			fieldIndex := t.first + j
			if fieldIndex < 0 || fieldIndex >= len(m.fields) {
				continue
			}
			offset = renvoAlignTo8(offset)
			m.fields[fieldIndex].offset = offset
			fieldSize := renvoTypeSize(m, m.fields[fieldIndex].typ)
			if fieldSize < 1 {
				fieldSize = renvoBackendValueSlotSize
			}
			offset += fieldSize
		}
		m.types[i].size = renvoAlignTo8(offset)
	}
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
	align := renvoNativeAlignment(size)
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
		renvoNativeTypeLayout(m, t.elem)
		size = renvoNativeIntSize
		align = renvoNativeAlignment(size)
	}
	if align < 1 {
		align = 1
	}
	t.size = size
	t.nativeAlign = align
	return size
}

func renvoNativeAlignment(size int) int {
	if size >= 8 && (renvoNativeIntSize == 8 || renvoTargetArch == renvoArchWasm32) {
		return 8
	}
	if size >= 4 {
		return 4
	}
	if size >= 2 {
		return 2
	}
	return 1
}

func renvoIntSliceContains(values []int, value int) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func renvoAppendIntCopy(values []int, value int) []int {
	out := make([]int, 0, len(values)+1)
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	out = append(out, value)
	return out
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
	size := renvoTypeSize(m, typ)
	if size < renvoBackendValueSlotSize {
		return renvoBackendValueSlotSize
	}
	return size
}

var renvoInvalidType renvoTypeInfo

func renvoResolveType(m *renvoMeta, typ int) *renvoTypeInfo {
	renvoNonNil(m)
	if typ >= 0 && typ < len(m.types) {
		t := &m.types[typ]
		if t.kind == renvoTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return renvoResolveType(m, t.elem)
		}
		if t.kind == renvoTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
			resolved := renvoFindResolvedNamedTypeIndex(m, typ)
			if resolved >= 0 {
				return renvoResolveType(m, resolved)
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
	return t.kind == renvoTypeInt
}

func renvoTypeKindIsScalarInt(kind int) bool {
	if kind > renvoTypeInvalid && kind < renvoTypeString {
		return true
	}
	return kind == renvoTypeInt8 || kind == renvoTypeInt16 || kind == renvoTypeInt32 || kind == renvoTypeUint16 || kind == renvoTypeUint32 || kind == renvoTypeUint64
}

func renvoTypeKindIsScalarValue(kind int) bool {
	return renvoTypeKindIsScalarInt(kind) || kind == renvoTypeFloat64
}

func renvoTypeKindUsesMemory(kind int) bool {
	return kind == renvoTypeString || (kind >= renvoTypeSlice && kind <= renvoTypeStruct) || (kind >= renvoTypeArray && kind <= renvoTypeMap)
}

func renvoScalarKindSize(kind int) int {
	if kind == renvoTypeByte || kind == renvoTypeBool || kind == renvoTypeInt8 {
		return 1
	}
	if kind == renvoTypeInt {
		return renvoNativeIntSize
	}
	if kind == renvoTypeInt16 || kind == renvoTypeUint16 {
		return 2
	}
	if kind == renvoTypeInt32 || kind == renvoTypeUint32 {
		return 4
	}
	return renvoBackendValueSlotSize
}

func renvoNativeScalarStorageSize(kind int) int {
	if kind == renvoTypePointer {
		return renvoNativeIntSize
	}
	return renvoScalarKindSize(kind)
}

func renvoAsmLoadPrimaryIndexTertiaryScalarOrPointer(a *renvoAsm, kind int) {
	renvoNonNil(a)
	elemSize := renvoScalarKindSize(kind)
	renvoAsmLoadPrimaryIndexTertiarySize(a, elemSize)
	renvoAsmNormalizePrimaryForKind(a, kind)
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
	return t.kind == renvoTypePointer && t.elem >= 0 && t.elem < len(m.types) && m.types[t.elem].nativeAlign > 0
}

func renvoTypeUsesHiddenResult(m *renvoMeta, typ int) bool {
	renvoNonNil(m)
	t := renvoResolveType(m, typ)
	renvoNonNil(t)
	return t.kind == renvoTypeStruct || t.kind == renvoTypeArray || t.kind == renvoTypeInterface
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
	for i := 0; i < aEnd-aStart; i++ {
		if renvo_runtime_UnsafeByteAt(src, aStart+i) != renvo_runtime_UnsafeByteAt(src, bStart+i) {
			return false
		}
	}
	return true
}

// Shared scalar code generation.
func renvoBindFunctionParams(g *renvoLinearGen, fnIndex int) {
	renvoNonNil(g)
	meta := g.meta
	fn := &meta.funcs[fnIndex]
	callWord := 0
	if renvoTypeUsesHiddenResult(meta, fn.resultType) {
		callWord = renvoBackendHiddenResultWordCount
	}
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := renvoAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if fn.literalTok > 0 && i == 0 {
			g.closureEnvOffset = offset
		}
		if renvoTypeIsSlice(meta, param.typ) {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			renvoStoreIncomingCallWord(g, callWord+2, offset-2*renvoBackendValueSlotSize)
			callWord += renvoBackendSliceWordCount
			continue
		}
		if renvoTypeIsString(meta, param.typ) {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			callWord += renvoBackendStringWordCount
			continue
		}
		paramType := renvoResolveType(meta, param.typ)
		renvoNonNil(paramType)
		if paramType.kind == renvoTypeInterface {
			renvoStoreIncomingCallWord(g, callWord, offset)
			renvoStoreIncomingCallWord(g, callWord+1, offset-renvoBackendValueSlotSize)
			callWord += 2
			continue
		}
		if paramType.kind == renvoTypeStruct || paramType.kind == renvoTypeArray {
			size := renvoTypeSize(meta, param.typ)
			wordSize := renvoCallWordSize(paramType.kind)
			for at := 0; at < size; at += wordSize {
				renvoStoreIncomingCallWord(g, callWord, offset-at)
				callWord++
			}
			continue
		}
		renvoStoreIncomingCallWord(g, callWord, offset)
		callWord++
	}
	renvoMoveCapturedLocals(g, true)
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
		g.bindingClosureCaptures = true
		renvoAddTypedLocal(g, capture.nameStart, capture.nameEnd, capture.typ)
		g.bindingClosureCaptures = false
		g.locals[g.localCount-1].captureOff = captureOff
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
	g.panicValueOff = g.asm.bssSize
	g.panicTypeOff = g.panicValueOff + renvoBackendValueSlotSize
	g.panicIDOff = g.panicTypeOff + renvoBackendValueSlotSize
	g.panicNextIDOff = g.panicIDOff + renvoBackendValueSlotSize
	g.panicPrevOff = g.panicNextIDOff + renvoBackendValueSlotSize
	g.panicDeferPendingOff = g.panicPrevOff + renvoBackendValueSlotSize
	g.panicRecoveredOff = g.panicDeferPendingOff + renvoBackendValueSlotSize
	g.asm.bssSize += 7 * renvoBackendValueSlotSize
}

func renvoEmitJumpIfBssEqualsStack(g *renvoLinearGen, bssOffset int, stackOffset int, label int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryBss(a, bssOffset)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryStack(a, stackOffset)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, label)
}

func renvoEmitStorePanicNodeField(g *renvoLinearGen, nodeOffset int, bssOffset int, displacement int) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryBss(&g.asm, bssOffset)
	renvoAsmLoadSecondaryStack(&g.asm, nodeOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, displacement)
}

func renvoEmitLoadPanicNodeField(g *renvoLinearGen, nodeOffset int, bssOffset int, displacement int) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryStackMemory(&g.asm, nodeOffset, displacement)
	renvoAsmStorePrimaryBss(&g.asm, bssOffset)
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
	g.panicEntryIDOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmCopyBssToStackSlot(&g.asm, g.panicIDOff, g.panicEntryIDOffset)
	g.panicRecoverAllowedOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmCopyBssToStackSlot(&g.asm, g.panicDeferPendingOff, g.panicRecoverAllowedOffset)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmStorePrimaryBss(&g.asm, g.panicDeferPendingOff)
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
	renvoAsmLoadPrimaryBss(&g.asm, g.panicIDOff)
	renvoAsmCmpPrimaryImm8(&g.asm, 0)
	noneLabel := renvoAsmNewLabel(&g.asm)
	renvoAsmJzLabel(&g.asm, noneLabel)
	samePanicLabel := renvoAsmNewLabel(&g.asm)
	renvoEmitJumpIfBssEqualsStack(g, g.panicIDOff, g.panicEntryIDOffset, samePanicLabel)
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
			if !ok || len(parts)/2 != fn.resultCount {
				return false
			}
			for i := 0; i < fn.resultCount; i++ {
				result := &g.meta.params[fn.firstResult+i]
				offset := renvoFindLocalOffset(g, result.nameStart, result.nameEnd)
				if offset < 0 {
					return false
				}
				ep := renvoNewExprParse()
				renvoNonNil(ep)
				root := renvoParseExpressionRoot(ep, g.prog, parts[i*2], parts[i*2+1])
				if root < 0 || !renvoEmitExprToLocal(g, ep, root, offset) {
					return false
				}
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
				} else if g.deferResultOffset <= 0 || !renvoEmitExprToLocal(g, ep, root, g.deferResultOffset) {
					return false
				}
			}
		}
	}
	renvoAsmJmpLabel(&g.asm, g.deferReturnLabel)
	return true
}

func renvoEmitFunctionControlEpilogue(g *renvoLinearGen) bool {
	renvoNonNil(g)
	if g.deferReturnLabel <= 0 {
		return false
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
		renvoAsmCopyBssToStackSlot(a, g.panicIDOff, savedPanicIDOffset)
		renvoAsmCopyBssToStackSlot(a, g.panicPrevOff, savedPanicPrevOffset)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmStorePrimaryBss(a, g.panicRecoveredOff)
		g.emittingDefers = true
		if !renvoEmitFunctionValueDispatch(g, site.funcType, handleOffset, argOffsets) {
			g.emittingDefers = false
			return false
		}
		g.emittingDefers = false
		panicStateReady := renvoAsmNewLabel(a)
		renvoAsmLoadPrimaryStack(a, savedPanicIDOffset)
		renvoAsmJzPrimary(a, panicStateReady)
		renvoEmitJumpIfBssEqualsStack(g, g.panicIDOff, savedPanicIDOffset, panicStateReady)
		renvoAsmLoadPrimaryBss(a, g.panicRecoveredOff)
		renvoAsmJnzPrimary(a, panicStateReady)
		renvoAsmLoadPrimaryBss(a, g.panicIDOff)
		renvoAsmJzPrimary(a, panicStateReady)
		renvoAsmLoadPrimaryStack(a, savedPanicPrevOffset)
		renvoAsmStorePrimaryBss(a, g.panicPrevOff)
		renvoAsmMarkLabel(a, panicStateReady)
		renvoAsmJmpMarkLabel(a, loopLabel, nextLabel)
	}
	renvoAsmJmpMarkLabel(a, loopLabel, doneDefers)
	renvoMoveCapturedLocals(g, true)
	panicReturn := renvoAsmNewLabel(a)
	normalReturn := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, g.panicIDOff)
	renvoAsmJzPrimary(a, normalReturn)
	renvoEmitJumpIfBssEqualsStack(g, g.panicIDOff, g.panicEntryIDOffset, normalReturn)
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
		if !renvoEmitCopySliceRegsToArena(g, fn.resultType) {
			return false
		}
	} else if renvoTypeIsString(g.meta, fn.resultType) {
		renvoAsmLoadPrimarySecondaryStack(a, g.deferResultOffset, g.deferResultOffset-renvoBackendValueSlotSize)
	} else {
		renvoAsmLoadPrimaryStack(a, g.deferResultOffset)
	}
	renvoAsmLeave(a)
	renvoAsmRet(a)
	return true
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
		if resolved.kind == renvoTypeStruct || resolved.kind == renvoTypeArray {
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
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, value)
		return
	}
	needsMovabs := renvoIntTokenNeedsMovabs(p, tokIndex)
	if needsMovabs {
		renvoAsmPrimaryImm64(a, value)
		return
	}
	renvoAsmPrimaryImm(a, value)
}
func renvoIntTokenNeedsMovabs(p *renvoProgram, tokIndex int) bool {
	renvoNonNil(p)
	tok := renvoTokAt(p, tokIndex)
	start := int(tok.start)
	end := int(tok.end)
	if end-start > 2 && renvo_runtime_UnsafeByteAt(p.src, start) == '0' {
		return false
	}
	digits := end - start
	if digits > 10 {
		return true
	}
	if digits < 10 {
		return false
	}
	limit := "2147483647"
	for i := 0; i < 10; i++ {
		c := renvo_runtime_UnsafeByteAt(p.src, start+i)
		if c > limit[i] {
			return true
		}
		if c < limit[i] {
			return false
		}
	}
	return false
}
func renvoAsmCopyPrimaryToSecondary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRdxRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRdxRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRdxRax(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 2, false) {
		return
	}
	renvoAsmEmit16(a, 0x5a50)
}
func renvoAsmCopyPrimaryToTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRcxRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRcxRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRcxRax(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 1, false) {
		return
	}
	renvoAsmEmit16(a, 0x5950)
}

func renvoAsmCopySecondaryToTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRcxRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRcxRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRcxRdx(a)
		return
	}
	renvoAsmEmit16(a, 0x5952)
}
func renvoAsmCopyTertiaryToPrimary(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmPushTertiary(a)
	renvoAsmPopPrimary(a)
}
func renvoAsmPushPrimary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPushRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPushRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPushRax(a)
		return
	}
	load := a.lastPrimaryLoad
	fold := renvoTargetArch == renvoArchAmd64 && load > 0 && load/8 == len(a.code)
	renvoAsmEmit8(a, 0x50)
	if fold {
		a.lastPrimaryLoad = -load
	}
}

func renvoAsmPushStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, offset)
	renvoAsmPushPrimary(a)
}

func renvoAsmPushStackWord(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchAmd64 || renvoTargetArch == renvoArch386 {
		renvoAsmEmit8(a, 0xff)
		if offset >= 0 && offset <= 128 {
			renvoAsmEmit8(a, 0x75)
			renvoAsmEmit8(a, -offset)
			return
		}
		renvoAsmEmit8(a, 0xb5)
		renvoAsmEmit32(a, -offset)
		return
	}
	renvoAsmPushStack(a, offset)
}
func renvoAsmPushTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPushRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPushRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPushRcx(a)
		return
	}
	renvoAsmEmit8(a, 0x51)
}
func renvoAsmPushSecondary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPushRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPushRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPushRdx(a)
		return
	}
	renvoAsmEmit8(a, 0x52)
}
func renvoAsmPushImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPushImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPushImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPushImm(a, imm)
		return
	}
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit2(a, 0x6a, imm)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		renvoAsmEmit8(a, 0x68)
		renvoAsmEmit32(a, imm)
		return
	}
	renvoAsmPrimaryImm(a, imm)
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
func renvoAsmPopPrimary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPopRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopRax(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 0, true) {
		return
	}
	renvoAsmEmit8(a, 0x58)
}
func renvoAsmPopTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPopRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 1, true) {
		return
	}
	renvoAsmEmit8(a, 0x59)
}
func renvoAsmPopSecondary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPopRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 2, true) {
		return
	}
	renvoAsmEmit8(a, 0x5a)
}
func renvoAsmPopCallWord1(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPopRsi(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopRsi(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopRsi(a)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && renvoAmd64RewritePrimaryLoad(a, 6, true) {
		return
	}
	renvoAsmEmit8(a, 0x5e)
}
func renvoAsmStorePrimaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8948, 0x45, 0x85)
	if renvoTargetArch == renvoArchAmd64 {
		a.lastPrimaryStoreEnd = len(a.code)
		a.lastPrimaryStoreOff = offset
	}
}
func renvoAsmStorePrimaryStackSize(a *renvoAsm, offset int, size int) {
	renvoNonNil(a)
	if size >= renvoNativeIntSize {
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
func renvoAsmStoreSecondaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8948, 0x55, 0x95)
}
func renvoAsmStorePrimarySecondaryStack(a *renvoAsm, primaryOffset int, secondaryOffset int) {
	renvoNonNil(a)
	renvoAsmStorePrimaryStack(a, primaryOffset)
	renvoAsmStoreSecondaryStack(a, secondaryOffset)
}
func renvoAsmLoadPrimaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchAmd64 {
		n := len(a.code)
		if a.lastPrimaryStoreEnd == n && a.lastPrimaryStoreOff == offset {
			return
		}
	}
	renvoAsmStackMem(a, offset, 0x8b48, 0x45, 0x85)
	if renvoTargetArch == renvoArchAmd64 {
		distance := 2
		if offset < 0 || offset > 128 {
			distance = 5
		}
		a.lastPrimaryLoad = len(a.code)*8 + distance
	}
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

func renvoAsmIncStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchAmd64 {
		// incq directly in the frame slot.  Loading the value into the primary
		// register and storing it back costs another seven to ten bytes at the
		// offsets used by larger compiler functions.
		renvoAsmStackMem(a, offset, 0xff48, 0x45, 0x85)
		return
	}
	renvoAsmLoadPrimaryStack(a, offset)
	renvoAsmIncPrimary(a)
	renvoAsmStorePrimaryStack(a, offset)
}

func renvoAsmJcmpStackStack(a *renvoAsm, left int, right int, label int, setcc int) {
	renvoNonNil(a)
	renvoAsmPushStack(a, left)
	renvoAsmLoadPrimaryStack(a, right)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, setcc)
	renvoAsmJnzPrimary(a, label)
}

func renvoAsmJcmpStackImm(a *renvoAsm, offset int, value int, label int, setcc int) {
	renvoNonNil(a)
	renvoAsmPushStack(a, offset)
	renvoAsmPrimaryImm(a, value)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, setcc)
	renvoAsmJnzPrimary(a, label)
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

func renvoAsmAddressPrimaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8d48, 0x45, 0x85)
}
func renvoAsmAddressCallWord0Stack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8d48, 0x7d, 0xbd)
}
func renvoAsmAddressCallWord1Stack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8d48, 0x75, 0xb5)
}
func renvoAsmLoadSecondaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8b48, 0x55, 0x95)
}
func renvoAsmLoadPrimarySecondaryStack(a *renvoAsm, primaryOffset int, secondaryOffset int) {
	renvoNonNil(a)
	renvoAsmLoadPrimaryStack(a, primaryOffset)
	renvoAsmLoadSecondaryStack(a, secondaryOffset)
}
func renvoAsmSecondaryDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm {
		return
	}
	if disp == 0 {
		renvoAsmEmit8(a, 0x02)
		return
	}
	if renvoAsmImmFits8Signed(disp) {
		renvoAsmEmit2(a, 0x42, disp)
		return
	}
	renvoAsmEmit8(a, 0x82)
	renvoAsmEmit32(a, disp)
}
func renvoAsmLoadTertiaryStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	renvoAsmStackMem(a, offset, 0x8b48, 0x4d, 0x8d)
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
func renvoAsmStoreSliceStack(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreSliceStack(a, offset)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreSliceStack(a, offset)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreSliceStack(a, offset)
		return
	}
	renvoAsmStorePrimarySecondaryStack(a, offset, offset-8)
	renvoAsmStackMem(a, offset-16, 0x8948, 0x4d, 0x8d)
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
func renvoAsmStoreByteMemSecondaryTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxMemRdxRcxSize(a, 1)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreAlMemRdxRcx1(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreAlMemRdxRcx1(a)
		return
	}
	renvoAsmEmit24(a, 0x0a0488)
}
func renvoAsmStorePrimaryMemSecondaryTertiarySize(a *renvoAsm, size int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if size == 1 {
		renvoAsmStoreByteMemSecondaryTertiary(a)
		return
	}
	if size == 2 {
		renvoAsmEmit32(a, 0x4a048966)
		return
	}
	if size == 4 {
		renvoAsmEmit24(a, 0x8a0489)
		return
	}
	renvoAsmStorePrimaryMemSecondaryTertiary8(a)
}
func renvoAsmIncTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmIncRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmIncRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmIncRcx(a)
		return
	}
	renvoAsmEmit16(a, 0xc1ff)
}
func renvoAsmIncPrimary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmIncRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmIncRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmIncRax(a)
		return
	}
	renvoAsmEmit16(a, 0xc0ff)
}
func renvoAsmMulTertiaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmImulRcxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmImulRcxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmImulRcxImm(a, imm)
		return
	}
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit3(a, 0x6b, 0xc9, imm)
		return
	}
	renvoAsmEmit16(a, 0xc969)
	renvoAsmEmit32(a, imm)
}
func renvoAsmRet(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmRet(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmRet(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmRet(a)
		return
	}
	renvoAsmEmit8(a, 0xc3)
}
func renvoAsmLeave(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLeave(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLeave(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLeave(a)
		return
	}
	renvoAsmEmit8(a, 0xc9)
}
func renvoAsmCallLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmCallLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmCallLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmCallLabel(a, label)
		return
	}
	renvoAsmEmit8(a, 0xe8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}
func renvoAsmJmpLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmJmpLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmJmpLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmJmpLabel(a, label)
		return
	}
	renvoAsmEmit8(a, 0xe9)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}
func renvoAsmJmpMarkLabel(a *renvoAsm, jumpLabel int, markLabel int) {
	renvoNonNil(a)
	renvoAsmJmpLabel(a, jumpLabel)
	renvoAsmMarkLabel(a, markLabel)
}
func renvoAsmJzLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmJzLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmJzLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmJzLabel(a, label)
		return
	}
	renvoAsmEmit16(a, 0x840f)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
}
func renvoAsmJnzLabel(a *renvoAsm, label int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmJnzLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmJnzLabel(a, label)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmJnzLabel(a, label)
		return
	}
	renvoAsmEmit16(a, 0x850f)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
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
	nameStart  int
	nameEnd    int
	nameHash   int
	offset     int
	captureOff int
	typ        int
	size       int
	constValue int
	constValid int
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
	prog                      *renvoProgram
	meta                      *renvoMeta
	asm                       renvoAsm
	funcLabels                []int
	funcReachable             []bool
	funcQueue                 []int
	currentFunc               int
	returnStruct              int
	closureEnvOffset          int
	bindingClosureCaptures    bool
	deferHeadOffset           int
	deferReturnLabel          int
	deferResultOffset         int
	panicEntryIDOffset        int
	panicRecoverAllowedOffset int
	deferSites                []renvoDeferSite
	emittingDefers            bool
	suppressPanicCheck        bool
	panicValueOff             int
	panicTypeOff              int
	panicIDOff                int
	panicNextIDOff            int
	panicPrevOff              int
	panicDeferPendingOff      int
	panicRecoveredOff         int
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
	locals                   []renvoLocalInfo
	localCount               int
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
	arenaAllocLabel          int
	arenaAllocEmitted        bool
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
}

func renvoAddStringData(g *renvoLinearGen, msg []byte) int {
	renvoNonNil(g)
	for off := 0; off+len(msg) < len(g.asm.data); off++ {
		match := g.asm.data[off+len(msg)] == 0
		for i := 0; match && i < len(msg); i++ {
			match = g.asm.data[off+i] == msg[i]
		}
		if match {
			return off
		}
	}
	msgOff := len(g.asm.data)
	for i := 0; i < len(msg); i++ {
		g.asm.data = append(g.asm.data, msg[i])
	}
	g.asm.data = append(g.asm.data, 0)
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
	renvoNonNil(g)
	var bp renvoBodyParse
	stmtData := make([]int, renvoStmtWordCount)
	renvoBodyStmtData = stmtData
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
		renvoBodyStmtData = stmtData
		bp.stmtCount = 0
		next := renvoParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || bp.stmtCount != 1 {
			return false
		}
		stmt := renvoBodyStmtAt(&bp, 0)
		if !bp.ok {
			return false
		}
		lastKind = stmt.kind
		i = next
		statementLocalBase := g.localCount
		statementStackBase := g.stackUsed
		if !renvoEmitLinearStmt(g, &stmt) {
			renvoPrintErr("renvo: failed to emit statement: ")
			write(2, prog.src[renvoTokStart(prog, stmt.startTok):renvoTokEnd(prog, stmt.endTok-1)], -1)
			renvoPrintErr("\n")
			return false
		}
		renvoReleaseStatementTemps(g, statementLocalBase, statementStackBase)
		if g.fixedPrunedReturns {
			g.fixedPrunedReturns = false
			g.lastRangeReturns = lastKind == renvoStmtReturn
			return true
		}
	}
	g.lastRangeReturns = lastKind == renvoStmtReturn
	if !bp.ok {
		return false
	}
	return true
}

func renvoReleaseStatementTemps(g *renvoLinearGen, localBase int, stackBase int) {
	renvoNonNil(g)
	for g.localCount > localBase {
		local := &g.locals[g.localCount-1]
		if local.nameStart != 0 || local.nameEnd != 0 {
			break
		}
		g.localCount--
	}
	if g.localCount > localBase {
		g.stackUsed = g.locals[g.localCount-1].offset
	} else {
		g.stackUsed = stackBase
	}
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
	typ := g.scopeValueType
	g.scopeValueType = 0
	if typ != 0 {
		offset := renvoAddTypedLocal(g, g.scopeValueNameStart, g.scopeValueNameEnd, typ)
		renvoCopyInterfaceValueToLocal(g, g.scopeValueOffset, typ, offset)
	}
	ok := renvoEmitLinearRange(g, start, end)
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	g.stackUsed = oldStackUsed
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
			if !renvoTokIsKind(g.prog, tok, renvoTokIdent) {
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
			if addressResult && localIndex < renvoNativeIntSize*8-1 && int(renvoTokStart(g.prog, tok)) == stmt.nameStart {
				g.checkedPointerLocals |= 1 << localIndex
			}
		}
	}
}

func renvoEmitLinearStmt(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	if stmt.kind == renvoStmtLabel || stmt.kind == renvoStmtGoto || stmt.kind == renvoStmtBreak || stmt.kind == renvoStmtContinue {
		g.checkedPointerLocals = 0
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
	if localIndex >= renvoNativeIntSize*8-1 {
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
			return renvoEmitBuiltinDelete(g, ep, rootIndex)
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
			if !renvoEmitScalarExprForKind(g, ep, rootIndex, resultResolved.kind) {
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
			renvoParseTopDeclGroup(g.meta, p, renvoTokType, start, stmt.endTok)
		} else {
			renvoParseTopDeclEntry(g.meta, p, renvoTokType, start, stmt.endTok)
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
		funcType = renvoEmitDeferredInterfaceMethodValue(g, ep, selector, call, payloadOffset)
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
	g.deferSites = append(g.deferSites, renvoDeferSite{funcType: funcType})
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

func renvoEmitDeferredInterfaceMethodValue(g *renvoLinearGen, ep *renvoExprParse, selector *renvoExpr, call *renvoExpr, offset int) int {
	renvoNonNil(g, ep, selector, call)
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
		if !renvoCallMatchesFuncType(renvoResolveType(g.meta, candidate), call) {
			continue
		}
		if funcType == 0 {
			funcType = candidate
		} else if !renvoTypesEquivalent(g.meta, funcType, candidate) {
			return 0
		}
		nextLabel := renvoAsmNewLabel(&g.asm)
		renvoEmitInterfaceReceiverMatch(g, receiverOffset, fn.receiverType, nextLabel)
		renvoEmitBoundMethodHandle(g, fnIndex, fn.receiverType, receiverOffset, renvoTypeSize(g.meta, fn.receiverType) > renvoBackendValueSlotSize, offset)
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
		g.fixedTargetState = 1
		if s.initStart >= s.initEnd {
			return
		}
		r := renvoEvalMetaConstExpr(g.meta, g.prog, s.initStart, s.initEnd, 0)
		if r.ok {
			g.fixedTargetValue = r.value
			return
		}
	}
}

const renvoFixedTargetUnknown = -2147483647

func renvoEvalFixedTargetInt(g *renvoLinearGen, ep *renvoExprParse, idx int, fixedTarget int, fixedTargetKnown bool) int {
	renvoNonNil(g, ep)
	p := g.prog
	renvoNonNil(p)
	if idx < 0 || idx >= len(ep.exprs) {
		return renvoFixedTargetUnknown
	}
	e := ep.exprs[idx]
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
		if fixedTarget >= renvoTargetLinuxAmd64 && fixedTarget <= renvoTargetWindowsArm64 {
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTargetArch") {
				return int(targetArchTable[fixedTarget])
			}
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTargetOS") {
				return int(targetOSTable[fixedTarget])
			}
			if renvoBytesEqualText(p.src, e.nameStart, e.nameEnd, "renvoTarget") {
				return fixedTarget
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
	e := ep.exprs[idx]
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
	if e.kind == renvoExprCall && e.argCount == 0 && fixedTarget >= renvoTargetLinuxAmd64 && fixedTarget <= renvoTargetWindowsArm64 {
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
func renvoEmitLinearElse(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.elseStart <= 0 {
		g.lastRangeReturns = false
		return true
	}
	if renvoTokIsKind(p, stmt.elseStart, renvoTokIf) && renvoTokIsKind(p, stmt.elseStart-1, renvoTokElse) {
		var nested renvoBodyParse
		oldStmtData := renvoBodyStmtData
		stmtData := make([]int, renvoStmtWordCount)
		renvoBodyStmtData = stmtData
		nested.prog = p
		nested.stmtCount = 0
		nested.ok = true
		next := renvoParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || nested.stmtCount != 1 {
			renvoBodyStmtData = oldStmtData
			return false
		}
		nestedStmt := renvoBodyStmtAt(&nested, 0)
		if !nested.ok {
			renvoBodyStmtData = oldStmtData
			return false
		}
		renvoBodyStmtData = oldStmtData
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
	if stmt.elseStart <= 0 {
		renvoAsmMarkLabel(a, endLabel)
		return true
	}
	if !thenReturns {
		renvoAsmJmpLabel(a, endLabel)
	}
	renvoAsmMarkLabel(a, elseLabel)
	if !renvoEmitLinearElse(g, stmt) {
		return false
	}
	renvoAsmMarkLabel(a, endLabel)
	return true
}
func renvoEmitLinearFor(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	a := &g.asm
	p := g.prog
	semi1 := renvoFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi1 >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, semi1)
	}
	rangeTok := renvoFindRangeToken(p, stmt.exprStart, stmt.exprEnd)
	if rangeTok >= stmt.exprStart {
		return renvoEmitLinearScopedControl(g, stmt, rangeTok)
	}
	endLabel := renvoNewControlLabel(g, 0)
	startLabel := renvoNewControlLabel(g, 1)
	g.pendingControl = 0
	renvoPushLoopLabels(g, endLabel, startLabel)
	renvoAsmMarkLabel(a, startLabel)
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

func renvoFindRangeToken(p *renvoProgram, start int, end int) int {
	renvoNonNil(p)
	for i := start; i < end; i++ {
		if renvoTokIdentIs(p, i, "range") {
			return i
		}
	}
	return start - 1
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
	} else if renvoTokCharIs(g.prog, split, ';') {
		ok = renvoEmitLinearClassicForScoped(g, stmt, split)
	} else {
		ok = renvoEmitLinearRangeForScoped(g, stmt, split)
	}
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	g.stackUsed = oldStackUsed
	return ok
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
	if resolved.kind != renvoTypeArray && resolved.kind != renvoTypeSlice && resolved.kind != renvoTypeMap && resolved.kind != renvoTypeString {
		return false
	}
	sourceOffset := renvoAddUnnamedLocal(g, sourceType)
	if !renvoEmitExprToLocal(g, source, sourceIndex, sourceOffset) {
		return false
	}
	sourceLenOffset := sourceOffset - 8
	if resolved.kind == renvoTypeMap {
		sourceLenOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmLoadPrimaryStack(a, sourceOffset)
		renvoEmitMapHeaderPtrLen(g)
		renvoAsmStorePrimaryStack(a, sourceOffset)
		renvoAsmCopyTertiaryToPrimary(a)
		renvoAsmStorePrimaryStack(a, sourceLenOffset)
	}
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
		if resolved.kind == renvoTypeMap {
			keyType = resolved.first
		} else if resolved.kind == renvoTypeString {
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
	if keyOffset > 0 && resolved.kind != renvoTypeMap {
		renvoAsmCopyStackSlot(a, indexOffset, keyOffset)
	}
	if resolved.kind == renvoTypeString {
		runeOffset := renvoAddUnnamedLocal(g, renvoTypeInt32)
		renvoEmitStringRangeDecode(g, sourceOffset, sourceLenOffset, indexOffset, runeOffset, widthOffset)
		if valueOffset > 0 {
			renvoAsmCopyStackSlot(a, runeOffset, valueOffset)
		}
	} else if resolved.kind == renvoTypeMap && (keyOffset > 0 || valueOffset > 0) {
		renvoAsmLoadTertiaryStack(a, indexOffset)
		renvoAsmMulTertiaryImm(a, renvoMapEntrySize)
		renvoAsmLoadSecondaryStack(a, sourceOffset)
		renvoAsmAddSecondaryTertiary(a)
		entryAddrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreSecondaryStack(a, entryAddrOffset)
		if keyOffset > 0 {
			renvoEmitCopyMemSecondaryToStack(g, keyOffset, renvoTypeSize(g.meta, resolved.first))
		}
		if valueOffset > 0 {
			renvoAsmLoadSecondaryStack(a, entryAddrOffset)
			renvoAsmAddSecondaryImm(a, 16)
			renvoEmitCopyMemSecondaryToStack(g, valueOffset, renvoTypeSize(g.meta, resolved.elem))
		}
	} else if valueOffset > 0 {
		renvoAsmLoadTertiaryStack(a, indexOffset)
		elemSize := renvoTypeSize(g.meta, resolved.elem)
		if elemSize != 1 {
			renvoAsmMulTertiaryImm(a, elemSize)
		}
		if resolved.kind == renvoTypeArray {
			renvoAsmAddressPrimaryStack(a, sourceOffset)
		} else {
			renvoAsmLoadPrimaryStack(a, sourceOffset)
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmAddSecondaryTertiary(a)
		renvoEmitCopyMemSecondaryToStack(g, valueOffset, elemSize)
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
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	rootIndex := -1
	typeSwitch, typeOperandEnd := renvoTypeSwitchOperandEnd(p, stmt.exprStart, stmt.exprEnd)
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
	valueOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	typeValueOffset := 0
	typeValueType := 0
	lenOffset := 0
	if stringSwitch {
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
		renvoAsmStorePrimaryStack(a, valueOffset)
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
			if !renvoEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, typeSwitch, clauseLabels[i]) {
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
func renvoTypeSwitchOperandEnd(p *renvoProgram, start int, end int) (bool, int) {
	renvoNonNil(p)
	if end-start < 5 || !renvoTokCharIs(p, end-4, '.') || !renvoTokCharIs(p, end-3, '(') || !renvoBytesEqualText(p.src, int(renvoTokStart(p, end-2)), int(renvoTokEnd(p, end-2)), "type") || !renvoTokCharIs(p, end-1, ')') {
		return false, end
	}
	return true, end - 4
}

func renvoCopyInterfaceValueToLocal(g *renvoLinearGen, sourceOffset int, typ int, valueOffset int) {
	renvoNonNil(g)
	if renvoResolveType(g.meta, typ).kind == renvoTypeInterface {
		renvoEmitCopyStackToStack(g, sourceOffset, valueOffset, 2*renvoBackendValueSlotSize)
		return
	}
	size := renvoTypeSize(g.meta, typ)
	if size <= renvoBackendValueSlotSize {
		renvoAsmCopyStackSlot(&g.asm, sourceOffset, valueOffset)
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
	if iface.first >= iface.count {
		renvoAsmLoadPrimaryStack(&g.asm, tagOffset)
		renvoAsmJnzPrimary(&g.asm, matchLabel)
		return
	}
	for candidate := 1; candidate < len(g.meta.types); candidate++ {
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

func renvoEmitSwitchCaseTests(g *renvoLinearGen, stmt *renvoStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, typeSwitch bool, matchLabel int) bool {
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
		} else if stringSwitch {
			if !renvoEmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, rootIndex, matchLabel) {
				return false
			}
		} else {
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
		if paren == 0 && brack == 0 && brace == 0 && renvoTokCharIs(p, i, ':') {
			return i
		}
		if renvoTokCharIs(p, i, '(') {
			paren++
		} else if renvoTokCharIs(p, i, ')') {
			paren--
		} else if renvoTokCharIs(p, i, '[') {
			brack++
		} else if renvoTokCharIs(p, i, ']') {
			brack--
		} else if renvoTokCharIs(p, i, '{') {
			brace++
		} else if renvoTokCharIs(p, i, '}') {
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
	return false
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
			if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm || renvoTargetArch == renvoArchWasm32 {
				renvoAsmLoadPrimaryStack(a, localOffset)
				renvoAsmPushImm(a, 1)
				renvoAsmPopTertiary(a)
				if inc {
					renvoAsmAddPrimaryTertiary(a)
				} else {
					renvoAsmSubPrimaryTertiary(a)
				}
				renvoAsmStorePrimaryStack(a, localOffset)
				return true
			}
			renvoAsmEmit16(a, 0xff48)
			if localOffset >= 0 && localOffset <= 128 {
				if inc {
					renvoAsmEmit8(a, 0x45)
				} else {
					renvoAsmEmit8(a, 0x4d)
				}
				renvoAsmEmit8(a, -localOffset)
			} else {
				if inc {
					renvoAsmEmit8(a, 0x85)
				} else {
					renvoAsmEmit8(a, 0x8d)
				}
				renvoAsmEmit32(a, -localOffset)
			}
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm || renvoTargetArch == renvoArchWasm32 {
			renvoAsmLoadPrimaryBss(a, globalOffset)
			renvoAsmPushImm(a, 1)
			renvoAsmPopTertiary(a)
			if inc {
				renvoAsmAddPrimaryTertiary(a)
			} else {
				renvoAsmSubPrimaryTertiary(a)
			}
			renvoAsmStorePrimaryBss(a, globalOffset)
			return true
		}
		if inc {
			renvoAsmEmit24(a, 0x05ff48)
		} else {
			renvoAsmEmit24(a, 0x0dff48)
		}
		at := len(a.code)
		renvoAsmEmit32(a, 0)
		renvoAsmAddAbsReloc(a, at, globalOffset, renvoAbsBssReloc)
		return true
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
func renvoEmitJumpIfTrue(g *renvoLinearGen, ep *renvoExprParse, idx int, trueLabel int) bool {
	renvoNonNil(g, ep)
	return renvoEmitJump(g, ep, idx, trueLabel, true)
}

func renvoEmitJump(g *renvoLinearGen, ep *renvoExprParse, idx int, label int, jumpIfTrue bool) bool {
	renvoNonNil(g, ep)
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
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
		if renvoEmitCompareJump(g, ep, e, label, jumpIfTrue) {
			return true
		}
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(p, e.tok, '!') {
		return renvoEmitJump(g, ep, e.left, label, !jumpIfTrue)
	}
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmCmpPrimaryImm8(a, 0)
	if jumpIfTrue {
		renvoAsmJnzLabel(a, label)
	} else {
		renvoAsmJzLabel(a, label)
	}
	return true
}
func renvoEmitCompareJumpOp(a *renvoAsm, c0 byte, c1 byte, label int, jumpIfTrue bool) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		cond := renvoWasm32CondEq
		if c0 == '=' {
			if jumpIfTrue {
				cond = renvoWasm32CondEq
			} else {
				cond = renvoWasm32CondNe
			}
		} else if c0 == '!' {
			if jumpIfTrue {
				cond = renvoWasm32CondNe
			} else {
				cond = renvoWasm32CondEq
			}
		} else if c0 == '<' {
			if c1 == '=' {
				if jumpIfTrue {
					cond = renvoWasm32CondLe
				} else {
					cond = renvoWasm32CondGt
				}
			} else {
				if jumpIfTrue {
					cond = renvoWasm32CondLt
				} else {
					cond = renvoWasm32CondGe
				}
			}
		} else if c1 == '=' {
			if jumpIfTrue {
				cond = renvoWasm32CondGe
			} else {
				cond = renvoWasm32CondLt
			}
		} else {
			if jumpIfTrue {
				cond = renvoWasm32CondGt
			} else {
				cond = renvoWasm32CondLe
			}
		}
		renvoWasm32EmitCondBranch(a, cond, label)
		return
	}
	if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchArm {
		cond := 0
		if c0 == '=' {
			if jumpIfTrue {
				cond = 0
			} else {
				cond = 1
			}
		} else if c0 == '!' {
			if jumpIfTrue {
				cond = 1
			} else {
				cond = 0
			}
		} else if c0 == '<' {
			if c1 == '=' {
				if jumpIfTrue {
					cond = 13
				} else {
					cond = 12
				}
			} else {
				if jumpIfTrue {
					cond = 11
				} else {
					cond = 10
				}
			}
		} else if c1 == '=' {
			if jumpIfTrue {
				cond = 10
			} else {
				cond = 11
			}
		} else {
			if jumpIfTrue {
				cond = 12
			} else {
				cond = 13
			}
		}
		if renvoTargetArch == renvoArchArm {
			renvoArmAsmBCondLabel(a, label, cond)
		} else {
			renvoAarch64AsmBCondLabel(a, label, cond)
		}
		return
	}
	op := 0
	if c0 == '=' {
		if jumpIfTrue {
			op = 33807
		} else {
			op = 34063
		}
	} else if c0 == '!' {
		if jumpIfTrue {
			op = 34063
		} else {
			op = 33807
		}
	} else if c0 == '<' {
		if c1 == '=' {
			if jumpIfTrue {
				op = 36367
			} else {
				op = 36623
			}
		} else {
			if jumpIfTrue {
				op = 35855
			} else {
				op = 36111
			}
		}
	} else if c1 == '=' {
		if jumpIfTrue {
			op = 36111
		} else {
			op = 35855
		}
	} else {
		if jumpIfTrue {
			op = 36623
		} else {
			op = 36367
		}
	}
	renvoAsmEmit16(a, op)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
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
	if renvoTypeIsInt(meta, s.typ) && renvoBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "renvoDefaultTarget") {
		renvoAsmPrimaryImm(&g.asm, renvoTarget)
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
	s.constValueOK = renvoInitDone
	return true
}

func renvoLinearInitGlobals(g *renvoLinearGen) bool {
	renvoNonNil(g)
	g.localCount = 0
	g.stackUsed = 0
	g.stackPeak = 0
	framePatch := renvoEmitGlobalInitFrameStart(g)
	meta := g.meta
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
	}
	for i := 0; i < len(meta.globals); i++ {
		if meta.globals[i].kind == renvoTokVar && !renvoLinearInitGlobal(g, i) {
			return false
		}
	}
	renvoEmitGlobalInitFrameEnd(g, framePatch)
	return true
}

func renvoEmitGlobalInitFrameStart(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchWasm32 {
		return -1
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmEmit(a, 0xa9bf7bfd)
		renvoAarch64AsmEmit(a, 0x910003fd)
		return renvoAarch64AsmFrameStart(a)
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmEmit(a, 0xe92d4800)
		renvoArmAsmMovRegReg(a, renvoArmRegFp, renvoArmRegSp)
		return renvoArmAsmFrameStart(a)
	}
	framePatch := len(a.code)
	renvoAsmEmit32(a, 0x000000c8)
	return framePatch
}

func renvoEmitGlobalInitFrameEnd(g *renvoLinearGen, framePatch int) {
	renvoNonNil(g)
	if renvoTargetArch != renvoArchWasm32 {
		renvoAsmLeave(&g.asm)
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPatchFrame(&g.asm, framePatch, g.stackPeak)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPatchFrame(&g.asm, framePatch, g.stackPeak)
		return
	}
	if framePatch < 0 {
		return
	}
	frame := renvoAlignValue(g.stackPeak, 16)
	if frame > 65520 {
		frame = 65520
	}
	g.asm.code[framePatch+1] = byte(frame & 255)
	g.asm.code[framePatch+2] = byte((frame >> 8) & 255)
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

func renvoEmitLinearAssign(g *renvoLinearGen, stmt *renvoStmt) bool {
	renvoNonNil(g, stmt)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	a := &g.asm
	tokenData := p.toks.data
	startBase := stmt.startTok * renvoTokenStride
	startKind := int(tokenData[startBase]) & 255
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	nextBase := startBase + renvoTokenStride
	if (stmt.kind == renvoStmtVar || startKind == renvoTokVar) && int(tokenData[nextBase])&255 == renvoTokIdent {
		nameStart = int(tokenData[nextBase+1])
		nameEnd = int(tokenData[nextBase+2])
	} else if startKind == renvoTokIdent {
		nameStart = int(tokenData[startBase+1])
		nameEnd = int(tokenData[startBase+2])
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
		if (renvoHasTopLevelComma(p, lhsStart, assignTok) || renvoHasTopLevelComma(p, assignTok+1, stmt.endTok)) && renvoEmitMultiAssign(g, stmt, assignTok) {
			return true
		}
	}
	if assignTok > stmt.startTok && compoundAssign {
		lhs := renvoNewExprParse()
		renvoNonNil(lhs)
		if renvoParseExpressionOK(lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == renvoExprIndex {
				container := renvoResolveType(meta, renvoInferParsedExprType(g, lhs, lhsRoot.left))
				renvoNonNil(container)
				elemType := renvoResolveType(meta, container.elem)
				renvoNonNil(elemType)
				if (container.kind != renvoTypeArray && container.kind != renvoTypeSlice) || !renvoTypeKindIsScalarInt(elemType.kind) {
					return false
				}
				elemSize := renvoScalarKindSize(elemType.kind)
				addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
				if !renvoEmitIndexAddressPrimary(g, lhs, lhsIndex) {
					return false
				}
				renvoAsmStorePrimaryStack(a, addrOffset)
				renvoAsmCopyPrimaryToSecondary(a)
				renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, elemSize)
				renvoAsmPushPrimary(a)
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if !renvoEmitIntExpr(g, rhs, rhsIndex) {
					return false
				}
				renvoAsmPopTertiary(a)
				if !renvoEmitPrimaryTertiaryOp(g, assignTok) {
					return false
				}
				renvoAsmNormalizePrimaryForKind(a, elemType.kind)
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
				return true
			}
			if lhsRoot.kind == renvoExprSelector {
				if !renvoEmitSelectorAddressSecondary(g, lhs, lhsIndex) {
					return false
				}
				lhsType := renvoInferParsedExprType(g, lhs, lhsIndex)
				lhsResolved := renvoResolveType(meta, lhsType)
				renvoNonNil(lhsResolved)
				addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
				renvoAsmStoreSecondaryStack(a, addrOffset)
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				if !renvoParseExpressionOK(rhs, p, assignTok+1, stmt.endTok) {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if lhsResolved.kind == renvoTypeString {
					if !renvoTok2Is(p, assignTok, '+', '=') || lhs.exprs[lhsRoot.left].kind != renvoExprIdent || !renvoEmitStringConcatPairValueRegs(g, lhs, lhsIndex, rhs, rhsIndex) {
						return false
					}
					renvoAsmPushStringRegs(a)
					renvoAsmLoadSecondaryStack(a, addrOffset)
					renvoAsmPopStoreStringMemSecondary(a, 0)
					return true
				}
				lhsSize := renvoScalarKindSize(lhsResolved.kind)
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, lhsSize)
				renvoAsmPushPrimary(a)
				if !renvoEmitIntExpr(g, rhs, rhsIndex) {
					return false
				}
				renvoAsmPopTertiary(a)
				if !renvoEmitPrimaryTertiaryOp(g, assignTok) {
					return false
				}
				renvoAsmNormalizePrimaryForKind(a, lhsResolved.kind)
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && renvoTokCharIs(p, assignTok, '=') && (startKind != renvoTokIdent || assignTok != stmt.startTok+1) {
		lhs := renvoNewExprParse()
		renvoNonNil(lhs)
		if renvoParseExpressionOK(lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			lhsType := renvoInferParsedExprType(g, lhs, lhsIndex)
			if lhsRoot.kind == renvoExprIndex {
				container := renvoResolveType(meta, renvoInferParsedExprType(g, lhs, lhsRoot.left))
				renvoNonNil(container)
				elemType := renvoResolveType(meta, container.elem)
				renvoNonNil(elemType)
				addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
				if container.kind == renvoTypeMap {
					if !renvoEmitMapEntryAddress(g, lhs, lhsRoot.left, lhsRoot.right, 1) {
						return false
					}
					renvoAsmCopyPrimaryToSecondary(a)
					renvoAsmAddSecondaryImm(a, 16)
					renvoAsmStoreSecondaryStack(a, addrOffset)
				} else if container.kind == renvoTypeArray || container.kind == renvoTypeSlice {
					if !renvoEmitIndexAddressPrimary(g, lhs, lhsIndex) {
						return false
					}
					renvoAsmStorePrimaryStack(a, addrOffset)
				} else {
					return false
				}
				rhs := renvoNewExprParse()
				renvoNonNil(rhs)
				rhsIndex := renvoParseExpressionRoot(rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if renvoTypeKindUsesMemory(elemType.kind) {
					return renvoEmitTypedExprToSavedMem(g, rhs, rhsIndex, container.elem, addrOffset)
				}
				if !renvoEmitScalarExprForKind(g, rhs, rhsIndex, elemType.kind) {
					return false
				}
				renvoAsmNormalizePrimaryForKind(a, elemType.kind)
				renvoAsmLoadSecondaryStack(a, addrOffset)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(elemType.kind))
				return true
			}
			lhsResolved := renvoResolveType(meta, lhsType)
			renvoNonNil(lhsResolved)
			if lhsRoot.kind == renvoExprSelector && renvoTypeKindUsesMemory(lhsResolved.kind) {
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
				lhsSize := renvoScalarKindSize(lhsResolved.kind)
				renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if renvoTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && compoundAssign {
			left := renvoNewExprParse()
			renvoNonNil(left)
			leftIndex := renvoParseExpressionRoot(left, p, stmt.startTok+1, assignTok)
			if leftIndex < 0 {
				return false
			}
			targetKind := renvoPointerTargetKind(g, left, leftIndex)
			targetSize := renvoScalarKindSize(targetKind)
			if !renvoEmitIntExpr(g, left, leftIndex) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmPushPrimary(a)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, targetSize)
			renvoAsmPushPrimary(a)
			right := renvoNewExprParse()
			renvoNonNil(right)
			rightIndex := renvoParseExpressionRoot(right, p, assignTok+1, stmt.endTok)
			if rightIndex < 0 {
				return false
			}
			if !renvoEmitScalarExprForKind(g, right, rightIndex, targetKind) {
				return false
			}
			renvoAsmPopTertiary(a)
			if !renvoEmitPrimaryTertiaryOp(g, assignTok) {
				return false
			}
			renvoAsmNormalizePrimaryForKind(a, targetKind)
			renvoAsmPopSecondary(a)
			renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, targetSize)
			return true
		}
		if renvoTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && renvoTokCharIs(p, assignTok, '=') {
			left := renvoNewExprParse()
			renvoNonNil(left)
			leftIndex := renvoParseExpressionRoot(left, p, stmt.startTok+1, assignTok)
			if leftIndex < 0 {
				return false
			}
			leftType := renvoInferParsedExprType(g, left, leftIndex)
			leftResolved := renvoResolveType(meta, leftType)
			renvoNonNil(leftResolved)
			if leftResolved.kind == renvoTypePointer {
				targetType := leftResolved.elem
				targetResolved := renvoResolveType(meta, targetType)
				renvoNonNil(targetResolved)
				if renvoTypeKindUsesMemory(targetResolved.kind) {
					if !renvoEmitIntExpr(g, left, leftIndex) {
						return false
					}
					renvoEmitRuntimeNonNilPrimary(g)
					addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
					renvoAsmStorePrimaryStack(a, addrOffset)
					right := renvoNewExprParse()
					renvoNonNil(right)
					rightIndex := renvoParseExpressionRoot(right, p, assignTok+1, stmt.endTok)
					if rightIndex < 0 {
						return false
					}
					return renvoEmitTypedExprToSavedMem(g, right, rightIndex, targetType, addrOffset)
				}
			}
			targetKind := renvoPointerTargetKind(g, left, leftIndex)
			targetSize := renvoScalarKindSize(targetKind)
			if !renvoEmitIntExpr(g, left, leftIndex) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmPushPrimary(a)
			right := renvoNewExprParse()
			renvoNonNil(right)
			rightIndex := renvoParseExpressionRoot(right, p, assignTok+1, stmt.endTok)
			if rightIndex < 0 {
				return false
			}
			if !renvoEmitScalarExprForKind(g, right, rightIndex, targetKind) {
				return false
			}
			renvoAsmPopSecondary(a)
			renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, targetSize)
			return true
		}
		return false
	}
	if nameEnd == nameStart+1 && renvo_runtime_UnsafeByteAt(p.src, nameStart) == '_' {
		if assignTok <= stmt.startTok || !renvoTokCharIs(p, assignTok, '=') {
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
		localIndex := renvoFindLocalIndex(g, int(tokenData[startBase+1]), int(tokenData[startBase+2]))
		if localIndex < 0 {
			return false
		}
		fieldBase := startBase + renvoTokenStride*2
		fieldNameStart := int(tokenData[fieldBase+1])
		fieldNameEnd := int(tokenData[fieldBase+2])
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
				if stmt.startTok+2 < typeEnd {
					typeResult := renvoParseType(meta, g.prog, stmt.startTok+2, typeEnd)
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
			offset = renvoAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			renvoAsmPrimaryImm(a, 0)
			renvoAsmStorePrimaryBss(a, globalOffset)
		} else {
			renvoZeroLocalAtOffset(g, offset)
			localType := renvoLocalTypeAtOffset(g, offset)
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
	trackLocalConst := globalOffset < 0 && fieldStackOffset < 0 && declaresLocal && renvoLocalConstTrackable(g, targetType, nameStart, nameEnd, stmt.endTok)
	localConst := renvoConstResult{}
	if trackLocalConst {
		localConst = renvoEvalConstExpr(g, ep, rootIndex)
	}
	if globalOffset < 0 && fieldStackOffset < 0 && !declaresLocal {
		renvoClearLocalConstAtOffset(g, offset)
	}
	if stmt.kind == renvoStmtShort {
		root := &ep.exprs[rootIndex]
		if root.kind == renvoExprCall && root.argCount >= 2 && renvoExprIdentCode(p, ep, root.left) == renvoIdentAppend {
			if !renvoEmitSliceValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, root.firstArg)) {
				return false
			}
			renvoAsmStoreSliceStack(a, offset)
		}
	}
	if renvoEmitAppendAssignGeneral(g, stmt, ep, assignTok) {
		if globalOffset < 0 && fieldStackOffset < 0 {
			renvoClearLocalConstAtOffset(g, offset)
		}
		return true
	}
	if compoundAssign {
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
		memoryOp := 0
		if renvoTok2Is(p, assignTok, '+', '=') {
			memoryOp = 0x0148
		} else if renvoTok2Is(p, assignTok, '-', '=') {
			memoryOp = 0x2948
		} else if renvoTok2Is(p, assignTok, '&', '=') {
			memoryOp = 0x2148
		} else if renvoTok2Is(p, assignTok, '|', '=') {
			memoryOp = 0x0948
		} else if renvoTok2Is(p, assignTok, '^', '=') {
			memoryOp = 0x3148
		}
		if renvoTargetArch == renvoArchAmd64 && memoryOp != 0 && globalOffset < 0 && fieldStackOffset < 0 && renvoTypeIsInt(meta, targetType) {
			if !renvoEmitScalarExprForKind(g, ep, rootIndex, targetResolved.kind) {
				return false
			}
			renvoAsmStackMem(a, offset, memoryOp, 0x45, 0x85)
			renvoClearLocalConstAtOffset(g, offset)
			return true
		}
		if globalOffset >= 0 {
			renvoAsmLoadPrimaryBss(a, globalOffset)
		} else {
			renvoAsmLoadPrimaryStack(a, offset)
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, rootIndex, targetResolved.kind) {
			return false
		}
		renvoAsmPopTertiary(a)
		if !renvoEmitPrimaryTertiaryOp(g, assignTok) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(a, targetResolved.kind)
		if globalOffset >= 0 {
			renvoAsmStorePrimaryBss(a, globalOffset)
		} else {
			renvoAsmStorePrimaryStack(a, offset)
			if fieldStackOffset < 0 {
				renvoClearLocalConstAtOffset(g, offset)
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
	if globalOffset >= 0 && (renvoTypeIsStruct(meta, targetType) || targetResolved.kind == renvoTypeInterface) {
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
			if trackLocalConst && localConst.ok {
				renvoSetLocalConstAtOffset(g, offset, localConst.value, targetResolved.kind)
			} else {
				renvoClearLocalConstAtOffset(g, offset)
			}
		}
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, rootIndex, targetResolved.kind) {
		return false
	}
	if globalOffset >= 0 {
		renvoAsmStorePrimaryBss(a, globalOffset)
	} else {
		renvoAsmStorePrimaryStack(a, offset)
		if fieldStackOffset < 0 {
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
	tempOffset := renvoAddUnnamedLocal(g, typ)
	if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
		return false
	}
	renvoAsmLoadSecondaryStack(&g.asm, addrOffset)
	renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, renvoTypeSize(g.meta, typ))
	return true
}

// renvoEmitGroupedTypedVarDecl handles VarSpecs whose identifier list shares one
// explicit type, such as "var first, second int". It returns zero when the
// statement is not such a declaration, one on success, and -1 on an emission
// failure.
func renvoEmitGroupedTypedVarDecl(g *renvoLinearGen, stmt *renvoStmt, assignTok int) int {
	renvoNonNil(g, stmt)
	p := g.prog
	if stmt.kind != renvoStmtVar || !renvoTokIsKind(p, stmt.startTok, renvoTokVar) {
		return 0
	}
	typeEnd := stmt.endTok
	if assignTok > stmt.startTok {
		typeEnd = assignTok
	}
	nameRanges := renvoFixedIntScratch(4)
	pos := stmt.startTok + 1
	for {
		if pos >= typeEnd || !renvoTokIsKind(p, pos, renvoTokIdent) {
			return 0
		}
		nameRanges = append(nameRanges, pos)
		nameRanges = append(nameRanges, pos+1)
		pos++
		if pos >= typeEnd || !renvoTokCharIs(p, pos, ',') {
			break
		}
		pos++
	}
	nameCount := len(nameRanges) / 2
	if nameCount < 2 || pos >= typeEnd {
		return 0
	}
	typeResult := renvoParseType(g.meta, p, pos, typeEnd)
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
	offsets := renvoFixedIntScratch(nameCount)
	for i := 0; i < nameCount; i++ {
		tok := nameRanges[i*2]
		nameStart := int(renvoTokStart(p, tok))
		nameEnd := int(renvoTokEnd(p, tok))
		offset := 0
		if nameEnd != nameStart+1 || renvo_runtime_UnsafeByteAt(p.src, nameStart) != '_' {
			offset = renvoAddTypedLocal(g, nameStart, nameEnd, typeResult.typ)
		}
		offsets = append(offsets, offset)
	}
	if assignTok <= stmt.startTok {
		for i := 0; i < nameCount; i++ {
			if offsets[i] != 0 {
				renvoZeroLocalAtOffset(g, offsets[i])
			}
		}
		return 1
	}
	size := renvoTypeCopySize(g.meta, typeResult.typ)
	for i := 0; i < nameCount; i++ {
		if offsets[i] != 0 {
			renvoEmitCopyStackToStack(g, temps[i], offsets[i], size)
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
	if lhsCount <= 1 && rhsCount <= 1 {
		return false
	}
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
	for i := 0; i < lhsCount; i++ {
		renvoClearLocalConstAssignTarget(g, stmt.kind, lhs[i*2], lhs[i*2+1])
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
		if !renvoEmitTempToTarget(g, stmt.kind, lhsStart, lhsEnd, tempOffsets[i], tempTypes[i]) {
			return false
		}
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
	typ := renvoInferParsedExprType(g, ep, root)
	if e.kind == renvoExprIndex {
		mapType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
		renvoNonNil(mapType)
		valueType := renvoResolveType(g.meta, mapType.elem)
		renvoNonNil(valueType)
		if mapType.kind != renvoTypeMap || (!renvoTypeKindIsScalarInt(valueType.kind) && valueType.kind != renvoTypePointer && valueType.kind != renvoTypeInterface) {
			return false
		}
		typ = mapType.elem
	} else if e.kind != renvoExprAssert {
		return false
	}
	value := renvoAddUnnamedLocal(g, typ)
	ok := renvoAddUnnamedLocal(g, renvoTypeBool)
	if e.kind == renvoExprAssert {
		if !renvoEmitTypeAssertionToLocal(g, ep, root, value, ok, false) {
			return false
		}
	} else {
		if !renvoEmitMapEntryAddress(g, ep, e.left, e.right, 0) {
			return false
		}
		a := &g.asm
		missing := renvoAsmNewLabel(a)
		done := renvoAsmNewLabel(a)
		renvoAsmJzPrimary(a, missing)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmAddSecondaryImm(a, 16)
		renvoEmitCopyMemSecondaryToStack(g, value, renvoTypeSize(g.meta, typ))
		renvoAsmStoreStackImm(a, ok, 1)
		renvoAsmJmpMarkLabel(a, done, missing)
		renvoZeroLocalAtOffset(g, value)
		renvoAsmStoreStackImm(a, ok, 0)
		renvoAsmMarkLabel(a, done)
	}
	if !renvoEmitTempToTarget(g, kind, lhs[0], lhs[1], value, typ) {
		return false
	}
	return renvoEmitTempToTarget(g, kind, lhs[2], lhs[3], ok, renvoTypeBool)
}

func renvoClearLocalConstAssignTarget(g *renvoLinearGen, kind int, targetStart int, targetEnd int) {
	renvoNonNil(g)
	p := g.prog
	ep := renvoNewExprParse()
	renvoNonNil(ep)
	if !renvoParseExpressionOK(ep, p, targetStart, targetEnd) {
		return
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != renvoExprIdent {
		return
	}
	if root.nameEnd == root.nameStart+1 && renvo_runtime_UnsafeByteAt(p.src, root.nameStart) == '_' {
		return
	}
	localIndex := renvoFindLocalIndex(g, root.nameStart, root.nameEnd)
	if kind == renvoStmtShort {
		localIndex = renvoFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
	}
	if localIndex >= 0 {
		renvoClearLocalConstAtOffset(g, g.locals[localIndex].offset)
	}
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
		baseType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, root.left))
		renvoNonNil(baseType)
		if baseType.kind == renvoTypeMap {
			if !renvoEmitMapEntryAddress(g, ep, root.left, root.right, 1) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			targetType := renvoResolveType(g.meta, baseType.elem)
			renvoNonNil(targetType)
			renvoAsmLoadPrimaryStack(&g.asm, tempOffset)
			renvoAsmStorePrimaryMemSecondaryDispSize(&g.asm, 16, renvoScalarKindSize(targetType.kind))
			return true
		}
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
func renvoEmitCopyStackToBss(g *renvoLinearGen, srcOffset int, bssOffset int, size int) {
	renvoNonNil(g)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	for at := 0; at < size; at += 8 {
		renvoAsmLoadPrimaryStack(&g.asm, srcOffset-at)
		renvoAsmStorePrimaryBss(&g.asm, bssOffset+at)
	}
}
func renvoEmitCopyBssToStack(g *renvoLinearGen, bssOffset int, destOffset int, size int) {
	renvoNonNil(g)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	for at := 0; at < size; at += 8 {
		renvoAsmCopyBssToStackSlot(&g.asm, bssOffset+at, destOffset-at)
	}
}
func renvoFindLocalIndexInCurrentScope(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	start := g.scopeBase
	if start < 0 {
		start = 0
	}
	index := renvoFindLocalIndex(g, nameStart, nameEnd)
	if index >= start {
		return index
	}
	return -1
}

func renvoSetLocalConstAtOffset(g *renvoLinearGen, offset int, value int, kind int) {
	renvoNonNil(g)
	value = renvoConvertConstInt(value, kind)
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValue = value
			g.locals[i].constValid = 1
			return
		}
	}
}

func renvoClearLocalConstAtOffset(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValid = 0
			return
		}
	}
}

func renvoLocalConstTrackable(g *renvoLinearGen, typ int, nameStart int, nameEnd int, afterTok int) bool {
	renvoNonNil(g)
	if renvoFixedTarget != 0 {
		return false
	}
	resolved := renvoResolveType(g.meta, typ)
	renvoNonNil(resolved)
	if !renvoTypeKindIsScalarInt(resolved.kind) {
		return false
	}
	return !renvoLocalNameWrittenAfter(g, nameStart, nameEnd, afterTok)
}

func renvoLocalNameWrittenAfter(g *renvoLinearGen, nameStart int, nameEnd int, afterTok int) bool {
	renvoNonNil(g)
	if nameEnd <= nameStart {
		return true
	}
	p := g.prog
	src := p.src
	nameSize := nameEnd - nameStart
	nameFirst := renvo_runtime_UnsafeByteAt(src, nameStart)
	tokenData := p.toks.data
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
		tokenStart := int(tokenData[base+1])
		tokenEnd := int(tokenData[base+2])
		if int(tokenData[base])&255 == renvoTokIdent && tokenEnd-tokenStart == nameSize && renvo_runtime_UnsafeByteAt(src, tokenStart) == nameFirst && renvoBytesEqualRange(src, tokenStart, tokenEnd, nameStart, nameEnd) {
			if renvoTokCharIs(p, i-1, '&') {
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
	data := p.toks.data
	for i < end {
		base := i * renvoTokenStride
		if base+2 < len(data) {
			c := byte(int(data[base]) >> 24)
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
	data := p.toks.data
	for i := start; i < end; i++ {
		base := i * renvoTokenStride
		if base >= len(data) {
			return false
		}
		c := byte(int(data[base]) >> 24)
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
		return renvoEmitStructReturnExpr(g, ep, rootIndex)
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
	e := ep.exprs[idx]
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
		callee := renvoExprIdentCode(p, ep, e.left)
		if callee == renvoIdentRecover && e.argCount == 0 {
			return renvoBuiltinTypeInterface
		}
		if callee == renvoIdentAppend && e.argCount >= 2 {
			return renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		if callee == renvoIdentByteSlice && e.argCount == 1 {
			return renvoAddType(meta, renvoTypeSlice, renvoTypeByte, 0, 0, renvoBackendSliceValueSize, 0, 0)
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
			return renvoTypeFloat64
		}
		if callee == renvoIdentComplex {
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
		if t.kind == renvoTypeSlice || t.kind == renvoTypeArray || t.kind == renvoTypeMap {
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
			return renvoAddType(meta, renvoTypeSlice, base.elem, 0, 0, renvoBackendSliceValueSize, 0, 0)
		}
		return baseType
	}
	if e.kind == renvoExprSelector {
		baseType := renvoInferParsedExprType(g, ep, e.left)
		fieldType := renvoStructFieldType(g, baseType, e.nameStart, e.nameEnd)
		if fieldType != 0 {
			return fieldType
		}
		fnIndex, expression := renvoMethodSelectorInfo(g, ep, idx)
		if fnIndex >= 0 {
			first := 1
			if expression {
				first = 0
			}
			return renvoFunctionTypeFromInfoStart(meta, fnIndex, first)
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
		if renvoTokCharIs(p, e.tok, '+') || renvoTokCharIs(p, e.tok, '-') {
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
		if renvoTok2Is(p, e.tok, '=', '=') || renvoTok2Is(p, e.tok, '!', '=') || renvoTokCharIs(p, e.tok, '<') || renvoTokCharIs(p, e.tok, '>') || renvoTok2Is(p, e.tok, '&', '&') || renvoTok2Is(p, e.tok, '|', '|') {
			return renvoTypeInt
		}
		leftTypeIndex := renvoInferParsedExprType(g, ep, e.left)
		rightTypeIndex := renvoInferParsedExprType(g, ep, e.right)
		leftType := renvoResolveType(meta, leftTypeIndex)
		renvoNonNil(leftType)
		rightType := renvoResolveType(meta, rightTypeIndex)
		renvoNonNil(rightType)
		if renvoTokCharIs(p, e.tok, '+') && (leftType.kind == renvoTypeString || rightType.kind == renvoTypeString) {
			return renvoTypeString
		}
		if leftType.kind == renvoTypeFloat64 || rightType.kind == renvoTypeFloat64 {
			return renvoTypeFloat64
		}
		if leftType.kind == renvoTypeComplex || rightType.kind == renvoTypeComplex {
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
	typeResult := renvoParseType(meta, p, e.tok, endTok)
	if typeResult.typ > 0 && typeResult.typ < len(meta.types) && meta.types[typeResult.typ].kind == renvoTypeArray && meta.types[typeResult.typ].count < 0 && e.kind == renvoExprComposite {
		count := 0
		next := 0
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			at := next
			if field.key >= 0 {
				key := renvoEvalConstExpr(g, ep, field.key)
				if !key.ok || key.value < 0 {
					return 0
				}
				at = key.value
			}
			next = at + 1
			if next > count {
				count = next
			}
		}
		g.meta.types[typeResult.typ].count = count
		g.meta.types[typeResult.typ].size = count * renvoTypeSize(g.meta, g.meta.types[typeResult.typ].elem)
	}
	return typeResult.typ
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
	if callee.kind != renvoExprIdent {
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
	for i := 0; i < g.localCount; i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < g.localCount; i++ {
		t := renvoResolveType(g.meta, g.locals[i].typ)
		renvoNonNil(t)
		if t.kind == renvoTypeStruct {
			for j := 0; j < t.count; j++ {
				field := g.meta.fields[t.first+j]
				if g.locals[i].offset-field.offset == offset {
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
	e := &ep.exprs[idx]
	if e.kind == renvoExprAssert {
		return renvoEmitTypeAssertionToLocal(g, ep, idx, offset, 0, true)
	}
	destResolved := renvoResolveType(meta, destType)
	renvoNonNil(destResolved)
	if destResolved.kind == renvoTypeComplex {
		if !renvoEmitComplexValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
		return true
	}
	if destResolved.kind == renvoTypeInterface {
		return renvoEmitInterfaceAssignToLocal(g, ep, idx, offset)
	}
	if destResolved.kind == renvoTypeMap {
		return renvoEmitMapAssignToLocal(g, ep, idx, destType, offset)
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
		renvoEmitCopyBssToStack(g, globalOffset, offset, size)
		return true
	}
	if destResolved.kind == renvoTypeArray {
		if e.kind == renvoExprCall {
			return renvoEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == renvoExprIndex {
			valueType := renvoInferParsedExprType(g, ep, idx)
			if renvoTypeSize(meta, valueType) != renvoTypeSize(meta, destType) || !renvoEmitIndexAddressPrimary(g, ep, idx) {
				return false
			}
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(meta, destType))
			return true
		}
		if e.kind == renvoExprSelector {
			fieldType := renvoInferParsedExprType(g, ep, idx)
			if renvoResolveType(meta, fieldType).kind != renvoTypeArray || renvoTypeSize(meta, fieldType) != renvoTypeSize(meta, destType) {
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
			if renvoResolveType(meta, valueType).kind != renvoTypeArray || renvoTypeSize(meta, valueType) != renvoTypeSize(meta, destType) {
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
		if e.kind != renvoExprComposite {
			return false
		}
		renvoZeroLocalAtOffset(g, offset)
		return renvoEmitCompositeFieldToStack(g, ep, idx, destType, offset)
	}
	if destResolved.kind == renvoTypeStruct {
		if e.kind == renvoExprCall {
			return renvoEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == renvoExprIndex {
			return renvoEmitIndexedStructToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == renvoExprSelector {
			fieldType := renvoInferParsedExprType(g, ep, idx)
			if !renvoTypeIsStruct(meta, fieldType) || renvoTypeSize(meta, fieldType) != renvoTypeSize(meta, destType) {
				return false
			}
			if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
				return false
			}
			size := renvoTypeSize(meta, destType)
			renvoEmitCopyMemSecondaryToStack(g, offset, size)
			return true
		}
		if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
			valueType := renvoInferParsedExprType(g, ep, idx)
			if !renvoTypeIsStruct(meta, valueType) || renvoTypeSize(meta, valueType) != renvoTypeSize(meta, destType) {
				return false
			}
			if !renvoEmitIntExpr(g, ep, e.left) {
				return false
			}
			renvoEmitRuntimeNonNilPrimary(g)
			renvoAsmCopyPrimaryToSecondary(&g.asm)
			size := renvoTypeSize(meta, destType)
			renvoEmitCopyMemSecondaryToStack(g, offset, size)
			return true
		}
		if e.kind == renvoExprComposite {
			renvoZeroLocalAtOffset(g, offset)
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
				if !renvoEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
					return false
				}
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
	if renvoTypeKindIsScalarValue(destResolved.kind) || destResolved.kind == renvoTypePointer || destResolved.kind == renvoTypeFunc {
		if !renvoEmitScalarExprForKind(g, ep, idx, destResolved.kind) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, offset)
		return true
	}
	if !renvoTypeIsSlice(meta, destType) {
		return false
	}
	if !renvoEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	renvoAsmStoreSliceStack(&g.asm, offset)
	return true
}

func renvoEmitInterfaceAssignToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, offset int) bool {
	renvoNonNil(g, ep)
	if idx >= 0 && idx < len(ep.exprs) {
		e := &ep.exprs[idx]
		if e.kind == renvoExprIdent && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
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
	}
	sourceType := renvoInferParsedExprType(g, ep, idx)
	source := renvoResolveType(g.meta, sourceType)
	renvoNonNil(source)
	if source.kind == renvoTypeInterface {
		e := &ep.exprs[idx]
		if e.kind == renvoExprCall {
			return renvoEmitStructCallToLocal(g, ep, idx, sourceType, offset)
		}
		if e.kind == renvoExprIndex {
			baseType := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
			renvoNonNil(baseType)
			if baseType.kind == renvoTypeMap {
				if !renvoEmitMapEntryAddress(g, ep, e.left, e.right, 0) {
					return false
				}
				missing := renvoAsmNewLabel(&g.asm)
				done := renvoAsmNewLabel(&g.asm)
				renvoAsmJzPrimary(&g.asm, missing)
				renvoAsmCopyPrimaryToSecondary(&g.asm)
				renvoAsmAddSecondaryImm(&g.asm, 16)
				renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, sourceType))
				renvoAsmJmpLabel(&g.asm, done)
				renvoAsmMarkLabel(&g.asm, missing)
				renvoZeroLocalAtOffset(g, offset)
				renvoAsmMarkLabel(&g.asm, done)
				return true
			}
		}
		if !renvoEmitAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(&g.asm)
		renvoEmitCopyMemSecondaryToStack(g, offset, renvoTypeSize(g.meta, sourceType))
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
	if size <= renvoBackendValueSlotSize {
		renvoAsmCopyStackSlot(&g.asm, valueOffset, offset)
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
	left := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.left))
	renvoNonNil(left)
	if left.kind == renvoTypeInterface {
		return true
	}
	right := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right))
	renvoNonNil(right)
	return right.kind == renvoTypeInterface
}

func renvoEmitInterfaceCompare(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	left := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	right := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitInterfaceAssignToLocal(g, ep, e.left, left) || !renvoEmitInterfaceAssignToLocal(g, ep, e.right, right) {
		return false
	}
	a := &g.asm
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
	renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, renvoInterfaceIndirectTypeBase, indirect, 0x9d)
	renvoAsmLoadPrimaryTertiaryStack(a, left, right)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJmpLabel(a, done)
	renvoAsmMarkLabel(a, indirect)
	for typ := 1; typ < len(g.meta.types); typ++ {
		tag := renvoRuntimeTypeTag(g.meta, typ)
		if tag != renvoInterfaceIndirectTypeBase+typ || renvoResolveType(g.meta, typ).kind == renvoTypeInterface {
			continue
		}
		next := renvoAsmNewLabel(a)
		renvoAsmJcmpStackImm(a, left-renvoBackendValueSlotSize, tag, next, 0x95)
		leftValue := renvoAddUnnamedLocal(g, typ)
		rightValue := renvoAddUnnamedLocal(g, typ)
		size := renvoTypeSize(g.meta, typ)
		renvoAsmLoadSecondaryStack(a, left)
		renvoEmitCopyMemSecondaryToStack(g, leftValue, size)
		renvoAsmLoadSecondaryStack(a, right)
		renvoEmitCopyMemSecondaryToStack(g, rightValue, size)
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
	if renvoTok2Is(g.prog, e.tok, '!', '=') {
		renvoAsmBoolNotPrimary(a)
	}
	return true
}

func renvoTypeComparable(meta *renvoMeta, typ int) bool {
	renvoNonNil(meta)
	t := renvoResolveType(meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypeSlice || t.kind == renvoTypeMap || t.kind == renvoTypeFunc {
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

func renvoRuntimeTypeTag(meta *renvoMeta, typ int) int {
	renvoNonNil(meta)
	if typ <= 0 || typ >= len(meta.types) {
		return 0
	}
	t := meta.types[typ]
	if t.kind == renvoTypeNamed && t.first == renvoNamedTypeAlias && t.elem > 0 {
		return renvoRuntimeTypeTag(meta, t.elem)
	}
	if !renvoTypeComparable(meta, typ) {
		return -typ
	}
	if renvoTypeSize(meta, typ) > renvoBackendValueSlotSize {
		return renvoInterfaceIndirectTypeBase + typ
	}
	return typ
}

const renvoMapEntrySize = 32

func renvoEmitMapValuePrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent && renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	mapType := renvoInferParsedExprType(g, ep, idx)
	if renvoResolveType(g.meta, mapType).kind != renvoTypeMap {
		return false
	}
	if e.kind == renvoExprComposite || (e.kind == renvoExprCall && renvoExprIdentCode(g.prog, ep, e.left) == renvoIdentMake) {
		offset := renvoAddUnnamedLocal(g, mapType)
		if !renvoEmitMapAssignToLocal(g, ep, idx, mapType, offset) {
			return false
		}
		renvoAsmLoadPrimaryStack(&g.asm, offset)
		return true
	}
	return renvoEmitMachineIntExpr(g, ep, idx)
}

func renvoEmitNewMapHeaderPrimary(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(a, sizeOffset, renvoBackendSliceValueSize)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(a, addrOffset)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 8)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 16)
	renvoAsmLoadPrimaryStack(a, addrOffset)
}

func renvoEmitMapAssignToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, mapType int, offset int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	resolved := renvoResolveType(g.meta, mapType)
	renvoNonNil(resolved)
	if resolved.kind != renvoTypeMap || !renvoTypeIsString(g.meta, resolved.first) {
		return false
	}
	if e.kind != renvoExprComposite && (e.kind != renvoExprCall || renvoExprIdentCode(g.prog, ep, e.left) != renvoIdentMake) {
		if !renvoEmitMapValuePrimary(g, ep, idx) {
			return false
		}
		renvoAsmStorePrimaryStack(a, offset)
		return true
	}
	entrySize := renvoMapEntrySize
	if e.kind == renvoExprComposite {
	} else if e.kind == renvoExprCall {
		if renvoExprIdentCode(g.prog, ep, e.left) != renvoIdentMake || e.argCount < 1 || e.argCount > 2 {
			return false
		}
		madeType := renvoTypeFromExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		if renvoResolveType(g.meta, madeType).kind != renvoTypeMap {
			return false
		}
		// A map capacity is a performance hint. Evaluate it for its side
		// effects, but let the shared growable descriptor allocate storage.
		if e.argCount == 2 && !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
			return false
		}
	} else {
		return false
	}
	renvoEmitNewMapHeaderPrimary(g)
	renvoAsmStorePrimaryStack(a, offset)
	if e.kind == renvoExprComposite {
		valueResolved := renvoResolveType(g.meta, resolved.elem)
		renvoNonNil(valueResolved)
		if !renvoTypeKindIsScalarInt(valueResolved.kind) && valueResolved.kind != renvoTypePointer && valueResolved.kind != renvoTypeInterface {
			return false
		}
		entryPtrOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		loc := renvoSliceLocation{offset: offset, mem: true, indirect: true}
		for i := 0; i < e.argCount; i++ {
			field := &ep.fields[e.firstArg+i]
			if !renvoEmitAppendDestPrimary(g, ep, &loc, entrySize) {
				return false
			}
			renvoAsmStorePrimaryStack(a, entryPtrOff)
			if field.key < 0 || !renvoEmitStringValueRegs(g, ep, field.key) {
				return false
			}
			renvoAsmPushStringRegs(a)
			renvoAsmLoadSecondaryStack(a, entryPtrOff)
			renvoAsmPopStoreStringMemSecondary(a, 0)
			valueOffset := renvoAddUnnamedLocal(g, resolved.elem)
			if !renvoEmitExprToLocal(g, ep, field.expr, valueOffset) {
				return false
			}
			renvoAsmLoadSecondaryStack(a, entryPtrOff)
			renvoAsmAddSecondaryImm(a, 16)
			renvoEmitCopyStackToMemSecondary(g, valueOffset, 0, renvoTypeSize(g.meta, resolved.elem))
		}
	}
	return true
}

func renvoEmitSliceReturnValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int, resultType int) bool {
	renvoNonNil(g, ep)
	if !renvoEmitSliceValueRegs(g, ep, idx) {
		return false
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
		if callee == renvoIdentAppend && e.argCount >= 1 {
			return renvoReturnedSliceCanReuseDescriptor(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
		}
		fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 && fnIndex < len(meta.funcs) {
			fn := &meta.funcs[fnIndex]
			if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistBytes") {
				return true
			}
			return renvoCallSliceResultCanReuseDescriptor(g, ep, idx, fnIndex)
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
	return renvoLocalIsCurrentFuncParam(g, localIndex)
}

func renvoCallSliceResultCanReuseDescriptor(g *renvoLinearGen, ep *renvoExprParse, idx int, fnIndex int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	if fnIndex < 0 || fnIndex >= len(g.meta.funcs) {
		return false
	}
	e := &ep.exprs[idx]
	fn := &g.meta.funcs[fnIndex]
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != renvoExprSelector {
			return false
		}
		receiverType := renvoInferParsedExprType(g, ep, callee.left)
		if renvoTypeIsSlice(g.meta, receiverType) && !renvoReturnedSliceCanReuseDescriptor(g, ep, callee.left) {
			return false
		}
	}
	for i := 0; i < e.argCount; i++ {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
		argType := renvoInferParsedExprType(g, ep, argIndex)
		if renvoTypeIsSlice(g.meta, argType) && !renvoReturnedSliceCanReuseDescriptor(g, ep, argIndex) {
			return false
		}
	}
	return true
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
	slackSize := 64
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
	indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	nonNilLabel := renvoAsmNewLabel(a)
	capOKLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	returnLabel := renvoAsmNewLabel(a)
	renvoAsmStorePrimarySecondaryStack(a, srcOff, lenOff)
	renvoAsmStackMem(a, capOff, 0x8948, 0x4d, 0x8d)
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
		renvoAsmStackMem(a, byteCountOff, 0x8948, 0x4d, 0x8d)
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
	if renvoTargetArch == renvoArchAmd64 {
		renvoAsmLoadPrimaryStack(a, destOff)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmLoadPrimaryStack(a, srcOff)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAsmLoadTertiaryStack(a, byteCountOff)
		renvoAsmEmit16(a, 0xa4f3)
	} else {
		renvoAsmStoreStackImm(a, indexOff, 0)
		renvoAsmMarkLabel(a, loopLabel)
		renvoAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
		renvoEmitAppendExpansionCopyElement(g, elemSize, srcOff, indexOff, destOff, indexOff)
		renvoAsmIncStack(a, indexOff)
		renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	}
	renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
	renvoAsmLoadTertiaryStack(a, capOff)
	renvoAsmMarkLabel(a, returnLabel)
	return true
}

func renvoEmitSliceValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
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
			baseOff := renvoAddUnnamedLocal(g, baseType)
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
			if elemSize != 1 {
				renvoAsmMulTertiaryImm(a, elemSize)
			}
			renvoAsmAddPrimaryTertiary(a)
			renvoAsmPopSecondary(a)
			renvoAsmPopTertiary(a)
			return true
		}
		return true
	}
	if e.kind == renvoExprIdent {
		if renvoBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
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
		if e.argCount >= 2 && callee == renvoIdentAppend {
			var stmt renvoStmt
			if !renvoEmitAppendAssignGeneral(g, &stmt, ep, 0) {
				return false
			}
			return renvoEmitSliceValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg))
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
			if conversion.kind == renvoTypeSlice && renvoTypeIsString(meta, renvoInferParsedExprType(g, ep, argIndex)) {
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
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !renvoEmitSliceLiteralBacking(g, ep, idx, sliceType, backingOff) {
		return false
	}
	renvoAsmPrimaryImm(a, e.argCount)
	renvoAsmPushPrimary(a)
	renvoAsmPrimaryBssAddr(a, backingOff)
	renvoAsmSecondaryImm(a, e.argCount)
	renvoAsmPopTertiary(a)
	return true
}
func renvoEmitSliceLiteralBacking(g *renvoLinearGen, ep *renvoExprParse, idx int, sliceType int, backingOff int) bool {
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
			renvoAsmPrimaryBssAddr(a, backingOff)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopStoreStringMemSecondary(a, disp)
			continue
		}
		if elemResolved.kind == renvoTypeInterface {
			tempOffset := renvoAddUnnamedLocal(g, elemType)
			if !renvoEmitInterfaceAssignToLocal(g, ep, field.expr, tempOffset) {
				return false
			}
			renvoEmitCopyStackToBss(g, tempOffset, backingOff+disp, elemSize)
			continue
		}
		if elemResolved.kind == renvoTypeStruct {
			addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
			renvoAsmPrimaryBssAddr(a, backingOff)
			renvoAsmCopyPrimaryToSecondary(a)
			if disp != 0 {
				renvoAsmAddSecondaryImm(a, disp)
			}
			renvoAsmStoreSecondaryStack(a, addrOffset)
			if !renvoEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if elemResolved.kind == renvoTypeArray || elemResolved.kind == renvoTypeMap || elemResolved.kind == renvoTypeSlice {
			tempOffset := renvoAddUnnamedLocal(g, elemType)
			if !renvoEmitTypedAssign(g, ep, field.expr, tempOffset) {
				return false
			}
			renvoAsmPrimaryBssAddr(a, backingOff+disp)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
			continue
		}
		if !renvoTypeKindIsScalarValue(elemResolved.kind) && elemResolved.kind != renvoTypePointer {
			return false
		}
		if elemResolved.kind == renvoTypeFloat64 {
			if !renvoEmitScalarExprForKind(g, ep, field.expr, renvoTypeFloat64) {
				return false
			}
		} else if !renvoEmitIntExpr(g, ep, field.expr) {
			return false
		}
		renvoAsmNormalizePrimaryForKind(a, elemResolved.kind)
		renvoAsmPushPrimary(a)
		renvoAsmPrimaryBssAddr(a, backingOff)
		renvoAsmCopyPrimaryToSecondary(a)
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
	backingSize := 32768
	backingConst := false
	lenConst := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1))
	if lenConst.ok && lenConst.value > 0 {
		backingSize = renvoStaticSliceBackingSize(lenConst.value*elemSize, elemSize)
		backingConst = true
	}
	if e.argCount == 3 {
		backingConst = false
		capConst := renvoEvalConstExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+2))
		if capConst.ok && capConst.value > 0 {
			backingSize = renvoStaticSliceBackingSize(capConst.value*elemSize, elemSize)
			backingConst = true
		}
	}
	if backingConst {
		zeroSize := 0
		if lenConst.ok && lenConst.value > 0 {
			zeroSize = lenConst.value * elemSize
		}
		renvoEmitMakeStaticRingPrimary(g, backingSize, zeroSize)
	} else {
		sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmLoadTertiaryStack(a, capOffset)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmCopyTertiaryToPrimary(a)
		renvoAsmStorePrimaryStack(a, sizeOffset)
		renvoEmitArenaAllocStackPrimary(g, sizeOffset)
		renvoEmitZeroDynamicMakeSlice(g, lenOffset, elemSize)
	}
	renvoAsmLoadSecondaryTertiaryStack(a, lenOffset, capOffset)
	return true
}

func renvoEmitZeroDynamicMakeSlice(g *renvoLinearGen, lenOffset int, elemSize int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadTertiaryStack(a, lenOffset)
	renvoAsmMulTertiaryImm(a, elemSize)
	renvoAsmCallLabel(a, renvoEnsureMakeZeroHelper(g))
}

func renvoEnsureMakeZeroHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.makeZeroEmitted {
		return g.makeZeroLabel
	}
	g.makeZeroEmitted = true
	g.makeZeroLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.makeZeroLabel)
	if renvoTargetArch == renvoArchAmd64 {
		// Preserve the result pointer and ABI call register while REP STOSB
		// clears RCX bytes beginning at RAX.
		renvoAsmEmitText(a, "\x50\x57\x50\x5f\x31\xc0\xf3\xaa\x5f\x58\xc3")
		renvoAsmMarkLabel(a, afterLabel)
		return g.makeZeroLabel
	}
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPushPrimary(a)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmJzPrimary(a, doneLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)
	renvoAsmAddSecondaryImm(a, 1)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmPushImm(a, 1)
	renvoAsmPopTertiary(a)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	renvoAsmPopPrimary(a)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.makeZeroLabel
}

func renvoMakeStaticRingSlotCount(backingSize int) int {
	if backingSize <= 4096 {
		return 3
	}
	if backingSize <= 65536 {
		return 2
	}
	return 1
}

func renvoEmitMakeStaticRingPrimary(g *renvoLinearGen, backingSize int, zeroSize int) {
	renvoNonNil(g)
	a := &g.asm
	slotCount := renvoMakeStaticRingSlotCount(backingSize)
	cursorOff := g.asm.bssSize
	dataOff := cursorOff + 8
	g.asm.bssSize += 8 + backingSize*slotCount
	noWrapLabel := renvoAsmNewLabel(a)
	renvoAsmLoadPrimaryBss(a, cursorOff)
	renvoAsmPushPrimary(a)
	renvoAsmIncPrimary(a)
	renvoAsmCmpPrimaryImm8(a, slotCount)
	renvoAsmJnzLabel(a, noWrapLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmMarkLabel(a, noWrapLabel)
	renvoAsmStorePrimaryBss(a, cursorOff)
	renvoAsmPopTertiary(a)
	renvoAsmMulTertiaryImm(a, backingSize)
	renvoAsmPrimaryBssAddr(a, dataOff)
	renvoAsmAddPrimaryTertiary(a)
	if zeroSize > 0 {
		if zeroSize > backingSize {
			zeroSize = backingSize
		}
		zeroSize = renvoAlignTo8(zeroSize)
		addrOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStorePrimaryStack(a, addrOff)
		renvoAsmCopyPrimaryToSecondary(a)
		if zeroSize <= 128 {
			renvoAsmPrimaryImm(a, 0)
			for at := 0; at < zeroSize; at += 8 {
				renvoAsmStorePrimaryMemSecondaryDisp(a, at)
			}
		} else {
			renvoAsmPrimaryImm(a, zeroSize)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmLoadPrimaryStack(a, addrOff)
			renvoAsmCallLabel(a, renvoEnsureMakeZeroHelper(g))
		}
		renvoAsmLoadPrimaryStack(a, addrOff)
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
	renvoAsmLoadPrimaryStack(a, destOff)
	renvoAsmCopyPrimaryToSecondary(a)
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
	if fieldResolved.kind == renvoTypeMap {
		return renvoEmitMapAssignToLocal(g, ep, idx, fieldType, destOffset)
	}
	if fieldResolved.kind == renvoTypeArray {
		e := &ep.exprs[idx]
		if e.kind != renvoExprComposite {
			return false
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
	if fieldResolved.kind == renvoTypeStruct || fieldResolved.kind == renvoTypeInterface {
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
	renvoAsmStorePrimaryStackSize(a, destOffset, renvoNativeScalarStorageSize(fieldResolved.kind))
	return true
}
func renvoEmitCopyStackToStack(g *renvoLinearGen, srcOffset int, destOffset int, size int) {
	renvoNonNil(g)
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

const renvoNativeCopyStackToStack = 1
const renvoNativeCopyStackToMem = 2
const renvoNativeCopyMemToStack = 3

func renvoEmitCopyNative(g *renvoLinearGen, srcOffset int, destOffset int, size int, mode int) {
	renvoNonNil(g)
	a := &g.asm
	for at := 0; at < size; at += renvoNativeIntSize {
		chunkSize := renvoNativeIntSize
		if size-at < chunkSize {
			chunkSize = size - at
		}
		if mode == renvoNativeCopyMemToStack {
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, at, chunkSize)
		} else {
			renvoAsmLoadPrimaryStack(a, srcOffset-at)
		}
		if mode == renvoNativeCopyStackToMem {
			renvoAsmStorePrimaryMemSecondaryDispSize(a, destOffset+at, chunkSize)
		} else {
			renvoAsmStorePrimaryStackSize(a, destOffset-at, chunkSize)
		}
	}
}

const renvoPushStack = 1
const renvoPushBss = 2

func renvoEmitPushWords(g *renvoLinearGen, offset int, size int, wordSize int, mode int) {
	renvoNonNil(g)
	size = renvoAlignValue(size, wordSize)
	for at := size - wordSize; at >= 0; at -= wordSize {
		if mode == renvoPushStack && wordSize == renvoNativeIntSize && (renvoTargetArch == renvoArchAmd64 || renvoTargetArch == renvoArch386) {
			renvoAsmPushStackWord(&g.asm, offset-at)
			continue
		}
		if mode == renvoPushStack {
			renvoAsmLoadPrimaryStack(&g.asm, offset-at)
		} else if mode == renvoPushBss {
			renvoAsmLoadPrimaryBss(&g.asm, offset+at)
		} else {
			renvoAsmLoadPrimaryMemSecondaryDisp(&g.asm, at)
		}
		renvoAsmPushPrimary(&g.asm)
	}
}
func renvoEmitIndexedStructToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int, offset int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	leftType := renvoInferParsedExprType(g, ep, e.left)
	sliceType := renvoResolveType(meta, leftType)
	renvoNonNil(sliceType)
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemType := renvoResolveType(meta, sliceType.elem)
	renvoNonNil(elemType)
	destResolved := renvoResolveType(meta, destType)
	renvoNonNil(destResolved)
	if elemType.kind != renvoTypeStruct || destResolved.kind != renvoTypeStruct {
		return false
	}
	elemSize := renvoTypeSize(meta, sliceType.elem)
	if renvoTypeSize(meta, destType) != elemSize {
		return false
	}
	if !renvoEmitIndexAddressPrimary(g, ep, idx) {
		return false
	}
	renvoAsmCopyPrimaryToSecondary(a)
	renvoEmitCopyMemSecondaryToStack(g, offset, elemSize)
	return true
}
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
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != renvoExprSelector {
			return -1, 0
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	wordCount := renvoEmitCallArgsReverse(g, ep, e, fn, receiverIndex)
	if wordCount < 0 {
		return -1, 0
	}
	wordCount++
	if receiverIndex >= 0 {
		words := renvoEmitMethodReceiverArgReverse(g, ep, receiverIndex, meta.params[fn.firstParam].typ)
		if words < 0 {
			words = renvoEmitMethodReceiverArgTokensReverse(g, receiverDotTok, meta.params[fn.firstParam].typ)
			if words < 0 {
				return -1, 0
			}
		}
		wordCount += words
	}
	return fnIndex, wordCount
}

func renvoEmitCallArgsReverse(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, fn *renvoFuncInfo, receiverIndex int) int {
	renvoNonNil(g, ep, e, fn)
	fixed := fn.paramCount
	if receiverIndex >= 0 {
		fixed--
	}
	wordCount := 0
	if e.nameStart == 0 && fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart == 1 {
		fixed--
		if e.argCount < fixed || !renvoEmitVariadicArgSliceReverse(g, ep, e.firstArg+fixed, e.argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
			return -1
		}
		wordCount = renvoBackendSliceWordCount
	} else {
		fixed = e.argCount
	}
	for i := fixed - 1; i >= 0; i-- {
		paramIndex := i
		if receiverIndex >= 0 {
			paramIndex++
		}
		words := renvoEmitCallParamArgReverse(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i), fn.firstParam+paramIndex)
		if words < 0 {
			return -1
		}
		wordCount += words
	}
	return wordCount
}
func renvoEmitStructCallToLocal(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int, offset int) bool {
	renvoNonNil(g, ep)
	if renvoIsInterfaceMethodCall(g, ep, idx) {
		return renvoEmitInterfaceMethodCall(g, ep, idx, offset, destType)
	}
	fnIndex, wordCount := renvoPrepareStructCall(g, ep, idx, destType)
	if fnIndex < 0 {
		return false
	}
	renvoAsmStackMem(&g.asm, offset, 0x8d48, 0x45, 0x85)
	renvoAsmPushPrimary(&g.asm)
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func renvoEmitStructCallToBss(g *renvoLinearGen, ep *renvoExprParse, idx int, destType int, offset int) bool {
	renvoNonNil(g, ep)
	fnIndex, wordCount := renvoPrepareStructCall(g, ep, idx, destType)
	if fnIndex < 0 {
		return false
	}
	renvoAsmPrimaryBssAddr(&g.asm, offset)
	renvoAsmPushPrimary(&g.asm)
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func renvoEmitUserCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if renvoIsInterfaceMethodCall(g, ep, idx) {
		return renvoEmitInterfaceMethodCall(g, ep, idx, 0, renvoInterfaceMethodCallResultType(g, ep, idx))
	}
	if renvoFunctionValueCalleeType(g, ep, e.left) != 0 {
		return renvoEmitFunctionValueCall(g, ep, idx)
	}
	fnIndex := renvoFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return renvoEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		return false
	}
	fn := &g.meta.funcs[fnIndex]
	if renvoEmitRuntimeArenaCall(g, ep, idx, fn) {
		return true
	}
	receiverIndex := -1
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != renvoExprSelector {
			return false
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	wordCount := renvoEmitCallArgsReverse(g, ep, &e, fn, receiverIndex)
	if wordCount < 0 {
		return false
	}
	if receiverIndex >= 0 {
		words := renvoEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			words = renvoEmitMethodReceiverArgTokensReverse(g, receiverDotTok, g.meta.params[fn.firstParam].typ)
			if words < 0 {
				return false
			}
		}
		wordCount += words
	}
	if fn.linkStatic != 0 {
		return renvoEmitLinkStaticCall(g, fn, wordCount)
	}
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
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
	} else {
		if e.kind != renvoExprIdent {
			return 0
		}
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
	i := meta.funcBuckets[hash%len(meta.funcBuckets)]
	for i >= 0 {
		fn := &meta.funcs[i]
		if fn.receiverType != 0 && renvoBytesEqualRange(p.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) && renvoMethodReceiverTypeMatches(meta, typ, fn.receiverType) {
			return i
		}
		i = meta.funcNext[i]
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
	if l.kind == renvoTypeMap {
		return renvoTypesEquivalent(meta, l.first, r.first) && renvoTypesEquivalent(meta, l.elem, r.elem)
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

func renvoEmitFunctionValueCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
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
	return renvoEmitFunctionValueDispatch(g, funcType, handleOffset, argOffsets)
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
	size := renvoTypeSize(g.meta, typ)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	wordSize := renvoCallWordSize(t.kind)
	renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
	return renvoAlignValue(size, wordSize) / wordSize
}

func renvoCallWordSize(kind int) int {
	if kind == renvoTypeArray {
		return renvoNativeIntSize
	}
	return renvoBackendValueSlotSize
}

func renvoEmitFunctionValueDispatch(g *renvoLinearGen, funcType int, handleOffset int, argOffsets []int) bool {
	renvoNonNil(g)
	meta := g.meta
	renvoNonNil(meta)
	doneLabel := renvoAsmNewLabel(&g.asm)
	funcInfo := renvoResolveType(meta, funcType)
	renvoNonNil(funcInfo)
	matched := false
	closureTagOffset := -1
	previousDeferPendingOffset := 0
	if g.emittingDefers {
		previousDeferPendingOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmCopyBssToStackSlot(&g.asm, g.panicDeferPendingOff, previousDeferPendingOffset)
	}
	hiddenResultOffset := 0
	resultType := funcInfo.elem
	if renvoTypeUsesHiddenResult(meta, resultType) {
		hiddenResultOffset = renvoAddUnnamedLocal(g, resultType)
		renvoZeroLocalAtOffset(g, hiddenResultOffset)
	}
	for fnIndex := 0; fnIndex < len(meta.funcs); fnIndex++ {
		mode := renvoFunctionValueMode(meta, fnIndex, funcType)
		direct := mode == renvoFunctionValueDirect || mode == renvoFunctionValueMethodExpression
		closure := mode == renvoFunctionValueClosure || mode == renvoFunctionValueBoundMethod
		if !direct && !closure {
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
		matched = true
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
		renvoAsmJcmpStackImm(&g.asm, compareOffset, fnIndex+1, nextLabel, 0x95)
		wordCount := 0
		for i := len(argOffsets) - 1; i >= 0; i-- {
			wordCount += renvoEmitTypedLocalArgReverse(g, argOffsets[i], g.meta.fields[funcInfo.first+i].typ)
		}
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
		if hiddenResultOffset > 0 {
			renvoAsmAddressPrimaryStack(&g.asm, hiddenResultOffset)
			renvoAsmPushPrimary(&g.asm)
			extra++
		}
		oldSuppress := g.suppressPanicCheck
		g.suppressPanicCheck = true
		if g.emittingDefers {
			renvoAsmPrimaryImm(&g.asm, 1)
			renvoAsmStorePrimaryBss(&g.asm, g.panicDeferPendingOff)
		}
		renvoEmitCallWithWordCount(g, fnIndex, wordCount+extra)
		if g.emittingDefers {
			renvoAsmLoadPrimaryStack(&g.asm, previousDeferPendingOffset)
			renvoAsmStorePrimaryBss(&g.asm, g.panicDeferPendingOff)
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
	if !matched {
		return false
	}
	// Calling a nil or otherwise invalid function value is not a valid return
	// path. Keep the result deterministic until the panic path handles it.
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
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_Exit") {
		return renvoEmitRuntimeExit(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaMark") {
		return renvoEmitRuntimeArenaMark(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaReset") {
		return renvoEmitRuntimeArenaReset(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistMark") {
		return renvoEmitRuntimeArenaPersistMark(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistReset") {
		return renvoEmitRuntimeArenaPersistReset(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistString") {
		return renvoEmitRuntimeArenaPersistString(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaPersistBytes") {
		return renvoEmitRuntimeArenaPersistBytes(g, ep, idx)
	}
	if renvoBytesEqualText(p.src, fn.nameStart, fn.nameEnd, "renvo_runtime_ArenaDiscard") {
		return renvoEmitRuntimeArenaDiscard(g, ep, idx)
	}
	return false
}

func renvoEmitRuntimeExit(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 1 || !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	return renvoEmitExitStatus(g)
}

func renvoEmitExitStatus(g *renvoLinearGen) bool {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmExit(a)
		return true
	}
	if renvoTargetArch == renvoArchAarch64 {
		if targetIsDarwin() {
			renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRax)
			renvoDarwinArm64CallImport(a, renvoDarwinImportExit)
		} else {
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPrimaryImm(a, 93)
			renvoAsmSyscall(a)
		}
		return true
	}
	if renvoTargetArch == renvoArchArm {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 1)
		renvoAsmSyscall(a)
		return true
	}
	if renvoTargetArch == renvoArch386 {
		if targetIsWindows() {
			renvoAsmPushPrimary(a)
			renvoWin386CallImport(a, renvoWinImportExitProcess)
		} else {
			renvoAsmCopyPrimaryToCallWord0(a)
			renvoAsmPrimaryImm(a, 1)
			renvoAsmSyscall(a)
		}
		return true
	}
	if renvoTargetArch != renvoArchAmd64 {
		return false
	}
	if targetIsWindows() {
		renvoAsmCopyPrimaryToTertiary(a)
		renvoWinAmd64CallImport(a, renvoWinImportExitProcess, 40)
	} else {
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmPrimaryImm(a, 60)
		renvoAsmSyscall(a)
	}
	return true
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
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	if g.deferReturnLabel <= 0 {
		return false
	}
	valueOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
	if !renvoEmitInterfaceAssignToLocal(g, ep, argIndex, valueOffset) {
		return false
	}
	return renvoEmitPanicState(g, valueOffset)
}

func renvoEmitPanicState(g *renvoLinearGen, valueOffset int) bool {
	renvoNonNil(g)
	noPrevious := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryBss(&g.asm, g.panicIDOff)
	renvoAsmJzPrimary(&g.asm, noPrevious)
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	nodeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(&g.asm, sizeOffset, 4*renvoBackendValueSlotSize)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(&g.asm, nodeOffset)
	renvoEmitStorePanicNodeField(g, nodeOffset, g.panicValueOff, 0)
	renvoEmitStorePanicNodeField(g, nodeOffset, g.panicTypeOff, renvoBackendValueSlotSize)
	renvoEmitStorePanicNodeField(g, nodeOffset, g.panicIDOff, 2*renvoBackendValueSlotSize)
	renvoEmitStorePanicNodeField(g, nodeOffset, g.panicPrevOff, 3*renvoBackendValueSlotSize)
	renvoAsmLoadPrimaryStack(&g.asm, nodeOffset)
	renvoAsmStorePrimaryBss(&g.asm, g.panicPrevOff)
	renvoAsmMarkLabel(&g.asm, noPrevious)
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset)
	renvoAsmStorePrimaryBss(&g.asm, g.panicValueOff)
	renvoAsmLoadPrimaryStack(&g.asm, valueOffset-renvoBackendValueSlotSize)
	renvoAsmStorePrimaryBss(&g.asm, g.panicTypeOff)
	renvoAsmLoadPrimaryBss(&g.asm, g.panicNextIDOff)
	renvoAsmIncPrimary(&g.asm)
	renvoAsmStorePrimaryBss(&g.asm, g.panicNextIDOff)
	renvoAsmStorePrimaryBss(&g.asm, g.panicIDOff)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmStorePrimaryBss(&g.asm, g.panicRecoveredOff)
	renvoAsmJmpLabel(&g.asm, g.deferReturnLabel)
	return true
}

func renvoEmitRuntimeFault(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	if g.meta.panicEnabled && g.deferReturnLabel > 0 {
		valueOffset := renvoAddUnnamedLocal(g, renvoBuiltinTypeInterface)
		renvoAsmStoreStackImm(a, valueOffset, 1)
		renvoAsmStoreStackImm(a, valueOffset-renvoBackendValueSlotSize, renvoRuntimeTypeTag(g.meta, renvoTypeInt))
		renvoEmitPanicState(g, valueOffset)
	} else {
		renvoAsmJmpLabel(a, renvoEnsureUncaughtRuntimeFaultHelper(g))
	}
}

func renvoEnsureUncaughtRuntimeFaultHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.runtimeFaultLabel > 0 {
		return g.runtimeFaultLabel - 1
	}
	a := &g.asm
	label := renvoAsmNewLabel(a)
	g.runtimeFaultLabel = label + 1
	after := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, after, label)
	renvoEmitStaticWrite(g, "panic\n", 2)
	renvoAsmPrimaryImm(a, 2)
	renvoEmitExitStatus(g)
	renvoAsmMarkLabel(a, after)
	return label
}

func renvoEmitRuntimeNonNilPrimary(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	if !g.meta.panicEnabled {
		if renvoTargetArch == renvoArchAmd64 {
			renvoAsmCallLabel(a, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeNonNilLabel, 0, "\x48\x85\xc0\x74\x01\xc3\xe9\x00\x00\x00\x00"))
		} else {
			renvoAsmCallLabel(a, renvoEnsureNonNilCheckHelper(g))
		}
		return
	}
	ok := renvoAsmNewLabel(a)
	renvoAsmJnzPrimary(a, ok)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(a, ok)
}

func renvoEnsureNonNilCheckHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchAmd64 {
		return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeNonNilLabel, 0, "\x48\x85\xc0\x74\x01\xc3\xe9\x00\x00\x00\x00")
	}
	if g.runtimeNonNilLabel > 0 {
		return g.runtimeNonNilLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.runtimeNonNilLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	ok := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmJnzPrimary(&g.asm, ok)
	renvoAsmJmpLabel(&g.asm, renvoEnsureUncaughtRuntimeFaultHelper(g))
	renvoAsmMarkLabel(&g.asm, ok)
	renvoAsmRet(&g.asm)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitRuntimeNonNilSecondary(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	if !g.meta.panicEnabled {
		if renvoTargetArch == renvoArchAmd64 {
			renvoAsmEmit24(a, 0xd5ff41)
			return
		}
		renvoAsmCallLabel(a, renvoEnsureNonNilSecondaryCheckHelper(g))
		return
	}
	renvoAsmPushSecondary(a)
	renvoAsmPopPrimary(a)
	renvoEmitRuntimeNonNilPrimary(g)
	renvoAsmCopyPrimaryToSecondary(a)
}

func renvoEnsureNonNilSecondaryCheckHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchAmd64 {
		return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeSecondaryLabel, 1, "\x48\x85\xd2\x74\x01\xc3\xe9\x00\x00\x00\x00")
	}
	if g.runtimeSecondaryLabel > 0 {
		return g.runtimeSecondaryLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.runtimeSecondaryLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	ok := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmPushSecondary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmJnzPrimary(&g.asm, ok)
	renvoAsmJmpLabel(&g.asm, renvoEnsureUncaughtRuntimeFaultHelper(g))
	renvoAsmMarkLabel(&g.asm, ok)
	renvoAsmRet(&g.asm)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoRuntimeNonNilLocalNeeded(g *renvoLinearGen, localIndex int) bool {
	renvoNonNil(g)
	if localIndex < renvoNativeIntSize*8-1 {
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
	if e.argCount != 2 || !renvoEmitSlicePtrLen(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAsmPushPrimary(&g.asm)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAsmCopyPrimaryToTertiary(&g.asm)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmLoadPrimaryIndexTertiarySize(&g.asm, size)
	if size == 4 {
		renvoAsmNormalizePrimaryForKind(&g.asm, renvoTypeInt32)
	}
	return true
}

func renvoEmitRuntimeTruncateSlice(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
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
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, renvoBackendValueSlotSize)
	return true
}

func renvoEmitRuntimeTrustPointer(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) bool {
	renvoNonNil(g, ep, e)
	for i := 0; i < e.argCount; i++ {
		arg := &ep.exprs[renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)]
		if arg.kind == renvoExprIdent {
			local := renvoFindLocalIndex(g, arg.nameStart, arg.nameEnd)
			if local >= 0 && local < renvoNativeIntSize*8-1 {
				g.checkedPointerLocals |= 1 << local
			}
		}
	}
	return true
}

func renvoEmitRuntimeNonNilLocalSecondary(g *renvoLinearGen, localIndex int) {
	renvoNonNil(g)
	if renvoRuntimeNonNilLocalNeeded(g, localIndex) {
		renvoEmitRuntimeNonNilSecondary(g)
	}
}

func renvoEmitBuiltinRecover(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
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
	exitLabel := renvoAsmNewLabel(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, g.panicIDOff)
	renvoAsmJzPrimary(a, normalLabel)
	renvoEmitStaticWrite(g, "panic: ", 2)
	renvoAsmLoadPrimaryBss(a, g.panicTypeOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPrimaryImm(a, renvoPanicTypeAssertionTag)
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, assertionLabel)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmPrimaryImm(a, renvoRuntimeTypeTag(g.meta, renvoTypeString))
	renvoAsmCmpTertiaryPrimarySet(a, 0x94)
	renvoAsmJnzPrimary(a, stringLabel)
	renvoEmitStaticWrite(g, "value", 2)
	renvoAsmJmpMarkLabel(a, exitLabel, stringLabel)
	renvoAsmLoadPrimaryBss(a, g.panicValueOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, renvoBackendValueSlotSize)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, g.panicValueOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmPopSecondary(a)
	renvoEmitWriteValueRegs(g, 2)
	renvoAsmJmpMarkLabel(a, exitLabel, assertionLabel)
	renvoEmitStaticWrite(g, "interface conversion failed", 2)
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
	renvoAsmLoadPrimaryBss(a, g.panicIDOff)
	renvoAsmJzPrimary(a, noneLabel)
	renvoAsmCopyBssToStackSlot(a, g.panicValueOff, offset)
	renvoAsmCopyBssToStackSlot(a, g.panicTypeOff, offset-renvoBackendValueSlotSize)
	renvoAsmStoreStackImm(a, g.panicRecoverAllowedOffset, 0)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmStorePrimaryBss(a, g.panicRecoveredOff)
	renvoAsmCopyBssToStackSlot(a, g.panicPrevOff, previousOffset)
	renvoAsmJzPrimary(a, clearLabel)
	renvoEmitLoadPanicNodeField(g, previousOffset, g.panicValueOff, 0)
	renvoEmitLoadPanicNodeField(g, previousOffset, g.panicTypeOff, renvoBackendValueSlotSize)
	renvoEmitLoadPanicNodeField(g, previousOffset, g.panicIDOff, 2*renvoBackendValueSlotSize)
	renvoEmitLoadPanicNodeField(g, previousOffset, g.panicPrevOff, 3*renvoBackendValueSlotSize)
	renvoAsmJmpMarkLabel(a, doneLabel, clearLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryBss(a, g.panicIDOff)
	renvoAsmStorePrimaryBss(a, g.panicPrevOff)
	renvoAsmJmpMarkLabel(a, doneLabel, noneLabel)
	renvoAsmStoreStackImm(a, offset, 0)
	renvoAsmStoreStackImm(a, offset-renvoBackendValueSlotSize, 0)
	renvoAsmMarkLabel(a, doneLabel)
	return true
}

func renvoEmitRuntimeArenaDiscard(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if renvoTargetArch != renvoArchAmd64 || renvoTargetOS != renvoOSLinux {
		renvoAsmPrimaryImm(&g.asm, 0)
		return true
	}
	a := &g.asm
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	doneLabel := renvoAsmNewLabel(a)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoAmd64AsmAddRaxImm32(a, 4095)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmStorePrimaryStack(a, startOff)
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1)) {
		return false
	}
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmLoadTertiaryStack(a, startOff)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAmd64AsmJccLabel(a, 0x8e, doneLabel)
	renvoAsmLoadPrimaryStack(a, startOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmSecondaryImm(a, 4)
	renvoAsmPrimaryImm(a, 28)
	renvoAsmSyscall(a)
	renvoAsmMarkLabel(a, doneLabel)
	renvoAsmPrimaryImm(a, 0)
	return true
}

func renvoEmitRuntimeArenaMark(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 0 {
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

func renvoEmitRuntimeArenaReset(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoStringHeapOffsets(g)
	a := &g.asm
	renvoAsmStorePrimaryBss(a, g.stringHeapOff)
	return true
}

func renvoEmitRuntimeArenaPersistMark(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 0 {
		return false
	}
	renvoEmitPersistentArenaReady(g)
	renvoAsmLoadPrimaryBss(&g.asm, g.stringHeapEndOff)
	return true
}

func renvoEmitRuntimeArenaPersistReset(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
		return false
	}
	renvoStringHeapOffsets(g)
	a := &g.asm
	if renvoTargetArch == renvoArchAmd64 && renvoTargetOS == renvoOSLinux {
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
	startOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	lenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmStorePrimaryStack(a, markOff)
	renvoAsmCopyBssToStackSlot(a, g.stringHeapEndOff, oldOff)
	renvoAsmLoadPrimaryStack(a, markOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
	renvoAsmLoadPrimaryStack(a, oldOff)
	renvoAmd64AsmAddRaxImm32(a, 4095)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmStorePrimaryStack(a, startOff)
	renvoAsmLoadPrimaryStack(a, markOff)
	renvoAmd64AsmAndRaxImm32(a, -4096)
	renvoAsmLoadTertiaryStack(a, startOff)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmStorePrimaryStack(a, lenOff)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAmd64AsmJccLabel(a, 0x8e, doneLabel)
	renvoAsmLoadPrimaryStack(a, startOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, lenOff)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmSecondaryImm(a, 4)
	renvoAsmPrimaryImm(a, 28)
	renvoAsmSyscall(a)
	renvoAsmMarkLabel(a, doneLabel)
}

func renvoAmd64AsmAddRaxImm32(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x0548)
	renvoAsmEmit32(a, imm)
}

func renvoAmd64AsmAndRaxImm32(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x2548)
	renvoAsmEmit32(a, imm)
}

func renvoEmitRuntimeArenaPersistString(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
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
	renvoNonNil(g, ep)
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitSliceValueRegs(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)) {
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
	renvoAsmLoadTertiaryStack(a, lenOff)
	return true
}

func renvoEmitCopyBytesToPersistent(g *renvoLinearGen, srcOff int, lenOff int, destOff int) {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchAmd64 {
		renvoAsmEmit8(a, 0x57)
		renvoAsmEmit8(a, 0x56)
		renvoAsmEmit8(a, 0x51)
		renvoAsmLoadPrimaryStack(a, destOff)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmLoadPrimaryStack(a, srcOff)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAsmLoadTertiaryStack(a, lenOff)
		renvoAsmEmit8(a, 0xfc)
		renvoAsmEmit16(a, 0xa4f3)
		renvoAsmEmit8(a, 0x59)
		renvoAsmEmit8(a, 0x5e)
		renvoAsmEmit8(a, 0x5f)
		return
	}
	indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmStoreStackImm(a, indexOff, 0)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
	renvoAsmLoadPrimaryTertiaryStack(a, srcOff, indexOff)
	renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
	renvoAsmPushPrimary(a)
	renvoAsmLoadSecondaryTertiaryStack(a, destOff, indexOff)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
	renvoAsmIncStack(a, indexOff)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
}

func renvoEmitPersistentAllocToPrimary(g *renvoLinearGen, sizeOff int) {
	renvoNonNil(g)
	a := &g.asm
	renvoAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	renvoAsmLoadTertiaryStack(a, sizeOff)
	renvoAsmSubPrimaryTertiary(a)
	renvoAsmStorePrimaryBss(a, g.stringHeapEndOff)
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

func renvoEmitLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if targetIsDarwin() {
		return renvoDarwinArm64EmitLinkStaticCall(g, fn, wordCount)
	}
	if renvoTargetOS != renvoOSWindows {
		return false
	}
	importID := renvoAsmAddWinStaticImport(&g.asm, fn.linkDLLStart, fn.linkDLLEnd, fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if renvoTargetArch == renvoArch386 {
		renvoWin386CallImport(&g.asm, importID)
		return true
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoWinArm64CallStaticImport(&g.asm, importID, wordCount)
		return true
	}
	if renvoTargetArch != renvoArchAmd64 {
		return false
	}
	renvoWinAmd64CallStaticImport(&g.asm, importID, wordCount)
	return true
}

func renvoWinAmd64CallStaticImport(a *renvoAsm, importID int, wordCount int) {
	renvoNonNil(a)
	if wordCount > 0 {
		renvoAsmPopTertiary(a)
	}
	if wordCount > 1 {
		renvoAsmPopSecondary(a)
	}
	if wordCount > 2 {
		renvoAsmEmit16(a, 0x5841)
	}
	if wordCount > 3 {
		renvoAsmEmit16(a, 0x5941)
	}
	stackWords := 0
	if wordCount > 4 {
		stackWords = wordCount - 4
	}
	// RENVO internal calls may leave the stack at either 16-byte parity while
	// evaluating an expression. Preserve the exact pending-argument pointer in
	// r10, align dynamically, then construct a fresh Win64 call area containing
	// shadow space, copied stack arguments, and a saved original rsp slot.
	renvoAsmEmit24(a, 0xe28949) // mov r10, rsp
	renvoAsmEmit4(a, 0x48, 0x83, 0xe4, 0xf0)
	savedRSPOff := 32 + stackWords*8
	allocation := renvoAlignValue(savedRSPOff+8, 16)
	if renvoAsmImmFits8Signed(allocation) {
		renvoAsmEmit4(a, 0x48, 0x83, 0xec, allocation)
	} else {
		renvoAsmEmit24(a, 0xec8148)
		renvoAsmEmit32(a, allocation)
	}
	for i := 0; i < stackWords; i++ {
		renvoWinAmd64LoadRAXFromR10(a, i*8)
		renvoWinAmd64StoreRAXToRSP(a, 32+i*8)
	}
	renvoWinAmd64StoreR10ToRSP(a, savedRSPOff)
	renvoAsmEmit16(a, 0x15ff)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
	renvoAsmEmit24(a, 0xc28949) // mov r10, rax
	renvoWinAmd64LoadRAXFromRSP(a, savedRSPOff)
	renvoAsmEmit24(a, 0xc48948) // mov rsp, rax
	if stackWords > 0 {
		adjust := stackWords * 8
		if renvoAsmImmFits8Signed(adjust) {
			renvoAsmEmit4(a, 0x48, 0x83, 0xc4, adjust)
		} else {
			renvoAsmEmit24(a, 0xc48148)
			renvoAsmEmit32(a, adjust)
		}
	}
	renvoAsmEmit24(a, 0xd0894c) // mov rax, r10
}

func renvoWinAmd64LoadRAXFromR10(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if offset <= 127 {
		renvoAsmEmit4(a, 0x49, 0x8b, 0x42, offset)
		return
	}
	renvoAsmEmit24(a, 0x828b49)
	renvoAsmEmit32(a, offset)
}

func renvoWinAmd64LoadRAXFromRSP(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if offset <= 127 {
		renvoAsmEmit5(a, 0x48, 0x8b, 0x44, 0x24, offset)
		return
	}
	renvoAsmEmit4(a, 0x48, 0x8b, 0x84, 0x24)
	renvoAsmEmit32(a, offset)
}

func renvoWinAmd64StoreRAXToRSP(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if offset <= 127 {
		renvoAsmEmit5(a, 0x48, 0x89, 0x44, 0x24, offset)
		return
	}
	renvoAsmEmit4(a, 0x48, 0x89, 0x84, 0x24)
	renvoAsmEmit32(a, offset)
}

func renvoWinAmd64StoreR10ToRSP(a *renvoAsm, offset int) {
	renvoNonNil(a)
	if offset <= 127 {
		renvoAsmEmit5(a, 0x4c, 0x89, 0x54, 0x24, offset)
		return
	}
	renvoAsmEmit4(a, 0x4c, 0x89, 0x94, 0x24)
	renvoAsmEmit32(a, offset)
}

func renvoEmitArbitrarySyscall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount < 1 || e.argCount > 7 {
		return false
	}
	if targetIsDarwin() {
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
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+i)
		if !renvoEmitSyscallArg(g, ep, argIndex) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
	}
	return renvoEmitSyscallFromStack(g, e.argCount)
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

func renvoEmitSyscallFromStack(g *renvoLinearGen, wordCount int) bool {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchAmd64 {
		renvoAsmPopPrimary(a)
		if wordCount > 1 {
			renvoAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			renvoAsmEmit8(a, 0x5e)
		}
		if wordCount > 3 {
			renvoAsmPopSecondary(a)
		}
		if wordCount > 4 {
			renvoAsmPopTertiary(a)
			renvoAsmEmit24(a, 0xca8949)
		}
		if wordCount > 5 {
			renvoAsmEmit16(a, 0x5841)
		}
		if wordCount > 6 {
			renvoAsmEmit16(a, 0x5941)
		}
		renvoAsmSyscall(a)
		return true
	}
	if renvoTargetArch == renvoArch386 {
		if wordCount > 6 {
			return false
		}
		renvoAsmPopPrimary(a)
		if wordCount > 1 {
			renvoAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			renvoAsmEmit8(a, 0x59)
		}
		if wordCount > 3 {
			renvoAsmPopSecondary(a)
		}
		if wordCount > 4 {
			renvoAsmEmit8(a, 0x5e)
		}
		if wordCount > 5 {
			renvoAsmEmit8(a, 0x5f)
		}
		renvoAsmSyscall(a)
		return true
	}
	if renvoTargetArch == renvoArchAarch64 {
		if targetIsDarwin() {
			if wordCount != 4 {
				return false
			}
			renvoAarch64AsmPopReg(a, 9)
			renvoAarch64AsmPopReg(a, 0)
			renvoAarch64AsmPopReg(a, 1)
			renvoAarch64AsmPopReg(a, 2)
			baseOff := a.bssSize
			a.bssSize += 8
			renvoAarch64AsmMovRegAbs(a, 3, baseOff, renvoAbsBssReloc)
			renvoDarwinArm64CallImport(a, renvoDarwinImportGetdirentries)
			return true
		}
		renvoAarch64AsmPopReg(a, renvoAarch64RegSys)
		if wordCount > 1 {
			renvoAarch64AsmPopReg(a, 0)
		}
		if wordCount > 2 {
			renvoAarch64AsmPopReg(a, 1)
		}
		if wordCount > 3 {
			renvoAarch64AsmPopReg(a, 2)
		}
		if wordCount > 4 {
			renvoAarch64AsmPopReg(a, 3)
		}
		if wordCount > 5 {
			renvoAarch64AsmPopReg(a, 4)
		}
		if wordCount > 6 {
			renvoAarch64AsmPopReg(a, 5)
		}
		renvoAarch64AsmEmit(a, 0xd4000001)
		return true
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopReg(a, renvoArmRegSys)
		if wordCount > 1 {
			renvoArmAsmPopReg(a, 0)
		}
		if wordCount > 2 {
			renvoArmAsmPopReg(a, 1)
		}
		if wordCount > 3 {
			renvoArmAsmPopReg(a, 2)
		}
		if wordCount > 4 {
			renvoArmAsmPopReg(a, 3)
		}
		if wordCount > 5 {
			renvoArmAsmPopReg(a, 4)
		}
		if wordCount > 6 {
			renvoArmAsmPopReg(a, 5)
		}
		renvoArmAsmEmit(a, 0xef000000)
		return true
	}
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRax)
		if wordCount > 1 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdi)
		}
		if wordCount > 2 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRsi)
		}
		if wordCount > 3 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdx)
		}
		if wordCount > 4 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
		}
		if wordCount > 5 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR8)
		}
		if wordCount > 6 {
			renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR9)
		}
		renvoAsmSyscall(a)
		return true
	}
	return false
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
		if resolved.kind == renvoTypeInterface {
			tempOffset := renvoAddUnnamedLocal(g, param.typ)
			if !renvoEmitInterfaceAssignToLocal(g, ep, idx, tempOffset) {
				return -1
			}
			renvoAsmPushStackWord(&g.asm, tempOffset-renvoBackendValueSlotSize)
			renvoAsmPushStackWord(&g.asm, tempOffset)
			return 2
		}
		source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
		renvoNonNil(source)
		if resolved.kind == renvoTypeFloat64 || source.kind == renvoTypeFloat64 {
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
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			actualExprType = g.locals[localIndex].typ
		}
	}
	actualExprResolved := renvoResolveType(meta, actualExprType)
	renvoNonNil(actualExprResolved)
	if receiver.kind == renvoTypePointer {
		if actualExprResolved.kind == renvoTypePointer {
			if !renvoEmitIntExpr(g, ep, idx) {
				return -1
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
		if size <= renvoBackendValueSlotSize {
			renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
			renvoAsmPushPrimary(a)
			return 1
		}
		renvoEmitPushWords(g, 0, size, renvoBackendValueSlotSize, 0)
		return size / renvoBackendValueSlotSize
	}
	return renvoEmitCallArgReverse(g, ep, idx)
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
func renvoEmitAddressPrimary(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
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
		return true
	}
	if e.kind == renvoExprIndex {
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		return true
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		if !renvoEmitIntExpr(g, ep, e.left) {
			return false
		}
		renvoEmitRuntimeNonNilPrimary(g)
		return true
	}
	return false
}
func renvoEmitVariadicArgSliceReverse(g *renvoLinearGen, ep *renvoExprParse, first int, count int, sliceType int) bool {
	renvoNonNil(g, ep)
	offset := renvoAddUnnamedLocal(g, sliceType)
	return renvoEmitVariadicArgsToLocal(g, ep, first, count, sliceType, offset) && renvoEmitTypedLocalArgReverse(g, offset, sliceType) == renvoBackendSliceWordCount
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
	if resolved.kind == renvoTypeStruct || resolved.kind == renvoTypeArray {
		return renvoEmitStructArgReverse(g, ep, idx, typ)
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprInt {
		value := renvoParseIntToken(p, e.tok)
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
	if renvoTargetArch != renvoArchAmd64 {
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
	for i := tuple.count - 1; i >= 0; i-- {
		field := g.meta.fields[tuple.first+i]
		size := renvoTypeCopySize(g.meta, field.typ)
		renvoEmitPushWords(g, offset-field.offset, size, renvoBackendValueSlotSize, renvoPushStack)
		wordCount += size / renvoBackendValueSlotSize
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
	wordSize := renvoCallWordSize(renvoResolveType(meta, typ).kind)
	wordCount := renvoAlignValue(size, wordSize) / wordSize
	if e.kind == renvoExprIdent {
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
			(elemType.kind != renvoTypeStruct && elemType.kind != renvoTypeArray) || renvoTypeSize(meta, sliceType.elem) != size {
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
	if e.kind == renvoExprComposite {
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitTypedAssign(g, ep, idx, offset) {
			return -1
		}
		renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
		return wordCount
	}
	if e.kind == renvoExprCall {
		offset := renvoAddUnnamedLocal(g, typ)
		if !renvoEmitStructCallToLocal(g, ep, idx, typ, offset) {
			return -1
		}
		renvoEmitPushWords(g, offset, size, wordSize, renvoPushStack)
		return wordCount
	}
	return -1
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
	if assignTok > stmt.startTok && !renvoAppendAssignLhsMatchesSource(p, stmt, ep, root, assignTok) {
		return renvoEmitAppendAssignDifferentSource(g, stmt, ep, root, assignTok)
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
	firstStart := root.tok + 1
	closeTok := renvoFindMatchingExprClose(p, root.tok+1, ep.end, '(', ')')
	if closeTok <= firstStart {
		return false
	}
	firstEnd := renvoFindExprBoundary(p, firstStart, closeTok)
	if firstEnd <= firstStart {
		return false
	}
	return renvoTokenRangesEqualSource(p, stmt.startTok, assignTok, firstStart, firstEnd)
}

func renvoTokenRangesEqualSource(p *renvoProgram, aStartTok int, aEndTok int, bStartTok int, bEndTok int) bool {
	renvoNonNil(p)
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
			typeTok := renvoFindAppendCompositeTypeToken(p, root.tok, stmt.endTok)
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
	if renvoTypeKindIsScalarValue(elem.kind) || elem.kind == renvoTypePointer {
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
	return false
}

func renvoEmitAppendLocalToLocation(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elem *renvoTypeInfo, elemType int, offset int) bool {
	renvoNonNil(g, locEp, loc, elem)
	a := &g.asm
	if renvoTypeKindIsScalarValue(elem.kind) || elem.kind == renvoTypePointer {
		elemSize := renvoScalarKindSize(elem.kind)
		renvoAsmPushStack(a, offset)
		if elem.kind == renvoTypePointer || renvoTargetArch == renvoArchAmd64 || renvoTargetArch == renvoArchAarch64 || elemSize == 1 || elemSize == 2 || elemSize == 4 {
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
	if elem.kind == renvoTypeStruct {
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
	if left.kind == renvoTypeFloat64 {
		return true
	}
	right := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, e.right))
	renvoNonNil(right)
	if right.kind == renvoTypeFloat64 {
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
		return typ.kind == renvoTypeFloat64
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
		return typ.kind == renvoTypeFloat64
	}
	return false
}
func renvoExprCanFoldConst(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 {
		return false
	}
	if idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == renvoExprInt {
		return true
	}
	if e.kind == renvoExprChar {
		return true
	}
	if e.kind == renvoExprBool {
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].constValid != 0
		}
		builtin := renvoEvalBuiltinConst(g, e.nameStart, e.nameEnd)
		if builtin.ok {
			return true
		}
		if renvoFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, renvoTokConst) >= 0 {
			return true
		}
		return false
	}
	if e.kind == renvoExprUnary {
		leftFold := renvoExprCanFoldConst(g, ep, e.left)
		return leftFold
	}
	if e.kind == renvoExprBinary {
		leftFold := renvoExprCanFoldConst(g, ep, e.left)
		if !leftFold {
			return false
		}
		rightFold := renvoExprCanFoldConst(g, ep, e.right)
		return rightFold
	}
	if e.kind == renvoExprCall {
		if e.argCount != 1 {
			return false
		}
		argIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
		argFold := renvoExprCanFoldConst(g, ep, argIndex)
		if !argFold {
			return false
		}
		calleeLeft := e.left
		if renvoConversionTypeFromExpr(g, ep, calleeLeft) != 0 {
			return true
		}
		calleeExpr := ep.exprs[calleeLeft]
		if calleeExpr.kind == renvoExprIdent {
			return renvoFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd) > 0
		}
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
	if elemSize == 1 {
		srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
		srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
		srcIndex := renvoAddUnnamedLocal(g, renvoTypeInt)
		if !renvoEmitSliceValueRegs(g, ep, valueIndex) {
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
		if !renvoEmitAppendDestPrimary(g, locEp, loc, elemSize) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
		renvoAsmIncStack(a, srcIndex)
		renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
		return true
	}
	srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcIndex := renvoAddUnnamedLocal(g, renvoTypeInt)
	destPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	destLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	headerOffset := 0
	if !renvoEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
	renvoAsmStoreStackImm(a, srcIndex, 0)
	if loc.mem {
		if !renvoEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		headerOffset = renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmStoreSecondaryStack(a, headerOffset)
		renvoEmitEnsureMemSlice(g, elemSize)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmStorePrimaryStack(a, destPtr)
		renvoAsmLoadPrimaryStackMemory(a, headerOffset, 8)
		renvoAsmStorePrimaryStack(a, destLen)
	} else if loc.global {
		renvoAsmCopyBssToStackSlot(a, loc.offset, destPtr)
		renvoAsmCopyBssToStackSlot(a, loc.offset+8, destLen)
	} else {
		renvoAsmCopyStackSlot(a, loc.offset, destPtr)
		renvoAsmCopyStackSlot(a, loc.offset-8, destLen)
	}
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmJgeStackStack(a, srcIndex, srcLen, doneLabel)
	renvoEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	renvoAsmIncStack(a, srcIndex)
	renvoAsmIncStack(a, destLen)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	renvoAsmLoadPrimaryStack(a, destLen)
	if loc.mem {
		renvoAsmLoadSecondaryStack(a, headerOffset)
		renvoAsmStorePrimaryMemSecondaryDisp(a, 8)
	} else if loc.global {
		renvoAsmStorePrimaryBss(a, loc.offset+8)
	} else {
		renvoAsmStorePrimaryStack(a, loc.offset-8)
	}
	return true
}
func renvoEmitAppendExpansionCopyElement(g *renvoLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	renvoNonNil(g)
	a := &g.asm
	if elemSize == 1 || elemSize == 2 || elemSize == 4 || elemSize == 8 {
		renvoAsmLoadPrimaryTertiaryStack(a, srcPtr, srcIndex)
		renvoAsmLoadPrimaryIndexTertiarySize(a, elemSize)
		renvoAsmPushPrimary(a)
		renvoAsmLoadSecondaryTertiaryStack(a, destPtr, destLen)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryTertiarySize(a, elemSize)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		renvoAsmLoadPrimaryTertiaryStack(a, srcPtr, srcIndex)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a, copyOff)
		renvoAsmPushPrimary(a)
		renvoAsmLoadSecondaryTertiaryStack(a, destPtr, destLen)
		renvoAsmMulTertiaryImm(a, elemSize)
		renvoAsmAddSecondaryTertiary(a)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryDisp(a, copyOff)
	}
}
func renvoFindAppendCompositeTypeToken(p *renvoProgram, openTok int, end int) int {
	renvoNonNil(p)
	if openTok < 0 || openTok >= end || !renvoTokCharIs(p, openTok, '(') {
		return -1
	}
	i := openTok + 1
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && renvoTokCharIs(p, i, ',') {
			typeTok := i + 1
			if renvoTokIsKind(p, typeTok, renvoTokIdent) && renvoTokCharIs(p, typeTok+1, '{') {
				return typeTok
			}
			return -1
		}
		if renvoTokCharIs(p, i, '(') {
			paren++
		} else if renvoTokCharIs(p, i, ')') {
			if paren == 0 {
				return -1
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
	return -1
}
func renvoEmitAppendScalarToLocation(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, elemKind int, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	elemSize := renvoScalarKindSize(elemKind)
	if !renvoEmitScalarExprForKind(g, ep, valueIndex, elemKind) {
		return false
	}
	renvoAsmPushPrimary(a)
	if elemKind == renvoTypePointer || renvoTargetArch == renvoArchAmd64 || renvoTargetArch == renvoArchAarch64 || elemSize == 1 || elemSize == 2 || elemSize == 4 {
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
	label := renvoEnsureAppendAddrHelper(g)
	if !renvoEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	renvoAsmSecondaryImm(&g.asm, elemSize)
	renvoAsmCallLabel(&g.asm, label)
	return true
}
func renvoEnsureAppendScalarHelper(g *renvoLinearGen, elemKind int) int {
	renvoNonNil(g)
	if renvoScalarKindSize(elemKind) == 1 {
		return renvoEnsureAppend8Helper(g)
	}
	return renvoEnsureAppend64Helper(g)
}
func renvoEmitAppendStringToLocation(g *renvoLinearGen, ep *renvoExprParse, locEp *renvoExprParse, loc *renvoSliceLocation, valueIndex int) bool {
	renvoNonNil(g, ep, locEp, loc)
	a := &g.asm
	renvoEnsureAppendAddrHelper(g)
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
	backingSize := renvoSliceBackingSize(elemSize)
	if renvoTargetArch == renvoArchWasm32 {
		backingSize = renvoWasm32FallbackSliceBackingSize
	}
	renvoEmitArenaAllocPrimary(g, backingSize)
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
func renvoEmitStringCompare(g *renvoLinearGen, ep *renvoExprParse, left int, right int, notEqual bool) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	label := renvoEnsureStringEqualHelper(g)
	rightExpr := &ep.exprs[right]
	if rightExpr.kind == renvoExprSelector {
		rightOff := renvoAddUnnamedLocal(g, renvoTypeString)
		if !renvoEmitStringValueRegs(g, ep, right) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(a, rightOff, rightOff-8)
		if !renvoEmitStringValueRegs(g, ep, left) {
			return false
		}
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
	if !renvoEmitTypedAssign(g, ep, e.left, left) || !renvoEmitTypedAssign(g, ep, e.right, right) {
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
		renvoAsmLoadPrimarySecondaryStack(a, left, left-8)
		renvoAsmPushStringRegs(a)
		renvoAsmLoadPrimarySecondaryStack(a, right, right-8)
		renvoAsmCopySecondaryToTertiary(a)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmPopCallWord1(a)
		renvoAsmCallLabel(a, renvoEnsureStringEqualHelper(g))
	} else if t.kind == renvoTypeComplex {
		renvoAsmJcmpStackStack(a, left, right, fail, 0x95)
		renvoAsmLoadPrimaryTertiaryStack(a, left-renvoBackendValueSlotSize, right-renvoBackendValueSlotSize)
		renvoAsmCmpTertiaryPrimarySet(a, 0x94)
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
	e := ep.exprs[idx]
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
	if destSlice.kind != renvoTypeSlice || srcSlice.kind != renvoTypeSlice {
		return false
	}
	elemSize := renvoTypeSize(g.meta, destSlice.elem)
	if elemSize != renvoTypeSize(g.meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	destLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcPtr := renvoAddUnnamedLocal(g, renvoTypeInt)
	srcLen := renvoAddUnnamedLocal(g, renvoTypeInt)
	copyCount := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, destPtr, destLen)
	if !renvoEmitSliceValueRegs(g, ep, srcIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, srcPtr, srcLen)
	renvoAsmStoreStackImm(a, copyCount, 0)
	loopLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmJgeStackStack(a, copyCount, destLen, doneLabel)
	renvoAsmJgeStackStack(a, copyCount, srcLen, doneLabel)
	renvoEmitAppendExpansionCopyElement(g, elemSize, srcPtr, copyCount, destPtr, copyCount)
	renvoAsmIncStack(a, copyCount)
	renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
	renvoAsmLoadPrimaryStack(a, copyCount)
	return true
}
func renvoEmitSliceBasePtrLenTokens(g *renvoLinearGen, p *renvoProgram, start int, end int, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, p, ep)
	meta := g.meta
	a := &g.asm
	if start+1 == end && renvoTokIsKind(p, start, renvoTokIdent) {
		nameStart := int(renvoTokStart(p, start))
		nameEnd := int(renvoTokEnd(p, start))
		localIndex := renvoFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !renvoTypeIsSlice(meta, g.locals[localIndex].typ) {
				return false
			}
			renvoAsmLoadPrimaryTertiaryStack(a, g.locals[localIndex].offset, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := renvoFindGlobalOffset(g, nameStart, nameEnd)
		globalType := renvoFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && renvoTypeIsSlice(meta, globalType) {
			renvoAsmLoadPrimaryBss(a, globalOffset+8)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && renvoTokIsKind(p, start, renvoTokIdent) && renvoTokCharIs(p, start+1, '.') && renvoTokIsKind(p, start+2, renvoTokIdent) {
		localIndex := renvoFindLocalIndex(g, int(renvoTokStart(p, start)), int(renvoTokEnd(p, start)))
		if localIndex < 0 {
			return false
		}
		fieldType := renvoStructFieldType(g, g.locals[localIndex].typ, int(renvoTokStart(p, start+2)), int(renvoTokEnd(p, start+2)))
		if !renvoTypeIsSlice(meta, fieldType) {
			return false
		}
		fieldOffset := renvoStructFieldOffset(g, g.locals[localIndex].typ, int(renvoTokStart(p, start+2)), int(renvoTokEnd(p, start+2)))
		if fieldOffset < 0 {
			return false
		}
		t := renvoResolveType(meta, g.locals[localIndex].typ)
		renvoNonNil(t)
		if t.kind == renvoTypePointer {
			renvoAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				renvoAsmAddSecondaryImm(a, fieldOffset)
			}
		} else {
			renvoAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		if renvoTargetArch == renvoArchWasm32 {
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmPopPrimary(a)
		} else {
			renvoAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		}
		return true
	}
	return renvoEmitSlicePtrLen(g, ep, idx)
}
func renvoEmitMapHeaderPtrLen(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	nilLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	renvoAsmJzPrimary(a, nilLabel)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmJmpMarkLabel(a, doneLabel, nilLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmMarkLabel(a, doneLabel)
	return
}

func renvoEmitMapPtrLen(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if !renvoEmitMapValuePrimary(g, ep, idx) {
		return false
	}
	renvoEmitMapHeaderPtrLen(g)
	return true
}

func renvoEmitDirectSelectorWords(g *renvoLinearGen, ep *renvoExprParse, idx int, primaryDisp int, tertiaryDisp int, size int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch != renvoArchAmd64 {
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
		renvoAsmMemDisp(&g.asm, tertiaryOffset, 0x8b48, 0x4a, 0x8a)
	}
	return true
}

func renvoEmitSlicePtrLen(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoResolveType(meta, renvoInferParsedExprType(g, ep, idx)).kind == renvoTypeMap {
		return renvoEmitMapPtrLen(g, ep, idx)
	}
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
		if renvoEmitDirectSelectorWords(g, ep, idx, 0, 8, renvoNativeIntSize) {
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		if renvoTargetArch == renvoArchWasm32 {
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
			renvoAsmCopyPrimaryToTertiary(a)
			renvoAsmPopPrimary(a)
		} else {
			renvoAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		}
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
		if renvoEmitDirectSelectorWords(g, ep, idx, 0, 16, renvoNativeIntSize) {
			return true
		}
		if !renvoEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 0)
		renvoAsmPushPrimary(a)
		renvoAsmLoadPrimaryMemSecondaryDisp(a, 16)
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmPopPrimary(a)
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
	elemSize := renvoTypeSize(meta, sliceType.elem)
	if sliceType.kind != renvoTypeArray && sliceType.kind != renvoTypeSlice {
		return false
	}
	baseExpr := &ep.exprs[indexExpr.left]
	if sliceType.kind == renvoTypeArray {
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
		} else if base.kind == renvoExprCall {
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
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmPopPrimary(a)
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
		if renvoTargetArch == renvoArchAmd64 {
			renvoAmd64CallIndexAddressHelper(a, elemSize)
		} else {
			renvoAsmCallLabel(a, renvoEnsureIndexAddressHelper(g, elemSize))
		}
		return true
	}
	renvoAsmPopTertiary(a)
	renvoEmitRuntimeBoundsCheck(g)
	renvoAsmCopySecondaryToTertiary(a)
	renvoAsmPopPrimary(a)
	if elemSize != 1 {
		renvoAsmMulTertiaryImm(a, elemSize)
	}
	renvoAsmAddPrimaryTertiary(a)
	return true
}

func renvoEnsureIndexAddressHelper(g *renvoLinearGen, elemSize int) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchAmd64 {
		if elemSize == 1 {
			return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeByteIndexLabel, 3, "\x48\x39\xd1\x73\x04\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00")
		}
		if elemSize == 8 {
			return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeWordIndexLabel, 4, "\x48\x39\xd1\x73\x08\x48\xc1\xe1\x03\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00")
		}
		return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeWideIndexLabel, 5, "\x48\x39\xd1\x73\x08\x48\x6b\xc9\x48\x48\x01\xc8\xc3\xe9\x00\x00\x00\x00")
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
	after := renvoAsmNewLabel(&g.asm)
	negative := renvoAsmNewLabel(&g.asm)
	invalid := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmPushPrimary(&g.asm)
	renvoAsmPushSecondary(&g.asm)
	renvoAsmCopyTertiaryToPrimary(&g.asm)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	renvoAsmCmpTertiaryPrimarySet(&g.asm, 0x9d)
	renvoAsmJzPrimary(&g.asm, negative)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	renvoAsmCmpTertiaryPrimarySet(&g.asm, 0x9c)
	renvoAsmJzPrimary(&g.asm, invalid)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	if elemSize != 1 {
		renvoAsmMulTertiaryImm(&g.asm, elemSize)
	}
	renvoAsmAddPrimaryTertiary(&g.asm)
	renvoAsmRet(&g.asm)
	renvoAsmMarkLabel(&g.asm, negative)
	renvoAsmPopTertiary(&g.asm)
	renvoAsmMarkLabel(&g.asm, invalid)
	renvoAsmPopTertiary(&g.asm)
	renvoAsmJmpLabel(&g.asm, renvoEnsureUncaughtRuntimeFaultHelper(g))
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitRuntimeBoundsCheck(g *renvoLinearGen) {
	renvoNonNil(g)
	a := &g.asm
	if !g.meta.panicEnabled {
		if renvoTargetArch == renvoArchAmd64 {
			renvoAsmEmit24(a, 0xd4ff41)
			return
		}
		renvoAsmCallLabel(a, renvoEnsureBoundsCheckHelper(g))
		return
	}
	done := renvoAsmNewLabel(a)
	if renvoTargetArch == renvoArchAmd64 {
		renvoAsmCallLabel(a, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeBoundsLabel, 2, "\x48\x89\xc2\x48\x39\xc8\x73\x01\xc3\xe9\x00\x00\x00\x00"))
	} else {
		renvoAsmCallLabel(a, renvoEnsureBoundsCheckHelper(g))
	}
	renvoAsmJnzPrimary(a, done)
	renvoEmitRuntimeFault(g)
	renvoAsmMarkLabel(a, done)
}

func renvoEmitSliceBoundsChecks(g *renvoLinearGen, lowOff int, highOff int, maxOff int, capOff int) {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchAmd64 {
		renvoAsmLoadPrimaryStack(a, capOff)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmLoadPrimaryStack(a, maxOff)
		renvoAsmCopyPrimaryToTertiary(a)
		renvoAsmLoadPrimaryStack(a, highOff)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadPrimaryStack(a, lowOff)
		renvoAsmCallLabel(a, renvoAmd64EnsureRuntimeCheck(g, &g.runtimeSliceBoundsLabel, 6, "\x48\x39\xc2\x72\x0b\x48\x39\xd1\x72\x06\x48\x39\xcf\x72\x01\xc3\xe9\x00\x00\x00\x00"))
		if g.meta.panicEnabled {
			done := renvoAsmNewLabel(a)
			renvoAsmJnzPrimary(a, done)
			renvoEmitRuntimeFault(g)
			renvoAsmMarkLabel(a, done)
		}
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
	if renvoTargetArch == renvoArchAmd64 {
		return renvoAmd64EnsureRuntimeCheck(g, &g.runtimeBoundsLabel, 2, "\x48\x89\xc2\x48\x39\xc8\x73\x01\xc3\xe9\x00\x00\x00\x00")
	}
	if g.runtimeBoundsLabel > 0 {
		return g.runtimeBoundsLabel - 1
	}
	label := renvoAsmNewLabel(&g.asm)
	g.runtimeBoundsLabel = label + 1
	after := renvoAsmNewLabel(&g.asm)
	invalid := renvoAsmNewLabel(&g.asm)
	renvoAsmJmpMarkLabel(&g.asm, after, label)
	renvoAsmCopyPrimaryToSecondary(&g.asm)
	renvoAsmPushTertiary(&g.asm)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	renvoAsmCmpTertiaryPrimarySet(&g.asm, 0x9d)
	renvoAsmJzPrimary(&g.asm, invalid)
	renvoAsmPopPrimary(&g.asm)
	renvoAsmCopySecondaryToTertiary(&g.asm)
	renvoAsmCmpTertiaryPrimarySet(&g.asm, 0x9c)
	if !g.meta.panicEnabled {
		valid := renvoAsmNewLabel(&g.asm)
		renvoAsmJnzPrimary(&g.asm, valid)
		renvoAsmJmpLabel(&g.asm, renvoEnsureUncaughtRuntimeFaultHelper(g))
		renvoAsmMarkLabel(&g.asm, valid)
		renvoAsmRet(&g.asm)
		renvoAsmMarkLabel(&g.asm, invalid)
		renvoAsmPopTertiary(&g.asm)
		renvoAsmJmpLabel(&g.asm, renvoEnsureUncaughtRuntimeFaultHelper(g))
		renvoAsmMarkLabel(&g.asm, after)
		return label
	}
	renvoAsmRet(&g.asm)
	renvoAsmMarkLabel(&g.asm, invalid)
	renvoAsmPopTertiary(&g.asm)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmRet(&g.asm)
	renvoAsmMarkLabel(&g.asm, after)
	return label
}

func renvoEmitBuiltinDelete(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	mapIndex := renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)
	return renvoEmitMapEntryAddress(g, ep, mapIndex, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1), 2)
}

func renvoEmitMapEntryAddress(g *renvoLinearGen, ep *renvoExprParse, mapIndex int, keyIndex int, mode int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	mapType := renvoInferParsedExprType(g, ep, mapIndex)
	if renvoResolveType(g.meta, mapType).kind != renvoTypeMap {
		return false
	}
	keyPtrOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	keyLenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	mapLenOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	entryOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	headerOff := renvoAddUnnamedLocal(g, renvoTypeInt)
	if !renvoEmitMapValuePrimary(g, ep, mapIndex) {
		return false
	}
	renvoAsmStorePrimaryStack(a, headerOff)
	if !renvoEmitStringValueRegs(g, ep, keyIndex) {
		return false
	}
	renvoAsmStorePrimarySecondaryStack(a, keyPtrOff, keyLenOff)
	renvoAsmStoreStackImm(a, indexOff, 0)
	loopLabel := renvoAsmNewLabel(a)
	notFoundLabel := renvoAsmNewLabel(a)
	foundLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	entrySize := renvoMapEntrySize
	renvoAsmLoadPrimaryStack(a, headerOff)
	renvoAsmJzPrimary(a, notFoundLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoAsmLoadPrimaryStackMemory(a, headerOff, 8)
	renvoAsmStorePrimaryStack(a, mapLenOff)
	renvoAsmJgeStackStack(a, indexOff, mapLenOff, notFoundLabel)
	renvoAsmLoadTertiaryStack(a, indexOff)
	renvoAsmMulTertiaryImm(a, entrySize)
	renvoAsmLoadPrimaryStackMemory(a, headerOff, 0)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmAddSecondaryTertiary(a)
	renvoAsmStoreSecondaryStack(a, entryOff)
	renvoAsmLoadPrimaryStack(a, keyPtrOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmPushStack(a, keyLenOff)
	renvoAsmPopCallWord1(a)
	renvoAsmLoadPrimaryStackMemory(a, entryOff, 0)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopSecondary(a)
	renvoAsmCallLabel(a, renvoEnsureStringEqualHelper(g))
	renvoAsmJnzPrimary(a, foundLabel)
	renvoAsmIncStack(a, indexOff)
	renvoAsmJmpMarkLabel(a, loopLabel, foundLabel)
	if mode == 2 {
		renvoAsmLoadPrimaryStackMemory(a, headerOff, 8)
		renvoAsmPushImm(a, 1)
		renvoAsmPopTertiary(a)
		renvoAsmSubPrimaryTertiary(a)
		renvoAsmStorePrimaryStack(a, mapLenOff)
		renvoAsmStorePrimaryMemSecondaryDisp(a, 8)
		renvoAsmLoadTertiaryStack(a, mapLenOff)
		renvoAsmMulTertiaryImm(a, entrySize)
		renvoAsmLoadPrimaryStackMemory(a, headerOff, 0)
		renvoAsmAddPrimaryTertiary(a)
		renvoAsmStorePrimaryStack(a, keyPtrOff)
		for at := 0; at < entrySize; at += 8 {
			renvoAsmLoadPrimaryStackMemory(a, keyPtrOff, at)
			renvoAsmPushPrimary(a)
			renvoAsmLoadSecondaryStack(a, entryOff)
			renvoAsmPopPrimary(a)
			renvoAsmStorePrimaryMemSecondaryDisp(a, at)
		}
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpLabel(a, doneLabel)
	}
	renvoAsmLoadPrimaryStack(a, entryOff)
	renvoAsmJmpMarkLabel(a, doneLabel, notFoundLabel)
	if mode == 1 {
		renvoAsmLoadPrimaryStack(a, headerOff)
		renvoAsmCmpPrimaryImm8(a, 0)
		nonNilLabel := renvoAsmNewLabel(a)
		renvoAsmJnzLabel(a, nonNilLabel)
		renvoAsmPrimaryImm(a, 2)
		if !renvoEmitExitStatus(g) {
			return false
		}
		renvoAsmMarkLabel(a, nonNilLabel)
		loc := renvoSliceLocation{offset: headerOff, typ: mapType, mem: true, indirect: true, ok: true}
		if !renvoEmitAppendDestPrimary(g, ep, &loc, entrySize) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadPrimaryStack(a, keyPtrOff)
		renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
		renvoAsmLoadPrimaryStack(a, keyLenOff)
		renvoAsmStorePrimaryMemSecondaryDisp(a, 8)
		renvoAsmCopySecondaryToPrimary(a)
	} else {
		renvoAsmPrimaryImm(a, 0)
	}
	renvoAsmMarkLabel(a, doneLabel)
	return true
}

func renvoEmitIndexExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	baseResolved := renvoResolveType(meta, renvoInferParsedExprType(g, ep, e.left))
	renvoNonNil(baseResolved)
	if baseResolved.kind == renvoTypeMap {
		if !renvoEmitMapEntryAddress(g, ep, e.left, e.right, 0) {
			return false
		}
		zeroLabel := renvoAsmNewLabel(a)
		doneLabel := renvoAsmNewLabel(a)
		renvoAsmJzPrimary(a, zeroLabel)
		renvoAsmCopyPrimaryToSecondary(a)
		valueType := renvoResolveType(meta, baseResolved.elem)
		renvoNonNil(valueType)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 16, renvoScalarKindSize(valueType.kind))
		renvoAsmJmpMarkLabel(a, doneLabel, zeroLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmMarkLabel(a, doneLabel)
		return true
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
		if !renvoTypeKindIsScalarValue(elem.kind) && elem.kind != renvoTypePointer && elem.kind != renvoTypeMap {
			return false
		}
		if !renvoEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadPrimaryMemSecondaryDispSize(a, 0, renvoScalarKindSize(elem.kind))
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

func renvoFindLocalIndex(g *renvoLinearGen, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if g.localCacheStart == nameStart && g.localCacheCount == g.localCount {
		return g.localCacheIndex
	}
	nameHash := renvoHashRange(g.prog.src, nameStart, nameEnd)
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].nameHash == nameHash && renvoBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			g.localCacheStart = nameStart
			g.localCacheCount = g.localCount
			g.localCacheIndex = i
			return i
		}
	}
	return -1
}
func renvoStructFieldIndex(g *renvoLinearGen, typ int, nameStart int, nameEnd int) int {
	renvoNonNil(g)
	if renvoLoadStructFieldPath(g, typ, nameStart, nameEnd) {
		return g.fieldIndex
	}
	return -1
}

func renvoLoadStructFieldPath(g *renvoLinearGen, typ int, nameStart int, nameEnd int) bool {
	renvoNonNil(g)
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
		return renvoStructFieldIndex(g, typ, field.nameStart, field.nameEnd)
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
		if renvoBytesEqualRange(g.prog.src, g.globals[i].nameStart, g.globals[i].nameEnd, nameStart, nameEnd) {
			return g.globals[i].offset
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
			value := renvoParseIntToken(g.prog, s.initStart)
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

func renvoLocalCapturedInCurrentFunction(g *renvoLinearGen, nameStart int, nameEnd int) bool {
	renvoNonNil(g)
	meta := g.meta
	p := g.prog
	renvoNonNil(meta)
	renvoNonNil(p)
	if g.bindingClosureCaptures || nameEnd <= nameStart {
		return false
	}
	outer := &meta.funcs[g.currentFunc]
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
	size := renvoTypeSize(g.meta, typ)
	if size < renvoBackendValueSlotSize {
		size = renvoBackendValueSlotSize
	}
	captureOff := 0
	if renvoLocalCapturedInCurrentFunction(g, nameStart, nameEnd) {
		g.stackUsed = renvoAlignTo8(g.stackUsed + renvoBackendValueSlotSize)
		renvoRecordStackPeak(g)
		captureOff = g.stackUsed
		renvoAllocateCapturedCell(g, captureOff, size)
	}
	g.stackUsed = renvoAlignTo8(g.stackUsed + size)
	renvoRecordStackPeak(g)
	offset := g.stackUsed
	if g.localCount >= len(g.locals) {
		renvoGrowLocalTable(g)
	}
	nameHash := 0
	if nameEnd > nameStart {
		nameHash = renvoHashRange(g.prog.src, nameStart, nameEnd)
	}
	g.locals[g.localCount] = renvoLocalInfo{nameStart: nameStart, nameEnd: nameEnd, nameHash: nameHash, offset: offset, captureOff: captureOff, typ: typ, size: size}
	g.localCount++
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
	if toCell {
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
	for i := 0; i < g.localCount; i++ {
		renvoMoveCapturedLocal(g, i, toCell)
	}
}

func renvoZeroLocalAtOffset(g *renvoLinearGen, offset int) {
	renvoNonNil(g)
	a := &g.asm
	size := 8
	typ := renvoTypeInt
	for i := 0; i < g.localCount; i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := renvoResolveType(g.meta, typ)
	renvoNonNil(t)
	if t.kind == renvoTypeSlice {
		renvoInitEmptySliceStack(g, offset)
		return
	}
	if renvoTargetArch == renvoArchAmd64 && size >= 64 {
		// Large zero values are common in the compiler's parser and metadata
		// structures.  A counted store avoids expanding each zero value into a
		// separate frame-relative instruction.
		renvoAsmAddressCallWord0Stack(a, offset)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmPushImm(a, (size+7)/8)
		renvoAsmPopTertiary(a)
		renvoAsmEmit3(a, 0xf3, 0x48, 0xab)
	} else {
		renvoAsmPrimaryImm(a, 0)
		step := renvoBackendValueSlotSize
		if t.kind == renvoTypeArray {
			step = renvoNativeIntSize
		}
		for at := 0; at < size; at += step {
			renvoAsmStorePrimaryStack(a, offset-at)
		}
	}
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
	i := meta.funcBuckets[hash%len(meta.funcBuckets)]
	for i >= 0 {
		f := meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && renvoBytesEqualRange(p.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			if !wantMethod || renvoMethodReceiverTypeMatches(g.meta, wantReceiverType, f.receiverType) {
				e.right = i + 1
				return i
			}
		}
		i = meta.funcNext[i]
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
	return renvoInterfaceMethodResultType(g, renvoInferParsedExprType(g, ep, selector.left), selector)
}

func renvoInterfaceMethodResultType(g *renvoLinearGen, interfaceType int, selector *renvoExpr) int {
	renvoNonNil(g, selector)
	iface := renvoResolveType(g.meta, interfaceType)
	renvoNonNil(iface)
	for required := iface.first; required < iface.count; {
		end := renvoStatementLineEnd(g.prog, required, iface.count)
		if renvoTokCharIs(g.prog, required+1, '(') {
			if renvoBytesEqualRange(g.prog.src, int(renvoTokStart(g.prog, required)), int(renvoTokEnd(g.prog, required)), selector.nameStart, selector.nameEnd) {
				var parsed renvoTypeResult
				renvoParseFuncSignatureInto(g.meta, g.prog, required+1, end, &parsed)
				return renvoResolveType(g.meta, parsed.typ).elem
			}
		} else {
			embedded := renvoParseType(g.meta, g.prog, required, end)
			if embedded.typ != 0 {
				if result := renvoInterfaceMethodResultType(g, embedded.typ, selector); result != 0 {
					return result
				}
			}
		}
		required = end
	}
	if iface.first >= iface.count && renvoBytesEqualText(g.prog.src, selector.nameStart, selector.nameEnd, "Error") {
		return renvoTypeString
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
		for i := call.argCount - 1; i >= 0; i-- {
			words := renvoEmitCallParamArgReverse(g, ep, renvo_runtime_UnsafeIntAt(ep.args, call.firstArg+i), fn.firstParam+i+1)
			if words < 0 {
				return false
			}
			wordCount += words
		}
		if receiverSize <= renvoBackendValueSlotSize {
			renvoAsmPushStackWord(&g.asm, receiverOffset)
			wordCount++
		} else {
			renvoAsmLoadSecondaryStack(&g.asm, receiverOffset)
			renvoEmitPushWords(g, 0, receiverSize, renvoBackendValueSlotSize, 0)
			wordCount += renvoAlignValue(receiverSize, renvoBackendValueSlotSize) / renvoBackendValueSlotSize
		}
		if usesHiddenResult {
			renvoAsmStackMem(&g.asm, resultOffset, 0x8d48, 0x45, 0x85)
			renvoAsmPushPrimary(&g.asm)
			wordCount++
		}
		renvoEmitCallWithWordCount(g, fnIndex, wordCount)
		renvoAsmJmpMarkLabel(&g.asm, doneLabel, nextLabel)
	}
	if !matched {
		return false
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

// Architecture target dispatch wrappers.
func renvoEmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EmitScalarFunction(g, fnInfoIndex)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EmitScalarFunction(g, fnInfoIndex)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEmitScalarFunction(g, fnInfoIndex)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitScalarFunction(g, fnInfoIndex)
	}
	return renvoAmd64EmitScalarFunction(g, fnInfoIndex)
}

func renvoEmitScalarFunctionScratch(g *renvoLinearGen, fnInfoIndex int) bool {
	renvoNonNil(g)
	g.checkedPointerLocals = 0
	g.invalidatedPointerLocals = 0
	persistentCapacity := renvoLinearPersistentCapacity(g)
	typeCount := len(g.meta.types)
	fieldCount := len(g.meta.fields)
	captureCount := len(g.meta.captures)
	mark := renvo_runtime_ArenaMark()
	ok := renvoEmitScalarFunction(g, fnInfoIndex)
	if len(g.meta.captures) == captureCount {
		renvoTruncTypes(&g.meta.types, typeCount)
		renvoTruncFields(&g.meta.fields, fieldCount)
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
	// The remaining slices are either fixed-size or completely populated before
	// function emission begins. Only slices which can grow while a function is
	// emitted need to prevent the scratch arena from being rewound.
	return cap(a.code) + cap(a.labelPos) + cap(a.labelSet) + cap(a.relocs) + cap(a.absRelocs) + cap(a.symbols) + cap(a.symbolName) + cap(a.winImports) + cap(a.darwinImports) + cap(a.darwinImportLabels) + cap(a.darwinImportUsed) + cap(a.data) + cap(g.breakLabels) + cap(g.continueLabels) + cap(m.types) + cap(m.fields) + cap(m.captures)
}

func renvoStoreIncomingCallWord(g *renvoLinearGen, word int, offset int) {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32StoreParamWord(g, word, offset)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64StoreParamWord(g, word, offset)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmStoreParamWord(g, word, offset)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386StoreParamWord(g, word, offset)
		return
	}
	renvoAmd64StoreParamWord(g, word, offset)
}
func renvoAsmPrimaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRaxImm(a, imm)
		return
	}
	renvoAmd64AsmMovRaxImm(a, imm)
}
func renvoAsmPrimaryImm64(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRaxImm64(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRaxImm64(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRaxImm64(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRaxImm64(a, imm)
		return
	}
	renvoAsmEmit16(a, 0xb848)
	renvoAsmEmit64(a, imm)
}
func renvoAsmSecondaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRdxImm(a, imm)
		return
	}
	if imm == 0 {
		renvoAsmEmit16(a, 0xd231)
		return
	}
	if renvoAsmImmFits8Signed(imm) {
		renvoAsmEmit2(a, 0x6a, imm)
		renvoAsmPopSecondary(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			renvoAsmEmit8(a, 0xba)
			renvoAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			renvoAsmEmit24(a, 0xc2c748)
			renvoAsmEmit32(a, imm)
			return
		}
	}
	renvoAsmEmit16(a, 0xba48)
	renvoAsmEmit64(a, imm)
}
func renvoAsmPrimaryDataAddr(a *renvoAsm, dataOff int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRaxDataAddr(a, dataOff)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRaxDataAddr(a, dataOff)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRaxDataAddr(a, dataOff)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRaxDataAddr(a, dataOff)
		return
	}
	renvoAsmEmit24(a, 0x058d48)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, dataOff, 0)
}
func renvoAsmPrimaryBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRaxBssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRaxBssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRaxBssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRaxBssAddr(a, bssOff)
		return
	}
	renvoAsmEmit24(a, 0x058d48)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}
func renvoAsmScratchBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovR10BssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovR10BssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovR10BssAddr(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovR10BssAddr(a, bssOff)
		return
	}
	renvoAmd64AsmMovR10BssAddr(a, bssOff)
}
func renvoAsmLoadPrimaryBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadRaxBss(a, bssOff)
		return
	}
	renvoAmd64AsmLoadRaxBss(a, bssOff)
}
func renvoAsmStorePrimaryBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreRaxBss(a, bssOff)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmStoreRaxBss(a, bssOff)
		return
	}
	renvoAmd64AsmStoreRaxBss(a, bssOff)
}
func renvoAsmCopyPrimaryToCallWord0(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRdiRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRdiRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRdiRax(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRdiRax(a)
		return
	}
	renvoAmd64AsmMovRdiRax(a)
}
func renvoAsmCopySecondaryToPrimary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRaxRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRaxRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRaxRdx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRaxRdx(a)
		return
	}
	renvoAmd64AsmMovRaxRdx(a)
}
func renvoAsmCopyPrimaryToCallWord1(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovRsiRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRsiRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovRsiRax(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovRsiRax(a)
		return
	}
	renvoAmd64AsmMovRsiRax(a)
}

func renvoAsmCopyPrimaryToCallWord4(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovR8Rax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovR8Rax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovR8Rax(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovR8Rax(a)
		return
	}
	renvoAmd64AsmMovR8Rax(a)
}
func renvoAsmCopyPrimaryToCallWord5(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmMovR9Rax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovR9Rax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMovR9Rax(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMovR9Rax(a)
		return
	}
	renvoAmd64AsmMovR9Rax(a)
}
func renvoAsmAddSecondaryTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmAddRdxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmAddRdxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmAddRdxRcx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmAddRdxRcx(a)
		return
	}
	renvoAmd64AsmAddRdxRcx(a)
}
func renvoAsmSyscall(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmSyscall(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmSyscall(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmSyscall(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmSyscall(a)
		return
	}
	renvoAmd64AsmSyscall(a)
}
func renvoAsmPopCallWord0(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmPopRdi(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmPopRdi(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmPopRdi(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmPopRdi(a)
		return
	}
	renvoAmd64AsmPopRdi(a)
}
func renvoAsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	renvoAmd64AsmStackMem(a, offset, base, disp8, disp32)
}
func renvoAsmAddSecondaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmAddRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmAddRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmAddRdxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmAddRdxImm(a, imm)
		return
	}
	renvoAmd64AsmAddRdxImm(a, imm)
}
func renvoAsmMemDisp(a *renvoAsm, disp int, op int, disp8 int, disp32 int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	renvoAmd64AsmMemDisp(a, disp, op, disp8, disp32)
}
func renvoAsmLoadQwordPrimaryIndexTertiary8(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	renvoAmd64AsmLoadQwordRaxIndexRcx8(a)
}
func renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	renvoAmd64AsmLoadQwordRaxIndexRcxDisp(a, disp)
}
func renvoAsmLoadPrimaryMemSecondaryDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	renvoAmd64AsmLoadRaxMemRdxDisp(a, disp)
}
func renvoAsmLoadPrimaryMemSecondaryDispSize(a *renvoAsm, disp int, size int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	renvoAmd64AsmLoadRaxMemRdxDispSize(a, disp, size)
}
func renvoAsmLoadBytePrimaryIndexTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadByteRaxIndexRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadByteRaxIndexRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadByteRaxIndexRcx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadByteRaxIndexRcx(a)
		return
	}
	renvoAmd64AsmLoadByteRaxIndexRcx(a)
}
func renvoAsmLoadPrimaryIndexTertiarySize(a *renvoAsm, size int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	renvoAmd64AsmLoadRaxIndexRcxSize(a, size)
}
func renvoAsmStorePrimaryMemSecondaryTertiary8(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxMemRdxRcx8(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreRaxMemRdxRcx8(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreRaxMemRdxRcx8(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmStoreRaxMemRdxRcx8(a)
		return
	}
	renvoAmd64AsmStoreRaxMemRdxRcx8(a)
}
func renvoAsmStorePrimaryMemSecondaryDisp(a *renvoAsm, disp int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	renvoAmd64AsmStoreRaxMemRdxDisp(a, disp)
}
func renvoAsmStorePrimaryMemSecondaryDispSize(a *renvoAsm, disp int, size int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	renvoAmd64AsmStoreRaxMemRdxDispSize(a, disp, size)
}
func renvoAsmNormalizePrimaryForKind(a *renvoAsm, kind int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmNormalizeRaxForKind(a, kind)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmNormalizeRaxForKind(a, kind)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmNormalizeRaxForKind(a, kind)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmNormalizeRaxForKind(a, kind)
		return
	}
	renvoAmd64AsmNormalizeRaxForKind(a, kind)
}
func renvoAsmIncMemSecondary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmIncMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmIncMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmIncMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmIncMemRdx(a)
		return
	}
	renvoAmd64AsmIncMemRdx(a)
}
func renvoAsmDecMemSecondary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmDecMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmDecMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmDecMemRdx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmDecMemRdx(a)
		return
	}
	renvoAmd64AsmDecMemRdx(a)
}
func renvoAsmBoolNotPrimary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmBoolNotRax(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmBoolNotRax(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmBoolNotRax(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmBoolNotRax(a)
		return
	}
	renvoAmd64AsmBoolNotRax(a)
}
func renvoAsmCmpPrimaryImm8(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmCmpRaxImm8(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmCmpRaxImm8(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmCmpRaxImm8(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmCmpRaxImm8(a, imm)
		return
	}
	renvoAmd64AsmCmpRaxImm8(a, imm)
}
func renvoAsmAddPrimaryTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmAddRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmAddRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmAddRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmAddRaxRcx(a)
		return
	}
	renvoAmd64AsmAddRaxRcx(a)
}
func renvoAsmSubPrimaryTertiary(a *renvoAsm) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmSubRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmSubRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmSubRaxRcx(a)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmSubRaxRcx(a)
		return
	}
	renvoAmd64AsmSubRaxRcx(a)
}
func renvoAsmShlTertiaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmShlRcxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmShlRcxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmShlRcxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmShlRcxImm(a, imm)
		return
	}
	renvoAmd64AsmShlRcxImm(a, imm)
}
func renvoAsmShlPrimaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmShlRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmShlRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmShlRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmShlRaxImm(a, imm)
		return
	}
	renvoAmd64AsmShlRaxImm(a, imm)
}
func renvoAsmSarPrimaryImm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmSarRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmSarRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmSarRaxImm(a, imm)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmSarRaxImm(a, imm)
		return
	}
	renvoAmd64AsmSarRaxImm(a, imm)
}
func renvoAsmDivLeftTertiaryRightPrimary(a *renvoAsm, mod bool) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmDivLeftRcxRightRax(a, mod)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmDivLeftRcxRightRax(a, mod)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmDivLeftRcxRightRax(a, mod)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmDivLeftRcxRightRax(a, mod)
		return
	}
	renvoAmd64AsmDivLeftRcxRightRax(a, mod)
}
func renvoAsmCmpTertiaryPrimarySet(a *renvoAsm, setcc int) {
	renvoNonNil(a)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32AsmCmpRcxRaxSet(a, setcc)
		return
	}
	if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmCmpRcxRaxSet(a, setcc)
		return
	}
	if renvoTargetArch == renvoArchArm {
		renvoArmAsmCmpRcxRaxSet(a, setcc)
		return
	}
	if renvoTargetArch == renvoArch386 {
		renvo386AsmCmpRcxRaxSet(a, setcc)
		return
	}
	renvoAmd64AsmCmpRcxRaxSet(a, setcc)
}
func renvoEmitSwitchStringCaseTest(g *renvoLinearGen, valueOffset int, lenOffset int, ep *renvoExprParse, idx int, matchLabel int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, idx, matchLabel)
	}
	return renvoAmd64EmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, idx, matchLabel)
}
func renvoEmitPrimaryTertiaryOp(g *renvoLinearGen, tok int) bool {
	renvoNonNil(g)
	divide := renvoTokCharIs(g.prog, tok, '/')
	mod := renvoTokCharIs(g.prog, tok, '%')
	if (divide || mod) && !g.meta.panicEnabled {
		renvoAsmCallLabel(&g.asm, renvoEnsureSignedDivisionHelper(g, mod))
		return true
	}
	done := -1
	if divide || mod {
		renvoEmitRuntimeNonNilPrimary(g)
		if renvoTargetArch != renvoArchAmd64 {
			done = renvoEmitSignedDivisionOverflowGuard(g, mod)
		}
	}
	ok := false
	if renvoTargetArch == renvoArchWasm32 {
		ok = renvoWasm32EmitRaxRcxOp(g, tok)
	} else if renvoTargetArch == renvoArchAarch64 {
		ok = renvoAarch64EmitRaxRcxOp(g, tok)
	} else if renvoTargetArch == renvoArchArm {
		ok = renvoArmEmitRaxRcxOp(g, tok)
	} else if renvoTargetArch == renvoArch386 {
		ok = renvo386EmitRaxRcxOp(g, tok)
	} else {
		ok = renvoAmd64EmitRaxRcxOp(g, tok)
	}
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
	after := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, after, label)
	nonzero := renvoAsmNewLabel(a)
	renvoAsmJnzPrimary(a, nonzero)
	renvoAsmJmpLabel(a, renvoEnsureUncaughtRuntimeFaultHelper(g))
	renvoAsmMarkLabel(a, nonzero)
	done := -1
	if renvoTargetArch != renvoArchAmd64 {
		done = renvoEmitSignedDivisionOverflowGuard(g, mod)
	}
	renvoAsmDivLeftTertiaryRightPrimary(a, mod)
	if done >= 0 {
		renvoAsmMarkLabel(a, done)
	}
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, after)
	return label
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
	renvoAsmShlPrimaryImm(a, renvoNativeIntSize*8-1)
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
func renvoEmitCompareJump(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr, label int, jumpIfTrue bool) bool {
	renvoNonNil(g, ep, e)
	if renvoBinaryComparesInterface(g, ep, e) {
		return false
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitCompareJump(g, ep, e, label, jumpIfTrue)
	}
	return renvoAmd64EmitCompareJump(g, ep, e, label, jumpIfTrue)
}
func renvoEmitStringValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
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
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitStringValueRegs(g, ep, idx)
	}
	return renvoGenericEmitStringValueRegs(g, ep, idx)
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
		leftType := renvoInferParsedExprType(g, ep, e.left)
		t := renvoResolveType(meta, leftType)
		renvoNonNil(t)
		if t.kind != renvoTypeSlice {
			return false
		}
		elem := renvoResolveType(meta, t.elem)
		renvoNonNil(elem)
		if elem.kind != renvoTypeString {
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
		renvoAsmShlTertiaryImm(a, 4)
		renvoAsmCopyPrimaryToSecondary(a)
		renvoAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		renvoAsmAddSecondaryTertiary(a)
		if renvoTargetArch == renvoArchAarch64 || renvoTargetArch == renvoArchWasm32 {
			renvoAsmPushPrimary(a)
			renvoAsmLoadPrimaryMemSecondaryDisp(a, 8)
			renvoAsmCopyPrimaryToSecondary(a)
			renvoAsmPopPrimary(a)
		} else {
			renvoAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		}
		return true
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
	g.stringHeapDataOff = g.stringHeapOff + 16
	g.asm.bssSize += 16 + renvoStringArenaSize(g)
}

func renvoStringArenaSize(g *renvoLinearGen) int {
	renvoNonNil(g)
	if g.arenaSize > 0 {
		return g.arenaSize
	}
	return renvoDefaultArenaSize(renvoTarget)
}

func renvoEmitArenaAllocPrimary(g *renvoLinearGen, size int) {
	renvoNonNil(g)
	label := renvoEnsureArenaAllocHelper(g)
	renvoAsmPrimaryImm(&g.asm, size)
	renvoAsmCallLabel(&g.asm, label)
}

func renvoEmitArenaAllocStackPrimary(g *renvoLinearGen, sizeOff int) {
	renvoNonNil(g)
	label := renvoEnsureArenaAllocHelper(g)
	renvoAsmLoadPrimaryStack(&g.asm, sizeOff)
	renvoAsmCallLabel(&g.asm, label)
}

func renvoEnsureArenaAllocHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	a := &g.asm
	if g.arenaAllocEmitted {
		return g.arenaAllocLabel
	}
	g.arenaAllocEmitted = true
	g.arenaAllocLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	readyLabel := renvoAsmNewLabel(a)
	oomLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.arenaAllocLabel)
	renvoStringHeapOffsets(g)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
	renvoAsmJnzPrimary(a, readyLabel)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	renvoAsmStorePrimaryBss(a, g.stringHeapOff)
	renvoAsmMarkLabel(a, readyLabel)
	renvoAsmLoadPrimaryBss(a, g.stringHeapOff)
	renvoAsmPushPrimary(a)
	renvoAsmPushPrimary(a)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmPopTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
	renvoAsmJzPrimary(a, oomLabel)
	renvoAsmPushTertiary(a)
	renvoAsmPrimaryBssAddr(a, g.stringHeapDataOff+renvoStringArenaSize(g))
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9e)
	renvoAsmJzPrimary(a, oomLabel)
	renvoAsmCopyTertiaryToPrimary(a)
	renvoAsmStorePrimaryBss(a, g.stringHeapOff)
	renvoAsmPopPrimary(a)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, oomLabel)
	renvoAsmPopPrimary(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.arenaAllocLabel
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

func renvoSliceBackingSize(elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if renvoFixedTarget != 0 {
		count := 4096
		maxSize := 65536
		if elemSize == 1 {
			count = 65536
			maxSize = 65536
		} else if elemSize >= 16 {
			count = 4096
			maxSize = 65536
		}
		size := elemSize * count
		if size < 8192 {
			size = 8192
		}
		if size > maxSize {
			size = maxSize
		}
		if size < elemSize {
			size = elemSize
		}
		return size
	}
	count := 8192
	maxSize := 65536
	size := elemSize * count
	if size < 4096 {
		return 4096
	}
	if size > maxSize {
		return maxSize
	}
	return size
}

func renvoAmd64ArenaSliceBackingSize(elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if renvoFixedTarget != 0 {
		size := elemSize * 64
		if size < 256 {
			size = 256
		}
		if size > 4096 {
			size = 4096
		}
		return size
	}
	size := elemSize * 2048
	if size < 2048 {
		return 2048
	}
	return size
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
	if renvoTargetArch != renvoArchAmd64 {
		indexOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoEmitArenaAllocStackPrimary(g, lenOff)
		renvoAsmStorePrimaryStack(a, destOff)
		renvoAsmStoreStackImm(a, indexOff, 0)
		loopLabel := renvoAsmNewLabel(a)
		doneLabel := renvoAsmNewLabel(a)
		renvoAsmMarkLabel(a, loopLabel)
		renvoAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
		renvoAsmLoadPrimaryTertiaryStack(a, srcOff, indexOff)
		renvoAsmLoadPrimaryIndexTertiarySize(a, 1)
		renvoAsmPushPrimary(a)
		renvoAsmLoadSecondaryTertiaryStack(a, destOff, indexOff)
		renvoAsmPopPrimary(a)
		renvoAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
		renvoAsmIncStack(a, indexOff)
		renvoAsmJmpMarkLabel(a, loopLabel, doneLabel)
		renvoAsmLoadPrimarySecondaryStack(a, destOff, lenOff)
		return true
	}
	renvoEmitArenaAllocStackPrimary(g, lenOff)
	renvoAsmStorePrimaryStack(a, destOff)
	renvoAsmLoadPrimaryStack(a, destOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmLoadPrimaryStack(a, srcOff)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmLoadTertiaryStack(a, lenOff)
	renvoAsmEmit16(a, 0xa4f3)
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
	byteSliceType := renvoAddType(g.meta, renvoTypeSlice, renvoTypeByte, 0, 0, renvoBackendSliceValueSize, 0, 0)
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
	byteSliceType := renvoAddType(g.meta, renvoTypeSlice, renvoTypeByte, 0, 0, renvoBackendSliceValueSize, 0, 0)
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
	byteSliceType := renvoAddType(g.meta, renvoTypeSlice, renvoTypeByte, 0, 0, renvoBackendSliceValueSize, 0, 0)
	offset := renvoAddUnnamedLocal(g, byteSliceType)
	renvoZeroLocalAtOffset(g, offset)
	loc := renvoSliceLocation{offset: offset, typ: byteSliceType, ok: true}
	if !renvoEmitStringConcatIntoLocation(g, left, leftIndex, &loc) || !renvoEmitStringConcatIntoLocation(g, right, rightIndex, &loc) {
		return false
	}
	return renvoEmitStringConcatLocationValueRegs(g, offset)
}

func renvoEmitStringConcatLocationValueRegs(g *renvoLinearGen, offset int) bool {
	renvoNonNil(g)
	a := &g.asm
	if renvoTargetArch == renvoArchAmd64 {
		destOff := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoEmitArenaAllocStackPrimary(g, offset-8)
		renvoAsmStorePrimaryStack(a, destOff)
		renvoAsmLoadPrimaryStack(a, destOff)
		renvoAsmCopyPrimaryToCallWord0(a)
		renvoAsmLoadPrimaryStack(a, offset)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAsmLoadTertiaryStack(a, offset-8)
		renvoAsmEmit16(a, 0xa4f3)
		renvoAsmLoadPrimarySecondaryStack(a, destOff, offset-8)
		return true
	}
	renvoAsmPushStack(a, offset)
	renvoAsmLoadSecondaryStack(a, offset-8)
	renvoAsmPopPrimary(a)
	return true
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
	a := &g.asm
	fieldResolved := renvoResolveType(g.meta, fieldType)
	renvoNonNil(fieldResolved)
	if fieldResolved.kind == renvoTypeMap {
		return renvoEmitMapCompositeFieldToMem(g, ep, idx, fieldType, addrOffset, fieldOffset)
	}
	if fieldResolved.kind == renvoTypeSlice {
		if !renvoEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmPushSliceRegs(a)
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoAsmPopStoreSliceMemSecondary(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == renvoTypeString {
		if !renvoEmitStringValueRegs(g, ep, idx) {
			return false
		}
		renvoAsmPushStringRegs(a)
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoAsmPopStoreStringMemSecondary(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == renvoTypeStruct || fieldResolved.kind == renvoTypeArray || fieldResolved.kind == renvoTypeInterface {
		tempOffset := renvoAddTypedLocal(g, 0, 0, fieldType)
		if !renvoEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		renvoAsmLoadSecondaryStack(a, addrOffset)
		renvoEmitCopyStackToMemSecondary(g, tempOffset, fieldOffset, renvoTypeSize(g.meta, fieldType))
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, idx, fieldResolved.kind) {
		return false
	}
	renvoAsmLoadSecondaryStack(a, addrOffset)
	renvoAsmStorePrimaryMemSecondaryDispSize(a, fieldOffset, renvoNativeScalarStorageSize(fieldResolved.kind))
	return true
}

func renvoEmitMapCompositeFieldToMem(g *renvoLinearGen, ep *renvoExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	renvoNonNil(g, ep)
	tempOffset := renvoAddTypedLocal(g, 0, 0, fieldType)
	if !renvoEmitMapAssignToLocal(g, ep, idx, fieldType, tempOffset) {
		return false
	}
	renvoAsmLoadPrimarySecondaryStack(&g.asm, tempOffset, addrOffset)
	renvoAsmStorePrimaryMemSecondaryDisp(&g.asm, fieldOffset)
	return true
}

func renvoEmitStructReturnExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	resultType := g.meta.funcs[g.currentFunc].resultType
	resultKind := renvoResolveType(g.meta, resultType).kind
	renvoNonNil(resultKind)
	if resultKind != renvoTypeStruct || idx >= 0 && idx < len(ep.exprs) && (ep.exprs[idx].kind == renvoExprAssert || ep.exprs[idx].kind == renvoExprSelector || ep.exprs[idx].kind == renvoExprIdent && renvoFindLocalIndex(g, ep.exprs[idx].nameStart, ep.exprs[idx].nameEnd) < 0) {
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
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitStructReturnExpr(g, ep, idx)
	}
	return renvoAmd64EmitStructReturnExpr(g, ep, idx)
}
func renvoEmitNamedConversionCall(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
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
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitNamedConversionCall(g, ep, idx)
	}
	return renvoAmd64EmitNamedConversionCall(g, ep, idx)
}
func renvoLinearMarkFunc(g *renvoLinearGen, fnIndex int) {
	renvoNonNil(g)
	if fnIndex < 0 || fnIndex >= len(g.funcReachable) {
		return
	}
	if g.funcReachable[fnIndex] {
		return
	}
	g.funcReachable[fnIndex] = true
	g.funcQueue = append(g.funcQueue, fnIndex)
	if renvoCompilerStripSymbols && renvoTargetArch != renvoArchWasm32 {
		return
	}
	src := g.meta.prog.src
	nameStart := g.meta.funcs[fnIndex].nameStart
	nameEnd := g.meta.funcs[fnIndex].nameEnd
	renvoAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func renvoInitFuncQueue(g *renvoLinearGen, count int) {
	renvoNonNil(g)
	g.funcReachable = make([]bool, count, count)
	for i := 0; i < count; i++ {
		g.funcReachable[i] = false
	}
	g.funcQueue = make([]int, 0, count)
}

func renvoEmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	renvoNonNil(g)
	renvoLinearMarkFunc(g, fnIndex)
	if renvoTargetArch == renvoArchWasm32 {
		renvoWasm32EmitCallWithWordCount(g, fnIndex, wordCount)
	} else if renvoTargetArch == renvoArchAarch64 {
		renvoAarch64EmitCallWithWordCount(g, fnIndex, wordCount)
	} else if renvoTargetArch == renvoArchArm {
		renvoArmEmitCallWithWordCount(g, fnIndex, wordCount)
	} else if renvoTargetArch == renvoArch386 {
		renvo386EmitCallWithWordCount(g, fnIndex, wordCount)
	} else {
		renvoAmd64EmitCallWithWordCount(g, fnIndex, wordCount)
	}
	renvoEmitPostCallPanicCheck(g)
}
func renvoExprIsPointerCompositeLiteral(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind != renvoExprUnary || !renvoTokCharIs(g.prog, e.tok, '&') || e.left < 0 || e.left >= len(ep.exprs) {
		return false
	}
	return ep.exprs[e.left].kind == renvoExprComposite
}
func renvoEmitPointerCompositeLiteral(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	e := &ep.exprs[idx]
	innerIndex := e.left
	inner := &ep.exprs[innerIndex]
	elemType := renvoInferParsedExprType(g, ep, innerIndex)
	resolved := renvoResolveType(g.meta, elemType)
	renvoNonNil(resolved)
	if resolved.kind != renvoTypeStruct {
		return false
	}
	size := renvoTypeSize(g.meta, elemType)
	if size <= 0 {
		return false
	}

	a := &g.asm
	sizeOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	addrOffset := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmStoreStackImm(a, sizeOffset, size)
	renvoEmitPersistentAllocToPrimary(g, sizeOffset)
	renvoAsmStorePrimaryStack(a, addrOffset)

	// Persistent storage begins in zero-filled BSS, but explicitly initialize
	// every value slot so this remains correct if the allocator later reuses
	// storage.
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPrimaryImm(a, 0)
	for at := 0; at < size; at += renvoBackendValueSlotSize {
		renvoAsmStorePrimaryMemSecondaryDisp(a, at)
	}
	for i := 0; i < inner.argCount; i++ {
		field := ep.fields[inner.firstArg+i]
		fieldIndex := renvoCompositeStructFieldIndex(g, elemType, &field, i)
		if fieldIndex < 0 {
			return false
		}
		fieldInfo := &g.meta.fields[fieldIndex]
		if fieldInfo.typ == 0 || !renvoEmitCompositeFieldToMem(g, ep, field.expr, fieldInfo.typ, addrOffset, fieldInfo.offset) {
			return false
		}
	}
	renvoAsmLoadPrimaryStack(a, addrOffset)
	return true
}
func renvoEmitMachineIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitIntExpr(g, ep, idx)
	}
	return renvoAmd64EmitIntExpr(g, ep, idx)
}

func renvoEmitIntExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx >= 0 && idx < len(ep.exprs) {
		e := &ep.exprs[idx]
		if e.kind == renvoExprBinary && renvoBinaryComparesInterface(g, ep, e) {
			return renvoEmitInterfaceCompare(g, ep, e)
		}
	}
	if idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprCall {
		callee := renvoExprIdentCode(g.prog, ep, ep.exprs[idx].left)
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
	if idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprAssert {
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
	if idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprFunc {
		return renvoEmitClosureValuePrimary(g, ep.exprs[idx].tok)
	}
	if idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == renvoExprSelector {
		fnIndex, expression := renvoMethodSelectorInfo(g, ep, idx)
		if fnIndex >= 0 {
			return renvoEmitMethodSelectorValuePrimary(g, ep, idx, fnIndex, expression)
		}
	}
	if renvoExprIsPointerCompositeLiteral(g, ep, idx) {
		return renvoEmitPointerCompositeLiteral(g, ep, idx)
	}
	if idx >= 0 && idx < len(ep.exprs) && renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind == renvoTypeMap {
		return renvoEmitMapValuePrimary(g, ep, idx)
	}
	return renvoEmitMachineIntExpr(g, ep, idx)
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

func renvoEmitComplexValueRegs(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == renvoExprAssert {
		offset := renvoAddUnnamedLocal(g, renvoInferParsedExprType(g, ep, idx))
		if !renvoEmitTypedAssign(g, ep, idx, offset) {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(&g.asm, offset, offset-renvoBackendValueSlotSize)
		return true
	}
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || renvoResolveType(g.meta, g.locals[localIndex].typ).kind != renvoTypeComplex {
			return false
		}
		renvoAsmLoadPrimarySecondaryStack(&g.asm, g.locals[localIndex].offset, g.locals[localIndex].offset-renvoBackendValueSlotSize)
		return true
	}
	if (e.kind == renvoExprInt || e.kind == renvoExprFloat) && renvoExprTokenIsImaginary(g.prog, e.tok) {
		renvoAsmPrimaryImm(&g.asm, 0)
		renvoAsmSecondaryImm(&g.asm, renvoParseImaginaryTokenScaled(g.prog, e.tok))
		return true
	}
	if e.kind == renvoExprCall && renvoExprIdentCode(g.prog, ep, e.left) == renvoIdentComplex {
		if e.argCount != 2 {
			return false
		}
		realOffset := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		if !renvoEmitScalarExprForKind(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg), renvoTypeFloat64) {
			return false
		}
		renvoAsmStorePrimaryStack(&g.asm, realOffset)
		if !renvoEmitScalarExprForKind(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg+1), renvoTypeFloat64) {
			return false
		}
		renvoAsmPushPrimary(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, realOffset)
		renvoAsmPopSecondary(&g.asm)
		return true
	}
	if e.kind == renvoExprBinary && (renvoTokCharIs(g.prog, e.tok, '+') || renvoTokCharIs(g.prog, e.tok, '-')) {
		leftReal := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		leftImag := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		rightReal := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		rightImag := renvoAddUnnamedLocal(g, renvoTypeFloat64)
		if !renvoEmitComplexValueRegs(g, ep, e.left) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, leftReal, leftImag)
		if !renvoEmitComplexValueRegs(g, ep, e.right) {
			return false
		}
		renvoAsmStorePrimarySecondaryStack(&g.asm, rightReal, rightImag)
		renvoEmitComplexBinaryComponent(g, leftReal, rightReal, renvoTokCharIs(g.prog, e.tok, '-'))
		renvoAsmStorePrimaryStack(&g.asm, leftReal)
		renvoEmitComplexBinaryComponent(g, leftImag, rightImag, renvoTokCharIs(g.prog, e.tok, '-'))
		renvoAsmPushPrimary(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, leftReal)
		renvoAsmPopSecondary(&g.asm)
		return true
	}
	if renvoTypeKindIsScalarValue(renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx)).kind) {
		if !renvoEmitScalarExprForKind(g, ep, idx, renvoTypeFloat64) {
			return false
		}
		renvoAsmSecondaryImm(&g.asm, 0)
		return true
	}
	return false
}

func renvoEmitComplexBinaryComponent(g *renvoLinearGen, left int, right int, subtract bool) {
	renvoNonNil(g)
	renvoAsmLoadPrimaryTertiaryStack(&g.asm, left, right)
	if subtract {
		renvoAsmSubPrimaryTertiary(&g.asm)
	} else {
		renvoAsmAddPrimaryTertiary(&g.asm)
	}
}

func renvoParseImaginaryTokenScaled(p *renvoProgram, tok int) int {
	renvoNonNil(p)
	start := int(renvoTokStart(p, tok))
	end := int(renvoTokEnd(p, tok))
	whole := 0
	fraction := 0
	divisor := 1
	afterDot := false
	for i := start; i < end; i++ {
		c := renvo_runtime_UnsafeByteAt(p.src, i)
		if c == '.' {
			afterDot = true
			continue
		}
		if c < '0' || c > '9' {
			continue
		}
		if afterDot {
			fraction = fraction*10 + int(c-'0')
			divisor *= 10
		} else {
			whole = whole*10 + int(c-'0')
		}
	}
	return whole*4 + fraction*4/divisor
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
		renvoAsmPrimaryImm(&g.asm, fnIndex+1)
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
	renvoAsmPrimaryImm(&g.asm, fnIndex+1)
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
	renvoAsmPrimaryImm(&g.asm, fnIndex+1)
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
	source := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, idx))
	renvoNonNil(source)
	if !renvoEmitIntExpr(g, ep, idx) {
		return false
	}
	renvoAsmNormalizePrimaryForKind(&g.asm, source.kind)
	if destKind == renvoTypeFloat64 && source.kind != renvoTypeFloat64 {
		renvoAsmShlPrimaryImm(&g.asm, 2)
	} else if destKind != renvoTypeFloat64 && source.kind == renvoTypeFloat64 {
		renvoAsmCopyPrimaryToTertiary(&g.asm)
		renvoAsmPrimaryImm(&g.asm, 4)
		renvoAsmDivLeftTertiaryRightPrimary(&g.asm, false)
	}
	renvoAsmNormalizePrimaryForKind(&g.asm, destKind)
	return true
}

func renvoArrayBuiltinCount(g *renvoLinearGen, ep *renvoExprParse, e *renvoExpr) int {
	renvoNonNil(g, ep, e)
	t := renvoResolveType(g.meta, renvoInferParsedExprType(g, ep, renvo_runtime_UnsafeIntAt(ep.args, e.firstArg)))
	renvoNonNil(t)
	if t.kind == renvoTypeArray {
		return t.count
	}
	return -1
}

func renvoEmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EmitFloatBinaryExpr(g, ep, idx)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EmitFloatBinaryExpr(g, ep, idx)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEmitFloatBinaryExpr(g, ep, idx)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitFloatBinaryExpr(g, ep, idx)
	}
	return renvoAmd64EmitFloatBinaryExpr(g, ep, idx)
}
func renvoEmitSliceSlotAddrs(g *renvoLinearGen, locEp *renvoExprParse, loc *renvoSliceLocation, elemSize int) bool {
	renvoNonNil(g, locEp, loc)
	if renvoTargetArch == renvoArchArm {
		return renvoArmEmitSliceSlotAddrs(g, locEp, loc, elemSize)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitSliceSlotAddrs(g, locEp, loc, elemSize)
	}
	return renvoAmd64EmitSliceSlotAddrs(g, locEp, loc, elemSize)
}
func renvoEnsureAppendAddrHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EnsureAppendAddrHelper(g)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EnsureAppendAddrHelper(g)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEnsureAppendAddrHelper(g)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EnsureAppendAddrHelper(g)
	}
	return renvoAmd64EnsureAppendAddrHelper(g)
}
func renvoEnsureAppend8Helper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EnsureAppend8Helper(g)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EnsureAppend8Helper(g)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEnsureAppend8Helper(g)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EnsureAppend8Helper(g)
	}
	return renvoAmd64EnsureAppend8Helper(g)
}
func renvoEnsureAppend64Helper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EnsureAppend64Helper(g)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EnsureAppend64Helper(g)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEnsureAppend64Helper(g)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EnsureAppend64Helper(g)
	}
	return renvoAmd64EnsureAppend64Helper(g)
}
func renvoEnsureStringEqualHelper(g *renvoLinearGen) int {
	renvoNonNil(g)
	if renvoTargetArch == renvoArchWasm32 {
		return renvoWasm32EnsureStringEqualHelper(g)
	}
	if renvoTargetArch == renvoArchAarch64 {
		return renvoAarch64EnsureStringEqualHelper(g)
	}
	if renvoTargetArch == renvoArchArm {
		return renvoArmEnsureStringEqualHelper(g)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EnsureStringEqualHelper(g)
	}
	return renvoAmd64EnsureStringEqualHelper(g)
}
func renvoEmitIndexedStructField(g *renvoLinearGen, ep *renvoExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitIndexedStructField(g, ep, indexIdx, fieldStart, fieldEnd)
	}
	return renvoAmd64EmitIndexedStructField(g, ep, indexIdx, fieldStart, fieldEnd)
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

func renvoEmitSelectorAddressSecondary(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	if renvoTargetArch == renvoArchAmd64 {
		return renvoAmd64EmitSelectorAddressRdx(g, ep, idx)
	}
	if renvoTargetArch == renvoArch386 {
		return renvo386EmitSelectorAddressRdx(g, ep, idx)
	}
	return renvoAmd64EmitSelectorAddressRdx(g, ep, idx)
}

func renvoEmitIndexedSelectorAddressSecondary(g *renvoLinearGen, ep *renvoExprParse, idx int, fieldOffset int) bool {
	renvoNonNil(g, ep)
	meta := g.meta
	a := &g.asm
	indexExpr := &ep.exprs[idx]
	leftType := renvoInferParsedExprType(g, ep, indexExpr.left)
	sliceType := renvoResolveType(meta, leftType)
	renvoNonNil(sliceType)
	if sliceType.kind != renvoTypeSlice {
		return false
	}
	elemType := renvoResolveType(meta, sliceType.elem)
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
