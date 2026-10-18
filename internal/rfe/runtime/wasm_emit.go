package runtime

import "fmt"

// wasmCode emits RFE SSA values as native i64 WASM locals. It consumes the same
// validated IR as the native backends; no guest decode or interpreter runs in
// the generated block. The module borrows its caller's linear memory.
type wasmCode struct {
	code           []byte
	ops            []Op
	inline         []bool
	stateSlots     []int
	stateWritten   []bool
	flagSlots      []wasmFlagSlots
	graphExit      *wasmGraphExit
	flagFunction   int
	refillFunction int
	sessionLocals  []int
	sessionOffsets []uint32
	pageWindow     []int            // private exact guest base and checked host RW base
	spanProofs     []wasmMemorySpan // private loop mappings invalidated at host reentry
}

func (w *wasmCode) byte(v byte) { w.code = append(w.code, v) }
func (w *wasmCode) u32(v uint32) {
	for v >= 128 {
		w.byte(byte(v) | 128)
		v >>= 7
	}
	w.byte(byte(v))
}
func (w *wasmCode) i64(v uint64) {
	w.byte(0x42)
	for {
		b := byte(v & 127)
		next := uint64(int64(v) >> 7)
		done := next == 0 && b&64 == 0 || next == ^uint64(0) && b&64 != 0
		if !done {
			b |= 128
		}
		w.byte(b)
		v = next
		if done {
			return
		}
	}
}
func (w *wasmCode) i32(v uint32) {
	w.byte(0x41)
	for {
		b := byte(v & 127)
		next := uint32(int32(v) >> 7)
		done := next == 0 && b&64 == 0 || next == ^uint32(0) && b&64 != 0
		if !done {
			b |= 128
		}
		w.byte(b)
		v = next
		if done {
			return
		}
	}
}
func (w *wasmCode) get(v int) {
	if v >= 2 && v-2 < len(w.inline) && w.inline[v-2] {
		w.graphValue(w.ops[v-2])
		return
	}
	w.byte(0x20)
	w.u32(uint32(v))
}
func (w *wasmCode) set(v int)     { w.byte(0x21); w.u32(uint32(v)) }
func (w *wasmCode) name(v string) { w.u32(uint32(len(v))); w.code = append(w.code, v...) }
func (w *wasmCode) section(id byte, payload []byte) {
	w.byte(id)
	w.u32(uint32(len(payload)))
	w.code = append(w.code, payload...)
}
func (w *wasmCode) mem(op byte, alignment, offset uint32) {
	w.byte(op)
	w.u32(alignment)
	w.u32(offset)
}
func (w *wasmCode) binary(a, b int, op byte) { w.get(a); w.get(b); w.byte(op) }
func (w *wasmCode) mask(width uint64) {
	if width == 32 {
		w.i64(0xffffffff)
		w.byte(0x83)
	}
}
func (w *wasmCode) booleanBit(shift uint64) { w.byte(0xad); w.i64(shift); w.byte(0x86) }

// EmitWasmBlock is the portable compiled-block backend. Unsupported operation
// families fail explicitly, so the engine cannot publish incomplete code.
func EmitWasmBlock(ops []Op, words int) ([]byte, error) {
	return emitWasmBlock(ops, words, nil)
}

// Layout is obtained from the compiled host's typed ABI, not assumed from
// amd64 offsets. WASM pointer fields are four bytes; architectural values stay
// eight bytes on both hosts.
type WasmMemoryLayout struct {
	PageSlots                                                             uint32 // zero preserves the native 64-slot geometry
	HashPages                                                             bool   // mix page-stride bits within the same 64 slots
	Retired, Status, Address, Clock, Pages                                uint32
	PageSize, PageNumber, PageData, PagePermissions, PageEpoch            uint32
	Total, Remaining, MemoryTotal, Blocks, LoopExits, LoopIterations      uint32
	MemoryRetired, Linked                                                 uint32
	ServiceWrite                                                          uint32
	ServiceSP, ServiceFP                                                  uint32 // zero disables private reentrant refill service
	LinkSource                                                            uint32 // optional last admitted PC; zero disables tracking
	DescriptorSize, DescriptorPC, DescriptorEntry, DescriptorInstructions uint32
	DescriptorHot                                                         uint32 // optional Reserved hint; zero disables demand upgrade
	BlockSets                                                             uint32 // zero selects the native-compatible 256 sets
	BlocksIndirect                                                        bool   // Blocks addresses a browser-owned slice header
}

