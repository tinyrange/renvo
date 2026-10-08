package runtime

import "fmt"

// Effect records are accepted only by CompileMemory, never by the pure evaluator
// or emitter. Memory values are not interned across loads/stores. All memory
// guards exit before the access and preserve the preceding committed state.
const (
	MemoryLoad  = Mul + 1
	MemoryStore = Mul + 2
	Progress    = Mul + 3
	Guard       = Mul + 4
)

func (b *Builder) MemoryLoad(address Value, size int) Value {
	return b.emit(Op{Kind: MemoryLoad, A: address, Imm: uint64(size)})
}
func (b *Builder) MemoryStore(address, value Value, size int) {
	b.emit(Op{Kind: MemoryStore, A: address, B: value, Imm: uint64(size)})
}
func (b *Builder) Guard(condition Value) { b.emit(Op{Kind: Guard, A: condition}) }

// Commit writes only dirty architectural slots in deterministic order. State
// forwarding remains valid, but checkpoints are observable roots, not deferred
// writes that may be moved past a fault or a guest-memory side effect.
func (b *Builder) Commit(words int) {
	for i := 0; i < words; i++ {
		if b.dirty[i] {
			b.emit(Op{Kind: StoreState, A: b.state[i], Imm: uint64(i)})
			delete(b.dirty, i)
		}
	}
}
func (b *Builder) Checkpoint(words, completed int) {
	b.Commit(words)
	b.emit(Op{Kind: Progress, Imm: uint64(completed)})
}
func memoryOperands(o Op) (bool, bool) {
	switch o.Kind {
	case Const, LoadState, Progress:
		return false, false
	case StoreState, Shl, Shr, MemoryLoad, Guard, RegionGuard, LoopContinue, PairHigh:
		return true, false
	default:
		return true, true
	}
}
func effectOnly(k int) bool {
	return k == MemoryPairStore || k == StoreState || k == MemoryStore || k == Progress || k == Guard || k == RegionGuard || k == LoopContinue
}
func (b *Builder) FinishMemory(words int) []Op {
	b.Commit(words)
	live := make([]bool, len(b.Ops))
	var visit func(Value)
	visit = func(v Value) {
		if live[v] {
			return
		}
		live[v] = true
		o := b.Ops[v]
		left, right := memoryOperands(o)
		if left {
			visit(o.A)
		}
		if right {
			visit(o.B)
		}
		if o.Kind == SelectValue {
			visit(Value(o.Imm))
		}
		if carryKind(o.Kind) || o.Kind == MemoryPairStore {
			visit(Value(o.Imm >> 8))
		}
	}
	for i, o := range b.Ops {
		if effectOnly(o.Kind) || o.Kind == MemoryLoad || o.Kind == PairHigh {
			visit(Value(i))
		}
	}
	remap := make([]Value, len(b.Ops))
	var out []Op
	for i, o := range b.Ops {
		if !live[i] {
			continue
		}
		remap[i] = Value(len(out))
		left, right := memoryOperands(o)
		if left {
			o.A = remap[o.A]
		}
		if right {
			o.B = remap[o.B]
		}
		if o.Kind == SelectValue {
			o.Imm = uint64(remap[Value(o.Imm)])
		}
		if carryKind(o.Kind) || o.Kind == MemoryPairStore {
			o.Imm = o.Imm&255 | uint64(remap[Value(o.Imm>>8)])<<8
		}
		out = append(out, o)
	}
	return out
}
func ValidateMemory(ops []Op, words int) error {
	if words < 1 || words > 256 || len(ops) == 0 || len(ops) > 2048 {
		return fmt.Errorf("invalid memory block dimensions")
	}
	for i, o := range ops {
		if carryKind(o.Kind) && (o.Imm>>8 >= uint64(i) || effectOnly(ops[o.Imm>>8].Kind) || !validFlags(Op{Kind: ArithmeticFlags, Imm: o.Imm & 255})) {
			return fmt.Errorf("invalid carry operand/width")
		}
		if !validFlags(o) || !validRich(o) {
			return fmt.Errorf("invalid flags width")
		}
		if !pureOperation(o.Kind) && (o.Kind < MemoryLoad || o.Kind > Guard) && o.Kind != RegionGuard && o.Kind != LoopContinue && o.Kind != PairHigh && o.Kind != MemoryPairStore {
			return fmt.Errorf("invalid memory IR operation")
		}
		if o.Kind == SelectValue && (o.Imm >= uint64(i) || effectOnly(ops[o.Imm].Kind)) {
			return fmt.Errorf("invalid select operand")
		}
		if o.Kind >= VariableShl && o.Kind <= RotateRight && o.Imm != 32 && o.Imm != 64 {
			return fmt.Errorf("invalid variable shift width")
		}
		if (o.Kind == LoadState || o.Kind == StoreState) && o.Imm >= uint64(words) {
			return fmt.Errorf("invalid state slot")
		}
		if (o.Kind == MemoryLoad || o.Kind == MemoryStore) && o.Imm != 1 && o.Imm != 2 && o.Imm != 4 && o.Imm != 8 && !(o.Kind == MemoryLoad && o.Imm == 16) {
			return fmt.Errorf("invalid memory access size")
		}
		if o.Kind == MemoryPairStore && (o.Imm&255 != 16 || o.Imm>>8 >= uint64(i) || effectOnly(ops[o.Imm>>8].Kind)) {
			return fmt.Errorf("invalid pair store operand")
		}
		if o.Kind == PairHigh && (i == 0 || o.A != Value(i-1) || ops[i-1].Kind != MemoryLoad || ops[i-1].Imm != 16 || o.B != 0 || o.Imm != 0) {
			return fmt.Errorf("invalid paired load result")
		}
		if o.Kind == MemoryLoad && o.Imm == 16 && (i+1 >= len(ops) || ops[i+1].Kind != PairHigh || ops[i+1].A != Value(i)) {
			return fmt.Errorf("missing paired load result")
		}
		if o.Kind == Progress && o.Imm > 256 {
			return fmt.Errorf("invalid progress checkpoint")
		}
		left, right := memoryOperands(o)
		for _, v := range []Value{o.A, o.B} {
			if !left {
				left = right
				continue
			}
			if v < 0 || int(v) >= i || effectOnly(ops[v].Kind) {
				return fmt.Errorf("invalid memory IR operand")
			}
			left = right
		}
	}
	return nil
}

