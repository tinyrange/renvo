package main

func renvoWinAmd64CallImport(a *renvoAsm, importID int) {
	renvoNonNil(a)
	// Expression evaluation can reach this boundary at either stack parity.
	// Preserve r12 on the original stack, use it to recover the exact rsp, and
	// construct a fresh, aligned Win64 shadow area for the imported call.
	base := len(a.code)
	renvoAsmEmitText(a, "\x41\x54\x49\x89\xe4\x48\x83\xe4\xf0\x48\x83\xec\x20\xff\x15\x00\x00\x00\x00\x49\x89\xc3\x4c\x89\xe4\x41\x5c\x4c\x89\xd8")
	renvoAsmAddWinImportReloc(a, base+15, importID)
}

func renvoWin386CallImport(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x15ff)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWinAmd64LoadImportPtrRax(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x058b48)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWinAmd64LoadImportPtrR9(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x0d8b4c)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWinAmd64LoadImportPtrR10(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit24(a, 0x158b4c)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386LoadImportPtrRax(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386LoadImportPtrRsi(a *renvoAsm, importID int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x358b)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddWinImportReloc(a, at, importID)
}

func renvoWin386StoreEcxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x0d89)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEbxBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xbb)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEcxBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xb9)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386MovEdiBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xbf)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386PushBssAddr(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0x68)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386LoadEsiBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit16(a, 0x358b)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWin386LoadEaxBss(a *renvoAsm, bssOff int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0xa1)
	at := len(a.code)
	renvoAsmEmit32(a, 0)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
}

func renvoWinAmd64EmitReadWriteHelper(g *renvoLinearGen, isWrite bool) int {
	renvoNonNil(g)
	// The template is the relaxed form of the instruction sequence previously
	// assembled one operation at a time. Explicit relocations record the BSS and
	// import operands that still vary between compiler invocations and self-host.
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
	} else if g.winReadEmitted {
		return g.winReadLabel
	}
	label := renvoAsmNewLabel(a)
	if isWrite {
		g.winWriteEmitted = true
		g.winWriteLabel = label
	} else {
		g.winReadEmitted = true
		g.winReadLabel = label
	}
	countOff := a.bssSize
	a.bssSize += 8
	posOff := a.bssSize
	a.bssSize += 8
	importID := renvoWinImportReadFile
	prefix := "\xe9\x32\x01\x00\x00\x57\x56\x52\x51\x48\x83\x7c\x24\x18\x00\x74\x02\xeb\x18\xb9\xf6"
	if isWrite {
		importID = renvoWinImportWriteFile
		prefix = "\xe9\x54\x01\x00\x00\x57\x56\x52\x51\x48\x83\x7c\x24\x18\x01\x74\x0a\x48\x83\x7c\x24\x18\x02\x74\x1c\xeb\x32\xb9\xf5\xff\xff\xff\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x44\x24\x18\xeb\x18\xb9\xf4"
	}
	base := len(a.code)
	renvoAsmEmitText(a, prefix)
	a.labelPos[label] = int32(base + 5)
	a.lastPrimaryStoreEnd = -1
	commonBase := len(a.code)
	renvoAsmEmitText(a, "\xff\xff\xff\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x44\x24\x18\x48\x83\x3c\x24\x00\x0f\x8c\x80\x00\x00\x00\x48\x8b\x4c\x24\x18\x31\xd2\x45\x31\xc0\x41\xb9\x01\x00\x00\x00\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x89\x05\x00\x00\x00\x00\x48\x8b\x4c\x24\x18\x48\x8b\x14\x24\x45\x31\xc0\x45\x31\xc9\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x8b\x4c\x24\x18\x48\x8b\x54\x24\x10\x4c\x8b\x44\x24\x08\x48\x8d\x05\x00\x00\x00\x00\x49\x89\xc1\x48\x83\xec\x28\x48\xc7\x44\x24\x20\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x83\xf8\x00\x74\x52\x48\x8b\x05\x00\x00\x00\x00\xeb\x4c\x48\x8b\x4c\x24\x18\x48\x8b\x54\x24\x10\x4c\x8b\x44\x24\x08\x48\x8d\x05\x00\x00\x00\x00\x49\x89\xc1\x48\x83\xec\x28\x48\xc7\x44\x24\x20\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x83\xf8\x00\x74\x0c\x48\x8b\x05\x00\x00\x00\x00\x48\x83\xc4\x20\xc3\x6a\xff\x58\x48\x83\xc4\x20\xc3\x6a\xff\x58\x48\x89\x05\x00\x00\x00\x00\x48\x8b\x4c\x24\x18\x48\x8b\x05\x00\x00\x00\x00\x50\x5a\x45\x31\xc0\x45\x31\xc9\x48\x83\xec\x28\xff\x15\x00\x00\x00\x00\x48\x83\xc4\x28\x48\x8b\x05\x00\x00\x00\x00\x48\x83\xc4\x20\xc3")
	if isWrite {
		renvoAsmAddWinImportReloc(a, base+38, renvoWinImportGetStdHandle)
	}
	renvoAsmAddWinImportReloc(a, commonBase+9, renvoWinImportGetStdHandle)
	renvoAsmAddWinImportReloc(a, commonBase+55, renvoWinImportSetFilePointer)
	renvoAsmAddAbsReloc(a, commonBase+66, posOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, commonBase+91, renvoWinImportSetFilePointer)
	renvoAsmAddAbsReloc(a, commonBase+117, countOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, commonBase+139, importID)
	renvoAsmAddAbsReloc(a, commonBase+155, countOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, commonBase+179, countOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, commonBase+201, importID)
	renvoAsmAddAbsReloc(a, commonBase+217, countOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, commonBase+240, countOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, commonBase+252, posOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, commonBase+270, renvoWinImportSetFilePointer)
	renvoAsmAddAbsReloc(a, commonBase+281, countOff, renvoAbsBssReloc)
	return label
}

