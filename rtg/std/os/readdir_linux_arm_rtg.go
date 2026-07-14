//go:build rtg && linux && arm

package os

const rtgGetdents64 = 217

func rtgReadDirChunk(fd int, buf []byte) int {
	return syscall(rtgGetdents64, fd, buf, len(buf))
}
