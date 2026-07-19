package main

func appMain(args []string) int {
	x := 0x12 ^ 0x21
	if !(x == 0x33) {
		print("RENVO-0207 bitwise_result_assigned_to_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
