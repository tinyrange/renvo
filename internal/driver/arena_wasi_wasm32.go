//go:build renvo && wasi && wasm32

package driver

func renvoFrontendCanResetArena() bool {
	// Reuse frontend scratch memory for the embedded backend after persisting
	// the compact unit and command options, as on the other arena targets.
	return true
}
