package main

// renvo:linkstatic /usr/lib/system/libcommonCrypto.dylib,CC_SHA256
func renvoDarwinCCSHA256(data []byte, length int, digest []byte) int { return 0 }

func renvo_runtime_ArenaPersistString(value string) string { return value }

const renvoDarwinArm64CodeOffset = 0x1000
const renvoDarwinArm64ImageBase = 0x100000000
const renvoDarwinArm64PageSize = 0x4000
const renvoDarwinArgEnvDescriptorSize = 32768
const renvoDarwinArgEnvDescriptorCount = renvoDarwinArgEnvDescriptorSize / 16

const renvoDarwinImportExit = 1
const renvoDarwinImportOpen = 2
const renvoDarwinImportClose = 3
const renvoDarwinImportRead = 4
const renvoDarwinImportWrite = 5
const renvoDarwinImportPread = 6
const renvoDarwinImportPwrite = 7
const renvoDarwinImportFchmod = 8
const renvoDarwinImportGetdirentries = 9
const renvoDarwinImportCount = 9

func compileDarwinArm64(input []int, output int) int {
	return compileDarwinArm64Arena(input, output, 0)
}

func compileDarwinArm64Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetDarwinArm64)
	return renvoCompileAarch64(input, output, arenaSize)
}

func renvoDarwinImportName(id int) string {
	if id == renvoDarwinImportExit {
		return "_exit"
	}
	if id == renvoDarwinImportOpen {
		return "_open"
	}
	if id == renvoDarwinImportClose {
		return "_close"
	}
	if id == renvoDarwinImportRead {
		return "_read"
	}
	if id == renvoDarwinImportWrite {
		return "_write"
	}
	if id == renvoDarwinImportPread {
		return "_pread"
	}
	if id == renvoDarwinImportPwrite {
		return "_pwrite"
	}
	if id == renvoDarwinImportFchmod {
		return "_fchmod"
	}
	if id == renvoDarwinImportGetdirentries {
		return "_getdirentries"
	}
	return ""
}

func renvoDarwinArm64ImportLabel(a *renvoAsm, id int) int {
	if len(a.darwinImportLabels) == 0 {
		a.darwinImportLabels = make([]int, renvoDarwinImportCount+1)
		a.darwinImportUsed = make([]bool, renvoDarwinImportCount+1)
		for i := 0; i <= renvoDarwinImportCount; i++ {
			a.darwinImportLabels[i] = -1
		}
	}
	if id <= 0 || id > renvoDarwinImportCount {
		return -1
	}
	if a.darwinImportLabels[id] < 0 {
		a.darwinImportLabels[id] = renvoAsmNewLabel(a)
	}
	a.darwinImportUsed[id] = true
	return a.darwinImportLabels[id]
}

func renvoDarwinArm64CallImport(a *renvoAsm, id int) {
	label := renvoDarwinArm64ImportLabel(a, id)
	renvoAarch64AsmCallLabel(a, label)
}

func renvoDarwinArm64CallVirtualArgs(a *renvoAsm, id int, argCount int) {
	if argCount > 2 {
		renvoAarch64AsmMovRegReg(a, renvoAarch64RegTmp, renvoAarch64RegRdx)
	}
	if argCount > 0 {
		renvoAarch64AsmMovRegReg(a, 0, renvoAarch64RegRdi)
	}
	if argCount > 1 {
		renvoAarch64AsmMovRegReg(a, 1, renvoAarch64RegRsi)
	}
	if argCount > 2 {
		renvoAarch64AsmMovRegReg(a, 2, renvoAarch64RegTmp)
	}
	if argCount > 3 {
		renvoAarch64AsmMovRegReg(a, 3, renvoAarch64RegR10)
	}
	renvoDarwinArm64CallImport(a, id)
}

func renvoAsmAddDarwinStaticImport(a *renvoAsm, dylib string, name string) int {
	if len(name) == 0 || name[0] != '_' {
		name = renvo_runtime_ArenaPersistString("_" + name)
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if a.darwinImports[i].dylib == dylib && a.darwinImports[i].name == name {
			a.darwinImports[i].used = true
			return i
		}
	}
	label := renvoAsmNewLabel(a)
	a.darwinImports = append(a.darwinImports, renvoDarwinStaticImport{dylib: dylib, name: name, label: label, used: true})
	return len(a.darwinImports) - 1
}

