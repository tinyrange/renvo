package main

// renvoSoftFloatSource is compiled only when a VM32 program reaches
// one of these helpers.  Keeping the fallback in ordinary Renvo source makes
// the IEEE implementation independent of the host compiler and exercises the
// same integer backend as user code.  The algorithms follow the structure of
// Go's runtime softfloat implementation (BSD licensed), with the division
// primitive expressed as a small fixed-width long division so it also works on
// the deliberately minimal VM32 instruction set.
const renvoSoftFloatSource = `
const __renvoSoftMant64 uint = 52
const __renvoSoftExp64 uint = 11
const __renvoSoftBias64 = -1023
const __renvoSoftNaN64 uint64 = 0x7ff8000000000000
const __renvoSoftInf64 uint64 = 0x7ff0000000000000
const __renvoSoftNeg64 uint64 = 0x8000000000000000

func __renvoSoftUnpack64(f uint64) (sign uint64, mant uint64, exp int) {
	sign = f & 0x8000000000000000
	mant = f & 0x000fffffffffffff
	exp = int(f >> __renvoSoftMant64) & 0x7ff
	if exp == 0x7ff {
		if mant != 0 {
			exp = 2049
			return
		}
		exp = 2048
		return
	}
	if exp == 0 {
		if mant != 0 {
			exp = __renvoSoftBias64 + 1
			for mant < uint64(1)<<__renvoSoftMant64 {
				mant <<= 1
				exp--
			}
		}
		return
	}
	mant |= uint64(1) << __renvoSoftMant64
	exp += __renvoSoftBias64
	return
}

func __renvoSoftPack64(sign uint64, mant uint64, exp int, trunc uint64) uint64 {
	mant0 := mant
	exp0 := exp
	trunc0 := trunc
	if mant == 0 {
		return sign
	}
	for mant < uint64(1)<<__renvoSoftMant64 {
		mant <<= 1
		exp--
	}
	for mant >= uint64(4)<<__renvoSoftMant64 {
		trunc |= mant & 1
		mant >>= 1
		exp++
	}
	if mant >= uint64(2)<<__renvoSoftMant64 {
		if mant&1 != 0 && (trunc != 0 || mant&2 != 0) {
			mant++
			if mant >= uint64(4)<<__renvoSoftMant64 {
				mant >>= 1
				exp++
			}
		}
		mant >>= 1
		exp++
	}
	if exp >= 1024 {
		return sign ^ 0x7ff0000000000000
	}
	if exp < __renvoSoftBias64+1 {
		if exp < __renvoSoftBias64-int(__renvoSoftMant64) {
			return sign
		}
		mant = mant0
		exp = exp0
		trunc = trunc0
		for exp < __renvoSoftBias64 {
			trunc |= mant & 1
			mant >>= 1
			exp++
		}
		if mant&1 != 0 && (trunc != 0 || mant&2 != 0) {
			mant++
		}
		mant >>= 1
		exp++
		if mant < uint64(1)<<__renvoSoftMant64 {
			return sign | mant
		}
	}
	return sign | uint64(exp-__renvoSoftBias64)<<__renvoSoftMant64 | mant&0x000fffffffffffff
}

func __renvoSoftAdd64(f uint64, g uint64) uint64 {
	fs, fm, fe := __renvoSoftUnpack64(f)
	gs, gm, ge := __renvoSoftUnpack64(g)
	fi := fe == 2048
	fn := fe == 2049
	gi := ge == 2048
	gn := ge == 2049
	if fn || gn {
		return 0x7ff8000000000000
	}
	if fi && gi && fs != gs {
		return 0x7ff8000000000000
	}
	if fi {
		return f
	}
	if gi {
		return g
	}
	if fm == 0 && gm == 0 && fs != 0 && gs != 0 {
		return f
	}
	if fm == 0 {
		if gm == 0 {
			g ^= gs
		}
		return g
	}
	if gm == 0 {
		return f
	}
	if fe < ge || fe == ge && fm < gm {
		tf, ts, tm, te := f, fs, fm, fe
		f, fs, fm, fe = g, gs, gm, ge
		g, gs, gm, ge = tf, ts, tm, te
	}
	shift := uint(fe - ge)
	fm <<= 2
	gm <<= 2
	trunc := uint64(0)
	if shift >= 64 {
		trunc = gm
		gm = 0
	} else if shift != 0 {
		trunc = gm & (uint64(1)<<shift - 1)
		gm >>= shift
	}
	if fs == gs {
		fm += gm
	} else {
		fm -= gm
		if trunc != 0 {
			fm--
		}
	}
	if fm == 0 {
		fs = 0
	}
	return __renvoSoftPack64(fs, fm, fe-2, trunc)
}

func __renvoSoftSub64(f uint64, g uint64) uint64 {
	return __renvoSoftAdd64(f, g^0x8000000000000000)
}

func __renvoSoftMul128(u uint64, v uint64) (lo uint64, hi uint64) {
	u0 := u & 0xffffffff
	u1 := u >> 32
	v0 := v & 0xffffffff
	v1 := v >> 32
	w0 := u0 * v0
	t := u1*v0 + w0>>32
	w1 := t & 0xffffffff
	w2 := t >> 32
	w1 += u0 * v1
	lo = u * v
	hi = u1*v1 + w2 + w1>>32
	return
}

func __renvoSoftMul64(f uint64, g uint64) uint64 {
	fs, fm, fe := __renvoSoftUnpack64(f)
	gs, gm, ge := __renvoSoftUnpack64(g)
	fi := fe == 2048
	fn := fe == 2049
	gi := ge == 2048
	gn := ge == 2049
	if fn || gn {
		return 0x7ff8000000000000
	}
	if fi && gi {
		return f ^ gs
	}
	if fi && gm == 0 || fm == 0 && gi {
		return 0x7ff8000000000000
	}
	if fm == 0 {
		return f ^ gs
	}
	if gm == 0 {
		return g ^ fs
	}
	lo, hi := __renvoSoftMul128(fm, gm)
	shift := __renvoSoftMant64 - 1
	trunc := lo & (uint64(1)<<shift - 1)
	mant := hi<<(64-shift) | lo>>shift
	return __renvoSoftPack64(fs^gs, mant, fe+ge-1, trunc)
}

func __renvoSoftDiv128(hi uint64, lo uint64, divisor uint64) (quotient uint64, remainder uint64) {
	quotient = 0
	remainder = hi
	for bit := 0; bit < 64; bit++ {
		overflow := remainder >> 63
		remainder = remainder<<1 | lo>>63
		lo <<= 1
		quotient <<= 1
		if overflow != 0 || remainder >= divisor {
			remainder -= divisor
			quotient |= 1
		}
	}
	return
}

func __renvoSoftDiv64(f uint64, g uint64) uint64 {
	if g&0x7fffffffffffffff == 0 {
		if f&0x7fffffffffffffff == 0 {
			return 0x7ff8000000000000
		}
		return (f^g)&0x8000000000000000 | 0x7ff0000000000000
	}
	fs, fm, fe := __renvoSoftUnpack64(f)
	gs, gm, ge := __renvoSoftUnpack64(g)
	fi := fe == 2048
	fn := fe == 2049
	gi := ge == 2048
	gn := ge == 2049
	if fn || gn {
		return 0x7ff8000000000000
	}
	if fi && gi {
		return 0x7ff8000000000000
	}
	if !fi && !gi && fm == 0 && gm == 0 {
		return 0x7ff8000000000000
	}
	if fi {
		return (fs ^ gs) | 0x7ff0000000000000
	}
	if !gi && gm == 0 {
		return (fs ^ gs) | 0x7ff0000000000000
	}
	if gi {
		return fs ^ gs
	}
	if fm == 0 {
		return fs ^ gs
	}
	shift := __renvoSoftMant64 + 2
	q, r := __renvoSoftDiv128(fm>>(64-shift), fm<<shift, gm)
	return __renvoSoftPack64(fs^gs, q, fe-ge-2, r)
}

func __renvoSoftCompareCode64(f uint64, g uint64) int {
	fa := f & 0x7fffffffffffffff
	ga := g & 0x7fffffffffffffff
	if fa > 0x7ff0000000000000 || ga > 0x7ff0000000000000 {
		return 2
	}
	if fa == 0 && ga == 0 {
		return 0
	}
	fs := f & 0x8000000000000000
	gs := g & 0x8000000000000000
	if fs > gs {
		return -1
	}
	if fs < gs {
		return 1
	}
	if fs == 0 {
		if f < g {
			return -1
		}
		if f > g {
			return 1
		}
	} else {
		if f > g {
			return -1
		}
		if f < g {
			return 1
		}
	}
	return 0
}

func __renvoSoftCompare64(f uint64, g uint64) (cmp int, nan bool) {
	nan = false
	cmp = __renvoSoftCompareCode64(f, g)
	if cmp == 2 {
		cmp = 0
		nan = true
	}
	return
}

func __renvoSoftEq64(f uint64, g uint64) bool {
	return __renvoSoftCompareCode64(f, g) == 0
}
func __renvoSoftNe64(f uint64, g uint64) bool {
	return __renvoSoftCompareCode64(f, g) != 0
}
func __renvoSoftLt64(f uint64, g uint64) bool {
	return __renvoSoftCompareCode64(f, g) < 0
}
func __renvoSoftLe64(f uint64, g uint64) bool {
	cmp := __renvoSoftCompareCode64(f, g)
	return cmp != 2 && cmp <= 0
}
func __renvoSoftGt64(f uint64, g uint64) bool {
	cmp := __renvoSoftCompareCode64(f, g)
	return cmp != 2 && cmp > 0
}
func __renvoSoftGe64(f uint64, g uint64) bool {
	cmp := __renvoSoftCompareCode64(f, g)
	return cmp != 2 && cmp >= 0
}

func __renvoSoftEqWord64(f uint64, g uint64) uint64 {
	if __renvoSoftEq64(f, g) { return 1 }
	return 0
}
func __renvoSoftNeWord64(f uint64, g uint64) uint64 {
	if __renvoSoftNe64(f, g) { return 1 }
	return 0
}
func __renvoSoftLtWord64(f uint64, g uint64) uint64 {
	if __renvoSoftLt64(f, g) { return 1 }
	return 0
}
func __renvoSoftLeWord64(f uint64, g uint64) uint64 {
	if __renvoSoftLe64(f, g) { return 1 }
	return 0
}
func __renvoSoftGtWord64(f uint64, g uint64) uint64 {
	if __renvoSoftGt64(f, g) { return 1 }
	return 0
}
func __renvoSoftGeWord64(f uint64, g uint64) uint64 {
	if __renvoSoftGe64(f, g) { return 1 }
	return 0
}

func __renvoSoftJoin64(low uint32, high uint32) uint64 {
	return uint64(high)<<32 | uint64(low)
}
func __renvoSoftEqWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftEq64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}
func __renvoSoftNeWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftNe64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}
func __renvoSoftLtWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftLt64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}
func __renvoSoftLeWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftLe64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}
func __renvoSoftGtWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftGt64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}
func __renvoSoftGeWords64(fl uint32, fh uint32, gl uint32, gh uint32) bool {
	return __renvoSoftGe64(__renvoSoftJoin64(fl, fh), __renvoSoftJoin64(gl, gh))
}

func __renvoSoftInt64To64(value int64) uint64 {
	sign := uint64(value) & 0x8000000000000000
	mant := uint64(value)
	if sign != 0 {
		mant = -mant
	}
	return __renvoSoftPack64(sign, mant, int(__renvoSoftMant64), 0)
}

func __renvoSoftUint64To64(value uint64) uint64 {
	if int64(value) >= 0 {
		return __renvoSoftInt64To64(int64(value))
	}
	rounded := value>>1 | value&1
	result := __renvoSoftInt64To64(int64(rounded))
	return __renvoSoftAdd64(result, result)
}

func __renvoSoftUint64To32Wide(value uint64) uint64 {
	return uint64(__renvoSoft64To32(__renvoSoftUint64To64(value)))
}

func __renvoSoft64ToInt64(f uint64) uint64 {
	sign, mant, exp := __renvoSoftUnpack64(f)
	if exp == 2048 || exp == 2049 {
		return 0x8000000000000000
	}
	if exp < 0 {
		return 0
	}
	if exp > 63 {
		return 0x8000000000000000
	}
	for exp > 52 {
		mant <<= 1
		exp--
	}
	for exp < 52 {
		mant >>= 1
		exp++
	}
	if sign != 0 {
		return -mant
	}
	return mant
}

func __renvoSoft64ToUint64(f uint64) uint64 {
	sign, mant, exp := __renvoSoftUnpack64(f)
	if sign != 0 || exp == 2048 || exp == 2049 {
		return 0
	}
	if exp < 0 {
		return 0
	}
	if exp > 63 {
		return 0xffffffffffffffff
	}
	for exp > 52 {
		mant <<= 1
		exp--
	}
	for exp < 52 {
		mant >>= 1
		exp++
	}
	return mant
}

func __renvoSoftUnpack32(f uint32) (sign uint32, mant uint32, exp int) {
	sign = f & 0x80000000
	mant = f & 0x007fffff
	exp = int(f>>23) & 0xff
	if exp == 0xff {
		if mant != 0 {
			exp = 257
			return
		}
		exp = 256
		return
	}
	if exp == 0 {
		if mant != 0 {
			exp = -126
			for mant < uint32(1)<<23 {
				mant <<= 1
				exp--
			}
		}
		return
	}
	mant |= uint32(1) << 23
	exp -= 127
	return
}

func __renvoSoftPack32(sign uint32, mant uint32, exp int, trunc uint32) uint32 {
	mant0 := mant
	exp0 := exp
	trunc0 := trunc
	if mant == 0 {
		return sign
	}
	for mant < uint32(1)<<23 {
		mant <<= 1
		exp--
	}
	for mant >= uint32(4)<<23 {
		trunc |= mant & 1
		mant >>= 1
		exp++
	}
	if mant >= uint32(2)<<23 {
		if mant&1 != 0 && (trunc != 0 || mant&2 != 0) {
			mant++
			if mant >= uint32(4)<<23 {
				mant >>= 1
				exp++
			}
		}
		mant >>= 1
		exp++
	}
	if exp >= 128 {
		return sign ^ 0x7f800000
	}
	if exp < -126 {
		if exp < -127-int(23) {
			return sign
		}
		mant = mant0
		exp = exp0
		trunc = trunc0
		for exp < -127 {
			trunc |= mant & 1
			mant >>= 1
			exp++
		}
		if mant&1 != 0 && (trunc != 0 || mant&2 != 0) {
			mant++
		}
		mant >>= 1
		exp++
		if mant < uint32(1)<<23 {
			return sign | mant
		}
	}
	return sign | uint32(exp+127)<<23 | mant&0x007fffff
}

func __renvoSoft64To32(f uint64) uint32 {
	fs, fm, fe := __renvoSoftUnpack64(f)
	fi := fe == 2048
	fn := fe == 2049
	if fn {
		return 0x7fc00000
	}
	fs32 := uint32(fs >> 32)
	if fi {
		return fs32 ^ 0x7f800000
	}
	const shift = 28
	return __renvoSoftPack32(fs32, uint32(fm>>shift), fe-1, uint32(fm&(uint64(1)<<shift-1)))
}

func __renvoSoft32To64(f uint32) uint64 {
	fs, fm, fe := __renvoSoftUnpack32(f)
	fi := fe == 256
	fn := fe == 257
	if fn {
		return 0x7ff8000000000000
	}
	fs64 := uint64(fs) << 32
	if fi {
		return fs64 ^ 0x7ff0000000000000
	}
	return __renvoSoftPack64(fs64, uint64(fm)<<29, fe, 0)
}

func __renvoSoftAdd32(f uint32, g uint32) uint32 {
	return __renvoSoft64To32(__renvoSoftAdd64(__renvoSoft32To64(f), __renvoSoft32To64(g)))
}
func __renvoSoftSub32(f uint32, g uint32) uint32 {
	return __renvoSoftAdd32(f, g^0x80000000)
}
func __renvoSoftMul32(f uint32, g uint32) uint32 {
	return __renvoSoft64To32(__renvoSoftMul64(__renvoSoft32To64(f), __renvoSoft32To64(g)))
}
func __renvoSoftDiv32(f uint32, g uint32) uint32 {
	return __renvoSoft64To32(__renvoSoftDiv64(__renvoSoft32To64(f), __renvoSoft32To64(g)))
}
func __renvoSoftInt64To32(value int64) uint32 {
	sign := uint64(value) & 0x8000000000000000
	mant := uint64(value)
	if sign != 0 {
		mant = -mant
	}
	exp := 23
	trunc := uint32(0)
	for mant >= uint64(1)<<32 {
		trunc |= uint32(mant) & 1
		mant >>= 1
		exp++
	}
	return __renvoSoftPack32(uint32(sign>>32), uint32(mant), exp, trunc)
}
func __renvoSoftInt64To32Wide(value int64) uint64 { return uint64(__renvoSoftInt64To32(value)) }
func __renvoSoft64To32Wide(value uint64) uint64 { return uint64(__renvoSoft64To32(value)) }
func __renvoSoft32To64Wide(value uint64) uint64 { return __renvoSoft32To64(uint32(value)) }
func __renvoSoft32To64Scalar(value uint32) uint64 { return __renvoSoft32To64(value) }
func __renvoSoft32WideTo64(value uint64) uint64 {
	f := value & 0xffffffff
	sign := f & 0x80000000
	mant := f & 0x007fffff
	exp := int(f>>23) & 0xff
	if exp == 0xff {
		if mant != 0 { return 0x7ff8000000000000 }
		return sign<<32 ^ 0x7ff0000000000000
	}
	if exp == 0 {
		if mant == 0 { return sign << 32 }
		exp = -126
		for mant < uint64(1)<<23 {
			mant <<= 1
			exp--
		}
	} else {
		mant |= uint64(1) << 23
		exp -= 127
	}
	return __renvoSoftPack64(sign<<32, mant<<29, exp, 0)
}
func __renvoSoft32WideToInt64(value uint64) uint64 {
	f := uint32(value)
	sign := f & 0x80000000
	mant := f & 0x007fffff
	exp := int(f>>23) & 0xff
	if exp == 0xff {
		return 0x8000000000000000
	}
	if exp == 0 {
		if mant == 0 {
			return 0
		}
		exp = -126
		for mant < uint32(1)<<23 {
			mant <<= 1
			exp--
		}
	} else {
		mant |= uint32(1) << 23
		exp -= 127
	}
	if exp < 0 {
		return 0
	}
	if exp > 63 {
		return 0x8000000000000000
	}
	for exp > 23 {
		mant <<= 1
		exp--
	}
	for exp < 23 {
		mant >>= 1
		exp++
	}
	if sign != 0 {
		return -uint64(mant)
	}
	return uint64(mant)
}
func __renvoSoft32WideToUint64(value uint64) uint64 {
	f := uint32(value)
	sign := f & 0x80000000
	mant := f & 0x007fffff
	exp := int(f>>23) & 0xff
	if sign != 0 || exp == 0xff {
		return 0
	}
	if exp == 0 {
		if mant == 0 {
			return 0
		}
		exp = -126
		for mant < uint32(1)<<23 {
			mant <<= 1
			exp--
		}
	} else {
		mant |= uint32(1) << 23
		exp -= 127
	}
	if exp < 0 {
		return 0
	}
	if exp > 63 {
		return 0xffffffffffffffff
	}
	wide := uint64(mant)
	for exp > 23 {
		wide <<= 1
		exp--
	}
	for exp < 23 {
		wide >>= 1
		exp++
	}
	return wide
}
func __renvoSoftAdd32Wide(f uint64, g uint64) uint64 { return uint64(__renvoSoftAdd32(uint32(f), uint32(g))) }
func __renvoSoftSub32Wide(f uint64, g uint64) uint64 { return uint64(__renvoSoftSub32(uint32(f), uint32(g))) }
func __renvoSoftMul32Wide(f uint64, g uint64) uint64 { return uint64(__renvoSoftMul32(uint32(f), uint32(g))) }
func __renvoSoftDiv32Wide(f uint64, g uint64) uint64 { return uint64(__renvoSoftDiv32(uint32(f), uint32(g))) }
`