func EmitWasmMemoryBlock(ops []Op, words int, layout WasmMemoryLayout) ([]byte, error) {
	return emitWasmBlock(ops, words, &layout)
}
func (w *wasmCode) contextValue(offset uint32, value uint64) {
	for i, field := range w.sessionOffsets {
		if field == offset {
			w.i64(value)
			w.set(w.sessionLocals[i])
			return
		}
	}
	w.get(1)
	w.i64(value)
	w.mem(0x37, 3, offset)
}
func (w *wasmCode) failIf(layout *WasmMemoryLayout, status uint64, address int) {
	w.byte(0x04)
	w.byte(0x40)
	if w.graphExit != nil {
		w.i64(status)
		w.set(w.graphExit.status)
		if address >= 0 {
			w.get(address)
			w.set(w.graphExit.address)
			w.i64(1)
			w.set(w.graphExit.hasAddress)
		}
	} else {
		w.flushState()
		w.contextValue(layout.Status, status)
		if address >= 0 {
			w.get(1)
			w.get(address)
			w.mem(0x37, 3, layout.Address)
		}
	}
	w.byte(0x0f)
	w.byte(0x0b)
}

func (w *wasmCode) flushState() {
	if w.graphExit != nil {
		return
	}
	w.flushSession()
	for slot, written := range w.stateWritten {
		if written {
			w.materializeFlags(slot)
			w.get(0)
			w.get(w.stateSlots[slot])
			w.mem(0x37, 3, uint32(slot*8))
		}
	}
}
func (w *wasmCode) ptr(local int, offset uint32) { w.get(local); w.byte(0xa7); w.mem(0x28, 2, offset) }

// wasmLiveOps drops overwritten architectural stores only within straight-line
// spans with no memory effect or possible exit. Loads are observation barriers
// for their slot. A backwards SSA walk then removes only total, unused values;
// all memory accesses, checkpoints and exit guards remain observable roots.
func wasmLiveOps(ops []Op) []bool {
	live := make([]bool, len(ops))
	deadStore := make([]bool, len(ops))
	var overwritten [256]bool
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		switch op.Kind {
		case MemoryLoad, MemoryStore, MemoryPairStore, Guard, RegionGuard, LoopContinue:
			clear(overwritten[:])
		case LoadState:
			overwritten[op.Imm] = false
		case StoreState:
			deadStore[i] = overwritten[op.Imm]
			overwritten[op.Imm] = true
		}
	}
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if effectOnly(op.Kind) && !deadStore[i] || op.Kind == MemoryLoad || op.Kind == PairHigh {
			live[i] = true
		}
		if !live[i] {
			continue
		}
		left, right := memoryOperands(op)
		if left {
			live[op.A] = true
		}
		if right {
			live[op.B] = true
		}
		if op.Kind == SelectValue {
			live[op.Imm] = true
		}
		if op.Kind == MemoryPairStore || carryKind(op.Kind) {
			live[op.Imm>>8] = true
		}
	}
	return live
}

type wasmScratch struct {
	body, module, section []byte
	fragment              bool // graph basic block: shared locals, no function boundary
	stateSlots            []int
	stateWritten          []bool
	flagSlots             []wasmFlagSlots
	graphExit             *wasmGraphExit
	flagFunction          int
	refillFunction        int
	pageWindow            []int
	sessionLocals         []int
	sessionOffsets        []uint32
}

func emitWasmBlock(ops []Op, words int, layout *WasmMemoryLayout) ([]byte, error) {
	return emitWasmBlockInto(ops, words, layout, nil)
}

// Borrowed output is consumed synchronously by the private browser installer.
// Exported emitters still return independent module bytes.
func emitWasmBlockInto(ops []Op, words int, layout *WasmMemoryLayout, scratch *wasmScratch) ([]byte, error) {
	return emitWasmCode(ops, words, layout, scratch, 0, false)
}

