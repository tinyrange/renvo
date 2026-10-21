package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	"testing"
)

func TestLiveWatchReferencesRetireWithoutLosingDecodedOrTableWatches(t *testing.T) {
	data := make([]byte, 4096)
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a & 4095, true, nil })
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	a := m.ProveCode([]engine.CodeRange{{4, 12}}, m.CodeMutation())
	b := m.ProveCode([]engine.CodeRange{{4104, 4112}}, m.CodeMutation())
	m.MarkPageTableEntry(10, 1)
	if a == nil || b == nil {
		t.Fatal("proof")
	}
	a.Close()
	if m.ownedWatches.Intersects(4, 4, 1) || !m.ownedWatches.Intersects(8, 4, 1) || !m.ownedWatches.Intersects(4, 12, 4) {
		t.Fatal("overlapping proof reference release")
	}
	b.Close()
	if m.ownedWatches.Intersects(0, 4096, 1) || !m.ownedWatches.Intersects(10, 1, 2) || !m.ownedWatches.Intersects(4, 12, 4) || len(m.proofs.refs) != 0 {
		t.Fatal("retired watch leaked or erased independent ownership")
	}
}

func TestReclamationCannotRestoreProofWithoutReinstallingNativeWatch(t *testing.T) {
	data := make([]byte, 4096)
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	p := m.ProveCode([]engine.CodeRange{{4, 12}}, m.CodeMutation())
	m.ReclaimCode()
	if p.Valid() || len(m.proofs.live) != 0 || m.ownedWatches.Intersects(4, 8, 1) {
		t.Fatal("reclamation retained proof authority")
	}
	if !p.Revalidate() || !m.ownedWatches.Intersects(4, 8, 1) {
		t.Fatal("restored proof bypasses live native byte watch")
	}
	m.BeforeWrite(4, 1)
	if p.Valid() {
		t.Fatal("restored reclaimed proof ignored code write")
	}
}

func TestClosingRAMRevokesRetainedProofsAndPublications(t *testing.T) {
	data := make([]byte, 4096)
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return a, true, nil })
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	p := m.ProveCode([]engine.CodeRange{{4, 12}}, m.CodeMutation())
	c := m.NativeContext()
	c.PublishLink(4, 0, 1, [17]uint8{})
	p.Bind(c.LookupLink(4), 0)
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if p.Valid() || p.Revalidate() || c.LookupLink(4) != nil || len(m.proofs.live) != 0 {
		t.Fatal("closed RAM retained code admission")
	}
}