func renvoWin386EmitReadWriteHelper(g *renvoLinearGen, isWrite bool) int {
	a := &g.asm
	if isWrite {
		if g.winWriteEmitted {
			return g.winWriteLabel
		}
		g.winWriteEmitted = true
		g.winWriteLabel = renvoAsmNewLabel(a)
	} else {
		if g.winReadEmitted {
			return g.winReadLabel
		}
		g.winReadEmitted = true
		g.winReadLabel = renvoAsmNewLabel(a)
	}
	countOff := a.bssSize
	a.bssSize = countOff + 16
	// The helper instruction streams are invariant. Relocation pairs encode
	// their patch offset and either a BSS slot or Windows import identifier.
	label := g.winReadLabel
	base := len(a.code)
	relocs := ""
	if isWrite {
		label = g.winWriteLabel
		renvoAsmEmitText(a, "\xe9\xc4\x00\x00\x00\x83\xfb\x01\x0f\x84\x0e\x00\x00\x00\x83\xfb\x02\x0f\x84\x14\x00\x00\x00\xe9\x19\x00\x00\x00\x6a\xf5\xff\x15\x00\x00\x00\x00\x89\xc3\xe9\x0a\x00\x00\x00\x6a\xf4\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x20\x08\x2f\x08\x49\x07\x4e\x01\x5e\x07\x67\x00\x70\x06\x7e\x00\x8a\x00\x93\x06\xa1\x00\xae\x00\xb7\x01\xbf\x07\xc4\x00"
	} else {
		renvoAsmEmitText(a, "\xe9\xac\x00\x00\x00\x83\xfb\x00\x0f\x84\x05\x00\x00\x00\xe9\x0a\x00\x00\x00\x6a\xf6\xff\x15\x00\x00\x00\x00\x89\xc3\x83\xf9\x00\x0f\x8c\x49\x00\x00\x00\x51\x52\x6a\x01\x6a\x00\x6a\x00\x53\xff\x15\x00\x00\x00\x00\xa3\x00\x00\x00\x00\x5a\x59\x51\x52\x6a\x00\x6a\x00\x51\x53\xff\x15\x00\x00\x00\x00\x5a\x59\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x2d\x00\x00\x00\xa1\x00\x00\x00\x00\xe9\x26\x00\x00\x00\x6a\x00\x68\x00\x00\x00\x00\x52\x56\x53\xff\x15\x00\x00\x00\x00\x83\xf8\x00\x0f\x84\x06\x00\x00\x00\xa1\x00\x00\x00\x00\xc3\x6a\xff\x58\xc3\x6a\xff\x58\xa3\x00\x00\x00\x00\x6a\x00\x6a\x00\xa1\x00\x00\x00\x00\x50\x53\xff\x15\x00\x00\x00\x00\xa1\x00\x00\x00\x00\xc3")
		relocs = "\x17\x08\x31\x07\x36\x01\x46\x07\x4f\x00\x58\x05\x66\x00\x72\x00\x7b\x05\x89\x00\x96\x00\x9f\x01\xa7\x07\xac\x00"
	}
	for i := 0; i < len(relocs); i += 2 {
		at := base + int(relocs[i])
		kind := int(relocs[i+1])
		if kind < 2 {
			renvoAsmAddAbsReloc(a, at, countOff+(kind<<3), renvoAbsBssReloc)
		} else {
			renvoAsmAddWinImportReloc(a, at, kind-2)
		}
	}
	a.labelPos[label] = int32(base + 5)
	return label
}

func renvoWin386SetStdHandle(a *renvoAsm, stdHandle int) {
	renvoNonNil(a)
	renvoAsmPushImm(a, stdHandle)
	renvoWin386CallImport(a, renvoWinImportGetStdHandle)
	renvoAsmCopyPrimaryToCallWord0(a)
}

func renvoWin386EmitKernelReadWriteCall(a *renvoAsm, importID int, countOff int) {
	renvoNonNil(a)
	renvoAsmPushImm(a, 0)
	renvoWin386PushBssAddr(a, countOff)
	renvoAsmPushSecondary(a)
	renvoAsmEmit8(a, 0x56)
	renvoAsmEmit8(a, 0x53)
	renvoWin386CallImport(a, importID)
}

func renvoWinAmd64TranslateCreateFileFlags(a *renvoAsm) {
	renvoNonNil(a)
	notRDWRLabel := renvoAsmNewLabel(a)
	accessDoneLabel := renvoAsmNewLabel(a)
	noCreateLabel := renvoAsmNewLabel(a)
	createDoneLabel := renvoAsmNewLabel(a)

	renvoAsmSecondaryImm(a, -2147483648)
	renvoAsmEmit2(a, 0xa8, 2)
	renvoAsmJzLabel(a, notRDWRLabel)
	renvoAsmSecondaryImm(a, -1073741824)
	renvoAsmJmpMarkLabel(a, accessDoneLabel, notRDWRLabel)
	renvoAsmEmit2(a, 0xa8, 1)
	renvoAsmJzLabel(a, accessDoneLabel)
	renvoAsmSecondaryImm(a, 0x40000000)
	renvoAsmMarkLabel(a, accessDoneLabel)

	renvoWinAmd64MovR10Imm(a, 3)
	renvoAsmEmit2(a, 0xa8, 64)
	renvoAsmJzLabel(a, noCreateLabel)
	renvoWinAmd64MovR10Imm(a, 4)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoWinAmd64MovR10Imm(a, 2)
	renvoAsmJmpMarkLabel(a, createDoneLabel, noCreateLabel)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoWinAmd64MovR10Imm(a, 5)
	renvoAsmMarkLabel(a, createDoneLabel)
	renvoAsmEmit8(a, 0x41)
	renvoAsmEmit8(a, 0xb8)
	renvoAsmEmit32(a, 3)
	renvoAsmEmit24(a, 0xc93145)
}

func renvoWinAmd64MovR10Imm(a *renvoAsm, imm int) {
	renvoNonNil(a)
	renvoAsmEmit8(a, 0x41)
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, imm)
}

