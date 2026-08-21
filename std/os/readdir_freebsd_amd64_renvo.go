//go:build renvo && freebsd && amd64

package os

const renvoGetdirentries = 554

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdirentries, fd, buf, len(buf), 0)
}