func renvoDarwinArm64EmitLinkStaticCall(g *renvoLinearGen, fn *renvoFuncInfo, wordCount int) bool {
	if renvoTargetArch != renvoArchAarch64 {
		return false
	}
	dylib := renvo_runtime_ArenaPersistString(renvoStringFromBytes(g.prog.src, fn.linkDLLStart, fn.linkDLLEnd))
	name := renvo_runtime_ArenaPersistString(renvoStringFromBytes(g.prog.src, fn.linkMethodStart, fn.linkMethodEnd))
	importIndex := renvoAsmAddDarwinStaticImport(&g.asm, dylib, name)
	a := &g.asm
	intReg := 0
	floatReg := 0
	consumed := 0
	objcRectCall := renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgRect")
	objcSizeCall := renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgSize")
	glOrthoCall := name == "glOrtho" || name == "_glOrtho"
	glPixelZoomCall := name == "glPixelZoom" || name == "_glPixelZoom"
	for i := 0; i < fn.paramCount; i++ {
		typ := renvoResolveType(g.meta, g.meta.params[fn.firstParam+i].typ)
		integerDouble := glOrthoCall || (objcRectCall && i >= 2 && i < 6) || (objcSizeCall && i >= 2 && i < 4)
		integerSingle := glPixelZoomCall
		if typ.kind == renvoTypeFloat64 || integerDouble || integerSingle {
			if floatReg >= 8 {
				return false
			}
			renvoAarch64AsmPopReg(a, 16)
			if integerSingle {
				renvoAarch64AsmEmit(a, 0x9e220000|(16<<5)|floatReg)
			} else {
				renvoAarch64AsmEmit(a, 0x9e620000|(16<<5)|floatReg)
			}
			if typ.kind == renvoTypeFloat64 && !integerDouble {
				// RENVO represents float64 values as integers scaled by four. Convert
				// them to an ABI double only at the foreign-call boundary.
				renvoAarch64AsmMovRegImm(a, 16, 4)
				renvoAarch64AsmEmit(a, 0x9e620000|(16<<5)|31)
				renvoAarch64AsmEmit(a, 0x1e601800|(31<<16)|(floatReg<<5)|floatReg)
			}
			floatReg++
			consumed++
			continue
		}
		if intReg >= 8 {
			return false
		}
		if typ.kind == renvoTypeString {
			renvoAarch64AsmPopReg(a, intReg)
			renvoAarch64AsmPopReg(a, 16)
			consumed += 2
			intReg++
			continue
		}
		if typ.kind == renvoTypeSlice {
			renvoAarch64AsmPopReg(a, intReg)
			renvoAarch64AsmPopReg(a, 16)
			renvoAarch64AsmPopReg(a, 16)
			consumed += 3
			intReg++
			continue
		}
		if typ.kind == renvoTypeStruct || typ.kind == renvoTypeArray {
			return false
		}
		renvoAarch64AsmPopReg(a, intReg)
		consumed++
		intReg++
	}
	if consumed != wordCount {
		renvoPrintErr("renvo: Darwin foreign-call argument layout mismatch\n")
		return false
	}
	for intReg < 8 {
		renvoAarch64AsmMovRegImm(a, intReg, 0)
		intReg++
	}
	renvoAsmCallLabel(a, a.darwinImports[importIndex].label)
	if renvoResolveType(g.meta, fn.resultType).kind == renvoTypeFloat64 {
		resultFloatReg := 0
		if renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgPointY") {
			resultFloatReg = 1
		}
		if renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgRectY") {
			resultFloatReg = 1
		}
		if renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgRectWidth") {
			resultFloatReg = 2
		}
		if renvoBytesEqualText(g.prog.src, fn.nameStart, fn.nameEnd, "objcMsgRectHeight") {
			resultFloatReg = 3
		}
		// Convert an ABI double result back to RENVO's scaled representation.
		renvoAarch64AsmMovRegImm(a, 16, 4)
		renvoAarch64AsmEmit(a, 0x9e620000|(16<<5)|31)
		renvoAarch64AsmEmit(a, 0x1e600800|(31<<16)|(resultFloatReg<<5)|resultFloatReg)
		renvoAarch64AsmEmit(a, 0x9e780000|(resultFloatReg<<5))
	}
	return true
}