func renvoWin386TranslateCreateFileFlags(a *renvoAsm) {
	renvoNonNil(a)
	notRDWRLabel := renvoAsmNewLabel(a)
	accessDoneLabel := renvoAsmNewLabel(a)
	noCreateLabel := renvoAsmNewLabel(a)
	createDoneLabel := renvoAsmNewLabel(a)

	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, -2147483648)
	renvoAsmEmit2(a, 0xa8, 2)
	renvoAsmJzLabel(a, notRDWRLabel)
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, -1073741824)
	renvoAsmJmpMarkLabel(a, accessDoneLabel, notRDWRLabel)
	renvoAsmEmit2(a, 0xa8, 1)
	renvoAsmJzLabel(a, accessDoneLabel)
	renvoAsmEmit8(a, 0xba)
	renvoAsmEmit32(a, 0x40000000)
	renvoAsmMarkLabel(a, accessDoneLabel)

	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 3)
	renvoAsmEmit2(a, 0xa8, 64)
	renvoAsmJzLabel(a, noCreateLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 4)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 2)
	renvoAsmJmpMarkLabel(a, createDoneLabel, noCreateLabel)
	renvoAsmEmit8(a, 0xa9)
	renvoAsmEmit32(a, 512)
	renvoAsmJzLabel(a, createDoneLabel)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 5)
	renvoAsmMarkLabel(a, createDoneLabel)
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
func renvoEmitWindowsReadWrite(g *renvoLinearGen, ep *renvoExprParse, idx int, isWrite bool) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	p := g.prog
	firstArg := ep.exprs[idx].firstArg
	argCount := ep.exprs[idx].argCount
	if argCount != 3 {
		return false
	}
	fdStart := ep.exprs[idx].tok + 1
	fdEnd := renvoFindExprBoundary(p, fdStart, ep.end)
	fdEp := renvoNewExprParse()
	renvoParseExpressionInto(fdEp, p, fdStart, fdEnd)
	if !fdEp.ok || len(fdEp.exprs) == 0 {
		return false
	}
	if !renvoEmitIntExpr(g, fdEp, len(fdEp.exprs)-1) {
		return false
	}
	renvoAsmPushPrimary(a)
	offIndex := ep.args[firstArg+2]
	offConst := renvoEvalConstExpr(g, ep, offIndex)
	if offConst.ok && offConst.value < 0 {
		renvoAsmPrimaryImm(a, -1)
	} else if offConst.ok {
		renvoAsmPrimaryImm(a, offConst.value)
	} else {
		if !renvoEmitIntExpr(g, ep, offIndex) {
			return false
		}
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitSlicePtrLen(g, ep, ep.args[firstArg+1]) {
		return false
	}
	if g.c.renvoTargetArch == renvoArch386 {
		label := renvoWin386EmitReadWriteHelper(g, isWrite)
		renvoAsmEmit16(a, 0xc689)
		renvoAsmEmit16(a, 0xca89)
		renvoAsmPopTertiary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		label := renvoWinArm64EmitReadWriteHelper(g, isWrite)
		renvoAsmCopyPrimaryToCallWord1(a)
		renvoAarch64AsmMovRegReg(a, renvoAarch64RegRdx, renvoAarch64RegRcx)
		renvoAsmPopTertiary(a)
		renvoAsmPopCallWord0(a)
		renvoAsmCallLabel(a, label)
		return true
	}
	label := renvoWinAmd64EmitReadWriteHelper(g, isWrite)
	renvoAsmCopyPrimaryToCallWord1(a)
	renvoAsmEmit24(a, 0xca8948)
	renvoAsmPopTertiary(a)
	renvoAsmPopCallWord0(a)
	renvoAsmCallLabel(a, label)
	return true
}

