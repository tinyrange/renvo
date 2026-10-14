//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/rfenativebridge"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
	"unsafe"
)

func TestOwnedDeadlineSessionProgressFaultAndExpiry(t *testing.T) {
	if !runimage.ForeignSessionsAvailable() {
		t.Skip("foreign timers unavailable")
	}
	a, err := runimage.NewCodeArena(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	dispatch, ok := backendcompiled.RenvoEmitLinkedDispatcher(1, 2, false)
	if !ok {
		t.Fatal("dispatcher")
	}
	d, err := a.InstallLinkedDispatcher(dispatch, 2)
	if err != nil {
		t.Fatal(err)
	}
	// One guest instruction loads a word and adds it, then jumps to itself.
	records := []int{0, 0, 0, 4096, 13, 0, 0, 8, 1, 0, 0, 0, 3, 2, 1, 0, 2, 3, 0, 0, 0, 0, 0, 0, 2, 5, 0, 1, 15, 0, 0, 1}
	code, ok := backendcompiled.RenvoEmitMemoryBlock(records, 2, false)
	if !ok {
		t.Fatal("memory block")
	}
	entry, err := a.InstallContextBlock(code, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AdmitLinkedBlock(entry, 2, 1); err != nil {
		t.Fatal(err)
	}
	call, err := a.PrepareLinkedCall(d, 2)
	if err != nil {
		t.Fatal(err)
	}
	m := new(runimage.LinkedContextABI)
	m.Blocks[0].Entry, m.Blocks[0].Instructions = uint64(entry), 1
	m.Blocks[0].Prefix = [17]uint8{0, 1}
	backing := make([]byte, 8192)
	backing[4096] = 17
	epoch := uint64(1)
	m.Clock = &epoch
	m.Pages[1].Number, m.Pages[1].Data, m.Pages[1].Permissions, m.Pages[1].Epoch = 1, (*[4096]byte)(backing[4096:]), 1, &epoch
	state := []uint64{0, 0}
	run := func(budget, deadline uint64) bool {
		t.Helper()
		before := state[0]
		expired, err := call.CallOwnedBackingSessionUntil(state, m, budget, backing, deadline)
		if err != nil {
			t.Fatal(err)
		}
		if m.Total > budget || m.Remaining != budget-m.Total || m.MemoryTotal != m.Total || state[0] != before+17*m.Total || state[1] != 0 {
			t.Fatal("incorrect exact progress", state, m.Total, m.Remaining, m.MemoryTotal)
		}
		if m.CodeView != [4]uint64{} || m.PreparedTargets != 0 || m.AdmissionEpoch != 0 || m.DescriptorBase != 0 || m.TimerFunction != 0 || m.TimerDeadline != 0 || m.TimerExpired != 0 {
			t.Fatal("borrow survived return")
		}
		return expired
	}
	// A caller-supplied callback address is never executable authority.
	m.TimerFunction, m.TimerDeadline, m.TimerExpired = 1, 1, 1
	// Non-multiple-of-64 budget and GC exercise the resident graph across polls.
	if run(1001, rfenativebridge.MonotonicNS()+1000000000) || m.Total != 1001 {
		t.Fatal("resource bound", m.Total)
	}
	runtime.GC()
	if run(65536, rfenativebridge.MonotonicNS()+1000000000) || m.Total != 65536 {
		t.Fatal("large session", m.Total)
	}
	// A genuine deadline, never retirement-derived time. Preemption may expire
	// before entry; either way the exact prefix and state must agree.
	if !run(65536, rfenativebridge.MonotonicNS()+50000) {
		t.Fatal("deadline was not polled")
	}
	if !run(65536, rfenativebridge.MonotonicNS()-1) || m.Total != 0 || m.Status != 0 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("expired entry executed")
	}
	// An independent epoch/state alias must still select compatibility;
	// a far-future timer never upgrades it to a staged foreign call.
	m.Pages[1].Epoch = &state[0]
	if run(65536, rfenativebridge.MonotonicNS()+1000000000) || m.Total != 64 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("epoch alias escaped short boundary")
	}
	m.Pages[1].Epoch = &epoch
	// Likewise for state backed by RAM, and for a replaced backing graph.
	replacement := make([]byte, 8192)
	replacement[4096] = 23
	m.Pages[1].Data = (*[4096]byte)(replacement[4096:])
	state = []uint64{0, 0}
	hit, err := call.CallOwnedBackingSessionUntil(state, m, 129, replacement, rfenativebridge.MonotonicNS()+1000000000)
	if err != nil || hit || m.Total != 129 || state[0] != 129*23 {
		t.Fatal("replacement graph", err, hit, state, m.Total)
	}
	alias := unsafe.Slice((*uint64)(unsafe.Pointer(&replacement[4096])), 2)
	alias[0], alias[1] = 23, 0
	hit, err = call.CallOwnedBackingSessionUntil(alias, m, 65536, replacement, rfenativebridge.MonotonicNS()+1000000000)
	if err != nil || hit || m.Total != 64 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("RAM alias escaped short boundary", err, hit, m.Total)
	}
	m.Pages[1].Data = (*[4096]byte)(backing[4096:])
	state = []uint64{0, 0}
	runtime.GC()
	// Numeric revocation remains authoritative with the same pointer graph.
	m.Pages[1].Permissions = 0
	if run(65536, rfenativebridge.MonotonicNS()+1000000000) || m.Total != 0 || m.Status != 1 || m.Address != 4096 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("permission fault bypassed", m.Status)
	}
}
