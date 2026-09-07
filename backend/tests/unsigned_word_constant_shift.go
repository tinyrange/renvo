package main

const wordHalf = ^uint(0) >> 1
const pointerHalf = ^uintptr(0) >> 1
const wideHalf = ^uint64(0) >> 1

func appMain(args []string) int {
	expected := uint64(0x7fffffffffffffff) >> (64 - Sizeof(int(0))*8)
	if uint64(wordHalf) != expected {
		return 1
	}
	if uint64(pointerHalf) != expected {
		return 3
	}
	if wideHalf != 0x7fffffffffffffff {
		return 4
	}
	if uint64(^uint(0)>>1) != expected || int(-8)>>uint(1) != -4 {
		return 2
	}
	print("PASS\n")
	return 0
}