func renvoAppendSoftFloatSource(src []byte) []byte {
	src = append(src, '\n')
	for i := 0; i < len(renvoSoftFloatSource); i++ {
		src = append(src, renvoSoftFloatSource[i])
	}
	return src
}

func renvoProgramNeedsSoftFloat(prog *renvoProgram) bool {
	for i := 0; i < renvoTokCount(prog); i++ {
		if renvoTokIsKind(prog, i, renvoTokFloat) {
			return true
		}
		if !renvoTokIsKind(prog, i, renvoTokIdent) {
			continue
		}
		tok := renvoTokAt(prog, i)
		if renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "float32") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "float64") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "complex64") ||
			renvoBytesEqualText(prog.src, int(tok.start), int(tok.end), "complex128") {
			return true
		}
	}
	return false
}

func compileWasiWasm32(input []int, output int) int {
	return compileWasiWasm32Arena(input, output, 0)
}

func compileWasiWasm32Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetWasiWasm32)
	return compileWasm32Arena(input, output, arenaSize)
}

func compileVM32(input []int, output int) int {
	return compileVM32Arena(input, output, 0)
}

func compileVM32Arena(input []int, output int, arenaSize int) int {
	renvoSetTarget(renvoTargetVM32)
	return compileWasm32Arena(input, output, arenaSize)
}

