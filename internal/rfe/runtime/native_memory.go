//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/rfeabi"
	"runtime"
	"unsafe"
)

// All offsets consumed by the native emitter are verified, never presumed from
// a platform name. Typed host pointers keep pages, epochs and the clock rooted.
func memoryLayoutOK() bool {
	var m MemoryContext
	var p NativePage
	var owner rfeabi.OwnedMemory
	var watches rfeabi.WriteWatches
	return unsafe.Offsetof(owner.Base) == 0 && unsafe.Offsetof(owner.Mask) == 8 && unsafe.Offsetof(owner.Backing) == 16 && unsafe.Offsetof(owner.Bytes) == 24 && unsafe.Offsetof(owner.Watches) == 32 && unsafe.Offsetof(owner.Clock) == 40 && unsafe.Offsetof(owner.DecodeEpochs) == 56 && unsafe.Offsetof(watches.Base) == 0 && unsafe.Offsetof(watches.Count) == 8 && unsafe.Sizeof(p) == backendcompiled.RenvoRFEPageSize && unsafe.Offsetof(p.Number) == 0 &&
		unsafe.Offsetof(p.Data) == backendcompiled.RenvoRFEPageData && unsafe.Offsetof(p.Permissions) == backendcompiled.RenvoRFEPagePermissions && unsafe.Offsetof(p.Epoch) == backendcompiled.RenvoRFEPageEpoch &&
		unsafe.Offsetof(m.Retired) == backendcompiled.RenvoRFERetired && unsafe.Offsetof(m.Status) == backendcompiled.RenvoRFEStatus && unsafe.Offsetof(m.Address) == backendcompiled.RenvoRFEAddress && unsafe.Offsetof(m.Clock) == backendcompiled.RenvoRFEClock && unsafe.Offsetof(m.Pages) == backendcompiled.RenvoRFEPages && unsafe.Offsetof(m.Memory) == backendcompiled.RenvoRFEOwnedMemory
}
func (n *Native) CompileMemory(ops []Op, words int) (int, error) {
	if !memoryLayoutOK() {
		return 0, fmt.Errorf("unsupported native memory ABI layout")
	}
	if err := ValidateMemory(ops, words); err != nil {
		return 0, err
	}
	records := nativeRecords(ops)
	code, body, ok := backendcompiled.RenvoEmitSharedBlock(records, words, true, runtime.GOARCH == "arm64")
	if !ok {
		return 0, fmt.Errorf("Renvo could not emit checked memory block")
	}
	entry, err := n.arena.InstallSharedBlock(code, words, true, body)
	if err == nil {
		n.Bytes += len(code)
	}
	return entry, err
}
func (n *Native) CallMemory(entry int, state []uint64, memory *MemoryContext) error {
	if memory == nil || !memoryLayoutOK() {
		return fmt.Errorf("invalid native memory context")
	}
	return n.arena.CallContext(entry, state, unsafe.Pointer(memory))
}
func (n *Native) CallBatchMemory(state []uint64, memory *MemoryContext, limit int, next func(int) (int, bool, error)) (int, error) {
	if memory == nil || !memoryLayoutOK() {
		return 0, fmt.Errorf("invalid native memory context")
	}
	return n.arena.CallBatchContext(state, unsafe.Pointer(memory), limit, next)
}
