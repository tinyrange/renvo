//go:build renvo && browser

package runtime

import "fmt"

// This exercises real table calls and proof rejection in the browser, not the
// portable evaluator. A failed descriptor must have no architectural effects.
func browserLinkProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	if err = n.PrepareLinks(4, 0); err != nil {
		return err
	}
	if err = browserLinkCacheProbe(n); err != nil {
		return err
	}
	first := []Op{{Kind: LoadState, Imm: 1}, {Kind: Const, Imm: 1}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 1}, {Kind: Const, Imm: 4}, {Kind: StoreState, A: 4, Imm: 0}}
	second := []Op{{Kind: LoadState, Imm: 1}, {Kind: Const, Imm: 4}, {Kind: Less, A: 0, B: 1}, {Kind: Const, Imm: 0}, {Kind: Const, Imm: 8}, {Kind: SelectValue, A: 2, B: 3, Imm: 4}, {Kind: StoreState, A: 5, Imm: 0}}
	a, err := n.Compile(first, 4)
	if err != nil {
		return err
	}
	b, err := n.Compile(second, 4)
	if err != nil {
		return err
	}
	beforeProof := n.Bytes
	proofBytes := len(n.proofRequest) - 24
	if err = n.PrepareTargetLink(0, a, 4, 1); err != nil {
		return err
	}
	if err = n.PrepareTargetLink(4, b, 4, 1); err != nil {
		return err
	}
	if proofBytes <= 0 || n.Bytes != beforeProof+2*proofBytes {
		return fmt.Errorf("WASM reused proof escaped per-admission byte accounting")
	}
	if err = n.PrepareTargetLink(0, a, 4, 1); err != nil || n.Bytes != beforeProof+2*proofBytes {
		return fmt.Errorf("WASM repeated immutable proof was reinstalled")
	}
	m := &MemoryContext{}
	m.PublishLink(0, a, 1, [17]uint8{})
	m.PublishLink(4, b, 1, [17]uint8{})
	state := []uint64{0, 0, 0, 100}
	if err = n.CallLinked(state, m, 1); err != nil {
		return err
	}
	if state[0] != 4 || state[1] != 1 || m.Total != 1 || m.Remaining != 0 {
		return fmt.Errorf("WASM linked budget mismatch")
	}
	if err = n.RunLinkedSession(state, m, 32); err != nil {
		return err
	}
	if state[0] != 8 || state[1] != 4 || m.Total != 7 || m.Remaining != 25 || m.MemoryTotal != 0 || m.LinkedSourcePC() != 4 {
		return fmt.Errorf("WASM linked transitions mismatch")
	}
	state[0] = 0
	before := state[1]
	link := m.LookupLink(0)
	link.Instructions = 2
	if err = n.CallLinked(state, m, 8); err == nil || m.Total != 0 || state[1] != before {
		return fmt.Errorf("WASM forged instruction proof accepted")
	}
	link.Instructions = 1
	link.Entry = 4096
	if err = n.CallLinked(state, m, 8); err == nil || m.Total != 0 || state[1] != before {
		return fmt.Errorf("WASM absent table entry accepted")
	}
	link.Entry = uint64(a)
	var region Builder
	region.MemoryInstructionProgress = true
	region.Checkpoint(4, 0)
	count := region.Binary(Add, region.Load(1), region.Constant(1))
	region.Store(1, count)
	region.Store(2, region.Elapsed(region.Load(3), 2, 1))
	region.Store(0, region.Constant(0))
	region.Checkpoint(4, 2)
	region.LoopContinue(region.Binary(Less, count, region.Constant(3)))
	entry, err := n.CompileLoop(region.FinishMemory(4), 4, 2)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 4, 2); err != nil {
		return err
	}
	m.ClearLinks()
	m.PublishLink(0, entry, 2, [17]uint8{})
	state[0], state[1] = 0, 0
	if err = n.RunLinkedSession(state, m, 5); err != nil {
		return err
	}
	if state[1] != 2 || state[2] != 106 || m.Total != 4 || m.Remaining != 1 {
		return fmt.Errorf("WASM region budget/elapsed mismatch")
	}
	state[1] = 0
	if err = n.RunLinkedSession(state, m, 64); err != nil {
		return err
	}
	if state[1] != 3 || state[2] != 110 || m.Total != 6 || m.Status != RegionExit || m.Remaining != 58 {
		return fmt.Errorf("WASM region exit mismatch")
	}
	// Closing an owner revokes public descriptors. Reusing empty bookkeeping
	// must never reuse owner identity or confer authority on old entry integers.
	m.ClaimLinks(n)
	if err = n.Close(); err != nil {
		return err
	}
	if m.LookupLink(0) != nil {
		return fmt.Errorf("WASM close retained public links")
	}
	fresh, err := NewNativeReusing(1<<20, n)
	if err != nil {
		return err
	}
	defer fresh.Close()
	if fresh == n {
		return fmt.Errorf("WASM recycled owner identity reused")
	}
	if err = fresh.PrepareLinks(4, 0); err != nil {
		return err
	}
	m.PublishLink(0, entry, 2, [17]uint8{})
	state[0] = 0
	before = state[1]
	if err = fresh.CallLinked(state, m, 64); err == nil || m.Total != 0 || state[1] != before {
		return fmt.Errorf("WASM recycled owner accepted stale target")
	}
	return nil
}
