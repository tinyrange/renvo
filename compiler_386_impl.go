//go:build rtg_target_386
// +build rtg_target_386

package main

const rtgNativeIntSize = 4

func rtgEmitScalarFunction(g *rtgLinearGen, fnInfoIndex int) bool {
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
	rtgAsmEmit32(a, 0x008000c8)
	if rtgTypeIsStruct(g.meta, metaFn.resultType) {
		g.returnStruct = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStackMem(a, g.returnStruct, 0x89, 0x5d, 0x9d)
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

func rtgBindFunctionParams(g *rtgLinearGen, fnIndex int) bool {
	meta := g.meta
	fn := &meta.funcs[fnIndex]
	reg := 0
	if rtgTypeIsStruct(meta, fn.resultType) {
		reg = 1
	}
	for i := 0; i < fn.paramCount; i++ {
		param := &meta.params[fn.firstParam+i]
		offset := rtgAddTypedLocal(g, param.nameStart, param.nameEnd, param.typ)
		if rtgTypeIsSlice(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) || !rtgStoreParamWord(g, reg+2, offset-16) {
				return false
			}
			reg += 3
			continue
		}
		if rtgTypeIsString(meta, param.typ) {
			if !rtgStoreParamWord(g, reg, offset) || !rtgStoreParamWord(g, reg+1, offset-8) {
				return false
			}
			reg += 2
			continue
		}
		if rtgTypeIsStruct(meta, param.typ) {
			size := rtgTypeSize(meta, param.typ)
			for at := 0; at < size; at += 8 {
				if !rtgStoreParamWord(g, reg, offset-at) {
					return false
				}
				reg++
			}
			continue
		}
		if !rtgStoreParamWord(g, reg, offset) {
			return false
		}
		reg++
	}
	return true
}

func rtgStoreParamWord(g *rtgLinearGen, reg int, offset int) bool {
	a := &g.asm
	if reg == 0 {
		rtgAsmStackMem(a, offset, 0x89, 0x5d, 0x9d)
		return true
	}
	if reg == 1 {
		rtgAsmStackMem(a, offset, 0x8948, 0x75, 0xb5)
		return true
	}
	if reg == 2 {
		rtgAsmStoreRdxStack(a, offset)
		return true
	}
	if reg == 3 {
		rtgAsmStackMem(a, offset, 0x8948, 0x4d, 0x8d)
		return true
	}
	if reg == 4 {
		rtgAsmStoreRaxStack(a, offset)
		return true
	}
	if reg == 5 {
		rtgAsmStackMem(a, offset, 0x89, 0x7d, 0xbd)
		return true
	}
	rtgAsmEmit16(a, 0x858b)
	rtgAsmEmit32(a, 8+(reg-6)*4)
	rtgAsmStoreRaxStack(a, offset)
	return true
}

func rtgAsmImmFits8Signed(imm int) bool {
	return imm >= -128 && imm <= 127
}

func rtgAsmMovRaxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc031)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRax(a)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		rtgAsmEmit8(a, 0xb8)
		rtgAsmEmit32(a, imm)
		return
	}
	rtgAsmEmit8(a, 0xb8)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovRaxIntToken(a *rtgAsm, p *rtgProgram, tokIndex int) {
	value := rtgParseIntToken(p, tokIndex)
	if rtgIntTokenNeedsMovabs(p, tokIndex) {
		rtgAsmMovRaxImm64(a, value)
		return
	}
	rtgAsmMovRaxImm(a, value)
}

func rtgAsmMovRaxImm64(a *rtgAsm, imm int) {
	rtgAsmMovRaxImm(a, imm)
}

func rtgIntTokenNeedsMovabs(p *rtgProgram, tokIndex int) bool {
	tok := &p.toks[tokIndex]
	start := tok.start
	if tok.end-start > 2 && p.src[start] == '0' {
		return false
	}
	digits := tok.end - start
	if digits > 10 {
		return true
	}
	if digits < 10 {
		return false
	}
	limit := "2147483647"
	for i := 0; i < 10; i++ {
		c := p.src[start+i]
		if c > limit[i] {
			return true
		}
		if c < limit[i] {
			return false
		}
	}
	return false
}

func rtgAsmMovRdxImm(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xd231)
		return
	}
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		rtgAsmPopRdx(a)
		return
	}
	if imm >= 0 {
		if imm <= 2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	} else {
		if imm >= -2147483647 {
			rtgAsmEmit8(a, 0xba)
			rtgAsmEmit32(a, imm)
			return
		}
	}
	rtgAsmEmit8(a, 0xba)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMovRaxDataAddr(a *rtgAsm, dataOff int) {
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, dataOff, 0)
}

func rtgAsmMovRaxBssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xb8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgAsmMovR10BssAddr(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xbb)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgAsmLoadRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xa1)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgAsmStoreRaxBss(a *rtgAsm, bssOff int) {
	rtgAsmEmit8(a, 0xa3)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddAbsReloc(a, at, bssOff, rtgAbsBssReloc)
}

func rtgAsmMovRdiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtgAsmMovRdxRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0x5a50)
}

func rtgAsmMovRaxRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xd089)
}

func rtgAsmMovRsiRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc189)
}

func rtgAsmMovArg1Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc689)
}

func rtgAsmMovRcxRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0x5950)
}

func rtgAsmMovR8Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtgAsmMovR9Rax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc389)
}

func rtgAsmMovRcxRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x5952)
}

func rtgAsmAddRdxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xca01)
}

func rtgAsmSyscall(a *rtgAsm) {
	rtgAsmEmit16(a, 0x80cd)
}

func rtgAsmPushRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x50)
}

func rtgAsmPushRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x51)
}

func rtgAsmPushRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x52)
}

func rtgAsmPushImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit2(a, 0x6a, imm)
		return
	}
	if imm >= -2147483647 && imm <= 2147483647 {
		rtgAsmEmit8(a, 0x68)
		rtgAsmEmit32(a, imm)
		return
	}
	rtgAsmMovRaxImm(a, imm)
	rtgAsmPushRax(a)
}

func rtgAsmPushSliceRegs(a *rtgAsm) {
	rtgAsmPushRcx(a)
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}

func rtgAsmPushStringRegs(a *rtgAsm) {
	rtgAsmPushRdx(a)
	rtgAsmPushRax(a)
}

func rtgAsmPopRax(a *rtgAsm) {
	rtgAsmEmit8(a, 0x58)
}

func rtgAsmPopRcx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x59)
}

func rtgAsmPopRdx(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5a)
}

func rtgAsmPopRdi(a *rtgAsm) {
	rtgAsmEmit8(a, 0x5b)
}

func rtgAsmStackMem(a *rtgAsm, offset int, base int, disp8 int, disp32 int) {
	if base > 0xff {
		rtgAsmEmit8(a, base>>8)
	} else {
		rtgAsmEmit8(a, base)
	}
	if offset >= 0 && offset <= 128 {
		rtgAsmEmit8(a, disp8)
		rtgAsmEmit8(a, -offset)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, -offset)
}

func rtgAsmStoreRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x45, 0x85)
}

func rtgAsmStoreRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8948, 0x55, 0x95)
}

func rtgAsmLoadRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x45, 0x85)
}

func rtgAsmLeaRaxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x45, 0x85)
}

func rtgAsmLeaRdiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x7d, 0xbd)
}

func rtgAsmLeaRsiStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8d48, 0x75, 0xb5)
}

func rtgAsmLoadRdxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x55, 0x95)
}

func rtgAsmAddRdxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit3(a, 0x83, 0xc2, imm)
		return
	}
	rtgAsmEmit16(a, 0xc281)
	rtgAsmEmit32(a, imm)
}

func rtgAsmMemDisp(a *rtgAsm, disp int, op int, disp8 int, disp32 int) {
	if op > 0xff {
		rtgAsmEmit8(a, op>>8)
	} else {
		rtgAsmEmit8(a, op)
	}
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, disp8, disp)
		return
	}
	rtgAsmEmit8(a, disp32)
	rtgAsmEmit32(a, disp)
}

func rtgAsmRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit8(a, 0x02)
		return
	}
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit2(a, 0x42, disp)
		return
	}
	rtgAsmEmit8(a, 0x82)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRcxStack(a *rtgAsm, offset int) {
	rtgAsmStackMem(a, offset, 0x8b48, 0x4d, 0x8d)
}

func rtgAsmStoreSliceStack(a *rtgAsm, offset int) {
	rtgAsmStoreRaxStack(a, offset)
	rtgAsmStoreRdxStack(a, offset-8)
	rtgAsmStackMem(a, offset-16, 0x8948, 0x4d, 0x8d)
}

func rtgAsmPopStoreStringMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
}

func rtgAsmPopStoreSliceMemRdx(a *rtgAsm, disp int) {
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+8)
	rtgAsmPopRax(a)
	rtgAsmStoreRaxMemRdxDisp(a, disp+16)
}

func rtgAsmLoadQwordRaxIndexRcx8(a *rtgAsm) {
	rtgAsmEmit24(a, 0xc8048b)
}

func rtgAsmLoadQwordRaxIndexRcxDisp(a *rtgAsm, disp int) {
	rtgAsmEmit8(a, 0x8b)
	if rtgAsmImmFits8Signed(disp) {
		rtgAsmEmit3(a, 0x44, 0x8, disp)
		return
	}
	rtgAsmEmit16(a, 0x0884)
	rtgAsmEmit32(a, disp)
}

func rtgAsmLoadRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit16(a, 0x028b)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8b48, 0x42, 0x82)
}

func rtgAsmLoadRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size == 1 {
		rtgAsmEmit16(a, 0xb60f)
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0xbf0f)
		rtgAsmRdxDisp(a, disp)
		return
	}
	rtgAsmLoadRaxMemRdxDisp(a, disp)
}

func rtgAsmLoadByteRaxIndexRcx(a *rtgAsm) {
	rtgAsmEmit32(a, 0x0804b60f)
}

func rtgAsmLoadRaxIndexRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmLoadByteRaxIndexRcx(a)
		return
	}
	if size == 2 {
		rtgAsmEmit32(a, 0x4804bf0f)
		return
	}
	if size == 4 {
		rtgAsmEmit24(a, 0x88048b)
		return
	}
	rtgAsmLoadQwordRaxIndexRcx8(a)
}

func rtgAsmStoreRaxMemRdxRcx8(a *rtgAsm) {
	rtgAsmEmit24(a, 0xca0489)
}

func rtgAsmStoreRaxMemRdxDisp(a *rtgAsm, disp int) {
	if disp == 0 {
		rtgAsmEmit16(a, 0x0289)
		return
	}
	rtgAsmMemDisp(a, disp, 0x8948, 0x42, 0x82)
}

func rtgAsmStoreRaxMemRdxDispSize(a *rtgAsm, disp int, size int) {
	if size == 1 {
		rtgAsmEmit8(a, 0x88)
		rtgAsmRdxDisp(a, disp)
		return
	}
	if size == 2 {
		rtgAsmEmit16(a, 0x8966)
		rtgAsmRdxDisp(a, disp)
		return
	}
	rtgAsmStoreRaxMemRdxDisp(a, disp)
}

func rtgAsmStoreAlMemRdxRcx1(a *rtgAsm) {
	rtgAsmEmit24(a, 0x0a0488)
}

func rtgAsmStoreRaxMemRdxRcxSize(a *rtgAsm, size int) {
	if size == 1 {
		rtgAsmStoreAlMemRdxRcx1(a)
		return
	}
	if size == 2 {
		rtgAsmEmit32(a, 0x4a048966)
		return
	}
	if size == 4 {
		rtgAsmEmit24(a, 0x8a0489)
		return
	}
	rtgAsmStoreRaxMemRdxRcx8(a)
}

func rtgAsmNormalizeRaxForKind(a *rtgAsm, kind int) {
	if kind == rtgTypeByte || kind == rtgTypeBool {
		rtgAsmEmit8(a, 0x25)
		rtgAsmEmit32(a, 0xff)
		return
	}
	if kind == rtgTypeInt16 {
		rtgAsmEmit8(a, 0x98)
	}
}

func rtgAsmIncMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x02ff)
}

func rtgAsmDecMemRdx(a *rtgAsm) {
	rtgAsmEmit16(a, 0x0aff)
}

func rtgAsmIncRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc1ff)
}

func rtgAsmIncRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc0ff)
}

func rtgAsmBoolNotRax(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc085)
	rtgAsmEmit24(a, 0xc0940f)
	rtgAsmEmit24(a, 0xc0b60f)
}

func rtgAsmCmpRaxImm8(a *rtgAsm, imm int) {
	if imm == 0 {
		rtgAsmEmit16(a, 0xc085)
		return
	}
	rtgAsmEmit3(a, 0x83, 0xf8, imm)
}

func rtgAsmAddRaxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc801)
}

func rtgAsmSubRaxRcx(a *rtgAsm) {
	rtgAsmEmit16(a, 0xc829)
}

func rtgAsmShlRcxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xe1, imm)
}

func rtgAsmShlRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xe0, imm)
}

func rtgAsmSarRaxImm(a *rtgAsm, imm int) {
	rtgAsmEmit3(a, 0xc1, 0xf8, imm)
}

func rtgAsmImulRcxImm(a *rtgAsm, imm int) {
	if rtgAsmImmFits8Signed(imm) {
		rtgAsmEmit3(a, 0x6b, 0xc9, imm)
		return
	}
	rtgAsmEmit16(a, 0xc969)
	rtgAsmEmit32(a, imm)
}

func rtgAsmDivLeftRcxRightRax(a *rtgAsm, mod bool) {
	rtgAsmEmit16(a, 0xc389)
	rtgAsmEmit16(a, 0xc889)
	rtgAsmEmit8(a, 0x99)
	rtgAsmEmit16(a, 0xfbf7)
	if mod {
		rtgAsmEmit16(a, 0xd089)
	}
}

func rtgAsmCmpRcxRaxSet(a *rtgAsm, setcc int) {
	rtgAsmEmit24(a, 0x0fc139)
	rtgAsmEmit3(a, setcc, 0xc0, 0xf)
	rtgAsmEmit16(a, 0xc0b6)
}

func rtgAsmRet(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc3)
}

func rtgAsmLeave(a *rtgAsm) {
	rtgAsmEmit8(a, 0xc9)
}

func rtgAsmCallLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe8)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgAsmJmpLabel(a *rtgAsm, label int) {
	rtgAsmEmit8(a, 0xe9)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgAsmJzLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 0x840f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgAsmJnzLabel(a *rtgAsm, label int) {
	rtgAsmEmit16(a, 0x850f)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

type rtgLocalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
	typ       int
	size      int
}

type rtgGlobalInfo struct {
	nameStart int
	nameEnd   int
	offset    int
}

type rtgSliceLocation struct {
	offset int
	typ    int
	expr   int
	mem    bool
	global bool
	ok     bool
}

type rtgLinearGen struct {
	prog               *rtgProgram
	meta               *rtgMeta
	asm                rtgAsm
	funcLabels         []int
	currentFunc        int
	returnStruct       int
	locals             []rtgLocalInfo
	stackUsed          int
	globals            []rtgGlobalInfo
	gotoLabels         []rtgGlobalInfo
	breakLabels        []int
	continueLabels     []int
	breakDepth         int
	continueDepth      int
	streqLabel         int
	streqEmitted       bool
	append8Label       int
	append8Emitted     bool
	append64Label      int
	append64Emitted    bool
	appendAddrLabel    int
	appendAddrEmitted  bool
	appendBytesLabel   int
	appendBytesEmitted bool
	copyWordsLabel     int
	copyWordsEmitted   bool
	lastRangeReturns   bool
	scopeBase          int
	constEvalIota      int
	constEvalIotaValid int
}

func rtgAddStringData(g *rtgLinearGen, msg []byte) int {
	msgOff := len(g.asm.data)
	for i := 0; i < len(msg); i++ {
		g.asm.data = append(g.asm.data, msg[i])
	}
	g.asm.data = append(g.asm.data, 0)
	return msgOff
}

func rtgEmitLinearRange(g *rtgLinearGen, start int, end int) bool {
	var bp rtgBodyParse
	var stmts []rtgStmt
	bp.prog = g.prog
	bp.stmts = stmts
	bp.ok = true
	i := start
	lastKind := 0
	for bp.ok && i < end {
		if i < 0 || i >= len(bp.prog.toks) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, ';') {
			i++
			continue
		}
		if bp.prog.toks[i].start < 0 || bp.prog.toks[i].start >= len(bp.prog.src) {
			return true
		}
		if rtgTokIsKind(bp.prog, i, rtgTokEOF) {
			return true
		}
		if rtgTokCharIs(bp.prog, i, '}') {
			return true
		}
		before := len(bp.stmts)
		next := rtgParseOneStatement(&bp, i, end)
		if !bp.ok || next <= i || len(bp.stmts) <= before {
			return false
		}
		stmt := bp.stmts[len(bp.stmts)-1]
		lastKind = stmt.kind
		i = next
		if !rtgEmitLinearStmt(g, &stmt) {
			return false
		}
	}
	g.lastRangeReturns = lastKind == rtgStmtReturn
	if !bp.ok {
		return false
	}
	return true
}

func rtgEmitScopedRange(g *rtgLinearGen, start int, end int) bool {
	oldLocals := g.locals
	oldScopeBase := g.scopeBase
	g.scopeBase = len(oldLocals)
	if !rtgEmitLinearRange(g, start, end) {
		return false
	}
	g.locals = oldLocals
	g.scopeBase = oldScopeBase
	return true
}

