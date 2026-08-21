//go:build renvo && netbsd && amd64

package os

const renvoGetdents30 = 390

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdents30, fd, buf, len(buf))
}
