//go:build renvo

package serial

import "unsafe"

func copyAddressToBytes(target []byte, source unsafe.Pointer) {
	address := uintptr(source)
	for i := 0; i < len(target); i++ {
		target[i] = (*[1]byte)(unsafe.Pointer(address + uintptr(i)))[0]
	}
}

func copyBytesToAddress(target unsafe.Pointer, source []byte) {
	address := uintptr(target)
	for i := 0; i < len(source); i++ {
		(*[1]byte)(unsafe.Pointer(address + uintptr(i)))[0] = source[i]
	}
}

func copyFromAddress(target unsafe.Pointer, source unsafe.Pointer, size int) {
	targetAddress := uintptr(target)
	sourceAddress := uintptr(source)
	for i := 0; i < size; i++ {
		(*[1]byte)(unsafe.Pointer(targetAddress + uintptr(i)))[0] = (*[1]byte)(unsafe.Pointer(sourceAddress + uintptr(i)))[0]
	}
}