// Loop leaves commit completed iterations before their back edge, leaving the
// final iteration's precise prefix for the ordinary dispatcher to account.
// No instruction, memory guard, architectural store or checkpoint is skipped.
func emitWasmLoop(ops []Op, words, instructions int, layout *WasmMemoryLayout) ([]byte, error) {
	return emitWasmLoopMode(ops, words, instructions, layout, false)
}
func emitWasmLoopMode(ops []Op, words, instructions int, layout *WasmMemoryLayout, codeVersions bool) ([]byte, error) {
	if layout == nil || instructions < 1 || instructions > 256 || len(ops) == 0 || ops[len(ops)-1].Kind != LoopContinue {
		return nil, fmt.Errorf("invalid WASM loop dimensions")
	}
	progress := uint64(0)
	mode := Value(-1)
	for i, op := range ops {
		if op.Kind == Progress {
			if op.Imm < progress || op.Imm > uint64(instructions) || mode >= 0 && mode != op.B {
				return nil, fmt.Errorf("invalid WASM loop checkpoint")
			}
			progress, mode = op.Imm, op.B
		}
		if op.Kind == LoopContinue && (i != len(ops)-1 || progress != uint64(instructions)) {
			return nil, fmt.Errorf("invalid WASM loop terminator")
		}
	}
	return emitWasmCode(ops, words, layout, nil, instructions, codeVersions)
}

