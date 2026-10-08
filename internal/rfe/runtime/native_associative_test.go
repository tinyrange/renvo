//go:build !renvo

package runtime

import (
	"encoding/binary"
	"testing"
	"unsafe"
)

// Four colliding guest PCs execute an independently calculated mixed-family
// cycle. Public descriptors and proofs deliberately occupy different ways.
func TestNativeAssociativeMixedCycle(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(3, 2); err != nil {
		t.Fatal(err)
	}
	pcs := []uint64{0, 0x1004, 0x2008, 0x300c}
	entries := make([]int, 4)
	for i, pc := range pcs {
		if NativeLinkSlot(pc) != NativeLinkSlot(pcs[0]) {
			t.Fatal("fixture does not collide")
		}
		var b Builder
		if i&1 != 0 {
			b.MemoryStore(b.Constant(4096+uint64(i)*8), b.Load(0), 8)
		}
		b.Store(0, b.Binary(Add, b.Load(0), b.Constant(uint64(i+1))))
		b.Store(2, b.Constant(pcs[(i+1)%4]))
		if i == 2 {
			entries[i], err = n.Compile(b.Finish(3), 3)
		} else {
			b.Checkpoint(3, 1)
			if i == 1 {
				entries[i], err = n.CompileMemory(b.FinishMemory(3), 3)
			} else {
				b.LoopContinue(b.Constant(0))
				entries[i], err = n.CompileLoop(b.FinishMemory(3), 3, 1)
			}
		}
		if err != nil {
			t.Fatal(i, err)
		}
	}
	for i := 3; i >= 0; i-- {
		if err = n.PrepareTargetLink(pcs[i], entries[i], 3, 1); err != nil {
			t.Fatal(err)
		}
	}
	for _, session := range []bool{false, true} {
		if session && !NativeSessionsAvailable() {
			continue
		}
		for _, budget := range []uint64{1, 2, 3, 4, 17, 63, 64, 65535, 65536} {
			if !session && budget > 64 {
				continue
			}
			page := new([4096]byte)
			clock, epoch := uint64(1), uint64(1)
			m := &MemoryContext{Clock: &clock}
			m.Fill(1, page, 3, &epoch)
			m.ClaimLinks(n)
			for i, pc := range pcs {
				prefix := [17]uint8{}
				if i&1 != 0 {
					prefix[1] = 1
				}
				m.PublishLink(pc, entries[i], 1, prefix)
			}
			state := []uint64{7, 99, pcs[0]}
			if session {
				err = n.RunLinkedSession(state, m, budget)
			} else {
				err = n.CallLinked(state, m, budget)
			}
			if err != nil {
				t.Fatal(session, budget, err)
			}
			want, mem := uint64(7), uint64(0)
			wantPage := [4096]byte{}
			for j := uint64(0); j < budget; j++ {
				i := int(j % 4)
				if i&1 != 0 {
					binary.LittleEndian.PutUint64(wantPage[i*8:], want)
					mem++
				}
				want += uint64(i + 1)
			}
			if state[0] != want || state[1] != 99 || state[2] != pcs[budget%4] || *page != wantPage || m.Total != budget || m.Remaining != 0 || m.MemoryTotal != mem || clock != 1+mem || epoch != clock || m.AdmissionEpoch != 0 {
				t.Fatal("mixed collision cycle", session, budget, state, m.Total, m.MemoryTotal, clock, epoch)
			}
		}
	}
}

