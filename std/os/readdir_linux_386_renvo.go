//go:build renvo && linux && 386

package os

const renvoGetdents64 = 220

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdents64, fd, buf, len(buf))
}