// NativePage contains only host-owned, GC-visible pointers. Permissions mirror
// the live mapping and must be invalidated on protect/unmap/remap. Epoch points
// at the same generation queried by the architectural memory implementation.
type NativePage struct {
	Number      uint64
	Data        *[4096]byte
	Permissions uint64
	Epoch       *uint64
}

// MemoryContext is the checked native-memory ABI, owned by one single-threaded
// address space and shared by its aliases. Native stores require non-executable
// writable pages and allocate a fresh clock generation before changing bytes.
// Executable stores, cross-page accesses and cache misses exit to architecture.
// Status: 0 success; 1 access slow path (Address); 2 failed non-memory guard;
// 3 invalid linked progress (a host implementation error, not a guest trap);
// 4 successful region side exit after a committed architectural checkpoint.
type MemoryContext struct {
	Retired         uint64
	Status          uint64
	Address         uint64
	Clock           *uint64
	Pages           [64]NativePage
	Remaining       uint64
	Total           uint64
	MemoryTotal     uint64
	CodeView        [4]uint64 // populated only while the arena lock is held
	Blocks          [1024]NativeLink
	LoopExits       uint64 // successful loop branches handled entirely in native dispatch
	LoopIterations  uint64
	PreparedTargets uint64 // borrowed arena-owned proof table, zero outside serialized calls
	DescriptorBase  uint64 // zero on public contexts; session-private borrowed descriptor view
	AdmissionEpoch  uint64 // always zero on public contexts; private session admission generation
	linkOwner       *Native
	linkVictim      uint64 // cold round-robin replacement; not part of the native ABI
}

func (m *MemoryContext) Fill(number uint64, data *[4096]byte, permissions uint64, epoch *uint64) {
	if number >= 1<<36 || data == nil || permissions & ^uint64(7) != 0 || epoch == nil {
		m.Forget(number)
		return
	}
	m.Pages[number&63] = NativePage{number, data, permissions, epoch}
}
func (m *MemoryContext) Forget(number uint64) {
	p := &m.Pages[number&63]
	if p.Number == number {
		*p = NativePage{}
	}
}
