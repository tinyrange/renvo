//go:build renvo && (linux || (darwin && arm64))

package time

func nativeInt64(b []byte, offset int) int64 {
	value := uint64(0)
	for i := 0; i < 8; i++ {
		value |= uint64(b[offset+i]) << uint(i*8)
	}
	return int64(value)
}
