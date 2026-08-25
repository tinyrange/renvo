package main

const renvoAbsWinImportReloc = 2

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

func renvoAsmAddWinImportReloc(a *renvoAsm, at int, importID int) {
	renvoNonNil(a)
	renvoAsmAddAbsReloc(a, at, importID, renvoAbsWinImportReloc)
}

func renvoAsmAddWinStaticImport(a *renvoAsm, dllStart int, dllEnd int, nameStart int, nameEnd int, src []byte) int {
	renvoNonNil(a)
	dll := renvoStringFromBytes(src, dllStart, dllEnd)
	name := renvoStringFromBytes(src, nameStart, nameEnd)
	for i := 0; i < len(a.staticImports); i++ {
		imp := a.staticImports[i]
		if imp.dll == dll && imp.name == name {
			return renvoWinImportFixedCount + 1 + i
		}
	}
	a.staticImports = append(a.staticImports, renvoStaticImport{dll: dll, name: name})
	return renvoWinImportFixedCount + len(a.staticImports)
}

func renvoAsmHasWinImportRelocs(a *renvoAsm) bool {
	renvoNonNil(a)
	for i := 2; i < len(a.absRelocs); i += 3 {
		if int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) == renvoAbsWinImportReloc {
			return true
		}
	}
	return false
}

