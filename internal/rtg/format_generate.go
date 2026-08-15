package rtg

import "strings"

func declarativeFormatImage(format Declaration) string {
	for i := 0; i < len(format.Statements); i++ {
		left, right, assignment := statementAssignment(format.Statements[i])
		if assignment && len(left) == 1 && left[0] == "image" && len(right) == 1 {
			return right[0]
		}
	}
	return ""
}

func declarativeFormatImageName(document Document, format Declaration) string {
	return "rtg" + exportedName(document.Unit) + exportedName(format.Name) + "Image"
}

func declarativeFormatAbsoluteRelocations(format Declaration) string {
	return declarativeFormatMarker(format, "patch_absolute",
		[]string{"relative32_le", "aarch64_page_address", "arm32_mov_address"})
}

func declarativeFormatImageRelocations(format Declaration) string {
	return declarativeFormatMarker(format, "patch_image",
		[]string{"pe_relative32", "pe_absolute32"})
}

func declarativeFormatMarker(format Declaration, field string, allowed []string) string {
	for i := 0; i < len(format.Statements); i++ {
		left, right, assignment := statementAssignment(format.Statements[i])
		if !assignment || len(left) != 1 || left[0] != field || len(right) != 1 {
			continue
		}
		for j := 0; j < len(allowed); j++ {
			if right[0] == allowed[j] {
				return right[0]
			}
		}
	}
	return ""
}

func declarativeFormatAbsoluteRelocationName(document Document, format Declaration) string {
	return "rtg" + exportedName(document.Unit) + exportedName(format.Name) + "PatchAbsolute"
}

func declarativeFormatImageRelocationName(document Document, format Declaration) string {
	return "rtg" + exportedName(document.Unit) + exportedName(format.Name) + "PatchImage"
}

func declarativeFormatKernelImage(format Declaration) string {
	for i := 0; i < len(format.Statements); i++ {
		left, right, assignment := statementAssignment(format.Statements[i])
		if assignment && len(left) == 1 && left[0] == "kernel_image" && len(right) == 1 {
			return right[0]
		}
	}
	return ""
}

func declarativeFormatKernelImageName(document Document, format Declaration) string {
	return "rtg" + exportedName(document.Unit) + exportedName(format.Name) + "KernelImage"
}

func appendDeclarativeKernelImage(out []byte, document Document, format Declaration,
	nativeEmitter bool) []byte {
	if declarativeFormatKernelImage(format) != "linux_module_elf" {
		return out
	}
	functionName := declarativeFormatKernelImageName(document, format)
	prefix := functionName[:len(functionName)-len("KernelImage")]
	emitter := "RTGEmitter"
	align := "RTGAlignValue"
	if nativeEmitter {
		emitter = "renvoAsm"
		align = "renvoAlignValue"
	}
	hookName := func(field string) string {
		algorithm, _ := architectureGoHook(format, field)
		if nativeEmitter {
			for i := 0; i < len(document.Declarations); i++ {
				declaration := document.Declarations[i]
				if declaration.Kind != DeclArch {
					continue
				}
				if external, found := embeddedExternalName(
					architectureExports(declaration), algorithm); found {
					return external
				}
			}
		}
		return "rtg" + exportedName(document.Unit) + exportedName(algorithm)
	}
	replacements := []string{
		"LINUXMODULE", prefix,
		"MODULEEMITTER", emitter,
		"MODULEALIGN", align,
		"MODULEAPPEND64", hookName("append64"),
		"MODULEHEADER", hookName("append_header"),
		"MODULERELOCATION", hookName("append_relocation"),
		"MODULESECTION", hookName("append_section"),
		"MODULESYMBOL", hookName("append_symbol"),
	}
	source := declarativeLinuxModuleSource
	for i := 0; i < len(replacements); i += 2 {
		source = strings.ReplaceAll(source, replacements[i], replacements[i+1])
	}
	source = strings.ReplaceAll(source, prefix+"ModuleImage", functionName)
	out = append(out, "\n// Generated from the declarative Linux-module ELF format.\n"...)
	return append(out, source...)
}