func compileWasm32Arena(input []int, output int, arenaSize int) int {
	src := renvoMakeByteScratch(655360)
	for i := 0; i < len(input); i++ {
		src = renvoReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog renvoProgram
	prog = renvoParseProgram(src)
	if !prog.ok {
		return 1
	}
	if renvoTarget == renvoTargetVM32 && renvoProgramNeedsSoftFloat(&prog) {
		src = renvoAppendSoftFloatSource(src)
		prog = renvoParseProgram(src)
		if !prog.ok {
			return 1
		}
	}
	var meta renvoMeta
	renvoBuildMetaInto(&prog, &meta)
	if !meta.ok {
		return 1
	}
	meta.arenaSize = renvoResolveArenaSize(renvoTarget, arenaSize)
	var result renvoCompileResult
	result = renvoTryCompileScalarProgramWasm32(&prog, &meta)
	if result.ok {
		data := result.data
		if renvoFixedTarget == 0 {
			data = renvoCompileOutputData(data, renvoTarget)
		}
		write(output, data, -1)
		return 0
	}
	renvoPrintErr("renvo: wasm32 compilation failed\n")
	return 1
}

func renvoTryCompileScalarProgramWasm32(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	appIndex := renvoProgramEntryFunction(p, meta)
	if appIndex < 0 {
		return renvoCompileResult{}
	}
	var g renvoLinearGen
	g.c = meta.c
	g.prog = p
	g.meta = meta
	g.arenaSize = meta.arenaSize
	if renvoFixedTarget == renvoTargetVM32 || renvoFixedTarget == 0 && meta.c.renvoTarget == renvoTargetVM32 {
		renvoLoadCompilerFixedTarget(&g)
		if g.fixedTargetState != 1 {
			// VM bytecode is an execution format, not a restriction on the targets
			// exposed by a compiler running inside the VM. Preserve dynamic target
			// selection so a runtime -t value remains authoritative.
			g.fixedTargetState = 1
			g.fixedTargetValue = 0
		}
	} else {
		g.fixedTargetState = 1
		g.fixedTargetValue = meta.c.renvoTarget
	}
	a := &g.asm
	renvoAsmInitWithContext(a, g.c)
	localSlotCapacity := len(meta.funcs) * 4
	if localSlotCapacity < 256 {
		localSlotCapacity = 256
	}
	a.wasmLocalSlots = make([]int32, 0, localSlotCapacity)
	for i := 0; i < len(meta.funcs); i++ {
		label := renvoAsmNewLabel(a)
		g.funcLabels = append(g.funcLabels, label)
	}
	renvoInitFuncQueue(&g, len(meta.funcs))
	renvoWasm32MarkFunc(&g, appIndex)
	renvoEmitInitializeThreadState(&g)
	renvoEmitPersistentArenaReady(&g)
	if !renvoLinearInitGlobals(&g) || !renvoEmitProgramEntryArgsWasm32(&g, appIndex) {
		return renvoCompileResult{}
	}
	renvoAsmCallLabel(a, g.funcLabels[appIndex])
	if !renvoEmitProgramPanicCheck(&g) {
		return renvoCompileResult{}
	}
	renvoWasm32AsmExit(a)
	for queueIndex := 0; queueIndex < len(g.funcQueue); queueIndex++ {
		i := g.funcQueue[queueIndex]
		if renvoDeferUnreadyQueuedClosure(&g, i) {
			continue
		}
		if !renvoEmitScalarFunctionScratch(&g, i) {
			if renvoFixedTarget == 0 {
				renvoPrintErr("renvo: wasm32 failed in function ")
				write(2, meta.prog.src[meta.funcs[i].nameStart:meta.funcs[i].nameEnd], -1)
				renvoPrintErr("\n")
			}
			return renvoCompileResult{}
		}
	}
	renvo_runtime_ArenaDiscard(meta.scratchStart, meta.scratchEnd)
	var result renvoCompileResult
	if renvoFixedTarget == renvoTargetVM32 || renvoFixedTarget == 0 && meta.c.renvoTarget == renvoTargetVM32 {
		result.data = renvoVMImage(a)
	} else {
		result.data = renvoWasm32Image(a)
	}
	result.ok = true
	return result
}
func renvoEmitProgramEntryArgsWasm32(g *renvoLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(g.meta, app.resultType) {
		return false
	}
	argsOff := g.asm.bssSize
	envDataOff := argsOff
	envLenOff := argsOff
	if renvoFixedTarget == renvoTargetVM32 || renvoFixedTarget == 0 && g.fixedTargetValue == 0 {
		// VM execution receives arguments from the VM host.
	} else {
		g.asm.bssSize += 32768
		envDataOff = g.asm.bssSize
		g.asm.bssSize += 32768
		envLenOff = g.asm.bssSize
		g.asm.bssSize += 8
	}
	renvoWasm32AsmBuildArgvEnvSlices(&g.asm, argsOff, envDataOff, envLenOff)
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
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !renvoTypeIsStringSlice(g.meta, second.typ) {
		return false
	}
	return true
}

func renvoTryCompileWasiWasm32(p *renvoProgram, meta *renvoMeta) renvoCompileResult {
	appIndex := renvoProgramEntryFunction(p, meta)
	if appIndex < 0 {
		var result renvoCompileResult
		return result
	}
	app := &meta.funcs[appIndex]
	if app.resultType != 0 && !renvoTypeIsInt(meta, app.resultType) {
		var result renvoCompileResult
		return result
	}
	if app.paramCount > 1 {
		var result renvoCompileResult
		return result
	}
	if app.paramCount == 1 {
		first := &meta.params[app.firstParam]
		if !renvoTypeIsStringSlice(meta, first.typ) {
			var result renvoCompileResult
			return result
		}
	}
	fn := &p.funcs[app.declIndex]
	var body renvoBodyParse
	body.prog = p
	body.ok = true
	statements := make([]renvoStmt, 0, 64)
	i := fn.bodyStart + 1
	for body.ok && i < fn.bodyEnd {
		if renvoTokCharIs(p, i, ';') {
			i++
			continue
		}
		if renvoTokCharIs(p, i, '}') || renvoTokIsKind(p, i, renvoTokEOF) {
			break
		}
		body.stmtCount = 0
		next := renvoParseOneStatement(&body, i, fn.bodyEnd)
		if !body.ok || next <= i || body.stmtCount != 1 {
			var result renvoCompileResult
			return result
		}
		statements = append(statements, body.stmt)
		i = next
	}
	if !body.ok {
		var result renvoCompileResult
		return result
	}
	data := renvoWasiWasm32EmitBinary(p, meta, statements)
	if len(data) == 0 {
		var result renvoCompileResult
		return result
	}
	var result renvoCompileResult
	result.data = data
	result.ok = true
	return result
}

func renvoWasiWasm32EmitBinary(p *renvoProgram, meta *renvoMeta, statements []renvoStmt) []byte {
	dataOff := 1024
	exitCode := 0
	var code renvoWasmBuffer
	var data []byte
	var gen renvoLinearGen
	gen.prog = p
	gen.meta = meta
	for i := 0; i < len(statements); i++ {
		stmt := &statements[i]
		if stmt.kind == renvoStmtExpr {
			var ep renvoExprParse
			renvoParseExpressionInto(&ep, p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			rootIndex := len(ep.exprs) - 1
			root := &ep.exprs[rootIndex]
			if root.kind != renvoExprCall || root.argCount != 1 || !renvoExprIsIdentText(p, &ep, root.left, "print") {
				return nil
			}
			arg := &ep.exprs[ep.args[root.firstArg]]
			if arg.kind != renvoExprString {
				return nil
			}
			msg := renvoDecodeStringToken(p, arg.tok)
			msgOff := dataOff + len(data)
			for j := 0; j < len(msg); j++ {
				data = append(data, msg[j])
			}
			renvoWasiWasm32AppendPrint(&code, msgOff, len(msg))
			continue
		}
		if stmt.kind == renvoStmtReturn {
			if stmt.exprStart == stmt.exprEnd {
				exitCode = 0
				continue
			}
			var ep renvoExprParse
			renvoParseExpressionInto(&ep, p, stmt.exprStart, stmt.exprEnd)
			if !ep.ok || len(ep.exprs) == 0 {
				return nil
			}
			result := renvoEvalConstExpr(&gen, &ep, len(ep.exprs)-1)
			if !result.ok {
				return nil
			}
			exitCode = result.value
			continue
		}
		return nil
	}
	renvoWasmAppendI32Const(&code, exitCode)
	renvoWasmAppendCall(&code, 1)

	var out renvoWasmBuffer
	renvoWasmAppendEncoded(&out, "\x00\x61\x73\x6d\x01\x00\x00\x00")
	renvoWasmAppendSection(&out, 1, renvoWasiWasm32TypeSection())
	renvoWasmAppendSection(&out, 2, renvoWasiWasm32ImportSection())
	renvoWasmAppendSection(&out, 3, renvoWasiWasm32FunctionSection())
	renvoWasmAppendSection(&out, 5, renvoWasiWasm32MemorySection())
	renvoWasmAppendSection(&out, 7, renvoWasiWasm32ExportSection())
	renvoWasmAppendSection(&out, 10, renvoWasiWasm32CodeSection(code.data))
	renvoWasmAppendSection(&out, 11, renvoWasiWasm32DataSection(dataOff, data))
	return out.data[:out.length]
}

func renvoWasm32EmitWideBinaryStack(g *renvoLinearGen, dest int, left int, right int, tok int, signed bool) bool {
	renvoNonNil(g)
	if g.c.renvoTarget == renvoTargetVM32 {
		return renvoWasm32EmitWideBinaryPortable(g, dest, left, right, tok, signed)
	}
	if renvoTokStarts2(g.prog, tok, '&', '^') {
		ones := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		renvoAsmStoreStackImm(&g.asm, ones, -1)
		renvoAsmStoreStackImm(&g.asm, ones-g.c.renvoNativeIntSize, -1)
		renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, right, ones, 0x85)
		renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, left, dest, 0x83)
		return true
	}
	op := 0
	if renvoTokStartsWith(g.prog, tok, '+') {
		op = 0x7c
	} else if renvoTokStartsWith(g.prog, tok, '-') {
		op = 0x7d
	} else if renvoTokStartsWith(g.prog, tok, '*') {
		op = 0x7e
	} else if renvoTokStartsWith(g.prog, tok, '/') {
		op = 0x80
		if signed {
			op = 0x7f
		}
	} else if renvoTokStartsWith(g.prog, tok, '%') {
		op = 0x82
		if signed {
			op = 0x81
		}
	} else if renvoTokStartsWith(g.prog, tok, '&') {
		op = 0x83
	} else if renvoTokStartsWith(g.prog, tok, '|') {
		op = 0x84
	} else if renvoTokStartsWith(g.prog, tok, '^') {
		op = 0x85
	} else if renvoTokStarts2(g.prog, tok, '<', '<') {
		op = 0x86
	} else if renvoTokStarts2(g.prog, tok, '>', '>') {
		op = 0x88
		if signed {
			op = 0x87
		}
	}
	if op == 0 {
		return false
	}
	if op >= 0x7f && op <= 0x82 {
		// Keep division by zero on Renvo's panic path rather than allowing a
		// WebAssembly trap, so recover continues to work.
		nonzero := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, right-g.c.renvoNativeIntSize)
		renvoAsmJnzPrimary(&g.asm, nonzero)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoEmitRuntimeNonNilPrimary(g)
		renvoAsmMarkLabel(&g.asm, nonzero)
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, left, right, op)
	return true
}

