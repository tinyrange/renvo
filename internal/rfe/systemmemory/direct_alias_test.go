//go:build !renvo && linux && amd64 && cgo

package systemmemory

import (
	"encoding/binary"
	"testing"
)

func TestDirectRAMPhysicalWatchRevokesAllVAAliases(t *testing.T) {
	for _, table := range []bool{false, true} {
		h := newDirectHarness(t, 4096)
		h.m.Resolve = func(a uint64, x bool) (uint64, bool, error) { return a & 4095, true, nil }
		binary.LittleEndian.PutUint64(h.m.Data, 17)
		h.m.NativeRefill(4096 + 19)
		h.m.NativeRefill(8192 + 31)
		if table {
			h.m.MarkPageTable(0)
		} else {
			if _, err := h.m.CodeVersion(4096); err != nil {
				t.Fatal(err)
			}
		}
		// Watch established through VA1 must make stores through VA2 exit.
		h.m.NativeRefill(8192 + 7)
		state, ctx := h.run(t, 8192, 256)
		if ctx.Status != 1 || ctx.Total != 1 || ctx.Address != 8192 || state[2] != 17 || binary.LittleEndian.Uint64(h.m.Data) != 17 {
			t.Fatal("physical alias bypassed watch", table, state, ctx.Status, ctx.Total)
		}
	}
}
