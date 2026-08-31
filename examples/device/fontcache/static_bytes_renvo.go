//go:build renvo

package fontcache

import "unsafe"

func staticBytes(data string) []byte {
	// Renvo's 32-bit native ABI gives every string/slice header component an
	// eight-byte slot. The embedded cache has program lifetime and glyph masks
	// are read-only, so a slice may safely retain the string's backing bytes.
	source := (*[4]uint32)(unsafe.Pointer(&data))
	var result []byte
	destination := (*[6]uint32)(unsafe.Pointer(&result))
	destination[0], destination[1] = source[0], source[1]
	destination[2], destination[3] = source[2], source[3]
	destination[4], destination[5] = source[2], source[3]
	return result
}
