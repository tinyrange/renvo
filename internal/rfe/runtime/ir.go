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
	Ops    []Op
	state  map[int]Value
	masks  []uint64
	values map[Op]Value
	dirty  map[int]bool // stores since the previous effect checkpoint
}

func (b *Builder) emit(op Op) Value {
	// Value numbering is limited to immutable expressions. Architectural loads
	// are tracked by the state map, never interned across a state update.
	pure := op.Kind == Const || op.Kind >= Add && pureOperation(op.Kind)
	if pure {
		if v, ok := b.values[op]; ok {
			return v
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
	case ArithmeticCondition, LogicalCondition:
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
		if b.values == nil {
			b.values = map[Op]Value{}
		}
		b.values[op] = v
	}
	return v
}
func (b *Builder) Constant(v uint64) Value { return b.emit(Op{Kind: Const, Imm: v}) }
func (b *Builder) Load(slot int) Value {
	if b.state == nil {
		b.state = map[int]Value{}
	}
	if v, ok := b.state[slot]; ok {
		return v
	}
	v := b.emit(Op{Kind: LoadState, Imm: uint64(slot)})
	b.state[slot] = v
	return v
}
func (b *Builder) Store(slot int, v Value) {
	if b.state == nil {
		b.state = map[int]Value{}
	}
	b.state[slot] = v
	if b.dirty == nil {
		b.dirty = map[int]bool{}
	}
	b.dirty[slot] = true
}
func operation(k int, a, b uint64) uint64 {
	switch k {
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
			// Low-bit truncation commutes with modular addition/subtraction/multiplication:
			// ((x & m) + y) & m == (x + y) & m for m = 2^n - 1.
			if mask&(mask+1) == 0 && (prior.Kind == Add || prior.Kind == Sub || prior.Kind == Mul) {
				strip := func(v Value) Value {
					o := b.Ops[v]
					if o.Kind == And && b.Ops[o.B].Kind == Const && b.Ops[o.B].Imm&mask == mask {
						return o.A
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
	live := make([]bool, len(b.Ops))
	var visit func(Value)
	visit = func(v Value) {
		if live[v] {
			return
		}
		live[v] = true
		o := b.Ops[v]
		if o.Kind >= Add {
			visit(o.A)
			if o.Kind != Shl && o.Kind != Shr {
				visit(o.B)
			}
			if o.Kind == SelectValue {
				visit(Value(o.Imm))
			}
			if carryKind(o.Kind) {
				visit(Value(o.Imm >> 8))
			}
		}
	}
	for i := 0; i < words; i++ {
		if v, ok := b.state[i]; ok {
			visit(v)
		}
	}
	remap := make([]Value, len(b.Ops))
	var out []Op
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
		if v, ok := b.state[i]; ok {
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
