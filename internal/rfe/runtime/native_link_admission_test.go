//go:build !renvo

package runtime

import "testing"

func TestNativeLinkedImmutableAdmissionChecksEverySelection(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	entry, err := n.Compile(b.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := n.Compile(b.Finish(2), 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][3]int{{entry + 1, 2, 1}, {entry + 16, 2, 1}, {wrong, 2, 1}, {n.linkEntry, 2, 1}, {entry, 3, 1}, {entry, 2, 0}, {entry, 2, 17}} {
		if n.AdmitLink(args[0], args[1], args[2]) == nil {
			t.Fatal("invalid admission", args)
		}
	}
	if err = n.AdmitLink(entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	if err = n.AdmitLink(entry, 2, 1); err != nil {
		t.Fatal("same immutable admission", err)
	}
	if n.AdmitLink(entry, 2, 2) == nil {
		t.Fatal("admission body changed")
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, count := range []uint64{0, 2, 16, 17, ^uint64(0), 1} {
		m.Blocks[0] = NativeLink{PC: 0, Entry: uint64(entry), Instructions: count}
		state := []uint64{0, 0}
		if err = n.CallLinked(state, m, 64); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			if state[0] != 64 || m.Total != 64 {
				t.Fatal("admitted leaf did not execute", state, m.Total)
			}
		} else if state[0] != 0 || m.Total != 0 {
			t.Fatal("forged count executed", count, state, m.Total)
		}
		if m.CodeView != [4]uint64{} {
			t.Fatal("view retained")
		}
	}
	// Appending code can reallocate the typed metadata, but admitted keys must
	// remain rooted and the borrowed pointer must be refreshed on every call.
	for i := 0; i < 40; i++ {
		if _, err = n.Compile(b.Finish(2), 2); err != nil {
			t.Fatal(err)
		}
	}
	state := []uint64{0, 0}
	if err = n.CallLinked(state, m, 64); err != nil || state[0] != 64 {
		t.Fatal("admission lost after append", err, state)
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	if n.AdmitLink(entry, 2, 1) == nil || n.CallLinked(state, m, 64) == nil {
		t.Fatal("closed admission accepted")
	}
}

func TestNativeLinkedAdmittedLoopAndMemoryFault(t *testing.T) {
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
	b.Store(2, b.Constant(4))
	b.Checkpoint(3, 1)
	b.MemoryLoad(b.Constant(8192), 8)
	b.Store(2, b.Constant(0))
	b.Checkpoint(3, 2)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(3), 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if n.AdmitLink(entry, 3, 1) == nil {
		t.Fatal("loop length changed")
	}
	if err = n.AdmitLink(entry, 3, 2); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{3, 0, 0}
	if err = n.CallLinked(state, m, 64); err != nil || m.Total != 0 || state[0] != 3 {
		t.Fatal("cached loop wrong count", err, state, m.Total)
	}
	m.PublishLink(0, entry, 2, [17]uint8{})
	if err = n.CallLinked(state, m, 64); err != nil || m.Total != 1 || m.Status != 1 || state[0] != 4 || state[2] != 4 || m.Address != 8192 {
		t.Fatal("cached loop fault", err, state, m.Total, m.Status, m.Address)
	}
}

// Compact admission keys must preserve both maximum dimensions without a bit
// collision between the ninth state-size bit and the four body-length bits.
func TestNativeLinkedAdmissionMaximumDimensions(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(256, 255); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(255, b.Constant(0))
	b.Checkpoint(256, 16)
	b.LoopContinue(b.Constant(1))
	entry, err := n.CompileLoop(b.FinishMemory(256), 256, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.AdmitLink(entry, 256, 16); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, count := range []int{1, 15, 16} {
		m.PublishLink(0, entry, count, [17]uint8{})
		state := make([]uint64, 256)
		if err = n.CallLinked(state, m, 64); err != nil {
			t.Fatal(err)
		}
		if count == 16 {
			if state[0] != 4 || m.Total != 64 || m.LoopIterations != 4 {
				t.Fatal("maximum dimensions", state[0], m.Total, m.LoopIterations)
			}
		} else if state[0] != 0 || m.Total != 0 {
			t.Fatal("compact key count collision", count, state[0], m.Total)
		}
	}
}
