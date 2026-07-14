package main

func rtgWasmAppendU32(out []byte, v int) []byte {
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

func rtgWasmAppendS32(out []byte, v int) []byte {
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

func rtgWasmAppendName(out []byte, name string) []byte {
	out = rtgWasmAppendU32(out, len(name))
	for i := 0; i < len(name); i++ {
		out = append(out, name[i])
	}
	return out
}

func rtgWasmAppendByteVec(out []byte, data []byte) []byte {
	out = rtgWasmAppendU32(out, len(data))
	for i := 0; i < len(data); i++ {
		out = append(out, data[i])
	}
	return out
}

func rtgWasmAppendEncoded(out []byte, encoded string) []byte {
	for i := 0; i < len(encoded); i++ {
		out = append(out, encoded[i])
	}
	return out
}

func rtgWasmAppendRecipe(out []byte, recipe string, p0 int, p1 int, p2 int, p3 int) []byte {
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
			out = rtgWasmAppendI32Const(out, value)
		} else if command == 1 {
			out = rtgWasmLocalGet(out, value)
		} else if command == 2 {
			out = rtgWasmLocalSet(out, value)
		} else if command == 3 {
			out = rtgWasmLocalTee(out, value)
		} else if command == 4 {
			out = rtgWasm32RegGet(out, value)
		} else if command == 5 {
			out = rtgWasm32RegSet(out, value)
		} else if command == 6 {
			out = rtgWasm32StackAddr(out, value)
		} else if command == 7 {
			out = rtgWasm32LoadSized(out, value)
		} else if command == 8 {
			out = rtgWasm32StoreSized(out, value)
		} else if command == 9 {
			out = rtgWasm32AppendCond(out, value)
		} else if command == 10 {
			out = rtgWasm32AppendBinaryOp(out, value)
		}
		i += 2
	}
	return out
}

func rtgWasmAppendSection(out []byte, id int, payload []byte) []byte {
	out = append(out, byte(id))
	out = rtgWasmAppendU32(out, len(payload))
	for i := 0; i < len(payload); i++ {
		out = append(out, payload[i])
	}
	return out
}

func rtgWasmAppendI32Const(out []byte, value int) []byte {
	out = append(out, 0x41)
	value = rtgWasm32Sign32(value)
	out = rtgWasmAppendS32(out, value)
	return out
}

func rtgWasmAppendCall(out []byte, index int) []byte {
	out = append(out, 0x10)
	out = rtgWasmAppendU32(out, index)
	return out
}

func rtgWasmAppendU32Fixed5(out []byte, v int) []byte {
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

func rtgWasmPatchU32Fixed5(out []byte, at int, v int) {
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

func rtgWasmAppendI32Store(out []byte) []byte {
	out = append(out, 0x36)
	out = rtgWasmAppendU32(out, 2)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasmAppend2(out []byte, a int, b int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	return out
}

func rtgWasmAppend3(out []byte, a int, b int, c int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	return out
}

func rtgWasmAppend4(out []byte, a int, b int, c int, d int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	out = append(out, byte(d))
	return out
}

func rtgWasmAppend5(out []byte, a int, b int, c int, d int, e int) []byte {
	out = append(out, byte(a))
	out = append(out, byte(b))
	out = append(out, byte(c))
	out = append(out, byte(d))
	out = append(out, byte(e))
	return out
}

const rtgWasm32RegRax = 0
const rtgWasm32RegRdx = 1
const rtgWasm32RegRcx = 2
const rtgWasm32RegRdi = 3
const rtgWasm32RegRsi = 4
const rtgWasm32RegR8 = 5
const rtgWasm32RegR9 = 6
const rtgWasm32RegR10 = 7

const rtgWasm32OpExit = 1
const rtgWasm32OpBuildArgsEnv = 2
const rtgWasm32OpMovRegImm = 3
const rtgWasm32OpMovRegReg = 4
const rtgWasm32OpPushReg = 5
const rtgWasm32OpPushImm = 6
const rtgWasm32OpPopReg = 7
const rtgWasm32OpLoadStack = 8
const rtgWasm32OpStoreStack = 9
const rtgWasm32OpLeaStack = 10
const rtgWasm32OpLoadMem = 11
const rtgWasm32OpStoreMem = 12
const rtgWasm32OpLoadIndex = 13
const rtgWasm32OpStoreIndex = 14
const rtgWasm32OpAddRegReg = 15
const rtgWasm32OpSubRegReg = 16
const rtgWasm32OpMulRegReg = 17
const rtgWasm32OpDivRegReg = 18
const rtgWasm32OpModRegReg = 19
const rtgWasm32OpAndRegReg = 20
const rtgWasm32OpOrRegReg = 21
const rtgWasm32OpXorRegReg = 22
const rtgWasm32OpAndNotRegReg = 23
const rtgWasm32OpShlRegReg = 24
const rtgWasm32OpShrRegReg = 25
const rtgWasm32OpAddRegImm = 26
const rtgWasm32OpMulRegImm = 27
const rtgWasm32OpIncReg = 28
const rtgWasm32OpIncMem = 29
const rtgWasm32OpDecMem = 30
const rtgWasm32OpBoolNot = 31
const rtgWasm32OpNegReg = 32
const rtgWasm32OpCmpRegImm = 33
const rtgWasm32OpCmpRegReg = 34
const rtgWasm32OpSetCond = 35
const rtgWasm32OpJmp = 36
const rtgWasm32OpJz = 37
const rtgWasm32OpJnz = 38
const rtgWasm32OpJCond = 39
const rtgWasm32OpCall = 40
const rtgWasm32OpRet = 41
const rtgWasm32OpSyscall = 42
const rtgWasm32OpNop = 43

const rtgWasm32CondEq = 0
const rtgWasm32CondNe = 1
const rtgWasm32CondLt = 2
const rtgWasm32CondLe = 3
const rtgWasm32CondGt = 4
const rtgWasm32CondGe = 5

func rtgWasm32EmitRegImm(a *rtgAsm, op int, reg int, imm int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = rtgAppend32(a.code, imm)
}

func rtgWasm32EmitRegReg(a *rtgAsm, op int, dst int, src int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(dst))
	a.code = append(a.code, byte(src))
}

func rtgWasm32EmitReg(a *rtgAsm, op int, reg int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
}

func rtgWasm32EmitStack(a *rtgAsm, op int, reg int, offset int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = rtgAppend32(a.code, offset)
}

func rtgWasm32EmitMem(a *rtgAsm, op int, reg int, base int, disp int, size int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = append(a.code, byte(base))
	a.code = rtgAppend32(a.code, disp)
	a.code = append(a.code, byte(size))
}

func rtgWasm32EmitIndex(a *rtgAsm, op int, reg int, base int, index int, scale int, disp int, size int) {
	a.code = append(a.code, byte(op))
	a.code = append(a.code, byte(reg))
	a.code = append(a.code, byte(base))
	a.code = append(a.code, byte(index))
	a.code = append(a.code, byte(scale))
	a.code = rtgAppend32(a.code, disp)
	a.code = append(a.code, byte(size))
}

func rtgWasm32EmitBranch(a *rtgAsm, op int, label int) {
	a.code = append(a.code, byte(op))
	at := len(a.code)
	a.code = rtgAppend32(a.code, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgWasm32EmitCondBranch(a *rtgAsm, cond int, label int) {
	a.code = append(a.code, byte(rtgWasm32OpJCond))
	a.code = append(a.code, byte(cond))
	at := len(a.code)
	a.code = rtgAppend32(a.code, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgWasm32EmitCallLabel(a *rtgAsm, label int, wordCount int) {
	a.code = append(a.code, byte(rtgWasm32OpCall))
	at := len(a.code)
	a.code = rtgAppend32(a.code, 0)
	rtgAsmAddReloc(a, at, label)
	a.code = rtgAppend32(a.code, wordCount)
}

func rtgWasm32StoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRdi, offset)
		return true
	}
	if reg == 1 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRsi, offset)
		return true
	}
	if reg == 2 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRdx, offset)
		return true
	}
	if reg == 3 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRcx, offset)
		return true
	}
	if reg == 4 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegR8, offset)
		return true
	}
	if reg == 5 {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegR9, offset)
		return true
	}
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRax)
	rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRax, offset)
	return true
}

func rtgWasm32AsmMovRaxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, imm)
}

func rtgWasm32AsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgWasm32AsmMovRaxImm(a, imm)
}

func rtgWasm32AsmMovRdxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, imm)
}

func rtgWasm32AsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, 0)
	rtgAsmAddAbsReloc(a, len(a.code)-4, dataOff, 0)
}

func rtgWasm32AsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRax, 0)
	rtgAsmAddAbsReloc(a, len(a.code)-4, bssOff, rtgAbsBssReloc)
}

func rtgWasm32AsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegR10, 0)
	rtgAsmAddAbsReloc(a, len(a.code)-4, bssOff, rtgAbsBssReloc)
}

func rtgWasm32AsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 0)
	rtgAsmAddAbsReloc(a, len(a.code)-4, bssOff, rtgAbsBssReloc)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdx, 0, 4)
}

func rtgWasm32AsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 0)
	rtgAsmAddAbsReloc(a, len(a.code)-4, bssOff, rtgAbsBssReloc)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRdx, 0, 4)
}

func rtgWasm32AsmMovRdiRax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdi, rtgWasm32RegRax)
}

func rtgWasm32AsmMovRaxRdx(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
}

func rtgWasm32AsmMovRdxRax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRax)
}

func rtgWasm32AsmMovRcxRax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
}

func rtgWasm32AsmMovRcxRdx(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRcx, rtgWasm32RegRdx)
}

func rtgWasm32AsmMovRsiRax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRsi, rtgWasm32RegRax)
}

func rtgWasm32AsmMovR8Rax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR8, rtgWasm32RegRax)
}

func rtgWasm32AsmMovR9Rax(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR9, rtgWasm32RegRax)
}

func rtgWasm32AsmAddRdxRcx(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpAddRegReg, rtgWasm32RegRdx, rtgWasm32RegRcx)
}

func rtgWasm32AsmSyscall(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpSyscall)
}

func rtgWasm32AsmPushRax(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPushReg, rtgWasm32RegRax)
}

func rtgWasm32AsmPushRcx(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPushReg, rtgWasm32RegRcx)
}

func rtgWasm32AsmPushRdx(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPushReg, rtgWasm32RegRdx)
}

func rtgWasm32AsmPushImm(a *rtgAsm, imm int) {
	rtgAsmEmit8(a, rtgWasm32OpPushImm)
	rtgAsmEmit32(a, imm)
}

