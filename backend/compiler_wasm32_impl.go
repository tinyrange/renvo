package main

func renvoWasmAppendU32(out []byte, v int) []byte {
	for i := 0; i < 5; i++ {
		b := byte(v & 0x7f)
		v = v >> 7
		if i == 4 {
			b = b & 0x0f
			out = append(out, b)
			return out
		}
		if v == 0 {
			out = append(out, b)
			return out
		}
		b = b | 0x80
		out = append(out, b)
	}
	return out
}

func renvoWasmAppendS32(out []byte, v int) []byte {
	for {
		b := byte(v & 0x7f)
		signSet := (b & 0x40) != 0
		v = v >> 7
		done := false
		if v == 0 && !signSet {
			done = true
		}
		if v == -1 && signSet {
			done = true
		}
		if !done {
			b = b | 0x80
		}
		out = append(out, b)
		if done {
			return out
		}
	}
}

func renvoWasmAppendName(out []byte, name string) []byte {
	out = renvoWasmAppendU32(out, len(name))
	for i := 0; i < len(name); i++ {
		out = append(out, name[i])
	}
	return out
}

func renvoWasmAppendByteVec(out []byte, data []byte) []byte {
	out = renvoWasmAppendU32(out, len(data))
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}

func renvoWasmAppendEncoded(out []byte, encoded string) []byte {
	for i := 0; i < len(encoded); i++ {
		out = append(out, encoded[i])
	}
	return out
}

func renvoWasmAppendRecipe(out []byte, recipe string, p0 int, p1 int, p2 int, p3 int) []byte {
	for i := 0; i < len(recipe); i++ {
		if recipe[i] != 0xff {
			out = append(out, recipe[i])
			continue
		}
		command := int(recipe[i+1])
		parameter := int(recipe[i+2])
		value := p0
		if parameter == 1 {
			value = p1
		} else if parameter == 2 {
			value = p2
		} else if parameter == 3 {
			value = p3
		}
		if command == 0 {
			out = renvoWasmAppendI32Const(out, value)
		} else if command == 1 {
			out = renvoWasmLocalGet(out, value)
		} else if command == 2 {
			out = renvoWasmLocalSet(out, value)
		} else if command == 3 {
			out = renvoWasmLocalTee(out, value)
		} else if command == 4 {
			out = renvoWasm32RegGet(out, value)
		} else if command == 5 {
			out = renvoWasm32RegSet(out, value)
		} else if command == 6 {
			out = renvoWasm32StackAddr(out, value)
		} else if command == 7 {
			out = renvoWasm32LoadSized(out, value)
		} else if command == 8 {
			out = renvoWasm32StoreSized(out, value)
		} else if command == 9 {
			out = renvoWasm32AppendCond(out, value)
		} else if command == 10 {
			out = renvoWasm32AppendBinaryOp(out, value)
		}
		i += 2
	}
	return out
}

func renvoWasmAppendSection(out []byte, id int, payload []byte) []byte {
	out = append(out, byte(id))
	out = renvoWasmAppendU32(out, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func renvoWasmAppendI32Const(out []byte, value int) []byte {
	out = append(out, 0x41)
	value = renvoWasm32Sign32(value)
	out = renvoWasmAppendS32(out, value)
	return out
}

func renvoWasmAppendCall(out []byte, index int) []byte {
	out = append(out, 0x10)
	out = renvoWasmAppendU32(out, index)
	return out
}

func renvoWasmAppendU32Fixed5(out []byte, v int) []byte {
	for i := 0; i < 5; i++ {
		b := byte(v & 0x7f)
		v = v >> 7
		if i < 4 {
			b = b | 0x80
		} else {
			b = b & 0x0f
		}
		out = append(out, b)
	}
	return out
}

func renvoWasmPatchU32Fixed5(out []byte, at int, v int) {
	for i := 0; i < 5; i++ {
		b := byte(v & 0x7f)
		v = v >> 7
		if i < 4 {
			b = b | 0x80
		} else {
			b = b & 0x0f
		}
		out[at+i] = b
	}
}

func renvoWasmAppendI32Store(out []byte) []byte {
	out = append(out, 0x36)
	out = renvoWasmAppendU32(out, 2)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasmAppend2(out []byte, a int, b int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	return out
}

func renvoWasmAppend3(out []byte, a int, b int, c int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	return out
}

func renvoWasmAppend4(out []byte, a int, b int, c int, d int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	out = append(out, byte(d))
	return out
}

func renvoWasmAppend5(out []byte, a int, b int, c int, d int, e int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	out = append(out, byte(d))
	out = append(out, byte(e))
	return out
}

const renvoWasm32RegRax = 0
const renvoWasm32RegRdx = 1
const renvoWasm32RegRcx = 2
const renvoWasm32RegRdi = 3
const renvoWasm32RegRsi = 4
const renvoWasm32RegR8 = 5
const renvoWasm32RegR9 = 6
const renvoWasm32RegR10 = 7

const renvoWasm32OpExit = 1
const renvoWasm32OpBuildArgsEnv = 2
const renvoWasm32OpMovRegImm = 3
const renvoWasm32OpMovRegReg = 4
const renvoWasm32OpPushReg = 5
const renvoWasm32OpPushImm = 6
const renvoWasm32OpPopReg = 7
const renvoWasm32OpLoadStack = 8
const renvoWasm32OpStoreStack = 9
const renvoWasm32OpLeaStack = 10
const renvoWasm32OpLoadMem = 11
const renvoWasm32OpStoreMem = 12
const renvoWasm32OpLoadIndex = 13
const renvoWasm32OpStoreIndex = 14
const renvoWasm32OpAddRegReg = 15
const renvoWasm32OpSubRegReg = 16
const renvoWasm32OpMulRegReg = 17
const renvoWasm32OpDivRegReg = 18
const renvoWasm32OpModRegReg = 19
const renvoWasm32OpAndRegReg = 20
const renvoWasm32OpOrRegReg = 21
const renvoWasm32OpXorRegReg = 22
const renvoWasm32OpAndNotRegReg = 23
const renvoWasm32OpShlRegReg = 24
const renvoWasm32OpShrRegReg = 25
const renvoWasm32OpAddRegImm = 26
const renvoWasm32OpMulRegImm = 27
const renvoWasm32OpIncReg = 28
const renvoWasm32OpIncMem = 29
const renvoWasm32OpDecMem = 30
const renvoWasm32OpBoolNot = 31
const renvoWasm32OpNegReg = 32
const renvoWasm32OpCmpRegImm = 33
const renvoWasm32OpCmpRegReg = 34
const renvoWasm32OpSetCond = 35
const renvoWasm32OpJmp = 36
const renvoWasm32OpJz = 37
const renvoWasm32OpJnz = 38
const renvoWasm32OpJCond = 39
const renvoWasm32OpCall = 40
const renvoWasm32OpRet = 41
const renvoWasm32OpSyscall = 42
const renvoWasm32OpNop = 43

const renvoWasm32CondEq = 0
const renvoWasm32CondNe = 1
const renvoWasm32CondLt = 2
const renvoWasm32CondLe = 3
const renvoWasm32CondGt = 4
const renvoWasm32CondGe = 5

func renvoWasm32EmitRegImm(a *renvoAsm, op int, reg int, imm int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = renvoAppend32(a.code, imm)
}

func renvoWasm32EmitRegReg(a *renvoAsm, op int, dst int, src int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(dst))
	a.code = append(a.code, byte(src))
}

func renvoWasm32EmitReg(a *renvoAsm, op int, reg int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
}

func renvoWasm32EmitStack(a *renvoAsm, op int, reg int, offset int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = renvoAppend32(a.code, offset)
}

func renvoWasm32EmitMem(a *renvoAsm, op int, reg int, base int, disp int, size int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = append(a.code, byte(base))
	a.code = renvoAppend32(a.code, disp)
	a.code = append(a.code, byte(size))
}

func renvoWasm32EmitIndex(a *renvoAsm, op int, reg int, base int, index int, scale int, disp int, size int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = append(a.code, byte(base))
	a.code = append(a.code, byte(index))
	a.code = append(a.code, byte(scale))
	a.code = renvoAppend32(a.code, disp)
	a.code = append(a.code, byte(size))
}

func renvoWasm32EmitBranch(a *renvoAsm, op int, label int) {
	a.code = append(a.code, byte(op))
	at := len(a.code)
	a.code = renvoAppend32(a.code, 0)
	renvoAsmAddReloc(a, at, label)
}

func renvoWasm32EmitCondBranch(a *renvoAsm, cond int, label int) {
	a.code = append(a.code, byte(renvoWasm32OpJCond))
	a.code = append(a.code, byte(cond))
	at := len(a.code)
	a.code = renvoAppend32(a.code, 0)
	renvoAsmAddReloc(a, at, label)
}

func renvoWasm32EmitCallLabel(a *renvoAsm, label int, wordCount int) {
	a.code = append(a.code, byte(renvoWasm32OpCall))
	at := len(a.code)
	a.code = renvoAppend32(a.code, 0)
	renvoAsmAddReloc(a, at, label)
	a.code = renvoAppend32(a.code, wordCount)
}

func renvoWasm32StoreParamWord(g *renvoLinearGen, reg int, offset int) {
	a := &g.asm
	if reg == 0 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdi, offset)
		return
	}
	if reg == 1 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRsi, offset)
		return
	}
	if reg == 2 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdx, offset)
		return
	}
	if reg == 3 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRcx, offset)
		return
	}
	if reg == 4 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegR8, offset)
		return
	}
	if reg == 5 {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegR9, offset)
		return
	}
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRax)
	renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRax, offset)
}

func renvoWasm32AsmMovRaxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, imm)
}

func renvoWasm32AsmMovRaxImm64(a *renvoAsm, imm int) {
	renvoWasm32AsmMovRaxImm(a, imm)
}

func renvoWasm32AsmMovRdxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, imm)
}

func renvoWasm32AsmMovRaxDataAddr(a *renvoAsm, dataOff int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, 0)
	renvoAsmAddAbsReloc(a, len(a.code)-4, dataOff, 0)
}

func renvoWasm32AsmMovRaxBssAddr(a *renvoAsm, bssOff int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRax, 0)
	renvoAsmAddAbsReloc(a, len(a.code)-4, bssOff, renvoAbsBssReloc)
}

func renvoWasm32AsmMovR10BssAddr(a *renvoAsm, bssOff int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegR10, 0)
	renvoAsmAddAbsReloc(a, len(a.code)-4, bssOff, renvoAbsBssReloc)
}

func renvoWasm32AsmLoadRaxBss(a *renvoAsm, bssOff int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegR10, 0)
	renvoAsmAddAbsReloc(a, len(a.code)-4, bssOff, renvoAbsBssReloc)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegR10, 0, 4)
}

func renvoWasm32AsmStoreRaxBss(a *renvoAsm, bssOff int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegR10, 0)
	renvoAsmAddAbsReloc(a, len(a.code)-4, bssOff, renvoAbsBssReloc)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRax, renvoWasm32RegR10, 0, 4)
}

func renvoWasm32AsmMovRdiRax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdi, renvoWasm32RegRax)
}

func renvoWasm32AsmMovRaxRdx(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
}

func renvoWasm32AsmMovRdxRax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
}

func renvoWasm32AsmMovRcxRax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
}

func renvoWasm32AsmMovRcxRdx(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRcx, renvoWasm32RegRdx)
}

func renvoWasm32AsmMovRsiRax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRsi, renvoWasm32RegRax)
}

func renvoWasm32AsmMovR8Rax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegR8, renvoWasm32RegRax)
}

func renvoWasm32AsmMovR9Rax(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegR9, renvoWasm32RegRax)
}

