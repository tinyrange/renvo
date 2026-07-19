package main

func appMain(args []string) int {
	x := 0
	if 8 >= 8 {
		x = 10
	}
	if x != 10 {
		print("RENVO-0360 comparison condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
