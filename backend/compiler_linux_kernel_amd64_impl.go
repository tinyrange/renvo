package main

func renvoBeginKernelModuleAmd64(g *renvoLinearGen, appIndex int) bool {
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
	// Linux/x86 indirect module entry points must be valid IBT targets.
	renvoAmd64KernelEntryPrologue(a)
	renvoLinearMarkFunc(g, appIndex)
	if !g.meta.panicEnabled {
		renvoAmd64InitRuntimeCheckRegs(g)
	}
	renvoEmitPersistentArenaReady(g)
	if !renvoLinearInitGlobals(g) {
		return false
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(g) {
		return false
	}
	renvoAsmPrimaryImm(a, 0)
	renvoAmd64KernelEntryEpilogue(a)
	if exitIndex >= 0 {
		g.kernelExitLabel = renvoAsmNewLabel(a)
		renvoAsmMarkLabel(a, g.kernelExitLabel)
		renvoAmd64KernelEntryPrologue(a)
		renvoLinearMarkFunc(g, exitIndex)
		if !g.meta.panicEnabled {
			renvoAmd64InitRuntimeCheckRegs(g)
		}
		renvoAsmCallLabel(a, g.funcLabels[exitIndex])
		renvoAmd64KernelEntryEpilogue(a)
	}
	return true
}

func renvoAmd64EmitKernelCallbackArgReverse(g *renvoLinearGen, ep *renvoExprParse, idx int, funcType int) int {
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
	// LEA callback(%rip), %rax. The relative label remains valid after the
	// relocatable module is loaded and avoids exporting an implementation symbol.
	renvoAsmEmitText(a, "\x48\x8d\x05")
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddReloc(a, at, label)
	renvoAsmPushPrimary(a)
	if first {
		after := renvoAsmNewLabel(a)
		renvoAsmJmpLabel(a, after)
		renvoAsmMarkLabel(a, label)
		renvoAmd64KernelEntryPrologue(a)
		if !g.meta.panicEnabled {
			renvoAmd64InitRuntimeCheckRegs(g)
		}
		renvoAsmCallLabel(a, g.funcLabels[fnIndex])
		renvoAmd64KernelEntryEpilogue(a)
		renvoAsmMarkLabel(a, after)
	}
	return 1
}

func renvoAmd64KernelEntryPrologue(a *renvoAsm) {
	renvoNonNil(a)
	// ENDBR64, a conventional frame, all SysV callee-saved registers used by
	// the Renvo runtime, and one alignment word before generated calls.
	renvoAsmEmitText(a, "\xf3\x0f\x1e\xfa\x55\x48\x89\xe5\x53\x41\x54\x41\x55\x41\x56\x41\x57\x48\x83\xec\x08")
}

func renvoAmd64KernelEntryEpilogue(a *renvoAsm) {
	renvoNonNil(a)
	renvoAsmEmitText(a, "\x48\x83\xc4\x08\x41\x5f\x41\x5e\x41\x5d\x41\x5c\x5b\xc9\xc3")
}

func renvoAmd64EmitKernelPrintValue(a *renvoAsm) {
	renvoNonNil(a)
	// _printk("%.*s", length, pointer). Preserve Go print semantics for '%'.
	renvoAsmPushPrimary(a)
	renvoAsmEmitText(a, "\x48\x89\xd6")
	renvoAsmEmit8(a, 0x5a)
	formatOff := len(a.data)
	a.data = append(a.data, '%', '.', '*', 's', 0)
	renvoAsmPrimaryDataAddr(a, formatOff)
	renvoAsmCopyPrimaryToCallWord0(a)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmEmit8(a, 0xe8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, 0, renvoAbsKernelPrintReloc)
}

func renvoAsmAddKernelImport(a *renvoAsm, src []byte, nameStart int, nameEnd int) int {
	renvoNonNil(a)
	if nameStart < 0 || nameEnd <= nameStart || nameEnd > len(src) {
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

func renvoAmd64EmitKernelLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if wordCount < 0 || wordCount > 6 {
		return false
	}
	a := &g.asm
	if wordCount > 0 {
		renvoAsmPopCallWord0(a)
	}
	if wordCount > 1 {
		renvoAsmPopCallWord1(a)
	}
	if wordCount > 2 {
		renvoAsmPopSecondary(a)
	}
	if wordCount > 3 {
		renvoAsmPopTertiary(a)
	}
	if wordCount > 4 {
		renvoAsmEmit16(a, 0x5841)
	}
	if wordCount > 5 {
		renvoAsmEmit16(a, 0x5941)
	}
	importID := renvoAsmAddKernelImport(a, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
	if importID < 0 {
		return false
	}
	renvoAsmEmit8(a, 0xe8)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, importID, renvoAbsKernelImportRelocBase+importID)
	return true
}

func renvoKernelImportName(a *renvoAsm, importID int) string {
	index := importID * 2
	if index < 0 || index+1 >= len(a.kernelImportOffsets) {
		return ""
	}
	start := a.kernelImportOffsets[index]
	end := a.kernelImportOffsets[index+1]
	return string(a.kernelImportNames[start:end])
}

func renvoKernelNameFromOutput(path string) string {
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' || path[i] == '\\' {
			start = i + 1
		}
	}
	end := len(path)
	if end-start > 3 && path[end-3] == '.' && path[end-2] == 'k' && path[end-1] == 'o' {
		end -= 3
	}
	var out []byte
	for i := start; i < end && len(out) < 55; i++ {
		ch := path[i]
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '_' {
			out = append(out, ch)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "renvo"
	}
	return string(out)
}

func renvoKernelReadFile(path string) []byte {
	fd := open(path, O_RDONLY)
	if fd < 0 {
		return nil
	}
	var out []byte
	out = renvoReadAll(fd, out)
	close(fd)
	return out
}

func renvoKernelTrimLine(data []byte) string {
	end := len(data)
	for end > 0 && (data[end-1] == '\n' || data[end-1] == '\r' || data[end-1] == 0) {
		end--
	}
	return string(data[:end])
}

func renvoPrepareKernelMetadata() bool {
	if len(renvoKernelBTF) == 0 || len(renvoKernelSymvers) == 0 || renvoKernelRelease == "" {
		release := renvoKernelReadFile("/proc/sys/kernel/osrelease")
		if len(release) == 0 {
			return false
		}
		renvoKernelRelease = renvoKernelTrimLine(release)
		renvoKernelVersion = renvoKernelTrimLine(renvoKernelReadFile("/proc/version"))
		renvoKernelBTF = renvoKernelReadFile("/sys/kernel/btf/vmlinux")
		symversPath := "/lib/modules/" + renvoKernelRelease + "/build/Module.symvers"
		renvoKernelSymvers = renvoKernelReadFile(symversPath)
	}
	if len(renvoKernelBTF) == 0 || len(renvoKernelSymvers) == 0 {
		return false
	}
	if renvoKernelModuleSize <= 0 {
		size, nameOff, initOff, exitOff, ok := renvoKernelBTFModuleLayout(renvoKernelBTF)
		if !ok || size <= 0 || nameOff < 0 || initOff < 0 {
			return false
		}
		renvoKernelModuleSize = size
		renvoKernelModuleNameOff = nameOff
		renvoKernelModuleInitOff = initOff
		renvoKernelModuleExitOff = exitOff
	}
	_, moduleOK := renvoKernelSymbolCRC("module_layout")
	return moduleOK
}

func renvoKernelGet32(data []byte, at int) int {
	if at < 0 || at+4 > len(data) {
		return 0
	}
	return int(data[at]) | int(data[at+1])<<8 | int(data[at+2])<<16 | int(data[at+3])<<24
}

func renvoKernelBTFString(data []byte, base int, off int, value string) bool {
	pos := base + off
	if off < 0 || pos < 0 || pos+len(value) >= len(data) {
		return false
	}
	for i := 0; i < len(value); i++ {
		if data[pos+i] != value[i] {
			return false
		}
	}
	return data[pos+len(value)] == 0
}

func renvoKernelBTFModuleLayout(data []byte) (int, int, int, int, bool) {
	if len(data) < 24 || data[0] != 0x9f || data[1] != 0xeb {
		return 0, 0, 0, 0, false
	}
	headerLen := renvoKernelGet32(data, 4)
	typeStart := headerLen + renvoKernelGet32(data, 8)
	typeEnd := typeStart + renvoKernelGet32(data, 12)
	stringStart := headerLen + renvoKernelGet32(data, 16)
	if headerLen < 24 || typeStart < headerLen || typeEnd > len(data) || stringStart < headerLen || stringStart >= len(data) {
		return 0, 0, 0, 0, false
	}
	pos := typeStart
	for pos+12 <= typeEnd {
		name := renvoKernelGet32(data, pos)
		info := renvoKernelGet32(data, pos+4)
		sizeType := renvoKernelGet32(data, pos+8)
		kind := (info >> 24) & 31
		vlen := info & 65535
		extra := 0
		if kind == 1 {
			extra = 4
		} else if kind == 3 {
			extra = 12
		} else if kind == 4 || kind == 5 {
			extra = vlen * 12
		} else if kind == 6 {
			extra = vlen * 8
		} else if kind == 13 {
			extra = vlen * 8
		} else if kind == 14 || kind == 17 {
			extra = 4
		} else if kind == 15 || kind == 19 {
			extra = vlen * 12
		}
		next := pos + 12 + extra
		if next > typeEnd || next <= pos {
			return 0, 0, 0, 0, false
		}
		if kind == 4 && renvoKernelBTFString(data, stringStart, name, "module") {
			nameOff := -1
			initOff := -1
			exitOff := -1
			member := pos + 12
			for i := 0; i < vlen; i++ {
				memberName := renvoKernelGet32(data, member)
				bitOff := renvoKernelGet32(data, member+8) & 0x00ffffff
				if bitOff%8 == 0 {
					if renvoKernelBTFString(data, stringStart, memberName, "name") {
						nameOff = bitOff / 8
					} else if renvoKernelBTFString(data, stringStart, memberName, "init") {
						initOff = bitOff / 8
					} else if renvoKernelBTFString(data, stringStart, memberName, "exit") {
						exitOff = bitOff / 8
					}
				}
				member += 12
			}
			return sizeType, nameOff, initOff, exitOff, nameOff >= 0 && initOff >= 0
		}
		pos = next
	}
	return 0, 0, 0, 0, false
}

func renvoKernelHexDigit(ch byte) int {
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

func renvoKernelSymbolCRC(symbol string) (int, bool) {
	data := renvoKernelSymvers
	line := 0
	for line < len(data) {
		end := line
		for end < len(data) && data[end] != '\n' {
			end++
		}
		firstTab := line
		for firstTab < end && data[firstTab] != '\t' {
			firstTab++
		}
		secondTab := firstTab + 1
		for secondTab < end && data[secondTab] != '\t' {
			secondTab++
		}
		if secondTab-firstTab-1 == len(symbol) {
			match := true
			for i := 0; i < len(symbol); i++ {
				if data[firstTab+1+i] != symbol[i] {
					match = false
				}
			}
			if match {
				value := 0
				start := line
				if start+2 <= firstTab && data[start] == '0' && data[start+1] == 'x' {
					start += 2
				}
				for i := start; i < firstTab; i++ {
					digit := renvoKernelHexDigit(data[i])
					if digit < 0 {
						return 0, false
					}
					value = (value << 4) | digit
				}
				return value, true
			}
		}
		line = end + 1
	}
	return 0, false
}

func renvoKernelSymbolGPLOnly(symbol string) bool {
	data := renvoKernelSymvers
	line := 0
	for line < len(data) {
		end := line
		for end < len(data) && data[end] != '\n' {
			end++
		}
		firstTab := line
		for firstTab < end && data[firstTab] != '\t' {
			firstTab++
		}
		secondTab := firstTab + 1
		for secondTab < end && data[secondTab] != '\t' {
			secondTab++
		}
		if secondTab-firstTab-1 == len(symbol) {
			match := true
			for i := 0; i < len(symbol); i++ {
				if data[firstTab+1+i] != symbol[i] {
					match = false
				}
			}
			if match {
				thirdTab := secondTab + 1
				for thirdTab < end && data[thirdTab] != '\t' {
					thirdTab++
				}
				exportStart := thirdTab + 1
				exportEnd := exportStart
				for exportEnd < end && data[exportEnd] != '\t' {
					exportEnd++
				}
				return exportEnd-exportStart >= 4 && data[exportEnd-4] == '_' && data[exportEnd-3] == 'G' && data[exportEnd-2] == 'P' && data[exportEnd-1] == 'L'
			}
		}
		line = end + 1
	}
	return false
}

func renvoKernelLicenseGPLCompatible() bool {
	license := renvoKernelLicense
	return license == "GPL" || license == "GPL v2" || license == "GPL and additional rights" || license == "Dual BSD/GPL" || license == "Dual MIT/GPL" || license == "Dual MPL/GPL"
}

func renvoKernelContains(text string, needle string) bool {
	if len(needle) == 0 || len(needle) > len(text) {
		return false
	}
	for i := 0; i+len(needle) <= len(text); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if text[i+j] != needle[j] {
				match = false
			}
		}
		if match {
			return true
		}
	}
	return false
}

