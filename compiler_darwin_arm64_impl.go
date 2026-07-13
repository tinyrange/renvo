package main

const rtgDarwinArm64CodeOffset = 0x1000
const rtgDarwinArm64ImageBase = 0x100000000
const rtgDarwinArm64PageSize = 0x4000

const rtgDarwinImportExit = 1
const rtgDarwinImportOpen = 2
const rtgDarwinImportClose = 3
const rtgDarwinImportRead = 4
const rtgDarwinImportWrite = 5
const rtgDarwinImportPread = 6
const rtgDarwinImportPwrite = 7
const rtgDarwinImportFchmod = 8
const rtgDarwinImportGetdirentries = 9
const rtgDarwinImportCount = 9

func compileDarwinArm64(input []int, output int) int {
	rtgSetTarget(rtgTargetDarwinArm64)
	return rtgCompileAarch64(input, output)
}

func rtgDarwinImportName(id int) string {
	if id == rtgDarwinImportExit {
		return "_exit"
	}
	if id == rtgDarwinImportOpen {
		return "_open"
	}
	if id == rtgDarwinImportClose {
		return "_close"
	}
	if id == rtgDarwinImportRead {
		return "_read"
	}
	if id == rtgDarwinImportWrite {
		return "_write"
	}
	if id == rtgDarwinImportPread {
		return "_pread"
	}
	if id == rtgDarwinImportPwrite {
		return "_pwrite"
	}
	if id == rtgDarwinImportFchmod {
		return "_fchmod"
	}
	if id == rtgDarwinImportGetdirentries {
		return "_getdirentries"
	}
	return ""
}

func rtgDarwinArm64ImportLabel(a *rtgAsm, id int) int {
	if len(a.darwinImportLabels) == 0 {
		a.darwinImportLabels = make([]int, rtgDarwinImportCount+1)
		a.darwinImportUsed = make([]bool, rtgDarwinImportCount+1)
		for i := 0; i <= rtgDarwinImportCount; i++ {
			a.darwinImportLabels[i] = -1
		}
	}
	if id <= 0 || id > rtgDarwinImportCount {
		return -1
	}
	if a.darwinImportLabels[id] < 0 {
		a.darwinImportLabels[id] = rtgAsmNewLabel(a)
	}
	a.darwinImportUsed[id] = true
	return a.darwinImportLabels[id]
}

func rtgDarwinArm64CallImport(a *rtgAsm, id int) {
	label := rtgDarwinArm64ImportLabel(a, id)
	rtgAarch64AsmCallLabel(a, label)
}

func rtgDarwinArm64CallVirtualArgs(a *rtgAsm, id int, argCount int) {
	if argCount > 2 {
		rtgAarch64AsmMovRegReg(a, rtgAarch64RegTmp, rtgAarch64RegRdx)
	}
	if argCount > 0 {
		rtgAarch64AsmMovRegReg(a, 0, rtgAarch64RegRdi)
	}
	if argCount > 1 {
		rtgAarch64AsmMovRegReg(a, 1, rtgAarch64RegRsi)
	}
	if argCount > 2 {
		rtgAarch64AsmMovRegReg(a, 2, rtgAarch64RegTmp)
	}
	if argCount > 3 {
		rtgAarch64AsmMovRegReg(a, 3, rtgAarch64RegR10)
	}
	rtgDarwinArm64CallImport(a, id)
}