func renvoWasm32EmitWideBinaryPortable(g *renvoLinearGen, dest int, left int, right int, tok int, signed bool) bool {
	if renvoTokStartsWith(g.prog, tok, '+') {
		renvoEmitWideAddStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '-') {
		renvoEmitWideSubStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '*') {
		renvoEmitWideMulStack(g, dest, left, right)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '/') || renvoTokStartsWith(g.prog, tok, '%') {
		renvoEmitWideDivStack(g, dest, left, right, signed, renvoTokStartsWith(g.prog, tok, '%'))
		return true
	}
	if renvoTokStarts2(g.prog, tok, '<', '<') || renvoTokStarts2(g.prog, tok, '>', '>') {
		renvoEmitWideShiftStack(g, dest, left, right, renvoTokStarts2(g.prog, tok, '>', '>'), signed)
		return true
	}
	if renvoTokStartsWith(g.prog, tok, '&') || renvoTokStartsWith(g.prog, tok, '|') || renvoTokStartsWith(g.prog, tok, '^') {
		for at := 0; at < renvoBackendValueSlotSize; at += g.c.renvoNativeIntSize {
			renvoAsmLoadPrimaryStack(&g.asm, right-at)
			renvoAsmLoadTertiaryStack(&g.asm, left-at)
			if !renvoEmitPrimaryTertiaryOp(g, tok) {
				return false
			}
			renvoAsmStorePrimaryStack(&g.asm, dest-at)
		}
		return true
	}
	return false
}

