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
	if m != nil {
		m.CodeView, m.PreparedTargets, m.DescriptorBase, m.AdmissionEpoch = [4]uint64{}, 0, 0, 0
	}
	if m == nil || !n.CanLinkMemory(m) || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || budget == 0 || budget > NativeSessionLimit {
		if n.linkCall != nil {
			n.linkCall.ReleaseOwnedSessionPins()
		}
		return fmt.Errorf("invalid native linked session")
	}
	if n.codeVersionedContexts != nil && n.codeVersionedContexts[m] {
		return n.linkCall.CallOwnedSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget)
	}
	return n.linkCall.CallSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget)
}