func rtgEmitProgramEntryArgsDarwinArm64(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(g.meta, app.resultType) {
		return false
	}
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !rtgTypeIsStringSlice(g.meta, first.typ) {
		return false
	}
	if app.paramCount == 2 {
		second := &g.meta.params[app.firstParam+1]
		if !rtgTypeIsStringSlice(g.meta, second.typ) {
			return false
		}
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envOff := g.asm.bssSize
	g.asm.bssSize += 32768
	rtgAsmBuildDarwinArgvEnvSlicesArm64(&g.asm, g.darwinEntryOff, argsOff, envOff)
	return true
}

func rtgAsmBuildDarwinArgvEnvSlicesArm64(a *rtgAsm, entryOff int, argsOff int, envOff int) {
	argLoop := rtgAsmNewLabel(a)
	argLenLoop := rtgAsmNewLabel(a)
	argLenDone := rtgAsmNewLabel(a)
	argsDone := rtgAsmNewLabel(a)
	envLoop := rtgAsmNewLabel(a)
	envLenLoop := rtgAsmNewLabel(a)
	envLenDone := rtgAsmNewLabel(a)
	envDone := rtgAsmNewLabel(a)

	// dyld calls LC_MAIN with argc, argv, envp in x0, x1, x2. Global
	// initialization runs first, so reload the entry values saved at startup.
	rtgAarch64AsmMovRegAbs(a, 9, entryOff, rtgAbsBssReloc)
	rtgAarch64AsmLoadRegMem(a, 13, 9, 0, 8)
	rtgAarch64AsmLoadRegMem(a, 14, 9, 8, 8)
	rtgAarch64AsmLoadRegMem(a, 15, 9, 16, 8)
	rtgAarch64AsmMovRegAbs(a, 10, argsOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 11, 0)
	rtgAsmMarkLabel(a, argLoop)
	rtgAarch64AsmCmpRegReg(a, 11, 13)
	rtgAarch64AsmBCondLabel(a, argsDone, 0)
	rtgAarch64AsmAddRegRegShift(a, 12, 14, 11, 3)
	rtgAarch64AsmLoadRegMem(a, 12, 12, 0, 8)
	rtgAarch64AsmStoreRegMem(a, 12, 10, 0, 8)
	rtgAarch64AsmMovRegImm(a, 0, 0)
	rtgAsmMarkLabel(a, argLenLoop)
	rtgAarch64AsmAddRegReg(a, 9, 12, 0)
	rtgAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 9, 0)
	rtgAarch64AsmBCondLabel(a, argLenDone, 0)
	rtgAarch64AsmAddRegImm(a, 0, 0, 1)
	rtgAarch64AsmJmpLabel(a, argLenLoop)
	rtgAsmMarkLabel(a, argLenDone)
	rtgAarch64AsmStoreRegMem(a, 0, 10, 8, 8)
	rtgAarch64AsmAddRegImm(a, 10, 10, 16)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmJmpLabel(a, argLoop)
	rtgAsmMarkLabel(a, argsDone)

	rtgAarch64AsmMovRegAbs(a, 10, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegImm(a, 11, 0)
	rtgAsmMarkLabel(a, envLoop)
	rtgAarch64AsmLoadRegMem(a, 12, 15, 0, 8)
	rtgAarch64AsmCmpRegImm(a, 12, 0)
	rtgAarch64AsmBCondLabel(a, envDone, 0)
	rtgAarch64AsmStoreRegMem(a, 12, 10, 0, 8)
	rtgAarch64AsmMovRegImm(a, 0, 0)
	rtgAsmMarkLabel(a, envLenLoop)
	rtgAarch64AsmAddRegReg(a, 9, 12, 0)
	rtgAarch64AsmLoadRegMem(a, 9, 9, 0, 1)
	rtgAarch64AsmCmpRegImm(a, 9, 0)
	rtgAarch64AsmBCondLabel(a, envLenDone, 0)
	rtgAarch64AsmAddRegImm(a, 0, 0, 1)
	rtgAarch64AsmJmpLabel(a, envLenLoop)
	rtgAsmMarkLabel(a, envLenDone)
	rtgAarch64AsmStoreRegMem(a, 0, 10, 8, 8)
	rtgAarch64AsmAddRegImm(a, 10, 10, 16)
	rtgAarch64AsmAddRegImm(a, 15, 15, 8)
	rtgAarch64AsmAddRegImm(a, 11, 11, 1)
	rtgAarch64AsmJmpLabel(a, envLoop)
	rtgAsmMarkLabel(a, envDone)

	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRdi, argsOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRsi, 13)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegRdx, 13)
	rtgAarch64AsmMovRegAbs(a, rtgAarch64RegRcx, envOff, rtgAbsBssReloc)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR8, 11)
	rtgAarch64AsmMovRegReg(a, rtgAarch64RegR9, 11)
}

