package runtime

// PairHigh is the second result of the immediately preceding 16-byte checked
// RAM load. It performs no access of its own; both halves are produced only
// after the complete pair passed the original load's fault/permission checks.
// This structural result is accepted only by memory emitters, not pure IR.
const (
	PairHigh        = 36
	MemoryPairStore = 37
)

// A pair store is one effect and one fresh generation, never two scalar stores.
func (b *Builder) MemoryPairStore(address, first, second Value) {
	b.emit(Op{Kind: MemoryPairStore, A: address, B: first, Imm: uint64(second)<<8 | 16})
}

func (b *Builder) MemoryPairLoad(address Value) (Value, Value) {
	low := b.MemoryLoad(address, 16)
	high := b.emit(Op{Kind: PairHigh, A: low})
	return low, high
}