func renvoWasm32EmitWideCompareStack(g *renvoLinearGen, left int, right int, tok int, signed bool) bool {
	renvoNonNil(g)
	if g.c.renvoTarget == renvoTargetVM32 {
		return renvoWasm32EmitWideComparePortable(g, left, right, tok, signed)
	}
	p := g.prog
	op := 0
	if renvoTok2Is(p, tok, '=', '=') {
		op = 0x51
	} else if renvoTok2Is(p, tok, '!', '=') {
		op = 0x52
	} else if renvoTok2Is(p, tok, '<', '=') {
		op = 0x58
		if signed {
			op = 0x57
		}
	} else if renvoTok2Is(p, tok, '>', '=') {
		op = 0x5a
		if signed {
			op = 0x59
		}
	} else if renvoTokCharIs(p, tok, '<') {
		op = 0x54
		if signed {
			op = 0x53
		}
	} else if renvoTokCharIs(p, tok, '>') {
		op = 0x56
		if signed {
			op = 0x55
		}
	}
	if op == 0 {
		return false
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideCompare, 0, left, right, op)
	return true
}

func renvoWasm32SoftCallWideUnary(g *renvoLinearGen, dest int, source int, helperName string) bool {
	fnIndex := renvoFindMetaFunctionText(g.meta, helperName)
	if fnIndex < 0 {
		return false
	}
	wordCount := renvoEmitTypedLocalArgReverse(g, source, renvoBuiltinTypeUint64)
	renvoAsmAddressPrimaryStack(&g.asm, dest)
	renvoAsmPushPrimary(&g.asm)
	wordCount += renvoBackendHiddenResultWordCount
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoWasm32SoftCallScalarToWide(g *renvoLinearGen, dest int, source int, helperName string) bool {
	fnIndex := renvoFindMetaFunctionText(g.meta, helperName)
	if fnIndex < 0 {
		return false
	}
	wordCount := renvoEmitTypedLocalArgReverse(g, source, renvoBuiltinTypeUint32)
	renvoAsmAddressPrimaryStack(&g.asm, dest)
	renvoAsmPushPrimary(&g.asm)
	wordCount += renvoBackendHiddenResultWordCount
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoWasm32SoftCallWideBinary(g *renvoLinearGen, dest int, left int, right int, helperName string) bool {
	fnIndex := renvoFindMetaFunctionText(g.meta, helperName)
	if fnIndex < 0 {
		return false
	}
	wordCount := renvoEmitTypedLocalArgReverse(g, left, renvoBuiltinTypeUint64)
	wordCount += renvoEmitTypedLocalArgReverse(g, right, renvoBuiltinTypeUint64)
	renvoAsmAddressPrimaryStack(&g.asm, dest)
	renvoAsmPushPrimary(&g.asm)
	wordCount += renvoBackendHiddenResultWordCount
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoWasm32SoftCallBoolBinary(g *renvoLinearGen, left int, right int, helperName string) bool {
	fnIndex := renvoFindMetaFunctionText(g.meta, helperName)
	if fnIndex < 0 {
		return false
	}
	wordCount := renvoEmitTypedLocalArgReverse(g, right, renvoBuiltinTypeUint64)
	wordCount += renvoEmitTypedLocalArgReverse(g, left, renvoBuiltinTypeUint64)
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoWasm32SoftCallBoolWords(g *renvoLinearGen, left int, right int, helperName string) bool {
	fnIndex := renvoFindMetaFunctionText(g.meta, helperName)
	if fnIndex < 0 {
		return false
	}
	wordCount := renvoEmitTypedLocalArgReverse(g, left, renvoBuiltinTypeUint32)
	wordCount += renvoEmitTypedLocalArgReverse(g, left-4, renvoBuiltinTypeUint32)
	wordCount += renvoEmitTypedLocalArgReverse(g, right, renvoBuiltinTypeUint32)
	wordCount += renvoEmitTypedLocalArgReverse(g, right-4, renvoBuiltinTypeUint32)
	renvoEmitCallWithWordCount(g, fnIndex, wordCount)
	return true
}

func renvoWasm32NativeFloatBinaryStack(g *renvoLinearGen, dest int, left int, right int, op byte, size int) bool {
	wasmOp := 0
	if size == 4 {
		if op == '+' {
			wasmOp = 0x92 // f32.add
		} else if op == '-' {
			wasmOp = 0x93 // f32.sub
		} else if op == '*' {
			wasmOp = 0x94 // f32.mul
		} else if op == '/' {
			wasmOp = 0x95 // f32.div
		}
	} else if size == 8 {
		if op == '+' {
			wasmOp = 0xa0 // f64.add
		} else if op == '-' {
			wasmOp = 0xa1 // f64.sub
		} else if op == '*' {
			wasmOp = 0xa2 // f64.mul
		} else if op == '/' {
			wasmOp = 0xa3 // f64.div
		}
	}
	if wasmOp == 0 {
		return false
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, left, right, wasmOp)
	return true
}

func renvoWasm32NativeFloatCompareStack(g *renvoLinearGen, left int, right int, kind int, c0 byte, c1 byte) bool {
	wasmOp := 0
	if kind == renvoTypeFloat32 {
		wasmOp = 0x5b // f32.eq
	} else if kind == renvoTypeFloat64 {
		wasmOp = 0x61 // f64.eq
	} else {
		return false
	}
	if c0 == '!' {
		wasmOp++
	} else if c0 == '<' {
		wasmOp += 2
		if c1 == '=' {
			wasmOp += 2
		}
	} else if c0 == '>' {
		wasmOp += 3
		if c1 == '=' {
			wasmOp += 2
		}
	} else if c0 != '=' {
		return false
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideCompare, 0, left, right, wasmOp)
	return true
}

func renvoWasm32NativeFloatConvertStack(g *renvoLinearGen, dest int, source int, sourceSize int, destSize int) {
	if sourceSize == destSize {
		renvoEmitCopyStackToStack(g, source, dest, sourceSize)
		return
	}
	wasmOp := 0xbb // f64.promote_f32
	if sourceSize == 8 && destSize == 4 {
		wasmOp = 0xb6 // f32.demote_f64
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, source, source, wasmOp)
}

func renvoWasm32NativeIntToFloatStack(g *renvoLinearGen, offset int, intSize int, floatSize int, signed bool) {
	wasmOp := 0
	if floatSize == 4 {
		wasmOp = 0xb2 // f32.convert_i32_s
		if intSize == 8 {
			wasmOp = 0xb4 // f32.convert_i64_s
		}
	} else if floatSize == 8 {
		wasmOp = 0xb7 // f64.convert_i32_s
		if intSize == 8 {
			wasmOp = 0xb9 // f64.convert_i64_s
		}
	}
	if !signed {
		wasmOp++
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, offset, offset, offset, wasmOp)
}

func renvoWasm32NativeFloatToIntStack(g *renvoLinearGen, dest int, source int, floatSize int, intSize int, signed bool) {
	wasmOp := 0
	if floatSize == 4 {
		wasmOp = 0xa8 // i32.trunc_f32_s
		if intSize == 8 {
			wasmOp = 0xae // i64.trunc_f32_s
		}
	} else if floatSize == 8 {
		wasmOp = 0xaa // i32.trunc_f64_s
		if intSize == 8 {
			wasmOp = 0xb0 // i64.trunc_f64_s
		}
	}
	if !signed {
		wasmOp++
	}
	renvoWasm32EmitWideOp(&g.asm, renvoWasm32OpWideBinary, dest, source, source, wasmOp)
}

func renvoWasm32SoftFloatBinaryStack(g *renvoLinearGen, dest int, left int, right int, op byte, size int) bool {
	if g.c.renvoTarget == renvoTargetWasiWasm32 {
		return renvoWasm32NativeFloatBinaryStack(g, dest, left, right, op, size)
	}
	if size != 4 && size != 8 {
		return false
	}
	helper := ""
	if size == 4 {
		if op == '+' {
			helper = "__renvoSoftAdd32Wide"
		} else if op == '-' {
			helper = "__renvoSoftSub32Wide"
		} else if op == '*' {
			helper = "__renvoSoftMul32Wide"
		} else if op == '/' {
			helper = "__renvoSoftDiv32Wide"
		}
	} else {
		if op == '+' {
			helper = "__renvoSoftAdd64"
		} else if op == '-' {
			helper = "__renvoSoftSub64"
		} else if op == '*' {
			helper = "__renvoSoftMul64"
		} else if op == '/' {
			helper = "__renvoSoftDiv64"
		}
	}
	if helper == "" {
		return false
	}
	if size == 4 {
		wideLeft := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		wideRight := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		wideResult := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoAsmStorePrimaryStack(&g.asm, wideLeft)
		renvoAsmStoreStackImm(&g.asm, wideLeft-4, 0)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoAsmStorePrimaryStack(&g.asm, wideRight)
		renvoAsmStoreStackImm(&g.asm, wideRight-4, 0)
		if !renvoWasm32SoftCallWideBinary(g, wideResult, wideLeft, wideRight, helper) {
			return false
		}
		renvoAsmLoadPrimaryStack(&g.asm, wideResult)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		return true
	}
	return renvoWasm32SoftCallWideBinary(g, dest, left, right, helper)
}

func renvoWasm32SoftFloatCompareStack(g *renvoLinearGen, left int, right int, kind int, c0 byte, c1 byte) bool {
	if g.c.renvoTarget == renvoTargetWasiWasm32 {
		return renvoWasm32NativeFloatCompareStack(g, left, right, kind, c0, c1)
	}
	if kind == renvoTypeFloat32 {
		return renvoWasm32SoftFloat32CompareInline(g, left, right, c0, c1)
	}
	if kind != renvoTypeFloat64 {
		return false
	}
	return renvoWasm32SoftFloatCompareInline(g, left, right, c0, c1)
}

func renvoWasm32SoftFloat32CompareInline(g *renvoLinearGen, left int, right int, c0 byte, c1 byte) bool {
	leftAbs := renvoAddUnnamedLocal(g, renvoTypeInt)
	rightAbs := renvoAddUnnamedLocal(g, renvoTypeInt)
	for i := 0; i < 2; i++ {
		source := left
		dest := leftAbs
		if i != 0 {
			source = right
			dest = rightAbs
		}
		renvoAsmLoadPrimaryStack(&g.asm, source)
		renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, 0x7fffffff)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest)
	}

	nan := renvoAsmNewLabel(&g.asm)
	zeroPair := renvoAsmNewLabel(&g.asm)
	compare := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoAsmJcmpStackImm(&g.asm, leftAbs, 0x7f800000, nan, 0x9f)
	renvoAsmJcmpStackImm(&g.asm, rightAbs, 0x7f800000, nan, 0x9f)
	leftNonzero := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, leftAbs)
	renvoAsmJnzPrimary(&g.asm, leftNonzero)
	renvoAsmLoadPrimaryStack(&g.asm, rightAbs)
	renvoAsmJzPrimary(&g.asm, zeroPair)
	renvoAsmMarkLabel(&g.asm, leftNonzero)
	renvoAsmJmpLabel(&g.asm, compare)

	renvoAsmMarkLabel(&g.asm, compare)
	if c0 == '=' || c0 == '!' {
		renvoEmitNativeCompareStack(g, left, right, 0x94)
		if c0 == '!' {
			renvoAsmBoolNotPrimary(&g.asm)
		}
		renvoAsmJmpLabel(&g.asm, done)
	} else {
		leftSign := renvoAddUnnamedLocal(g, renvoTypeInt)
		rightSign := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoAsmShrPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, leftSign)
		renvoAsmLoadPrimaryStack(&g.asm, right)
		renvoAsmShrPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, rightSign)
		leftNegative := renvoAsmNewLabel(&g.asm)
		bothPositive := renvoAsmNewLabel(&g.asm)
		bothNegative := renvoAsmNewLabel(&g.asm)
		relationalDone := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, leftSign)
		renvoAsmJnzPrimary(&g.asm, leftNegative)
		renvoAsmLoadPrimaryStack(&g.asm, rightSign)
		renvoAsmJzPrimary(&g.asm, bothPositive)
		result := 0
		if c0 == '>' {
			result = 1
		}
		renvoAsmPrimaryImm(&g.asm, result)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, bothPositive)
		setcc := renvoFloat32RelationSetcc(c0, c1)
		renvoEmitNativeCompareStack(g, left, right, setcc)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, leftNegative)
		renvoAsmLoadPrimaryStack(&g.asm, rightSign)
		renvoAsmJnzPrimary(&g.asm, bothNegative)
		result = 0
		if c0 == '<' {
			result = 1
		}
		renvoAsmPrimaryImm(&g.asm, result)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, bothNegative)
		renvoEmitNativeCompareStack(g, right, left, setcc)
		renvoAsmMarkLabel(&g.asm, relationalDone)
		renvoAsmJmpLabel(&g.asm, done)
	}

	renvoAsmMarkLabel(&g.asm, nan)
	result := 0
	if c0 == '!' {
		result = 1
	}
	renvoAsmPrimaryImm(&g.asm, result)
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, zeroPair)
	result = 0
	if c0 == '=' || c0 == '<' && c1 == '=' || c0 == '>' && c1 == '=' {
		result = 1
	}
	renvoAsmPrimaryImm(&g.asm, result)
	renvoAsmMarkLabel(&g.asm, done)
	return true
}

