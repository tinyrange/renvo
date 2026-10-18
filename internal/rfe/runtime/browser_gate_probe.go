//go:build renvo && browser

package runtime

import "fmt"

// These calls execute through the compiled AOT table syscall, not the legacy
// host fd_write path or a Go evaluator. Private host metadata must remain
// authoritative even if a copied Native has forged local entry dimensions.
func browserGateProbe() error {
	n, err := NewNative(4096)
	if err != nil {
		return err
	}
	if n.gateOwner <= 0 || n.Bytes != len(emitWasmGate()) {
		n.Close()
		return fmt.Errorf("WASM gate owner/byte accounting mismatch")
	}
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: Const, Imm: 1}, {Kind: Add, A: 0, B: 1}, {Kind: StoreState, A: 2, Imm: 0}}
	entry, err := n.Compile(ops, 1)
	if err != nil {
		n.Close()
		return err
	}
	state := []uint64{10, 20}
	if err = n.Call(entry, state[:1]); err != nil || state[0] != 11 {
		n.Close()
		return fmt.Errorf("WASM direct gate execution mismatch")
	}
	slot := browserEntrySlot(entry)
	n.entries[slot].words = 2
	if n.Call(entry, state) == nil || state[0] != 11 || state[1] != 20 {
		n.Close()
		return fmt.Errorf("WASM gate trusted forged state dimensions")
	}
	n.entries[slot].words = 1
	n.entries[slot].memory = true
	if n.CallMemory(entry, state[:1], new(MemoryContext)) == nil || state[0] != 11 {
		n.Close()
		return fmt.Errorf("WASM gate trusted forged memory dimensions")
	}
	n.entries[slot].memory = false
	stale := *n
	stale.entries = append([]browserEntry(nil), n.entries...)
	if err = n.Close(); err != nil {
		return err
	}
	// The stale copy retains a nonnil device and independent nonzero metadata.
	// Its root-table slot was revoked and must return an error, never trap or run.
	if stale.Call(entry, state[:1]) == nil || state[0] != 11 {
		return fmt.Errorf("WASM closed gate owner revived through a copy")
	}
	replacement, err := NewNativeReusing(4096, n)
	if err != nil {
		return err
	}
	defer replacement.Close()
	if replacement.gateOwner == stale.gateOwner {
		return fmt.Errorf("WASM gate owner token repeated")
	}
	if _, err = replacement.Compile(ops, 1); err != nil {
		return err
	}
	if stale.Call(entry, state[:1]) == nil || state[0] != 11 {
		return fmt.Errorf("WASM replacement revived stale gate")
	}
	return nil
}
