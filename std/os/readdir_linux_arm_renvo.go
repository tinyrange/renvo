//go:build renvo && linux && arm

package os

const renvoGetdents64 = 217

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdents64, fd, buf, len(buf))
}