func rtgWasm32AsmPopRax(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRax)
}

func rtgWasm32AsmPopRcx(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRcx)
}

func rtgWasm32AsmPopRdx(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRdx)
}

func rtgWasm32AsmPopRdi(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRdi)
}

func rtgWasm32AsmPopRsi(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegRsi)
}

func rtgWasm32AsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	op := base & 0xff00
	regCode := disp8
	reg := rtgWasm32RegRax
	if base == 0x894c || base == 0x8b4c {
		if regCode == 0x4d || regCode == 0x8d {
			reg = rtgWasm32RegR9
		} else {
			reg = rtgWasm32RegR8
		}
	} else if regCode == 0x55 || regCode == 0x95 {
		reg = rtgWasm32RegRdx
	} else if regCode == 0x4d || regCode == 0x8d {
		reg = rtgWasm32RegRcx
	} else if regCode == 0x7d || regCode == 0xbd {
		reg = rtgWasm32RegRdi
	} else if regCode == 0x75 || regCode == 0xb5 {
		reg = rtgWasm32RegRsi
	}
	if op == 0x8900 || op == 0x8948 || op == 0x894c {
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, reg, offset)
		return
	}
	if op == 0x8b00 || op == 0x8b48 || op == 0x8b4c {
		rtgWasm32EmitStack(a, rtgWasm32OpLoadStack, reg, offset)
		return
	}
	rtgWasm32EmitStack(a, rtgWasm32OpLeaStack, reg, offset)
}

func rtgWasm32AsmAddRdxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdx, imm)
}

func rtgWasm32AsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdx, disp, 4)
}

func rtgWasm32AsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdx, disp, size)
}

func rtgWasm32AsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgWasm32EmitIndex(a, rtgWasm32OpLoadIndex, rtgWasm32RegRax, rtgWasm32RegRax, rtgWasm32RegRcx, 1, 0, 1)
}

func rtgWasm32AsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	scale := size
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitIndex(a, rtgWasm32OpLoadIndex, rtgWasm32RegRax, rtgWasm32RegRax, rtgWasm32RegRcx, scale, 0, size)
}

func rtgWasm32AsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgWasm32EmitIndex(a, rtgWasm32OpLoadIndex, rtgWasm32RegRax, rtgWasm32RegRax, rtgWasm32RegRcx, 8, 0, 4)
}

func rtgWasm32AsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgWasm32EmitIndex(a, rtgWasm32OpLoadIndex, rtgWasm32RegRax, rtgWasm32RegRax, rtgWasm32RegRcx, 1, disp, 4)
}

func rtgWasm32AsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgWasm32EmitIndex(a, rtgWasm32OpStoreIndex, rtgWasm32RegRax, rtgWasm32RegRdx, rtgWasm32RegRcx, 8, 0, 4)
}

func rtgWasm32AsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRdx, disp, 4)
}

func rtgWasm32AsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRdx, disp, size)
}

func rtgWasm32AsmStoreRaxMemRdxRcxSize(a *rtgAsm, size int) {
	scale := size
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitIndex(a, rtgWasm32OpStoreIndex, rtgWasm32RegRax, rtgWasm32RegRdx, rtgWasm32RegRcx, scale, 0, size)
}

func rtgWasm32AsmStoreSliceStack(a *rtgAsm, offset int) {
	rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRax, offset)
	rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRdx, offset-8)
	rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRcx, offset-16)
}

func rtgWasm32AsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 255)
		rtgWasm32EmitRegReg(a, rtgWasm32OpAndRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		return
	}
	if kind == rtgTypeInt8 {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 24)
		rtgWasm32EmitRegReg(a, rtgWasm32OpShlRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 24)
		rtgWasm32EmitRegReg(a, rtgWasm32OpShrRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		return
	}
	if kind == rtgTypeInt16 {
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 16)
		rtgWasm32EmitRegReg(a, rtgWasm32OpShlRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, 16)
		rtgWasm32EmitRegReg(a, rtgWasm32OpShrRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
	}
}

func rtgWasm32AsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpIncMem)
	rtgAsmEmit8(a, rtgWasm32RegRdx)
}

func rtgWasm32AsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpDecMem)
	rtgAsmEmit8(a, rtgWasm32RegRdx)
}

func rtgWasm32AsmBoolNotRax(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpBoolNot, rtgWasm32RegRax)
}

func rtgWasm32AsmCmpRaxImm8(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegRax, imm)
}

func rtgWasm32AsmAddRaxRcx(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpAddRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
}

func rtgWasm32AsmSubRaxRcx(a *rtgAsm) {
	rtgWasm32EmitRegReg(a, rtgWasm32OpSubRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
}

func rtgWasm32AsmShlRcxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, imm)
	rtgWasm32EmitRegReg(a, rtgWasm32OpShlRegReg, rtgWasm32RegRcx, rtgWasm32RegRdx)
}

func rtgWasm32AsmShlRaxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, imm)
	rtgWasm32EmitRegReg(a, rtgWasm32OpShlRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
}

func rtgWasm32AsmSarRaxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRdx, imm)
	rtgWasm32EmitRegReg(a, rtgWasm32OpShrRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
}

func rtgWasm32AsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	if mod {
		rtgWasm32EmitRegReg(a, rtgWasm32OpModRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
	} else {
		rtgWasm32EmitRegReg(a, rtgWasm32OpDivRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
	}
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
}

func rtgWasm32AsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	cond := rtgWasm32CondEq
	if setcc == 0x95 {
		cond = rtgWasm32CondNe
	} else if setcc == 0x9c {
		cond = rtgWasm32CondLt
	} else if setcc == 0x9e {
		cond = rtgWasm32CondLe
	} else if setcc == 0x9f {
		cond = rtgWasm32CondGt
	} else if setcc == 0x9d {
		cond = rtgWasm32CondGe
	}
	rtgWasm32EmitRegReg(a, rtgWasm32OpCmpRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
	rtgAsmEmit8(a, rtgWasm32OpSetCond)
	rtgAsmEmit8(a, cond)
}

func rtgWasm32AsmIncRcx(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpIncReg, rtgWasm32RegRcx)
}

func rtgWasm32AsmIncRax(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpIncReg, rtgWasm32RegRax)
}

func rtgWasm32AsmImulRcxImm(a *rtgAsm, imm int) {
	rtgWasm32EmitRegImm(a, rtgWasm32OpMulRegImm, rtgWasm32RegRcx, imm)
}

func rtgWasm32AsmLeave(a *rtgAsm) {
}

func rtgWasm32AsmRet(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpRet)
}

func rtgWasm32AsmCallLabel(a *rtgAsm, label int) {
	rtgWasm32EmitCallLabel(a, label, 0)
}

func rtgWasm32AsmJmpLabel(a *rtgAsm, label int) {
	rtgWasm32EmitBranch(a, rtgWasm32OpJmp, label)
}

func rtgWasm32AsmJzLabel(a *rtgAsm, label int) {
	rtgWasm32EmitBranch(a, rtgWasm32OpJz, label)
}

func rtgWasm32AsmJnzLabel(a *rtgAsm, label int) {
	rtgWasm32EmitBranch(a, rtgWasm32OpJnz, label)
}

func rtgWasm32AsmNegRax(a *rtgAsm) {
	rtgWasm32EmitReg(a, rtgWasm32OpNegReg, rtgWasm32RegRax)
}

func rtgWasm32AsmBuildArgvEnvSlices(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	rtgAsmEmit8(a, rtgWasm32OpBuildArgsEnv)
	at := len(a.code)
	rtgAsmEmit32(a, bssOff)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
	at = len(a.code)
	rtgAsmEmit32(a, envOff)
	rtgAsmAddAbsReloc(a, at, envOff, rtgAbsBssReloc)
	at = len(a.code)
	rtgAsmEmit32(a, envLenOff)
	rtgAsmAddAbsReloc(a, at, envLenOff, rtgAbsBssReloc)
}

func rtgWasm32AsmExit(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpExit)
}

