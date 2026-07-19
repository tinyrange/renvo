package main

func appMain(args []string) int {
	b := byte(65)
	if !(int(b)+1 == 66) {
		print("RENVO-0272 conversion_inside_arithmetic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
