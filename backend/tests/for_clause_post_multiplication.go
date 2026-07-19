package main

func appMain(args []string) int {
	sum := 0
	for i := 1; i < 8; i *= 2 {
		sum = sum + i
	}
	if sum != 7 {
		print("RENVO-0407 post multiply failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
