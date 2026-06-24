package main

func rtgWasmAppendU32(out []byte, v int) []byte {
	for {
		b := byte(v & 0x7f)
		v = v >> 7
		if v != 0 {
			b = b | 0x80
		}
		out = append(out, b)
		if v == 0 {
			return out
		}
	}
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
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, reg)
	rtgAsmEmit32(a, imm)
}

func rtgWasm32EmitRegReg(a *rtgAsm, op int, dst int, src int) {
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, dst)
	rtgAsmEmit8(a, src)
}

func rtgWasm32EmitReg(a *rtgAsm, op int, reg int) {
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, reg)
}

func rtgWasm32EmitStack(a *rtgAsm, op int, reg int, offset int) {
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, reg)
	rtgAsmEmit32(a, offset)
}

func rtgWasm32EmitMem(a *rtgAsm, op int, reg int, base int, disp int, size int) {
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, reg)
	rtgAsmEmit8(a, base)
	rtgAsmEmit32(a, disp)
	rtgAsmEmit8(a, size)
}

func rtgWasm32EmitIndex(a *rtgAsm, op int, reg int, base int, index int, scale int, disp int, size int) {
	rtgAsmEmit8(a, op)
	rtgAsmEmit8(a, reg)
	rtgAsmEmit8(a, base)
	rtgAsmEmit8(a, index)
	rtgAsmEmit8(a, scale)
	rtgAsmEmit32(a, disp)
	rtgAsmEmit8(a, size)
}

