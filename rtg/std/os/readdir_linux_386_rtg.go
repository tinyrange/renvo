//go:build rtg && linux && 386

package os

const rtgGetdents64 = 220

func rtgReadDirChunk(fd int, buf []byte) int {
	return syscall(rtgGetdents64, fd, buf, len(buf))
}
