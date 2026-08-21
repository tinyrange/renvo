//go:build renvo && openbsd && amd64

package os

const renvoGetdents = 99

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdents, fd, buf, len(buf))
}
