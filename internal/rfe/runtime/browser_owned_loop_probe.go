//go:build renvo && browser

package runtime

import "fmt"

// Ownership is a host admission, never a property of mutable descriptors or
// flags. Check both emitted effects and every public path to the dispatcher.
func browserOwnedLoopProbe(n *Native, ops []Op, generic *MemoryContext) error {
	clock, epoch := uint64(5), uint64(3)
	page := new([4096]byte)
	owner := &MemoryContext{}
	owner.Clock = &clock
	owner.Fill(1, page, 3, &epoch)
	entry, err := n.CompileCodeVersionedLoop(ops, 5, 2, owner)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 5, 2); err != nil {
		return err
	}
	owner.ClaimLinks(n)
	owner.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	state := []uint64{0, 0, 4096, 0, 100}
	if err = n.RunLinkedSession(state, owner, 5); err != nil {
		return err
	}
	if state[1] != 2 || state[2] != 4112 || state[3] != 108 || owner.Total != 4 || owner.Remaining != 1 || owner.MemoryTotal != 2 || owner.LoopIterations != 2 || owner.Status != 0 || clock != 5 || epoch != 3 {
		return fmt.Errorf("WASM owned loop budget/generation mismatch")
	}
	state[0], state[1], state[2], state[3] = 0, 0, 8184, 0
	if err = n.RunLinkedSession(state, owner, 64); err != nil {
		return err
	}
	if state[1] != 2 || state[2] != 8192 || state[3] != 104 || owner.Total != 3 || owner.Remaining != 61 || owner.MemoryTotal != 1 || owner.Retired != 1 || owner.Status != 1 || owner.Address != 8192 || page[4088] != 1 || clock != 5 || epoch != 3 {
		return fmt.Errorf("WASM owned loop precise fault mismatch")
	}
	// This context previously made a successful generic linked call. Lifecycle
	// tracking, copied owner fields, and a valid immutable entry proof confer
	// no authority to use the owner's optimized code.
	forged := *owner
	for _, untrusted := range []*MemoryContext{generic, &forged} {
		untrusted.ClaimLinks(n)
		untrusted.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
		untrusted.wasmLinked, untrusted.Remaining = 1, 64
		state = []uint64{0, 0, 4096, 0, 100}
		if n.CanLinkMemory(untrusted) || n.CallLinked(state, untrusted, 64) == nil ||
			n.CallMemory(entry, state, untrusted) == nil || n.CallMemory(n.dispatchEntry, state, untrusted) == nil ||
			state[1] != 0 || clock != 5 || epoch != 3 {
			return fmt.Errorf("WASM generic context inherited owned authority")
		}
	}
	// Installing an owned loop must not change ordinary checked leaf stores.
	var b Builder
	b.Checkpoint(5, 0)
	b.MemoryStore(b.Load(2), b.Constant(77), 8)
	b.Checkpoint(5, 1)
	ordinary, err := n.CompileMemory(b.FinishMemory(5), 5)
	if err != nil {
		return err
	}
	state = []uint64{0, 0, 4096, 0, 100}
	owner.wasmLinked = 0
	if err = n.CallMemory(ordinary, state, owner); err != nil {
		return err
	}
	if owner.Status != 0 || owner.Retired != 1 || clock != 6 || epoch != 6 || page[0] != 77 {
		return fmt.Errorf("WASM generic leaf lost store generations")
	}
	return browserOwnedSpanProbe(n, owner, page, &clock, &epoch)
}

func browserOwnedSpanProbe(n *Native, owner *MemoryContext, page *[4096]byte, clock, epoch *uint64) error {
	var b Builder
	b.MemoryInstructionProgress = true
	b.Checkpoint(5, 0)
	address, count := b.Load(2), b.Load(1)
	for i := 0; i < 4; i++ {
		b.MemoryPairStore(b.Binary(Add, address, b.Constant(uint64(i*16))), count, b.Constant(uint64(i+77)))
		b.Checkpoint(5, i+1)
	}
	b.Store(2, b.Binary(Add, address, b.Constant(64)))
	count = b.Binary(Add, count, b.Constant(1))
	b.Store(1, count)
	b.Store(3, b.Elapsed(b.Load(4), 5, 2))
	condition := b.Binary(Less, count, b.Constant(4))
	b.Store(0, b.Choose(condition, b.Constant(0), b.Constant(4)))
	b.Checkpoint(5, 5)
	b.LoopContinue(condition)
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(5), 5, 5, owner)
	if err != nil {
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 5, 5); err != nil {
		return err
	}
	owner.PublishLink(0, entry, 5, [17]uint8{0, 1, 2, 3, 4, 4})
	state := []uint64{0, 0, 4096, 0, 100}
	if err = n.RunLinkedSession(state, owner, 11); err != nil {
		return err
	}
	// Elapsed(base, scale, completed): on iteration two the linked prefix is
	// five instructions, so the exact sampled clock is 100 + (5 + 2)*5.
	if state[1] != 2 || state[2] != 4224 || state[3] != 135 || owner.Total != 10 || owner.Remaining != 1 || owner.MemoryTotal != 8 || owner.Status != 0 || owner.LoopIterations != 2 || *clock != 6 || *epoch != 6 || page[64] != 1 {
		return fmt.Errorf("WASM owned span budget/state mismatch")
	}
	state = []uint64{0, 0, 8160, 0, 100}
	if err = n.RunLinkedSession(state, owner, 64); err != nil {
		return err
	}
	if state[1] != 0 || state[2] != 8160 || state[3] != 0 || owner.Total != 2 || owner.MemoryTotal != 2 || owner.Status != 1 || owner.Address != 8192 || *clock != 6 || *epoch != 6 || page[4088] != 78 {
		return fmt.Errorf("WASM owned span fault skipped prior pair stores")
	}
	return nil
}