func rtgWasm32EmitBranch(a *rtgAsm, op int, label int) {
	rtgAsmEmit8(a, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgWasm32EmitCondBranch(a *rtgAsm, cond int, label int) {
	rtgAsmEmit8(a, rtgWasm32OpJCond)
	rtgAsmEmit8(a, cond)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
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
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitIndex(a, rtgWasm32OpLoadIndex, rtgWasm32RegRax, rtgWasm32RegRax, rtgWasm32RegRcx, size, 0, size)
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
	if size > 4 {
		size = 4
	}
	rtgWasm32EmitIndex(a, rtgWasm32OpStoreIndex, rtgWasm32RegRax, rtgWasm32RegRdx, rtgWasm32RegRcx, size, 0, size)
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
	rtgAsmEmit8(a, rtgWasm32OpNop)
}

func rtgWasm32AsmRet(a *rtgAsm) {
	rtgAsmEmit8(a, rtgWasm32OpRet)
}

func rtgWasm32AsmCallLabel(a *rtgAsm, label int) {
	rtgWasm32EmitBranch(a, rtgWasm32OpCall, label)
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

const rtgWasm32ProgramBase = 1048576
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
const rtgWasm32ImportArgsSizesGet = 7
const rtgWasm32ImportArgsGet = 8
const rtgWasm32ImportEnvironSizesGet = 9
const rtgWasm32ImportEnvironGet = 10
const rtgWasm32ImportProcExit = 11
const rtgWasm32StartFuncIndex = 12

const rtgWasm32LocalPc = 0
const rtgWasm32LocalSp = 1
const rtgWasm32LocalFp = 2
const rtgWasm32LocalCsp = 3
const rtgWasm32LocalRax = 4
const rtgWasm32LocalRdx = 5
const rtgWasm32LocalRcx = 6
const rtgWasm32LocalRdi = 7
const rtgWasm32LocalRsi = 8
const rtgWasm32LocalR8 = 9
const rtgWasm32LocalR9 = 10
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

func rtgWasm32DecodeOne(code []byte, pc int) rtgWasm32Instr {
	var ins rtgWasm32Instr
	ins.pc = pc
	ins.op = int(code[pc])
	ins.next = pc + 1
	if ins.op == rtgWasm32OpMovRegImm {
		ins.a = int(code[pc+1])
		ins.b = rtgWasm32GetS32(code, pc+2)
		ins.next = pc + 6
		return ins
	}
	if ins.op == rtgWasm32OpMovRegReg || ins.op == rtgWasm32OpAddRegReg || ins.op == rtgWasm32OpSubRegReg || ins.op == rtgWasm32OpMulRegReg || ins.op == rtgWasm32OpDivRegReg || ins.op == rtgWasm32OpModRegReg || ins.op == rtgWasm32OpAndRegReg || ins.op == rtgWasm32OpOrRegReg || ins.op == rtgWasm32OpXorRegReg || ins.op == rtgWasm32OpAndNotRegReg || ins.op == rtgWasm32OpShlRegReg || ins.op == rtgWasm32OpShrRegReg || ins.op == rtgWasm32OpCmpRegReg {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.next = pc + 3
		return ins
	}
	if ins.op == rtgWasm32OpPushReg || ins.op == rtgWasm32OpPopReg || ins.op == rtgWasm32OpIncReg || ins.op == rtgWasm32OpIncMem || ins.op == rtgWasm32OpDecMem || ins.op == rtgWasm32OpBoolNot || ins.op == rtgWasm32OpNegReg || ins.op == rtgWasm32OpSetCond {
		ins.a = int(code[pc+1])
		ins.next = pc + 2
		return ins
	}
	if ins.op == rtgWasm32OpPushImm {
		ins.a = rtgWasm32GetS32(code, pc+1)
		ins.next = pc + 5
		return ins
	}
	if ins.op == rtgWasm32OpLoadStack || ins.op == rtgWasm32OpStoreStack || ins.op == rtgWasm32OpLeaStack || ins.op == rtgWasm32OpAddRegImm || ins.op == rtgWasm32OpMulRegImm || ins.op == rtgWasm32OpCmpRegImm {
		ins.a = int(code[pc+1])
		ins.b = rtgWasm32GetS32(code, pc+2)
		ins.next = pc + 6
		return ins
	}
	if ins.op == rtgWasm32OpLoadMem || ins.op == rtgWasm32OpStoreMem {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = rtgWasm32GetS32(code, pc+3)
		ins.d = int(code[pc+7])
		ins.next = pc + 8
		return ins
	}
	if ins.op == rtgWasm32OpLoadIndex || ins.op == rtgWasm32OpStoreIndex {
		ins.a = int(code[pc+1])
		ins.b = int(code[pc+2])
		ins.c = int(code[pc+3])
		ins.d = int(code[pc+4])
		ins.e = rtgWasm32GetS32(code, pc+5)
		ins.f = int(code[pc+9])
		ins.next = pc + 10
		return ins
	}
	if ins.op == rtgWasm32OpJmp || ins.op == rtgWasm32OpJz || ins.op == rtgWasm32OpJnz || ins.op == rtgWasm32OpCall {
		ins.a = rtgGet32At(code, pc+1)
		ins.next = pc + 5
		return ins
	}
	if ins.op == rtgWasm32OpJCond {
		ins.a = int(code[pc+1])
		ins.b = rtgGet32At(code, pc+2)
		ins.next = pc + 6
		return ins
	}
	if ins.op == rtgWasm32OpBuildArgsEnv {
		ins.a = rtgGet32At(code, pc+1)
		ins.b = rtgGet32At(code, pc+5)
		ins.c = rtgGet32At(code, pc+9)
		ins.next = pc + 13
		return ins
	}
	return ins
}

func rtgWasm32Decode(code []byte) []rtgWasm32Instr {
	out := make([]rtgWasm32Instr, 0, 262144)
	pc := 0
	for pc < len(code) {
		ins := rtgWasm32DecodeOne(code, pc)
		if ins.next <= pc {
			ins.next = pc + 1
		}
		out = append(out, ins)
		pc = ins.next
	}
	return out
}

func rtgWasm32BuildPcIndex(instrs []rtgWasm32Instr, codeLen int) []int {
	pcIndex := make([]int, codeLen+1, 2097152)
	for i := 0; i < len(pcIndex); i++ {
		pcIndex[i] = -1
	}
	for i := 0; i < len(instrs); i++ {
		pc := instrs[i].pc
		if pc >= 0 && pc < len(pcIndex) {
			pcIndex[pc] = i
		}
	}
	return pcIndex
}

func rtgWasm32InstrIndexForPcFast(pcIndex []int, instrCount int, pc int) int {
	if pc >= 0 && pc < len(pcIndex) {
		idx := pcIndex[pc]
		if idx >= 0 {
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
	if op == rtgWasm32OpCall || op == rtgWasm32OpRet {
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

func rtgWasm32BuildBlockStarts(instrs []rtgWasm32Instr, pcIndex []int) []int {
	starts := make([]int, 0, 65536)
	instrCount := len(instrs)
	marks := make([]int, instrCount+1, 262144)
	for i := 0; i < len(marks); i++ {
		marks[i] = 0
	}
	if instrCount > 0 {
		marks[0] = 1
	}
	for i := 0; i < instrCount; i++ {
		ins := instrs[i]
		if rtgWasm32IsControlOp(ins.op) {
			if i+1 < instrCount {
				marks[i+1] = 1
			}
		}
		if rtgWasm32OpHasTarget(ins.op) {
			targetIndex := rtgWasm32InstrIndexForPcFast(pcIndex, instrCount, ins.a)
			if targetIndex < instrCount {
				marks[targetIndex] = 1
			}
		}
		if ins.op == rtgWasm32OpJCond {
			targetIndex := rtgWasm32InstrIndexForPcFast(pcIndex, instrCount, ins.b)
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
	blockIndex := make([]int, instrCount+1, 262144)
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
	return rtgWasmAppendU32(out, local)
}

func rtgWasmLocalSet(out []byte, local int) []byte {
	out = append(out, 0x21)
	return rtgWasmAppendU32(out, local)
}

func rtgWasmLocalTee(out []byte, local int) []byte {
	out = append(out, 0x22)
	return rtgWasmAppendU32(out, local)
}

func rtgWasmI32Load(out []byte, align int, off int) []byte {
	out = append(out, 0x28)
	out = rtgWasmAppendU32(out, align)
	return rtgWasmAppendU32(out, off)
}

func rtgWasmI32Load8U(out []byte) []byte {
	out = append(out, 0x2d)
	out = rtgWasmAppendU32(out, 0)
	return rtgWasmAppendU32(out, 0)
}

func rtgWasmI32Load16S(out []byte) []byte {
	out = append(out, 0x2e)
	out = rtgWasmAppendU32(out, 1)
	return rtgWasmAppendU32(out, 0)
}

func rtgWasmI32Store(out []byte, align int, off int) []byte {
	out = append(out, 0x36)
	out = rtgWasmAppendU32(out, align)
	return rtgWasmAppendU32(out, off)
}

func rtgWasmI32Store8(out []byte) []byte {
	out = append(out, 0x3a)
	out = rtgWasmAppendU32(out, 0)
	return rtgWasmAppendU32(out, 0)
}

func rtgWasmI32Store16(out []byte) []byte {
	out = append(out, 0x3b)
	out = rtgWasmAppendU32(out, 1)
	return rtgWasmAppendU32(out, 0)
}

func rtgWasmI64Const(out []byte, value int) []byte {
	out = append(out, 0x42)
	return rtgWasmAppendS32(out, value)
}

func rtgWasmBr(out []byte, depth int) []byte {
	out = append(out, 0x0c)
	return rtgWasmAppendU32(out, depth)
}

func rtgWasmBrIf(out []byte, depth int) []byte {
	out = append(out, 0x0d)
	return rtgWasmAppendU32(out, depth)
}

func rtgWasm32SetPc(out []byte, pc int) []byte {
	out = rtgWasmAppendI32Const(out, pc)
	return rtgWasmLocalSet(out, rtgWasm32LocalPc)
}

func rtgWasm32RegGet(out []byte, reg int) []byte {
	return rtgWasmLocalGet(out, rtgWasm32RegLocal(reg))
}

func rtgWasm32RegSet(out []byte, reg int) []byte {
	return rtgWasmLocalSet(out, rtgWasm32RegLocal(reg))
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
		return rtgWasmI32Load8U(out)
	}
	if size == 2 {
		return rtgWasmI32Load16S(out)
	}
	return rtgWasmI32Load(out, 2, 0)
}

func rtgWasm32StoreSized(out []byte, size int) []byte {
	if size == 1 {
		return rtgWasmI32Store8(out)
	}
	if size == 2 {
		return rtgWasmI32Store16(out)
	}
	return rtgWasmI32Store(out, 2, 0)
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
	return rtgWasmI32Store(out, 2, 0)
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
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasm32AppendIov(out, rtgWasm32LocalRsi, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchN)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdWrite)
	out = rtgWasm32AppendErrnoResult(out, rtgWasm32ScratchN)
	out = append(out, 0x05)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 0)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasm32AppendIov(out, rtgWasm32LocalRsi, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchN)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdRead)
	out = rtgWasm32AppendErrnoResult(out, rtgWasm32ScratchN)
	out = append(out, 0x05)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 18)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasm32AppendIov(out, rtgWasm32LocalRsi, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR10)
	out = append(out, 0xac)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchN)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdPwrite)
	out = rtgWasm32AppendErrnoResult(out, rtgWasm32ScratchN)
	out = append(out, 0x05)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 17)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasm32AppendIov(out, rtgWasm32LocalRsi, rtgWasm32LocalRdx)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchIov)
	out = rtgWasmAppendI32Const(out, 1)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR10)
	out = append(out, 0xac)
	out = rtgWasmAppendI32Const(out, rtgWasm32ScratchN)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdPread)
	out = rtgWasm32AppendErrnoResult(out, rtgWasm32ScratchN)
	out = append(out, 0x05)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 3)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdClose)
	out = rtgWasm32AppendErrnoOnlyResult(out)
	out = append(out, 0x05)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
	out = rtgWasmAppendI32Const(out, 91)
	out = append(out, 0x46)
	out = rtgWasmAppend2(out, 0x04, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRdi)
	out = rtgWasmAppendI32Const(out, rtgWasm32FdstatScratch)
	out = rtgWasmAppendCall(out, rtgWasm32ImportFdstatGet)
	out = rtgWasm32AppendErrnoOnlyResult(out)
	out = append(out, 0x05)
	out = rtgWasm32AppendOpen(out)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	return out
}

