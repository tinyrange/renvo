package runtime

import "fmt"

// Block IR uses uint64 modular values and explicit state slots. Narrow guest
// arithmetic must explicitly mask its results; host flags are never guest flags.
// Shifts are immediate, with counts >=64 producing zero.
const (
	Const = iota
	LoadState
	StoreState
	Add
	Sub
	And
	Or
	Xor
	Shl
	Shr
	Equal
	Less
	Mul // uint64 modular multiplication; appended to preserve existing record IDs
)

type Value int
type Op struct {
	Kind int
	A, B Value
	Imm  uint64
}
type Builder struct {
	MemoryInstructionProgress bool                        // checkpoints group accesses into guest memory instructions
	MemoryAddress             func(*Builder, Value) Value // optional architectural key/guard policy
	BeforeMemoryStore         func(*Builder)              // optional architectural reservation guard, before any store checks
	Ops                       []Op
	state                     map[int]Value
	stateSlots                [256]Value
	statePresent              [8]uint32
	dirtySlots                [8]uint32
	masks                     []uint64
	valueBuckets              []Value // immutable Ops indices plus one; zero is empty
	valueCount                int
	dirty                     map[int]bool // stores since the previous effect checkpoint
	finishLive                []bool
	finishRemap               []Value
}

// Hash every component of the immutable expression. The index stores no
// duplicate keys: exact equality against Ops resolves all hash collisions.
// Capacity grows geometrically with the number of emitted values, not the
// magnitude of immediates or architectural register numbers.
func builderValueHash(op Op) uint64 {
	value := op.Imm ^ uint64(op.Kind)*uint64(0x9e3779b97f4a7c15) ^ uint64(op.A)*uint64(0xbf58476d1ce4e5b9) ^ uint64(op.B)*uint64(0x94d049bb133111eb)
	value = (value ^ (value >> 30)) * uint64(0xbf58476d1ce4e5b9)
	value = (value ^ (value >> 27)) * uint64(0x94d049bb133111eb)
	return value ^ (value >> 31)
}

func (b *Builder) valueBucket(op Op) int {
	if len(b.valueBuckets) == 0 || b.valueCount*2 >= len(b.valueBuckets) {
		size := len(b.valueBuckets) * 2
		if size < 16 {
			size = 16
		}
		buckets := make([]Value, size)
		for _, entry := range b.valueBuckets {
			if entry != 0 {
				slot := int(builderValueHash(b.Ops[entry-1]) & uint64(size-1))
				for buckets[slot] != 0 {
					slot = (slot + 1) & (size - 1)
				}
				buckets[slot] = entry
			}
		}
		b.valueBuckets = buckets
	}
	slot := int(builderValueHash(op) & uint64(len(b.valueBuckets)-1))
	for b.valueBuckets[slot] != 0 && b.Ops[b.valueBuckets[slot]-1] != op {
		slot = (slot + 1) & (len(b.valueBuckets) - 1)
	}
	return slot
}

func (b *Builder) emit(op Op) Value {
	// Value numbering is limited to immutable expressions. Architectural loads
	// are tracked by the state map, never interned across a state update.
	pure := op.Kind == Const || op.Kind >= Add && pureOperation(op.Kind)
	bucket := 0
	if pure {
		bucket = b.valueBucket(op)
		if entry := b.valueBuckets[bucket]; entry != 0 {
			return entry - 1
		}
	}
	v := Value(len(b.Ops))
	mask := ^uint64(0)
	switch op.Kind {
	case Const:
		mask = op.Imm
	case MemoryLoad:
		if op.Imm == 1 || op.Imm == 2 || op.Imm == 4 {
			mask = (uint64(1) << (op.Imm * 8)) - 1
		}
	case ArithmeticStatus:
		mask = 0x8d5
	case StatusBits:
		mask = 0xc4
	case ArithmeticCondition, LogicalCondition, ByteParity:
		mask = 1
	case UnsignedDivide, SignedDivide, UnsignedRemainder, SignedRemainder:
		if op.Imm == 32 {
			mask = 0xffffffff
		}
	case CarryArithmetic:
		if op.Imm&127 == 32 {
			mask = 0xffffffff
		}
	case LeadingZeros, LeadingSigns:
		mask = 127
	case ReverseBits, ReverseBytes:
		if op.Imm&255 == 32 {
			mask = 0xffffffff
		}
	case ArithmeticFlags, LogicalFlags, CarryFlags:
		mask = 0xf0000000
	case SelectValue:
		mask = b.masks[op.B] | b.masks[Value(op.Imm)]
	case VariableShl, VariableShr, ArithmeticShr, RotateRight:
		if op.Imm == 32 {
			mask = 0xffffffff
		}
	case Mul:
		// A boolean multiplicand selects either zero or the other operand.
		// Its possible bits cannot spread as they do in a general product.
		if b.masks[op.A] <= 1 {
			mask = b.masks[op.B]
		} else if b.masks[op.B] <= 1 {
			mask = b.masks[op.A]
		}
	case And:
		mask = b.masks[op.A] & b.masks[op.B]
	case Or, Xor:
		mask = b.masks[op.A] | b.masks[op.B]
	case Equal, Less:
		mask = 1
	case Shl:
		mask = b.masks[op.A] << op.Imm
	case Shr:
		mask = b.masks[op.A] >> op.Imm
	}
	b.Ops = append(b.Ops, op)
	b.masks = append(b.masks, mask)
	if pure {
		b.valueBuckets[bucket] = v + 1
		b.valueCount++
	}
	return v
}

