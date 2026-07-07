//go:build rtg && linux && amd64

package driver

func rtgFrontendCanResetArena() bool {
	// The embedded backend decodes the unit after frontend lowering; keep the
	// frontend arena live across that in-memory handoff.
	return false
}