func appendDeclarativeFormatImage(out []byte, document Document, target ResolvedTarget, nativeEmitter bool) []byte {
	format := target.Executable
	if declarativeFormatImage(format) == "pe_executable" {
		return appendDeclarativePEImage(out, document, target, nativeEmitter)
	}
	if declarativeFormatImage(format) != "elf_executable" {
		return out
	}
	bits, _ := integerField(document, format, "address_bits")
	machine, _ := integerField(document, format, "machine")
	flags, _ := integerField(document, format, "flags")
	codeOffset, _ := integerField(document, format, "code_offset")
	alignment, _ := integerField(document, format, "file_alignment")
	patch, _ := architectureGoHook(format, "patch_absolute")
	prefix := "rtg" + exportedName(document.Unit)
	patchName := prefix + exportedName(patch)
	if declarativeFormatAbsoluteRelocations(format) != "" {
		patchName = declarativeFormatAbsoluteRelocationName(document, format)
		out = appendDeclarativeAbsoluteRelocationPatcher(
			out, document, format, nativeEmitter)
	}

	out = append(out, "\n// Generated directly from the declarative ELF executable format.\nfunc "...)
	out = append(out, declarativeFormatImageName(document, format)...)
	out = append(out, "(a *"...)
	if nativeEmitter {
		out = append(out, "renvoAsm"...)
	} else {
		out = append(out, "RTGEmitter"...)
	}
	out = append(out, ") []byte {\n"...)
	out = append(out, "\tcodeOffset := "...)
	out = appendDecimalFrame(out, codeOffset)
	out = append(out, "\n\tdataOffset := codeOffset + len(a.Code())\n"...)
	out = append(out, "\tfileSize := dataOffset + len(a.Data())\n"...)
	out = append(out, "\tbssOffset := (fileSize + "...)
	out = appendDecimalFrame(out, alignment-1)
	out = append(out, ") & -"...)
	out = appendDecimalFrame(out, alignment)
	out = append(out, "\n\t"...)
	out = append(out, patchName...)
	out = append(out, "(a, codeOffset, dataOffset, bssOffset)\n"...)
	out = append(out, "\tcode := a.Code()\n\tdata := a.Data()\n\tvar result []byte\n"...)
	if bits == 64 {
		out = append(out, "\tresult = append(result, 0x7f, 'E', 'L', 'F', 2, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0)\n"...)
	} else {
		out = append(out, "\tresult = append(result, 0x7f, 'E', 'L', 'F', 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0)\n"...)
	}
	out = appendFormatInteger(out, 2, "3")
	out = appendFormatInteger(out, 2, decimalFrame(machine))
	out = appendFormatInteger(out, 4, "1")
	if bits == 64 {
		out = appendFormatInteger(out, 8, "codeOffset")
		out = appendFormatInteger(out, 8, "64")
		out = appendFormatInteger(out, 8, "0")
		out = appendFormatInteger(out, 4, decimalFrame(flags))
		out = appendFormatInteger(out, 2, "64")
		out = appendFormatInteger(out, 2, "56")
	} else {
		out = appendFormatInteger(out, 4, "codeOffset")
		out = appendFormatInteger(out, 4, "52")
		out = appendFormatInteger(out, 4, "0")
		out = appendFormatInteger(out, 4, decimalFrame(flags))
		out = appendFormatInteger(out, 2, "52")
		out = appendFormatInteger(out, 2, "32")
	}
	out = appendFormatInteger(out, 2, "2")
	out = appendFormatInteger(out, 2, "0")
	out = appendFormatInteger(out, 2, "0")
	out = appendFormatInteger(out, 2, "0")
	if bits == 64 {
		out = appendELF64ProgramHeaders(out, alignment)
	} else {
		out = appendELF32ProgramHeaders(out, alignment)
	}
	out = append(out, "\tresult = append(result, code...)\n\tresult = append(result, data...)\n\treturn result\n}\n"...)
	return out
}