func renvoEmitProgramEntryArgsDarwinArm64(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !renvoTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 2 {
		second := &g.meta.params[app.firstParam+1]
		if !renvoTypeIsStringSlice(g.meta, second.typ) {
			return false
		}
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += renvoDarwinArgEnvDescriptorSize
	envOff := g.asm.bssSize
	g.asm.bssSize += renvoDarwinArgEnvDescriptorSize
	renvoAsmBuildDarwinArgvEnvSlicesArm64(&g.asm, g.darwinEntryOff, argsOff, envOff)
	return true
}

func renvoAsmBuildDarwinArgvEnvSlicesArm64(a *renvoAsm, entryOff int, argsOff int, envOff int) {
	argLoop := renvoAsmNewLabel(a)
	argLenLoop := renvoAsmNewLabel(a)
	argLenDone := renvoAsmNewLabel(a)
	argsDone := renvoAsmNewLabel(a)
	envLoop := renvoAsmNewLabel(a)
	envLenLoop := renvoAsmNewLabel(a)
	envLenDone := renvoAsmNewLabel(a)
	envDone := renvoAsmNewLabel(a)

	// dyld calls LC_MAIN with argc, argv, envp in x0, x1, x2. Global
	// initialization runs first, so reload the entry values saved at startup.
	renvoAarch64AsmMovRegAbs(a, 9, entryOff, renvoAbsBssReloc)
	renvoAarch64AsmLoadRegMem(a, 13, 9, 0, 8)
	renvoAarch64AsmLoadRegMem(a, 14, 9, 8, 8)
	renvoAarch64AsmLoadRegMem(a, 15, 9, 16, 8)
	renvoAarch64AsmMovRegAbs(a, 10, argsOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 11, 0)
	renvoAsmMarkLabel(a, argLoop)
	renvoAarch64AsmCmpRegImm(a, 11, renvoDarwinArgEnvDescriptorCount)
	renvoAarch64AsmBCondLabel(a, argsDone, 10)
	renvoAarch64AsmCmpRegReg(a, 11, 13)
	renvoAarch64AsmBCondLabel(a, argsDone, 0)
	renvoAarch64AsmAddRegRegShift(a, 12, 14, 11, 3)
	renvoAarch64AsmLoadRegMem(a, 12, 12, 0, 8)
	renvoAarch64AsmStoreRegMem(a, 12, 10, 0, 8)
	renvoAarch64AsmMovRegImm(a, 0, 0)
	renvoAsmMarkLabel(a, argLenLoop)
	renvoAarch64AsmAddRegReg(a, 9, 12, 0)
	renvoAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 9, 0)
	renvoAarch64AsmBCondLabel(a, argLenDone, 0)
	renvoAarch64AsmAddRegImm(a, 0, 0, 1)
	renvoAarch64AsmJmpLabel(a, argLenLoop)
	renvoAsmMarkLabel(a, argLenDone)
	renvoAarch64AsmStoreRegMem(a, 0, 10, 8, 8)
	renvoAarch64AsmAddRegImm(a, 10, 10, 16)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmJmpLabel(a, argLoop)
	renvoAsmMarkLabel(a, argsDone)
	renvoAarch64AsmMovRegReg(a, 13, 11)

	renvoAarch64AsmMovRegAbs(a, 10, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegImm(a, 11, 0)
	renvoAsmMarkLabel(a, envLoop)
	renvoAarch64AsmCmpRegImm(a, 11, renvoDarwinArgEnvDescriptorCount)
	renvoAarch64AsmBCondLabel(a, envDone, 10)
	renvoAarch64AsmLoadRegMem(a, 12, 15, 0, 8)
	renvoAarch64AsmCmpRegImm(a, 12, 0)
	renvoAarch64AsmBCondLabel(a, envDone, 0)
	renvoAarch64AsmStoreRegMem(a, 12, 10, 0, 8)
	renvoAarch64AsmMovRegImm(a, 0, 0)
	renvoAsmMarkLabel(a, envLenLoop)
	renvoAarch64AsmAddRegReg(a, 9, 12, 0)
	renvoAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	renvoAarch64AsmCmpRegImm(a, 9, 0)
	renvoAarch64AsmBCondLabel(a, envLenDone, 0)
	renvoAarch64AsmAddRegImm(a, 0, 0, 1)
	renvoAarch64AsmJmpLabel(a, envLenLoop)
	renvoAsmMarkLabel(a, envLenDone)
	renvoAarch64AsmStoreRegMem(a, 0, 10, 8, 8)
	renvoAarch64AsmAddRegImm(a, 10, 10, 16)
	renvoAarch64AsmAddRegImm(a, 15, 15, 8)
	renvoAarch64AsmAddRegImm(a, 11, 11, 1)
	renvoAarch64AsmJmpLabel(a, envLoop)
	renvoAsmMarkLabel(a, envDone)

	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRdi, argsOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRsi, 13)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, 13)
	renvoAarch64AsmMovRegAbs(a, renvoAarch64RegRcx, envOff, renvoAbsBssReloc)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR8, 11)
	renvoAarch64AsmMovRegReg(a, renvoAarch64RegR9, 11)
}

func renvoDarwinAppendULEB(out []byte, value int) []byte {
	for {
		b := byte(value & 127)
		value = value >> 7
		if value != 0 {
			b = b | 128
		}
		out = append(out, b)
		if value == 0 {
			return out
		}
	}
}

func renvoDarwinAppendName16(out []byte, name string) []byte {
	for i := 0; i < 16; i++ {
		if i < len(name) {
			out = append(out, name[i])
		} else {
			out = append(out, 0)
		}
	}
	return out
}

func renvoDarwinAppendSegment64(out []byte, name string, vmaddr int, vmsize int, fileoff int, filesize int, maxprot int, initprot int) []byte {
	out = renvoAppend32(out, 0x19)
	out = renvoAppend32(out, 72)
	out = renvoDarwinAppendName16(out, name)
	out = renvoAppend64(out, vmaddr)
	out = renvoAppend64(out, vmsize)
	out = renvoAppend64(out, fileoff)
	out = renvoAppend64(out, filesize)
	out = renvoAppend32(out, maxprot)
	out = renvoAppend32(out, initprot)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	return out
}

