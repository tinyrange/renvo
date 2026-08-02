package main

func appMain(args []string) int {
	value := uint64(1)<<32 |
		uint64(2)
	if value != 0x100000002 {
		print("FAIL: multiline binary expression\n")
		return 1
	}
	print("PASS\n")
	return 0
}
