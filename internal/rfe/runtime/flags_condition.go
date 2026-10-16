package runtime

// Direct predicates retain the arithmetic operands, rather than packing NZCV
// and immediately extracting it again. Architectural NZCV stores remain roots.
const (
	ArithmeticCondition = 24
	LogicalCondition    = 25
)

func conditionFlags(flags uint64, condition int) bool {
	n, z, c, v := flags&(1<<31) != 0, flags&(1<<30) != 0, flags&(1<<29) != 0, flags&(1<<28) != 0
	result := true
	switch condition >> 1 {
	case 0:
		result = z
	case 1:
		result = c
	case 2:
		result = n
	case 3:
		result = v
	case 4:
		result = c && !z
	case 5:
		result = n == v
	case 6:
		result = n == v && !z
	}
	if condition&1 != 0 && condition != 15 {
		result = !result
	}
	return result
}
func (b *Builder) FlagCondition(flags Value, condition int) (Value, bool) {
	if condition < 0 || condition > 15 {
		return 0, false
	}
	if condition >= 14 {
		return b.Constant(1), true
	}
	o := b.Ops[flags]
	if o.Kind == Const {
		if conditionFlags(o.Imm, condition) {
			return b.Constant(1), true
		}
		return b.Constant(0), true
	}
	if o.Kind != ArithmeticFlags && o.Kind != LogicalFlags {
		return 0, false
	}
	// Adding/subtracting zero has the logical result A, with no overflow.
	// Subtraction has no borrow (C=1); addition and logical flags have C=0.
	// Mask before testing: high bits are not part of a 32-bit flag result.
	width := o.Imm & 127
	if width == 32 || width == 64 {
		mask := ^uint64(0) >> (64 - width)
		if o.Kind == LogicalFlags || b.masks[o.B]&mask == 0 {
			return b.resultCondition(o.A, width, condition, o.Kind == ArithmeticFlags && o.Imm&128 != 0), true
		}
	}
	return b.emit(Op{Kind: o.Kind + 2, A: o.A, B: o.B, Imm: o.Imm | uint64(condition)<<8}), true
}

// Conditions on a result with V=0 and a known C. This does not replace the
// packed flag word: other consumers still observe its complete NZCV value.
func (b *Builder) resultCondition(value Value, width uint64, condition int, carry bool) Value {
	invert := condition&1 != 0
	pair := condition >> 1
	var result Value
	switch pair {
	case 1, 3, 4:
		known := carry && pair != 3
		if pair != 4 || !known {
			if known != invert {
				return b.Constant(1)
			}
			return b.Constant(0)
		}
		// HI with C=1 is !Z; LS is Z.
		pair, invert = 0, !invert
	case 5:
		// GE is !N when V=0.
		pair, invert = 2, !invert
	case 6:
		if b.masks[value]&(uint64(1)<<(width-1)) == 0 {
			// A known nonnegative result makes GT exactly !Z.
			pair, invert = 0, !invert
		} else {
			return b.emit(Op{Kind: LogicalCondition, A: value, B: value, Imm: width | uint64(condition)<<8})
		}
	}
	if pair == 0 {
		mask := ^uint64(0) >> (64 - width)
		result = b.Binary(Equal, b.Binary(And, value, b.Constant(mask)), b.Constant(0))
	} else {
		result = b.Binary(And, b.Shift(Shr, value, width-1), b.Constant(1))
	}
	if invert {
		result = b.Binary(Xor, result, b.Constant(1))
	}
	return result
}
