//go:build !renvo

package runtime

import (
	hostruntime "runtime"
	"testing"
)

func TestNativeDynamicSuccessorsExceedFixedCapacityAndHonorTombstones(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	n, m, ram := ownedFixture(t)
	if !n.AdmitCodeVersionedContext(m) {
		t.Fatal("owned context admission")
	}
	m.ClaimLinks(n)
	m.SelectLinkTable(NewNativeLinkTable())
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(4)))
	entry, err := n.Compile(b.Finish(1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareLinks(1, 0); err != nil {
		t.Fatal(err)
	}
	const count = 3000
	// Growth intentionally does not copy proof-bound publications. Repopulate
	// until the final grown directory contains every simultaneously live PC.
	for {
		before := m.Links.Mask
		for i := 0; i < count; i++ {
			m.PublishLink(1+uint64(i)*4, entry, 1, [17]uint8{})
		}
		if before == m.Links.Mask {
			break
		}
	}
	for i := 0; i < count; i++ {
		if m.LookupLink(1+uint64(i)*4) == nil {
			t.Fatal("live publication evicted", i)
		}
	}
	hostruntime.GC()
	state := []uint64{1}
	if err = n.RunLinkedBackingSession(state, m, count, ram); err != nil || state[0] != 1+count*4 || m.Total != count || m.Remaining != 0 || m.Status != 0 {
		t.Fatal("native successor directory fell back to Go", state, m.Total, m.Status, err)
	}
	// Revocation is a tombstone, not the end of a collision cluster.
	victim := m.LookupLink(101)
	victim.Instructions = 0
	state[0] = 1
	if err = n.RunLinkedBackingSession(state, m, count, ram); err != nil || state[0] != 101 || m.Total != 25 {
		t.Fatal("revoked body remained admitted", state, m.Total, err)
	}
	state[0] = 105
	if err = n.RunLinkedBackingSession(state, m, 1, ram); err != nil || state[0] != 109 || m.Total != 1 {
		t.Fatal("tombstone hid unrelated successor", state, m.Total, err)
	}
	m.LinkEpoch = ^uint64(0)
	m.ClearLinks()
	if m.LookupLink(105) != nil {
		t.Fatal("epoch wrap resurrected old publication")
	}
}

func TestNativeDynamicSuccessorProbeStopsAtFullTombstoneCycle(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	n, m, ram := ownedFixture(t)
	if !n.AdmitCodeVersionedContext(m) {
		t.Fatal("owned context")
	}
	m.ClaimLinks(n)
	m.SelectLinkTable(NewNativeLinkTable())
	if err := n.PrepareLinks(1, 0); err != nil {
		t.Fatal(err)
	}
	for i := range m.Links.Entries() {
		m.Links.Entries()[i] = NativeLink{PC: 2, Generation: m.LinkEpoch}
	}
	state := []uint64{1}
	if err := n.RunLinkedBackingSession(state, m, 1, ram); err != nil || m.Total != 0 || state[0] != 1 {
		t.Fatal("full successor table did not return a bounded miss", err, m.Total, state)
	}
}