// Reset reuses compilation scratch only. Finish/FinishMemory return independent
// operation slices, so previously installed blocks and their proofs never alias
// the next translation. No architectural or executable state lives here.
func (b *Builder) Reset(memoryProgress bool) {
	b.Ops = b.Ops[:0]
	b.masks = b.masks[:0]
	clear(b.state)
	clear(b.statePresent[:])
	clear(b.dirtySlots[:])
	clear(b.valueBuckets)
	b.valueCount = 0
	clear(b.dirty)
	b.MemoryAddress = nil
	b.BeforeMemoryStore = nil
	b.MemoryInstructionProgress = memoryProgress
}
func (b *Builder) finishScratch() ([]bool, []Value) {
	if cap(b.finishLive) < len(b.Ops) {
		b.finishLive = make([]bool, len(b.Ops))
	} else {
		b.finishLive = b.finishLive[:len(b.Ops)]
		clear(b.finishLive)
	}
	if cap(b.finishRemap) < len(b.Ops) {
		b.finishRemap = make([]Value, len(b.Ops))
	} else {
		b.finishRemap = b.finishRemap[:len(b.Ops)]
		clear(b.finishRemap)
	}
	return b.finishLive, b.finishRemap
}
func (b *Builder) Constant(v uint64) Value { return b.emit(Op{Kind: Const, Imm: v}) }
func (b *Builder) stateValue(slot int) (Value, bool) {
	if slot >= 0 && slot < len(b.stateSlots) {
		if b.statePresent[slot>>5]&(uint32(1)<<uint(slot&31)) != 0 {
			return b.stateSlots[slot], true
		}
		return 0, false
	}
	v, ok := b.state[slot]
	return v, ok
}
func (b *Builder) setStateValue(slot int, value Value) {
	if slot >= 0 && slot < len(b.stateSlots) {
		b.stateSlots[slot] = value
		b.statePresent[slot>>5] |= uint32(1) << uint(slot&31)
		return
	}
	if b.state == nil {
		b.state = map[int]Value{}
	}
	b.state[slot] = value
}
func (b *Builder) Load(slot int) Value {
	if v, ok := b.stateValue(slot); ok {
		return v
	}
	v := b.emit(Op{Kind: LoadState, Imm: uint64(slot)})
	b.setStateValue(slot, v)
	return v
}
func (b *Builder) Store(slot int, v Value) {
	b.setStateValue(slot, v)
	if slot >= 0 && slot < len(b.stateSlots) {
		b.dirtySlots[slot>>5] |= uint32(1) << uint(slot&31)
		return
	}
	if b.dirty == nil {
		b.dirty = map[int]bool{}
	}
	b.dirty[slot] = true
}
func operation(k int, a, b uint64) uint64 {
	switch k {
	case ByteParity:
		return byteParity(a)
	case Add:
		return a + b
	case Sub:
		return a - b
	case Mul:
		return a * b
	case And:
		return a & b
	case Or:
		return a | b
	case Xor:
		return a ^ b
	case Shl:
		if b >= 64 {
			return 0
		}
		return a << b
	case Shr:
		if b >= 64 {
			return 0
		}
		return a >> b
	case Equal:
		if a == b {
			return 1
		}
	case Less:
		if a < b {
			return 1
		}
	}
	return 0
}
func (b *Builder) Binary(k int, a, c Value) Value {
	if b.Ops[a].Kind == Const && b.Ops[c].Kind == Const {
		return b.Constant(operation(k, b.Ops[a].Imm, b.Ops[c].Imm))
	}
	if (k == And || k == Or || k == Xor || k == Add || k == Mul) && b.Ops[a].Kind == Const {
		a, c = c, a
	}

	if k == Equal {
		if b.Ops[a].Kind == Const {
			a, c = c, a
		}
		// A zero modular difference tests equality of the participating low
		// words. Preserve truncation: unequal high bits cannot affect it.
		if b.Ops[c].Kind == Const && b.Ops[c].Imm == 0 {
			difference := b.Ops[a]
			mask := ^uint64(0)
			if difference.Kind == And && b.Ops[difference.B].Kind == Const {
				mask = b.Ops[difference.B].Imm
				difference = b.Ops[difference.A]
			}
			if difference.Kind == Sub && mask&(mask+1) == 0 {
				return b.Binary(Equal, b.Binary(And, difference.A, b.Constant(mask)), b.Binary(And, difference.B, b.Constant(mask)))
			}
		}
		if b.Ops[c].Kind == Const && b.Ops[a].Kind == SelectValue {
			selected := b.Ops[a]
			t, f := b.Ops[selected.B], b.Ops[Value(selected.Imm)]
			if t.Kind == Const && f.Kind == Const && t.Imm != f.Imm {
				if b.Ops[c].Imm == t.Imm {
					return selected.A
				}
				if b.Ops[c].Imm == f.Imm {
					return b.Binary(Equal, selected.A, b.Constant(0))
				}
				return b.Constant(0)
			}
		}
	}
	if b.Ops[c].Kind == Const {
		mask := b.Ops[c].Imm
		if (k == Or || k == Xor || k == Add || k == Sub) && mask == 0 {
			return a
		}
		if k == Mul {
			if mask == 0 {
				return b.Constant(0)
			}
			if mask == 1 {
				return a
			}
			if mask&(mask-1) == 0 {
				var amount uint64
				for mask > 1 {
					mask >>= 1
					amount++
				}
				return b.Shift(Shl, a, amount)
			}
		}
		if k == And {
			if b.masks[a]&mask == 0 {
				return b.Constant(0)
			}
			if b.masks[a]&^mask == 0 {
				return a
			}
			prior := b.Ops[a]
			if prior.Kind == ArithmeticStatus {
				field := mask & 0x8d5
				if field != 0 && field&(field-1) == 0 {
					return b.arithmeticStatusField(prior, field)
				}
			}
			// Extract single status fields without materializing the packed word.
			if prior.Kind == StatusBits {
				v := b.Binary(And, prior.A, b.Constant(^uint64(0)>>(64-prior.Imm)))
				switch mask & 0xc4 {
				case 4:
					return b.Shift(Shl, b.Binary(Xor, b.ByteParity(v), b.Constant(1)), 2)
				case 64:
					return b.Shift(Shl, b.Binary(Equal, v, b.Constant(0)), 6)
				case 128:
					return b.Shift(Shl, b.Shift(Shr, v, prior.Imm-1), 7)
				}
			}
			// Push extraction through an immediate shift of a bitfield.
			// This exposes disjoint fields to the ordinary mask/OR rules.
			if prior.Kind == Shr && prior.Imm < 64 {
				source := b.Ops[prior.A]
				if source.Kind == Or || source.Kind == And || source.Kind == Shl || source.Kind == StatusBits || source.Kind == ArithmeticStatus {
					return b.Shift(Shr, b.Binary(And, prior.A, b.Constant(mask<<prior.Imm)), prior.Imm)
				}
			}
			// Low-bit truncation commutes with modular addition/subtraction/multiplication:
			// ((x & m) + y) & m == (x + y) & m for m = 2^n - 1.
			if mask&(mask+1) == 0 && (prior.Kind == Add || prior.Kind == Sub || prior.Kind == Mul) {
				strip := func(v Value) Value {
					o := b.Ops[v]
					if o.Kind == And && b.Ops[o.B].Kind == Const && b.Ops[o.B].Imm&mask == mask {
						return o.A
					}
					// (x XOR sign)-sign changes only bits above that sign bit.
					// They cannot affect a narrower modular result, even when
					// the sign extension itself still has an observable user.
					if o.Kind == Sub && b.Ops[o.B].Kind == Const {
						sign := b.Ops[o.B].Imm
						x := b.Ops[o.A]
						if sign != 0 && sign&(sign-1) == 0 && mask <= sign|(sign-1) && x.Kind == Xor && x.B == o.B {
							return x.A
						}
					}
					return v
				}
				x, y := strip(prior.A), strip(prior.B)
				if x != prior.A || y != prior.B {
					return b.Binary(And, b.Binary(prior.Kind, x, y), c)
				}
			}
			if prior.Kind == And && b.Ops[prior.B].Kind == Const {
				return b.Binary(And, prior.A, b.Constant(mask&b.Ops[prior.B].Imm))
			}
			// Remove overwritten flag computations using ordinary bit algebra.
			// No CPU-specific flags or state-slot identities enter this optimizer.
			if prior.Kind == Or {
				return b.Binary(Or, b.Binary(And, prior.A, c), b.Binary(And, prior.B, c))
			}
		}
	}
	if a == c {
		switch k {
		case And, Or:
			return a
		case Xor, Sub:
			return b.Constant(0)
		case Equal:
			return b.Constant(1)
		case Less:
			return b.Constant(0)
		}
	}
	return b.emit(Op{Kind: k, A: a, B: c})
}
func (b *Builder) Shift(k int, a Value, count uint64) Value {
	if count == 0 {
		return a
	}
	if count >= 64 {
		return b.Constant(0)
	}
	source := b.Ops[a]
	if k == Shr && source.Kind == Shl && source.Imm == count && b.masks[source.A]>>(64-count) == 0 {
		return source.A
	}
	if b.Ops[a].Kind == Const {
		return b.Constant(operation(k, b.Ops[a].Imm, count))
	}
	return b.emit(Op{Kind: k, A: a, Imm: count})
}
func (b *Builder) Choose(c, t, f Value) Value {
	// Conditions follow Go-style truth values even when supplied by an IR client.
	if b.masks[c]&^uint64(1) != 0 {
		c = b.Binary(Equal, b.Binary(Equal, c, b.Constant(0)), b.Constant(0))
	}
	if b.Ops[c].Kind == Const {
		if b.Ops[c].Imm != 0 {
			return t
		}
		return f
	}
	// A boolean selecting one bit is just a shift, not a full-width mask.
	if b.Ops[t].Kind == Const && b.Ops[f].Kind == Const && b.Ops[f].Imm == 0 {
		v := b.Ops[t].Imm
		if v != 0 && v&(v-1) == 0 {
			var shift uint64
			for v > 1 {
				v >>= 1
				shift++
			}
			if shift == 0 {
				return c
			}
			return b.Shift(Shl, c, shift)
		}
	}
	if t == f {
		return t
	}
	return b.emit(Op{Kind: SelectValue, A: c, B: t, Imm: uint64(f)})
}

