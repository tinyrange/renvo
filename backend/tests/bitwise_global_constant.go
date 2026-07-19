package main

const bitGlobalMask = 0xf0 & 0xcc

func appMain(args []string) int {
	if !(bitGlobalMask == 0xc0) {
		print("RENVO-0223 bitwise_global_constant failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
