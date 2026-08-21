//go:build renvo && darwin && arm64

package time

const renvoClockGettimeIntrinsic = 228

// syscall is recognized by the compiler. On Darwin the selector is lowered to
// libSystem clock_gettime; no raw Darwin syscall number is embedded.
func syscall(selector int, clockID int, result []byte) int { return -1 }

func Now() Time {
	var timespec [16]byte
	if syscall(renvoClockGettimeIntrinsic, 0, timespec[:]) != 0 {
		return Time{}
	}
	result := Unix(nativeInt64(timespec[:], 0), nativeInt64(timespec[:], 8))
	if syscall(renvoClockGettimeIntrinsic, 6, timespec[:]) == 0 {
		result.mono = nativeInt64(timespec[:], 0)*1000000000 + nativeInt64(timespec[:], 8)
	}
	return result
}