func TestNativeSessionAdmissionIsFresh(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
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
	if err = n.PrepareTargetLink(0, entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	m := new(MemoryContext)
	m.ClaimLinks(n)
	for _, mode := range []string{"count", "entry", "pc", "clear", "other-context", "zero-hole"} {
		m.ClearLinks()
		m.PublishLink(0, entry, 1, [17]uint8{})
		state := []uint64{11, 0}
		if err = n.RunLinkedSession(state, m, 17); err != nil || state[0] != 28 {
			t.Fatal("warm", err, state)
		}
		target := m
		switch mode {
		case "count":
			m.LookupLink(0).Instructions = 2
		case "entry":
			m.LookupLink(0).Entry++
		case "pc":
			m.LookupLink(0).PC = 0x1004
		case "clear":
			m.ClearLinks()
		case "other-context":
			target = new(MemoryContext)
			target.ClaimLinks(n)
		case "zero-hole":
			record := *m.LookupLink(0)
			m.ClearLinks()
			m.Blocks[768] = record
		}
		// Public data cannot grant authority, even if it resembles a past token.
		target.AdmissionEpoch = 1
		state = []uint64{11, 0}
		if err = n.RunLinkedSession(state, target, 17); err != nil {
			t.Fatal(mode, err)
		}
		want := uint64(0)
		if mode == "zero-hole" {
			want = 17
		}
		if state[0] != 11+want || target.Total != want || target.Remaining != 17-want || target.AdmissionEpoch != 0 {
			t.Fatal(mode, state, target.Total, target.Remaining)
		}
	}
}

// First warm a cached target, then let guest RAM mutate both its descriptor
// and the public epoch field. The alias path must never acquire cache authority.
func TestNativeSessionAliasedDescriptorCannotEnableCache(t *testing.T) {
	if !NativeSessionsAvailable() {
		t.Skip("foreign session unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(2, 1); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.MemoryStore(b.Constant(4096+16), b.Constant(2), 8)
	b.Checkpoint(2, 1)
	b.MemoryStore(b.Constant(8192+4088), b.Constant(1), 8)
	b.Store(0, b.Binary(Add, b.Load(0), b.Constant(1)))
	b.Store(1, b.Constant(4))
	b.Checkpoint(2, 2)
	entry, err := n.CompileMemory(b.FinishMemory(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(0, entry, 2, 2); err != nil {
		t.Fatal(err)
	}
	var next Builder
	next.Store(0, next.Binary(Add, next.Load(0), next.Constant(100)))
	next.Store(1, next.Constant(0))
	second, err := n.Compile(next.Finish(2), 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = n.PrepareTargetLink(4, second, 2, 1); err != nil {
		t.Fatal(err)
	}
	clock, e1, e2 := uint64(1), uint64(1), uint64(1)
	m := &MemoryContext{Clock: &clock}
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 1, 2})
	m.PublishLink(4, second, 1, [17]uint8{})
	m.Fill(1, new([4096]byte), 3, &e1)
	m.Fill(2, new([4096]byte), 3, &e2)
	if err = n.RunLinkedSession([]uint64{0, 0}, m, 6); err != nil {
		t.Fatal(err)
	}
	// Mutate Entry (at byte 8), not Instructions, to an invalid unaligned offset.
	// The stored 2 is guaranteed not to be an admitted entry.
	m.Fill(1, (*[4096]byte)(unsafe.Add(unsafe.Pointer(&m.Blocks[NativeLinkSlot(4)]), -8)), 3, &e1)
	m.Fill(2, (*[4096]byte)(unsafe.Add(unsafe.Pointer(&m.AdmissionEpoch), -4088)), 3, &e2)
	state := []uint64{0, 0}
	if err = n.RunLinkedSession(state, m, 65536); err != nil {
		t.Fatal(err)
	}
	if state[0] != 1 || state[1] != 4 || m.Total != 2 || m.MemoryTotal != 2 || m.Remaining != 65534 || m.AdmissionEpoch != 0 {
		t.Fatal("aliased authority", state, m.Total, m.MemoryTotal, m.Remaining)
	}
}

func TestNativeAssociativeEvictionFullTags(t *testing.T) {
	m := new(MemoryContext)
	pcs := []uint64{0, 0x1004, 0x2008, 0x300c, 0x4010}
	for i, pc := range pcs {
		m.PublishLink(pc, i*16, 1, [17]uint8{})
	}
	present := 0
	for i, pc := range pcs {
		if link := m.LookupLink(pc); link != nil {
			present++
			if link.PC != pc || link.Entry != uint64(i*16) {
				t.Fatal("wrong full-PC entry")
			}
		}
	}
	if present != 4 {
		t.Fatal("not bounded four-way replacement", present)
	}
	m.ClearLinks()
	for _, pc := range pcs {
		if m.LookupLink(pc) != nil {
			t.Fatal("clear retained a way")
		}
	}
}
