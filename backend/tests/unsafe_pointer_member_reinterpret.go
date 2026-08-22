package main

import "unsafe"

type unsafePointerMemberHolder struct {
	words [2]uint32
}

func unsafePointerMemberLoad(address *uint64) uint64 {
	return *address
}

func appMain(args []string) int {
	var storage unsafePointerMemberHolder
	storage.words[0] = 0x11223344
	storage.words[1] = 0x55667788
	value := unsafePointerMemberLoad((*uint64)(unsafe.Pointer(&storage.words)))
	if value != 0x5566778811223344 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