func rtgEmitLinearStmt(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.kind == rtgStmtExpr {
		if rtgEmitLinearPrintStmt(g, stmt) {
			return true
		}
		if rtgEmitLinearIncDec(g, stmt.startTok, stmt.endTok) {
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		root := &ep.exprs[rootIndex]
		if root.kind != rtgExprCall {
			return false
		}
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtVar || stmt.kind == rtgStmtShort || stmt.kind == rtgStmtAssign {
		if !rtgEmitLinearAssign(g, stmt) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtReturn {
		if stmt.exprStart == stmt.exprEnd {
			rtgAsmMovRaxImm(a, 0)
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		resultType := g.meta.funcs[g.currentFunc].resultType
		if rtgTypeIsTuple(g.meta, resultType) {
			if !rtgEmitTupleReturn(g, stmt.exprStart, stmt.exprEnd) {
				return false
			}
			rtgAsmLeave(a)
			rtgAsmRet(a)
			return true
		}
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if rtgTypeIsStruct(g.meta, resultType) {
			if !rtgEmitStructReturnExpr(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsSlice(g.meta, resultType) {
			if !rtgEmitSliceValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else if rtgTypeIsString(g.meta, resultType) {
			if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
				return false
			}
		} else {
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			resultResolved := rtgResolveType(g.meta, resultType)
			rtgAsmNormalizeRaxForKind(a, resultResolved.kind)
		}
		rtgAsmLeave(a)
		rtgAsmRet(a)
		return true
	}
	if stmt.kind == rtgStmtIf {
		return rtgEmitLinearIf(g, stmt)
	}
	if stmt.kind == rtgStmtFor {
		return rtgEmitLinearFor(g, stmt)
	}
	if stmt.kind == rtgStmtSwitch {
		return rtgEmitLinearSwitch(g, stmt)
	}
	if stmt.kind == rtgStmtBlock {
		if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
			return false
		}
		return true
	}
	if stmt.kind == rtgStmtGoto {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmJmpLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtLabel {
		label := rtgFindOrCreateGotoLabel(g, stmt.nameStart, stmt.nameEnd)
		rtgAsmMarkLabel(a, label)
		return true
	}
	if stmt.kind == rtgStmtBreak {
		if g.breakDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.breakLabels[g.breakDepth-1])
		return true
	}
	if stmt.kind == rtgStmtContinue {
		if g.continueDepth == 0 {
			return false
		}
		rtgAsmJmpLabel(a, g.continueLabels[g.continueDepth-1])
		return true
	}
	return false
}

func rtgFindOrCreateGotoLabel(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.gotoLabels); i++ {
		info := g.gotoLabels[i]
		if rtgBytesEqualRange(g.prog.src, info.nameStart, info.nameEnd, nameStart, nameEnd) {
			return info.offset
		}
	}
	label := rtgAsmNewLabel(&g.asm)
	g.gotoLabels = append(g.gotoLabels, rtgGlobalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: label})
	return label
}

func rtgEmitLinearIf(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	endLabel := rtgAsmNewLabel(a)
	elseLabel := endLabel
	if stmt.elseStart > 0 {
		elseLabel = rtgAsmNewLabel(a)
	}
	if !rtgEmitJumpIfFalse(g, &ep, rootIndex, elseLabel) {
		return false
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	thenReturns := g.lastRangeReturns
	if stmt.elseStart <= 0 {
		rtgAsmMarkLabel(a, endLabel)
		return true
	}
	if !thenReturns {
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, elseLabel)
	if rtgTokIsKind(p, stmt.elseStart, rtgTokIf) && rtgTokIsKind(p, stmt.elseStart-1, rtgTokElse) {
		var nested rtgBodyParse
		var stmts []rtgStmt
		nested.prog = p
		nested.stmts = stmts
		nested.ok = true
		next := rtgParseOneStatement(&nested, stmt.elseStart, stmt.elseEnd)
		if !nested.ok || next != stmt.elseEnd || len(nested.stmts) != 1 {
			return false
		}
		nestedStmt := nested.stmts[0]
		if !rtgEmitLinearStmt(g, &nestedStmt) {
			return false
		}
	} else if !rtgEmitScopedRange(g, stmt.elseStart, stmt.elseEnd) {
		return false
	}
	rtgAsmMarkLabel(a, endLabel)
	return true
}

func rtgEmitLinearFor(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	semi1 := rtgFindTokenTextInRange(p, stmt.exprStart, stmt.exprEnd, ';')
	if semi1 >= stmt.exprStart {
		return rtgEmitLinearClassicFor(g, stmt, semi1)
	}
	startLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, startLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if stmt.exprStart < stmt.exprEnd {
		ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}

func rtgEmitLinearSwitch(g *rtgLinearGen, stmt *rtgStmt) bool {
	a := &g.asm
	p := g.prog
	if stmt.exprStart >= stmt.exprEnd {
		return false
	}
	ep := rtgParseExpression(p, stmt.exprStart, stmt.exprEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	switchType := rtgInferParsedExprType(g, &ep, rootIndex)
	stringSwitch := rtgTypeIsString(g.meta, switchType)
	valueOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOffset := 0
	if stringSwitch {
		lenOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
		rtgAsmStoreRdxStack(a, lenOffset)
	} else {
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmStoreRaxStack(a, valueOffset)
	}

	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.breakDepth = len(g.breakLabels)

	var clauseStarts []int
	var clauseLabels []int
	defaultLabel := endLabel
	hasDefault := false
	i := stmt.bodyStart
	for i < stmt.bodyEnd {
		clause := rtgFindNextSwitchClause(p, i, stmt.bodyEnd)
		if clause >= stmt.bodyEnd {
			break
		}
		label := rtgAsmNewLabel(a)
		clauseStarts = append(clauseStarts, clause)
		clauseLabels = append(clauseLabels, label)
		if rtgTokIsKind(p, clause, rtgTokDefault) {
			defaultLabel = label
			hasDefault = true
		}
		i = clause + 1
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		if rtgTokIsKind(p, clause, rtgTokCase) {
			if !rtgEmitSwitchCaseTests(g, stmt, clause, valueOffset, lenOffset, stringSwitch, clauseLabels[i]) {
				return false
			}
		}
	}
	if hasDefault {
		rtgAsmJmpLabel(a, defaultLabel)
	} else {
		rtgAsmJmpLabel(a, endLabel)
	}
	for i := 0; i < len(clauseStarts); i++ {
		clause := clauseStarts[i]
		colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
		if colon <= clause {
			return false
		}
		bodyEnd := rtgFindNextSwitchClause(p, colon+1, stmt.bodyEnd)
		rtgAsmMarkLabel(a, clauseLabels[i])
		if !rtgEmitScopedRange(g, colon+1, bodyEnd) {
			return false
		}
		rtgAsmJmpLabel(a, endLabel)
	}
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	return true
}

func rtgEmitSwitchCaseTests(g *rtgLinearGen, stmt *rtgStmt, clause int, valueOffset int, lenOffset int, stringSwitch bool, matchLabel int) bool {
	a := &g.asm
	p := g.prog
	colon := rtgFindSwitchClauseColon(p, clause+1, stmt.bodyEnd)
	if colon <= clause+1 {
		return false
	}
	i := clause + 1
	for i < colon {
		valueEnd := rtgFindExprBoundary(p, i, colon)
		if valueEnd <= i {
			return false
		}
		ep := rtgParseExpression(p, i, valueEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if stringSwitch {
			if !rtgEmitSwitchStringCaseTest(g, valueOffset, lenOffset, &ep, rootIndex, matchLabel) {
				return false
			}
		} else {
			rtgAsmLoadRaxStack(a, valueOffset)
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, &ep, rootIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			rtgAsmCmpRcxRaxSet(a, 0x94)
			rtgAsmCmpRaxImm8(a, 0)
			rtgAsmJnzLabel(a, matchLabel)
		}
		i = valueEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}

func rtgEmitSwitchStringCaseTest(g *rtgLinearGen, valueOffset int, lenOffset int, ep *rtgExprParse, idx int, matchLabel int) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmLoadRaxStack(a, valueOffset)
	rtgAsmMovRdiRax(a)
	rtgAsmLoadRaxStack(a, lenOffset)
	rtgAsmMovArg1Rax(a)
	rtgAsmCallLabel(a, label)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, matchLabel)
	return true
}

func rtgFindNextSwitchClause(p *rtgProgram, start int, end int) int {
	depth := 0
	i := start
	for i < end {
		if depth == 0 && (rtgTokIsKind(p, i, rtgTokCase) || rtgTokIsKind(p, i, rtgTokDefault)) {
			return i
		}
		if rtgTokCharIs(p, i, '{') {
			depth++
		} else if rtgTokCharIs(p, i, '}') {
			if depth > 0 {
				depth--
			}
		}
		i++
	}
	return end
}

func rtgFindSwitchClauseColon(p *rtgProgram, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ':') {
			return i
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace == 0 {
				return end
			}
			brace--
		}
		i++
	}
	return end
}

func rtgEmitLinearClassicFor(g *rtgLinearGen, stmt *rtgStmt, semi1 int) bool {
	a := &g.asm
	p := g.prog
	semi2 := rtgFindTokenTextInRange(p, semi1+1, stmt.exprEnd, ';')
	if semi2 <= semi1 {
		return false
	}
	if !rtgEmitLinearSimpleRange(g, stmt.exprStart, semi1) {
		return false
	}
	startLabel := rtgAsmNewLabel(a)
	postLabel := rtgAsmNewLabel(a)
	endLabel := rtgAsmNewLabel(a)
	oldBreakDepth := g.breakDepth
	oldContinueDepth := g.continueDepth
	g.breakLabels = append(g.breakLabels, endLabel)
	g.continueLabels = append(g.continueLabels, postLabel)
	g.breakDepth = len(g.breakLabels)
	g.continueDepth = len(g.continueLabels)
	rtgAsmMarkLabel(a, startLabel)
	if semi1+1 < semi2 {
		ep := rtgParseExpression(p, semi1+1, semi2)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitJumpIfFalse(g, &ep, rootIndex, endLabel) {
			return false
		}
	}
	if !rtgEmitScopedRange(g, stmt.bodyStart, stmt.bodyEnd) {
		return false
	}
	rtgAsmMarkLabel(a, postLabel)
	if !rtgEmitLinearSimpleRange(g, semi2+1, stmt.exprEnd) {
		return false
	}
	rtgAsmJmpLabel(a, startLabel)
	rtgAsmMarkLabel(a, endLabel)
	g.breakDepth = oldBreakDepth
	g.continueDepth = oldContinueDepth
	return true
}

func rtgEmitLinearSimpleRange(g *rtgLinearGen, start int, end int) bool {
	p := g.prog
	if start >= end {
		return true
	}
	if rtgEmitLinearIncDec(g, start, end) {
		return true
	}
	assignTok := rtgFindAssignmentToken(p, start, end)
	if assignTok > start {
		kind := rtgStmtAssign
		if rtgTok2Is(p, assignTok, ':', '=') {
			kind = rtgStmtShort
		}
		nameStart := 0
		nameEnd := 0
		if rtgTokIsKind(p, start, rtgTokIdent) {
			nameStart = p.toks[start].start
			nameEnd = p.toks[start].end
		}
		stmt := rtgStmt{kind: kind, startTok: start, endTok: end, exprStart: assignTok + 1, exprEnd: end, nameStart: nameStart, nameEnd: nameEnd}
		return rtgEmitLinearAssign(g, &stmt)
	}
	return false
}

func rtgEmitLinearIncDec(g *rtgLinearGen, start int, end int) bool {
	a := &g.asm
	p := g.prog
	if start+2 > end {
		return false
	}
	opTok := end - 1
	if !rtgTok2Is(p, opTok, '+', '+') && !rtgTok2Is(p, opTok, '-', '-') {
		return false
	}
	ep := rtgParseExpression(p, start, opTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	inc := rtgTok2Is(p, opTok, '+', '+')
	if root.kind == rtgExprIdent {
		localOffset := rtgFindLocalOffset(g, root.nameStart, root.nameEnd)
		if localOffset >= 0 {
			rtgAsmEmit16(a, 0xff48)
			if localOffset >= 0 && localOffset <= 128 {
				if inc {
					rtgAsmEmit8(a, 0x45)
				} else {
					rtgAsmEmit8(a, 0x4d)
				}
				rtgAsmEmit8(a, -localOffset)
			} else {
				if inc {
					rtgAsmEmit8(a, 0x85)
				} else {
					rtgAsmEmit8(a, 0x8d)
				}
				rtgAsmEmit32(a, -localOffset)
			}
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		if inc {
			rtgAsmEmit24(a, 0x05ff48)
		} else {
			rtgAsmEmit24(a, 0x0dff48)
		}
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, globalOffset, rtgAbsBssReloc)
		return true
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		if inc {
			rtgAsmIncMemRdx(a)
		} else {
			rtgAsmDecMemRdx(a)
		}
		return true
	}
	return false
}

func rtgEmitRaxRcxOp(g *rtgLinearGen, tok int) bool {
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
		rtgAsmAddRaxRcx(a)
		return true
	}
	if c0 == '-' {
		rtgAsmEmit16(a, 0xc129)
		rtgAsmEmit16(a, 0xc889)
		return true
	}
	if c0 == '*' {
		rtgAsmEmit24(a, 0xc1af0f)
		return true
	}
	if c0 == '/' {
		rtgAsmDivLeftRcxRightRax(a, false)
		return true
	}
	if c0 == '%' {
		rtgAsmDivLeftRcxRightRax(a, true)
		return true
	}
	if c0 == '&' {
		if c1 == '^' {
			rtgAsmEmit16(a, 0xd0f7)
			rtgAsmEmit16(a, 0xc821)
		} else {
			rtgAsmEmit16(a, 0xc821)
		}
		return true
	}
	if c0 == '|' {
		rtgAsmEmit16(a, 0xc809)
		return true
	}
	if c0 == '^' {
		rtgAsmEmit16(a, 0xc831)
		return true
	}
	if c0 == '<' {
		if c1 == '<' {
			rtgAsmEmit16(a, 0xca89)
			rtgAsmEmit16(a, 0xc189)
			rtgAsmEmit16(a, 0xd089)
			rtgAsmEmit16(a, 0xe0d3)
		} else if c1 == '=' {
			rtgAsmCmpRcxRaxSet(a, 0x9e)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9c)
		}
		return true
	}
	if c0 == '>' {
		if c1 == '>' {
			rtgAsmEmit16(a, 0xca89)
			rtgAsmEmit16(a, 0xc189)
			rtgAsmEmit16(a, 0xd089)
			rtgAsmEmit16(a, 0xf8d3)
		} else if c1 == '=' {
			rtgAsmCmpRcxRaxSet(a, 0x9d)
		} else {
			rtgAsmCmpRcxRaxSet(a, 0x9f)
		}
		return true
	}
	if c0 == '=' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x94)
		return true
	}
	if c0 == '!' && c1 == '=' {
		rtgAsmCmpRcxRaxSet(a, 0x95)
		return true
	}
	return false
}

func rtgEmitJumpIfFalse(g *rtgLinearGen, ep *rtgExprParse, idx int, falseLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '&', '&') {
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			return rtgEmitJumpIfFalse(g, ep, e.right, falseLabel)
		}
		if rtgTok2Is(p, e.tok, '|', '|') {
			trueLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			if !rtgEmitJumpIfFalse(g, ep, e.right, falseLabel) {
				return false
			}
			rtgAsmMarkLabel(a, trueLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, falseLabel, false) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfTrue(g, ep, e.left, falseLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJzLabel(a, falseLabel)
	return true
}

func rtgEmitJumpIfTrue(g *rtgLinearGen, ep *rtgExprParse, idx int, trueLabel int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '|', '|') {
			if !rtgEmitJumpIfTrue(g, ep, e.left, trueLabel) {
				return false
			}
			return rtgEmitJumpIfTrue(g, ep, e.right, trueLabel)
		}
		if rtgTok2Is(p, e.tok, '&', '&') {
			falseLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, e.left, falseLabel) {
				return false
			}
			if !rtgEmitJumpIfTrue(g, ep, e.right, trueLabel) {
				return false
			}
			rtgAsmMarkLabel(a, falseLabel)
			return true
		}
		if rtgEmitCompareJump(g, ep, e, trueLabel, true) {
			return true
		}
	}
	if e.kind == rtgExprUnary && rtgTokCharIs(p, e.tok, '!') {
		return rtgEmitJumpIfFalse(g, ep, e.left, trueLabel)
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, trueLabel)
	return true
}

