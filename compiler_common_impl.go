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

const rtgArchAmd64 = 1
const rtgArch386 = 2
const rtgArchAarch64 = 3
const rtgArchArm = 4
const rtgArchWasm32 = 5

const rtgOSLinux = 1
const rtgOSWindows = 2

var rtgTargetArch int = rtgArchAmd64
var rtgTargetOS int = rtgOSLinux
var rtgNativeIntSize int = 8
var rtgCurrentTarget int = rtgTargetLinuxAmd64

func rtgSetTarget(target int) {
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
	name  []byte
	label int
}

type rtgAsm struct {
	code       []byte
	labelPos   []int
	labelSet   []bool
	relocs     []rtgLabelRef
	absRelocs  []rtgAbsRef
	symbols    []rtgAsmSymbol
	data       []byte
	bssSize    int
	codeOffset int
	dataOffset int
}

const rtgWasm32FallbackSliceBackingSize = 4096

func rtgAsmInit(a *rtgAsm) {
	var code []byte
	var labelPos []int
	var labelSet []bool
	var relocs []rtgLabelRef
	var absRelocs []rtgAbsRef
	var symbols []rtgAsmSymbol
	var data []byte
	if rtgTargetArch == rtgArchWasm32 {
		code = make([]byte, 0, 655360)
		labelPos = make([]int, 0, 6144)
		labelSet = make([]bool, 0, 6144)
		relocs = make([]rtgLabelRef, 0, 12288)
		absRelocs = make([]rtgAbsRef, 0, 2048)
		symbols = make([]rtgAsmSymbol, 0, 2048)
	} else {
		code = make([]byte, 0, 786432)
		labelPos = make([]int, 0, 8192)
		labelSet = make([]bool, 0, 8192)
		relocs = make([]rtgLabelRef, 0, 24576)
		absRelocs = make([]rtgAbsRef, 0, 4096)
		symbols = make([]rtgAsmSymbol, 0, 4096)
	}
	data = make([]byte, 0, 16384)
	a.code = code
	a.labelPos = labelPos
	a.labelSet = labelSet
	a.relocs = relocs
	a.absRelocs = absRelocs
	a.symbols = symbols
	a.data = data
	a.bssSize = 0
	a.codeOffset = 0
	a.dataOffset = 0
}

func rtgAsmNewLabel(a *rtgAsm) int {
	a.labelPos = append(a.labelPos, 0)
	a.labelSet = append(a.labelSet, false)
	label := len(a.labelPos) - 1
	return label
}

func rtgAsmMarkLabel(a *rtgAsm, label int) {
	if label >= 0 && label < len(a.labelPos) {
		codeLen := len(a.code)
		a.labelPos[label] = codeLen
		a.labelSet[label] = true
	}
}

func rtgAsmEmit8(a *rtgAsm, v int) {
	a.code = append(a.code, byte(v))
}

func rtgAsmEmit2(a *rtgAsm, v0 int, v1 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
}

func rtgAsmEmit3(a *rtgAsm, v0 int, v1 int, v2 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
	a.code = append(a.code, byte(v2))
}

func rtgAsmEmit4(a *rtgAsm, v0 int, v1 int, v2 int, v3 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
	a.code = append(a.code, byte(v2))
	a.code = append(a.code, byte(v3))
}

func rtgAsmEmit5(a *rtgAsm, v0 int, v1 int, v2 int, v3 int, v4 int) {
	a.code = append(a.code, byte(v0))
	a.code = append(a.code, byte(v1))
	a.code = append(a.code, byte(v2))
	a.code = append(a.code, byte(v3))
	a.code = append(a.code, byte(v4))
}

func rtgAsmAddAbsReloc(a *rtgAsm, at int, off int, kind int) {
	a.absRelocs = append(a.absRelocs, rtgAbsRef{at: at, off: off, kind: kind})
}

func rtgAsmAddReloc(a *rtgAsm, at int, label int) {
	a.relocs = append(a.relocs, rtgLabelRef{at: at, label: label})
}

func rtgAsmAddFuncSymbol(a *rtgAsm, src []byte, nameStart int, nameEnd int, label int) {
	var name []byte
	for i := nameStart; i < nameEnd; i++ {
		name = append(name, src[i])
	}
	a.symbols = append(a.symbols, rtgAsmSymbol{name: name, label: label})
}

func rtgAsmEmit32(a *rtgAsm, v int) {
	a.code = rtgAppend32(a.code, v)
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

func rtgAsmPatch(a *rtgAsm) {
	if rtgTargetArch == rtgArchArm {
		for i := 0; i < len(a.relocs); i++ {
			r := a.relocs[i]
			if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
				target := a.labelPos[r.label]
				disp := target - (r.at + 8)
				insn := rtgGet32At(a.code, r.at)
				if (insn & 0x0e000000) == 0x0a000000 {
					rtgPut32At(a.code, r.at, (insn&0xff000000)|((disp/4)&0x00ffffff))
				}
			}
		}
		a.dataOffset = a.codeOffset + len(a.code)
		return
	}
	if rtgTargetArch == rtgArchAarch64 {
		for i := 0; i < len(a.relocs); i++ {
			r := a.relocs[i]
			if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
				target := a.labelPos[r.label]
				disp := target - r.at
				insn := rtgGet32At(a.code, r.at)
				if (insn & 0xfc000000) == 0x94000000 {
					rtgPut32At(a.code, r.at, 0x94000000|((disp/4)&0x03ffffff))
				} else if (insn & 0xfc000000) == 0x14000000 {
					rtgPut32At(a.code, r.at, 0x14000000|((disp/4)&0x03ffffff))
				} else if (insn & 0xff000010) == 0x54000000 {
					rtgPut32At(a.code, r.at, (insn&0xff00001f)|(((disp/4)&0x7ffff)<<5))
				}
			}
		}
		a.dataOffset = a.codeOffset + len(a.code)
		return
	}
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			target := a.labelPos[r.label]
			disp := target - (r.at + 4)
			rtgPut32At(a.code, r.at, disp)
		}
	}
	a.dataOffset = a.codeOffset + len(a.code)
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		next := a.codeOffset + r.at + 4
		disp := target - next
		rtgPut32At(a.code, r.at, disp)
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
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	out = append(out, byte(v>>16))
	out = append(out, byte(v>>24))
	out = append(out, byte(v>>32))
	out = append(out, byte(v>>40))
	out = append(out, byte(v>>48))
	out = append(out, byte(v>>56))
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
	text := s
	for i := 0; i < len(text); i++ {
		out = append(out, text[i])
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
		if s.label >= 0 && s.label < len(a.labelPos) && a.labelSet[s.label] {
			nameOff := len(sec.strtab)
			sec.strtab = rtgAppendBytesZ(sec.strtab, s.name)
			value := base + a.codeOffset + a.labelPos[s.label]
			sec.symtab = rtgAppendElf64Sym(sec.symtab, nameOff, 18, 1, value, 0)
		}
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
		if s.label >= 0 && s.label < len(a.labelPos) && a.labelSet[s.label] {
			nameOff := len(sec.strtab)
			sec.strtab = rtgAppendBytesZ(sec.strtab, s.name)
			value := base + a.codeOffset + a.labelPos[s.label]
			sec.symtab = rtgAppendElf32Sym(sec.symtab, nameOff, 18, 1, value, 0)
		}
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

type rtgWinImportLayout struct {
	importRVA    int
	importSize   int
	kernelIATRVA int
	thunkSize    int
}

func rtgWinImportCount() int {
	return 8
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
	if id == rtgWinImportExitProcess {
		return "ExitProcess"
	}
	return ""
}

func rtgWinImportIATRVA(layout rtgWinImportLayout, id int) int {
	return layout.kernelIATRVA + (id-1)*layout.thunkSize
}

func rtgAsmAddWinImportReloc(a *rtgAsm, at int, importID int) {
	rtgAsmAddAbsReloc(a, at, importID, rtgAbsWinImportReloc)
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
	kernelCount := rtgWinImportCount()
	thunkSize := 4
	if is64 {
		thunkSize = 8
	}
	dataRVA := a.dataOffset
	if dataRVA == 0 {
		dataRVA = a.codeOffset + len(a.code)
	}
	importOff := rtgAlignValue(len(a.data), thunkSize)
	a.data = rtgAppendUntil(a.data, importOff)
	descOff := importOff
	kernelILTOff := descOff + 40
	kernelIATOff := kernelILTOff + (kernelCount+1)*thunkSize
	nameOff := kernelIATOff + (kernelCount+1)*thunkSize
	a.data = rtgAppendUntil(a.data, nameOff)

	var nameOffsets []int
	for id := 1; id <= rtgWinImportCount(); id++ {
		nameOffsets = append(nameOffsets, len(a.data))
		a.data = rtgAppend16(a.data, 0)
		a.data = rtgAppendStringZ(a.data, rtgWinImportName(id))
		if len(a.data)%2 != 0 {
			a.data = append(a.data, 0)
		}
	}
	kernelNameOff := len(a.data)
	a.data = rtgAppendStringZ(a.data, "kernel32.dll")

	for i := 0; i < kernelCount; i++ {
		nameRVA := dataRVA + nameOffsets[i]
		if is64 {
			rtgPut64U32At(a.data, kernelILTOff+i*thunkSize, nameRVA)
			rtgPut64U32At(a.data, kernelIATOff+i*thunkSize, nameRVA)
		} else {
			rtgPut32At(a.data, kernelILTOff+i*thunkSize, nameRVA)
			rtgPut32At(a.data, kernelIATOff+i*thunkSize, nameRVA)
		}
	}
	rtgPut32At(a.data, descOff, dataRVA+kernelILTOff)
	rtgPut32At(a.data, descOff+12, dataRVA+kernelNameOff)
	rtgPut32At(a.data, descOff+16, dataRVA+kernelIATOff)

	layout.importRVA = dataRVA + importOff
	layout.importSize = len(a.data) - importOff
	layout.kernelIATRVA = dataRVA + kernelIATOff
	layout.thunkSize = thunkSize
	return layout
}

func rtgPut64At(out []byte, at int, v int) {
	out[at] = byte(v)
	out[at+1] = byte(v >> 8)
	out[at+2] = byte(v >> 16)
	out[at+3] = byte(v >> 24)
	out[at+4] = byte(v >> 32)
	out[at+5] = byte(v >> 40)
	out[at+6] = byte(v >> 48)
	out[at+7] = byte(v >> 56)
}

func rtgPut64U32At(out []byte, at int, v int) {
	rtgPut32At(out, at, v)
	rtgPut32At(out, at+4, 0)
}

func rtgAsmPatchWindows(a *rtgAsm, layout rtgWinImportLayout, imageBase int, is64 bool) {
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			target := a.labelPos[r.label]
			disp := target - (r.at + 4)
			rtgPut32At(a.code, r.at, disp)
		}
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
	out = append(out, 'P')
	out = append(out, 'E')
	out = append(out, 0)
	out = append(out, 0)
	out = rtgAppend16(out, 0x8664)
	out = rtgAppend16(out, 2)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 240)
	out = rtgAppend16(out, 0x22)
	out = rtgAppend16(out, 0x20b)
	out = append(out, 1)
	out = append(out, 0)
	out = rtgAppend32(out, textRawSize)
	out = rtgAppend32(out, dataRawSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, entryRVA)
	out = rtgAppend32(out, rtgWinSectionRVA)
	out = rtgAppend64U32(out, rtgWinImageBase)
	out = rtgAppend32(out, rtgWinSectionAlign)
	out = rtgAppend32(out, rtgWinFileAlign)
	out = rtgAppend16(out, 4)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 4)
	out = rtgAppend16(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, sizeOfImage)
	out = rtgAppend32(out, rtgWinHeadersSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 3)
	out = rtgAppend16(out, 0)
	out = rtgAppend64U32(out, 0x100000)
	out = rtgAppend64U32(out, 0x100000)
	out = rtgAppend64U32(out, 0x100000)
	out = rtgAppend64U32(out, 0x1000)
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