func renvoWasm32AsmAddRdxRcx(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpAddRegReg, renvoWasm32RegRdx, renvoWasm32RegRcx)
}

func renvoWasm32AsmSyscall(a *renvoAsm) {
	renvoAsmEmit8(a, renvoWasm32OpSyscall)
}

func renvoWasm32AsmPushRax(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRax)
}

func renvoWasm32AsmPushRcx(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRcx)
}

func renvoWasm32AsmPushRdx(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPushReg, renvoWasm32RegRdx)
}

func renvoWasm32AsmPushImm(a *renvoAsm, imm int) {
	renvoAsmEmit8(a, renvoWasm32OpPushImm)
	renvoAsmEmit32(a, imm)
}

func renvoWasm32AsmPopRax(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRax)
}

func renvoWasm32AsmPopRcx(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRcx)
}

func renvoWasm32AsmPopRdx(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdx)
}

func renvoWasm32AsmPopRdi(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRdi)
}

func renvoWasm32AsmPopRsi(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegRsi)
}

func renvoWasm32AsmStackMem(a *renvoAsm, offset int, base int, disp8 int, disp32 int) {
	op := base & 0xff00
	regCode := disp8
	reg := renvoWasm32RegRax
	if base == 0x894c || base == 0x8b4c {
		if regCode == 0x4d || regCode == 0x8d {
			reg = renvoWasm32RegR9
		} else {
			reg = renvoWasm32RegR8
		}
	} else if regCode == 0x55 || regCode == 0x95 {
		reg = renvoWasm32RegRdx
	} else if regCode == 0x4d || regCode == 0x8d {
		reg = renvoWasm32RegRcx
	} else if regCode == 0x7d || regCode == 0xbd {
		reg = renvoWasm32RegRdi
	} else if regCode == 0x75 || regCode == 0xb5 {
		reg = renvoWasm32RegRsi
	}
	if op == 0x8900 || op == 0x8948 || op == 0x894c {
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, reg, offset)
		return
	}
	if op == 0x8b00 || op == 0x8b48 || op == 0x8b4c {
		renvoWasm32EmitStack(a, renvoWasm32OpLoadStack, reg, offset)
		return
	}
	renvoWasm32EmitStack(a, renvoWasm32OpLeaStack, reg, offset)
}

func renvoWasm32AsmAddRdxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRdx, imm)
}

func renvoWasm32AsmLoadRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, disp, 4)
}

func renvoWasm32AsmLoadRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	if size > 4 {
		size = 4
	}
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdx, disp, size)
}

func renvoWasm32AsmLoadByteRaxIndexRcx(a *renvoAsm) {
	renvoWasm32EmitIndex(a, renvoWasm32OpLoadIndex, renvoWasm32RegRax, renvoWasm32RegRax, renvoWasm32RegRcx, 1, 0, 1)
}

func renvoWasm32AsmLoadRaxIndexRcxSize(a *renvoAsm, size int) {
	scale := size
	if size > 4 {
		size = 4
	}
	renvoWasm32EmitIndex(a, renvoWasm32OpLoadIndex, renvoWasm32RegRax, renvoWasm32RegRax, renvoWasm32RegRcx, scale, 0, size)
}

func renvoWasm32AsmLoadQwordRaxIndexRcx8(a *renvoAsm) {
	renvoWasm32EmitIndex(a, renvoWasm32OpLoadIndex, renvoWasm32RegRax, renvoWasm32RegRax, renvoWasm32RegRcx, 8, 0, 4)
}

func renvoWasm32AsmLoadQwordRaxIndexRcxDisp(a *renvoAsm, disp int) {
	renvoWasm32EmitIndex(a, renvoWasm32OpLoadIndex, renvoWasm32RegRax, renvoWasm32RegRax, renvoWasm32RegRcx, 1, disp, 4)
}

func renvoWasm32AsmStoreRaxMemRdxRcx8(a *renvoAsm) {
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRax, renvoWasm32RegRdx, renvoWasm32RegRcx, 8, 0, 4)
}

func renvoWasm32AsmStoreRaxMemRdxDisp(a *renvoAsm, disp int) {
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRax, renvoWasm32RegRdx, disp, 4)
}

func renvoWasm32AsmStoreRaxMemRdxDispSize(a *renvoAsm, disp int, size int) {
	if size > 4 {
		size = 4
	}
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRax, renvoWasm32RegRdx, disp, size)
}

func renvoWasm32AsmStoreRaxMemRdxRcxSize(a *renvoAsm, size int) {
	scale := size
	if size > 4 {
		size = 4
	}
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRax, renvoWasm32RegRdx, renvoWasm32RegRcx, scale, 0, size)
}

func renvoWasm32AsmStoreSliceStack(a *renvoAsm, offset int) {
	renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRax, offset)
	renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdx, offset-8)
	renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRcx, offset-16)
}

func renvoWasm32AsmNormalizeRaxForKind(a *renvoAsm, kind int) {
	if kind == renvoTypeByte {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 255)
		renvoWasm32EmitRegReg(a, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		return
	}
	if kind == renvoTypeInt8 {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 24)
		renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 24)
		renvoWasm32EmitRegReg(a, renvoWasm32OpShrRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		return
	}
	if kind == renvoTypeInt16 {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 16)
		renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 16)
		renvoWasm32EmitRegReg(a, renvoWasm32OpShrRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		return
	}
	if kind == renvoTypeUint16 {
		renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, 65535)
		renvoWasm32EmitRegReg(a, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	}
}

func renvoWasm32AsmIncMemRdx(a *renvoAsm) {
	renvoAsmEmit8(a, renvoWasm32OpIncMem)
	renvoAsmEmit8(a, renvoWasm32RegRdx)
}

func renvoWasm32AsmDecMemRdx(a *renvoAsm) {
	renvoAsmEmit8(a, renvoWasm32OpDecMem)
	renvoAsmEmit8(a, renvoWasm32RegRdx)
}

func renvoWasm32AsmBoolNotRax(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpBoolNot, renvoWasm32RegRax)
}

func renvoWasm32AsmCmpRaxImm8(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpCmpRegImm, renvoWasm32RegRax, imm)
}

func renvoWasm32AsmAddRaxRcx(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpAddRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
}

func renvoWasm32AsmSubRaxRcx(a *renvoAsm) {
	renvoWasm32EmitRegReg(a, renvoWasm32OpSubRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
}

func renvoWasm32AsmShlRcxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, imm)
	renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRcx, renvoWasm32RegRdx)
}

func renvoWasm32AsmShlRaxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, imm)
	renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
}

func renvoWasm32AsmSarRaxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRdx, imm)
	renvoWasm32EmitRegReg(a, renvoWasm32OpShrRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
}

func renvoWasm32AsmDivLeftRcxRightRax(a *renvoAsm, mod bool) {
	if mod {
		renvoWasm32EmitRegReg(a, renvoWasm32OpModRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
	} else {
		renvoWasm32EmitRegReg(a, renvoWasm32OpDivRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
	}
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
}

func renvoWasm32AsmCmpRcxRaxSet(a *renvoAsm, setcc int) {
	cond := renvoWasm32CondEq
	if setcc == 0x95 {
		cond = renvoWasm32CondNe
	} else if setcc == 0x9c {
		cond = renvoWasm32CondLt
	} else if setcc == 0x9e {
		cond = renvoWasm32CondLe
	} else if setcc == 0x9f {
		cond = renvoWasm32CondGt
	} else if setcc == 0x9d {
		cond = renvoWasm32CondGe
	}
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
	renvoAsmEmit8(a, renvoWasm32OpSetCond)
	renvoAsmEmit8(a, cond)
}

func renvoWasm32AsmIncRcx(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpIncReg, renvoWasm32RegRcx)
}

func renvoWasm32AsmIncRax(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpIncReg, renvoWasm32RegRax)
}

func renvoWasm32AsmImulRcxImm(a *renvoAsm, imm int) {
	renvoWasm32EmitRegImm(a, renvoWasm32OpMulRegImm, renvoWasm32RegRcx, imm)
}

func renvoWasm32AsmLeave(a *renvoAsm) {
}

func renvoWasm32AsmRet(a *renvoAsm) {
	renvoAsmEmit8(a, renvoWasm32OpRet)
}

func renvoWasm32AsmCallLabel(a *renvoAsm, label int) {
	renvoWasm32EmitCallLabel(a, label, 0)
}

func renvoWasm32AsmJmpLabel(a *renvoAsm, label int) {
	renvoWasm32EmitBranch(a, renvoWasm32OpJmp, label)
}

func renvoWasm32AsmJzLabel(a *renvoAsm, label int) {
	renvoWasm32EmitBranch(a, renvoWasm32OpJz, label)
}

func renvoWasm32AsmJnzLabel(a *renvoAsm, label int) {
	renvoWasm32EmitBranch(a, renvoWasm32OpJnz, label)
}

func renvoWasm32AsmNegRax(a *renvoAsm) {
	renvoWasm32EmitReg(a, renvoWasm32OpNegReg, renvoWasm32RegRax)
}

func renvoWasm32AsmBuildArgvEnvSlices(a *renvoAsm, bssOff int, envOff int, envLenOff int) {
	renvoAsmEmit8(a, renvoWasm32OpBuildArgsEnv)
	at := len(a.code)
	renvoAsmEmit32(a, bssOff)
	renvoAsmAddAbsReloc(a, at, bssOff, renvoAbsBssReloc)
	at = len(a.code)
	renvoAsmEmit32(a, envOff)
	renvoAsmAddAbsReloc(a, at, envOff, renvoAbsBssReloc)
	at = len(a.code)
	renvoAsmEmit32(a, envLenOff)
	renvoAsmAddAbsReloc(a, at, envLenOff, renvoAbsBssReloc)
}

func renvoWasm32AsmExit(a *renvoAsm) {
	renvoAsmEmit8(a, renvoWasm32OpExit)
}

type renvoWasm32Instr struct {
	pc   int
	next int
	op   int
	a    int
	b    int
	c    int
	d    int
	e    int
	f    int
}

const renvoWasm32ProgramBase = 262144
const renvoWasm32StackGuardSize = 16384
const renvoWasm32ExprStackSize = 1048576
const renvoWasm32CallStackSize = 65536
const renvoWasm32FrameStackSize = 8388608
const renvoWasm32ImageOutputCapacity = 2097152
const renvoWasm32RoutineBodyCapacity = 8192
const renvoWasm32ScratchIov = 0
const renvoWasm32ScratchN = 8
const renvoWasm32ScratchFd = 12
const renvoWasm32ArgsCountPtr = 16
const renvoWasm32ArgsSizePtr = 20
const renvoWasm32EnvCountPtr = 24
const renvoWasm32EnvSizePtr = 28
const renvoWasm32FdstatScratch = 64
const renvoWasm32ArgsPtrArea = 4096
const renvoWasm32ArgsDataArea = 8192
const renvoWasm32EnvPtrArea = 65536
const renvoWasm32EnvDataArea = 131072

const renvoWasm32ImportFdWrite = 0
const renvoWasm32ImportFdRead = 1
const renvoWasm32ImportFdPread = 2
const renvoWasm32ImportFdPwrite = 3
const renvoWasm32ImportPathOpen = 4
const renvoWasm32ImportFdClose = 5
const renvoWasm32ImportFdstatGet = 6
const renvoWasm32ImportFdReaddir = 7
const renvoWasm32ImportArgsSizesGet = 8
const renvoWasm32ImportArgsGet = 9
const renvoWasm32ImportEnvironSizesGet = 10
const renvoWasm32ImportEnvironGet = 11
const renvoWasm32ImportProcExit = 12
const renvoWasm32StartFuncIndex = 13
const renvoWasm32VmFuncType = 7
const renvoWasm32VmFuncBase = 14