func rtgDarwinAppendULEB(out []byte, value int) []byte {
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

func rtgDarwinAppendName16(out []byte, name string) []byte {
	for i := 0; i < 16; i++ {
		if i < len(name) {
			out = append(out, name[i])
		} else {
			out = append(out, 0)
		}
	}
	return out
}

func rtgDarwinAppendSegment64(out []byte, name string, vmaddr int, vmsize int, fileoff int, filesize int, maxprot int, initprot int) []byte {
	out = rtgAppend32(out, 0x19)
	out = rtgAppend32(out, 72)
	out = rtgDarwinAppendName16(out, name)
	out = rtgAppend64(out, vmaddr)
	out = rtgAppend64(out, vmsize)
	out = rtgAppend64(out, fileoff)
	out = rtgAppend64(out, filesize)
	out = rtgAppend32(out, maxprot)
	out = rtgAppend32(out, initprot)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	return out
}

func rtgDarwinAppendSection64(out []byte, section string, segment string, addr int, size int, offset int, align int, flags int) []byte {
	out = rtgDarwinAppendName16(out, section)
	out = rtgDarwinAppendName16(out, segment)
	out = rtgAppend64(out, addr)
	out = rtgAppend64(out, size)
	out = rtgAppend32(out, offset)
	out = rtgAppend32(out, align)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, flags)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	return out
}

func rtgDarwinAppendTextSegment64(out []byte, codeSize int, fileSize int) []byte {
	out = rtgAppend32(out, 0x19)
	out = rtgAppend32(out, 152)
	out = rtgDarwinAppendName16(out, "__TEXT")
	out = rtgAppend64(out, rtgDarwinArm64ImageBase)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend64(out, 0)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend32(out, 5)
	out = rtgAppend32(out, 5)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 0)
	out = rtgDarwinAppendSection64(out, "__text", "__TEXT", rtgDarwinArm64ImageBase+rtgDarwinArm64CodeOffset, codeSize, rtgDarwinArm64CodeOffset, 2, 0x80000400)
	return out
}

func rtgDarwinAppendDataSegment64(out []byte, dataFileOff int, dataFileSize int, bssSize int) []byte {
	vmSize := rtgAlignValue(dataFileSize+bssSize, rtgDarwinArm64PageSize)
	out = rtgAppend32(out, 0x19)
	out = rtgAppend32(out, 232)
	out = rtgDarwinAppendName16(out, "__DATA")
	out = rtgAppend64(out, rtgDarwinArm64ImageBase+dataFileOff)
	out = rtgAppend64(out, vmSize)
	out = rtgAppend64(out, dataFileOff)
	out = rtgAppend64(out, dataFileSize)
	out = rtgAppend32(out, 3)
	out = rtgAppend32(out, 3)
	out = rtgAppend32(out, 2)
	out = rtgAppend32(out, 0)
	out = rtgDarwinAppendSection64(out, "__data", "__DATA", rtgDarwinArm64ImageBase+dataFileOff, dataFileSize, dataFileOff, 3, 0)
	out = rtgDarwinAppendSection64(out, "__bss", "__DATA", rtgDarwinArm64ImageBase+dataFileOff+dataFileSize, bssSize, 0, 3, 1)
	return out
}

func rtgDarwinPatchArm64Adrp(code []byte, at int, pc int, target int) {
	delta := (target >> 12) - (pc >> 12)
	imm := delta & 0x1fffff
	insn := 0x90000010 | ((imm & 3) << 29) | (((imm >> 2) & 0x7ffff) << 5)
	rtgPut32At(code, at, insn)
}

