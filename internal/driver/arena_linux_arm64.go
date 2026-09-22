//go:build renvo && linux && arm64

package driver

func renvoFrontendCanResetArena() bool {
	// Reuse frontend scratch storage after persisting the backend handoff.
	return true
}
