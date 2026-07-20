//go:build renvo && darwin && arm64

package driver

func renvoFrontendCanResetArena() bool {
	// Darwin/arm64 uses the same persistent handoff as Linux/amd64. Reset the
	// transient frontend pages before invoking the embedded backend so repeated
	// IDE builds remain bounded instead of retaining an entire build each time.
	return true
}
