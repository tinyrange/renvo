//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
	"unsafe"
)

// Exercise pin lifetime, not just a cached pointer comparison: actual native
// loads must follow replacement pages, fresh numeric permissions must still
// fault, compatibility aliases must retain their 64-instruction boundary,
// and GC/installation/closure must never retain an invalid native view.
func TestOwnedSessionPinsRefreshGCAndRelease(t *testing.T) {
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
	// state[0] += load64(4096); PC = 0; one retired memory instruction.
	records := []int{0, 0, 0, 4096, 13, 0, 0, 8, 1, 0, 0, 0, 3, 2, 1, 0, 2, 3, 0, 0, 0, 0, 0, 0, 2, 5, 0, 1, 15, 0, 0, 1}
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
		if err = call.CallOwnedSession(state, m, 65536); err != nil {
			t.Fatal(err)
		}
		if state[0] != value*65536 || state[1] != 0 || m.Total != 65536 || m.MemoryTotal != 65536 || m.Remaining != 0 || m.Status != 0 || m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.DescriptorBase != 0 || m.AdmissionEpoch != 0 {
			t.Fatal("native result/borrow", state, m.Total, m.MemoryTotal, m.Remaining, m.Status)
		}
		if runimage.OwnedSessionPinsForTest(a) == nil {
			t.Fatal("eligible budget stop lost pins")
		}
	}
	run(17)
	first := runimage.OwnedSessionPinsForTest(a)
	runtime.GC()
	run(17)
	if runimage.OwnedSessionPinsForTest(a) != first {
		t.Fatal("unchanged owned graph not reused")
	}
	// Change each identity independently, including pointers in inactive slots.
	for _, change := range []string{"page", "epoch", "clock", "inactive-slot"} {
		old := runimage.OwnedSessionPinsForTest(a)
		switch change {
		case "page":
			page = new([4096]byte)
			page[0] = 29
			m.Pages[1].Data = page
		case "epoch":
			epoch = new(uint64)
			*epoch = 2
			m.Pages[1].Epoch = epoch
		case "clock":
			clock = new(uint64)
			*clock = 3
			m.Clock = clock
		case "inactive-slot":
			m.Pages[63].Data, m.Pages[63].Epoch = new([4096]byte), new(uint64)
		}
		runtime.GC()
		run(29)
		if runimage.OwnedSessionPinsForTest(a) == old {
			t.Fatal("pin identity was not refreshed", change)
		}
	}
	// Pointer equality cannot authorize changed permissions or page tags.
	m.Pages[1].Permissions = 0
	state := []uint64{11, 0}
	if err = call.CallOwnedSession(state, m, 65536); err != nil || state[0] != 11 || m.Total != 0 || m.Status != 1 || m.Address != 4096 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("pin reuse authorized unreadable page", err, state, m.Total, m.Status)
	}
	m.Pages[1].Permissions = 1
	run(29)
	m.Pages[1].Number = 2
	state = []uint64{11, 0}
	if err = call.CallOwnedSession(state, m, 65536); err != nil || state[0] != 11 || m.Total != 0 || m.Status != 1 || m.Address != 4096 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("pin reuse authorized changed page tag", err, state, m.Total, m.Status)
	}
	m.Pages[1].Number = 1
	run(29)
	// A new context with the same pages must replace, not extend, the lease.
	old := runimage.OwnedSessionPinsForTest(a)
	replacement := new(runimage.LinkedContextABI)
	*replacement = *m
	m = replacement
	runtime.GC()
	run(29)
	if runimage.OwnedSessionPinsForTest(a) == old {
		t.Fatal("replacement context reused the old lease")
	}
	// A fresh session must re-read descriptors even when all pin identities
	// match. A zero-progress link miss must release even with a small budget.
	m.Blocks[0].PC = 4
	state = []uint64{11, 0}
	if err = call.CallOwnedSession(state, m, 128); err != nil || state[0] != 11 || m.Total != 0 || m.Status != 0 || m.Remaining != 128 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("pin reuse authorized stale descriptor", err, state, m.Total, m.Status, m.Remaining)
	}
	m.Blocks[0].PC = 0
	run(29)
	if _, err = a.InstallContextBlock(code, 2); err != nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("installation retained pins", err)
	}
	run(29)
	if call.CallOwnedSession(state, m, 0) == nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("invalid call retained pins")
	}
	run(29)
	if err = call.CallSession([]uint64{0, 0}, m, 64); err != nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("ordinary session retained owned cache", err)
	}
	run(29)
	// Alias the actual architectural state with guest data. This remains the
	// existing conservative path, even after an eligible owned call warmed it.
	alias := unsafe.Slice((*uint64)(unsafe.Pointer(page)), 2)
	alias[0], alias[1] = 17, 0
	if err = call.CallOwnedSession(alias, m, 65536); err != nil || m.Total != 64 || m.Remaining != 65536-64 || alias[0] != 0 || alias[1] != 0 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("alias bypassed compatibility boundary", err, alias, m.Total, m.Remaining)
	}
	page[0] = 29
	run(29)
	if err = a.Close(); err != nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("close retained pins", err)
	}
	runtime.GC()
	if call.CallOwnedSession([]uint64{0, 0}, m, 64) == nil {
		t.Fatal("closed owner executed")
	}
}