func renvoDarwinAppendSection64(out []byte, section string, segment string, addr int, size int, offset int, align int, flags int) []byte {
	out = renvoDarwinAppendName16(out, section)
	out = renvoDarwinAppendName16(out, segment)
	out = renvoAppend64(out, addr)
	out = renvoAppend64(out, size)
	out = renvoAppend32(out, offset)
	out = renvoAppend32(out, align)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, flags)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	return out
}

func renvoDarwinAppendTextSegment64(out []byte, codeSize int, fileSize int) []byte {
	out = renvoAppend32(out, 0x19)
	out = renvoAppend32(out, 152)
	out = renvoDarwinAppendName16(out, "__TEXT")
	out = renvoAppend64(out, renvoDarwinArm64ImageBase)
	out = renvoAppend64(out, fileSize)
	out = renvoAppend64(out, 0)
	out = renvoAppend64(out, fileSize)
	out = renvoAppend32(out, 5)
	out = renvoAppend32(out, 5)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 0)
	out = renvoDarwinAppendSection64(out, "__text", "__TEXT", renvoDarwinArm64ImageBase+renvoDarwinArm64CodeOffset, codeSize, renvoDarwinArm64CodeOffset, 2, 0x80000400)
	return out
}

func renvoDarwinAppendDataSegment64(out []byte, dataFileOff int, dataFileSize int, bssSize int) []byte {
	vmSize := renvoAlignValue(dataFileSize+bssSize, renvoDarwinArm64PageSize)
	out = renvoAppend32(out, 0x19)
	out = renvoAppend32(out, 232)
	out = renvoDarwinAppendName16(out, "__DATA")
	out = renvoAppend64(out, renvoDarwinArm64ImageBase+dataFileOff)
	out = renvoAppend64(out, vmSize)
	out = renvoAppend64(out, dataFileOff)
	out = renvoAppend64(out, dataFileSize)
	out = renvoAppend32(out, 3)
	out = renvoAppend32(out, 3)
	out = renvoAppend32(out, 2)
	out = renvoAppend32(out, 0)
	out = renvoDarwinAppendSection64(out, "__data", "__DATA", renvoDarwinArm64ImageBase+dataFileOff, dataFileSize, dataFileOff, 3, 0)
	out = renvoDarwinAppendSection64(out, "__bss", "__DATA", renvoDarwinArm64ImageBase+dataFileOff+dataFileSize, bssSize, 0, 3, 1)
	return out
}

func renvoDarwinPatchArm64Adrp(code []byte, at int, pc int, target int) {
	delta := (target >> 12) - (pc >> 12)
	imm := delta & 0x1fffff
	insn := 0x90000010 | ((imm & 3) << 29) | (((imm >> 2) & 0x7ffff) << 5)
	renvoPut32At(code, at, insn)
}

func renvoDarwinStaticDylibs(a *renvoAsm) []string {
	var out []string
	for i := 0; i < len(a.darwinImports); i++ {
		imp := a.darwinImports[i]
		if !imp.used || imp.dylib == "/usr/lib/libSystem.B.dylib" {
			continue
		}
		found := false
		for j := 0; j < len(out); j++ {
			if out[j] == imp.dylib {
				found = true
			}
		}
		if !found {
			out = append(out, imp.dylib)
		}
	}
	return out
}

func renvoDarwinDylibOrdinal(a *renvoAsm, dylib string) int {
	if dylib == "/usr/lib/libSystem.B.dylib" {
		return 1
	}
	dylibs := renvoDarwinStaticDylibs(a)
	for i := 0; i < len(dylibs); i++ {
		if dylibs[i] == dylib {
			return i + 2
		}
	}
	return 0
}

func renvoDarwinDylibCommandSize(path string) int {
	return renvoAlignValue(24+len(path)+1, 8)
}

func renvoDarwinAppendDylibCommand(out []byte, path string) []byte {
	start := len(out)
	size := renvoDarwinDylibCommandSize(path)
	out = renvoAppend32(out, 0x0c)
	out = renvoAppend32(out, size)
	out = renvoAppend32(out, 24)
	out = renvoAppend32(out, 2)
	out = renvoAppend32(out, 0x00010000)
	out = renvoAppend32(out, 0x00010000)
	out = renvoAppendStringZ(out, path)
	return renvoAppendUntil(out, start+size)
}