func renvoKernelVermagic() string {
	out := renvoKernelRelease
	if renvoKernelContains(renvoKernelVersion, " SMP ") || renvoKernelContains(renvoKernelVersion, " SMP") {
		out += " SMP"
	}
	if renvoKernelContains(renvoKernelVersion, "PREEMPT") {
		out += " preempt"
	}
	if renvoKernelModuleExitOff >= 0 {
		out += " mod_unload"
	}
	if crc, ok := renvoKernelSymbolCRC("module_layout"); ok && crc != 0 {
		out += " modversions"
	}
	return out + " "
}

func renvoKernelAppendString(out []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return append(out, 0)
}

func renvoKernelAppendSym64(out []byte, name int, info int, section int, value int, size int) []byte {
	out = renvoAppend32(out, name)
	out = append(out, byte(info), 0)
	out = renvoAppend16(out, section)
	out = renvoAppend64(out, value)
	out = renvoAppend64(out, size)
	return out
}

func renvoKernelAppendRela(out []byte, offset int, symbol int, kind int, addend int) []byte {
	out = renvoAppend64(out, offset)
	out = renvoAppend64(out, (symbol<<32)|kind)
	out = renvoAppend64(out, addend)
	return out
}

func renvoKernelAppendVersion(out []byte, symbol string, crc int) []byte {
	start := len(out)
	out = renvoAppend64(out, crc)
	for i := 0; i < len(symbol) && i < 55; i++ {
		out = append(out, symbol[i])
	}
	out = append(out, 0)
	return renvoAppendUntil(out, start+64)
}