func rtgEmitCompareJump(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr, label int, jumpIfTrue bool) bool {
	p := g.prog
	if e.tok < 0 || e.tok >= len(p.toks) {
		return false
	}
	start := p.toks[e.tok].start
	end := p.toks[e.tok].end
	if start >= end {
		return false
	}
	c0 := p.src[start]
	c1 := byte(0)
	if start+1 < end {
		c1 = p.src[start+1]
	}
	isCompare := false
	if c0 == '=' && c1 == '=' {
		isCompare = true
	} else if c0 == '!' && c1 == '=' {
		isCompare = true
	} else if c0 == '<' && c1 != '<' {
		isCompare = true
	} else if c0 == '>' && c1 != '>' {
		isCompare = true
	}
	if !isCompare {
		return false
	}
	leftIndex := e.left
	rightIndex := e.right
	right := &ep.exprs[rightIndex]
	rightValue := 0
	rightOK := false
	if right.kind == rtgExprInt {
		rightValue = rtgParseIntToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprChar {
		rightValue = rtgParseCharToken(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprBool {
		rightValue = rtgBoolTokenValue(p, right.tok)
		rightOK = true
	} else if right.kind == rtgExprIdent {
		rightValue = rtgFindSmallConstByName(g, right.nameStart, right.nameEnd)
		rightOK = rightValue >= -128
	}
	if rightOK && rtgAsmImmFits8Signed(rightValue) {
		if !rtgEmitIntExpr(g, ep, leftIndex) {
			return false
		}
		rtgAsmCmpRaxImm8(&g.asm, rightValue)
		rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
		return true
	}
	if c0 == '=' || c0 == '!' {
		if right.kind == rtgExprString {
			return false
		}
		if right.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, right.nameStart, right.nameEnd)
			if localIndex >= 0 {
				if rtgTypeIsString(g.meta, g.locals[localIndex].typ) {
					return false
				}
			}
		}
	}
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(&g.asm)
	if !rtgEmitIntExpr(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(&g.asm)
	rtgAsmEmit16(&g.asm, 0xc139)
	if c0 == '<' {
		c0 = '>'
	} else if c0 == '>' {
		c0 = '<'
	}
	rtgEmitCompareJumpOp(&g.asm, c0, c1, label, jumpIfTrue)
	return true
}

func rtgEmitCompareJumpOp(a *rtgAsm, c0 byte, c1 byte, label int, jumpIfTrue bool) {
	op := 0
	if c0 == '=' {
		if jumpIfTrue {
			op = 33807
		} else {
			op = 34063
		}
	} else if c0 == '!' {
		if jumpIfTrue {
			op = 34063
		} else {
			op = 33807
		}
	} else if c0 == '<' {
		if c1 == '=' {
			if jumpIfTrue {
				op = 36367
			} else {
				op = 36623
			}
		} else {
			if jumpIfTrue {
				op = 35855
			} else {
				op = 36111
			}
		}
	} else if c1 == '=' {
		if jumpIfTrue {
			op = 36111
		} else {
			op = 35855
		}
	} else {
		if jumpIfTrue {
			op = 36623
		} else {
			op = 36367
		}
	}
	rtgAsmEmit16(a, op)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, label)
}

func rtgLinearInitGlobals(g *rtgLinearGen) bool {
	meta := g.meta
	a := &g.asm
	for i := 0; i < len(meta.globals); i++ {
		s := meta.globals[i]
		if s.kind != rtgTokVar {
			continue
		}
		off := g.asm.bssSize
		g.globals = append(g.globals, rtgGlobalInfo{nameStart: s.nameStart, nameEnd: s.nameEnd, offset: off})
		size := rtgTypeSize(meta, s.typ)
		if size < 8 {
			size = 8
		}
		g.asm.bssSize += rtgAlignTo8(size)
		if s.initStart < s.initEnd {
			ep := rtgParseExpression(g.prog, s.initStart, s.initEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return false
			}
			rootIndex := len(ep.exprs) - 1
			if rtgTypeIsString(meta, s.typ) {
				if !rtgEmitStringValueRegs(g, &ep, rootIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				continue
			}
			if rtgTypeIsSlice(meta, s.typ) {
				root := &ep.exprs[rootIndex]
				if root.kind != rtgExprComposite {
					return false
				}
				if !rtgEmitSliceLiteralRegs(g, &ep, rootIndex, s.typ) {
					return false
				}
				rtgAsmPushRcx(a)
				rtgAsmPushRdx(a)
				rtgAsmStoreRaxBss(a, off)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+8)
				rtgAsmPopRax(a)
				rtgAsmStoreRaxBss(a, off+16)
				continue
			}
			if rtgTypeIsStruct(meta, s.typ) {
				if !rtgEmitGlobalStructInit(g, &ep, rootIndex, s.typ, off) {
					return false
				}
				continue
			}
			resolved := rtgResolveType(meta, s.typ)
			root := &ep.exprs[rootIndex]
			if resolved.kind == rtgTypePointer && root.kind == rtgExprUnary && rtgTokCharIs(g.prog, root.tok, '&') {
				inner := &ep.exprs[root.left]
				if inner.kind != rtgExprIdent {
					return false
				}
				targetOff := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if targetOff < 0 {
					return false
				}
				rtgAsmMovRaxBssAddr(a, targetOff)
				rtgAsmStoreRaxBss(a, off)
				continue
			}
			constResult := rtgEvalConstExpr(g, &ep, rootIndex)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(a, constResult.value)
			rtgAsmNormalizeRaxForKind(a, resolved.kind)
			rtgAsmStoreRaxBss(a, off)
		} else if rtgTypeIsSlice(meta, s.typ) {
			rtgEmitInitEmptySliceBss(g, s.typ, off)
		}
	}
	return true
}

func rtgEmitGlobalStructInit(g *rtgLinearGen, ep *rtgExprParse, rootIndex int, typ int, off int) bool {
	root := &ep.exprs[rootIndex]
	if root.kind != rtgExprComposite {
		return false
	}
	for i := 0; i < root.argCount; i++ {
		field := ep.fields[root.firstArg+i]
		fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
		if fieldIndex < 0 {
			return false
		}
		fieldOffset := g.meta.fields[fieldIndex].offset
		fieldType := g.meta.fields[fieldIndex].typ
		if fieldType == 0 {
			return false
		}
		fieldResolved := rtgResolveType(g.meta, fieldType)
		if fieldResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushRdx(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
			rtgAsmPopRax(&g.asm)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset+8)
		} else if fieldResolved.kind == rtgTypeStruct || fieldResolved.kind == rtgTypeSlice {
			return false
		} else {
			constResult := rtgEvalConstExpr(g, ep, field.expr)
			if !constResult.ok {
				return false
			}
			rtgAsmMovRaxImm(&g.asm, constResult.value)
			rtgAsmNormalizeRaxForKind(&g.asm, fieldResolved.kind)
			rtgAsmStoreRaxBss(&g.asm, off+fieldOffset)
		}
	}
	return true
}

func rtgEmitInitEmptySliceBss(g *rtgLinearGen, sliceType int, off int) {
	a := &g.asm
	t := rtgResolveType(g.meta, sliceType)
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxBss(a, off)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxBss(a, off+8)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxBss(a, off+16)
}

func rtgEmitLinearAssign(g *rtgLinearGen, stmt *rtgStmt) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	nameStart := stmt.nameStart
	nameEnd := stmt.nameEnd
	if (stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar)) && rtgTokIsKind(p, stmt.startTok+1, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok+1].start
		nameEnd = p.toks[stmt.startTok+1].end
	} else if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) {
		nameStart = p.toks[stmt.startTok].start
		nameEnd = p.toks[stmt.startTok].end
	}
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok && rtgEmitMultiAssign(g, stmt, assignTok) {
		return true
	}
	if assignTok > stmt.startTok && rtgTokIsCompoundAssign(p, assignTok) {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice || !rtgTypeKindIsScalarInt(elemType.kind) {
					return false
				}
				elemSize := rtgScalarKindSize(elemType.kind)
				indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
					return false
				}
				rtgAsmStoreRaxStack(a, indexOffset)
				if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
					return false
				}
				rtgAsmStoreRaxStack(a, ptrOffset)
				rtgAsmLoadRaxStack(a, ptrOffset)
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRcxStack(a, indexOffset)
				rtgAsmLoadRaxIndexRcxSize(a, elemSize)
				rtgAsmPushRax(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizeRaxForKind(a, elemType.kind)
				rtgAsmLoadRdxStack(a, ptrOffset)
				rtgAsmLoadRcxStack(a, indexOffset)
				rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
				lhsResolved := rtgResolveType(meta, lhsType)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmLoadRaxMemRdxDispSize(a, 0, lhsSize)
				rtgAsmPushRax(a)
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRcx(a)
				if !rtgEmitRaxRcxOp(g, assignTok) {
					return false
				}
				rtgAsmNormalizeRaxForKind(a, lhsResolved.kind)
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmStoreRaxMemRdxDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			lhsRoot := &lhs.exprs[lhsIndex]
			lhsType := rtgInferParsedExprType(g, &lhs, lhsIndex)
			if lhsRoot.kind == rtgExprIndex {
				baseEnd := rtgFindTokenTextInRange(p, stmt.startTok, assignTok, '[')
				if baseEnd <= stmt.startTok {
					return false
				}
				baseEp := rtgParseExpression(p, stmt.startTok, baseEnd)
				if !baseEp.ok || len(baseEp.exprs) == 0 {
					return false
				}
				baseIndex := len(baseEp.exprs) - 1
				leftType := rtgInferParsedExprType(g, &baseEp, baseIndex)
				sliceType := rtgResolveType(meta, leftType)
				elemType := rtgResolveType(meta, sliceType.elem)
				if sliceType.kind != rtgTypeSlice {
					return false
				}
				scalarElem := rtgTypeKindIsScalarInt(elemType.kind)
				indexOffset := 0
				if scalarElem {
					indexOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
				}
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if scalarElem {
					if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmNormalizeRaxForKind(a, elemType.kind)
					rtgAsmPushRax(a)
					rtgAsmLoadRaxStack(a, indexOffset)
					rtgAsmPushRax(a)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmMovRdxRax(a)
					rtgAsmPopRcx(a)
					rtgAsmPopRax(a)
					elemSize := rtgScalarKindSize(elemType.kind)
					rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
					return true
				}
				if elemType.kind == rtgTypeString {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					if !rtgEmitIntExpr(g, &lhs, lhsRoot.right) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					if !rtgEmitStringValueRegs(g, &rhs, rhsIndex) {
						return false
					}
					rtgAsmPushStringRegs(a)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmMovRdxRax(a)
					rtgAsmLoadRcxStack(a, indexOffset)
					rtgAsmShlRcxImm(a, 4)
					rtgAsmAddRdxRcx(a)
					rtgAsmPopStoreStringMemRdx(a, 0)
					return true
				}
				if elemType.kind == rtgTypeStruct {
					indexOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					ptrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
					indexEnd := rtgFindMatchingExprClose(p, baseEnd+1, assignTok, '[', ']')
					if indexEnd <= baseEnd+1 {
						return false
					}
					indexEp := rtgParseExpression(p, baseEnd+1, indexEnd)
					if !indexEp.ok || len(indexEp.exprs) == 0 {
						return false
					}
					indexRoot := len(indexEp.exprs) - 1
					if !rtgEmitIntExpr(g, &indexEp, indexRoot) {
						return false
					}
					rtgAsmStoreRaxStack(a, indexOffset)
					if !rtgEmitSliceBasePtrLenTokens(g, p, stmt.startTok, baseEnd, &baseEp, baseIndex) {
						return false
					}
					rtgAsmStoreRaxStack(a, ptrOffset)
					rtgAsmLoadRaxStack(a, ptrOffset)
					rtgAsmMovRdxRax(a)
					rtgAsmLoadRcxStack(a, indexOffset)
					elemSize := rtgTypeSize(meta, sliceType.elem)
					rtgAsmImulRcxImm(a, elemSize)
					rtgAsmAddRdxRcx(a)
					rtgAsmStoreRdxStack(a, destOffset)
					if !rtgEmitCompositeFieldToMem(g, &rhs, rhsIndex, sliceType.elem, destOffset, 0) {
						return false
					}
					return true
				}
				return false
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsSlice(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				if rtgEmitAppendAssignGeneral(g, stmt, &rhs) {
					return true
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitSliceValueRegs(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPushSliceRegs(a)
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgAsmPopStoreSliceMemRdx(a, 0)
				return true
			}
			if lhsRoot.kind == rtgExprSelector && rtgTypeIsStruct(meta, lhsType) {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
				rtgAsmStoreRdxStack(a, addrOffset)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				size := rtgTypeSize(meta, lhsType)
				tempOffset := rtgAddTypedLocal(g, 0, 0, lhsType)
				if !rtgEmitTypedAssign(g, &rhs, rhsIndex, tempOffset) {
					return false
				}
				rtgAsmLoadRdxStack(a, addrOffset)
				rtgEmitCopyStackToMemRdx(g, tempOffset, 0, size)
				return true
			}
			if lhsRoot.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, &lhs, lhsIndex) {
					return false
				}
				rtgAsmPushRdx(a)
				rhs := rtgParseExpression(p, assignTok+1, stmt.endTok)
				if !rhs.ok || len(rhs.exprs) == 0 {
					return false
				}
				rhsIndex := len(rhs.exprs) - 1
				if !rtgEmitIntExpr(g, &rhs, rhsIndex) {
					return false
				}
				rtgAsmPopRdx(a)
				lhsResolved := rtgResolveType(meta, lhsType)
				rtgAsmNormalizeRaxForKind(a, lhsResolved.kind)
				lhsSize := rtgScalarKindSize(lhsResolved.kind)
				rtgAsmStoreRaxMemRdxDispSize(a, 0, lhsSize)
				return true
			}
		}
	}
	if nameEnd <= nameStart {
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokIsCompoundAssign(p, assignTok) {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmMovRdxRax(a)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, targetSize)
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmPopRcx(a)
			if !rtgEmitRaxRcxOp(g, assignTok) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, targetKind)
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDispSize(a, 0, targetSize)
			return true
		}
		if rtgTokCharIs(p, stmt.startTok, '*') && assignTok > stmt.startTok && rtgTokCharIs(p, assignTok, '=') {
			left := rtgParseExpression(p, stmt.startTok+1, assignTok)
			if !left.ok || len(left.exprs) == 0 {
				return false
			}
			leftIndex := len(left.exprs) - 1
			targetKind := rtgPointerTargetKind(g, &left, leftIndex)
			targetSize := rtgScalarKindSize(targetKind)
			if !rtgEmitIntExpr(g, &left, leftIndex) {
				return false
			}
			rtgAsmPushRax(a)
			right := rtgParseExpression(p, assignTok+1, stmt.endTok)
			if !right.ok || len(right.exprs) == 0 {
				return false
			}
			rightIndex := len(right.exprs) - 1
			if !rtgEmitIntExpr(g, &right, rightIndex) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, targetKind)
			rtgAsmPopRdx(a)
			rtgAsmStoreRaxMemRdxDispSize(a, 0, targetSize)
			return true
		}
		return false
	}
	if nameEnd == nameStart+1 && p.src[nameStart] == '_' {
		if assignTok <= stmt.startTok || !rtgTokCharIs(p, assignTok, '=') {
			return true
		}
		ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
		return ep.ok
	}
	offset := rtgFindLocalOffset(g, nameStart, nameEnd)
	if stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) || stmt.kind == rtgStmtShort {
		offset = -1
	}
	globalOffset := -1
	fieldStackOffset := -1
	if rtgTokIsKind(p, stmt.startTok, rtgTokIdent) && rtgTokCharIs(p, stmt.startTok+1, '.') && rtgTokIsKind(p, stmt.startTok+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[stmt.startTok].start, p.toks[stmt.startTok].end)
		if localIndex < 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[stmt.startTok+2].start, p.toks[stmt.startTok+2].end)
		if fieldOffset < 0 {
			return false
		}
		fieldStackOffset = g.locals[localIndex].offset - fieldOffset
		offset = fieldStackOffset
	}
	if offset < 0 {
		if stmt.kind == rtgStmtAssign && !rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
			globalOffset = rtgFindGlobalOffset(g, nameStart, nameEnd)
			if globalOffset < 0 {
				return false
			}
		} else {
			localType := rtgTypeInt
			if stmt.kind == rtgStmtVar || rtgTokIsKind(p, stmt.startTok, rtgTokVar) {
				typeEnd := assignTok
				if assignTok <= stmt.startTok {
					typeEnd = stmt.endTok
				}
				if stmt.startTok+2 < typeEnd {
					typeResult := rtgParseType(meta, g.prog, stmt.startTok+2, typeEnd)
					if typeResult.typ != 0 {
						localType = typeResult.typ
					}
				}
			}
			if stmt.kind == rtgStmtShort {
				inferredType := rtgInferExprType(g, assignTok+1, stmt.endTok)
				if assignTok+2 < stmt.endTok && rtgTokIsKind(p, assignTok+1, rtgTokIdent) && rtgTokCharIs(p, assignTok+2, '(') {
					fnIndex := -1
					for i := 0; i < len(g.meta.funcs); i++ {
						f := &g.meta.funcs[i]
						if rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, p.toks[assignTok+1].start, p.toks[assignTok+1].end) {
							fnIndex = i
						}
					}
					if fnIndex >= 0 {
						inferredType = meta.funcs[fnIndex].resultType
					}
				}
				if inferredType != 0 {
					localType = inferredType
				}
			}
			offset = rtgAddTypedLocal(g, nameStart, nameEnd, localType)
		}
	}
	if assignTok <= stmt.startTok {
		if globalOffset >= 0 {
			rtgAsmMovRaxImm(a, 0)
			rtgAsmStoreRaxBss(a, globalOffset)
		} else {
			rtgZeroLocalAtOffset(g, offset)
		}
		return true
	}
	ep := rtgParseExpression(p, assignTok+1, stmt.endTok)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	targetType := rtgTypeInt
	if globalOffset >= 0 {
		targetType = rtgFindGlobalType(g, nameStart, nameEnd)
	} else {
		targetType = rtgLocalTypeAtOffset(g, offset)
	}
	targetResolved := rtgResolveType(meta, targetType)
	if stmt.kind == rtgStmtShort {
		root := &ep.exprs[rootIndex]
		if root.kind == rtgExprCall && root.argCount == 2 && rtgExprIdentCode(p, &ep, root.left) == rtgIdentAppend {
			if !rtgEmitSliceValueRegs(g, &ep, ep.args[root.firstArg]) {
				return false
			}
			rtgAsmStoreSliceStack(a, offset)
		}
	}
	if rtgEmitAppendAssignGeneral(g, stmt, &ep) {
		return true
	}
	if rtgTokIsCompoundAssign(p, assignTok) {
		if globalOffset >= 0 {
			rtgAsmLoadRaxBss(a, globalOffset)
		} else {
			rtgAsmLoadRaxStack(a, offset)
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, &ep, rootIndex) {
			return false
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, assignTok) {
			return false
		}
		rtgAsmNormalizeRaxForKind(a, targetResolved.kind)
		if globalOffset >= 0 {
			rtgAsmStoreRaxBss(a, globalOffset)
		} else {
			rtgAsmStoreRaxStack(a, offset)
		}
		return true
	}
	if globalOffset < 0 && rtgEmitTypedAssign(g, &ep, rootIndex, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, &ep, rootIndex) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, targetResolved.kind)
	if globalOffset >= 0 {
		rtgAsmStoreRaxBss(a, globalOffset)
	} else {
		rtgAsmStoreRaxStack(a, offset)
	}
	return true
}

func rtgEmitMultiAssign(g *rtgLinearGen, stmt *rtgStmt, assignTok int) bool {
	p := g.prog
	var lhs []int
	var rhs []int
	lhs = rtgSplitTopLevelComma(p, stmt.startTok, assignTok, lhs)
	rhs = rtgSplitTopLevelComma(p, assignTok+1, stmt.endTok, rhs)
	lhsCount := len(lhs) / 2
	rhsCount := len(rhs) / 2
	if lhsCount <= 1 && rhsCount <= 1 {
		return false
	}
	if lhsCount > 1 && rhsCount == 1 {
		if rtgEmitTupleCallAssign(g, stmt.kind, lhs, lhsCount, rhs[0], rhs[1]) {
			return true
		}
	}
	if lhsCount != rhsCount {
		return false
	}
	var tempOffsets []int
	var tempTypes []int
	for i := 0; i < rhsCount; i++ {
		rhsStart := rhs[i*2]
		rhsEnd := rhs[i*2+1]
		ep := rtgParseExpression(p, rhsStart, rhsEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		typ := rtgInferParsedExprType(g, &ep, rootIndex)
		if typ == 0 {
			typ = rtgTypeInt
		}
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		if !rtgEmitExprToLocal(g, &ep, rootIndex, offset) {
			return false
		}
		tempOffsets = append(tempOffsets, offset)
		tempTypes = append(tempTypes, typ)
	}
	for i := 0; i < lhsCount; i++ {
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, stmt.kind, lhsStart, lhsEnd, tempOffsets[i], tempTypes[i]) {
			return false
		}
	}
	return true
}