func renvoEmitWindowsOpen(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmEmit16(a, 0xc689)
		renvoAsmPopPrimary(a)
		renvoWin386TranslateCreateFileFlags(a)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 0x80)
		renvoAsmPushTertiary(a)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 3)
		renvoAsmPushSecondary(a)
		renvoAsmEmit8(a, 0x56)
		renvoWin386CallImport(a, renvoWinImportCreateFileA)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAarch64AsmMovRegReg(a, 9, 0)
		renvoAsmPopPrimary(a)
		renvoWinArm64TranslateCreateFileFlags(a)
		renvoAarch64AsmMovRegReg(a, 0, 9)
		renvoWinArm64CallImport(a, renvoWinImportCreateFileA)
		return true
	}
	renvoAsmPushPrimary(a)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmPopTertiary(a)
	renvoAsmPopPrimary(a)
	renvoWinAmd64TranslateCreateFileFlags(a)
	// CreateFileA adds three stack arguments after its shadow space. As with
	// the simpler import helper, dynamically align a fresh call area and
	// restore the exact expression stack afterwards.
	base := len(a.code)
	renvoAsmEmitText(a, "\x41\x54\x49\x89\xe4\x48\x83\xe4\xf0\x48\x83\xec\x40\x44\x89\x54\x24\x20\xc7\x44\x24\x28\x80\x00\x00\x00\x48\xc7\x44\x24\x30\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x49\x89\xc3\x4c\x89\xe4\x41\x5c\x4c\x89\xd8")
	renvoAsmAddWinImportReloc(a, base+37, renvoWinImportCreateFileA)
	return true
}

