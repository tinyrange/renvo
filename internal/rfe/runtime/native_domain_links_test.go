//go:build !renvo

package runtime

import (
	"runtime"
	"testing"
)

func TestGenericLinksAcceptByteAlignedPCWithoutPrivatePreparedProof(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	for _, pc := range []uint64{1, 2, 3, 0x1003} {
		var b Builder
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
		b.Store(1, b.Constant(pc))
		entry, err := n.Compile(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.AdmitLink(entry, 2, 1); err != nil {
			t.Fatal(err)
		}
		m := new(MemoryContext)
		m.ClaimLinks(n)
		m.PublishLink(pc, entry, 1, [17]uint8{})
		state := []uint64{0, pc}
		if err = n.CallLinked(state, m, 16); err != nil || state[0] != 16 || m.Total != 16 {
			t.Fatal("byte-aligned generic successor punted to host", pc, state, m.Total, err)
		}
	}
}
func TestSelectedCodeTablesRetainModesAndRevokeInactivePublication(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	entries := make([]int, 2)
	for i, delta := range []uint64{1, 7} {
		var b Builder
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(delta)))
		b.Store(1, b.Constant(3))
		entries[i], err = n.Compile(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.AdmitLink(entries[i], 2, 1); err != nil {
			t.Fatal(err)
		}
	}
	tables := []*NativeLinkTable{new(NativeLinkTable), new(NativeLinkTable)}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for i, table := range tables {
		m.SelectLinkTable(table)
		m.PublishLink(3, entries[i], 1, [17]uint8{})
	}
	for i := 0; i < 12; i++ {
		index := i & 1
		m.SelectLinkTable(tables[index])
		runtime.GC()
		state := []uint64{0, 3}
		delta := uint64(1)
		if index == 1 {
			delta = 7
		}
		if err = n.RunLinkedSession(state, m, 16); err != nil || state[0] != 16*delta || m.Total != 16 || m.DescriptorBase != 0 {
			t.Fatal("code domain lost or borrowed table leaked", i, state, m.Total, m.DescriptorBase, err)
		}
	}
	// Synchronous invalidation can revoke an inactive table without changing the
	// current mode. The next selection must not execute the old native body.
	tables[0].Blocks[NativeLinkSlot(3)].Instructions = 0
	m.SelectLinkTable(tables[0])
	state := []uint64{0, 3}
	if err = n.RunLinkedSession(state, m, 16); err != nil || state[0] != 0 || m.Total != 0 {
		t.Fatal("inactive publication resurrected", state, m.Total, err)
	}
	m.SelectLinkTable(tables[1])
	state[0] = 0
	if err = n.RunLinkedSession(state, m, 16); err != nil || state[0] != 112 {
		t.Fatal("unrelated mode publication discarded", state, err)
	}
}
