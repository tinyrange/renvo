//go:build rtg && linux && (aarch64 || arm64)

package os

const rtgGetdents64 = 61

func rtgReadDirChunk(fd int, buf []byte) int {
	return syscall(rtgGetdents64, fd, buf, len(buf))
}