func rtgDarwinMachHeader(a *rtgAsm, textFileSize int, dataFileOff int, dataFileSize int, dataVMSize int, linkeditOff int, linkeditSize int, bindOff int, bindSize int, symOff int, symbolCount int, strOff int, strSize int, undefinedCount int, sigOff int, sigSize int) []byte {
	out := make([]byte, 0, a.codeOffset)
	out = rtgAppend32(out, 0xfeedfacf)
	out = rtgAppend32(out, 0x0100000c)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 2)
	out = rtgAppend32(out, 13)
	out = rtgAppend32(out, 856)
	out = rtgAppend32(out, 0x200085)
	out = rtgAppend32(out, 0)
	out = rtgDarwinAppendSegment64(out, "__PAGEZERO", 0, rtgDarwinArm64ImageBase, 0, 0, 0, 0)
	out = rtgDarwinAppendTextSegment64(out, len(a.code), textFileSize)
	out = rtgDarwinAppendDataSegment64(out, dataFileOff, dataFileSize, a.bssSize)
	out = rtgDarwinAppendSegment64(out, "__LINKEDIT", rtgDarwinArm64ImageBase+dataFileOff+dataVMSize, rtgAlignValue(linkeditSize, rtgDarwinArm64PageSize), linkeditOff, linkeditSize, 1, 1)
	out = rtgAppend32(out, 0x80000022)
	out = rtgAppend32(out, 48)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, bindOff)
	out = rtgAppend32(out, bindSize)
	for i := 0; i < 6; i++ {
		out = rtgAppend32(out, 0)
	}
	out = rtgAppend32(out, 0x02)
	out = rtgAppend32(out, 24)
	out = rtgAppend32(out, symOff)
	out = rtgAppend32(out, symbolCount)
	out = rtgAppend32(out, strOff)
	out = rtgAppend32(out, strSize)
	out = rtgAppend32(out, 0x0b)
	out = rtgAppend32(out, 80)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, undefinedCount)
	for i := 0; i < 12; i++ {
		out = rtgAppend32(out, 0)
	}
	cmdStart := len(out)
	out = rtgAppend32(out, 0x0e)
	out = rtgAppend32(out, 32)
	out = rtgAppend32(out, 12)
	out = rtgAppendStringZ(out, "/usr/lib/dyld")
	out = rtgAppendUntil(out, cmdStart+32)
	cmdStart = len(out)
	out = rtgAppend32(out, 0x0c)
	out = rtgAppend32(out, 56)
	out = rtgAppend32(out, 24)
	out = rtgAppend32(out, 2)
	out = rtgAppend32(out, 0x00010000)
	out = rtgAppend32(out, 0x00010000)
	out = rtgAppendStringZ(out, "/usr/lib/libSystem.B.dylib")
	out = rtgAppendUntil(out, cmdStart+56)
	out = rtgAppend32(out, 0x1b)
	out = rtgAppend32(out, 24)
	out = rtgAppend32(out, 0x52544758)
	out = rtgAppend32(out, 0x44415257)
	out = rtgAppend32(out, 0x494e4152)
	out = rtgAppend32(out, 0x4d363400)
	out = rtgAppend32(out, 0x32)
	out = rtgAppend32(out, 24)
	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 0x000b0000)
	out = rtgAppend32(out, 0x000b0000)
	out = rtgAppend32(out, 0)
	out = rtgAppend32(out, 0x80000028)
	out = rtgAppend32(out, 24)
	out = rtgAppend64U32(out, a.codeOffset)
	out = rtgAppend64U32(out, 0)
	out = rtgAppend32(out, 0x1d)
	out = rtgAppend32(out, 16)
	out = rtgAppend32(out, sigOff)
	out = rtgAppend32(out, sigSize)
	return rtgAppendUntil(out, a.codeOffset)
}

type rtgDarwinImportLayout struct {
	gotOffs []int
}