const renvoWasm32LocalSp = 0
const renvoWasm32LocalFp = 1
const renvoWasm32LocalRax = 2
const renvoWasm32LocalRdx = 3
const renvoWasm32LocalRcx = 4
const renvoWasm32LocalRdi = 5
const renvoWasm32LocalRsi = 6
const renvoWasm32LocalR8 = 7
const renvoWasm32LocalR9 = 8
const renvoWasm32LocalPc = 9
const renvoWasm32LocalCsp = 10
const renvoWasm32LocalR10 = 11
const renvoWasm32LocalFlag = 12
const renvoWasm32LocalTmp = 13
const renvoWasm32LocalTmp2 = 14
const renvoWasm32LocalTmp3 = 15

func renvoWasm32RegLocal(reg int) int {
	if reg == renvoWasm32RegRdx {
		return renvoWasm32LocalRdx
	}
	if reg == renvoWasm32RegRcx {
		return renvoWasm32LocalRcx
	}
	if reg == renvoWasm32RegRdi {
		return renvoWasm32LocalRdi
	}
	if reg == renvoWasm32RegRsi {
		return renvoWasm32LocalRsi
	}
	if reg == renvoWasm32RegR8 {
		return renvoWasm32LocalR8
	}
	if reg == renvoWasm32RegR9 {
		return renvoWasm32LocalR9
	}
	if reg == renvoWasm32RegR10 {
		return renvoWasm32LocalR10
	}
	return renvoWasm32LocalRax
}

func renvoWasm32Sign32(v int) int {
	if v > 2147483647 {
		v -= 2147483647
		v -= 2147483647
		v -= 2
	}
	return v
}

func renvoWasm32GetS32(in []byte, at int) int {
	return renvoWasm32Sign32(renvoGet32At(in, at))
}

func renvoWasm32NextInstructionPc(code []byte, pc int) int {
	op := int(code[pc])
	if op == renvoWasm32OpMovRegImm || op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack || op == renvoWasm32OpLeaStack || op == renvoWasm32OpAddRegImm || op == renvoWasm32OpMulRegImm || op == renvoWasm32OpCmpRegImm || op == renvoWasm32OpJCond {
		return pc + 6
	}
	if op == renvoWasm32OpMovRegReg || op == renvoWasm32OpAddRegReg || op == renvoWasm32OpSubRegReg || op == renvoWasm32OpMulRegReg || op == renvoWasm32OpDivRegReg || op == renvoWasm32OpModRegReg || op == renvoWasm32OpAndRegReg || op == renvoWasm32OpOrRegReg || op == renvoWasm32OpXorRegReg || op == renvoWasm32OpAndNotRegReg || op == renvoWasm32OpShlRegReg || op == renvoWasm32OpShrRegReg || op == renvoWasm32OpCmpRegReg {
		return pc + 3
	}
	if op == renvoWasm32OpPushReg || op == renvoWasm32OpPopReg || op == renvoWasm32OpIncReg || op == renvoWasm32OpIncMem || op == renvoWasm32OpDecMem || op == renvoWasm32OpBoolNot || op == renvoWasm32OpNegReg || op == renvoWasm32OpSetCond {
		return pc + 2
	}
	if op == renvoWasm32OpPushImm || op == renvoWasm32OpJmp || op == renvoWasm32OpJz || op == renvoWasm32OpJnz {
		return pc + 5
	}
	if op == renvoWasm32OpLoadMem || op == renvoWasm32OpStoreMem {
		return pc + 8
	}
	if op == renvoWasm32OpLoadIndex || op == renvoWasm32OpStoreIndex {
		return pc + 10
	}
	if op == renvoWasm32OpCall {
		return pc + 9
	}
	if op == renvoWasm32OpBuildArgsEnv {
		return pc + 13
	}
	return pc + 1
}

func renvoWasm32DecodeOne(code []byte, pc int, next int) renvoWasm32Instr {
	var ins renvoWasm32Instr
	op := int(code[pc])
	ins.pc = pc
	ins.op = op
	ins.next = next
	if op == renvoWasm32OpMovRegImm {
		ins.a = int(code[pc+1])
		ins.b = renvoWasm32GetS32(code, pc+2)
		return ins
	}
	if op == renvoWasm32OpMovRegReg || op == renvoWasm32OpAddRegReg || op == renvoWasm32OpSubRegReg || op == renvoWasm32OpMulRegReg || op == renvoWasm32OpDivRegReg || op == renvoWasm32OpModRegReg || op == renvoWasm32OpAndRegReg || op == renvoWasm32OpOrRegReg || op == renvoWasm32OpXorRegReg || op == renvoWasm32OpAndNotRegReg || op == renvoWasm32OpShlRegReg || op == renvoWasm32OpShrRegReg || op == renvoWasm32OpCmpRegReg {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		return ins
	}
	if op == renvoWasm32OpPushReg || op == renvoWasm32OpPopReg || op == renvoWasm32OpIncReg || op == renvoWasm32OpIncMem || op == renvoWasm32OpDecMem || op == renvoWasm32OpBoolNot || op == renvoWasm32OpNegReg || op == renvoWasm32OpSetCond {
		ins.a = int(code[pc+1])
		return ins
	}
	if op == renvoWasm32OpPushImm {
		ins.a = renvoWasm32GetS32(code, pc+1)
		return ins
	}
	if op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack || op == renvoWasm32OpLeaStack || op == renvoWasm32OpAddRegImm || op == renvoWasm32OpMulRegImm || op == renvoWasm32OpCmpRegImm {
		ins.a = int(code[pc+1])
		ins.b = renvoWasm32GetS32(code, pc+2)
		return ins
	}
	if op == renvoWasm32OpLoadMem || op == renvoWasm32OpStoreMem {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = renvoWasm32GetS32(code, pc+3)
		ins.d = int(code[pc+7])
		return ins
	}
	if op == renvoWasm32OpLoadIndex || op == renvoWasm32OpStoreIndex {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = int(code[pc+3])
		ins.d = int(code[pc+4])
		ins.e = renvoWasm32GetS32(code, pc+5)
		ins.f = int(code[pc+9])
		return ins
	}
	if op == renvoWasm32OpJmp || op == renvoWasm32OpJz || op == renvoWasm32OpJnz {
		ins.a = renvoGet32At(code, pc+1)
		return ins
	}
	if op == renvoWasm32OpCall {
		ins.a = renvoGet32At(code, pc+1)
		ins.b = renvoGet32At(code, pc+5)
		return ins
	}
	if op == renvoWasm32OpJCond {
		ins.a = int(code[pc+1])
		ins.b = renvoGet32At(code, pc+2)
		return ins
	}
	if op == renvoWasm32OpBuildArgsEnv {
		ins.a = renvoGet32At(code, pc+1)
		ins.b = renvoGet32At(code, pc+5)
		ins.c = renvoGet32At(code, pc+9)
		return ins
	}
	return ins
}

func renvoWasm32InstructionPcs(code []byte) []int {
	out := make([]int, 0, 131072)
	pc := 0
	for pc < len(code) {
		out = append(out, pc)
		pc = renvoWasm32NextInstructionPc(code, pc)
	}
	return out
}

func renvoWasm32DecodePcRange(code []byte, pcs []int) []renvoWasm32Instr {
	out := make([]renvoWasm32Instr, 0, len(pcs))
	for i := 0; i < len(pcs); i++ {
		next := renvoWasm32NextInstructionPc(code, pcs[i])
		if i+1 < len(pcs) {
			next = pcs[i+1]
		}
		out = append(out, renvoWasm32DecodeOne(code, pcs[i], next))
	}
	return out
}

