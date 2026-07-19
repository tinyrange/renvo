package main

func appMain(args []string) int {
	x := 0xaa & 0x0f
	if !(x == 0x0a) {
		print("RENVO-0209 bitwise_result_used_in_if_comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
