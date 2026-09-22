//go:build renvo && linux && arm

package driver

func renvoFrontendCanResetArena() bool {
	// The compact unit and command options are persisted before this handoff.
	// Reuse frontend scratch storage for the embedded backend, as on amd64.
	return true
}
