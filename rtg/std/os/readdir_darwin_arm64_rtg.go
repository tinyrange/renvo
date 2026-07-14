//go:build rtg && darwin && arm64

package os

// The backend recognizes this intrinsic selector and lowers it to
// getdirentries from libSystem; it is not issued as a Darwin syscall number.
const rtgGetdirentriesIntrinsic = 217

func rtgReadDirChunk(fd int, buf []byte) int {
	return syscall(rtgGetdirentriesIntrinsic, fd, buf, len(buf))
}
