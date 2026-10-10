package runtime

// Flags operations return architectural NZCV in bits 31..28. ArithmeticFlags
// uses width 32/64, plus bit 7 for subtraction; it does not consume host flags
// left by another IR operation. LogicalFlags describes the already computed A.
const (
	ArithmeticFlags = 22
	LogicalFlags    = 23
)

func validFlags(o Op) bool {
	if o.Kind == ArithmeticStatus {
		return o.Imm == 8 || o.Imm == 16 || o.Imm == 32 || o.Imm == 64 || o.Imm == 136 || o.Imm == 144 || o.Imm == 160 || o.Imm == 192
	}
	if o.Kind == StatusBits {
		return o.Imm == 8 || o.Imm == 16 || o.Imm == 32 || o.Imm == 64
	}
	if o.Kind == ArithmeticCondition || o.Kind == LogicalCondition {
		if o.Imm>>8 > 15 {
			return false
		}
		o.Kind -= 2
		o.Imm &= 255
	}
	if o.Kind == ArithmeticFlags {
		return o.Imm == 32 || o.Imm == 64 || o.Imm == 160 || o.Imm == 192
	}
	if o.Kind == LogicalFlags {
		return o.Imm == 32 || o.Imm == 64
	}
	return true
}
func flagsOperation(kind int, a, rhs, encoding uint64) uint64 {
	width := encoding & 127
	m := ^uint64(0)
	if width == 32 {
		m = 0xffffffff
	}
	a &= m
	rhs &= m
	result, carry, overflow := a, false, false
	sign := uint64(1) << (width - 1)
	if kind == ArithmeticFlags {
		if encoding&128 != 0 {
			result = (a - rhs) & m
			carry = a >= rhs
			overflow = (a^rhs)&(a^result)&sign != 0
		} else {
			result = (a + rhs) & m
			carry = result < a
			overflow = ^(a^rhs)&(a^result)&sign != 0
		}
	}
	var flags uint64
	if result&sign != 0 {
		flags |= 1 << 31
	}
	if result == 0 {
		flags |= 1 << 30
	}
	if carry {
		flags |= 1 << 29
	}
	if overflow {
		flags |= 1 << 28
	}
	return flags
}
func (b *Builder) ArithmeticFlags(a, rhs Value, width int, sub bool) Value {
	encoding := uint64(width)
	if sub {
		encoding |= 128
	}
	if b.Ops[a].Kind == Const && b.Ops[rhs].Kind == Const && validFlags(Op{Kind: ArithmeticFlags, Imm: encoding}) {
		return b.Constant(flagsOperation(ArithmeticFlags, b.Ops[a].Imm, b.Ops[rhs].Imm, encoding))
	}
	return b.emit(Op{Kind: ArithmeticFlags, A: a, B: rhs, Imm: encoding})
}
func (b *Builder) LogicalFlags(value Value, width int) Value {
	if b.Ops[value].Kind == Const && (width == 32 || width == 64) {
		return b.Constant(flagsOperation(LogicalFlags, b.Ops[value].Imm, 0, uint64(width)))
	}
	return b.emit(Op{Kind: LogicalFlags, A: value, B: value, Imm: uint64(width)})
}