func renvoDarwinMachHeader(a *renvoAsm, textFileSize int, dataFileOff int, dataFileSize int, dataVMSize int, linkeditOff int, linkeditSize int, bindOff int, bindSize int, symOff int, symbolCount int, strOff int, strSize int, undefinedCount int, sigOff int, sigSize int) []byte {
	dylibs := renvoDarwinStaticDylibs(a)
	commandSize := 856
	for i := 0; i < len(dylibs); i++ {
		commandSize += renvoDarwinDylibCommandSize(dylibs[i])
	}
	out := make([]byte, 0, a.codeOffset)
	out = renvoAppend32(out, 0xfeedfacf)
	out = renvoAppend32(out, 0x0100000c)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 2)
	out = renvoAppend32(out, 13+len(dylibs))
	out = renvoAppend32(out, commandSize)
	out = renvoAppend32(out, 0x200085)
	out = renvoAppend32(out, 0)
	out = renvoDarwinAppendSegment64(out, "__PAGEZERO", 0, renvoDarwinArm64ImageBase, 0, 0, 0, 0)
	out = renvoDarwinAppendTextSegment64(out, len(a.code), textFileSize)
	out = renvoDarwinAppendDataSegment64(out, dataFileOff, dataFileSize, a.bssSize)
	out = renvoDarwinAppendSegment64(out, "__LINKEDIT", renvoDarwinArm64ImageBase+dataFileOff+dataVMSize, renvoAlignValue(linkeditSize, renvoDarwinArm64PageSize), linkeditOff, linkeditSize, 1, 1)
	out = renvoAppend32(out, 0x80000022)
	out = renvoAppend32(out, 48)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, bindOff)
	out = renvoAppend32(out, bindSize)
	for i := 0; i < 6; i++ {
		out = renvoAppend32(out, 0)
	}
	out = renvoAppend32(out, 0x02)
	out = renvoAppend32(out, 24)
	out = renvoAppend32(out, symOff)
	out = renvoAppend32(out, symbolCount)
	out = renvoAppend32(out, strOff)
	out = renvoAppend32(out, strSize)
	out = renvoAppend32(out, 0x0b)
	out = renvoAppend32(out, 80)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, undefinedCount)
	for i := 0; i < 12; i++ {
		out = renvoAppend32(out, 0)
	}
	cmdStart := len(out)
	out = renvoAppend32(out, 0x0e)
	out = renvoAppend32(out, 32)
	out = renvoAppend32(out, 12)
	out = renvoAppendStringZ(out, "/usr/lib/dyld")
	out = renvoAppendUntil(out, cmdStart+32)
	out = renvoDarwinAppendDylibCommand(out, "/usr/lib/libSystem.B.dylib")
	for i := 0; i < len(dylibs); i++ {
		out = renvoDarwinAppendDylibCommand(out, dylibs[i])
	}
	out = renvoAppend32(out, 0x1b)
	out = renvoAppend32(out, 24)
	out = renvoAppend32(out, 0x52544758)
	out = renvoAppend32(out, 0x44415257)
	out = renvoAppend32(out, 0x494e4152)
	out = renvoAppend32(out, 0x4d363400)
	out = renvoAppend32(out, 0x32)
	out = renvoAppend32(out, 24)
	out = renvoAppend32(out, 1)
	out = renvoAppend32(out, 0x000b0000)
	out = renvoAppend32(out, 0x000b0000)
	out = renvoAppend32(out, 0)
	out = renvoAppend32(out, 0x80000028)
	out = renvoAppend32(out, 24)
	out = renvoAppend64U32(out, a.codeOffset)
	out = renvoAppend64U32(out, 0)
	out = renvoAppend32(out, 0x1d)
	out = renvoAppend32(out, 16)
	out = renvoAppend32(out, sigOff)
	out = renvoAppend32(out, sigSize)
	return renvoAppendUntil(out, a.codeOffset)
}

type renvoDarwinImportLayout struct {
	gotOffs       []int
	staticGotOffs []int
}