func appendDeclarativeAbsoluteRelocationPatcher(out []byte, document Document,
	format Declaration, nativeEmitter bool) []byte {
	encoding := declarativeFormatAbsoluteRelocations(format)
	if encoding == "" {
		return out
	}
	emitter := "RTGEmitter"
	get32 := "RTGGet32At"
	if nativeEmitter {
		emitter = "renvoAsm"
		get32 = "renvoGet32At"
	}
	out = append(out, "\n// Generated from the declarative executable relocation encoding.\nfunc "...)
	out = append(out, declarativeFormatAbsoluteRelocationName(document, format)...)
	out = append(out, "(a *"...)
	out = append(out, emitter...)
	out = append(out, ", codeOffset int, dataOffset int, bssOffset int) {\n"...)
	out = append(out, "\tfor i := 0; i < a.AbsoluteRelocationCount(); i++ {\n"...)
	out = append(out, "\t\tat := a.AbsoluteRelocationOffset(i)\n"...)
	out = append(out, "\t\taddend := a.AbsoluteRelocationAddend(i)\n"...)
	out = append(out, "\t\tkind := a.AbsoluteRelocationKind(i)\n"...)
	resolver := "RTGResolveAbsoluteRelocation"
	if nativeEmitter {
		resolver = "renvoRTGResolveAbsoluteRelocation"
	}
	out = append(out, "\t\ttarget, resolved := "...)
	out = append(out, resolver...)
	out = append(out, "(a, kind, addend, dataOffset, bssOffset)\n"...)
	out = append(out, "\t\tif !resolved { continue }\n"...)
	if encoding == "relative32_le" {
		bias := formatIntegerDefault(document, format, "patch_pc_bias", 4)
		out = append(out, "\t\ta.PatchUint32(at, target-(codeOffset+at"...)
		if bias < 0 {
			out = append(out, '-')
			bias = -bias
		} else {
			out = append(out, '+')
		}
		out = appendDecimalFrame(out, bias)
		out = append(out, "))\n"...)
	} else if encoding == "aarch64_page_address" {
		out = append(out, "\t\tinsn := "...)
		out = append(out, get32...)
		out = append(out, "(a.Code(), at)\n"...)
		out = append(out, declarativeAArch64AbsoluteRelocationBody...)
	} else {
		helper, _ := architectureGoHook(format, "patch_address")
		out = append(out, "\t\tinsn := "...)
		out = append(out, get32...)
		out = append(out, "(a.Code(), at)\n\t\treg := (insn >> 12) & 15\n\t\t"...)
		out = append(out, "rtg"...)
		out = append(out, exportedName(document.Unit)...)
		out = append(out, exportedName(helper)...)
		out = append(out, "(a, at, reg, target-(codeOffset+at+16))\n"...)
	}
	out = append(out, "\t}\n}\n"...)
	return out
}

const declarativeAArch64AbsoluteRelocationBody = `
		reg := insn & 31
		pcPage := (codeOffset + at) & -4096
		targetPage := target & -4096
		pageDelta := (targetPage - pcPage) >> 12
		immlo := pageDelta & 3
		immhi := pageDelta >> 2 & 524287
		a.PatchUint32(at, 0x90000000|(immlo<<29)|(immhi<<5)|reg)
		a.PatchUint32(at+4, 0x91000000|((target&4095)<<10)|(reg<<5)|reg)
		a.PatchUint32(at+8, 0xd503201f)
		a.PatchUint32(at+12, 0xd503201f)
`

func decimalFrame(value int) string {
	var out []byte
	out = appendDecimalFrame(out, value)
	return string(out)
}

func appendFormatInteger(out []byte, width int, expression string) []byte {
	out = append(out, "\tresult = append(result"...)
	for byteIndex := 0; byteIndex < width; byteIndex++ {
		out = append(out, ", byte(("...)
		out = append(out, expression...)
		out = append(out, ") >> "...)
		out = appendDecimalFrame(out, byteIndex*8)
		out = append(out, ')')
	}
	out = append(out, ")\n"...)
	return out
}

