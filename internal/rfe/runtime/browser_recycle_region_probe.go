//go:build renvo && browser

package runtime

import "fmt"

// Optimized code must not carry its old memory-context admission into a new
// owner. Code and its exact proof can survive; that privileged contract cannot.
func browserRecycleRegionProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	if err = n.PrepareLinks(2, 0); err != nil {
		n.Close()
		return err
	}
	m := new(MemoryContext)
	clock := uint64(1)
	m.Clock = &clock
	var b Builder
	b.MemoryInstructionProgress = true
	count := b.Binary(Add, b.Load(1), b.Constant(1))
	b.Store(1, count)
	b.Store(0, b.Constant(0))
	b.Checkpoint(2, 1)
	b.LoopContinue(b.Binary(Less, count, b.Constant(3)))
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(2), 2, 1, m)
	if err != nil {
		n.Close()
		return err
	}
	if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
		n.Close()
		return err
	}
	if err = n.SuspendForRecycle(); err != nil {
		n.Close()
		return err
	}
	fresh, err := n.RecycleKeeping([]int{entry})
	if err != nil {
		n.Close()
		return err
	}
	defer fresh.Close()
	if err = fresh.PrepareLinks(2, 0); err != nil {
		return err
	}
	m.ClaimLinks(fresh)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{0, 0}
	if fresh.CanLinkMemory(m) || fresh.CallLinked(state, m, 8) == nil || state[1] != 0 {
		return fmt.Errorf("WASM transferred region inherited memory admission")
	}
	if !fresh.AdmitCodeVersionedContext(m) {
		return fmt.Errorf("WASM fresh region admission failed")
	}
	before := fresh.Bytes
	if err = fresh.PrepareTargetLink(0, entry, 2, 1); err != nil || fresh.Bytes != before {
		return fmt.Errorf("WASM exact retained region proof was reinstalled")
	}
	if err = fresh.CallLinked(state, m, 8); err != nil || state[1] != 3 || m.Total != 3 || m.Remaining != 5 {
		return fmt.Errorf("WASM retained region progress mismatch")
	}
	return nil
}
