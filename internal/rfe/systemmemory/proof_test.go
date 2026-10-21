package systemmemory

import (
	"renvo.dev/internal/rfe/engine"
	"testing"
)

func TestLocalProofsRevokeExactBytesAliasesAndOnlyAffectedRoot(t *testing.T) {
	root := 1
	data := make([]byte, 4*4096)
	m := New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) { return uint64(root-1)*4096 + (a & 4095), true, nil })
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	keys := []*engine.CodeIdentity{new(engine.CodeIdentity), new(engine.CodeIdentity)}
	m.SelectNativeContext(keys[0])
	one := m.NativeContext()
	a := m.ProveCode([]engine.CodeRange{{0x1000, 0x1004}}, m.CodeMutation())
	b := m.ProveCode([]engine.CodeRange{{0x1080, 0x1084}}, m.CodeMutation())
	alias := m.ProveCode([]engine.CodeRange{{0x2000, 0x2004}}, m.CodeMutation())
	region := m.ProveCode([]engine.CodeRange{{0x1000, 0x1004}, {0x1180, 0x1184}}, m.CodeMutation())
	if a == nil || b == nil || alias == nil || region == nil {
		t.Fatal("proof construction")
	}
	one.PublishLink(0x1000, 0, 1, [17]uint8{})
	a.Bind(one.LookupLink(0x1000), 0)
	one.PublishLink(0x1080, 16, 1, [17]uint8{})
	b.Bind(one.LookupLink(0x1080), 16)
	root = 2
	m.SelectNativeContext(keys[1])
	two := m.NativeContext()
	other := m.ProveCode([]engine.CodeRange{{0x1000, 0x1004}}, m.CodeMutation())
	two.PublishLink(0x1000, 32, 1, [17]uint8{})
	other.Bind(two.LookupLink(0x1000), 32)
	stamp := m.CodeContext()
	m.BeforeWrite(0x80, 1)
	if b.Valid() || !a.Valid() || !region.Valid() || !alias.Valid() || !other.Valid() || one.LookupLink(0x1080) != nil || two.LookupLink(0x1000) == nil || m.CodeContext() != stamp {
		t.Fatal("disjoint code mutation invalidated unrelated proof/publication")
	}
	m.BeforeWrite(0, 1)
	if a.Valid() || alias.Valid() || region.Valid() || !other.Valid() || one.LookupLink(0x1000) != nil || two.LookupLink(0x1000) == nil {
		t.Fatal("physical code alias or root coherence")
	}
	// Rebuild root one at the same VA. Revoking that VA must not erase the
	// unrelated root-two proof even though both publish the same linear PC.
	root = 1
	m.SelectNativeContext(keys[0])
	a = m.ProveCode([]engine.CodeRange{{0x1000, 0x1004}}, m.CodeMutation())
	m.InvalidateTaggedRange(keys[0], 0x1000, 4096)
	if a.Valid() || !other.Valid() || two.LookupLink(0x1000) == nil {
		t.Fatal("root-local mapping revocation")
	}
	m.InvalidateMappings()
	if other.Valid() || two.LookupLink(0x1000) != nil {
		t.Fatal("full flush retained execution authority")
	}
	other.Close()
	if len(m.proofs.live) != 0 || len(m.proofs.physical) != 0 || len(m.proofs.virtual) != 0 {
		t.Fatal("closed flushed proof retained ownership")
	}
}
func TestProofRejectsMutationDuringDiscoveryAndRevokesIndependentTails(t *testing.T) {
	data := make([]byte, 4*4096)
	mutate := false
	var m *RAM
	m = New(data, func() bool { return true }, func(a uint64, x bool) (uint64, bool, error) {
		if mutate {
			mutate = false
			m.BeforeWrite(0, 1)
		}
		if a >= 4096 {
			return a + 4096, true, nil
		}
		return a, true, nil
	})
	m.EnableOwnedMemory()
	m.EnableCodeProofs()
	first := m.ProveCode([]engine.CodeRange{{0, 4}}, m.CodeMutation())
	if first == nil {
		t.Fatal("initial proof")
	}
	mutation := m.CodeMutation()
	mutate = true
	if p := m.ProveCode([]engine.CodeRange{{0, 4}}, mutation); p != nil || first.Valid() {
		t.Fatal("walker mutation admitted stale decoded bytes")
	}
	p := m.ProveCode([]engine.CodeRange{{4094, 4098}}, m.CodeMutation())
	if p == nil {
		t.Fatal("straddling proof")
	}
	m.BeforeWrite(8192, 1)
	if p.Valid() {
		t.Fatal("nonadjacent physical tail was not watched")
	}
}
