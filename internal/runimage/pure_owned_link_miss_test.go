//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
)

// A fault-free cold target ends a native session, not the lifetime of its
// host-owned pointers. Exercise actual loads after GC and pointer replacement,
// then prove that numeric permission revocation still faults and releases pins.
func TestOwnedSessionPinsAcrossProgressingLinkMiss(t *testing.T) {
	if !runimage.ForeignSessionsAvailable() {
		t.Skip("foreign boundary unavailable")
	}
	a, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	arm := runtime.GOARCH == "arm64"
	dispatch, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, arm)
	if !ok {
		t.Fatal("dispatcher")
	}
	d, err := a.InstallLinkedDispatcher(dispatch, 2)
	if err != nil {
		t.Fatal(err)
	}
	// state[0] += load64(4096); PC=4 (unpublished); one retired instruction.
	records := []int{0, 0, 0, 4096, 13, 0, 0, 8, 1, 0, 0, 0, 3, 2, 1, 0, 2, 3, 0, 0, 0, 0, 0, 4, 2, 5, 0, 1, 15, 0, 0, 1}
	code, ok := backendcompiled.RenvoEmitMemoryBlock(records, 2, arm)
	if !ok {
		t.Fatal("memory block")
	}
	entry, err := a.InstallContextBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.AdmitLinkedBlock(entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	call, err := a.PrepareLinkedCall(d, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(runimage.LinkedContextABI)
	m.Blocks[0].PC, m.Blocks[0].Entry, m.Blocks[0].Instructions = 0, uint64(entry), 1
	m.Blocks[0].Prefix = [17]uint8{0, 1}
	page, epoch, clock := new([4096]byte), new(uint64), new(uint64)
	page[0], *epoch, *clock = 17, 1, 1
	m.Pages[1].Number, m.Pages[1].Data, m.Pages[1].Permissions, m.Pages[1].Epoch = 1, page, 1, epoch
	m.Clock = clock
	run := func(value uint64) {
		state := []uint64{0, 0}
		if err := call.CallOwnedSession(state, m, 65536); err != nil {
			t.Fatal(err)
		}
		if state[0] != value || state[1] != 4 || m.Total != 1 || m.Remaining != 65535 || m.Status != 0 {
			t.Fatal("incorrect early-exit progress", state, m.Total, m.Remaining, m.Status)
		}
		if m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 || m.AdmissionEpoch != 0 {
			t.Fatal("native borrow survived return")
		}
		if runimage.OwnedSessionPinsForTest(a) == nil {
			t.Fatal("progressing cold target discarded pins")
		}
	}
	run(17)
	first := runimage.OwnedSessionPinsForTest(a)
	runtime.GC()
	run(17)
	if runimage.OwnedSessionPinsForTest(a) != first {
		t.Fatal("unchanged graph repinned")
	}
	replacement := new([4096]byte)
	replacement[0] = 29
	m.Pages[1].Data = replacement
	runtime.GC()
	run(29)
	if runimage.OwnedSessionPinsForTest(a) == first {
		t.Fatal("replacement page reused stale pins")
	}
	m.Pages[1].Permissions = 0
	state := []uint64{11, 0}
	if err := call.CallOwnedSession(state, m, 65536); err != nil || state[0] != 11 || m.Total != 0 || m.Status != 1 || m.Address != 4096 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("retained pins bypassed permission fault", err, state, m.Status)
	}
	if err = a.Close(); err != nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("close retained pins", err)
	}
	runtime.GC()
}
