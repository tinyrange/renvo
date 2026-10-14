//go:build !renvo

package runimage_test

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"testing"
	"unsafe"
)

// A fault-free cold target ends a native session, not the lifetime of its
// host-owned pointers. Exercise actual loads after GC and pointer replacement,
// then prove that numeric permission revocation still faults and releases pins.
func TestOwnedBackingPinsCheckInteriorPagesGCReplacementAndFault(t *testing.T) {
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
	backing := make([]byte, 8192)
	page, epoch, clock := (*[4096]byte)(backing[4096:]), new(uint64), new(uint64)
	page[0], *epoch, *clock = 17, 1, 1
	m.Pages[1].Number, m.Pages[1].Data, m.Pages[1].Permissions, m.Pages[1].Epoch = 1, page, 1, epoch
	m.Clock = clock
	run := func(value uint64) {
		state := []uint64{0, 0}
		if err := call.CallOwnedBackingSession(state, m, 65536, backing); err != nil {
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
	backing = make([]byte, 8192)
	replacement := (*[4096]byte)(backing[4096:])
	replacement[0] = 29
	m.Pages[1].Data = replacement
	runtime.GC()
	run(29)
	if runimage.OwnedSessionPinsForTest(a) == first {
		t.Fatal("replacement page reused stale pins")
	}
	stateBefore := []uint64{11, 0}
	if err := call.CallOwnedBackingSession(stateBefore, m, 65536, backing[:4096]); err == nil || stateBefore[0] != 11 || stateBefore[1] != 0 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("interior page escaped backing extent", err, stateBefore)
	}
	run(29)
	// Even an inactive descriptor must belong to the supplied backing.
	m.Pages[63].Data = new([4096]byte)
	if err := call.CallOwnedBackingSession(stateBefore, m, 65536, backing); err == nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("inactive out-of-backing pointer admitted", err)
	}
	m.Pages[63].Data = nil
	run(29)
	m.Pages[1].Permissions = 0
	state := []uint64{11, 0}
	if err := call.CallOwnedBackingSession(state, m, 65536, backing); err != nil || state[0] != 11 || m.Total != 0 || m.Status != 1 || m.Address != 4096 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("retained pins bypassed permission fault", err, state, m.Status)
	}
	m.Pages[1].Permissions = 1
	// State itself may be a view into the pinned RAM. Backing deduplication
	// must not suppress the conservative 64-instruction alias boundary.
	m.Blocks[1] = m.Blocks[0]
	m.Blocks[1].PC = 4
	if err := call.PrepareTarget(4, entry, 1); err != nil {
		t.Fatal(err)
	}
	// Independent epoch pointers can alias state even with disjoint data.
	// Reads do not test epoch values (epochs stamp writes), but the pointer
	// alias must still select the original 64-instruction compatibility path
	// instead of running a large foreign slice with staged architectural state.
	epochAlias := []uint64{1, 0}
	m.Pages[1].Epoch = &epochAlias[0]
	replacement[0] = 29
	if err := call.CallOwnedBackingSession(epochAlias, m, 65536, backing); err != nil || m.Total != 64 || m.Remaining != 65536-64 || m.Status != 0 || epochAlias[0] != 1+29*64 || epochAlias[1] != 4 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("backing optimization missed an independent epoch alias", err, epochAlias, m.Total, m.Status)
	}
	m.Pages[1].Epoch = epoch
	alias := unsafe.Slice((*uint64)(unsafe.Pointer(replacement)), 2)
	alias[0], alias[1] = 17, 0
	if err := call.CallOwnedBackingSession(alias, m, 65536, backing); err != nil || m.Total != 64 || m.Remaining != 65536-64 || alias[0] != 0 || alias[1] != 4 || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("backing aliases bypassed conservative boundary", err, alias, m.Total, m.Remaining)
	}
	if err = a.Close(); err != nil || runimage.OwnedSessionPinsForTest(a) != nil {
		t.Fatal("close retained pins", err)
	}
	runtime.GC()
}
