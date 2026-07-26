//go:build renvo && vm && vm32

package os

const renvoVMGetdents64 = 217

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoVMGetdents64, fd, buf, len(buf))
}
