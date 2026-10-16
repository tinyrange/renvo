//go:build !renvo

package runtime

import (
	"renvo.dev/internal/rfenativebridge"
	"testing"
)

// Index collisions and malformed raw tags must never confer access; the
// preceding committed architectural effect must survive each slow exit once.
func TestNativeRawHashedSlotsFullTagsAndPrefix(t *testing.T) {
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	var b Builder
	b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
	b.Checkpoint(3, 1)
	b.Store(2, b.MemoryLoad(b.Load(0), 8))
	b.Checkpoint(3, 2)
	entry, err := n.CompileMemory(b.FinishMemory(3), 3)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(1)
	a, z := new([4096]byte), new([4096]byte)
	a[0] = 17
	z[0] = 23
	m := new(MemoryContext)
	m.HashPages = 1
	m.Clock = &epoch
	m.Fill(1, a, 1, &epoch)
	m.Fill(65, z, 1, &epoch)
	for _, trial := range []struct{ address, want, status uint64 }{{4096, 17, 0}, {65 * 4096, 23, 0}, {1 << 48, 999, 1}, {66 * 4096, 999, 1}} {
		state := []uint64{trial.address, 41, 999}
		if err = n.CallMemory(entry, state, m); err != nil {
			t.Fatal(err)
		}
		retired := uint64(2)
		if trial.status != 0 {
			retired = 1
		}
		if state[1] != 42 || state[2] != trial.want || m.Retired != retired || m.Status != trial.status || trial.status != 0 && m.Address != trial.address {
			t.Fatal("hashed precise prefix", trial, state, m.Retired, m.Status, m.Address)
		}
	}
	// Page 64 collides with 1 in this mode. Replacement cannot admit page 1
	// merely by matching the slot, and Forget of the old tag cannot erase 64.
	m.Fill(64, z, 1, &epoch)
	m.Forget(1)
	state := []uint64{4096, 41, 999}
	if err = n.CallMemory(entry, state, m); err != nil {
		t.Fatal(err)
	}
	if m.Status != 1 || m.Retired != 1 || state[2] != 999 {
		t.Fatal("colliding tag admitted")
	}
	state = []uint64{64 * 4096, 41, 999}
	if err = n.CallMemory(entry, state, m); err != nil {
		t.Fatal(err)
	}
	if m.Status != 0 || state[2] != 23 {
		t.Fatal("old-tag revocation erased new page")
	}
	// A forged high tag in a slot cannot bypass the full 48-bit address check.
	high := uint64(1) << 48
	m.Pages[m.PageIndex(high>>12)] = NativePage{Number: high >> 12, Data: z, Permissions: 1, Epoch: &epoch}
	state = []uint64{high, 41, 999}
	if err = n.CallMemory(entry, state, m); err != nil {
		t.Fatal(err)
	}
	if m.Status != 1 || m.Retired != 1 || state[2] != 999 {
		t.Fatal("noncanonical forged tag admitted")
	}
}

// The serialized foreign image must carry the opt-in raw-tag slot policy,
// including across genuine monotonic polls, without borrowing extra authority.
func TestNativeHashedSlotsForeignDeadlineSession(t *testing.T) {
	if !NativeSessionsAvailable() || rfenativebridge.MonotonicNS() == 0 {
		t.Skip("foreign timers unavailable")
	}
	n, err := NewNative(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err = n.PrepareLinks(4, 3); err != nil {
		t.Fatal(err)
	}
	var b Builder
	b.Store(1, b.Binary(Add, b.Load(1), b.Constant(1)))
	b.Checkpoint(4, 1)
	b.Store(2, b.MemoryLoad(b.Load(0), 8))
	b.Store(3, b.Constant(0))
	b.Checkpoint(4, 2)
	b.LoopContinue(b.Constant(1))
	m := new(MemoryContext)
	m.HashPages = 1
	entry, err := n.CompileCodeVersionedLoop(b.FinishMemory(4), 4, 2, m)
	if err != nil {
		t.Fatal(err)
	}
	backing := make([]byte, 66*4096)
	backing[65*4096] = 23
	epoch := uint64(1)
	m.Clock = &epoch
	m.Fill(65, (*[4096]byte)(backing[65*4096:]), 1, &epoch)
	m.ClaimLinks(n)
	m.PublishLink(0, entry, 2, [17]uint8{0, 0, 1})
	if err = n.PrepareTargetLink(0, entry, 4, 2); err != nil {
		t.Fatal(err)
	}
	state := []uint64{65 * 4096, 0, 999, 0}
	hit, err := n.RunLinkedBackingSessionUntil(state, m, 1001, backing, rfenativebridge.MonotonicNS()+1000000000)
	if err != nil || hit || m.Status != 0 || m.Total != 1000 || m.Remaining != 1 || m.MemoryTotal != 500 || state[1] != 500 || state[2] != 23 || m.HashPages != 1 {
		t.Fatal("foreign hash policy lost", err, hit, state, m.Total, m.Status, m.MemoryTotal)
	}
}
