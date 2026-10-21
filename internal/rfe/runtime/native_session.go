//go:build !renvo

package runtime

import (
	"fmt"
	"renvo.dev/internal/runimage"
	"unsafe"
)

const NativeSessionLimit = runimage.SessionInstructionLimit

func NativeSessionsAvailable() bool { return runimage.ForeignSessionsAvailable() }

// RunLinkedSession is one supported foreign call, not a loop of short assembly
// calls. Retirement is bounded by budget and always reconstructed before return.
func (n *Native) RunLinkedSession(state []uint64, m *MemoryContext, budget uint64) error {
	return n.runLinkedSession(state, m, budget, nil)
}

// RunLinkedBackingSession only deduplicates pins for an already admitted owner.
func (n *Native) RunLinkedBackingSession(state []uint64, m *MemoryContext, budget uint64, backing []byte) error {
	return n.runLinkedSession(state, m, budget, backing)
}
func (n *Native) runLinkedSession(state []uint64, m *MemoryContext, budget uint64, backing []byte) error {
	if m != nil {
		m.CodeView, m.PreparedTargets, m.DescriptorBase, m.AdmissionEpoch = [4]uint64{}, 0, 0, 0
	}
	if m == nil || !n.CanLinkMemory(m) || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || budget == 0 || budget > NativeSessionLimit && m.Stop == nil {
		if n.linkCall != nil {
			n.linkCall.ReleaseOwnedSessionPins()
		}
		return fmt.Errorf("invalid native linked session")
	}
	if n.codeVersionedContexts != nil && n.codeVersionedContexts[m] {
		if len(backing) != 0 {
			return n.linkCall.CallOwnedBackingSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget, backing)
		}
		return n.linkCall.CallOwnedSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget)
	}
	return n.linkCall.CallSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget)
}
