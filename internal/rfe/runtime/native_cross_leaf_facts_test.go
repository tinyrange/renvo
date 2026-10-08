//go:build !renvo

package runtime

import "testing"

func TestNativeCrossLeafFactsPermissionsWidthsAndCallBoundary(t *testing.T) {
	for _, trial := range []struct {
		name                    string
		permissions             uint64
		writeFirst, writeSecond bool
		width                   int
		address                 uint64
		boundary                bool
	}{
		{"read does not admit write", 1, false, true, 8, 4096, false},
		{"write does not admit read", 2, true, false, 8, 4096, false},
		{"execute forbids native write", 7, false, true, 8, 4096, false},
		{"hit still checks width", 1, false, false, 8, 8191, false},
		{"hit still checks high bits", 1, false, false, 1, 4096 + (uint64(1) << 48), false},
		{"new call discards read proof", 1, false, false, 1, 4096, true},
	} {
		t.Run(trial.name, func(t *testing.T) {
			n, err := NewNative(1 << 20)
			if err != nil {
				t.Fatal(err)
			}
			defer n.Close()
			if err = n.PrepareLinks(2, 1); err != nil {
				t.Fatal(err)
			}
			var entries [2]int
			for i := 0; i < 2; i++ {
				var b Builder
				addr := uint64(4096)
				if trial.name == "hit still checks width" {
					addr = 8191
				}
				if i == 1 {
					addr = trial.address
				}
				write := trial.writeFirst
				size := 1
				if i == 1 {
					write, size = trial.writeSecond, trial.width
				}
				if write {
					b.MemoryStore(b.Constant(addr), b.Constant(42), size)
				} else {
					b.Store(0, b.MemoryLoad(b.Constant(addr), size))
				}
				b.Store(1, b.Constant(uint64((i+1)*4)))
				b.Checkpoint(2, 1)
				entry, err := n.CompileMemory(b.FinishMemory(2), 2)
				if err != nil {
					t.Fatal(err)
				}
				if err = n.PrepareTargetLink(uint64(i*4), entry, 2, 1); err != nil {
					t.Fatal(err)
				}
				entries[i] = entry
			}
			var page [4096]byte
			page[0], page[4095] = 17, 19
			clock, epoch := uint64(1), uint64(1)
			m := &MemoryContext{NativeContext: NativeContext{Clock: &clock}}
			m.Fill(1, &page, trial.permissions, &epoch)
			m.ClaimLinks(n)
			m.PublishLink(0, entries[0], 1, [17]uint8{0, 1})
			m.PublishLink(4, entries[1], 1, [17]uint8{0, 1})
			state := []uint64{99, 0}
			wantTotal := uint64(1)
			if trial.boundary {
				if err := n.CallLinked(state, m, 1); err != nil || m.Total != 1 || m.Status != 0 || state[1] != 4 {
					t.Fatal("initial proof", err, state, m.Total, m.Status)
				}
				m.Fill(1, &page, 0, &epoch)
				wantTotal = 0
			}
			if err := n.CallLinked(state, m, 64); err != nil {
				t.Fatal(err)
			}
			if m.Total != wantTotal || m.MemoryTotal != wantTotal || m.Status != 1 || m.Address != trial.address || state[1] != 4 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 {
				t.Fatal("cross-leaf proof widened", state, m.Total, m.MemoryTotal, m.Status, m.Address)
			}
		})
	}
}
