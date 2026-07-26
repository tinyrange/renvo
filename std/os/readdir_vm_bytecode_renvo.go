//go:build renvo && vm && bytecode

package os

const renvoVMGetdents64 = 217

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoVMGetdents64, fd, buf, len(buf))
}