func renvoWasm32PcLowerBound(pcs []int, pc int) int {
	lo := 0
	hi := len(pcs)
	for lo < hi {
		mid := (lo + hi) / 2
		if pcs[mid] < pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func renvoWasm32InstrLowerBound(instrs []renvoWasm32Instr, pc int) int {
	lo := 0
	hi := len(instrs)
	for lo < hi {
		mid := (lo + hi) / 2
		if int(instrs[mid].pc) < pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

func renvoWasm32InstrIndexForPcLocal(instrs []renvoWasm32Instr, instrCount int, pc int) int {
	idx := renvoWasm32InstrLowerBound(instrs, pc)
	if idx < instrCount {
		if int(instrs[idx].pc) == pc {
			return idx
		}
	}
	return instrCount
}

func renvoWasm32IsControlOp(op int) bool {
	if op == renvoWasm32OpExit || op == renvoWasm32OpBuildArgsEnv {
		return true
	}
	if op == renvoWasm32OpJmp || op == renvoWasm32OpJz || op == renvoWasm32OpJnz || op == renvoWasm32OpJCond {
		return true
	}
	if op == renvoWasm32OpCall {
		return true
	}
	if op == renvoWasm32OpRet {
		return true
	}
	return false
}

func renvoWasm32OpHasTarget(op int) bool {
	if op == renvoWasm32OpJmp {
		return true
	}
	if op == renvoWasm32OpJz {
		return true
	}
	if op == renvoWasm32OpJnz {
		return true
	}
	if op == renvoWasm32OpCall {
		return true
	}
	return false
}

func renvoWasm32BuildBlockStartsLocal(instrs []renvoWasm32Instr) []int {
	starts := make([]int, 0, 512)
	instrCount := len(instrs)
	marks := make([]int, instrCount+1, instrCount+1)
	for i := 0; i < len(marks); i++ {
		marks[i] = 0
	}
	if instrCount > 0 {
		marks[0] = 1
	}
	for i := 0; i < instrCount; i++ {
		ins := &instrs[i]
		op := int(ins.op)
		if renvoWasm32IsControlOp(op) {
			if i+1 < instrCount {
				marks[i+1] = 1
			}
		}
		if renvoWasm32OpHasTarget(op) {
			targetIndex := renvoWasm32InstrIndexForPcLocal(instrs, instrCount, int(ins.a))
			if targetIndex < instrCount {
				marks[targetIndex] = 1
			}
		}
		if op == renvoWasm32OpJCond {
			targetIndex := renvoWasm32InstrIndexForPcLocal(instrs, instrCount, int(ins.b))
			if targetIndex < instrCount {
				marks[targetIndex] = 1
			}
		}
	}
	for i := 0; i < instrCount; i++ {
		if marks[i] != 0 {
			starts = append(starts, i)
		}
	}
	return starts
}

func renvoWasm32BlockEnd(starts []int, blockIndex int, instrCount int) int {
	if blockIndex+1 < len(starts) {
		return starts[blockIndex+1]
	}
	return instrCount
}

func renvoWasm32BuildInstrBlockIndex(starts []int, instrCount int) []int {
	blockIndex := make([]int, instrCount+1, instrCount+1)
	block := 0
	for i := 0; i < instrCount; i++ {
		if block+1 < len(starts) && i >= starts[block+1] {
			block++
		}
		blockIndex[i] = block
	}
	blockIndex[instrCount] = len(starts)
	return blockIndex
}

func renvoWasm32BlockForInstrFast(blockIndex []int, idx int) int {
	if idx >= 0 && idx < len(blockIndex) {
		return blockIndex[idx]
	}
	return len(blockIndex) - 1
}

func renvoWasmLocalGet(out []byte, local int) []byte {
	out = append(out, 0x20)
	out = renvoWasmAppendU32(out, local)
	return out
}

func renvoWasmLocalSet(out []byte, local int) []byte {
	out = append(out, 0x21)
	out = renvoWasmAppendU32(out, local)
	return out
}

func renvoWasmLocalTee(out []byte, local int) []byte {
	out = append(out, 0x22)
	out = renvoWasmAppendU32(out, local)
	return out
}

func renvoWasmI32Load(out []byte, align int, off int) []byte {
	out = append(out, 0x28)
	out = renvoWasmAppendU32(out, align)
	out = renvoWasmAppendU32(out, off)
	return out
}

func renvoWasmI32Load8U(out []byte) []byte {
	out = append(out, 0x2d)
	out = renvoWasmAppendU32(out, 0)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasmI32Load16S(out []byte) []byte {
	out = append(out, 0x2e)
	out = renvoWasmAppendU32(out, 1)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasmI32Store(out []byte, align int, off int) []byte {
	out = append(out, 0x36)
	out = renvoWasmAppendU32(out, align)
	out = renvoWasmAppendU32(out, off)
	return out
}

func renvoWasmI32Store8(out []byte) []byte {
	out = append(out, 0x3a)
	out = renvoWasmAppendU32(out, 0)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasmI32Store16(out []byte) []byte {
	out = append(out, 0x3b)
	out = renvoWasmAppendU32(out, 1)
	out = renvoWasmAppendU32(out, 0)
	return out
}

func renvoWasmI64Const(out []byte, value int) []byte {
	out = append(out, 0x42)
	out = renvoWasmAppendS32(out, value)
	return out
}

func renvoWasmBr(out []byte, depth int) []byte {
	out = append(out, 0x0c)
	out = renvoWasmAppendU32(out, depth)
	return out
}

func renvoWasmBrIf(out []byte, depth int) []byte {
	out = append(out, 0x0d)
	out = renvoWasmAppendU32(out, depth)
	return out
}

func renvoWasm32SetPc(out []byte, pc int) []byte {
	out = renvoWasmAppendI32Const(out, pc)
	out = renvoWasmLocalSet(out, renvoWasm32LocalPc)
	return out
}

func renvoWasm32RegGet(out []byte, reg int) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32RegLocal(reg))
	return out
}

func renvoWasm32RegSet(out []byte, reg int) []byte {
	out = renvoWasmLocalSet(out, renvoWasm32RegLocal(reg))
	return out
}

func renvoWasm32StackAddr(out []byte, offset int) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32LocalFp)
	out = renvoWasmAppendI32Const(out, offset)
	out = append(out, 0x6b)
	return out
}

func renvoWasm32MemAddr(out []byte, base int, disp int) []byte {
	out = renvoWasm32RegGet(out, base)
	if disp != 0 {
		out = renvoWasmAppendI32Const(out, disp)
		out = append(out, 0x6a)
	}
	return out
}

func renvoWasm32IndexAddr(out []byte, base int, index int, scale int, disp int) []byte {
	out = renvoWasm32RegGet(out, base)
	out = renvoWasm32RegGet(out, index)
	if scale != 1 {
		out = renvoWasmAppendI32Const(out, scale)
		out = append(out, 0x6c)
	}
	out = append(out, 0x6a)
	if disp != 0 {
		out = renvoWasmAppendI32Const(out, disp)
		out = append(out, 0x6a)
	}
	return out
}

func renvoWasm32LoadSized(out []byte, size int) []byte {
	if size == 1 {
		out = renvoWasmI32Load8U(out)
		return out
	}
	if size == 2 {
		out = renvoWasmI32Load16S(out)
		return out
	}
	out = renvoWasmI32Load(out, 2, 0)
	return out
}

func renvoWasm32StoreSized(out []byte, size int) []byte {
	if size == 1 {
		out = renvoWasmI32Store8(out)
		return out
	}
	if size == 2 {
		out = renvoWasmI32Store16(out)
		return out
	}
	out = renvoWasmI32Store(out, 2, 0)
	return out
}

func renvoWasm32AppendCond(out []byte, cond int) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32LocalFlag)
	out = renvoWasmAppendI32Const(out, 0)
	if cond == renvoWasm32CondNe {
		out = append(out, 0x47)
	} else if cond == renvoWasm32CondLt {
		out = append(out, 0x48)
	} else if cond == renvoWasm32CondLe {
		out = append(out, 0x4c)
	} else if cond == renvoWasm32CondGt {
		out = append(out, 0x4a)
	} else if cond == renvoWasm32CondGe {
		out = append(out, 0x4e)
	} else {
		out = append(out, 0x46)
	}
	return out
}

func renvoWasm32Patch(a *renvoAsm, dataBase int, bssBase int) {
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			renvoPut32At(a.code, r.at, a.labelPos[r.label])
		}
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := dataBase + r.off
		if r.kind == renvoAbsBssReloc {
			target = bssBase + r.off
		}
		renvoPut32At(a.code, r.at, target)
	}
}

func renvoWasm32AppendStoreConst(out []byte, addr int, value int) []byte {
	out = renvoWasmAppendI32Const(out, addr)
	out = renvoWasmAppendI32Const(out, value)
	out = renvoWasmI32Store(out, 2, 0)
	return out
}

func renvoWasm32AppendIov(out []byte, ptrLocal int, lenLocal int) []byte {
	out = renvoWasmAppendI32Const(out, renvoWasm32ScratchIov)
	out = renvoWasmLocalGet(out, ptrLocal)
	out = renvoWasmI32Store(out, 2, 0)
	out = renvoWasmAppendI32Const(out, renvoWasm32ScratchIov+4)
	out = renvoWasmLocalGet(out, lenLocal)
	out = renvoWasmI32Store(out, 2, 0)
	return out
}

func renvoWasm32AppendErrnoResult(out []byte, resultPtr int) []byte {
	out = renvoWasmLocalTee(out, renvoWasm32LocalTmp)
	out = append(out, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmAppendI32Const(out, resultPtr)
	out = renvoWasmI32Load(out, 2, 0)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = append(out, 0x05)
	out = renvoWasmAppendI32Const(out, -1)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func renvoWasm32AppendErrnoOnlyResult(out []byte) []byte {
	out = renvoWasmLocalTee(out, renvoWasm32LocalTmp)
	out = append(out, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = append(out, 0x05)
	out = renvoWasmAppendI32Const(out, -1)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func renvoWasm32AppendSyscall(out []byte) []byte {
	return renvoWasmAppendEncoded(out, "\x20\x02\x41\x01\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x41\x08\x10\x00\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x00\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x41\x08\x10\x01\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x12\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x20\x0b\xac\x41\x08\x10\x03\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x11\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x20\x0b\xac\x41\x08\x10\x02\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x03\x46\x04\x40\x20\x05\x10\x05\x22\x0d\x45\x04\x40\x41\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\xdb\x00\x46\x04\x40\x20\x05\x41\xc0\x00\x10\x06\x22\x0d\x45\x04\x40\x41\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\xd9\x01\x46\x04\x40\x20\x05\x20\x06\x20\x03\x42\x00\x41\x08\x10\x07\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x03\x41\x00\x4b\x04\x40\x20\x05\x20\x03\x41\x01\x6b\x6a\x2d\x00\x00\x45\x04\x40\x20\x03\x41\x01\x6b\x21\x03\x0b\x0b\x41\x00\x21\x0d\x20\x06\x41\xc0\x00\x71\x45\x45\x04\x40\x20\x0d\x41\x01\x72\x21\x0d\x0b\x20\x06\x41\x80\x04\x71\x45\x45\x04\x40\x20\x0d\x41\x08\x72\x21\x0d\x0b\x20\x05\x2d\x00\x00\x41\x2f\x46\x04\x40\x20\x05\x41\x01\x6a\x21\x05\x20\x03\x41\x01\x6b\x21\x03\x0b\x41\x03\x21\x0e\x02\x40\x03\x40\x20\x0e\x41\x00\x20\x05\x20\x03\x20\x0d\x20\x06\x41\x02\x71\x45\x04\x7e\x42\x82\x80\x01\x05\x42\xe6\x80\x80\x01\x0b\x42\x00\x41\x00\x41\x0c\x10\x04\x22\x0f\x45\x04\x40\x41\x0c\x28\x02\x00\x21\x02\x0c\x02\x0b\x20\x0e\x41\x01\x6a\x22\x0e\x41\x08\x4c\x0d\x00\x0b\x41\x7f\x21\x02\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b")
}

func renvoWasm32AppendOpen(out []byte) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmAppendI32Const(out, 0)
	out = append(out, 0x4b)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdi)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmAppendI32Const(out, 1)
	out = renvoWasmAppend2(out, 0x6b, 0x6a)
	out = renvoWasmI32Load8U(out)
	out = append(out, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmAppendI32Const(out, 1)
	out = append(out, 0x6b)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRdx)
	out = append(out, 0x0b)
	out = append(out, 0x0b)

	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmLocalSet(out, renvoWasm32LocalTmp)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRsi)
	out = renvoWasmAppendI32Const(out, 64)
	out = renvoWasmAppend3(out, 0x71, 0x45, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalTmp)
	out = renvoWasmAppendI32Const(out, 1)
	out = append(out, 0x72)
	out = renvoWasmLocalSet(out, renvoWasm32LocalTmp)
	out = append(out, 0x0b)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRsi)
	out = renvoWasmAppendI32Const(out, 512)
	out = renvoWasmAppend3(out, 0x71, 0x45, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalTmp)
	out = renvoWasmAppendI32Const(out, 8)
	out = append(out, 0x72)
	out = renvoWasmLocalSet(out, renvoWasm32LocalTmp)
	out = append(out, 0x0b)

	out = renvoWasmLocalGet(out, renvoWasm32LocalRdi)
	out = renvoWasmI32Load8U(out)
	out = renvoWasmAppendI32Const(out, 47)
	out = append(out, 0x46)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdi)
	out = renvoWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRdi)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmAppendI32Const(out, 1)
	out = append(out, 0x6b)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRdx)
	out = append(out, 0x0b)

	out = renvoWasmAppendI32Const(out, 3)
	out = renvoWasmLocalSet(out, renvoWasm32LocalTmp2)
	out = renvoWasmAppend2(out, 0x02, 0x40)
	out = renvoWasmAppend2(out, 0x03, 0x40)
	out = renvoWasmLocalGet(out, renvoWasm32LocalTmp2)
	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdi)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmLocalGet(out, renvoWasm32LocalTmp)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRsi)
	out = renvoWasmAppendI32Const(out, 2)
	out = renvoWasmAppend2(out, 0x71, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x7e)
	out = renvoWasmI64Const(out, 16386)
	out = append(out, 0x05)
	out = renvoWasmI64Const(out, 2097254)
	out = append(out, 0x0b)
	out = renvoWasmI64Const(out, 0)
	out = renvoWasmAppendI32Const(out, 0)
	out = renvoWasmAppendI32Const(out, renvoWasm32ScratchFd)
	out = renvoWasmAppendCall(out, renvoWasm32ImportPathOpen)
	out = renvoWasmLocalTee(out, renvoWasm32LocalTmp3)
	out = append(out, 0x45)
	out = renvoWasmAppend2(out, 0x04, 0x40)
	out = renvoWasmAppendI32Const(out, renvoWasm32ScratchFd)
	out = renvoWasmI32Load(out, 2, 0)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = renvoWasmBr(out, 2)
	out = append(out, 0x0b)
	out = renvoWasmLocalGet(out, renvoWasm32LocalTmp2)
	out = renvoWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = renvoWasmLocalTee(out, renvoWasm32LocalTmp2)
	out = renvoWasmAppendI32Const(out, 8)
	out = append(out, 0x4c)
	out = renvoWasmBrIf(out, 0)
	out = append(out, 0x0b)
	out = renvoWasmAppendI32Const(out, -1)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func renvoWasm32AppendStringSliceBuild(out []byte, ptrArea int, countLocal int, destAddr int) []byte {
	return renvoWasmAppendRecipe(out, "\x41\x00\x21\x0d\x02\x40\x03\x40\x20\x0d\xff\x01\x01\x4f\x0d\x01\xff\x00\x00\x20\x0d\x41\x04\x6c\x6a\x28\x02\x00\x21\x0e\x41\x00\x21\x0f\x02\x40\x03\x40\x20\x0e\x20\x0f\x6a\x2d\x00\x00\x45\x0d\x01\x20\x0f\x41\x01\x6a\x21\x0f\x0c\x00\x0b\x0b\xff\x00\x02\x20\x0d\x41\x10\x6c\x6a\x20\x0e\x36\x02\x00\xff\x00\x03\x20\x0d\x41\x10\x6c\x6a\x20\x0f\x36\x02\x00\x20\x0d\x41\x01\x6a\x21\x0d\x0c\x00\x0b\x0b", ptrArea, countLocal, destAddr, destAddr+8)
}

