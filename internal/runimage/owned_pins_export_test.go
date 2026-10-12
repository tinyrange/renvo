//go:build !renvo

package runimage

// Test-only opaque identity; never exposes native addresses or mutable state.
func OwnedSessionPinsForTest(a *CodeArena) interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ownedPins == nil {
		return nil
	}
	return a.ownedPins
}
