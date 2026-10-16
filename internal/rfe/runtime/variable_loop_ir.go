package runtime

// LoopCount is a structural pure-loop annotation. Its bounded SSA value is
// the actual number of guest instructions in this iteration, not the number
// of speculatively evaluated host expressions. It is not a runtime effect.
const LoopCount = 46

func (b *Builder) LoopCount(count Value) { b.emit(Op{Kind: LoopCount, A: count}) }

// PureIf evaluates only immutable expressions and merges architectural slots.
// No memory, guard, elapsed read or committed state effect may be speculated.
// Failure invalidates the tentative builder; callers must discard it.
func (b *Builder) PureIf(condition Value, words int, lower func(*Builder) bool) bool {
	before := make(map[int]Value, len(b.state))
	for slot, value := range b.state {
		before[slot] = value
	}
	start := len(b.Ops)
	if !lower(b) {
		return false
	}
	for _, o := range b.Ops[start:] {
		if o.Kind == LoadState {
			before[int(o.Imm)] = Value(0)
			continue
		}
		if o.Kind == StoreState || !pureOperation(o.Kind) {
			return false
		}
	}
	// A newly loaded slot is still the pre-branch architectural input. Reuse
	// that SSA input instead of introducing a duplicate loop phi.
	for i := start; i < len(b.Ops); i++ {
		if b.Ops[i].Kind == LoadState {
			before[int(b.Ops[i].Imm)] = Value(i)
		}
	}
	after := make(map[int]Value, len(b.state))
	for slot, value := range b.state {
		after[slot] = value
	}
	for slot := 0; slot < words; slot++ {
		next, changed := after[slot]
		if !changed || !b.dirty[slot] {
			continue
		}
		old, known := before[slot]
		if !known {
			old = b.emit(Op{Kind: LoadState, Imm: uint64(slot)})
		}
		b.Store(slot, b.Choose(condition, next, old))
	}
	return true
}

// ChoiceOf proves a lowered direct successor from its SSA expression, rather
// than trusting discovery metadata to refine a dynamic architectural PC.
func (b *Builder) ChoiceOf(v Value, first, second uint64) bool {
	o := b.Ops[v]
	if o.Kind == Const {
		return o.Imm == first || o.Imm == second
	}
	return o.Kind == SelectValue && b.ChoiceOf(o.B, first, second) && b.ChoiceOf(Value(o.Imm), first, second)
}

func boundedLoopCount(ops []Op, value Value, limit int) bool {
	lo, hi := make([]int, len(ops)), make([]int, len(ops))
	for i, o := range ops {
		lo[i], hi[i] = -1, -1
		switch o.Kind {
		case Const:
			if o.Imm <= 256 {
				lo[i], hi[i] = int(o.Imm), int(o.Imm)
			}
		case Add:
			if lo[o.A] >= 0 && lo[o.B] >= 0 && hi[o.A]+hi[o.B] <= 256 {
				lo[i], hi[i] = lo[o.A]+lo[o.B], hi[o.A]+hi[o.B]
			}
		case Equal, Less, ArithmeticCondition, LogicalCondition, ByteParity:
			lo[i], hi[i] = 0, 1
		case And:
			if ops[o.B].Kind == Const && ops[o.B].Imm <= 256 {
				lo[i], hi[i] = 0, int(ops[o.B].Imm)
			}
		case Shl:
			if lo[o.A] >= 0 && o.Imm < 9 && uint64(hi[o.A])<<o.Imm <= 256 {
				lo[i], hi[i] = lo[o.A]<<o.Imm, hi[o.A]<<o.Imm
			}
		case Mul:
			if lo[o.A] >= 0 && lo[o.B] >= 0 && hi[o.A]*hi[o.B] <= 256 {
				lo[i], hi[i] = lo[o.A]*lo[o.B], hi[o.A]*hi[o.B]
			}
		case SelectValue:
			if lo[o.B] >= 0 && lo[o.Imm] >= 0 {
				lo[i], hi[i] = min(lo[o.B], lo[o.Imm]), max(hi[o.B], hi[o.Imm])
			}
		}
	}
	return lo[value] > 0 && hi[value] <= limit
}