func renvoWasm32AppendBuildArgsEnv(out []byte, argsAddr int, envAddr int, envLenAddr int) []byte {
	out = renvoWasmAppendRecipe(out, "\x41\x10\x41\x14\x10\x08\x1a\x41\x80\x20\x41\x80\xc0\x00\x10\x09\x1a\x41\x10\x28\x02\x00\x21\x06\x20\x06\x21\x03\xff\x00\x00\x21\x05", argsAddr, 0, 0, 0)
	out = renvoWasm32AppendStringSliceBuild(out, renvoWasm32ArgsPtrArea, renvoWasm32LocalRsi, argsAddr)
	out = renvoWasmAppendRecipe(out, "\x41\x18\x41\x1c\x10\x0a\x1a\x41\x80\x80\x04\x41\x80\x80\x08\x10\x0b\x1a\x41\x18\x28\x02\x00\x21\x07\x20\x07\x21\x08\xff\x00\x01\x21\x04\xff\x00\x02\x20\x07\x36\x02\x00", 0, envAddr, envLenAddr, 0)
	out = renvoWasm32AppendStringSliceBuild(out, renvoWasm32EnvPtrArea, renvoWasm32LocalR8, envAddr)
	return out
}

func renvoWasm32AppendBinaryOp(out []byte, op int) []byte {
	if op == renvoWasm32OpAddRegReg {
		return append(out, 0x6a)
	}
	if op == renvoWasm32OpSubRegReg {
		return append(out, 0x6b)
	}
	if op == renvoWasm32OpMulRegReg {
		return append(out, 0x6c)
	}
	if op == renvoWasm32OpDivRegReg {
		return append(out, 0x6d)
	}
	if op == renvoWasm32OpModRegReg {
		return append(out, 0x6f)
	}
	if op == renvoWasm32OpAndRegReg {
		return append(out, 0x71)
	}
	if op == renvoWasm32OpOrRegReg {
		return append(out, 0x72)
	}
	if op == renvoWasm32OpXorRegReg {
		return append(out, 0x73)
	}
	if op == renvoWasm32OpShlRegReg {
		return append(out, 0x74)
	}
	if op == renvoWasm32OpShrRegReg {
		return append(out, 0x75)
	}
	return out
}

func renvoWasm32AppendInstr(out []byte, ins *renvoWasm32Instr, nextIndex int, targetIndex int, loopDepth int, exitDepth int, callStackBase int, frameSize int) []byte {
	op := int(ins.op)
	argA := int(ins.a)
	argB := int(ins.b)
	if op == renvoWasm32OpExit {
		out = renvoWasmLocalGet(out, renvoWasm32LocalRax)
		out = renvoWasmAppendCall(out, renvoWasm32ImportProcExit)
		return out
	}
	if op == renvoWasm32OpBuildArgsEnv {
		out = renvoWasm32AppendBuildArgsEnv(out, argA, argB, int(ins.c))
		out = renvoWasm32SetPc(out, nextIndex)
		out = renvoWasmBr(out, loopDepth)
		return out
	}
	if op == renvoWasm32OpMovRegImm {
		out = renvoWasmAppendRecipe(out, "\xff\x00\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == renvoWasm32OpMovRegReg {
		out = renvoWasmAppendRecipe(out, "\xff\x04\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == renvoWasm32OpPushReg {
		out = renvoWasmAppendRecipe(out, "\x20\x00\xff\x04\x00\x36\x02\x00\x20\x00\x41\x04\x6a\x21\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpPushImm {
		out = renvoWasmAppendRecipe(out, "\x20\x00\xff\x00\x00\x36\x02\x00\x20\x00\x41\x04\x6a\x21\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpPopReg {
		out = renvoWasmAppendRecipe(out, "\x20\x00\x41\x04\x6b\x22\x00\x28\x02\x00\xff\x05\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpLoadStack {
		out = renvoWasmAppendRecipe(out, "\xff\x06\x01\x28\x02\x00\xff\x05\x00", argA, argB, 0, 0)
	} else if op == renvoWasm32OpStoreStack {
		out = renvoWasmAppendRecipe(out, "\xff\x06\x01\xff\x04\x00\x36\x02\x00", argA, argB, 0, 0)
	} else if op == renvoWasm32OpLeaStack {
		out = renvoWasmAppendRecipe(out, "\xff\x06\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == renvoWasm32OpLoadMem {
		out = renvoWasm32MemAddr(out, argB, int(ins.c))
		out = renvoWasm32LoadSized(out, int(ins.d))
		out = renvoWasm32RegSet(out, argA)
	} else if op == renvoWasm32OpStoreMem {
		out = renvoWasm32MemAddr(out, argB, int(ins.c))
		out = renvoWasm32RegGet(out, argA)
		out = renvoWasm32StoreSized(out, int(ins.d))
	} else if op == renvoWasm32OpLoadIndex {
		out = renvoWasm32IndexAddr(out, argB, int(ins.c), int(ins.d), int(ins.e))
		out = renvoWasm32LoadSized(out, int(ins.f))
		out = renvoWasm32RegSet(out, argA)
	} else if op == renvoWasm32OpStoreIndex {
		out = renvoWasm32IndexAddr(out, argB, int(ins.c), int(ins.d), int(ins.e))
		out = renvoWasm32RegGet(out, argA)
		out = renvoWasm32StoreSized(out, int(ins.f))
	} else if op >= renvoWasm32OpAddRegReg && op <= renvoWasm32OpShrRegReg {
		if op == renvoWasm32OpAndNotRegReg {
			out = renvoWasm32RegGet(out, argA)
			out = renvoWasm32RegGet(out, argB)
			out = renvoWasmAppend4(out, 0x41, 0x7f, 0x73, 0x71)
		} else {
			out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x01\xff\x0a\x02", argA, argB, op, 0)
		}
		out = renvoWasm32RegSet(out, argA)
	} else if op == renvoWasm32OpAddRegImm || op == renvoWasm32OpMulRegImm {
		if op == renvoWasm32OpAddRegImm {
			out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6a\xff\x05\x00", argA, argB, 0, 0)
		} else {
			out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6c\xff\x05\x00", argA, argB, 0, 0)
		}
	} else if op == renvoWasm32OpIncReg {
		out = renvoWasmAppendRecipe(out, "\xff\x04\x00\x41\x01\x6a\xff\x05\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpIncMem || op == renvoWasm32OpDecMem {
		if op == renvoWasm32OpIncMem {
			out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x00\x28\x02\x00\x41\x01\x6a\x36\x02\x00", argA, 0, 0, 0)
		} else {
			out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x00\x28\x02\x00\x41\x01\x6b\x36\x02\x00", argA, 0, 0, 0)
		}
	} else if op == renvoWasm32OpBoolNot {
		out = renvoWasmAppendRecipe(out, "\xff\x04\x00\x45\xff\x05\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpNegReg {
		out = renvoWasmAppendRecipe(out, "\x41\x00\xff\x04\x00\x6b\xff\x05\x00", argA, 0, 0, 0)
	} else if op == renvoWasm32OpCmpRegImm {
		out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6b\x21\x0c", argA, argB, 0, 0)
	} else if op == renvoWasm32OpCmpRegReg {
		out = renvoWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x01\x6b\x21\x0c", argA, argB, 0, 0)
	} else if op == renvoWasm32OpSetCond {
		out = renvoWasmAppendRecipe(out, "\xff\x09\x00\x21\x02", argA, 0, 0, 0)
	} else if op == renvoWasm32OpJmp {
		out = renvoWasm32SetPc(out, targetIndex)
		return renvoWasmBr(out, loopDepth)
	} else if op == renvoWasm32OpJz || op == renvoWasm32OpJnz || op == renvoWasm32OpJCond {
		if op == renvoWasm32OpJz {
			out = renvoWasmLocalGet(out, renvoWasm32LocalFlag)
			out = append(out, 0x45)
		} else if op == renvoWasm32OpJnz {
			out = renvoWasmLocalGet(out, renvoWasm32LocalFlag)
			out = renvoWasmAppend2(out, 0x45, 0x45)
		} else {
			out = renvoWasm32AppendCond(out, argA)
		}
		out = renvoWasmAppend2(out, 0x04, 0x40)
		out = renvoWasm32SetPc(out, targetIndex)
		out = append(out, 0x05)
		out = renvoWasm32SetPc(out, nextIndex)
		out = append(out, 0x0b)
		return renvoWasmBr(out, loopDepth)
	} else if op == renvoWasm32OpCall {
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, nextIndex)
		out = renvoWasmI32Store(out, 2, 0)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmLocalGet(out, renvoWasm32LocalFp)
		out = renvoWasmI32Store(out, 2, 4)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, 8)
		out = append(out, 0x6a)
		out = renvoWasmLocalSet(out, renvoWasm32LocalCsp)
		out = renvoWasmLocalGet(out, renvoWasm32LocalFp)
		out = renvoWasmAppendI32Const(out, frameSize)
		out = append(out, 0x6b)
		out = renvoWasmLocalSet(out, renvoWasm32LocalFp)
		out = renvoWasm32SetPc(out, targetIndex)
		return renvoWasmBr(out, loopDepth)
	} else if op == renvoWasm32OpRet {
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, callStackBase)
		out = append(out, 0x46)
		out = renvoWasmAppend2(out, 0x04, 0x40)
		out = renvoWasmLocalGet(out, renvoWasm32LocalRax)
		out = renvoWasmAppendCall(out, renvoWasm32ImportProcExit)
		out = append(out, 0x05)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, 8)
		out = append(out, 0x6b)
		out = renvoWasmLocalTee(out, renvoWasm32LocalCsp)
		out = renvoWasmI32Load(out, 2, 0)
		out = renvoWasmLocalSet(out, renvoWasm32LocalPc)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmI32Load(out, 2, 4)
		out = renvoWasmLocalSet(out, renvoWasm32LocalFp)
		out = append(out, 0x0b)
		return renvoWasmBr(out, loopDepth)
	} else if op == renvoWasm32OpSyscall {
		out = renvoWasm32AppendSyscall(out)
	}
	if loopDepth < 0 {
		return out
	}
	out = renvoWasm32SetPc(out, nextIndex)
	return renvoWasmBr(out, loopDepth)
}

func renvoWasm32AppendDirectArgs(out []byte) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32LocalSp)
	out = renvoWasmLocalGet(out, renvoWasm32LocalFp)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRax)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRcx)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdi)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRsi)
	out = renvoWasmLocalGet(out, renvoWasm32LocalR8)
	out = renvoWasmLocalGet(out, renvoWasm32LocalR9)
	return out
}

