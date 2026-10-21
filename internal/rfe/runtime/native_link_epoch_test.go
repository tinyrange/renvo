//go:build !renvo

package runtime

import "testing"

func TestLinkGenerationRevokesPreparedAndGenericEntries(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	entry, err := n.Compile(b.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{0, 0}
	if err = n.RunLinkedSession(state, m, 100); err != nil || state[0] != 100 {
		t.Fatal("warm", state, err)
	}
	m.ClearLinks()
	// Raw descriptors and immutable prepared target survive in storage. Their
	// old generation must not execute through either native admission path.
	if m.Blocks[0].Instructions == 0 {
		t.Fatal("invalidation still clears the table")
	}
	for _, session := range []bool{false, true} {
		state[0] = 0
		if session {
			err = n.RunLinkedSession(state, m, 100)
		} else {
			err = n.CallLinked(state, m, 64)
		}
		if err != nil || state[0] != 0 || m.Total != 0 {
			t.Fatal("stale generation executed", session, state, err)
		}
	}
	m.LinkEpoch = ^uint64(0)
	m.ClearLinks()
	if m.LookupLink(0) != nil || m.Blocks[0].Instructions != 0 {
		t.Fatal("epoch wrap resurrected old storage")
	}
	m.PublishLink(0, entry, 1, [17]uint8{})
	if err = n.CallLinked(state, m, 64); err != nil || state[0] != 64 {
		t.Fatal("fresh generation failed", state, err)
	}
}