func renvoDarwinPrepareImports(a *renvoAsm) renvoDarwinImportLayout {
	var layout renvoDarwinImportLayout
	layout.gotOffs = make([]int, renvoDarwinImportCount+1)
	layout.staticGotOffs = make([]int, len(a.darwinImports))
	stubAts := make([]int, renvoDarwinImportCount+1)
	staticStubAts := make([]int, len(a.darwinImports))
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		label := a.darwinImportLabels[id]
		renvoAarch64AsmAlign(a)
		renvoAsmMarkLabel(a, label)
		stubAts[id] = len(a.code)
		renvoAarch64AsmEmit(a, 0x90000010)
		renvoAarch64AsmEmit(a, 0xf9400210)
		renvoAarch64AsmEmit(a, 0xd61f0200)
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if !a.darwinImports[i].used {
			continue
		}
		renvoAarch64AsmAlign(a)
		renvoAsmMarkLabel(a, a.darwinImports[i].label)
		staticStubAts[i] = len(a.code)
		renvoAarch64AsmEmit(a, 0x90000010)
		renvoAarch64AsmEmit(a, 0xf9400210)
		renvoAarch64AsmEmit(a, 0xd61f0200)
	}
	renvoAsmPatch(a)
	dataFileOff := renvoAlignValue(a.codeOffset+len(a.code), renvoDarwinArm64PageSize)
	a.dataOffset = dataFileOff
	a.data = renvoAppendUntil(a.data, renvoAlignValue(len(a.data), 8))
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		layout.gotOffs[id] = len(a.data)
		a.data = renvoAppend64U32(a.data, 0)
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if !a.darwinImports[i].used {
			continue
		}
		layout.staticGotOffs[i] = len(a.data)
		a.data = renvoAppend64U32(a.data, 0)
	}
	a.data = renvoAppendUntil(a.data, renvoAlignValue(len(a.data), renvoDarwinArm64PageSize))
	renvoAsmPatchAarch64AbsDarwin(a)
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		stubAt := stubAts[id]
		target := renvoDarwinArm64ImageBase + dataFileOff + layout.gotOffs[id]
		pc := renvoDarwinArm64ImageBase + a.codeOffset + stubAt
		renvoDarwinPatchArm64Adrp(a.code, stubAt, pc, target)
		pageOff := target & 0xfff
		renvoPut32At(a.code, stubAt+4, 0xf9400210|((pageOff/8)<<10))
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if !a.darwinImports[i].used {
			continue
		}
		stubAt := staticStubAts[i]
		target := renvoDarwinArm64ImageBase + dataFileOff + layout.staticGotOffs[i]
		pc := renvoDarwinArm64ImageBase + a.codeOffset + stubAt
		renvoDarwinPatchArm64Adrp(a.code, stubAt, pc, target)
		pageOff := target & 0xfff
		renvoPut32At(a.code, stubAt+4, 0xf9400210|((pageOff/8)<<10))
	}
	return layout
}

func renvoDarwinBuildBindData(a *renvoAsm, gotOffs []int, staticGotOffs []int) []byte {
	bind := make([]byte, 0, 256)
	bind = append(bind, 0x11)
	bind = append(bind, 0x51)
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		bind = append(bind, 0x40)
		bind = renvoAppendStringZ(bind, renvoDarwinImportName(id))
		bind = append(bind, 0x72)
		bind = renvoDarwinAppendULEB(bind, gotOffs[id])
		bind = append(bind, 0x90)
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if !a.darwinImports[i].used {
			continue
		}
		ordinal := renvoDarwinDylibOrdinal(a, a.darwinImports[i].dylib)
		if ordinal <= 0 || ordinal >= 16 {
			continue
		}
		bind = append(bind, byte(0x10|ordinal))
		bind = append(bind, 0x40)
		bind = renvoAppendStringZ(bind, a.darwinImports[i].name)
		bind = append(bind, 0x72)
		bind = renvoDarwinAppendULEB(bind, staticGotOffs[i])
		bind = append(bind, 0x90)
	}
	return append(bind, 0)
}

func renvoDarwinBuildStringTable(a *renvoAsm) []byte {
	strtab := make([]byte, 0, 128)
	strtab = append(strtab, 0)
	strtab = renvoAppendStringZ(strtab, "_main")
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		strtab = renvoAppendStringZ(strtab, renvoDarwinImportName(id))
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if a.darwinImports[i].used {
			strtab = renvoAppendStringZ(strtab, a.darwinImports[i].name)
		}
	}
	return strtab
}

func renvoDarwinBuildSymbolTable(a *renvoAsm) []byte {
	symtab := make([]byte, 0, 16*(renvoDarwinImportCount+1))
	symtab = renvoAppend32(symtab, 1)
	symtab = append(symtab, 0x0f)
	symtab = append(symtab, 1)
	symtab = renvoAppend16(symtab, 0)
	symtab = renvoAppend64(symtab, renvoDarwinArm64ImageBase+a.codeOffset)
	nameOff := 7
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		symtab = renvoAppend32(symtab, nameOff)
		symtab = append(symtab, 1)
		symtab = append(symtab, 0)
		symtab = renvoAppend16(symtab, 0x100)
		symtab = renvoAppend64(symtab, 0)
		nameOff += len(renvoDarwinImportName(id)) + 1
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if !a.darwinImports[i].used {
			continue
		}
		symtab = renvoAppend32(symtab, nameOff)
		symtab = append(symtab, 1)
		symtab = append(symtab, 0)
		symtab = renvoAppend16(symtab, 0x100)
		symtab = renvoAppend64(symtab, 0)
		nameOff += len(a.darwinImports[i].name) + 1
	}
	return symtab
}

func renvoDarwinUsedImportCount(a *renvoAsm) int {
	count := 0
	for id := 1; id <= renvoDarwinImportCount; id++ {
		if len(a.darwinImportUsed) > id && a.darwinImportUsed[id] {
			count++
		}
	}
	for i := 0; i < len(a.darwinImports); i++ {
		if a.darwinImports[i].used {
			count++
		}
	}
	return count
}