func renvoFloat32RelationSetcc(c0 byte, c1 byte) int {
	if c0 == '<' {
		if c1 == '=' {
			return 0x96
		}
		return 0x92
	}
	if c1 == '=' {
		return 0x93
	}
	return 0x97
}

func renvoWasm32SoftConvertFloatStack(g *renvoLinearGen, dest int, source int, sourceSize int, destSize int) {
	if g.c.renvoTarget == renvoTargetWasiWasm32 {
		renvoWasm32NativeFloatConvertStack(g, dest, source, sourceSize, destSize)
		return
	}
	if sourceSize == destSize {
		renvoEmitCopyStackToStack(g, source, dest, sourceSize)
		return
	}
	if sourceSize == 4 && destSize == 8 {
		wideSource := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		renvoAsmLoadPrimaryStack(&g.asm, source)
		renvoAsmStorePrimaryStack(&g.asm, wideSource)
		renvoAsmStoreStackImm(&g.asm, wideSource-4, 0)
		renvoWasm32SoftCallWideUnary(g, dest, wideSource, "__renvoSoft32WideTo64")
		return
	}
	if sourceSize == 8 && destSize == 4 {
		wideResult := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		if renvoWasm32SoftCallWideUnary(g, wideResult, source, "__renvoSoft64To32Wide") {
			renvoAsmLoadPrimaryStack(&g.asm, wideResult)
			renvoAsmStorePrimaryStack(&g.asm, dest)
		}
	}
}

func renvoWasm32SoftIntToFloatStack(g *renvoLinearGen, offset int, intSize int, floatSize int, signed bool) {
	if g.c.renvoTarget == renvoTargetWasiWasm32 {
		renvoWasm32NativeIntToFloatStack(g, offset, intSize, floatSize, signed)
		return
	}
	if floatSize != 4 && floatSize != 8 {
		return
	}
	if intSize < 8 {
		if signed {
			renvoAsmLoadPrimaryStack(&g.asm, offset)
			renvoAsmSarPrimaryImm(&g.asm, 31)
			renvoAsmStorePrimaryStack(&g.asm, offset-4)
		} else {
			renvoAsmStoreStackImm(&g.asm, offset-4, 0)
		}
	}
	result := renvoAddUnnamedLocal(g, renvoTypeFloat64)
	helper := "__renvoSoftInt64To64"
	if !signed {
		helper = "__renvoSoftUint64To64"
	}
	if floatSize == 4 {
		helper = "__renvoSoftInt64To32Wide"
		if !signed {
			helper = "__renvoSoftUint64To32Wide"
		}
	}
	if renvoWasm32SoftCallWideUnary(g, result, offset, helper) {
		renvoEmitCopyStackToStack(g, result, offset, floatSize)
	}
}

func renvoWasm32SoftFloatToIntStack(g *renvoLinearGen, dest int, source int, floatSize int, intSize int, signed bool) {
	if g.c.renvoTarget == renvoTargetWasiWasm32 {
		renvoWasm32NativeFloatToIntStack(g, dest, source, floatSize, intSize, signed)
		return
	}
	wideSource := source
	if floatSize == 4 {
		bits := renvoAddUnnamedLocal(g, renvoBuiltinTypeUint64)
		renvoAsmLoadPrimaryStack(&g.asm, source)
		renvoAsmStorePrimaryStack(&g.asm, bits)
		renvoAsmStoreStackImm(&g.asm, bits-4, 0)
		wideSource = bits
	}
	result := renvoAddUnnamedLocal(g, renvoTypeInt64)
	helper := "__renvoSoft64ToInt64"
	if !signed {
		helper = "__renvoSoft64ToUint64"
	}
	if floatSize == 4 {
		helper = "__renvoSoft32WideToInt64"
		if !signed {
			helper = "__renvoSoft32WideToUint64"
		}
	}
	if renvoWasm32SoftCallWideUnary(g, result, wideSource, helper) {
		renvoEmitCopyStackToStack(g, result, dest, intSize)
	}
}

