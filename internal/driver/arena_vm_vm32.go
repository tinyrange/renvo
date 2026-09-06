//go:build renvo && vm && vm32

package driver

func renvoFrontendCanResetArena() bool {
	// VM32 uses the same compact persistent handoff as the native self-hosted
	// compilers. Reclaim the frontend's transient allocations before starting
	// the backend so the two phases fit in the VM's fixed arena.
	return true
}