// Finish drops dead values and defers state writes until all pure expressions
// have been evaluated. Slot ordering and emitted code are deterministic.
func (b *Builder) Finish(words int) []Op {
	live, remap := b.finishScratch()
	for i := 0; i < words; i++ {
		if v, ok := b.stateValue(i); ok {
			live[v] = true
		}
	}
	// Every SSA operand was emitted before its consumer. One reverse pass
	// computes the same transitive closure without recursion or closure calls.
	for i := len(b.Ops) - 1; i >= 0; i-- {
		if !live[i] {
			continue
		}
		o := b.Ops[i]
		if o.Kind >= Add {
			live[o.A] = true
			if o.Kind != Shl && o.Kind != Shr {
				live[o.B] = true
			}
			if o.Kind == SelectValue {
				live[Value(o.Imm)] = true
			}
			if carryKind(o.Kind) {
				live[Value(o.Imm>>8)] = true
			}
		}
	}
	out := make([]Op, 0, len(b.Ops)+words)
	for i, o := range b.Ops {
		if !live[i] {
			continue
		}
		remap[i] = Value(len(out))
		if o.Kind >= Add {
			o.A = remap[o.A]
			if o.Kind != Shl && o.Kind != Shr {
				o.B = remap[o.B]
			}
			if o.Kind == SelectValue {
				o.Imm = uint64(remap[Value(o.Imm)])
			}
			if carryKind(o.Kind) {
				o.Imm = o.Imm&255 | uint64(remap[Value(o.Imm>>8)])<<8
			}
		}
		out = append(out, o)
	}
	for i := 0; i < words; i++ {
		if v, ok := b.stateValue(i); ok {
			out = append(out, Op{Kind: StoreState, A: remap[v], Imm: uint64(i)})
		}
	}
	return out
}
func Validate(ops []Op, words int) error {
	if words < 1 || words > 256 || len(ops) == 0 || len(ops) > 2048 {
		return fmt.Errorf("invalid block dimensions")
	}
	for i, o := range ops {
		if carryKind(o.Kind) && (o.Imm>>8 >= uint64(i) || effectOnly(ops[o.Imm>>8].Kind) || !validFlags(Op{Kind: ArithmeticFlags, Imm: o.Imm & 255})) {
			return fmt.Errorf("invalid carry operand/width")
		}
		if !validFlags(o) || !validRich(o) || !validDivision(o) {
			return fmt.Errorf("invalid flags width")
		}
		if !pureOperation(o.Kind) {
			return fmt.Errorf("invalid IR operation %d", o.Kind)
		}
		if o.Kind == SelectValue && (o.Imm >= uint64(i) || ops[o.Imm].Kind == StoreState) {
			return fmt.Errorf("invalid select operand")
		}
		if o.Kind >= VariableShl && o.Kind <= RotateRight && o.Imm != 32 && o.Imm != 64 {
			return fmt.Errorf("invalid variable shift width")
		}
		if (o.Kind == LoadState || o.Kind == StoreState) && o.Imm >= uint64(words) {
			return fmt.Errorf("invalid state slot")
		}
		if o.Kind >= StoreState && (o.A < 0 || int(o.A) >= i || ops[o.A].Kind == StoreState) {
			return fmt.Errorf("invalid IR operand")
		}
		if o.Kind >= Add && o.Kind != Shl && o.Kind != Shr && (o.B < 0 || int(o.B) >= i || ops[o.B].Kind == StoreState) {
			return fmt.Errorf("invalid IR operand")
		}
	}
	return nil
}
func Interpret(ops []Op, state []uint64) error {
	if err := Validate(ops, len(state)); err != nil {
		return err
	}
	values := make([]uint64, len(ops))
	interpretValues(ops, state, values)
	return nil
}

