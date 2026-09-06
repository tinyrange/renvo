//go:build renvo && !(linux && amd64) && !(darwin && arm64) && !(vm && vm32)

package driver

func renvoFrontendCanResetArena() bool {
	return false
}
