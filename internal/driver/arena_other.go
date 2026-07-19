//go:build renvo && !(linux && amd64)

package driver

func renvoFrontendCanResetArena() bool {
	return false
}