func rtgEmitTupleCallAssign(g *rtgLinearGen, kind int, lhs []int, lhsCount int, rhsStart int, rhsEnd int) bool {
	p := g.prog
	ep := rtgParseExpression(p, rhsStart, rhsEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	if root.kind != rtgExprCall {
		return false
	}
	fnIndex := rtgFuncInfoFromCall(g, &ep, root.left)
	if fnIndex < 0 {
		return false
	}
	resultType := g.meta.funcs[fnIndex].resultType
	if !rtgTypeIsTuple(g.meta, resultType) {
		return false
	}
	tuple := rtgResolveType(g.meta, resultType)
	if tuple.count != lhsCount {
		return false
	}
	offset := rtgAddTypedLocal(g, 0, 0, resultType)
	if !rtgEmitStructCallToLocal(g, &ep, rootIndex, resultType, offset) {
		return false
	}
	for i := 0; i < lhsCount; i++ {
		field := g.meta.fields[tuple.first+i]
		lhsStart := lhs[i*2]
		lhsEnd := lhs[i*2+1]
		if !rtgEmitTempToTarget(g, kind, lhsStart, lhsEnd, offset-field.offset, field.typ) {
			return false
		}
	}
	return true
}

func rtgEmitExprToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	if rtgEmitTypedAssign(g, ep, idx, offset) {
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmStoreRaxStack(&g.asm, offset)
	return true
}

func rtgEmitTempToTarget(g *rtgLinearGen, kind int, targetStart int, targetEnd int, tempOffset int, tempType int) bool {
	p := g.prog
	ep := rtgParseExpression(p, targetStart, targetEnd)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	root := &ep.exprs[rootIndex]
	size := rtgTypeSize(g.meta, tempType)
	if size < 8 {
		size = 8
	}
	if root.kind == rtgExprIdent {
		if root.nameEnd == root.nameStart+1 && p.src[root.nameStart] == '_' {
			return true
		}
		localIndex := rtgFindLocalIndex(g, root.nameStart, root.nameEnd)
		if kind == rtgStmtShort {
			localIndex = rtgFindLocalIndexInCurrentScope(g, root.nameStart, root.nameEnd)
			if localIndex < 0 {
				offset := rtgAddTypedLocal(g, root.nameStart, root.nameEnd, tempType)
				rtgEmitCopyStackToStack(g, tempOffset, offset, size)
				return true
			}
		}
		if localIndex >= 0 {
			rtgEmitCopyStackToStack(g, tempOffset, g.locals[localIndex].offset, size)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, root.nameStart, root.nameEnd)
		if globalOffset < 0 {
			return false
		}
		rtgEmitCopyStackToBss(g, tempOffset, globalOffset, size)
		return true
	}
	if kind == rtgStmtShort {
		return false
	}
	if root.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, &ep, rootIndex) {
			return false
		}
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, &ep, rootIndex) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	if root.kind == rtgExprUnary && rtgTokCharIs(p, root.tok, '*') {
		if !rtgEmitIntExpr(g, &ep, root.left) {
			return false
		}
		rtgAsmMovRdxRax(&g.asm)
		targetType := rtgInferParsedExprType(g, &ep, rootIndex)
		targetSize := rtgTypeSize(g.meta, targetType)
		if targetSize < 8 {
			targetSize = 8
		}
		rtgEmitCopyStackToMemRdx(g, tempOffset, 0, targetSize)
		return true
	}
	return false
}

func rtgEmitCopyStackToBss(g *rtgLinearGen, srcOffset int, bssOffset int, size int) {
	if size < 8 {
		size = 8
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(&g.asm, srcOffset-at)
		rtgAsmStoreRaxBss(&g.asm, bssOffset+at)
	}
}

func rtgFindLocalIndexInCurrentScope(g *rtgLinearGen, nameStart int, nameEnd int) int {
	start := g.scopeBase
	if start < 0 {
		start = 0
	}
	for i := len(g.locals) - 1; i >= start; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgSplitTopLevelComma(p *rtgProgram, start int, end int, ranges []int) []int {
	partStart := start
	paren := 0
	brack := 0
	brace := 0
	i := start
	for i < end {
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren > 0 {
				paren--
			}
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			if brack > 0 {
				brack--
			}
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			if brace > 0 {
				brace--
			}
		} else if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			ranges = append(ranges, partStart)
			ranges = append(ranges, i)
			partStart = i + 1
		}
		i++
	}
	if partStart < end {
		ranges = append(ranges, partStart)
		ranges = append(ranges, end)
	}
	return ranges
}

func rtgEmitTupleReturn(g *rtgLinearGen, start int, end int) bool {
	resultType := g.meta.funcs[g.currentFunc].resultType
	tuple := rtgResolveType(g.meta, resultType)
	var parts []int
	parts = rtgSplitTopLevelComma(g.prog, start, end, parts)
	count := len(parts) / 2
	if count == tuple.count {
		for i := 0; i < count; i++ {
			partStart := parts[i*2]
			partEnd := parts[i*2+1]
			field := g.meta.fields[tuple.first+i]
			if !rtgEmitTupleReturnField(g, partStart, partEnd, field.typ, field.offset) {
				return false
			}
		}
		return true
	}
	if count == 1 {
		ep := rtgParseExpression(g.prog, start, end)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		return rtgEmitStructReturnExpr(g, &ep, rootIndex)
	}
	return false
}

func rtgEmitTupleReturnField(g *rtgLinearGen, start int, end int, typ int, fieldOffset int) bool {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return false
	}
	rootIndex := len(ep.exprs) - 1
	tempOffset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitExprToLocal(g, &ep, rootIndex, tempOffset) {
		return false
	}
	size := rtgTypeSize(g.meta, typ)
	if size < 8 {
		size = 8
	}
	rtgAsmLoadRdxStack(&g.asm, g.returnStruct)
	rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
	return true
}

func rtgInferExprType(g *rtgLinearGen, start int, end int) int {
	ep := rtgParseExpression(g.prog, start, end)
	if !ep.ok || len(ep.exprs) == 0 {
		return 0
	}
	rootIndex := len(ep.exprs) - 1
	return rtgInferParsedExprType(g, &ep, rootIndex)
}

func rtgInferParsedExprType(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	meta := g.meta
	e := ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return rtgTypeInt
	}
	if e.kind == rtgExprFloat {
		return rtgTypeFloat64
	}
	if e.kind == rtgExprString {
		return rtgTypeString
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			return g.locals[localIndex].typ
		}
		for i := 0; i < len(meta.globals); i++ {
			s := &meta.globals[i]
			if rtgBytesEqualRange(p.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return s.typ
			}
		}
		constStringTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constStringTok >= 0 {
			return rtgTypeString
		}
		return rtgTypeInt
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if callee == rtgIdentAppend && e.argCount == 2 {
			return rtgInferParsedExprType(g, ep, ep.args[e.firstArg])
		}
		if callee == rtgIdentByteSlice && e.argCount == 1 {
			return rtgAddType(meta, rtgTypeSlice, rtgTypeByte, 0, 0, 24, 0, 0)
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
			}
		}
		if callee == rtgIdentInt {
			return rtgTypeInt
		}
		if callee == rtgIdentInt16 {
			return rtgTypeInt16
		}
		if callee == rtgIdentInt32 {
			return rtgTypeInt32
		}
		if callee == rtgIdentInt64 {
			return rtgTypeInt64
		}
		if callee == rtgIdentByte {
			return rtgTypeByte
		}
		if callee == rtgIdentLen || callee == rtgIdentOpen || callee == rtgIdentClose || callee == rtgIdentRead || callee == rtgIdentWrite || callee == rtgIdentChmod || callee == rtgIdentCopy {
			return rtgTypeInt
		}
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex >= 0 {
			return meta.funcs[fnIndex].resultType
		}
		if e.argCount == 1 {
			calleeExpr := &ep.exprs[e.left]
			if calleeExpr.kind == rtgExprIdent {
				namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
				if namedType > 0 {
					return namedType
				}
			}
		}
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, leftType)
		if t.kind == rtgTypeSlice {
			return t.elem
		}
		if t.kind == rtgTypeString {
			return rtgTypeByte
		}
	}
	if e.kind == rtgExprSlice {
		return rtgInferParsedExprType(g, ep, e.left)
	}
	if e.kind == rtgExprSelector {
		baseType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			t = rtgResolveType(meta, t.elem)
		}
		if t.kind == rtgTypeStruct {
			for i := 0; i < t.count; i++ {
				field := &meta.fields[t.first+i]
				if rtgBytesEqualRange(p.src, field.nameStart, field.nameEnd, e.nameStart, e.nameEnd) {
					return field.typ
				}
			}
		}
	}
	if e.kind == rtgExprComposite {
		return rtgTypeFromExpr(g, ep, idx)
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgInferParsedExprType(g, ep, e.left)
		}
		if rtgTokCharIs(p, e.tok, '&') {
			elemType := rtgInferParsedExprType(g, ep, e.left)
			if elemType == 0 {
				return 0
			}
			return rtgAddType(meta, rtgTypePointer, elemType, 0, 0, 8, 0, 0)
		}
		if rtgTokCharIs(p, e.tok, '*') {
			innerType := rtgInferParsedExprType(g, ep, e.left)
			inner := rtgResolveType(meta, innerType)
			if inner.kind == rtgTypePointer {
				return inner.elem
			}
		}
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') {
			return rtgTypeInt
		}
		leftType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.left))
		rightType := rtgResolveType(meta, rtgInferParsedExprType(g, ep, e.right))
		if leftType.kind == rtgTypeFloat64 || rightType.kind == rtgTypeFloat64 {
			return rtgTypeFloat64
		}
	}
	return rtgTypeInt
}

func rtgPointerTargetKind(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	pointerType := rtgTypeInt
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			pointerType = g.locals[localIndex].typ
		} else {
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalType != 0 {
				pointerType = globalType
			}
		}
	} else {
		pointerType = rtgInferParsedExprType(g, ep, idx)
	}
	pointerResolved := rtgResolveType(g.meta, pointerType)
	if pointerResolved.kind == rtgTypePointer {
		targetResolved := rtgResolveType(g.meta, pointerResolved.elem)
		return targetResolved.kind
	}
	return pointerResolved.kind
}

func rtgTypeFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	e := &ep.exprs[idx]
	if e.tok < 0 || e.tok >= len(p.toks) {
		return 0
	}
	endTok := e.tok
	for endTok < len(p.toks) && p.toks[endTok].end <= e.nameEnd {
		endTok++
	}
	typeResult := rtgParseType(g.meta, p, e.tok, endTok)
	return typeResult.typ
}

func rtgFindTypeByText(g *rtgLinearGen, name string) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualText(g.prog.src, t.nameStart, t.nameEnd, name) {
			return i
		}
	}
	return 0
}

func rtgFindTypeByRange(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.types); i++ {
		t := &g.meta.types[i]
		if t.nameEnd > t.nameStart && rtgBytesEqualRange(g.prog.src, t.nameStart, t.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return 0
}

func rtgLocalTypeAtOffset(g *rtgLinearGen, offset int) int {
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			return g.locals[i].typ
		}
	}
	for i := 0; i < len(g.locals); i++ {
		t := rtgResolveType(g.meta, g.locals[i].typ)
		if t.kind == rtgTypeStruct {
			for j := 0; j < t.count; j++ {
				field := g.meta.fields[t.first+j]
				if g.locals[i].offset-field.offset == offset {
					return field.typ
				}
			}
		}
	}
	return 0
}

func rtgEmitTypedAssign(g *rtgLinearGen, ep *rtgExprParse, idx int, offset int) bool {
	meta := g.meta
	destType := rtgLocalTypeAtOffset(g, offset)
	e := &ep.exprs[idx]
	destResolved := rtgResolveType(meta, destType)
	if destResolved.kind == rtgTypeStruct {
		if e.kind == rtgExprIdent {
			localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
			if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != rtgTypeSize(meta, destType) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyStackToStack(g, g.locals[localIndex].offset, offset, size)
			return true
		}
		if e.kind == rtgExprCall {
			return rtgEmitStructCallToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprIndex {
			return rtgEmitIndexedStructToLocal(g, ep, idx, destType, offset)
		}
		if e.kind == rtgExprSelector {
			fieldType := rtgInferParsedExprType(g, ep, idx)
			if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != rtgTypeSize(meta, destType) {
				return false
			}
			if !rtgEmitSelectorAddressRdx(g, ep, idx) {
				return false
			}
			size := rtgTypeSize(meta, destType)
			rtgEmitCopyMemRdxToStack(g, offset, size)
			return true
		}
		if e.kind == rtgExprComposite {
			rtgZeroLocalAtOffset(g, offset)
			for i := 0; i < e.argCount; i++ {
				field := ep.fields[e.firstArg+i]
				fieldIndex := rtgCompositeStructFieldIndex(g, destType, &field, i)
				if fieldIndex < 0 {
					return false
				}
				fieldOffset := g.meta.fields[fieldIndex].offset
				fieldType := g.meta.fields[fieldIndex].typ
				if fieldType == 0 {
					return false
				}
				if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
					return false
				}
			}
			return true
		}
		return false
	}
	if destResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(&g.asm, offset)
		rtgAsmStoreRdxStack(&g.asm, offset-8)
		return true
	}
	if rtgTypeKindIsScalarInt(destResolved.kind) {
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		rtgAsmNormalizeRaxForKind(&g.asm, destResolved.kind)
		rtgAsmStoreRaxStack(&g.asm, offset)
		return true
	}
	if !rtgTypeIsSlice(meta, destType) {
		return false
	}
	if !rtgEmitSliceValueRegs(g, ep, idx) {
		return false
	}
	rtgAsmStoreSliceStack(&g.asm, offset)
	return true
}

func rtgEmitSliceValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, e.left) {
			return false
		}
		if e.firstArg >= 0 {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			baseResolved := rtgResolveType(meta, baseType)
			if baseResolved.kind != rtgTypeSlice {
				return false
			}
			elemSize := rtgTypeSize(meta, baseResolved.elem)
			if elemSize < 1 {
				elemSize = 8
			}
			baseOff := rtgAddTypedLocal(g, 0, 0, baseType)
			lowOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			highOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmStoreSliceStack(a, baseOff)
			if !rtgEmitIntExpr(g, ep, e.firstArg) {
				return false
			}
			rtgAsmStoreRaxStack(a, lowOff)
			if e.right >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmStoreRaxStack(a, highOff)
			} else {
				rtgAsmLoadRaxStack(a, baseOff-8)
				rtgAsmStoreRaxStack(a, highOff)
			}
			rtgAsmLoadRaxStack(a, baseOff-16)
			rtgAsmLoadRcxStack(a, lowOff)
			rtgAsmSubRaxRcx(a)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, highOff)
			rtgAsmLoadRcxStack(a, lowOff)
			rtgAsmSubRaxRcx(a)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, baseOff)
			rtgAsmLoadRcxStack(a, lowOff)
			if elemSize != 1 {
				rtgAsmImulRcxImm(a, elemSize)
			}
			rtgAsmAddRaxRcx(a)
			rtgAsmPopRdx(a)
			rtgAsmPopRcx(a)
			return true
		}
		if e.right >= 0 {
			rtgAsmPushRax(a)
			rtgAsmPushRcx(a)
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmMovRdxRax(a)
			rtgAsmPopRcx(a)
			rtgAsmPopRax(a)
		}
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(meta, globalType) {
				return false
			}
			rtgAsmLoadRaxBss(a, globalOffset+16)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPopRdx(a)
			rtgAsmPopRcx(a)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-16)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 16)
		rtgAsmMovRcxRax(a)
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		return rtgEmitSliceLiteralRegs(g, ep, idx, sliceType)
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if e.argCount == 2 && callee == rtgIdentAppend {
			var stmt rtgStmt
			if !rtgEmitAppendAssignGeneral(g, &stmt, ep) {
				return false
			}
			return rtgEmitSliceValueRegs(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 2 || e.argCount == 3 {
			if callee == rtgIdentMake {
				return rtgEmitMakeSliceRegs(g, ep, idx)
			}
		}
		if e.argCount == 1 && callee == rtgIdentByteSlice {
			return rtgEmitByteSliceConversionRegs(g, ep, idx)
		}
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, callType) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitSliceLiteralRegs(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := e.argCount * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	if !rtgEmitSliceLiteralBacking(g, ep, idx, sliceType, backingOff) {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, e.argCount)
	rtgAsmPopRcx(a)
	return true
}

func rtgEmitSliceLiteralBacking(g *rtgLinearGen, ep *rtgExprParse, idx int, sliceType int, backingOff int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemType := t.elem
	elemResolved := rtgResolveType(g.meta, elemType)
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	for i := 0; i < e.argCount; i++ {
		field := ep.fields[e.firstArg+i]
		if field.nameEnd > field.nameStart {
			return false
		}
		disp := i * elemSize
		if elemResolved.kind == rtgTypeString {
			if !rtgEmitStringValueRegs(g, ep, field.expr) {
				return false
			}
			rtgAsmPushStringRegs(a)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopStoreStringMemRdx(a, disp)
			continue
		}
		if elemResolved.kind == rtgTypeStruct {
			addrOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			if disp != 0 {
				rtgAsmAddRdxImm(a, disp)
			}
			rtgAsmStoreRdxStack(a, addrOffset)
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, elemType, addrOffset, 0) {
				return false
			}
			continue
		}
		if !rtgTypeKindIsScalarInt(elemResolved.kind) {
			return false
		}
		if !rtgEmitIntExpr(g, ep, field.expr) {
			return false
		}
		rtgAsmNormalizeRaxForKind(a, elemResolved.kind)
		rtgAsmPushRax(a)
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDispSize(a, disp, elemSize)
	}
	return true
}

func rtgEmitMakeSliceRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 && e.argCount != 3 {
		return false
	}
	sliceType := rtgTypeFromExpr(g, ep, ep.args[e.firstArg])
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	lenOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	capOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
		return false
	}
	rtgAsmStoreRaxStack(a, lenOffset)
	if e.argCount == 3 {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+2]) {
			return false
		}
		rtgAsmStoreRaxStack(a, capOffset)
	} else {
		rtgAsmLoadRaxStack(a, lenOffset)
		rtgAsmStoreRaxStack(a, capOffset)
	}
	backingSize := 32768
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOffset)
	rtgAsmLoadRcxStack(a, capOffset)
	return true
}

func rtgEmitByteSliceConversionRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	srcOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	lenOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	idxOff := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	backingOff := g.asm.bssSize
	backingSize := 32768
	g.asm.bssSize += backingSize
	argIndex := ep.args[e.firstArg]
	if !rtgEmitStringValueRegs(g, ep, argIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcOff)
	rtgAsmStoreRdxStack(a, lenOff)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, idxOff)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, lenOff)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcOff)
	rtgAsmPopRcx(a)
	rtgAsmLoadByteRaxIndexRcx(a)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRcx(a)
	rtgAsmPopRax(a)
	rtgAsmStoreAlMemRdxRcx1(a)
	rtgAsmLoadRaxStack(a, idxOff)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, idxOff)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmLoadRdxStack(a, lenOff)
	rtgAsmMovRcxRdx(a)
	return true
}

func rtgEmitStringValueRegs(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(g.prog, e.tok)
		msgOff := rtgAddStringData(g, msg)
		msgLen := len(msg)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmMovRdxImm(a, msgLen)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			rtgAsmPushRax(a)
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(g.prog, constTok)
			msgOff := rtgAddStringData(g, msg)
			msgLen := len(msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
			rtgAsmMovRdxImm(a, msgLen)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
		rtgAsmMovRdxRax(a)
		rtgAsmEmit24(a, 0x08048b)
		rtgAsmAddRdxRcx(a)
		rtgAsmMemDisp(a, 8, 0x8b48, 0x52, 0x92)
		return true
	}
	if e.kind == rtgExprSelector {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, valueType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPushRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		if !rtgEmitUserCall(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitCompositeFieldToStack(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, destOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreSliceStack(a, destOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmStoreRaxStack(a, destOffset)
		rtgAsmStoreRdxStack(a, destOffset-8)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgEmitCopyStackToStack(g, tempOffset, destOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, fieldResolved.kind)
	rtgAsmStoreRaxStack(a, destOffset)
	return true
}

func rtgEmitCompositeFieldToMem(g *rtgLinearGen, ep *rtgExprParse, idx int, fieldType int, addrOffset int, fieldOffset int) bool {
	a := &g.asm
	fieldResolved := rtgResolveType(g.meta, fieldType)
	if fieldResolved.kind == rtgTypeSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushSliceRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreSliceMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeString {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmPushStringRegs(a)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgAsmPopStoreStringMemRdx(a, fieldOffset)
		return true
	}
	if fieldResolved.kind == rtgTypeStruct {
		tempOffset := rtgAddTypedLocal(g, 0, 0, fieldType)
		if !rtgEmitTypedAssign(g, ep, idx, tempOffset) {
			return false
		}
		size := rtgTypeSize(g.meta, fieldType)
		rtgAsmLoadRdxStack(a, addrOffset)
		rtgEmitCopyStackToMemRdx(g, tempOffset, fieldOffset, size)
		return true
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, fieldResolved.kind)
	rtgAsmLoadRdxStack(a, addrOffset)
	fieldSize := rtgScalarKindSize(fieldResolved.kind)
	rtgAsmStoreRaxMemRdxDispSize(a, fieldOffset, fieldSize)
	return true
}

func rtgEmitCopyStackToStack(g *rtgLinearGen, srcOffset int, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}

func rtgEmitCopyStackToMemRdx(g *rtgLinearGen, srcOffset int, destDisp int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		if destDisp != 0 {
			rtgAsmAddRdxImm(a, destDisp)
		}
		rtgAsmEmit16(a, 0x5f52)
		rtgAsmLeaRsiStack(a, srcOffset)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxStack(a, srcOffset-at)
		rtgAsmStoreRaxMemRdxDisp(a, destDisp+at)
	}
}

func rtgEmitCopyMemRdxToStack(g *rtgLinearGen, destOffset int, size int) {
	a := &g.asm
	if size > 16 {
		label := rtgEnsureCopyWordsHelper(g)
		rtgAsmLeaRdiStack(a, destOffset)
		rtgAsmEmit16(a, 0x5e52)
		rtgAsmMovRdxImm(a, size/8)
		rtgAsmCallLabel(a, label)
		return
	}
	for at := 0; at < size; at += 8 {
		rtgAsmLoadRaxMemRdxDisp(a, at)
		rtgAsmStoreRaxStack(a, destOffset-at)
	}
}

func rtgEmitPushStackWords(g *rtgLinearGen, offset int, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxStack(&g.asm, offset-at)
		rtgAsmPushRax(&g.asm)
	}
}

func rtgEmitPushMemRdxWords(g *rtgLinearGen, size int) {
	for at := size - 8; at >= 0; at -= 8 {
		rtgAsmLoadRaxMemRdxDisp(&g.asm, at)
		rtgAsmPushRax(&g.asm)
	}
}

func rtgEmitIndexedStructToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	leftType := rtgInferParsedExprType(g, ep, e.left)
	sliceType := rtgResolveType(meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(meta, sliceType.elem)
	destResolved := rtgResolveType(meta, destType)
	if elemType.kind != rtgTypeStruct || destResolved.kind != rtgTypeStruct {
		return false
	}
	elemSize := rtgTypeSize(meta, sliceType.elem)
	if rtgTypeSize(meta, destType) != elemSize {
		return false
	}
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, e.left) {
		return false
	}
	rtgAsmPopRcx(a)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmMovRdxRax(a)
	rtgAsmAddRdxRcx(a)
	rtgEmitCopyMemRdxToStack(g, offset, elemSize)
	return true
}

func rtgEmitStructCallToLocal(g *rtgLinearGen, ep *rtgExprParse, idx int, destType int, offset int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 || !rtgTypeIsStruct(g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	if rtgTypeSize(g.meta, destType) != rtgTypeSize(g.meta, g.meta.funcs[fnIndex].resultType) {
		return false
	}
	wordCount := 1
	for i := e.argCount - 1; i >= 0; i-- {
		words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
		if words < 0 {
			return false
		}
		wordCount += words
	}
	rtgAsmStackMem(&g.asm, offset, 0x8d48, 0x45, 0x85)
	rtgAsmPushRax(&g.asm)
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func rtgEmitStructReturnExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if g.returnStruct <= 0 {
		return false
	}
	e := &ep.exprs[idx]
	resultType := g.meta.funcs[g.currentFunc].resultType
	size := rtgTypeSize(meta, resultType)
	rtgAsmLoadRdxStack(a, g.returnStruct)
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return false
		}
		rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, size)
		return true
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgAsmLoadRcxStack(a, g.returnStruct)
		for at := 0; at < size; at += 8 {
			rtgAsmLoadRaxMemRdxDisp(a, at)
			if at == 0 {
				rtgAsmEmit16(a, 0x0189)
			} else {
				rtgAsmMemDisp(a, at, 0x8948, 0x41, 0x81)
			}
		}
		return true
	}
	if e.kind == rtgExprComposite {
		rtgAsmMovRaxImm(a, 0)
		for at := 0; at < size; at += 8 {
			rtgAsmStoreRaxMemRdxDisp(a, at)
		}
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, resultType, &field, i)
			if fieldIndex < 0 {
				return false
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if fieldType == 0 {
				return false
			}
			if !rtgEmitCompositeFieldToMem(g, ep, field.expr, fieldType, g.returnStruct, fieldOffset) {
				return false
			}
		}
		return true
	}
	if e.kind == rtgExprCall {
		fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
		if fnIndex < 0 || !rtgTypeIsStruct(meta, meta.funcs[fnIndex].resultType) {
			return false
		}
		if rtgTypeSize(meta, meta.funcs[fnIndex].resultType) != size {
			return false
		}
		wordCount := 1
		for i := e.argCount - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[e.firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
		rtgAsmLoadRaxStack(a, g.returnStruct)
		rtgAsmPushRax(a)
		rtgEmitCallWithWordCount(g, fnIndex, wordCount)
		return true
	}
	return false
}

func rtgEmitUserCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	fnIndex := rtgFuncInfoFromCall(g, ep, e.left)
	if fnIndex < 0 {
		return rtgEmitNamedConversionCall(g, ep, idx)
	}
	if fnIndex >= len(g.funcLabels) {
		return false
	}
	firstArg := e.firstArg
	argCount := e.argCount
	expanded := e.nameStart
	wordCount := 0
	fn := &g.meta.funcs[fnIndex]
	receiverIndex := -1
	receiverDotTok := 0
	if fn.receiverType != 0 {
		callee := &ep.exprs[e.left]
		if callee.kind != rtgExprSelector {
			return false
		}
		receiverIndex = callee.left
		receiverDotTok = callee.tok
	}
	if expanded == 0 && fn.paramCount > 0 && g.meta.params[fn.firstParam+fn.paramCount-1].initStart == 1 {
		fixed := fn.paramCount - 1
		if receiverIndex >= 0 {
			fixed--
		}
		if argCount < fixed {
			return false
		}
		if receiverIndex >= 0 {
			if !rtgEmitVariadicArgSliceReverse(g, ep, firstArg+fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		} else {
			if !rtgEmitVariadicArgSliceFromCallReverse(g, ep, idx, fixed, argCount-fixed, g.meta.params[fn.firstParam+fn.paramCount-1].typ) {
				return false
			}
		}
		wordCount = 3
		for i := fixed - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
	} else {
		for i := argCount - 1; i >= 0; i-- {
			words := rtgEmitCallArgReverse(g, ep, ep.args[firstArg+i])
			if words < 0 {
				return false
			}
			wordCount += words
		}
	}
	if receiverIndex >= 0 {
		words := rtgEmitMethodReceiverArgReverse(g, ep, receiverIndex, g.meta.params[fn.firstParam].typ)
		if words < 0 {
			words = rtgEmitMethodReceiverArgTokensReverse(g, receiverDotTok, g.meta.params[fn.firstParam].typ)
			if words < 0 {
				return false
			}
		}
		wordCount += words
	}
	rtgEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func rtgEmitMethodReceiverArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, receiverType int) int {
	meta := g.meta
	a := &g.asm
	receiver := rtgResolveType(meta, receiverType)
	exprType := rtgInferParsedExprType(g, ep, idx)
	exprResolved := rtgResolveType(meta, exprType)
	if receiver.kind == rtgTypePointer {
		if exprResolved.kind == rtgTypePointer {
			if !rtgEmitIntExpr(g, ep, idx) {
				return -1
			}
			rtgAsmPushRax(a)
			return 1
		}
		if !rtgEmitAddressRax(g, ep, idx) {
			return -1
		}
		rtgAsmPushRax(a)
		return 1
	}
	if receiver.kind == rtgTypeStruct && exprResolved.kind == rtgTypePointer {
		if !rtgEmitIntExpr(g, ep, idx) {
			return -1
		}
		rtgAsmMovRdxRax(a)
		size := rtgTypeSize(meta, receiverType)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	return rtgEmitCallArgReverse(g, ep, idx)
}

func rtgEmitMethodReceiverArgTokensReverse(g *rtgLinearGen, dotTok int, receiverType int) int {
	if dotTok <= 0 {
		return -1
	}
	start := dotTok - 1
	if !rtgTokIsKind(g.prog, start, rtgTokIdent) {
		return -1
	}
	receiverEp := rtgParseExpression(g.prog, start, dotTok)
	if !receiverEp.ok || len(receiverEp.exprs) == 0 {
		return -1
	}
	return rtgEmitMethodReceiverArgReverse(g, &receiverEp, len(receiverEp.exprs)-1, receiverType)
}

func rtgEmitAddressRax(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			rtgAsmLeaRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 {
			rtgAsmMovRaxBssAddr(a, globalOffset)
			return true
		}
	}
	if e.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmMovRaxRdx(a)
		return true
	}
	if e.kind == rtgExprIndex {
		if !rtgEmitIndexAddressRax(g, ep, idx) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitNamedConversionCall(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	e := &ep.exprs[idx]
	if e.argCount != 1 {
		return false
	}
	calleeExpr := &ep.exprs[e.left]
	if calleeExpr.kind != rtgExprIdent {
		return false
	}
	namedType := rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd)
	resolved := rtgResolveType(g.meta, namedType)
	if resolved.kind == rtgTypeString {
		return rtgEmitStringValueRegs(g, ep, ep.args[e.firstArg])
	}
	resolvedKind := resolved.kind
	if rtgTypeKindIsScalarInt(resolvedKind) {
		if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
			return false
		}
		rtgAsmNormalizeRaxForKind(&g.asm, resolvedKind)
		return true
	}
	return false
}

func rtgEmitVariadicArgSliceReverse(g *rtgLinearGen, ep *rtgExprParse, first int, count int, sliceType int) bool {
	fieldFirst := len(ep.fields)
	for i := 0; i < count; i++ {
		var field rtgCompositeField
		field.expr = ep.args[first+i]
		ep.fields = append(ep.fields, field)
	}
	idx := len(ep.exprs)
	var expr rtgExpr
	expr.kind = rtgExprComposite
	expr.firstArg = fieldFirst
	expr.argCount = count
	ep.exprs = append(ep.exprs, expr)
	if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
		return false
	}
	rtgAsmPushSliceRegs(&g.asm)
	return true
}

func rtgEmitVariadicArgSliceFromCallReverse(g *rtgLinearGen, ep *rtgExprParse, callIdx int, skip int, count int, sliceType int) bool {
	a := &g.asm
	call := &ep.exprs[callIdx]
	t := rtgResolveType(g.meta, sliceType)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	if !rtgTypeKindIsScalarInt(elem.kind) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, t.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	backingSize := 32768
	needSize := count * elemSize
	if needSize > backingSize {
		backingSize = rtgAlignTo8(needSize)
	}
	if backingSize < elemSize {
		backingSize = elemSize
	}
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	closeTok := rtgFindMatchingExprClose(g.prog, call.tok+1, ep.end, '(', ')')
	if closeTok <= call.tok {
		return false
	}
	pos := call.tok + 1
	argIndex := 0
	emitted := 0
	for pos < closeTok && emitted < count {
		argEnd := rtgFindExprBoundary(g.prog, pos, closeTok)
		if rtgTokCharIs(g.prog, argEnd, '{') {
			compositeEnd := rtgSkipBalanced(g.prog, argEnd, '{', '}')
			if compositeEnd > argEnd {
				argEnd = compositeEnd
			}
		}
		if argIndex >= skip {
			argEp := rtgParseExpression(g.prog, pos, argEnd)
			if !argEp.ok || len(argEp.exprs) == 0 {
				return false
			}
			rootIndex := len(argEp.exprs) - 1
			if !rtgEmitIntExpr(g, &argEp, rootIndex) {
				return false
			}
			rtgAsmNormalizeRaxForKind(a, elem.kind)
			disp := emitted * elemSize
			rtgAsmPushRax(a)
			rtgAsmMovRaxBssAddr(a, backingOff)
			rtgAsmMovRdxRax(a)
			rtgAsmPopRax(a)
			rtgAsmStoreRaxMemRdxDispSize(a, disp, elemSize)
			emitted++
		}
		pos = argEnd
		if rtgTokCharIs(g.prog, pos, ',') {
			pos++
		}
		argIndex++
	}
	if emitted != count {
		return false
	}
	capacity := backingSize / elemSize
	rtgAsmMovRaxImm(a, capacity)
	rtgAsmPushRax(a)
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmMovRdxImm(a, count)
	rtgAsmPopRcx(a)
	rtgAsmPushSliceRegs(a)
	return true
}

func rtgEmitCallWithWordCount(g *rtgLinearGen, fnIndex int, wordCount int) {
	a := &g.asm
	if wordCount > 0 {
		rtgAsmPopRdi(a)
	}
	if wordCount > 1 {
		rtgAsmEmit8(a, 0x5e)
	}
	if wordCount > 2 {
		rtgAsmPopRdx(a)
	}
	if wordCount > 3 {
		rtgAsmPopRcx(a)
	}
	if wordCount > 4 {
		rtgAsmPopRax(a)
	}
	if wordCount > 5 {
		rtgAsmEmit8(a, 0x5f)
	}
	rtgAsmCallLabel(a, g.funcLabels[fnIndex])
	if wordCount > 6 {
		imm := (wordCount - 6) * 4
		if rtgAsmImmFits8Signed(imm) {
			rtgAsmEmit3(a, 0x83, 0xc4, imm)
		} else {
			rtgAsmEmit16(a, 0xc481)
			rtgAsmEmit32(a, imm)
		}
	}
}

func rtgEmitCallArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	p := g.prog
	a := &g.asm
	typ := rtgInferParsedExprType(g, ep, idx)
	if rtgTypeIsSlice(g.meta, typ) {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushSliceRegs(&g.asm)
		return 3
	}
	if rtgTypeIsString(g.meta, typ) {
		if !rtgEmitStringValueRegs(g, ep, idx) {
			return -1
		}
		rtgAsmPushStringRegs(&g.asm)
		return 2
	}
	if rtgTypeIsTuple(g.meta, typ) {
		return rtgEmitTupleArgReverse(g, ep, idx, typ)
	}
	if rtgTypeIsStruct(g.meta, typ) {
		return rtgEmitStructArgReverse(g, ep, idx, typ)
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt {
		value := rtgParseIntToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmPushImm(a, value)
		return 1
	}
	if e.kind == rtgExprIdent {
		constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
		if constResult.ok {
			rtgAsmPushImm(a, constResult.value)
			return 1
		}
	}
	if !rtgEmitIntExpr(g, ep, idx) {
		return -1
	}
	rtgAsmPushRax(a)
	return 1
}

func rtgEmitTupleArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	e := &ep.exprs[idx]
	if e.kind != rtgExprCall {
		return -1
	}
	offset := rtgAddTypedLocal(g, 0, 0, typ)
	if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
		return -1
	}
	tuple := rtgResolveType(g.meta, typ)
	wordCount := 0
	for i := tuple.count - 1; i >= 0; i-- {
		field := g.meta.fields[tuple.first+i]
		size := rtgTypeSize(g.meta, field.typ)
		if size < 8 {
			size = 8
		}
		rtgEmitPushStackWords(g, offset-field.offset, size)
		wordCount += size / 8
	}
	return wordCount
}

func rtgEmitStructArgReverse(g *rtgLinearGen, ep *rtgExprParse, idx int, typ int) int {
	meta := g.meta
	a := &g.asm
	size := rtgTypeSize(meta, typ)
	if size <= 0 {
		return -1
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 || rtgTypeSize(meta, g.locals[localIndex].typ) != size {
			return -1
		}
		rtgEmitPushStackWords(g, g.locals[localIndex].offset, size)
		return size / 8
	}
	if e.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, e.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct || rtgTypeSize(meta, sliceType.elem) != size {
			return -1
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return -1
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, e.left) {
			return -1
		}
		rtgAsmPopRcx(a)
		rtgAsmImulRcxImm(a, size)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsStruct(meta, fieldType) || rtgTypeSize(meta, fieldType) != size {
			return -1
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return -1
		}
		rtgEmitPushMemRdxWords(g, size)
		return size / 8
	}
	if e.kind == rtgExprComposite {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		rtgZeroLocalAtOffset(g, offset)
		for i := 0; i < e.argCount; i++ {
			field := ep.fields[e.firstArg+i]
			fieldIndex := rtgCompositeStructFieldIndex(g, typ, &field, i)
			if fieldIndex < 0 {
				return -1
			}
			fieldOffset := g.meta.fields[fieldIndex].offset
			fieldType := g.meta.fields[fieldIndex].typ
			if !rtgEmitCompositeFieldToStack(g, ep, field.expr, fieldType, offset-fieldOffset) {
				return -1
			}
		}
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	if e.kind == rtgExprCall {
		offset := rtgAddTypedLocal(g, 0, 0, typ)
		if !rtgEmitStructCallToLocal(g, ep, idx, typ, offset) {
			return -1
		}
		rtgEmitPushStackWords(g, offset, size)
		return size / 8
	}
	return -1
}

func rtgEmitIntExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	if (e.kind == rtgExprUnary || e.kind == rtgExprBinary || e.kind == rtgExprCall) && rtgExprCanFoldConst(g, ep, idx) {
		constResult := rtgEvalConstExpr(g, ep, idx)
		if constResult.ok {
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
	}
	if e.kind == rtgExprInt {
		rtgAsmMovRaxIntToken(a, p, e.tok)
		return true
	}
	if e.kind == rtgExprFloat {
		value := rtgParseFloatTokenScaled(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			constResult := rtgEvalConstByName(g, e.nameStart, e.nameEnd)
			if !constResult.ok {
				globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
				if globalOffset < 0 {
					return false
				}
				rtgAsmLoadRaxBss(a, globalOffset)
				return true
			}
			rtgAsmMovRaxImm(a, constResult.value)
			return true
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprChar {
		value := rtgParseCharToken(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprBool {
		value := rtgBoolTokenValue(p, e.tok)
		rtgAsmMovRaxImm(a, value)
		return true
	}
	if e.kind == rtgExprCall {
		callee := rtgExprIdentCode(p, ep, e.left)
		if e.argCount == 1 && (callee == rtgIdentInt || callee == rtgIdentInt64) {
			return rtgEmitIntExpr(g, ep, ep.args[e.firstArg])
		}
		if e.argCount == 1 && (callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32) {
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			if callee == rtgIdentByte {
				rtgAsmNormalizeRaxForKind(a, rtgTypeByte)
			} else if callee == rtgIdentInt16 {
				rtgAsmNormalizeRaxForKind(a, rtgTypeInt16)
			} else {
				rtgAsmNormalizeRaxForKind(a, rtgTypeInt32)
			}
			return true
		}
		if e.argCount == 1 && callee == rtgIdentLen {
			arg := &ep.exprs[ep.args[e.firstArg]]
			if arg.kind == rtgExprString {
				msg := rtgDecodeStringToken(p, arg.tok)
				msgLen := len(msg)
				rtgAsmMovRaxImm(a, msgLen)
				return true
			}
			if arg.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
				if localIndex >= 0 && (rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) || rtgTypeIsString(g.meta, g.locals[localIndex].typ)) {
					rtgAsmLoadRaxStack(a, g.locals[localIndex].offset-8)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, arg.nameStart, arg.nameEnd)
				globalType := rtgFindGlobalType(g, arg.nameStart, arg.nameEnd)
				if globalOffset >= 0 && (rtgTypeIsString(g.meta, globalType) || rtgTypeIsSlice(g.meta, globalType)) {
					rtgAsmLoadRaxBss(a, globalOffset+8)
					return true
				}
				constTok := rtgFindConstStringToken(g, arg.nameStart, arg.nameEnd)
				if constTok >= 0 {
					msg := rtgDecodeStringToken(p, constTok)
					msgLen := len(msg)
					rtgAsmMovRaxImm(a, msgLen)
					return true
				}
			}
			if arg.kind == rtgExprSelector {
				if !rtgEmitSlicePtrLen(g, ep, ep.args[e.firstArg]) {
					return false
				}
				rtgAsmEmit16(a, 0x5851)
				return true
			}
			if arg.kind == rtgExprUnary && rtgTokCharIs(p, arg.tok, '*') {
				if !rtgEmitIntExpr(g, ep, arg.left) {
					return false
				}
				rtgAsmMovRdxRax(a)
				rtgAsmLoadRaxMemRdxDisp(a, 8)
				return true
			}
		}
		if callee == rtgIdentOpen {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitStringPtrExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmPopRcx(a)
			rtgAsmMovRdxImm(a, 493)
			rtgAsmMovRaxImm(a, rtgLinuxSysOpen)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentClose {
			if e.argCount != 1 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmMovRdiRax(a)
			rtgAsmMovRaxImm(a, rtgLinuxSysClose)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentChmod {
			if e.argCount != 2 {
				return false
			}
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg]) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitIntExpr(g, ep, ep.args[e.firstArg+1]) {
				return false
			}
			rtgAsmMovRsiRax(a)
			rtgAsmPopRdi(a)
			rtgAsmMovRaxImm(a, rtgLinuxSysFchmod)
			rtgAsmSyscall(a)
			return true
		}
		if callee == rtgIdentRead {
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysReadSeq, rtgLinuxSysReadAt)
		}
		if callee == rtgIdentWrite {
			return rtgEmitBuiltinReadWrite(g, ep, idx, rtgLinuxSysWriteSeq, rtgLinuxSysWriteAt)
		}
		if callee == rtgIdentCopy {
			return rtgEmitBuiltinCopy(g, ep, idx)
		}
		return rtgEmitUserCall(g, ep, idx)
	}
	if e.kind == rtgExprIndex {
		return rtgEmitIndexExpr(g, ep, idx)
	}
	if e.kind == rtgExprSelector {
		base := &ep.exprs[e.left]
		if base.kind == rtgExprCall {
			baseType := rtgInferParsedExprType(g, ep, e.left)
			if !rtgTypeIsStruct(g.meta, baseType) {
				return false
			}
			fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
			if fieldOffset < 0 {
				return false
			}
			fieldType := rtgStructFieldType(g, baseType, e.nameStart, e.nameEnd)
			offset := rtgAddTypedLocal(g, 0, 0, baseType)
			if !rtgEmitStructCallToLocal(g, ep, e.left, baseType, offset) {
				return false
			}
			rtgAsmStackMem(a, offset-fieldOffset, 0x8d48, 0x55, 0x95)
			fieldResolved := rtgResolveType(g.meta, fieldType)
			fieldSize := rtgScalarKindSize(fieldResolved.kind)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, fieldSize)
			return true
		}
		if base.kind == rtgExprIndex {
			return rtgEmitIndexedStructField(g, ep, e.left, e.nameStart, e.nameEnd)
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		fieldType := rtgInferParsedExprType(g, ep, idx)
		fieldResolved := rtgResolveType(g.meta, fieldType)
		fieldSize := rtgScalarKindSize(fieldResolved.kind)
		rtgAsmLoadRaxMemRdxDispSize(a, 0, fieldSize)
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '&') {
			inner := &ep.exprs[e.left]
			if inner.kind == rtgExprIdent {
				localIndex := rtgFindLocalIndex(g, inner.nameStart, inner.nameEnd)
				if localIndex >= 0 {
					rtgAsmStackMem(a, g.locals[localIndex].offset, 0x8d48, 0x45, 0x85)
					return true
				}
				globalOffset := rtgFindGlobalOffset(g, inner.nameStart, inner.nameEnd)
				if globalOffset >= 0 {
					rtgAsmMovRaxBssAddr(a, globalOffset)
					return true
				}
				return false
			}
			if inner.kind == rtgExprSelector {
				if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
					return false
				}
				rtgAsmEmit16(a, 0x5852)
				return true
			}
			if inner.kind == rtgExprIndex {
				return rtgEmitIndexAddressRax(g, ep, e.left)
			}
			return false
		}
		if rtgTokCharIs(p, e.tok, '*') {
			if !rtgEmitIntExpr(g, ep, e.left) {
				return false
			}
			rtgAsmMovRdxRax(a)
			targetKind := rtgPointerTargetKind(g, ep, e.left)
			size := rtgScalarKindSize(targetKind)
			rtgAsmLoadRaxMemRdxDispSize(a, 0, size)
			return true
		}
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		if rtgTokCharIs(p, e.tok, '-') {
			rtgAsmEmit16(a, 0xd8f7)
			return true
		}
		if rtgTokCharIs(p, e.tok, '+') {
			return true
		}
		if rtgTokCharIs(p, e.tok, '!') {
			rtgAsmBoolNotRax(a)
			return true
		}
		return false
	}
	if e.kind == rtgExprBinary {
		if rtgBinaryUsesFloat(g, ep, e) {
			return rtgEmitFloatBinaryExpr(g, ep, idx)
		}
		if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			falseLabel := rtgAsmNewLabel(a)
			endLabel := rtgAsmNewLabel(a)
			if !rtgEmitJumpIfFalse(g, ep, idx, falseLabel) {
				return false
			}
			rtgAsmMovRaxImm(a, 1)
			rtgAsmJmpLabel(a, endLabel)
			rtgAsmMarkLabel(a, falseLabel)
			rtgAsmMovRaxImm(a, 0)
			rtgAsmMarkLabel(a, endLabel)
			return true
		}
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') {
			leftType := rtgInferParsedExprType(g, ep, e.left)
			rightType := rtgInferParsedExprType(g, ep, e.right)
			if rtgTypeIsString(g.meta, leftType) || rtgTypeIsString(g.meta, rightType) {
				notEqual := rtgTok2Is(p, e.tok, '!', '=')
				return rtgEmitStringCompare(g, ep, e.left, e.right, notEqual)
			}
		}
		rightExpr := &ep.exprs[e.right]
		rightKind := rightExpr.kind
		rightTok := rightExpr.tok
		if !rtgEmitIntExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPushRax(a)
		if rightKind == rtgExprInt {
			rtgAsmMovRaxIntToken(a, p, rightTok)
		} else if rightKind == rtgExprChar {
			value := rtgParseCharToken(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else if rightKind == rtgExprBool {
			value := rtgBoolTokenValue(p, rightTok)
			rtgAsmMovRaxImm(a, value)
		} else {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
		}
		rtgAsmPopRcx(a)
		if !rtgEmitRaxRcxOp(g, e.tok) {
			return false
		}
		return true
	}
	return false
}

func rtgEmitAppendAssignGeneral(g *rtgLinearGen, stmt *rtgStmt, ep *rtgExprParse) bool {
	p := g.prog
	if len(ep.exprs) == 0 {
		return false
	}
	root := &ep.exprs[len(ep.exprs)-1]
	if root.kind != rtgExprCall || root.argCount != 2 || rtgExprIdentCode(p, ep, root.left) != rtgIdentAppend {
		return false
	}
	var loc rtgSliceLocation
	locEp := ep
	assignTok := rtgFindAssignmentToken(p, stmt.startTok, stmt.endTok)
	if assignTok > stmt.startTok {
		lhs := rtgParseExpression(p, stmt.startTok, assignTok)
		if lhs.ok && len(lhs.exprs) > 0 {
			lhsIndex := len(lhs.exprs) - 1
			rtgSetSliceLocationFromExpr(g, &lhs, lhsIndex, &loc)
			locEp = &lhs
		}
	}
	if !loc.ok {
		rtgSetSliceLocationFromExpr(g, ep, ep.args[root.firstArg], &loc)
		locEp = ep
	}
	if !loc.ok {
		return false
	}
	t := rtgResolveType(g.meta, loc.typ)
	if t.kind != rtgTypeSlice {
		return false
	}
	elem := rtgResolveType(g.meta, t.elem)
	valueIndex := ep.args[root.firstArg+1]
	if root.nameStart == 1 {
		return rtgEmitAppendExpansionToLocation(g, ep, locEp, &loc, t.elem, valueIndex)
	}
	if elem.kind == rtgTypeStruct {
		value := &ep.exprs[valueIndex]
		if value.kind != rtgExprComposite {
			if value.kind == rtgExprUnary && rtgTokCharIs(p, value.tok, '*') {
				return rtgEmitAppendStructDeref(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			if value.kind == rtgExprIdent {
				typeTok := value.tok
				if !rtgTokCharIs(p, typeTok+1, '{') {
					typeTok = 0
					for i := 0; i < len(p.toks); i++ {
						if p.toks[i].start == value.nameStart {
							typeTok = i
						}
					}
				}
				if rtgTokCharIs(p, typeTok+1, '{') {
					return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
				}
				return rtgEmitAppendStructLocal(g, ep, locEp, &loc, t.elem, valueIndex)
			}
			typeTok := rtgFindAppendCompositeTypeToken(p, root.tok, stmt.endTok)
			if typeTok >= 0 {
				return rtgEmitAppendStructCompositeTokens(g, locEp, &loc, t.elem, typeTok)
			}
			return false
		}
		if !rtgEmitAppendStructComposite(g, ep, locEp, &loc, t.elem, valueIndex) {
			return false
		}
		return true
	}
	if rtgTypeKindIsScalarInt(elem.kind) {
		if !rtgEmitAppendScalarToLocation(g, ep, locEp, &loc, elem.kind, valueIndex) {
			return false
		}
		return true
	}
	if elem.kind == rtgTypeString {
		if !rtgEmitAppendStringToLocation(g, ep, locEp, &loc, valueIndex) {
			return false
		}
		return true
	}
	return false
}

func rtgBinaryUsesFloat(g *rtgLinearGen, ep *rtgExprParse, e *rtgExpr) bool {
	p := g.prog
	if rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
		return false
	}
	left := rtgInferParsedExprType(g, ep, e.left)
	if left == rtgTypeFloat64 {
		return true
	}
	right := rtgInferParsedExprType(g, ep, e.right)
	if right == rtgTypeFloat64 {
		return true
	}
	if !ep.hasFloat {
		return false
	}
	if rtgExprValueIsFloat(g, ep, e.left) {
		return true
	}
	return rtgExprValueIsFloat(g, ep, e.right)
}

func rtgExprValueIsFloat(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	e := &ep.exprs[idx]
	if e.kind == rtgExprFloat {
		return true
	}
	if e.kind == rtgExprUnary {
		if rtgTokCharIs(p, e.tok, '+') || rtgTokCharIs(p, e.tok, '-') {
			return rtgExprValueIsFloat(g, ep, e.left)
		}
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	if e.kind == rtgExprBinary {
		if rtgTok2Is(p, e.tok, '=', '=') || rtgTok2Is(p, e.tok, '!', '=') || rtgTokCharIs(p, e.tok, '<') || rtgTokCharIs(p, e.tok, '>') || rtgTok2Is(p, e.tok, '&', '&') || rtgTok2Is(p, e.tok, '|', '|') {
			return false
		}
		if rtgExprValueIsFloat(g, ep, e.left) {
			return true
		}
		return rtgExprValueIsFloat(g, ep, e.right)
	}
	if e.kind == rtgExprIdent || e.kind == rtgExprCall || e.kind == rtgExprIndex || e.kind == rtgExprSelector {
		typ := rtgResolveType(g.meta, rtgInferParsedExprType(g, ep, idx))
		return typ.kind == rtgTypeFloat64
	}
	return false
}

func rtgEmitFloatBinaryExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
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
		rtgAsmEmit24(a, 0xc1af0f)
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
	if !rtgEmitIntExpr(g, ep, e.left) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitIntExpr(g, ep, e.right) {
		return false
	}
	rtgAsmPopRcx(a)
	return rtgEmitRaxRcxOp(g, e.tok)
}

func rtgExprCanFoldConst(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	if idx < 0 || idx >= len(ep.exprs) {
		return false
	}
	e := &ep.exprs[idx]
	if e.kind == rtgExprInt || e.kind == rtgExprChar || e.kind == rtgExprBool {
		return true
	}
	if e.kind == rtgExprIdent {
		if rtgFindLocalIndex(g, e.nameStart, e.nameEnd) >= 0 {
			return false
		}
		builtin := rtgEvalBuiltinConst(g, e.nameStart, e.nameEnd)
		if builtin.ok {
			return true
		}
		for i := 0; i < len(g.meta.globals); i++ {
			s := &g.meta.globals[i]
			if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, e.nameStart, e.nameEnd) {
				return true
			}
		}
		return false
	}
	if e.kind == rtgExprUnary {
		return rtgExprCanFoldConst(g, ep, e.left)
	}
	if e.kind == rtgExprBinary {
		return rtgExprCanFoldConst(g, ep, e.left) && rtgExprCanFoldConst(g, ep, e.right)
	}
	if e.kind == rtgExprCall {
		if e.argCount != 1 || !rtgExprCanFoldConst(g, ep, ep.args[e.firstArg]) {
			return false
		}
		callee := rtgExprIdentCode(g.prog, ep, e.left)
		if callee == rtgIdentInt || callee == rtgIdentByte || callee == rtgIdentInt16 || callee == rtgIdentInt32 || callee == rtgIdentInt64 {
			return true
		}
		calleeExpr := &ep.exprs[e.left]
		if calleeExpr.kind == rtgExprIdent {
			return rtgFindTypeByRange(g, calleeExpr.nameStart, calleeExpr.nameEnd) > 0
		}
	}
	return false
}

func rtgEmitAppendExpansionToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	elemSize := rtgTypeSize(g.meta, elemType)
	if elemSize < 1 {
		elemSize = 8
	}
	sourceType := rtgInferParsedExprType(g, ep, valueIndex)
	source := rtgResolveType(g.meta, sourceType)
	if source.kind != rtgTypeSlice {
		return false
	}
	if rtgTypeSize(g.meta, source.elem) != elemSize {
		return false
	}
	if elemSize == 1 {
		if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmPushRdx(a)
		if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmPopRdx(a)
		rtgAsmPopRax(a)
		label := rtgEnsureAppendBytesHelper(g)
		rtgAsmCallLabel(a, label)
		return true
	}
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcIndex := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	headerOffset := 0
	if !rtgEmitSliceValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, srcIndex)
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		headerOffset = rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
		rtgAsmStoreRdxStack(a, headerOffset)
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmLoadRaxMemRdxDisp(a, 8)
		rtgAsmStoreRaxStack(a, destLen)
	} else if loc.global {
		rtgAsmLoadRaxBss(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxBss(a, loc.offset+8)
		rtgAsmStoreRaxStack(a, destLen)
	} else {
		rtgAsmLoadRaxStack(a, loc.offset)
		rtgAsmStoreRaxStack(a, destPtr)
		rtgAsmLoadRaxStack(a, loc.offset-8)
		rtgAsmStoreRaxStack(a, destLen)
	}
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, srcIndex, destPtr, destLen)
	rtgAsmLoadRaxStack(a, srcIndex)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, srcIndex)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, destLen)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, destLen)
	if loc.mem {
		rtgAsmLoadRdxStack(a, headerOffset)
		rtgAsmStoreRaxMemRdxDisp(a, 8)
	} else if loc.global {
		rtgAsmStoreRaxBss(a, loc.offset+8)
	} else {
		rtgAsmStoreRaxStack(a, loc.offset-8)
	}
	return true
}

func rtgEmitAppendExpansionCopyElement(g *rtgLinearGen, elemSize int, srcPtr int, srcIndex int, destPtr int, destLen int) {
	a := &g.asm
	if elemSize == 1 || elemSize == 2 || elemSize == 4 || elemSize == 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmLoadRaxIndexRcxSize(a, elemSize)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxRcxSize(a, elemSize)
		return
	}
	for copyOff := 0; copyOff < elemSize; copyOff += 8 {
		rtgAsmLoadRaxStack(a, srcPtr)
		rtgAsmLoadRcxStack(a, srcIndex)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmLoadQwordRaxIndexRcxDisp(a, copyOff)
		rtgAsmPushRax(a)
		rtgAsmLoadRdxStack(a, destPtr)
		rtgAsmLoadRcxStack(a, destLen)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmAddRdxRcx(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDisp(a, copyOff)
	}
}

func rtgFindAppendCompositeTypeToken(p *rtgProgram, openTok int, end int) int {
	if openTok < 0 || openTok >= end || !rtgTokCharIs(p, openTok, '(') {
		return -1
	}
	i := openTok + 1
	paren := 0
	brack := 0
	brace := 0
	for i < end {
		if paren == 0 && brack == 0 && brace == 0 && rtgTokCharIs(p, i, ',') {
			typeTok := i + 1
			if rtgTokIsKind(p, typeTok, rtgTokIdent) && rtgTokCharIs(p, typeTok+1, '{') {
				return typeTok
			}
			return -1
		}
		if rtgTokCharIs(p, i, '(') {
			paren++
		} else if rtgTokCharIs(p, i, ')') {
			if paren == 0 {
				return -1
			}
			paren--
		} else if rtgTokCharIs(p, i, '[') {
			brack++
		} else if rtgTokCharIs(p, i, ']') {
			brack--
		} else if rtgTokCharIs(p, i, '{') {
			brace++
		} else if rtgTokCharIs(p, i, '}') {
			brace--
		}
		i++
	}
	return -1
}

func rtgEmitAppendScalarToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemKind int, valueIndex int) bool {
	a := &g.asm
	elemSize := rtgScalarKindSize(elemKind)
	if !rtgEmitIntExpr(g, ep, valueIndex) {
		return false
	}
	rtgAsmNormalizeRaxForKind(a, elemKind)
	rtgAsmPushRax(a)
	if elemSize == 2 || elemSize == 4 {
		if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmPopRax(a)
		rtgAsmStoreRaxMemRdxDispSize(a, 0, elemSize)
		return true
	}
	label := rtgEnsureAppendScalarHelper(g, elemKind)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmPopRdx(a)
	rtgAsmCallLabel(a, label)
	return true
}

func rtgEmitSliceSlotAddrs(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	a := &g.asm
	if loc.mem {
		if loc.expr < 0 || loc.expr >= len(locEp.exprs) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, locEp, loc.expr) {
			return false
		}
		rtgEmitEnsureMemSlice(g, elemSize)
		rtgAsmEmit16(a, 0x5f52)
		rtgAsmEmit16(a, 0x728d)
		rtgAsmEmit8(a, 8)
		return true
	}
	if loc.global {
		rtgAsmEmit16(a, 0x3d8d)
		at := len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset, rtgAbsBssReloc)
		rtgAsmEmit16(a, 0x358d)
		at = len(a.code)
		rtgAsmEmit32(a, 0)
		rtgAsmAddAbsReloc(a, at, loc.offset+8, rtgAbsBssReloc)
		return true
	}
	rtgAsmLeaRdiStack(a, loc.offset)
	rtgAsmLeaRsiStack(a, loc.offset-8)
	return true
}

func rtgEmitAppendDestRax(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemSize int) bool {
	label := rtgEnsureAppendAddrHelper(g)
	if !rtgEmitSliceSlotAddrs(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxImm(&g.asm, elemSize)
	rtgAsmCallLabel(&g.asm, label)
	return true
}

func rtgEnsureAppendScalarHelper(g *rtgLinearGen, elemKind int) int {
	if rtgScalarKindSize(elemKind) == 1 {
		return rtgEnsureAppend8Helper(g)
	}
	return rtgEnsureAppend64Helper(g)
}

func rtgEnsureAppendAddrHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendAddrEmitted {
		return g.appendAddrLabel
	}
	g.appendAddrEmitted = true
	g.appendAddrLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendAddrLabel)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xcaaf0f)
	rtgAsmAddRaxRcx(a)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmIncRcx(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendAddrLabel
}

func rtgEnsureAppend8Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append8Emitted {
		return g.append8Label
	}
	g.append8Emitted = true
	g.append8Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append8Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0x081488)
	rtgAsmIncRcx(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append8Label
}

func rtgEnsureAppend64Helper(g *rtgLinearGen) int {
	a := &g.asm
	if g.append64Emitted {
		return g.append64Label
	}
	g.append64Emitted = true
	g.append64Label = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.append64Label)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x078b)
	rtgAsmEmit24(a, 0xc81489)
	rtgAsmIncRcx(a)
	rtgAsmEmit16(a, 0x0e89)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.append64Label
}

func rtgEnsureAppendBytesHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.appendBytesEmitted {
		return g.appendBytesLabel
	}
	g.appendBytesEmitted = true
	g.appendBytesLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.appendBytesLabel)
	rtgAsmEmit16(a, 0x0e8b)
	rtgAsmEmit16(a, 0x3f8b)
	rtgAsmEmit16(a, 0xcf01)
	rtgAsmEmit16(a, 0x1601)
	rtgAsmEmit16(a, 0xc689)
	rtgAsmEmit16(a, 0xd189)
	rtgAsmEmit16(a, 0xa4f3)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.appendBytesLabel
}

func rtgEnsureCopyWordsHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.copyWordsEmitted {
		return g.copyWordsLabel
	}
	g.copyWordsEmitted = true
	g.copyWordsLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.copyWordsLabel)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit16(a, 0xd285)
	rtgAsmJzLabel(a, doneLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit16(a, 0x068b)
	rtgAsmEmit16(a, 0x0789)
	rtgAsmEmit24(a, 0x04468b)
	rtgAsmEmit24(a, 0x044789)
	rtgAsmEmit3(a, 0x83, 0xc6, 8)
	rtgAsmEmit3(a, 0x83, 0xc7, 8)
	rtgAsmEmit8(a, 0x4a)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.copyWordsLabel
}

func rtgEmitAppendStringToLocation(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, valueIndex int) bool {
	a := &g.asm
	rtgEnsureAppendAddrHelper(g)
	if !rtgEmitStringValueRegs(g, ep, valueIndex) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitAppendDestRax(g, locEp, loc, 16) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgAsmPopStoreStringMemRdx(a, 0)
	return true
}

func rtgSetSliceLocationFromExpr(g *rtgLinearGen, ep *rtgExprParse, idx int, loc *rtgSliceLocation) {
	e := &ep.exprs[idx]
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || !rtgTypeIsSlice(g.meta, globalType) {
				return
			}
			loc.offset = globalOffset
			loc.typ = globalType
			loc.global = true
			loc.ok = true
			return
		}
		if !rtgTypeIsSlice(g.meta, g.locals[localIndex].typ) {
			return
		}
		loc.offset = g.locals[localIndex].offset
		loc.typ = g.locals[localIndex].typ
		loc.ok = true
		return
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(g.meta, fieldType) {
			return
		}
		loc.expr = idx
		loc.typ = fieldType
		loc.mem = true
		loc.ok = true
		return
	}
}

func rtgEmitEnsureMemSlice(g *rtgLinearGen, elemSize int) {
	a := &g.asm
	if elemSize < 1 {
		elemSize = 8
	}
	okLabel := rtgAsmNewLabel(a)
	rtgAsmLoadRaxMemRdxDisp(a, 0)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, okLabel)
	backingSize := 2097152
	backingOff := g.asm.bssSize
	g.asm.bssSize += backingSize
	rtgAsmMovRaxBssAddr(a, backingOff)
	rtgAsmStoreRaxMemRdxDisp(a, 0)
	rtgAsmMovRaxImm(a, backingSize/elemSize)
	rtgAsmStoreRaxMemRdxDisp(a, 16)
	rtgAsmMarkLabel(a, okLabel)
}

func rtgEmitAppendStructCompositeTokens(g *rtgLinearGen, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, typeTok int) bool {
	p := g.prog
	openTok := typeTok + 1
	closeTok := rtgSkipBalanced(p, openTok, '{', '}')
	if closeTok <= openTok {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	destOffset := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	rtgAsmStoreRaxStack(&g.asm, destOffset)
	i := openTok + 1
	for i < closeTok-1 {
		if !rtgTokIsKind(p, i, rtgTokIdent) || !rtgTokCharIs(p, i+1, ':') {
			return false
		}
		fieldTok := p.toks[i]
		exprStart := i + 2
		exprEnd := rtgFindExprBoundary(p, exprStart, closeTok-1)
		ep := rtgParseExpression(p, exprStart, exprEnd)
		if !ep.ok || len(ep.exprs) == 0 {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, elemType, fieldTok.start, fieldTok.end)
		if fieldOffset < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, elemType, fieldTok.start, fieldTok.end)
		if fieldType == 0 {
			return false
		}
		rootIndex := len(ep.exprs) - 1
		if !rtgEmitCompositeFieldToMem(g, &ep, rootIndex, fieldType, destOffset, fieldOffset) {
			return false
		}
		i = exprEnd
		if rtgTokCharIs(p, i, ',') {
			i++
		}
	}
	return true
}

func rtgEmitAppendStructDeref(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	a := &g.asm
	value := &ep.exprs[valueIndex]
	valueType := rtgInferParsedExprType(g, ep, valueIndex)
	if !rtgTypeIsStruct(g.meta, valueType) || rtgTypeSize(g.meta, valueType) != rtgTypeSize(g.meta, elemType) {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitIntExpr(g, ep, value.left) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyMemRdxToStack(g, tempOffset, elemSize)
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(a)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
	return true
}

func rtgEmitAppendStructLocal(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	value := &ep.exprs[valueIndex]
	localIndex := rtgFindLocalIndex(g, value.nameStart, value.nameEnd)
	if localIndex < 0 {
		return false
	}
	elemSize := rtgTypeSize(g.meta, elemType)
	if rtgTypeSize(g.meta, g.locals[localIndex].typ) != elemSize {
		return false
	}
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, g.locals[localIndex].offset, 0, elemSize)
	return true
}

func rtgEmitAppendStructComposite(g *rtgLinearGen, ep *rtgExprParse, locEp *rtgExprParse, loc *rtgSliceLocation, elemType int, valueIndex int) bool {
	elemSize := rtgTypeSize(g.meta, elemType)
	tempOffset := rtgAddTypedLocal(g, 0, 0, elemType)
	if !rtgEmitTypedAssign(g, ep, valueIndex, tempOffset) {
		return false
	}
	if !rtgEmitAppendDestRax(g, locEp, loc, elemSize) {
		return false
	}
	rtgAsmMovRdxRax(&g.asm)
	rtgEmitCopyStackToMemRdx(g, tempOffset, 0, elemSize)
	return true
}

func rtgEmitStringCompare(g *rtgLinearGen, ep *rtgExprParse, left int, right int, notEqual bool) bool {
	a := &g.asm
	label := rtgEnsureStringEqualHelper(g)
	if !rtgEmitStringValueRegs(g, ep, left) {
		return false
	}
	rtgAsmPushStringRegs(a)
	if !rtgEmitStringValueRegs(g, ep, right) {
		return false
	}
	rtgAsmMovRcxRdx(a)
	rtgAsmMovRdxRax(a)
	rtgAsmPopRdi(a)
	rtgAsmEmit8(a, 0x5e)
	rtgAsmCallLabel(a, label)
	if notEqual {
		rtgAsmBoolNotRax(a)
	}
	return true
}

func rtgEnsureStringEqualHelper(g *rtgLinearGen) int {
	a := &g.asm
	if g.streqEmitted {
		return g.streqLabel
	}
	g.streqEmitted = true
	g.streqLabel = rtgAsmNewLabel(a)
	afterLabel := rtgAsmNewLabel(a)
	rtgAsmJmpLabel(a, afterLabel)
	rtgAsmMarkLabel(a, g.streqLabel)
	notEqualLabel := rtgAsmNewLabel(a)
	equalLabel := rtgAsmNewLabel(a)
	loopLabel := rtgAsmNewLabel(a)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmEmit16(a, 0xce39)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit16(a, 0xf685)
	rtgAsmJzLabel(a, equalLabel)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit16(a, 0x0b8a)
	rtgAsmEmit16(a, 0x0a38)
	rtgAsmJnzLabel(a, notEqualLabel)
	rtgAsmEmit8(a, 0x43)
	rtgAsmEmit8(a, 0x42)
	rtgAsmEmit8(a, 0x4e)
	rtgAsmJnzLabel(a, loopLabel)
	rtgAsmMarkLabel(a, equalLabel)
	rtgAsmMovRaxImm(a, 1)
	rtgAsmMarkLabel(a, notEqualLabel)
	rtgAsmRet(a)
	rtgAsmMarkLabel(a, afterLabel)
	return g.streqLabel
}

func rtgEmitBuiltinCopy(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.argCount != 2 {
		return false
	}
	destIndex := ep.args[e.firstArg]
	srcIndex := ep.args[e.firstArg+1]
	destType := rtgInferParsedExprType(g, ep, destIndex)
	srcType := rtgInferParsedExprType(g, ep, srcIndex)
	destSlice := rtgResolveType(meta, destType)
	srcSlice := rtgResolveType(meta, srcType)
	if destSlice.kind != rtgTypeSlice || srcSlice.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(meta, destSlice.elem)
	if elemSize != rtgTypeSize(meta, srcSlice.elem) {
		return false
	}
	if elemSize < 1 {
		elemSize = 8
	}
	destPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	destLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcPtr := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	srcLen := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	count := rtgAddTypedLocal(g, 0, 0, rtgTypeInt)
	if !rtgEmitSliceValueRegs(g, ep, destIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, destPtr)
	rtgAsmStoreRdxStack(a, destLen)
	if !rtgEmitSliceValueRegs(g, ep, srcIndex) {
		return false
	}
	rtgAsmStoreRaxStack(a, srcPtr)
	rtgAsmStoreRdxStack(a, srcLen)
	rtgAsmMovRaxImm(a, 0)
	rtgAsmStoreRaxStack(a, count)
	loopLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, destLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmPushRax(a)
	rtgAsmLoadRaxStack(a, srcLen)
	rtgAsmPopRcx(a)
	rtgAsmCmpRcxRaxSet(a, 0x9d)
	rtgAsmCmpRaxImm8(a, 0)
	rtgAsmJnzLabel(a, doneLabel)
	rtgEmitAppendExpansionCopyElement(g, elemSize, srcPtr, count, destPtr, count)
	rtgAsmLoadRaxStack(a, count)
	rtgAsmIncRax(a)
	rtgAsmStoreRaxStack(a, count)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)
	rtgAsmLoadRaxStack(a, count)
	return true
}

func rtgEmitSliceBasePtrLenTokens(g *rtgLinearGen, p *rtgProgram, start int, end int, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	if start+1 == end && rtgTokIsKind(p, start, rtgTokIdent) {
		nameStart := p.toks[start].start
		nameEnd := p.toks[start].end
		localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, nameStart, nameEnd)
		globalType := rtgFindGlobalType(g, nameStart, nameEnd)
		if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		return false
	}
	if start+3 == end && rtgTokIsKind(p, start, rtgTokIdent) && rtgTokCharIs(p, start+1, '.') && rtgTokIsKind(p, start+2, rtgTokIdent) {
		localIndex := rtgFindLocalIndex(g, p.toks[start].start, p.toks[start].end)
		if localIndex < 0 {
			return false
		}
		fieldType := rtgStructFieldType(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if !rtgTypeIsSlice(meta, fieldType) {
			return false
		}
		fieldOffset := rtgStructFieldOffset(g, g.locals[localIndex].typ, p.toks[start+2].start, p.toks[start+2].end)
		if fieldOffset < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
		} else {
			rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		return true
	}
	return rtgEmitSlicePtrLen(g, ep, idx)
}

