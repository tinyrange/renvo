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
	return memoryLayoutOK() && unsafe.Offsetof(m.NativeContext) == 0 &&
		unsafe.Sizeof(m.NativeContext) == backendcompiled.RenvoRFEContextSize &&
		unsafe.Offsetof(m.linkOwner) == backendcompiled.RenvoRFEContextSize &&
		unsafe.Sizeof(b) == backendcompiled.RenvoRFEDescriptorSize &&
		unsafe.Offsetof(b.PC) == 0 && unsafe.Offsetof(b.Entry) == 8 &&
		unsafe.Offsetof(b.Instructions) == 16 && unsafe.Offsetof(b.Reserved) == 24 &&
		unsafe.Offsetof(b.Prefix) == backendcompiled.RenvoRFEDescriptorPrefix &&
		unsafe.Offsetof(m.Remaining) == backendcompiled.RenvoRFERemaining &&
		unsafe.Offsetof(m.Total) == backendcompiled.RenvoRFETotal &&
		unsafe.Offsetof(m.MemoryTotal) == backendcompiled.RenvoRFEMemoryTotal &&
		unsafe.Offsetof(m.CodeView) == backendcompiled.RenvoRFECodeView &&
		unsafe.Offsetof(m.Blocks) == backendcompiled.RenvoRFEBlocks &&
		unsafe.Offsetof(m.LoopExits) == backendcompiled.RenvoRFELoopExits &&
		unsafe.Offsetof(m.LoopIterations) == backendcompiled.RenvoRFELoopIterations &&
		unsafe.Offsetof(m.PreparedTargets) == backendcompiled.RenvoRFEPreparedTargets &&
		unsafe.Offsetof(m.DescriptorBase) == backendcompiled.RenvoRFEDescriptorBase &&
		unsafe.Offsetof(m.AdmissionEpoch) == backendcompiled.RenvoRFEAdmissionEpoch &&
		unsafe.Offsetof(m.DirectGuest) == backendcompiled.RenvoRFEDirectGuest &&
		unsafe.Offsetof(m.DirectHost) == backendcompiled.RenvoRFEDirectHost &&
		unsafe.Offsetof(m.DirectSize) == backendcompiled.RenvoRFEDirectSize &&
		unsafe.Offsetof(m.TimerFunction) == backendcompiled.RenvoRFETimerFunction &&
		unsafe.Offsetof(m.TimerDeadline) == backendcompiled.RenvoRFETimerDeadline &&
		unsafe.Offsetof(m.TimerExpired) == backendcompiled.RenvoRFETimerExpired &&
		unsafe.Offsetof(m.HashPages) == backendcompiled.RenvoRFEHashPages
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
	if n.profileDispatch {
		code, ok = backendcompiled.RenvoEmitProfileLinkedDispatcher(pc, words, runtime.GOARCH == "arm64")
	}
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
	if m != nil {
		m.TimerFunction, m.TimerDeadline, m.TimerExpired = 0, 0, 0
	}
	if m == nil || !n.CanLinkMemory(m) || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || budget == 0 || budget > 64 {
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

// PrepareTargetLinkWithTail binds an original leaf to its immutable full-region
// admission; neither executable pointer nor count is accepted from guest RAM.
func (n *Native) PrepareTargetLinkWithTail(pc uint64, entry, words, instructions, tailEntry, tailInstructions int) error {
	if err := n.AdmitLink(entry, words, instructions); err != nil {
		return err
	}
	if err := n.AdmitLink(tailEntry, words, tailInstructions); err != nil {
		return err
	}
	return n.linkCall.PrepareTargetWithTail(pc, entry, instructions, tailEntry, tailInstructions)
}

// RunLinkedQuanta schedules at most sixteen separate <=64-instruction native
// calls. Host accounting is aggregated; no guest state is assigned here. Only
// generation-ready published targets can start another call. A miss, fault,
// zero progress or a cold publication returns to the engine outside the lock.
func (n *Native) RunLinkedQuanta(state []uint64, m *MemoryContext, limit int, remaining, generation uint64) (int, error) {
	if m != nil {
		m.TimerFunction, m.TimerDeadline, m.TimerExpired = 0, 0, 0
	}
	if m == nil || !n.CanLinkMemory(m) || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || limit < 1 || limit > 16 || remaining == 0 {
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

// DispatchCounters returns optional private successor-cache diagnostics.
func (n *Native) DispatchCounters() [5]uint64 { return n.arena.DispatchCounters() }