func renvoWasm32AppendStateResults(out []byte) []byte {
	out = renvoWasmLocalGet(out, renvoWasm32LocalRax)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRdx)
	out = renvoWasmLocalGet(out, renvoWasm32LocalRcx)
	return out
}

func renvoWasm32AppendStateReturn(out []byte) []byte {
	out = renvoWasm32AppendStateResults(out)
	out = append(out, 0x0f)
	return out
}

func renvoWasm32FindRoutineIndex(routinePcs []int, pc int) int {
	lo := 0
	hi := len(routinePcs)
	for lo < hi {
		mid := (lo + hi) / 2
		value := routinePcs[mid]
		if value == pc {
			return mid
		}
		if value < pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return -1
}

func renvoWasm32MarkFunc(g *renvoLinearGen, fnIndex int) {
	if fnIndex < 0 || fnIndex >= len(g.funcReachable) {
		return
	}
	if g.funcReachable[fnIndex] {
		return
	}
	g.funcReachable[fnIndex] = true
	g.funcQueue = append(g.funcQueue, fnIndex)
	src := g.meta.prog.src
	nameStart := g.meta.funcs[fnIndex].nameStart
	nameEnd := g.meta.funcs[fnIndex].nameEnd
	renvoAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func renvoWasm32AppendDirectCall(out []byte, funcIndex int, frameSize int) []byte {
	out = renvoWasm32AppendDirectArgs(out)
	out = renvoWasmAppendCall(out, funcIndex)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRcx)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRdx)
	out = renvoWasmLocalSet(out, renvoWasm32LocalRax)
	return out
}

func renvoWasm32AppendInstrDirect(out []byte, ins *renvoWasm32Instr, nextIndex int, targetIndex int, loopDepth int, callStackBase int, frameSize int, hasFrame bool, routinePcs []int) []byte {
	op := int(ins.op)
	argA := int(ins.a)
	argB := int(ins.b)
	if op == renvoWasm32OpExit {
		out = renvoWasmLocalGet(out, renvoWasm32LocalRax)
		out = renvoWasmAppendCall(out, renvoWasm32ImportProcExit)
		out = append(out, 0x00)
		return out
	}
	if op == renvoWasm32OpCall {
		routineIndex := renvoWasm32FindRoutineIndex(routinePcs, argA)
		if routineIndex < 0 {
			out = renvoWasm32AppendInstr(out, ins, nextIndex, targetIndex, loopDepth, loopDepth+1, callStackBase, frameSize)
			return out
		}
		out = renvoWasm32AppendDirectCall(out, renvoWasm32VmFuncBase+routineIndex, frameSize)
		if argB > 6 {
			// Direct wasm calls pass SP by value; drop caller-owned stack args.
			out = renvoWasmLocalGet(out, renvoWasm32LocalSp)
			out = renvoWasmAppendI32Const(out, (argB-6)*4)
			out = append(out, 0x6b)
			out = renvoWasmLocalSet(out, renvoWasm32LocalSp)
		}
		if loopDepth < 0 {
			return out
		}
		out = renvoWasm32SetPc(out, nextIndex)
		return renvoWasmBr(out, loopDepth)
	}
	if op == renvoWasm32OpRet {
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, callStackBase)
		out = append(out, 0x46)
		out = renvoWasmAppend2(out, 0x04, 0x40)
		if hasFrame {
			out = renvoWasmLocalGet(out, renvoWasm32LocalFp)
			out = renvoWasmAppendI32Const(out, frameSize)
			out = append(out, 0x6a)
			out = renvoWasmLocalSet(out, renvoWasm32LocalFp)
		}
		out = renvoWasm32AppendStateReturn(out)
		out = append(out, 0x05)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmAppendI32Const(out, 8)
		out = append(out, 0x6b)
		out = renvoWasmLocalTee(out, renvoWasm32LocalCsp)
		out = renvoWasmI32Load(out, 2, 0)
		out = renvoWasmLocalSet(out, renvoWasm32LocalPc)
		out = renvoWasmLocalGet(out, renvoWasm32LocalCsp)
		out = renvoWasmI32Load(out, 2, 4)
		out = renvoWasmLocalSet(out, renvoWasm32LocalFp)
		out = append(out, 0x0b)
		if loopDepth >= 0 {
			out = renvoWasmBr(out, loopDepth)
		}
		return out
	}
	out = renvoWasm32AppendInstr(out, ins, nextIndex, targetIndex, loopDepth, loopDepth+1, callStackBase, frameSize)
	return out
}

func renvoWasm32CanFusePair(first *renvoWasm32Instr, second *renvoWasm32Instr) bool {
	if second.op == renvoWasm32OpPopReg {
		if first.op == renvoWasm32OpPushReg {
			return true
		}
		if first.op == renvoWasm32OpPushImm {
			return true
		}
	}
	if first.op == renvoWasm32OpStoreStack && second.op == renvoWasm32OpLoadStack && first.b == second.b {
		return true
	}
	return false
}

func renvoWasm32AppendFusedPair(out []byte, first *renvoWasm32Instr, second *renvoWasm32Instr) []byte {
	if second.op == renvoWasm32OpPopReg {
		if first.op == renvoWasm32OpPushReg {
			if first.a == second.a {
				return out
			}
			out = renvoWasm32RegGet(out, int(first.a))
			return renvoWasm32RegSet(out, int(second.a))
		}
		out = renvoWasmAppendI32Const(out, int(first.a))
		return renvoWasm32RegSet(out, int(second.a))
	}
	out = renvoWasm32StackAddr(out, int(first.b))
	out = renvoWasm32RegGet(out, int(first.a))
	out = renvoWasmI32Store(out, 2, 0)
	if first.a == second.a {
		return out
	}
	out = renvoWasm32RegGet(out, int(first.a))
	return renvoWasm32RegSet(out, int(second.a))
}

func renvoWasm32PcInList(pcs []int, pc int) bool {
	for i := 0; i < len(pcs); i++ {
		if pcs[i] == pc {
			return true
		}
	}
	return false
}

func renvoWasm32SortPcs(pcs []int) []int {
	for i := 1; i < len(pcs); i++ {
		value := pcs[i]
		j := i - 1
		for j >= 0 && pcs[j] > value {
			pcs[j+1] = pcs[j]
			j--
		}
		pcs[j+1] = value
	}
	return pcs
}

func renvoWasm32SymbolPcs(a *renvoAsm) []int {
	pcs := make([]int, 0, 2048)
	for i := 0; i < len(a.symbols); i++ {
		label := a.symbols[i].label
		if label >= 0 && label < len(a.labelPos) && a.labelSet[label] {
			pc := a.labelPos[label]
			if !renvoWasm32PcInList(pcs, pc) {
				pcs = append(pcs, pc)
			}
		}
	}
	pcs = renvoWasm32SortPcs(pcs)
	return pcs
}

func renvoWasm32RoutinePcs(a *renvoAsm, code []byte, instrPcs []int) []int {
	pcs := make([]int, 0, 1024)
	marks := make([]byte, len(code)+1)
	pcs = append(pcs, 0)
	marks[0] = 1
	for i := 0; i < len(a.symbols); i++ {
		label := a.symbols[i].label
		if label >= 0 && label < len(a.labelPos) && a.labelSet[label] {
			pc := a.labelPos[label]
			if pc >= 0 && pc < len(marks) && marks[pc] == 0 {
				pcs = append(pcs, pc)
				marks[pc] = 1
			}
		}
	}
	for i := 0; i < len(instrPcs); i++ {
		pc := instrPcs[i]
		if int(code[pc]) == renvoWasm32OpCall {
			targetPc := renvoGet32At(code, pc+1)
			targetIndex := renvoWasm32PcLowerBound(instrPcs, targetPc)
			if targetIndex < len(instrPcs) && instrPcs[targetIndex] == targetPc {
				if targetPc >= 0 && targetPc < len(marks) && marks[targetPc] == 0 {
					pcs = append(pcs, targetPc)
					marks[targetPc] = 1
				}
			}
		}
	}
	pcs = renvoWasm32SortPcs(pcs)
	return pcs
}