type rtgWasm32Instr struct {
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

const rtgWasm32ProgramBase = 262144
const rtgWasm32StackGuardSize = 16384
const rtgWasm32ExprStackSize = 1048576
const rtgWasm32CallStackSize = 65536
const rtgWasm32FrameStackSize = 8388608
const rtgWasm32ImageOutputCapacity = 2097152
const rtgWasm32RoutineBodyCapacity = 8192
const rtgWasm32ScratchIov = 0
const rtgWasm32ScratchN = 8
const rtgWasm32ScratchFd = 12
const rtgWasm32ArgsCountPtr = 16
const rtgWasm32ArgsSizePtr = 20
const rtgWasm32EnvCountPtr = 24
const rtgWasm32EnvSizePtr = 28
const rtgWasm32FdstatScratch = 64
const rtgWasm32ArgsPtrArea = 4096
const rtgWasm32ArgsDataArea = 8192
const rtgWasm32EnvPtrArea = 65536
const rtgWasm32EnvDataArea = 131072

const rtgWasm32ImportFdWrite = 0
const rtgWasm32ImportFdRead = 1
const rtgWasm32ImportFdPread = 2
const rtgWasm32ImportFdPwrite = 3
const rtgWasm32ImportPathOpen = 4
const rtgWasm32ImportFdClose = 5
const rtgWasm32ImportFdstatGet = 6
const rtgWasm32ImportFdReaddir = 7
const rtgWasm32ImportArgsSizesGet = 8
const rtgWasm32ImportArgsGet = 9
const rtgWasm32ImportEnvironSizesGet = 10
const rtgWasm32ImportEnvironGet = 11
const rtgWasm32ImportProcExit = 12
const rtgWasm32StartFuncIndex = 13
const rtgWasm32VmFuncType = 7
const rtgWasm32VmFuncBase = 14

const rtgWasm32LocalSp = 0
const rtgWasm32LocalFp = 1
const rtgWasm32LocalRax = 2
const rtgWasm32LocalRdx = 3
const rtgWasm32LocalRcx = 4
const rtgWasm32LocalRdi = 5
const rtgWasm32LocalRsi = 6
const rtgWasm32LocalR8 = 7
const rtgWasm32LocalR9 = 8
const rtgWasm32LocalPc = 9
const rtgWasm32LocalCsp = 10
const rtgWasm32LocalR10 = 11
const rtgWasm32LocalFlag = 12
const rtgWasm32LocalTmp = 13
const rtgWasm32LocalTmp2 = 14
const rtgWasm32LocalTmp3 = 15

func rtgWasm32RegLocal(reg int) int {
	if reg == rtgWasm32RegRdx {
		return rtgWasm32LocalRdx
	}
	if reg == rtgWasm32RegRcx {
		return rtgWasm32LocalRcx
	}
	if reg == rtgWasm32RegRdi {
		return rtgWasm32LocalRdi
	}
	if reg == rtgWasm32RegRsi {
		return rtgWasm32LocalRsi
	}
	if reg == rtgWasm32RegR8 {
		return rtgWasm32LocalR8
	}
	if reg == rtgWasm32RegR9 {
		return rtgWasm32LocalR9
	}
	if reg == rtgWasm32RegR10 {
		return rtgWasm32LocalR10
	}
	return rtgWasm32LocalRax
}

func rtgWasm32Sign32(v int) int {
	if v > 2147483647 {
		v -= 2147483647
		v -= 2147483647
		v -= 2
	}
	return v
}

func rtgWasm32GetS32(in []byte, at int) int {
	return rtgWasm32Sign32(rtgGet32At(in, at))
}

func rtgWasm32NextInstructionPc(code []byte, pc int) int {
	op := int(code[pc])
	if op == rtgWasm32OpMovRegImm || op == rtgWasm32OpLoadStack || op == rtgWasm32OpStoreStack || op == rtgWasm32OpLeaStack || op == rtgWasm32OpAddRegImm || op == rtgWasm32OpMulRegImm || op == rtgWasm32OpCmpRegImm || op == rtgWasm32OpJCond {
		return pc + 6
	}
	if op == rtgWasm32OpMovRegReg || op == rtgWasm32OpAddRegReg || op == rtgWasm32OpSubRegReg || op == rtgWasm32OpMulRegReg || op == rtgWasm32OpDivRegReg || op == rtgWasm32OpModRegReg || op == rtgWasm32OpAndRegReg || op == rtgWasm32OpOrRegReg || op == rtgWasm32OpXorRegReg || op == rtgWasm32OpAndNotRegReg || op == rtgWasm32OpShlRegReg || op == rtgWasm32OpShrRegReg || op == rtgWasm32OpCmpRegReg {
		return pc + 3
	}
	if op == rtgWasm32OpPushReg || op == rtgWasm32OpPopReg || op == rtgWasm32OpIncReg || op == rtgWasm32OpIncMem || op == rtgWasm32OpDecMem || op == rtgWasm32OpBoolNot || op == rtgWasm32OpNegReg || op == rtgWasm32OpSetCond {
		return pc + 2
	}
	if op == rtgWasm32OpPushImm || op == rtgWasm32OpJmp || op == rtgWasm32OpJz || op == rtgWasm32OpJnz {
		return pc + 5
	}
	if op == rtgWasm32OpLoadMem || op == rtgWasm32OpStoreMem {
		return pc + 8
	}
	if op == rtgWasm32OpLoadIndex || op == rtgWasm32OpStoreIndex {
		return pc + 10
	}
	if op == rtgWasm32OpCall {
		return pc + 9
	}
	if op == rtgWasm32OpBuildArgsEnv {
		return pc + 13
	}
	return pc + 1
}

func rtgWasm32DecodeOne(code []byte, pc int, next int) rtgWasm32Instr {
	var ins rtgWasm32Instr
	op := int(code[pc])
	ins.pc = pc
	ins.op = op
	ins.next = next
	if op == rtgWasm32OpMovRegImm {
		ins.a = int(code[pc+1])
		ins.b = rtgWasm32GetS32(code, pc+2)
		return ins
	}
	if op == rtgWasm32OpMovRegReg || op == rtgWasm32OpAddRegReg || op == rtgWasm32OpSubRegReg || op == rtgWasm32OpMulRegReg || op == rtgWasm32OpDivRegReg || op == rtgWasm32OpModRegReg || op == rtgWasm32OpAndRegReg || op == rtgWasm32OpOrRegReg || op == rtgWasm32OpXorRegReg || op == rtgWasm32OpAndNotRegReg || op == rtgWasm32OpShlRegReg || op == rtgWasm32OpShrRegReg || op == rtgWasm32OpCmpRegReg {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		return ins
	}
	if op == rtgWasm32OpPushReg || op == rtgWasm32OpPopReg || op == rtgWasm32OpIncReg || op == rtgWasm32OpIncMem || op == rtgWasm32OpDecMem || op == rtgWasm32OpBoolNot || op == rtgWasm32OpNegReg || op == rtgWasm32OpSetCond {
		ins.a = int(code[pc+1])
		return ins
	}
	if op == rtgWasm32OpPushImm {
		ins.a = rtgWasm32GetS32(code, pc+1)
		return ins
	}
	if op == rtgWasm32OpLoadStack || op == rtgWasm32OpStoreStack || op == rtgWasm32OpLeaStack || op == rtgWasm32OpAddRegImm || op == rtgWasm32OpMulRegImm || op == rtgWasm32OpCmpRegImm {
		ins.a = int(code[pc+1])
		ins.b = rtgWasm32GetS32(code, pc+2)
		return ins
	}
	if op == rtgWasm32OpLoadMem || op == rtgWasm32OpStoreMem {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = rtgWasm32GetS32(code, pc+3)
		ins.d = int(code[pc+7])
		return ins
	}
	if op == rtgWasm32OpLoadIndex || op == rtgWasm32OpStoreIndex {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = int(code[pc+3])
		ins.d = int(code[pc+4])
		ins.e = rtgWasm32GetS32(code, pc+5)
		ins.f = int(code[pc+9])
		return ins
	}
	if op == rtgWasm32OpJmp || op == rtgWasm32OpJz || op == rtgWasm32OpJnz {
		ins.a = rtgGet32At(code, pc+1)
		return ins
	}
	if op == rtgWasm32OpCall {
		ins.a = rtgGet32At(code, pc+1)
		ins.b = rtgGet32At(code, pc+5)
		return ins
	}
	if op == rtgWasm32OpJCond {
		ins.a = int(code[pc+1])
		ins.b = rtgGet32At(code, pc+2)
		return ins
	}
	if op == rtgWasm32OpBuildArgsEnv {
		ins.a = rtgGet32At(code, pc+1)
		ins.b = rtgGet32At(code, pc+5)
		ins.c = rtgGet32At(code, pc+9)
		return ins
	}
	return ins
}

func rtgWasm32InstructionPcs(code []byte) []int {
	out := make([]int, 0, 131072)
	pc := 0
	for pc < len(code) {
		out = append(out, pc)
		pc = rtgWasm32NextInstructionPc(code, pc)
	}
	return out
}

func rtgWasm32DecodePcRange(code []byte, pcs []int) []rtgWasm32Instr {
	out := make([]rtgWasm32Instr, 0, len(pcs))
	for i := 0; i < len(pcs); i++ {
		next := rtgWasm32NextInstructionPc(code, pcs[i])
		if i+1 < len(pcs) {
			next = pcs[i+1]
		}
		out = append(out, rtgWasm32DecodeOne(code, pcs[i], next))
	}
	return out
}

func rtgWasm32PcLowerBound(pcs []int, pc int) int {
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

func rtgWasm32InstrLowerBound(instrs []rtgWasm32Instr, pc int) int {
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

func rtgWasm32InstrIndexForPcLocal(instrs []rtgWasm32Instr, instrCount int, pc int) int {
	idx := rtgWasm32InstrLowerBound(instrs, pc)
	if idx < instrCount {
		if int(instrs[idx].pc) == pc {
			return idx
		}
	}
	return instrCount
}

func rtgWasm32IsControlOp(op int) bool {
	if op == rtgWasm32OpExit || op == rtgWasm32OpBuildArgsEnv {
		return true
	}
	if op == rtgWasm32OpJmp || op == rtgWasm32OpJz || op == rtgWasm32OpJnz || op == rtgWasm32OpJCond {
		return true
	}
	if op == rtgWasm32OpCall {
		return true
	}
	if op == rtgWasm32OpRet {
		return true
	}
	return false
}

func rtgWasm32OpHasTarget(op int) bool {
	if op == rtgWasm32OpJmp {
		return true
	}
	if op == rtgWasm32OpJz {
		return true
	}
	if op == rtgWasm32OpJnz {
		return true
	}
	if op == rtgWasm32OpCall {
		return true
	}
	return false
}

func rtgWasm32BuildBlockStartsLocal(instrs []rtgWasm32Instr) []int {
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
		if rtgWasm32IsControlOp(op) {
			if i+1 < instrCount {
				marks[i+1] = 1
			}
		}
		if rtgWasm32OpHasTarget(op) {
			targetIndex := rtgWasm32InstrIndexForPcLocal(instrs, instrCount, int(ins.a))
			if targetIndex < instrCount {
				marks[targetIndex] = 1
			}
		}
		if op == rtgWasm32OpJCond {
			targetIndex := rtgWasm32InstrIndexForPcLocal(instrs, instrCount, int(ins.b))
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

func rtgWasm32BlockEnd(starts []int, blockIndex int, instrCount int) int {
	if blockIndex+1 < len(starts) {
		return starts[blockIndex+1]
	}
	return instrCount
}

func rtgWasm32BuildInstrBlockIndex(starts []int, instrCount int) []int {
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

func rtgWasm32BlockForInstrFast(blockIndex []int, idx int) int {
	if idx >= 0 && idx < len(blockIndex) {
		return blockIndex[idx]
	}
	return len(blockIndex) - 1
}

func rtgWasmLocalGet(out []byte, local int) []byte {
	out = append(out, 0x20)
	out = rtgWasmAppendU32(out, local)
	return out
}

func rtgWasmLocalSet(out []byte, local int) []byte {
	out = append(out, 0x21)
	out = rtgWasmAppendU32(out, local)
	return out
}

func rtgWasmLocalTee(out []byte, local int) []byte {
	out = append(out, 0x22)
	out = rtgWasmAppendU32(out, local)
	return out
}

func rtgWasmI32Load(out []byte, align int, off int) []byte {
	out = append(out, 0x28)
	out = rtgWasmAppendU32(out, align)
	out = rtgWasmAppendU32(out, off)
	return out
}

func rtgWasmI32Load8U(out []byte) []byte {
	out = append(out, 0x2d)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasmI32Load16S(out []byte) []byte {
	out = append(out, 0x2e)
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasmI32Store(out []byte, align int, off int) []byte {
	out = append(out, 0x36)
	out = rtgWasmAppendU32(out, align)
	out = rtgWasmAppendU32(out, off)
	return out
}

func rtgWasmI32Store8(out []byte) []byte {
	out = append(out, 0x3a)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasmI32Store16(out []byte) []byte {
	out = append(out, 0x3b)
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendU32(out, 0)
	return out
}

func rtgWasmI64Const(out []byte, value int) []byte {
	out = append(out, 0x42)
	out = rtgWasmAppendS32(out, value)
	return out
}

func rtgWasmBr(out []byte, depth int) []byte {
	out = append(out, 0x0c)
	out = rtgWasmAppendU32(out, depth)
	return out
}

func rtgWasmBrIf(out []byte, depth int) []byte {
	out = append(out, 0x0d)
	out = rtgWasmAppendU32(out, depth)
	return out
}

func rtgWasm32SetPc(out []byte, pc int) []byte {
	out = rtgWasmAppendI32Const(out, pc)
	out = rtgWasmLocalSet(out, rtgWasm32LocalPc)
	return out
}

func rtgWasm32RegGet(out []byte, reg int) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32RegLocal(reg))
	return out
}

func rtgWasm32RegSet(out []byte, reg int) []byte {
	out = rtgWasmLocalSet(out, rtgWasm32RegLocal(reg))
	return out
}

func rtgWasm32StackAddr(out []byte, offset int) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32LocalFp)
	out = rtgWasmAppendI32Const(out, offset)
	out = append(out, 0x6b)
	return out
}

func rtgWasm32MemAddr(out []byte, base int, disp int) []byte {
	out = rtgWasm32RegGet(out, base)
	if disp != 0 {
		out = rtgWasmAppendI32Const(out, disp)
		out = append(out, 0x6a)
	}
	return out
}

func rtgWasm32IndexAddr(out []byte, base int, index int, scale int, disp int) []byte {
	out = rtgWasm32RegGet(out, base)
	out = rtgWasm32RegGet(out, index)
	if scale != 1 {
		out = rtgWasmAppendI32Const(out, scale)
		out = append(out, 0x6c)
	}
	out = append(out, 0x6a)
	if disp != 0 {
		out = rtgWasmAppendI32Const(out, disp)
		out = append(out, 0x6a)
	}
	return out
}

func rtgWasm32LoadSized(out []byte, size int) []byte {
	if size == 1 {
		out = rtgWasmI32Load8U(out)
		return out
	}
	if size == 2 {
		out = rtgWasmI32Load16S(out)
		return out
	}
	out = rtgWasmI32Load(out, 2, 0)
	return out
}

func rtgWasm32StoreSized(out []byte, size int) []byte {
	if size == 1 {
		out = rtgWasmI32Store8(out)
		return out
	}
	if size == 2 {
		out = rtgWasmI32Store16(out)
		return out
	}
	out = rtgWasmI32Store(out, 2, 0)
	return out
}

func rtgWasm32AppendCond(out []byte, cond int) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32LocalFlag)
	out = rtgWasmAppendI32Const(out, 0)
	if cond == rtgWasm32CondNe {
		out = append(out, 0x47)
	} else if cond == rtgWasm32CondLt {
		out = append(out, 0x48)
	} else if cond == rtgWasm32CondLe {
		out = append(out, 0x4c)
	} else if cond == rtgWasm32CondGt {
		out = append(out, 0x4a)
	} else if cond == rtgWasm32CondGe {
		out = append(out, 0x4e)
	} else {
		out = append(out, 0x46)
	}
	return out
}

func rtgWasm32Patch(a *rtgAsm, dataBase int, bssBase int) {
	for i := 0; i < len(a.relocs); i++ {
		r := a.relocs[i]
		if r.label >= 0 && r.label < len(a.labelPos) && a.labelSet[r.label] {
			rtgPut32At(a.code, r.at, a.labelPos[r.label])
		}
	}
	for i := 0; i < len(a.absRelocs); i++ {
		r := a.absRelocs[i]
		target := dataBase + r.off
		if r.kind == rtgAbsBssReloc {
			target = bssBase + r.off
		}
		rtgPut32At(a.code, r.at, target)
	}
}

func rtgWasm32AppendStoreConst(out []byte, addr int, value int) []byte {
	out = rtgWasmAppendI32Const(out, addr)
	out = rtgWasmAppendI32Const(out, value)
	out = rtgWasmI32Store(out, 2, 0)
	return out
}

func rtgWasm32AppendIov(out []byte, ptrLocal int, lenLocal int) []byte {
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov)
	out = rtgWasmLocalGet(out, ptrLocal)
	out = rtgWasmI32Store(out, 2, 0)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov+4)
	out = rtgWasmLocalGet(out, lenLocal)
	out = rtgWasmI32Store(out, 2, 0)
	return out
}

func rtgWasm32AppendErrnoResult(out []byte, resultPtr int) []byte {
	out = rtgWasmLocalTee(out, rtgWasm32LocalTmp)
	out = append(out, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmAppendI32Const(out, resultPtr)
	out = rtgWasmI32Load(out, 2, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = append(out, 0x05)
	out = rtgWasmAppendI32Const(out, -1)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func rtgWasm32AppendErrnoOnlyResult(out []byte) []byte {
	out = rtgWasmLocalTee(out, rtgWasm32LocalTmp)
	out = append(out, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = append(out, 0x05)
	out = rtgWasmAppendI32Const(out, -1)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func rtgWasm32AppendSyscall(out []byte) []byte {
	return rtgWasmAppendEncoded(out, "\x20\x02\x41\x01\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x41\x08\x10\x00\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x00\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x41\x08\x10\x01\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x12\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x20\x0b\xac\x41\x08\x10\x03\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x11\x46\x04\x40\x41\x00\x20\x06\x36\x02\x00\x41\x04\x20\x03\x36\x02\x00\x20\x05\x41\x00\x41\x01\x20\x0b\xac\x41\x08\x10\x02\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\x03\x46\x04\x40\x20\x05\x10\x05\x22\x0d\x45\x04\x40\x41\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\xdb\x00\x46\x04\x40\x20\x05\x41\xc0\x00\x10\x06\x22\x0d\x45\x04\x40\x41\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x02\x41\xd9\x01\x46\x04\x40\x20\x05\x20\x06\x20\x03\x42\x00\x41\x08\x10\x07\x22\x0d\x45\x04\x40\x41\x08\x28\x02\x00\x21\x02\x05\x41\x7f\x21\x02\x0b\x05\x20\x03\x41\x00\x4b\x04\x40\x20\x05\x20\x03\x41\x01\x6b\x6a\x2d\x00\x00\x45\x04\x40\x20\x03\x41\x01\x6b\x21\x03\x0b\x0b\x41\x00\x21\x0d\x20\x06\x41\xc0\x00\x71\x45\x45\x04\x40\x20\x0d\x41\x01\x72\x21\x0d\x0b\x20\x06\x41\x80\x04\x71\x45\x45\x04\x40\x20\x0d\x41\x08\x72\x21\x0d\x0b\x20\x05\x2d\x00\x00\x41\x2f\x46\x04\x40\x20\x05\x41\x01\x6a\x21\x05\x20\x03\x41\x01\x6b\x21\x03\x0b\x41\x03\x21\x0e\x02\x40\x03\x40\x20\x0e\x41\x00\x20\x05\x20\x03\x20\x0d\x20\x06\x41\x02\x71\x45\x04\x7e\x42\x82\x80\x01\x05\x42\xe6\x80\x80\x01\x0b\x42\x00\x41\x00\x41\x0c\x10\x04\x22\x0f\x45\x04\x40\x41\x0c\x28\x02\x00\x21\x02\x0c\x02\x0b\x20\x0e\x41\x01\x6a\x22\x0e\x41\x08\x4c\x0d\x00\x0b\x41\x7f\x21\x02\x0b\x0b\x0b\x0b\x0b\x0b\x0b\x0b")
}

func rtgWasm32AppendOpen(out []byte) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmAppendI32Const(out, 0)
	out = append(out, 0x4b)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmAppend2(out, 0x6b, 0x6a)
	out = rtgWasmI32Load8U(out)
	out = append(out, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6b)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdx)
	out = append(out, 0x0b)
	out = append(out, 0x0b)

	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRsi)
	out = rtgWasmAppendI32Const(out, 64)
	out = rtgWasmAppend3(out, 0x71, 0x45, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x72)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp)
	out = append(out, 0x0b)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRsi)
	out = rtgWasmAppendI32Const(out, 512)
	out = rtgWasmAppend3(out, 0x71, 0x45, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 8)
	out = append(out, 0x72)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp)
	out = append(out, 0x0b)

	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmI32Load8U(out)
	out = rtgWasmAppendI32Const(out, 47)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6b)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdx)
	out = append(out, 0x0b)

	out = rtgWasmAppendI32Const(out, 3)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp2)
	out = rtgWasmAppend2(out, 0x02, 0x40)
	out = rtgWasmAppend2(out, 0x03, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp2)
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRsi)
	out = rtgWasmAppendI32Const(out, 2)
	out = rtgWasmAppend2(out, 0x71, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x7e)
	out = rtgWasmI64Const(out, 16386)
	out = append(out, 0x05)
	out = rtgWasmI64Const(out, 2097254)
	out = append(out, 0x0b)
	out = rtgWasmI64Const(out, 0)
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchFd)
	out = rtgWasmAppendCall(out, rtgWasm32ImportPathOpen)
	out = rtgWasmLocalTee(out, rtgWasm32LocalTmp3)
	out = append(out, 0x45)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchFd)
	out = rtgWasmI32Load(out, 2, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = rtgWasmBr(out, 2)
	out = append(out, 0x0b)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp2)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = rtgWasmLocalTee(out, rtgWasm32LocalTmp2)
	out = rtgWasmAppendI32Const(out, 8)
	out = append(out, 0x4c)
	out = rtgWasmBrIf(out, 0)
	out = append(out, 0x0b)
	out = rtgWasmAppendI32Const(out, -1)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	out = append(out, 0x0b)
	return out
}

