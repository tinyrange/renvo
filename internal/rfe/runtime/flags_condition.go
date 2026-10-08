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
	return b.emit(Op{Kind: o.Kind + 2, A: o.A, B: o.B, Imm: o.Imm | uint64(condition)<<8}), true
}
