package runtime

// StatusBits packs even low-byte parity, zero, and sign into bits 2, 6, and 7.
// The explicit word width is 8, 16, 32, or 64; no previous host flags are read.
const StatusBits = 43

func statusBits(value, width uint64) uint64 {
	if width < 64 {
		value &= (uint64(1) << width) - 1
	}
	result := (byteParity(value)^1)<<2 | ((value>>(width-1))&1)<<7
	if value == 0 {
		result |= 64
	}
	return result
}
func (b *Builder) StatusBits(value Value, width int) Value {
	if b.Ops[value].Kind == Const && (width == 8 || width == 16 || width == 32 || width == 64) {
		return b.Constant(statusBits(b.Ops[value].Imm, uint64(width)))
	}
	return b.emit(Op{Kind: StatusBits, A: value, B: value, Imm: uint64(width)})
}
