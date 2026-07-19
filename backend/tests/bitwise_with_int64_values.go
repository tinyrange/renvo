package main

func appMain(args []string) int {
	var x int64 = 0xf0
	var y int64 = 0x0f
	if !(x|y == 255) {
		print("RENVO-0219 bitwise_with_int64_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
