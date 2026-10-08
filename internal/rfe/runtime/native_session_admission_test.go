//go:build !renvo

package runtime

import "testing"

func TestNativeSessionWideAdmissionAndEmbeddedOwners(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("supported foreign boundary unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	var b Builder
	for instruction := 1; instruction <= 256; instruction++ {
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
		b.Store(2, b.Constant(0))
		b.Checkpoint(3, instruction)
	}
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 256)
	if err != nil {
		t.Fatal(err)
	}
	if n.PrepareTargetLink(0, entry, 3, 16) == nil || n.PrepareTargetLink(0, entry, 3, 255) == nil || n.PrepareTargetLink(0, entry, 3, 257) == nil {
		t.Fatal("wide admission lost exact static count")
	}
	if err = n.PrepareTargetLink(0, entry, 3, 256); err != nil {
		t.Fatal(err)
	}
	// The architectural array may be a field of an object that also contains
	// interfaces, maps and pointer graphs. Do not pass its enclosing allocation
	// directly to C, or pin an unrelated enclosing graph to evade cgo checks.
	owner := struct {
		State   [3]uint64
		Private map[string]interface{}
	}{Private: map[string]interface{}{"native": n, "slice": []byte{1, 2, 3}}}
	for _, mode := range []string{"hit", "wrong-count", "huge-count", "wrong-entry", "wrong-pc", "clear", "proof-collision"} {
		owner.State = [3]uint64{17, 99, 0}
		m := new(MemoryContext)
		m.ClaimLinks(n)
		m.PublishLink(0, entry, 256, [17]uint8{})
		switch mode {
		case "wrong-count":
			m.Blocks[0].Instructions = 16
		case "huge-count":
			m.Blocks[0].Instructions = 257
		case "wrong-entry":
			m.Blocks[0].Entry++
		case "wrong-pc":
			m.Blocks[0].PC = 4096
		case "clear":
			m.ClearLinks()
		case "proof-collision":
			if err = n.PrepareTargetLink(4096, entry, 3, 256); err != nil {
				t.Fatal(err)
			}
		}
		if err = n.RunLinkedSession(owner.State[:], m, 65536); err != nil {
			t.Fatal(mode, err)
		}
		want := uint64(0)
		if mode == "hit" || mode == "proof-collision" {
			want = 65536
		}
		if owner.State != [3]uint64{17 + want, 99, 0} || m.Total != want || m.Remaining != 65536-want || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 {
			t.Fatal(mode, owner.State, m.Total, m.Remaining)
		}
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, budget := range []uint64{0, 65537} {
		m.CodeView, m.PreparedTargets, m.DescriptorBase = [4]uint64{1, 2, 3, 4}, 1, 1
		if n.RunLinkedSession(owner.State[:], m, budget) == nil || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 {
			t.Fatal("invalid session retained borrowed views")
		}
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	if n.RunLinkedSession(owner.State[:], m, 256) == nil {
		t.Fatal("closed session accepted")
	}
}