func rtgWasm32AppendStringSliceBuild(out []byte, ptrArea int, countLocal int, destAddr int) []byte {
	return rtgWasmAppendRecipe(out, "\x41\x00\x21\x0d\x02\x40\x03\x40\x20\x0d\xff\x01\x01\x4f\x0d\x01\xff\x00\x00\x20\x0d\x41\x04\x6c\x6a\x28\x02\x00\x21\x0e\x41\x00\x21\x0f\x02\x40\x03\x40\x20\x0e\x20\x0f\x6a\x2d\x00\x00\x45\x0d\x01\x20\x0f\x41\x01\x6a\x21\x0f\x0c\x00\x0b\x0b\xff\x00\x02\x20\x0d\x41\x10\x6c\x6a\x20\x0e\x36\x02\x00\xff\x00\x03\x20\x0d\x41\x10\x6c\x6a\x20\x0f\x36\x02\x00\x20\x0d\x41\x01\x6a\x21\x0d\x0c\x00\x0b\x0b", ptrArea, countLocal, destAddr, destAddr+8)
}

func rtgWasm32AppendBuildArgsEnv(out []byte, argsAddr int, envAddr int, envLenAddr int) []byte {
	out = rtgWasmAppendRecipe(out, "\x41\x10\x41\x14\x10\x08\x1a\x41\x80\x20\x41\x80\xc0\x00\x10\x09\x1a\x41\x10\x28\x02\x00\x21\x06\x20\x06\x21\x03\xff\x00\x00\x21\x05", argsAddr, 0, 0, 0)
	out = rtgWasm32AppendStringSliceBuild(out, rtgWasm32ArgsPtrArea, rtgWasm32LocalRsi, argsAddr)
	out = rtgWasmAppendRecipe(out, "\x41\x18\x41\x1c\x10\x0a\x1a\x41\x80\x80\x04\x41\x80\x80\x08\x10\x0b\x1a\x41\x18\x28\x02\x00\x21\x07\x20\x07\x21\x08\xff\x00\x01\x21\x04\xff\x00\x02\x20\x07\x36\x02\x00", 0, envAddr, envLenAddr, 0)
	out = rtgWasm32AppendStringSliceBuild(out, rtgWasm32EnvPtrArea, rtgWasm32LocalR8, envAddr)
	return out
}