func renvoWasm32NextPcAfter(pcs []int, pc int, limit int) int {
	lo := 0
	hi := len(pcs)
	for lo < hi {
		mid := (lo + hi) / 2
		if pcs[mid] <= pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	if lo < len(pcs) && pcs[lo] < limit {
		return pcs[lo]
	}
	return limit
}

func renvoWasm32SortedPcContains(pcs []int, pc int) bool {
	lo := 0
	hi := len(pcs)
	for lo < hi {
		mid := (lo + hi) / 2
		if pcs[mid] < pc {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo < len(pcs) && pcs[lo] == pc
}

func renvoWasm32FirstRetAfter(code []byte, instrPcs []int, startPc int, limit int) int {
	i := renvoWasm32PcLowerBound(instrPcs, startPc)
	for i < len(instrPcs) {
		pc := instrPcs[i]
		if pc >= limit {
			break
		}
		if int(code[pc]) == renvoWasm32OpRet {
			return pc + 1
		}
		i++
	}
	return limit
}

func renvoWasm32RoutineEndPc(startPc int, codeLen int, symbolPcs []int, code []byte, instrPcs []int) int {
	nextSymbol := renvoWasm32NextPcAfter(symbolPcs, startPc, codeLen)
	if startPc == 0 || renvoWasm32SortedPcContains(symbolPcs, startPc) {
		return nextSymbol
	}
	return renvoWasm32FirstRetAfter(code, instrPcs, startPc, nextSymbol)
}

func renvoWasm32AppendDirectRoutineBody(body []byte, instrs []renvoWasm32Instr, codeLen int, routinePcs []int, callStackBase int, frameSize int, hasFrame bool) []byte {
	blockStarts := renvoWasm32BuildBlockStartsLocal(instrs)
	instrBlockIndex := renvoWasm32BuildInstrBlockIndex(blockStarts, len(instrs))
	body = renvoWasmAppendU32(body, 1)
	body = renvoWasmAppendU32(body, 7)
	body = append(body, 0x7f)
	body = renvoWasmAppendI32Const(body, 0)
	body = renvoWasmLocalSet(body, renvoWasm32LocalPc)
	body = renvoWasmAppendI32Const(body, callStackBase)
	body = renvoWasmLocalSet(body, renvoWasm32LocalCsp)
	if hasFrame {
		body = renvoWasmLocalGet(body, renvoWasm32LocalFp)
		body = renvoWasmAppendI32Const(body, frameSize)
		body = append(body, 0x6b)
		body = renvoWasmLocalSet(body, renvoWasm32LocalFp)
	}
	if len(blockStarts) == 0 {
		body = renvoWasm32AppendStateResults(body)
		body = append(body, 0x0b)
		return body
	}
	body = renvoWasmAppend2(body, 0x02, 0x40)
	body = renvoWasmAppend2(body, 0x03, 0x40)
	for i := 0; i < len(blockStarts); i++ {
		body = renvoWasmAppend2(body, 0x02, 0x40)
	}
	body = renvoWasmLocalGet(body, renvoWasm32LocalPc)
	body = append(body, 0x0e)
	body = renvoWasmAppendU32(body, len(blockStarts))
	for i := 0; i < len(blockStarts); i++ {
		body = renvoWasmAppendU32(body, len(blockStarts)-1-i)
	}
	defaultDepth := len(blockStarts) - 1
	body = renvoWasmAppendU32(body, defaultDepth)
	for blockIndex := len(blockStarts) - 1; blockIndex >= 0; blockIndex-- {
		body = append(body, 0x0b)
		start := blockStarts[blockIndex]
		end := renvoWasm32BlockEnd(blockStarts, blockIndex, len(instrs))
		i := start
		for i < end {
			ins := &instrs[i]
			if i+1 < end && renvoWasm32CanFusePair(ins, &instrs[i+1]) {
				body = renvoWasm32AppendFusedPair(body, ins, &instrs[i+1])
				if i+2 >= end {
					nextIndex := renvoWasm32InstrIndexForPcLocal(instrs, len(instrs), int(instrs[i+1].next))
					nextBlock := renvoWasm32BlockForInstrFast(instrBlockIndex, nextIndex)
					body = renvoWasm32SetPc(body, nextBlock)
					body = renvoWasmBr(body, blockIndex)
				}
				i += 2
				continue
			}
			if i+1 < end {
				body = renvoWasm32AppendInstrDirect(body, ins, 0, 0, -1, callStackBase, frameSize, hasFrame, routinePcs)
			} else {
				nextIndex := renvoWasm32InstrIndexForPcLocal(instrs, len(instrs), int(ins.next))
				nextBlock := renvoWasm32BlockForInstrFast(instrBlockIndex, nextIndex)
				targetBlock := 0
				op := int(ins.op)
				if renvoWasm32OpHasTarget(op) || op == renvoWasm32OpJCond {
					targetPc := int(ins.a)
					if ins.op == renvoWasm32OpJCond {
						targetPc = int(ins.b)
					}
					targetIndex := renvoWasm32InstrIndexForPcLocal(instrs, len(instrs), targetPc)
					targetBlock = renvoWasm32BlockForInstrFast(instrBlockIndex, targetIndex)
				}
				body = renvoWasm32AppendInstrDirect(body, ins, nextBlock, targetBlock, blockIndex, callStackBase, frameSize, hasFrame, routinePcs)
			}
			i++
		}
	}
	body = renvoWasmBr(body, 1)
	body = append(body, 0x0b)
	body = append(body, 0x0b)
	body = renvoWasm32AppendStateResults(body)
	body = append(body, 0x0b)
	return body
}

func renvoWasm32AppendDirectStartBody(body []byte, topFunc int, exprStackBase int, callStackBase int, frameTop int) []byte {
	body = renvoWasmAppendU32(body, 1)
	body = renvoWasmAppendU32(body, 16)
	body = append(body, 0x7f)
	body = renvoWasmAppendI32Const(body, 0)
	body = renvoWasmLocalSet(body, renvoWasm32LocalPc)
	body = renvoWasmAppendI32Const(body, exprStackBase)
	body = renvoWasmLocalSet(body, renvoWasm32LocalSp)
	body = renvoWasmAppendI32Const(body, frameTop)
	body = renvoWasmLocalSet(body, renvoWasm32LocalFp)
	body = renvoWasmAppendI32Const(body, callStackBase)
	body = renvoWasmLocalSet(body, renvoWasm32LocalCsp)
	body = renvoWasm32AppendDirectArgs(body)
	body = renvoWasmAppendCall(body, topFunc)
	body = renvoWasmLocalSet(body, renvoWasm32LocalRcx)
	body = renvoWasmLocalSet(body, renvoWasm32LocalRdx)
	body = renvoWasmLocalSet(body, renvoWasm32LocalRax)
	body = renvoWasmLocalGet(body, renvoWasm32LocalRax)
	body = renvoWasmAppendCall(body, renvoWasm32ImportProcExit)
	body = append(body, 0x0b)
	return body
}

func renvoWasm32TypeSectionFull() []byte {
	return renvoWasmAppendEncoded(nil, "\x08\x60\x04\x7f\x7f\x7f\x7f\x01\x7f\x60\x05\x7f\x7f\x7f\x7e\x7f\x01\x7f\x60\x09\x7f\x7f\x7f\x7f\x7f\x7e\x7e\x7f\x7f\x01\x7f\x60\x01\x7f\x01\x7f\x60\x01\x7f\x00\x60\x00\x00\x60\x02\x7f\x7f\x01\x7f\x60\x09\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x03\x7f\x7f\x7f")
}

func renvoWasm32AppendImport(out []byte, name string, typ int) []byte {
	out = renvoWasmAppendName(out, "wasi_snapshot_preview1")
	out = renvoWasmAppendName(out, name)
	out = append(out, 0x00)
	return renvoWasmAppendU32(out, typ)
}

func renvoWasm32ImportSectionFull() []byte {
	return renvoWasmAppendEncoded(nil, "\x0d\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x77\x72\x69\x74\x65\x00\x00\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x07\x66\x64\x5f\x72\x65\x61\x64\x00\x00\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x70\x72\x65\x61\x64\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x66\x64\x5f\x70\x77\x72\x69\x74\x65\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x70\x61\x74\x68\x5f\x6f\x70\x65\x6e\x00\x02\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x63\x6c\x6f\x73\x65\x00\x03\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0d\x66\x64\x5f\x66\x64\x73\x74\x61\x74\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0a\x66\x64\x5f\x72\x65\x61\x64\x64\x69\x72\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0e\x61\x72\x67\x73\x5f\x73\x69\x7a\x65\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x61\x72\x67\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x11\x65\x6e\x76\x69\x72\x6f\x6e\x5f\x73\x69\x7a\x65\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0b\x65\x6e\x76\x69\x72\x6f\x6e\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x70\x72\x6f\x63\x5f\x65\x78\x69\x74\x00\x04")
}

func renvoWasm32FunctionSectionDirect(routineCount int) []byte {
	var out []byte
	out = renvoWasmAppendU32(out, routineCount+1)
	out = renvoWasmAppendU32(out, 5)
	for i := 0; i < routineCount; i++ {
		out = renvoWasmAppendU32(out, renvoWasm32VmFuncType)
	}
	return out
}

func renvoWasm32MemorySectionFull(memSize int) []byte {
	pages := (memSize + 65535) / 65536
	if pages < 16 {
		pages = 16
	}
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = renvoWasmAppendU32(out, pages)
	return out
}

func renvoWasm32ExportSectionFull() []byte {
	return renvoWasmAppendEncoded(nil, "\x02\x06\x6d\x65\x6d\x6f\x72\x79\x02\x00\x06\x5f\x73\x74\x61\x72\x74\x00\x0d")
}

func renvoWasm32AppendCodeSectionDirect(out []byte, a *renvoAsm, instrPcs []int, routinePcs []int, symbolPcs []int, codeLen int, callStackBase int, frameTop int, exprStackBase int) []byte {
	frameSize := 6144
	out = append(out, 10)
	lenAt := len(out)
	out = renvoWasmAppendU32Fixed5(out, 0)
	payloadStart := len(out)
	out = renvoWasmAppendU32(out, len(routinePcs)+1)
	startLenAt := len(out)
	out = renvoWasmAppendU32Fixed5(out, 0)
	startBody := len(out)
	out = renvoWasm32AppendDirectStartBody(out, renvoWasm32VmFuncBase, exprStackBase, callStackBase, frameTop)
	renvoWasmPatchU32Fixed5(out, startLenAt, len(out)-startBody)
	for i := 0; i < len(routinePcs); i++ {
		startPc := routinePcs[i]
		endPc := renvoWasm32RoutineEndPc(startPc, codeLen, symbolPcs, a.code, instrPcs)
		startIndex := renvoWasm32PcLowerBound(instrPcs, startPc)
		endIndex := renvoWasm32PcLowerBound(instrPcs, endPc)
		routineInstrPcs := instrPcs[startIndex:endIndex]
		hasFrame := startPc != 0 && renvoWasm32SortedPcContains(symbolPcs, startPc)
		out = renvoWasm32AppendDirectRoutine(out, a.code, routineInstrPcs, codeLen, routinePcs, callStackBase, frameSize, hasFrame)
	}
	renvoWasmPatchU32Fixed5(out, lenAt, len(out)-payloadStart)
	return out
}

func renvoWasm32EnsureAdditionalCapacity(out []byte, additional int) []byte {
	need := len(out) + additional
	if need <= cap(out) {
		return out
	}
	nextCap := cap(out) * 2
	if nextCap < need {
		nextCap = need
	}
	next := make([]byte, len(out), nextCap)
	copy(next, out)
	return next
}

func renvoWasm32AppendDirectRoutine(out []byte, code []byte, instrPcs []int, codeLen int, routinePcs []int, callStackBase int, frameSize int, hasFrame bool) []byte {
	out = renvoWasm32EnsureAdditionalCapacity(out, len(instrPcs)*16+renvoWasm32RoutineBodyCapacity)
	mark := renvo_runtime_ArenaMark()
	instrs := renvoWasm32DecodePcRange(code, instrPcs)
	oldCap := cap(out)
	lenAt := len(out)
	out = renvoWasmAppendU32Fixed5(out, 0)
	bodyStart := len(out)
	out = renvoWasm32AppendDirectRoutineBody(out, instrs, codeLen, routinePcs, callStackBase, frameSize, hasFrame)
	renvoWasmPatchU32Fixed5(out, lenAt, len(out)-bodyStart)
	if cap(out) == oldCap {
		renvo_runtime_ArenaReset(mark)
	}
	return out
}

func renvoWasm32DataSectionFull(dataBase int, data []byte) []byte {
	var out []byte
	out = renvoWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = renvoWasmAppendI32Const(out, dataBase)
	out = append(out, 0x0b)
	out = renvoWasmAppendByteVec(out, data)
	return out
}

func renvoWasm32Image(a *renvoAsm) []byte {
	dataBase := renvoWasm32ProgramBase
	bssBase := renvoAlignTo8(dataBase + len(a.data))
	renvoWasm32Patch(a, dataBase, bssBase)
	instrPcs := renvoWasm32InstructionPcs(a.code)
	symbolPcs := renvoWasm32SymbolPcs(a)
	routinePcs := renvoWasm32RoutinePcs(a, a.code, instrPcs)
	exprStackBase := bssBase + a.bssSize + renvoWasm32StackGuardSize
	callStackBase := exprStackBase + renvoWasm32ExprStackSize
	frameTop := callStackBase + renvoWasm32CallStackSize + renvoWasm32FrameStackSize
	memSize := bssBase + a.bssSize + renvoWasm32StackGuardSize + renvoWasm32ExprStackSize + renvoWasm32CallStackSize + renvoWasm32FrameStackSize + renvoWasm32StackGuardSize
	out := make([]byte, 0, renvoWasm32ImageOutputCapacity)
	out = append(out, 0x00)
	out = append(out, 0x61)
	out = append(out, 0x73)
	out = append(out, 0x6d)
	out = append(out, 0x01)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = renvoWasmAppendSection(out, 1, renvoWasm32TypeSectionFull())
	out = renvoWasmAppendSection(out, 2, renvoWasm32ImportSectionFull())
	out = renvoWasmAppendSection(out, 3, renvoWasm32FunctionSectionDirect(len(routinePcs)))
	out = renvoWasmAppendSection(out, 5, renvoWasm32MemorySectionFull(memSize))
	out = renvoWasmAppendSection(out, 7, renvoWasm32ExportSectionFull())
	out = renvoWasm32AppendCodeSectionDirect(out, a, instrPcs, routinePcs, symbolPcs, len(a.code), callStackBase, frameTop, exprStackBase)
	if len(a.data) > 0 {
		out = renvoWasmAppendSection(out, 11, renvoWasm32DataSectionFull(dataBase, a.data))
	}
	return out
}

func renvoWasm32EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldLocalCount := g.localCount
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldClosureEnvOffset := g.closureEnvOffset
	oldDeferHeadOffset := g.deferHeadOffset
	oldDeferReturnLabel := g.deferReturnLabel
	oldDeferResultOffset := g.deferResultOffset
	oldDeferSites := g.deferSites
	oldEmittingDefers := g.emittingDefers
	oldSuppressPanicCheck := g.suppressPanicCheck
	oldStackUsed := g.stackUsed
	oldStackPeak := g.stackPeak
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []renvoLocalInfo
	var gotoLabels []renvoGlobalInfo
	locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	gotoLabels = make([]renvoGlobalInfo, 0, 0)
	g.locals = locals
	g.localCount = 0
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.stackUsed = 0
	g.stackPeak = 0
	renvoAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	if renvoTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = renvoAddTypedLocal(g, 0, 0, renvoTypeInt)
		renvoWasm32EmitStack(a, renvoWasm32OpStoreStack, renvoWasm32RegRdi, g.returnStruct)
	}
	renvoBindFunctionParams(g, fnInfoIndex)
	if !renvoBindClosureCaptures(g, fnInfoIndex) {
		return false
	}
	if !renvoBindNamedResults(g, fnInfoIndex) {
		return false
	}
	if !renvoPrepareFunctionControl(g) {
		return false
	}
	if !renvoEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if g.deferReturnLabel > 0 {
		if !g.lastRangeReturns {
			renvoAsmJmpLabel(a, g.deferReturnLabel)
		}
		if !renvoEmitFunctionControlEpilogue(g) {
			return false
		}
	} else if !g.lastRangeReturns {
		renvoMoveCapturedLocals(g, true)
		renvoAsmPrimaryImm(a, 0)
		renvoAsmLeave(a)
		renvoAsmRet(a)
	}
	g.locals = oldLocals
	g.localCount = oldLocalCount
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.closureEnvOffset = oldClosureEnvOffset
	g.deferHeadOffset = oldDeferHeadOffset
	g.deferReturnLabel = oldDeferReturnLabel
	g.deferResultOffset = oldDeferResultOffset
	g.deferSites = oldDeferSites
	g.emittingDefers = oldEmittingDefers
	g.suppressPanicCheck = oldSuppressPanicCheck
	g.stackUsed = oldStackUsed
	g.stackPeak = oldStackPeak
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func renvoWasm32EmitCallWithWordCount(g *renvoLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	renvoWasm32MarkFunc(g, fnIndex)
	if wordCount > 0 {
		renvoWasm32AsmPopRdi(a)
	}
	if wordCount > 1 {
		renvoWasm32AsmPopRsi(a)
	}
	if wordCount > 2 {
		renvoWasm32AsmPopRdx(a)
	}
	if wordCount > 3 {
		renvoWasm32AsmPopRcx(a)
	}
	if wordCount > 4 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR8)
	}
	if wordCount > 5 {
		renvoWasm32EmitReg(a, renvoWasm32OpPopReg, renvoWasm32RegR9)
	}
	renvoWasm32EmitCallLabel(a, g.funcLabels[fnIndex], wordCount)
}

