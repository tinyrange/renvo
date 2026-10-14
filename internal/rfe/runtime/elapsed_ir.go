package runtime

// NativeElapsed reads a caller-supplied emulated clock base plus scaled native
// retirement since entry. It is contextual, not a pure/commonable expression:
// each read observes its exact instruction prefix and native loop iteration.
// The caller imports a fresh base at each native scheduling boundary. Ordinary
// leaf calls start at zero; linked dispatch includes preceding linked bodies.
const NativeElapsed = 45

func (b *Builder) Elapsed(base Value, scale, completed int) Value {
	return b.emit(Op{Kind: NativeElapsed, A: base, B: Value(scale), Imm: uint64(completed)})
}