func rtgAppendPEHeader32(out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	sizeOfImage := rtgAlignValue(dataRVA+dataVirtualSize, rtgWinSectionAlign)
	out = rtgAppendDOSStub(out)
	out = append(out, 'P')
	out = append(out, 'E')
	out = append(out, 0)
	out = append(out, 0)
	out = rtgAppend16(out, 0x14c)
	out = rtgAppend16(out, 2)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 224)
	out = rtgAppend16(out, 0x102)
	out = rtgAppend16(out, 0x10b)
	out = append(out, 1)
	out = append(out, 0)
	out = rtgAppend32(out, textRawSize)
	out = rtgAppend32(out, dataRawSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, entryRVA)
	out = rtgAppend32(out, rtgWinSectionRVA)
	out = rtgAppend32(out, dataRVA)
	out = rtgAppend32(out, rtgWinImageBase)
	out = rtgAppend32(out, rtgWinSectionAlign)
	out = rtgAppend32(out, rtgWinFileAlign)
	out = rtgAppend16(out, 4)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 4)
	out = rtgAppend16(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, sizeOfImage)
	out = rtgAppend32(out, rtgWinHeadersSize)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 3)
	out = rtgAppend16(out, 0)
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
	out = append(out, 'M')
	out = append(out, 'Z')
	out = rtgAppend16(out, 0x90)
	out = rtgAppend16(out, 3)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 4)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0xffff)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0xb8)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0x40)
	out = rtgAppend16(out, 0)
	out = rtgAppendUntil(out, 0x3c)
	out = rtgAppend32(out, 0x80)
	out = rtgAppendUntil(out, 0x40)
	out = append(out, 0x0e)
	out = append(out, 0x1f)
	out = append(out, 0xba)
	out = append(out, 0x0e)
	out = append(out, 0x00)
	out = append(out, 0xb4)
	out = append(out, 0x09)
	out = append(out, 0xcd)
	out = append(out, 0x21)
	out = append(out, 0xb8)
	out = append(out, 0x01)
	out = append(out, 0x4c)
	out = append(out, 0xcd)
	out = append(out, 0x21)
	out = rtgAppendStringZ(out, "This program cannot be run in DOS mode.\r\r\n$")
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

type rtgToken struct {
	kind  int
	start int
	end   int
	line  int
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
	toks  []rtgToken
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
	prog  *rtgProgram
	stmts []rtgStmt
	ok    bool
}

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
	declIndex    int
	nameStart    int
	nameEnd      int
	firstParam   int
	paramCount   int
	resultType   int
	receiverType int
	bodyStart    int
	bodyEnd      int
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
	p.decls = make([]rtgDecl, 0, 512)
	p.funcs = make([]rtgFuncDecl, 0, 1280)
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

	for i < len(p.toks) && p.toks[i].kind != rtgTokEOF {
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
			kind := p.toks[i].kind
			i++
			if rtgTokCharIs(&p, i, '(') {
				end := rtgSkipBalanced(&p, i, '(', ')')
				if end <= i {
					rtgProgramError(&p, rtgDiagParseGroupedDecl)
					return p
				}
				var decl rtgDecl
				decl.kind = kind
				decl.nameStart = p.toks[start].start
				decl.nameEnd = p.toks[start].end
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
			name := &p.toks[i]
			i++
			end := rtgSkipTopLevelLine(&p, i)
			var decl rtgDecl
			decl.kind = kind
			decl.nameStart = name.start
			decl.nameEnd = name.end
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
		for receiverEnd < len(p.toks) && !rtgTokCharIs(p, receiverEnd, ')') {
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
	fn.nameStart = p.toks[i].start
	fn.nameEnd = p.toks[i].end
	i++

	for i < len(p.toks) && !rtgTokCharIs(p, i, '{') && p.toks[i].kind != rtgTokEOF {
		i++
	}
	if !rtgTokCharIs(p, i, '{') {
		return
	}
	fn.bodyStart = i
	depth := 1
	i++
	for i < len(p.toks) && depth > 0 {
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
	for i < len(p.toks) && depth > 0 {
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
	if start >= len(p.toks) {
		return start
	}
	line := p.toks[start-1].line
	i := start
	depth := 0
	for i < len(p.toks) {
		if p.toks[i].kind == rtgTokEOF {
			return i
		}
		if p.toks[i].line != line && depth == 0 {
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

func rtgScan(src []byte) []rtgToken {
	toks := make([]rtgToken, 0, 122880)
	i := 0
	line := 1
	for i < len(src) {
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
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			i += 2
			for i < len(src) && src[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		if rtgIsIdentStart(c) {
			i++
			start := i - 1
			for i < len(src) && rtgIsIdentPart(src[i]) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgKeywordKind(src, start, i), start: start, end: i, line: line})
			continue
		}
		if rtgIsDigit(c) {
			start := i
			kind := rtgTokNumber
			if c == '0' && i+1 < len(src) && (src[i+1] == 'x' || src[i+1] == 'X' || src[i+1] == 'b' || src[i+1] == 'B') {
				i += 2
				for i < len(src) && rtgIsIdentPart(src[i]) {
					i++
				}
			} else {
				i++
				for i < len(src) && rtgIsDigit(src[i]) {
					i++
				}
				if i < len(src) && src[i] == '.' {
					kind = rtgTokFloat
					i++
					for i < len(src) && rtgIsDigit(src[i]) {
						i++
					}
				}
			}
			toks = append(toks, rtgToken{kind: kind, start: start, end: i, line: line})
			continue
		}
		if c == '"' {
			start := i
			i++
			for i < len(src) && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					if src[i] == '\n' {
						line++
					}
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokString, start: start, end: i, line: line})
			continue
		}
		if c == '\'' {
			start := i
			i++
			for i < len(src) && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i += 2
				} else {
					i++
				}
			}
			if i < len(src) {
				i++
			}
			toks = append(toks, rtgToken{kind: rtgTokChar, start: start, end: i, line: line})
			continue
		}
		start := i
		i = rtgScanOperator(src, i)
		toks = append(toks, rtgToken{kind: rtgTokOp, start: start, end: i, line: line})
	}
	toks = append(toks, rtgToken{kind: rtgTokEOF, start: len(src), end: len(src), line: line})
	return toks
}

func rtgScanOperator(src []byte, i int) int {
	if i+2 <= len(src) {
		c0 := src[i]
		c1 := src[i+1]
		if c1 == '=' {
			if c0 == ':' || c0 == '=' || c0 == '!' || c0 == '<' || c0 == '>' || c0 == '+' || c0 == '-' || c0 == '*' || c0 == '/' || c0 == '%' {
				return i + 2
			}
		}
		if c0 == '&' && (c1 == '&' || c1 == '^') {
			return i + 2
		}
		if c0 == '|' && c1 == '|' {
			return i + 2
		}
		if c0 == '<' && c1 == '<' {
			return i + 2
		}
		if c0 == '>' && c1 == '>' {
			return i + 2
		}
		if c0 == '+' && c1 == '+' {
			return i + 2
		}
		if c0 == '-' && c1 == '-' {
			return i + 2
		}
	}
	return i + 1
}

func rtgKeywordKind(src []byte, start int, end int) int {
	n := end - start
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
	if i >= len(p.toks) {
		return false
	}
	return p.toks[i].kind == kind
}

func rtgTokCharIs(p *rtgProgram, i int, c byte) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	start := p.toks[i].start
	end := p.toks[i].end
	return end-start == 1 && p.src[start] == c
}

func rtgTok2Is(p *rtgProgram, i int, a byte, b byte) bool {
	if i < 0 || i >= len(p.toks) {
		return false
	}
	start := p.toks[i].start
	end := p.toks[i].end
	return end-start == 2 && p.src[start] == a && p.src[start+1] == b
}

func rtgBoolTokenValue(p *rtgProgram, tok int) int {
	start := p.toks[tok].start
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
		if src[start] == 'a' && src[start+1] == 'p' && src[start+2] == 'p' && src[start+3] == 'e' && src[start+4] == 'n' && src[start+5] == 'd' {
			return rtgIdentAppend
		}
		if src[start] == '[' && src[start+1] == ']' && src[start+2] == 'b' && src[start+3] == 'y' && src[start+4] == 't' && src[start+5] == 'e' {
			return rtgIdentByteSlice
		}
	}
	return 0
}

func rtgIsIdentStart(c byte) bool {
	if c >= 'a' {
		if c <= 'z' {
			return true
		}
	}
	if c >= 'A' {
		if c <= 'Z' {
			return true
		}
	}
	if c == '_' {
		return true
	}
	return false
}

func rtgIsIdentPart(c byte) bool {
	if rtgIsIdentStart(c) {
		return true
	}
	if rtgIsDigit(c) {
		return true
	}
	return false
}

