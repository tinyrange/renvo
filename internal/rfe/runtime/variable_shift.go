package runtime

// Appended IDs preserve pure 0..12 and memory-effect 13..16 records. Variable
// shifts explicitly select a 32/64-bit word and reduce the count modulo width.
const (
	VariableShl   = 17
	VariableShr   = 18
	ArithmeticShr = 19
	RotateRight   = 20
)

func pureOperation(k int) bool {
	return k >= Const && k <= Mul || k >= VariableShl && k <= LogicalCondition || k >= UnsignedMulHigh && k <= CarryFlags
}
func variableOperation(k int, value, count uint64, width uint64) uint64 {
	count &= width - 1
	if width == 32 {
		value = uint64(uint32(value))
	}
	switch k {
	case VariableShl:
		value <<= count
	case VariableShr:
		value >>= count
	case ArithmeticShr:
		if width == 32 {
			value = uint64(uint32(int32(value) >> count))
		} else {
			value = uint64(int64(value) >> count)
		}
	case RotateRight:
		value = value>>count | value<<(width-count)
	}
	if width == 32 {
		value = uint64(uint32(value))
	}
	return value
}
func (b *Builder) VariableShift(k int, value, count Value, width int) Value {
	if k >= VariableShl && k <= RotateRight && b.Ops[value].Kind == Const && b.Ops[count].Kind == Const && (width == 32 || width == 64) {
		return b.Constant(variableOperation(k, b.Ops[value].Imm, b.Ops[count].Imm, uint64(width)))
	}
	return b.emit(Op{Kind: k, A: value, B: count, Imm: uint64(width)})
}