func rtgWasm32AppendOpen(out []byte) []byte {
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
	out = rtgWasmI64Const(out, 2097254)
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
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppend2(out, 0x02, 0x40)
	out = rtgWasmAppend2(out, 0x03, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmLocalGet(out, countLocal)
	out = append(out, 0x4f)
	out = rtgWasmBrIf(out, 1)
	out = rtgWasmAppendI32Const(out, ptrArea)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 4)
	out = rtgWasmAppend2(out, 0x6c, 0x6a)
	out = rtgWasmI32Load(out, 2, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp2)
	out = rtgWasmAppendI32Const(out, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp3)
	out = rtgWasmAppend2(out, 0x02, 0x40)
	out = rtgWasmAppend2(out, 0x03, 0x40)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp2)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp3)
	out = append(out, 0x6a)
	out = rtgWasmI32Load8U(out)
	out = append(out, 0x45)
	out = rtgWasmBrIf(out, 1)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp3)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp3)
	out = rtgWasmBr(out, 0)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	out = rtgWasmAppendI32Const(out, destAddr)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 16)
	out = rtgWasmAppend2(out, 0x6c, 0x6a)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp2)
	out = rtgWasmI32Store(out, 2, 0)
	out = rtgWasmAppendI32Const(out, destAddr+8)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 16)
	out = rtgWasmAppend2(out, 0x6c, 0x6a)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp3)
	out = rtgWasmI32Store(out, 2, 0)
	out = rtgWasmLocalGet(out, rtgWasm32LocalTmp)
	out = rtgWasmAppendI32Const(out, 1)
	out = append(out, 0x6a)
	out = rtgWasmLocalSet(out, rtgWasm32LocalTmp)
	out = rtgWasmBr(out, 0)
	out = append(out, 0x0b)
	out = append(out, 0x0b)
	return out
}

