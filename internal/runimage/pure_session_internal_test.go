//go:build !renvo

package runimage

// ForceSessionEpochForTesting makes the otherwise unreachable wrap boundary
// testable by the external execution test. It is absent from production builds.
func ForceSessionEpochForTesting(a *CodeArena, epoch uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessionEpoch = epoch
}