func appendELF64ProgramHeaders(out []byte, alignment int) []byte {
	out = appendFormatInteger(out, 4, "1")
	out = appendFormatInteger(out, 4, "5")
	out = appendFormatInteger(out, 8, "0")
	out = appendFormatInteger(out, 8, "0")
	out = appendFormatInteger(out, 8, "0")
	out = appendFormatInteger(out, 8, "fileSize")
	out = appendFormatInteger(out, 8, "fileSize")
	out = appendFormatInteger(out, 8, decimalFrame(alignment))
	out = appendFormatInteger(out, 4, "1")
	out = appendFormatInteger(out, 4, "6")
	out = appendFormatInteger(out, 8, "bssOffset")
	out = appendFormatInteger(out, 8, "bssOffset")
	out = appendFormatInteger(out, 8, "bssOffset")
	out = appendFormatInteger(out, 8, "0")
	out = appendFormatInteger(out, 8, "a.BSSSize()")
	return appendFormatInteger(out, 8, decimalFrame(alignment))
}

func appendELF32ProgramHeaders(out []byte, alignment int) []byte {
	out = appendFormatInteger(out, 4, "1")
	out = appendFormatInteger(out, 4, "0")
	out = appendFormatInteger(out, 4, "0")
	out = appendFormatInteger(out, 4, "0")
	out = appendFormatInteger(out, 4, "fileSize")
	out = appendFormatInteger(out, 4, "fileSize")
	out = appendFormatInteger(out, 4, "5")
	out = appendFormatInteger(out, 4, decimalFrame(alignment))
	out = appendFormatInteger(out, 4, "1")
	out = appendFormatInteger(out, 4, "bssOffset")
	out = appendFormatInteger(out, 4, "bssOffset")
	out = appendFormatInteger(out, 4, "bssOffset")
	out = appendFormatInteger(out, 4, "0")
	out = appendFormatInteger(out, 4, "a.BSSSize()")
	out = appendFormatInteger(out, 4, "6")
	return appendFormatInteger(out, 4, decimalFrame(alignment))
}

func runtimePEImports(runtime Declaration) (string, []string) {
	for i := 0; i < len(runtime.Statements); i++ {
		tokens := runtime.Statements[i].Tokens
		if len(tokens) < 6 || tokens[0] != "imports" || tokens[2] != "=" || tokens[3] != "[" {
			continue
		}
		library := valueName(tokens[1])
		var names []string
		for j := 4; j < len(tokens) && tokens[j] != "]"; j++ {
			if tokens[j] != "," {
				names = append(names, valueName(tokens[j]))
			}
		}
		return library, names
	}
	return "", nil
}

func formatIntegerDefault(document Document, format Declaration, name string, fallback int) int {
	if value, ok := integerField(document, format, name); ok {
		return value
	}
	return fallback
}

