package runtime

// ArithmeticStatus returns CF/PF/AF/ZF/SF/OF at bits 0/2/4/6/7/11.
// Width is 8/16/32/64, with bit 7 selecting subtraction (borrow polarity).
const ArithmeticStatus = 44

func arithmeticStatus(a, b, encoding uint64) uint64 {
	width := encoding & 127
	mask := ^uint64(0) >> (64 - width)
	a &= mask
	b &= mask
	result := (a + b) & mask
	carry, overflow := result < a, ^(a^b)&(a^result)&(uint64(1)<<(width-1)) != 0
	if encoding&128 != 0 {
		result = (a - b) & mask
		carry = a < b
		overflow = (a^b)&(a^result)&(uint64(1)<<(width-1)) != 0
	}
	flags := statusBits(result, width) | ((a ^ b ^ result) & 16)
	if carry {
		flags |= 1
	}
	if overflow {
		flags |= 2048
	}
	return flags
}
func (b *Builder) ArithmeticStatus(a, rhs Value, width int, sub bool) Value {
	encoding := uint64(width)
	if sub {
		encoding |= 128
	}
	if b.Ops[a].Kind == Const && b.Ops[rhs].Kind == Const && validFlags(Op{Kind: ArithmeticStatus, Imm: encoding}) {
		return b.Constant(arithmeticStatus(b.Ops[a].Imm, b.Ops[rhs].Imm, encoding))
	}
	return b.emit(Op{Kind: ArithmeticStatus, A: a, B: rhs, Imm: encoding})
}

func (b *Builder) arithmeticStatusField(o Op, mask uint64) Value {
	width := o.Imm & 127
	word := b.Constant(^uint64(0) >> (64 - width))
	a, rhs := b.Binary(And, o.A, word), b.Binary(And, o.B, word)
	kind := Add
	if o.Imm&128 != 0 {
		kind = Sub
	}
	result := b.Binary(And, b.Binary(kind, a, rhs), word)
	if mask == 1 {
		if kind == Sub {
			return b.Binary(Less, a, rhs)
		}
		return b.Binary(Less, result, a)
	}
	if mask&0xc4 != 0 {
		return b.Binary(And, b.StatusBits(result, int(width)), b.Constant(mask))
	}
	ab := b.Binary(Xor, a, rhs)
	av := b.Binary(Xor, a, result)
	if mask == 16 {
		return b.Binary(And, b.Binary(Xor, ab, result), b.Constant(16))
	}
	if kind == Add {
		ab = b.Binary(Xor, ab, b.Constant(^uint64(0)))
	}
	overflow := b.Binary(And, ab, av)
	return b.Shift(Shl, b.Binary(And, b.Shift(Shr, overflow, width-1), b.Constant(1)), 11)
}
