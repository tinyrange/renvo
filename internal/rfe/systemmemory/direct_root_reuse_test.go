//go:build !renvo && linux && amd64 && cgo

package systemmemory

import (
	"encoding/binary"
	"testing"
)

// Boot roots must not permanently consume the bounded reservation cache or
// hide later user mappings. Recycling must never resurrect revoked pages.
func TestDirectWindowsFollowMoreThanEightRootsWithoutStaleAuthority(t *testing.T) {
	h := newDirectHarness(t, 4096)
	entry := int(h.m.NativeContext().LookupLink(0).Entry)
	for root := uint64(0); root < 12; root++ {
		h.root = root
		h.base = root*directWindowSize + 4096
		h.m.InvalidateMappings()
		h.m.NativeRefill(h.base)
		ctx := h.m.NativeContext()
		if ctx.DirectGuest != root*directWindowSize || ctx.DirectSize != directWindowSize {
			t.Fatal("cold root starved direct window", root, ctx.DirectGuest, ctx.DirectSize)
		}
		d := h.m.direct.(*mappedRAM)
		if len(d.windows) > 8 {
			t.Fatal("reservation cap exceeded", len(d.windows))
		}
		ctx.ClaimLinks(h.n)
		if !h.n.AdmitCodeVersionedContext(ctx) {
			t.Fatal("context")
		}
		ctx.PublishLink(0, entry, 2, [17]uint8{})
		binary.LittleEndian.PutUint64(h.m.Data, root+31)
		state, ctx := h.run(t, h.base, 2)
		if ctx.Status != 0 || ctx.Total != 2 || state[2] != root+31 || binary.LittleEndian.Uint64(h.m.Data) != root+32 {
			t.Fatal("wrong current root", root, state, ctx.Status)
		}
		// Reactivate another page after revoking all pages. The old page remains
		// protected even when its virtual reservation was recycled for this root.
		h.m.InvalidateMappings()
		h.m.NativeRefill(h.base + 4096)
		state, ctx = h.run(t, h.base, 2)
		if ctx.Status != 1 || ctx.Total != 0 || state[2] != 999 {
			t.Fatal("recycled stale page", root, state, ctx.Status, ctx.Total)
		}
	}
}
