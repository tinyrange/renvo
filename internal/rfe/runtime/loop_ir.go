package runtime

// LoopContinue is a structural terminator accepted only by CompileLoop. Its
// condition decides whether another whole iteration may execute. It neither
// faults nor returns to Go; the native loop reconstructs state on its exits.
const LoopContinue = 35

func (b *Builder) LoopContinue(condition Value) { b.emit(Op{Kind: LoopContinue, A: condition}) }
