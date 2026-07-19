//go:build renvo && linux && amd64

package driver

func renvoFrontendCanResetArena() bool {
	// Run the embedded backend from the compact persistent handoff assembled in
	// RunRenvoCommand. Releasing the frontend's transient pages here prevents the
	// two compiler phases from accumulating in the same RSS peak.
	return true
}