func renvoAppendWinImports(a *renvoAsm, layout *renvoWinImportLayout) {
	renvoNonNil(a, layout)
	thunkSize := 4
	if a.c.renvoTargetArch != renvoArch386 {
		thunkSize = 8
	}
	layout.thunkSize = thunkSize
	dataRVA := a.dataOffset
	if dataRVA == 0 {
		dataRVA = a.codeOffset + len(a.code)
	}
	layout.iatRVAs = make([]int, renvoWinImportFixedCount+len(a.staticImports)+1)
	groupCount := len(a.staticImports) + 1
	importOff := renvoAlignValue(len(a.data), thunkSize)
	a.data = renvoAppendUntil(a.data, importOff)
	descOff := importOff
	kernelILTOff := descOff + (groupCount+1)*20
	kernelIATOff := kernelILTOff + (renvoWinImportFixedCount+1)*thunkSize
	customTablesOff := kernelIATOff + (renvoWinImportFixedCount+1)*thunkSize
	tableOff := customTablesOff + len(a.staticImports)*4*thunkSize
	a.data = renvoAppendUntil(a.data, tableOff)

	for id := 1; id <= renvoWinImportFixedCount; id++ {
		slot := (id - 1) * thunkSize
		renvoAppendWinImportEntry(a, layout, kernelILTOff+slot, kernelIATOff+slot, id, renvoWinImportName(id))
	}
	for i := 0; i < len(a.staticImports); i++ {
		imp := a.staticImports[i]
		iltOff := customTablesOff + i*4*thunkSize
		renvoAppendWinImportEntry(a, layout, iltOff, iltOff+2*thunkSize, renvoWinImportFixedCount+1+i, imp.name)
	}
	for i := 0; i < groupCount; i++ {
		iltOff := kernelILTOff
		iatOff := kernelIATOff
		dllNameOff := len(a.data)
		dll := "kernel32.dll"
		if i > 0 {
			dll = a.staticImports[i-1].dll
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
	layout.kernelIATRVA = renvo_runtime_UnsafeIntAt(layout.iatRVAs, renvoWinImportGetStdHandle)
}

func renvoAppendWinImportEntry(a *renvoAsm, layout *renvoWinImportLayout, iltAt int, iatAt int, id int, name string) {
	renvoNonNil(a, layout)
	nameAt := len(a.data)
	a.data = renvoAppend16(a.data, 0)
	a.data = renvoAppendStringZ(a.data, name)
	if len(a.data)&1 != 0 {
		a.data = append(a.data, 0)
	}
	nameRVA := a.dataOffset + nameAt
	renvoPut32At(a.data, iltAt, nameRVA)
	renvoPut32At(a.data, iatAt, nameRVA)
	layout.iatRVAs[id] = a.dataOffset + iatAt
}

func renvoAsmPatchWindows(a *renvoAsm, layout renvoWinImportLayout) {
	renvoNonNil(a)
	for i := 0; i+1 < len(a.relocs); i += 2 {
		at := int(renvo_runtime_UnsafeInt32At(a.relocs, i))
		label := int(renvo_runtime_UnsafeInt32At(a.relocs, i+1))
		if label < 0 {
			continue
		}
		target := renvoAsmLabelPosition(a, label)
		if target < 0 {
			continue
		}
		disp := target - (at + 4)
		renvoPut32At(a.code, at, disp)
	}
	if a.dataOffset == 0 {
		a.dataOffset = a.codeOffset + len(a.code)
	}
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		if kind == renvoAbsWinImportReloc {
			target := renvo_runtime_UnsafeIntAt(layout.iatRVAs, off)
			if a.c.renvoTargetArch != renvoArch386 {
				next := a.codeOffset + at + 4
				renvoPut32At(a.code, at, target-next)
			} else {
				renvoPut32At(a.code, at, renvoWinImageBase+target)
			}
			continue
		}
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		if a.c.renvoTargetArch != renvoArch386 {
			next := a.codeOffset + at + 4
			renvoPut32At(a.code, at, target-next)
		} else {
			renvoPut32At(a.code, at, renvoWinImageBase+target)
		}
	}
}

func renvoAppendPEHeader64(out []byte, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	return renvoAppendPEHeader64WithContext(renvoLegacyCompileContext(), out, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, importRVA, importSize, iatRVA, iatSize)
}

func renvoAppendPEHeader64WithContext(context *renvoCompileContext, out []byte, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	machine := 0x8664
	imageBase := renvoWinImageBase
	stackReserve := 0x800000
	stackCommit := 0x1000
	if context.renvoTargetArch == renvoArchAarch64 {
		machine = 0xaa64
		imageBase = 0x140000000
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
	renvoPut32At(out, opt+68, 0x01000000|context.windowsSubsystem)
	renvoPut32At(out, opt+72, stackReserve)
	renvoPut32At(out, opt+80, stackCommit)
	renvoPut32At(out, opt+120, importRVA)
	renvoPut32At(out, opt+124, importSize)
	renvoPut32At(out, opt+208, iatRVA)
	renvoPut32At(out, opt+212, iatSize)
	out = renvoAppendPESection(out, textVirtualSize, renvoWinSectionRVA, textRawSize, renvoWinHeadersSize, 0x60000020)
	out = renvoAppendPESection(out, dataVirtualSize, dataRVA, dataRawSize, renvoWinHeadersSize+textRawSize, 0xc0000040)
	out = renvoAppendUntil(out, renvoWinHeadersSize)
	return out
}

func renvoAppendPEHeader32(out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	return renvoAppendPEHeader32WithContext(renvoLegacyCompileContext(), out, entryRVA, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, importRVA, importSize, iatRVA, iatSize)
}

func renvoAppendPEHeader32WithContext(context *renvoCompileContext, out []byte, entryRVA int, textRawSize int, textVirtualSize int, dataRVA int, dataRawSize int, dataVirtualSize int, importRVA int, importSize int, iatRVA int, iatSize int) []byte {
	sizeOfImage := renvoAlignValue(dataRVA+dataVirtualSize, renvoWinSectionAlign)
	out = renvoAppend32(out, 0x5a4d)
	out = renvoAppendUntil(out, 0x3c)
	out = renvoAppend32(out, 0x80)
	out = renvoAppendUntil(out, 0x80)
	pe := len(out)
	// Start with the invariant COFF and PE32 fields, then patch image-specific
	// sizes and directories before appending the two section headers.
	out = renvoAppendStringZ(out, "\x50\x45\x00\x00\x4c\x01\x02\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\xe0\x00\x02\x01\x0b\x01\x01\x00\x00\x02\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00\x10\x00\x00\x00\x20\x00\x00\x00\x00\x40\x00\x00\x10\x00\x00\x00\x02\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x04\x00\x00\x00\x00\x00\x00\x00\x00\x30\x00\x04\x00\x02\x00\x00\x00\x00\x00\x00\x03\x00\x00\x01\x00\x00\x10\x00\x00\x00\x10\x00\x00\x00\x10\x00\x00\x10\x00\x00\x00\x00\x00\x00\x10\x00\x00\x00")
	out = renvoAppendUntil(out, pe+248)
	opt := pe + 24
	renvoPut32At(out, opt+4, textRawSize)
	renvoPut32At(out, opt+8, dataRawSize)
	renvoPut32At(out, opt+16, entryRVA)
	renvoPut32At(out, opt+24, dataRVA)
	renvoPut32At(out, opt+56, sizeOfImage)
	renvoPut32At(out, opt+68, 0x01000000|context.windowsSubsystem)
	renvoPut32At(out, opt+104, importRVA)
	renvoPut32At(out, opt+108, importSize)
	renvoPut32At(out, opt+192, iatRVA)
	renvoPut32At(out, opt+196, iatSize)
	out = renvoAppendPESection(out, textVirtualSize, renvoWinSectionRVA, textRawSize, renvoWinHeadersSize, 0x60000020)
	out = renvoAppendPESection(out, dataVirtualSize, dataRVA, dataRawSize, renvoWinHeadersSize+textRawSize, 0xc0000040)
	out = renvoAppendUntil(out, renvoWinHeadersSize)
	return out
}

func renvoAppendPESection(out []byte, virtualSize int, rva int, rawSize int, rawPtr int, characteristics int) []byte {
	start := len(out)
	out = renvoAppendUntil(out, start+40)
	name := 0x7461642e
	tail := byte('a')
	if characteristics == 0x60000020 {
		name = 0x7865742e
		tail = 't'
	}
	renvoPut32At(out, start, name)
	renvoPut32At(out, start+4, int(tail))
	renvoPut32At(out, start+8, virtualSize)
	renvoPut32At(out, start+12, rva)
	renvoPut32At(out, start+16, rawSize)
	renvoPut32At(out, start+20, rawPtr)
	renvoPut32At(out, start+36, characteristics)
	return out
}