func appendDeclarativePEImage(out []byte, document Document, target ResolvedTarget, nativeEmitter bool) []byte {
	format := target.Executable
	prefix := declarativeFormatImageName(document, format)
	prefix = prefix[:len(prefix)-len("Image")]
	emitter := "RTGEmitter"
	align := "RTGAlignValue"
	if nativeEmitter {
		emitter = "renvoAsm"
		align = "renvoAlignValue"
	}
	bits, _ := integerField(document, format, "address_bits")
	machine, _ := integerField(document, format, "machine")
	codeOffset, _ := integerField(document, format, "code_offset")
	fileAlignment, _ := integerField(document, format, "file_alignment")
	sectionAlignment, _ := integerField(document, format, "section_alignment")
	imageBase, _ := integerField(document, format, "image_base")
	imageBaseHigh, _ := integerField(document, format, "image_base_high")
	patch, _ := architectureGoHook(format, "patch_image")
	patchName := "rtg" + exportedName(document.Unit) + exportedName(patch)
	if declarativeFormatImageRelocations(format) != "" {
		patchName = declarativeFormatImageRelocationName(document, format)
		out = appendDeclarativePEImagePatcher(out, document, format, nativeEmitter)
	}
	coffCharacteristics := formatIntegerDefault(document, format, "coff_characteristics", 0x22)
	dllCharacteristics := formatIntegerDefault(document, format, "dll_characteristics", 0x100)
	osMajor := formatIntegerDefault(document, format, "os_version_major", 4)
	osMinor := formatIntegerDefault(document, format, "os_version_minor", 0)
	imageMajor := formatIntegerDefault(document, format, "image_version_major", 0)
	imageMinor := formatIntegerDefault(document, format, "image_version_minor", 0)
	subsystemMajor := formatIntegerDefault(document, format, "subsystem_version_major", 4)
	subsystemMinor := formatIntegerDefault(document, format, "subsystem_version_minor", 0)
	stackReserve := formatIntegerDefault(document, format, "stack_reserve", 0x800000)
	stackCommit := formatIntegerDefault(document, format, "stack_commit", 0x1000)
	heapReserve := formatIntegerDefault(document, format, "heap_reserve", 0x100000)
	heapCommit := formatIntegerDefault(document, format, "heap_commit", 0x1000)
	library, imports := runtimePEImports(target.Runtime)
	var importSource []byte
	importSource = append(importSource, "[]string{"...)
	for i := 0; i < len(imports); i++ {
		if i != 0 {
			importSource = append(importSource, ',')
		}
		importSource = appendQuoted(importSource, imports[i])
	}
	importSource = append(importSource, '}')

	replacements := []string{
		"PEPREFIX", prefix,
		"PEEMITTER", emitter,
		"PEALIGN", align,
		"PEPATCH", patchName,
		"PEBITS", decimalFrame(bits),
		"PEMACHINE", decimalFrame(machine),
		"PECODEOFFSET", decimalFrame(codeOffset),
		"PEFILEALIGN", decimalFrame(fileAlignment),
		"PESECTIONALIGN", decimalFrame(sectionAlignment),
		"PEIMAGEBASEHIGH", decimalFrame(imageBaseHigh),
		"PEIMAGEBASE", decimalFrame(imageBase),
		"PECOFFCHAR", decimalFrame(coffCharacteristics),
		"PEDLLCHAR", decimalFrame(dllCharacteristics),
		"PEOSMAJOR", decimalFrame(osMajor),
		"PEOSMINOR", decimalFrame(osMinor),
		"PEIMAGEMAJOR", decimalFrame(imageMajor),
		"PEIMAGEMINOR", decimalFrame(imageMinor),
		"PESUBMAJOR", decimalFrame(subsystemMajor),
		"PESUBMINOR", decimalFrame(subsystemMinor),
		"PESTACKRESERVE", decimalFrame(stackReserve),
		"PESTACKCOMMIT", decimalFrame(stackCommit),
		"PEHEAPRESERVE", decimalFrame(heapReserve),
		"PEHEAPCOMMIT", decimalFrame(heapCommit),
		"PELIBRARY", string(appendQuoted(nil, library)),
		"PEIMPORTS", string(importSource),
	}
	source := declarativePEImageSource
	for i := 0; i < len(replacements); i += 2 {
		source = strings.ReplaceAll(source, replacements[i], replacements[i+1])
	}
	out = append(out, "\n// Generated directly from the declarative PE executable format.\n"...)
	return append(out, source...)
}

