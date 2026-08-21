//go:build renvo && linux

package time

// syscall is a compiler intrinsic. The target-selected number is clock_gettime,
// and the result is the Linux timespec ABI for the selected architecture.
func syscall(number int, clockID int, result []byte) int { return -1 }

func Now() Time {
	var timespec [16]byte
	if syscall(clockGettimeSyscall, 0, timespec[:]) != 0 {
		return Time{}
	}
	result := Unix(nativeInt64(timespec[:], 0), nativeInt64(timespec[:], 8))
	if syscall(clockGettimeSyscall, 1, timespec[:]) == 0 {
		result.mono = nativeInt64(timespec[:], 0)*1000000000 + nativeInt64(timespec[:], 8)
	}
	return result
}
