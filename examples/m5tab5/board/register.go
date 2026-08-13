package board

import "unsafe"

func load32(address uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(address))
}

func store32(address uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(address)) = value
}

func update32(address uintptr, clear, set uint32) {
	store32(address, load32(address)&^clear|set)
}

func delay(cycles int) {
	for index := 0; index < cycles; index++ {
	}
}
