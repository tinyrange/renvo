//go:build renvo && !(linux && (amd64 || arm || arm64 || 386)) && !(darwin && arm64) && !(windows && (amd64 || 386)) && !(vm && vm32) && !(wasi && wasm32)

package driver

func renvoFrontendCanResetArena() bool {
	return false
}
