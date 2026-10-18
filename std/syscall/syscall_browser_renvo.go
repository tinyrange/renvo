//go:build renvo && browser

package syscall

// BrowserInvoke enters a host-private Wasm arena gate. The owner token is not
// a native code pointer; the gate independently checks its lifetime, entry and
// dimensions. This target extension uses Renvo's existing raw syscall ABI.
func BrowserInvoke(owner, entry, state, words, context int) int {
	return renvoSyscallBrowser(0x524645, owner, entry, state, words, context)
}

// The linker preserves renvoSyscall-prefixed declarations as the target syscall
// intrinsic, just as for the other platform-specific syscall adapters.
func renvoSyscallBrowser(number, a0, a1, a2, a3, a4 int) int { return -1 }
