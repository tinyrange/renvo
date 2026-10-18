package runtime

// Integer lowering uses only WASM integer instructions. In particular division
// guards the host traps so the IR remains total at zero and MIN/-1.
func (w *wasmCode) word(local int, width uint64) {
	w.get(local)
	if width < 64 {
		w.i64(^uint64(0) >> (64 - width))
		w.byte(0x83)
	}
}
func (w *wasmCode) signWord(local int, width uint64) {
	w.word(local, width)
	if width < 64 {
		w.i64(64 - width)
		w.byte(0x86)
		w.i64(64 - width)
		w.byte(0x87)
	}
}
func (w *wasmCode) bit(local int, shift uint64) {
	w.get(local)
	w.i64(shift)
	w.byte(0x88)
	w.i64(1)
	w.byte(0x83)
}
func (w *wasmCode) parity(local int) {
	w.word(local, 8)
	w.byte(0x7b)
	w.i64(1)
	w.byte(0x83)
}
func (w *wasmCode) status(local int, width uint64) {
	w.parity(local)
	w.i64(1)
	w.byte(0x85)
	w.i64(2)
	w.byte(0x86)
	w.get(local)
	w.byte(0x50)
	w.booleanBit(6)
	w.byte(0x84)
	w.bit(local, width-1)
	w.i64(7)
	w.byte(0x86)
	w.byte(0x84)
}
func (w *wasmCode) integer(o Op, a, b, scratch int) bool {
	x, y, r, t, u, v := scratch, scratch+1, scratch+2, scratch+3, scratch+4, scratch+5
	width := o.Imm & 127
	switch o.Kind {
	case ByteParity:
		w.parity(a)
	case StatusBits:
		w.word(a, width)
		w.set(r)
		w.status(r, width)
	case ArithmeticStatus, CarryArithmetic, CarryFlags:
		w.word(a, width)
		w.set(x)
		w.word(b, width)
		w.set(y)
		sub := o.Imm&128 != 0
		w.binary(x, y, 0x7c)
		if sub {
			w.byte(0x1a)
			w.binary(x, y, 0x7d)
		}
		w.set(t)
		if o.Kind != ArithmeticStatus {
			w.word(int(o.Imm>>8)+2, 1)
			w.set(u)
			w.get(t)
			if sub {
				w.i64(1)
				w.get(u)
				w.byte(0x7d)
				w.byte(0x7d)
			} else {
				w.get(u)
				w.byte(0x7c)
			}
		} else {
			w.get(t)
		}
		if width < 64 {
			w.i64(^uint64(0) >> (64 - width))
			w.byte(0x83)
		}
		w.set(r)
		if o.Kind == CarryArithmetic {
			w.get(r)
			break
		}
		if o.Kind == ArithmeticStatus {
			w.status(r, width)
		} else {
			w.bit(r, width-1)
			w.i64(31)
			w.byte(0x86)
			w.get(r)
			w.byte(0x50)
			w.booleanBit(30)
			w.byte(0x84)
		}
		if sub {
			if o.Kind == ArithmeticStatus {
				w.binary(x, y, 0x54)
			} else {
				w.binary(x, y, 0x5a)
				w.binary(x, y, 0x51)
				w.byte(0x45)
				w.get(u)
				w.byte(0xa7)
				w.byte(0x72)
				w.byte(0x71)
			}
		} else if width < 64 && o.Kind != ArithmeticStatus {
			w.get(t)
			w.get(u)
			w.byte(0x7c)
			w.i64(width)
			w.byte(0x88)
			w.byte(0x50)
			w.byte(0x45)
		} else {
			if o.Kind == ArithmeticStatus {
				w.binary(r, x, 0x54)
			} else {
				w.binary(r, t, 0x54)
				w.binary(t, x, 0x54)
				w.byte(0x72)
			}
		}
		if o.Kind == ArithmeticStatus {
			w.byte(0xad)
		} else {
			w.booleanBit(29)
		}
		w.byte(0x84)
		if o.Kind == ArithmeticStatus {
			w.binary(x, y, 0x85)
			w.get(r)
			w.byte(0x85)
			w.i64(16)
			w.byte(0x83)
			w.byte(0x84)
		}
		w.binary(x, y, 0x85)
		if !sub {
			w.i64(^uint64(0))
			w.byte(0x85)
		}
		w.binary(x, r, 0x85)
		w.byte(0x83)
		w.i64(width - 1)
		w.byte(0x88)
		w.i64(1)
		w.byte(0x83)
		shift := uint64(28)
		if o.Kind == ArithmeticStatus {
			shift = 11
		}
		w.i64(shift)
		w.byte(0x86)
		w.byte(0x84)
	case LeadingZeros, LeadingSigns:
		w.word(a, width)
		w.set(x)
		if o.Kind == LeadingSigns {
			w.get(x)
			w.get(x)
			w.i64(^uint64(0))
			w.byte(0x85)
			w.bit(x, width-1)
			w.byte(0x50)
			w.byte(0x1b)
			w.set(x)
		}
		w.word(x, width)
		w.byte(0x79)
		if width == 32 {
			w.i64(32)
			w.byte(0x7d)
		}
		if o.Kind == LeadingSigns {
			w.i64(1)
			w.byte(0x7d)
		}
	case ReverseBits, ReverseBytes:
		width = o.Imm & 255
		w.i64(0)
		w.set(r)
		step, group := uint64(1), width
		if o.Kind == ReverseBytes {
			step = 8
			group = o.Imm >> 8
		}
		for offset := uint64(0); offset < width; offset += group {
			for j := uint64(0); j < group; j += step {
				w.get(r)
				w.get(a)
				w.i64(offset + j)
				w.byte(0x88)
				w.i64((uint64(1) << step) - 1)
				w.byte(0x83)
				w.i64(offset + group - step - j)
				w.byte(0x86)
				w.byte(0x84)
				w.set(r)
			}
		}
		w.get(r)
	case UnsignedMulHigh, SignedMulHigh:
		w.word(a, 32)
		w.word(b, 32)
		w.byte(0x7e)
		w.i64(32)
		w.byte(0x88)
		w.set(t)
		w.get(a)
		w.i64(32)
		w.byte(0x88)
		w.word(b, 32)
		w.byte(0x7e)
		w.set(u)
		w.word(a, 32)
		w.get(b)
		w.i64(32)
		w.byte(0x88)
		w.byte(0x7e)
		w.set(v)
		w.get(t)
		w.word(u, 32)
		w.byte(0x7c)
		w.word(v, 32)
		w.byte(0x7c)
		w.i64(32)
		w.byte(0x88)
		w.set(r)
		w.get(a)
		w.i64(32)
		w.byte(0x88)
		w.get(b)
		w.i64(32)
		w.byte(0x88)
		w.byte(0x7e)
		w.get(u)
		w.i64(32)
		w.byte(0x88)
		w.byte(0x7c)
		w.get(v)
		w.i64(32)
		w.byte(0x88)
		w.byte(0x7c)
		w.get(r)
		w.byte(0x7c)
		if o.Kind == SignedMulHigh {
			w.get(b)
			w.i64(0)
			w.get(a)
			w.i64(0)
			w.byte(0x53)
			w.byte(0x1b)
			w.byte(0x7d)
			w.get(a)
			w.i64(0)
			w.get(b)
			w.i64(0)
			w.byte(0x53)
			w.byte(0x1b)
			w.byte(0x7d)
		}
	case UnsignedDivide, SignedDivide, UnsignedRemainder, SignedRemainder:
		signed := o.Kind == SignedDivide || o.Kind == SignedRemainder
		rem := o.Kind == UnsignedRemainder || o.Kind == SignedRemainder
		if signed {
			w.signWord(a, width)
			w.set(x)
			w.signWord(b, width)
			w.set(y)
		} else {
			w.word(a, width)
			w.set(x)
			w.word(b, width)
			w.set(y)
		}
		w.get(y)
		w.byte(0x50)
		w.byte(0x04)
		w.byte(0x7e)
		if rem {
			w.get(x)
		} else {
			w.i64(^uint64(0))
		}
		w.byte(0x05)
		if signed {
			w.get(x)
			w.i64(uint64(1) << 63)
			w.byte(0x51)
			w.get(y)
			w.i64(^uint64(0))
			w.byte(0x51)
			w.byte(0x71)
			w.byte(0x04)
			w.byte(0x7e)
			if rem {
				w.i64(0)
			} else {
				w.get(x)
			}
			w.byte(0x05)
		}
		code := byte(0x80)
		if signed {
			code = 0x7f
		}
		if rem {
			code += 2
		}
		w.binary(x, y, code)
		if signed {
			w.byte(0x0b)
		}
		w.byte(0x0b)
		if width < 64 {
			w.i64(^uint64(0) >> (64 - width))
			w.byte(0x83)
		}
	default:
		return false
	}
	return true
}