func renvoAsmImageDarwinArm64(a *renvoAsm) []byte {
	imports := renvoDarwinPrepareImports(a)
	dataFileOff := renvoAlignValue(a.codeOffset+len(a.code), renvoDarwinArm64PageSize)
	bind := renvoDarwinBuildBindData(a, imports.gotOffs, imports.staticGotOffs)
	symtab := renvoDarwinBuildSymbolTable(a)
	strtab := renvoDarwinBuildStringTable(a)
	undefinedCount := renvoDarwinUsedImportCount(a)

	textFileSize := dataFileOff
	dataFileSize := len(a.data)
	dataVMSize := renvoAlignValue(len(a.data)+a.bssSize, renvoDarwinArm64PageSize)
	linkeditOff := dataFileOff + dataFileSize
	bindOff := linkeditOff
	symOff := renvoAlignValue(bindOff+len(bind), 8)
	strOff := symOff + len(symtab)
	sigOff := renvoAlignValue(strOff+len(strtab), 16)
	sigSize := renvoDarwinCodeSignatureSize(sigOff, "renvo")
	linkeditSize := sigOff + sigSize - linkeditOff

	out := renvoDarwinMachHeader(a, textFileSize, dataFileOff, dataFileSize, dataVMSize, linkeditOff, linkeditSize, bindOff, len(bind), symOff, 1+undefinedCount, strOff, len(strtab), undefinedCount, sigOff, sigSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = renvoAppendUntil(out, dataFileOff)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = renvoAppendUntil(out, bindOff)
	for i := 0; i < len(bind); i++ {
		out = append(out, bind[i])
	}
	out = renvoAppendUntil(out, symOff)
	for i := 0; i < len(symtab); i++ {
		out = append(out, symtab[i])
	}
	for i := 0; i < len(strtab); i++ {
		out = append(out, strtab[i])
	}
	out = renvoAppendUntil(out, sigOff)
	sig := renvoDarwinCodeSignature(out, "renvo", textFileSize)
	for i := 0; i < len(sig); i++ {
		out = append(out, sig[i])
	}
	return out
}

func renvoAsmPatchAarch64AbsDarwin(a *renvoAsm) {
	for i := 0; i+2 < len(a.absRelocs); i += 3 {
		at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i))
		off := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+1))
		kind := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i+2))
		target := a.dataOffset + off
		if kind == renvoAbsBssReloc {
			target = renvoAsmBssOffset(a) + off
		}
		insn := renvoGet32At(a.code, at)
		reg := insn & 31
		address := renvoDarwinArm64ImageBase + target
		pc := renvoDarwinArm64ImageBase + a.codeOffset + at
		delta := (address >> 12) - (pc >> 12)
		imm := delta & 0x1fffff
		renvoPut32At(a.code, at, 0x90000000|((imm&3)<<29)|(((imm>>2)&0x7ffff)<<5)|reg)
		renvoPut32At(a.code, at+4, 0x91000000|((address&0xfff)<<10)|(reg<<5)|reg)
		renvoPut32At(a.code, at+8, 0xd503201f)
		renvoPut32At(a.code, at+12, 0xd503201f)
	}
}

func renvoDarwinAppendBE32(out []byte, value int) []byte {
	out = append(out, byte(value>>24))
	out = append(out, byte(value>>16))
	out = append(out, byte(value>>8))
	out = append(out, byte(value))
	return out
}

func renvoDarwinAppendBE64(out []byte, value int) []byte {
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, value)
	return out
}

func renvoDarwinCodeSignatureSize(codeLimit int, ident string) int {
	slots := (codeLimit + 16383) / 16384
	cdSize := 88 + len(ident) + 1 + slots*32
	return 20 + cdSize
}

func renvoDarwinCodeSignature(code []byte, ident string, execLimit int) []byte {
	slots := (len(code) + 16383) / 16384
	identOff := 88
	hashOff := identOff + len(ident) + 1
	cdSize := hashOff + slots*32
	totalSize := 20 + cdSize
	out := make([]byte, 0, totalSize)
	out = renvoDarwinAppendBE32(out, 0xfade0cc0)
	out = renvoDarwinAppendBE32(out, totalSize)
	out = renvoDarwinAppendBE32(out, 1)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, 20)
	out = renvoDarwinAppendBE32(out, 0xfade0c02)
	out = renvoDarwinAppendBE32(out, cdSize)
	out = renvoDarwinAppendBE32(out, 0x20400)
	out = renvoDarwinAppendBE32(out, 0x2)
	out = renvoDarwinAppendBE32(out, hashOff)
	out = renvoDarwinAppendBE32(out, identOff)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, slots)
	out = renvoDarwinAppendBE32(out, len(code))
	out = append(out, 32)
	out = append(out, 2)
	out = append(out, 0)
	out = append(out, 14)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE32(out, 0)
	out = renvoDarwinAppendBE64(out, 0)
	out = renvoDarwinAppendBE64(out, 0)
	out = renvoDarwinAppendBE64(out, execLimit)
	out = renvoDarwinAppendBE64(out, 1)
	for i := 0; i < len(ident); i++ {
		out = append(out, ident[i])
	}
	out = append(out, 0)
	for slot := 0; slot < slots; slot++ {
		start := slot * 16384
		end := start + 16384
		if end > len(code) {
			end = len(code)
		}
		hash := renvoDarwinSHA256(code[start:end])
		for i := 0; i < len(hash); i++ {
			out = append(out, hash[i])
		}
	}
	return out
}

