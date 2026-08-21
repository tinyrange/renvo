//go:build renvo && amd64 && (freebsd || openbsd || netbsd)

package driver

// Native linked-image sessions require a target-specific anonymous-memory
// loader. Executable compilation and ordinary process execution remain fully
// supported on BSD hosts.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	return -1
}