func rtgDarwinPrepareImports(a *rtgAsm) rtgDarwinImportLayout {
	var layout rtgDarwinImportLayout
	layout.gotOffs = make([]int, rtgDarwinImportCount+1)
	stubAts := make([]int, rtgDarwinImportCount+1)
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		label := a.darwinImportLabels[id]
		rtgAarch64AsmAlign(a)
		rtgAsmMarkLabel(a, label)
		stubAts[id] = len(a.code)
		rtgAarch64AsmEmit(a, 0x90000010)
		rtgAarch64AsmEmit(a, 0xf9400210)
		rtgAarch64AsmEmit(a, 0xd61f0200)
	}
	rtgAsmPatch(a)
	dataFileOff := rtgAlignValue(a.codeOffset+len(a.code), rtgDarwinArm64PageSize)
	a.dataOffset = dataFileOff
	a.data = rtgAppendUntil(a.data, rtgAlignValue(len(a.data), 8))
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		layout.gotOffs[id] = len(a.data)
		a.data = rtgAppend64U32(a.data, 0)
	}
	a.data = rtgAppendUntil(a.data, rtgAlignValue(len(a.data), rtgDarwinArm64PageSize))
	rtgAsmPatchAarch64AbsDarwin(a)
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		stubAt := stubAts[id]
		target := rtgDarwinArm64ImageBase + dataFileOff + layout.gotOffs[id]
		pc := rtgDarwinArm64ImageBase + a.codeOffset + stubAt
		rtgDarwinPatchArm64Adrp(a.code, stubAt, pc, target)
		pageOff := target & 0xfff
		rtgPut32At(a.code, stubAt+4, 0xf9400210|((pageOff/8)<<10))
	}
	return layout
}

func rtgDarwinBuildBindData(a *rtgAsm, gotOffs []int) []byte {
	bind := make([]byte, 0, 256)
	bind = append(bind, 0x11)
	bind = append(bind, 0x51)
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		bind = append(bind, 0x40)
		bind = rtgAppendStringZ(bind, rtgDarwinImportName(id))
		bind = append(bind, 0x72)
		bind = rtgDarwinAppendULEB(bind, gotOffs[id])
		bind = append(bind, 0x90)
	}
	return append(bind, 0)
}

func rtgDarwinBuildStringTable(a *rtgAsm) []byte {
	strtab := make([]byte, 0, 128)
	strtab = append(strtab, 0)
	strtab = rtgAppendStringZ(strtab, "_main")
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		strtab = rtgAppendStringZ(strtab, rtgDarwinImportName(id))
	}
	return strtab
}

func rtgDarwinBuildSymbolTable(a *rtgAsm) []byte {
	symtab := make([]byte, 0, 16*(rtgDarwinImportCount+1))
	symtab = rtgAppend32(symtab, 1)
	symtab = append(symtab, 0x0f)
	symtab = append(symtab, 1)
	symtab = rtgAppend16(symtab, 0)
	symtab = rtgAppend64(symtab, rtgDarwinArm64ImageBase+a.codeOffset)
	nameOff := 7
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) <= id || !a.darwinImportUsed[id] {
			continue
		}
		symtab = rtgAppend32(symtab, nameOff)
		symtab = append(symtab, 1)
		symtab = append(symtab, 0)
		symtab = rtgAppend16(symtab, 0x100)
		symtab = rtgAppend64(symtab, 0)
		nameOff += len(rtgDarwinImportName(id)) + 1
	}
	return symtab
}

func rtgDarwinUsedImportCount(a *rtgAsm) int {
	count := 0
	for id := 1; id <= rtgDarwinImportCount; id++ {
		if len(a.darwinImportUsed) > id && a.darwinImportUsed[id] {
			count++
		}
	}
	return count
}