func renvoWasm32SoftNegateStack(g *renvoLinearGen, offset int, size int) {
	highOffset := offset
	if size == 8 {
		highOffset -= 4
	}
	renvoAsmLoadPrimaryStack(&g.asm, highOffset)
	renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, -2147483648)
	renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryStack(&g.asm, highOffset)
}

func renvoWasm32SoftFloatAbsStack(g *renvoLinearGen, dest int, source int) {
	renvoEmitCopyStackToStack(g, source, dest, 8)
	renvoAsmLoadPrimaryStack(&g.asm, dest-4)
	renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, 0x7fffffff)
	renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryStack(&g.asm, dest-4)
}

func renvoWasm32SoftFloatOrderKeyStack(g *renvoLinearGen, dest int, source int) {
	renvoEmitCopyStackToStack(g, source, dest, 8)
	negative := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoAsmJcmpStackImm(&g.asm, source-4, 0, negative, 0x9c)
	renvoAsmLoadPrimaryStack(&g.asm, dest-4)
	renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, -2147483648)
	renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpXorRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
	renvoAsmStorePrimaryStack(&g.asm, dest-4)
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, negative)
	renvoAsmLoadPrimaryStack(&g.asm, dest)
	renvoAsmBitwiseNotPrimary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, dest)
	renvoAsmLoadPrimaryStack(&g.asm, dest-4)
	renvoAsmBitwiseNotPrimary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, dest-4)
	renvoAsmMarkLabel(&g.asm, done)
}

func renvoWasm32SoftFloatCompareInline(g *renvoLinearGen, left int, right int, c0 byte, c1 byte) bool {
	leftHigh := renvoAddUnnamedLocal(g, renvoTypeInt)
	rightHigh := renvoAddUnnamedLocal(g, renvoTypeInt)
	for i := 0; i < 2; i++ {
		source := left
		dest := leftHigh
		if i != 0 {
			source = right
			dest = rightHigh
		}
		renvoAsmLoadPrimaryStack(&g.asm, source-4)
		renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpMovRegImm, renvoWasm32RegRcx, 0x7fffffff)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpAndRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest)
	}

	nan := renvoAsmNewLabel(&g.asm)
	zeroPair := renvoAsmNewLabel(&g.asm)
	compare := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	for i := 0; i < 2; i++ {
		high := leftHigh
		low := left
		if i != 0 {
			high = rightHigh
			low = right
		}
		notMaxExponent := renvoAsmNewLabel(&g.asm)
		renvoAsmJcmpStackImm(&g.asm, high, 0x7ff00000, nan, 0x9f)
		renvoAsmJcmpStackImm(&g.asm, high, 0x7ff00000, notMaxExponent, 0x95)
		renvoAsmLoadPrimaryStack(&g.asm, low)
		renvoAsmJnzPrimary(&g.asm, nan)
		renvoAsmMarkLabel(&g.asm, notMaxExponent)
	}

	leftNonzero := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, leftHigh)
	renvoAsmJnzPrimary(&g.asm, leftNonzero)
	renvoAsmLoadPrimaryStack(&g.asm, left)
	renvoAsmJnzPrimary(&g.asm, leftNonzero)
	renvoAsmLoadPrimaryStack(&g.asm, rightHigh)
	renvoAsmJnzPrimary(&g.asm, compare)
	renvoAsmLoadPrimaryStack(&g.asm, right)
	renvoAsmJzPrimary(&g.asm, zeroPair)
	renvoAsmMarkLabel(&g.asm, leftNonzero)
	renvoAsmJmpLabel(&g.asm, compare)

	renvoAsmMarkLabel(&g.asm, compare)
	if c0 == '=' || c0 == '!' {
		renvoWasm32SoftFloatRawEquality(g, left, right, c0 == '!')
		renvoAsmJmpLabel(&g.asm, done)
	} else {
		leftSign := renvoAddUnnamedLocal(g, renvoTypeInt)
		rightSign := renvoAddUnnamedLocal(g, renvoTypeInt)
		renvoAsmLoadPrimaryStack(&g.asm, left-4)
		renvoAsmShrPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, leftSign)
		renvoAsmLoadPrimaryStack(&g.asm, right-4)
		renvoAsmShrPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, rightSign)
		leftNegative := renvoAsmNewLabel(&g.asm)
		bothPositive := renvoAsmNewLabel(&g.asm)
		bothNegative := renvoAsmNewLabel(&g.asm)
		relationalDone := renvoAsmNewLabel(&g.asm)
		renvoAsmLoadPrimaryStack(&g.asm, leftSign)
		renvoAsmJnzPrimary(&g.asm, leftNegative)
		renvoAsmLoadPrimaryStack(&g.asm, rightSign)
		renvoAsmJzPrimary(&g.asm, bothPositive)
		result := 0
		if c0 == '>' {
			result = 1
		}
		renvoAsmPrimaryImm(&g.asm, result)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, bothPositive)
		renvoWasm32SoftFloatRawRelation(g, left, right, c0, c1)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, leftNegative)
		renvoAsmLoadPrimaryStack(&g.asm, rightSign)
		renvoAsmJnzPrimary(&g.asm, bothNegative)
		result = 0
		if c0 == '<' {
			result = 1
		}
		renvoAsmPrimaryImm(&g.asm, result)
		renvoAsmJmpLabel(&g.asm, relationalDone)
		renvoAsmMarkLabel(&g.asm, bothNegative)
		renvoWasm32SoftFloatRawRelation(g, right, left, c0, c1)
		renvoAsmMarkLabel(&g.asm, relationalDone)
		renvoAsmJmpLabel(&g.asm, done)
	}

	renvoAsmMarkLabel(&g.asm, nan)
	result := 0
	if c0 == '!' {
		result = 1
	}
	renvoAsmPrimaryImm(&g.asm, result)
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, zeroPair)
	result = 0
	if c0 == '=' || c0 == '<' && c1 == '=' || c0 == '>' && c1 == '=' {
		result = 1
	}
	renvoAsmPrimaryImm(&g.asm, result)
	renvoAsmMarkLabel(&g.asm, done)
	return true
}

func renvoWasm32SoftFloatRawEquality(g *renvoLinearGen, left int, right int, notEqualResult bool) {
	notEqual := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoEmitNativeCompareStack(g, left-4, right-4, 0x94)
	renvoAsmJzPrimary(&g.asm, notEqual)
	renvoEmitNativeCompareStack(g, left, right, 0x94)
	renvoAsmJmpMarkLabel(&g.asm, done, notEqual)
	renvoAsmPrimaryImm(&g.asm, 0)
	renvoAsmMarkLabel(&g.asm, done)
	if notEqualResult {
		renvoAsmBoolNotPrimary(&g.asm)
	}
}

func renvoWasm32SoftFloatRawRelation(g *renvoLinearGen, left int, right int, c0 byte, c1 byte) {
	if c0 == '<' && c1 == '=' {
		renvoEmitWideLessStack(g, right, left, false)
		renvoAsmBoolNotPrimary(&g.asm)
	} else if c0 == '<' {
		renvoEmitWideLessStack(g, left, right, false)
	} else if c0 == '>' && c1 == '=' {
		renvoEmitWideLessStack(g, left, right, false)
		renvoAsmBoolNotPrimary(&g.asm)
	} else {
		renvoEmitWideLessStack(g, right, left, false)
	}
}

func renvoEmitVM32WideShiftBy(g *renvoLinearGen, count int, opcode int) {
	renvoAsmLoadTertiaryStack(&g.asm, count)
	renvoWasm32EmitRegReg(&g.asm, opcode, renvoWasm32RegRax, renvoWasm32RegRcx)
}

