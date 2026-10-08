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
	if m == nil || m.linkOwner != n || n.linkWords == 0 || len(state) != n.linkWords || budget == 0 || budget > NativeSessionLimit {
		return fmt.Errorf("invalid native linked session")
	}
	return n.linkCall.CallSession(state, (*runimage.LinkedContextABI)(unsafe.Pointer(m)), budget)
}
