//go:build !renvo

package runtime

import "testing"

func TestPreparedTargetFullTagsCountsOffsetsAndOwners(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(2, b.Constant(0))
	entry, err := n.Compile(b.Finish(3), 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, pc := range []uint64{1, 2, 3} {
		if n.PrepareTargetLink(pc, entry, 3, 1) == nil {
			t.Fatal("unaligned proof", pc)
		}
	}
	if err = n.PrepareTargetLink(0, entry, 3, 1); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, count := range []uint64{0, 2, 17, ^uint64(0), 1} {
		m.Blocks[0] = NativeLink{PC: 0, Entry: uint64(entry), Instructions: count}
		state := []uint64{0, 0, 0}
		if err = n.CallLinked(state, m, 64); err != nil {
			t.Fatal(err)
		}
		want := uint64(0)
		if count == 1 {
			want = 64
		}
		if state[0] != want || m.Total != want || m.PreparedTargets != 0 || m.CodeView != [4]uint64{} {
			t.Fatal("forged count or borrow", count, state, m.Total)
		}
	}
	for _, offset := range []uint64{uint64(entry + 1), uint64(entry + 16), ^uint64(0)} {
		m.Blocks[0] = NativeLink{PC: 0, Entry: offset, Instructions: 1}
		state := []uint64{0, 0, 0}
		if err = n.CallLinked(state, m, 64); err != nil || m.Total != 0 || state[0] != 0 {
			t.Fatal("forged entry", offset, err)
		}
	}
	// A collision replaces the opaque proof; the generic checked admission still
	// handles a legitimate older publication without retaining a pointer cache.
	if err = n.PrepareTargetLink(4096, entry, 3, 1); err != nil {
		t.Fatal(err)
	}
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{0, 0, 0}
	if err = n.CallLinked(state, m, 3); err != nil || state[0] != 3 || m.Total != 3 {
		t.Fatal("collision fallback", err, state, m.Total)
	}
	m.Blocks[0].PC = 4096
	state[0] = 0
	if err = n.CallLinked(state, m, 64); err != nil || state[0] != 0 || m.Total != 0 {
		t.Fatal("full PC tag ignored", err, state)
	}
	other, err := NewNative(16384)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err = other.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	if err = other.CallLinked(state, m, 1); err == nil {
		t.Fatal("cross-owner context admitted")
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	if n.PrepareTargetLink(0, entry, 3, 1) == nil || n.CallLinked(state, m, 1) == nil || m.PreparedTargets != 0 {
		t.Fatal("closed owner accepted or borrowed proof leaked")
	}
}