func rtgWasm32AppendBinaryOp(out []byte, op int) []byte {
	if op == rtgWasm32OpAddRegReg {
		return append(out, 0x6a)
	}
	if op == rtgWasm32OpSubRegReg {
		return append(out, 0x6b)
	}
	if op == rtgWasm32OpMulRegReg {
		return append(out, 0x6c)
	}
	if op == rtgWasm32OpDivRegReg {
		return append(out, 0x6d)
	}
	if op == rtgWasm32OpModRegReg {
		return append(out, 0x6f)
	}
	if op == rtgWasm32OpAndRegReg {
		return append(out, 0x71)
	}
	if op == rtgWasm32OpOrRegReg {
		return append(out, 0x72)
	}
	if op == rtgWasm32OpXorRegReg {
		return append(out, 0x73)
	}
	if op == rtgWasm32OpShlRegReg {
		return append(out, 0x74)
	}
	if op == rtgWasm32OpShrRegReg {
		return append(out, 0x75)
	}
	return out
}

func rtgWasm32AppendInstr(out []byte, ins *rtgWasm32Instr, nextIndex int, targetIndex int, loopDepth int, exitDepth int, callStackBase int, frameSize int) []byte {
	op := int(ins.op)
	argA := int(ins.a)
	argB := int(ins.b)
	if op == rtgWasm32OpExit {
		out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
		out = rtgWasmAppendCall(out, rtgWasm32ImportProcExit)
		return out
	}
	if op == rtgWasm32OpBuildArgsEnv {
		out = rtgWasm32AppendBuildArgsEnv(out, argA, argB, int(ins.c))
		out = rtgWasm32SetPc(out, nextIndex)
		out = rtgWasmBr(out, loopDepth)
		return out
	}
	if op == rtgWasm32OpMovRegImm {
		out = rtgWasmAppendRecipe(out, "\xff\x00\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == rtgWasm32OpMovRegReg {
		out = rtgWasmAppendRecipe(out, "\xff\x04\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == rtgWasm32OpPushReg {
		out = rtgWasmAppendRecipe(out, "\x20\x00\xff\x04\x00\x36\x02\x00\x20\x00\x41\x04\x6a\x21\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpPushImm {
		out = rtgWasmAppendRecipe(out, "\x20\x00\xff\x00\x00\x36\x02\x00\x20\x00\x41\x04\x6a\x21\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpPopReg {
		out = rtgWasmAppendRecipe(out, "\x20\x00\x41\x04\x6b\x22\x00\x28\x02\x00\xff\x05\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpLoadStack {
		out = rtgWasmAppendRecipe(out, "\xff\x06\x01\x28\x02\x00\xff\x05\x00", argA, argB, 0, 0)
	} else if op == rtgWasm32OpStoreStack {
		out = rtgWasmAppendRecipe(out, "\xff\x06\x01\xff\x04\x00\x36\x02\x00", argA, argB, 0, 0)
	} else if op == rtgWasm32OpLeaStack {
		out = rtgWasmAppendRecipe(out, "\xff\x06\x01\xff\x05\x00", argA, argB, 0, 0)
	} else if op == rtgWasm32OpLoadMem {
		out = rtgWasm32MemAddr(out, argB, int(ins.c))
		out = rtgWasm32LoadSized(out, int(ins.d))
		out = rtgWasm32RegSet(out, argA)
	} else if op == rtgWasm32OpStoreMem {
		out = rtgWasm32MemAddr(out, argB, int(ins.c))
		out = rtgWasm32RegGet(out, argA)
		out = rtgWasm32StoreSized(out, int(ins.d))
	} else if op == rtgWasm32OpLoadIndex {
		out = rtgWasm32IndexAddr(out, argB, int(ins.c), int(ins.d), int(ins.e))
		out = rtgWasm32LoadSized(out, int(ins.f))
		out = rtgWasm32RegSet(out, argA)
	} else if op == rtgWasm32OpStoreIndex {
		out = rtgWasm32IndexAddr(out, argB, int(ins.c), int(ins.d), int(ins.e))
		out = rtgWasm32RegGet(out, argA)
		out = rtgWasm32StoreSized(out, int(ins.f))
	} else if op >= rtgWasm32OpAddRegReg && op <= rtgWasm32OpShrRegReg {
		if op == rtgWasm32OpAndNotRegReg {
			out = rtgWasm32RegGet(out, argA)
			out = rtgWasm32RegGet(out, argB)
			out = rtgWasmAppend4(out, 0x41, 0x7f, 0x73, 0x71)
		} else {
			out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x01\xff\x0a\x02", argA, argB, op, 0)
		}
		out = rtgWasm32RegSet(out, argA)
	} else if op == rtgWasm32OpAddRegImm || op == rtgWasm32OpMulRegImm {
		if op == rtgWasm32OpAddRegImm {
			out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6a\xff\x05\x00", argA, argB, 0, 0)
		} else {
			out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6c\xff\x05\x00", argA, argB, 0, 0)
		}
	} else if op == rtgWasm32OpIncReg {
		out = rtgWasmAppendRecipe(out, "\xff\x04\x00\x41\x01\x6a\xff\x05\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpIncMem || op == rtgWasm32OpDecMem {
		if op == rtgWasm32OpIncMem {
			out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x00\x28\x02\x00\x41\x01\x6a\x36\x02\x00", argA, 0, 0, 0)
		} else {
			out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x00\x28\x02\x00\x41\x01\x6b\x36\x02\x00", argA, 0, 0, 0)
		}
	} else if op == rtgWasm32OpBoolNot {
		out = rtgWasmAppendRecipe(out, "\xff\x04\x00\x45\xff\x05\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpNegReg {
		out = rtgWasmAppendRecipe(out, "\x41\x00\xff\x04\x00\x6b\xff\x05\x00", argA, 0, 0, 0)
	} else if op == rtgWasm32OpCmpRegImm {
		out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x00\x01\x6b\x21\x0c", argA, argB, 0, 0)
	} else if op == rtgWasm32OpCmpRegReg {
		out = rtgWasmAppendRecipe(out, "\xff\x04\x00\xff\x04\x01\x6b\x21\x0c", argA, argB, 0, 0)
	} else if op == rtgWasm32OpSetCond {
		out = rtgWasmAppendRecipe(out, "\xff\x09\x00\x21\x02", argA, 0, 0, 0)
	} else if op == rtgWasm32OpJmp {
		out = rtgWasm32SetPc(out, targetIndex)
		return rtgWasmBr(out, loopDepth)
	} else if op == rtgWasm32OpJz || op == rtgWasm32OpJnz || op == rtgWasm32OpJCond {
		if op == rtgWasm32OpJz {
			out = rtgWasmLocalGet(out, rtgWasm32LocalFlag)
			out = append(out, 0x45)
		} else if op == rtgWasm32OpJnz {
			out = rtgWasmLocalGet(out, rtgWasm32LocalFlag)
			out = rtgWasmAppend2(out, 0x45, 0x45)
		} else {
			out = rtgWasm32AppendCond(out, argA)
		}
		out = rtgWasmAppend2(out, 0x04, 0x40)
		out = rtgWasm32SetPc(out, targetIndex)
		out = append(out, 0x05)
		out = rtgWasm32SetPc(out, nextIndex)
		out = append(out, 0x0b)
		return rtgWasmBr(out, loopDepth)
	} else if op == rtgWasm32OpCall {
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, nextIndex)
		out = rtgWasmI32Store(out, 2, 0)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmLocalGet(out, rtgWasm32LocalFp)
		out = rtgWasmI32Store(out, 2, 4)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, 8)
		out = append(out, 0x6a)
		out = rtgWasmLocalSet(out, rtgWasm32LocalCsp)
		out = rtgWasmLocalGet(out, rtgWasm32LocalFp)
		out = rtgWasmAppendI32Const(out, frameSize)
		out = append(out, 0x6b)
		out = rtgWasmLocalSet(out, rtgWasm32LocalFp)
		out = rtgWasm32SetPc(out, targetIndex)
		return rtgWasmBr(out, loopDepth)
	} else if op == rtgWasm32OpRet {
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, callStackBase)
		out = append(out, 0x46)
		out = rtgWasmAppend2(out, 0x04, 0x40)
		out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
		out = rtgWasmAppendCall(out, rtgWasm32ImportProcExit)
		out = append(out, 0x05)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, 8)
		out = append(out, 0x6b)
		out = rtgWasmLocalTee(out, rtgWasm32LocalCsp)
		out = rtgWasmI32Load(out, 2, 0)
		out = rtgWasmLocalSet(out, rtgWasm32LocalPc)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmI32Load(out, 2, 4)
		out = rtgWasmLocalSet(out, rtgWasm32LocalFp)
		out = append(out, 0x0b)
		return rtgWasmBr(out, loopDepth)
	} else if op == rtgWasm32OpSyscall {
		out = rtgWasm32AppendSyscall(out)
	}
	if loopDepth < 0 {
		return out
	}
	out = rtgWasm32SetPc(out, nextIndex)
	return rtgWasmBr(out, loopDepth)
}

func rtgWasm32AppendDirectArgs(out []byte) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
	out = rtgWasmLocalGet(out, rtgWasm32LocalFp)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRcx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRsi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR8)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR9)
	return out
}

func rtgWasm32AppendStateResults(out []byte) []byte {
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRcx)
	return out
}

