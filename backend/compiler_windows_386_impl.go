package main

func compileWindows386(input []int, output int) int {
	return compileWindows386Arena(input, output, 0)
}

func compileWindows386Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWindows386)
	var src []byte
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if !prog.ok {
		return 1
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgram386Scratch(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
		return 0
	}
	renvoPrintErr("renvo: compilation failed\n")
	return 1
}

func renvoAsmBuildWindowsArgvEnvSlices386(a *renvoAsm, bssOff int, argsTextOff int, argsLenOff int, envOff int, envLenOff int) {
	base := len(a.code)
	renvoAsmEmitText(a, "\xff\x15\x00\x00\x00\x00\x89\xc6\xbf\x00\x00\x00\x00\xb8\x00\x00\x00\x00\x89\xc2\x31\xdb\x80\x3e\x00\x0f\x84\x75\x00\x00\x00\x80\x3e\x20\x0f\x84\x09\x00\x00\x00\x80\x3e\x09\x0f\x85\x06\x00\x00\x00\x46\xe9\xdf\xff\xff\xff\x89\x17\x31\xc9\x31\xed\x8a\x06\x3c\x00\x0f\x84\x34\x00\x00\x00\x3c\x22\x0f\x85\x09\x00\x00\x00\x83\xf5\x01\x46\xe9\xe5\xff\xff\xff\x83\xfd\x00\x0f\x85\x10\x00\x00\x00\x3c\x20\x0f\x84\x12\x00\x00\x00\x3c\x09\x0f\x84\x0a\x00\x00\x00\x88\x02\x46\x42\x41\xe9\xc2\xff\xff\xff\xc6\x02\x00\x42\x89\x4f\x08\x83\xc7\x10\x43\x3c\x00\x0f\x84\x06\x00\x00\x00\x46\xe9\x82\xff\xff\xff\x89\xd8\xa3\x00\x00\x00\x00\xff\x15\x00\x00\x00\x00\x89\xc6\xbf\x00\x00\x00\x00\x31\xdb\x80\x3e\x00\x0f\x84\x23\x00\x00\x00\x89\x37\x31\xc9\x80\x3c\x0e\x00\x0f\x84\x06\x00\x00\x00\x41\xe9\xf0\xff\xff\xff\x89\x4f\x08\x01\xce\x46\x83\xc7\x10\x43\xe9\xd4\xff\xff\xff\x89\xd8\xa3\x00\x00\x00\x00\xbb\x00\x00\x00\x00\x8b\x35\x00\x00\x00\x00\x89\xf2\xb9\x00\x00\x00\x00\xa1\x00\x00\x00\x00\x89\xc7")
	renvoAsmAddWinImportReloc(a, base+2, renvoWinImportGetCommandLineA)
	renvoAsmAddAbsReloc(a, base+9, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+14, argsTextOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+151, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddWinImportReloc(a, base+157, renvoWinImportGetEnvironmentStringsA)
	renvoAsmAddAbsReloc(a, base+164, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+217, envLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+222, bssOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+228, argsLenOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+235, envOff, renvoAbsBssReloc)
	renvoAsmAddAbsReloc(a, base+240, envLenOff, renvoAbsBssReloc)
}

func renvoAsmImageWindows386(a *renvoAsm) []byte {
	for (a.codeOffset+len(a.code))%4 != 0 {
		a.code = append(a.code, 0)
	}
	textVirtualSize := len(a.code)
	textRawSize := renvoAlignValue(textVirtualSize, renvoWinFileAlign)
	dataRVA := renvoAlignValue(a.codeOffset+textVirtualSize, renvoWinSectionAlign)
	a.dataOffset = dataRVA
	var imports renvoWinImportLayout
	if renvoAsmHasWinImportRelocs(a) {
		renvoAppendWinImports(a, &imports)
	}
	renvoAsmPatchWindows(a, imports)
	dataRawSize := renvoAlignValue(len(a.data), renvoWinFileAlign)
	dataVirtualSize := len(a.data) + a.bssSize
	iatSize := 0
	if imports.kernelIATRVA != 0 {
		iatSize = (renvoWinImportFixedCount + 1) * imports.thunkSize
	}
	var out []byte
	out = renvoAppendPEHeader32(out, a.codeOffset, textRawSize, textVirtualSize, dataRVA, dataRawSize, dataVirtualSize, imports.importRVA, imports.importSize, imports.kernelIATRVA, iatSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize)
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	out = renvoAppendUntil(out, renvoWinHeadersSize+textRawSize+dataRawSize)
	if renvoFixedTarget == 0 && renvoCompilerEmitImage {
		for i := 0; i+2 < len(a.absRelocs); i += 3 {
			at := int(renvo_runtime_UnsafeInt32At(a.absRelocs, i)) & 2147483647
			out = renvoAppend32(out, a.codeOffset+at)
		}
		out = renvoAppend32(out, len(a.absRelocs)/3)
		out = append(out, 'R', 'B', 'R', '1')
		return renvoAppendReplLinkTable(out, a)
	}
	return out
}