func renvoKernelAppendELFHeader(out []byte, shoff int, shnum int, shstrndx int) []byte {
	out = append(out, 0x7f, 'E', 'L', 'F', 2, 1, 1, 0)
	out = renvoAppendUntil(out, 16)
	out = renvoAppend16(out, 1)
	out = renvoAppend16(out, 62)
	out = renvoAppend32(out, 1)
	out = renvoAppend64(out, 0)
	out = renvoAppend64(out, 0)
	out = renvoAppend64(out, shoff)
	out = renvoAppend32(out, 0)
	out = renvoAppend16(out, 64)
	out = renvoAppend16(out, 0)
	out = renvoAppend16(out, 0)
	out = renvoAppend16(out, 64)
	out = renvoAppend16(out, shnum)
	out = renvoAppend16(out, shstrndx)
	return out
}

func renvoKernelAppendShdr64(out []byte, name int, kind int, flags int, off int, size int, link int, info int, align int, entsize int) []byte {
	out = renvoAppend32(out, name)
	out = renvoAppend32(out, kind)
	out = renvoAppend64(out, flags)
	out = renvoAppend64(out, 0)
	out = renvoAppend64(out, off)
	out = renvoAppend64(out, size)
	out = renvoAppend32(out, link)
	out = renvoAppend32(out, info)
	out = renvoAppend64(out, align)
	out = renvoAppend64(out, entsize)
	return out
}