func rtgAsmImageDarwinArm64(a *rtgAsm) []byte {
	imports := rtgDarwinPrepareImports(a)
	dataFileOff := rtgAlignValue(a.codeOffset+len(a.code), rtgDarwinArm64PageSize)
	bind := rtgDarwinBuildBindData(a, imports.gotOffs)
	symtab := rtgDarwinBuildSymbolTable(a)
	strtab := rtgDarwinBuildStringTable(a)
	undefinedCount := rtgDarwinUsedImportCount(a)

	textFileSize := dataFileOff
	dataFileSize := len(a.data)
	dataVMSize := rtgAlignValue(len(a.data)+a.bssSize, rtgDarwinArm64PageSize)
	linkeditOff := dataFileOff + dataFileSize
	bindOff := linkeditOff
	symOff := rtgAlignValue(bindOff+len(bind), 8)
	strOff := symOff + len(symtab)
	sigOff := rtgAlignValue(strOff+len(strtab), 16)
	sigSize := rtgDarwinCodeSignatureSize(sigOff, "rtg")
	linkeditSize := sigOff + sigSize - linkeditOff

	out := rtgDarwinMachHeader(a, textFileSize, dataFileOff, dataFileSize, dataVMSize, linkeditOff, linkeditSize, bindOff, len(bind), symOff, 1+undefinedCount, strOff, len(strtab), undefinedCount, sigOff, sigSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = rtgAppendUntil(out, dataFileOff)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = rtgAppendUntil(out, bindOff)
	for i := 0; i < len(bind); i++ {
		out = append(out, bind[i])
	}
	out = rtgAppendUntil(out, symOff)
	for i := 0; i < len(symtab); i++ {
		out = append(out, symtab[i])
	}
	for i := 0; i < len(strtab); i++ {
		out = append(out, strtab[i])
	}
	out = rtgAppendUntil(out, sigOff)
	sig := rtgDarwinCodeSignature(out, "rtg", textFileSize)
	for i := 0; i < len(sig); i++ {
		out = append(out, sig[i])
	}
	return out
}

func rtgAsmPatchAarch64AbsDarwin(a *rtgAsm) {
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := a.dataOffset + r.off
		if r.kind == rtgAbsBssReloc {
			target = a.dataOffset + len(a.data) + r.off
		}
		insn := rtgGet32At(a.code, r.at)
		reg := insn & 31
		address := rtgDarwinArm64ImageBase + target
		pc := rtgDarwinArm64ImageBase + a.codeOffset + r.at
		delta := (address >> 12) - (pc >> 12)
		imm := delta & 0x1fffff
		rtgPut32At(a.code, r.at, 0x90000000|((imm&3)<<29)|(((imm>>2)&0x7ffff)<<5)|reg)
		rtgPut32At(a.code, r.at+4, 0x91000000|((address&0xfff)<<10)|(reg<<5)|reg)
		rtgPut32At(a.code, r.at+8, 0xd503201f)
		rtgPut32At(a.code, r.at+12, 0xd503201f)
	}
}

func rtgDarwinAppendBE32(out []byte, value int) []byte {
	out = append(out, byte(value>>24))
	out = append(out, byte(value>>16))
	out = append(out, byte(value>>8))
	out = append(out, byte(value))
	return out
}

func rtgDarwinAppendBE64(out []byte, value int) []byte {
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, value)
	return out
}

func rtgDarwinCodeSignatureSize(codeLimit int, ident string) int {
	slots := (codeLimit + 16383) / 16384
	cdSize := 88 + len(ident) + 1 + slots*32
	return 20 + cdSize
}

func rtgDarwinCodeSignature(code []byte, ident string, execLimit int) []byte {
	slots := (len(code) + 16383) / 16384
	identOff := 88
	hashOff := identOff + len(ident) + 1
	cdSize := hashOff + slots*32
	totalSize := 20 + cdSize
	out := make([]byte, 0, totalSize)
	out = rtgDarwinAppendBE32(out, 0xfade0cc0)
	out = rtgDarwinAppendBE32(out, totalSize)
	out = rtgDarwinAppendBE32(out, 1)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, 20)
	out = rtgDarwinAppendBE32(out, 0xfade0c02)
	out = rtgDarwinAppendBE32(out, cdSize)
	out = rtgDarwinAppendBE32(out, 0x20400)
	out = rtgDarwinAppendBE32(out, 0x2)
	out = rtgDarwinAppendBE32(out, hashOff)
	out = rtgDarwinAppendBE32(out, identOff)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, slots)
	out = rtgDarwinAppendBE32(out, len(code))
	out = append(out, 32)
	out = append(out, 2)
	out = append(out, 0)
	out = append(out, 14)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE32(out, 0)
	out = rtgDarwinAppendBE64(out, 0)
	out = rtgDarwinAppendBE64(out, 0)
	out = rtgDarwinAppendBE64(out, execLimit)
	out = rtgDarwinAppendBE64(out, 1)
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
		hash := rtgDarwinSHA256(code[start:end])
		for i := 0; i < len(hash); i++ {
			out = append(out, hash[i])
		}
	}
	return out
}

