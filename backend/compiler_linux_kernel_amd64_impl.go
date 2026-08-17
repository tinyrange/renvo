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
	renvoEmitInitializeThreadState(g)
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
	// The definition-owned callback sequence uses RIP-relative LEA, so it
	// remains valid after the relocatable module is loaded.
	renvoAmd64KernelCallbackAddress(a, label)
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

func renvoAsmAddKernelImport(a *renvoAsm, src []byte, nameStart int, nameEnd int) int {
	renvoNonNil(a)
	if nameStart < 0 || nameEnd <= nameStart || nameEnd > len(src) {
		return -1
	}
	var name []byte
	for i := nameStart; i < nameEnd; i++ {
		name = append(name, src[i])
	}
	return renvoAsmAddExternalImportName(a, string(name))
}

func renvoAmd64EmitKernelLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	renvoNonNil(g, fn)
	if wordCount < 0 || wordCount > 6 {
		return false
	}
	a := &g.asm
	importID := renvoAsmAddKernelImport(a, g.prog.src, fn.linkMethodStart, fn.linkMethodEnd)
	if importID < 0 {
		return false
	}
	renvoAmd64EmitKernelStaticCall(a, importID, wordCount)
	return true
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
		size, nameOff, initOff, exitOff, ok := renvoAmd64KernelBTFModuleLayout(renvoKernelBTF)
		if !ok || size <= 0 || nameOff < 0 || initOff < 0 {
			return false
		}
		renvoKernelModuleSize = size
		renvoKernelModuleNameOff = nameOff
		renvoKernelModuleInitOff = initOff
		renvoKernelModuleExitOff = exitOff
	}
	// CONFIG_MODVERSIONS=n kernels do not publish module_layout in
	// Module.symvers. BTF plus vermagic still provide the required layout and
	// compatibility inputs; individual external imports remain validated
	// against Module.symvers while the image is assembled.
	return true
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