func renvoDarwinSHA256Constants() []int {
	return []int{
		0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
		0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
		0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
		0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
		0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
		0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
		0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
		0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
	}
}

func renvoDarwinSHA256Schedule(msg []byte, chunk int, w []int) {
	for i := 0; i < 16; i++ {
		at := chunk + i*4
		w[i] = (int(msg[at])<<24 | int(msg[at+1])<<16 | int(msg[at+2])<<8 | int(msg[at+3])) & 0xffffffff
	}
	for i := 16; i < 64; i++ {
		x := w[i-15] & 0xffffffff
		y := w[i-2] & 0xffffffff
		s0 := ((x >> 7) | ((x << 25) & 0xffffffff)) ^ ((x >> 18) | ((x << 14) & 0xffffffff)) ^ (x >> 3)
		s1 := ((y >> 17) | ((y << 15) & 0xffffffff)) ^ ((y >> 19) | ((y << 13) & 0xffffffff)) ^ (y >> 10)
		w[i] = (w[i-16] + s0 + w[i-7] + s1) & 0xffffffff
	}
}

func renvoDarwinSHA256Rounds(w []int, k []int, hvals []int) {
	a := hvals[0]
	b := hvals[1]
	c := hvals[2]
	d := hvals[3]
	e := hvals[4]
	f := hvals[5]
	g := hvals[6]
	h := hvals[7]
	for i := 0; i < 64; i++ {
		e32 := e & 0xffffffff
		a32 := a & 0xffffffff
		s1 := ((e32 >> 6) | ((e32 << 26) & 0xffffffff)) ^ ((e32 >> 11) | ((e32 << 21) & 0xffffffff)) ^ ((e32 >> 25) | ((e32 << 7) & 0xffffffff))
		choose := (e & f) ^ ((e ^ 0xffffffff) & g)
		t1 := (h + s1 + choose + k[i] + w[i]) & 0xffffffff
		s0 := ((a32 >> 2) | ((a32 << 30) & 0xffffffff)) ^ ((a32 >> 13) | ((a32 << 19) & 0xffffffff)) ^ ((a32 >> 22) | ((a32 << 10) & 0xffffffff))
		majority := (a & b) ^ (a & c) ^ (b & c)
		t2 := (s0 + majority) & 0xffffffff
		h = g
		g = f
		f = e
		e = (d + t1) & 0xffffffff
		d = c
		c = b
		b = a
		a = (t1 + t2) & 0xffffffff
	}
	hvals[0] = (hvals[0] + a) & 0xffffffff
	hvals[1] = (hvals[1] + b) & 0xffffffff
	hvals[2] = (hvals[2] + c) & 0xffffffff
	hvals[3] = (hvals[3] + d) & 0xffffffff
	hvals[4] = (hvals[4] + e) & 0xffffffff
	hvals[5] = (hvals[5] + f) & 0xffffffff
	hvals[6] = (hvals[6] + g) & 0xffffffff
	hvals[7] = (hvals[7] + h) & 0xffffffff
}

func renvoDarwinSHA256Compress(msg []byte, hvals []int) {
	k := renvoDarwinSHA256Constants()
	w := make([]int, 64)
	for chunk := 0; chunk < len(msg); chunk += 64 {
		renvoDarwinSHA256Schedule(msg, chunk, w)
		renvoDarwinSHA256Rounds(w, k, hvals)
	}
}

func renvoDarwinSHA256(data []byte) []byte {
	hardware := make([]byte, 32)
	if renvoDarwinCCSHA256(data, len(data), hardware) != 0 {
		return hardware
	}
	msgLen := len(data) + 1 + 8
	msgLen = renvoAlignValue(msgLen, 64)
	msg := make([]byte, msgLen)
	for i := 0; i < len(data); i++ {
		msg[i] = data[i]
	}
	msg[len(data)] = 0x80
	bits := len(data) * 8
	for i := 0; i < 8; i++ {
		msg[msgLen-1-i] = byte(bits >> (i * 8))
	}
	vals := []int{0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19}
	renvoDarwinSHA256Compress(msg, vals)
	out := make([]byte, 0, 32)
	for i := 0; i < len(vals); i++ {
		out = append(out, byte(vals[i]>>24))
		out = append(out, byte(vals[i]>>16))
		out = append(out, byte(vals[i]>>8))
		out = append(out, byte(vals[i]))
	}
	return out
}
