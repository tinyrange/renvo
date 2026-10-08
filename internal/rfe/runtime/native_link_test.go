//go:build !renvo

package runtime

import "testing"

func TestNativeLinkedBudgetsAndAdmission(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	b := &Builder{}
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(0))
	entry, err := n.Compile(b.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, budget := range []uint64{1, 2, 63, 64} {
		m.PublishLink(0, entry, 1, [17]uint8{})
		state := []uint64{0, 0}
		if err = n.CallLinked(state, m, budget); err != nil {
			t.Fatal(err)
		}
		if state[0] != budget || m.Total != budget || m.Remaining != 0 || m.CodeView != [4]uint64{} {
			t.Fatalf("budget %d: state=%v progress=%d/%d view=%v", budget, state, m.Total, m.Remaining, m.CodeView)
		}
	}
	wrong, err := n.Compile(b.Finish(2), 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, offset := range []uint64{uint64(entry + 1), uint64(entry + 16), uint64(n.Bytes + 65536), uint64(wrong), uint64(n.linkEntry), ^uint64(0)} {
		m.Blocks[0] = NativeLink{PC: 0, Entry: offset, Instructions: 1}
		state := []uint64{0, 0}
		if err = n.CallLinked(state, m, 64); err != nil {
			t.Fatal(err)
		}
		if m.Total != 0 || state[0] != 0 {
			t.Fatalf("invalid offset %d executed", offset)
		}
	}
	for _, record := range []NativeLink{{PC: 4096, Entry: uint64(entry), Instructions: 1}, {PC: 0, Entry: uint64(entry), Instructions: 0}, {PC: 0, Entry: uint64(entry), Instructions: 17}, {PC: 0, Entry: uint64(entry), Instructions: 2}} {
		m.Blocks[0] = record
		state := []uint64{0, 0}
		if err = n.CallLinked(state, m, 1); err != nil {
			t.Fatal(err)
		}
		if m.Total != 0 || state[0] != 0 {
			t.Fatalf("invalid record executed: %+v", record)
		}
	}
	m.PublishLink(0, entry, 1, [17]uint8{})
	state := []uint64{0, 2}
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if m.Total != 0 {
		t.Fatal("unaligned PC executed")
	}
	m.ClaimLinks(nil)
	if m.Blocks[0].Instructions != 0 || n.CallLinked(state, m, 64) == nil {
		t.Fatal("owner mismatch accepted")
	}
	if n.PrepareLinks(3, 1) == nil {
		t.Fatal("shape changed")
	}
}
func TestNativeLinkedMixedFaultAndNoReplay(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	pure := &Builder{}
	pure.Store(0, pure.Constant(9))
	pure.Store(1, pure.Constant(4))
	first, err := n.Compile(pure.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	effect := &Builder{}
	effect.Checkpoint(2, 0)
	effect.MemoryStore(effect.Constant(4096), effect.Constant(42), 1)
	effect.Store(1, effect.Constant(8))
	effect.Checkpoint(2, 1)
	effect.MemoryLoad(effect.Constant(8192), 1)
	effect.Store(1, effect.Constant(12))
	effect.Checkpoint(2, 2)
	second, err := n.CompileMemory(effect.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	var page [4096]byte
	clock, epoch := uint64(1), uint64(1)
	m := &MemoryContext{Clock: &clock}
	m.Fill(1, &page, 3, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, first, 1, [17]uint8{})
	m.PublishLink(4, second, 2, [17]uint8{0, 1, 2})
	state := []uint64{0, 0}
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if state[0] != 9 || state[1] != 8 || page[0] != 42 || clock != 2 || epoch != 2 || m.Total != 2 || m.MemoryTotal != 1 || m.Status != 1 || m.Address != 8192 || m.Retired != 1 {
		t.Fatalf("bad partial: state%v page%d clock%d total%d mem%d status%d retired%d", state, page[0], clock, m.Total, m.MemoryTotal, m.Status, m.Retired)
	}
	// Resumption starts at the missing successor, never replays the first store.
	if err = n.CallLinked(state, m, 64); err != nil {
		t.Fatal(err)
	}
	if m.Total != 0 || clock != 2 {
		t.Fatal("prefix replayed")
	}
}

func TestNativeLinkedInvalidProgressAndClosure(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, completed := range []int{1, 2, 17, 256} {
		var b Builder
		b.Checkpoint(2, completed)
		b.Guard(b.Constant(0))
		entry, err := n.CompileMemory(b.FinishMemory(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		m.PublishLink(0, entry, 1, [17]uint8{})
		state := []uint64{12, 0}
		if n.CallLinked(state, m, 1) == nil || m.Status != 3 || m.Total != 0 || state[0] != 12 || m.CodeView != [4]uint64{} {
			t.Fatal("invalid partial progress accepted", completed, m.Status, m.Total, state)
		}
	}
	for _, budget := range []uint64{0, 65, ^uint64(0)} {
		if n.CallLinked([]uint64{0, 0}, m, budget) == nil {
			t.Fatal("unbounded call accepted", budget)
		}
	}
	if err = n.Close(); err != nil {
		t.Fatal(err)
	}
	if n.CallLinked([]uint64{0, 0}, m, 64) == nil || m.CodeView != [4]uint64{} {
		t.Fatal("closed arena retained executable view")
	}
}