func rtgEmitSlicePtrLen(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprSlice {
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmMovRcxRdx(a)
		return true
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
			globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
			if globalOffset < 0 || (!rtgTypeIsSlice(meta, globalType) && !rtgTypeIsString(meta, globalType)) {
				return false
			}
			rtgAsmLoadRaxBss(a, globalOffset+8)
			rtgAsmMovRcxRax(a)
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		if !rtgTypeIsSlice(meta, g.locals[localIndex].typ) && !rtgTypeIsString(meta, g.locals[localIndex].typ) {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmLoadRcxStack(a, g.locals[localIndex].offset-8)
		return true
	}
	if e.kind == rtgExprComposite {
		sliceType := rtgTypeFromExpr(g, ep, idx)
		if !rtgTypeIsSlice(meta, sliceType) {
			return false
		}
		if !rtgEmitSliceLiteralRegs(g, ep, idx, sliceType) {
			return false
		}
		rtgAsmMovRcxRdx(a)
		return true
	}
	if e.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, fieldType) && !rtgTypeIsString(meta, fieldType) {
			return false
		}
		if !rtgEmitSelectorAddressRdx(g, ep, idx) {
			return false
		}
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmMemDisp(a, 8, 0x8b48, 0x4a, 0x8a)
		return true
	}
	if e.kind == rtgExprCall {
		valueType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsSlice(meta, valueType) {
			return false
		}
		if !rtgEmitSliceValueRegs(g, ep, idx) {
			return false
		}
		rtgAsmMovRcxRdx(a)
		return true
	}
	return false
}

func rtgEmitIndexedStructField(g *rtgLinearGen, ep *rtgExprParse, indexIdx int, fieldStart int, fieldEnd int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftIndex := indexExpr.left
	rightIndex := indexExpr.right
	leftType := rtgInferParsedExprType(g, ep, leftIndex)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemType := rtgResolveType(g.meta, sliceType.elem)
	if elemType.kind != rtgTypeStruct {
		return false
	}
	fieldOffset := rtgStructFieldOffset(g, sliceType.elem, fieldStart, fieldEnd)
	if fieldOffset < 0 {
		return false
	}
	fieldType := rtgStructFieldType(g, sliceType.elem, fieldStart, fieldEnd)
	if !rtgEmitIntExpr(g, ep, rightIndex) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, leftIndex) {
		return false
	}
	rtgAsmPopRcx(a)
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	rtgAsmImulRcxImm(a, elemSize)
	rtgAsmAddRaxRcx(a)
	rtgAsmMovRdxRax(a)
	fieldResolved := rtgResolveType(g.meta, fieldType)
	fieldSize := rtgScalarKindSize(fieldResolved.kind)
	rtgAsmLoadRaxMemRdxDispSize(a, fieldOffset, fieldSize)
	return true
}

func rtgEmitIndexAddressRax(g *rtgLinearGen, ep *rtgExprParse, indexIdx int) bool {
	a := &g.asm
	indexExpr := &ep.exprs[indexIdx]
	leftType := rtgInferParsedExprType(g, ep, indexExpr.left)
	sliceType := rtgResolveType(g.meta, leftType)
	if sliceType.kind != rtgTypeSlice {
		return false
	}
	elemSize := rtgTypeSize(g.meta, sliceType.elem)
	if elemSize < 1 {
		elemSize = 8
	}
	if !rtgEmitIntExpr(g, ep, indexExpr.right) {
		return false
	}
	rtgAsmPushRax(a)
	if !rtgEmitSlicePtrLen(g, ep, indexExpr.left) {
		return false
	}
	rtgAsmPopRcx(a)
	if elemSize != 1 {
		rtgAsmImulRcxImm(a, elemSize)
	}
	rtgAsmAddRaxRcx(a)
	return true
}

func rtgEmitIndexExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	p := g.prog
	a := &g.asm
	e := &ep.exprs[idx]
	left := &ep.exprs[e.left]
	if left.kind == rtgExprString {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		msg := rtgDecodeStringToken(g.prog, left.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmPushRax(a)
		rtgAsmMovRaxDataAddr(a, msgOff)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	if left.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, left.nameStart, left.nameEnd)
			globalType := rtgFindGlobalType(g, left.nameStart, left.nameEnd)
			if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			if globalOffset >= 0 && rtgTypeIsSlice(meta, globalType) {
				t := rtgResolveType(meta, globalType)
				elem := rtgResolveType(meta, t.elem)
				if !rtgTypeKindIsScalarInt(elem.kind) {
					return false
				}
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				rtgAsmPushRax(a)
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmPopRcx(a)
				elemSize := rtgScalarKindSize(elem.kind)
				rtgAsmLoadRaxIndexRcxSize(a, elemSize)
				return true
			}
			constTok := rtgFindConstStringToken(g, left.nameStart, left.nameEnd)
			if constTok >= 0 {
				if !rtgEmitIntExpr(g, ep, e.right) {
					return false
				}
				msg := rtgDecodeStringToken(g.prog, constTok)
				msgOff := rtgAddStringData(g, msg)
				rtgAsmPushRax(a)
				rtgAsmMovRaxDataAddr(a, msgOff)
				rtgAsmPopRcx(a)
				rtgAsmLoadByteRaxIndexRcx(a)
				return true
			}
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarInt(elem.kind) {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			rtgAsmPopRcx(a)
			elemSize := rtgScalarKindSize(elem.kind)
			rtgAsmLoadRaxIndexRcxSize(a, elemSize)
			return true
		}
	}
	if left.kind == rtgExprSelector {
		fieldType := rtgInferParsedExprType(g, ep, e.left)
		t := rtgResolveType(meta, fieldType)
		if t.kind == rtgTypeString {
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			rtgAsmLoadByteRaxIndexRcx(a)
			return true
		}
		if t.kind == rtgTypeSlice {
			elem := rtgResolveType(meta, t.elem)
			if !rtgTypeKindIsScalarInt(elem.kind) {
				return false
			}
			if !rtgEmitIntExpr(g, ep, e.right) {
				return false
			}
			rtgAsmPushRax(a)
			if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
				return false
			}
			rtgAsmLoadRaxMemRdxDisp(a, 0)
			rtgAsmPopRcx(a)
			elemSize := rtgScalarKindSize(elem.kind)
			rtgAsmLoadRaxIndexRcxSize(a, elemSize)
			return true
		}
	}
	if left.kind == rtgExprUnary && rtgTokCharIs(p, left.tok, '*') {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitIntExpr(g, ep, left.left) {
			return false
		}
		rtgAsmMovRdxRax(a)
		rtgAsmLoadRaxMemRdxDisp(a, 0)
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	if left.kind == rtgExprIndex {
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitStringPtrExpr(g, ep, e.left) {
			return false
		}
		rtgAsmPopRcx(a)
		rtgAsmLoadByteRaxIndexRcx(a)
		return true
	}
	return false
}

func rtgEmitStringPtrExpr(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	p := g.prog
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	if e.kind == rtgExprString {
		msg := rtgDecodeStringToken(p, e.tok)
		msgOff := rtgAddStringData(g, msg)
		rtgAsmMovRaxDataAddr(a, msgOff)
		return true
	}
	if e.kind == rtgExprCall && e.argCount == 1 && rtgExprIsIdentText(p, ep, e.left, "string") {
		argIndex := ep.args[e.firstArg]
		arg := &ep.exprs[argIndex]
		if arg.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, arg.nameStart, arg.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeByte {
			return false
		}
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		return true
	}
	if e.kind == rtgExprCall {
		callType := rtgInferParsedExprType(g, ep, idx)
		if !rtgTypeIsString(meta, callType) {
			return false
		}
		return rtgEmitStringValueRegs(g, ep, idx)
	}
	if e.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, e.nameStart, e.nameEnd)
		if localIndex >= 0 {
			if !rtgTypeIsString(meta, g.locals[localIndex].typ) {
				return false
			}
			rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
			return true
		}
		globalOffset := rtgFindGlobalOffset(g, e.nameStart, e.nameEnd)
		globalType := rtgFindGlobalType(g, e.nameStart, e.nameEnd)
		if globalOffset >= 0 && rtgTypeIsString(meta, globalType) {
			rtgAsmLoadRaxBss(a, globalOffset)
			return true
		}
		constTok := rtgFindConstStringToken(g, e.nameStart, e.nameEnd)
		if constTok >= 0 {
			msg := rtgDecodeStringToken(p, constTok)
			msgOff := rtgAddStringData(g, msg)
			rtgAsmMovRaxDataAddr(a, msgOff)
			return true
		}
		return false
	}
	if e.kind == rtgExprIndex {
		left := &ep.exprs[e.left]
		if left.kind != rtgExprIdent {
			return false
		}
		localIndex := rtgFindLocalIndex(g, left.nameStart, left.nameEnd)
		if localIndex < 0 {
			return false
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind != rtgTypeSlice {
			return false
		}
		elem := rtgResolveType(meta, t.elem)
		if elem.kind != rtgTypeString {
			return false
		}
		if !rtgEmitIntExpr(g, ep, e.right) {
			return false
		}
		rtgAsmPushRax(a)
		rtgAsmLoadRaxStack(a, g.locals[localIndex].offset)
		rtgAsmPopRcx(a)
		rtgAsmShlRcxImm(a, 4)
		rtgAsmEmit24(a, 0x08048b)
		return true
	}
	return false
}

func rtgFindLocalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	localIndex := rtgFindLocalIndex(g, nameStart, nameEnd)
	if localIndex < 0 {
		return -1
	}
	return g.locals[localIndex].offset
}

func rtgFindLocalIndex(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := len(g.locals) - 1; i >= 0; i-- {
		if rtgBytesEqualRange(g.prog.src, g.locals[i].nameStart, g.locals[i].nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}

func rtgEmitSelectorAddressRdx(g *rtgLinearGen, ep *rtgExprParse, idx int) bool {
	meta := g.meta
	a := &g.asm
	e := &ep.exprs[idx]
	base := &ep.exprs[e.left]
	baseType := rtgInferParsedExprType(g, ep, e.left)
	fieldOffset := rtgStructFieldOffset(g, baseType, e.nameStart, e.nameEnd)
	if fieldOffset < 0 {
		return false
	}
	if base.kind == rtgExprIndex {
		leftType := rtgInferParsedExprType(g, ep, base.left)
		sliceType := rtgResolveType(meta, leftType)
		elemType := rtgResolveType(meta, sliceType.elem)
		if sliceType.kind != rtgTypeSlice || elemType.kind != rtgTypeStruct {
			return false
		}
		if !rtgEmitIntExpr(g, ep, base.right) {
			return false
		}
		rtgAsmPushRax(a)
		if !rtgEmitSlicePtrLen(g, ep, base.left) {
			return false
		}
		rtgAsmPopRcx(a)
		elemSize := rtgTypeSize(meta, sliceType.elem)
		rtgAsmImulRcxImm(a, elemSize)
		rtgAsmMovRdxRax(a)
		rtgAsmAddRdxRcx(a)
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	if base.kind == rtgExprIdent {
		localIndex := rtgFindLocalIndex(g, base.nameStart, base.nameEnd)
		if localIndex < 0 {
			globalOffset := rtgFindGlobalOffset(g, base.nameStart, base.nameEnd)
			globalType := rtgFindGlobalType(g, base.nameStart, base.nameEnd)
			t := rtgResolveType(meta, globalType)
			if globalOffset < 0 {
				return false
			}
			if t.kind == rtgTypePointer {
				rtgAsmLoadRaxBss(a, globalOffset)
				rtgAsmMovRdxRax(a)
				if fieldOffset != 0 {
					rtgAsmAddRdxImm(a, fieldOffset)
				}
				return true
			}
			if t.kind != rtgTypeStruct {
				return false
			}
			rtgAsmMovRaxBssAddr(a, globalOffset)
			rtgAsmMovRdxRax(a)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		t := rtgResolveType(meta, g.locals[localIndex].typ)
		if t.kind == rtgTypePointer {
			rtgAsmLoadRdxStack(a, g.locals[localIndex].offset)
			if fieldOffset != 0 {
				rtgAsmAddRdxImm(a, fieldOffset)
			}
			return true
		}
		rtgAsmStackMem(a, g.locals[localIndex].offset-fieldOffset, 0x8d48, 0x55, 0x95)
		return true
	}
	if base.kind == rtgExprSelector {
		if !rtgEmitSelectorAddressRdx(g, ep, e.left) {
			return false
		}
		t := rtgResolveType(meta, baseType)
		if t.kind == rtgTypePointer {
			rtgAsmEmit16(a, 0x128b)
		}
		if fieldOffset != 0 {
			rtgAsmAddRdxImm(a, fieldOffset)
		}
		return true
	}
	return false
}

func rtgStructFieldIndex(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	meta := g.meta
	if typ < 0 || typ >= len(meta.types) {
		return -1
	}
	t := rtgResolveType(meta, typ)
	if t.kind == rtgTypePointer && t.elem > 0 && t.elem < len(meta.types) {
		t = rtgResolveType(meta, t.elem)
	}
	if t.kind != rtgTypeStruct {
		return -1
	}
	for i := 0; i < t.count; i++ {
		field := &meta.fields[t.first+i]
		if rtgBytesEqualRange(g.prog.src, field.nameStart, field.nameEnd, nameStart, nameEnd) {
			return t.first + i
		}
	}
	return -1
}

func rtgStructFieldOffset(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return -1
	}
	return g.meta.fields[fieldIndex].offset
}

func rtgStructFieldType(g *rtgLinearGen, typ int, nameStart int, nameEnd int) int {
	fieldIndex := rtgStructFieldIndex(g, typ, nameStart, nameEnd)
	if fieldIndex < 0 {
		return 0
	}
	return g.meta.fields[fieldIndex].typ
}

func rtgCompositeStructFieldIndex(g *rtgLinearGen, typ int, field *rtgCompositeField, pos int) int {
	if field.nameEnd > field.nameStart {
		return rtgStructFieldIndex(g, typ, field.nameStart, field.nameEnd)
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind != rtgTypeStruct || pos < 0 || pos >= t.count {
		return -1
	}
	return t.first + pos
}

func rtgFindGlobalOffset(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.globals); i++ {
		if rtgBytesEqualRange(g.prog.src, g.globals[i].nameStart, g.globals[i].nameEnd, nameStart, nameEnd) {
			return g.globals[i].offset
		}
	}
	return -1
}

func rtgFindGlobalType(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokVar && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			return s.typ
		}
	}
	return 0
}

func rtgFindConstStringToken(g *rtgLinearGen, nameStart int, nameEnd int) int {
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.initStart+1 == s.initEnd && rtgTokIsKind(g.prog, s.initStart, rtgTokString) {
				return s.initStart
			}
		}
	}
	return -1
}

func rtgFindSmallConstByName(g *rtgLinearGen, nameStart int, nameEnd int) int {
	if rtgFindLocalIndex(g, nameStart, nameEnd) >= 0 {
		return -129
	}
	if rtgBytesEqualText(g.prog.src, nameStart, nameEnd, "nil") {
		return 0
	}
	for i := 0; i < len(g.meta.globals); i++ {
		s := &g.meta.globals[i]
		if s.kind == rtgTokConst && rtgBytesEqualRange(g.prog.src, s.nameStart, s.nameEnd, nameStart, nameEnd) {
			if s.initStart+1 != s.initEnd {
				return -129
			}
			if rtgTokIsKind(g.prog, s.initStart, rtgTokNumber) {
				value := rtgParseIntToken(g.prog, s.initStart)
				if rtgAsmImmFits8Signed(value) {
					return value
				}
			}
			if rtgTokIsKind(g.prog, s.initStart, rtgTokChar) {
				value := rtgParseCharToken(g.prog, s.initStart)
				if rtgAsmImmFits8Signed(value) {
					return value
				}
			}
			return -129
		}
	}
	return -129
}

func rtgAddTypedLocal(g *rtgLinearGen, nameStart int, nameEnd int, typ int) int {
	size := rtgTypeSize(g.meta, typ)
	if size < 8 {
		size = 8
	}
	g.stackUsed = rtgAlignTo8(g.stackUsed + size)
	offset := g.stackUsed
	g.locals = append(g.locals, rtgLocalInfo{nameStart: nameStart, nameEnd: nameEnd, offset: offset, typ: typ, size: size})
	return offset
}

func rtgZeroLocalAtOffset(g *rtgLinearGen, offset int) {
	a := &g.asm
	size := 8
	typ := rtgTypeInt
	for i := 0; i < len(g.locals); i++ {
		if g.locals[i].offset == offset {
			size = g.locals[i].size
			typ = g.locals[i].typ
		}
	}
	t := rtgResolveType(g.meta, typ)
	if t.kind == rtgTypeSlice {
		elemSize := rtgTypeSize(g.meta, t.elem)
		if elemSize < 1 {
			elemSize = 8
		}
		backingSize := 2097152
		backingOff := g.asm.bssSize
		g.asm.bssSize += backingSize
		rtgAsmMovRaxBssAddr(a, backingOff)
		rtgAsmStoreRaxStack(a, offset)
		rtgAsmMovRaxImm(a, 0)
		rtgAsmStoreRaxStack(a, offset-8)
		rtgAsmMovRaxImm(a, backingSize/elemSize)
		rtgAsmStoreRaxStack(a, offset-16)
		return
	}
	rtgAsmMovRaxImm(a, 0)
	for at := 0; at < size; at += 8 {
		rtgAsmStoreRaxStack(a, offset-at)
	}
}

func rtgFuncInfoFromCall(g *rtgLinearGen, ep *rtgExprParse, idx int) int {
	e := &ep.exprs[idx]
	nameStart := e.nameStart
	nameEnd := e.nameEnd
	wantMethod := false
	if e.kind == rtgExprSelector {
		wantMethod = true
	} else if e.kind != rtgExprIdent {
		return -1
	}
	for i := 0; i < len(g.meta.funcs); i++ {
		f := &g.meta.funcs[i]
		isMethod := f.receiverType != 0
		if isMethod == wantMethod && rtgBytesEqualRange(g.prog.src, f.nameStart, f.nameEnd, nameStart, nameEnd) {
			return i
		}
	}
	return -1
}