// interpretValues requires validated, immutable IR and scratch for every op.
func interpretValues(ops []Op, state, values []uint64) {
	for i, o := range ops {
		switch o.Kind {
		case Const:
			values[i] = o.Imm
		case LoadState:
			values[i] = state[o.Imm]
		case StoreState:
			state[o.Imm] = values[o.A]
		case Shl, Shr:
			values[i] = operation(o.Kind, values[o.A], o.Imm)
		case ArithmeticStatus:
			values[i] = arithmeticStatus(values[o.A], values[o.B], o.Imm)
		case StatusBits:
			values[i] = statusBits(values[o.A], o.Imm)
		case ArithmeticCondition, LogicalCondition:
			values[i] = 0
			if conditionFlags(flagsOperation(o.Kind-2, values[o.A], values[o.B], o.Imm&255), int(o.Imm>>8)) {
				values[i] = 1
			}
		case ArithmeticFlags, LogicalFlags:
			values[i] = flagsOperation(o.Kind, values[o.A], values[o.B], o.Imm)
		case SelectValue:
			values[i] = values[o.Imm]
			if values[o.A] != 0 {
				values[i] = values[o.B]
			}
		case UnsignedDivide, SignedDivide, UnsignedRemainder, SignedRemainder:
			values[i] = divisionOperation(o.Kind, values[o.A], values[o.B], o.Imm)
		case CarryArithmetic, CarryFlags:
			values[i] = carryOperation(o.Kind, values[o.A], values[o.B], values[o.Imm>>8], o.Imm&255)
		case UnsignedMulHigh, SignedMulHigh, LeadingZeros, LeadingSigns, ReverseBytes, ReverseBits:
			values[i] = richOperation(o.Kind, values[o.A], values[o.B], o.Imm)
		case VariableShl, VariableShr, ArithmeticShr, RotateRight:
			values[i] = variableOperation(o.Kind, values[o.A], values[o.B], o.Imm)
		default:
			values[i] = operation(o.Kind, values[o.A], values[o.B])
		}
	}
}