func rtgIsDigit(c byte) bool {
	if c >= '0' {
		if c <= '9' {
			return true
		}
	}
	return false
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

func rtgDecodeStringToken(p *rtgProgram, tokIndex int) []byte {
	tok := &p.toks[tokIndex]
	src := p.src
	var out []byte
	i := tok.start + 1
	end := tok.end - 1
	for i < end {
		if src[i] == '\\' && i+1 < end {
			i++
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
	tok := &p.toks[tokIndex]
	src := p.src
	start := tok.start
	base := 10
	if tok.end-start > 2 && src[start] == '0' {
		prefix := src[start+1]
		if prefix == 'x' || prefix == 'X' {
			base = 16
			start += 2
		} else if prefix == 'b' || prefix == 'B' {
			base = 2
			start += 2
		}
	}
	if base == 10 && tok.end-start > 1 && src[start] == '0' {
		base = 8
		start++
	}
	n := 0
	for i := start; i < tok.end; i++ {
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
	tok := &p.toks[tokIndex]
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

func rtgParseCharToken(p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
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
		ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
		if !ep.ok || len(ep.exprs) == 0 {
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

func rtgEvalConstExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) rtgConstResult {
	p := g.prog
	e := &ep.exprs[idx]
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
			result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
			if result.ok {
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
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				resolved := rtgResolveType(g.meta, namedType)
				if resolved.kind == rtgTypeInt || resolved.kind == rtgTypeInt16 || resolved.kind == rtgTypeInt32 || resolved.kind == rtgTypeInt64 || resolved.kind == rtgTypeBool {
					result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
					if result.ok {
						result.value = rtgConvertConstInt(result.value, resolved.kind)
					}
					return result
				}
				if resolved.kind == rtgTypeByte {
					result := rtgEvalConstExpr(g, ep, ep.args[e.firstArg])
					if result.ok {
						result.value = rtgConvertConstInt(result.value, rtgTypeByte)
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
	if tok < 0 || tok >= len(p.toks) {
		var r rtgConstResult
		return r
	}
	start := p.toks[tok].start
	end := p.toks[tok].end
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
	e := &ep.exprs[idx]
	if e.kind != rtgExprIdent {
		return false
	}
	return rtgBytesEqualText(p.src, e.nameStart, e.nameEnd, text)
}

func rtgParseExpression(p *rtgProgram, start int, end int) rtgExprParse {
	var ep rtgExprParse
	ep.prog = p
	ep.pos = start
	ep.end = end
	ep.exprs = make([]rtgExpr, 0, 512)
	ep.args = make([]int, 0, 512)
	ep.fields = make([]rtgCompositeField, 0, 512)
	ep.ok = true
	rtgParseBinaryExpr(&ep, 1)
	if ep.pos < ep.end {
		rtgExprError(&ep, rtgDiagParseExpression)
	}
	return ep
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
			var compositeFields []rtgCompositeField
			ep.pos++
			for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
				var field rtgCompositeField
				if rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) && rtgTokCharIs(ep.prog, ep.pos+1, ':') {
					nameTok := ep.prog.toks[ep.pos]
					ep.pos += 2
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					fieldRoot := rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					field.nameStart = nameTok.start
					field.nameEnd = nameTok.end
					field.expr = fieldRoot
					ep.pos = fieldEnd
				} else if rtgTokCharIs(ep.prog, ep.pos, '{') {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseImplicitCompositeExpr(ep)
					ep.end = oldEnd
					ep.pos = fieldEnd
				} else {
					fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
					oldEnd := ep.end
					ep.end = fieldEnd
					field.expr = rtgParseBinaryExpr(ep, 1)
					ep.end = oldEnd
					ep.pos = fieldEnd
				}
				compositeFields = append(compositeFields, field)
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
				oldEnd := ep.end
				if colon > indexStart {
					ep.pos = indexStart
					ep.end = colon
					low = rtgParseBinaryExpr(ep, 1)
				}
				if colon+1 < indexEnd {
					ep.pos = colon + 1
					ep.end = indexEnd
					high = rtgParseBinaryExpr(ep, 1)
				}
				ep.end = oldEnd
				ep.pos = indexEnd + 1
				left = rtgAddExpr(ep, rtgExprSlice, indexTok, left, high, low, 0, 0, 0)
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
			nameTok := ep.prog.toks[ep.pos+1]
			ep.pos += 2
			left = rtgAddExpr(ep, rtgExprSelector, dotTok, left, 0, 0, 0, nameTok.start, nameTok.end)
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
	var compositeFields []rtgCompositeField
	ep.pos++
	for ep.ok && ep.pos < ep.end && !rtgTokCharIs(ep.prog, ep.pos, '}') {
		if !rtgTokIsKind(ep.prog, ep.pos, rtgTokIdent) || !rtgTokCharIs(ep.prog, ep.pos+1, ':') {
			rtgExprError(ep, rtgDiagParseComposite)
			return 0
		}
		nameTok := ep.prog.toks[ep.pos]
		ep.pos += 2
		fieldEnd := rtgFindExprBoundary(ep.prog, ep.pos, ep.end)
		oldEnd := ep.end
		ep.end = fieldEnd
		fieldRoot := rtgParseBinaryExpr(ep, 1)
		ep.end = oldEnd
		compositeFields = append(compositeFields, rtgCompositeField{nameStart: nameTok.start, nameEnd: nameTok.end, expr: fieldRoot})
		ep.pos = fieldEnd
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

func rtgParsePrimaryExpr(ep *rtgExprParse) int {
	if ep.pos >= ep.end {
		rtgExprError(ep, rtgDiagParseExpression)
		return 0
	}
	tok := &ep.prog.toks[ep.pos]
	if rtgTokCharIs(ep.prog, ep.pos, '[') && rtgTokCharIs(ep.prog, ep.pos+1, ']') && rtgTokIsKind(ep.prog, ep.pos+2, rtgTokIdent) {
		startTok := ep.pos
		nameTok := ep.prog.toks[ep.pos+2]
		ep.pos += 3
		return rtgAddExpr(ep, rtgExprIdent, startTok, 0, 0, 0, 0, ep.prog.toks[startTok].start, nameTok.end)
	}
	if tok.kind == rtgTokIdent {
		ep.pos++
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "true") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		if rtgBytesEqualText(ep.prog.src, tok.start, tok.end, "false") {
			return rtgAddExpr(ep, rtgExprBool, ep.pos-1, 0, 0, 0, 0, 0, 0)
		}
		return rtgAddExpr(ep, rtgExprIdent, ep.pos-1, 0, 0, 0, 0, tok.start, tok.end)
	}
	if tok.kind == rtgTokNumber {
		ep.pos++
		return rtgAddExpr(ep, rtgExprInt, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokFloat {
		ep.pos++
		ep.hasFloat = true
		return rtgAddExpr(ep, rtgExprFloat, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokString {
		ep.pos++
		return rtgAddExpr(ep, rtgExprString, ep.pos-1, 0, 0, 0, 0, 0, 0)
	}
	if tok.kind == rtgTokChar {
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
	if pos < 0 || pos >= len(p.toks) {
		return 0
	}
	start := p.toks[pos].start
	end := p.toks[pos].end
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
		bp.stmts = append(bp.stmts, stmt)
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
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
		}
		rtgAddStmt(bp, rtgStmtGoto, start, endTok, 0, 0, 0, 0, 0, 0, nameStart, nameEnd)
		return endTok
	}
	if rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, ':') {
		name := &p.toks[start]
		rtgAddStmt(bp, rtgStmtLabel, start, start+2, 0, 0, 0, 0, 0, 0, name.start, name.end)
		return start + 2
	}
	if rtgTokIsKind(p, start, rtgTokVar) || rtgTokIsKind(p, start, rtgTokConst) {
		endTok := rtgStatementLineEnd(p, start+1, end)
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start+1, rtgTokIdent) {
			nameStart = p.toks[start+1].start
			nameEnd = p.toks[start+1].end
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
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
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
	bp.stmts = append(bp.stmts, stmt)
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
	line := p.toks[start].line
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
			if p.toks[i].line != line {
				if rtgTokCharIs(p, i, '{') {
					return i
				}
				if rtgTokIsKind(p, i, rtgTokReturn) || rtgTokIsKind(p, i, rtgTokIf) || rtgTokIsKind(p, i, rtgTokFor) || rtgTokIsKind(p, i, rtgTokSwitch) || rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault) || rtgTokIsKind(p, i, rtgTokVar) || rtgTokIsKind(p, i, rtgTokConst) || rtgTokIsKind(p, i, rtgTokBreak) || rtgTokIsKind(p, i, rtgTokContinue) || rtgTokIsKind(p, i, rtgTokGoto) {
					return i
				}
				if rtgLineContinuesAfterPrevToken(p, i) {
					line = p.toks[i].line
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
		if i > start && p.toks[i].line != line && paren == 0 && brack == 0 && brace == 0 {
			if rtgLineContinuesAfterPrevToken(p, i) {
				line = p.toks[i].line
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
	tokStart := p.toks[prev].start
	tokEnd := p.toks[prev].end
	if tokEnd <= tokStart {
		return false
	}
	c := p.src[tokStart]
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
		tok := &p.toks[i]
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
	p := pp
	var m rtgMeta
	m.prog = p
	m.types = make([]rtgTypeInfo, 0, 2048)
	m.fields = make([]rtgFieldInfo, 0, 512)
	m.globals = make([]rtgSymbolInfo, 0, 512)
	m.params = make([]rtgSymbolInfo, 0, 3072)
	m.funcs = make([]rtgFuncInfo, 0, 1280)
	m.globalBuckets = make([]int, 1024)
	for i := 0; i < len(m.globalBuckets); i++ {
		m.globalBuckets[i] = -1
	}
	m.funcBuckets = make([]int, 2048)
	for i := 0; i < len(m.funcBuckets); i++ {
		m.funcBuckets[i] = -1
	}
	m.ok = true
	rtgInitBuiltinTypes(&m)

	for i := 0; i < len(p.decls); i++ {
		decl := p.decls[i]
		if decl.kind != rtgTokType && decl.kind != rtgTokVar && decl.kind != rtgTokConst {
			continue
		}
		entryStart := decl.startTok + 1
		if rtgTokCharIs(p, entryStart, '(') {
			groupEnd := decl.endTok
			if decl.kind == rtgTokConst {
				rtgParseConstDecls(&m, p, entryStart+1, groupEnd-1)
				continue
			}
			j := entryStart + 1
			for j < groupEnd-1 {
				if rtgTokIsKind(p, j, rtgTokIdent) {
					entryEnd := rtgStatementLineEnd(p, j, groupEnd-1)
					rtgParseTopDeclEntry(&m, p, decl.kind, j, entryEnd)
					if entryEnd <= j {
						j++
					} else {
						j = entryEnd
					}
				} else {
					j++
				}
			}
			continue
		}
		if decl.kind == rtgTokConst {
			rtgParseConstDecls(&m, p, entryStart, decl.endTok)
		} else {
			rtgParseTopDeclEntry(&m, p, decl.kind, entryStart, decl.endTok)
		}
	}
	for i := 0; i < len(p.funcs); i++ {
		rtgParseFuncInfo(&m, i)
	}
	rtgBuildFuncLookup(&m)

	return m
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
	if len(m.globalBuckets) > 0 && len(m.globalNext) == len(m.globals) {
		hash := rtgHashRange(m.prog.src, nameStart, nameEnd)
		bucket := hash % len(m.globalBuckets)
		i := m.globalBuckets[bucket]
		for i >= 0 {
			s := &m.globals[i]
			if s.kind == kind && rtgBytesEqualRange(m.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
				return i
			}
			i = m.globalNext[i]
		}
		return -1
	}
	for i := 0; i < len(m.globals); i++ {
		s := &m.globals[i]
		if s.kind == kind && rtgBytesEqualRange(m.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgBuildFuncLookup(m *rtgMeta) {
	m.funcNext = make([]int, len(m.funcs), 1280)
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

func rtgInitBuiltinTypes(m *rtgMeta) {
	rtgAddBuiltinType(m, rtgTypeInvalid, 0)
	rtgAddBuiltinType(m, rtgTypeInt, rtgNativeIntSize)
	rtgAddBuiltinType(m, rtgTypeInt64, 8)
	rtgAddBuiltinType(m, rtgTypeByte, 1)
	rtgAddBuiltinType(m, rtgTypeBool, 1)
	rtgAddBuiltinType(m, rtgTypeString, 16)
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
	var prevValues []int
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
		var names []int
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
			var newValues []int
			newValues = rtgSplitTopLevelComma(p, eq+1, specEnd, newValues)
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
			name := &p.toks[nameTok]
			if rtgBytesEqualText(p.src, name.start, name.end, "_") {
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
			sym.nameStart = name.start
			sym.nameEnd = name.end
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
	ep := rtgParseExpression(p, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		var r rtgConstResult
		return r
	}
	rootIndex := len(ep.exprs) - 1
	return rtgEvalMetaParsedConstExpr(m, p, &ep, rootIndex, iotaValue)
}

func rtgEvalMetaParsedConstExpr(m *rtgMeta, p *rtgProgram, ep *rtgExprParse, idx int, iotaValue int) rtgConstResult {
	e := &ep.exprs[idx]
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
	name := &p.toks[start]
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
		if m.types[typeResult.typ].kind == rtgTypeStruct || m.types[typeResult.typ].kind == rtgTypePointer || m.types[typeResult.typ].kind == rtgTypeSlice {
			m.types[typeResult.typ].nameStart = name.start
			m.types[typeResult.typ].nameEnd = name.end
		} else {
			size := rtgTypeSize(m, typeResult.typ)
			rtgAddType(m, rtgTypeNamed, typeResult.typ, 0, 0, size, name.start, name.end)
		}
		return
	}
	eq := start
	j := start + 1
	for j < end {
		if j >= 0 && j < len(p.toks) {
			tok := &p.toks[j]
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
	rtgMetaAppendGlobal(m, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, kind: kind, typ: typ, initStart: initStart, initEnd: initEnd})
}

func rtgParseVarDeclEntry(m *rtgMeta, p *rtgProgram, start int, end int) {
	eq := rtgFindConstSpecEqual(p, start, end)
	headEnd := end
	if eq > start {
		headEnd = eq
	}
	var names []int
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
		values = rtgSplitTopLevelComma(p, eq+1, end, values)
	}
	valueCount := len(values) / 2
	if valueCount != 0 && valueCount != len(names) {
		rtgMetaError(m, rtgDiagMetaTopDecl)
		return
	}
	for i := 0; i < len(names); i++ {
		nameTok := names[i]
		name := &p.toks[nameTok]
		if rtgBytesEqualText(p.src, name.start, name.end, "_") {
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
		rtgMetaAppendGlobal(m, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, kind: rtgTokVar, typ: symType, initStart: initStart, initEnd: initEnd})
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
	m.funcs = append(m.funcs, rtgFuncInfo{declIndex: fnIndex, nameStart: nameStart, nameEnd: nameEnd, firstParam: firstParam, paramCount: paramCount, resultType: resultType, receiverType: receiverType, bodyStart: fn.bodyStart + 1, bodyEnd: fn.bodyEnd})
}

func rtgParseFuncResultType(m *rtgMeta, p *rtgProgram, start int, end int) int {
	if rtgTokCharIs(p, start, '(') {
		closeTok := rtgFindMatchingExprClose(p, start+1, end, '(', ')')
		if closeTok > start && closeTok <= end {
			var parts []int
			parts = rtgSplitTopLevelComma(p, start+1, closeTok, parts)
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
		if fieldSize < 8 {
			fieldSize = 8
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
		name := &p.toks[i]
		typeStart := i + 1
		entryEnd := typeStart
		depth := 0
		for entryEnd < end {
			if depth == 0 && rtgTokCharIs(p, entryEnd, ',') {
				break
			}
			if rtgTokCharIs(p, entryEnd, '[') || rtgTokCharIs(p, entryEnd, '{') || rtgTokCharIs(p, entryEnd, '(') {
				depth++
			} else if rtgTokCharIs(p, entryEnd, ']') || rtgTokCharIs(p, entryEnd, '}') || rtgTokCharIs(p, entryEnd, ')') {
				depth--
			}
			entryEnd++
		}
		variadic := 0
		if rtgTokCharIs(p, typeStart, '.') && rtgTokCharIs(p, typeStart+1, '.') && rtgTokCharIs(p, typeStart+2, '.') {
			variadic = 1
		}
		typeResult := rtgParseType(m, p, typeStart, entryEnd)
		if typeResult.typ == 0 {
			rtgMetaError(m, rtgDiagMetaParamList)
			return
		}
		m.params = append(m.params, rtgSymbolInfo{nameStart: name.start, nameEnd: name.end, typ: typeResult.typ, initStart: variadic})
		*count = *count + 1
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
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, 24, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokCharIs(p, start, '*') {
		elem := rtgParseType(m, p, start+1, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypePointer, elem.typ, 0, 0, 8, 0, 0)
		return rtgTypeResult{typ: typ, next: elem.next}
	}
	if rtgTokCharIs(p, start, '[') && rtgTokCharIs(p, start+1, ']') {
		elem := rtgParseType(m, p, start+2, end)
		if elem.typ == 0 {
			return rtgTypeResult{next: start}
		}
		typ := rtgAddType(m, rtgTypeSlice, elem.typ, 0, 0, 24, 0, 0)
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
				name := &p.toks[i]
				lineEnd := rtgStatementLineEnd(p, i, closeTok)
				fieldType := rtgParseType(m, p, i+1, lineEnd)
				if fieldType.typ == 0 {
					return rtgTypeResult{next: start}
				}
				offset = rtgAlignTo8(offset)
				m.fields = append(m.fields, rtgFieldInfo{nameStart: name.start, nameEnd: name.end, typ: fieldType.typ, offset: offset})
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
	tok := &p.toks[tokIndex]
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int") {
		return rtgTypeInt
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int16") {
		return rtgTypeInt16
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int32") {
		return rtgTypeInt32
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "int64") {
		return rtgTypeInt64
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "byte") {
		return rtgTypeByte
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "bool") {
		return rtgTypeBool
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "string") {
		return rtgTypeString
	}
	if rtgBytesEqualText(p.src, tok.start, tok.end, "float64") {
		return rtgTypeFloat64
	}
	return 0
}

func rtgNamedTypeFromToken(m *rtgMeta, p *rtgProgram, tokIndex int) int {
	tok := &p.toks[tokIndex]
	for i := 0; i < len(m.types); i++ {
		if m.types[i].nameEnd > m.types[i].nameStart && rtgBytesEqualRange(p.src, m.types[i].nameStart, m.types[i].nameEnd, tok.start, tok.end) {
			return i
		}
	}
	return rtgAddType(m, rtgTypeNamed, 0, 0, 0, 8, tok.start, tok.end)
}

func rtgAddType(m *rtgMeta, kind int, elem int, first int, count int, size int, nameStart int, nameEnd int) int {
	m.types = append(m.types, rtgTypeInfo{kind: kind, elem: elem, first: first, count: count, size: size, nameStart: nameStart, nameEnd: nameEnd})
	index := len(m.types) - 1
	return index
}

func rtgTypeSize(m *rtgMeta, typ int) int {
	t := rtgResolveType(m, typ)
	if t.size > 0 {
		return t.size
	}
	return 8
}

func rtgResolveType(m *rtgMeta, typ int) rtgTypeInfo {
	if typ >= 0 && typ < len(m.types) {
		t := m.types[typ]
		if t.kind == rtgTypeNamed && t.elem > 0 && t.elem < len(m.types) {
			return m.types[t.elem]
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
	return 8
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
	rem := v % 8
	if rem == 0 {
		return v
	}
	return v + 8 - rem
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
	reg := 0
	if rtgTypeIsStruct(meta, fn.resultType) {
		reg = 1
	}
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := rtgAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if rtgTypeIsSlice(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) || !rtgStoreParamWord(g, reg+2, offset-16) {
				return false
			}
			reg += 3
			continue
		}
		if rtgTypeIsString(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) {
				return false
			}
			reg += 2
			continue
		}
		if rtgTypeIsStruct(meta, param.typ) {
			size := rtgTypeSize(meta, param.typ)
			for at := 0; at < size; at += 8 {
				if !rtgStoreParamWord(g, reg, offset-at) {
					return false
				}
				reg++
			}
			continue
		}
		if !rtgStoreParamWord(g, reg, offset) {
			return false
		}
		reg++
	}
	return true
}
func rtgAsmImmFits8Signed(imm int) bool {
	return imm >= -128 && imm <= 127
}
func rtgAsmMovRaxIntToken(a *rtgAsm, p *rtgProgram, tokIndex int) {
	if rtgTargetArch == rtgArchWasm32 {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, rtgParseIntToken(p, tokIndex))
		return
	}
	needsMovabs := rtgIntTokenNeedsMovabs(p, tokIndex)
	value := rtgParseIntToken(p, tokIndex)
	if needsMovabs {
		rtgAsmMovRaxImm64(a, value)
		return
	}
	rtgAsmMovRaxImm(a, value)
}
func rtgIntTokenNeedsMovabs(p *rtgProgram, tokIndex int) bool {
	tok := &p.toks[tokIndex]
	start := tok.start
	if tok.end-start > 2 && p.src[start] == '0' {
		return false
	}
	digits := tok.end - start
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
func rtgAsmMovRdxRax(a *rtgAsm) {
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
func rtgAsmMovRcxRax(a *rtgAsm) {
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
func rtgAsmMovRcxRdx(a *rtgAsm) {
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
func rtgAsmPushRax(a *rtgAsm) {
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
func rtgAsmPushRcx(a *rtgAsm) {
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
func rtgAsmPushRdx(a *rtgAsm) {
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
	rtgAsmMovRaxImm(a, imm)
	rtgAsmPushRax(a)
}
func rtgAsmPushSliceRegs(a *rtgAsm) {
	rtgAsmPushRcx(a)
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}
func rtgAsmPushStringRegs(a *rtgAsm) {
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}
func rtgAsmPopRax(a *rtgAsm) {
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
func rtgAsmPopRcx(a *rtgAsm) {
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
func rtgAsmPopRdx(a *rtgAsm) {
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
func rtgAsmPopRsi(a *rtgAsm) {
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
func rtgAsmStoreRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x45, 0x85)
}
func rtgAsmStoreRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x55, 0x95)
}
func rtgAsmLoadRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x45, 0x85)
}
func rtgAsmLeaRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x45, 0x85)
}
func rtgAsmLeaRdiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x7d, 0xbd)
}
func rtgAsmLeaRsiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x75, 0xb5)
}
func rtgAsmLoadRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x55, 0x95)
}
func rtgAsmRdxDisp(a *rtgAsm, disp int) {
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
func rtgAsmLoadRcxStack(a *rtgAsm, offset int) {
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
	rtgAsmStoreRaxStack(a, offset)
	rtgAsmStoreRdxStack(a, offset-8)
	rtgAsmStackMem(a, offset-16, 0x8948, 0x4d, 0x8d)
}
func rtgAsmPopStoreStringMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
}
func rtgAsmPopStoreSliceMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+16)
}
func rtgAsmStoreAlMemRdxRcx1(a *rtgAsm) {
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
func rtgAsmStoreRaxMemRdxRcxSize(a *rtgAsm, size int) {
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
		rtgAsmStoreAlMemRdxRcx1(a)
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
	rtgAsmStoreRaxMemRdxRcx8(a)
}
func rtgAsmIncRcx(a *rtgAsm) {
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
func rtgAsmIncRax(a *rtgAsm) {
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
func rtgAsmImulRcxImm(a *rtgAsm, imm int) {
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
	winReadLabel       int
	winReadEmitted     bool
	winWriteLabel      int
	winWriteEmitted    bool
	lastRangeReturns   bool
	scopeBase          int
	constEvalIota      int
	constEvalIotaValid int
}

func rtgAddStringData(g *rtgLinearGen, msg []byte) int {
	msgOff := len(g.asm.data)
	for i := 0; i < len(msg); i++ {
		g.asm.data = append(g.asm.data, msg[i])
	}
	g.asm.data = append(g.asm.data, 0)
	return msgOff
}
func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	stmts := make([]rtgStmt, 0, 1024)
	bp.prog = g.prog
	bp.stmts = stmts
	bp.ok = true
	i := start
	lastKind := 0
	for bp.ok && i < end {
		if i < 0 || i >= len(bp.prog.toks) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, ';') {
			i++
			continue
		}
		if bp.prog.toks[i].start < 0 || bp.prog.toks[i].start >= len(bp.prog.src) {
			return true
		}
		if rtgTokIsKind(bp.prog, i, rtgTokEOF) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, '}') {
			return true
		}
		before := len(bp.stmts)
		next := rtgParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || len(bp.stmts) <= before {
			return false
		}
		stmt := bp.stmts[len(bp.stmts)-1]
		lastKind = stmt.kind
		i = next
		if !rtgEmitLinearStmt(g, &stmt) {
			rtgPrintErr("rtg: statement failed near line ")
			rtgPrintIntErr(bp.prog.toks[stmt.startTok].line)
			rtgPrintErr("\n")
			write(2, bp.prog.src[bp.prog.toks[stmt.startTok].start:bp.prog.toks[stmt.endTok-1].end], -1)
			rtgPrintErr("\n")
			return false
		}
	}
	g.lastRangeReturns = lastKind == rtgStmtReturn
	if !bp.ok {
		return false
	}
	return true
}
func rtgEmitScopedRange(g *rtgLinearGen, start int, end int) bool {
	oldLocals := g.locals
	oldScopeBase := g.scopeBase
	g.scopeBase = len(oldLocals)
	if !rtgEmitLinearRange(g, start, end) {
		return false
	}
	g.locals = oldLocals
	g.scopeBase = oldScopeBase
	return true
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
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		root := &ep.exprs[rootIndex]
		if root.kind != rtgExprCall {
			if rtgTargetArch == rtgArchAarch64 {
				rtgPrintErr("rtg: aarch64 expr stmt root not call\n")
			}
			return false
		}
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			if rtgTargetArch == rtgArchAarch64 {
				rtgPrintErr("rtg: aarch64 expr stmt call emission failed\n")
			}
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
			rtgAsmMovRaxImm(a, 0)
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
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if rtgTypeIsStruct(g.meta, resultType) {
			if !rtgEmitStructReturnExpr(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsSlice(g.meta, resultType) {
			if !rtgEmitSliceValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsString(g.meta, resultType) {
			if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else {
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			resultResolved := rtgResolveType(g.meta, resultType)
			rtgAsmNormalizeRaxForKind(a, resultResolved.kind)
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
func rtgEmitLinearIf(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
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
	if rtgTokIsKind(p, stmt.elseStart, rtgTokIf) && rtgTokIsKind(p, stmt.elseStart-1, rtgTokElse) {
		var nested rtgBodyParse
		stmts := make([]rtgStmt, 0, 16)
		nested.prog = p
		nested.stmts = stmts
		nested.ok = true
		next := rtgParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || len(nested.stmts) != 1 {
			return false
		}
		nestedStmt := nested.stmts[0]
		if !rtgEmitLinearStmt(g, &nestedStmt) {
			return false
		}
	} else if !rtgEmitScopedRange(g, stmt.elseStart, stmt.elseEnd) {
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
		return rtgEmitLinearClassicFor(g, stmt, semi1)
	}
	startLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, startLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}
func rtgEmitLinearSwitch(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.exprStart >= stmt.exprEnd {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	switchType := rtgInferParsedExprType(g, &ep, rootIndex)
	stringSwitch := rtgTypeIsString(g.meta, switchType)
	valueOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOffset := 0
	if stringSwitch {
		lenOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
		rtgAsmStoreRdxStack(a, lenOffset)
	} else {
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
	}

	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	var clauseStarts []int
	var clauseLabels []int
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
		ep := rtgParseExpression(p, i, valueEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if stringSwitch {
			if !rtgEmitSwitchStringCaseTest(g, valueOffset, lenOffset, &ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			rtgAsmLoadRaxStack(a, valueOffset)
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			rtgAsmCmpRcxRaxSet(a, 0x94)
			rtgAsmCmpRaxImm8(a, 0)
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
func rtgEmitLinearClassicFor(g *rtgLinearGen, stmt *rtgStmt, semi1 int) bool {
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
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, postLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if semi1+1 < semi2 {
		ep := rtgParseExpression(p, semi1+1, semi2)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
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
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
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
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
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
	ep := rtgParseExpression(p, start, opTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	inc := rtgTok2Is(p, opTok, '+', '+')
	if root.kind == rtgExprIdent {
		localOffset := rtgFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			rtgClearLocalConstAtOffset(g, localOffset)
			if rtgTargetArch == rtgArchAarch64 || rtgTargetArch == rtgArchArm || rtgTargetArch == rtgArchWasm32 {
				rtgAsmLoadRaxStack(a, localOffset)
				rtgAsmPushImm(a, 1)
				rtgAsmPopRcx(a)
				if inc {
					rtgAsmAddRaxRcx(a)
				} else {
					rtgAsmSubRaxRcx(a)
				}
				rtgAsmStoreRaxStack(a, localOffset)
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
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPushImm(a, 1)
			rtgAsmPopRcx(a)
			if inc {
				rtgAsmAddRaxRcx(a)
			} else {
				rtgAsmSubRaxRcx(a)
			}
			rtgAsmStoreRaxBss(a, globalOffset)
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
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
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
	rtgAsmCmpRaxImm8(a, 0)
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
	rtgAsmCmpRaxImm8(a, 0)
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
		if size < 8 {
			size = 8
		}
		g.asm.bssSize += rtgAlignTo8(size)
		if rtgTypeIsInt(meta, s.typ) && rtgBytesEqualText(g.prog.src, s.nameStart, s.nameEnd, "rtgCompilerDefaultTarget") {
			rtgAsmMovRaxImm(a, rtgCurrentTarget)
			rtgAsmStoreRaxBss(a, off)
			continue
		}
		if s.initStart < s.initEnd {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return false
			}
			rootIndex := len(ep.exprs) - 1
			if rtgTypeIsString(meta, s.typ) {
				if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				continue
			}
			if rtgTypeIsSlice(meta, s.typ) {
				root := &ep.exprs[rootIndex]
				if root.kind != rtgExprComposite {
					return false
				}
				if !rtgEmitSliceLiteralRegs(g, &ep, rootIndex, s.typ) {
					return false
				}
				rtgAsmPushRcx(a)
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+16)
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
				rtgAsmMovRaxBssAddr(a, targetOff)
				rtgAsmStoreRaxBss(a, off)
				continue
			}
			constResult := rtgEvalConstExpr(g, &ep, rootIndex)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(a, constResult.value)
			rtgAsmNormalizeRaxForKind(a, resolved.kind)
			rtgAsmStoreRaxBss(a, off)
		} else if rtgTypeIsSlice(meta, s.typ) {
			rtgEmitInitEmptySliceBss(g, s.typ, off)
		}
	}
	return true
}
func rtgEmitGlobalStructInit(g *rtgLinearGen, ep *rtgExprParse, rootIndex int, typ int, off int) bool {
	root := &ep.exprs[rootIndex]
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
			rtgAsmPushRdx(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
			rtgAsmPopRax(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset+8)
		} else if fieldResolved.kind == rtgTypeStruct || fieldResolved.kind == rtgTypeSlice {
			return false
		} else {
			constResult := rtgEvalConstExpr(g, ep, field.expr)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(&g.asm, constResult.value)
			rtgAsmNormalizeRaxForKind(&g.asm, fieldResolved.kind)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
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
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxBss(a, off)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxBss(a, off+8)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxBss(a, off+16)
}
func rtgEmitLinearAssign(g *rtgLinearGen, stmt *rtgStmt) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if (stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar)) && rtgTokIsKind(p, stmt.startTok+1, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok+1].start
		nameEnd = p.toks[stmt.startTok+1].end
	} else if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok].start
		nameEnd = p.toks[stmt.startTok].end
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	compoundAssign := false
	if assignTok >= 0 && assignTok < len(p.toks) {
		tok := &p.toks[assignTok]
		if tok.end-tok.start == 2 && p.src[tok.start+1] == '=' {
			c := p.src[tok.start]
			compoundAssign = c == '+' || c == '-' || c == '*' || c == '/' || c == '%'
		}
	}
	if assignTok > stmt.startTok && rtgEmitMultiAssign(g, stmt, assignTok) {
		return true
	}
	if assignTok > stmt.startTok && compoundAssign {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice || !rtgTypeKindIsScalarInt(elemType.kind) {
					return false
				}
				elemSize := rtgScalarKindSize(elemType.kind)
				indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
					return false
				}
				rtgAsmStoreRaxStack(a, indexOffset)
				if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
					return false
				}
				rtgAsmStoreRaxStack(a, ptrOffset)
				rtgAsmLoadRaxStack(a, ptrOffset)
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRcxStack(a, indexOffset)
				rtgAsmLoadRaxIndexRcxSize(a, elemSize)
				rtgAsmPushRax(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizeRaxForKind(a, elemType.kind)
				rtgAsmLoadRdxStack(a, ptrOffset)
				rtgAsmLoadRcxStack(a, indexOffset)
				rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
				lhsResolved := rtgResolveType(meta, lhsType)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmLoadRaxMemRdxDispSize(a, 0, lhsSize)
				rtgAsmPushRax(a)
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizeRaxForKind(a, lhsResolved.kind)
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmStoreRaxMemRdxDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice {
					return false
				}
				scalarElem := rtgTypeKindIsScalarInt(elemType.kind)
				indexOffset := 0
				if scalarElem {
					indexOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
				}
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if scalarElem {
					if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmNormalizeRaxForKind(a, elemType.kind)
					rtgAsmPushRax(a)
					rtgAsmLoadRaxStack(a, indexOffset)
					rtgAsmPushRax(a)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmMovRdxRax(a)
					rtgAsmPopRcx(a)
					rtgAsmPopRax(a)
					elemSize := rtgScalarKindSize(elemType.kind)
					rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
					return true
				}
				if elemType.kind == rtgTypeString {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					if !rtgEmitStringValueRegs(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmPushStringRegs(a)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmLoadRcxStack(a, indexOffset)
					rtgAsmShlRcxImm(a, 4)
					rtgAsmMovRdxRax(a)
					rtgAsmAddRdxRcx(a)
					rtgAsmPopStoreStringMemRdx(a, 0)
					return true
				}
				if elemType.kind == rtgTypeStruct {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					indexEnd := rtgFindMatchingExprClose(p, baseEnd+1, assignTok, '[', ']')
					if indexEnd <= baseEnd+1 {
						return false
					}
					indexEp := rtgParseExpression(p, baseEnd+1, indexEnd)
					if !indexEp.ok || len(indexEp.exprs) == 0 {
						return false
					}
					indexRoot := len(indexEp.exprs) - 1
					if !rtgEmitIntExpr(g, &indexEp, indexRoot) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmMovRdxRax(a)
					rtgAsmLoadRcxStack(a, indexOffset)
					elemSize := rtgTypeSize(meta, sliceType.elem)
					rtgAsmImulRcxImm(a, elemSize)
					rtgAsmAddRdxRcx(a)
					rtgAsmStoreRdxStack(a, destOffset)
					if !rtgEmitCompositeFieldToMem(g, &rhs, rhsIndex, sliceType.elem, destOffset, 0) {
						return false
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsSlice(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				if rtgEmitAppendAssignGeneral(g, stmt, &rhs) {
					return true
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitSliceValueRegs(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPushSliceRegs(a)
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmPopStoreSliceMemRdx(a, 0)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsStruct(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				size := rtgTypeSize(meta, lhsType)
				tempOffset := rtgAddTypedLocal(g, 0, 0, lhsType)
				if !rtgEmitTypedAssign(g, &rhs, rhsIndex, tempOffset) {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgEmitCopyStackToMemRdx(g, tempOffset, 0, size)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				lhsResolved := rtgResolveType(meta, lhsType)
				rtgAsmNormalizeRaxForKind(a, lhsResolved.kind)
				rtgAsmPopRdx(a)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				rtgAsmStoreRaxMemRdxDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && compoundAssign {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmMovRdxRax(a)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, targetSize)
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			if !rtgEmitRaxRcxOp(g, assignTok) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, targetKind)
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDispSize(a, 0, targetSize)
			return true
		}
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, targetKind)
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDispSize(a, 0, targetSize)
			return true
		}
		return false
	}
	if nameEnd == nameStart+1 && p.src[nameStart] == '_' {
		if assignTok <= stmt.startTok || !rtgTokCharIs(p, assignTok, '=') {
			return true
		}
		ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
		return ep.ok
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
		localIndex := rtgFindLocalIndex(g, p.toks[stmt.startTok].start, p.toks[stmt.startTok].end)
		if localIndex < 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[stmt.startTok+2].start, p.toks[stmt.startTok+2].end)
		if fieldOffset < 0 {
			return false
		}
		fieldType = rtgStructFieldType(g, g.locals[localIndex].typ, p.toks[stmt.startTok+2].start, p.toks[stmt.startTok+2].end)
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
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
				if assignTok+2 < stmt.endTok && rtgTokIsKind(p, assignTok+1, rtgTokIdent) && rtgTokCharIs(p, assignTok+2, '(') {
					fnIndex := -1
					for i := 0; i < len(g.meta.funcs); i++ {
						f := &g.meta.funcs[i]
						if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, p.toks[assignTok+1].start, p.toks[assignTok+1].end) {
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
			rtgAsmMovRaxImm(a, 0)
			rtgAsmStoreRaxBss(a, globalOffset)
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
	ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
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
	if rtgEmitAppendAssignGeneral(g, stmt, &ep) {
		if globalOffset < 0 && fieldStackOffset < 0 {
			rtgClearLocalConstAtOffset(g, offset)
		}
		return true
	}
	if compoundAssign {
		if globalOffset >= 0 {
			rtgAsmLoadRaxBss(a, globalOffset)
		} else {
			rtgAsmLoadRaxStack(a, offset)
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, assignTok) {
			return false
		}
		rtgAsmNormalizeRaxForKind(a, targetResolved.kind)
		if globalOffset >= 0 {
			rtgAsmStoreRaxBss(a, globalOffset)
		} else {
			rtgAsmStoreRaxStack(a, offset)
			if fieldStackOffset < 0 {
				rtgClearLocalConstAtOffset(g, offset)
			}
		}
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
	if !rtgEmitIntExpr(g, &ep, rootIndex) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, targetResolved.kind)
	if globalOffset >= 0 {
		rtgAsmStoreRaxBss(a, globalOffset)
	} else {
		rtgAsmStoreRaxStack(a, offset)
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
	var lhs []int
	var rhs []int
	lhs = rtgSplitTopLevelComma(p, stmt.startTok, assignTok, lhs)
	rhs = rtgSplitTopLevelComma(p, assignTok+1, stmt.endTok, rhs)
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
	var tempOffsets []int
	var tempTypes []int
	for i := 0; i < rhsCount; i++ {
		rhsStart := rhs[i*2]
		rhsEnd := rhs[i*2+1]
		ep := rtgParseExpression(p, rhsStart, rhsEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		typ := rtgInferParsedExprType(g, &ep, rootIndex)
		if typ == 0 {
			typ = rtgTypeInt
		}
		offset := rtgAddTypedLocal(g, 0, 0, typ)
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
	ep := rtgParseExpression(p, targetStart, targetEnd)
	if !ep.ok || len(ep.exprs) == 0 {
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
	ep := rtgParseExpression(p, rhsStart, rhsEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
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
	offset := rtgAddTypedLocal(g, 0, 0, resultType)
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
	rtgAsmStoreRaxStack(&g.asm, offset)
	return true
}
func rtgEmitTempToTarget(g *rtgLinearGen, kind int, targetStart int, targetEnd int, tempOffset int, tempType int) bool {
	p := g.prog
	ep := rtgParseExpression(p, targetStart, targetEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	size := rtgTypeSize(g.meta, tempType)
	if size < 8 {
		size = 8
	}
	if root.kind == rtgExprIdent {
		if root.nameEnd == root.nameStart+1 && p.src[root.nameStart] == '_' {
			return true
		}
		localIndex := rtgFindLocalIndex(g, root.nameStart, root.nameEnd)
		if kind == rtgStmtShort {
			localIndex = rtgFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
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
	if kind == rtgStmtShort {
		return false
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	return false
}
func rtgEmitCopyStackToBss(g *rtgLinearGen, srcOffset int, bssOffset int, size int) {
	if size < 8 {
		size = 8
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(&g.asm, srcOffset-at)
		rtgAsmStoreRaxBss(&g.asm, bssOffset+at)
	}
}
func rtgFindLocalIndexInCurrentScope(g *rtgLinearGen, nameStart int, nameEnd int) int {
	start := g.scopeBase
	if start < 0 {
		start = 0
	}
	for i := len(g.locals) - 1; i >= start; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgSetLocalConstAtOffset(g *rtgLinearGen, offset int, value int) {
	for i := len(g.locals) - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValue = value
			g.locals[i].constValid = 1
			return
		}
	}
}

func rtgClearLocalConstAtOffset(g *rtgLinearGen, offset int) {
	for i := len(g.locals) - 1; i >= 0; i-- {
		if g.locals[i].offset == offset {
			g.locals[i].constValid = 0
			return
		}
	}
}

func rtgLocalConstTrackable(g *rtgLinearGen, typ int, nameStart int, nameEnd int, afterTok int) bool {
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
	end := len(p.toks)
	if g.currentFunc >= 0 && g.currentFunc < len(g.meta.funcs) {
		end = g.meta.funcs[g.currentFunc].bodyEnd
	}
	i := afterTok
	if i < 0 {
		i = 0
	}
	for i < end {
		if rtgTokIsKind(p, i, rtgTokIdent) && rtgBytesEqualRange(p.src, p.toks[i].start, p.toks[i].end, nameStart, nameEnd) {
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

func rtgSplitTopLevelComma(p *rtgProgram, start int, end int, ranges []int) []int {
	partStart := start
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
		} else if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			ranges = append(ranges, partStart)
			ranges = append(ranges, i)
			partStart = i + 1
		}
		i++
	}
	if partStart < end {
		ranges = append(ranges, partStart)
		ranges = append(ranges, end)
	}
	return ranges
}
func rtgEmitTupleReturn(g *rtgLinearGen, start int, end int) bool {
	resultType := g.meta.funcs[g.currentFunc].resultType
	tuple := rtgResolveType(g.meta, resultType)
	var parts []int
	parts = rtgSplitTopLevelComma(g.prog, start, end, parts)
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
		ep := rtgParseExpression(g.prog, start, end)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		return rtgEmitStructReturnExpr(g, &ep, rootIndex)
	}
	return false
}
func rtgEmitTupleReturnField(g *rtgLinearGen, start int, end int, typ int, fieldOffset int) bool {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	tempOffset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitExprToLocal(g, &ep, rootIndex, tempOffset) {
		return false
	}
	size := rtgTypeSize(g.meta, typ)
	if size < 8 {
		size = 8
	}
	rtgAsmLoadRdxStack(&g.asm, g.returnStruct)
	rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
	return true
}
func rtgInferExprType(g *rtgLinearGen, start int, end int) int {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return 0
	}
	rootIndex := len(ep.exprs) - 1
	return rtgInferParsedExprType(g, &ep, rootIndex)
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
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee == rtgIdentAppend && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if callee == rtgIdentByteSlice && e.argCount == 1 {
			return rtgAddType(meta, rtgTypeSlice, rtgTypeByte, 0, 0, 24, 0, 0)
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
			}
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
		if callee == rtgIdentLen || callee == rtgIdentOpen || callee == rtgIdentClose || callee == rtgIdentRead || callee == rtgIdentWrite || callee == rtgIdentChmod || callee == rtgIdentCopy {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			return meta.funcs[fnIndex].resultType
		}
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				if namedType > 0 {
					return namedType
				}
			}
		}
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind == rtgTypeSlice {
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
			return rtgAddType(meta, rtgTypePointer, elemType, 0, 0, 8, 0, 0)
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
			pointerType = g.locals[localIndex].typ
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
	if e.tok < 0 || e.tok >= len(p.toks) {
		return 0
	}
	endTok := e.tok
	for endTok < len(p.toks) && p.toks[endTok].end <= e.nameEnd {
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
func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < len(g.locals); i++ {
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
			if !rtgEmitSelectorAddressRdx(g, ep, idx) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyMemRdxToStack(g, offset, size)
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
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		return true
	}
	if rtgTypeKindIsScalarInt(destResolved.kind) {
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		rtgAsmNormalizeRaxForKind(&g.asm, destResolved.kind)
		rtgAsmStoreRaxStack(&g.asm, offset)
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
func rtgEmitSliceValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, e.left) {
			return false
		}
		if e.firstArg >= 0 {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			baseResolved := rtgResolveType(meta, baseType)
			if baseResolved.kind != rtgTypeSlice {
				return false
			}
			elemSize := rtgTypeSize(meta, baseResolved.elem)
			if elemSize < 1 {
				elemSize = 8
			}
			baseOff := rtgAddTypedLocal(g, 0, 0, baseType)
			lowOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			highOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmStoreSliceStack(a, baseOff)
			if !rtgEmitIntExpr(g, ep, e.firstArg) {
				return false
			}
			rtgAsmStoreRaxStack(a, lowOff)
			if e.right >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmStoreRaxStack(a, highOff)
			} else {
				rtgAsmLoadRaxStack(a, baseOff-8)
				rtgAsmStoreRaxStack(a, highOff)
			}
			rtgAsmLoadRaxStack(a, baseOff-16)
			rtgAsmLoadRcxStack(a, lowOff)
			rtgAsmSubRaxRcx(a)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, highOff)
			rtgAsmLoadRcxStack(a, lowOff)
			rtgAsmSubRaxRcx(a)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, baseOff)
			rtgAsmLoadRcxStack(a, lowOff)
			if elemSize != 1 {
				rtgAsmImulRcxImm(a, elemSize)
			}
			rtgAsmAddRaxRcx(a)
			rtgAsmPopRdx(a)
			rtgAsmPopRcx(a)
			return true
		}
		if e.right >= 0 {
			rtgAsmPushRax(a)
			rtgAsmPushRcx(a)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmMovRdxRax(a)
			rtgAsmPopRcx(a)
			rtgAsmPopRax(a)
		}
		return true
	}
	if e.kind == rtgExprIdent {
		if rtgBytesEqualText(g.prog.src, e.nameStart, e.nameEnd, "nil") {
			rtgAsmMovRaxImm(a, 0)
			rtgAsmMovRdxImm(a, 0)
			rtgAsmMovRcxRdx(a)
			return true
		}
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(meta, globalType) {
				return false
			}
			rtgAsmLoadRaxBss(a, globalOffset+16)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPopRdx(a)
			rtgAsmPopRcx(a)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 16)
		rtgAsmMovRcxRax(a)
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
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
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if e.argCount == 2 && callee == rtgIdentAppend {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, &stmt, ep) {
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
		baseOff := rtgAddTypedLocal(g, 0, 0, rtgTypeString)
		lowOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		highOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRaxStack(a, baseOff)
		rtgAsmMovRaxRdx(a)
		rtgAsmStoreRaxStack(a, baseOff-8)
		if !rtgEmitIntExpr(g, ep, e.firstArg) {
			return false
		}
		rtgAsmStoreRaxStack(a, lowOff)
		if e.right >= 0 {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmStoreRaxStack(a, highOff)
		} else {
			rtgAsmLoadRaxStack(a, baseOff-8)
			rtgAsmStoreRaxStack(a, highOff)
		}
		rtgAsmLoadRaxStack(a, highOff)
		rtgAsmLoadRcxStack(a, lowOff)
		rtgAsmSubRaxRcx(a)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, baseOff)
		rtgAsmLoadRcxStack(a, lowOff)
		rtgAsmAddRaxRcx(a)
		rtgAsmPopRdx(a)
		return true
	}
	if e.right >= 0 {
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
	}
	return true
}
func rtgEmitSliceLiteralRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int) bool {
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
	backingSize := 32768
	needSize := e.argCount * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !rtgEmitSliceLiteralBacking(g, ep, idx, sliceType, backingOff) {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, e.argCount)
	rtgAsmPopRcx(a)
	return true
}
func rtgEmitSliceLiteralBacking(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int, backingOff int) bool {
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
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopStoreStringMemRdx(a, disp)
			continue
		}
		if elemResolved.kind == rtgTypeStruct {
			addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			if disp != 0 {
				rtgAsmAddRdxImm(a, disp)
			}
			rtgAsmStoreRdxStack(a, addrOffset)
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if !rtgTypeKindIsScalarInt(elemResolved.kind) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmNormalizeRaxForKind(a, elemResolved.kind)
		rtgAsmPushRax(a)
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDispSize(a, disp, elemSize)
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
	lenOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	capOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmStoreRaxStack(a, lenOffset)
	if e.argCount == 3 {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+2]) {
			return false
		}
		rtgAsmStoreRaxStack(a, capOffset)
	} else {
		rtgAsmLoadRaxStack(a, lenOffset)
		rtgAsmStoreRaxStack(a, capOffset)
	}
	backingSize := 32768
	lenConst := rtgEvalConstExpr(g, ep, ep.args[e.firstArg+1])
	if lenConst.ok && lenConst.value > 0 {
		needSize := lenConst.value * elemSize
		if needSize > backingSize {
			backingSize = rtgAlignTo8(needSize)
		}
	}
	if e.argCount == 3 {
		capConst := rtgEvalConstExpr(g, ep, ep.args[e.firstArg+2])
		if capConst.ok && capConst.value > 0 {
			needSize := capConst.value * elemSize
			if needSize > backingSize {
				backingSize = rtgAlignTo8(needSize)
			}
		}
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOffset)
	rtgAsmLoadRcxStack(a, capOffset)
	return true
}
func rtgEmitByteSliceConversionRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	idxOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	backingOff := g.asm.bssSize
	backingSize := 32768
	g.asm.bssSize += backingSize
	argIndex := ep.args[e.firstArg]
	if !rtgEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcOff)
	rtgAsmStoreRdxStack(a, lenOff)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, idxOff)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, lenOff)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcOff)
	rtgAsmPopRcx(a)
	rtgAsmLoadByteRaxIndexRcx(a)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRcx(a)
	rtgAsmPopRax(a)
	rtgAsmStoreAlMemRdxRcx1(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, idxOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOff)
	rtgAsmMovRcxRdx(a)
	return true
}
func rtgEmitCompositeFieldToStack(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, destOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
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
		rtgAsmStoreRaxStack(a, destOffset)
		rtgAsmStoreRdxStack(a, destOffset-8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgEmitCopyStackToStack(g, tempOffset, destOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, fieldResolved.kind)
	rtgAsmStoreRaxStack(a, destOffset)
	return true
}
func rtgEmitCopyStackToStack(g *rtgLinearGen, srcOffset int, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}
func rtgEmitCopyStackToMemRdx(g *rtgLinearGen, srcOffset int, destDisp int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		if destDisp != 0 {
			rtgAsmAddRdxImm(a, destDisp)
		}
		rtgAsmPushRdx(a)
		if rtgTargetArch == rtgArch386 {
			rtgAsmEmit8(a, 0x5f)
		} else {
			rtgAsmPopRdi(a)
		}
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxMemRdxDisp(a, destDisp+at)
	}
}
func rtgEmitCopyMemRdxToStack(g *rtgLinearGen, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmPushRdx(a)
		rtgAsmPopRsi(a)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxMemRdxDisp(a, at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}
func rtgEmitPushStackWords(g *rtgLinearGen, offset int, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxStack(&g.asm, offset-at)
		rtgAsmPushRax(&g.asm)
	}
}
func rtgEmitPushMemRdxWords(g *rtgLinearGen, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
		rtgAsmPushRax(&g.asm)
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
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, e.left) {
		return false
	}
	rtgAsmPopRcx(a)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmMovRdxRax(a)
	rtgAsmAddRdxRcx(a)
	rtgEmitCopyMemRdxToStack(g, offset, elemSize)
	return true
}
func rtgEmitStructCallToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !rtgTypeIsStruct(g.meta, g.meta.funcs[fnIndex].resultType) {
		if rtgTargetArch == rtgArchAarch64 {
			rtgPrintErr("rtg: aarch64 struct call target rejected\n")
		}
		return false
	}
	if rtgTypeSize(g.meta, destType) != rtgTypeSize(g.meta, g.meta.funcs[fnIndex].resultType) {
		if rtgTargetArch == rtgArchAarch64 {
			rtgPrintErr("rtg: aarch64 struct call size mismatch\n")
		}
		return false
	}
	wordCount := 1
	for i := e.argCount - 1; i >= 0; i-- {
		argIndex := ep.args[e.firstArg+i]
		words := rtgEmitCallArgReverse(g, ep, argIndex)
		if words < 0 {
			if rtgTargetArch == rtgArchAarch64 {
				rtgPrintErr("rtg: aarch64 struct call argument failed\n")
			}
			return false
		}
		wordCount += words
	}
	rtgAsmStackMem(&g.asm, offset, 0x8d48, 0x45, 0x85)
	rtgAsmPushRax(&g.asm)
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func rtgEmitUserCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		if rtgTargetArch == rtgArchAarch64 {
			rtgPrintErr("rtg: aarch64 user call target not found\n")
		}
		return rtgEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		if rtgTargetArch == rtgArchAarch64 {
			rtgPrintErr("rtg: aarch64 user call label missing\n")
		}
		return false
	}
	firstArg := e.firstArg
	argCount := e.argCount
	expanded := e.nameStart
	wordCount := 0
	fn := &g.meta.funcs[fnIndex]
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
			words := rtgEmitCallArgReverse(g, ep, argIndex)
			if words < 0 {
				if rtgTargetArch == rtgArchAarch64 {
					rtgPrintErr("rtg: aarch64 user call arg failed\n")
				}
				return false
			}
			wordCount += words
		}
	} else {
		for i := argCount - 1; i >= 0; i-- {
			argIndex := ep.args[firstArg+i]
			words := rtgEmitCallArgReverse(g, ep, argIndex)
			if words < 0 {
				if rtgTargetArch == rtgArchAarch64 {
					rtgPrintErr("rtg: aarch64 user call arg failed\n")
				}
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
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}
func rtgEmitMethodReceiverArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, receiverType int) int {
	meta := g.meta
	a := &g.asm
	receiver := rtgResolveType(meta, receiverType)
	exprType := rtgInferParsedExprType(g, ep, idx)
	exprResolved := rtgResolveType(meta, exprType)
	if receiver.kind == rtgTypePointer {
		if exprResolved.kind == rtgTypePointer {
			if !rtgEmitIntExpr(g, ep, idx) {
				return -1
			}
			rtgAsmPushRax(a)
			return 1
		}
		if !rtgEmitAddressRax(g, ep, idx) {
			return -1
		}
		rtgAsmPushRax(a)
		return 1
	}
	if receiver.kind == rtgTypeStruct && exprResolved.kind == rtgTypePointer {
		if !rtgEmitIntExpr(g, ep, idx) {
			return -1
		}
		rtgAsmMovRdxRax(a)
		size := rtgTypeSize(meta, receiverType)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
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
	receiverEp := rtgParseExpression(g.prog, start, dotTok)
	if !receiverEp.ok || len(receiverEp.exprs) == 0 {
		return -1
	}
	return rtgEmitMethodReceiverArgReverse(g, &receiverEp, len(receiverEp.exprs)-1, receiverType)
}
func rtgEmitAddressRax(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			rtgAsmLeaRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			rtgAsmMovRaxBssAddr(a, globalOffset)
			return true
		}
	}
	if e.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmMovRaxRdx(a)
		return true
	}
	if e.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, ep, idx) {
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
	if !rtgTypeKindIsScalarInt(elem.kind) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := count * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
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
			argEp := rtgParseExpression(g.prog, pos, argEnd)
			if !argEp.ok || len(argEp.exprs) == 0 {
				return false
			}
			rootIndex := len(argEp.exprs) - 1
			if !rtgEmitIntExpr(g, &argEp, rootIndex) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, elem.kind)
			disp := emitted * elemSize
			rtgAsmPushRax(a)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			rtgAsmStoreRaxMemRdxDispSize(a, disp, elemSize)
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
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, count)
	rtgAsmPopRcx(a)
	rtgAsmPushSliceRegs(a)
	return true
}
func rtgEmitCallArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	a := &g.asm
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsSlice(g.meta, typ) {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushSliceRegs(&g.asm)
		return 3
	}
	if rtgTypeIsString(g.meta, typ) {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushStringRegs(&g.asm)
		return 2
	}
	if rtgTypeIsTuple(g.meta, typ) {
		return rtgEmitTupleArgReverse(g, ep, idx, typ)
	}
	if rtgTypeIsStruct(g.meta, typ) {
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
		constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		if constResult.ok {
			rtgAsmPushImm(a, constResult.value)
			return 1
		}
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return -1
	}
	rtgAsmPushRax(a)
	return 1
}
func rtgEmitTupleArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	e := &ep.exprs[idx]
	if e.kind != rtgExprCall {
		return -1
	}
	offset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := rtgResolveType(g.meta, typ)
	wordCount := 0
	for i := tuple.count - 1; i >= 0; i-- {
		field := g.meta.fields[tuple.first+i]
		size := rtgTypeSize(g.meta, field.typ)
		if size < 8 {
			size = 8
		}
		rtgEmitPushStackWords(g, offset-field.offset, size)
		wordCount += size / 8
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
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return -1
		}
		rtgEmitPushStackWords(g, g.locals[localIndex].offset, size)
		return size / 8
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
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return -1
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != size {
			return -1
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return -1
		}
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprComposite {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
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
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	if e.kind == rtgExprCall {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
			return -1
		}
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	return -1
}
func rtgEmitAppendAssignGeneral(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse) bool {
	p := g.prog
	if len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || rtgExprIdentCode(p, ep, root.left) != rtgIdentAppend {
		return false
	}
	var loc rtgSliceLocation
	locEp := ep
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
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
	t := rtgResolveType(g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if root.nameStart == 1 {
		return rtgEmitAppendExpansionToLocation(g, ep, locEp, &loc, t.elem, valueIndex)
	}
	if elem.kind == rtgTypeStruct {
		value := &ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprUnary && rtgTokCharIs(p, value.tok, '*') {
				return rtgEmitAppendStructDeref(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokCharIs(p, typeTok+1, '{') {
					typeTok = 0
					for i := 0; i < len(p.toks); i++ {
						if p.toks[i].start == value.nameStart {
							typeTok = i
						}
					}
				}
				if rtgTokCharIs(p, typeTok+1, '{') {
					return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			typeTok := rtgFindAppendCompositeTypeToken(p, root.tok, stmt.endTok)
			if typeTok >= 0 {
				return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, locEp, &loc, t.elem, valueIndex) {
			return false
		}
		return true
	}
	if rtgTypeKindIsScalarInt(elem.kind) {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, &loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeString {
		if !rtgEmitAppendStringToLocation(g, ep, locEp, &loc, valueIndex) {
			return false
		}
		return true
	}
	return false
}
func rtgBinaryUsesFloat(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr) bool {
	p := g.prog
	if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
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
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
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
		return rtgExprCanFoldConst(g, ep, e.left)
	}
	if e.kind == rtgExprBinary {
		return rtgExprCanFoldConst(g, ep, e.left) && rtgExprCanFoldConst(g, ep, e.right)
	}
	if e.kind == rtgExprCall {
		if e.argCount != 1 || !rtgExprCanFoldConst(g, ep, ep.args[e.firstArg]) {
			return false
		}
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32 || callee == rtgIdentInt64 {
			return true
		}
		calleeExpr := &ep.exprs[e.left]
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
		if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmPushRdx(a)
		if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
		label := rtgEnsureAppendBytesHelper(g)
		rtgAsmCallLabel(a, label)
		return true
	}
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcIndex := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	headerOffset := 0
	if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, srcIndex)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		headerOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(a, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmStoreRaxStack(a, destLen)
	} else if loc.global {
		rtgAsmLoadRaxBss(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxBss(a, loc.offset+8)
		rtgAsmStoreRaxStack(a, destLen)
	} else {
		rtgAsmLoadRaxStack(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxStack(a, loc.offset-8)
		rtgAsmStoreRaxStack(a, destLen)
	}
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, srcIndex)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, destLen)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, destLen)
	if loc.mem {
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmStoreRaxMemRdxDisp(a, 8)
	} else if loc.global {
		rtgAsmStoreRaxBss(a, loc.offset+8)
	} else {
		rtgAsmStoreRaxStack(a, loc.offset-8)
	}
	return true
}
func rtgEmitAppendExpansionCopyElement(g *rtgLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	a := &g.asm
	if elemSize == 1 || elemSize == 2 || elemSize == 4 || elemSize == 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmLoadRaxIndexRcxSize(a, elemSize)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmLoadQwordRaxIndexRcxDisp(a, copyOff)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmAddRdxRcx(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDisp(a, copyOff)
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
	if !rtgEmitIntExpr(g, ep, valueIndex) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, elemKind)
	rtgAsmPushRax(a)
	if elemSize == 2 || elemSize == 4 {
		if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDispSize(a, 0, elemSize)
		return true
	}
	label := rtgEnsureAppendScalarHelper(g, elemKind)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmPopRdx(a)
	rtgAsmCallLabel(a, label)
	return true
}
func rtgEmitAppendDestRax(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	label := rtgEnsureAppendAddrHelper(g)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxImm(&g.asm, elemSize)
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
	if !rtgEmitAppendDestRax(g, locEp, loc, 16) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgAsmPopStoreStringMemRdx(a, 0)
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
}
func rtgEmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadRaxMemRdxDisp(a, 0)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	if rtgTargetArch == rtgArchWasm32 {
		backingSize = rtgWasm32FallbackSliceBackingSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxMemRdxDisp(a, 0)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxMemRdxDisp(a, 16)
	rtgAsmMarkLabel(a, okLabel)
}
func rtgEmitEnsureLocalSlice(g *rtgLinearGen, offset int, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadRaxStack(a, offset)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	if rtgTargetArch == rtgArchWasm32 {
		backingSize = rtgWasm32FallbackSliceBackingSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxStack(a, offset)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, offset-8)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxStack(a, offset-16)
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
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rtgAsmStoreRaxStack(&g.asm, destOffset)
	i := openTok + 1
	for i < closeTok-1 {
		if !rtgTokIsKind(p, i, rtgTokIdent) || !rtgTokCharIs(p, i+1, ':') {
			return false
		}
		fieldTok := p.toks[i]
		exprStart := i + 2
		exprEnd := rtgFindExprBoundary(p, exprStart, closeTok-1)
		ep := rtgParseExpression(p, exprStart, exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, elemType, fieldTok.start, fieldTok.end)
		if fieldOffset < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, elemType, fieldTok.start, fieldTok.end)
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
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitIntExpr(g, ep, value.left) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyMemRdxToStack(g, tempOffset, elemSize)
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
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
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, elemSize)
	return true
}
func rtgEmitAppendStructComposite(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitTypedAssign(g, ep, valueIndex, tempOffset) {
		return false
	}
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
	return true
}
func rtgEmitStringCompare(g *rtgLinearGen, ep *rtgExprParse, left int, right int, notEqual bool) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, left) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitStringValueRegs(g, ep, right) {
		return false
	}
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRdi(a)
	rtgAsmPopRsi(a)
	rtgAsmCallLabel(a, label)
	if notEqual {
		rtgAsmBoolNotRax(a)
	}
	return true
}
func rtgEmitBuiltinCopy(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	destIndex := ep.args[e.firstArg]
	srcIndex := ep.args[e.firstArg+1]
	destType := rtgInferParsedExprType(g, ep, destIndex)
	srcType := rtgInferParsedExprType(g, ep, srcIndex)
	destSlice := rtgResolveType(meta, destType)
	srcSlice := rtgResolveType(meta, srcType)
	if destSlice.kind != rtgTypeSlice || srcSlice.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(meta, destSlice.elem)
	if elemSize != rtgTypeSize(meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	count := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, destPtr)
	rtgAsmStoreRdxStack(a, destLen)
	if !rtgEmitSliceValueRegs(g, ep, srcIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, count)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, count, destPtr, count)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, count)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	return true
}
func rtgEmitSliceBasePtrLenTokens(g *rtgLinearGen, p *rtgProgram, start int, end int, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if start+1 == end && rtgTokIsKind(p, start, rtgTokIdent) {
		nameStart := p.toks[start].start
		nameEnd := p.toks[start].end
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, nameStart, nameEnd)
		globalType := rtgFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, '.') && rtgTokIsKind(p, start+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[start].start, p.toks[start].end)
		if localIndex < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if !rtgTypeIsSlice(meta, fieldType) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if fieldOffset < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
		} else {
			rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushRax(a)
			rtgAsmLoadRaxMemRdxDisp(a, 8)
			rtgAsmMovRcxRax(a)
			rtgAsmPopRax(a)
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
		rtgAsmMovRcxRdx(a)
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
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) && !rtgTypeIsString(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
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
		rtgAsmMovRcxRdx(a)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, fieldType) && !rtgTypeIsString(meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		if rtgTargetArch == rtgArchWasm32 {
			rtgAsmPushRax(a)
			rtgAsmLoadRaxMemRdxDisp(a, 8)
			rtgAsmMovRcxRax(a)
			rtgAsmPopRax(a)
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
		rtgAsmMovRcxRdx(a)
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
		rtgAsmMovRcxRdx(a)
		return true
	}
	return false
}
func rtgEmitIndexAddressRax(g *rtgLinearGen, ep *rtgExprParse, indexIdx int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
		return false
	}
	rtgAsmPopRcx(a)
	if elemSize != 1 {
		rtgAsmImulRcxImm(a, elemSize)
	}
	rtgAsmAddRaxRcx(a)
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
		rtgAsmPushRax(a)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
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
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
				t := rtgResolveType(meta, globalType)
				elem := rtgResolveType(meta, t.elem)
				if !rtgTypeKindIsScalarInt(elem.kind) {
					return false
				}
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				elemSize := rtgScalarKindSize(elem.kind)
				rtgAsmLoadRaxIndexRcxSize(a, elemSize)
				return true
			}
			constTok := rtgFindConstStringToken(g, left.nameStart, left.nameEnd)
			if constTok >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				msg := rtgDecodeStringToken(g.prog, constTok)
				msgOff := rtgAddStringData(g, msg)
				rtgAsmPushRax(a)
				rtgAsmMovRaxDataAddr(a, msgOff)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarInt(elem.kind) {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			elemSize := rtgScalarKindSize(elem.kind)
			rtgAsmLoadRaxIndexRcxSize(a, elemSize)
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
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarInt(elem.kind) {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			elemSize := rtgScalarKindSize(elem.kind)
			rtgAsmLoadRaxIndexRcxSize(a, elemSize)
			return true
		}
	}
	if left.kind == rtgExprUnary && rtgTokCharIs(p, left.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, left.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	if left.kind == rtgExprIndex {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitStringPtrExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
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
	for i := len(g.locals) - 1; i >= 0; i-- {
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
	for i := 0; i < t.count; i++ {
		field := &meta.fields[t.first+i]
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
	if size < 8 {
		size = 8
	}
	g.stackUsed = rtgAlignTo8(g.stackUsed + size)
	offset := g.stackUsed
	g.locals = append(g.locals, rtgLocalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: offset, typ: typ, size: size})
	return offset
}
func rtgZeroLocalAtOffset(g *rtgLinearGen, offset int) {
	a := &g.asm
	size := 8
	typ := rtgTypeInt
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind == rtgTypeSlice {
		elemSize := rtgTypeSize(g.meta, t.elem)
		if elemSize < 1 {
			elemSize = 8
		}
		backingSize := 2097152
		if rtgTargetArch == rtgArchWasm32 {
			backingSize = rtgWasm32FallbackSliceBackingSize
		}
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmStoreRaxStack(a, offset)
		rtgAsmMovRaxImm(a, 0)
		rtgAsmStoreRaxStack(a, offset-8)
		rtgAsmMovRaxImm(a, backingSize/elemSize)
		rtgAsmStoreRaxStack(a, offset-16)
		return
	}
	rtgAsmMovRaxImm(a, 0)
	for at := 0; at < size; at += 8 {
		rtgAsmStoreRaxStack(a, offset-at)
	}
}
func rtgFuncInfoFromCall(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	e := &ep.exprs[idx]
	nameStart := e.nameStart
	nameEnd := e.nameEnd
	wantMethod := false
	if e.kind == rtgExprSelector {
		wantMethod = true
	} else if e.kind != rtgExprIdent {
		return -1
	}
	if len(g.meta.funcBuckets) > 0 && len(g.meta.funcNext) == len(g.meta.funcs) {
		hash := rtgHashRange(g.prog.src, nameStart, nameEnd)
		bucket := hash % len(g.meta.funcBuckets)
		i := g.meta.funcBuckets[bucket]
		for i >= 0 {
			f := &g.meta.funcs[i]
			isMethod := f.receiverType != 0
			if isMethod == wantMethod && rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
				return i
			}
			i = g.meta.funcNext[i]
		}
		return -1
	}
	for i := 0; i < len(g.meta.funcs); i++ {
		f := &g.meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
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
func rtgStoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	if rtgTargetArch == rtgArchWasm32 {
		return rtgWasm32StoreParamWord(g, reg, offset)
	}
	if rtgTargetArch == rtgArchAarch64 {
		return rtgAarch64StoreParamWord(g, reg, offset)
	}
	if rtgTargetArch == rtgArchArm {
		return rtgArmStoreParamWord(g, reg, offset)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386StoreParamWord(g, reg, offset)
	}
	return rtgAmd64StoreParamWord(g, reg, offset)
}
func rtgAsmMovRaxImm(a *rtgAsm, imm int) {
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
func rtgAsmMovRaxImm64(a *rtgAsm, imm int) {
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
	rtgAmd64AsmMovRaxImm64(a, imm)
}
func rtgAsmMovRdxImm(a *rtgAsm, imm int) {
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
	rtgAmd64AsmMovRdxImm(a, imm)
}
func rtgAsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
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
	rtgAmd64AsmMovRaxDataAddr(a, dataOff)
}
func rtgAsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
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
	rtgAmd64AsmMovRaxBssAddr(a, bssOff)
}
func rtgAsmMovR10BssAddr(a *rtgAsm, bssOff int) {
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
func rtgAsmLoadRaxBss(a *rtgAsm, bssOff int) {
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
func rtgAsmStoreRaxBss(a *rtgAsm, bssOff int) {
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
func rtgAsmMovRdiRax(a *rtgAsm) {
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
func rtgAsmMovRaxRdx(a *rtgAsm) {
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
func rtgAsmMovRsiRax(a *rtgAsm) {
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
func rtgAsmMovR8Rax(a *rtgAsm) {
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
func rtgAsmMovR9Rax(a *rtgAsm) {
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
func rtgAsmAddRdxRcx(a *rtgAsm) {
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
func rtgAsmPopRdi(a *rtgAsm) {
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
func rtgAsmAddRdxImm(a *rtgAsm, imm int) {
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
func rtgAsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
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
func rtgAsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
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
func rtgAsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
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
func rtgAsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
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
func rtgAsmLoadByteRaxIndexRcx(a *rtgAsm) {
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
func rtgAsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
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
func rtgAsmStoreRaxMemRdxRcx8(a *rtgAsm) {
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
func rtgAsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
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
func rtgAsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
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
func rtgAsmNormalizeRaxForKind(a *rtgAsm, kind int) {
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
func rtgAsmIncMemRdx(a *rtgAsm) {
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
func rtgAsmDecMemRdx(a *rtgAsm) {
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
func rtgAsmBoolNotRax(a *rtgAsm) {
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
func rtgAsmCmpRaxImm8(a *rtgAsm, imm int) {
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
func rtgAsmAddRaxRcx(a *rtgAsm) {
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
func rtgAsmSubRaxRcx(a *rtgAsm) {
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
func rtgAsmShlRcxImm(a *rtgAsm, imm int) {
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
func rtgAsmShlRaxImm(a *rtgAsm, imm int) {
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
func rtgAsmSarRaxImm(a *rtgAsm, imm int) {
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
func rtgAsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
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
func rtgAsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
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
func rtgEmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
	if e.kind == rtgExprBinary && rtgTokCharIs(g.prog, e.tok, '+') && rtgTypeIsString(g.meta, rtgInferParsedExprType(g, ep, idx)) {
		return rtgEmitStringConcatValueRegs(g, ep, idx)
	}
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitStringValueRegs(g, ep, idx)
	}
	return rtgAmd64EmitStringValueRegs(g, ep, idx)
}

func rtgEmitStringConcatValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	byteSliceType := rtgAddType(g.meta, rtgTypeSlice, rtgTypeByte, 0, 0, 24, 0, 0)
	offset := rtgAddTypedLocal(g, 0, 0, byteSliceType)
	rtgZeroLocalAtOffset(g, offset)
	loc := rtgSliceLocation{offset: offset, typ: byteSliceType, ok: true}
	if !rtgEmitStringConcatIntoLocation(g, ep, idx, &loc) {
		return false
	}
	rtgAsmLoadRaxStack(&g.asm, offset)
	rtgAsmPushRax(&g.asm)
	rtgAsmLoadRaxStack(&g.asm, offset-8)
	rtgAsmMovRdxRax(&g.asm)
	rtgAsmPopRax(&g.asm)
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
	return rtgEmitAppendStringBytesToLocation(g, ep, idx, loc)
}

func rtgEmitAppendStringBytesToLocation(g *rtgLinearGen, ep *rtgExprParse, idx int, loc *rtgSliceLocation) bool {
	a := &g.asm
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmPushRax(a)
	rtgAsmPushRdx(a)
	if !rtgEmitSliceSlotAddrs(g, ep, loc, 1) {
		return false
	}
	rtgAsmPopRdx(a)
	rtgAsmPopRax(a)
	label := rtgEnsureAppendBytesHelper(g)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgEmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitCompositeFieldToMem(g, ep, idx, fieldType, addrOffset, fieldOffset)
	}
	return rtgAmd64EmitCompositeFieldToMem(g, ep, idx, fieldType, addrOffset, fieldOffset)
}
func rtgEmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitStructReturnExpr(g, ep, idx)
	}
	return rtgAmd64EmitStructReturnExpr(g, ep, idx)
}
func rtgEmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitNamedConversionCall(g, ep, idx)
	}
	return rtgAmd64EmitNamedConversionCall(g, ep, idx)
}
func rtgEmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
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
func rtgEmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitIntExpr(g, ep, idx)
	}
	return rtgAmd64EmitIntExpr(g, ep, idx)
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
func rtgEmitSelectorAddressRdx(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if rtgTargetArch == rtgArch386 {
		return rtg386EmitSelectorAddressRdx(g, ep, idx)
	}
	return rtgAmd64EmitSelectorAddressRdx(g, ep, idx)
}
