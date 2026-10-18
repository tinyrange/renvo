//go:build renvo && browser

package runtime

import "fmt"

// Run actual generated loop WASM with an odd budget and a fault after a fully
// completed iteration. The caller accounts only the final precise prefix.
func browserLoopProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	if err = n.PrepareLinks(5, 0); err != nil {
		return err
	}
	var b Builder
	b.MemoryInstructionProgress = true
	b.Checkpoint(5, 0)
	count := b.Binary(Add, b.Load(1), b.Constant(1))
	b.Store(1, count)
	b.Store(0, b.Constant(0))
	b.Checkpoint(5, 1)
	address := b.Load(2)
	b.MemoryStore(address, count, 8)
	b.Store(2, b.Binary(Add, address, b.Constant(8)))
	b.Store(3, b.Elapsed(b.Load(4), 2, 2))
	b.Checkpoint(5, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(5), 5, 2)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 5, 2); err != nil {
		return err
	}
	var data [4096]byte
	clock, epoch := uint64(5), uint64(3)
	m := &MemoryContext{}
	m.Clock = &clock
	m.Fill(1, &data, 3, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	state := []uint64{0, 0, 4096, 0, 100}
	// Loop entries require the private dispatcher's budget ABI. A caller's
	// public linked flag must not turn CallMemory into an admitted loop call.
	m.wasmLinked = 1
	if n.CallMemory(entry, state, m) == nil || state[1] != 0 {
		return fmt.Errorf("WASM loop accepted ordinary memory entry")
	}
	m.wasmLinked = 0
	if err = n.RunLinkedSession(state, m, 5); err != nil {
		return err
	}
	if state[1] != 2 || state[2] != 4112 || state[3] != 108 || m.Total != 4 || m.Remaining != 1 || m.MemoryTotal != 2 || m.LoopIterations != 2 || m.Status != 0 || clock != 7 || epoch != 7 {
		return fmt.Errorf("WASM resident loop budget/elapsed mismatch")
	}
	state[0], state[1], state[2], state[3] = 0, 0, 8184, 0
	if err = n.RunLinkedSession(state, m, 64); err != nil {
		return err
	}
	if state[1] != 2 || state[2] != 8192 || state[3] != 104 || m.Total != 3 || m.Remaining != 61 || m.MemoryTotal != 1 || m.Retired != 1 || m.wasmMemoryRetired != 0 || m.LoopIterations != 1 || m.Status != 1 || m.Address != 8192 || data[4088] != 1 || clock != 8 || epoch != 8 {
		return fmt.Errorf("WASM resident loop precise fault mismatch")
	}
	link := m.LookupLink(0)
	link.Instructions = 1
	state[0], state[1], state[2] = 0, 0, 4096
	if n.CallLinked(state, m, 64) == nil || m.Total != 0 || state[1] != 0 || clock != 8 {
		return fmt.Errorf("WASM resident loop accepted forged instruction count")
	}
	return browserOwnedLoopProbe(n, b.FinishMemory(5), m)
}