func rtgWasm32AppendStateReturn(out []byte) []byte {
	out = rtgWasm32AppendStateResults(out)
	out = append(out, 0x0f)
	return out
}

func rtgWasm32FindRoutineIndex(routinePcs []int, pc int) int {
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

func rtgWasm32MarkFunc(g *rtgLinearGen, fnIndex int) {
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
	rtgAsmAddFuncSymbol(&g.asm, src, nameStart, nameEnd, g.funcLabels[fnIndex])
}

func rtgWasm32AppendDirectCall(out []byte, funcIndex int, frameSize int) []byte {
	out = rtgWasm32AppendDirectArgs(out)
	out = rtgWasmAppendCall(out, funcIndex)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRcx)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdx)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	return out
}

func rtgWasm32AppendInstrDirect(out []byte, ins *rtgWasm32Instr, nextIndex int, targetIndex int, loopDepth int, callStackBase int, frameSize int, hasFrame bool, routinePcs []int) []byte {
	op := int(ins.op)
	argA := int(ins.a)
	argB := int(ins.b)
	if op == rtgWasm32OpExit {
		out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
		out = rtgWasmAppendCall(out, rtgWasm32ImportProcExit)
		out = append(out, 0x00)
		return out
	}
	if op == rtgWasm32OpCall {
		routineIndex := rtgWasm32FindRoutineIndex(routinePcs, argA)
		if routineIndex < 0 {
			out = rtgWasm32AppendInstr(out, ins, nextIndex, targetIndex, loopDepth, loopDepth+1, callStackBase, frameSize)
			return out
		}
		out = rtgWasm32AppendDirectCall(out, rtgWasm32VmFuncBase+routineIndex, frameSize)
		if argB > 6 {
			// Direct wasm calls pass SP by value; drop caller-owned stack args.
			out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
			out = rtgWasmAppendI32Const(out, (argB-6)*4)
			out = append(out, 0x6b)
			out = rtgWasmLocalSet(out, rtgWasm32LocalSp)
		}
		if loopDepth < 0 {
			return out
		}
		out = rtgWasm32SetPc(out, nextIndex)
		return rtgWasmBr(out, loopDepth)
	}
	if op == rtgWasm32OpRet {
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, callStackBase)
		out = append(out, 0x46)
		out = rtgWasmAppend2(out, 0x04, 0x40)
		if hasFrame {
			out = rtgWasmLocalGet(out, rtgWasm32LocalFp)
			out = rtgWasmAppendI32Const(out, frameSize)
			out = append(out, 0x6a)
			out = rtgWasmLocalSet(out, rtgWasm32LocalFp)
		}
		out = rtgWasm32AppendStateReturn(out)
		out = append(out, 0x05)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmAppendI32Const(out, 8)
		out = append(out, 0x6b)
		out = rtgWasmLocalTee(out, rtgWasm32LocalCsp)
		out = rtgWasmI32Load(out, 2, 0)
		out = rtgWasmLocalSet(out, rtgWasm32LocalPc)
		out = rtgWasmLocalGet(out, rtgWasm32LocalCsp)
		out = rtgWasmI32Load(out, 2, 4)
		out = rtgWasmLocalSet(out, rtgWasm32LocalFp)
		out = append(out, 0x0b)
		if loopDepth >= 0 {
			out = rtgWasmBr(out, loopDepth)
		}
		return out
	}
	out = rtgWasm32AppendInstr(out, ins, nextIndex, targetIndex, loopDepth, loopDepth+1, callStackBase, frameSize)
	return out
}

func rtgWasm32CanFusePair(first *rtgWasm32Instr, second *rtgWasm32Instr) bool {
	if second.op == rtgWasm32OpPopReg {
		if first.op == rtgWasm32OpPushReg {
			return true
		}
		if first.op == rtgWasm32OpPushImm {
			return true
		}
	}
	if first.op == rtgWasm32OpStoreStack && second.op == rtgWasm32OpLoadStack && first.b == second.b {
		return true
	}
	return false
}

func rtgWasm32AppendFusedPair(out []byte, first *rtgWasm32Instr, second *rtgWasm32Instr) []byte {
	if second.op == rtgWasm32OpPopReg {
		if first.op == rtgWasm32OpPushReg {
			if first.a == second.a {
				return out
			}
			out = rtgWasm32RegGet(out, int(first.a))
			return rtgWasm32RegSet(out, int(second.a))
		}
		out = rtgWasmAppendI32Const(out, int(first.a))
		return rtgWasm32RegSet(out, int(second.a))
	}
	out = rtgWasm32StackAddr(out, int(first.b))
	out = rtgWasm32RegGet(out, int(first.a))
	out = rtgWasmI32Store(out, 2, 0)
	if first.a == second.a {
		return out
	}
	out = rtgWasm32RegGet(out, int(first.a))
	return rtgWasm32RegSet(out, int(second.a))
}

func rtgWasm32PcInList(pcs []int, pc int) bool {
	for i := 0; i < len(pcs); i++ {
		if pcs[i] == pc {
			return true
		}
	}
	return false
}

func rtgWasm32SortPcs(pcs []int) []int {
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

func rtgWasm32SymbolPcs(a *rtgAsm) []int {
	pcs := make([]int, 0, 2048)
	for i := 0; i < len(a.symbols); i++ {
		label := a.symbols[i].label
		if label >= 0 && label < len(a.labelPos) && a.labelSet[label] {
			pc := a.labelPos[label]
			if !rtgWasm32PcInList(pcs, pc) {
				pcs = append(pcs, pc)
			}
		}
	}
	pcs = rtgWasm32SortPcs(pcs)
	return pcs
}

func rtgWasm32RoutinePcs(a *rtgAsm, code []byte, instrPcs []int) []int {
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
		if int(code[pc]) == rtgWasm32OpCall {
			targetPc := rtgGet32At(code, pc+1)
			targetIndex := rtgWasm32PcLowerBound(instrPcs, targetPc)
			if targetIndex < len(instrPcs) && instrPcs[targetIndex] == targetPc {
				if targetPc >= 0 && targetPc < len(marks) && marks[targetPc] == 0 {
					pcs = append(pcs, targetPc)
					marks[targetPc] = 1
				}
			}
		}
	}
	pcs = rtgWasm32SortPcs(pcs)
	return pcs
}

