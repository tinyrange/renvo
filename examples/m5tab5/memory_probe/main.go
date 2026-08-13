package main

import "unsafe"

func load32(address uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(address))
}

func store32(address uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(address)) = value
}

func testWord(address uintptr) bool {
	original := load32(address)
	store32(address, 0x5aa59669)
	first := load32(address) == 0x5aa59669
	store32(address, 0xa55a6996)
	second := load32(address) == 0xa55a6996
	store32(address, original)
	return first && second
}

func main() {
	for delay := 0; delay < 5000000; delay++ {
	}
	print("PSRAM PROBE START\n")
	for delay := 0; delay < 5000000; delay++ {
	}
	message := "PSRAM BASE FAIL\n"
	if !testWord(0x48000000) {
		for {
			print(message)
		}
	}
	print("PSRAM BASE PASS\n")
	for delay := 0; delay < 5000000; delay++ {
	}
	message = "PSRAM 2M FAIL\n"
	if !testWord(0x48200000) {
		for {
			print(message)
		}
	}
	message = "PSRAM 2M PASS\n"
	for {
		print(message)
		for delay := 0; delay < 1000000; delay++ {
		}
	}
}
