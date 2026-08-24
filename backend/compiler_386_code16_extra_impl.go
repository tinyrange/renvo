package main

func renvo386Code16LocalSize(g *renvoLinearGen, typ int, size int) int {
	if g.c.code16 && renvoProgramUsesC11Semantics(g.prog) && renvoTypeSize(g.meta, typ) <= 4 {
		kind := renvoResolveType(g.meta, typ).kind
		if renvoTypeKindIsScalarInt(kind) || kind == renvoTypePointer || kind == renvoTypeFunc {
			return 4
		}
	}
	if size < renvoBackendValueSlotSize {
		return renvoBackendValueSlotSize
	}
	return size
}

func renvo386EmitWideIdentToLocal(g *renvoLinearGen, e *renvoExpr, offset int) bool {
	if e.kind != renvoExprIdent {
		return false
	}
	localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
	if localIndex < 0 {
		return false
	}
	if g.locals[localIndex].offset != offset {
		renvoEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, renvoBackendValueSlotSize)
	}
	return true
}

// renvo386EmitWideIntExprFast returns -1 when the ordinary lowering should run,
// zero on an emission failure, and one when it emitted the expression.
func renvo386EmitWideIntExprFast(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 && renvo386TryLoadInlineLocal(g, localIndex) {
			kind := renvoResolveType(g.meta, g.locals[localIndex].typ).kind
			renvoAsmNormalizePrimaryForKind(&g.asm, kind)
			return 1
		}
		return -1
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		return renvoEmitCDirectDeref(g, ep, idx)
	}
	if e.kind != renvoExprBinary {
		return -1
	}
	right := &ep.exprs[e.right]
	if renvo386EmitWideImmediateBinary(g, ep, idx, e, right) {
		return 1
	}
	if right.kind != renvoExprIdent {
		return -1
	}
	rightLocal := renvoFindLocalIndex(g, right.nameStart, right.nameEnd)
	memoryOpcode := 0
	memoryMultiply := false
	if renvoTokCharIs(g.prog, e.tok, '+') {
		memoryOpcode = 0x03
	} else if renvoTokCharIs(g.prog, e.tok, '-') {
		memoryOpcode = 0x2b
	} else if renvoTokCharIs(g.prog, e.tok, '*') {
		memoryMultiply = true
	} else if renvoTokCharIs(g.prog, e.tok, '&') {
		memoryOpcode = 0x23
	} else if renvoTokCharIs(g.prog, e.tok, '|') {
		memoryOpcode = 0x0b
	} else if renvoTokCharIs(g.prog, e.tok, '^') {
		memoryOpcode = 0x33
	}
	if rightLocal < 0 || renvoTypeSize(g.meta, g.locals[rightLocal].typ) != 4 || memoryOpcode == 0 && !memoryMultiply {
		return -1
	}
	if !renvoEmitIntExpr(g, ep, e.left) {
		return 0
	}
	if memoryMultiply {
		renvoAsmEmit8(&g.asm, 0x0f)
		renvoAsmStackMem(&g.asm, g.locals[rightLocal].offset, 0xaf, 0x45, 0x85)
	} else {
		renvoAsmStackMem(&g.asm, g.locals[rightLocal].offset, memoryOpcode, 0x45, 0x85)
	}
	resultType := renvoInferParsedExprType(g, ep, idx)
	result := renvoResolveType(g.meta, resultType)
	renvoAsmNormalizePrimaryForKind(&g.asm, result.kind)
	return 1
}

func renvo386EmitDirectCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) bool {
	if !renvo386TryLoadCallWordsFromFrame(&g.asm, wordCount) {
		return false
	}
	renvoAsmCallLabel(&g.asm, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if renvoAsmImmFits8Signed(imm) {
			renvoAsmEmit3(&g.asm, 0x83, 0xc4, imm)
		} else {
			renvoAsmEmit16(&g.asm, 0xc481)
			renvoAsmEmit32(&g.asm, imm)
		}
	}
	return true
}

// renvo386EmitNativeIntExprFast returns -1 when the ordinary lowering should
// run, zero on an emission failure, and one when it emitted the expression.
func renvo386EmitNativeIntExprFast(g *renvoLinearGen, ep *renvoExprParse, idx int) int {
	e := &ep.exprs[idx]
	if e.kind == renvoExprIdent {
		localIndex := renvoFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 && renvo386TryLoadInlineLocal(g, localIndex) {
			kind := renvoResolveType(g.meta, g.locals[localIndex].typ).kind
			renvoAsmNormalizePrimaryForKind(&g.asm, kind)
			return 1
		}
		return -1
	}
	if e.kind == renvoExprUnary && renvoTokCharIs(g.prog, e.tok, '*') {
		return renvoEmitCDirectDeref(g, ep, idx)
	}
	if e.kind != renvoExprBinary {
		return -1
	}
	p := g.prog
	right := &ep.exprs[e.right]
	value := 0
	immediate := false
	if right.kind == renvoExprInt {
		value = renvoParseIntToken(p, right.tok)
		immediate = true
	} else if right.kind == renvoExprChar {
		value = renvoParseCharToken(p, right.tok)
		immediate = true
	} else if right.kind == renvoExprIdent {
		value = renvoFindSmallConstByName(g, right.nameStart, right.nameEnd)
		immediate = value >= -128
	}
	if p.compilerInt32 && immediate {
		immediate = renvoEvalConstExpr(g, ep, e.right).ok
	}
	immOpcode := 0
	immGroup := 0
	immMultiply := false
	immShift := 0
	if renvoTokCharIs(p, e.tok, '+') {
		immOpcode, immGroup = 0x05, 0xc0
	} else if renvoTokCharIs(p, e.tok, '-') {
		immOpcode, immGroup = 0x2d, 0xe8
	} else if renvoTokCharIs(p, e.tok, '*') {
		immMultiply = true
	} else if renvoTokCharIs(p, e.tok, '&') {
		immOpcode, immGroup = 0x25, 0xe0
	} else if renvoTokCharIs(p, e.tok, '|') {
		immOpcode, immGroup = 0x0d, 0xc8
	} else if renvoTokCharIs(p, e.tok, '^') {
		immOpcode, immGroup = 0x35, 0xf0
	} else if renvoTok2Is(p, e.tok, '<', '<') {
		immShift = 0xe0
	} else if renvoTok2Is(p, e.tok, '>', '>') {
		immShift = 0xf8
		if renvoExprHasUnsignedIntType(g, ep, e.left) {
			immShift = 0xe8
		}
	}
	if !immediate || immOpcode == 0 && !immMultiply && immShift == 0 || value < -2147483647 || value > 2147483647 ||
		immShift != 0 && (value < 0 || value >= 32) {
		return -1
	}
	if renvo386EmitNativeImmediateBinary(g, ep, idx, e.left, value, immOpcode, immGroup, immMultiply, immShift) {
		return 1
	}
	return 0
}
