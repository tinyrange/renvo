package runtime

const (
	CarryArithmetic = 33
	CarryFlags      = 34
)

// Imm encodes a third SSA operand above the width/subtraction byte.
func carryKind(k int) bool { return k == CarryArithmetic || k == CarryFlags }
func carryOperation(kind int, a, rhs, carry, encoding uint64) uint64 {
	width, sub := encoding&127, encoding&128 != 0
	carry &= 1
	m := ^uint64(0)
	if width == 32 {
		m = 0xffffffff
	}
	a &= m
	rhs &= m
	var result, cout uint64
	if sub {
		partial := a - rhs
		result = partial - (1 - carry)
		if a >= rhs && partial >= 1-carry {
			cout = 1
		}
	} else {
		partial := a + rhs
		result = partial + carry
		if partial < a || result < partial {
			cout = 1
		}
	}
	if width == 32 {
		if sub {
			if a >= rhs && (a > rhs || carry != 0) {
				cout = 1
			} else {
				cout = 0
			}
		} else {
			cout = result >> 32
		}
		result &= m
	}
	if kind == CarryArithmetic {
		return result
	}
	sign := uint64(1) << (width - 1)
	var flags uint64
	if result&sign != 0 {
		flags |= 1 << 31
	}
	if result == 0 {
		flags |= 1 << 30
	}
	if cout != 0 {
		flags |= 1 << 29
	}
	overflow := ^(a ^ rhs) & (a ^ result) & sign
	if sub {
		overflow = (a ^ rhs) & (a ^ result) & sign
	}
	if overflow != 0 {
		flags |= 1 << 28
	}
	return flags
}
func (b *Builder) Carry(kind int, a, rhs, carry Value, width int, sub bool) Value {
	encoding := uint64(width)
	if sub {
		encoding |= 128
	}
	if (width == 32 || width == 64) && b.Ops[a].Kind == Const && b.Ops[rhs].Kind == Const && b.Ops[carry].Kind == Const {
		return b.Constant(carryOperation(kind, b.Ops[a].Imm, b.Ops[rhs].Imm, b.Ops[carry].Imm, encoding))
	}
	return b.emit(Op{Kind: kind, A: a, B: rhs, Imm: encoding | uint64(carry)<<8})
}