func rtgWasm32NextPcAfter(pcs []int, pc int, limit int) int {
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

func rtgWasm32SortedPcContains(pcs []int, pc int) bool {
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

func rtgWasm32FirstRetAfter(code []byte, instrPcs []int, startPc int, limit int) int {
	i := rtgWasm32PcLowerBound(instrPcs, startPc)
	for i < len(instrPcs) {
		pc := instrPcs[i]
		if pc >= limit {
			break
		}
		if int(code[pc]) == rtgWasm32OpRet {
			return pc + 1
		}
		i++
	}
	return limit
}

func rtgWasm32RoutineEndPc(startPc int, codeLen int, symbolPcs []int, code []byte, instrPcs []int) int {
	nextSymbol := rtgWasm32NextPcAfter(symbolPcs, startPc, codeLen)
	if startPc == 0 || rtgWasm32SortedPcContains(symbolPcs, startPc) {
		return nextSymbol
	}
	return rtgWasm32FirstRetAfter(code, instrPcs, startPc, nextSymbol)
}

func rtgWasm32AppendDirectRoutineBody(body []byte, instrs []rtgWasm32Instr, codeLen int, routinePcs []int, callStackBase int, frameSize int, hasFrame bool) []byte {
	blockStarts := rtgWasm32BuildBlockStartsLocal(instrs)
	instrBlockIndex := rtgWasm32BuildInstrBlockIndex(blockStarts, len(instrs))
	body = rtgWasmAppendU32(body, 1)
	body = rtgWasmAppendU32(body, 7)
	body = append(body, 0x7f)
	body = rtgWasmAppendI32Const(body, 0)
	body = rtgWasmLocalSet(body, rtgWasm32LocalPc)
	body = rtgWasmAppendI32Const(body, callStackBase)
	body = rtgWasmLocalSet(body, rtgWasm32LocalCsp)
	if hasFrame {
		body = rtgWasmLocalGet(body, rtgWasm32LocalFp)
		body = rtgWasmAppendI32Const(body, frameSize)
		body = append(body, 0x6b)
		body = rtgWasmLocalSet(body, rtgWasm32LocalFp)
	}
	if len(blockStarts) == 0 {
		body = rtgWasm32AppendStateResults(body)
		body = append(body, 0x0b)
		return body
	}
	body = rtgWasmAppend2(body, 0x02, 0x40)
	body = rtgWasmAppend2(body, 0x03, 0x40)
	for i := 0; i < len(blockStarts); i++ {
		body = rtgWasmAppend2(body, 0x02, 0x40)
	}
	body = rtgWasmLocalGet(body, rtgWasm32LocalPc)
	body = append(body, 0x0e)
	body = rtgWasmAppendU32(body, len(blockStarts))
	for i := 0; i < len(blockStarts); i++ {
		body = rtgWasmAppendU32(body, len(blockStarts)-1-i)
	}
	defaultDepth := len(blockStarts) - 1
	body = rtgWasmAppendU32(body, defaultDepth)
	for blockIndex := len(blockStarts) - 1; blockIndex >= 0; blockIndex-- {
		body = append(body, 0x0b)
		start := blockStarts[blockIndex]
		end := rtgWasm32BlockEnd(blockStarts, blockIndex, len(instrs))
		i := start
		for i < end {
			ins := &instrs[i]
			if i+1 < end && rtgWasm32CanFusePair(ins, &instrs[i+1]) {
				body = rtgWasm32AppendFusedPair(body, ins, &instrs[i+1])
				if i+2 >= end {
					nextIndex := rtgWasm32InstrIndexForPcLocal(instrs, len(instrs), int(instrs[i+1].next))
					nextBlock := rtgWasm32BlockForInstrFast(instrBlockIndex, nextIndex)
					body = rtgWasm32SetPc(body, nextBlock)
					body = rtgWasmBr(body, blockIndex)
				}
				i += 2
				continue
			}
			if i+1 < end {
				body = rtgWasm32AppendInstrDirect(body, ins, 0, 0, -1, callStackBase, frameSize, hasFrame, routinePcs)
			} else {
				nextIndex := rtgWasm32InstrIndexForPcLocal(instrs, len(instrs), int(ins.next))
				nextBlock := rtgWasm32BlockForInstrFast(instrBlockIndex, nextIndex)
				targetBlock := 0
				op := int(ins.op)
				if rtgWasm32OpHasTarget(op) || op == rtgWasm32OpJCond {
					targetPc := int(ins.a)
					if ins.op == rtgWasm32OpJCond {
						targetPc = int(ins.b)
					}
					targetIndex := rtgWasm32InstrIndexForPcLocal(instrs, len(instrs), targetPc)
					targetBlock = rtgWasm32BlockForInstrFast(instrBlockIndex, targetIndex)
				}
				body = rtgWasm32AppendInstrDirect(body, ins, nextBlock, targetBlock, blockIndex, callStackBase, frameSize, hasFrame, routinePcs)
			}
			i++
		}
	}
	body = rtgWasmBr(body, 1)
	body = append(body, 0x0b)
	body = append(body, 0x0b)
	body = rtgWasm32AppendStateResults(body)
	body = append(body, 0x0b)
	return body
}

func rtgWasm32AppendDirectStartBody(body []byte, topFunc int, exprStackBase int, callStackBase int, frameTop int) []byte {
	body = rtgWasmAppendU32(body, 1)
	body = rtgWasmAppendU32(body, 16)
	body = append(body, 0x7f)
	body = rtgWasmAppendI32Const(body, 0)
	body = rtgWasmLocalSet(body, rtgWasm32LocalPc)
	body = rtgWasmAppendI32Const(body, exprStackBase)
	body = rtgWasmLocalSet(body, rtgWasm32LocalSp)
	body = rtgWasmAppendI32Const(body, frameTop)
	body = rtgWasmLocalSet(body, rtgWasm32LocalFp)
	body = rtgWasmAppendI32Const(body, callStackBase)
	body = rtgWasmLocalSet(body, rtgWasm32LocalCsp)
	body = rtgWasm32AppendDirectArgs(body)
	body = rtgWasmAppendCall(body, topFunc)
	body = rtgWasmLocalSet(body, rtgWasm32LocalRcx)
	body = rtgWasmLocalSet(body, rtgWasm32LocalRdx)
	body = rtgWasmLocalSet(body, rtgWasm32LocalRax)
	body = rtgWasmLocalGet(body, rtgWasm32LocalRax)
	body = rtgWasmAppendCall(body, rtgWasm32ImportProcExit)
	body = append(body, 0x0b)
	return body
}

func rtgWasm32TypeSectionFull() []byte {
	return rtgWasmAppendEncoded(nil, "\x08\x60\x04\x7f\x7f\x7f\x7f\x01\x7f\x60\x05\x7f\x7f\x7f\x7e\x7f\x01\x7f\x60\x09\x7f\x7f\x7f\x7f\x7f\x7e\x7e\x7f\x7f\x01\x7f\x60\x01\x7f\x01\x7f\x60\x01\x7f\x00\x60\x00\x00\x60\x02\x7f\x7f\x01\x7f\x60\x09\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x7f\x03\x7f\x7f\x7f")
}

func rtgWasm32AppendImport(out []byte, name string, typ int) []byte {
	out = rtgWasmAppendName(out, "wasi_snapshot_preview1")
	out = rtgWasmAppendName(out, name)
	out = append(out, 0x00)
	return rtgWasmAppendU32(out, typ)
}

func rtgWasm32ImportSectionFull() []byte {
	return rtgWasmAppendEncoded(nil, "\x0d\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x77\x72\x69\x74\x65\x00\x00\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x07\x66\x64\x5f\x72\x65\x61\x64\x00\x00\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x70\x72\x65\x61\x64\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x66\x64\x5f\x70\x77\x72\x69\x74\x65\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x70\x61\x74\x68\x5f\x6f\x70\x65\x6e\x00\x02\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x66\x64\x5f\x63\x6c\x6f\x73\x65\x00\x03\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0d\x66\x64\x5f\x66\x64\x73\x74\x61\x74\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0a\x66\x64\x5f\x72\x65\x61\x64\x64\x69\x72\x00\x01\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0e\x61\x72\x67\x73\x5f\x73\x69\x7a\x65\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x08\x61\x72\x67\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x11\x65\x6e\x76\x69\x72\x6f\x6e\x5f\x73\x69\x7a\x65\x73\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x0b\x65\x6e\x76\x69\x72\x6f\x6e\x5f\x67\x65\x74\x00\x06\x16\x77\x61\x73\x69\x5f\x73\x6e\x61\x70\x73\x68\x6f\x74\x5f\x70\x72\x65\x76\x69\x65\x77\x31\x09\x70\x72\x6f\x63\x5f\x65\x78\x69\x74\x00\x04")
}

func rtgWasm32FunctionSectionDirect(routineCount int) []byte {
	var out []byte
	out = rtgWasmAppendU32(out, routineCount+1)
	out = rtgWasmAppendU32(out, 5)
	for i := 0; i < routineCount; i++ {
		out = rtgWasmAppendU32(out, rtgWasm32VmFuncType)
	}
	return out
}

func rtgWasm32MemorySectionFull(memSize int) []byte {
	pages := (memSize + 65535) / 65536
	if pages < 16 {
		pages = 16
	}
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, pages)
	return out
}

func rtgWasm32ExportSectionFull() []byte {
	return rtgWasmAppendEncoded(nil, "\x02\x06\x6d\x65\x6d\x6f\x72\x79\x02\x00\x06\x5f\x73\x74\x61\x72\x74\x00\x0d")
}

func rtgWasm32AppendCodeSectionDirect(out []byte, a *rtgAsm, instrPcs []int, routinePcs []int, symbolPcs []int, codeLen int, callStackBase int, frameTop int, exprStackBase int) []byte {
	frameSize := 6144
	out = append(out, 10)
	lenAt := len(out)
	out = rtgWasmAppendU32Fixed5(out, 0)
	payloadStart := len(out)
	out = rtgWasmAppendU32(out, len(routinePcs)+1)
	startLenAt := len(out)
	out = rtgWasmAppendU32Fixed5(out, 0)
	startBody := len(out)
	out = rtgWasm32AppendDirectStartBody(out, rtgWasm32VmFuncBase, exprStackBase, callStackBase, frameTop)
	rtgWasmPatchU32Fixed5(out, startLenAt, len(out)-startBody)
	for i := 0; i < len(routinePcs); i++ {
		startPc := routinePcs[i]
		endPc := rtgWasm32RoutineEndPc(startPc, codeLen, symbolPcs, a.code, instrPcs)
		startIndex := rtgWasm32PcLowerBound(instrPcs, startPc)
		endIndex := rtgWasm32PcLowerBound(instrPcs, endPc)
		routineInstrPcs := instrPcs[startIndex:endIndex]
		hasFrame := startPc != 0 && rtgWasm32SortedPcContains(symbolPcs, startPc)
		out = rtgWasm32AppendDirectRoutine(out, a.code, routineInstrPcs, codeLen, routinePcs, callStackBase, frameSize, hasFrame)
	}
	rtgWasmPatchU32Fixed5(out, lenAt, len(out)-payloadStart)
	return out
}

func rtgWasm32EnsureAdditionalCapacity(out []byte, additional int) []byte {
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

func rtgWasm32AppendDirectRoutine(out []byte, code []byte, instrPcs []int, codeLen int, routinePcs []int, callStackBase int, frameSize int, hasFrame bool) []byte {
	out = rtgWasm32EnsureAdditionalCapacity(out, len(instrPcs)*16+rtgWasm32RoutineBodyCapacity)
	mark := rtg_runtime_ArenaMark()
	instrs := rtgWasm32DecodePcRange(code, instrPcs)
	oldCap := cap(out)
	lenAt := len(out)
	out = rtgWasmAppendU32Fixed5(out, 0)
	bodyStart := len(out)
	out = rtgWasm32AppendDirectRoutineBody(out, instrs, codeLen, routinePcs, callStackBase, frameSize, hasFrame)
	rtgWasmPatchU32Fixed5(out, lenAt, len(out)-bodyStart)
	if cap(out) == oldCap {
		rtg_runtime_ArenaReset(mark)
	}
	return out
}

func rtgWasm32DataSectionFull(dataBase int, data []byte) []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x00)
	out = rtgWasmAppendI32Const(out, dataBase)
	out = append(out, 0x0b)
	out = rtgWasmAppendByteVec(out, data)
	return out
}

func rtgWasm32Image(a *rtgAsm) []byte {
	dataBase := rtgWasm32ProgramBase
	bssBase := rtgAlignTo8(dataBase + len(a.data))
	rtgWasm32Patch(a, dataBase, bssBase)
	instrPcs := rtgWasm32InstructionPcs(a.code)
	symbolPcs := rtgWasm32SymbolPcs(a)
	routinePcs := rtgWasm32RoutinePcs(a, a.code, instrPcs)
	exprStackBase := bssBase + a.bssSize + rtgWasm32StackGuardSize
	callStackBase := exprStackBase + rtgWasm32ExprStackSize
	frameTop := callStackBase + rtgWasm32CallStackSize + rtgWasm32FrameStackSize
	memSize := bssBase + a.bssSize + rtgWasm32StackGuardSize + rtgWasm32ExprStackSize + rtgWasm32CallStackSize + rtgWasm32FrameStackSize + rtgWasm32StackGuardSize
	out := make([]byte, 0, rtgWasm32ImageOutputCapacity)
	out = append(out, 0x00)
	out = append(out, 0x61)
	out = append(out, 0x73)
	out = append(out, 0x6d)
	out = append(out, 0x01)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = append(out, 0x00)
	out = rtgWasmAppendSection(out, 1, rtgWasm32TypeSectionFull())
	out = rtgWasmAppendSection(out, 2, rtgWasm32ImportSectionFull())
	out = rtgWasmAppendSection(out, 3, rtgWasm32FunctionSectionDirect(len(routinePcs)))
	out = rtgWasmAppendSection(out, 5, rtgWasm32MemorySectionFull(memSize))
	out = rtgWasmAppendSection(out, 7, rtgWasm32ExportSectionFull())
	out = rtgWasm32AppendCodeSectionDirect(out, a, instrPcs, routinePcs, symbolPcs, len(a.code), callStackBase, frameTop, exprStackBase)
	if len(a.data) > 0 {
		out = rtgWasmAppendSection(out, 11, rtgWasm32DataSectionFull(dataBase, a.data))
	}
	return out
}

