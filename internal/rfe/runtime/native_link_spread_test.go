//go:build !renvo

package runtime

import "testing"

// Nearby word-aligned code entries must not thrash one four-way set solely
// because address bits between the word offset and page number were ignored.
// Eight independently compiled bodies exercise both Go publication and native
// selection; the expected state comes from scalar cycle arithmetic.
func TestNativeNearbyCodeEntriesStayLinked(t *testing.T) {
	n, e := NewNative(1 << 20)
	if e != nil {
		t.Fatal(e)
	}
	defer n.Close()
	if e = n.PrepareLinks(2, 1); e != nil {
		t.Fatal(e)
	}
	pcs := make([]uint64, 8)
	for i := range pcs {
		pcs[i] = 0x800000 + uint64(i)*1024 + uint64(i/4)*4
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for i, pc := range pcs {
		var b Builder
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(uint64(i+1))))
		b.Store(1, b.Constant(pcs[(i+1)%len(pcs)]))
		entry, err := n.Compile(b.Finish(2), 2)
		if err != nil {
			t.Fatal(err)
		}
		if err = n.PrepareTargetLink(pc, entry, 2, 1); err != nil {
			t.Fatal(err)
		}
		m.PublishLink(pc, entry, 1, [17]uint8{})
	}
	for _, pc := range pcs {
		if m.LookupLink(pc) == nil {
			t.Fatal("nearby code entry evicted", pc)
		}
	}
	for _, foreign := range []bool{false, true} {
		if foreign && !NativeSessionsAvailable() {
			continue
		}
		for _, budget := range []uint64{1, 7, 8, 9, 17, 63, 64, 65, 4095, 65536} {
			if !foreign && budget > 64 {
				continue
			}
			state := []uint64{19, pcs[0]}
			want := uint64(19)
			for i := uint64(0); i < budget; i++ {
				want += i%8 + 1
			}
			if foreign {
				e = n.RunLinkedSession(state, m, budget)
			} else {
				e = n.CallLinked(state, m, budget)
			}
			if e != nil || state[0] != want || state[1] != pcs[budget%8] || m.Total != budget || m.Remaining != 0 || m.Status != 0 {
				t.Fatal("nearby native cycle", foreign, budget, e, state, want, m.Total, m.Remaining, m.Status)
			}
		}
	}
	// Warming the new hash does not retain descriptor authority in a later
	// session. Revoking the fourth body stops after the first three exactly.
	link := m.LookupLink(pcs[3])
	link.Entry++
	state := []uint64{0, pcs[0]}
	if e = n.CallLinked(state, m, 17); e != nil {
		t.Fatal(e)
	}
	if state[0] != 6 || state[1] != pcs[3] || m.Total != 3 || m.Remaining != 14 || m.Status != 0 {
		t.Fatal("revoked nearby entry", state, m.Total, m.Remaining, m.Status)
	}
}