func emitWasmCode(ops []Op, words int, layout *WasmMemoryLayout, scratch *wasmScratch, loopInstructions int, codeVersions bool) ([]byte, error) {
	var err error
	if layout != nil && layout.PageSlots != 0 && (layout.PageSlots > 4096 || layout.PageSlots&(layout.PageSlots-1) != 0) {
		return nil, fmt.Errorf("invalid WASM page cache geometry")
	}
	if layout == nil {
		err = Validate(ops, words)
	} else {
		err = ValidateMemory(ops, words)
	}
	if err != nil {
		return nil, err
	}
	var body wasmCode
	body.ops, body.inline = ops, make([]bool, len(ops))
	if scratch != nil && scratch.fragment {
		body.flagSlots, body.flagFunction = scratch.flagSlots, scratch.flagFunction
		body.refillFunction = scratch.refillFunction
		body.pageWindow = scratch.pageWindow
		body.sessionLocals, body.sessionOffsets = scratch.sessionLocals, scratch.sessionOffsets
		body.graphExit = scratch.graphExit
	}
	deferredFlags := make([]bool, len(ops))
	var spans []wasmMemorySpan
	var spanAccesses []int
	if codeVersions && (scratch == nil || !scratch.fragment) {
		spans, spanAccesses = wasmMemorySpans(ops)
		for i := range spans {
			base := len(ops) + 22 + words + 3*i
			spans[i].page, spans[i].data, spans[i].fast = base, base+1, base+2
		}
	}
	body.spanProofs = spans
	live := wasmLiveOps(ops)
	uses := make([]int, len(ops))
	storeOnly := make([]bool, len(ops))
	for i := range storeOnly {
		storeOnly[i] = true
	}
	for i, op := range ops {
		if !live[i] {
			continue
		}
		left, right := memoryOperands(op)
		if left {
			uses[op.A]++
			if op.Kind != StoreState {
				storeOnly[op.A] = false
			}
		}
		if right {
			uses[op.B]++
			storeOnly[op.B] = false
		}
		if op.Kind == SelectValue {
			uses[op.Imm]++
			storeOnly[op.Imm] = false
		}
		if op.Kind == MemoryPairStore || carryKind(op.Kind) {
			v := op.Imm >> 8
			uses[v]++
			storeOnly[v] = false
		}
	}
	for i, op := range ops {
		deferredFlags[i] = len(body.flagSlots) != 0 && (op.Kind == ArithmeticFlags || op.Kind == LogicalFlags) && uses[i] == 1 && storeOnly[i]
		body.inline[i] = op.Kind == Const && (scratch != nil && scratch.fragment || int64(op.Imm) >= -64 && int64(op.Imm) <= 63 || uses[i] == 1 && storeOnly[i])
	}
	if scratch != nil && scratch.fragment {
		wasmInlineGraphValues(ops, live, uses, body.inline)
	}
	if scratch != nil {
		body.code = scratch.body[:0]
	} else {
		body.code = make([]byte, 0, len(ops)*24+128)
	}
	locals := len(ops) + 20
	if codeVersions {
		locals += words + 3*len(spans)
	}
	if scratch == nil || !scratch.fragment {
		body.u32(1)
		body.u32(uint32(locals))
		body.byte(0x7e)
	}
	loopRemaining, loopCount := len(ops)+18, len(ops)+19
	if loopInstructions != 0 {
		body.contextLoad(layout.Linked)
		body.byte(0x50)
		body.failIf(layout, 3, -1)
		body.contextLoad(layout.Remaining)
		body.set(loopRemaining)
		body.get(loopRemaining)
		body.i64(uint64(loopInstructions))
		body.byte(0x54)
		body.get(loopRemaining)
		body.i64(65536)
		body.byte(0x56)
		body.byte(0x72)
		body.failIf(layout, 3, -1)
	}
	if codeVersions {
		// The owner keeps RAM disjoint from this architectural state. Carry
		// each observed word in a WASM local; every exit publishes all words
		// written by the loop, including prior iterations' carried values.
		if scratch != nil && scratch.fragment {
			body.stateSlots, body.stateWritten = scratch.stateSlots, scratch.stateWritten
		} else {
			body.stateSlots, body.stateWritten = make([]int, words), make([]bool, words)
			for _, op := range ops {
				if op.Kind == LoadState || op.Kind == StoreState {
					body.stateSlots[op.Imm] = len(ops) + 22 + int(op.Imm)
					if op.Kind == StoreState {
						body.stateWritten[op.Imm] = true
					}
				}
			}
			for slot, local := range body.stateSlots {
				if local != 0 {
					body.get(0)
					body.mem(0x29, 3, uint32(slot*8))
					body.set(local)
				}
			}
		}
	}
	if scratch != nil && scratch.fragment {
		// Scratch locals are reused by unrelated CFG blocks. Never carry a
		// mapping proof from a different block or across its refill callback.
		for _, span := range spans {
			body.i64(0)
			body.set(span.data)
			body.i64(0)
			body.set(span.fast)
		}
	}
	if loopInstructions != 0 {
		body.byte(0x03)
		body.byte(0x40)
	}
	if layout != nil {
		body.contextValue(layout.Retired, 0)
		body.contextValue(layout.Status, 0)
		body.contextValue(layout.MemoryRetired, 0)
	}
	accesses, previousProgress := 0, uint64(0)
	pendingMemory, instructionProgress := false, false
	progressPending := false
	pendingRetired, pendingAccesses := uint64(0), uint64(0)
	for _, op := range ops {
		if op.Kind == Progress && op.B == 1 {
			instructionProgress = true
		}
	}
	for index, op := range ops {
		if !live[index] || deferredFlags[index] || body.inline[index] && op.Kind != Const {
			continue
		}
		a, b, result := int(op.A)+2, int(op.B)+2, index+2
		// Checkpoint counters are observed only by an exit or the caller.
		// Publish the latest exact pair before every possibly exiting access
		// or guard, and on normal return; pure straight-line spans need not
		// repeatedly overwrite those same two context words.
		if progressPending && (op.Kind == MemoryLoad || op.Kind == MemoryStore || op.Kind == MemoryPairStore || op.Kind == Guard || op.Kind == RegionGuard || op.Kind == LoopContinue) {
			body.contextValue(layout.Retired, pendingRetired)
			body.contextValue(layout.MemoryRetired, pendingAccesses)
			progressPending = false
		}
		switch op.Kind {
		case Progress:
			pendingRetired, pendingAccesses = op.Imm, uint64(accesses)
			progressPending = true
			if op.Imm > previousProgress {
				pendingMemory = false
			}
			previousProgress = op.Imm
			continue
		case LoopContinue:
			if loopInstructions != 0 {
				body.i64(1)
				body.set(loopCount)
				body.contextAdd(layout.LoopIterations, loopCount)
			}
			body.get(a)
			body.byte(0x50)
			body.failIf(layout, RegionExit, -1)
			if loopInstructions != 0 {
				// Preserve one final iteration for the dispatcher. The private
				// remaining local bounds repetition even if guest data aliases
				// a public context field in a generic memory configuration.
				body.get(loopRemaining)
				body.i64(uint64(loopInstructions * 2))
				body.byte(0x54)
				body.byte(0x04)
				body.byte(0x40)
				body.flushState()
				body.byte(0x0f)
				body.byte(0x0b)
				body.i64(uint64(loopInstructions))
				body.set(loopCount)
				body.contextAdd(layout.Total, loopCount)
				body.i64(pendingAccesses)
				body.set(loopCount)
				body.contextAdd(layout.MemoryTotal, loopCount)
				body.get(loopRemaining)
				body.i64(uint64(loopInstructions))
				body.byte(0x7d)
				body.set(loopRemaining)
				body.get(1)
				body.get(loopRemaining)
				body.mem(0x37, 3, layout.Remaining)
				body.byte(0x0c)
				body.u32(0)
			}
			continue
		case Guard, RegionGuard:
			body.get(a)
			body.byte(0x50)
			status := uint64(2)
			if op.Kind == RegionGuard {
				status = 4
			}
			body.failIf(layout, status, -1)
			continue
		case NativeElapsed:
			body.get(a)
			body.get(1)
			body.mem(0x29, 3, layout.Linked)
			body.byte(0x50)
			body.byte(0x04)
			body.byte(0x7e)
			body.i64(0)
			body.byte(0x05)
			body.get(1)
			body.mem(0x29, 3, layout.Total)
			body.byte(0x0b)
			body.i64(op.Imm)
			body.byte(0x7c)
			body.i64(uint64(op.B))
			body.byte(0x7e)
			body.byte(0x7c)
		case PairHigh:
			body.get(len(ops) + 6)
			body.byte(0xa7)
			body.mem(0x29, 0, 8)
		case MemoryLoad, MemoryStore, MemoryPairStore:
			if !instructionProgress || !pendingMemory {
				accesses++
			}
			pendingMemory = true
			width := op.Imm
			if op.Kind == MemoryPairStore {
				width = 16
			}
			page, data, clock, epoch := len(ops)+5, len(ops)+6, len(ops)+7, len(ops)+8
			span := -1
			if len(spanAccesses) != 0 {
				span = spanAccesses[index]
			}
			window := span < 0 && len(body.pageWindow) != 0
			if window {
				body.windowBegin(a, data, len(ops)+20, width)
			}
			if span >= 0 {
				if spans[span].first == index {
					body.spanGuard(spans[span], len(ops)+20, page, layout)
				}
				body.get(spans[span].fast)
				body.byte(0x50)
				body.byte(0x04)
				body.byte(0x40)
			}
			if layout.ServiceSP != 0 {
				body.byte(0x03)
				body.byte(0x40) // allocation-free refill retry guards only
			}
			body.get(1)
			body.i32(layout.Pages)
			body.byte(0x6a)
			body.pageSlot(a, layout)
			body.byte(0xa7)
			body.i32(layout.PageSize)
			body.byte(0x6c)
			body.byte(0x6a)
			body.byte(0xad)
			body.set(page)
			body.get(page)
			body.byte(0xa7)
			body.mem(0x29, 0, layout.PageNumber)
			body.get(a)
			body.i64(12)
			body.byte(0x88)
			body.byte(0x52)
			body.get(a)
			body.i64(4095)
			body.byte(0x83)
			body.i64(4096 - width)
			body.byte(0x56)
			body.byte(0x72)
			body.ptr(page, layout.PageData)
			body.byte(0x45)
			body.byte(0x72)
			body.get(page)
			body.byte(0xa7)
			body.mem(0x29, 0, layout.PagePermissions)
			if op.Kind == MemoryLoad {
				body.i64(1)
				body.byte(0x83)
				body.byte(0x50)
			} else {
				body.i64(6)
				body.byte(0x83)
				body.i64(2)
				body.byte(0x52)
			}
			body.byte(0x72)
			if layout.ServiceSP != 0 {
				body.refillFailIf(layout, a, op.Kind != MemoryLoad)
				body.byte(0x0b) // guard retry loop; no architectural effect inside
			} else {
				body.failIf(layout, 1, a)
			}
			// The explicitly admitted owner uses generations only for code.
			// Checked writable RAM cannot alias metadata or executable code;
			// mapping changes and later execute grants refresh its code stamp.
			if op.Kind != MemoryLoad && !codeVersions {
				body.get(1)
				body.mem(0x28, 2, layout.Clock)
				body.byte(0xad)
				body.set(clock)
				body.get(clock)
				body.byte(0x50)
				body.failIf(layout, 1, a)
				body.ptr(page, layout.PageEpoch)
				body.byte(0xad)
				body.set(epoch)
				body.get(epoch)
				body.byte(0x50)
				body.failIf(layout, 1, a)
				body.get(clock)
				body.byte(0xa7)
				body.mem(0x29, 0, 0)
				body.i64(^uint64(0))
				body.byte(0x51)
				body.failIf(layout, 1, a)
				body.get(clock)
				body.byte(0xa7)
				body.get(clock)
				body.byte(0xa7)
				body.mem(0x29, 0, 0)
				body.i64(1)
				body.byte(0x7c)
				body.mem(0x37, 0, 0)
				body.get(epoch)
				body.byte(0xa7)
				body.get(clock)
				body.byte(0xa7)
				body.mem(0x29, 0, 0)
				body.mem(0x37, 0, 0)
			}
			body.ptr(page, layout.PageData)
			body.byte(0xad)
			body.get(a)
			body.i64(4095)
			body.byte(0x83)
			body.byte(0x7c)
			body.set(data)
			if window {
				body.windowEnd(a, page, layout)
			}
			if span >= 0 {
				body.byte(0x05)
				body.get(spans[span].data)
				body.get(a)
				body.i64(4095)
				body.byte(0x83)
				body.byte(0x7c)
				body.set(data)
				body.byte(0x0b)
			}
			body.get(data)
			body.byte(0xa7)
			if op.Kind == MemoryLoad {
				code := byte(0x29)
				switch width {
				case 1:
					code = 0x31
				case 2:
					code = 0x33
				case 4:
					code = 0x35
				}
				body.mem(code, 0, 0)
			} else {
				body.get(b)
				code := byte(0x37)
				switch width {
				case 1:
					code = 0x3c
				case 2:
					code = 0x3d
				case 4:
					code = 0x3e
				}
				body.mem(code, 0, 0)
				if op.Kind == MemoryPairStore {
					body.get(data)
					body.byte(0xa7)
					body.get(int(op.Imm>>8) + 2)
					body.mem(0x37, 0, 8)
				}
				continue
			}
		case Const:
			if body.inline[index] {
				continue
			}
			body.i64(op.Imm)
		case LoadState:
			if codeVersions {
				body.materializeFlags(int(op.Imm))
				body.get(body.stateSlots[op.Imm])
			} else {
				body.get(0)
				body.mem(0x29, 3, uint32(op.Imm*8))
			}
		case StoreState:
			if codeVersions {
				if deferredFlags[op.A] {
					body.deferFlags(int(op.Imm), ops[op.A])
					continue
				}
				body.forgetFlags(int(op.Imm))
				body.get(a)
				body.set(body.stateSlots[op.Imm])
			} else {
				body.get(0)
				body.get(a)
				body.mem(0x37, 3, uint32(op.Imm*8))
			}
			continue
		case Add, Sub, Mul, And, Or, Xor, Equal, Less:
			code := byte(0x7c)
			switch op.Kind {
			case Sub:
				code = 0x7d
			case Mul:
				code = 0x7e
			case And:
				code = 0x83
			case Or:
				code = 0x84
			case Xor:
				code = 0x85
			case Equal:
				code = 0x51
			case Less:
				code = 0x54
			}
			body.binary(a, b, code)
			if op.Kind == Equal || op.Kind == Less {
				body.byte(0xad)
			}
		case Shl, Shr:
			if op.Imm >= 64 {
				body.i64(0)
			} else {
				body.get(a)
				body.i64(op.Imm)
				if op.Kind == Shl {
					body.byte(0x86)
				} else {
					body.byte(0x88)
				}
			}
		case SelectValue:
			body.get(b)
			body.get(int(op.Imm) + 2)
			body.get(a)
			body.i64(0)
			body.byte(0x52)
			body.byte(0x1b)
		case VariableShl, VariableShr, ArithmeticShr, RotateRight:
			body.get(a)
			body.mask(op.Imm)
			if op.Kind == ArithmeticShr && op.Imm == 32 {
				body.i64(32)
				body.byte(0x86)
				body.i64(32)
				body.byte(0x87)
			}
			body.get(b)
			body.i64(op.Imm - 1)
			body.byte(0x83)
			switch op.Kind {
			case VariableShl:
				body.byte(0x86)
			case VariableShr:
				body.byte(0x88)
			case ArithmeticShr:
				body.byte(0x87)
			case RotateRight:
				if op.Imm == 64 {
					body.byte(0x8a)
				} else {
					body.byte(0x88)
					body.get(a)
					body.mask(32)
					body.i64(32)
					body.get(b)
					body.i64(31)
					body.byte(0x83)
					body.byte(0x7d)
					body.byte(0x86)
					body.byte(0x84)
				}
			}
			body.mask(op.Imm)
		case LogicalFlags, ArithmeticFlags, LogicalCondition, ArithmeticCondition:
			condition := op.Kind == LogicalCondition || op.Kind == ArithmeticCondition
			if condition && body.directCondition(op, a, b) {
				break
			}
			if condition {
				op.Kind -= 2
			}
			x, y, r := len(ops)+2, len(ops)+3, len(ops)+4
			body.packedFlags(op, a, b, x, y, r)
			if condition {
				body.set(result)
				body.condition(result, int(op.Imm>>8))
			}
		default:
			if !body.integer(op, a, b, len(ops)+10) {
				return nil, fmt.Errorf("unsupported WASM RFE operation %d", op.Kind)
			}
		}
		body.set(result)
	}
	if progressPending {
		body.contextValue(layout.Retired, pendingRetired)
		body.contextValue(layout.MemoryRetired, pendingAccesses)
	}
	if loopInstructions != 0 {
		body.byte(0x0b)
	}
	if scratch != nil && scratch.fragment {
		return body.code, nil
	}
	body.flushState()
	body.byte(0x0b)
	var module wasmCode
	if scratch != nil {
		module.code = scratch.module[:0]
	} else {
		module.code = make([]byte, 0, len(body.code)+96)
	}
	module.code = append(module.code, 0, 97, 115, 109, 1, 0, 0, 0)
	service := layout != nil && layout.ServiceSP != 0
	types := []byte{1, 0x60, 2, 0x7f, 0x7f, 0}
	if service {
		types[0] = 2
		types = append(types, 0x60, 9, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 3, 0x7f, 0x7f, 0x7f)
	}
	module.section(1, types)
	var imports wasmCode
	if service {
		imports.u32(2)
	} else {
		imports.u32(1)
	}
	imports.name("env")
	imports.name("memory")
	imports.byte(2)
	imports.byte(0)
	imports.u32(0)
	if service {
		imports.name("env")
		imports.name("service")
		imports.byte(0)
		imports.u32(1)
	}
	module.section(2, imports.code)
	module.section(3, []byte{1, 0})
	var exports wasmCode
	exports.u32(1)
	exports.name("run")
	exports.byte(0)
	if service {
		exports.u32(1)
	} else {
		exports.u32(0)
	}
	module.section(7, exports.code)
	var code wasmCode
	if scratch != nil {
		code.code = scratch.section[:0]
	} else {
		code.code = make([]byte, 0, len(body.code)+6)
	}
	code.u32(1)
	code.u32(uint32(len(body.code)))
	code.code = append(code.code, body.code...)
	module.section(10, code.code)
	if scratch != nil {
		scratch.body = body.code
		scratch.module = module.code
		scratch.section = code.code
	}
	return module.code, nil
}
