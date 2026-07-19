//go:build renvo && darwin && arm64

package os

// The backend recognizes this intrinsic selector and lowers it to
// getdirentries from libSystem; it is not issued as a Darwin syscall number.
const renvoGetdirentriesIntrinsic = 217

func renvoReadDirChunk(fd int, buf []byte) int {
	return syscall(renvoGetdirentriesIntrinsic, fd, buf, len(buf))
}