func renvoEmitVM32WideShiftStack(g *renvoLinearGen, dest int, left int, count int, right bool, signed bool) {
	renvoEmitCopyStackToStack(g, left, dest, renvoBackendValueSlotSize)
	large := renvoAsmNewLabel(&g.asm)
	atLeastWord := renvoAsmNewLabel(&g.asm)
	done := renvoAsmNewLabel(&g.asm)
	renvoAsmLoadPrimaryStack(&g.asm, count-g.c.renvoNativeIntSize)
	renvoAsmJnzPrimary(&g.asm, large)
	// A count whose low word has its sign bit set is necessarily at least 2^31.
	renvoAsmJcmpStackImm(&g.asm, count, 0, large, 0x9c)
	renvoAsmJcmpStackImm(&g.asm, count, 32, atLeastWord, 0x9d)
	renvoAsmLoadPrimaryStack(&g.asm, count)
	renvoAsmJzPrimary(&g.asm, done)
	inverse := renvoAddUnnamedLocal(g, renvoTypeInt)
	renvoAsmPrimaryImm(&g.asm, 32)
	renvoAsmLoadTertiaryStack(&g.asm, count)
	renvoAsmSubPrimaryTertiary(&g.asm)
	renvoAsmStorePrimaryStack(&g.asm, inverse)
	if right {
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShrUnsignedRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShlRegReg)
		renvoAsmLoadTertiaryStack(&g.asm, dest)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		opcode := renvoWasm32OpShrUnsignedRegReg
		if signed {
			opcode = renvoWasm32OpShrRegReg
		}
		renvoEmitVM32WideShiftBy(g, count, opcode)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
	} else {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShrUnsignedRegReg)
		renvoAsmLoadTertiaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoWasm32EmitRegReg(&g.asm, renvoWasm32OpOrRegReg, renvoWasm32RegRax, renvoWasm32RegRcx)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, count, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest)
	}
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, atLeastWord)
	renvoAsmJcmpStackImm(&g.asm, count, 64, large, 0x9d)
	renvoAsmLoadPrimaryStack(&g.asm, count)
	renvoWasm32EmitRegImm(&g.asm, renvoWasm32OpAddRegImm, renvoWasm32RegRax, -32)
	renvoAsmStorePrimaryStack(&g.asm, inverse)
	if right {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		opcode := renvoWasm32OpShrUnsignedRegReg
		if signed {
			opcode = renvoWasm32OpShrRegReg
		}
		renvoEmitVM32WideShiftBy(g, inverse, opcode)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		if signed {
			renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
			renvoAsmSarPrimaryImm(&g.asm, 31)
			renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		} else {
			renvoAsmStoreStackImm(&g.asm, dest-g.c.renvoNativeIntSize, 0)
		}
	} else {
		renvoAsmLoadPrimaryStack(&g.asm, left)
		renvoEmitVM32WideShiftBy(g, inverse, renvoWasm32OpShlRegReg)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
		renvoAsmStoreStackImm(&g.asm, dest, 0)
	}
	renvoAsmJmpLabel(&g.asm, done)
	renvoAsmMarkLabel(&g.asm, large)
	if right && signed {
		renvoAsmLoadPrimaryStack(&g.asm, left-g.c.renvoNativeIntSize)
		renvoAsmSarPrimaryImm(&g.asm, 31)
		renvoAsmStorePrimaryStack(&g.asm, dest)
		renvoAsmStorePrimaryStack(&g.asm, dest-g.c.renvoNativeIntSize)
	} else {
		renvoAsmStoreStackImm(&g.asm, dest, 0)
		renvoAsmStoreStackImm(&g.asm, dest-g.c.renvoNativeIntSize, 0)
	}
	renvoAsmMarkLabel(&g.asm, done)
}

func renvoWasm32EmitWideComparePortable(g *renvoLinearGen, left int, right int, tok int, signed bool) bool {
	return renvoEmitPortableWideCompareStack(g, left, right, tok, signed)
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

func renvoWasm32RangesOverlap(aOffset int, aSize int, bOffset int, bSize int) bool {
	aStart := aOffset - aSize
	bStart := bOffset - bSize
	return aStart < bOffset && bStart < aOffset
}

func renvoWasm32RecordDirectLocals(g *renvoLinearGen, functionPC int) {
	a := &g.asm
	candidates := make([]int, 0, g.localCount)
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		if local.size != renvoBackendValueSlotSize || local.captureOff != 0 || renvoTypeSize(g.meta, local.typ) > g.c.renvoNativeIntSize {
			continue
		}
		found := false
		for j := 0; j < len(candidates); j++ {
			if candidates[j] == local.offset {
				found = true
				break
			}
		}
		if !found {
			candidates = append(candidates, local.offset)
		}
	}
	for i := 0; i < g.localCount; i++ {
		local := &g.locals[i]
		direct := local.size == renvoBackendValueSlotSize && local.captureOff == 0 && renvoTypeSize(g.meta, local.typ) <= g.c.renvoNativeIntSize
		for j := 0; j < len(candidates); j++ {
			candidate := candidates[j]
			if candidate != 0 && renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, local.offset, local.size) && (!direct || candidate != local.offset) {
				candidates[j] = 0
			}
			if candidate != 0 && local.captureOff > 0 && renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, local.captureOff, renvoBackendValueSlotSize) {
				candidates[j] = 0
			}
		}
	}
	for pc := functionPC; pc < len(a.code); pc += int(renvoWasm32InstructionSizes[int(renvo_runtime_UnsafeByteAt(a.code, pc))]) {
		op := int(renvo_runtime_UnsafeByteAt(a.code, pc))
		// Wide operations read and write whole frame-backed slots. Stack slots
		// are reused across expression lifetimes, so a scalar local at the same
		// offset cannot be proven coherent with those accesses. Keep the routine
		// frame-backed when either operation is present.
		if op == renvoWasm32OpWideBinary || op == renvoWasm32OpWideCompare {
			for j := 0; j < len(candidates); j++ {
				candidates[j] = 0
			}
			continue
		}
		memoryOffsets := make([]int, 0, 3)
		memorySizes := make([]int, 0, 3)
		if op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack {
			memoryOffsets = append(memoryOffsets, renvoWasm32GetS32(a.code, pc+2))
			memorySizes = append(memorySizes, g.c.renvoNativeIntSize)
		} else if op == renvoWasm32OpLeaStack {
			memoryOffsets = append(memoryOffsets, renvoWasm32GetS32(a.code, pc+2))
			memorySizes = append(memorySizes, renvoBackendValueSlotSize)
		}
		for k := 0; k < len(memoryOffsets); k++ {
			for j := 0; j < len(candidates); j++ {
				candidate := candidates[j]
				if candidate == 0 {
					continue
				}
				directScalarAccess := (op == renvoWasm32OpLoadStack || op == renvoWasm32OpStoreStack) && memoryOffsets[k] == candidate
				if !directScalarAccess {
					if renvoWasm32RangesOverlap(candidate, renvoBackendValueSlotSize, memoryOffsets[k], memorySizes[k]) {
						candidates[j] = 0
					}
				}
			}
		}
	}
	recordStart := len(a.wasmLocalSlots)
	a.wasmLocalSlots = append(a.wasmLocalSlots, int32(functionPC), 0)
	for i := 0; i < len(candidates); i++ {
		if candidates[i] != 0 {
			a.wasmLocalSlots = append(a.wasmLocalSlots, int32(candidates[i]))
		}
	}
	a.wasmLocalSlots[recordStart+1] = int32(len(a.wasmLocalSlots) - recordStart - 2)
}

func renvoWasm32EmitScalarFunction(g *renvoLinearGen, fnInfoIndex int) bool {
	a := &g.asm
	metaFn := &g.meta.funcs[fnInfoIndex]
	fn := &g.prog.funcs[metaFn.declIndex]
	g.locals = make([]renvoLocalInfo, renvoFunctionLocalCap(fn))
	g.localCount = 0
	g.gotoLabels = nil
	g.breakDepth = 0
	g.continueDepth = 0
	g.pendingControl = 0
	g.currentFunc = fnInfoIndex
	g.returnStruct = 0
	g.closureEnvOffset = 0
	g.deferHeadOffset = 0
	g.deferReturnLabel = 0
	g.deferResultOffset = 0
	g.deferSites = nil
	g.emittingDefers = false
	g.suppressPanicCheck = false
	g.stackUsed = 0
	g.stackPeak = 0
	g.lastRangeReturns = false
	functionPC := len(a.code)
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
	renvoWasm32RecordDirectLocals(g, functionPC)
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

func renvoWasm32EmitRaxRcxOp(g *renvoLinearGen, tok int, unsigned bool) bool {
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
			opcode := renvoWasm32OpShrRegReg
			if unsigned {
				opcode = renvoWasm32OpShrUnsignedRegReg
			}
			renvoWasm32EmitRegReg(a, opcode, renvoWasm32RegRax, renvoWasm32RegRdx)
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
	if g.meta.panicEnabled {
		allocOKLabel := renvoAsmNewLabel(a)
		renvoAsmJnzPrimary(a, allocOKLabel)
		renvoAsmJmpLabel(a, returnLabel)
		renvoAsmMarkLabel(a, allocOKLabel)
	}
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
