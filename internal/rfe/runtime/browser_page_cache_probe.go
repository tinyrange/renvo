//go:build renvo && browser

package runtime

import "fmt"

// Execute a working set exceeding the old 64-entry geometry, then verify that
// a copied context and mapping revocation cannot retain or share authority.
func browserPageWorkingSetProbe() error {
	n, err := NewNative(1 << 20)
	if err != nil {
		return err
	}
	defer n.Close()
	ops := []Op{{Kind: LoadState, Imm: 0}, {Kind: MemoryLoad, A: 0, Imm: 1}, {Kind: StoreState, A: 1, Imm: 1}}
	entry, err := n.CompileMemory(ops, 2)
	if err != nil {
		return err
	}
	var data [4096]byte
	data[0] = 71
	clock, epoch := uint64(5), uint64(5)
	m := &MemoryContext{}
	m.Clock = &clock
	for number := uint64(1); number <= 128; number++ {
		m.Fill(number, &data, 1, &epoch)
	}
	for number := uint64(1); number <= 128; number++ {
		state := []uint64{number << 12, 99}
		if err = n.CallMemory(entry, state, m); err != nil {
			return err
		}
		if m.Status != 0 || state[1] != 71 {
			return fmt.Errorf("compiled browser page working set mismatch")
		}
	}
	copied := *m
	copied.ClearPages()
	state := []uint64{4096, 99}
	if err = n.CallMemory(entry, state, &copied); err != nil {
		return err
	}
	if copied.Status != 1 || state[1] != 99 {
		return fmt.Errorf("copied page authority survived clear")
	}
	if err = n.CallMemory(entry, state, m); err != nil {
		return err
	}
	if m.Status != 0 || state[1] != 71 {
		return fmt.Errorf("copied page cache aliases original")
	}
	m.ClearPages()
	state[1] = 99
	if err = n.CallMemory(entry, state, m); err != nil {
		return err
	}
	if m.Status != 1 || state[1] != 99 {
		return fmt.Errorf("page authority survived mapping revocation")
	}
	return nil
}