func rtgWasm32AppendBuildArgsEnv(out []byte, argsAddr int, envAddr int, envLenAddr int) []byte {
	out = rtgWasmAppendI32Const(out, rtgWasm32ArgsCountPtr)
	out = rtgWasmAppendI32Const(out, rtgWasm32ArgsSizePtr)
	out = rtgWasmAppendCall(out, rtgWasm32ImportArgsSizesGet)
	out = append(out, 0x1a)
	out = rtgWasmAppendI32Const(out, rtgWasm32ArgsPtrArea)
	out = rtgWasmAppendI32Const(out, rtgWasm32ArgsDataArea)
	out = rtgWasmAppendCall(out, rtgWasm32ImportArgsGet)
	out = append(out, 0x1a)
	out = rtgWasmAppendI32Const(out, rtgWasm32ArgsCountPtr)
	out = rtgWasmI32Load(out, 2, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRsi)
	out = rtgWasmLocalGet(out, rtgWasm32LocalRsi)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdx)
	out = rtgWasmAppendI32Const(out, argsAddr)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRdi)
	out = rtgWasm32AppendStringSliceBuild(out, rtgWasm32ArgsPtrArea, rtgWasm32LocalRsi, argsAddr)

	out = rtgWasmAppendI32Const(out, rtgWasm32EnvCountPtr)
	out = rtgWasmAppendI32Const(out, rtgWasm32EnvSizePtr)
	out = rtgWasmAppendCall(out, rtgWasm32ImportEnvironSizesGet)
	out = append(out, 0x1a)
	out = rtgWasmAppendI32Const(out, rtgWasm32EnvPtrArea)
	out = rtgWasmAppendI32Const(out, rtgWasm32EnvDataArea)
	out = rtgWasmAppendCall(out, rtgWasm32ImportEnvironGet)
	out = append(out, 0x1a)
	out = rtgWasmAppendI32Const(out, rtgWasm32EnvCountPtr)
	out = rtgWasmI32Load(out, 2, 0)
	out = rtgWasmLocalSet(out, rtgWasm32LocalR8)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR8)
	out = rtgWasmLocalSet(out, rtgWasm32LocalR9)
	out = rtgWasmAppendI32Const(out, envAddr)
	out = rtgWasmLocalSet(out, rtgWasm32LocalRcx)
	out = rtgWasmAppendI32Const(out, envLenAddr)
	out = rtgWasmLocalGet(out, rtgWasm32LocalR8)
	out = rtgWasmI32Store(out, 2, 0)
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

