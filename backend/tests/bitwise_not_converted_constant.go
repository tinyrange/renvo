package main

var bitwiseNotConvertedMask uint64 = ^uint64(0)

func appMain(args []string) int {
	if bitwiseNotConvertedMask != 0xffffffffffffffff {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
