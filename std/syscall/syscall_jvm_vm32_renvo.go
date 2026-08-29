//go:build renvo && jvm && vm32

package syscall

// renvoSyscallJVM is recognized by the linker as the JVM host-call boundary.
func renvoSyscallJVM(number int, first string, firstLen int, output []byte, outputLen int) int {
	return 0
}

func JVMCall(number int, first string, output []byte) int {
	return renvoSyscallJVM(number, first, len(first), output, len(output))
}
