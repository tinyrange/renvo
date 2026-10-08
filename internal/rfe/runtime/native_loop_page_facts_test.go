//go:build !renvo

package runtime

import "testing"

func TestNativeLoopReadWritePageFactsDoNotAuthorizeEachOther(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	address := b.Load(0)
	b.MemoryStore(address, b.Constant(41), 8)
	b.Checkpoint(3, 1)
	b.Store(1, b.MemoryLoad(address, 8))
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	page := new([4096]byte)
	clock, epoch := uint64(1), uint64(1)
	m := &MemoryContext{Clock: &clock}
	m.Fill(1, page, 2, &epoch) // writable, not readable
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{})
	state := []uint64{4096, 99, 0}
	if err = n.CallLinked(state, m, 64); err != nil || m.Total != 1 || m.MemoryTotal != 1 || m.Status != 1 || state[1] != 99 || page[0] != 41 || clock != 2 {
		t.Fatal("write fact authorized read", err, state, m.Total, m.Status, clock)
	}
	// A forged full page tag still cannot admit an address above 48 bits.
	upper := uint64(1) << 48
	m.Pages[(upper>>12)&63] = NativePage{Number: upper >> 12, Data: page, Permissions: 3, Epoch: &epoch}
	state = []uint64{upper, 99, 0}
	if err = n.CallLinked(state, m, 64); err != nil || m.Total != 0 || m.Status != 1 || m.Address != upper || clock != 2 {
		t.Fatal("noncanonical fact", err, m.Total, m.Status, m.Address, clock)
	}
}

func TestNativeLoopPageHitStillChecksEveryAccessWidth(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{2, 4, 8} {
		var b Builder
		address := b.Load(0)
		b.Store(1, b.MemoryLoad(address, width))
		b.Store(0, b.Binary(Add, address, b.Constant(1)))
		b.Checkpoint(3, 1)
		b.LoopContinue(b.Constant(1))
		entry, err := n.CompileLoop(b.FinishMemory(3), 3, 1)
		if err != nil {
			t.Fatal(err)
		}
		page := new([4096]byte)
		for i := range page {
			page[i] = 0x11
		}
		m := new(MemoryContext)
		epoch := uint64(1)
		m.Fill(1, page, 1, &epoch)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 1, [17]uint8{})
		state := []uint64{8192 - uint64(width), 0, 0}
		if err = n.CallLinked(state, m, 64); err != nil || m.Total != 1 || m.MemoryTotal != 1 || m.Status != 1 || m.Address != 8193-uint64(width) || state[0] != m.Address {
			t.Fatal("width guard skipped on page hit", width, err, state, m.Total, m.Status, m.Address)
		}
	}
}
