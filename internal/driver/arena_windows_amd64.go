//go:build renvo && windows && amd64

package driver

func renvoFrontendCanResetArena() bool {
	// The compact unit and command options are persisted before the embedded
	// backend runs. Reuse frontend scratch instead of retaining both phases.
	return true
}
