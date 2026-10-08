//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/runimage"
	"runtime"
	"unsafe"
)

func linkLayoutOK() bool {
	var m MemoryContext
	var b NativeLink
	var q runimage.LinkedContextABI
	if unsafe.Sizeof(q) != unsafe.Offsetof(m.linkOwner) ||
		unsafe.Offsetof(q.Retired) != unsafe.Offsetof(m.Retired) || unsafe.Offsetof(q.Status) != unsafe.Offsetof(m.Status) || unsafe.Offsetof(q.Address) != unsafe.Offsetof(m.Address) ||
		unsafe.Offsetof(q.Clock) != unsafe.Offsetof(m.Clock) || unsafe.Offsetof(q.Pages) != unsafe.Offsetof(m.Pages) || unsafe.Sizeof(q.Pages[0]) != unsafe.Sizeof(m.Pages[0]) ||
		unsafe.Offsetof(q.Pages[0].Number) != unsafe.Offsetof(m.Pages[0].Number) || unsafe.Offsetof(q.Pages[0].Data) != unsafe.Offsetof(m.Pages[0].Data) || unsafe.Offsetof(q.Pages[0].Permissions) != unsafe.Offsetof(m.Pages[0].Permissions) || unsafe.Offsetof(q.Pages[0].Epoch) != unsafe.Offsetof(m.Pages[0].Epoch) ||
		unsafe.Offsetof(q.Remaining) != unsafe.Offsetof(m.Remaining) || unsafe.Offsetof(q.Total) != unsafe.Offsetof(m.Total) || unsafe.Offsetof(q.MemoryTotal) != unsafe.Offsetof(m.MemoryTotal) ||
		unsafe.Offsetof(q.CodeView) != unsafe.Offsetof(m.CodeView) || unsafe.Offsetof(q.Blocks) != unsafe.Offsetof(m.Blocks) || unsafe.Sizeof(q.Blocks[0]) != unsafe.Sizeof(b) || unsafe.Offsetof(q.Blocks[0].Prefix) != unsafe.Offsetof(b.Prefix) || unsafe.Offsetof(q.Blocks[0].PC) != unsafe.Offsetof(b.PC) || unsafe.Offsetof(q.Blocks[0].Entry) != unsafe.Offsetof(b.Entry) || unsafe.Offsetof(q.Blocks[0].Instructions) != unsafe.Offsetof(b.Instructions) || unsafe.Offsetof(q.Blocks[0].Reserved) != unsafe.Offsetof(b.Reserved) ||
		unsafe.Offsetof(q.LoopExits) != unsafe.Offsetof(m.LoopExits) || unsafe.Offsetof(q.LoopIterations) != unsafe.Offsetof(m.LoopIterations) || unsafe.Offsetof(q.PreparedTargets) != unsafe.Offsetof(m.PreparedTargets) || unsafe.Offsetof(q.DescriptorBase) != unsafe.Offsetof(m.DescriptorBase) || unsafe.Offsetof(q.AdmissionEpoch) != unsafe.Offsetof(m.AdmissionEpoch) {
		return false
	}
	return memoryLayoutOK() && unsafe.Sizeof(b) == 64 && unsafe.Offsetof(b.Prefix) == 32 &&
		unsafe.Offsetof(m.Remaining) == 2080 && unsafe.Offsetof(m.Total) == 2088 && unsafe.Offsetof(m.MemoryTotal) == 2096 && unsafe.Offsetof(m.CodeView) == 2104 && unsafe.Offsetof(m.Blocks) == 2136 && unsafe.Offsetof(m.LoopExits) == 67672 && unsafe.Offsetof(m.LoopIterations) == 67680 && unsafe.Offsetof(m.PreparedTargets) == 67688 && unsafe.Offsetof(m.DescriptorBase) == 67696 && unsafe.Offsetof(m.AdmissionEpoch) == 67704
}

// PrepareLinks compiles the bounded dispatcher outside the arena call lock.
func (n *Native) PrepareLinks(words, pc int) error {
	if n.linkWords != 0 {
		if n.linkWords != words || n.linkPC != pc {
			return fmt.Errorf("native dispatcher state shape changed")
		}
		return nil
	}
	if !linkLayoutOK() {
		return fmt.Errorf("unsupported native link ABI")
	}
	code, ok := backendcompiled.RenvoEmitLinkedDispatcher(pc, words, runtime.GOARCH == "arm64")
	if !ok {
		return fmt.Errorf("could not emit native dispatcher")
	}
	entry, err := n.arena.InstallLinkedDispatcher(code, words)
	if err != nil {
		return err
	}
	call, err := n.arena.PrepareLinkedCall(entry, words)
	if err != nil {
		return err
	}
	n.linkCall = call
	n.linkEntry, n.linkWords, n.linkPC = entry, words, pc
	n.Bytes += len(code)
	_ = n.NameEntry(entry, "rfe_native_dispatcher")
	return nil
}
func (n *Native) CallLinked(state []uint64, m *MemoryContext, budget uint64) error {
	if m == nil || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || budget == 0 || budget > 64 {
		return fmt.Errorf("invalid native linked call")
	}
	m.DescriptorBase, m.AdmissionEpoch = 0, 0
	m.Remaining, m.Total, m.MemoryTotal, m.Status, m.Retired = budget, 0, 0, 0, 0
	m.LoopExits, m.LoopIterations = 0, 0
	err := n.linkCall.CallWithTargets(state, unsafe.Pointer(m), &m.CodeView, &m.PreparedTargets)
	if err != nil {
		return err
	}
	if (m.Status > 2 && m.Status != RegionExit) || m.Total > budget || m.Remaining != budget-m.Total || m.MemoryTotal > m.Total {
		return fmt.Errorf("invalid native linked progress")
	}
	return nil
}

// AdmitLink is cold publication work; the native dispatcher consumes only the
// arena-owned immutable proof, never this Go method, during guest execution.
func (n *Native) AdmitLink(entry, words, instructions int) error {
	if n.linkWords == 0 || words != n.linkWords {
		return fmt.Errorf("invalid linked state shape")
	}
	return n.arena.AdmitLinkedBlock(entry, words, instructions)
}

// RunLinkedQuanta schedules at most sixteen separate <=64-instruction native
// calls. Host accounting is aggregated; no guest state is assigned here. Only
// generation-ready published targets can start another call. A miss, fault,
// zero progress or a cold publication returns to the engine outside the lock.
func (n *Native) RunLinkedQuanta(state []uint64, m *MemoryContext, limit int, remaining, generation uint64) (int, error) {
	if m == nil || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || limit < 1 || limit > 16 || remaining == 0 {
		return 0, fmt.Errorf("invalid native linked quanta")
	}
	return n.linkCall.CallQuanta(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), n.linkPC, limit, remaining, generation)
}

func (n *Native) PrepareTargetLink(pc uint64, entry, words, instructions int) error {
	if err := n.AdmitLink(entry, words, instructions); err != nil {
		return err
	}
	return n.linkCall.PrepareTarget(pc, entry, instructions)
}
