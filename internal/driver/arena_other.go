//go:build renvo && !(linux && (amd64 || arm)) && !(darwin && arm64) && !(vm && vm32) && !(wasi && wasm32)

package driver

func renvoFrontendCanResetArena() bool {
	return false
}