func renvoWasm32EmitRaxRcxOp(g *renvoLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= renvoTokCount(p) {
		return false
	}
	start := renvoTokStart(p, tok)
	end := renvoTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		renvoWasm32AsmAddRaxRcx(a)
		return true
	}
	if c0 == '-' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRcx)
		renvoWasm32EmitRegReg(a, renvoWasm32OpSubRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
		renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		return true
	}
	if c0 == '*' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '/' {
		renvoWasm32AsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		renvoWasm32AsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpAndNotRegReg, renvoWasm32RegRcx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		} else {
			renvoWasm32EmitRegReg(a, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		}
		return true
	}
	if c0 == '|' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '^' {
		renvoWasm32EmitRegReg(a, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
			renvoWasm32EmitRegReg(a, renvoWasm32OpShlRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		} else if c1 == '=' {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9e)
		} else {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRdx, renvoWasm32RegRax)
			renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
			renvoWasm32EmitRegReg(a, renvoWasm32OpShrRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
		} else if c1 == '=' {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9d)
		} else {
			renvoWasm32AsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		renvoWasm32AsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		renvoWasm32AsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func renvoWasm32EmitFloatBinaryExpr(g *renvoLinearGen, ep *renvoExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if renvoTokCharIs(p, e.tok, '*') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmSarPrimaryImm(a, 2)
		return true
	}
	if renvoTokCharIs(p, e.tok, '/') {
		if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
			return false
		}
		renvoAsmShlPrimaryImm(a, 2)
		renvoAsmPushPrimary(a)
		if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
			return false
		}
		renvoAsmPopTertiary(a)
		renvoAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !renvoEmitScalarExprForKind(g, ep, e.left, renvoTypeFloat64) {
		return false
	}
	renvoAsmPushPrimary(a)
	if !renvoEmitScalarExprForKind(g, ep, e.right, renvoTypeFloat64) {
		return false
	}
	renvoAsmPopTertiary(a)
	return renvoEmitPrimaryTertiaryOp(g, e.tok)
}

func renvoWasm32EnsureAppendAddrHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := renvoEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.appendAddrLabel)
	noGrowLabel := renvoAsmNewLabel(a)
	capNonZeroLabel := renvoAsmNewLabel(a)
	capReadyLabel := renvoAsmNewLabel(a)
	copyLoopLabel := renvoAsmNewLabel(a)
	copyDoneLabel := renvoAsmNewLabel(a)
	returnLabel := renvoAsmNewLabel(a)
	ptrSlotOff := g.asm.bssSize
	lenSlotOff := ptrSlotOff + 4
	capSlotOff := lenSlotOff + 4
	elemSizeOff := capSlotOff + 4
	oldLenOff := elemSizeOff + 4
	oldPtrOff := oldLenOff + 4
	newCapOff := oldPtrOff + 4
	allocSizeOff := newCapOff + 4
	copySizeOff := allocSizeOff + 4
	destOff := copySizeOff + 4
	copyIndexOff := destOff + 4
	g.asm.bssSize += 44

	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR8, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegR9, 0, 4)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegR8, renvoWasm32RegRcx)
	renvoWasm32EmitCondBranch(a, renvoWasm32CondLt, noGrowLabel)

	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	renvoAsmStorePrimaryBss(a, elemSizeOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRdi)
	renvoAsmStorePrimaryBss(a, ptrSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRsi)
	renvoAsmStorePrimaryBss(a, lenSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegR9)
	renvoAsmStorePrimaryBss(a, capSlotOff)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegR8)
	renvoAsmStorePrimaryBss(a, oldLenOff)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoAsmStorePrimaryBss(a, oldPtrOff)

	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, capNonZeroLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, 16)
	renvoAsmJmpMarkLabel(a, capReadyLabel, capNonZeroLabel)
	renvoWasm32EmitRegReg(a, renvoWasm32OpAddRegReg, renvoWasm32RegRcx, renvoWasm32RegR8)
	renvoAsmMarkLabel(a, capReadyLabel)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMovRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryBss(a, newCapOff)
	renvoAsmLoadPrimaryBss(a, elemSizeOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, newCapOff)
	renvoAsmPopTertiary(a)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryBss(a, allocSizeOff)

	renvoAsmLoadPrimaryBss(a, allocSizeOff)
	renvoAsmCallLabel(a, arenaAllocLabel)
	renvoAsmStorePrimaryBss(a, destOff)

	renvoAsmLoadPrimaryBss(a, oldLenOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, elemSizeOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopPrimary(a)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRax, renvoWasm32RegRdx)
	renvoAsmStorePrimaryBss(a, copySizeOff)
	renvoAsmPrimaryImm(a, 0)
	renvoAsmStorePrimaryBss(a, copyIndexOff)
	renvoAsmMarkLabel(a, copyLoopLabel)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, copySizeOff)
	renvoAsmPopTertiary(a)
	renvoAsmCmpTertiaryPrimarySet(a, 0x9d)
	renvoAsmCmpPrimaryImm8(a, 0)
	renvoAsmJnzLabel(a, copyDoneLabel)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, oldPtrOff)
	renvoAsmPopTertiary(a)
	renvoAsmLoadBytePrimaryIndexTertiary(a)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmCopyPrimaryToSecondary(a)
	renvoAsmPopTertiary(a)
	renvoAsmPopPrimary(a)
	renvoAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
	renvoAsmLoadPrimaryBss(a, copyIndexOff)
	renvoAsmIncPrimary(a)
	renvoAsmStorePrimaryBss(a, copyIndexOff)
	renvoAsmJmpMarkLabel(a, copyLoopLabel, copyDoneLabel)

	renvoAsmLoadPrimaryBss(a, ptrSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, capSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, newCapOff)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, lenSlotOff)
	renvoAsmPushPrimary(a)
	renvoAsmLoadPrimaryBss(a, oldLenOff)
	renvoAsmIncPrimary(a)
	renvoAsmPopSecondary(a)
	renvoAsmStorePrimaryMemSecondaryDisp(a, 0)
	renvoAsmLoadPrimaryBss(a, copySizeOff)
	renvoAsmCopyPrimaryToTertiary(a)
	renvoAsmLoadPrimaryBss(a, destOff)
	renvoAsmAddPrimaryTertiary(a)
	renvoAsmJmpLabel(a, returnLabel)

	renvoAsmMarkLabel(a, noGrowLabel)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitRegReg(a, renvoWasm32OpMulRegReg, renvoWasm32RegRcx, renvoWasm32RegRdx)
	renvoWasm32AsmAddRaxRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmMarkLabel(a, returnLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func renvoWasm32EnsureAppend8Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append8Label)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRdx, renvoWasm32RegRax, renvoWasm32RegRcx, 1, 0, 1)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func renvoWasm32EnsureAppend64Helper(g *renvoLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.append64Label)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegRax, renvoWasm32RegRdi, 0, 4)
	renvoWasm32EmitIndex(a, renvoWasm32OpStoreIndex, renvoWasm32RegRdx, renvoWasm32RegRax, renvoWasm32RegRcx, 8, 0, 4)
	renvoWasm32AsmIncRcx(a)
	renvoWasm32EmitMem(a, renvoWasm32OpStoreMem, renvoWasm32RegRcx, renvoWasm32RegRsi, 0, 4)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func renvoWasm32EnsureStringEqualHelper(g *renvoLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = renvoAsmNewLabel(a)
	afterLabel := renvoAsmNewLabel(a)
	notEqualLabel := renvoAsmNewLabel(a)
	equalLabel := renvoAsmNewLabel(a)
	loopLabel := renvoAsmNewLabel(a)
	renvoAsmJmpMarkLabel(a, afterLabel, g.streqLabel)
	renvoAsmPrimaryImm(a, 0)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegRsi, renvoWasm32RegRcx)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpCmpRegImm, renvoWasm32RegRsi, 0)
	renvoAsmJzLabel(a, equalLabel)
	renvoAsmMarkLabel(a, loopLabel)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR8, renvoWasm32RegRdi, 0, 1)
	renvoWasm32EmitMem(a, renvoWasm32OpLoadMem, renvoWasm32RegR9, renvoWasm32RegRdx, 0, 1)
	renvoWasm32EmitRegReg(a, renvoWasm32OpCmpRegReg, renvoWasm32RegR8, renvoWasm32RegR9)
	renvoAsmJnzLabel(a, notEqualLabel)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRdi, 1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRdx, 1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpAddRegImm, renvoWasm32RegRsi, -1)
	renvoWasm32EmitRegImm(a, renvoWasm32OpCmpRegImm, renvoWasm32RegRsi, 0)
	renvoAsmJnzLabel(a, loopLabel)
	renvoAsmMarkLabel(a, equalLabel)
	renvoAsmPrimaryImm(a, 1)
	renvoAsmMarkLabel(a, notEqualLabel)
	renvoAsmRet(a)
	renvoAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
