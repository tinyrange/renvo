package main

func appMain(args []string) int {
	if !((2+3*4 < 20) && ((0xf0 & 0x0f) == 0)) {
		print("RENVO-0274 three_level_mixed_expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
