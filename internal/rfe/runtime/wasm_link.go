package runtime

import "fmt"

// The dispatcher runs entirely in WASM. Its private table is worker-owned;
// public descriptors grant no authority without a matching immutable proof.
func wasmFunctionModule(body []byte, proof, table bool) []byte {
	var m, imports, exports, code wasmCode
	m.code = []byte{0, 97, 115, 109, 1, 0, 0, 0}
	types := []byte{2, 0x60, 2, 0x7f, 0x7f, 0, 0x60, 4, 0x7e, 0x7e, 0x7e, 0x7f, 1, 0x7f}
	m.section(1, types)
	if !proof {
		count := uint32(1)
		if table {
			count++
		}
		imports.u32(count)
		imports.name("env")
		imports.name("memory")
		imports.byte(2)
		imports.byte(0)
		imports.u32(0)
		if table {
			imports.name("env")
			imports.name("table")
			imports.byte(1)
			imports.byte(0x70)
			imports.byte(0)
			imports.u32(0)
		}
		m.section(2, imports.code)
	}
	typeIndex := byte(0)
	name := "run"
	if proof {
		typeIndex = 1
		name = "prove"
	}
	m.section(3, []byte{1, typeIndex})
	exports.u32(1)
	exports.name(name)
	exports.byte(0)
	exports.u32(0)
	m.section(7, exports.code)
	code.u32(1)
	code.u32(uint32(len(body)))
	code.code = append(code.code, body...)
	m.section(10, code.code)
	return m.code
}
func emitWasmProof() []byte {
	// One code shape, separate immutable imported globals for every admission.
	// The worker can compile this tiny module once without sharing authority:
	// each instance still captures its own exact PC, entry, count and dimensions.
	var m, imports, exports, code, w wasmCode
	m.code = []byte{0, 97, 115, 109, 1, 0, 0, 0}
	m.section(1, []byte{1, 0x60, 4, 0x7e, 0x7e, 0x7e, 0x7f, 1, 0x7f})
	imports.u32(4)
	for i, name := range [4]string{"pc", "entry", "instructions", "words"} {
		imports.name("proof")
		imports.name(name)
		imports.byte(3) // global
		kind := byte(0x7e)
		if i == 3 {
			kind = 0x7f
		}
		imports.byte(kind)
		imports.byte(0) // immutable, never emulator-memory-backed
	}
	m.section(2, imports.code)
	m.section(3, []byte{1, 0})
	exports.u32(1)
	exports.name("prove")
	exports.byte(0)
	exports.u32(0)
	m.section(7, exports.code)
	w.u32(0)
	for i := 0; i < 4; i++ {
		w.get(i)
		w.byte(0x23) // global.get
		w.u32(uint32(i))
		compare := byte(0x51)
		if i == 3 {
			compare = 0x46
		}
		w.byte(compare)
		if i != 0 {
			w.byte(0x71)
		}
	}
	w.byte(0x0b)
	code.u32(1)
	code.u32(uint32(len(w.code)))
	code.code = append(code.code, w.code...)
	m.section(10, code.code)
	return m.code
}
func (w *wasmCode) contextLoad(offset uint32) {
	for i, field := range w.sessionOffsets {
		if field == offset {
			w.get(w.sessionLocals[i])
			return
		}
	}
	w.get(1)
	w.mem(0x29, 3, offset)
}
func (w *wasmCode) contextAdd(offset uint32, local int) {
	w.get(1)
	w.contextLoad(offset)
	w.get(local)
	w.byte(0x7c)
	w.mem(0x37, 3, offset)
}
func EmitWasmDispatcher(words, pcSlot int, l WasmMemoryLayout) ([]byte, error) {
	if words < 1 || words > 256 || pcSlot < 0 || pcSlot >= words || l.DescriptorSize == 0 || l.MemoryRetired == l.Retired {
		return nil, fmt.Errorf("invalid WASM dispatcher layout")
	}
	sets := uint64(l.BlockSets)
	if sets == 0 {
		sets = 256
	}
	if sets > 4096 || sets&(sets-1) != 0 {
		return nil, fmt.Errorf("invalid WASM descriptor set count")
	}
	coherentProgress := wasmDisjointProgress(l)
	var w wasmCode
	// All scratch is i64; context and state parameters are i32.
	w.u32(1)
	if l.DescriptorHot != 0 {
		w.u32(10)
	} else {
		w.u32(9)
	}
	w.byte(0x7e)
	const pc = 2
	const descriptor = 3
	const entry = 4
	const instructions = 5
	const slot = 6
	const found = 7
	const retired = 8
	const mem = 9
	const index = 10
	const demand = 11
	w.byte(0x03)
	w.byte(0x40) // bounded dispatcher loop
	w.get(0)
	w.mem(0x29, 3, uint32(pcSlot*8))
	w.set(pc)
	w.get(pc)
	w.i64(2)
	w.byte(0x88)
	w.get(pc)
	w.i64(12)
	w.byte(0x88)
	w.byte(0x85)
	w.i64(sets - 1)
	w.byte(0x83)
	w.set(slot)
	w.i64(0)
	w.set(found)
	for way := uint64(0); way < 4; way++ {
		w.get(found)
		w.byte(0x50)
		w.byte(0x04)
		w.byte(0x40)
		w.get(1)
		if l.BlocksIndirect {
			w.mem(0x28, 2, l.Blocks)
			w.byte(0xad)
			w.i64(way * sets * uint64(l.DescriptorSize))
		} else {
			w.byte(0xad)
			w.i64(uint64(l.Blocks) + way*sets*uint64(l.DescriptorSize))
		}
		w.byte(0x7c)
		w.get(slot)
		w.i64(uint64(l.DescriptorSize))
		w.byte(0x7e)
		w.byte(0x7c)
		w.set(descriptor)
		w.get(descriptor)
		w.byte(0xa7)
		w.mem(0x29, 3, l.DescriptorPC)
		w.get(pc)
		w.byte(0x51)
		w.get(descriptor)
		w.byte(0xa7)
		w.mem(0x29, 3, l.DescriptorInstructions)
		w.byte(0x50)
		w.byte(0x45)
		w.byte(0x71)
		w.byte(0x04)
		w.byte(0x40)
		w.get(descriptor)
		w.byte(0xa7)
		w.mem(0x29, 3, l.DescriptorEntry)
		w.set(entry)
		w.get(descriptor)
		w.byte(0xa7)
		w.mem(0x29, 3, l.DescriptorInstructions)
		w.set(instructions)
		w.i64(1)
		w.set(found)
		w.byte(0x0b)
		w.byte(0x0b)
	}
	w.get(found)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x40)
	w.byte(0x0f)
	w.byte(0x0b)
	w.get(instructions)
	w.contextLoad(l.Remaining)
	w.byte(0x56)
	w.byte(0x04)
	w.byte(0x40)
	w.byte(0x0f)
	w.byte(0x0b)
	w.get(instructions)
	w.i64(256)
	w.byte(0x56)
	w.failIf(&l, 3, -1)
	w.get(entry)
	w.i64(15)
	w.byte(0x83)
	w.byte(0x50)
	w.byte(0x45)
	w.failIf(&l, 3, -1)
	// Maximum 16384 immutable entries. Bounds precede call_indirect, including
	// proof slot occupancy, which only the host can install.
	w.get(entry)
	w.i64(16)
	w.byte(0x54)
	w.get(entry)
	w.i64(16384 * 16)
	w.byte(0x5a)
	w.byte(0x72)
	w.failIf(&l, 3, -1)
	w.get(entry)
	w.i64(3)
	w.byte(0x88)
	w.set(index) // 2*(entry/16)
	w.get(index)
	w.i64(1)
	w.byte(0x7c)
	w.byte(0xa7)
	w.byte(0x25)
	w.u32(0)
	w.byte(0xd1)
	w.failIf(&l, 3, -1)
	w.get(pc)
	w.get(entry)
	w.get(instructions)
	w.i32(uint32(words))
	w.get(index)
	w.i64(1)
	w.byte(0x7c)
	w.byte(0xa7)
	w.byte(0x11)
	w.u32(1)
	w.u32(0)
	w.byte(0x45)
	w.failIf(&l, 3, -1)
	if l.DescriptorHot != 0 {
		// Count only after the full ordinary immutable proof has succeeded.
		// Reserved is an untrusted optimization hint, never execution authority.
		w.get(descriptor)
		w.byte(0xa7)
		w.mem(0x29, 3, l.DescriptorHot)
		w.set(demand)
		w.get(demand)
		w.i64(^uint64(0))
		w.byte(0x52)
		w.byte(0x04)
		w.byte(0x40)
		w.get(demand)
		w.byte(0x50)
		w.byte(0x04)
		w.byte(0x40)
		w.i64(256)
		w.set(demand)
		w.byte(0x0b)
		w.get(descriptor)
		w.byte(0xa7)
		w.get(demand)
		w.i64(1)
		w.byte(0x7d)
		w.byte(0x22)
		w.u32(demand)
		w.mem(0x37, 3, l.DescriptorHot)
		w.get(demand)
		w.byte(0x50)
		w.contextLoad(l.Total)
		w.byte(0x50)
		w.byte(0x45)
		w.byte(0x71)
		w.byte(0x04)
		w.byte(0x40)
		w.contextValue(l.Status, 5)
		w.byte(0x0f)
		w.byte(0x0b)
		w.byte(0x0b)
	}
	// Record only an admitted source, not the successor that failed lookup.
	// This is an optional engine hint and never substitutes for a target proof.
	if l.LinkSource != 0 {
		w.get(1)
		w.get(pc)
		w.mem(0x37, 3, l.LinkSource)
	}
	// The private proof and table target cannot change during this synchronous
	// call. Repeated execution at the identical PC can keep that admission in
	// WASM locals; every access guard and retirement/budget check still runs.
	w.byte(0x03)
	w.byte(0x40)
	w.get(1)
	w.get(instructions)
	w.mem(0x37, 3, l.Retired)
	w.contextValue(l.MemoryRetired, 0)
	w.contextValue(l.Status, 0)
	w.get(0)
	w.get(1)
	w.get(index)
	w.byte(0xa7)
	w.byte(0x11)
	w.u32(0)
	w.u32(0)
	// Descriptor/hash/entry scratch is dead after admission. Reuse it rather
	// than increasing the dispatcher's frame. PC remains memory-observed AFTER
	// progress/status stores because generic state can alias those words.
	if coherentProgress {
		w.contextLoad(l.Status)
		w.set(entry)
	}
	w.contextLoad(l.Retired)
	w.set(retired)
	w.contextLoad(l.MemoryRetired)
	w.set(mem)
	w.binary(retired, instructions, 0x56)
	w.binary(mem, retired, 0x56)
	w.byte(0x72)
	w.failIf(&l, 3, -1)
	w.dispatchProgressLoad(l.Status, coherentProgress, entry)
	w.byte(0x50)
	w.binary(retired, instructions, 0x52)
	w.byte(0x71)
	w.failIf(&l, 3, -1)
	// A successful side exit must consume a nonzero prefix; otherwise a
	// descriptor cycle could spin without reducing the bounded budget.
	w.dispatchProgressLoad(l.Status, coherentProgress, entry)
	w.i64(RegionExit)
	w.byte(0x51)
	w.get(retired)
	w.byte(0x50)
	w.byte(0x71)
	w.failIf(&l, 3, -1)
	w.contextAdd(l.Total, retired)
	w.contextAdd(l.MemoryTotal, mem)
	w.get(1)
	w.contextLoad(l.Remaining)
	w.get(retired)
	w.byte(0x7d)
	if coherentProgress {
		w.byte(0x22)
		w.u32(found)
	}
	w.mem(0x37, 3, l.Remaining)
	// A checkpointed region side exit is a valid dynamic successor, not a
	// memory fault. Keep distinct successors in the proof-checked dispatcher.
	// An explicit exit to the same root still returns to the caller.
	w.dispatchProgressLoad(l.Status, coherentProgress, entry)
	w.i64(RegionExit)
	w.byte(0x51)
	w.get(0)
	w.mem(0x29, 3, uint32(pcSlot*8))
	w.get(pc)
	w.byte(0x52)
	w.byte(0x71)
	w.byte(0x04)
	w.byte(0x40)
	one := found
	if coherentProgress {
		one = slot
	}
	w.i64(1)
	w.set(one)
	w.contextAdd(l.LoopExits, one)
	w.contextValue(l.Status, 0)
	if coherentProgress {
		w.i64(0)
		w.set(entry)
	}
	w.byte(0x0b)
	w.dispatchProgressLoad(l.Status, coherentProgress, entry)
	w.byte(0x50)
	w.byte(0x45)
	w.byte(0x04)
	w.byte(0x40)
	w.byte(0x0f)
	w.byte(0x0b)
	w.dispatchProgressLoad(l.Remaining, coherentProgress, found)
	w.byte(0x50)
	w.byte(0x04)
	w.byte(0x40)
	w.byte(0x0f)
	w.byte(0x0b)
	w.get(0)
	w.mem(0x29, 3, uint32(pcSlot*8))
	w.get(pc)
	w.byte(0x51)
	w.dispatchProgressLoad(l.Remaining, coherentProgress, found)
	w.get(instructions)
	w.byte(0x5a) // remaining >= one immutable block
	w.byte(0x71)
	w.byte(0x0d)
	w.u32(0) // same admitted target; no host or descriptor mutation intervened
	w.byte(0x0c)
	w.u32(1) // different successor: validate a fresh descriptor and proof
	w.byte(0x0b)
	w.byte(0x0b)
	w.byte(0x0b)
	return wasmFunctionModule(w.code, false, true), nil
}

// Loads can be carried over progress stores only for disjoint ABI words.
// Unusual/overlapping caller layouts retain the original sequence of loads.
func wasmDisjointProgress(l WasmMemoryLayout) bool {
	offsets := [7]uint32{l.Retired, l.MemoryRetired, l.Status, l.Total, l.MemoryTotal, l.Remaining, l.LoopExits}
	for i, a := range offsets {
		for j, b := range offsets {
			if i < j && uint64(a) < uint64(b)+8 && uint64(b) < uint64(a)+8 {
				return false
			}
		}
	}
	return true
}
func (w *wasmCode) dispatchProgressLoad(offset uint32, coherent bool, local int) {
	if coherent {
		w.get(local)
	} else {
		w.contextLoad(offset)
	}
}