func appendDeclarativePEImagePatcher(out []byte, document Document, format Declaration,
	nativeEmitter bool) []byte {
	encoding := declarativeFormatImageRelocations(format)
	if encoding == "" {
		return out
	}
	emitter := "RTGEmitter"
	if nativeEmitter {
		emitter = "renvoAsm"
	}
	codeOffset, _ := integerField(document, format, "code_offset")
	imageBase, _ := integerField(document, format, "image_base")
	out = append(out, "\n// Generated from the declarative PE relocation encoding.\nfunc "...)
	out = append(out, declarativeFormatImageRelocationName(document, format)...)
	out = append(out, "(out *"...)
	out = append(out, emitter...)
	out = append(out, ", dataRVA int, dataSize int, iat []int) {\n"...)
	out = append(out, "\tfor i := 0; i < out.AbsoluteRelocationCount(); i++ {\n"...)
	out = append(out, "\t\tat := out.AbsoluteRelocationOffset(i)\n"...)
	out = append(out, "\t\taddend := out.AbsoluteRelocationAddend(i)\n"...)
	out = append(out, "\t\tkind := out.AbsoluteRelocationKind(i)\n"...)
	resolver := "RTGResolveAbsoluteRelocation"
	if nativeEmitter {
		resolver = "renvoRTGResolveAbsoluteRelocation"
	}
	out = append(out, "\t\ttarget, resolved := "...)
	out = append(out, resolver...)
	out = append(out, "(out, kind, addend, dataRVA, dataRVA+dataSize)\n"...)
	out = append(out, "\t\tif kind == RTGRelocationImport && addend > 0 && addend < len(iat) { target = iat[addend]; resolved = true }\n"...)
	out = append(out, "\t\tif !resolved { continue }\n"...)
	if encoding == "pe_relative32" {
		out = append(out, "\t\tout.PatchUint32(at, target-("...)
		out = appendDecimalFrame(out, codeOffset)
		out = append(out, "+at+4))\n"...)
	} else {
		out = append(out, "\t\tout.PatchUint32(at, "...)
		out = appendDecimalFrame(out, imageBase)
		out = append(out, "+target)\n"...)
	}
	out = append(out, "\t}\n}\n"...)
	return out
}

