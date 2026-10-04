//go:build renvo && windows && 386

package driver

func renvoFrontendCanResetArena() bool {
	// The command handoff persists the unit and options before starting the
	// embedded backend. Reuse frontend scratch as on the other x86 hosts.
	return true
}