func renvoAsmImageKernelModuleAmd64(a *renvoAsm, initLabel int, exitLabel int) []byte {
	renvoNonNil(a)
	renvoAsmPatch(a)
	initPos := renvoAsmLabelPosition(a, initLabel)
	exitPos := renvoAsmLabelPosition(a, exitLabel)
	if initPos < 0 || a.c.kernel.kernelModuleSize <= 0 {
		return nil
	}
	hasPrint := false
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		if int(a.absRelocs[i+2]) == renvoAbsKernelPrintReloc {
			hasPrint = true
		}
	}

	var strtab []byte
	strtab = append(strtab, 0)
	initName := len(strtab)
	strtab = renvoKernelAppendString(strtab, "init_module")
	exitName := len(strtab)
	strtab = renvoKernelAppendString(strtab, "cleanup_module")
	thisName := len(strtab)
	strtab = renvoKernelAppendString(strtab, "__this_module")
	printName := len(strtab)
	strtab = renvoKernelAppendString(strtab, "_printk")
	var importNames []int
	for i := 0; i+1 < len(a.kernelImportOffsets); i += 2 {
		importNames = append(importNames, len(strtab))
		strtab = renvoKernelAppendString(strtab, renvoKernelImportName(a, i/2))
	}

	var symtab []byte
	symtab = renvoKernelAppendSym64(symtab, 0, 0, 0, 0, 0)
	symtab = renvoKernelAppendSym64(symtab, 0, 3, 1, 0, 0)
	symtab = renvoKernelAppendSym64(symtab, 0, 3, 3, 0, 0)
	symtab = renvoKernelAppendSym64(symtab, 0, 3, 4, 0, 0)
	symtab = renvoKernelAppendSym64(symtab, 0, 3, 7, 0, 0)
	initSym := 5
	symtab = renvoKernelAppendSym64(symtab, initName, 18, 1, initPos, 0)
	exitSym := 0
	if exitPos >= 0 {
		exitSym = len(symtab) / 24
		symtab = renvoKernelAppendSym64(symtab, exitName, 18, 1, exitPos, 0)
	}
	symtab = renvoKernelAppendSym64(symtab, thisName, 17, 7, 0, a.c.kernel.kernelModuleSize)
	printSym := 0
	if hasPrint {
		printSym = len(symtab) / 24
		symtab = renvoKernelAppendSym64(symtab, printName, 16, 0, 0, 0)
	}
	var importSymbols []int
	for i := 0; i < len(importNames); i++ {
		importSymbols = append(importSymbols, len(symtab)/24)
		symtab = renvoKernelAppendSym64(symtab, importNames[i], 16, 0, 0, 0)
	}

	var relaText []byte
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(a.absRelocs[i]) & 2147483647
		off := int(a.absRelocs[i+1]) & 2147483647
		kind := int(a.absRelocs[i+2]) & 2147483647
		if kind == renvoAbsBssReloc {
			relaText = renvoKernelAppendRela(relaText, at, 3, 2, off-4)
		} else if kind == renvoAbsKernelPrintReloc {
			relaText = renvoKernelAppendRela(relaText, at, printSym, 4, -4)
		} else if kind >= renvoAbsKernelImportRelocBase {
			importID := kind - renvoAbsKernelImportRelocBase
			if importID < 0 || importID >= len(importSymbols) {
				return nil
			}
			relaText = renvoKernelAppendRela(relaText, at, importSymbols[importID], 4, -4)
		} else {
			relaText = renvoKernelAppendRela(relaText, at, 2, 2, off-4)
		}
	}

	thisModule := make([]byte, a.c.kernel.kernelModuleSize)
	if a.c.kernel.kernelNameOff+len(a.c.kernel.kernelModuleName) >= len(thisModule) {
		return nil
	}
	for i := 0; i < len(a.c.kernel.kernelModuleName); i++ {
		thisModule[a.c.kernel.kernelNameOff+i] = a.c.kernel.kernelModuleName[i]
	}
	var relaThis []byte
	relaThis = renvoKernelAppendRela(relaThis, a.c.kernel.kernelInitOff, initSym, 1, 0)
	if exitSym != 0 && a.c.kernel.kernelExitOff >= 0 {
		relaThis = renvoKernelAppendRela(relaThis, a.c.kernel.kernelExitOff, exitSym, 1, 0)
	}

	var versions []byte
	moduleCRC, moduleOK := renvoKernelSymbolCRC("module_layout")
	if !moduleOK {
		return nil
	}
	versions = renvoKernelAppendVersion(versions, "module_layout", moduleCRC)
	if hasPrint {
		printCRC, ok := renvoKernelSymbolCRC("_printk")
		if !ok {
			return nil
		}
		versions = renvoKernelAppendVersion(versions, "_printk", printCRC)
	}
	for i := 0; i < len(importSymbols); i++ {
		name := renvoKernelImportName(a, i)
		if renvoKernelSymbolGPLOnly(name) && !renvoKernelLicenseGPLCompatible() {
			renvoPrintErr("renvo: GPL-only kernel symbol requires a GPL-compatible module license: ")
			renvoPrintErr(name)
			renvoPrintErr("\n")
			return nil
		}
		crc, ok := renvoKernelSymbolCRC(name)
		if !ok {
			return nil
		}
		versions = renvoKernelAppendVersion(versions, name, crc)
	}

	var modinfo []byte
	modinfo = renvoKernelAppendString(modinfo, "license="+a.c.kernel.kernelLicense)
	modinfo = renvoKernelAppendString(modinfo, "depends=")
	modinfo = renvoKernelAppendString(modinfo, "name="+a.c.kernel.kernelModuleName)
	modinfo = renvoKernelAppendString(modinfo, "vermagic="+renvoKernelVermagic())

	var shstr []byte
	shstr = append(shstr, 0)
	textName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".text")
	relaTextName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".rela.text")
	rodataName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".rodata")
	bssName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".bss")
	modinfoName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".modinfo")
	versionsName := len(shstr)
	shstr = renvoKernelAppendString(shstr, "__versions")
	thisSectionName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".gnu.linkonce.this_module")
	relaThisName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".rela.gnu.linkonce.this_module")
	symtabName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".symtab")
	strtabName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".strtab")
	shstrName := len(shstr)
	shstr = renvoKernelAppendString(shstr, ".shstrtab")

	var out []byte
	out = renvoAppendUntil(out, 64)
	textOff := renvoAlignValue(len(out), 16)
	out = renvoAppendUntil(out, textOff)
	out = append(out, a.code...)
	relaTextOff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, relaTextOff)
	out = append(out, relaText...)
	rodataOff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, rodataOff)
	out = append(out, a.data...)
	bssOff := len(out)
	modinfoOff := len(out)
	out = append(out, modinfo...)
	versionsOff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, versionsOff)
	out = append(out, versions...)
	thisOff := renvoAlignValue(len(out), 64)
	out = renvoAppendUntil(out, thisOff)
	out = append(out, thisModule...)
	relaThisOff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, relaThisOff)
	out = append(out, relaThis...)
	symtabOff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, symtabOff)
	out = append(out, symtab...)
	strtabOff := len(out)
	out = append(out, strtab...)
	shstrOff := len(out)
	out = append(out, shstr...)
	shoff := renvoAlignValue(len(out), 8)
	out = renvoAppendUntil(out, shoff)
	out = renvoKernelAppendShdr64(out, 0, 0, 0, 0, 0, 0, 0, 0, 0)
	out = renvoKernelAppendShdr64(out, textName, 1, 6, textOff, len(a.code), 0, 0, 16, 0)
	out = renvoKernelAppendShdr64(out, relaTextName, 4, 64, relaTextOff, len(relaText), 9, 1, 8, 24)
	out = renvoKernelAppendShdr64(out, rodataName, 1, 2, rodataOff, len(a.data), 0, 0, 8, 0)
	out = renvoKernelAppendShdr64(out, bssName, 8, 3, bssOff, a.bssSize, 0, 0, 8, 0)
	out = renvoKernelAppendShdr64(out, modinfoName, 1, 2, modinfoOff, len(modinfo), 0, 0, 1, 0)
	out = renvoKernelAppendShdr64(out, versionsName, 1, 2, versionsOff, len(versions), 0, 0, 8, 0)
	out = renvoKernelAppendShdr64(out, thisSectionName, 1, 3, thisOff, len(thisModule), 0, 0, 64, 0)
	out = renvoKernelAppendShdr64(out, relaThisName, 4, 64, relaThisOff, len(relaThis), 9, 7, 8, 24)
	out = renvoKernelAppendShdr64(out, symtabName, 2, 0, symtabOff, len(symtab), 10, 5, 8, 24)
	out = renvoKernelAppendShdr64(out, strtabName, 3, 0, strtabOff, len(strtab), 0, 0, 1, 0)
	out = renvoKernelAppendShdr64(out, shstrName, 3, 0, shstrOff, len(shstr), 0, 0, 1, 0)
	var header []byte
	header = renvoKernelAppendELFHeader(header, shoff, 12, 11)
	for i := 0; i < len(header); i++ {
		out[i] = header[i]
	}
	return out
}
