//go:build renvo && linux && (aarch64 || arm64)

package os

const renvoGetdents64 = 61

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdents64, fd, buf, len(buf))
}