func rtgWasm32AppendInstr(out []byte, ins rtgWasm32Instr, nextIndex int, targetIndex int, loopDepth int, exitDepth int, callStackBase int, frameSize int) []byte {
	if ins.op == rtgWasm32OpExit {
		out = rtgWasmLocalGet(out, rtgWasm32LocalRax)
		out = rtgWasmAppendCall(out, rtgWasm32ImportProcExit)
		return out
	}
	if ins.op == rtgWasm32OpBuildArgsEnv {
		out = rtgWasm32AppendBuildArgsEnv(out, ins.a, ins.b, ins.c)
		out = rtgWasm32SetPc(out, nextIndex)
		return rtgWasmBr(out, loopDepth)
	}
	if ins.op == rtgWasm32OpMovRegImm {
		out = rtgWasmAppendI32Const(out, ins.b)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpMovRegReg {
		out = rtgWasm32RegGet(out, ins.b)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpPushReg {
		out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmI32Store(out, 2, 0)
		out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
		out = rtgWasmAppendI32Const(out, 4)
		out = append(out, 0x6a)
		out = rtgWasmLocalSet(out, rtgWasm32LocalSp)
	} else if ins.op == rtgWasm32OpPushImm {
		out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
		out = rtgWasmAppendI32Const(out, ins.a)
		out = rtgWasmI32Store(out, 2, 0)
		out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
		out = rtgWasmAppendI32Const(out, 4)
		out = append(out, 0x6a)
		out = rtgWasmLocalSet(out, rtgWasm32LocalSp)
	} else if ins.op == rtgWasm32OpPopReg {
		out = rtgWasmLocalGet(out, rtgWasm32LocalSp)
		out = rtgWasmAppendI32Const(out, 4)
		out = append(out, 0x6b)
		out = rtgWasmLocalTee(out, rtgWasm32LocalSp)
		out = rtgWasmI32Load(out, 2, 0)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpLoadStack {
		out = rtgWasm32StackAddr(out, ins.b)
		out = rtgWasmI32Load(out, 2, 0)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpStoreStack {
		out = rtgWasm32StackAddr(out, ins.b)
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmI32Store(out, 2, 0)
	} else if ins.op == rtgWasm32OpLeaStack {
		out = rtgWasm32StackAddr(out, ins.b)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpLoadMem {
		out = rtgWasm32MemAddr(out, ins.b, ins.c)
		out = rtgWasm32LoadSized(out, ins.d)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpStoreMem {
		out = rtgWasm32MemAddr(out, ins.b, ins.c)
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasm32StoreSized(out, ins.d)
	} else if ins.op == rtgWasm32OpLoadIndex {
		out = rtgWasm32IndexAddr(out, ins.b, ins.c, ins.d, ins.e)
		out = rtgWasm32LoadSized(out, ins.f)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpStoreIndex {
		out = rtgWasm32IndexAddr(out, ins.b, ins.c, ins.d, ins.e)
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasm32StoreSized(out, ins.f)
	} else if ins.op >= rtgWasm32OpAddRegReg && ins.op <= rtgWasm32OpShrRegReg {
		out = rtgWasm32RegGet(out, ins.a)
		if ins.op == rtgWasm32OpAndNotRegReg {
			out = rtgWasm32RegGet(out, ins.b)
			out = rtgWasmAppend4(out, 0x41, 0x7f, 0x73, 0x71)
		} else {
			out = rtgWasm32RegGet(out, ins.b)
			out = rtgWasm32AppendBinaryOp(out, ins.op)
		}
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpAddRegImm || ins.op == rtgWasm32OpMulRegImm {
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmAppendI32Const(out, ins.b)
		if ins.op == rtgWasm32OpAddRegImm {
			out = append(out, 0x6a)
		} else {
			out = append(out, 0x6c)
		}
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpIncReg {
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmAppendI32Const(out, 1)
		out = append(out, 0x6a)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpIncMem || ins.op == rtgWasm32OpDecMem {
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmI32Load(out, 2, 0)
		out = rtgWasmAppendI32Const(out, 1)
		if ins.op == rtgWasm32OpIncMem {
			out = append(out, 0x6a)
		} else {
			out = append(out, 0x6b)
		}
		out = rtgWasmI32Store(out, 2, 0)
	} else if ins.op == rtgWasm32OpBoolNot {
		out = rtgWasm32RegGet(out, ins.a)
		out = append(out, 0x45)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpNegReg {
		out = rtgWasmAppendI32Const(out, 0)
		out = rtgWasm32RegGet(out, ins.a)
		out = append(out, 0x6b)
		out = rtgWasm32RegSet(out, ins.a)
	} else if ins.op == rtgWasm32OpCmpRegImm {
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasmAppendI32Const(out, ins.b)
		out = append(out, 0x6b)
		out = rtgWasmLocalSet(out, rtgWasm32LocalFlag)
	} else if ins.op == rtgWasm32OpCmpRegReg {
		out = rtgWasm32RegGet(out, ins.a)
		out = rtgWasm32RegGet(out, ins.b)
		out = append(out, 0x6b)
		out = rtgWasmLocalSet(out, rtgWasm32LocalFlag)
	} else if ins.op == rtgWasm32OpSetCond {
		out = rtgWasm32AppendCond(out, ins.a)
		out = rtgWasmLocalSet(out, rtgWasm32LocalRax)
	} else if ins.op == rtgWasm32OpJmp {
		out = rtgWasm32SetPc(out, targetIndex)
		return rtgWasmBr(out, loopDepth)
	} else if ins.op == rtgWasm32OpJz || ins.op == rtgWasm32OpJnz || ins.op == rtgWasm32OpJCond {
		if ins.op == rtgWasm32OpJz {
			out = rtgWasmLocalGet(out, rtgWasm32LocalFlag)
			out = append(out, 0x45)
		} else if ins.op == rtgWasm32OpJnz {
			out = rtgWasmLocalGet(out, rtgWasm32LocalFlag)
			out = rtgWasmAppend2(out, 0x45, 0x45)
		} else {
			out = rtgWasm32AppendCond(out, ins.a)
		}
		out = rtgWasmAppend2(out, 0x04, 0x40)
		out = rtgWasm32SetPc(out, targetIndex)
		out = append(out, 0x05)
		out = rtgWasm32SetPc(out, nextIndex)
		out = append(out, 0x0b)
		return rtgWasmBr(out, loopDepth)
	} else if ins.op == rtgWasm32OpCall {
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
	} else if ins.op == rtgWasm32OpRet {
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
	} else if ins.op == rtgWasm32OpSyscall {
		out = rtgWasm32AppendSyscall(out)
	}
	if loopDepth < 0 {
		return out
	}
	out = rtgWasm32SetPc(out, nextIndex)
	return rtgWasmBr(out, loopDepth)
}

func rtgWasm32CodeBody(a *rtgAsm, dataBase int, bssBase int) []byte {
	instrs := rtgWasm32Decode(a.code)
	pcIndex := rtgWasm32BuildPcIndex(instrs, len(a.code))
	blockStarts := rtgWasm32BuildBlockStarts(instrs, pcIndex)
	instrBlockIndex := rtgWasm32BuildInstrBlockIndex(blockStarts, len(instrs))
	frameSize := 32768
	exprStackBase := bssBase + a.bssSize + 65536
	callStackBase := exprStackBase + 16777216
	frameTop := callStackBase + 1048576 + 67108864
	body := make([]byte, 0, 4194304)
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
	body = rtgWasmAppendU32(body, len(blockStarts))
	for blockIndex := len(blockStarts) - 1; blockIndex >= 0; blockIndex-- {
		body = append(body, 0x0b)
		start := blockStarts[blockIndex]
		end := rtgWasm32BlockEnd(blockStarts, blockIndex, len(instrs))
		for i := start; i < end; i++ {
			ins := instrs[i]
			if i+1 < end {
				body = rtgWasm32AppendInstr(body, ins, 0, 0, -1, 0, callStackBase, frameSize)
			} else {
				nextIndex := rtgWasm32InstrIndexForPcFast(pcIndex, len(instrs), ins.next)
				nextBlock := rtgWasm32BlockForInstrFast(instrBlockIndex, nextIndex)
				targetBlock := 0
				if rtgWasm32OpHasTarget(ins.op) || ins.op == rtgWasm32OpJCond {
					targetPc := ins.a
					if ins.op == rtgWasm32OpJCond {
						targetPc = ins.b
					}
					targetIndex := rtgWasm32InstrIndexForPcFast(pcIndex, len(instrs), targetPc)
					targetBlock = rtgWasm32BlockForInstrFast(instrBlockIndex, targetIndex)
				}
				body = rtgWasm32AppendInstr(body, ins, nextBlock, targetBlock, blockIndex, blockIndex+1, callStackBase, frameSize)
			}
		}
	}
	body = rtgWasmBr(body, 1)
	body = append(body, 0x0b)
	body = append(body, 0x0b)
	body = append(body, 0x0b)
	return body
}

func rtgWasm32TypeSectionFull() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 7)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 4)
	out = rtgWasmAppend4(out, 0x7f, 0x7f, 0x7f, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 5)
	out = rtgWasmAppend5(out, 0x7f, 0x7f, 0x7f, 0x7e, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 9)
	out = rtgWasmAppend5(out, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f)
	out = rtgWasmAppend4(out, 0x7e, 0x7e, 0x7f, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	out = rtgWasmAppendU32(out, 0)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendU32(out, 0)
	out = append(out, 0x60)
	out = rtgWasmAppendU32(out, 2)
	out = rtgWasmAppend2(out, 0x7f, 0x7f)
	out = rtgWasmAppendU32(out, 1)
	out = append(out, 0x7f)
	return out
}

func rtgWasm32AppendImport(out []byte, name string, typ int) []byte {
	out = rtgWasmAppendName(out, "wasi_snapshot_preview1")
	out = rtgWasmAppendName(out, name)
	out = append(out, 0x00)
	return rtgWasmAppendU32(out, typ)
}

func rtgWasm32ImportSectionFull() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 12)
	out = rtgWasm32AppendImport(out, "fd_write", 0)
	out = rtgWasm32AppendImport(out, "fd_read", 0)
	out = rtgWasm32AppendImport(out, "fd_pread", 1)
	out = rtgWasm32AppendImport(out, "fd_pwrite", 1)
	out = rtgWasm32AppendImport(out, "path_open", 2)
	out = rtgWasm32AppendImport(out, "fd_close", 3)
	out = rtgWasm32AppendImport(out, "fd_fdstat_get", 6)
	out = rtgWasm32AppendImport(out, "args_sizes_get", 6)
	out = rtgWasm32AppendImport(out, "args_get", 6)
	out = rtgWasm32AppendImport(out, "environ_sizes_get", 6)
	out = rtgWasm32AppendImport(out, "environ_get", 6)
	out = rtgWasm32AppendImport(out, "proc_exit", 4)
	return out
}

func rtgWasm32FunctionSectionFull() []byte {
	var out []byte
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendU32(out, 5)
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
	var out []byte
	out = rtgWasmAppendU32(out, 2)
	out = rtgWasmAppendName(out, "memory")
	out = append(out, 0x02)
	out = rtgWasmAppendU32(out, 0)
	out = rtgWasmAppendName(out, "_start")
	out = append(out, 0x00)
	out = rtgWasmAppendU32(out, rtgWasm32StartFuncIndex)
	return out
}

func rtgWasm32CodeSectionFull(body []byte) []byte {
	out := make([]byte, 0, 4194304)
	out = rtgWasmAppendU32(out, 1)
	out = rtgWasmAppendByteVec(out, body)
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
	body := rtgWasm32CodeBody(a, dataBase, bssBase)
	memSize := bssBase + a.bssSize + 65536 + 16777216 + 1048576 + 67108864 + 65536
	out := make([]byte, 0, 4194304)
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
	out = rtgWasmAppendSection(out, 3, rtgWasm32FunctionSectionFull())
	out = rtgWasmAppendSection(out, 5, rtgWasm32MemorySectionFull(memSize))
	out = rtgWasmAppendSection(out, 7, rtgWasm32ExportSectionFull())
	out = rtgWasmAppendSection(out, 10, rtgWasm32CodeSectionFull(body))
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
	oldBreak := g.breakDepth
	oldContinue := g.continueDepth
	oldCurrent := g.currentFunc
	oldReturnStruct := g.returnStruct
	oldStackUsed := g.stackUsed
	oldGotoLabels := g.gotoLabels
	oldLastRangeReturns := g.lastRangeReturns
	var locals []rtgLocalInfo
	var gotoLabels []rtgGlobalInfo
	g.locals = locals
	g.gotoLabels = gotoLabels
	g.breakDepth = 0
	g.continueDepth = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.stackUsed = 0
	rtgAsmMarkLabel(a, g.funcLabels[fnInfoIndex])
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgWasm32EmitStack(a, rtgWasm32OpStoreStack, rtgWasm32RegRdi, g.returnStruct)
	}
	if !rtgBindFunctionParams(g, fnInfoIndex) {
		return false
	}
	if !rtgEmitLinearRange(g, fn.bodyStart+1, fn.bodyEnd) {
		return false
	}
	if !g.lastRangeReturns {
		rtgAsmMovRaxImm(a, 0)
		rtgAsmLeave(a)
		rtgAsmRet(a)
	}
	g.locals = oldLocals
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
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
}

func rtgWasm32EmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
	a := &g.asm
	p := g.prog
	if tok < 0 || tok >= len(p.toks) {
		return false
	}
	start := p.toks[tok].start
	end := p.toks[tok].end
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
			rtgWasm32EmitRegReg(a, rtgWasm32OpAndNotRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
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
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRax, rtgWasm32RegRcx)
		rtgAsmSarRaxImm(a, 2)
		return true
	}
	if rtgTokCharIs(p, e.tok, '/') {
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmShlRaxImm(a, 2)
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	return rtgAmd64EmitFloatBinaryExpr(g, ep, idx)
}

func rtgWasm32EnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRax, rtgWasm32RegRdi, 0, 4)
	rtgWasm32EmitRegReg(a, rtgWasm32OpMulRegReg, rtgWasm32RegRcx, rtgWasm32RegRdx)
	rtgWasm32AsmAddRaxRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpLoadMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
	rtgWasm32AsmIncRcx(a)
	rtgWasm32EmitMem(a, rtgWasm32OpStoreMem, rtgWasm32RegRcx, rtgWasm32RegRsi, 0, 4)
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
	rtgAsmMovRaxImm(a, 0)
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
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}
