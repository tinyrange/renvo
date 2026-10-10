package runtime

// ByteParity is the odd parity of the low eight bits, independent of upper bits.
// Keep the record append-only. Like other unary rich operations, B aliases A.
const ByteParity = 42

func byteParity(value uint64) uint64 {
	value &= 255
	value ^= value >> 4
	value ^= value >> 2
	value ^= value >> 1
	return value & 1
}

func (b *Builder) ByteParity(value Value) Value {
	if b.Ops[value].Kind == Const {
		return b.Constant(byteParity(b.Ops[value].Imm))
	}
	return b.emit(Op{Kind: ByteParity, A: value, B: value})
}
