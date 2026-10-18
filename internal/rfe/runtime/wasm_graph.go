package runtime

import "fmt"

// GraphBlock is one independently checkpointed architectural basic block.
// Successors are hints for internal control flow, never guessed dynamic edges.
// Every transfer compares the actual full architectural PC before taking it.
type GraphBlock struct {
	PC             uint64
	Instructions   int
	Ops            []Op
	Successors     [4]uint64
	SuccessorCount int
}

// emitWasmGraph coalesces already validated blocks into one bounded module.
// Basic-block fragments retain exact instruction checkpoints. Only completed internal
// transitions are committed to the session; the final leaf's precise prefix is
// left for the ordinary proof-checked caller, exactly as with resident loops.
func emitWasmGraph(blocks []GraphBlock, words, pcSlot int, layout WasmMemoryLayout) ([]byte, error) {
	if len(blocks) < 2 || len(blocks) > 256 || words < 1 || words > 256 || pcSlot < 0 || pcSlot >= words {
		return nil, fmt.Errorf("invalid WASM graph dimensions")
	}
	total := 0
	for i, block := range blocks {
		if block.Instructions < 1 || block.Instructions > 16 || block.SuccessorCount < 0 || block.SuccessorCount > len(block.Successors) {
			return nil, fmt.Errorf("invalid WASM graph block")
		}
		if err := ValidateMemory(block.Ops, words); err != nil {
			return nil, err
		}
		total += block.Instructions
		for j := 0; j < i; j++ {
			if blocks[j].PC == block.PC {
				return nil, fmt.Errorf("duplicate WASM graph PC")
			}
		}
		progress := uint64(0)
		for _, op := range block.Ops {
			if op.Kind == LoopContinue || op.Kind == RegionGuard {
				return nil, fmt.Errorf("nested WASM graph region")
			}
			if op.Kind == Progress {
				if op.Imm < progress || op.Imm > uint64(block.Instructions) {
					return nil, fmt.Errorf("invalid WASM graph progress")
				}
				progress = op.Imm
			}
		}
		if progress != uint64(block.Instructions) {
			return nil, fmt.Errorf("missing WASM graph checkpoint")
		}
	}
	if total > 256 {
		return nil, fmt.Errorf("WASM graph instruction bound exceeded")
	}
	blocks = wasmGraphOrder(blocks)
	// SSA and memory scratch may overlap between independent blocks, but
	// architectural words live beyond every block's largest scratch extent.
	stateBase := 2
	for _, block := range blocks {
		end := len(block.Ops) + 22 + words
		if end > stateBase {
			stateBase = end
		}
	}
	slots, written := make([]int, words), make([]bool, words)
	for i := range slots {
		slots[i] = stateBase + i
	}
	for _, block := range blocks {
		for _, op := range block.Ops {
			if op.Kind == StoreState && op.Imm < uint64(words) {
				written[op.Imm] = true
			}
		}
	}
	index, retired, mem, remaining := stateBase+words, stateBase+words+1, stateBase+words+2, stateBase+words+3
	flagWords := make([]bool, words)
	for _, block := range blocks {
		for _, op := range block.Ops {
			if op.Kind == StoreState && (block.Ops[op.A].Kind == ArithmeticFlags || block.Ops[op.A].Kind == LogicalFlags) {
				flagWords[op.Imm] = true
			}
		}
	}
	flagSlots := make([]wasmFlagSlots, words)
	nextLocal, flagCount := remaining+1, 0
	for slot, used := range flagWords {
		if used {
			flagSlots[slot] = wasmFlagSlots{nextLocal, nextLocal + 1, nextLocal + 2}
			nextLocal += 3
			flagCount++
		}
	}
	if flagCount == 0 {
		flagSlots = nil
	}
	flagFunction := 1
	if layout.ServiceSP != 0 {
		flagFunction++
	}
	refillFunction := 0
	if layout.ServiceSP != 0 {
		refillFunction = flagFunction
		if flagCount != 0 {
			refillFunction++
		}
	}
	pageWindow := []int{nextLocal, nextLocal + 1}
	nextLocal += 2
	var exit *wasmGraphExit
	if len(blocks) >= 2 {
		exit = &wasmGraphExit{nextLocal, nextLocal + 1, nextLocal + 2}
		nextLocal += 3
	}
	var sessionLocals []int
	var sessionOffsets []uint32
	if exit != nil && wasmGraphSessionSafe(layout) {
		sessionLocals = []int{nextLocal, nextLocal + 1}
		sessionOffsets = []uint32{layout.Retired, layout.MemoryRetired}
		nextLocal += 2
		if wasmGraphStatusSafe(layout) {
			// Reuse the consumed header index until a checked backedge.
			sessionLocals = append(sessionLocals, index)
			sessionOffsets = append(sessionOffsets, layout.Status)
		}
	}
	var w wasmCode
	w.sessionLocals, w.sessionOffsets = sessionLocals, sessionOffsets
	w.graphExit = exit
	w.pageWindow = pageWindow
	w.stateSlots, w.stateWritten = slots, written
	w.flagSlots, w.flagFunction = flagSlots, flagFunction
	w.u32(1)
	w.u32(uint32(nextLocal - 2)) // parameters 0..1 are not locals
	w.byte(0x7e)
	headerEnd := len(w.code)
	for slot, local := range slots {
		w.get(0)
		w.mem(0x29, 3, uint32(slot*8))
		w.set(local)
	}
	w.contextValue(layout.Retired, 0)
	w.contextValue(layout.MemoryRetired, 0)
	w.contextLoad(layout.Linked)
	w.byte(0x50)
	w.failIf(&layout, 3, -1)
	w.contextLoad(layout.Remaining)
	w.i64(uint64(total))
	w.byte(0x54)
	w.contextLoad(layout.Remaining)
	w.i64(NativeSessionLimit)
	w.byte(0x56)
	w.byte(0x72)
	w.failIf(&layout, 3, -1)
	w.byte(0x03)
	w.byte(0x40)
	for range blocks {
		w.byte(0x02)
		w.byte(0x40)
	}
	w.get(index)
	w.byte(0xa7)
	w.byte(0x0e)
	w.u32(uint32(len(blocks)))
	for i := range blocks {
		w.u32(uint32(i))
	}
	w.u32(0)
	for i, block := range blocks {
		w.byte(0x0b)
		selfLoop := false
		for _, target := range block.Successors[:block.SuccessorCount] {
			if target == block.PC {
				selfLoop = true
			}
		}
		extraDepth := 0
		if selfLoop {
			w.byte(0x03)
			w.byte(0x40)
			extraDepth = 1
		}
		fragment, err := emitWasmCode(block.Ops, words, &layout, &wasmScratch{fragment: true, stateSlots: slots, stateWritten: written, flagSlots: flagSlots, flagFunction: flagFunction, refillFunction: refillFunction, pageWindow: pageWindow, graphExit: exit, sessionLocals: sessionLocals, sessionOffsets: sessionOffsets}, 0, true)
		if err != nil {
			return nil, err
		}
		w.code = append(w.code, fragment...)
		w.contextLoad(layout.Status)
		w.byte(0x50)
		w.byte(0x45)
		w.byte(0x04)
		w.byte(0x40)
		w.flushState()
		w.byte(0x0f)
		w.byte(0x0b)
		w.contextLoad(layout.Retired)
		w.set(retired)
		w.contextLoad(layout.MemoryRetired)
		w.set(mem)
		w.contextLoad(layout.Remaining)
		w.set(remaining)
		w.get(retired)
		w.i64(uint64(block.Instructions))
		w.byte(0x52)
		w.binary(mem, retired, 0x56)
		w.byte(0x72)
		w.failIf(&layout, 3, -1)
		for _, target := range block.Successors[:block.SuccessorCount] {
			next := -1
			for j := range blocks {
				if blocks[j].PC == target {
					next = j
					break
				}
			}
			if next < 0 {
				continue
			}
			w.get(slots[pcSlot])
			w.i64(target)
			w.byte(0x51)
			w.get(remaining)
			w.i64(uint64(block.Instructions + blocks[next].Instructions))
			w.byte(0x5a)
			w.byte(0x71)
			w.byte(0x04)
			w.byte(0x40)
			w.contextAdd(layout.Total, retired)
			w.contextAdd(layout.MemoryTotal, mem)
			w.get(1)
			w.get(remaining)
			w.get(retired)
			w.byte(0x7d)
			w.mem(0x37, 3, layout.Remaining)
			if next == i && selfLoop {
				w.byte(0x0c)
				w.u32(1) // selected self-edge's if surrounds the local loop
			} else if next > i {
				w.byte(0x0c)
				w.u32(uint32(next - i + extraDepth))
			} else {
				w.i64(uint64(next))
				w.set(index)
				w.byte(0x0c)
				w.u32(uint32(len(blocks) - i + extraDepth))
			}
			w.byte(0x0b)
		}
		if wasmGraphLinkSourceSafe(layout) {
			w.contextValue(layout.LinkSource, block.PC)
		}
		w.contextValue(layout.Status, RegionExit)
		w.flushState()
		w.byte(0x0f)
		if selfLoop {
			w.byte(0x0b)
		}
	}
	w.byte(0x0b)
	if exit != nil {
		outlined, err := wasmGraphExitBranches(w.code[headerEnd:])
		if err != nil {
			return nil, err
		}
		w.code = append(w.code[:headerEnd], 0x02, 0x40)
		w.code = append(w.code, outlined...)
		w.byte(0x0b) // common exit block
		w.graphExit = nil
		w.flushState()
		// failIf writes status/address AFTER architectural publication. Keep that
		// order even for a generic state pointer aliasing context storage. Refill
		// failure and ordinary region exits already published their status.
		w.get(exit.status)
		w.byte(0x50)
		w.byte(0x45)
		w.byte(0x04)
		w.byte(0x40)
		w.get(1)
		w.get(exit.status)
		w.mem(0x37, 3, layout.Status)
		w.get(exit.hasAddress)
		w.byte(0x50)
		w.byte(0x45)
		w.byte(0x04)
		w.byte(0x40)
		w.get(1)
		w.get(exit.address)
		w.mem(0x37, 3, layout.Address)
		w.byte(0x0b)
		w.byte(0x0b)
		w.byte(0x0b)
	} else {
		w.byte(0x0b)
	}
	// One ordinary entry/proof and the same service import, with internal
	// branches rather than function calls or public descriptor transitions.
	var m, imports, exports, code wasmCode
	m.code = []byte{0, 97, 115, 109, 1, 0, 0, 0}
	service := layout.ServiceSP != 0
	types := []byte{1, 0x60, 2, 0x7f, 0x7f, 0}
	base := 0
	if service {
		base = 1
		types[0] = 2
		types = append(types, 0x60, 9, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 3, 0x7f, 0x7f, 0x7f)
	}
	flagType := types[0]
	if flagCount != 0 {
		types[0]++
		types = append(types, 0x60, 3, 0x7e, 0x7e, 0x7e, 1, 0x7e)
	}
	refillType := types[0]
	if service {
		types[0]++
		types = append(types, 0x60, 3, 0x7f, 0x7f, 0x7e, 1, 0x7f)
	}
	m.section(1, types)
	imports.u32(uint32(base + 1))
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
	m.section(2, imports.code)
	var declarations wasmCode
	functionCount := uint32(1)
	if flagCount != 0 {
		functionCount = 2
	}
	if service {
		functionCount++
	}
	declarations.u32(functionCount)
	declarations.u32(0)
	if flagCount != 0 {
		declarations.u32(uint32(flagType))
	}
	if service {
		declarations.u32(uint32(refillType))
	}
	m.section(3, declarations.code)
	exports.u32(1)
	exports.name("run")
	exports.byte(0)
	exports.u32(uint32(base))
	m.section(7, exports.code)
	code.u32(functionCount)
	code.u32(uint32(len(w.code)))
	code.code = append(code.code, w.code...)
	if flagCount != 0 {
		for _, body := range wasmFlagFunctions(base) {
			code.u32(uint32(len(body)))
			code.code = append(code.code, body...)
		}
	}
	if service {
		refillBody := wasmGraphRefillFunction(layout)
		code.u32(uint32(len(refillBody)))
		code.code = append(code.code, refillBody...)
	}
	m.section(10, code.code)
	return m.code, nil
}