const declarativePEImageSource = `
const PEPREFIXWordBytes = PEBITS / 8

func PEPREFIXPut16(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
}

func PEPREFIXPut32(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}

func PEPREFIXPutWord(out []byte, at int, value int) {
	PEPREFIXPut32(out, at, value)
	if PEPREFIXWordBytes == 8 {
		PEPREFIXPut32(out, at+4, value>>32)
	}
}

func PEPREFIXUntil(out []byte, size int) []byte {
	for len(out) < size {
		out = append(out, 0)
	}
	return out
}

func PEPREFIXStringZ(out []byte, value string) []byte {
	out = append(out, []byte(value)...)
	return append(out, 0)
}

func PEPREFIXImportEntry(data []byte, iat []int, dataRVA int, iltAt int, iatAt int, slot int, id int, name string) []byte {
	nameAt := len(data)
	data = append(data, 0, 0)
	data = PEPREFIXStringZ(data, name)
	if len(data)&1 != 0 {
		data = append(data, 0)
	}
	nameRVA := dataRVA + nameAt
	PEPREFIXPutWord(data, iltAt+slot*PEPREFIXWordBytes, nameRVA)
	PEPREFIXPutWord(data, iatAt+slot*PEPREFIXWordBytes, nameRVA)
	iat[id] = dataRVA + iatAt + slot*PEPREFIXWordBytes
	return data
}

func PEPREFIXBuildImports(out *PEEMITTER, data []byte, dataRVA int) ([]byte, int, int, int, int, []int) {
	names := PEIMPORTS
	staticCount := out.StaticImportCount()
	importAt := PEALIGN(len(data), PEPREFIXWordBytes)
	data = PEPREFIXUntil(data, importAt+(staticCount+2)*20)
	iltAt := len(data)
	data = PEPREFIXUntil(data, iltAt+(len(names)+1)*PEPREFIXWordBytes)
	iatAt := len(data)
	data = PEPREFIXUntil(data, iatAt+(len(names)+1)*PEPREFIXWordBytes)
	customAt := len(data)
	data = PEPREFIXUntil(data, customAt+staticCount*4*PEPREFIXWordBytes)
	iat := make([]int, len(names)+1+staticCount)
	for index := 0; index < len(names); index++ {
		data = PEPREFIXImportEntry(data, iat, dataRVA, iltAt, iatAt, index, index+1, names[index])
	}
	for index := 0; index < staticCount; index++ {
		customILT := customAt + index*4*PEPREFIXWordBytes
		customIAT := customILT + 2*PEPREFIXWordBytes
		data = PEPREFIXImportEntry(data, iat, dataRVA, customILT, customIAT, 0, len(names)+1+index, out.StaticImportName(index))
	}
	dllAt := len(data)
	data = PEPREFIXStringZ(data, PELIBRARY)
	PEPREFIXPut32(data, importAt, dataRVA+iltAt)
	PEPREFIXPut32(data, importAt+12, dataRVA+dllAt)
	PEPREFIXPut32(data, importAt+16, dataRVA+iatAt)
	for index := 0; index < staticCount; index++ {
		customILT := customAt + index*4*PEPREFIXWordBytes
		customIAT := customILT + 2*PEPREFIXWordBytes
		dllAt = len(data)
		data = PEPREFIXStringZ(data, out.StaticImportDLL(index))
		descriptor := importAt + (index+1)*20
		PEPREFIXPut32(data, descriptor, dataRVA+customILT)
		PEPREFIXPut32(data, descriptor+12, dataRVA+dllAt)
		PEPREFIXPut32(data, descriptor+16, dataRVA+customIAT)
	}
	return data, dataRVA+importAt, len(data)-importAt, dataRVA+iatAt,
		customAt+staticCount*4*PEPREFIXWordBytes-iatAt, iat
}

func PEPREFIXSection(out []byte, name string, virtualSize int, rva int, rawSize int, rawAt int, flags int) []byte {
	start := len(out)
	out = PEPREFIXUntil(out, start+40)
	for index := 0; index < len(name) && index < 8; index++ {
		out[start+index] = name[index]
	}
	PEPREFIXPut32(out, start+8, virtualSize)
	PEPREFIXPut32(out, start+12, rva)
	PEPREFIXPut32(out, start+16, rawSize)
	PEPREFIXPut32(out, start+20, rawAt)
	PEPREFIXPut32(out, start+36, flags)
	return out
}

func PEPREFIXHeader(textRaw int, textSize int, dataRVA int, dataRaw int, dataSize int,
	importRVA int, importSize int, iatRVA int, iatSize int, subsystem int) []byte {
	out := PEPREFIXUntil(nil, PEFILEALIGN)
	PEPREFIXPut16(out, 0, 0x5a4d)
	PEPREFIXPut32(out, 0x3c, 0x80)
	pe := 0x80
	PEPREFIXPut32(out, pe, 0x00004550)
	PEPREFIXPut16(out, pe+4, PEMACHINE)
	PEPREFIXPut16(out, pe+6, 2)
	optionalSize := 0xf0
	if PEPREFIXWordBytes == 4 {
		optionalSize = 0xe0
	}
	PEPREFIXPut16(out, pe+20, optionalSize)
	PEPREFIXPut16(out, pe+22, PECOFFCHAR)
	opt := pe + 24
	if PEPREFIXWordBytes == 8 {
		PEPREFIXPut16(out, opt, 0x20b)
	} else {
		PEPREFIXPut16(out, opt, 0x10b)
	}
	PEPREFIXPut32(out, opt+4, textRaw)
	PEPREFIXPut32(out, opt+8, dataRaw)
	PEPREFIXPut32(out, opt+16, PECODEOFFSET)
	PEPREFIXPut32(out, opt+20, PECODEOFFSET)
	if PEPREFIXWordBytes == 8 {
		PEPREFIXPut32(out, opt+24, PEIMAGEBASE)
		PEPREFIXPut32(out, opt+28, PEIMAGEBASEHIGH)
	} else {
		PEPREFIXPut32(out, opt+24, dataRVA)
		PEPREFIXPut32(out, opt+28, PEIMAGEBASE)
	}
	PEPREFIXPut32(out, opt+32, PESECTIONALIGN)
	PEPREFIXPut32(out, opt+36, PEFILEALIGN)
	PEPREFIXPut16(out, opt+40, PEOSMAJOR)
	PEPREFIXPut16(out, opt+42, PEOSMINOR)
	PEPREFIXPut16(out, opt+44, PEIMAGEMAJOR)
	PEPREFIXPut16(out, opt+46, PEIMAGEMINOR)
	PEPREFIXPut16(out, opt+48, PESUBMAJOR)
	PEPREFIXPut16(out, opt+50, PESUBMINOR)
	PEPREFIXPut32(out, opt+56, PEALIGN(dataRVA+dataSize, PESECTIONALIGN))
	PEPREFIXPut32(out, opt+60, PEFILEALIGN)
	PEPREFIXPut16(out, opt+68, subsystem)
	PEPREFIXPut16(out, opt+70, PEDLLCHAR)
	if PEPREFIXWordBytes == 8 {
		PEPREFIXPutWord(out, opt+72, PESTACKRESERVE)
		PEPREFIXPutWord(out, opt+80, PESTACKCOMMIT)
		PEPREFIXPutWord(out, opt+88, PEHEAPRESERVE)
		PEPREFIXPutWord(out, opt+96, PEHEAPCOMMIT)
		PEPREFIXPut32(out, opt+108, 16)
		PEPREFIXPut32(out, opt+120, importRVA)
		PEPREFIXPut32(out, opt+124, importSize)
		PEPREFIXPut32(out, opt+208, iatRVA)
		PEPREFIXPut32(out, opt+212, iatSize)
		out = out[:opt+240]
	} else {
		PEPREFIXPut32(out, opt+72, PESTACKRESERVE)
		PEPREFIXPut32(out, opt+76, PESTACKCOMMIT)
		PEPREFIXPut32(out, opt+80, PEHEAPRESERVE)
		PEPREFIXPut32(out, opt+84, PEHEAPCOMMIT)
		PEPREFIXPut32(out, opt+92, 16)
		PEPREFIXPut32(out, opt+104, importRVA)
		PEPREFIXPut32(out, opt+108, importSize)
		PEPREFIXPut32(out, opt+192, iatRVA)
		PEPREFIXPut32(out, opt+196, iatSize)
		out = out[:opt+224]
	}
	out = PEPREFIXSection(out, ".text", textSize, PECODEOFFSET, textRaw, PEFILEALIGN, 0x60000020)
	out = PEPREFIXSection(out, ".data", dataSize, dataRVA, dataRaw, PEFILEALIGN+textRaw, 0xc0000040)
	return PEPREFIXUntil(out, PEFILEALIGN)
}

func PEPREFIXImage(out *PEEMITTER) []byte {
	code := out.Code()
	for len(code)%8 != 0 {
		code = append(code, 0)
	}
	out.SetCode(code)
	textSize := len(code)
	textRaw := PEALIGN(textSize, PEFILEALIGN)
	dataRVA := PEALIGN(PECODEOFFSET+textSize, PESECTIONALIGN)
	data, importRVA, importSize, iatRVA, iatSize, iat := PEPREFIXBuildImports(out, out.Data(), dataRVA)
	out.SetData(data)
	PEPATCH(out, dataRVA, len(data), iat)
	dataRaw := PEALIGN(len(data), PEFILEALIGN)
	dataSize := len(data) + out.BSSSize()
	image := PEPREFIXHeader(textRaw, textSize, dataRVA, dataRaw, dataSize,
		importRVA, importSize, iatRVA, iatSize, out.WindowsSubsystem())
	image = append(image, out.Code()...)
	image = PEPREFIXUntil(image, PEFILEALIGN+textRaw)
	image = append(image, data...)
	return PEPREFIXUntil(image, PEFILEALIGN+textRaw+dataRaw)
}
`