func rtgWasm32EmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	oldLocals := g.locals
	oldLocalCount := g.localCount
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldStackUsed := g.stackUsed
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	locals = make([]rtgLocalInfo, rtgFunctionLocalCap(fn))
	gotoLabels = make([]rtgGlobalInfo, 0, 0)
	g.locals = locals
	g.localCount = 0
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	if rtgTypeUsesHiddenResult(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRdi, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgBindNamedResults(g, fnInfoIndex) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if !g.lastRangeReturns {
		rtgAsmPrimaryImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	g.locals = oldLocals
	g.localCount = oldLocalCount
	g.breakDepth = oldBreak
	g.continueDepth = oldContinue
	g.currentFunc = oldCurrent
	g.returnStruct = oldReturnStruct
	g.stackUsed = oldStackUsed
	g.gotoLabels = oldGotoLabels
	g.lastRangeReturns = oldLastRangeReturns
	return true
}

func rtgWasm32EmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	rtgWasm32MarkFunc(g, fnIndex)
	if wordCount > 0 {
		rtgWasm32AsmPopRdi(a)
	}
	if wordCount > 1 {
		rtgWasm32AsmPopRsi(a)
	}
	if wordCount > 2 {
		rtgWasm32AsmPopRdx(a)
	}
	if wordCount > 3 {
		rtgWasm32AsmPopRcx(a)
	}
	if wordCount > 4 {
		rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegR8)
	}
	if wordCount > 5 {
		rtgWasm32EmitReg(a, rtgWasm32OpPopReg, rtgWasm32RegR9)
	}
	rtgWasm32EmitCallLabel(a, g.funcLabels[fnIndex], wordCount)
}

func rtgWasm32EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= rtgTokCount(p) {
		return false
	}
	start := rtgTokStart(p, tok)
	end := rtgTokEnd(p, tok)
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	if c0 == '+' {
		rtgWasm32AsmAddRaxRcx(a)
		return true
	}
	if c0 == '-' {
		rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRcx)
		rtgWasm32EmitRegReg(a, rtgWasm32OpSubRegReg, rtgWasm32RegRdx, rtgWasm32RegRax)
		rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		return true
	}
	if c0 == '*' {
		rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		return true
	}
	if c0 == '/' {
		rtgWasm32AsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgWasm32AsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgWasm32EmitRegReg(a, rtgWasm32OpAndNotRegReg, rtgWasm32RegRcx, rtgWasm32RegRax)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		} else {
			rtgWasm32EmitRegReg(a, rtgWasm32OpAndRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		}
		return true
	}
	if c0 == '|' {
		rtgWasm32EmitRegReg(a, rtgWasm32OpOrRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		return true
	}
	if c0 == '^' {
		rtgWasm32EmitRegReg(a, rtgWasm32OpXorRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRax)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
			rtgWasm32EmitRegReg(a, rtgWasm32OpShlRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		} else if c1 == '=' {
			rtgWasm32AsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgWasm32AsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRax)
			rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
			rtgWasm32EmitRegReg(a, rtgWasm32OpShrRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
		} else if c1 == '=' {
			rtgWasm32AsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgWasm32AsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgWasm32AsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgWasm32AsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtgWasm32EmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if rtgTokCharIs(p, e.tok, '*') {
		if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
			return false
		}
		rtgAsmPushPrimary(a)
		if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		rtgAsmSarPrimaryImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
			return false
		}
		rtgAsmShlPrimaryImm(a, 2)
		rtgAsmPushPrimary(a)
		if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
			return false
		}
		rtgAsmPopTertiary(a)
		rtgAsmDivLeftTertiaryRightPrimary(a, false)
		return true
	}
	if !rtgEmitScalarExprForKind(g, ep, e.left, rtgTypeFloat64) {
		return false
	}
	rtgAsmPushPrimary(a)
	if !rtgEmitScalarExprForKind(g, ep, e.right, rtgTypeFloat64) {
		return false
	}
	rtgAsmPopTertiary(a)
	return rtgEmitPrimaryTertiaryOp(g, e.tok)
}

func rtgWasm32EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	arenaAllocLabel := rtgEnsureArenaAllocHelper(g)
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	noGrowLabel := rtgAsmNewLabel(a)
	capNonZeroLabel := rtgAsmNewLabel(a)
	capReadyLabel := rtgAsmNewLabel(a)
	copyLoopLabel := rtgAsmNewLabel(a)
	copyDoneLabel := rtgAsmNewLabel(a)
	returnLabel := rtgAsmNewLabel(a)
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

	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegR8, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegR9, 0, 4)
	rtgWasm32EmitRegReg(a, rtgWasm32OpCmpRegReg, rtgWasm32RegR8, rtgWasm32RegRcx)
	rtgWasm32EmitCondBranch(a, rtgWasm32CondLt, noGrowLabel)

	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
	rtgAsmStorePrimaryBss(a, elemSizeOff)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRdi)
	rtgAsmStorePrimaryBss(a, ptrSlotOff)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRsi)
	rtgAsmStorePrimaryBss(a, lenSlotOff)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegR9)
	rtgAsmStorePrimaryBss(a, capSlotOff)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegR8)
	rtgAsmStorePrimaryBss(a, oldLenOff)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgAsmStorePrimaryBss(a, oldPtrOff)

	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, capNonZeroLabel)
	rtgWasm32EmitRegImm(a, rtgWasm32OpMovRegImm, rtgWasm32RegRcx, 16)
	rtgAsmJmpLabel(a, capReadyLabel)
	rtgAsmMarkLabel(a, capNonZeroLabel)
	rtgWasm32EmitRegReg(a, rtgWasm32OpAddRegReg, rtgWasm32RegRcx, rtgWasm32RegR8)
	rtgAsmMarkLabel(a, capReadyLabel)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
	rtgAsmStorePrimaryBss(a, newCapOff)
	rtgAsmLoadPrimaryBss(a, elemSizeOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, newCapOff)
	rtgAsmPopTertiary(a)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
	rtgAsmStorePrimaryBss(a, allocSizeOff)

	rtgAsmLoadPrimaryBss(a, allocSizeOff)
	rtgAsmCallLabel(a, arenaAllocLabel)
	rtgAsmStorePrimaryBss(a, destOff)

	rtgAsmLoadPrimaryBss(a, oldLenOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, elemSizeOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopPrimary(a)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRax, rtgWasm32RegRdx)
	rtgAsmStorePrimaryBss(a, copySizeOff)
	rtgAsmPrimaryImm(a, 0)
	rtgAsmStorePrimaryBss(a, copyIndexOff)
	rtgAsmMarkLabel(a, copyLoopLabel)
	rtgAsmLoadPrimaryBss(a, copyIndexOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, copySizeOff)
	rtgAsmPopTertiary(a)
	rtgAsmCmpTertiaryPrimarySet(a, 0x9d)
	rtgAsmCmpPrimaryImm8(a, 0)
	rtgAsmJnzLabel(a, copyDoneLabel)
	rtgAsmLoadPrimaryBss(a, copyIndexOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, oldPtrOff)
	rtgAsmPopTertiary(a)
	rtgAsmLoadBytePrimaryIndexTertiary(a)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, copyIndexOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, destOff)
	rtgAsmCopyPrimaryToSecondary(a)
	rtgAsmPopTertiary(a)
	rtgAsmPopPrimary(a)
	rtgAsmStorePrimaryMemSecondaryTertiarySize(a, 1)
	rtgAsmLoadPrimaryBss(a, copyIndexOff)
	rtgAsmIncPrimary(a)
	rtgAsmStorePrimaryBss(a, copyIndexOff)
	rtgAsmJmpLabel(a, copyLoopLabel)
	rtgAsmMarkLabel(a, copyDoneLabel)

	rtgAsmLoadPrimaryBss(a, ptrSlotOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, destOff)
	rtgAsmPopSecondary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
	rtgAsmLoadPrimaryBss(a, capSlotOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, newCapOff)
	rtgAsmPopSecondary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
	rtgAsmLoadPrimaryBss(a, lenSlotOff)
	rtgAsmPushPrimary(a)
	rtgAsmLoadPrimaryBss(a, oldLenOff)
	rtgAsmIncPrimary(a)
	rtgAsmPopSecondary(a)
	rtgAsmStorePrimaryMemSecondaryDisp(a, 0)
	rtgAsmLoadPrimaryBss(a, copySizeOff)
	rtgAsmCopyPrimaryToTertiary(a)
	rtgAsmLoadPrimaryBss(a, destOff)
	rtgAsmAddPrimaryTertiary(a)
	rtgAsmJmpLabel(a, returnLabel)

	rtgAsmMarkLabel(a, noGrowLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRcx, rtgWasm32RegRdx)
	rtgWasm32AsmAddRaxRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32AsmIncRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgAsmMarkLabel(a, returnLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgWasm32EnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitIndex(a, rtgWasm32OpStoreIndex, rtgWasm32RegRdx, rtgWasm32RegRax, rtgWasm32RegRcx, 1, 0, 1)
	rtgWasm32AsmIncRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtgWasm32EnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitIndex(a, rtgWasm32OpStoreIndex, rtgWasm32RegRdx, rtgWasm32RegRax, rtgWasm32RegRcx, 8, 0, 4)
	rtgWasm32AsmIncRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtgWasm32EnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR9, rtgWasm32RegRdx)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegR8, rtgWasm32RegRdx)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMovRegReg, rtgWasm32RegRdx, rtgWasm32RegRax)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegR10, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitRegReg(a, rtgWasm32OpAddRegReg, rtgWasm32RegR10, rtgWasm32RegRcx)
	rtgAsmMarkLabel(a, loopLabel)
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegR9, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdx, 0, 1)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegR10, 0, 1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdx, 1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegR10, 1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegR9, -1)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitRegReg(a, rtgWasm32OpAddRegReg, rtgWasm32RegRax, rtgWasm32RegR8)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRsi, 0, 4)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

func rtgWasm32EnsureCopyWordsHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.copyWordsEmitted {
		return g.copyWordsLabel
	}
	g.copyWordsEmitted = true
	g.copyWordsLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.copyWordsLabel)
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegRdx, 0)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRsi, 8)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdi, 8)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdx, -1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegRdx, 0)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}

func rtgWasm32EnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.streqLabel)
	rtgAsmPrimaryImm(a, 0)
	rtgWasm32EmitRegReg(a, rtgWasm32OpCmpRegReg, rtgWasm32RegRsi, rtgWasm32RegRcx)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegRsi, 0)
	rtgAsmJzLabel(a, equalLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegR8, rtgWasm32RegRdi, 0, 1)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegR9, rtgWasm32RegRdx, 0, 1)
	rtgWasm32EmitRegReg(a, rtgWasm32OpCmpRegReg, rtgWasm32RegR8, rtgWasm32RegR9)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdi, 1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRdx, 1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpAddRegImm, rtgWasm32RegRsi, -1)
	rtgWasm32EmitRegImm(a, rtgWasm32OpCmpRegImm, rtgWasm32RegRsi, 0)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmPrimaryImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
