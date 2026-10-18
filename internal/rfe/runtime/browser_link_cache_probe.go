//go:build renvo && browser

package runtime

import "fmt"

func browserLinkCacheProbe(n *Native) error {
	m := &MemoryContext{}
	// More than the old 1024-entry capacity, with distinct full-PC tags.
	for i := 0; i < 1536; i++ {
		m.PublishLink(uint64(i)<<12, 16, 1, [17]uint8{})
	}
	for i := 0; i < 1536; i++ {
		if m.LookupLink(uint64(i)<<12) == nil {
			return fmt.Errorf("WASM descriptor working set was truncated")
		}
	}
	highPC := uint64(0xffff800080cca9dc)
	m.PublishLink(highPC, 16, 1, [17]uint8{})
	if m.LookupLink(highPC) == nil {
		return fmt.Errorf("WASM high-PC publication lost its tag")
	}
	copied := *m
	copied.ClearLinks()
	copied.PublishLink(4, 16, 1, [17]uint8{})
	if m.LookupLink(0) == nil || m.LookupLink(4) != nil || copied.LookupLink(0) != nil {
		return fmt.Errorf("WASM copied context aliased descriptor authority")
	}
	m.ClearLinks()
	pcs := [4]uint64{0x4000, 0x4000 | 1<<26, 0x4000 | 2<<26, 0x4000 | 3<<26}
	for i, pc := range pcs {
		ops := []Op{{Kind: Const, Imm: uint64(i + 1)}, {Kind: StoreState, A: 0, Imm: 1}, {Kind: Const, Imm: 8}, {Kind: StoreState, A: 2, Imm: 0}}
		entry, err := n.Compile(ops, 4)
		if err != nil {
			return err
		}
		if err = n.PrepareTargetLink(pc, entry, 4, 1); err != nil {
			return err
		}
		m.PublishLink(pc, entry, 1, [17]uint8{})
	}
	for i, pc := range pcs {
		state := []uint64{pc, 0, 0, 0}
		if err := n.CallLinked(state, m, 1); err != nil {
			return err
		}
		if state[0] != 8 || state[1] != uint64(i+1) || m.Total != 1 || m.Remaining != 0 {
			return fmt.Errorf("WASM indirect descriptor-way dispatch mismatch")
		}
	}
	if m.LookupLink(1<<48) != nil {
		return fmt.Errorf("WASM descriptor full PC tag lost")
	}
	m.ClearLinks()
	if m.LookupLink(pcs[0]) != nil {
		return fmt.Errorf("WASM cleared descriptor retained authority")
	}
	return nil
}
