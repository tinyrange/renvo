//go:build renvo && browser

package runtime

import "fmt"

func browserRecycleProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	if err = n.PrepareLinks(2, 0); err != nil {
		n.Close()
		return err
	}
	leaf := func(delta, next uint64) []Op {
		return []Op{{Kind: LoadState, Imm: 1}, {Kind: Const, Imm: delta}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 1}, {Kind: Const, Imm: next}, {Kind: StoreState, A: 4, Imm: 0}}
	}
	a, err := n.Compile(leaf(1, 4), 2)
	if err != nil {
		n.Close()
		return err
	}
	b, err := n.Compile(leaf(7, 8), 2)
	if err != nil {
		n.Close()
		return err
	}
	if err = n.PrepareTargetLink(0, a, 2, 1); err != nil {
		n.Close()
		return err
	}
	if err = n.PrepareTargetLink(4, b, 2, 1); err != nil {
		n.Close()
		return err
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, a, 1, [17]uint8{})
	m.PublishLink(4, b, 1, [17]uint8{})
	state := []uint64{0, 10}
	if err = n.CallLinked(state, m, 2); err != nil || state[1] != 18 {
		n.Close()
		return fmt.Errorf("WASM recycle probe initial execution")
	}
	stale := *n
	staleIndependent := *n
	staleIndependent.entries = append([]browserEntry(nil), n.entries...)
	dispatcher := n.dispatchEntry
	expected := n.entries[b>>4].bytes + n.entries[a>>4].bytes + 2*(len(n.proofRequest)-24) + n.gateBytes
	if err = n.SuspendForRecycle(); err != nil {
		n.Close()
		return err
	}
	if n.Call(a, state) == nil || m.LookupLink(0) != nil || m.linkOwner != nil {
		n.Close()
		return fmt.Errorf("WASM retirement did not revoke execution/links")
	}
	if _, err = n.RecycleKeeping([]int{dispatcher}); err == nil {
		n.Close()
		return fmt.Errorf("WASM dispatcher survived transfer")
	}
	if _, err = n.RecycleKeeping([]int{a, a}); err == nil {
		n.Close()
		return fmt.Errorf("WASM duplicate transfer entry admitted")
	}
	fresh, err := n.RecycleKeeping([]int{b, a})
	if err != nil {
		n.Close()
		return err
	}
	defer fresh.Close()
	if fresh.Bytes != expected || fresh.entryCount != 2 || fresh.gateOwner == stale.gateOwner || fresh.entries[b>>4].proof != (browserProof{4, 1}) || fresh.entries[a>>4].proof != (browserProof{0, 1}) {
		return fmt.Errorf("WASM transfer accounting/owner/proof mismatch")
	}
	state[0], state[1] = 0, 10
	if stale.Call(a, state) == nil || staleIndependent.Call(b, state) == nil || state[1] != 10 {
		return fmt.Errorf("WASM stale owner executed compacted code")
	}
	// Shared arrays AND the shared service file have moved to fresh. A stale
	// Close must not clear or close either resource under the new owner.
	if err = stale.Close(); err != nil {
		return err
	}
	if err = staleIndependent.Close(); err != nil {
		return err
	}
	if err = stale.Close(); err != nil {
		return err
	}
	if err = staleIndependent.Close(); err != nil {
		return err
	}
	if err = fresh.Call(b, state); err != nil || state[0] != 8 || state[1] != 17 {
		return fmt.Errorf("WASM transferred code lost after stale close")
	}
	if err = fresh.PrepareLinks(2, 0); err != nil {
		return err
	}
	m.ClaimLinks(fresh)
	m.PublishLink(0, b, 1, [17]uint8{})
	m.PublishLink(4, b, 1, [17]uint8{})
	state[0], state[1] = 0, 10
	if fresh.CallLinked(state, m, 2) == nil || m.Total != 0 || state[1] != 10 {
		return fmt.Errorf("WASM transfer admitted a proof at the wrong full PC")
	}
	m.PublishLink(0, a, 1, [17]uint8{})
	if err = fresh.CallLinked(state, m, 2); err != nil || m.Total != 2 || state[0] != 8 || state[1] != 18 {
		return fmt.Errorf("WASM retained code/proofs did not execute")
	}
	return browserRecycleRegionProbe()
}
