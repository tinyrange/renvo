//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"runtime"
	"unsafe"
)

// All offsets consumed by the native emitter are verified, never presumed from
// a platform name. Typed host pointers keep pages, epochs and the clock rooted.
func memoryLayoutOK() bool {
	var m MemoryContext
	var p NativePage
	return unsafe.Sizeof(p) == 32 && unsafe.Offsetof(p.Data) == 8 && unsafe.Offsetof(p.Permissions) == 16 && unsafe.Offsetof(p.Epoch) == 24 &&
		unsafe.Offsetof(m.Retired) == 0 && unsafe.Offsetof(m.Status) == 8 && unsafe.Offsetof(m.Address) == 16 && unsafe.Offsetof(m.Clock) == 24 && unsafe.Offsetof(m.Pages) == 32
}
func (n *Native) CompileMemory(ops []Op, words int) (int, error) {
	if !memoryLayoutOK() {
		return 0, fmt.Errorf("unsupported native memory ABI layout")
	}
	if err := ValidateMemory(ops, words); err != nil {
		return 0, err
	}
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
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
