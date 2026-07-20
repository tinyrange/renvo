//go:build renvo && !(linux && amd64) && !(darwin && arm64)

package driver

func renvoFrontendCanResetArena() bool {
	return false
}