func renvoEmitWindowsClose(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	failLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportCloseHandle)
		renvoAsmEmit3(a, 0x83, 0xf8, 0)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoWinArm64CallImport(a, renvoWinImportCloseHandle)
		renvoAsmCmpPrimaryImm8(a, 0)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	renvoAsmCopyPrimaryToTertiary(a)
	renvoWinAmd64CallImport(a, renvoWinImportCloseHandle)
	renvoAsmEmit3(a, 0x83, 0xf8, 0)
	renvoAsmJzLabel(a, failLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
	renvoAsmPrimaryImm(a, -1)
	renvoAsmMarkLabel(a, doneLabel)
	return true
}

func renvoEmitWindowsChmod(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	renvoNonNil(g, ep)
	a := &g.asm
	e := ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg]) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	failLabel := renvoAsmNewLabel(a)
	doneLabel := renvoAsmNewLabel(a)
	if g.c.renvoTargetArch == renvoArch386 {
		renvoAsmPopPrimary(a)
		renvoAsmPushImm(a, 1)
		renvoAsmPushImm(a, 0)
		renvoAsmPushImm(a, 0)
		renvoAsmPushPrimary(a)
		renvoWin386CallImport(a, renvoWinImportSetFilePointer)
		renvoAsmEmit3(a, 0x83, 0xf8, -1)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	if g.c.renvoTargetArch == renvoArchAarch64 {
		renvoAsmPopPrimary(a)
		renvoAarch64AsmMovRegImm(a, 1, 0)
		renvoAarch64AsmMovRegImm(a, 2, 0)
		renvoAarch64AsmMovRegImm(a, 3, 1)
		renvoWinArm64CallImport(a, renvoWinImportSetFilePointer)
		renvoAarch64AsmCmpRegImm(a, 0, -1)
		renvoAsmJzLabel(a, failLabel)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
		renvoAsmPrimaryImm(a, -1)
		renvoAsmMarkLabel(a, doneLabel)
		return true
	}
	renvoAsmPopTertiary(a)
	renvoAsmEmit16(a, 0xd231)
	renvoAsmEmit8(a, 0x41)
	renvoAsmEmit8(a, 0xb9)
	renvoAsmEmit32(a, 1)
	renvoAsmEmit24(a, 0xc03145)
	renvoWinAmd64CallImport(a, renvoWinImportSetFilePointer)
	renvoAsmEmit3(a, 0x83, 0xf8, -1)
	renvoAsmJzLabel(a, failLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmJmpMarkLabel(a, doneLabel, failLabel)
	renvoAsmPrimaryImm(a, -1)
	renvoAsmMarkLabel(a, doneLabel)
	return true
}