func rtgDarwinROR32(value int, shift int) int {
	value = value & 0xffffffff
	return ((value >> shift) | ((value << (32 - shift)) & 0xffffffff)) & 0xffffffff
}

func rtgDarwinSHA256Constants() []int {
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

func rtgDarwinSHA256Schedule(msg []byte, chunk int, w []int) {
	for i := 0; i < 16; i++ {
		at := chunk + i*4
		w[i] = (int(msg[at])<<24 | int(msg[at+1])<<16 | int(msg[at+2])<<8 | int(msg[at+3])) & 0xffffffff
	}
	for i := 16; i < 64; i++ {
		s0 := rtgDarwinROR32(w[i-15], 7) ^ rtgDarwinROR32(w[i-15], 18) ^ (w[i-15] >> 3)
		s1 := rtgDarwinROR32(w[i-2], 17) ^ rtgDarwinROR32(w[i-2], 19) ^ (w[i-2] >> 10)
		w[i] = (w[i-16] + s0 + w[i-7] + s1) & 0xffffffff
	}
}

func rtgDarwinSHA256Round(state []int, word int, constant int) {
	s1 := rtgDarwinROR32(state[4], 6) ^ rtgDarwinROR32(state[4], 11) ^ rtgDarwinROR32(state[4], 25)
	ch := (state[4] & state[5]) ^ ((state[4] ^ 0xffffffff) & state[6])
	t1 := (state[7] + s1 + ch + constant + word) & 0xffffffff
	s0 := rtgDarwinROR32(state[0], 2) ^ rtgDarwinROR32(state[0], 13) ^ rtgDarwinROR32(state[0], 22)
	maj := (state[0] & state[1]) ^ (state[0] & state[2]) ^ (state[1] & state[2])
	t2 := (s0 + maj) & 0xffffffff
	state[7] = state[6]
	state[6] = state[5]
	state[5] = state[4]
	state[4] = (state[3] + t1) & 0xffffffff
	state[3] = state[2]
	state[2] = state[1]
	state[1] = state[0]
	state[0] = (t1 + t2) & 0xffffffff
}

func rtgDarwinSHA256Rounds(w []int, k []int, hvals []int) {
	state := make([]int, 8)
	for i := 0; i < 8; i++ {
		state[i] = hvals[i]
	}
	for i := 0; i < 64; i++ {
		rtgDarwinSHA256Round(state, w[i], k[i])
	}
	for i := 0; i < 8; i++ {
		hvals[i] = (hvals[i] + state[i]) & 0xffffffff
	}
}

func rtgDarwinSHA256Compress(msg []byte, hvals []int) {
	k := rtgDarwinSHA256Constants()
	w := make([]int, 64)
	for chunk := 0; chunk < len(msg); chunk += 64 {
		rtgDarwinSHA256Schedule(msg, chunk, w)
		rtgDarwinSHA256Rounds(w, k, hvals)
	}
}

func rtgDarwinSHA256(data []byte) []byte {
	msgLen := len(data) + 1 + 8
	msgLen = rtgAlignValue(msgLen, 64)
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
	rtgDarwinSHA256Compress(msg, vals)
	out := make([]byte, 0, 32)
	for i := 0; i < len(vals); i++ {
		out = append(out, byte(vals[i]>>24))
		out = append(out, byte(vals[i]>>16))
		out = append(out, byte(vals[i]>>8))
		out = append(out, byte(vals[i]))
	}
	return out
}
