//go:build !renvo

package serial

import "unsafe"

func copyAddressToBytes(target []byte, source unsafe.Pointer) {
	copy(target, unsafe.Slice((*byte)(source), len(target)))
}

func copyBytesToAddress(target unsafe.Pointer, source []byte) {
	copy(unsafe.Slice((*byte)(target), len(source)), source)
}

func copyFromAddress(target unsafe.Pointer, source unsafe.Pointer, size int) {
	copy(unsafe.Slice((*byte)(target), size), unsafe.Slice((*byte)(source), size))
}
