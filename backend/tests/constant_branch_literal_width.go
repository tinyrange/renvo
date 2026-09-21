package main

const highFirst uint64 = 0x100000001
const highSecond uint64 = 0x200000001
const highMax uint64 = 0xffffffffffffffff
const halfMax = ^uint64(0) >> 1

func appMain(args []string) int {
	if highFirst != 0x100000001 {
		return 1
	}
	if highFirst == 0x200000001 {
		return 2
	}
	if highSecond == 1 {
		return 3
	}
	if highMax != 0xffffffffffffffff {
		return 4
	}
	if halfMax != 0x7fffffffffffffff {
		return 5
	}
	if 0x100000001 != highFirst {
		return 6
	}
	print("PASS\n")
	return 0
}
