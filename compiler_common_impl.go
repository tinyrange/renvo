package main

const rtgAbsBssReloc = 1
const rtgAbsWinImportReloc = 2

const rtgTargetLinuxAmd64 = 1
const rtgTargetLinux386 = 2
const rtgTargetLinuxAarch64 = 3
const rtgTargetLinuxArm = 4
const rtgTargetWindowsAmd64 = 5
const rtgTargetWindows386 = 6
const rtgTargetWasiWasm32 = 7
const rtgTargetDarwinArm64 = 8

const rtgArchAmd64 = 1
const rtgArch386 = 2
const rtgArchAarch64 = 3
const rtgArchArm = 4
const rtgArchWasm32 = 5

const rtgOSLinux = 1
const rtgOSWindows = 2
const rtgOSDarwin = 3
const rtgOSWasi = 4

const rtgEndianLittle = 1
const rtgEndianBig = 2

const rtgAddressModelFlat = 1
const rtgAddressModelHarvard = 2
const rtgAddressModelSegmented = 3
const rtgAddressModelBanked = 4

const rtgPointerSpaceData = 1
const rtgPointerSpaceCode = 2
const rtgPointerSpaceFunction = 3
const rtgPointerSpaceGeneric = 4

const rtgRuntimePrint = 1
const rtgRuntimeOpen = 2
const rtgRuntimeClose = 4
const rtgRuntimeRead = 8
const rtgRuntimeWrite = 16
const rtgRuntimeChmod = 32
const rtgRuntimeHosted = 64
const rtgRuntimeHeap = 128
const rtgRuntimeVolatileMemory = 256
const rtgRuntimeInterrupts = 512

const rtgHeapNone = 0
const rtgHeapBump = 1
const rtgHeapExternal = 2

const rtgOOMTrap = 1
const rtgOOMResult = 2
const rtgOOMPanic = 3

const rtgVolatileWidth8 = 1
const rtgVolatileWidth16 = 2
const rtgVolatileWidth32 = 4
const rtgVolatileWidth64 = 8

const rtgInterruptNone = 0
const rtgInterruptVector = 1

const rtgFloatScaledInteger = 1
const rtgFloatIEEEHardware = 2
const rtgFloatIEEESoft = 3

// The current normalized backend stores scalar values in eight-byte virtual
// slots even when the target address or language int is narrower. Keep this
// distinct from the target data model so future C and small-device backends do
// not mistake an internal lowering detail for a machine ABI requirement.
const rtgBackendValueSlotSize = 8
const rtgBackendStringWordCount = 2
const rtgBackendSliceWordCount = 3
const rtgBackendHiddenResultWordCount = 1
const rtgBackendRegisterCallWordCount = 6
const rtgBackendStringValueSize = rtgBackendValueSlotSize * rtgBackendStringWordCount
const rtgBackendSliceValueSize = rtgBackendValueSlotSize * rtgBackendSliceWordCount

type rtgTargetProfile struct {
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

func rtgProfileForTarget(target int) (rtgTargetProfile, bool) {
	var p rtgTargetProfile
	p.target = target
	p.charBits = 8
	p.endian = rtgEndianLittle
	p.backendSlotSize = rtgBackendValueSlotSize
	p.addressModel = rtgAddressModelFlat
	p.runtimeCaps = rtgRuntimePrint | rtgRuntimeOpen | rtgRuntimeClose | rtgRuntimeRead | rtgRuntimeWrite | rtgRuntimeChmod | rtgRuntimeHosted
	p.heapModel = rtgHeapNone
	p.oomModel = rtgOOMResult
	p.interruptModel = rtgInterruptNone
	p.floatModel = rtgFloatScaledInteger
	if target == rtgTargetLinux386 || target == rtgTargetWindows386 {
		p.arch = rtgArch386
		p.intBits = 32
		p.pointerBits = 32
		p.maxAlign = 4
	} else if target == rtgTargetLinuxArm {
		p.arch = rtgArchArm
		p.intBits = 32
		p.pointerBits = 32
		p.maxAlign = 4
	} else if target == rtgTargetWasiWasm32 {
		p.arch = rtgArchWasm32
		p.intBits = 32
		p.pointerBits = 32
		p.maxAlign = 8
	} else if target == rtgTargetLinuxAarch64 || target == rtgTargetDarwinArm64 {
		p.arch = rtgArchAarch64
		p.intBits = 64
		p.pointerBits = 64
		p.maxAlign = 8
	} else if target == rtgTargetLinuxAmd64 || target == rtgTargetWindowsAmd64 {
		p.arch = rtgArchAmd64
		p.intBits = 64
		p.pointerBits = 64
		p.maxAlign = 8
	} else {
		return p, false
	}
	p.os = rtgOSLinux
	if target == rtgTargetWindowsAmd64 || target == rtgTargetWindows386 {
		p.os = rtgOSWindows
	}
	if target == rtgTargetDarwinArm64 {
		p.os = rtgOSDarwin
	}
	if target == rtgTargetWasiWasm32 {
		p.os = rtgOSWasi
	}
	p.codePointerBits = p.pointerBits
	p.funcPointerBits = p.pointerBits
	return p, true
}

func rtgProfileHasRuntime(p rtgTargetProfile, capability int) bool {
	return p.runtimeCaps&capability == capability
}

func rtgProfileIsValid(p rtgTargetProfile) bool {
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
	if p.endian != rtgEndianLittle && p.endian != rtgEndianBig {
		return false
	}
	if p.backendSlotSize < 1 || p.maxAlign < 1 {
		return false
	}
	if p.addressModel < rtgAddressModelFlat || p.addressModel > rtgAddressModelBanked {
		return false
	}
	if p.heapModel < rtgHeapNone || p.heapModel > rtgHeapExternal {
		return false
	}
	if p.oomModel < rtgOOMTrap || p.oomModel > rtgOOMPanic {
		return false
	}
	if rtgProfileHasRuntime(p, rtgRuntimeHeap) && p.heapModel == rtgHeapNone {
		return false
	}
	if rtgProfileHasRuntime(p, rtgRuntimeVolatileMemory) && p.volatileWidths == 0 {
		return false
	}
	if rtgProfileHasRuntime(p, rtgRuntimeInterrupts) && p.interruptModel == rtgInterruptNone {
		return false
	}
	if p.floatModel < rtgFloatScaledInteger || p.floatModel > rtgFloatIEEESoft {
		return false
	}
	return true
}

var rtgTargetArch int = rtgArchAmd64
var rtgTargetOS int = rtgOSLinux
var rtgNativeIntSize int = 8
var rtgCurrentTarget int = rtgTargetLinuxAmd64
var rtgCompilerArenaSize int

// These bodies are used by the host Go build. Self-hosted compilers lower the
// calls as arena intrinsics so large, phase-local scratch data can be reclaimed.
func rtg_runtime_ArenaMark() int { return 0 }

func rtg_runtime_ArenaReset(mark int) {}

func rtgSetTarget(target int) {
	if rtgCompilerFixedTarget != 0 {
		rtgCurrentTarget = rtgCompilerFixedTarget
		if rtgCompilerFixedTarget == rtgTargetWindows386 {
			rtgTargetOS = rtgOSWindows
			rtgTargetArch = rtgArch386
			rtgNativeIntSize = 4
			return
		}
		if rtgCompilerFixedTarget == rtgTargetWindowsAmd64 {
			rtgTargetOS = rtgOSWindows
			rtgTargetArch = rtgArchAmd64
			rtgNativeIntSize = 8
			return
		}
		if rtgCompilerFixedTarget == rtgTargetDarwinArm64 {
			rtgTargetOS = rtgOSDarwin
			rtgTargetArch = rtgArchAarch64
			rtgNativeIntSize = 8
			return
		}
		if rtgCompilerFixedTarget == rtgTargetWasiWasm32 {
			rtgTargetOS = rtgOSWasi
			rtgTargetArch = rtgArchWasm32
			rtgNativeIntSize = 4
			return
		}
		if rtgCompilerFixedTarget == rtgTargetLinux386 {
			rtgTargetOS = rtgOSLinux
			rtgTargetArch = rtgArch386
			rtgNativeIntSize = 4
			return
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxArm {
			rtgTargetOS = rtgOSLinux
			rtgTargetArch = rtgArchArm
			rtgNativeIntSize = 4
			return
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxAarch64 {
			rtgTargetOS = rtgOSLinux
			rtgTargetArch = rtgArchAarch64
			rtgNativeIntSize = 8
			return
		}
		rtgTargetOS = rtgOSLinux
		rtgTargetArch = rtgArchAmd64
		rtgNativeIntSize = 8
		return
	}
	rtgCurrentTarget = target
	rtgTargetOS = rtgOSLinux
	if target == rtgTargetLinux386 {
		rtgTargetArch = rtgArch386
		rtgNativeIntSize = 4
		return
	}
	if target == rtgTargetWindows386 {
		rtgTargetOS = rtgOSWindows
		rtgTargetArch = rtgArch386
		rtgNativeIntSize = 4
		return
	}
	if target == rtgTargetLinuxAarch64 {
		rtgTargetArch = rtgArchAarch64
		rtgNativeIntSize = 8
		return
	}
	if target == rtgTargetDarwinArm64 {
		rtgTargetOS = rtgOSDarwin
		rtgTargetArch = rtgArchAarch64
		rtgNativeIntSize = 8
		return
	}
	if target == rtgTargetLinuxArm {
		rtgTargetArch = rtgArchArm
		rtgNativeIntSize = 4
		return
	}
	if target == rtgTargetWindowsAmd64 {
		rtgTargetOS = rtgOSWindows
		rtgTargetArch = rtgArchAmd64
		rtgNativeIntSize = 8
		return
	}
	if target == rtgTargetWasiWasm32 {
		rtgTargetOS = rtgOSWasi
		rtgTargetArch = rtgArchWasm32
		rtgNativeIntSize = 4
		return
	}
	rtgTargetArch = rtgArchAmd64
	rtgNativeIntSize = 8
}

func rtgTargetIsWindows() bool {
	return rtgTargetOS == rtgOSWindows
}

func rtgTargetIsDarwin() bool {
	return rtgTargetOS == rtgOSDarwin
}

type rtgLabelRef struct {
	at    int
	label int
}

type rtgAbsRef struct {
	at   int
	off  int
	kind int
}

type rtgAsmSymbol struct {
	nameStart int
	nameEnd   int
	label     int
}

type rtgWinStaticImport struct {
	dll  string
	name string
}

type rtgAsm struct {
	code               []byte
	labelPos           []int
	labelSet           []bool
	relocs             []rtgLabelRef
	absRelocs          []rtgAbsRef
	symbols            []rtgAsmSymbol
	symbolName         []byte
	winImports         []rtgWinStaticImport
	darwinImportLabels []int
	darwinImportUsed   []bool
	data               []byte
	bssSize            int
	codeOffset         int
	dataOffset         int
}

const rtgWasm32FallbackSliceBackingSize = 4096

func rtgAsmInit(a *rtgAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgLabelRef
	var absRelocs []rtgAbsRef
	var symbols []rtgAsmSymbol
	var symbolName []byte
	var winImports []rtgWinStaticImport
	var data []byte
	if rtgCompilerFixedTarget != 0 {
		code = make([]byte, 0, 2097152)
		labelPos = make([]int, 0, 16384)
		labelSet = make([]bool, 0, 16384)
		relocs = make([]rtgLabelRef, 0, 32768)
		absRelocs = make([]rtgAbsRef, 0, 16384)
		symbols = make([]rtgAsmSymbol, 0, 1024)
		if rtgCompilerStripSymbols && rtgTargetArch != rtgArchWasm32 {
			symbols = make([]rtgAsmSymbol, 0, 0)
		}
	} else if rtgTargetArch == rtgArchWasm32 {
		code = make([]byte, 0, 655360)
		labelPos = make([]int, 0, 32768)
		labelSet = make([]bool, 0, 32768)
		relocs = make([]rtgLabelRef, 0, 65536)
		absRelocs = make([]rtgAbsRef, 0, 32768)
		symbols = make([]rtgAsmSymbol, 0, 2048)
	} else {
		code = make([]byte, 0, 2097152)
		labelPos = make([]int, 0, 32768)
		labelSet = make([]bool, 0, 32768)
		relocs = make([]rtgLabelRef, 0, 65536)
		absRelocs = make([]rtgAbsRef, 0, 32768)
		symbols = make([]rtgAsmSymbol, 0, 4096)
	}
	data = make([]byte, 0, 16384)
	if rtgCompilerStripSymbols && rtgTargetArch != rtgArchWasm32 {
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
	a.data = data
	a.bssSize = 0
	a.codeOffset = 0
	a.dataOffset = 0
}

func rtgAsmNewLabel(a *rtgAsm) int {
	label := len(a.labelPos)
	a.labelPos = a.labelPos[:label+1]
	a.labelSet = a.labelSet[:label+1]
	a.labelPos[label] = 0
	a.labelSet[label] = false
	return label
}

func rtgAsmMarkLabel(a *rtgAsm, label int) {
	if label < 0 {
		return
	}
	if label >= len(a.labelPos) || label >= len(a.labelSet) {
		return
	}
	codeLen := len(a.code)
	a.labelPos[label] = codeLen
	a.labelSet[label] = true
}

func rtgAsmEmit8(a *rtgAsm, v int) {
	index := len(a.code)
	a.code = a.code[:index+1]
	a.code[index] = byte(v)
}

func rtgAsmEmit2(a *rtgAsm, v0 int, v1 int) {
	index := len(a.code)
	a.code = a.code[:index+2]
	a.code[index] = byte(v0)
	a.code[index+1] = byte(v1)
}

func rtgAsmEmit3(a *rtgAsm, v0 int, v1 int, v2 int) {
	index := len(a.code)
	a.code = a.code[:index+3]
	a.code[index] = byte(v0)
	a.code[index+1] = byte(v1)
	a.code[index+2] = byte(v2)
}

func rtgAsmEmit4(a *rtgAsm, v0 int, v1 int, v2 int, v3 int) {
	index := len(a.code)
	a.code = a.code[:index+4]
	a.code[index] = byte(v0)
	a.code[index+1] = byte(v1)
	a.code[index+2] = byte(v2)
	a.code[index+3] = byte(v3)
}

func rtgAsmEmit5(a *rtgAsm, v0 int, v1 int, v2 int, v3 int, v4 int) {
	index := len(a.code)
	a.code = a.code[:index+5]
	a.code[index] = byte(v0)
	a.code[index+1] = byte(v1)
	a.code[index+2] = byte(v2)
	a.code[index+3] = byte(v3)
	a.code[index+4] = byte(v4)
}

func rtgAsmAddAbsReloc(a *rtgAsm, at int, off int, kind int) {
	index := len(a.absRelocs)
	a.absRelocs = a.absRelocs[:index+1]
	a.absRelocs[index].at = at & 2147483647
	a.absRelocs[index].off = off & 2147483647
	a.absRelocs[index].kind = kind & 2147483647
}

func rtgAsmAddReloc(a *rtgAsm, at int, label int) {
	index := len(a.relocs)
	a.relocs = a.relocs[:index+1]
	a.relocs[index].at = at & 2147483647
	a.relocs[index].label = label & 2147483647
}

func rtgAsmAddFuncSymbol(a *rtgAsm, src []byte, nameStart int, nameEnd int, label int) {
	if rtgCompilerStripSymbols && rtgTargetArch != rtgArchWasm32 {
		return
	}
	start := len(a.symbolName)
	for i := nameStart; i < nameEnd; i++ {
		a.symbolName = append(a.symbolName, src[i])
	}
	end := len(a.symbolName)
	var sym rtgAsmSymbol
	sym.nameStart = start
	sym.nameEnd = end
	sym.label = label
	a.symbols = append(a.symbols, sym)
}

func rtgAsmEmit32(a *rtgAsm, v int) {
	a.code = rtgAppend32(a.code, v)
}

func rtgFixedByteScratch(capacity int) []byte {
	if rtgCompilerFixedTarget != 0 {
		return make([]byte, 0, capacity)
	}
	var out []byte
	return out
}

func rtgFixedIntScratch(capacity int) []int {
	if rtgCompilerFixedTarget != 0 {
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

func rtgFixedCompositeFieldScratch(capacity int) []rtgCompositeField {
	if rtgCompilerFixedTarget != 0 {
		if capacity <= 8 {
			capacity = 8
		}
		return make([]rtgCompositeField, 0, capacity)
	}
	var out []rtgCompositeField
	return out
}

func rtgAsmEmit64(a *rtgAsm, v int) {
	a.code = rtgAppend64(a.code, v)
}

func rtgAsmEmit16(a *rtgAsm, v int) {
	rtgAsmEmit8(a, v)
	rtgAsmEmit8(a, v>>8)
}

func rtgAsmEmit24(a *rtgAsm, v int) {
	rtgAsmEmit8(a, v)
	rtgAsmEmit8(a, v>>8)
	rtgAsmEmit8(a, v>>16)
}

func rtgAmd64RelaxBranches(a *rtgAsm) {
	oldCode := a.code
	oldLen := len(oldCode)
	shortOp := make([]int, oldLen, oldLen)
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
		if at >= 1 && oldCode[at-1] == 0xe9 {
			shortOp[at-1] = 0xeb
			continue
		}
		if at >= 2 && oldCode[at-2] == 0x0f && oldCode[at-1] >= 0x80 && oldCode[at-1] <= 0x8f {
			shortOp[at-2] = int(oldCode[at-1]) - 0x10
		}
	}
	positions := make([]int, oldLen+1, oldLen+1)
	next := make([]byte, 0, cap(oldCode))
	for i := 0; i < oldLen; {
		positions[i] = len(next)
		op := shortOp[i]
		if op != 0 {
			n := 5
			if oldCode[i] == 0x0f {
				n = 6
			}
			next = append(next, byte(op))
			next = append(next, 0)
			for j := 1; j < n; j++ {
				positions[i+j] = len(next) - 1
			}
			i += n
			continue
		}
		next = append(next, oldCode[i])
		i++
	}
	positions[oldLen] = len(next)
	var relocs []rtgLabelRef
	relocs = make([]rtgLabelRef, 0, len(a.relocs))
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		at := r.at & 2147483647
		label := r.label & 2147483647
		start := at - 1
		if at >= 2 && oldCode[at-2] == 0x0f {
			start = at - 2
		}
		if start >= 0 && start < oldLen && shortOp[start] != 0 && label >= 0 && label < len(a.labelPos) && label < len(a.labelSet) && a.labelSet[label] {
			newAt := positions[start] + 1
			target := positions[a.labelPos[label]]
			disp := target - (newAt + 1)
			if disp >= -128 && disp <= 127 {
				next[newAt] = byte(disp)
				continue
			}
		}
		r.at = positions[at]
		relocs = append(relocs, r)
	}
	for i := 0; i < len(a.absRelocs); i++ {
		at := a.absRelocs[i].at & 2147483647
		a.absRelocs[i].at = positions[at]
	}
	for i := 0; i < len(a.labelPos); i++ {
		if a.labelSet[i] {
			a.labelPos[i] = positions[a.labelPos[i]]
		}
	}
	a.code = next
	a.relocs = relocs
}

func rtgAsmPatch(a *rtgAsm) {
	if rtgTargetArch == rtgArchArm {
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
			insn := rtgGet32At(a.code, at)
			if (insn & 0x0e000000) == 0x0a000000 {
				rtgPut32At(a.code, at, (insn&0xff000000)|((disp/4)&0x00ffffff))
			}
		}
		a.dataOffset = a.codeOffset + len(a.code)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
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
			insn := rtgGet32At(a.code, at)
			if (insn & 0xfc000000) == 0x94000000 {
				rtgPut32At(a.code, at, 0x94000000|((disp/4)&0x03ffffff))
			} else if (insn & 0xfc000000) == 0x14000000 {
				rtgPut32At(a.code, at, 0x14000000|((disp/4)&0x03ffffff))
			} else if (insn & 0xff000010) == 0x54000000 {
				rtgPut32At(a.code, at, (insn&0xff00001f)|(((disp/4)&0x7ffff)<<5))
			}
		}
		a.dataOffset = a.codeOffset + len(a.code)
		return
	}
	if rtgTargetArch == rtgArchAmd64 && rtgCompilerFixedTarget == 0 {
		rtgAmd64RelaxBranches(a)
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
		rtgPut32At(a.code, at, disp)
	}
	a.dataOffset = a.codeOffset + len(a.code)
	for i := 0; i < len(a.absRelocs); i++ {
		at := a.absRelocs[i].at & 2147483647
		off := a.absRelocs[i].off & 2147483647
		kind := a.absRelocs[i].kind & 2147483647
		target := a.dataOffset + off
		if kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + off
		}
		next := a.codeOffset + at + 4
		disp := target - next
		rtgPut32At(a.code, at, disp)
	}
}

func rtgGet32At(in []byte, at int) int {
	return int(in[at]) | (int(in[at+1]) << 8) | (int(in[at+2]) << 16) | (int(in[at+3]) << 24)
}

func rtgPut32At(out []byte, at int, v int) {
	b0 := byte(v)
	b1 := byte(v >> 8)
	b2 := byte(v >> 16)
	b3 := byte(v >> 24)
	out[at] = b0
	out[at+1] = b1
	out[at+2] = b2
	out[at+3] = b3
}

func rtgAppend16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func rtgAppend32(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	return out
}

func rtgAppend64(out []byte, v int) []byte {
	out = rtgAppend32(out, v)
	out = rtgAppend32(out, v>>32)
	return out
}

func rtgAppend64U32(out []byte, v int) []byte {
	out = rtgAppend32(out, v)
	out = rtgAppend32(out, 0)
	return out
}

type rtgElf64SymbolSections struct {
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

func rtgAlignValue(v int, align int) int {
	rem := v % align
	if rem == 0 {
		return v
	}
	return v + align - rem
}

func rtgAppendUntil(out []byte, size int) []byte {
	for len(out) < size {
		out = append(out, 0)
	}
	return out
}

func rtgAppendStringZ(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return out
}

func rtgAppendBytesZ(out []byte, s []byte) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return out
}

func rtgAppendBytesRangeZ(out []byte, s []byte, start int, end int) []byte {
	for i := start; i < end; i++ {
		out = append(out, s[i])
	}
	out = append(out, 0)
	return out
}

func rtgAppendElf64Sym(out []byte, name int, info int, shndx int, value int, size int) []byte {
	out = rtgAppend32(out, name)
	out = append(out, byte(info))
	out = append(out, 0)
	out = rtgAppend16(out, shndx)
	out = rtgAppend64U32(out, value)
	out = rtgAppend64U32(out, size)
	return out
}

func rtgAppendElf64Shdr(out []byte, name int, typ int, flags int, addr int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = rtgAppend32(out, name)
	out = rtgAppend32(out, typ)
	out = rtgAppend64U32(out, flags)
	out = rtgAppend64U32(out, addr)
	out = rtgAppend64U32(out, off)
	out = rtgAppend64U32(out, size)
	out = rtgAppend32(out, link)
	out = rtgAppend32(out, info)
	out = rtgAppend64U32(out, align)
	out = rtgAppend64U32(out, entsize)
	return out
}

func rtgBuildElf64SymbolSections(a *rtgAsm, base int, entryOff int, loadFileSize int) rtgElf64SymbolSections {
	var sec rtgElf64SymbolSections
	sec.symtab = make([]byte, 0, (len(a.symbols)+2)*24)
	sec.strtab = make([]byte, 0, len(a.symbolName)+16)
	sec.shstrtab = make([]byte, 0, 64)
	sec.shstrtab = append(sec.shstrtab, 0)
	sec.textName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".text")
	sec.dataName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".data")
	sec.bssName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".bss")
	sec.symtabName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".symtab")
	sec.strtabName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".strtab")
	sec.shstrName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".shstrtab")

	sec.strtab = append(sec.strtab, 0)
	startName := len(sec.strtab)
	sec.strtab = rtgAppendStringZ(sec.strtab, "_start")
	sec.symtab = rtgAppendElf64Sym(sec.symtab, 0, 0, 0, 0, 0)
	sec.symtab = rtgAppendElf64Sym(sec.symtab, startName, 18, 1, base+entryOff, 0)
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
		sec.strtab = rtgAppendBytesRangeZ(sec.strtab, a.symbolName, s.nameStart, s.nameEnd)
		value := base + a.codeOffset + a.labelPos[label]
		sec.symtab = rtgAppendElf64Sym(sec.symtab, nameOff, 18, 1, value, 0)
	}

	sec.symtabOff = rtgAlignValue(loadFileSize, 8)
	sec.strtabOff = sec.symtabOff + len(sec.symtab)
	sec.shstrOff = sec.strtabOff + len(sec.strtab)
	sec.shoff = rtgAlignValue(sec.shstrOff+len(sec.shstrtab), 8)
	return sec
}

func rtgAppendElf64SectionHeaders(out []byte, sec *rtgElf64SymbolSections, a *rtgAsm, base int) []byte {
	codeOff := a.codeOffset
	codeSize := len(a.code)
	dataOff := a.dataOffset
	dataSize := len(a.data)
	bssSize := a.bssSize
	symtabSize := len(sec.symtab)
	strtabSize := len(sec.strtab)
	shstrtabSize := len(sec.shstrtab)

	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)

	out = rtgAppend32(out, sec.textName)
	out = rtgAppend32(out, 1)
	out = rtgAppend64U32(out, 6)
	out = rtgAppend64U32(out, base+codeOff)
	out = rtgAppend64U32(out, codeOff)
	out = rtgAppend64U32(out, codeSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 16)
	out = rtgAppend64U32(out, 0)

	out = rtgAppend32(out, sec.dataName)
	out = rtgAppend32(out, 1)
	out = rtgAppend64U32(out, 3)
	out = rtgAppend64U32(out, base+dataOff)
	out = rtgAppend64U32(out, dataOff)
	out = rtgAppend64U32(out, dataSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 8)
	out = rtgAppend64U32(out, 0)

	out = rtgAppend32(out, sec.bssName)
	out = rtgAppend32(out, 8)
	out = rtgAppend64U32(out, 3)
	out = rtgAppend64U32(out, base+dataOff+dataSize)
	out = rtgAppend64U32(out, dataOff+dataSize)
	out = rtgAppend64U32(out, bssSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 8)
	out = rtgAppend64U32(out, 0)

	out = rtgAppend32(out, sec.symtabName)
	out = rtgAppend32(out, 2)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, sec.symtabOff)
	out = rtgAppend64U32(out, symtabSize)
	out = rtgAppend32(out, 5)
	out = rtgAppend32(out, 1)
	out = rtgAppend64U32(out, 8)
	out = rtgAppend64U32(out, 24)

	out = rtgAppend32(out, sec.strtabName)
	out = rtgAppend32(out, 3)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, sec.strtabOff)
	out = rtgAppend64U32(out, strtabSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 1)
	out = rtgAppend64U32(out, 0)

	out = rtgAppend32(out, sec.shstrName)
	out = rtgAppend32(out, 3)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend64U32(out, sec.shstrOff)
	out = rtgAppend64U32(out, shstrtabSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend64U32(out, 1)
	out = rtgAppend64U32(out, 0)
	return out
}

type rtgElf32SymbolSections struct {
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

func rtgAppendElf32Sym(out []byte, name int, info int, shndx int, value int, size int) []byte {
	out = rtgAppend32(out, name)
	out = rtgAppend32(out, value)
	out = rtgAppend32(out, size)
	out = append(out, byte(info))
	out = append(out, 0)
	out = rtgAppend16(out, shndx)
	return out
}

func rtgAppendElf32Shdr(out []byte, name int, typ int, flags int, addr int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = rtgAppend32(out, name)
	out = rtgAppend32(out, typ)
	out = rtgAppend32(out, flags)
	out = rtgAppend32(out, addr)
	out = rtgAppend32(out, off)
	out = rtgAppend32(out, size)
	out = rtgAppend32(out, link)
	out = rtgAppend32(out, info)
	out = rtgAppend32(out, align)
	out = rtgAppend32(out, entsize)
	return out
}

func rtgBuildElf32SymbolSections(a *rtgAsm, base int, entryOff int, loadFileSize int) rtgElf32SymbolSections {
	var sec rtgElf32SymbolSections
	sec.symtab = make([]byte, 0, (len(a.symbols)+2)*16)
	sec.strtab = make([]byte, 0, len(a.symbolName)+16)
	sec.shstrtab = make([]byte, 0, 64)
	sec.shstrtab = append(sec.shstrtab, 0)
	sec.textName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".text")
	sec.dataName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".data")
	sec.bssName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".bss")
	sec.symtabName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".symtab")
	sec.strtabName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".strtab")
	sec.shstrName = len(sec.shstrtab)
	sec.shstrtab = rtgAppendStringZ(sec.shstrtab, ".shstrtab")

	sec.strtab = append(sec.strtab, 0)
	startName := len(sec.strtab)
	sec.strtab = rtgAppendStringZ(sec.strtab, "_start")
	sec.symtab = rtgAppendElf32Sym(sec.symtab, 0, 0, 0, 0, 0)
	sec.symtab = rtgAppendElf32Sym(sec.symtab, startName, 18, 1, base+entryOff, 0)
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
		sec.strtab = rtgAppendBytesRangeZ(sec.strtab, a.symbolName, s.nameStart, s.nameEnd)
		value := base + a.codeOffset + a.labelPos[label]
		sec.symtab = rtgAppendElf32Sym(sec.symtab, nameOff, 18, 1, value, 0)
	}

	sec.symtabOff = rtgAlignValue(loadFileSize, 4)
	sec.strtabOff = sec.symtabOff + len(sec.symtab)
	sec.shstrOff = sec.strtabOff + len(sec.strtab)
	sec.shoff = rtgAlignValue(sec.shstrOff+len(sec.shstrtab), 4)
	return sec
}

func rtgAppendElf32SectionHeaders(out []byte, sec *rtgElf32SymbolSections, a *rtgAsm, base int) []byte {
	codeOff := a.codeOffset
	codeSize := len(a.code)
	dataOff := a.dataOffset
	dataSize := len(a.data)
	bssSize := a.bssSize
	symtabSize := len(sec.symtab)
	strtabSize := len(sec.strtab)
	shstrtabSize := len(sec.shstrtab)

	out = rtgAppendElf32Shdr(out, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	out = rtgAppendElf32Shdr(out, sec.textName, 1, 6, base+codeOff, codeOff, codeSize, 0, 0, 16, 0)
	out = rtgAppendElf32Shdr(out, sec.dataName, 1, 3, base+dataOff, dataOff, dataSize, 0, 0, 4, 0)
	out = rtgAppendElf32Shdr(out, sec.bssName, 8, 3, base+dataOff+dataSize, dataOff+dataSize, bssSize, 0, 0, 4, 0)
	out = rtgAppendElf32Shdr(out, sec.symtabName, 2, 0, 0, sec.symtabOff, symtabSize, 5, 1, 4, 16)
	out = rtgAppendElf32Shdr(out, sec.strtabName, 3, 0, 0, sec.strtabOff, strtabSize, 0, 0, 1, 0)
	out = rtgAppendElf32Shdr(out, sec.shstrName, 3, 0, 0, sec.shstrOff, shstrtabSize, 0, 0, 1, 0)
	return out
}

const rtgWinImageBase = 0x400000
const rtgWinSectionRVA = 0x1000
const rtgWinHeadersSize = 0x200
const rtgWinFileAlign = 0x200
const rtgWinSectionAlign = 0x1000

const rtgWinImportCreateFileA = 1
const rtgWinImportCloseHandle = 2
const rtgWinImportReadFile = 3
const rtgWinImportWriteFile = 4
const rtgWinImportSetFilePointer = 5
const rtgWinImportGetStdHandle = 6
const rtgWinImportGetCommandLineA = 7
const rtgWinImportExitProcess = 8
const rtgWinImportGetEnvironmentStringsA = 9
const rtgWinImportFixedCount = 9

type rtgWinImportLayout struct {
	importRVA    int
	importSize   int
	kernelIATRVA int
	thunkSize    int
	iatRVAs      []int
}

func rtgWinImportName(id int) string {
	if id == rtgWinImportCreateFileA {
		return "CreateFileA"
	}
	if id == rtgWinImportCloseHandle {
		return "CloseHandle"
	}
	if id == rtgWinImportReadFile {
		return "ReadFile"
	}
	if id == rtgWinImportWriteFile {
		return "WriteFile"
	}
	if id == rtgWinImportSetFilePointer {
		return "SetFilePointer"
	}
	if id == rtgWinImportGetStdHandle {
		return "GetStdHandle"
	}
	if id == rtgWinImportGetCommandLineA {
		return "GetCommandLineA"
	}
	if id == rtgWinImportGetEnvironmentStringsA {
		return "GetEnvironmentStringsA"
	}
	return "ExitProcess"
}

func rtgWinImportIATRVA(layout rtgWinImportLayout, id int) int {
	if id >= 0 && id < len(layout.iatRVAs) && layout.iatRVAs[id] != 0 {
		return layout.iatRVAs[id]
	}
	return layout.kernelIATRVA + (id-1)*layout.thunkSize
}

func rtgAsmAddWinImportReloc(a *rtgAsm, at int, importID int) {
	rtgAsmAddAbsReloc(a, at, importID, rtgAbsWinImportReloc)
}

func rtgAsmAddWinStaticImport(a *rtgAsm, dllStart int, dllEnd int, nameStart int, nameEnd int, src []byte) int {
	dll := rtgStringFromBytes(src, dllStart, dllEnd)
	name := rtgStringFromBytes(src, nameStart, nameEnd)
	for i := 0; i < len(a.winImports); i++ {
		imp := a.winImports[i]
		if imp.dll == dll && imp.name == name {
			return rtgWinImportFixedCount + 1 + i
		}
	}
	a.winImports = append(a.winImports, rtgWinStaticImport{dll: dll, name: name})
	return rtgWinImportFixedCount + len(a.winImports)
}

func rtgStringFromBytes(src []byte, start int, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	out := rtgFixedByteScratch(end - start)
	for i := start; i < end; i++ {
		out = append(out, src[i])
	}
	return string(out)
}

func rtgAsmHasWinImportRelocs(a *rtgAsm) bool {
	for i := 0; i < len(a.absRelocs); i++ {
		if a.absRelocs[i].kind == rtgAbsWinImportReloc {
			return true
		}
	}
	return false
}

func rtgAppendWinImports(a *rtgAsm, is64 bool) rtgWinImportLayout {
	var layout rtgWinImportLayout
	thunkSize := 4
	if is64 {
		thunkSize = 8
	}
	dataRVA := a.dataOffset
	if dataRVA == 0 {
		dataRVA = a.codeOffset + len(a.code)
	}
	layout.iatRVAs = make([]int, rtgWinImportFixedCount+len(a.winImports)+1)
	groupCount := len(a.winImports) + 1
	importOff := rtgAlignValue(len(a.data), thunkSize)
	a.data = rtgAppendUntil(a.data, importOff)
	descOff := importOff
	tableOff := descOff + (groupCount+1)*20
	groupILTOffs := make([]int, groupCount)
	groupIATOffs := make([]int, groupCount)
	for i := 0; i < groupCount; i++ {
		slots := 1
		if i == 0 {
			slots = rtgWinImportFixedCount
		}
		groupILTOffs[i] = tableOff
		tableOff += (slots + 1) * thunkSize
		groupIATOffs[i] = tableOff
		tableOff += (slots + 1) * thunkSize
	}
	a.data = rtgAppendUntil(a.data, tableOff)

	for id := 1; id <= rtgWinImportFixedCount; id++ {
		rtgAppendWinImportEntry(a, &layout, groupILTOffs[0], groupIATOffs[0], dataRVA, id, id-1, thunkSize, rtgWinImportName(id))
	}
	for i := 0; i < len(a.winImports); i++ {
		imp := a.winImports[i]
		rtgAppendWinImportEntry(a, &layout, groupILTOffs[i+1], groupIATOffs[i+1], dataRVA, rtgWinImportFixedCount+1+i, 0, thunkSize, imp.name)
	}
	for i := 0; i < groupCount; i++ {
		dllNameOff := len(a.data)
		dll := "kernel32.dll"
		if i > 0 {
			dll = a.winImports[i-1].dll
		}
		a.data = rtgAppendStringZ(a.data, dll)
		at := descOff + i*20
		rtgPut32At(a.data, at, dataRVA+groupILTOffs[i])
		rtgPut32At(a.data, at+12, dataRVA+dllNameOff)
		rtgPut32At(a.data, at+16, dataRVA+groupIATOffs[i])
	}

	layout.importRVA = dataRVA + importOff
	layout.importSize = len(a.data) - importOff
	layout.kernelIATRVA = layout.iatRVAs[rtgWinImportGetStdHandle]
	layout.thunkSize = thunkSize
	return layout
}

func rtgAppendWinImportEntry(a *rtgAsm, layout *rtgWinImportLayout, iltOff int, iatOff int, dataRVA int, id int, slot int, thunkSize int, name string) {
	nameAt := len(a.data)
	a.data = rtgAppend16(a.data, 0)
	a.data = rtgAppendStringZ(a.data, name)
	if len(a.data)%2 != 0 {
		a.data = append(a.data, 0)
	}
	nameRVA := dataRVA + nameAt
	iltAt := iltOff + slot*thunkSize
	iatAt := iatOff + slot*thunkSize
	if thunkSize == 8 {
		rtgPut64U32At(a.data, iltAt, nameRVA)
		rtgPut64U32At(a.data, iatAt, nameRVA)
	} else {
		rtgPut32At(a.data, iltAt, nameRVA)
		rtgPut32At(a.data, iatAt, nameRVA)
	}
	layout.iatRVAs[id] = dataRVA + iatAt
}

func rtgPut64U32At(out []byte, at int, v int) {
	rtgPut32At(out, at, v)
	rtgPut32At(out, at+4, 0)
}

func rtgAsmPatchWindows(a *rtgAsm, layout rtgWinImportLayout, imageBase int, is64 bool) {
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
		rtgPut32At(a.code, r.at, disp)
	}
	if a.dataOffset == 0 {
		a.dataOffset = a.codeOffset + len(a.code)
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		if r.kind == rtgAbsWinImportReloc {
			target := rtgWinImportIATRVA(layout, r.off)
			if is64 {
				next := a.codeOffset + r.at + 4
				rtgPut32At(a.code, r.at, target-next)
			} else {
				rtgPut32At(a.code, r.at, imageBase+target)
			}
			continue
		}
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		if is64 {
			next := a.codeOffset + r.at + 4
			rtgPut32At(a.code, r.at, target-next)
		} else {
			rtgPut32At(a.code, r.at, imageBase+target)
		}
	}
}

func rtgAppendPEHeader64(out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	sizeOfImage := rtgAlignValue(dataRVA+dataVirtualSize, rtgWinSectionAlign)
	out = rtgAppendDOSStub(out)
	out = rtgAppend32(out, 0x4550)
	out = rtgAppend32(out, 0x00028664)
	out = rtgAppendUntil(out, len(out)+12)
	out = rtgAppend32(out, 0x002200f0)
	out = rtgAppend32(out, 0x0001020b)
	out = rtgAppend32(out, textRawSize)
	out = rtgAppend32(out, dataRawSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, entryRVA)
	out = rtgAppend32(out, rtgWinSectionRVA)
	out = rtgAppend64U32(out, rtgWinImageBase)
	out = rtgAppend32(out, rtgWinSectionAlign)
	out = rtgAppend32(out, rtgWinFileAlign)
	out = rtgAppend64U32(out, 4)
	out = rtgAppend64U32(out, 4)
	out = rtgAppend32(out, sizeOfImage)
	out = rtgAppend32(out, rtgWinHeadersSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 3)
	for i := 0; i < 3; i++ {
		out = rtgAppend64U32(out, 0x100000)
	}
	out = rtgAppend64U32(out, 0x1000)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 16)
	out = rtgAppendUntil(out, len(out)+8)
	out = rtgAppend32(out, importRVA)
	out = rtgAppend32(out, importSize)
	for i := 2; i < 12; i++ {
		out = rtgAppend32(out, 0)
		out = rtgAppend32(out, 0)
	}
	out = rtgAppend32(out, iatRVA)
	out = rtgAppend32(out, iatSize)
	for i := 13; i < 16; i++ {
		out = rtgAppend32(out, 0)
		out = rtgAppend32(out, 0)
	}
	out = rtgAppendPESection(out, ".text", textVirtualSize, rtgWinSectionRVA, textRawSize, rtgWinHeadersSize, 0x60000020)
	out = rtgAppendPESection(out, ".data", dataVirtualSize, dataRVA, dataRawSize, rtgWinHeadersSize+textRawSize, 0xc0000040)
	out = rtgAppendUntil(out, rtgWinHeadersSize)
	return out
}

func rtgAppendPEHeader32(out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	sizeOfImage := rtgAlignValue(dataRVA+dataVirtualSize, rtgWinSectionAlign)
	out = rtgAppendDOSStub(out)
	out = rtgAppend32(out, 0x4550)
	out = rtgAppend32(out, 0x0002014c)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0x010200e0)
	out = rtgAppend32(out, 0x0001010b)
	out = rtgAppend32(out, textRawSize)
	out = rtgAppend32(out, dataRawSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, entryRVA)
	out = rtgAppend32(out, rtgWinSectionRVA)
	out = rtgAppend32(out, dataRVA)
	out = rtgAppend32(out, rtgWinImageBase)
	out = rtgAppend32(out, rtgWinSectionAlign)
	out = rtgAppend32(out, rtgWinFileAlign)
	out = rtgAppend64U32(out, 4)
	out = rtgAppend64U32(out, 4)
	out = rtgAppend32(out, sizeOfImage)
	out = rtgAppend32(out, rtgWinHeadersSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 3)
	out = rtgAppend32(out, 0x100000)
	out = rtgAppend32(out, 0x100000)
	out = rtgAppend32(out, 0x100000)
	out = rtgAppend32(out, 0x1000)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 16)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, importRVA)
	out = rtgAppend32(out, importSize)
	for i := 2; i < 12; i++ {
		out = rtgAppend32(out, 0)
		out = rtgAppend32(out, 0)
	}
	out = rtgAppend32(out, iatRVA)
	out = rtgAppend32(out, iatSize)
	for i := 13; i < 16; i++ {
		out = rtgAppend32(out, 0)
		out = rtgAppend32(out, 0)
	}
	out = rtgAppendPESection(out, ".text", textVirtualSize, rtgWinSectionRVA, textRawSize, rtgWinHeadersSize, 0x60000020)
	out = rtgAppendPESection(out, ".data", dataVirtualSize, dataRVA, dataRawSize, rtgWinHeadersSize+textRawSize, 0xc0000040)
	out = rtgAppendUntil(out, rtgWinHeadersSize)
	return out
}

func rtgAppendDOSStub(out []byte) []byte {
	out = rtgAppend32(out, 0x5a4d)
	out = rtgAppendUntil(out, 0x3c)
	out = rtgAppend32(out, 0x80)
	out = rtgAppendUntil(out, 0x80)
	return out
}

func rtgAppendPESection(out []byte, name string, virtualSize int, rva int, rawSize int, rawPtr int, characteristics int) []byte {
	for i := 0; i < 8; i++ {
		if i < len(name) {
			out = append(out, name[i])
		} else {
			out = append(out, 0)
		}
	}
	out = rtgAppend32(out, virtualSize)
	out = rtgAppend32(out, rva)
	out = rtgAppend32(out, rawSize)
	out = rtgAppend32(out, rawPtr)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend32(out, characteristics)
	return out
}

const rtgTokEOF = 0
const rtgTokIdent = 1
const rtgTokNumber = 2
const rtgTokFloat = 3
const rtgTokString = 4
const rtgTokChar = 5
const rtgTokPackage = 6
const rtgTokConst = 7
const rtgTokVar = 8
const rtgTokType = 9
const rtgTokFunc = 10
const rtgTokStruct = 11
const rtgTokReturn = 12
const rtgTokIf = 13
const rtgTokElse = 14
const rtgTokFor = 15
const rtgTokBreak = 16
const rtgTokContinue = 17
const rtgTokGoto = 18
const rtgTokSwitch = 19
const rtgTokCase = 20
const rtgTokDefault = 21
const rtgTokOp = 22

type rtgTokens struct {
	data []byte
}

type rtgToken struct {
	kind  int
	start int
	end   int
	line  int
}

const rtgTokenStride = 8

func rtgTokCount(p *rtgProgram) int {
	return len(p.toks.data) / rtgTokenStride
}

func rtgTokKind(p *rtgProgram, i int) int {
	return int(p.toks.data[i<<3])
}

func rtgTokStart(p *rtgProgram, i int) int {
	data := p.toks.data
	off := (i << 3) + 1
	return int(data[off]) | int(data[off+1])<<8 | int(data[off+2])<<16
}

func rtgTokEnd(p *rtgProgram, i int) int {
	data := p.toks.data
	base := i << 3
	startOff := base + 1
	lenOff := base + 4
	start := int(data[startOff]) | int(data[startOff+1])<<8 | int(data[startOff+2])<<16
	size := int(data[lenOff])
	if int(data[base]) != rtgTokOp {
		size = size | int(data[lenOff+1])<<8
	}
	return start + size
}

func rtgTokLine(p *rtgProgram, i int) int {
	data := p.toks.data
	off := (i << 3) + 6
	return int(data[off]) | int(data[off+1])<<8
}

func rtgTokAt(p *rtgProgram, i int) rtgToken {
	var tok rtgToken
	tok.kind = rtgTokKind(p, i)
	tok.start = rtgTokStart(p, i)
	tok.end = rtgTokEnd(p, i)
	tok.line = rtgTokLine(p, i)
	return tok
}

type rtgDecl struct {
	kind      int
	nameStart int
	nameEnd   int
	startTok  int
	endTok    int
}

type rtgFuncDecl struct {
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

type rtgProgram struct {
	src   []byte
	toks  rtgTokens
	decls []rtgDecl
	funcs []rtgFuncDecl
	ok    bool
}

const rtgExprBad = 0
const rtgExprIdent = 1
const rtgExprInt = 2
const rtgExprFloat = 3
const rtgExprString = 4
const rtgExprChar = 5
const rtgExprBool = 6
const rtgExprUnary = 7
const rtgExprBinary = 8
const rtgExprCall = 9
const rtgExprIndex = 10
const rtgExprSelector = 11
const rtgExprComposite = 12
const rtgExprSlice = 13

const rtgStmtBad = 0
const rtgStmtReturn = 1
const rtgStmtIf = 2
const rtgStmtFor = 3
const rtgStmtBreak = 4
const rtgStmtContinue = 5
const rtgStmtGoto = 6
const rtgStmtLabel = 7
const rtgStmtVar = 8
const rtgStmtShort = 9
const rtgStmtAssign = 10
const rtgStmtExpr = 11
const rtgStmtSwitch = 12
const rtgStmtBlock = 13

type rtgExpr struct {
	kind      int
	tok       int
	left      int
	right     int
	firstArg  int
	argCount  int
	nameStart int
	nameEnd   int
}

type rtgExprParse struct {
	prog     *rtgProgram
	pos      int
	end      int
	exprs    []rtgExpr
	args     []int
	fields   []rtgCompositeField
	ok       bool
	hasFloat bool
}

type rtgCompositeField struct {
	nameStart int
	nameEnd   int
	key       int
	expr      int
}

type rtgStmt struct {
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

type rtgBodyParse struct {
	prog      *rtgProgram
	stmtCount int
	ok        bool
}

const rtgStmtWordCount = 11

var rtgBodyStmtData []int

const rtgTypeInvalid = 0
const rtgTypeInt = 1
const rtgTypeInt64 = 2
const rtgTypeByte = 3
const rtgTypeBool = 4
const rtgTypeString = 5
const rtgTypeFloat64 = 6
const rtgTypeInt16 = 7
const rtgTypeInt32 = 8
const rtgTypePointer = 9
const rtgTypeSlice = 10
const rtgTypeStruct = 11
const rtgTypeNamed = 12
const rtgTypeArray = 13
const rtgTypeMap = 14

type rtgTypeInfo struct {
	kind      int
	elem      int
	first     int
	count     int
	size      int
	nameStart int
	nameEnd   int
}

type rtgFieldInfo struct {
	nameStart int
	nameEnd   int
	typ       int
	offset    int
}

type rtgSymbolInfo struct {
	nameStart    int
	nameEnd      int
	kind         int
	typ          int
	initStart    int
	initEnd      int
	iotaValue    int
	constValue   int
	constValueOK int
}

type rtgFuncInfo struct {
	declIndex       int
	nameStart       int
	nameEnd         int
	firstParam      int
	paramCount      int
	resultType      int
	receiverType    int
	bodyStart       int
	bodyEnd         int
	linkStatic      int
	linkDLLStart    int
	linkDLLEnd      int
	linkMethodStart int
	linkMethodEnd   int
}

type rtgMeta struct {
	prog          *rtgProgram
	types         []rtgTypeInfo
	fields        []rtgFieldInfo
	globals       []rtgSymbolInfo
	params        []rtgSymbolInfo
	funcs         []rtgFuncInfo
	globalBuckets []int
	globalNext    []int
	funcBuckets   []int
	funcNext      []int
	ok            bool
}

type rtgCompileResult struct {
	data []byte
	ok   bool
}

type rtgConstResult struct {
	value int
	ok    bool
}

type rtgTypeResult struct {
	typ  int
	next int
}

const rtgIdentAppend = 1
const rtgIdentByteSlice = 2
const rtgIdentMake = 3
const rtgIdentInt = 5
const rtgIdentInt64 = 6
const rtgIdentByte = 7
const rtgIdentLen = 8
const rtgIdentOpen = 9
const rtgIdentClose = 10
const rtgIdentRead = 11
const rtgIdentWrite = 12
const rtgIdentChmod = 13
const rtgIdentCopy = 14
const rtgIdentInt16 = 15
const rtgIdentInt32 = 16
const rtgIdentSyscall = 17
const rtgIdentString = 18
const rtgIdentCap = 19

const rtgDiagParseMissingPackage = 1
const rtgDiagParseMissingPackageName = 2
const rtgDiagParsePackageName = 3
const rtgDiagParseGroupedDecl = 4
const rtgDiagParseTopDecl = 5
const rtgDiagParseFuncDecl = 6
const rtgDiagParseStatement = 7
const rtgDiagParseExpression = 8
const rtgDiagParseComposite = 9
const rtgDiagParseCall = 10
const rtgDiagParseIndex = 11
const rtgDiagParseParen = 12
const rtgDiagMetaConstDecl = 20
const rtgDiagMetaTopDecl = 21
const rtgDiagMetaFuncDecl = 22
const rtgDiagMetaResultType = 23
const rtgDiagMetaParamList = 24
const rtgDiagAppMainRequired = 40
const rtgDiagMainRequiresAppMain = 41
const rtgDiagAppMainSignature = 42
const rtgDiagGlobalCodegen = 50
const rtgDiagFunctionCodegen = 51
const rtgDiagCompileFailed = 52
const rtgDiagFunctionParams = 53
const rtgDiagStatementCodegen = 54
const rtgDiagAssignmentCodegen = 55
const rtgDiagReturnCodegen = 56
const rtgDiagConditionCodegen = 57
const rtgDiagSwitchCodegen = 58
const rtgDiagCallCodegen = 59
const rtgDiagBreakOutsideLoop = 60
const rtgDiagContinueOutsideLoop = 61
const rtgDiagUnsupportedStatement = 62

func rtgProgramError(p *rtgProgram, diag int) {
	p.ok = false
}

func rtgMetaError(m *rtgMeta, diag int) {
	m.ok = false
}

func rtgExprError(ep *rtgExprParse, diag int) {
	ep.ok = false
}

func rtgParseProgram(src []byte) rtgProgram {
	var p rtgProgram
	p.src = src
	p.toks = rtgScan(src)
	declCap := len(src)/1024 + 64
	funcCap := len(src)/768 + 64
	p.decls = make([]rtgDecl, 0, declCap)
	p.funcs = make([]rtgFuncDecl, 0, funcCap)
	p.ok = true

	i := 0
	if !rtgTokIsKind(&p, i, rtgTokPackage) {
		rtgProgramError(&p, rtgDiagParseMissingPackage)
		return p
	}
	i++
	if !rtgTokIsKind(&p, i, rtgTokIdent) {
		rtgProgramError(&p, rtgDiagParseMissingPackageName)
		return p
	}
	i++

	for i < rtgTokCount(&p) && rtgTokKind(&p, i) != rtgTokEOF {
		if rtgTokIsKind(&p, i, rtgTokPackage) {
			i++
			if !rtgTokIsKind(&p, i, rtgTokIdent) {
				rtgProgramError(&p, rtgDiagParsePackageName)
				return p
			}
			i++
			continue
		}
		if rtgTokIsKind(&p, i, rtgTokConst) || rtgTokIsKind(&p, i, rtgTokVar) || rtgTokIsKind(&p, i, rtgTokType) {
			start := i
			kind := int(rtgTokKind(&p, i))
			i++
			if rtgTokCharIs(&p, i, '(') {
				end := rtgSkipBalanced(&p, i, '(', ')')
				if end <= i {
					rtgProgramError(&p, rtgDiagParseGroupedDecl)
					return p
				}
				var decl rtgDecl
				decl.kind = kind
				decl.nameStart = int(rtgTokStart(&p, start))
				decl.nameEnd = int(rtgTokEnd(&p, start))
				decl.startTok = start
				decl.endTok = end
				p.decls = append(p.decls, decl)
				i = end
				continue
			}
			if !rtgTokIsKind(&p, i, rtgTokIdent) {
				rtgProgramError(&p, rtgDiagParseTopDecl)
				return p
			}
			name := rtgTokAt(&p, i)
			i++
			end := rtgSkipTopLevelLine(&p, i)
			var decl rtgDecl
			decl.kind = kind
			decl.nameStart = int(name.start)
			decl.nameEnd = int(name.end)
			decl.startTok = start
			decl.endTok = end
			p.decls = append(p.decls, decl)
			i = end
			continue
		}
		if rtgTokIsKind(&p, i, rtgTokFunc) {
			var fn rtgFuncDecl
			rtgParseFuncDecl(&p, i, &fn)
			if fn.endTok <= i {
				rtgProgramError(&p, rtgDiagParseFuncDecl)
				return p
			}
			p.funcs = append(p.funcs, fn)
			i = fn.endTok
			continue
		}
		i++
	}

	return p
}

func rtgParseFuncDecl(p *rtgProgram, start int, fn *rtgFuncDecl) {
	fn.startTok = start
	i := start + 1
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		receiverEnd := i + 1
		for receiverEnd < rtgTokCount(p) && !rtgTokCharIs(p, receiverEnd, ')') {
			receiverEnd++
		}
		if receiverEnd <= i {
			return
		}
		fn.receiverStart = i + 1
		fn.receiverEnd = receiverEnd
		i = receiverEnd + 1
	}
	if !rtgTokIsKind(p, i, rtgTokIdent) {
		return
	}
	fn.nameTok = i
	fn.nameStart = int(rtgTokStart(p, i))
	fn.nameEnd = int(rtgTokEnd(p, i))
	i++

	for i < rtgTokCount(p) && !rtgTokCharIs(p, i, '{') && rtgTokKind(p, i) != rtgTokEOF {
		i++
	}
	if !rtgTokCharIs(p, i, '{') {
		return
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < rtgTokCount(p) && depth > 0 {
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
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

func rtgSkipBalanced(p *rtgProgram, start int, open byte, close byte) int {
	if !rtgTokCharIs(p, start, open) {
		return start
	}
	depth := 1
	i := start + 1
	for i < rtgTokCount(p) && depth > 0 {
		if rtgTokCharIs(p, i, open) {
			depth++
		} else if rtgTokCharIs(p, i, close) {
			depth--
		}
		i++
	}
	if depth != 0 {
		return start
	}
	return i
}

func rtgSkipTopLevelLine(p *rtgProgram, start int) int {
	if start >= rtgTokCount(p) {
		return start
	}
	line := rtgTokLine(p, start-1)
	i := start
	depth := 0
	for i < rtgTokCount(p) {
		if rtgTokKind(p, i) == rtgTokEOF {
			return i
		}
		if rtgTokLine(p, i) != line && depth == 0 {
			return i
		}
		if rtgTokCharIs(p, i, '{') || rtgTokCharIs(p, i, '(') {
			depth++
		} else if rtgTokCharIs(p, i, '}') || rtgTokCharIs(p, i, ')') {
			depth--
		}
		i++
	}
	return i
}

func rtgScan(src []byte) rtgTokens {
	srcLen := len(src)
	tokenCap := 524288
	if rtgCompilerFixedTarget != 0 {
		tokenCap = srcLen/4 + 8192
	}
	var toks rtgTokens
	toks.data = make([]byte, 0, tokenCap*rtgTokenStride)
	i := 0
	line := 1
	for i < srcLen {
		c := src[i]
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}
		if c == '\n' {
			line++
			i++
			continue
		}
		if c == '/' && i+1 < srcLen && src[i+1] == '/' {
			i += 2
			for i < srcLen && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < srcLen && src[i+1] == '*' {
			i += 2
			for i+1 < srcLen && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < srcLen {
				i += 2
			}
			continue
		}
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			i++
			start := i - 1
			for i < srcLen {
				cc := src[i]
				if !((cc >= 'a' && cc <= 'z') || (cc >= 'A' && cc <= 'Z') || (cc >= '0' && cc <= '9') || cc == '_') {
					break
				}
				i++
			}
			base := len(toks.data)
			toks.data = toks.data[:base+rtgTokenStride]
			size := i - start
			data := toks.data
			data[base] = byte(rtgKeywordKind(src, start, i))
			data[base+1] = byte(start)
			data[base+2] = byte(start >> 8)
			data[base+3] = byte(start >> 16)
			data[base+4] = byte(size)
			data[base+5] = byte(size >> 8)
			data[base+6] = byte(line)
			data[base+7] = byte(line >> 8)
			continue
		}
		if c >= '0' && c <= '9' {
			start := i
			kind := rtgTokNumber
			if c == '0' && i+1 < srcLen && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B') {
				hex := src[i+1] == 'x' || src[i+1] == 'X'
				i += 2
				for i < srcLen {
					cc := src[i]
					if cc == '.' && hex {
						kind = rtgTokFloat
						i++
						continue
					}
					if hex && (cc == 'p' || cc == 'P') {
						kind = rtgTokFloat
						i++
						if i < srcLen && (src[i] == '+' || src[i] == '-') {
							i++
						}
						for i < srcLen && ((src[i] >= '0' && src[i] <= '9') || src[i] == '_') {
							i++
						}
						break
					}
					if !((cc >= 'a' && cc <= 'z') || (cc >= 'A' && cc <= 'Z') || (cc >= '0' && cc <= '9') || cc == '_') {
						break
					}
					i++
				}
			} else {
				i++
				for i < srcLen && src[i] >= '0' && src[i] <= '9' {
					i++
				}
				if i < srcLen && src[i] == '.' {
					kind = rtgTokFloat
					i++
					for i < srcLen && src[i] >= '0' && src[i] <= '9' {
						i++
					}
				}
			}
			base := len(toks.data)
			toks.data = toks.data[:base+rtgTokenStride]
			size := i - start
			data := toks.data
			data[base] = byte(kind)
			data[base+1] = byte(start)
			data[base+2] = byte(start >> 8)
			data[base+3] = byte(start >> 16)
			data[base+4] = byte(size)
			data[base+5] = byte(size >> 8)
			data[base+6] = byte(line)
			data[base+7] = byte(line >> 8)
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < srcLen && src[i] != '"' {
				if src[i] == '\\' && i+1 < srcLen {
					i += 2
				} else {
					if src[i] == '\n' {
						line++
					}
					i++
				}
			}
			if i < srcLen {
				i++
			}
			rtgScanAppendToken(&toks, rtgTokString, start, i-start, line)
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < srcLen && src[i] != '\'' {
				if src[i] == '\\' && i+1 < srcLen {
					i += 2
				} else {
					i++
				}
			}
			if i < srcLen {
				i++
			}
			rtgScanAppendToken(&toks, rtgTokChar, start, i-start, line)
			continue
		}
		start := i
		i++
		if i < srcLen {
			c1 := src[i]
			two := false
			if c1 == '=' {
				if c == ':' || c == '=' || c == '!' || c == '<' || c == '>' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' {
					two = true
				}
			} else if c == '&' && (c1 == '&' || c1 == '^') {
				two = true
			} else if c == '|' && c1 == '|' {
				two = true
			} else if c == '<' && c1 == '<' {
				two = true
			} else if c == '>' && c1 == '>' {
				two = true
			} else if c == '+' && c1 == '+' {
				two = true
			} else if c == '-' && c1 == '-' {
				two = true
			}
			if two {
				i++
			}
		}
		base := len(toks.data)
		toks.data = toks.data[:base+rtgTokenStride]
		size := i - start
		data := toks.data
		data[base] = byte(rtgTokOp)
		data[base+1] = byte(start)
		data[base+2] = byte(start >> 8)
		data[base+3] = byte(start >> 16)
		data[base+4] = byte(size)
		data[base+5] = src[start]
		data[base+6] = byte(line)
		data[base+7] = byte(line >> 8)
	}
	rtgScanAppendToken(&toks, rtgTokEOF, srcLen, 0, line)
	return toks
}

func rtgScanAppendToken(toks *rtgTokens, kind int, start int, size int, line int) {
	base := len(toks.data)
	toks.data = toks.data[:base+rtgTokenStride]
	data := toks.data
	data[base] = byte(kind)
	data[base+1] = byte(start)
	data[base+2] = byte(start >> 8)
	data[base+3] = byte(start >> 16)
	data[base+4] = byte(size)
	data[base+5] = byte(size >> 8)
	data[base+6] = byte(line)
	data[base+7] = byte(line >> 8)
}

func rtgKeywordKind(src []byte, start int, end int) int {
	n := end - start
	if n > 8 {
		return rtgTokIdent
	}
	h := 0
	for i := start; i < end; i++ {
		h = h*5 + int(src[i])
	}
	if n == 2 {
		if h == 627 {
			return rtgTokIf
		}
	}
	if n == 3 {
		if h == 3549 {
			return rtgTokVar
		}
		if h == 3219 {
			return rtgTokFor
		}
	}
	if n == 4 {
		if h == 18186 {
			return rtgTokType
		}
		if h == 16324 {
			return rtgTokFunc
		}
		if h == 16001 {
			return rtgTokElse
		}
		if h == 16341 {
			return rtgTokGoto
		}
		if h == 15476 {
			return rtgTokCase
		}
	}
	if n == 5 {
		if h == 79191 {
			return rtgTokConst
		}
		if h == 78617 {
			return rtgTokBreak
		}
	}
	if n == 6 {
		if h == 449661 {
			return rtgTokStruct
		}
		if h == 437480 {
			return rtgTokReturn
		}
		if h == 450374 {
			return rtgTokSwitch
		}
	}
	if n == 7 {
		if h == 2131416 {
			return rtgTokPackage
		}
		if h == 1957581 {
			return rtgTokDefault
		}
	}
	if n == 8 {
		if h == 9901561 {
			return rtgTokContinue
		}
	}
	return rtgTokIdent
}

func rtgTokIsKind(p *rtgProgram, i int, kind int) bool {
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i << 3
	if base >= len(data) {
		return false
	}
	return int(data[base]) == kind
}

func rtgTokIdentIs(p *rtgProgram, i int, text string) bool {
	return rtgTokIsKind(p, i, rtgTokIdent) && rtgBytesEqualText(p.src, int(rtgTokStart(p, i)), int(rtgTokEnd(p, i)), text)
}

func rtgTokCharIs(p *rtgProgram, i int, c byte) bool {
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i << 3
	if base >= len(data) {
		return false
	}
	if int(data[base]) != rtgTokOp {
		return false
	}
	lenOff := base + 4
	size := int(data[lenOff])
	if size != 1 {
		return false
	}
	return data[lenOff+1] == c
}

func rtgTok2Is(p *rtgProgram, i int, a byte, b byte) bool {
	if i < 0 {
		return false
	}
	data := p.toks.data
	base := i << 3
	if base >= len(data) {
		return false
	}
	if int(data[base]) != rtgTokOp {
		return false
	}
	lenOff := base + 4
	size := int(data[lenOff])
	if size != 2 {
		return false
	}
	if data[lenOff+1] != a {
		return false
	}
	startOff := base + 1
	start := int(data[startOff]) | int(data[startOff+1])<<8 | int(data[startOff+2])<<16
	return p.src[start+1] == b
}

func rtgBoolTokenValue(p *rtgProgram, tok int) int {
	start := rtgTokStart(p, tok)
	if p.src[start] == 't' {
		return 1
	}
	return 0
}

func rtgExprIdentCode(p *rtgProgram, ep *rtgExprParse, idx int) int {
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return 0
	}
	src := p.src
	start := e.nameStart
	n := e.nameEnd - e.nameStart
	if start < 0 || e.nameEnd > len(src) || n <= 0 {
		return 0
	}
	if n == 3 {
		if src[start] == 'c' && src[start+1] == 'a' && src[start+2] == 'p' {
			return rtgIdentCap
		}
		if src[start] == 'i' && src[start+1] == 'n' && src[start+2] == 't' {
			return rtgIdentInt
		}
		if src[start] == 'l' && src[start+1] == 'e' && src[start+2] == 'n' {
			return rtgIdentLen
		}
	}
	if n == 4 {
		if src[start] == 'm' && src[start+1] == 'a' && src[start+2] == 'k' && src[start+3] == 'e' {
			return rtgIdentMake
		}
		if src[start] == 'u' && src[start+1] == 'i' && src[start+2] == 'n' && src[start+3] == 't' {
			return rtgIdentInt
		}
		if src[start] == 'b' && src[start+1] == 'y' && src[start+2] == 't' && src[start+3] == 'e' {
			return rtgIdentByte
		}
		if src[start] == 'o' && src[start+1] == 'p' && src[start+2] == 'e' && src[start+3] == 'n' {
			return rtgIdentOpen
		}
		if src[start] == 'r' && src[start+1] == 'e' && src[start+2] == 'a' && src[start+3] == 'd' {
			return rtgIdentRead
		}
		if src[start] == 'c' && src[start+1] == 'o' && src[start+2] == 'p' && src[start+3] == 'y' {
			return rtgIdentCopy
		}
	}
	if n == 5 {
		if src[start] == 'u' && src[start+1] == 'i' && src[start+2] == 'n' && src[start+3] == 't' && src[start+4] == '8' {
			return rtgIdentInt
		}
		if src[start] == 'i' && src[start+1] == 'n' && src[start+2] == 't' {
			if src[start+3] == '1' && src[start+4] == '6' {
				return rtgIdentInt16
			}
			if src[start+3] == '3' && src[start+4] == '2' {
				return rtgIdentInt32
			}
			if src[start+3] == '6' && src[start+4] == '4' {
				return rtgIdentInt64
			}
		}
		if src[start] == 'c' && src[start+1] == 'l' && src[start+2] == 'o' && src[start+3] == 's' && src[start+4] == 'e' {
			return rtgIdentClose
		}
		if src[start] == 'w' && src[start+1] == 'r' && src[start+2] == 'i' && src[start+3] == 't' && src[start+4] == 'e' {
			return rtgIdentWrite
		}
		if src[start] == 'c' && src[start+1] == 'h' && src[start+2] == 'm' && src[start+3] == 'o' && src[start+4] == 'd' {
			return rtgIdentChmod
		}
	}
	if n == 6 {
		if src[start] == 'u' && src[start+1] == 'i' && src[start+2] == 'n' && src[start+3] == 't' {
			if src[start+4] == '1' && src[start+5] == '6' {
				return rtgIdentInt
			}
			if src[start+4] == '3' && src[start+5] == '2' {
				return rtgIdentInt
			}
			if src[start+4] == '6' && src[start+5] == '4' {
				return rtgIdentInt
			}
		}
		if src[start] == 'a' && src[start+1] == 'p' && src[start+2] == 'p' && src[start+3] == 'e' && src[start+4] == 'n' && src[start+5] == 'd' {
			return rtgIdentAppend
		}
		if src[start] == 's' && src[start+1] == 't' && src[start+2] == 'r' && src[start+3] == 'i' && src[start+4] == 'n' && src[start+5] == 'g' {
			return rtgIdentString
		}
		if src[start] == '[' && src[start+1] == ']' && src[start+2] == 'b' && src[start+3] == 'y' && src[start+4] == 't' && src[start+5] == 'e' {
			return rtgIdentByteSlice
		}
	}
	if n == 7 {
		if src[start] == 's' && src[start+1] == 'y' && src[start+2] == 's' && src[start+3] == 'c' && src[start+4] == 'a' && src[start+5] == 'l' && src[start+6] == 'l' {
			return rtgIdentSyscall
		}
	}
	return 0
}

func rtgBytesEqualText(src []byte, start int, end int, text string) bool {
	if end-start != len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func rtgHexDigitValue(ch byte) int {
	if ch >= '0' && ch <= '9' {
		return int(ch - '0')
	}
	if ch >= 'a' && ch <= 'f' {
		return int(ch-'a') + 10
	}
	if ch >= 'A' && ch <= 'F' {
		return int(ch-'A') + 10
	}
	return -1
}

func rtgDecodeStringToken(p *rtgProgram, tokIndex int) []byte {
	tok := rtgTokAt(p, tokIndex)
	src := p.src
	i := int(tok.start) + 1
	end := int(tok.end) - 1
	out := rtgFixedByteScratch(end - i)
	for i < end {
		if src[i] == '\\' && i+1 < end {
			i++
			if src[i] == 'x' && i+2 < end {
				hi := rtgHexDigitValue(src[i+1])
				lo := rtgHexDigitValue(src[i+2])
				if hi >= 0 && lo >= 0 {
					out = append(out, byte(hi*16+lo))
					i += 3
					continue
				}
			}
			if src[i] == 'n' {
				out = append(out, '\n')
			} else if src[i] == 't' {
				out = append(out, '\t')
			} else if src[i] == 'r' {
				out = append(out, '\r')
			} else if src[i] == '"' {
				out = append(out, '"')
			} else if src[i] == '\\' {
				out = append(out, '\\')
			} else {
				out = append(out, src[i])
			}
			i++
			continue
		}
		out = append(out, src[i])
		i++
	}
	return out
}

func rtgParseIntToken(p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	src := p.src
	start := int(tok.start)
	end := int(tok.end)
	base := 10
	if end-start > 2 && src[start] == '0' {
		prefix := src[start+1]
		if prefix == 'x' || prefix == 'X' {
			base = 16
			start += 2
		} else if prefix == 'b' || prefix == 'B' {
			base = 2
			start += 2
		}
	}
	if base == 10 && end-start > 1 && src[start] == '0' {
		base = 8
		start++
	}
	n := 0
	for i := start; i < end; i++ {
		d := 0
		if src[i] >= '0' && src[i] <= '9' {
			d = int(src[i] - '0')
		} else if src[i] >= 'a' && src[i] <= 'f' {
			d = int(src[i]-'a') + 10
		} else if src[i] >= 'A' && src[i] <= 'F' {
			d = int(src[i]-'A') + 10
		}
		n = n*base + d
	}
	return n
}

func rtgParseFloatTokenScaled(p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	if tok.start+2 < tok.end && p.src[tok.start] == '0' && (p.src[tok.start+1] == 'x' || p.src[tok.start+1] == 'X') {
		return rtgParseHexFloatTokenScaled(p, tokIndex)
	}
	value := 0
	i := tok.start
	for i < tok.end && p.src[i] != '.' {
		if p.src[i] >= '0' && p.src[i] <= '9' {
			value = value*10 + int(p.src[i]-'0')
		}
		i++
	}
	value = value * 4
	if i < tok.end && p.src[i] == '.' {
		i++
		frac := 0
		scale := 1
		for i < tok.end {
			if p.src[i] >= '0' && p.src[i] <= '9' {
				frac = frac*10 + int(p.src[i]-'0')
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

func rtgParseHexFloatTokenScaled(p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	src := p.src
	i := tok.start + 2
	mantissa := 0
	fracDigits := 0
	afterDot := false
	for i < tok.end {
		c := src[i]
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
		digit := rtgHexFloatDigit(c)
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
	if i < tok.end && (src[i] == 'p' || src[i] == 'P') {
		i++
		if i < tok.end && src[i] == '-' {
			sign = -1
			i++
		} else if i < tok.end && src[i] == '+' {
			i++
		}
		for i < tok.end {
			c := src[i]
			if c >= '0' && c <= '9' {
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

func rtgHexFloatDigit(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'f' {
		return int(c-'a') + 10
	}
	if c >= 'A' && c <= 'F' {
		return int(c-'A') + 10
	}
	return -1
}

func rtgParseCharToken(p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	src := p.src
	i := tok.start + 1
	if i >= tok.end-1 {
		return 0
	}
	if src[i] != '\\' {
		return int(src[i])
	}
	i++
	if i >= tok.end-1 {
		return 0
	}
	if src[i] == 'n' {
		return 10
	}
	if src[i] == 't' {
		return 9
	}
	if src[i] == 'r' {
		return 13
	}
	if src[i] == '\\' {
		return 92
	}
	if src[i] == '\'' {
		return 39
	}
	return int(src[i])
}

func rtgEvalConstByName(g *rtgLinearGen, nameStart int, nameEnd int) rtgConstResult {
	builtin := rtgEvalBuiltinConst(g, nameStart, nameEnd)
	if builtin.ok {
		return builtin
	}
	symIndex := rtgFindMetaGlobalIndex(g.meta, nameStart, nameEnd, rtgTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.constValueOK != 0 {
			return rtgConstResultOk(s.constValue)
		}
		var ep rtgExprParse
		if !rtgParseExpressionOK(&ep, g.prog, s.initStart, s.initEnd) {
			var r rtgConstResult
			return r
		}
		rootIndex := len(ep.exprs) - 1
		oldIota := g.constEvalIota
		oldIotaValid := g.constEvalIotaValid
		g.constEvalIota = s.iotaValue
		g.constEvalIotaValid = 1
		result := rtgEvalConstExpr(g, &ep, rootIndex)
		value := result.value
		ok := result.ok
		g.constEvalIota = oldIota
		g.constEvalIotaValid = oldIotaValid
		if ok {
			return rtgConstResultOk(value)
		}
		var r rtgConstResult
		return r
	}
	var r rtgConstResult
	return r
}

func rtgConstResultOk(value int) rtgConstResult {
	var r rtgConstResult
	r.value = value
	r.ok = true
	return r
}

func rtgConvertConstInt(value int, kind int) int {
	if kind == rtgTypeByte {
		return value & 0xff
	}
	if kind == rtgTypeInt && rtgNativeIntSize == 4 {
		kind = rtgTypeInt32
	}
	if kind == rtgTypeInt16 {
		value = value & 0xffff
		if value >= 0x8000 {
			value -= 0x10000
		}
		return value
	}
	if kind == rtgTypeInt32 {
		limit := 2147483647
		if value > limit {
			value -= limit
			value -= limit
			value -= 2
		}
		return value
	}
	return value
}

func rtgConvertConstScalar(value int, sourceKind int, destKind int) int {
	if destKind == rtgTypeFloat64 && sourceKind != rtgTypeFloat64 {
		return value << 2
	}
	if destKind != rtgTypeFloat64 && sourceKind == rtgTypeFloat64 {
		value = value >> 2
	}
	return rtgConvertConstInt(value, destKind)
}

func rtgEvalConstExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) rtgConstResult {
	p := g.prog
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		value := rtgParseIntToken(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		return rtgConstResultOk(value)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if g.locals[localIndex].constValid != 0 {
				return rtgConstResultOk(g.locals[localIndex].constValue)
			}
			var r rtgConstResult
			return r
		}
		result := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		return result
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32 || callee == rtgIdentInt64) {
			argIndex := ep.args[e.firstArg]
			result := rtgEvalConstExpr(g, ep, argIndex)
			if result.ok {
				destKind := rtgTypeInt
				if callee == rtgIdentByte {
					destKind = rtgTypeByte
				}
				if callee == rtgIdentInt16 {
					destKind = rtgTypeInt16
				}
				if callee == rtgIdentInt32 {
					destKind = rtgTypeInt32
				}
				if callee == rtgIdentInt64 {
					destKind = rtgTypeInt64
				}
				source := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, argIndex))
				result.value = rtgConvertConstScalar(result.value, source.kind, destKind)
			}
			return result
		}
		if e.argCount == 1 {
			conversionType := rtgConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				resolved := rtgResolveType(g.meta, conversionType)
				if rtgTypeKindIsScalarValue(resolved.kind) {
					argIndex := ep.args[e.firstArg]
					result := rtgEvalConstExpr(g, ep, argIndex)
					if result.ok {
						source := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, argIndex))
						result.value = rtgConvertConstScalar(result.value, source.kind, resolved.kind)
					}
					return result
				}
			}
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprUnary {
		inner := rtgEvalConstExpr(g, ep, e.left)
		if !inner.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokCharIs(p, e.tok, '-') {
			return rtgConstResultOk(-inner.value)
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return rtgConstResultOk(inner.value)
		}
		if rtgTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		opTok := e.tok
		rightIndex := e.right
		rightExpr := &ep.exprs[rightIndex]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		left := rtgEvalConstExpr(g, ep, e.left)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		if rtgTok2Is(p, e.tok, '&', '&') {
			if left.value == 0 {
				return rtgConstResultOk(0)
			}
			right := rtgEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		if rtgTok2Is(p, e.tok, '|', '|') {
			if left.value != 0 {
				return rtgConstResultOk(1)
			}
			right := rtgEvalConstExpr(g, ep, rightIndex)
			if !right.ok {
				var r rtgConstResult
				return r
			}
			if right.value != 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var right rtgConstResult
		if rightKind == rtgExprInt {
			value := rtgParseIntToken(p, rightTok)
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			right = rtgConstResultOk(value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			right = rtgConstResultOk(value)
		} else {
			right = rtgEvalConstExpr(g, ep, rightIndex)
		}
		if !right.ok {
			var r rtgConstResult
			return r
		}
		result := rtgEvalConstBinary(g, opTok, left.value, right.value)
		return result
	}
	var r rtgConstResult
	return r
}

func rtgEvalConstBinary(g *rtgLinearGen, tok int, left int, right int) rtgConstResult {
	p := g.prog
	if tok < 0 || tok >= rtgTokCount(p) {
		var r rtgConstResult
		return r
	}
	start := rtgTokStart(p, tok)
	end := rtgTokEnd(p, tok)
	n := end - start
	value := 0
	ok := true
	if n == 1 {
		c := p.src[start]
		if c == '+' {
			value = left + right
		} else if c == '-' {
			value = left - right
		} else if c == '*' {
			value = left * right
		} else if c == '/' {
			if right == 0 {
				var r rtgConstResult
				return r
			}
			value = left / right
		} else if c == '%' {
			if right == 0 {
				var r rtgConstResult
				return r
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
		c0 := p.src[start]
		c1 := p.src[start+1]
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
		return rtgConstResultOk(value)
	}
	var r rtgConstResult
	return r
}

func rtgExprIsIdentText(p *rtgProgram, ep *rtgExprParse, idx int, text string) bool {
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return false
	}
	return rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
}

func rtgExprIsStringBuiltin(p *rtgProgram, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return false
	}
	start := e.nameStart
	if e.nameEnd-start != 6 {
		return false
	}
	src := p.src
	if start < 0 || e.nameEnd > len(src) {
		return false
	}
	return src[start] == 's' && src[start+1] == 't' && src[start+2] == 'r' && src[start+3] == 'i' && src[start+4] == 'n' && src[start+5] == 'g'
}

func rtgParseExpression(p *rtgProgram, start int, end int) rtgExprParse {
	var ep rtgExprParse
	rtgParseExpressionInto(&ep, p, start, end)
	return ep
}

func rtgParseExpressionInto(ep *rtgExprParse, p *rtgProgram, start int, end int) {
	var zero rtgExprParse
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
	ep.exprs = make([]rtgExpr, 0, capacity)
	ep.args = make([]int, 0, capacity)
	ep.fields = make([]rtgCompositeField, 0, (capacity+1)/2)
	ep.ok = true
	rtgParseBinaryExpr(ep, 1)
	if ep.pos < ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
	}
}

func rtgParseExpressionOK(ep *rtgExprParse, p *rtgProgram, start int, end int) bool {
	return rtgParseExpressionRoot(ep, p, start, end) >= 0
}

func rtgParseExpressionRoot(ep *rtgExprParse, p *rtgProgram, start int, end int) int {
	rtgParseExpressionInto(ep, p, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return -1
	}
	return len(ep.exprs) - 1
}

func rtgParseBinaryExpr(ep *rtgExprParse, minPrec int) int {
	left := rtgParseUnaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		prec := rtgTokenPrecedence(ep.prog, ep.pos)
		if prec < minPrec {
			break
		}
		opTok := ep.pos
		ep.pos++
		right := rtgParseBinaryExpr(ep, prec+1)
		left = rtgAddExpr(ep, rtgExprBinary, opTok, left, right, 0, 0, 0, 0)
	}
	return left
}

func rtgParseUnaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
		return 0
	}
	if rtgTokCharIs(ep.prog, ep.pos, '+') || rtgTokCharIs(ep.prog, ep.pos, '-') || rtgTokCharIs(ep.prog, ep.pos, '!') || rtgTokCharIs(ep.prog, ep.pos, '&') || rtgTokCharIs(ep.prog, ep.pos, '*') {
		opTok := ep.pos
		ep.pos++
		inner := rtgParseUnaryExpr(ep)
		return rtgAddExpr(ep, rtgExprUnary, opTok, inner, 0, 0, 0, 0, 0)
	}
	return rtgParsePostfixExpr(ep)
}

func rtgParsePostfixExpr(ep *rtgExprParse) int {
	left := rtgParsePrimaryExpr(ep)
	for ep.ok && ep.pos < ep.end {
		if rtgTokCharIs(ep.prog, ep.pos, '{') {
			base := &ep.exprs[left]
			if base.kind != rtgExprIdent {
				rtgExprError(ep, rtgDiagParseComposite)
				return left
			}
			compositeFields := rtgFixedCompositeFieldScratch(8)
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
				compositeFields = append(compositeFields, rtgParseCompositeField(ep))
				if rtgTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !rtgTokCharIs(ep.prog, ep.pos, '}') {
				rtgExprError(ep, rtgDiagParseComposite)
				return left
			}
			ep.pos++
			first := len(ep.fields)
			for i := 0; i < len(compositeFields); i++ {
				field := compositeFields[i]
				ep.fields = append(ep.fields, field)
			}
			count := len(compositeFields)
			left = rtgAddExpr(ep, rtgExprComposite, base.tok, 0, 0, first, count, base.nameStart, base.nameEnd)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '(') {
			callTok := ep.pos
			callExpanded := false
			ep.pos++
			argsStart := ep.pos
			scanPos := ep.pos
			count := 0
			for scanPos < ep.end && !rtgTokCharIs(ep.prog, scanPos, ')') {
				argEnd := rtgFindExprBoundary(ep.prog, scanPos, ep.end)
				if rtgTokCharIs(ep.prog, argEnd, '{') {
					closeTok := rtgSkipBalanced(ep.prog, argEnd, '{', '}')
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				count++
				scanPos = argEnd
				if rtgTokCharIs(ep.prog, scanPos, ',') {
					scanPos++
				}
			}
			first := len(ep.args)
			for i := 0; i < count; i++ {
				ep.args = append(ep.args, 0)
			}
			argIndex := 0
			ep.pos = argsStart
			for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, ')') {
				argEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
				if rtgTokCharIs(ep.prog, argEnd, '{') {
					closeTok := rtgSkipBalanced(ep.prog, argEnd, '{', '}')
					if closeTok > argEnd {
						argEnd = closeTok
					}
				}
				parseEnd := argEnd
				if argEnd-ep.pos >= 4 && rtgTokCharIs(ep.prog, argEnd-3, '.') && rtgTokCharIs(ep.prog, argEnd-2, '.') && rtgTokCharIs(ep.prog, argEnd-1, '.') {
					callExpanded = true
					parseEnd = argEnd - 3
				}
				oldEnd := ep.end
				ep.end = parseEnd
				argRoot := rtgParseBinaryExpr(ep, 1)
				ep.end = oldEnd
				ep.args[first+argIndex] = argRoot
				argIndex++
				ep.pos = argEnd
				if rtgTokCharIs(ep.prog, ep.pos, ',') {
					ep.pos++
				}
			}
			if !rtgTokCharIs(ep.prog, ep.pos, ')') {
				rtgExprError(ep, rtgDiagParseCall)
				return left
			}
			ep.pos++
			expanded := 0
			if callExpanded {
				expanded = 1
			}
			left = rtgAddExpr(ep, rtgExprCall, callTok, left, 0, first, count, expanded, 0)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '[') {
			indexTok := ep.pos
			ep.pos++
			indexStart := ep.pos
			indexEnd := rtgFindMatchingExprClose(ep.prog, ep.pos, ep.end, '[', ']')
			if indexEnd <= ep.pos {
				rtgExprError(ep, rtgDiagParseIndex)
				return left
			}
			colon := rtgFindSliceColon(ep.prog, indexStart, indexEnd)
			if colon >= 0 {
				low := -1
				high := -1
				max := -1
				highEnd := indexEnd
				secondColon := rtgFindSliceColon(ep.prog, colon+1, indexEnd)
				if secondColon >= 0 {
					highEnd = secondColon
				}
				oldEnd := ep.end
				if colon > indexStart {
					ep.pos = indexStart
					ep.end = colon
					low = rtgParseBinaryExpr(ep, 1)
				}
				if colon+1 < highEnd {
					ep.pos = colon + 1
					ep.end = highEnd
					high = rtgParseBinaryExpr(ep, 1)
				}
				if secondColon >= 0 && secondColon+1 < indexEnd {
					ep.pos = secondColon + 1
					ep.end = indexEnd
					max = rtgParseBinaryExpr(ep, 1)
				}
				ep.end = oldEnd
				ep.pos = indexEnd + 1
				left = rtgAddExpr(ep, rtgExprSlice, indexTok, left, high, low, 0, max, 0)
				continue
			}
			oldEnd := ep.end
			ep.end = indexEnd
			right := rtgParseBinaryExpr(ep, 1)
			ep.end = oldEnd
			ep.pos = indexEnd + 1
			left = rtgAddExpr(ep, rtgExprIndex, indexTok, left, right, 0, 0, 0, 0)
			continue
		}
		if rtgTokCharIs(ep.prog, ep.pos, '.') && rtgTokIsKind(ep.prog, ep.pos+1, rtgTokIdent) {
			dotTok := ep.pos
			nameTok := rtgTokAt(ep.prog, ep.pos+1)
			ep.pos += 2
			left = rtgAddExpr(ep, rtgExprSelector, dotTok, left, 0, 0, 0, int(nameTok.start), int(nameTok.end))
			continue
		}
		break
	}
	return left
}

func rtgFindSliceColon(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ':') {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			brace--
		}
	}
	return -1
}

func rtgParseImplicitCompositeExpr(ep *rtgExprParse) int {
	openTok := ep.pos
	if !rtgTokCharIs(ep.prog, ep.pos, '{') {
		rtgExprError(ep, rtgDiagParseComposite)
		return 0
	}
	compositeFields := rtgFixedCompositeFieldScratch(8)
	ep.pos++
	for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
		compositeFields = append(compositeFields, rtgParseCompositeField(ep))
		if rtgTokCharIs(ep.prog, ep.pos, ',') {
			ep.pos++
		}
	}
	if !rtgTokCharIs(ep.prog, ep.pos, '}') {
		rtgExprError(ep, rtgDiagParseComposite)
		return 0
	}
	ep.pos++
	first := len(ep.fields)
	for i := 0; i < len(compositeFields); i++ {
		field := compositeFields[i]
		ep.fields = append(ep.fields, field)
	}
	count := len(compositeFields)
	return rtgAddExpr(ep, rtgExprComposite, openTok, 0, 0, first, count, 0, 0)
}

func rtgParseCompositeField(ep *rtgExprParse) rtgCompositeField {
	var field rtgCompositeField
	field.key = -1
	fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
	colon := rtgFindSliceColon(ep.prog, ep.pos, fieldEnd)
	oldEnd := ep.end
	if colon >= ep.pos {
		keyStart := ep.pos
		ep.end = colon
		field.key = rtgParseBinaryExpr(ep, 1)
		if colon == keyStart+1 && rtgTokIsKind(ep.prog, keyStart, rtgTokIdent) {
			field.nameStart = int(rtgTokStart(ep.prog, keyStart))
			field.nameEnd = int(rtgTokEnd(ep.prog, keyStart))
		}
		ep.pos = colon + 1
	}
	ep.end = fieldEnd
	if rtgTokCharIs(ep.prog, ep.pos, '{') {
		field.expr = rtgParseImplicitCompositeExpr(ep)
	} else {
		field.expr = rtgParseBinaryExpr(ep, 1)
	}
	ep.end = oldEnd
	ep.pos = fieldEnd
	return field
}

func rtgParsePrimaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
		return 0
	}
	if rtgTokCharIs(ep.prog, ep.pos, '[') && rtgTokIsKind(ep.prog, ep.pos+1, rtgTokNumber) && rtgTokCharIs(ep.prog, ep.pos+2, ']') {
		startTok := ep.pos
		typeEnd := ep.pos + 3
		for rtgTokCharIs(ep.prog, typeEnd, '[') && rtgTokIsKind(ep.prog, typeEnd+1, rtgTokNumber) && rtgTokCharIs(ep.prog, typeEnd+2, ']') {
			typeEnd += 3
		}
		if rtgTokIsKind(ep.prog, typeEnd, rtgTokIdent) {
			nameEnd := int(rtgTokEnd(ep.prog, typeEnd))
			ep.pos = typeEnd + 1
			return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, int(rtgTokStart(ep.prog, startTok)), nameEnd)
		}
	}
	if rtgTokCharIs(ep.prog, ep.pos, '[') && rtgTokCharIs(ep.prog, ep.pos+1, ']') && rtgTokIsKind(ep.prog, ep.pos+2, rtgTokIdent) {
		startTok := ep.pos
		nameEnd := int(rtgTokEnd(ep.prog, ep.pos+2))
		ep.pos += 3
		return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, int(rtgTokStart(ep.prog, startTok)), nameEnd)
	}
	if rtgTokCharIs(ep.prog, ep.pos, '[') && rtgTokCharIs(ep.prog, ep.pos+1, ']') && rtgTokCharIs(ep.prog, ep.pos+2, '*') && rtgTokIsKind(ep.prog, ep.pos+3, rtgTokIdent) {
		startTok := ep.pos
		nameEnd := int(rtgTokEnd(ep.prog, ep.pos+3))
		ep.pos += 4
		return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, int(rtgTokStart(ep.prog, startTok)), nameEnd)
	}
	if rtgTokIdentIs(ep.prog, ep.pos, "map") && rtgTokCharIs(ep.prog, ep.pos+1, '[') {
		startTok := ep.pos
		closeTok := rtgFindMatchingExprClose(ep.prog, ep.pos+2, ep.end, '[', ']')
		if closeTok > ep.pos+1 && closeTok+1 < ep.end && rtgTokIsKind(ep.prog, closeTok+1, rtgTokIdent) {
			nameEnd := int(rtgTokEnd(ep.prog, closeTok+1))
			ep.pos = closeTok + 2
			return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, int(rtgTokStart(ep.prog, startTok)), nameEnd)
		}
	}
	if rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) {
		tokStart := int(rtgTokStart(ep.prog, ep.pos))
		tokEnd := int(rtgTokEnd(ep.prog, ep.pos))
		ep.pos++
		if rtgBytesEqualText(ep.prog.src, tokStart, tokEnd, "true") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		if rtgBytesEqualText(ep.prog.src, tokStart, tokEnd, "false") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		return rtgAddExpr(ep, rtgExprIdent, ep.pos-1, 0, 0, 0, 0, tokStart, tokEnd)
	}
	if rtgTokIsKind(ep.prog, ep.pos, rtgTokNumber) {
		ep.pos++
		return rtgAddExpr(ep, rtgExprInt, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if rtgTokIsKind(ep.prog, ep.pos, rtgTokFloat) {
		ep.pos++
		ep.hasFloat = true
		return rtgAddExpr(ep, rtgExprFloat, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if rtgTokIsKind(ep.prog, ep.pos, rtgTokString) {
		ep.pos++
		return rtgAddExpr(ep, rtgExprString, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if rtgTokIsKind(ep.prog, ep.pos, rtgTokChar) {
		ep.pos++
		return rtgAddExpr(ep, rtgExprChar, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if rtgTokCharIs(ep.prog, ep.pos, '(') {
		ep.pos++
		inner := rtgParseBinaryExpr(ep, 1)
		if !rtgTokCharIs(ep.prog, ep.pos, ')') {
			rtgExprError(ep, rtgDiagParseParen)
			return inner
		}
		ep.pos++
		return inner
	}
	rtgExprError(ep, rtgDiagParseExpression)
	return 0
}

func rtgAddExpr(ep *rtgExprParse, kind int, tok int, left int, right int, firstArg int, argCount int, nameStart int, nameEnd int) int {
	var e rtgExpr
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

func rtgTokenPrecedence(p *rtgProgram, pos int) int {
	if pos < 0 || pos >= rtgTokCount(p) {
		return 0
	}
	start := rtgTokStart(p, pos)
	end := rtgTokEnd(p, pos)
	if end-start == 1 {
		c := p.src[start]
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
		c0 := p.src[start]
		c1 := p.src[start+1]
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

func rtgFindExprBoundary(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, '{') {
			closeTok := rtgSkipBalanced(p, i, '{', '}')
			if closeTok > i {
				i = closeTok
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && (rtgTokCharIs(p, i, ',') || rtgTokCharIs(p, i, ')') || rtgTokCharIs(p, i, ']') || rtgTokCharIs(p, i, '}')) {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren == 0 {
				return i
			}
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			if brack == 0 {
				return i
			}
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return i
			}
			brace--
		}
		i++
	}
	return i
}

func rtgFindMatchingExprClose(p *rtgProgram, start int, end int, open byte, close byte) int {
	depth := 0
	i := start
	for i < end {
		if rtgTokCharIs(p, i, open) {
			depth++
		} else if rtgTokCharIs(p, i, close) {
			if depth == 0 {
				return i
			}
			depth--
		}
		i++
	}
	return start
}

func rtgParseOneStatement(bp *rtgBodyParse, start int, end int) int {
	p := bp.prog
	if start >= end {
		return end
	}
	if rtgTokIsKind(p, start, rtgTokReturn) {
		exprEnd := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtReturn, start, exprEnd, start+1, exprEnd, 0, 0, 0, 0, 0, 0)
		return exprEnd
	}
	if rtgTokIsKind(p, start, rtgTokIf) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		stmt := rtgStmt{kind: rtgStmtIf, startTok: start, endTok: bodyEnd + 1, exprStart: start + 1, exprEnd: bodyStart, bodyStart: bodyStart + 1, bodyEnd: bodyEnd}
		next := bodyEnd + 1
		if rtgTokIsKind(p, next, rtgTokElse) {
			if rtgTokIsKind(p, next+1, rtgTokIf) {
				foundEnd := rtgFindIfStatementEnd(p, next+1, end)
				if foundEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 1
				stmt.elseEnd = foundEnd
				stmt.endTok = foundEnd
				next = foundEnd
			} else if rtgTokCharIs(p, next+1, '{') {
				elseBodyEnd := rtgFindMatchingBrace(p, next+1, end)
				if elseBodyEnd <= next+1 {
					return start
				}
				stmt.elseStart = next + 2
				stmt.elseEnd = elseBodyEnd
				stmt.endTok = elseBodyEnd + 1
				next = elseBodyEnd + 1
			}
		}
		rtgAppendStmt(bp, stmt)
		return next
	}
	if rtgTokIsKind(p, start, rtgTokSwitch) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		rtgAddStmt(bp, rtgStmtSwitch, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokIsKind(p, start, rtgTokFor) {
		bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
		if bodyStart <= start {
			return start
		}
		bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
		if bodyEnd <= bodyStart {
			return start
		}
		rtgAddStmt(bp, rtgStmtFor, start, bodyEnd+1, start+1, bodyStart, bodyStart+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokCharIs(p, start, '{') {
		bodyEnd := rtgFindMatchingBrace(p, start, end)
		if bodyEnd <= start {
			return start
		}
		rtgAddStmt(bp, rtgStmtBlock, start, bodyEnd+1, 0, 0, start+1, bodyEnd, 0, 0, 0, 0)
		return bodyEnd + 1
	}
	if rtgTokIsKind(p, start, rtgTokBreak) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtBreak, start, endTok, 0, 0, 0, 0, 0, 0, 0, 0)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokContinue) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		rtgAddStmt(bp, rtgStmtContinue, start, endTok, 0, 0, 0, 0, 0, 0, 0, 0)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokGoto) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = int(rtgTokStart(p, start+1))
			nameEnd = int(rtgTokEnd(p, start+1))
		}
		rtgAddStmt(bp, rtgStmtGoto, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, ':') {
		name := rtgTokAt(p, start)
		rtgAddStmt(bp, rtgStmtLabel, start, start+2, 0, 0, 0, 0, 0, 0, int(name.start), int(name.end))
		return start + 2
	}
	if rtgTokIsKind(p, start, rtgTokVar) || rtgTokIsKind(p, start, rtgTokConst) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = int(rtgTokStart(p, start+1))
			nameEnd = int(rtgTokEnd(p, start+1))
		}
		rtgAddStmt(bp, rtgStmtVar, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	lineEnd := rtgStatementLineEnd(p, start, end)
	assignTok := rtgFindAssignmentToken(p, start, lineEnd)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTok2Is(p, assignTok, ':', '=') {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = int(rtgTokStart(p, start))
			nameEnd = int(rtgTokEnd(p, start))
		}
		rtgAddStmt(bp, kind, start, lineEnd, assignTok+1, lineEnd, 0, 0, 0, 0, nameStart, nameEnd)
		return lineEnd
	}
	rtgAddStmt(bp, rtgStmtExpr, start, lineEnd, start, lineEnd, 0, 0, 0, 0, 0, 0)
	return lineEnd
}

func rtgAddStmt(bp *rtgBodyParse, kind int, startTok int, endTok int, exprStart int, exprEnd int, bodyStart int, bodyEnd int, elseStart int, elseEnd int, nameStart int, nameEnd int) {
	var stmt rtgStmt
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
	rtgAppendStmt(bp, stmt)
}

func rtgAppendStmt(bp *rtgBodyParse, stmt rtgStmt) {
	base := bp.stmtCount * rtgStmtWordCount
	if bp.stmtCount < 0 || base < 0 || base+rtgStmtWordCount > len(rtgBodyStmtData) {
		bp.ok = false
		return
	}
	rtgBodyStmtData[base+0] = stmt.kind
	rtgBodyStmtData[base+1] = stmt.startTok
	rtgBodyStmtData[base+2] = stmt.endTok
	rtgBodyStmtData[base+3] = stmt.exprStart
	rtgBodyStmtData[base+4] = stmt.exprEnd
	rtgBodyStmtData[base+5] = stmt.bodyStart
	rtgBodyStmtData[base+6] = stmt.bodyEnd
	rtgBodyStmtData[base+7] = stmt.elseStart
	rtgBodyStmtData[base+8] = stmt.elseEnd
	rtgBodyStmtData[base+9] = stmt.nameStart
	rtgBodyStmtData[base+10] = stmt.nameEnd
	bp.stmtCount++
}

func rtgBodyStmtAt(bp *rtgBodyParse, index int) rtgStmt {
	var stmt rtgStmt
	base := index * rtgStmtWordCount
	if index < 0 || base < 0 || base+rtgStmtWordCount > len(rtgBodyStmtData) {
		bp.ok = false
		return stmt
	}
	stmt.kind = rtgBodyStmtData[base+0]
	stmt.startTok = rtgBodyStmtData[base+1]
	stmt.endTok = rtgBodyStmtData[base+2]
	stmt.exprStart = rtgBodyStmtData[base+3]
	stmt.exprEnd = rtgBodyStmtData[base+4]
	stmt.bodyStart = rtgBodyStmtData[base+5]
	stmt.bodyEnd = rtgBodyStmtData[base+6]
	stmt.elseStart = rtgBodyStmtData[base+7]
	stmt.elseEnd = rtgBodyStmtData[base+8]
	stmt.nameStart = rtgBodyStmtData[base+9]
	stmt.nameEnd = rtgBodyStmtData[base+10]
	return stmt
}

func rtgFindIfStatementEnd(p *rtgProgram, start int, end int) int {
	if !(rtgTokIsKind(p, start, rtgTokIf)) {
		return start
	}
	bodyStart := rtgFindStatementBodyOpen(p, start+1, end)
	if bodyStart <= start {
		return start
	}
	bodyEnd := rtgFindMatchingBrace(p, bodyStart, end)
	if bodyEnd <= bodyStart {
		return start
	}
	next := bodyEnd + 1
	if rtgTokIsKind(p, next, rtgTokElse) {
		if rtgTokIsKind(p, next+1, rtgTokIf) {
			return rtgFindIfStatementEnd(p, next+1, end)
		}
		if rtgTokCharIs(p, next+1, '{') {
			elseEnd := rtgFindMatchingBrace(p, next+1, end)
			if elseEnd <= next+1 {
				return start
			}
			return elseEnd + 1
		}
	}
	return next
}

func rtgStatementLineEnd(p *rtgProgram, start int, end int) int {
	if start >= end {
		return end
	}
	line := rtgTokLine(p, start)
	i := start
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if i > start && paren == 0 && brack == 0 && brace == 0 {
			if rtgTokIsKind(p, i, rtgTokEOF) {
				return i
			}
			if rtgTokCharIs(p, i, ';') {
				return i
			}
			if rtgTokLine(p, i) != line {
				if rtgTokCharIs(p, i, '{') {
					return i
				}
				if rtgTokIsKind(p, i, rtgTokReturn) || rtgTokIsKind(p, i, rtgTokIf) || rtgTokIsKind(p, i, rtgTokFor) || rtgTokIsKind(p, i, rtgTokSwitch) || rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault) || rtgTokIsKind(p, i, rtgTokVar) || rtgTokIsKind(p, i, rtgTokConst) || rtgTokIsKind(p, i, rtgTokBreak) || rtgTokIsKind(p, i, rtgTokContinue) || rtgTokIsKind(p, i, rtgTokGoto) {
					return i
				}
				if rtgLineContinuesAfterPrevToken(p, i) {
					line = rtgTokLine(p, i)
				} else {
					return i
				}
			}
		}
		closed := false
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
			closed = true
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
			closed = true
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return i
			}
			brace--
			closed = true
		}
		if i > start && rtgTokLine(p, i) != line && paren == 0 && brack == 0 && brace == 0 {
			if rtgLineContinuesAfterPrevToken(p, i) {
				line = rtgTokLine(p, i)
			} else {
				if closed {
					return i + 1
				}
				return i
			}
		}
		i++
	}
	return i
}

func rtgLineContinuesAfterPrevToken(p *rtgProgram, i int) bool {
	if i <= 0 {
		return false
	}
	prev := i - 1
	tok := rtgTokAt(p, prev)
	tokStart := tok.start
	tokEnd := tok.end
	if tokEnd <= tokStart {
		return false
	}
	c := p.src[tokStart]
	if c == ',' {
		return true
	}
	if c == '*' || c == '&' {
		return true
	}
	if c == '+' {
		if tokEnd == tokStart+1 || p.src[tokStart+1] != '+' {
			return true
		}
	}
	return false
}

func rtgFindNextTokenText(p *rtgProgram, start int, end int, text byte) int {
	i := start
	for i < end {
		if rtgTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start
}

func rtgFindStatementBodyOpen(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	for i < end {
		tok := rtgTokAt(p, i)
		if tok.end == tok.start+1 {
			c := p.src[tok.start]
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
					closeTok := rtgSkipBalanced(p, i, '{', '}')
					if closeTok > i && closeTok < end && rtgTokCharIs(p, closeTok, '{') {
						i = closeTok
						continue
					}
					return i
				}
				closeTok := rtgSkipBalanced(p, i, '{', '}')
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

func rtgFindMatchingBrace(p *rtgProgram, openTok int, end int) int {
	if !rtgTokCharIs(p, openTok, '{') {
		return openTok
	}
	depth := 1
	i := openTok + 1
	for i < end {
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return openTok
}

func rtgFindAssignmentToken(p *rtgProgram, start int, end int) int {
	i := start
	paren := 0
	brack := 0
	for i < end {
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if paren == 0 && brack == 0 {
			if rtgTokCharIs(p, i, '=') || rtgTok2Is(p, i, ':', '=') || rtgTok2Is(p, i, '+', '=') || rtgTok2Is(p, i, '-', '=') || rtgTok2Is(p, i, '*', '=') || rtgTok2Is(p, i, '/', '=') || rtgTok2Is(p, i, '%', '=') {
				return i
			}
		}
		i++
	}
	return start
}

func rtgBuildMeta(pp *rtgProgram) rtgMeta {
	var m rtgMeta
	rtgBuildMetaInto(pp, &m)
	return m
}

func rtgBuildMetaInto(pp *rtgProgram, m *rtgMeta) {
	p := pp
	m.prog = p
	typeCap := len(p.decls)*4 + 256
	fieldCap := len(p.decls)*2 + 256
	globalCap := len(p.decls) + 128
	paramCap := len(p.funcs)*3 + 256
	m.types = make([]rtgTypeInfo, 0, typeCap)
	m.fields = make([]rtgFieldInfo, 0, fieldCap)
	m.globals = make([]rtgSymbolInfo, 0, globalCap)
	m.params = make([]rtgSymbolInfo, 0, paramCap)
	m.funcs = make([]rtgFuncInfo, 0, len(p.funcs)+128)
	m.globalBuckets = make([]int, globalCap*2)
	for i := 0; i < len(m.globalBuckets); i++ {
		m.globalBuckets[i] = -1
	}
	m.funcBuckets = make([]int, (len(p.funcs)+128)*2)
	for i := 0; i < len(m.funcBuckets); i++ {
		m.funcBuckets[i] = -1
	}
	m.ok = true
	rtgInitBuiltinTypes(m)

	parsedGroupStart := -1
	parsedGroupEnd := -1
	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != rtgTokType && decl.kind != rtgTokVar && decl.kind != rtgTokConst {
			continue
		}
		if !rtgTokIsKind(p, decl.startTok, decl.kind) {
			groupStart, groupEnd, isGroup := rtgFindContainingTopDeclGroup(p, decl.kind, decl.startTok, decl.endTok)
			if isGroup {
				if parsedGroupStart == groupStart && parsedGroupEnd == groupEnd {
					continue
				}
				rtgParseTopDeclGroup(m, p, decl.kind, groupStart+1, groupEnd)
				parsedGroupStart = groupStart
				parsedGroupEnd = groupEnd
				continue
			}
			rtgParseTopDeclEntry(m, p, decl.kind, decl.startTok, decl.endTok)
			continue
		}
		entryStart := decl.startTok + 1
		if rtgTokCharIs(p, entryStart, '(') {
			rtgParseTopDeclGroup(m, p, decl.kind, entryStart, decl.endTok)
			continue
		}
		if decl.kind == rtgTokConst {
			rtgParseConstDecls(m, p, entryStart, decl.endTok)
		} else {
			rtgParseTopDeclEntry(m, p, decl.kind, entryStart, decl.endTok)
		}
	}
	rtgFinalizeTypeLayouts(m)
	for i := 0; i < len(p.funcs); i++ {
		rtgParseFuncInfo(m, i)
	}
	rtgBuildFuncLookup(m)
	rtgResolveGlobalCallTypes(m)
}

func rtgFindContainingTopDeclGroup(p *rtgProgram, kind int, start int, end int) (int, int, bool) {
	i := start - 1
	for i >= 0 {
		if rtgTokIsKind(p, i, kind) && rtgTokCharIs(p, i+1, '(') {
			groupClose := rtgSkipBalanced(p, i+1, '(', ')')
			if groupClose >= end {
				return i, groupClose + 1, true
			}
		}
		i--
	}
	return 0, 0, false
}

func rtgParseTopDeclGroup(m *rtgMeta, p *rtgProgram, kind int, openTok int, endTok int) {
	if !rtgTokCharIs(p, openTok, '(') || endTok <= openTok+1 {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	groupEnd := endTok
	if rtgTokCharIs(p, endTok-1, ')') {
		groupEnd = endTok - 1
	}
	if kind == rtgTokConst {
		rtgParseConstDecls(m, p, openTok+1, groupEnd)
		return
	}
	j := openTok + 1
	for j < groupEnd {
		if rtgTokIsKind(p, j, rtgTokIdent) {
			entryEnd := rtgStatementLineEnd(p, j, groupEnd)
			rtgParseTopDeclEntry(m, p, kind, j, entryEnd)
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

func rtgHashRange(src []byte, start int, end int) int {
	h := 0
	for i := start; i < end; i++ {
		h = (h*33 + int(src[i])) & 2147483647
	}
	return h
}

func rtgMetaAppendGlobal(m *rtgMeta, sym rtgSymbolInfo) {
	index := len(m.globals)
	m.globals = append(m.globals, sym)
	if len(m.globalBuckets) == 0 {
		return
	}
	m.globalNext = append(m.globalNext, -1)
	hash := rtgHashRange(m.prog.src, sym.nameStart, sym.nameEnd)
	bucket := hash % len(m.globalBuckets)
	m.globalNext[index] = m.globalBuckets[bucket]
	m.globalBuckets[bucket] = index
}

func rtgFindMetaGlobalIndex(m *rtgMeta, nameStart int, nameEnd int, kind int) int {
	if len(m.globalBuckets) == 0 {
		return -1
	}
	hash := rtgHashRange(m.prog.src, nameStart, nameEnd)
	i := m.globalBuckets[hash%len(m.globalBuckets)]
	for i >= 0 {
		s := m.globals[i]
		if s.kind == kind && rtgBytesEqualRange(m.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = m.globalNext[i]
	}
	return -1
}

func rtgBuildFuncLookup(m *rtgMeta) {
	m.funcNext = make([]int, len(m.funcs))
	for i := 0; i < len(m.funcNext); i++ {
		m.funcNext[i] = -1
	}
	for i := 0; i < len(m.funcs); i++ {
		f := &m.funcs[i]
		hash := rtgHashRange(m.prog.src, f.nameStart, f.nameEnd)
		bucket := hash % len(m.funcBuckets)
		m.funcNext[i] = m.funcBuckets[bucket]
		m.funcBuckets[bucket] = i
	}
}

func rtgFindMetaFunction(m *rtgMeta, nameStart int, nameEnd int) int {
	if len(m.funcBuckets) == 0 {
		return -1
	}
	hash := rtgHashRange(m.prog.src, nameStart, nameEnd)
	i := m.funcBuckets[hash%len(m.funcBuckets)]
	for i >= 0 {
		fn := &m.funcs[i]
		if fn.receiverType == 0 && rtgBytesEqualRange(m.prog.src, fn.nameStart, fn.nameEnd, nameStart, nameEnd) {
			return i
		}
		i = m.funcNext[i]
	}
	return -1
}

func rtgResolveGlobalCallTypes(m *rtgMeta) {
	for i := 0; i < len(m.globals); i++ {
		global := &m.globals[i]
		if global.typ != 0 || global.initStart >= global.initEnd {
			continue
		}
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, m.prog, global.initStart, global.initEnd)
		if rootIndex < 0 {
			continue
		}
		root := &ep.exprs[rootIndex]
		if root.kind != rtgExprCall || root.left < 0 || root.left >= len(ep.exprs) {
			continue
		}
		callee := &ep.exprs[root.left]
		if callee.kind != rtgExprIdent {
			continue
		}
		fnIndex := rtgFindMetaFunction(m, callee.nameStart, callee.nameEnd)
		if fnIndex >= 0 {
			global.typ = m.funcs[fnIndex].resultType
		}
	}
}

func rtgInitBuiltinTypes(m *rtgMeta) {
	rtgAddBuiltinType(m, rtgTypeInvalid, 0)
	rtgAddBuiltinType(m, rtgTypeInt, rtgNativeIntSize)
	rtgAddBuiltinType(m, rtgTypeInt64, 8)
	rtgAddBuiltinType(m, rtgTypeByte, 1)
	rtgAddBuiltinType(m, rtgTypeBool, 1)
	rtgAddBuiltinType(m, rtgTypeString, rtgBackendStringValueSize)
	rtgAddBuiltinType(m, rtgTypeFloat64, 8)
	rtgAddBuiltinType(m, rtgTypeInt16, 2)
	rtgAddBuiltinType(m, rtgTypeInt32, 4)
}

func rtgAddBuiltinType(m *rtgMeta, kind int, size int) {
	m.types = append(m.types, rtgTypeInfo{kind: kind, size: size})
}

func rtgParseConstDecls(m *rtgMeta, p *rtgProgram, start int, end int) {
	prevTypeStart := 0
	prevTypeEnd := 0
	prevValues := rtgFixedIntScratch(8)
	iotaValue := 0
	j := start
	for j < end {
		if !rtgTokIsKind(p, j, rtgTokIdent) {
			j++
			continue
		}
		specEnd := rtgStatementLineEnd(p, j, end)
		if specEnd <= j {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		eq := rtgFindConstSpecEqual(p, j, specEnd)
		headEnd := specEnd
		if eq > j {
			headEnd = eq
		}
		names := rtgFixedIntScratch(4)
		k := j
		for k < headEnd {
			if !rtgTokIsKind(p, k, rtgTokIdent) {
				break
			}
			names = append(names, k)
			k++
			if rtgTokCharIs(p, k, ',') {
				k++
				continue
			}
			break
		}
		if len(names) == 0 {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		if eq > j {
			prevTypeStart = k
			prevTypeEnd = headEnd
			newValues, ok := rtgSplitTopLevelComma(p, eq+1, specEnd)
			if !ok {
				rtgMetaError(m, rtgDiagMetaConstDecl)
				return
			}
			prevValues = newValues
		}
		valueCount := len(prevValues) / 2
		if valueCount == 0 {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		if valueCount != len(names) {
			rtgMetaError(m, rtgDiagMetaConstDecl)
			return
		}
		typ := 0
		if prevTypeStart < prevTypeEnd {
			typeResult := rtgParseType(m, p, prevTypeStart, prevTypeEnd)
			typ = typeResult.typ
		}
		for i := 0; i < len(names); i++ {
			nameTok := names[i]
			name := rtgTokAt(p, nameTok)
			if rtgBytesEqualText(p.src, int(name.start), int(name.end), "_") {
				continue
			}
			initStart := prevValues[i*2]
			initEnd := prevValues[i*2+1]
			constType := typ
			if constType == 0 {
				constType = rtgInferTopLiteralType(m, p, initStart, initEnd)
			}
			if constType == 0 {
				constType = rtgTypeInt
			}
			var sym rtgSymbolInfo
			sym.nameStart = int(name.start)
			sym.nameEnd = int(name.end)
			sym.kind = rtgTokConst
			sym.typ = constType
			sym.initStart = initStart
			sym.initEnd = initEnd
			sym.iotaValue = iotaValue
			constResult := rtgEvalMetaConstExpr(m, p, initStart, initEnd, iotaValue)
			if constResult.ok {
				sym.constValue = constResult.value
				sym.constValueOK = 1
			}
			rtgMetaAppendGlobal(m, sym)
		}
		iotaValue++
		j = specEnd
	}
}

func rtgEvalMetaConstExpr(m *rtgMeta, p *rtgProgram, start int, end int, iotaValue int) rtgConstResult {
	var ep rtgExprParse
	if !rtgParseExpressionOK(&ep, p, start, end) {
		var r rtgConstResult
		return r
	}
	rootIndex := len(ep.exprs) - 1
	return rtgEvalMetaParsedConstExpr(m, p, &ep, rootIndex, iotaValue)
}

func rtgEvalMetaParsedConstExpr(m *rtgMeta, p *rtgProgram, ep *rtgExprParse, idx int, iotaValue int) rtgConstResult {
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		return rtgConstResultOk(rtgParseIntToken(p, e.tok))
	}
	if e.kind == rtgExprFloat {
		return rtgConstResultOk(rtgParseFloatTokenScaled(p, e.tok))
	}
	if e.kind == rtgExprChar {
		return rtgConstResultOk(rtgParseCharToken(p, e.tok))
	}
	if e.kind == rtgExprBool {
		return rtgConstResultOk(rtgBoolTokenValue(p, e.tok))
	}
	if e.kind == rtgExprIdent {
		if rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, "iota") {
			return rtgConstResultOk(iotaValue)
		}
		symIndex := rtgFindMetaGlobalIndex(m, e.nameStart, e.nameEnd, rtgTokConst)
		if symIndex >= 0 {
			s := &m.globals[symIndex]
			if s.constValueOK != 0 {
				return rtgConstResultOk(s.constValue)
			}
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprCall {
		if e.argCount == 1 {
			result := rtgEvalMetaParsedConstExpr(m, p, ep, ep.args[e.firstArg], iotaValue)
			if result.ok {
				callee := rtgExprIdentCode(p, ep, e.left)
				if callee == rtgIdentByte {
					result.value = rtgConvertConstInt(result.value, rtgTypeByte)
				}
				if callee == rtgIdentInt16 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt16)
				}
				if callee == rtgIdentInt32 {
					result.value = rtgConvertConstInt(result.value, rtgTypeInt32)
				}
			}
			return result
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprUnary {
		inner := rtgEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !inner.ok {
			var r rtgConstResult
			return r
		}
		if rtgTokCharIs(p, e.tok, '-') {
			return rtgConstResultOk(-inner.value)
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return rtgConstResultOk(inner.value)
		}
		if rtgTokCharIs(p, e.tok, '!') {
			if inner.value == 0 {
				return rtgConstResultOk(1)
			}
			return rtgConstResultOk(0)
		}
		var r rtgConstResult
		return r
	}
	if e.kind == rtgExprBinary {
		left := rtgEvalMetaParsedConstExpr(m, p, ep, e.left, iotaValue)
		if !left.ok {
			var r rtgConstResult
			return r
		}
		right := rtgEvalMetaParsedConstExpr(m, p, ep, e.right, iotaValue)
		if !right.ok {
			var r rtgConstResult
			return r
		}
		var g rtgLinearGen
		g.prog = p
		return rtgEvalConstBinary(&g, e.tok, left.value, right.value)
	}
	var r rtgConstResult
	return r
}

func rtgFindConstSpecEqual(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren > 0 {
				paren--
			}
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			if brack > 0 {
				brack--
			}
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace > 0 {
				brace--
			}
		} else if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, '=') {
			return i
		}
		i++
	}
	return start
}

func rtgParseTopDeclEntry(m *rtgMeta, p *rtgProgram, kind int, start int, end int) {
	if start >= end || !rtgTokIsKind(p, start, rtgTokIdent) {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	name := rtgTokAt(p, start)
	if kind == rtgTokVar {
		rtgParseVarDeclEntry(m, p, start, end)
		return
	}
	if kind == rtgTokType {
		typeStart := start + 1
		if rtgTokCharIs(p, typeStart, '=') {
			typeStart++
		}
		typeResult := rtgParseType(m, p, typeStart, end)
		if typeResult.typ == 0 || typeResult.next > end {
			rtgMetaError(m, rtgDiagMetaTopDecl)
			return
		}
		directNamedType := rtgTokIsKind(p, typeStart, rtgTokStruct) || rtgTokCharIs(p, typeStart, '*') || rtgTokCharIs(p, typeStart, '[')
		if directNamedType && (m.types[typeResult.typ].kind == rtgTypeStruct || m.types[typeResult.typ].kind == rtgTypePointer || m.types[typeResult.typ].kind == rtgTypeSlice) {
			m.types[typeResult.typ].nameStart = int(name.start)
			m.types[typeResult.typ].nameEnd = int(name.end)
		} else {
			size := rtgTypeSize(m, typeResult.typ)
			rtgAddType(m, rtgTypeNamed, typeResult.typ, 0, 0, size, int(name.start), int(name.end))
		}
		return
	}
	eq := start
	j := start + 1
	for j < end {
		if j >= 0 && j < rtgTokCount(p) {
			tok := rtgTokAt(p, j)
			if tok.kind == rtgTokOp && tok.end-tok.start == 1 && p.src[tok.start] == '=' {
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
		typeResult := rtgParseType(m, p, start+1, typeEnd)
		typ = typeResult.typ
	}
	if typ == 0 && initStart < initEnd {
		typ = rtgInferTopLiteralType(m, p, initStart, initEnd)
	}
	rtgMetaAppendGlobal(m, rtgSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
}

func rtgParseVarDeclEntry(m *rtgMeta, p *rtgProgram, start int, end int) {
	eq := rtgFindConstSpecEqual(p, start, end)
	headEnd := end
	if eq > start {
		headEnd = eq
	}
	names := rtgFixedIntScratch(4)
	k := start
	for k < headEnd {
		if !rtgTokIsKind(p, k, rtgTokIdent) {
			break
		}
		names = append(names, k)
		k++
		if rtgTokCharIs(p, k, ',') {
			k++
			continue
		}
		break
	}
	if len(names) == 0 {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	typ := 0
	if k < headEnd {
		typeResult := rtgParseType(m, p, k, headEnd)
		typ = typeResult.typ
	}
	var values []int
	if eq > start {
		valueBuf, ok := rtgSplitTopLevelComma(p, eq+1, end)
		if !ok {
			rtgMetaError(m, rtgDiagMetaTopDecl)
			return
		}
		values = valueBuf
	}
	valueCount := len(values) / 2
	if valueCount != 0 && valueCount != len(names) {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	for i := 0; i < len(names); i++ {
		nameTok := names[i]
		name := rtgTokAt(p, nameTok)
		if rtgBytesEqualText(p.src, int(name.start), int(name.end), "_") {
			continue
		}
		initStart := end
		initEnd := end
		symType := typ
		if valueCount != 0 {
			initStart = values[i*2]
			initEnd = values[i*2+1]
			if symType == 0 {
				symType = rtgInferTopLiteralType(m, p, initStart, initEnd)
			}
		}
		rtgMetaAppendGlobal(m, rtgSymbolInfo{nameStart: int(name.start), nameEnd: int(name.end), kind: rtgTokVar, typ: symType, initStart: initStart, initEnd: initEnd})
	}
}

func rtgInferTopLiteralType(m *rtgMeta, p *rtgProgram, start int, end int) int {
	if start+1 == end && rtgTokIsKind(p, start, rtgTokString) {
		return rtgTypeString
	}
	if start+1 == end && rtgTokIsKind(p, start, rtgTokFloat) {
		return rtgTypeFloat64
	}
	open := start
	depth := 0
	for open < end {
		if depth == 0 && rtgTokCharIs(p, open, '{') {
			typeResult := rtgParseType(m, p, start, open)
			if typeResult.typ != 0 {
				return typeResult.typ
			}
			return 0
		}
		if rtgTokCharIs(p, open, '(') || rtgTokCharIs(p, open, '[') {
			depth++
		} else if rtgTokCharIs(p, open, ')') || rtgTokCharIs(p, open, ']') {
			if depth > 0 {
				depth--
			}
		}
		open++
	}
	return 0
}

func rtgParseFuncInfo(m *rtgMeta, fnIndex int) {
	p := m.prog
	fn := p.funcs[fnIndex]
	nameStart := fn.nameStart
	nameEnd := fn.nameEnd
	nameTok := fn.nameTok
	if nameTok <= fn.startTok {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	lparen := rtgFindNextTokenText(p, nameTok+1, fn.bodyStart, '(')
	if lparen <= nameTok {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	rparen := rtgFindMatchingExprClose(p, lparen+1, fn.bodyStart, '(', ')')
	if rparen <= lparen {
		rtgMetaError(m, rtgDiagMetaFuncDecl)
		return
	}
	firstParam := len(m.params)
	paramCount := 0
	receiverType := 0
	if fn.receiverStart < fn.receiverEnd {
		beforeReceiver := len(m.params)
		rtgParseParamList(m, p, fn.receiverStart, fn.receiverEnd, &paramCount)
		if len(m.params) <= beforeReceiver {
			rtgMetaError(m, rtgDiagMetaFuncDecl)
			return
		}
		receiverType = m.params[beforeReceiver].typ
	}
	rtgParseParamList(m, p, lparen+1, rparen, &paramCount)
	resultType := 0
	if rparen+1 < fn.bodyStart {
		resultType = rtgParseFuncResultType(m, p, rparen+1, fn.bodyStart)
	}
	linkOK := 0
	linkDLLStart := 0
	linkDLLEnd := 0
	linkMethodStart := 0
	linkMethodEnd := 0
	if rtgHasLinkStaticDirectivePrefix(p, fn.nameStart) {
		linkStatic := rtgParseLinkStaticDirective(p, fn.nameStart)
		linkOK = linkStatic.ok
		linkDLLStart = linkStatic.dllStart
		linkDLLEnd = linkStatic.dllEnd
		linkMethodStart = linkStatic.methodStart
		linkMethodEnd = linkStatic.methodEnd
	}
	m.funcs = append(m.funcs, rtgFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd, linkStatic: linkOK, linkDLLStart: linkDLLStart, linkDLLEnd: linkDLLEnd, linkMethodStart: linkMethodStart, linkMethodEnd: linkMethodEnd})
}

type rtgLinkStaticDirective struct {
	ok          int
	dllStart    int
	dllEnd      int
	methodStart int
	methodEnd   int
}

func rtgHasLinkStaticDirectivePrefix(p *rtgProgram, pos int) bool {
	src := p.src
	if pos < 0 || pos > len(src) {
		return false
	}
	lineStart := pos
	for lineStart > 0 {
		prev := lineStart - 1
		c := src[prev]
		if c == '\n' {
			break
		}
		lineStart--
	}
	end := lineStart
	for end > 0 {
		prev := end - 1
		c := src[prev]
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
		c := src[prev]
		if c == '\n' {
			break
		}
		start--
	}
	for start < end {
		c := src[start]
		if c != ' ' && c != '\t' {
			break
		}
		start++
	}
	prefix := "// rtg:linkstatic "
	return rtgBytesHasText(src, start, end, prefix)
}

func rtgParseLinkStaticDirective(p *rtgProgram, pos int) rtgLinkStaticDirective {
	var d rtgLinkStaticDirective
	src := p.src
	if pos < 0 || pos > len(src) {
		return d
	}
	lineStart := pos
	for lineStart > 0 {
		prev := lineStart - 1
		c := src[prev]
		if c == '\n' {
			break
		}
		lineStart--
	}
	end := lineStart
	for end > 0 {
		prev := end - 1
		c := src[prev]
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
		c := src[prev]
		if c == '\n' {
			break
		}
		start--
	}
	for start < end && (src[start] == ' ' || src[start] == '\t') {
		start++
	}
	prefix := "// rtg:linkstatic "
	if !rtgBytesHasText(src, start, end, prefix) {
		return d
	}
	bodyStart := start + len(prefix)
	for bodyStart < end && (src[bodyStart] == ' ' || src[bodyStart] == '\t') {
		bodyStart++
	}
	comma := bodyStart
	for comma < end && src[comma] != ',' {
		comma++
	}
	if comma <= bodyStart || comma >= end {
		return d
	}
	dllEnd := comma
	for dllEnd > bodyStart && (src[dllEnd-1] == ' ' || src[dllEnd-1] == '\t') {
		dllEnd--
	}
	methodStart := comma + 1
	for methodStart < end && (src[methodStart] == ' ' || src[methodStart] == '\t') {
		methodStart++
	}
	methodEnd := end
	for methodEnd > methodStart && (src[methodEnd-1] == ' ' || src[methodEnd-1] == '\t') {
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

func rtgBytesHasText(src []byte, start int, end int, text string) bool {
	if start < 0 || end > len(src) || end-start < len(text) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if src[start+i] != text[i] {
			return false
		}
	}
	return true
}

func rtgParseFuncResultType(m *rtgMeta, p *rtgProgram, start int, end int) int {
	if rtgTokCharIs(p, start, '(') {
		closeTok := rtgFindMatchingExprClose(p, start+1, end, '(', ')')
		if closeTok > start && closeTok <= end {
			parts, ok := rtgSplitTopLevelComma(p, start+1, closeTok)
			if !ok {
				return 0
			}
			count := len(parts) / 2
			if count > 1 {
				return rtgBuildTupleType(m, p, parts)
			}
			if count == 1 {
				typeResult := rtgParseType(m, p, parts[0], parts[1])
				return typeResult.typ
			}
		}
	}
	typeResult := rtgParseType(m, p, start, end)
	return typeResult.typ
}

func rtgBuildTupleType(m *rtgMeta, p *rtgProgram, parts []int) int {
	firstField := len(m.fields)
	count := len(parts) / 2
	offset := 0
	for i := 0; i < count; i++ {
		typeStart := parts[i*2]
		typeEnd := parts[i*2+1]
		typeResult := rtgParseType(m, p, typeStart, typeEnd)
		if typeResult.typ == 0 {
			rtgMetaError(m, rtgDiagMetaResultType)
			return 0
		}
		offset = rtgAlignTo8(offset)
		m.fields = append(m.fields, rtgFieldInfo{typ: typeResult.typ, offset: offset})
		fieldSize := rtgTypeSize(m, typeResult.typ)
		if fieldSize < rtgBackendValueSlotSize {
			fieldSize = rtgBackendValueSlotSize
		}
		offset += fieldSize
	}
	size := rtgAlignTo8(offset)
	return rtgAddType(m, rtgTypeStruct, 0, firstField, count, size, 0, 0)
}

func rtgParseParamList(m *rtgMeta, p *rtgProgram, start int, end int, count *int) {
	i := start
	for i < end {
		for i < end && rtgTokCharIs(p, i, ',') {
			i++
		}
		if i >= end {
			return
		}
		if !rtgTokIsKind(p, i, rtgTokIdent) {
			rtgMetaError(m, rtgDiagMetaParamList)
			return
		}
		groupStart := i
		nameTok := i
		entryEnd := rtgFindExprBoundary(p, nameTok+1, end)
		for nameTok+1 == entryEnd {
			if entryEnd >= end {
				rtgMetaError(m, rtgDiagMetaParamList)
				return
			}
			nameTok = entryEnd + 1
			if !rtgTokIsKind(p, nameTok, rtgTokIdent) {
				rtgMetaError(m, rtgDiagMetaParamList)
				return
			}
			entryEnd = rtgFindExprBoundary(p, nameTok+1, end)
		}
		typeStart := nameTok + 1
		variadic := 0
		if rtgTokCharIs(p, typeStart, '.') && rtgTokCharIs(p, typeStart+1, '.') && rtgTokCharIs(p, typeStart+2, '.') {
			variadic = 1
		}
		typeResult := rtgParseType(m, p, typeStart, entryEnd)
		if typeResult.typ == 0 {
			rtgMetaError(m, rtgDiagMetaParamList)
			return
		}
		paramTok := groupStart
		for {
			name := rtgTokAt(p, paramTok)
			var paramInfo rtgSymbolInfo
			paramInfo.nameStart = int(name.start)
			paramInfo.nameEnd = int(name.end)
			paramInfo.typ = typeResult.typ
			paramInfo.initStart = variadic
			m.params = append(m.params, paramInfo)
			*count = *count + 1
			if paramTok == nameTok {
				break
			}
			paramEntryEnd := rtgFindExprBoundary(p, paramTok+1, end)
			paramTok = paramEntryEnd + 1
		}
		i = entryEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
}

func rtgParseType(m *rtgMeta, p *rtgProgram, start int, end int) rtgTypeResult {
	if start >= end {
		return rtgTypeResult{next: start}
	}
	if rtgTokCharIs(p, start, '.') && rtgTokCharIs(p, start+1, '.') && rtgTokCharIs(p, start+2, '.') {
		elem := rtgParseType(m, p, start+3, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, rtgBackendSliceValueSize, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokCharIs(p, start, '*') {
		elem := rtgParseType(m, p, start+1, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddPointerType(m, elem.typ, rtgPointerSpaceData)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokIdentIs(p, start, "map") && rtgTokCharIs(p, start+1, '[') {
		closeTok := rtgFindMatchingExprClose(p, start+2, end, '[', ']')
		if closeTok <= start+1 {
			return rtgTypeResult{next: start}
		}
		key := rtgParseType(m, p, start+2, closeTok)
		value := rtgParseType(m, p, closeTok+1, end)
		if key.typ == 0 || value.typ == 0 {
			return rtgTypeResult{next: start}
		}
		return rtgTypeResult{typ: rtgAddType(m, rtgTypeMap, value.typ, key.typ, 0, 24, 0, 0), next: value.next}
	}
	if rtgTokCharIs(p, start, '[') && rtgTokIsKind(p, start+1, rtgTokNumber) && rtgTokCharIs(p, start+2, ']') {
		count := rtgParseIntToken(p, start+1)
		elem := rtgParseType(m, p, start+3, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		return rtgTypeResult{typ: rtgAddType(m, rtgTypeArray, elem.typ, 0, count, count*rtgTypeSize(m, elem.typ), 0, 0), next: elem.next}
	}
	if rtgTokCharIs(p, start, '[') && rtgTokCharIs(p, start+1, ']') {
		elem := rtgParseType(m, p, start+2, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, rtgBackendSliceValueSize, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokIsKind(p, start, rtgTokStruct) && rtgTokCharIs(p, start+1, '{') {
		closeTok := rtgFindMatchingBrace(p, start+1, end)
		if closeTok <= start+1 {
			return rtgTypeResult{next: start}
		}
		firstField := len(m.fields)
		count := 0
		offset := 0
		i := start + 2
		for i < closeTok {
			if rtgTokIsKind(p, i, rtgTokIdent) {
				fieldNameTok := rtgTokAt(p, i)
				nameStart := int(fieldNameTok.start)
				nameEnd := int(fieldNameTok.end)
				lineEnd := rtgStatementLineEnd(p, i, closeTok)
				fieldType := rtgParseType(m, p, i+1, lineEnd)
				if fieldType.typ == 0 {
					return rtgTypeResult{next: start}
				}
				offset = rtgAlignTo8(offset)
				var fieldInfo rtgFieldInfo
				fieldInfo.nameStart = nameStart
				fieldInfo.nameEnd = nameEnd
				fieldInfo.typ = fieldType.typ
				fieldInfo.offset = offset
				m.fields = append(m.fields, fieldInfo)
				offset += rtgTypeSize(m, fieldType.typ)
				count++
				i = lineEnd
			} else {
				i++
			}
		}
		size := rtgAlignTo8(offset)
		typ := rtgAddType(m, rtgTypeStruct, 0, firstField, count, size, 0, 0)
		return rtgTypeResult{typ: typ, next: closeTok + 1}
	}
	if rtgTokIsKind(p, start, rtgTokIdent) {
		builtin := rtgBuiltinTypeFromToken(p, start)
		if builtin != 0 {
			return rtgTypeResult{typ: builtin, next: start + 1}
		}
		return rtgTypeResult{typ: rtgNamedTypeFromToken(m, p, start), next: start + 1}
	}
	return rtgTypeResult{next: start}
}

func rtgBuiltinTypeFromToken(p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	start := int(tok.start)
	end := int(tok.end)
	if rtgBytesEqualText(p.src, start, end, "int") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "int16") {
		return rtgTypeInt16
	}
	if rtgBytesEqualText(p.src, start, end, "int32") {
		return rtgTypeInt32
	}
	if rtgBytesEqualText(p.src, start, end, "int64") {
		return rtgTypeInt64
	}
	if rtgBytesEqualText(p.src, start, end, "uint") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "uint8") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "uint16") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "uint32") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "uint64") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, start, end, "byte") {
		return rtgTypeByte
	}
	if rtgBytesEqualText(p.src, start, end, "bool") {
		return rtgTypeBool
	}
	if rtgBytesEqualText(p.src, start, end, "string") {
		return rtgTypeString
	}
	if rtgBytesEqualText(p.src, start, end, "error") {
		return rtgTypeString
	}
	if rtgBytesEqualText(p.src, start, end, "float64") {
		return rtgTypeFloat64
	}
	return 0
}

func rtgNamedTypeFromToken(m *rtgMeta, p *rtgProgram, tokIndex int) int {
	tok := rtgTokAt(p, tokIndex)
	start := int(tok.start)
	end := int(tok.end)
	for i := 0; i < len(m.types); i++ {
		if m.types[i].nameEnd > m.types[i].nameStart && rtgBytesEqualRange(p.src, m.types[i].nameStart, m.types[i].nameEnd, start, end) {
			return i
		}
	}
	return rtgAddType(m, rtgTypeNamed, 0, 0, 0, rtgBackendValueSlotSize, start, end)
}

func rtgAddType(m *rtgMeta, kind int, elem int, first int, count int, size int, nameStart int, nameEnd int) int {
	m.types = append(m.types, rtgTypeInfo{kind: kind, elem: elem, first: first, count: count, size: size, nameStart: nameStart, nameEnd: nameEnd})
	index := len(m.types) - 1
	return index
}

func rtgAddPointerType(m *rtgMeta, elem int, addressSpace int) int {
	if addressSpace < rtgPointerSpaceData || addressSpace > rtgPointerSpaceGeneric {
		addressSpace = rtgPointerSpaceData
	}
	return rtgAddType(m, rtgTypePointer, elem, addressSpace, 0, rtgBackendValueSlotSize, 0, 0)
}

func rtgPointerAddressSpace(m *rtgMeta, typ int) int {
	t := rtgResolveType(m, typ)
	if t.kind != rtgTypePointer {
		return 0
	}
	if t.first < rtgPointerSpaceData || t.first > rtgPointerSpaceGeneric {
		return rtgPointerSpaceData
	}
	return t.first
}

func rtgFinalizeTypeLayouts(m *rtgMeta) {
	for i := 0; i < len(m.types); i++ {
		t := m.types[i]
		if t.kind != rtgTypeStruct {
			continue
		}
		offset := 0
		for j := 0; j < t.count; j++ {
			fieldIndex := t.first + j
			if fieldIndex < 0 || fieldIndex >= len(m.fields) {
				continue
			}
			offset = rtgAlignTo8(offset)
			m.fields[fieldIndex].offset = offset
			fieldSize := rtgTypeLayoutSize(m, m.fields[fieldIndex].typ, nil)
			if fieldSize < 1 {
				fieldSize = rtgBackendValueSlotSize
			}
			offset += fieldSize
		}
		m.types[i].size = rtgAlignTo8(offset)
	}
}

func rtgTypeLayoutSize(m *rtgMeta, typ int, seen []int) int {
	if typ < 0 || typ >= len(m.types) {
		return rtgBackendValueSlotSize
	}
	if rtgIntSliceContains(seen, typ) {
		return rtgBackendValueSlotSize
	}
	seen = rtgAppendIntCopy(seen, typ)
	t := m.types[typ]
	if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
		return rtgTypeLayoutSize(m, t.elem, seen)
	}
	if t.kind == rtgTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
		resolved := rtgFindResolvedNamedTypeIndex(m, typ)
		if resolved >= 0 {
			return rtgTypeLayoutSize(m, resolved, seen)
		}
	}
	if t.kind == rtgTypeStruct {
		offset := 0
		for i := 0; i < t.count; i++ {
			fieldIndex := t.first + i
			if fieldIndex < 0 || fieldIndex >= len(m.fields) {
				continue
			}
			offset = rtgAlignTo8(offset)
			fieldSize := rtgTypeLayoutSize(m, m.fields[fieldIndex].typ, seen)
			if fieldSize < 1 {
				fieldSize = rtgBackendValueSlotSize
			}
			offset += fieldSize
		}
		return rtgAlignTo8(offset)
	}
	if t.size > 0 {
		return t.size
	}
	return rtgBackendValueSlotSize
}

func rtgIntSliceContains(values []int, value int) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func rtgAppendIntCopy(values []int, value int) []int {
	out := make([]int, 0, len(values)+1)
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	out = append(out, value)
	return out
}

func rtgFindResolvedNamedTypeIndex(m *rtgMeta, typ int) int {
	if typ < 0 || typ >= len(m.types) {
		return -1
	}
	t := m.types[typ]
	for i := 0; i < len(m.types); i++ {
		if i == typ {
			continue
		}
		other := m.types[i]
		if other.nameEnd <= other.nameStart {
			continue
		}
		if !rtgBytesEqualRange(m.prog.src, other.nameStart, other.nameEnd, t.nameStart, t.nameEnd) {
			continue
		}
		if other.kind != rtgTypeNamed || other.elem > 0 {
			return i
		}
	}
	return -1
}

func rtgTypeSize(m *rtgMeta, typ int) int {
	t := rtgResolveType(m, typ)
	if t.size > 0 {
		return t.size
	}
	return rtgBackendValueSlotSize
}

func rtgResolveType(m *rtgMeta, typ int) rtgTypeInfo {
	if typ >= 0 && typ < len(m.types) {
		t := m.types[typ]
		if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return rtgResolveType(m, t.elem)
		}
		if t.kind == rtgTypeNamed && t.elem == 0 && t.nameEnd > t.nameStart {
			for i := 0; i < len(m.types); i++ {
				other := m.types[i]
				if i != typ && other.nameEnd > other.nameStart && rtgBytesEqualRange(m.prog.src, other.nameStart, other.nameEnd, t.nameStart, t.nameEnd) {
					if other.kind != rtgTypeNamed || other.elem > 0 {
						return rtgResolveType(m, i)
					}
				}
			}
		}
		return t
	}
	var t rtgTypeInfo
	return t
}

func rtgTypeIsSlice(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeSlice
}

func rtgTypeIsStringSlice(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	return rtgTypeIsString(m, t.elem)
}

func rtgTypeIsString(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeString
}

func rtgTypeIsInt(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeInt
}

func rtgTypeKindIsScalarInt(kind int) bool {
	if kind > rtgTypeInvalid && kind < rtgTypeString {
		return true
	}
	return kind == rtgTypeInt16 || kind == rtgTypeInt32
}

func rtgTypeKindIsScalarIntOrPointer(kind int) bool {
	if rtgTypeKindIsScalarInt(kind) {
		return true
	}
	return kind == rtgTypePointer
}

func rtgTypeKindIsScalarValue(kind int) bool {
	return rtgTypeKindIsScalarInt(kind) || kind == rtgTypeFloat64
}

func rtgScalarKindSize(kind int) int {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		return 1
	}
	if kind == rtgTypeInt {
		return rtgNativeIntSize
	}
	if kind >= rtgTypeInt16 {
		return (kind - 6) * 2
	}
	return rtgBackendValueSlotSize
}

func rtgAsmLoadPrimaryIndexTertiaryScalarOrPointer(a *rtgAsm, kind int) {
	if kind == rtgTypePointer {
		rtgAsmLoadQwordPrimaryIndexTertiary8(a)
		return
	}
	elemSize := rtgScalarKindSize(kind)
	rtgAsmLoadPrimaryIndexTertiarySize(a, elemSize)
}

func rtgTypeIsStruct(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	return t.kind == rtgTypeStruct
}

func rtgTypeIsTuple(m *rtgMeta, typ int) bool {
	t := rtgResolveType(m, typ)
	if t.kind != rtgTypeStruct || t.count <= 1 {
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

func rtgAlignTo8(v int) int {
	rem := v % rtgBackendValueSlotSize
	if rem == 0 {
		return v
	}
	return v + rtgBackendValueSlotSize - rem
}

func rtgFindTokenTextInRange(p *rtgProgram, start int, end int, text byte) int {
	i := start
	for i < end {
		if rtgTokCharIs(p, i, text) {
			return i
		}
		i++
	}
	return start - 1
}

func rtgBytesEqualRange(src []byte, aStart int, aEnd int, bStart int, bEnd int) bool {
	if aEnd-aStart != bEnd-bStart {
		return false
	}
	for i := 0; i < aEnd-aStart; i++ {
		if src[aStart+i] != src[bStart+i] {
			return false
		}
	}
	return true
}

// Shared scalar code generation.
func rtgBindFunctionParams(g *rtgLinearGen, fnIndex int) bool {
	meta := g.meta
	fn := &meta.funcs[fnIndex]
	callWord := 0
	if rtgTypeIsStruct(meta, fn.resultType) {
		callWord = rtgBackendHiddenResultWordCount
	}
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := rtgAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if rtgTypeIsSlice(meta, param.typ) {
			if !rtgStoreIncomingCallWord(g, callWord, offset) ||
				!rtgStoreIncomingCallWord(g, callWord+1, offset-rtgBackendValueSlotSize) ||
				!rtgStoreIncomingCallWord(g, callWord+2, offset-2*rtgBackendValueSlotSize) {
				return false
			}
			callWord += rtgBackendSliceWordCount
			continue
		}
		if rtgTypeIsString(meta, param.typ) {
			if !rtgStoreIncomingCallWord(g, callWord, offset) ||
				!rtgStoreIncomingCallWord(g, callWord+1, offset-rtgBackendValueSlotSize) {
				return false
			}
			callWord += rtgBackendStringWordCount
			continue
		}
		paramType := rtgResolveType(meta, param.typ)
		if paramType.kind == rtgTypeStruct || paramType.kind == rtgTypeArray {
			size := rtgTypeSize(meta, param.typ)
			wordSize := rtgBackendValueSlotSize
			if paramType.kind == rtgTypeArray {
				wordSize = rtgNativeIntSize
			}
			for at := 0; at < size; at += wordSize {
				if !rtgStoreIncomingCallWord(g, callWord, offset-at) {
					return false
				}
				callWord++
			}
			continue
		}
		if !rtgStoreIncomingCallWord(g, callWord, offset) {
			return false
		}
		callWord++
	}
	return true
}
func rtgAsmImmFits8Signed(imm int) bool {
	return imm >= -128 && imm <= 127
}
func rtgAsmLoadPrimaryIntToken(a *rtgAsm, p *rtgProgram, tokIndex int) {
	value := rtgParseIntToken(p, tokIndex)
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, value)
		return
	}
	needsMovabs := rtgIntTokenNeedsMovabs(p, tokIndex)
	if needsMovabs {
		rtgAsmPrimaryImm64(a, value)
		return
	}
	rtgAsmPrimaryImm(a, value)
}
func rtgIntTokenNeedsMovabs(p *rtgProgram, tokIndex int) bool {
	tok := rtgTokAt(p, tokIndex)
	start := int(tok.start)
	end := int(tok.end)
	if end-start > 2 && p.src[start] == '0' {
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
		c := p.src[start+i]
		if c > limit[i] {
			return true
		}
		if c < limit[i] {
			return false
		}
	}
	return false
}
func rtgAsmCopyPrimaryToSecondary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRdxRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRdxRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRdxRax(a)
		return
	}
	rtgAsmEmit16(a, 0x5a50)
}
func rtgAsmCopyPrimaryToTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRcxRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRcxRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRcxRax(a)
		return
	}
	rtgAsmEmit16(a, 0x5950)
}
func rtgAsmCopySecondaryToTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRcxRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRcxRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRcxRdx(a)
		return
	}
	rtgAsmEmit16(a, 0x5952)
}
func rtgAsmPushPrimary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPushRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPushRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPushRax(a)
		return
	}
	rtgAsmEmit8(a, 0x50)
}

func rtgAsmPushStackWord(a *rtgAsm, offset int) {
	if rtgTargetArch == rtgArchAmd64 || rtgTargetArch == rtgArch386 {
		rtgAsmEmit8(a, 0xff)
		if offset >= 0 && offset <= 128 {
			rtgAsmEmit8(a, 0x75)
			rtgAsmEmit8(a, -offset)
			return
		}
		rtgAsmEmit8(a, 0xb5)
		rtgAsmEmit32(a, -offset)
		return
	}
	rtgAsmLoadPrimaryStack(a, offset)
	rtgAsmPushPrimary(a)
}
func rtgAsmPushTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPushRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPushRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPushRcx(a)
		return
	}
	rtgAsmEmit8(a, 0x51)
}
func rtgAsmPushSecondary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPushRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPushRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPushRdx(a)
		return
	}
	rtgAsmEmit8(a, 0x52)
}
func rtgAsmPushImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPushImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPushImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPushImm(a, imm)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		rtgAsmEmit8(a, 0x68)
		rtgAsmEmit32(a, imm)
		return
	}
	rtgAsmPrimaryImm(a, imm)
	rtgAsmPushPrimary(a)
}
func rtgAsmPushSliceRegs(a *rtgAsm) {
	rtgAsmPushTertiary(a)
	rtgAsmPushSecondary(a)
	rtgAsmPushPrimary(a)
}
func rtgAsmPushStringRegs(a *rtgAsm) {
	rtgAsmPushSecondary(a)
	rtgAsmPushPrimary(a)
}
func rtgAsmPopPrimary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPopRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPopRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopRax(a)
		return
	}
	rtgAsmEmit8(a, 0x58)
}
func rtgAsmPopTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPopRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPopRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopRcx(a)
		return
	}
	rtgAsmEmit8(a, 0x59)
}
func rtgAsmPopSecondary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPopRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPopRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopRdx(a)
		return
	}
	rtgAsmEmit8(a, 0x5a)
}
func rtgAsmPopCallWord1(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPopRsi(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPopRsi(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopRsi(a)
		return
	}
	rtgAsmEmit8(a, 0x5e)
}
func rtgAsmStorePrimaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x45, 0x85)
}
func rtgAsmStoreSecondaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x55, 0x95)
}
func rtgAsmLoadPrimaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x45, 0x85)
}

func rtgAsmStoreStackImm(a *rtgAsm, offset int, value int) {
	rtgAsmPrimaryImm(a, value)
	rtgAsmStorePrimaryStack(a, offset)
}

func rtgAsmCopyStackSlot(a *rtgAsm, src int, dest int) {
	rtgAsmLoadPrimaryStack(a, src)
	rtgAsmStorePrimaryStack(a, dest)
}

func rtgAsmIncStack(a *rtgAsm, offset int) {
	rtgAsmLoadPrimaryStack(a, offset)
	rtgAsmIncPrimary(a)
	rtgAsmStorePrimaryStack(a, offset)
}

func rtgAsmJgeStackStack(a *rtgAsm, left int, right int, label int) {
	rtgAsmLoadPrimaryStack(a, left)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryStack(a, right)
	rtgAsmPopTertiary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9d)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, label)
}

func rtgAsmAddressPrimaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x45, 0x85)
}
func rtgAsmAddressCallWord0Stack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x7d, 0xbd)
}
func rtgAsmAddressCallWord1Stack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x75, 0xb5)
}
func rtgAsmLoadSecondaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x55, 0x95)
}
func rtgAsmSecondaryDisp(a *rtgAsm, disp int) {
	if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm {
		return
	}
	if disp == 0 {
		rtgAsmEmit8(a, 0x02)
		return
	}
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, 0x42, disp)
		return
	}
	rtgAsmEmit8(a, 0x82)
	rtgAsmEmit32(a, disp)
}
func rtgAsmLoadTertiaryStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x4d, 0x8d)
}
func rtgAsmStoreSliceStack(a *rtgAsm, offset int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreSliceStack(a, offset)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreSliceStack(a, offset)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreSliceStack(a, offset)
		return
	}
	rtgAsmStorePrimaryStack(a, offset)
	rtgAsmStoreSecondaryStack(a, offset-8)
	rtgAsmStackMem(a, offset-16, 0x8948, 0x4d, 0x8d)
}
func rtgAsmStoreStringBss(a *rtgAsm, offset int) {
	rtgAsmPushSecondary(a)
	rtgAsmStorePrimaryBss(a, offset)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryBss(a, offset+8)
}
func rtgAsmStoreSliceBss(a *rtgAsm, offset int) {
	rtgAsmPushTertiary(a)
	rtgAsmPushSecondary(a)
	rtgAsmStorePrimaryBss(a, offset)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryBss(a, offset+8)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryBss(a, offset+16)
}
func rtgAsmPopStoreStringMemSecondary(a *rtgAsm, disp int) {
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp+8)
}
func rtgAsmPopStoreSliceMemSecondary(a *rtgAsm, disp int) {
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp+8)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, disp+16)
}
func rtgAsmStoreByteMemSecondaryTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxMemRdxRcxSize(a, 1)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreAlMemRdxRcx1(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreAlMemRdxRcx1(a)
		return
	}
	rtgAsmEmit24(a, 0x0a0488)
}
func rtgAsmStorePrimaryMemSecondaryTertiarySize(a *rtgAsm, size int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreRaxMemRdxRcxSize(a, size)
		return
	}
	if size == 1 {
		rtgAsmStoreByteMemSecondaryTertiary(a)
		return
	}
	if size == 2 {
		rtgAsmEmit32(a, 0x4a048966)
		return
	}
	if size == 4 {
		rtgAsmEmit24(a, 0x8a0489)
		return
	}
	rtgAsmStorePrimaryMemSecondaryTertiary8(a)
}
func rtgAsmIncTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmIncRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmIncRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmIncRcx(a)
		return
	}
	rtgAsmEmit16(a, 0xc1ff)
}
func rtgAsmIncPrimary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmIncRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmIncRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmIncRax(a)
		return
	}
	rtgAsmEmit16(a, 0xc0ff)
}
func rtgAsmMulTertiaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmImulRcxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmImulRcxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmImulRcxImm(a, imm)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit3(a, 0x6b, 0xc9, imm)
		return
	}
	rtgAsmEmit16(a, 0xc969)
	rtgAsmEmit32(a, imm)
}
func rtgAsmRet(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmRet(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmRet(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmRet(a)
		return
	}
	rtgAsmEmit8(a, 0xc3)
}
func rtgAsmLeave(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLeave(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLeave(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLeave(a)
		return
	}
	rtgAsmEmit8(a, 0xc9)
}
func rtgAsmCallLabel(a *rtgAsm, label int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmCallLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmCallLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmCallLabel(a, label)
		return
	}
	rtgAsmEmit8(a, 0xe8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}
func rtgAsmJmpLabel(a *rtgAsm, label int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmJmpLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmJmpLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmJmpLabel(a, label)
		return
	}
	rtgAsmEmit8(a, 0xe9)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}
func rtgAsmJzLabel(a *rtgAsm, label int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmJzLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmJzLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmJzLabel(a, label)
		return
	}
	rtgAsmEmit16(a, 0x840f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}
func rtgAsmJnzLabel(a *rtgAsm, label int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmJnzLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmJnzLabel(a, label)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmJnzLabel(a, label)
		return
	}
	rtgAsmEmit16(a, 0x850f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

type rtgLocalInfo struct {
	nameStart  int
	nameEnd    int
	offset     int
	typ        int
	size       int
	constValue int
	constValid int
}

type rtgGlobalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
}

type rtgSliceLocation struct {
	offset int
	typ    int
	expr   int
	mem    bool
	deref  bool
	param  bool
	global bool
	ok     bool
}

type rtgLinearGen struct {
	prog               *rtgProgram
	meta               *rtgMeta
	asm                rtgAsm
	funcLabels         []int
	funcReachable      []bool
	funcQueue          []int
	currentFunc        int
	returnStruct       int
	locals             []rtgLocalInfo
	localCount         int
	stackUsed          int
	globals            []rtgGlobalInfo
	gotoLabels         []rtgGlobalInfo
	breakLabels        []int
	continueLabels     []int
	breakDepth         int
	continueDepth      int
	streqLabel         int
	streqEmitted       bool
	append8Label       int
	append8Emitted     bool
	append64Label      int
	append64Emitted    bool
	appendAddrLabel    int
	appendAddrEmitted  bool
	appendBytesLabel   int
	appendBytesEmitted bool
	copyWordsLabel     int
	copyWordsEmitted   bool
	arenaAllocLabel    int
	arenaAllocEmitted  bool
	makeZeroLabel      int
	makeZeroEmitted    bool
	stringHeapOff      int
	stringHeapEndOff   int
	stringHeapDataOff  int
	stringHeapReady    int
	winReadLabel       int
	winReadEmitted     bool
	winWriteLabel      int
	winWriteEmitted    bool
	printIntLabel      int
	printIntEmitted    bool
	printIntBufferOff  int
	darwinEntryOff     int
	lastRangeReturns   bool
	scopeBase          int
	constEvalIota      int
	constEvalIotaValid int
	fixedTargetValue   int
	fixedTargetState   int
	fixedPrunedReturns bool
}

func rtgAddStringData(g *rtgLinearGen, msg []byte) int {
	msgOff := len(g.asm.data)
	for i := 0; i < len(msg); i++ {
		g.asm.data = append(g.asm.data, msg[i])
	}
	g.asm.data = append(g.asm.data, 0)
	return msgOff
}

func rtgFunctionLocalCap(fn *rtgFuncDecl) int {
	localCap := 16
	if fn.bodyEnd-fn.bodyStart > 512 {
		localCap = 32
	}
	return localCap
}

func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	stmtData := make([]int, rtgStmtWordCount)
	rtgBodyStmtData = stmtData
	prog := g.meta.prog
	bp.prog = prog
	bp.stmtCount = 0
	bp.ok = true
	i := start
	lastKind := 0
	for bp.ok && i < end {
		if i < 0 || i >= rtgTokCount(prog) {
			return true
		}
		if rtgTokCharIs(prog, i, ';') {
			i++
			continue
		}
		if rtgTokIsKind(prog, i, rtgTokEOF) {
			return true
		}
		if rtgTokCharIs(prog, i, '}') {
			return true
		}
		rtgBodyStmtData = stmtData
		bp.stmtCount = 0
		next := rtgParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || bp.stmtCount != 1 {
			return false
		}
		stmt := rtgBodyStmtAt(&bp, 0)
		if !bp.ok {
			return false
		}
		lastKind = stmt.kind
		i = next
		if !rtgEmitLinearStmt(g, &stmt) {
			return false
		}
		if g.fixedPrunedReturns {
			g.fixedPrunedReturns = false
			return true
		}
	}
	g.lastRangeReturns = lastKind == rtgStmtReturn
	if !bp.ok {
		return false
	}
	return true
}
func rtgEmitScopedRange(g *rtgLinearGen, start int, end int) bool {
	oldLocalCount := g.localCount
	oldScopeBase := g.scopeBase
	oldStackUsed := g.stackUsed
	g.scopeBase = oldLocalCount
	ok := rtgEmitLinearRange(g, start, end)
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	g.stackUsed = oldStackUsed
	return ok
}
func rtgEmitLinearStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.kind == rtgStmtExpr {
		if rtgEmitLinearPrintStmt(g, stmt) {
			return true
		}
		if rtgEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		root := &ep.exprs[rootIndex]
		if root.kind != rtgExprCall {
			return false
		}
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtVar || stmt.kind == rtgStmtShort || stmt.kind == rtgStmtAssign {
		if !rtgEmitLinearAssign(g, stmt) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtReturn {
		if stmt.exprStart == stmt.exprEnd {
			rtgAsmPrimaryImm(a, 0)
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		resultType := g.meta.funcs[g.currentFunc].resultType
		if rtgTypeIsTuple(g.meta, resultType) {
			if !rtgEmitTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
				return false
			}
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		if rtgTypeIsStruct(g.meta, resultType) {
			if !rtgEmitStructReturnExpr(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsSlice(g.meta, resultType) {
			if !rtgEmitSliceReturnValueRegs(g, &ep, rootIndex, resultType) {
				return false
			}
		} else if rtgTypeIsString(g.meta, resultType) {
			if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else {
			resultResolved := rtgResolveType(g.meta, resultType)
			if !rtgEmitScalarExprForKind(g, &ep, rootIndex, resultResolved.kind) {
				return false
			}
		}
		rtgAsmLeave(a)
		rtgAsmRet(a)
		return true
	}
	if stmt.kind == rtgStmtIf {
		return rtgEmitLinearIf(g, stmt)
	}
	if stmt.kind == rtgStmtFor {
		return rtgEmitLinearFor(g, stmt)
	}
	if stmt.kind == rtgStmtSwitch {
		return rtgEmitLinearSwitch(g, stmt)
	}
	if stmt.kind == rtgStmtBlock {
		if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtGoto {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmJmpLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtLabel {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmMarkLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtBreak {
		if g.breakDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.breakLabels[g.breakDepth-1])
		return true
	}
	if stmt.kind == rtgStmtContinue {
		if g.continueDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.continueLabels[g.continueDepth-1])
		return true
	}
	return false
}
func rtgFindOrCreateGotoLabel(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.gotoLabels); i++ {
		info := g.gotoLabels[i]
		if rtgBytesEqualRange(g.prog.src, info.nameStart, info.nameEnd, nameStart, nameEnd) {
			return info.offset
		}
	}
	label := rtgAsmNewLabel(&g.asm)
	g.gotoLabels = append(g.gotoLabels, rtgGlobalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: label})
	return label
}
func rtgLoadCompilerFixedTarget(g *rtgLinearGen) {
	if g.fixedTargetState != 0 {
		return
	}
	g.fixedTargetState = -1
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if !rtgBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "rtgCompilerFixedTarget") {
			continue
		}
		g.fixedTargetState = 1
		if s.initStart >= s.initEnd {
			return
		}
		r := rtgEvalMetaConstExpr(g.meta, g.prog, s.initStart, s.initEnd, 0)
		if r.ok {
			g.fixedTargetValue = r.value
			return
		}
	}
}

func rtgFindCompilerFixedTarget(g *rtgLinearGen) int {
	rtgLoadCompilerFixedTarget(g)
	return g.fixedTargetValue
}
func rtgCompilerFixedTargetKnown(g *rtgLinearGen) bool {
	rtgLoadCompilerFixedTarget(g)
	return g.fixedTargetState == 1
}
func rtgExprRangeMayUseFixedTarget(p *rtgProgram, start int, end int) bool {
	for i := start; i < end; i++ {
		if !rtgTokIsKind(p, i, rtgTokIdent) {
			continue
		}
		nameStart := rtgTokStart(p, i)
		nameEnd := rtgTokEnd(p, i)
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgCompilerFixedTarget") {
			return true
		}
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgTargetArch") {
			return true
		}
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgTargetOS") {
			return true
		}
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgCurrentTarget") {
			return true
		}
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgTargetIsWindows") {
			return true
		}
		if rtgBytesEqualText(p.src, nameStart, nameEnd, "rtgTargetIsDarwin") {
			return true
		}
	}
	return false
}
func rtgFixedTargetArch(target int) int {
	if rtgCompilerFixedTarget != 0 {
		if rtgCompilerFixedTarget == rtgTargetWindows386 || rtgCompilerFixedTarget == rtgTargetLinux386 {
			return rtgArch386
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxAarch64 || rtgCompilerFixedTarget == rtgTargetDarwinArm64 {
			return rtgArchAarch64
		}
		if rtgCompilerFixedTarget == rtgTargetLinuxArm {
			return rtgArchArm
		}
		if rtgCompilerFixedTarget == rtgTargetWasiWasm32 {
			return rtgArchWasm32
		}
		return rtgArchAmd64
	}
	if target == rtgTargetLinux386 || target == rtgTargetWindows386 {
		return rtgArch386
	}
	if target == rtgTargetLinuxAarch64 || target == rtgTargetDarwinArm64 {
		return rtgArchAarch64
	}
	if target == rtgTargetLinuxArm {
		return rtgArchArm
	}
	if target == rtgTargetWasiWasm32 {
		return rtgArchWasm32
	}
	if target == rtgTargetLinuxAmd64 || target == rtgTargetWindowsAmd64 {
		return rtgArchAmd64
	}
	return 0
}
func rtgFixedTargetOS(target int) int {
	if rtgCompilerFixedTarget != 0 {
		if rtgCompilerFixedTarget == rtgTargetWindowsAmd64 || rtgCompilerFixedTarget == rtgTargetWindows386 {
			return rtgOSWindows
		}
		if rtgCompilerFixedTarget == rtgTargetDarwinArm64 {
			return rtgOSDarwin
		}
		if rtgCompilerFixedTarget == rtgTargetWasiWasm32 {
			return rtgOSWasi
		}
		return rtgOSLinux
	}
	if target == rtgTargetWindowsAmd64 || target == rtgTargetWindows386 {
		return rtgOSWindows
	}
	if target == rtgTargetDarwinArm64 {
		return rtgOSDarwin
	}
	if target == rtgTargetWasiWasm32 {
		return rtgOSWasi
	}
	if target != 0 {
		return rtgOSLinux
	}
	return 0
}
func rtgEvalFixedTargetInt(g *rtgLinearGen, ep *rtgExprParse, idx int, fixedTarget int, fixedTargetKnown bool) (int, bool) {
	if idx < 0 {
		return 0, false
	}
	if idx >= len(ep.exprs) {
		return 0, false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		return rtgParseIntToken(g.prog, e.tok), true
	}
	if e.kind == rtgExprChar {
		return rtgParseCharToken(g.prog, e.tok), true
	}
	if e.kind == rtgExprBool {
		return rtgBoolTokenValue(g.prog, e.tok), true
	}
	if e.kind == rtgExprIdent {
		if fixedTarget != 0 && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "rtgTargetArch") {
			return rtgFixedTargetArch(fixedTarget), true
		}
		if fixedTarget != 0 && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "rtgTargetOS") {
			return rtgFixedTargetOS(fixedTarget), true
		}
		if fixedTarget != 0 && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "rtgCurrentTarget") {
			return fixedTarget, true
		}
		if fixedTargetKnown && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "rtgCompilerFixedTarget") {
			return fixedTarget, true
		}
		value := rtgFindSmallConstByName(g, e.nameStart, e.nameEnd)
		if value >= -128 {
			return value, true
		}
	}
	return 0, false
}
func rtgEvalFixedTargetBool(g *rtgLinearGen, ep *rtgExprParse, idx int, fixedTarget int, fixedTargetKnown bool) int {
	if !fixedTargetKnown && fixedTarget == 0 {
		return -1
	}
	if idx < 0 {
		return -1
	}
	if idx >= len(ep.exprs) {
		return -1
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprBool {
		return rtgBoolTokenValue(g.prog, e.tok)
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '!') {
		inner := rtgEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
		if inner == 0 {
			return 1
		}
		if inner == 1 {
			return 0
		}
		return -1
	}
	if e.kind == rtgExprCall && e.argCount == 0 && rtgExprIsIdentText(g.prog, ep, e.left, "rtgTargetIsWindows") {
		if fixedTarget == 0 {
			return -1
		}
		if rtgFixedTargetOS(fixedTarget) == rtgOSWindows {
			return 1
		}
		return 0
	}
	if e.kind == rtgExprCall && e.argCount == 0 && rtgExprIsIdentText(g.prog, ep, e.left, "rtgTargetIsDarwin") {
		if fixedTarget == 0 {
			return -1
		}
		if rtgFixedTargetOS(fixedTarget) == rtgOSDarwin {
			return 1
		}
		return 0
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(g.prog, e.tok, '&', '&') {
			left := rtgEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
			if left == 0 {
				return 0
			}
			right := rtgEvalFixedTargetBool(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if left == 1 && right == 1 {
				return 1
			}
			if right == 0 {
				return 0
			}
			return -1
		}
		if rtgTok2Is(g.prog, e.tok, '|', '|') {
			left := rtgEvalFixedTargetBool(g, ep, e.left, fixedTarget, fixedTargetKnown)
			if left == 1 {
				return 1
			}
			right := rtgEvalFixedTargetBool(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if left == 0 && right == 0 {
				return 0
			}
			if right == 1 {
				return 1
			}
			return -1
		}
		if rtgTok2Is(g.prog, e.tok, '=', '=') || rtgTok2Is(g.prog, e.tok, '!', '=') {
			left, leftOK := rtgEvalFixedTargetInt(g, ep, e.left, fixedTarget, fixedTargetKnown)
			right, rightOK := rtgEvalFixedTargetInt(g, ep, e.right, fixedTarget, fixedTargetKnown)
			if !leftOK || !rightOK {
				return -1
			}
			eq := left == right
			if rtgTok2Is(g.prog, e.tok, '!', '=') {
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
func rtgEmitLinearElse(g *rtgLinearGen, stmt *rtgStmt) bool {
	p := g.prog
	if stmt.elseStart <= 0 {
		g.lastRangeReturns = false
		return true
	}
	if rtgTokIsKind(p, stmt.elseStart, rtgTokIf) && rtgTokIsKind(p, stmt.elseStart-1, rtgTokElse) {
		var nested rtgBodyParse
		oldStmtData := rtgBodyStmtData
		stmtData := make([]int, rtgStmtWordCount)
		rtgBodyStmtData = stmtData
		nested.prog = p
		nested.stmtCount = 0
		nested.ok = true
		next := rtgParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || nested.stmtCount != 1 {
			rtgBodyStmtData = oldStmtData
			return false
		}
		nestedStmt := rtgBodyStmtAt(&nested, 0)
		if !nested.ok {
			rtgBodyStmtData = oldStmtData
			return false
		}
		rtgBodyStmtData = oldStmtData
		return rtgEmitLinearStmt(g, &nestedStmt)
	}
	return rtgEmitScopedRange(g, stmt.elseStart, stmt.elseEnd)
}
func rtgEmitLinearIf(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	semi := rtgFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi >= stmt.exprStart {
		return rtgEmitLinearScopedControl(g, stmt, semi)
	}
	fixedValue := -1
	if rtgExprRangeMayUseFixedTarget(p, stmt.exprStart, stmt.exprEnd) {
		var fixedEp rtgExprParse
		if rtgParseExpressionOK(&fixedEp, p, stmt.exprStart, stmt.exprEnd) {
			fixedValue = rtgEvalFixedTargetBool(g, &fixedEp, len(fixedEp.exprs)-1, rtgFindCompilerFixedTarget(g), rtgCompilerFixedTargetKnown(g))
		}
	}
	if fixedValue >= 0 {
		ok := false
		if fixedValue == 1 {
			ok = rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd)
		} else {
			ok = rtgEmitLinearElse(g, stmt)
		}
		if !ok {
			return false
		}
		if g.lastRangeReturns {
			g.fixedPrunedReturns = true
		}
		return true
	}
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, p, stmt.exprStart, stmt.exprEnd)
	if rootIndex < 0 {
		return false
	}
	endLabel := rtgAsmNewLabel(a)
	elseLabel := endLabel
	if stmt.elseStart > 0 {
		elseLabel = rtgAsmNewLabel(a)
	}
	if !rtgEmitJumpIfFalse(g, &ep, rootIndex, elseLabel) {
		return false
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	thenReturns := g.lastRangeReturns
	if stmt.elseStart <= 0 {
		rtgAsmMarkLabel(a, endLabel)
		return true
	}
	if !thenReturns {
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, elseLabel)
	if !rtgEmitLinearElse(g, stmt) {
		return false
	}
	rtgAsmMarkLabel(a, endLabel)
	return true
}
func rtgEmitLinearFor(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	semi1 := rtgFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi1 >= stmt.exprStart {
		return rtgEmitLinearScopedControl(g, stmt, semi1)
	}
	rangeTok := rtgFindRangeToken(p, stmt.exprStart, stmt.exprEnd)
	if rangeTok >= stmt.exprStart {
		return rtgEmitLinearScopedControl(g, stmt, rangeTok)
	}
	startLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	rtgPushLoopLabels(g, endLabel, startLabel)
	rtgAsmMarkLabel(a, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, stmt.exprStart, stmt.exprEnd)
		if rootIndex < 0 {
			return false
		}
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	rtgPopLoopLabels(g)
	return true
}

func rtgPushLoopLabels(g *rtgLinearGen, breakLabel int, continueLabel int) {
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

func rtgPopLoopLabels(g *rtgLinearGen) {
	g.breakDepth--
	g.continueDepth--
}

func rtgFindRangeToken(p *rtgProgram, start int, end int) int {
	for i := start; i < end; i++ {
		if rtgTokIdentIs(p, i, "range") {
			return i
		}
	}
	return start - 1
}

func rtgEmitLinearScopedControl(g *rtgLinearGen, stmt *rtgStmt, split int) bool {
	oldLocalCount := g.localCount
	oldScopeBase := g.scopeBase
	g.scopeBase = oldLocalCount
	ok := false
	if stmt.kind == rtgStmtIf {
		if rtgEmitLinearSimpleRange(g, stmt.exprStart, split) {
			oldExprStart := stmt.exprStart
			stmt.exprStart = split + 1
			ok = rtgEmitLinearIf(g, stmt)
			stmt.exprStart = oldExprStart
		}
	} else if rtgTokCharIs(g.prog, split, ';') {
		ok = rtgEmitLinearClassicForScoped(g, stmt, split)
	} else {
		ok = rtgEmitLinearRangeForScoped(g, stmt, split)
	}
	g.localCount = oldLocalCount
	g.scopeBase = oldScopeBase
	return ok
}

func rtgEmitLinearRangeForScoped(g *rtgLinearGen, stmt *rtgStmt, rangeTok int) bool {
	p := g.prog
	a := &g.asm
	if rangeTok+1 >= stmt.exprEnd {
		return false
	}
	var source rtgExprParse
	sourceIndex := rtgParseExpressionRoot(&source, p, rangeTok+1, stmt.exprEnd)
	if sourceIndex < 0 {
		return false
	}
	sourceType := rtgInferParsedExprType(g, &source, sourceIndex)
	resolved := rtgResolveType(g.meta, sourceType)
	if resolved.kind != rtgTypeArray && resolved.kind != rtgTypeSlice && resolved.kind != rtgTypeMap {
		return false
	}
	sourceOffset := rtgAddUnnamedLocal(g, sourceType)
	if !rtgEmitExprToLocal(g, &source, sourceIndex, sourceOffset) {
		return false
	}
	indexOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStoreStackImm(a, indexOffset, 0)

	keyOffset := 0
	valueOffset := 0
	if rangeTok > stmt.exprStart {
		assignTok := rtgFindAssignmentToken(p, stmt.exprStart, rangeTok)
		if assignTok < stmt.exprStart || assignTok >= rangeTok {
			return false
		}
		targets, ok := rtgSplitTopLevelComma(p, stmt.exprStart, assignTok)
		if !ok || len(targets) < 2 || len(targets) > 4 {
			return false
		}
		short := rtgTok2Is(p, assignTok, ':', '=')
		keyType := rtgTypeInt
		if resolved.kind == rtgTypeMap {
			keyType = resolved.first
		}
		keyOffset = rtgRangeTargetOffset(g, targets[0], targets[1], keyType, short)
		if keyOffset < 0 {
			return false
		}
		if len(targets) == 4 {
			valueOffset = rtgRangeTargetOffset(g, targets[2], targets[3], resolved.elem, short)
			if valueOffset < 0 {
				return false
			}
		}
	}

	startLabel := rtgAsmNewLabel(a)
	continueLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	rtgPushLoopLabels(g, endLabel, continueLabel)
	rtgAsmMarkLabel(a, startLabel)
	rtgAsmLoadPrimaryStack(a, indexOffset)
	rtgAsmPushPrimary(a)
	if resolved.kind == rtgTypeArray {
		rtgAsmPrimaryImm(a, resolved.count)
	} else {
		rtgAsmLoadPrimaryStack(a, sourceOffset-8)
	}
	rtgAsmPopTertiary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9d)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, endLabel)
	if keyOffset > 0 && resolved.kind != rtgTypeMap {
		rtgAsmCopyStackSlot(a, indexOffset, keyOffset)
	}
	if resolved.kind == rtgTypeMap && (keyOffset > 0 || valueOffset > 0) {
		rtgAsmLoadTertiaryStack(a, indexOffset)
		rtgAsmMulTertiaryImm(a, rtgMapEntrySize)
		rtgAsmLoadSecondaryStack(a, sourceOffset)
		rtgAsmAddSecondaryTertiary(a)
		entryAddrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgAsmStoreSecondaryStack(a, entryAddrOffset)
		if keyOffset > 0 {
			rtgEmitCopyMemSecondaryToStack(g, keyOffset, rtgTypeSize(g.meta, resolved.first))
		}
		if valueOffset > 0 {
			rtgAsmLoadSecondaryStack(a, entryAddrOffset)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 16)
			rtgAsmStorePrimaryStack(a, valueOffset)
		}
	} else if valueOffset > 0 {
		rtgAsmLoadTertiaryStack(a, indexOffset)
		elemSize := rtgTypeSize(g.meta, resolved.elem)
		if elemSize != 1 {
			rtgAsmMulTertiaryImm(a, elemSize)
		}
		if resolved.kind == rtgTypeArray {
			rtgAsmAddressPrimaryStack(a, sourceOffset)
		} else {
			rtgAsmLoadPrimaryStack(a, sourceOffset)
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmAddSecondaryTertiary(a)
		rtgEmitCopyMemSecondaryToStack(g, valueOffset, elemSize)
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmMarkLabel(a, continueLabel)
	rtgAsmIncStack(a, indexOffset)
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	rtgPopLoopLabels(g)
	return true
}

func rtgRangeTargetOffset(g *rtgLinearGen, start int, end int, typ int, short bool) int {
	p := g.prog
	if end != start+1 || !rtgTokIsKind(p, start, rtgTokIdent) {
		return -1
	}
	nameStart := int(rtgTokStart(p, start))
	nameEnd := int(rtgTokEnd(p, start))
	if rtgBytesEqualText(p.src, nameStart, nameEnd, "_") {
		return 0
	}
	localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
	if short {
		localIndex = rtgFindLocalIndexInCurrentScope(g, nameStart, nameEnd)
		if localIndex < 0 {
			return rtgAddTypedLocal(g, nameStart, nameEnd, typ)
		}
	}
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}
func rtgEmitLinearSwitch(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.exprStart >= stmt.exprEnd {
		return false
	}
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, p, stmt.exprStart, stmt.exprEnd)
	if rootIndex < 0 {
		return false
	}
	switchType := rtgInferParsedExprType(g, &ep, rootIndex)
	stringSwitch := rtgTypeIsString(g.meta, switchType)
	valueOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOffset := 0
	if stringSwitch {
		lenOffset = rtgAddUnnamedLocal(g, rtgTypeInt)
		if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStorePrimaryStack(a, valueOffset)
		rtgAsmStoreSecondaryStack(a, lenOffset)
	} else {
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStorePrimaryStack(a, valueOffset)
	}

	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	clauseStarts := rtgFixedIntScratch(8)
	clauseLabels := rtgFixedIntScratch(8)
	defaultLabel := endLabel
	hasDefault := false
	i := stmt.bodyStart
	for i < stmt.bodyEnd {
		clause := rtgFindNextSwitchClause(p, i, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		label := rtgAsmNewLabel(a)
		clauseStarts = append(clauseStarts, clause)
		clauseLabels = append(clauseLabels, label)
		if rtgTokIsKind(p, clause, rtgTokDefault) {
			defaultLabel = label
			hasDefault = true
		}
		i = clause + 1
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		if rtgTokIsKind(p, clause, rtgTokCase) {
			if !rtgEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, clauseLabels[i]) {
				return false
			}
		}
	}
	if hasDefault {
		rtgAsmJmpLabel(a, defaultLabel)
	} else {
		rtgAsmJmpLabel(a, endLabel)
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := rtgFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		rtgAsmMarkLabel(a, clauseLabels[i])
		if !rtgEmitScopedRange(g, colon+1, bodyEnd) {
			return false
		}
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}
func rtgEmitSwitchCaseTests(g *rtgLinearGen, stmt *rtgStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, matchLabel int) bool {
	a := &g.asm
	p := g.prog
	colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
	if colon <= clause+1 {
		return false
	}
	i := clause + 1
	for i < colon {
		valueEnd := rtgFindExprBoundary(p, i, colon)
		if valueEnd <= i {
			return false
		}
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, i, valueEnd)
		if rootIndex < 0 {
			return false
		}
		if stringSwitch {
			if !rtgEmitSwitchStringCaseTest(g, valueOffset, lenOffset, &ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			rtgAsmLoadPrimaryStack(a, valueOffset)
			rtgAsmPushPrimary(a)
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			rtgAsmPopTertiary(a)
			rtgAsmCmpTertiaryPrimarySet(a, 0x94)
			rtgAsmCmpPrimaryImm8(a, 0)
			rtgAsmJnzLabel(a, matchLabel)
		}
		i = valueEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}
func rtgFindNextSwitchClause(p *rtgProgram, start int, end int) int {
	depth := 0
	i := start
	for i < end {
		if depth == 0 && (rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault)) {
			return i
		}
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			if depth > 0 {
				depth--
			}
		}
		i++
	}
	return end
}
func rtgFindSwitchClauseColon(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ':') {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return end
			}
			brace--
		}
		i++
	}
	return end
}
func rtgEmitLinearClassicForScoped(g *rtgLinearGen, stmt *rtgStmt, semi1 int) bool {
	a := &g.asm
	p := g.prog
	semi2 := rtgFindTokenTextInRange(p, semi1+1, stmt.exprEnd, ';')
	if semi2 <= semi1 {
		return false
	}
	if !rtgEmitLinearSimpleRange(g, stmt.exprStart, semi1) {
		return false
	}
	startLabel := rtgAsmNewLabel(a)
	postLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	rtgPushLoopLabels(g, endLabel, postLabel)
	rtgAsmMarkLabel(a, startLabel)
	if semi1+1 < semi2 {
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, semi1+1, semi2)
		if rootIndex < 0 {
			return false
		}
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmMarkLabel(a, postLabel)
	if !rtgEmitLinearSimpleRange(g, semi2+1, stmt.exprEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	rtgPopLoopLabels(g)
	return true
}
func rtgEmitLinearSimpleRange(g *rtgLinearGen, start int, end int) bool {
	p := g.prog
	if start >= end {
		return true
	}
	if rtgEmitLinearIncDec(g, start, end) {
		return true
	}
	assignTok := rtgFindAssignmentToken(p, start, end)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTok2Is(p, assignTok, ':', '=') {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = int(rtgTokStart(p, start))
			nameEnd = int(rtgTokEnd(p, start))
		}
		stmt := rtgStmt{kind: kind, startTok: start, endTok: end, exprStart: assignTok + 1, exprEnd: end, nameStart: nameStart, nameEnd: nameEnd}
		return rtgEmitLinearAssign(g, &stmt)
	}
	return false
}
func rtgEmitLinearIncDec(g *rtgLinearGen, start int, end int) bool {
	a := &g.asm
	p := g.prog
	if start+2 > end {
		return false
	}
	opTok := end - 1
	if !rtgTok2Is(p, opTok, '+', '+') && !rtgTok2Is(p, opTok, '-', '-') {
		return false
	}
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, p, start, opTok)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	inc := rtgTok2Is(p, opTok, '+', '+')
	if root.kind == rtgExprIdent {
		localOffset := rtgFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			rtgClearLocalConstAtOffset(g, localOffset)
			if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
				rtgAsmLoadPrimaryStack(a, localOffset)
				rtgAsmPushImm(a, 1)
				rtgAsmPopTertiary(a)
				if inc {
					rtgAsmAddPrimaryTertiary(a)
				} else {
					rtgAsmSubPrimaryTertiary(a)
				}
				rtgAsmStorePrimaryStack(a, localOffset)
				return true
			}
			rtgAsmEmit16(a, 0xff48)
			if localOffset >= 0 && localOffset <= 128 {
				if inc {
					rtgAsmEmit8(a, 0x45)
				} else {
					rtgAsmEmit8(a, 0x4d)
				}
				rtgAsmEmit8(a, -localOffset)
			} else {
				if inc {
					rtgAsmEmit8(a, 0x85)
				} else {
					rtgAsmEmit8(a, 0x8d)
				}
				rtgAsmEmit32(a, -localOffset)
			}
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
			rtgAsmLoadPrimaryBss(a, globalOffset)
			rtgAsmPushImm(a, 1)
			rtgAsmPopTertiary(a)
			if inc {
				rtgAsmAddPrimaryTertiary(a)
			} else {
				rtgAsmSubPrimaryTertiary(a)
			}
			rtgAsmStorePrimaryBss(a, globalOffset)
			return true
		}
		if inc {
			rtgAsmEmit24(a, 0x05ff48)
		} else {
			rtgAsmEmit24(a, 0x0dff48)
		}
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, globalOffset, rtgAbsBssReloc)
		return true
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressSecondary(g, &ep, rootIndex) {
			return false
		}
		if inc {
			rtgAsmIncMemSecondary(a)
		} else {
			rtgAsmDecMemSecondary(a)
		}
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressPrimary(g, &ep, rootIndex) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		if inc {
			rtgAsmIncMemSecondary(a)
		} else {
			rtgAsmDecMemSecondary(a)
		}
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		if inc {
			rtgAsmIncMemSecondary(a)
		} else {
			rtgAsmDecMemSecondary(a)
		}
		return true
	}
	return false
}
func rtgEmitJumpIfFalse(g *rtgLinearGen, ep *rtgExprParse, idx int, falseLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '&', '&') {
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			return rtgEmitJumpIfFalse(g, ep, e.right, falseLabel)
		}
		if rtgTok2Is(p, e.tok, '|', '|') {
			trueLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			if !rtgEmitJumpIfFalse(g, ep, e.right, falseLabel) {
				return false
			}
			rtgAsmMarkLabel(a, trueLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, falseLabel, false) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfTrue(g, ep, e.left, falseLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, falseLabel)
	return true
}
func rtgEmitJumpIfTrue(g *rtgLinearGen, ep *rtgExprParse, idx int, trueLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '|', '|') {
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			return rtgEmitJumpIfTrue(g, ep, e.right, trueLabel)
		}
		if rtgTok2Is(p, e.tok, '&', '&') {
			falseLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			if !rtgEmitJumpIfTrue(g, ep, e.right, trueLabel) {
				return false
			}
			rtgAsmMarkLabel(a, falseLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, trueLabel, true) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfFalse(g, ep, e.left, trueLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, trueLabel)
	return true
}
func rtgEmitCompareJumpOp(a *rtgAsm, c0 byte, c1 byte, label int, jumpIfTrue bool) {
	if rtgTargetArch == rtgArchWasm32 {
		cond := rtgWasm32CondEq
		if c0 == '=' {
			if jumpIfTrue {
				cond = rtgWasm32CondEq
			} else {
				cond = rtgWasm32CondNe
			}
		} else if c0 == '!' {
			if jumpIfTrue {
				cond = rtgWasm32CondNe
			} else {
				cond = rtgWasm32CondEq
			}
		} else if c0 == '<' {
			if c1 == '=' {
				if jumpIfTrue {
					cond = rtgWasm32CondLe
				} else {
					cond = rtgWasm32CondGt
				}
			} else {
				if jumpIfTrue {
					cond = rtgWasm32CondLt
				} else {
					cond = rtgWasm32CondGe
				}
			}
		} else if c1 == '=' {
			if jumpIfTrue {
				cond = rtgWasm32CondGe
			} else {
				cond = rtgWasm32CondLt
			}
		} else {
			if jumpIfTrue {
				cond = rtgWasm32CondGt
			} else {
				cond = rtgWasm32CondLe
			}
		}
		rtgWasm32EmitCondBranch(a, cond, label)
		return
	}
	if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm {
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
		if rtgTargetArch == rtgArchArm {
			rtgArmAsmBCondLabel(a, label, cond)
		} else {
			rtgAarch64AsmBCondLabel(a, label, cond)
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
	rtgAsmEmit16(a, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}
func rtgLinearInitGlobals(g *rtgLinearGen) bool {
	meta := g.meta
	a := &g.asm
	for i := 0; i < len(meta.globals); i++ {
		s := meta.globals[i]
		if s.kind != rtgTokVar {
			continue
		}
		off := g.asm.bssSize
		g.globals = append(g.globals, rtgGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		size := rtgTypeSize(meta, s.typ)
		if size < rtgBackendValueSlotSize {
			size = rtgBackendValueSlotSize
		}
		g.asm.bssSize += rtgAlignTo8(size)
		if rtgTypeIsInt(meta, s.typ) && rtgBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "rtgCompilerDefaultTarget") {
			rtgAsmPrimaryImm(a, rtgCurrentTarget)
			rtgAsmStorePrimaryBss(a, off)
			continue
		}
		if s.initStart < s.initEnd {
			var ep rtgExprParse
			rootIndex := rtgParseExpressionRoot(&ep, g.prog, s.initStart, s.initEnd)
			if rootIndex < 0 {
				return false
			}
			if rtgTypeIsString(meta, s.typ) {
				if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
					return false
				}
				rtgAsmPushSecondary(a)
				rtgAsmStorePrimaryBss(a, off)
				rtgAsmPopPrimary(a)
				rtgAsmStorePrimaryBss(a, off+8)
				continue
			}
			if rtgTypeIsSlice(meta, s.typ) {
				root := &ep.exprs[rootIndex]
				if root.kind != rtgExprComposite {
					return false
				}
				if !rtgEmitGlobalSliceLiteralRegs(g, &ep, rootIndex, s.typ) {
					return false
				}
				rtgAsmPushTertiary(a)
				rtgAsmPushSecondary(a)
				rtgAsmStorePrimaryBss(a, off)
				rtgAsmPopPrimary(a)
				rtgAsmStorePrimaryBss(a, off+8)
				rtgAsmPopPrimary(a)
				rtgAsmStorePrimaryBss(a, off+16)
				continue
			}
			if rtgTypeIsStruct(meta, s.typ) {
				if !rtgEmitGlobalStructInit(g, &ep, rootIndex, s.typ, off) {
					return false
				}
				continue
			}
			resolved := rtgResolveType(meta, s.typ)
			root := &ep.exprs[rootIndex]
			if resolved.kind == rtgTypePointer && root.kind == rtgExprUnary && rtgTokCharIs(g.prog, root.tok, '&') {
				inner := &ep.exprs[root.left]
				if inner.kind != rtgExprIdent {
					return false
				}
				targetOff := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if targetOff < 0 {
					return false
				}
				rtgAsmPrimaryBssAddr(a, targetOff)
				rtgAsmStorePrimaryBss(a, off)
				continue
			}
			if !rtgEmitScalarExprForKind(g, &ep, rootIndex, resolved.kind) {
				return false
			}
			rtgAsmStorePrimaryBss(a, off)
		} else if rtgTypeIsSlice(meta, s.typ) {
			rtgEmitInitEmptySliceBss(g, s.typ, off)
		}
	}
	return true
}
func rtgEmitGlobalStructInit(g *rtgLinearGen, ep *rtgExprParse, rootIndex int, typ int, off int) bool {
	root := &ep.exprs[rootIndex]
	if root.kind == rtgExprCall {
		return rtgEmitStructCallToBss(g, ep, rootIndex, typ, off)
	}
	if root.kind != rtgExprComposite {
		return false
	}
	for i := 0; i < root.argCount; i++ {
		field := ep.fields[root.firstArg+i]
		fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
		if fieldIndex < 0 {
			return false
		}
		fieldOffset := g.meta.fields[fieldIndex].offset
		fieldType := g.meta.fields[fieldIndex].typ
		if fieldType == 0 {
			return false
		}
		fieldResolved := rtgResolveType(g.meta, fieldType)
		if fieldResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmStoreStringBss(&g.asm, off+fieldOffset)
		} else if fieldResolved.kind == rtgTypeStruct {
			if !rtgEmitGlobalStructInit(g, ep, field.expr, fieldType, off+fieldOffset) {
				return false
			}
		} else if fieldResolved.kind == rtgTypeSlice {
			if !rtgEmitSliceLiteralRegs(g, ep, field.expr, fieldType) {
				return false
			}
			rtgAsmStoreSliceBss(&g.asm, off+fieldOffset)
		} else {
			constResult := rtgEvalConstExpr(g, ep, field.expr)
			if !constResult.ok {
				return false
			}
			source := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, field.expr))
			value := rtgConvertConstScalar(constResult.value, source.kind, fieldResolved.kind)
			rtgAsmPrimaryImm(&g.asm, value)
			rtgAsmStorePrimaryBss(&g.asm, off+fieldOffset)
		}
	}
	return true
}
func rtgEmitInitEmptySliceBss(g *rtgLinearGen, sliceType int, off int) {
	a := &g.asm
	t := rtgResolveType(g.meta, sliceType)
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmStorePrimaryBss(a, off)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmStorePrimaryBss(a, off+8)
	rtgAsmPrimaryImm(a, backingSize/elemSize)
	rtgAsmStorePrimaryBss(a, off+16)
}

func rtgEmitLinearAssign(g *rtgLinearGen, stmt *rtgStmt) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if (stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar)) && rtgTokIsKind(p, stmt.startTok+1, rtgTokIdent) {
		nameStart = int(rtgTokStart(p, stmt.startTok+1))
		nameEnd = int(rtgTokEnd(p, stmt.startTok+1))
	} else if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) {
		nameStart = int(rtgTokStart(p, stmt.startTok))
		nameEnd = int(rtgTokEnd(p, stmt.startTok))
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	compoundAssign := false
	if assignTok >= 0 && assignTok < rtgTokCount(p) {
		tok := rtgTokAt(p, assignTok)
		if tok.end-tok.start == 2 && p.src[tok.start+1] == '=' {
			c := p.src[tok.start]
			compoundAssign = c == '+' || c == '-' || c == '*' || c == '/' || c == '%'
		}
	}
	if assignTok > stmt.startTok && rtgEmitMultiAssign(g, stmt, assignTok) {
		return true
	}
	if assignTok > stmt.startTok && compoundAssign {
		var lhs rtgExprParse
		if rtgParseExpressionOK(&lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				var baseEp rtgExprParse
				baseIndex := rtgParseExpressionRoot(&baseEp, p, stmt.startTok, baseEnd)
				if baseIndex < 0 {
					return false
				}
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice || !rtgTypeKindIsScalarInt(elemType.kind) {
					return false
				}
				elemSize := rtgScalarKindSize(elemType.kind)
				indexOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				ptrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
					return false
				}
				rtgAsmStorePrimaryStack(a, indexOffset)
				if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
					return false
				}
				rtgAsmStorePrimaryStack(a, ptrOffset)
				rtgAsmLoadSecondaryStack(a, ptrOffset)
				rtgAsmLoadTertiaryStack(a, indexOffset)
				rtgAsmLoadPrimaryIndexTertiarySize(a, elemSize)
				rtgAsmPushPrimary(a)
				var rhs rtgExprParse
				rhsIndex := rtgParseExpressionRoot(&rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopTertiary(a)
				if !rtgEmitPrimaryTertiaryOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizePrimaryForKind(a, elemType.kind)
				rtgAsmLoadSecondaryStack(a, ptrOffset)
				rtgAsmLoadTertiaryStack(a, indexOffset)
				rtgAsmStorePrimaryMemSecondaryTertiarySize(a, elemSize)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressSecondary(g, &lhs, lhsIndex) {
					return false
				}
				lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
				lhsResolved := rtgResolveType(meta, lhsType)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				rtgAsmStoreSecondaryStack(a, addrOffset)
				var rhs rtgExprParse
				if !rtgParseExpressionOK(&rhs, p, assignTok+1, stmt.endTok) {
					return false
				}
				rtgAsmLoadSecondaryStack(a, addrOffset)
				rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, lhsSize)
				rtgAsmPushPrimary(a)
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopTertiary(a)
				if !rtgEmitPrimaryTertiaryOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizePrimaryForKind(a, lhsResolved.kind)
				rtgAsmLoadSecondaryStack(a, addrOffset)
				rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
		var lhs rtgExprParse
		if rtgParseExpressionOK(&lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				var baseEp rtgExprParse
				baseIndex := rtgParseExpressionRoot(&baseEp, p, stmt.startTok, baseEnd)
				if baseIndex < 0 {
					return false
				}
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				mapIndex := sliceType.kind == rtgTypeMap && rtgTypeKindIsScalarIntOrPointer(elemType.kind)
				arrayIndex := sliceType.kind == rtgTypeArray && rtgTypeKindIsScalarInt(elemType.kind)
				if mapIndex || arrayIndex {
					if mapIndex {
						if !rtgEmitMapEntryAddress(g, &lhs, lhsIndex, true) {
							return false
						}
					} else if !rtgEmitIndexAddressPrimary(g, &lhs, lhsIndex) {
						return false
					}
					addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					rtgAsmStorePrimaryStack(a, addrOffset)
					var rhs rtgExprParse
					if !rtgParseExpressionOK(&rhs, p, assignTok+1, stmt.endTok) || !rtgEmitIntExpr(g, &rhs, len(rhs.exprs)-1) {
						return false
					}
					rtgAsmNormalizePrimaryForKind(a, elemType.kind)
					rtgAsmLoadSecondaryStack(a, addrOffset)
					disp := 0
					if mapIndex {
						disp = 16
					}
					rtgAsmStorePrimaryMemSecondaryDispSize(a, disp, rtgScalarKindSize(elemType.kind))
					return true
				}
				if sliceType.kind != rtgTypeSlice {
					return false
				}
				scalarElem := rtgTypeKindIsScalarInt(elemType.kind)
				indexOffset := 0
				if scalarElem {
					indexOffset = rtgAddUnnamedLocal(g, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStorePrimaryStack(a, indexOffset)
				}
				var rhs rtgExprParse
				rhsIndex := rtgParseExpressionRoot(&rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if scalarElem {
					if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmNormalizePrimaryForKind(a, elemType.kind)
					rtgAsmPushPrimary(a)
					rtgAsmLoadPrimaryStack(a, indexOffset)
					rtgAsmPushPrimary(a)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmCopyPrimaryToSecondary(a)
					rtgAsmPopTertiary(a)
					rtgAsmPopPrimary(a)
					elemSize := rtgScalarKindSize(elemType.kind)
					rtgAsmStorePrimaryMemSecondaryTertiarySize(a, elemSize)
					return true
				}
				if elemType.kind == rtgTypeString {
					indexOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					ptrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					indexEnd := rtgFindMatchingExprClose(p, baseEnd+1, assignTok, '[', ']')
					if indexEnd <= baseEnd+1 {
						return false
					}
					var indexEp rtgExprParse
					indexRoot := rtgParseExpressionRoot(&indexEp, p, baseEnd+1, indexEnd)
					if indexRoot < 0 {
						return false
					}
					if !rtgEmitIntExpr(g, &indexEp, indexRoot) {
						return false
					}
					rtgAsmStorePrimaryStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStorePrimaryStack(a, ptrOffset)
					var stringRhs rtgExprParse
					stringRhsIndex := rtgParseExpressionRoot(&stringRhs, p, assignTok+1, stmt.endTok)
					if stringRhsIndex < 0 {
						return false
					}
					if !rtgEmitStringValueRegs(g, &stringRhs, stringRhsIndex) {
						return false
					}
					rtgAsmPushStringRegs(a)
					rtgAsmLoadPrimaryStack(a, ptrOffset)
					rtgAsmLoadTertiaryStack(a, indexOffset)
					rtgAsmShlTertiaryImm(a, 4)
					rtgAsmCopyPrimaryToSecondary(a)
					rtgAsmAddSecondaryTertiary(a)
					rtgAsmPopStoreStringMemSecondary(a, 0)
					return true
				}
				if elemType.kind == rtgTypeStruct {
					indexOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					ptrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					destOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					indexEnd := rtgFindMatchingExprClose(p, baseEnd+1, assignTok, '[', ']')
					if indexEnd <= baseEnd+1 {
						return false
					}
					var indexEp rtgExprParse
					indexRoot := rtgParseExpressionRoot(&indexEp, p, baseEnd+1, indexEnd)
					if indexRoot < 0 {
						return false
					}
					if !rtgEmitIntExpr(g, &indexEp, indexRoot) {
						return false
					}
					rtgAsmStorePrimaryStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStorePrimaryStack(a, ptrOffset)
					rtgAsmLoadSecondaryStack(a, ptrOffset)
					rtgAsmLoadTertiaryStack(a, indexOffset)
					elemSize := rtgTypeSize(meta, sliceType.elem)
					rtgAsmMulTertiaryImm(a, elemSize)
					rtgAsmAddSecondaryTertiary(a)
					rtgAsmStoreSecondaryStack(a, destOffset)
					var structRhs rtgExprParse
					structRhsIndex := rtgParseExpressionRoot(&structRhs, p, assignTok+1, stmt.endTok)
					if structRhsIndex < 0 {
						return false
					}
					structRhsRoot := structRhs.exprs[structRhsIndex]
					if structRhsRoot.kind == rtgExprIndex {
						sourceType := rtgInferParsedExprType(g, &structRhs, structRhsIndex)
						if !rtgTypeIsStruct(meta, sourceType) {
							return false
						}
						if rtgTypeSize(meta, sourceType) != elemSize {
							return false
						}
						srcOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
						tempOffset := rtgAddUnnamedLocal(g, sliceType.elem)
						if !rtgEmitIndexAddressPrimary(g, &structRhs, structRhsIndex) {
							return false
						}
						rtgAsmStorePrimaryStack(a, srcOffset)
						rtgAsmLoadSecondaryStack(a, srcOffset)
						rtgEmitCopyMemSecondaryToStack(g, tempOffset, elemSize)
						rtgAsmLoadSecondaryStack(a, destOffset)
						rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
						return true
					}
					if !rtgEmitCompositeFieldToMem(g, &rhs, rhsIndex, sliceType.elem, destOffset, 0) {
						return false
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsSlice(meta, lhsType) {
				if !rtgEmitSelectorAddressSecondary(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				rtgAsmStoreSecondaryStack(a, addrOffset)
				var rhs rtgExprParse
				if !rtgParseExpressionOK(&rhs, p, assignTok+1, stmt.endTok) {
					return false
				}
				if rtgEmitAppendAssignGeneral(g, stmt, &rhs, assignTok) {
					return true
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitSliceValueRegs(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPushSliceRegs(a)
				rtgAsmLoadSecondaryStack(a, addrOffset)
				rtgAsmPopStoreSliceMemSecondary(a, 0)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsString(meta, lhsType) {
				if !rtgEmitSelectorAddressSecondary(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				rtgAsmStoreSecondaryStack(a, addrOffset)
				var rhs rtgExprParse
				rhsIndex := rtgParseExpressionRoot(&rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if !rtgEmitStringValueRegs(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPushStringRegs(a)
				rtgAsmLoadSecondaryStack(a, addrOffset)
				rtgAsmPopStoreStringMemSecondary(a, 0)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsStruct(meta, lhsType) {
				if !rtgEmitSelectorAddressSecondary(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
				rtgAsmStoreSecondaryStack(a, addrOffset)
				var rhs rtgExprParse
				rhsIndex := rtgParseExpressionRoot(&rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				size := rtgTypeSize(meta, lhsType)
				tempOffset := rtgAddUnnamedLocal(g, lhsType)
				if !rtgEmitTypedAssign(g, &rhs, rhsIndex, tempOffset) {
					return false
				}
				rtgAsmLoadSecondaryStack(a, addrOffset)
				rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, size)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressSecondary(g, &lhs, lhsIndex) {
					return false
				}
				rtgAsmPushSecondary(a)
				var rhs rtgExprParse
				rhsIndex := rtgParseExpressionRoot(&rhs, p, assignTok+1, stmt.endTok)
				if rhsIndex < 0 {
					return false
				}
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				lhsResolved := rtgResolveType(meta, lhsType)
				rtgAsmNormalizePrimaryForKind(a, lhsResolved.kind)
				rtgAsmPopSecondary(a)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && compoundAssign {
			var left rtgExprParse
			leftIndex := rtgParseExpressionRoot(&left, p, stmt.startTok+1, assignTok)
			if leftIndex < 0 {
				return false
			}
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushPrimary(a)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, targetSize)
			rtgAsmPushPrimary(a)
			var right rtgExprParse
			rightIndex := rtgParseExpressionRoot(&right, p, assignTok+1, stmt.endTok)
			if rightIndex < 0 {
				return false
			}
			if !rtgEmitScalarExprForKind(g, &right, rightIndex, targetKind) {
				return false
			}
			rtgAsmPopTertiary(a)
			if !rtgEmitPrimaryTertiaryOp(g, assignTok) {
				return false
			}
			rtgAsmNormalizePrimaryForKind(a, targetKind)
			rtgAsmPopSecondary(a)
			rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, targetSize)
			return true
		}
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
			var left rtgExprParse
			leftIndex := rtgParseExpressionRoot(&left, p, stmt.startTok+1, assignTok)
			if leftIndex < 0 {
				return false
			}
			leftType := rtgInferParsedExprType(g, &left, leftIndex)
			leftResolved := rtgResolveType(meta, leftType)
			if leftResolved.kind == rtgTypePointer {
				targetType := leftResolved.elem
				targetResolved := rtgResolveType(meta, targetType)
				if targetResolved.kind == rtgTypeSlice || targetResolved.kind == rtgTypeString || targetResolved.kind == rtgTypeStruct {
					if !rtgEmitIntExpr(g, &left, leftIndex) {
						return false
					}
					addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
					rtgAsmStorePrimaryStack(a, addrOffset)
					var right rtgExprParse
					rightIndex := rtgParseExpressionRoot(&right, p, assignTok+1, stmt.endTok)
					if rightIndex < 0 {
						return false
					}
					if targetResolved.kind == rtgTypeSlice {
						if !rtgEmitSliceValueRegs(g, &right, rightIndex) {
							return false
						}
						rtgAsmPushSliceRegs(a)
						rtgAsmLoadSecondaryStack(a, addrOffset)
						rtgAsmPopStoreSliceMemSecondary(a, 0)
						return true
					}
					if targetResolved.kind == rtgTypeString {
						if !rtgEmitStringValueRegs(g, &right, rightIndex) {
							return false
						}
						rtgAsmPushStringRegs(a)
						rtgAsmLoadSecondaryStack(a, addrOffset)
						rtgAsmPopStoreStringMemSecondary(a, 0)
						return true
					}
					size := rtgTypeSize(meta, targetType)
					tempOffset := rtgAddUnnamedLocal(g, targetType)
					if !rtgEmitTypedAssign(g, &right, rightIndex, tempOffset) {
						return false
					}
					rtgAsmLoadSecondaryStack(a, addrOffset)
					rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, size)
					return true
				}
			}
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushPrimary(a)
			var right rtgExprParse
			rightIndex := rtgParseExpressionRoot(&right, p, assignTok+1, stmt.endTok)
			if rightIndex < 0 {
				return false
			}
			if !rtgEmitScalarExprForKind(g, &right, rightIndex, targetKind) {
				return false
			}
			rtgAsmPopSecondary(a)
			rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, targetSize)
			return true
		}
		return false
	}
	if nameEnd == nameStart+1 && p.src[nameStart] == '_' {
		if assignTok <= stmt.startTok || !rtgTokCharIs(p, assignTok, '=') {
			return true
		}
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, assignTok+1, stmt.endTok)
		if rootIndex < 0 {
			return false
		}
		discardType := rtgInferParsedExprType(g, &ep, rootIndex)
		if discardType != 0 {
			discardOffset := rtgAddUnnamedLocal(g, discardType)
			if rtgEmitTypedAssign(g, &ep, rootIndex, discardOffset) {
				return true
			}
		}
		return rtgEmitIntExpr(g, &ep, rootIndex)
	}
	var ep rtgExprParse
	if assignTok > stmt.startTok {
		if !rtgParseExpressionOK(&ep, p, assignTok+1, stmt.endTok) {
			return false
		}
	}
	declaresLocal := stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) || stmt.kind == rtgStmtShort
	offset := rtgFindLocalOffset(g, nameStart, nameEnd)
	if declaresLocal {
		offset = -1
	}
	globalOffset := -1
	fieldStackOffset := -1
	fieldType := 0
	if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) && rtgTokCharIs(p, stmt.startTok+1, '.') && rtgTokIsKind(p, stmt.startTok+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, int(rtgTokStart(p, stmt.startTok)), int(rtgTokEnd(p, stmt.startTok)))
		if localIndex < 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, int(rtgTokStart(p, stmt.startTok+2)), int(rtgTokEnd(p, stmt.startTok+2)))
		if fieldOffset < 0 {
			return false
		}
		fieldType = rtgStructFieldType(g, g.locals[localIndex].typ, int(rtgTokStart(p, stmt.startTok+2)), int(rtgTokEnd(p, stmt.startTok+2)))
		if fieldType == 0 {
			return false
		}
		fieldStackOffset = g.locals[localIndex].offset - fieldOffset
		offset = fieldStackOffset
	}
	if offset < 0 {
		if stmt.kind == rtgStmtAssign && !rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
			globalOffset = rtgFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := rtgTypeInt
			if stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				if stmt.startTok+2 < typeEnd {
					typeResult := rtgParseType(meta, g.prog, stmt.startTok+2, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				} else if assignTok > stmt.startTok {
					inferredType := rtgInferParsedExprType(g, &ep, len(ep.exprs)-1)
					if inferredType != 0 {
						localType = inferredType
					}
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferParsedExprType(g, &ep, len(ep.exprs)-1)
				if assignTok+2 < stmt.endTok && rtgTokIsKind(p, assignTok+1, rtgTokIdent) && rtgTokCharIs(p, assignTok+2, '(') {
					fnIndex := -1
					for i := 0; i < len(g.meta.funcs); i++ {
						f := &g.meta.funcs[i]
						if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, int(rtgTokStart(p, assignTok+1)), int(rtgTokEnd(p, assignTok+1))) {
							fnIndex = i
						}
					}
					if fnIndex >= 0 {
						inferredType = meta.funcs[fnIndex].resultType
					}
				}
				if inferredType != 0 {
					localType = inferredType
				}
			}
			offset = rtgAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			rtgAsmPrimaryImm(a, 0)
			rtgAsmStorePrimaryBss(a, globalOffset)
		} else {
			rtgZeroLocalAtOffset(g, offset)
			localType := rtgLocalTypeAtOffset(g, offset)
			if declaresLocal && fieldStackOffset < 0 && rtgLocalConstTrackable(g, localType, nameStart, nameEnd, stmt.endTok) {
				rtgSetLocalConstAtOffset(g, offset, 0)
			} else {
				rtgClearLocalConstAtOffset(g, offset)
			}
		}
		return true
	}
	rootIndex := len(ep.exprs) - 1
	targetType := rtgTypeInt
	if globalOffset >= 0 {
		targetType = rtgFindGlobalType(g, nameStart, nameEnd)
	} else if fieldStackOffset >= 0 {
		targetType = fieldType
	} else {
		targetType = rtgLocalTypeAtOffset(g, offset)
	}
	targetResolved := rtgResolveType(meta, targetType)
	trackLocalConst := globalOffset < 0 && fieldStackOffset < 0 && declaresLocal && rtgLocalConstTrackable(g, targetType, nameStart, nameEnd, stmt.endTok)
	localConst := rtgConstResult{}
	if trackLocalConst {
		localConst = rtgEvalConstExpr(g, &ep, rootIndex)
	}
	if globalOffset < 0 && fieldStackOffset < 0 && !declaresLocal {
		rtgClearLocalConstAtOffset(g, offset)
	}
	if stmt.kind == rtgStmtShort {
		root := &ep.exprs[rootIndex]
		if root.kind == rtgExprCall && root.argCount == 2 && rtgExprIdentCode(p, &ep, root.left) == rtgIdentAppend {
			if !rtgEmitSliceValueRegs(g, &ep, ep.args[root.firstArg]) {
				return false
			}
			rtgAsmStoreSliceStack(a, offset)
		}
	}
	if rtgEmitAppendAssignGeneral(g, stmt, &ep, assignTok) {
		if globalOffset < 0 && fieldStackOffset < 0 {
			rtgClearLocalConstAtOffset(g, offset)
		}
		return true
	}
	if compoundAssign {
		if globalOffset >= 0 {
			rtgAsmLoadPrimaryBss(a, globalOffset)
		} else {
			rtgAsmLoadPrimaryStack(a, offset)
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitScalarExprForKind(g, &ep, rootIndex, targetResolved.kind) {
			return false
		}
		rtgAsmPopTertiary(a)
		if !rtgEmitPrimaryTertiaryOp(g, assignTok) {
			return false
		}
		rtgAsmNormalizePrimaryForKind(a, targetResolved.kind)
		if globalOffset >= 0 {
			rtgAsmStorePrimaryBss(a, globalOffset)
		} else {
			rtgAsmStorePrimaryStack(a, offset)
			if fieldStackOffset < 0 {
				rtgClearLocalConstAtOffset(g, offset)
			}
		}
		return true
	}
	if globalOffset >= 0 && rtgTypeIsString(meta, targetType) {
		if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreStringBss(a, globalOffset)
		return true
	}
	if globalOffset >= 0 && rtgTypeIsSlice(meta, targetType) {
		if !rtgEmitSliceValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreSliceBss(a, globalOffset)
		return true
	}
	if globalOffset >= 0 && rtgTypeIsStruct(meta, targetType) {
		tempOffset := rtgAddUnnamedLocal(g, targetType)
		if !rtgEmitTypedAssign(g, &ep, rootIndex, tempOffset) {
			return false
		}
		size := rtgTypeSize(meta, targetType)
		rtgEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if globalOffset < 0 && rtgEmitTypedAssign(g, &ep, rootIndex, offset) {
		if fieldStackOffset < 0 {
			if trackLocalConst && localConst.ok {
				rtgSetLocalConstAtOffset(g, offset, localConst.value)
			} else {
				rtgClearLocalConstAtOffset(g, offset)
			}
		}
		return true
	}
	if !rtgEmitScalarExprForKind(g, &ep, rootIndex, targetResolved.kind) {
		return false
	}
	if globalOffset >= 0 {
		rtgAsmStorePrimaryBss(a, globalOffset)
	} else {
		rtgAsmStorePrimaryStack(a, offset)
		if fieldStackOffset < 0 {
			if trackLocalConst && localConst.ok {
				rtgSetLocalConstAtOffset(g, offset, localConst.value)
			} else {
				rtgClearLocalConstAtOffset(g, offset)
			}
		}
	}
	return true
}
func rtgEmitMultiAssign(g *rtgLinearGen, stmt *rtgStmt, assignTok int) bool {
	p := g.prog
	lhsStart := stmt.startTok
	if stmt.kind == rtgStmtVar && rtgTokIsKind(p, lhsStart, rtgTokVar) {
		lhsStart++
	}
	lhs, ok := rtgSplitTopLevelComma(p, lhsStart, assignTok)
	if !ok {
		return false
	}
	rhs, ok := rtgSplitTopLevelComma(p, assignTok+1, stmt.endTok)
	if !ok {
		return false
	}
	lhsCount := len(lhs) / 2
	rhsCount := len(rhs) / 2
	if lhsCount <= 1 && rhsCount <= 1 {
		return false
	}
	if lhsCount > 1 && rhsCount == 1 {
		if rtgEmitTupleCallAssign(g, stmt.kind, lhs, lhsCount, rhs[0], rhs[1]) {
			return true
		}
	}
	if lhsCount != rhsCount {
		return false
	}
	for i := 0; i < lhsCount; i++ {
		rtgClearLocalConstAssignTarget(g, stmt.kind, lhs[i*2], lhs[i*2+1])
	}
	tempOffsets := rtgFixedIntScratch(4)
	tempTypes := rtgFixedIntScratch(4)
	for i := 0; i < rhsCount; i++ {
		rhsStart := rhs[i*2]
		rhsEnd := rhs[i*2+1]
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, p, rhsStart, rhsEnd)
		if rootIndex < 0 {
			return false
		}
		typ := rtgInferParsedExprType(g, &ep, rootIndex)
		if typ == 0 {
			typ = rtgTypeInt
		}
		offset := rtgAddUnnamedLocal(g, typ)
		if !rtgEmitExprToLocal(g, &ep, rootIndex, offset) {
			return false
		}
		tempOffsets = append(tempOffsets, offset)
		tempTypes = append(tempTypes, typ)
	}
	for i := 0; i < lhsCount; i++ {
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, stmt.kind, lhsStart, lhsEnd, tempOffsets[i], tempTypes[i]) {
			return false
		}
	}
	return true
}

func rtgClearLocalConstAssignTarget(g *rtgLinearGen, kind int, targetStart int, targetEnd int) {
	p := g.prog
	var ep rtgExprParse
	if !rtgParseExpressionOK(&ep, p, targetStart, targetEnd) {
		return
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprIdent {
		return
	}
	if root.nameEnd == root.nameStart+1 && p.src[root.nameStart] == '_' {
		return
	}
	localIndex := rtgFindLocalIndex(g, root.nameStart, root.nameEnd)
	if kind == rtgStmtShort {
		localIndex = rtgFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
	}
	if localIndex >= 0 {
		rtgClearLocalConstAtOffset(g, g.locals[localIndex].offset)
	}
}

func rtgEmitTupleCallAssign(g *rtgLinearGen, kind int, lhs []int, lhsCount int, rhsStart int, rhsEnd int) bool {
	p := g.prog
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, p, rhsStart, rhsEnd)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	if root.kind != rtgExprCall {
		return false
	}
	fnIndex := rtgFuncInfoFromCall(g, &ep, root.left)
	if fnIndex < 0 {
		return false
	}
	resultType := g.meta.funcs[fnIndex].resultType
	if !rtgTypeIsTuple(g.meta, resultType) {
		return false
	}
	tuple := rtgResolveType(g.meta, resultType)
	if tuple.count != lhsCount {
		return false
	}
	offset := rtgAddUnnamedLocal(g, resultType)
	if !rtgEmitStructCallToLocal(g, &ep, rootIndex, resultType, offset) {
		return false
	}
	for i := 0; i < lhsCount; i++ {
		field := g.meta.fields[tuple.first+i]
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, kind, lhsStart, lhsEnd, offset-field.offset, field.typ) {
			return false
		}
	}
	return true
}
func rtgEmitExprToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	if rtgEmitTypedAssign(g, ep, idx, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmStorePrimaryStack(&g.asm, offset)
	return true
}
func rtgEmitTempToTarget(g *rtgLinearGen, kind int, targetStart int, targetEnd int, tempOffset int, tempType int) bool {
	p := g.prog
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, p, targetStart, targetEnd)
	if rootIndex < 0 {
		return false
	}
	root := &ep.exprs[rootIndex]
	size := rtgTypeSize(g.meta, tempType)
	if size < rtgBackendValueSlotSize {
		size = rtgBackendValueSlotSize
	}
	if root.kind == rtgExprIdent {
		if root.nameEnd == root.nameStart+1 && p.src[root.nameStart] == '_' {
			return true
		}
		localIndex := rtgFindLocalIndex(g, root.nameStart, root.nameEnd)
		if kind == rtgStmtShort || kind == rtgStmtVar {
			if kind == rtgStmtVar {
				localIndex = -1
			} else {
				localIndex = rtgFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
			}
			if localIndex < 0 {
				offset := rtgAddTypedLocal(g, root.nameStart, root.nameEnd, tempType)
				rtgEmitCopyStackToStack(g, tempOffset, offset, size)
				rtgClearLocalConstAtOffset(g, offset)
				return true
			}
		}
		if localIndex >= 0 {
			rtgEmitCopyStackToStack(g, tempOffset, g.locals[localIndex].offset, size)
			rtgClearLocalConstAtOffset(g, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		rtgEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if kind == rtgStmtShort || kind == rtgStmtVar {
		return false
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressSecondary(g, &ep, rootIndex) {
			return false
		}
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < rtgBackendValueSlotSize {
			targetSize = rtgBackendValueSlotSize
		}
		rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprIndex {
		baseType := rtgResolveType(g.meta, rtgInferParsedExprType(g, &ep, root.left))
		if baseType.kind == rtgTypeMap {
			if !rtgEmitMapEntryAddress(g, &ep, rootIndex, true) {
				return false
			}
			rtgAsmCopyPrimaryToSecondary(&g.asm)
			targetType := rtgResolveType(g.meta, baseType.elem)
			rtgAsmLoadPrimaryStack(&g.asm, tempOffset)
			rtgAsmStorePrimaryMemSecondaryDispSize(&g.asm, 16, rtgScalarKindSize(targetType.kind))
			return true
		}
		if !rtgEmitIndexAddressPrimary(g, &ep, rootIndex) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < rtgBackendValueSlotSize {
			targetSize = rtgBackendValueSlotSize
		}
		rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < rtgBackendValueSlotSize {
			targetSize = rtgBackendValueSlotSize
		}
		rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, targetSize)
		return true
	}
	return false
}
func rtgEmitCopyStackToBss(g *rtgLinearGen, srcOffset int, bssOffset int, size int) {
	if size < rtgBackendValueSlotSize {
		size = rtgBackendValueSlotSize
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadPrimaryStack(&g.asm, srcOffset-at)
		rtgAsmStorePrimaryBss(&g.asm, bssOffset+at)
	}
}
func rtgFindLocalIndexInCurrentScope(g *rtgLinearGen, nameStart int, nameEnd int) int {
	start := g.scopeBase
	if start < 0 {
		start = 0
	}
	for i := g.localCount - 1; i >= start; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgSetLocalConstAtOffset(g *rtgLinearGen, offset int, value int) {
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValue = value
			g.locals[i].constValid = 1
			return
		}
	}
}

func rtgClearLocalConstAtOffset(g *rtgLinearGen, offset int) {
	for i := g.localCount - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValid = 0
			return
		}
	}
}

func rtgLocalConstTrackable(g *rtgLinearGen, typ int, nameStart int, nameEnd int, afterTok int) bool {
	if rtgCompilerFixedTarget != 0 {
		return false
	}
	resolved := rtgResolveType(g.meta, typ)
	if !rtgTypeKindIsScalarInt(resolved.kind) {
		return false
	}
	return !rtgLocalNameWrittenAfter(g, nameStart, nameEnd, afterTok)
}

func rtgLocalNameWrittenAfter(g *rtgLinearGen, nameStart int, nameEnd int, afterTok int) bool {
	if nameEnd <= nameStart {
		return true
	}
	p := g.prog
	end := rtgTokCount(p)
	if g.currentFunc >= 0 && g.currentFunc < len(g.meta.funcs) {
		end = g.meta.funcs[g.currentFunc].bodyEnd
	}
	i := afterTok
	if i < 0 {
		i = 0
	}
	for i < end {
		if rtgTokIsKind(p, i, rtgTokIdent) && rtgBytesEqualRange(p.src, int(rtgTokStart(p, i)), int(rtgTokEnd(p, i)), nameStart, nameEnd) {
			if rtgTokCharIs(p, i-1, '&') {
				return true
			}
			if rtgTok2Is(p, i+1, '+', '+') || rtgTok2Is(p, i+1, '-', '-') {
				return true
			}
			lineEnd := rtgStatementLineEnd(p, i, end)
			assignTok := rtgFindAssignmentToken(p, i, lineEnd)
			if assignTok > i {
				return true
			}
		}
		i++
	}
	return false
}

func rtgSplitTopLevelComma(p *rtgProgram, start int, end int) ([]int, bool) {
	var ranges []int
	if rtgCompilerFixedTarget != 0 {
		ranges = make([]int, 0, 16)
	}
	partStart := start
	depth := 0
	i := start
	data := p.toks.data
	for i < end {
		base := i << 3
		if base+5 < len(data) && int(data[base]) == rtgTokOp && data[base+4] == 1 {
			c := data[base+5]
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

func rtgEmitTupleReturn(g *rtgLinearGen, start int, end int) bool {
	resultType := g.meta.funcs[g.currentFunc].resultType
	tuple := rtgResolveType(g.meta, resultType)
	parts, ok := rtgSplitTopLevelComma(g.prog, start, end)
	if !ok {
		return false
	}
	count := len(parts) / 2
	if count == tuple.count {
		for i := 0; i < count; i++ {
			partStart := parts[i*2]
			partEnd := parts[i*2+1]
			field := g.meta.fields[tuple.first+i]
			if !rtgEmitTupleReturnField(g, partStart, partEnd, field.typ, field.offset) {
				return false
			}
		}
		return true
	}
	if count == 1 {
		var ep rtgExprParse
		rootIndex := rtgParseExpressionRoot(&ep, g.prog, start, end)
		if rootIndex < 0 {
			return false
		}
		return rtgEmitStructReturnExpr(g, &ep, rootIndex)
	}
	return false
}
func rtgEmitTupleReturnField(g *rtgLinearGen, start int, end int, typ int, fieldOffset int) bool {
	var ep rtgExprParse
	rootIndex := rtgParseExpressionRoot(&ep, g.prog, start, end)
	if rootIndex < 0 {
		return false
	}
	if rtgTypeIsSlice(g.meta, typ) {
		if !rtgEmitSliceReturnValueRegs(g, &ep, rootIndex, typ) {
			return false
		}
		rtgAsmPushSliceRegs(&g.asm)
		rtgAsmLoadSecondaryStack(&g.asm, g.returnStruct)
		rtgAsmPopStoreSliceMemSecondary(&g.asm, fieldOffset)
		return true
	}
	tempOffset := rtgAddUnnamedLocal(g, typ)
	if !rtgEmitExprToLocal(g, &ep, rootIndex, tempOffset) {
		return false
	}
	size := rtgTypeSize(g.meta, typ)
	if size < rtgBackendValueSlotSize {
		size = rtgBackendValueSlotSize
	}
	rtgAsmLoadSecondaryStack(&g.asm, g.returnStruct)
	rtgEmitCopyStackToMemSecondary(g, tempOffset, fieldOffset, size)
	return true
}
func rtgInferParsedExprType(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	meta := g.meta
	e := ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return rtgTypeInt
	}
	if e.kind == rtgExprFloat {
		return rtgTypeFloat64
	}
	if e.kind == rtgExprString {
		return rtgTypeString
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].typ
		}
		symIndex := rtgFindMetaGlobalIndex(meta, e.nameStart, e.nameEnd, rtgTokVar)
		if symIndex < 0 {
			symIndex = rtgFindMetaGlobalIndex(meta, e.nameStart, e.nameEnd, rtgTokConst)
		}
		if symIndex >= 0 {
			return meta.globals[symIndex].typ
		}
		constStringTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constStringTok >= 0 {
			return rtgTypeString
		}
		return rtgTypeInt
	}
	if e.kind == rtgExprCall {
		if rtgExprIsErrorStringCall(g, ep, idx) {
			return rtgTypeString
		}
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee == rtgIdentAppend && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if callee == rtgIdentByteSlice && e.argCount == 1 {
			return rtgAddType(meta, rtgTypeSlice, rtgTypeByte, 0, 0, rtgBackendSliceValueSize, 0, 0)
		}
		if callee == rtgIdentString && e.argCount == 1 {
			return rtgTypeString
		}
		if callee == rtgIdentMake && e.argCount >= 1 {
			return rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
		}
		if callee == rtgIdentInt {
			return rtgTypeInt
		}
		if callee == rtgIdentInt16 {
			return rtgTypeInt16
		}
		if callee == rtgIdentInt32 {
			return rtgTypeInt32
		}
		if callee == rtgIdentInt64 {
			return rtgTypeInt64
		}
		if callee == rtgIdentByte {
			return rtgTypeByte
		}
		if callee == rtgIdentCap || callee == rtgIdentLen || callee == rtgIdentOpen || callee == rtgIdentClose || callee == rtgIdentRead || callee == rtgIdentWrite || callee == rtgIdentChmod || callee == rtgIdentCopy || callee == rtgIdentSyscall {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			return meta.funcs[fnIndex].resultType
		}
		if e.argCount == 1 {
			conversionType := rtgConversionTypeFromExpr(g, ep, e.left)
			if conversionType != 0 {
				return conversionType
			}
		}
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind == rtgTypeSlice || t.kind == rtgTypeArray || t.kind == rtgTypeMap {
			return t.elem
		}
		if t.kind == rtgTypeString {
			return rtgTypeByte
		}
	}
	if e.kind == rtgExprSlice {
		return rtgInferParsedExprType(g, ep, e.left)
	}
	if e.kind == rtgExprSelector {
		baseType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			t = rtgResolveType(meta, t.elem)
		}
		if t.kind == rtgTypeStruct {
			for i := 0; i < t.count; i++ {
				field := &meta.fields[t.first+i]
				if rtgBytesEqualRange(p.src, field.nameStart, field.nameEnd, e.nameStart, e.nameEnd) {
					return field.typ
				}
			}
		}
	}
	if e.kind == rtgExprComposite {
		return rtgTypeFromExpr(g, ep, idx)
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgInferParsedExprType(g, ep, e.left)
		}
		if rtgTokCharIs(p, e.tok, '&') {
			elemType := rtgInferParsedExprType(g, ep, e.left)
			if elemType == 0 {
				return 0
			}
			return rtgAddPointerType(meta, elemType, rtgPointerSpaceData)
		}
		if rtgTokCharIs(p, e.tok, '*') {
			innerType := rtgInferParsedExprType(g, ep, e.left)
			inner := rtgResolveType(meta, innerType)
			if inner.kind == rtgTypePointer {
				return inner.elem
			}
		}
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') {
			return rtgTypeInt
		}
		leftType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.left))
		rightType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.right))
		if rtgTokCharIs(p, e.tok, '+') && (leftType.kind == rtgTypeString || rightType.kind == rtgTypeString) {
			return rtgTypeString
		}
		if leftType.kind == rtgTypeFloat64 || rightType.kind == rtgTypeFloat64 {
			return rtgTypeFloat64
		}
	}
	return rtgTypeInt
}
func rtgPointerTargetKind(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	pointerType := rtgTypeInt
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			localInfo := &g.locals[localIndex]
			pointerType = localInfo.typ
		} else {
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalType != 0 {
				pointerType = globalType
			}
		}
	} else {
		pointerType = rtgInferParsedExprType(g, ep, idx)
	}
	pointerResolved := rtgResolveType(g.meta, pointerType)
	if pointerResolved.kind == rtgTypePointer {
		targetResolved := rtgResolveType(g.meta, pointerResolved.elem)
		return targetResolved.kind
	}
	return pointerResolved.kind
}
func rtgTypeFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	e := &ep.exprs[idx]
	if e.tok < 0 || e.tok >= rtgTokCount(p) {
		return 0
	}
	endTok := e.tok
	for endTok < rtgTokCount(p) && int(rtgTokEnd(p, endTok)) <= e.nameEnd {
		endTok++
	}
	typeResult := rtgParseType(g.meta, p, e.tok, endTok)
	return typeResult.typ
}
func rtgFindTypeByText(g *rtgLinearGen, name string) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualText(g.prog.src, t.nameStart, t.nameEnd, name) {
			return i
		}
	}
	return 0
}
func rtgFindTypeByRange(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualRange(g.prog.src, t.nameStart, t.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return 0
}
func rtgConversionTypeFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	if idx < 0 || idx >= len(ep.exprs) {
		return 0
	}
	callee := &ep.exprs[idx]
	if callee.kind != rtgExprIdent {
		return 0
	}
	builtin := rtgBuiltinTypeFromToken(g.prog, callee.tok)
	if builtin != 0 {
		return builtin
	}
	return rtgFindTypeByRange(g, callee.nameStart, callee.nameEnd)
}
func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < g.localCount; i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < g.localCount; i++ {
		t := rtgResolveType(g.meta, g.locals[i].typ)
		if t.kind == rtgTypeStruct {
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
func rtgEmitTypedAssign(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	meta := g.meta
	destType := rtgLocalTypeAtOffset(g, offset)
	e := &ep.exprs[idx]
	destResolved := rtgResolveType(meta, destType)
	if destResolved.kind == rtgTypeMap {
		return rtgEmitMapAssignToLocal(g, ep, idx, destType, offset)
	}
	if destResolved.kind == rtgTypeArray {
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != rtgTypeSize(meta, destType) {
				return false
			}
			rtgEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, rtgTypeSize(meta, destType))
			return true
		}
		return rtgEmitCompositeFieldToStack(g, ep, idx, destType, offset)
	}
	if destResolved.kind == rtgTypeStruct {
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != rtgTypeSize(meta, destType) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, size)
			return true
		}
		if e.kind == rtgExprCall {
			return rtgEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprIndex {
			return rtgEmitIndexedStructToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprSelector {
			fieldType := rtgInferParsedExprType(g, ep, idx)
			if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != rtgTypeSize(meta, destType) {
				return false
			}
			if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyMemSecondaryToStack(g, offset, size)
			return true
		}
		if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
			valueType := rtgInferParsedExprType(g, ep, idx)
			if !rtgTypeIsStruct(meta, valueType) || rtgTypeSize(meta, valueType) != rtgTypeSize(meta, destType) {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmCopyPrimaryToSecondary(&g.asm)
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyMemSecondaryToStack(g, offset, size)
			return true
		}
		if e.kind == rtgExprComposite {
			rtgZeroLocalAtOffset(g, offset)
			for i := 0; i < e.argCount; i++ {
				field := ep.fields[e.firstArg+i]
				fieldIndex := rtgCompositeStructFieldIndex(g, destType, &field, i)
				if fieldIndex < 0 {
					return false
				}
				fieldOffset := g.meta.fields[fieldIndex].offset
				fieldType := g.meta.fields[fieldIndex].typ
				if fieldType == 0 {
					return false
				}
				if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
					return false
				}
			}
			return true
		}
		return false
	}
	if destResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStorePrimaryStack(&g.asm, offset)
		rtgAsmStoreSecondaryStack(&g.asm, offset-8)
		return true
	}
	if rtgTypeKindIsScalarValue(destResolved.kind) || destResolved.kind == rtgTypePointer {
		if !rtgEmitScalarExprForKind(g, ep, idx, destResolved.kind) {
			return false
		}
		rtgAsmStorePrimaryStack(&g.asm, offset)
		return true
	}
	if !rtgTypeIsSlice(meta, destType) {
		return false
	}
	if !rtgEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmStoreSliceStack(&g.asm, offset)
	return true
}

const rtgMapEntrySize = 24

func rtgEmitMapAssignToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, mapType int, offset int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	resolved := rtgResolveType(g.meta, mapType)
	if resolved.kind != rtgTypeMap || !rtgTypeIsString(g.meta, resolved.first) {
		return false
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgResolveType(g.meta, g.locals[localIndex].typ).kind != rtgTypeMap {
			return false
		}
		rtgEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, 24)
		return true
	}
	entrySize := rtgMapEntrySize
	if e.kind == rtgExprComposite {
	} else if e.kind == rtgExprCall {
		if rtgExprIdentCode(g.prog, ep, e.left) != rtgIdentMake || e.argCount < 1 || e.argCount > 2 {
			return false
		}
		madeType := rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
		if rtgResolveType(g.meta, madeType).kind != rtgTypeMap {
			return false
		}
		// A map capacity is a performance hint. Evaluate it for its side
		// effects, but let the shared growable descriptor allocate storage.
		if e.argCount == 2 && !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
			return false
		}
	} else {
		return false
	}
	rtgAsmStoreStackImm(a, offset, 0)
	rtgAsmStorePrimaryStack(a, offset-8)
	rtgAsmStorePrimaryStack(a, offset-16)
	if e.kind == rtgExprComposite {
		valueResolved := rtgResolveType(g.meta, resolved.elem)
		if !rtgTypeKindIsScalarIntOrPointer(valueResolved.kind) {
			return false
		}
		entryPtrOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		loc := rtgSliceLocation{offset: offset}
		for i := 0; i < e.argCount; i++ {
			field := &ep.fields[e.firstArg+i]
			if !rtgEmitAppendDestPrimary(g, ep, &loc, entrySize) {
				return false
			}
			rtgAsmStorePrimaryStack(a, entryPtrOff)
			if field.key < 0 || !rtgEmitStringValueRegs(g, ep, field.key) {
				return false
			}
			rtgAsmPushStringRegs(a)
			rtgAsmLoadSecondaryStack(a, entryPtrOff)
			rtgAsmPopStoreStringMemSecondary(a, 0)
			if !rtgEmitIntExpr(g, ep, field.expr) {
				return false
			}
			rtgAsmNormalizePrimaryForKind(a, valueResolved.kind)
			rtgAsmPushPrimary(a)
			rtgAsmLoadSecondaryStack(a, entryPtrOff)
			rtgAsmPopPrimary(a)
			rtgAsmStorePrimaryMemSecondaryDispSize(a, 16, rtgScalarKindSize(valueResolved.kind))
		}
	}
	return true
}

func rtgEmitSliceReturnValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, resultType int) bool {
	if !rtgEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	if rtgReturnedSliceCanReuseDescriptor(g, ep, idx) {
		return true
	}
	return rtgEmitCopySliceRegsToArena(g, resultType)
}

func rtgReturnedSliceCanReuseDescriptor(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 {
		return false
	}
	if idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if callee == rtgIdentAppend && e.argCount >= 1 {
			return rtgReturnedSliceCanReuseDescriptor(g, ep, ep.args[e.firstArg])
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 && fnIndex < len(g.meta.funcs) {
			fn := &g.meta.funcs[fnIndex]
			if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaPersistBytes") {
				return true
			}
			return rtgCallSliceResultCanReuseDescriptor(g, ep, idx, fnIndex)
		}
	}
	if e.kind != rtgExprIdent {
		return false
	}
	if rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
		return true
	}
	localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
	if localIndex < 0 {
		return true
	}
	return rtgLocalIsCurrentFuncParam(g, localIndex)
}

func rtgCallSliceResultCanReuseDescriptor(g *rtgLinearGen, ep *rtgExprParse, idx int, fnIndex int) bool {
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
		if callee.kind != rtgExprSelector {
			return false
		}
		receiverType := rtgInferParsedExprType(g, ep, callee.left)
		if rtgTypeIsSlice(g.meta, receiverType) && !rtgReturnedSliceCanReuseDescriptor(g, ep, callee.left) {
			return false
		}
	}
	for i := 0; i < e.argCount; i++ {
		argIndex := ep.args[e.firstArg+i]
		argType := rtgInferParsedExprType(g, ep, argIndex)
		if rtgTypeIsSlice(g.meta, argType) && !rtgReturnedSliceCanReuseDescriptor(g, ep, argIndex) {
			return false
		}
	}
	return true
}

func rtgLocalIsCurrentFuncParam(g *rtgLinearGen, localIndex int) bool {
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

func rtgEmitCopySliceRegsToArena(g *rtgLinearGen, sliceType int) bool {
	a := &g.asm
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	slackSize := 64
	if elemSize > slackSize {
		slackSize = elemSize
	}
	slackCapacity := slackSize / elemSize
	srcOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	capOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	copyCapOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	byteCountOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	allocSizeOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	destOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	indexOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	nonNilLabel := rtgAsmNewLabel(a)
	capOKLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	returnLabel := rtgAsmNewLabel(a)
	rtgAsmStorePrimaryStack(a, srcOff)
	rtgAsmStoreSecondaryStack(a, lenOff)
	rtgAsmStackMem(a, capOff, 0x8948, 0x4d, 0x8d)
	rtgAsmLoadPrimaryStack(a, lenOff)
	rtgAsmPushImm(a, slackCapacity)
	rtgAsmPopTertiary(a)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmStorePrimaryStack(a, copyCapOff)
	rtgAsmLoadPrimaryStack(a, capOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryStack(a, copyCapOff)
	rtgAsmPopTertiary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9e)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, capOKLabel)
	rtgAsmCopyStackSlot(a, copyCapOff, capOff)
	rtgAsmMarkLabel(a, capOKLabel)
	rtgAsmLoadPrimaryStack(a, srcOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, nonNilLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmLoadSecondaryStack(a, lenOff)
	rtgAsmLoadTertiaryStack(a, capOff)
	rtgAsmJmpLabel(a, returnLabel)
	rtgAsmMarkLabel(a, nonNilLabel)
	rtgAsmLoadPrimaryStack(a, lenOff)
	if elemSize != 1 {
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmStackMem(a, byteCountOff, 0x8948, 0x4d, 0x8d)
	} else {
		rtgAsmStorePrimaryStack(a, byteCountOff)
	}
	rtgAsmLoadPrimaryStack(a, capOff)
	if elemSize != 1 {
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmPushTertiary(a)
		rtgAsmPopPrimary(a)
	}
	rtgAsmStorePrimaryStack(a, allocSizeOff)
	rtgEmitArenaAllocStackPrimary(g, allocSizeOff)
	rtgAsmStorePrimaryStack(a, destOff)
	if rtgTargetArch == rtgArchAmd64 {
		rtgAsmLoadPrimaryStack(a, destOff)
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmLoadPrimaryStack(a, srcOff)
		rtgAsmCopyPrimaryToCallWord1(a)
		rtgAsmLoadTertiaryStack(a, byteCountOff)
		rtgAsmEmit16(a, 0xa4f3)
	} else {
		rtgAsmStoreStackImm(a, indexOff, 0)
		rtgAsmMarkLabel(a, loopLabel)
		rtgAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
		rtgEmitAppendExpansionCopyElement(g, elemSize, srcOff, indexOff, destOff, indexOff)
		rtgAsmIncStack(a, indexOff)
		rtgAsmJmpLabel(a, loopLabel)
		rtgAsmMarkLabel(a, doneLabel)
	}
	rtgAsmLoadPrimaryStack(a, destOff)
	rtgAsmLoadSecondaryStack(a, lenOff)
	rtgAsmLoadTertiaryStack(a, capOff)
	rtgAsmMarkLabel(a, returnLabel)
	return true
}

func rtgEmitSliceValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, e.left) {
			return false
		}
		if e.firstArg >= 0 || e.nameStart >= 0 {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			baseResolved := rtgResolveType(meta, baseType)
			if baseResolved.kind != rtgTypeSlice {
				return false
			}
			elemSize := rtgTypeSize(meta, baseResolved.elem)
			if elemSize < 1 {
				elemSize = 8
			}
			baseOff := rtgAddUnnamedLocal(g, baseType)
			lowOff := rtgAddUnnamedLocal(g, rtgTypeInt)
			highOff := rtgAddUnnamedLocal(g, rtgTypeInt)
			maxOff := rtgAddUnnamedLocal(g, rtgTypeInt)
			rtgAsmStoreSliceStack(a, baseOff)
			if e.firstArg >= 0 {
				if !rtgEmitIntExpr(g, ep, e.firstArg) {
					return false
				}
			} else {
				rtgAsmPrimaryImm(a, 0)
			}
			rtgAsmStorePrimaryStack(a, lowOff)
			if e.right >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmStorePrimaryStack(a, highOff)
			} else {
				rtgAsmCopyStackSlot(a, baseOff-8, highOff)
			}
			if e.nameStart >= 0 {
				if !rtgEmitIntExpr(g, ep, e.nameStart) {
					return false
				}
				rtgAsmStorePrimaryStack(a, maxOff)
			} else {
				rtgAsmCopyStackSlot(a, baseOff-16, maxOff)
			}
			rtgAsmLoadPrimaryStack(a, maxOff)
			rtgAsmLoadTertiaryStack(a, lowOff)
			rtgAsmSubPrimaryTertiary(a)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryStack(a, highOff)
			rtgAsmLoadTertiaryStack(a, lowOff)
			rtgAsmSubPrimaryTertiary(a)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryStack(a, baseOff)
			rtgAsmLoadTertiaryStack(a, lowOff)
			if elemSize != 1 {
				rtgAsmMulTertiaryImm(a, elemSize)
			}
			rtgAsmAddPrimaryTertiary(a)
			rtgAsmPopSecondary(a)
			rtgAsmPopTertiary(a)
			return true
		}
		if e.right >= 0 {
			rtgAsmPushPrimary(a)
			rtgAsmPushTertiary(a)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopTertiary(a)
			rtgAsmPopPrimary(a)
		}
		return true
	}
	if e.kind == rtgExprIdent {
		if rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
			rtgAsmPrimaryImm(a, 0)
			rtgAsmSecondaryImm(a, 0)
			rtgAsmCopySecondaryToTertiary(a)
			return true
		}
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(meta, globalType) {
				return false
			}
			rtgAsmLoadPrimaryBss(a, globalOffset+16)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset+8)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset)
			rtgAsmPopSecondary(a)
			rtgAsmPopTertiary(a)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset-8)
		rtgAsmLoadTertiaryStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 16)
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmPopSecondary(a)
		rtgAsmPopPrimary(a)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 16)
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmPopSecondary(a)
		rtgAsmPopPrimary(a)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		return rtgEmitSliceLiteralRegs(g, ep, idx, sliceType)
	}
	if e.kind == rtgExprCall {
		prog := g.prog
		calleeLeft := e.left
		callee := rtgExprIdentCode(prog, ep, calleeLeft)
		if e.argCount == 2 && callee == rtgIdentAppend {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, &stmt, ep, 0) {
				return false
			}
			return rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgEmitMakeSliceRegs(g, ep, idx)
			}
		}
		if e.argCount == 1 && callee == rtgIdentByteSlice {
			return rtgEmitByteSliceConversionRegs(g, ep, idx)
		}
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, callType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}
func rtgEmitStringSliceValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind != rtgExprSlice {
		return false
	}
	baseType := rtgInferParsedExprType(g, ep, e.left)
	if !rtgTypeIsString(meta, baseType) {
		return false
	}
	if !rtgEmitStringValueRegs(g, ep, e.left) {
		return false
	}
	if e.firstArg >= 0 {
		baseOff := rtgAddUnnamedLocal(g, rtgTypeString)
		lowOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		highOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgAsmStorePrimaryStack(a, baseOff)
		rtgAsmCopySecondaryToPrimary(a)
		rtgAsmStorePrimaryStack(a, baseOff-8)
		if !rtgEmitIntExpr(g, ep, e.firstArg) {
			return false
		}
		rtgAsmStorePrimaryStack(a, lowOff)
		if e.right >= 0 {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmStorePrimaryStack(a, highOff)
		} else {
			rtgAsmCopyStackSlot(a, baseOff-8, highOff)
		}
		rtgAsmLoadPrimaryStack(a, highOff)
		rtgAsmLoadTertiaryStack(a, lowOff)
		rtgAsmSubPrimaryTertiary(a)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryStack(a, baseOff)
		rtgAsmLoadTertiaryStack(a, lowOff)
		rtgAsmAddPrimaryTertiary(a)
		rtgAsmPopSecondary(a)
		return true
	}
	if e.right >= 0 {
		rtgAsmPushPrimary(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
	}
	return true
}
func rtgEmitSliceLiteralRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int) bool {
	return rtgEmitSliceLiteralRegsWithMode(g, ep, idx, sliceType, false)
}
func rtgEmitGlobalSliceLiteralRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int) bool {
	return rtgEmitSliceLiteralRegsWithMode(g, ep, idx, sliceType, true)
}
func rtgEmitSliceLiteralRegsWithMode(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int, globalInit bool) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	needSize := e.argCount * elemSize
	backingSize := rtgStaticSliceBackingSize(needSize, elemSize)
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !rtgEmitSliceLiteralBacking(g, ep, idx, sliceType, backingOff, globalInit) {
		return false
	}
	rtgAsmPrimaryImm(a, e.argCount)
	rtgAsmPushPrimary(a)
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmSecondaryImm(a, e.argCount)
	rtgAsmPopTertiary(a)
	return true
}
func rtgEmitSliceLiteralBacking(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int, backingOff int, globalInit bool) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemType := t.elem
	elemResolved := rtgResolveType(g.meta, elemType)
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		if field.nameEnd > field.nameStart {
			return false
		}
		disp := i * elemSize
		if elemResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushStringRegs(a)
			rtgAsmPrimaryBssAddr(a, backingOff)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopStoreStringMemSecondary(a, disp)
			continue
		}
		if elemResolved.kind == rtgTypeStruct {
			if globalInit {
				if !rtgEmitGlobalStructInit(g, ep, field.expr, elemType, backingOff+disp) {
					return false
				}
				continue
			}
			addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
			rtgAsmPrimaryBssAddr(a, backingOff)
			rtgAsmCopyPrimaryToSecondary(a)
			if disp != 0 {
				rtgAsmAddSecondaryImm(a, disp)
			}
			rtgAsmStoreSecondaryStack(a, addrOffset)
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if !rtgTypeKindIsScalarInt(elemResolved.kind) && elemResolved.kind != rtgTypePointer {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmNormalizePrimaryForKind(a, elemResolved.kind)
		rtgAsmPushPrimary(a)
		rtgAsmPrimaryBssAddr(a, backingOff)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryDispSize(a, disp, elemSize)
	}
	return true
}
func rtgEmitMakeSliceRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 && e.argCount != 3 {
		return false
	}
	sliceType := rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	lenOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	capOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmStorePrimaryStack(a, lenOffset)
	if e.argCount == 3 {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+2]) {
			return false
		}
		rtgAsmStorePrimaryStack(a, capOffset)
	} else {
		rtgAsmCopyStackSlot(a, lenOffset, capOffset)
	}
	backingSize := 32768
	backingConst := false
	lenConst := rtgEvalConstExpr(g, ep, ep.args[e.firstArg+1])
	if lenConst.ok && lenConst.value > 0 {
		backingSize = rtgStaticSliceBackingSize(lenConst.value*elemSize, elemSize)
		backingConst = true
	}
	if e.argCount == 3 {
		backingConst = false
		capConst := rtgEvalConstExpr(g, ep, ep.args[e.firstArg+2])
		if capConst.ok && capConst.value > 0 {
			backingSize = rtgStaticSliceBackingSize(capConst.value*elemSize, elemSize)
			backingConst = true
		}
	}
	if backingConst {
		zeroSize := 0
		if lenConst.ok && lenConst.value > 0 {
			zeroSize = lenConst.value * elemSize
		}
		rtgEmitMakeStaticRingPrimary(g, backingSize, zeroSize)
	} else {
		sizeOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgAsmLoadTertiaryStack(a, capOffset)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmPushTertiary(a)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryStack(a, sizeOffset)
		rtgEmitArenaAllocStackPrimary(g, sizeOffset)
		rtgEmitZeroDynamicMakeSlice(g, lenOffset, elemSize)
	}
	rtgAsmLoadSecondaryStack(a, lenOffset)
	rtgAsmLoadTertiaryStack(a, capOffset)
	return true
}

func rtgEmitZeroDynamicMakeSlice(g *rtgLinearGen, lenOffset int, elemSize int) {
	a := &g.asm
	rtgAsmLoadTertiaryStack(a, lenOffset)
	rtgAsmMulTertiaryImm(a, elemSize)
	rtgAsmCallLabel(a, rtgEnsureMakeZeroHelper(g))
}

func rtgEnsureMakeZeroHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.makeZeroEmitted {
		return g.makeZeroLabel
	}
	g.makeZeroEmitted = true
	g.makeZeroLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.makeZeroLabel)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPushPrimary(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)
	rtgAsmAddSecondaryImm(a, 1)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmPushImm(a, 1)
	rtgAsmPopTertiary(a)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmPopPrimary(a)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.makeZeroLabel
}

func rtgMakeStaticRingSlotCount(backingSize int) int {
	if backingSize <= 4096 {
		return 3
	}
	if backingSize <= 65536 {
		return 2
	}
	return 1
}

func rtgEmitMakeStaticRingPrimary(g *rtgLinearGen, backingSize int, zeroSize int) {
	a := &g.asm
	slotCount := rtgMakeStaticRingSlotCount(backingSize)
	cursorOff := g.asm.bssSize
	dataOff := cursorOff + 8
	g.asm.bssSize += 8 + backingSize*slotCount
	noWrapLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryBss(a, cursorOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmIncPrimary(a)
	rtgAsmCmpPrimaryImm8(a, slotCount)
	rtgAsmJnzLabel(a, noWrapLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmMarkLabel(a, noWrapLabel)
	rtgAsmStorePrimaryBss(a, cursorOff)
	rtgAsmMulTertiaryImm(a, backingSize)
	rtgAsmPrimaryBssAddr(a, dataOff)
	rtgAsmAddPrimaryTertiary(a)
	if zeroSize > 0 {
		if zeroSize > backingSize {
			zeroSize = backingSize
		}
		zeroSize = rtgAlignTo8(zeroSize)
		addrOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgAsmStorePrimaryStack(a, addrOff)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPrimaryImm(a, 0)
		for at := 0; at < zeroSize; at += 8 {
			rtgAsmStorePrimaryMemSecondaryDisp(a, at)
		}
		rtgAsmLoadPrimaryStack(a, addrOff)
	}
}
func rtgEmitByteSliceConversionRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	idxOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	argIndex := ep.args[e.firstArg]
	if !rtgEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	rtgAsmStorePrimaryStack(a, srcOff)
	rtgAsmStoreSecondaryStack(a, lenOff)
	rtgAsmStoreStackImm(a, idxOff, 0)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, idxOff, lenOff, doneLabel)
	rtgAsmLoadPrimaryStack(a, idxOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryStack(a, srcOff)
	rtgAsmPopTertiary(a)
	rtgAsmLoadBytePrimaryIndexTertiary(a)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryStack(a, idxOff)
	rtgAsmPushPrimary(a)
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStoreByteMemSecondaryTertiary(a)
	rtgAsmIncStack(a, idxOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmLoadSecondaryStack(a, lenOff)
	rtgAsmCopySecondaryToTertiary(a)
	return true
}
func rtgEmitCompositeFieldToStack(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, destOffset int) bool {
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeArray {
		e := &ep.exprs[idx]
		if e.kind != rtgExprComposite {
			return false
		}
		elemSize := rtgTypeSize(g.meta, fieldResolved.elem)
		for i := 0; i < e.argCount && i < fieldResolved.count; i++ {
			if !rtgEmitCompositeFieldToStack(g, ep, ep.fields[e.firstArg+i].expr, fieldResolved.elem, destOffset-i*elemSize) {
				return false
			}
		}
		return true
	}
	a := &g.asm
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreSliceStack(a, destOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStorePrimaryStack(a, destOffset)
		rtgAsmStoreSecondaryStack(a, destOffset-8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddUnnamedLocal(g, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgEmitCopyStackToStack(g, tempOffset, destOffset, size)
		return true
	}
	if !rtgEmitScalarExprForKind(g, ep, idx, fieldResolved.kind) {
		return false
	}
	rtgAsmStorePrimaryStack(a, destOffset)
	return true
}
func rtgEmitCopyStackToStack(g *rtgLinearGen, srcOffset int, destOffset int, size int) {
	a := &g.asm
	for at := 0; at < size; at += 8 {
		rtgAsmCopyStackSlot(a, srcOffset-at, destOffset-at)
	}
}
func rtgEmitCopyStackToMemSecondary(g *rtgLinearGen, srcOffset int, destDisp int, size int) {
	a := &g.asm
	for at := 0; at < size; at += 8 {
		rtgAsmLoadPrimaryStack(a, srcOffset-at)
		rtgAsmStorePrimaryMemSecondaryDisp(a, destDisp+at)
	}
}
func rtgEmitCopyMemSecondaryToStack(g *rtgLinearGen, destOffset int, size int) {
	a := &g.asm
	for at := 0; at < size; at += 8 {
		rtgAsmLoadPrimaryMemSecondaryDisp(a, at)
		rtgAsmStorePrimaryStack(a, destOffset-at)
	}
}
func rtgEmitPushStackWords(g *rtgLinearGen, offset int, size int, wordSize int) {
	for at := size - wordSize; at >= 0; at -= wordSize {
		rtgAsmLoadPrimaryStack(&g.asm, offset-at)
		rtgAsmPushPrimary(&g.asm)
	}
}
func rtgEmitPushBssWords(g *rtgLinearGen, offset int, size int, wordSize int) {
	for at := size - wordSize; at >= 0; at -= wordSize {
		rtgAsmLoadPrimaryBss(&g.asm, offset+at)
		rtgAsmPushPrimary(&g.asm)
	}
}
func rtgEmitPushMemSecondaryWords(g *rtgLinearGen, size int, wordSize int) {
	for at := size - wordSize; at >= 0; at -= wordSize {
		rtgAsmLoadPrimaryMemSecondaryDisp(&g.asm, at)
		rtgAsmPushPrimary(&g.asm)
	}
}
func rtgEmitIndexedStructToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	leftType := rtgInferParsedExprType(g, ep, e.left)
	sliceType := rtgResolveType(meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(meta, sliceType.elem)
	destResolved := rtgResolveType(meta, destType)
	if elemType.kind != rtgTypeStruct || destResolved.kind != rtgTypeStruct {
		return false
	}
	elemSize := rtgTypeSize(meta, sliceType.elem)
	if rtgTypeSize(meta, destType) != elemSize {
		return false
	}
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitSlicePtrLen(g, ep, e.left) {
		return false
	}
	rtgAsmPopTertiary(a)
	rtgAsmMulTertiaryImm(a, elemSize)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmAddSecondaryTertiary(a)
	rtgEmitCopyMemSecondaryToStack(g, offset, elemSize)
	return true
}
func rtgPrepareStructCall(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int) (int, int) {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !rtgTypeIsStruct(g.meta, g.meta.funcs[fnIndex].resultType) {
		return -1, 0
	}
	if rtgTypeSize(g.meta, destType) != rtgTypeSize(g.meta, g.meta.funcs[fnIndex].resultType) {
		return -1, 0
	}
	fn := &g.meta.funcs[fnIndex]
	receiverIndex := -1
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != rtgExprSelector {
			return -1, 0
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	wordCount := 1
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := ep.args[e.firstArg+i]
		paramIndex := i
		if receiverIndex >= 0 {
			paramIndex = i + 1
		}
		words := rtgEmitCallParamArgReverse(g, ep, argIndex, fn.firstParam+paramIndex)
		if words < 0 {
			return -1, 0
		}
		wordCount += words
	}
	if receiverIndex >= 0 {
		words := rtgEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			words = rtgEmitMethodReceiverArgTokensReverse(g, receiverDotTok, g.meta.params[fn.firstParam].typ)
			if words < 0 {
				return -1, 0
			}
		}
		wordCount += words
	}
	return fnIndex, wordCount
}
func rtgEmitStructCallToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	fnIndex, wordCount := rtgPrepareStructCall(g, ep, idx, destType)
	if fnIndex < 0 {
		return false
	}
	rtgAsmStackMem(&g.asm, offset, 0x8d48, 0x45, 0x85)
	rtgAsmPushPrimary(&g.asm)
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func rtgEmitStructCallToBss(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	fnIndex, wordCount := rtgPrepareStructCall(g, ep, idx, destType)
	if fnIndex < 0 {
		return false
	}
	rtgAsmPrimaryBssAddr(&g.asm, offset)
	rtgAsmPushPrimary(&g.asm)
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func rtgEmitUserCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return rtgEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		return false
	}
	firstArg := e.firstArg
	argCount := e.argCount
	expanded := e.nameStart
	wordCount := 0
	fn := &g.meta.funcs[fnIndex]
	if rtgEmitRuntimeArenaCall(g, ep, idx, fn) {
		return true
	}
	receiverIndex := -1
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != rtgExprSelector {
			return false
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	if expanded == 0 && fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart == 1 {
		fixed := fn.paramCount - 1
		if receiverIndex >= 0 {
			fixed--
		}
		if argCount < fixed {
			return false
		}
		if receiverIndex >= 0 {
			if !rtgEmitVariadicArgSliceReverse(g, ep, firstArg+fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		} else {
			if !rtgEmitVariadicArgSliceFromCallReverse(g, ep, idx, fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		}
		wordCount = 3
		for i := fixed - 1; i >= 0; i-- {
			argIndex := ep.args[firstArg+i]
			paramIndex := i
			if receiverIndex >= 0 {
				paramIndex = i + 1
			}
			words := rtgEmitCallParamArgReverse(g, ep, argIndex, fn.firstParam+paramIndex)
			if words < 0 {
				return false
			}
			wordCount += words
		}
	} else {
		for i := argCount - 1; i >= 0; i-- {
			argIndex := ep.args[firstArg+i]
			paramIndex := i
			if receiverIndex >= 0 {
				paramIndex = i + 1
			}
			words := rtgEmitCallParamArgReverse(g, ep, argIndex, fn.firstParam+paramIndex)
			if words < 0 {
				return false
			}
			wordCount += words
		}
	}
	if receiverIndex >= 0 {
		words := rtgEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			words = rtgEmitMethodReceiverArgTokensReverse(g, receiverDotTok, g.meta.params[fn.firstParam].typ)
			if words < 0 {
				return false
			}
		}
		wordCount += words
	}
	if fn.linkStatic != 0 {
		return rtgEmitLinkStaticCall(g, fn, wordCount)
	}
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func rtgEmitRuntimeArenaCall(g *rtgLinearGen, ep *rtgExprParse, idx int, fn *rtgFuncInfo) bool {
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_Exit") {
		return rtgEmitRuntimeExit(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaMark") {
		return rtgEmitRuntimeArenaMark(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaReset") {
		return rtgEmitRuntimeArenaReset(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaPersistMark") {
		return rtgEmitRuntimeArenaPersistMark(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaPersistReset") {
		return rtgEmitRuntimeArenaPersistReset(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaPersistString") {
		return rtgEmitRuntimeArenaPersistString(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaPersistBytes") {
		return rtgEmitRuntimeArenaPersistBytes(g, ep, idx)
	}
	if rtgBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "rtg_runtime_ArenaDiscard") {
		return rtgEmitRuntimeArenaDiscard(g, ep, idx)
	}
	return false
}

func rtgEmitRuntimeExit(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 1 || !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	a := &g.asm
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmExit(a)
		return true
	}
	if rtgTargetArch == rtgArchAarch64 {
		if rtgTargetIsDarwin() {
			rtgAarch64AsmMovRegReg(a, 0, rtgAarch64RegRax)
			rtgDarwinArm64CallImport(a, rtgDarwinImportExit)
		} else {
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPrimaryImm(a, 93)
			rtgAsmSyscall(a)
		}
		return true
	}
	if rtgTargetArch == rtgArchArm {
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmPrimaryImm(a, 1)
		rtgAsmSyscall(a)
		return true
	}
	if rtgTargetArch == rtgArch386 {
		if rtgTargetIsWindows() {
			rtgAsmPushPrimary(a)
			rtgWin386CallImport(a, rtgWinImportExitProcess)
		} else {
			rtgAsmCopyPrimaryToCallWord0(a)
			rtgAsmPrimaryImm(a, 1)
			rtgAsmSyscall(a)
		}
		return true
	}
	if rtgTargetArch != rtgArchAmd64 {
		return false
	}
	if rtgTargetIsWindows() {
		rtgAsmCopyPrimaryToTertiary(a)
		rtgWinAmd64CallImport(a, rtgWinImportExitProcess, 40)
	} else {
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmPrimaryImm(a, 60)
		rtgAsmSyscall(a)
	}
	return true
}

func rtgEmitRuntimeArenaDiscard(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if rtgTargetArch != rtgArchAmd64 || rtgTargetOS != rtgOSLinux {
		rtgAsmPrimaryImm(&g.asm, 0)
		return true
	}
	a := &g.asm
	startOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	doneLabel := rtgAsmNewLabel(a)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgAmd64AsmAddRaxImm32(a, 4095)
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmStorePrimaryStack(a, startOff)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmLoadTertiaryStack(a, startOff)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmStorePrimaryStack(a, lenOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAmd64AsmJccLabel(a, 0x8e, doneLabel)
	rtgAsmLoadPrimaryStack(a, startOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, lenOff)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmSecondaryImm(a, 4)
	rtgAsmPrimaryImm(a, 28)
	rtgAsmSyscall(a)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmPrimaryImm(a, 0)
	return true
}

func rtgEmitRuntimeArenaMark(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 0 {
		return false
	}
	a := &g.asm
	rtgStringHeapOffsets(g)
	readyLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, readyLabel)
	rtgAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	rtgAsmMarkLabel(a, readyLabel)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	return true
}

func rtgEmitRuntimeArenaReset(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgStringHeapOffsets(g)
	a := &g.asm
	if rtgTargetArch == rtgArchAmd64 && rtgTargetOS == rtgOSLinux {
		rtgEmitRuntimeArenaResetMadvise(g)
		return true
	}
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	return true
}

func rtgEmitRuntimeArenaResetMadvise(g *rtgLinearGen) {
	a := &g.asm
	markOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	oldOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	startOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmStorePrimaryStack(a, markOff)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	rtgAsmStorePrimaryStack(a, oldOff)
	rtgEmitRuntimeArenaClampOldToPersistent(g, oldOff)
	rtgAsmLoadPrimaryStack(a, markOff)
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	rtgAsmLoadPrimaryStack(a, markOff)
	rtgAmd64AsmAddRaxImm32(a, 4095)
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmStorePrimaryStack(a, startOff)
	rtgAsmLoadPrimaryStack(a, oldOff)
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmLoadTertiaryStack(a, startOff)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmStorePrimaryStack(a, lenOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAmd64AsmJccLabel(a, 0x8e, doneLabel)
	rtgAsmLoadPrimaryStack(a, startOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, lenOff)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmSecondaryImm(a, 4)
	rtgAsmPrimaryImm(a, 28)
	rtgAsmSyscall(a)
	rtgAsmMarkLabel(a, doneLabel)
}

func rtgEmitRuntimeArenaClampOldToPersistent(g *rtgLinearGen, oldOff int) {
	a := &g.asm
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmLoadTertiaryStack(a, oldOff)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9f)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmStorePrimaryStack(a, oldOff)
	rtgAsmMarkLabel(a, doneLabel)
}

func rtgEmitRuntimeArenaPersistMark(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 0 {
		return false
	}
	rtgEmitPersistentArenaReady(g)
	rtgAsmLoadPrimaryBss(&g.asm, g.stringHeapEndOff)
	return true
}

func rtgEmitRuntimeArenaPersistReset(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	rtgStringHeapOffsets(g)
	a := &g.asm
	if rtgTargetArch == rtgArchAmd64 && rtgTargetOS == rtgOSLinux {
		rtgEmitRuntimeArenaPersistResetMadvise(g)
		return true
	}
	rtgAsmStorePrimaryBss(a, g.stringHeapEndOff)
	return true
}

func rtgEmitRuntimeArenaPersistResetMadvise(g *rtgLinearGen) {
	a := &g.asm
	markOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	oldOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	startOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmStorePrimaryStack(a, markOff)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmStorePrimaryStack(a, oldOff)
	rtgAsmLoadPrimaryStack(a, markOff)
	rtgAsmStorePrimaryBss(a, g.stringHeapEndOff)
	rtgAsmLoadPrimaryStack(a, oldOff)
	rtgAmd64AsmAddRaxImm32(a, 4095)
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmStorePrimaryStack(a, startOff)
	rtgAsmLoadPrimaryStack(a, markOff)
	rtgAmd64AsmAndRaxImm32(a, -4096)
	rtgAsmLoadTertiaryStack(a, startOff)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmStorePrimaryStack(a, lenOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAmd64AsmJccLabel(a, 0x8e, doneLabel)
	rtgAsmLoadPrimaryStack(a, startOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, lenOff)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmSecondaryImm(a, 4)
	rtgAsmPrimaryImm(a, 28)
	rtgAsmSyscall(a)
	rtgAsmMarkLabel(a, doneLabel)
}

func rtgAmd64AsmAddRaxImm32(a *rtgAsm, imm int) {
	rtgAsmEmit16(a, 0x0548)
	rtgAsmEmit32(a, imm)
}

func rtgAmd64AsmAndRaxImm32(a *rtgAsm, imm int) {
	rtgAsmEmit16(a, 0x2548)
	rtgAsmEmit32(a, imm)
}

func rtgEmitRuntimeArenaPersistString(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg]) {
		return false
	}
	a := &g.asm
	srcOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	destOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStorePrimaryStack(a, srcOff)
	rtgAsmStoreSecondaryStack(a, lenOff)
	rtgEmitPersistentAllocToPrimary(g, lenOff)
	rtgAsmStorePrimaryStack(a, destOff)
	rtgEmitCopyBytesToPersistent(g, srcOff, lenOff, destOff)
	rtgAsmLoadPrimaryStack(a, destOff)
	rtgAsmLoadSecondaryStack(a, lenOff)
	return true
}

func rtgEmitRuntimeArenaPersistBytes(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg]) {
		return false
	}
	a := &g.asm
	srcOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	destOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStorePrimaryStack(a, srcOff)
	rtgAsmStoreSecondaryStack(a, lenOff)
	rtgEmitPersistentAllocToPrimary(g, lenOff)
	rtgAsmStorePrimaryStack(a, destOff)
	rtgEmitCopyBytesToPersistent(g, srcOff, lenOff, destOff)
	rtgAsmLoadPrimaryStack(a, destOff)
	rtgAsmLoadSecondaryStack(a, lenOff)
	rtgAsmLoadTertiaryStack(a, lenOff)
	return true
}

func rtgEmitCopyBytesToPersistent(g *rtgLinearGen, srcOff int, lenOff int, destOff int) {
	a := &g.asm
	if rtgTargetArch == rtgArchAmd64 {
		rtgAsmEmit8(a, 0x57)
		rtgAsmEmit8(a, 0x56)
		rtgAsmEmit8(a, 0x51)
		rtgAsmLoadPrimaryStack(a, destOff)
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmLoadPrimaryStack(a, srcOff)
		rtgAsmCopyPrimaryToCallWord1(a)
		rtgAsmLoadTertiaryStack(a, lenOff)
		rtgAsmEmit8(a, 0xfc)
		rtgAsmEmit16(a, 0xa4f3)
		rtgAsmEmit8(a, 0x59)
		rtgAsmEmit8(a, 0x5e)
		rtgAsmEmit8(a, 0x5f)
		return
	}
	indexOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmStoreStackImm(a, indexOff, 0)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
	rtgAsmLoadPrimaryStack(a, srcOff)
	rtgAsmLoadTertiaryStack(a, indexOff)
	rtgAsmLoadPrimaryIndexTertiarySize(a, 1)
	rtgAsmPushPrimary(a)
	rtgAsmLoadSecondaryStack(a, destOff)
	rtgAsmLoadTertiaryStack(a, indexOff)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
	rtgAsmIncStack(a, indexOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
}

func rtgEmitPersistentAllocToPrimary(g *rtgLinearGen, sizeOff int) {
	a := &g.asm
	rtgEmitPersistentArenaReady(g)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmLoadTertiaryStack(a, sizeOff)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmStorePrimaryBss(a, g.stringHeapEndOff)
}

func rtgEmitPersistentArenaReady(g *rtgLinearGen) {
	a := &g.asm
	rtgStringHeapOffsets(g)
	readyLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryBss(a, g.stringHeapEndOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, readyLabel)
	rtgAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	rtgAsmPushImm(a, rtgStringArenaSize())
	rtgAsmPopTertiary(a)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmStorePrimaryBss(a, g.stringHeapEndOff)
	lowReadyLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, lowReadyLabel)
	rtgAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	rtgAsmMarkLabel(a, lowReadyLabel)
	rtgAsmMarkLabel(a, readyLabel)
}

func rtgEmitLinkStaticCall(g *rtgLinearGen, fn *rtgFuncInfo, wordCount int) bool {
	if rtgTargetOS != rtgOSWindows {
		return false
	}
	importID := rtgAsmAddWinStaticImport(&g.asm, fn.linkDLLStart, fn.linkDLLEnd, fn.linkMethodStart, fn.linkMethodEnd, g.prog.src)
	if rtgTargetArch == rtgArch386 {
		rtgWin386CallImport(&g.asm, importID)
		return true
	}
	if rtgTargetArch != rtgArchAmd64 {
		return false
	}
	rtgWinAmd64CallStaticImport(&g.asm, importID, wordCount)
	return true
}

func rtgWinAmd64CallStaticImport(a *rtgAsm, importID int, wordCount int) {
	if wordCount > 0 {
		rtgAsmPopTertiary(a)
	}
	if wordCount > 1 {
		rtgAsmPopSecondary(a)
	}
	if wordCount > 2 {
		rtgAsmEmit16(a, 0x5841)
	}
	if wordCount > 3 {
		rtgAsmEmit16(a, 0x5941)
	}
	stackWords := 0
	shadow := 40
	// The RTG amd64 call frame is eight bytes off the Windows call-site
	// alignment when all arguments fit in registers. Reserve the usual shadow
	// space plus one alignment slot for that common case.
	if wordCount > 4 {
		stackWords = wordCount - 4
		shadow = 32
	}
	rtgAsmEmit4(a, 0x48, 0x83, 0xec, shadow)
	rtgAsmEmit16(a, 0x15ff)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddWinImportReloc(a, at, importID)
	adjust := shadow + stackWords*8
	if rtgAsmImmFits8Signed(adjust) {
		rtgAsmEmit4(a, 0x48, 0x83, 0xc4, adjust)
	} else {
		rtgAsmEmit24(a, 0xc48148)
		rtgAsmEmit32(a, adjust)
	}
}

func rtgEmitArbitrarySyscall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount < 1 || e.argCount > 7 {
		return false
	}
	if rtgTargetIsDarwin() {
		if e.argCount != 4 {
			return false
		}
		number := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
		// The Darwin directory adapter uses one compiler-intrinsic selector,
		// which is lowered to libc getdirentries rather than issued as a raw
		// Darwin syscall number.
		if !number.ok || number.value != 217 {
			return false
		}
	}
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := ep.args[e.firstArg+i]
		if !rtgEmitSyscallArg(g, ep, argIndex) {
			return false
		}
		rtgAsmPushPrimary(&g.asm)
	}
	return rtgEmitSyscallFromStack(g, e.argCount)
}

func rtgEmitSyscallArg(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsString(g.meta, typ) {
		return rtgEmitStringPtrExpr(g, ep, idx)
	}
	if rtgTypeIsSlice(g.meta, typ) {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		return true
	}
	return rtgEmitIntExpr(g, ep, idx)
}

func rtgEmitSyscallFromStack(g *rtgLinearGen, wordCount int) bool {
	a := &g.asm
	if rtgTargetArch == rtgArchAmd64 {
		rtgAsmPopPrimary(a)
		if wordCount > 1 {
			rtgAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			rtgAsmEmit8(a, 0x5e)
		}
		if wordCount > 3 {
			rtgAsmPopSecondary(a)
		}
		if wordCount > 4 {
			rtgAsmPopTertiary(a)
			rtgAsmEmit24(a, 0xca8949)
		}
		if wordCount > 5 {
			rtgAsmEmit16(a, 0x5841)
		}
		if wordCount > 6 {
			rtgAsmEmit16(a, 0x5941)
		}
		rtgAsmSyscall(a)
		return true
	}
	if rtgTargetArch == rtgArch386 {
		if wordCount > 6 {
			return false
		}
		rtgAsmPopPrimary(a)
		if wordCount > 1 {
			rtgAsmPopCallWord0(a)
		}
		if wordCount > 2 {
			rtgAsmEmit8(a, 0x59)
		}
		if wordCount > 3 {
			rtgAsmPopSecondary(a)
		}
		if wordCount > 4 {
			rtgAsmEmit8(a, 0x5e)
		}
		if wordCount > 5 {
			rtgAsmEmit8(a, 0x5f)
		}
		rtgAsmSyscall(a)
		return true
	}
	if rtgTargetArch == rtgArchAarch64 {
		if rtgTargetIsDarwin() {
			if wordCount != 4 {
				return false
			}
			rtgAarch64AsmPopReg(a, 9)
			rtgAarch64AsmPopReg(a, 0)
			rtgAarch64AsmPopReg(a, 1)
			rtgAarch64AsmPopReg(a, 2)
			baseOff := a.bssSize
			a.bssSize += 8
			rtgAarch64AsmMovRegAbs(a, 3, baseOff, rtgAbsBssReloc)
			rtgDarwinArm64CallImport(a, rtgDarwinImportGetdirentries)
			return true
		}
		rtgAarch64AsmPopReg(a, rtgAarch64RegSys)
		if wordCount > 1 {
			rtgAarch64AsmPopReg(a, 0)
		}
		if wordCount > 2 {
			rtgAarch64AsmPopReg(a, 1)
		}
		if wordCount > 3 {
			rtgAarch64AsmPopReg(a, 2)
		}
		if wordCount > 4 {
			rtgAarch64AsmPopReg(a, 3)
		}
		if wordCount > 5 {
			rtgAarch64AsmPopReg(a, 4)
		}
		if wordCount > 6 {
			rtgAarch64AsmPopReg(a, 5)
		}
		rtgAarch64AsmEmit(a, 0xd4000001)
		return true
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopReg(a, rtgArmRegSys)
		if wordCount > 1 {
			rtgArmAsmPopReg(a, 0)
		}
		if wordCount > 2 {
			rtgArmAsmPopReg(a, 1)
		}
		if wordCount > 3 {
			rtgArmAsmPopReg(a, 2)
		}
		if wordCount > 4 {
			rtgArmAsmPopReg(a, 3)
		}
		if wordCount > 5 {
			rtgArmAsmPopReg(a, 4)
		}
		if wordCount > 6 {
			rtgArmAsmPopReg(a, 5)
		}
		rtgArmAsmEmit(a, 0xef000000)
		return true
	}
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRax)
		if wordCount > 1 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRdi)
		}
		if wordCount > 2 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRsi)
		}
		if wordCount > 3 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRdx)
		}
		if wordCount > 4 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRcx)
		}
		if wordCount > 5 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegR8)
		}
		if wordCount > 6 {
			rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegR9)
		}
		rtgAsmSyscall(a)
		return true
	}
	return false
}
func rtgEmitCallParamArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, paramIndex int) int {
	if paramIndex >= 0 && paramIndex < len(g.meta.params) {
		param := &g.meta.params[paramIndex]
		if rtgTypeIsSlice(g.meta, param.typ) {
			e := &ep.exprs[idx]
			if e.kind == rtgExprIdent && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
				if !rtgEmitSliceValueRegs(g, ep, idx) {
					return -1
				}
				rtgAsmPushSliceRegs(&g.asm)
				return 3
			}
		}
		resolved := rtgResolveType(g.meta, param.typ)
		source := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		if resolved.kind == rtgTypeFloat64 || source.kind == rtgTypeFloat64 {
			if !rtgEmitScalarExprForKind(g, ep, idx, resolved.kind) {
				return -1
			}
			rtgAsmPushPrimary(&g.asm)
			return 1
		}
	}
	return rtgEmitCallArgReverse(g, ep, idx)
}
func rtgEmitMethodReceiverArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, receiverType int) int {
	meta := g.meta
	a := &g.asm
	receiver := rtgResolveType(meta, receiverType)
	exprType := rtgInferParsedExprType(g, ep, idx)
	actualExprType := exprType
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			actualExprType = g.locals[localIndex].typ
		}
	}
	actualExprResolved := rtgResolveType(meta, actualExprType)
	if receiver.kind == rtgTypePointer {
		if actualExprResolved.kind == rtgTypePointer {
			if !rtgEmitIntExpr(g, ep, idx) {
				return -1
			}
			rtgAsmPushPrimary(a)
			return 1
		}
		if !rtgEmitAddressPrimary(g, ep, idx) {
			return -1
		}
		rtgAsmPushPrimary(a)
		return 1
	}
	if receiver.kind != rtgTypePointer && actualExprResolved.kind == rtgTypePointer {
		if !rtgEmitIntExpr(g, ep, idx) {
			return -1
		}
		rtgAsmCopyPrimaryToSecondary(a)
		size := rtgTypeSize(meta, receiverType)
		if size <= rtgBackendValueSlotSize {
			rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, size)
			rtgAsmPushPrimary(a)
			return 1
		}
		rtgEmitPushMemSecondaryWords(g, size, rtgBackendValueSlotSize)
		return size / rtgBackendValueSlotSize
	}
	return rtgEmitCallArgReverse(g, ep, idx)
}
func rtgEmitMethodReceiverArgTokensReverse(g *rtgLinearGen, dotTok int, receiverType int) int {
	if dotTok <= 0 {
		return -1
	}
	start := dotTok - 1
	if !rtgTokIsKind(g.prog, start, rtgTokIdent) {
		return -1
	}
	var receiverEp rtgExprParse
	if !rtgParseExpressionOK(&receiverEp, g.prog, start, dotTok) {
		return -1
	}
	return rtgEmitMethodReceiverArgReverse(g, &receiverEp, len(receiverEp.exprs)-1, receiverType)
}
func rtgEmitAddressPrimary(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			rtgAsmAddressPrimaryStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			rtgAsmPrimaryBssAddr(a, globalOffset)
			return true
		}
	}
	if e.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmCopySecondaryToPrimary(a)
		return true
	}
	if e.kind == rtgExprIndex {
		if !rtgEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}
func rtgEmitVariadicArgSliceReverse(g *rtgLinearGen, ep *rtgExprParse, first int, count int, sliceType int) bool {
	fieldFirst := len(ep.fields)
	for i := 0; i < count; i++ {
		var field rtgCompositeField
		field.expr = ep.args[first+i]
		ep.fields = append(ep.fields, field)
	}
	idx := len(ep.exprs)
	var expr rtgExpr
	expr.kind = rtgExprComposite
	expr.firstArg = fieldFirst
	expr.argCount = count
	ep.exprs = append(ep.exprs, expr)
	if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
		return false
	}
	rtgAsmPushSliceRegs(&g.asm)
	return true
}
func rtgEmitVariadicArgSliceFromCallReverse(g *rtgLinearGen, ep *rtgExprParse, callIdx int, skip int, count int, sliceType int) bool {
	a := &g.asm
	call := &ep.exprs[callIdx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	if !rtgTypeKindIsScalarValue(elem.kind) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	needSize := count * elemSize
	backingSize := rtgStaticSliceBackingSize(needSize, elemSize)
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	closeTok := rtgFindMatchingExprClose(g.prog, call.tok+1, ep.end, '(', ')')
	if closeTok <= call.tok {
		return false
	}
	pos := call.tok + 1
	argIndex := 0
	emitted := 0
	for pos < closeTok && emitted < count {
		argEnd := rtgFindExprBoundary(g.prog, pos, closeTok)
		if rtgTokCharIs(g.prog, argEnd, '{') {
			compositeEnd := rtgSkipBalanced(g.prog, argEnd, '{', '}')
			if compositeEnd > argEnd {
				argEnd = compositeEnd
			}
		}
		if argIndex >= skip {
			var argEp rtgExprParse
			rootIndex := rtgParseExpressionRoot(&argEp, g.prog, pos, argEnd)
			if rootIndex < 0 {
				return false
			}
			if !rtgEmitScalarExprForKind(g, &argEp, rootIndex, elem.kind) {
				return false
			}
			disp := emitted * elemSize
			rtgAsmPushPrimary(a)
			rtgAsmPrimaryBssAddr(a, backingOff)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopPrimary(a)
			rtgAsmStorePrimaryMemSecondaryDispSize(a, disp, elemSize)
			emitted++
		}
		pos = argEnd
		if rtgTokCharIs(g.prog, pos, ',') {
			pos++
		}
		argIndex++
	}
	if emitted != count {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmPrimaryImm(a, capacity)
	rtgAsmPushPrimary(a)
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmSecondaryImm(a, count)
	rtgAsmPopTertiary(a)
	rtgAsmPushSliceRegs(a)
	return true
}
func rtgEmitCallArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	a := &g.asm
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsSlice(g.meta, typ) {
		e := &ep.exprs[idx]
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex >= 0 {
				offset := g.locals[localIndex].offset
				rtgAsmPushStackWord(a, offset-16)
				rtgAsmPushStackWord(a, offset-8)
				rtgAsmPushStackWord(a, offset)
				return 3
			}
		}
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushSliceRegs(&g.asm)
		return 3
	}
	if rtgTypeIsString(g.meta, typ) {
		e := &ep.exprs[idx]
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex >= 0 {
				offset := g.locals[localIndex].offset
				rtgAsmPushStackWord(a, offset-8)
				rtgAsmPushStackWord(a, offset)
				return 2
			}
		}
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushStringRegs(&g.asm)
		return 2
	}
	if rtgTypeIsTuple(g.meta, typ) {
		return rtgEmitTupleArgReverse(g, ep, idx, typ)
	}
	resolved := rtgResolveType(g.meta, typ)
	if resolved.kind == rtgTypeStruct || resolved.kind == rtgTypeArray {
		return rtgEmitStructArgReverse(g, ep, idx, typ)
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt {
		value := rtgParseIntToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprIdent {
		constResult := rtgEvalConstExpr(g, ep, idx)
		if constResult.ok {
			rtgAsmPushImm(a, constResult.value)
			return 1
		}
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			rtgAsmPushStackWord(a, g.locals[localIndex].offset)
			return 1
		}
	}
	if e.kind == rtgExprSelector {
		if offset, ok := rtgLocalStructSelectorOffset(g, ep, idx); ok {
			rtgAsmPushStackWord(a, offset)
			return 1
		}
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return -1
	}
	rtgAsmPushPrimary(a)
	return 1
}

func rtgLocalStructSelectorOffset(g *rtgLinearGen, ep *rtgExprParse, idx int) (int, bool) {
	if rtgTargetArch != rtgArchAmd64 {
		return 0, false
	}
	e := &ep.exprs[idx]
	if e.kind != rtgExprSelector {
		return 0, false
	}
	base := &ep.exprs[e.left]
	if base.kind != rtgExprIdent {
		return 0, false
	}
	localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
	if localIndex < 0 {
		return 0, false
	}
	baseType := rtgResolveType(g.meta, g.locals[localIndex].typ)
	if baseType.kind != rtgTypeStruct {
		return 0, false
	}
	fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return 0, false
	}
	return g.locals[localIndex].offset - fieldOffset, true
}

func rtgEmitTupleArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	e := &ep.exprs[idx]
	if e.kind != rtgExprCall {
		return -1
	}
	offset := rtgAddUnnamedLocal(g, typ)
	if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := rtgResolveType(g.meta, typ)
	wordCount := 0
	for i := tuple.count - 1; i >= 0; i-- {
		field := g.meta.fields[tuple.first+i]
		size := rtgTypeSize(g.meta, field.typ)
		if size < rtgBackendValueSlotSize {
			size = rtgBackendValueSlotSize
		}
		rtgEmitPushStackWords(g, offset-field.offset, size, rtgBackendValueSlotSize)
		wordCount += size / rtgBackendValueSlotSize
	}
	return wordCount
}
func rtgEmitStructArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	meta := g.meta
	a := &g.asm
	size := rtgTypeSize(meta, typ)
	if size <= 0 {
		return -1
	}
	e := &ep.exprs[idx]
	wordSize := rtgBackendValueSlotSize
	if rtgResolveType(meta, typ).kind == rtgTypeArray {
		wordSize = rtgNativeIntSize
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if rtgTypeSize(meta, g.locals[localIndex].typ) != size {
				return -1
			}
			rtgEmitPushStackWords(g, g.locals[localIndex].offset, size, wordSize)
			return size / wordSize
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset < 0 || rtgTypeSize(meta, globalType) != size {
			return -1
		}
		rtgEmitPushBssWords(g, globalOffset, size, wordSize)
		return size / wordSize
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return -1
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return -1
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return -1
		}
		rtgAsmPopTertiary(a)
		rtgAsmMulTertiaryImm(a, size)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmAddSecondaryTertiary(a)
		rtgEmitPushMemSecondaryWords(g, size, wordSize)
		return size / wordSize
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != size {
			return -1
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return -1
		}
		rtgEmitPushMemSecondaryWords(g, size, wordSize)
		return size / wordSize
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(meta, valueType) || rtgTypeSize(meta, valueType) != size {
			return -1
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return -1
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgEmitPushMemSecondaryWords(g, size, wordSize)
		return size / wordSize
	}
	if e.kind == rtgExprComposite {
		offset := rtgAddUnnamedLocal(g, typ)
		rtgZeroLocalAtOffset(g, offset)
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
			if fieldIndex < 0 {
				return -1
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
				return -1
			}
		}
		rtgEmitPushStackWords(g, offset, size, wordSize)
		return size / wordSize
	}
	if e.kind == rtgExprCall {
		offset := rtgAddUnnamedLocal(g, typ)
		if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
			return -1
		}
		rtgEmitPushStackWords(g, offset, size, wordSize)
		return size / wordSize
	}
	return -1
}
func rtgEmitAppendAssignGeneral(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse, assignTok int) bool {
	p := g.prog
	if len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || rtgExprIdentCode(p, ep, root.left) != rtgIdentAppend {
		return false
	}
	if assignTok > stmt.startTok && !rtgAppendAssignLhsMatchesSource(p, stmt, ep, root, assignTok) {
		return rtgEmitAppendAssignDifferentSource(g, stmt, ep, root, assignTok)
	}
	var loc rtgSliceLocation
	locEp := ep
	if assignTok > stmt.startTok {
		var lhs rtgExprParse
		if rtgParseExpressionOK(&lhs, p, stmt.startTok, assignTok) {
			lhsIndex := len(lhs.exprs) - 1
			rtgSetSliceLocationFromExpr(g, &lhs, lhsIndex, &loc)
			locEp = &lhs
		}
	}
	if !loc.ok {
		rtgSetSliceLocationFromExpr(g, ep, ep.args[root.firstArg], &loc)
		locEp = ep
	}
	if !loc.ok {
		return false
	}
	return rtgEmitAppendToLocation(g, stmt, ep, locEp, &loc, root)
}

func rtgEmitAppendAssignDifferentSource(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse, root *rtgExpr, assignTok int) bool {
	p := g.prog
	var lhs rtgExprParse
	lhsIndex := rtgParseExpressionRoot(&lhs, p, stmt.startTok, assignTok)
	if lhsIndex < 0 {
		return false
	}
	var lhsLoc rtgSliceLocation
	rtgSetSliceLocationFromExpr(g, &lhs, lhsIndex, &lhsLoc)
	if !lhsLoc.ok {
		return false
	}
	sourceType := rtgInferParsedExprType(g, ep, ep.args[root.firstArg])
	if !rtgTypeIsSlice(g.meta, sourceType) {
		return false
	}
	tempOffset := rtgAddUnnamedLocal(g, sourceType)
	if !rtgEmitSliceValueRegs(g, ep, ep.args[root.firstArg]) {
		return false
	}
	rtgAsmStoreSliceStack(&g.asm, tempOffset)
	tempLoc := rtgSliceLocation{offset: tempOffset, typ: sourceType, ok: true}
	if !rtgEmitAppendToLocation(g, stmt, ep, ep, &tempLoc, root) {
		return false
	}
	rtgAsmLoadPrimaryStack(&g.asm, tempOffset)
	rtgAsmLoadSecondaryStack(&g.asm, tempOffset-8)
	rtgAsmLoadTertiaryStack(&g.asm, tempOffset-16)
	return rtgStoreSliceRegsToLocation(g, &lhs, &lhsLoc)
}

func rtgStoreSliceRegsToLocation(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation) bool {
	if loc.global {
		rtgAsmStoreSliceBss(&g.asm, loc.offset)
		return true
	}
	if loc.mem {
		rtgAsmPushSliceRegs(&g.asm)
		if !rtgEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		rtgAsmPopStoreSliceMemSecondary(&g.asm, 0)
		return true
	}
	rtgAsmStoreSliceStack(&g.asm, loc.offset)
	return true
}

func rtgEmitSliceLocationHeaderAddressSecondary(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation) bool {
	if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
		return false
	}
	if loc.deref {
		if !rtgEmitIntExpr(g, locEp, loc.expr) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(&g.asm)
		return true
	}
	return rtgEmitSelectorAddressSecondary(g, locEp, loc.expr)
}

func rtgAppendAssignLhsMatchesSource(p *rtgProgram, stmt *rtgStmt, ep *rtgExprParse, root *rtgExpr, assignTok int) bool {
	firstStart := root.tok + 2
	closeTok := rtgFindMatchingExprClose(p, root.tok+1, ep.end, '(', ')')
	if closeTok <= firstStart {
		return false
	}
	firstEnd := rtgFindExprBoundary(p, firstStart, closeTok)
	if firstEnd <= firstStart {
		return false
	}
	return rtgTokenRangesEqualSource(p, stmt.startTok, assignTok, firstStart, firstEnd)
}

func rtgTokenRangesEqualSource(p *rtgProgram, aStartTok int, aEndTok int, bStartTok int, bEndTok int) bool {
	for aStartTok < aEndTok && rtgTokIsSpaceLike(p, aStartTok) {
		aStartTok++
	}
	for bStartTok < bEndTok && rtgTokIsSpaceLike(p, bStartTok) {
		bStartTok++
	}
	for aEndTok > aStartTok && rtgTokIsSpaceLike(p, aEndTok-1) {
		aEndTok--
	}
	for bEndTok > bStartTok && rtgTokIsSpaceLike(p, bEndTok-1) {
		bEndTok--
	}
	if aStartTok >= aEndTok || bStartTok >= bEndTok {
		return false
	}
	aStart := int(rtgTokStart(p, aStartTok))
	aEnd := int(rtgTokEnd(p, aEndTok-1))
	bStart := int(rtgTokStart(p, bStartTok))
	bEnd := int(rtgTokEnd(p, bEndTok-1))
	return rtgBytesEqualRange(p.src, aStart, aEnd, bStart, bEnd)
}

func rtgTokIsSpaceLike(p *rtgProgram, tok int) bool {
	if tok < 0 || tok >= rtgTokCount(p) {
		return false
	}
	return rtgTokCharIs(p, tok, ';')
}

func rtgEmitAppendToLocation(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, root *rtgExpr) bool {
	p := g.prog
	t := rtgResolveType(g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if root.nameStart == 1 {
		if elem.kind == rtgTypeByte && rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, valueIndex)) {
			return rtgEmitAppendStringBytesToLocation(g, ep, valueIndex, locEp, loc)
		}
		return rtgEmitAppendExpansionToLocation(g, ep, locEp, loc, t.elem, valueIndex)
	}
	if elem.kind == rtgTypeStruct {
		value := &ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprUnary && rtgTokCharIs(p, value.tok, '*') {
				return rtgEmitAppendStructDeref(g, ep, locEp, loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokCharIs(p, typeTok+1, '{') {
					typeTok = 0
					for i := root.tok; i < stmt.endTok; i++ {
						if int(rtgTokStart(p, i)) == value.nameStart {
							typeTok = i
							break
						}
					}
				}
				if rtgTokCharIs(p, typeTok+1, '{') {
					return rtgEmitAppendStructCompositeTokens(g, locEp, loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, locEp, loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprCall {
				return rtgEmitAppendStructComposite(g, ep, locEp, loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprIndex || value.kind == rtgExprSelector {
				valueType := rtgInferParsedExprType(g, ep, valueIndex)
				if rtgTypeIsStruct(g.meta, valueType) && rtgTypeSize(g.meta, valueType) == rtgTypeSize(g.meta, t.elem) {
					return rtgEmitAppendStructComposite(g, ep, locEp, loc, t.elem, valueIndex)
				}
			}
			typeTok := rtgFindAppendCompositeTypeToken(p, root.tok, stmt.endTok)
			if typeTok >= 0 {
				return rtgEmitAppendStructCompositeTokens(g, locEp, loc, t.elem, typeTok)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, locEp, loc, t.elem, valueIndex) {
			return false
		}
		return true
	}
	if rtgTypeKindIsScalarInt(elem.kind) {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeFloat64 {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeString {
		if !rtgEmitAppendStringToLocation(g, ep, locEp, loc, valueIndex) {
			return false
		}
		return true
	}
	return false
}
func rtgBinaryUsesFloat(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr) bool {
	p := g.prog
	if rtgTok2Is(p, e.tok, '&', '&') {
		return false
	}
	if rtgTok2Is(p, e.tok, '|', '|') {
		return false
	}
	left := rtgInferParsedExprType(g, ep, e.left)
	if left == rtgTypeFloat64 {
		return true
	}
	right := rtgInferParsedExprType(g, ep, e.right)
	if right == rtgTypeFloat64 {
		return true
	}
	if !ep.hasFloat {
		return false
	}
	if rtgExprValueIsFloat(g, ep, e.left) {
		return true
	}
	return rtgExprValueIsFloat(g, ep, e.right)
}
func rtgExprValueIsFloat(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == rtgExprFloat {
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgExprValueIsFloat(g, ep, e.left)
		}
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') || rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			return false
		}
		if rtgExprValueIsFloat(g, ep, e.left) {
			return true
		}
		return rtgExprValueIsFloat(g, ep, e.right)
	}
	if e.kind == rtgExprIdent || e.kind == rtgExprCall || e.kind == rtgExprIndex || e.kind == rtgExprSelector {
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	return false
}
func rtgExprCanFoldConst(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 {
		return false
	}
	if idx >= len(ep.exprs) {
		return false
	}
	e := ep.exprs[idx]
	if e.kind == rtgExprInt {
		return true
	}
	if e.kind == rtgExprChar {
		return true
	}
	if e.kind == rtgExprBool {
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].constValid != 0
		}
		builtin := rtgEvalBuiltinConst(g, e.nameStart, e.nameEnd)
		if builtin.ok {
			return true
		}
		if rtgFindMetaGlobalIndex(g.meta, e.nameStart, e.nameEnd, rtgTokConst) >= 0 {
			return true
		}
		return false
	}
	if e.kind == rtgExprUnary {
		leftFold := rtgExprCanFoldConst(g, ep, e.left)
		return leftFold
	}
	if e.kind == rtgExprBinary {
		leftFold := rtgExprCanFoldConst(g, ep, e.left)
		if !leftFold {
			return false
		}
		rightFold := rtgExprCanFoldConst(g, ep, e.right)
		return rightFold
	}
	if e.kind == rtgExprCall {
		if e.argCount != 1 {
			return false
		}
		argIndex := ep.args[e.firstArg]
		argFold := rtgExprCanFoldConst(g, ep, argIndex)
		if !argFold {
			return false
		}
		prog := g.prog
		calleeLeft := e.left
		callee := rtgExprIdentCode(prog, ep, calleeLeft)
		if callee == rtgIdentInt {
			return true
		}
		if callee == rtgIdentByte {
			return true
		}
		if callee == rtgIdentInt16 {
			return true
		}
		if callee == rtgIdentInt32 {
			return true
		}
		if callee == rtgIdentInt64 {
			return true
		}
		calleeExpr := ep.exprs[calleeLeft]
		if calleeExpr.kind == rtgExprIdent {
			return rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd) > 0
		}
	}
	return false
}
func rtgEmitAppendExpansionToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	sourceType := rtgInferParsedExprType(g, ep, valueIndex)
	source := rtgResolveType(g.meta, sourceType)
	if source.kind != rtgTypeSlice {
		return false
	}
	if rtgTypeSize(g.meta, source.elem) != elemSize {
		return false
	}
	if elemSize == 1 {
		srcPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
		srcLen := rtgAddUnnamedLocal(g, rtgTypeInt)
		srcIndex := rtgAddUnnamedLocal(g, rtgTypeInt)
		if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		rtgAsmStorePrimaryStack(a, srcPtr)
		rtgAsmStoreSecondaryStack(a, srcLen)
		rtgAsmStoreStackImm(a, srcIndex, 0)
		loopLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmMarkLabel(a, loopLabel)
		rtgAsmJgeStackStack(a, srcIndex, srcLen, doneLabel)
		rtgAsmLoadPrimaryStack(a, srcPtr)
		rtgAsmLoadTertiaryStack(a, srcIndex)
		rtgAsmLoadPrimaryIndexTertiarySize(a, 1)
		rtgAsmPushPrimary(a)
		if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
		rtgAsmIncStack(a, srcIndex)
		rtgAsmJmpLabel(a, loopLabel)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	srcPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcLen := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcIndex := rtgAddUnnamedLocal(g, rtgTypeInt)
	destPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
	destLen := rtgAddUnnamedLocal(g, rtgTypeInt)
	headerOffset := 0
	if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmStorePrimaryStack(a, srcPtr)
	rtgAsmStoreSecondaryStack(a, srcLen)
	rtgAsmStoreStackImm(a, srcIndex, 0)
	if loc.mem {
		if !rtgEmitSliceLocationHeaderAddressSecondary(g, locEp, loc) {
			return false
		}
		headerOffset = rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgAsmStoreSecondaryStack(a, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmStorePrimaryStack(a, destPtr)
		rtgAsmLoadSecondaryStack(a, headerOffset)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
		rtgAsmStorePrimaryStack(a, destLen)
	} else if loc.global {
		rtgAsmLoadPrimaryBss(a, loc.offset)
		rtgAsmStorePrimaryStack(a, destPtr)
		rtgAsmLoadPrimaryBss(a, loc.offset+8)
		rtgAsmStorePrimaryStack(a, destLen)
	} else {
		rtgAsmCopyStackSlot(a, loc.offset, destPtr)
		rtgAsmCopyStackSlot(a, loc.offset-8, destLen)
	}
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, srcIndex, srcLen, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	rtgAsmIncStack(a, srcIndex)
	rtgAsmIncStack(a, destLen)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadPrimaryStack(a, destLen)
	if loc.mem {
		rtgAsmLoadSecondaryStack(a, headerOffset)
		rtgAsmStorePrimaryMemSecondaryDisp(a, 8)
	} else if loc.global {
		rtgAsmStorePrimaryBss(a, loc.offset+8)
	} else {
		rtgAsmStorePrimaryStack(a, loc.offset-8)
	}
	return true
}
func rtgEmitAppendExpansionCopyElement(g *rtgLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	a := &g.asm
	if elemSize == 1 || elemSize == 2 || elemSize == 4 || elemSize == 8 {
		rtgAsmLoadPrimaryStack(a, srcPtr)
		rtgAsmLoadTertiaryStack(a, srcIndex)
		rtgAsmLoadPrimaryIndexTertiarySize(a, elemSize)
		rtgAsmPushPrimary(a)
		rtgAsmLoadSecondaryStack(a, destPtr)
		rtgAsmLoadTertiaryStack(a, destLen)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryTertiarySize(a, elemSize)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		rtgAsmLoadPrimaryStack(a, srcPtr)
		rtgAsmLoadTertiaryStack(a, srcIndex)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a, copyOff)
		rtgAsmPushPrimary(a)
		rtgAsmLoadSecondaryStack(a, destPtr)
		rtgAsmLoadTertiaryStack(a, destLen)
		rtgAsmMulTertiaryImm(a, elemSize)
		rtgAsmAddSecondaryTertiary(a)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryDisp(a, copyOff)
	}
}
func rtgFindAppendCompositeTypeToken(p *rtgProgram, openTok int, end int) int {
	if openTok < 0 || openTok >= end || !rtgTokCharIs(p, openTok, '(') {
		return -1
	}
	i := openTok + 1
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			typeTok := i + 1
			if rtgTokIsKind(p, typeTok, rtgTokIdent) && rtgTokCharIs(p, typeTok+1, '{') {
				return typeTok
			}
			return -1
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren == 0 {
				return -1
			}
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			brace--
		}
		i++
	}
	return -1
}
func rtgEmitAppendScalarToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemKind int, valueIndex int) bool {
	a := &g.asm
	elemSize := rtgScalarKindSize(elemKind)
	if !rtgEmitScalarExprForKind(g, ep, valueIndex, elemKind) {
		return false
	}
	rtgAsmPushPrimary(a)
	if rtgTargetArch == rtgArchAmd64 || rtgTargetArch == rtgArchAarch64 || elemSize == 1 || elemSize == 2 || elemSize == 4 {
		if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, elemSize)
		return true
	}
	label := rtgEnsureAppendScalarHelper(g, elemKind)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmPopSecondary(a)
	rtgAsmCallLabel(a, label)
	return true
}
func rtgEmitAppendDestPrimary(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	label := rtgEnsureAppendAddrHelper(g)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmSecondaryImm(&g.asm, elemSize)
	rtgAsmCallLabel(&g.asm, label)
	return true
}
func rtgEnsureAppendScalarHelper(g *rtgLinearGen, elemKind int) int {
	if rtgScalarKindSize(elemKind) == 1 {
		return rtgEnsureAppend8Helper(g)
	}
	return rtgEnsureAppend64Helper(g)
}
func rtgEmitAppendStringToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, valueIndex int) bool {
	a := &g.asm
	rtgEnsureAppendAddrHelper(g)
	if !rtgEmitStringValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitAppendDestPrimary(g, locEp, loc, 16) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopStoreStringMemSecondary(a, 0)
	return true
}
func rtgSetSliceLocationFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int, loc *rtgSliceLocation) {
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(g.meta, globalType) {
				return
			}
			loc.offset = globalOffset
			loc.typ = globalType
			loc.global = true
			loc.ok = true
			return
		}
		if !rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) {
			return
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.param = rtgLocalIsCurrentFuncParam(g, localIndex)
		loc.ok = true
		return
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(g.meta, fieldType) {
			return
		}
		loc.expr = idx
		loc.typ = fieldType
		loc.mem = true
		loc.ok = true
		return
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(g.meta, valueType) {
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
func rtgEmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	if rtgTargetArch == rtgArchWasm32 {
		backingSize = rtgWasm32FallbackSliceBackingSize
	}
	if rtgTargetArch == rtgArchAmd64 {
		backingSize = rtgAmd64SliceBackingSize(elemSize)
	}
	if rtgTargetArch == rtgArchAmd64 {
		rtgEmitArenaAllocPrimary(g, backingSize)
	} else {
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmPrimaryBssAddr(a, backingOff)
	}
	rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
	rtgAsmPrimaryImm(a, backingSize/elemSize)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 16)
	rtgAsmMarkLabel(a, okLabel)
}
func rtgEmitEnsureLocalSlice(g *rtgLinearGen, offset int, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryStack(a, offset)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	if rtgTargetArch == rtgArchWasm32 {
		backingSize = rtgWasm32FallbackSliceBackingSize
	}
	if rtgTargetArch == rtgArchAmd64 {
		backingSize = rtgAmd64SliceBackingSize(elemSize)
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmPrimaryBssAddr(a, backingOff)
	rtgAsmStorePrimaryStack(a, offset)
	rtgAsmStoreStackImm(a, offset-8, 0)
	rtgAsmStoreStackImm(a, offset-16, backingSize/elemSize)
	rtgAsmMarkLabel(a, okLabel)
}
func rtgEmitEnsureLocalSliceArena(g *rtgLinearGen, offset int, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadPrimaryStack(a, offset)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := rtgAmd64ArenaSliceBackingSize(elemSize)
	rtgEmitArenaAllocPrimary(g, backingSize)
	rtgAsmStorePrimaryStack(a, offset)
	rtgAsmStoreStackImm(a, offset-8, 0)
	rtgAsmStoreStackImm(a, offset-16, backingSize/elemSize)
	rtgAsmMarkLabel(a, okLabel)
}
func rtgEmitAppendStructCompositeTokens(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, typeTok int) bool {
	p := g.prog
	openTok := typeTok + 1
	closeTok := rtgSkipBalanced(p, openTok, '{', '}')
	if closeTok <= openTok {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	destOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStorePrimaryStack(&g.asm, destOffset)
	i := openTok + 1
	for i < closeTok-1 {
		if !rtgTokIsKind(p, i, rtgTokIdent) || !rtgTokCharIs(p, i+1, ':') {
			return false
		}
		fieldTok := rtgTokAt(p, i)
		exprStart := i + 2
		exprEnd := rtgFindExprBoundary(p, exprStart, closeTok-1)
		var ep rtgExprParse
		if !rtgParseExpressionOK(&ep, p, exprStart, exprEnd) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, elemType, int(fieldTok.start), int(fieldTok.end))
		if fieldOffset < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, elemType, int(fieldTok.start), int(fieldTok.end))
		if fieldType == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitCompositeFieldToMem(g, &ep, rootIndex, fieldType, destOffset, fieldOffset) {
			return false
		}
		i = exprEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}
func rtgEmitAppendStructDeref(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	value := &ep.exprs[valueIndex]
	valueType := rtgInferParsedExprType(g, ep, valueIndex)
	if !rtgTypeIsStruct(g.meta, valueType) || rtgTypeSize(g.meta, valueType) != rtgTypeSize(g.meta, elemType) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddUnnamedLocal(g, elemType)
	if !rtgEmitIntExpr(g, ep, value.left) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(a)
	rtgEmitCopyMemSecondaryToStack(g, tempOffset, elemSize)
	if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(a)
	rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
	return true
}
func rtgEmitAppendStructLocal(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	value := &ep.exprs[valueIndex]
	localIndex := rtgFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if rtgTypeSize(g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(&g.asm)
	rtgEmitCopyStackToMemSecondary(g, g.locals[localIndex].offset, 0, elemSize)
	return true
}
func rtgEmitAppendStructComposite(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddUnnamedLocal(g, elemType)
	if !rtgEmitTypedAssign(g, ep, valueIndex, tempOffset) {
		return false
	}
	if !rtgEmitAppendDestPrimary(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(&g.asm)
	rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, elemSize)
	return true
}
func rtgEmitStringCompare(g *rtgLinearGen, ep *rtgExprParse, left int, right int, notEqual bool) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	rightExpr := &ep.exprs[right]
	if rightExpr.kind == rtgExprSelector {
		rightOff := rtgAddUnnamedLocal(g, rtgTypeString)
		if !rtgEmitStringValueRegs(g, ep, right) {
			return false
		}
		rtgAsmStorePrimaryStack(a, rightOff)
		rtgAsmStoreSecondaryStack(a, rightOff-8)
		if !rtgEmitStringValueRegs(g, ep, left) {
			return false
		}
		rtgAsmPushStringRegs(a)
		rtgAsmLoadPrimaryStack(a, rightOff)
		rtgAsmLoadSecondaryStack(a, rightOff-8)
		rtgAsmCopySecondaryToTertiary(a)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopCallWord0(a)
		rtgAsmPopCallWord1(a)
		rtgAsmCallLabel(a, label)
		if notEqual {
			rtgAsmBoolNotPrimary(a)
		}
		return true
	}
	if !rtgEmitStringValueRegs(g, ep, left) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitStringValueRegs(g, ep, right) {
		return false
	}
	rtgAsmCopySecondaryToTertiary(a)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopCallWord0(a)
	rtgAsmPopCallWord1(a)
	rtgAsmCallLabel(a, label)
	if notEqual {
		rtgAsmBoolNotPrimary(a)
	}
	return true
}
func rtgEmitBuiltinCopy(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	destIndex := ep.args[e.firstArg]
	srcIndex := ep.args[e.firstArg+1]
	destType := rtgInferParsedExprType(g, ep, destIndex)
	srcType := rtgInferParsedExprType(g, ep, srcIndex)
	destSlice := rtgResolveType(g.meta, destType)
	srcSlice := rtgResolveType(g.meta, srcType)
	if destSlice.kind != rtgTypeSlice || srcSlice.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, destSlice.elem)
	if elemSize != rtgTypeSize(g.meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
	destLen := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcLen := rtgAddUnnamedLocal(g, rtgTypeInt)
	copyCount := rtgAddUnnamedLocal(g, rtgTypeInt)
	if !rtgEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	rtgAsmStorePrimaryStack(a, destPtr)
	rtgAsmStoreSecondaryStack(a, destLen)
	if !rtgEmitSliceValueRegs(g, ep, srcIndex) {
		return false
	}
	rtgAsmStorePrimaryStack(a, srcPtr)
	rtgAsmStoreSecondaryStack(a, srcLen)
	rtgAsmStoreStackImm(a, copyCount, 0)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, copyCount, destLen, doneLabel)
	rtgAsmJgeStackStack(a, copyCount, srcLen, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, copyCount, destPtr, copyCount)
	rtgAsmIncStack(a, copyCount)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadPrimaryStack(a, copyCount)
	return true
}
func rtgEmitSliceBasePtrLenTokens(g *rtgLinearGen, p *rtgProgram, start int, end int, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if start+1 == end && rtgTokIsKind(p, start, rtgTokIdent) {
		nameStart := int(rtgTokStart(p, start))
		nameEnd := int(rtgTokEnd(p, start))
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			rtgAsmLoadTertiaryStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, nameStart, nameEnd)
		globalType := rtgFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
			rtgAsmLoadPrimaryBss(a, globalOffset+8)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, '.') && rtgTokIsKind(p, start+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, int(rtgTokStart(p, start)), int(rtgTokEnd(p, start)))
		if localIndex < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, int(rtgTokStart(p, start+2)), int(rtgTokEnd(p, start+2)))
		if !rtgTypeIsSlice(meta, fieldType) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, int(rtgTokStart(p, start+2)), int(rtgTokEnd(p, start+2)))
		if fieldOffset < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddSecondaryImm(a, fieldOffset)
			}
		} else {
			rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmPopPrimary(a)
		} else {
			rtgAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		}
		return true
	}
	return rtgEmitSlicePtrLen(g, ep, idx)
}
func rtgEmitSlicePtrLen(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || (!rtgTypeIsSlice(meta, globalType) && !rtgTypeIsString(meta, globalType)) {
				return false
			}
			rtgAsmLoadPrimaryBss(a, globalOffset+8)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) && !rtgTypeIsString(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		rtgAsmLoadTertiaryStack(a, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		rtgAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, fieldType) && !rtgTypeIsString(meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmPopPrimary(a)
		} else {
			rtgAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		}
		return true
	}
	if e.kind == rtgExprIndex {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmCopySecondaryToTertiary(a)
		return true
	}
	if e.kind == rtgExprCall {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmCopySecondaryToTertiary(a)
		return true
	}
	return false
}
func rtgEmitSlicePtrCap(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(meta, globalType) {
				return false
			}
			rtgAsmLoadPrimaryBss(a, globalOffset+16)
			rtgAsmCopyPrimaryToTertiary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
		rtgAsmLoadTertiaryStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 16)
		rtgAsmCopyPrimaryToTertiary(a)
		rtgAsmPopPrimary(a)
		return true
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
		if !rtgTypeIsSlice(meta, rtgInferParsedExprType(g, ep, idx)) {
			return false
		}
		return rtgEmitSliceValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprCall {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		return rtgEmitSliceValueRegs(g, ep, idx)
	}
	return false
}
func rtgEmitIndexAddressPrimary(g *rtgLinearGen, ep *rtgExprParse, indexIdx int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	sliceType := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, indexExpr.left))
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	if sliceType.kind != rtgTypeArray && sliceType.kind != rtgTypeSlice {
		return false
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushPrimary(a)
	if sliceType.kind == rtgTypeArray {
		base := &ep.exprs[indexExpr.left]
		if base.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
			if localIndex < 0 {
				return false
			}
			rtgAsmAddressPrimaryStack(a, g.locals[localIndex].offset)
		} else if base.kind == rtgExprIndex {
			if !rtgEmitIndexAddressPrimary(g, ep, indexExpr.left) {
				return false
			}
		} else {
			return false
		}
	} else {
		if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
			return false
		}
	}
	rtgAsmPopTertiary(a)
	if elemSize != 1 {
		rtgAsmMulTertiaryImm(a, elemSize)
	}
	rtgAsmAddPrimaryTertiary(a)
	return true
}

func rtgEmitMapEntryAddress(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, create bool) bool {
	a := &g.asm
	e := &ep.exprs[indexIdx]
	mapExpr := &ep.exprs[e.left]
	if mapExpr.kind != rtgExprIdent {
		return false
	}
	localIndex := rtgFindLocalIndex(g, mapExpr.nameStart, mapExpr.nameEnd)
	if localIndex < 0 {
		return false
	}
	keyPtrOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	keyLenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	indexOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	entryOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	if !rtgEmitStringValueRegs(g, ep, e.right) {
		return false
	}
	rtgAsmStorePrimaryStack(a, keyPtrOff)
	rtgAsmStoreSecondaryStack(a, keyLenOff)
	rtgAsmStoreStackImm(a, indexOff, 0)
	loopLabel := rtgAsmNewLabel(a)
	notFoundLabel := rtgAsmNewLabel(a)
	foundLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	entrySize := rtgMapEntrySize
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, indexOff, g.locals[localIndex].offset-8, notFoundLabel)
	rtgAsmLoadTertiaryStack(a, indexOff)
	rtgAsmMulTertiaryImm(a, entrySize)
	rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset)
	rtgAsmAddSecondaryTertiary(a)
	rtgAsmStoreSecondaryStack(a, entryOff)
	rtgAsmLoadPrimaryStack(a, keyPtrOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, keyLenOff)
	rtgAsmPushPrimary(a)
	rtgAsmPopCallWord1(a)
	rtgAsmLoadSecondaryStack(a, entryOff)
	rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPopSecondary(a)
	rtgAsmCallLabel(a, rtgEnsureStringEqualHelper(g))
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, foundLabel)
	rtgAsmIncStack(a, indexOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, foundLabel)
	rtgAsmLoadPrimaryStack(a, entryOff)
	rtgAsmJmpLabel(a, doneLabel)
	rtgAsmMarkLabel(a, notFoundLabel)
	if create {
		loc := rtgSliceLocation{offset: g.locals[localIndex].offset}
		if !rtgEmitAppendDestPrimary(g, ep, &loc, entrySize) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadPrimaryStack(a, keyPtrOff)
		rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
		rtgAsmLoadPrimaryStack(a, keyLenOff)
		rtgAsmStorePrimaryMemSecondaryDisp(a, 8)
		rtgAsmCopySecondaryToPrimary(a)
	} else {
		rtgAsmPrimaryImm(a, 0)
	}
	rtgAsmMarkLabel(a, doneLabel)
	return true
}

func rtgEmitIndexExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	left := &ep.exprs[e.left]
	if left.kind == rtgExprString {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		msg := rtgDecodeStringToken(g.prog, left.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmPushPrimary(a)
		rtgAsmPrimaryDataAddr(a, msgOff)
		rtgAsmPopTertiary(a)
		rtgAsmLoadBytePrimaryIndexTertiary(a)
		return true
	}
	baseResolved := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.left))
	if baseResolved.kind == rtgTypeMap {
		if !rtgEmitMapEntryAddress(g, ep, idx, false) {
			return false
		}
		zeroLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmCmpPrimaryImm8(a, 0)
		rtgAsmJzLabel(a, zeroLabel)
		rtgAsmCopyPrimaryToSecondary(a)
		valueType := rtgResolveType(meta, baseResolved.elem)
		rtgAsmLoadPrimaryMemSecondaryDispSize(a, 16, rtgScalarKindSize(valueType.kind))
		rtgAsmJmpLabel(a, doneLabel)
		rtgAsmMarkLabel(a, zeroLabel)
		rtgAsmPrimaryImm(a, 0)
		rtgAsmMarkLabel(a, doneLabel)
		return true
	}
	if baseResolved.kind == rtgTypeArray {
		elemSize := rtgTypeSize(meta, baseResolved.elem)
		if !rtgEmitIndexAddressPrimary(g, ep, idx) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadPrimaryMemSecondaryDispSize(a, 0, elemSize)
		return true
	}
	if left.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, left.nameStart, left.nameEnd)
			globalType := rtgFindGlobalType(g, left.nameStart, left.nameEnd)
			if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushPrimary(a)
				rtgAsmLoadPrimaryBss(a, globalOffset)
				rtgAsmPopTertiary(a)
				rtgAsmLoadBytePrimaryIndexTertiary(a)
				return true
			}
			if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
				t := rtgResolveType(meta, globalType)
				elem := rtgResolveType(meta, t.elem)
				if !rtgTypeKindIsScalarValue(elem.kind) && elem.kind != rtgTypePointer {
					return false
				}
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushPrimary(a)
				rtgAsmLoadPrimaryBss(a, globalOffset)
				rtgAsmPopTertiary(a)
				rtgAsmLoadPrimaryIndexTertiaryScalarOrPointer(a, elem.kind)
				return true
			}
			constTok := rtgFindConstStringToken(g, left.nameStart, left.nameEnd)
			if constTok >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				msg := rtgDecodeStringToken(g.prog, constTok)
				msgOff := rtgAddStringData(g, msg)
				rtgAsmPushPrimary(a)
				rtgAsmPrimaryDataAddr(a, msgOff)
				rtgAsmPopTertiary(a)
				rtgAsmLoadBytePrimaryIndexTertiary(a)
				return true
			}
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			rtgAsmPopTertiary(a)
			rtgAsmLoadBytePrimaryIndexTertiary(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarValue(elem.kind) && elem.kind != rtgTypePointer {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			rtgAsmPopTertiary(a)
			rtgAsmLoadPrimaryIndexTertiaryScalarOrPointer(a, elem.kind)
			return true
		}
	}
	if left.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, fieldType)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitSelectorAddressSecondary(g, ep, e.left) {
				return false
			}
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
			rtgAsmPopTertiary(a)
			rtgAsmLoadBytePrimaryIndexTertiary(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarValue(elem.kind) && elem.kind != rtgTypePointer {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitSelectorAddressSecondary(g, ep, e.left) {
				return false
			}
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
			rtgAsmPopTertiary(a)
			rtgAsmLoadPrimaryIndexTertiaryScalarOrPointer(a, elem.kind)
			return true
		}
	}
	if left.kind == rtgExprUnary && rtgTokCharIs(p, left.tok, '*') {
		valueType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, valueType)
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarValue(elem.kind) && elem.kind != rtgTypePointer {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushPrimary(a)
			if !rtgEmitIntExpr(g, ep, left.left) {
				return false
			}
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
			rtgAsmPopTertiary(a)
			rtgAsmLoadPrimaryIndexTertiaryScalarOrPointer(a, elem.kind)
			return true
		}
		if t.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitIntExpr(g, ep, left.left) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPopTertiary(a)
		rtgAsmLoadBytePrimaryIndexTertiary(a)
		return true
	}
	if left.kind == rtgExprIndex {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitStringPtrExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmLoadBytePrimaryIndexTertiary(a)
		return true
	}
	return false
}
func rtgFindLocalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}

func rtgFindLocalIndex(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := g.localCount - 1; i >= 0; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}
func rtgStructFieldIndex(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	meta := g.meta
	if typ < 0 || typ >= len(meta.types) {
		return -1
	}
	t := rtgResolveType(meta, typ)
	if t.kind == rtgTypePointer && t.elem > 0 && t.elem < len(meta.types) {
		t = rtgResolveType(meta, t.elem)
	}
	if t.kind != rtgTypeStruct {
		return -1
	}
	fields := meta.fields
	for i := 0; i < t.count; i++ {
		field := fields[t.first+i]
		if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			return t.first + i
		}
	}
	return -1
}
func rtgStructFieldOffset(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return -1
	}
	return g.meta.fields[fieldIndex].offset
}
func rtgStructFieldType(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return 0
	}
	return g.meta.fields[fieldIndex].typ
}
func rtgCompositeStructFieldIndex(g *rtgLinearGen, typ int, field *rtgCompositeField, pos int) int {
	if field.nameEnd > field.nameStart {
		return rtgStructFieldIndex(g, typ, field.nameStart, field.nameEnd)
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind != rtgTypeStruct || pos < 0 || pos >= t.count {
		return -1
	}
	return t.first + pos
}
func rtgFindGlobalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.globals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.globals[i].nameStart, g.globals[i].nameEnd, nameStart, nameEnd) {
			return g.globals[i].offset
		}
	}
	return -1
}
func rtgFindGlobalType(g *rtgLinearGen, nameStart int, nameEnd int) int {
	symIndex := rtgFindMetaGlobalIndex(g.meta, nameStart, nameEnd, rtgTokVar)
	if symIndex >= 0 {
		return g.meta.globals[symIndex].typ
	}
	return 0
}
func rtgFindConstStringToken(g *rtgLinearGen, nameStart int, nameEnd int) int {
	symIndex := rtgFindMetaGlobalIndex(g.meta, nameStart, nameEnd, rtgTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.initStart+1 == s.initEnd && rtgTokIsKind(g.prog, s.initStart, rtgTokString) {
			return s.initStart
		}
	}
	return -1
}
func rtgFindSmallConstByName(g *rtgLinearGen, nameStart int, nameEnd int) int {
	if rtgFindLocalIndex(g, nameStart, nameEnd) >= 0 {
		return -129
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "nil") {
		return 0
	}
	symIndex := rtgFindMetaGlobalIndex(g.meta, nameStart, nameEnd, rtgTokConst)
	if symIndex >= 0 {
		s := &g.meta.globals[symIndex]
		if s.initStart+1 != s.initEnd {
			return -129
		}
		if rtgTokIsKind(g.prog, s.initStart, rtgTokNumber) {
			value := rtgParseIntToken(g.prog, s.initStart)
			if rtgAsmImmFits8Signed(value) {
				return value
			}
		}
		if rtgTokIsKind(g.prog, s.initStart, rtgTokChar) {
			value := rtgParseCharToken(g.prog, s.initStart)
			if rtgAsmImmFits8Signed(value) {
				return value
			}
		}
		return -129
	}
	return -129
}
func rtgAddTypedLocal(g *rtgLinearGen, nameStart int, nameEnd int, typ int) int {
	size := rtgTypeSize(g.meta, typ)
	if size < rtgBackendValueSlotSize {
		size = rtgBackendValueSlotSize
	}
	g.stackUsed = rtgAlignTo8(g.stackUsed + size)
	offset := g.stackUsed
	if g.localCount >= len(g.locals) {
		rtgGrowLocalTable(g)
	}
	g.locals[g.localCount] = rtgLocalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: offset, typ: typ, size: size}
	g.localCount++
	return offset
}

func rtgAddUnnamedLocal(g *rtgLinearGen, typ int) int {
	return rtgAddTypedLocal(g, 0, 0, typ)
}

func rtgGrowLocalTable(g *rtgLinearGen) {
	newCap := len(g.locals) * 2
	if newCap < 64 {
		newCap = 64
	}
	newLocals := make([]rtgLocalInfo, newCap)
	for i := 0; i < g.localCount; i++ {
		newLocals[i] = g.locals[i]
	}
	g.locals = newLocals
}

func rtgZeroLocalAtOffset(g *rtgLinearGen, offset int) {
	a := &g.asm
	size := 8
	typ := rtgTypeInt
	for i := 0; i < g.localCount; i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind == rtgTypeSlice {
		rtgInitEmptySliceStack(g, offset, typ)
		return
	}
	rtgAsmPrimaryImm(a, 0)
	for at := 0; at < size; at += 8 {
		rtgAsmStorePrimaryStack(a, offset-at)
	}
	if t.kind == rtgTypeStruct {
		rtgInitStructSliceFields(g, typ, offset)
	}
}
func rtgInitEmptySliceStack(g *rtgLinearGen, offset int, typ int) {
	a := &g.asm
	t := rtgResolveType(g.meta, typ)
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	if rtgTargetArch == rtgArchAmd64 || rtgTargetArch == rtgArch386 || rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm {
		rtgAsmStoreStackImm(a, offset, 0)
		rtgAsmStorePrimaryStack(a, offset-8)
		rtgAsmStorePrimaryStack(a, offset-16)
		return
	}
	backingSize := 2097152
	if rtgTargetArch == rtgArchWasm32 {
		backingSize = rtgWasm32FallbackSliceBackingSize
	}
	if rtgTargetArch == rtgArchAmd64 {
		rtgEmitArenaAllocPrimary(g, backingSize)
	} else {
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmPrimaryBssAddr(a, backingOff)
	}
	rtgAsmStorePrimaryStack(a, offset)
	rtgAsmStoreStackImm(a, offset-8, 0)
	rtgAsmStoreStackImm(a, offset-16, backingSize/elemSize)
}
func rtgInitStructSliceFields(g *rtgLinearGen, typ int, offset int) {
	t := rtgResolveType(g.meta, typ)
	if t.kind != rtgTypeStruct {
		return
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		fieldOffset := offset - field.offset
		fieldType := rtgResolveType(g.meta, field.typ)
		if fieldType.kind == rtgTypeSlice {
			rtgInitEmptySliceStack(g, fieldOffset, field.typ)
		} else if fieldType.kind == rtgTypeStruct {
			rtgInitStructSliceFields(g, field.typ, fieldOffset)
		}
	}
}
func rtgEmitCopyReturnedStructSliceFields(g *rtgLinearGen, typ int, srcOffset int, destOffset int) bool {
	t := rtgResolveType(g.meta, typ)
	if t.kind != rtgTypeStruct {
		return true
	}
	for i := 0; i < t.count; i++ {
		field := g.meta.fields[t.first+i]
		fieldType := rtgResolveType(g.meta, field.typ)
		fieldSrcOffset := srcOffset - field.offset
		fieldDestOffset := destOffset + field.offset
		if fieldType.kind == rtgTypeSlice {
			rtgAsmLoadPrimaryStack(&g.asm, fieldSrcOffset)
			rtgAsmLoadSecondaryStack(&g.asm, fieldSrcOffset-8)
			rtgAsmLoadTertiaryStack(&g.asm, fieldSrcOffset-16)
			if !rtgEmitCopySliceRegsToArena(g, field.typ) {
				return false
			}
			rtgAsmPushSliceRegs(&g.asm)
			rtgAsmLoadSecondaryStack(&g.asm, g.returnStruct)
			rtgAsmPopStoreSliceMemSecondary(&g.asm, fieldDestOffset)
		} else if fieldType.kind == rtgTypeStruct {
			if !rtgEmitCopyReturnedStructSliceFields(g, field.typ, fieldSrcOffset, fieldDestOffset) {
				return false
			}
		}
	}
	return true
}
func rtgFuncInfoFromCall(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	e := ep.exprs[idx]
	nameStart := e.nameStart
	nameEnd := e.nameEnd
	wantMethod := false
	wantReceiverType := 0
	if e.kind == rtgExprSelector {
		wantMethod = true
		wantReceiverType = rtgInferParsedExprType(g, ep, e.left)
	} else if e.kind != rtgExprIdent {
		return -1
	}
	if len(g.meta.funcBuckets) == 0 {
		return -1
	}
	hash := rtgHashRange(g.prog.src, nameStart, nameEnd)
	i := g.meta.funcBuckets[hash%len(g.meta.funcBuckets)]
	interfaceCandidate := -1
	actualResolved := rtgResolveType(g.meta, wantReceiverType)
	allowInterfaceFallback := wantMethod && actualResolved.kind == rtgTypeNamed && actualResolved.elem == 0
	for i >= 0 {
		f := g.meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			if !wantMethod || rtgMethodReceiverTypeMatches(g.meta, wantReceiverType, f.receiverType) {
				return i
			}
			if allowInterfaceFallback {
				if interfaceCandidate >= 0 {
					return -1
				}
				interfaceCandidate = i
			}
		}
		i = g.meta.funcNext[i]
	}
	return interfaceCandidate
}

func rtgMethodReceiverTypeMatches(meta *rtgMeta, actual int, declared int) bool {
	if actual == declared {
		return true
	}
	if actual == 0 {
		return false
	}
	actualType := rtgResolveType(meta, actual)
	if actualType.kind == rtgTypePointer {
		actual = actualType.elem
	}
	declaredType := rtgResolveType(meta, declared)
	if declaredType.kind == rtgTypePointer {
		declared = declaredType.elem
	}
	return actual == declared
}

// Architecture target dispatch wrappers.
func rtgEmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EmitScalarFunction(g, fnInfoIndex)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EmitScalarFunction(g, fnInfoIndex)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEmitScalarFunction(g, fnInfoIndex)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitScalarFunction(g, fnInfoIndex)
	}
	return rtgAmd64EmitScalarFunction(g, fnInfoIndex)
}
func rtgStoreIncomingCallWord(g *rtgLinearGen, word int, offset int) bool {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32StoreParamWord(g, word, offset)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64StoreParamWord(g, word, offset)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmStoreParamWord(g, word, offset)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386StoreParamWord(g, word, offset)
	}
	return rtgAmd64StoreParamWord(g, word, offset)
}
func rtgAsmPrimaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRaxImm(a, imm)
		return
	}
	rtgAmd64AsmMovRaxImm(a, imm)
}
func rtgAsmPrimaryImm64(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRaxImm64(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRaxImm64(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRaxImm64(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRaxImm64(a, imm)
		return
	}
	rtgAsmEmit16(a, 0xb848)
	rtgAsmEmit64(a, imm)
}
func rtgAsmSecondaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRdxImm(a, imm)
		return
	}
	if imm == 0 {
		rtgAsmEmit16(a, 0xd231)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopSecondary(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			rtgAsmEmit24(a, 0xc2c748)
			rtgAsmEmit32(a, imm)
			return
		}
	}
	rtgAsmEmit16(a, 0xba48)
	rtgAsmEmit64(a, imm)
}
func rtgAsmPrimaryDataAddr(a *rtgAsm, dataOff int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRaxDataAddr(a, dataOff)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRaxDataAddr(a, dataOff)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRaxDataAddr(a, dataOff)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRaxDataAddr(a, dataOff)
		return
	}
	rtgAsmEmit24(a, 0x058d48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, dataOff, 0)
}
func rtgAsmPrimaryBssAddr(a *rtgAsm, bssOff int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRaxBssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRaxBssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRaxBssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRaxBssAddr(a, bssOff)
		return
	}
	rtgAsmEmit24(a, 0x058d48)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}
func rtgAsmScratchBssAddr(a *rtgAsm, bssOff int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovR10BssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovR10BssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovR10BssAddr(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovR10BssAddr(a, bssOff)
		return
	}
	rtgAmd64AsmMovR10BssAddr(a, bssOff)
}
func rtgAsmLoadPrimaryBss(a *rtgAsm, bssOff int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadRaxBss(a, bssOff)
		return
	}
	rtgAmd64AsmLoadRaxBss(a, bssOff)
}
func rtgAsmStorePrimaryBss(a *rtgAsm, bssOff int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreRaxBss(a, bssOff)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmStoreRaxBss(a, bssOff)
		return
	}
	rtgAmd64AsmStoreRaxBss(a, bssOff)
}
func rtgAsmCopyPrimaryToCallWord0(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRdiRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRdiRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRdiRax(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRdiRax(a)
		return
	}
	rtgAmd64AsmMovRdiRax(a)
}
func rtgAsmCopySecondaryToPrimary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRaxRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRaxRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRaxRdx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRaxRdx(a)
		return
	}
	rtgAmd64AsmMovRaxRdx(a)
}
func rtgAsmCopyPrimaryToCallWord1(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovRsiRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovRsiRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovRsiRax(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovRsiRax(a)
		return
	}
	rtgAmd64AsmMovRsiRax(a)
}
func rtgAsmCopyPrimaryToCallWord4(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovR8Rax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovR8Rax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovR8Rax(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovR8Rax(a)
		return
	}
	rtgAmd64AsmMovR8Rax(a)
}
func rtgAsmCopyPrimaryToCallWord5(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmMovR9Rax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMovR9Rax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMovR9Rax(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMovR9Rax(a)
		return
	}
	rtgAmd64AsmMovR9Rax(a)
}
func rtgAsmAddSecondaryTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmAddRdxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmAddRdxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmAddRdxRcx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmAddRdxRcx(a)
		return
	}
	rtgAmd64AsmAddRdxRcx(a)
}
func rtgAsmSyscall(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmSyscall(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmSyscall(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmSyscall(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmSyscall(a)
		return
	}
	rtgAmd64AsmSyscall(a)
}
func rtgAsmPopCallWord0(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmPopRdi(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmPopRdi(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmPopRdi(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmPopRdi(a)
		return
	}
	rtgAmd64AsmPopRdi(a)
}
func rtgAsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmStackMem(a, offset, base, disp8, disp32)
		return
	}
	rtgAmd64AsmStackMem(a, offset, base, disp8, disp32)
}
func rtgAsmAddSecondaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmAddRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmAddRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmAddRdxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmAddRdxImm(a, imm)
		return
	}
	rtgAmd64AsmAddRdxImm(a, imm)
}
func rtgAsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmMemDisp(a, disp, op, disp8, disp32)
		return
	}
	rtgAmd64AsmMemDisp(a, disp, op, disp8, disp32)
}
func rtgAsmLoadQwordPrimaryIndexTertiary8(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadQwordRaxIndexRcx8(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadQwordRaxIndexRcx8(a)
		return
	}
	rtgAmd64AsmLoadQwordRaxIndexRcx8(a)
}
func rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a *rtgAsm, disp int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadQwordRaxIndexRcxDisp(a, disp)
		return
	}
	rtgAmd64AsmLoadQwordRaxIndexRcxDisp(a, disp)
}
func rtgAsmLoadPrimaryMemSecondaryDisp(a *rtgAsm, disp int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadRaxMemRdxDisp(a, disp)
		return
	}
	rtgAmd64AsmLoadRaxMemRdxDisp(a, disp)
}
func rtgAsmLoadPrimaryMemSecondaryDispSize(a *rtgAsm, disp int, size int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadRaxMemRdxDispSize(a, disp, size)
		return
	}
	rtgAmd64AsmLoadRaxMemRdxDispSize(a, disp, size)
}
func rtgAsmLoadBytePrimaryIndexTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadByteRaxIndexRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadByteRaxIndexRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadByteRaxIndexRcx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadByteRaxIndexRcx(a)
		return
	}
	rtgAmd64AsmLoadByteRaxIndexRcx(a)
}
func rtgAsmLoadPrimaryIndexTertiarySize(a *rtgAsm, size int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmLoadRaxIndexRcxSize(a, size)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmLoadRaxIndexRcxSize(a, size)
		return
	}
	rtgAmd64AsmLoadRaxIndexRcxSize(a, size)
}
func rtgAsmStorePrimaryMemSecondaryTertiary8(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxMemRdxRcx8(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreRaxMemRdxRcx8(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreRaxMemRdxRcx8(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmStoreRaxMemRdxRcx8(a)
		return
	}
	rtgAmd64AsmStoreRaxMemRdxRcx8(a)
}
func rtgAsmStorePrimaryMemSecondaryDisp(a *rtgAsm, disp int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmStoreRaxMemRdxDisp(a, disp)
		return
	}
	rtgAmd64AsmStoreRaxMemRdxDisp(a, disp)
}
func rtgAsmStorePrimaryMemSecondaryDispSize(a *rtgAsm, disp int, size int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmStoreRaxMemRdxDispSize(a, disp, size)
		return
	}
	rtgAmd64AsmStoreRaxMemRdxDispSize(a, disp, size)
}
func rtgAsmNormalizePrimaryForKind(a *rtgAsm, kind int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmNormalizeRaxForKind(a, kind)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmNormalizeRaxForKind(a, kind)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmNormalizeRaxForKind(a, kind)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmNormalizeRaxForKind(a, kind)
		return
	}
	rtgAmd64AsmNormalizeRaxForKind(a, kind)
}
func rtgAsmIncMemSecondary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmIncMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmIncMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmIncMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmIncMemRdx(a)
		return
	}
	rtgAmd64AsmIncMemRdx(a)
}
func rtgAsmDecMemSecondary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmDecMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmDecMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmDecMemRdx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmDecMemRdx(a)
		return
	}
	rtgAmd64AsmDecMemRdx(a)
}
func rtgAsmBoolNotPrimary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmBoolNotRax(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmBoolNotRax(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmBoolNotRax(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmBoolNotRax(a)
		return
	}
	rtgAmd64AsmBoolNotRax(a)
}
func rtgAsmCmpPrimaryImm8(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmCmpRaxImm8(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmCmpRaxImm8(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmCmpRaxImm8(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmCmpRaxImm8(a, imm)
		return
	}
	rtgAmd64AsmCmpRaxImm8(a, imm)
}
func rtgAsmAddPrimaryTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmAddRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmAddRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmAddRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmAddRaxRcx(a)
		return
	}
	rtgAmd64AsmAddRaxRcx(a)
}
func rtgAsmSubPrimaryTertiary(a *rtgAsm) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmSubRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmSubRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmSubRaxRcx(a)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmSubRaxRcx(a)
		return
	}
	rtgAmd64AsmSubRaxRcx(a)
}
func rtgAsmShlTertiaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmShlRcxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmShlRcxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmShlRcxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmShlRcxImm(a, imm)
		return
	}
	rtgAmd64AsmShlRcxImm(a, imm)
}
func rtgAsmShlPrimaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmShlRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmShlRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmShlRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmShlRaxImm(a, imm)
		return
	}
	rtgAmd64AsmShlRaxImm(a, imm)
}
func rtgAsmSarPrimaryImm(a *rtgAsm, imm int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmSarRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmSarRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmSarRaxImm(a, imm)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmSarRaxImm(a, imm)
		return
	}
	rtgAmd64AsmSarRaxImm(a, imm)
}
func rtgAsmDivLeftTertiaryRightPrimary(a *rtgAsm, mod bool) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmDivLeftRcxRightRax(a, mod)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmDivLeftRcxRightRax(a, mod)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmDivLeftRcxRightRax(a, mod)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmDivLeftRcxRightRax(a, mod)
		return
	}
	rtgAmd64AsmDivLeftRcxRightRax(a, mod)
}
func rtgAsmCmpTertiaryPrimarySet(a *rtgAsm, setcc int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32AsmCmpRcxRaxSet(a, setcc)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64AsmCmpRcxRaxSet(a, setcc)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmAsmCmpRcxRaxSet(a, setcc)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386AsmCmpRcxRaxSet(a, setcc)
		return
	}
	rtgAmd64AsmCmpRcxRaxSet(a, setcc)
}
func rtgEmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep *rtgExprParse, idx int, matchLabel int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, idx, matchLabel)
	}
	return rtgAmd64EmitSwitchStringCaseTest(g, valueOffset, lenOffset, ep, idx, matchLabel)
}
func rtgEmitPrimaryTertiaryOp(g *rtgLinearGen, tok int) bool {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EmitRaxRcxOp(g, tok)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EmitRaxRcxOp(g, tok)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEmitRaxRcxOp(g, tok)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitRaxRcxOp(g, tok)
	}
	return rtgAmd64EmitRaxRcxOp(g, tok)
}
func rtgEmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitCompareJump(g, ep, e, label, jumpIfTrue)
	}
	return rtgAmd64EmitCompareJump(g, ep, e, label, jumpIfTrue)
}
func rtgEmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent && rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
		rtgAsmPrimaryImm(&g.asm, 0)
		rtgAsmSecondaryImm(&g.asm, 0)
		return true
	}
	if rtgExprIsErrorStringCall(g, ep, idx) {
		callee := &ep.exprs[e.left]
		return rtgEmitStringValueRegs(g, ep, callee.left)
	}
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		if rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, argIndex)) {
			return rtgEmitStringValueRegs(g, ep, argIndex)
		}
		argType := rtgInferParsedExprType(g, ep, argIndex)
		argResolved := rtgResolveType(g.meta, argType)
		if argResolved.kind == rtgTypeSlice {
			elem := rtgResolveType(g.meta, argResolved.elem)
			if elem.kind == rtgTypeByte {
				return rtgEmitByteSliceStringCopyValueRegs(g, ep, argIndex)
			}
		}
	}
	if e.kind == rtgExprBinary && rtgTokCharIs(g.prog, e.tok, '+') && rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, idx)) {
		return rtgEmitStringConcatValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(g.prog, e.tok, '*') {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(g.meta, valueType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmCopyPrimaryToSecondary(&g.asm)
		rtgAsmLoadPrimaryMemSecondaryDisp(&g.asm, 0)
		rtgAsmPushPrimary(&g.asm)
		rtgAsmLoadPrimaryMemSecondaryDisp(&g.asm, 8)
		rtgAsmCopyPrimaryToSecondary(&g.asm)
		rtgAsmPopPrimary(&g.asm)
		return true
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitStringValueRegs(g, ep, idx)
	}
	return rtgGenericEmitStringValueRegs(g, ep, idx)
}
func rtgGenericEmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog, e.tok)
		msgOff := rtgAddStringData(g, msg)
		msgLen := len(msg)
		rtgAsmPrimaryDataAddr(a, msgOff)
		rtgAsmSecondaryImm(a, msgLen)
		return true
	}
	if e.kind == rtgExprSlice {
		return rtgEmitStringSliceValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadPrimaryStack(a, g.locals[localIndex].offset)
			rtgAsmLoadSecondaryStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadPrimaryBss(a, globalOffset)
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryBss(a, globalOffset+8)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopPrimary(a)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog, constTok)
			msgOff := rtgAddStringData(g, msg)
			msgLen := len(msg)
			rtgAsmPrimaryDataAddr(a, msgOff)
			rtgAsmSecondaryImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmShlTertiaryImm(a, 4)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmLoadQwordPrimaryIndexTertiaryDisp(a, 0)
		rtgAsmAddSecondaryTertiary(a)
		if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushPrimary(a)
			rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
			rtgAsmCopyPrimaryToSecondary(a)
			rtgAsmPopPrimary(a)
		} else {
			rtgAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		}
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressSecondary(g, ep, idx) {
			return false
		}
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 0)
		rtgAsmPushPrimary(a)
		rtgAsmLoadPrimaryMemSecondaryDisp(a, 8)
		rtgAsmCopyPrimaryToSecondary(a)
		rtgAsmPopPrimary(a)
		return true
	}
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(g.prog, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		argType := rtgInferParsedExprType(g, ep, argIndex)
		argResolved := rtgResolveType(meta, argType)
		if argResolved.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, argResolved.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		if !rtgEmitSlicePtrLen(g, ep, argIndex) {
			return false
		}
		rtgAsmPushTertiary(a)
		rtgAsmPopSecondary(a)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgStringHeapOffsets(g *rtgLinearGen) {
	if g.stringHeapReady != 0 {
		return
	}
	g.stringHeapReady = 1
	g.stringHeapOff = g.asm.bssSize
	g.stringHeapEndOff = g.stringHeapOff + 8
	g.stringHeapDataOff = g.stringHeapOff + 16
	g.asm.bssSize += 16 + rtgStringArenaSize()
}

func rtgStringArenaSize() int {
	if rtgCompilerArenaSize > 0 {
		return rtgCompilerArenaSize
	}
	if rtgTargetArch == rtgArchAmd64 || rtgTargetArch == rtgArch386 || rtgTargetArch == rtgArchWasm32 {
		return 1073741824
	}
	return 805306368
}

func rtgEmitArenaAllocPrimary(g *rtgLinearGen, size int) {
	label := rtgEnsureArenaAllocHelper(g)
	rtgAsmPrimaryImm(&g.asm, size)
	rtgAsmCallLabel(&g.asm, label)
}

func rtgEmitArenaAllocStackPrimary(g *rtgLinearGen, sizeOff int) {
	label := rtgEnsureArenaAllocHelper(g)
	rtgAsmLoadPrimaryStack(&g.asm, sizeOff)
	rtgAsmCallLabel(&g.asm, label)
}

func rtgEnsureArenaAllocHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.arenaAllocEmitted {
		return g.arenaAllocLabel
	}
	g.arenaAllocEmitted = true
	g.arenaAllocLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	readyLabel := rtgAsmNewLabel(a)
	oomLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.arenaAllocLabel)
	rtgStringHeapOffsets(g)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, readyLabel)
	rtgAsmPrimaryBssAddr(a, g.stringHeapDataOff)
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	rtgAsmMarkLabel(a, readyLabel)
	rtgAsmLoadPrimaryBss(a, g.stringHeapOff)
	rtgAsmPushPrimary(a)
	rtgAsmPushPrimary(a)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmPushPrimary(a)
	rtgAsmPopTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9d)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, oomLabel)
	rtgAsmPushTertiary(a)
	rtgAsmPrimaryBssAddr(a, g.stringHeapDataOff+rtgStringArenaSize())
	rtgAsmPopTertiary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9e)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, oomLabel)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryBss(a, g.stringHeapOff)
	rtgAsmPopPrimary(a)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, oomLabel)
	rtgAsmPopPrimary(a)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.arenaAllocLabel
}

const rtgPrintIntBufferSize = 24

func rtgEmitPrintIntBufferByte(g *rtgLinearGen) {
	a := &g.asm
	lenOff := g.printIntBufferOff + rtgPrintIntBufferSize + 8
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, lenOff)
	rtgAsmPushPrimary(a)
	rtgAsmPrimaryImm(a, rtgPrintIntBufferSize-1)
	rtgAsmPopTertiary(a)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPrimaryBssAddr(a, g.printIntBufferOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStoreByteMemSecondaryTertiary(a)
	rtgAsmLoadPrimaryBss(a, lenOff)
	rtgAsmIncPrimary(a)
	rtgAsmStorePrimaryBss(a, lenOff)
}

func rtgEnsurePrintIntHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.printIntEmitted {
		return g.printIntLabel
	}
	g.printIntEmitted = true
	g.printIntBufferOff = a.bssSize
	a.bssSize += rtgPrintIntBufferSize + 24
	g.printIntLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	positiveDigitLabel := rtgAsmNewLabel(a)
	digitReadyLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	valueOff := g.printIntBufferOff + rtgPrintIntBufferSize
	lenOff := valueOff + 8
	negativeOff := lenOff + 8
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.printIntLabel)
	rtgAsmStorePrimaryBss(a, valueOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9c)
	rtgAsmStorePrimaryBss(a, negativeOff)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmStorePrimaryBss(a, lenOff)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadPrimaryBss(a, valueOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPrimaryImm(a, 10)
	rtgAsmDivLeftTertiaryRightPrimary(a, true)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, negativeOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, positiveDigitLabel)
	rtgAsmPrimaryImm(a, '0')
	rtgAsmPopTertiary(a)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmJmpLabel(a, digitReadyLabel)
	rtgAsmMarkLabel(a, positiveDigitLabel)
	rtgAsmPrimaryImm(a, '0')
	rtgAsmPopTertiary(a)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmMarkLabel(a, digitReadyLabel)
	rtgEmitPrintIntBufferByte(g)
	rtgAsmLoadPrimaryBss(a, valueOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPrimaryImm(a, 10)
	rtgAsmDivLeftTertiaryRightPrimary(a, false)
	rtgAsmStorePrimaryBss(a, valueOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmLoadPrimaryBss(a, negativeOff)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmPrimaryImm(a, '-')
	rtgEmitPrintIntBufferByte(g)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadPrimaryBss(a, lenOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmCopySecondaryToPrimary(a)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmPrimaryBssAddr(a, g.printIntBufferOff+rtgPrintIntBufferSize)
	rtgAsmSubPrimaryTertiary(a)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.printIntLabel
}

func rtgAmd64SliceBackingSize(elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if rtgCompilerFixedTarget != 0 {
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

func rtgAmd64ArenaSliceBackingSize(elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if rtgCompilerFixedTarget != 0 {
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

func rtgStaticSliceBackingSize(needSize int, elemSize int) int {
	if elemSize < 1 {
		elemSize = 8
	}
	if needSize < elemSize {
		needSize = elemSize
	}
	return rtgAlignTo8(needSize)
}

func rtgEmitByteSliceStringCopyValueRegs(g *rtgLinearGen, ep *rtgExprParse, argIndex int) bool {
	a := &g.asm
	if !rtgEmitSlicePtrLen(g, ep, argIndex) {
		return false
	}
	srcOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	lenOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	destOff := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStorePrimaryStack(a, srcOff)
	rtgAsmPushTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryStack(a, lenOff)
	if rtgTargetArch != rtgArchAmd64 {
		indexOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgEmitArenaAllocStackPrimary(g, lenOff)
		rtgAsmStorePrimaryStack(a, destOff)
		rtgAsmStoreStackImm(a, indexOff, 0)
		loopLabel := rtgAsmNewLabel(a)
		doneLabel := rtgAsmNewLabel(a)
		rtgAsmMarkLabel(a, loopLabel)
		rtgAsmJgeStackStack(a, indexOff, lenOff, doneLabel)
		rtgAsmLoadPrimaryStack(a, srcOff)
		rtgAsmLoadTertiaryStack(a, indexOff)
		rtgAsmLoadPrimaryIndexTertiarySize(a, 1)
		rtgAsmPushPrimary(a)
		rtgAsmLoadSecondaryStack(a, destOff)
		rtgAsmLoadTertiaryStack(a, indexOff)
		rtgAsmPopPrimary(a)
		rtgAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
		rtgAsmIncStack(a, indexOff)
		rtgAsmJmpLabel(a, loopLabel)
		rtgAsmMarkLabel(a, doneLabel)
		rtgAsmLoadPrimaryStack(a, destOff)
		rtgAsmLoadSecondaryStack(a, lenOff)
		return true
	}
	rtgEmitArenaAllocStackPrimary(g, lenOff)
	rtgAsmStorePrimaryStack(a, destOff)
	rtgAsmLoadPrimaryStack(a, destOff)
	rtgAsmCopyPrimaryToCallWord0(a)
	rtgAsmLoadPrimaryStack(a, srcOff)
	rtgAsmCopyPrimaryToCallWord1(a)
	rtgAsmLoadTertiaryStack(a, lenOff)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmLoadPrimaryStack(a, destOff)
	rtgAsmLoadSecondaryStack(a, lenOff)
	return true
}

func rtgEmitStringConcatValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	byteSliceType := rtgAddType(g.meta, rtgTypeSlice, rtgTypeByte, 0, 0, rtgBackendSliceValueSize, 0, 0)
	offset := rtgAddUnnamedLocal(g, byteSliceType)
	rtgZeroLocalAtOffset(g, offset)
	loc := rtgSliceLocation{offset: offset, typ: byteSliceType, ok: true}
	if !rtgEmitStringConcatIntoLocation(g, ep, idx, &loc) {
		return false
	}
	if rtgTargetArch == rtgArchAmd64 {
		destOff := rtgAddUnnamedLocal(g, rtgTypeInt)
		rtgEmitArenaAllocStackPrimary(g, offset-8)
		rtgAsmStorePrimaryStack(a, destOff)
		rtgAsmLoadPrimaryStack(a, destOff)
		rtgAsmCopyPrimaryToCallWord0(a)
		rtgAsmLoadPrimaryStack(a, offset)
		rtgAsmCopyPrimaryToCallWord1(a)
		rtgAsmLoadTertiaryStack(a, offset-8)
		rtgAsmEmit16(a, 0xa4f3)
		rtgAsmLoadPrimaryStack(a, destOff)
		rtgAsmLoadSecondaryStack(a, offset-8)
		return true
	}
	rtgAsmLoadPrimaryStack(a, offset)
	rtgAsmPushPrimary(a)
	rtgAsmLoadSecondaryStack(a, offset-8)
	rtgAsmPopPrimary(a)
	return true
}

func rtgEmitStringConcatIntoLocation(g *rtgLinearGen, ep *rtgExprParse, idx int, loc *rtgSliceLocation) bool {
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary && rtgTokCharIs(g.prog, e.tok, '+') && rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, idx)) {
		if !rtgEmitStringConcatIntoLocation(g, ep, e.left, loc) {
			return false
		}
		return rtgEmitStringConcatIntoLocation(g, ep, e.right, loc)
	}
	return rtgEmitAppendStringBytesToLocation(g, ep, idx, ep, loc)
}

func rtgEmitAppendStringBytesToLocation(g *rtgLinearGen, ep *rtgExprParse, idx int, locEp *rtgExprParse, loc *rtgSliceLocation) bool {
	a := &g.asm
	srcPtr := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcLen := rtgAddUnnamedLocal(g, rtgTypeInt)
	srcIndex := rtgAddUnnamedLocal(g, rtgTypeInt)
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmStorePrimaryStack(a, srcPtr)
	rtgAsmStoreSecondaryStack(a, srcLen)
	rtgAsmStoreStackImm(a, srcIndex, 0)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmJgeStackStack(a, srcIndex, srcLen, doneLabel)
	rtgAsmLoadPrimaryStack(a, srcPtr)
	rtgAsmLoadTertiaryStack(a, srcIndex)
	rtgAsmLoadPrimaryIndexTertiarySize(a, 1)
	rtgAsmPushPrimary(a)
	if !rtgEmitAppendDestPrimary(g, locEp, loc, 1) {
		return false
	}
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryDispSize(a, 0, 1)
	rtgAsmIncStack(a, srcIndex)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	return true
}

func rtgEmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitCompositeFieldToMem(g, ep, idx, fieldType, addrOffset, fieldOffset)
	}
	return rtgAmd64EmitCompositeFieldToMem(g, ep, idx, fieldType, addrOffset, fieldOffset)
}
func rtgEmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx >= 0 && idx < len(ep.exprs) && ep.exprs[idx].kind == rtgExprSelector {
		resultType := g.meta.funcs[g.currentFunc].resultType
		size := rtgTypeSize(g.meta, resultType)
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(g.meta, valueType) || rtgTypeSize(g.meta, valueType) != size || g.returnStruct <= 0 {
			return false
		}
		tempOffset := rtgAddUnnamedLocal(g, resultType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		rtgAsmLoadSecondaryStack(&g.asm, g.returnStruct)
		rtgEmitCopyStackToMemSecondary(g, tempOffset, 0, size)
		return true
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitStructReturnExpr(g, ep, idx)
	}
	return rtgAmd64EmitStructReturnExpr(g, ep, idx)
}
func rtgEmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount == 1 {
		conversionType := rtgConversionTypeFromExpr(g, ep, e.left)
		if conversionType != 0 {
			resolved := rtgResolveType(g.meta, conversionType)
			if rtgTypeKindIsScalarValue(resolved.kind) {
				return rtgEmitScalarExprForKind(g, ep, ep.args[e.firstArg], resolved.kind)
			}
		}
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitNamedConversionCall(g, ep, idx)
	}
	return rtgAmd64EmitNamedConversionCall(g, ep, idx)
}
func rtgLinearMarkFunc(g *rtgLinearGen, fnIndex int) {
	if fnIndex < 0 || fnIndex >= len(g.funcReachable) {
		return
	}
	if g.funcReachable[fnIndex] {
		return
	}
	g.funcReachable[fnIndex] = true
	g.funcQueue = append(g.funcQueue, fnIndex)
	if rtgCompilerStripSymbols && rtgTargetArch != rtgArchWasm32 {
		return
	}
	src := g.meta.prog.src
	nameStart := g.meta.funcs[fnIndex].nameStart
	nameEnd := g.meta.funcs[fnIndex].nameEnd
	rtgAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func rtgInitFuncQueue(g *rtgLinearGen, count int) {
	g.funcReachable = make([]bool, count, count)
	for i := 0; i < count; i++ {
		g.funcReachable[i] = false
	}
	g.funcQueue = make([]int, 0, count)
}

func rtgEmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	rtgLinearMarkFunc(g, fnIndex)
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitCallWithWordCount(g, fnIndex, wordCount)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		rtgAarch64EmitCallWithWordCount(g, fnIndex, wordCount)
		return
	}
	if rtgTargetArch == rtgArchArm {
		rtgArmEmitCallWithWordCount(g, fnIndex, wordCount)
		return
	}
	if rtgTargetArch == rtgArch386 {
		rtg386EmitCallWithWordCount(g, fnIndex, wordCount)
		return
	}
	rtgAmd64EmitCallWithWordCount(g, fnIndex, wordCount)
}
func rtgExprIsPointerCompositeLiteral(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind != rtgExprUnary || !rtgTokCharIs(g.prog, e.tok, '&') || e.left < 0 || e.left >= len(ep.exprs) {
		return false
	}
	return ep.exprs[e.left].kind == rtgExprComposite
}
func rtgEmitPointerCompositeLiteral(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	innerIndex := e.left
	inner := &ep.exprs[innerIndex]
	elemType := rtgInferParsedExprType(g, ep, innerIndex)
	resolved := rtgResolveType(g.meta, elemType)
	if resolved.kind != rtgTypeStruct {
		return false
	}
	size := rtgTypeSize(g.meta, elemType)
	if size <= 0 {
		return false
	}

	a := &g.asm
	sizeOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	addrOffset := rtgAddUnnamedLocal(g, rtgTypeInt)
	rtgAsmStoreStackImm(a, sizeOffset, size)
	rtgEmitPersistentAllocToPrimary(g, sizeOffset)
	rtgAsmStorePrimaryStack(a, addrOffset)

	// Persistent storage begins in zero-filled BSS, but explicitly initialize
	// every value slot so this remains correct if the allocator later reuses
	// storage.
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPrimaryImm(a, 0)
	for at := 0; at < size; at += rtgBackendValueSlotSize {
		rtgAsmStorePrimaryMemSecondaryDisp(a, at)
	}
	for i := 0; i < inner.argCount; i++ {
		field := ep.fields[inner.firstArg+i]
		fieldIndex := rtgCompositeStructFieldIndex(g, elemType, &field, i)
		if fieldIndex < 0 {
			return false
		}
		fieldInfo := &g.meta.fields[fieldIndex]
		if fieldInfo.typ == 0 || !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldInfo.typ, addrOffset, fieldInfo.offset) {
			return false
		}
	}
	rtgAsmLoadPrimaryStack(a, addrOffset)
	return true
}
func rtgEmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgExprIsPointerCompositeLiteral(g, ep, idx) {
		return rtgEmitPointerCompositeLiteral(g, ep, idx)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitIntExpr(g, ep, idx)
	}
	return rtgAmd64EmitIntExpr(g, ep, idx)
}

func rtgEmitScalarExprForKind(g *rtgLinearGen, ep *rtgExprParse, idx int, destKind int) bool {
	source := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	if destKind == rtgTypeFloat64 && source.kind != rtgTypeFloat64 {
		rtgAsmShlPrimaryImm(&g.asm, 2)
	} else if destKind != rtgTypeFloat64 && source.kind == rtgTypeFloat64 {
		rtgAsmSarPrimaryImm(&g.asm, 2)
	}
	rtgAsmNormalizePrimaryForKind(&g.asm, destKind)
	return true
}

func rtgArrayBuiltinCount(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr) int {
	t := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, ep.args[e.firstArg]))
	if t.kind == rtgTypeArray {
		return t.count
	}
	return -1
}

func rtgEmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EmitFloatBinaryExpr(g, ep, idx)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EmitFloatBinaryExpr(g, ep, idx)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEmitFloatBinaryExpr(g, ep, idx)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitFloatBinaryExpr(g, ep, idx)
	}
	return rtgAmd64EmitFloatBinaryExpr(g, ep, idx)
}
func rtgEmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	if rtgTargetArch == rtgArchArm {
		return rtgArmEmitSliceSlotAddrs(g, locEp, loc, elemSize)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitSliceSlotAddrs(g, locEp, loc, elemSize)
	}
	return rtgAmd64EmitSliceSlotAddrs(g, locEp, loc, elemSize)
}
func rtgEnsureAppendAddrHelper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureAppendAddrHelper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureAppendAddrHelper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureAppendAddrHelper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureAppendAddrHelper(g)
	}
	return rtgAmd64EnsureAppendAddrHelper(g)
}
func rtgEnsureAppend8Helper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureAppend8Helper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureAppend8Helper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureAppend8Helper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureAppend8Helper(g)
	}
	return rtgAmd64EnsureAppend8Helper(g)
}
func rtgEnsureAppend64Helper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureAppend64Helper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureAppend64Helper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureAppend64Helper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureAppend64Helper(g)
	}
	return rtgAmd64EnsureAppend64Helper(g)
}
func rtgEnsureAppendBytesHelper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureAppendBytesHelper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureAppendBytesHelper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureAppendBytesHelper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureAppendBytesHelper(g)
	}
	return rtgAmd64EnsureAppendBytesHelper(g)
}
func rtgEnsureCopyWordsHelper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureCopyWordsHelper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureCopyWordsHelper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureCopyWordsHelper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureCopyWordsHelper(g)
	}
	return rtgAmd64EnsureCopyWordsHelper(g)
}
func rtgEnsureStringEqualHelper(g *rtgLinearGen) int {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32EnsureStringEqualHelper(g)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64EnsureStringEqualHelper(g)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmEnsureStringEqualHelper(g)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EnsureStringEqualHelper(g)
	}
	return rtgAmd64EnsureStringEqualHelper(g)
}
func rtgEmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitIndexedStructField(g, ep, indexIdx, fieldStart, fieldEnd)
	}
	return rtgAmd64EmitIndexedStructField(g, ep, indexIdx, fieldStart, fieldEnd)
}
func rtgEmitStringPtrExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitStringPtrExpr(g, ep, idx)
	}
	return rtgAmd64EmitStringPtrExpr(g, ep, idx)
}

func rtgExprIsErrorStringCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 {
		return false
	}
	if idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind != rtgExprCall {
		return false
	}
	if e.argCount != 0 {
		return false
	}
	if e.left < 0 {
		return false
	}
	if e.left >= len(ep.exprs) {
		return false
	}
	callee := &ep.exprs[e.left]
	if callee.kind != rtgExprSelector {
		return false
	}
	if !rtgBytesEqualText(g.prog.src, callee.nameStart, callee.nameEnd, "Error") {
		return false
	}
	return rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, callee.left))
}

func rtgEmitSelectorAddressSecondary(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArchAmd64 {
		return rtgAmd64EmitSelectorAddressRdx(g, ep, idx)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitSelectorAddressRdx(g, ep, idx)
	}
	return rtgAmd64EmitSelectorAddressRdx(g, ep, idx)
}
